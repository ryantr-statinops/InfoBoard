package index

// Form keeps the display and search representations of one field independent.
// Display text is whitespace-sanitized but preserves user-visible casing; search
// text is NFKC-folded and whitespace-collapsed for matching.
type Form struct {
	Field   Field
	Display string
	Search  string
}

// BuildForm validates, bounds, and normalizes one field. A missing optional field
// is represented by an empty form rather than a fabricated token.
func BuildForm(field Field, raw string, maxScalars int) (Form, *NormalizeError) {
	if err := validate(raw); err != nil {
		return Form{}, err
	}
	display := collapseWhitespace(truncateScalars(raw, maxScalars))
	search := collapseWhitespace(normalizeForm(display))
	return Form{Field: field, Display: display, Search: search}, nil
}
