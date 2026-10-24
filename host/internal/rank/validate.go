package rank

import "unicode/utf8"

// Bounds applied before any candidate is scored.
const (
	MaxCandidates       = 10000
	MaxQueryTokens      = 64
	MaxPhrases          = 8
	MaxTokenScalars     = 64
	MaxFieldTextScalars = 2048
	MaxTokensPerField   = 64
	MaxResultLimit      = 100
)

// ErrorCode is the closed set of bounded ranking input failures.
type ErrorCode string

const (
	ErrInvalidProfile     ErrorCode = "INVALID_PROFILE"
	ErrInvalidRevision    ErrorCode = "INVALID_REVISION"
	ErrInvalidModel       ErrorCode = "INVALID_MODEL_VERSION"
	ErrInvalidTimestamp   ErrorCode = "INVALID_TIMESTAMP"
	ErrInvalidLimit       ErrorCode = "INVALID_LIMIT"
	ErrEmptyQuery         ErrorCode = "EMPTY_QUERY"
	ErrTooManyQueryTokens ErrorCode = "TOO_MANY_QUERY_TOKENS"
	ErrInvalidQueryToken  ErrorCode = "INVALID_QUERY_TOKEN"
	ErrInvalidPhrase      ErrorCode = "INVALID_PHRASE"
	ErrTooManyPhrases     ErrorCode = "TOO_MANY_PHRASES"
	ErrTooManyCandidates  ErrorCode = "TOO_MANY_CANDIDATES"
	ErrInvalidIdentity    ErrorCode = "INVALID_CANDIDATE_IDENTITY"
	ErrCrossProfile       ErrorCode = "CROSS_PROFILE_CANDIDATE"
	ErrDuplicateCandidate ErrorCode = "DUPLICATE_CANDIDATE"
	ErrInvalidField       ErrorCode = "INVALID_FIELD"
	ErrDuplicateField     ErrorCode = "DUPLICATE_FIELD"
	ErrUnboundedField     ErrorCode = "UNBOUNDED_FIELD"
	ErrTooManyFieldTokens ErrorCode = "TOO_MANY_FIELD_TOKENS"
	ErrInvalidFieldToken  ErrorCode = "INVALID_FIELD_TOKEN"
	ErrInvalidActivation  ErrorCode = "INVALID_ACTIVATION"
)

// Error is a typed, redacted ranking failure. It never carries input values.
type Error struct{ Code ErrorCode }

func (e *Error) Error() string { return string(e.Code) }

// validate rejects malformed or unbounded input before scoring.
func validate(input Input) *Error {
	if input.ProfileID == "" {
		return &Error{Code: ErrInvalidProfile}
	}
	if input.ProjectionRevision == 0 {
		return &Error{Code: ErrInvalidRevision}
	}
	if input.ModelVersion != ModelVersion {
		return &Error{Code: ErrInvalidModel}
	}
	if input.RankingTimestamp < 0 {
		return &Error{Code: ErrInvalidTimestamp}
	}
	if input.Limit < 1 || input.Limit > MaxResultLimit {
		return &Error{Code: ErrInvalidLimit}
	}
	if len(input.QueryTokens) == 0 {
		return &Error{Code: ErrEmptyQuery}
	}
	if len(input.QueryTokens) > MaxQueryTokens {
		return &Error{Code: ErrTooManyQueryTokens}
	}
	for _, token := range input.QueryTokens {
		if !boundedToken(token) {
			return &Error{Code: ErrInvalidQueryToken}
		}
	}
	if len(input.Phrases) > MaxPhrases {
		return &Error{Code: ErrTooManyPhrases}
	}
	for _, phrase := range input.Phrases {
		if len(phrase) == 0 || len(phrase) > MaxQueryTokens {
			return &Error{Code: ErrInvalidPhrase}
		}
		for _, token := range phrase {
			if !boundedToken(token) {
				return &Error{Code: ErrInvalidPhrase}
			}
		}
	}
	if len(input.Candidates) > MaxCandidates {
		return &Error{Code: ErrTooManyCandidates}
	}
	seen := make(map[[2]int64]struct{}, len(input.Candidates))
	for _, candidate := range input.Candidates {
		if errorCode := validateCandidate(input.ProfileID, candidate, seen); errorCode != nil {
			return errorCode
		}
	}
	return nil
}

func validateCandidate(profileID string, candidate Candidate, seen map[[2]int64]struct{}) *Error {
	if candidate.ProfileID != profileID {
		return &Error{Code: ErrCrossProfile}
	}
	if candidate.TabID <= 0 || candidate.WindowID <= 0 {
		return &Error{Code: ErrInvalidIdentity}
	}
	if candidate.GroupID != nil && *candidate.GroupID <= 0 {
		return &Error{Code: ErrInvalidIdentity}
	}
	if candidate.ActivatedAt < 0 {
		return &Error{Code: ErrInvalidActivation}
	}
	key := [2]int64{candidate.TabID, candidate.WindowID}
	if _, exists := seen[key]; exists {
		return &Error{Code: ErrDuplicateCandidate}
	}
	seen[key] = struct{}{}

	fields := make(map[Field]struct{}, len(candidate.Fields))
	for _, field := range candidate.Fields {
		if !validField(field.Field) || field.Field == FieldState {
			return &Error{Code: ErrInvalidField}
		}
		if _, exists := fields[field.Field]; exists {
			return &Error{Code: ErrDuplicateField}
		}
		fields[field.Field] = struct{}{}
		if !utf8.ValidString(field.Text) || utf8.RuneCountInString(field.Text) > MaxFieldTextScalars {
			return &Error{Code: ErrUnboundedField}
		}
		if len(field.Tokens) > MaxTokensPerField {
			return &Error{Code: ErrTooManyFieldTokens}
		}
		for _, token := range field.Tokens {
			if !boundedToken(token) {
				return &Error{Code: ErrInvalidFieldToken}
			}
		}
	}
	return nil
}

func validField(field Field) bool {
	switch field {
	case FieldTitle, FieldDomain, FieldURL, FieldWindow, FieldGroup, FieldState:
		return true
	default:
		return false
	}
}

func boundedToken(token string) bool {
	return token != "" && utf8.ValidString(token) && utf8.RuneCountInString(token) <= MaxTokenScalars
}
