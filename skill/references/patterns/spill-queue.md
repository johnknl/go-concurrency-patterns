# Spill Queue

Absorb a backlog that briefly exceeds what memory should hold, without making the producer wait for the consumer and without dropping items.

- **Also known as:** Paging, Overflow to disk, Tiered buffer
- **Purpose:** Data flow
- **Scale:** Pattern
- **Mechanisms:** Sync
- **Concerns:** Backpressure, Ordering
- **Concepts:** [Overload Policy](../concepts/overload-policy.md), [Backpressure](../concepts/backpressure.md), [Ordering](../concepts/ordering.md)
- **Code:** [`patterns/spill`](https://github.com/johnknl/go-concurrency-patterns/tree/main/patterns/spill)

## Problem

A producer that should not wait for its consumer, such as a network reader or a callback from a library, feeds a
consumer that is sometimes slower. A [Bounded Queue](bounded-queue.md) with room for the worst burst holds too much memory; a smaller
one forces a choice between blocking the producer and [dropping items](load-shedding.md). Disk is cheaper and larger
than memory, but a queue that lives entirely on disk pays for I/O on every item, even when the consumer keeps up.

## Mechanism

The queue is a FIFO in two tiers: a bounded slice in memory for its head, and a file on disk for its tail. Each
record in the file is a length prefix followed by the item, appended at a write offset and read back from a read
offset. An item goes to memory while memory has room and the file is empty. Once anything is on disk, new items go
to disk too, even if memory has room again; otherwise they would overtake the items on disk. When memory runs
empty, `pop` refills it from the file, oldest first. When the file holds no more records, it is truncated to zero
and both offsets reset, so the file does not grow forever.

The disk is bounded as well (`maxDisk`): when the file is full, `push` returns `errDiskFull`, and the caller is
back at the [overload policy](../concepts/overload-policy.md#every-limit) it tried to postpone: block, drop, or
reject. A spill queue makes the buffer larger, and a larger buffer [only delays the
choice](../concepts/overload-policy.md#buffer-delays); with a consumer that is slower on average, it fills anyway,
and only [backpressure](../concepts/backpressure.md#slowest-stage) or shedding helps.

```go
type spillQueue struct {
	file     *os.File  // the tail of the queue, behind mem
	cond     sync.Cond // signaled when an item arrives or the queue closes
	mem      []string  // the head of the queue, at most memCap items
	memCap   int
	onDisk   int   // records in the file, all of them behind mem
	readOff  int64 // offset of the oldest record still in the file
	writeOff int64 // offset where the next record goes
	maxDisk  int64
	spilled  int // items that went through the file, for the example
	mu       sync.Mutex
	closed   bool
}

func openSpillQueue(dir string, memCap int, maxDisk int64) (*spillQueue, error) {
	f, err := os.CreateTemp(dir, "spill-*")
	if err != nil {
		return nil, err
	}

	q := &spillQueue{file: f, memCap: memCap, maxDisk: maxDisk}
	q.cond.L = &q.mu

	return q, nil
}

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

func (q *spillQueue) close() error {
	q.mu.Lock()
	defer q.mu.Unlock()

	if q.closed {
		return nil
	}

	q.closed = true
	q.mem = nil
	q.cond.Broadcast()

	return errors.Join(q.file.Close(), os.Remove(q.file.Name()))
}
```

- **Order across tiers.** Memory always holds the oldest items and the file the newer ones, and the file is read in
  sequence, so the queue is one FIFO: [items from one producer leave in the order they
  entered](../concepts/ordering.md#sequential-fifo).
- **The producer does not wait for the consumer, but it is not free of waiting.** It waits for the mutex and, once
  memory is full, for a write to the file, so the latency of a push includes disk latency, and a refill by `pop`
  holds the mutex while it reads. That is admission to a bounded queue, not non-blocking execution; a producer that
  must never wait on I/O needs the write on another goroutine, which is a different design.
- **Opening a queue deletes nothing.** It creates one new file with a unique name in a directory that must exist,
  and `close` removes only that file. A file left behind by a process that did not close its queue stays where it
  is, and the new queue starts empty.

A mutex and one `sync.Cond` are the simplest structure here: `push` and `pop` are short critical sections, `pop`
waits on the condition for an item or for `close`, and the file and its offsets are only touched under the mutex. An owner goroutine with an input and an output channel would serve too, but it adds a
goroutine to start and stop, and the queue has no other work to do. The disk I/O does happen under the lock, which
stalls the producer during a refill; that is acceptable for a local file, and the price of keeping the order simple.

**Five items, room for two.** The first two items stay in memory; the other three go through the disk and come
back in order.

```go
func main() {
	tmp, _ := os.MkdirTemp("", "spill")
	defer func() { _ = os.RemoveAll(tmp) }()

	q, err := openSpillQueue(tmp, 2, 1<<20)
	if err != nil {
		fmt.Println(err)

		return
	}
	defer func() { _ = q.close() }()

	for _, item := range []string{"a", "b", "c", "d", "e"} {
		if err := q.push(item); err != nil {
			fmt.Println(err)
		}
	}

	var got []string
	for range 5 {
		item, _, _ := q.pop()
		got = append(got, item)
	}

	fmt.Println(got)
	fmt.Println("through disk:", q.spilled)

	// Output:
	// [a b c d e]
	// through disk: 3
}
```

## Applicability

Use it when bursts are occasional and larger than memory should hold, and the producer cannot wait. When the
consumer is slower on average, no buffer is large enough: use backpressure or [Load Shedding](load-shedding.md).

## Consequences

- Memory use is bounded by `memCap`, and disk use by `maxDisk`; neither the producer nor the consumer chooses
  which tier an item takes.
- Without a backlog, items stay in memory and the disk costs nothing.
- Once a backlog spills, every item until it drains takes the slow path, including items that would fit in memory.
- At the disk limit the producer gets an error, so the overload decision is postponed, not removed.

## Related patterns

- [Bounded Queue](bounded-queue.md) is the in-memory tier alone, with blocking as its policy at the limit.
- [Load Shedding](load-shedding.md) is the policy to apply when the disk is full too.

## Anti-patterns

- [Buffer as a Band-Aid](../anti-patterns/oversized-buffer.md): Hide a producer that outruns its consumer behind a huge channel buffer, so the failure comes later and larger.
- [Goroutine as a Buffer](../anti-patterns/goroutine-as-buffer.md): Park a goroutine on a channel send so that the caller never blocks, which makes every pending value cost a goroutine instead of a channel slot.

## Sources

Original work.
