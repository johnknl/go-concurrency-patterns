# Premature Abstraction

Extract a generic, configurable concurrency helper before its uses are known, so each caller sets flags for the behavior it needs, the flags interact, and every caller depends on code paths it does not use.

- **Violates:** [Plain code over generality](../principles.md#principle-plain-code)
- **Purpose:** Execution
- **Mechanisms:** Channels, Sync
- **Concerns:** Composition
- **Code:** [`antipatterns/prematureabstraction`](https://github.com/johnknl/go-concurrency-patterns/tree/main/antipatterns/prematureabstraction)

## Symptoms

- The second time a worker pool is needed it becomes a generic helper, `pool[J, R any]`, with an options struct
  (workers, buffer size, ordered results, stop on first error, error callback), each option added for one caller.
- Boolean options that switch modes, with branches on them inside the helper's loop.
- Callers that each set a different subset, and none that sets them all.
- Options that only make sense together, or that silently override each other.
- A result whose meaning depends on the flags: zero values that may be results or failures, a slice that may be
  shorter than the input.

## Example

**Faulty**

The first caller sets `ordered` and `onError`; the failed job leaves an empty string that looks like a result.
The second sets `stopOnError` and `onError`; `stopOnError` takes precedence, and the log callback never runs.

```go
type options struct {
	onError     func(i int, err error)
	workers     int
	buffer      int
	ordered     bool
	stopOnError bool
}

func pool[J, R any](jobs []J, f func(J) (R, error), o options) ([]R, error) {
	if o.workers < 1 {
		o.workers = 1
	}

	var (
		mu       sync.Mutex
		out      []R
		firstErr error
		stopped  atomic.Bool
		wg       sync.WaitGroup
	)

	if o.ordered {
		out = make([]R, len(jobs))
	}

	in := make(chan int, o.buffer)

	for range o.workers {
		wg.Go(func() {
			for i := range in {
				if o.stopOnError && stopped.Load() {
					continue
				}

				r, err := f(jobs[i])

				mu.Lock()

				switch {
				case err != nil && o.stopOnError:
					if firstErr == nil {
						firstErr = err
					}

					stopped.Store(true)
				case err != nil && o.onError != nil:
					o.onError(i, err)
				case err != nil:
					// dropped
				case o.ordered:
					out[i] = r
				default:
					out = append(out, r)
				}

				mu.Unlock()
			}
		})
	}

	for i := range jobs {
		in <- i
	}

	close(in)
	wg.Wait()

	if firstErr != nil {
		return nil, firstErr
	}

	return out, nil
}

func main() {
	names := []string{"a", "bad", "c"}

	var errs []error

	out, _ := pool(names, process, options{
		workers: 2,
		ordered: true,
		onError: func(_ int, err error) { errs = append(errs, err) },
	})
	fmt.Printf("%q %v\n", out, errs)

	var mu sync.Mutex

	logged := 0

	_, err := pool(names, process, options{
		workers:     2,
		stopOnError: true,
		onError: func(int, error) {
			mu.Lock()
			logged++
			mu.Unlock()
		},
	})
	fmt.Println(err, "logged:", logged)

	// Output:
	// ["a.done" "" "c.done"] [bad: bad input]
	// bad: bad input logged: 0
}
```

**Fixed**

Each call site writes its own worker pool with only what it needs: the first keeps results and errors by
index, the second cancels a context at the first error and logs it. Changes to one do not reach the other.

```go
func processAll(names []string) ([]string, error) {
	out := make([]string, len(names))
	errs := make([]error, len(names))
	in := make(chan int)

	var wg sync.WaitGroup
	for range 2 {
		wg.Go(func() {
			for i := range in {
				out[i], errs[i] = process(names[i])
			}
		})
	}

	for i := range names {
		in <- i
	}

	close(in)
	wg.Wait()

	return out, errors.Join(errs...)
}

func processUntilError(ctx context.Context, names []string, logf func(error)) error {
	ctx, cancel := context.WithCancelCause(ctx)
	defer cancel(nil)

	in := make(chan string)

	var wg sync.WaitGroup
	for range 2 {
		wg.Go(func() {
			for name := range in {
				if ctx.Err() != nil {
					continue
				}

				if _, err := process(name); err != nil {
					logf(err)
					cancel(err)
				}
			}
		})
	}

	for _, name := range names {
		in <- name
	}

	close(in)
	wg.Wait()

	return context.Cause(ctx)
}

func main() {
	names := []string{"a", "bad", "c"}

	out, err := processAll(names)
	fmt.Printf("%q %v\n", out, err)

	logged := 0
	err = processUntilError(context.Background(), names, func(error) { logged++ })
	fmt.Println(err, "logged:", logged)

	// Output:
	// ["a.done" "" "c.done"] bad: bad input
	// bad: bad input logged: 1
}
```

## Why it goes wrong

Two order flags and three error modes make six modes, of which the callers use two; the other four, including one
that drops errors without a trace, still have to be read, tested, and maintained. The flags interact (`stopOnError`
turns `onError` off; `ordered` decides between a zero value and a gap), so a caller must read the helper to predict
a combination, which is what the abstraction was meant to spare it. The next special case becomes another flag, and
removing the helper later means untangling which caller relies on which combination.

The mistake is timing: "duplication is far cheaper than the wrong abstraction" (Metz), and "a little copying is
better than a little dependency" (Go proverb). An abstraction earns its place when the code repeats for the same
reason, has a plain name, and no caller needs a flag to make it behave.

## Detection

- A generic concurrency helper with a configuration struct and fewer than three callers.
- Boolean options, especially pairs whose combination is not documented or tested.
- A `switch` or `if` chain on options inside the per-item loop.

## Fixed by

- [Worker Pool](../patterns/worker-pool.md): Process any number of jobs with a fixed amount of concurrency.
- [Future](../patterns/future.md): Overlap independent operations and collect each result when it is needed.

Use [Future](../patterns/future.md) when a caller needs one result in the background rather than a pool.

## Sources

- [Metz (2016)](../sources.md#source-metz-2016)
- [Pike (2015)](../sources.md#source-pike-2015), "A little copying is better than a little dependency"
