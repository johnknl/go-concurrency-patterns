---
title: Timers and Tickers
aliases: ['time.Timer', 'time.Ticker']
intent: The guarantees Go's timers and tickers give about when they fire, what their channels deliver, and when they are released.
concerns: [latency, rate-control, leak-prevention]
sources:
  - id: go-time
  - id: go-testing-synctest
---
# Timers and Tickers

<!-- meta -->
The guarantees Go's timers and tickers give about when they fire, what their channels deliver, and when they are released.

<div class="cp-props" markdown>

| | |
|---|---|
| Also known as | time.Timer, time.Ticker |
| Concerns | [Latency](../catalog/concern.md#concern-latency), [Rate control](../catalog/concern.md#concern-rate-control), [Leak prevention](../catalog/concern.md#concern-leak-prevention) |

</div>
<!-- /meta -->

## Why it matters

Timeouts, flush intervals, rate limits, and deadlines are built on `time.Timer` and `time.Ticker`. Their semantics
changed in Go 1.23, and much advice written before then (drain the channel before `Reset`, never use `time.After` in
a loop, always `defer Stop`) is now unnecessary or wrong. Patterns rely on the current guarantees, listed here once.

## Properties

### Timer channels are synchronous since Go 1.23 { #synchronous-channels }

When the main module's `go.mod` says `go 1.23` or later and the `asynctimerchan` GODEBUG setting is not set, a
timer's or ticker's channel is unbuffered. After `Stop` or `Reset` returns, no receive gets a value from before the
call. A timer can be stopped and reset without draining its channel. Below Go 1.23, or with `asynctimerchan=1`, the
channel has a buffer of one, a stale value can remain after `Stop`, and code must stop and drain before `Reset`.

### Unreferenced timers are garbage collected { #collected }

Since Go 1.23, a timer or ticker that nothing references can be collected even if it was never stopped. `time.After`
in a loop or a `select` is therefore fine: each iteration's timer is released once it is unreachable. Reusing one
timer still saves an allocation per iteration when that matters.

### Stop releases early; it is not required { #stop-optional }

`defer t.Stop()` releases a timer or ticker before it becomes unreachable and makes its end explicit in the code, but
omitting it no longer leaks. Stopping a ticker remains good practice where its lifetime is not obvious.

### A ticker holds one pending tick { #ticker-drops }

A ticker's channel holds at most one tick that has not been received. When the reader falls behind, later ticks are
dropped rather than queued. The period then stretches: after a slow iteration the next tick may be ready at once,
and the ones missed in between are gone. A ticker never delivers a burst to catch up, and it cannot be used to count
elapsed periods.

### A timer fires after at least its duration { #at-least }

A timer fires no earlier than its duration, and later by the scheduling delay: the runtime must notice it, and the
receiving goroutine must be scheduled. Under load the delay can be large. Timeouts are therefore lower bounds for
waiting, not exact times, and a `select` that also has other ready cases may pick one of those first.

### testing/synctest makes time deterministic { #synctest }

Inside a `testing/synctest` bubble (`synctest.Test`, Go 1.25), time is fake: it advances only when every goroutine
in the bubble is blocked, and then jumps to the next timer. Tests of timeouts, tickers, and rate limits run instantly
and give the same result every time, without sleeps or tolerances.

## Used by

<!-- usedby -->
- [Batcher](../patterns/batcher.md): one reusable timer is stopped without draining (Go 1.23).
- [Deadline](../stdlib/deadline.md): timers are collected when unreferenced and fire after at least their duration.
- [Pacing](../patterns/pacing.md): one pending tick, optional Stop, and at-least delays.
- [State Machine](../patterns/state-machine.md): a timeout is a timer whose firing is sent to the owner as an event.
- [Synctest](../stdlib/synctest.md): timers and tickers in a bubble run on a fake clock.
- [Timed Send](../patterns/timed-send.md): one reused timer needs no draining and fires after at least its duration.
- [Token Bucket](../patterns/token-bucket.md): tests use synctest, and the channel variant loses ticks when behind.
- [Windowing](../patterns/windowing.md): a Ticker drops ticks while emit blocks; stopping it is optional.
<!-- /usedby -->

## Related concepts

- [Cooperative Cancellation](cooperative-cancellation.md): a deadline is cancellation triggered by a timer.
- [Overload Policy](overload-policy.md#options): bounded waiting uses a timer.

## Sources

<!-- sources -->
- [The Go Authors (n.d.h)](../sources.md#source-go-time)
- [The Go Authors (n.d.g)](../sources.md#source-go-testing-synctest)
<!-- /sources -->
