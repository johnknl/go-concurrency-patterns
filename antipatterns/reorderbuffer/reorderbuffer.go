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

// item is a value tagged at the source with its position in the input, counting from 0.
type item struct {
	val string
	seq int
}

// reorderUnbounded emits items in sequence order and buffers everything that arrives ahead of a gap, with no
// limit. When an item never arrives, nothing after it is emitted, and the buffer grows by one for every item that
// does. It returns how many items were still buffered when the input ended.
func reorderUnbounded(in <-chan item, emit func(item)) int {
	pending := map[int]item{}
	next := 0

	for it := range in {
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
	}

	return len(pending)
}

// reorderBounded emits items in sequence order and holds at most limit items behind a gap. One more, or the end
// of the input, gives up on the gap: it reports each missing sequence number to skip and moves past it. An item
// that arrives after its gap was given up on, or twice, is reported to late and dropped.
func reorderBounded(in <-chan item, limit int, emit func(item), skip func(seq int), late func(item)) {
	pending := make(map[int]item, limit+1)
	next := 0

	advance := func() {
		for {
			v, ok := pending[next]
			if !ok {
				return
			}

			delete(pending, next)
			emit(v)
			next++
		}
	}

	for it := range in {
		if _, dup := pending[it.seq]; dup || it.seq < next {
			late(it)

			continue
		}

		pending[it.seq] = it
		advance()

		for len(pending) > limit {
			skip(next)
			next++
			advance()
		}
	}

	for len(pending) > 0 {
		skip(next)
		next++
		advance()
	}
}
