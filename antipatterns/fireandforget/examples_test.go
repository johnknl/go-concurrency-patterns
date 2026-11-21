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

package fireandforget

import (
	"errors"
	"fmt"
)

func Example_fireAndForget() {
	lost := make(chan error)

	err := leakyStart(func() error { return errors.New("disk full") }, func(err error) { lost <- err })
	fmt.Println("start returned:", err)

	// Only the injected hook ever sees the error; receiving here also lets the goroutine finish.
	fmt.Println("error seen elsewhere:", <-lost)

	// Output:
	// start returned: <nil>
	// error seen elsewhere: disk full
}

func Example_owned() {
	err := runAll(
		func() error { return nil },
		func() error { return errors.New("disk full") },
		func() error { return errors.New("timeout") },
	)
	fmt.Println(err)

	// Output:
	// disk full
	// timeout
}
