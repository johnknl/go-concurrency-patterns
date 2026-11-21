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
	"bytes"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"

	"gopkg.in/yaml.v3"
)

const repoURL = "https://github.com/johnknl/go-concurrency-patterns/tree/main/"

// axis is one classification: a named list of values.
type axis struct {
	byKey    map[string]value
	Title    string `yaml:"title"`
	Question string `yaml:"question"`
	order    []string
	Values   yaml.Node `yaml:"values"`
}

type value struct {
	Title       string `yaml:"title"`
	Description string `yaml:"description"`
	Guide       string `yaml:"guide"` // optional: how to choose among the patterns of this value
}

type taxonomy struct {
	Sources   map[string]work  `yaml:"sources"`
	axes      map[string]*axis // by axis name, as used in catalog markers
	Purpose   axis             `yaml:"purpose"`
	Scale     axis             `yaml:"scale"`
	Mechanism axis             `yaml:"mechanism"`
	Concern   axis             `yaml:"concern"`
	Principle axis             `yaml:"principle"`
}

type sourceRef struct {
	ID  string `yaml:"id"`
	At  string `yaml:"at"`
	URL string `yaml:"url"`
}

// pattern is the front matter of a pattern page.
type pattern struct {
	err        error           // why the page is invalid, if it is
	markers    map[string]bool // the generated parts the page has
	Library    string          `yaml:"library"` // the usual library implementation, shown as a note
	Package    string          `yaml:"package"`
	Title      string          `yaml:"title"`
	Intent     string          `yaml:"intent"`
	Purpose    string          `yaml:"purpose"`
	Scale      string          `yaml:"scale"`
	path       string          // slash path relative to the repository root
	Superseded string          `yaml:"superseded"`
	Sources    []sourceRef     `yaml:"sources"`
	Composes   []part          `yaml:"composes"`
	Aliases    []string        `yaml:"aliases"` // other names, shown in the properties
	Remedies   []string        `yaml:"remedies"`
	Violates   []string        `yaml:"violates"`
	Mechanisms []string        `yaml:"mechanisms"`
	headings   []string        // the ## headings, in order
	Concerns   []string        `yaml:"concerns"`
	Concepts   []part          `yaml:"concepts"` // concepts a page relies on; on an anti-pattern, those it breaks
	Provides   []string        `yaml:"provides"` // on a stdlib page, the symbols it covers
	links      []string        // the concept slugs the page links to outside generated blocks
	anti       bool            // the page is an anti-pattern
	concept    bool            // the page is a concept: shared background, no code
	stdlib     bool            // the page is a guide to what the standard library provides
	Original   bool            `yaml:"original"`
	hasBlock   bool            // the page has an antipatterns block
}

// site is everything the generated markdown is derived from besides Go code.
type site struct {
	tax      *taxonomy
	nav      string
	patterns []*pattern
}

