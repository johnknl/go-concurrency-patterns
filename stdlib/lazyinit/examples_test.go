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

package lazyinit

import (
	"fmt"
	"sync"
)

func Example_once() {
	var wg sync.WaitGroup

	names := make([]string, 50)

	for i := range names {
		wg.Go(func() { names[i] = loadConfig().name })
	}

	wg.Wait()
	fmt.Println(names[0], names[49], builds)

	// Output:
	// production production 1
}

func Example_onceValue() {
	calls := 0
	table := sync.OnceValue(func() []int {
		calls++

		squares := make([]int, 10)
		for i := range squares {
			squares[i] = i * i
		}

		return squares
	})

	var wg sync.WaitGroup

	got := make([]int, 10)

	for i := range got {
		wg.Go(func() { got[i] = table()[i] })
	}

	wg.Wait()
	fmt.Println(got, calls)

	// Output:
	// [0 1 4 9 16 25 36 49 64 81] 1
}

func Example_onceValuesError() {
	open := sync.OnceValues(func() (string, error) {
		return "", fmt.Errorf("config file missing")
	})

	for range 2 {
		_, err := open()
		fmt.Println(err)
	}

	// Output:
	// config file missing
	// config file missing
}

func Example_panic() {
	calls := 0

	var once sync.Once

	run := func() (recovered any) {
		defer func() { recovered = recover() }()

		once.Do(func() {
			calls++
			panic("boom")
		})

		return nil
	}

	fmt.Println(run(), run(), calls)

	f := sync.OnceFunc(func() {
		calls++
		panic("boom")
	})

	call := func() (recovered any) {
		defer func() { recovered = recover() }()

		f()

		return nil
	}

	fmt.Println(call(), call(), calls)

	// Output:
	// boom <nil> 1
	// boom boom 2
}
