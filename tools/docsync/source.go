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
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

type loader struct {
	pkgs map[string]map[string]*symbol
	root string
}

func newLoader(root string) *loader {
	return &loader{root: root, pkgs: map[string]map[string]*symbol{}}
}

// symbol is a top-level declaration, or one spec of a grouped declaration.
type symbol struct {
	fset *token.FileSet
	fn   *ast.FuncDecl
	gen  *ast.GenDecl
	spec ast.Spec
	src  []byte
}

type modifiers struct {
	doc, sig, body, main bool
}

func (l *loader) render(refs []string) (string, error) {
	if len(refs) == 0 {
		return "", errNoRefs
	}

	parts := make([]string, 0, len(refs))

	for _, ref := range refs {
		part, err := l.renderRef(ref)
		if err != nil {
			return "", fmt.Errorf("%s: %w", ref, err)
		}

		parts = append(parts, part)
	}

	return strings.Join(parts, "\n\n"), nil
}

func (l *loader) renderRef(ref string) (string, error) {
	dir, rest, ok := strings.Cut(ref, "#")
	if !ok || dir == "" || rest == "" {
		return "", fmt.Errorf("want <dir>#<symbol>")
	}

	name, mods, _ := strings.Cut(rest, ":")

	var m modifiers

	for mod := range strings.SplitSeq(mods, ",") {
		switch mod {
		case "":
		case "doc":
			m.doc = true
		case "sig":
			m.sig = true
		case "body":
			m.body = true
		case "main":
			m.main = true
		default:
			return "", fmt.Errorf("unknown modifier %q", mod)
		}
	}

	if (m.body || m.main) && (m.sig || m.doc || (m.body && m.main)) {
		return "", fmt.Errorf("body and main stand alone")
	}

	syms, err := l.load(dir)
	if err != nil {
		return "", err
	}

	sym, ok := syms[name]
	if !ok {
		return "", fmt.Errorf("symbol not found in %s", dir)
	}

	return sym.render(m)
}

func (l *loader) load(dir string) (map[string]*symbol, error) {
	if syms, ok := l.pkgs[dir]; ok {
		return syms, nil
	}

	paths, err := filepath.Glob(filepath.Join(l.root, dir, "*.go"))
	if err != nil {
		return nil, err
	}

	if len(paths) == 0 {
		return nil, fmt.Errorf("no Go files in %s", dir)
	}

	syms := map[string]*symbol{}
	fset := token.NewFileSet()

	for _, path := range paths {
		src, readErr := os.ReadFile(path) //nolint:gosec // paths come from globbing the repository
		if readErr != nil {
			return nil, readErr
		}

		file, parseErr := parser.ParseFile(fset, path, src, parser.ParseComments)
		if parseErr != nil {
			return nil, parseErr
		}

		for _, decl := range file.Decls {
			if err = index(syms, decl, src, fset); err != nil {
				return nil, fmt.Errorf("%s: %w", path, err)
			}
		}
	}

	l.pkgs[dir] = syms

	return syms, nil
}

func index(syms map[string]*symbol, decl ast.Decl, src []byte, fset *token.FileSet) error {
	add := func(name string, s *symbol) error {
		if name == "_" || name == "init" {
			return nil
		}

		if _, dup := syms[name]; dup {
			return fmt.Errorf("duplicate symbol %s", name)
		}

		s.src, s.fset = src, fset
		syms[name] = s

		return nil
	}

	switch d := decl.(type) {
	case *ast.FuncDecl:
		name := d.Name.Name
		if d.Recv != nil && len(d.Recv.List) == 1 {
			name = receiverName(d.Recv.List[0].Type) + "." + name
		}

		return add(name, &symbol{fn: d})
	case *ast.GenDecl:
		for _, spec := range d.Specs {
			switch s := spec.(type) {
			case *ast.TypeSpec:
				if err := add(s.Name.Name, &symbol{gen: d, spec: s}); err != nil {
					return err
				}
			case *ast.ValueSpec:
				for _, n := range s.Names {
					if err := add(n.Name, &symbol{gen: d, spec: s}); err != nil {
						return err
					}
				}
			}
		}
	}

	return nil
}

