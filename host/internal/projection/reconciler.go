package projection

import (
	"github.com/ryantr-statinops/InfoBoard/host/internal/domain"
	"github.com/ryantr-statinops/InfoBoard/host/internal/protocol"
)

// Cause is the closed, tab-free vocabulary explaining why a boundary input was
// rejected. The missing_* values name the mandatory field the extension or the
// session failed to send, and the mismatched_* values name a field that contradicts
// the bound partition or lineage. No cause carries a title, URL, query, token, page
// value, or free-form text.
type Cause string

const (
	CausePartitionUnbound              Cause = "partition_unbound"
	CauseMissingProfileID              Cause = "missing_profile_id"
	CauseMissingContextKind            Cause = "missing_context_kind"
	CauseMissingProjectionEpoch        Cause = "missing_projection_epoch"
	CauseMissingProjectionRevision     Cause = "missing_projection_revision"
	CauseMissingTabIdentityProfileID   Cause = "missing_tab_identity_profile_id"
	CauseMissingTabIdentityContextKind Cause = "missing_tab_identity_context_kind"
	CauseMissingTabIdentityTabID       Cause = "missing_tab_identity_tab_id"
	CauseMissingRecord                 Cause = "missing_record"
	CauseMismatchedProfileID           Cause = "mismatched_profile_id"
	CauseMismatchedContextKind         Cause = "mismatched_context_kind"
	CauseMismatchedProjectionEpoch     Cause = "mismatched_projection_epoch"
	CauseMismatchedProjectionRevision  Cause = "mismatched_projection_revision"
	CauseMismatchedTabIdentity         Cause = "mismatched_tab_identity"
	CauseUnexpectedRecord              Cause = "unexpected_record"
	CauseUnknownOperation              Cause = "unknown_operation"
	CauseBoundsExceeded                Cause = "bounds_exceeded"
)

var causes = map[Cause]struct{}{
	CausePartitionUnbound: {}, CauseMissingProfileID: {}, CauseMissingContextKind: {},
	CauseMissingProjectionEpoch: {}, CauseMissingProjectionRevision: {},
	CauseMissingTabIdentityProfileID: {}, CauseMissingTabIdentityContextKind: {},
	CauseMissingTabIdentityTabID: {}, CauseMissingRecord: {},
	CauseMismatchedProfileID: {}, CauseMismatchedContextKind: {},
	CauseMismatchedProjectionEpoch: {}, CauseMismatchedProjectionRevision: {},
	CauseMismatchedTabIdentity: {}, CauseUnexpectedRecord: {},
	CauseUnknownOperation: {}, CauseBoundsExceeded: {},
}

// KnownCause reports whether a cause belongs to the closed vocabulary. An unknown
// cause is dropped from bounded diagnostics rather than reported, so no
// tab-derived or free-form value can reach a diagnostic sink through this field.
func (cause Cause) KnownCause() bool {
	_, known := causes[cause]
	return known
}

// Rejection is the bounded result of a failed boundary validation. Code is the
// IP-07 wire code the protocol layer reports: an incomplete mandatory field yields
// the retryable SNAPSHOT_REQUIRED because a complete authoritative snapshot repairs
// it, while a complete field that contradicts the bound partition yields the
// non-retryable PROFILE_MISMATCH. Index is the rejected record or operation
// position, or -1 when the whole input failed its fence. Nothing in a rejection is
// derived from tab content.
type Rejection struct {
	Code             protocol.ErrorCode
	Cause            Cause
	Resync           protocol.ResyncReason
	ExpectedRevision uint64
	ExpectedSequence uint64
	Index            int
}

// Error implements error. The rendered text contains only closed-vocabulary
// values, so it is safe to log and to classify into a runtime failure.
func (rejection *Rejection) Error() string {
	return "projection rejected: " + string(rejection.Code) + " (" + string(rejection.Cause) + ")"
}

// ResyncRequired reports whether the host must ask the extension for a new
// authoritative snapshot instead of continuing from an uncertain lineage.
func (rejection *Rejection) ResyncRequired() bool { return rejection.Resync != "" }

func reject(code protocol.ErrorCode, cause Cause, resync protocol.ResyncReason, expectedRevision, expectedSequence uint64, index int) *Rejection {
	return &Rejection{Code: code, Cause: cause, Resync: resync, ExpectedRevision: expectedRevision, ExpectedSequence: expectedSequence, Index: index}
}