func loadSite(root string) (*site, error) {
	raw, err := os.ReadFile(filepath.Join(root, "taxonomy.yml")) //nolint:gosec // fixed name in the repository
	if err != nil {
		return nil, err
	}

	var tax taxonomy
	if err = yaml.Unmarshal(raw, &tax); err != nil {
		return nil, fmt.Errorf("taxonomy.yml: %w", err)
	}

	tax.axes = map[string]*axis{
		"purpose": &tax.Purpose, "scale": &tax.Scale, "mechanism": &tax.Mechanism, "concern": &tax.Concern,
	}
	if err = tax.prepare(); err != nil {
		return nil, fmt.Errorf("taxonomy.yml: %w", err)
	}

	if err = tax.Principle.decode(); err != nil {
		return nil, fmt.Errorf("taxonomy.yml: principle: %w", err)
	}

	for name, a := range tax.axes {
		if err = a.decode(); err != nil {
			return nil, fmt.Errorf("taxonomy.yml: %s: %w", name, err)
		}
	}

	s := &site{tax: &tax}

	if nav, navErr := os.ReadFile(filepath.Join(root, "mkdocs.yml")); navErr == nil { //nolint:gosec // fixed name
		s.nav = string(nav)
	}

	var files []string

	for _, dir := range append(slices.Clone(patternDirs), "anti-patterns", "concepts") {
		found, globErr := filepath.Glob(filepath.Join(root, "docs", dir, "*.md"))
		if globErr != nil {
			return nil, globErr
		}

		slices.Sort(found)
		files = append(files, slices.DeleteFunc(found, func(f string) bool { return filepath.Base(f) == "index.md" })...)
	}

	for _, file := range files {
		rel, relErr := filepath.Rel(root, file)
		if relErr != nil {
			return nil, relErr
		}

		// An invalid page only fails what depends on it, so that pages can be written one at a time.
		p, loadErr := loadPattern(file, filepath.ToSlash(rel))
		if loadErr == nil {
			loadErr = s.validate(root, p)
		}

		if p == nil {
			p = &pattern{path: filepath.ToSlash(rel), anti: isAnti(filepath.ToSlash(rel))}
		}

		if loadErr != nil {
			p.err = fmt.Errorf("%s: %w", rel, loadErr)
		}

		s.patterns = append(s.patterns, p)
	}

	for _, p := range s.patterns {
		if p.err != nil {
			continue
		}

		for _, slug := range p.Remedies {
			target, targetErr := s.pattern(s.slugPath(slug))

			switch {
			case targetErr != nil:
				p.err = fmt.Errorf("%s: remedies %q: %w", p.path, slug, targetErr)
			case !target.hasBlock:
				p.err = fmt.Errorf("%s: remedies %q: that page needs an <!-- antipatterns --> block", p.path, slug)
			}

			if p.err != nil {
				break
			}
		}

		for _, c := range p.Composes {
			if _, err = s.pattern(s.slugPath(c.Slug)); err != nil {
				p.err = fmt.Errorf("%s: composes %q: %w", p.path, c.Slug, err)

				break
			}
		}

		if p.err == nil {
			p.err = s.checkConcepts(p)
		}
	}

	return s, nil
}

func (a *axis) decode() error {
	a.byKey = map[string]value{}

	if a.Values.Kind != yaml.MappingNode {
		return errors.New("values must be a mapping")
	}

	for i := 0; i+1 < len(a.Values.Content); i += 2 {
		key := a.Values.Content[i].Value

		var v value
		if err := a.Values.Content[i+1].Decode(&v); err != nil {
			return err
		}

		a.order = append(a.order, key)
		a.byKey[key] = v
	}

	return nil
}

func loadPattern(file, rel string) (*pattern, error) {
	raw, err := os.ReadFile(file) //nolint:gosec // paths come from globbing the repository
	if err != nil {
		return nil, err
	}

	rest, ok := bytes.CutPrefix(raw, []byte("---\n"))
	if !ok {
		return nil, errors.New("missing front matter")
	}

	front, _, ok := bytes.Cut(rest, []byte("\n---\n"))
	if !ok {
		return nil, errors.New("unterminated front matter")
	}

	p := &pattern{
		path:     rel,
		anti:     isAnti(rel),
		concept:  isConcept(rel),
		stdlib:   isStdlib(rel),
		links:    conceptLinks(raw),
		hasBlock: bytes.Contains(raw, []byte("<!-- antipatterns -->")),
		headings: sectionHeadings(raw),
		markers:  markerKinds(raw),
	}

	dec := yaml.NewDecoder(bytes.NewReader(front))
	dec.KnownFields(true)

	if err = dec.Decode(p); err != nil {
		return nil, err
	}

	return p, nil
}

func isAnti(path string) bool { return strings.HasPrefix(path, "docs/anti-patterns/") }

func isConcept(path string) bool { return strings.HasPrefix(path, "docs/concepts/") }

func isStdlib(path string) bool { return strings.HasPrefix(path, "docs/stdlib/") }

// conceptPath is the page of a concept slug.
func conceptPath(slug string) string { return "docs/concepts/" + slug + ".md" }

// conceptLink matches a markdown link to a concept page; the slug is the last path element.
var conceptLink = regexp.MustCompile(`\]\((?:[^)\s]*/)?concepts/([a-z0-9-]+)\.md[)#]`)

// generatedBlock matches a generated part of a page, markers included.
var generatedBlock = regexp.MustCompile(`(?s)<!-- (\w+)(?::[^>]*)? -->.*?<!-- /\w+ -->`)

