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

package activeobject

import (
	"context"
	"fmt"
	"sync"
)

// Example_activeObject calls the account from many goroutines. There is no lock, and the total is exact.
func Example_activeObject() {
	ctx, cancel := context.WithCancel(context.Background())
	acct := newAccount(ctx)

	var wg sync.WaitGroup
	for range 100 {
		wg.Go(func() { _ = acct.Deposit(10) })
	}

	wg.Wait()

	total, _ := acct.Balance()
	fmt.Println("balance:", total)
	fmt.Println("withdraw 5000:", acct.Withdraw(5000))
	fmt.Println("withdraw 400:", acct.Withdraw(400))

	cancel()
	<-acct.done

	_, err := acct.Balance()
	fmt.Println("after stop:", err)

	// Output:
	// balance: 1000
	// withdraw 5000: insufficient funds
	// withdraw 400: <nil>
	// after stop: account stopped
}
