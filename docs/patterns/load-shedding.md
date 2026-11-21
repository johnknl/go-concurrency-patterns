---
title: Load Shedding
aliases: ['Dropping', 'Tail drop (drop newest)', 'Head drop (drop oldest)']
intent: Keep the producer and the latency bounded under overload by dropping the work that does not fit.
purpose: resilience
scale: idiom
package: patterns/loadshedding
mechanisms: [channels, select]
concerns: [overload, latency]
sources:
  - id: nygard-2018
    at: 'Ch. 5, "Shed Load"'
  - id: sre-overload
concepts:
  - backpressure: 'it is the choice when the source cannot wait; items that must not be lost need backpressure.'
  - overload-policy: 'drop newest and drop oldest, with the drops counted.'
---
# Load Shedding

<!-- meta -->
Keep the producer and the latency bounded under overload by dropping the work that does not fit.

<div class="cp-props" markdown>

| | |
|---|---|
| Also known as | Dropping, Tail drop (drop newest), Head drop (drop oldest) |
| Purpose | [Resilience](../catalog/purpose.md#purpose-resilience) |
| Scale | [Idiom](../catalog/scale.md#scale-idiom) |
| Mechanisms | [Channels](../catalog/mechanism.md#mechanism-channels), [Select](../catalog/mechanism.md#mechanism-select) |
| Concerns | [Overload](../catalog/concern.md#concern-overload), [Latency](../catalog/concern.md#concern-latency) |
| Concepts | [Backpressure](../concepts/backpressure.md), [Overload Policy](../concepts/overload-policy.md) |
| Code | [`patterns/loadshedding`](https://github.com/johnknl/go-concurrency-patterns/tree/main/patterns/loadshedding) |

</div>
<!-- /meta -->

## Problem

When offered load exceeds capacity, a queue allowed to grow moves the problem instead of solving it: every item
waits longer, and by the time old items are served their results may no longer be wanted. Dropping some items keeps
the wait of the rest short: when the source cannot wait, something is shed
([backpressure](../concepts/backpressure.md#cannot-wait)). Dropping newest or oldest are two of the
[overload options](../concepts/overload-policy.md#options).

## Mechanism

A send inside a `select` with a `default` case never blocks: if the queue has no room, the item is dropped and the
drop counted, so the loss is visible. To drop the oldest instead, take the head to free a slot and retry. The
`default` in that eviction `select` is essential: if a receiver emptied the queue between the two steps, there is
nothing to take and the producer would block. With several senders, another one can take the freed slot, so one
call may evict more than one item while reporting a single drop; keep one sender per queue when the count must be
exact. `dropOldest` needs a buffered queue; on an unbuffered one it spins until a receiver arrives.

<!-- code: patterns/loadshedding#dropNewest patterns/loadshedding#dropOldest patterns/loadshedding#drain patterns/loadshedding#Example_dropNewest:main patterns/loadshedding#Example_dropOldest:main -->
```go
func dropNewest(queue chan int, v int) bool {
	select {
	case queue <- v:
		return true
	default:
		return false
	}
}

func dropOldest(queue chan int, v int) (dropped bool) {
	for {
		select {
		case queue <- v:
			return dropped
		default:
		}

		select {
		case <-queue:
			dropped = true
		default:
		}
	}
}

func drain(queue chan int) []int {
	close(queue)

	var got []int
	for v := range queue {
		got = append(got, v)
	}

	return got
}

func main() {
	queue := make(chan int, 3)
	dropped := 0

	for v := range 6 {
		if !dropNewest(queue, v) {
			dropped++
		}
	}

	fmt.Println("kept:", drain(queue), "dropped:", dropped)

	// Output:
	// kept: [0 1 2] dropped: 3
}

func main() {
	queue := make(chan int, 3)
	dropped := 0

	for v := range 6 {
		if dropOldest(queue, v) {
			dropped++
		}
	}

	fmt.Println("kept:", drain(queue), "dropped:", dropped)

	// Output:
	// kept: [3 4 5] dropped: 3
}
```
<!-- /code -->

## Applicability

Drop oldest when only the latest value matters (positions, sensor readings, UI state); either form suits sampled or
approximate data (telemetry, logs) and requests the caller can retry. Items that must not be lost, such as commands,
payments, or state changes, need [backpressure](../concepts/backpressure.md#feedback) or a durable queue.

## Consequences

- Data is lost. Expose the drop count; it is the signal for monitoring or for rejecting work earlier.
- Drop newest keeps the items that have waited longest, which may be stale; drop oldest keeps the freshest, in order.
- The producer never blocks, but a queued item still waits up to capacity times service time.
- A fixed capacity sheds only when full, regardless of an item's importance.

## Related patterns

- [Bounded Queue](bounded-queue.md) provides the bounded buffer.
- [Timed Send](timed-send.md) waits for a limited time before giving up.

## Anti-patterns

<!-- antipatterns -->
- [Buffer as a Band-Aid](../anti-patterns/oversized-buffer.md): Hide a producer that outruns its consumer behind a huge channel buffer, so the failure comes later and larger.
- [Goroutine as a Buffer](../anti-patterns/goroutine-as-buffer.md): Park a goroutine on a channel send so that the caller never blocks, which makes every pending value cost a goroutine instead of a channel slot.
<!-- /antipatterns -->

## Sources

<!-- sources -->
- [Nygard (2018)](../sources.md#source-nygard-2018), Ch. 5, "Shed Load"
- [Cuervo (2016)](../sources.md#source-sre-overload)
<!-- /sources -->