// missingIdentity rejects an absent or unusable mandatory identity field. The
// remedy is a complete authoritative snapshot, so the outcome is the retryable
// SNAPSHOT_REQUIRED rather than a session-level incompatibility.
func missingIdentity(cause Cause, resync protocol.ResyncReason, expectedRevision, expectedSequence uint64, index int) *Rejection {
	return reject(protocol.CodeSnapshotRequired, cause, resync, expectedRevision, expectedSequence, index)
}

// SnapshotRequest is the boundary input for one authoritative snapshot. Profile,
// context, epoch, and revision are declared exactly as IP-04 allocated them and
// IP-07 delivered them, and Records are the decoded IP-07 snapshot payload. An
// empty record list is an intentional successful empty snapshot only when all four
// fence fields are present and consistent.
type SnapshotRequest struct {
	ProfileID   string
	ContextKind domain.ContextKind
	Epoch       string
	Revision    uint64
	Sequence    uint64
	Records     []protocol.TabRecord
}

// SnapshotInput is one validated authoritative snapshot. It carries the mandatory
// fence exactly once plus the IP-02 records that belong to it, so a later commit
// cannot re-derive or renumber the lineage.
type SnapshotInput struct {
	Partition Partition
	Epoch     string
	Revision  uint64
	Sequence  uint64
	Records   []domain.EligibleTabRecord
}

// DeltaRequest is the boundary input for one ordered delta. Revision is the target
// revision the envelope declared; Delta is the decoded IP-07 delta payload whose
// base revision, sequence range, and ordered operations come from IP-04.
type DeltaRequest struct {
	ProfileID   string
	ContextKind domain.ContextKind
	Epoch       string
	Revision    uint64
	Delta       protocol.Delta
}

// DeltaInput is one validated ordered delta expressed as IP-02 projection events.
// Sequence, predecessor revision, identity, and epoch are taken from the bound
// partition and the declared fence; the host neither renumbers sequences nor
// advances revisions here.
type DeltaInput struct {
	Partition Partition
	Epoch     string
	Revision  uint64
	Events    []domain.ProjectionEvent
}

// HandoffRequest is the boundary input for one published projection status or
// index handoff. Every field is mandatory because a handoff that cannot state its
// profile, context, epoch, and revision has no identity to hand off.
type HandoffRequest struct {
	ProfileID   string
	ContextKind domain.ContextKind
	Epoch       string
	Revision    uint64
}

// Reconciler validates the projection boundary of one bound profile/context
// partition and maps already decoded IP-07 payloads into the IP-02 domain types. It
// owns no browser state, no transport, and no durable store: IP-09 calls it from the
// protocol seam, and host/internal/domain remains the only authority on projection
// mutation.
type Reconciler struct {
	partition Partition
	state     *State
}

// New binds a reconciler to one profile/context partition. A partition without a
// real profile id or with an unknown context is rejected instead of repaired: the
// host never manufactures a profile, context, epoch, or revision for the peer.
func New(partition Partition) (*Reconciler, *Rejection) {
	if !partition.Valid() {
		return nil, reject(protocol.CodeProfileMismatch, CausePartitionUnbound, protocol.ResyncProfileChanged, 0, 0, -1)
	}
	return &Reconciler{partition: partition, state: newState(partition)}, nil
}

// Partition returns the bound profile/context partition.
func (reconciler *Reconciler) Partition() Partition { return reconciler.partition }

// State returns the read-only committed state of the bound partition. A caller
// receives a copy of the committed view and has no way to mutate the map.
func (reconciler *Reconciler) State() *State { return reconciler.state }