// conceptLinks lists the concepts a page links to in its own text, leaving out generated blocks and code.
func conceptLinks(raw []byte) []string {
	text := generatedBlock.ReplaceAllString(fencedCode.ReplaceAllString(string(raw), ""), "")

	var out []string

	for _, m := range conceptLink.FindAllStringSubmatch(text, -1) {
		if !slices.Contains(out, m[1]) {
			out = append(out, m[1])
		}
	}

	return out
}

// patternDirs are the directories under docs whose pages can be remedies or parts of a composite.
var patternDirs = []string{"patterns", stdlibKind}

// stdlibKind names standard library guides, as a directory and as a catalog.
const stdlibKind = "stdlib"

// slugPath is the page a slug in front matter refers to: the first of the pattern directories that has it.
func (s *site) slugPath(slug string) string {
	for _, dir := range patternDirs {
		path := "docs/" + dir + "/" + slug + ".md"
		if _, err := s.pattern(path); err == nil || !errors.Is(err, errNotPage) {
			return path
		}
	}

	return "docs/patterns/" + slug + ".md"
}

// validateKind checks the fields that only patterns or only anti-patterns have.
func validateKind(p *pattern) error {
	switch {
	case p.anti && len(p.Remedies) == 0:
		return errors.New("an anti-pattern needs remedies")
	case p.anti && len(p.Violates) == 0:
		return errors.New("an anti-pattern names the principles it violates")
	case p.anti && len(p.Composes) > 0:
		return errors.New("an anti-pattern cannot compose")
	case !p.anti && len(p.Violates) > 0:
		return errors.New("violates belongs on anti-patterns")
	case !p.anti && len(p.Remedies) > 0:
		return errors.New("remedies belong on anti-patterns")
	case p.anti && p.Superseded != "":
		return errors.New("superseded belongs on patterns")
	case p.anti && p.Library != "":
		return errors.New("library belongs on patterns")
	}

	return nil
}

func (s *site) validate(root string, p *pattern) error {
	if err := validateKind(p); err != nil {
		return err
	}

	if err := validateLayout(p); err != nil {
		return err
	}

	if err := validateFields(p); err != nil {
		return err
	}

	if err := s.validateClassification(p); err != nil {
		return err
	}

	if err := s.validateVocabulary(p); err != nil {
		return err
	}

	if p.concept {
		if p.Package != "" {
			return errors.New("a concept has no package")
		}
	} else if info, err := os.Stat(filepath.Join(root, filepath.FromSlash(p.Package))); err != nil || !info.IsDir() {
		return fmt.Errorf("package %q is not a directory", p.Package)
	}

	if s.nav != "" && !strings.Contains(s.nav, strings.TrimPrefix(p.path, "docs/")) {
		return errors.New("not listed in the nav of mkdocs.yml")
	}

	return nil
}

// validateFields checks the front matter fields every page needs, and those that belong to one kind of page.
func validateFields(p *pattern) error {
	switch {
	case p.Title == "":
		return errors.New("title is required")
	case p.Intent == "":
		return errors.New("intent is required")
	case len(p.Mechanisms) == 0 && !p.concept && !p.stdlib:
		return errors.New("at least one mechanism is required")
	case p.stdlib && len(p.Provides) == 0:
		return errors.New("a stdlib page names what it covers in provides")
	case !p.stdlib && len(p.Provides) > 0:
		return errors.New("provides belongs on stdlib pages")
	case p.concept && (len(p.Concepts) > 0 || len(p.Composes) > 0 || p.Superseded != "" || p.Library != ""):
		return errors.New("a concept has no concepts, composes, superseded or library")
	case len(p.Concerns) == 0:
		return errors.New("at least one concern is required")
	case len(p.Sources) == 0 && !p.Original:
		return errors.New("at least one source is required, or original: true")
	case len(p.Sources) > 0 && p.Original:
		return errors.New("original: true cannot be combined with sources")
	}

	return nil
}

// validateClassification checks purpose and scale: patterns and anti-patterns have them, concepts and stdlib guides
// do not.
func (s *site) validateClassification(p *pattern) error {
	if p.concept || p.stdlib {
		if p.Purpose != "" || p.Scale != "" || len(p.Mechanisms) > 0 {
			return errors.New("purpose, scale and mechanisms belong on patterns")
		}
	} else {
		if _, ok := s.tax.Purpose.byKey[p.Purpose]; !ok {
			return fmt.Errorf("unknown purpose %q", p.Purpose)
		}

		if _, ok := s.tax.Scale.byKey[p.Scale]; !ok && (!p.anti || p.Scale != "") {
			return fmt.Errorf("unknown scale %q", p.Scale)
		}
	}

	return nil
}

