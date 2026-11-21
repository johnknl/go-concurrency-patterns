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

package monitor

import (
	"sync"
)

// bank owns its balances and the mutex that guards them. Every access goes through a method that holds the lock.
// Invariant, whenever mu is free: the balances add up to the amount deposited.
type bank struct {
	balances map[string]int
	mu       sync.Mutex
}

func newBank(balances map[string]int) *bank {
	return &bank{balances: balances}
}

// transfer moves amount from one account to another if the source has enough. Both changes happen under one lock,
// so no other method sees money that has left one account but not reached the other.
func (b *bank) transfer(from, to string, amount int) bool {
	b.mu.Lock()
	defer b.mu.Unlock()

	if b.balances[from] < amount {
		return false
	}

	b.balances[from] -= amount
	b.balances[to] += amount

	return true
}

func (b *bank) total() int {
	b.mu.Lock()
	defer b.mu.Unlock()

	sum := 0
	for _, v := range b.balances {
		sum += v
	}

	return sum
}

func (b *bank) balance(account string) int {
	b.mu.Lock()
	defer b.mu.Unlock()

	return b.balances[account]
}

// registry is read far more often than it is written, so readers share the lock.
type registry struct {
	names map[int]string
	mu    sync.RWMutex
}

func (r *registry) lookup(id int) (string, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	name, ok := r.names[id]

	return name, ok
}

func (r *registry) register(id int, name string) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.names[id] = name
}
