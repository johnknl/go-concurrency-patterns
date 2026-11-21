---
title: Condition Variable
intent: Let goroutines wait until a condition on shared state holds, and wake them when it may have changed.
provides: [sync.Cond, sync.NewCond, context.AfterFunc]
package: stdlib/condvar
concerns: [data-race-safety]
sources:
  - id: hoare-1974
  - id: lampson-redell-1980
  - id: go-sync
concepts:
  - overload-policy: 'the budget waits for capacity, rejects what can never fit, and the wait stays cancelable.'
---
# Condition Variable

<!-- meta -->
Let goroutines wait until a condition on shared state holds, and wake them when it may have changed.

<div class="cp-props" markdown>

| | |
|---|---|
| Provides | `sync.Cond`, `sync.NewCond`, `context.AfterFunc` |
| Concerns | [Data race safety](../catalog/concern.md#concern-data-race-safety) |
| Concepts | [Overload Policy](../concepts/overload-policy.md) |
| Code | [`stdlib/condvar`](https://github.com/johnknl/go-concurrency-patterns/tree/main/stdlib/condvar) |

</div>
<!-- /meta -->

## What it provides

`sync.Cond` is tied to a `Locker`. `Wait` atomically unlocks it and suspends the goroutine; when woken it locks
again before returning. `Signal` wakes one waiter, `Broadcast` all of them. Go's `Cond` has Mesa semantics: a woken
goroutine does not run at once, and by the time it holds the lock again the state may have changed.

Reach for it when waiters wait for different predicates over state already guarded by a mutex, or for a change that
repeats. A channel wakes one receiver per send, or every receiver once when closed, which fits neither case. For a
single hand-off, a bounded buffer, or a one-time event, use a channel: a send, a buffered channel
([Bounded Queue](../patterns/bounded-queue.md)), or a `close`. The `sync` documentation recommends channels for
these simple cases.

## Using it

Wait in a loop, `for !condition { c.Wait() }`, and change the state with the lock held.

**Waiters with different conditions.** In this example a budget hands out units of a shared capacity, such as bytes
of memory, and each caller waits until enough units for its own request are free. A release may satisfy some
waiters and not others, so it calls `Broadcast` and every waiter rechecks its own condition. A request larger than
the capacity is rejected, since it would wait forever: the limit has a policy, wait or reject
([every limit needs a policy](../concepts/overload-policy.md#every-limit)).

<!-- code: stdlib/condvar#budget stdlib/condvar#newBudget stdlib/condvar#budget.acquire stdlib/condvar#budget.release stdlib/condvar#Example_budget:main -->
```go
type budget struct {
	cond      *sync.Cond
	available int
	capacity  int
	mu        sync.Mutex
}

func newBudget(capacity int) *budget {
	b := &budget{available: capacity, capacity: capacity}
	b.cond = sync.NewCond(&b.mu)

	return b
}

func (b *budget) acquire(ctx context.Context, n int) error {
	if n > b.capacity {
		return errTooLarge
	}

	// Wait cannot select on ctx.Done, so cancellation wakes every waiter, and each checks its context.
	stop := context.AfterFunc(ctx, func() {
		b.mu.Lock()
		defer b.mu.Unlock()

		b.cond.Broadcast()
	})
	defer stop()

	b.mu.Lock()
	defer b.mu.Unlock()

	for b.available < n {
		if ctx.Err() != nil {
			return context.Cause(ctx)
		}

		b.cond.Wait()
	}

	b.available -= n

	return nil
}

func (b *budget) release(n int) {
	b.mu.Lock()
	b.available += n
	b.mu.Unlock()

	b.cond.Broadcast()
}

func main() {
	ctx := context.Background()
	b := newBudget(9)
	_ = b.acquire(ctx, 9) // everything is in use

	got := make(chan int)
	for _, n := range []int{6, 3} {
		go func() {
			if b.acquire(ctx, n) == nil {
				got <- n
			}
		}()
	}

	b.release(4) // enough for 3, not for 6
	fmt.Println("got", <-got)

	b.release(5) // 1 + 5 = 6
	fmt.Println("got", <-got)

	// Output:
	// got 3
	// got 6
}
```
<!-- /code -->

**Cancellation.** `Wait` cannot select on a context. `context.AfterFunc` closes the gap: when the context ends, it
takes the lock and calls `Broadcast`, and the waiter returns the context's cause on its next check. Taking the lock
matters: without it the wake-up can fall between a waiter's check of the context and its `Wait`, and be lost.

<!-- code: stdlib/condvar#Example_cancel:main -->
```go
func main() {
	b := newBudget(1)
	_ = b.acquire(context.Background(), 1)

	ctx, cancel := context.WithCancelCause(context.Background())
	done := make(chan error)

	go func() { done <- b.acquire(ctx, 1) }()

	cancel(fmt.Errorf("caller gave up"))
	fmt.Println(<-done)

	// Output:
	// caller gave up
}
```
<!-- /code -->

**Repeated broadcast.** A value changes many times and watchers wait for the next change. A version counter in the
locked state tells a watcher whether it has seen the current value; every change increments it and calls
`Broadcast`. With channels, every change has to close the current channel and install a new one. A watcher that is
slow sees only the latest value, not every one.

<!-- code: stdlib/condvar#watched stdlib/condvar#newWatched stdlib/condvar#watched.set stdlib/condvar#watched.next stdlib/condvar#Example_watch:main -->
```go
type watched struct {
	cond    *sync.Cond
	value   int
	version uint64
	mu      sync.Mutex
}

func newWatched() *watched {
	w := &watched{}
	w.cond = sync.NewCond(&w.mu)

	return w
}

func (w *watched) set(v int) {
	w.mu.Lock()
	w.value = v
	w.version++
	w.mu.Unlock()

	w.cond.Broadcast()
}

func (w *watched) next(seen uint64) (int, uint64) {
	w.mu.Lock()
	defer w.mu.Unlock()

	for w.version == seen {
		w.cond.Wait()
	}

	return w.value, w.version
}

func main() {
	w := newWatched()
	ack := make(chan struct{})

	go func() {
		var seen uint64
		for range 3 {
			var v int
			v, seen = w.next(seen)
			fmt.Println("saw", v)
			ack <- struct{}{}
		}
	}()

	for v := range 3 {
		w.set(v + 1)
		<-ack
	}

	// Output:
	// saw 1
	// saw 2
	// saw 3
}
```
<!-- /code -->

## Pitfalls

- Waiting with `if` instead of `for`: with Mesa semantics the condition may no longer hold when `Wait` returns.
- Changing the state without the lock: a waiter can check the condition, miss the change and the signal, and sleep
  forever. `Signal` and `Broadcast` themselves do not require the lock.
- With different conditions on one `Cond`, `Signal` can wake a waiter whose condition still does not hold while
  another that could proceed sleeps on. Use `Broadcast`, or one `Cond` per condition.
- `Broadcast` wakes every waiter to recheck, even those that still cannot proceed; with many waiters and frequent
  releases this costs CPU. Waiting is not first-come, first-served: a small request can overtake a large one
  indefinitely. `golang.org/x/sync/semaphore` keeps waiters in order with a channel per waiter.
- `Wait` ignores contexts; without a wake-up from `context.AfterFunc`, a canceled waiter sleeps on, though
  [waiting must stay cancelable](../concepts/overload-policy.md#wait-cancelable).
- A `Cond` must not be copied after first use.

## Related pages

- [Monitor](../patterns/monitor.md) is the mutex-guarded state a condition variable waits on.
- [Group Commit](../patterns/group-commit.md) uses one `Cond` for several waits that change together.
- [Resource Pool](../patterns/resource-pool.md) waits for a free resource with a `Cond` in its mutex variant.
- [Semaphore](../patterns/semaphore.md) is the channel form when every caller needs one unit.

## Sources

<!-- sources -->
- [Hoare (1974)](../sources.md#source-hoare-1974)
- [Lampson and Redell (1980)](../sources.md#source-lampson-redell-1980)
- [The Go Authors (n.d.e)](../sources.md#source-go-sync)
<!-- /sources -->
