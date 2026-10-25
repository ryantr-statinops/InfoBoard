package phase12

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strconv"
	"testing"
	"time"

	"github.com/ryantr-statinops/InfoBoard/host/internal/domain"
	"github.com/ryantr-statinops/InfoBoard/host/internal/index"
	"github.com/ryantr-statinops/InfoBoard/host/internal/protocol"
	"github.com/ryantr-statinops/InfoBoard/host/internal/query"
	"github.com/ryantr-statinops/InfoBoard/host/internal/rank"
)

type fixtureObservation struct {
	Signal string          `json:"signal"`
	Value  json.RawMessage `json:"value"`
}

type fixtureExpected struct {
	Observations      []fixtureObservation `json:"observations"`
	PrivacyAssertions []string             `json:"privacy_assertions"`
}

type fixtureCase struct {
	Query         string  `json:"query"`
	TabIDs        []int64 `json:"tab_ids"`
	QuerySequence uint64  `json:"query_sequence"`
	ResultLimit   *int    `json:"result_limit"`
	AcceptedLimit *int    `json:"accepted_limit"`
	ResultCount   *int    `json:"result_count"`
	ErrorCode     string  `json:"error_code"`
	Status        string  `json:"status"`
}

type fixtureInput struct {
	ProfileID          string           `json:"profile_id"`
	ProjectionRevision uint64           `json:"projection_revision"`
	IndexRevision      uint64           `json:"index_revision"`
	RankingTimestamp   int64            `json:"ranking_timestamp"`
	Query              string           `json:"query"`
	ResultLimit        *int             `json:"result_limit"`
	CurrentWindowID    int64            `json:"current_window_id"`
	CurrentGroupID     *int64           `json:"current_group_id"`
	QuerySequence      uint64           `json:"query_sequence"`
	PersistenceState   string           `json:"persistence_state"`
	Scenario           string           `json:"scenario"`
	Records            []fixtureRecord  `json:"records"`
	Recency            map[string]int64 `json:"recency"`
	DefaultResultLimit int              `json:"default_result_limit"`
	Cases              []fixtureCase    `json:"cases"`
}

type fixtureRecord struct {
	TabID              int64   `json:"tab_id"`
	ProfileID          string  `json:"profile_id"`
	Revision           uint64  `json:"revision"`
	Eligible           bool    `json:"eligible"`
	Pinned             bool    `json:"pinned"`
	Active             bool    `json:"active"`
	WindowID           int64   `json:"window_id"`
	GroupID            *int64  `json:"group_id"`
	Title              string  `json:"title"`
	URL                string  `json:"url"`
	Domain             string  `json:"domain"`
	WindowLabel        string  `json:"window_label"`
	GroupLabel         string  `json:"group_label"`
	TitleDisplay       string  `json:"title_display"`
	URLDisplay         string  `json:"url_display"`
	DomainDisplay      string  `json:"domain_display"`
	WindowLabelDisplay *string `json:"window_label_display"`
	GroupLabelDisplay  *string `json:"group_label_display"`
}

type fixtureFile struct {
	FixtureID      string          `json:"fixture_id"`
	OwnerPhase     string          `json:"owner_phase"`
	RequirementIDs []string        `json:"requirement_ids"`
	Category       string          `json:"category"`
	Input          fixtureInput    `json:"input"`
	Expected       fixtureExpected `json:"expected"`
}

var queryFixtureIDs = []string{
	"FX-QUERY-FIELD-MATCH",
	"FX-QUERY-LIVE-REVISIONS",
	"FX-QUERY-EMPTY-NO-RESULTS",
	"FX-QUERY-INVALID-OVERLONG",
	"FX-QUERY-RESULT-LIMIT",
	"FX-QUERY-UNKNOWN-REVISION",
	"FX-QUERY-REBUILDING-REVISION",
	"FX-QUERY-HOST-DOWN-RECOVERY",
	"FX-QUERY-PERSISTENCE-DEGRADED",
	"FX-QUERY-DETERMINISTIC-HANDOFF",
	"FX-QUERY-ADVERSARIAL-BOUNDS",
	"FX-QUERY-1000-TABS-64-CHARS",
}

