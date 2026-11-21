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

package pointerchannel

type note struct{ text string }

// badShare sends a pointer and then edits the note it points to. The
// receiver reads the note before and after the edit. The acknowledgments
// order every access, so there is no data race, but the receiver sees the edit.
func badShare() (first, second string) {
	ch := make(chan *note)
	ack := make(chan struct{})
	again := make(chan struct{})
	res := make(chan [2]string)
	go func() {
		p := <-ch
		a := p.text
		ack <- struct{}{}
		<-again
		res <- [2]string{a, p.text}
	}()
	n := &note{"draft"}
	ch <- n
	<-ack
	n.text = "edited"
	again <- struct{}{}
	r := <-res
	return r[0], r[1]
}

// copyShare sends the note by value. The receiver has its own copy, so the
// same edit by the sender is not visible to it.
func copyShare() (first, second string) {
	ch := make(chan note)
	ack := make(chan struct{})
	again := make(chan struct{})
	res := make(chan [2]string)
	go func() {
		v := <-ch
		a := v.text
		ack <- struct{}{}
		<-again
		res <- [2]string{a, v.text}
	}()
	n := note{"draft"}
	ch <- n
	<-ack
	n.text = "edited"
	again <- struct{}{}
	r := <-res
	return r[0], r[1]
}
