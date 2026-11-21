---
title: Closing from the Wrong Side
intent: A channel is closed by a goroutine that does not own it, so a later send or a second close panics.
purpose: lifecycle
package: antipatterns/wrongsideclose
mechanisms: [channels, sync]
concerns: [leak-prevention, error-handling]
remedies: [close-with-cause]
violates: [ownership]
concepts:
  - channel-ownership: 'a receiver closes a channel it does not own.'
  - goroutine-leak-prevention: 'each sender selects on done so it can leave.'
sources:
  - id: tu-2019
---
# Closing from the Wrong Side

<!-- meta -->
A channel is closed by a goroutine that does not own it, so a later send or a second close panics.

<div class="cp-props" markdown>

| | |
|---|---|
| Violates | [Clear ownership](../index.md#principle-ownership) |
| Purpose | [Lifecycle](../catalog/purpose.md#purpose-lifecycle) |
| Mechanisms | [Channels](../catalog/mechanism.md#mechanism-channels), [Sync](../catalog/mechanism.md#mechanism-sync) |
| Concerns | [Leak prevention](../catalog/concern.md#concern-leak-prevention), [Error handling](../catalog/concern.md#concern-error-handling) |
| Breaks | [Channel Ownership](../concepts/channel-ownership.md), [Goroutine Leak Prevention](../concepts/goroutine-leak-prevention.md) |
| Code | [`antipatterns/wrongsideclose`](https://github.com/johnknl/go-concurrency-patterns/tree/main/antipatterns/wrongsideclose) |

</div>
<!-- /meta -->

## Symptoms

- A receiver closes a channel to say "enough", or several senders each close it when they finish.
- The program crashes with `panic: send on closed channel` or `panic: close of closed channel`, in whichever
  goroutine performs the operation; unrecovered, it ends the program.
- The panics depend on timing, so they may appear only under load or at shutdown.

## Example

=== "Faulty"

    Each function performs the wrong-side operation directly and recovers the panic so the example can print it. In
    a real program the close and the send are in different goroutines, in no fixed order.

    <!-- code: antipatterns/wrongsideclose#badSendAfterClose antipatterns/wrongsideclose#badCloseTwice antipatterns/wrongsideclose#Example_wrongSideClose:main -->

    ```go
    func badSendAfterClose() (msg any) {
    	defer func() { msg = recover() }()
    	ch := make(chan int, 1)
    	close(ch)
    	ch <- 1
    	return nil
    }

    func badCloseTwice() (msg any) {
    	defer func() { msg = recover() }()
    	ch := make(chan int)
    	close(ch)
    	close(ch)
    	return nil
    }

    func main() {
    	fmt.Println(badSendAfterClose())
    	fmt.Println(badCloseTwice())
    	// Output:
    	// send on closed channel
    	// close of closed channel
    }
    ```

    <!-- /code -->

=== "Fixed"

    Only the sender closes, usually with `defer close(out)`; a receiver that wants to stop early closes a separate
    `done` channel (or cancels a context), which the sender watches in a `select` around each send.

    <!-- code: antipatterns/wrongsideclose#produce antipatterns/wrongsideclose#Example_fixed:main -->

    ```go
    func produce(senders int, done <-chan struct{}) <-chan int {
    	out := make(chan int)
    	var wg sync.WaitGroup
    	for i := range senders {
    		wg.Go(func() {
    			for j := range 2 {
    				select {
    				case out <- i*10 + j:
    				case <-done:
    					return
    				}
    			}
    		})
    	}
    	go func() {
    		wg.Wait()
    		close(out)
    	}()
    	return out
    }

    func main() {
    	sum := 0
    	for v := range produce(3, nil) {
    		sum += v
    	}
    	fmt.Println("sum:", sum)

    	done := make(chan struct{})
    	out := produce(3, done)
    	<-out
    	close(done)
    	for range out {
    	}
    	fmt.Println("stopped early")
    	// Output:
    	// sum: 63
    	// stopped early
    }
    ```

    <!-- /code -->

## Why it goes wrong

`close` is the [sender's](../concepts/channel-ownership.md#sender-closes) statement that no more values will come.
After it, every send [panics](../concepts/channel-ownership.md#close-panics), including one already blocked, and a
second close panics. A receiver that closes breaks every unfinished sender; among several senders,
none knows whether the others are done. Checking first does not help: a receive cannot tell a closed channel from a
zero value without the two-value form, which consumes a value, and another goroutine can close between check and send.

With several senders, none closes: [one closer goroutine](../concepts/channel-ownership.md#closer-waits) waits for
them with a `sync.WaitGroup`, then closes. The `done` case in each sender's `select` is
[essential](../concepts/goroutine-leak-prevention.md#select-on-done), or a sender blocked after the receiver stopped
leaks; the
receiver then drains the data channel until it is closed, which guarantees every sender has returned.

## Detection

- The panic's stack trace points at the send or the second close.
- `-race` reports a close not ordered with a send on the same channel, if both happen during the run. No `go vet`
  analyzer checks who closes a channel.
- In review: `close(ch)` in a function that only receives from `ch`, or `close` calls in more than one place.

## Fixed by

<!-- remedies -->
- [Close with Cause](../patterns/close-with-cause.md): Tell the receivers of a closed channel why it was closed.
<!-- /remedies -->

## Sources

<!-- sources -->
- [Tu et al. (2019)](../sources.md#source-tu-2019)
<!-- /sources -->
