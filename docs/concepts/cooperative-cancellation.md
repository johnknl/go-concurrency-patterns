---
title: Cooperative Cancellation
aliases: ['Preemptable work', 'Cancellation']
intent: Stop work whose result is no longer needed by having the work itself check a signal and return, leaving state consistent.
concerns: [cancellation, latency, leak-prevention]
sources:
  - id: cox-buday
    at: 'Ch. 5, "Timeouts and Cancellation"'
  - id: blog-context
  - id: blog-pipelines
    at: '"Explicit cancellation"'
---
# Cooperative Cancellation

<!-- meta -->
Stop work whose result is no longer needed by having the work itself check a signal and return, leaving state consistent.

<div class="cp-props" markdown>

| | |
|---|---|
| Also known as | Preemptable work, Cancellation |
| Concerns | [Cancellation](../catalog/concern.md#concern-cancellation), [Latency](../catalog/concern.md#concern-latency), [Leak prevention](../catalog/concern.md#concern-leak-prevention) |

</div>
<!-- /meta -->

## Why it matters

Go cannot stop a goroutine from the outside. Canceling a context, or closing a done channel, only announces that the
result is no longer wanted. Work that does not look at the announcement keeps running, holds its resources, delays
whoever waits for it to finish, and may change shared state after its caller has gone. When it does stop, it may have
done half of what it set out to do. Cancellation is therefore a protocol between the canceler and the work, and the
work has to keep its side.

## Properties

### Canceling is a signal, not an effect { #signal-not-effect }

`cancel()` closes the context's `Done` channel and records an error; nothing else happens. The work stops only when
it next looks at `Done` or `Err` and returns. Between the signal and the stop there is always a delay, and code
that cancels and then needs the work to have ended must also wait for it (see
[Goroutine Leak Prevention](goroutine-leak-prevention.md#observable-end)).

### Every blocking point selects on Done { #blocking-selects-done }

A goroutine blocked on a channel operation, a lock, or a timer does not see a cancellation. Each blocking channel
operation therefore sits in a `select` with `<-ctx.Done()` (or a done channel), and each blocking call takes the
context. One blocking point that does not is enough to make the whole operation ignore cancellation.

```go
select {
case out <- v:
case <-ctx.Done():
	return context.Cause(ctx)
}
```

### Work is split into units with check points { #check-points }

CPU-bound work and loops have no blocking point to select on. They check `ctx.Err()` between units of work: an
iteration, a chunk, a call. The unit is the granularity of cancellation.

### Latency is bounded by the longest unit { #latency-bound }

After a cancel, the work runs until its next check point. The worst-case cancellation latency is the longest unit
between two checks, plus scheduling. A unit that calls something that ignores the context (a blocking syscall, a
library without a context parameter) extends that bound to the length of the call.

### Return the context's error or cause { #return-cause }

Canceled work returns `ctx.Err()` (`context.Canceled` or `context.DeadlineExceeded`), or `context.Cause(ctx)` when
the canceler supplied a reason with `WithCancelCause` or `WithDeadlineCause`. The caller can then tell cancellation
from failure, and errors that arrive after cancellation are not mistaken for the reason the work stopped.

### Ignoring the context delays whoever waits { #ignore-delays }

The cost of not cooperating falls on others: a `WaitGroup.Wait`, an `errgroup.Wait`, a server shutdown, or a parent
that joins its children all wait for the slowest goroutine to notice. A timeout is only as good as the slowest
participant's cooperation.

### Select does not prefer cancellation { #select-random }

When several cases of a `select` are ready, Go picks one pseudo-randomly. A pending cancellation is therefore not
always taken first: a loop whose input is also ready may process a few more values before it sees `Done`. If no
further work may start after cancellation, check `ctx.Err()` (or a non-blocking receive from `Done`) at the top of
each iteration as well; it narrows the window but cannot close it, since the cancel can arrive just after the check.

### Half-done work is undone or idempotent { #consistent-state }

Work stopped between two steps leaves the effect of the first. Either the steps are arranged so that every stop point
leaves a consistent state (write to a temporary place and publish at the end, roll back on return, use a
transaction), or the work is idempotent so that running it again from the start is harmless.

### Retried or duplicated work is pure or idempotent { #duplicates-idempotent }

Patterns that start the same work more than once, racing it, hedging it, retrying a compare-and-swap, or restarting
a failed goroutine, cancel or discard all but one result. The discarded runs may already have had effects. Such work
must be pure (no effects outside its result) or idempotent (doing it twice equals doing it once).

## Used by

<!-- usedby -->
- [Cancel on Error](../patterns/cancel-on-error.md): the first error cancels the others, which stop at their own pace and return the cause.
- [Close with Cause](../patterns/close-with-cause.md): the context variant reports the cause with context.Cause.
- [Compare-and-Swap Loop](../patterns/compare-and-swap-loop.md): the update may be retried, so it must be pure.
- [Context Propagation](../stdlib/context-propagation.md): canceling closes Done; each function stops and returns the error or cause.
- [Copy-on-Write Snapshot](../patterns/copy-on-write-snapshot.md): a CAS writer may repeat its change, so the change must be pure.
- [Deadline](../stdlib/deadline.md): the deadline only closes Done; the operation stops at its next check and returns the error.
- [For-Select Loop](../patterns/for-select-loop.md): every blocking case selects on Done, which select does not prefer.
- [Healing Goroutines](../patterns/healing-goroutines.md): the ward must observe its context, and redone work must be idempotent.
- [Or-Done](../patterns/or-done.md): select picks at random, so a few values may pass after done.
- [Request Racing](../patterns/request-racing.md): requests are duplicated, and a replica that ignores its context delays the caller.
- [Timed Send](../patterns/timed-send.md): the wait selects on the context as well as the timer.
<!-- /usedby -->

## Violated by

<!-- violatedby -->
- [Ignored Context](../anti-patterns/ignored-context.md): the function never selects on Done or checks Err.
<!-- /violatedby -->

## Related concepts

- [Goroutine Leak Prevention](goroutine-leak-prevention.md): cancellation is the usual way a goroutine is told to end.
- [Timers and Tickers](timers.md): deadlines are cancellation driven by a timer.

## Sources

<!-- sources -->
- [Cox-Buday (2017)](../sources.md#source-cox-buday), Ch. 5, "Timeouts and Cancellation"
- [Ajmani (2014a)](../sources.md#source-blog-context)
- [Ajmani (2014b)](../sources.md#source-blog-pipelines), "Explicit cancellation"
<!-- /sources -->
