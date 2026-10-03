package protocol

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"strconv"
	"sync"
	"time"

	"github.com/ryantr-statinops/InfoBoard/host/internal/domain"
	"github.com/ryantr-statinops/InfoBoard/host/internal/runtime"
)

// State is the protocol-level connection state. It is deliberately separate
// from the runtime lifecycle substate: the runtime owns the process, this type
// owns the wire contract inside one connection.
type State string

const (
	StateUnbound      State = "unbound"
	StateBound        State = "bound"
	StateReady        State = "ready"
	StateDegraded     State = "degraded"
	StateIncompatible State = "incompatible"
	StateDisconnected State = "disconnected"
	StateShutdown     State = "shutdown"
)

// SyncState is the projection and index view the protocol needs. The
// projection owner (IP-05) and index owner (IP-10) supply it.
type SyncState struct {
	Known      bool
	Epoch      string
	Revision   uint64
	Sequence   uint64
	Count      uint64
	Rebuilding bool
}

// RequestContext is the bounded identity of one accepted request.
type RequestContext struct {
	RequestID          string
	ProfileID          domain.ProfileID
	ContextKind        domain.ContextKind
	ProjectionRevision uint64
	ReceivedAt         time.Time
}

// SyncRequest is one authoritative snapshot or ordered delta handoff.
type SyncRequest struct {
	Context  RequestContext
	Epoch    string
	Revision uint64
	Sequence uint64
	Snapshot *Snapshot
	Delta    *Delta
}

// SyncOutcome is the committed projection result for a sync request.
type SyncOutcome struct {
	Accepted bool
	Revision uint64
	Sequence uint64
	Count    uint64
	Epoch    string
	Reason   ResyncReason
	Rebuild  bool
}

// SyncOwner is the projection and index hook. It is supplied by IP-05 and
// IP-10; the protocol layer never mutates projection state itself.
type SyncOwner interface {
	ApplySnapshot(ctx context.Context, request SyncRequest) (SyncOutcome, error)
	ApplyDelta(ctx context.Context, request SyncRequest) (SyncOutcome, error)
	State() SyncState
}

// QueryRequest is one bounded lexical query.
type QueryRequest struct {
	Context         RequestContext
	Text            string
	ResultLimit     int
	CurrentWindowID int64
}

// QueryOwner is the lexical index hook supplied by IP-10 and IP-12.
type QueryOwner interface {
	Query(ctx context.Context, request QueryRequest) (QueryResult, error)
}

// ActivationHook records the extension-owned activation outcome for recency
// metadata and diagnostics.
type ActivationHook interface {
	RecordActivation(ctx context.Context, request RequestContext, status ActivationStatus) error
}

// Config is the immutable protocol configuration.
type Config struct {
	HostVersion         string
	RankingModelVersion string
	Limits              Limits
	Budgets             Budgets
}

// Validate rejects an unusable protocol configuration before a connection is
// served.
func (config Config) Validate() error {
	if config.HostVersion == "" || len(config.HostVersion) > MaxOpaqueValueBytes {
		return ErrInvalidLimits
	}
	if err := config.Limits.Validate(); err != nil {
		return err
	}
	return config.Budgets.Validate()
}

type ledgerEntry struct {
	digest   [sha256.Size]byte
	response []byte
}

type sessionHooks struct {
	sync       SyncOwner
	query      QueryOwner
	activation ActivationHook
}

// Session is the IP-07 protocol handler for one Native Messaging connection.
// It implements runtime.ProtocolHandler, so the runtime keeps sole ownership of
// the streams while this type owns the wire contract.
type Session struct {
	config Config
	hooks  sessionHooks

	mu            sync.Mutex
	state         State
	profileID     domain.ProfileID
	contextKind   domain.ContextKind
	ledger        map[string]ledgerEntry
	requests      map[string]struct{}
	revision      uint64
	sequence      uint64
	count         uint64
	epoch         string
	rebuilding    bool
	persistence   string
	sessionID     string
	notifications uint64
	counters      HealthCounters
	now           func() time.Time
}

// NewSession builds a protocol session. Nil hooks are allowed and fail closed:
// without a sync owner the connection never reaches ready, and without a query
// owner every query is refused rather than answered from a partial index.
func NewSession(config Config, syncOwner SyncOwner, queryOwner QueryOwner, activationHook ActivationHook) (*Session, error) {
	if err := config.Validate(); err != nil {
		return nil, err
	}
	return &Session{
		config:    config,
		sessionID: newSessionIdentity(),
		hooks:     sessionHooks{sync: syncOwner, query: queryOwner, activation: activationHook},
		state:     StateUnbound,
		ledger:    make(map[string]ledgerEntry, 16),
		requests:  make(map[string]struct{}, 16),
		now:       time.Now,
	}, nil
}

