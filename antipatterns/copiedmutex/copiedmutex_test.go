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

package copiedmutex

import (
	"sync"
	"testing"
	"time"
)

func TestCopyOf(t *testing.T) {
	t.Run("the copy has its own lock", func(t *testing.T) {
		c := &counter{}
		cp := copyOf(c)
		cp.mu.Lock()
		cp.n++
		if !c.mu.TryLock() {
			t.Fatal("locking the copy locked the original")
		}
		c.mu.Unlock()
		cp.mu.Unlock()
		if c.n != 0 {
			t.Errorf("original n = %d, want 0: the update went to the copy", c.n)
		}
	})
	t.Run("a copy of a locked mutex stays locked", func(t *testing.T) {
		c := &counter{}
		c.mu.Lock()
		cp := copyOf(c)
		c.mu.Unlock()
		if cp.mu.TryLock() {
			t.Fatal("copy of a locked mutex is unlocked")
		}
	})
	t.Run("copies values", func(t *testing.T) {
		c := &counter{n: 5}
		if cp := copyOf(c); cp.n != 5 {
			t.Errorf("copy n = %d, want 5", cp.n)
		}
	})
}

func TestSafeCounter_Inc(t *testing.T) {
	t.Run("concurrent increments", func(t *testing.T) {
		c := &safeCounter{}
		var wg sync.WaitGroup
		for range 1000 {
			wg.Go(c.Inc)
		}
		wg.Wait()
		if c.n != 1000 {
			t.Errorf("n = %d, want 1000", c.n)
		}
	})
	t.Run("waits for the shared lock", func(t *testing.T) {
		c := &safeCounter{}
		c.mu.Lock()
		done := make(chan struct{})
		go func() {
			defer close(done)
			c.Inc()
		}()
		select {
		case <-done:
			t.Error("Inc finished while the lock was held")
		case <-time.After(20 * time.Millisecond):
		}
		c.mu.Unlock()
		<-done
		if c.n != 1 {
			t.Errorf("n = %d, want 1", c.n)
		}
	})
}
