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

package snapshot

import (
	"fmt"
	"sync"
)

// Example_snapshot reads the counters after other goroutines have recorded events, then shows that the snapshot is a
// copy.
func Example_snapshot() {
	s := newStats()

	var wg sync.WaitGroup
	for range 50 {
		wg.Go(func() { s.record("hit") })
		wg.Go(func() { s.record("miss") })
	}

	wg.Wait()

	snap := s.snapshot()
	fmt.Println(snap["hit"], snap["miss"])

	// The snapshot is a copy: changing it does not change the owner's state.
	snap["hit"] = 0
	s.record("hit")
	fmt.Println(s.snapshot()["hit"])

	close(s.quit)
	<-s.done

	// Output:
	// 50 50
	// 51
}
