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

// Mutants that survive on purpose: none.

package deadline

import (
	"context"
	"errors"
	"testing"
	"testing/synctest"
	"time"
)

func TestWait(t *testing.T) {
	t.Run("returns DeadlineExceeded when the timeout passes", func(t *testing.T) {
		synctest.Test(t, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), time.Second)
			defer cancel()

			start := time.Now()

			if err := wait(ctx); !errors.Is(err, context.DeadlineExceeded) {
				t.Fatalf("got %v, want context.DeadlineExceeded", err)
			}

			if elapsed := time.Since(start); elapsed != time.Second {
				t.Fatalf("returned after %v, want 1s", elapsed)
			}
		})
	})

	t.Run("returns Canceled when canceled", func(t *testing.T) {
		synctest.Test(t, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			errs := make(chan error, 1)

			go func() { errs <- wait(ctx) }()

			synctest.Wait()

			if len(errs) != 0 {
				t.Fatal("wait returned before cancel")
			}

			cancel()

			if err := <-errs; !errors.Is(err, context.Canceled) {
				t.Fatalf("got %v, want context.Canceled", err)
			}
		})
	})
}

func TestWithTimer(t *testing.T) {
	t.Run("times out when done never closes", func(t *testing.T) {
		synctest.Test(t, func(t *testing.T) {
			start := time.Now()

			if err := withTimer(time.Second, make(chan struct{})); err == nil {
				t.Fatal("got nil, want a timeout error")
			}

			if elapsed := time.Since(start); elapsed != time.Second {
				t.Fatalf("returned after %v, want 1s", elapsed)
			}
		})
	})

	t.Run("returns nil when done closes first", func(t *testing.T) {
		synctest.Test(t, func(t *testing.T) {
			done := make(chan struct{})

			time.AfterFunc(time.Second-time.Nanosecond, func() { close(done) })

			if err := withTimer(time.Second, done); err != nil {
				t.Fatalf("got %v, want nil", err)
			}
		})
	})

	t.Run("returns nil when done is already closed", func(t *testing.T) {
		done := make(chan struct{})
		close(done)

		if err := withTimer(time.Hour, done); err != nil {
			t.Fatalf("got %v, want nil", err)
		}
	})
}
