---
title: Premature Parallelism
intent: Spread cheap work on data that is already in hand across goroutines and channels, so the handoffs cost more than the work, the order is lost, and a loop's worth of code becomes a pipeline to manage.
purpose: execution
package: antipatterns/prematureparallel
mechanisms: [channels, sync]
concerns: [throughput, composition]
remedies: [worker-pool, fan-out-fan-in]
violates: [simplest-mechanism]
concepts:
  - ordering: 'fanning out loses the input order.'
original: true
---
# Premature Parallelism

<!-- meta -->
Spread cheap work on data that is already in hand across goroutines and channels, so the handoffs cost more than the work, the order is lost, and a loop's worth of code becomes a pipeline to manage.

<div class="cp-props" markdown>

| | |
|---|---|
| Violates | [The simplest mechanism that works](../index.md#principle-simplest-mechanism) |
| Purpose | [Execution](../catalog/purpose.md#purpose-execution) |
| Mechanisms | [Channels](../catalog/mechanism.md#mechanism-channels), [Sync](../catalog/mechanism.md#mechanism-sync) |
| Concerns | [Throughput](../catalog/concern.md#concern-throughput), [Composition](../catalog/concern.md#concern-composition) |
| Breaks | [Ordering](../concepts/ordering.md) |
| Code | [`antipatterns/prematureparallel`](https://github.com/johnknl/go-concurrency-patterns/tree/main/antipatterns/prematureparallel) |

</div>
<!-- /meta -->

## Symptoms

- A slice known up front, with cheap work per item, is fed through a channel to workers and collected from a second
  channel.
- A generator goroutine that ranges over a slice only to send its elements.
- Workers whose loop body is one cheap expression.
- A sort, or an index carried with each result, to restore the input order.
- Benchmarks in which the concurrent version is slower than a loop, and slower still with more workers.

## Example

=== "Faulty"

    `squaresParallel` makes ten channel handoffs and starts six goroutines for five multiplications, then sorts to
    get a predictable order, which is not the input order.

    <!-- code: antipatterns/prematureparallel#squaresParallel antipatterns/prematureparallel#Example_parallel:main -->

    ```go
    func squaresParallel(nums []int, workers int) (out []int, goroutines, handoffs int) {
    	var sends atomic.Int64

    	jobs := make(chan int)

    	go func() {
    		defer close(jobs)

    		for _, n := range nums {
    			jobs <- n
    			sends.Add(1)
    		}
    	}()

    	results := make(chan int)

    	var wg sync.WaitGroup
    	for range workers {
    		wg.Go(func() {
    			for n := range jobs {
    				results <- n * n
    				sends.Add(1)
    			}
    		})
    	}

    	go func() {
    		wg.Wait()
    		close(results)
    	}()

    	for r := range results {
    		out = append(out, r)
    	}

    	slices.Sort(out) // the order was lost on the way

    	return out, workers + 2, int(sends.Load())
    }

    func main() {
    	out, goroutines, handoffs := squaresParallel([]int{5, 1, 4, 2, 3}, 4)

    	fmt.Println(out)
    	fmt.Println("goroutines:", goroutines, "handoffs:", handoffs, "multiplications:", len(out))

    	// Output:
    	// [1 4 9 16 25]
    	// goroutines: 6 handoffs: 10 multiplications: 5
    }
    ```

    <!-- /code -->

=== "Fixed"

    Use a loop.

    <!-- code: antipatterns/prematureparallel#squares antipatterns/prematureparallel#Example_loop:main -->

    ```go
    func squares(nums []int) []int {
    	out := make([]int, len(nums))
    	for i, n := range nums {
    		out[i] = n * n
    	}

    	return out
    }

    func main() {
    	fmt.Println(squares([]int{5, 1, 4, 2, 3}))

    	// Output:
    	// [25 1 16 4 9]
    }
    ```

    <!-- /code -->

**When the work per item is expensive or blocks.** 
A goroutine per item with a `sync.WaitGroup`, each writing its own index of the result slice, keeps the order
without a result channel; a semaphore bounds how many run at once.

<!-- code: antipatterns/prematureparallel#mapBounded antipatterns/prematureparallel#Example_bounded:main -->
```go
func mapBounded[T, R any](in []T, limit int, f func(T) R) []R {
	out := make([]R, len(in))
	sem := make(chan struct{}, limit)

	var wg sync.WaitGroup
	for i, v := range in {
		sem <- struct{}{}

		wg.Go(func() {
			defer func() { <-sem }()

			out[i] = f(v)
		})
	}

	wg.Wait()

	return out
}

func main() {
	fetch := func(id int) string { return fmt.Sprintf("page-%d", id) } // stands in for blocking I/O

	fmt.Println(mapBounded([]int{3, 1, 2}, 2, fetch))

	// Output:
	// [page-3 page-1 page-2]
}
```
<!-- /code -->

## Why it goes wrong

A channel operation synchronizes two goroutines and can park one, which costs far more than a multiplication; the
workers spend their time waiting, and more workers add contention instead of throughput. The structure also costs
correctness work: results arrive in completion order ([ordering](../concepts/ordering.md#fan-out-loses)) and need an
index to keep the input order, and the function
owns goroutines that must all end, including the one that closes the result channel, so a later early return leaks
them.

Parallelism is justified when items block, as on I/O, or when CPU work per item is large compared to a goroutine
start and a handoff, and a measurement shows the gain. Data that arrives over time is the case for a
[Pipeline](../patterns/pipeline.md).

## Detection

- A channel fed from a slice, read by workers doing a few arithmetic operations.
- A sort after the collection loop that the sequential code did not need.
- CPU profiles dominated by `runtime.chansend`, `runtime.chanrecv`, and the scheduler.

## Fixed by

<!-- remedies -->
- [Worker Pool](../patterns/worker-pool.md): Process any number of jobs with a fixed amount of concurrency.
- [Fan-Out/Fan-In](../patterns/fan-out-fan-in.md): Process independent items at the same time and collect all their results.
<!-- /remedies -->

## Sources

<!-- sources -->
Original work.
<!-- /sources -->