// BindSnapshot validates the mandatory fence of one authoritative snapshot and maps
// its records into IP-02 domain records. Every record must repeat the declared
// profile, context, epoch, and revision: a record missing one of them is refused as
// an incomplete identity, and a record naming another partition or another lineage
// is refused as a mismatch, before any state is touched. Text and count bounds stay
// owned by the IP-07 decoder and are not re-implemented here.
func (reconciler *Reconciler) BindSnapshot(request SnapshotRequest) (SnapshotInput, *Rejection) {
	if len(request.Records) > protocol.MaxSnapshotRecords {
		return SnapshotInput{}, missingIdentity(CauseBoundsExceeded, "", 0, 0, -1)
	}
	if request.Revision == 0 || request.Sequence == 0 {
		return SnapshotInput{}, missingIdentity(CauseMissingProjectionRevision, protocol.ResyncRevisionMismatch, 0, 0, -1)
	}
	epoch, rejection := reconciler.fence(request.ProfileID, request.ContextKind, request.Epoch, request.Revision, request.Sequence)
	if rejection != nil {
		return SnapshotInput{}, rejection
	}
	records := make([]domain.EligibleTabRecord, 0, len(request.Records))
	for index := range request.Records {
		record, rejection := reconciler.record(request.Records[index], epoch, request.Revision, index)
		if rejection != nil {
			return SnapshotInput{}, rejection
		}
		records = append(records, record)
	}
	return SnapshotInput{
		Partition: reconciler.partition,
		Epoch:     epoch,
		Revision:  request.Revision,
		Sequence:  request.Sequence,
		Records:   records,
	}, nil
}

// BindDelta validates the mandatory fence of one ordered delta and maps its
// operations into IP-02 projection events in the declared order. Each operation
// inherits its sequence from the contiguous range the IP-07 decoder established and
// its predecessor revision from the accepted revision plus its position, so an
// accepted event advances the lineage exactly once. Unknown operations, records
// without a browser tab identity, and records belonging to another partition, epoch,
// or revision are rejected without partial staging.
func (reconciler *Reconciler) BindDelta(request DeltaRequest) (DeltaInput, *Rejection) {
	if len(request.Delta.Operations) == 0 || len(request.Delta.Operations) > protocol.MaxDeltaOperations {
		return DeltaInput{}, missingIdentity(CauseBoundsExceeded, "", 0, 0, -1)
	}
	if request.Revision == 0 {
		return DeltaInput{}, reject(protocol.CodeRevisionMismatch, CauseMissingProjectionRevision, protocol.ResyncRevisionMismatch, request.Delta.BaseRevision, request.Delta.SequenceStart, -1)
	}
	if request.Revision < request.Delta.BaseRevision {
		return DeltaInput{}, reject(protocol.CodeRevisionMismatch, CauseMismatchedProjectionRevision, protocol.ResyncRevisionMismatch, request.Delta.BaseRevision, request.Delta.SequenceStart, -1)
	}
	if request.Delta.SequenceStart == 0 {
		return DeltaInput{}, reject(protocol.CodeRevisionMismatch, CauseMissingProjectionRevision, protocol.ResyncSequenceGap, request.Delta.BaseRevision, 0, -1)
	}
	epoch, rejection := reconciler.fence(request.ProfileID, request.ContextKind, request.Epoch, request.Revision, request.Delta.SequenceStart)
	if rejection != nil {
		return DeltaInput{}, rejection
	}
	events := make([]domain.ProjectionEvent, 0, len(request.Delta.Operations))
	for index := range request.Delta.Operations {
		event, rejection := reconciler.event(request.Delta, index, epoch)
		if rejection != nil {
			return DeltaInput{}, rejection
		}
		events = append(events, event)
	}
	return DeltaInput{
		Partition: reconciler.partition,
		Epoch:     epoch,
		Revision:  request.Revision,
		Events:    events,
	}, nil
}

// BindHandoff validates the identity of one published status or index handoff and
// returns the bounded view of the committed state. Identity is never inferred from
// the record values a handoff happens to carry, and a handoff is never announced as
// queryable while the committed lineage is unknown or still needs a snapshot.
func (reconciler *Reconciler) BindHandoff(request HandoffRequest) (Handoff, *Rejection) {
	if _, rejection := reconciler.fence(request.ProfileID, request.ContextKind, request.Epoch, request.Revision, 0); rejection != nil {
		return Handoff{}, rejection
	}
	return reconciler.state.Handoff(), nil
}

