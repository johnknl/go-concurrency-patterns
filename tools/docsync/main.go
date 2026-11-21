// MIT License
//
// Copyright (C) 2026 John Kleijn
//
// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to deal
// in the Software without restriction, including without limitation the rights
// to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
// copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in all
// copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
//

// Command docsync keeps the generated parts of the documentation in sync with their sources. A generated part
// is delimited by two comment lines, and everything between them is rewritten.
//
// Code from the Go packages:
//
//	<!-- code: patterns/pipeline#Stage patterns/pipeline#Then:sig,doc -->
//	<!-- /code -->
//
// A reference is <package directory>#<symbol>[:<modifiers>], where the directory is relative to the repository
// root and the symbol is a function, a type, a constant, a variable, a method (Type.Method), or an example
// function. All symbols of a block are rendered, in order, into one Go code block. Modifiers, separated by
// commas:
//
//	doc   include the doc comment
//	sig   a function or method without its body
//	body  only the statements inside a function
//	main  the statements inside a function, as the body of func main (for examples)
//
// Classification, taken from the front matter of the pattern pages and from taxonomy.yml:
//
//	<!-- meta -->           on a pattern page: its purpose, scale, mechanisms, concerns, and code
//	<!-- /meta -->
//	<!-- sources -->        on a pattern page: its sources
//	<!-- /sources -->
//	<!-- composition -->    on a pattern page: the patterns named in its composes list
//	<!-- /composition -->
//	<!-- remedies -->       on an anti-pattern page: the patterns that fix it (its remedies list)
//	<!-- /remedies -->
//	<!-- antipatterns -->   on a pattern page: the anti-patterns that name it as a remedy
//	<!-- /antipatterns -->
//	<!-- catalog: NAME [level=N] [id=SOURCE] -->
//	    the patterns grouped by purpose, scale, mechanism, concern, or source, or NAME `anti-pattern` for the
//	    anti-patterns with their remedies, or `superseded` for the patterns marked superseded
//	<!-- /catalog -->
//	<!-- bibliography -->  the references of taxonomy.yml, with the pages that cite each
//	<!-- /bibliography -->
//
// Usage:
//
//	docsync sync [file...]    rewrite the generated parts, of the given files or of all markdown files
//	docsync check [file...]   exit with an error when a generated part is out of date
package main

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strconv"
	"strings"
)

func main() {
	if len(os.Args) < 2 || (os.Args[1] != "sync" && os.Args[1] != "check") {
		_, _ = fmt.Fprintln(os.Stderr, "usage: docsync sync|check [file...]")
		os.Exit(2)
	}

	if err := run(".", os.Args[1] == "check", os.Args[2:]); err != nil {
		_, _ = fmt.Fprintln(os.Stderr, "docsync:", err)
		os.Exit(1)
	}
}

func run(root string, check bool, only []string) error {
	files, err := markdownFiles(root)
	if err != nil {
		return err
	}

	if len(only) > 0 {
		files = only
	}

	loader := newLoader(root)

	st, err := loadSite(root)
	if err != nil {
		return err
	}

	var stale []string

	for _, file := range files {
		raw, readErr := os.ReadFile(file) //nolint:gosec // paths come from walking the repository
		if readErr != nil {
			return readErr
		}

		rel, relErr := filepath.Rel(root, file)
		if relErr != nil {
			return relErr
		}

		updated, renderErr := render(string(raw), filepath.ToSlash(rel), loader, st)
		if renderErr != nil {
			return fmt.Errorf("%s: %w", file, renderErr)
		}

		if updated == string(raw) {
			continue
		}

		stale = append(stale, file)

		if !check {
			if writeErr := os.WriteFile(file, []byte(updated), 0o644); writeErr != nil { //nolint:gosec // docs are public
				return writeErr
			}

			_, _ = fmt.Fprintln(os.Stdout, "updated", file)
		}
	}

	for _, path := range st.unusedConcepts() {
		_, _ = fmt.Fprintln(os.Stderr, "docsync: warning:", path, "is a concept no page relies on")
	}

	if check && len(stale) > 0 {
		return fmt.Errorf("out of sync, run `go tool docsync sync`: %s", strings.Join(stale, ", "))
	}

	return nil
}

