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

package spill

// Mutants that survive on purpose:
// - none.

import (
	"errors"
	"fmt"
	"os"
	"strconv"
	"sync"
	"testing"
)

func newQueue(t *testing.T, memCap int, maxDisk int64) *spillQueue {
	t.Helper()

	q, err := openSpillQueue(t.TempDir(), memCap, maxDisk)
	if err != nil {
		t.Fatal(err)
	}

	t.Cleanup(func() { _ = q.close() })

	return q
}

func mustPop(t *testing.T, q *spillQueue) string {
	t.Helper()

	item, ok, err := q.pop()
	if err != nil || !ok {
		t.Fatalf("pop: %q %v %v", item, ok, err)
	}

	return item
}

func TestOrderAcrossTiers(t *testing.T) {
	q := newQueue(t, 3, 1<<20)

	next := 0
	push := func(n int) {
		for range n {
			if err := q.push(strconv.Itoa(next)); err != nil {
				t.Fatal(err)
			}
			next++
		}
	}

	want := 0
	pop := func(n int) {
		for range n {
			if got := mustPop(t, q); got != strconv.Itoa(want) {
				t.Fatalf("got %s, want %d", got, want)
			}
			want++
		}
	}

	push(10) // 3 in memory, 7 on disk
	pop(2)   // memory has room again, but the disk is not empty
	push(2)  // so these go to disk
	pop(1)   // memory empty
	pop(4)   // refills of 3 from disk
	push(1)
	pop(6)

	if q.spilled != 10 {
		t.Fatalf("spilled = %d, want 10", q.spilled)
	}

	if q.onDisk != 0 || len(q.mem) != 0 {
		t.Fatalf("not empty: %d on disk, %d in memory", q.onDisk, len(q.mem))
	}

	// Empty again: the next items stay in memory.
	push(3)

	if q.spilled != 10 {
		t.Fatalf("spilled = %d after drain, want 10", q.spilled)
	}

	pop(3)
}

func TestDrainedFileIsEmptied(t *testing.T) {
	q := newQueue(t, 1, 1<<20)

	for i := range 5 {
		if err := q.push(strconv.Itoa(i)); err != nil {
			t.Fatal(err)
		}
	}

	size := func() int64 {
		t.Helper()

		info, err := q.file.Stat()
		if err != nil {
			t.Fatal(err)
		}

		return info.Size()
	}

	full := size()
	if full != 4*(lenSize+1) {
		t.Fatalf("size = %d with 4 records", full)
	}

	mustPop(t, q)
	mustPop(t, q) // refills one record from disk; three remain

	if got := size(); got != full || q.readOff != lenSize+1 {
		t.Fatalf("size = %d, readOff = %d, want %d and %d", got, q.readOff, full, lenSize+1)
	}

	for i := 2; i < 5; i++ {
		if got := mustPop(t, q); got != strconv.Itoa(i) {
			t.Fatalf("got %s, want %d", got, i)
		}
	}

	if got := size(); got != 0 || q.readOff != 0 || q.writeOff != 0 {
		t.Fatalf("size = %d, offsets %d %d after draining, want 0", got, q.readOff, q.writeOff)
	}
}

func TestDiskFull(t *testing.T) {
	q := newQueue(t, 2, 16<<10)

	payload := string(make([]byte, 1000))

	var (
		pushed int
		err    error
	)

	for pushed = 0; pushed < 1000; pushed++ {
		if err = q.push(fmt.Sprint(pushed, payload)); err != nil {
			break
		}
	}

	if !errors.Is(err, errDiskFull) {
		t.Fatalf("err = %v after %d pushes, want errDiskFull", err, pushed)
	}

	// The rejected item was not queued, and everything before it is intact.
	for i := range pushed {
		if got := mustPop(t, q); got != fmt.Sprint(i, payload) {
			t.Fatalf("item %d: got %.10q", i, got)
		}
	}

	// Popping freed the disk.
	if err := q.push("after"); err != nil {
		t.Fatal(err)
	}

	if got := mustPop(t, q); got != "after" {
		t.Fatalf("got %q", got)
	}
}

func TestStartsEmpty(t *testing.T) {
	dir := t.TempDir()

	q, err := openSpillQueue(dir, 1, 1<<20)
	if err != nil {
		t.Fatal(err)
	}

	for _, item := range []string{"a", "b", "c"} {
		if err = q.push(item); err != nil {
			t.Fatal(err)
		}
	}

	// A crash: the process loses the queue without closing it, and its file stays behind.
	stale := q.file.Name()
	if err = q.file.Close(); err != nil {
		t.Fatal(err)
	}

	q, err = openSpillQueue(dir, 1, 1<<20)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = q.close() }()

	if err := q.push("x"); err != nil {
		t.Fatal(err)
	}

	if got := mustPop(t, q); got != "x" {
		t.Fatalf("got %q, want the queue to start empty", got)
	}

	if _, err := os.Stat(stale); err != nil {
		t.Fatalf("opening a queue deleted an existing file: %v", err)
	}
}

func TestClose(t *testing.T) {
	dir := t.TempDir()

	q, err := openSpillQueue(dir, 1, 1<<20)
	if err != nil {
		t.Fatal(err)
	}

	name := q.file.Name()

	waiting := make(chan bool)

	go func() {
		_, ok, _ := q.pop()
		waiting <- ok
	}()

	if err := q.close(); err != nil {
		t.Fatal(err)
	}

	if ok := <-waiting; ok {
		t.Fatal("pop on a closed queue returned an item")
	}

	if _, err := os.Stat(name); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("spill file still there: %v", err)
	}

	if _, err := os.Stat(dir); err != nil {
		t.Fatalf("close removed more than its own file: %v", err)
	}

	if err := q.push("a"); !errors.Is(err, errClosed) {
		t.Fatalf("push after close: %v", err)
	}

	if _, ok, _ := q.pop(); ok {
		t.Fatal("pop after close returned an item")
	}

	if err := q.close(); err != nil {
		t.Fatalf("second close: %v", err)
	}
}

func TestCloseDiscardsItems(t *testing.T) {
	q := newQueue(t, 1, 1<<20)

	_ = q.push("a")
	_ = q.push("b")
	_ = q.close()

	if _, ok, _ := q.pop(); ok {
		t.Fatal("pop after close returned an item")
	}
}

func TestConcurrent(t *testing.T) {
	const producers, perProducer = 4, 200

	q := newQueue(t, 8, 8<<20)

	var wg sync.WaitGroup
	for p := range producers {
		wg.Go(func() {
			for i := range perProducer {
				if err := q.push(fmt.Sprintf("%d %d", p, i)); err != nil {
					t.Error(err)
				}
			}
		})
	}

	// Each producer's items arrive in its own order.
	next := make([]int, producers)
	for range producers * perProducer {
		var p, i int
		if _, err := fmt.Sscanf(mustPop(t, q), "%d %d", &p, &i); err != nil {
			t.Fatal(err)
		}

		if i != next[p] {
			t.Fatalf("producer %d: got %d, want %d", p, i, next[p])
		}

		next[p]++
	}

	wg.Wait()
}
