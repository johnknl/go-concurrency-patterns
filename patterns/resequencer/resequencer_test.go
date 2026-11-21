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
//   - buffer-channel on jobs and results in orderedMap: the tokens bound the items in flight whatever the
//     channels buffer, so order and the window are unchanged.
//   - buffer-channel on done in orderedMap: it is only closed, never sent on.

package resequencer

import (
	"errors"
	"fmt"
	"math/rand/v2"
	"slices"
	"strings"
	"sync/atomic"
	"testing"
	"testing/synctest"
)

func feed(seqs []int) <-chan item {
	in := make(chan item, len(seqs))
	for _, s := range seqs {
		in <- item{seq: s, val: fmt.Sprint(s)}
	}
	close(in)

	return in
}

func collect(seqs []int, limit int) ([]int, error) {
	var got []int
	err := resequence(feed(seqs), limit, func(it item) { got = append(got, it.seq) })

	return got, err
}

func TestResequence(t *testing.T) {
	tests := []struct {
		name  string
		seqs  []int
		limit int
		want  []int
		err   error
	}{
		{"empty", nil, 0, nil, nil},
		{"in order needs no buffer", []int{0, 1, 2, 3}, 0, []int{0, 1, 2, 3}, nil},
		{"reversed", []int{3, 2, 1, 0}, 3, []int{0, 1, 2, 3}, nil},
		{"interleaved", []int{1, 0, 3, 2, 5, 4}, 1, []int{0, 1, 2, 3, 4, 5}, nil},
		{"bound reached exactly", []int{2, 1, 0}, 2, []int{0, 1, 2}, nil},
		{"bound exceeded", []int{3, 2, 1, 0}, 2, nil, errGapTooLarge},
		{"bound exceeded after progress", []int{0, 2, 3}, 1, []int{0}, errGapTooLarge},
		{"duplicate already emitted", []int{0, 1, 0}, 2, []int{0, 1}, errDuplicate},
		{"duplicate still buffered", []int{1, 1}, 2, nil, errDuplicate},
		{"missing at the end", []int{0, 2}, 2, []int{0}, errMissing},
		{"missing first", []int{1}, 2, nil, errMissing},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := collect(tt.seqs, tt.limit)
			if !slices.Equal(got, tt.want) {
				t.Errorf("emitted %v, want %v", got, tt.want)
			}
			if !errors.Is(err, tt.err) {
				t.Errorf("err = %v, want %v", err, tt.err)
			}
		})
	}
}

func TestResequence_shuffled(t *testing.T) {
	r := rand.New(rand.NewPCG(1, 2))
	for n := range 50 {
		seqs := r.Perm(n)
		got, err := collect(seqs, n)
		if err != nil || len(got) != n {
			t.Fatalf("perm %v: got %v, %v", seqs, got, err)
		}
		for i, s := range got {
			if s != i {
				t.Fatalf("perm %v: got %v", seqs, got)
			}
		}
	}
}

func TestResequence_gapWaits(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		in := make(chan item)
		var emitted atomic.Int64
		done := make(chan error)
		go func() { done <- resequence(in, 8, func(item) { emitted.Add(1) }) }()

		for _, s := range []int{1, 2, 3} {
			in <- item{seq: s}
		}
		synctest.Wait()
		if n := emitted.Load(); n != 0 {
			t.Fatalf("emitted %d before the gap was filled", n)
		}

		in <- item{seq: 0}
		synctest.Wait()
		if n := emitted.Load(); n != 4 {
			t.Fatalf("emitted %d after the gap was filled, want 4", n)
		}

		close(in)
		if err := <-done; err != nil {
			t.Fatal(err)
		}
	})
}

func TestOrderedMap(t *testing.T) {
	t.Run("keeps input order with any completion order", func(t *testing.T) {
		vals := strings.Fields("a b c d e f g h i j")
		for _, workers := range []int{1, 2, 5} {
			for _, window := range []int{1, 2, 10} {
				got, err := orderedMap(vals, workers, window, strings.ToUpper)
				want := strings.Fields("A B C D E F G H I J")
				if err != nil || !slices.Equal(got, want) {
					t.Fatalf("workers %d window %d: %v, %v", workers, window, got, err)
				}
			}
		}
	})

	t.Run("a slow first item holds the window", func(t *testing.T) {
		synctest.Test(t, func(t *testing.T) {
			release := make(chan struct{})
			var started atomic.Int64
			f := func(s string) string {
				started.Add(1)
				if s == "a" {
					<-release
				}
				return s
			}
			done := make(chan []string)
			go func() {
				out, _ := orderedMap(strings.Fields("a b c d e f"), 4, 3, f)
				done <- out
			}()
			synctest.Wait()
			if n := started.Load(); n != 3 {
				t.Fatalf("%d items started behind a stalled first item, want the window of 3", n)
			}
			close(release)
			if got := <-done; !slices.Equal(got, strings.Fields("a b c d e f")) {
				t.Fatalf("got %v", got)
			}
		})
	})

	t.Run("empty input", func(t *testing.T) {
		got, err := orderedMap(nil, 2, 2, strings.ToUpper)
		if err != nil || len(got) != 0 {
			t.Fatalf("got %v, %v", got, err)
		}
	})
}
