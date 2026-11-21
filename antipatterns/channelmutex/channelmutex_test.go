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

package channelmutex

// Mutants that survive on purpose:
//   - buffer-channel on get, done, quit, and reply: a buffer changes when a send completes, not what the owner
//     computes or when stop returns.

import (
	"sync"
	"testing"
	"testing/synctest"
)

func TestOwnedCounter(t *testing.T) {
	t.Run("counts concurrent increments", func(t *testing.T) {
		c := newOwnedCounter()
		defer c.stop()

		var wg sync.WaitGroup
		for range 50 {
			wg.Go(c.inc)
		}

		wg.Wait()

		if n := c.value(); n != 50 {
			t.Fatalf("got %d, want 50", n)
		}
	})

	t.Run("a call after stop blocks", func(t *testing.T) {
		synctest.Test(t, func(t *testing.T) {
			c := newOwnedCounter()
			c.inc()
			c.stop()

			returned := make(chan struct{})

			go func() {
				defer close(returned)
				select {
				case c.incr <- struct{}{}:
				case <-t.Context().Done():
				}
			}()

			synctest.Wait()

			select {
			case <-returned:
				t.Fatal("inc after stop returned")
			default:
			}
		})
	})
}

func TestCounter(t *testing.T) {
	var (
		c  counter
		a  atomicCounter
		wg sync.WaitGroup
	)

	if c.value() != 0 || a.value() != 0 {
		t.Fatal("zero value is not zero")
	}

	for range 50 {
		wg.Go(func() {
			c.inc()
			a.inc()
			_ = c.value() // a read during writes, for the race detector
		})
	}

	wg.Wait()

	if c.value() != 50 || a.value() != 50 {
		t.Fatalf("got %d and %d, want 50", c.value(), a.value())
	}
}
