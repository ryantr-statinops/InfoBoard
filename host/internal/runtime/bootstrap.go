// Package runtime owns the InfoBoard Native Messaging host process and its
// single-connection session boundary.
//
// Transport boundary: the host talks to Chrome or Edge only through the
// browser-provided framed stdin/stdout channel. One reader goroutine owns the
// input stream, one mutex-guarded writer owns the output stream, and no handler
// writes to either stream directly. The runtime never opens a listening socket,
// dials a remote endpoint, spawns a child process, or writes diagnostics to
// stdout. Recovery from a failed session is a fresh browser-provided connection
// with a new epoch, never a reconnect to another transport.
//
// Lifecycle contract:
//
//	Starting -> Handshaking -> Synchronizing -> Ready
//	Ready    -> Synchronizing when an authoritative resync is required
//	any live -> Recovering when a new connection is pending
//	any      -> Unavailable for a fail-closed incompatibility
//	any      -> Closing as the idempotent terminal state
//
// Ready is the only state that publishes availability "healthy", and it is
// reached only after protocol negotiation and an authoritative projection
// snapshot have both been accepted. A session with an unknown profile, an
// incompatible protocol version, invalid limits, or a rebuilding projection
// stays unavailable or recovering and cannot accept query work.
//
// This package stops at the transport and lifecycle boundary. The Native
// Messaging envelope, message types, and wire error literals belong to
// host/internal/protocol (IP-07), which implements ProtocolHandler and maps the
// FailureClass values below onto its own wire codes.
package runtime

import (
	"crypto/rand"
	"io"
	"regexp"
	"strings"
	"time"

	"github.com/ryantr-statinops/InfoBoard/host/internal/domain"
)

// Limits centralizes every bounded runtime budget. IP-07 supplies its wire
// level field bounds through the same struct so no timeout, size, or concurrency
// limit is duplicated as a magic number.
type Limits struct {
	MaxFrameBytes          int
	MaxQueuedFrames        int
	MaxConcurrentRequests  int
	StartupTimeout         time.Duration
	HandshakeTimeout       time.Duration
	SynchronizationTimeout time.Duration
	RequestTimeout         time.Duration
	ShutdownTimeout        time.Duration
	MaxDiagnostics         int
	MaxDiagnosticBytes     int
}

const (
	absoluteMaxFrameBytes     = 8 << 20
	absoluteMaxQueuedFrames   = 4096
	absoluteMaxConcurrency    = 256
	absoluteMaxDuration       = 5 * time.Minute
	absoluteMaxDiagnostics    = 4096
	absoluteMaxDiagnosticSize = 1 << 20
)

// DefaultLimits returns the bounded production budgets. Values mirror the
// IP-07 timeout table so both layers agree without duplicating constants.
func DefaultLimits() Limits {
	return Limits{
		MaxFrameBytes:          1 << 20,
		MaxQueuedFrames:        64,
		MaxConcurrentRequests:  8,
		StartupTimeout:         5 * time.Second,
		HandshakeTimeout:       2 * time.Second,
		SynchronizationTimeout: 2 * time.Second,
		RequestTimeout:         100 * time.Millisecond,
		ShutdownTimeout:        time.Second,
		MaxDiagnostics:         256,
		MaxDiagnosticBytes:     64 << 10,
	}
}

// Validate rejects an unsupported configuration before any allocation, reader,
// or goroutine exists.
func (limits Limits) Validate() error {
	invalid := func() error { return fail(FailureConfiguration, ErrInvalidLimits) }
	switch {
	case limits.MaxFrameBytes <= 0, limits.MaxFrameBytes > absoluteMaxFrameBytes,
		limits.MaxQueuedFrames <= 0, limits.MaxQueuedFrames > absoluteMaxQueuedFrames,
		limits.MaxConcurrentRequests <= 0, limits.MaxConcurrentRequests > absoluteMaxConcurrency,
		limits.StartupTimeout <= 0, limits.StartupTimeout > absoluteMaxDuration,
		limits.HandshakeTimeout <= 0, limits.HandshakeTimeout > absoluteMaxDuration,
		limits.SynchronizationTimeout <= 0, limits.SynchronizationTimeout > absoluteMaxDuration,
		limits.RequestTimeout <= 0, limits.RequestTimeout > absoluteMaxDuration,
		limits.ShutdownTimeout <= 0, limits.ShutdownTimeout > absoluteMaxDuration,
		limits.MaxDiagnostics <= 0, limits.MaxDiagnostics > absoluteMaxDiagnostics,
		limits.MaxDiagnosticBytes <= 0, limits.MaxDiagnosticBytes > absoluteMaxDiagnosticSize:
		return invalid()
	}
	if limits.HandshakeTimeout > limits.StartupTimeout {
		return invalid()
	}
	return nil
}

// Invocation is the verified browser invocation. Chrome and Edge pass the
// calling extension origin, optionally followed by a parent window handle.
type Invocation struct {
	Origin string
}

