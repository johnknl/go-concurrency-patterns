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

package tee

// Mutants that survive on purpose:

import (
	"context"
	"slices"
	"sync"
	"testing"
	"testing/synctest"
)

func TestGenerate(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		if got := collect(generate(t.Context(), 1, 2, 3)); !slices.Equal(got, []int{1, 2, 3}) {
			t.Fatalf("got %v", got)
		}

		ctx, cancel := context.WithCancel(t.Context())
		out := generate(ctx, 1, 2)
		cancel()
		synctest.Wait()

		if v, ok := <-out; ok {
			t.Fatalf("received %d after cancel, want a closed channel", v)
		}
	})
}

func TestTee(t *testing.T) {
	for _, tc := range []struct {
		name string
		nums []int
	}{
		{"none", nil},
		{"one", []int{1}},
		{"many", []int{1, 2, 3, 4, 5}},
	} {
		t.Run(tc.name+" reaches both sides in order", func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) {
				left, right := tee(t.Context(), generate(t.Context(), tc.nums...))

				var l, r []int

				var wg sync.WaitGroup
				wg.Go(func() { l = collect(left) })
				wg.Go(func() { r = collect(right) })
				wg.Wait()

				if !slices.Equal(l, tc.nums) || !slices.Equal(r, tc.nums) {
					t.Fatalf("left %v, right %v, want %v on both", l, r, tc.nums)
				}
			})
		})
	}

	for _, first := range []string{"left", "right"} {
		t.Run("the slower side holds back the faster one when "+first+" reads first", func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) {
				ctx, cancel := context.WithCancel(t.Context())
				defer cancel()

				in := make(chan int)
				left, right := tee(ctx, in)

				fast, slow := left, right
				if first == "right" {
					fast, slow = right, left
				}

				in <- 1
				if got := <-fast; got != 1 {
					t.Fatalf("got %d, want 1", got)
				}

				synctest.Wait()

				select {
				case v := <-fast:
					t.Fatalf("fast side got %d before the slow side read 1", v)
				case in <- 2:
					t.Fatal("tee took the next value before the slow side read 1")
				default:
				}

				if got := <-slow; got != 1 {
					t.Fatalf("got %d, want 1", got)
				}

				in <- 2
				if a, b := <-fast, <-slow; a != 2 || b != 2 {
					t.Fatalf("got %d and %d, want 2 on both", a, b)
				}
			})
		})
	}

	t.Run("cancel while waiting for input closes both", func(t *testing.T) {
		synctest.Test(t, func(t *testing.T) {
			ctx, cancel := context.WithCancel(t.Context())
			left, right := tee(ctx, make(chan int))

			cancel()

			if v, ok := <-left; ok {
				t.Fatalf("left received %d, want closed", v)
			}

			if v, ok := <-right; ok {
				t.Fatalf("right received %d, want closed", v)
			}
		})
	})

	t.Run("cancel while one side has not read closes both", func(t *testing.T) {
		synctest.Test(t, func(t *testing.T) {
			ctx, cancel := context.WithCancel(t.Context())
			in := make(chan int)
			left, right := tee(ctx, in)

			in <- 1
			<-left
			cancel()
			synctest.Wait()

			if v, ok := <-right; ok {
				t.Fatalf("right received %d, want closed", v)
			}

			if v, ok := <-left; ok {
				t.Fatalf("left received %d, want closed", v)
			}
		})
	})
}

func TestCollect(t *testing.T) {
	ch := make(chan int, 2)
	ch <- 1
	ch <- 2
	close(ch)

	if got := collect(ch); !slices.Equal(got, []int{1, 2}) {
		t.Fatalf("got %v", got)
	}
}
