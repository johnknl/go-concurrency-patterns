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

package doublecheck

import (
	"sync"
	"sync/atomic"
)

const loadedName = "loaded"

type config struct {
	name string
}

var (
	badMu       sync.Mutex
	badInstance *config
)

// badGet reads badInstance without synchronization before taking the lock.
func badGet() *config {
	if badInstance == nil { // unsynchronized read: races with the write below
		badMu.Lock()
		if badInstance == nil {
			badInstance = &config{name: loadedName}
		}
		badMu.Unlock()
	}
	return badInstance
}

var getConfig = sync.OnceValue(func() *config {
	return &config{name: loadedName}
})

var (
	mu       sync.Mutex
	instance atomic.Pointer[config]
)

// get loads atomically on the fast path and stores under the lock, so initialization runs once.
func get() *config {
	if c := instance.Load(); c != nil {
		return c
	}
	mu.Lock()
	defer mu.Unlock()
	if c := instance.Load(); c != nil {
		return c
	}
	c := &config{name: loadedName}
	instance.Store(c)
	return c
}
