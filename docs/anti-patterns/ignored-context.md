---
title: Ignored Context
intent: A function takes a context and never checks it, so cancellation and deadlines have no effect.
purpose: lifecycle
package: antipatterns/ignoredcontext
mechanisms: [context, select, channels]
concerns: [cancellation, leak-prevention]
remedies: [context-propagation, deadline]
violates: [ownership]
concepts:
  - cooperative-cancellation: 'the function never selects on Done or checks Err.'
sources:
  - id: blog-context
---
# Ignored Context

<!-- meta -->
A function takes a context and never checks it, so cancellation and deadlines have no effect.

<div class="cp-props" markdown>

| | |
|---|---|
| Violates | [Clear ownership](../index.md#principle-ownership) |
| Purpose | [Lifecycle](../catalog/purpose.md#purpose-lifecycle) |
| Mechanisms | [Context](../catalog/mechanism.md#mechanism-context), [Select](../catalog/mechanism.md#mechanism-select), [Channels](../catalog/mechanism.md#mechanism-channels) |
| Concerns | [Cancellation](../catalog/concern.md#concern-cancellation), [Leak prevention](../catalog/concern.md#concern-leak-prevention) |
| Breaks | [Cooperative Cancellation](../concepts/cooperative-cancellation.md) |
| Code | [`antipatterns/ignoredcontext`](https://github.com/johnknl/go-concurrency-patterns/tree/main/antipatterns/ignoredcontext) |

</div>
<!-- /meta -->

## Symptoms

- Requests the client has abandoned keep using workers, connections, and memory until the work ends.
- Timeouts do not shorten calls, and shutdown waits for the slowest operation.
- Goroutine counts rise under load; a dump shows goroutines blocked in the same receive or call long after their
  contexts were canceled.

## Example

=== "Faulty"

    `badWait` blocks on a gate channel, standing in for anything other than its context. After cancel, `ctx.Err()`
    is set but the function has not returned (a `select` with `default` probes this) until the gate opens.

    <!-- code: antipatterns/ignoredcontext#badWait antipatterns/ignoredcontext#Example_ignoredContext:main -->

    ```go
    func badWait(_ context.Context, gate <-chan struct{}) error {
    	<-gate
    	return nil
    }

    func main() {
    	gate := make(chan struct{})
    	ctx, cancel := context.WithCancel(context.Background())
    	done := make(chan error, 1)
    	go func() { done <- badWait(ctx, gate) }()

    	cancel()
    	fmt.Println("ctx.Err:", ctx.Err())
    	select {
    	case <-done:
    		fmt.Println("returned")
    	default:
    		fmt.Println("still running")
    	}

    	close(gate)
    	fmt.Println("result:", <-done)
    	// Output:
    	// ctx.Err: context canceled
    	// still running
    	// result: <nil>
    }
    ```

    <!-- /code -->

=== "Fixed"

    `wait` selects on `ctx.Done()` around the blocking step and returns at once after `cancel`, with
    `context.Cause` reporting the reason given to `WithCancelCause`.

    <!-- code: antipatterns/ignoredcontext#wait antipatterns/ignoredcontext#Example_fixed:main -->

    ```go
    func wait(ctx context.Context, gate <-chan struct{}) error {
    	select {
    	case <-gate:
    		return nil
    	case <-ctx.Done():
    		return context.Cause(ctx)
    	}
    }

    func main() {
    	gate := make(chan struct{})
    	ctx, cancel := context.WithCancelCause(context.Background())
    	done := make(chan error, 1)
    	go func() { done <- wait(ctx, gate) }()

    	cancel(errors.New("client went away"))
    	fmt.Println("result:", <-done)
    	// Output: result: client went away
    }
    ```

    <!-- /code -->

## Why it goes wrong

Canceling a context stops no goroutine; it only
[signals](../concepts/cooperative-cancellation.md#signal-not-effect). A function blocked on a channel receive, a lock,
or a call that takes no context cannot be interrupted from outside, so its caller
[waits](../concepts/cooperative-cancellation.md#ignore-delays) however the context was canceled.

Select on `ctx.Done()` around [every step that can
block](../concepts/cooperative-cancellation.md#blocking-selects-done), check `ctx.Err()` [between units of CPU-bound
work](../concepts/cooperative-cancellation.md#check-points),
[return](../concepts/cooperative-cancellation.md#return-cause) `ctx.Err()` or `context.Cause(ctx)`, and pass the context
on to calls that take one.

## Detection

- A test that cancels while the work is blocked, as in the example, and checks that the function returns.
- Goroutine dumps and a rising goroutine count show the blocked goroutines.
- A `ctx` parameter never used in the body, which linters for unused parameters report; `go vet` does not
  (`lostcancel` only checks that a cancel function is called). Passing `ctx` only to functions that also ignore it
  does not count as using it.

## Fixed by

<!-- remedies -->
- [Context Propagation](../stdlib/context-propagation.md): Stop all the work done for a request together, and give it the request's deadline and scoped values.
- [Deadline](../stdlib/deadline.md): Bound how long an operation may take.
<!-- /remedies -->

## Sources

<!-- sources -->
- [Ajmani (2014a)](../sources.md#source-blog-context)
<!-- /sources -->
