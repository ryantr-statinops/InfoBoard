package phase06

import (
	"context"
	"encoding/binary"
	"encoding/json"
	"go/parser"
	"go/token"
	"io"
	"os"
	"path/filepath"
	"reflect"
	goruntime "runtime"
	"strconv"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/ryantr-statinops/InfoBoard/host/internal/domain"
	hostruntime "github.com/ryantr-statinops/InfoBoard/host/internal/runtime"
)

const helloPayload = `{"type":"hello","protocol":1}`

// slack absorbs scheduler delay when asserting that a bounded operation stayed
// inside its own budget; it never hides an unbounded wait.
const slack = 2 * time.Second

func expectedStringsOf(t *testing.T, fixture HostFixture, key string) []string {
	t.Helper()
	return expectStrings(t, fixture.FixtureID, fixture.Expected, key)
}

func expectedStringOf(t *testing.T, fixture HostFixture, key string) string {
	t.Helper()
	return expectString(t, fixture.FixtureID, fixture.Expected, key)
}

func expectedBoolOf(t *testing.T, fixture HostFixture, key string) bool {
	t.Helper()
	return expectBool(t, fixture.FixtureID, fixture.Expected, key)
}

func expectedIntOf(t *testing.T, fixture HostFixture, key string) int64 {
	t.Helper()
	return expectInt(t, fixture.FixtureID, fixture.Expected, key)
}

func expectedClassOf(t *testing.T, fixture HostFixture, key string) hostruntime.FailureClass {
	t.Helper()
	raw, declared := fixture.Expected[key]
	if !declared || string(raw) == "null" {
		return hostruntime.FailureNone
	}
	var value string
	if err := json.Unmarshal(raw, &value); err != nil {
		t.Fatalf("%s: %s is not a class literal: %v", fixture.FixtureID, key, err)
	}
	return hostruntime.FailureClass(value)
}

// assertPath compares the published lifecycle path with the fixture expectation.
func assertPath(t *testing.T, key string, got []hostruntime.LifecycleState, want []string) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("%s path = %v, want %v", key, got, want)
	}
	for index := range want {
		if got[index] != hostruntime.LifecycleState(want[index]) {
			t.Fatalf("%s path = %v, want %v", key, got, want)
		}
	}
}

// assertStdoutIsFrameOnly proves the browser-provided stdout carries protocol
// frames and nothing else: no diagnostics, no panic text, no stray bytes.
func assertStdoutIsFrameOnly(t *testing.T, key string, rig *rig, wantFrames int64) []DecodedFrame {
	t.Helper()
	raw := rig.output.Bytes()
	frames, stray := ScanFrames(raw)
	if stray != 0 {
		t.Fatalf("%s: %d stray stdout bytes across %d captured bytes", key, stray, len(raw))
	}
	if int64(len(frames)) != wantFrames {
		t.Fatalf("%s: frames = %d, want %d", key, len(frames), wantFrames)
	}
	for _, frame := range frames {
		if frame.Size != 4+len(frame.Payload) || !strings.HasSuffix(string(frame.Payload), "}") {
			t.Fatalf("%s: frame at offset %d is not a complete whole frame", key, frame.Offset)
		}
	}
	return frames
}

// assertSentinelsAbsent proves the bounded diagnostic sink carries state metadata
// only, never a raw title, URL, query, token, or page marker.
func assertSentinelsAbsent(t *testing.T, key string, rig *rig) {
	t.Helper()
	entries := rig.sink.Entries()
	if len(entries) == 0 {
		t.Fatalf("%s: the diagnostic sink recorded nothing, so redaction was not exercised", key)
	}
	minimum, declared := rig.fixture.Expected["diagnostic_records_at_least"]
	if declared {
		var want int
		if err := json.Unmarshal(minimum, &want); err != nil {
			t.Fatalf("%s: diagnostic_records_at_least is not an integer", rig.fixture.FixtureID)
		}
		if len(entries) < want {
			t.Fatalf("%s: diagnostic records = %d, want at least %d", key, len(entries), want)
		}
	}
	sentinels := rig.corpus.PrivacySentinels
	needles := []string{sentinels.Title, sentinels.URL, sentinels.Query, sentinels.Token, sentinels.PageMarker}
	for _, entry := range entries {
		encoded, err := json.Marshal(entry)
		if err != nil {
			t.Fatal(err)
		}
		lowered := strings.ToLower(string(encoded))
		for _, needle := range needles {
			if strings.Contains(lowered, strings.ToLower(needle)) {
				t.Fatalf("%s: diagnostic %s leaked the sensitive sentinel %q", key, entry.Event, needle)
			}
		}
		if len(entry.SessionID) != 32 {
			t.Fatalf("%s: diagnostic session identity %q is not the bounded opaque form", key, entry.SessionID)
		}
	}
}

// assertHealthHasNoRawTabFields is a structural privacy guard: no public status
// type may grow a field that could carry raw tab content.
func assertHealthHasNoRawTabFields(t *testing.T) {
	t.Helper()
	forbidden := []string{"title", "url", "query", "token", "cookie", "content", "domain", "path", "search"}
	for _, structure := range []reflect.Type{
		reflect.TypeOf(hostruntime.HealthSnapshot{}),
		reflect.TypeOf(hostruntime.Diagnostic{}),
		reflect.TypeOf(hostruntime.Notification{}),
	} {
		for index := 0; index < structure.NumField(); index++ {
			name := strings.ToLower(structure.Field(index).Name)
			for _, needle := range forbidden {
				if strings.Contains(name, needle) {
					t.Fatalf("%s.%s must not expose raw tab content", structure.Name(), structure.Field(index).Name)
				}
			}
		}
	}
}