func (s *site) validateVocabulary(p *pattern) error {
	for _, m := range p.Mechanisms {
		if _, ok := s.tax.Mechanism.byKey[m]; !ok {
			return fmt.Errorf("unknown mechanism %q", m)
		}
	}

	for _, c := range p.Concerns {
		if _, ok := s.tax.Concern.byKey[c]; !ok {
			return fmt.Errorf("unknown concern %q", c)
		}
	}

	for _, v := range p.Violates {
		if _, ok := s.tax.Principle.byKey[v]; !ok {
			return fmt.Errorf("unknown principle %q", v)
		}
	}

	for _, src := range p.Sources {
		if _, ok := s.tax.Sources[src.ID]; !ok {
			return fmt.Errorf("unknown source %q", src.ID)
		}
	}

	return nil
}

var errNotPage = errors.New("not a pattern page")

// checkConcepts checks that the concepts a page declares exist, and that its text links exactly those.
func (s *site) checkConcepts(p *pattern) error {
	if p.concept {
		return nil
	}

	declared := make([]string, len(p.Concepts))

	for i, c := range p.Concepts {
		if _, err := s.pattern(conceptPath(c.Slug)); err != nil {
			return fmt.Errorf("%s: concepts %q: %w", p.path, c.Slug, err)
		}

		declared[i] = c.Slug
	}

	for _, slug := range p.links {
		if !slices.Contains(declared, slug) {
			return fmt.Errorf("%s: links concept %q, which is not in its concepts", p.path, slug)
		}
	}

	for _, slug := range declared {
		if !slices.Contains(p.links, slug) {
			return fmt.Errorf("%s: concept %q is declared but not linked in the text", p.path, slug)
		}
	}

	return nil
}

// unusedConcepts lists the concept pages that no page relies on.
func (s *site) unusedConcepts() []string {
	var out []string

	for _, c := range s.patterns {
		if !c.concept || c.err != nil {
			continue
		}

		slug := strings.TrimSuffix(filepath.Base(c.path), ".md")
		used := slices.ContainsFunc(s.patterns, func(p *pattern) bool {
			return slices.ContainsFunc(p.Concepts, func(x part) bool { return x.Slug == slug })
		})

		if !used {
			out = append(out, c.path)
		}
	}

	return out
}

// renderUsedBy lists the pages that rely on the concept, with the aspect each uses; renderViolatedBy lists the
// anti-patterns that break it.
func (s *site) renderUsedBy(page *pattern, anti bool) (string, error) {
	slug := strings.TrimSuffix(filepath.Base(page.path), ".md")

	var lines []string

	for _, p := range s.sorted(func(p *pattern) bool { return p.anti == anti && !p.concept }) {
		for _, c := range p.Concepts {
			if c.Slug != slug {
				continue
			}

			text := p.Intent
			if c.Role != "" {
				text = c.Role
			}

			lines = append(lines, fmt.Sprintf("- [%s](%s): %s", p.Title, link(page.path, p.path),
				strings.Join(strings.Fields(text), " ")))
		}
	}

	if len(lines) == 0 {
		return "", errors.New("no page names this concept; remove the block or the concept")
	}

	return strings.Join(lines, "\n"), nil
}

func (s *site) pattern(path string) (*pattern, error) {
	for _, p := range s.patterns {
		if p.path == path {
			return p, p.err
		}
	}

	return nil, errNotPage
}

// link is the markdown link from the page at from to the repository file at to.
func link(from, to string) string {
	rel, err := filepath.Rel(filepath.Dir(from), to)
	if err != nil {
		return to
	}

	return filepath.ToSlash(rel)
}

