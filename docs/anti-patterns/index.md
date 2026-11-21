# Anti-patterns

An anti-pattern is a solution that violates one of the [principles](../index.md#principles) where a solution
that follows them exists, whether it fails yet or not. Each page names the principle it violates, describes the
symptom, shows the faulty code next to the symptom it produces, explains why it goes wrong, how to detect it, and
which patterns fix it. The faulty code is compiled and tested like the rest, but it is only run in a controlled
setting that shows the symptom and cleans up after itself.

The anti-patterns below are grouped by the principle they violate.

<!-- catalog: anti-pattern level=2 -->
## Clear ownership

Every goroutine, channel and mutable value has one owner that decides its lifetime and alone closes or mutates it. See the [principle](../index.md#principle-ownership).

| Anti-pattern | What goes wrong | Fixed by |
|---|---|---|
| [Closing from the Wrong Side](wrong-side-close.md) | A channel is closed by a goroutine that does not own it, so a later send or a second close panics. | [Close with Cause](../patterns/close-with-cause.md) |
| [Copied Mutex](copied-mutex.md) | Copy a value that contains a mutex, so the copy has its own lock state and code that locks it does not exclude code that locks the original. | [Monitor](../patterns/monitor.md) |
| [Fire-and-Forget Goroutine](fire-and-forget-goroutine.md) | Start a goroutine and return, so nobody can wait for it, learn of its error or stop it. | [Result with Error](../patterns/result-with-error.md) |
| [Ignored Context](ignored-context.md) | A function takes a context and never checks it, so cancellation and deadlines have no effect. | [Context Propagation](../stdlib/context-propagation.md), [Deadline](../stdlib/deadline.md) |
| [Pointer over a Channel](pointer-over-channel.md) | A pointer, slice, or map is sent over a channel and then used by both sides, so the data is shared although it was handed over. | [Snapshot via Owner](../patterns/snapshot-via-owner.md), [Copy-on-Write Snapshot](../patterns/copy-on-write-snapshot.md) |

## Synchronous first

A function does its work in the caller's goroutine and returns; concurrency is the caller's choice and needs a reason. See the [principle](../index.md#principle-synchronous-first).

| Anti-pattern | What goes wrong | Fixed by |
|---|---|---|
| [Channel Generator](channel-generator.md) | Produce a sequence in a goroutine of its own and hand it to the consumer through a channel, where a callback or an iterator would produce it in the consumer's goroutine. | [Pipeline](../patterns/pipeline.md) |
| [Needless Asynchrony](needless-asynchrony.md) | Start a goroutine inside a function and hand back a channel, where returning the result would do, so the caller loses the error path, the cancellation, and control over the goroutine. | [Future](../patterns/future.md) |

## The simplest mechanism that works

Use the lowest rung that solves the problem; climb only when the one below cannot. See the [principle](../index.md#principle-simplest-mechanism).

| Anti-pattern | What goes wrong | Fixed by |
|---|---|---|
| [Channel Where a Mutex Would Do](channel-instead-of-mutex.md) | Guard plain state with an owner goroutine and request and reply channels, where a mutex or an atomic next to the state would do, so every call pays a round trip and the state gains a goroutine whose lifetime someone has to manage. | [Monitor](../patterns/monitor.md), [Compare-and-Swap Loop](../patterns/compare-and-swap-loop.md) |
| [Premature Parallelism](premature-parallelism.md) | Spread cheap work on data that is already in hand across goroutines and channels, so the handoffs cost more than the work, the order is lost, and a loop's worth of code becomes a pipeline to manage. | [Worker Pool](../patterns/worker-pool.md), [Fan-Out/Fan-In](../patterns/fan-out-fan-in.md) |

## Synchronize explicitly

Ordering and exclusion come from channels, locks, atomics or wait groups, never from timing; locks are short and ordered. See the [principle](../index.md#principle-explicit-synchronization).

| Anti-pattern | What goes wrong | Fixed by |
|---|---|---|
| [Double-Checked Locking](double-checked-locking.md) | Test a lazily initialized pointer without synchronization and lock only when it is nil, so readers race with the writer and can see a pointer to an incompletely initialized value. | [Lazy Initialization](../stdlib/lazy-initialization.md), [Compare-and-Swap Loop](../patterns/compare-and-swap-loop.md) |
| [Implicit State Machine](implicit-state-machine.md) | Keep a lifecycle in several boolean flags under a mutex, so combinations that should be impossible are reachable and every method has to re-check them. | [State Machine](../patterns/state-machine.md) |
| [Inconsistent Lock Order](inconsistent-lock-order.md) | Take two locks in different orders in different goroutines, so each can hold one while waiting for the other, and both wait forever. | [Monitor](../patterns/monitor.md), [Active Object](../patterns/active-object.md) |
| [Lock Held While Blocking](lock-held-while-blocking.md) | Hold a mutex across a channel operation, I/O or a callback, so everyone else who needs the lock waits, and a deadlock follows if the other side needs it too. | [Active Object](../patterns/active-object.md), [Snapshot via Owner](../patterns/snapshot-via-owner.md) |
| [Sleeping to Synchronize](sleep-synchronization.md) | Use time.Sleep or a guessed delay to wait for another goroutine instead of waiting for its completion. | [Future](../patterns/future.md), [Heartbeat](../patterns/heartbeat.md), [Watermark](../patterns/watermark.md) |

## Bound everything

Goroutines, queues, retries and memory have limits, and the code decides what happens at the limit. See the [principle](../index.md#principle-bounded).

| Anti-pattern | What goes wrong | Fixed by |
|---|---|---|
| [Batch Without a Time Bound](batch-without-time-bound.md) | Collect items into batches that are flushed only when full, so at low load an item waits indefinitely and the last partial batch is lost at shutdown. | [Batcher](../patterns/batcher.md), [Group Commit](../patterns/group-commit.md) |
| [Buffer as a Band-Aid](oversized-buffer.md) | Hide a producer that outruns its consumer behind a huge channel buffer, so the failure comes later and larger. | [Bounded Queue](../patterns/bounded-queue.md), [Load Shedding](../patterns/load-shedding.md), [Spill Queue](../patterns/spill-queue.md) |
| [Goroutine as a Buffer](goroutine-as-buffer.md) | Park a goroutine on a channel send so that the caller never blocks, which makes every pending value cost a goroutine instead of a channel slot. | [Bounded Queue](../patterns/bounded-queue.md), [Load Shedding](../patterns/load-shedding.md), [Spill Queue](../patterns/spill-queue.md) |
| [Unbounded Goroutines](unbounded-goroutines.md) | Start one goroutine per input item, so concurrency grows with the input instead of with capacity. | [Worker Pool](../patterns/worker-pool.md), [Semaphore](../patterns/semaphore.md) |
| [Unbounded Reorder Buffer](unbounded-reorder-buffer.md) | Hold everything that arrives behind a missing item until it arrives, with no limit, so one slow or lost item stops all output while the buffer grows. | [Resequencer](../patterns/resequencer.md), [Watermark](../patterns/watermark.md) |

## Plain code over generality

An abstraction or dependency earns its place; misuse is handled by documented contracts. See the [principle](../index.md#principle-plain-code).

| Anti-pattern | What goes wrong | Fixed by |
|---|---|---|
| [Premature Abstraction](premature-abstraction.md) | Extract a generic, configurable concurrency helper before its uses are known, so each caller sets flags for the behavior it needs, the flags interact, and every caller depends on code paths it does not use. | [Worker Pool](../patterns/worker-pool.md), [Future](../patterns/future.md) |

## Prefer explicit inputs

Prefer behavior that follows from explicit inputs (state, events, and injected clocks and random sources), where the same inputs must give the same result. See the [principle](../index.md#principle-explicit-inputs).

| Anti-pattern | What goes wrong | Fixed by |
|---|---|---|
| [Non-Deterministic State Machine](non-deterministic-state-machine.md) | A transition whose outcome is not part of its input, so the same state and event can lead to different states. | [State Machine](../patterns/state-machine.md) |
<!-- /catalog -->
