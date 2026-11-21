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

package healing

import (
	"context"
)

// ward is a restartable unit of work. It sends pulses on the first channel and results on the second, and
// closes both when it ends.
type ward func(ctx context.Context) (<-chan struct{}, <-chan int)

// steward starts a ward and restarts it when no pulse arrives before the channel from timeout fires. The
// ward's results are forwarded on the returned channel.
func steward(ctx context.Context, timeout func() <-chan struct{}, start ward) <-chan int {
	out := make(chan int)

	go func() {
		defer close(out)

		for {
			wardCtx, cancel := context.WithCancel(ctx)
			heartbeat, results := start(wardCtx)
			expired := timeout()

		watch:
			for {
				select {
				case <-ctx.Done():
					cancel()
					return
				case _, ok := <-heartbeat:
					if !ok {
						cancel()
						return
					}

					expired = timeout()
				case n, ok := <-results:
					if !ok {
						results = nil

						continue
					}

					select {
					case out <- n:
					case <-ctx.Done():
						cancel()
						return
					}
				case <-expired:
					cancel()

					for range heartbeat {
					}

					break watch
				}
			}
		}
	}()

	return out
}

// supervised turns a steward over start into a ward, so that another steward can supervise it. It pulses on
// every forwarded result and closes its channels only after the inner steward has closed its own.
func supervised(timeout func() <-chan struct{}, start ward) ward {
	return func(ctx context.Context) (<-chan struct{}, <-chan int) {
		heartbeat := make(chan struct{}, 1)
		results := make(chan int)

		go func() {
			defer close(heartbeat)
			defer close(results)

			for n := range steward(ctx, timeout, start) {
				select {
				case heartbeat <- struct{}{}:
				default:
				}

				select {
				case results <- n:
				case <-ctx.Done():
				}
			}
		}()

		return heartbeat, results
	}
}
