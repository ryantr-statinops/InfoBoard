package runtime

import (
	"bytes"
	"context"
	"encoding/binary"
	"errors"
	"io"
	goruntime "runtime"
	"sync"
	"syscall"
	"testing"
	"time"

	"github.com/ryantr-statinops/InfoBoard/host/internal/domain"
)

type frameRecorder struct {
	mu         sync.Mutex
	inflight   int
	units      [][]byte
	bad        int
	block      chan struct{}
	blockAfter int
	writes     int
}

func (rec *frameRecorder) Write(chunk []byte) (int, error) {
	rec.mu.Lock()
	rec.inflight++
	if rec.inflight > 1 {
		rec.bad++
	}
	rec.units = append(rec.units, append([]byte(nil), chunk...))
	rec.writes++
	blocked := rec.block
	if rec.writes <= rec.blockAfter {
		blocked = nil
	}
	rec.mu.Unlock()
	if blocked != nil {
		<-blocked
	}
	rec.mu.Lock()
	rec.inflight--
	rec.mu.Unlock()
	return len(chunk), nil
}

func frame(payload string) []byte {
	body := []byte(payload)
	out := make([]byte, 4+len(body))
	binary.LittleEndian.PutUint32(out[:4], uint32(len(body)))
	copy(out[4:], body)
	return out
}

type fakeProtocol struct {
	binding      Binding
	negotiateErr error
	accept       func(ctx context.Context, channel FrameChannel, payload []byte) error
	mu           sync.Mutex
	notified     []Notification
	frames       []string
}

func (p *fakeProtocol) Negotiate(ctx context.Context, channel FrameChannel, payload []byte) (Binding, error) {
	if p.negotiateErr != nil {
		return Binding{}, p.negotiateErr
	}
	p.mu.Lock()
	p.frames = append(p.frames, string(payload))
	p.mu.Unlock()
	return p.binding, channel.WriteFrame(ctx, []byte("hello_ack"))
}

func (p *fakeProtocol) Accept(ctx context.Context, channel FrameChannel, payload []byte) error {
	if p.accept != nil {
		return p.accept(ctx, channel, payload)
	}
	p.mu.Lock()
	p.frames = append(p.frames, string(payload))
	p.mu.Unlock()
	return nil
}

func (p *fakeProtocol) Notify(ctx context.Context, channel FrameChannel, notification Notification) error {
	p.mu.Lock()
	p.notified = append(p.notified, notification)
	p.mu.Unlock()
	if channel != nil {
		return channel.WriteFrame(ctx, []byte("status:"+string(notification.State)))
	}
	return nil
}

func (p *fakeProtocol) Shutdown(context.Context) error { return nil }

type fakeProjection struct {
	status ProjectionStatus
	err    error
}

func (p fakeProjection) Synchronize(context.Context, Target) (ProjectionStatus, error) {
	return p.status, p.err
}

type fakePersistence struct {
	prepareStatus PersistenceStatus
	prepareErr    error
	flushErr      error
	mu            sync.Mutex
	flushed       int
	closed        int
}

func (p *fakePersistence) Prepare(context.Context) (PersistenceStatus, error) {
	return p.prepareStatus, p.prepareErr
}

func (p *fakePersistence) Flush(context.Context) error {
	p.mu.Lock()
	p.flushed++
	p.mu.Unlock()
	return p.flushErr
}

func (p *fakePersistence) Close() error {
	p.mu.Lock()
	p.closed++
	p.mu.Unlock()
	return nil
}

type blockingReader struct {
	release chan struct{}
	mu      sync.Mutex
	parked  int
}

func (r *blockingReader) Read([]byte) (int, error) {
	r.mu.Lock()
	r.parked++
	r.mu.Unlock()
	<-r.release
	return 0, io.EOF
}

func (r *blockingReader) holding() int {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.parked
}

func classOf(err error) FailureClass {
	class, _ := ClassifyFailure(err)
	return class
}

func concat(parts ...[]byte) []byte {
	var out []byte
	for _, part := range parts {
		out = append(out, part...)
	}
	return out
}

const testOrigin = "chrome-extension://abcdefghijklmnopabcdefghijklmnop"
const testProfile = "123e4567-e89b-42d3-a456-426614174000"

