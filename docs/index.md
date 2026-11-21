# Go Concurrency Patterns

The recurring arrangements of goroutines, channels, `context`, and `sync`, named and described like design
patterns: the problem, the mechanism in plain Go, when to use it, and what it costs. Each pattern is a small package
in [the repository](https://github.com/johnknl/go-concurrency-patterns); the code on the pages comes straight from it,
and it is tested, including with mutation tests.

## Principles

Seven principles decide what is a pattern here and what is an [anti-pattern](anti-patterns/index.md). They follow
Go's own guidance, cited with each, but they are this catalog's principles, not rules of the language.

### 1. Clear ownership { #principle-ownership }

Every goroutine, channel, and mutable value has one owner, which ends and waits for its goroutines, receives their
results and errors, and alone closes the channel or writes the value. *Ask: who stops it, who waits for it, who
closes it, who may write it?* ([Goroutine Lifetimes](https://go.dev/wiki/CodeReviewComments#goroutine-lifetimes))

### 2. Synchronous first { #principle-synchronous-first }

A function does its work in the caller's goroutine and returns; concurrency is the caller's choice and needs a
reason. Sequences are callbacks, iterators, or `Next` methods. *Ask: does anything have to run while the caller
runs?* ([Synchronous Functions](https://go.dev/wiki/CodeReviewComments#synchronous-functions))

### 3. The simplest mechanism that works { #principle-simplest-mechanism }

Go up this list only as far as needed: a plain function; `go` with a `sync.WaitGroup`
([Fan-Out/Fan-In](patterns/fan-out-fan-in.md)); a mutex next to its state ([Monitor](patterns/monitor.md)); a
channel that hands off data or work ([Pipeline](patterns/pipeline.md), [Worker Pool](patterns/worker-pool.md)); a
goroutine that owns the state ([Active Object](patterns/active-object.md)); a composition
([PubSub Hub](patterns/pubsub-hub.md)). *Ask: would the step below work?*
([Use a sync.Mutex or a channel?](https://go.dev/wiki/MutexOrChannel))

### 4. Synchronize explicitly { #principle-explicit-synchronization }

Every ordering the code relies on comes from a channel, a lock, an atomic, or a wait group, never from timing. Locks
are short, never held across a blocking call, and taken in one order. *Ask: what makes this happen before that?*
([The Go memory model](https://go.dev/ref/mem))

### 5. Bound everything { #principle-bounded }

Goroutines, queues, retries, and memory have limits, and the code decides what happens at the limit: wait, drop, or
fail. *Ask: what is the limit, and what happens when it is reached?*

### 6. Plain code over generality { #principle-plain-code }

A type, helper, or dependency earns its place by removing real repetition; misuse is handled by documented
contracts, as for `time.Ticker.C`. *Ask: would copying the few lines be simpler?*
([Go Proverbs](https://go-proverbs.github.io/))

### 7. Prefer explicit inputs { #principle-explicit-inputs }

Prefer code whose behavior follows from its explicit inputs: its state, the events it receives, and injected
dependencies such as a clock or a random source. Such code can be tested on its own and a failure reproduced from
the inputs that caused it. It is a preference, not a rule: where nothing is replayed or compared, a simpler design
that lets the scheduler or the clock decide is often the better choice (principle 3). *Ask: will anyone need the
same inputs to give the same result, and does that justify the structure?*
([Schneider (1990)](https://doi.org/10.1145/98163.98167))

## Anti-patterns

An anti-pattern violates a principle where a solution that follows them exists, whether it fails yet or not. Leaks,
races, deadlocks, lost errors, and unbounded memory are how it eventually shows. A design that only makes it possible
for others to violate a principle is not one.

## Finding your way

The [catalog](catalog/index.md) lists the patterns by purpose, scale, mechanism, and concern, with guides that
compare the patterns for a concern. The [anti-patterns](anti-patterns/index.md) are listed by the principle they
violate, and every page cites its sources in the [bibliography](sources.md).
