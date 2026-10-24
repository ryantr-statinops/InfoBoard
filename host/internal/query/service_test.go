package query

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/ryantr-statinops/InfoBoard/host/internal/domain"
	"github.com/ryantr-statinops/InfoBoard/host/internal/index"
	"github.com/ryantr-statinops/InfoBoard/host/internal/protocol"
)

const testProfile = "123e4567-e89b-42d3-a456-426614174000"

type fakeSource struct {
	state      *index.State
	ready      uint64
	readyOK    bool
	rebuilding bool
}

func (source *fakeSource) State() *index.State { return source.state }
func (source *fakeSource) Ready() (uint64, bool) {
	return source.ready, source.readyOK
}
func (source *fakeSource) Rebuilding() bool { return source.rebuilding }

type fakeDefaults map[string]int

func (defaults fakeDefaults) ResultLimit(profileID string) (int, bool) {
	value, ok := defaults[profileID]
	return value, ok
}

type fakeHealth string

func (health fakeHealth) PersistenceState() string { return string(health) }

type fakeRecency map[int64]int64

func (recency fakeRecency) ActivatedAt(_ string, tabID int64) (int64, bool) {
	value, ok := recency[tabID]
	return value, ok
}

type fakeObserver struct {
	observations []Observation
}

func (observer *fakeObserver) ObserveQuery(observation Observation) {
	observer.observations = append(observer.observations, observation)
}

func readySource() *fakeSource {
	return &fakeSource{state: &index.State{ProfileID: testProfile, Revision: 4}, ready: 4, readyOK: true}
}

func testService(t *testing.T, source Source, defaults Defaults, health Health, observer Observer) *Service {
	t.Helper()
	return testServiceWithRecency(t, source, defaults, nil, health, observer)
}

func testServiceWithRecency(t *testing.T, source Source, defaults Defaults, recency Recency, health Health, observer Observer) *Service {
	t.Helper()
	service, err := New(source, defaults, recency, health, observer, Config{
		FallbackResultLimit: 5,
		MaxResultLimit:      5,
		Now:                 func() time.Time { return time.Unix(100, 0) },
	})
	if err != nil {
		t.Fatal(err)
	}
	return service
}

func testRequest() protocol.QueryRequest {
	return protocol.QueryRequest{
		Context: protocol.RequestContext{
			ProfileID:          domain.ProfileID(testProfile),
			ProjectionRevision: 4,
		},
		Text:            "docs",
		ResultLimit:     5,
		HasResultLimit:  true,
		CurrentWindowID: 1,
	}
}

func failureCode(t *testing.T, err error) protocol.ErrorCode {
	t.Helper()
	var failure *protocol.Error
	if !errors.As(err, &failure) {
		t.Fatalf("error is not a protocol failure: %v", err)
	}
	return failure.Code
}

func TestNewRequiresSourceAndFallback(t *testing.T) {
	if _, err := New(nil, nil, nil, nil, nil, Config{FallbackResultLimit: 5}); err == nil {
		t.Fatal("nil source accepted")
	}
	if _, err := New(readySource(), nil, nil, nil, nil, Config{}); err == nil {
		t.Fatal("missing fallback accepted")
	}
	if _, err := New(readySource(), nil, nil, nil, nil, Config{FallbackResultLimit: 6, MaxResultLimit: 5}); err == nil {
		t.Fatal("fallback above maximum accepted")
	}
}

