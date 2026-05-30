package main

import (
	"bytes"
	"io"
	"strings"
	"testing"
)

func TestStreamReplace(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		input   string
		find    string
		replace string
		want    string
	}{
		{name: "no match", input: "hello", find: "z", replace: "q", want: "hello"},
		{name: "simple", input: "foo bar foo", find: "foo", replace: "baz", want: "baz bar baz"},
		{name: "span boundary", input: "xxababc", find: "ab", replace: "X", want: "xxXXc"},
		{name: "empty input", input: "", find: "a", replace: "b", want: ""},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			var out bytes.Buffer
			changed, err := streamReplace(strings.NewReader(tc.input), &out, []byte(tc.find), []byte(tc.replace))
			if err != nil {
				t.Fatal(err)
			}
			if tc.input != tc.want && !changed {
				t.Fatal("expected changed=true")
			}
			if tc.input == tc.want && changed {
				t.Fatal("expected changed=false")
			}
			if out.String() != tc.want {
				t.Fatalf("got %q; want %q", out.String(), tc.want)
			}
		})
	}
}

func TestStreamReplaceLargeWithSmallReads(t *testing.T) {
	find := "needle"
	replace := "pin"
	input := strings.Repeat("hay", 1000) + find + strings.Repeat("stack", 1000)
	want := strings.Replace(input, find, replace, 1)

	var out bytes.Buffer
	r := &smallReader{data: []byte(input), step: 3}
	changed, err := streamReplace(r, &out, []byte(find), []byte(replace))
	if err != nil {
		t.Fatal(err)
	}
	if !changed {
		t.Fatal("expected replacement")
	}
	if out.String() != want {
		t.Fatalf("output length %d; want %d", out.Len(), len(want))
	}
}

type smallReader struct {
	data []byte
	step int
	off  int
}

func (r *smallReader) Read(p []byte) (int, error) {
	if r.off >= len(r.data) {
		return 0, io.EOF
	}
	n := r.step
	if n > len(p) {
		n = len(p)
	}
	if n > len(r.data)-r.off {
		n = len(r.data) - r.off
	}
	copy(p, r.data[r.off:r.off+n])
	r.off += n
	return n, nil
}