func testLimits() Limits {
	limits := DefaultLimits()
	limits.HandshakeTimeout = 60 * time.Millisecond
	limits.SynchronizationTimeout = 60 * time.Millisecond
	limits.RequestTimeout = 60 * time.Millisecond
	limits.ShutdownTimeout = 80 * time.Millisecond
	limits.MaxConcurrentRequests = 4
	return limits
}

func validBinding() Binding {
	return Binding{ProfileID: domain.ProfileID(testProfile), ContextKind: domain.ContextNormal, ProtocolVersion: 1, HostVersion: "test"}
}

func readyProjection() fakeProjection {
	return fakeProjection{status: ProjectionStatus{Known: true, Epoch: "123e4567-e89b-42d3-a456-426614174001", Revision: 1, Count: 3, Ready: true}}
}

func TestRuntimeBootstrapRejectsUnsupportedInvocation(t *testing.T) {
	if _, err := ParseInvocation(nil); classOf(err) != FailureConfiguration {
		t.Fatalf("missing origin accepted: %v", err)
	}
	if _, err := ParseInvocation([]string{"not-an-origin"}); err == nil {
		t.Fatal("invalid origin accepted")
	}
	if _, err := ParseInvocation([]string{testOrigin, "--nope"}); err == nil {
		t.Fatal("unknown argument accepted")
	}
	if _, err := ParseInvocation([]string{testOrigin, "--parent-window=42"}); err != nil {
		t.Fatalf("documented parent window argument rejected: %v", err)
	}
	if err := (Limits{MaxFrameBytes: 1 << 30}).Validate(); err == nil {
		t.Fatal("oversized limits accepted")
	}
	config := DefaultConfig("test", nil, testLimits())
	config.AllowedOrigins = nil
	invocation := Invocation{Origin: testOrigin}
	if _, err := Bootstrap(config, invocation, bytes.NewReader(nil), io.Discard, Dependencies{}); classOf(err) != FailureRegistration {
		t.Fatalf("unregistered origin accepted: %v", err)
	}
	config.AllowedOrigins = []string{"chrome-extension://aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"}
	if _, err := Bootstrap(config, invocation, bytes.NewReader(nil), io.Discard, Dependencies{}); classOf(err) != FailureRegistration {
		t.Fatalf("unregistered origin accepted: %v", err)
	}
}

func TestRuntimeHealthySessionReachesReadyAndClosesOnEOF(t *testing.T) {
	var observed []HealthSnapshot
	input := bytes.NewReader(concat(frame("hello"), frame("query-a"), frame("query-b")))
	recorder := &frameRecorder{}
	protocol := &fakeProtocol{binding: validBinding(), accept: func(ctx context.Context, channel FrameChannel, payload []byte) error {
		return channel.WriteFrame(ctx, []byte("result:"+string(payload)))
	}}
	store := &fakePersistence{prepareStatus: PersistenceReady}
	config := DefaultConfig("test", NewBoundedSink(io.Discard, 4096), testLimits())
	config.AllowedOrigins = []string{testOrigin}
	session, err := Bootstrap(config, Invocation{Origin: testOrigin}, input, recorder, Dependencies{
		Protocol:    protocol,
		Projection:  readyProjection(),
		Persistence: store,
	})
	if err != nil {
		t.Fatal(err)
	}
	session.Subscribe(HealthObserverFunc(func(snapshot HealthSnapshot) { observed = append(observed, snapshot) }))
	outcome := session.Run(context.Background())
	if outcome.ExitCode != ExitClean || !outcome.Clean {
		t.Fatalf("clean EOF outcome=%+v", outcome)
	}
	if session.Health().State != StateClosing || session.Health().Availability != AvailabilityUnavailable {
		t.Fatalf("terminal state=%+v", session.Health())
	}
	sawReady := false
	for _, snapshot := range observed {
		if snapshot.State == StateReady {
			sawReady = true
			if snapshot.Availability != AvailabilityHealthy {
				t.Fatalf("ready availability=%s", snapshot.Availability)
			}
			if snapshot.ProjectionRevision != 1 || snapshot.ProjectionCount != 3 || !snapshot.ProjectionKnown {
				t.Fatalf("projection metadata=%+v", snapshot)
			}
		}
	}
	if !sawReady {
		t.Fatal("never published healthy")
	}
	recorder.mu.Lock()
	units, bad := len(recorder.units), recorder.bad
	recorder.mu.Unlock()
	if bad != 0 {
		t.Fatalf("interleaved writes=%d", bad)
	}
	if units == 0 {
		t.Fatal("no frame written")
	}
	for _, unit := range recorder.units {
		if len(unit) < 4 || int(binary.LittleEndian.Uint32(unit[:4])) != len(unit)-4 {
			t.Fatalf("non-frame bytes on stdout: %q", unit)
		}
	}
	if store.flushed != 1 || store.closed != 1 {
		t.Fatalf("persistence flush=%d close=%d", store.flushed, store.closed)
	}
	for _, entry := range session.Diagnostics().Entries() {
		if entry.Event == "" || entry.SessionID == "" || entry.Persistence == "" {
			t.Fatalf("incomplete diagnostic=%+v", entry)
		}
	}
}

