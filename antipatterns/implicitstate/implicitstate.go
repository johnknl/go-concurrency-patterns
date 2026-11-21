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

package implicitstate

import (
	"errors"
	"sync"
)

// badServer keeps its lifecycle in three flags. Eight combinations exist; only four are meant to.
type badServer struct {
	mu       sync.Mutex
	started  bool
	stopping bool
	closed   bool
}

func (s *badServer) Start(init func()) error {
	s.mu.Lock()
	if s.started || s.closed {
		s.mu.Unlock()
		return errors.New("already started or closed")
	}
	s.mu.Unlock()

	init() // slow setup, without the lock

	s.mu.Lock()
	s.started = true
	s.mu.Unlock()
	return nil
}

func (s *badServer) Stop() {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed {
		return
	}
	s.stopping = true
	// ... release resources ...
	s.stopping = false
	s.closed = true
}

func (s *badServer) Running() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.started && !s.stopping && !s.closed
}

type lifecycle int

const (
	idle lifecycle = iota
	starting
	running
	stopped
)

func (l lifecycle) String() string {
	return [...]string{"idle", "starting", "running", "stopped"}[l]
}

var errNotIdle = errors.New("not idle")
var errStoppedDuringStart = errors.New("stopped during start")

// server keeps its lifecycle in one field; every value is a legal state.
type server struct {
	mu    sync.Mutex
	state lifecycle
}

func (s *server) Start(init func()) error {
	s.mu.Lock()
	if s.state != idle {
		s.mu.Unlock()
		return errNotIdle
	}
	s.state = starting
	s.mu.Unlock()

	init()

	s.mu.Lock()
	defer s.mu.Unlock()
	if s.state != starting { // Stop came first while init ran
		return errStoppedDuringStart
	}
	s.state = running
	return nil
}

func (s *server) Stop() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.state = stopped
}

func (s *server) State() lifecycle {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.state
}
