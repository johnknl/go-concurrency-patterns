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

package nondetstate

import "fmt"

func Example_replayDiverges() {
	defer func(f func() bool) { attemptSucceeds = f }(attemptSucceeds)
	log := []badEvent{"start", "attempt"}

	attemptSucceeds = func() bool { return true } // the live run
	fmt.Println("live:  ", badReplay(log))

	attemptSucceeds = func() bool { return false } // the replay draws again
	fmt.Println("replay:", badReplay(log))
	// Output:
	// live:   done
	// replay: failed
}

func Example_replayIdentical() {
	// The live run decides the outcome once and records it in the event.
	log := []event{{kind: "start"}, attempt(func() bool { return true })}
	fmt.Println("live:  ", replay(log))
	fmt.Println("replay:", replay(log))
	// Output:
	// live:   done
	// replay: done
}
