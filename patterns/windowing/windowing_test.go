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

package windowing

// Mutants that survive on purpose:
// - none.

import (
	"context"
	"errors"
	"slices"
	"testing"
	"testing/synctest"
	"time"
)

// epoch is the fake time at which a synctest bubble starts.
var epoch = time.Date(2000, 1, 1, 0, 0, 0, 0, time.UTC)

// run starts f with a ticker of one second inside a bubble and returns the input, the emitted windows, and the
// error that f returns.
func run(
	t *testing.T,
	f func(ctx context.Context, in <-chan int, tick <-chan time.Time, emit func(window)) error,
) (chan<- int, <-chan window, <-chan error) {
	t.Helper()

	in := make(chan int)
	out := make(chan window, 100)
	errc := make(chan error, 1)
	ticker := time.NewTicker(time.Second)

	go func() {
		defer ticker.Stop()
		errc <- f(t.Context(), in, ticker.C, func(w window) { out <- w })
	}()

	return in, out, errc
}

// emitted returns the windows emitted since the last call, once every goroutine has settled.
func emitted(out <-chan window) []window {
	synctest.Wait()

	var got []window
	for len(out) > 0 {
		got = append(got, <-out)
	}

	return got
}

func check(t *testing.T, got []window, want ...window) {
	t.Helper()

	if !slices.EqualFunc(got, want, func(a, b window) bool {
		return a.end.Equal(b.end) && a.count == b.count && a.sum == b.sum
	}) {
		t.Fatalf("emitted %v, want %v", got, want)
	}
}

func sec(n float64) time.Time { return epoch.Add(time.Duration(n * float64(time.Second))) }

func TestTumbling(t *testing.T) {
	t.Run("one window per tick, empty ones included, partial one on close", func(t *testing.T) {
		synctest.Test(t, func(t *testing.T) {
			in, out, errc := run(t, tumbling)

			in <- 1
			in <- 2
			time.Sleep(time.Second - time.Nanosecond)
			check(t, emitted(out))

			in <- 4
			time.Sleep(time.Nanosecond)
			check(t, emitted(out), window{end: sec(1), count: 3, sum: 7})

			time.Sleep(time.Second)
			check(t, emitted(out), window{end: sec(2)})

			time.Sleep(time.Second / 2)
			in <- 5
			close(in)
			check(t, emitted(out), window{end: sec(2.5), count: 1, sum: 5})

			if err := <-errc; err != nil {
				t.Fatal(err)
			}
		})
	})

	t.Run("cancel returns the context error", func(t *testing.T) {
		ctx, cancel := context.WithCancel(t.Context())
		cancel()

		err := tumbling(ctx, make(chan int), make(chan time.Time), func(window) { t.Fatal("emitted") })
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("err = %v, want %v", err, context.Canceled)
		}
	})
}

func TestSliding(t *testing.T) {
	sliding3 := func(ctx context.Context, in <-chan int, tick <-chan time.Time, emit func(window)) error {
		return sliding(ctx, in, tick, 3, emit)
	}

	t.Run("reports the last three periods at every tick", func(t *testing.T) {
		synctest.Test(t, func(t *testing.T) {
			in, out, errc := run(t, sliding3)

			in <- 1
			time.Sleep(time.Second)
			check(t, emitted(out), window{end: sec(1), count: 1, sum: 1})

			in <- 2
			in <- 3
			time.Sleep(time.Second)
			check(t, emitted(out), window{end: sec(2), count: 3, sum: 6})

			in <- 10
			time.Sleep(time.Second)
			check(t, emitted(out), window{end: sec(3), count: 4, sum: 16})

			time.Sleep(time.Second)
			check(t, emitted(out), window{end: sec(4), count: 3, sum: 15})

			time.Sleep(time.Second)
			check(t, emitted(out), window{end: sec(5), count: 1, sum: 10})

			time.Sleep(time.Second)
			check(t, emitted(out), window{end: sec(6)})

			in <- 7
			close(in)
			check(t, emitted(out), window{end: sec(6), count: 1, sum: 7})

			if err := <-errc; err != nil {
				t.Fatal(err)
			}
		})
	})

	t.Run("cancel returns the context error", func(t *testing.T) {
		ctx, cancel := context.WithCancel(t.Context())
		cancel()

		err := sliding(ctx, make(chan int), make(chan time.Time), 2, func(window) { t.Fatal("emitted") })
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("err = %v, want %v", err, context.Canceled)
		}
	})
}
