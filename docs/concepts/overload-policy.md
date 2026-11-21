---
title: Overload Policy
aliases: ['Slow-consumer policy', 'Overflow policy']
intent: Decide deliberately what happens when a limit is reached, trading loss, latency, and memory.
concerns: [overload, backpressure, latency]
sources:
  - id: nygard-2018
    at: 'Ch. 5, "Shed Load" and "Create Back Pressure"'
  - id: sre-overload
  - id: reactive-streams
---
# Overload Policy

<!-- meta -->
Decide deliberately what happens when a limit is reached, trading loss, latency, and memory.

<div class="cp-props" markdown>

| | |
|---|---|
| Also known as | Slow-consumer policy, Overflow policy |
| Concerns | [Overload](../catalog/concern.md#concern-overload), [Backpressure](../catalog/concern.md#concern-backpressure), [Latency](../catalog/concern.md#concern-latency) |

</div>
<!-- /meta -->

## Why it matters

Every bounded resource, a channel buffer, a semaphore, a pool, a subscriber's queue, eventually reaches its limit.
At that moment some code runs, and it implements a policy whether or not anyone chose one. A plain send waits; a
`default` case drops; a growing slice spends memory. Each policy sacrifices something different, and the wrong one
lets one slow consumer stall everyone, or loses data that had to arrive.

## Properties

### Every limit needs a policy { #every-limit }

Wherever a bound exists, the code that hits it must do one of a small number of things: wait, wait for a while, drop,
reject, evict, or spill. The choice belongs to the design, stated next to the bound, not to whichever channel
operation happened to be written.

### The options trade loss, latency, and memory { #options }

| Policy | When the limit is reached | Loses | Costs | Suits |
|---|---|---|---|---|
| Wait ([Backpressure](backpressure.md)) | The producer blocks until there is room | Nothing | Latency; the slowest consumer sets the pace | Every value matters; the source can wait |
| Wait with a bound ([Timed Send](../patterns/timed-send.md)) | Wait up to a timeout or deadline, then fall back to another policy | Values after the timeout | Up to the timeout per operation | Short stalls are normal, long ones are failures |
| Drop newest ([Load Shedding](../patterns/load-shedding.md)) | The new value is discarded, and counted | Recent values | Nothing for the producer | Samples, metrics, best-effort events |
| Drop oldest | The oldest queued value is discarded to make room | Stale values | Nothing for the producer | Latest-value-wins streams (state, positions) |
| Reject | The caller gets an error at once | The request, visibly | The caller must handle it | Requests whose caller can retry or go elsewhere |
| Evict ([Slow-Consumer Eviction](../patterns/slow-consumer-eviction.md)) | The slow consumer is disconnected | That consumer's stream | It must reconnect or resync | Many independent consumers; one must not hold up the rest |
| Spill ([Spill Queue](../patterns/spill-queue.md)) | Overflow is written to disk, in order | Nothing, until the disk limit | Disk I/O and complexity | Bursts larger than memory should hold, when the work tolerates delay |

Dropping silently is never a policy: count what is dropped or rejected, so overload is visible.

### A bounded buffer only delays the choice { #buffer-delays }

A buffer turns an instant limit into a later one. Under sustained overload it fills, and the policy decides again,
now with added latency for everything already queued. A buffer is sized for bursts; it is not an alternative to a
policy, and a larger one makes the eventual choice later and larger.

### Per-consumer or shared limits { #per-consumer }

When one producer feeds several consumers, a shared limit (one buffer, one blocking send per consumer in turn) lets
the slowest consumer set the rate for all. Per-consumer limits isolate them: each consumer has its own buffer, and
the policy applies to that consumer only, so a slow one loses its own values or is evicted while the others continue.
A shared limit suits work that must be balanced; per-consumer limits suit broadcast.

### Waiting must still be cancelable { #wait-cancelable }

Any policy that waits, plain or bounded, also selects on cancellation, so a producer whose caller left does not wait
for a consumer that will never come back ([Cooperative Cancellation](cooperative-cancellation.md#blocking-selects-done)).

## Used by

<!-- usedby -->
- [Bounded Queue](../patterns/bounded-queue.md): the capacity only delays the choice between waiting and rejecting.
- [Condition Variable](../stdlib/condition-variable.md): the budget waits for capacity, rejects what can never fit, and the wait stays cancelable.
- [Heartbeat](../patterns/heartbeat.md): a pulse nobody reads is dropped, so the heartbeat never stalls the work.
- [Load Shedding](../patterns/load-shedding.md): drop newest and drop oldest, with the drops counted.
- [PubSub Hub](../patterns/pubsub-hub.md): a full subscriber drops the message, counted, per subscriber.
- [Replay Buffer](../patterns/replay-buffer.md): each subscriber needs a buffer or a policy of its own.
- [Resource Pool](../patterns/resource-pool.md): an empty pool makes get wait, and the wait selects on a context.
- [Semaphore](../patterns/semaphore.md): a full semaphore waits, waits with a bound, or refuses.
- [Slow Consumer Eviction](../patterns/slow-consumer-eviction.md): evict a consumer after repeated missed deliveries, per consumer.
- [Spill Queue](../patterns/spill-queue.md): at the disk limit the policy applies again.
- [Timed Send](../patterns/timed-send.md): a bounded wait, still cancelable, then the caller decides.
<!-- /usedby -->

## Violated by

<!-- violatedby -->
- [Buffer as a Band-Aid](../anti-patterns/oversized-buffer.md): a buffer only delays the overload decision.
- [Goroutine as a Buffer](../anti-patterns/goroutine-as-buffer.md): the decision between waiting and dropping is never made.
<!-- /violatedby -->

## Related concepts

- [Backpressure](backpressure.md): the policy of waiting, and why it needs a bound.
- [Timers and Tickers](timers.md): bounded waiting is built on timers.

## Sources

<!-- sources -->
- [Nygard (2018)](../sources.md#source-nygard-2018), Ch. 5, "Shed Load" and "Create Back Pressure"
- [Cuervo (2016)](../sources.md#source-sre-overload)
- [Reactive Streams (n.d.)](../sources.md#source-reactive-streams)
<!-- /sources -->
