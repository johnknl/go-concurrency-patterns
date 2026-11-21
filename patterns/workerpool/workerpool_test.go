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

// Mutants that survive on purpose: none.

package workerpool

import "testing"

func TestProcess(t *testing.T) {
	tests := []struct {
		id      int
		want    int
		wantErr bool
	}{
		{0, 0, true},
		{1, 10, false},
		{2, 20, false},
		{3, 30, false},
		{4, 0, true},
		{6, 60, false},
		{8, 0, true},
	}

	for _, tt := range tests {
		out, err := process(tt.id)
		if (err != nil) != tt.wantErr || out != tt.want {
			t.Errorf("process(%d) = %d, %v; want %d, error %v", tt.id, out, err, tt.want, tt.wantErr)
		}
	}
}
