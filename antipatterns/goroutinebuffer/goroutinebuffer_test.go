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

// Mutants that survive on purpose: none.

package goroutinebuffer

import (
	"slices"
	"testing"
	"testing/synctest"
)

func TestMailbox_Put(t *testing.T) {
	t.Run("never blocks and parks one goroutine per pending value", func(t *testing.T) {
		synctest.Test(t, func(t *testing.T) {
			const n = 100
			m := &mailbox{out: make(chan int)}

			for i := range n {
				m.Put(i)
			}
			synctest.Wait()
			if got := m.parked.Load(); got != n {
				t.Fatalf("parked = %d, want %d", got, n)
			}

			got := make([]int, 0, n)
			for range n {
				got = append(got, <-m.out)
			}
			slices.Sort(got)
			for i, v := range got {
				if v != i {
					t.Fatalf("values = %v, want each of 0..%d once", got, n-1)
				}
			}

			synctest.Wait()
			if p := m.parked.Load(); p != 0 {
				t.Fatalf("parked after drain = %d, want 0", p)
			}
		})
	})
}

func TestQueue_Put(t *testing.T) {
	t.Run("accepts up to capacity, then sheds", func(t *testing.T) {
		q := newQueue(3)
		for i := range 3 {
			if !q.Put(i) {
				t.Fatalf("Put(%d) rejected below capacity", i)
			}
		}
		if q.dropped != 0 {
			t.Fatalf("dropped = %d, want 0", q.dropped)
		}
		for i := 3; i < 5; i++ {
			if q.Put(i) {
				t.Fatalf("Put(%d) accepted on a full queue", i)
			}
		}
		if q.dropped != 2 {
			t.Fatalf("dropped = %d, want 2", q.dropped)
		}
		for i := range 3 {
			if v := <-q.items; v != i {
				t.Fatalf("received %d, want %d (FIFO)", v, i)
			}
		}
		if !q.Put(9) {
			t.Fatal("Put rejected after draining")
		}
	})

	t.Run("zero capacity drops everything", func(t *testing.T) {
		q := newQueue(0)
		if q.Put(1) || q.dropped != 1 {
			t.Fatalf("Put accepted or dropped=%d", q.dropped)
		}
	})

	t.Run("starts no goroutine", func(t *testing.T) {
		synctest.Test(t, func(_ *testing.T) {
			q := newQueue(1)
			q.Put(1)
			q.Put(2)
			// synctest fails the test if a goroutine is still running at the end of the bubble.
		})
	})
}
