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

package main

import (
	"fmt"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"slices"
	"strconv"
	"strings"
)

// skillDir is where the agent skill is generated, relative to the repository root. Its SKILL.md is an index of the
// catalog, and references/ holds the pages with their code filled in.
const skillDir = "skill"

const skillIntro = "# Go Concurrency Patterns\n\n" +
	"A catalog of Go concurrency patterns, concepts, standard library guides, and anti-patterns. The code in the\n" +
	"reference files is tested with the race detector and uses only the standard library.\n\n" +
	"## How to use this skill\n\n" +
	"- The principles below are this catalog's opinions about good Go concurrency, not rules of the language. Apply\n" +
	"  them with judgment; principle 3 (the simplest mechanism that works) usually wins a conflict.\n" +
	"- Do not read the whole catalog. Use the indexes below to find the page that fits the problem, then read that\n" +
	"  file in `references/`. Each page has the problem, the mechanism with code, when it fits, and what it costs.\n" +
	"- When reviewing code, go through the anti-pattern checklist and name the page for each finding.\n" +
	"- Verify concurrent code with `go test -race`, and test timing with `testing/synctest`; see\n" +
	"  `references/stdlib/synctest.md`.\n\n"

const skillDescription = "Use when writing, reviewing, or designing Go code that uses goroutines, channels, sync, " +
	"context, timers, or atomics: choosing a concurrency pattern, spotting anti-patterns (goroutine leaks, " +
	"sleeping to synchronize, closing from the wrong side, unbounded goroutines, ignored contexts), and structuring " +
	"owner goroutines, pipelines, worker pools, rate limits, queues, and shutdown."

var (
	frontMatter   = regexp.MustCompile(`(?s)\A---\n.*?\n---\n`)
	commentLine   = regexp.MustCompile(`(?m)^[ \t]*<!-- /?[a-z]+(?::.*)? -->[ \t]*\n`)
	mermaidBlock  = regexp.MustCompile("(?s)```mermaid\n.*?```\n")
	htmlTag       = regexp.MustCompile(`</?(?:small|span|div)\b[^>]*>`)
	markdownLink  = regexp.MustCompile(`\[([^\]]*)\]\(([^)\s]+)\)`)
	headingID     = regexp.MustCompile(` *\{ #[a-z0-9-]+ \}`)
	blankRuns     = regexp.MustCompile(`\n{3,}`)
	propsRow      = regexp.MustCompile(`^\| (.+?) \| (.+) \|$`)
	tabLine       = regexp.MustCompile(`^=== "(.+)"$`)
	admonition    = regexp.MustCompile(`^!!! \w+ "(.+)"$`)
	detectionBody = regexp.MustCompile(`(?s)\n## Detection\n(.*?)(?:\n## |\z)`)
)

// exportSkill returns the files of the agent skill, by slash path relative to the skill directory. pages holds
// the generated markdown of every page, by slash path relative to the repository root.
func exportSkill(st *site, pages map[string]string) map[string]string {
	files := map[string]string{}

	for rel, text := range pages {
		out, ok := exportPath(rel)
		if !ok {
			continue
		}

		files[out] = skillPage(rel, out, text)
	}

	index, ok := pages["docs/index.md"]
	if ok {
		files["references/principles.md"] = skillPrinciples(index, "docs/index.md", "references/principles.md")
	}

	files["SKILL.md"] = skillIndex(st, pages)

	return files
}

// exportPath is where the page at rel, relative to the repository root, goes in the skill, if it goes there.
func exportPath(rel string) (string, bool) {
	switch rel {
	case "docs/index.md":
		return "references/principles.md", true
	case "docs/sources.md":
		return "references/sources.md", true
	}

	dir, name := path.Split(rel)
	for _, kind := range []string{"patterns", "anti-patterns", "concepts", stdlibKind} {
		if dir == "docs/"+kind+"/" && name != "index.md" && strings.HasSuffix(name, ".md") {
			return "references/" + kind + "/" + name, true
		}
	}

	return "", false
}

