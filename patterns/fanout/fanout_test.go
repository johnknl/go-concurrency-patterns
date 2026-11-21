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

package fanout

// Mutants that survive on purpose:

import (
	"slices"
	"testing"
	"testing/synctest"
)

func collect(ch <-chan int) []int {
	var got []int
	for v := range ch {
		got = append(got, v)
	}

	return got
}

func TestEmit(t *testing.T) {
	for _, tc := range []struct {
		name string
		nums []int
	}{
		{"none", nil},
		{"one", []int{1}},
		{"many in order", []int{5, 3, 4}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) {
				if got := collect(emit(tc.nums...)); !slices.Equal(got, tc.nums) {
					t.Fatalf("got %v, want %v", got, tc.nums)
				}
			})
		})
	}

	t.Run("runs no further ahead than the consumer", func(t *testing.T) {
		synctest.Test(t, func(t *testing.T) {
			out := emit(1, 2, 3)
			synctest.Wait()

			if n := len(out); n != 0 {
				t.Fatalf("%d values queued before anyone read, want 0", n)
			}

			collect(out)
		})
	})
}

func TestMerge(t *testing.T) {
	t.Run("output is unbuffered", func(t *testing.T) {
		synctest.Test(t, func(t *testing.T) {
			out := merge(emit(1, 2), emit(3))
			synctest.Wait()

			if n := len(out); n != 0 {
				t.Fatalf("%d values queued before anyone read, want 0", n)
			}

			collect(out)
		})
	})

	t.Run("no inputs closes at once", func(t *testing.T) {
		synctest.Test(t, func(t *testing.T) {
			if got := collect(merge()); len(got) != 0 {
				t.Fatalf("got %v, want nothing", got)
			}
		})
	})

	t.Run("delivers every value exactly once and keeps each input's order", func(t *testing.T) {
		synctest.Test(t, func(t *testing.T) {
			got := collect(merge(emit(1, 2, 3), emit(), emit(10, 20), emit(100)))

			var small, large []int

			for _, v := range got {
				if v < 10 {
					small = append(small, v)
				} else if v < 100 {
					large = append(large, v)
				}
			}

			if !slices.Equal(small, []int{1, 2, 3}) || !slices.Equal(large, []int{10, 20}) {
				t.Fatalf("per-input order lost: %v", got)
			}

			slices.Sort(got)
			if want := []int{1, 2, 3, 10, 20, 100}; !slices.Equal(got, want) {
				t.Fatalf("got %v, want %v", got, want)
			}
		})
	})

	t.Run("a silent input does not hold back the others", func(t *testing.T) {
		synctest.Test(t, func(t *testing.T) {
			silent := make(chan int)
			out := merge(silent, emit(1, 2))

			if a, b := <-out, <-out; a+b != 3 {
				t.Fatalf("got %d and %d, want 1 and 2", a, b)
			}

			close(silent)

			if v, ok := <-out; ok {
				t.Fatalf("received %d, want a closed channel", v)
			}
		})
	})

	t.Run("output stays open until the last input closes", func(t *testing.T) {
		synctest.Test(t, func(t *testing.T) {
			first, second := make(chan int), make(chan int)
			out := merge(first, second)

			close(first)
			synctest.Wait()

			select {
			case v, ok := <-out:
				t.Fatalf("output ready (%d, %v) while an input is still open", v, ok)
			default:
			}

			second <- 7
			if got := <-out; got != 7 {
				t.Fatalf("got %d, want 7", got)
			}

			close(second)

			if v, ok := <-out; ok {
				t.Fatalf("received %d, want a closed channel", v)
			}
		})
	})
}
