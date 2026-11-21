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

package monitor

import (
	"sync"
	"testing"
	"time"
)

// assertWaitsForLock holds mu, runs f in a goroutine and checks that f does not finish until mu is released. A
// broken f that skips the lock finishes within the grace period; a correct f can never fail this check.
func assertWaitsForLock(t *testing.T, mu sync.Locker, f func()) {
	t.Helper()
	mu.Lock()
	done := make(chan struct{})
	go func() {
		defer close(done)
		f()
	}()
	select {
	case <-done:
		mu.Unlock()
		t.Fatal("finished while the lock was held by someone else")
	case <-time.After(20 * time.Millisecond):
	}
	mu.Unlock()
	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("did not finish after the lock was released")
	}
}

func TestBank_transfer(t *testing.T) {
	t.Run("moves the amount", func(t *testing.T) {
		b := newBank(map[string]int{"a": 100, "b": 0})
		if !b.transfer("a", "b", 30) {
			t.Fatal("transfer refused")
		}
		if got, want := b.balance("a"), 70; got != want {
			t.Errorf("a = %d, want %d", got, want)
		}
		if got, want := b.balance("b"), 30; got != want {
			t.Errorf("b = %d, want %d", got, want)
		}
	})
	t.Run("whole balance is allowed", func(t *testing.T) {
		b := newBank(map[string]int{"a": 50})
		if !b.transfer("a", "b", 50) {
			t.Fatal("transfer of the exact balance refused")
		}
		if b.balance("a") != 0 || b.balance("b") != 50 {
			t.Errorf("balances = %d, %d, want 0, 50", b.balance("a"), b.balance("b"))
		}
	})
	t.Run("insufficient funds changes nothing", func(t *testing.T) {
		b := newBank(map[string]int{"a": 10, "b": 5})
		if b.transfer("a", "b", 11) {
			t.Fatal("overdraft allowed")
		}
		if b.balance("a") != 10 || b.balance("b") != 5 {
			t.Errorf("balances = %d, %d, want 10, 5", b.balance("a"), b.balance("b"))
		}
	})
	t.Run("unknown account has zero balance", func(t *testing.T) {
		b := newBank(map[string]int{})
		if b.transfer("x", "y", 1) {
			t.Fatal("transfer from an empty account allowed")
		}
		if b.balance("x") != 0 {
			t.Errorf("balance = %d, want 0", b.balance("x"))
		}
	})
	t.Run("concurrent transfers keep the total", func(t *testing.T) {
		b := newBank(map[string]int{"a": 1000, "b": 1000, "c": 1000})
		accounts := []string{"a", "b", "c"}
		var wg sync.WaitGroup
		for i := range 300 {
			wg.Go(func() { b.transfer(accounts[i%3], accounts[(i+1)%3], i%7) })
			wg.Go(func() {
				if got := b.total(); got != 3000 {
					t.Errorf("total = %d during transfers, want 3000", got)
				}
			})
		}
		wg.Wait()
		if got := b.total(); got != 3000 {
			t.Errorf("total = %d, want 3000", got)
		}
	})
	t.Run("waits for the lock", func(t *testing.T) {
		b := newBank(map[string]int{"a": 1})
		assertWaitsForLock(t, &b.mu, func() { b.transfer("a", "b", 1) })
	})
}

func TestBank_total(t *testing.T) {
	if got := newBank(map[string]int{}).total(); got != 0 {
		t.Errorf("empty total = %d, want 0", got)
	}
	b := newBank(map[string]int{"a": 1, "b": 2, "c": 3})
	if got := b.total(); got != 6 {
		t.Errorf("total = %d, want 6", got)
	}
	assertWaitsForLock(t, &b.mu, func() { b.total() })
}

func TestBank_balance(t *testing.T) {
	b := newBank(map[string]int{"a": 7})
	if got := b.balance("a"); got != 7 {
		t.Errorf("balance = %d, want 7", got)
	}
	assertWaitsForLock(t, &b.mu, func() { b.balance("a") })
}

func TestRegistry(t *testing.T) {
	t.Run("lookup of a missing id", func(t *testing.T) {
		r := &registry{names: map[int]string{}}
		if name, ok := r.lookup(1); ok || name != "" {
			t.Errorf("lookup = %q, %v, want \"\", false", name, ok)
		}
	})
	t.Run("register then lookup", func(t *testing.T) {
		r := &registry{names: map[int]string{}}
		r.register(1, "one")
		r.register(1, "uno")
		if name, ok := r.lookup(1); !ok || name != "uno" {
			t.Errorf("lookup = %q, %v, want \"uno\", true", name, ok)
		}
	})
	t.Run("readers share the lock", func(t *testing.T) {
		r := &registry{names: map[int]string{1: "one"}}
		r.mu.RLock()
		defer r.mu.RUnlock()
		done := make(chan string)
		go func() {
			name, _ := r.lookup(1)
			done <- name
		}()
		select {
		case name := <-done:
			if name != "one" {
				t.Errorf("lookup = %q, want one", name)
			}
		case <-time.After(5 * time.Second):
			t.Fatal("lookup blocked behind another reader")
		}
	})
	t.Run("register waits for readers", func(t *testing.T) {
		r := &registry{names: map[int]string{}}
		assertWaitsForLock(t, r.mu.RLocker(), func() { r.register(1, "one") })
	})
	t.Run("lookup waits for a writer", func(t *testing.T) {
		r := &registry{names: map[int]string{}}
		assertWaitsForLock(t, &r.mu, func() { r.lookup(1) })
	})
	t.Run("concurrent readers and writers", func(t *testing.T) {
		r := &registry{names: map[int]string{}}
		var wg sync.WaitGroup
		for i := range 100 {
			wg.Go(func() { r.register(i, "x") })
			wg.Go(func() { r.lookup(i) })
		}
		wg.Wait()
		if len(r.names) != 100 {
			t.Errorf("registered %d names, want 100", len(r.names))
		}
	})
}
