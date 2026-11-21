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

package fanout

import (
	"fmt"
	"slices"
	"sync"
)

// Example_workers starts several workers that read from the same channel (fan-out) and send to the same
// channel (fan-in). The output is closed once every worker is done.
func Example_workers() {
	in := emit(1, 2, 3, 4, 5, 6, 7, 8)
	out := make(chan int)

	var wg sync.WaitGroup

	for range 3 {
		wg.Go(func() {
			for n := range in {
				out <- n * n
			}
		})
	}

	go func() {
		wg.Wait()
		close(out)
	}()

	var squares []int
	for n := range out {
		squares = append(squares, n)
	}

	slices.Sort(squares) // completion order is not defined
	fmt.Println(squares)

	// Output:
	// [1 4 9 16 25 36 49 64]
}

// Example_merge combines independent sources into one stream.
func Example_merge() {
	var all []int
	for n := range merge(emit(1, 2), emit(3), emit(4, 5, 6)) {
		all = append(all, n)
	}

	slices.Sort(all)
	fmt.Println(all)

	// Output:
	// [1 2 3 4 5 6]
}