func markdownFiles(root string) ([]string, error) {
	var files []string

	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if d.IsDir() {
			name := d.Name()
			if path != root && (strings.HasPrefix(name, ".") || name == "site" || name == "bin") {
				return filepath.SkipDir
			}

			return nil
		}

		if strings.HasSuffix(path, ".md") {
			files = append(files, path)
		}

		return nil
	})

	slices.Sort(files)

	return files, err
}

var markerPattern = regexp.MustCompile(`^<!-- (` +
	`code|catalog|meta|sources|composition|remedies|antipatterns|usedby|violatedby|bibliography` +
	`)(?::(.*?))? -->$`)

// render regenerates every generated block in the markdown text of the file at path. Marker lines inside fenced
// code are ignored, so the syntax can be documented.
func render(text, path string, loader *loader, st *site) (string, error) {
	lines := strings.Split(text, "\n")
	out := make([]string, 0, len(lines))

	var fence string

	for i := 0; i < len(lines); i++ {
		line := lines[i]
		trimmed := strings.TrimSpace(line)

		if fence != "" {
			if strings.HasPrefix(trimmed, fence) {
				fence = ""
			}

			out = append(out, line)

			continue
		}

		if strings.HasPrefix(trimmed, "```") || strings.HasPrefix(trimmed, "~~~") {
			fence = trimmed[:3]
			out = append(out, line)

			continue
		}

		m := markerPattern.FindStringSubmatch(trimmed)
		if m == nil {
			out = append(out, line)
			continue
		}

		closing := "<!-- /" + m[1] + " -->"
		indent := line[:len(line)-len(strings.TrimLeft(line, " \t"))]

		end := slices.IndexFunc(lines[i+1:], func(l string) bool { return strings.TrimSpace(l) == closing })
		if end < 0 {
			return "", fmt.Errorf("line %d: missing %s", i+1, closing)
		}

		body, err := generate(m[1], strings.Fields(m[2]), path, loader, st)
		if err != nil {
			return "", fmt.Errorf("line %d: %w", i+1, err)
		}

		// A marker inside an indented block, such as a tab or a collapsible admonition, keeps its content there.
		if indent != "" {
			body = "\n" + body + "\n" // inside an indented block, keep the content apart from the markers
		}

		out = append(out, line, indentLines(body, indent), indent+closing)
		i += end + 1
	}

	return strings.Join(out, "\n"), nil
}

// indentLines prefixes every non-empty line of text with indent.
func indentLines(text, indent string) string {
	if indent == "" {
		return text
	}

	lines := strings.Split(text, "\n")
	for i, l := range lines {
		if l != "" {
			lines[i] = indent + l
		}
	}

	return strings.Join(lines, "\n")
}

func generate(kind string, args []string, path string, loader *loader, st *site) (string, error) {
	switch kind {
	case "code":
		code, err := loader.render(args)
		if err != nil {
			return "", err
		}

		return "```go\n" + code + "\n```", nil
	case "catalog":
		return renderCatalogMarker(args, path, st)
	case "bibliography":
		return st.renderBibliography(path), nil
	}

	page, err := st.pattern(path)
	if err != nil {
		return "", err
	}

	switch kind {
	case "meta":
		return st.renderMeta(page), nil
	case "composition":
		return st.renderComposition(page)
	case "remedies":
		return st.renderRemedies(page)
	case "antipatterns":
		return st.renderAntiPatterns(page)
	case "usedby":
		return st.renderUsedBy(page, false)
	case "violatedby":
		return st.renderUsedBy(page, true)
	}

	return st.renderSources(page), nil
}

func renderCatalogMarker(args []string, path string, st *site) (string, error) {
	if len(args) == 0 {
		return "", errors.New("catalog needs a name: purpose, mechanism, concern, or source")
	}

	level, only := 3, ""

	for _, arg := range args[1:] {
		switch {
		case strings.HasPrefix(arg, "level="):
			var err error
			if level, err = strconv.Atoi(strings.TrimPrefix(arg, "level=")); err != nil {
				return "", err
			}
		case strings.HasPrefix(arg, "id="):
			only = strings.TrimPrefix(arg, "id=")
		default:
			return "", fmt.Errorf("unknown catalog option %q", arg)
		}
	}

	return st.renderCatalog(path, args[0], level, only)
}

var errNoRefs = errors.New("no symbols listed")