// renderMeta is the opening of a page: the intent as its lede, its properties, and the notes that change what a
// reader does: superseded and library.
func (s *site) renderMeta(page *pattern) string {
	parts := []string{strings.Join(strings.Fields(page.Intent), " "), s.renderProperties(page)}

	if page.Superseded != "" {
		parts = append(parts, "!!! warning \"Largely superseded\"\n    "+strings.Join(strings.Fields(page.Superseded), " "))
	}

	if page.Library != "" {
		parts = append(parts, "!!! note \"Library\"\n    "+strings.Join(strings.Fields(page.Library), " "))
	}

	return strings.Join(parts, "\n\n")
}

func (s *site) conceptTitle(slug string) string {
	if c, err := s.pattern(conceptPath(slug)); err == nil {
		return c.Title
	}

	return slug
}

// renderProperties is the table of a page's other names and classification, linked to the catalog.
func (s *site) renderProperties(page *pattern) string {
	ref := func(axisName, key string, a *axis) string {
		target := "docs/catalog/" + axisName + ".md"

		return fmt.Sprintf("[%s](%s#%s-%s)", a.byKey[key].Title, link(page.path, target), axisName, key)
	}

	join := func(axisName string, keys []string, a *axis) string {
		refs := make([]string, len(keys))
		for i, k := range keys {
			refs[i] = ref(axisName, k, a)
		}

		return strings.Join(refs, ", ")
	}

	rows := []string{"| | |", "|---|---|"}
	row := func(name, value string) { rows = append(rows, "| "+name+" | "+value+" |") }

	if len(page.Aliases) > 0 {
		row("Also known as", strings.Join(page.Aliases, ", "))
	}

	if len(page.Provides) > 0 {
		row("Provides", "`"+strings.Join(page.Provides, "`, `")+"`")
	}

	if page.concept {
		row("Concerns", join("concern", page.Concerns, &s.tax.Concern))

		return "<div class=\"cp-props\" markdown>\n\n" + strings.Join(rows, "\n") + "\n\n</div>"
	}

	if len(page.Violates) > 0 {
		row("Violates", s.principleLinks(page.path, page.Violates))
	}

	if page.Purpose != "" {
		row("Purpose", ref("purpose", page.Purpose, &s.tax.Purpose))
	}

	if page.Scale != "" {
		row("Scale", ref("scale", page.Scale, &s.tax.Scale))
	}

	if page.Superseded != "" {
		row("Status", "[Largely superseded]("+link(page.path, "docs/catalog/superseded.md")+")")
	}

	if len(page.Mechanisms) > 0 {
		row("Mechanisms", join("mechanism", page.Mechanisms, &s.tax.Mechanism))
	}

	row("Concerns", join("concern", page.Concerns, &s.tax.Concern))

	if len(page.Concepts) > 0 {
		refs := make([]string, len(page.Concepts))
		for i, c := range page.Concepts {
			refs[i] = fmt.Sprintf("[%s](%s)", s.conceptTitle(c.Slug), link(page.path, conceptPath(c.Slug)))
		}

		name := "Concepts"
		if page.anti {
			name = "Breaks"
		}

		row(name, strings.Join(refs, ", "))
	}
	row("Code", "[`"+page.Package+"`]("+repoURL+page.Package+")")

	return "<div class=\"cp-props\" markdown>\n\n" + strings.Join(rows, "\n") + "\n\n</div>"
}

// catalogSuperseded lists the patterns that later features have largely replaced, with what to use instead.
func (s *site) catalogSuperseded(from string) string {
	superseded := s.sorted(func(p *pattern) bool { return p.Superseded != "" })
	rows := make([]string, 0, len(superseded)+2)
	rows = append(rows, "| Pattern | Instead |", "|---|---|")

	for _, p := range superseded {
		instead := strings.Join(strings.Fields(p.Superseded), " ")
		rows = append(rows, fmt.Sprintf("| [%s](%s) | %s |", p.Title, link(from, p.path), instead))
	}

	return strings.Join(rows, "\n")
}

func (s *site) renderSources(page *pattern) string {
	if page.Original {
		return "Original work."
	}

	lines := make([]string, len(page.Sources))

	for i, src := range page.Sources {
		lines[i] = "- " + s.sourceText(page.path, src)
	}

	return strings.Join(lines, "\n")
}

