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

// Mutants that survive on purpose:
//   - remove-critical-section in every method: the examples and tests call the methods from one goroutine, to show
//     an impossible flag combination deterministically; the locks only matter against data races.

package implicitstate

import (
	"errors"
	"testing"
)

func TestBadServerReachesImpossibleState(t *testing.T) {
	s := &badServer{}
	if err := s.Start(func() { s.Stop() }); err != nil {
		t.Fatal(err)
	}
	if !s.started || !s.closed {
		t.Fatalf("started=%v closed=%v, want both true", s.started, s.closed)
	}
}

func TestBadServerNormalLifecycle(t *testing.T) {
	s := &badServer{}
	if s.Running() {
		t.Fatal("running before start")
	}
	if err := s.Start(func() {}); err != nil {
		t.Fatal(err)
	}
	if !s.Running() {
		t.Fatal("not running after start")
	}
	if s.Start(func() {}) == nil {
		t.Fatal("second start succeeded")
	}
	s.Stop()
	s.Stop()
	if s.Running() || s.stopping || !s.closed {
		t.Fatal("still running after stop")
	}
}

func TestServerLifecycle(t *testing.T) {
	s := &server{}
	if s.State() != idle {
		t.Fatal("not idle")
	}
	if err := s.Start(func() {
		if s.State() != starting {
			t.Error("not starting during init")
		}
	}); err != nil {
		t.Fatal(err)
	}
	if s.State() != running {
		t.Fatalf("state %v, want running", s.State())
	}
	if !errors.Is(s.Start(func() {}), errNotIdle) {
		t.Fatal("second start not rejected")
	}
	s.Stop()
	if s.State() != stopped {
		t.Fatal("not stopped")
	}
}

func TestServerStopDuringStart(t *testing.T) {
	s := &server{}
	if !errors.Is(s.Start(func() { s.Stop() }), errStoppedDuringStart) {
		t.Fatal("start did not report the stop")
	}
	if s.State() != stopped {
		t.Fatalf("state %v, want stopped", s.State())
	}
}

func TestServerStartAfterStop(t *testing.T) {
	s := &server{}
	s.Stop()
	called := false
	if !errors.Is(s.Start(func() { called = true }), errNotIdle) || called {
		t.Fatal("start after stop ran")
	}
}

func TestLifecycleString(t *testing.T) {
	for l, want := range []string{"idle", "starting", "running", "stopped"} {
		if got := lifecycle(l).String(); got != want {
			t.Errorf("%d: %q, want %q", l, got, want)
		}
	}
}
