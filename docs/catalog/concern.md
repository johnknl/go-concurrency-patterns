# By concern

Which problem the pattern addresses. Patterns with a shared concern are often alternatives or companions. See the [other
axes](index.md).

<!-- catalog: concern level=2 -->
## Determinism { #concern-determinism }

Getting the same result from the same inputs, for replay, tests, and reproducing failures.

| Page | Type | Intent |
|---|---|---|
| [Determinism](../concepts/determinism.md) | Concept | A computation is deterministic when the same starting state and the same inputs, in the same order, always produce the same result. |
| [State Machine](../patterns/state-machine.md) | Pattern | Change a stateful object only in steps that are legal for its current state, so that the same events in the same order always lead to the same state. |
| [Synctest](../stdlib/synctest.md) | Standard library | Test concurrent and time-dependent code deterministically, with fake time and a way to wait until every goroutine is blocked. |

## Backpressure { #concern-backpressure }

Making a fast producer wait for a slow consumer.

| Page | Type | Intent |
|---|---|---|
| [Backpressure](../concepts/backpressure.md) | Concept | Let the consumer's remaining capacity limit the producer's rate, so a faster producer slows down instead of piling up work. |
| [Bounded Queue](../patterns/bounded-queue.md) | Pattern | Let a producer run ahead of its consumer by a fixed, chosen amount. |
| [Overload Policy](../concepts/overload-policy.md) | Concept | Decide deliberately what happens when a limit is reached, trading loss, latency, and memory. |
| [Pipeline](../patterns/pipeline.md) | Pattern | Process a stream in independent steps that run at the same time, each handing its output to the next. |
| [Spill Queue](../patterns/spill-queue.md) | Pattern | Absorb a backlog that briefly exceeds what memory should hold, without making the producer wait for the consumer and without dropping items. |
| [Worker Pool](../patterns/worker-pool.md) | Pattern | Process any number of jobs with a fixed amount of concurrency. |

## Bounded concurrency { #concern-bounded-concurrency }

Capping how many things run, or how many resources are used, at once.

When callers need only permission to proceed, a [Semaphore](../patterns/semaphore.md) bounds how many run at once. When a fixed number of distinct, expensive resources such as connections must be shared, a [Resource Pool](../patterns/resource-pool.md) hands them out and makes callers wait when all are in use. When the work itself should run on a fixed set of long-lived goroutines fed from a queue, use a [Worker Pool](../patterns/worker-pool.md). An [Object Cache](../patterns/object-cache.md) reuses temporary objects but bounds nothing.

| Page | Type | Intent |
|---|---|---|
| [Resource Pool](../patterns/resource-pool.md) | Pattern | Share a fixed number of expensive resources among many goroutines, and wait when all are in use. |
| [Semaphore](../patterns/semaphore.md) | Pattern | Limit how many operations use a resource at the same time. |
| [Worker Pool](../patterns/worker-pool.md) | Pattern | Process any number of jobs with a fixed amount of concurrency. |

## Cancellation { #concern-cancellation }

Stopping work that is no longer needed.

A [Deadline](../stdlib/deadline.md) bounds how long the caller waits for an operation, and it only signals the operation. [Cooperative cancellation](../concepts/cooperative-cancellation.md) is what makes a long operation notice the signal: it checks the context between units, stops within one unit, and leaves state consistent. [Context Propagation](../stdlib/context-propagation.md) carries the deadline and the cancellation from the top of a request down the call tree, so every step shares one bound. [Cancel on Error](../patterns/cancel-on-error.md) handles a group of concurrent tasks that build one result: the first failure cancels the rest.

| Page | Type | Intent |
|---|---|---|
| [Cancel on Error](../patterns/cancel-on-error.md) | Pattern | When one of several concurrent tasks fails, stop the others and report the error that caused it. |
| [Context Propagation](../stdlib/context-propagation.md) | Standard library | Stop all the work done for a request together, and give it the request's deadline and scoped values. |
| [Cooperative Cancellation](../concepts/cooperative-cancellation.md) | Concept | Stop work whose result is no longer needed by having the work itself check a signal and return, leaving state consistent. |
| [Deadline](../stdlib/deadline.md) | Standard library | Bound how long an operation may take. |
| [For-Select Loop](../patterns/for-select-loop.md) | Pattern | Keep a goroutine waiting on several events, or doing work, while it still notices a stop signal. |
| [Goroutine Leak Prevention](../concepts/goroutine-leak-prevention.md) | Concept | Give every goroutine an owner that is responsible for its end, and a way to end that does not depend on a peer that may have left. |
| [Or-Done](../patterns/or-done.md) | Pattern | Stop reading a channel you do not control when your own work is canceled. |

