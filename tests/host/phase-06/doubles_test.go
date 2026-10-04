package phase06

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/ryantr-statinops/InfoBoard/host/internal/domain"
	"github.com/ryantr-statinops/InfoBoard/host/internal/runtime"
)

// stateRecorder observes every published lifecycle transition. Transitions and
// counter refreshes both notify observers, so consecutive duplicates collapse.
type stateRecorder struct {
	mu        sync.Mutex
	states    []runtime.LifecycleState
	avail     []runtime.Availability
	snapshots []runtime.HealthSnapshot
	notify    chan runtime.HealthSnapshot
}

func newStateRecorder() *stateRecorder {
	return &stateRecorder{notify: make(chan runtime.HealthSnapshot, 256)}
}

func (rec *stateRecorder) Observe(snapshot runtime.HealthSnapshot) {
	rec.mu.Lock()
	if len(rec.states) == 0 || rec.states[len(rec.states)-1] != snapshot.State {
		rec.states = append(rec.states, snapshot.State)
	}
	if len(rec.avail) == 0 || rec.avail[len(rec.avail)-1] != snapshot.Availability {
		rec.avail = append(rec.avail, snapshot.Availability)
	}
	rec.snapshots = append(rec.snapshots, snapshot)
	rec.mu.Unlock()
	select {
	case rec.notify <- snapshot:
	default:
	}
}

// Path is the ordered, de-duplicated lifecycle substate path.
func (rec *stateRecorder) Path() []runtime.LifecycleState {
	rec.mu.Lock()
	defer rec.mu.Unlock()
	return append([]runtime.LifecycleState(nil), rec.states...)
}

// Availabilities is the ordered, de-duplicated public availability path.
func (rec *stateRecorder) Availabilities() []runtime.Availability {
	rec.mu.Lock()
	defer rec.mu.Unlock()
	return append([]runtime.Availability(nil), rec.avail...)
}

// WithState reports whether the published path contains state.
func (rec *stateRecorder) WithState(state runtime.LifecycleState) bool {
	for _, seen := range rec.Path() {
		if seen == state {
			return true
		}
	}
	return false
}

// AwaitState blocks until state is published or the budget expires.
func (rec *stateRecorder) AwaitState(t *testing.T, state runtime.LifecycleState, budget time.Duration) {
	t.Helper()
	deadline := time.After(budget)
	for {
		if rec.WithState(state) {
			return
		}
		select {
		case snapshot := <-rec.notify:
			if snapshot.State == state {
				return
			}
		case <-deadline:
			t.Fatalf("state %s was never published; observed path %v with last class %s", state, rec.Path(), rec.lastFailure())
		}
	}
}

// AwaitAvailability blocks until the public availability is published.
func (rec *stateRecorder) AwaitAvailability(t *testing.T, availability runtime.Availability, budget time.Duration) {
	t.Helper()
	deadline := time.After(budget)
	for {
		if rec.WithAvailability(availability) {
			return
		}
		select {
		case snapshot := <-rec.notify:
			if snapshot.Availability == availability {
				return
			}
		case <-deadline:
			t.Fatalf("availability %s was never published; observed %v with last class %s", availability, rec.Availabilities(), rec.lastFailure())
		}
	}
}

// AwaitFailure blocks until the runtime publishes the given failure class.
func (rec *stateRecorder) AwaitFailure(t *testing.T, failure runtime.FailureClass, budget time.Duration) {
	t.Helper()
	deadline := time.After(budget)
	for {
		rec.mu.Lock()
		seen := false
		for _, snapshot := range rec.snapshots {
			if snapshot.Failure == failure {
				seen = true
				break
			}
		}
		rec.mu.Unlock()
		if seen {
			return
		}
		select {
		case <-rec.notify:
		case <-deadline:
			t.Fatalf("failure class %s was never published; path %v", failure, rec.Path())
		}
	}
}

// WithAvailability reports whether the published path contained availability.
func (rec *stateRecorder) WithAvailability(availability runtime.Availability) bool {
	for _, seen := range rec.Availabilities() {
		if seen == availability {
			return true
		}
	}
	return false
}

func (rec *stateRecorder) lastFailure() runtime.FailureClass {
	rec.mu.Lock()
	defer rec.mu.Unlock()
	if len(rec.snapshots) == 0 {
		return runtime.FailureNone
	}
	return rec.snapshots[len(rec.snapshots)-1].Failure
}

// diagnosticRecorder captures the bounded, redacted diagnostic sink.
type diagnosticRecorder struct {
	mu      sync.Mutex
	entries []runtime.Diagnostic
	limit   int
}

func newDiagnosticRecorder(limit int) *diagnosticRecorder { return &diagnosticRecorder{limit: limit} }

func (rec *diagnosticRecorder) Write(entry runtime.Diagnostic) {
	rec.mu.Lock()
	defer rec.mu.Unlock()
	if len(rec.entries) < rec.limit {
		rec.entries = append(rec.entries, entry)
	}
}