func (s *site) sourceText(from string, src sourceRef) string {
	text := "[" + s.tax.Sources[src.ID].label + "](" + link(from, "docs/sources.md") + "#source-" + src.ID + ")"

	if src.At != "" {
		text += ", " + src.At
	}

	if src.URL != "" {
		text += " ([link](" + src.URL + "))"
	}

	return text
}

// renderCatalog lists the patterns grouped by the values of an axis, or by source for "source".
func (s *site) renderCatalog(from, name string, level int, only string) (string, error) {
	for _, p := range s.patterns {
		if p.err != nil {
			return "", p.err
		}
	}

	if name == "source" {
		return s.catalogBySource(from, level, only)
	}

	if name == antiPatternKind {
		return s.catalogAnti(from, level), nil
	}

	if name == "superseded" {
		return s.catalogSuperseded(from), nil
	}

	if name == "concept" || name == stdlibKind {
		rows := []string{"| Page | Summary |", "|---|---|"}
		keep := func(p *pattern) bool { return name == "concept" && p.concept || name == stdlibKind && p.stdlib }
		for _, p := range s.sorted(keep) {
			rows = append(rows, fmt.Sprintf("| [%s](%s) | %s |", p.Title, link(from, p.path), p.Intent))
		}

		return strings.Join(rows, "\n"), nil
	}

	a, ok := s.tax.axes[name]
	if !ok {
		return "", fmt.Errorf("unknown catalog %q", name)
	}

	var out []string

	for _, key := range a.order {
		var rows []string

		onlyPatterns := name != "concern"

		for _, p := range s.sorted(func(p *pattern) bool {
			return !p.anti && hasValue(p, name, key) && (!onlyPatterns || !p.concept && !p.stdlib)
		}) {
			if onlyPatterns {
				rows = append(rows, fmt.Sprintf("| [%s](%s) | %s |", p.Title, link(from, p.path), p.Intent))
			} else {
				rows = append(rows, fmt.Sprintf("| [%s](%s) | %s | %s |", p.Title, link(from, p.path), pageType(p), p.Intent))
			}
		}

		if len(rows) == 0 {
			continue
		}

		v := a.byKey[key]
		table := "| Pattern | Intent |\n|---|---|\n" + strings.Join(rows, "\n")
		if !onlyPatterns {
			table = "| Page | Type | Intent |\n|---|---|---|\n" + strings.Join(rows, "\n")
		}
		out = append(out, heading(level, v.Title, name+"-"+key), v.Description)
		if v.Guide != "" {
			out = append(out, strings.TrimSpace(v.Guide))
		}

		out = append(out, table)
	}

	return strings.Join(out, "\n\n"), nil
}

func (s *site) catalogBySource(from string, level int, only string) (string, error) {
	if only != "" {
		if _, ok := s.tax.Sources[only]; !ok {
			return "", fmt.Errorf("unknown source %q", only)
		}
	}

	var out []string

	for _, id := range s.tax.sortedSources() {
		if only != "" && id != only {
			continue
		}

		type row struct{ at, text string }

		var rows []row

		for _, p := range s.sorted(func(*pattern) bool { return true }) {
			for _, src := range p.Sources {
				if src.ID != id {
					continue
				}

				at := src.At
				if at == "" {
					at = "whole text"
				}

				rows = append(rows, row{src.At, fmt.Sprintf("| [%s](%s) | %s |", p.Title, link(from, p.path), at)})
			}
		}

		if len(rows) == 0 {
			continue
		}

		slices.SortStableFunc(rows, func(a, b row) int { return strings.Compare(a.at, b.at) })

		lines := make([]string, len(rows))
		for i, r := range rows {
			lines[i] = r.text
		}

		w := s.tax.Sources[id]
		out = append(out, heading(level, w.label, "coverage-"+id), w.reference(),
			"| Pattern | Where |\n|---|---|\n"+strings.Join(lines, "\n"))
	}

	return strings.Join(out, "\n\n"), nil
}

// pageType names the kind of a page in a mixed list.
func pageType(p *pattern) string {
	switch {
	case p.concept:
		return "Concept"
	case p.stdlib:
		return "Standard library"
	default:
		return "Pattern"
	}
}

func heading(level int, title, id string) string {
	return strings.Repeat("#", level) + " " + title + " { #" + id + " }"
}

