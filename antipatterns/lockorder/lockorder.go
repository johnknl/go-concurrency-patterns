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

package lockorder

import (
	"sync"
)

type account struct {
	id      int
	mu      sync.Mutex
	balance int
}

// badTransfer locks from, then to, so two transfers in opposite directions lock in opposite orders. between
// runs while only from is locked; it lets the example pause there and, by returning false, abandon the transfer.
func badTransfer(from, to *account, amount int, between func() bool) bool {
	from.mu.Lock()
	defer from.mu.Unlock()
	if !between() {
		return false
	}
	to.mu.Lock()
	defer to.mu.Unlock()
	from.balance -= amount
	to.balance += amount
	return true
}

// transfer always locks the account with the lower id first, so all transfers agree on the order.
func transfer(from, to *account, amount int) {
	first, second := from, to
	if to.id < from.id {
		first, second = to, from
	}
	first.mu.Lock()
	defer first.mu.Unlock()
	second.mu.Lock()
	defer second.mu.Unlock()
	from.balance -= amount
	to.balance += amount
}
