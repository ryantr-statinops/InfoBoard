package index

// TokenRef records where an inferred token came from. Positions are zero-based
// indices in the normalized search form; they never expose raw input.
type TokenRef struct {
	Field    Field
	Position int
	Source   string
}

// Provenance is the ordered, bounded set of token references for one record.
type Provenance struct {
	Refs []TokenRef
}

// Append adds one reference unless the per-field bound is reached.
func (p *Provenance) Append(ref TokenRef) {
	p.Refs = append(p.Refs, ref)
}
