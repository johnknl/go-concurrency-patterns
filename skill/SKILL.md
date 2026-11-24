---
name: Go Concurrency Patterns
description: "Use when writing, reviewing, or designing Go code that uses goroutines, channels, sync, context, timers, or atomics: choosing a concurrency pattern, spotting anti-patterns (goroutine leaks, sleeping to synchronize, closing from the wrong side, unbounded goroutines, ignored contexts), and structuring owner goroutines, pipelines, worker pools, rate limits, queues, and shutdown."
---

# Go Concurrency Patterns

A catalog of Go concurrency patterns, concepts, standard library guides, and anti-patterns. The code in the
reference files is tested with the race detector and uses only the standard library.

## How to use this skill

- The principles below are this catalog's opinions about good Go concurrency, not rules of the language. Apply
  them with judgment; principle 3 (the simplest mechanism that works) usually wins a conflict.
- Do not read the whole catalog. Use the indexes below to find the page that fits the problem, then read that
  file in `references/`. Each page has the problem, the mechanism with code, when it fits, and what it costs.
- When reviewing code, go through the anti-pattern checklist and name the page for each finding.
- Verify concurrent code with `go test -race`, and test timing with `testing/synctest`; see
  `references/stdlib/synctest.md`.

## Principles

Seven principles decide what is a pattern here and what is an anti-pattern. They follow
Go's own guidance, cited with each, but they are this catalog's principles, not rules of the language.

### 1. Clear ownership

