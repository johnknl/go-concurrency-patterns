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

package watermark

import (
	"cmp"
	"context"
	"errors"
	"maps"
	"math"
	"slices"
	"sync"
)

// message is an event with its event time, or a punctuation that promises no more events before at.
type message struct {
	value string
	at    int
	punct bool
}

// tagged is a message as the owner receives it: which input sent it, or that the input has closed. seq counts
// the messages the owner has received.
type tagged struct {
	msg    message
	input  int
	seq    int
	closed bool
}

var errFull = errors.New("watermark: buffer full")

// merge emits the events of inputs in event-time order. Each input must send its messages in non-decreasing
// event time. An event is emitted once the watermark, the least latest time over the open inputs, has reached
// it. More than limit waiting events is an error.
func merge(ctx context.Context, inputs []<-chan message, limit int, emit func(message)) error {
	ctx, cancel := context.WithCancel(ctx)

	var wg sync.WaitGroup
	defer wg.Wait()
	defer cancel()

	in := make(chan tagged)
	for i, input := range inputs {
		wg.Go(func() { forward(ctx, i, input, in) })
	}

	latest := make(map[int]int) // per open input that has sent; open inputs that have not sent are absent
	open := len(inputs)
	closed := make([]bool, len(inputs))
	var waiting []tagged // ordered by before

	for seq := 0; open > 0; seq++ {
		var t tagged
		select {
		case <-ctx.Done():
			return ctx.Err()
		case t = <-in:
		}
		t.seq = seq

		switch {
		case t.closed:
			closed[t.input] = true
			delete(latest, t.input)
			open--
		case t.msg.punct:
			latest[t.input] = t.msg.at
		default:
			latest[t.input] = t.msg.at
			i, _ := slices.BinarySearchFunc(waiting, t, before)
			waiting = slices.Insert(waiting, i, t)
		}

		wm, ok := watermark(latest, open)
		for ok && len(waiting) > 0 && waiting[0].msg.at <= wm {
			emit(waiting[0].msg)
			waiting = waiting[1:]
		}

		if len(waiting) > limit {
			return errFull
		}
	}

	return nil
}

// watermark returns the least latest time over the open inputs. It is not known while an open input has not
// sent anything. When no input is open, every event is due.
func watermark(latest map[int]int, open int) (int, bool) {
	if len(latest) < open {
		return 0, false
	}

	if open == 0 {
		return math.MaxInt, true
	}

	return slices.Min(slices.Collect(maps.Values(latest))), true
}

// forward hands the messages of one input to the owner, then reports that the input has closed.
func forward(ctx context.Context, i int, input <-chan message, out chan<- tagged) {
	for {
		t := tagged{input: i}

		select {
		case <-ctx.Done():
			return
		case m, ok := <-input:
			t.msg, t.closed = m, !ok
		}

		select {
		case <-ctx.Done():
			return
		case out <- t:
		}

		if t.closed {
			return
		}
	}
}

// before orders events by event time, then by input, then by arrival, so that equal times come out in a fixed
// order and the events of one input in the order it sent them.
func before(a, b tagged) int {
	return cmp.Or(cmp.Compare(a.msg.at, b.msg.at), cmp.Compare(a.input, b.input), cmp.Compare(a.seq, b.seq))
}

// window is the number of events with event time in [start, start+size).
type window struct {
	start int
	count int
}

// windows counts the events of in per tumbling event-time window of the given size. A punctuation on in carries
// the watermark. A window is emitted once the watermark has passed its end by lateness; an event for a window
// that has already been emitted is passed to late. Event times are not negative. More than limit open windows is
// an error.
func windows(
	ctx context.Context, in <-chan message, size, lateness, limit int, emit func(window), late func(message),
) error {
	open := make(map[int]int) // window start -> count
	wm := math.MinInt

	for {
		var m message
		var ok bool
		select {
		case <-ctx.Done():
			return ctx.Err()
		case m, ok = <-in:
		}

		if !ok {
			wm = math.MaxInt
		} else if m.punct {
			wm = max(wm, m.at)
		} else {
			start := m.at - m.at%size
			if _, isOpen := open[start]; !isOpen && start+size+lateness <= wm {
				late(m)
			} else {
				open[start]++
			}
		}

		for _, start := range slices.Sorted(maps.Keys(open)) {
			if start+size+lateness > wm {
				break
			}
			emit(window{start: start, count: open[start]})
			delete(open, start)
		}

		if len(open) > limit {
			return errFull
		}

		if !ok {
			return nil
		}
	}
}
