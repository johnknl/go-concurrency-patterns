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
//

package needlessasync

import (
	"context"
	"errors"
	"fmt"
	"sync/atomic"
)

var errNotFound = errors.New("not found")

// lookup is the work: fast, and it can fail.
func lookup(id int) (string, error) {
	if id < 0 {
		return "", fmt.Errorf("user %d: %w", id, errNotFound)
	}

	return fmt.Sprintf("user-%d", id), nil
}

type result struct {
	err  error
	name string
}

// pending counts the goroutines of userAsync that have not delivered their result yet.
var pending atomic.Int64

// userAsync is the asynchronous form. It starts a goroutine and returns a channel, so the caller cannot pass a
// context, has to receive to learn the result or the error, and leaves the goroutine blocked forever when it
// stops listening.
func userAsync(id int) <-chan result {
	ch := make(chan result)

	pending.Add(1)

	go func() {
		defer pending.Add(-1)

		name, err := lookup(id)
		ch <- result{name: name, err: err}
	}()

	return ch
}

// user is the synchronous form. It returns its result, honors the context, and starts nothing. A caller that
// wants concurrency adds it with go and a WaitGroup.
func user(ctx context.Context, id int) (string, error) {
	if err := ctx.Err(); err != nil {
		return "", err
	}

	return lookup(id)
}
