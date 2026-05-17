package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestWriteRemovesTempOnRenameFailure(t *testing.T) {
	dir := t.TempDir()
	dest := filepath.Join(dir, "dest")
	if err := os.WriteFile(dest, []byte("old"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(dest); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(dest, 0o755); err != nil {
		t.Fatal(err)
	}

	f := &File{Path: dest}
	err := f.writeContent([]byte("new"))
	if err == nil {
		t.Fatal("expected rename to fail when dest is a directory")
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range entries {
		if e.Name() != "dest" {
			t.Errorf("leftover temp file %q in %s", e.Name(), dir)
		}
	}
}

func TestWriteSucceedsForRegularFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "f.txt")
	if err := os.WriteFile(path, []byte("old"), 0o644); err != nil {
		t.Fatal(err)
	}

	f := NewFile(path)
	if err := f.writeContent([]byte("new")); err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "new" {
		t.Fatalf("file content = %q; want %q", got, "new")
	}
}
