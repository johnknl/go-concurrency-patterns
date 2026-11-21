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

package racing

import (
	"context"
	"sync"
)

// replica answers with its name once released, or records that it was canceled.
func replica(name string, release <-chan struct{}, canceled chan<- string) func(context.Context) string {
	return func(ctx context.Context) string {
		select {
		case <-release:
			return name
		case <-ctx.Done():
			canceled <- name
			return ""
		}
	}
}

// first sends the same request to every replica, returns the first answer, cancels the others, and waits
// until they have ended.
func first(ctx context.Context, replicas ...func(context.Context) string) string {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	answers := make(chan string, len(replicas))

	var wg sync.WaitGroup

	for _, r := range replicas {
		wg.Go(func() { answers <- r(ctx) })
	}

	answer := <-answers

	cancel()
	wg.Wait()

	return answer
}