// fence resolves the mandatory profile, context, epoch, and revision of one input
// against the bound partition. An absent or unusable field is refused as an
// incomplete identity and a complete field naming another partition or lineage is
// refused as a mismatch; nothing here is defaulted, guessed, or carried over from
// the committed state.
func (reconciler *Reconciler) fence(profileID string, kind domain.ContextKind, epoch string, revision, sequence uint64) (string, *Rejection) {
	if _, err := domain.ParseProfileID(profileID); err != nil {
		return "", missingIdentity(CauseMissingProfileID, protocol.ResyncProfileChanged, revision, sequence, -1)
	}
	if domain.ProfileID(profileID) != reconciler.partition.ProfileID {
		return "", reject(protocol.CodeProfileMismatch, CauseMismatchedProfileID, protocol.ResyncProfileChanged, revision, sequence, -1)
	}
	if !knownContext(kind) {
		return "", missingIdentity(CauseMissingContextKind, protocol.ResyncProfileChanged, revision, sequence, -1)
	}
	if kind != reconciler.partition.ContextKind {
		return "", reject(protocol.CodeProfileMismatch, CauseMismatchedContextKind, protocol.ResyncProfileChanged, revision, sequence, -1)
	}
	if !validEpoch(epoch) {
		return "", missingIdentity(CauseMissingProjectionEpoch, protocol.ResyncRevisionMismatch, revision, sequence, -1)
	}
	if revision == 0 {
		return "", missingIdentity(CauseMissingProjectionRevision, protocol.ResyncRevisionMismatch, revision, sequence, -1)
	}
	return epoch, nil
}

// record maps one IP-07 tab record onto the IP-02 eligible record of the bound
// partition. The record must name the same profile, context, epoch, and revision the
// fence declared, and it must carry a real browser tab identity. A record that does
// not is refused at its position with a bounded cause instead of being mapped under
// a substituted identity.
func (reconciler *Reconciler) record(record protocol.TabRecord, epoch string, revision uint64, index int) (domain.EligibleTabRecord, *Rejection) {
	if _, err := domain.ParseProfileID(record.ProfileID); err != nil {
		return domain.EligibleTabRecord{}, missingIdentity(CauseMissingTabIdentityProfileID, protocol.ResyncProfileChanged, revision, 0, index)
	}
	if domain.ProfileID(record.ProfileID) != reconciler.partition.ProfileID {
		return domain.EligibleTabRecord{}, reject(protocol.CodeProfileMismatch, CauseMismatchedProfileID, protocol.ResyncProfileChanged, revision, 0, index)
	}
	if !knownContext(record.ContextKind) {
		return domain.EligibleTabRecord{}, missingIdentity(CauseMissingTabIdentityContextKind, protocol.ResyncProfileChanged, revision, 0, index)
	}
	if record.ContextKind != reconciler.partition.ContextKind {
		return domain.EligibleTabRecord{}, reject(protocol.CodeProfileMismatch, CauseMismatchedContextKind, protocol.ResyncProfileChanged, revision, 0, index)
	}
	if !usableTabID(record.TabID) {
		return domain.EligibleTabRecord{}, missingIdentity(CauseMissingTabIdentityTabID, "", revision, 0, index)
	}
	if record.ProjectionEpoch == "" {
		return domain.EligibleTabRecord{}, missingIdentity(CauseMissingProjectionEpoch, protocol.ResyncRevisionMismatch, revision, 0, index)
	}
	if record.ProjectionEpoch != epoch {
		return domain.EligibleTabRecord{}, missingIdentity(CauseMismatchedProjectionEpoch, protocol.ResyncRevisionMismatch, revision, 0, index)
	}
	if record.ProjectionRevision == 0 {
		return domain.EligibleTabRecord{}, missingIdentity(CauseMissingProjectionRevision, protocol.ResyncRevisionMismatch, revision, 0, index)
	}
	if record.ProjectionRevision != revision {
		return domain.EligibleTabRecord{}, reject(protocol.CodeRevisionMismatch, CauseMismatchedProjectionRevision, protocol.ResyncRevisionMismatch, revision, 0, index)
	}
	return domain.EligibleTabRecord{
		ProfileID:          reconciler.partition.ProfileID,
		ContextKind:        reconciler.partition.ContextKind,
		TabIdentity:        reconciler.partition.Identity(record.TabID),
		WindowID:           record.WindowID,
		GroupID:            record.GroupID,
		TitleDisplay:       record.TitleDisplay,
		TitleSearch:        record.TitleSearch,
		URLSearch:          record.URLSearch,
		URLDisplay:         record.URLDisplay,
		DomainDisplay:      record.DomainDisplay,
		DomainSearch:       record.DomainSearch,
		WindowLabelDisplay: record.WindowLabelDisplay,
		WindowLabelSearch:  record.WindowLabelSearch,
		GroupLabelDisplay:  record.GroupLabelDisplay,
		GroupLabelSearch:   record.GroupLabelSearch,
		Pinned:             record.Pinned,
		Active:             record.Active,
		Eligible:           record.Eligible,
		ObservedAt:         record.ObservedAt,
		ProjectionEpoch:    epoch,
		ProjectionRevision: revision,
	}, nil
}

