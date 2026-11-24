# State Machine

Change a stateful object only in steps that are legal for its current state, so that the same events in the same order always lead to the same state.

- **Also known as:** Deterministic State Machine, Event-driven state machine
- **Purpose:** Execution
- **Scale:** Pattern
- **Mechanisms:** Channels, Select, Context
- **Concerns:** Determinism, Data race safety, Ordering
- **Concepts:** [Determinism](../concepts/determinism.md), [Confinement](../concepts/confinement.md), [Timers and Tickers](../concepts/timers.md), [Goroutine Leak Prevention](../concepts/goroutine-leak-prevention.md)
- **Code:** [`patterns/statemachine`](https://github.com/johnknl/go-concurrency-patterns/tree/main/patterns/statemachine)

## Problem

An object such as a job, a connection, or an order goes through stages, and what may happen next depends on the
stage it is in: a job that has finished cannot be started again, and a cancel that arrives after the result changes
nothing. Events arrive from several goroutines, from work in the background, and from timers. When the rules are
spread over methods that each check a few flags, some combinations of flags are states nobody meant to allow, and
the result of a sequence of events depends on more than the events. Such an object cannot be tested exhaustively,
and a failure cannot be reproduced from the events that caused it.

## Mechanism

Two parts, kept apart:

- **The transition** is one function, `next(state, event)`, that returns the new state and the effects to run, as
  values. It reads nothing but its arguments: no clock, no random numbers, no I/O
  ([a pure transition](../concepts/determinism.md#pure-transition)). An event that is not legal in the current
  state returns `errIllegal` and leaves the state unchanged.
- **The owner** is a goroutine that holds the state as a local variable
  ([confinement by an owner goroutine](../concepts/confinement.md#owner-goroutine)), receives events on a channel,
  calls `next`, records the event, and runs the effects. An effect that produces a result, such as the work
  finishing, sends that result back to the owner as a new event. The owner ignores illegal events and does not
  record them; that is this example's policy, and a reply with the error is the other common choice.

The example is a job:

```go
func next(s state, e event) (state, []effect, error) {
	switch {
	case s == pending && e.kind == start:
		return running, []effect{startWork}, nil
	case s == pending && e.kind == cancel:
		return canceled, nil, nil
	case s == running && e.kind == succeeded:
		return done, nil, nil
	case s == running && e.kind == workFailed:
		return failed, nil, nil
	case s == running && e.kind == cancel:
		return canceled, []effect{stopWork}, nil
	}

	return s, nil, fmt.Errorf("%w: %v in %v", errIllegal, e.kind, s)
}

func (j *job) run(ctx context.Context) {
	workCtx, stop := context.WithCancel(ctx)

	var helpers sync.WaitGroup

	defer func() {
		close(j.stopped)
		stop()
		helpers.Wait()
		close(j.done)
	}()

	s := pending

	for {
		select {
		case <-ctx.Done():
			j.final = s
			return
		case e := <-j.events:
			ns, effects, err := j.transition(s, e)
			if err != nil {
				continue // illegal events are ignored and not recorded
			}

			j.history = append(j.history, e)
			s = ns

			for _, f := range effects {
				j.execute(workCtx, stop, &helpers, f)
			}

			if s.final() {
				j.final = s
				return
			}
		}
	}
}

func (j *job) execute(ctx context.Context, stop context.CancelFunc, helpers *sync.WaitGroup, f effect) {
	switch f {
	case startWork:
		helpers.Go(func() {
			if err := j.work(ctx); err != nil {
				j.send(event{kind: workFailed, reason: err.Error()})
				return
			}

			j.send(event{kind: succeeded})
		})
	case stopWork:
		stop()
	case armTimer:
		helpers.Go(func() {
			t := time.NewTimer(j.timeout)
			defer t.Stop()

			select {
			case <-t.C:
				j.send(event{kind: timeout})
			case <-ctx.Done():
			}
		})
	}
}

func (j *job) send(e event) bool {
	select {
	case j.events <- e:
		return true
	case <-j.stopped:
		return false
	}
}

func main() {
	release := make(chan struct{})
	work := func(ctx context.Context) error {
		select {
		case <-release:
			return nil
		case <-ctx.Done():
			return ctx.Err()
		}
	}

	j := startJob(context.Background(), next, work, 0)
	j.send(event{kind: start})
	j.send(event{kind: start})
	close(release)
	<-j.done

	fmt.Println("final:", j.final)
	fmt.Println("history:", j.history)

	// Output:
	// final: done
	// history: [start succeeded]
}
```

What makes it correct:

- **Every pair is decided.** `next` is a table over state × event; every pair not listed is illegal. The tests check
  all of them, which is possible only because the transition is a function of two small values.
- **Same inputs, same state.** The owner applies events in one order, the order it receives them, and `next`
  depends on nothing else ([same inputs](../concepts/determinism.md#same-inputs)), so folding the recorded history
  gives the state the owner reached. Everything a transition needs is in the event: the failure reason is a field
  of the `failed` event, not something read later ([materialize the choice](../concepts/determinism.md#materialize)).
  Which of two concurrent senders wins is still decided by the scheduler; the history records the outcome.
- **Effects outside.** Starting work, stopping it, and arming a timer happen in `execute`, never in `next`. Their
  results come back on the same channel as any other event, so they are ordered with the rest.
- **A clean end.** When the state is final or the context is canceled, the owner stops receiving, cancels the work,
  waits for every goroutine it started, and only then closes `done`. A send after that returns `false` instead of
  blocking ([the end is observable](../concepts/goroutine-leak-prevention.md#observable-end)). Cancellation of the
  context stops the owner in whatever state it is in, and is not an event of the machine.

**Replay.** Rebuilding the state is a fold of the history through the same `next`, with no effects run:

```go
func replay(transition func(state, event) (state, []effect, error), events []event) state {
	s := pending
	for _, e := range events {
		s, _, _ = transition(s, e)
	}

	return s
}

func main() {
	history := []event{{kind: start}, {kind: workFailed, reason: "disk full"}}

	fmt.Println(replay(next, history))
	fmt.Println(replay(next, history[:1]))

	// Output:
	// failed
	// running
}
```

## Composition

- [For-Select Loop](for-select-loop.md): the owner waits for the next event or for cancellation.
- [Active Object](active-object.md): events are the commands, and the owner is the only goroutine that holds the state.

## Applicability

Use it when an object has a lifecycle with rules about what may happen in which stage, when events come from
several sources (callers, background work, timers), or when the behavior must be tested exhaustively and failures
reproduced from their events. For a flag that is set once, a [`sync.Once`](https://pkg.go.dev/sync#Once) or a
closed channel is enough; for two or three states with no effects, a `switch` on a field under a
[mutex](monitor.md) is simpler. Determinism is worth this structure only where something replays or tests the events
([where it is needed](../concepts/determinism.md#where-needed)).

## Consequences

- The rules are in one place, and a test can cover every state × event pair.
- The state can be rebuilt from its events, and two copies that apply the same events agree.
- Effects need a round trip: the result of work is a new event, not a return value, so code that would be a
  straight sequence of calls becomes several transitions.
- Anything the transition reads besides its arguments, such as `time.Now`, a random number, or a map iterated in
  its own order, breaks replay without any test failing. Inject such sources into the owner and pass their values
  in events ([inject](../concepts/determinism.md#inject)).
- The owner has the costs of any owner goroutine: one channel round trip per event and the throughput of one
  goroutine ([owner costs](../concepts/confinement.md#owner-costs)).
- Replay repeats transitions, not effects. Effects that must not run twice need their own protection.

## Variants

### Timeout as an event

A deadline on `running` is a timer whose firing is one more event. `nextTimed` adds an `armTimer` effect to
`start` and a `running` + `timeout` transition; the owner arms the timer, and its firing comes back on the event
channel like any other result. The transition never reads the clock, so it stays pure, and the history records
whether the timeout or the result came first. In tests the timer runs on fake time with
[testing/synctest](../concepts/timers.md#synctest).

```go
func nextTimed(s state, e event) (state, []effect, error) {
	switch {
	case s == pending && e.kind == start:
		return running, []effect{startWork, armTimer}, nil
	case s == running && e.kind == timeout:
		return failed, []effect{stopWork}, nil
	}

	return next(s, e)
}

func main() {
	work := func(ctx context.Context) error {
		<-ctx.Done()
		return ctx.Err()
	}

	j := startJob(context.Background(), nextTimed, work, 10*time.Millisecond)
	j.send(event{kind: start})
	<-j.done

	fmt.Println("final:", j.final)
	fmt.Println("history:", j.history)

	// Output:
	// final: failed
	// history: [start timeout]
}
```

### Transition under a lock

Without background effects, `next` can be called by each caller under a mutex instead of by an owner goroutine. The
transition stays the same pure function; only who applies it changes, and the order is the order in which callers
take the lock.

## Related patterns

- [Active Object](active-object.md) is the owner without the rule about legal transitions; a state machine adds a
  pure `next` and makes the order of events the whole input.

## Anti-patterns

- [Implicit State Machine](../anti-patterns/implicit-state-machine.md): Keep a lifecycle in several boolean flags under a mutex, so combinations that should be impossible are reachable and every method has to re-check them.
- [Non-Deterministic State Machine](../anti-patterns/non-deterministic-state-machine.md): A transition whose outcome is not part of its input, so the same state and event can lead to different states.

## Sources

[Schneider (1990)](../sources.md#source-schneider-1990) describes a state machine as state variables changed by
commands, whose outputs are determined only by the sequence of requests it processes, and builds replication on
it: replicas that start in the same state and process the same requests in the same order produce the same outputs.

- [Schneider (1990)](../sources.md#source-schneider-1990)
