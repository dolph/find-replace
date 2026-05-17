package main

import (
	"os"
	"path/filepath"
	"runtime"
	"syscall"
	"testing"
)

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

	NewFile(path).Write("new")

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