// SetClock replaces the wall clock used for bounded request stamps so fixture
// runs stay deterministic.
func (session *Session) SetClock(now func() time.Time) {
	if now == nil {
		return
	}
	session.mu.Lock()
	defer session.mu.Unlock()
	session.now = now
}

// State reports the current protocol state.
func (session *Session) State() State {
	session.mu.Lock()
	defer session.mu.Unlock()
	return session.state
}

// Negotiate handles the single handshake frame. It is the only frame accepted
// before the connection is bound, and it never mutates projection state.
func (session *Session) Negotiate(ctx context.Context, channel runtime.FrameChannel, frame []byte) (runtime.Binding, error) {
	session.mu.Lock()
	if session.state != StateUnbound {
		session.mu.Unlock()
		return runtime.Binding{}, session.refuse(ctx, channel, "", NewError(CodeProtocolMismatch), false)
	}
	session.mu.Unlock()

	envelope, failure := DecodeEnvelope(frame, session.config.Limits)
	if failure != nil {
		session.mu.Lock()
		if failure.Code == CodeProtocolMismatch {
			session.state = StateIncompatible
		} else {
			session.state = StateUnbound
		}
		session.mu.Unlock()
		return runtime.Binding{}, session.refuse(ctx, channel, "", failure, false)
	}
	if envelope.Type != TypeHello {
		session.mu.Lock()
		session.state = StateUnbound
		session.mu.Unlock()
		return runtime.Binding{}, session.refuse(ctx, channel, envelope.RequestID, NewError(CodeSnapshotRequired), false)
	}
	hello, failure := DecodeHello(envelope.Payload, session.config.Limits)
	if failure != nil {
		session.mu.Lock()
		session.state = StateIncompatible
		session.mu.Unlock()
		return runtime.Binding{}, session.refuse(ctx, channel, envelope.RequestID, failure, failure.Code == CodeProtocolMismatch)
	}
	profile, err := domain.ParseProfileID(envelope.ProfileID)
	if err != nil {
		session.mu.Lock()
		session.state = StateIncompatible
		session.mu.Unlock()
		return runtime.Binding{}, session.refuse(ctx, channel, envelope.RequestID, NewError(CodeProfileMismatch), false)
	}
	acknowledgement := HelloAck{
		Protocol:            CurrentVersion,
		HostVersion:         session.config.HostVersion,
		Limits:              session.config.Limits,
		RankingModelVersion: session.config.RankingModelVersion,
		Persistence:         session.persistenceState(),
	}
	if failure := session.respond(ctx, channel, envelope, TypeHelloAck, acknowledgement, [sha256.Size]byte{}); failure != nil {
		session.mu.Lock()
		session.state = StateIncompatible
		session.mu.Unlock()
		return runtime.Binding{}, failure
	}
	session.mu.Lock()
	session.profileID = profile
	session.contextKind = hello.ContextKind
	session.state = StateBound
	session.mu.Unlock()
	return runtime.Binding{
		ProfileID:       profile,
		ContextKind:     hello.ContextKind,
		ProtocolVersion: CurrentVersion,
		HostVersion:     session.config.HostVersion,
	}, nil
}