// startHealthySession drives one fixture session to Ready through the ordered
// startup handoff and returns the session, rig, and outcome channel.
func startHealthySession(t *testing.T, ctx context.Context, fixture HostFixture) (*hostruntime.Session, *rig, <-chan hostruntime.Outcome) {
	t.Helper()
	rig := newOwnedRig(t, ctx, fixture.FixtureID)
	rig.QueueFrames(helloPayload)
	session, err := rig.Bootstrap(t)
	if err != nil {
		t.Fatalf("%s: bootstrap: %v", fixture.FixtureID, err)
	}
	outcomes := rig.Start(ctx, session)
	rig.states.AwaitState(t, hostruntime.StateReady, 2*(rig.limits.HandshakeTimeout+rig.limits.SynchronizationTimeout)+slack)
	return session, rig, outcomes
}

func TestFixtureHostStartHealthy(t *testing.T) {
	assertHealthHasNoRawTabFields(t)
	ctx := context.Background()
	corpus := LoadCorpus(t)
	fixture := corpus.Fixture(t, "FX-HOST-START-HEALTHY")
	session, rig, outcomes := startHealthySession(t, ctx, fixture)

	health := session.Health()
	if health.Availability != hostruntime.AvailabilityHealthy {
		t.Fatalf("availability at ready = %s, want healthy", health.Availability)
	}
	if !health.AcceptsWork() {
		t.Fatal("a ready session refused work")
	}
	for _, snapshot := range rig.states.snapshots {
		if snapshot.State != hostruntime.StateReady && snapshot.Availability == hostruntime.AvailabilityHealthy {
			t.Fatalf("state %s reported healthy before synchronization readiness", snapshot.State)
		}
		if snapshot.AcceptsWork() && snapshot.State != hostruntime.StateReady {
			t.Fatalf("state %s accepted work before ready", snapshot.State)
		}
	}
	if !health.ProjectionKnown {
		t.Fatal("ready session did not record a known projection")
	}
	if int64(health.ProjectionRevision) != expectedIntOf(t, fixture, "projection_revision") {
		t.Fatalf("projection revision = %d, want %d", health.ProjectionRevision, expectedIntOf(t, fixture, "projection_revision"))
	}
	if int64(health.ProjectionCount) != expectedIntOf(t, fixture, "projection_count") {
		t.Fatalf("projection count = %d, want %d", health.ProjectionCount, expectedIntOf(t, fixture, "projection_count"))
	}
	if string(health.Persistence) != expectedStringOf(t, fixture, "persistence") {
		t.Fatalf("persistence = %s, want %s", health.Persistence, expectedStringOf(t, fixture, "persistence"))
	}
	if health.ProfileID != domain.ProfileID(corpus.Identity.ProfileID) {
		t.Fatalf("profile identity = %s, want the bound fixture profile", health.ProfileID)
	}
	if health.ProjectionEpoch != corpus.Identity.ProjectionEpoch {
		t.Fatalf("projection epoch = %s, want the authoritative snapshot epoch", health.ProjectionEpoch)
	}

	negotiates, _, _, _ := rig.protocol.counts()
	if negotiates != 1 {
		t.Fatalf("protocol negotiations = %d, want exactly one", negotiates)
	}
	if targets := rig.projector.observedTargets(); len(targets) != 1 {
		t.Fatalf("projection synchronizations = %d, want exactly one", len(targets))
	} else if targets[0].ProfileID != health.ProfileID || targets[0].SessionID != session.ID() {
		t.Fatalf("synchronization target = %+v, want the bound profile and session identity", targets[0])
	}
	if preps, flushes, _ := rig.store.counts(); preps != 1 || flushes != 0 {
		t.Fatalf("persistence before close: prepares=%d flushes=%d, want one prepare and no flush", preps, flushes)
	}

	session.Close(nil)
	outcome := awaitOutcome(t, outcomes, rig.limits.ShutdownTimeout+slack)
	if !outcome.Clean || outcome.Failure != expectedClassOf(t, fixture, "failure_class") {
		t.Fatalf("outcome = %+v, want a clean close", outcome)
	}
	if outcome.Retryable != expectedBoolOf(t, fixture, "retryable") {
		t.Fatalf("outcome retryable = %t, want %t", outcome.Retryable, expectedBoolOf(t, fixture, "retryable"))
	}
	if outcome.ExitCode != hostruntime.ExitClean {
		t.Fatalf("exit code = %d, want %d", outcome.ExitCode, hostruntime.ExitClean)
	}
	assertPath(t, "session", rig.states.Path(), expectedStringsOf(t, fixture, "lifecycle_path"))
	availabilities := rig.states.Availabilities()
	wantBeforeReady := expectedStringsOf(t, fixture, "availability_before_ready")
	if len(availabilities) == 0 || string(availabilities[0]) != wantBeforeReady[0] {
		t.Fatalf("availability before ready = %v, want %v", availabilities, wantBeforeReady)
	}
	if _, flushes, closes := rig.store.counts(); flushes != 1 || closes != 1 {
		t.Fatalf("shutdown side effects: flushes=%d closes=%d, want exactly one flush and close", flushes, closes)
	}
	if _, _, shutdowns, _ := rig.protocol.counts(); shutdowns != 1 {
		t.Fatalf("protocol shutdown calls = %d, want exactly one", shutdowns)
	}
	assertStdoutIsFrameOnly(t, "session", rig, expectedIntOf(t, fixture, "stdout_frames"))
	assertSentinelsAbsent(t, "session", rig)
}

