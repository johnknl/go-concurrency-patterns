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

package pubsub

import (
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
)

var (
	errUnsubscribed = errors.New("unsubscribed")
	errClosed       = errors.New("hub closed")
)

// sub is one subscriber. err is written under the hub's lock before ch is closed, so it can be read once ch is
// closed.
type sub struct {
	ch  chan int
	err error
}

// hub is a registry of subscribers guarded by a mutex. Publishers deliver from their own goroutines.
type hub struct {
	subs    map[*sub]struct{}
	dropped atomic.Int64
	mu      sync.RWMutex
	closed  bool
}

func newHub() *hub {
	return &hub{subs: map[*sub]struct{}{}}
}

func (h *hub) subscribe(buffer int) *sub {
	s := &sub{ch: make(chan int, buffer)}

	h.mu.Lock()
	defer h.mu.Unlock()

	if h.closed {
		s.err = errClosed
		close(s.ch)

		return s
	}

	h.subs[s] = struct{}{}

	return s
}

// unsubscribe holds the write lock, so no publisher is sending to s while its channel is closed.
func (h *hub) unsubscribe(s *sub) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if _, ok := h.subs[s]; !ok {
		return
	}

	delete(h.subs, s)
	s.err = errUnsubscribed
	close(s.ch)
}

// publish copies msg to every subscriber. It holds the read lock for the whole delivery, so the delivery must not
// block: a full subscriber loses the message.
func (h *hub) publish(msg int) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	for s := range h.subs {
		select {
		case s.ch <- msg:
		default:
			h.dropped.Add(1)
		}
	}
}

func (h *hub) close() {
	h.mu.Lock()
	defer h.mu.Unlock()

	h.closed = true

	for s := range h.subs {
		s.err = errClosed
		close(s.ch)
	}

	clear(h.subs)
}

func drain(name string, s *sub) string {
	var got []int
	for msg := range s.ch {
		got = append(got, msg)
	}

	return fmt.Sprint(name, " ", got, " ", s.err)
}
