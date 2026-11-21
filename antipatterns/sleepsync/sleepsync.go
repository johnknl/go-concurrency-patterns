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

package sleepsync

import (
	"sync/atomic"
	"time"
)

// job is work that finishes only after gate is closed, which stands in for a slow dependency.
type job struct {
	done   chan struct{}
	result atomic.Int64
}

func startJob(gate <-chan struct{}) *job {
	j := &job{done: make(chan struct{})}

	go func() {
		defer close(j.done)

		<-gate
		j.result.Store(42)
	}()

	return j
}

// badRead sleeps for a guessed time and then reads the result.
func badRead(j *job) int64 {
	time.Sleep(time.Millisecond)

	return j.result.Load()
}

// read waits until the job reports completion.
func read(j *job) int64 {
	<-j.done

	return j.result.Load()
}