func TestRuntimeRejectsOversizedFrameAndMalformedFrames(t *testing.T) {
	oversized := make([]byte, 4)
	binary.LittleEndian.PutUint32(oversized, 1<<20+1)
	input := bytes.NewReader(concat(oversized, frame("hello")))
	recorder := &frameRecorder{}
	config := DefaultConfig("test", nil, testLimits())
	config.AllowedOrigins = []string{testOrigin}
	session, err := Bootstrap(config, Invocation{Origin: testOrigin}, input, recorder, Dependencies{
		Protocol:   &fakeProtocol{binding: validBinding()},
		Projection: readyProjection(),
	})
	if err != nil {
		t.Fatal(err)
	}
	outcome := session.Run(context.Background())
	if outcome.Failure != FailureFrameTooLarge || outcome.ExitCode != ExitStartupFailure {
		t.Fatalf("oversized frame outcome=%+v", outcome)
	}
	if outcome.Retryable {
		t.Fatal("oversized frame must not be retryable")
	}
	partial := []byte{3, 0, 0}
	session2, err := Bootstrap(config, Invocation{Origin: testOrigin}, bytes.NewReader(partial), recorder, Dependencies{})
	if err != nil {
		t.Fatal(err)
	}
	if outcome := session2.Run(context.Background()); outcome.Failure != FailureTransportTruncated {
		t.Fatalf("truncated header outcome=%+v", outcome)
	}
}

func TestRuntimeHandshakeTimeoutIsBoundedAndRetryable(t *testing.T) {
	reader := &blockingReader{release: make(chan struct{})}
	defer close(reader.release)
	config := DefaultConfig("test", nil, testLimits())
	config.AllowedOrigins = []string{testOrigin}
	session, err := Bootstrap(config, Invocation{Origin: testOrigin}, reader, io.Discard, Dependencies{})
	if err != nil {
		t.Fatal(err)
	}
	started := time.Now()
	outcome := session.Run(context.Background())
	elapsed := time.Since(started)
	if outcome.Failure != FailureHandshakeTimeout || !outcome.Retryable || outcome.ExitCode != ExitStartupFailure {
		t.Fatalf("handshake timeout outcome=%+v", outcome)
	}
	if elapsed > 2*time.Second {
		t.Fatalf("handshake budget unbounded: %s", elapsed)
	}
}

func TestRuntimeProfileAndProtocolMismatchFailClosed(t *testing.T) {
	config := DefaultConfig("test", nil, testLimits())
	config.AllowedOrigins = []string{testOrigin}
	bad := validBinding()
	bad.ProfileID = "not-a-profile"
	session, err := Bootstrap(config, Invocation{Origin: testOrigin}, bytes.NewReader(frame("hello")), io.Discard, Dependencies{Protocol: &fakeProtocol{binding: bad}})
	if err != nil {
		t.Fatal(err)
	}
	if outcome := session.Run(context.Background()); outcome.Failure != FailureProfileMismatch || outcome.ExitCode != ExitProtocolFailure {
		t.Fatalf("profile mismatch outcome=%+v", outcome)
	}
	badVersion := validBinding()
	badVersion.ProtocolVersion = 9
	session2, _ := Bootstrap(config, Invocation{Origin: testOrigin}, bytes.NewReader(frame("hello")), io.Discard, Dependencies{Protocol: &fakeProtocol{binding: badVersion}})
	if outcome := session2.Run(context.Background()); outcome.Failure != FailureProtocolIncompatible || outcome.ExitCode != ExitProtocolFailure {
		t.Fatalf("protocol mismatch outcome=%+v", outcome)
	}
	notReady := fakeProjection{status: ProjectionStatus{Known: true, Revision: 0}}
	session3, _ := Bootstrap(config, Invocation{Origin: testOrigin}, bytes.NewReader(frame("hello")), io.Discard, Dependencies{Protocol: &fakeProtocol{binding: validBinding()}, Projection: notReady})
	outcome3 := session3.Run(context.Background())
	if outcome3.Failure != FailureSnapshotRejected || !outcome3.Retryable {
		t.Fatalf("missing snapshot outcome=%+v", outcome3)
	}
	if state := session3.Health().State; state != StateClosing {
		t.Fatalf("unready session state=%s", state)
	}
}

