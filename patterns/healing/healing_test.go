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

package healing

// Mutants that survive on purpose:
// - remove-cancel healing.go:51 and :83: ctx is done, so the ward's context, derived from it, is already canceled.
// - remove-cancel healing.go:60 and :73: the ward has closed both channels, so it has ended; cancel only releases
//   the context.

import (
	"context"
	"slices"
	"sync"
	"testing"
	"testing/synctest"
)

// fakeWard is a ward whose runs are driven by the test. Each run sends the values of its script, pulsing before
// each one, and then either ends or, if stuck, waits until it is canceled.
type fakeWard struct {
	events []string
	runs   int
	sent   int
	mu     sync.Mutex
}

func (w *fakeWard) record(event string) {
	w.mu.Lock()
	defer w.mu.Unlock()

	w.events = append(w.events, event)
}

func (w *fakeWard) sentCount() int {
	w.mu.Lock()
	defer w.mu.Unlock()

	return w.sent
}

func (w *fakeWard) log() []string {
	w.mu.Lock()
	defer w.mu.Unlock()

	return slices.Clone(w.events)
}

// start returns a ward whose run i (counting from 1) executes scripts[i-1]; values < 0 mean "get stuck".
func (w *fakeWard) start(scripts ...[]int) ward {
	return func(ctx context.Context) (<-chan struct{}, <-chan int) {
		w.mu.Lock()
		w.runs++
		run := w.runs
		w.mu.Unlock()

		w.record("start")

		heartbeat := make(chan struct{}, 1)
		results := make(chan int)

		go func() {
			defer close(heartbeat)
			defer close(results)
			defer w.record("end")

			for _, n := range scripts[run-1] {
				if n < 0 {
					<-ctx.Done()
					return
				}

				select {
				case heartbeat <- struct{}{}:
				default:
				}

				select {
				case results <- n:
					w.mu.Lock()
					w.sent++
					w.mu.Unlock()
				case <-ctx.Done():
					return
				}
			}
		}()

		return heartbeat, results
	}
}

// timeouts returns a timeout function that hands out the given channels in order, and nil (never) after.
func timeouts(chans ...chan struct{}) (func() <-chan struct{}, *int) {
	calls := 0

	return func() <-chan struct{} {
		calls++
		if calls <= len(chans) {
			return chans[calls-1]
		}

		return nil
	}, &calls
}

func closed() chan struct{} {
	c := make(chan struct{})
	close(c)

	return c
}

