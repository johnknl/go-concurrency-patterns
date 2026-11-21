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

package fireandforget

import (
	"errors"
	"sync"
)

// leakyStart starts the work and returns. Nobody can wait for the goroutine, stop it or receive its error; report
// stands in for the log line or metric where the error would end up.
func leakyStart(work func() error, report func(error)) error {
	if work == nil {
		return errors.New("no work")
	}

	go func() {
		if err := work(); err != nil {
			report(err)
		}
	}()

	return nil
}

// runAll owns the goroutines it starts: it waits for all of them and returns their errors joined, in task order.
func runAll(tasks ...func() error) error {
	errs := make([]error, len(tasks))

	var wg sync.WaitGroup

	for i, task := range tasks {
		wg.Go(func() { errs[i] = task() })
	}

	wg.Wait()

	return errors.Join(errs...)
}
