---
title: Memory Visibility
aliases: ['Happens-before', 'Safe publication']
intent: A write made in one goroutine is guaranteed visible in another only if a synchronization edge of the Go memory model orders the two.
concerns: [data-race-safety]
sources:
  - id: go-memory-model
  - id: go-sync
  - id: go-sync-atomic
  - id: herlihy-shavit-2012
---
# Memory Visibility

<!-- meta -->
A write made in one goroutine is guaranteed visible in another only if a synchronization edge of the Go memory model orders the two.

<div class="cp-props" markdown>

| | |
|---|---|
| Also known as | Happens-before, Safe publication |
| Concerns | [Data race safety](../catalog/concern.md#concern-data-race-safety) |

</div>
<!-- /meta -->

## Why it matters

Compilers and processors reorder memory operations, and caches delay them. Without synchronization, one goroutine
may see another's writes late, partly, in a different order, or never; a program whose goroutines touch the same
memory without ordering has a data race, and its behavior is not defined by the source. The Go memory model lists
the operations that order memory between goroutines. Every correct handoff of data relies on one of them.

## Properties

### Writes before an edge are visible after it { #edge-publishes }

If operation A is synchronized before operation B, every write that happened before A in A's goroutine is visible to
every read after B in B's goroutine. Synchronization is transitive. Data needs no lock of its own when it is written
before an edge and read only after it, and not written again; this is safe publication.

### A send happens before its receive completes { #send-before-receive }

A send on a channel is synchronized before the completion of the corresponding receive. Whatever the sender wrote
before sending, including into the value it sends, is visible to the receiver after the receive. For an unbuffered
channel the receive is also synchronized before the send completes.

### A close happens before a receive that sees it { #close-before-receive }

Closing a channel is synchronized before a receive that returns because the channel is closed. Every receiver that
observes the close, through `ok == false`, the end of a `range`, or a ready `case <-done`, sees all writes made
before the close.

### The kth receive happens before the (k+C)th send { #capacity-edge }

On a channel of capacity C, the *k*th receive is synchronized before the *k+C*th send completes. This edge is what
makes a buffered channel usable as a counting semaphore: work done before releasing a slot (a receive) is visible to
the goroutine whose acquire (a send) that slot unblocks.

### Unlock happens before the next Lock { #unlock-before-lock }

For a `sync.Mutex` or `sync.RWMutex`, the *n*th `Unlock` is synchronized before the *n+1*th `Lock` returns, and an
`RUnlock` before the next `Lock`. Writes made while holding the lock are visible to the next holder. A read that
does not take the lock gets no such guarantee, even if it "only reads".

### An atomic store happens before a load that observes it { #atomic-store-load }

If an atomic load observes the value written by an atomic store, the store is synchronized before the load. Go's
atomics are sequentially consistent: all atomic operations appear in one order that every goroutine agrees on.
Publishing a pointer with `atomic.Pointer[T].Store` makes the pointed-to data, written before the store, visible to a
`Load` that returns that pointer, provided nobody writes the data afterwards.

### Once's function happens before any Do returns { #once-before-do }

The single call of `f` in `once.Do(f)` is synchronized before the return of every `once.Do(f)`. Everything `f`
initialized is visible to all callers after `Do` returns, including those that waited and those that came later.
`sync.OnceFunc`, `OnceValue`, and `OnceValues` give the same guarantee.

### A plain pointer or a sleep gives no edge { #no-edge }

Reading a variable without synchronization, even a pointer or a bool that another goroutine set, creates no edge:
the reader may see the new pointer and old contents, or never see the change. Waiting does not create one either:
`time.Sleep` orders nothing, however long it is. Checking a plain flag before taking a lock (double-checked locking)
fails for the same reason; use `sync.Once` or an atomic.

### Store a value before the close that announces it { #store-before-close }

To pass a value, such as an error cause or a final result, with a one-shot signal, write it before closing the
channel and read it only after observing the close. The close edge then publishes it, and no lock is needed. Writing
it after the close, or reading it before the close is observed, is a data race.

```go
c.err = err // written once, before the close
close(c.done)

<-c.done
return c.err // read only after the close is observed
```

## Used by

<!-- usedby -->
- [Active Object](../patterns/active-object.md): the command send and the reply send order the effect before the caller sees it.
- [Close with Cause](../patterns/close-with-cause.md): the cause is stored before the close that announces it.
- [Compare-and-Swap Loop](../patterns/compare-and-swap-loop.md): a successful swap publishes the new value to the loads that observe it.
- [Copy-on-Write Snapshot](../patterns/copy-on-write-snapshot.md): the pointer store publishes the fully built copy to readers.
- [Future](../patterns/future.md): the send of the result publishes it to the receiver.
- [Lazy Initialization](../stdlib/lazy-initialization.md): the writes of the once function are visible to every caller of Do.
- [Monitor](../patterns/monitor.md): unlock happens before the next lock.
- [Online Attach](../patterns/online-attach.md): the cause is stored before the close that announces it.
- [PubSub Hub](../patterns/pubsub-hub.md): the reason is stored before the close; lock edges order membership changes.
- [Semaphore](../patterns/semaphore.md): the capacity edge caps the holders at C.
- [Slow Consumer Eviction](../patterns/slow-consumer-eviction.md): the cause is stored before the close that announces it.
<!-- /usedby -->

## Violated by

<!-- violatedby -->
- [Double-Checked Locking](../anti-patterns/double-checked-locking.md): a nil check on a plain pointer gives no happens-before edge.
- [Pointer over a Channel](../anti-patterns/pointer-over-channel.md): the send orders only the accesses before it.
- [Sleeping to Synchronize](../anti-patterns/sleep-synchronization.md): a sleep gives no happens-before edge.
<!-- /violatedby -->

## Related concepts

- [Confinement](confinement.md): data confined to one goroutine needs no edges, except at the hand-off.
- [Channel Ownership](channel-ownership.md): who may send and close the channel whose edges publish the data.

## Sources

<!-- sources -->
- [The Go Authors (2022)](../sources.md#source-go-memory-model)
- [The Go Authors (n.d.e)](../sources.md#source-go-sync)
- [The Go Authors (n.d.f)](../sources.md#source-go-sync-atomic)
- [Herlihy and Shavit (2012)](../sources.md#source-herlihy-shavit-2012)
<!-- /sources -->
