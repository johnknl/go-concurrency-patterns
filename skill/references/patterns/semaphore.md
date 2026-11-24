# Semaphore

Limit how many operations use a resource at the same time.

- **Also known as:** Counting semaphore
- **Purpose:** Coordination
- **Scale:** Idiom
- **Mechanisms:** Channels, Select, Sync
- **Concerns:** Bounded concurrency
- **Concepts:** [Memory Visibility](../concepts/memory-visibility.md), [Overload Policy](../concepts/overload-policy.md)
- **Code:** [`patterns/semaphore`](https://github.com/johnknl/go-concurrency-patterns/tree/main/patterns/semaphore)

> **Library.**
> [`golang.org/x/sync/semaphore`](https://pkg.go.dev/golang.org/x/sync/semaphore) provides a weighted semaphore whose `Acquire` takes a context, for permits of different sizes or when a type is preferred over a channel.

## Problem

More goroutines want to use a resource at once than the resource can serve. Each goroutine should run its own
code and wait only for access to the resource, so that at most a fixed number use it at the same time.

## Mechanism

A buffered channel is the semaphore: a send takes a permit and blocks while all *C* are taken, and a receive
returns one. The *k*th receive happens before the *k+C*th send completes
([memory visibility](../concepts/memory-visibility.md#capacity-edge)), which is what caps the holders at *C*.

In this example, twenty goroutines share three permits and record the highest number that held one at once.

```go
func main() {
	sem := make(chan struct{}, 3)

	var (
		wg           sync.WaitGroup
		mu           sync.Mutex
		active, peak int
	)

	for range 20 {
		wg.Go(func() {
			sem <- struct{}{}
			defer func() { <-sem }()

			mu.Lock()
			active++
			peak = max(peak, active)
			mu.Unlock()

			mu.Lock()
			active--
			mu.Unlock()
		})
	}

	wg.Wait()
	fmt.Println(peak <= 3)

	// Output:
	// true
}
```

## Applicability

Use it when a resource supports a fixed number of concurrent users and the callers already have their own
goroutines. Use a `sync.Mutex` when the limit is one, and a [Worker Pool](worker-pool.md) when the number of
goroutines should be bounded too.

## Consequences

- A permit that is not released is lost for good: release with `defer` right after acquiring.
- Waiters are not served in a guaranteed order, and all permits have the same size.
- A full semaphore needs a policy ([overload policy](../concepts/overload-policy.md#every-limit)): wait, wait with a
  bound, or refuse, as in the variants below.
- Only the guarded section is bounded; the waiting goroutines still exist.

## Variants

### Bounded acquire

Make the acquire one case of a `select` with a context, so a caller gives up after a while
([waiting must be cancelable](../concepts/overload-policy.md#wait-cancelable)).

```go
func main() {
	sem := make(chan struct{}, 1)
	sem <- struct{}{} // the only permit is taken

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	select {
	case sem <- struct{}{}:
		fmt.Println("acquired")
	case <-ctx.Done():
		fmt.Println(ctx.Err())
	}

	// Output:
	// context canceled
}
```

### Try-acquire

A `select` with a `default` case takes a permit only if one is free now, which lets a caller shed load.

```go
func main() {
	sem := make(chan struct{}, 1)

	for range 2 {
		select {
		case sem <- struct{}{}:
			fmt.Println("acquired")
		default:
			fmt.Println("busy")
		}
	}

	// Output:
	// acquired
	// busy
}
```

## Related patterns

- [Worker Pool](worker-pool.md) bounds concurrency by owning the goroutines.
- [Resource Pool](resource-pool.md) hands out specific resources instead of permits.

## Anti-patterns

- [Unbounded Goroutines](../anti-patterns/unbounded-goroutines.md): Start one goroutine per input item, so concurrency grows with the input instead of with capacity.

## Sources

- [Dijkstra (1965)](../sources.md#source-dijkstra-1965)
- [The Go Authors (n.d.a)](../sources.md#source-effective-go), "Channels"
