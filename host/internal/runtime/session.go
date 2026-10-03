package runtime

import (
	"context"
	"encoding/binary"
	"errors"
	"io"
	"sync"
	"time"

	"github.com/ryantr-statinops/InfoBoard/host/internal/domain"
)

// Unblocker is implemented by an input stream that can wake a blocked read
// during bounded shutdown. os.File satisfies the seam through io.Closer; an
// injected fake implements Unblock directly. Exactly one reader owns the
// stream and only the session terminal path may call it.
type Unblocker interface {
	Unblock()
}

// FrameReader is the runtime half of the framed Native Messaging transport: it
// reads one length-prefixed frame from the browser-provided stdin and enforces
// the frame size limit before allocating the payload buffer. Exactly one reader
// owns the input stream for the lifetime of a session.
type FrameReader struct {
	source io.Reader
	limits Limits
	mu     sync.Mutex
	frames uint64
	once   sync.Once
	closed bool
}

func newFrameReader(source io.Reader, limits Limits) *FrameReader {
	return &FrameReader{source: source, limits: limits}
}

// Interrupt wakes a read that is blocked on the browser pipe so the bounded
// shutdown can finish. It preserves single-reader ownership: only the session
// terminal path calls it, exactly once, and never reads from the stream itself.
func (reader *FrameReader) Interrupt() {
	reader.once.Do(func() {
		reader.mu.Lock()
		reader.closed = true
		reader.mu.Unlock()
		if unblocker, ok := reader.source.(Unblocker); ok {
			unblocker.Unblock()
			return
		}
		if closer, ok := reader.source.(io.Closer); ok {
			_ = closer.Close()
		}
	})
}

// interruptible reports whether a blocked read can be woken. An interruptible
// stream is read directly on the reader goroutine so the hot path allocates no
// helper goroutine per frame.
func (reader *FrameReader) interruptible() bool {
	if _, ok := reader.source.(Unblocker); ok {
		return true
	}
	_, ok := reader.source.(io.Closer)
	return ok
}

// ReadFrame reads one 4-byte little-endian prefixed payload. A truncated frame,
// a zero length, or an oversized length is rejected before allocation so
// partial or malformed input fails closed without unbounded buffering.
func (reader *FrameReader) ReadFrame(ctx context.Context) ([]byte, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if reader.interruptible() {
		return reader.read()
	}
	// A stream that cannot be unblocked is read on a helper goroutine so session
	// cancellation ends the read even when the source ignores its context. The
	// helper owns no session state and its result is dropped when abandoned.
	type outcome struct {
		frame []byte
		err   error
	}
	outcomes := make(chan outcome, 1)
	go func() {
		frame, err := reader.read()
		outcomes <- outcome{frame: frame, err: err}
	}()
	select {
	case completed := <-outcomes:
		return completed.frame, completed.err
	case <-ctx.Done():
		return nil, context.Cause(ctx)
	}
}

func (reader *FrameReader) read() ([]byte, error) {
	var header [4]byte
	if _, err := io.ReadFull(reader.source, header[:]); err != nil {
		if errors.Is(err, io.ErrUnexpectedEOF) {
			return nil, fail(FailureTransportTruncated, ErrFrameTruncated)
		}
		return nil, err
	}
	size := binary.LittleEndian.Uint32(header[:])
	if size == 0 {
		return nil, fail(FailureTransportMalformed, ErrEmptyFrame)
	}
	if uint64(size) > uint64(reader.limits.MaxFrameBytes) {
		return nil, fail(FailureFrameTooLarge, ErrFrameTooLarge)
	}
	payload := make([]byte, size)
	if _, err := io.ReadFull(reader.source, payload); err != nil {
		if errors.Is(err, io.EOF) || errors.Is(err, io.ErrUnexpectedEOF) {
			return nil, fail(FailureTransportTruncated, ErrFrameTruncated)
		}
		return nil, err
	}
	reader.mu.Lock()
	reader.frames++
	reader.mu.Unlock()
	return payload, nil
}

