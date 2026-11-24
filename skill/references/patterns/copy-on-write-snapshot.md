# Copy-on-Write Snapshot

Let many readers see a consistent version of read-mostly data without waiting for writers or for each other.

- **Also known as:** Copy-on-write, Read-copy-update (RCU), in its simplest form
- **Purpose:** Coordination
- **Scale:** Pattern
- **Mechanisms:** Atomic, Sync
- **Concerns:** Data race safety, Throughput
- **Concepts:** [Memory Visibility](../concepts/memory-visibility.md), [Confinement](../concepts/confinement.md), [Cooperative Cancellation](../concepts/cooperative-cancellation.md)
- **Code:** [`patterns/cowsnapshot`](https://github.com/johnknl/go-concurrency-patterns/tree/main/patterns/cowsnapshot)

## Problem

Shared state is read very often and changed rarely. A `sync.RWMutex` makes every read take a lock, and readers on
many cores contend on the lock's shared state. A reader may also need a consistent view across many reads, even
while a writer publishes a change.

## Mechanism

The current version lives in an `atomic.Pointer`. A reader loads it once and reads everything it needs from that
version, for as long as it needs it. A writer holds a mutex, clones the current value, changes the clone, and stores
it: every change made in one update becomes visible at once, as one new version. The old version is garbage
collected when no reader holds it. A read is one atomic load.

The two rules are what the pattern is for: load once per read, and change only through one update. An API that loads
per field, or publishes per field, lets a reader combine fields from different versions.

In this example the state is a configuration, and a writer moves a service: `host` and `port` change in one
`update`, so every reader sees either the old pair or the new one.

```go
// config is never modified after it is stored in a snapshot pointer.
type config map[string]string

type snapshot struct {
	cur atomic.Pointer[config]
	mu  sync.Mutex
}

func (s *snapshot) load() config {
	return *s.cur.Load()
}

func (s *snapshot) update(fn func(next config)) {
	s.mu.Lock()
	defer s.mu.Unlock()

	next := maps.Clone(*s.cur.Load())
	fn(next)
	s.cur.Store(&next)
}

func main() {
	var s snapshot
	s.cur.Store(&config{"host": "a.example", "port": "80"})

	old := s.load()

	var wg sync.WaitGroup
	for range 4 {
		wg.Go(func() {
			c := s.load() // one load per read: host and port come from the same version
			_ = c["host"] + ":" + c["port"]
		})
	}
	wg.Go(func() {
		s.update(func(next config) {
			next["host"] = "b.example"
			next["port"] = "8080"
		})
	})
	wg.Wait()

	cur := s.load()
	fmt.Println(old["host"]+":"+old["port"], cur["host"]+":"+cur["port"])

	// Output:
	// a.example:80 b.example:8080
}
```

What makes it correct:

- **Publication.** A reader that sees the new pointer also sees the writes that built the value
  ([atomic store before load](../concepts/memory-visibility.md#atomic-store-load)).
- **Immutability.** A value is never changed after it is published, so readers need nothing else.
- **One writer at a time.** Without the mutex, two writers clone the same version and one update is lost.
- **Initialization.** The pointer must hold a value before the first `load` or `update`; the example stores one up
  front.

The `sync/atomic` documentation shows the same idiom with `atomic.Value`; `atomic.Pointer[T]` is the typed
equivalent and needs no type assertion.

## Composition

- [Compare-and-Swap Loop](compare-and-swap-loop.md): with several writers and no mutex, publishing the copy is a CAS loop on the pointer.

## Applicability

Use it when reads vastly outnumber writes, the value is small enough to copy on each write, and readers can work
with a slightly old version. Use a `sync.RWMutex` or `sync.Mutex` when writes are frequent, the value is large, or
readers must always see the latest write. `sync.Map` covers some read-mostly map cases with per-key updates.

## Consequences

- Reads take no lock and never block, and each reader sees a consistent version.
- Every write copies the whole value and leaves garbage behind; frequent writes to a large value are costly.
- Readers may act on a stale version: consistent, but not necessarily current.
- Modifying a published value, or anything shared with it, is a data race; a nested map or slice is shared by the
  shallow clone ([shallow copy](../concepts/confinement.md#shallow-copy)).

## Variants

### Several writers with CompareAndSwap

Each writer copies, modifies, and swaps only if the pointer is still the one it copied, retrying otherwise. No
writer waits, but the copy and the change are repeated on each retry, so the change must be pure
([duplicated work](../concepts/cooperative-cancellation.md#duplicates-idempotent)) and this suits rare writes.

```go
func updateCAS(cur *atomic.Pointer[config], fn func(next config)) {
	for {
		old := cur.Load()
		next := maps.Clone(*old)
		fn(next)

		if cur.CompareAndSwap(old, &next) {
			return
		}
	}
}

func main() {
	var cur atomic.Pointer[config]
	cur.Store(&config{})

	var wg sync.WaitGroup
	for i := range 10 {
		wg.Go(func() {
			updateCAS(&cur, func(next config) { next[fmt.Sprint("k", i)] = "v" })
		})
	}
	wg.Wait()

	fmt.Println(len(*cur.Load()))

	// Output:
	// 10
}
```

## Related patterns

- [Snapshot via Owner](snapshot-via-owner.md) hands out copies from a goroutine that owns the state.
- [Monitor](monitor.md) guards mutable state with a lock instead.

## Anti-patterns

- [Pointer over a Channel](../anti-patterns/pointer-over-channel.md): A pointer, slice, or map is sent over a channel and then used by both sides, so the data is shared although it was handed over.

## Sources

- [The Go Authors (n.d.f)](../sources.md#source-go-sync-atomic), Value, examples "Config" and "ReadMostly" ([link](https://pkg.go.dev/sync/atomic#example-Value-ReadMostly))
- [Herlihy and Shavit (2012)](../sources.md#source-herlihy-shavit-2012)
