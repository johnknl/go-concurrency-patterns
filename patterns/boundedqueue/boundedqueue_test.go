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

package boundedqueue

// Mutants that survive on purpose:

import (
	"slices"
	"testing"
	"testing/synctest"
)

func TestFill(t *testing.T) {
	t.Run("zero values just closes", func(t *testing.T) {
		queue := make(chan int)
		fill(queue, 0)

		if v, ok := <-queue; ok {
			t.Fatalf("received %d, want a closed channel", v)
		}
	})

	for _, n := range []int{1, 3, 4} {
		t.Run("fits in the buffer without a consumer", func(t *testing.T) {
			queue := make(chan int, 4)
			fill(queue, n)

			if len(queue) != n {
				t.Fatalf("len %d, want %d", len(queue), n)
			}

			var got []int
			for v := range queue {
				got = append(got, v)
			}

			if want := slices.Collect(func(yield func(int) bool) {
				for i := range n {
					yield(i)
				}
			}); !slices.Equal(got, want) {
				t.Fatalf("got %v, want %v", got, want)
			}
		})
	}

	t.Run("a full buffer holds the producer back until the consumer reads", func(t *testing.T) {
		synctest.Test(t, func(t *testing.T) {
			queue := make(chan int, 2)
			done := make(chan struct{})

			go func() {
				fill(queue, 4)
				close(done)
			}()

			synctest.Wait()

			if len(queue) != 2 {
				t.Fatalf("len %d, want the producer blocked at cap 2", len(queue))
			}

			select {
			case <-done:
				t.Fatal("fill returned with values left to send")
			default:
			}

			if got := <-queue; got != 0 {
				t.Fatalf("got %d, want 0", got)
			}

			synctest.Wait()

			if len(queue) != 2 {
				t.Fatalf("len %d after one receive, want exactly one more send (2)", len(queue))
			}

			var got []int
			for v := range queue {
				got = append(got, v)
			}

			<-done

			if !slices.Equal(got, []int{1, 2, 3}) {
				t.Fatalf("got %v, want [1 2 3]", got)
			}
		})
	})
}
