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

package closecause

import (
	"errors"
	"fmt"
	"slices"
	"sync"
	"testing"
)

func TestStream_finish(t *testing.T) {
	errReset := errors.New("connection reset")

	t.Run("nil cause is a normal end", func(t *testing.T) {
		s := newStream()
		s.finish(nil)

		if _, ok := <-s.C; ok {
			t.Fatal("C delivered a value, want it closed")
		}

		if err := s.Err(); err != nil {
			t.Fatalf("Err() = %v, want nil", err)
		}
	})

	t.Run("the cause is visible once C is closed", func(t *testing.T) {
		s := newStream()

		go s.finish(errReset)

		for range s.C {
		}

		if err := s.Err(); !errors.Is(err, errReset) {
			t.Fatalf("Err() = %v, want %v", err, errReset)
		}
	})

	t.Run("only the first call has an effect", func(t *testing.T) {
		s := newStream()
		s.finish(errReset)
		s.finish(nil)
		s.finish(errors.New("later"))

		if err := s.Err(); !errors.Is(err, errReset) {
			t.Fatalf("Err() = %v, want %v", err, errReset)
		}
	})

	t.Run("concurrent calls close once and keep one cause", func(t *testing.T) {
		s := newStream()

		causes := make([]error, 8)
		for i := range causes {
			causes[i] = fmt.Errorf("cause %d", i)
		}

		var wg sync.WaitGroup
		for _, cause := range causes {
			wg.Go(func() { s.finish(cause) })
		}

		wg.Go(func() { _ = s.Err() })
		wg.Wait()

		if !slices.Contains(causes, s.Err()) {
			t.Fatalf("Err() = %v, want one of the causes", s.Err())
		}
	})
}

func TestStream_Err(t *testing.T) {
	s := newStream()
	s.C <- 1

	if err := s.Err(); err != nil {
		t.Fatalf("Err() before finish = %v, want nil", err)
	}
}

func TestNewStream(t *testing.T) {
	s := newStream()

	select {
	case s.C <- 1:
	default:
		t.Fatal("the first send blocked, want room for one value")
	}

	select {
	case s.C <- 2:
		t.Fatal("the second send succeeded, want the producer held back after one value")
	default:
	}
}

func TestDrain(t *testing.T) {
	t.Run("no values", func(t *testing.T) {
		s := newStream()
		s.finish(nil)

		if got, want := drain("s", s), "s 0 <nil>"; got != want {
			t.Fatalf("got %q, want %q", got, want)
		}
	})

	t.Run("counts values sent before and after it starts", func(t *testing.T) {
		s := newStream()
		s.C <- 1

		go func() {
			s.C <- 2
			s.C <- 3
			s.finish(errors.New("dropped"))
		}()

		if got, want := drain("s", s), "s 3 dropped"; got != want {
			t.Fatalf("got %q, want %q", got, want)
		}
	})
}