func TestQueryResolvesLimits(t *testing.T) {
	service := testService(t, readySource(), nil, nil, nil)
	result, err := service.Query(context.Background(), testRequest())
	if err != nil {
		t.Fatal(err)
	}
	if result.AcceptedLimit != 5 {
		t.Fatalf("explicit limit = %d", result.AcceptedLimit)
	}

	request := testRequest()
	request.HasResultLimit = false
	request.ResultLimit = 0
	result, err = service.Query(context.Background(), request)
	if err != nil {
		t.Fatal(err)
	}
	if result.AcceptedLimit != 5 {
		t.Fatalf("fallback limit = %d", result.AcceptedLimit)
	}

	service = testService(t, readySource(), fakeDefaults{testProfile: 2}, nil, nil)
	result, err = service.Query(context.Background(), request)
	if err != nil {
		t.Fatal(err)
	}
	if result.AcceptedLimit != 2 {
		t.Fatalf("profile default = %d", result.AcceptedLimit)
	}

	service = testService(t, readySource(), fakeDefaults{testProfile: 9}, nil, nil)
	result, err = service.Query(context.Background(), request)
	if err != nil {
		t.Fatal(err)
	}
	if result.AcceptedLimit != 5 {
		t.Fatalf("default cap = %d", result.AcceptedLimit)
	}
}

func TestQueryRejectsInvalidLimits(t *testing.T) {
	service := testService(t, readySource(), nil, nil, nil)
	for _, limit := range []int{0, -1, 6} {
		request := testRequest()
		request.ResultLimit = limit
		if _, err := service.Query(context.Background(), request); failureCode(t, err) != protocol.CodePayloadLimit {
			t.Fatalf("limit %d accepted", limit)
		}
	}
}

func TestQueryValidatesContextAndQuery(t *testing.T) {
	service := testService(t, readySource(), nil, nil, nil)

	request := testRequest()
	request.Context.ProfileID = ""
	if _, err := service.Query(context.Background(), request); failureCode(t, err) != protocol.CodeProfileMismatch {
		t.Fatal("empty profile accepted")
	}

	request = testRequest()
	request.CurrentWindowID = -1
	if _, err := service.Query(context.Background(), request); failureCode(t, err) != protocol.CodeInvalidFrame {
		t.Fatal("negative window accepted")
	}

	request = testRequest()
	group := int64(-1)
	request.CurrentGroupID = &group
	if _, err := service.Query(context.Background(), request); failureCode(t, err) != protocol.CodeInvalidFrame {
		t.Fatal("negative group accepted")
	}

	request = testRequest()
	request.Text = strings.Repeat("x", index.MaxQueryScalars+1)
	if _, err := service.Query(context.Background(), request); failureCode(t, err) != protocol.CodePayloadLimit {
		t.Fatal("overlong query accepted")
	}
}

func TestQueryRevisionGates(t *testing.T) {
	source := readySource()
	service := testService(t, source, nil, nil, nil)
	request := testRequest()
	request.Context.ProjectionRevision = 3
	var mismatch *protocol.Error
	if _, err := service.Query(context.Background(), request); !errors.As(err, &mismatch) ||
		mismatch.Code != protocol.CodeRevisionMismatch || mismatch.ExpectedRevision == nil || *mismatch.ExpectedRevision != 4 {
		t.Fatalf("unknown revision error = %+v", err)
	}

	source.rebuilding = true
	request = testRequest()
	if _, err := service.Query(context.Background(), request); failureCode(t, err) != protocol.CodeIndexRebuilding {
		t.Fatal("rebuilding index accepted")
	}

	source = &fakeSource{state: &index.State{ProfileID: testProfile, Revision: 4}, ready: 4, readyOK: false}
	service = testService(t, source, nil, nil, nil)
	if _, err := service.Query(context.Background(), testRequest()); failureCode(t, err) != protocol.CodeRevisionMismatch {
		t.Fatal("not-ready index accepted")
	}

	source = readySource()
	source.state.ProfileID = "other"
	service = testService(t, source, nil, nil, nil)
	if _, err := service.Query(context.Background(), testRequest()); failureCode(t, err) != protocol.CodeProfileMismatch {
		t.Fatal("profile mismatch accepted")
	}
}

func TestQueryHonorsCancellation(t *testing.T) {
	service := testService(t, readySource(), nil, nil, nil)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := service.Query(ctx, testRequest()); !errors.Is(err, context.Canceled) {
		t.Fatalf("cancelled query error = %v", err)
	}
}

