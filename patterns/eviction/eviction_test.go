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

package eviction

// Mutants that survive on purpose: none.

import (
	"errors"
	"slices"
	"testing"
	"testing/synctest"
	"time"
)

func newSub(name string, buffer int) *subscriber {
	return &subscriber{name: name, ch: make(chan int, buffer)}
}

func TestTimedSend(t *testing.T) {
	t.Run("sends into a free slot", func(t *testing.T) {
		synctest.Test(t, func(t *testing.T) {
			ch := make(chan int, 1)

			if !timedSend(ch, 1, time.NewTimer(time.Hour), time.Second) {
				t.Fatal("send failed")
			}
		})
	})

	t.Run("gives up after d", func(t *testing.T) {
		synctest.Test(t, func(t *testing.T) {
			timer := time.NewTimer(time.Hour)
			timer.Stop()
			start := time.Now()

			if timedSend(make(chan int), 1, timer, time.Second) {
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
			timer := time.NewTimer(time.Hour)
			timer.Stop()

			go func() {
				time.Sleep(time.Second / 2)
				<-ch
			}()

			if !timedSend(ch, 1, timer, time.Second) {
				t.Fatal("send failed")
			}

			if timer.Stop() {
				t.Fatal("timer still running")
			}
		})
	})
}

func TestBroadcast(t *testing.T) {
	t.Run("no subscribers", func(t *testing.T) {
		if got := broadcast(nil, 1, time.NewTimer(time.Hour), 0); len(got) != 0 {
			t.Fatalf("got %d subscribers, want 0", len(got))
		}
	})

	t.Run("delivers to every subscriber that keeps up", func(t *testing.T) {
		synctest.Test(t, func(t *testing.T) {
			a, b := newSub("a", 2), newSub("b", 2)
			subs := []*subscriber{a, b}
			timer := time.NewTimer(time.Hour)

			subs = broadcast(subs, 1, timer, 0)
			subs = broadcast(subs, 2, timer, 0)

			if len(subs) != 2 {
				t.Fatalf("remaining = %d, want 2", len(subs))
			}

			for _, s := range []*subscriber{a, b} {
				close(s.ch)

				if got := slices.Collect(chanValues(s.ch)); !slices.Equal(got, []int{1, 2}) {
					t.Fatalf("%s received %v, want [1 2]", s.name, got)
				}
			}
		})
	})

	t.Run("evicts after more than tolerance consecutive misses", func(t *testing.T) {
		for _, tolerance := range []int{0, 1, 3} {
			synctest.Test(t, func(t *testing.T) {
				stuck := newSub("stuck", 0)
				subs := []*subscriber{stuck}
				timer := time.NewTimer(time.Hour)

				for v := range tolerance {
					subs = broadcast(subs, v, timer, tolerance)
				}

				if len(subs) != 1 || stuck.err != nil {
					t.Fatalf("tolerance %d: evicted after %d misses", tolerance, tolerance)
				}

				if stuck.strikes != tolerance {
					t.Fatalf("strikes = %d, want %d", stuck.strikes, tolerance)
				}

				subs = broadcast(subs, tolerance, timer, tolerance)

				if len(subs) != 0 {
					t.Fatalf("tolerance %d: not evicted after %d misses", tolerance, tolerance+1)
				}

				if !errors.Is(stuck.err, errTooSlow) {
					t.Fatalf("cause = %v, want %v", stuck.err, errTooSlow)
				}

				if _, open := <-stuck.ch; open {
					t.Fatal("channel of evicted subscriber not closed")
				}
			})
		}
	})

	t.Run("a delivery clears the strikes", func(t *testing.T) {
		synctest.Test(t, func(t *testing.T) {
			s := newSub("flaky", 1)
			subs := []*subscriber{s}
			timer := time.NewTimer(time.Hour)

			// Miss, deliver, miss, deliver, ...: never two misses in a row.
			for v := range 6 {
				if v%2 == 1 {
					<-s.ch // make room
				}

				subs = broadcast(subs, v, timer, 1)
			}

			if len(subs) != 1 || s.err != nil {
				t.Fatal("subscriber evicted although its misses were not consecutive")
			}
		})
	})

	t.Run("the others keep receiving and keep their order", func(t *testing.T) {
		synctest.Test(t, func(t *testing.T) {
			a, stuck, c := newSub("a", 3), newSub("stuck", 0), newSub("c", 3)
			subs := []*subscriber{a, stuck, c}
			timer := time.NewTimer(time.Hour)

			for v := range 3 {
				subs = broadcast(subs, v, timer, 0)
			}

			if len(subs) != 2 || subs[0] != a || subs[1] != c {
				t.Fatalf("remaining = %v, want [a c]", names(subs))
			}

			for _, s := range []*subscriber{a, c} {
				close(s.ch)

				if got := slices.Collect(chanValues(s.ch)); !slices.Equal(got, []int{0, 1, 2}) {
					t.Fatalf("%s received %v, want [0 1 2]", s.name, got)
				}
			}
		})
	})

	t.Run("each miss costs at most the timeout, and none after eviction", func(t *testing.T) {
		synctest.Test(t, func(t *testing.T) {
			subs := []*subscriber{newSub("stuck", 0)}
			timer := time.NewTimer(time.Hour)
			start := time.Now()

			for v := range 5 {
				subs = broadcast(subs, v, timer, 1)
			}

			if got := time.Since(start); got != 2*time.Millisecond {
				t.Fatalf("5 broadcasts took %v, want 2ms", got)
			}
		})
	})
}

func TestRun(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		want := []string{
			"stuck open: false cause: too many missed deliveries",
			"healthy got 0",
			"healthy got 1",
			"healthy got 2",
			"remaining: 1",
		}

		if got := run(2); !slices.Equal(got, want) {
			t.Fatalf("run(2) = %q, want %q", got, want)
		}
	})
}

func chanValues(ch <-chan int) func(func(int) bool) {
	return func(yield func(int) bool) {
		for v := range ch {
			if !yield(v) {
				return
			}
		}
	}
}

func names(subs []*subscriber) []string {
	out := make([]string, 0, len(subs))
	for _, s := range subs {
		out = append(out, s.name)
	}

	return out
}
