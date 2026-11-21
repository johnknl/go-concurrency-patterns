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

package loadshedding

// dropNewest discards v when the queue is full. It reports whether v was queued.
func dropNewest(queue chan int, v int) bool {
	select {
	case queue <- v:
		return true
	default:
		return false
	}
}

// dropOldest makes room by discarding the head of the queue. With several senders, another sender can fill
// the freed slot, so one call may evict more than one value while reporting one drop.
func dropOldest(queue chan int, v int) (dropped bool) {
	for {
		select {
		case queue <- v:
			return dropped
		default:
		}

		select {
		case <-queue:
			dropped = true
		default:
		}
	}
}

func drain(queue chan int) []int {
	close(queue)

	var got []int
	for v := range queue {
		got = append(got, v)
	}

	return got
}