// Accept handles every frame after the connection is bound.
func (session *Session) Accept(ctx context.Context, channel runtime.FrameChannel, frame []byte) error {
	envelope, failure := DecodeEnvelope(frame, session.config.Limits)
	if failure != nil {
		return session.refuse(ctx, channel, "", failure, false)
	}
	session.mu.Lock()
	switch session.state {
	case StateShutdown, StateDisconnected:
		session.mu.Unlock()
		return session.refuse(ctx, channel, envelope.RequestID, NewError(CodeHostShutdown), true)
	case StateIncompatible:
		session.mu.Unlock()
		return session.refuse(ctx, channel, envelope.RequestID, NewError(CodeProtocolMismatch), true)
	}
	boundProfile := session.profileID
	bound := session.state != StateUnbound
	session.mu.Unlock()

	if envelope.Type.Direction() != DirectionToHost {
		return session.refuse(ctx, channel, envelope.RequestID, NewError(CodeInvalidFrame), false)
	}
	if !bound {
		return session.refuse(ctx, channel, envelope.RequestID, NewError(CodeSnapshotRequired), false)
	}
	if profile, err := domain.ParseProfileID(envelope.ProfileID); err != nil || profile != boundProfile {
		return session.refuse(ctx, channel, envelope.RequestID, NewError(CodeProfileMismatch), false)
	}

	digest := sha256.Sum256(frame)
	replayed, duplicate, conflicting := session.classifyRequest(envelope.RequestID, digest)
	session.mu.Lock()
	session.counters.RequestsStarted++
	session.mu.Unlock()
	if conflicting {
		// The same request identity with different bytes fails closed.
		session.closeLocked(StateShutdown)
		return session.refuse(ctx, channel, envelope.RequestID, NewError(CodeInvalidFrame), true)
	}
	if duplicate {
		if len(replayed) == 0 {
			return nil
		}
		return channel.WriteFrame(ctx, replayed)
	}
	if err := session.dispatch(ctx, channel, envelope, digest); err != nil {
		return err
	}
	return nil
}

func (session *Session) dispatch(ctx context.Context, channel runtime.FrameChannel, envelope Envelope, digest [sha256.Size]byte) error {
	switch envelope.Type {
	case TypeSnapshot:
		return session.handleSnapshot(ctx, channel, envelope, digest)
	case TypeDelta:
		return session.handleDelta(ctx, channel, envelope, digest)
	case TypeQuery:
		return session.handleQuery(ctx, channel, envelope, digest)
	case TypeHealth:
		return session.handleHealth(ctx, channel, envelope, digest)
	case TypeActivationObserved, TypeActivationFailed:
		return session.handleActivation(ctx, channel, envelope, digest)
	case TypeHello:
		return session.refuse(ctx, channel, envelope.RequestID, NewError(CodeProtocolMismatch), false)
	default:
		return session.refuse(ctx, channel, envelope.RequestID, NewError(CodeInvalidFrame), false)
	}
}

func (session *Session) handleSnapshot(ctx context.Context, channel runtime.FrameChannel, envelope Envelope, digest [sha256.Size]byte) error {
	snapshot, failure := DecodeSnapshot(envelope.Payload, session.config.Limits)
	if failure != nil {
		return session.refuse(ctx, channel, envelope.RequestID, failure, false)
	}
	request := SyncRequest{
		Context:  session.requestContext(envelope),
		Revision: envelope.ProjectionRevision,
		Sequence: snapshot.Sequence,
		Snapshot: &snapshot,
	}
	if failure := session.validateRecords(request); failure != nil {
		return session.refuse(ctx, channel, envelope.RequestID, failure, false)
	}
	budgeted, cancel := context.WithTimeout(ctx, session.config.Budgets.SyncApply)
	defer cancel()
	outcome := SyncOutcome{}
	if session.hooks.sync == nil {
		return session.refuse(ctx, channel, envelope.RequestID, NewError(CodeSnapshotRequired), true)
	}
	applied, err := session.hooks.sync.ApplySnapshot(budgeted, request)
	if err != nil {
		return session.refuse(ctx, channel, envelope.RequestID, session.classifySyncError(budgeted, ctx, err), true)
	}
	outcome = applied
	if !outcome.Accepted {
		session.markRebuilding()
		return session.requireResync(ctx, channel, envelope.RequestID, outcome, NewError(CodeRevisionMismatch))
	}
	session.commit(outcome)
	return session.respond(ctx, channel, envelope, TypeSyncAck, SyncAck{AcceptedRevision: outcome.Revision, AcceptedSequence: outcome.Sequence}, digest)
}

