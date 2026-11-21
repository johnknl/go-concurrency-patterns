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
//   - gremlins INVERT_LOOPCTRL activeobject.go:75: break leaves the switch, which skips the withdrawal just like
//     continue does.
//   - mutate buffer-channel activeobject.go:52 (commands): commands stay in FIFO order, and a buffered deposit lost
//     at cancel cannot be observed afterwards.
//   - mutate unbuffer-channel activeobject.go:106, 115: Withdraw and Balance always receive their reply.

package activeobject

import (
	"context"
	"errors"
	"sync"
	"testing"
	"testing/synctest"
)

func TestAccount(t *testing.T) {
	t.Run("starts empty", func(t *testing.T) {
		synctest.Test(t, func(t *testing.T) {
			a := newAccount(t.Context())

			if b, err := a.Balance(); b != 0 || err != nil {
				t.Fatalf("got %d, %v", b, err)
			}
		})
	})

	t.Run("deposit and withdraw", func(t *testing.T) {
		synctest.Test(t, func(t *testing.T) {
			a := newAccount(t.Context())

			if err := a.Deposit(10); err != nil {
				t.Fatal(err)
			}

			if err := a.Withdraw(4); err != nil {
				t.Fatal(err)
			}

			if b, _ := a.Balance(); b != 6 {
				t.Fatalf("balance %d, want 6", b)
			}
		})
	})

	t.Run("withdrawing the whole balance is allowed", func(t *testing.T) {
		synctest.Test(t, func(t *testing.T) {
			a := newAccount(t.Context())
			_ = a.Deposit(5)

			if err := a.Withdraw(5); err != nil {
				t.Fatalf("got %v, want nil", err)
			}

			if b, _ := a.Balance(); b != 0 {
				t.Fatalf("balance %d, want 0", b)
			}
		})
	})

	t.Run("insufficient funds leave the balance alone", func(t *testing.T) {
		synctest.Test(t, func(t *testing.T) {
			a := newAccount(t.Context())
			_ = a.Deposit(5)

			if err := a.Withdraw(6); !errors.Is(err, errInsufficient) {
				t.Fatalf("got %v, want %v", err, errInsufficient)
			}

			if b, _ := a.Balance(); b != 5 {
				t.Fatalf("balance %d, want 5", b)
			}
		})
	})

	t.Run("concurrent callers are serialized", func(t *testing.T) {
		synctest.Test(t, func(t *testing.T) {
			a := newAccount(t.Context())

			var wg sync.WaitGroup
			for range 100 {
				wg.Go(func() { _ = a.Deposit(1) })
			}

			wg.Wait()

			var failed int

			var mu sync.Mutex
			for range 150 {
				wg.Go(func() {
					if a.Withdraw(1) != nil {
						mu.Lock()
						failed++
						mu.Unlock()
					}
				})
			}

			wg.Wait()

			if b, _ := a.Balance(); b != 0 || failed != 50 {
				t.Fatalf("balance %d, failed %d; want 0 and 50", b, failed)
			}
		})
	})

	t.Run("cancel stops the owner", func(t *testing.T) {
		synctest.Test(t, func(t *testing.T) {
			ctx, cancel := context.WithCancel(t.Context())
			a := newAccount(ctx)
			_ = a.Deposit(5)

			cancel()
			<-a.done

			if err := a.Deposit(1); !errors.Is(err, errStopped) {
				t.Fatalf("Deposit: got %v", err)
			}

			if err := a.Withdraw(1); !errors.Is(err, errStopped) {
				t.Fatalf("Withdraw: got %v", err)
			}

			if b, err := a.Balance(); b != 0 || !errors.Is(err, errStopped) {
				t.Fatalf("Balance: got %d, %v", b, err)
			}
		})
	})

	t.Run("cancel releases a caller waiting to send", func(t *testing.T) {
		synctest.Test(t, func(t *testing.T) {
			ctx, cancel := context.WithCancel(t.Context())
			a := newAccount(ctx)

			cancel()

			if err := a.Deposit(1); err != nil && !errors.Is(err, errStopped) {
				t.Fatalf("got %v", err)
			}
		})
	})
}
