# Or-Done

Stop reading a channel you do not control when your own work is canceled.

- **Purpose:** Data flow
- **Scale:** Idiom
- **Status:** Largely superseded
- **Mechanisms:** Channels, Select
- **Concerns:** Composition, Cancellation, Leak prevention
- **Concepts:** [Goroutine Leak Prevention](../concepts/goroutine-leak-prevention.md), [Cooperative Cancellation](../concepts/cooperative-cancellation.md)
- **Code:** [`patterns/ordone`](https://github.com/johnknl/go-concurrency-patterns/tree/main/patterns/ordone)

> **Largely superseded.**
> Done channels have been largely superseded by `context.Context`. The wrapper works the same with `ctx.Done()` in place of `done`; only code that still hands out bare done channels needs the done-channel form.

## Problem

A consumer that reads from a channel owned by other code can block for as long as that code neither sends nor
closes. Adding a `select` on `done` at every receive makes each loop longer and hides its logic.

## Mechanism

The wrapper takes a done channel (or a context) and a channel, and returns a channel. A goroutine selects on both for
each receive and forwards values; it returns, and closes its output, when the input or done is closed. The send is also
guarded by done, so the goroutine does not leak on a consumer that has left
([select on done](../concepts/goroutine-leak-prevention.md#select-on-done)).

```go
func orDone(done <-chan struct{}, in <-chan int) <-chan int {
	out := make(chan int)

	go func() {
		defer close(out)

		for {
			select {
			case <-done:
				return
			case v, ok := <-in:
				if !ok {
					return
				}

				select {
				case out <- v:
				case <-done:
					return
				}
			}
		}
	}()

	return out
}

func main() {
	done := make(chan struct{})
	defer close(done)

	for v := range orDone(done, source()) {
		fmt.Println(v)

		if v == 3 {
			break
		}
	}

	// Output:
	// 1
	// 2
	// 3
}
```

## Applicability

Use it when a consumer reads from a channel that has no cancellation of its own and the same `select` on done would
otherwise be repeated in several loops. For a single loop, write the `select` inline. The in-memory source in the
example stands in for one that blocks on its own ([Channel Generator](../anti-patterns/channel-generator.md)).

## Consequences

- Each value passes through an extra goroutine and channel handoff.
- The wrapper stops only its own goroutine; the source needs its own way to end.
- A value already received from the source when done closes is dropped.
- When both done and the input are ready, `select` [picks one at
  random](../concepts/cooperative-cancellation.md#select-random),
  so a few more values may be forwarded after done is closed.

## Related patterns

- [Tee](tee.md) can read its input through an Or-Done.
- [Pipeline](pipeline.md) stages use the same `select` in each stage.

## Sources

- [Cox-Buday (2017)](../sources.md#source-cox-buday), Ch. 4, "The or-done-channel"
