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
	"encoding/json"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"
)

const fixture = `package p

// Max is the limit.
const Max = 3

const (
	// A is the first.
	A    = 1
	B    = 2
	Long = 3
)

// Box holds a value.
type Box[T any] struct{ v T }

// Get returns the value.
func (b *Box[T]) Get() T {
	return b.v
}

// Run does the work.
func Run(n int) int {
	if n > 0 {
		return n
	}

	return 0
}
`

func writeFixture(t *testing.T) *loader {
	t.Helper()

	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, "p"), 0o755); err != nil {
		t.Fatal(err)
	}

	if err := os.WriteFile(filepath.Join(dir, "p", "p.go"), []byte(fixture), 0o644); err != nil {
		t.Fatal(err)
	}

	return newLoader(dir)
}

func TestLoader_render(t *testing.T) {
	l := writeFixture(t)

	tests := map[string]struct {
		want string
		refs []string
	}{
		"const without doc":       {"const Max = 3", []string{"p#Max"}},
		"const with doc":          {"// Max is the limit.\nconst Max = 3", []string{"p#Max:doc"}},
		"grouped const":           {"const B = 2", []string{"p#B"}},
		"grouped const with doc":  {"// A is the first.\nconst A = 1", []string{"p#A:doc"}},
		"generic type":            {"type Box[T any] struct{ v T }", []string{"p#Box"}},
		"method signature":        {"func (b *Box[T]) Get() T", []string{"p#Box.Get:sig"}},
		"function body":           {"if n > 0 {\n\treturn n\n}\n\nreturn 0", []string{"p#Run:body"}},
		"function as main":        {"func main() {\n\tif n > 0 {\n\t\treturn n\n\t}\n\n\treturn 0\n}", []string{"p#Run:main"}},
		"several in one block":    {"const Max = 3\n\nfunc Run(n int) int", []string{"p#Max", "p#Run:sig"}},
		"method with doc and sig": {"// Get returns the value.\nfunc (b *Box[T]) Get() T", []string{"p#Box.Get:doc,sig"}},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			got, err := l.render(tc.refs)
			if err != nil {
				t.Fatal(err)
			}

			if got != tc.want {
				t.Fatalf("got:\n%s\nwant:\n%s", got, tc.want)
			}
		})
	}

	for _, ref := range []string{"p#Missing", "q#X", "p#Run:nope", "p#Max:body", "p#Run:body,sig", "p#Run:main,doc", "p#Max:main", "Run"} {
		if _, err := l.render([]string{ref}); err == nil {
			t.Errorf("%s: expected an error", ref)
		}
	}
}

func TestRender(t *testing.T) {
	l := writeFixture(t)

	doc := "text\n<!-- code: p#Max -->\nstale\n<!-- /code -->\nmore\n"
	want := "text\n<!-- code: p#Max -->\n```go\nconst Max = 3\n```\n<!-- /code -->\nmore\n"

	got, err := render(doc, "doc.md", l, nil)
	if err != nil || got != want {
		t.Fatalf("got %q, %v", got, err)
	}

	again, err := render(got, "doc.md", l, nil)
	if err != nil || again != got {
		t.Fatalf("not idempotent: %q, %v", again, err)
	}

	t.Run("ignores markers in fenced code", func(t *testing.T) {
		in := "```markdown\n<!-- code: nope#X -->\n<!-- /code -->\n```\n"

		if got, err := render(in, "doc.md", l, nil); err != nil || got != in {
			t.Fatalf("got %q, %v", got, err)
		}
	})

	t.Run("unclosed marker", func(t *testing.T) {
		if _, err := render("<!-- code: p#Max -->\n", "doc.md", l, nil); err == nil || !strings.Contains(err.Error(), "missing") {
			t.Fatalf("got %v", err)
		}
	})
}