func receiverName(expr ast.Expr) string {
	switch e := expr.(type) {
	case *ast.StarExpr:
		return receiverName(e.X)
	case *ast.IndexExpr:
		return receiverName(e.X)
	case *ast.IndexListExpr:
		return receiverName(e.X)
	case *ast.Ident:
		return e.Name
	}

	return ""
}

func (s *symbol) text(from, to token.Pos) string {
	return string(s.src[s.fset.Position(from).Offset:s.fset.Position(to).Offset])
}

func (s *symbol) render(m modifiers) (string, error) {
	if s.fn != nil {
		return s.renderFunc(m)
	}

	if m.body || m.main {
		return "", fmt.Errorf("body and main apply to functions only")
	}

	if s.gen.Lparen.IsValid() {
		// One spec of a group: repeat the keyword so that it reads as a declaration of its own. The padding that
		// aligned it with the rest of the group is collapsed.
		code := s.gen.Tok.String() + " " + collapseSpaces(s.text(s.spec.Pos(), s.spec.End()))

		if doc := specDoc(s.spec); m.doc && doc != nil {
			code = dedent(s.text(doc.Pos(), doc.End())) + "\n" + code
		}

		return code, nil
	}

	from := s.gen.Pos()
	if m.doc && s.gen.Doc != nil {
		from = s.gen.Doc.Pos()
	}

	return s.text(from, s.gen.End()), nil
}

func specDoc(spec ast.Spec) *ast.CommentGroup {
	switch s := spec.(type) {
	case *ast.TypeSpec:
		return s.Doc
	case *ast.ValueSpec:
		return s.Doc
	}

	return nil
}

func (s *symbol) renderFunc(m modifiers) (string, error) {
	fn := s.fn

	if m.body || m.main {
		if fn.Body == nil {
			return "", fmt.Errorf("function has no body")
		}

		body := dedent(s.text(fn.Body.Lbrace+1, fn.Body.Rbrace))
		if !m.main {
			return body, nil
		}

		return "func main() {\n" + indent(body) + "\n}", nil
	}

	from := fn.Pos()
	if m.doc && fn.Doc != nil {
		from = fn.Doc.Pos()
	}

	to := fn.End()
	if m.sig {
		to = fn.Type.End()
	}

	return s.text(from, to), nil
}

// dedent removes the blank lines around the text and the indentation that all lines share.
func dedent(text string) string {
	lines := strings.Split(strings.Trim(text, "\r\n"), "\n")

	indent := -1

	for _, line := range lines {
		if strings.TrimSpace(line) == "" {
			continue
		}

		n := len(line) - len(strings.TrimLeft(line, "\t "))
		if indent < 0 || n < indent {
			indent = n
		}
	}

	for i, line := range lines {
		if len(line) >= indent && indent > 0 {
			lines[i] = line[indent:]
		} else if strings.TrimSpace(line) == "" {
			lines[i] = ""
		}
	}

	return strings.TrimRight(strings.Join(lines, "\n"), " \t\n")
}

func indent(text string) string {
	lines := strings.Split(text, "\n")
	for i, line := range lines {
		if line != "" {
			lines[i] = "\t" + line
		}
	}

	return strings.Join(lines, "\n")
}

// Alignment padding that gofmt puts into a spec of a group: after the names, and before the =.
var (
	padAfterNames = regexp.MustCompile(`^([\pL\pN_]+(?:, [\pL\pN_]+)*) {2,}`)
	padBeforeEq   = regexp.MustCompile(`^([^"]*?\S) {2,}=`)
)

// collapseSpaces removes the alignment padding from a one-line spec of a group. Multi-line specs, such as struct
// types, and the rest of the line, including string literals, are left alone.
func collapseSpaces(text string) string {
	if strings.Contains(text, "\n") {
		return text
	}

	return padBeforeEq.ReplaceAllString(padAfterNames.ReplaceAllString(text, "$1 "), "$1 =")
}
