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

// Mutants that survive on purpose: none.

package objectcache

import (
	"bytes"
	"sync"
	"testing"
)

func TestRender(t *testing.T) {
	t.Run("formats the greeting", func(t *testing.T) {
		if got := render("gopher"); got != "hello, gopher" {
			t.Fatalf("got %q", got)
		}
	})

	t.Run("a reused buffer holds no old content", func(t *testing.T) {
		for range 100 {
			if got := render("a"); got != "hello, a" {
				t.Fatalf("got %q, want %q", got, "hello, a")
			}
		}

		buf := buffers.Get().(*bytes.Buffer) //nolint:errcheck // New only creates buffers
		defer buffers.Put(buf)

		if buf.Len() != 0 {
			t.Fatalf("buffer from the pool holds %q", buf.String())
		}
	})

	t.Run("concurrent callers", func(t *testing.T) {
		var wg sync.WaitGroup
		for range 50 {
			wg.Go(func() {
				if got := render("x"); got != "hello, x" {
					t.Errorf("got %q", got)
				}
			})
		}

		wg.Wait()
	})
}
