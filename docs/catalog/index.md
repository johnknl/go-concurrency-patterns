# Catalog

Concurrency patterns do not divide into one tidy tree, so every pattern is classified along four independent
axes. Each page shows its classification at the top, and each axis has a page that lists the patterns by it.
The navigation follows the purpose. The vocabulary is defined in `taxonomy.yml`.

| Axis | Question | Values per pattern |
|---|---|---|
| [Purpose](purpose.md) | What is the pattern for? | One. The navigation follows it. |
| [Scale](scale.md) | How big is the pattern? | One. Composites list the patterns they combine. |
| [Mechanism](mechanism.md) | Which Go building blocks does it use? | One or more. |
| [Concern](concern.md) | Which problem does it address? | One or more. |

Patterns that later features have largely replaced are marked [superseded](superseded.md).

## Naming and layout

Patterns carry their established name, such as Worker Pool or Future, with other names listed under *Also known as* in
the properties at the top. Where Go's idiom differs from the textbook form, the page says so. Every pattern lists its
[sources](../sources.md).

Every pattern page, whatever its scale, has the same layout:

| Section | Content |
|---|---|
| Lede | The intent in one sentence, the properties (other names, classification, code), and notes that change what a reader does: a library to use, or a newer option. |
| Problem | The general problem the pattern solves, not an example of it. |
| Mechanism | The pattern in plain Go and what makes it correct. Further examples follow it. |
| Composition | For a pattern built from others: the role each part plays. |
| Applicability | When to use it, and when something simpler or different fits better. |
| Consequences | What it costs, and the ways it goes wrong. |
| Variants | Alternative forms with a different trade-off, when there are any: one subheading each, with its code. |
| Related patterns | Neighbors, alternatives, and combinations. |
| Anti-patterns | The anti-patterns it fixes, when there are any. |
| Sources | Where the pattern is described. |

An anti-pattern page has the lede with the principle it violates, then Symptoms, Example (the faulty and the fixed
code side by side), Why it goes wrong, Detection, Fixed by, and Sources.

[Concepts](../concepts/index.md) hold the background that several patterns share, so pattern pages link to it
instead of repeating it: Why it matters, Properties, Forms, Used by, Violated by, Related concepts, and Sources.
[Standard library guides](../stdlib/index.md) show how to use what Go already provides: What it provides, Using it,
Pitfalls, Related pages, and Sources. The catalogs by purpose, scale and mechanism list only patterns; the catalog
by concern lists all three kinds.

## Conventions in the code

The code follows the [principles](../index.md#principles). In addition:


- Standard library only. A pattern is a way of arranging goroutines, channels, `sync` and `context`, so the code
  shows that arrangement directly. Where a third-party package is the usual choice, the page says so.
- The code is the smallest that is correct. It is meant to be copied into a program and adapted.
- Every goroutine has a way to stop: a closed input, a canceled `context.Context`, or a buffered send that
  cannot block.
- The goroutine that sends on a channel is the one that closes it.
- The code is tested. The mechanism of each pattern is a small package, the samples on the page are `Example`
  functions that `go test` runs and compares with their `// Output:` comment, and unit tests cover the rest of the
  contract. Mutation testing checks that the tests notice when the mechanism is broken: a missing close, a select
  without its cancellation case, an unlocked critical section, a different channel buffer.