func (reader *FrameReader) count() uint64 {
	reader.mu.Lock()
	defer reader.mu.Unlock()
	return reader.frames
}

// FrameWriter is the only writer that owns the browser-provided stdout. A single
// writer goroutine drains a bounded queue and emits each frame as one whole
// write, so concurrent handlers can never interleave partial frames. Callers
// observe their own deadline instead of blocking forever on a stuck pipe.
type FrameWriter struct {
	sinkRef   io.Writer
	queue     chan writeRequest
	closing   chan struct{}
	done      chan struct{}
	limits    Limits
	closeOnce sync.Once
	mu        sync.Mutex
	frames    uint64
	closed    bool
	err       error
}

type writeRequest struct {
	payload []byte
	result  chan error
}

func newFrameWriter(sink io.Writer, limits Limits) *FrameWriter {
	writer := &FrameWriter{
		sinkRef: sink,
		queue:   make(chan writeRequest, limits.MaxQueuedFrames),
		closing: make(chan struct{}),
		done:    make(chan struct{}),
		limits:  limits,
	}
	go writer.loop(sink)
	return writer
}

func (writer *FrameWriter) loop(sink io.Writer) {
	defer close(writer.done)
	for {
		select {
		case request := <-writer.queue:
			err := writer.emit(sink, request.payload)
			writer.mu.Lock()
			writer.frames++
			writer.mu.Unlock()
			request.result <- err
		case <-writer.closing:
			for {
				select {
				case request := <-writer.queue:
					request.result <- fail(FailureTransportUnavailable, ErrWriterClosed)
				default:
					return
				}
			}
		}
	}
}

func (writer *FrameWriter) emit(sink io.Writer, payload []byte) error {
	frame := make([]byte, 4+len(payload))
	binary.LittleEndian.PutUint32(frame[:4], uint32(len(payload)))
	copy(frame[4:], payload)
	written, err := sink.Write(frame)
	if err != nil {
		return err
	}
	if written != len(frame) {
		return fail(FailureTransportTruncated, ErrFrameTruncated)
	}
	if flusher, ok := sink.(interface{ Flush() error }); ok {
		return flusher.Flush()
	}
	return nil
}

// WriteFrame validates and queues one complete frame, then waits for it under
// the caller's deadline. It returns the caller's cancellation cause when the
// budget expires instead of waiting indefinitely.
func (writer *FrameWriter) WriteFrame(ctx context.Context, payload []byte) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if len(payload) == 0 {
		return fail(FailureTransportMalformed, ErrEmptyFrame)
	}
	if len(payload) > writer.limits.MaxFrameBytes {
		return fail(FailureFrameTooLarge, ErrFrameTooLarge)
	}
	select {
	case <-writer.closing:
		return fail(FailureTransportUnavailable, ErrWriterClosed)
	default:
	}
	request := writeRequest{payload: payload, result: make(chan error, 1)}
	select {
	case writer.queue <- request:
	case <-writer.closing:
		return fail(FailureTransportUnavailable, ErrWriterClosed)
	case <-ctx.Done():
		return context.Cause(ctx)
	}
	select {
	case err := <-request.result:
		return err
	case <-writer.closing:
		select {
		case err := <-request.result:
			return err
		default:
			return fail(FailureTransportUnavailable, ErrWriterClosed)
		}
	case <-ctx.Done():
		return context.Cause(ctx)
	}
}

// Close stops accepting frames and releases the output stream exactly once. It
// waits at most budget for a stuck write, then closes the underlying stream so
// the write fails and the writer goroutine can exit.
func (writer *FrameWriter) Close(budget time.Duration) error {
	executed := false
	writer.closeOnce.Do(func() {
		executed = true
		close(writer.closing)
		if budget <= 0 {
			budget = time.Millisecond
		}
		ticks := time.NewTimer(budget)
		defer ticks.Stop()
		select {
		case <-writer.done:
		case <-ticks.C:
			if closer, ok := writer.sink().(io.Closer); ok {
				writer.err = closer.Close()
			}
		}
		writer.mu.Lock()
		writer.closed = true
		writer.mu.Unlock()
	})
	if !executed {
		<-writer.done
	}
	return writer.err
}

