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
//   - mutate buffer-channel statemachine.go:151, 152 (stopped, done): both are only closed, never sent on, so a
//     buffer changes nothing.

package statemachine

import (
	"context"
	"errors"
	"slices"
	"sync"
	"testing"
	"testing/synctest"
	"time"
)

var (
	allStates = []state{pending, running, done, failed, canceled}
	allKinds  = []eventKind{start, succeeded, workFailed, cancel, timeout}
)

type outcome struct {
	to      state
	effects []effect
}

// checkGrid checks every state × event pair: listed pairs are legal, all others must be illegal and change nothing.
func checkGrid(t *testing.T, transition func(state, event) (state, []effect, error), legal map[[2]int]outcome) {
	t.Helper()

	for _, s := range allStates {
		for _, k := range allKinds {
			got, effects, err := transition(s, event{kind: k})

			want, ok := legal[[2]int{int(s), int(k)}]
			if !ok {
				if !errors.Is(err, errIllegal) || got != s || effects != nil {
					t.Errorf("%v + %v = (%v, %v, %v), want illegal and unchanged", s, k, got, effects, err)
				}

				continue
			}

			if err != nil || got != want.to || !slices.Equal(effects, want.effects) {
				t.Errorf("%v + %v = (%v, %v, %v), want (%v, %v)", s, k, got, effects, err, want.to, want.effects)
			}
		}
	}
}

func TestNextTable(t *testing.T) {
	checkGrid(t, next, map[[2]int]outcome{
		{int(pending), int(start)}:      {running, []effect{startWork}},
		{int(pending), int(cancel)}:     {canceled, nil},
		{int(running), int(succeeded)}:  {done, nil},
		{int(running), int(workFailed)}: {failed, nil},
		{int(running), int(cancel)}:     {canceled, []effect{stopWork}},
	})
}

func TestNextTimedTable(t *testing.T) {
	checkGrid(t, nextTimed, map[[2]int]outcome{
		{int(pending), int(start)}:      {running, []effect{startWork, armTimer}},
		{int(pending), int(cancel)}:     {canceled, nil},
		{int(running), int(succeeded)}:  {done, nil},
		{int(running), int(workFailed)}: {failed, nil},
		{int(running), int(cancel)}:     {canceled, []effect{stopWork}},
		{int(running), int(timeout)}:    {failed, []effect{stopWork}},
	})
}

func TestStrings(t *testing.T) {
	if got := (event{kind: workFailed, reason: "x"}).String(); got != "failed(x)" {
		t.Errorf("got %q", got)
	}

	if canceled.String() != "canceled" || cancel.String() != "cancel" {
		t.Error("wrong names")
	}
}

func TestReplayIsDeterministic(t *testing.T) {
	histories := [][]event{
		nil,
		{{kind: start}},
		{{kind: start}, {kind: succeeded}},
		{{kind: start}, {kind: start}, {kind: workFailed, reason: "boom"}},
		{{kind: cancel}, {kind: start}},
		{{kind: start}, {kind: timeout}},
	}
	want := []state{pending, running, done, failed, canceled, running}

	for i, h := range histories {
		first := replay(next, h)
		for range 100 {
			if got := replay(next, h); got != first {
				t.Fatalf("history %v: replay gave %v, then %v", h, first, got)
			}
		}

		if first != want[i] {
			t.Errorf("history %v: got %v, want %v", h, first, want[i])
		}
	}
}

// TestOwnerHistoryReplays runs the owner with many concurrent senders and checks that folding what it recorded
// gives the state it ended in.
func TestOwnerHistoryReplays(t *testing.T) {
	for range 50 {
		release := make(chan struct{})
		work := func(ctx context.Context) error {
			select {
			case <-release:
				return errors.New("boom")
			case <-ctx.Done():
				return ctx.Err()
			}
		}

		j := startJob(t.Context(), next, work, 0)

		var wg sync.WaitGroup
		for i := range 20 {
			wg.Go(func() {
				if i%2 == 0 {
					j.send(event{kind: start})
				} else {
					j.send(event{kind: cancel})
				}
			})
		}

		close(release)
		wg.Wait()
		<-j.done

		if !j.final.final() {
			t.Fatalf("ended in %v", j.final)
		}

		if got := replay(next, j.history); got != j.final {
			t.Fatalf("history %v replays to %v, owner ended in %v", j.history, got, j.final)
		}
	}
}

func TestFailureReasonIsRecorded(t *testing.T) {
	work := func(context.Context) error { return errors.New("disk full") }

	j := startJob(t.Context(), next, work, 0)
	j.send(event{kind: start})
	<-j.done

	if j.final != failed || len(j.history) != 2 || j.history[1].reason != "disk full" {
		t.Fatalf("final %v, history %v", j.final, j.history)
	}
}

func TestCancelEventStopsWork(t *testing.T) {
	stoppedWork := make(chan struct{})
	work := func(ctx context.Context) error {
		<-ctx.Done()
		close(stoppedWork)

		return ctx.Err()
	}

	j := startJob(t.Context(), next, work, 0)
	j.send(event{kind: start})
	j.send(event{kind: cancel})
	<-j.done
	<-stoppedWork

	if j.final != canceled {
		t.Fatalf("final %v", j.final)
	}

	if j.send(event{kind: start}) {
		t.Fatal("send after the owner stopped reported delivery")
	}
}

func TestContextCancellationStopsOwner(t *testing.T) {
	ctx, cancelCtx := context.WithCancel(t.Context())
	workEnded := make(chan struct{})
	work := func(ctx context.Context) error {
		defer close(workEnded)
		<-ctx.Done()

		return ctx.Err()
	}

	j := startJob(ctx, next, work, 0)
	j.send(event{kind: start})
	cancelCtx()
	<-j.done

	select {
	case <-workEnded:
	default:
		t.Fatal("done closed before the work ended")
	}

	if j.final != running {
		t.Fatalf("final %v, want running", j.final)
	}
}

func TestDoneWaitsForEffects(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		work := func(ctx context.Context) error {
			<-ctx.Done()
			time.Sleep(time.Second) // slow cleanup

			return ctx.Err()
		}

		begin := time.Now()
		j := startJob(t.Context(), next, work, 0)
		j.send(event{kind: start})
		j.send(event{kind: cancel})
		<-j.done

		if elapsed := time.Since(begin); elapsed != time.Second {
			t.Fatalf("done after %v, want after the work's cleanup", elapsed)
		}
	})
}

func TestTimeout(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		work := func(ctx context.Context) error {
			<-ctx.Done()
			return ctx.Err()
		}

		begin := time.Now()
		j := startJob(t.Context(), nextTimed, work, time.Minute)
		j.send(event{kind: start})
		<-j.done

		if j.final != failed || time.Since(begin) != time.Minute {
			t.Fatalf("final %v after %v", j.final, time.Since(begin))
		}
	})
}

func TestTimerDoesNotFireAfterSuccess(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		begin := time.Now()
		j := startJob(t.Context(), nextTimed, func(context.Context) error { return nil }, time.Minute)
		j.send(event{kind: start})
		<-j.done

		if elapsed := time.Since(begin); elapsed != 0 {
			t.Fatalf("done after %v: the timer outlived the job", elapsed)
		}

		if j.final != done || !slices.Equal(j.history, []event{{kind: start}, {kind: succeeded}}) {
			t.Fatalf("final %v, history %v", j.final, j.history)
		}
	})
}
