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

// Mutants that survive on purpose:

package cowsnapshot

import (
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
)

func newSnapshot(c config) *snapshot {
	s := &snapshot{}
	s.cur.Store(&c)
	return s
}

func set(key, val string) func(config) {
	return func(next config) { next[key] = val }
}

func TestSnapshot_load(t *testing.T) {
	s := newSnapshot(config{"a": "1"})
	if got := s.load()["a"]; got != "1" {
		t.Errorf("load()[a] = %q, want 1", got)
	}
}

func TestSnapshot_update(t *testing.T) {
	t.Run("publishes the new value", func(t *testing.T) {
		s := newSnapshot(config{})
		s.update(set("a", "1"))
		s.update(set("a", "2"))
		if got := s.load()["a"]; got != "2" {
			t.Errorf("load()[a] = %q, want 2", got)
		}
	})
	t.Run("an old snapshot never changes", func(t *testing.T) {
		s := newSnapshot(config{"a": "1"})
		old := s.load()
		s.update(set("a", "2"))
		s.update(set("b", "3"))
		if len(old) != 1 || old["a"] != "1" {
			t.Errorf("old snapshot = %v, want map[a:1]", old)
		}
		if got := s.load(); len(got) != 2 || got["a"] != "2" {
			t.Errorf("snapshot = %v, want map[a:2 b:3]", got)
		}
	})
	t.Run("readers do not wait for the writer lock", func(t *testing.T) {
		s := newSnapshot(config{})
		s.mu.Lock()
		done := make(chan struct{})
		go func() {
			defer close(done)
			s.update(set("a", "1"))
		}()
		// A reader is never blocked by a writer, even while the writer lock is held.
		if got := s.load()["a"]; got != "" {
			t.Errorf("load()[a] = %q before update finished, want empty", got)
		}
		s.mu.Unlock()
		<-done
		if got := s.load()["a"]; got != "1" {
			t.Errorf("load()[a] = %q, want 1", got)
		}
	})
	t.Run("a reader never sees a mixed version", func(t *testing.T) {
		s := newSnapshot(config{"x": "0", "y": "0"})
		var wg sync.WaitGroup
		wg.Go(func() {
			for i := range 200 {
				v := fmt.Sprint(i)
				s.update(func(next config) { next["x"], next["y"] = v, v })
			}
		})
		wg.Go(func() {
			for range 2000 {
				if c := s.load(); c["x"] != c["y"] {
					t.Errorf("mixed version: x=%s y=%s", c["x"], c["y"])
					return
				}
			}
		})
		wg.Wait()
	})
	t.Run("concurrent writers lose no update", func(t *testing.T) {
		for range 20 {
			s := newSnapshot(config{})
			var wg sync.WaitGroup
			for i := range 50 {
				wg.Go(func() { s.update(set(fmt.Sprint(i), "x")) })
				wg.Go(func() { _ = s.load()[fmt.Sprint(i)] })
			}
			wg.Wait()
			if got := len(s.load()); got != 50 {
				t.Fatalf("snapshot has %d keys, want 50", got)
			}
		}
	})
}

func TestUpdateCAS(t *testing.T) {
	t.Run("publishes a copy", func(t *testing.T) {
		var cur atomic.Pointer[config]
		old := config{"a": "1"}
		cur.Store(&old)
		updateCAS(&cur, set("b", "2"))
		if len(old) != 1 {
			t.Errorf("old snapshot = %v, want map[a:1]", old)
		}
		if got := *cur.Load(); len(got) != 2 || got["a"] != "1" || got["b"] != "2" {
			t.Errorf("snapshot = %v, want map[a:1 b:2]", got)
		}
	})
	t.Run("concurrent writers lose no update", func(t *testing.T) {
		var cur atomic.Pointer[config]
		cur.Store(&config{})
		var wg sync.WaitGroup
		for i := range 100 {
			wg.Go(func() { updateCAS(&cur, set(fmt.Sprint(i), "x")) })
		}
		wg.Wait()
		if got := len(*cur.Load()); got != 100 {
			t.Errorf("snapshot has %d keys, want 100", got)
		}
	})
}