func TestRuntimeConcurrentResponsesStaySerialized(t *testing.T) {
	input := bytes.NewReader(concat(frame("hello"), frame("a"), frame("b"), frame("c"), frame("d")))
	recorder := &frameRecorder{}
	protocol := &fakeProtocol{binding: validBinding(), accept: func(ctx context.Context, channel FrameChannel, payload []byte) error {
		time.Sleep(time.Millisecond)
		return channel.WriteFrame(ctx, []byte("result:"+string(payload)))
	}}
	config := DefaultConfig("test", nil, testLimits())
	config.AllowedOrigins = []string{testOrigin}
	session, err := Bootstrap(config, Invocation{Origin: testOrigin}, input, recorder, Dependencies{Protocol: protocol, Projection: readyProjection()})
	if err != nil {
		t.Fatal(err)
	}
	outcome := session.Run(context.Background())
	if outcome.ExitCode != ExitClean {
		t.Fatalf("outcome=%+v", outcome)
	}
	recorder.mu.Lock()
	defer recorder.mu.Unlock()
	if recorder.bad != 0 {
		t.Fatalf("interleaved writes=%d", recorder.bad)
	}
	if len(recorder.units) < 5 {
		t.Fatalf("frames written=%d", len(recorder.units))
	}
	for _, unit := range recorder.units {
		if len(unit) < 4 || int(binary.LittleEndian.Uint32(unit[:4])) != len(unit)-4 {
			t.Fatalf("non-frame bytes: %q", unit)
		}
	}
}

func TestRuntimeCloseIsIdempotentAndDrainIsBounded(t *testing.T) {
	recorder := &frameRecorder{block: make(chan struct{}), blockAfter: 1}
	input := bytes.NewReader(concat(frame("hello"), frame("a"), frame("b")))
	started := make(chan struct{})
	protocol := &fakeProtocol{binding: validBinding(), accept: func(ctx context.Context, channel FrameChannel, payload []byte) error {
		close(started)
		<-ctx.Done()
		return ctx.Err()
	}}
	config := DefaultConfig("test", nil, testLimits())
	config.AllowedOrigins = []string{testOrigin}
	session, err := Bootstrap(config, Invocation{Origin: testOrigin}, input, recorder, Dependencies{Protocol: protocol, Projection: readyProjection()})
	if err != nil {
		t.Fatal(err)
	}
	results := make(chan Outcome, 1)
	go func() { results <- session.Run(context.Background()) }()
	<-started
	first := session.Close(nil)
	second := session.Close(nil)
	close(recorder.block)
	outcome := <-results
	if first.ExitCode != second.ExitCode || outcome.ExitCode != first.ExitCode {
		t.Fatalf("close not idempotent: %+v %+v %+v", first, second, outcome)
	}
	if session.Health().AcceptsWork() {
		t.Fatal("finished session still accepts work")
	}
}

