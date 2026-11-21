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

package main

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

const fixture = `package p

import (
	"context"
	"sync"
)

type box struct {
	mu sync.Mutex
	n  int
}

func (b *box) inc() {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.n++
}

func gen(ctx context.Context, ok bool) <-chan int {
	out := make(chan int)
	buf := make(chan int, 4)
	var wg sync.WaitGroup
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	go func() {
		defer close(out)
		if ok {
			select {
			case out <- 1:
			case <-ctx.Done():
			default:
			}
		}
	}()

	wg.Wait()
	_ = buf

	return out
}
`

func TestMutantsOf(t *testing.T) {
	file := filepath.Join(t.TempDir(), "p.go")
	if err := os.WriteFile(file, []byte(fixture), 0o600); err != nil {
		t.Fatal(err)
	}

	ms, err := mutantsOf(file)
	if err != nil {
		t.Fatal(err)
	}

	ops := make([]string, 0, len(ms))
	for _, m := range ms {
		ops = append(ops, m.op)
	}

	slices.Sort(ops)

	want := []string{
		"buffer-channel", "go-to-sync", "negate-condition", "remove-cancel", "remove-close",
		"remove-critical-section", "remove-default", "remove-done-case", "remove-wait", "unbuffer-channel",
	}
	if !slices.Equal(ops, want) {
		t.Fatalf("got %v\nwant %v", ops, want)
	}

	src := []byte(fixture)

	for _, m := range ms {
		got := string(apply(src, m.edits))

		switch m.op {
		case "remove-critical-section":
			if strings.Contains(got, "b.mu.Lock()") || strings.Contains(got, "b.mu.Unlock()") {
				t.Errorf("%s: lock still present:\n%s", m.op, got)
			}
		case "negate-condition":
			if !strings.Contains(got, "if !(ok) {") {
				t.Errorf("%s: condition not negated:\n%s", m.op, got)
			}
		case "unbuffer-channel":
			if !strings.Contains(got, "buf := make(chan int)") {
				t.Errorf("%s: buffer still present:\n%s", m.op, got)
			}
		case "buffer-channel":
			if !strings.Contains(got, "out := make(chan int, 1)") {
				t.Errorf("%s: no buffer added:\n%s", m.op, got)
			}
		case "go-to-sync":
			if strings.Contains(got, "go func()") {
				t.Errorf("%s: go statement still present:\n%s", m.op, got)
			}
		}
	}
}
