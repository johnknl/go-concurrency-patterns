# Concepts

Concepts are the shared background that patterns rely on: channel ownership, cancellation, memory visibility, and
similar ideas. Each page defines one idea abstractly and states its properties, without code of its own; pattern pages
link to the specific property they depend on instead of repeating it.

<!-- catalog: concept -->
| Page | Summary |
|---|---|
| [Backpressure](backpressure.md) | Let the consumer's remaining capacity limit the producer's rate, so a faster producer slows down instead of piling up work. |
| [Channel Ownership](channel-ownership.md) | Give every channel one owner, the single sender or the closer that coordinates several, so that a close comes after the last send and is never raced or repeated. |
| [Confinement](confinement.md) | Keep data reachable from only one goroutine at a time, so it needs no synchronization. |
| [Cooperative Cancellation](cooperative-cancellation.md) | Stop work whose result is no longer needed by having the work itself check a signal and return, leaving state consistent. |
| [Determinism](determinism.md) | A computation is deterministic when the same starting state and the same inputs, in the same order, always produce the same result. |
| [Goroutine Leak Prevention](goroutine-leak-prevention.md) | Give every goroutine an owner that is responsible for its end, and a way to end that does not depend on a peer that may have left. |
| [Memory Visibility](memory-visibility.md) | A write made in one goroutine is guaranteed visible in another only if a synchronization edge of the Go memory model orders the two. |
| [Ordering](ordering.md) | Which orders between values a concurrent design preserves, which it loses, and what restoring a lost order costs. |
| [Overload Policy](overload-policy.md) | Decide deliberately what happens when a limit is reached, trading loss, latency, and memory. |
| [Timers and Tickers](timers.md) | The guarantees Go's timers and tickers give about when they fire, what their channels deliver, and when they are released. |
<!-- /catalog -->