func (writer *FrameWriter) sink() io.Writer { return writer.sinkRef }

func (writer *FrameWriter) isClosed() bool {
	select {
	case <-writer.closing:
		return true
	default:
		return false
	}
}

func (writer *FrameWriter) count() uint64 {
	writer.mu.Lock()
	defer writer.mu.Unlock()
	return writer.frames
}

// Binding is the negotiated session identity returned by the protocol owner.
// The runtime re-validates it against the installed registration assumptions
// instead of trusting the peer to select a profile.
type Binding struct {
	ProfileID       domain.ProfileID
	ContextKind     domain.ContextKind
	ProtocolVersion int
	HostVersion     string
	SnapshotApplied bool
}

// Target identifies the authoritative projection the session must synchronize
// before it can publish Ready.
type Target struct {
	SessionID   string
	ProfileID   domain.ProfileID
	ContextKind domain.ContextKind
}

// ProjectionStatus is the committed projection metadata reported by the
// projection owner. Ready is published only for a known, non-empty lineage.
type ProjectionStatus struct {
	Known    bool
	Epoch    string
	Revision uint64
	Count    uint64
	Ready    bool
}

// ProjectionSync is the IP-05 hook. It acquires an authoritative snapshot for
// the bound profile and context and returns the committed projection metadata.
type ProjectionSync interface {
	Synchronize(ctx context.Context, target Target) (ProjectionStatus, error)
}

// Persistence is the optional IP-08 hook. A nil Persistence runs the session
// with persistence disabled, and a Prepare failure only degrades the reported
// persistence status; it never makes lexical runtime unavailable by itself.
type Persistence interface {
	Prepare(ctx context.Context) (PersistenceStatus, error)
	Flush(ctx context.Context) error
	Close() error
}

// NotificationReason marks why the runtime is pushing a lifecycle notice to the
// protocol owner. The protocol layer owns the resulting wire representation.
type NotificationReason string

const (
	NotificationStartup  NotificationReason = "startup"
	NotificationHealth   NotificationReason = "health"
	NotificationFailure  NotificationReason = "failure"
	NotificationShutdown NotificationReason = "shutdown"
)

// Notification is the runtime-owned lifecycle event handed to the protocol
// owner. It contains bounded state metadata only.
type Notification struct {
	Reason       NotificationReason
	State        LifecycleState
	Availability Availability
	Failure      FailureClass
	Retryable    bool
	Snapshot     HealthSnapshot
}

// FrameChannel is the serialized output seam the protocol owner uses to emit
// complete frames to the peer. It is the only supported path to the browser
// provided stdout stream, so concurrent handlers cannot interleave frames.
type FrameChannel interface {
	WriteFrame(ctx context.Context, payload []byte) error
}

// ProtocolHandler is the IP-07 hook. The runtime owns the process, transport,
// cancellation, and health boundaries; the protocol owner owns the Native
// Messaging envelope, message types, and wire error literals.
type ProtocolHandler interface {
	Negotiate(ctx context.Context, channel FrameChannel, frame []byte) (Binding, error)
	Accept(ctx context.Context, channel FrameChannel, frame []byte) error
	Notify(ctx context.Context, channel FrameChannel, notification Notification) error
	Shutdown(ctx context.Context) error
}

type sealedProtocol struct{}

func (sealedProtocol) Negotiate(context.Context, FrameChannel, []byte) (Binding, error) {
	return Binding{}, fail(FailureHandshakeRejected, ErrProtocolUnavailable)
}

func (sealedProtocol) Accept(context.Context, FrameChannel, []byte) error { return nil }

func (sealedProtocol) Notify(context.Context, FrameChannel, Notification) error { return nil }

func (sealedProtocol) Shutdown(context.Context) error { return nil }

type sealedProjection struct{}

func (sealedProjection) Synchronize(context.Context, Target) (ProjectionStatus, error) {
	return ProjectionStatus{}, fail(FailureSnapshotRejected, ErrProjectionUnavailable)
}

