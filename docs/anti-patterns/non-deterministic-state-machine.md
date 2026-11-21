---
title: Non-Deterministic State Machine
aliases: [Impure transition function]
intent: A transition whose outcome is not part of its input, so the same state and event can lead to different states.
purpose: execution
package: antipatterns/nondetstate
mechanisms: [channels]
concerns: [determinism]
remedies: [state-machine]
violates: [explicit-inputs]
concepts:
  - determinism: 'the same state and the same events give the same state, only if every choice is an input.'
sources:
  - id: schneider-1990
---
# Non-Deterministic State Machine

<!-- meta -->
A transition whose outcome is not part of its input, so the same state and event can lead to different states.

<div class="cp-props" markdown>

| | |
|---|---|
| Also known as | Impure transition function |
| Violates | [Prefer explicit inputs](../index.md#principle-explicit-inputs) |
| Purpose | [Execution](../catalog/purpose.md#purpose-execution) |
| Mechanisms | [Channels](../catalog/mechanism.md#mechanism-channels) |
| Concerns | [Determinism](../catalog/concern.md#concern-determinism) |
| Breaks | [Determinism](../concepts/determinism.md) |
| Code | [`antipatterns/nondetstate`](https://github.com/johnknl/go-concurrency-patterns/tree/main/antipatterns/nondetstate) |

</div>
<!-- /meta -->

## Symptoms

- Replaying the events that led to a failure does not lead to it again.
- A test of a transition passes or fails from run to run, or only with a fixed seed.
- The transition function reads the clock, a random number, a map in iteration order, or calls another system.

## Example

=== "Faulty"

    `badNext` decides the outcome of an attempt itself, by calling `attemptSucceeds`; the event says only that an
    attempt happened. The example replaces the source so that the live run succeeds and the replay fails: the same
    log replays to a different state.

    <!-- code: antipatterns/nondetstate#attemptSucceeds antipatterns/nondetstate#badNext antipatterns/nondetstate#Example_replayDiverges:main -->

    ```go
    var attemptSucceeds = func() bool { return rand.IntN(2) == 0 }

    func badNext(s state, e badEvent) state {
    	switch {
    	case s == pending && e == start:
    		return running
    	case s == running && e == attemptKind:
    		if attemptSucceeds() {
    			return done
    		}
    		return failed
    	}
    	return s
    }

    func main() {
    	defer func(f func() bool) { attemptSucceeds = f }(attemptSucceeds)
    	log := []badEvent{"start", "attempt"}

    	attemptSucceeds = func() bool { return true } // the live run
    	fmt.Println("live:  ", badReplay(log))

    	attemptSucceeds = func() bool { return false } // the replay draws again
    	fmt.Println("replay:", badReplay(log))
    	// Output:
    	// live:   done
    	// replay: failed
    }
    ```

    <!-- /code -->

=== "Fixed"

    The outcome is a field of the event. `attempt` draws it once, from a source the caller injects, and the event
    that carries it is what gets recorded. `next` reads only its state and its event, so every replay of the log
    gives the state the live run reached.

    <!-- code: antipatterns/nondetstate#event antipatterns/nondetstate#next antipatterns/nondetstate#attempt antipatterns/nondetstate#Example_replayIdentical:main -->

    ```go
    type event struct {
    	kind string
    	ok   bool
    }

    func next(s state, e event) state {
    	switch {
    	case s == pending && e.kind == start:
    		return running
    	case s == running && e.kind == attemptKind:
    		if e.ok {
    			return done
    		}
    		return failed
    	}
    	return s
    }

    func attempt(succeeds func() bool) event {
    	return event{kind: attemptKind, ok: succeeds()}
    }

    func main() {
    	// The live run decides the outcome once and records it in the event.
    	log := []event{{kind: "start"}, attempt(func() bool { return true })}
    	fmt.Println("live:  ", replay(log))
    	fmt.Println("replay:", replay(log))
    	// Output:
    	// live:   done
    	// replay: done
    }
    ```

    <!-- /code -->

## Why it goes wrong

A state machine can be tested on its own, and a failure reproduced from its events, only if
[the same state and the same inputs in the same order give the same result](../concepts/determinism.md#same-inputs).
A choice made inside the transition is an input that nobody recorded, so the replay makes the choice again and may
make it differently.

A [modelled choice](../concepts/determinism.md#modelled) is not the problem: a job that goes from `running` to `done` or to `failed` is a legitimate state
machine. What breaks replay is where the choice is made. The fix
[makes the choice an input](../concepts/determinism.md#materialize): a field of the event, decided by the owner
with an [injected source](../concepts/determinism.md#inject) before the transition runs, and recorded with the
event. The randomness stays; it moves out of the transition and into the recorded events. Where nothing is ever replayed or
tested against recorded events, this costs more than it gains; see [where determinism is needed](../concepts/determinism.md#where-needed).

## Detection

- Read every transition for calls to `time.Now`, `math/rand`, `os.Getenv`, the network, or the disk, and for
  `range` over a map or a `select` with several ready cases; see the
  [sources of nondeterminism in Go](../concepts/determinism.md#sources).
- Replay recorded events twice, or with a different random source, and compare the states.
- Run the tests with `-count` and `-shuffle=on`; a test of a transition that needs a seed to pass is a sign.

## Fixed by

<!-- remedies -->
- [State Machine](../patterns/state-machine.md): Change a stateful object only in steps that are legal for its current state, so that the same events in the same order always lead to the same state.
<!-- /remedies -->

A [pure transition function](../concepts/determinism.md#pure-transition) returns the new state and the effects
to run; the effects run outside it, and their results come back as events, which are recorded like any other.

## Sources

<!-- sources -->
- [Schneider (1990)](../sources.md#source-schneider-1990)
<!-- /sources -->
