package diagnostics

import (
	"context"
	"errors"
	"sync"
	"time"
)

// RepairOperation is a bounded recovery action. None of these operations can
// close browser tabs, start a listener, or delete stored data.
type RepairOperation string

const (
	RepairRetryConnection   RepairOperation = "retry_connection"
	RepairResyncProjection  RepairOperation = "resync_projection"
	RepairRebuildIndex      RepairOperation = "rebuild_index"
	RepairRepairPersistence RepairOperation = "repair_persistence"
)

func (operation RepairOperation) Valid() bool {
	switch operation {
	case RepairRetryConnection, RepairResyncProjection, RepairRebuildIndex, RepairRepairPersistence:
		return true
	default:
		return false
	}
}

// RepairStatus is the terminal outcome of one repair request.
type RepairStatus string

const (
	RepairSucceeded RepairStatus = "succeeded"
	RepairFailed    RepairStatus = "failed"
	RepairExhausted RepairStatus = "exhausted"
	RepairCancelled RepairStatus = "cancelled"
	RepairCooldown  RepairStatus = "cooldown"
	RepairInFlight  RepairStatus = "in_flight"
)

func (status RepairStatus) Valid() bool {
	switch status {
	case RepairSucceeded, RepairFailed, RepairExhausted, RepairCancelled, RepairCooldown, RepairInFlight:
		return true
	default:
		return false
	}
}

// RepairResult is the observable outcome of a repair attempt. It never
// contains a raw backend error or browser value.
type RepairResult struct {
	Operation     RepairOperation   `json:"operation"`
	Status        RepairStatus      `json:"status"`
	Attempts      int               `json:"attempts"`
	ElapsedMillis int64             `json:"elapsed_ms"`
	FinalHealth   State             `json:"final_health"`
	Retryable     bool              `json:"retryable"`
	Action        RecommendedAction `json:"recommended_action"`
	ErrorClass    ErrorClass        `json:"error_class,omitempty"`
}

// Validate checks the bounded result vocabulary.
func (result RepairResult) Validate() error {
	if !result.Operation.Valid() || !result.Status.Valid() || !result.FinalHealth.Valid() ||
		!result.Action.Valid() || !result.ErrorClass.Valid() ||
		result.Attempts < 0 || result.ElapsedMillis < 0 {
		return ErrInvalidEvent
	}
	return nil
}

// RepairBackend is the owning component's confirmation boundary. A repair is
// never reported successful until the corresponding method returns nil.
type RepairBackend interface {
	RetryConnection(context.Context, int) error
	ResyncProjection(context.Context, uint64) error
	RebuildIndex(context.Context, uint64) error
	RepairPersistence(context.Context) error
}

// RepairOptions controls the bounded budgets. Zero values use the phase-16
// defaults; tests may shrink them while preserving the same semantics.
type RepairOptions struct {
	RetryConnectionBudget time.Duration
	PersistenceDeadline   time.Duration
	Backoff               []time.Duration
	Cooldown              time.Duration
	Now                   func() time.Time
	Sleep                 func(context.Context, time.Duration) error
}

// RepairController serializes repair work and enforces count, deadline,
// cooldown, and one-in-flight bounds.
type RepairController struct {
	mu              sync.Mutex
	health          *Health
	recorder        *Recorder
	backend         RepairBackend
	options         RepairOptions
	active          bool
	activeOperation RepairOperation
	activeRevision  uint64
	activeCancel    context.CancelFunc
	activeDone      chan struct{}
	cooldownUntil   map[RepairOperation]time.Time
}

// NewRepairController creates a bounded repair controller.
func NewRepairController(health *Health, recorder *Recorder, backend RepairBackend, options RepairOptions) *RepairController {
	if options.RetryConnectionBudget <= 0 {
		options.RetryConnectionBudget = 5 * time.Second
	}
	if options.PersistenceDeadline <= 0 {
		options.PersistenceDeadline = 10 * time.Second
	}
	if len(options.Backoff) == 0 {
		options.Backoff = []time.Duration{100 * time.Millisecond, 500 * time.Millisecond, time.Second}
	}
	if options.Cooldown < 0 {
		options.Cooldown = 0
	}
	if options.Now == nil {
		options.Now = time.Now
	}
	if options.Sleep == nil {
		options.Sleep = sleepContext
	}
	return &RepairController{
		health:        health,
		recorder:      recorder,
		backend:       backend,
		options:       options,
		cooldownUntil: make(map[RepairOperation]time.Time),
	}
}

