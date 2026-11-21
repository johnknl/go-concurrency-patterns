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

package prematureparallel

import (
	"slices"
	"sync"
	"sync/atomic"
)

// squaresParallel squares a slice that is known up front with a generator, workers, and a result channel. Each
// item crosses two channels to be multiplied once, and the results arrive in completion order, so they have to be
// sorted. It also reports how many goroutines it started and how many channel handoffs it made.
func squaresParallel(nums []int, workers int) (out []int, goroutines, handoffs int) {
	var sends atomic.Int64

	jobs := make(chan int)

	go func() {
		defer close(jobs)

		for _, n := range nums {
			jobs <- n
			sends.Add(1)
		}
	}()

	results := make(chan int)

	var wg sync.WaitGroup
	for range workers {
		wg.Go(func() {
			for n := range jobs {
				results <- n * n
				sends.Add(1)
			}
		})
	}

	go func() {
		wg.Wait()
		close(results)
	}()

	for r := range results {
		out = append(out, r)
	}

	slices.Sort(out) // the order was lost on the way

	return out, workers + 2, int(sends.Load())
}

// squares is the fix: a loop. The order is kept and nothing is started.
func squares(nums []int) []int {
	out := make([]int, len(nums))
	for i, n := range nums {
		out[i] = n * n
	}

	return out
}

// mapBounded is the fix when f blocks or is expensive: one goroutine per item, at most limit at a time, each
// writing its own index, so the order is kept without a channel for the results.
func mapBounded[T, R any](in []T, limit int, f func(T) R) []R {
	out := make([]R, len(in))
	sem := make(chan struct{}, limit)

	var wg sync.WaitGroup
	for i, v := range in {
		sem <- struct{}{}

		wg.Go(func() {
			defer func() { <-sem }()

			out[i] = f(v)
		})
	}

	wg.Wait()

	return out
}
