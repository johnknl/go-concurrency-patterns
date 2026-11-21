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

import (
	"iter"
)

// ring keeps the last len(items) values. It has no lock: only one goroutine uses it.
type ring struct {
	items []int
	next  int
	count int
}

func newRing(n int) *ring { return &ring{items: make([]int, n)} }

func (r *ring) add(v int) {
	r.items[r.next] = v
	r.next = (r.next + 1) % len(r.items)
	r.count = min(r.count+1, len(r.items))
}

// all yields the stored values, oldest first.
func (r *ring) all() iter.Seq[int] {
	return func(yield func(int) bool) {
		start := (r.next - r.count + len(r.items)) % len(r.items)

		for i := range r.count {
			if !yield(r.items[(start+i)%len(r.items)]) {
				return
			}
		}
	}
}

// owner is the only goroutine that touches the ring and the subscribers. A new subscriber is replayed and
// added to the set between two messages, so it sees no gap and no duplicate.
func owner(in <-chan int, attach <-chan chan int, keep int) {
	history := newRing(keep)

	var subs []chan int

	for {
		select {
		case v, ok := <-in:
			if !ok {
				for _, s := range subs {
					close(s)
				}

				return
			}

			history.add(v)

			for _, s := range subs {
				s <- v
			}
		case s := <-attach:
			for v := range history.all() {
				s <- v
			}

			subs = append(subs, s)
		}
	}
}
