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

// Mutants that survive on purpose:
//   - CONDITIONALS_BOUNDARY unbounded.go:38 (n <= m to n < m): when n == m the CAS stores the same value, no change.
//   - buffer-channel unbounded.go:60 (jobs buffered): the producer pulls one more item off the slice, which is not
//     observable through the API; the number of items in work is still limited by the workers.

package unbounded

import (
	"slices"
	"sync"
	"sync/atomic"
	"testing"
	"testing/synctest"
)

func TestPeak(t *testing.T) {
	var p peak
	p.enter()
	p.enter()
	p.leave()
	p.enter()
	p.enter()
	p.leave()
	p.leave()
	p.leave()
	if got := p.max.Load(); got != 3 {
		t.Fatalf("max = %d, want 3", got)
	}
	if got := p.cur.Load(); got != 0 {
		t.Fatalf("cur = %d, want 0", got)
	}
}

// recorder collects the items work was called with.
type recorder struct {
	items []int
	mu    sync.Mutex
}

func (r *recorder) add(i int) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.items = append(r.items, i)
}

func (r *recorder) sorted() []int {
	r.mu.Lock()
	defer r.mu.Unlock()
	return slices.Sorted(slices.Values(r.items))
}

func TestBadFanOut(t *testing.T) {
	t.Run("no items", func(t *testing.T) {
		badFanOut(nil, func(int) { t.Error("work called") })
	})

	t.Run("one goroutine per item", func(t *testing.T) {
		synctest.Test(t, func(t *testing.T) {
			const n = 50
			var (
				inFlight atomic.Int64
				r        recorder
			)
			gate := make(chan struct{})
			done := make(chan struct{})
			go func() {
				badFanOut(slices.Collect(func(yield func(int) bool) {
					for i := range n {
						yield(i)
					}
				}), func(i int) {
					inFlight.Add(1)
					r.add(i)
					<-gate
				})
				close(done)
			}()

			synctest.Wait()
			if got := inFlight.Load(); got != n {
				t.Fatalf("goroutines blocked in work = %d, want %d", got, n)
			}
			close(gate)
			<-done
			if got := r.sorted(); len(got) != n || got[0] != 0 || got[n-1] != n-1 {
				t.Fatalf("items = %v", got)
			}
		})
	})
}

func TestBoundedFanOut(t *testing.T) {
	seq := func(n int) []int {
		s := make([]int, n)
		for i := range s {
			s[i] = i
		}
		return s
	}

	t.Run("no items", func(t *testing.T) {
		synctest.Test(t, func(t *testing.T) {
			boundedFanOut(nil, 4, func(int) { t.Error("work called") })
		})
	})

	for _, tc := range []struct{ items, limit int }{{1, 1}, {10, 1}, {10, 3}, {3, 10}} {
		t.Run("every item exactly once", func(t *testing.T) {
			var r recorder
			boundedFanOut(seq(tc.items), tc.limit, r.add)
			if got := r.sorted(); !slices.Equal(got, seq(tc.items)) {
				t.Fatalf("items=%d limit=%d: processed %v", tc.items, tc.limit, got)
			}
		})
	}

	t.Run("concurrency reaches but never exceeds the limit, producer waits", func(t *testing.T) {
		synctest.Test(t, func(t *testing.T) {
			const (
				n     = 10
				limit = 3
			)
			var (
				p       peak
				started atomic.Int64
			)
			gate := make(chan struct{})
			done := make(chan struct{})
			go func() {
				boundedFanOut(seq(n), limit, func(int) {
					p.enter()
					started.Add(1)
					<-gate
					p.leave()
				})
				close(done)
			}()

			// All workers are busy: the producer is held at the unbuffered send, nothing more starts.
			synctest.Wait()
			if got := started.Load(); got != limit {
				t.Fatalf("started while workers block = %d, want %d", got, limit)
			}
			select {
			case <-done:
				t.Fatal("boundedFanOut returned with work pending")
			default:
			}

			close(gate)
			<-done
			if got := started.Load(); got != n {
				t.Fatalf("started = %d, want %d", got, n)
			}
			if got := p.max.Load(); got != limit {
				t.Fatalf("peak = %d, want %d", got, limit)
			}
		})
	})
}
