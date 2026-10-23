// Package index owns InfoBoard's in-memory lexical normalization and index.
// It performs no network, storage, browser API, page evaluation, or subprocess work.
package index

// NormalizerVersion stamps every normalized representation so a rebuild is
// reproducible and a stale index can be identified.
const NormalizerVersion = "lex-1"

// Field is the closed set of indexed tab fields. State fields are explicit so
// producers never fold pinned/active into free text.
type Field string

const (
	FieldTitle       Field = "title"
	FieldURL         Field = "url"
	FieldDomain      Field = "domain"
	FieldWindowLabel Field = "window_label"
	FieldGroupLabel  Field = "group_label"
	FieldPinned      Field = "pinned"
	FieldActive      Field = "active"
)

// Bounds are the scalar budgets applied before token allocation.
const (
	MaxTitleScalars   = 512
	MaxURIScalars     = 2048
	MaxLabelScalars   = 128
	MaxDomainScalars  = 253
	MaxTokenScalars   = 64
	MaxQueryScalars   = 64
	MaxTokensPerField = 64
)
