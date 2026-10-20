package rank

// Package rank owns deterministic ranking and explanations (IP-11).
type Ranker struct{}

func New() *Ranker { return &Ranker{} }

