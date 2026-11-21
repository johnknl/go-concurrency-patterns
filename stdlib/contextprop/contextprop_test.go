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

package contextprop

import (
	"context"
	"errors"
	"testing"
	"testing/synctest"
	"time"
)

func TestRequestID(t *testing.T) {
	t.Run("absent", func(t *testing.T) {
		id, ok := requestID(context.Background())
		if id != "" || ok {
			t.Fatalf("got (%q, %v), want (\"\", false)", id, ok)
		}
	})

	t.Run("visible in derived contexts", func(t *testing.T) {
		ctx, cancel := context.WithCancel(withRequestID(context.Background(), "req-1"))
		defer cancel()

		if id, ok := requestID(ctx); id != "req-1" || !ok {
			t.Fatalf("got (%q, %v), want (\"req-1\", true)", id, ok)
		}
	})

	t.Run("a child value shadows the parent's without changing it", func(t *testing.T) {
		parent := withRequestID(context.Background(), "parent")
		child := withRequestID(parent, "child")

		if id, _ := requestID(child); id != "child" {
			t.Fatalf("child: got %q, want %q", id, "child")
		}

		if id, _ := requestID(parent); id != "parent" {
			t.Fatalf("parent: got %q, want %q", id, "parent")
		}
	})

	t.Run("a string key with the same name does not collide", func(t *testing.T) {
		//nolint:staticcheck,revive // deliberately uses a built-in key type to show it cannot collide
		ctx := context.WithValue(context.Background(), "requestKey", "other")

		if _, ok := requestID(ctx); ok {
			t.Fatal("found a request ID stored under a different key type")
		}
	})
}

func TestWorker(t *testing.T) {
	t.Run("blocks until its context is canceled and reports why", func(t *testing.T) {
		synctest.Test(t, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			result := make(chan string, 1)

			go worker(ctx, "w", result)

			synctest.Wait()

			if len(result) != 0 {
				t.Fatal("worker reported before its context ended")
			}

			cancel()

			if got, want := <-result, "w: context canceled"; got != want {
				t.Fatalf("got %q, want %q", got, want)
			}
		})
	})

	t.Run("parent cancellation reaches it", func(t *testing.T) {
		synctest.Test(t, func(t *testing.T) {
			parent, cancelParent := context.WithCancel(context.Background())
			child, cancelChild := context.WithCancel(parent)
			defer cancelChild()

			result := make(chan string)

			go worker(child, "child", result)

			cancelParent()

			if got, want := <-result, "child: context canceled"; got != want {
				t.Fatalf("got %q, want %q", got, want)
			}
		})
	})

	t.Run("a timeout is reported as deadline exceeded", func(t *testing.T) {
		synctest.Test(t, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), time.Second)
			defer cancel()

			result := make(chan string)
			start := time.Now()

			go worker(ctx, "w", result)

			if got, want := <-result, "w: context deadline exceeded"; got != want {
				t.Fatalf("got %q, want %q", got, want)
			}

			if elapsed := time.Since(start); elapsed != time.Second {
				t.Fatalf("reported after %v, want 1s", elapsed)
			}
		})
	})
}

func TestCall(t *testing.T) {
	errGone := errors.New("client gone")

	tests := []struct {
		name   string
		cancel func(context.CancelCauseFunc)
		want   error
	}{
		{"with a cause", func(c context.CancelCauseFunc) { c(errGone) }, errGone},
		{"without a cause", func(c context.CancelCauseFunc) { c(nil) }, context.Canceled},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) {
				root, cancel := context.WithCancelCause(context.Background())
				child, stop := context.WithTimeout(root, time.Hour)
				defer stop()

				causes := make(chan error, 1)

				go call(child, causes)

				synctest.Wait()

				if len(causes) != 0 {
					t.Fatal("call reported before its context ended")
				}

				tt.cancel(cancel)

				if got := <-causes; !errors.Is(got, tt.want) {
					t.Fatalf("got %v, want %v", got, tt.want)
				}

				if !errors.Is(child.Err(), context.Canceled) {
					t.Fatalf("child.Err() = %v, want context.Canceled", child.Err())
				}
			})
		})
	}

	t.Run("a deadline is its own cause", func(t *testing.T) {
		synctest.Test(t, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), time.Second)
			defer cancel()

			causes := make(chan error)

			go call(ctx, causes)

			if got := <-causes; !errors.Is(got, context.DeadlineExceeded) {
				t.Fatalf("got %v, want context.DeadlineExceeded", got)
			}
		})
	})
}
