# By purpose

What the pattern is for. Every pattern has exactly one purpose, and the navigation follows it. See the [other
axes](index.md).

<!-- catalog: purpose level=2 -->
## Data flow { #purpose-data-flow }

How values move between goroutines.

| Pattern | Intent |
|---|---|
| [Batcher](../patterns/batcher.md) | Spread a per-call cost over a stream of items while bounding how long any item waits. |
| [Bounded Queue](../patterns/bounded-queue.md) | Let a producer run ahead of its consumer by a fixed, chosen amount. |
| [Fan-Out/Fan-In](../patterns/fan-out-fan-in.md) | Process independent items at the same time and collect all their results. |
| [Or-Done](../patterns/or-done.md) | Stop reading a channel you do not control when your own work is canceled. |
| [Pipeline](../patterns/pipeline.md) | Process a stream in independent steps that run at the same time, each handing its output to the next. |
| [PubSub Hub](../patterns/pubsub-hub.md) | Deliver every published message to each current subscriber, while subscribers come and go. |
| [Replay Buffer](../patterns/replay-buffer.md) | Let a late subscriber catch up on the last n messages before it receives live ones. |
| [Resequencer](../patterns/resequencer.md) | Consume results in input order when concurrent work delivers them out of order. |
| [Spill Queue](../patterns/spill-queue.md) | Absorb a backlog that briefly exceeds what memory should hold, without making the producer wait for the consumer and without dropping items. |
| [Tee](../patterns/tee.md) | Give each value of one stream to two consumers that process it independently. |
| [Watermark](../patterns/watermark.md) | Act on events in the order of their timestamps when they arrive from several inputs at different speeds, without guessing how long to wait for the slower ones. |
| [Windowing](../patterns/windowing.md) | Compute a count, sum, or other aggregate over each span of time of a stream, for fixed spans or for the most recent one. |

## Execution { #purpose-execution }

Who runs the work, and how the outcome comes back.

| Pattern | Intent |
|---|---|
| [Active Object](../patterns/active-object.md) | Run all operations on an object one at a time, in the order they arrive, while callers either continue or wait for the result. |
| [Future](../patterns/future.md) | Overlap independent operations and collect each result when it is needed. |
| [Group Commit](../patterns/group-commit.md) | Let concurrent callers share one execution of an expensive operation, without adding latency when only one caller is waiting. |
| [State Machine](../patterns/state-machine.md) | Change a stateful object only in steps that are legal for its current state, so that the same events in the same order always lead to the same state. |
| [Worker Pool](../patterns/worker-pool.md) | Process any number of jobs with a fixed amount of concurrency. |

## Coordination { #purpose-coordination }

How goroutines wait for, limit, or choose among each other, and how shared state is kept safe.

| Pattern | Intent |
|---|---|
| [Compare-and-Swap Loop](../patterns/compare-and-swap-loop.md) | Update a single shared value from its old value without making other writers wait. |
| [Copy-on-Write Snapshot](../patterns/copy-on-write-snapshot.md) | Let many readers see a consistent version of read-mostly data without waiting for writers or for each other. |
| [For-Select Loop](../patterns/for-select-loop.md) | Keep a goroutine waiting on several events, or doing work, while it still notices a stop signal. |
| [Monitor](../patterns/monitor.md) | Make a type's state safe to use from many goroutines, with no access left unprotected. |
| [Object Cache](../patterns/object-cache.md) | Avoid allocating and garbage-collecting the same kind of temporary object over and over in a hot path. |
| [Online Attach](../patterns/online-attach.md) | Let members join and leave a running component without stopping it or racing with its work. |
| [Pacing](../patterns/pacing.md) | Start calls at a fixed interval, one at a time, with no bursts. |
| [Resource Pool](../patterns/resource-pool.md) | Share a fixed number of expensive resources among many goroutines, and wait when all are in use. |
| [Semaphore](../patterns/semaphore.md) | Limit how many operations use a resource at the same time. |
| [Snapshot via Owner](../patterns/snapshot-via-owner.md) | Read a consistent copy of state that one goroutine owns, without sharing the state. |
| [Token Bucket](../patterns/token-bucket.md) | Limit the average rate of an operation while allowing a bounded burst. |

## Lifecycle { #purpose-lifecycle }

How goroutines are stopped, and kept from leaking.

| Pattern | Intent |
|---|---|
| [Close with Cause](../patterns/close-with-cause.md) | Tell the receivers of a closed channel why it was closed. |

## Resilience { #purpose-resilience }

How errors, slowness, and failure are handled to keep a system working.

| Pattern | Intent |
|---|---|
| [Cancel on Error](../patterns/cancel-on-error.md) | When one of several concurrent tasks fails, stop the others and report the error that caused it. |
| [Healing Goroutines](../patterns/healing-goroutines.md) | Detect a stuck worker and restart it. |
| [Heartbeat](../patterns/heartbeat.md) | Tell a slow worker from a stuck one, and know when work has started, without sleeping. |
| [Load Shedding](../patterns/load-shedding.md) | Keep the producer and the latency bounded under overload by dropping the work that does not fit. |
| [Request Racing](../patterns/request-racing.md) | Get the fastest answer from several equivalent replicas and stop the others. |
| [Result with Error](../patterns/result-with-error.md) | Get each error from concurrent work to the code that can decide what to do with it. |
| [Slow Consumer Eviction](../patterns/slow-consumer-eviction.md) | Keep a consumer that keeps missing deliveries from holding back the others. |
| [Timed Send](../patterns/timed-send.md) | Hand a value to a consumer, but give up after a deadline and tell the caller whether it was delivered. |
<!-- /catalog -->
