package index

// FieldSpec describes one indexable field and whether it accepts missing input.
type FieldSpec struct {
	Field      Field
	Optional   bool
	MaxScalars int
}

// FieldTable is the closed field table for a tab record.
var FieldTable = []FieldSpec{
	{Field: FieldTitle, Optional: false, MaxScalars: MaxTitleScalars},
	{Field: FieldURL, Optional: false, MaxScalars: MaxURIScalars},
	{Field: FieldDomain, Optional: false, MaxScalars: MaxDomainScalars},
	{Field: FieldWindowLabel, Optional: true, MaxScalars: MaxLabelScalars},
	{Field: FieldGroupLabel, Optional: true, MaxScalars: MaxLabelScalars},
	{Field: FieldPinned, Optional: true, MaxScalars: MaxTokenScalars},
	{Field: FieldActive, Optional: true, MaxScalars: MaxTokenScalars},
}

// LookupField returns the bounded spec for a field.
func LookupField(field Field) (FieldSpec, bool) {
	for _, spec := range FieldTable {
		if spec.Field == field {
			return spec, true
		}
	}
	return FieldSpec{}, false
}
