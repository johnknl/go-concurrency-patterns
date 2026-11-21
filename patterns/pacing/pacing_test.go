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

package pacing

import (
	"context"
	"errors"
	"sync"
	"testing"
	"testing/synctest"
	"time"
)

func TestPace(t *testing.T) {
	t.Run("spaces calls by the ticker interval", func(t *testing.T) {
		synctest.Test(t, func(t *testing.T) {
			ticker := time.NewTicker(time.Second)
			defer ticker.Stop()

			start := time.Now()

			for range 3 {
				if err := pace(t.Context(), ticker.C); err != nil {
					t.Fatal(err)
				}
			}

			if got := time.Since(start); got != 3*time.Second {
				t.Fatalf("3 calls took %v, want 3s", got)
			}
		})
	})

	t.Run("after an idle period one call goes through at once", func(t *testing.T) {
		synctest.Test(t, func(t *testing.T) {
			ticker := time.NewTicker(time.Second)
			defer ticker.Stop()

			time.Sleep(10 * time.Second)
			start := time.Now()

			for range 3 {
				if err := pace(t.Context(), ticker.C); err != nil {
					t.Fatal(err)
				}
			}

			if got := time.Since(start); got != 2*time.Second {
				t.Fatalf("3 calls after idling took %v, want 2s", got)
			}
		})
	})

	t.Run("returns the cause when ctx ends first", func(t *testing.T) {
		synctest.Test(t, func(t *testing.T) {
			cause := errors.New("shutting down")
			ctx, cancel := context.WithCancelCause(t.Context())

			done := make(chan error)
			go func() { done <- pace(ctx, make(chan time.Time)) }()

			synctest.Wait()
			cancel(cause)

			if err := <-done; !errors.Is(err, cause) {
				t.Fatalf("err = %v, want %v", err, cause)
			}
		})
	})
}

func TestPacer(t *testing.T) {
	t.Run("first call goes at once, the rest one interval apart", func(t *testing.T) {
		synctest.Test(t, func(t *testing.T) {
			p := &pacer{interval: time.Second}
			start := time.Now()

			var (
				mu    sync.Mutex
				times []time.Duration
				wg    sync.WaitGroup
			)

			for range 4 {
				wg.Go(func() {
					if err := p.wait(t.Context()); err != nil {
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

	t.Run("idle time does not build up a burst", func(t *testing.T) {
		synctest.Test(t, func(t *testing.T) {
			p := &pacer{interval: time.Second}
			_ = p.wait(t.Context())

			time.Sleep(10 * time.Second)
			start := time.Now()

			for range 3 {
				_ = p.wait(t.Context())
			}

			if got := time.Since(start); got != 2*time.Second {
				t.Fatalf("3 calls after idling took %v, want 2s", got)
			}
		})
	})

	t.Run("a call due now wins over an ended ctx", func(t *testing.T) {
		synctest.Test(t, func(t *testing.T) {
			ctx, cancel := context.WithCancel(t.Context())
			cancel()

			for range 20 {
				p := &pacer{interval: time.Second}

				if err := p.wait(ctx); err != nil {
					t.Fatalf("err = %v, want nil", err)
				}
			}
		})
	})

	t.Run("a canceled wait returns the cause and keeps its slot", func(t *testing.T) {
		synctest.Test(t, func(t *testing.T) {
			p := &pacer{interval: time.Second}
			_ = p.wait(t.Context())

			cause := errors.New("shutting down")
			ctx, cancel := context.WithCancelCause(t.Context())

			done := make(chan error)
			go func() { done <- p.wait(ctx) }()

			synctest.Wait()
			cancel(cause)

			if err := <-done; !errors.Is(err, cause) {
				t.Fatalf("err = %v, want %v", err, cause)
			}

			start := time.Now()
			_ = p.wait(t.Context())

			if got := time.Since(start); got != 2*time.Second {
				t.Fatalf("next call waited %v, want 2s", got)
			}
		})
	})
}
