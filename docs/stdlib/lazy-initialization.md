---
title: Lazy Initialization
aliases: ['Once', 'Initialize-on-demand']
intent: Build a value on first use, exactly once, safely from many goroutines.
provides: [sync.Once, sync.OnceFunc, sync.OnceValue, sync.OnceValues]
package: stdlib/lazyinit
concerns: [data-race-safety]
sources:
  - id: go-sync
  - id: go-memory-model
concepts:
  - memory-visibility: 'the writes of the once function are visible to every caller of Do.'
---
# Lazy Initialization

<!-- meta -->
Build a value on first use, exactly once, safely from many goroutines.

<div class="cp-props" markdown>

| | |
|---|---|
| Also known as | Once, Initialize-on-demand |
| Provides | `sync.Once`, `sync.OnceFunc`, `sync.OnceValue`, `sync.OnceValues` |
| Concerns | [Data race safety](../catalog/concern.md#concern-data-race-safety) |
| Concepts | [Memory Visibility](../concepts/memory-visibility.md) |
| Code | [`stdlib/lazyinit`](https://github.com/johnknl/go-concurrency-patterns/tree/main/stdlib/lazyinit) |

</div>
<!-- /meta -->

## What it provides

`sync.Once.Do(f)` calls `f` only on the first call; every other call, concurrent or later, returns only after `f` has
finished, and all callers see everything `f` wrote ([Once's function happens before any Do
returns](../concepts/memory-visibility.md#once-before-do)). Since Go 1.21, `sync.OnceFunc`, `sync.OnceValue` and
`sync.OnceValues` wrap a function so that it runs once and its result is cached, without a separate `Once` and variable.

Reach for them for a value that is expensive, may not be needed, and never changes once built. If it is always
needed, a package-level variable or `init` is simpler. If it must be rebuilt, use a
[Monitor](../patterns/monitor.md).

## Using it

In this example, 50 goroutines call `loadConfig` and the builder runs once.

<!-- code: stdlib/lazyinit#configOnce stdlib/lazyinit#loadConfig stdlib/lazyinit#Example_once:main -->
```go
var configOnce sync.Once

func loadConfig() *config {
	configOnce.Do(func() {
		builds++
		cfg = &config{name: "production"}
	})

	return cfg
}

func main() {
	var wg sync.WaitGroup

	names := make([]string, 50)

	for i := range names {
		wg.Go(func() { names[i] = loadConfig().name })
	}

	wg.Wait()
	fmt.Println(names[0], names[49], builds)

	// Output:
	// production production 1
}
```
<!-- /code -->

**OnceValue.** The wrapper holds the `Once` and the result itself.

<!-- code: stdlib/lazyinit#Example_onceValue:main -->
```go
func main() {
	calls := 0
	table := sync.OnceValue(func() []int {
		calls++

		squares := make([]int, 10)
		for i := range squares {
			squares[i] = i * i
		}

		return squares
	})

	var wg sync.WaitGroup

	got := make([]int, 10)

	for i := range got {
		wg.Go(func() { got[i] = table()[i] })
	}

	wg.Wait()
	fmt.Println(got, calls)

	// Output:
	// [0 1 4 9 16 25 36 49 64 81] 1
}
```
<!-- /code -->

**A cached error.** The function fails once; every later call returns the same error without running it again.

<!-- code: stdlib/lazyinit#Example_onceValuesError:main -->
```go
func main() {
	open := sync.OnceValues(func() (string, error) {
		return "", fmt.Errorf("config file missing")
	})

	for range 2 {
		_, err := open()
		fmt.Println(err)
	}

	// Output:
	// config file missing
	// config file missing
}
```
<!-- /code -->

**A panic in Once.Do and in OnceFunc.** After a panic, `Once.Do` does nothing on later calls, while the function from
`OnceFunc` panics again.

<!-- code: stdlib/lazyinit#Example_panic:main -->
```go
func main() {
	calls := 0

	var once sync.Once

	run := func() (recovered any) {
		defer func() { recovered = recover() }()

		once.Do(func() {
			calls++
			panic("boom")
		})

		return nil
	}

	fmt.Println(run(), run(), calls)

	f := sync.OnceFunc(func() {
		calls++
		panic("boom")
	})

	call := func() (recovered any) {
		defer func() { recovered = recover() }()

		f()

		return nil
	}

	fmt.Println(call(), call(), calls)

	// Output:
	// boom <nil> 1
	// boom boom 2
}
```
<!-- /code -->

## Pitfalls

- An error is not retried: `OnceValues` caches it with the value, and every call returns it.
- If `f` panics, `Once.Do` considers it returned: later calls do nothing and the value stays unset. The functions
  from `OnceFunc`, `OnceValue` and `OnceValues` panic again with the same value on every call.
- Calling `Do` on the same `Once` from inside `f` deadlocks.
- A hand-rolled unsynchronized `if !initialized` check is a data race; see
  [Double-Checked Locking](../anti-patterns/double-checked-locking.md).
- A `Once` must not be copied after first use.

## Related pages

- [Monitor](../patterns/monitor.md) for state that changes after it is built.
- [Future](../patterns/future.md) starts the work eagerly in a goroutine and lets callers wait for its result.

## Anti-patterns

<!-- antipatterns -->
- [Double-Checked Locking](../anti-patterns/double-checked-locking.md): Test a lazily initialized pointer without synchronization and lock only when it is nil, so readers race with the writer and can see a pointer to an incompletely initialized value.
<!-- /antipatterns -->

## Sources

<!-- sources -->
- [The Go Authors (n.d.e)](../sources.md#source-go-sync)
- [The Go Authors (2022)](../sources.md#source-go-memory-model)
<!-- /sources -->
