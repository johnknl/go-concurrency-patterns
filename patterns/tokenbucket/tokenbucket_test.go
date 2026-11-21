// MIT License
//
// Copyright (C) 2026 John Kleijn
//
// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to deal
// in the Software without restriction, including without limitation the rights
// to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
// copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in all
// copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
//

// Mutants that survive on purpose: none.

package tokenbucket

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"testing/synctest"
	"time"
)

func TestAllow(t *testing.T) {
	t.Run("starts full and refuses beyond the burst", func(t *testing.T) {
		synctest.Test(t, func(t *testing.T) {
			b := newBucket(1, 3)

			for i := range 3 {
				if !b.allow() {
					t.Fatalf("request %d refused within the burst", i+1)
				}
			}

			if b.allow() {
				t.Fatal("request beyond the burst allowed")
			}
		})
	})

	t.Run("earns tokens at the rate", func(t *testing.T) {
		synctest.Test(t, func(t *testing.T) {
			b := newBucket(2, 1)
			b.allow()

			time.Sleep(499 * time.Millisecond)

			if b.allow() {
				t.Fatal("token allowed before it was earned")
			}

			time.Sleep(time.Millisecond)

			if !b.allow() {
				t.Fatal("earned token refused")
			}
		})
	})

	t.Run("refill is capped at the burst", func(t *testing.T) {
		synctest.Test(t, func(t *testing.T) {
			b := newBucket(1, 2)

			time.Sleep(time.Hour)

			for range 2 {
				if !b.allow() {
					t.Fatal("refused within the burst")
				}
			}

			if b.allow() {
				t.Fatal("idle time earned more than the burst")
			}
		})
	})

	t.Run("concurrent callers share the burst", func(t *testing.T) {
		synctest.Test(t, func(t *testing.T) {
			b := newBucket(1, 50)

			var (
				allowed atomic.Int64
				wg      sync.WaitGroup
			)

			for range 100 {
				wg.Go(func() {
					if b.allow() {
						allowed.Add(1)
					}
				})
			}

			wg.Wait()

			if got := allowed.Load(); got != 50 {
				t.Fatalf("allowed = %d, want 50", got)
			}
		})
	})
}

func TestWait(t *testing.T) {
	t.Run("an available token wins over an ended ctx", func(t *testing.T) {
		synctest.Test(t, func(t *testing.T) {
			ctx, cancel := context.WithCancel(t.Context())
			cancel()

			for range 20 {
				b := newBucket(1, 1)

				if err := b.wait(ctx); err != nil {
					t.Fatalf("err = %v, want nil", err)
				}
			}
		})
	})

	t.Run("concurrent cancellations all return their tokens", func(t *testing.T) {
		synctest.Test(t, func(t *testing.T) {
			b := newBucket(1, 1)
			b.allow()

			ctx, cancel := context.WithCancel(t.Context())

			var wg sync.WaitGroup

			for range 10 {
				wg.Go(func() { _ = b.wait(ctx) })
			}

			synctest.Wait()
			cancel()
			wg.Wait()

			time.Sleep(time.Second)

			if !b.allow() {
				t.Fatal("tokens of the canceled waits not returned")
			}
		})
	})
	t.Run("returns at once while tokens are available", func(t *testing.T) {
		synctest.Test(t, func(t *testing.T) {
			b := newBucket(1, 2)
			start := time.Now()

			for range 2 {
				if err := b.wait(t.Context()); err != nil {
					t.Fatal(err)
				}
			}

			if d := time.Since(start); d != 0 {
				t.Fatalf("waited %v within the burst", d)
			}
		})
	})

	t.Run("concurrent callers are spaced by the rate", func(t *testing.T) {
		synctest.Test(t, func(t *testing.T) {
			b := newBucket(1, 1)
			start := time.Now()

			var (
				mu    sync.Mutex
				times []time.Duration
				wg    sync.WaitGroup
			)

			for range 4 {
				wg.Go(func() {
					if err := b.wait(t.Context()); err != nil {
						t.Error(err)
					}

					mu.Lock()
					times = append(times, time.Since(start))
					mu.Unlock()
				})
			}

			wg.Wait()

			for i, want := range []time.Duration{0, time.Second, 2 * time.Second, 3 * time.Second} {
				if times[i] != want {
					t.Fatalf("times = %v, want 0s 1s 2s 3s", times)
				}
			}
		})
	})

	t.Run("a canceled wait returns its token and the cause", func(t *testing.T) {
		synctest.Test(t, func(t *testing.T) {
			b := newBucket(1, 1)
			b.allow()

			cause := errors.New("shutting down")
			ctx, cancel := context.WithCancelCause(t.Context())

			done := make(chan error)
			go func() { done <- b.wait(ctx) }()

			time.Sleep(500 * time.Millisecond)
			cancel(cause)

			if err := <-done; !errors.Is(err, cause) {
				t.Fatalf("err = %v, want %v", err, cause)
			}

			time.Sleep(499 * time.Millisecond)

			if b.allow() {
				t.Fatal("token allowed before it was earned")
			}

			time.Sleep(time.Millisecond)

			if !b.allow() {
				t.Fatal("token of the canceled wait not returned")
			}
		})
	})
}

