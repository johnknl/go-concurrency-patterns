# Snapshot via Owner

Read a consistent copy of state that one goroutine owns, without sharing the state.

- **Purpose:** Coordination
- **Scale:** Idiom
- **Mechanisms:** Channels, Select
- **Concerns:** Data race safety
- **Concepts:** [Confinement](../concepts/confinement.md), [Goroutine Leak Prevention](../concepts/goroutine-leak-prevention.md)
- **Code:** [`patterns/snapshot`](https://github.com/johnknl/go-concurrency-patterns/tree/main/patterns/snapshot)

## Problem

State is confined to one goroutine ([confinement](../concepts/confinement.md#owner-goroutine)), and other
goroutines need to read it. Reading it directly is a data race, and
a lock would undo the confinement.

## Mechanism

The owner receives reply channels and sends a copy of the state on each. A reply channel is new per request and has
capacity 1, so the owner's send completes even when the requester has stopped waiting
([reply buffer of one](../concepts/goroutine-leak-prevention.md#reply-buffer-one)). In this example the state is a map
of event counts, and closing `done` when the loop returns lets the caller wait for the owner to end
([observable end](../concepts/goroutine-leak-prevention.md#observable-end)).

```go
type stats struct {
	events    chan string
	snapshots chan chan map[string]int
	quit      chan struct{}
	done      chan struct{}
}

func newStats() *stats {
	s := &stats{
		events:    make(chan string),
		snapshots: make(chan chan map[string]int),
		quit:      make(chan struct{}),
		done:      make(chan struct{}),
	}
	go s.run()

	return s
}

func (s *stats) run() {
	defer close(s.done)

	counts := map[string]int{}

	for {
		select {
		case name := <-s.events:
			counts[name]++
		case reply := <-s.snapshots:
			// The reply has capacity 1, so this send never blocks, even if the requester has left.
			reply <- maps.Clone(counts)
		case <-s.quit:
			return
		}
	}
}

func (s *stats) record(name string) { s.events <- name }

func (s *stats) snapshot() map[string]int {
	reply := make(chan map[string]int, 1)
	s.snapshots <- reply

	return <-reply
}

func main() {
	s := newStats()

	var wg sync.WaitGroup
	for range 50 {
		wg.Go(func() { s.record("hit") })
		wg.Go(func() { s.record("miss") })
	}

	wg.Wait()

	snap := s.snapshot()
	fmt.Println(snap["hit"], snap["miss"])

	// The snapshot is a copy: changing it does not change the owner's state.
	snap["hit"] = 0
	s.record("hit")
	fmt.Println(s.snapshot()["hit"])

	close(s.quit)
	<-s.done

	// Output:
	// 50 50
	// 51
}
```

The same loop receives requests and events one at a time, so a snapshot is consistent as of one moment between two
events. In this example `events` is unbuffered, so once `record` returns, a later snapshot includes the event. The copy
is outdated at once.

## Applicability

Use it when state is owned by a goroutine, for example in an [Active Object](active-object.md), and others need to
observe it. For state without an owner, a mutex is simpler: adding an owner only to serve reads is
[Channel Instead of Mutex](../anti-patterns/channel-instead-of-mutex.md).

## Consequences

- The copy costs in proportion to the state. `maps.Clone` is shallow: pointer, slice or map
  values are still shared with the owner ([shallow copy](../concepts/confinement.md#shallow-copy)).
- A busy owner delays readers, and after the owner has stopped, a send on `snapshots` blocks forever; requesters
  that can outlive the owner also select on `done` or a context ([owner costs](../concepts/confinement.md#owner-costs)).

## Related patterns

- [Active Object](active-object.md) uses it for queries.
- [Future](future.md) also delivers one result on a channel with capacity 1.
- [Copy-on-Write Snapshot](copy-on-write-snapshot.md) publishes an immutable copy that readers load without asking
  an owner.

## Anti-patterns

- [Lock Held While Blocking](../anti-patterns/lock-held-while-blocking.md): Hold a mutex across a channel operation, I/O or a callback, so everyone else who needs the lock waits, and a deadlock follows if the other side needs it too.
- [Pointer over a Channel](../anti-patterns/pointer-over-channel.md): A pointer, slice, or map is sent over a channel and then used by both sides, so the data is shared although it was handed over.

## Sources

- [Gerrand (2010b)](../sources.md#source-blog-share-memory)
