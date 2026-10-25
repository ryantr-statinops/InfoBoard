package query

import (
	"context"
	"errors"
	"time"
	"unicode/utf8"

	"github.com/ryantr-statinops/InfoBoard/host/internal/index"
	"github.com/ryantr-statinops/InfoBoard/host/internal/protocol"
	"github.com/ryantr-statinops/InfoBoard/host/internal/rank"
)

// Service orchestrates bounded lexical queries over one published index state.
type Service struct {
	source   Source
	defaults Defaults
	recency  Recency
	health   Health
	observer Observer
	config   Config
}

// New validates and builds the query service. FallbackResultLimit must come
// from the IP-08 safe default; query does not redefine durable defaults.
func New(source Source, defaults Defaults, recency Recency, health Health, observer Observer, config Config) (*Service, error) {
	if source == nil {
		return nil, errors.New("query: source required")
	}
	if config.FallbackResultLimit < 1 {
		return nil, errors.New("query: fallback result limit required")
	}
	if config.MaxResultLimit <= 0 || config.MaxResultLimit > protocol.MaxResults {
		config.MaxResultLimit = protocol.MaxResults
	}
	if config.FallbackResultLimit > config.MaxResultLimit {
		return nil, errors.New("query: fallback result limit exceeds maximum")
	}
	if config.RankingModelVersion == "" {
		config.RankingModelVersion = rank.ModelVersion
	}
	if config.Now == nil {
		config.Now = time.Now
	}
	return &Service{
		source:   source,
		defaults: defaults,
		recency:  recency,
		health:   health,
		observer: observer,
		config:   config,
	}, nil
}

// Query validates the request before any index work and returns a bounded
// result bound to the current profile and projection revision.
func (s *Service) Query(ctx context.Context, request protocol.QueryRequest) (protocol.QueryResult, error) {
	started := time.Now()
	if err := validateRequest(request); err != nil {
		return protocol.QueryResult{}, err
	}
	if err := ctx.Err(); err != nil {
		return protocol.QueryResult{}, err
	}
	limit, failure := s.resolveLimit(request)
	if failure != nil {
		return protocol.QueryResult{}, failure
	}
	state, failure := s.readyState(request)
	if failure != nil {
		return protocol.QueryResult{}, failure
	}
	if err := ctx.Err(); err != nil {
		return protocol.QueryResult{}, err
	}

	prepared, normalizeFailure := index.PrepareQuery(request.Text)
	if normalizeFailure != nil {
		return protocol.QueryResult{}, queryNormalizeFailure(normalizeFailure, request.Text)
	}
	result := protocol.QueryResult{
		Results:             make([]protocol.ResultRow, 0),
		RankingModelVersion: s.config.RankingModelVersion,
		ProjectionRevision:  request.Context.ProjectionRevision,
		Status:              protocol.QueryStatusNoResults,
		AcceptedLimit:       limit,
		QuerySequence:       request.QuerySequence,
		IndexState:          "ready",
		PersistenceState:    s.persistenceState(),
		FreshnessAgeMillis:  0,
	}
	if prepared.Empty {
		rows, candidates := s.contextRows(state, request, limit)
		result.Results = rows
		result.Status = protocol.QueryStatusEmptyQuery
		result.Counters.Candidates = int64(candidates)
		result.Counters.Returned = int64(len(rows))
		result.Counters.TotalMicros = time.Since(started).Microseconds()
		s.annotatePersistence(&result)
		s.observe(result)
		return result, nil
	}

	candidates, err := s.candidates(state, request)
	if err != nil {
		return protocol.QueryResult{}, protocol.NewError(protocol.CodeInternalFailure)
	}
	output, err := rank.Rank(rank.Input{
		ProfileID:          state.ProfileID,
		ProjectionRevision: state.Revision,
		ModelVersion:       s.config.RankingModelVersion,
		RankingTimestamp:   s.config.Now().Unix(),
		QueryTokens:        prepared.Tokens,
		Phrases:            prepared.Phrases,
		CurrentWindowID:    request.CurrentWindowID,
		CurrentGroupID:     rankGroupID(request.CurrentGroupID),
		Limit:              limit,
		Candidates:         candidates,
	})
	if err != nil {
		return protocol.QueryResult{}, rankFailure(err)
	}
	if err := ctx.Err(); err != nil {
		return protocol.QueryResult{}, err
	}

	result.Results = resultRows(state, output.Results)
	if len(result.Results) > 0 {
		result.Status = protocol.QueryStatusResults
	}
	result.RankingModelVersion = output.ModelVersion
	result.Counters.Candidates = int64(len(candidates))
	result.Counters.Returned = int64(len(result.Results))
	result.Counters.TotalMicros = time.Since(started).Microseconds()
	s.annotatePersistence(&result)
	s.observe(result)
	return result, nil
}

