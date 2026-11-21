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

package loadshedding

import "fmt"

// Example_dropNewest keeps the first values and counts the rest as dropped.
func Example_dropNewest() {
	queue := make(chan int, 3)
	dropped := 0

	for v := range 6 {
		if !dropNewest(queue, v) {
			dropped++
		}
	}

	fmt.Println("kept:", drain(queue), "dropped:", dropped)

	// Output:
	// kept: [0 1 2] dropped: 3
}

// Example_dropOldest keeps the latest values and counts the evicted ones.
func Example_dropOldest() {
	queue := make(chan int, 3)
	dropped := 0

	for v := range 6 {
		if dropOldest(queue, v) {
			dropped++
		}
	}

	fmt.Println("kept:", drain(queue), "dropped:", dropped)

	// Output:
	// kept: [3 4 5] dropped: 3
}
