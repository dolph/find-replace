package main

import (
	"os"
	"path/filepath"
	"testing"
)

// TestNewFile exercises NewFile's path-resolution behavior.
func TestNewFile(t *testing.T) {
	tmp := t.TempDir()

	tests := []struct {
		name string
		// input is the raw path passed to NewFile.
		input string
		// want is the absolute path the resulting *File should expose. If
		// empty, the test computes the expected value via filepath.Abs(input)
		// at runtime (useful for inputs that are inherently relative to the
		// test process's working directory).
		want string
	}{
		{
			name:  "absolute path is returned cleaned",
			input: filepath.Join(tmp, "foo"),
			want:  filepath.Join(tmp, "foo"),
		},
		{
			name:  "absolute path with redundant separators is cleaned",
			input: tmp + "//foo///bar",
			want:  filepath.Join(tmp, "foo", "bar"),
		},
		{
			name:  "absolute path with .. is resolved",
			input: filepath.Join(tmp, "a", "..", "b"),
			want:  filepath.Join(tmp, "b"),
		},
		{
			name:  "relative path is resolved to absolute",
			input: "relative/path",
			// want is computed below because it depends on the test
			// process's working directory.
		},
		{
			name:  "relative path with .. is resolved",
			input: "a/../b",
		},
		{
			name:  "dot is resolved to the working directory",
			input: ".",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			want := tc.want
			if want == "" {
				abs, err := filepath.Abs(tc.input)
				if err != nil {
					t.Fatalf("filepath.Abs(%q) returned unexpected error: %v", tc.input, err)
				}
				want = abs
			}

			got, err := NewFile(tc.input)
			if err != nil {
				t.Fatalf("NewFile(%q) returned unexpected error: %v", tc.input, err)
			}
			if got == nil {
				t.Fatalf("NewFile(%q) returned nil", tc.input)
			}
			if got.Path != want {
				t.Errorf("NewFile(%q).Path = %q; want %q", tc.input, got.Path, want)
			}
			if !filepath.IsAbs(got.Path) {
				t.Errorf("NewFile(%q).Path = %q; want an absolute path", tc.input, got.Path)
			}
		})
	}
}

func TestReadSkipsBinaryWithNUL(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "mixed.txt")
	content := []byte("text prefix\x00binary suffix")
	if err := os.WriteFile(path, content, 0o644); err != nil {
		t.Fatal(err)
	}

	f, err := NewFile(path)
	if err != nil {
		t.Fatal(err)
	}
	got, err := f.Read()
	if err != nil {
		t.Fatal(err)
	}
	if got != "" {
		t.Fatalf("Read() = %q; want empty for NUL-containing file", got)
	}
}

func TestReadReturnsShortTextFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "short.txt")
	if err := os.WriteFile(path, []byte("hello"), 0o644); err != nil {
		t.Fatal(err)
	}
	f, err := NewFile(path)
	if err != nil {
		t.Fatal(err)
	}
	got, err := f.Read()
	if err != nil {
		t.Fatal(err)
	}
	if got != "hello" {
		t.Fatalf("Read() = %q; want %q", got, "hello")
	}
}

