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

package heartbeat

// Mutants that survive on purpose:
// - none. gremlins reports ARITHMETIC_BASE at heartbeat.go:51 as "not covered" because Go's coverage does not
//   count a select case expression; replacing n*n by n+n by hand makes TestSquare fail.

import (
	"context"
	"testing"
	"testing/synctest"
	"time"
)

// pending reports whether a pulse is waiting on heartbeat, without blocking.
func pending(heartbeat <-chan struct{}) bool {
	select {
	case _, ok := <-heartbeat:
		return ok
	default:
		return false
	}
}

func TestPulse(t *testing.T) {
	t.Run("delivers when the buffer has room", func(t *testing.T) {
		heartbeat := make(chan struct{}, 1)

		pulse(heartbeat)

		if !pending(heartbeat) {
			t.Fatal("pulse was not delivered")
		}
	})

	t.Run("drops instead of blocking when nobody took the previous pulse", func(t *testing.T) {
		heartbeat := make(chan struct{}, 1)

		pulse(heartbeat)
		pulse(heartbeat)
		pulse(heartbeat)

		if got := len(heartbeat); got != 1 {
			t.Fatalf("buffered pulses = %d, want 1", got)
		}
	})

	t.Run("drops on an unbuffered channel without a listener", func(_ *testing.T) {
		pulse(make(chan struct{}))
	})
}

func TestSquare(t *testing.T) {
	t.Run("no input closes both channels", func(t *testing.T) {
		synctest.Test(t, func(t *testing.T) {
			heartbeat, results := square(t.Context())

			if _, ok := <-results; ok {
				t.Fatal("results not closed")
			}

			if _, ok := <-heartbeat; ok {
				t.Fatal("heartbeat not closed")
			}
		})
	})

	t.Run("pulses before each unit and sends results in order", func(t *testing.T) {
		synctest.Test(t, func(t *testing.T) {
			heartbeat, results := square(t.Context(), 1, 2, 3, -4)

			for _, want := range []int{1, 4, 9, 16} {
				synctest.Wait()

				if !pending(heartbeat) {
					t.Fatalf("no pulse before result %d", want)
				}

				if got := <-results; got != want {
					t.Fatalf("result = %d, want %d", got, want)
				}
			}

			if _, ok := <-results; ok {
				t.Fatal("results not closed after the last unit")
			}
		})
	})

	t.Run("does not run ahead of the consumer", func(t *testing.T) {
		synctest.Test(t, func(t *testing.T) {
			heartbeat, results := square(t.Context(), 1, 2)

			synctest.Wait()
			<-heartbeat // pulse of unit 1

			// The worker is held at the send of result 1, so unit 2 has not started.
			synctest.Wait()

			if pending(heartbeat) {
				t.Fatal("unit 2 started before result 1 was taken")
			}

			<-results
			synctest.Wait()

			if !pending(heartbeat) {
				t.Fatal("unit 2 did not pulse after result 1 was taken")
			}

			<-results
		})
	})

	t.Run("a missing listener does not stall the work", func(t *testing.T) {
		synctest.Test(t, func(t *testing.T) {
			_, results := square(t.Context(), 1, 2, 3)

			count := 0
			for range results {
				count++
			}

			if count != 3 {
				t.Fatalf("got %d results, want 3", count)
			}
		})
	})

	t.Run("cancel ends the worker and closes both channels", func(t *testing.T) {
		synctest.Test(t, func(t *testing.T) {
			ctx, cancel := context.WithCancel(t.Context())
			heartbeat, results := square(ctx, 1, 2, 3)

			<-results
			cancel()
			synctest.Wait()

			if _, ok := <-results; ok {
				t.Fatal("results not closed after cancel")
			}

			for range heartbeat {
			}
		})
	})
}

func TestTicking(t *testing.T) {
	t.Run("pulses once per interval", func(t *testing.T) {
		synctest.Test(t, func(t *testing.T) {
			ctx, cancel := context.WithCancel(t.Context())
			defer cancel()

			heartbeat := ticking(ctx, time.Second)

			time.Sleep(time.Second - time.Nanosecond)
			synctest.Wait()

			if pending(heartbeat) {
				t.Fatal("pulse before the interval elapsed")
			}

			time.Sleep(time.Nanosecond)
			synctest.Wait()

			if !pending(heartbeat) {
				t.Fatal("no pulse after one interval")
			}
		})
	})

	t.Run("keeps one pulse while nobody listens", func(t *testing.T) {
		synctest.Test(t, func(t *testing.T) {
			ctx, cancel := context.WithCancel(t.Context())
			defer cancel()

			heartbeat := ticking(ctx, time.Second)

			time.Sleep(10 * time.Second)
			synctest.Wait()

			if got := len(heartbeat); got != 1 {
				t.Fatalf("buffered pulses = %d, want 1", got)
			}
		})
	})

	t.Run("cancel closes the heartbeat", func(t *testing.T) {
		synctest.Test(t, func(t *testing.T) {
			ctx, cancel := context.WithCancel(t.Context())
			heartbeat := ticking(ctx, time.Second)

			cancel()

			if _, ok := <-heartbeat; ok {
				t.Fatal("heartbeat not closed after cancel")
			}
		})
	})
}
