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

package forselect

import (
	"context"
	"fmt"
)

// Example_iterationVariable sends the values of a loop on a channel.
func Example_iterationVariable() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	out := make(chan string)

	go func() {
		defer close(out)

		send(ctx, out, []string{"a", "b", "c"})
	}()

	for s := range out {
		fmt.Println(s)
	}

	// Output:
	// a
	// b
	// c
}

// Example_untilStopped blocks in the loop until the context is canceled.
func Example_untilStopped() {
	ctx, cancel := context.WithCancel(context.Background())
	in := make(chan int)
	result := make(chan int)

	go func() {
		result <- wait(ctx, in)
	}()

	in <- 1
	in <- 2
	cancel()
	fmt.Println(<-result)

	// Output:
	// 3
}

// Example_defaultWork keeps working between checks of the context.
func Example_defaultWork() {
	ctx, cancel := context.WithCancel(context.Background())
	started := make(chan struct{})
	result := make(chan int)

	go func() {
		result <- work(ctx, started)
	}()

	<-started
	cancel()
	fmt.Println(<-result >= 1)

	// Output:
	// true
}
