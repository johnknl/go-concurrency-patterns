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

package sleepsync

import (
	"fmt"
)

func Example_sleep() {
	gate := make(chan struct{})
	j := startJob(gate)

	// The job cannot finish before the gate opens, and the gate opens only after the sleep returned.
	fmt.Println("read after sleep:", badRead(j))

	close(gate)
	<-j.done

	fmt.Println("result:", j.result.Load())

	// Output:
	// read after sleep: 0
	// result: 42
}

func Example_wait() {
	gate := make(chan struct{})
	j := startJob(gate)

	go close(gate)

	fmt.Println("read after wait:", read(j))

	// Output:
	// read after wait: 42
}