// event maps one ordered IP-07 delta operation onto an IP-02 projection event. A
// remove carries only the tab identity; every other operation carries the full
// record of the tab it changes, and that record must describe the same tab in this
// partition, epoch, and target revision.
func (reconciler *Reconciler) event(delta protocol.Delta, index int, epoch string) (domain.ProjectionEvent, *Rejection) {
	operation := delta.Operations[index]
	sequence := delta.SequenceStart + uint64(index)
	predecessor := delta.BaseRevision + uint64(index)
	target := predecessor + 1
	if !operation.Kind.KnownOperation() {
		return domain.ProjectionEvent{}, reject(protocol.CodeInvalidFrame, CauseUnknownOperation, "", predecessor, sequence, index)
	}
	if operation.Sequence != sequence || sequence == 0 {
		return domain.ProjectionEvent{}, reject(protocol.CodeRevisionMismatch, CauseMismatchedProjectionRevision, protocol.ResyncSequenceGap, predecessor, sequence, index)
	}
	if !usableTabID(operation.TabID) {
		return domain.ProjectionEvent{}, missingIdentity(CauseMissingTabIdentityTabID, "", predecessor, sequence, index)
	}
	identity := reconciler.partition.Identity(operation.TabID)
	event := domain.ProjectionEvent{
		ProfileID:        reconciler.partition.ProfileID,
		ContextKind:      reconciler.partition.ContextKind,
		Epoch:            epoch,
		Sequence:         sequence,
		PreviousRevision: predecessor,
		Identity:         identity,
	}
	if operation.Kind == protocol.OperationRemove {
		if operation.Record != nil {
			return domain.ProjectionEvent{}, reject(protocol.CodeInvalidFrame, CauseUnexpectedRecord, "", predecessor, sequence, index)
		}
		event.Operation = "remove"
		return event, nil
	}
	if operation.Record == nil {
		return domain.ProjectionEvent{}, reject(protocol.CodeInvalidFrame, CauseMissingRecord, "", predecessor, sequence, index)
	}
	record, rejection := reconciler.record(*operation.Record, epoch, target, index)
	if rejection != nil {
		rejection.ExpectedSequence = sequence
		return domain.ProjectionEvent{}, rejection
	}
	if !record.TabIdentity.Equal(identity) {
		return domain.ProjectionEvent{}, reject(protocol.CodeInvalidFrame, CauseMismatchedTabIdentity, "", predecessor, sequence, index)
	}
	event.Operation = "upsert"
	event.Record = &record
	return event, nil
}

// knownContext reports whether a context kind is one of the two supported
// partitions. An absent or placeholder context such as "default" is not a partition,
// so it is refused instead of being treated as the normal context.
func knownContext(kind domain.ContextKind) bool {
	return kind == domain.ContextNormal || kind == domain.ContextPrivate
}

// usableTabID reports whether a browser tab id can carry a real tab identity.
// Chromium tab ids are positive, and an absent tab_identity.tab_id decodes to zero,
// which is exactly the value a synthetic fallback would produce; accepting it would
// publish a phantom record, so zero and negative ids are refused here.
func usableTabID(tabID int64) bool {
	return tabID > 0 && uint64(tabID) <= domain.MaxProjectionRevision
}

// validEpoch reports whether the declared epoch is a present, bounded, printable
// opaque value. The exact epoch lineage shape stays the IP-02 domain validator's rule
// at commit time; this boundary check only refuses an absent or unbounded epoch.
func validEpoch(epoch string) bool {
	if epoch == "" || len(epoch) > protocol.MaxEpochBytes {
		return false
	}
	for index := 0; index < len(epoch); index++ {
		if epoch[index] < 0x20 || epoch[index] == 0x7f {
			return false
		}
	}
	return true
}
