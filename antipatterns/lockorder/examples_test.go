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
	"fmt"
	"sync"
)

func Example_circularWait() {
	a, b := &account{id: 1, balance: 100}, &account{id: 2, balance: 100}
	held := make(chan struct{})
	goAB, goBA := make(chan bool), make(chan bool)
	var wg sync.WaitGroup
	var doneAB, doneBA bool
	wg.Go(func() {
		doneAB = badTransfer(a, b, 10, func() bool { held <- struct{}{}; return <-goAB })
	})
	wg.Go(func() {
		doneBA = badTransfer(b, a, 20, func() bool { held <- struct{}{}; return <-goBA })
	})
	<-held
	<-held

	// Each transfer holds its first lock, and its next lock is the other's first: a circular wait.
	fmt.Println("a->b can lock b:", b.mu.TryLock())
	fmt.Println("b->a can lock a:", a.mu.TryLock())

	// Letting both continue would deadlock. Abandon b->a so that it unlocks b, then let a->b finish.
	goBA <- false
	goAB <- true
	wg.Wait()
	fmt.Println("a->b done:", doneAB, "b->a done:", doneBA)
	fmt.Println("balances:", a.balance, b.balance)
	// Output:
	// a->b can lock b: false
	// b->a can lock a: false
	// a->b done: true b->a done: false
	// balances: 90 110
}

func Example_lockOrderByID() {
	a, b := &account{id: 1, balance: 1000}, &account{id: 2, balance: 1000}
	var wg sync.WaitGroup
	for range 100 {
		wg.Go(func() { transfer(a, b, 3) })
		wg.Go(func() { transfer(b, a, 2) })
	}
	wg.Wait()
	fmt.Println("balances:", a.balance, b.balance)
	// Output:
	// balances: 900 1100
}
