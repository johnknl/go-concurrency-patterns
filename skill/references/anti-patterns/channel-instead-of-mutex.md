# Channel Where a Mutex Would Do

Guard plain state with an owner goroutine and request and reply channels, where a mutex or an atomic next to the state would do, so every call pays a round trip and the state gains a goroutine whose lifetime someone has to manage.

- **Violates:** [The simplest mechanism that works](../principles.md#principle-simplest-mechanism)
- **Purpose:** Coordination
- **Mechanisms:** Channels, Sync
- **Concerns:** Data race safety, Composition
- **Breaks:** [Confinement](../concepts/confinement.md)
- **Code:** [`antipatterns/channelmutex`](https://github.com/johnknl/go-concurrency-patterns/tree/main/antipatterns/channelmutex)

## Symptoms

- A goroutine whose loop is a `select` over request channels, each case reading or updating one variable.
- Requests that carry a reply channel, created per call, for a getter.
- A `stop` or `Close` method on a type that only holds data, and tests that must remember to call it.
- Callers that hang after shutdown because the owner no longer receives.

## Example

**Faulty**

`ownedCounter` counts with an owner goroutine, and a hundred concurrent increments give the right count. After
`stop` nobody receives, so a later `inc` would block forever, shown here with a non-blocking send.

```go
type ownedCounter struct {
	incr chan struct{}
	get  chan chan int
	done chan struct{}
	quit chan struct{}
}

func (c *ownedCounter) loop() {
	defer close(c.done)

	n := 0

	for {
		select {
		case <-c.incr:
			n++
		case reply := <-c.get:
			reply <- n
		case <-c.quit:
			return
		}
	}
}

func (c *ownedCounter) inc() { c.incr <- struct{}{} }

func (c *ownedCounter) value() int {
	reply := make(chan int)
	c.get <- reply

	return <-reply
}

func (c *ownedCounter) stop() {
	close(c.quit)
	<-c.done
}

func main() {
	c := newOwnedCounter()

	var wg sync.WaitGroup
	for range 100 {
		wg.Go(c.inc)
	}

	wg.Wait()
	fmt.Println("count:", c.value())

	c.stop()

	select {
	case c.incr <- struct{}{}:
		fmt.Println("inc after stop: accepted")
	default:
		fmt.Println("inc after stop: would block forever")
	}

	// Output:
	// count: 100
	// inc after stop: would block forever
}
```

**Fixed**

A mutex next to the state, as a [Monitor](../patterns/monitor.md), or an atomic for a single number. Nothing is
started, so nothing has to be stopped.

```go
type counter struct {
	mu sync.Mutex
	n  int
}

func (c *counter) inc() {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.n++
}

func (c *counter) value() int {
	c.mu.Lock()
	defer c.mu.Unlock()

	return c.n
}

type atomicCounter struct {
	n atomic.Int64
}

func main() {
	var (
		c  counter
		a  atomicCounter
		wg sync.WaitGroup
	)

	for range 100 {
		wg.Go(func() {
			c.inc()
			a.inc()
		})
	}

	wg.Wait()
	fmt.Println("mutex:", c.value(), "atomic:", a.value())

	// Output:
	// mutex: 100 atomic: 100
}
```

## Why it goes wrong

This is an [owner goroutine](../concepts/confinement.md#owner-goroutine), or an [Active
Object](../patterns/active-object.md), applied where the state is all the goroutine has. It is free of data races, but
it adds a lifetime to reason about: someone has to start the goroutine, stop it, and make sure nobody calls it
afterwards ([owner costs](../concepts/confinement.md#owner-costs)). Each call is two channel operations and a context
switch where an uncontended lock is an atomic operation, and a getter allocates a reply channel. A mutex gives the same
exclusion, and its zero value is ready to use. The Go wiki's rule of thumb: channels for passing ownership, distributing
work, and communicating results; mutexes for caches and state.

The owner goroutine is justified when it [has work of its own](../concepts/confinement.md#owner-has-work): timers, I/O,
side effects that must happen in order, or a policy that would otherwise hold a lock while waiting ([Lock Held While
Blocking](lock-held-while-blocking.md)).

## Detection

- A goroutine whose cases only read or assign fields, and whose callers immediately wait for a reply.
- A type that needs `stop` but starts no I/O and no timer.
- Goroutine profiles with one idle owner goroutine per instance of a data type.

## Fixed by

- [Monitor](../patterns/monitor.md): Make a type's state safe to use from many goroutines, with no access left unprotected.
- [Compare-and-Swap Loop](../patterns/compare-and-swap-loop.md): Update a single shared value from its old value without making other writers wait.

Use the Compare-and-Swap Loop when the update of a single number depends on its old value.

## Sources

- [The Go Authors (n.d.i)](../sources.md#source-go-wiki-mutex-or-channel)
