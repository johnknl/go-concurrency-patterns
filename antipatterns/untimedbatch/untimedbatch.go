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

package untimedbatch

import "time"

// sizeOnly flushes only when a batch is full. Under low load a partial batch waits for items that may never come,
// and when in is closed the partial batch is dropped.
func sizeOnly(in <-chan string, maxSize int, flush func([]string)) <-chan struct{} {
	done := make(chan struct{})

	go func() {
		defer close(done)

		var batch []string

		for item := range in {
			batch = append(batch, item)
			if len(batch) == maxSize {
				flush(batch)
				batch = nil
			}
		}
	}()

	return done
}

// sizeOrTime is the fix, the Batcher pattern. It flushes a batch when it holds maxSize items, when maxWait has
// passed since its first item, or when in is closed, and closes the returned channel after the last flush.
func sizeOrTime(in <-chan string, maxSize int, maxWait time.Duration, flush func([]string)) <-chan struct{} {
	done := make(chan struct{})

	go func() {
		defer close(done)

		timer := time.NewTimer(maxWait)
		timer.Stop()

		var batch []string

		send := func() {
			timer.Stop()
			flush(batch)
			batch = nil
		}

		for {
			select {
			case item, ok := <-in:
				if !ok {
					if len(batch) > 0 {
						send()
					}

					return
				}

				batch = append(batch, item)
				if len(batch) == 1 {
					timer.Reset(maxWait)
				}

				if len(batch) == maxSize {
					send()
				}
			case <-timer.C:
				send()
			}
		}
	}()

	return done
}
