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

package watermark

// Mutants that survive on purpose:
// - buffer-channel at the owner's input channel: a buffer lets forwarders run ahead, but the owner still handles
//   every message in the order it receives it and emits only what the watermark allows.
// - INVERT_LOOPCTRL of the break in windows: the starts are sorted, so continue skips only windows that are also
//   not due.

import (
	"context"
	"errors"
	"fmt"
	"math"
	"slices"
	"testing"
	"testing/synctest"
)

// harness runs merge on n inputs that the test feeds by hand, and records what it emits.
type harness struct {
	out    chan string
	err    chan error
	inputs []chan message
}

func start(ctx context.Context, n, limit int) *harness {
	h := &harness{out: make(chan string, 100), err: make(chan error, 1)}
	ins := make([]<-chan message, n)
	for i := range n {
		h.inputs = append(h.inputs, make(chan message))
		ins[i] = h.inputs[i]
	}

	go func() {
		h.err <- merge(ctx, ins, limit, func(m message) { h.out <- fmt.Sprint(m.at, m.value) })
	}()

	return h
}

// emitted returns what merge has emitted since the last call, once every goroutine has settled.
func (h *harness) emitted() []string {
	synctest.Wait()

	var got []string
	for len(h.out) > 0 {
		got = append(got, <-h.out)
	}

	return got
}

func (h *harness) event(i, at int) { h.inputs[i] <- message{at: at, value: string(rune('a' + i))} }

func (h *harness) punct(i, at int) { h.inputs[i] <- message{at: at, punct: true} }

func check(t *testing.T, got []string, want ...string) {
	t.Helper()

	if !slices.Equal(got, want) {
		t.Fatalf("emitted %v, want %v", got, want)
	}
}

func TestMerge(t *testing.T) {
	t.Run("no inputs", func(t *testing.T) {
		err := merge(t.Context(), nil, 0, func(message) { t.Fatal("emitted") })
		if err != nil {
			t.Fatal(err)
		}
	})

	t.Run("emits in event-time order, equal times by input", func(t *testing.T) {
		synctest.Test(t, func(t *testing.T) {
			h := start(t.Context(), 3, 10)

			h.event(0, 5)
			h.event(1, 1)
			h.event(2, 5)
			h.event(1, 5)
			h.event(2, 6)
			h.event(0, 9)
			for _, in := range h.inputs {
				close(in)
			}

			check(t, h.emitted(), "1b", "5a", "5b", "5c", "6c", "9a")

			if err := <-h.err; err != nil {
				t.Fatal(err)
			}
		})
	})

	t.Run("keeps the order of equal times within an input", func(t *testing.T) {
		synctest.Test(t, func(t *testing.T) {
			h := start(t.Context(), 2, 10)

			for _, v := range []string{"p", "q", "r", "s", "t"} {
				h.inputs[0] <- message{at: 1, value: v}
			}
			close(h.inputs[0])
			close(h.inputs[1])

			check(t, h.emitted(), "1p", "1q", "1r", "1s", "1t")
			<-h.err
		})
	})

	t.Run("an input that has not sent holds everything back", func(t *testing.T) {
		synctest.Test(t, func(t *testing.T) {
			h := start(t.Context(), 2, 10)

			h.event(0, 1)
			h.event(0, 2)
			check(t, h.emitted())

			h.event(1, 5)
			check(t, h.emitted(), "1a", "2a")

			h.event(0, 5)
			check(t, h.emitted(), "5a", "5b")

			close(h.inputs[0])
			close(h.inputs[1])
			check(t, h.emitted())
			<-h.err
		})
	})

	t.Run("a punctuation advances the watermark without an event", func(t *testing.T) {
		synctest.Test(t, func(t *testing.T) {
			h := start(t.Context(), 2, 10)

			h.event(0, 1)
			h.event(0, 4)
			h.punct(1, 3)
			check(t, h.emitted(), "1a")

			h.punct(1, 4)
			check(t, h.emitted(), "4a")

			close(h.inputs[0])
			close(h.inputs[1])
			<-h.err
		})
	})

	t.Run("a closed input no longer holds the watermark", func(t *testing.T) {
		synctest.Test(t, func(t *testing.T) {
			h := start(t.Context(), 3, 10)

			h.event(0, 1)
			h.event(1, 2)
			h.event(0, 7)
			check(t, h.emitted())

			close(h.inputs[2])
			check(t, h.emitted(), "1a", "2b")

			close(h.inputs[1])
			check(t, h.emitted(), "7a")

			close(h.inputs[0])
			if err := <-h.err; err != nil {
				t.Fatal(err)
			}
		})
	})

	t.Run("more waiting events than the limit fails", func(t *testing.T) {
		synctest.Test(t, func(t *testing.T) {
			h := start(t.Context(), 2, 2)

			h.event(0, 1)
			h.event(0, 2)
			synctest.Wait()

			select {
			case err := <-h.err:
				t.Fatalf("failed at the limit: %v", err)
			default:
			}

			h.event(0, 3)
			if err := <-h.err; !errors.Is(err, errFull) {
				t.Fatalf("err = %v, want %v", err, errFull)
			}
		})
	})

	t.Run("events emitted at once do not count against the limit", func(t *testing.T) {
		synctest.Test(t, func(t *testing.T) {
			h := start(t.Context(), 1, 0)

			h.event(0, 1)
			h.event(0, 2)
			check(t, h.emitted(), "1a", "2a")

			close(h.inputs[0])
			if err := <-h.err; err != nil {
				t.Fatal(err)
			}
		})
	})

	t.Run("cancel returns the context error and stops the forwarders", func(t *testing.T) {
		synctest.Test(t, func(t *testing.T) {
			ctx, cancel := context.WithCancel(t.Context())
			h := start(ctx, 2, 10)

			h.event(0, 1)
			cancel()

			if err := <-h.err; !errors.Is(err, context.Canceled) {
				t.Fatalf("err = %v, want %v", err, context.Canceled)
			}

			close(h.inputs[0])
			close(h.inputs[1])
		})
	})

	t.Run("a forwarder blocked on its close report stops on cancel", func(t *testing.T) {
		synctest.Test(t, func(t *testing.T) {
			done := make(chan struct{})
			input := make(chan message)
			ctx, cancel := context.WithCancel(t.Context())

			go func() {
				defer close(done)
				forward(ctx, 0, input, make(chan tagged))
			}()

			close(input)
			synctest.Wait()
			cancel()
			<-done
		})
	})
}

