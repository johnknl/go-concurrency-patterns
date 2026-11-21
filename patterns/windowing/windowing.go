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
	"time"
)

// window is the count and sum of the values received in one window, reported with the time it ended.
type window struct {
	end   time.Time
	count int
	sum   int
}

// tumbling reports the count and sum of the values of in per window, which each value of tick ends. When in is
// closed, it reports the window in progress and returns.
func tumbling(ctx context.Context, in <-chan int, tick <-chan time.Time, emit func(window)) error {
	var w window

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case v, ok := <-in:
			if !ok {
				w.end = time.Now()
				emit(w)
				return nil
			}
			w.count++
			w.sum += v
		case now := <-tick:
			w.end = now
			emit(w)
			w = window{}
		}
	}
}

// bucket is the count and sum of one period of a sliding window.
type bucket struct {
	count int
	sum   int
}

// sliding reports, at each value of tick, the count and sum of the values of in over the last n periods,
// the current one included. When in is closed, it reports once more and returns.
func sliding(ctx context.Context, in <-chan int, tick <-chan time.Time, n int, emit func(window)) error {
	ring := make([]bucket, n)
	i := 0

	total := func(end time.Time) window {
		w := window{end: end}
		for _, b := range ring {
			w.count += b.count
			w.sum += b.sum
		}

		return w
	}

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case v, ok := <-in:
			if !ok {
				emit(total(time.Now()))
				return nil
			}
			ring[i].count++
			ring[i].sum += v
		case now := <-tick:
			emit(total(now))
			i = (i + 1) % n
			ring[i] = bucket{}
		}
	}
}
