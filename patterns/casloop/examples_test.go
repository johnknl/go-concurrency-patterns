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

package casloop

import (
	"fmt"
	"sync"
	"sync/atomic"
)

// Example_max tracks the largest value seen by many goroutines without a lock.
func Example_max() {
	var peak atomic.Int64
	var wg sync.WaitGroup

	for i := range 100 {
		wg.Go(func() { storeMax(&peak, int64(i)) })
	}

	wg.Wait()
	fmt.Println(peak.Load())

	// Output:
	// 99
}

// Example_bounded lets 50 goroutines compete for 10 slots; the counter never exceeds the limit.
func Example_bounded() {
	var used, granted atomic.Int64
	var wg sync.WaitGroup

	for range 50 {
		wg.Go(func() {
			if tryIncrement(&used, 10) {
				granted.Add(1)
			}
		})
	}

	wg.Wait()
	fmt.Println(used.Load(), granted.Load())

	// Output:
	// 10 10
}

// Example_pointer updates two fields together by swapping a pointer to an immutable struct.
func Example_pointer() {
	var p atomic.Pointer[stats]
	p.Store(&stats{})

	var wg sync.WaitGroup
	for i := range 10 {
		wg.Go(func() { record(&p, int64(i)) })
	}

	wg.Wait()
	s := p.Load()
	fmt.Println(s.count, s.sum)

	// Output:
	// 10 45
}

// Example_add needs no loop: when the new value does not depend on a decision about the old one, Add is enough.
func Example_add() {
	var hits atomic.Int64
	var wg sync.WaitGroup

	for range 100 {
		wg.Go(func() { hits.Add(1) })
	}

	wg.Wait()
	fmt.Println(hits.Load())

	// Output:
	// 100
}