## Leak prevention { #concern-leak-prevention }

Making sure that every goroutine can end.

| Page | Type | Intent |
|---|---|---|
| [Channel Ownership](../concepts/channel-ownership.md) | Concept | Give every channel one owner, the single sender or the closer that coordinates several, so that a close comes after the last send and is never raced or repeated. |
| [Close with Cause](../patterns/close-with-cause.md) | Pattern | Tell the receivers of a closed channel why it was closed. |
| [Cooperative Cancellation](../concepts/cooperative-cancellation.md) | Concept | Stop work whose result is no longer needed by having the work itself check a signal and return, leaving state consistent. |
| [Goroutine Leak Prevention](../concepts/goroutine-leak-prevention.md) | Concept | Give every goroutine an owner that is responsible for its end, and a way to end that does not depend on a peer that may have left. |
| [Or-Done](../patterns/or-done.md) | Pattern | Stop reading a channel you do not control when your own work is canceled. |
| [Synctest](../stdlib/synctest.md) | Standard library | Test concurrent and time-dependent code deterministically, with fake time and a way to wait until every goroutine is blocked. |
| [Timers and Tickers](../concepts/timers.md) | Concept | The guarantees Go's timers and tickers give about when they fire, what their channels deliver, and when they are released. |

## Data race safety { #concern-data-race-safety }

Sharing or handing over data without races.

| Page | Type | Intent |
|---|---|---|
| [Active Object](../patterns/active-object.md) | Pattern | Run all operations on an object one at a time, in the order they arrive, while callers either continue or wait for the result. |
| [Channel Ownership](../concepts/channel-ownership.md) | Concept | Give every channel one owner, the single sender or the closer that coordinates several, so that a close comes after the last send and is never raced or repeated. |
| [Compare-and-Swap Loop](../patterns/compare-and-swap-loop.md) | Pattern | Update a single shared value from its old value without making other writers wait. |
| [Condition Variable](../stdlib/condition-variable.md) | Standard library | Let goroutines wait until a condition on shared state holds, and wake them when it may have changed. |
| [Confinement](../concepts/confinement.md) | Concept | Keep data reachable from only one goroutine at a time, so it needs no synchronization. |
| [Copy-on-Write Snapshot](../patterns/copy-on-write-snapshot.md) | Pattern | Let many readers see a consistent version of read-mostly data without waiting for writers or for each other. |
| [Lazy Initialization](../stdlib/lazy-initialization.md) | Standard library | Build a value on first use, exactly once, safely from many goroutines. |
| [Memory Visibility](../concepts/memory-visibility.md) | Concept | A write made in one goroutine is guaranteed visible in another only if a synchronization edge of the Go memory model orders the two. |
| [Monitor](../patterns/monitor.md) | Pattern | Make a type's state safe to use from many goroutines, with no access left unprotected. |
| [Online Attach](../patterns/online-attach.md) | Pattern | Let members join and leave a running component without stopping it or racing with its work. |
| [Snapshot via Owner](../patterns/snapshot-via-owner.md) | Pattern | Read a consistent copy of state that one goroutine owns, without sharing the state. |
| [State Machine](../patterns/state-machine.md) | Pattern | Change a stateful object only in steps that are legal for its current state, so that the same events in the same order always lead to the same state. |

## Error handling { #concern-error-handling }

Carrying, deciding on, and reporting errors from concurrent work.

| Page | Type | Intent |
|---|---|---|
| [Cancel on Error](../patterns/cancel-on-error.md) | Pattern | When one of several concurrent tasks fails, stop the others and report the error that caused it. |
| [Close with Cause](../patterns/close-with-cause.md) | Pattern | Tell the receivers of a closed channel why it was closed. |
| [Future](../patterns/future.md) | Pattern | Overlap independent operations and collect each result when it is needed. |
| [Result with Error](../patterns/result-with-error.md) | Pattern | Get each error from concurrent work to the code that can decide what to do with it. |
| [Worker Pool](../patterns/worker-pool.md) | Pattern | Process any number of jobs with a fixed amount of concurrency. |

## Latency { #concern-latency }

Answering sooner, or giving up on a slow answer.

