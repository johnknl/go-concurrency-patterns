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
//   - mutate buffer-channel onlineattach.go:56 (done): done is only closed.

package onlineattach

import (
	"context"
	"errors"
	"slices"
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
	t.Run("a new subscriber gets the last two messages, then live ones", func(t *testing.T) {
		for _, tt := range []struct {
			before []int
			want   []int
		}{
			{nil, []int{9}},
			{[]int{1}, []int{1, 9}},
			{[]int{1, 2}, []int{1, 2, 9}},
			{[]int{1, 2, 3}, []int{2, 3, 9}},
		} {
			synctest.Test(t, func(t *testing.T) {
				h := newHub(t.Context())
				for _, m := range tt.before {
					h.publish(m)
				}

				s := h.subscribe(10)
				h.publish(9)
				h.unsubscribe(s)

				if got := received(s); !slices.Equal(got, tt.want) {
					t.Fatalf("got %v, want %v", got, tt.want)
				}

				if !errors.Is(s.err, errDetached) {
					t.Fatalf("err %v, want %v", s.err, errDetached)
				}
			})
		}
	})

	t.Run("every subscriber gets each message", func(t *testing.T) {
		synctest.Test(t, func(t *testing.T) {
			h := newHub(t.Context())
			a, b := h.subscribe(5), h.subscribe(5)

			h.publish(1)
			h.publish(2)
			h.unsubscribe(a)
			h.publish(3)
			h.unsubscribe(b)

			if got := received(a); !slices.Equal(got, []int{1, 2}) {
				t.Fatalf("a got %v", got)
			}

			if got := received(b); !slices.Equal(got, []int{1, 2, 3}) {
				t.Fatalf("b got %v", got)
			}
		})
	})

	t.Run("a full subscriber loses messages without blocking the hub", func(t *testing.T) {
		synctest.Test(t, func(t *testing.T) {
			ctx, cancel := context.WithCancel(t.Context())
			h := newHub(ctx)
			slow := h.subscribe(1)
			fast := h.subscribe(10)

			for m := range 4 {
				h.publish(m)
			}

			cancel()
			<-h.done

			if got := received(slow); !slices.Equal(got, []int{0}) {
				t.Fatalf("slow got %v", got)
			}

			if got := received(fast); !slices.Equal(got, []int{0, 1, 2, 3}) {
				t.Fatalf("fast got %v", got)
			}

			if h.dropped != 3 {
				t.Fatalf("dropped %d, want 3", h.dropped)
			}
		})
	})

	t.Run("history that does not fit is dropped too", func(t *testing.T) {
		synctest.Test(t, func(t *testing.T) {
			ctx, cancel := context.WithCancel(t.Context())
			h := newHub(ctx)
			h.publish(1)
			h.publish(2)

			s := h.subscribe(1)

			cancel()
			<-h.done

			if got := received(s); !slices.Equal(got, []int{1}) || h.dropped != 1 {
				t.Fatalf("got %v, dropped %d", got, h.dropped)
			}
		})
	})

	t.Run("no message published after unsubscribe returns is delivered", func(t *testing.T) {
		for range 1000 {
			ctx, cancel := context.WithCancel(t.Context())
			h := newHub(ctx) // a fresh hub, so there is no history
			s := h.subscribe(1)
			h.unsubscribe(s)
			h.publish(1)
			cancel()
			<-h.done

			if got := received(s); got != nil {
				t.Fatalf("got %v after unsubscribe", got)
			}
		}
	})

	t.Run("unsubscribe twice is harmless", func(t *testing.T) {
		synctest.Test(t, func(t *testing.T) {
			h := newHub(t.Context())
			s := h.subscribe(1)
			h.unsubscribe(s)
			h.unsubscribe(s)
			h.unsubscribe(&sub{ch: make(chan int)})

			if !errors.Is(s.err, errDetached) {
				t.Fatalf("err %v", s.err)
			}
		})
	})

	t.Run("cancel closes every subscriber with the cause", func(t *testing.T) {
		synctest.Test(t, func(t *testing.T) {
			cause := errors.New("shutdown")
			ctx, cancel := context.WithCancelCause(t.Context())
			h := newHub(ctx)
			a, b := h.subscribe(1), h.subscribe(1)

			cancel(cause)

			for _, s := range []*sub{a, b} {
				if got := received(s); got != nil || !errors.Is(s.err, cause) {
					t.Fatalf("got %v, %v", got, s.err)
				}
			}
		})
	})

	t.Run("after stop nothing blocks and new subscribers are closed", func(t *testing.T) {
		synctest.Test(t, func(t *testing.T) {
			ctx, cancel := context.WithCancel(t.Context())
			h := newHub(ctx)
			cancel()
			<-h.done

			h.publish(1)

			s := h.subscribe(1)
			h.unsubscribe(s)

			if got := received(s); got != nil || !errors.Is(s.err, errStopped) {
				t.Fatalf("got %v, %v", got, s.err)
			}
		})
	})

	t.Run("drain", func(t *testing.T) {
		s := &sub{ch: make(chan int, 2), err: errDetached}
		s.ch <- 1
		s.ch <- 2
		close(s.ch)

		if got := drain("x", s); got != "x [1 2] detached" {
			t.Fatalf("got %q", got)
		}
	})
}
