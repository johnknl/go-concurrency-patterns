---
title: Needless Asynchrony
intent: Start a goroutine inside a function and hand back a channel, where returning the result would do, so the caller loses the error path, the cancellation, and control over the goroutine.
purpose: execution
package: antipatterns/needlessasync
mechanisms: [channels]
concerns: [leak-prevention, error-handling, cancellation]
remedies: [future]
violates: [synchronous-first]
concepts:
  - goroutine-leak-prevention: 'the callee starts a goroutine the caller must end.'
sources:
  - id: go-code-review-comments
    at: 'Synchronous Functions'
---
# Needless Asynchrony

<!-- meta -->
Start a goroutine inside a function and hand back a channel, where returning the result would do, so the caller loses the error path, the cancellation, and control over the goroutine.

<div class="cp-props" markdown>

| | |
|---|---|
| Violates | [Synchronous first](../index.md#principle-synchronous-first) |
| Purpose | [Execution](../catalog/purpose.md#purpose-execution) |
| Mechanisms | [Channels](../catalog/mechanism.md#mechanism-channels) |
| Concerns | [Leak prevention](../catalog/concern.md#concern-leak-prevention), [Error handling](../catalog/concern.md#concern-error-handling), [Cancellation](../catalog/concern.md#concern-cancellation) |
| Breaks | [Goroutine Leak Prevention](../concepts/goroutine-leak-prevention.md) |
| Code | [`antipatterns/needlessasync`](https://github.com/johnknl/go-concurrency-patterns/tree/main/antipatterns/needlessasync) |

</div>
<!-- /meta -->

## Symptoms

- A function such as `func User(id int) <-chan Result` wraps an ordinary call that could return `(string, error)`.
- Callers receive from the returned channel on the next line, so nothing runs concurrently.
- Errors travel in a result struct, or are logged inside the goroutine and lost.
- The function takes no context, or takes one and still cannot stop work that the caller stopped waiting for.
- Goroutines pile up, blocked on a send, after callers returned early on another error or a timeout.

## Example

=== "Faulty"

    `userAsync` starts a goroutine per call and returns an unbuffered channel. The abandoned first call stays blocked
    on its send; the second is received at once, so its goroutine bought nothing.

    <!-- code: antipatterns/needlessasync#result antipatterns/needlessasync#userAsync antipatterns/needlessasync#Example_needlessAsync:main -->

    ```go
    type result struct {
    	err  error
    	name string
    }

    func userAsync(id int) <-chan result {
    	ch := make(chan result)

    	pending.Add(1)

    	go func() {
    		defer pending.Add(-1)

    		name, err := lookup(id)
    		ch <- result{name: name, err: err}
    	}()

    	return ch
    }

    func main() {
    	abandoned := userAsync(2) // the caller returns early and never receives
    	fmt.Println("goroutines blocked:", pending.Load())

    	r := <-userAsync(1)
    	fmt.Println(r.name, r.err)

    	<-abandoned // only so that the example does not leave the goroutine behind

    	// Output:
    	// goroutines blocked: 1
    	// user-1 <nil>
    }
    ```

    <!-- /code -->

=== "Fixed"

    Return the result and take a context if the work can block. The caller decides whether to run it concurrently
    and keeps the errors and the goroutines it starts.

    <!-- code: antipatterns/needlessasync#user antipatterns/needlessasync#Example_synchronous:main -->

    ```go
    func user(ctx context.Context, id int) (string, error) {
    	if err := ctx.Err(); err != nil {
    		return "", err
    	}

    	return lookup(id)
    }

    func main() {
    	ctx := context.Background()

    	var (
    		a, b       string
    		errA, errB error
    		wg         sync.WaitGroup
    	)

    	wg.Go(func() { a, errA = user(ctx, 1) })
    	wg.Go(func() { b, errB = user(ctx, -1) })
    	wg.Wait()

    	fmt.Println(a, b == "")
    	fmt.Println(errors.Join(errA, errB))

    	// Output:
    	// user-1 true
    	// user -1: not found
    }
    ```

    <!-- /code -->

## Why it goes wrong

The goroutine's lifetime ends only when somebody receives, so a caller that stops listening leaks it. A buffer of one
removes the leak but not the rest: the caller still cannot cancel, the error still arrives in a struct, and every call
pays for a goroutine and a channel. The signature also decides for every caller that the work runs concurrently;
concurrency built into an API is hard to remove, while a caller that wants it can add one `go` statement and own the
goroutine ([goroutine leak prevention](../concepts/goroutine-leak-prevention.md#owner-ends)).

A channel is the right return type for events over time (`time.Ticker`, `signal.Notify`) or a stream that arrives on
its own, such as a socket. A computed sequence is not such a stream; see [Channel Generator](channel-generator.md).

## Detection

- `go func() { ... ch <- v }()` followed by `return ch`, for work that does not produce values over time.
- Call sites that receive immediately: `<-f(...)`.
- Goroutine and goroutine leak profiles show the abandoned sends; `go vet` and the race detector do not report it.

## Fixed by

<!-- remedies -->
- [Future](../patterns/future.md): Overlap independent operations and collect each result when it is needed.
<!-- /remedies -->

A [Future](../patterns/future.md) at the call site is the asynchronous form the caller chooses; for a sequence,
`iter.Seq` gives lazy values without a goroutine.

## Sources

<!-- sources -->
- [The Go Authors (n.d.b)](../sources.md#source-go-code-review-comments), Synchronous Functions
<!-- /sources -->