// Run executes one bounded repair. The caller may cancel through ctx or
// Cancel. A second concurrent request returns in_flight; a request inside the
// post-exhaustion cooldown returns cooldown.
func (controller *RepairController) Run(ctx context.Context, operation RepairOperation, revision uint64) RepairResult {
	if controller == nil || controller.backend == nil || !operation.Valid() {
		return RepairResult{Operation: operation, Status: RepairFailed, Action: ActionOpenDiagnostics, ErrorClass: ErrorInternalFailure}
	}
	if ctx == nil {
		ctx = context.Background()
	}
	controller.mu.Lock()
	if controller.active {
		controller.mu.Unlock()
		return controller.result(operation, RepairInFlight, 0, 0, true, ActionWait, ErrorInternalFailure, "")
	}
	if until, ok := controller.cooldownUntil[operation]; ok && controller.options.Now().Before(until) {
		controller.mu.Unlock()
		return controller.result(operation, RepairCooldown, 0, 0, true, ActionWait, controller.operationClass(operation), "")
	}
	controller.active = true
	controller.activeOperation = operation
	controller.activeRevision = revision
	controller.activeDone = make(chan struct{})
	controller.mu.Unlock()
	defer controller.finish(operation)

	start := controller.options.Now()
	policy := controller.policy(operation)
	controller.record(ReasonRepairStart, controller.operationClass(operation), policy.Action, true)

	operationContext, cancel := context.WithCancel(ctx)
	if operation == RepairRetryConnection {
		operationContext, cancel = context.WithTimeout(ctx, controller.options.RetryConnectionBudget)
	} else if operation == RepairRepairPersistence {
		operationContext, cancel = context.WithTimeout(ctx, controller.options.PersistenceDeadline)
	}
	defer cancel()
	controller.mu.Lock()
	controller.activeCancel = cancel
	controller.mu.Unlock()

	if operation == RepairRebuildIndex {
		controller.beginIndexRebuild()
	}

	attempts := 0
	var lastClass ErrorClass
	for attempt := 1; attempt <= policy.Policy.MaxAttempts; attempt++ {
		if operationContext.Err() != nil {
			result := controller.cancelled(operation, attempts, start, lastClass)
			if operation == RepairRebuildIndex {
				controller.finishIndexRebuild(revision, false)
			}
			return result
		}
		attempts = attempt
		err := controller.invoke(operationContext, operation, attempt, revision)
		if err == nil {
			if operation == RepairRebuildIndex {
				if controller.health != nil && controller.health.Snapshot().ProjectionRevision != revision {
					controller.finishIndexRebuild(revision, false)
					controller.record(ReasonIndexBuildFailure, ErrorRevisionMismatch, ActionRefreshProjection, true)
					return controller.result(operation, RepairFailed, attempts, controller.elapsed(start), true, ActionRefreshProjection, ErrorRevisionMismatch, controller.finalHealth())
				}
				controller.finishIndexRebuild(revision, true)
			}
			if operation == RepairRepairPersistence {
				controller.finishPersistence()
			}
			controller.record(ReasonRepairFinish, ErrorNone, ActionNone, false)
			return controller.result(operation, RepairSucceeded, attempts, controller.elapsed(start), false, ActionNone, ErrorNone, controller.finalHealth())
		}
		lastClass = classForError(err, controller.operationClass(operation))
		if operationContext.Err() != nil || errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
			if operation == RepairRebuildIndex {
				controller.finishIndexRebuild(revision, false)
			}
			return controller.cancelled(operation, attempts, start, lastClass)
		}
		if lastClass == ErrorSchemaUpgradeRequired || lastClass == ErrorProtocolMismatch {
			break
		}
		if attempt == policy.Policy.MaxAttempts {
			break
		}
		if err := controller.sleep(operationContext, controller.backoff(attempt)); err != nil {
			if operation == RepairRebuildIndex {
				controller.finishIndexRebuild(revision, false)
			}
			return controller.cancelled(operation, attempts, start, lastClass)
		}
	}

	if operation == RepairRebuildIndex {
		controller.finishIndexRebuild(revision, false)
	}
	controller.mu.Lock()
	controller.cooldownUntil[operation] = controller.options.Now().Add(controller.cooldown(operation))
	controller.mu.Unlock()
	action := policy.ExhaustedAction
	retryable := true
	if lastClass == ErrorSchemaUpgradeRequired {
		if mapping, ok := Lookup(lastClass); ok {
			action = mapping.Action
			retryable = mapping.Retryable
		}
	}
	controller.record(ReasonRepairExhausted, lastClass, action, retryable)
	return controller.result(operation, RepairExhausted, attempts, controller.elapsed(start), retryable, action, lastClass, controller.finalHealth())
}

