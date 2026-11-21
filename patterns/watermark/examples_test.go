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
	"context"
	"fmt"
)

// send sends msgs on a new input from their own goroutine, which closes the input after the last one.
func send(msgs ...message) <-chan message {
	input := make(chan message)

	go func() {
		defer close(input)

		for _, m := range msgs {
			input <- m
		}
	}()

	return input
}

// Example_merge merges two inputs that send at their own pace into one stream in event-time order.
func Example_merge() {
	a := send(message{at: 1, value: "a1"}, message{at: 4, value: "a4"}, message{at: 6, value: "a6"})
	b := send(message{at: 2, value: "b2"}, message{at: 3, value: "b3"}, message{at: 7, value: "b7"})

	err := merge(context.Background(), []<-chan message{a, b}, 10, func(m message) {
		fmt.Println(m.at, m.value)
	})
	fmt.Println("err:", err)

	// Output:
	// 1 a1
	// 2 b2
	// 3 b3
	// 4 a4
	// 6 a6
	// 7 b7
	// err: <nil>
}

// Example_punctuation shows an input without events that sends a punctuation instead, so the events of the
// other input that lie before the watermark, 3, are emitted while both inputs stay open. The event at 4 is
// emitted when the inputs close.
func Example_punctuation() {
	a := make(chan message)
	idle := make(chan message)
	seen := make(chan struct{})

	go func() {
		defer close(a)

		for _, at := range []int{1, 2, 3, 4} {
			a <- message{at: at, value: "a"}
		}
		<-seen
	}()

	go func() {
		defer close(idle)

		idle <- message{at: 10, punct: true}
		<-seen
	}()

	err := merge(context.Background(), []<-chan message{a, idle}, 10, func(m message) {
		fmt.Println(m.at, m.value)
		if m.at == 3 {
			close(seen)
		}
	})
	fmt.Println("err:", err)

	// Output:
	// 1 a
	// 2 a
	// 3 a
	// 4 a
	// err: <nil>
}

// Example_windows counts events per event-time window of 5, with an allowed lateness of 2.
func Example_windows() {
	in := send(
		message{at: 1}, message{at: 3}, message{at: 6},
		message{at: 6, punct: true},
		message{at: 4}, // behind the watermark, within the allowed lateness
		message{at: 8, punct: true},
		message{at: 2}, // its window has been emitted
		message{at: 9},
	)

	err := windows(context.Background(), in, 5, 2, 10,
		func(w window) { fmt.Printf("[%d, %d): %d events\n", w.start, w.start+5, w.count) },
		func(m message) { fmt.Println("late:", m.at) },
	)
	fmt.Println("err:", err)

	// Output:
	// [0, 5): 3 events
	// late: 2
	// [5, 10): 2 events
	// err: <nil>
}
