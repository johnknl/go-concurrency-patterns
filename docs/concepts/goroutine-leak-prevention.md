---
title: Goroutine Leak Prevention
aliases: ['Goroutine lifetime management', 'Done channel']
intent: Give every goroutine an owner that is responsible for its end, and a way to end that does not depend on a peer that may have left.
concerns: [leak-prevention, cancellation]
sources:
  - id: cox-buday
    at: 'Ch. 4, "Preventing Goroutine Leaks"'
  - id: blog-pipelines
    at: '"Explicit cancellation"'
---
# Goroutine Leak Prevention

<!-- meta -->
Give every goroutine an owner that is responsible for its end, and a way to end that does not depend on a peer that may have left.

<div class="cp-props" markdown>

| | |
|---|---|
| Also known as | Goroutine lifetime management, Done channel |
| Concerns | [Leak prevention](../catalog/concern.md#concern-leak-prevention), [Cancellation](../catalog/concern.md#concern-cancellation) |

</div>
<!-- /meta -->

## Why it matters

The runtime does not collect goroutines. A goroutine blocked on a channel that nobody will use again keeps its stack
and everything it references for the life of the process. Leaks are silent: nothing fails, memory and goroutine
counts grow, and a goroutine that wakes up late may act on state its creator has already abandoned. A goroutine is
leak-free only if its end is planned at the moment it is started.

## Properties

### The owner is responsible for the end { #owner-ends }

Whoever starts a goroutine owns it: it decides when the goroutine must stop, gives it the means to stop (a context
or a done channel), and does not return, or does not consider its own work finished, while the goroutine may still
run. Ownership can be handed on, but there is always exactly one owner.

### The end is observable { #observable-end }

Telling a goroutine to stop is not the same as knowing it stopped. The goroutine publishes a termination notice: it
closes a channel, calls `wg.Done`, or returns through `errgroup`. The owner waits on that notice before it releases
what the goroutine uses or reports that it has shut down.

```go
terminated := make(chan struct{})
go func() {
	defer close(terminated)
	// ...
}()
```

### Every send and receive whose peer may leave selects on done { #select-on-done }

A goroutine blocks forever if it sends to a receiver that stopped reading, or receives from a sender that will never
send or close. Every channel operation whose peer may go away is therefore one case of a `select` that also waits on
`ctx.Done()` or a done channel. It is the peer leaving that matters, not the direction: a send in a producer and a
receive in a consumer both need it.

### An unread output stalls the chain { #unread-output }

In a chain of goroutines connected by channels, a consumer that stops reading blocks its producer at the next send,
which blocks the producer's producer, up to the source. One stage that returns without draining or canceling leaks
every stage upstream of it. Stopping a chain means canceling it, not just abandoning its output.

### A reply channel has a buffer of one { #reply-buffer-one }

When a goroutine answers one request on a channel created for it, give that channel a capacity of one. The answer
then fits in the buffer, and the answering goroutine finishes even if the requester gave up (on a timeout or
cancellation) and will never receive. With an unbuffered reply channel the answerer blocks forever in that case.

```go
reply := make(chan result, 1)
```

## Used by

<!-- usedby -->
- [Active Object](../patterns/active-object.md): sends select on done, and replies have a buffer of one.
- [Fan-Out/Fan-In](../patterns/fan-out-fan-in.md): sends without a select leak when the consumer leaves.
- [For-Select Loop](../patterns/for-select-loop.md): every loop has an exit.
- [Future](../patterns/future.md): the result channel has a buffer of one, so the goroutine always ends.
- [Or-Done](../patterns/or-done.md): both the receive and the forwarding send select on done.
- [Pipeline](../patterns/pipeline.md): every send also selects on the context.
- [Request Racing](../patterns/request-racing.md): the answers channel has room for every replica, so losers can finish.
- [Result with Error](../patterns/result-with-error.md): every send selects on the context, so a consumer can stop early.
- [Snapshot via Owner](../patterns/snapshot-via-owner.md): the reply has a buffer of one and done marks the end.
- [State Machine](../patterns/state-machine.md): the owner waits for every effect it started before it reports that it is done.
- [Tee](../patterns/tee.md): every receive and send selects on the context.
- [Watermark](../patterns/watermark.md): forwarders select on the context on receive and send.
<!-- /usedby -->

## Violated by

<!-- violatedby -->
- [Channel Generator](../anti-patterns/channel-generator.md): the consumer must end a producer it did not start.
- [Closing from the Wrong Side](../anti-patterns/wrong-side-close.md): each sender selects on done so it can leave.
- [Fire-and-Forget Goroutine](../anti-patterns/fire-and-forget-goroutine.md): nobody owns the goroutine, so nobody ends it.
- [Goroutine as a Buffer](../anti-patterns/goroutine-as-buffer.md): a goroutine with no receiver leaks with its value.
- [Needless Asynchrony](../anti-patterns/needless-asynchrony.md): the callee starts a goroutine the caller must end.
<!-- /violatedby -->

## Related concepts

- [Cooperative Cancellation](cooperative-cancellation.md): how the stop signal reaches the work.
- [Channel Ownership](channel-ownership.md): a close is how a sender tells its receivers to stop waiting.
- [Backpressure](backpressure.md): a blocked send is backpressure while the consumer lives, and a leak after it left.

## Sources

<!-- sources -->
- [Cox-Buday (2017)](../sources.md#source-cox-buday), Ch. 4, "Preventing Goroutine Leaks"
- [Ajmani (2014b)](../sources.md#source-blog-pipelines), "Explicit cancellation"
<!-- /sources -->
