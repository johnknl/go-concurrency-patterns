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

package wrongsideclose

import (
	"fmt"
)

func Example_wrongSideClose() {
	fmt.Println(badSendAfterClose())
	fmt.Println(badCloseTwice())
	// Output:
	// send on closed channel
	// close of closed channel
}

func Example_fixed() {
	sum := 0
	for v := range produce(3, nil) {
		sum += v
	}
	fmt.Println("sum:", sum)

	done := make(chan struct{})
	out := produce(3, done)
	<-out
	close(done)
	for range out {
	}
	fmt.Println("stopped early")
	// Output:
	// sum: 63
	// stopped early
}
