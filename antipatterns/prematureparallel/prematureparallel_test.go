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
//

package prematureparallel

// Mutants that survive on purpose:
//   - buffer-channel on jobs and results: a buffer changes how the handoffs overlap, not their number or the
//     sorted result.

import (
	"slices"
	"sync"
	"testing"
	"testing/synctest"
)

func TestSquaresParallel(t *testing.T) {
	for _, workers := range []int{1, 3} {
		out, g, h := squaresParallel([]int{3, 1, 2}, workers)
		if !slices.Equal(out, []int{1, 4, 9}) {
			t.Fatalf("got %v", out)
		}

		if g != workers+2 || h != 6 {
			t.Fatalf("goroutines %d, handoffs %d", g, h)
		}
	}

	if out, _, h := squaresParallel(nil, 2); len(out) != 0 || h != 0 {
		t.Fatalf("got %v, %d", out, h)
	}
}

func TestSquares(t *testing.T) {
	if got := squares([]int{3, -1, 2}); !slices.Equal(got, []int{9, 1, 4}) {
		t.Fatalf("got %v", got)
	}
}

func TestMapBounded(t *testing.T) {
	t.Run("keeps the order", func(t *testing.T) {
		got := mapBounded([]int{4, 2, 3, 1}, 2, func(n int) int { return n * 10 })
		if !slices.Equal(got, []int{40, 20, 30, 10}) {
			t.Fatalf("got %v", got)
		}
	})

	t.Run("runs at most limit at a time", func(t *testing.T) {
		synctest.Test(t, func(t *testing.T) {
			var (
				mu            sync.Mutex
				running, peak int
			)

			release := make(chan struct{})
			done := make(chan []int)

			go func() {
				done <- mapBounded(make([]int, 5), 2, func(int) int {
					mu.Lock()
					running++
					peak = max(peak, running)
					mu.Unlock()

					<-release

					mu.Lock()
					running--
					mu.Unlock()

					return 1
				})
			}()

			synctest.Wait()

			mu.Lock()
			if running != 2 {
				t.Errorf("%d running, want 2", running)
			}
			mu.Unlock()

			close(release)

			if got := <-done; !slices.Equal(got, []int{1, 1, 1, 1, 1}) {
				t.Fatalf("got %v", got)
			}

			if peak != 2 {
				t.Fatalf("peak %d, want 2", peak)
			}
		})
	})
}
