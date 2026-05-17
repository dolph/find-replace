package main

import (
	"testing"
)

func TestIsTextBytes(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		in   []byte
		want bool
	}{
		{name: "empty", in: nil, want: true},
		{name: "ascii", in: []byte("hello"), want: true},
		{name: "nul", in: []byte("hi\x00there"), want: false},
		{name: "invalid utf8", in: []byte{0xff, 0xfe}, want: false},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if got := isTextBytes(tc.in); got != tc.want {
				t.Fatalf("isTextBytes(%q) = %v; want %v", tc.in, got, tc.want)
			}
		})
	}
}
