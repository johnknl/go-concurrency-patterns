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
//   - remove-critical-section in lockedPool.get (the lock in the AfterFunc callback): without it, a wake-up can
//     fall between a waiter's check of ctx.Err and its call to Wait. The window is too narrow to hit
//     deterministically.

package resourcepool

import (
	"context"
	"errors"
	"testing"
	"testing/synctest"
)

func TestNewPool(t *testing.T) {
	for _, n := range []int{0, 1, 4} {
		pool := newPool(n)

		if len(pool) != n || cap(pool) != n {
			t.Fatalf("n=%d: len %d cap %d", n, len(pool), cap(pool))
		}

		for want := range n {
			if c := <-pool; c.id != want {
				t.Fatalf("n=%d: got id %d, want %d", n, c.id, want)
			}
		}
	}
}

func TestGet(t *testing.T) {
	t.Run("takes an available resource", func(t *testing.T) {
		pool := newPool(1)

		c, err := get(context.Background(), pool)
		if err != nil || c == nil || c.id != 0 {
			t.Fatalf("got %v, %v", c, err)
		}

		if len(pool) != 0 {
			t.Fatal("resource still in the pool")
		}
	})

	t.Run("a done context ends the wait", func(t *testing.T) {
		pool := newPool(0)

		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		c, err := get(ctx, pool)
		if c != nil || !errors.Is(err, context.Canceled) {
			t.Fatalf("got %v, %v", c, err)
		}
	})

	t.Run("waits until a resource is returned", func(t *testing.T) {
		synctest.Test(t, func(t *testing.T) {
			pool := newPool(1)
			a, _ := get(context.Background(), pool)

			got := make(chan *conn)
			go func() {
				c, _ := get(context.Background(), pool)
				got <- c
			}()

			synctest.Wait()

			select {
			case <-got:
				t.Fatal("get returned from an empty pool")
			default:
			}

			put(pool, a)

			if c := <-got; c != a {
				t.Fatal("did not get the returned resource")
			}
		})
	})
}

func TestPut(t *testing.T) {
	pool := newPool(2)
	a, _ := get(context.Background(), pool)
	b, _ := get(context.Background(), pool)

	put(pool, a)
	put(pool, b)

	if len(pool) != 2 {
		t.Fatalf("len %d, want 2", len(pool))
	}
}

func TestLockedPool(t *testing.T) {
	t.Run("creates n resources", func(t *testing.T) {
		for _, n := range []int{0, 1, 4} {
			pool := newLockedPool(n)

			if len(pool.free) != n {
				t.Fatalf("n=%d: %d free", n, len(pool.free))
			}

			for i, c := range pool.free {
				if c.id != i {
					t.Fatalf("n=%d: got id %d at %d", n, c.id, i)
				}
			}
		}
	})

	t.Run("takes the last returned resource", func(t *testing.T) {
		pool := newLockedPool(2)

		c, err := pool.get(context.Background())
		if err != nil || c.id != 1 || len(pool.free) != 1 {
			t.Fatalf("got %v, %v, %d free", c, err, len(pool.free))
		}
	})

	t.Run("a done context ends the wait", func(t *testing.T) {
		pool := newLockedPool(0)

		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		c, err := pool.get(ctx)
		if c != nil || !errors.Is(err, context.Canceled) {
			t.Fatalf("got %v, %v", c, err)
		}
	})

	t.Run("a free resource is taken even with a done context", func(t *testing.T) {
		pool := newLockedPool(1)

		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		if c, err := pool.get(ctx); c == nil || err != nil {
			t.Fatalf("got %v, %v", c, err)
		}
	})

	t.Run("waits until a resource is returned", func(t *testing.T) {
		synctest.Test(t, func(t *testing.T) {
			pool := newLockedPool(1)
			a, _ := pool.get(context.Background())

			got := make(chan *conn)
			go func() {
				c, _ := pool.get(context.Background())
				got <- c
			}()

			synctest.Wait()

			select {
			case <-got:
				t.Fatal("get returned from an empty pool")
			default:
			}

			pool.put(a)

			if c := <-got; c != a {
				t.Fatal("did not get the returned resource")
			}
		})
	})

	t.Run("canceling ends a wait in progress", func(t *testing.T) {
		synctest.Test(t, func(t *testing.T) {
			pool := newLockedPool(0)
			ctx, cancel := context.WithCancel(context.Background())

			errs := make(chan error)
			go func() {
				_, err := pool.get(ctx)
				errs <- err
			}()

			synctest.Wait()
			cancel()

			if err := <-errs; !errors.Is(err, context.Canceled) {
				t.Fatalf("got %v", err)
			}
		})
	})

	t.Run("put wakes one waiter per resource", func(t *testing.T) {
		synctest.Test(t, func(t *testing.T) {
			pool := newLockedPool(0)

			got := make(chan *conn, 2)
			for range 2 {
				go func() {
					c, _ := pool.get(context.Background())
					got <- c
				}()
			}

			synctest.Wait()
			pool.put(&conn{id: 7})
			synctest.Wait()

			if len(got) != 1 {
				t.Fatalf("%d waiters got a resource, want 1", len(got))
			}

			pool.put(&conn{id: 8})
			<-got
			<-got
		})
	})
}