func TestRuntimeConcurrentWorkIsBounded(t *testing.T) {
	parts := [][]byte{frame("hello"), frame("a")}
	for i := 0; i < 64; i++ {
		parts = append(parts, frame("q"))
	}
	payload := concat(parts...)
	recorder := &frameRecorder{}
	release := make(chan struct{})
	var inFlight, peak int
	var mu sync.Mutex
	protocol := &fakeProtocol{binding: validBinding(), accept: func(ctx context.Context, channel FrameChannel, _ []byte) error {
		mu.Lock()
		inFlight++
		if inFlight > peak {
			peak = inFlight
		}
		mu.Unlock()
		select {
		case <-release:
		case <-ctx.Done():
		}
		mu.Lock()
		inFlight--
		mu.Unlock()
		return nil
	}}
	limits := testLimits()
	limits.MaxConcurrentRequests = 2
	limits.RequestTimeout = 30 * time.Millisecond
	config := DefaultConfig("test", nil, limits)
	config.AllowedOrigins = []string{testOrigin}
	session, err := Bootstrap(config, Invocation{Origin: testOrigin}, bytes.NewReader(payload), recorder, Dependencies{Protocol: protocol, Projection: readyProjection()})
	if err != nil {
		t.Fatal(err)
	}
	outcome := session.Run(context.Background())
	close(release)
	if outcome.ExitCode != ExitClean {
		t.Fatalf("outcome=%+v", outcome)
	}
	mu.Lock()
	defer mu.Unlock()
	if peak > limits.MaxConcurrentRequests {
		t.Fatalf("concurrency peak=%d exceeds cap", peak)
	}
	if session.Health().RequestsRejected == 0 {
		t.Fatal("saturated work was not rejected observably")
	}
}

func TestRuntimeCancellationStopsInFlightWork(t *testing.T) {
	input := bytes.NewReader(concat(frame("hello"), frame("a")))
	entered := make(chan struct{})
	observed := make(chan error, 1)
	protocol := &fakeProtocol{binding: validBinding(), accept: func(ctx context.Context, channel FrameChannel, _ []byte) error {
		select {
		case <-entered:
		default:
			close(entered)
		}
		<-ctx.Done()
		observed <- context.Cause(ctx)
		return nil
	}}
	config := DefaultConfig("test", nil, testLimits())
	config.AllowedOrigins = []string{testOrigin}
	session, err := Bootstrap(config, Invocation{Origin: testOrigin}, input, io.Discard, Dependencies{Protocol: protocol, Projection: readyProjection()})
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	results := make(chan Outcome, 1)
	go func() { results <- session.Run(ctx) }()
	<-entered
	cancel()
	outcome := <-results
	if outcome.ExitCode != ExitClean {
		t.Fatalf("cancelled outcome=%+v", outcome)
	}
	select {
	case cause := <-observed:
		if cause == nil {
			t.Fatal("request context not cancelled with a cause")
		}
	default:
		t.Fatal("in-flight work was not cancelled")
	}
}

func TestRuntimePersistenceDegradationKeepsRuntimeAvailable(t *testing.T) {
	store := &fakePersistence{prepareErr: context.DeadlineExceeded}
	config := DefaultConfig("test", nil, testLimits())
	config.AllowedOrigins = []string{testOrigin}
	session, err := Bootstrap(config, Invocation{Origin: testOrigin}, bytes.NewReader(frame("hello")), io.Discard, Dependencies{
		Protocol:    &fakeProtocol{binding: validBinding()},
		Projection:  readyProjection(),
		Persistence: store,
	})
	if err != nil {
		t.Fatal(err)
	}
	if outcome := session.Run(context.Background()); outcome.ExitCode != ExitClean {
		t.Fatalf("degraded persistence outcome=%+v", outcome)
	}
	if session.Health().Persistence != PersistenceDegraded {
		t.Fatalf("persistence status=%s", session.Health().Persistence)
	}
}

func TestRuntimeDiagnosticsStayRedacted(t *testing.T) {
	var lines bytes.Buffer
	config := DefaultConfig("test", NewBoundedSink(&lines, 8192), testLimits())
	config.AllowedOrigins = []string{testOrigin}
	session, err := Bootstrap(config, Invocation{Origin: testOrigin}, bytes.NewReader(frame("hello-secret-title")), io.Discard, Dependencies{Protocol: &fakeProtocol{binding: validBinding()}, Projection: readyProjection()})
	if err != nil {
		t.Fatal(err)
	}
	session.Run(context.Background())
	if bytes.Contains(lines.Bytes(), []byte("secret-title")) {
		t.Fatalf("frame content leaked into diagnostics: %s", lines.String())
	}
	if lines.Len() == 0 {
		t.Fatal("no diagnostics written")
	}
}

