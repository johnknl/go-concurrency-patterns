---
title: Resource Pool
aliases: ['Connection pool', 'Fixed-size pool']
intent: Share a fixed number of expensive resources among many goroutines, and wait when all are in use.
purpose: coordination
scale: idiom
package: patterns/resourcepool
mechanisms: [channels, sync]
concerns: [bounded-concurrency]
concepts:
  - overload-policy: 'an empty pool makes get wait, and the wait selects on a context.'
original: true
---
# Resource Pool

<!-- meta -->
Share a fixed number of expensive resources among many goroutines, and wait when all are in use.

<div class="cp-props" markdown>

| | |
|---|---|
| Also known as | Connection pool, Fixed-size pool |
| Purpose | [Coordination](../catalog/purpose.md#purpose-coordination) |
| Scale | [Idiom](../catalog/scale.md#scale-idiom) |
| Mechanisms | [Channels](../catalog/mechanism.md#mechanism-channels), [Sync](../catalog/mechanism.md#mechanism-sync) |
| Concerns | [Bounded concurrency](../catalog/concern.md#concern-bounded-concurrency) |
| Concepts | [Overload Policy](../concepts/overload-policy.md) |
| Code | [`patterns/resourcepool`](https://github.com/johnknl/go-concurrency-patterns/tree/main/patterns/resourcepool) |

</div>
<!-- /meta -->

## Problem

Some resources are expensive to create or limited in number. Many goroutines borrow one now and then, use it, and
give it back; the number in existence stays bounded, and when all are in use, the next borrower waits.

## Mechanism

A buffered channel, filled up front, holds the resources. `get` receives one and waits while the channel is empty; `put`
sends it back, and never blocks, because the buffer is full only when every resource is back. The number in use at once
is exact by construction. `get` also selects on a context, so a caller can stop waiting
([waiting must be cancelable](../concepts/overload-policy.md#wait-cancelable)). In this example, the resources
are `conn` values that only carry an id.

<!-- code: patterns/resourcepool#conn patterns/resourcepool#newPool patterns/resourcepool#get patterns/resourcepool#put patterns/resourcepool#Example_borrow:main -->
```go
type conn struct{ id int }

func newPool(n int) chan *conn {
	pool := make(chan *conn, n)

	for id := range n {
		pool <- &conn{id: id}
	}

	return pool
}

func get(ctx context.Context, pool chan *conn) (*conn, error) {
	select {
	case c := <-pool:
		return c, nil
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

func put(pool chan *conn, c *conn) { pool <- c }

func main() {
	ctx := context.Background()
	pool := newPool(2)

	a, _ := get(ctx, pool)
	b, _ := get(ctx, pool)

	expired, cancel := context.WithCancel(ctx)
	cancel()

	_, err := get(expired, pool)
	fmt.Println("third get:", err)

	put(pool, a)
	c, _ := get(ctx, pool)
	fmt.Println("reused after release:", c == a)

	put(pool, b)
	put(pool, c)

	// Output:
	// third get: context canceled
	// reused after release: true
}
```
<!-- /code -->

**Eight workers, two connections.** At most two connections are in use at any time.

<!-- code: patterns/resourcepool#Example_boundedUse:main -->
```go
func main() {
	pool := newPool(2)

	var inUse atomic.Int32
	var exceeded atomic.Bool

	var wg sync.WaitGroup

	for range 8 {
		wg.Go(func() {
			c, err := get(context.Background(), pool)
			if err != nil {
				return
			}
			defer put(pool, c)

			if inUse.Add(1) > 2 {
				exceeded.Store(true)
			}

			inUse.Add(-1)
		})
	}

	wg.Wait()

	fmt.Println("more than 2 in use:", exceeded.Load())

	// Output:
	// more than 2 in use: false
}
```
<!-- /code -->

## Applicability

Use it when each resource is a distinct, expensive object whose number must be bounded, and all can be created up
front. When goroutines only need permission to proceed, a [Semaphore](semaphore.md) suffices; for temporary objects
that only cost allocation, an [Object Cache](object-cache.md) is simpler.

## Consequences

- A resource that is not returned is lost for good; once all are lost, every `get` waits until its context ends, or
  forever. Return it with `defer put(...)` right after a successful `get`.
- A resource must not be used after `put`; the next user may already have it.
- Returning a broken resource, such as a closed connection, hands it to the next user. Put a new one instead.
- Putting a resource that did not come from the pool blocks when the pool is full, and otherwise lets more than the
  fixed number circulate.

## Variants

### Mutex and condition variable

The free resources are a slice guarded by a `sync.Mutex`, and `get` waits on a `sync.Cond` that `put` signals. The
pool can report how many are free, grow or shrink, and hand out the most recently returned resource first, which
keeps a few warm and lets the rest idle. The cost is more code: `sync.Cond` cannot wait on a context, so
`context.AfterFunc` wakes all waiters when the context ends, and its callback must take the lock to avoid a lost
wake-up. Unlike the `select`, `get` prefers a free resource over a context that has already ended.

<!-- code: patterns/resourcepool#lockedPool patterns/resourcepool#newLockedPool patterns/resourcepool#lockedPool.get patterns/resourcepool#lockedPool.put patterns/resourcepool#Example_lockedPool:main -->
```go
type lockedPool struct {
	cond *sync.Cond
	free []*conn
	mu   sync.Mutex
}

func newLockedPool(n int) *lockedPool {
	p := &lockedPool{free: make([]*conn, 0, n)}
	p.cond = sync.NewCond(&p.mu)

	for id := range n {
		p.free = append(p.free, &conn{id: id})
	}

	return p
}

func (p *lockedPool) get(ctx context.Context) (*conn, error) {
	stop := context.AfterFunc(ctx, func() {
		p.mu.Lock()
		defer p.mu.Unlock()

		p.cond.Broadcast()
	})
	defer stop()

	p.mu.Lock()
	defer p.mu.Unlock()

	for len(p.free) == 0 {
		if err := ctx.Err(); err != nil {
			return nil, err
		}

		p.cond.Wait()
	}

	c := p.free[len(p.free)-1]
	p.free = p.free[:len(p.free)-1]

	return c, nil
}

func (p *lockedPool) put(c *conn) {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.free = append(p.free, c)
	p.cond.Signal()
}

func main() {
	ctx := context.Background()
	pool := newLockedPool(2)

	a, _ := pool.get(ctx)
	b, _ := pool.get(ctx)

	expired, cancel := context.WithCancel(ctx)
	cancel()

	_, err := pool.get(expired)
	fmt.Println("third get:", err)

	pool.put(a)
	pool.put(b)

	c, _ := pool.get(ctx)
	fmt.Println("last returned comes first:", c == b)

	pool.put(c)

	// Output:
	// third get: context canceled
	// last returned comes first: true
}
```
<!-- /code -->

## Related patterns

- [Condition Variable](../stdlib/condition-variable.md) is the waiting mechanism of the mutex variant.
- [Semaphore](semaphore.md) bounds concurrent use without handing out a specific resource.
- [Object Cache](object-cache.md) reuses temporary objects without bounding their number.
- [Worker Pool](worker-pool.md) keeps goroutines, not objects, for reuse.

## Sources

<!-- sources -->
Original work.
<!-- /sources -->
