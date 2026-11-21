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

package forselect

import (
	"context"
	"slices"
	"testing"
	"testing/synctest"
)

func TestSend(t *testing.T) {
	t.Run("sends every item in order", func(t *testing.T) {
		synctest.Test(t, func(t *testing.T) {
			out := make(chan string)

			go func() {
				defer close(out)
				send(context.Background(), out, []string{"a", "b", "c"})
			}()

			var got []string
			for s := range out {
				got = append(got, s)
			}

			if want := []string{"a", "b", "c"}; !slices.Equal(got, want) {
				t.Fatalf("got %v, want %v", got, want)
			}
		})
	})

	t.Run("no items", func(_ *testing.T) {
		send(context.Background(), nil, nil)
	})

	t.Run("canceling stops it while it waits for a reader", func(t *testing.T) {
		synctest.Test(t, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			out := make(chan string)
			returned := make(chan struct{})

			go func() {
				defer close(returned)
				send(ctx, out, []string{"a", "b", "c"})
			}()

			if got := <-out; got != "a" {
				t.Fatalf("got %q, want %q", got, "a")
			}

			synctest.Wait()

			select {
			case <-returned:
				t.Fatal("send returned with items left and the context live")
			default:
			}

			cancel()
			<-returned
		})
	})

	t.Run("an already canceled context sends nothing", func(_ *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		send(ctx, make(chan string), []string{"a"})
	})
}

func TestWait(t *testing.T) {
	t.Run("sums what arrives until the context is canceled", func(t *testing.T) {
		synctest.Test(t, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			in := make(chan int)
			sum := make(chan int, 1)

			go func() { sum <- wait(ctx, in) }()

			for n := 1; n <= 4; n++ {
				in <- n
			}

			synctest.Wait()

			if len(sum) != 0 {
				t.Fatal("wait returned before the context was canceled")
			}

			cancel()

			if got := <-sum; got != 10 {
				t.Fatalf("got %d, want 10", got)
			}
		})
	})

	t.Run("nothing arrives", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		if got := wait(ctx, nil); got != 0 {
			t.Fatalf("got %d, want 0", got)
		}
	})
}

func TestWork(t *testing.T) {
	t.Run("works until the context is canceled", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		started := make(chan struct{})
		steps := make(chan int)

		go func() { steps <- work(ctx, started) }()

		<-started
		cancel()

		if got := <-steps; got < 1 {
			t.Fatalf("got %d steps, want at least 1", got)
		}
	})

	t.Run("an already canceled context takes no step", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		started := make(chan struct{})

		if got := work(ctx, started); got != 0 {
			t.Fatalf("got %d steps, want 0", got)
		}

		select {
		case <-started:
			t.Fatal("started was closed without a step")
		default:
		}
	})
}
