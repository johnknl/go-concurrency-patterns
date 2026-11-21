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

// Mutants that survive on purpose:
// - remove-critical-section condvar.go:57 (the lock around Broadcast in the AfterFunc callback): without it the
//   wake-up can fall between a waiter's ctx.Err check and its Wait, and that waiter sleeps until the next release.
//   The window is a few instructions wide; no test hits it reliably.

package condvar

import (
	"context"
	"errors"
	"sync"
	"testing"
	"testing/synctest"
)

// blocked reports whether done is still empty once every goroutine in the bubble is durably blocked; a goroutine in
// sync.Cond.Wait counts as durably blocked.
func blocked[T any](done <-chan T) bool {
	synctest.Wait()

	select {
	case <-done:
		return false
	default:
		return true
	}
}

func TestBudget(t *testing.T) {
	ctx := context.Background()

	t.Run("takes and returns units", func(t *testing.T) {
		b := newBudget(5)
		if err := b.acquire(ctx, 3); err != nil {
			t.Fatal(err)
		}
		if err := b.acquire(ctx, 2); err != nil {
			t.Fatal(err)
		}
		if b.available != 0 {
			t.Fatalf("available = %d, want 0", b.available)
		}
		b.release(5)
		if b.available != 5 {
			t.Fatalf("available = %d, want 5", b.available)
		}
	})

	t.Run("rejects more than the capacity", func(t *testing.T) {
		if err := newBudget(5).acquire(ctx, 6); !errors.Is(err, errTooLarge) {
			t.Fatalf("err = %v, want errTooLarge", err)
		}
	})

	t.Run("waits until enough is released", func(t *testing.T) {
		synctest.Test(t, func(t *testing.T) {
			b := newBudget(4)
			_ = b.acquire(ctx, 4)
			done := make(chan error)
			go func() { done <- b.acquire(ctx, 3) }()

			b.release(2)
			if !blocked(done) {
				t.Fatal("acquire(3) returned with 2 available")
			}
			b.release(1)
			if err := <-done; err != nil {
				t.Fatal(err)
			}
			if b.available != 0 {
				t.Fatalf("available = %d, want 0", b.available)
			}
		})
	})

	t.Run("one release wakes every waiter that fits", func(_ *testing.T) {
		b := newBudget(4)
		_ = b.acquire(ctx, 4)
		var wg sync.WaitGroup
		for range 4 {
			wg.Go(func() { _ = b.acquire(ctx, 1) })
		}
		b.release(4)
		wg.Wait()
	})

	t.Run("cancellation ends a wait", func(t *testing.T) {
		synctest.Test(t, func(t *testing.T) {
			b := newBudget(1)
			_ = b.acquire(ctx, 1)
			cctx, cancel := context.WithCancel(ctx)
			done := make(chan error)
			go func() { done <- b.acquire(cctx, 1) }()
			if !blocked(done) {
				t.Fatal("acquire returned without units")
			}
			cancel()
			if err := <-done; !errors.Is(err, context.Canceled) {
				t.Fatalf("err = %v, want context.Canceled", err)
			}
			if b.available != 0 {
				t.Fatalf("available = %d, want 0", b.available)
			}
		})
	})

	t.Run("an ended context does not wait", func(t *testing.T) {
		b := newBudget(1)
		_ = b.acquire(ctx, 1)
		cctx, cancel := context.WithCancel(ctx)
		cancel()
		if err := b.acquire(cctx, 1); !errors.Is(err, context.Canceled) {
			t.Fatalf("err = %v, want context.Canceled", err)
		}
	})
}

func TestWatched(t *testing.T) {
	t.Run("returns at once when the version differs", func(t *testing.T) {
		w := newWatched()
		w.set(7)
		if v, ver := w.next(0); v != 7 || ver != 1 {
			t.Fatalf("next(0) = %d, %d, want 7, 1", v, ver)
		}
	})

	t.Run("waits for a change", func(t *testing.T) {
		synctest.Test(t, func(t *testing.T) {
			w := newWatched()
			w.set(1)
			done := make(chan int)
			go func() {
				v, _ := w.next(1)
				done <- v
			}()
			if !blocked(done) {
				t.Fatal("next returned without a change")
			}
			w.set(1) // the same value is still a new version
			if v := <-done; v != 1 {
				t.Fatalf("value = %d, want 1", v)
			}
		})
	})

	t.Run("wakes every watcher", func(_ *testing.T) {
		w := newWatched()
		var wg sync.WaitGroup
		for range 5 {
			wg.Go(func() { w.next(0) })
		}
		w.set(1)
		wg.Wait()
	})
}
