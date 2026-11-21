---
title: Pacing
aliases: ['Fixed-interval rate limiting', 'Leaky bucket as a queue']
intent: Start calls at a fixed interval, one at a time, with no bursts.
purpose: coordination
scale: idiom
package: patterns/pacing
mechanisms: [time]
concerns: [rate-control]
concepts:
  - timers: 'one pending tick, optional Stop, and at-least delays.'
original: true
---
# Pacing

<!-- meta -->
Start calls at a fixed interval, one at a time, with no bursts.

<div class="cp-props" markdown>

| | |
|---|---|
| Also known as | Fixed-interval rate limiting, Leaky bucket as a queue |
| Purpose | [Coordination](../catalog/purpose.md#purpose-coordination) |
| Scale | [Idiom](../catalog/scale.md#scale-idiom) |
| Mechanisms | [Time](../catalog/mechanism.md#mechanism-time) |
| Concerns | [Rate control](../catalog/concern.md#concern-rate-control) |
| Concepts | [Timers and Tickers](../concepts/timers.md) |
| Code | [`patterns/pacing`](https://github.com/johnknl/go-concurrency-patterns/tree/main/patterns/pacing) |

</div>
<!-- /meta -->

## Problem

A receiver should get calls evenly spaced, never several at once, however many callers share it and however long
they have been idle.


## Mechanism

Callers receive from one `time.Ticker` before each call. Each tick is received by one caller, so calls start at
least one interval apart, however many goroutines share the ticker. A `select` with the context lets a caller stop
waiting.

<!-- code: patterns/pacing#pace:doc patterns/pacing#Example_pacing:main -->
```go
// pace waits for the next tick, or returns the cause of ctx if ctx ends first. Callers share one ticker, and each
// tick lets one caller through.
func pace(ctx context.Context, tick <-chan time.Time) error {
	select {
	case <-tick:
		return nil
	case <-ctx.Done():
		return context.Cause(ctx)
	}
}

func main() {
	ctx := context.Background()

	ticker := time.NewTicker(time.Millisecond)
	defer ticker.Stop()

	for i := 1; i <= 3; i++ {
		fmt.Println("request", i, pace(ctx, ticker.C))
	}

	// Output:
	// request 1 <nil>
	// request 2 <nil>
	// request 3 <nil>
}
```
<!-- /code -->

A ticker holds at most one pending tick ([timers](../concepts/timers.md#ticker-drops)), so after an idle period one
call goes through at once and the rest wait for later ticks.

## Applicability

Use it when the receiver needs evenly spaced calls or its limit is strict over short spans, and the callers can
wait. If they would rather be refused, use a [Token Bucket](token-bucket.md)'s `allow`; if short bursts are
acceptable, a Token Bucket gives lower latency for the same average rate.

## Consequences

- Stop the ticker when the callers are done (`defer ticker.Stop()`); it is not required, but marks where it ends
  ([stop is optional](../concepts/timers.md#stop-optional)).
- The first call waits a full interval.
- Waiting callers are not served in a specified order.
- Throughput never exceeds one call per interval, even after idle time; each call starts at least one interval
  after the previous one, plus scheduling delay ([timers](../concepts/timers.md#at-least)).

## Variants

### Next free slot

Without a ticker, keep the time of the next free slot behind a mutex. Each caller takes `next = max(now, next)`,
moves `next` one interval on, and waits until its slot. The first call goes at once, idle time earns no burst, and
no ticker has to be stopped. A caller whose context ends keeps its slot, leaving a gap of one interval.

<!-- code: patterns/pacing#pacer:doc patterns/pacing#pacer.wait patterns/pacing#Example_pacer:main -->
```go
// pacer spaces calls at least interval apart without a ticker. next is the earliest time the next call may start.
type pacer struct {
	next     time.Time
	interval time.Duration
	mu       sync.Mutex
}

func (p *pacer) wait(ctx context.Context) error {
	p.mu.Lock()
	now := time.Now()
	at := p.next
	if at.Before(now) {
		at = now
	}
	p.next = at.Add(p.interval)
	p.mu.Unlock()

	delay := at.Sub(now)
	if delay <= 0 {
		return nil
	}

	timer := time.NewTimer(delay)
	defer timer.Stop()

	select {
	case <-timer.C:
		return nil
	case <-ctx.Done():
		return context.Cause(ctx)
	}
}

func main() {
	p := &pacer{interval: time.Millisecond}
	start := time.Now()

	for i := 1; i <= 3; i++ {
		fmt.Println("request", i, p.wait(context.Background()))
	}

	fmt.Println("took at least 2ms:", time.Since(start) >= 2*time.Millisecond)

	// Output:
	// request 1 <nil>
	// request 2 <nil>
	// request 3 <nil>
	// took at least 2ms: true
}
```
<!-- /code -->

## Related patterns

- [Token Bucket](token-bucket.md) allows a burst on top of the average rate; with a burst of 1 it paces.
- [Deadline](../stdlib/deadline.md) for the `select` with the context.

## Sources

<!-- sources -->
Original work.
<!-- /sources -->
