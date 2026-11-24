# Timed Send

Hand a value to a consumer, but give up after a deadline and tell the caller whether it was delivered.

- **Also known as:** Send with timeout, Deadline send
- **Purpose:** Resilience
- **Scale:** Idiom
- **Mechanisms:** Channels, Select, Time
- **Concerns:** Latency, Overload
- **Concepts:** [Overload Policy](../concepts/overload-policy.md), [Backpressure](../concepts/backpressure.md), [Cooperative Cancellation](../concepts/cooperative-cancellation.md), [Timers and Tickers](../concepts/timers.md)
- **Code:** [`patterns/timedsend`](https://github.com/johnknl/go-concurrency-patterns/tree/main/patterns/timedsend)

## Problem

A plain send blocks for as long as the consumer does not receive, so one consumer can stall a producer that serves
several. Bounding the wait gives the producer a result to act on: drop the value, count a miss, or evict the consumer,
one of the [overload options](../concepts/overload-policy.md#options). The bound belongs to each send, not to the
producer's whole run: one timeout, configured once, limits every send that has to wait, while sends that can complete at
once cost nothing extra.

## Mechanism

First a `select` with a `default` case tries the send without blocking; with room or a waiting receiver it succeeds
and no timer is involved, even if the context is already canceled. Only when it fails does the function arm a
timer with the configured duration and `select` on the send, the timer, and the context, so the context still
bounds the whole operation and the timer bounds this one send; the wait stays cancelable
([waiting must be cancelable](../concepts/overload-policy.md#wait-cancelable),
[blocking points select on Done](../concepts/cooperative-cancellation.md#blocking-selects-done)).

Reusing one timer saves an allocation per slow send (`time.After` is also correct, since
[unreferenced timers are collected](../concepts/timers.md#collected)). Reuse needs no draining before `Reset`
([synchronous timer channels](../concepts/timers.md#synchronous-channels)).

```go
func timedSend(ctx context.Context, ch chan<- int, v int, t *time.Timer, d time.Duration) bool {
	select {
	case ch <- v: // fast path: no timer is armed
		return true
	default:
	}

	t.Reset(d)
	defer t.Stop()

	select {
	case ch <- v:
		return true
	case <-t.C:
		return false
	case <-ctx.Done():
		return false
	}
}

func newStoppedTimer() *time.Timer {
	t := time.NewTimer(time.Hour)
	t.Stop()

	return t
}

func main() {
	ctx := context.Background()
	t := newStoppedTimer()

	// One free slot: the first send takes the fast path, the second finds the
	// channel full, and nobody reads, so it times out.
	ch := make(chan int, 1)
	fmt.Println(timedSend(ctx, ch, 1, t, time.Millisecond))
	fmt.Println(timedSend(ctx, ch, 2, t, time.Millisecond))

	// A receiver arrives while the sender waits on the timer.
	unbuffered := make(chan int)
	got := make(chan int)

	go func() { got <- <-unbuffered }()

	fmt.Println(timedSend(ctx, unbuffered, 3, t, time.Minute))
	fmt.Println(<-got)

	// A canceled context ends the wait even though the timer is far away.
	canceled, cancel := context.WithCancel(ctx)
	cancel()

	fmt.Println(timedSend(canceled, unbuffered, 4, t, time.Hour))

	// Output:
	// true
	// false
	// true
	// 3
	// false
}
```

## Applicability

Use it when a producer must not wait beyond a known bound per value and can react to a missed delivery. A deadline
on the context does not replace it: the context bounds the whole operation, not each send, and deriving a context
with `context.WithTimeout` for every send allocates a context and a timer each time, even for sends that would
succeed at once. When a value must not be lost and waiting is acceptable, a plain or context-aware send gives
[backpressure](../concepts/backpressure.md#feedback); a timed send is for a source that
[cannot wait](../concepts/backpressure.md#cannot-wait).

## Consequences

- The timer belongs to one goroutine; concurrent calls must not share it.
- `false` does not say whether the timeout or the context ended the wait; return `context.Cause(ctx)` if it matters
  ([return the cause](../concepts/cooperative-cancellation.md#return-cause)).
- On a timeout the value was not sent, and the caller owns it again.
- The timeout is not exact: scheduling delay adds to it ([at least its duration](../concepts/timers.md#at-least)),
  and `select` picks pseudo-randomly when the send and the timer are ready together
  ([select does not prefer cancellation](../concepts/cooperative-cancellation.md#select-random)).
- Every miss costs the full timeout, so stuck consumers visited in sequence add up.

## Related patterns

- [Deadline](../stdlib/deadline.md) bounds a wait on a result.
- [Slow Consumer Eviction](slow-consumer-eviction.md) counts missed deliveries and removes the consumer.
- [Load Shedding](load-shedding.md) is the non-blocking first phase on its own.

## Sources

- [Gerrand (2010a)](../sources.md#source-blog-timeouts)