| Page | Type | Intent |
|---|---|---|
| [Batcher](../patterns/batcher.md) | Pattern | Spread a per-call cost over a stream of items while bounding how long any item waits. |
| [Bounded Queue](../patterns/bounded-queue.md) | Pattern | Let a producer run ahead of its consumer by a fixed, chosen amount. |
| [Cooperative Cancellation](../concepts/cooperative-cancellation.md) | Concept | Stop work whose result is no longer needed by having the work itself check a signal and return, leaving state consistent. |
| [Deadline](../stdlib/deadline.md) | Standard library | Bound how long an operation may take. |
| [Fan-Out/Fan-In](../patterns/fan-out-fan-in.md) | Pattern | Process independent items at the same time and collect all their results. |
| [Future](../patterns/future.md) | Pattern | Overlap independent operations and collect each result when it is needed. |
| [Group Commit](../patterns/group-commit.md) | Pattern | Let concurrent callers share one execution of an expensive operation, without adding latency when only one caller is waiting. |
| [Load Shedding](../patterns/load-shedding.md) | Pattern | Keep the producer and the latency bounded under overload by dropping the work that does not fit. |
| [Ordering](../concepts/ordering.md) | Concept | Which orders between values a concurrent design preserves, which it loses, and what restoring a lost order costs. |
| [Overload Policy](../concepts/overload-policy.md) | Concept | Decide deliberately what happens when a limit is reached, trading loss, latency, and memory. |
| [Request Racing](../patterns/request-racing.md) | Pattern | Get the fastest answer from several equivalent replicas and stop the others. |
| [Timed Send](../patterns/timed-send.md) | Pattern | Hand a value to a consumer, but give up after a deadline and tell the caller whether it was delivered. |
| [Timers and Tickers](../concepts/timers.md) | Concept | The guarantees Go's timers and tickers give about when they fire, what their channels deliver, and when they are released. |
| [Watermark](../patterns/watermark.md) | Pattern | Act on events in the order of their timestamps when they arrive from several inputs at different speeds, without guessing how long to wait for the slower ones. |

## Throughput { #concern-throughput }

Doing more work in the same time.

| Page | Type | Intent |
|---|---|---|
| [Backpressure](../concepts/backpressure.md) | Concept | Let the consumer's remaining capacity limit the producer's rate, so a faster producer slows down instead of piling up work. |
| [Batcher](../patterns/batcher.md) | Pattern | Spread a per-call cost over a stream of items while bounding how long any item waits. |
| [Bounded Queue](../patterns/bounded-queue.md) | Pattern | Let a producer run ahead of its consumer by a fixed, chosen amount. |
| [Compare-and-Swap Loop](../patterns/compare-and-swap-loop.md) | Pattern | Update a single shared value from its old value without making other writers wait. |
| [Copy-on-Write Snapshot](../patterns/copy-on-write-snapshot.md) | Pattern | Let many readers see a consistent version of read-mostly data without waiting for writers or for each other. |
| [Fan-Out/Fan-In](../patterns/fan-out-fan-in.md) | Pattern | Process independent items at the same time and collect all their results. |
| [Group Commit](../patterns/group-commit.md) | Pattern | Let concurrent callers share one execution of an expensive operation, without adding latency when only one caller is waiting. |
| [Object Cache](../patterns/object-cache.md) | Pattern | Avoid allocating and garbage-collecting the same kind of temporary object over and over in a hot path. |
| [Ordering](../concepts/ordering.md) | Concept | Which orders between values a concurrent design preserves, which it loses, and what restoring a lost order costs. |
| [Pipeline](../patterns/pipeline.md) | Pattern | Process a stream in independent steps that run at the same time, each handing its output to the next. |

## Rate control { #concern-rate-control }

Limiting how often something happens over time.

To limit how often an operation starts, choose by what may happen in a short span and whether callers wait or are refused. A [Token Bucket](../patterns/token-bucket.md) enforces an average rate and lets a bounded burst through after idle time; callers can be refused or wait, and `golang.org/x/time/rate` is the production implementation. [Pacing](../patterns/pacing.md) spaces calls one interval apart with no burst; every caller waits its turn. To limit how many operations run at once instead of how often they start, use a [Semaphore](../patterns/semaphore.md).

| Page | Type | Intent |
|---|---|---|
| [Pacing](../patterns/pacing.md) | Pattern | Start calls at a fixed interval, one at a time, with no bursts. |
| [Timers and Tickers](../concepts/timers.md) | Concept | The guarantees Go's timers and tickers give about when they fire, what their channels deliver, and when they are released. |
| [Token Bucket](../patterns/token-bucket.md) | Pattern | Limit the average rate of an operation while allowing a bounded burst. |

