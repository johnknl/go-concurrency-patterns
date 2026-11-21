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

package batcher

// Mutants that survive on purpose:
// - buffer-channel at batcher.go:31: done is only closed, never sent on, so a buffer changes nothing.

import (
	"slices"
	"sync"
	"testing"
	"testing/synctest"
	"time"
)

type flushAt struct {
	at    time.Duration
	batch []string
}

// recorder starts a batcher whose flushes are recorded with the fake time since start.
func recorder(maxSize int, maxWait time.Duration) (chan<- string, <-chan struct{}, func() []flushAt) {
	start := time.Now()
	in := make(chan string, 4)

	var (
		mu  sync.Mutex
		got []flushAt
	)

	done := batcher(in, maxSize, maxWait, func(batch []string) {
		mu.Lock()
		defer mu.Unlock()

		got = append(got, flushAt{time.Since(start), batch})
	})

	flushes := func() []flushAt {
		mu.Lock()
		defer mu.Unlock()

		return slices.Clone(got)
	}

	return in, done, flushes
}

func check(t *testing.T, got, want []flushAt) {
	t.Helper()

	eq := func(a, b flushAt) bool { return a.at == b.at && slices.Equal(a.batch, b.batch) }
	if !slices.EqualFunc(got, want, eq) {
		t.Fatalf("flushes = %v, want %v", got, want)
	}
}

func TestFlushAtSize(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		in, done, got := recorder(2, time.Second)

		in <- "a"
		in <- "b"
		in <- "c"
		synctest.Wait()
		check(t, got(), []flushAt{{0, []string{"a", "b"}}})

		in <- "d"
		synctest.Wait()
		check(t, got(), []flushAt{{0, []string{"a", "b"}}, {0, []string{"c", "d"}}})

		close(in)
		<-done
	})
}

func TestFlushAtTime(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		in, done, got := recorder(10, time.Second)

		in <- "a"
		time.Sleep(400 * time.Millisecond)
		in <- "b" // joins the batch; the deadline stays at 1s after a

		time.Sleep(600*time.Millisecond - time.Nanosecond)
		synctest.Wait()
		check(t, got(), nil)

		time.Sleep(time.Nanosecond)
		synctest.Wait()
		check(t, got(), []flushAt{{time.Second, []string{"a", "b"}}})

		// The next batch gets a fresh deadline from its own first item.
		time.Sleep(time.Second)
		in <- "c"
		time.Sleep(time.Second)
		synctest.Wait()
		check(t, got(), []flushAt{{time.Second, []string{"a", "b"}}, {3 * time.Second, []string{"c"}}})

		close(in)
		<-done
	})
}

func TestSizeFlushStopsTimer(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		in, done, got := recorder(2, time.Second)

		in <- "a"
		in <- "b"
		time.Sleep(500 * time.Millisecond)
		in <- "c" // a timer left over from [a b] would flush [c] at 1s

		time.Sleep(time.Second - time.Nanosecond)
		synctest.Wait()
		check(t, got(), []flushAt{{0, []string{"a", "b"}}})

		time.Sleep(time.Nanosecond)
		synctest.Wait()
		check(t, got(), []flushAt{{0, []string{"a", "b"}}, {1500 * time.Millisecond, []string{"c"}}})

		close(in)
		<-done
	})
}

func TestNoFlushForEmptyBatch(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		in, done, got := recorder(2, time.Second)

		time.Sleep(time.Hour)
		synctest.Wait()
		check(t, got(), nil)

		close(in)
		<-done
		check(t, got(), nil)
	})
}

func TestFlushOnClose(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		in, done, got := recorder(10, time.Hour)

		in <- "a"
		in <- "b"
		time.Sleep(time.Minute)
		close(in)
		<-done // the batcher has ended, so no goroutine is left in the bubble

		check(t, got(), []flushAt{{time.Minute, []string{"a", "b"}}})
	})
}

func TestFullInputBlocksSender(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		in := make(chan string, 1)
		release := make(chan struct{})
		done := batcher(in, 1, time.Second, func([]string) { <-release })

		in <- "a" // taken by the batcher, which is held in flush
		synctest.Wait()
		in <- "b" // fills the buffer

		sent := make(chan struct{})
		go func() {
			in <- "c"
			close(sent)
		}()

		synctest.Wait()

		select {
		case <-sent:
			t.Fatal("send on a full input did not wait")
		default:
		}

		release <- struct{}{}
		release <- struct{}{}
		<-sent
		release <- struct{}{}
		close(in)
		<-done
	})
}
