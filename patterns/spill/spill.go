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

import (
	"encoding/binary"
	"errors"
	"io"
	"os"
	"sync"
)

var (
	errClosed   = errors.New("queue closed")
	errDiskFull = errors.New("spill file full")
)

const lenSize = 4 // bytes in the length prefix of a record

// spillQueue is a FIFO whose first memCap items are in memory and whose rest is in a file on disk.
type spillQueue struct {
	file     *os.File  // the tail of the queue, behind mem
	cond     sync.Cond // signaled when an item arrives or the queue closes
	dir      string
	mem      []string // the head of the queue, at most memCap items
	memCap   int
	onDisk   int   // records in the file, all of them behind mem
	readOff  int64 // offset of the oldest record still in the file
	writeOff int64 // offset where the next record goes
	maxDisk  int64
	spilled  int // items that went through the file, for the example
	mu       sync.Mutex
	closed   bool
}

// openSpillQueue starts an empty queue with its spill file in dir.
func openSpillQueue(dir string, memCap int, maxDisk int64) (*spillQueue, error) {
	if err := os.RemoveAll(dir); err != nil {
		return nil, err
	}

	if err := os.MkdirAll(dir, 0o700); err != nil {
		return nil, err
	}

	f, err := os.CreateTemp(dir, "spill-*")
	if err != nil {
		return nil, err
	}

	q := &spillQueue{file: f, dir: dir, memCap: memCap, maxDisk: maxDisk}
	q.cond.L = &q.mu

	return q, nil
}

// push adds an item without blocking. It fails with errDiskFull when the disk tier is full too.
func (q *spillQueue) push(item string) error {
	q.mu.Lock()
	defer q.mu.Unlock()

	if q.closed {
		return errClosed
	}

	// Once anything is on disk, new items go to disk too, or they would overtake it.
	if q.onDisk == 0 && len(q.mem) < q.memCap {
		q.mem = append(q.mem, item)
	} else {
		if err := q.write(item); err != nil {
			return err
		}

		q.onDisk++
		q.spilled++
	}

	q.cond.Signal()

	return nil
}

// write appends one record, a length prefix and the item, at the end of the file.
func (q *spillQueue) write(item string) error {
	n := int64(lenSize + len(item))
	if q.writeOff+n > q.maxDisk {
		return errDiskFull
	}

	rec := binary.BigEndian.AppendUint32(make([]byte, 0, n), uint32(len(item))) //nolint:gosec // items are small
	rec = append(rec, item...)

	if _, err := q.file.WriteAt(rec, q.writeOff); err != nil {
		return err
	}

	q.writeOff += n

	return nil
}

// pop removes the oldest item, waiting for one. It returns false once the queue is closed.
func (q *spillQueue) pop() (string, bool, error) {
	q.mu.Lock()
	defer q.mu.Unlock()

	for len(q.mem) == 0 && q.onDisk == 0 && !q.closed {
		q.cond.Wait()
	}

	if q.closed {
		return "", false, nil
	}

	if len(q.mem) == 0 {
		if err := q.refill(); err != nil {
			return "", false, err
		}
	}

	item := q.mem[0]
	q.mem = q.mem[1:]

	return item, true, nil
}

// refill moves the oldest records from the file to memory, in order. Once the file holds no more records, it is
// emptied, so that its space is reclaimed.
func (q *spillQueue) refill() error {
	r := io.NewSectionReader(q.file, q.readOff, q.writeOff-q.readOff)

	var prefix [lenSize]byte

	for range min(q.memCap, q.onDisk) {
		if _, err := io.ReadFull(r, prefix[:]); err != nil {
			return err
		}

		payload := make([]byte, binary.BigEndian.Uint32(prefix[:]))
		if _, err := io.ReadFull(r, payload); err != nil {
			return err
		}

		q.mem = append(q.mem, string(payload))
		q.onDisk--
		q.readOff += int64(lenSize + len(payload))
	}

	if q.onDisk > 0 {
		return nil
	}

	q.readOff, q.writeOff = 0, 0

	return q.file.Truncate(0)
}

// close discards the queue's contents, wakes waiting consumers, and deletes the file.
func (q *spillQueue) close() error {
	q.mu.Lock()
	defer q.mu.Unlock()

	if q.closed {
		return nil
	}

	q.closed = true
	q.mem = nil
	q.cond.Broadcast()

	return errors.Join(q.file.Close(), os.RemoveAll(q.dir))
}
