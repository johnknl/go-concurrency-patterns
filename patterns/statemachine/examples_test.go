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
	"fmt"
	"time"
)

// Example_stateMachine runs a job through its owner. The second start is illegal in running and is ignored.
func Example_stateMachine() {
	release := make(chan struct{})
	work := func(ctx context.Context) error {
		select {
		case <-release:
			return nil
		case <-ctx.Done():
			return ctx.Err()
		}
	}

	j := startJob(context.Background(), next, work, 0)
	j.send(event{kind: start})
	j.send(event{kind: start})
	close(release)
	<-j.done

	fmt.Println("final:", j.final)
	fmt.Println("history:", j.history)

	// Output:
	// final: done
	// history: [start succeeded]
}

// Example_replay folds recorded events into a state again, without running any work.
func Example_replay() {
	history := []event{{kind: start}, {kind: workFailed, reason: "disk full"}}

	fmt.Println(replay(next, history))
	fmt.Println(replay(next, history[:1]))

	// Output:
	// failed
	// running
}

// Example_timeout uses nextTimed: the timer's firing is an event, so the transition to failed stays pure.
func Example_timeout() {
	work := func(ctx context.Context) error {
		<-ctx.Done()
		return ctx.Err()
	}

	j := startJob(context.Background(), nextTimed, work, 10*time.Millisecond)
	j.send(event{kind: start})
	<-j.done

	fmt.Println("final:", j.final)
	fmt.Println("history:", j.history)

	// Output:
	// final: failed
	// history: [start timeout]
}
