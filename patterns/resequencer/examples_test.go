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

package resequencer

import (
	"fmt"
	"strings"
)

func Example_resequence() {
	in := make(chan item, 5)
	for _, seq := range []int{2, 0, 4, 1, 3} {
		in <- item{seq: seq, val: string(rune('a' + seq))}
	}
	close(in)

	err := resequence(in, 4, func(it item) { fmt.Print(it.val, " ") })
	fmt.Println(err)

	// Output:
	// a b c d e <nil>
}

func Example_gap() {
	in := make(chan item, 4)
	for _, seq := range []int{0, 2, 3, 4} {
		in <- item{seq: seq, val: fmt.Sprint(seq)}
	}
	close(in)

	err := resequence(in, 2, func(it item) { fmt.Print(it.val, " ") })
	fmt.Println(err)

	// Output:
	// 0 waiting for seq 1 with 3 buffered: too many items buffered behind a gap
}

func Example_orderedMap() {
	words := strings.Fields("the results come back in input order")

	out, err := orderedMap(words, 3, 4, strings.ToUpper)
	fmt.Println(out, err)

	// Output:
	// [THE RESULTS COME BACK IN INPUT ORDER] <nil>
}
