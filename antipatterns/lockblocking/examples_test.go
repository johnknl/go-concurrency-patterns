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

package lockblocking

import (
	"fmt"
	"sync"
)

func Example_lockHeldDuringSend() {
	s := &badStore{value: 42}
	out := make(chan int) // unbuffered: the send blocks until someone receives
	started := make(chan struct{})
	var wg sync.WaitGroup
	wg.Go(func() {
		close(started)
		s.Publish(out)
	})
	<-started

	// Retry until the lock is held by Publish. It is released only after the send completes, so a
	// successful TryLock would mean Publish has not locked yet, and the loop tries again.
	for s.mu.TryLock() {
		s.mu.Unlock()
	}
	fmt.Println("lock held while the send is blocked")

	fmt.Println("received:", <-out)
	wg.Wait()
	fmt.Println("lock free after send:", s.mu.TryLock())
	// Output:
	// lock held while the send is blocked
	// received: 42
	// lock free after send: true
}

func Example_deliverAfterUnlock() {
	s := &store{value: 42}
	inDeliver := make(chan int)
	release := make(chan struct{})
	var wg sync.WaitGroup
	wg.Go(func() {
		s.Publish(func(v int) {
			inDeliver <- v
			<-release // stands in for a slow send, I/O or callback
		})
	})

	fmt.Println("delivering:", <-inDeliver)
	// Publish is blocked inside deliver, and the lock is free.
	fmt.Println("lock free while delivery is blocked:", s.mu.TryLock())
	s.mu.Unlock()
	close(release)
	wg.Wait()
	// Output:
	// delivering: 42
	// lock free while delivery is blocked: true
}
