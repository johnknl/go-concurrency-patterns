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

package tee

import (
	"context"
)

// generate sends nums and closes the channel.
func generate(ctx context.Context, nums ...int) <-chan int {
	out := make(chan int)

	go func() {
		defer close(out)

		for _, n := range nums {
			select {
			case out <- n:
			case <-ctx.Done():
				return
			}
		}
	}()

	return out
}

// tee sends every value of in to both returned channels. Both must be read.
func tee(ctx context.Context, in <-chan int) (left, right <-chan int) {
	out1, out2 := make(chan int), make(chan int)

	go func() {
		defer close(out1)
		defer close(out2)

		for {
			var v int

			select {
			case n, ok := <-in:
				if !ok {
					return
				}

				v = n
			case <-ctx.Done():
				return
			}

			c1, c2 := out1, out2

			for range 2 {
				select {
				case c1 <- v:
					c1 = nil
				case c2 <- v:
					c2 = nil
				case <-ctx.Done():
					return
				}
			}
		}
	}()

	return out1, out2
}

// collect reads ch until it is closed.
func collect(ch <-chan int) []int {
	var got []int
	for v := range ch {
		got = append(got, v)
	}

	return got
}
