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

package unbounded

import (
	"fmt"
	"sync"
	"sync/atomic"
)

func Example_unbounded() {
	const n = 100

	var (
		p       peak
		started sync.WaitGroup
	)

	started.Add(n)

	gate := make(chan struct{})

	// The gate holds every goroutine until all of them have started.
	go func() {
		started.Wait()
		close(gate)
	}()

	badFanOut(make([]int, n), func(int) {
		p.enter()
		started.Done()
		<-gate
		p.leave()
	})

	fmt.Println("items:", n, "peak concurrency:", p.max.Load())

	// Output:
	// items: 100 peak concurrency: 100
}

func Example_bounded() {
	const (
		n     = 100
		limit = 4
	)

	var (
		p    peak
		done atomic.Int64
	)

	boundedFanOut(make([]int, n), limit, func(int) {
		p.enter()
		done.Add(1)
		p.leave()
	})

	fmt.Println("done:", done.Load(), "peak within limit:", p.max.Load() <= limit)

	// Output:
	// done: 100 peak within limit: true
}
