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

// Mutants that survive on purpose: none (the package has no mechanism outside its examples).

package semaphore

import (
	"context"
	"errors"
	"sync"
	"testing"
	"testing/synctest"
)

// The package has no mechanism of its own: the semaphore is a buffered channel. These tests pin down the idiom.

func TestChannelSemaphore(t *testing.T) {
	t.Run("at most cap holders at a time", func(t *testing.T) {
		synctest.Test(t, func(t *testing.T) {
			sem := make(chan struct{}, 3)
			gate := make(chan struct{})

			var (
				wg     sync.WaitGroup
				mu     sync.Mutex
				active int
			)

			for range 5 {
				wg.Go(func() {
					sem <- struct{}{}
					defer func() { <-sem }()

					mu.Lock()
					active++
					mu.Unlock()

					<-gate
				})
			}

			synctest.Wait()

			if active != 3 {
				t.Fatalf("%d holders, want 3", active)
			}

			close(gate)
			wg.Wait()

			if active != 5 || len(sem) != 0 {
				t.Fatalf("active %d, permits held %d", active, len(sem))
			}
		})
	})

	t.Run("acquire gives up when the context ends", func(t *testing.T) {
		synctest.Test(t, func(t *testing.T) {
			sem := make(chan struct{}, 1)
			sem <- struct{}{}

			ctx, cancel := context.WithCancel(t.Context())
			errc := make(chan error)

			go func() {
				select {
				case sem <- struct{}{}:
					errc <- nil
				case <-ctx.Done():
					errc <- ctx.Err()
				}
			}()

			synctest.Wait()
			cancel()

			if err := <-errc; !errors.Is(err, context.Canceled) {
				t.Fatalf("got %v, want context.Canceled", err)
			}
		})
	})

	t.Run("try acquire does not wait", func(t *testing.T) {
		sem := make(chan struct{}, 1)
		try := func() bool {
			select {
			case sem <- struct{}{}:
				return true
			default:
				return false
			}
		}

		if !try() || try() {
			t.Fatal("want first try to succeed and second to fail")
		}

		<-sem

		if !try() {
			t.Fatal("permit not available after release")
		}
	})
}
