---
title: Buffer as a Band-Aid
intent: Hide a producer that outruns its consumer behind a huge channel buffer, so the failure comes later and larger.
purpose: data-flow
package: antipatterns/oversizedbuffer
mechanisms: [channels]
concerns: [backpressure, overload, latency]
remedies: [bounded-queue, load-shedding, spill-queue]
violates: [bounded]
concepts:
  - backpressure: 'a buffer absorbs bursts, not a sustained rate difference.'
  - overload-policy: 'a buffer only delays the overload decision.'
sources:
  - id: sre-overload
---
# Buffer as a Band-Aid

<!-- meta -->
Hide a producer that outruns its consumer behind a huge channel buffer, so the failure comes later and larger.

<div class="cp-props" markdown>

| | |
|---|---|
| Violates | [Bound everything](../index.md#principle-bounded) |
| Purpose | [Data flow](../catalog/purpose.md#purpose-data-flow) |
| Mechanisms | [Channels](../catalog/mechanism.md#mechanism-channels) |
| Concerns | [Backpressure](../catalog/concern.md#concern-backpressure), [Overload](../catalog/concern.md#concern-overload), [Latency](../catalog/concern.md#concern-latency) |
| Breaks | [Backpressure](../concepts/backpressure.md), [Overload Policy](../concepts/overload-policy.md) |
| Code | [`antipatterns/oversizedbuffer`](https://github.com/johnknl/go-concurrency-patterns/tree/main/antipatterns/oversizedbuffer) |

</div>
<!-- /meta -->

## Symptoms

- A blocked producer or a deadlocked send was "fixed" by enlarging the channel buffer; the rate difference between
  producer and consumer is unchanged.
- The producer never blocks and looks healthy, while memory grows: `make` allocates the full buffer up front, and
  what queued items point to grows with the backlog.
- Latency rises: an item with `n` items ahead of it waits roughly `n` divided by the consumer's rate.
- Eventually the program blocks or is killed at a moment hard to relate to the change that caused it.

## Example

=== "Faulty"

    `badProduce` queues everything in a channel sized for it. The producer finishes with nothing consumed, `len`
    equals `cap`, and the last item waits for the whole backlog.

    <!-- code: antipatterns/oversizedbuffer#badProduce antipatterns/oversizedbuffer#Example_oversizedBuffer:main -->

    ```go
    func badProduce(n int) chan int {
    	queue := make(chan int, n)
    	for i := range n {
    		queue <- i
    	}
    	return queue
    }

    func main() {
    	queue := badProduce(1000)

    	// The producer has finished and nothing has been consumed: the queue holds everything.
    	fmt.Println("queued:", len(queue), "of", cap(queue))

    	// The first item received waits behind the whole backlog; the newest item had 999 items ahead of it.
    	first := <-queue
    	fmt.Println("first item:", first, "items queued behind it:", len(queue))
    	// Output:
    	// queued: 1000 of 1000
    	// first item: 0 items queued behind it: 999
    }
    ```

    <!-- /code -->

=== "Fixed"

    Size the buffer for a burst and decide what happens when it is full: the producer waits (a plain send,
    backpressure) or the item is dropped (`select` with `default`, load shedding). `shed` drops and counts; which
    is right depends on whether losing items is acceptable.

    <!-- code: antipatterns/oversizedbuffer#shed antipatterns/oversizedbuffer#Example_boundedQueue:main -->

    ```go
    func shed(n int, queue chan<- int) (accepted, dropped int) {
    	for i := range n {
    		select {
    		case queue <- i:
    			accepted++
    		default:
    			dropped++
    		}
    	}
    	return accepted, dropped
    }

    func main() {
    	queue := make(chan int, 10)
    	accepted, dropped := shed(1000, queue)
    	fmt.Println("accepted:", accepted, "dropped:", dropped, "queued:", len(queue), "of", cap(queue))
    	// Output:
    	// accepted: 10 dropped: 990 queued: 10 of 10
    }
    ```

    <!-- /code -->

## Why it goes wrong

A buffer absorbs a burst the consumer later catches up on; throughput stays that of the
[slowest stage](../concepts/backpressure.md#slowest-stage). Any finite buffer fills and an unbounded one grows until
memory runs out; a larger buffer only [moves that point](../concepts/overload-policy.md#buffer-delays). Because
a channel is first in, first out, the buffer turns overload into latency, and work whose result is no longer needed
is still done. The blocking send it removed was the signal that the consumer could not keep up.

## Detection

- `len(ch)` near `cap(ch)`, or growing without bound, in metrics (each read is only a snapshot).
- The age of items when dequeued, not only the queue length.
- Large buffer sizes with no derivation, or ones raised to make a hang disappear.
- A heap profile attributes the buffer to its `make`, and queued data to the code that allocated it.

## Fixed by

<!-- remedies -->
- [Bounded Queue](../patterns/bounded-queue.md): Let a producer run ahead of its consumer by a fixed, chosen amount.
- [Load Shedding](../patterns/load-shedding.md): Keep the producer and the latency bounded under overload by dropping the work that does not fit.
- [Spill Queue](../patterns/spill-queue.md): Absorb a backlog that briefly exceeds what memory should hold, without making the producer wait for the consumer and without dropping items.
<!-- /remedies -->

## Sources

<!-- sources -->
- [Cuervo (2016)](../sources.md#source-sre-overload)
<!-- /sources -->
