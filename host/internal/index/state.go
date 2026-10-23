package index

// StateLabels returns the explicit, queryable state labels for a tab. Labels are
// closed values so they stay stable across locales and rebuilds.
func StateLabels(pinned, active bool) []Form {
	labels := make([]Form, 0, 2)
	if pinned {
		labels = append(labels, Form{Field: FieldPinned, Display: "pinned", Search: "pinned"})
	}
	if active {
		labels = append(labels, Form{Field: FieldActive, Display: "active", Search: "active"})
	}
	return labels
}
