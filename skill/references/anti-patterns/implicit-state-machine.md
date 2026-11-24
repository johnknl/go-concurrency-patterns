# Implicit State Machine

Keep a lifecycle in several boolean flags under a mutex, so combinations that should be impossible are reachable and every method has to re-check them.

- **Also known as:** Boolean flag soup
- **Violates:** [Synchronize explicitly](../principles.md#principle-explicit-synchronization)
- **Purpose:** Lifecycle
- **Mechanisms:** Sync
- **Concerns:** Data race safety, Liveness
- **Code:** [`antipatterns/implicitstate`](https://github.com/johnknl/go-concurrency-patterns/tree/main/antipatterns/implicitstate)

## Symptoms

- A type has fields such as `started`, `stopping` and `closed`, and its methods test combinations of them:
  `s.started && !s.stopping && !s.closed`.
- A value ends up both started and closed, or a `Stop` during `Start` is lost, and the next call returns an error
  that no state should produce.
- Nobody can list the states and the transitions between them without reading every method.

## Example

**Faulty**

`badServer.Start` checks the flags, releases the lock for a slow setup, and then sets `started`. A `Stop` that
runs during the setup sets `closed`, and `Start` sets `started` afterwards: a server that is started and closed
at once, which no method expects. The example calls `Stop` from the setup to force that order.

```go
type badServer struct {
	mu       sync.Mutex
	started  bool
	stopping bool
	closed   bool
}

func (s *badServer) Start(init func()) error {
	s.mu.Lock()
	if s.started || s.closed {
		s.mu.Unlock()
		return errors.New("already started or closed")
	}
	s.mu.Unlock()

	init() // slow setup, without the lock

	s.mu.Lock()
	s.started = true
	s.mu.Unlock()
	return nil
}

func (s *badServer) Stop() {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed {
		return
	}
	s.stopping = true
	// ... release resources ...
	s.stopping = false
	s.closed = true
}

func main() {
	s := &badServer{}
	// Stop arrives while Start runs its setup outside the lock.
	err := s.Start(func() { s.Stop() })
	fmt.Println("start error:", err)
	fmt.Println("started:", s.started, "closed:", s.closed)
	fmt.Println("running:", s.Running())
	fmt.Println("start again:", s.Start(func() {}))
	// Output:
	// start error: <nil>
	// started: true closed: true
	// running: false
	// start again: already started or closed
}
```

**Fixed**

`server` has one field, `state`, and each of its four values is a legal state. `Start` moves to `starting`
before it releases the lock, so a `Stop` during the setup is a defined transition, `starting` to `stopped`, and
`Start` sees it when it takes the lock again.

```go
type lifecycle int

type server struct {
	mu    sync.Mutex
	state lifecycle
}

func (s *server) Start(init func()) error {
	s.mu.Lock()
	if s.state != idle {
		s.mu.Unlock()
		return errNotIdle
	}
	s.state = starting
	s.mu.Unlock()

	init()

	s.mu.Lock()
	defer s.mu.Unlock()
	if s.state != starting { // Stop came first while init ran
		return errStoppedDuringStart
	}
	s.state = running
	return nil
}

func (s *server) Stop() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.state = stopped
}

func main() {
	s := &server{}
	err := s.Start(func() {
		fmt.Println("during start:", s.State())
		s.Stop()
	})
	fmt.Println("start error:", err)
	fmt.Println("state:", s.State())
	fmt.Println("start again:", s.Start(func() {}))
	// Output:
	// during start: starting
	// start error: stopped during start
	// state: stopped
	// start again: not idle
}
```

## Why it goes wrong

Three flags have eight combinations; the lifecycle has four states. The other four are not excluded by anything
but the order in which the methods happen to run, and that order is not synchronized: `Start` assumes that nothing
changes between its check and its write, while it released the lock in between. The intermediate state, "setting
up", has no representation, so `Stop` cannot see it and does the wrong thing. Each new flag doubles the
combinations, and each method grows another check.

## Detection

- Look for two or more booleans that describe the same lifecycle, and for conditions that combine them.
- Look for a lock released and taken again inside one operation, with a flag written after the second `Lock`
  that was tested before the first `Unlock`.
- The race detector does not report it: every access is under the lock.

## Fixed by

- [State Machine](../patterns/state-machine.md): Change a stateful object only in steps that are legal for its current state, so that the same events in the same order always lead to the same state.

Name the states as one enumerated field, give the in-between steps (`starting`, `stopping`) their own values, and
write down which event moves which state where. With an owner goroutine, as in
[State Machine](../patterns/state-machine.md), the transitions run one at a time and the lock goes away.

## Sources

Original work.
