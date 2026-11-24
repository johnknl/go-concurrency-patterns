# Fire-and-Forget Goroutine

Start a goroutine and return, so nobody can wait for it, learn of its error or stop it.

- **Violates:** [Clear ownership](../principles.md#principle-ownership)
- **Purpose:** Lifecycle
- **Mechanisms:** Channels, Sync
- **Concerns:** Leak prevention, Error handling
- **Breaks:** [Goroutine Leak Prevention](../concepts/goroutine-leak-prevention.md)
- **Code:** [`antipatterns/fireandforget`](https://github.com/johnknl/go-concurrency-patterns/tree/main/antipatterns/fireandforget)

## Symptoms

- Errors from the work are logged at best and lost at worst; the caller reports success.
- Work is cut off when `main` returns or a test finishes.
- A panic in the goroutine terminates the whole process; the caller cannot `recover` it.
- Tests that check the result are flaky or need sleeps, and goroutines accumulate when the work blocks.

## Example

**Faulty**

`leakyStart` runs `go work()` and returns `nil` whatever happens. A `report` hook stands in for a log line: the
error reaches only the hook.

```go
func leakyStart(work func() error, report func(error)) error {
	if work == nil {
		return errors.New("no work")
	}

	go func() {
		if err := work(); err != nil {
			report(err)
		}
	}()

	return nil
}

func main() {
	lost := make(chan error)

	err := leakyStart(func() error { return errors.New("disk full") }, func(err error) { lost <- err })
	fmt.Println("start returned:", err)

	// Only the injected hook ever sees the error; receiving here also lets the goroutine finish.
	fmt.Println("error seen elsewhere:", <-lost)

	// Output:
	// start returned: <nil>
	// error seen elsewhere: disk full
}
```

**Fixed**

The function that starts a goroutine waits for it and returns its error. `runAll` keeps one error slot per task,
waits with a `sync.WaitGroup`, and returns the errors joined with `errors.Join`.

```go
func runAll(tasks ...func() error) error {
	errs := make([]error, len(tasks))

	var wg sync.WaitGroup

	for i, task := range tasks {
		wg.Go(func() { errs[i] = task() })
	}

	wg.Wait()

	return errors.Join(errs...)
}

func main() {
	err := runAll(
		func() error { return nil },
		func() error { return errors.New("disk full") },
		func() error { return errors.New("timeout") },
	)
	fmt.Println(err)

	// Output:
	// disk full
	// timeout
}
```

## Why it goes wrong

A goroutine has no result and no handle. The starter can learn about it only through channels, a `sync.WaitGroup`
or shared state set up before `go`, and a fire-and-forget function sets up none. The runtime does not wait for
goroutines when `main` returns, does not carry their errors anywhere, and does not stop them. A panic not recovered
inside the goroutine itself ends the process. A goroutine started from `init` is the extreme case: no caller exists
to own it, and only package-level state can reach it. Every goroutine needs an owner responsible for its end
([goroutine leak prevention](../concepts/goroutine-leak-prevention.md#owner-ends)).

## Detection

- For every `go` statement, ask who waits for it, who receives its error, and what stops it.
- `go vet` does not check for this; the race detector reports it only when the goroutine's writes race with a read.
- A goroutine dump, the `pprof` goroutine profile, or a check for leftover goroutines at the end of a test shows
  goroutines that outlive their starter. The `goroutineleak` profile reports those blocked on a channel or lock that
  no running goroutine can reach.
- A test that needs `time.Sleep` to see a function's effect hints at work nobody can wait for.

## Fixed by

- [Result with Error](../patterns/result-with-error.md): Get each error from concurrent work to the code that can decide what to do with it.

If the work must outlive the call, return a handle (a done channel, an error channel or a future), add a context or
done channel to stop it, and recover panics at the goroutine boundary if they must not take the process down.

## Sources

- [Tu et al. (2019)](../sources.md#source-tu-2019)
- [Harsanyi (2022)](../sources.md#source-harsanyi-2022)
