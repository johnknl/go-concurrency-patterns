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

package implicitstate

import "fmt"

func Example_stopDuringStart() {
	s := &badServer{}
	// Stop arrives while Start runs its setup outside the lock.
	err := s.Start(func() { s.Stop() })
	fmt.Println("start error:", err)
	fmt.Println("started:", s.started, "closed:", s.closed)
	fmt.Println("running:", s.Running())
	fmt.Println("start again:", s.Start(func() {}))
	// Output:
	// start error: <nil>
	// started: true closed: true
	// running: false
	// start again: already started or closed
}

func Example_explicitState() {
	s := &server{}
	err := s.Start(func() {
		fmt.Println("during start:", s.State())
		s.Stop()
	})
	fmt.Println("start error:", err)
	fmt.Println("state:", s.State())
	fmt.Println("start again:", s.Start(func() {}))
	// Output:
	// during start: starting
	// start error: stopped during start
	// state: stopped
	// start again: not idle
}
