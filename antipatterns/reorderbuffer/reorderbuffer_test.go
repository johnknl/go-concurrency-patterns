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

// Mutants that survive on purpose:
//   - ARITHMETIC_BASE on limit+1 in reorderBounded: it is only the map's size hint.

package reorderbuffer

import (
	"fmt"
	"slices"
	"testing"
)

func feed(seqs []int) <-chan item {
	in := make(chan item, len(seqs))
	for _, s := range seqs {
		in <- item{seq: s}
	}
	close(in)

	return in
}

func TestReorderUnbounded(t *testing.T) {
	var got []int
	left := reorderUnbounded(feed([]int{2, 0, 1, 4, 5}), func(it item) { got = append(got, it.seq) })
	if !slices.Equal(got, []int{0, 1, 2}) || left != 2 {
		t.Fatalf("emitted %v, left %d; want [0 1 2], 2", got, left)
	}
}

func TestReorderBounded(t *testing.T) {
	tests := []struct {
		name  string
		seqs  []int
		limit int
		want  string
	}{
		{"empty", nil, 2, ""},
		{"in order", []int{0, 1, 2}, 0, "e0 e1 e2"},
		{"reorders within the bound", []int{2, 1, 0}, 2, "e0 e1 e2"},
		{"gives up at the bound", []int{1, 2, 3}, 2, "s0 e1 e2 e3"},
		{"bound of zero skips at once", []int{1, 0}, 0, "s0 e1 l0"},
		{"skips a run of gaps", []int{3, 4, 5}, 1, "s0 s1 s2 e3 e4 e5"},
		{"gap at the end of input", []int{0, 2, 4}, 5, "e0 s1 e2 s3 e4"},
		{"duplicate buffered", []int{1, 1, 0}, 3, "l1 e0 e1"},
		{"duplicate emitted", []int{0, 0}, 3, "e0 l0"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var log []string
			reorderBounded(feed(tt.seqs), tt.limit,
				func(it item) { log = append(log, fmt.Sprint("e", it.seq)) },
				func(seq int) { log = append(log, fmt.Sprint("s", seq)) },
				func(it item) { log = append(log, fmt.Sprint("l", it.seq)) },
			)
			if got := fmt.Sprint(log); got != "["+tt.want+"]" {
				t.Errorf("got %s, want [%s]", got, tt.want)
			}
		})
	}
}
