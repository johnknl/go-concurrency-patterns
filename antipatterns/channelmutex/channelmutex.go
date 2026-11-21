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
	"sync"
	"sync/atomic"
)

// ownedCounter guards a count with an owner goroutine. Every call is a round trip through a channel, and the
// goroutine has to be started and stopped.
type ownedCounter struct {
	incr chan struct{}
	get  chan chan int
	done chan struct{}
	quit chan struct{}
}

func newOwnedCounter() *ownedCounter {
	c := &ownedCounter{
		incr: make(chan struct{}),
		get:  make(chan chan int),
		done: make(chan struct{}),
		quit: make(chan struct{}),
	}

	go c.loop()

	return c
}

func (c *ownedCounter) loop() {
	defer close(c.done)

	n := 0

	for {
		select {
		case <-c.incr:
			n++
		case reply := <-c.get:
			reply <- n
		case <-c.quit:
			return
		}
	}
}

func (c *ownedCounter) inc() { c.incr <- struct{}{} }

func (c *ownedCounter) value() int {
	reply := make(chan int)
	c.get <- reply

	return <-reply
}

// stop ends the owner goroutine and waits for it. A call to inc or value after stop blocks forever.
func (c *ownedCounter) stop() {
	close(c.quit)
	<-c.done
}

// counter is the fix: the mutex sits next to the state it guards, and there is nothing to start or stop.
type counter struct {
	mu sync.Mutex
	n  int
}

func (c *counter) inc() {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.n++
}

func (c *counter) value() int {
	c.mu.Lock()
	defer c.mu.Unlock()

	return c.n
}

// atomicCounter is the fix for a single number.
type atomicCounter struct {
	n atomic.Int64
}

func (c *atomicCounter) inc()         { c.n.Add(1) }
func (c *atomicCounter) value() int64 { return c.n.Load() }
