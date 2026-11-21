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
//   - buffer-channel resulterr.go:71: lookupQuietly has no observable side effect that a buffer could reorder;
//     the page promises nothing about how far it runs ahead.

package resulterr

import (
	"context"
	"errors"
	"slices"
	"testing"
	"testing/synctest"
)

func TestLookup(t *testing.T) {
	if v, err := lookup(3); v != "record-3" || err != nil {
		t.Fatalf("lookup(3) = (%q, %v), want (\"record-3\", nil)", v, err)
	}

	if _, err := lookup(4); !errors.Is(err, errNotFound) || err.Error() != "lookup 4: not found" {
		t.Fatalf("lookup(4) error = %v, want it to wrap errNotFound", err)
	}
}

func TestLookupAll(t *testing.T) {
	t.Run("no ids", func(t *testing.T) {
		synctest.Test(t, func(t *testing.T) {
			if _, ok := <-lookupAll(context.Background()); ok {
				t.Fatal("got a result, want the channel closed")
			}
		})
	})

	t.Run("reports every outcome in order, errors included", func(t *testing.T) {
		synctest.Test(t, func(t *testing.T) {
			var got []result
			for r := range lookupAll(context.Background(), 1, 2, 3) {
				got = append(got, r)
			}

			if len(got) != 3 {
				t.Fatalf("got %d results, want 3", len(got))
			}

			for i, r := range got {
				if r.id != i+1 {
					t.Fatalf("result %d has id %d, want %d", i, r.id, i+1)
				}
			}

			if got[0].value != "record-1" || got[0].err != nil || got[2].value != "record-3" || got[2].err != nil {
				t.Fatalf("odd ids: got %+v and %+v, want values without errors", got[0], got[2])
			}

			if got[1].value != "" || !errors.Is(got[1].err, errNotFound) {
				t.Fatalf("id 2: got %+v, want errNotFound", got[1])
			}
		})
	})

	t.Run("cancel stops a producer that waits for its reader", func(t *testing.T) {
		synctest.Test(t, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			out := lookupAll(ctx, 1, 3, 5)

			if r := <-out; r.id != 1 {
				t.Fatalf("got id %d, want 1", r.id)
			}

			cancel()
			synctest.Wait() // the producer has seen the cancel and returned

			if r, ok := <-out; ok {
				t.Fatalf("got %+v after cancel, want the channel closed", r)
			}
		})
	})

	t.Run("an already canceled context delivers nothing", func(t *testing.T) {
		synctest.Test(t, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			cancel()

			out := lookupAll(ctx, 1, 3)
			synctest.Wait()

			if _, ok := <-out; ok {
				t.Fatal("got a result, want the channel closed")
			}
		})
	})
}

func TestLookupQuietly(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		var logs []string

		var got []string
		for v := range lookupQuietly(&logs, 1, 2, 3, 4) {
			got = append(got, v)
		}

		if want := []string{"record-1", "record-3"}; !slices.Equal(got, want) {
			t.Fatalf("values %v, want %v", got, want)
		}

		if want := []string{"lookup 2: not found", "lookup 4: not found"}; !slices.Equal(logs, want) {
			t.Fatalf("logs %v, want %v", logs, want)
		}
	})
}