func loadQueryFixtures(t testing.TB) map[string]fixtureFile {
	t.Helper()
	paths, err := filepath.Glob(filepath.Join("..", "..", "..", "fixtures", "query", "phase-12", "*.json"))
	if err != nil {
		t.Fatalf("glob fixtures: %v", err)
	}
	if len(paths) != len(queryFixtureIDs) {
		t.Fatalf("fixture files = %d, want %d", len(paths), len(queryFixtureIDs))
	}
	fixtures := make(map[string]fixtureFile, len(paths))
	for _, path := range paths {
		raw, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("read %s: %v", path, err)
		}
		var fixture fixtureFile
		if err := json.Unmarshal(raw, &fixture); err != nil {
			t.Fatalf("decode %s: %v", path, err)
		}
		if fixture.FixtureID == "" || fixture.OwnerPhase != "IP-12" {
			t.Fatalf("%s has invalid ownership metadata", path)
		}
		if len(fixture.Expected.Observations) == 0 || len(fixture.Expected.PrivacyAssertions) == 0 {
			t.Fatalf("%s has no observable/privacy outcome", path)
		}
		fixtures[fixture.FixtureID] = fixture
	}
	for _, id := range queryFixtureIDs {
		if _, ok := fixtures[id]; !ok {
			t.Fatalf("missing fixture %s", id)
		}
	}
	return fixtures
}

func observationMap(t *testing.T, fixture fixtureFile) map[string]json.RawMessage {
	t.Helper()
	values := make(map[string]json.RawMessage, len(fixture.Expected.Observations))
	for _, observation := range fixture.Expected.Observations {
		if observation.Signal == "" {
			t.Fatalf("%s has an empty observation signal", fixture.FixtureID)
		}
		values[observation.Signal] = observation.Value
	}
	return values
}

func observationValue[T any](t *testing.T, values map[string]json.RawMessage, signal string) T {
	t.Helper()
	raw, ok := values[signal]
	if !ok {
		t.Fatalf("missing observation %s", signal)
	}
	var value T
	if err := json.Unmarshal(raw, &value); err != nil {
		t.Fatalf("decode observation %s: %v", signal, err)
	}
	return value
}

func (record fixtureRecord) toIndex(profileID string, revision uint64) index.Record {
	if record.Revision == 0 {
		record.Revision = revision
	}
	if record.ProfileID == "" {
		record.ProfileID = profileID
	}
	return index.Record{
		TabID:              record.TabID,
		ProfileID:          record.ProfileID,
		Revision:           record.Revision,
		Eligible:           record.Eligible,
		Pinned:             record.Pinned,
		Active:             record.Active,
		WindowID:           record.WindowID,
		GroupID:            record.GroupID,
		Title:              record.Title,
		URL:                record.URL,
		Domain:             record.Domain,
		WindowLabel:        record.WindowLabel,
		GroupLabel:         record.GroupLabel,
		TitleDisplay:       record.TitleDisplay,
		URLDisplay:         record.URLDisplay,
		DomainDisplay:      record.DomainDisplay,
		WindowLabelDisplay: record.WindowLabelDisplay,
		GroupLabelDisplay:  record.GroupLabelDisplay,
	}
}

func buildFixtureState(t *testing.T, input fixtureInput) *index.State {
	t.Helper()
	revision := input.IndexRevision
	if revision == 0 {
		revision = input.ProjectionRevision
	}
	records := make([]index.Record, 0, len(input.Records))
	for _, record := range input.Records {
		records = append(records, record.toIndex(input.ProfileID, revision))
	}
	staged, err := index.Stage(input.ProfileID, revision, records)
	if err != nil {
		t.Fatalf("stage fixture: %v", err)
	}
	return staged.Commit()
}

type fixtureSource struct {
	state      *index.State
	ready      uint64
	readyOK    bool
	rebuilding bool
}

func (source fixtureSource) State() *index.State { return source.state }
func (source fixtureSource) Ready() (uint64, bool) {
	return source.ready, source.readyOK
}
func (source fixtureSource) Rebuilding() bool { return source.rebuilding }

type fixtureHealth string

func (health fixtureHealth) PersistenceState() string { return string(health) }

type fixtureDefaults map[string]int

func (defaults fixtureDefaults) ResultLimit(profileID string) (int, bool) {
	value, ok := defaults[profileID]
	return value, ok
}

type fixtureRecency map[int64]int64

func (recency fixtureRecency) ActivatedAt(_ string, tabID int64) (int64, bool) {
	value, ok := recency[tabID]
	return value, ok
}