func TestFixtureHostAbsent(t *testing.T) {
	ctx := context.Background()
	corpus := LoadCorpus(t)
	fixture := corpus.Fixture(t, "FX-HOST-ABSENT")
	rig := newOwnedRig(t, ctx, "FX-HOST-ABSENT")
	rig.origin = expectString(t, fixture.FixtureID, fixture.Scenario, "invocation_origin")
	rig.allowed = nil
	rig.config.AllowedOrigins = nil

	session, err := rig.Bootstrap(t)
	if err == nil {
		t.Fatal("an unregistered host produced a session instead of failing closed")
	}
	if session != nil {
		t.Fatal("a session was returned for an unregistered origin")
	}
	class, retryable := hostruntime.ClassifyFailure(err)
	if class != expectedClassOf(t, fixture, "failure_class") {
		t.Fatalf("failure class = %s, want %s", class, expectedClassOf(t, fixture, "failure_class"))
	}
	if retryable != expectedBoolOf(t, fixture, "retryable") {
		t.Fatalf("retryable = %t, want %t", retryable, expectedBoolOf(t, fixture, "retryable"))
	}
	if class.Retryable() && expectedStringOf(t, fixture, "recovery_action") == "repair_registration" {
		t.Fatal("a registration failure must offer repair rather than a blind retry")
	}
	if strings.Contains(strings.ToLower(err.Error()), strings.ToLower(corpus.Identity.Origin)) {
		t.Fatalf("bootstrap error %q leaked the invocation origin", err.Error())
	}
	entries := rig.config.Diagnostics.Entries()
	if len(entries) < int(expectedIntOf(t, fixture, "diagnostic_records_at_least")) {
		t.Fatalf("diagnostic records = %d, want the rejected-bootstrap record", len(entries))
	}
	last := entries[len(entries)-1]
	if last.Availability != hostruntime.Availability(expectedStringOf(t, fixture, "availability")) {
		t.Fatalf("bootstrap diagnostic availability = %s, want %s", last.Availability, expectedStringOf(t, fixture, "availability"))
	}
	if last.Level != hostruntime.LevelError {
		t.Fatalf("bootstrap diagnostic level = %s, want error", last.Level)
	}
	assertStdoutIsFrameOnly(t, "absent", rig, expectedIntOf(t, fixture, "stdout_frames"))
	if got := int64(len(rig.output.Bytes())); got != expectedIntOf(t, fixture, "stdout_bytes") {
		t.Fatalf("stdout bytes = %d, want %d", got, expectedIntOf(t, fixture, "stdout_bytes"))
	}
	if rig.input.BytesRead() != 0 || rig.input.Reads() != 0 {
		t.Fatalf("stdin consumed = %d bytes over %d reads, want none before registration is accepted", rig.input.BytesRead(), rig.input.Reads())
	}
	if _, flushes, closes := rig.store.counts(); flushes != 0 || closes != 0 {
		t.Fatal("a rejected bootstrap must not touch the persistence dependency")
	}
	assertSentinelsAbsent(t, "absent", rig)
}

