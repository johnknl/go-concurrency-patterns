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

// Mutants that survive on purpose:
//   - workers < 1 to workers <= 1: both give one worker for workers == 1.
//   - unbuffer-channel on in of pool, buffer-channel on in of processAll and processUntilError: a buffer only
//     changes when a worker picks up a job.
//   - remove-cancel of the deferred cancel in processUntilError: it releases the context's resources, which no
//     test observes.

import (
	"context"
	"errors"
	"slices"
	"sync/atomic"
	"testing"
)

func TestPool(t *testing.T) {
	names := []string{"a", "bad", "c", "d"}

	t.Run("ordered keeps the input order and leaves zero values", func(t *testing.T) {
		out, err := pool(names, process, options{workers: 3, ordered: true})
		if err != nil || !slices.Equal(out, []string{"a.done", "", "c.done", "d.done"}) {
			t.Fatalf("got %q, %v", out, err)
		}
	})

	t.Run("unordered drops failed jobs", func(t *testing.T) {
		out, err := pool(names, process, options{buffer: 2})
		slices.Sort(out)

		if err != nil || !slices.Equal(out, []string{"a.done", "c.done", "d.done"}) {
			t.Fatalf("got %q, %v", out, err)
		}
	})

	t.Run("onError sees the index", func(t *testing.T) {
		var got []int

		_, _ = pool(names, process, options{onError: func(i int, _ error) { got = append(got, i) }})
		if !slices.Equal(got, []int{1}) {
			t.Fatalf("got %v", got)
		}
	})

	t.Run("stopOnError returns the error and disables onError", func(t *testing.T) {
		var calls, runs atomic.Int64

		f := func(s string) (string, error) {
			runs.Add(1)
			return process(s)
		}

		out, err := pool([]string{"bad", "a", "b"}, f, options{
			stopOnError: true,
			onError:     func(int, error) { calls.Add(1) },
		})
		if out != nil || !errors.Is(err, errBad) || calls.Load() != 0 {
			t.Fatalf("got %q, %v, %d calls", out, err, calls.Load())
		}

		if runs.Load() != 1 {
			t.Fatalf("%d runs after the error, want 1", runs.Load())
		}
	})
}

func TestProcessAll(t *testing.T) {
	out, err := processAll([]string{"a", "bad", "c"})
	if !slices.Equal(out, []string{"a.done", "", "c.done"}) || !errors.Is(err, errBad) {
		t.Fatalf("got %q, %v", out, err)
	}

	if out, err := processAll([]string{"x"}); err != nil || out[0] != "x.done" {
		t.Fatalf("got %q, %v", out, err)
	}
}

func TestProcessUntilError(t *testing.T) {
	var logged atomic.Int64

	logf := func(error) { logged.Add(1) }

	if err := processUntilError(context.Background(), []string{"a", "b"}, logf); err != nil || logged.Load() != 0 {
		t.Fatalf("got %v, %d logged", err, logged.Load())
	}

	err := processUntilError(context.Background(), []string{"bad", "a", "bad", "bad"}, logf)
	if !errors.Is(err, errBad) || logged.Load() < 1 {
		t.Fatalf("got %v, %d logged", err, logged.Load())
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	if err := processUntilError(ctx, []string{"bad"}, logf); !errors.Is(err, context.Canceled) {
		t.Fatalf("got %v", err)
	}
}
