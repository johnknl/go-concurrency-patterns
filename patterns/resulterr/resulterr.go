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

package resulterr

import (
	"context"
	"errors"
	"fmt"
)

var errNotFound = errors.New("not found")

// result carries the outcome of one unit of work: the value and the error travel together.
type result struct {
	err   error
	value string
	id    int
}

func lookup(id int) (string, error) {
	if id%2 == 0 {
		return "", fmt.Errorf("lookup %d: %w", id, errNotFound)
	}

	return fmt.Sprintf("record-%d", id), nil
}

// lookupAll does the work and reports every outcome. It does not decide what an error means.
func lookupAll(ctx context.Context, ids ...int) <-chan result {
	out := make(chan result)

	go func() {
		defer close(out)

		for _, id := range ids {
			value, err := lookup(id)

			select {
			case out <- result{id: id, value: value, err: err}:
			case <-ctx.Done():
				return
			}
		}
	}()

	return out
}

// lookupQuietly is the contrast: it logs the error and moves on, so its consumer cannot tell that anything failed.
func lookupQuietly(logs *[]string, ids ...int) <-chan string {
	out := make(chan string)

	go func() {
		defer close(out)

		for _, id := range ids {
			value, err := lookup(id)
			if err != nil {
				*logs = append(*logs, err.Error())
				continue
			}

			out <- value
		}
	}()

	return out
}
