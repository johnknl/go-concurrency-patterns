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

package cancelonerr

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"testing/synctest"
)

var errBoom = errors.New("boom")

func TestRun(t *testing.T) {
	t.Run("no tasks", func(t *testing.T) {
		if err := run(context.Background()); err != nil {
			t.Fatalf("got %v, want nil", err)
		}
	})

	t.Run("all tasks succeed", func(t *testing.T) {
		var calls atomic.Int32
		ok := func(ctx context.Context) error {
			calls.Add(1)

			return ctx.Err()
		}

		if err := run(context.Background(), ok, ok, ok); err != nil {
			t.Fatalf("got %v, want nil", err)
		}

		if calls.Load() != 3 {
			t.Fatalf("%d tasks ran, want 3", calls.Load())
		}
	})

	t.Run("the first error cancels the others and is returned", func(t *testing.T) {
		synctest.Test(t, func(t *testing.T) {
			var stopped atomic.Int32
			wait := func(ctx context.Context) error {
				<-ctx.Done()
				stopped.Add(1)

				if !errors.Is(context.Cause(ctx), errBoom) {
					t.Errorf("task sees cause %v, want errBoom", context.Cause(ctx))
				}

				return ctx.Err()
			}
			fail := func(context.Context) error { return errBoom }

			if err := run(context.Background(), wait, fail, wait); !errors.Is(err, errBoom) {
				t.Fatalf("got %v, want errBoom", err)
			}

			if stopped.Load() != 2 {
				t.Fatalf("%d tasks stopped, want 2", stopped.Load())
			}
		})
	})

	t.Run("a later error does not replace the first", func(t *testing.T) {
		synctest.Test(t, func(t *testing.T) {
			second := errors.New("second")
			first := func(context.Context) error { return errBoom }
			later := func(ctx context.Context) error {
				<-ctx.Done()

				return second
			}

			if err := run(context.Background(), first, later); !errors.Is(err, errBoom) {
				t.Fatalf("got %v, want errBoom", err)
			}
		})
	})

	t.Run("run waits for every task before it returns", func(t *testing.T) {
		synctest.Test(t, func(t *testing.T) {
			release := make(chan struct{})
			var finished atomic.Bool
			slow := func(context.Context) error { //nolint:unparam // a task must return an error
				<-release
				finished.Store(true)

				return nil
			}
			fail := func(context.Context) error { return errBoom }

			errs := make(chan error, 1)
			go func() { errs <- run(context.Background(), slow, fail) }()

			synctest.Wait()

			if len(errs) != 0 {
				t.Fatal("run returned while a task was still running")
			}

			close(release)

			if err := <-errs; !errors.Is(err, errBoom) || !finished.Load() {
				t.Fatalf("got %v (finished %v), want errBoom after the slow task", err, finished.Load())
			}
		})
	})

	t.Run("the context is canceled after run returns", func(t *testing.T) {
		var got context.Context
		keep := func(ctx context.Context) error {
			got = ctx

			return nil
		}

		if err := run(context.Background(), keep); err != nil {
			t.Fatalf("got %v, want nil", err)
		}

		if got.Err() == nil {
			t.Fatal("the derived context is still live")
		}
	})

	t.Run("a canceled parent is reported", func(t *testing.T) {
		ctx, cancel := context.WithCancelCause(context.Background())
		cancel(errBoom)

		wait := func(ctx context.Context) error {
			<-ctx.Done()

			return nil
		}

		if err := run(ctx, wait); !errors.Is(err, errBoom) {
			t.Fatalf("got %v, want errBoom", err)
		}
	})
}
