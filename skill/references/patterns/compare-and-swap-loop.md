# Compare-and-Swap Loop

Update a single shared value from its old value without making other writers wait.

- **Also known as:** Optimistic update, Read-modify-write loop
- **Purpose:** Coordination
- **Scale:** Idiom
- **Mechanisms:** Atomic
- **Concerns:** Data race safety, Throughput
- **Concepts:** [Memory Visibility](../concepts/memory-visibility.md), [Cooperative Cancellation](../concepts/cooperative-cancellation.md)
- **Code:** [`patterns/casloop`](https://github.com/johnknl/go-concurrency-patterns/tree/main/patterns/casloop)

## Problem

A shared word must be updated with a value computed from its current value, and only if a condition holds. `Add`
cannot express "only if": the decision depends on the current value, which may change between the read and the
write. A mutex costs more than one word needs.

## Mechanism

`CompareAndSwap(old, new)` writes `new` only if the value still equals `old`, and reports whether it did. A failed swap
means someone else won; the loop reloads and tries again, so no update is lost. A successful swap publishes the new
value to every later load that observes it ([an atomic store happens before the load that sees
it](../concepts/memory-visibility.md#atomic-store-load)). In this example the word holds the largest value reported.

```go
func storeMax(v *atomic.Int64, x int64) {
	for {
		old := v.Load()
		if x <= old {
			return
		}
		if v.CompareAndSwap(old, x) {
			return
		}
	}
}

func main() {
	var peak atomic.Int64
	var wg sync.WaitGroup

	for i := range 100 {
		wg.Go(func() { storeMax(&peak, int64(i)) })
	}

	wg.Wait()
	fmt.Println(peak.Load())

	// Output:
	// 99
}
```

## Applicability

Use it for one word or pointer updated by a short rule on its current value, under low to moderate contention. Use a
`sync.Mutex` when several values that do not fit behind one pointer change together, or the update has side effects.

## Consequences

- The update may run several times, so it must be pure: no I/O, sends, or shared writes
  ([duplicated work](../concepts/cooperative-cancellation.md#duplicates-idempotent)).
- Under heavy contention retries burn CPU; a mutex, whose waiters sleep, can then be faster.
- There is no fairness: a goroutine can lose the race without bound.
- ABA: a change from A to B and back to A goes unnoticed. For numbers that is harmless when the update depends
  only on the value; for pointers it matters when node identity carries meaning, as in lock-free stacks. Freshly
  allocated values behind `atomic.Pointer[T]` avoid it, since the garbage collector does not reuse memory the loop
  still points to; recycling objects (`sync.Pool`, a free list) brings it back.

## Variants

### Refuse an update

The loop can decide not to swap. In this example a counter stops at a hard limit and reports the refusal.

```go
func tryIncrement(n *atomic.Int64, limit int64) bool {
	for {
		old := n.Load()
		if old >= limit {
			return false
		}
		if n.CompareAndSwap(old, old+1) {
			return true
		}
	}
}

func main() {
	var used, granted atomic.Int64
	var wg sync.WaitGroup

	for range 50 {
		wg.Go(func() {
			if tryIncrement(&used, 10) {
				granted.Add(1)
			}
		})
	}

	wg.Wait()
	fmt.Println(used.Load(), granted.Load())

	// Output:
	// 10 10
}
```

### Plain atomic operation

Without a decision, `Add`, `Swap` or `Store` updates in one step and never retries.

```go
func main() {
	var hits atomic.Int64
	var wg sync.WaitGroup

	for range 100 {
		wg.Go(func() { hits.Add(1) })
	}

	wg.Wait()
	fmt.Println(hits.Load())

	// Output:
	// 100
}
```

### Several fields behind one pointer

Swap a pointer to a new copy of an immutable struct. Every update allocates. `record` expects an initial value
to be stored first; a nil pointer would panic.

```go
// stats is immutable once published.
type stats struct {
	count, sum int64
}

func record(p *atomic.Pointer[stats], x int64) {
	for {
		old := p.Load()
		next := &stats{count: old.count + 1, sum: old.sum + x}
		if p.CompareAndSwap(old, next) {
			return
		}
	}
}

func main() {
	var p atomic.Pointer[stats]
	p.Store(&stats{})

	var wg sync.WaitGroup
	for i := range 10 {
		wg.Go(func() { record(&p, int64(i)) })
	}

	wg.Wait()
	s := p.Load()
	fmt.Println(s.count, s.sum)

	// Output:
	// 10 45
}
```

## Related patterns

- [Copy-on-Write Snapshot](copy-on-write-snapshot.md) publishes larger read-mostly values the same way.
- [Monitor](monitor.md) guards several values with a mutex.
- [Lazy Initialization](../stdlib/lazy-initialization.md) covers one-time setup.

## Anti-patterns

- [Channel Where a Mutex Would Do](../anti-patterns/channel-instead-of-mutex.md): Guard plain state with an owner goroutine and request and reply channels, where a mutex or an atomic next to the state would do, so every call pays a round trip and the state gains a goroutine whose lifetime someone has to manage.
- [Double-Checked Locking](../anti-patterns/double-checked-locking.md): Test a lazily initialized pointer without synchronization and lock only when it is nil, so readers race with the writer and can see a pointer to an incompletely initialized value.

## Sources

- [Herlihy and Shavit (2012)](../sources.md#source-herlihy-shavit-2012)
- [The Go Authors (n.d.f)](../sources.md#source-go-sync-atomic)