// skillPage turns a page of the site into a reference file: no front matter, generated markers, diagrams, or
// HTML, and links that point to other reference files, or are plain text when their target is not in the skill.
func skillPage(rel, out, text string) string {
	if rel == "docs/index.md" {
		return skillPrinciples(text, rel, out)
	}

	text = frontMatter.ReplaceAllString(text, "")

	return skillMarkdown(text, rel, out)
}

// skillPrinciples is the principles section of the home page.
func skillPrinciples(index, rel, out string) string {
	start := strings.Index(index, "\n## Principles\n")
	if start < 0 {
		return ""
	}

	text := index[start+1:]
	if end := strings.Index(text, "\n## Finding your way"); end >= 0 {
		text = text[:end+1]
	}

	text = strings.Replace(text, "## Principles", "# Principles", 1)

	return skillMarkdown(text, rel, out)
}

// skillMarkdown converts the markdown of a site page, whose links are relative to the page at rel, into plain
// markdown for the file out in the skill.
func skillMarkdown(text, rel, out string) string {
	text = commentLine.ReplaceAllString(text, "")
	text = mermaidBlock.ReplaceAllString(text, "")
	text = flattenBlocks(text)

	parts := splitFences(text)
	for i, part := range parts {
		if part.code {
			continue
		}

		prose := htmlTag.ReplaceAllString(part.text, "")
		prose = rewriteLinks(prose, rel, out)
		parts[i].text = prose
	}

	var b strings.Builder
	for _, part := range parts {
		b.WriteString(part.text)
	}

	return strings.TrimSpace(blankRuns.ReplaceAllString(b.String(), "\n\n")) + "\n"
}

type textPart struct {
	text string
	code bool
}

// splitFences splits markdown into prose and fenced code, so that rewriting leaves code alone.
func splitFences(text string) []textPart {
	var parts []textPart

	var cur []string

	inCode := false
	flush := func(code bool) {
		if len(cur) > 0 {
			parts = append(parts, textPart{text: strings.Join(cur, ""), code: code})
			cur = nil
		}
	}

	for _, line := range strings.SplitAfter(text, "\n") {
		trimmed := strings.TrimSpace(line)

		switch {
		case !inCode && strings.HasPrefix(trimmed, "```"):
			flush(false)

			inCode = true

			cur = append(cur, line)
		case inCode && strings.HasPrefix(trimmed, "```"):
			cur = append(cur, line)
			flush(true)

			inCode = false
		default:
			cur = append(cur, line)
		}
	}

	flush(inCode)

	return parts
}

// flattenBlocks turns the constructs of the site into plain markdown: the properties table into a list, tabs into
// bold labels, and admonitions into block quotes.
func flattenBlocks(text string) string {
	lines := strings.Split(text, "\n")
	out := make([]string, 0, len(lines))

	inProps, inTab, inNote := false, false, false

	for _, line := range lines {
		switch {
		case strings.HasPrefix(line, `<div class="cp-props"`):
			inProps = true

			continue
		case inProps && strings.HasPrefix(line, "</div>"):
			inProps = false

			continue
		case inProps:
			if m := propsRow.FindStringSubmatch(line); m != nil && strings.TrimSpace(m[1]) != "" &&
				!strings.HasPrefix(m[1], "---") {
				out = append(out, "- **"+m[1]+":** "+m[2])
			}

			continue
		}

		if inTab || inNote {
			switch {
			case strings.HasPrefix(line, "    "):
				line = strings.TrimPrefix(line, "    ")
				if inNote {
					line = "> " + line
				}
			case line == "":
			default:
				inTab, inNote = false, false
			}
		}

		if m := tabLine.FindStringSubmatch(line); m != nil && !inTab && !inNote {
			inTab = true

			out = append(out, "**"+m[1]+"**", "")

			continue
		}

		if m := admonition.FindStringSubmatch(line); m != nil && !inNote {
			inNote = true

			out = append(out, "> **"+m[1]+".**")

			continue
		}

		out = append(out, line)
	}

	return strings.Join(out, "\n")
}

