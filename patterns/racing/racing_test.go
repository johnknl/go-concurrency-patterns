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

package racing

// Mutants that survive on purpose:
// - remove-cancel racing.go:47 (defer cancel): the explicit cancel before wg.Wait already canceled ctx.

import (
	"context"
	"slices"
	"sync"
	"testing"
	"testing/synctest"
)

func TestReplica(t *testing.T) {
	t.Run("answers with its name once released", func(t *testing.T) {
		release := make(chan struct{})
		close(release)

		if got := replica("a", release, nil)(t.Context()); got != "a" {
			t.Fatalf("answer = %q, want %q", got, "a")
		}
	})

	t.Run("records the cancellation", func(t *testing.T) {
		canceled := make(chan string, 1)
		ctx, cancel := context.WithCancel(t.Context())
		cancel()

		if got := replica("a", nil, canceled)(ctx); got != "" {
			t.Fatalf("answer = %q, want empty", got)
		}

		if got := <-canceled; got != "a" {
			t.Fatalf("canceled = %q, want %q", got, "a")
		}
	})
}

func TestFirst(t *testing.T) {
	t.Run("one replica", func(t *testing.T) {
		release := make(chan struct{})
		close(release)

		if got := first(t.Context(), replica("only", release, nil)); got != "only" {
			t.Fatalf("answer = %q, want %q", got, "only")
		}
	})

	t.Run("returns the first answer and the others have ended", func(t *testing.T) {
		synctest.Test(t, func(t *testing.T) {
			release := make(chan struct{})
			never := make(chan struct{})
			canceled := make(chan string, 3)

			done := make(chan string)
			go func() {
				done <- first(t.Context(),
					replica("a", never, canceled),
					replica("b", never, canceled),
					replica("c", release, canceled),
				)
			}()

			synctest.Wait()
			close(release)

			if got := <-done; got != "c" {
				t.Fatalf("answer = %q, want %q", got, "c")
			}

			// The losers have recorded their cancellation before first returned.
			close(canceled)

			losers := slices.Sorted(func(yield func(string) bool) {
				for name := range canceled {
					if !yield(name) {
						return
					}
				}
			})

			if want := []string{"a", "b"}; !slices.Equal(losers, want) {
				t.Fatalf("canceled = %v, want %v", losers, want)
			}
		})
	})

	t.Run("replicas run concurrently", func(t *testing.T) {
		const n = 5

		var started sync.WaitGroup
		started.Add(n)

		all := make(chan struct{})
		go func() {
			started.Wait()
			close(all)
		}()

		// Each replica answers only after every replica has started, which needs them to run at the same time.
		r := func(ctx context.Context) string {
			started.Done()

			select {
			case <-all:
				return "answer"
			case <-ctx.Done():
				return ""
			}
		}

		if got := first(t.Context(), slices.Repeat([]func(context.Context) string{r}, n)...); got != "answer" {
			t.Fatalf("answer = %q, want %q", got, "answer")
		}
	})

	t.Run("a late answer from a loser does not block it", func(t *testing.T) {
		synctest.Test(t, func(t *testing.T) {
			release := make(chan struct{})
			close(release)

			canceled := make(chan string, 3)

			// Every replica can answer; only one answer is taken, the others go into the buffer.
			got := first(t.Context(),
				replica("x", release, canceled),
				replica("x", release, canceled),
				replica("x", release, canceled),
			)

			if got != "x" {
				t.Fatalf("answer = %q, want %q", got, "x")
			}
		})
	})

	t.Run("canceled parent cancels every replica", func(t *testing.T) {
		synctest.Test(t, func(t *testing.T) {
			canceled := make(chan string, 2)
			ctx, cancel := context.WithCancel(t.Context())
			cancel()

			if got := first(ctx, replica("a", nil, canceled), replica("b", nil, canceled)); got != "" {
				t.Fatalf("answer = %q, want empty", got)
			}

			if len(canceled) != 2 {
				t.Fatalf("canceled replicas = %d, want 2", len(canceled))
			}
		})
	})
}
