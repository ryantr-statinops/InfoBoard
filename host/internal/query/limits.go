package query

import (
	"github.com/ryantr-statinops/InfoBoard/host/internal/protocol"
)

// resolveLimit applies the bounded precedence: explicit caller limit, then the
// profile default, then the IP-08 fallback. An explicit out-of-range value is
// rejected instead of silently capped.
func (s *Service) resolveLimit(request protocol.QueryRequest) (int, *protocol.Error) {
	max := s.config.MaxResultLimit
	if max <= 0 || max > protocol.MaxResults {
		max = protocol.MaxResults
	}
	if request.HasResultLimit {
		if request.ResultLimit <= 0 || request.ResultLimit > max {
			return 0, protocol.NewError(protocol.CodePayloadLimit).WithBound("result_limit", max, request.ResultLimit)
		}
		return request.ResultLimit, nil
	}
	limit := s.config.FallbackResultLimit
	if s.defaults != nil {
		if configured, ok := s.defaults.ResultLimit(string(request.Context.ProfileID)); ok && configured > 0 {
			limit = configured
		}
	}
	if limit < 1 {
		limit = s.config.FallbackResultLimit
	}
	if limit > max {
		limit = max
	}
	return limit, nil
}
