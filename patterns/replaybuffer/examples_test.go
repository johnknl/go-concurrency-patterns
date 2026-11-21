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

package replaybuffer

import (
	"fmt"
	"sync"
)

// Example_replay attaches a subscriber after five messages. It receives the last three, then the live ones.
func Example_replay() {
	in := make(chan int)
	attach := make(chan chan int)

	var wg sync.WaitGroup

	wg.Go(func() { owner(in, attach, 3) })

	for v := 1; v <= 5; v++ {
		in <- v
	}

	late := make(chan int, 8) // large enough for the replay and the live messages
	attach <- late

	in <- 6

	in <- 7

	close(in)
	wg.Wait()

	for v := range late {
		fmt.Println(v)
	}

	// Output:
	// 3
	// 4
	// 5
	// 6
	// 7
}
