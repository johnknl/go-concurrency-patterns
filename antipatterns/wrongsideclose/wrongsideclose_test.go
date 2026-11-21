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
//   - buffer-channel wrongsideclose.go:42 (badCloseTwice channel buffered): closing twice panics either way.
//   - unbuffer-channel wrongsideclose.go:33 (badSendAfterClose channel unbuffered): a send on a closed channel
//     panics either way.
//   - gremlins reports ARITHMETIC_BASE at wrongsideclose.go:58 as not covered although the line runs (100% statement
//     coverage) and TestProduce checks the exact values; gremlins does not see coverage inside the wg.Go closure.

package wrongsideclose

import (
	"runtime"
	"slices"
	"testing"
	"testing/synctest"
)

func TestBadSendAfterClose(t *testing.T) {
	err, ok := badSendAfterClose().(runtime.Error)
	if !ok || err.Error() != "send on closed channel" {
		t.Fatalf("recovered %v, want send on closed channel", err)
	}
}

func TestBadCloseTwice(t *testing.T) {
	err, ok := badCloseTwice().(runtime.Error)
	if !ok || err.Error() != "close of closed channel" {
		t.Fatalf("recovered %v, want close of closed channel", err)
	}
}

func TestProduce(t *testing.T) {
	t.Run("no senders closes out", func(t *testing.T) {
		synctest.Test(t, func(t *testing.T) {
			if _, ok := <-produce(0, nil); ok {
				t.Fatal("received a value from zero senders")
			}
		})
	})

	t.Run("delivers every value once, then closes", func(t *testing.T) {
		synctest.Test(t, func(t *testing.T) {
			var got []int
			for v := range produce(3, nil) {
				got = append(got, v)
			}
			slices.Sort(got)
			if want := []int{0, 1, 10, 11, 20, 21}; !slices.Equal(got, want) {
				t.Fatalf("values = %v, want %v", got, want)
			}
		})
	})

	t.Run("senders wait for the receiver", func(t *testing.T) {
		synctest.Test(t, func(t *testing.T) {
			done := make(chan struct{})
			out := produce(2, done)
			synctest.Wait()
			// Nothing is lost while nobody receives: the first value is still there.
			if _, ok := <-out; !ok {
				t.Fatal("out closed before any value was received")
			}
			close(done)
			for range out {
			}
		})
	})

	t.Run("a sender blocked on an absent receiver leaves when done closes", func(t *testing.T) {
		synctest.Test(t, func(t *testing.T) {
			done := make(chan struct{})
			out := produce(1, done)
			synctest.Wait() // the sender is parked on its first send

			close(done)
			synctest.Wait()
			if v, ok := <-out; ok {
				t.Fatalf("received %d: the value was handed off without a receiver", v)
			}
		})
	})

	t.Run("receiver stops by closing done, never out", func(t *testing.T) {
		synctest.Test(t, func(t *testing.T) {
			done := make(chan struct{})
			out := produce(3, done)
			close(done)
			synctest.Wait()

			// Every sender saw done and returned, so the closer has closed out.
			if v, ok := <-out; ok {
				t.Fatalf("received %d after done was closed and senders settled", v)
			}
		})
	})
}
