package runtime

import (
	"context"
	"time"
)

// LifecycleState is the observable session substate of the host. The public
// Availability is derived from it and never replaces it.
type LifecycleState string

const (
	StateStarting      LifecycleState = "starting"
	StateHandshaking   LifecycleState = "handshaking"
	StateSynchronizing LifecycleState = "synchronizing"
	StateReady         LifecycleState = "ready"
	StateClosing       LifecycleState = "closing"
	StateUnavailable   LifecycleState = "unavailable"
	StateRecovering    LifecycleState = "recovering"
)

const (
	ExitClean           = 0
	ExitStartupFailure  = 2
	ExitProtocolFailure = 3
	ExitRuntimeFailure  = 4
)

func (state LifecycleState) String() string { return string(state) }

// Terminal reports whether the session is finished and can no longer accept
// work or report itself healthy.
func (state LifecycleState) Terminal() bool { return state == StateClosing }

var lifecycleTransitions = map[LifecycleState]map[LifecycleState]struct{}{
	StateStarting:      {StateHandshaking: {}, StateUnavailable: {}, StateRecovering: {}, StateClosing: {}},
	StateHandshaking:   {StateSynchronizing: {}, StateUnavailable: {}, StateRecovering: {}, StateClosing: {}},
	StateSynchronizing: {StateReady: {}, StateUnavailable: {}, StateRecovering: {}, StateClosing: {}},
	StateReady:         {StateSynchronizing: {}, StateUnavailable: {}, StateRecovering: {}, StateClosing: {}},
	StateRecovering:    {StateSynchronizing: {}, StateUnavailable: {}, StateClosing: {}},
	StateUnavailable:   {StateRecovering: {}, StateClosing: {}},
	StateClosing:       {},
}

// transitionAllowed implements the handoff contract:
//
//	Starting -> Handshaking -> Synchronizing -> Ready
//	Ready    -> Synchronizing when an authoritative resync is required
//	any live -> Recovering when a new connection is pending
//	any      -> Unavailable for a fail-closed incompatibility
//	any      -> Closing as the idempotent terminal state
func transitionAllowed(from, to LifecycleState) bool {
	if from == to {
		return true
	}
	allowed, known := lifecycleTransitions[from]
	if !known {
		return false
	}
	_, permitted := allowed[to]
	return permitted
}

// availabilityOf maps a lifecycle substate onto the public availability
// contract. Only Ready is healthy; closing and unavailable are unavailable;
// every other live substate is still recovering.
func availabilityOf(state LifecycleState) Availability {
	switch state {
	case StateReady:
		return AvailabilityHealthy
	case StateClosing, StateUnavailable:
		return AvailabilityUnavailable
	default:
		return AvailabilityRecovering
	}
}

// Outcome is the deterministic process or session result the entrypoint turns
// into an exit status.
type Outcome struct {
	Clean     bool
	Failure   FailureClass
	Retryable bool
	ExitCode  int
}

// exitCodeFor distinguishes a clean close from startup, protocol, and runtime
// failure without exposing any input-derived text.
func exitCodeFor(failure FailureClass, readyReached bool) int {
	switch failure {
	case FailureNone:
		return ExitClean
	case FailureProtocolIncompatible, FailureProfileMismatch:
		return ExitProtocolFailure
	}
	if !readyReached {
		return ExitStartupFailure
	}
	return ExitRuntimeFailure
}

func newOutcome(failure FailureClass, readyReached bool) Outcome {
	return Outcome{
		Clean:     failure == FailureNone,
		Failure:   failure,
		Retryable: failure != FailureNone && failure.Retryable(),
		ExitCode:  exitCodeFor(failure, readyReached),
	}
}

// publish moves the session to a legal state and notifies observers. An illegal
// transition never invents a new state; it only records the failure class.
func (s *Session) publish(state LifecycleState, failure FailureClass, retryable bool) {
	if !transitionAllowed(s.health.current().State, state) {
		s.health.mutate(func(snapshot *HealthSnapshot) {
			snapshot.Failure = failure
			snapshot.Retryable = retryable
		})
		return
	}
	s.health.transition(state, failure, retryable)
	published := s.health.current()
	s.diagnostics.Observe(published, levelForState(failure), EventStateChanged, 0, int64(published.Transitions))
}

func levelForState(failure FailureClass) DiagnosticLevel {
	if failure == FailureNone {
		return LevelInfo
	}
	if failure.Retryable() {
		return LevelWarn
	}
	return LevelError
}

func levelForError(err error) DiagnosticLevel {
	if failure, _ := ClassifyFailure(err); failure.Retryable() {
		return LevelWarn
	}
	return LevelError
}

// closeCause maps a session outcome onto the cancellation cause propagated to
// every in-flight request context.
func closeCause(failure FailureClass) error {
	if failure == FailureNone {
		return ErrSessionClosing
	}
	return fail(failure, ErrSessionClosing)
}

