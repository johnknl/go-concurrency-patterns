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

package contextprop

import (
	"context"
	"fmt"
)

// requestKey is an unexported type, so no other package can collide with or read this key directly.
type requestKey struct{}

// withRequestID attaches a request-scoped value.
func withRequestID(parent context.Context, id string) context.Context {
	return context.WithValue(parent, requestKey{}, id)
}

// requestID reads the value back.
func requestID(c context.Context) (string, bool) {
	id, ok := c.Value(requestKey{}).(string)

	return id, ok
}

// worker blocks until its context is done and reports why.
func worker(ctx context.Context, name string, result chan<- string) {
	<-ctx.Done()

	result <- fmt.Sprintf("%s: %v", name, ctx.Err())
}

// call stands for a function lower in the call tree that blocks until its context ends and reports the cause.
func call(ctx context.Context, causes chan<- error) {
	<-ctx.Done()

	causes <- context.Cause(ctx)
}