var (
	parentWindowArgument = regexp.MustCompile(`^--parent-window=[0-9]{1,10}$`)
	extensionOrigin      = regexp.MustCompile(`^chrome-extension://[a-p]{32}$`)
)

// ParseInvocation accepts only the documented browser argument shapes. Any
// other token fails closed before tab data could be accepted.
func ParseInvocation(args []string) (Invocation, error) {
	if len(args) == 0 {
		return Invocation{}, fail(FailureConfiguration, ErrMissingOrigin)
	}
	if len(args) > 2 || (len(args) == 2 && !parentWindowArgument.MatchString(args[1])) {
		return Invocation{}, fail(FailureConfiguration, ErrUnsupportedArgument)
	}
	origin := args[0]
	if len(origin) > 256 || strings.ContainsAny(origin, " \t\r\n") || !extensionOrigin.MatchString(origin) {
		return Invocation{}, fail(FailureConfiguration, ErrInvalidOrigin)
	}
	return Invocation{Origin: origin}, nil
}

// Config is the validated process configuration.
type Config struct {
	HostVersion     string
	ProtocolVersion int
	AllowedOrigins  []string
	ExpectedProfile domain.ProfileID
	Limits          Limits
	Clock           Clock
	Diagnostics     *Diagnostics
}

// DefaultConfig returns a bounded configuration bound to a diagnostic sink.
func DefaultConfig(hostVersion string, sink DiagnosticSink, limits Limits) Config {
	clock := SystemClock()
	return Config{
		HostVersion:     hostVersion,
		ProtocolVersion: 1,
		Limits:          limits,
		Clock:           clock,
		Diagnostics:     NewDiagnostics(sink, limits.MaxDiagnostics, limits.MaxDiagnosticBytes),
	}
}

func (config Config) withDefaults() Config {
	if config.Clock == nil {
		config.Clock = SystemClock()
	}
	if config.Diagnostics == nil {
		config.Diagnostics = NewDiagnostics(nil, config.Limits.MaxDiagnostics, config.Limits.MaxDiagnosticBytes)
	}
	return config
}

func (config Config) validate(invocation Invocation) error {
	invalid := func(err error) error { return fail(FailureConfiguration, err) }
	if len(config.HostVersion) == 0 || len(config.HostVersion) > 32 {
		return invalid(ErrInvalidLimits)
	}
	if config.ProtocolVersion <= 0 {
		return invalid(ErrInvalidLimits)
	}
	if err := config.Limits.Validate(); err != nil {
		return err
	}
	if config.ExpectedProfile != "" {
		if _, err := domain.ParseProfileID(string(config.ExpectedProfile)); err != nil {
			return fail(FailureProfileMismatch, ErrBindingInvalid)
		}
	}
	if len(config.AllowedOrigins) == 0 {
		return fail(FailureRegistration, ErrOriginNotAllowed)
	}
	allowed := false
	for _, origin := range config.AllowedOrigins {
		if !extensionOrigin.MatchString(origin) {
			return fail(FailureRegistration, ErrInvalidOrigin)
		}
		if origin == invocation.Origin {
			allowed = true
		}
	}
	if !allowed {
		return fail(FailureRegistration, ErrOriginNotAllowed)
	}
	return nil
}

// Bootstrap validates limits, registration assumptions, and the browser
// invocation, then constructs one session that is ready for protocol handoff.
// It returns a classified error rather than a partially started session.
func Bootstrap(config Config, invocation Invocation, input io.Reader, output io.Writer, deps Dependencies) (*Session, error) {
	prepared := config.withDefaults()
	if err := prepared.validate(invocation); err != nil {
		prepared.Diagnostics.Observe(HealthSnapshot{
			SessionID:       newSessionID(),
			State:           StateStarting,
			Availability:    AvailabilityUnavailable,
			HostVersion:     prepared.HostVersion,
			ProtocolVersion: prepared.ProtocolVersion,
			Persistence:     PersistenceUnknown,
		}, LevelError, EventBootstrapRejected, 0, 0)
		return nil, err
	}
	if input == nil || output == nil {
		return nil, fail(FailureConfiguration, ErrInvalidLimits)
	}
	session := newSession(prepared, input, output, deps)
	session.diagnostics.Observe(session.health.current(), LevelInfo, EventBootstrapAccepted, 0, 0)
	return session, nil
}

const sessionIDBytes = 16

var sessionIDAlphabet = "0123456789abcdef"

// readRandom is the only entropy source in the host. It never touches the
// network, a socket, or a child process.
func readRandom(buffer []byte) (int, error) { return rand.Read(buffer) }

func newSessionID() string {
	raw := make([]byte, sessionIDBytes)
	read, err := readRandom(raw)
	if err != nil {
		return "00000000000000000000000000000000"
	}
	id := make([]byte, 0, sessionIDBytes*2)
	for _, value := range raw[:read] {
		id = append(id, sessionIDAlphabet[value>>4], sessionIDAlphabet[value&0x0f])
	}
	if len(id) != sessionIDBytes*2 {
		return "00000000000000000000000000000000"
	}
	return string(id)
}
