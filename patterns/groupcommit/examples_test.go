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

package groupcommit

import (
	"errors"
	"fmt"
	"strings"
	"sync"
)

// Example_idle commits one item at a time. Nobody else is waiting, so each item is flushed at once, alone.
func Example_idle() {
	c := newCommitter(64, func(items []string) error {
		fmt.Println("flush", items)

		if strings.HasPrefix(items[0], "bad") {
			return errors.New("disk full")
		}

		return nil
	})

	fmt.Println(c.commit("a"))
	fmt.Println(c.commit("bad"))
	fmt.Println(c.commit("b"))

	// Output:
	// flush [a]
	// <nil>
	// flush [bad]
	// disk full
	// flush [b]
	// <nil>
}

// Example_concurrent commits from many goroutines. How the items are grouped depends on the schedule; every item
// is flushed exactly once, and no batch exceeds the bound.
func Example_concurrent() {
	var (
		mu      sync.Mutex
		flushed int
		largest int
	)

	c := newCommitter(16, func(items []string) error {
		mu.Lock()
		defer mu.Unlock()

		flushed += len(items)
		largest = max(largest, len(items))

		return nil
	})

	var wg sync.WaitGroup
	for i := range 100 {
		wg.Go(func() { _ = c.commit(fmt.Sprint(i)) })
	}
	wg.Wait()

	fmt.Println("items flushed:", flushed)
	fmt.Println("within bound:", largest <= 16)

	// Output:
	// items flushed: 100
	// within bound: true
}
