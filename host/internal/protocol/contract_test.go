package protocol

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/ryantr-statinops/InfoBoard/host/internal/domain"
	"github.com/ryantr-statinops/InfoBoard/host/internal/runtime"
)

const testProfile = "123e4567-e89b-42d3-a456-426614174000"
const testEpoch = "123e4567-e89b-42d3-a456-426614174001"

type recorder struct {
	mu     sync.Mutex
	frames [][]byte
}

func (r *recorder) WriteFrame(_ context.Context, payload []byte) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.frames = append(r.frames, append([]byte(nil), payload...))
	return nil
}

func (r *recorder) envelopes(t *testing.T) []Envelope {
	t.Helper()
	r.mu.Lock()
	defer r.mu.Unlock()
	decoded := make([]Envelope, 0, len(r.frames))
	for _, frame := range r.frames {
		envelope, err := DecodeEnvelope(frame, DefaultLimits())
		if err != nil {
			t.Fatalf("host emitted an invalid frame: %v", err)
		}
		decoded = append(decoded, envelope)
	}
	return decoded
}

func (r *recorder) reset() {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.frames = nil
}

func testConfig() Config {
	return Config{HostVersion: "0.1.0", RankingModelVersion: "rank-1", Limits: DefaultLimits(), Budgets: DefaultBudgets()}
}

func frame(t *testing.T, messageType MessageType, requestID string, revision uint64, payload string) []byte {
	t.Helper()
	encoded, err := EncodeEnvelope(Envelope{
		Protocol:           CurrentVersion,
		Type:               messageType,
		RequestID:          requestID,
		ProfileID:          testProfile,
		ProjectionRevision: revision,
		Payload:            json.RawMessage(payload),
	}, DefaultLimits())
	if err != nil {
		t.Fatalf("encode %s: %v", messageType, err)
	}
	return encoded
}

func helloFrame(t *testing.T) []byte {
	t.Helper()
	return frame(t, TypeHello, "r-1", 0, `{"extension_version":"1.0.0","browser_family":"chrome","browser_version":"126.0","context_kind":"normal","capabilities":["query"],"supported_protocol_versions":[1]}`)
}

type fakeSync struct {
	mu        sync.Mutex
	state     SyncState
	snapshots int
	deltas    int
	delay     time.Duration
	failWith  error
}

