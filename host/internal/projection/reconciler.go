package projection

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"sort"

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
	CauseInvalidRecord                 Cause = "invalid_record"
	CauseDuplicateTabIdentity          Cause = "duplicate_tab_identity"
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
	CauseMissingTabIdentityTabID: {}, CauseMissingRecord: {}, CauseInvalidRecord: {}, CauseDuplicateTabIdentity: {},
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

// SnapshotInput is one validated authoritative snapshot. Records are in canonical
// TabIdentity order, and Digest is the fixed-length SHA-256 of their canonical JSON
// encoding. The original IP-04 fence is carried unchanged to the later commit seam.
type SnapshotInput struct {
	Partition Partition
	Epoch     string
	Revision  uint64
	Sequence  uint64
	Records   []domain.EligibleTabRecord
	Digest    string
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

// DeltaOperation is one validated IP-07 operation mapped onto the bound IP-02
// identity. Record is nil only for remove; its projection_revision distinguishes an
// effective upsert from an accepted no-op upsert.
type DeltaOperation struct {
	Sequence uint64
	Kind     protocol.OperationKind
	Identity domain.TabIdentity
	Record   *domain.EligibleTabRecord
}

// DeltaInput is a fully validated, ordered delta batch. It preserves the wire
// predecessor, target revision, and IP-04 sequence range for the reducer; no
// sequence or revision is allocated at this boundary.
type DeltaInput struct {
	Partition     Partition
	Epoch         string
	BaseRevision  uint64
	Revision      uint64
	SequenceStart uint64
	SequenceEnd   uint64
	Operations    []DeltaOperation
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
// is refused as a mismatch, before any state is touched.
//
// A fully fenced snapshot for the known profile/context is authority even when it
// carries no record at all: zero records is an intentional successful empty
// projection, not a permission failure, a partial read, or unavailable profile
// state. The IP-07 payload has no field for how the browser read finished, so an
// acquisition that did not complete is not a snapshot at all: it arrives as a
// status or resync signal and leaves the committed lineage untouched. Because that
// distinction is carried by the peer, the host accepts authority only for a
// complete fence and proves every record against the IP-02 record contract before
// returning it. Wire shape and scalar budgets stay owned by the IP-07 decoder and
// are not re-implemented here.
func (reconciler *Reconciler) BindSnapshot(request SnapshotRequest) (SnapshotInput, *Rejection) {
	if len(request.Records) > protocol.MaxSnapshotRecords {
		return SnapshotInput{}, missingIdentity(CauseBoundsExceeded, "", 0, 0, -1)
	}
	if request.Revision == 0 || request.Sequence == 0 {
		return SnapshotInput{}, missingIdentity(CauseMissingProjectionRevision, protocol.ResyncRevisionMismatch, 0, 0, -1)
	}
	if request.Revision != 1 {
		return SnapshotInput{}, reject(protocol.CodeSnapshotRequired, CauseMismatchedProjectionRevision, protocol.ResyncRevisionMismatch, 1, request.Sequence, -1)
	}
	epoch, rejection := reconciler.fence(request.ProfileID, request.ContextKind, request.Epoch, request.Revision, request.Sequence)
	if rejection != nil {
		return SnapshotInput{}, rejection
	}
	records := make([]domain.EligibleTabRecord, 0, len(request.Records))
	seen := make(map[domain.TabIdentity]struct{}, len(request.Records))
	for index := range request.Records {
		record, rejection := reconciler.record(request.Records[index], epoch, request.Revision, index)
		if rejection != nil {
			return SnapshotInput{}, rejection
		}
		// A record that IP-02 would refuse is refused here, so no record outside the
		// domain contract can reach an accepted SnapshotInput. The IP-07 decoder
		// bounds the same values more loosely than IP-02, and it cannot see the
		// eligibility rule or the epoch shape at all. The remedy is a complete
		// authoritative snapshot, so the snapshot is reported as stale rather than
		// as an absent field.
		if !record.Valid(reconciler.partition.ProfileID, reconciler.partition.ContextKind, epoch) {
			return SnapshotInput{}, reject(protocol.CodeSnapshotRequired, CauseInvalidRecord, protocol.ResyncSnapshotStale, request.Revision, 0, index)
		}
		if _, exists := seen[record.TabIdentity]; exists {
			return SnapshotInput{}, reject(protocol.CodeSnapshotRequired, CauseDuplicateTabIdentity, protocol.ResyncSnapshotStale, request.Revision, 0, index)
		}
		seen[record.TabIdentity] = struct{}{}
		records = append(records, record)
	}
	sort.Slice(records, func(i, j int) bool {
		return identityLess(records[i].TabIdentity, records[j].TabIdentity)
	})
	digest, err := canonicalSnapshotDigest(records)
	if err != nil {
		return SnapshotInput{}, reject(protocol.CodeSnapshotRequired, CauseInvalidRecord, protocol.ResyncSnapshotStale, request.Revision, 0, -1)
	}
	return SnapshotInput{
		Partition: reconciler.partition,
		Epoch:     epoch,
		Revision:  request.Revision,
		Sequence:  request.Sequence,
		Records:   records,
		Digest:    digest,
	}, nil
}

// canonicalSnapshotDigest hashes the canonical JSON encoding of sorted records. Each
// record is encoded separately so the digest does not allocate a second full snapshot.
func canonicalSnapshotDigest(records []domain.EligibleTabRecord) (string, error) {
	hash := sha256.New()
	_, _ = hash.Write([]byte("InfoBoard projection snapshot v1\x00["))
	for index, record := range records {
		if index > 0 {
			_, _ = hash.Write([]byte{','})
		}
		encoded, err := json.Marshal(record)
		if err != nil {
			return "", err
		}
		_, _ = hash.Write(encoded)
	}
	_, _ = hash.Write([]byte{']'})
	return hex.EncodeToString(hash.Sum(nil)), nil
}

// BindDelta validates one bounded wire batch without deriving predecessor revisions
// from operation position: no-op operations consume an IP-04 sequence without
// advancing projection_revision. The state reducer checks the evolving predecessor
// revision and decides which operations are effective.
func (reconciler *Reconciler) BindDelta(request DeltaRequest) (DeltaInput, *Rejection) {
	delta := request.Delta
	if len(delta.Operations) == 0 || len(delta.Operations) > protocol.MaxDeltaOperations {
		return DeltaInput{}, missingIdentity(CauseBoundsExceeded, "", 0, 0, -1)
	}
	if request.Revision == 0 || delta.BaseRevision == 0 {
		return DeltaInput{}, reject(protocol.CodeRevisionMismatch, CauseMissingProjectionRevision, protocol.ResyncRevisionMismatch, delta.BaseRevision, delta.SequenceStart, -1)
	}
	if request.Revision < delta.BaseRevision || request.Revision > domain.MaxProjectionRevision {
		return DeltaInput{}, reject(protocol.CodeRevisionMismatch, CauseMismatchedProjectionRevision, protocol.ResyncRevisionMismatch, delta.BaseRevision, delta.SequenceStart, -1)
	}
	if delta.SequenceStart == 0 || delta.SequenceEnd < delta.SequenceStart || delta.SequenceEnd-delta.SequenceStart+1 != uint64(len(delta.Operations)) {
		return DeltaInput{}, reject(protocol.CodeRevisionMismatch, CauseMismatchedProjectionRevision, protocol.ResyncSequenceGap, delta.BaseRevision, delta.SequenceStart, -1)
	}
	epoch, rejection := reconciler.fence(request.ProfileID, request.ContextKind, request.Epoch, request.Revision, delta.SequenceStart)
	if rejection != nil {
		return DeltaInput{}, rejection
	}
	operations := make([]DeltaOperation, 0, len(delta.Operations))
	for index, wire := range delta.Operations {
		sequence := delta.SequenceStart + uint64(index)
		if wire.Sequence != sequence {
			return DeltaInput{}, reject(protocol.CodeRevisionMismatch, CauseMismatchedProjectionRevision, protocol.ResyncSequenceGap, delta.BaseRevision, sequence, index)
		}
		if !wire.Kind.KnownOperation() {
			return DeltaInput{}, reject(protocol.CodeInvalidFrame, CauseUnknownOperation, "", delta.BaseRevision, sequence, index)
		}
		if !usableTabID(wire.TabID) {
			return DeltaInput{}, missingIdentity(CauseMissingTabIdentityTabID, "", delta.BaseRevision, sequence, index)
		}
		identity := reconciler.partition.Identity(wire.TabID)
		operation := DeltaOperation{Sequence: sequence, Kind: wire.Kind, Identity: identity}
		if wire.Kind == protocol.OperationRemove {
			if wire.Record != nil {
				return DeltaInput{}, reject(protocol.CodeInvalidFrame, CauseUnexpectedRecord, "", delta.BaseRevision, sequence, index)
			}
			operations = append(operations, operation)
			continue
		}
		if wire.Record == nil {
			return DeltaInput{}, reject(protocol.CodeInvalidFrame, CauseMissingRecord, "", delta.BaseRevision, sequence, index)
		}
		recordRevision := wire.Record.ProjectionRevision
		if recordRevision < delta.BaseRevision || recordRevision > request.Revision {
			return DeltaInput{}, reject(protocol.CodeRevisionMismatch, CauseMismatchedProjectionRevision, protocol.ResyncRevisionMismatch, request.Revision, sequence, index)
		}
		record, rejection := reconciler.record(*wire.Record, epoch, recordRevision, index)
		if rejection != nil {
			rejection.ExpectedSequence = sequence
			return DeltaInput{}, rejection
		}
		if !record.Valid(reconciler.partition.ProfileID, reconciler.partition.ContextKind, epoch) {
			return DeltaInput{}, reject(protocol.CodeSnapshotRequired, CauseInvalidRecord, protocol.ResyncSnapshotStale, request.Revision, sequence, index)
		}
		if record.TabIdentity != identity {
			return DeltaInput{}, reject(protocol.CodeInvalidFrame, CauseMismatchedTabIdentity, "", request.Revision, sequence, index)
		}
		operation.Record = &record
		operations = append(operations, operation)
	}
	return DeltaInput{
		Partition:     reconciler.partition,
		Epoch:         epoch,
		BaseRevision:  delta.BaseRevision,
		Revision:      request.Revision,
		SequenceStart: delta.SequenceStart,
		SequenceEnd:   delta.SequenceEnd,
		Operations:    operations,
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
