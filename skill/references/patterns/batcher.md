# Batcher

Spread a per-call cost over a stream of items while bounding how long any item waits.

- **Also known as:** Micro-batching, Size-or-time batching
- **Purpose:** Data flow
- **Scale:** Pattern
- **Mechanisms:** Channels, Select, Time
- **Concerns:** Batching, Throughput, Latency
- **Concepts:** [Confinement](../concepts/confinement.md), [Timers and Tickers](../concepts/timers.md), [Channel Ownership](../concepts/channel-ownership.md), [Backpressure](../concepts/backpressure.md)
- **Code:** [`patterns/batcher`](https://github.com/johnknl/go-concurrency-patterns/tree/main/patterns/batcher)

## Problem

Items arrive one at a time at a sink with a fixed cost per call, so sending many items per call is cheaper than one.
But when items arrive slowly, a batch that waits until it is full holds them for an unbounded time. A batch has to
go out when it is full or when its oldest item has waited long enough, and the last partial batch when the stream
ends.

## Mechanism

One goroutine owns the batch and a timer, [confined](../concepts/confinement.md#lexical) to it, and selects on both
the input and the timer. It flushes at `maxSize`,
when the timer fires, or when the input is closed, then closes `done` for its owner to wait on.

```go
func batcher(in <-chan string, maxSize int, maxWait time.Duration, flush func([]string)) <-chan struct{} {
	done := make(chan struct{})

	go func() {
		defer close(done)

		timer := time.NewTimer(maxWait)
		timer.Stop()

		var batch []string

		send := func() {
			timer.Stop()
			flush(batch)
			batch = nil
		}

		for {
			select {
			case item, ok := <-in:
				if !ok {
					if len(batch) > 0 {
						send()
					}

					return
				}

				batch = append(batch, item)
				if len(batch) == 1 {
					timer.Reset(maxWait)
				}

				if len(batch) == maxSize {
					send()
				}
			case <-timer.C:
				send()
			}
		}
	}()

	return done
}
```

What makes it correct:

- **The timer runs only while the batch is not empty.** It is created stopped, started by the first item, and
  stopped by every flush, so an empty batcher never wakes up and never flushes an empty batch.
- **One reusable timer.** `send` stops the timer without draining it, which is correct since Go 1.23
  ([synchronous timer channels](../concepts/timers.md#synchronous-channels)).
- **The deadline belongs to the first item.** Later items do not move it, so no item waits longer than `maxWait`
  plus the flush in front of it.
- **Close flushes the rest.** The producer closes `in`; the batcher [drains
  it](../concepts/channel-ownership.md#drain-then-close),
  flushes what it holds and ends. It checks `ok` and returns, since a
  [closed channel is always ready](../concepts/channel-ownership.md#closed-always-ready).
- **The batch is handed over.** `flush` receives the batch and the batcher starts a new one, so the
  [ownership moves](../concepts/confinement.md#hand-off) with the slice.
- **[Bounded](../principles.md#principle-bounded) input.** `in` is buffered; when it is full, a send waits
  ([backpressure](../concepts/backpressure.md#feedback)). A producer that cannot wait uses a [Timed
  Send](timed-send.md) or
  [Load Shedding](load-shedding.md).

An owner goroutine is the [simplest mechanism](../principles.md#principle-simplest-mechanism) here because the timer
must act while no producer is calling; without a time bound a mutex is enough
([Channel Instead of Mutex](../anti-patterns/channel-instead-of-mutex.md)).

**Flushed by size and close, and by time.** In the first example the batch never reaches its one-hour timer; in
the second, the timer flushes a batch of one item.

```go
func main() {
	in := make(chan string, 8)
	done := batcher(in, 3, time.Hour, func(batch []string) { fmt.Println("flush", batch) })

	for _, item := range []string{"a", "b", "c", "d", "e"} {
		in <- item
	}
	close(in)
	<-done

	// Output:
	// flush [a b c]
	// flush [d e]
}

func main() {
	in := make(chan string, 8)
	flushed := make(chan []string)
	done := batcher(in, 100, 10*time.Millisecond, func(batch []string) { flushed <- batch })

	in <- "lonely"
	fmt.Println("flush", <-flushed)

	close(in)
	<-done

	// Output:
	// flush [lonely]
}
```

## Applicability

Use it when a per-call cost dominates and the sink accepts batches, producers do not wait for their own item's
result, and an item may wait for a bounded time. When every caller needs its own result before it continues,
[Group Commit](group-commit.md) batches without a timer or a goroutine, and without the added wait at low load.
When items are rare and the per-call cost is small, send each item directly.

## Consequences

- At most one flush per `maxSize` items plus one per `maxWait`; the wait of an item is bounded, and shutdown loses
  nothing.
- At low load every item waits up to `maxWait`.
- `flush` runs in the batcher's goroutine, so a slow sink fills the input and producers wait. A flush in another
  goroutine needs its own bound on in-flight batches.
- Errors of `flush` have no caller: the sink logs, retries, or reports them on a channel the owner reads.
- The owner must close `in` and wait on `done` before it releases what `flush` uses; a producer that sends after
  the close panics.

## Related patterns

- [Group Commit](group-commit.md) batches synchronous callers without a timer.
- [For-Select Loop](for-select-loop.md) is the loop the batcher runs.

## Anti-patterns

- [Batch Without a Time Bound](../anti-patterns/batch-without-time-bound.md): Collect items into batches that are flushed only when full, so at low load an item waits indefinitely and the last partial batch is lost at shutdown.

## Sources

Original work.
