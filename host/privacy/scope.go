package privacy

import (
	"errors"
	"fmt"
	"unicode/utf8"

	"github.com/ryantr-statinops/InfoBoard/host/internal/domain"
	"github.com/ryantr-statinops/InfoBoard/host/internal/protocol"
)

var ErrInvalidScope = errors.New("privacy: invalid scope")

// Scope is the host-side identity tuple for one session and all state derived
// from it. PrivateNonce is ephemeral and never persisted.
type Scope struct {
	BrowserFamily protocol.BrowserFamily
	ProfileID     domain.ProfileID
	ContextKind   domain.ContextKind
	PrivateNonce  string
}

// ParseScope validates a scope before it can be bound to a session.
func ParseScope(browserFamily, profileID, contextKind, privateNonce string) (Scope, error) {
	family := protocol.BrowserFamily(browserFamily)
	if !family.KnownBrowser() {
		return Scope{}, fmt.Errorf("%w: browser family", ErrInvalidScope)
	}
	profile, err := domain.ParseProfileID(profileID)
	if err != nil {
		return Scope{}, fmt.Errorf("%w: profile id", ErrInvalidScope)
	}
	kind := domain.ContextKind(contextKind)
	if kind != domain.ContextNormal && kind != domain.ContextPrivate {
		return Scope{}, fmt.Errorf("%w: context kind", ErrInvalidScope)
	}
	if kind == domain.ContextNormal && privateNonce != "" {
		return Scope{}, fmt.Errorf("%w: normal context has a nonce", ErrInvalidScope)
	}
	if privateNonce != "" {
		if !utf8.ValidString(privateNonce) || len(privateNonce) > 128 {
			return Scope{}, fmt.Errorf("%w: private nonce", ErrInvalidScope)
		}
	}
	return Scope{
		BrowserFamily: family,
		ProfileID:     profile,
		ContextKind:   kind,
		PrivateNonce:  privateNonce,
	}, nil
}

// Key returns a stable in-memory key. It is never persisted.
func (scope Scope) Key() string {
	return string(scope.BrowserFamily) + "\x00" + string(scope.ProfileID) + "\x00" +
		string(scope.ContextKind) + "\x00" + scope.PrivateNonce
}

// MatchesRequestContext verifies that a bound request context belongs to scope.
func (scope Scope) MatchesRequestContext(request protocol.RequestContext) bool {
	return request.ProfileID == scope.ProfileID && request.ContextKind == scope.ContextKind
}

// ValidatedFrame is a frame whose envelope and payload already passed the
// privacy boundary. It carries no decoded payload so callers remain responsible
// for handing the original bounded bytes to the owning protocol path.
type ValidatedFrame struct {
	Envelope protocol.Envelope
	Scope    Scope
}

// ValidateFrame checks scope before any projection, index, or storage path is
// allowed to decode or apply a request. IP-07 owns envelope and payload limits;
// this function consumes its decoders rather than duplicating them.
func ValidateFrame(frame []byte, bound Scope, limits protocol.Limits) (ValidatedFrame, *protocol.Error) {
	envelope, failure := protocol.DecodeEnvelope(frame, limits)
	if failure != nil {
		return ValidatedFrame{}, failure
	}
	if envelope.ProfileID != string(bound.ProfileID) {
		return ValidatedFrame{}, protocol.NewError(protocol.CodeProfileMismatch)
	}
	switch envelope.Type {
	case protocol.TypeHello:
		hello, failure := protocol.DecodeHello(envelope.Payload, limits)
		if failure != nil {
			return ValidatedFrame{}, failure
		}
		if hello.BrowserFamily != bound.BrowserFamily || hello.ContextKind != bound.ContextKind {
			return ValidatedFrame{}, protocol.NewError(protocol.CodeProfileMismatch)
		}
	case protocol.TypeSnapshot:
		snapshot, failure := protocol.DecodeSnapshot(envelope.Payload, limits)
		if failure != nil {
			return ValidatedFrame{}, failure
		}
		for _, record := range snapshot.Tabs {
			if !bound.matchesRecord(record) {
				return ValidatedFrame{}, protocol.NewError(protocol.CodeProfileMismatch)
			}
		}
	case protocol.TypeDelta:
		delta, failure := protocol.DecodeDelta(envelope.Payload, limits)
		if failure != nil {
			return ValidatedFrame{}, failure
		}
		for _, operation := range delta.Operations {
			if operation.Record != nil && !bound.matchesRecord(*operation.Record) {
				return ValidatedFrame{}, protocol.NewError(protocol.CodeProfileMismatch)
			}
		}
	case protocol.TypeQuery:
		if _, failure := protocol.DecodeQuery(envelope.Payload, limits); failure != nil {
			return ValidatedFrame{}, failure
		}
	case protocol.TypeActivationObserved, protocol.TypeActivationFailed:
		if _, failure := protocol.DecodeActivationStatus(envelope.Payload, envelope.Type); failure != nil {
			return ValidatedFrame{}, failure
		}
	case protocol.TypeLifecycle:
		if _, failure := protocol.DecodeLifecycle(envelope.Payload, bound.ProfileID); failure != nil {
			return ValidatedFrame{}, failure
		}
	case protocol.TypeHealth:
		// Health has no identity-bearing payload. The envelope scope is enough.
	default:
		return ValidatedFrame{}, protocol.NewError(protocol.CodeInvalidFrame)
	}
	return ValidatedFrame{Envelope: envelope, Scope: bound}, nil
}

func (scope Scope) matchesRecord(record protocol.TabRecord) bool {
	return record.ProfileID == string(scope.ProfileID) && record.ContextKind == scope.ContextKind
}

// RejectPrivateDurableWrite is the repository-boundary guard for private data.
// It is called before any durable write and returns PROFILE_MISMATCH because a
// private scope can never be represented as a normal durable row.
func RejectPrivateDurableWrite(scope Scope) *protocol.Error {
	if scope.ContextKind == domain.ContextPrivate {
		return protocol.NewError(protocol.CodeProfileMismatch)
	}
	return nil
}
