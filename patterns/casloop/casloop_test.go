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
//   - CONDITIONALS_BOUNDARY casloop.go:33 (x <= old -> x < old): when x == old the mutant swaps old for an equal
//     value, which is indistinguishable from returning.

package casloop

import (
	"sync"
	"sync/atomic"
	"testing"
)

func TestStoreMax(t *testing.T) {
	t.Run("raises to a larger value", func(t *testing.T) {
		var v atomic.Int64
		v.Store(3)
		storeMax(&v, 5)
		if got := v.Load(); got != 5 {
			t.Errorf("v = %d, want 5", got)
		}
	})
	t.Run("keeps a larger or equal value", func(t *testing.T) {
		var v atomic.Int64
		v.Store(5)
		storeMax(&v, 3)
		storeMax(&v, 5)
		if got := v.Load(); got != 5 {
			t.Errorf("v = %d, want 5", got)
		}
	})
	t.Run("negative values", func(t *testing.T) {
		var v atomic.Int64
		v.Store(-10)
		storeMax(&v, -3)
		if got := v.Load(); got != -3 {
			t.Errorf("v = %d, want -3", got)
		}
	})
	t.Run("concurrent callers keep the maximum", func(t *testing.T) {
		// Every goroutine stores a fresh, larger ticket, so CAS conflicts are frequent; a caller that gave up after
		// a failed CAS would lose the last ticket now and then.
		for range 50 {
			var v, ticket atomic.Int64
			var wg sync.WaitGroup
			for range 8 {
				wg.Go(func() {
					for range 200 {
						storeMax(&v, ticket.Add(1))
					}
				})
			}
			wg.Wait()
			if got, want := v.Load(), ticket.Load(); got != want {
				t.Fatalf("v = %d, want the largest ticket %d", got, want)
			}
		}
	})
}

func TestTryIncrement(t *testing.T) {
	t.Run("counts up to the limit", func(t *testing.T) {
		var n atomic.Int64
		for i := range 3 {
			if !tryIncrement(&n, 3) {
				t.Fatalf("increment %d refused below the limit", i+1)
			}
		}
		if tryIncrement(&n, 3) {
			t.Fatal("increment past the limit allowed")
		}
		if got := n.Load(); got != 3 {
			t.Errorf("n = %d, want 3", got)
		}
	})
	t.Run("zero limit", func(t *testing.T) {
		var n atomic.Int64
		if tryIncrement(&n, 0) {
			t.Fatal("increment allowed with limit 0")
		}
		if got := n.Load(); got != 0 {
			t.Errorf("n = %d, want 0", got)
		}
	})
	t.Run("concurrent callers never exceed the limit", func(t *testing.T) {
		const limit = 100
		var n, granted atomic.Int64
		var wg sync.WaitGroup
		for range 16 {
			wg.Go(func() {
				for range 20 {
					if tryIncrement(&n, limit) {
						granted.Add(1)
					}
				}
			})
		}
		wg.Wait()
		if got := n.Load(); got != limit {
			t.Errorf("n = %d, want %d", got, limit)
		}
		if got := granted.Load(); got != limit {
			t.Errorf("%d increments granted, want %d", got, limit)
		}
	})
}

func TestRecord(t *testing.T) {
	t.Run("one value", func(t *testing.T) {
		var p atomic.Pointer[stats]
		p.Store(&stats{})
		record(&p, 7)
		if got := *p.Load(); got != (stats{count: 1, sum: 7}) {
			t.Errorf("stats = %+v, want {count:1 sum:7}", got)
		}
	})
	t.Run("does not modify the published value", func(t *testing.T) {
		var p atomic.Pointer[stats]
		before := &stats{count: 2, sum: 5}
		p.Store(before)
		record(&p, 1)
		if *before != (stats{count: 2, sum: 5}) {
			t.Errorf("old stats changed to %+v", *before)
		}
		if got := *p.Load(); got != (stats{count: 3, sum: 6}) {
			t.Errorf("stats = %+v, want {count:3 sum:6}", got)
		}
	})
	t.Run("concurrent callers lose no update", func(t *testing.T) {
		var p atomic.Pointer[stats]
		p.Store(&stats{})
		var wg sync.WaitGroup
		for i := range 1000 {
			wg.Go(func() { record(&p, int64(i)) })
		}
		wg.Wait()
		if got := *p.Load(); got != (stats{count: 1000, sum: 999 * 1000 / 2}) {
			t.Errorf("stats = %+v, want {count:1000 sum:499500}", got)
		}
	})
}