func TestWatermark(t *testing.T) {
	tests := []struct {
		name   string
		latest map[int]int
		open   int
		want   int
		ok     bool
	}{
		{"unknown while an open input has not sent", map[int]int{0: 3}, 2, 0, false},
		{"least latest time", map[int]int{0: 3, 1: 1, 2: 2}, 3, 1, true},
		{"everything is due when no input is open", map[int]int{}, 0, math.MaxInt, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := watermark(tt.latest, tt.open)
			if got != tt.want || ok != tt.ok {
				t.Fatalf("watermark = %d, %v; want %d, %v", got, ok, tt.want, tt.ok)
			}
		})
	}
}

// windowRun runs windows of size 10 on msgs and returns the emitted windows as start:count pairs, the late events, and the
// error.
func windowRun(t *testing.T, lateness, limit int, msgs ...message) ([]window, []int, error) {
	t.Helper()

	in := make(chan message, len(msgs))
	for _, m := range msgs {
		in <- m
	}
	close(in)

	var got []window
	var late []int
	err := windows(t.Context(), in, 10, lateness, limit,
		func(w window) { got = append(got, w) },
		func(m message) { late = append(late, m.at) },
	)

	return got, late, err
}

func ev(at int) message { return message{at: at} }

func wm(at int) message { return message{at: at, punct: true} }

func TestWindows(t *testing.T) {
	t.Run("counts per window and flushes on close in order", func(t *testing.T) {
		got, late, err := windowRun(t, 0, 10, ev(25), ev(3), ev(9), ev(10), ev(19), ev(0))
		want := []window{{0, 3}, {10, 2}, {20, 1}}
		if !slices.Equal(got, want) || late != nil || err != nil {
			t.Fatalf("got %v, late %v, err %v; want %v", got, late, err, want)
		}
	})

	t.Run("emits a window once the watermark reaches its end", func(t *testing.T) {
		synctest.Test(t, func(t *testing.T) {
			in := make(chan message)
			out := make(chan window, 10)
			go func() {
				_ = windows(t.Context(), in, 10, 0, 10, func(w window) { out <- w }, func(message) {})
			}()

			in <- ev(5)
			in <- wm(9)
			synctest.Wait()
			if len(out) != 0 {
				t.Fatal("window emitted before the watermark reached its end")
			}

			in <- wm(10)
			synctest.Wait()
			if len(out) != 1 {
				t.Fatal("window not emitted when the watermark reached its end")
			}

			close(in)
		})
	})

	t.Run("accepts events within the allowed lateness", func(t *testing.T) {
		got, late, err := windowRun(t, 3, 10, ev(1), wm(12), ev(2), ev(15))
		want := []window{{0, 2}, {10, 1}}
		if !slices.Equal(got, want) || late != nil || err != nil {
			t.Fatalf("got %v, late %v, err %v; want %v", got, late, err, want)
		}
	})

	t.Run("accepts a first event within the allowed lateness", func(t *testing.T) {
		got, late, err := windowRun(t, 3, 10, wm(12), ev(2))
		want := []window{{0, 1}}
		if !slices.Equal(got, want) || late != nil || err != nil {
			t.Fatalf("got %v, late %v, err %v; want %v", got, late, err, want)
		}
	})

	t.Run("passes events for an emitted window to late", func(t *testing.T) {
		got, late, err := windowRun(t, 3, 10, ev(1), wm(13), ev(2), ev(9), ev(10))
		want := []window{{0, 1}, {10, 1}}
		if !slices.Equal(got, want) || !slices.Equal(late, []int{2, 9}) || err != nil {
			t.Fatalf("got %v, late %v, err %v; want %v, late [2 9]", got, late, err, want)
		}
	})

	t.Run("a lower watermark does not move it back", func(t *testing.T) {
		got, late, err := windowRun(t, 0, 10, wm(10), wm(5), ev(3))
		if got != nil || !slices.Equal(late, []int{3}) || err != nil {
			t.Fatalf("got %v, late %v, err %v; want late [3]", got, late, err)
		}
	})

	t.Run("more open windows than the limit fails", func(t *testing.T) {
		_, _, err := windowRun(t, 0, 2, ev(1), ev(11))
		if err != nil {
			t.Fatalf("failed at the limit: %v", err)
		}

		_, _, err = windowRun(t, 0, 2, ev(1), ev(11), ev(21))
		if !errors.Is(err, errFull) {
			t.Fatalf("err = %v, want %v", err, errFull)
		}
	})

	t.Run("cancel returns the context error", func(t *testing.T) {
		ctx, cancel := context.WithCancel(t.Context())
		cancel()

		err := windows(ctx, make(chan message), 10, 0, 10, func(window) {}, func(message) {})
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("err = %v, want %v", err, context.Canceled)
		}
	})
}
