// Package index owns lexical normalization and in-memory index (IP-10).
package index

// Index is the in-memory lexical index.
type Index struct{}

func New() *Index { return &Index{} }
