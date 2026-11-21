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
	"sync/atomic"
)

// mailbox hands values to out without ever blocking the caller. Each Put parks a goroutine on the send, so a
// pending value costs a goroutine instead of a slot in a channel.
type mailbox struct {
	out    chan int
	parked atomic.Int64
}

func (m *mailbox) Put(v int) {
	m.parked.Add(1)

	go func() {
		defer m.parked.Add(-1)

		m.out <- v
	}()
}

// queue is a bounded FIFO: a buffered channel. Put never starts a goroutine. When the buffer is full the caller
// decides what to do: here it learns that the value was not accepted and can shed it, or block by using a plain
// send instead.
type queue struct {
	items   chan int
	dropped int
}

func newQueue(capacity int) *queue {
	return &queue{items: make(chan int, capacity)}
}

func (q *queue) Put(v int) bool {
	select {
	case q.items <- v:
		return true
	default:
		q.dropped++

		return false
	}
}
