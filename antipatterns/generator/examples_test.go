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

package generator

import (
	"context"
	"fmt"
)

// Example_channel consumes an unbounded sequence and stops it by canceling the context.
func Example_channel() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	for n := range naturals(ctx) {
		if n == 5 {
			break
		}

		fmt.Println(n)
	}

	// Output:
	// 0
	// 1
	// 2
	// 3
	// 4
}

// Example_iterator consumes the sequence with range-over-func.
func Example_iterator() {
	for n := range naturalsSeq() {
		if n == 3 {
			break
		}

		fmt.Println(n)
	}

	// Output:
	// 0
	// 1
	// 2
}
