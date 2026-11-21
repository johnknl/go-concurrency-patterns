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

package cowsnapshot

import (
	"fmt"
	"sync"
	"sync/atomic"
)

// Example_snapshot moves a service: host and port change in one version, so a reader never sees a mixed pair.
func Example_snapshot() {
	var s snapshot
	s.cur.Store(&config{"host": "a.example", "port": "80"})

	old := s.load()

	var wg sync.WaitGroup
	for range 4 {
		wg.Go(func() {
			c := s.load() // one load per read: host and port come from the same version
			_ = c["host"] + ":" + c["port"]
		})
	}
	wg.Go(func() {
		s.update(func(next config) {
			next["host"] = "b.example"
			next["port"] = "8080"
		})
	})
	wg.Wait()

	cur := s.load()
	fmt.Println(old["host"]+":"+old["port"], cur["host"]+":"+cur["port"])

	// Output:
	// a.example:80 b.example:8080
}

// Example_compareAndSwap lets several writers update the config without a mutex.
func Example_compareAndSwap() {
	var cur atomic.Pointer[config]
	cur.Store(&config{})

	var wg sync.WaitGroup
	for i := range 10 {
		wg.Go(func() {
			updateCAS(&cur, func(next config) { next[fmt.Sprint("k", i)] = "v" })
		})
	}
	wg.Wait()

	fmt.Println(len(*cur.Load()))

	// Output:
	// 10
}
