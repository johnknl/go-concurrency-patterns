---
title: Batch Without a Time Bound
intent: Collect items into batches that are flushed only when full, so at low load an item waits indefinitely and the last partial batch is lost at shutdown.
purpose: data-flow
package: antipatterns/untimedbatch
mechanisms: [channels]
concerns: [batching, latency]
remedies: [batcher, group-commit]
violates: [bounded]
original: true
---
# Batch Without a Time Bound

<!-- meta -->
Collect items into batches that are flushed only when full, so at low load an item waits indefinitely and the last partial batch is lost at shutdown.

<div class="cp-props" markdown>

| | |
|---|---|
| Violates | [Bound everything](../index.md#principle-bounded) |
| Purpose | [Data flow](../catalog/purpose.md#purpose-data-flow) |
| Mechanisms | [Channels](../catalog/mechanism.md#mechanism-channels) |
| Concerns | [Batching](../catalog/concern.md#concern-batching), [Latency](../catalog/concern.md#concern-latency) |
| Code | [`antipatterns/untimedbatch`](https://github.com/johnknl/go-concurrency-patterns/tree/main/antipatterns/untimedbatch) |

</div>
<!-- /meta -->

## Symptoms

- Metrics, logs, or events show up late, or only in bursts, during quiet periods.
- The last few items before a shutdown or a deploy are missing.
- A test that sends fewer items than the batch size hangs or sees nothing. Tests that send full batches pass.

## Example

=== "Faulty"

    `sizeOnly` flushes only when the batch has `maxSize` items. Two items sent to a batcher of four are never
    flushed, and closing the input drops them. The tests show the same after three fake days.

    <!-- code: antipatterns/untimedbatch#sizeOnly antipatterns/untimedbatch#Example_sizeOnly:main -->

    ```go
    func sizeOnly(in <-chan string, maxSize int, flush func([]string)) <-chan struct{} {
    	done := make(chan struct{})

    	go func() {
    		defer close(done)

    		var batch []string

    		for item := range in {
    			batch = append(batch, item)
    			if len(batch) == maxSize {
    				flush(batch)
    				batch = nil
    			}
    		}
    	}()

    	return done
    }

    func main() {
    	in := make(chan string, 8)
    	flushed := make(chan []string, 8)
    	done := sizeOnly(in, 4, func(batch []string) { flushed <- batch })

    	in <- "a"
    	in <- "b"

    	select {
    	case batch := <-flushed:
    		fmt.Println("flush", batch)
    	case <-time.After(50 * time.Millisecond):
    		fmt.Println("nothing flushed yet")
    	}

    	close(in)
    	<-done
    	fmt.Println("flushed after close:", len(flushed))

    	// Output:
    	// nothing flushed yet
    	// flushed after close: 0
    }
    ```

    <!-- /code -->

=== "Fixed"

    The [Batcher](../patterns/batcher.md) starts a timer at the first item of each batch, flushes when it fires,
    and flushes the rest when the input is closed.

    <!-- code: antipatterns/untimedbatch#sizeOrTime antipatterns/untimedbatch#Example_timeBound:main -->

    ```go
    func sizeOrTime(in <-chan string, maxSize int, maxWait time.Duration, flush func([]string)) <-chan struct{} {
    	done := make(chan struct{})

    	go func() {
    		defer close(done)

    		timer := time.NewTimer(maxWait)
    		timer.Stop()

    		var batch []string

    		send := func() {
    			timer.Stop()
    			flush(batch)
    			batch = nil
    		}

    		for {
    			select {
    			case item, ok := <-in:
    				if !ok {
    					if len(batch) > 0 {
    						send()
    					}

    					return
    				}

    				batch = append(batch, item)
    				if len(batch) == 1 {
    					timer.Reset(maxWait)
    				}

    				if len(batch) == maxSize {
    					send()
    				}
    			case <-timer.C:
    				send()
    			}
    		}
    	}()

    	return done
    }

    func main() {
    	in := make(chan string, 8)
    	flushed := make(chan []string, 8)
    	done := sizeOrTime(in, 4, 10*time.Millisecond, func(batch []string) { flushed <- batch })

    	in <- "a"
    	fmt.Println("flush", <-flushed)

    	in <- "b"
    	close(in)
    	<-done
    	fmt.Println("flush", <-flushed)

    	// Output:
    	// flush [a]
    	// flush [b]
    }
    ```

    <!-- /code -->

## Why it goes wrong

The wait of an item depends on the arrival of later items, which the batcher does not control. The batch size bounds
memory, but nothing bounds the latency of an item, and nothing decides what happens when it has waited too long. The
missing flush on close has the same cause: the code flushes only on a condition about later items, and the end of
the input is the case where none come.

## Detection

- A batching loop whose only flush condition is `len(batch) == maxSize`, with no timer, ticker, or deadline in its
  `select`.
- A `for item := range in` loop that ends without using `batch`.
- Tests that only send multiples of the batch size.

## Fixed by

<!-- remedies -->
- [Batcher](../patterns/batcher.md): Spread a per-call cost over a stream of items while bounding how long any item waits.
- [Group Commit](../patterns/group-commit.md): Let concurrent callers share one execution of an expensive operation, without adding latency when only one caller is waiting.
<!-- /remedies -->

When callers wait for the result of their own item, [Group Commit](../patterns/group-commit.md) needs no timer: on
an idle system a caller flushes at once, and batches form only while a flush is running.

## Sources

<!-- sources -->
Original work.
<!-- /sources -->
