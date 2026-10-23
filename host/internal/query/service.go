package query

// Package query owns query API and degraded results (IP-12).
type Service struct {
	readyRevision int64
}

func New() *Service { return &Service{} }

// SetReadyRevision records the currently published revision.
func (s *Service) SetReadyRevision(rev int64) { s.readyRevision = rev }

// IsRevisionReady reports whether the requested revision is ready.
func (s *Service) IsRevisionReady(rev int64) bool { return rev == s.readyRevision }
