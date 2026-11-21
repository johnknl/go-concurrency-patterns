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

package untimedbatch

import (
	"fmt"
	"time"
)

// Example_sizeOnly sends two items to a batcher that flushes at four. They stay unflushed however long one
// waits, and closing the input drops them.
func Example_sizeOnly() {
	in := make(chan string, 8)
	flushed := make(chan []string, 8)
	done := sizeOnly(in, 4, func(batch []string) { flushed <- batch })

	in <- "a"
	in <- "b"

	select {
	case batch := <-flushed:
		fmt.Println("flush", batch)
	case <-time.After(50 * time.Millisecond):
		fmt.Println("nothing flushed yet")
	}

	close(in)
	<-done
	fmt.Println("flushed after close:", len(flushed))

	// Output:
	// nothing flushed yet
	// flushed after close: 0
}

// Example_timeBound is the fix: the first item of a batch starts a timer, and the rest is flushed on close.
func Example_timeBound() {
	in := make(chan string, 8)
	flushed := make(chan []string, 8)
	done := sizeOrTime(in, 4, 10*time.Millisecond, func(batch []string) { flushed <- batch })

	in <- "a"
	fmt.Println("flush", <-flushed)

	in <- "b"
	close(in)
	<-done
	fmt.Println("flush", <-flushed)

	// Output:
	// flush [a]
	// flush [b]
}
