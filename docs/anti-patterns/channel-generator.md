---
title: Channel Generator
intent: Produce a sequence in a goroutine of its own and hand it to the consumer through a channel, where a callback or an iterator would produce it in the consumer's goroutine.
purpose: data-flow
package: antipatterns/generator
mechanisms: [channels, select, context]
concerns: [leak-prevention, composition]
remedies: [pipeline]
violates: [synchronous-first]
concepts:
  - goroutine-leak-prevention: 'the consumer must end a producer it did not start.'
sources:
  - id: cox-buday
    at: 'Ch. 4, "Some Handy Generators"'
  - id: pike-2012
    at: 'Generator: function that returns a channel'
---
# Channel Generator

<!-- meta -->
Produce a sequence in a goroutine of its own and hand it to the consumer through a channel, where a callback or an iterator would produce it in the consumer's goroutine.

<div class="cp-props" markdown>

| | |
|---|---|
| Violates | [Synchronous first](../index.md#principle-synchronous-first) |
| Purpose | [Data flow](../catalog/purpose.md#purpose-data-flow) |
| Mechanisms | [Channels](../catalog/mechanism.md#mechanism-channels), [Select](../catalog/mechanism.md#mechanism-select), [Context](../catalog/mechanism.md#mechanism-context) |
| Concerns | [Leak prevention](../catalog/concern.md#concern-leak-prevention), [Composition](../catalog/concern.md#concern-composition) |
| Breaks | [Goroutine Leak Prevention](../concepts/goroutine-leak-prevention.md) |
| Code | [`antipatterns/generator`](https://github.com/johnknl/go-concurrency-patterns/tree/main/antipatterns/generator) |

</div>
<!-- /meta -->

## Symptoms

- Every consumer has to cancel a context, or drain the channel, to stop the producer; one that `break`s out of
  its loop and returns leaks the goroutine.
- A panic in the producer crashes the program, and the consumer cannot recover it.
- The function takes a context only so that its goroutine can end.
- Each value costs a channel handoff between two goroutines, many times the cost of a cheap computation.

## Example

=== "Faulty"

    `naturals` is a careful channel generator: the send selects on the context, so a consumer that cancels does
    stop it. The consumer still has to carry that context and cancel it, here with a deferred `cancel`.

    <!-- code: antipatterns/generator#naturals antipatterns/generator#Example_channel:main -->

    ```go
    func naturals(ctx context.Context) <-chan int {
    	out := make(chan int)

    	go func() {
    		defer close(out)

    		for i := 0; ; i++ {
    			select {
    			case out <- i:
    			case <-ctx.Done():
    				return
    			}
    		}
    	}()

    	return out
    }

    func main() {
    	ctx, cancel := context.WithCancel(context.Background())
    	defer cancel()

    	for n := range naturals(ctx) {
    		if n == 5 {
    			break
    		}

    		fmt.Println(n)
    	}

    	// Output:
    	// 0
    	// 1
    	// 2
    	// 3
    	// 4
    }
    ```

    <!-- /code -->

=== "Fixed"

    Return an iterator. The values are produced in the consumer's goroutine, `break` stops the sequence, a panic
    reaches the consumer, and there is nothing to cancel.

    <!-- code: antipatterns/generator#naturalsSeq antipatterns/generator#Example_iterator:main -->

    ```go
    func naturalsSeq() iter.Seq[int] {
    	return func(yield func(int) bool) {
    		for i := 0; ; i++ {
    			if !yield(i) {
    				return
    			}
    		}
    	}
    }

    func main() {
    	for n := range naturalsSeq() {
    		if n == 3 {
    			break
    		}

    		fmt.Println(n)
    	}

    	// Output:
    	// 0
    	// 1
    	// 2
    }
    ```

    <!-- /code -->

## Why it goes wrong

It is often presented as the Generator pattern, because it shows goroutines and channels at work. But a sequence that
the program computes itself needs nothing running beside the consumer ([synchronous
first](../index.md#principle-synchronous-first)); the goroutine only moves the work out of the caller's control. The
producer then ends only when the sequence ends, the consumer drains it, or the consumer cancels. Every caller has to get
that right, and those that do not leak a goroutine: the function started it, but the caller must end it ([goroutine leak
prevention](../concepts/goroutine-leak-prevention.md#owner-ends)). The values cross a goroutine boundary one at a time,
and a panic stays on the producer's side of it.

A callback (`filepath.Walk`, `sync.Map.Range`), a `Next` method (`bufio.Scanner.Scan`), or since Go 1.23 an
iterator, which is a callback that `range` understands, produces the same values synchronously.

## Detection

- A function that starts a goroutine whose only job is to compute values and send them, and returns the channel.
- At call sites: a context created and canceled only to stop the producer, or a drain loop after a `break`.
- Goroutine profiles and the goroutine leak profile show producers blocked on a send after their consumer left.
  `go vet` and the race detector do not report it.

## Fixed by

<!-- remedies -->
- [Pipeline](../patterns/pipeline.md): Process a stream in independent steps that run at the same time, each handing its output to the next.
<!-- /remedies -->

A goroutine is justified when the source blocks on its own (a socket, a file, a ticker, a signal) and the consumer
should not wait for it. That is a source stage of a [Pipeline](../patterns/pipeline.md), with a context and an owner.

## Sources

Pike (2012) uses the goroutine generator to demonstrate channels, and Cox-Buday (2017) to feed pipelines; neither
presents it as the way to return a sequence from an API.

<!-- sources -->
- [Cox-Buday (2017)](../sources.md#source-cox-buday), Ch. 4, "Some Handy Generators"
- [Pike (2012)](../sources.md#source-pike-2012), Generator: function that returns a channel
<!-- /sources -->
