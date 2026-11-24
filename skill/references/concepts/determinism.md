# Determinism

A computation is deterministic when the same starting state and the same inputs, in the same order, always produce the same result.

- **Also known as:** Reproducibility, Replayability
- **Concerns:** Determinism

## Why it matters

A computation whose result depends on something that is not among its inputs cannot be repeated. A test fails once
in a thousand runs and never under the debugger; the trace of events that led to a failure does not lead to it again;
a transition cannot be tested on its own because its result depends on what else was running.
In each case the hidden input is a choice made by the scheduler, the clock, a random source, or the runtime, and
nobody wrote it down. Determinism is what the preference for [explicit inputs](../principles.md#principle-explicit-inputs)
buys: everything that decides the outcome is passed in, so the outcome can be reproduced.

## Properties

### Same state and same inputs give the same result { #same-inputs }

A deterministic step is a function of the current state and the input it receives, nothing else. Applied to the
same state with the same inputs in the same order, a sequence of steps produces the same states and the same
outputs. Order is part of the input: two inputs applied in a different order may give a different result, so a
replay must reproduce the order too (see [Ordering](ordering.md)). Two runs that start in the same state and apply the same events in the same
order end in the same state.

### Sources of nondeterminism in Go { #sources }

Go programs pick up hidden inputs from several places:

- `select` with more than one ready case chooses one uniformly at random.
- Map iteration order is unspecified and deliberately varies between runs.
- Goroutine scheduling: which goroutine runs first, and how their steps interleave, is up to the runtime and
  depends on `GOMAXPROCS`, load, and timing.
- Time: `time.Now`, timers and tickers return values that differ on every run (see [Timers](timers.md)).
- Randomness: `math/rand` and `crypto/rand`.
- Concurrent appends: results collected from several goroutines into one slice or channel arrive in completion
  order, not submission order.
- External calls: the network, the file system, other processes, and their errors and latencies.
- The environment: environment variables, flags, host name, process id, configuration read at run time.

None of these is wrong in itself. Each becomes a problem only where a step that must be reproducible reads it
directly.

### Nondeterminism is an unrecorded input { #materialize }

A nondeterministic choice is an input that nobody recorded. The remedy is to materialize it: make the choice
before the step, pass it in as part of the input (a field of the event, an argument), and record it where replay
matters. A timeout becomes an event `timedOut{at}`; a random backoff becomes a field `delay` chosen by the caller;
which of two ready channels won becomes the event that the owner received and logged. Randomness cannot be
removed, only moved out of the step and recorded; after that, replaying the record gives the same result.

### Inject the clock and the random source { #inject }

The clock and the random source belong to the owner that drives the steps, not to the step. The owner receives
them as dependencies (a `func() time.Time`, a `*rand.Rand` with a known seed, an interface) and reads them once,
outside the transition, turning the value into an input. A transition never calls `time.Now` or `rand.Int`
itself. Tests then pass a fixed clock or seed, and a replay passes the recorded values.

```go
now := o.clock()             // read outside, by the owner
s, effs := next(s, tick{now}) // the transition only sees the value
```

### A pure transition returns effects as values { #pure-transition }

The strongest form keeps the deciding step pure: `next(state, event) (state, []effect)` returns the new state and
a list of effects (send a reply, start a timer, write a record) as plain values, and performs none of them. The
owner executes the effects; their results, including errors and elapsed time, come back as new events. All
nondeterminism then lives in the owner and in the effects, and everything the transition decided can be replayed
by folding the recorded events through `next` without touching the outside world.

### Where determinism is needed, and where not { #where-needed }

Determinism is required where a computation is run more than once and the runs must agree: tests that must pass or
fail reliably, reproducing a failure from the events that led to it, and comparing two implementations on the same
events. It is not required for work whose result is consumed once and not
reproduced: a fan-out whose results are combined by a commutative operation, a load balancer choosing a worker, a
cache eviction, a retry jitter that is not replayed. Forcing determinism there costs throughput and code for no
benefit, and the simpler nondeterministic design is the better one. The question is always whether anyone will run the step again and expect the same answer.

### Reproducing a failure { #reproduce }

A concurrency failure that depends on scheduling must be made to recur before it can be fixed. The tools, roughly
from cheapest:

- `go test -race` detects data races on the interleavings that actually happen.
- `-count=N` repeats tests to hit rare interleavings; `-shuffle=on` (with the printed seed) varies test order.
- Varying `GOMAXPROCS` (1 and more than the number of cores) changes scheduling.
- Hooks in the code under test, channels the test controls, that force a specific interleaving instead of waiting
  for it.
- `testing/synctest` runs goroutines in a bubble with a fake clock that advances only when every goroutine in the
  bubble is durably blocked, so code that uses time runs deterministically and without real sleeps
  (see [Synctest](../stdlib/synctest.md)).

A failure that can be reproduced only by luck points to an unrecorded input; materializing it usually also makes
the test deterministic.

### Modelled nondeterminism is fine { #modelled }

A model may be nondeterministic on purpose: a state machine where state A can go to B or to C, a test or model
checker that explores every order of events. That is a description of the possible behaviors, not a defect. The
defect is unrecorded nondeterminism in an implementation that is meant to be replayed: a choice made inside the
step, from a source nobody logged, so the replay takes a different branch than the original run. Keep the choice;
make it an input.

## Used by

- [State Machine](../patterns/state-machine.md): the transition function is pure; effects run outside it and report back as events.
- [Synctest](../stdlib/synctest.md): a bubble removes time and scheduling as sources of flaky test results.

## Violated by

- [Non-Deterministic State Machine](../anti-patterns/non-deterministic-state-machine.md): the same state and the same events give the same state, only if every choice is an input.

## Related concepts

- [Ordering](ordering.md): order is part of the input; a replay must apply inputs in the recorded order.
- [Memory Visibility](memory-visibility.md): a data race is a hidden input that makes even a single run undefined.
- [Timers](timers.md): time is a source of nondeterminism; a timer firing becomes an event with the time in it.
- [Cooperative Cancellation](cooperative-cancellation.md): a cancellation arrives at an arbitrary point; treated as
  an event, it is recorded like any other input.

## Sources

- [Schneider (1990)](../sources.md#source-schneider-1990)
- [The Go Authors (n.d.g)](../sources.md#source-go-testing-synctest)