func TestQueryObservesBoundedStatus(t *testing.T) {
	observer := &fakeObserver{}
	service := testService(t, readySource(), nil, fakeHealth("degraded"), observer)
	result, err := service.Query(context.Background(), testRequest())
	if err != nil {
		t.Fatal(err)
	}
	if result.Status != protocol.QueryStatusNoResults || result.PersistenceState != "degraded" || result.IndexState != "ready" {
		t.Fatalf("query status = %+v", result)
	}
	if len(observer.observations) != 1 || observer.observations[0].Status != protocol.QueryStatusNoResults {
		t.Fatalf("observations = %+v", observer.observations)
	}
}

func testState(t *testing.T, records ...index.Record) *index.State {
	t.Helper()
	staged, err := index.Stage(testProfile, 4, records)
	if err != nil {
		t.Fatal(err)
	}
	return staged.Commit()
}

func sourceFor(state *index.State) *fakeSource {
	return &fakeSource{state: state, ready: state.Revision, readyOK: true}
}

func record(id int64, title string) index.Record {
	return index.Record{
		TabID:         id,
		ProfileID:     testProfile,
		Revision:      4,
		Eligible:      true,
		Active:        id == 10,
		WindowID:      1,
		Title:         title,
		URL:           "https://example.test/docs",
		Domain:        "example.test",
		TitleDisplay:  title,
		URLDisplay:    "https://example.test/docs",
		DomainDisplay: "example.test",
	}
}

func hasString(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}

func TestQueryRanksIndexedRecords(t *testing.T) {
	group := int64(7)
	item := record(10, "example docs")
	item.GroupID = &group
	state := testState(t, item)
	service := testService(t, sourceFor(state), nil, fakeHealth("ready"), nil)
	request := testRequest()
	request.Text = "example"
	request.CurrentGroupID = &group

	result, err := service.Query(context.Background(), request)
	if err != nil {
		t.Fatal(err)
	}
	if result.Status != protocol.QueryStatusResults || len(result.Results) != 1 {
		t.Fatalf("query result = %+v", result)
	}
	row := result.Results[0]
	if row.TabID != 10 || row.WindowID != 1 || row.GroupID == nil || *row.GroupID != 7 {
		t.Fatalf("row identity = %+v", row)
	}
	if row.TitleDisplay != "example docs" || row.DomainDisplay != "example.test" || row.Score <= 0 {
		t.Fatalf("row metadata = %+v", row)
	}
	if row.ResultID == "" || len(row.ResultID) != 32 {
		t.Fatalf("result id = %q", row.ResultID)
	}
	if !hasString(row.Explanation, "title_match") || !hasString(row.Explanation, "current_window") || !hasString(row.Explanation, "same_group") {
		t.Fatalf("explanations = %v", row.Explanation)
	}
	if result.RankingModelVersion != "rank-v1" || result.AcceptedLimit != 5 || result.PersistenceState != "healthy" {
		t.Fatalf("result metadata = %+v", result)
	}
	if result.Counters.Candidates != 1 || result.Counters.Returned != 1 || result.Counters.TotalMicros < 0 {
		t.Fatalf("counters = %+v", result.Counters)
	}

	repeated, err := service.Query(context.Background(), request)
	if err != nil {
		t.Fatal(err)
	}
	if repeated.Results[0].ResultID != row.ResultID || repeated.Results[0].Score != row.Score {
		t.Fatalf("repeated result differs: %+v vs %+v", repeated.Results[0], row)
	}
}

