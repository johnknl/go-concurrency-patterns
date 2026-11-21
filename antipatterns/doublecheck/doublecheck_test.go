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

package doublecheck

import (
	"sync"
	"testing"
	"time"
)

// finishes reports whether f returns within a short grace period. It is used to tell apart "returns without the
// lock" from "waits for the lock", and the caller must make f return eventually.
func finishes(f func()) (done <-chan struct{}, quickly bool) {
	ch := make(chan struct{})
	go func() {
		defer close(ch)
		f()
	}()
	select {
	case <-ch:
		return ch, true
	case <-time.After(20 * time.Millisecond):
		return ch, false
	}
}

// These tests reset the package-level singletons, so they must not run in parallel with each other or the examples.

func TestBadGet(t *testing.T) {
	t.Run("first call initializes under the lock", func(t *testing.T) {
		badInstance = nil
		badMu.Lock()
		done, quickly := finishes(func() { badGet() })
		if quickly {
			t.Error("badGet initialized without taking the lock")
		}
		badMu.Unlock()
		<-done
		if badInstance == nil || badInstance.name != loadedName {
			t.Fatalf("badInstance = %+v, want name %q", badInstance, loadedName)
		}
	})
	t.Run("the defect: later calls read without the lock", func(t *testing.T) {
		// The fast path returns while another goroutine holds the lock, so it gets no happens-before edge with the
		// write. Run concurrently with the first call, that read is a data race.
		badInstance = nil
		first := badGet()
		badMu.Lock()
		done, quickly := finishes(func() {
			if badGet() != first {
				t.Error("second call returned a different instance")
			}
		})
		badMu.Unlock()
		<-done
		if !quickly {
			t.Error("fast path waited for the lock")
		}
	})
}

func TestGetConfig(t *testing.T) {
	results := make([]*config, 8)
	var wg sync.WaitGroup
	for i := range results {
		wg.Go(func() { results[i] = getConfig() })
	}
	wg.Wait()
	for _, c := range results {
		if c != results[0] {
			t.Fatal("callers got different instances")
		}
	}
	if results[0].name != loadedName {
		t.Errorf("name = %q, want %q", results[0].name, loadedName)
	}
}

func TestGet(t *testing.T) {
	t.Run("first call initializes under the lock", func(t *testing.T) {
		instance.Store(nil)
		mu.Lock()
		done, quickly := finishes(func() { get() })
		if quickly {
			t.Error("get initialized without taking the lock")
		}
		mu.Unlock()
		<-done
		if c := instance.Load(); c == nil || c.name != loadedName {
			t.Fatalf("instance = %+v, want name %q", c, loadedName)
		}
	})
	t.Run("fast path is an atomic load", func(t *testing.T) {
		instance.Store(nil)
		first := get()
		mu.Lock()
		done, quickly := finishes(func() {
			if get() != first {
				t.Error("second call returned a different instance")
			}
		})
		mu.Unlock()
		<-done
		if !quickly {
			t.Error("fast path waited for the lock")
		}
	})
	t.Run("concurrent first calls share one instance", func(t *testing.T) {
		instance.Store(nil)
		results := make([]*config, 16)
		start := make(chan struct{})
		var wg sync.WaitGroup
		for i := range results {
			wg.Go(func() {
				<-start
				results[i] = get()
			})
		}
		close(start)
		wg.Wait()
		for _, c := range results {
			if c != results[0] {
				t.Fatal("callers got different instances")
			}
		}
	})
}
