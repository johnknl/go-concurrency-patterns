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
	"sync"
)

// badSendAfterClose plays the receiver that closes the channel it reads from.
// The sender's next send panics.
func badSendAfterClose() (msg any) {
	defer func() { msg = recover() }()
	ch := make(chan int, 1)
	close(ch)
	ch <- 1
	return nil
}

// badCloseTwice plays two goroutines that both consider themselves the closer.
func badCloseTwice() (msg any) {
	defer func() { msg = recover() }()
	ch := make(chan int)
	close(ch)
	close(ch)
	return nil
}

// produce starts several senders. None of them closes out: a single closer
// goroutine does, after every sender has returned. A receiver that wants to
// stop closes done, which the senders watch.
func produce(senders int, done <-chan struct{}) <-chan int {
	out := make(chan int)
	var wg sync.WaitGroup
	for i := range senders {
		wg.Go(func() {
			for j := range 2 {
				select {
				case out <- i*10 + j:
				case <-done:
					return
				}
			}
		})
	}
	go func() {
		wg.Wait()
		close(out)
	}()
	return out
}
