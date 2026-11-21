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

package healing

import (
	"context"
	"fmt"
)

// Example_restart runs a ward that is stuck on its first run. The injected timeout fires at once for the
// first ward and never for the second.
func Example_restart() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	runs := 0

	start := func(ctx context.Context) (<-chan struct{}, <-chan int) {
		runs++
		run := runs

		heartbeat := make(chan struct{}, 1)
		results := make(chan int)

		go func() {
			defer close(heartbeat)
			defer close(results)

			if run == 1 {
				fmt.Println("ward 1: stuck")
				<-ctx.Done()
				fmt.Println("ward 1: canceled")

				return
			}

			for n := range 3 {
				select {
				case heartbeat <- struct{}{}:
				default:
				}

				select {
				case results <- n:
				case <-ctx.Done():
					return
				}
			}
		}()

		return heartbeat, results
	}

	unhealthy := make(chan struct{})
	close(unhealthy)

	calls := 0

	timeout := func() <-chan struct{} {
		calls++
		if calls == 1 {
			return unhealthy
		}

		return nil
	}

	for n := range steward(ctx, timeout, start) {
		fmt.Println("result:", n)
	}

	fmt.Println("wards started:", runs)

	// Output:
	// ward 1: stuck
	// ward 1: canceled
	// result: 0
	// result: 1
	// result: 2
	// wards started: 2
}

// Example_supervisionTree supervises a steward with another steward. The leaf ward is stuck on its first run, and
// only the outer timeout notices: it fires once the leaf is stuck, and the outer steward restarts the whole
// subtree.
func Example_supervisionTree() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	runs := 0
	stuck := make(chan struct{})

	leaf := func(ctx context.Context) (<-chan struct{}, <-chan int) {
		runs++
		run := runs

		heartbeat := make(chan struct{}, 1)
		results := make(chan int)

		go func() {
			defer close(heartbeat)
			defer close(results)

			if run == 1 {
				fmt.Println("leaf 1: stuck")
				close(stuck)
				<-ctx.Done()

				return
			}

			for n := range 3 {
				select {
				case heartbeat <- struct{}{}:
				default:
				}

				select {
				case results <- n:
				case <-ctx.Done():
					return
				}
			}
		}()

		return heartbeat, results
	}

	never := func() <-chan struct{} { return nil }

	calls := 0
	outer := func() <-chan struct{} {
		calls++
		if calls == 1 {
			return stuck
		}

		return nil
	}

	for n := range steward(ctx, outer, supervised(never, leaf)) {
		fmt.Println("result:", n)
	}

	fmt.Println("leaves started:", runs)

	// Output:
	// leaf 1: stuck
	// result: 0
	// result: 1
	// result: 2
	// leaves started: 2
}
