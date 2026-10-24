package query

import (
	"context"
	"errors"
	"strings"
	"testing"

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
	service, err := New(source, defaults, nil, health, observer, Config{
		FallbackResultLimit: 5,
		MaxResultLimit:      5,
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
	if _, err := service.Query(context.Background(), request); failureCode(t, err) != protocol.CodeRevisionMismatch {
		t.Fatal("unknown revision accepted")
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
