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

package needlessasync

// Mutants that survive on purpose:

import (
	"context"
	"errors"
	"testing"
	"testing/synctest"
)

func TestUserAsync(t *testing.T) {
	t.Run("delivers the result and the error", func(t *testing.T) {
		if r := <-userAsync(3); r.name != "user-3" || r.err != nil {
			t.Fatalf("got %q, %v", r.name, r.err)
		}

		if r := <-userAsync(-1); !errors.Is(r.err, errNotFound) {
			t.Fatalf("got %v, want %v", r.err, errNotFound)
		}
	})

	t.Run("an abandoned call leaves its goroutine blocked", func(t *testing.T) {
		synctest.Test(t, func(t *testing.T) {
			before := pending.Load()
			ch := userAsync(1)
			synctest.Wait() // the goroutine is durably blocked on its send

			if n := pending.Load() - before; n != 1 {
				t.Fatalf("%d goroutines blocked, want 1", n)
			}

			select {
			case r := <-ch: // releasing it is the only way to end it
				if r.name != "user-1" {
					t.Fatalf("got %q", r.name)
				}
			default:
				t.Fatal("no value waiting: the goroutine did not block on the send")
			}
		})
	})
}

func TestUser(t *testing.T) {
	t.Run("returns the result", func(t *testing.T) {
		for _, id := range []int{0, 2} {
			want := map[int]string{0: "user-0", 2: "user-2"}[id]
			if name, err := user(context.Background(), id); name != want || err != nil {
				t.Fatalf("user(%d) = %q, %v; want %q, nil", id, name, err, want)
			}
		}
	})

	t.Run("returns the error", func(t *testing.T) {
		if _, err := user(context.Background(), -2); !errors.Is(err, errNotFound) {
			t.Fatalf("got %v, want %v", err, errNotFound)
		}
	})

	t.Run("honors a canceled context", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		if _, err := user(ctx, 2); !errors.Is(err, context.Canceled) {
			t.Fatalf("got %v, want %v", err, context.Canceled)
		}
	})
}
