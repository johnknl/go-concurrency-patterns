# Slow Consumer Eviction

Keep a consumer that keeps missing deliveries from holding back the others.

- **Also known as:** Subscriber dropping, Slow subscriber disconnect
- **Purpose:** Resilience
- **Scale:** Idiom
- **Mechanisms:** Channels, Select, Time
- **Concerns:** Overload, Liveness
- **Concepts:** [Overload Policy](../concepts/overload-policy.md), [Backpressure](../concepts/backpressure.md), [Channel Ownership](../concepts/channel-ownership.md), [Memory Visibility](../concepts/memory-visibility.md)
- **Code:** [`patterns/eviction`](https://github.com/johnknl/go-concurrency-patterns/tree/main/patterns/eviction)

## Problem

A producer that delivers to several consumers in turn waits for each one. [Timed Send](timed-send.md) bounds the
wait, but a consumer that never recovers costs the timeout on every value, and every consumer after it waits that
long too. Evicting is one of the [overload options](../concepts/overload-policy.md#options), applied
[per consumer](../concepts/overload-policy.md#per-consumer), for a producer that
[cannot wait](../concepts/backpressure.md#cannot-wait).

## Mechanism

Each consumer record holds its channel and a strike counter. A timed send that misses adds a strike, and a delivery
resets it, so only consecutive misses count. When the strikes exceed the tolerance (0 evicts on the first miss),
the producer, as the only sender, closes the channel and removes the record; a later send or second close would
panic ([close panics](../concepts/channel-ownership.md#close-panics)). A close alone gives no reason
([close has no reason](../concepts/channel-ownership.md#close-no-reason)), so the producer stores the reason in the
record before the `close`, as in [Close with Cause](close-with-cause.md); the close publishes it without a lock
([store before close](../concepts/memory-visibility.md#store-before-close)).

In this example, a consumer that never reads misses values 0 and 1 and is evicted on the second miss (tolerance
1). The healthy consumer always has room, so the output does not depend on timing.

```go
type subscriber struct {
	err     error
	ch      chan int
	name    string
	strikes int
}

func broadcast(subs []*subscriber, v int, t *time.Timer, tolerance int) []*subscriber {
	for _, s := range subs {
		if timedSend(s.ch, v, t, time.Millisecond) {
			s.strikes = 0
			continue
		}

		s.strikes++
		if s.strikes > tolerance {
			s.err = errTooSlow
			close(s.ch)
		}
	}

	return slices.DeleteFunc(subs, func(s *subscriber) bool { return s.err != nil })
}

func run(tolerance int) (lines []string) {
	t := time.NewTimer(time.Hour)
	t.Stop()

	healthy := &subscriber{name: "healthy", ch: make(chan int, 3)}
	stuck := &subscriber{name: "stuck", ch: make(chan int)} // never read until evicted
	subs := []*subscriber{healthy, stuck}

	for v := range 3 {
		subs = broadcast(subs, v, t, tolerance)
	}

	_, open := <-stuck.ch
	lines = append(lines, fmt.Sprintf("stuck open: %t cause: %v", open, stuck.err))

	close(healthy.ch)

	for v := range healthy.ch {
		lines = append(lines, fmt.Sprintf("healthy got %d", v))
	}

	lines = append(lines, fmt.Sprintf("remaining: %d", len(subs)))

	return lines
}

func main() {
	for _, line := range run(1) {
		fmt.Println(line)
	}

	// Output:
	// stuck open: false cause: too many missed deliveries
	// healthy got 0
	// healthy got 1
	// healthy got 2
	// remaining: 1
}
```

## Applicability

Use it when one producer serves many independent consumers, none may delay the rest, and consumers can detect
closure and re-subscribe or be treated as lost. When every consumer must receive every value, use
[backpressure](../concepts/backpressure.md#feedback) or a [Replay Buffer](replay-buffer.md).

## Consequences

- Values sent while a consumer was stuck, and after its eviction, are lost to it unless something replays them.
- Without the cause, an eviction looks like a normal end of stream.
- Timeout and tolerance trade off: small values evict briefly busy consumers, large ones slow the others longer.
- Only the producer may send on, close, or update a record; hand new subscribers to it by channel.

## Related patterns

- [Timed Send](timed-send.md) provides the miss.
- [Close with Cause](close-with-cause.md) records why the channel was closed.
- [Online Attach](online-attach.md) lets the owner wait and count without a lock.

## Sources

Original work.