// Dependencies carries the owner hooks. Protocol and Projection fall back to
// fail-closed implementations until IP-05 and IP-07 supply theirs, so the host
// never accepts tab data it cannot validate or index.
type Dependencies struct {
	Protocol    ProtocolHandler
	Projection  ProjectionSync
	Persistence Persistence
}

func (deps Dependencies) resolved() Dependencies {
	if deps.Protocol == nil {
		deps.Protocol = sealedProtocol{}
	}
	if deps.Projection == nil {
		deps.Projection = sealedProjection{}
	}
	return deps
}

type frameResult struct {
	frame []byte
	err   error
}

// Session owns exactly one browser-provided Native Messaging connection.
// Nothing is process-global: state lives on the session value, and each new
// connection starts from Starting with an empty lineage.
type Session struct {
	id          string
	config      Config
	limits      Limits
	clock       Clock
	diagnostics *Diagnostics
	health      *healthTracker
	reader      *FrameReader
	writer      *FrameWriter
	protocol    ProtocolHandler
	projector   ProjectionSync
	store       Persistence

	rootCtx    context.Context
	cancelRoot context.CancelCauseFunc

	requests *requestRegistry
	slots    chan struct{}
	frames   chan frameResult
	inflight sync.WaitGroup

	mu           sync.Mutex
	started      bool
	readyReached bool
	stopCause    error

	finishOnce    sync.Once
	finishOutcome Outcome
	finished      chan struct{}
	shutdownAt    time.Time
}

func newSession(config Config, input io.Reader, output io.Writer, deps Dependencies) *Session {
	resolved := deps.resolved()
	rootCtx, cancelRoot := context.WithCancelCause(context.Background())
	sessionID := newSessionID()
	return &Session{
		id:          sessionID,
		config:      config,
		limits:      config.Limits,
		clock:       config.Clock,
		diagnostics: config.Diagnostics,
		health:      newHealthTracker(config.Clock, sessionID, config.HostVersion, config.ProtocolVersion),
		reader:      newFrameReader(input, config.Limits),
		writer:      newFrameWriter(output, config.Limits),
		protocol:    resolved.Protocol,
		projector:   resolved.Projection,
		store:       resolved.Persistence,
		rootCtx:     rootCtx,
		cancelRoot:  cancelRoot,
		requests:    newRequestRegistry(),
		slots:       make(chan struct{}, config.Limits.MaxConcurrentRequests),
		frames:      make(chan frameResult, config.Limits.MaxQueuedFrames),
		finished:    make(chan struct{}),
	}
}

// ID returns the opaque session identity recorded in bounded diagnostics.
func (s *Session) ID() string { return s.id }

// WriteFrame emits one complete frame through the single serialized writer. It
// is the only supported way for a protocol handler to reach the peer.
func (s *Session) WriteFrame(ctx context.Context, payload []byte) error {
	if err := s.writer.WriteFrame(ctx, payload); err != nil {
		class, _ := ClassifyFailure(err)
		s.diagnostics.Observe(s.health.current(), levelForError(err), EventFrameWriteFailed, 0, 1)
		return fail(class, ErrSessionClosing)
	}
	s.health.count(func(snapshot *HealthSnapshot) { snapshot.FramesWritten = s.writer.count() })
	return nil
}

// Health returns the current bounded runtime status.
func (s *Session) Health() HealthSnapshot { return s.health.current() }

// Subscribe registers a lifecycle observer such as the IP-16 diagnostics sink.
func (s *Session) Subscribe(observer HealthObserver) { s.health.subscribe(observer) }

// Diagnostics returns the bounded diagnostic recorder for local inspection.
func (s *Session) Diagnostics() *Diagnostics { return s.diagnostics }