func TestQueryCapsResultsAndReportsNoResults(t *testing.T) {
	state := testState(t, record(10, "example docs"), record(11, "example notes"))
	service := testService(t, sourceFor(state), nil, fakeHealth("ready"), nil)

	request := testRequest()
	request.Text = "example"
	request.ResultLimit = 1
	result, err := service.Query(context.Background(), request)
	if err != nil {
		t.Fatal(err)
	}
	if result.Status != protocol.QueryStatusResults || len(result.Results) != 1 {
		t.Fatalf("limited result = %+v", result)
	}

	request = testRequest()
	request.Text = "absent"
	result, err = service.Query(context.Background(), request)
	if err != nil {
		t.Fatal(err)
	}
	if result.Status != protocol.QueryStatusNoResults || len(result.Results) != 0 {
		t.Fatalf("no-result response = %+v", result)
	}
}

func TestQueryRejectsMalformedQueryWithoutMutation(t *testing.T) {
	state := testState(t, record(10, "example docs"))
	service := testService(t, sourceFor(state), nil, fakeHealth("ready"), nil)
	request := testRequest()
	request.Text = `alpha "beta`
	if _, err := service.Query(context.Background(), request); failureCode(t, err) != protocol.CodeInvalidFrame {
		t.Fatal("unbalanced phrase accepted")
	}
	request.Text = "safe\x00value"
	if _, err := service.Query(context.Background(), request); failureCode(t, err) != protocol.CodeInvalidFrame {
		t.Fatal("control character accepted")
	}
	if state.Count != 1 {
		t.Fatalf("index mutated: %+v", state)
	}
}

func TestEmptyQueryReturnsBoundedContextList(t *testing.T) {
	group7 := int64(7)
	group8 := int64(8)
	current := record(10, "current")
	current.GroupID = &group7
	current.Pinned = true
	otherWindow := record(11, "other")
	otherWindow.WindowID = 2
	otherWindow.GroupID = &group8
	sameWindow := record(12, "same-window")
	sameWindow.GroupID = &group8

	state := testState(t, current, otherWindow, sameWindow)
	service := testServiceWithRecency(t, sourceFor(state), nil, fakeRecency{10: 100, 11: 300, 12: 200}, fakeHealth("ready"), nil)
	request := testRequest()
	request.Text = "   "
	request.CurrentWindowID = 1
	request.CurrentGroupID = &group7
	request.ResultLimit = 2

	result, err := service.Query(context.Background(), request)
	if err != nil {
		t.Fatal(err)
	}
	if result.Status != protocol.QueryStatusEmptyQuery || len(result.Results) != 2 {
		t.Fatalf("empty-query result = %+v", result)
	}
	if result.Results[0].TabID != 10 || result.Results[1].TabID != 12 {
		t.Fatalf("context order = %+v", result.Results)
	}
	if result.Counters.Candidates != 3 || result.Counters.Returned != 2 {
		t.Fatalf("context counters = %+v", result.Counters)
	}
	if !hasString(result.Results[0].Explanation, "current_window") ||
		!hasString(result.Results[0].Explanation, "same_group") ||
		!hasString(result.Results[0].Explanation, "pinned") ||
		!hasString(result.Results[0].Explanation, "recent_activation") {
		t.Fatalf("context explanations = %v", result.Results[0].Explanation)
	}
	repeated, err := service.Query(context.Background(), request)
	if err != nil {
		t.Fatal(err)
	}
	if repeated.Results[0].ResultID != result.Results[0].ResultID {
		t.Fatalf("context result id is not deterministic: %q vs %q", repeated.Results[0].ResultID, result.Results[0].ResultID)
	}
}

func TestEmptyProjectionReturnsEmptyQuery(t *testing.T) {
	state := testState(t)
	service := testService(t, sourceFor(state), nil, fakeHealth("ready"), nil)
	request := testRequest()
	request.Text = ""
	result, err := service.Query(context.Background(), request)
	if err != nil {
		t.Fatal(err)
	}
	if result.Status != protocol.QueryStatusEmptyQuery || len(result.Results) != 0 {
		t.Fatalf("empty projection result = %+v", result)
	}
	if result.Counters.Candidates != 0 || result.Counters.Returned != 0 {
		t.Fatalf("empty projection counters = %+v", result.Counters)
	}
}
