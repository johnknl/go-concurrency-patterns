# Adding a pattern

A pattern is a Go package and a page. Take [Pipeline](patterns/pipeline.md) and `patterns/pipeline` as the
model.

1. **Code:** create `patterns/<name>/` with
   - `doc.go`: only the package comment;
   - `<name>.go`: the mechanism, as plain standard-library code in unexported functions and types. No exported
     API and no reusable abstractions;
   - `examples_test.go`: `Example_<name>` functions with an `// Output:` comment, which the page shows;
   - `<name>_test.go`: unit tests of the mechanism, including its edge cases and its concurrency properties.

   Examples and tests are internal tests of the package. They must be deterministic and free of data races, must
   not sleep to wait for another goroutine, and must not leave a goroutine behind. Tests of timing use
   `testing/synctest`.
2. **Scope:** a page is one pattern when it has one problem (an intent with no "or"), one mechanism (variants
   only refine it; a variant with its own applicability and consequences is a pattern of its own), its own
   consequences, and a name. A problem area with several patterns is a topic: its patterns get pages of their own,
   and the choice between them is described in the `guide` of the concern in `taxonomy.yml`, shown on the
   [concern](catalog/concern.md) page.
3. **Page:** the intent states the problem, not the mechanism: if a variant on the page could not satisfy the
   intent, the intent or the main mechanism is wrong. Create `docs/patterns/<slug>.md` with the sections listed in
   [Catalog](catalog/index.md#naming-and-layout), in that order, and with the front matter below. `docsync` checks
   the sections and their order.
   Add it to `nav` in `mkdocs.yml`, in the section of its purpose.
4. **Generate:** run `go tool docsync sync`, which fills in the code, the properties, and the sources.
5. **Check:** run `make headers`, then `make check`, and `make mutation` to see which changes to the mechanism
   the tests do not detect.

A pattern that later language or library features have largely replaced keeps its page and sets
`superseded:` in its front matter to one or two sentences that name the simpler option. The option must solve the
problem as the intent states it. `docsync` shows the text as a warning below the lede, marks the
status in the properties, and lists the pattern on the [Superseded](catalog/superseded.md) page.

When a library is the usual way to apply a pattern, `library:` names it in one sentence, and `docsync` shows it as a
note in the same place.

## Concepts and standard library guides

Besides patterns and anti-patterns, the catalog has two other kinds of page.

- **Concept** (`docs/concepts/`): shared background that several patterns rely on, such as who closes a channel or
  what makes a write visible to another goroutine. It is abstract: a definition, why it matters, and its
  properties, each a `###` heading with an id that patterns link to. It has no code package. Sections: Why it
  matters, Properties, Forms (optional), Used by, Violated by (optional), Related concepts (optional), Sources.
  Front matter: `title`, `aliases`, `intent` (the definition), `concerns`, `sources`.
- **Standard library guide** (`docs/stdlib/`): how to use what Go already provides, correctly, with its code in
  `stdlib/<name>/`. Sections: What it provides, Using it, Pitfalls, Related pages, Anti-patterns (optional),
  Sources. Front matter as for a pattern, without `purpose`, `scale` and `mechanisms`, and with `provides`: the
  symbols it covers.

A pattern or anti-pattern page does not repeat background: a point that two pages make belongs in a concept, and the
page links to the property, `[the sender closes](../concepts/channel-ownership.md#sender-closes)`, and declares the
concept in its front matter with the aspect it uses:

```yaml
concepts:
  - channel-ownership: 'each stage closes its own output.'
```

On an anti-pattern, `concepts` lists the concepts it breaks. `docsync` checks that the declared concepts and the
concept links in the text match, and generates the Used by and Violated by lists of each concept.

## The agent skill

`skill/` is generated from the pages by `go tool docsync sync`: `SKILL.md` indexes the principles, the concern
guides, the pages, and the Detection sections of the anti-patterns, and `references/` has every page with its code
filled in. Diagrams, the properties markup and links outside the skill are removed or turned into text. Do not edit
these files; change the pages, and `go tool docsync check` fails when the skill is out of date.

## Writing for the reader

Readers come with a problem and want the code that solves it, its cost, and its limits. Write for readability, not
for a word count.

- The lede is the intent, generated from the front matter; the page does not repeat it.
- Show the mechanism and one example. Each further example starts with a bold title and a sentence on what it
  shows; code is never folded.
- Each variant is a `###` heading under Variants with a brief description of its trade-off, followed by its code,
  if any, below it:

    ```markdown
    ### Bounded acquire

    Make the acquire one case of a `select` with a context, so a caller gives up after a while.

    <!-- code: patterns/semaphore#Example_acquireWithContext:main -->
    <!-- /code -->
    ```

- On anti-pattern pages the faulty and the fixed code are tabs of one Example section:

    ```markdown
    === "Faulty"

        One or two sentences.

        <!-- code: ... -->
        <!-- /code -->

    === "Fixed"

        ...
    ```

- Add a diagram only when it shows more than the code does: several parties, or an order of events.
- Say a thing once. Link to a [principle](index.md#principles), a pattern, or an anti-pattern instead of explaining it
  again.

## Front matter

```yaml
---
title: Pipeline
aliases: [Pipes and filters]   # optional; other names
intent: One sentence, used in the catalogs.
purpose: data-flow              # exactly one
scale: pattern                  # idiom, pattern, or composite; concepts have none
composes:                             # optional; the patterns it is built from, each with its role here
  - bounded-queue: 'each consumer has a buffer.'
superseded: Use X instead.            # optional; see above
library: Use package Y.               # optional; the usual library implementation, shown as a note
package: patterns/pipeline
mechanisms: [channels, select]  # one or more
concepts:                       # optional; the concepts the page relies on, each with the aspect it uses
  - backpressure: 'a full channel blocks the stage before it.'
concerns: [composition]         # one or more
sources:                        # one or more
  - id: cox-buday
    at: 'Ch. 4, "Pipelines"'
  - id: blog-pipelines
---
```

A pattern that no published source describes sets `original: true` instead of `sources`, and its Sources section
reads "Original work.".

Values for `purpose`, `mechanisms`, `concerns`, and the source `id`s come from `taxonomy.yml`, and `docsync`
rejects anything else. Add a value there first when the vocabulary is missing one. A source entry can carry a `url`
to the exact place, next to its `at`.

A page contains these generated parts, and everything between the markers is rewritten:

```markdown
<!-- meta -->             right under the title: the intent, the properties, and the notes
<!-- /meta -->

<!-- composition -->      in Composition, only when `composes` is set
<!-- /composition -->

<!-- sources -->          in Sources
<!-- /sources -->

<!-- usedby -->           on a concept, in Used by
<!-- /usedby -->

<!-- violatedby -->       on a concept, in Violated by
<!-- /violatedby -->
```

## Code blocks

A `code` marker lists the symbols to show:

```markdown
<!-- code: patterns/pipeline#square patterns/pipeline#Example_stages:main -->
<!-- /code -->
```

A reference is `<package directory>#<symbol>[:<modifiers>]`. The directory is relative to the repository root.
The symbol is a function, type, constant, variable, method (`Type.Method`), or example function, from the
package's regular and test files. All references of a marker end up in one Go code block, separated by blank
lines, in the order listed, so a block can combine symbols from several packages.

| Modifier | Effect |
|---|---|
| `doc` | Include the doc comment. |
| `sig` | Functions and methods without the body. |
| `body` | Only the statements inside a function. |
| `main` | The statements inside a function, wrapped in `func main()`. Use it for examples, so the block reads as a program. |

Modifiers combine with commas, except that `body` and `main` stand alone. Markers inside fenced code blocks are not
processed, which is why the examples above are not rewritten.

## Catalogs

`<!-- catalog: purpose -->`, `scale`, `mechanism`, `concern`, and `source` list the patterns grouped by that axis. They
take an optional `level=N` for the heading level of the groups.

`go tool docsync check` fails when anything generated is out of date, and CI runs it.

## Anti-patterns

An anti-pattern is a page in `docs/anti-patterns/` with its code in `antipatterns/<name>/`, written like a pattern
with these differences:

- Front matter has `remedies: [slug, ...]`, the patterns that fix it, instead of `composes`, and `violates: [key,
  ...]`, the [principles](index.md#principles) it violates (keys in `taxonomy.yml`). It has no `scale`.
  The generated `<!-- remedies -->` block lists the remedies.
- Every remedy page needs an `<!-- antipatterns -->` block, which lists the anti-patterns that name it. `docsync`
  checks this.
- Sections: Summary, Symptom, Example, Why it goes wrong, Detection, Fix, Fixed by, Sources.
- The faulty code is a plain unexported function, in `<name>.go`, that an `Example` runs in a controlled setting, so it
  shows the
  symptom deterministically, without a data race under `go test -race`, and leaves no goroutine behind. Show it
  with a `code` marker. The fix is shown the same way, next to it.
- Anti-patterns are left out of the catalogs by purpose, scale, mechanism and concern. They have their own
  catalog.
