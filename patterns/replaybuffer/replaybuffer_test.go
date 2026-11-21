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

package replaybuffer

// Mutants that survive on purpose:

import (
	"slices"
	"sync"
	"testing"
	"testing/synctest"
)

func TestRing(t *testing.T) {
	for _, tc := range []struct {
		name string
		size int
		add  []int
		want []int
	}{
		{"empty", 3, nil, nil},
		{"fewer than size", 3, []int{1, 2}, []int{1, 2}},
		{"exactly size", 3, []int{1, 2, 3}, []int{1, 2, 3}},
		{"one over size drops the oldest", 3, []int{1, 2, 3, 4}, []int{2, 3, 4}},
		{"wraps many times", 3, []int{1, 2, 3, 4, 5, 6, 7, 8}, []int{6, 7, 8}},
		{"size one keeps the newest", 1, []int{1, 2, 3}, []int{3}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			r := newRing(tc.size)
			for _, v := range tc.add {
				r.add(v)
			}

			if got := slices.Collect(r.all()); !slices.Equal(got, tc.want) {
				t.Fatalf("got %v, want %v", got, tc.want)
			}
		})
	}

	t.Run("break stops the replay", func(t *testing.T) {
		r := newRing(3)
		r.add(1)
		r.add(2)

		for v := range r.all() {
			if v != 1 {
				t.Fatalf("got %d, want only 1", v)
			}

			break
		}
	})
}

func collect(ch <-chan int) []int {
	var got []int
	for v := range ch {
		got = append(got, v)
	}

	return got
}

func TestOwner(t *testing.T) {
	t.Run("early subscriber sees only live messages", func(t *testing.T) {
		synctest.Test(t, func(t *testing.T) {
			in, attach := make(chan int), make(chan chan int)
			go owner(in, attach, 3)

			sub := make(chan int, 8)
			attach <- sub

			for v := 1; v <= 4; v++ {
				in <- v
			}

			close(in)

			if got := collect(sub); !slices.Equal(got, []int{1, 2, 3, 4}) {
				t.Fatalf("got %v", got)
			}
		})
	})

	t.Run("late subscriber gets the last n, then live, for every subscriber", func(t *testing.T) {
		synctest.Test(t, func(t *testing.T) {
			in, attach := make(chan int), make(chan chan int)
			go owner(in, attach, 2)

			in <- 1
			a := make(chan int, 8)
			attach <- a

			in <- 2
			in <- 3
			b := make(chan int, 8)
			attach <- b

			in <- 4
			close(in)

			if got := collect(a); !slices.Equal(got, []int{1, 2, 3, 4}) {
				t.Fatalf("a got %v, want [1 2 3 4]", got)
			}

			if got := collect(b); !slices.Equal(got, []int{2, 3, 4}) {
				t.Fatalf("b got %v, want [2 3 4]", got)
			}
		})
	})

	t.Run("close without subscribers ends the owner", func(t *testing.T) {
		synctest.Test(t, func(_ *testing.T) {
			in := make(chan int)
			go owner(in, make(chan chan int), 1)

			in <- 1
			close(in)
		})
	})

	t.Run("a slow subscriber holds the owner back", func(t *testing.T) {
		synctest.Test(t, func(t *testing.T) {
			in, attach := make(chan int), make(chan chan int)
			go owner(in, attach, 2)

			sub := make(chan int) // unbuffered: every delivery waits for this reader
			attach <- sub
			in <- 1
			synctest.Wait()

			select {
			case in <- 2:
				t.Fatal("owner took a new message before the subscriber read the previous one")
			default:
			}

			if got := <-sub; got != 1 {
				t.Fatalf("got %d, want 1", got)
			}

			close(in)

			if v, ok := <-sub; ok {
				t.Fatalf("received %d, want a closed channel", v)
			}
		})
	})

	t.Run("attaching while publishing gives no gap and no duplicate", func(t *testing.T) {
		const n, keep = 200, 4

		in, attach := make(chan int), make(chan chan int)

		var wg sync.WaitGroup
		wg.Go(func() { owner(in, attach, keep) })

		subs := make([]chan int, 20)
		for i := range subs {
			subs[i] = make(chan int, n+keep)
		}

		attached := make(chan struct{})

		wg.Go(func() {
			for v := 1; v <= n; v++ {
				in <- v
			}

			<-attached
			close(in)
		})

		for _, s := range subs {
			attach <- s
		}

		close(attached)
		wg.Wait()

		for i, s := range subs {
			got := collect(s)
			if len(got) == 0 || got[len(got)-1] != n {
				t.Fatalf("sub %d: last %d, want %d", i, got[len(got)-1], n)
			}

			for j := 1; j < len(got); j++ {
				if got[j] != got[j-1]+1 {
					t.Fatalf("sub %d: gap or duplicate in %v", i, got)
				}
			}
		}
	})
}
