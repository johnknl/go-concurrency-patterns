# Close with Cause

Tell the receivers of a closed channel why it was closed.

- **Purpose:** Lifecycle
- **Scale:** Idiom
- **Mechanisms:** Channels, Sync
- **Concerns:** Error handling, Leak prevention
- **Concepts:** [Channel Ownership](../concepts/channel-ownership.md), [Memory Visibility](../concepts/memory-visibility.md), [Cooperative Cancellation](../concepts/cooperative-cancellation.md)
- **Code:** [`patterns/closecause`](https://github.com/johnknl/go-concurrency-patterns/tree/main/patterns/closecause)

## Problem

A closed channel says that no more values will come, and nothing else
([close has no reason](../concepts/channel-ownership.md#close-no-reason)): the reader cannot tell a normal end from a
dropped subscriber, a cancellation, or a read error, and treating all of these as a normal end hides failures.

## Mechanism

The owner of the channel stores the reason, then closes the channel. When the reader's
`range` loop ends, it calls `Err()`; `nil` means a normal end. The store must happen before the close, so a reader
that has seen the close also sees the reason
([store before close](../concepts/memory-visibility.md#store-before-close)). One close reaches every reader
([close broadcasts](../concepts/channel-ownership.md#close-broadcasts)).

```go
type stream struct {
	err  error
	C    chan int
	once sync.Once
	mu   sync.Mutex
}

func newStream() *stream { return &stream{C: make(chan int, 1)} }

func (s *stream) finish(cause error) {
	s.once.Do(func() {
		s.mu.Lock()
		s.err = cause
		s.mu.Unlock()

		close(s.C)
	})
}

func (s *stream) Err() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	return s.err
}

func drain(name string, s *stream) string {
	n := 0
	for range s.C {
		n++
	}

	return fmt.Sprint(name, " ", n, " ", s.Err())
}

func main() {
	ok := newStream()
	ok.C <- 1
	ok.finish(nil)
	ok.finish(errors.New("ignored"))

	failed := newStream()
	failed.C <- 1
	failed.finish(errors.New("connection reset"))

	fmt.Println(drain("ok", ok))
	fmt.Println(drain("failed", failed))

	// Output:
	// ok 1 <nil>
	// failed 1 connection reset
}
```

The mutex only makes `Err` safe to call before the close. `sync.Once` matters when `finish` has several callers,
such as a producer that ends normally and an owner that drops the stream: only the first cause is kept, and the
channel is closed once, where a second close would panic. The standard library uses the same shape:
`bufio.Scanner.Err` and `sql.Rows.Err` are called after the loop, and `context.Cause` returns why a context ended.

## Applicability

Use it when a stream can end for more than one reason and the reader acts differently on each. For a single result,
return `(value, error)` instead.

## Consequences

- `Err` returns `nil` before the close, which looks like success. Call it after the channel is closed.
- `sync.Once` prevents a double close, not a send after the close. Sends and `finish` must come from the same owner,
  or be serialized with it.

## Variants

### Atomic pointer

Keep the cause in an `atomic.Pointer` instead of behind a mutex.

### Context cause

When the stream is tied to a context, cancel it with `context.WithCancelCause` and let the reader call
`context.Cause` ([return the cause](../concepts/cooperative-cancellation.md#return-cause)).

## Related patterns

- [Result with Error](result-with-error.md) covers other ways to carry errors.
- [PubSub Hub](pubsub-hub.md) and [Online Attach](online-attach.md) use it to tell a subscriber why its stream ended.
- [Context Propagation](../stdlib/context-propagation.md) carries a cause down a tree of contexts.

## Anti-patterns

- [Closing from the Wrong Side](../anti-patterns/wrong-side-close.md): A channel is closed by a goroutine that does not own it, so a later send or a second close panics.

## Sources

- [The Go Authors (n.d.j)](../sources.md#source-go-bufio-scanner)
