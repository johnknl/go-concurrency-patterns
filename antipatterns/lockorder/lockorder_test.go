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
//   - CONDITIONALS_BOUNDARY lockorder.go:52 (to.id < from.id -> <=): differs only for two accounts with the same
//     id, which the ordering scheme rules out.

package lockorder

import (
	"sync"
	"testing"
	"time"
)

func TestBadTransfer(t *testing.T) {
	t.Run("moves the amount", func(t *testing.T) {
		a, b := &account{id: 1, balance: 100}, &account{id: 2}
		if !badTransfer(a, b, 30, func() bool { return true }) {
			t.Fatal("transfer abandoned")
		}
		if a.balance != 70 || b.balance != 30 {
			t.Errorf("balances = %d, %d, want 70, 30", a.balance, b.balance)
		}
	})
	t.Run("abandoned transfer changes nothing", func(t *testing.T) {
		a, b := &account{id: 1, balance: 100}, &account{id: 2}
		if badTransfer(a, b, 30, func() bool { return false }) {
			t.Fatal("abandoned transfer reported done")
		}
		if a.balance != 100 || b.balance != 0 {
			t.Errorf("balances = %d, %d, want 100, 0", a.balance, b.balance)
		}
		if !a.mu.TryLock() || !b.mu.TryLock() {
			t.Error("abandoned transfer left a lock held")
		}
	})
	t.Run("the defect: opposite transfers wait on each other", func(t *testing.T) {
		a, b := &account{id: 1, balance: 100}, &account{id: 2, balance: 100}
		held := make(chan struct{})
		proceedAB, proceedBA := make(chan bool), make(chan bool)
		var wg sync.WaitGroup
		var doneAB, doneBA bool
		wg.Go(func() {
			doneAB = badTransfer(a, b, 10, func() bool { held <- struct{}{}; return <-proceedAB })
		})
		wg.Go(func() {
			doneBA = badTransfer(b, a, 20, func() bool { held <- struct{}{}; return <-proceedBA })
		})
		<-held
		<-held
		// Each holds its first lock, which is the other's second: neither can ever proceed.
		if a.mu.TryLock() || b.mu.TryLock() {
			t.Fatal("a first lock is free; there is no circular wait")
		}
		proceedBA <- false // break the cycle
		proceedAB <- true
		wg.Wait()
		if !doneAB || doneBA {
			t.Errorf("done = %v, %v, want true, false", doneAB, doneBA)
		}
		if a.balance != 90 || b.balance != 110 {
			t.Errorf("balances = %d, %d, want 90, 110", a.balance, b.balance)
		}
	})
	t.Run("takes the second lock", func(t *testing.T) {
		a, b := &account{id: 1, balance: 100}, &account{id: 2}
		b.mu.Lock()
		done := make(chan struct{})
		go func() {
			defer close(done)
			badTransfer(a, b, 10, func() bool { return true })
		}()
		select {
		case <-done:
			t.Error("transfer finished while the destination was locked")
		case <-time.After(20 * time.Millisecond):
		}
		b.mu.Unlock()
		<-done
	})
}

func TestTransfer(t *testing.T) {
	t.Run("moves the amount both ways", func(t *testing.T) {
		a, b := &account{id: 1, balance: 100}, &account{id: 2, balance: 100}
		transfer(a, b, 30)
		transfer(b, a, 5)
		if a.balance != 75 || b.balance != 125 {
			t.Errorf("balances = %d, %d, want 75, 125", a.balance, b.balance)
		}
	})
	t.Run("locks the lower id first", func(t *testing.T) {
		for _, tc := range []struct {
			name string
			from *account
			to   *account
		}{
			{"low to high", &account{id: 1}, &account{id: 2}},
			{"high to low", &account{id: 2}, &account{id: 1}},
		} {
			t.Run(tc.name, func(t *testing.T) {
				low, high := tc.from, tc.to
				if high.id < low.id {
					low, high = high, low
				}
				low.mu.Lock()
				done := make(chan struct{})
				go func() {
					defer close(done)
					transfer(tc.from, tc.to, 1)
				}()
				// Give transfer time to take any lock it would take before the lower id; it must take none.
				select {
				case <-done:
					t.Fatal("transfer finished while the lower id was locked")
				case <-time.After(20 * time.Millisecond):
				}
				if !high.mu.TryLock() {
					t.Error("transfer locked the higher id before the lower one")
				} else {
					high.mu.Unlock()
				}
				low.mu.Unlock()
				<-done
			})
		}
	})
	t.Run("opposite concurrent transfers do not deadlock", func(t *testing.T) {
		a, b := &account{id: 1, balance: 1000}, &account{id: 2, balance: 1000}
		var wg sync.WaitGroup
		for range 200 {
			wg.Go(func() { transfer(a, b, 3) })
			wg.Go(func() { transfer(b, a, 2) })
		}
		done := make(chan struct{})
		go func() { wg.Wait(); close(done) }()
		select {
		case <-done:
		case <-time.After(5 * time.Second):
			t.Fatal("transfers deadlocked")
		}
		if a.balance != 800 || b.balance != 1200 {
			t.Errorf("balances = %d, %d, want 800, 1200", a.balance, b.balance)
		}
	})
}
