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
//

package prematureabstraction

import (
	"context"
	"fmt"
	"sync"
)

// Example_pool shows two callers of the configurable pool. The first gets an empty string for the failed job,
// which looks like a result. The second sets onError to log and stopOnError to stop, and stopOnError silently
// switches onError off.
func Example_pool() {
	names := []string{"a", "bad", "c"}

	var errs []error

	out, _ := pool(names, process, options{
		workers: 2,
		ordered: true,
		onError: func(_ int, err error) { errs = append(errs, err) },
	})
	fmt.Printf("%q %v\n", out, errs)

	var mu sync.Mutex

	logged := 0

	_, err := pool(names, process, options{
		workers:     2,
		stopOnError: true,
		onError: func(int, error) {
			mu.Lock()
			logged++
			mu.Unlock()
		},
	})
	fmt.Println(err, "logged:", logged)

	// Output:
	// ["a.done" "" "c.done"] [bad: bad input]
	// bad: bad input logged: 0
}

// Example_inline shows each caller with its own few lines of worker pool, doing only what it needs.
func Example_inline() {
	names := []string{"a", "bad", "c"}

	out, err := processAll(names)
	fmt.Printf("%q %v\n", out, err)

	logged := 0
	err = processUntilError(context.Background(), names, func(error) { logged++ })
	fmt.Println(err, "logged:", logged)

	// Output:
	// ["a.done" "" "c.done"] bad: bad input
	// bad: bad input logged: 1
}
