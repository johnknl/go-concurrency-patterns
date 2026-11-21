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

package future

import "errors"

var errNegative = errors.New("negative input")

// square is the work: an ordinary synchronous function that returns its result and its error.
func square(n int) (int, error) {
	if n < 0 {
		return 0, errNegative
	}

	return n * n, nil
}

type result struct {
	err error
	val int
}

// start is the future, owned by the calling code: it runs fn in a goroutine and returns a channel that will
// receive the one result. The channel has room for it, so the goroutine can finish even when the caller stops
// waiting.
func start(fn func() (int, error)) <-chan result {
	ch := make(chan result, 1)

	go func() {
		v, err := fn()
		ch <- result{val: v, err: err}
	}()

	return ch
}
