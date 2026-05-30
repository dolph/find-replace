package main

import (
	"bytes"
	"unicode/utf8"
)

// isTextBytes reports whether b is valid UTF-8 and contains no NUL bytes.
func isTextBytes(b []byte) bool {
	if len(b) == 0 {
		return true
	}
	if bytes.IndexByte(b, 0) >= 0 {
		return false
	}
	return utf8.Valid(b)
}