// Run owns the whole session: it starts the single input reader, performs the
// ordered startup handoff, serves bounded concurrent work, and finishes exactly
// once with a deterministic outcome.
//
// A clean end of input stream exits immediately. A bounded startup or runtime
// failure exits with its classified outcome. A broken transport after Ready
// publishes recovering with retryability and waits for Close, because the
// extension owns the decision to open the replacement connection and a fresh
// session must start from a new epoch on its own.
func (s *Session) Run(ctx context.Context) Outcome {
	if !s.begin() {
		return newOutcome(FailureInternal, false)
	}
	go s.watch(ctx)
	go s.readLoop()
	err := s.startup()
	if err == nil {
		err = s.serve()
	}
	if err != nil && !isCleanStreamEnd(err) && !s.stopped() {
		failure, retryable := ClassifyFailure(err)
		if failure != FailureNone {
			s.reportFailure(failure, retryable)
			if retryable && recoverableTransport(failure) && s.reachedReady() {
				<-s.finished
				return s.finishOutcome
			}
		}
	}
	return s.finish(s.outcomeFor(err))
}

// recoverableTransport reports whether a failure is a broken browser transport
// that a new connection can replace. Budget and contract failures are not: they
// are terminal for this process because retrying the same session would repeat
// them.
func recoverableTransport(failure FailureClass) bool {
	switch failure {
	case FailureTransportBrokenPipe, FailureTransportUnavailable:
		return true
	default:
		return false
	}
}

func (s *Session) begin() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.started {
		return false
	}
	s.started = true
	return true
}

func (s *Session) watch(ctx context.Context) {
	select {
	case <-ctx.Done():
		cause := context.Cause(ctx)
		s.mu.Lock()
		s.stopCause = cause
		s.mu.Unlock()
		s.cancelRoot(cause)
	case <-s.finished:
	}
}

func (s *Session) stopped() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.stopCause != nil || ctxCancelled(s.root())
}

func (s *Session) root() context.Context { return s.rootCtx }

func (s *Session) persistence() Persistence { return s.store }

func (s *Session) remainingBudget() time.Duration {
	if s.shutdownAt.IsZero() {
		return s.limits.ShutdownTimeout
	}
	remaining := s.shutdownAt.Sub(s.clock.Now())
	if remaining <= 0 {
		return time.Millisecond
	}
	return remaining
}

func ctxCancelled(ctx context.Context) bool {
	cause := context.Cause(ctx)
	return cause != nil && (errors.Is(cause, context.Canceled) || errors.Is(cause, context.DeadlineExceeded))
}

func isCleanStreamEnd(err error) bool {
	return err == nil || errors.Is(err, io.EOF) || errors.Is(err, ErrSessionClosing)
}

func (s *Session) outcomeFor(err error) Outcome {
	readyReached := s.reachedReady()
	if err == nil || s.stopped() || isCleanStreamEnd(err) {
		return newOutcome(FailureNone, readyReached)
	}
	failure, retryable := ClassifyFailure(err)
	outcome := newOutcome(failure, readyReached)
	outcome.Retryable = retryable && failure != FailureNone
	return outcome
}

// readLoop is the only goroutine that touches the input stream.
func (s *Session) readLoop() {
	for {
		frame, err := s.reader.ReadFrame(s.root())
		if err != nil {
			s.deliver(frameResult{err: err})
			s.diagnostics.Observe(s.health.current(), levelForError(err), EventFrameReadFailed, 0, 1)
			return
		}
		if !s.deliver(frameResult{frame: frame}) {
			return
		}
	}
}

func (s *Session) deliver(result frameResult) bool {
	select {
	case s.frames <- result:
		return true
	case <-s.root().Done():
		return false
	}
}

func (s *Session) startup() error {
	s.preparePersistence()
	if err := s.negotiate(); err != nil {
		return err
	}
	return s.synchronize()
}

