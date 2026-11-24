# Unbounded Reorder Buffer

Hold everything that arrives behind a missing item until it arrives, with no limit, so one slow or lost item stops all output while the buffer grows.

- **Violates:** [Bound everything](../principles.md#principle-bounded)
- **Purpose:** Data flow
- **Mechanisms:** Channels
- **Concerns:** Ordering, Overload
- **Breaks:** [Ordering](../concepts/ordering.md)
- **Code:** [`antipatterns/reorderbuffer`](https://github.com/johnknl/go-concurrency-patterns/tree/main/antipatterns/reorderbuffer)

## Symptoms

- A stage restores order by keeping early results in a map until their predecessors arrive, with no limit on how
  long it waits or how much it holds.
- Output stops while producers keep working at normal throughput.
- Memory of the ordering stage grows steadily until the process is killed.
- At shutdown, computed results are lost because they were still behind a gap.

## Example

**Faulty**

`reorderUnbounded` emits in order and buffers everything else. Item 0 never arrives, so nothing is emitted and
the buffer holds every later item.

```go
// reorderUnbounded emits items in sequence order and buffers everything that arrives ahead of a gap, with no
// limit. When an item never arrives, nothing after it is emitted, and the buffer grows by one for every item that
// does. It returns how many items were still buffered when the input ended.
func reorderUnbounded(in <-chan item, emit func(item)) int {
	pending := map[int]item{}
	next := 0

	for it := range in {
		pending[it.seq] = it

		for {
			v, ok := pending[next]
			if !ok {
				break
			}

			delete(pending, next)
			emit(v)
			next++
		}
	}

	return len(pending)
}

func main() {
	for _, n := range []int{10, 100, 1000} {
		in := make(chan item, n)
		for seq := 1; seq <= n; seq++ { // item 0 never arrives
			in <- item{seq: seq}
		}
		close(in)

		emitted := 0
		buffered := reorderUnbounded(in, func(item) { emitted++ })
		fmt.Printf("after %d items: emitted %d, buffered %d\n", n, emitted, buffered)
	}

	// Output:
	// after 10 items: emitted 0, buffered 10
	// after 100 items: emitted 0, buffered 100
	// after 1000 items: emitted 0, buffered 1000
}
```

**Fixed**

`reorderBounded` holds at most `limit` items behind a gap; one more gives up on the gap, reports the missing
sequence number, and moves on. A late missing item or a duplicate is reported and dropped.

```go
// reorderBounded emits items in sequence order and holds at most limit items behind a gap. One more, or the end
// of the input, gives up on the gap: it reports each missing sequence number to skip and moves past it. An item
// that arrives after its gap was given up on, or twice, is reported to late and dropped.
func reorderBounded(in <-chan item, limit int, emit func(item), skip func(seq int), late func(item)) {
	pending := make(map[int]item, limit+1)
	next := 0

	advance := func() {
		for {
			v, ok := pending[next]
			if !ok {
				return
			}

			delete(pending, next)
			emit(v)
			next++
		}
	}

	for it := range in {
		if _, dup := pending[it.seq]; dup || it.seq < next {
			late(it)

			continue
		}

		pending[it.seq] = it
		advance()

		for len(pending) > limit {
			skip(next)
			next++
			advance()
		}
	}

	for len(pending) > 0 {
		skip(next)
		next++
		advance()
	}
}

func main() {
	in := make(chan item, 7)
	for _, seq := range []int{1, 2, 3, 4, 0, 5, 6} { // item 0 arrives after the bound gave up on it
		in <- item{seq: seq, val: fmt.Sprint(seq)}
	}
	close(in)

	reorderBounded(in, 3,
		func(it item) { fmt.Println("emit", it.val) },
		func(seq int) { fmt.Println("skip", seq) },
		func(it item) { fmt.Println("late", it.val) },
	)

	// Output:
	// skip 0
	// emit 1
	// emit 2
	// emit 3
	// emit 4
	// late 0
	// emit 5
	// emit 6
}
```

## Why it goes wrong

A reorder buffer depends on every sequence number arriving exactly once, and a crashed producer, a dropped message or a
skipped number breaks that. The buffer is bounded only by the gap, and the gap only by the producer that owes the
missing item: restoring order costs head-of-line blocking and [needs a bound](../concepts/ordering.md#restore-cost).
More workers do not help: while one is stuck, the others finish later items and grow the buffer.

## Detection

- A map or slice keyed by sequence number, deleted from only when `next` arrives, with no length check or timeout.
- The ordering stage's buffer or memory rising while its output rate is zero.

## Fixed by

- [Resequencer](../patterns/resequencer.md): Consume results in input order when concurrent work delivers them out of order.
- [Watermark](../patterns/watermark.md): Act on events in the order of their timestamps when they arrive from several inputs at different speeds, without guessing how long to wait for the slower ones.

Other policies at the bound: fail, as the Resequencer does, or limit items in flight at the source so the buffer
cannot overflow. To bound by time instead of count, a Watermark or a timeout decides when to give up on the gap.

## Sources

Original work.
