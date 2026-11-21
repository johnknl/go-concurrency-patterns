---
title: Channel Ownership
aliases: ['Who closes a channel']
intent: Give every channel one owner that alone sends the last value and closes it, so a close is never raced or repeated.
concerns: [data-race-safety, leak-prevention]
sources:
  - id: cox-buday
    at: 'Ch. 4, "Confinement"'
  - id: blog-pipelines
  - id: go-memory-model
---
# Channel Ownership

<!-- meta -->
Give every channel one owner that alone sends the last value and closes it, so a close is never raced or repeated.

<div class="cp-props" markdown>

| | |
|---|---|
| Also known as | Who closes a channel |
| Concerns | [Data race safety](../catalog/concern.md#concern-data-race-safety), [Leak prevention](../catalog/concern.md#concern-leak-prevention) |

</div>
<!-- /meta -->

## Why it matters

`close` is a statement about the future: no value will ever be sent again. Only a goroutine that controls every send
can make that statement truthfully. When the wrong goroutine closes, a send that is still in flight panics, two
closers panic each other, and a receiver that relies on the close to finish its loop either ends early or never ends.
Channel ownership assigns that statement to exactly one party and makes the rest of a design follow from it.

## Properties

### Only the owning sender closes { #sender-closes }

The owner of a channel is the goroutine (or the single piece of code) that sends on it. It alone closes the channel,
and it closes once, after its last send. Receivers never close: a receiver cannot know whether a sender is about to
send, and checking first does not help, since another goroutine can close between the check and the send.

### Several senders need a separate closer { #closer-waits }

When several goroutines send on one channel, none of them owns it. A separate closer waits until every sender has
returned, typically with a `sync.WaitGroup`, and then closes:

```go
go func() {
	wg.Wait()
	close(out)
}()
```

The closer is the owner. Each sender must still be able to return when the receiver has stopped (it selects on a
done channel or a context), or `wg.Wait` never returns; see
[Goroutine Leak Prevention](goroutine-leak-prevention.md#select-on-done).

### Send after close and double close panic { #close-panics }

A send on a closed channel panics, including a send that was already blocked when the close happened. A second
`close` panics. Closing a nil channel panics. None of these can be recovered into correct behavior; they are
ownership errors, and the fix is to give the channel a single owner.

### Receivers drain the buffer, then see the close { #drain-then-close }

Closing does not discard buffered values. Receivers get every value sent before the close, in order, and only then
the zero value with `ok == false`. A `for v := range ch` loop ends after the last buffered value. A receiver that
drains a channel until it is closed therefore knows that every sender has finished.

### A closed channel is always ready { #closed-always-ready }

A receive from a closed channel never blocks. In a `select` loop, a closed input case is ready on every iteration
and the loop spins on zero values. Either check `ok` and stop, or set the channel variable to `nil`: a nil channel is
never ready, which disables its case while the other cases continue.

```go
case v, ok := <-in:
	if !ok {
		in = nil // this case is never ready again
		continue
	}
```

### A stage never closes its input { #stage-owns-output }

In a chain of stages, each stage owns its output and closes it when it returns (`defer close(out)`). Its input
belongs to the stage before it. A stage that closes its input breaks the upstream sender. Ownership flows downstream
with the data, and so does the close: the source closes first, and each stage closes its output after its input is
drained.

### A close says only that no more values come { #close-no-reason }

A close carries one bit: the stream ended. It does not say whether it ended because the work was complete, failed, or
was canceled. A receiver that needs the reason must get it from somewhere else, such as a value stored before the
close ([Close with Cause](../patterns/close-with-cause.md)) or an error returned by the owner.

### A close wakes every receiver { #close-broadcasts }

A send wakes one receiver; a close wakes all of them, at once, and every later receive also succeeds immediately.
This makes closing a channel of `struct{}` the standard one-shot broadcast: a done channel, a ready signal, a
termination notice. It works only once; a signal that repeats needs a new channel or another mechanism (see
[Condition Variable](../stdlib/condition-variable.md)). A close also synchronizes memory with every receiver that
observes it ([Memory Visibility](memory-visibility.md#close-before-receive)).

## Used by

<!-- usedby -->
- [Batcher](../patterns/batcher.md): the producer closes in; the batcher drains it and flushes the rest.
- [Bounded Queue](../patterns/bounded-queue.md): after close, receivers still drain the queued values.
- [Close with Cause](../patterns/close-with-cause.md): a close says no more values but not why, and reaches every reader.
- [Fan-Out/Fan-In](../patterns/fan-out-fan-in.md): a closer waits for every worker, then closes the shared output.
- [For-Select Loop](../patterns/for-select-loop.md): a closed input is always ready and must be disabled.
- [Healing Goroutines](../patterns/healing-goroutines.md): a closed results channel is set to nil; the closed heartbeat marks the end of the ward.
- [Online Attach](../patterns/online-attach.md): the owner is the only sender and closer of member channels.
- [Pipeline](../patterns/pipeline.md): each stage closes its own output, never its input.
- [PubSub Hub](../patterns/pubsub-hub.md): the hub closes subscriber channels under the write lock, at most once.
- [Slow Consumer Eviction](../patterns/slow-consumer-eviction.md): only the producer closes, once, and the close carries no reason by itself.
- [Tee](../patterns/tee.md): tee closes both outputs it owns, never its input.
- [Worker Pool](../patterns/worker-pool.md): the submitter closes jobs; a closer closes results after every worker returns.
<!-- /usedby -->

## Violated by

<!-- violatedby -->
- [Closing from the Wrong Side](../anti-patterns/wrong-side-close.md): a receiver closes a channel it does not own.
<!-- /violatedby -->

## Related concepts

- [Goroutine Leak Prevention](goroutine-leak-prevention.md): a sender blocked forever on a channel nobody reads
  keeps the closer waiting.
- [Memory Visibility](memory-visibility.md): send and close are synchronization edges.
- [Confinement](confinement.md): ownership of a channel is a form of confinement of its send side.

## Sources

<!-- sources -->
- [Cox-Buday (2017)](../sources.md#source-cox-buday), Ch. 4, "Confinement"
- [Ajmani (2014b)](../sources.md#source-blog-pipelines)
- [The Go Authors (2022)](../sources.md#source-go-memory-model)
<!-- /sources -->
