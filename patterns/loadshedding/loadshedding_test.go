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

package loadshedding

// Mutants that survive on purpose:
// - remove-default loadshedding.go:48: the eviction select only finds nothing to take when a receiver empties the
//   queue between the two selects, or on an unbuffered queue before a receiver arrives. Neither interleaving can be
//   forced without sleeps; the unbuffered subtest kills the mutant only when dropOldest runs before the receiver.

import (
	"slices"
	"sync"
	"testing"
	"time"
)

func TestDropNewest(t *testing.T) {
	t.Run("queues while there is room", func(t *testing.T) {
		queue := make(chan int, 2)

		if !dropNewest(queue, 1) || !dropNewest(queue, 2) {
			t.Fatal("value dropped although the queue had room")
		}

		if got := drain(queue); !slices.Equal(got, []int{1, 2}) {
			t.Fatalf("queue = %v, want [1 2]", got)
		}
	})

	t.Run("drops the new value when full", func(t *testing.T) {
		queue := make(chan int, 1)
		dropNewest(queue, 1)

		if dropNewest(queue, 2) {
			t.Fatal("value queued on a full queue")
		}

		if got := drain(queue); !slices.Equal(got, []int{1}) {
			t.Fatalf("queue = %v, want [1]", got)
		}
	})

	t.Run("unbuffered queue without a receiver drops", func(t *testing.T) {
		if dropNewest(make(chan int), 1) {
			t.Fatal("value queued without a receiver")
		}
	})
}

func TestDropOldest(t *testing.T) {
	t.Run("queues without dropping while there is room", func(t *testing.T) {
		queue := make(chan int, 2)

		if dropOldest(queue, 1) || dropOldest(queue, 2) {
			t.Fatal("reported a drop although the queue had room")
		}

		if got := drain(queue); !slices.Equal(got, []int{1, 2}) {
			t.Fatalf("queue = %v, want [1 2]", got)
		}
	})

	t.Run("evicts the head when full", func(t *testing.T) {
		queue := make(chan int, 2)
		dropOldest(queue, 1)
		dropOldest(queue, 2)

		if !dropOldest(queue, 3) {
			t.Fatal("no drop reported on a full queue")
		}

		if got := drain(queue); !slices.Equal(got, []int{2, 3}) {
			t.Fatalf("queue = %v, want [2 3]", got)
		}
	})

	t.Run("does not block waiting for a value on an unbuffered queue", func(t *testing.T) {
		// There is never a value to evict, so dropOldest retries until a receiver takes v.
		queue := make(chan int)
		done := make(chan bool)

		go func() { done <- dropOldest(queue, 7) }()

		if got := <-queue; got != 7 {
			t.Fatalf("received %d, want 7", got)
		}

		select {
		case dropped := <-done:
			if dropped {
				t.Fatal("reported a drop on an unbuffered queue")
			}
		case <-time.After(5 * time.Second):
			t.Fatal("dropOldest did not return")
		}
	})

	t.Run("concurrent senders keep the queue full and never block", func(t *testing.T) {
		queue := make(chan int, 3)

		var wg sync.WaitGroup
		for g := range 8 {
			wg.Go(func() {
				for i := range 100 {
					dropOldest(queue, g*100+i)
				}
			})
		}

		wg.Wait()

		if got := len(drain(queue)); got != 3 {
			t.Fatalf("kept %d values, want 3", got)
		}
	})
}

func TestDrain(t *testing.T) {
	t.Run("empty", func(t *testing.T) {
		if got := drain(make(chan int, 1)); got != nil {
			t.Fatalf("got %v, want nil", got)
		}
	})

	t.Run("returns values in queue order and closes the queue", func(t *testing.T) {
		queue := make(chan int, 3)
		queue <- 3
		queue <- 1
		queue <- 2

		if got := drain(queue); !slices.Equal(got, []int{3, 1, 2}) {
			t.Fatalf("got %v, want [3 1 2]", got)
		}

		if _, ok := <-queue; ok {
			t.Fatal("queue not closed")
		}
	})
}
