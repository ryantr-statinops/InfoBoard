package domain

import (
	"encoding/json"
	"errors"
	"regexp"
	"sort"
	"unicode/utf8"
)

const MaxProjectionRevision uint64 = 9007199254740991
const maxRememberedEvents = 1000

type ProfileID string
type ContextKind string

const (
	ContextNormal  ContextKind = "normal"
	ContextPrivate ContextKind = "private"
)

var uuidV4 = regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-4[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$`)

func ParseProfileID(value string) (ProfileID, error) {
	if len(value) != 36 || !uuidV4.MatchString(value) {
		return "", errors.New("INVALID_PROFILE_ID")
	}
	return ProfileID(value), nil
}
func validContext(kind ContextKind) bool { return kind == ContextNormal || kind == ContextPrivate }

type TabIdentity struct {
	ProfileID   ProfileID   `json:"profile_id"`
	ContextKind ContextKind `json:"context_kind"`
	TabID       int64       `json:"tab_id"`
}

func (identity TabIdentity) Equal(other TabIdentity) bool { return identity == other }

type EligibleTabRecord struct {
	ProfileID          ProfileID   `json:"profile_id"`
	ContextKind        ContextKind `json:"context_kind"`
	TabIdentity        TabIdentity `json:"tab_identity"`
	WindowID           int64       `json:"window_id"`
	GroupID            *int64      `json:"group_id"`
	TitleDisplay       string      `json:"title_display"`
	TitleSearch        string      `json:"title_search"`
	URLSearch          string      `json:"url_search"`
	URLDisplay         string      `json:"url_display"`
	DomainDisplay      string      `json:"domain_display"`
	DomainSearch       string      `json:"domain_search"`
	WindowLabelDisplay *string     `json:"window_label_display,omitempty"`
	WindowLabelSearch  *string     `json:"window_label_search,omitempty"`
	GroupLabelDisplay  *string     `json:"group_label_display,omitempty"`
	GroupLabelSearch   *string     `json:"group_label_search,omitempty"`
	Pinned             bool        `json:"pinned"`
	Active             bool        `json:"active"`
	Eligible           bool        `json:"eligible"`
	ObservedAt         int64       `json:"observed_at"`
	ProjectionEpoch    string      `json:"projection_epoch"`
	ProjectionRevision uint64      `json:"projection_revision"`
}

func boundedText(value string, maxBytes, maxScalars int) bool {
	return len(value) <= maxBytes && (maxScalars == 0 || utf8.RuneCountInString(value) <= maxScalars)
}
func optionalBounded(value *string, maxBytes, maxScalars int) bool {
	return value == nil || boundedText(*value, maxBytes, maxScalars)
}
func (record EligibleTabRecord) Valid(profile ProfileID, kind ContextKind, epoch string) bool {
	identity := record.TabIdentity
	return validContext(kind) && record.ProfileID == profile && record.ContextKind == kind && identity.ProfileID == profile && identity.ContextKind == kind && identity.TabID >= 0 && uint64(identity.TabID) <= MaxProjectionRevision && record.WindowID >= 0 && uint64(record.WindowID) <= MaxProjectionRevision && (record.GroupID == nil || (*record.GroupID >= 0 && uint64(*record.GroupID) <= MaxProjectionRevision)) && boundedText(record.TitleDisplay, 2048, 512) && boundedText(record.TitleSearch, 2048, 512) && boundedText(record.URLSearch, 2048, 0) && boundedText(record.URLDisplay, 2048, 0) && boundedText(record.DomainDisplay, 255, 0) && boundedText(record.DomainSearch, 255, 0) && optionalBounded(record.WindowLabelDisplay, 2048, 512) && optionalBounded(record.WindowLabelSearch, 2048, 512) && optionalBounded(record.GroupLabelDisplay, 2048, 512) && optionalBounded(record.GroupLabelSearch, 2048, 512) && record.Eligible && record.ObservedAt >= 0 && record.ProjectionRevision <= MaxProjectionRevision && record.ProjectionEpoch == epoch && uuidV4.MatchString(epoch)
}

type ProjectionState struct {
	ProfileID   ProfileID
	ContextKind ContextKind
	Epoch       string
	Revision    uint64
	Records     map[TabIdentity]EligibleTabRecord
	Seen        map[uint64]string
}

func NewProjection(profile ProfileID, kind ContextKind) *ProjectionState {
	return &ProjectionState{ProfileID: profile, ContextKind: kind, Records: make(map[TabIdentity]EligibleTabRecord), Seen: make(map[uint64]string)}
}
func (state *ProjectionState) Snapshot(epoch string, rows []EligibleTabRecord) error {
	if _, err := ParseProfileID(string(state.ProfileID)); err != nil || !validContext(state.ContextKind) || !uuidV4.MatchString(epoch) || len(rows) > 10000 {
		return errors.New("SNAPSHOT_REQUIRED")
	}
	next := make(map[TabIdentity]EligibleTabRecord, len(rows))
	for _, row := range rows {
		if !row.Valid(state.ProfileID, state.ContextKind, epoch) || row.ProjectionRevision != 1 {
			return errors.New("SNAPSHOT_REQUIRED")
		}
		if _, exists := next[row.TabIdentity]; exists {
			return errors.New("SNAPSHOT_REQUIRED")
		}
		next[row.TabIdentity] = row
	}
	state.Epoch, state.Revision, state.Records, state.Seen = epoch, 1, next, make(map[uint64]string)
	return nil
}

type ProjectionEvent struct {
	ProfileID        ProfileID          `json:"profile_id"`
	ContextKind      ContextKind        `json:"context_kind"`
	Epoch            string             `json:"epoch"`
	Sequence         uint64             `json:"sequence"`
	PreviousRevision uint64             `json:"previous_revision"`
	Operation        string             `json:"operation"`
	Identity         TabIdentity        `json:"tab_identity"`
	Record           *EligibleTabRecord `json:"record,omitempty"`
}

func (state *ProjectionState) Apply(event ProjectionEvent) string {
	if event.ProfileID != state.ProfileID || event.ContextKind != state.ContextKind || !validContext(event.ContextKind) || event.Epoch != state.Epoch || state.Epoch == "" || event.Sequence == 0 || event.Sequence > MaxProjectionRevision || event.PreviousRevision > MaxProjectionRevision {
		return "SNAPSHOT_REQUIRED"
	}
	raw, err := json.Marshal(event)
	if err != nil {
		return "SNAPSHOT_REQUIRED"
	}
	signature := string(raw)
	if old, exists := state.Seen[event.Sequence]; exists {
		if old == signature {
			return "DUPLICATE"
		}
		return "REVISION_MISMATCH"
	}
	if state.Revision >= MaxProjectionRevision {
		return "REVISION_MISMATCH"
	}
	if event.Sequence != state.Revision+1 || event.PreviousRevision != state.Revision {
		return "SNAPSHOT_REQUIRED"
	}
	if event.Identity.ProfileID != state.ProfileID || event.Identity.ContextKind != state.ContextKind || event.Identity.TabID < 0 || uint64(event.Identity.TabID) > MaxProjectionRevision {
		return "SNAPSHOT_REQUIRED"
	}
	if event.Operation == "upsert" && (event.Record == nil || !event.Record.Valid(state.ProfileID, state.ContextKind, state.Epoch) || event.Record.ProjectionRevision != state.Revision+1 || event.Record.TabIdentity != event.Identity) {
		return "SNAPSHOT_REQUIRED"
	}
	if event.Operation == "remove" {
		if _, exists := state.Records[event.Identity]; !exists {
			return "SNAPSHOT_REQUIRED"
		}
	} else if event.Operation != "upsert" {
		return "SNAPSHOT_REQUIRED"
	}
	next := make(map[TabIdentity]EligibleTabRecord, len(state.Records)+1)
	for identity, row := range state.Records {
		next[identity] = row
	}
	if event.Operation == "remove" {
		delete(next, event.Identity)
	} else {
		next[event.Identity] = *event.Record
	}
	state.Records, state.Revision = next, state.Revision+1
	state.Seen[event.Sequence] = signature
	if len(state.Seen) > maxRememberedEvents {
		delete(state.Seen, event.Sequence-maxRememberedEvents)
	}
	return "APPLIED"
}
func (state *ProjectionState) ApplyBatch(events []ProjectionEvent) string {
	if len(events) > 1000 {
		return "SNAPSHOT_REQUIRED"
	}
	staged := *state
	staged.Records = make(map[TabIdentity]EligibleTabRecord, len(state.Records))
	for identity, row := range state.Records {
		staged.Records[identity] = row
	}
	staged.Seen = make(map[uint64]string, len(state.Seen))
	for sequence, signature := range state.Seen {
		staged.Seen[sequence] = signature
	}
	applied := false
	for _, event := range events {
		outcome := staged.Apply(event)
		if outcome == "SNAPSHOT_REQUIRED" || outcome == "REVISION_MISMATCH" {
			return outcome
		}
		if outcome == "APPLIED" {
			applied = true
		}
	}
	if !applied {
		return "DUPLICATE"
	}
	state.Epoch, state.Revision, state.Records, state.Seen = staged.Epoch, staged.Revision, staged.Records, staged.Seen
	return "APPLIED"
}

type ActivationReference struct {
	ProfileID          ProfileID   `json:"profile_id"`
	ContextKind        ContextKind `json:"context_kind"`
	TabID              int64       `json:"tab_id"`
	ProjectionEpoch    string      `json:"projection_epoch"`
	ProjectionRevision uint64      `json:"projection_revision"`
	ResultID           string      `json:"result_id"`
}

func (reference ActivationReference) Valid(state *ProjectionState) bool {
	identity := TabIdentity{ProfileID: reference.ProfileID, ContextKind: reference.ContextKind, TabID: reference.TabID}
	_, exists := state.Records[identity]
	return state.Epoch != "" && reference.ProfileID == state.ProfileID && reference.ContextKind == state.ContextKind && reference.ProjectionEpoch == state.Epoch && reference.ProjectionRevision == state.Revision && reference.TabID >= 0 && uint64(reference.TabID) <= MaxProjectionRevision && boundedText(reference.ResultID, 128, 0) && exists
}

type ActivationSource string

const (
	ActivationKeyboardEnter ActivationSource = "keyboard_enter"
	ActivationSurfaceSelect ActivationSource = "surface_select"
)

type ActivationMetadata struct {
	ProfileID       ProfileID        `json:"profile_id"`
	ContextKind     ContextKind      `json:"context_kind"`
	TabIdentity     TabIdentity      `json:"tab_identity"`
	Domain          string           `json:"domain"`
	ActivatedAt     int64            `json:"activated_at"`
	Source          ActivationSource `json:"source"`
	StorageSequence uint64           `json:"storage_sequence"`
}

func RetainActivations(rows []ActivationMetadata, now int64) []ActivationMetadata {
	retained := make([]ActivationMetadata, 0, len(rows))
	for _, row := range rows {
		validSource := row.Source == ActivationKeyboardEnter || row.Source == ActivationSurfaceSelect
		if row.ContextKind == ContextNormal && row.TabIdentity.ProfileID == row.ProfileID && row.TabIdentity.ContextKind == ContextNormal && boundedText(row.Domain, 255, 0) && row.ActivatedAt >= now-30*86400 && row.ActivatedAt <= now && validSource {
			retained = append(retained, row)
		}
	}
	sort.Slice(retained, func(i, j int) bool {
		if retained[i].ActivatedAt == retained[j].ActivatedAt {
			return retained[i].StorageSequence < retained[j].StorageSequence
		}
		return retained[i].ActivatedAt < retained[j].ActivatedAt
	})
	if len(retained) > 500 {
		retained = retained[len(retained)-500:]
	}
	return retained
}
