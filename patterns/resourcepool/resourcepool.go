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

package resourcepool

import (
	"context"
	"sync"
)

type conn struct{ id int }

// newPool creates a fixed set of expensive resources and puts them all in a buffered channel.
func newPool(n int) chan *conn {
	pool := make(chan *conn, n)

	for id := range n {
		pool <- &conn{id: id}
	}

	return pool
}

// get takes a resource, waiting until one is returned or ctx is done.
func get(ctx context.Context, pool chan *conn) (*conn, error) {
	select {
	case c := <-pool:
		return c, nil
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

// put returns a resource. The channel's capacity equals the number of resources, so it never blocks.
func put(pool chan *conn, c *conn) { pool <- c }

// lockedPool keeps the free resources in a slice guarded by a mutex. A condition variable lets get wait until put
// returns one.
type lockedPool struct {
	cond *sync.Cond
	free []*conn
	mu   sync.Mutex
}

func newLockedPool(n int) *lockedPool {
	p := &lockedPool{free: make([]*conn, 0, n)}
	p.cond = sync.NewCond(&p.mu)

	for id := range n {
		p.free = append(p.free, &conn{id: id})
	}

	return p
}

// get takes the most recently returned resource, waiting until one is free or ctx is done. sync.Cond knows nothing
// of contexts, so context.AfterFunc wakes the waiters when ctx ends; it takes the lock first, so the wake-up cannot
// fall between a waiter's check of ctx.Err and its call to Wait.
func (p *lockedPool) get(ctx context.Context) (*conn, error) {
	stop := context.AfterFunc(ctx, func() {
		p.mu.Lock()
		defer p.mu.Unlock()

		p.cond.Broadcast()
	})
	defer stop()

	p.mu.Lock()
	defer p.mu.Unlock()

	for len(p.free) == 0 {
		if err := ctx.Err(); err != nil {
			return nil, err
		}

		p.cond.Wait()
	}

	c := p.free[len(p.free)-1]
	p.free = p.free[:len(p.free)-1]

	return c, nil
}

// put returns a resource and wakes one waiter.
func (p *lockedPool) put(c *conn) {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.free = append(p.free, c)
	p.cond.Signal()
}
