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

package condvar

import (
	"context"
	"fmt"
)

// Example_budget has two callers waiting for different amounts. A release that covers only the smaller one lets it
// through and keeps the other waiting.
func Example_budget() {
	ctx := context.Background()
	b := newBudget(9)
	_ = b.acquire(ctx, 9) // everything is in use

	got := make(chan int)
	for _, n := range []int{6, 3} {
		go func() {
			if b.acquire(ctx, n) == nil {
				got <- n
			}
		}()
	}

	b.release(4) // enough for 3, not for 6
	fmt.Println("got", <-got)

	b.release(5) // 1 + 5 = 6
	fmt.Println("got", <-got)

	// Output:
	// got 3
	// got 6
}

// Example_cancel ends a wait that no release would end.
func Example_cancel() {
	b := newBudget(1)
	_ = b.acquire(context.Background(), 1)

	ctx, cancel := context.WithCancelCause(context.Background())
	done := make(chan error)

	go func() { done <- b.acquire(ctx, 1) }()

	cancel(fmt.Errorf("caller gave up"))
	fmt.Println(<-done)

	// Output:
	// caller gave up
}

// Example_watch wakes a watcher on every change, which closing a channel can do only once.
func Example_watch() {
	w := newWatched()
	ack := make(chan struct{})

	go func() {
		var seen uint64
		for range 3 {
			var v int
			v, seen = w.next(seen)
			fmt.Println("saw", v)
			ack <- struct{}{}
		}
	}()

	for v := range 3 {
		w.set(v + 1)
		<-ack
	}

	// Output:
	// saw 1
	// saw 2
	// saw 3
}
