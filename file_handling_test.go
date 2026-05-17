package main

import (
	"math/rand"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"
)

func TestNewFile(t *testing.T) {
}

func TestFileWriteDoesNotFollowPredictableTempSymlink(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("symlink creation requires elevated privileges on many Windows setups")
	}

	if os.Getenv("FIND_REPLACE_SYMLINK_HELPER") == "1" {
		runFileWriteSymlinkScenario(t)
		return
	}

	cmd := exec.Command(os.Args[0], "-test.run=^TestFileWriteDoesNotFollowPredictableTempSymlink$")
	cmd.Env = append(os.Environ(), "FIND_REPLACE_SYMLINK_HELPER=1", "GODEBUG=randseednop=0")
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("File.Write followed a predictable temp-file symlink:\n%s", output)
	}
}

func runFileWriteSymlinkScenario(t *testing.T) {
	dir := t.TempDir()
	targetPath := filepath.Join(dir, "target.txt")
	victimPath := filepath.Join(dir, "victim.txt")

	if err := os.WriteFile(targetPath, []byte("old contents"), 0o644); err != nil {
		t.Fatalf("os.WriteFile(%q): %v", targetPath, err)
	}
	if err := os.WriteFile(victimPath, []byte("victim contents"), 0o644); err != nil {
		t.Fatalf("os.WriteFile(%q): %v", victimPath, err)
	}

	rand.Seed(1)
	predictableTempPath := filepath.Join(dir, RandomString(20))
	rand.Seed(1)

	if err := os.Symlink(victimPath, predictableTempPath); err != nil {
		t.Fatalf("os.Symlink(%q, %q): %v", victimPath, predictableTempPath, err)
	}

	NewFile(targetPath).Write("new contents")

	victimContents, err := os.ReadFile(victimPath)
	if err != nil {
		t.Fatalf("os.ReadFile(%q): %v", victimPath, err)
	}
	if string(victimContents) != "victim contents" {
		t.Fatalf("victim file was modified via predictable temp symlink: %q", victimContents)
	}

	info, err := os.Lstat(targetPath)
	if err != nil {
		t.Fatalf("os.Lstat(%q): %v", targetPath, err)
	}
	if info.Mode()&os.ModeSymlink != 0 {
		t.Fatalf("target path was replaced by a symlink")
	}

	targetContents, err := os.ReadFile(targetPath)
	if err != nil {
		t.Fatalf("os.ReadFile(%q): %v", targetPath, err)
	}
	if string(targetContents) != "new contents" {
		t.Fatalf("target contents = %q; want %q", targetContents, "new contents")
	}
}
