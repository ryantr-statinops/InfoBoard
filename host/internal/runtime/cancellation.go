package runtime

import (
	"context"
	"errors"
	"io"
	"sync"
	"syscall"
	"time"
)

// Clock is the injectable time seam. Deterministic fixture runs supply a fake
// clock so startup, handshake, request, and shutdown budgets are observable
// without wall-clock sleeps.
type Clock interface {
	Now() time.Time
	NewTimer(duration time.Duration) (<-chan time.Time, func() bool)
}

type systemClock struct{}

// SystemClock returns the wall-clock implementation used by the process entry
// point. Tests inject a deterministic Clock instead.
func SystemClock() Clock { return systemClock{} }

func (systemClock) Now() time.Time { return time.Now() }

func (systemClock) NewTimer(duration time.Duration) (<-chan time.Time, func() bool) {
	timer := time.NewTimer(duration)
	return timer.C, timer.Stop
}

var (
	ErrSessionClosing        = errors.New("runtime: session closing")
	ErrCancelled             = errors.New("runtime: session cancelled")
	ErrStartupBudget         = errors.New("runtime: startup budget exceeded")
	ErrHandshakeBudget       = errors.New("runtime: handshake budget exceeded")
	ErrSynchronizationBudget = errors.New("runtime: synchronization budget exceeded")
	ErrRequestBudget         = errors.New("runtime: request budget exceeded")
	ErrShutdownBudget        = errors.New("runtime: shutdown budget exceeded")
	ErrInvalidLimits         = errors.New("runtime: limits invalid")
	ErrUnsupportedArgument   = errors.New("runtime: unsupported invocation argument")
	ErrMissingOrigin         = errors.New("runtime: browser origin missing")
	ErrInvalidOrigin         = errors.New("runtime: browser origin invalid")
	ErrOriginNotAllowed      = errors.New("runtime: browser origin not registered")
	ErrProtocolUnavailable   = errors.New("runtime: protocol handler unavailable")
	ErrProjectionUnavailable = errors.New("runtime: projection owner unavailable")
	ErrFrameTooLarge         = errors.New("runtime: frame exceeds limit")
	ErrEmptyFrame            = errors.New("runtime: frame length invalid")
	ErrFrameTruncated        = errors.New("runtime: frame truncated")
	ErrQueueOverflow         = errors.New("runtime: frame queue overflow")
	ErrConcurrencyLimit      = errors.New("runtime: concurrent request limit")
	ErrSnapshotNotReady      = errors.New("runtime: authoritative snapshot not ready")
	ErrBindingInvalid        = errors.New("runtime: session binding invalid")
	ErrWriterClosed          = errors.New("runtime: output stream closed")
	ErrSessionRunning        = errors.New("runtime: session already running")
)

// FailureError carries a safe runtime classification. Error text is limited to
// the classification literal, so wrapped causes cannot leak frame content.
type FailureError struct {
	Class     FailureClass
	Retryable bool
	Err       error
}

func (failure *FailureError) Error() string { return string(failure.Class) }

func (failure *FailureError) Unwrap() error { return failure.Err }

// fail builds a classified error. Retryability always comes from the canonical
// FailureClass table, so a construction site can never contradict the value an
// observer or a fixture reads back from the class.
func fail(class FailureClass, err error) error {
	return &FailureError{Class: class, Retryable: class.Retryable(), Err: err}
}

// ClassifyFailure maps a bounded set of transport, budget, and context errors
// onto a safe FailureClass. Retryability always comes from the single canonical
// FailureClass.Retryable table so a mapped error and its class can never
// disagree. It never inspects frame content.
func ClassifyFailure(err error) (FailureClass, bool) {
	class := classify(err)
	return class, class.Retryable()
}

