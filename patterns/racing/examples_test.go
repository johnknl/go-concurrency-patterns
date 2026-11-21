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

package racing

import (
	"context"
	"fmt"
	"slices"
)

// Example_firstAnswer releases only replica b, so it is the one that answers. The other two are canceled and
// have ended by the time first returns.
func Example_firstAnswer() {
	release := make(chan struct{})
	canceled := make(chan string, 2)

	close(release)

	never := make(chan struct{})

	answer := first(context.Background(),
		replica("a", never, canceled),
		replica("b", release, canceled),
		replica("c", never, canceled),
	)

	close(canceled)

	losers := slices.Sorted(func(yield func(string) bool) {
		for name := range canceled {
			if !yield(name) {
				return
			}
		}
	})

	fmt.Println("answer:", answer)
	fmt.Println("canceled:", losers)

	// Output:
	// answer: b
	// canceled: [a c]
}