func (session *Session) handleDelta(ctx context.Context, channel runtime.FrameChannel, envelope Envelope, digest [sha256.Size]byte) error {
	delta, failure := DecodeDelta(envelope.Payload, session.config.Limits)
	if failure != nil {
		return session.refuse(ctx, channel, envelope.RequestID, failure, false)
	}
	state := session.syncState()
	if !state.Known {
		return session.refuse(ctx, channel, envelope.RequestID, NewError(CodeSnapshotRequired), true)
	}
	if delta.BaseRevision != state.Revision {
		session.markRebuilding()
		return session.requireResync(ctx, channel, envelope.RequestID, SyncOutcome{Revision: state.Revision, Sequence: state.Sequence}, NewError(CodeRevisionMismatch))
	}
	if state.Sequence != 0 && delta.SequenceStart != state.Sequence+1 {
		session.markRebuilding()
		return session.requireResync(ctx, channel, envelope.RequestID, SyncOutcome{Revision: state.Revision, Sequence: state.Sequence}, NewError(CodeRevisionMismatch))
	}
	if envelope.ProjectionRevision != state.Revision+1 {
		session.markRebuilding()
		return session.requireResync(ctx, channel, envelope.RequestID, SyncOutcome{Revision: state.Revision, Sequence: state.Sequence}, NewError(CodeRevisionMismatch))
	}
	request := SyncRequest{
		Context:  session.requestContext(envelope),
		Revision: envelope.ProjectionRevision,
		Sequence: delta.SequenceEnd,
		Delta:    &delta,
	}
	for index := range delta.Operations {
		if record := delta.Operations[index].Record; record != nil {
			if failure := session.validateRecord(request.Context, *record); failure != nil {
				return session.refuse(ctx, channel, envelope.RequestID, failure, false)
			}
		}
	}
	budgeted, cancel := context.WithTimeout(ctx, session.config.Budgets.SyncApply)
	defer cancel()
	if session.hooks.sync == nil {
		return session.refuse(ctx, channel, envelope.RequestID, NewError(CodeSnapshotRequired), true)
	}
	outcome, err := session.hooks.sync.ApplyDelta(budgeted, request)
	if err != nil {
		return session.refuse(ctx, channel, envelope.RequestID, session.classifySyncError(budgeted, ctx, err), true)
	}
	if !outcome.Accepted {
		session.markRebuilding()
		return session.requireResync(ctx, channel, envelope.RequestID, outcome, NewError(CodeRevisionMismatch))
	}
	session.commit(outcome)
	return session.respond(ctx, channel, envelope, TypeSyncAck, SyncAck{AcceptedRevision: outcome.Revision, AcceptedSequence: outcome.Sequence}, digest)
}

func (session *Session) handleQuery(ctx context.Context, channel runtime.FrameChannel, envelope Envelope, digest [sha256.Size]byte) error {
	query, failure := DecodeQuery(envelope.Payload, session.config.Limits)
	if failure != nil {
		return session.refuse(ctx, channel, envelope.RequestID, failure, false)
	}
	state := session.syncState()
	if !state.Known {
		return session.refuse(ctx, channel, envelope.RequestID, NewError(CodeSnapshotRequired), true)
	}
	if state.Rebuilding || session.isRebuilding() {
		return session.refuse(ctx, channel, envelope.RequestID, NewError(CodeIndexRebuilding), true)
	}
	if envelope.ProjectionRevision != state.Revision {
		return session.refuse(ctx, channel, envelope.RequestID, NewError(CodeRevisionMismatch), true)
	}
	if session.hooks.query == nil {
		return session.refuse(ctx, channel, envelope.RequestID, NewError(CodeIndexRebuilding), true)
	}
	budget := session.config.Budgets.QueryHostBudget
	if session.config.Budgets.QueryRequestBudget < budget {
		budget = session.config.Budgets.QueryRequestBudget
	}
	budgeted, cancel := context.WithTimeoutCause(ctx, budget, ErrQueryBudget)
	defer cancel()
	result, err := session.hooks.query.Query(budgeted, QueryRequest{
		Context:         session.requestContext(envelope),
		Text:            query.Text,
		ResultLimit:     query.ResultLimit,
		CurrentWindowID: query.CurrentWindowID,
	})
	if err != nil {
		if errors.Is(context.Cause(budgeted), ErrQueryBudget) {
			return session.refuse(ctx, channel, envelope.RequestID, NewError(CodeQueryTimeout), true)
		}
		if ctx.Err() != nil {
			// The runtime cancelled this request: drop the response safely.
			session.forget(envelope.RequestID)
			return nil
		}
		return session.refuse(ctx, channel, envelope.RequestID, NewError(CodeInternalFailure), false)
	}
	if len(result.Results) > session.config.Limits.MaxResults {
		return session.refuse(ctx, channel, envelope.RequestID, NewError(CodePayloadLimit).WithBound("result_count", session.config.Limits.MaxResults, len(result.Results)), false)
	}
	result.ProjectionRevision = state.Revision
	result.RankingModelVersion = session.config.RankingModelVersion
	result.Counters.Returned = int64(len(result.Results))
	// A degraded store stays visible in the counters annotation while the
	// lexical result remains available.
	if session.persistenceState() == string(runtime.PersistenceDegraded) {
		result.Counters.DegradedCode = CodePersistenceDegraded
	}
	return session.respond(ctx, channel, envelope, TypeQueryResult, result, digest)
}

