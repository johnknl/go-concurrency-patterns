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

package boundedqueue

import "fmt"

// Example_queue shows a buffer absorbing a producer that runs ahead of any consumer.
func Example_queue() {
	queue := make(chan int, 4)

	fill(queue, 4)
	fmt.Println("len:", len(queue), "cap:", cap(queue))

	for n := range queue {
		fmt.Println("got", n)
	}

	fmt.Println("len:", len(queue), "cap:", cap(queue))

	// Output:
	// len: 4 cap: 4
	// got 0
	// got 1
	// got 2
	// got 3
	// len: 0 cap: 4
}

// Example_full shows that a full buffer applies backpressure again: a send that would block is not ready.
func Example_full() {
	queue := make(chan int, 2)
	queue <- 1
	queue <- 2

	select {
	case queue <- 3:
		fmt.Println("accepted")
	default:
		fmt.Println("full, producer would wait")
	}

	// Output:
	// full, producer would wait
}

// Example_batch shows a stage that is cheaper per batch: it takes everything queued, up to a limit, in one go.
func Example_batch() {
	queue := make(chan int, 8)
	fill(queue, 7)

	for {
		batch := make([]int, 0, 3)

		for n := range queue {
			batch = append(batch, n)

			if len(batch) == cap(batch) {
				break
			}
		}

		if len(batch) == 0 {
			break
		}

		fmt.Println("write", batch)
	}

	// Output:
	// write [0 1 2]
	// write [3 4 5]
	// write [6]
}