Every goroutine, channel, and mutable value has one owner, which ends and waits for its goroutines, receives their
results and errors, and alone closes the channel or writes the value. *Ask: who stops it, who waits for it, who
closes it, who may write it?* ([Goroutine Lifetimes](https://go.dev/wiki/CodeReviewComments#goroutine-lifetimes))

### 2. Synchronous first

A function does its work in the caller's goroutine and returns; concurrency is the caller's choice and needs a
reason. Sequences are callbacks, iterators, or `Next` methods. *Ask: does anything have to run while the caller
runs?* ([Synchronous Functions](https://go.dev/wiki/CodeReviewComments#synchronous-functions))

### 3. The simplest mechanism that works

Go up this list only as far as needed: a plain function; `go` with a `sync.WaitGroup`
([Fan-Out/Fan-In](references/patterns/fan-out-fan-in.md)); a mutex next to its state ([Monitor](references/patterns/monitor.md)); a
channel that hands off data or work ([Pipeline](references/patterns/pipeline.md), [Worker Pool](references/patterns/worker-pool.md)); a
goroutine that owns the state ([Active Object](references/patterns/active-object.md)); a composition
([PubSub Hub](references/patterns/pubsub-hub.md)). *Ask: would the step below work?*
([Use a sync.Mutex or a channel?](https://go.dev/wiki/MutexOrChannel))

### 4. Synchronize explicitly

Every ordering the code relies on comes from a channel, a lock, an atomic, or a wait group, never from timing. Locks
are short, never held across a blocking call, and taken in one order. *Ask: what makes this happen before that?*
([The Go memory model](https://go.dev/ref/mem))

### 5. Bound everything

Goroutines, queues, retries, and memory have limits, and the code decides what happens at the limit: wait, drop, or
fail. *Ask: what is the limit, and what happens when it is reached?*

### 6. Plain code over generality

A type, helper, or dependency earns its place by removing real repetition; misuse is handled by documented
contracts, as for `time.Ticker.C`. *Ask: would copying the few lines be simpler?*
([Go Proverbs](https://go-proverbs.github.io/))

### 7. Prefer explicit inputs

Prefer code whose behavior follows from its explicit inputs: its state, the events it receives, and injected
dependencies such as a clock or a random source. Such code can be tested on its own and a failure reproduced from
the inputs that caused it. It is a preference, not a rule: where nothing is replayed or compared, a simpler design
that lets the scheduler or the clock decide is often the better choice (principle 3). *Ask: will anyone need the
same inputs to give the same result, and does that justify the structure?*
([Schneider (1990)](https://doi.org/10.1145/98163.98167))

## Anti-patterns

An anti-pattern violates a principle where a solution that follows them exists, whether it fails yet or not. Leaks,
races, deadlocks, lost errors, and unbounded memory are how it eventually shows. A design that only makes it possible
for others to violate a principle is not one.

## Choosing a pattern

### Bounded concurrency

Capping how many things run, or how many resources are used, at once. When callers need only permission to proceed, a [Semaphore](references/patterns/semaphore.md) bounds how many run at once. When a fixed number of distinct, expensive resources such as connections must be shared, a [Resource Pool](references/patterns/resource-pool.md) hands them out and makes callers wait when all are in use. When the work itself should run on a fixed set of long-lived goroutines fed from a queue, use a [Worker Pool](references/patterns/worker-pool.md). An [Object Cache](references/patterns/object-cache.md) reuses temporary objects but bounds nothing.

### Cancellation

Stopping work that is no longer needed. A [Deadline](references/stdlib/deadline.md) bounds how long the caller waits for an operation, and it only signals the operation. [Cooperative cancellation](references/concepts/cooperative-cancellation.md) is what makes a long operation notice the signal: it checks the context between units, stops within one unit, and leaves state consistent. [Context Propagation](references/stdlib/context-propagation.md) carries the deadline and the cancellation from the top of a request down the call tree, so every step shares one bound. [Cancel on Error](references/patterns/cancel-on-error.md) handles a group of concurrent tasks that build one result: the first failure cancels the rest.

### Rate control

Limiting how often something happens over time. To limit how often an operation starts, choose by what may happen in a short span and whether callers wait or are refused. A [Token Bucket](references/patterns/token-bucket.md) enforces an average rate and lets a bounded burst through after idle time; callers can be refused or wait, and `golang.org/x/time/rate` is the production implementation. [Pacing](references/patterns/pacing.md) spaces calls one interval apart with no burst; every caller waits its turn. To limit how many operations run at once instead of how often they start, use a [Semaphore](references/patterns/semaphore.md).

## Patterns

### Data flow

How values move between goroutines.

- [Batcher](references/patterns/batcher.md): Spread a per-call cost over a stream of items while bounding how long any item waits.
- [Bounded Queue](references/patterns/bounded-queue.md): Let a producer run ahead of its consumer by a fixed, chosen amount.
- [Fan-Out/Fan-In](references/patterns/fan-out-fan-in.md): Process independent items at the same time and collect all their results.
- [Or-Done](references/patterns/or-done.md): Stop reading a channel you do not control when your own work is canceled.
- [Pipeline](references/patterns/pipeline.md): Process a stream in independent steps that run at the same time, each handing its output to the next.
- [PubSub Hub](references/patterns/pubsub-hub.md): Deliver every published message to each current subscriber, while subscribers come and go.
- [Replay Buffer](references/patterns/replay-buffer.md): Let a late subscriber catch up on the last n messages before it receives live ones.
- [Resequencer](references/patterns/resequencer.md): Consume results in input order when concurrent work delivers them out of order.
- [Spill Queue](references/patterns/spill-queue.md): Absorb a backlog that briefly exceeds what memory should hold, without making the producer wait for the consumer and without dropping items.
- [Tee](references/patterns/tee.md): Give each value of one stream to two consumers that process it independently.
- [Watermark](references/patterns/watermark.md): Act on events in the order of their timestamps when they arrive from several inputs at different speeds, without guessing how long to wait for the slower ones.
- [Windowing](references/patterns/windowing.md): Compute a count, sum, or other aggregate over each span of time of a stream, for fixed spans or for the most recent one.

### Execution

Who runs the work, and how the outcome comes back.

- [Active Object](references/patterns/active-object.md): Run all operations on an object one at a time, in the order they arrive, while callers either continue or wait for the result.
- [Future](references/patterns/future.md): Overlap independent operations and collect each result when it is needed.
- [Group Commit](references/patterns/group-commit.md): Let concurrent callers share one execution of an expensive operation, without adding latency when only one caller is waiting.
- [State Machine](references/patterns/state-machine.md): Change a stateful object only in steps that are legal for its current state, so that the same events in the same order always lead to the same state.
- [Worker Pool](references/patterns/worker-pool.md): Process any number of jobs with a fixed amount of concurrency.

### Coordination

How goroutines wait for, limit, or choose among each other, and how shared state is kept safe.

- [Compare-and-Swap Loop](references/patterns/compare-and-swap-loop.md): Update a single shared value from its old value without making other writers wait.
- [Copy-on-Write Snapshot](references/patterns/copy-on-write-snapshot.md): Let many readers see a consistent version of read-mostly data without waiting for writers or for each other.
- [For-Select Loop](references/patterns/for-select-loop.md): Keep a goroutine waiting on several events, or doing work, while it still notices a stop signal.
- [Monitor](references/patterns/monitor.md): Make a type's state safe to use from many goroutines, with no access left unprotected.
- [Object Cache](references/patterns/object-cache.md): Avoid allocating and garbage-collecting the same kind of temporary object over and over in a hot path.
- [Online Attach](references/patterns/online-attach.md): Let members join and leave a running component without stopping it or racing with its work.
- [Pacing](references/patterns/pacing.md): Start calls at a fixed interval, one at a time, with no bursts.
- [Resource Pool](references/patterns/resource-pool.md): Share a fixed number of expensive resources among many goroutines, and wait when all are in use.
- [Semaphore](references/patterns/semaphore.md): Limit how many operations use a resource at the same time.
- [Snapshot via Owner](references/patterns/snapshot-via-owner.md): Read a consistent copy of state that one goroutine owns, without sharing the state.
- [Token Bucket](references/patterns/token-bucket.md): Limit the average rate of an operation while allowing a bounded burst.

### Lifecycle

How goroutines are stopped, and kept from leaking.

- [Close with Cause](references/patterns/close-with-cause.md): Tell the receivers of a closed channel why it was closed.

### Resilience

How errors, slowness, and failure are handled to keep a system working.

- [Cancel on Error](references/patterns/cancel-on-error.md): When one of several concurrent tasks fails, stop the others and report the error that caused it.
- [Healing Goroutines](references/patterns/healing-goroutines.md): Detect a stuck worker and restart it.
- [Heartbeat](references/patterns/heartbeat.md): Tell a slow worker from a stuck one, and know when work has started, without sleeping.
- [Load Shedding](references/patterns/load-shedding.md): Keep the producer and the latency bounded under overload by dropping the work that does not fit.
- [Request Racing](references/patterns/request-racing.md): Get the fastest answer from several equivalent replicas and stop the others.
- [Result with Error](references/patterns/result-with-error.md): Get each error from concurrent work to the code that can decide what to do with it.
- [Slow Consumer Eviction](references/patterns/slow-consumer-eviction.md): Keep a consumer that keeps missing deliveries from holding back the others.
- [Timed Send](references/patterns/timed-send.md): Hand a value to a consumer, but give up after a deadline and tell the caller whether it was delivered.

## Concepts

Shared background that the patterns link to.

- [Backpressure](references/concepts/backpressure.md): Let the consumer's remaining capacity limit the producer's rate, so a faster producer slows down instead of piling up work.
- [Channel Ownership](references/concepts/channel-ownership.md): Give every channel one owner, the single sender or the closer that coordinates several, so that a close comes after the last send and is never raced or repeated.
- [Confinement](references/concepts/confinement.md): Keep data reachable from only one goroutine at a time, so it needs no synchronization.
- [Cooperative Cancellation](references/concepts/cooperative-cancellation.md): Stop work whose result is no longer needed by having the work itself check a signal and return, leaving state consistent.
- [Determinism](references/concepts/determinism.md): A computation is deterministic when the same starting state and the same inputs, in the same order, always produce the same result.
- [Goroutine Leak Prevention](references/concepts/goroutine-leak-prevention.md): Give every goroutine an owner that is responsible for its end, and a way to end that does not depend on a peer that may have left.
- [Memory Visibility](references/concepts/memory-visibility.md): A write made in one goroutine is guaranteed visible in another only if a synchronization edge of the Go memory model orders the two.
- [Ordering](references/concepts/ordering.md): Which orders between values a concurrent design preserves, which it loses, and what restoring a lost order costs.
- [Overload Policy](references/concepts/overload-policy.md): Decide deliberately what happens when a limit is reached, trading loss, latency, and memory.
- [Timers and Tickers](references/concepts/timers.md): The guarantees Go's timers and tickers give about when they fire, what their channels deliver, and when they are released.

## Standard library guides

How to use what Go already provides.

- [Condition Variable](references/stdlib/condition-variable.md): Let goroutines wait until a condition on shared state holds, and wake them when it may have changed.
- [Context Propagation](references/stdlib/context-propagation.md): Stop all the work done for a request together, and give it the request's deadline and scoped values.
- [Deadline](references/stdlib/deadline.md): Bound how long an operation may take.
- [Lazy Initialization](references/stdlib/lazy-initialization.md): Build a value on first use, exactly once, safely from many goroutines.
- [Synctest](references/stdlib/synctest.md): Test concurrent and time-dependent code deterministically, with fake time and a way to wait until every goroutine is blocked.

## Anti-pattern checklist

For a review, look for each of these; the linked page has the example and the fix.

### [Batch Without a Time Bound](references/anti-patterns/batch-without-time-bound.md)

Collect items into batches that are flushed only when full, so at low load an item waits indefinitely and the last partial batch is lost at shutdown.

Violates: Bound everything. Fixed by: [Batcher](references/patterns/batcher.md), [Group Commit](references/patterns/group-commit.md).

Look for:

- A batching loop whose only flush condition is `len(batch) == maxSize`, with no timer, ticker, or deadline in its
  `select`.
- A `for item := range in` loop that ends without using `batch`.
- Tests that only send multiples of the batch size.

### [Buffer as a Band-Aid](references/anti-patterns/oversized-buffer.md)

Hide a producer that outruns its consumer behind a huge channel buffer, so the failure comes later and larger.

Violates: Bound everything. Fixed by: [Bounded Queue](references/patterns/bounded-queue.md), [Load Shedding](references/patterns/load-shedding.md), [Spill Queue](references/patterns/spill-queue.md).

Look for:

- `len(ch)` near `cap(ch)`, or growing without bound, in metrics (each read is only a snapshot).
- The age of items when dequeued, not only the queue length.
- Large buffer sizes with no derivation, or ones raised to make a hang disappear.
- A heap profile attributes the buffer to its `make`, and queued data to the code that allocated it.

### [Channel Generator](references/anti-patterns/channel-generator.md)

Produce a sequence in a goroutine of its own and hand it to the consumer through a channel, where a callback or an iterator would produce it in the consumer's goroutine.

Violates: Synchronous first. Fixed by: [Pipeline](references/patterns/pipeline.md).

Look for:

- A function that starts a goroutine whose only job is to compute values and send them, and returns the channel.
- At call sites: a context created and canceled only to stop the producer, or a drain loop after a `break`.
- Goroutine profiles and the goroutine leak profile show producers blocked on a send after their consumer left.
  `go vet` and the race detector do not report it.

### [Channel Where a Mutex Would Do](references/anti-patterns/channel-instead-of-mutex.md)

Guard plain state with an owner goroutine and request and reply channels, where a mutex or an atomic next to the state would do, so every call pays a round trip and the state gains a goroutine whose lifetime someone has to manage.

Violates: The simplest mechanism that works. Fixed by: [Monitor](references/patterns/monitor.md), [Compare-and-Swap Loop](references/patterns/compare-and-swap-loop.md).

Look for:

- A goroutine whose cases only read or assign fields, and whose callers immediately wait for a reply.
- A type that needs `stop` but starts no I/O and no timer.
- Goroutine profiles with one idle owner goroutine per instance of a data type.

### [Closing from the Wrong Side](references/anti-patterns/wrong-side-close.md)

A channel is closed by a goroutine that does not own it, so a later send or a second close panics.

Violates: Clear ownership. Fixed by: [Close with Cause](references/patterns/close-with-cause.md).

Look for:

- The panic's stack trace points at the send or the second close.
- `-race` reports a close not ordered with a send on the same channel, if both happen during the run. No `go vet`
  analyzer checks who closes a channel.
- In review: `close(ch)` in a function that only receives from `ch`, or `close` calls in more than one place.

### [Copied Mutex](references/anti-patterns/copied-mutex.md)

Copy a value that contains a mutex, so the copy has its own lock state and code that locks it does not exclude code that locks the original.

Violates: Clear ownership. Fixed by: [Monitor](references/patterns/monitor.md).

Look for:

- `go vet`'s copylocks analyzer, which `go test` runs by default, reports copies of types containing a lock
  (`sync.Mutex`, `RWMutex`, `WaitGroup`, `Cond`, `Once`, or any type whose pointer has `Lock` and `Unlock`): "passes
  lock by value", "assignment copies lock value", "return copies lock value", "call of ... copies lock value",
  "range var ... copies lock".
- vet sees only copies where the static type contains a lock; copies through a type parameter or `reflect` are not
  reported.
- The race detector reports the resulting races when they happen during a run.

### [Double-Checked Locking](references/anti-patterns/double-checked-locking.md)

Test a lazily initialized pointer without synchronization and lock only when it is nil, so readers race with the writer and can see a pointer to an incompletely initialized value.

Violates: Synchronize explicitly. Fixed by: [Lazy Initialization](references/stdlib/lazy-initialization.md), [Compare-and-Swap Loop](references/patterns/compare-and-swap-loop.md).

Look for:

- `go test -race` reports the race only when a test runs the read and the write concurrently.
- In review, look for a nil or flag check outside a lock on a variable written inside it.

### [Fire-and-Forget Goroutine](references/anti-patterns/fire-and-forget-goroutine.md)

Start a goroutine and return, so nobody can wait for it, learn of its error or stop it.

Violates: Clear ownership. Fixed by: [Result with Error](references/patterns/result-with-error.md).

Look for:

- For every `go` statement, ask who waits for it, who receives its error, and what stops it.
- `go vet` does not check for this; the race detector reports it only when the goroutine's writes race with a read.
- A goroutine dump, the `pprof` goroutine profile, or a check for leftover goroutines at the end of a test shows
  goroutines that outlive their starter. The `goroutineleak` profile reports those blocked on a channel or lock that
  no running goroutine can reach.
- A test that needs `time.Sleep` to see a function's effect hints at work nobody can wait for.

### [Goroutine as a Buffer](references/anti-patterns/goroutine-as-buffer.md)

Park a goroutine on a channel send so that the caller never blocks, which makes every pending value cost a goroutine instead of a channel slot.

Violates: Bound everything. Fixed by: [Bounded Queue](references/patterns/bounded-queue.md), [Load Shedding](references/patterns/load-shedding.md), [Spill Queue](references/patterns/spill-queue.md).

Look for:

- `runtime.NumGoroutine` or `/sched/goroutines` tracking the backlog, not the number of workers.
- A goroutine profile (`/debug/pprof/goroutine?debug=1`) with thousands of identical stacks ending in a send.
- `/memory/classes/heap/stacks:bytes` growing with the backlog.
- The `goroutineleak` profile reports senders that no running goroutine can reach.
- `go` statements whose body is a single send. `go vet` and the race detector do not report it.

### [Ignored Context](references/anti-patterns/ignored-context.md)

A function takes a context and never checks it, so cancellation and deadlines have no effect.

Violates: Clear ownership. Fixed by: [Context Propagation](references/stdlib/context-propagation.md), [Deadline](references/stdlib/deadline.md).

Look for:

- A test that cancels while the work is blocked, as in the example, and checks that the function returns.
- Goroutine dumps and a rising goroutine count show the blocked goroutines.
- A `ctx` parameter never used in the body, which linters for unused parameters report; `go vet` does not
  (`lostcancel` only checks that a cancel function is called). Passing `ctx` only to functions that also ignore it
  does not count as using it.

### [Implicit State Machine](references/anti-patterns/implicit-state-machine.md)

Keep a lifecycle in several boolean flags under a mutex, so combinations that should be impossible are reachable and every method has to re-check them.

Violates: Synchronize explicitly. Fixed by: [State Machine](references/patterns/state-machine.md).

Look for:

- Look for two or more booleans that describe the same lifecycle, and for conditions that combine them.
- Look for a lock released and taken again inside one operation, with a flag written after the second `Lock`
  that was tested before the first `Unlock`.
- The race detector does not report it: every access is under the lock.

### [Inconsistent Lock Order](references/anti-patterns/inconsistent-lock-order.md)

Take two locks in different orders in different goroutines, so each can hold one while waiting for the other, and both wait forever.

Violates: Synchronize explicitly. Fixed by: [Monitor](references/patterns/monitor.md), [Active Object](references/patterns/active-object.md).

Look for:

- A goroutine dump (`SIGQUIT`, or the `pprof` goroutine profile) shows goroutines blocked in `sync.(*Mutex).Lock`,
  each holding a lock another one waits for.
- In review, find code that holds one lock while taking another, and check every path takes them in the same order.
- The race detector does not report deadlocks, and `go vet` does not check lock order.

### [Lock Held While Blocking](references/anti-patterns/lock-held-while-blocking.md)

Hold a mutex across a channel operation, I/O or a callback, so everyone else who needs the lock waits, and a deadlock follows if the other side needs it too.

Violates: Synchronize explicitly. Fixed by: [Active Object](references/patterns/active-object.md), [Snapshot via Owner](references/patterns/snapshot-via-owner.md).

Look for:

- Look between `Lock` and `Unlock` for channel operations, `select`, I/O, sleeps, and calls to function values or
  interface methods.
- Goroutine dumps and the mutex and block profiles show the holder and its waiters. The race detector and
  `go vet` do not report it.

### [Needless Asynchrony](references/anti-patterns/needless-asynchrony.md)

Start a goroutine inside a function and hand back a channel, where returning the result would do, so the caller loses the error path, the cancellation, and control over the goroutine.

Violates: Synchronous first. Fixed by: [Future](references/patterns/future.md).

Look for:

- `go func() { ... ch <- v }()` followed by `return ch`, for work that does not produce values over time.
- Call sites that receive immediately: `<-f(...)`.
- Goroutine and goroutine leak profiles show the abandoned sends; `go vet` and the race detector do not report it.

### [Non-Deterministic State Machine](references/anti-patterns/non-deterministic-state-machine.md)

A transition whose outcome is not part of its input, so the same state and event can lead to different states.

Violates: Prefer explicit inputs. Fixed by: [State Machine](references/patterns/state-machine.md).

Look for:

- Read every transition for calls to `time.Now`, `math/rand`, `os.Getenv`, the network, or the disk, and for
  `range` over a map or a `select` with several ready cases; see the
  [sources of nondeterminism in Go](references/concepts/determinism.md#sources).
- Replay recorded events twice, or with a different random source, and compare the states.
- Run the tests with `-count` and `-shuffle=on`; a test of a transition that needs a seed to pass is a sign.

### [Pointer over a Channel](references/anti-patterns/pointer-over-channel.md)

A pointer, slice, or map is sent over a channel and then used by both sides, so the data is shared although it was handed over.

Violates: Clear ownership. Fixed by: [Snapshot via Owner](references/patterns/snapshot-via-owner.md), [Copy-on-Write Snapshot](references/patterns/copy-on-write-snapshot.md).

Look for:

- `go test -race` reports the unordered accesses, if the test runs both. When a handshake orders them, as in the
  example, nothing is reported and the aliasing is a logic error.
- No `go vet` analyzer checks this. In review, look at what a channel element type contains and what the sender does
  with a value after sending it.

### [Premature Abstraction](references/anti-patterns/premature-abstraction.md)

Extract a generic, configurable concurrency helper before its uses are known, so each caller sets flags for the behavior it needs, the flags interact, and every caller depends on code paths it does not use.

Violates: Plain code over generality. Fixed by: [Worker Pool](references/patterns/worker-pool.md), [Future](references/patterns/future.md).

Look for:

- A generic concurrency helper with a configuration struct and fewer than three callers.
- Boolean options, especially pairs whose combination is not documented or tested.
- A `switch` or `if` chain on options inside the per-item loop.

### [Premature Parallelism](references/anti-patterns/premature-parallelism.md)

Spread cheap work on data that is already in hand across goroutines and channels, so the handoffs cost more than the work, the order is lost, and a loop's worth of code becomes a pipeline to manage.

Violates: The simplest mechanism that works. Fixed by: [Worker Pool](references/patterns/worker-pool.md), [Fan-Out/Fan-In](references/patterns/fan-out-fan-in.md).

Look for:

- A channel fed from a slice, read by workers doing a few arithmetic operations.
- A sort after the collection loop that the sequential code did not need.
- CPU profiles dominated by `runtime.chansend`, `runtime.chanrecv`, and the scheduler.

### [Sleeping to Synchronize](references/anti-patterns/sleep-synchronization.md)

Use time.Sleep or a guessed delay to wait for another goroutine instead of waiting for its completion.

Violates: Synchronize explicitly. Fixed by: [Future](references/patterns/future.md), [Heartbeat](references/patterns/heartbeat.md), [Watermark](references/patterns/watermark.md).

Look for:

- `time.Sleep` that is not a deliberate delay, especially right after a `go` statement and in tests.
- `go test -race` reports the read when the result is a plain variable and both accesses happen in the run, even when
  the sleep is long enough for the value to be correct. No `go vet` analyzer checks this.
- Tests under load (`-cpu`, `-count`, `stress`) make the failures appear.

### [Unbounded Goroutines](references/anti-patterns/unbounded-goroutines.md)

Start one goroutine per input item, so concurrency grows with the input instead of with capacity.

Violates: Bound everything. Fixed by: [Worker Pool](references/patterns/worker-pool.md), [Semaphore](references/patterns/semaphore.md).

Look for:

- `go` statements inside loops over input of unknown or large size.
- Goroutine count (`runtime.NumGoroutine`, the goroutine profile) and memory growing with load.
- Load tests with large inputs; the race detector and `go vet` do not report it.
- Descriptor, connection, or rate-limit errors downstream.

### [Unbounded Reorder Buffer](references/anti-patterns/unbounded-reorder-buffer.md)

Hold everything that arrives behind a missing item until it arrives, with no limit, so one slow or lost item stops all output while the buffer grows.

Violates: Bound everything. Fixed by: [Resequencer](references/patterns/resequencer.md), [Watermark](references/patterns/watermark.md).

Look for:

- A map or slice keyed by sequence number, deleted from only when `next` arrives, with no length check or timeout.
- The ordering stage's buffer or memory rising while its output rate is zero.
