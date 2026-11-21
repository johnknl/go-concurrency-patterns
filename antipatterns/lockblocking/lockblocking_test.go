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

// Mutants that survive on purpose: none.

package lockblocking

import (
	"testing"
	"testing/synctest"
)

func TestBadStore_Publish(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		s := &badStore{value: 42}
		out := make(chan int)
		done := make(chan struct{})
		go func() {
			s.Publish(out)
			close(done)
		}()

		// Publish is blocked on the send and still holds the lock.
		synctest.Wait()
		if s.mu.TryLock() {
			t.Fatal("lock free while the send is blocked")
		}

		// A second publisher cannot even read the value: it waits for the lock.
		second := make(chan struct{})
		go func() {
			s.Publish(out)
			close(second)
		}()

		if v := <-out; v != 42 {
			t.Fatalf("received %d, want 42", v)
		}
		<-done
		if v := <-out; v != 42 {
			t.Fatalf("received %d, want 42", v)
		}
		<-second

		if !s.mu.TryLock() {
			t.Fatal("lock still held after the sends")
		}
		s.mu.Unlock()
	})
}

func TestStore_Publish(t *testing.T) {
	t.Run("deliver runs without the lock", func(t *testing.T) {
		s := &store{value: 42}
		var got int
		s.Publish(func(v int) {
			got = v
			if !s.mu.TryLock() {
				t.Error("lock held during deliver")
				return
			}
			s.mu.Unlock()
		})
		if got != 42 {
			t.Fatalf("delivered %d, want 42", got)
		}
	})

	t.Run("reads the value under the lock", func(t *testing.T) {
		// A writer holds the lock while Publish starts. Whether Publish is not yet at Lock or blocked on it, it can
		// only read the value after the writer unlocks. (A blocked Lock is not durable for synctest.Wait, so the
		// race detector is what catches a read without the lock.)
		s := &store{value: 1}
		got := make(chan int, 1)

		s.mu.Lock()
		go s.Publish(func(v int) { got <- v })
		s.value = 2
		s.mu.Unlock()

		if v := <-got; v != 2 {
			t.Fatalf("delivered %d, want 2", v)
		}
	})
}
