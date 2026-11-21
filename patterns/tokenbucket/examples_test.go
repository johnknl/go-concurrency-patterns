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

package tokenbucket

import (
	"context"
	"fmt"
	"time"
)

// Example_tokenBucket allows a burst of 3 at once; the next request is refused until a token is earned, and a
// waiting request ends with its context.
func Example_tokenBucket() {
	b := newBucket(1.0/3600, 3) // one token per hour

	for i := 1; i <= 4; i++ {
		fmt.Println("request", i, b.allow())
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	fmt.Println("request 5", b.wait(ctx))

	// 1000 tokens per second, burst 1: the second wait lasts about a millisecond.
	fast := newBucket(1000, 1)
	start := time.Now()
	_ = fast.wait(context.Background())
	_ = fast.wait(context.Background())
	fmt.Println("waited about 1ms:", time.Since(start) >= 999*time.Microsecond)

	// Output:
	// request 1 true
	// request 2 true
	// request 3 true
	// request 4 false
	// request 5 context canceled
	// waited about 1ms: true
}

// Example_channelBucket drives the refill from a channel instead of a clock: a burst of 3 is available at once,
// the fourth request has to wait for a tick.
func Example_channelBucket() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	tick := make(chan time.Time)
	tokens := newChanBucket(ctx, 3, tick)

	for i := 1; i <= 3; i++ {
		fmt.Println("request", i, take(ctx, tokens))
	}

	// The bucket is empty, so only the canceled context is ready.
	canceled, stop := context.WithCancel(ctx)
	stop()
	fmt.Println("request 4", take(canceled, tokens))

	tick <- time.Time{}
	fmt.Println("request 5", take(ctx, tokens))

	// Output:
	// request 1 <nil>
	// request 2 <nil>
	// request 3 <nil>
	// request 4 context canceled
	// request 5 <nil>
}
