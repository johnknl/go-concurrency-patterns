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

package cowsnapshot

import (
	"maps"
	"sync"
	"sync/atomic"
)

// config is never modified after it is stored in a snapshot pointer.
type config map[string]string

// snapshot holds the current config. Readers call load; writers call update, which holds mu while it copies and
// stores.
type snapshot struct {
	cur atomic.Pointer[config]
	mu  sync.Mutex
}

// load returns the current config, without a lock. The caller must not modify it.
func (s *snapshot) load() config {
	return *s.cur.Load()
}

// update publishes a copy of the current config, changed by fn, as one new version.
func (s *snapshot) update(fn func(next config)) {
	s.mu.Lock()
	defer s.mu.Unlock()

	next := maps.Clone(*s.cur.Load())
	fn(next)
	s.cur.Store(&next)
}

// updateCAS publishes a copy changed by fn, retrying if another writer published first. fn may run more than once.
func updateCAS(cur *atomic.Pointer[config], fn func(next config)) {
	for {
		old := cur.Load()
		next := maps.Clone(*old)
		fn(next)

		if cur.CompareAndSwap(old, &next) {
			return
		}
	}
}