func TestRuntimeFrameWriterRejectsOversizedPayload(t *testing.T) {
	limits := testLimits()
	writer := newFrameWriter(io.Discard, limits)
	if err := writer.WriteFrame(context.Background(), make([]byte, limits.MaxFrameBytes+1)); classOf(err) != FailureFrameTooLarge {
		t.Fatalf("oversized payload accepted: %v", err)
	}
	if err := writer.WriteFrame(context.Background(), nil); classOf(err) != FailureTransportMalformed {
		t.Fatalf("empty payload accepted: %v", err)
	}
}

type failAfterReader struct {
	data []byte
	fail error
	done bool
}

func (reader *failAfterReader) Read(buffer []byte) (int, error) {
	if !reader.done && len(reader.data) > 0 {
		read := copy(buffer, reader.data)
		reader.data = reader.data[read:]
		return read, nil
	}
	reader.done = true
	return 0, reader.fail
}

func TestRuntimeBrokenPipeAfterReadyRecoversForNewConnection(t *testing.T) {
	input := &failAfterReader{data: frame("hello"), fail: syscall.EPIPE}
	config := DefaultConfig("test", nil, testLimits())
	config.AllowedOrigins = []string{testOrigin}
	protocol := &fakeProtocol{binding: validBinding()}
	session, err := Bootstrap(config, Invocation{Origin: testOrigin}, input, io.Discard, Dependencies{Protocol: protocol, Projection: readyProjection()})
	if err != nil {
		t.Fatal(err)
	}
	var observed []HealthSnapshot
	session.Subscribe(HealthObserverFunc(func(snapshot HealthSnapshot) { observed = append(observed, snapshot) }))
	results := make(chan Outcome, 1)
	go func() { results <- session.Run(context.Background()) }()

	deadline := time.Now().Add(2 * time.Second)
	for session.Health().State != StateRecovering && time.Now().Before(deadline) {
		time.Sleep(time.Millisecond)
	}
	recovering := session.Health()
	if recovering.State != StateRecovering {
		t.Fatalf("state after a broken pipe = %s, want recovering", recovering.State)
	}
	if recovering.Availability != AvailabilityRecovering || recovering.Failure != FailureTransportBrokenPipe || !recovering.Retryable {
		t.Fatalf("recovering snapshot=%+v", recovering)
	}
	if recovering.AcceptsWork() {
		t.Fatal("a recovering session accepted work")
	}
	if recovering.ProjectionRevision != 1 || recovering.ProjectionCount != 3 {
		t.Fatalf("a recovering session dropped its last safe projection metadata: %+v", recovering)
	}
	select {
	case outcome := <-results:
		t.Fatalf("a broken transport exited before close: %+v", outcome)
	default:
	}
	if outcome := session.Close(nil); !outcome.Clean || outcome.ExitCode != ExitClean {
		t.Fatalf("close after a broken transport = %+v", outcome)
	}
	if outcome := <-results; !outcome.Clean {
		t.Fatalf("run outcome = %+v", outcome)
	}
	protocol.mu.Lock()
	notifications := len(protocol.notified)
	protocol.mu.Unlock()
	if notifications == 0 {
		t.Fatal("no failure notification reached the protocol owner")
	}
	path := snapshotPath(observed)
	if len(path) < 5 || path[len(path)-1] != string(StateClosing) {
		t.Fatalf("lifecycle path=%v", path)
	}
	sawRecovering := false
	for index, snapshot := range observed {
		if snapshot.State == StateRecovering {
			sawRecovering = true
			if index == 0 || observed[index-1].State != StateReady {
				t.Fatalf("recovering was not published from ready: %v", path)
			}
		}
		if snapshot.State != StateReady && snapshot.Availability == AvailabilityHealthy {
			t.Fatalf("non-ready state reported healthy: %+v", snapshot)
		}
	}
	if !sawRecovering {
		t.Fatalf("recovering was never published: %v", path)
	}
}

func snapshotPath(observed []HealthSnapshot) []string {
	path := make([]string, 0, len(observed))
	for _, snapshot := range observed {
		path = append(path, string(snapshot.State))
	}
	return path
}

type unblockableReader struct {
	release   chan struct{}
	mu        sync.Mutex
	unblocked int
	closed    int
}

func newUnblockableReader() *unblockableReader {
	return &unblockableReader{release: make(chan struct{})}
}

