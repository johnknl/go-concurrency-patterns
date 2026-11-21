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

package reorderbuffer

import "fmt"

func Example_unboundedBuffer() {
	for _, n := range []int{10, 100, 1000} {
		in := make(chan item, n)
		for seq := 1; seq <= n; seq++ { // item 0 never arrives
			in <- item{seq: seq}
		}
		close(in)

		emitted := 0
		buffered := reorderUnbounded(in, func(item) { emitted++ })
		fmt.Printf("after %d items: emitted %d, buffered %d\n", n, emitted, buffered)
	}

	// Output:
	// after 10 items: emitted 0, buffered 10
	// after 100 items: emitted 0, buffered 100
	// after 1000 items: emitted 0, buffered 1000
}

func Example_boundedBuffer() {
	in := make(chan item, 7)
	for _, seq := range []int{1, 2, 3, 4, 0, 5, 6} { // item 0 arrives after the bound gave up on it
		in <- item{seq: seq, val: fmt.Sprint(seq)}
	}
	close(in)

	reorderBounded(in, 3,
		func(it item) { fmt.Println("emit", it.val) },
		func(seq int) { fmt.Println("skip", seq) },
		func(it item) { fmt.Println("late", it.val) },
	)

	// Output:
	// skip 0
	// emit 1
	// emit 2
	// emit 3
	// emit 4
	// late 0
	// emit 5
	// emit 6
}
