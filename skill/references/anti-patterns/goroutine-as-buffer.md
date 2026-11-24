# Goroutine as a Buffer

Park a goroutine on a channel send so that the caller never blocks, which makes every pending value cost a goroutine instead of a channel slot.

- **Violates:** [Bound everything](../principles.md#principle-bounded)
- **Purpose:** Data flow
- **Mechanisms:** Channels
- **Concerns:** Backpressure, Overload, Leak prevention
- **Breaks:** [Backpressure](../concepts/backpressure.md), [Overload Policy](../concepts/overload-policy.md), [Goroutine Leak Prevention](../concepts/goroutine-leak-prevention.md)
- **Code:** [`antipatterns/goroutinebuffer`](https://github.com/johnknl/go-concurrency-patterns/tree/main/antipatterns/goroutinebuffer)

## Symptoms

- The goroutine count and memory grow with the backlog.
- Values arrive in an order that differs from the order of the calls.
- The producer is never slowed down, so an overload shows up as exhausted memory.

## Example

**Faulty**

`mailbox.Put` never blocks: it writes `go func() { ch <- v }()`. With no consumer, 1000 values are held by 1000
parked goroutines.

```go
type mailbox struct {
	out    chan int
	parked atomic.Int64
}

func (m *mailbox) Put(v int) {
	m.parked.Add(1)

	go func() {
		defer m.parked.Add(-1)

		m.out <- v
	}()
}

func main() {
	m := &mailbox{out: make(chan int)}

	for i := range 1000 {
		m.Put(i)
	}

	fmt.Println("pending values held by goroutines:", m.parked.Load())

	for range 1000 {
		<-m.out
	}

	// Output:
	// pending values held by goroutines: 1000
}
```

**Fixed**

A buffered channel sized to the expected burst, with a decision for when it is full. `queue.Put` sheds the value
and counts it; a plain send would block the caller instead, passing the slowness back to the producer.

```go
type queue struct {
	items   chan int
	dropped int
}

func newQueue(capacity int) *queue {
	return &queue{items: make(chan int, capacity)}
}

func (q *queue) Put(v int) bool {
	select {
	case q.items <- v:
		return true
	default:
		q.dropped++

		return false
	}
}

func main() {
	q := newQueue(3)

	for i := range 5 {
		q.Put(i)
	}

	fmt.Println("queued:", len(q.items), "of", cap(q.items), "dropped:", q.dropped)

	close(q.items)

	for v := range q.items {
		fmt.Println("received:", v)
	}

	// Output:
	// queued: 3 of 3 dropped: 2
	// received: 0
	// received: 1
	// received: 2
}
```

**Lost order.** The language does not specify the order in which blocked senders are served. The current runtime serves
them in the order they blocked, but the order in which they reach the send is up to the scheduler. A gate delays the
goroutine of value 1, and value 2 overtakes it.

```go
func main() {
	out := make(chan int)
	gate := make(chan struct{})

	go func() {
		<-gate

		out <- 1
	}()

	go func() {
		out <- 2
	}()

	fmt.Println("first received:", <-out)

	close(gate)

	fmt.Println("second received:", <-out)

	// Output:
	// first received: 2
	// second received: 1
}
```

## Why it goes wrong

A buffered channel stores a value in a slot of the element's size, allocated once. A parked goroutine needs a stack of
at least 2 KiB, a descriptor, and a closure holding the value: two orders of magnitude more for small elements. The
channel's buffer is also first in, first out by specification, knows its length (`len`), and has a limit (`cap`) at
which the sender must decide: wait ([backpressure](../concepts/backpressure.md#feedback)) or drop ([load
shedding](../patterns/load-shedding.md)). Goroutines have no limit, so [no
feedback](../concepts/backpressure.md#needs-bound) reaches the source and no [overload
policy](../concepts/overload-policy.md#buffer-delays) is ever chosen; the backlog grows until memory runs out, a
costlier form of [Unbounded Goroutines](unbounded-goroutines.md) and [Buffer as a Band-Aid](oversized-buffer.md). When
no receiver comes, every goroutine leaks along with its value, and the caller that started it cannot end it ([goroutine
leak prevention](../concepts/goroutine-leak-prevention.md#owner-ends)).

It usually starts as a way out of a deadlock, such as a goroutine sending on a channel it also receives from. The
goroutine hides the deadlock and keeps its cause: producer and consumer coupled without a queue between them. A
goroutine that does real work before its send is not this anti-pattern.

## Detection

- `runtime.NumGoroutine` or `/sched/goroutines` tracking the backlog, not the number of workers.
- A goroutine profile (`/debug/pprof/goroutine?debug=1`) with thousands of identical stacks ending in a send.
- `/memory/classes/heap/stacks:bytes` growing with the backlog.
- The `goroutineleak` profile reports senders that no running goroutine can reach.
- `go` statements whose body is a single send. `go vet` and the race detector do not report it.

## Fixed by

- [Bounded Queue](../patterns/bounded-queue.md): Let a producer run ahead of its consumer by a fixed, chosen amount.
- [Load Shedding](../patterns/load-shedding.md): Keep the producer and the latency bounded under overload by dropping the work that does not fit.
- [Spill Queue](../patterns/spill-queue.md): Absorb a backlog that briefly exceeds what memory should hold, without making the producer wait for the consumer and without dropping items.

If the caller can neither block nor drop, the backlog has to go somewhere bounded and cheaper than memory: a
[Spill Queue](../patterns/spill-queue.md) keeps order and moves the overflow to disk, which has a limit of its own.

## Sources

Original work.