// annotatePersistence keeps SQLite health observable without making it a
// prerequisite for the in-memory lexical path.
func (s *Service) annotatePersistence(result *protocol.QueryResult) {
	if result.PersistenceState == "degraded" {
		result.Counters.DegradedCode = protocol.CodePersistenceDegraded
	}
}

func queryNormalizeFailure(failure *index.NormalizeError, raw string) *protocol.Error {
	if failure.Code == index.ErrQueryTooLong {
		return protocol.NewError(protocol.CodePayloadLimit).WithBound("query_scalars", index.MaxQueryScalars, utf8.RuneCountInString(raw))
	}
	return protocol.NewError(protocol.CodeInvalidFrame)
}

func rankFailure(err error) *protocol.Error {
	var failure *rank.Error
	if errors.As(err, &failure) && failure.Code == rank.ErrEmptyQuery {
		return protocol.NewError(protocol.CodeInvalidFrame)
	}
	return protocol.NewError(protocol.CodeInternalFailure)
}

func validateRequest(request protocol.QueryRequest) *protocol.Error {
	if request.Context.ProfileID == "" {
		return protocol.NewError(protocol.CodeProfileMismatch)
	}
	if request.CurrentWindowID < 0 {
		return protocol.NewError(protocol.CodeInvalidFrame)
	}
	if request.CurrentGroupID != nil && *request.CurrentGroupID < 0 {
		return protocol.NewError(protocol.CodeInvalidFrame)
	}
	if utf8.RuneCountInString(request.Text) > index.MaxQueryScalars {
		return protocol.NewError(protocol.CodePayloadLimit).WithBound("query_scalars", index.MaxQueryScalars, utf8.RuneCountInString(request.Text))
	}
	return nil
}

func (s *Service) readyState(request protocol.QueryRequest) (*index.State, *protocol.Error) {
	state := s.source.State()
	if state == nil {
		return nil, protocol.NewError(protocol.CodeRevisionMismatch)
	}
	if s.source.Rebuilding() {
		return nil, protocol.NewError(protocol.CodeIndexRebuilding)
	}
	revision, ready := s.source.Ready()
	if !ready || revision != request.Context.ProjectionRevision || state.Revision != request.Context.ProjectionRevision {
		expected := state.Revision
		return nil, protocol.NewError(protocol.CodeRevisionMismatch).WithExpectation(&expected, nil)
	}
	if state.ProfileID != string(request.Context.ProfileID) {
		return nil, protocol.NewError(protocol.CodeProfileMismatch)
	}
	return state, nil
}

func (s *Service) persistenceState() string {
	if s.health == nil {
		return "unknown"
	}
	state := s.health.PersistenceState()
	if state == "ready" {
		return "healthy"
	}
	if state == "" {
		return "unknown"
	}
	return state
}

func (s *Service) observe(result protocol.QueryResult) {
	if s.observer == nil {
		return
	}
	s.observer.ObserveQuery(Observation{
		Status:           result.Status,
		Candidates:       int(result.Counters.Candidates),
		Returned:         len(result.Results),
		TotalMicros:      result.Counters.TotalMicros,
		PersistenceState: result.PersistenceState,
	})
}
