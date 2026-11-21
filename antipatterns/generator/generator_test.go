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

package generator

// Mutants that survive on purpose:

import (
	"context"
	"testing"
	"testing/synctest"
)

func TestNaturals(t *testing.T) {
	t.Run("counts up from zero", func(t *testing.T) {
		synctest.Test(t, func(t *testing.T) {
			ctx, cancel := context.WithCancel(t.Context())
			defer cancel()

			out := naturals(ctx)
			for want := range 100 {
				if got := <-out; got != want {
					t.Fatalf("got %d, want %d", got, want)
				}
			}
		})
	})

	t.Run("runs no further ahead than the consumer", func(t *testing.T) {
		synctest.Test(t, func(t *testing.T) {
			ctx, cancel := context.WithCancel(t.Context())
			defer cancel()

			out := naturals(ctx)
			synctest.Wait()

			if n := len(out); n != 0 {
				t.Fatalf("%d values queued before anyone read, want 0", n)
			}

			if got := <-out; got != 0 {
				t.Fatalf("first value %d, want 0", got)
			}
		})
	})

	t.Run("cancel before reading closes the channel", func(t *testing.T) {
		synctest.Test(t, func(t *testing.T) {
			ctx, cancel := context.WithCancel(t.Context())
			out := naturals(ctx)

			cancel()
			synctest.Wait() // the generator is blocked on a send nobody takes, so it must pick ctx.Done

			if v, ok := <-out; ok {
				t.Fatalf("received %d after cancel, want a closed channel", v)
			}
		})
	})

	t.Run("cancel after reading stops the goroutine", func(t *testing.T) {
		synctest.Test(t, func(t *testing.T) {
			ctx, cancel := context.WithCancel(t.Context())
			out := naturals(ctx)

			<-out
			<-out
			cancel()
			synctest.Wait()

			if v, ok := <-out; ok {
				t.Fatalf("received %d after cancel, want a closed channel", v)
			}
		})
	})
}

func TestNaturalsSeq(t *testing.T) {
	t.Run("counts up from zero", func(t *testing.T) {
		want := 0
		for got := range naturalsSeq() {
			if got != want {
				t.Fatalf("got %d, want %d", got, want)
			}

			want++
			if want == 100 {
				break
			}
		}
	})

	t.Run("stops when the consumer breaks", func(_ *testing.T) {
		// A sequence that kept calling yield after break would panic here.
		for v := range naturalsSeq() {
			if v == 0 {
				break
			}
		}
	})
}