// rewriteLinks points the links of a page at rel to the reference files they became, relative to the file out,
// and replaces a link to anything outside the skill with its text.
func rewriteLinks(text, rel, out string) string {
	return markdownLink.ReplaceAllStringFunc(text, func(link string) string {
		m := markdownLink.FindStringSubmatch(link)
		label, target := m[1], m[2]

		if strings.Contains(target, "://") || strings.HasPrefix(target, "mailto:") {
			return link
		}

		file, anchor, _ := strings.Cut(target, "#")
		if file == "" {
			return link
		}

		dest, ok := exportPath(path.Join(path.Dir(rel), file))
		if !ok {
			return label
		}

		from := path.Dir(out)

		relative, err := filepath.Rel(from, dest)
		if err != nil {
			return label
		}

		if anchor != "" {
			relative += "#" + anchor
		}

		return "[" + label + "](" + filepath.ToSlash(relative) + ")"
	})
}

// skillIndex is SKILL.md: when to use the skill, the principles, how to choose, an index of the pages, and a
// checklist of the anti-patterns.
func skillIndex(st *site, pages map[string]string) string {
	var b strings.Builder

	b.WriteString("---\nname: Go Concurrency Patterns\ndescription: " + strconv.Quote(skillDescription) + "\n---\n\n")
	b.WriteString(skillIntro)

	if index, ok := pages["docs/index.md"]; ok {
		principles := skillPrinciples(index, "docs/index.md", "SKILL.md")
		principles = strings.Replace(principles, "# Principles", "## Principles", 1)
		principles = headingID.ReplaceAllString(principles, "")
		principles = strings.ReplaceAll(principles, "\n### ", "\n### ")

		b.WriteString(strings.TrimSpace(principles) + "\n\n")
	}

	b.WriteString(skillChoosing(st))
	b.WriteString(skillPatternIndex(st))
	b.WriteString(skillChecklist(st, pages))

	return strings.TrimSpace(blankRuns.ReplaceAllString(b.String(), "\n\n")) + "\n"
}

// skillChoosing lists the guides of the concerns, which compare the patterns that address each.
func skillChoosing(st *site) string {
	var b strings.Builder

	b.WriteString("## Choosing a pattern\n\n")

	for _, key := range st.tax.Concern.order {
		v := st.tax.Concern.byKey[key]
		if strings.TrimSpace(v.Guide) == "" {
			continue
		}

		guide := strings.Join(strings.Fields(v.Guide), " ")
		guide = rewriteLinks(guide, "docs/catalog/concern.md", "SKILL.md")

		b.WriteString("### " + v.Title + "\n\n" + v.Description + " " + guide + "\n\n")
	}

	return b.String()
}

// skillPatternIndex lists the pages by kind, with the intent of each.
func skillPatternIndex(st *site) string {
	var b strings.Builder

	line := func(p *pattern) {
		dest, ok := exportPath(p.path)
		if !ok {
			return
		}

		b.WriteString("- [" + p.Title + "](" + dest + "): " + strings.Join(strings.Fields(p.Intent), " ") + "\n")
	}

	b.WriteString("## Patterns\n\n")

	for _, key := range st.tax.Purpose.order {
		v := st.tax.Purpose.byKey[key]

		members := st.sorted(func(p *pattern) bool {
			return !p.anti && !p.concept && !p.stdlib && p.Purpose == key
		})
		if len(members) == 0 {
			continue
		}

		b.WriteString("### " + v.Title + "\n\n" + v.Description + "\n\n")

		for _, p := range members {
			line(p)
		}

		b.WriteString("\n")
	}

	b.WriteString("## Concepts\n\nShared background that the patterns link to.\n\n")

	for _, p := range st.sorted(func(p *pattern) bool { return p.concept }) {
		line(p)
	}

	b.WriteString("\n## Standard library guides\n\nHow to use what Go already provides.\n\n")

	for _, p := range st.sorted(func(p *pattern) bool { return p.stdlib }) {
		line(p)
	}

	b.WriteString("\n")

	return b.String()
}