func (s *Session) preparePersistence() {
	store := s.persistence()
	if store == nil {
		s.health.mutate(func(snapshot *HealthSnapshot) { snapshot.Persistence = PersistenceDisabled })
		s.diagnostics.Observe(s.health.current(), LevelInfo, EventPersistenceSettled, 0, 0)
		return
	}
	s.diagnostics.Observe(s.health.current(), LevelInfo, EventPersistenceStarted, 0, 0)
	ctx, cancel := withBudget(s.root(), s.limits.StartupTimeout, ErrStartupBudget)
	defer cancel()
	status, err := boundedCall(ctx, func() (PersistenceStatus, error) { return store.Prepare(ctx) })
	if err != nil {
		s.health.mutate(func(snapshot *HealthSnapshot) { snapshot.Persistence = PersistenceDegraded })
		s.diagnostics.Observe(s.health.current(), LevelWarn, EventPersistenceSettled, s.limits.StartupTimeout.Milliseconds(), 1)
		return
	}
	if status != PersistenceReady && status != PersistenceDegraded && status != PersistenceDisabled {
		status = PersistenceDegraded
	}
	s.health.mutate(func(snapshot *HealthSnapshot) { snapshot.Persistence = status })
	s.diagnostics.Observe(s.health.current(), LevelInfo, EventPersistenceSettled, 0, 0)
}

func (s *Session) negotiate() error {
	s.publish(StateHandshaking, FailureNone, false)
	s.diagnostics.Observe(s.health.current(), LevelInfo, EventHandshakeStarted, 0, 0)
	ctx, cancel := withBudget(s.root(), s.limits.HandshakeTimeout, ErrHandshakeBudget)
	defer cancel()
	result, err := s.awaitFrame(ctx)
	if err != nil {
		return err
	}
	if result.err != nil {
		return result.err
	}
	binding, err := boundedCall(ctx, func() (Binding, error) { return s.protocol.Negotiate(ctx, s, result.frame) })
	if err != nil {
		return err
	}
	if err := validateBinding(binding, s.config); err != nil {
		return err
	}
	s.health.mutate(func(snapshot *HealthSnapshot) {
		snapshot.ProfileID = binding.ProfileID
		snapshot.ContextKind = binding.ContextKind
		snapshot.ProtocolVersion = binding.ProtocolVersion
		if binding.HostVersion != "" {
			snapshot.HostVersion = binding.HostVersion
		}
	})
	s.diagnostics.Observe(s.health.current(), LevelInfo, EventHandshakeStarted, s.limits.HandshakeTimeout.Milliseconds(), 1)
	return nil
}

func (s *Session) synchronize() error {
	s.publish(StateSynchronizing, FailureNone, false)
	s.diagnostics.Observe(s.health.current(), LevelInfo, EventSynchronizeStarted, 0, 0)
	ctx, cancel := withBudget(s.root(), s.limits.SynchronizationTimeout, ErrSynchronizationBudget)
	defer cancel()
	target := Target{SessionID: s.id, ProfileID: s.health.current().ProfileID, ContextKind: s.health.current().ContextKind}
	status, err := boundedCall(ctx, func() (ProjectionStatus, error) { return s.projector.Synchronize(ctx, target) })
	if err != nil {
		return err
	}
	if !status.Known || !status.Ready || status.Revision == 0 {
		return fail(FailureSnapshotRejected, ErrSnapshotNotReady)
	}
	s.health.mutate(func(snapshot *HealthSnapshot) {
		snapshot.ProjectionKnown = true
		snapshot.ProjectionEpoch = status.Epoch
		snapshot.ProjectionRevision = status.Revision
		snapshot.ProjectionCount = status.Count
	})
	s.mu.Lock()
	s.readyReached = true
	s.mu.Unlock()
	s.publish(StateReady, FailureNone, false)
	s.diagnostics.Observe(s.health.current(), LevelInfo, EventSynchronizeStarted, s.limits.SynchronizationTimeout.Milliseconds(), 1)
	return nil
}

func (s *Session) awaitFrame(ctx context.Context) (frameResult, error) {
	select {
	case result := <-s.frames:
		return result, nil
	case <-s.finished:
		return frameResult{}, ErrSessionClosing
	case <-ctx.Done():
		return frameResult{}, context.Cause(ctx)
	}
}

func (s *Session) serve() error {
	for {
		select {
		case <-s.finished:
			return ErrSessionClosing
		case <-s.root().Done():
			return context.Cause(s.root())
		case result := <-s.frames:
			if result.err != nil {
				return result.err
			}
			s.dispatch(result.frame)
		}
	}
}

