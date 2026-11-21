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

package copiedmutex

import (
	"fmt"
	"sync"
)

func Example_copyHasItsOwnLock() {
	c := &counter{}
	cp := copyOf(c) // what a method with receiver (c counter) gets
	cp.mu.Lock()
	cp.n++
	fmt.Println("original lock free while the copy is held:", c.mu.TryLock())
	c.mu.Unlock()
	cp.mu.Unlock()
	fmt.Println("original n:", c.n)
	// Output:
	// original lock free while the copy is held: true
	// original n: 0
}

func Example_copyOfLockedMutexIsLocked() {
	c := &counter{}
	c.mu.Lock()
	cp := copyOf(c) // copied while locked
	c.mu.Unlock()
	fmt.Println("copy free after original unlocked:", cp.mu.TryLock())
	// Output:
	// copy free after original unlocked: false
}

func Example_pointerReceiver() {
	counters := []*safeCounter{{}, {}}
	var wg sync.WaitGroup
	for range 100 {
		for _, c := range counters { // c is a *safeCounter, the mutex is not copied
			wg.Go(c.Inc)
		}
	}
	wg.Wait()
	fmt.Println(counters[0].n, counters[1].n)
	// Output:
	// 100 100
}
