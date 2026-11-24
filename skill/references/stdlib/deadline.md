# Deadline

Bound how long an operation may take.

- **Provides:** `context.WithTimeout`, `context.WithDeadline`, `context.WithTimeoutCause`, `context.WithDeadlineCause`, `time.Timer`
- **Concerns:** Cancellation, Latency
- **Concepts:** [Cooperative Cancellation](../concepts/cooperative-cancellation.md), [Timers and Tickers](../concepts/timers.md)
- **Code:** [`stdlib/deadline`](https://github.com/johnknl/go-concurrency-patterns/tree/main/stdlib/deadline)

## What it provides

`context.WithTimeout` and `context.WithDeadline` return a context whose `Done` channel closes at a point in time, after
which `Err` returns `context.DeadlineExceeded`. `context.WithTimeoutCause` and `context.WithDeadlineCause` attach an
error that `context.Cause` returns, so the caller can tell which bound expired. Without a context, a `time.Timer`
bounds a single wait.

Reach for them for calls to remote or unreliable resources and for waits on a goroutine that may never answer. A
deadline set once at the top of a request covers all work done for it, and each step can derive a shorter one
([Context Propagation](context-propagation.md)).

## Using it

The operation waits on `ctx.Done()` in a `select` next to its real work and returns `ctx.Err()` when it closes; a
single wait without a context selects on a timer's channel instead
([return the cause](../concepts/cooperative-cancellation.md#return-cause)). In these examples, the deadline has already
passed and the operation never completes, so the output does not depend on timing.

```go
func wait(ctx context.Context) error {
	<-ctx.Done()

	return ctx.Err()
}

func main() {
	ctx, cancel := context.WithDeadline(context.Background(), time.Now().Add(-time.Second))
	defer cancel()

	err := wait(ctx)

	fmt.Println(err)
	fmt.Println(errors.Is(err, context.DeadlineExceeded))

	// Output:
	// context deadline exceeded
	// true
}

func withTimer(d time.Duration, done <-chan struct{}) error {
	timer := time.NewTimer(d)
	defer timer.Stop()

	select {
	case <-done:
		return nil
	case <-timer.C:
		return errors.New("timed out")
	}
}

func main() {
	never := make(chan struct{})

	fmt.Println(withTimer(time.Millisecond, never))

	// Output:
	// timed out
}
```

## Pitfalls

- The caller gets an answer within the bound, but the operation keeps running until it notices the cancellation
  ([signal, not effect](../concepts/cooperative-cancellation.md#signal-not-effect)), at most one unit of work later
  ([latency bound](../concepts/cooperative-cancellation.md#latency-bound)).
- The bound is a lower bound plus scheduling delay ([at least its duration](../concepts/timers.md#at-least)).
- A deadline does not undo effects that already happened.
- A deadline that is too short turns slow successes into failures, and retrying them adds load.
- `time.After` in a loop creates a timer on each pass. This costs allocations, not a leak
  ([unreferenced timers are collected](../concepts/timers.md#collected)); a reused `time.Timer` or a context
  avoids it. For the same reason, in this example `defer timer.Stop()` in `withTimer` only releases the timer early.
- Forgetting to call the cancel function keeps the context's timer until it fires.

## Related pages

- [Context Propagation](context-propagation.md) carries the deadline down a call tree.
- [Close with Cause](../patterns/close-with-cause.md) records why an operation was canceled.
- [Future](../patterns/future.md) can be combined with a deadline on the receive.

## Anti-patterns

- [Ignored Context](../anti-patterns/ignored-context.md): A function takes a context and never checks it, so cancellation and deadlines have no effect.

## Sources

- [Cox-Buday (2017)](../sources.md#source-cox-buday), Ch. 5, "Timeouts and Cancellation"
- [Gerrand (2010a)](../sources.md#source-blog-timeouts)