func (rec *diagnosticRecorder) Entries() []runtime.Diagnostic {
	rec.mu.Lock()
	defer rec.mu.Unlock()
	return append([]runtime.Diagnostic(nil), rec.entries...)
}

// protocolDouble is the deterministic IP-07 protocol owner double.
type protocolDouble struct {
	mu            sync.Mutex
	binding       runtime.Binding
	negotiateErr  error
	negotiateGate *Gate
	acceptErr     error
	acceptGate    *Gate
	accept        func(ctx context.Context, channel runtime.FrameChannel, frame []byte) error
	notifications []runtime.Notification
	negotiates    int
	accepts       int
	shutdowns     int
}

func (p *protocolDouble) Negotiate(ctx context.Context, _ runtime.FrameChannel, _ []byte) (runtime.Binding, error) {
	p.mu.Lock()
	p.negotiates++
	gate, failure, binding := p.negotiateGate, p.negotiateErr, p.binding
	p.mu.Unlock()
	if gate != nil {
		if err := gate.Wait(ctx, ctx.Err()); err != nil {
			return runtime.Binding{}, err
		}
	}
	if failure != nil {
		return runtime.Binding{}, failure
	}
	return binding, nil
}

func (p *protocolDouble) Accept(ctx context.Context, channel runtime.FrameChannel, frame []byte) error {
	p.mu.Lock()
	p.accepts++
	gate, failure, hook := p.acceptGate, p.acceptErr, p.accept
	p.mu.Unlock()
	if hook != nil {
		return hook(ctx, channel, frame)
	}
	if gate != nil {
		if err := gate.Wait(ctx, ctx.Err()); err != nil {
			return err
		}
	}
	return failure
}

func (p *protocolDouble) Notify(_ context.Context, _ runtime.FrameChannel, notification runtime.Notification) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.notifications = append(p.notifications, notification)
	return nil
}

func (p *protocolDouble) Shutdown(context.Context) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.shutdowns++
	return nil
}

func (p *protocolDouble) counts() (negotiates, accepts, shutdowns int, notifications []runtime.Notification) {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.negotiates, p.accepts, p.shutdowns, append([]runtime.Notification(nil), p.notifications...)
}

// projectorDouble is the deterministic IP-05 projection owner double.
type projectorDouble struct {
	mu     sync.Mutex
	gate   *Gate
	status runtime.ProjectionStatus
	err    error
	calls  int
	target []runtime.Target
}

func (p *projectorDouble) Synchronize(ctx context.Context, target runtime.Target) (runtime.ProjectionStatus, error) {
	p.mu.Lock()
	p.calls++
	p.target = append(p.target, target)
	gate, status, failure := p.gate, p.status, p.err
	p.mu.Unlock()
	if gate != nil {
		if err := gate.Wait(ctx, ctx.Err()); err != nil {
			return runtime.ProjectionStatus{}, err
		}
	}
	if failure != nil {
		return runtime.ProjectionStatus{}, failure
	}
	return status, nil
}

func (p *projectorDouble) observedTargets() []runtime.Target {
	p.mu.Lock()
	defer p.mu.Unlock()
	return append([]runtime.Target(nil), p.target...)
}

// storeDouble is the deterministic IP-08 persistence owner double.
type storeDouble struct {
	mu         sync.Mutex
	status     runtime.PersistenceStatus
	prepareErr error
	flushGate  *Gate
	flushErr   error
	prepares   int
	flushes    int
	closes     int
}

func (s *storeDouble) Prepare(ctx context.Context) (runtime.PersistenceStatus, error) {
	s.mu.Lock()
	s.prepares++
	status, failure := s.status, s.prepareErr
	s.mu.Unlock()
	if err := ctx.Err(); err != nil {
		return runtime.PersistenceUnknown, err
	}
	if failure != nil {
		return runtime.PersistenceUnknown, failure
	}
	return status, nil
}

func (s *storeDouble) Flush(ctx context.Context) error {
	s.mu.Lock()
	s.flushes++
	gate, failure := s.flushGate, s.flushErr
	s.mu.Unlock()
	if gate != nil {
		if err := gate.Wait(ctx, errShutdownBudget); err != nil {
			return err
		}
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	return failure
}

func (s *storeDouble) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.closes++
	return nil
}

func (s *storeDouble) counts() (prepares, flushes, closes int) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.prepares, s.flushes, s.closes
}

var errShutdownBudget = errors.New("phase06: shutdown budget")

// rig bundles one complete deterministic session harness.
type rig struct {
	corpus     Corpus
	fixture    HostFixture
	clock      *FakeClock
	limits     runtime.Limits
	input      *FakeReader
	output     *FakeWriter
	states     *stateRecorder
	sink       *diagnosticRecorder
	protocol   *protocolDouble
	projector  *projectorDouble
	store      *storeDouble
	invocation runtime.Invocation
	config     runtime.Config
	origin     string
	allowed    []string
}