func (s *fakeSync) ApplySnapshot(ctx context.Context, _ SyncRequest) (SyncOutcome, error) {
	if s.delay > 0 {
		select {
		case <-time.After(s.delay):
		case <-ctx.Done():
			return SyncOutcome{}, ctx.Err()
		}
	}
	if s.failWith != nil {
		return SyncOutcome{}, s.failWith
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.snapshots++
	s.state = SyncState{Known: true, Epoch: testEpoch, Revision: 1, Sequence: 1, Count: 3}
	return SyncOutcome{Accepted: true, Revision: 1, Sequence: 1, Count: 3, Epoch: testEpoch}, nil
}

func (s *fakeSync) ApplyDelta(ctx context.Context, request SyncRequest) (SyncOutcome, error) {
	if s.delay > 0 {
		select {
		case <-time.After(s.delay):
		case <-ctx.Done():
			return SyncOutcome{}, ctx.Err()
		}
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if request.Delta == nil {
		return SyncOutcome{}, fmt.Errorf("delta required")
	}
	s.deltas++
	s.state = SyncState{Known: true, Epoch: testEpoch, Revision: request.Revision, Sequence: request.Sequence, Count: 3}
	return SyncOutcome{Accepted: true, Revision: request.Revision, Sequence: request.Sequence, Count: 3, Epoch: testEpoch}, nil
}

func (s *fakeSync) State() SyncState {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.state
}

type fakeQuery struct {
	delay  time.Duration
	fail   error
	seen   int
	result QueryResult
}

func (q *fakeQuery) Query(ctx context.Context, request QueryRequest) (QueryResult, error) {
	q.seen++
	if q.delay > 0 {
		select {
		case <-time.After(q.delay):
		case <-ctx.Done():
			return QueryResult{}, ctx.Err()
		}
	}
	if q.fail != nil {
		return QueryResult{}, q.fail
	}
	if q.result.Results == nil {
		return QueryResult{Results: []ResultRow{{ResultID: request.Context.RequestID, TabID: 10, WindowID: 1, TitleDisplay: "one", DomainDisplay: "example.test"}}}, nil
	}
	return q.result, nil
}

func newSession(t *testing.T, syncOwner SyncOwner, queryOwner QueryOwner) *Session {
	t.Helper()
	session, err := NewSession(testConfig(), syncOwner, queryOwner, nil)
	if err != nil {
		t.Fatal(err)
	}
	return session
}

func mustBind(t *testing.T, session *Session, out *recorder) {
	t.Helper()
	if _, err := session.Negotiate(context.Background(), out, helloFrame(t)); err != nil {
		t.Fatalf("negotiate: %v", err)
	}
	if session.State() != StateBound {
		t.Fatalf("state after hello = %s", session.State())
	}
}

func snapshotPayload(tabs int) string {
	rows := make([]string, 0, tabs)
	for index := 0; index < tabs; index++ {
		rows = append(rows, fmt.Sprintf(`{"profile_id":%q,"context_kind":"normal","tab_id":%d,"window_id":1,"title_display":"t","title_search":"t","url_display":"example.test","url_search":"example.test","domain_display":"example.test","domain_search":"example.test","pinned":false,"active":false,"eligible":true,"observed_at":1,"projection_epoch":%q,"projection_revision":1}`, testProfile, index+10, testEpoch))
	}
	return fmt.Sprintf(`{"tabs":[%s],"sequence":1}`, strings.Join(rows, ","))
}

func TestProtocol_EnvelopeRejectsEveryMalformedShape(t *testing.T) {
	valid := fmt.Sprintf(`{"protocol":1,"type":"hello","request_id":"r-1","profile_id":%q,"projection_revision":0,"payload":{}}`, testProfile)
	if _, err := DecodeEnvelope([]byte(valid), DefaultLimits()); err != nil {
		t.Fatalf("valid envelope rejected: %v", err)
	}
	cases := map[string]struct {
		payload string
		want    ErrorCode
	}{
		"missing_field":      {`{"protocol":1,"type":"hello","request_id":"r-1","profile_id":"p","payload":{}}`, CodeInvalidFrame},
		"null_field":         {`{"protocol":1,"type":"hello","request_id":null,"profile_id":"p","projection_revision":0,"payload":{}}`, CodeInvalidFrame},
		"wrong_type":         {`{"protocol":"1","type":"hello","request_id":"r-1","profile_id":"p","projection_revision":0,"payload":{}}`, CodeInvalidFrame},
		"duplicate_key":      {`{"protocol":1,"protocol":1,"type":"hello","request_id":"r-1","profile_id":"p","projection_revision":0,"payload":{}}`, CodeInvalidFrame},
		"unknown_field":      {`{"protocol":1,"type":"hello","request_id":"r-1","profile_id":"p","projection_revision":0,"payload":{},"extra":1}`, CodeInvalidFrame},
		"trailing_bytes":     {valid + `{}`, CodeInvalidFrame},
		"trailing_garbage":   {valid + `x`, CodeInvalidFrame},
		"not_an_object":      {`[]`, CodeInvalidFrame},
		"top_level_null":     {`null`, CodeInvalidFrame},
		"unknown_type":       {`{"protocol":1,"type":"teleport","request_id":"r-1","profile_id":"p","projection_revision":0,"payload":{}}`, CodeInvalidFrame},
		"empty_request_id":   {`{"protocol":1,"type":"hello","request_id":"","profile_id":"p","projection_revision":0,"payload":{}}`, CodeInvalidFrame},
		"empty_profile_id":   {`{"protocol":1,"type":"hello","request_id":"r-1","profile_id":"","projection_revision":0,"payload":{}}`, CodeInvalidFrame},
		"negative_revision":  {`{"protocol":1,"type":"hello","request_id":"r-1","profile_id":"p","projection_revision":-1,"payload":{}}`, CodeInvalidFrame},
		"float_revision":     {`{"protocol":1,"type":"hello","request_id":"r-1","profile_id":"p","projection_revision":1.5,"payload":{}}`, CodeInvalidFrame},
		"payload_not_object": {`{"protocol":1,"type":"hello","request_id":"r-1","profile_id":"p","projection_revision":0,"payload":[]}`, CodeInvalidFrame},
		"unsupported_major":  {`{"protocol":2,"type":"hello","request_id":"r-1","profile_id":"p","projection_revision":0,"payload":{}}`, CodeProtocolMismatch},
	}
	for name, testCase := range cases {
		t.Run(name, func(t *testing.T) {
			_, failure := DecodeEnvelope([]byte(testCase.payload), DefaultLimits())
			if failure == nil {
				t.Fatalf("accepted a malformed envelope")
			}
			if failure.Code != testCase.want {
				t.Fatalf("code = %s, want %s", failure.Code, testCase.want)
			}
			if strings.Contains(failure.Error(), testCase.payload) {
				t.Fatal("the error text carried frame content")
			}
		})
	}
	long := strings.Repeat("r", MaxRequestIDBytes+1)
	if _, failure := DecodeEnvelope([]byte(fmt.Sprintf(`{"protocol":1,"type":"hello","request_id":%q,"profile_id":%q,"projection_revision":0,"payload":{}}`, long, testProfile)), DefaultLimits()); failure == nil || failure.Code != CodeInvalidFrame {
		t.Fatalf("an oversized request identity was accepted: %v", failure)
	}
	oversized := make([]byte, DefaultLimits().MaxPayloadBytes+1)
	if _, failure := DecodeEnvelope(oversized, DefaultLimits()); failure == nil || failure.Code != CodePayloadLimit {
		t.Fatalf("an oversized payload was accepted: %v", failure)
	}
}

func TestProtocol_EnvelopeRoundTripIsCanonical(t *testing.T) {
	envelope := Envelope{
		Protocol:           CurrentVersion,
		Type:               TypeQuery,
		RequestID:          "r-2",
		ProfileID:          testProfile,
		ProjectionRevision: 7,
		Payload:            json.RawMessage(`{"query":"go","result_limit":5,"current_window_id":1}`),
	}
	encoded, failure := EncodeEnvelope(envelope, DefaultLimits())
	if failure != nil {
		t.Fatal(failure)
	}
	decoded, failure := DecodeEnvelope(encoded, DefaultLimits())
	if failure != nil {
		t.Fatal(failure)
	}
	if decoded.Type != TypeQuery || decoded.ProjectionRevision != 7 || decoded.ProfileID != testProfile {
		t.Fatalf("round trip lost fields: %+v", decoded)
	}
	again, failure := EncodeEnvelope(decoded, DefaultLimits())
	if failure != nil {
		t.Fatal(failure)
	}
	if string(again) != string(encoded) {
		t.Fatalf("encoding is not canonical:\n%s\n%s", encoded, again)
	}
}

func TestProtocol_HandshakeBindsProfileAndRejectsDataFirst(t *testing.T) {
	out := &recorder{}
	session := newSession(t, &fakeSync{}, &fakeQuery{})
	first := frame(t, TypeSnapshot, "r-9", 1, snapshotPayload(1))
	if _, err := session.Negotiate(context.Background(), out, first); err == nil {
		t.Fatal("a snapshot before hello was accepted")
	}
	mustBind(t, session, out)
	acknowledged := out.envelopes(t)
	if len(acknowledged) != 1 || acknowledged[0].Type != TypeHelloAck {
		t.Fatalf("handshake frames = %+v", acknowledged)
	}
	var ack HelloAck
	if err := json.Unmarshal(acknowledged[0].Payload, &ack); err != nil {
		t.Fatal(err)
	}
	if ack.Protocol != CurrentVersion || ack.HostVersion != "0.1.0" || ack.RankingModelVersion != "rank-1" || ack.Limits.MaxResults != MaxResults {
		t.Fatalf("hello_ack payload = %+v", ack)
	}
	if strings.Contains(string(acknowledged[0].Payload), testProfile) {
		t.Fatal("hello_ack repeated the profile identity value")
	}
	if err := session.Accept(context.Background(), out, frame(t, TypeHello, "r-2", 0, `{"extension_version":"1.0.0","browser_family":"chrome","browser_version":"126.0","context_kind":"normal","capabilities":["query"],"supported_protocol_versions":[1]}`)); err == nil {
		t.Fatal("a second hello on a bound connection was accepted")
	}
}

func TestProtocol_HandshakeFailsClosedOnIncompatibleClient(t *testing.T) {
	out := &recorder{}
	session := newSession(t, &fakeSync{}, &fakeQuery{})
	incompatible := frame(t, TypeHello, "r-1", 0, `{"extension_version":"1.0.0","browser_family":"chrome","browser_version":"126.0","context_kind":"normal","capabilities":["query"],"supported_protocols":[9]}`)
	if _, err := session.Negotiate(context.Background(), out, incompatible); err == nil {
		t.Fatal("an incompatible client was acknowledged")
	}
	if session.State() != StateIncompatible {
		t.Fatalf("state after an incompatible hello = %s", session.State())
	}
	sync := &fakeSync{}
	if sync.snapshots != 0 {
		t.Fatal("an incompatible handshake touched the projection")
	}
	if err := session.Accept(context.Background(), out, frame(t, TypeSnapshot, "r-2", 1, snapshotPayload(1))); err == nil {
		t.Fatal("an incompatible session accepted tab data")
	}
}

func TestProtocol_OrderingSnapshotIsAuthorityAndSyncs(t *testing.T) {
	syncOwner := &fakeSync{}
	out := &recorder{}
	session := newSession(t, syncOwner, &fakeQuery{})
	mustBind(t, session, out)
	out.reset()
	if err := session.Accept(context.Background(), out, frame(t, TypeSnapshot, "r-3", 1, snapshotPayload(3))); err != nil {
		t.Fatalf("snapshot: %v", err)
	}
	frames := out.envelopes(t)
	if len(frames) != 1 || frames[0].Type != TypeSyncAck {
		t.Fatalf("snapshot frames = %+v", frames)
	}
	var ack SyncAck
	if err := json.Unmarshal(frames[0].Payload, &ack); err != nil {
		t.Fatal(err)
	}
	if ack.AcceptedRevision != 1 || ack.AcceptedSequence != 1 {
		t.Fatalf("sync_ack = %+v", ack)
	}
	if session.State() != StateReady {
		t.Fatalf("state after a snapshot = %s", session.State())
	}
	if syncOwner.snapshots != 1 {
		t.Fatalf("snapshot applications = %d", syncOwner.snapshots)
	}
}

func TestProtocol_OrderingDeltaGapsFailClosed(t *testing.T) {
	syncOwner := &fakeSync{}
	out := &recorder{}
	session := newSession(t, syncOwner, &fakeQuery{})
	mustBind(t, session, out)
	if err := session.Accept(context.Background(), out, frame(t, TypeDelta, "r-4", 1, `{"base_revision":0,"sequence_start":2,"sequence_end":2,"operations":[{"operation":"remove","tab_id":11}]}`)); err == nil {
		t.Fatal("a delta before a snapshot was accepted")
	}
	if err := session.Accept(context.Background(), out, frame(t, TypeSnapshot, "r-5", 1, snapshotPayload(3))); err != nil {
		t.Fatal(err)
	}
	out.reset()
	if err := session.Accept(context.Background(), out, frame(t, TypeDelta, "r-6", 2, `{"base_revision":1,"sequence_start":9,"sequence_end":9,"operations":[{"operation":"remove","tab_id":11}]}`)); err == nil {
		t.Fatal("a delta with a sequence gap was accepted")
	}
	frames := out.envelopes(t)
	if len(frames) < 2 {
		t.Fatalf("gap frames = %+v", frames)
	}
	if frames[0].Type != TypeResyncRequired {
		t.Fatalf("first gap frame = %s", frames[0].Type)
	}
	var resync ResyncRequired
	if err := json.Unmarshal(frames[0].Payload, &resync); err != nil {
		t.Fatal(err)
	}
	if !resync.Reason.KnownReason() || resync.ExpectedRevision != 1 || resync.ExpectedSequence != 1 {
		t.Fatalf("resync payload = %+v", resync)
	}
	last := frames[len(frames)-1]
	if last.Type != TypeError {
		t.Fatalf("last gap frame = %s", last.Type)
	}
	var failure ErrorPayload
	if err := json.Unmarshal(last.Payload, &failure); err != nil {
		t.Fatal(err)
	}
	if failure.Code != CodeRevisionMismatch || !failure.Retryable {
		t.Fatalf("gap error = %+v", failure)
	}
	if syncOwner.deltas != 0 {
		t.Fatal("a gapped delta reached the projection owner")
	}
	out.reset()
	out.reset()
	unknown := `{"base_revision":1,"sequence_start":2,"sequence_end":2,"operations":[{"operation":"archive","tab_id":11}]}`
	if err := session.Accept(context.Background(), out, frame(t, TypeDelta, "r-7", 2, unknown)); err == nil {
		t.Fatal("an unknown delta operation was accepted")
	}
	if code := lastErrorCode(t, out); code != CodeInvalidFrame {
		t.Fatalf("unknown operation code = %s", code)
	}
	if syncOwner.deltas != 0 {
		t.Fatal("an invalid delta reached the projection owner")
	}
}

func TestProtocol_OrderingLedgerReplaysAndFailsClosed(t *testing.T) {
	syncOwner := &fakeSync{}
	out := &recorder{}
	session := newSession(t, syncOwner, &fakeQuery{})
	mustBind(t, session, out)
	snapshot := frame(t, TypeSnapshot, "r-8", 1, snapshotPayload(2))
	if err := session.Accept(context.Background(), out, snapshot); err != nil {
		t.Fatal(err)
	}
	out.reset()
	if err := session.Accept(context.Background(), out, snapshot); err != nil {
		t.Fatal(err)
	}
	replays := out.envelopes(t)
	if len(replays) != 1 || replays[0].Type != TypeSyncAck {
		t.Fatalf("duplicate replay = %+v", replays)
	}
	if syncOwner.snapshots != 1 {
		t.Fatalf("a duplicate snapshot re-applied the projection: %d", syncOwner.snapshots)
	}
	out.reset()
	changed := frame(t, TypeSnapshot, "r-8", 1, snapshotPayload(3))
	if err := session.Accept(context.Background(), out, changed); err == nil {
		t.Fatal("the same request identity with different bytes was accepted")
	}
	if session.State() != StateShutdown {
		t.Fatalf("state after a conflicting replay = %s", session.State())
	}
	if syncOwner.snapshots != 1 {
		t.Fatalf("a conflicting replay applied the projection: %d", syncOwner.snapshots)
	}
}

func TestProtocol_LimitsQueryBoundsAndRevisionMatrix(t *testing.T) {
	syncOwner := &fakeSync{}
	out := &recorder{}
	session := newSession(t, syncOwner, &fakeQuery{})
	mustBind(t, session, out)
	if err := session.Accept(context.Background(), out, frame(t, TypeQuery, "r-10", 1, `{"query":"go","result_limit":5,"current_window_id":1}`)); err == nil {
		t.Fatal("a query before a snapshot was accepted")
	}
	if err := session.Accept(context.Background(), out, frame(t, TypeSnapshot, "r-11", 1, snapshotPayload(2))); err != nil {
		t.Fatal(err)
	}
	out.reset()
	if err := session.Accept(context.Background(), out, frame(t, TypeQuery, "r-12", 99, `{"query":"go","result_limit":5,"current_window_id":1}`)); err == nil {
		t.Fatal("a query against an unknown revision was accepted")
	}
	if code := lastErrorCode(t, out); code != CodeRevisionMismatch {
		t.Fatalf("unknown revision code = %s", code)
	}
	out.reset()
	if err := session.Accept(context.Background(), out, frame(t, TypeQuery, "r-13", 1, `{"query":"go","result_limit":51,"current_window_id":1}`)); err == nil {
		t.Fatal("an oversized result limit was accepted")
	}
	if code := lastErrorCode(t, out); code != CodePayloadLimit {
		t.Fatalf("result limit code = %s", code)
	}
	out.reset()
	long := strings.Repeat("g", MaxQueryScalars+1)
	if err := session.Accept(context.Background(), out, frame(t, TypeQuery, "r-14", 1, fmt.Sprintf(`{"query":%q,"result_limit":5,"current_window_id":1}`, long))); err == nil {
		t.Fatal("an oversized query was accepted")
	}
	if code := lastErrorCode(t, out); code != CodePayloadLimit {
		t.Fatalf("query length code = %s", code)
	}
}

func TestProtocol_TimeoutQueryBudgetIsBounded(t *testing.T) {
	syncOwner := &fakeSync{}
	out := &recorder{}
	slow := &fakeQuery{delay: 200 * time.Millisecond}
	config := testConfig()
	config.Budgets.QueryHostBudget = 30 * time.Millisecond
	session, err := NewSession(config, syncOwner, slow, nil)
	if err != nil {
		t.Fatal(err)
	}
	mustBind(t, session, out)
	if err := session.Accept(context.Background(), out, frame(t, TypeSnapshot, "r-30", 1, snapshotPayload(2))); err != nil {
		t.Fatal(err)
	}
	out.reset()
	started := time.Now()
	if err := session.Accept(context.Background(), out, frame(t, TypeQuery, "r-31", 1, `{"query":"go","result_limit":5,"current_window_id":1}`)); err == nil {
		t.Fatal("a query past its host budget was answered")
	}
	if elapsed := time.Since(started); elapsed > 150*time.Millisecond {
		t.Fatalf("the query host budget was not enforced: %s", elapsed)
	}
	if code := lastErrorCode(t, out); code != CodeQueryTimeout {
		t.Fatalf("timeout code = %s", code)
	}
	if len(out.envelopes(t)) != 1 {
		t.Fatal("a timed out query returned a partial result")
	}
}

func TestProtocol_LimitsRebuildingAndDegradedPersistence(t *testing.T) {
	syncOwner := &fakeSync{}
	out := &recorder{}
	session := newSession(t, syncOwner, &fakeQuery{})
	session.SetPersistence(runtime.PersistenceDegraded)
	mustBind(t, session, out)
	if err := session.Accept(context.Background(), out, frame(t, TypeSnapshot, "r-17", 1, snapshotPayload(2))); err != nil {
		t.Fatal(err)
	}
	out.reset()
	if err := session.Accept(context.Background(), out, frame(t, TypeQuery, "r-18", 1, `{"query":"go","result_limit":5,"current_window_id":1}`)); err != nil {
		t.Fatal(err)
	}
	frames := out.envelopes(t)
	if len(frames) != 1 || frames[0].Type != TypeQueryResult {
		t.Fatalf("query frames = %+v", frames)
	}
	var result QueryResult
	if err := json.Unmarshal(frames[0].Payload, &result); err != nil {
		t.Fatal(err)
	}
	if result.Counters.DegradedCode != CodePersistenceDegraded {
		t.Fatalf("degraded persistence annotation = %q", result.Counters.DegradedCode)
	}
	if len(result.Results) != 1 {
		t.Fatalf("a degraded store removed lexical results: %+v", result.Results)
	}
	session.SetRebuilding(true)
	out.reset()
	if err := session.Accept(context.Background(), out, frame(t, TypeQuery, "r-19", 1, `{"query":"go","result_limit":5,"current_window_id":1}`)); err == nil {
		t.Fatal("a query while rebuilding was answered")
	}
	if code := lastErrorCode(t, out); code != CodeIndexRebuilding {
		t.Fatalf("rebuilding code = %s", code)
	}
}

func TestProtocol_HandshakeProfileMismatchNeverMutates(t *testing.T) {
	syncOwner := &fakeSync{}
	out := &recorder{}
	session := newSession(t, syncOwner, &fakeQuery{})
	mustBind(t, session, out)
	other := "223e4567-e89b-42d3-a456-426614174000"
	encoded, err := EncodeEnvelope(Envelope{
		Protocol: CurrentVersion, Type: TypeSnapshot, RequestID: "r-20",
		ProfileID: other, ProjectionRevision: 1, Payload: json.RawMessage(snapshotPayload(1)),
	}, DefaultLimits())
	if err != nil {
		t.Fatal(err)
	}
	if err := session.Accept(context.Background(), out, encoded); err == nil {
		t.Fatal("a cross-profile frame was accepted")
	}
	if code := lastErrorCode(t, out); code != CodeProfileMismatch {
		t.Fatalf("cross-profile code = %s", code)
	}
	if syncOwner.snapshots != 0 {
		t.Fatal("a cross-profile frame reached the projection owner")
	}
	crossRecord := strings.Replace(snapshotPayload(1), testProfile, other, 1)
	if err := session.Accept(context.Background(), out, frame(t, TypeSnapshot, "r-21", 1, crossRecord)); err == nil {
		t.Fatal("a cross-profile record was accepted")
	}
	if syncOwner.snapshots != 0 {
		t.Fatal("a cross-profile record reached the projection owner")
	}
}

func TestProtocol_DisconnectHealthAndActivationStatus(t *testing.T) {
	syncOwner := &fakeSync{}
	out := &recorder{}
	session := newSession(t, syncOwner, &fakeQuery{})
	session.SetPersistence(runtime.PersistenceReady)
	mustBind(t, session, out)
	if err := session.Accept(context.Background(), out, frame(t, TypeSnapshot, "r-22", 1, snapshotPayload(2))); err != nil {
		t.Fatal(err)
	}
	out.reset()
	if err := session.Accept(context.Background(), out, frame(t, TypeHealth, "r-23", 1, `{}`)); err != nil {
		t.Fatal(err)
	}
	frames := out.envelopes(t)
	if len(frames) != 1 || frames[0].Type != TypeHealthResult {
		t.Fatalf("health frames = %+v", frames)
	}
	encoded, err := json.Marshal(frames[0].Payload)
	if err != nil {
		t.Fatal(err)
	}
	for _, forbidden := range []string{"title", "url", "query", "token", "cookie", "domain_display", "tab_id"} {
		if strings.Contains(string(encoded), forbidden) {
			t.Fatalf("health_result leaked %q: %s", forbidden, encoded)
		}
	}
	var health HealthResult
	if err := json.Unmarshal(frames[0].Payload, &health); err != nil {
		t.Fatal(err)
	}
	if health.HostState != string(StateReady) || health.StorageState != string(runtime.PersistenceReady) || health.Availability != string(runtime.AvailabilityHealthy) {
		t.Fatalf("health_result = %+v", health)
	}
	out.reset()
	if err := session.Accept(context.Background(), out, frame(t, TypeActivationObserved, "r-24", 1, `{"result_reference":{"tab_id":10,"projection_epoch":"`+testEpoch+`","projection_revision":1,"result_id":"res-1"},"tab_id":10,"projection_revision":1,"outcome":"activated"}`)); err != nil {
		t.Fatal(err)
	}
	frames = out.envelopes(t)
	if len(frames) != 1 || frames[0].Type != TypeActivationAck {
		t.Fatalf("activation frames = %+v", frames)
	}
	out.reset()
	failed := `{"result_reference":{"tab_id":10,"projection_epoch":"` + testEpoch + `","projection_revision":99,"result_id":"res-1"},"tab_id":10,"projection_revision":99,"error_code":"REVISION_MISMATCH"}`
	if err := session.Accept(context.Background(), out, frame(t, TypeActivationFailed, "r-25", 99, failed)); err == nil {
		t.Fatal("an activation report against a stale revision was accepted")
	}
	if code := lastErrorCode(t, out); code != CodeRevisionMismatch {
		t.Fatalf("stale activation code = %s", code)
	}
}

func TestProtocol_DisconnectDirectionAndPostDisconnectRules(t *testing.T) {
	out := &recorder{}
	session := newSession(t, &fakeSync{}, &fakeQuery{})
	mustBind(t, session, out)
	if err := session.Accept(context.Background(), out, frame(t, TypeQueryResult, "r-26", 0, `{"results":[]}`)); err == nil {
		t.Fatal("a host-to-extension message from the extension was accepted")
	}
	if code := lastErrorCode(t, out); code != CodeInvalidFrame {
		t.Fatalf("direction violation code = %s", code)
	}
	session.Disconnect()
	out.reset()
	if err := session.Accept(context.Background(), out, frame(t, TypeHealth, "r-27", 0, `{}`)); err == nil {
		t.Fatal("a frame after disconnect was accepted")
	}
	if code := lastErrorCode(t, out); code != CodeHostShutdown {
		t.Fatalf("post-disconnect code = %s", code)
	}
	if err := session.Shutdown(context.Background()); err != nil {
		t.Fatal(err)
	}
	if err := session.Shutdown(context.Background()); err != nil {
		t.Fatalf("shutdown is not idempotent: %v", err)
	}
}

func TestProtocol_DisconnectNotifyMapsRuntimeFailures(t *testing.T) {
	out := &recorder{}
	session := newSession(t, &fakeSync{}, &fakeQuery{})
	mustBind(t, session, out)
	out.reset()
	notification := runtime.Notification{
		Reason:       runtime.NotificationFailure,
		State:        runtime.StateRecovering,
		Availability: runtime.AvailabilityRecovering,
		Failure:      runtime.FailureTransportBrokenPipe,
		Retryable:    true,
		Snapshot:     runtime.HealthSnapshot{SessionID: strings.Repeat("a", 32)},
	}
	if err := session.Notify(context.Background(), out, notification); err != nil {
		t.Fatal(err)
	}
	frames := out.envelopes(t)
	if len(frames) != 1 || frames[0].Type != TypeError {
		t.Fatalf("notify frames = %+v", frames)
	}
	var payload ErrorPayload
	if err := json.Unmarshal(frames[0].Payload, &payload); err != nil {
		t.Fatal(err)
	}
	if payload.Code != CodeHostShutdown || !payload.Retryable {
		t.Fatalf("notify payload = %+v", payload)
	}
	if !payload.Code.KnownCode() {
		t.Fatalf("code %s is outside the exact error enum", payload.Code)
	}
}

func TestProtocol_LimitsAndErrorEnumAreClosed(t *testing.T) {
	if err := DefaultLimits().Validate(); err != nil {
		t.Fatalf("default limits rejected: %v", err)
	}
	if err := DefaultBudgets().Validate(); err != nil {
		t.Fatalf("default budgets rejected: %v", err)
	}
	invalid := DefaultLimits()
	invalid.MaxResults = MaxResults + 1
	if err := invalid.Validate(); err == nil {
		t.Fatal("an over-bound result limit was accepted")
	}
	invalid = DefaultLimits()
	invalid.MaxPayloadBytes = 0
	if err := invalid.Validate(); err == nil {
		t.Fatal("an unbounded payload limit was accepted")
	}
	if CodePayloadLimit.Retryable() || CodeInvalidFrame.Retryable() || CodeProfileMismatch.Retryable() {
		t.Fatal("a contract violation was marked retryable")
	}
	if !CodeSnapshotRequired.Retryable() || !CodeRevisionMismatch.Retryable() || !CodeIndexRebuilding.Retryable() {
		t.Fatal("a recoverable class was marked terminal")
	}
	known := []ErrorCode{
		CodeInvalidFrame, CodeProtocolMismatch, CodeProfileMismatch, CodePayloadLimit,
		CodeSnapshotRequired, CodeRevisionMismatch, CodeIndexRebuilding, CodeQueryTimeout,
		CodePersistenceDegraded, CodeHostShutdown, CodeInternalFailure,
	}
	if len(known) != 11 {
		t.Fatalf("error enum size = %d", len(known))
	}
	for _, code := range known {
		if !code.KnownCode() {
			t.Fatalf("%s is not in the enum", code)
		}
	}
	if ErrorCode("SOMETHING_ELSE").KnownCode() {
		t.Fatal("an unknown code was accepted")
	}
	for _, messageType := range []MessageType{TypeHello, TypeHelloAck, TypeSnapshot, TypeDelta, TypeSyncAck,
		TypeResyncRequired, TypeQuery, TypeQueryResult, TypeActivationObserved, TypeActivationFailed,
		TypeActivationAck, TypeHealth, TypeHealthResult, TypeError} {
		if !messageType.KnownType() || messageType.Direction() == "" {
			t.Fatalf("%s is not in the message enum", messageType)
		}
	}
}

func TestProtocol_TimeoutSyncApplyStaysBounded(t *testing.T) {
	slow := &fakeSync{delay: 500 * time.Millisecond}
	out := &recorder{}
	config := testConfig()
	config.Budgets.SyncApply = 50 * time.Millisecond
	session, err := NewSession(config, slow, &fakeQuery{}, nil)
	if err != nil {
		t.Fatal(err)
	}
	mustBind(t, session, out)
	out.reset()
	started := time.Now()
	if err := session.Accept(context.Background(), out, frame(t, TypeSnapshot, "r-28", 1, snapshotPayload(1))); err == nil {
		t.Fatal("a snapshot past its apply budget was accepted")
	}
	if elapsed := time.Since(started); elapsed > 400*time.Millisecond {
		t.Fatalf("the apply budget was not enforced: %s", elapsed)
	}
	if code := lastErrorCode(t, out); code != CodeSnapshotRequired {
		t.Fatalf("apply timeout code = %s", code)
	}
}

func lastErrorCode(t *testing.T, out *recorder) ErrorCode {
	t.Helper()
	frames := out.envelopes(t)
	for index := len(frames) - 1; index >= 0; index-- {
		if frames[index].Type != TypeError {
			continue
		}
		var payload ErrorPayload
		if err := json.Unmarshal(frames[index].Payload, &payload); err != nil {
			t.Fatal(err)
		}
		return payload.Code
	}
	t.Fatalf("no error frame among %d frames", len(frames))
	return ""
}

var _ SyncOwner = (*fakeSync)(nil)
var _ QueryOwner = (*fakeQuery)(nil)
var _ runtime.ProtocolHandler = (*Session)(nil)
var _ domain.ProfileID