func (reader *unblockableReader) Read([]byte) (int, error) {
	<-reader.release
	return 0, io.EOF
}

func (reader *unblockableReader) Unblock() {
	reader.mu.Lock()
	reader.unblocked++
	reader.mu.Unlock()
	select {
	case <-reader.release:
	default:
		close(reader.release)
	}
}

func (reader *unblockableReader) counts() (int, int) {
	reader.mu.Lock()
	defer reader.mu.Unlock()
	return reader.unblocked, reader.closed
}

func TestRuntimeCloseInterruptsABlockedRead(t *testing.T) {
	input := newUnblockableReader()
	config := DefaultConfig("test", nil, testLimits())
	config.AllowedOrigins = []string{testOrigin}
	session, err := Bootstrap(config, Invocation{Origin: testOrigin}, input, io.Discard, Dependencies{
		Protocol:   &fakeProtocol{binding: validBinding()},
		Projection: readyProjection(),
	})
	if err != nil {
		t.Fatal(err)
	}
	baseline := goruntime.NumGoroutine()
	results := make(chan Outcome, 1)
	go func() { results <- session.Run(context.Background()) }()
	waitFor(t, "the handshake to park on the browser pipe", func() bool {
		return input.unblocked == 0 && goruntime.NumGoroutine() > baseline
	})
	started := time.Now()
	outcome := session.Close(nil)
	if elapsed := time.Since(started); elapsed > 500*time.Millisecond {
		t.Fatalf("close waited %s on a blocked read", elapsed)
	}
	if !outcome.Clean || outcome.ExitCode != ExitClean {
		t.Fatalf("close outcome=%+v", outcome)
	}
	if final := <-results; !final.Clean {
		t.Fatalf("run outcome=%+v", final)
	}
	unblocked, _ := input.counts()
	if unblocked != 1 {
		t.Fatalf("reader interrupt calls=%d, want exactly one", unblocked)
	}
	waitFor(t, "the reader goroutine to exit", func() bool {
		return goruntime.NumGoroutine() <= baseline+1
	})
}

func TestRuntimeBlockedReadIsCancellableWithoutAnUnblocker(t *testing.T) {
	reader := &blockingReader{release: make(chan struct{})}
	defer close(reader.release)
	limits := testLimits()
	input := newFrameReader(reader, limits)
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() {
		_, err := input.ReadFrame(ctx)
		done <- err
	}()
	waitFor(t, "the read to park on the fake pipe", func() bool { return reader.holding() > 0 })
	cancel()
	select {
	case err := <-done:
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("cancelled read error=%v", err)
		}
	case <-time.After(time.Second):
		t.Fatal("a cancelled read never returned")
	}
}

func TestRuntimeDiagnosticTransitionCountsAreObservable(t *testing.T) {
	input := bytes.NewReader(frame("hello"))
	config := DefaultConfig("test", nil, testLimits())
	config.AllowedOrigins = []string{testOrigin}
	session, err := Bootstrap(config, Invocation{Origin: testOrigin}, input, io.Discard, Dependencies{
		Protocol:   &fakeProtocol{binding: validBinding()},
		Projection: readyProjection(),
	})
	if err != nil {
		t.Fatal(err)
	}
	session.Run(context.Background())
	transitions := uint64(0)
	sawCount := false
	for _, entry := range session.Diagnostics().Entries() {
		if entry.Event != EventStateChanged {
			continue
		}
		transitions++
		if entry.Count == 0 {
			t.Fatalf("state change %q carried no transition count", entry.Event)
		}
		if uint64(entry.Count) != transitions {
			t.Fatalf("state change %d reported count %d", transitions, entry.Count)
		}
		if entry.Count == int64(len(entry.State)) {
			t.Fatalf("state change count %d mirrors the state name length", entry.Count)
		}
		sawCount = true
	}
	if !sawCount || transitions < 4 {
		t.Fatalf("observed %d state changes, want the full startup path", transitions)
	}
	if session.Health().Transitions != transitions {
		t.Fatalf("snapshot transitions=%d, diagnostics=%d", session.Health().Transitions, transitions)
	}
}

func waitFor(t *testing.T, what string, condition func() bool) {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if condition() {
			return
		}
		time.Sleep(time.Millisecond)
	}
	t.Fatalf("timed out waiting for %s", what)
}