func TestFixtureHostRecovering(t *testing.T) {
	corpus := LoadCorpus(t)
	fixture := corpus.Fixture(t, "FX-HOST-RECOVERING")
	disruptions := expectSlice(t, fixture.FixtureID, fixture.Scenario, "session_1_disruptions")
	if int64(len(disruptions)) != expectedIntOf(t, fixture, "session_1_disruptions") {
		t.Fatalf("declared %d disruptions, want %d", len(disruptions), expectedIntOf(t, fixture, "session_1_disruptions"))
	}

	for _, disruption := range disruptions {
		name := expectString(t, fixture.FixtureID, disruption, "name")
		t.Run(name, func(t *testing.T) {
			ctx := context.Background()
			wantClass := hostruntime.FailureClass(expectString(t, fixture.FixtureID, disruption, "observed_failure_class"))
			wantRetryable := expectBool(t, fixture.FixtureID, disruption, "retryable")
			wantAvailability := hostruntime.Availability(expectString(t, fixture.FixtureID, disruption, "availability"))
			terminal := expectBool(t, fixture.FixtureID, disruption, "terminal_without_close")

			session, rig, outcomes := startHealthySession(t, ctx, fixture)
			failedID := session.ID()
			switch name {
			case "broken_pipe_on_stdin":
				// An unexpected browser-side pipe failure must not look like a
				// clean end of stream.
				rig.input.FailAfter(syscall.EPIPE)
			case "closed_browser_handle":
				rig.input.FailAfter(syscall.EBADF)
			case "truncated_frame":
				rig.input.Queue(frameHeader(64))
				rig.input.Queue([]byte(`{"type":"snap`))
				rig.input.FailAfter(io.EOF)
			default:
				t.Fatalf("unhandled disruption %q", name)
			}

			if wantAvailability != hostruntime.AvailabilityHealthy {
				rig.states.AwaitFailure(t, wantClass, rig.limits.RequestTimeout+slack)
				observed := session.Health()
				if observed.Availability != wantAvailability {
					t.Fatalf("availability = %s, want %s", observed.Availability, wantAvailability)
				}
				if observed.Failure != wantClass {
					t.Fatalf("failure class = %s, want %s", observed.Failure, wantClass)
				}
				if observed.Retryable != wantRetryable {
					t.Fatalf("retryable = %t, want %t", observed.Retryable, wantRetryable)
				}
				if expectedBoolOf(t, fixture, "session_1_never_healthy_after_disruption") && observed.Availability == hostruntime.AvailabilityHealthy {
					t.Fatal("the disrupted session still reported healthy")
				}
				if observed.AcceptsWork() {
					t.Fatalf("a %s session still accepted work", wantAvailability)
				}
				if expectedBoolOf(t, fixture, "session_1_stale_projection_served") && observed.ProjectionRevision == 0 && wantAvailability == hostruntime.AvailabilityRecovering {
					t.Fatal("the recovering session dropped its last safe projection revision")
				}
			}

			if expectedBoolOf(t, fixture, "session_1_failure_is_reported_to_protocol_owner") {
				waitForCondition(t, "the failure notification to reach the protocol owner", func() bool {
					_, _, _, notes := rig.protocol.counts()
					for _, note := range notes {
						if note.Reason == hostruntime.NotificationFailure && note.Failure == wantClass {
							return true
						}
					}
					return false
				})
				_, _, _, notes := rig.protocol.counts()
				for _, note := range notes {
					if note.Reason != hostruntime.NotificationFailure || note.Failure != wantClass {
						continue
					}
					if note.Retryable != wantRetryable {
						t.Fatalf("%s notified retryable=%t, want %t", name, note.Retryable, wantRetryable)
					}
					if note.Snapshot.Availability == hostruntime.AvailabilityHealthy && note.Failure != hostruntime.FailureNone {
						t.Fatalf("%s notified a healthy session alongside a failure", name)
					}
				}
			}

			if terminal {
				// A terminal contract failure ends the session by itself.
				outcome := awaitOutcome(t, outcomes, rig.limits.ShutdownTimeout+slack)
				if outcome.Failure != wantClass {
					t.Fatalf("%s outcome class = %s, want %s", name, outcome.Failure, wantClass)
				}
				if outcome.Retryable != wantRetryable {
					t.Fatalf("%s outcome retryable = %t, want %t", name, outcome.Retryable, wantRetryable)
				}
				if outcome.Clean {
					t.Fatalf("%s was reported as a clean end of stream", name)
				}
			} else {
				// A recoverable browser transport keeps the session published as
				// recovering until the extension gives up on this connection.
				if expectedBoolOf(t, fixture, "session_1_recoverable_transport_stays_recovering") {
					select {
					case early := <-outcomes:
						t.Fatalf("%s ended before a close was requested: %+v", name, early)
					case <-time.After(100 * time.Millisecond):
					}
				}
				session.Close(nil)
				outcome := awaitOutcome(t, outcomes, rig.limits.ShutdownTimeout+slack)
				if !outcome.Clean {
					t.Fatalf("%s outcome = %+v, want a clean close after the failure was reported", name, outcome)
				}
			}
			if got := session.Health().State; got != hostruntime.StateClosing {
				t.Fatalf("%s final state = %s, want closing", name, got)
			}
			assertSentinelsAbsent(t, name, rig)

			// A fresh browser-provided connection plus an authoritative snapshot
			// must reach a healthy state without reinstalling anything.
			freshCtx := context.Background()
			fresh2 := newOwnedRig(t, freshCtx, "FX-HOST-RECOVERING")
			fresh2.projector.status = hostruntime.ProjectionStatus{
				Known:    true,
				Epoch:    corpus.Identity.Session2ProjectionEpoch,
				Revision: uint64(expectedIntOf(t, fixture, "session_2_projection_revision")),
				Count:    uint64(expectedIntOf(t, fixture, "session_2_projection_count")),
				Ready:    true,
			}
			fresh2.QueueFrames(helloPayload)
			healthy, err := fresh2.Bootstrap(t)
			if err != nil {
				t.Fatalf("fresh session bootstrap: %v", err)
			}
			freshOutcomes := fresh2.Start(freshCtx, healthy)
			fresh2.states.AwaitState(t, hostruntime.StateReady, 2*(fresh2.limits.HandshakeTimeout+fresh2.limits.SynchronizationTimeout)+slack)

			health := healthy.Health()
			if health.Availability != hostruntime.Availability(expectedStringOf(t, fixture, "session_2_availability")) {
				t.Fatalf("fresh availability = %s, want %s", health.Availability, expectedStringOf(t, fixture, "session_2_availability"))
			}
			if health.Failure != expectedClassOf(t, fixture, "session_2_failure_class") {
				t.Fatalf("fresh failure = %s, want none", health.Failure)
			}
			if health.ProjectionEpoch != corpus.Identity.Session2ProjectionEpoch {
				t.Fatalf("fresh projection epoch = %s, want the authoritative second snapshot", health.ProjectionEpoch)
			}
			if expectedBoolOf(t, fixture, "session_ids_distinct") && healthy.ID() == failedID {
				t.Fatalf("the fresh session reused the failed session identity %q", failedID)
			}
			if expectedBoolOf(t, fixture, "reached_healthy_without_reinstall") && !health.AcceptsWork() {
				t.Fatal("a fresh session plus snapshot did not return to a workable state")
			}
			healthy.Close(nil)
			if secondOutcome := awaitOutcome(t, freshOutcomes, fresh2.limits.ShutdownTimeout+slack); !secondOutcome.Clean {
				t.Fatalf("fresh session outcome = %+v, want clean", secondOutcome)
			}
			assertPath(t, "session_2", fresh2.states.Path(), expectedStringsOf(t, fixture, "session_2_lifecycle_path"))
			assertStdoutIsFrameOnly(t, "session_1", rig, expectedIntOf(t, fixture, "stdout_frames"))
			assertStdoutIsFrameOnly(t, "session_2", fresh2, expectedIntOf(t, fixture, "stdout_frames"))
			assertSentinelsAbsent(t, "session_2", fresh2)
		})
	}
}

func TestFixtureHostStdioOwnership(t *testing.T) {
	ctx := context.Background()
	corpus := LoadCorpus(t)
	fixture := corpus.Fixture(t, "FX-HOST-STDIO-OWNERSHIP")
	session, rig, outcomes := startHealthySession(t, ctx, fixture)

	concurrency := expectedIntOf(t, fixture, "concurrent_requests")
	if concurrency != corpus.Limits.MaxConcurrentRequests {
		t.Fatalf("concurrent_requests = %d, want the artifact concurrency budget %d", concurrency, corpus.Limits.MaxConcurrentRequests)
	}
	release := make(chan struct{})
	rig.protocol.accept = func(callCtx context.Context, channel hostruntime.FrameChannel, _ []byte) error {
		select {
		case <-release:
		case <-callCtx.Done():
			return callCtx.Err()
		}
		return channel.WriteFrame(callCtx, []byte(`{"type":"query_result"}`))
	}
	for index := int64(0); index < concurrency; index++ {
		rig.QueueFrames(`{"type":"query"}`)
	}
	waitForCondition(t, "every request to reach the protocol owner", func() bool {
		_, accepts, _, _ := rig.protocol.counts()
		return int64(accepts) >= concurrency
	})
	close(release)
	waitForCondition(t, "all responses to reach the serialized writer", func() bool {
		return session.Health().FramesWritten >= uint64(expectedIntOf(t, fixture, "frames_written"))
	})

	if overlaps := int64(rig.output.OverlappingWrites()); overlaps != expectedIntOf(t, fixture, "overlapping_stdout_writes") {
		t.Fatalf("overlapping stdout writes = %d, want %d serialized writes", overlaps, expectedIntOf(t, fixture, "overlapping_stdout_writes"))
	}
	frames := assertStdoutIsFrameOnly(t, "stdio", rig, expectedIntOf(t, fixture, "stdout_frames"))
	if expectedBoolOf(t, fixture, "frames_written_matches_health_snapshot") {
		if int64(session.Health().FramesWritten) != int64(len(frames)) {
			t.Fatalf("health frames written = %d, captured %d", session.Health().FramesWritten, len(frames))
		}
	}
	if expectedBoolOf(t, fixture, "diagnostic_records_bounded_by_max_diagnostic_records") {
		if got := len(rig.sink.Entries()); got > int(corpus.Limits.MaxDiagnosticRecords) {
			t.Fatalf("diagnostic records = %d, above the bounded retention %d", got, corpus.Limits.MaxDiagnosticRecords)
		}
	}
	if health := session.Health(); health.Availability != hostruntime.Availability(expectedStringOf(t, fixture, "availability_at_ready")) {
		t.Fatalf("availability = %s, want healthy", health.Availability)
	}
	waitForCondition(t, "all requests to complete", func() bool {
		return session.Health().RequestsCompleted >= uint64(concurrency)
	})

	session.Close(nil)
	if outcome := awaitOutcome(t, outcomes, rig.limits.ShutdownTimeout+slack); !outcome.Clean {
		t.Fatalf("outcome = %+v, want clean", outcome)
	}
	assertWriterRefusesAfterClose(t, "stdio", session)
	assertSentinelsAbsent(t, "stdio", rig)
}

