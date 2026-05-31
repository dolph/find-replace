package main

import (
	"fmt"
	"regexp"
)

// Pattern is the shared match/replace engine for file contents and renames.
// Literal mode escapes the pattern with regexp.QuoteMeta so existing
// find/replace semantics are preserved without exposing a regex flag yet.
type Pattern struct {
	re      *regexp.Regexp
	replace string
	literal bool
}

// Compile builds a Pattern. When literal is true, pattern is treated as a
// plain string; otherwise it is interpreted as RE2 syntax with $N capture
// references supported in replacement.
func Compile(pattern, replacement string, literal bool) (*Pattern, error) {
	expr := pattern
	if literal {
		expr = regexp.QuoteMeta(pattern)
	}
	re, err := regexp.Compile(expr)
	if err != nil {
		return nil, fmt.Errorf("compile pattern: %w", err)
	}
	return &Pattern{
		re:      re,
		replace: replacement,
		literal: literal,
	}, nil
}

func (p *Pattern) Match(s string) bool {
	return p.re.MatchString(s)
}

func (p *Pattern) Replace(s string) string {
	return p.re.ReplaceAllString(s, p.replace)
}
