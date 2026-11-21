---
title: Confinement
aliases: ['Thread confinement', 'Ownership transfer']
intent: Keep data reachable from only one goroutine at a time, so it needs no synchronization.
concerns: [data-race-safety]
sources:
  - id: cox-buday
    at: 'Ch. 4, "Confinement"'
  - id: blog-share-memory
  - id: go-wiki-mutex-or-channel
---
# Confinement

<!-- meta -->
Keep data reachable from only one goroutine at a time, so it needs no synchronization.

<div class="cp-props" markdown>

| | |
|---|---|
| Also known as | Thread confinement, Ownership transfer |
| Concerns | [Data race safety](../catalog/concern.md#concern-data-race-safety) |

</div>
<!-- /meta -->

## Why it matters

A data race needs two goroutines accessing the same memory, at least one of them writing, without synchronization.
Synchronizing every access costs effort and performance, and a missed one is a bug. Confinement removes the
condition instead: if only one goroutine can reach the data at any moment, there is nothing to synchronize. "Share
memory by communicating" is confinement with hand-off.

## Properties

### Data reachable from one goroutine needs no synchronization { #one-goroutine }

Data that only one goroutine can reach at a time can be read and written with plain operations. The guarantee is
about reachability, not intent: a pointer kept in a struct, a closure, or a global makes the data reachable from
wherever that pointer can be read.

### Lexical confinement { #lexical }

Data created inside a function and never stored where another goroutine can see it is confined by scope. The compiler
enforces it: local variables, values passed by copy, and disjoint parts of a slice given to different goroutines.
This is the strongest form and the cheapest.

### Confinement by hand-off { #hand-off }

Ownership moves with the value: the sender gives it away (on a channel, to a goroutine it starts) and must not touch
it afterwards; the receiver is now the only one that may. The send provides the memory edge
([Memory Visibility](memory-visibility.md#send-before-receive)); the discipline of not touching it again is the
programmer's.

### A shallow copy shares what it points to { #shallow-copy }

Copying a struct, or sending it by value, copies its fields, not the maps, slices, and pointers in them. After a
hand-off or a snapshot by copy, both sides still share the nested data. Either copy it deeply, hand it off as a
whole and stop using it, or make it immutable.

### Confinement by an owner goroutine { #owner-goroutine }

One goroutine holds the data for its whole life, and others reach it only by sending requests and receiving replies.
The data never leaves the owner, so it is confined although many goroutines use it.

### An owner goroutine is a throughput limit { #owner-costs }

Every access goes through one goroutine, one request at a time. The owner's speed bounds the total rate; a slow
request delays every caller queued behind it; each access costs at least two channel operations and two goroutine
switches, much more than an uncontended mutex. Requests sent to an owner that has stopped block forever unless they
also select on its done channel ([Goroutine Leak Prevention](goroutine-leak-prevention.md#select-on-done)).

### An owner is worth it when it has work of its own { #owner-has-work }

If the owner only serves reads and writes, a mutex does the same with less code and cost. An owner goroutine pays off
when it also does work nobody asked for: timers, batching, flushing, sending to consumers, reacting to several
inputs in one `select`. Then the state belongs naturally to the loop that does that work.

### Ad hoc versus enforced { #ad-hoc-enforced }

Ad hoc confinement holds by convention: the data is reachable, but the code agrees not to touch it. It breaks on the
first change that forgets the convention, and the race detector finds it only on the paths a test exercises.
Enforced confinement makes access impossible: unexported fields reachable only from the owner's loop, values passed
by copy, receive-only and send-only channel types. Prefer the enforced form and document any ad hoc one.

## Forms

| Form | Who may touch the data | Enforced by | Cost |
|---|---|---|---|
| [Lexical](#lexical) | The one goroutine whose scope holds it | The compiler | None |
| [Hand-off](#hand-off) | The current owner; ownership moves with the value | Discipline | One send per transfer |
| [Owner goroutine](#owner-goroutine) | Only the owner; others send requests | Unexported state and channels | A goroutine and a round trip per access |

## Used by

<!-- usedby -->
- [Active Object](../patterns/active-object.md): the state is a local variable of the owner goroutine.
- [Batcher](../patterns/batcher.md): the batch and timer are local to one goroutine; flush takes the batch over.
- [Copy-on-Write Snapshot](../patterns/copy-on-write-snapshot.md): the shallow clone shares nested maps and slices with the published version.
- [For-Select Loop](../patterns/for-select-loop.md): an owner goroutine is a for-select loop.
- [Monitor](../patterns/monitor.md): an owner goroutine is the alternative when operations block.
- [Online Attach](../patterns/online-attach.md): the member set belongs to the owner goroutine.
- [Replay Buffer](../patterns/replay-buffer.md): ring and subscriber set belong to one owner goroutine that replays and delivers.
- [Snapshot via Owner](../patterns/snapshot-via-owner.md): the state belongs to an owner goroutine and the copy is shallow.
- [State Machine](../patterns/state-machine.md): the state is a local variable of the owner goroutine.
<!-- /usedby -->

## Violated by

<!-- violatedby -->
- [Channel Where a Mutex Would Do](../anti-patterns/channel-instead-of-mutex.md): an owner goroutine with no work of its own.
- [Inconsistent Lock Order](../anti-patterns/inconsistent-lock-order.md): an owner goroutine needs no second lock.
- [Lock Held While Blocking](../anti-patterns/lock-held-while-blocking.md): state given to an owner goroutine needs no lock.
- [Pointer over a Channel](../anti-patterns/pointer-over-channel.md): ownership must move with the value, and copies are shallow.
<!-- /violatedby -->

## Related concepts

- [Memory Visibility](memory-visibility.md): the edge that makes a hand-off safe.
- [Channel Ownership](channel-ownership.md): confinement of the right to send and close.
- [Ordering](ordering.md): an owner goroutine handles requests in the order it receives them.

## Sources

<!-- sources -->
- [Cox-Buday (2017)](../sources.md#source-cox-buday), Ch. 4, "Confinement"
- [Gerrand (2010b)](../sources.md#source-blog-share-memory)
- [The Go Authors (n.d.i)](../sources.md#source-go-wiki-mutex-or-channel)
<!-- /sources -->
