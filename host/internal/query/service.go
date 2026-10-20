package query

// Package query owns query API and degraded results (IP-12).
type Service struct{}

func New() *Service { return &Service{} }

