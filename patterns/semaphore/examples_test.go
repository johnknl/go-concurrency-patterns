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

package semaphore_test

import (
	"context"
	"fmt"
	"sync"
)

// Example_semaphore uses a buffered channel as a counting semaphore: sending takes a permit, receiving
// returns one. At most three of the twenty goroutines run the guarded section at the same time.
func Example_semaphore() {
	sem := make(chan struct{}, 3)

	var (
		wg           sync.WaitGroup
		mu           sync.Mutex
		active, peak int
	)

	for range 20 {
		wg.Go(func() {
			sem <- struct{}{}
			defer func() { <-sem }()

			mu.Lock()
			active++
			peak = max(peak, active)
			mu.Unlock()

			mu.Lock()
			active--
			mu.Unlock()
		})
	}

	wg.Wait()
	fmt.Println(peak <= 3)

	// Output:
	// true
}

// Example_acquireWithContext waits for a permit only as long as the context allows.
func Example_acquireWithContext() {
	sem := make(chan struct{}, 1)
	sem <- struct{}{} // the only permit is taken

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	select {
	case sem <- struct{}{}:
		fmt.Println("acquired")
	case <-ctx.Done():
		fmt.Println(ctx.Err())
	}

	// Output:
	// context canceled
}

// Example_tryAcquire does not wait: a select with a default case reports whether a permit was free.
func Example_tryAcquire() {
	sem := make(chan struct{}, 1)

	for range 2 {
		select {
		case sem <- struct{}{}:
			fmt.Println("acquired")
		default:
			fmt.Println("busy")
		}
	}

	// Output:
	// acquired
	// busy
}