// skillChecklist is the anti-patterns with what to look for, from the Detection section of each page.
func skillChecklist(st *site, pages map[string]string) string {
	var b strings.Builder

	b.WriteString("## Anti-pattern checklist\n\n")
	b.WriteString("For a review, look for each of these; the linked page has the example and the fix.\n\n")

	for _, p := range st.sorted(func(p *pattern) bool { return p.anti }) {
		dest, ok := exportPath(p.path)
		if !ok {
			continue
		}

		b.WriteString("### [" + p.Title + "](" + dest + ")\n\n")
		b.WriteString(strings.Join(strings.Fields(p.Intent), " ") + "\n\n")

		if len(p.Violates) > 0 {
			titles := make([]string, len(p.Violates))
			for i, k := range p.Violates {
				titles[i] = st.tax.Principle.byKey[k].Title
			}

			b.WriteString("Violates: " + strings.Join(titles, ", ") + ".")

			if fixes := skillFixes(st, p); fixes != "" {
				b.WriteString(" Fixed by: " + fixes + ".")
			}

			b.WriteString("\n\n")
		}

		if m := detectionBody.FindStringSubmatch(pages[p.path]); m != nil {
			detection := strings.TrimSpace(m[1])
			detection = rewriteLinks(detection, p.path, "SKILL.md")

			b.WriteString("Look for:\n\n" + detection + "\n\n")
		}
	}

	return b.String()
}

// skillFixes lists the remedies of an anti-pattern as links.
func skillFixes(st *site, p *pattern) string {
	var fixes []string

	for _, slug := range p.Remedies {
		target, err := st.pattern(st.slugPath(slug))
		if err != nil {
			continue
		}

		if dest, ok := exportPath(target.path); ok {
			fixes = append(fixes, "["+target.Title+"]("+dest+")")
		}
	}

	return strings.Join(fixes, ", ")
}

// syncSkill brings the skill directory in line with files. With check set it changes nothing and returns the
// files that are missing, different, or left over.
func syncSkill(root string, files map[string]string, check bool) ([]string, error) {
	dir := filepath.Join(root, skillDir)

	var stale []string

	existing := map[string]bool{}

	err := filepath.WalkDir(dir, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			if os.IsNotExist(err) {
				return nil
			}

			return err
		}

		if d.IsDir() {
			return nil
		}

		rel, relErr := filepath.Rel(dir, p)
		if relErr != nil {
			return relErr
		}

		existing[filepath.ToSlash(rel)] = true

		return nil
	})
	if err != nil {
		return nil, err
	}

	for name, content := range files {
		file := filepath.Join(dir, filepath.FromSlash(name))

		raw, readErr := os.ReadFile(file) //nolint:gosec // paths built from the export
		if readErr != nil || string(raw) != content {
			stale = append(stale, path.Join(skillDir, name))
		}
	}

	for name := range existing {
		if _, ok := files[name]; !ok {
			stale = append(stale, path.Join(skillDir, name))
		}
	}

	slices.Sort(stale)

	if check || len(stale) == 0 {
		return stale, nil
	}

	if err = os.RemoveAll(dir); err != nil {
		return nil, err
	}

	for name, content := range files {
		target := filepath.Join(dir, filepath.FromSlash(name))
		if err = os.MkdirAll(filepath.Dir(target), 0o755); err != nil { //nolint:gosec // generated files are public
			return nil, err
		}

		if err = os.WriteFile(target, []byte(content), 0o644); err != nil { //nolint:gosec // generated files are public
			return nil, fmt.Errorf("%s: %w", name, err)
		}
	}

	return stale, nil
}
