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

package timedsend

// Mutants that survive on purpose: none.

import (
	"context"
	"testing"
	"testing/synctest"
	"time"
)

func TestTimedSend(t *testing.T) {
	t.Run("fast path sends into a free slot without waiting", func(t *testing.T) {
		synctest.Test(t, func(t *testing.T) {
			ch := make(chan int, 1)
			start := time.Now()

			if !timedSend(t.Context(), ch, 1, newStoppedTimer(), time.Hour) {
				t.Fatal("send failed")
			}

			if time.Since(start) != 0 {
				t.Fatal("fast path waited")
			}

			if got := <-ch; got != 1 {
				t.Fatalf("received %d, want 1", got)
			}
		})
	})

	t.Run("gives up after exactly d", func(t *testing.T) {
		synctest.Test(t, func(t *testing.T) {
			start := time.Now()

			if timedSend(t.Context(), make(chan int), 1, newStoppedTimer(), time.Second) {
				t.Fatal("send reported success without a receiver")
			}

			if got := time.Since(start); got != time.Second {
				t.Fatalf("waited %v, want 1s", got)
			}
		})
	})

	t.Run("a receiver arriving within d takes the value", func(t *testing.T) {
		synctest.Test(t, func(t *testing.T) {
			ch := make(chan int)
			got := make(chan int)

			go func() {
				time.Sleep(time.Second - time.Nanosecond)
				got <- <-ch
			}()

			if !timedSend(t.Context(), ch, 5, newStoppedTimer(), time.Second) {
				t.Fatal("send failed")
			}

			if v := <-got; v != 5 {
				t.Fatalf("received %d, want 5", v)
			}
		})
	})

	t.Run("cancel during the wait ends it", func(t *testing.T) {
		synctest.Test(t, func(t *testing.T) {
			ctx, cancel := context.WithCancel(t.Context())
			done := make(chan bool)

			go func() { done <- timedSend(ctx, make(chan int), 1, newStoppedTimer(), time.Hour) }()

			synctest.Wait()
			cancel()

			if <-done {
				t.Fatal("send reported success after cancel")
			}
		})
	})

	t.Run("the timer is stopped and reusable", func(t *testing.T) {
		synctest.Test(t, func(t *testing.T) {
			timer := newStoppedTimer()
			ch := make(chan int)

			// A wait that a receiver ends early leaves the timer stopped, with nothing in its channel.
			go func() { <-ch }()

			if !timedSend(t.Context(), ch, 1, timer, time.Second) {
				t.Fatal("send failed")
			}

			if timer.Stop() {
				t.Fatal("timer still running after timedSend returned")
			}

			// The next wait gets its full duration, not a stale expiry from the earlier call.
			time.Sleep(time.Hour)
			start := time.Now()

			if timedSend(t.Context(), ch, 2, timer, time.Second) {
				t.Fatal("send reported success without a receiver")
			}

			if got := time.Since(start); got != time.Second {
				t.Fatalf("reused timer waited %v, want 1s", got)
			}
		})
	})
}

func TestNewStoppedTimer(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		timer := newStoppedTimer()

		time.Sleep(2 * time.Hour)

		select {
		case <-timer.C:
			t.Fatal("stopped timer fired")
		default:
		}
	})
}
