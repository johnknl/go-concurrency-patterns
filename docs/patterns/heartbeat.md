---
title: Heartbeat
aliases: ['Liveness signal', 'Keep-alive']
intent: Tell a slow worker from a stuck one, and know when work has started, without sleeping.
purpose: resilience
scale: idiom
package: patterns/heartbeat
mechanisms: [channels, select, time]
concerns: [liveness]
sources:
  - id: cox-buday
    at: 'Ch. 5, "Heartbeats"'
concepts:
  - overload-policy: 'a pulse nobody reads is dropped, so the heartbeat never stalls the work.'
---
# Heartbeat

<!-- meta -->
Tell a slow worker from a stuck one, and know when work has started, without sleeping.

<div class="cp-props" markdown>

| | |
|---|---|
| Also known as | Liveness signal, Keep-alive |
| Purpose | [Resilience](../catalog/purpose.md#purpose-resilience) |
| Scale | [Idiom](../catalog/scale.md#scale-idiom) |
| Mechanisms | [Channels](../catalog/mechanism.md#mechanism-channels), [Select](../catalog/mechanism.md#mechanism-select), [Time](../catalog/mechanism.md#mechanism-time) |
| Concerns | [Liveness](../catalog/concern.md#concern-liveness) |
| Concepts | [Overload Policy](../concepts/overload-policy.md) |
| Code | [`patterns/heartbeat`](https://github.com/johnknl/go-concurrency-patterns/tree/main/patterns/heartbeat) |

</div>
<!-- /meta -->

## Problem

A goroutine that has produced no result for a while may be busy or stuck; from outside the two look the same. A
pulse on a separate channel lets an observer treat a missing pulse as trouble, and a pulse per unit of work tells a
test that the unit has started, which replaces a sleep.

## Mechanism

The worker owns a heartbeat channel and closes it when it ends. It sends inside a `select` with a `default` case,
so a pulse nobody waits for is dropped instead of stalling the work (drop newest, one of the
[overload options](../concepts/overload-policy.md#options), with a
[limit per consumer](../concepts/overload-policy.md#per-consumer)); a buffer of one keeps a pulse for a listener
that arrives slightly later. In this example the consumer takes each pulse before the result, so none is dropped.

<!-- code: patterns/heartbeat#pulse patterns/heartbeat#square patterns/heartbeat#Example_perUnitOfWork:main -->
```go
func pulse(heartbeat chan<- struct{}) {
	select {
	case heartbeat <- struct{}{}:
	default:
	}
}

func square(ctx context.Context, nums ...int) (<-chan struct{}, <-chan int) {
	heartbeat := make(chan struct{}, 1)
	results := make(chan int)

	go func() {
		defer close(heartbeat)
		defer close(results)

		for _, n := range nums {
			pulse(heartbeat)

			select {
			case results <- n * n:
			case <-ctx.Done():
				return
			}
		}
	}()

	return heartbeat, results
}

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	heartbeat, results := square(ctx, 2, 3, 4)

	for range 3 {
		<-heartbeat
		fmt.Println("unit started")
		fmt.Println("result:", <-results)
	}

	// Output:
	// unit started
	// result: 4
	// unit started
	// result: 9
	// unit started
	// result: 16
}
```
<!-- /code -->

**Nobody listens.** Nobody reads the heartbeat, and the work still completes.

<!-- code: patterns/heartbeat#Example_noListener:main -->
```go
func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	_, results := square(ctx, 2, 3, 4)

	for n := range results {
		fmt.Println("result:", n)
	}

	// Output:
	// result: 4
	// result: 9
	// result: 16
}
```
<!-- /code -->

## Applicability

Use it when a long-running goroutine may get stuck and something must notice, or when a test needs to know that a
goroutine has started a unit of work.

## Consequences

- A pulse shows only that the goroutine reached the send, not that its results are correct.
- Pulses are dropped when nobody waits, so an observer that falls behind cannot count them.
- A pulse per unit of work costs nothing while idle but says nothing during a long unit.
- The observer decides what a missing pulse means; [Healing Goroutines](healing-goroutines.md) builds on this.

## Variants

### Pulse at an interval

A ticker drives the pulse independently of the work, so liveness shows during long units, at the cost of a
timer wakeup per interval; the interval must be well below the observer's timeout.

<!-- code: patterns/heartbeat#ticking patterns/heartbeat#Example_interval:main -->
```go
func ticking(ctx context.Context, interval time.Duration) <-chan struct{} {
	heartbeat := make(chan struct{}, 1)

	go func() {
		defer close(heartbeat)

		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				pulse(heartbeat)
			}
		}
	}()

	return heartbeat
}

func main() {
	ctx, cancel := context.WithCancel(context.Background())

	heartbeat := ticking(ctx, time.Millisecond)

	for range 3 {
		<-heartbeat
	}

	cancel()

	for range heartbeat {
	}

	fmt.Println("alive, then stopped")

	// Output:
	// alive, then stopped
}
```
<!-- /code -->

## Related patterns

- [Healing Goroutines](healing-goroutines.md) restarts a goroutine whose pulse stops.

## Anti-patterns

<!-- antipatterns -->
- [Sleeping to Synchronize](../anti-patterns/sleep-synchronization.md): Use time.Sleep or a guessed delay to wait for another goroutine instead of waiting for its completion.
<!-- /antipatterns -->

## Sources

<!-- sources -->
- [Cox-Buday (2017)](../sources.md#source-cox-buday), Ch. 5, "Heartbeats"
<!-- /sources -->