func hasValue(p *pattern, axisName, key string) bool {
	switch axisName {
	case "purpose":
		return p.Purpose == key
	case "scale":
		return p.Scale == key
	case "mechanism":
		return slices.Contains(p.Mechanisms, key)
	default:
		return slices.Contains(p.Concerns, key)
	}
}

func (s *site) sorted(keep func(*pattern) bool) []*pattern {
	var out []*pattern

	for _, p := range s.patterns {
		if keep(p) {
			out = append(out, p)
		}
	}

	slices.SortFunc(out, func(a, b *pattern) int { return strings.Compare(a.Title, b.Title) })

	return out
}

// part is one entry of composes: a pattern slug, and optionally the role it plays in the composing pattern.
// In the front matter it is either a slug, or a mapping from the slug to the role.
type part struct {
	Slug string
	Role string
}

func (c *part) UnmarshalYAML(node *yaml.Node) error {
	switch node.Kind {
	case yaml.ScalarNode:
		c.Slug = node.Value

		return nil
	case yaml.MappingNode:
		if len(node.Content) == 2 {
			c.Slug, c.Role = node.Content[0].Value, node.Content[1].Value

			return nil
		}
	}

	return errors.New("composes: want a slug or slug: role")
}

// renderComposition lists the patterns that the page's pattern is built from.
func (s *site) renderComposition(page *pattern) (string, error) {
	if len(page.Composes) == 0 {
		return "", errors.New("composes is empty")
	}

	lines := make([]string, len(page.Composes))

	for i, c := range page.Composes {
		target := s.slugPath(c.Slug)

		p, err := s.pattern(target)
		if err != nil {
			return "", fmt.Errorf("%s: %w", c.Slug, err)
		}

		text := p.Intent
		if c.Role != "" {
			text = strings.Join(strings.Fields(c.Role), " ")
		}

		lines[i] = fmt.Sprintf("- [%s](%s): %s", p.Title, link(page.path, target), text)
	}

	return strings.Join(lines, "\n"), nil
}

// principleLinks links the principles to their statement on the home page.
func (s *site) principleLinks(from string, keys []string) string {
	parts := make([]string, len(keys))

	for i, k := range keys {
		parts[i] = fmt.Sprintf("[%s](%s#principle-%s)", s.tax.Principle.byKey[k].Title, link(from, "docs/index.md"), k)
	}

	return strings.Join(parts, ", ")
}

// catalogAnti lists the anti-patterns grouped by the principle they violate, with the patterns that fix them.
func (s *site) catalogAnti(from string, level int) string {
	var out []string

	for _, key := range s.tax.Principle.order {
		anti := s.sorted(func(p *pattern) bool { return p.anti && slices.Contains(p.Violates, key) })
		if len(anti) == 0 {
			continue
		}

		rows := make([]string, 0, len(anti)+2)
		rows = append(rows, "| Anti-pattern | What goes wrong | Fixed by |", "|---|---|---|")

		for _, p := range anti {
			fixes := make([]string, len(p.Remedies))

			for i, slug := range p.Remedies {
				title := slug
				if t, err := s.pattern(s.slugPath(slug)); t != nil && err == nil {
					title = t.Title
				}

				fixes[i] = fmt.Sprintf("[%s](%s)", title, link(from, s.slugPath(slug)))
			}

			rows = append(rows, fmt.Sprintf("| [%s](%s) | %s | %s |",
				p.Title, link(from, p.path), p.Intent, strings.Join(fixes, ", ")))
		}

		v := s.tax.Principle.byKey[key]
		title := strings.Repeat("#", level) + " " + v.Title
		lead := v.Description + " See the [principle](" + link(from, "docs/index.md") + "#principle-" + key + ")."
		out = append(out, title, lead, strings.Join(rows, "\n"))
	}

	return strings.Join(out, "\n\n")
}

// renderRemedies lists the patterns that fix the anti-pattern of the page.
func (s *site) renderRemedies(page *pattern) (string, error) {
	if !page.anti {
		return "", errors.New("remedies are for anti-patterns")
	}

	lines := make([]string, len(page.Remedies))

	for i, slug := range page.Remedies {
		target := s.slugPath(slug)

		p, err := s.pattern(target)
		if err != nil {
			return "", fmt.Errorf("%s: %w", slug, err)
		}

		lines[i] = fmt.Sprintf("- [%s](%s): %s", p.Title, link(page.path, target), p.Intent)
	}

	return strings.Join(lines, "\n"), nil
}

