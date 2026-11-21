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

package lockblocking

import (
	"sync"
)

type badStore struct {
	mu    sync.Mutex
	value int
}

// Publish sends the value to out while holding the lock.
func (s *badStore) Publish(out chan<- int) {
	s.mu.Lock()
	defer s.mu.Unlock()
	out <- s.value
}

type store struct {
	mu    sync.Mutex
	value int
}

// Publish copies the value under the lock and calls deliver after unlocking.
func (s *store) Publish(deliver func(int)) {
	s.mu.Lock()
	v := s.value
	s.mu.Unlock()
	deliver(v)
}
