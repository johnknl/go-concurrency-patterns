# Tee

Give each value of one stream to two consumers that process it independently.

- **Purpose:** Data flow
- **Scale:** Idiom
- **Mechanisms:** Channels, Select
- **Concerns:** Composition
- **Concepts:** [Channel Ownership](../concepts/channel-ownership.md), [Goroutine Leak Prevention](../concepts/goroutine-leak-prevention.md)
- **Code:** [`patterns/tee`](https://github.com/johnknl/go-concurrency-patterns/tree/main/patterns/tee)

## Problem

A value sent on a channel goes to one receiver. When two consumers each need every value of a stream, the stream has
to be duplicated. Cox-Buday calls this a tee-channel.

## Mechanism

For each value, `tee` takes local copies of the two output channels and runs a `select` twice; a successful send
sets its copy to nil, and a nil channel is never ready, so the second round sends to the other output. Either can
go first. Every receive and send also selects on `ctx.Done()`, so cancellation ends the loop at any point
([select on done](../concepts/goroutine-leak-prevention.md#select-on-done)). Both outputs, owned by `tee`, close when
the input ends or the context is canceled ([stage owns output](../concepts/channel-ownership.md#stage-owns-output)).

```go
func tee(ctx context.Context, in <-chan int) (left, right <-chan int) {
	out1, out2 := make(chan int), make(chan int)

	go func() {
		defer close(out1)
		defer close(out2)

		for {
			var v int

			select {
			case n, ok := <-in:
				if !ok {
					return
				}

				v = n
			case <-ctx.Done():
				return
			}

			c1, c2 := out1, out2

			for range 2 {
				select {
				case c1 <- v:
					c1 = nil
				case c2 <- v:
					c2 = nil
				case <-ctx.Done():
					return
				}
			}
		}
	}()

	return out1, out2
}

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	a, b := tee(ctx, generate(ctx, 1, 2, 3))

	var first, second []int

	var wg sync.WaitGroup
	wg.Go(func() { first = collect(a) })
	wg.Go(func() { second = collect(b) })
	wg.Wait()

	fmt.Println(first)
	fmt.Println(second)

	// Output:
	// [1 2 3]
	// [1 2 3]
}
```

## Applicability

Use it when two consumers need every value at the same pace, such as branches of a [Pipeline](pipeline.md). When
both are plain functions, one loop calling both needs no tee. For consumers with independent pace, use a
[PubSub Hub](pubsub-hub.md) with a buffer per subscriber. The in-memory source of the example stands in for a
blocking one ([Channel Generator](../anti-patterns/channel-generator.md)).

## Consequences

- The slower consumer sets the pace of both. The consumers must run concurrently: reading one output to the end
  first deadlocks.
- An unread output blocks the stream until the context is canceled
  ([unread output](../concepts/goroutine-leak-prevention.md#unread-output)).
- A pointer, slice, or map value is shared by both consumers, which must not modify it without synchronization.
- More than two outputs need a larger loop or nested tees.

## Related patterns

- [Or-Done](or-done.md) lets the reader of the input stop on done.
- [Fan-Out/Fan-In](fan-out-fan-in.md) divides values among consumers instead of copying them.

## Sources

- [Cox-Buday (2017)](../sources.md#source-cox-buday), Ch. 4, "The tee-channel"