// finish runs the single terminal sequence: publish Closing, cancel in-flight
// work, drain within the shutdown budget, flush bounded persistence, emit a
// final safe status while the channel is still writable, then close the stream
// and dependencies exactly once. It is safe to call repeatedly.
func (s *Session) finish(outcome Outcome) Outcome {
	executed := false
	s.finishOnce.Do(func() {
		executed = true
		s.finishOutcome = outcome
		start := s.clock.Now()
		s.shutdownAt = s.clock.Now().Add(s.limits.ShutdownTimeout)
		s.diagnostics.Observe(s.health.current(), LevelInfo, EventShutdownStarted, 0, 0)
		s.publish(StateClosing, outcome.Failure, outcome.Retryable)
		cancelled := s.requests.cancelAll(closeCause(outcome.Failure))
		s.reader.Interrupt()
		drained := s.drain(s.remainingBudget())
		s.flushPersistence()
		s.notifyShutdown(outcome)
		s.shutdownProtocol()
		_ = s.writer.Close(s.remainingBudget())
		s.closeStore()
		s.diagnostics.Observe(s.health.current(), levelForOutcome(outcome), EventSessionClosed, s.clock.Now().Sub(start).Milliseconds(), int64(cancelled))
		s.health.count(func(snapshot *HealthSnapshot) {
			snapshot.FramesRead = s.reader.count()
			snapshot.FramesWritten = s.writer.count()
			started, completed, rejected := s.requests.counters()
			snapshot.RequestsStarted, snapshot.RequestsCompleted, snapshot.RequestsRejected = started, completed, rejected
			snapshot.DroppedDiagnostics = s.diagnostics.Dropped()
		})
		_ = drained
		close(s.finished)
	})
	if !executed {
		<-s.finished
	}
	return s.finishOutcome
}

func levelForOutcome(outcome Outcome) DiagnosticLevel {
	if outcome.Clean {
		return LevelInfo
	}
	if outcome.Retryable {
		return LevelWarn
	}
	return LevelError
}

// drain waits only up to budget for in-flight handlers. Native Messaging pipes
// have no portable write deadline, so the session terminates at the budget
// rather than waiting indefinitely on a blocked write.
func (s *Session) drain(budget time.Duration) bool {
	if budget <= 0 {
		return false
	}
	ticks, stop := s.clock.NewTimer(budget)
	defer stop()
	done := make(chan struct{})
	go func() {
		s.inflight.Wait()
		close(done)
	}()
	select {
	case <-done:
		return true
	case <-ticks:
		return false
	case <-s.root().Done():
		return false
	}
}

func (s *Session) flushPersistence() {
	store := s.persistence()
	if store == nil {
		return
	}
	budget := s.remainingBudget()
	ctx, cancel := withBudget(context.Background(), budget, ErrShutdownBudget)
	defer cancel()
	if _, err := boundedCall(ctx, func() (struct{}, error) { return struct{}{}, store.Flush(ctx) }); err != nil {
		s.health.mutate(func(snapshot *HealthSnapshot) { snapshot.Persistence = PersistenceDegraded })
		s.diagnostics.Observe(s.health.current(), LevelWarn, EventPersistenceFlushed, budget.Milliseconds(), 1)
		return
	}
	s.diagnostics.Observe(s.health.current(), LevelInfo, EventPersistenceFlushed, budget.Milliseconds(), 0)
}

func (s *Session) notifyShutdown(outcome Outcome) {
	if s.writer.isClosed() {
		return
	}
	ctx, cancel := withBudget(context.Background(), s.remainingBudget(), ErrShutdownBudget)
	defer cancel()
	notification := Notification{
		Reason:       NotificationShutdown,
		State:        StateClosing,
		Availability: AvailabilityUnavailable,
		Failure:      outcome.Failure,
		Retryable:    outcome.Retryable,
		Snapshot:     s.health.current(),
	}
	if err := s.protocol.Notify(ctx, s, notification); err != nil {
		// A final status frame is best effort: the extension treats process
		// absence or EOF using the last published safe state.
		s.diagnostics.Observe(s.health.current(), LevelWarn, EventFrameWriteFailed, 0, 0)
	}
}

func (s *Session) shutdownProtocol() {
	ctx, cancel := withBudget(context.Background(), s.remainingBudget(), ErrShutdownBudget)
	defer cancel()
	if _, err := boundedCall(ctx, func() (struct{}, error) { return struct{}{}, s.protocol.Shutdown(ctx) }); err != nil {
		s.diagnostics.Observe(s.health.current(), LevelWarn, EventShutdownDrained, 0, 0)
	}
}

func (s *Session) closeStore() {
	if store := s.persistence(); store != nil {
		_ = store.Close()
	}
}

// Close ends the session idempotently. A finished session never accepts new
// work and never reports itself healthy.
func (s *Session) Close(reason error) Outcome {
	failure := FailureShutdown
	if reason == nil {
		failure = FailureNone
	} else if classified, _ := ClassifyFailure(reason); classified != FailureNone {
		failure = classified
	}
	return s.finish(newOutcome(failure, s.reachedReady()))
}

func (s *Session) reachedReady() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.readyReached
}
