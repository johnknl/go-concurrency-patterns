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

package nondetstate

import "math/rand/v2"

type state string

const (
	pending state = "pending"
	running state = "running"
	done    state = "done"
	failed  state = "failed"
)

const (
	start       = "start"
	attemptKind = "attempt"
)

// badEvent names what happened, but not how the attempt turned out.
type badEvent string

// attemptSucceeds is read inside the transition. Tests replace it; in production it is random.
var attemptSucceeds = func() bool { return rand.IntN(2) == 0 } //nolint:gosec // not security-relevant

func badNext(s state, e badEvent) state {
	switch {
	case s == pending && e == start:
		return running
	case s == running && e == attemptKind:
		if attemptSucceeds() {
			return done
		}
		return failed
	}
	return s
}

func badReplay(log []badEvent) state {
	s := pending
	for _, e := range log {
		s = badNext(s, e)
	}
	return s
}

// event carries the outcome of an attempt, so the transition only reads its inputs.
type event struct {
	kind string
	ok   bool
}

func next(s state, e event) state {
	switch {
	case s == pending && e.kind == start:
		return running
	case s == running && e.kind == attemptKind:
		if e.ok {
			return done
		}
		return failed
	}
	return s
}

func replay(log []event) state {
	s := pending
	for _, e := range log {
		s = next(s, e)
	}
	return s
}

// attempt decides the outcome outside the transition, with an injected source, and returns it as an event to
// record.
func attempt(succeeds func() bool) event {
	return event{kind: attemptKind, ok: succeeds()}
}
