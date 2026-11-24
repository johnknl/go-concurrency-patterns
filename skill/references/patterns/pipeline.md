# Pipeline

Process a stream in independent steps that run at the same time, each handing its output to the next.

- **Also known as:** Pipes and filters
- **Purpose:** Data flow
- **Scale:** Pattern
- **Mechanisms:** Channels, Select, Context
- **Concerns:** Composition, Backpressure, Throughput
- **Concepts:** [Channel Ownership](../concepts/channel-ownership.md), [Goroutine Leak Prevention](../concepts/goroutine-leak-prevention.md), [Backpressure](../concepts/backpressure.md), [Ordering](../concepts/ordering.md)
- **Code:** [`patterns/pipeline`](https://github.com/johnknl/go-concurrency-patterns/tree/main/patterns/pipeline)

## Problem

A stream of values goes through several processing steps in a fixed order. Written as one loop that runs all steps
for a value before it takes the next, the steps cannot overlap in time: the time per value is the sum of all steps,
and one step's waiting holds up all the others. The steps also become one function that cannot be tested in parts.

## Mechanism

A stage is a function that takes a receive-only channel and returns a receive-only channel. Inside, one goroutine
reads until its input is closed, sends results, and closes its own output; it never closes its input
([channel ownership](../concepts/channel-ownership.md#stage-owns-output)). Stages are
connected by passing the result of one call to the next. The source is the first stage, the consumer of the last
channel is the sink.

Each stage owns one goroutine and the channel it writes to, and reads the channel of the stage before it:

Run one after another, a value passes all steps before the next value starts, so the time per value is the sum of
the steps. As a pipeline, each stage works on its own value at the same time as the others:

In this example the source is `generate` and the stages are `square` and `above`.

```go
func generate(ctx context.Context, nums ...int) <-chan int {
	out := make(chan int)

	go func() {
		defer close(out)

		for _, n := range nums {
			select {
			case out <- n:
			case <-ctx.Done():
				return
			}
		}
	}()

	return out
}

func square(ctx context.Context, in <-chan int) <-chan int {
	out := make(chan int)

	go func() {
		defer close(out)

		for n := range in {
			select {
			case out <- n * n:
			case <-ctx.Done():
				return
			}
		}
	}()

	return out
}

func above(ctx context.Context, limit int, in <-chan int) <-chan int {
	out := make(chan int)

	go func() {
		defer close(out)

		for n := range in {
			if n <= limit {
				continue
			}

			select {
			case out <- n:
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

	numbers := generate(ctx, 1, 2, 3, 4, 5, 6)

	for n := range above(ctx, 10, square(ctx, numbers)) {
		fmt.Println(n)
	}

	// Output:
	// 16
	// 25
	// 36
}
```

What makes it correct:

- **Closing propagates downstream.** When the generator closes its channel, each stage drains its input and closes
  its own ([drain then close](../concepts/channel-ownership.md#drain-then-close)).
- **Every send selects on `ctx.Done()`**, so a stage whose consumer stopped reading does not leak
  ([select on done](../concepts/goroutine-leak-prevention.md#select-on-done)). The receives can use a plain `range`,
  because on cancellation each upstream stage returns and closes its output, which ends the `range` of the next.
- **Unbuffered channels** carry a slow stage's [backpressure](../concepts/backpressure.md#feedback) back
  [to the source](../concepts/backpressure.md#reaches-source).

## Applicability

Use it when processing is a sequence of independent steps over a stream of values, especially when the steps
differ in latency, for example CPU work followed by I/O. For a few values or cheap steps, call the steps one after
another in a plain function; the channel handoffs cost more than the work
([Premature Parallelism](../anti-patterns/premature-parallelism.md)). The in-memory source in the examples stands
in for one that blocks on its own ([Channel Generator](../anti-patterns/channel-generator.md)).

## Consequences

- Order is preserved, because each stage is [sequential](../concepts/ordering.md#sequential-fifo).
- Throughput is [bounded by the slowest stage](../concepts/backpressure.md#slowest-stage); run that stage on several
  workers with
  [Fan-Out/Fan-In](fan-out-fan-in.md).
- Errors need a path of their own: send a struct with a value and an error, or cancel the shared context
  ([Cancel on Error](cancel-on-error.md)).
- Every stage is a goroutine. Leaving the last output unread blocks the whole chain until the context is canceled
  ([unread output](../concepts/goroutine-leak-prevention.md#unread-output)).

## Related patterns

- [Fan-Out/Fan-In](fan-out-fan-in.md) parallelizes one slow stage.
- [Bounded Queue](bounded-queue.md) puts a buffer between stages.

## Anti-patterns

- [Channel Generator](../anti-patterns/channel-generator.md): Produce a sequence in a goroutine of its own and hand it to the consumer through a channel, where a callback or an iterator would produce it in the consumer's goroutine.

## Sources

- [Cox-Buday (2017)](../sources.md#source-cox-buday), Ch. 4, "Pipelines"
- [Ajmani (2014b)](../sources.md#source-blog-pipelines), "What is a pipeline?"
