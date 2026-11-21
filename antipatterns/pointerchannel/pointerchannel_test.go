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

// Mutants that survive on purpose:
//   - buffer-channel pointerchannel.go:31-34 and 54-57 (ch, ack, again, res buffered): every hand-off is followed by
//     a receive that waits for the other side, so the order of all accesses is the same with or without buffers.

package pointerchannel

import (
	"testing"
	"testing/synctest"
)

func TestBadShare(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		first, second := badShare()
		if first != "draft" || second != "edited" {
			t.Fatalf("badShare = (%q, %q), want the receiver to see the sender's edit", first, second)
		}
	})
}

func TestCopyShare(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		first, second := copyShare()
		if first != "draft" || second != "draft" {
			t.Fatalf("copyShare = (%q, %q), want the receiver's copy unchanged", first, second)
		}
	})
}
