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

package onlineattach

import (
	"context"
	"errors"
	"fmt"
)

// Example_onlineAttach shows two subscribers that receive every message, one that detaches halfway, a late subscriber
// that is caught up from the history, and a slow subscriber that loses messages.
func Example_onlineAttach() {
	ctx, cancel := context.WithCancelCause(context.Background())
	errClosed := errors.New("hub closed")

	h := newHub(ctx)

	a := h.subscribe(4)
	b := h.subscribe(4)
	slow := h.subscribe(1)

	h.publish(1)
	h.publish(2)

	h.unsubscribe(b)

	h.publish(3)

	late := h.subscribe(4)

	h.publish(4)

	cancel(errClosed)
	<-h.done

	fmt.Println(drain("a", a))
	fmt.Println(drain("b", b))
	fmt.Println(drain("late", late))
	fmt.Println(drain("slow", slow))
	fmt.Println("dropped", h.dropped)

	// Output:
	// a [1 2 3 4] hub closed
	// b [1 2] detached
	// late [2 3 4] hub closed
	// slow [1] hub closed
	// dropped 3
}
