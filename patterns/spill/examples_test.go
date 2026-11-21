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

package spill

import (
	"fmt"
	"os"
	"path/filepath"
)

// Example_spill pushes five items into a queue that holds two in memory, then pops them all.
func Example_spill() {
	tmp, _ := os.MkdirTemp("", "spill")
	defer func() { _ = os.RemoveAll(tmp) }()

	q, err := openSpillQueue(filepath.Join(tmp, "queue"), 2, 1<<20)
	if err != nil {
		fmt.Println(err)

		return
	}
	defer func() { _ = q.close() }()

	for _, item := range []string{"a", "b", "c", "d", "e"} {
		if err := q.push(item); err != nil {
			fmt.Println(err)
		}
	}

	var got []string
	for range 5 {
		item, _, _ := q.pop()
		got = append(got, item)
	}

	fmt.Println(got)
	fmt.Println("through disk:", q.spilled)

	// Output:
	// [a b c d e]
	// through disk: 3
}
