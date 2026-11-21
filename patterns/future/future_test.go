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

package future

import (
	"errors"
	"testing"
	"testing/synctest"
)

func TestSquare(t *testing.T) {
	for _, n := range []int{0, 1, 12} {
		if v, err := square(n); err != nil || v != n*n {
			t.Fatalf("square(%d) = %d, %v; want %d, nil", n, v, err, n*n)
		}
	}

	if _, err := square(-1); !errors.Is(err, errNegative) {
		t.Fatalf("square(-1) error = %v, want %v", err, errNegative)
	}
}

func TestStart(t *testing.T) {
	t.Run("delivers the value and the error", func(t *testing.T) {
		synctest.Test(t, func(t *testing.T) {
			if r := <-start(func() (int, error) { return square(5) }); r.err != nil || r.val != 25 {
				t.Fatalf("got %+v, want 25", r)
			}

			if r := <-start(func() (int, error) { return square(-5) }); !errors.Is(r.err, errNegative) {
				t.Fatalf("got %+v, want %v", r, errNegative)
			}
		})
	})

	t.Run("runs fn concurrently with the caller", func(t *testing.T) {
		synctest.Test(t, func(t *testing.T) {
			release := make(chan struct{})
			ch := start(func() (int, error) {
				<-release
				return 1, nil
			})

			// start returned while fn is still blocked.
			close(release)

			if r := <-ch; r.val != 1 {
				t.Fatalf("got %+v", r)
			}
		})
	})

	t.Run("the goroutine finishes when nobody receives", func(t *testing.T) {
		synctest.Test(t, func(t *testing.T) {
			ch := start(func() (int, error) { return square(3) })
			synctest.Wait()

			if len(ch) != 1 {
				t.Fatalf("result not buffered: len %d", len(ch))
			}
			// synctest fails the test if the goroutine is still blocked on its send.
		})
	})
}
