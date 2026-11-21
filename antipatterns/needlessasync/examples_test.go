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
//

package needlessasync

import (
	"context"
	"errors"
	"fmt"
	"sync"
)

// Example_needlessAsync shows the two costs of the asynchronous form. A caller that returns early abandons the
// channel, and the goroutine stays blocked on its send. A caller that needs the value at once receives straight
// away, so the goroutine gained nothing.
func Example_needlessAsync() {
	abandoned := userAsync(2) // the caller returns early and never receives
	fmt.Println("goroutines blocked:", pending.Load())

	r := <-userAsync(1)
	fmt.Println(r.name, r.err)

	<-abandoned // only so that the example does not leave the goroutine behind

	// Output:
	// goroutines blocked: 1
	// user-1 <nil>
}

// Example_synchronous shows the synchronous form. The concurrency is the caller's choice, and the caller keeps the
// errors and the context.
func Example_synchronous() {
	ctx := context.Background()

	var (
		a, b       string
		errA, errB error
		wg         sync.WaitGroup
	)

	wg.Go(func() { a, errA = user(ctx, 1) })
	wg.Go(func() { b, errB = user(ctx, -1) })
	wg.Wait()

	fmt.Println(a, b == "")
	fmt.Println(errors.Join(errA, errB))

	// Output:
	// user-1 true
	// user -1: not found
}
