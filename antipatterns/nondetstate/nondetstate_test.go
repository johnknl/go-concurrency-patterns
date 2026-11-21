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

package nondetstate

import "testing"

func TestBadNextDependsOnHiddenSource(t *testing.T) {
	defer func(f func() bool) { attemptSucceeds = f }(attemptSucceeds)
	for _, ok := range []bool{true, false} {
		attemptSucceeds = func() bool { return ok }
		want := map[bool]state{true: done, false: failed}[ok]
		if got := badReplay([]badEvent{"start", "attempt"}); got != want {
			t.Errorf("ok=%v: %v, want %v", ok, got, want)
		}
	}
}

func TestBadNextIgnoresIllegalEvents(t *testing.T) {
	if got := badNext(pending, "attempt"); got != pending {
		t.Errorf("attempt while pending: %v", got)
	}
	if got := badNext(done, "start"); got != done {
		t.Errorf("start while done: %v", got)
	}
}

func TestDefaultSourceProducesBoth(t *testing.T) {
	seen := map[bool]bool{}
	for i := 0; i < 1000 && len(seen) < 2; i++ {
		seen[attemptSucceeds()] = true
	}
	if len(seen) != 2 {
		t.Fatal("default source is not random")
	}
}

func TestNext(t *testing.T) {
	cases := []struct {
		s    state
		e    event
		want state
	}{
		{pending, event{kind: "start"}, running},
		{pending, event{kind: "attempt", ok: true}, pending},
		{running, event{kind: "attempt", ok: true}, done},
		{running, event{kind: "attempt"}, failed},
		{running, event{kind: "start"}, running},
		{done, event{kind: "start"}, done},
	}
	for _, c := range cases {
		if got := next(c.s, c.e); got != c.want {
			t.Errorf("next(%v, %v) = %v, want %v", c.s, c.e, got, c.want)
		}
	}
}

func TestReplayIsDeterministic(t *testing.T) {
	for _, ok := range []bool{true, false} {
		log := []event{{kind: "start"}, attempt(func() bool { return ok })}
		first := replay(log)
		for range 10 {
			if got := replay(log); got != first {
				t.Fatalf("replay %v, first %v", got, first)
			}
		}
		if want := map[bool]state{true: done, false: failed}[ok]; first != want {
			t.Errorf("ok=%v: %v, want %v", ok, first, want)
		}
	}
}
