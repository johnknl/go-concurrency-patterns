---
title: For-Select Loop
intent: Keep a goroutine waiting on several events, or doing work, while it still notices a stop signal.
purpose: coordination
scale: idiom
package: patterns/forselect
mechanisms: [channels, select, context]
concerns: [composition, cancellation]
concepts:
  - cooperative-cancellation: 'every blocking case selects on Done, which select does not prefer.'
  - channel-ownership: 'a closed input is always ready and must be disabled.'
  - goroutine-leak-prevention: 'every loop has an exit.'
  - confinement: 'an owner goroutine is a for-select loop.'
sources:
  - id: cox-buday
    at: 'Ch. 4, "The for-select Loop"'
---
# For-Select Loop

<!-- meta -->
Keep a goroutine waiting on several events, or doing work, while it still notices a stop signal.

<div class="cp-props" markdown>

| | |
|---|---|
| Purpose | [Coordination](../catalog/purpose.md#purpose-coordination) |
| Scale | [Idiom](../catalog/scale.md#scale-idiom) |
| Mechanisms | [Channels](../catalog/mechanism.md#mechanism-channels), [Select](../catalog/mechanism.md#mechanism-select), [Context](../catalog/mechanism.md#mechanism-context) |
| Concerns | [Composition](../catalog/concern.md#concern-composition), [Cancellation](../catalog/concern.md#concern-cancellation) |
| Concepts | [Cooperative Cancellation](../concepts/cooperative-cancellation.md), [Channel Ownership](../concepts/channel-ownership.md), [Goroutine Leak Prevention](../concepts/goroutine-leak-prevention.md), [Confinement](../concepts/confinement.md) |
| Code | [`patterns/forselect`](https://github.com/johnknl/go-concurrency-patterns/tree/main/patterns/forselect) |

</div>
<!-- /meta -->

## Problem

A goroutine that loops over channel operations needs a way out: a plain send or receive whose peer is gone blocks
forever.

## Mechanism

The loop body is a `select` with one case for the stop signal, which returns, and cases for the real work; every
other case leads back to the top. The stop signal is `ctx.Done()` of a
[context](https://pkg.go.dev/context), which is closed when the context is canceled. The most common form sends each
item in its own `select` together with the `ctx.Done()` case.

[Cox-Buday (2017)](../sources.md#source-cox-buday) uses a plain `done` channel that the caller closes. This page uses
a context instead, which is the idiomatic form today: it carries the same signal and also deadlines and the
cancellation cause, and many library calls accept it. A `done` channel works the same way in each loop below.

<!-- code: patterns/forselect#send patterns/forselect#Example_iterationVariable:main -->
```go
func send(ctx context.Context, out chan<- string, items []string) {
	for _, item := range items {
		select {
		case <-ctx.Done():
			return
		case out <- item:
		}
	}
}

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	out := make(chan string)

	go func() {
		defer close(out)

		send(ctx, out, []string{"a", "b", "c"})
	}()

	for s := range out {
		fmt.Println(s)
	}

	// Output:
	// a
	// b
	// c
}
```
<!-- /code -->

## Applicability

Use it when a goroutine serves more than one channel, or sends or receives a value that may never be matched and
must stay stoppable. A goroutine that only reads one channel until it is closed needs only `for range`.

## Consequences

- A case body that blocks is not watching `ctx.Done()`; it must select on it too
  ([every blocking point selects on Done](../concepts/cooperative-cancellation.md#blocking-selects-done)).
- With `default`, the loop spins when idle; use it only when each turn does real work.
- A pending cancellation is not always taken first
  ([select does not prefer cancellation](../concepts/cooperative-cancellation.md#select-random)). Code that must not
  do more work after cancellation checks `ctx.Err()` again, or in a separate `select` before the others.
- Cancellation is noticed between iterations: one step of work is the granularity of stopping.
- A receive from a closed channel is always ready: test `ok` and return, or set the channel to `nil`
  ([a closed channel is always ready](../concepts/channel-ownership.md#closed-always-ready)).
- `break` inside `select` leaves only the `select`. Use `return` or a labeled break.

## Variants

### Block until ready

With no `default` case, `select` blocks until one case is ready, so the goroutine uses no CPU while idle.
`wait` assumes that `in` is never closed.

<!-- code: patterns/forselect#wait patterns/forselect#Example_untilStopped:main -->
```go
func wait(ctx context.Context, in <-chan int) (sum int) {
	for {
		select {
		case <-ctx.Done():
			return sum
		case n := <-in:
			sum += n
		}
	}
}

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	in := make(chan int)
	result := make(chan int)

	go func() {
		result <- wait(ctx, in)
	}()

	in <- 1
	in <- 2
	cancel()
	fmt.Println(<-result)

	// Output:
	// 3
}
```
<!-- /code -->

### Work in the default case

If no other case is ready, `select` runs `default` at once: a step of work, then the context is checked again.

<!-- code: patterns/forselect#work patterns/forselect#Example_defaultWork:main -->
```go
func work(ctx context.Context, started chan<- struct{}) (steps int) {
	for {
		select {
		case <-ctx.Done():
			return steps
		default:
		}

		steps++

		if steps == 1 {
			close(started)
		}
	}
}

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	started := make(chan struct{})
	result := make(chan int)

	go func() {
		result <- work(ctx, started)
	}()

	<-started
	cancel()
	fmt.Println(<-result >= 1)

	// Output:
	// true
}
```
<!-- /code -->

## Related patterns

- [Goroutine Leak Prevention](../concepts/goroutine-leak-prevention.md#observable-end) explains why every loop needs
  an exit, and how its end is observed.
- [Pipeline](pipeline.md) stages are for-select loops.
- [Confinement](../concepts/confinement.md#owner-goroutine) uses the loop to run an owner goroutine.

## Sources

<!-- sources -->
- [Cox-Buday (2017)](../sources.md#source-cox-buday), Ch. 4, "The for-select Loop"
<!-- /sources -->