## Liveness { #concern-liveness }

Knowing that a goroutine is alive, and restoring it when it is not.

| Page | Type | Intent |
|---|---|---|
| [Healing Goroutines](../patterns/healing-goroutines.md) | Pattern | Detect a stuck worker and restart it. |
| [Heartbeat](../patterns/heartbeat.md) | Pattern | Tell a slow worker from a stuck one, and know when work has started, without sleeping. |
| [Slow Consumer Eviction](../patterns/slow-consumer-eviction.md) | Pattern | Keep a consumer that keeps missing deliveries from holding back the others. |

## Ordering { #concern-ordering }

Restoring or deciding the order of values that arrive out of order.

| Page | Type | Intent |
|---|---|---|
| [Ordering](../concepts/ordering.md) | Concept | Which orders between values a concurrent design preserves, which it loses, and what restoring a lost order costs. |
| [Resequencer](../patterns/resequencer.md) | Pattern | Consume results in input order when concurrent work delivers them out of order. |
| [Spill Queue](../patterns/spill-queue.md) | Pattern | Absorb a backlog that briefly exceeds what memory should hold, without making the producer wait for the consumer and without dropping items. |
| [State Machine](../patterns/state-machine.md) | Pattern | Change a stateful object only in steps that are legal for its current state, so that the same events in the same order always lead to the same state. |
| [Watermark](../patterns/watermark.md) | Pattern | Act on events in the order of their timestamps when they arrive from several inputs at different speeds, without guessing how long to wait for the slower ones. |

## Batching { #concern-batching }

Combining many small operations into fewer large ones.

| Page | Type | Intent |
|---|---|---|
| [Batcher](../patterns/batcher.md) | Pattern | Spread a per-call cost over a stream of items while bounding how long any item waits. |
| [Group Commit](../patterns/group-commit.md) | Pattern | Let concurrent callers share one execution of an expensive operation, without adding latency when only one caller is waiting. |
| [Windowing](../patterns/windowing.md) | Pattern | Compute a count, sum, or other aggregate over each span of time of a stream, for fixed spans or for the most recent one. |

## Overload { #concern-overload }

Staying correct and responsive when work arrives faster than it can be handled.

| Page | Type | Intent |
|---|---|---|
| [Backpressure](../concepts/backpressure.md) | Concept | Let the consumer's remaining capacity limit the producer's rate, so a faster producer slows down instead of piling up work. |
| [Load Shedding](../patterns/load-shedding.md) | Pattern | Keep the producer and the latency bounded under overload by dropping the work that does not fit. |
| [Overload Policy](../concepts/overload-policy.md) | Concept | Decide deliberately what happens when a limit is reached, trading loss, latency, and memory. |
| [PubSub Hub](../patterns/pubsub-hub.md) | Pattern | Deliver every published message to each current subscriber, while subscribers come and go. |
| [Slow Consumer Eviction](../patterns/slow-consumer-eviction.md) | Pattern | Keep a consumer that keeps missing deliveries from holding back the others. |
| [Timed Send](../patterns/timed-send.md) | Pattern | Hand a value to a consumer, but give up after a deadline and tell the caller whether it was delivered. |

## Composition { #concern-composition }

Combining small concurrent parts into larger ones.

| Page | Type | Intent |
|---|---|---|
| [Active Object](../patterns/active-object.md) | Pattern | Run all operations on an object one at a time, in the order they arrive, while callers either continue or wait for the result. |
| [Context Propagation](../stdlib/context-propagation.md) | Standard library | Stop all the work done for a request together, and give it the request's deadline and scoped values. |
| [For-Select Loop](../patterns/for-select-loop.md) | Pattern | Keep a goroutine waiting on several events, or doing work, while it still notices a stop signal. |
| [Online Attach](../patterns/online-attach.md) | Pattern | Let members join and leave a running component without stopping it or racing with its work. |
| [Or-Done](../patterns/or-done.md) | Pattern | Stop reading a channel you do not control when your own work is canceled. |
| [Pipeline](../patterns/pipeline.md) | Pattern | Process a stream in independent steps that run at the same time, each handing its output to the next. |
| [PubSub Hub](../patterns/pubsub-hub.md) | Pattern | Deliver every published message to each current subscriber, while subscribers come and go. |
| [Replay Buffer](../patterns/replay-buffer.md) | Pattern | Let a late subscriber catch up on the last n messages before it receives live ones. |
| [Tee](../patterns/tee.md) | Pattern | Give each value of one stream to two consumers that process it independently. |
<!-- /catalog -->
