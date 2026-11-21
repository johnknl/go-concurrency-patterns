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

package closecause

import (
	"fmt"
	"sync"
)

// stream is a channel plus the reason it was closed.
type stream struct {
	err  error
	C    chan int
	once sync.Once
	mu   sync.Mutex
}

func newStream() *stream { return &stream{C: make(chan int, 1)} }

// finish stores the reason and then closes C. Only the first call has an effect, so calling it from
// several places cannot panic with a double close.
func (s *stream) finish(cause error) {
	s.once.Do(func() {
		s.mu.Lock()
		s.err = cause
		s.mu.Unlock()

		close(s.C)
	})
}

// Err returns nil while C is open and for a normal end, and the cause after a failed one.
func (s *stream) Err() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	return s.err
}

func drain(name string, s *stream) string {
	n := 0
	for range s.C {
		n++
	}

	return fmt.Sprint(name, " ", n, " ", s.Err())
}
