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

package untimedbatch

// Mutants that survive on purpose:
// - buffer-channel at untimedbatch.go:30 and :52: done is only closed, never sent on, so a buffer changes nothing.

import (
	"slices"
	"sync"
	"testing"
	"testing/synctest"
	"time"
)

// record returns a flush function and a way to read the batches it received.
func record() (func([]string), func() [][]string) {
	var (
		mu  sync.Mutex
		got [][]string
	)

	flush := func(batch []string) {
		mu.Lock()
		defer mu.Unlock()

		got = append(got, batch)
	}

	read := func() [][]string {
		mu.Lock()
		defer mu.Unlock()

		return slices.Clone(got)
	}

	return flush, read
}

func check(t *testing.T, got, want [][]string) {
	t.Helper()

	if !slices.EqualFunc(got, want, slices.Equal) {
		t.Fatalf("batches = %v, want %v", got, want)
	}
}

func TestSizeOnly(t *testing.T) {
	t.Run("flushes a full batch", func(t *testing.T) {
		synctest.Test(t, func(t *testing.T) {
			flush, got := record()
			in := make(chan string, 4)
			done := sizeOnly(in, 2, flush)

			in <- "a"
			in <- "b"
			synctest.Wait()
			check(t, got(), [][]string{{"a", "b"}})

			close(in)
			<-done
		})
	})

	t.Run("a partial batch waits forever and is lost on close", func(t *testing.T) {
		synctest.Test(t, func(t *testing.T) {
			flush, got := record()
			in := make(chan string, 4)
			done := sizeOnly(in, 2, flush)

			in <- "a"
			in <- "b"
			in <- "c"

			for range 3 {
				time.Sleep(24 * time.Hour)
				synctest.Wait()
				check(t, got(), [][]string{{"a", "b"}})
			}

			close(in)
			<-done
			check(t, got(), [][]string{{"a", "b"}})
		})
	})
}

func TestSizeOrTime(t *testing.T) {
	t.Run("flushes a partial batch after maxWait", func(t *testing.T) {
		synctest.Test(t, func(t *testing.T) {
			flush, got := record()
			in := make(chan string, 4)
			done := sizeOrTime(in, 2, time.Second, flush)

			in <- "a"
			in <- "b"
			in <- "c"

			time.Sleep(time.Second - time.Nanosecond)
			synctest.Wait()
			check(t, got(), [][]string{{"a", "b"}})

			time.Sleep(time.Nanosecond)
			synctest.Wait()
			check(t, got(), [][]string{{"a", "b"}, {"c"}})

			time.Sleep(time.Hour)
			synctest.Wait()
			check(t, got(), [][]string{{"a", "b"}, {"c"}})

			close(in)
			<-done
		})
	})

	t.Run("flushes the rest on close", func(t *testing.T) {
		synctest.Test(t, func(t *testing.T) {
			flush, got := record()
			in := make(chan string, 4)
			done := sizeOrTime(in, 4, time.Hour, flush)

			in <- "a"
			close(in)
			<-done
			check(t, got(), [][]string{{"a"}})
		})
	})
	t.Run("does not flush an empty batch on close", func(t *testing.T) {
		synctest.Test(t, func(t *testing.T) {
			flush, got := record()
			in := make(chan string)
			done := sizeOrTime(in, 4, time.Hour, flush)

			close(in)
			<-done
			check(t, got(), nil)
		})
	})
}
