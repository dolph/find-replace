package main

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// TestUnreadableSubdirSkipped verifies that an unreadable subdirectory does
// not abort the entire walk — see issue #6. The remaining tree must still be
// processed and the walker must record the error so the caller knows.
func TestUnreadableSubdirSkipped(t *testing.T) {
	if os.Getuid() == 0 {
		t.Skip("running as root; permission bits don't apply")
	}
	if runtime.GOOS == "windows" {
		t.Skip("permission semantics differ on windows")
	}

	root := t.TempDir()

	// Subdir with no read permission.
	denied := filepath.Join(root, "denied-alpha")
	if err := os.Mkdir(denied, 0o000); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(denied, 0o755) })

	// Sibling that should still be processed.
	openable := filepath.Join(root, "open-alpha.txt")
	if err := os.WriteFile(openable, []byte("alpha"), 0o600); err != nil {
		t.Fatal(err)
	}

	fr := findReplace{find: "alpha", replace: "BETA"}
	fr.WalkDir(NewFile(root))

	// The sibling file content was rewritten.
	got, _ := os.ReadFile(filepath.Join(root, "open-BETA.txt"))
	if string(got) != "BETA" {
		t.Errorf("sibling not rewritten; got %q", got)
	}

	// And the walker recorded at least one error.
	if fr.errors == 0 {
		t.Error("expected walker to record an error for the unreadable directory")
	}
}

// TestRejectsEmptyFind ensures the CLI fails fast when given an empty FIND
// argument — see issue #10. Runs as a subprocess against the built binary.
func TestRejectsEmptyFind(t *testing.T) {
	bin := buildBinary(t)
	cmd := exec.Command(bin, "", "anything")
	cmd.Dir = t.TempDir()
	out, err := cmd.CombinedOutput()
	if err == nil {
		t.Fatalf("expected non-zero exit, got success; output: %s", out)
	}
	if !bytes.Contains(out, []byte("FIND")) {
		t.Errorf("expected error message to mention FIND, got %q", out)
	}
}

// TestRejectsIdenticalFindReplace ensures the CLI fails fast when given
// FIND==REPLACE.
func TestRejectsIdenticalFindReplace(t *testing.T) {
	bin := buildBinary(t)
	cmd := exec.Command(bin, "alpha", "alpha")
	cmd.Dir = t.TempDir()
	out, err := cmd.CombinedOutput()
	if err == nil {
		t.Fatalf("expected non-zero exit, got success; output: %s", out)
	}
}

// TestExitCodeNonZeroOnError ensures that recording at least one error during
// a walk causes the binary to exit non-zero — see issue #11.
func TestExitCodeNonZeroOnError(t *testing.T) {
	bin := buildBinary(t)
	work := t.TempDir()

	// Create a rename collision so the run records an error.
	if err := os.WriteFile(filepath.Join(work, "alpha"), []byte(""), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(work, "BETA"), []byte(""), 0o600); err != nil {
		t.Fatal(err)
	}

	cmd := exec.Command(bin, "alpha", "BETA")
	cmd.Dir = work
	out, err := cmd.CombinedOutput()
	if err == nil {
		t.Fatalf("expected non-zero exit, got success; output: %s", out)
	}
}

// TestBigFileStreamsThroughBoundedMemory exercises the streaming rewriter on
// a file substantially larger than the buffer, ensuring correctness without
// loading the file into memory all at once. (See issue #8 — full validation
// of memory bounds requires runtime.MemStats and is brittle, so we settle
// for correctness on a multi-megabyte file plus the targeted boundary tests
// in TestStreamReplaceMatchAcrossBuffers.)
func TestBigFileStreamsThroughBoundedMemory(t *testing.T) {
	d := t.TempDir()
	path := filepath.Join(d, "big.txt")

	// 5 MB of text with the find string sprinkled throughout.
	chunk := strings.Repeat("alpha bravo charlie delta\n", 1024) // ~25 KB
	body := strings.Repeat(chunk, 200)                            // ~5 MB
	if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}

	changed, err := rewriteFile(path, []byte("bravo"), []byte("BB"), statOrFail(t, path))
	if err != nil {
		t.Fatal(err)
	}
	if !changed {
		t.Fatal("expected change=true")
	}

	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	want := strings.ReplaceAll(body, "bravo", "BB")
	if !bytes.Equal(got, []byte(want)) {
		t.Errorf("rewrite mismatch (got len=%d, want len=%d)", len(got), len(want))
	}
}

func buildBinary(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	bin := filepath.Join(dir, "find-replace")
	cmd := exec.Command("go", "build", "-o", bin, ".")
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("build failed: %v\n%s", err, out)
	}
	return bin
}

// TestGitAsFileNotRewritten verifies that a `.git` file (worktree/submodule
// linkage) is skipped, not rewritten. See issue #19.
func TestGitAsFileNotRewritten(t *testing.T) {
	root := t.TempDir()
	gitFile := filepath.Join(root, ".git")
	original := "gitdir: ../some-other-dir/.git/worktrees/me\n"
	if err := os.WriteFile(gitFile, []byte(original), 0o644); err != nil {
		t.Fatal(err)
	}

	fr := findReplace{find: "gitdir", replace: "WAS_REWRITTEN"}
	fr.WalkDir(NewFile(root))
	if fr.errors != 0 {
		t.Fatalf("walk reported %d errors", fr.errors)
	}

	got, err := os.ReadFile(gitFile)
	if err != nil {
		t.Fatalf(".git file disappeared: %v", err)
	}
	if string(got) != original {
		t.Errorf(".git file was rewritten: %q", got)
	}
}

// TestStaleTempFilesSkipped verifies that orphan .find-replace-* files from a
// crashed prior run are not picked up as targets. See issue #21.
func TestStaleTempFilesSkipped(t *testing.T) {
	root := t.TempDir()

	stale := filepath.Join(root, ".find-replace-orphan-alpha")
	if err := os.WriteFile(stale, []byte("alpha"), 0o600); err != nil {
		t.Fatal(err)
	}
	regular := filepath.Join(root, "alpha.txt")
	if err := os.WriteFile(regular, []byte("alpha"), 0o600); err != nil {
		t.Fatal(err)
	}

	fr := findReplace{find: "alpha", replace: "BETA"}
	fr.WalkDir(NewFile(root))

	// The stale temp file must remain — neither rewritten nor renamed.
	if got, _ := os.ReadFile(stale); string(got) != "alpha" {
		t.Errorf("stale temp file was rewritten: %q", got)
	}

	// The regular file was rewritten and renamed.
	got, _ := os.ReadFile(filepath.Join(root, "BETA.txt"))
	if string(got) != "BETA" {
		t.Errorf("regular file not rewritten/renamed: %q", got)
	}
}
