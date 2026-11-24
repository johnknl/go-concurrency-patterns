# Group Commit

Let concurrent callers share one execution of an expensive operation, without adding latency when only one caller is waiting.

- **Also known as:** Commit group, Opportunistic batching, Smart batching
- **Purpose:** Execution
- **Scale:** Pattern
- **Mechanisms:** Sync
- **Concerns:** Batching, Throughput, Latency
- **Code:** [`patterns/groupcommit`](https://github.com/johnknl/go-concurrency-patterns/tree/main/patterns/groupcommit)

## Problem

Some operations cost about the same for one item as for many. When many goroutines each need one and wait for its
result, running it once per caller caps throughput at the operation's rate. Collecting items for a fixed time forms
groups, but delays every caller, including a lone one on an idle system.

## Mechanism

A caller appends its item to the open batch under a mutex. If no flush is running, it leads: it takes the batch,
flushes it outside the lock, stores the result in it, and wakes the waiters. Otherwise it waits in the batch that
the next leader takes. There is no goroutine and no timer: batches form only from callers that arrive while a
flush runs. The classic case, and this example, is a write-ahead log, where an fsync of ten records costs about as
much as one. Flat combining is the same idea for operations on a shared data structure.

```go
type batch struct {
	err   error
	items []string
	done  bool
}

type committer struct {
	flush    func([]string) error
	open     *batch
	cond     sync.Cond
	maxBatch int
	mu       sync.Mutex
	flushing bool
}

func newCommitter(maxBatch int, flush func([]string) error) *committer {
	c := &committer{flush: flush, maxBatch: maxBatch, open: &batch{}}
	c.cond.L = &c.mu

	return c
}

func (c *committer) commit(item string) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	for len(c.open.items) >= c.maxBatch {
		c.cond.Wait() // the open batch is full: wait until it has been taken
	}

	b := c.open
	b.items = append(b.items, item)

	for c.flushing && !b.done {
		c.cond.Wait()
	}

	if b.done {
		return b.err // another caller led this batch
	}

	// Nobody is flushing: lead this batch, and only this one.
	c.flushing = true
	c.open = &batch{}
	c.mu.Unlock()

	err := c.flush(b.items)

	c.mu.Lock()
	b.err, b.done = err, true
	c.flushing = false
	c.cond.Broadcast()

	return err
}
```

- **One flush at a time, in order.** Only a caller that sees `flushing` false under the mutex leads, so the flush
  never runs concurrently with itself, and batches are flushed in the order they were opened.
- **Each caller gets its own batch's error,** stored in the batch that every member points to.
- **Fair leadership.** A leader flushes only its own batch; a caller that missed it is in the next one.
- **Bounded batches.** At `maxBatch` items, new callers wait until a leader has taken the batch.

One `sync.Cond` covers the three waits (room in the batch, flusher free, own batch done), which all change at the
end of a flush.

**An idle committer, and callers under load.** An idle committer flushes each item alone, at once. Under load the
grouping depends on the schedule.

```go
func main() {
	c := newCommitter(64, func(items []string) error {
		fmt.Println("flush", items)

		if strings.HasPrefix(items[0], "bad") {
			return errors.New("disk full")
		}

		return nil
	})

	fmt.Println(c.commit("a"))
	fmt.Println(c.commit("bad"))
	fmt.Println(c.commit("b"))

	// Output:
	// flush [a]
	// <nil>
	// flush [bad]
	// disk full
	// flush [b]
	// <nil>
}

func main() {
	var (
		mu      sync.Mutex
		flushed int
		largest int
	)

	c := newCommitter(16, func(items []string) error {
		mu.Lock()
		defer mu.Unlock()

		flushed += len(items)
		largest = max(largest, len(items))

		return nil
	})

	var wg sync.WaitGroup
	for i := range 100 {
		wg.Go(func() { _ = c.commit(fmt.Sprint(i)) })
	}
	wg.Wait()

	fmt.Println("items flushed:", flushed)
	fmt.Println("within bound:", largest <= 16)

	// Output:
	// items flushed: 100
	// within bound: true
}
```

## Applicability

Use it when many goroutines call a synchronous operation whose cost is mostly fixed per execution, each needs the
result for its own item, and the operation accepts a group of items. Hand the items to a [Batcher](batcher.md)
instead when callers do not wait for the result, or when a partial batch must be flushed after a deadline. Grouping
gains nothing when the cost grows linearly with the number of items.

## Consequences

- Throughput rises with load; at low load there is no added latency and no time parameter to tune.
- The API stays synchronous: `commit` returns an error, and the caller starts nothing.
- The worst-case latency is about two flushes: the one running when a caller arrives, and its own.
- If the flush blocks, every caller blocks; bound the flush with a deadline.
- `commit` takes no context: once an item has joined a batch, it is flushed anyway.

## Related patterns

- [Batcher](batcher.md) groups items by size and time in an owner goroutine, for callers that do not wait.
- Request coalescing (`golang.org/x/sync/singleflight`) shares one call among callers that ask for the same thing;
  group commit combines different items.
- [Monitor](monitor.md) and [Condition Variable](../stdlib/condition-variable.md) are the structure it is built on.

## Anti-patterns

- [Batch Without a Time Bound](../anti-patterns/batch-without-time-bound.md): Collect items into batches that are flushed only when full, so at low load an item waits indefinitely and the last partial batch is lost at shutdown.

## Sources

- [DeWitt et al. (1984)](../sources.md#source-dewitt-1984), Section 5.2, "Limits to Transaction Throughput"
- [Hendler et al. (2010)](../sources.md#source-hendler-2010)