func TestSteward(t *testing.T) {
	t.Run("forwards the results of a healthy ward and ends with it", func(t *testing.T) {
		synctest.Test(t, func(t *testing.T) {
			w := &fakeWard{}
			timeout, _ := timeouts()

			var got []int
			for n := range steward(t.Context(), timeout, w.start([]int{1, 2, 3})) {
				got = append(got, n)
			}

			if want := []int{1, 2, 3}; !slices.Equal(got, want) {
				t.Fatalf("results = %v, want %v", got, want)
			}

			if w.runs != 1 {
				t.Fatalf("runs = %d, want 1", w.runs)
			}
		})
	})

	t.Run("a ward without results ends the steward", func(t *testing.T) {
		synctest.Test(t, func(t *testing.T) {
			w := &fakeWard{}
			timeout, _ := timeouts()

			for range steward(t.Context(), timeout, w.start(nil)) {
				t.Fatal("unexpected result")
			}
		})
	})

	t.Run("every pulse starts a new timeout", func(t *testing.T) {
		synctest.Test(t, func(t *testing.T) {
			w := &fakeWard{}
			timeout, calls := timeouts()

			for range steward(t.Context(), timeout, w.start([]int{1, 2, 3})) {
			}

			// One timeout at the start and at most one per pulse; pulses can be dropped, but the first is not.
			if *calls < 2 || *calls > 4 {
				t.Fatalf("timeout calls = %d, want 2..4", *calls)
			}
		})
	})

	t.Run("a stuck ward is canceled, has ended, and is restarted", func(t *testing.T) {
		synctest.Test(t, func(t *testing.T) {
			w := &fakeWard{}
			timeout, _ := timeouts(closed())

			var got []int
			for n := range steward(t.Context(), timeout, w.start([]int{-1}, []int{7, 8})) {
				got = append(got, n)
			}

			if want := []int{7, 8}; !slices.Equal(got, want) {
				t.Fatalf("results = %v, want %v", got, want)
			}

			if want := []string{"start", "end", "start", "end"}; !slices.Equal(w.log(), want) {
				t.Fatalf("events = %v, want %v", w.log(), want)
			}
		})
	})

	t.Run("a ward that stalls after some results is restarted", func(t *testing.T) {
		synctest.Test(t, func(t *testing.T) {
			w := &fakeWard{}
			expire := make(chan struct{})
			timeout, _ := timeouts(nil, expire)

			out := steward(t.Context(), timeout, w.start([]int{1, -1}, []int{2}))

			if got := <-out; got != 1 {
				t.Fatalf("first result = %d, want 1", got)
			}

			synctest.Wait()
			close(expire)

			if got := <-out; got != 2 {
				t.Fatalf("second result = %d, want 2", got)
			}

			if _, ok := <-out; ok {
				t.Fatal("out not closed")
			}

			if w.runs != 2 {
				t.Fatalf("runs = %d, want 2", w.runs)
			}
		})
	})

	t.Run("results closed before the heartbeat are not taken again", func(t *testing.T) {
		synctest.Test(t, func(t *testing.T) {
			heartbeat := make(chan struct{})
			results := make(chan int)
			close(results)

			start := func(context.Context) (<-chan struct{}, <-chan int) { return heartbeat, results }
			timeout, _ := timeouts()

			out := steward(t.Context(), timeout, start)

			synctest.Wait()
			close(heartbeat)

			if _, ok := <-out; ok {
				t.Fatal("out not closed")
			}
		})
	})

	t.Run("buffered results of an ended ward are forwarded before the steward ends", func(t *testing.T) {
		// Both channels are ready when the ward has ended: a closed heartbeat and buffered results. Every pass
		// of the select may pick either, so repeat until a lost result would have shown.
		for range 100 {
			synctest.Test(t, func(t *testing.T) {
				start := func(context.Context) (<-chan struct{}, <-chan int) {
					heartbeat := make(chan struct{})
					results := make(chan int, 3)
					results <- 1
					results <- 2
					results <- 3
					close(heartbeat)
					close(results)

					return heartbeat, results
				}

				timeout := func() <-chan struct{} { return nil }

				var got []int
				for n := range steward(t.Context(), timeout, start) {
					got = append(got, n)
				}

				if !slices.Equal(got, []int{1, 2, 3}) {
					t.Fatalf("forwarded %v, want [1 2 3]", got)
				}
			})
		}
	})

	t.Run("does not take the next result before the previous one was forwarded", func(t *testing.T) {
		synctest.Test(t, func(t *testing.T) {
			w := &fakeWard{}
			timeout, _ := timeouts()

			out := steward(t.Context(), timeout, w.start([]int{1, 2, 3}))

			synctest.Wait()

			if sent := w.sentCount(); sent != 1 {
				t.Fatalf("ward sent %d results while the consumer took none, want 1", sent)
			}

			for range out {
			}
		})
	})

	t.Run("cancel closes out while the consumer is not reading", func(t *testing.T) {
		synctest.Test(t, func(t *testing.T) {
			w := &fakeWard{}
			timeout, _ := timeouts()
			ctx, cancel := context.WithCancel(t.Context())

			out := steward(ctx, timeout, w.start([]int{1}))

			synctest.Wait() // the steward holds result 1
			cancel()
			synctest.Wait()

			if n, ok := <-out; ok {
				t.Fatalf("got %d after cancel, want out closed", n)
			}
		})
	})

	t.Run("cancel closes out even if the ward ignores it", func(t *testing.T) {
		synctest.Test(t, func(t *testing.T) {
			release := make(chan struct{})
			heartbeat := make(chan struct{})

			start := func(context.Context) (<-chan struct{}, <-chan int) {
				go func() {
					<-release
					close(heartbeat)
				}()

				return heartbeat, nil
			}
			timeout, _ := timeouts()
			ctx, cancel := context.WithCancel(t.Context())

			out := steward(ctx, timeout, start)

			cancel()
			synctest.Wait()

			select {
			case _, ok := <-out:
				if ok {
					t.Fatal("unexpected result")
				}
			default:
				t.Fatal("out still open after cancel")
			}

			close(release)
		})
	})

	t.Run("cancel ends the ward", func(t *testing.T) {
		synctest.Test(t, func(t *testing.T) {
			w := &fakeWard{}
			timeout, _ := timeouts()
			ctx, cancel := context.WithCancel(t.Context())

			out := steward(ctx, timeout, w.start([]int{-1}))

			synctest.Wait()
			cancel()

			for range out {
			}

			synctest.Wait()

			if want := []string{"start", "end"}; !slices.Equal(w.log(), want) {
				t.Fatalf("events = %v, want %v", w.log(), want)
			}
		})
	})
}

func TestSupervised(t *testing.T) {
	t.Run("forwards the inner results with a pulse each and ends with the inner steward", func(t *testing.T) {
		synctest.Test(t, func(t *testing.T) {
			w := &fakeWard{}
			inner, _ := timeouts()

			heartbeat, results := supervised(inner, w.start([]int{1, 2}))(t.Context())

			for _, want := range []int{1, 2} {
				synctest.Wait()

				select {
				case <-heartbeat:
				default:
					t.Fatalf("no pulse before result %d", want)
				}

				if got := <-results; got != want {
					t.Fatalf("result = %d, want %d", got, want)
				}
			}

			if _, ok := <-results; ok {
				t.Fatal("results not closed")
			}

			for range heartbeat {
			}
		})
	})

	t.Run("an unread heartbeat does not block the results", func(t *testing.T) {
		synctest.Test(t, func(t *testing.T) {
			w := &fakeWard{}
			inner, _ := timeouts()

			_, results := supervised(inner, w.start([]int{1, 2, 3}))(t.Context())

			var got []int
			for n := range results {
				got = append(got, n)
			}

			if want := []int{1, 2, 3}; !slices.Equal(got, want) {
				t.Fatalf("results = %v, want %v", got, want)
			}
		})
	})

	t.Run("the inner steward restarts its ward", func(t *testing.T) {
		synctest.Test(t, func(t *testing.T) {
			w := &fakeWard{}
			inner, _ := timeouts(closed())
			outer, _ := timeouts()

			var got []int
			for n := range steward(t.Context(), outer, supervised(inner, w.start([]int{-1}, []int{5}))) {
				got = append(got, n)
			}

			if want := []int{5}; !slices.Equal(got, want) {
				t.Fatalf("results = %v, want %v", got, want)
			}
		})
	})

	t.Run("cancel ends the subtree before the heartbeat closes", func(t *testing.T) {
		synctest.Test(t, func(t *testing.T) {
			w := &fakeWard{}
			inner, _ := timeouts()
			ctx, cancel := context.WithCancel(t.Context())

			heartbeat, results := supervised(inner, w.start([]int{1, -1}))(ctx)

			synctest.Wait() // the adapter holds result 1
			cancel()

			for range heartbeat {
			}

			if _, ok := <-results; ok {
				t.Fatal("result forwarded after cancel")
			}
		})
	})
}
