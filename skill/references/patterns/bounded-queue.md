# Bounded Queue

Let a producer run ahead of its consumer by a fixed, chosen amount.

- **Also known as:** Buffered channel, Queuing
- **Purpose:** Data flow
- **Scale:** Idiom
- **Mechanisms:** Channels
- **Concerns:** Backpressure, Throughput, Latency
- **Concepts:** [Channel Ownership](../concepts/channel-ownership.md), [Backpressure](../concepts/backpressure.md), [Overload Policy](../concepts/overload-policy.md)
- **Code:** [`patterns/boundedqueue`](https://github.com/johnknl/go-concurrency-patterns/tree/main/patterns/boundedqueue)

## Problem

When a producer hands values directly to a consumer, a short stall or burst on one side stalls the other. A buffer
between them decouples them, but its capacity decides how much memory and delay a backlog may cost.

## Mechanism

`make(chan T, n)` holds up to `n` values in FIFO order. A send blocks only when the buffer is full, a receive only
when it is empty. After `close`, receivers still get the queued values, and `range` ends once the buffer is empty
([drain then close](../concepts/channel-ownership.md#drain-then-close)).
In this example the capacity is 4, so `fill` returns before anything is read; a fifth value would block it forever.

```go
func fill(queue chan<- int, n int) {
	for i := range n {
		queue <- i
	}

	close(queue)
}

func main() {
	queue := make(chan int, 4)

	fill(queue, 4)
	fmt.Println("len:", len(queue), "cap:", cap(queue))

	for n := range queue {
		fmt.Println("got", n)
	}

	fmt.Println("len:", len(queue), "cap:", cap(queue))

	// Output:
	// len: 4 cap: 4
	// got 0
	// got 1
	// got 2
	// got 3
	// len: 0 cap: 4
}
```

When the buffer is full, a plain send waits for the consumer ([backpressure](../concepts/backpressure.md#feedback))
unless the producer rejects the value. Like [every limit](../concepts/overload-policy.md#every-limit), it needs a
chosen policy; the capacity only [delays that choice](../concepts/overload-policy.md#buffer-delays).

Two implementation concerns come with the queue: what a producer does when it is full, and how a consumer takes
what has piled up.

**Reject when full.** A send in a `select` with `default` fails at once when the buffer is full, so the producer
can drop or report the value ([Load Shedding](load-shedding.md)).

```go
func main() {
	queue := make(chan int, 2)
	queue <- 1
	queue <- 2

	select {
	case queue <- 3:
		fmt.Println("accepted")
	default:
		fmt.Println("full, producer would wait")
	}

	// Output:
	// full, producer would wait
}
```

**Drain as one batch.** A consumer that is cheaper per batch takes everything queued, up to a limit, in one
receive loop. In this example the limit is 3.

```go
func main() {
	queue := make(chan int, 8)
	fill(queue, 7)

	for {
		batch := make([]int, 0, 3)

		for n := range queue {
			batch = append(batch, n)

			if len(batch) == cap(batch) {
				break
			}
		}

		if len(batch) == 0 {
			break
		}

		fmt.Println("write", batch)
	}

	// Output:
	// write [0 1 2]
	// write [3 4 5]
	// write [6]
}
```

## Applicability

Use it at the entry of a pipeline, so the first stage accepts work while later stages are busy, or before a stage
that handles queued items together. It speeds nothing up in front of the slowest stage, and a producer that is
permanently faster still fills it. The example's in-memory source stands in for a blocking one
([Channel Generator](../anti-patterns/channel-generator.md)).

## Consequences

- The long-run rate is that of the [slowest stage](../concepts/backpressure.md#slowest-stage); a buffer only absorbs
  bursts.
- Queued values wait for those ahead. By Little's law, the items in a stable system equal the arrival rate times
  the time each spends in it (`L = λ · W`), so a fuller queue means a longer wait: 1000 values served at 100 per
  second delay the last by 10 s.
- A large buffer shows overload late and costs its full capacity up front; size it from a known burst or batch.
- Queued values are lost if the consumer stops without draining.
- An unbounded queue removes the backpressure ([no bound, no feedback](../concepts/backpressure.md#needs-bound));
  memory and latency then grow without limit.

## Related patterns

- [Pipeline](pipeline.md) places queues between stages.
- [Semaphore](semaphore.md) uses a buffered channel to limit concurrency.

## Anti-patterns

- [Buffer as a Band-Aid](../anti-patterns/oversized-buffer.md): Hide a producer that outruns its consumer behind a huge channel buffer, so the failure comes later and larger.
- [Goroutine as a Buffer](../anti-patterns/goroutine-as-buffer.md): Park a goroutine on a channel send so that the caller never blocks, which makes every pending value cost a goroutine instead of a channel slot.

## Sources

- [Cox-Buday (2017)](../sources.md#source-cox-buday), Ch. 4, "Queuing"