// newRig builds a registered-host rig whose budgets, identities, and privacy
// sentinels all come from the shared artifact.
func newRig(t *testing.T, ctx context.Context, fixtureID string) *rig {
	t.Helper()
	corpus := LoadCorpus(t)
	fixture := corpus.Fixture(t, fixtureID)
	limits := corpus.Limits
	clock := NewFakeClock()
	sink := newDiagnosticRecorder(int(limits.MaxDiagnosticRecords))
	recorder := newStateRecorder()
	runtimeLimits := runtime.Limits{
		MaxFrameBytes:          int(limits.MaxFrameBytes),
		MaxQueuedFrames:        int(limits.MaxQueuedFrames),
		MaxConcurrentRequests:  int(limits.MaxConcurrentRequests),
		MaxDiagnostics:         int(limits.MaxDiagnosticRecords),
		MaxDiagnosticBytes:     int(limits.MaxDiagnosticBytes),
		StartupTimeout:         limits.StartupBudget(),
		HandshakeTimeout:       limits.HandshakeBudget(),
		SynchronizationTimeout: limits.SynchronizationBudget(),
		RequestTimeout:         limits.RequestBudget(),
		ShutdownTimeout:        limits.ShutdownBudget(),
	}
	rig := &rig{
		corpus:    corpus,
		fixture:   fixture,
		clock:     clock,
		limits:    runtimeLimits,
		input:     NewFakeReader(ctx),
		output:    NewFakeWriter(ctx),
		states:    recorder,
		sink:      sink,
		protocol:  &protocolDouble{binding: readyBinding(corpus)},
		projector: &projectorDouble{status: readyProjection(corpus)},
		store:     &storeDouble{status: runtime.PersistenceReady},
		origin:    corpus.Identity.Origin,
		allowed:   append([]string(nil), corpus.Identity.AllowedOrigins...),
	}
	rig.config = runtime.Config{
		HostVersion:     corpus.Identity.HostVersion,
		ProtocolVersion: corpus.Identity.ProtocolVersion,
		AllowedOrigins:  rig.allowed,
		ExpectedProfile: domain.ProfileID(corpus.Identity.ProfileID),
		Limits:          runtimeLimits,
		Clock:           clock,
		Diagnostics:     runtime.NewDiagnostics(sink, int(limits.MaxDiagnosticRecords), int(limits.MaxDiagnosticBytes)),
	}
	return rig
}

func readyBinding(corpus Corpus) runtime.Binding {
	return runtime.Binding{
		ProfileID:       domain.ProfileID(corpus.Identity.ProfileID),
		ContextKind:     domain.ContextNormal,
		ProtocolVersion: corpus.Identity.ProtocolVersion,
		HostVersion:     corpus.Identity.HostVersion,
		SnapshotApplied: true,
	}
}

func readyProjection(corpus Corpus) runtime.ProjectionStatus {
	return runtime.ProjectionStatus{
		Known:    true,
		Epoch:    corpus.Identity.ProjectionEpoch,
		Revision: 1,
		Count:    3,
		Ready:    true,
	}
}

// newOwnedRig registers teardown so every fake transport is released even when a
// test fails part way through.
func newOwnedRig(t *testing.T, ctx context.Context, fixtureID string) *rig {
	t.Helper()
	rig := newRig(t, ctx, fixtureID)
	t.Cleanup(rig.Release)
	return rig
}

// Bootstrap validates the fixture invocation and returns the session or the
// classified bootstrap failure.
func (rig *rig) Bootstrap(t *testing.T) (*runtime.Session, error) {
	t.Helper()
	invocation, err := runtime.ParseInvocation([]string{rig.origin})
	if err != nil {
		t.Fatalf("parse fixture invocation: %v", err)
	}
	rig.invocation = invocation
	session, err := runtime.Bootstrap(rig.config, invocation, rig.input, rig.output, runtime.Dependencies{
		Protocol:    rig.protocol,
		Projection:  rig.projector,
		Persistence: rig.store,
	})
	if session != nil {
		session.Subscribe(rig.states)
	}
	return session, err
}

// Release ends the fake stdin stream and releases any parked read, so the single
// reader goroutine always leaves the test instead of leaking.
func (rig *rig) Release() {
	rig.input.Unblock()
	rig.input.EndStream()
	rig.output.Gate().Open()
}

// QueueFrames enqueues whole protocol frames on the browser-provided stdin.
func (rig *rig) QueueFrames(payloads ...string) {
	for _, payload := range payloads {
		rig.input.Queue(EncodeFrame([]byte(payload)))
	}
}

// Start runs the session and returns a channel carrying its single outcome.
func (rig *rig) Start(ctx context.Context, session *runtime.Session) <-chan runtime.Outcome {
	results := make(chan runtime.Outcome, 1)
	go func() { results <- session.Run(ctx) }()
	return results
}

// awaitOutcome waits for the single terminal outcome within the shutdown budget.
func awaitOutcome(t *testing.T, results <-chan runtime.Outcome, budget time.Duration) runtime.Outcome {
	t.Helper()
	select {
	case outcome := <-results:
		return outcome
	case <-time.After(budget):
		t.Fatal("session did not finish within its bounded budget")
		return runtime.Outcome{}
	}
}
