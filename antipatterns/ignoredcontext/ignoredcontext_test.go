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

package ignoredcontext

import (
	"context"
	"errors"
	"testing"
	"testing/synctest"
	"time"
)

func TestBadWait(t *testing.T) {
	t.Run("keeps waiting after cancel", func(t *testing.T) {
		synctest.Test(t, func(t *testing.T) {
			gate := make(chan struct{})
			ctx, cancel := context.WithCancel(t.Context())
			done := make(chan error, 1)
			go func() { done <- badWait(ctx, gate) }()

			cancel()
			synctest.Wait()
			select {
			case err := <-done:
				t.Fatalf("badWait returned %v after cancel, want it still blocked", err)
			default:
			}

			close(gate)
			if err := <-done; err != nil {
				t.Fatalf("badWait = %v, want nil: the cancellation is lost", err)
			}
		})
	})

	t.Run("already canceled context is ignored too", func(t *testing.T) {
		gate := make(chan struct{})
		close(gate)
		ctx, cancel := context.WithCancel(t.Context())
		cancel()
		if err := badWait(ctx, gate); err != nil {
			t.Fatalf("badWait = %v, want nil", err)
		}
	})
}

func TestWait(t *testing.T) {
	t.Run("gate opens", func(t *testing.T) {
		synctest.Test(t, func(t *testing.T) {
			gate := make(chan struct{})
			done := make(chan error, 1)
			go func() { done <- wait(t.Context(), gate) }()
			synctest.Wait()
			select {
			case err := <-done:
				t.Fatalf("wait returned %v before the gate opened", err)
			default:
			}
			close(gate)
			if err := <-done; err != nil {
				t.Fatalf("wait = %v, want nil", err)
			}
		})
	})

	t.Run("canceled before the call", func(t *testing.T) {
		ctx, cancel := context.WithCancel(t.Context())
		cancel()
		if err := wait(ctx, make(chan struct{})); !errors.Is(err, context.Canceled) {
			t.Fatalf("wait = %v, want context.Canceled", err)
		}
	})

	t.Run("canceled while waiting returns the cause", func(t *testing.T) {
		synctest.Test(t, func(t *testing.T) {
			cause := errors.New("client went away")
			ctx, cancel := context.WithCancelCause(t.Context())
			done := make(chan error, 1)
			go func() { done <- wait(ctx, make(chan struct{})) }()

			synctest.Wait()
			cancel(cause)
			if err := <-done; !errors.Is(err, cause) {
				t.Fatalf("wait = %v, want %v", err, cause)
			}
		})
	})

	t.Run("deadline", func(t *testing.T) {
		synctest.Test(t, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(t.Context(), time.Second)
			defer cancel()
			start := time.Now()
			if err := wait(ctx, make(chan struct{})); !errors.Is(err, context.DeadlineExceeded) {
				t.Fatalf("wait = %v, want context.DeadlineExceeded", err)
			}
			if d := time.Since(start); d != time.Second {
				t.Fatalf("wait returned after %v, want 1s", d)
			}
		})
	})
}
