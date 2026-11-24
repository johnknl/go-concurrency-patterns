# Monitor

Make a type's state safe to use from many goroutines, with no access left unprotected.

- **Also known as:** Monitor Object, Synchronized object, Mutex guarding
- **Purpose:** Coordination
- **Scale:** Pattern
- **Mechanisms:** Sync
- **Concerns:** Data race safety
- **Concepts:** [Memory Visibility](../concepts/memory-visibility.md), [Confinement](../concepts/confinement.md)
- **Code:** [`patterns/monitor`](https://github.com/johnknl/go-concurrency-patterns/tree/main/patterns/monitor)

## Problem

Several goroutines use the same state, and some operations change several parts of it at once. Unsynchronized
access is a data race, and a reader that sees only part of a change sees a state that never existed. Locking by
hand at every access is easy to forget.

## Mechanism

The type has an unexported `sync.Mutex` and unexported state, and exposes only methods. Each method locks at the top and
unlocks with `defer`, so it is atomic with respect to the others; colloquially this is guarding the state with a mutex.
The invariant (in this example: the balances add up to the deposits) may be broken inside a method, but holds whenever
the lock is free. Unlocking a mutex happens before the next lock of it returns
([memory visibility](../concepts/memory-visibility.md#unlock-before-lock)), so each method sees every write of the
methods that held the lock before it. The monitor must own its state: `newBank` copies the map it is given, because
a map the caller kept could be changed without the lock ([a shallow copy shares nested
values](../concepts/confinement.md#shallow-copy), which does not matter here as the values are integers).

```go
// bank owns its balances and the mutex that guards them. Every access goes through a method that holds the lock.
// Invariant, whenever mu is free: the balances add up to the amount deposited.
type bank struct {
	balances map[string]int
	mu       sync.Mutex
}

func newBank(balances map[string]int) *bank {
	return &bank{balances: maps.Clone(balances)}
}

// transfer moves amount from one account to another if the source has enough. Both changes happen under one lock,
// so no other method sees money that has left one account but not reached the other.
func (b *bank) transfer(from, to string, amount int) bool {
	b.mu.Lock()
	defer b.mu.Unlock()

	if b.balances[from] < amount {
		return false
	}

	b.balances[from] -= amount
	b.balances[to] += amount

	return true
}

func (b *bank) total() int {
	b.mu.Lock()
	defer b.mu.Unlock()

	sum := 0
	for _, v := range b.balances {
		sum += v
	}

	return sum
}

func main() {
	b := newBank(map[string]int{"a": 1000, "b": 1000})

	var wg sync.WaitGroup

	for range 100 {
		wg.Go(func() { b.transfer("a", "b", 3) })
		wg.Go(func() { b.transfer("b", "a", 1) })
		wg.Go(func() { _ = b.total() })
	}

	wg.Wait()
	fmt.Println(b.balance("a"), b.balance("b"), b.total())
	fmt.Println("overdraft allowed:", b.transfer("a", "b", 5000))

	// Output:
	// 800 1200 2000
	// overdraft allowed: false
}
```

## Applicability

Use it when several goroutines share a small piece of state, such as a map, a counter, or a cache, and each
operation is short and does not block. The Go wiki puts it as: a mutex for caches and state, channels for passing
ownership and coordinating goroutines. When the operations are long, block, or must be cancellable, give the state
to one goroutine that has work of its own ([confinement](../concepts/confinement.md#owner-has-work),
[Active Object](active-object.md)).

## Consequences

- Callers cannot forget to lock, because they reach the state only through methods. Unexported fields keep other
  packages out; inside the package this is a convention.
- Never block while holding the lock: no channel operations, I/O, or callbacks
  ([Lock Held While Blocking](../anti-patterns/lock-held-while-blocking.md)).
- `sync.Mutex` is not reentrant: a method that calls another locking method of the same type deadlocks. Split out
  an unexported helper that expects the lock to be held.
- Operations that span two monitors need a fixed lock order.
- A mutex must not be copied after first use: use pointer receivers; `go vet` (copylocks) reports copies.
- Embedding `sync.Mutex` anonymously adds `Lock` and `Unlock` to the type's API and lets any caller lock it; a named
  field (`mu`) keeps the lock an implementation detail.

## Variants

### Read-write lock

When reads dominate and each read does real work, a `sync.RWMutex` lets readers run together; for trivially short
reads a plain `Mutex` is usually as fast or faster. Once a goroutine calls `Lock`, new `RLock` calls block until the
writer is done, so a reader must not take `RLock` again while it holds it.

```go
type registry struct {
	names map[int]string
	mu    sync.RWMutex
}

func (r *registry) lookup(id int) (string, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	name, ok := r.names[id]

	return name, ok
}

func (r *registry) register(id int, name string) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.names[id] = name
}

func main() {
	r := &registry{names: map[int]string{}}

	var wg sync.WaitGroup

	for i := range 10 {
		wg.Go(func() { r.register(i, fmt.Sprint("worker-", i)) })
	}

	wg.Wait()

	names := make([]string, 20)

	for i := range names {
		wg.Go(func() { names[i], _ = r.lookup(i % 10) })
	}

	wg.Wait()
	fmt.Println(names[3], names[17])

	// Output:
	// worker-3 worker-7
}
```

## Related patterns

- [Condition Variable](../stdlib/condition-variable.md) adds waiting for a condition on the monitor's state.
- [Confinement](../concepts/confinement.md#owner-goroutine) and [Active Object](active-object.md) protect state with a single owning
  goroutine and channels instead of a lock.
- [Lazy Initialization](../stdlib/lazy-initialization.md) for state that is built once.

## Anti-patterns

- [Channel Where a Mutex Would Do](../anti-patterns/channel-instead-of-mutex.md): Guard plain state with an owner goroutine and request and reply channels, where a mutex or an atomic next to the state would do, so every call pays a round trip and the state gains a goroutine whose lifetime someone has to manage.
- [Copied Mutex](../anti-patterns/copied-mutex.md): Copy a value that contains a mutex, so the copy has its own lock state and code that locks it does not exclude code that locks the original.
- [Inconsistent Lock Order](../anti-patterns/inconsistent-lock-order.md): Take two locks in different orders in different goroutines, so each can hold one while waiting for the other, and both wait forever.

## Sources

- [Hoare (1974)](../sources.md#source-hoare-1974)
- [Lampson and Redell (1980)](../sources.md#source-lampson-redell-1980)
- [Schmidt et al. (2000)](../sources.md#source-schmidt-2000), Monitor Object
- [The Go Authors (n.d.e)](../sources.md#source-go-sync)
