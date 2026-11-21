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

package oversizedbuffer

// badProduce "fixes" a blocked producer with a buffer large enough for everything. The producer never blocks,
// and the mismatch with the consumer stays.
func badProduce(n int) chan int {
	queue := make(chan int, n)
	for i := range n {
		queue <- i
	}
	return queue
}

// shed offers every item to a bounded queue and drops the ones that do not fit, returning how many it
// accepted and dropped. Replacing the select with a plain send would apply backpressure instead.
func shed(n int, queue chan<- int) (accepted, dropped int) {
	for i := range n {
		select {
		case queue <- i:
			accepted++
		default:
			dropped++
		}
	}
	return accepted, dropped
}
