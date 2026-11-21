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

package lazyinit

import (
	"errors"
	"sync"
	"testing"
)

// loadConfig uses package-level state that the example may already have initialized, so this test only checks what
// holds in any order: the same config, built exactly once.
func TestLoadConfig(t *testing.T) {
	results := make([]*config, 20)
	var wg sync.WaitGroup
	for i := range results {
		wg.Go(func() { results[i] = loadConfig() })
	}
	wg.Wait()
	for _, c := range results {
		if c != results[0] {
			t.Fatal("callers got different configs")
		}
	}
	if results[0].name != "production" {
		t.Errorf("name = %q, want production", results[0].name)
	}
	if builds != 1 {
		t.Errorf("config built %d times, want 1", builds)
	}
}

func TestOnce(t *testing.T) {
	t.Run("concurrent callers wait for the first to finish", func(t *testing.T) {
		var once sync.Once
		release := make(chan struct{})
		var value, calls int
		var wg sync.WaitGroup
		for range 10 {
			wg.Go(func() {
				once.Do(func() {
					calls++
					<-release
					value = 42
				})
				if value != 42 {
					t.Errorf("Do returned before initialization finished: value = %d", value)
				}
			})
		}
		close(release)
		wg.Wait()
		if calls != 1 {
			t.Errorf("f ran %d times, want 1", calls)
		}
	})
	t.Run("a panic counts as done", func(t *testing.T) {
		var once sync.Once
		calls := 0
		func() {
			defer func() { _ = recover() }()
			once.Do(func() { calls++; panic("boom") })
		}()
		once.Do(func() { calls++ })
		if calls != 1 {
			t.Errorf("f ran %d times, want 1", calls)
		}
	})
}

func TestOnceValues(t *testing.T) {
	calls := 0
	errMissing := errors.New("missing")
	open := sync.OnceValues(func() (int, error) {
		calls++
		return 0, errMissing
	})
	for range 3 {
		if _, err := open(); !errors.Is(err, errMissing) {
			t.Errorf("err = %v, want %v", err, errMissing)
		}
	}
	if calls != 1 {
		t.Errorf("f ran %d times, want 1: an error is cached, not retried", calls)
	}
}

func TestOnceFunc_panicRepeats(t *testing.T) {
	calls := 0
	f := sync.OnceFunc(func() { calls++; panic("boom") })
	for range 2 {
		func() {
			defer func() {
				if r := recover(); r != "boom" {
					t.Errorf("recovered %v, want boom", r)
				}
			}()
			f()
		}()
	}
	if calls != 1 {
		t.Errorf("f ran %d times, want 1", calls)
	}
}
