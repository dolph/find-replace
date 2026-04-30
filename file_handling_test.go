package main

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func statOrFail(t testing.TB, path string) os.FileInfo {
	t.Helper()
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	return info
}

func TestLooksBinary(t *testing.T) {
	tests := []struct {
		name  string
		input []byte
		want  bool
	}{
		{"empty", []byte{}, false},
		{"plain text", []byte("hello world\n"), false},
		{"utf8", []byte("héllo wörld"), false},
		{"NUL byte", []byte("hello\x00world"), true},
		{"invalid UTF-8", []byte{0xff, 0xfe, 0xfd}, true},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := looksBinary(tc.input); got != tc.want {
				t.Errorf("looksBinary(%q) = %v, want %v", tc.input, got, tc.want)
			}
		})
	}
}

func TestStreamReplace(t *testing.T) {
	tests := []struct {
		name    string
		in      string
		find    string
		replace string
		want    string
		changed bool
	}{
		{"no match", "alpha bravo", "zulu", "yankee", "alpha bravo", false},
		{"single match", "alpha", "ph", "f", "alfa", true},
		{"multiple matches", "alphaalpha", "ph", "f", "alfaalfa", true},
		{"match at start", "alpha bravo", "alpha", "delta", "delta bravo", true},
		{"match at end", "alpha bravo", "bravo", "delta", "alpha delta", true},
		{"longer replacement", "ab", "a", "xxxxxx", "xxxxxxb", true},
		{"shorter replacement", "abcabc", "abc", "x", "xx", true},
		{"newlines preserved", "foo\nbar\nfoo", "foo", "BAZ", "BAZ\nbar\nBAZ", true},
		{"replace with empty", "remove this please", "this ", "", "remove please", true},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var out bytes.Buffer
			stats, err := streamReplace(&out, strings.NewReader(tc.in),
				[]byte(tc.find), []byte(tc.replace))
			if err != nil {
				t.Fatal(err)
			}
			if got := out.String(); got != tc.want {
				t.Errorf("output = %q, want %q", got, tc.want)
			}
			if stats.changed != tc.changed {
				t.Errorf("changed = %v, want %v", stats.changed, tc.changed)
			}
		})
	}
}

// TestStreamReplaceMatchAcrossBuffers exercises a match that spans the
// boundary between two buffer-fills. Builds an input that's larger than the
// rewrite buffer with the match deliberately straddling the boundary.
func TestStreamReplaceMatchAcrossBuffers(t *testing.T) {
	find := "needle"
	replace := "PIN"

	// Place the find string at offset rewriteBufSize - 3 so the first 3
	// bytes are in the first buffer fill and the last 3 are in the second.
	prefix := strings.Repeat("a", rewriteBufSize-3)
	tail := strings.Repeat("b", 100)
	in := prefix + find + tail
	want := prefix + replace + tail

	var out bytes.Buffer
	stats, err := streamReplace(&out, strings.NewReader(in), []byte(find), []byte(replace))
	if err != nil {
		t.Fatal(err)
	}
	if !stats.changed {
		t.Fatal("expected changed=true")
	}
	if got := out.String(); got != want {
		t.Errorf("boundary match not handled correctly (got len=%d, want len=%d)",
			len(got), len(want))
	}
}

func TestRewriteFileSkipsBinary(t *testing.T) {
	d := t.TempDir()
	path := filepath.Join(d, "binary")
	original := []byte{0x00, 0x01, 0x02, 'a', 'l', 'p', 'h', 'a'}
	if err := os.WriteFile(path, original, 0o600); err != nil {
		t.Fatal(err)
	}

	changed, err := rewriteFile(path, []byte("alpha"), []byte("BETA"), statOrFail(t, path))
	if err != nil {
		t.Fatal(err)
	}
	if changed {
		t.Error("expected binary file to be skipped")
	}
	got, _ := os.ReadFile(path)
	if !bytes.Equal(got, original) {
		t.Errorf("binary file was modified: %q", got)
	}
}

