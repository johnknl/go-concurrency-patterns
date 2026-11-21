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

package ordone

// No mutants survive.

import (
	"slices"
	"testing"
	"testing/synctest"
)

func TestSource(t *testing.T) {
	var got []int
	for v := range source() {
		got = append(got, v)
	}

	if want := []int{1, 2, 3, 4, 5}; !slices.Equal(got, want) {
		t.Fatalf("got %v, want %v", got, want)
	}
}

func TestOrDone(t *testing.T) {
	t.Run("forwards every value until in closes", func(t *testing.T) {
		synctest.Test(t, func(t *testing.T) {
			var got []int
			for v := range orDone(make(chan struct{}), source()) {
				got = append(got, v)
			}

			if want := []int{1, 2, 3, 4, 5}; !slices.Equal(got, want) {
				t.Fatalf("got %v, want %v", got, want)
			}
		})
	})

	t.Run("empty input closes the output", func(t *testing.T) {
		synctest.Test(t, func(t *testing.T) {
			in := make(chan int)
			close(in)

			if v, ok := <-orDone(make(chan struct{}), in); ok {
				t.Fatalf("received %d, want a closed channel", v)
			}
		})
	})

	t.Run("done closed before start forwards nothing", func(t *testing.T) {
		synctest.Test(t, func(t *testing.T) {
			done := make(chan struct{})
			close(done)

			out := orDone(done, source())
			synctest.Wait() // a value taken from in can only go out to a reader, and there is none yet

			if v, ok := <-out; ok {
				t.Fatalf("received %d, want a closed channel", v)
			}
		})
	})

	t.Run("done while waiting for in", func(t *testing.T) {
		synctest.Test(t, func(t *testing.T) {
			done := make(chan struct{})
			in := make(chan int) // never sends, never closes: the owner does not cooperate
			out := orDone(done, in)

			synctest.Wait()
			close(done)

			if v, ok := <-out; ok {
				t.Fatalf("received %d, want a closed channel", v)
			}
		})
	})

	t.Run("done while waiting for the reader", func(t *testing.T) {
		synctest.Test(t, func(t *testing.T) {
			done := make(chan struct{})
			in := make(chan int)
			out := orDone(done, in)

			in <- 1 // taken, now blocked on out
			close(done)
			synctest.Wait()

			if v, ok := <-out; ok {
				t.Fatalf("received %d, want a closed channel", v)
			}
		})
	})

	t.Run("hands over one value at a time", func(t *testing.T) {
		synctest.Test(t, func(t *testing.T) {
			done := make(chan struct{})
			defer close(done)

			in := make(chan int, 3)
			in <- 1
			in <- 2
			in <- 3

			out := orDone(done, in)
			synctest.Wait()

			if n := len(in); n != 2 {
				t.Fatalf("%d values left in in, want 2: orDone may hold only the one it is sending", n)
			}

			if got := <-out; got != 1 {
				t.Fatalf("got %d, want 1", got)
			}
		})
	})
}
