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

package eviction

import (
	"errors"
	"fmt"
	"slices"
	"time"
)

var errTooSlow = errors.New("too many missed deliveries")

type subscriber struct {
	err     error
	ch      chan int
	name    string
	strikes int
}

// timedSend reports whether v was sent on ch within d.
func timedSend(ch chan<- int, v int, t *time.Timer, d time.Duration) bool {
	select {
	case ch <- v:
		return true
	default:
	}

	t.Reset(d)
	defer t.Stop()

	select {
	case ch <- v:
		return true
	case <-t.C:
		return false
	}
}

// broadcast sends v to every subscriber and returns those that remain.
// A miss is a strike; a delivery clears the strikes. A subscriber with
// more than tolerance consecutive strikes is closed with a cause.
func broadcast(subs []*subscriber, v int, t *time.Timer, tolerance int) []*subscriber {
	for _, s := range subs {
		if timedSend(s.ch, v, t, time.Millisecond) {
			s.strikes = 0
			continue
		}

		s.strikes++
		if s.strikes > tolerance {
			s.err = errTooSlow
			close(s.ch)
		}
	}

	return slices.DeleteFunc(subs, func(s *subscriber) bool { return s.err != nil })
}

func run(tolerance int) (lines []string) {
	t := time.NewTimer(time.Hour)
	t.Stop()

	healthy := &subscriber{name: "healthy", ch: make(chan int, 3)}
	stuck := &subscriber{name: "stuck", ch: make(chan int)} // never read until evicted
	subs := []*subscriber{healthy, stuck}

	for v := range 3 {
		subs = broadcast(subs, v, t, tolerance)
	}

	_, open := <-stuck.ch
	lines = append(lines, fmt.Sprintf("stuck open: %t cause: %v", open, stuck.err))

	close(healthy.ch)

	for v := range healthy.ch {
		lines = append(lines, fmt.Sprintf("healthy got %d", v))
	}

	lines = append(lines, fmt.Sprintf("remaining: %d", len(subs)))

	return lines
}
