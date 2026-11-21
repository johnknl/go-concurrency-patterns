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

package pipeline

// Mutants that survive on purpose: none. (gremlins reports ARITHMETIC_BASE at pipeline.go:57 as not covered
// because `n * n` sits in a select case with an empty body; TestSquare checks the squares.)

import (
	"context"
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

// feed returns an unbuffered channel the test controls, closed when the test ends.
func feed(t *testing.T) chan int {
	in := make(chan int)
	t.Cleanup(func() { close(in) })

	return in
}

func TestGenerate(t *testing.T) {
	for _, tc := range []struct {
		name string
		nums []int
	}{
		{"none", nil},
		{"one", []int{7}},
		{"many in order", []int{3, 1, 2, 5, 4}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) {
				if got := collect(generate(t.Context(), tc.nums...)); !slices.Equal(got, tc.nums) {
					t.Fatalf("got %v, want %v", got, tc.nums)
				}
			})
		})
	}

	t.Run("cancel while blocked closes the channel", func(t *testing.T) {
		synctest.Test(t, func(t *testing.T) {
			ctx, cancel := context.WithCancel(t.Context())
			out := generate(ctx, 1, 2, 3)

			if got := <-out; got != 1 {
				t.Fatalf("got %d, want 1", got)
			}

			cancel()
			synctest.Wait()

			if v, ok := <-out; ok {
				t.Fatalf("received %d after cancel, want a closed channel", v)
			}
		})
	})
}

func TestSquare(t *testing.T) {
	t.Run("squares every value in order", func(t *testing.T) {
		synctest.Test(t, func(t *testing.T) {
			got := collect(square(t.Context(), generate(t.Context(), -3, 0, 1, 2, 3)))
			if want := []int{9, 0, 1, 4, 9}; !slices.Equal(got, want) {
				t.Fatalf("got %v, want %v", got, want)
			}
		})
	})

	t.Run("closes when its input closes", func(t *testing.T) {
		synctest.Test(t, func(t *testing.T) {
			if got := collect(square(t.Context(), generate(t.Context()))); len(got) != 0 {
				t.Fatalf("got %v, want nothing", got)
			}
		})
	})

	t.Run("cancel while blocked on send leaves without its input closing", func(t *testing.T) {
		synctest.Test(t, func(t *testing.T) {
			ctx, cancel := context.WithCancel(t.Context())
			in := feed(t)
			out := square(ctx, in)

			in <- 4
			cancel()
			synctest.Wait()

			if v, ok := <-out; ok {
				t.Fatalf("received %d after cancel, want a closed channel", v)
			}
		})
	})
}

func TestAbove(t *testing.T) {
	t.Run("drops values that do not exceed the limit", func(t *testing.T) {
		synctest.Test(t, func(t *testing.T) {
			got := collect(above(t.Context(), 5, generate(t.Context(), 7, 4, 5, 6, -1, 100)))
			if want := []int{7, 6, 100}; !slices.Equal(got, want) {
				t.Fatalf("got %v, want %v", got, want)
			}
		})
	})

	t.Run("cancel while blocked on send leaves without its input closing", func(t *testing.T) {
		synctest.Test(t, func(t *testing.T) {
			ctx, cancel := context.WithCancel(t.Context())
			in := feed(t)
			out := above(ctx, 0, in)

			in <- 1
			cancel()
			synctest.Wait()

			if v, ok := <-out; ok {
				t.Fatalf("received %d after cancel, want a closed channel", v)
			}
		})
	})
}

func TestPipeline(t *testing.T) {
	t.Run("stages compose", func(t *testing.T) {
		synctest.Test(t, func(t *testing.T) {
			ctx := t.Context()
			got := collect(above(ctx, 10, square(ctx, generate(ctx, 1, 2, 3, 4, 5))))

			if want := []int{16, 25}; !slices.Equal(got, want) {
				t.Fatalf("got %v, want %v", got, want)
			}
		})
	})

	t.Run("one cancel stops every stage when the consumer walks away", func(t *testing.T) {
		synctest.Test(t, func(t *testing.T) {
			ctx, cancel := context.WithCancel(t.Context())
			out := above(ctx, 0, square(ctx, generate(ctx, 1, 2, 3, 4, 5)))

			if got := <-out; got != 1 {
				t.Fatalf("got %d, want 1", got)
			}

			cancel()
			synctest.Wait()

			// The value that was in flight may or may not be delivered; then the channel must close.
			for range out {
			}
		})
	})
}
