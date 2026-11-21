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

package batcher

import (
	"fmt"
	"time"
)

// Example_sizeAndClose fills one batch to its size, then closes the input with two items left over, which are
// flushed before the batcher ends.
func Example_sizeAndClose() {
	in := make(chan string, 8)
	done := batcher(in, 3, time.Hour, func(batch []string) { fmt.Println("flush", batch) })

	for _, item := range []string{"a", "b", "c", "d", "e"} {
		in <- item
	}
	close(in)
	<-done

	// Output:
	// flush [a b c]
	// flush [d e]
}

// Example_time sends one item and waits for its batch, which the timer flushes because no more items arrive.
func Example_time() {
	in := make(chan string, 8)
	flushed := make(chan []string)
	done := batcher(in, 100, 10*time.Millisecond, func(batch []string) { flushed <- batch })

	in <- "lonely"
	fmt.Println("flush", <-flushed)

	close(in)
	<-done

	// Output:
	// flush [lonely]
}