func TestFixtureHostTimeout(t *testing.T) {
	corpus := LoadCorpus(t)
	fixture := corpus.Fixture(t, "FX-HOST-TIMEOUT")
	divisor := expectInt(t, fixture.FixtureID, fixture.Scenario, "budget_scaling_divisor")
	cases := expectSlice(t, fixture.FixtureID, fixture.Scenario, "cases")
	if int64(len(cases)) != expectedIntOf(t, fixture, "timed_out_operations") {
		t.Fatalf("declared %d timeout cases, want %d", len(cases), expectedIntOf(t, fixture, "timed_out_operations"))
	}
	before := goruntime.NumGoroutine()

	for _, testCase := range cases {
		name := expectString(t, fixture.FixtureID, testCase, "name")
		t.Run(name, func(t *testing.T) {
			ctx := context.Background()
			budgetName := expectString(t, fixture.FixtureID, testCase, "budget")
			wantClass := hostruntime.FailureClass(expectString(t, fixture.FixtureID, testCase, "observed_failure_class"))
			wantRetryable := expectBool(t, fixture.FixtureID, testCase, "retryable")
			blocked := expectString(t, fixture.FixtureID, testCase, "blocked")
			via := expectString(t, fixture.FixtureID, testCase, "observed_via")
			budget, ok := corpus.Limits.Budget(budgetName)
			if !ok {
				t.Fatalf("unknown fixture budget %q", budgetName)
			}
			budget /= time.Duration(divisor)
			if budget <= 0 {
				t.Fatalf("scaled budget for %s is not positive", budgetName)
			}

			rig := newOwnedRig(t, ctx, "FX-HOST-TIMEOUT")
			rig.limits = scaleLimits(rig.limits, budgetName, budget)
			rig.config.Limits = rig.limits
			rig.input = NewFakeReader(ctx)
			rig.output = NewFakeWriter(ctx)
			switch blocked {
			case "protocol_negotiate":
				rig.protocol.negotiateGate = NewBlockingGate()
			case "projection_synchronize":
				rig.projector.gate = NewBlockingGate()
			case "persistence_flush":
				rig.store.flushGate = NewBlockingGate()
			case "protocol_accept":
			default:
				t.Fatalf("unhandled blocked operation %q", blocked)
			}
			if blocked != "protocol_negotiate" {
				rig.QueueFrames(helloPayload)
			}

			session, err := rig.Bootstrap(t)
			if err != nil {
				t.Fatalf("bootstrap: %v", err)
			}
			outcomes := rig.Start(ctx, session)
			started := time.Now()

			switch via {
			case "outcome":
				var outcome hostruntime.Outcome
				if blocked == "persistence_flush" {
					rig.states.AwaitState(t, hostruntime.StateReady, 2*(rig.limits.HandshakeTimeout+rig.limits.SynchronizationTimeout)+slack)
					session.Close(hostruntime.ErrShutdownBudget)
					outcome = awaitOutcome(t, outcomes, budget+slack)
				} else {
					outcome = awaitOutcome(t, outcomes, budget+slack)
				}
				if elapsed := time.Since(started); elapsed > budget+slack {
					t.Fatalf("%s took %s, above its %s budget", name, elapsed, budget)
				}
				if outcome.Failure != wantClass {
					t.Fatalf("%s failure class = %s, want %s", name, outcome.Failure, wantClass)
				}
				if outcome.Retryable != wantRetryable {
					t.Fatalf("%s retryable = %t, want %t", name, outcome.Retryable, wantRetryable)
				}
				if wantRetryable != (outcome.Failure != hostruntime.FailureNone && outcome.Retryable) {
					t.Fatalf("%s classified retryable=%t for class %s, want %t", name, outcome.Retryable, outcome.Failure, wantRetryable)
				}
				if outcome.ExitCode == hostruntime.ExitClean {
					t.Fatalf("%s reported a clean exit code for a bounded timeout", name)
				}
				if session.Health().Availability == hostruntime.AvailabilityHealthy {
					t.Fatalf("%s left the session healthy after a bounded timeout", name)
				}
				if blocked == "persistence_flush" {
					if session.Health().Persistence != hostruntime.PersistenceDegraded {
						t.Fatalf("persistence = %s, want degraded after a bounded flush timeout", session.Health().Persistence)
					}
					if !expectedBoolOf(t, fixture, "persistence_degrades_without_breaking_lexical_runtime") {
						t.Fatal("a degraded store must be observable")
					}
					classified, classRetryable := hostruntime.ClassifyFailure(hostruntime.ErrShutdownBudget)
					if classified != wantClass || classRetryable != wantRetryable {
						t.Fatalf("shutdown budget maps to (%s,%t), want (%s,%t)", classified, classRetryable, wantClass, wantRetryable)
					}
				}
			case "notification":
				if blocked != "protocol_accept" {
					t.Fatalf("notification observation is not defined for %q", blocked)
				}
				rig.states.AwaitState(t, hostruntime.StateReady, 2*(rig.limits.HandshakeTimeout+rig.limits.SynchronizationTimeout)+slack)
				rig.protocol.acceptGate = NewBlockingGate()
				rig.QueueFrames(`{"type":"query"}`)
				waitForCondition(t, "the request to reach the protocol owner", func() bool {
					_, accepts, _, _ := rig.protocol.counts()
					return accepts >= 1
				})
				waitForCondition(t, "the request budget to expire", func() bool {
					_, _, _, notes := rig.protocol.counts()
					for _, note := range notes {
						if note.Failure == wantClass {
							return true
						}
					}
					return false
				})
				if elapsed := time.Since(started); elapsed > budget+slack {
					t.Fatalf("%s took %s, above its %s budget", name, elapsed, budget)
				}
				_, _, _, notes := rig.protocol.counts()
				sawRetryable := false
				for _, note := range notes {
					if note.Failure != wantClass {
						continue
					}
					if note.Retryable == wantRetryable {
						sawRetryable = true
					}
					if note.Snapshot.Availability != hostruntime.AvailabilityHealthy && note.Failure == hostruntime.FailureNone {
						t.Fatalf("%s notified an unhealthy session with no failure", name)
					}
				}
				if !sawRetryable {
					t.Fatalf("%s published no retryable %s notification", name, wantClass)
				}
				if classified, classRetryable := hostruntime.ClassifyFailure(context.DeadlineExceeded); classified != wantClass || classRetryable != wantRetryable {
					t.Fatalf("an expired request budget maps to (%s,%t), want (%s,%t)", classified, classRetryable, wantClass, wantRetryable)
				}
				session.Close(nil)
				if closing := awaitOutcome(t, outcomes, budget+slack); !closing.Clean {
					t.Fatalf("%s closing outcome = %+v, want clean", name, closing)
				}
			default:
				t.Fatalf("unhandled observation seam %q", via)
			}
			assertStdoutIsFrameOnly(t, name, rig, expectedIntOf(t, fixture, "stdout_frames"))
			assertSentinelsAbsent(t, name, rig)
		})
	}

	if leaked := goruntime.NumGoroutine() - before; leaked > 2 {
		t.Fatalf("goroutines grew by %d across the timeout cases, want no leaked worker", leaked)
	}
}

