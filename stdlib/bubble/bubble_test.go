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

package bubble

import (
	"errors"
	"testing"
	"testing/synctest"
	"time"
)

func TestTimeout(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		start := time.Now()

		_, err := receive(make(chan int), time.Minute) // nobody sends

		if !errors.Is(err, errTimeout) {
			t.Fatalf("err = %v, want errTimeout", err)
		}
		if waited := time.Since(start); waited != time.Minute {
			t.Fatalf("waited %v, want exactly 1m", waited)
		}
	})
}

func TestValueBeforeTimeout(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		c := make(chan int)
		go func() {
			time.Sleep(time.Minute - time.Nanosecond)
			c <- 42
		}()

		v, err := receive(c, time.Minute)
		if err != nil || v != 42 {
			t.Fatalf("got %d, %v, want 42, nil", v, err)
		}
	})
}

func TestWait(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		in := make(chan int)
		total := 0
		done := make(chan struct{})

		go func() {
			sum(in, &total)
			close(done)
		}()

		in <- 1
		in <- 2
		synctest.Wait() // sum is durably blocked on its next receive, so it has added 2

		if total != 3 {
			t.Fatalf("total = %d, want 3", total)
		}

		close(in)
		<-done
	})
}

func TestCondWait(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		g := newGate()
		passed := false

		go func() {
			g.wait()
			passed = true
		}()

		synctest.Wait() // the goroutine is in Cond.Wait, which is durably blocking
		if passed {
			t.Fatal("passed a closed gate")
		}

		g.release()
		synctest.Wait()

		if !passed {
			t.Fatal("did not pass an open gate")
		}
	})
}

func TestSleep(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		ticks := 0
		ticker := time.NewTicker(time.Second)
		stop := make(chan struct{})
		done := make(chan struct{})

		go func() {
			defer close(done)
			defer ticker.Stop()

			for {
				select {
				case <-ticker.C:
					ticks++
				case <-stop:
					return
				}
			}
		}()

		// The ticker fires at 3s, the same instant this sleep ends; synctest.Sleep also waits for the goroutine.
		synctest.Sleep(3 * time.Second)

		if ticks != 3 {
			t.Fatalf("ticks = %d, want 3", ticks)
		}

		close(stop)
		<-done
	})
}
