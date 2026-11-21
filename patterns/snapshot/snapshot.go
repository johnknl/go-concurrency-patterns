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

package snapshot

import (
	"maps"
)

// stats owns the counters. Only run touches the map.
type stats struct {
	events    chan string
	snapshots chan chan map[string]int
	quit      chan struct{}
	done      chan struct{}
}

func newStats() *stats {
	s := &stats{
		events:    make(chan string),
		snapshots: make(chan chan map[string]int),
		quit:      make(chan struct{}),
		done:      make(chan struct{}),
	}
	go s.run()

	return s
}

func (s *stats) run() {
	defer close(s.done)

	counts := map[string]int{}

	for {
		select {
		case name := <-s.events:
			counts[name]++
		case reply := <-s.snapshots:
			// The reply has capacity 1, so this send never blocks, even if the requester has left.
			reply <- maps.Clone(counts)
		case <-s.quit:
			return
		}
	}
}

func (s *stats) record(name string) { s.events <- name }

// snapshot asks the owner for a copy of the counters.
func (s *stats) snapshot() map[string]int {
	reply := make(chan map[string]int, 1)
	s.snapshots <- reply

	return <-reply
}
