// Package projection owns the host mirror of the extension's authoritative
// browser-profile tab projection.
//
// Boundary: the extension owns browser authority. IP-04 allocates the
// projection epoch, event sequence, and projection revision, and IP-02 owns the
// profile, context, tab identity, and projection types this package reuses.
// This package maps those already allocated values into the IP-02 domain
// projection, mirrors them into one committed in-memory state per
// profile/context partition, and hands that state to the session, index, and
// query seams. It allocates no epoch or revision of its own, reads no browser
// state, and writes no durable storage.
//
// Fail-closed rules: profile_id, context_kind, projection_epoch, and
// projection_revision are mandatory on every state, record, event, and handoff.
// A missing, malformed, or mismatched fence field is rejected before any map,
// counter, or status changes, and no synthetic identity, profile, context, epoch,
// or revision is substituted for data the extension did not send.
//
// Layering: an IP-07 payload is the only host-visible carrier of the IP-04
// normalized event values, so this package consumes those payload types and maps
// them into the IP-02 domain types. It defines no wire shape, no transport, and no
// session lifecycle of its own; host/internal/runtime consumes projection, never
// the other way around.
package projection

import (
	"sort"
	"sync"

	"github.com/ryantr-statinops/InfoBoard/host/internal/domain"
)

// Status is the observable lifecycle status of one projection partition. The
// values are the IP-02/IP-04 projection lifecycle names so the extension and the
// host report one vocabulary instead of two parallel ones.
type Status string

const (
	StatusUninitialized  Status = "Uninitialized"
	StatusSnapshotReady  Status = "SnapshotReady"
	StatusReady          Status = "Ready"
	StatusResyncRequired Status = "ResyncRequired"
	StatusStaleReference Status = "StaleReference"
)

// String implements fmt.Stringer.
func (status Status) String() string { return string(status) }

// HasLineage reports whether the status refers to a known epoch. Only a partition
// with a committed lineage may be published to the index or query seams.
func (status Status) HasLineage() bool {
	switch status {
	case StatusSnapshotReady, StatusReady, StatusResyncRequired, StatusStaleReference:
		return true
	default:
		return false
	}
}

// Partition is the mandatory profile/context key of every projection value.
// Normal and private contexts remain separate partitions, and equal browser tab,
// window, and group ids in two profiles never share a state, revision, or epoch.
type Partition struct {
	ProfileID   domain.ProfileID
	ContextKind domain.ContextKind
}

// Identity returns the IP-02 tab identity of one browser tab id inside this
// partition. The identity is derived from the bound partition only; the host
// never invents a profile or context for a tab id it received without one.
func (partition Partition) Identity(tabID int64) domain.TabIdentity {
	return domain.TabIdentity{ProfileID: partition.ProfileID, ContextKind: partition.ContextKind, TabID: tabID}
}

// Valid reports whether the partition names a real profile and a known context.
// An unparsable profile id or unknown context is never accepted as a partition,
// because a synthetic partition would silently merge or lose browser tabs.
func (partition Partition) Valid() bool {
	if _, err := domain.ParseProfileID(string(partition.ProfileID)); err != nil {
		return false
	}
	return partition.ContextKind == domain.ContextNormal || partition.ContextKind == domain.ContextPrivate
}

// Counters are the bounded projection counters. They carry accepted, duplicate,
// stale, rejected, and recovery counts only: no title, URL, query, token, or page
// value is representable here.
type Counters struct {
	SnapshotsCommitted uint64
	SnapshotsRejected  uint64
	EventsAccepted     uint64
	EventsDuplicate    uint64
	EventsStale        uint64
	EventsRejected     uint64
	ResyncRequests     uint64
}

// Committed is the immutable read-only view of one partition's committed state.
// Its epoch and revision are read back from the IP-02 domain projection that was
// committed, so this package holds no epoch or revision counter that could drift
// from the accepted lineage. Every accessor returns a copy; the map that backs a
// committed state is never exposed, so no caller can mutate it.
type Committed struct {
	Partition Partition
	Status    Status
	Epoch     string
	Revision  uint64
	Counters  Counters
	records   []domain.EligibleTabRecord
}

// newCommitted builds the immutable committed view from an already validated
// IP-02 domain projection. Records are copied into canonical identity order so
// the accepted identity set does not depend on Go map iteration order.
func newCommitted(partition Partition, status Status, epoch string, revision uint64, counters Counters, projection *domain.ProjectionState) Committed {
	committed := Committed{Partition: partition, Status: status, Epoch: epoch, Revision: revision, Counters: counters}
	if projection == nil {
		return committed
	}
	committed.records = make([]domain.EligibleTabRecord, 0, len(projection.Records))
	for _, record := range projection.Records {
		committed.records = append(committed.records, record)
	}
	sort.Slice(committed.records, func(i, j int) bool {
		return identityLess(committed.records[i].TabIdentity, committed.records[j].TabIdentity)
	})
	return committed
}