// Resync cancels an older resync when a newer projection revision is known,
// then starts the newer request. If the older request cannot finish within ctx,
// the newer request is not started.
func (controller *RepairController) Resync(ctx context.Context, revision uint64) RepairResult {
	if controller == nil {
		return RepairResult{Operation: RepairResyncProjection, Status: RepairFailed, Action: ActionOpenDiagnostics, ErrorClass: ErrorInternalFailure}
	}
	if ctx == nil {
		ctx = context.Background()
	}
	for {
		controller.mu.Lock()
		if controller.active && controller.activeOperation == RepairResyncProjection && revision > controller.activeRevision {
			cancel := controller.activeCancel
			done := controller.activeDone
			controller.mu.Unlock()
			if cancel != nil {
				cancel()
			}
			select {
			case <-done:
				continue
			case <-ctx.Done():
				return controller.result(RepairResyncProjection, RepairCancelled, 0, 0, true, ActionRetry, ErrorSnapshotRequired, "")
			}
		}
		controller.mu.Unlock()
		return controller.Run(ctx, RepairResyncProjection, revision)
	}
}

// Cancel cancels the active operation of the given kind, if any.
func (controller *RepairController) Cancel(operation RepairOperation) {
	if controller == nil {
		return
	}
	controller.mu.Lock()
	cancel := controller.activeCancel
	active := controller.activeOperation
	controller.mu.Unlock()
	if active == operation && cancel != nil {
		cancel()
	}
}

// Active reports whether any repair is currently in flight.
func (controller *RepairController) Active() bool {
	if controller == nil {
		return false
	}
	controller.mu.Lock()
	defer controller.mu.Unlock()
	return controller.active
}

func (controller *RepairController) invoke(ctx context.Context, operation RepairOperation, attempt int, revision uint64) error {
	switch operation {
	case RepairRetryConnection:
		return controller.backend.RetryConnection(ctx, attempt)
	case RepairResyncProjection:
		return controller.backend.ResyncProjection(ctx, revision)
	case RepairRebuildIndex:
		return controller.backend.RebuildIndex(ctx, revision)
	case RepairRepairPersistence:
		return controller.backend.RepairPersistence(ctx)
	default:
		return NewSafeError(ErrorInternalFailure, nil)
	}
}

func (controller *RepairController) finish(operation RepairOperation) {
	controller.mu.Lock()
	controller.active = false
	controller.activeOperation = ""
	controller.activeRevision = 0
	controller.activeCancel = nil
	if controller.activeDone != nil {
		close(controller.activeDone)
		controller.activeDone = nil
	}
	controller.mu.Unlock()
}

func (controller *RepairController) beginIndexRebuild() {
	if controller.health == nil {
		return
	}
	current := controller.health.Snapshot()
	switch current.State {
	case StateReady, StateDegraded:
		current.State = StateRecovering
	case StateRecovering, StateSynchronizing:
	default:
		return
	}
	current.Index = IndexBuilding
	current.Retryable = true
	_ = controller.health.Transition(current)
	controller.record(ReasonIndexBuildStart, ErrorNone, ActionWait, true)
}

func (controller *RepairController) finishIndexRebuild(revision uint64, success bool) {
	if controller.health == nil {
		return
	}
	current := controller.health.Snapshot()
	if success {
		current.Index = IndexReady
		current.IndexedRevision = revision
		current.ProjectionRevision = revision
		current.Retryable = false
		if current.State == StateRecovering {
			current.State = StateSynchronizing
			if err := controller.health.Transition(current); err != nil {
				controller.record(ReasonIndexBuildFailure, ErrorInternalFailure, ActionRetry, true)
				return
			}
			current = controller.health.Snapshot()
		}
		current.Index = IndexReady
		current.IndexedRevision = revision
		current.ProjectionRevision = revision
		current.Retryable = false
		if current.State == StateSynchronizing || current.State == StateReady || current.State == StateDegraded {
			current.State = StateReady
			if current.Storage != StorageHealthy {
				current.State = StateDegraded
			}
			_ = controller.health.Transition(current)
		}
		controller.record(ReasonIndexBuildFinish, ErrorNone, ActionNone, false)
		return
	}
	current.Index = IndexFailed
	current.State = StateRecovering
	current.Retryable = true
	_ = controller.health.Transition(current)
	controller.record(ReasonIndexBuildFailure, ErrorInternalFailure, ActionRetry, true)
}

