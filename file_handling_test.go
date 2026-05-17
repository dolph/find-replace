package main

import (
	"os"
	"testing"
	"time"
)

func TestFileWritePreservesModTime(t *testing.T) {
	dir := t.TempDir()
	path := dir + "/mtime.txt"
	if err := os.WriteFile(path, []byte("before"), 0644); err != nil {
		t.Fatal(err)
	}
	want := time.Date(2019, 3, 14, 15, 9, 26, 0, time.UTC)
	if err := os.Chtimes(path, want, want); err != nil {
		t.Fatal(err)
	}

	f := NewFile(path)
	f.Write("after")

	got, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if !got.ModTime().Equal(want) {
		t.Fatalf("ModTime = %v, want %v", got.ModTime(), want)
	}
}