func (session *Session) handleHealth(ctx context.Context, channel runtime.FrameChannel, envelope Envelope, digest [sha256.Size]byte) error {
	budgeted, cancel := context.WithTimeout(ctx, session.config.Budgets.Health)
	defer cancel()
	select {
	case <-budgeted.Done():
		return session.refuse(ctx, channel, envelope.RequestID, NewError(CodeHostShutdown), true)
	default:
	}
	return session.respond(ctx, channel, envelope, TypeHealthResult, session.healthResult(), digest)
}

func (session *Session) handleActivation(ctx context.Context, channel runtime.FrameChannel, envelope Envelope, digest [sha256.Size]byte) error {
	status, failure := DecodeActivationStatus(envelope.Payload, envelope.Type)
	if failure != nil {
		return session.refuse(ctx, channel, envelope.RequestID, failure, false)
	}
	state := session.syncState()
	if !state.Known {
		return session.refuse(ctx, channel, envelope.RequestID, NewError(CodeSnapshotRequired), true)
	}
	if envelope.ProjectionRevision != state.Revision {
		return session.refuse(ctx, channel, envelope.RequestID, NewError(CodeRevisionMismatch), true)
	}
	acknowledgement := ActivationAck{Accepted: true, ProjectionRevision: state.Revision}
	budgeted, cancel := context.WithTimeout(ctx, session.config.Budgets.ActivationStatus)
	defer cancel()
	if session.hooks.activation != nil {
		if err := session.hooks.activation.RecordActivation(budgeted, session.requestContext(envelope), status); err != nil {
			acknowledgement.Accepted = false
		}
	}
	return session.respond(ctx, channel, envelope, TypeActivationAck, acknowledgement, digest)
}

// Notify maps a runtime lifecycle notification onto the wire contract. It is
// best effort: a notification never blocks shutdown.
func (session *Session) Notify(ctx context.Context, channel runtime.FrameChannel, notification runtime.Notification) error {
	if channel == nil {
		return nil
	}
	session.mu.Lock()
	profile := session.profileID
	bound := session.state != StateUnbound && profile != ""
	session.mu.Unlock()
	if !bound {
		return nil
	}
	code, mapped := codeForRuntimeFailure(notification.Failure)
	if notification.Failure != runtime.FailureNone && !mapped {
		return nil
	}
	payload := ErrorPayload{
		Code:       code,
		Retryable:  notification.Retryable,
		MessageKey: code.MessageKey(),
	}
	if notification.Failure == runtime.FailureNone {
		payload.MessageKey = MessageHealthReported
	}
	envelope := Envelope{
		Protocol:  CurrentVersion,
		Type:      TypeError,
		RequestID: session.notificationRequestID(),
		ProfileID: string(profile),
		Payload:   EmptyPayload(),
	}
	encoded, encodeFailure := PayloadOf(payload)
	if encodeFailure != nil {
		return encodeFailure
	}
	envelope.Payload = encoded
	frame, failure := EncodeEnvelope(envelope, session.config.Limits)
	if failure != nil {
		return failure
	}
	if err := channel.WriteFrame(ctx, frame); err != nil {
		return NewError(CodeHostShutdown)
	}
	return nil
}

// Shutdown clears connection-local protocol state. A later connection starts
// from a clean unbound state and needs its own hello and snapshot.
func (session *Session) Shutdown(ctx context.Context) error {
	session.mu.Lock()
	defer session.mu.Unlock()
	if session.state == StateShutdown {
		return nil
	}
	session.state = StateShutdown
	session.ledger = make(map[string]ledgerEntry, 16)
	session.requests = make(map[string]struct{}, 16)
	session.epoch = ""
	session.revision = 0
	session.sequence = 0
	session.count = 0
	session.rebuilding = false
	return nil
}

// Disconnect records a transport-level disconnect so in-flight work fails with
// HOST_SHUTDOWN and no later frame is accepted.
func (session *Session) Disconnect() {
	session.closeLocked(StateDisconnected)
}

func (session *Session) closeLocked(state State) {
	session.mu.Lock()
	defer session.mu.Unlock()
	if session.state == StateShutdown {
		return
	}
	session.state = state
	session.ledger = make(map[string]ledgerEntry, 16)
	session.requests = make(map[string]struct{}, 16)
}