func fixtureService(t *testing.T, input fixtureInput, source query.Source) *query.Service {
	t.Helper()
	var defaults query.Defaults
	if input.DefaultResultLimit > 0 {
		defaults = fixtureDefaults{input.ProfileID: input.DefaultResultLimit}
	}
	var recency query.Recency
	if len(input.Recency) > 0 {
		values := make(fixtureRecency, len(input.Recency))
		for key, value := range input.Recency {
			tabID, err := strconv.ParseInt(key, 10, 64)
			if err != nil {
				t.Fatalf("invalid recency tab id %q", key)
			}
			values[tabID] = value
		}
		recency = values
	}
	health := fixtureHealth(input.PersistenceState)
	if health == "" {
		health = "ready"
	}
	timestamp := input.RankingTimestamp
	if timestamp == 0 {
		timestamp = 1000
	}
	service, err := query.New(source, defaults, recency, health, nil, query.Config{
		FallbackResultLimit: 5,
		MaxResultLimit:      5,
		RankingModelVersion: rank.ModelVersion,
		Now:                 func() time.Time { return time.Unix(timestamp, 0) },
	})
	if err != nil {
		t.Fatalf("new query service: %v", err)
	}
	return service
}

func fixtureRequest(input fixtureInput, text string, sequence uint64, limit *int) protocol.QueryRequest {
	request := protocol.QueryRequest{
		Context: protocol.RequestContext{
			ProfileID:          domain.ProfileID(input.ProfileID),
			ContextKind:        domain.ContextNormal,
			ProjectionRevision: input.ProjectionRevision,
			ReceivedAt:         time.Unix(input.RankingTimestamp, 0),
		},
		Text:            text,
		CurrentWindowID: input.CurrentWindowID,
		CurrentGroupID:  input.CurrentGroupID,
		QuerySequence:   sequence,
	}
	if limit != nil {
		request.ResultLimit = *limit
		request.HasResultLimit = true
	}
	return request
}

func resultTabIDs(result protocol.QueryResult) []int64 {
	ids := make([]int64, 0, len(result.Results))
	for _, row := range result.Results {
		ids = append(ids, row.TabID)
	}
	return ids
}

func requireTabs(t *testing.T, got, want []int64) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("tab ids = %v, want %v", got, want)
	}
	for index := range got {
		if got[index] != want[index] {
			t.Fatalf("tab ids = %v, want %v", got, want)
		}
	}
}

func requireCode(t *testing.T, err error, want protocol.ErrorCode) {
	t.Helper()
	var failure *protocol.Error
	if !errors.As(err, &failure) || failure.Code != want {
		t.Fatalf("error = %v, want %s", err, want)
	}
}

func TestQueryFixtureCatalogIsComplete(t *testing.T) {
	fixtures := loadQueryFixtures(t)
	for _, fixture := range fixtures {
		if len(fixture.RequirementIDs) == 0 {
			t.Fatalf("%s has no requirements", fixture.FixtureID)
		}
		if fixture.Category != "query" && fixture.Category != "recovery" {
			t.Fatalf("%s has unexpected category %q", fixture.FixtureID, fixture.Category)
		}
	}
}

func TestQueryFieldMatchFixture(t *testing.T) {
	fixtures := loadQueryFixtures(t)
	fixture := fixtures["FX-QUERY-FIELD-MATCH"]
	state := buildFixtureState(t, fixture.Input)
	source := fixtureSource{state: state, ready: state.Revision, readyOK: true}
	service := fixtureService(t, fixture.Input, source)
	values := observationMap(t, fixture)
	expected := observationValue[[]struct {
		Query  string  `json:"query"`
		TabIDs []int64 `json:"tab_ids"`
	}](t, values, "case_results")
	if len(expected) != len(fixture.Input.Cases) {
		t.Fatalf("expected cases = %d, cases = %d", len(expected), len(fixture.Input.Cases))
	}
	for index, item := range fixture.Input.Cases {
		result, err := service.Query(context.Background(), fixtureRequest(fixture.Input, item.Query, 1, fixture.Input.ResultLimit))
		if err != nil {
			t.Fatalf("%s query %q: %v", fixture.FixtureID, item.Query, err)
		}
		if result.Status != protocol.QueryStatusResults {
			t.Fatalf("%s query %q status = %s", fixture.FixtureID, item.Query, result.Status)
		}
		requireTabs(t, resultTabIDs(result), expected[index].TabIDs)
	}
}

