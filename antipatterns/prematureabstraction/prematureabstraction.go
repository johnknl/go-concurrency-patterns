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

package prematureabstraction

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
)

var errBad = errors.New("bad input")

// process is the work both callers do.
func process(name string) (string, error) {
	if name == "bad" {
		return "", fmt.Errorf("%s: %w", name, errBad)
	}

	return name + ".done", nil
}

// options configure pool. Each field was added for one caller; together they make six error and order modes.
type options struct {
	onError     func(i int, err error)
	workers     int
	buffer      int
	ordered     bool
	stopOnError bool
}

// pool is the premature abstraction: a generic worker pool with flags, written before its callers were known.
func pool[J, R any](jobs []J, f func(J) (R, error), o options) ([]R, error) {
	if o.workers < 1 {
		o.workers = 1
	}

	var (
		mu       sync.Mutex
		out      []R
		firstErr error
		stopped  atomic.Bool
		wg       sync.WaitGroup
	)

	if o.ordered {
		out = make([]R, len(jobs))
	}

	in := make(chan int, o.buffer)

	for range o.workers {
		wg.Go(func() {
			for i := range in {
				if o.stopOnError && stopped.Load() {
					continue
				}

				r, err := f(jobs[i])

				mu.Lock()

				switch {
				case err != nil && o.stopOnError:
					if firstErr == nil {
						firstErr = err
					}

					stopped.Store(true)
				case err != nil && o.onError != nil:
					o.onError(i, err)
				case err != nil:
					// dropped
				case o.ordered:
					out[i] = r
				default:
					out = append(out, r)
				}

				mu.Unlock()
			}
		})
	}

	for i := range jobs {
		in <- i
	}

	close(in)
	wg.Wait()

	if firstErr != nil {
		return nil, firstErr
	}

	return out, nil
}

// processAll is the fix for the caller that wants every result in input order and every error.
func processAll(names []string) ([]string, error) {
	out := make([]string, len(names))
	errs := make([]error, len(names))
	in := make(chan int)

	var wg sync.WaitGroup
	for range 2 {
		wg.Go(func() {
			for i := range in {
				out[i], errs[i] = process(names[i])
			}
		})
	}

	for i := range names {
		in <- i
	}

	close(in)
	wg.Wait()

	return out, errors.Join(errs...)
}

// processUntilError is the fix for the caller that stops at the first error and logs it.
func processUntilError(ctx context.Context, names []string, logf func(error)) error {
	ctx, cancel := context.WithCancelCause(ctx)
	defer cancel(nil)

	in := make(chan string)

	var wg sync.WaitGroup
	for range 2 {
		wg.Go(func() {
			for name := range in {
				if ctx.Err() != nil {
					continue
				}

				if _, err := process(name); err != nil {
					logf(err)
					cancel(err)
				}
			}
		})
	}

	for _, name := range names {
		in <- name
	}

	close(in)
	wg.Wait()

	return context.Cause(ctx)
}
