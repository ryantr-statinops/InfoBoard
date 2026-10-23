package index

// Eligibility is the posting gate for one tab record. An ineligible tab never
// produces a posting, regardless of how many fields it carries.
type Eligibility struct {
	Eligible bool
}

// AllowsPostings reports whether the record may enter the index.
func (e Eligibility) AllowsPostings() bool { return e.Eligible }
