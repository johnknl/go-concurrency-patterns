# Context Propagation

Stop all the work done for a request together, and give it the request's deadline and scoped values.

- **Also known as:** Cancellation tree, Request context
- **Provides:** `context.Context`, `context.Background`, `context.WithCancel`, `context.WithCancelCause`, `context.WithDeadline`, `context.WithTimeout`, `context.WithValue`, `context.WithoutCancel`, `context.AfterFunc`, `context.Cause`
- **Concerns:** Cancellation, Composition
- **Concepts:** [Cooperative Cancellation](../concepts/cooperative-cancellation.md)
- **Code:** [`stdlib/contextprop`](https://github.com/johnknl/go-concurrency-patterns/tree/main/stdlib/contextprop)

## What it provides

The `context` package carries a stop signal, a deadline, a cancellation cause, and request-scoped values across
calls, goroutines, and package boundaries. Deriving with `WithCancel`, `WithCancelCause`, `WithTimeout`,
`WithDeadline`, or `WithValue` adds a node to a tree rooted at `context.Background()` or a framework's context, such
as `(*http.Request).Context()`. Canceling a context closes its `Done` channel and those of all its descendants, never
its parent or siblings. `Err` then returns `context.Canceled` or `context.DeadlineExceeded`.

Reach for it for request-scoped work that may be canceled or time out, such as network or database calls, and for
goroutines started for a request. Inside one package, where only a stop signal is needed, a done channel is enough.

## Using it

Each function receives a `context.Context` as its first parameter, named `ctx`, and passes it, or a context derived from
it, to everything it calls or starts. Code that blocks selects on `ctx.Done()`, returns, and reports `ctx.Err()`
([return the cause](../concepts/cooperative-cancellation.md#return-cause)). In this example, canceling one branch leaves
its parent and sibling running; canceling the root stops the rest.

```go
func worker(ctx context.Context, name string, result chan<- string) {
	<-ctx.Done()

	result <- fmt.Sprintf("%s: %v", name, ctx.Err())
}

func main() {
	root, cancelRoot := context.WithCancel(context.Background())
	defer cancelRoot()

	left, cancelLeft := context.WithCancel(root)
	defer cancelLeft()

	right, cancelRight := context.WithCancel(root)
	defer cancelRight()

	cancelLeft()
	<-left.Done()
	fmt.Println("left:", left.Err(), "| root:", root.Err(), "| right:", right.Err())

	result := make(chan string)

	go worker(right, "right", result)

	cancelRoot()
	fmt.Println(<-result)
	fmt.Println("root:", root.Err())

	// Output:
	// left: context canceled | root: <nil> | right: <nil>
	// right: context canceled
	// root: context canceled
}
```

### Derived deadline

A derived deadline ends the child without affecting its parent, and is never later than the parent's: a callee
can tighten its caller's limit but not extend it.

```go
func main() {
	parent, cancelParent := context.WithCancel(context.Background())
	defer cancelParent()

	child, cancel := context.WithDeadline(parent, time.Unix(0, 0))
	defer cancel()

	<-child.Done()
	fmt.Println(child.Err())
	fmt.Println(errors.Is(child.Err(), context.DeadlineExceeded))
	fmt.Println(parent.Err())

	// Output:
	// context deadline exceeded
	// true
	// <nil>
}
```

### Cancellation cause

With `WithCancelCause` the canceler records an error: `ctx.Err()` in every descendant still returns
`context.Canceled`, and `context.Cause` returns the recorded error. `WithTimeoutCause` and `WithDeadlineCause`
set the cause used when the deadline passes.

```go
func call(ctx context.Context, causes chan<- error) {
	<-ctx.Done()

	causes <- context.Cause(ctx)
}

func main() {
	root, cancel := context.WithCancelCause(context.Background())
	defer cancel(nil)

	child, stop := context.WithTimeout(root, time.Hour)
	defer stop()

	causes := make(chan error)

	go call(child, causes)
	go call(child, causes)

	cancel(errors.New("client gone"))

	fmt.Println(<-causes)
	fmt.Println(<-causes)
	fmt.Println(child.Err())

	// Output:
	// client gone
	// client gone
	// context canceled
}
```

### Request-scoped values

Values are for data that travels with a request across API boundaries, such as a request ID or an authenticated
principal. Use an unexported key type, so keys from different packages cannot collide, and wrap access in typed
functions.

```go
type requestKey struct{}

func withRequestID(parent context.Context, id string) context.Context {
	return context.WithValue(parent, requestKey{}, id)
}

func requestID(c context.Context) (string, bool) {
	id, ok := c.Value(requestKey{}).(string)

	return id, ok
}

func main() {
	c := withRequestID(context.Background(), "req-42")

	child, cancel := context.WithCancel(c)
	defer cancel()

	id, ok := requestID(child)
	fmt.Println(id, ok)

	_, ok = requestID(context.Background())
	fmt.Println(ok)

	// Output:
	// req-42 true
	// false
}
```

### Detached and callback contexts

`WithoutCancel` keeps the parent's values but is not canceled with it, for work that must outlive the request, such
as an audit write. `AfterFunc` runs a function in its own goroutine once a context is done, to interrupt operations
that cannot select on `Done`. It also combines stop signals: `context.AfterFunc(other, cancel)` cancels this context
when another one ends, so one `ctx.Done()` covers both.

## Pitfalls

- Not calling the cancel function on every path: the context and its timer are not released until the parent ends
  or the timer fires. Use `defer cancel()`; `go vet`'s `lostcancel` check reports some of these.
- Cancellation is cooperative: a goroutine that neither checks `Done` nor passes the context on keeps running
  ([signal, not effect](../concepts/cooperative-cancellation.md#signal-not-effect)).
- Storing a context in a struct instead of passing it explicitly; the `context` documentation asks for the latter.
- Using values for optional parameters. Value lookup walks up the chain and the values are untyped; keep them few
  and small.
- Expecting `Err` to say why: after `WithCancelCause` it is still `context.Canceled`; use `context.Cause`.

## Related pages

- [Deadline](deadline.md) bounds a single operation.
- [Close with Cause](../patterns/close-with-cause.md) records why a context or channel ended.
- [Pipeline](../patterns/pipeline.md) uses a context to stop its stages.

## Anti-patterns

- [Ignored Context](../anti-patterns/ignored-context.md): A function takes a context and never checks it, so cancellation and deadlines have no effect.

## Sources

- [Cox-Buday (2017)](../sources.md#source-cox-buday), Ch. 4, "The context Package"
- [Ajmani (2014a)](../sources.md#source-blog-context)
