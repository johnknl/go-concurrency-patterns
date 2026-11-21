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

package onlineattach

import (
	"context"
	"errors"
	"fmt"
)

var (
	errDetached = errors.New("detached")
	errStopped  = errors.New("hub stopped")
)

// sub is one subscriber. The hub's owner goroutine writes err before it closes ch, so err can be read after
// ch is closed.
type sub struct {
	ch  chan int
	err error
}

type hub struct {
	pub     chan int
	attach  chan *sub
	detach  chan *sub
	done    chan struct{}
	dropped int // owned by run, read after done is closed
}

func newHub(ctx context.Context) *hub {
	h := &hub{
		pub:    make(chan int),
		attach: make(chan *sub),
		detach: make(chan *sub),
		done:   make(chan struct{}),
	}

	go h.run(ctx)

	return h
}

// run is the owner: the subscriber set and the history are only touched here.
func (h *hub) run(ctx context.Context) {
	defer close(h.done)

	subs := map[*sub]bool{}

	var history []int

	for {
		select {
		case msg := <-h.pub:
			history = append(history, msg)
			history = history[max(0, len(history)-2):]

			for s := range subs {
				h.deliver(s, msg)
			}
		case s := <-h.attach:
			for _, msg := range history {
				h.deliver(s, msg)
			}

			subs[s] = true
		case s := <-h.detach:
			if subs[s] {
				delete(subs, s)

				s.err = errDetached
				close(s.ch)
			}
		case <-ctx.Done():
			for s := range subs {
				s.err = context.Cause(ctx)
				close(s.ch)
			}

			return
		}
	}
}

// deliver never blocks the owner: a full subscriber loses the message.
func (h *hub) deliver(s *sub, msg int) {
	select {
	case s.ch <- msg:
	default:
		h.dropped++
	}
}

// subscribe asks the owner to attach a new subscriber. It returns once the owner has received the request, and the
// owner attaches it before handling anything else, so the subscriber is offered the history and then exactly the
// messages published after this point.
func (h *hub) subscribe(buffer int) *sub {
	s := &sub{ch: make(chan int, buffer)}

	select {
	case h.attach <- s:
	case <-h.done:
		s.err = errStopped
		close(s.ch)
	}

	return s
}

// unsubscribe asks the owner to detach s. The owner closes the channel, so it is never closed while a send
// to it is in progress.
func (h *hub) unsubscribe(s *sub) {
	select {
	case h.detach <- s:
	case <-h.done:
	}
}

func (h *hub) publish(msg int) {
	select {
	case h.pub <- msg:
	case <-h.done:
	}
}

func drain(name string, s *sub) string {
	var got []int
	for msg := range s.ch {
		got = append(got, msg)
	}

	return fmt.Sprint(name, " ", got, " ", s.err)
}
