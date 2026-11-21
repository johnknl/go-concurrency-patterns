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

package unbounded

import (
	"sync"
	"sync/atomic"
)

// peak records how many calls are inside work at the same time.
type peak struct{ cur, max atomic.Int64 }

func (p *peak) enter() {
	n := p.cur.Add(1)

	for {
		m := p.max.Load()
		if n <= m || p.max.CompareAndSwap(m, n) {
			return
		}
	}
}

func (p *peak) leave() { p.cur.Add(-1) }

// badFanOut starts one goroutine per item, however many items there are.
func badFanOut(items []int, work func(int)) {
	var wg sync.WaitGroup

	for _, item := range items {
		wg.Go(func() { work(item) })
	}

	wg.Wait()
}

// boundedFanOut runs work for every item on at most limit goroutines; the unbuffered jobs channel makes the
// producer wait while all workers are busy.
func boundedFanOut(items []int, limit int, work func(int)) {
	jobs := make(chan int)

	var wg sync.WaitGroup

	for range limit {
		wg.Go(func() {
			for item := range jobs {
				work(item)
			}
		})
	}

	for _, item := range items {
		jobs <- item
	}

	close(jobs)
	wg.Wait()
}
