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
//

package channelmutex

import (
	"fmt"
	"sync"
)

// Example_ownerGoroutine counts with an owner goroutine. It works, but each call is a channel round trip, the
// goroutine must be stopped, and a call after stop would block forever.
func Example_ownerGoroutine() {
	c := newOwnedCounter()

	var wg sync.WaitGroup
	for range 100 {
		wg.Go(c.inc)
	}

	wg.Wait()
	fmt.Println("count:", c.value())

	c.stop()

	select {
	case c.incr <- struct{}{}:
		fmt.Println("inc after stop: accepted")
	default:
		fmt.Println("inc after stop: would block forever")
	}

	// Output:
	// count: 100
	// inc after stop: would block forever
}

// Example_mutex counts with a mutex and with an atomic. Nothing is started, so nothing has to be stopped.
func Example_mutex() {
	var (
		c  counter
		a  atomicCounter
		wg sync.WaitGroup
	)

	for range 100 {
		wg.Go(func() {
			c.inc()
			a.inc()
		})
	}

	wg.Wait()
	fmt.Println("mutex:", c.value(), "atomic:", a.value())

	// Output:
	// mutex: 100 atomic: 100
}
