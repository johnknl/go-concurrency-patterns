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

package windowing

import (
	"context"
	"fmt"
	"time"
)

// feed sends the steps on in and tick, in order, from its own goroutine, then cancels: an int is a value, a
// time.Time a tick. Both channels are unbuffered, so the receiver has taken every step before the next one is sent.
func feed(cancel context.CancelFunc, steps ...any) (<-chan int, <-chan time.Time) {
	in := make(chan int)
	tick := make(chan time.Time)

	go func() {
		defer cancel()

		for _, s := range steps {
			switch s := s.(type) {
			case int:
				in <- s
			case time.Time:
				tick <- s
			}
		}
	}()

	return in, tick
}

// at returns a time on a fixed day, so that the examples print fixed times.
func at(sec int) time.Time {
	return time.Date(2026, 1, 1, 12, 0, sec, 0, time.UTC)
}

// Example_tumbling counts and sums the values per window. The ticks are sent by hand; a program passes the C
// field of a time.Ticker.
func Example_tumbling() {
	ctx, cancel := context.WithCancel(context.Background())
	in, tick := feed(cancel, 1, 2, at(10), 3, at(20), at(30))

	_ = tumbling(ctx, in, tick, func(w window) {
		fmt.Printf("%s: count %d, sum %d\n", w.end.Format(time.TimeOnly), w.count, w.sum)
	})

	// Output:
	// 12:00:10: count 2, sum 3
	// 12:00:20: count 1, sum 3
	// 12:00:30: count 0, sum 0
}

// Example_sliding reports every period the count and sum of the last two periods.
func Example_sliding() {
	ctx, cancel := context.WithCancel(context.Background())
	in, tick := feed(cancel, 1, 2, at(10), 3, at(20), 4, at(30), at(40))

	_ = sliding(ctx, in, tick, 2, func(w window) {
		fmt.Printf("%s: count %d, sum %d\n", w.end.Format(time.TimeOnly), w.count, w.sum)
	})

	// Output:
	// 12:00:10: count 2, sum 3
	// 12:00:20: count 3, sum 6
	// 12:00:30: count 2, sum 7
	// 12:00:40: count 1, sum 4
}
