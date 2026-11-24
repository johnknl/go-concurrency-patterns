# Token Bucket

Limit the average rate of an operation while allowing a bounded burst.

- **Also known as:** Throttling, Rate limiter
- **Purpose:** Coordination
- **Scale:** Pattern
- **Mechanisms:** Sync, Time
- **Concerns:** Rate control
- **Concepts:** [Timers and Tickers](../concepts/timers.md)
- **Code:** [`patterns/tokenbucket`](https://github.com/johnknl/go-concurrency-patterns/tree/main/patterns/tokenbucket)

> **Library.**
> Production code usually uses [`golang.org/x/time/rate`](https://pkg.go.dev/golang.org/x/time/rate), whose `Limiter` is the token bucket on this page with reservations, waiting, and a changeable rate.

## Problem

An operation must not start more often than a given average rate, but a caller that has been idle may start a few
at once. Limiting how many calls run at the same time does not help, because fast calls can still arrive in large
numbers; the limit is on how often calls start.

## Mechanism

Each call spends a token, tokens are earned at a fixed rate, and the bucket holds at most a burst of them. The bucket
is a [Monitor](monitor.md): a mutex, the number of tokens, and the time of the last update, with no goroutine and no
ticker. Each call first adds the tokens earned since the last update, `elapsed × rate`, capped at the burst. `allow`
then takes a token if there is one. `wait` always takes a token, which may leave the count negative: the deficit is
the amount still to be earned, so the delay is `-tokens / rate`. Reserving before waiting queues concurrent callers,
each waiting one token longer than the one before, and a waiting caller holds a timer, not the lock. If the context
ends first, the caller adds its token back.

```go
// bucket holds up to burst tokens and gains rate tokens per second. It has no goroutine: the tokens earned since
// the last call are added when the next call arrives. A negative token count means callers have reserved tokens
// that are still being earned.
type bucket struct {
	last   time.Time
	tokens float64
	rate   float64
	burst  float64
	mu     sync.Mutex
}

// refill adds the tokens earned since the last call, up to burst. The caller holds mu.
func (b *bucket) refill() {
	now := time.Now()
	b.tokens = min(b.burst, b.tokens+now.Sub(b.last).Seconds()*b.rate)
	b.last = now
}

func (b *bucket) allow() bool {
	b.mu.Lock()
	defer b.mu.Unlock()

	b.refill()

	if b.tokens < 1 {
		return false
	}

	b.tokens--

	return true
}

func (b *bucket) wait(ctx context.Context) error {
	b.mu.Lock()
	b.refill()
	b.tokens--
	delay := time.Duration(-b.tokens / b.rate * float64(time.Second))
	b.mu.Unlock()

	if delay <= 0 {
		return nil
	}

	timer := time.NewTimer(delay)
	defer timer.Stop()

	select {
	case <-timer.C:
		return nil
	case <-ctx.Done():
		b.mu.Lock()
		b.refill()
		b.tokens++
		b.mu.Unlock()

		return context.Cause(ctx)
	}
}

func main() {
	b := newBucket(1.0/3600, 3) // one token per hour

	for i := 1; i <= 4; i++ {
		fmt.Println("request", i, b.allow())
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	fmt.Println("request 5", b.wait(ctx))

	// 1000 tokens per second, burst 1: the second wait lasts about a millisecond.
	fast := newBucket(1000, 1)
	start := time.Now()
	_ = fast.wait(context.Background())
	_ = fast.wait(context.Background())
	fmt.Println("waited about 1ms:", time.Since(start) >= 999*time.Microsecond)

	// Output:
	// request 1 true
	// request 2 true
	// request 3 true
	// request 4 false
	// request 5 context canceled
	// waited about 1ms: true
}
```

The tests run inside `testing/synctest`, so they check exact times without injecting a clock
([timers](../concepts/timers.md#synctest)).

## Applicability

Use it when a downstream service enforces an average rate and tolerates short bursts, or when many goroutines must
share one rate. The `Limiter` of `golang.org/x/time/rate` adds `Reserve`, `WaitN`, a changeable rate, and an
immediate error when a context deadline would pass before the token is earned. To spread calls evenly with no burst,
use [Pacing](pacing.md).

## Consequences

- Burst and sustained rate are set separately. With a burst of 1 the bucket behaves like [Pacing](pacing.md).
- The bucket starts full, so a burst is allowed at once; start with zero tokens when that is not wanted.
- Idle time earns at most a burst; the limit over any span `t` is `burst + rate × t`.
- No goroutine has to be stopped, and an idle bucket does no work.
- `wait` serves callers in the order they reserved. A canceled reservation returns its token, but callers that
  reserved after it still wait as long as before; the token goes to the next caller that arrives.
- `wait` with a context that has already ended still returns `nil` if a token is available at once.
- The rate must be positive: with a rate of zero, `wait` computes an infinite delay.

## Variants

### Channel of tokens

The bucket is a buffered channel whose capacity is the burst. A goroutine adds a token on each tick of a `time.Ticker`
and drops it when the channel is full; callers receive a token in a `select` with the context. In this example, the
ticks come from a channel that the example sends on itself. This costs a goroutine to own and stop, refill work while
nobody calls, and a rate no finer than the tick, with dropped ticks when the goroutine falls behind
([a ticker holds one pending tick](../concepts/timers.md#ticker-drops)); see [Channel Where a Mutex Would
Do](../anti-patterns/channel-instead-of-mutex.md).

```go
// newChanBucket returns a token bucket that holds up to burst tokens and starts full. A goroutine adds one token
// for every value received from tick, and drops it when the bucket is full.
func newChanBucket(ctx context.Context, burst int, tick <-chan time.Time) <-chan struct{} {
	tokens := make(chan struct{}, burst)

	for range burst {
		tokens <- struct{}{}
	}

	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case <-tick:
				select {
				case tokens <- struct{}{}:
				default:
				}
			}
		}
	}()

	return tokens
}

func take(ctx context.Context, tokens <-chan struct{}) error {
	select {
	case <-tokens:
		return nil
	case <-ctx.Done():
		return context.Cause(ctx)
	}
}

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	tick := make(chan time.Time)
	tokens := newChanBucket(ctx, 3, tick)

	for i := 1; i <= 3; i++ {
		fmt.Println("request", i, take(ctx, tokens))
	}

	// The bucket is empty, so only the canceled context is ready.
	canceled, stop := context.WithCancel(ctx)
	stop()
	fmt.Println("request 4", take(canceled, tokens))

	tick <- time.Time{}
	fmt.Println("request 5", take(ctx, tokens))

	// Output:
	// request 1 <nil>
	// request 2 <nil>
	// request 3 <nil>
	// request 4 context canceled
	// request 5 <nil>
}
```

## Related patterns

- [Pacing](pacing.md) spaces calls one interval apart, with no burst.
- [Semaphore](semaphore.md) limits how many operations run at the same time; a token bucket limits how often they
  start.
- [Worker Pool](worker-pool.md) bounds concurrency; the workers can take a token before each job.

## Sources

- [Cox-Buday (2017)](../sources.md#source-cox-buday), Ch. 5, "Rate Limiting"
- [The Go Authors (n.d.d)](../sources.md#source-golang-x-time-rate)