func TestQueryEmptyAndNoResultsFixture(t *testing.T) {
	fixtures := loadQueryFixtures(t)
	fixture := fixtures["FX-QUERY-EMPTY-NO-RESULTS"]
	state := buildFixtureState(t, fixture.Input)
	source := fixtureSource{state: state, ready: state.Revision, readyOK: true}
	service := fixtureService(t, fixture.Input, source)
	values := observationMap(t, fixture)

	empty, err := service.Query(context.Background(), fixtureRequest(fixture.Input, "", 1, fixture.Input.ResultLimit))
	if err != nil {
		t.Fatal(err)
	}
	if string(empty.Status) != observationValue[string](t, values, "empty_status") {
		t.Fatalf("empty status = %s", empty.Status)
	}
	if len(empty.Results) != observationValue[int](t, values, "empty_context_count") {
		t.Fatalf("empty context count = %d", len(empty.Results))
	}
	if len(empty.Results) == 0 || empty.Results[0].TabID != observationValue[int64](t, values, "empty_context_first_tab_id") {
		t.Fatalf("empty context first = %+v", empty.Results)
	}

	missing, err := service.Query(context.Background(), fixtureRequest(fixture.Input, "missing", 2, fixture.Input.ResultLimit))
	if err != nil {
		t.Fatal(err)
	}
	if string(missing.Status) != observationValue[string](t, values, "no_result_status") {
		t.Fatalf("no-result status = %s", missing.Status)
	}
	if len(missing.Results) != observationValue[int](t, values, "no_result_count") {
		t.Fatalf("no-result count = %d", len(missing.Results))
	}
}

func TestQueryInvalidAndOverlongFixture(t *testing.T) {
	fixtures := loadQueryFixtures(t)
	fixture := fixtures["FX-QUERY-INVALID-OVERLONG"]
	state := buildFixtureState(t, fixture.Input)
	source := fixtureSource{state: state, ready: state.Revision, readyOK: true}
	service := fixtureService(t, fixture.Input, source)
	values := observationMap(t, fixture)

	for index, item := range fixture.Input.Cases {
		_, err := service.Query(context.Background(), fixtureRequest(fixture.Input, item.Query, uint64(index+1), fixture.Input.ResultLimit))
		requireCode(t, err, protocol.ErrorCode(item.ErrorCode))
	}
	if state.Count != len(fixture.Input.Records) {
		t.Fatalf("index mutated: count = %d", state.Count)
	}
	if observationValue[bool](t, values, "index_mutation") {
		t.Fatal("fixture unexpectedly asserts index mutation")
	}
}

func TestQueryResultLimitFixture(t *testing.T) {
	fixtures := loadQueryFixtures(t)
	fixture := fixtures["FX-QUERY-RESULT-LIMIT"]
	state := buildFixtureState(t, fixture.Input)
	source := fixtureSource{state: state, ready: state.Revision, readyOK: true}
	service := fixtureService(t, fixture.Input, source)
	values := observationMap(t, fixture)

	for _, item := range fixture.Input.Cases {
		result, err := service.Query(context.Background(), fixtureRequest(fixture.Input, fixture.Input.Query, 1, item.ResultLimit))
		if item.ErrorCode != "" {
			requireCode(t, err, protocol.ErrorCode(item.ErrorCode))
			continue
		}
		if err != nil {
			t.Fatalf("limit %v: %v", item.ResultLimit, err)
		}
		if item.AcceptedLimit != nil && result.AcceptedLimit != *item.AcceptedLimit {
			t.Fatalf("accepted limit = %d, want %d", result.AcceptedLimit, *item.AcceptedLimit)
		}
		if item.ResultCount != nil && len(result.Results) != *item.ResultCount {
			t.Fatalf("result count = %d, want %d", len(result.Results), *item.ResultCount)
		}
	}
	if observationValue[int](t, values, "omitted_limit") != 5 {
		t.Fatal("fixture omitted limit changed")
	}
}

func TestQueryRevisionAndRebuildingFixtures(t *testing.T) {
	fixtures := loadQueryFixtures(t)

	unknown := fixtures["FX-QUERY-UNKNOWN-REVISION"]
	unknownState := buildFixtureState(t, unknown.Input)
	unknownService := fixtureService(t, unknown.Input, fixtureSource{state: unknownState, ready: unknownState.Revision, readyOK: true})
	_, err := unknownService.Query(context.Background(), fixtureRequest(unknown.Input, unknown.Input.Query, 1, unknown.Input.ResultLimit))
	requireCode(t, err, protocol.CodeRevisionMismatch)
	var failure *protocol.Error
	if !errors.As(err, &failure) || failure.ExpectedRevision == nil || *failure.ExpectedRevision != unknownState.Revision {
		t.Fatalf("revision expectation = %+v", failure)
	}

	rebuilding := fixtures["FX-QUERY-REBUILDING-REVISION"]
	rebuildingState := buildFixtureState(t, rebuilding.Input)
	rebuildingService := fixtureService(t, rebuilding.Input, fixtureSource{state: rebuildingState, ready: rebuildingState.Revision, readyOK: true, rebuilding: true})
	_, err = rebuildingService.Query(context.Background(), fixtureRequest(rebuilding.Input, rebuilding.Input.Query, 1, rebuilding.Input.ResultLimit))
	requireCode(t, err, protocol.CodeIndexRebuilding)
}

