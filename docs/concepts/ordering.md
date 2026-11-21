---
title: Ordering
aliases: ['Message ordering', 'FIFO order']
intent: Which orders between values a concurrent design preserves, which it loses, and what restoring a lost order costs.
concerns: [ordering, latency, throughput]
sources:
  - id: hohpe-woolf-2003
  - id: go-memory-model
---
# Ordering

<!-- meta -->
Which orders between values a concurrent design preserves, which it loses, and what restoring a lost order costs.

<div class="cp-props" markdown>

| | |
|---|---|
| Also known as | Message ordering, FIFO order |
| Concerns | [Ordering](../catalog/concern.md#concern-ordering), [Latency](../catalog/concern.md#concern-latency), [Throughput](../catalog/concern.md#concern-throughput) |

</div>
<!-- /meta -->

## Why it matters

Concurrency trades order for throughput. Whenever values travel through more than one goroutine at once, the order
in which they arrive differs from the order in which they were produced. That is harmless for independent work and
wrong for updates to the same thing, log entries, or responses matched to requests. A design has to state which
order it promises and pay for it deliberately.

## Properties

### One goroutine preserves FIFO order { #sequential-fifo }

A single channel delivers the values of one sender in the order they were sent. A single goroutine, or a stage that
handles one value at a time, emits results in the order it received inputs. A chain of such stages preserves order
end to end.

### Concurrent senders have no order between them { #concurrent-senders }

When several goroutines send on one channel, each sender's values stay in order relative to each other, but the
interleaving between senders is arbitrary and changes from run to run. "Sent first" is not even well defined without
synchronization between the senders.

### Fan-out loses order { #fan-out-loses }

Distributing values over several workers and merging their results reorders them: a short task overtakes a long one.
The output order depends on processing times and scheduling, not on input order.

### Restoring order costs head-of-line blocking and a buffer { #restore-cost }

To emit results in input order after fan-out, number the inputs and hold every result that arrives early until all
earlier ones have arrived ([Resequencer](../patterns/resequencer.md)). One slow item then delays all results after
it (head-of-line blocking), and the held results need a buffer that must be bounded, or a slow item lets it grow
without limit. Latency of the ordered output is that of the slowest item in the window.

### Per-key order is often enough { #per-key }

Usually only values about the same thing need order: events of one account, updates to one key. Routing all values
with the same key to the same worker (by hashing the key) keeps per-key order while different keys run in parallel,
without a resequencer. Global order is needed only when the consumer compares values across keys.

## Used by

<!-- usedby -->
- [Active Object](../patterns/active-object.md): the owner runs commands in one total order.
- [Fan-Out/Fan-In](../patterns/fan-out-fan-in.md): results arrive in completion order; per-input order survives a merge.
- [Pipeline](../patterns/pipeline.md): each stage is sequential, so order is kept.
- [PubSub Hub](../patterns/pubsub-hub.md): order holds per publisher, not between concurrent publishers.
- [Resequencer](../patterns/resequencer.md): restoring order costs head-of-line blocking and a bounded buffer; keys come from the source.
- [Spill Queue](../patterns/spill-queue.md): one queue keeps FIFO order across both tiers.
- [Watermark](../patterns/watermark.md): merged inputs have no order; restoring it costs waiting and a bounded buffer.
<!-- /usedby -->

## Violated by

<!-- violatedby -->
- [Premature Parallelism](../anti-patterns/premature-parallelism.md): fanning out loses the input order.
- [Unbounded Reorder Buffer](../anti-patterns/unbounded-reorder-buffer.md): restoring order needs a bound on the reorder buffer.
<!-- /violatedby -->

## Related concepts

- [Backpressure](backpressure.md): a resequencer's buffer needs a bound like any other.
- [Confinement](confinement.md#owner-goroutine): an owner goroutine serializes requests in arrival order.

## Sources

<!-- sources -->
- [Hohpe and Woolf (2003)](../sources.md#source-hohpe-woolf-2003)
- [The Go Authors (2022)](../sources.md#source-go-memory-model)
<!-- /sources -->
