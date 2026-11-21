---
title: Inconsistent Lock Order
intent: Take two locks in different orders in different goroutines, so each can hold one while waiting for the other, and both wait forever.
purpose: coordination
package: antipatterns/lockorder
mechanisms: [sync]
concerns: [liveness]
remedies: [monitor, active-object]
violates: [explicit-synchronization]
concepts:
  - confinement: 'an owner goroutine needs no second lock.'
sources:
  - id: coffman-1971
---
# Inconsistent Lock Order

<!-- meta -->
Take two locks in different orders in different goroutines, so each can hold one while waiting for the other, and both wait forever.

<div class="cp-props" markdown>

| | |
|---|---|
| Violates | [Synchronize explicitly](../index.md#principle-explicit-synchronization) |
| Purpose | [Coordination](../catalog/purpose.md#purpose-coordination) |
| Mechanisms | [Sync](../catalog/mechanism.md#mechanism-sync) |
| Concerns | [Liveness](../catalog/concern.md#concern-liveness) |
| Breaks | [Confinement](../concepts/confinement.md) |
| Code | [`antipatterns/lockorder`](https://github.com/johnknl/go-concurrency-patterns/tree/main/antipatterns/lockorder) |

</div>
<!-- /meta -->

## Symptoms

- The goroutines involved, and every goroutine that later needs either lock, stop making progress.
- The runtime reports "all goroutines are asleep - deadlock!" only when every goroutine is blocked, so a server
  hangs without an error.
- It depends on timing, so tests can pass for a long time.

## Example

=== "Faulty"

    `badTransfer` locks `from` and then `to`. The example pauses a->b and b->a between their two locks, and
    `TryLock` shows each one's next lock is taken. It then abandons b->a so that a->b can finish.

    <!-- code: antipatterns/lockorder#account antipatterns/lockorder#badTransfer antipatterns/lockorder#Example_circularWait:main -->

    ```go
    type account struct {
    	id      int
    	mu      sync.Mutex
    	balance int
    }

    func badTransfer(from, to *account, amount int, between func() bool) bool {
    	from.mu.Lock()
    	defer from.mu.Unlock()
    	if !between() {
    		return false
    	}
    	to.mu.Lock()
    	defer to.mu.Unlock()
    	from.balance -= amount
    	to.balance += amount
    	return true
    }

    func main() {
    	a, b := &account{id: 1, balance: 100}, &account{id: 2, balance: 100}
    	held := make(chan struct{})
    	goAB, goBA := make(chan bool), make(chan bool)
    	var wg sync.WaitGroup
    	var doneAB, doneBA bool
    	wg.Go(func() {
    		doneAB = badTransfer(a, b, 10, func() bool { held <- struct{}{}; return <-goAB })
    	})
    	wg.Go(func() {
    		doneBA = badTransfer(b, a, 20, func() bool { held <- struct{}{}; return <-goBA })
    	})
    	<-held
    	<-held

    	// Each transfer holds its first lock, and its next lock is the other's first: a circular wait.
    	fmt.Println("a->b can lock b:", b.mu.TryLock())
    	fmt.Println("b->a can lock a:", a.mu.TryLock())

    	// Letting both continue would deadlock. Abandon b->a so that it unlocks b, then let a->b finish.
    	goBA <- false
    	goAB <- true
    	wg.Wait()
    	fmt.Println("a->b done:", doneAB, "b->a done:", doneBA)
    	fmt.Println("balances:", a.balance, b.balance)
    	// Output:
    	// a->b can lock b: false
    	// b->a can lock a: false
    	// a->b done: true b->a done: false
    	// balances: 90 110
    }
    ```

    <!-- /code -->

=== "Fixed"

    `transfer` always locks the account with the lower id first, so transfers in both directions take the same
    first lock. 100 transfers in each direction run concurrently.

    <!-- code: antipatterns/lockorder#transfer antipatterns/lockorder#Example_lockOrderByID:main -->

    ```go
    func transfer(from, to *account, amount int) {
    	first, second := from, to
    	if to.id < from.id {
    		first, second = to, from
    	}
    	first.mu.Lock()
    	defer first.mu.Unlock()
    	second.mu.Lock()
    	defer second.mu.Unlock()
    	from.balance -= amount
    	to.balance += amount
    }

    func main() {
    	a, b := &account{id: 1, balance: 1000}, &account{id: 2, balance: 1000}
    	var wg sync.WaitGroup
    	for range 100 {
    		wg.Go(func() { transfer(a, b, 3) })
    		wg.Go(func() { transfer(b, a, 2) })
    	}
    	wg.Wait()
    	fmt.Println("balances:", a.balance, b.balance)
    	// Output:
    	// balances: 900 1100
    }
    ```

    <!-- /code -->

## Why it goes wrong

Coffman, Elphick and Shoshani name four conditions that together allow deadlock: mutual exclusion, hold and wait,
no preemption, and circular wait. Mutexes provide the first three: one holder at a time, a goroutine keeps its first
lock while waiting for the second, and nothing takes a lock away. Locking in argument-dependent order adds the cycle.

## Detection

- A goroutine dump (`SIGQUIT`, or the `pprof` goroutine profile) shows goroutines blocked in `sync.(*Mutex).Lock`,
  each holding a lock another one waits for.
- In review, find code that holds one lock while taking another, and check every path takes them in the same order.
- The race detector does not report deadlocks, and `go vet` does not check lock order.

## Fixed by

<!-- remedies -->
- [Monitor](../patterns/monitor.md): Make a type's state safe to use from many goroutines, with no access left unprotected.
- [Active Object](../patterns/active-object.md): Run all operations on an object one at a time, in the order they arrive, while callers either continue or wait for the result.
<!-- /remedies -->

An [Active Object](../patterns/active-object.md) avoids holding two locks: a single
[owner goroutine](../concepts/confinement.md#owner-goroutine) applies the transfers one at a time.

## Sources

<!-- sources -->
- [Coffman et al. (1971)](../sources.md#source-coffman-1971)
<!-- /sources -->
