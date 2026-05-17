package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestNewFile(t *testing.T) {
	t.Run("absolute path", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, "foo.txt")
		if err := os.WriteFile(path, []byte("x"), 0o644); err != nil {
			t.Fatal(err)
		}

		want, err := filepath.Abs(path)
		if err != nil {
			t.Fatal(err)
		}

		f := NewFile(path)
		if f.Path != want {
			t.Errorf("NewFile(%q).Path = %q; want %q", path, f.Path, want)
		}
	})

	t.Run("relative path", func(t *testing.T) {
		dir := t.TempDir()
		orig, err := os.Getwd()
		if err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() { _ = os.Chdir(orig) })
		if err := os.Chdir(dir); err != nil {
			t.Fatal(err)
		}

		rel := "bar.txt"
		if err := os.WriteFile(rel, []byte("y"), 0o644); err != nil {
			t.Fatal(err)
		}

		want, err := filepath.Abs(rel)
		if err != nil {
			t.Fatal(err)
		}

		f := NewFile(rel)
		if f.Path != want {
			t.Errorf("NewFile(%q).Path = %q; want %q", rel, f.Path, want)
		}
	})
}

func TestReadSkipsBinaryWithNUL(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "mixed.txt")
	content := []byte("text prefix\x00binary suffix")
	if err := os.WriteFile(path, content, 0o644); err != nil {
		t.Fatal(err)
	}

	got := NewFile(path).Read()
	if got != "" {
		t.Fatalf("Read() = %q; want empty for NUL-containing file", got)
	}
}
