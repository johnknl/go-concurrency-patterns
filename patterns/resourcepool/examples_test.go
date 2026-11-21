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

package resourcepool

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
)

// Example_borrow shows that the pool is bounded: when it is empty, get waits until a resource is returned or
// the context ends.
func Example_borrow() {
	ctx := context.Background()
	pool := newPool(2)

	a, _ := get(ctx, pool)
	b, _ := get(ctx, pool)

	expired, cancel := context.WithCancel(ctx)
	cancel()

	_, err := get(expired, pool)
	fmt.Println("third get:", err)

	put(pool, a)
	c, _ := get(ctx, pool)
	fmt.Println("reused after release:", c == a)

	put(pool, b)
	put(pool, c)

	// Output:
	// third get: context canceled
	// reused after release: true
}

// Example_boundedUse runs more workers than there are connections. Each worker holds a connection for the
// duration of its work, so at most two are in use at once.
func Example_boundedUse() {
	pool := newPool(2)

	var inUse atomic.Int32
	var exceeded atomic.Bool

	var wg sync.WaitGroup

	for range 8 {
		wg.Go(func() {
			c, err := get(context.Background(), pool)
			if err != nil {
				return
			}
			defer put(pool, c)

			if inUse.Add(1) > 2 {
				exceeded.Store(true)
			}

			inUse.Add(-1)
		})
	}

	wg.Wait()

	fmt.Println("more than 2 in use:", exceeded.Load())

	// Output:
	// more than 2 in use: false
}

// Example_lockedPool uses a mutex-guarded free list. The most recently returned resource is handed out first.
func Example_lockedPool() {
	ctx := context.Background()
	pool := newLockedPool(2)

	a, _ := pool.get(ctx)
	b, _ := pool.get(ctx)

	expired, cancel := context.WithCancel(ctx)
	cancel()

	_, err := pool.get(expired)
	fmt.Println("third get:", err)

	pool.put(a)
	pool.put(b)

	c, _ := pool.get(ctx)
	fmt.Println("last returned comes first:", c == b)

	pool.put(c)

	// Output:
	// third get: context canceled
	// last returned comes first: true
}
