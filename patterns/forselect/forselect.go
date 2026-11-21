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

import "context"

// send sends each item on out, and stops early when the context is canceled.
func send(ctx context.Context, out chan<- string, items []string) {
	for _, item := range items {
		select {
		case <-ctx.Done():
			return
		case out <- item:
		}
	}
}

// wait loops until it is told to stop, summing what arrives meanwhile. The only exit is the ctx.Done() case.
func wait(ctx context.Context, in <-chan int) (sum int) {
	for {
		select {
		case <-ctx.Done():
			return sum
		case n := <-in:
			sum += n
		}
	}
}

// work does a unit of work in the default case whenever the context is not done. It closes started after its first
// step and returns the number of steps taken when the context is canceled.
func work(ctx context.Context, started chan<- struct{}) (steps int) {
	for {
		select {
		case <-ctx.Done():
			return steps
		default:
		}

		steps++

		if steps == 1 {
			close(started)
		}
	}
}