func TestSite(t *testing.T) {
	root := t.TempDir()

	write := func(name, content string) {
		t.Helper()

		path := filepath.Join(root, name)
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}

		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	write("taxonomy.yml", `
principle:
  values:
    ownership: {title: Clear ownership, description: One owner.}
purpose:
  values:
    data-flow: {title: Data flow, description: Values move., guide: Choose by the consumer.}
scale:
  values:
    idiom: {title: Idiom, description: Small.}
    composite: {title: Composite, description: Built from others.}
mechanism:
  values:
    channels: {title: Channels, description: Send and receive.}
concern:
  values:
    backpressure: {title: Backpressure, description: Wait for the consumer., guide: Prefer waiting.}
sources:
  book: {type: book, authors: ["Doe, Jane Q.", "Roe, Rich"], year: 2020, title: "The book", publisher: Press, url: "https://example.com/book"}
  talk-a: {type: talk, authors: ["Doe, Jane Q."], year: 2021, title: Alpha, container: Conf}
  talk-b: {type: talk, authors: ["Doe, Jane Q."], year: 2021, title: Beta, container: Conf}
`)
	write("mkdocs.yml", "nav:\n  - patterns/pipe.md\n  - patterns/hub.md\n  - patterns/solo.md\n  - anti-patterns/leaky.md\n  - concepts/flow.md\n  - stdlib/once.md\n")
	write("patterns/pipe/doc.go", "package pipe\n\nconst Max = 3\n")
	write("docs/patterns/pipe.md", `---
title: Pipe
aliases: [Chain, Line]
intent: Connect stages.
purpose: data-flow
scale: idiom
superseded: Use a loop.
library: Use pkg.
package: patterns/pipe
mechanisms: [channels]
concerns: [backpressure]
sources:
  - id: book
    at: ch. 1
  - id: talk-a
  - id: talk-b
---
# Pipe

<!-- meta -->
<!-- /meta -->

## Problem

x

## Mechanism

??? example "More"
    <!-- code: patterns/pipe#Max -->
    <!-- /code -->

## Applicability

x

## Consequences

x

## Related patterns

x

## Anti-patterns

<!-- antipatterns -->
<!-- /antipatterns -->

## Sources

<!-- sources -->
<!-- /sources -->

`)
	write("patterns/hub/doc.go", "package hub\n")
	write("docs/patterns/hub.md", `---
title: Hub
intent: Combine pipes.
purpose: data-flow
scale: composite
composes:
  - pipe: the stages

package: patterns/hub
mechanisms: [channels]
concerns: [backpressure]
sources:
  - id: book
---
# Hub

<!-- meta -->
<!-- /meta -->

## Problem

## Mechanism

## Composition

<!-- composition -->
<!-- /composition -->

## Applicability

## Consequences

## Related patterns

## Sources

<!-- sources -->
<!-- /sources -->

`)
	write("patterns/solo/doc.go", "package solo\n")
	write("docs/patterns/solo.md", `---
title: Solo
intent: Stand alone.
purpose: data-flow
scale: idiom
original: true
package: patterns/solo
mechanisms: [channels]
concerns: [backpressure]
concepts:
  - flow: 'the send waits'
---
# Solo

<!-- meta -->
<!-- /meta -->

## Problem

## Mechanism

It [waits](../concepts/flow.md#wait).

## Applicability

## Consequences

## Related patterns

## Sources

<!-- sources -->
<!-- /sources -->

`)
	write("antipatterns/leaky/doc.go", "package leaky\n")
	write("docs/anti-patterns/leaky.md", `---
title: Leaky
intent: Forget to stop.
purpose: data-flow
package: antipatterns/leaky
mechanisms: [channels]
concerns: [backpressure]
remedies: [pipe]
violates: [ownership]
concepts: [flow]
original: true
---
# Leaky

<!-- meta -->
<!-- /meta -->

## Symptoms

## Example

## Why it goes wrong

It ignores [flow](../concepts/flow.md).

## Detection

- Look at [Pipe](../patterns/pipe.md) and the [catalog](../catalog/purpose.md).

## Fixed by

<!-- remedies -->
<!-- /remedies -->

## Sources

<!-- sources -->
<!-- /sources -->

`)
	write("docs/concepts/flow.md", `---
title: Flow
intent: Producers wait for consumers.
concerns: [backpressure]
original: true
---
# Flow

<!-- meta -->
<!-- /meta -->

## Why it matters

## Properties

### Wait { #wait }

## Used by

<!-- usedby -->
<!-- /usedby -->

## Violated by

<!-- violatedby -->
<!-- /violatedby -->

## Sources

<!-- sources -->
<!-- /sources -->
`)
	write("stdlib/once/doc.go", "package once\n")
	write("docs/stdlib/once.md", `---
title: Once
intent: Run setup once.
provides: [sync.Once]
package: stdlib/once
concerns: [backpressure]
original: true
---
# Once

<!-- meta -->
<!-- /meta -->

## What it provides

## Using it

## Pitfalls

## Related pages

## Sources

<!-- sources -->
<!-- /sources -->
`)
	write("docs/concepts/index.md", "<!-- catalog: concept -->\n<!-- /catalog -->\n<!-- catalog: concern level=2 -->\n<!-- /catalog -->\n")
	write("docs/index.md", "# Home\n\n## Principles\n\n### 1. Clear ownership { #principle-ownership }\n\nOne owner. See [Pipe](patterns/pipe.md).\n\n"+
		"## Finding your way\n\n<!-- catalog: purpose level=2 -->\n<!-- /catalog -->\n<!-- catalog: source id=book -->\n<!-- /catalog -->\n<!-- catalog: anti-pattern -->\n<!-- /catalog -->\n<!-- catalog: superseded -->\n<!-- /catalog -->\n")
	write("docs/sources.md", "<!-- bibliography -->\n<!-- /bibliography -->\n")

	if err := run(root, false, nil); err != nil {
		t.Fatal(err)
	}

	page := read(t, root, "docs/patterns/pipe.md")
	for _, want := range []string{
		"<!-- meta -->\nConnect stages.\n\n<div class=\"cp-props\" markdown>\n\n| | |\n|---|---|\n" +
			"| Also known as | Chain, Line |\n| Purpose | [Data flow](../catalog/purpose.md#purpose-data-flow) |\n" +
			"| Scale | [Idiom](../catalog/scale.md#scale-idiom) |\n| Status | [Largely superseded](../catalog/superseded.md) |",
		"| Concerns | [Backpressure](../catalog/concern.md#concern-backpressure) |",
		"| Code | [`patterns/pipe`](" + repoURL + "patterns/pipe) |\n\n</div>\n\n!!! warning \"Largely superseded\"\n" +
			"    Use a loop.\n\n!!! note \"Library\"\n    Use pkg.\n<!-- /meta -->",
		"    <!-- code: patterns/pipe#Max -->\n\n    ```go\n    const Max = 3\n    ```\n\n    <!-- /code -->",
		"- [Doe and Roe (2020)](../sources.md#source-book), ch. 1",
	} {
		if !strings.Contains(page, want) {
			t.Errorf("page lacks %q:\n%s", want, page)
		}
	}

	index := read(t, root, "docs/index.md")
	for _, want := range []string{
		"## Data flow { #purpose-data-flow }\n\nValues move.\n\nChoose by the consumer.",
		"| [Pipe](patterns/pipe.md) | Connect stages. |",
		"| [Pipe](patterns/pipe.md) | ch. 1 |",
		"| [Leaky](anti-patterns/leaky.md) | Forget to stop. | [Pipe](patterns/pipe.md) |",
		"| [Pipe](patterns/pipe.md) | Use a loop. |",
		"## Clear ownership",
		"Doe, J. Q., & Roe, R. (2020). *The book*. Press. <https://example.com/book>",
	} {
		if !strings.Contains(index, want) {
			t.Errorf("index lacks %q:\n%s", want, index)
		}
	}

	if hub := read(t, root, "docs/patterns/hub.md"); !strings.Contains(hub, "- [Pipe](pipe.md): the stages") {
		t.Errorf("composition missing:\n%s", hub)
	}

	if got := read(t, root, "docs/patterns/pipe.md"); !strings.Contains(got, "- [Leaky](../anti-patterns/leaky.md): Forget to stop.") {
		t.Errorf("antipatterns block missing:\n%s", got)
	}

	leaky := read(t, root, "docs/anti-patterns/leaky.md")
	if !strings.Contains(leaky, "- [Pipe](../patterns/pipe.md): Connect stages.") || strings.Contains(leaky, "| Scale |") ||
		!strings.Contains(leaky, "| Violates | [Clear ownership](../index.md#principle-ownership) |") {
		t.Errorf("remedies block wrong:\n%s", leaky)
	}

	skillPipe := read(t, root, "skill/references/patterns/pipe.md")
	for _, want := range []string{
		"# Pipe\n\nConnect stages.\n\n- **Also known as:** Chain, Line\n- **Purpose:** Data flow\n",
		"> **Largely superseded.**\n> Use a loop.",
		"const Max = 3",
		"- [Doe and Roe (2020)](../sources.md#source-book), ch. 1",
	} {
		if !strings.Contains(skillPipe, want) {
			t.Errorf("skill page lacks %q:\n%s", want, skillPipe)
		}
	}

	for _, unwanted := range []string{"<!--", "<div", "title: Pipe", "(../catalog/", "mermaid"} {
		if strings.Contains(skillPipe, unwanted) {
			t.Errorf("skill page has %q:\n%s", unwanted, skillPipe)
		}
	}

	skillIndex := read(t, root, "skill/SKILL.md")
	for _, want := range []string{
		"---\nname: go-concurrency\ndescription: \"Use when",
		"### 1. Clear ownership\n\nOne owner. See [Pipe](references/patterns/pipe.md).",
		"- [Pipe](references/patterns/pipe.md): Connect stages.",
		"### [Leaky](references/anti-patterns/leaky.md)",
		"Violates: Clear ownership. Fixed by: [Pipe](references/patterns/pipe.md).",
		"Look for:\n\n- Look at [Pipe](references/patterns/pipe.md) and the catalog.",
	} {
		if !strings.Contains(skillIndex, want) {
			t.Errorf("SKILL.md lacks %q:\n%s", want, skillIndex)
		}
	}

	front, _, _ := strings.Cut(strings.TrimPrefix(skillIndex, "---\n"), "\n---\n")

	var meta struct {
		Name        string `yaml:"name"`
		Description string `yaml:"description"`
	}
	if err := yaml.Unmarshal([]byte(front), &meta); err != nil || meta.Name != "go-concurrency" ||
		!strings.HasPrefix(meta.Description, "Use when") {
		t.Errorf("SKILL.md front matter does not parse: %v %+v", err, meta)
	}

	if got := read(t, root, "skill/references/principles.md"); !strings.HasPrefix(got, "# Principles\n") {
		t.Errorf("principles wrong:\n%s", got)
	}

	flow := read(t, root, "docs/concepts/flow.md")
	for _, want := range []string{
		"| Concerns | [Backpressure](../catalog/concern.md#concern-backpressure) |\n\n</div>",
		"<!-- usedby -->\n- [Solo](../patterns/solo.md): the send waits\n<!-- /usedby -->",
		"<!-- violatedby -->\n- [Leaky](../anti-patterns/leaky.md): Forget to stop.\n<!-- /violatedby -->",
	} {
		if !strings.Contains(flow, want) {
			t.Errorf("concept lacks %q:\n%s", want, flow)
		}
	}

	if strings.Contains(flow, "| Code |") || strings.Contains(flow, "| Purpose |") {
		t.Errorf("concept shows pattern properties:\n%s", flow)
	}

	if once := read(t, root, "docs/stdlib/once.md"); !strings.Contains(once, "| Provides | `sync.Once` |") ||
		strings.Contains(once, "| Purpose |") {
		t.Errorf("stdlib properties wrong:\n%s", once)
	}

	concepts := read(t, root, "docs/concepts/index.md")
	for _, want := range []string{
		"| [Flow](flow.md) | Producers wait for consumers. |",
		"| [Solo](../patterns/solo.md) | Pattern | Stand alone. |",
		"| [Flow](flow.md) | Concept | Producers wait for consumers. |",
		"| [Once](../stdlib/once.md) | Standard library | Run setup once. |",
	} {
		if !strings.Contains(concepts, want) {
			t.Errorf("concepts index lacks %q:\n%s", want, concepts)
		}
	}

	if solo := read(t, root, "docs/patterns/solo.md"); !strings.Contains(solo, "| Concepts | [Flow](../concepts/flow.md) |") {
		t.Errorf("concepts row missing:\n%s", solo)
	}

	if solo := read(t, root, "docs/patterns/solo.md"); !strings.Contains(solo, "<!-- sources -->\nOriginal work.\n<!-- /sources -->") {
		t.Errorf("original work missing:\n%s", solo)
	}

	biblio := read(t, root, "docs/sources.md")
	for _, want := range []string{
		`<span id="source-book"></span>Doe, J. Q., & Roe, R. (2020).`,
		"Doe, J. Q. (2021a). *Alpha* [Conference presentation]. Conf.",
		"Doe, J. Q. (2021b). *Beta* [Conference presentation]. Conf.",
		"[Pipe](patterns/pipe.md) (ch. 1)",
	} {
		if !strings.Contains(biblio, want) {
			t.Errorf("bibliography lacks %q:\n%s", want, biblio)
		}
	}

	if err := run(root, true, nil); err != nil {
		t.Fatalf("not idempotent: %v", err)
	}

	t.Run("publishes the skill as an HTTP catalog", func(t *testing.T) {
		dest := t.TempDir()

		if err := writeCatalog(root, dest); err != nil {
			t.Fatal(err)
		}

		var index struct {
			Skills []struct {
				Name    string   `json:"name"`
				Version string   `json:"version"`
				Files   []string `json:"files"`
			} `json:"skills"`
		}

		raw, err := os.ReadFile(filepath.Join(dest, "skill", "index.json"))
		if err != nil {
			t.Fatal(err)
		}

		if err = json.Unmarshal(raw, &index); err != nil || len(index.Skills) != 1 {
			t.Fatalf("index.json: %v %s", err, raw)
		}

		first := index.Skills[0]
		if first.Name != "go-concurrency" || first.Version == "" || !slices.Contains(first.Files, "SKILL.md") ||
			!slices.Contains(first.Files, "references/patterns/pipe.md") {
			t.Errorf("index.json = %+v", first)
		}

		if _, err = os.Stat(filepath.Join(dest, "skill", "go-concurrency", "references", "patterns", "pipe.md")); err != nil {
			t.Errorf("file not published: %v", err)
		}

		write("skill/references/patterns/pipe.md", read(t, root, "skill/references/patterns/pipe.md")+"changed\n")

		again := t.TempDir()
		if err = writeCatalog(root, again); err != nil {
			t.Fatal(err)
		}

		raw2, _ := os.ReadFile(filepath.Join(again, "skill", "index.json"))
		if string(raw2) == string(raw) {
			t.Error("the version did not change with the content")
		}

		if err = run(root, false, nil); err != nil {
			t.Fatal(err)
		}
	})

	t.Run("a changed or leftover skill file is out of date", func(t *testing.T) {
		original := read(t, root, "skill/references/patterns/pipe.md")

		write("skill/references/patterns/pipe.md", original+"edited\n")

		if err := run(root, true, nil); err == nil || !strings.Contains(err.Error(), "skill/references/patterns/pipe.md") {
			t.Fatalf("changed file: got %v", err)
		}

		if err := run(root, false, nil); err != nil {
			t.Fatal(err)
		}

		write("skill/references/patterns/old.md", "left over\n")

		if err := run(root, true, nil); err == nil || !strings.Contains(err.Error(), "skill/references/patterns/old.md") {
			t.Fatalf("leftover file: got %v", err)
		}

		if err := run(root, false, nil); err != nil {
			t.Fatal(err)
		}

		if _, err := os.Stat(filepath.Join(root, "skill", "references", "patterns", "old.md")); !os.IsNotExist(err) {
			t.Fatalf("leftover file kept: %v", err)
		}

		if got := read(t, root, "skill/references/patterns/pipe.md"); got != original {
			t.Fatalf("file not regenerated:\n%s", got)
		}

	})

	t.Run("rejects unknown values", func(t *testing.T) {
		write("docs/patterns/pipe.md", strings.Replace(page, "concerns: [backpressure]", "concerns: [nope]", 1))

		if err := run(root, true, nil); err == nil || !strings.Contains(err.Error(), "unknown concern") {
			t.Fatalf("got %v", err)
		}
	})

	t.Run("checks concept links against concepts", func(t *testing.T) {
		write("docs/patterns/pipe.md", strings.Replace(page, "## Mechanism\n", "## Mechanism\n\nSee [flow](../concepts/flow.md).\n", 1))

		if err := run(root, true, nil); err == nil || !strings.Contains(err.Error(), "not in its concepts") {
			t.Fatalf("got %v", err)
		}

		solo := read(t, root, "docs/patterns/solo.md")
		write("docs/patterns/pipe.md", page)
		write("docs/patterns/solo.md", strings.Replace(solo, "It [waits](../concepts/flow.md#wait).", "It waits.", 1))

		if err := run(root, true, nil); err == nil || !strings.Contains(err.Error(), "declared but not linked") {
			t.Fatalf("got %v", err)
		}
	})
}