func (s *Session) dispatch(frame []byte) {
	select {
	case s.slots <- struct{}{}:
	default:
		s.rejectFrame(FailureConcurrencyLimit)
		return
	}
	ctx, release, _ := s.requests.begin(s.root(), s.limits.RequestTimeout)
	s.inflight.Add(1)
	go func() {
		defer func() {
			release()
			<-s.slots
			s.inflight.Done()
		}()
		_, err := boundedCall(ctx, func() (struct{}, error) { return struct{}{}, s.protocol.Accept(ctx, s, frame) })
		if err != nil {
			failure, retryable := ClassifyFailure(err)
			s.diagnostics.Observe(s.health.current(), levelForError(err), EventRequestFailed, 0, 1)
			s.notify(failure, retryable)
			return
		}
		s.requests.complete()
		s.syncCounters()
	}()
}

func (s *Session) rejectFrame(failure FailureClass) {
	s.requests.reject()
	s.diagnostics.Observe(s.health.current(), LevelWarn, EventRequestRejected, 0, 1)
	s.notify(failure, true)
	if s.requests.inflight() >= s.limits.MaxConcurrentRequests*8 {
		s.reportFailure(FailureQueueOverflow, true)
	}
}

func (s *Session) syncCounters() {
	started, completed, rejected := s.requests.counters()
	framesRead, framesWritten := s.reader.count(), s.writer.count()
	s.health.count(func(snapshot *HealthSnapshot) {
		snapshot.FramesRead = framesRead
		snapshot.FramesWritten = framesWritten
		snapshot.RequestsStarted = started
		snapshot.RequestsCompleted = completed
		snapshot.RequestsRejected = rejected
		snapshot.DroppedDiagnostics = s.diagnostics.Dropped()
	})
}

// reportFailure publishes unavailable or recovering before the terminal close so
// the extension observes the failure with its retryability.
func (s *Session) reportFailure(failure FailureClass, retryable bool) {
	state := StateUnavailable
	if retryable {
		state = StateRecovering
	}
	if s.health.current().State.Terminal() {
		return
	}
	s.publish(state, failure, retryable)
	s.diagnostics.Observe(s.health.current(), levelForState(failure), EventSessionFailed, 0, 1)
	s.notify(failure, retryable)
}

func (s *Session) notify(failure FailureClass, retryable bool) {
	if s.writer.isClosed() || failure == FailureNone {
		return
	}
	ctx, cancel := withBudget(s.root(), s.limits.RequestTimeout, ErrRequestBudget)
	defer cancel()
	notification := Notification{
		Reason:       NotificationFailure,
		State:        s.health.current().State,
		Availability: s.health.current().Availability,
		Failure:      failure,
		Retryable:    retryable,
		Snapshot:     s.health.current(),
	}
	if _, err := boundedCall(ctx, func() (struct{}, error) { return struct{}{}, s.protocol.Notify(ctx, s, notification) }); err != nil {
		s.diagnostics.Observe(s.health.current(), LevelWarn, EventFrameWriteFailed, 0, 1)
	}
}

func (s *Session) finishedRequests() uint64 {
	_, completed, _ := s.requests.counters()
	return completed
}

func validateBinding(binding Binding, config Config) error {
	if _, err := domain.ParseProfileID(string(binding.ProfileID)); err != nil {
		return fail(FailureProfileMismatch, ErrBindingInvalid)
	}
	if binding.ContextKind != domain.ContextNormal && binding.ContextKind != domain.ContextPrivate {
		return fail(FailureProfileMismatch, ErrBindingInvalid)
	}
	if config.ExpectedProfile != "" && config.ExpectedProfile != binding.ProfileID {
		return fail(FailureProfileMismatch, ErrBindingInvalid)
	}
	if binding.ProtocolVersion != config.ProtocolVersion {
		return fail(FailureProtocolIncompatible, ErrBindingInvalid)
	}
	return nil
}
