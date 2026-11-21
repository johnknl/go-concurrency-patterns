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

// Mutants that survive on purpose: none.

package fireandforget

import (
	"errors"
	"slices"
	"sync/atomic"
	"testing"
	"testing/synctest"
)

func TestLeakyStart(t *testing.T) {
	t.Run("nil work is rejected and nothing runs", func(t *testing.T) {
		called := false
		if err := leakyStart(nil, func(error) { called = true }); err == nil {
			t.Fatal("leakyStart(nil) returned nil error")
		}
		if called {
			t.Fatal("report called for nil work")
		}
	})

	t.Run("returns before the work finishes and the error bypasses the caller", func(t *testing.T) {
		synctest.Test(t, func(t *testing.T) {
			gate := make(chan struct{})
			var reported atomic.Value

			err := leakyStart(func() error {
				<-gate
				return errors.New("disk full")
			}, func(err error) { reported.Store(err) })
			if err != nil {
				t.Fatalf("leakyStart returned %v, want nil", err)
			}

			// The caller is back while the goroutine is still blocked: nobody owns it.
			synctest.Wait()
			if reported.Load() != nil {
				t.Fatal("error reported before the work finished")
			}

			close(gate)
			synctest.Wait()
			if got, _ := reported.Load().(error); got == nil || got.Error() != "disk full" {
				t.Fatalf("report got %v, want disk full", got)
			}
		})
	})

	t.Run("success is not reported", func(t *testing.T) {
		synctest.Test(t, func(t *testing.T) {
			var called atomic.Bool
			if err := leakyStart(func() error { return nil }, func(error) { called.Store(true) }); err != nil {
				t.Fatal(err)
			}
			synctest.Wait()
			if called.Load() {
				t.Fatal("report called for a nil error")
			}
		})
	})
}

func TestRunAll(t *testing.T) {
	t.Run("no tasks", func(t *testing.T) {
		if err := runAll(); err != nil {
			t.Fatalf("runAll() = %v, want nil", err)
		}
	})

	t.Run("all succeed", func(t *testing.T) {
		if err := runAll(func() error { return nil }, func() error { return nil }); err != nil {
			t.Fatalf("runAll = %v, want nil", err)
		}
	})

	t.Run("errors are joined in task order, not completion order", func(t *testing.T) {
		synctest.Test(t, func(t *testing.T) {
			gates := []chan struct{}{make(chan struct{}), make(chan struct{}), make(chan struct{})}
			names := []string{"a", "b", "c"}
			tasks := make([]func() error, len(gates))
			for i := range gates {
				tasks[i] = func() error {
					<-gates[i]
					return errors.New(names[i])
				}
			}

			done := make(chan error)
			go func() { done <- runAll(tasks...) }()

			// Finish the tasks last to first; runAll must not return until all are done.
			for i, gate := range slices.Backward(gates) {
				synctest.Wait()
				select {
				case <-done:
					t.Fatalf("runAll returned with task %d still running", i)
				default:
				}
				close(gate)
			}

			if err := <-done; err == nil || err.Error() != "a\nb\nc" {
				t.Fatalf("runAll = %q, want a\\nb\\nc", err)
			}
		})
	})

	t.Run("tasks run concurrently", func(t *testing.T) {
		synctest.Test(t, func(t *testing.T) {
			const n = 3
			var started atomic.Int64
			all := make(chan struct{})
			tasks := make([]func() error, n)
			for i := range tasks {
				tasks[i] = func() error {
					if started.Add(1) == n {
						close(all)
					}
					<-all // only passes when every task has started
					return nil
				}
			}
			if err := runAll(tasks...); err != nil {
				t.Fatal(err)
			}
		})
	})
}
