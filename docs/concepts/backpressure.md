---
title: Backpressure
aliases: ['Flow control']
intent: Let the consumer's remaining capacity limit the producer's rate, so a faster producer slows down instead of piling up work.
concerns: [backpressure, overload, throughput]
sources:
  - id: reactive-streams
  - id: nygard-2018
    at: 'Ch. 5, "Create Back Pressure"'
---
# Backpressure

<!-- meta -->
Let the consumer's remaining capacity limit the producer's rate, so a faster producer slows down instead of piling up work.

<div class="cp-props" markdown>

| | |
|---|---|
| Also known as | Flow control |
| Concerns | [Backpressure](../catalog/concern.md#concern-backpressure), [Overload](../catalog/concern.md#concern-overload), [Throughput](../catalog/concern.md#concern-throughput) |

</div>
<!-- /meta -->

## Why it matters

A producer faster than its consumer creates a difference that must go somewhere. Without feedback it accumulates:
in a queue, in goroutines, in memory, and as latency, until the process fails. Backpressure turns the consumer's
limit into a signal the producer obeys, so the system runs at the rate it can sustain and its memory stays bounded.

## Properties

### The consumer's capacity limits the producer { #feedback }

Backpressure is feedback: the producer can go ahead only as far as the consumer has room. In Go a channel send gives
it without further code: it blocks while the buffer is full, or until a receiver is ready on an unbuffered channel.
The producer's lead over the consumer is at most the channel's capacity.

### It must reach the source { #reaches-source }

Each stage that waits for its consumer stops reading its own input, which makes its producer wait, and so on. The
feedback is useful only if it propagates all the way to the source, the place that can actually slow down (stop
reading a socket, stop accepting requests). A stage that decouples itself from the stage after it, with an unbounded
queue or a goroutine per value, cuts the chain, and the excess accumulates there.

### It needs a bound { #needs-bound }

Feedback exists only where something can be full. Every queue, buffer, and set of in-flight goroutines between source
and consumer needs a fixed limit. A buffer that grows on demand, or a goroutine parked per pending value, never says
"full", so the producer never waits; it only moves the problem into memory.

### Throughput is the slowest stage's; a buffer absorbs bursts { #slowest-stage }

In steady state a chain moves at the rate of its slowest stage, whatever its buffers. A buffer helps only with
variation: it lets a fast stage keep working through a short stall downstream, or absorb a burst upstream. A buffer
larger than the bursts it must absorb adds latency (items wait in it) and hides overload until it is full, without
raising throughput.

### When the source cannot wait, something is shed { #cannot-wait }

Backpressure assumes the producer can wait. A source that cannot, such as live events, a sensor, or clients that
will not slow down, still produces at its own rate, and when the bound is reached something must give: a value is
dropped, a request is rejected, a consumer is evicted. Choosing what is an [Overload Policy](overload-policy.md).

## Used by

<!-- usedby -->
- [Batcher](../patterns/batcher.md): a full input makes producers wait.
- [Bounded Queue](../patterns/bounded-queue.md): a full buffer blocks the producer; without a bound there is no feedback.
- [Load Shedding](../patterns/load-shedding.md): it is the choice when the source cannot wait; items that must not be lost need backpressure.
- [Pipeline](../patterns/pipeline.md): unbuffered channels carry a slow stage back to the source.
- [Resequencer](../patterns/resequencer.md): the window makes the source wait; throughput stays that of the slowest item.
- [Slow Consumer Eviction](../patterns/slow-consumer-eviction.md): the producer cannot wait, so a stuck consumer is shed.
- [Spill Queue](../patterns/spill-queue.md): the disk widens the buffer; it does not replace backpressure.
- [Timed Send](../patterns/timed-send.md): it is for a source that cannot wait; waiting without bound is backpressure.
- [Worker Pool](../patterns/worker-pool.md): an unbuffered jobs channel makes the submitter wait for a free worker.
<!-- /usedby -->

## Violated by

<!-- violatedby -->
- [Buffer as a Band-Aid](../anti-patterns/oversized-buffer.md): a buffer absorbs bursts, not a sustained rate difference.
- [Goroutine as a Buffer](../anti-patterns/goroutine-as-buffer.md): parked goroutines have no bound, so no feedback reaches the producer.
- [Unbounded Goroutines](../anti-patterns/unbounded-goroutines.md): with no bound, no feedback reaches the producer.
<!-- /violatedby -->

## Related concepts

- [Overload Policy](overload-policy.md): waiting is one of the policies; the others apply when waiting is not allowed.
- [Goroutine Leak Prevention](goroutine-leak-prevention.md#select-on-done): a blocked send must still be cancelable,
  or a consumer that left turns backpressure into a leak.

## Sources

<!-- sources -->
- [Reactive Streams (n.d.)](../sources.md#source-reactive-streams)
- [Nygard (2018)](../sources.md#source-nygard-2018), Ch. 5, "Create Back Pressure"
<!-- /sources -->
