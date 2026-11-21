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

package pubsub

import (
	"errors"
	"slices"
	"sync"
	"testing"
	"testing/synctest"
)

func received(s *sub) []int {
	var got []int
	for msg := range s.ch {
		got = append(got, msg)
	}

	return got
}

func TestHub(t *testing.T) {
	t.Run("publish with no subscribers", func(t *testing.T) {
		h := newHub()
		h.publish(1)

		if h.dropped.Load() != 0 {
			t.Fatal("dropped a message nobody wanted")
		}
	})

	t.Run("every subscriber gets each message from when it subscribed", func(t *testing.T) {
		h := newHub()
		a := h.subscribe(5)
		h.publish(1)
		b := h.subscribe(5)
		h.publish(2)
		h.unsubscribe(a)
		h.publish(3)
		h.close()

		if got := received(a); !slices.Equal(got, []int{1, 2}) || !errors.Is(a.err, errUnsubscribed) {
			t.Fatalf("a got %v, %v", got, a.err)
		}

		if got := received(b); !slices.Equal(got, []int{2, 3}) || !errors.Is(b.err, errClosed) {
			t.Fatalf("b got %v, %v", got, b.err)
		}
	})

	t.Run("a full subscriber loses messages", func(t *testing.T) {
		h := newHub()
		slow := h.subscribe(1)

		for m := range 4 {
			h.publish(m)
		}

		h.close()

		if got := received(slow); !slices.Equal(got, []int{0}) || h.dropped.Load() != 3 {
			t.Fatalf("got %v, dropped %d", got, h.dropped.Load())
		}
	})

	t.Run("unsubscribe twice and after close is harmless", func(t *testing.T) {
		h := newHub()
		s := h.subscribe(1)
		h.unsubscribe(s)
		h.unsubscribe(s)

		t2 := h.subscribe(1)
		h.close()
		h.unsubscribe(t2)

		if !errors.Is(s.err, errUnsubscribed) || !errors.Is(t2.err, errClosed) {
			t.Fatalf("got %v and %v", s.err, t2.err)
		}
	})

	t.Run("subscribe after close gets a closed channel", func(t *testing.T) {
		h := newHub()
		h.close()

		s := h.subscribe(1)
		h.publish(1)

		if got := received(s); got != nil || !errors.Is(s.err, errClosed) {
			t.Fatalf("got %v, %v", got, s.err)
		}
	})

	t.Run("publish never blocks on a subscriber nobody reads", func(t *testing.T) {
		synctest.Test(t, func(t *testing.T) {
			h := newHub()
			s := h.subscribe(0)

			done := make(chan struct{})
			go func() {
				defer close(done)
				h.publish(1)
			}()

			synctest.Wait()

			select {
			case <-done:
			default:
				t.Fatal("publish blocked")
			}

			h.close()

			if received(s) != nil {
				t.Fatal("unexpected message")
			}
		})
	})

	t.Run("concurrent publishers and unsubscribers", func(t *testing.T) {
		h := newHub()

		subs := make([]*sub, 10)
		for i := range subs {
			subs[i] = h.subscribe(100)
		}

		var wg sync.WaitGroup
		for i := range 10 {
			wg.Go(func() { h.publish(i) })
		}

		for _, s := range subs[:5] {
			wg.Go(func() { h.unsubscribe(s) })
		}

		wg.Wait()
		h.close()

		for _, s := range subs[5:] {
			if got := received(s); len(got) != 10 {
				t.Fatalf("got %v, want all 10", got)
			}
		}

		for _, s := range subs[:5] {
			if !errors.Is(s.err, errUnsubscribed) {
				t.Fatalf("err %v", s.err)
			}

			_ = received(s)
		}
	})

	t.Run("concurrent subscribe, publish and close", func(t *testing.T) {
		h := newHub()

		var (
			wg   sync.WaitGroup
			mu   sync.Mutex
			subs []*sub
		)

		for i := range 20 {
			wg.Go(func() {
				s := h.subscribe(100)

				mu.Lock()
				subs = append(subs, s)
				mu.Unlock()
			})
			wg.Go(func() { h.publish(i) })
		}

		wg.Go(h.close)
		wg.Wait()

		for _, s := range subs {
			_ = received(s)

			if !errors.Is(s.err, errClosed) {
				t.Fatalf("err %v, want %v", s.err, errClosed)
			}
		}
	})

	t.Run("drain", func(t *testing.T) {
		s := &sub{ch: make(chan int, 1), err: errClosed}
		s.ch <- 4
		close(s.ch)

		if got := drain("x", s); got != "x [4] hub closed" {
			t.Fatalf("got %q", got)
		}
	})
}
