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

package workerpool

import (
	"fmt"
	"slices"
	"sync"
)

// Example_pool runs a fixed number of workers over a jobs channel. Submitting blocks while all workers are
// busy, so the number of jobs in flight never exceeds the number of workers.
func Example_pool() {
	jobs := make(chan int)
	results := make(chan result)

	var wg sync.WaitGroup

	for range 3 {
		wg.Go(func() {
			for id := range jobs {
				out, err := process(id)
				results <- result{id: id, out: out, err: err}
			}
		})
	}

	go func() {
		defer close(jobs)

		for id := 1; id <= 5; id++ {
			jobs <- id
		}
	}()

	go func() {
		wg.Wait()
		close(results)
	}()

	var all []result
	for r := range results {
		all = append(all, r)
	}

	slices.SortFunc(all, func(a, b result) int { return a.id - b.id })

	for _, r := range all {
		if r.err != nil {
			fmt.Printf("job %d failed: %v\n", r.id, r.err)
			continue
		}

		fmt.Printf("job %d = %d\n", r.id, r.out)
	}

	// Output:
	// job 1 = 10
	// job 2 = 20
	// job 3 = 30
	// job 4 failed: unlucky
	// job 5 = 50
}
