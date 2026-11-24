# PubSub Hub

Deliver every published message to each current subscriber, while subscribers come and go.

- **Also known as:** Publish-Subscribe, Publisher-Subscriber, Event bus
- **Purpose:** Data flow
- **Scale:** Composite
- **Mechanisms:** Channels, Select, Sync, Atomic
- **Concerns:** Composition, Overload
- **Concepts:** [Channel Ownership](../concepts/channel-ownership.md), [Memory Visibility](../concepts/memory-visibility.md), [Overload Policy](../concepts/overload-policy.md), [Ordering](../concepts/ordering.md)
- **Code:** [`patterns/pubsub`](https://github.com/johnknl/go-concurrency-patterns/tree/main/patterns/pubsub)

## Problem

A sender that delivers to its receivers directly must know who they are, and every receiver that comes or goes
changes the sender. Senders should publish without knowing their receivers, receivers should come and go while
senders run, and one receiver that stops reading should not stop the others.

## Mechanism

The hub is a set of subscribers guarded by a `sync.RWMutex`; each subscriber has its own buffered channel. The hub
needs no goroutine of its own.

```go
type sub struct {
	ch  chan int
	err error
}

type hub struct {
	subs    map[*sub]struct{}
	dropped atomic.Int64
	mu      sync.RWMutex
	closed  bool
}

func (h *hub) subscribe(buffer int) *sub {
	s := &sub{ch: make(chan int, buffer)}

	h.mu.Lock()
	defer h.mu.Unlock()

	if h.closed {
		s.err = errClosed
		close(s.ch)

		return s
	}

	h.subs[s] = struct{}{}

	return s
}

func (h *hub) unsubscribe(s *sub) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if _, ok := h.subs[s]; !ok {
		return
	}

	delete(h.subs, s)
	s.err = errUnsubscribed
	close(s.ch)
}

func (h *hub) publish(msg int) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	for s := range h.subs {
		select {
		case s.ch <- msg:
		default:
			h.dropped.Add(1)
		}
	}
}

func (h *hub) close() {
	h.mu.Lock()
	defer h.mu.Unlock()

	h.closed = true

	for s := range h.subs {
		s.err = errClosed
		close(s.ch)
	}

	clear(h.subs)
}

func main() {
	h := newHub()

	a := h.subscribe(4)
	b := h.subscribe(4)
	slow := h.subscribe(1)

	h.publish(1)
	h.publish(2)

	h.unsubscribe(b)

	h.publish(3)

	late := h.subscribe(4)

	h.publish(4)

	h.close()

	fmt.Println(drain("a", a))
	fmt.Println(drain("b", b))
	fmt.Println(drain("late", late))
	fmt.Println(drain("slow", slow))
	fmt.Println("dropped", h.dropped.Load())

	// Output:
	// a [1 2 3 4] hub closed
	// b [1 2] unsubscribed
	// late [4] hub closed
	// slow [1] hub closed
	// dropped 3
}
```

- **Publish** takes the read lock and copies the message to every subscriber, from the publisher's goroutine;
  several publishers deliver at once.
- **Delivery never blocks.** A delivery that waited for a slow subscriber under the read lock would stall every
  other publisher and every subscribe ([Lock Held While Blocking](../anti-patterns/lock-held-while-blocking.md)).
  A full subscriber loses the message, which is counted: a drop-newest choice among the
  [overload options](../concepts/overload-policy.md#options), applied [per
  subscriber](../concepts/overload-policy.md#per-consumer).
  The buffer size is the burst a subscriber may fall
  behind by.
- **Subscribe and unsubscribe** take the write lock, so they apply between deliveries
  ([unlock before lock](../concepts/memory-visibility.md#unlock-before-lock)). A channel is closed only under the
  write lock and sent to only under the read lock, so a [send on a closed
  channel](../concepts/channel-ownership.md#close-panics)
  cannot happen.
- **Closing.** The reason is stored before the close, which publishes it to the subscriber
  ([close before receive](../concepts/memory-visibility.md#close-before-receive)). The membership check in unsubscribe
  prevents a second close
  (a panic) after `close` or a concurrent unsubscribe; the `closed` flag gives a subscriber that arrives after
  `close` an already closed channel instead of one nobody will close.

A late subscriber receives only what is published after it subscribed.

Each subscriber buffer has the concerns of a [Bounded Queue](bounded-queue.md): what to do when it is full (here: drop,
not block) and what happens to queued messages on unsubscribe or close (here: they stay readable until drained).

**Concurrent publishers.** With a buffer large enough, a subscriber receives every message from publishers running in
parallel.

```go
func main() {
	h := newHub()
	s := h.subscribe(100)

	var wg sync.WaitGroup

	for p := range 4 {
		wg.Go(func() {
			for i := range 25 {
				h.publish(p*100 + i)
			}
		})
	}

	wg.Wait()
	h.close()

	n := 0
	for range s.ch {
		n++
	}

	fmt.Println("received", n, "dropped", h.dropped.Load())

	// Output:
	// received 100 dropped 0
}
```

## Composition

- [Tee](tee.md): every subscriber gets its own copy of each message.
- [Load Shedding](load-shedding.md): a full subscriber loses the message instead of blocking the publisher.
- [Close with Cause](close-with-cause.md): unsubscribing and closing the hub record why a stream ended.

## Applicability

Use a hub when several independent receivers each need every message, receivers come and go while producers run,
and the publisher must not wait for the slowest receiver. When each message should go to exactly one receiver, use
[Fan-Out/Fan-In](fan-out-fan-in.md); for a fixed number of copies, [Tee](tee.md) is simpler.

## Consequences

- A publish costs time in proportion to the number of subscribers, paid by the publisher.
- A waiting writer blocks new readers in `sync.RWMutex`, so a subscribe briefly holds up all publishers.
- A full subscriber loses messages. Waiting, with a timeout or until eviction, does not fit under the lock; those
  [options](../concepts/overload-policy.md#options) need the owner goroutine variant.
- Order is preserved per publisher and per subscriber ([sequential FIFO](../concepts/ordering.md#sequential-fifo));
  between [concurrent publishers](../concepts/ordering.md#concurrent-senders) there is none.

## Variants

### Owner goroutine

An owner holds the set, and publish, subscribe and unsubscribe are messages to it. No lock; the owner may wait
for, evict, or replay history to a subscriber, at the cost of a goroutine with a lifecycle and a round trip per
operation. This is [Online Attach](online-attach.md).

```go
func (h *hub) run(ctx context.Context) {
	defer close(h.done)

	subs := map[*sub]bool{}

	var history []int

	for {
		select {
		case msg := <-h.pub:
			history = append(history, msg)
			history = history[max(0, len(history)-2):]

			for s := range subs {
				h.deliver(s, msg)
			}
		case s := <-h.attach:
			for _, msg := range history {
				h.deliver(s, msg)
			}

			subs[s] = true
		case s := <-h.detach:
			if subs[s] {
				delete(subs, s)

				s.err = errDetached
				close(s.ch)
			}
		case <-ctx.Done():
			for s := range subs {
				s.err = context.Cause(ctx)
				close(s.ch)
			}

			return
		}
	}
}
```

### Topics

One subscriber set per key.

### Copy-on-write set

The set in an `atomic.Pointer`, replaced on every change, so publishing takes no lock. Unsubscribe can then not
close the channel safely, since a publisher may still use an old copy; mark the subscriber gone and let it drain.

### Late subscribers

A [Replay Buffer](replay-buffer.md) must be updated and replayed under the write lock, so the new subscriber sees
no gap and no duplicate.

## Related patterns

- [Online Attach](online-attach.md) is the lock-free form, with one owner goroutine.
- [Tee](tee.md) copies to a fixed set of outputs.
- [Fan-Out/Fan-In](fan-out-fan-in.md) shares messages between consumers instead of copying them.

## Sources

- [Buschmann et al. (1996)](../sources.md#source-buschmann-1996), Publisher-Subscriber
