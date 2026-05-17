package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestWalkDirContinuesOnUnreadableDirectory(t *testing.T) {
	root := newTestDir("", "*")
	defer os.Remove(root.Path)

	readable := newTestFile(root.Path, "ok.txt", "hello")
	defer os.Remove(readable.Path)

	noRead := filepath.Join(root.Path, "private")
	if err := os.Mkdir(noRead, 0o700); err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(noRead)
	if err := os.Chmod(noRead, 0o000); err != nil {
		t.Skip("cannot chmod directory for permission test:", err)
	}
	defer func() { _ = os.Chmod(noRead, 0o700) }()

	fr := findReplace{find: "hello", replace: "hi"}
	fr.WalkDir(fr.newFile(root.Path))

	if !fr.hadErrors.Load() {
		t.Fatal("expected hadErrors after unreadable directory")
	}

	got, ok := NewFile(readable.Path).Read()
	if !ok || got != "hi" {
		t.Fatalf("readable file not rewritten: got %q ok=%v", got, ok)
	}
}