func TestRewriteFileNoMatchLeavesOriginalUntouched(t *testing.T) {
	d := t.TempDir()
	path := filepath.Join(d, "f.txt")
	if err := os.WriteFile(path, []byte("hello world"), 0o600); err != nil {
		t.Fatal(err)
	}
	stat0, _ := os.Stat(path)

	changed, err := rewriteFile(path, []byte("xyz"), []byte("abc"), statOrFail(t, path))
	if err != nil {
		t.Fatal(err)
	}
	if changed {
		t.Error("expected no change")
	}

	stat1, _ := os.Stat(path)
	if !stat0.ModTime().Equal(stat1.ModTime()) {
		t.Error("file was rewritten despite no match (mtime changed)")
	}

	// And no temp files left behind.
	entries, _ := os.ReadDir(d)
	if len(entries) != 1 {
		var names []string
		for _, e := range entries {
			names = append(names, e.Name())
		}
		t.Errorf("expected 1 file in dir, got %d: %v", len(entries), names)
	}
}

func TestRewriteFileLeavesNoTempfile(t *testing.T) {
	d := t.TempDir()
	path := filepath.Join(d, "f.txt")
	if err := os.WriteFile(path, []byte("hello alpha world"), 0o600); err != nil {
		t.Fatal(err)
	}

	changed, err := rewriteFile(path, []byte("alpha"), []byte("BETA"), statOrFail(t, path))
	if err != nil {
		t.Fatal(err)
	}
	if !changed {
		t.Error("expected change=true")
	}

	entries, _ := os.ReadDir(d)
	if len(entries) != 1 {
		var names []string
		for _, e := range entries {
			names = append(names, e.Name())
		}
		t.Errorf("expected exactly one file (no temp files leaked), got %v", names)
	}
}

func TestRewriteFilePreservesMode(t *testing.T) {
	d := t.TempDir()
	path := filepath.Join(d, "f.txt")
	if err := os.WriteFile(path, []byte("alpha"), 0o640); err != nil {
		t.Fatal(err)
	}

	changed, err := rewriteFile(path, []byte("alpha"), []byte("BETA"), statOrFail(t, path))
	if err != nil {
		t.Fatal(err)
	}
	if !changed {
		t.Fatal("expected change=true")
	}
	stat, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if stat.Mode().Perm() != 0o640 {
		t.Errorf("mode = %v, want %v", stat.Mode().Perm(), os.FileMode(0o640))
	}
}

func TestRenameNoReplaceFailsOnExist(t *testing.T) {
	d := t.TempDir()
	src := filepath.Join(d, "src")
	dst := filepath.Join(d, "dst")
	if err := os.WriteFile(src, []byte("source"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(dst, []byte("destination"), 0o600); err != nil {
		t.Fatal(err)
	}

	err := renameNoReplace(src, dst)
	if !errors.Is(err, os.ErrExist) {
		t.Errorf("expected ErrExist, got %v", err)
	}

	// Source must still exist; dst content untouched.
	if _, err := os.Stat(src); err != nil {
		t.Errorf("source was removed: %v", err)
	}
	got, _ := os.ReadFile(dst)
	if string(got) != "destination" {
		t.Errorf("dst was overwritten: %q", got)
	}
}

func TestRenameNoReplaceSucceeds(t *testing.T) {
	d := t.TempDir()
	src := filepath.Join(d, "src")
	dst := filepath.Join(d, "dst")
	if err := os.WriteFile(src, []byte("source"), 0o600); err != nil {
		t.Fatal(err)
	}

	if err := renameNoReplace(src, dst); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(src); !errors.Is(err, os.ErrNotExist) {
		t.Errorf("source still exists: %v", err)
	}
	got, _ := os.ReadFile(dst)
	if string(got) != "source" {
		t.Errorf("dst content = %q, want source", got)
	}
}
