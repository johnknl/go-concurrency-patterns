---
title: Synctest
aliases: ['testing/synctest', 'Fake time in tests']
intent: Test concurrent and time-dependent code deterministically, with fake time and a way to wait until every goroutine is blocked.
provides: [testing/synctest.Test, testing/synctest.Wait, testing/synctest.Sleep]
package: stdlib/bubble
concerns: [determinism, leak-prevention]
sources:
  - id: go-testing-synctest
concepts:
  - determinism: 'a bubble removes time and scheduling as sources of flaky test results.'
  - timers: 'timers and tickers in a bubble run on a fake clock.'
---
# Synctest

<!-- meta -->
Test concurrent and time-dependent code deterministically, with fake time and a way to wait until every goroutine is blocked.

<div class="cp-props" markdown>

| | |
|---|---|
| Also known as | testing/synctest, Fake time in tests |
| Provides | `testing/synctest.Test`, `testing/synctest.Wait`, `testing/synctest.Sleep` |
| Concerns | [Determinism](../catalog/concern.md#concern-determinism), [Leak prevention](../catalog/concern.md#concern-leak-prevention) |
| Concepts | [Determinism](../concepts/determinism.md), [Timers and Tickers](../concepts/timers.md) |
| Code | [`stdlib/bubble`](https://github.com/johnknl/go-concurrency-patterns/tree/main/stdlib/bubble) |

</div>
<!-- /meta -->

## What it provides

`synctest.Test(t, f)` runs `f` in a *bubble*: goroutines started inside belong to it, and channels, timers and tickers
created inside are associated with it. In the bubble the `time` package uses a fake clock that starts at midnight UTC
2000-01-01 and [advances only when every goroutine in the bubble is durably blocked](../concepts/timers.md#synctest),
jumping to the next time that unblocks one. A minute-long timeout takes no real time, and the time it measures is
exact. `Test` waits for all goroutines of the bubble to exit, and fails the test when they deadlock.

`synctest.Wait()` blocks until every other goroutine in the bubble is durably blocked. After it returns, the test can
read what those goroutines wrote without a race: they are all parked. `synctest.Sleep(d)` (Go 1.27) is
`time.Sleep(d)` followed by `synctest.Wait()`. `Test` and `Wait` exist since Go 1.25. The claims on this page are
checked against the package documentation for Go 1.27.

A goroutine is *durably blocked* when only another goroutine in the same bubble can unblock it. In Go 1.27 these
operations durably block:

- a blocking send or receive on a channel created in the bubble;
- a blocking `select` whose cases are all channels created in the bubble;
- `sync.Cond.Wait`;
- `sync.WaitGroup.Wait`, when `Add` was called in the bubble;
- `time.Sleep`.

Nothing else does. In particular, locking a `sync.Mutex` or `sync.RWMutex`, blocking on I/O such as a network read,
and system calls block a goroutine but not durably, because an event outside the bubble could release it.

This makes a bubble a tool to [reproduce](../concepts/determinism.md#reproduce) timing behavior: time and the
moment the test looks are no longer chosen by the scheduler.

## Using it

**A timeout without waiting.** `receive` waits for a value for at most a duration. In the bubble, nobody sends, every
goroutine is blocked, and the clock jumps to the timer: the test checks that it waited exactly one minute, and returns
at once.

<!-- code: stdlib/bubble#receive stdlib/bubble#TestTimeout -->
```go
func receive(c <-chan int, d time.Duration) (int, error) {
	timer := time.NewTimer(d)
	defer timer.Stop()

	select {
	case v := <-c:
		return v, nil
	case <-timer.C:
		return 0, errTimeout
	}
}

func TestTimeout(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		start := time.Now()

		_, err := receive(make(chan int), time.Minute) // nobody sends

		if !errors.Is(err, errTimeout) {
			t.Fatalf("err = %v, want errTimeout", err)
		}
		if waited := time.Since(start); waited != time.Minute {
			t.Fatalf("waited %v, want exactly 1m", waited)
		}
	})
}
```
<!-- /code -->

**Observing a blocked goroutine.** `synctest.Wait` returns when `sum` is parked on its next receive, so it has
processed everything sent so far, and the test reads the total without a sleep or a lock.

<!-- code: stdlib/bubble#sum stdlib/bubble#TestWait -->
```go
func sum(in <-chan int, total *int) {
	for v := range in {
		*total += v
	}
}

func TestWait(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		in := make(chan int)
		total := 0
		done := make(chan struct{})

		go func() {
			sum(in, &total)
			close(done)
		}()

		in <- 1
		in <- 2
		synctest.Wait() // sum is durably blocked on its next receive, so it has added 2

		if total != 3 {
			t.Fatalf("total = %d, want 3", total)
		}

		close(in)
		<-done
	})
}
```
<!-- /code -->

**Waiting on a condition.** `sync.Cond.Wait` is durably blocking, so a goroutine that waits on a condition can be
observed before and after the condition changes. A goroutine blocked in `Mutex.Lock` cannot (see below).

<!-- code: stdlib/bubble#gate.wait stdlib/bubble#gate.release stdlib/bubble#TestCondWait -->
```go
func (g *gate) wait() {
	g.mu.Lock()
	defer g.mu.Unlock()

	for !g.open {
		g.cond.Wait()
	}
}

func (g *gate) release() {
	g.mu.Lock()
	g.open = true
	g.mu.Unlock()

	g.cond.Broadcast()
}

func TestCondWait(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		g := newGate()
		passed := false

		go func() {
			g.wait()
			passed = true
		}()

		synctest.Wait() // the goroutine is in Cond.Wait, which is durably blocking
		if passed {
			t.Fatal("passed a closed gate")
		}

		g.release()
		synctest.Wait()

		if !passed {
			t.Fatal("did not pass an open gate")
		}
	})
}
```
<!-- /code -->

**Settling after sleep.** When the test and the code under test wake at the same instant, which of them runs first is
not defined. `synctest.Sleep` sleeps and then waits until the others are blocked again, so the ticker's third tick
has been counted.

<!-- code: stdlib/bubble#TestSleep -->
```go
func TestSleep(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		ticks := 0
		ticker := time.NewTicker(time.Second)
		stop := make(chan struct{})
		done := make(chan struct{})

		go func() {
			defer close(done)
			defer ticker.Stop()

			for {
				select {
				case <-ticker.C:
					ticks++
				case <-stop:
					return
				}
			}
		}()

		// The ticker fires at 3s, the same instant this sleep ends; synctest.Sleep also waits for the goroutine.
		synctest.Sleep(3 * time.Second)

		if ticks != 3 {
			t.Fatalf("ticks = %d, want 3", ticks)
		}

		close(stop)
		<-done
	})
}
```
<!-- /code -->

## Pitfalls

- **A goroutine blocked on a mutex is not durably blocked.** `synctest.Wait` does not return while another goroutine
  waits in `Lock`, and time does not advance; if the test itself holds that lock, it hangs instead of failing as a
  deadlock. We hit this in the test of [Lock Held While Blocking](../anti-patterns/lock-held-while-blocking.md): the
  test cannot wait for `Publish` to block on the lock, so it holds the lock while it starts `Publish`, reads the
  result from a channel, and relies on the race detector to catch a read without the lock. Test such code without
  `Wait` at that point, or wait on a channel or `sync.Cond` instead.
  `sync.Cond.Wait` itself is durably blocking; waking a goroutine of a bubble that is blocked in `Cond.Wait` from
  outside the bubble is a fatal error.
- **Channels created outside the bubble.** A goroutine blocked on such a channel is not durably blocked, so it keeps
  `Wait` waiting and the clock still. A `select` is durable only when every case is a bubble channel. In the other
  direction, operating on a bubble's channel, timer or ticker from outside the bubble panics. Create the channels
  the code uses inside the `f` passed to `Test`.
- **A `sync.WaitGroup` package variable.** A `var wg sync.WaitGroup` at package level cannot be associated with a
  bubble, and waiting on it may not be durably blocking; a `*WaitGroup` in a package variable is fine.
- **Real I/O and the network.** Blocking I/O and system calls are not durable, so a goroutine reading from a socket
  keeps the bubble from going idle. Use an in-process fake such as `net.Pipe`, not a loopback connection. Avoid
  external processes and goroutines not started by the test.
- **Leftover goroutines.** `Test` waits for every goroutine of the bubble to exit, and fails on a deadlock, so a
  goroutine the test forgets to stop shows up as a failure; stop it before `f` returns. Time stops advancing once
  the root goroutine of the bubble has exited.
- **Restrictions on `t`.** In the bubble, `T.Run`, `T.Parallel` and `T.Deadline` must not be called, `Test` must not
  be nested, and `Wait` must not be called from several goroutines of a bubble at once. `T.Cleanup` functions run
  inside the bubble, and `T.Context` is canceled by a bubble channel.

## Related pages

- [Determinism](../concepts/determinism.md) lists the other ways to reproduce a concurrency failure.
- [Timers](../concepts/timers.md) explains the timers that the fake clock drives.
- [Token Bucket](../patterns/token-bucket.md), [Batcher](../patterns/batcher.md) and [Pacing](../patterns/pacing.md)
  test their timing in a bubble.
- [Needless Asynchrony](../anti-patterns/needless-asynchrony.md) uses `synctest.Wait` to see a goroutine blocked on
  its send.
- [Lock Held While Blocking](../anti-patterns/lock-held-while-blocking.md) has the test that cannot wait on a mutex.
- [Condition Variable](condition-variable.md) covers `sync.Cond`, which a bubble can wait on.

## Sources

<!-- sources -->
- [The Go Authors (n.d.g)](../sources.md#source-go-testing-synctest)
<!-- /sources -->
