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
	"errors"
	"fmt"
	"sync"
)

var (
	errDuplicate   = errors.New("duplicate sequence number")
	errGapTooLarge = errors.New("too many items buffered behind a gap")
	errMissing     = errors.New("missing sequence number")
)

// item is a value tagged at the source with its position in the input, counting from 0.
type item struct {
	val string
	seq int
}

// resequence receives items in any order and calls emit with them in sequence order, in the caller's goroutine.
// It holds at most limit items that arrived ahead of a gap; one more, a duplicate, or an input that ends with a
// gap still open is an error. On an error it returns at once and stops receiving, so the caller must stop the
// senders.
func resequence(in <-chan item, limit int, emit func(item)) error {
	pending := make(map[int]item, limit)
	next := 0

	for it := range in {
		if _, ok := pending[it.seq]; ok || it.seq < next {
			return fmt.Errorf("seq %d: %w", it.seq, errDuplicate)
		}

		pending[it.seq] = it

		for {
			v, ok := pending[next]
			if !ok {
				break
			}

			delete(pending, next)
			emit(v)
			next++
		}

		if len(pending) > limit {
			return fmt.Errorf("waiting for seq %d with %d buffered: %w", next, len(pending), errGapTooLarge)
		}
	}

	if len(pending) > 0 {
		return fmt.Errorf("seq %d: %w", next, errMissing)
	}

	return nil
}

// orderedMap applies f to vals with the given number of workers and returns the results in input order. A token
// taken before an item is dispatched and returned when its result is emitted keeps at most window items between
// the source and the output, so the resequencer never holds more than window-1, and with contiguous, unique
// sequence numbers it cannot fail. The source goroutine, the workers, and the closer all end before it returns.
func orderedMap(vals []string, workers, window int, f func(string) string) ([]string, error) {
	tokens := make(chan struct{}, window)
	jobs := make(chan item)
	results := make(chan item)

	go func() {
		defer close(jobs)

		for i, v := range vals {
			tokens <- struct{}{}
			jobs <- item{seq: i, val: v}
		}
	}()

	var wg sync.WaitGroup

	for range workers {
		wg.Go(func() {
			for it := range jobs {
				results <- item{seq: it.seq, val: f(it.val)}
			}
		})
	}

	done := make(chan struct{})

	go func() {
		defer close(done)

		wg.Wait()
		close(results)
	}()

	out := make([]string, 0, len(vals))
	err := resequence(results, window, func(it item) {
		out = append(out, it.val)
		<-tokens
	})

	<-done

	return out, err
}
