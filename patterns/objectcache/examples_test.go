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

package objectcache

import (
	"bytes"
	"fmt"
)

// Example_syncPool shows that a buffer from the pool is always empty. Whether it is a reused buffer or a new
// one makes no difference to the output.
func Example_syncPool() {
	for _, name := range []string{"ann", "bob", "cy"} {
		buf := buffers.Get().(*bytes.Buffer) //nolint:errcheck // New only creates buffers
		fmt.Println("empty:", buf.Len() == 0)
		buffers.Put(buf)

		fmt.Println(render(name))
	}

	// Output:
	// empty: true
	// hello, ann
	// empty: true
	// hello, bob
	// empty: true
	// hello, cy
}
