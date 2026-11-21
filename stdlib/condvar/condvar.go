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
	"errors"
	"sync"
)

// errTooLarge reports a request that the budget could never grant.
var errTooLarge = errors.New("request exceeds the budget")

// budget hands out units of a shared capacity, such as bytes of memory. Each caller waits for its own amount.
type budget struct {
	cond      *sync.Cond
	available int
	capacity  int
	mu        sync.Mutex
}

func newBudget(capacity int) *budget {
	b := &budget{available: capacity, capacity: capacity}
	b.cond = sync.NewCond(&b.mu)

	return b
}

// acquire waits until n units are available and takes them, or returns the context's cause.
func (b *budget) acquire(ctx context.Context, n int) error {
	if n > b.capacity {
		return errTooLarge
	}

	// Wait cannot select on ctx.Done, so cancellation wakes every waiter, and each checks its context.
	stop := context.AfterFunc(ctx, func() {
		b.mu.Lock()
		defer b.mu.Unlock()

		b.cond.Broadcast()
	})
	defer stop()

	b.mu.Lock()
	defer b.mu.Unlock()

	for b.available < n {
		if ctx.Err() != nil {
			return context.Cause(ctx)
		}

		b.cond.Wait()
	}

	b.available -= n

	return nil
}

// release returns n units. Waiters need different amounts, so all of them recheck.
func (b *budget) release(n int) {
	b.mu.Lock()
	b.available += n
	b.mu.Unlock()

	b.cond.Broadcast()
}

// watched is a value that changes many times; watchers wait for the next change.
type watched struct {
	cond    *sync.Cond
	value   int
	version uint64
	mu      sync.Mutex
}

func newWatched() *watched {
	w := &watched{}
	w.cond = sync.NewCond(&w.mu)

	return w
}

// set stores v as a new version and wakes every watcher.
func (w *watched) set(v int) {
	w.mu.Lock()
	w.value = v
	w.version++
	w.mu.Unlock()

	w.cond.Broadcast()
}

// next waits until the version differs from seen, and returns the current value and version.
func (w *watched) next(seen uint64) (int, uint64) {
	w.mu.Lock()
	defer w.mu.Unlock()

	for w.version == seen {
		w.cond.Wait()
	}

	return w.value, w.version
}
