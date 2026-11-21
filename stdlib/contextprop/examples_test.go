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

package contextprop

import (
	"context"
	"errors"
	"fmt"
	"time"
)

// Example_tree cancels a parent and shows that every derived context ends, while a canceled child leaves its parent alone.
func Example_tree() {
	root, cancelRoot := context.WithCancel(context.Background())
	defer cancelRoot()

	left, cancelLeft := context.WithCancel(root)
	defer cancelLeft()

	right, cancelRight := context.WithCancel(root)
	defer cancelRight()

	cancelLeft()
	<-left.Done()
	fmt.Println("left:", left.Err(), "| root:", root.Err(), "| right:", right.Err())

	result := make(chan string)

	go worker(right, "right", result)

	cancelRoot()
	fmt.Println(<-result)
	fmt.Println("root:", root.Err())

	// Output:
	// left: context canceled | root: <nil> | right: <nil>
	// right: context canceled
	// root: context canceled
}

// Example_deadline shows that a deadline in the past ends the context at once, with a distinct error.
func Example_deadline() {
	parent, cancelParent := context.WithCancel(context.Background())
	defer cancelParent()

	child, cancel := context.WithDeadline(parent, time.Unix(0, 0))
	defer cancel()

	<-child.Done()
	fmt.Println(child.Err())
	fmt.Println(errors.Is(child.Err(), context.DeadlineExceeded))
	fmt.Println(parent.Err())

	// Output:
	// context deadline exceeded
	// true
	// <nil>
}

// Example_cause cancels the root with a cause; every derived context reports Canceled from Err and the
// root's cause from context.Cause.
func Example_cause() {
	root, cancel := context.WithCancelCause(context.Background())
	defer cancel(nil)

	child, stop := context.WithTimeout(root, time.Hour)
	defer stop()

	causes := make(chan error)

	go call(child, causes)
	go call(child, causes)

	cancel(errors.New("client gone"))

	fmt.Println(<-causes)
	fmt.Println(<-causes)
	fmt.Println(child.Err())

	// Output:
	// client gone
	// client gone
	// context canceled
}

// Example_values passes a request-scoped value down the tree; contexts derived from it see it too.
func Example_values() {
	c := withRequestID(context.Background(), "req-42")

	child, cancel := context.WithCancel(c)
	defer cancel()

	id, ok := requestID(child)
	fmt.Println(id, ok)

	_, ok = requestID(context.Background())
	fmt.Println(ok)

	// Output:
	// req-42 true
	// false
}