func (controller *RepairController) finishPersistence() {
	if controller.health == nil {
		return
	}
	current := controller.health.Snapshot()
	if current.Storage == StorageHealthy {
		return
	}
	current.Storage = StorageHealthy
	if current.State == StateDegraded || current.State == StateReady {
		current.State = StateReady
	}
	_ = controller.health.Transition(current)
}

func (controller *RepairController) policy(operation RepairOperation) Mapping {
	class := controller.operationClass(operation)
	mapping, _ := Lookup(class)
	return mapping
}

func (controller *RepairController) operationClass(operation RepairOperation) ErrorClass {
	switch operation {
	case RepairRetryConnection:
		return ErrorHostMissing
	case RepairResyncProjection:
		return ErrorSnapshotRequired
	case RepairRebuildIndex:
		return ErrorInternalFailure
	case RepairRepairPersistence:
		return ErrorPersistenceDegraded
	default:
		return ErrorInternalFailure
	}
}

func (controller *RepairController) cooldown(operation RepairOperation) time.Duration {
	if controller.options.Cooldown > 0 {
		return controller.options.Cooldown
	}
	policy := controller.policy(operation)
	return time.Duration(policy.Policy.CooldownMillis) * time.Millisecond
}

func (controller *RepairController) backoff(attempt int) time.Duration {
	if attempt <= 0 {
		return 0
	}
	index := attempt - 1
	if index >= len(controller.options.Backoff) {
		index = len(controller.options.Backoff) - 1
	}
	return controller.options.Backoff[index]
}

func (controller *RepairController) sleep(ctx context.Context, duration time.Duration) error {
	if duration <= 0 {
		return ctx.Err()
	}
	return controller.options.Sleep(ctx, duration)
}

func (controller *RepairController) elapsed(start time.Time) int64 {
	elapsed := controller.options.Now().Sub(start).Milliseconds()
	if elapsed < 0 {
		return 0
	}
	return elapsed
}

func (controller *RepairController) finalHealth() State {
	if controller.health == nil {
		return ""
	}
	return controller.health.Snapshot().State
}

func (controller *RepairController) cancelled(operation RepairOperation, attempts int, start time.Time, class ErrorClass) RepairResult {
	if class == ErrorNone {
		class = controller.operationClass(operation)
	}
	controller.record(ReasonRepairFinish, class, ActionRetry, true)
	return controller.result(operation, RepairCancelled, attempts, controller.elapsed(start), true, ActionRetry, class, controller.finalHealth())
}

func (controller *RepairController) result(operation RepairOperation, status RepairStatus, attempts int, elapsed int64, retryable bool, action RecommendedAction, class ErrorClass, final State) RepairResult {
	if final == "" {
		final = StateRecovering
	}
	return RepairResult{
		Operation: operation, Status: status, Attempts: attempts, ElapsedMillis: elapsed,
		FinalHealth: final, Retryable: retryable, Action: action, ErrorClass: class,
	}
}

func (controller *RepairController) record(reason ReasonCode, class ErrorClass, action RecommendedAction, retryable bool) {
	if controller.recorder == nil {
		return
	}
	_, _ = controller.recorder.Record(Event{
		Component:         ComponentRepair,
		ReasonCode:        reason,
		ErrorClass:        class,
		Retryable:         retryable,
		RecommendedAction: action,
		Timing:            EventTiming{Latency: LatencyUnknown},
	})
}

func classForError(err error, fallback ErrorClass) ErrorClass {
	var safe SafeError
	if errors.As(err, &safe) && safe.Class.Valid() {
		return safe.Class
	}
	return fallback
}

func sleepContext(ctx context.Context, duration time.Duration) error {
	timer := time.NewTimer(duration)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}