func TestFixtureHostCancelDisconnect(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	corpus := LoadCorpus(t)
	fixture := corpus.Fixture(t, "FX-HOST-CANCEL-DISCONNECT")
	session, rig, outcomes := startHealthySession(t, ctx, fixture)

	before := goruntime.NumGoroutine()
	held := make(chan struct{})
	rig.protocol.accept = func(callCtx context.Context, _ hostruntime.FrameChannel, _ []byte) error {
		select {
		case <-held:
			return nil
		case <-callCtx.Done():
			return callCtx.Err()
		}
	}
	inflight := expectedIntOf(t, fixture, "in_flight_stopped")
	for index := int64(0); index < inflight; index++ {
		rig.QueueFrames(`{"type":"query"}`)
	}
	waitForCondition(t, "every request to reach the protocol owner", func() bool {
		_, accepts, _, _ := rig.protocol.counts()
		return int64(accepts) >= inflight
	})

	cancel()
	outcome := awaitOutcome(t, outcomes, rig.limits.ShutdownTimeout+slack)
	if !outcome.Clean {
		t.Fatalf("outcome = %+v, want a clean client disconnect", outcome)
	}
	if outcome.Failure != expectedClassOf(t, fixture, "failure_class") {
		t.Fatalf("failure class = %s, want none", outcome.Failure)
	}

	calls := expectedIntOf(t, fixture, "close_calls_observed")
	for index := int64(0); index < calls; index++ {
		session.Close(nil)
	}
	if _, flushes, closes := rig.store.counts(); flushes != 1 || closes != 1 {
		t.Fatalf("close side effects: flushes=%d closes=%d, want exactly one of each", flushes, closes)
	}
	if preps, _, _ := rig.store.counts(); preps != 1 {
		t.Fatalf("persistence prepares = %d, want exactly one", preps)
	}
	assertWriterRefusesAfterClose(t, "cancel", session)
	if _, _, shutdowns, _ := rig.protocol.counts(); shutdowns != 1 {
		t.Fatalf("protocol shutdown calls = %d, want exactly one", shutdowns)
	}
	terminal := session.Health()
	if terminal.State != hostruntime.StateClosing {
		t.Fatalf("terminal state = %s, want closing", terminal.State)
	}
	if terminal.Availability != hostruntime.Availability(expectedStringOf(t, fixture, "availability")) {
		t.Fatalf("terminal availability = %s, want %s", terminal.Availability, expectedStringOf(t, fixture, "availability"))
	}
	if terminal.AcceptsWork() {
		t.Fatal("a finished session still accepted work")
	}
	assertPath(t, "cancel", rig.states.Path(), expectedStringsOf(t, fixture, "lifecycle_path"))

	// An unrelated concurrent session must not be cancelled by this one.
	otherCtx := context.Background()
	other := newOwnedRig(t, otherCtx, "FX-HOST-CANCEL-DISCONNECT")
	other.QueueFrames(helloPayload)
	neighbour, err := other.Bootstrap(t)
	if err != nil {
		t.Fatalf("neighbour bootstrap: %v", err)
	}
	otherOutcomes := other.Start(otherCtx, neighbour)
	other.states.AwaitState(t, hostruntime.StateReady, 2*(other.limits.HandshakeTimeout+other.limits.SynchronizationTimeout)+slack)
	if !neighbour.Health().AcceptsWork() {
		t.Fatal("closing one session left an unrelated session unusable")
	}
	if neighbour.ID() == session.ID() {
		t.Fatal("two live sessions shared one identity")
	}
	neighbour.Close(nil)
	if neighbourOutcome := awaitOutcome(t, otherOutcomes, other.limits.ShutdownTimeout+slack); !neighbourOutcome.Clean {
		t.Fatalf("neighbour outcome = %+v, want clean", neighbourOutcome)
	}
	if expectedBoolOf(t, fixture, "unrelated_session_cancelled") {
		t.Fatal("an unrelated session was cancelled")
	}

	if leaked := goruntime.NumGoroutine() - before; leaked > 2 {
		t.Fatalf("goroutines grew by %d, want no leaked worker", leaked)
	}
	assertStdoutIsFrameOnly(t, "cancel", rig, expectedIntOf(t, fixture, "stdout_frames"))
	assertSentinelsAbsent(t, "cancel", rig)
}

