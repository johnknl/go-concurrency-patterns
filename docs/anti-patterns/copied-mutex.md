---
title: Copied Mutex
intent: Copy a value that contains a mutex, so the copy has its own lock state and code that locks it does not exclude code that locks the original.
purpose: coordination
package: antipatterns/copiedmutex
mechanisms: [sync]
concerns: [data-race-safety]
remedies: [monitor]
violates: [ownership]
sources:
  - id: go-vet-copylock
  - id: go-sync
---
# Copied Mutex

<!-- meta -->
Copy a value that contains a mutex, so the copy has its own lock state and code that locks it does not exclude code that locks the original.

<div class="cp-props" markdown>

| | |
|---|---|
| Violates | [Clear ownership](../index.md#principle-ownership) |
| Purpose | [Coordination](../catalog/purpose.md#purpose-coordination) |
| Mechanisms | [Sync](../catalog/mechanism.md#mechanism-sync) |
| Concerns | [Data race safety](../catalog/concern.md#concern-data-race-safety) |
| Code | [`antipatterns/copiedmutex`](https://github.com/johnknl/go-concurrency-patterns/tree/main/antipatterns/copiedmutex) |

</div>
<!-- /meta -->

## Symptoms

- Locking appears to have no effect: the race detector reports races on fields that are always accessed under the
  lock.
- Updates made through a method are lost.
- A goroutine blocks forever on a copy that was made while the original was locked.

## Example

=== "Faulty"

    A value receiver copies the struct and its mutex. `go vet` rejects this form:

    ```go
    // Rejected by go vet: "Inc passes lock by value: <pkg>.counter contains sync.Mutex".
    func (c counter) Inc() {
    	c.mu.Lock()
    	defer c.mu.Unlock()
    	c.n++ // increments the copy
    }
    ```

    To run the consequence, the examples copy with a generic helper, `copyOf`, which vet does not see. Locking the
    copy leaves the original's mutex free and its counter unchanged; a copy of a locked counter stays locked after
    the original is unlocked, so a `Lock` on it would block forever.

    <!-- code: antipatterns/copiedmutex#counter antipatterns/copiedmutex#copyOf antipatterns/copiedmutex#Example_copyHasItsOwnLock:main antipatterns/copiedmutex#Example_copyOfLockedMutexIsLocked:main -->

    ```go
    type counter struct {
    	mu sync.Mutex
    	n  int
    }

    func copyOf[T any](v *T) T { return *v }

    func main() {
    	c := &counter{}
    	cp := copyOf(c) // what a method with receiver (c counter) gets
    	cp.mu.Lock()
    	cp.n++
    	fmt.Println("original lock free while the copy is held:", c.mu.TryLock())
    	c.mu.Unlock()
    	cp.mu.Unlock()
    	fmt.Println("original n:", c.n)
    	// Output:
    	// original lock free while the copy is held: true
    	// original n: 0
    }

    func main() {
    	c := &counter{}
    	c.mu.Lock()
    	cp := copyOf(c) // copied while locked
    	c.mu.Unlock()
    	fmt.Println("copy free after original unlocked:", cp.mu.TryLock())
    	// Output:
    	// copy free after original unlocked: false
    }
    ```

    <!-- /code -->

=== "Fixed"

    Use pointer receivers, store and pass `*T`, range over `[]*T` or by index, and keep the mutex in a type used
    only through a pointer. 100 concurrent increments on each of two counters give exact counts.

    <!-- code: antipatterns/copiedmutex#safeCounter antipatterns/copiedmutex#safeCounter.Inc antipatterns/copiedmutex#Example_pointerReceiver:main -->

    ```go
    type safeCounter struct {
    	mu sync.Mutex
    	n  int
    }

    func (c *safeCounter) Inc() {
    	c.mu.Lock()
    	defer c.mu.Unlock()
    	c.n++
    }

    func main() {
    	counters := []*safeCounter{{}, {}}
    	var wg sync.WaitGroup
    	for range 100 {
    		for _, c := range counters { // c is a *safeCounter, the mutex is not copied
    			wg.Go(c.Inc)
    		}
    	}
    	wg.Wait()
    	fmt.Println(counters[0].n, counters[1].n)
    	// Output:
    	// 100 100
    }
    ```

    <!-- /code -->

## Why it goes wrong

`sync.Mutex` holds its state in plain fields, so copying a struct that contains it (passing or returning by value, a
value receiver, an assignment, a `range` variable) yields a second, independent mutex. The `sync` documentation says
a `Mutex` must not be copied after first use. Exclusion holds only among goroutines that lock the same mutex, so code
locking the copy and code locking the original can run together, and their accesses to data reachable from both
(maps, slices, pointers) race. A copy taken while the mutex is locked, or has waiters, carries that state with no
goroutine that will ever unlock it.

## Detection

- `go vet`'s copylocks analyzer, which `go test` runs by default, reports copies of types containing a lock
  (`sync.Mutex`, `RWMutex`, `WaitGroup`, `Cond`, `Once`, or any type whose pointer has `Lock` and `Unlock`): "passes
  lock by value", "assignment copies lock value", "return copies lock value", "call of ... copies lock value",
  "range var ... copies lock".
- vet sees only copies where the static type contains a lock; copies through a type parameter or `reflect` are not
  reported.
- The race detector reports the resulting races when they happen during a run.

## Fixed by

<!-- remedies -->
- [Monitor](../patterns/monitor.md): Make a type's state safe to use from many goroutines, with no access left unprotected.
<!-- /remedies -->

## Sources

<!-- sources -->
- [The Go Authors (n.d.c)](../sources.md#source-go-vet-copylock)
- [The Go Authors (n.d.e)](../sources.md#source-go-sync)
<!-- /sources -->
