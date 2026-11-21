---
title: Future
aliases: ['Promise', 'Task', 'Deferred']
intent: Overlap independent operations and collect each result when it is needed.
purpose: execution
scale: idiom
package: patterns/future
mechanisms: [channels, select]
concerns: [latency, error-handling]
concepts:
  - goroutine-leak-prevention: 'the result channel has a buffer of one, so the goroutine always ends.'
  - memory-visibility: 'the send of the result publishes it to the receiver.'
sources:
  - id: baker-hewitt-1977
---
# Future

<!-- meta -->
Overlap independent operations and collect each result when it is needed.

<div class="cp-props" markdown>

| | |
|---|---|
| Also known as | Promise, Task, Deferred |
| Purpose | [Execution](../catalog/purpose.md#purpose-execution) |
| Scale | [Idiom](../catalog/scale.md#scale-idiom) |
| Mechanisms | [Channels](../catalog/mechanism.md#mechanism-channels), [Select](../catalog/mechanism.md#mechanism-select) |
| Concerns | [Latency](../catalog/concern.md#concern-latency), [Error handling](../catalog/concern.md#concern-error-handling) |
| Concepts | [Goroutine Leak Prevention](../concepts/goroutine-leak-prevention.md), [Memory Visibility](../concepts/memory-visibility.md) |
| Code | [`patterns/future`](https://github.com/johnknl/go-concurrency-patterns/tree/main/patterns/future) |

</div>
<!-- /meta -->

## Problem

A computation needs the results of several independent, slow operations. Run one after another they cost the sum of
their latencies; started together and collected when needed, they cost about the longest one.

## Mechanism

The caller wraps a synchronous function: it starts a goroutine that calls the function and sends the result on a
channel with a buffer of one, then receives when it needs the value. `start` is that caller-side helper, local to
the code that joins the results; an API never returns a future
([Needless Asynchrony](../anti-patterns/needless-asynchrony.md)).

<!-- code: patterns/future#square patterns/future#result patterns/future#start patterns/future#Example_future:main -->
```go
func square(n int) (int, error) {
	if n < 0 {
		return 0, errNegative
	}

	return n * n, nil
}

type result struct {
	err error
	val int
}

func start(fn func() (int, error)) <-chan result {
	ch := make(chan result, 1)

	go func() {
		v, err := fn()
		ch <- result{val: v, err: err}
	}()

	return ch
}

func main() {
	a := start(func() (int, error) { return square(3) })
	b := start(func() (int, error) { return square(4) })

	fmt.Println("doing other work")

	ra, rb := <-a, <-b
	fmt.Println(ra.val+rb.val, ra.err, rb.err)

	// Output:
	// doing other work
	// 25 <nil> <nil>
}
```
<!-- /code -->

The buffer of one is essential: the goroutine completes its send even when the caller never receives, so it ends
once the computation returns ([a reply channel has a buffer of one](../concepts/goroutine-leak-prevention.md#reply-buffer-one)).
The send happens before the receive completes, so the caller sees the result with no further synchronization
([memory visibility](../concepts/memory-visibility.md#send-before-receive)).

## Applicability

Use it when a few independent, blocking operations are started together and joined at different points. When they
are all joined at once, `go` with a `sync.WaitGroup` writing to local variables is simpler
([Fan-Out/Fan-In](fan-out-fan-in.md)). Use a [Pipeline](pipeline.md) for a stream, and a
[Worker Pool](worker-pool.md) for many jobs.

## Consequences

- The channel yields its value once; a second receive blocks forever. To let several goroutines wait, store the
  result and then close a channel ([store before the close](../concepts/memory-visibility.md#store-before-close)).
- The work starts at the call whether or not anyone receives, and giving up on the result does not stop it. Pass a
  context into the computation if it should stop; the code that starts it is responsible for its end
  ([goroutine leak prevention](../concepts/goroutine-leak-prevention.md#owner-ends)).
- A channel carries one value, so the error goes in the result.

## Variants

### Bounded wait

Receive in a `select` that also waits on a context. Cancellation ends the wait but not the computation, whose
result can still be received later.

<!-- code: patterns/future#Example_timeout:main -->
```go
func main() {
	release := make(chan struct{})
	slow := make(chan result, 1)

	go func() {
		<-release
		slow <- result{val: 1}
	}()

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // stands in for a deadline that has passed

	select {
	case r := <-slow:
		fmt.Println("got", r.val)
	case <-ctx.Done():
		fmt.Println("gave up:", ctx.Err())
	}

	close(release)
	fmt.Println("later:", (<-slow).val)

	// Output:
	// gave up: context canceled
	// later: 1
}
```
<!-- /code -->

## Related patterns

- [Worker Pool](worker-pool.md) runs many jobs on bounded workers.
- [Semaphore](semaphore.md) can limit how many futures run at once.

## Anti-patterns

<!-- antipatterns -->
- [Needless Asynchrony](../anti-patterns/needless-asynchrony.md): Start a goroutine inside a function and hand back a channel, where returning the result would do, so the caller loses the error path, the cancellation, and control over the goroutine.
- [Premature Abstraction](../anti-patterns/premature-abstraction.md): Extract a generic, configurable concurrency helper before its uses are known, so each caller sets flags for the behavior it needs, the flags interact, and every caller depends on code paths it does not use.
- [Sleeping to Synchronize](../anti-patterns/sleep-synchronization.md): Use time.Sleep or a guessed delay to wait for another goroutine instead of waiting for its completion.
<!-- /antipatterns -->

## Sources

<!-- sources -->
- [Baker and Hewitt (1977)](../sources.md#source-baker-hewitt-1977)
<!-- /sources -->
