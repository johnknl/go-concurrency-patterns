# By scale

How big the pattern is. Idioms are the vocabulary, patterns are built from a few idioms, and composites are
components made by combining patterns. A composite page lists what it is composed of. See the
[other axes](index.md).

<!-- catalog: scale level=2 -->
## Idiom { #scale-idiom }

One or two mechanisms arranged in a few lines, often a single function.

| Pattern | Intent |
|---|---|
| [Bounded Queue](../patterns/bounded-queue.md) | Let a producer run ahead of its consumer by a fixed, chosen amount. |
| [Close with Cause](../patterns/close-with-cause.md) | Tell the receivers of a closed channel why it was closed. |
| [Compare-and-Swap Loop](../patterns/compare-and-swap-loop.md) | Update a single shared value from its old value without making other writers wait. |
| [For-Select Loop](../patterns/for-select-loop.md) | Keep a goroutine waiting on several events, or doing work, while it still notices a stop signal. |
| [Future](../patterns/future.md) | Overlap independent operations and collect each result when it is needed. |
| [Heartbeat](../patterns/heartbeat.md) | Tell a slow worker from a stuck one, and know when work has started, without sleeping. |
| [Load Shedding](../patterns/load-shedding.md) | Keep the producer and the latency bounded under overload by dropping the work that does not fit. |
| [Object Cache](../patterns/object-cache.md) | Avoid allocating and garbage-collecting the same kind of temporary object over and over in a hot path. |
| [Or-Done](../patterns/or-done.md) | Stop reading a channel you do not control when your own work is canceled. |
| [Pacing](../patterns/pacing.md) | Start calls at a fixed interval, one at a time, with no bursts. |
| [Replay Buffer](../patterns/replay-buffer.md) | Let a late subscriber catch up on the last n messages before it receives live ones. |
| [Resource Pool](../patterns/resource-pool.md) | Share a fixed number of expensive resources among many goroutines, and wait when all are in use. |
| [Result with Error](../patterns/result-with-error.md) | Get each error from concurrent work to the code that can decide what to do with it. |
| [Semaphore](../patterns/semaphore.md) | Limit how many operations use a resource at the same time. |
| [Slow Consumer Eviction](../patterns/slow-consumer-eviction.md) | Keep a consumer that keeps missing deliveries from holding back the others. |
| [Snapshot via Owner](../patterns/snapshot-via-owner.md) | Read a consistent copy of state that one goroutine owns, without sharing the state. |
| [Tee](../patterns/tee.md) | Give each value of one stream to two consumers that process it independently. |
| [Timed Send](../patterns/timed-send.md) | Hand a value to a consumer, but give up after a deadline and tell the caller whether it was delivered. |
| [Windowing](../patterns/windowing.md) | Compute a count, sum, or other aggregate over each span of time of a stream, for fixed spans or for the most recent one. |

## Pattern { #scale-pattern }

Several cooperating goroutines or parts with distinct roles.

| Pattern | Intent |
|---|---|
| [Batcher](../patterns/batcher.md) | Spread a per-call cost over a stream of items while bounding how long any item waits. |
| [Cancel on Error](../patterns/cancel-on-error.md) | When one of several concurrent tasks fails, stop the others and report the error that caused it. |
| [Copy-on-Write Snapshot](../patterns/copy-on-write-snapshot.md) | Let many readers see a consistent version of read-mostly data without waiting for writers or for each other. |
| [Fan-Out/Fan-In](../patterns/fan-out-fan-in.md) | Process independent items at the same time and collect all their results. |
| [Group Commit](../patterns/group-commit.md) | Let concurrent callers share one execution of an expensive operation, without adding latency when only one caller is waiting. |
| [Healing Goroutines](../patterns/healing-goroutines.md) | Detect a stuck worker and restart it. |
| [Monitor](../patterns/monitor.md) | Make a type's state safe to use from many goroutines, with no access left unprotected. |
| [Online Attach](../patterns/online-attach.md) | Let members join and leave a running component without stopping it or racing with its work. |
| [Pipeline](../patterns/pipeline.md) | Process a stream in independent steps that run at the same time, each handing its output to the next. |
| [Request Racing](../patterns/request-racing.md) | Get the fastest answer from several equivalent replicas and stop the others. |
| [Resequencer](../patterns/resequencer.md) | Consume results in input order when concurrent work delivers them out of order. |
| [Spill Queue](../patterns/spill-queue.md) | Absorb a backlog that briefly exceeds what memory should hold, without blocking the producer and without dropping items. |
| [State Machine](../patterns/state-machine.md) | Change a stateful object only in steps that are legal for its current state, so that the same events in the same order always lead to the same state. |
| [Token Bucket](../patterns/token-bucket.md) | Limit the average rate of an operation while allowing a bounded burst. |
| [Watermark](../patterns/watermark.md) | Act on events in the order of their timestamps when they arrive from several inputs at different speeds, without guessing how long to wait for the slower ones. |
| [Worker Pool](../patterns/worker-pool.md) | Process any number of jobs with a fixed amount of concurrency. |

## Composite { #scale-composite }

A component built by combining other patterns of this catalog.

| Pattern | Intent |
|---|---|
| [Active Object](../patterns/active-object.md) | Run all operations on an object one at a time, in the order they arrive, while callers either continue or wait for the result. |
| [PubSub Hub](../patterns/pubsub-hub.md) | Deliver every published message to each current subscriber, while subscribers come and go. |
<!-- /catalog -->