// identityLess orders tab identities by the immutable IP-02 identity tuple. The
// comparison is total and value based, so canonical output is independent of
// input order and map iteration order.
func identityLess(left, right domain.TabIdentity) bool {
	if left.ProfileID != right.ProfileID {
		return left.ProfileID < right.ProfileID
	}
	if left.ContextKind != right.ContextKind {
		return left.ContextKind < right.ContextKind
	}
	return left.TabID < right.TabID
}

// Count returns the number of committed eligible records.
func (committed Committed) Count() int { return len(committed.records) }

// Records returns the committed records in canonical identity order. The result
// is a copy of an immutable slice; mutating it cannot affect committed state.
func (committed Committed) Records() []domain.EligibleTabRecord {
	return append([]domain.EligibleTabRecord(nil), committed.records...)
}

// Identities returns the exact accepted identity set in canonical order.
func (committed Committed) Identities() []domain.TabIdentity {
	identities := make([]domain.TabIdentity, 0, len(committed.records))
	for _, record := range committed.records {
		identities = append(identities, record.TabIdentity)
	}
	return identities
}

// Lookup returns the committed record for one browser tab id in this partition.
func (committed Committed) Lookup(tabID int64) (domain.EligibleTabRecord, bool) {
	target := committed.Partition.Identity(tabID)
	index := sort.Search(len(committed.records), func(i int) bool { return !identityLess(committed.records[i].TabIdentity, target) })
	if index >= len(committed.records) || committed.records[index].TabIdentity != target {
		return domain.EligibleTabRecord{}, false
	}
	return committed.records[index], true
}

// Handoff is the bounded index-handoff view of a committed partition. It names
// the profile, context, epoch, revision, status, and exact accepted identity set
// so IP-09, IP-10, IP-12, and IP-16 never need access to the committed map. It
// carries no tab metadata and no raw title, URL, query, token, or page value.
type Handoff struct {
	ProfileID   domain.ProfileID
	ContextKind domain.ContextKind
	Epoch       string
	Revision    uint64
	Status      Status
	Count       int
	Identities  []domain.TabIdentity
	Queryable   bool
}

// State is the single owner of one partition's committed state. It publishes one
// immutable Committed view per accepted lineage and copies it out to every
// reader, so a reader observes either the previous complete map or the new
// complete map and never a partially loaded one.
type State struct {
	mu        sync.Mutex
	partition Partition
	committed *Committed
}

func newState(partition Partition) *State { return &State{partition: partition} }

// Partition returns the profile/context partition this state is bound to.
func (state *State) Partition() Partition { return state.partition }

// Status returns the current observable projection status.
func (state *State) Status() Status { return state.Committed().Status }

// Epoch returns the committed projection epoch, or an empty string before the
// first authoritative snapshot commits.
func (state *State) Epoch() string { return state.Committed().Epoch }

// Revision returns the committed projection revision, or zero before the first
// authoritative snapshot commits. The value always comes from the committed
// lineage; the host never advances it on its own.
func (state *State) Revision() uint64 { return state.Committed().Revision }

// Counters returns a copy of the bounded projection counters.
func (state *State) Counters() Counters { return state.Committed().Counters }

// Committed returns the current immutable committed view. An uninitialized
// partition reports StatusUninitialized, an empty epoch, revision zero, and an
// empty identity set rather than a partial map.
func (state *State) Committed() Committed {
	state.mu.Lock()
	defer state.mu.Unlock()
	if state.committed == nil {
		return newCommitted(state.partition, StatusUninitialized, "", 0, Counters{}, nil)
	}
	return *state.committed
}

// Handoff returns the bounded index-handoff view of the committed state. A
// partition without a known committed lineage, or one whose status still requires
// a resynchronizing snapshot, is never announced as queryable.
func (state *State) Handoff() Handoff {
	committed := state.Committed()
	identities := committed.Identities()
	return Handoff{
		ProfileID:   committed.Partition.ProfileID,
		ContextKind: committed.Partition.ContextKind,
		Epoch:       committed.Epoch,
		Revision:    committed.Revision,
		Status:      committed.Status,
		Count:       len(identities),
		Identities:  identities,
		Queryable:   committed.Status == StatusReady,
	}
}
