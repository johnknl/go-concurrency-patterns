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

package casloop

import (
	"sync/atomic"
)

// storeMax raises v to x unless v already holds a larger value.
func storeMax(v *atomic.Int64, x int64) {
	for {
		old := v.Load()
		if x <= old {
			return
		}
		if v.CompareAndSwap(old, x) {
			return
		}
	}
}

// tryIncrement adds one to n unless that would exceed limit, and reports whether it did.
func tryIncrement(n *atomic.Int64, limit int64) bool {
	for {
		old := n.Load()
		if old >= limit {
			return false
		}
		if n.CompareAndSwap(old, old+1) {
			return true
		}
	}
}

// stats is immutable once published.
type stats struct {
	count, sum int64
}

// record replaces the published stats with a new value derived from the old one.
func record(p *atomic.Pointer[stats], x int64) {
	for {
		old := p.Load()
		next := &stats{count: old.count + 1, sum: old.sum + x}
		if p.CompareAndSwap(old, next) {
			return
		}
	}
}