// renderAntiPatterns lists the anti-patterns that name the page's pattern as a remedy.
func (s *site) renderAntiPatterns(page *pattern) (string, error) {
	slug := strings.TrimSuffix(filepath.Base(page.path), ".md")

	names := s.sorted(func(p *pattern) bool { return p.anti && slices.Contains(p.Remedies, slug) })
	lines := make([]string, 0, len(names))

	for _, p := range names {
		lines = append(lines, fmt.Sprintf("- [%s](%s): %s", p.Title, link(page.path, p.path), p.Intent))
	}

	if len(lines) == 0 {
		return "", errors.New("no anti-pattern names this pattern as a remedy; remove the block")
	}

	return strings.Join(lines, "\n"), nil
}

// antiPatternKind names anti-pattern pages, as a catalog and as a kind of page.
const antiPatternKind = "anti-pattern"

var fencedCode = regexp.MustCompile("(?s)```.*?```")

// sourcesSection is the last section of every layout.
const sourcesSection = "Sources"

// The layouts of the pages: the ## sections in order, and which of them may be left out.
var (
	patternLayout = []layoutSection{
		{"Problem", true}, {"Mechanism", true}, {"Composition", false}, {"Applicability", true},
		{"Consequences", true}, {"Variants", false}, {"Related patterns", true}, {"Anti-patterns", false},
		{sourcesSection, true},
	}
	conceptLayout = []layoutSection{
		{"Why it matters", true}, {"Properties", true}, {"Forms", false}, {"Used by", true}, {"Violated by", false},
		{"Related concepts", false}, {sourcesSection, true},
	}
	stdlibLayout = []layoutSection{
		{"What it provides", true}, {"Using it", true}, {"Pitfalls", true}, {"Related pages", true},
		{"Anti-patterns", false}, {sourcesSection, true},
	}
	antiLayout = []layoutSection{
		{"Symptoms", true}, {"Example", true}, {"Why it goes wrong", true}, {"Detection", true},
		{"Fixed by", true}, {sourcesSection, true},
	}
	sectionHeading = regexp.MustCompile(`(?m)^## (.+)$`)
	markerKind     = regexp.MustCompile(`(?m)^\s*<!-- (\w+)(?::[^>]*)? -->$`)
)

type layoutSection struct {
	name     string
	required bool
}

func sectionHeadings(raw []byte) []string {
	text := fencedCode.ReplaceAllString(string(raw), "")

	matches := sectionHeading.FindAllStringSubmatch(text, -1)
	out := make([]string, 0, len(matches))

	for _, m := range matches {
		out = append(out, strings.TrimSpace(m[1]))
	}

	return out
}

func markerKinds(raw []byte) map[string]bool {
	out := map[string]bool{}
	for _, m := range markerKind.FindAllStringSubmatch(string(raw), -1) {
		out[m[1]] = true
	}

	return out
}

// validateLayout checks that a page has the sections of its layout, in order, and the generated parts every page
// has.
func validateLayout(p *pattern) error {
	layout := patternLayout

	switch {
	case p.anti:
		layout = antiLayout
	case p.concept:
		layout = conceptLayout
	case p.stdlib:
		layout = stdlibLayout
	}

	i := 0

	for _, h := range p.headings {
		for i < len(layout) && layout[i].name != h {
			if layout[i].required {
				return fmt.Errorf("section %q is missing before %q", layout[i].name, h)
			}
			i++
		}

		if i == len(layout) {
			return fmt.Errorf("section %q is not part of the layout, or out of order", h)
		}
		i++
	}

	for ; i < len(layout); i++ {
		if layout[i].required {
			return fmt.Errorf("section %q is missing", layout[i].name)
		}
	}

	for _, kind := range []string{"meta", "sources"} {
		if !p.markers[kind] {
			return fmt.Errorf("the %s marker is missing", kind)
		}
	}

	if p.concept && !p.markers["usedby"] {
		return errors.New("the usedby marker is missing")
	}

	if len(p.Composes) > 0 && !p.markers["composition"] {
		return errors.New("composes is set, but the composition marker is missing")
	}

	return nil
}