func read(t *testing.T, root, name string) string {
	t.Helper()

	raw, err := os.ReadFile(filepath.Join(root, name))
	if err != nil {
		t.Fatal(err)
	}

	return string(raw)
}

func TestCollapseSpaces(t *testing.T) {
	for in, want := range map[string]string{
		"A    = 1":                 "A = 1",
		"mu       sync.Mutex":      "mu sync.Mutex",
		"a, b   int    = 1, 2":     "a, b int = 1, 2",
		`msg   = "keep   this"`:    `msg = "keep   this"`,
		"x int":                    "x int",
		"s struct {\n\ta   int\n}": "s struct {\n\ta   int\n}",
	} {
		if got := collapseSpaces(in); got != want {
			t.Errorf("collapseSpaces(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestValidateLayout(t *testing.T) {
	page := func(headings ...string) *pattern {
		return &pattern{
			headings: headings,
			markers:  map[string]bool{"meta": true, "sources": true},
		}
	}

	ok := page("Problem", "Mechanism", "Applicability", "Consequences", "Variants", "Related patterns", "Sources")
	if err := validateLayout(ok); err != nil {
		t.Fatalf("valid layout rejected: %v", err)
	}

	for name, p := range map[string]*pattern{
		"missing section": page("Problem", "Mechanism", "Applicability", "Related patterns", "Sources"),
		"out of order":    page("Mechanism", "Problem", "Applicability", "Consequences", "Related patterns", "Sources"),
		"unknown section": page("Problem", "Intent", "Mechanism", "Applicability", "Consequences", "Related patterns", "Sources"),
	} {
		if err := validateLayout(p); err == nil {
			t.Errorf("%s: no error", name)
		}
	}

	noMarker := page("Problem", "Mechanism", "Applicability", "Consequences", "Related patterns", "Sources")
	delete(noMarker.markers, "sources")

	if err := validateLayout(noMarker); err == nil {
		t.Error("missing sources marker: no error")
	}
}