// SessionID returns the opaque, bounded connection identity reported in
// health_result.
func (session *Session) SessionID() string {
	session.mu.Lock()
	defer session.mu.Unlock()
	return session.sessionID
}

// newSessionIdentity returns a bounded opaque connection identity. It never
// carries tab, profile, or path data.
func newSessionIdentity() string {
	raw := make([]byte, 8)
	if _, err := rand.Read(raw); err != nil {
		return "0000000000000000"
	}
	return hex.EncodeToString(raw)
}

func (session *Session) persistenceState() string {
	session.mu.Lock()
	defer session.mu.Unlock()
	if session.persistence == "" {
		return string(runtime.PersistenceUnknown)
	}
	return session.persistence
}

// SetPersistence records the storage posture reported in hello_ack and
// health_result. A degraded store never blocks lexical results.
func (session *Session) SetPersistence(status runtime.PersistenceStatus) {
	session.mu.Lock()
	defer session.mu.Unlock()
	if status == "" {
		status = runtime.PersistenceUnknown
	}
	session.persistence = string(status)
}

// SetRebuilding marks the index as rebuilding so queries are refused with
// INDEX_REBUILDING instead of answered from a partial index.
func (session *Session) SetRebuilding(rebuilding bool) {
	session.mu.Lock()
	defer session.mu.Unlock()
	session.rebuilding = rebuilding
	if !rebuilding && session.state == StateDegraded {
		session.state = StateReady
	}
	if rebuilding && session.state == StateReady {
		session.state = StateDegraded
	}
}

func (session *Session) isRebuilding() bool {
	session.mu.Lock()
	defer session.mu.Unlock()
	return session.rebuilding
}

func (session *Session) markRebuilding() {
	session.SetRebuilding(true)
}

func (session *Session) syncState() SyncState {
	session.mu.Lock()
	local := SyncState{
		Known:      session.state == StateReady,
		Epoch:      session.epoch,
		Revision:   session.revision,
		Sequence:   session.sequence,
		Count:      session.count,
		Rebuilding: session.rebuilding,
	}
	owner := session.hooks.sync
	session.mu.Unlock()
	if owner == nil {
		return local
	}
	owned := owner.State()
	if !owned.Known {
		return local
	}
	return owned
}

func (session *Session) commit(outcome SyncOutcome) {
	session.mu.Lock()
	defer session.mu.Unlock()
	session.revision = outcome.Revision
	session.sequence = outcome.Sequence
	session.count = outcome.Count
	session.epoch = outcome.Epoch
	session.rebuilding = session.rebuilding || outcome.Rebuild
	if session.rebuilding {
		session.state = StateDegraded
		return
	}
	session.state = StateReady
}

func (session *Session) requestContext(envelope Envelope) RequestContext {
	session.mu.Lock()
	defer session.mu.Unlock()
	now := session.now
	if now == nil {
		now = time.Now
	}
	return RequestContext{
		RequestID:          envelope.RequestID,
		ProfileID:          session.profileID,
		ContextKind:        session.contextKind,
		ProjectionRevision: envelope.ProjectionRevision,
		ReceivedAt:         now(),
	}
}

func (session *Session) validateRecords(request SyncRequest) *Error {
	if request.Snapshot == nil {
		return NewError(CodeInvalidFrame)
	}
	for index := range request.Snapshot.Tabs {
		if failure := session.validateRecord(request.Context, request.Snapshot.Tabs[index]); failure != nil {
			return failure
		}
	}
	return nil
}

// validateRecord enforces profile and context isolation before any record is
// applied. A cross-profile record is rejected, never merged.
func (session *Session) validateRecord(request RequestContext, record TabRecord) *Error {
	if string(request.ProfileID) != record.ProfileID {
		return NewError(CodeProfileMismatch)
	}
	if string(request.ContextKind) != string(record.ContextKind) {
		return NewError(CodeProfileMismatch)
	}
	if _, err := domain.ParseProfileID(record.ProfileID); err != nil {
		return NewError(CodeProfileMismatch)
	}
	return nil
}

