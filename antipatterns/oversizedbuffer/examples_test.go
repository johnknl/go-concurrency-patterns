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

import "fmt"

func Example_oversizedBuffer() {
	queue := badProduce(1000)

	// The producer has finished and nothing has been consumed: the queue holds everything.
	fmt.Println("queued:", len(queue), "of", cap(queue))

	// The first item received waits behind the whole backlog; the newest item had 999 items ahead of it.
	first := <-queue
	fmt.Println("first item:", first, "items queued behind it:", len(queue))
	// Output:
	// queued: 1000 of 1000
	// first item: 0 items queued behind it: 999
}

func Example_boundedQueue() {
	queue := make(chan int, 10)
	accepted, dropped := shed(1000, queue)
	fmt.Println("accepted:", accepted, "dropped:", dropped, "queued:", len(queue), "of", cap(queue))
	// Output:
	// accepted: 10 dropped: 990 queued: 10 of 10
}