func TestQueryPersistenceAndDeterministicFixtures(t *testing.T) {
	fixtures := loadQueryFixtures(t)

	degraded := fixtures["FX-QUERY-PERSISTENCE-DEGRADED"]
	degradedState := buildFixtureState(t, degraded.Input)
	degradedService := fixtureService(t, degraded.Input, fixtureSource{state: degradedState, ready: degradedState.Revision, readyOK: true})
	result, err := degradedService.Query(context.Background(), fixtureRequest(degraded.Input, degraded.Input.Query, 1, degraded.Input.ResultLimit))
	if err != nil {
		t.Fatal(err)
	}
	if result.Status != protocol.QueryStatusResults || result.PersistenceState != "degraded" || result.Counters.DegradedCode != protocol.CodePersistenceDegraded {
		t.Fatalf("degraded result = %+v", result)
	}

	deterministic := fixtures["FX-QUERY-DETERMINISTIC-HANDOFF"]
	deterministicState := buildFixtureState(t, deterministic.Input)
	deterministicService := fixtureService(t, deterministic.Input, fixtureSource{state: deterministicState, ready: deterministicState.Revision, readyOK: true})
	first, err := deterministicService.Query(context.Background(), fixtureRequest(deterministic.Input, deterministic.Input.Query, 1, deterministic.Input.ResultLimit))
	if err != nil {
		t.Fatal(err)
	}
	second, err := deterministicService.Query(context.Background(), fixtureRequest(deterministic.Input, deterministic.Input.Query, 2, deterministic.Input.ResultLimit))
	if err != nil {
		t.Fatal(err)
	}
	if len(first.Results) != len(second.Results) {
		t.Fatalf("deterministic row count differs: %d vs %d", len(first.Results), len(second.Results))
	}
	for index := range first.Results {
		if first.Results[index].ResultID != second.Results[index].ResultID {
			t.Fatalf("result id differs at %d", index)
		}
	}
	if first.ProjectionRevision != deterministic.Input.ProjectionRevision {
		t.Fatalf("projection revision = %d", first.ProjectionRevision)
	}
}

func TestQueryAdversarialBoundsFixture(t *testing.T) {
	fixtures := loadQueryFixtures(t)
	fixture := fixtures["FX-QUERY-ADVERSARIAL-BOUNDS"]
	state := buildFixtureState(t, fixture.Input)
	source := fixtureSource{state: state, ready: state.Revision, readyOK: true}
	service := fixtureService(t, fixture.Input, source)
	values := observationMap(t, fixture)

	for index, item := range fixture.Input.Cases {
		queryText := item.Query
		if queryText == "" {
			queryText = fixture.Input.Query
		}
		_, err := service.Query(context.Background(), fixtureRequest(fixture.Input, queryText, uint64(index+1), item.ResultLimit))
		requireCode(t, err, protocol.ErrorCode(item.ErrorCode))
	}
	if state.Count != len(fixture.Input.Records) {
		t.Fatalf("index mutated: count = %d", state.Count)
	}
	if observationValue[bool](t, values, "index_mutation") {
		t.Fatal("fixture unexpectedly asserts index mutation")
	}
}

func TestQuery1000TabsFixture(t *testing.T) {
	fixtures := loadQueryFixtures(t)
	fixture := fixtures["FX-QUERY-1000-TABS-64-CHARS"]
	state := buildFixtureState(t, fixture.Input)
	if len(state.Records) != 1000 {
		t.Fatalf("records = %d", len(state.Records))
	}
	source := fixtureSource{state: state, ready: state.Revision, readyOK: true}
	service := fixtureService(t, fixture.Input, source)
	result, err := service.Query(context.Background(), fixtureRequest(fixture.Input, fixture.Input.Query, 1, fixture.Input.ResultLimit))
	if err != nil {
		t.Fatal(err)
	}
	if result.Status != protocol.QueryStatusResults || len(result.Results) != 5 {
		t.Fatalf("1000-tab result = %+v", result)
	}
	if result.Counters.Candidates != 1000 {
		t.Fatalf("candidates = %d", result.Counters.Candidates)
	}
}
