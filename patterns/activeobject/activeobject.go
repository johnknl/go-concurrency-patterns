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
	"errors"
)

var (
	errStopped      = errors.New("account stopped")
	errInsufficient = errors.New("insufficient funds")
)

// Commands are plain values. Those that need an answer carry a reply channel with room for one value.
type deposit struct{ amount int }

type withdraw struct {
	reply  chan error
	amount int
}

type balance struct{ reply chan int }

// account is the proxy. The total lives only inside run, so no field here is shared state.
type account struct {
	commands chan any
	done     chan struct{}
}

func newAccount(ctx context.Context) *account {
	a := &account{commands: make(chan any), done: make(chan struct{})}
	go a.run(ctx)

	return a
}

// run is the owner: it executes one command at a time.
func (a *account) run(ctx context.Context) {
	defer close(a.done)

	total := 0

	for {
		select {
		case <-ctx.Done():
			return
		case command := <-a.commands:
			switch c := command.(type) {
			case deposit:
				total += c.amount
			case withdraw:
				if c.amount > total {
					c.reply <- errInsufficient
					continue
				}

				total -= c.amount
				c.reply <- nil
			case balance:
				c.reply <- total
			}
		}
	}
}

// send hands a command to the owner, or reports that the owner has stopped.
func (a *account) send(command any) bool {
	select {
	case a.commands <- command:
		return true
	case <-a.done:
		return false
	}
}

func (a *account) Deposit(amount int) error {
	if !a.send(deposit{amount}) {
		return errStopped
	}

	return nil
}

func (a *account) Withdraw(amount int) error {
	reply := make(chan error, 1)
	if !a.send(withdraw{amount: amount, reply: reply}) {
		return errStopped
	}

	return <-reply
}

func (a *account) Balance() (int, error) {
	reply := make(chan int, 1)
	if !a.send(balance{reply}) {
		return 0, errStopped
	}

	return <-reply, nil
}
