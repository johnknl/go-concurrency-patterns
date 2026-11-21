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

package bubble

import (
	"errors"
	"sync"
	"time"
)

// errTimeout reports that no value arrived in time.
var errTimeout = errors.New("timed out")

// receive waits at most d for a value from c.
func receive(c <-chan int, d time.Duration) (int, error) {
	timer := time.NewTimer(d)
	defer timer.Stop()

	select {
	case v := <-c:
		return v, nil
	case <-timer.C:
		return 0, errTimeout
	}
}

// sum adds the values from in to *total until in is closed.
func sum(in <-chan int, total *int) {
	for v := range in {
		*total += v
	}
}

// gate blocks callers of wait until open is called.
type gate struct {
	cond *sync.Cond
	mu   sync.Mutex
	open bool
}

func newGate() *gate {
	g := &gate{}
	g.cond = sync.NewCond(&g.mu)

	return g
}

func (g *gate) wait() {
	g.mu.Lock()
	defer g.mu.Unlock()

	for !g.open {
		g.cond.Wait()
	}
}

func (g *gate) release() {
	g.mu.Lock()
	g.open = true
	g.mu.Unlock()

	g.cond.Broadcast()
}
