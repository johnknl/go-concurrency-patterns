---
title: Sleeping to Synchronize
intent: Use time.Sleep or a guessed delay to wait for another goroutine instead of waiting for its completion.
purpose: coordination
package: antipatterns/sleepsync
mechanisms: [time, channels, atomic]
concerns: [composition]
remedies: [future, heartbeat, watermark]
violates: [explicit-synchronization]
concepts:
  - memory-visibility: 'a sleep gives no happens-before edge.'
sources:
  - id: tu-2019
  - id: harsanyi-2022
---
# Sleeping to Synchronize

<!-- meta -->
Use time.Sleep or a guessed delay to wait for another goroutine instead of waiting for its completion.

<div class="cp-props" markdown>

| | |
|---|---|
| Violates | [Synchronize explicitly](../index.md#principle-explicit-synchronization) |
| Purpose | [Coordination](../catalog/purpose.md#purpose-coordination) |
| Mechanisms | [Time](../catalog/mechanism.md#mechanism-time), [Channels](../catalog/mechanism.md#mechanism-channels), [Atomic](../catalog/mechanism.md#mechanism-atomic) |
| Concerns | [Composition](../catalog/concern.md#concern-composition) |
| Breaks | [Memory Visibility](../concepts/memory-visibility.md) |
| Code | [`antipatterns/sleepsync`](https://github.com/johnknl/go-concurrency-patterns/tree/main/antipatterns/sleepsync) |

</div>
<!-- /meta -->

## Symptoms

- A goroutine is started and the caller sleeps, hoping the work is done when the sleep ends.
- Results sometimes missing or zero, mostly in CI or under load; tests "fixed" by raising the sleep.
- Test suites and shutdown paths that always take the full delay, longer than the work needs.
- Reports from `go test -race` on the read that follows the sleep.

## Example

=== "Faulty"

    The job finishes only after a gate that is closed after `badRead` returns, so it has certainly not finished when
    the sleep ends. The result is an atomic only to keep the example race-free; a plain variable would also race.

    <!-- code: antipatterns/sleepsync#job antipatterns/sleepsync#startJob antipatterns/sleepsync#badRead antipatterns/sleepsync#Example_sleep:main -->

    ```go
    type job struct {
    	done   chan struct{}
    	result atomic.Int64
    }

    func startJob(gate <-chan struct{}) *job {
    	j := &job{done: make(chan struct{})}

    	go func() {
    		defer close(j.done)

    		<-gate
    		j.result.Store(42)
    	}()

    	return j
    }

    func badRead(j *job) int64 {
    	time.Sleep(time.Millisecond)

    	return j.result.Load()
    }

    func main() {
    	gate := make(chan struct{})
    	j := startJob(gate)

    	// The job cannot finish before the gate opens, and the gate opens only after the sleep returned.
    	fmt.Println("read after sleep:", badRead(j))

    	close(gate)
    	<-j.done

    	fmt.Println("result:", j.result.Load())

    	// Output:
    	// read after sleep: 0
    	// result: 42
    }
    ```

    <!-- /code -->

=== "Fixed"

    The job closes `done` when finished and `read` receives from it; the close happens before the receive returns,
    so the result written before it is visible. A `sync.WaitGroup` or a [future](../patterns/future.md) works too.

    <!-- code: antipatterns/sleepsync#read antipatterns/sleepsync#Example_wait:main -->

    ```go
    func read(j *job) int64 {
    	<-j.done

    	return j.result.Load()
    }

    func main() {
    	gate := make(chan struct{})
    	j := startJob(gate)

    	go close(gate)

    	fmt.Println("read after wait:", read(j))

    	// Output:
    	// read after wait: 42
    }
    ```

    <!-- /code -->

## Why it goes wrong

`time.Sleep` waits for time, not for another goroutine's state, so no duration is correct: too short on a loaded
machine, too long everywhere else. Only synchronization (channels, mutexes, `sync.WaitGroup`, atomics) orders events
between goroutines; a sleep [does not](../concepts/memory-visibility.md#no-edge), so a plain read after it races with
the write.

## Detection

- `time.Sleep` that is not a deliberate delay, especially right after a `go` statement and in tests.
- `go test -race` reports the read when the result is a plain variable and both accesses happen in the run, even when
  the sleep is long enough for the value to be correct. No `go vet` analyzer checks this.
- Tests under load (`-cpu`, `-count`, `stress`) make the failures appear.

## Fixed by

<!-- remedies -->
- [Future](../patterns/future.md): Overlap independent operations and collect each result when it is needed.
- [Heartbeat](../patterns/heartbeat.md): Tell a slow worker from a stuck one, and know when work has started, without sleeping.
- [Watermark](../patterns/watermark.md): Act on events in the order of their timestamps when they arrive from several inputs at different speeds, without guessing how long to wait for the slower ones.
<!-- /remedies -->

For long-running work, a heartbeat lets the waiter tell a slow job from a stuck one, which a sleep cannot.

## Sources

<!-- sources -->
- [Tu et al. (2019)](../sources.md#source-tu-2019)
- [Harsanyi (2022)](../sources.md#source-harsanyi-2022)
<!-- /sources -->
