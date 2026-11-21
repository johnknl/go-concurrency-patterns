---
title: Unbounded Goroutines
intent: Start one goroutine per input item, so concurrency grows with the input instead of with capacity.
purpose: execution
package: antipatterns/unbounded
mechanisms: [channels, sync]
concerns: [bounded-concurrency, overload]
remedies: [worker-pool, semaphore]
violates: [bounded]
concepts:
  - backpressure: 'with no bound, no feedback reaches the producer.'
sources:
  - id: harsanyi-2022
---
# Unbounded Goroutines

<!-- meta -->
Start one goroutine per input item, so concurrency grows with the input instead of with capacity.

<div class="cp-props" markdown>

| | |
|---|---|
| Violates | [Bound everything](../index.md#principle-bounded) |
| Purpose | [Execution](../catalog/purpose.md#purpose-execution) |
| Mechanisms | [Channels](../catalog/mechanism.md#mechanism-channels), [Sync](../catalog/mechanism.md#mechanism-sync) |
| Concerns | [Bounded concurrency](../catalog/concern.md#concern-bounded-concurrency), [Overload](../catalog/concern.md#concern-overload) |
| Breaks | [Backpressure](../concepts/backpressure.md) |
| Code | [`antipatterns/unbounded`](https://github.com/johnknl/go-concurrency-patterns/tree/main/antipatterns/unbounded) |

</div>
<!-- /meta -->

## Symptoms

- A loop over the input starts a goroutine per item; goroutine counts and memory grow with input size and spike on
  large batches.
- `too many open files`, connection pool exhaustion, or rate-limit and timeout errors downstream.
- Latency of everything else in the process rises, and the process may be killed for running out of memory.
- Works in tests with ten items, fails in production with a million.

## Example

=== "Faulty"

    `badFanOut` runs one goroutine per item. Each call enters a `peak` counter and waits at a gate that opens only
    when all calls have started, so the peak equals the number of items.

    <!-- code: antipatterns/unbounded#peak antipatterns/unbounded#badFanOut antipatterns/unbounded#Example_unbounded:main -->

    ```go
    type peak struct{ cur, max atomic.Int64 }

    func badFanOut(items []int, work func(int)) {
    	var wg sync.WaitGroup

    	for _, item := range items {
    		wg.Go(func() { work(item) })
    	}

    	wg.Wait()
    }

    func main() {
    	const n = 100

    	var (
    		p       peak
    		started sync.WaitGroup
    	)

    	started.Add(n)

    	gate := make(chan struct{})

    	// The gate holds every goroutine until all of them have started.
    	go func() {
    		started.Wait()
    		close(gate)
    	}()

    	badFanOut(make([]int, n), func(int) {
    		p.enter()
    		started.Done()
    		<-gate
    		p.leave()
    	})

    	fmt.Println("items:", n, "peak concurrency:", p.max.Load())

    	// Output:
    	// items: 100 peak concurrency: 100
    }
    ```

    <!-- /code -->

=== "Fixed"

    `boundedFanOut` starts a fixed number of workers reading from a channel; the send blocks while all are busy, so
    the producer slows down too.

    <!-- code: antipatterns/unbounded#boundedFanOut antipatterns/unbounded#Example_bounded:main -->

    ```go
    func boundedFanOut(items []int, limit int, work func(int)) {
    	jobs := make(chan int)

    	var wg sync.WaitGroup

    	for range limit {
    		wg.Go(func() {
    			for item := range jobs {
    				work(item)
    			}
    		})
    	}

    	for _, item := range items {
    		jobs <- item
    	}

    	close(jobs)
    	wg.Wait()
    }

    func main() {
    	const (
    		n     = 100
    		limit = 4
    	)

    	var (
    		p    peak
    		done atomic.Int64
    	)

    	boundedFanOut(make([]int, n), limit, func(int) {
    		p.enter()
    		done.Add(1)
    		p.leave()
    	})

    	fmt.Println("done:", done.Load(), "peak within limit:", p.max.Load() <= limit)

    	// Output:
    	// done: 100 peak within limit: true
    }
    ```

    <!-- /code -->

## Why it goes wrong

Goroutines are cheap; what each holds is not: connections, descriptors, memory, the capacity of the service it
calls. When the work blocks on something scarce, goroutines pile up in front of it, and since the loop never waits,
no [backpressure](../concepts/backpressure.md#needs-bound) reaches the producer. The limit is whatever runs out first.

## Detection

- `go` statements inside loops over input of unknown or large size.
- Goroutine count (`runtime.NumGoroutine`, the goroutine profile) and memory growing with load.
- Load tests with large inputs; the race detector and `go vet` do not report it.
- Descriptor, connection, or rate-limit errors downstream.

## Fixed by

<!-- remedies -->
- [Worker Pool](../patterns/worker-pool.md): Process any number of jobs with a fixed amount of concurrency.
- [Semaphore](../patterns/semaphore.md): Limit how many operations use a resource at the same time.
<!-- /remedies -->

With a semaphore, acquire before the `go` statement: acquired inside the goroutine, it bounds the work in progress
but not the number of goroutines waiting for a slot.

## Sources

<!-- sources -->
- [Harsanyi (2022)](../sources.md#source-harsanyi-2022)
<!-- /sources -->
