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

package groupcommit

// Mutants that survive on purpose:
// - none.

import (
	"errors"
	"slices"
	"strings"
	"sync"
	"testing"
	"testing/synctest"
)

var errBad = errors.New("bad item")

// gatedFlush records every batch and holds each flush until the test sends on release. It fails a batch that
// contains "bad", and reports a flush that runs while another one is running.
type gatedFlush struct {
	t       *testing.T
	release chan struct{}
	batches [][]string
	active  int
	mu      sync.Mutex
}

func newGatedFlush(t *testing.T) *gatedFlush {
	return &gatedFlush{t: t, release: make(chan struct{})}
}

func (g *gatedFlush) flush(items []string) error {
	g.mu.Lock()
	g.active++
	if g.active > 1 {
		g.t.Error("flush called concurrently")
	}
	g.batches = append(g.batches, slices.Sorted(slices.Values(items)))
	g.mu.Unlock()

	<-g.release

	g.mu.Lock()
	g.active--
	g.mu.Unlock()

	if slices.Contains(items, "bad") {
		return errBad
	}

	return nil
}

func (g *gatedFlush) got() string {
	g.mu.Lock()
	defer g.mu.Unlock()

	parts := make([]string, len(g.batches))
	for i, b := range g.batches {
		parts[i] = "[" + strings.Join(b, " ") + "]"
	}

	return strings.Join(parts, " ")
}

// start commits item in a new goroutine, waits until it is blocked, and returns a channel with its result.
func start(c *committer, item string) <-chan error {
	res := make(chan error, 1)

	go func() { res <- c.commit(item) }()

	synctest.Wait()

	return res
}

func returned(res <-chan error) bool {
	return len(res) == 1
}

func TestIdleFlushesAlone(t *testing.T) {
	var batches [][]string

	c := newCommitter(8, func(items []string) error {
		batches = append(batches, slices.Clone(items))
		return nil
	})

	for _, item := range []string{"a", "b"} {
		if err := c.commit(item); err != nil {
			t.Fatal(err)
		}
	}

	if want := [][]string{{"a"}, {"b"}}; !slices.EqualFunc(batches, want, slices.Equal) {
		t.Fatalf("batches = %v, want %v", batches, want)
	}
}

func TestCallersDuringFlushFormOneBatch(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		g := newGatedFlush(t)
		c := newCommitter(8, g.flush)

		a := start(c, "a")
		b := start(c, "b")
		cc := start(c, "c")
		d := start(c, "d")

		if got := g.got(); got != "[a]" {
			t.Fatalf("while a is flushed: batches = %s", got)
		}

		g.release <- struct{}{}
		synctest.Wait()

		// The leader of the first batch has returned; one of b, c, d leads the second.
		if !returned(a) || returned(b) || returned(cc) || returned(d) {
			t.Fatal("a must return after its own batch, b, c and d must still wait")
		}

		if got := g.got(); got != "[a] [b c d]" {
			t.Fatalf("batches = %s, want [a] [b c d]", got)
		}

		g.release <- struct{}{}

		for _, res := range []<-chan error{a, b, cc, d} {
			if err := <-res; err != nil {
				t.Fatal(err)
			}
		}
	})
}

func TestErrorReachesOnlyItsBatch(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		g := newGatedFlush(t)
		c := newCommitter(8, g.flush)

		a := start(c, "a")
		bad := start(c, "bad")
		b := start(c, "b")

		g.release <- struct{}{}
		synctest.Wait()
		g.release <- struct{}{}

		if err := <-a; err != nil {
			t.Fatalf("a: %v, want nil", err)
		}

		for name, res := range map[string]<-chan error{"bad": bad, "b": b} {
			if err := <-res; !errors.Is(err, errBad) {
				t.Fatalf("%s: %v, want %v", name, err, errBad)
			}
		}

		after := start(c, "after")
		g.release <- struct{}{}

		if err := <-after; err != nil {
			t.Fatalf("after a failed batch: %v, want nil", err)
		}

		if got := g.got(); got != "[a] [b bad] [after]" {
			t.Fatalf("batches = %s", got)
		}
	})
}

func TestBatchSizeIsBounded(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		g := newGatedFlush(t)
		c := newCommitter(2, g.flush)

		a := start(c, "a")
		b := start(c, "b")
		cc := start(c, "c")
		d := start(c, "d") // the open batch [b c] is full: d waits for room

		g.release <- struct{}{}
		synctest.Wait()

		if got := g.got(); got != "[a] [b c]" {
			t.Fatalf("batches = %s, want [a] [b c]", got)
		}

		g.release <- struct{}{}
		synctest.Wait()
		g.release <- struct{}{}

		for _, res := range []<-chan error{a, b, cc, d} {
			if err := <-res; err != nil {
				t.Fatal(err)
			}
		}

		if got := g.got(); got != "[a] [b c] [d]" {
			t.Fatalf("batches = %s, want [a] [b c] [d]", got)
		}
	})
}
