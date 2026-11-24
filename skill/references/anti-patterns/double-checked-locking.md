# Double-Checked Locking

Test a lazily initialized pointer without synchronization and lock only when it is nil, so readers race with the writer and can see a pointer to an incompletely initialized value.

- **Violates:** [Synchronize explicitly](../principles.md#principle-explicit-synchronization)
- **Purpose:** Coordination
- **Mechanisms:** Sync, Atomic
- **Concerns:** Data race safety
- **Breaks:** [Memory Visibility](../concepts/memory-visibility.md)
- **Code:** [`antipatterns/doublecheck`](https://github.com/johnknl/go-concurrency-patterns/tree/main/antipatterns/doublecheck)

## Symptoms

- A lazy getter checks a plain variable for nil without the lock, and locks and checks again only to initialize.
- `go test -race` reports a race between the write in the getter and the unlocked read when two goroutines call it.
- Without the race detector the program usually works; rarely, a caller uses a value whose fields are not yet set.

## Example

**Faulty**

`badGet` checks `badInstance` before locking. The example calls it from one goroutine only, so it runs without a
data race; two concurrent callers are what the race detector reports.

```go
const loadedName = "loaded"

type config struct {
	name string
}

var badMu sync.Mutex

var badInstance *config

func badGet() *config {
	if badInstance == nil { // unsynchronized read: races with the write below
		badMu.Lock()
		if badInstance == nil {
			badInstance = &config{name: loadedName}
		}
		badMu.Unlock()
	}
	return badInstance
}

func main() {
	// Called from one goroutine only: two concurrent callers would be a data race.
	a, b := badGet(), badGet()
	fmt.Println(a.name, a == b)
	// Output:
	// loaded true
}
```

**Fixed**

Use `sync.Once`, or `sync.OnceValue` when initialization returns the value. Completion of the function happens
before any call returns, so every caller sees the initialized value.

```go
var getConfig = sync.OnceValue(func() *config {
	return &config{name: loadedName}
})

func main() {
	results := make([]*config, 8)
	var wg sync.WaitGroup
	for i := range results {
		wg.Go(func() { results[i] = getConfig() })
	}
	wg.Wait()
	same := true
	for _, c := range results {
		same = same && c == results[0]
	}
	fmt.Println(results[0].name, same)
	// Output:
	// loaded true
}
```

**Keep a plain fast path with `atomic.Pointer`.** The store under the lock and the atomic load on the fast path are
synchronized, so a reader that sees the pointer also sees the writes made before it was stored. The second check under
the lock keeps initialization to one goroutine.

```go
var mu sync.Mutex

var instance atomic.Pointer[config]

func get() *config {
	if c := instance.Load(); c != nil {
		return c
	}
	mu.Lock()
	defer mu.Unlock()
	if c := instance.Load(); c != nil {
		return c
	}
	c := &config{name: loadedName}
	instance.Store(c)
	return c
}

func main() {
	results := make([]*config, 8)
	var wg sync.WaitGroup
	for i := range results {
		wg.Go(func() { results[i] = get() })
	}
	wg.Wait()
	same := true
	for _, c := range results {
		same = same && c == results[0]
	}
	fmt.Println(results[0].name, same)
	// Output:
	// loaded true
}
```

## Why it goes wrong

A read sees a write only across a happens-before edge ([memory
visibility](../concepts/memory-visibility.md#edge-publishes)). The unlocked `badInstance == nil` check is not ordered
with the store in another goroutine, so it is a data race. Seeing a non-nil pointer gives [no
edge](../concepts/memory-visibility.md#no-edge) with the initializing writes, so the reader may not see the fields set
before the pointer was stored. The edges that do work here are
[`sync.Once`](../concepts/memory-visibility.md#once-before-do), an [atomic store and
load](../concepts/memory-visibility.md#atomic-store-load), or the [mutex on both
sides](../concepts/memory-visibility.md#unlock-before-lock). The memory model names double-checked locking as an
incorrect idiom; Meyers and Alexandrescu describe the same problem for C++.

## Detection

- `go test -race` reports the race only when a test runs the read and the write concurrently.
- In review, look for a nil or flag check outside a lock on a variable written inside it.

## Fixed by

- [Lazy Initialization](../stdlib/lazy-initialization.md): Build a value on first use, exactly once, safely from many goroutines.
- [Compare-and-Swap Loop](../patterns/compare-and-swap-loop.md): Update a single shared value from its old value without making other writers wait.

## Sources

- [Meyers and Alexandrescu (2004)](../sources.md#source-meyers-alexandrescu-2004)
- [The Go Authors (2022)](../sources.md#source-go-memory-model)
- [The Go Authors (n.d.e)](../sources.md#source-go-sync)
