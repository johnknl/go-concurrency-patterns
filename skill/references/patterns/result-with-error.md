# Result with Error

Get each error from concurrent work to the code that can decide what to do with it.

- **Also known as:** Error Handling
- **Purpose:** Resilience
- **Scale:** Idiom
- **Mechanisms:** Channels
- **Concerns:** Error handling
- **Concepts:** [Goroutine Leak Prevention](../concepts/goroutine-leak-prevention.md)
- **Code:** [`patterns/resulterr`](https://github.com/johnknl/go-concurrency-patterns/tree/main/patterns/resulterr)

## Problem

A goroutine that hits an error knows what failed but not what the result is for, and it cannot return the error to
a caller, because nobody waits on its function call. If it handles the error itself, the consumer cannot see the
failure and treats a partial result as complete.

## Mechanism

Send a struct that holds the value and the error. The goroutine reports every outcome; the consumer, which knows
what the values are for, decides whether to retry, skip, or stop. When it stops early, it cancels the context, and
the goroutine's send selects on `ctx.Done()` so it can leave
([select on done](../concepts/goroutine-leak-prevention.md#select-on-done)).

```go
type result struct {
	err   error
	value string
	id    int
}

func lookupAll(ctx context.Context, ids ...int) <-chan result {
	out := make(chan result)

	go func() {
		defer close(out)

		for _, id := range ids {
			value, err := lookup(id)

			select {
			case out <- result{id: id, value: value, err: err}:
			case <-ctx.Done():
				return
			}
		}
	}()

	return out
}

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	for r := range lookupAll(ctx, 1, 2, 3, 4, 5) {
		if errors.Is(r.err, errNotFound) {
			fmt.Println("skip:", r.err)
			continue
		}

		if r.err != nil {
			fmt.Println("stop:", r.err)
			return
		}

		fmt.Println("got", r.value)
	}

	// Output:
	// got record-1
	// skip: lookup 2: not found
	// got record-3
	// skip: lookup 4: not found
	// got record-5
}
```

**The contrast: log and continue.** The goroutine logs the error and drops the item, and the consumer cannot tell that
anything went wrong.

```go
func lookupQuietly(logs *[]string, ids ...int) <-chan string {
	out := make(chan string)

	go func() {
		defer close(out)

		for _, id := range ids {
			value, err := lookup(id)
			if err != nil {
				*logs = append(*logs, err.Error())
				continue
			}

			out <- value
		}
	}()

	return out
}

func main() {
	var logs []string

	for value := range lookupQuietly(&logs, 1, 2, 3) {
		fmt.Println("got", value)
	}

	for _, line := range logs {
		fmt.Println("log:", line)
	}

	// Output:
	// got record-1
	// got record-3
	// log: lookup 2: not found
}
```

## Applicability

Use it when the failure of work in a goroutine matters to the code that started it, and the right reaction depends
on information the goroutine lacks, such as the caller's retry budget. The example's lookups run one after another,
so returning `[]result` would serve as well; the channel pays off when results arrive over time or from several
goroutines. When any failure aborts everything, use [Cancel on Error](cancel-on-error.md).

## Consequences

- Every consumer must check the error field before using the value, or it silently uses a zero value.
- A consumer that stops early must cancel the producer, or the producer blocks on its next send.
- Logging in the goroutine still helps diagnostics, but does not replace reporting the error.

## Related patterns

- [Cancel on Error](cancel-on-error.md) stops sibling work on the first failure and reports it.
- [Pipeline](pipeline.md) stages can send results of this form.
- [Future](future.md) returns a value and an error through its placeholder.

## Anti-patterns

- [Fire-and-Forget Goroutine](../anti-patterns/fire-and-forget-goroutine.md): Start a goroutine and return, so nobody can wait for it, learn of its error or stop it.

## Sources

- [Cox-Buday (2017)](../sources.md#source-cox-buday), Ch. 4, "Error Handling"
