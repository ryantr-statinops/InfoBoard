package index

import (
	"net/url"
	"strings"
)

// URLSegments splits a URL into bounded host labels and path segments. It never
// emits the whole URL as one token and never performs a network request.
type URLSegments struct {
	Host []string
	Path []string
}

// ParseURLSegments parses and tokenizes a URL without retaining raw input.
func ParseURLSegments(raw string) (URLSegments, *NormalizeError) {
	if err := validate(raw); err != nil {
		return URLSegments{}, err
	}
	parsed, err := url.Parse(truncateScalars(raw, MaxURIScalars))
	if err != nil {
		return URLSegments{}, &NormalizeError{Code: ErrControlChar}
	}
	host := strings.Split(parsed.Hostname(), ".")
	path := strings.FieldsFunc(parsed.EscapedPath(), func(r rune) bool { return r == '/' })
	return URLSegments{
		Host: tokenize(normalizeForm(strings.Join(host, " "))),
		Path: tokenize(normalizeForm(strings.Join(path, " "))),
	}, nil
}