func (session *Session) classifySyncError(budgeted, parent context.Context, err error) *Error {
	if errors.Is(context.Cause(budgeted), context.DeadlineExceeded) {
		return NewError(CodeSnapshotRequired)
	}
	if parent.Err() != nil {
		return NewError(CodeHostShutdown)
	}
	var typed *Error
	if errors.As(err, &typed) {
		return typed
	}
	if code, mapped := codeForRuntimeFailure(runtimeFailureOf(err)); mapped {
		return NewError(code)
	}
	return NewError(CodeInternalFailure)
}

// classifyRequest implements the (connection, request_id) ledger rule: exact
// duplicate bytes replay the same response without a second side effect, while
// the same identity with different bytes fails closed.
func (session *Session) classifyRequest(requestID string, digest [sha256.Size]byte) ([]byte, bool, bool) {
	session.mu.Lock()
	defer session.mu.Unlock()
	if entry, known := session.ledger[requestID]; known {
		if entry.digest == digest {
			return entry.response, true, false
		}
		return nil, false, true
	}
	if _, pending := session.requests[requestID]; pending {
		return nil, true, false
	}
	session.requests[requestID] = struct{}{}
	return nil, false, false
}

func (session *Session) remember(requestID string, digest [sha256.Size]byte, response []byte) {
	session.mu.Lock()
	defer session.mu.Unlock()
	delete(session.requests, requestID)
	session.ledger[requestID] = ledgerEntry{digest: digest, response: response}
}

func (session *Session) forget(requestID string) {
	session.mu.Lock()
	defer session.mu.Unlock()
	delete(session.requests, requestID)
}

// respond writes one correlated response and records it for idempotent replay.
func (session *Session) respond(ctx context.Context, channel runtime.FrameChannel, request Envelope, messageType MessageType, payload any, digest [sha256.Size]byte) error {
	encoded, failure := PayloadOf(payload)
	if failure != nil {
		return failure
	}
	envelope := Envelope{
		Protocol:           CurrentVersion,
		Type:               messageType,
		RequestID:          request.RequestID,
		ProfileID:          request.ProfileID,
		ProjectionRevision: request.ProjectionRevision,
		Payload:            encoded,
	}
	frame, failure := EncodeEnvelope(envelope, session.config.Limits)
	if failure != nil {
		return failure
	}
	session.mu.Lock()
	session.counters.RequestsCompleted++
	session.mu.Unlock()
	session.remember(request.RequestID, digest, frame)
	return channel.WriteFrame(ctx, frame)
}

// refuse writes one typed error response and keeps session state unchanged.
func (session *Session) refuse(ctx context.Context, channel runtime.FrameChannel, requestID string, failure *Error, retryable bool) error {
	if failure == nil {
		failure = NewError(CodeInternalFailure)
	}
	failure.Retryable = failure.Code.Retryable()
	payload := ErrorPayload{
		Code:             failure.Code,
		Retryable:        failure.Retryable,
		MessageKey:       failure.MessageKey,
		RetryAfterMillis: failure.RetryAfterMillis,
		ExpectedRevision: failure.ExpectedRevision,
		ExpectedSequence: failure.ExpectedSequence,
	}
	encoded, encodeFailure := PayloadOf(payload)
	if encodeFailure != nil {
		return encodeFailure
	}
	session.mu.Lock()
	profile := session.profileID
	session.counters.ProtocolErrors++
	bound := session.state != StateUnbound && profile != ""
	session.mu.Unlock()
	if requestID != "" {
		session.forget(requestID)
	}
	if channel == nil || (!bound && failure.Code != CodeSnapshotRequired && failure.Code != CodeProtocolMismatch) {
		// Nothing is bound yet: the failure is reported through the returned
		// error so the runtime can classify it without an unauthenticated frame.
		return failure
	}
	envelope := Envelope{
		Protocol:  CurrentVersion,
		Type:      TypeError,
		RequestID: requestID,
		ProfileID: string(profile),
		Payload:   encoded,
	}
	frame, encodeFailure := EncodeEnvelope(envelope, session.config.Limits)
	if encodeFailure != nil {
		return encodeFailure
	}
	if err := channel.WriteFrame(ctx, frame); err != nil {
		return failure
	}
	if retryable && failure.Code == CodeProtocolMismatch {
		session.closeLocked(StateIncompatible)
	}
	return failure
}