func TestChanBucket(t *testing.T) {
	t.Run("starts full", func(t *testing.T) {
		synctest.Test(t, func(t *testing.T) {
			tokens := newChanBucket(t.Context(), 3, nil)

			if got := len(tokens); got != 3 {
				t.Fatalf("tokens = %d, want 3", got)
			}
		})
	})

	t.Run("one tick adds one token", func(t *testing.T) {
		synctest.Test(t, func(t *testing.T) {
			tick := make(chan time.Time)
			tokens := newChanBucket(t.Context(), 3, tick)

			for range 3 {
				<-tokens
			}

			tick <- time.Time{}
			synctest.Wait()

			if got := len(tokens); got != 1 {
				t.Fatalf("tokens = %d, want 1", got)
			}
		})
	})

	t.Run("ticks on a full bucket are dropped", func(t *testing.T) {
		synctest.Test(t, func(t *testing.T) {
			tick := make(chan time.Time)
			tokens := newChanBucket(t.Context(), 2, tick)

			tick <- time.Time{}
			tick <- time.Time{}
			synctest.Wait()

			<-tokens
			<-tokens
			synctest.Wait()

			if got := len(tokens); got != 0 {
				t.Fatalf("tokens after draining a full bucket = %d, want 0", got)
			}
		})
	})

	t.Run("refills from a ticker", func(t *testing.T) {
		synctest.Test(t, func(t *testing.T) {
			ticker := time.NewTicker(time.Second)
			defer ticker.Stop()

			tokens := newChanBucket(t.Context(), 1, ticker.C)
			start := time.Now()

			for range 4 {
				if err := take(t.Context(), tokens); err != nil {
					t.Fatal(err)
				}
			}

			if got := time.Since(start); got != 3*time.Second {
				t.Fatalf("4 requests with burst 1 took %v, want 3s", got)
			}
		})
	})

	t.Run("cancel stops the refill goroutine", func(t *testing.T) {
		synctest.Test(t, func(t *testing.T) {
			ctx, cancel := context.WithCancel(t.Context())
			newChanBucket(ctx, 1, make(chan time.Time))
			cancel()
			synctest.Wait()
		})
	})

	t.Run("take returns the cause when ctx ends first", func(t *testing.T) {
		synctest.Test(t, func(t *testing.T) {
			cause := errors.New("shutting down")
			ctx, cancel := context.WithCancelCause(t.Context())

			done := make(chan error)
			go func() { done <- take(ctx, make(chan struct{})) }()

			synctest.Wait()
			cancel(cause)

			if err := <-done; !errors.Is(err, cause) {
				t.Fatalf("err = %v, want %v", err, cause)
			}
		})
	})
}
