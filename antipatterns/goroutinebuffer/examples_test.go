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

package goroutinebuffer

import (
	"fmt"
)

// Example_parkedGoroutines shows that the backlog is held by goroutines, one per pending value, and that
// nothing slows the producer down.
func Example_parkedGoroutines() {
	m := &mailbox{out: make(chan int)}

	for i := range 1000 {
		m.Put(i)
	}

	fmt.Println("pending values held by goroutines:", m.parked.Load())

	for range 1000 {
		<-m.out
	}

	// Output:
	// pending values held by goroutines: 1000
}

// Example_lostOrder shows that the order in which parked goroutines reach the channel is decided by the
// scheduler, not by the order of the calls. Here the goroutine of value 1 is held back by a gate, the way the
// scheduler may delay it, and value 2 overtakes it.
func Example_lostOrder() {
	out := make(chan int)
	gate := make(chan struct{})

	go func() {
		<-gate

		out <- 1
	}()

	go func() {
		out <- 2
	}()

	fmt.Println("first received:", <-out)

	close(gate)

	fmt.Println("second received:", <-out)

	// Output:
	// first received: 2
	// second received: 1
}

// Example_boundedQueue shows the same hand-off with a buffered channel: the backlog is visible, bounded, and in
// order, and no goroutine is parked for it.
func Example_boundedQueue() {
	q := newQueue(3)

	for i := range 5 {
		q.Put(i)
	}

	fmt.Println("queued:", len(q.items), "of", cap(q.items), "dropped:", q.dropped)

	close(q.items)

	for v := range q.items {
		fmt.Println("received:", v)
	}

	// Output:
	// queued: 3 of 3 dropped: 2
	// received: 0
	// received: 1
	// received: 2
}
