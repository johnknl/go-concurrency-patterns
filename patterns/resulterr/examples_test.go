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

package resulterr

import (
	"context"
	"errors"
	"fmt"
)

// Example_callerDecides lets the consumer, which knows what the values are for, choose the policy.
// Here a missing record is skipped, and any other error would stop the work.
func Example_callerDecides() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	for r := range lookupAll(ctx, 1, 2, 3, 4, 5) {
		if errors.Is(r.err, errNotFound) {
			fmt.Println("skip:", r.err)
			continue
		}

		if r.err != nil {
			fmt.Println("stop:", r.err)
			return
		}

		fmt.Println("got", r.value)
	}

	// Output:
	// got record-1
	// skip: lookup 2: not found
	// got record-3
	// skip: lookup 4: not found
	// got record-5
}

// Example_logAndContinue shows what the consumer sees when the goroutine handles its own errors.
func Example_logAndContinue() {
	var logs []string

	for value := range lookupQuietly(&logs, 1, 2, 3) {
		fmt.Println("got", value)
	}

	for _, line := range logs {
		fmt.Println("log:", line)
	}

	// Output:
	// got record-1
	// got record-3
	// log: lookup 2: not found
}
