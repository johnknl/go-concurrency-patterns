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

package heartbeat

import (
	"context"
	"fmt"
	"time"
)

// Example_perUnitOfWork waits for the pulse of each unit before taking its result, without any sleeping.
func Example_perUnitOfWork() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	heartbeat, results := square(ctx, 2, 3, 4)

	for range 3 {
		<-heartbeat
		fmt.Println("unit started")
		fmt.Println("result:", <-results)
	}

	// Output:
	// unit started
	// result: 4
	// unit started
	// result: 9
	// unit started
	// result: 16
}

// Example_noListener shows that a missing listener does not stall the work.
func Example_noListener() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	_, results := square(ctx, 2, 3, 4)

	for n := range results {
		fmt.Println("result:", n)
	}

	// Output:
	// result: 4
	// result: 9
	// result: 16
}

// Example_interval takes a few pulses of a worker that is alive, then cancels it and waits until it has ended.
func Example_interval() {
	ctx, cancel := context.WithCancel(context.Background())

	heartbeat := ticking(ctx, time.Millisecond)

	for range 3 {
		<-heartbeat
	}

	cancel()

	for range heartbeat {
	}

	fmt.Println("alive, then stopped")

	// Output:
	// alive, then stopped
}
