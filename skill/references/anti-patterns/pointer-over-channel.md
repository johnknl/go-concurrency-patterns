# Pointer over a Channel

A pointer, slice, or map is sent over a channel and then used by both sides, so the data is shared although it was handed over.

- **Violates:** [Clear ownership](../principles.md#principle-ownership)
- **Purpose:** Coordination
- **Mechanisms:** Channels
- **Concerns:** Data race safety
- **Breaks:** [Memory Visibility](../concepts/memory-visibility.md), [Confinement](../concepts/confinement.md)
- **Code:** [`antipatterns/pointerchannel`](https://github.com/johnknl/go-concurrency-patterns/tree/main/antipatterns/pointerchannel)

## Symptoms

- A pointer, slice, or map is sent over a channel and the sender keeps using it.
- The receiver sees values change after it received them, or the race detector reports a read and a write of the
  same memory in two goroutines.
- The bugs are intermittent and depend on when the sender's next write happens.

## Example

**Faulty**

`badShare` sends a `*note`, then edits it after a handshake. Every access is ordered by a channel operation, so
there is no data race, which isolates the aliasing: the receiver's second read shows the sender's edit.

```go
type note struct{ text string }

func badShare() (first, second string) {
	ch := make(chan *note)
	ack := make(chan struct{})
	again := make(chan struct{})
	res := make(chan [2]string)
	go func() {
		p := <-ch
		a := p.text
		ack <- struct{}{}
		<-again
		res <- [2]string{a, p.text}
	}()
	n := &note{"draft"}
	ch <- n
	<-ack
	n.text = "edited"
	again <- struct{}{}
	r := <-res
	return r[0], r[1]
}

func main() {
	first, second := badShare()
	fmt.Println(first, second)
	// Output: draft edited
}
```

**Fixed**

`copyShare` sends a value, so each side has its own copy, and with the same handshake the receiver's value does
not change.

```go
func copyShare() (first, second string) {
	ch := make(chan note)
	ack := make(chan struct{})
	again := make(chan struct{})
	res := make(chan [2]string)
	go func() {
		v := <-ch
		a := v.text
		ack <- struct{}{}
		<-again
		res <- [2]string{a, v.text}
	}()
	n := note{"draft"}
	ch <- n
	<-ack
	n.text = "edited"
	again <- struct{}{}
	r := <-res
	return r[0], r[1]
}

func main() {
	first, second := copyShare()
	fmt.Println(first, second)
	// Output: draft draft
}
```

## Why it goes wrong

A send [orders](../concepts/memory-visibility.md#send-before-receive) the sender's earlier writes before the
receive and copies the sent value; for a pointer, slice, or map that value is a reference. After the send both sides
reach the same memory, and [nothing orders](../concepts/memory-visibility.md#no-edge) their later accesses. Sending a
struct by value is [shallow](../concepts/confinement.md#shallow-copy) too.

The fix is to decide who owns the data. Either send a copy, cloning nested slices and maps (`slices.Clone`,
`maps.Clone`), or [hand ownership over](../concepts/confinement.md#hand-off): after the send, the sender does not
touch the data until the receiver sends it back.

## Detection

- `go test -race` reports the unordered accesses, if the test runs both. When a handshake orders them, as in the
  example, nothing is reported and the aliasing is a logic error.
- No `go vet` analyzer checks this. In review, look at what a channel element type contains and what the sender does
  with a value after sending it.

## Fixed by

- [Snapshot via Owner](../patterns/snapshot-via-owner.md): Read a consistent copy of state that one goroutine owns, without sharing the state.
- [Copy-on-Write Snapshot](../patterns/copy-on-write-snapshot.md): Let many readers see a consistent version of read-mostly data without waiting for writers or for each other.

For shared read-mostly state, publish immutable copies with a [Copy-on-Write
Snapshot](../patterns/copy-on-write-snapshot.md); for state one goroutine owns, ask the owner for a copy with [Snapshot
via Owner](../patterns/snapshot-via-owner.md).

## Sources

- [Gerrand (2010b)](../sources.md#source-blog-share-memory)