func TestFixtureHostResourceLimit(t *testing.T) {
	ctx := context.Background()
	corpus := LoadCorpus(t)
	fixture := corpus.Fixture(t, "FX-HOST-RESOURCE-LIMIT")
	cases := expectSlice(t, fixture.FixtureID, fixture.Scenario, "rejected_inputs")
	if int64(len(cases)) != expectedIntOf(t, fixture, "rejected_inputs") {
		t.Fatalf("declared %d rejected cases, want %d", len(cases), expectedIntOf(t, fixture, "rejected_inputs"))
	}
	seen := map[string]bool{}

	for _, testCase := range cases {
		name := expectString(t, fixture.FixtureID, testCase, "case")
		wantClass := hostruntime.FailureClass(expectString(t, fixture.FixtureID, testCase, "observed_failure_class"))
		wantRetryable := expectBool(t, fixture.FixtureID, testCase, "retryable")
		via := expectString(t, fixture.FixtureID, testCase, "observed_via")
		if seen[name] {
			t.Fatalf("rejected case %q is duplicated", name)
		}
		seen[name] = true

		t.Run(name, func(t *testing.T) {
			rig := newOwnedRig(t, ctx, "FX-HOST-RESOURCE-LIMIT")
			switch name {
			case "oversized_frame":
				rig.input.Queue(frameHeader(uint32(corpus.Limits.MaxFrameBytes + 1)))
			case "empty_frame":
				rig.input.Queue(frameHeader(0))
			case "truncated_frame":
				rig.input.Queue(append(frameHeader(64), []byte(`{"type":"snap`)...))
				rig.input.EndStream()
			case "concurrency_overflow":
				held := make(chan struct{})
				rig.protocol.accept = func(callCtx context.Context, _ hostruntime.FrameChannel, _ []byte) error {
					select {
					case <-held:
						return nil
					case <-callCtx.Done():
						return callCtx.Err()
					}
				}
				rig.QueueFrames(helloPayload)
				for index := 0; index <= rig.limits.MaxConcurrentRequests; index++ {
					rig.QueueFrames(`{"type":"query"}`)
				}
			case "invalid_limits":
				rig.config.Limits = zeroLimits(rig.limits, corpus.Limits.ZeroedLimits())
			default:
				t.Fatalf("unhandled rejected case %q", name)
			}

			if name == "invalid_limits" {
				session, err := rig.Bootstrap(t)
				if err == nil || session != nil {
					t.Fatal("an invalid limits configuration produced a session")
				}
				class, _ := hostruntime.ClassifyFailure(err)
				if class != wantClass {
					t.Fatalf("failure class = %s, want %s", class, wantClass)
				}
				if classified, retryable := hostruntime.ClassifyFailure(err); classified != class || retryable != wantRetryable {
					t.Fatalf("classified retryable = %t for class %s, want %t", retryable, class, wantRetryable)
				}
				if rig.input.Reads() != 0 || rig.input.BytesRead() != 0 {
					t.Fatal("an invalid configuration consumed browser input")
				}
				assertStdoutIsFrameOnly(t, name, rig, 0)
				return
			}

			session, err := rig.Bootstrap(t)
			if err != nil {
				t.Fatalf("bootstrap: %v", err)
			}
			outcomes := rig.Start(ctx, session)
			if name == "concurrency_overflow" {
				waitForCondition(t, "the concurrency cap to reject a request", func() bool {
					_, _, _, notes := rig.protocol.counts()
					for _, note := range notes {
						if note.Failure == wantClass {
							return true
						}
					}
					return false
				})
				health := session.Health()
				if health.RequestsStarted > uint64(rig.limits.MaxConcurrentRequests) {
					t.Fatalf("started requests = %d, above the concurrency cap %d", health.RequestsStarted, rig.limits.MaxConcurrentRequests)
				}
				_, _, _, notifications := rig.protocol.counts()
				if via != "notification" {
					t.Fatalf("case %q is observed through %q, not a notification", name, via)
				}
				sawRejection := false
				for _, notification := range notifications {
					if notification.Failure == wantClass {
						sawRejection = true
						if notification.Retryable != wantRetryable {
							t.Fatalf("a %s rejection reported retryable=%t, want %t", wantClass, notification.Retryable, wantRetryable)
						}
					}
				}
				if !sawRejection {
					t.Fatalf("no %s notification was published to the protocol owner", wantClass)
				}
				session.Close(nil)
				awaitOutcome(t, outcomes, rig.limits.ShutdownTimeout+slack)
				assertStdoutIsFrameOnly(t, name, rig, 0)
				return
			}

			outcome := awaitOutcome(t, outcomes, rig.limits.ShutdownTimeout+slack)
			if outcome.Failure != wantClass {
				t.Fatalf("failure class = %s, want %s", outcome.Failure, wantClass)
			}
			if outcome.Retryable != wantRetryable {
				t.Fatalf("classified retryable = %t for class %s, want %t", outcome.Retryable, outcome.Failure, wantRetryable)
			}
			if via != "outcome" {
				t.Fatalf("case %q is observed through %q, not an outcome", name, via)
			}
			if name == "oversized_frame" {
				if asked := int64(rig.input.MaxRequestedBuffer()); asked != expectedIntOf(t, fixture, "max_read_requested_bytes_for_oversized_frame") {
					t.Fatalf("largest read buffer requested = %d, want %d so the limit is enforced before allocation", asked, expectedIntOf(t, fixture, "max_read_requested_bytes_for_oversized_frame"))
				}
			}
			if wantRetryable != expectedBoolOf(t, fixture, "transport_rejections_retryable") {
				t.Fatalf("transport rejection retryable=%t contradicts the fixture", wantRetryable)
			}
			for _, state := range expectedStringsOf(t, fixture, "availability_path_contains") {
				if !rig.states.WithState(hostruntime.LifecycleState(state)) {
					t.Fatalf("path %v does not contain %s", rig.states.Path(), state)
				}
			}
			if expectedIntOf(t, fixture, "mutations_after_rejection") != 0 {
				t.Fatal("mutations_after_rejection must stay observable at zero")
			}
			assertStdoutIsFrameOnly(t, name, rig, expectedIntOf(t, fixture, "stdout_frames"))
			assertSentinelsAbsent(t, name, rig)
		})
	}
}

