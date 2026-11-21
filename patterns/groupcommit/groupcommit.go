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

package groupcommit

import "sync"

// batch is one group of items that is flushed together. Its callers read err once done is set.
type batch struct {
	err   error
	items []string
	done  bool
}

// committer combines concurrent commits into batches and flushes one batch at a time.
type committer struct {
	flush    func([]string) error
	open     *batch
	cond     sync.Cond
	maxBatch int
	mu       sync.Mutex
	flushing bool
}

func newCommitter(maxBatch int, flush func([]string) error) *committer {
	c := &committer{flush: flush, maxBatch: maxBatch, open: &batch{}}
	c.cond.L = &c.mu

	return c
}

// commit returns when the batch that holds item has been flushed, with the error of that flush.
func (c *committer) commit(item string) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	for len(c.open.items) >= c.maxBatch {
		c.cond.Wait() // the open batch is full: wait until it has been taken
	}

	b := c.open
	b.items = append(b.items, item)

	for c.flushing && !b.done {
		c.cond.Wait()
	}

	if b.done {
		return b.err // another caller led this batch
	}

	// Nobody is flushing: lead this batch, and only this one.
	c.flushing = true
	c.open = &batch{}
	c.mu.Unlock()

	err := c.flush(b.items)

	c.mu.Lock()
	b.err, b.done = err, true
	c.flushing = false
	c.cond.Broadcast()

	return err
}
