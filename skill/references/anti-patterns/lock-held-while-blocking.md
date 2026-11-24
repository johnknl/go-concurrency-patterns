# Lock Held While Blocking

Hold a mutex across a channel operation, I/O or a callback, so everyone else who needs the lock waits, and a deadlock follows if the other side needs it too.

- **Violates:** [Synchronize explicitly](../principles.md#principle-explicit-synchronization)
- **Purpose:** Coordination
- **Mechanisms:** Sync, Channels
- **Concerns:** Data race safety, Liveness
- **Breaks:** [Confinement](../concepts/confinement.md)
- **Code:** [`antipatterns/lockblocking`](https://github.com/johnknl/go-concurrency-patterns/tree/main/antipatterns/lockblocking)

## Symptoms

- Unrelated operations on the same value stall for as long as a receiver, a disk, or the network is slow.
- In the worst case the program stops: a goroutine dump shows one goroutine blocked in a channel operation and
  others in `sync.(*Mutex).Lock`. The runtime reports a deadlock only when no goroutine at all can run, so in a
  server it looks like a hang.

## Example

**Faulty**

`badStore.Publish` sends on an unbuffered channel while holding `mu`. With nobody receiving, `TryLock` shows
that the lock stays held for the whole blocked send.

```go
type badStore struct {
	mu    sync.Mutex
	value int
}

func (s *badStore) Publish(out chan<- int) {
	s.mu.Lock()
	defer s.mu.Unlock()
	out <- s.value
}

func main() {
	s := &badStore{value: 42}
	out := make(chan int) // unbuffered: the send blocks until someone receives
	started := make(chan struct{})
	var wg sync.WaitGroup
	wg.Go(func() {
		close(started)
		s.Publish(out)
	})
	<-started

	// Retry until the lock is held by Publish. It is released only after the send completes, so a
	// successful TryLock would mean Publish has not locked yet, and the loop tries again.
	for s.mu.TryLock() {
		s.mu.Unlock()
	}
	fmt.Println("lock held while the send is blocked")

	fmt.Println("received:", <-out)
	wg.Wait()
	fmt.Println("lock free after send:", s.mu.TryLock())
	// Output:
	// lock held while the send is blocked
	// received: 42
	// lock free after send: true
}
```

**Fixed**

`store.Publish` copies what it needs under the lock, unlocks, and then delivers. While the delivery blocks, the
lock is free. The value delivered is a snapshot, which may be out of date when it is received.

```go
type store struct {
	mu    sync.Mutex
	value int
}

func (s *store) Publish(deliver func(int)) {
	s.mu.Lock()
	v := s.value
	s.mu.Unlock()
	deliver(v)
}

func main() {
	s := &store{value: 42}
	inDeliver := make(chan int)
	release := make(chan struct{})
	var wg sync.WaitGroup
	wg.Go(func() {
		s.Publish(func(v int) {
			inDeliver <- v
			<-release // stands in for a slow send, I/O or callback
		})
	})

	fmt.Println("delivering:", <-inDeliver)
	// Publish is blocked inside deliver, and the lock is free.
	fmt.Println("lock free while delivery is blocked:", s.mu.TryLock())
	s.mu.Unlock()
	close(release)
	wg.Wait()
	// Output:
	// delivering: 42
	// lock free while delivery is blocked: true
}
```

## Why it goes wrong

A mutex protects a short critical section; anything that can block for an unbounded time makes the critical
section unbounded. The send completes only when another goroutine receives, so how long the lock is held depends
on that goroutine. If the receiver first needs the same lock, the two wait for each other.

## Detection

- Look between `Lock` and `Unlock` for channel operations, `select`, I/O, sleeps, and calls to function values or
  interface methods.
- Goroutine dumps and the mutex and block profiles show the holder and its waiters. The race detector and
  `go vet` do not report it.

## Fixed by

- [Active Object](../patterns/active-object.md): Run all operations on an object one at a time, in the order they arrive, while callers either continue or wait for the result.
- [Snapshot via Owner](../patterns/snapshot-via-owner.md): Read a consistent copy of state that one goroutine owns, without sharing the state.

When the state is changed by many goroutines, giving it to one
[owner goroutine](../concepts/confinement.md#owner-goroutine) removes the lock altogether.

## Sources

- [Tu et al. (2019)](../sources.md#source-tu-2019)
