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

package statemachine

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"
)

var errIllegal = errors.New("illegal transition")

type state int

const (
	pending state = iota
	running
	done
	failed
	canceled
)

func (s state) String() string {
	return [...]string{"pending", "running", "done", "failed", "canceled"}[s]
}

func (s state) final() bool { return s == done || s == failed || s == canceled }

type eventKind int

const (
	start eventKind = iota
	succeeded
	workFailed
	cancel
	timeout
)

func (k eventKind) String() string {
	return [...]string{"start", "succeeded", "failed", "cancel", "timeout"}[k]
}

// event is an input. Everything a transition depends on is in it, such as why the work failed.
type event struct {
	reason string
	kind   eventKind
}

func (e event) String() string {
	if e.reason != "" {
		return fmt.Sprintf("%v(%s)", e.kind, e.reason)
	}

	return e.kind.String()
}

// effect is work the transition asks for. The owner runs it; next only returns it.
type effect int

const (
	startWork effect = iota
	stopWork
	armTimer
)

// next is the whole machine: it reads only its arguments, so the same state and event always give the same result.
// An illegal event leaves the state unchanged and returns errIllegal.
func next(s state, e event) (state, []effect, error) {
	switch {
	case s == pending && e.kind == start:
		return running, []effect{startWork}, nil
	case s == pending && e.kind == cancel:
		return canceled, nil, nil
	case s == running && e.kind == succeeded:
		return done, nil, nil
	case s == running && e.kind == workFailed:
		return failed, nil, nil
	case s == running && e.kind == cancel:
		return canceled, []effect{stopWork}, nil
	}

	return s, nil, fmt.Errorf("%w: %v in %v", errIllegal, e.kind, s)
}

// nextTimed adds a deadline to running: start also arms a timer, and its firing is one more event.
func nextTimed(s state, e event) (state, []effect, error) {
	switch {
	case s == pending && e.kind == start:
		return running, []effect{startWork, armTimer}, nil
	case s == running && e.kind == timeout:
		return failed, []effect{stopWork}, nil
	}

	return next(s, e)
}

// replay folds events into a state, as the owner does, without running any effect.
func replay(transition func(state, event) (state, []effect, error), events []event) state {
	s := pending
	for _, e := range events {
		s, _, _ = transition(s, e)
	}

	return s
}

// job is the owner of one machine. The state lives only inside run.
type job struct {
	transition func(state, event) (state, []effect, error)
	work       func(context.Context) error
	events     chan event
	stopped    chan struct{} // closed when run stops receiving
	done       chan struct{} // closed when run and its helpers have ended
	history    []event       // the inputs in the order applied; read after done
	timeout    time.Duration
	final      state // read after done
}

func startJob(
	ctx context.Context,
	transition func(state, event) (state, []effect, error),
	work func(context.Context) error,
	timeout time.Duration,
) *job {
	j := &job{
		transition: transition,
		work:       work,
		timeout:    timeout,
		events:     make(chan event),
		stopped:    make(chan struct{}),
		done:       make(chan struct{}),
	}
	go j.run(ctx)

	return j
}

// send delivers an event, or reports that the owner has stopped.
func (j *job) send(e event) bool {
	select {
	case j.events <- e:
		return true
	case <-j.stopped:
		return false
	}
}

func (j *job) run(ctx context.Context) {
	workCtx, stop := context.WithCancel(ctx)

	var helpers sync.WaitGroup

	defer func() {
		close(j.stopped)
		stop()
		helpers.Wait()
		close(j.done)
	}()

	s := pending

	for {
		select {
		case <-ctx.Done():
			j.final = s
			return
		case e := <-j.events:
			ns, effects, err := j.transition(s, e)
			if err != nil {
				continue // illegal events are ignored and not recorded
			}

			j.history = append(j.history, e)
			s = ns

			for _, f := range effects {
				j.execute(workCtx, stop, &helpers, f)
			}

			if s.final() {
				j.final = s
				return
			}
		}
	}
}

// execute runs one effect. Results come back to the owner as events.
func (j *job) execute(ctx context.Context, stop context.CancelFunc, helpers *sync.WaitGroup, f effect) {
	switch f {
	case startWork:
		helpers.Go(func() {
			if err := j.work(ctx); err != nil {
				j.send(event{kind: workFailed, reason: err.Error()})
				return
			}

			j.send(event{kind: succeeded})
		})
	case stopWork:
		stop()
	case armTimer:
		helpers.Go(func() {
			t := time.NewTimer(j.timeout)
			defer t.Stop()

			select {
			case <-t.C:
				j.send(event{kind: timeout})
			case <-ctx.Done():
			}
		})
	}
}
