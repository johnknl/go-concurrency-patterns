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
//   - mutate buffer-channel snapshot.go:40, 41, 42 (snapshots, quit, done): the requester always waits for the
//     reply, and quit and done are only closed, so buffering changes nothing.
//   - mutate unbuffer-channel snapshot.go:71: snapshot always receives its reply, so the owner never blocks on it.

package snapshot

import (
	"maps"
	"sync"
	"testing"
	"testing/synctest"
)

func stop(s *stats) {
	close(s.quit)
	<-s.done
}

func TestStats(t *testing.T) {
	t.Run("empty snapshot", func(t *testing.T) {
		synctest.Test(t, func(t *testing.T) {
			s := newStats()
			defer stop(s)

			if got := s.snapshot(); len(got) != 0 {
				t.Fatalf("got %v, want empty", got)
			}
		})
	})

	t.Run("counts concurrent records", func(t *testing.T) {
		synctest.Test(t, func(t *testing.T) {
			s := newStats()
			defer stop(s)

			var wg sync.WaitGroup
			for range 10 {
				wg.Go(func() {
					s.record("a")
					s.record("b")
				})
			}

			wg.Go(func() { s.record("c") })
			wg.Wait()

			want := map[string]int{"a": 10, "b": 10, "c": 1}
			if got := s.snapshot(); !maps.Equal(got, want) {
				t.Fatalf("got %v, want %v", got, want)
			}
		})
	})

	t.Run("a record is counted before record returns", func(t *testing.T) {
		s := newStats()
		defer stop(s)

		for want := 1; want <= 1000; want++ {
			s.record("a")

			if got := s.snapshot()["a"]; got != want {
				t.Fatalf("got %d, want %d", got, want)
			}
		}
	})

	t.Run("record waits until the owner takes the event", func(t *testing.T) {
		synctest.Test(t, func(t *testing.T) {
			s := newStats()
			defer stop(s)

			busy := make(chan map[string]int) // unbuffered: the owner blocks replying until we read
			s.snapshots <- busy

			recorded := make(chan struct{})
			go func() {
				defer close(recorded)
				s.record("a")
			}()

			synctest.Wait()

			select {
			case <-recorded:
				t.Fatal("record returned while the owner was busy")
			default:
			}

			<-busy
			<-recorded
		})
	})

	t.Run("a snapshot is a copy", func(t *testing.T) {
		synctest.Test(t, func(t *testing.T) {
			s := newStats()
			defer stop(s)

			s.record("a")

			first := s.snapshot()
			first["a"] = 100
			first["z"] = 1

			s.record("a")

			want := map[string]int{"a": 2}
			if got := s.snapshot(); !maps.Equal(got, want) {
				t.Fatalf("got %v, want %v", got, want)
			}
		})
	})

	t.Run("the owner does not wait for a requester that left", func(t *testing.T) {
		synctest.Test(t, func(_ *testing.T) {
			s := newStats()

			s.snapshots <- make(chan map[string]int, 1) // nobody reads the reply

			stop(s)
		})
	})

	t.Run("quit closes done", func(t *testing.T) {
		synctest.Test(t, func(t *testing.T) {
			s := newStats()
			synctest.Wait()

			select {
			case <-s.done:
				t.Fatal("done closed before quit")
			default:
			}

			stop(s)
		})
	})
}