func TestFixtureHostNoLoopback(t *testing.T) {
	ctx := context.Background()
	corpus := LoadCorpus(t)
	fixture := corpus.Fixture(t, "FX-HOST-NO-LOOPBACK")

	guarded := expectedStringsOf(t, fixture, "guarded_source_packages")
	if imports := hostRuntimeImports(t, guarded); len(imports) != int(expectedIntOf(t, fixture, "network_packages_imported_by_host_runtime")) {
		t.Fatalf("host runtime imports a guarded package: %v", imports)
	}
	session, rig, outcomes := startHealthySession(t, ctx, fixture)
	if health := session.Health(); health.Availability != hostruntime.Availability(expectedStringOf(t, fixture, "availability_at_ready")) {
		t.Fatalf("availability = %s, want healthy", health.Availability)
	}
	if expectedIntOf(t, fixture, "child_process_spawn_attempts") != 0 {
		t.Fatal("child process spawn attempts must remain unobservable")
	}
	session.Close(nil)
	if outcome := awaitOutcome(t, outcomes, rig.limits.ShutdownTimeout+slack); !outcome.Clean {
		t.Fatalf("outcome = %+v, want clean", outcome)
	}
	if got := expectedStringsOf(t, fixture, "transports"); len(got) != 2 || got[0] != "stdin" || got[1] != "stdout" {
		t.Fatalf("declared transports = %v, want the browser-provided pair only", got)
	}
	if rig.input.Reads() == 0 {
		t.Fatal("no browser input was consumed, so the transport seam was not exercised")
	}
	assertStdoutIsFrameOnly(t, "no-loopback", rig, expectedIntOf(t, fixture, "stdout_frames"))
	assertSentinelsAbsent(t, "no-loopback", rig)
}

// hostRuntimeImports is the static no-loopback guard: it walks the host module
// source and reports any import of a network or process package.
func hostRuntimeImports(t *testing.T, guarded []string) []string {
	t.Helper()
	blocked := map[string]bool{}
	for _, name := range guarded {
		blocked[name] = true
	}
	root := filepath.Clean("../../../host")
	entries, err := os.ReadDir(root)
	if err != nil {
		t.Fatal(err)
	}
	found := []string{}
	fileSet := token.NewFileSet()
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		pkgs, err := parser.ParseDir(fileSet, filepath.Join(root, entry.Name()), nil, parser.ImportsOnly)
		if err != nil {
			t.Fatalf("parse %s: %v", entry.Name(), err)
		}
		for path, pkg := range pkgs {
			for _, file := range pkg.Files {
				for _, spec := range file.Imports {
					value, err := strconv.Unquote(spec.Path.Value)
					if err != nil {
						continue
					}
					if blocked[value] {
						found = append(found, filepath.Join(path, value))
					}
				}
			}
		}
	}
	return found
}

// assertWriterRefusesAfterClose proves the single serialized writer released the
// browser-provided stream: any later frame fails closed instead of being emitted.
func assertWriterRefusesAfterClose(t *testing.T, key string, session *hostruntime.Session) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	err := session.WriteFrame(ctx, []byte(`{"type":"late"}`))
	if err == nil {
		t.Fatalf("%s: a frame accepted after the session closed would contaminate stdout", key)
	}
	class, _ := hostruntime.ClassifyFailure(err)
	if class != hostruntime.FailureTransportUnavailable {
		t.Fatalf("%s: post-close frame class = %s, want transport_unavailable", key, class)
	}
}

func frameHeader(size uint32) []byte {
	header := make([]byte, 4)
	binary.LittleEndian.PutUint32(header, size)
	return header
}

func scaleLimits(limits hostruntime.Limits, name string, budget time.Duration) hostruntime.Limits {
	switch name {
	case "startup_budget_ms":
		limits.StartupTimeout = budget
	case "handshake_budget_ms":
		limits.HandshakeTimeout = budget
	case "synchronization_budget_ms":
		limits.SynchronizationTimeout = budget
	case "request_budget_ms":
		limits.RequestTimeout = budget
	case "shutdown_budget_ms":
		limits.ShutdownTimeout = budget
	}
	return limits
}

func zeroLimits(limits hostruntime.Limits, zeroed map[string]int64) hostruntime.Limits {
	limits.MaxFrameBytes = int(zeroed["max_frame_bytes"])
	limits.MaxQueuedFrames = int(zeroed["max_queued_frames"])
	limits.MaxConcurrentRequests = int(zeroed["max_concurrent_requests"])
	limits.MaxDiagnostics = int(zeroed["max_diagnostic_records"])
	limits.MaxDiagnosticBytes = int(zeroed["max_diagnostic_bytes"])
	limits.StartupTimeout = 0
	limits.HandshakeTimeout = 0
	limits.SynchronizationTimeout = 0
	limits.RequestTimeout = 0
	limits.ShutdownTimeout = 0
	return limits
}

func waitForCondition(t *testing.T, what string, ready func() bool) {
	t.Helper()
	deadline := time.Now().Add(30 * time.Second)
	for time.Now().Before(deadline) {
		if ready() {
			return
		}
		time.Sleep(time.Millisecond)
	}
	t.Fatalf("timed out waiting for %s", what)
}
