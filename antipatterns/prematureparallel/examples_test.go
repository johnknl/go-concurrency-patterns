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

package prematureparallel

import "fmt"

// Example_parallel squares five numbers with a generator, four workers, and a result channel.
func Example_parallel() {
	out, goroutines, handoffs := squaresParallel([]int{5, 1, 4, 2, 3}, 4)

	fmt.Println(out)
	fmt.Println("goroutines:", goroutines, "handoffs:", handoffs, "multiplications:", len(out))

	// Output:
	// [1 4 9 16 25]
	// goroutines: 6 handoffs: 10 multiplications: 5
}

// Example_loop squares the same numbers in a loop, in input order.
func Example_loop() {
	fmt.Println(squares([]int{5, 1, 4, 2, 3}))

	// Output:
	// [25 1 16 4 9]
}

// Example_bounded runs work that is worth a goroutine, at most two at a time, and keeps the input order.
func Example_bounded() {
	fetch := func(id int) string { return fmt.Sprintf("page-%d", id) } // stands in for blocking I/O

	fmt.Println(mapBounded([]int{3, 1, 2}, 2, fetch))

	// Output:
	// [page-3 page-1 page-2]
}