func classify(err error) FailureClass {
	if err == nil {
		return FailureNone
	}
	var classified *FailureError
	if errors.As(err, &classified) {
		return classified.Class
	}
	switch {
	case errors.Is(err, ErrRequestBudget):
		return FailureRequestTimeout
	case errors.Is(err, ErrHandshakeBudget):
		return FailureHandshakeTimeout
	case errors.Is(err, ErrStartupBudget):
		return FailureStartupTimeout
	case errors.Is(err, ErrSynchronizationBudget):
		return FailureSynchronizationTimeout
	case errors.Is(err, ErrShutdownBudget):
		return FailureShutdown
	case errors.Is(err, ErrQueueOverflow):
		return FailureQueueOverflow
	case errors.Is(err, ErrConcurrencyLimit):
		return FailureConcurrencyLimit
	case errors.Is(err, ErrFrameTooLarge):
		return FailureFrameTooLarge
	case errors.Is(err, ErrEmptyFrame):
		return FailureTransportMalformed
	case errors.Is(err, ErrFrameTruncated):
		return FailureTransportTruncated
	case errors.Is(err, io.ErrUnexpectedEOF):
		return FailureTransportTruncated
	case errors.Is(err, io.EOF):
		return FailureTransportEOF
	case errors.Is(err, syscall.EPIPE), errors.Is(err, syscall.ECONNRESET):
		return FailureTransportBrokenPipe
	case errors.Is(err, syscall.EBADF):
		return FailureTransportUnavailable
	case errors.Is(err, context.DeadlineExceeded):
		return FailureRequestTimeout
	case errors.Is(err, context.Canceled):
		return FailureCancelled
	default:
		return FailureInternal
	}
}

// withBudget derives a cancellable child bounded by budget, preserving the
// parent's earlier deadline when the parent already expires sooner.
func withBudget(parent context.Context, budget time.Duration, cause error) (context.Context, func()) {
	if budget <= 0 {
		return context.WithCancel(parent)
	}
	return context.WithTimeoutCause(parent, budget, cause)
}

// boundedCall runs an owner callback under the caller's budget. A callback that
// ignores its context cannot hold the session past the budget, and a late return
// cannot block on the buffered result channel, so no worker leaks. A panic is
// contained and reported as an internal failure so no panic text can reach the
// protocol stream.
func boundedCall[T any](ctx context.Context, work func() (T, error)) (T, error) {
	type result struct {
		value T
		err   error
	}
	results := make(chan result, 1)
	go func() {
		defer func() {
			if recovered := recover(); recovered != nil {
				results <- result{err: fail(FailureInternal, ErrSessionClosing)}
			}
		}()
		value, err := work()
		results <- result{value: value, err: err}
	}()
	select {
	case completed := <-results:
		return completed.value, completed.err
	case <-ctx.Done():
		var zero T
		return zero, context.Cause(ctx)
	}
}

type requestRegistry struct {
	mu        sync.Mutex
	next      uint64
	active    map[uint64]context.CancelCauseFunc
	started   uint64
	completed uint64
	rejected  uint64
}

func newRequestRegistry() *requestRegistry {
	return &requestRegistry{active: make(map[uint64]context.CancelCauseFunc)}
}

func (registry *requestRegistry) begin(parent context.Context, budget time.Duration) (context.Context, func(), uint64) {
	budgeted, releaseBudget := withBudget(parent, budget, ErrRequestBudget)
	ctx, cancel := context.WithCancelCause(budgeted)
	registry.mu.Lock()
	registry.next++
	id := registry.next
	registry.active[id] = cancel
	registry.started++
	registry.mu.Unlock()
	release := func() {
		cancel(nil)
		releaseBudget()
		registry.mu.Lock()
		delete(registry.active, id)
		registry.mu.Unlock()
	}
	return ctx, release, id
}

func (registry *requestRegistry) cancelAll(cause error) int {
	registry.mu.Lock()
	cancels := make([]context.CancelCauseFunc, 0, len(registry.active))
	for _, cancel := range registry.active {
		cancels = append(cancels, cancel)
	}
	registry.mu.Unlock()
	for _, cancel := range cancels {
		cancel(cause)
	}
	return len(cancels)
}

func (registry *requestRegistry) complete() {
	registry.mu.Lock()
	registry.completed++
	registry.mu.Unlock()
}

func (registry *requestRegistry) reject() {
	registry.mu.Lock()
	registry.rejected++
	registry.mu.Unlock()
}

func (registry *requestRegistry) inflight() int {
	registry.mu.Lock()
	defer registry.mu.Unlock()
	return len(registry.active)
}

func (registry *requestRegistry) counters() (uint64, uint64, uint64) {
	registry.mu.Lock()
	defer registry.mu.Unlock()
	return registry.started, registry.completed, registry.rejected
}
