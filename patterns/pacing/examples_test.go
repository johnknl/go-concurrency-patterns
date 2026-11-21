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

package pacing

import (
	"context"
	"fmt"
	"time"
)

// Example_pacing spaces requests by waiting for a time.Ticker before each one. The output does not depend on the
// interval.
func Example_pacing() {
	ctx := context.Background()

	ticker := time.NewTicker(time.Millisecond)
	defer ticker.Stop()

	for i := 1; i <= 3; i++ {
		fmt.Println("request", i, pace(ctx, ticker.C))
	}

	// Output:
	// request 1 <nil>
	// request 2 <nil>
	// request 3 <nil>
}

// Example_pacer spaces requests a millisecond apart without a ticker; the first one starts at once.
func Example_pacer() {
	p := &pacer{interval: time.Millisecond}
	start := time.Now()

	for i := 1; i <= 3; i++ {
		fmt.Println("request", i, p.wait(context.Background()))
	}

	fmt.Println("took at least 2ms:", time.Since(start) >= 2*time.Millisecond)

	// Output:
	// request 1 <nil>
	// request 2 <nil>
	// request 3 <nil>
	// took at least 2ms: true
}
