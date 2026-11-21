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

// Mutants that survive on purpose: none.

package oversizedbuffer

import "testing"

func TestBadProduce(t *testing.T) {
	for _, n := range []int{0, 1, 100} {
		queue := badProduce(n)
		if len(queue) != n || cap(queue) != n {
			t.Fatalf("n=%d: len=%d cap=%d, want both %d", n, len(queue), cap(queue), n)
		}
		// FIFO: item i waits behind the i items before it.
		for i := range n {
			if v := <-queue; v != i {
				t.Fatalf("n=%d: received %d, want %d", n, v, i)
			}
		}
	}
}

func TestShed(t *testing.T) {
	tests := []struct {
		name              string
		n, capacity       int
		accepted, dropped int
	}{
		{"nothing offered", 0, 5, 0, 0},
		{"unbuffered without receiver drops all", 5, 0, 0, 5},
		{"fits", 3, 5, 3, 0},
		{"exactly full", 5, 5, 5, 0},
		{"overflow is dropped", 10, 4, 4, 6},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			queue := make(chan int, tc.capacity)
			accepted, dropped := shed(tc.n, queue)
			if accepted != tc.accepted || dropped != tc.dropped {
				t.Fatalf("shed = (%d, %d), want (%d, %d)", accepted, dropped, tc.accepted, tc.dropped)
			}
			// The queue never grows past its bound and keeps the oldest items in order.
			if len(queue) != tc.accepted {
				t.Fatalf("queued %d, want %d", len(queue), tc.accepted)
			}
			for i := range tc.accepted {
				if v := <-queue; v != i {
					t.Fatalf("received %d, want %d", v, i)
				}
			}
		})
	}

	t.Run("never blocks on a full queue", func(t *testing.T) {
		queue := make(chan int, 1)
		queue <- -1
		if accepted, dropped := shed(3, queue); accepted != 0 || dropped != 3 {
			t.Fatalf("shed = (%d, %d), want (0, 3)", accepted, dropped)
		}
	})
}