// requireResync emits the resync signal and its typed error without applying
// anything partially.
func (session *Session) requireResync(ctx context.Context, channel runtime.FrameChannel, requestID string, outcome SyncOutcome, failure *Error) error {
	revision, sequence := outcome.Revision, outcome.Sequence
	reason := outcome.Reason
	if !reason.KnownReason() {
		reason = ResyncRevisionMismatch
	}
	failure = failure.WithExpectation(&revision, &sequence).WithRetryAfter(0)
	session.mu.Lock()
	profile := session.profileID
	session.mu.Unlock()
	encoded, encodeFailure := PayloadOf(ResyncRequired{
		Reason:           reason,
		ExpectedRevision: revision,
		ExpectedSequence: sequence,
	})
	if encodeFailure != nil {
		return encodeFailure
	}
	envelope := Envelope{
		Protocol:           CurrentVersion,
		Type:               TypeResyncRequired,
		RequestID:          requestID,
		ProfileID:          string(profile),
		ProjectionRevision: revision,
		Payload:            encoded,
	}
	frame, encodeFailure := EncodeEnvelope(envelope, session.config.Limits)
	if encodeFailure != nil {
		return encodeFailure
	}
	if err := channel.WriteFrame(ctx, frame); err != nil {
		return failure
	}
	return session.refuse(ctx, channel, requestID, failure, true)
}

func (session *Session) healthResult() HealthResult {
	session.mu.Lock()
	defer session.mu.Unlock()
	persistence := session.persistence
	if persistence == "" {
		persistence = string(runtime.PersistenceUnknown)
	}
	indexState := "ready"
	switch {
	case session.rebuilding:
		indexState = "rebuilding"
	case session.state != StateReady && session.state != StateDegraded:
		indexState = "unknown"
	}
	return HealthResult{
		Availability:       availabilityFor(session.state),
		HostState:          string(session.state),
		ProtocolVersion:    CurrentVersion,
		SessionID:          session.sessionID,
		StorageState:       persistence,
		IndexState:         indexState,
		ProjectionRevision: session.revision,
		ProjectionSequence: session.sequence,
		FreshnessAgeMillis: 0,
		Retryable:          session.state == StateDegraded || session.rebuilding,
		Counters:           session.counters,
	}
}

// availabilityFor maps the protocol state onto the public availability the
// search surface consumes. Only a ready, non-degraded session is healthy.
func availabilityFor(state State) string {
	switch state {
	case StateReady:
		return string(runtime.AvailabilityHealthy)
	case StateIncompatible, StateDisconnected, StateShutdown:
		return string(runtime.AvailabilityUnavailable)
	default:
		return string(runtime.AvailabilityRecovering)
	}
}

// notificationRequestID produces a bounded, connection-unique identity for a
// host-initiated frame so it can never collide with a client request identity.
func (session *Session) notificationRequestID() string {
	session.mu.Lock()
	defer session.mu.Unlock()
	session.notifications++
	return "host-" + strconv.FormatUint(session.notifications, 10)
}

// codeForRuntimeFailure maps a runtime failure class onto the exact wire code.
// An unmapped class never reaches the peer.
func codeForRuntimeFailure(class runtime.FailureClass) (ErrorCode, bool) {
	switch class {
	case runtime.FailureNone:
		return CodeHostShutdown, true
	case runtime.FailureProtocolIncompatible:
		return CodeProtocolMismatch, true
	case runtime.FailureProfileMismatch:
		return CodeProfileMismatch, true
	case runtime.FailureFrameTooLarge, runtime.FailureQueueOverflow, runtime.FailureConcurrencyLimit,
		runtime.FailureTransportMalformed, runtime.FailureTransportTruncated:
		return CodePayloadLimit, true
	case runtime.FailureSnapshotRejected, runtime.FailureSynchronizationTimeout:
		return CodeSnapshotRequired, true
	case runtime.FailureRequestTimeout, runtime.FailureHandshakeTimeout, runtime.FailureStartupTimeout:
		return CodeQueryTimeout, true
	case runtime.FailureDependency:
		return CodePersistenceDegraded, true
	case runtime.FailureShutdown, runtime.FailureCancelled, runtime.FailureTransportEOF,
		runtime.FailureTransportBrokenPipe, runtime.FailureTransportUnavailable:
		return CodeHostShutdown, true
	case runtime.FailureHandshakeRejected, runtime.FailureConfiguration, runtime.FailureRegistration:
		return CodeHostShutdown, true
	default:
		return CodeInternalFailure, true
	}
}

// runtimeFailureOf exposes the runtime classifier to the protocol layer without
// duplicating the mapping table.
func runtimeFailureOf(err error) runtime.FailureClass {
	class, _ := runtime.ClassifyFailure(err)
	return class
}
