package lifecycle

import (
	"context"

	"github.com/ryantr-statinops/InfoBoard/host/internal/domain"
)

// LifecycleStatus is the bounded outcome of one lifecycle operation.
type LifecycleStatus string

const (
	StatusComplete   LifecycleStatus = "complete"
	StatusDegraded   LifecycleStatus = "degraded"
	StatusRetryable  LifecycleStatus = "retryable"
	StatusIncomplete LifecycleStatus = "incomplete"
)

const (
	OperationLoad      = "load"
	OperationUpdate    = "update"
	OperationReset     = "reset"
	OperationUninstall = "uninstall"
)

const (
	StepStorageReset      = "storage_reset"
	StepIndexCleared      = "index_cleared"
	StepSnapshotRequested = "snapshot_requested"
	StepStorageUninstall  = "storage_uninstall"
	StepPathsRemoved      = "paths_removed"
)

// ProjectionReset is the observable handoff from the lifecycle owner to the
// projection/index owner after the profile's durable rows are gone.
type ProjectionReset struct {
	Revision          uint64
	IndexCleared      bool
	SnapshotRequested bool
}

// ResetRuntime is injected by the composition layer. It owns in-memory index
// clearing and the full-snapshot request; the lifecycle package never mutates
// projection state directly.
type ResetRuntime interface {
	ResetProjection(ctx context.Context, profileID domain.ProfileID) (ProjectionReset, error)
}

// LifecycleResult is the redacted UI/diagnostics contract for load, update,
// reset, and uninstall. It contains no raw titles, URLs, queries, or arbitrary
// filesystem paths.
type LifecycleResult struct {
	Operation          string           `json:"operation"`
	ProfileID          domain.ProfileID `json:"profile_id"`
	Status             LifecycleStatus  `json:"status"`
	Retryable          bool             `json:"retryable"`
	Persistence        PersistenceState `json:"persistence"`
	Durable            bool             `json:"durable"`
	ErrorClass         string           `json:"error_class"`
	CompletedSteps     []string         `json:"completed_steps"`
	OwnedStateVerified bool             `json:"owned_state_verified"`
	ProjectionRevision uint64           `json:"projection_revision"`
	Configuration      ProfileConfig    `json:"configuration"`
}
