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

package future

import (
	"context"
	"fmt"
)

// Example_future overlaps two calls of a synchronous function, does other work, and receives the results when
// they are needed.
func Example_future() {
	a := start(func() (int, error) { return square(3) })
	b := start(func() (int, error) { return square(4) })

	fmt.Println("doing other work")

	ra, rb := <-a, <-b
	fmt.Println(ra.val+rb.val, ra.err, rb.err)

	// Output:
	// doing other work
	// 25 <nil> <nil>
}

// Example_timeout bounds the wait with the context. The computation is not affected, and its result can still
// be received later.
func Example_timeout() {
	release := make(chan struct{})
	slow := make(chan result, 1)

	go func() {
		<-release
		slow <- result{val: 1}
	}()

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // stands in for a deadline that has passed

	select {
	case r := <-slow:
		fmt.Println("got", r.val)
	case <-ctx.Done():
		fmt.Println("gave up:", ctx.Err())
	}

	close(release)
	fmt.Println("later:", (<-slow).val)

	// Output:
	// gave up: context canceled
	// later: 1
}
