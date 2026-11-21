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
	"fmt"
	"sync"
)

func Example_monitor() {
	b := newBank(map[string]int{"a": 1000, "b": 1000})

	var wg sync.WaitGroup

	for range 100 {
		wg.Go(func() { b.transfer("a", "b", 3) })
		wg.Go(func() { b.transfer("b", "a", 1) })
		wg.Go(func() { _ = b.total() })
	}

	wg.Wait()
	fmt.Println(b.balance("a"), b.balance("b"), b.total())
	fmt.Println("overdraft allowed:", b.transfer("a", "b", 5000))

	// Output:
	// 800 1200 2000
	// overdraft allowed: false
}

func Example_readWrite() {
	r := &registry{names: map[int]string{}}

	var wg sync.WaitGroup

	for i := range 10 {
		wg.Go(func() { r.register(i, fmt.Sprint("worker-", i)) })
	}

	wg.Wait()

	names := make([]string, 20)

	for i := range names {
		wg.Go(func() { names[i], _ = r.lookup(i % 10) })
	}

	wg.Wait()
	fmt.Println(names[3], names[17])

	// Output:
	// worker-3 worker-7
}
