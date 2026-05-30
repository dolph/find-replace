package main

import (
	"os"
	"path/filepath"
	"runtime"
	"syscall"
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

func TestWritePreservesOwnership(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("ownership preservation not implemented on Windows")
	}

	dir := t.TempDir()
	path := filepath.Join(dir, "f.txt")
	if err := os.WriteFile(path, []byte("old"), 0o644); err != nil {
		t.Fatal(err)
	}

	before, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	beforeStat, ok := before.Sys().(*syscall.Stat_t)
	if !ok {
		t.Skip("syscall.Stat_t not available")
	}

	f, err := NewFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := f.Write("new"); err != nil {
		t.Fatal(err)
	}

	after, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	afterStat, ok := after.Sys().(*syscall.Stat_t)
	if !ok {
		t.Fatal("expected syscall.Stat_t after rewrite")
	}
	if afterStat.Uid != beforeStat.Uid || afterStat.Gid != beforeStat.Gid {
		t.Fatalf("ownership changed: uid %d->%d gid %d->%d",
			beforeStat.Uid, afterStat.Uid, beforeStat.Gid, afterStat.Gid)
	}
	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "new" {
		t.Fatalf("content = %q; want %q", got, "new")
	}
}

