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

// Mutants that survive on purpose:
//   - buffer-channel sleepsync.go:37 (done buffered): done is only closed, never sent on; buffering changes nothing.

package sleepsync

import (
	"testing"
	"testing/synctest"
	"time"
)

func TestStartJob(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		gate := make(chan struct{})
		j := startJob(gate)

		synctest.Wait()
		select {
		case <-j.done:
			t.Fatal("job done before the gate opened")
		default:
		}
		if got := j.result.Load(); got != 0 {
			t.Fatalf("result before gate = %d, want 0", got)
		}

		close(gate)
		<-j.done
		if got := j.result.Load(); got != 42 {
			t.Fatalf("result = %d, want 42", got)
		}
	})
}

func TestBadRead(t *testing.T) {
	t.Run("job slower than the guess: reads the zero value", func(t *testing.T) {
		synctest.Test(t, func(t *testing.T) {
			gate := make(chan struct{})
			j := startJob(gate)
			start := time.Now()

			if got := badRead(j); got != 0 {
				t.Fatalf("badRead = %d, want 0 (job not finished)", got)
			}
			if waited := time.Since(start); waited != time.Millisecond {
				t.Fatalf("badRead waited %v, want the guessed 1ms", waited)
			}

			close(gate)
			<-j.done
		})
	})

	t.Run("job faster than the guess: still waits the full guess", func(t *testing.T) {
		synctest.Test(t, func(t *testing.T) {
			gate := make(chan struct{})
			close(gate)
			j := startJob(gate)
			<-j.done
			start := time.Now()

			if got := badRead(j); got != 42 {
				t.Fatalf("badRead = %d, want 42", got)
			}
			if waited := time.Since(start); waited != time.Millisecond {
				t.Fatalf("badRead waited %v, want 1ms", waited)
			}
		})
	})
}

func TestRead(t *testing.T) {
	t.Run("waits exactly as long as the job", func(t *testing.T) {
		synctest.Test(t, func(t *testing.T) {
			gate := make(chan struct{})
			j := startJob(gate)
			got := make(chan int64, 1)
			go func() { got <- read(j) }()

			time.Sleep(time.Hour) // far longer than any guess
			synctest.Wait()
			select {
			case v := <-got:
				t.Fatalf("read returned %d before the job finished", v)
			default:
			}

			close(gate)
			if v := <-got; v != 42 {
				t.Fatalf("read = %d, want 42", v)
			}
		})
	})

	t.Run("finished job returns without waiting", func(t *testing.T) {
		synctest.Test(t, func(t *testing.T) {
			gate := make(chan struct{})
			close(gate)
			j := startJob(gate)
			synctest.Wait()
			start := time.Now()
			if v := read(j); v != 42 {
				t.Fatalf("read = %d, want 42", v)
			}
			if d := time.Since(start); d != 0 {
				t.Fatalf("read took %v", d)
			}
		})
	})
}
