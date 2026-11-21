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

package timedsend

import (
	"context"
	"fmt"
	"time"
)

func Example_timedSend() {
	ctx := context.Background()
	t := newStoppedTimer()

	// One free slot: the first send takes the fast path, the second finds the
	// channel full, and nobody reads, so it times out.
	ch := make(chan int, 1)
	fmt.Println(timedSend(ctx, ch, 1, t, time.Millisecond))
	fmt.Println(timedSend(ctx, ch, 2, t, time.Millisecond))

	// A receiver arrives while the sender waits on the timer.
	unbuffered := make(chan int)
	got := make(chan int)

	go func() { got <- <-unbuffered }()

	fmt.Println(timedSend(ctx, unbuffered, 3, t, time.Minute))
	fmt.Println(<-got)

	// A canceled context ends the wait even though the timer is far away.
	canceled, cancel := context.WithCancel(ctx)
	cancel()

	fmt.Println(timedSend(canceled, unbuffered, 4, t, time.Hour))

	// Output:
	// true
	// false
	// true
	// 3
	// false
}
