package main

import (
	"bytes"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"syscall"
	"testing"
	"time"
)

func statSysOrSkip(t *testing.T, path string) *syscall.Stat_t {
	t.Helper()
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	st, ok := info.Sys().(*syscall.Stat_t)
	if !ok {
		t.Skip("syscall.Stat_t not available on this platform")
	}
	return st
}

// TestSymlinkNotFollowed verifies that find-replace does not follow symbolic
// links out of the current working tree. Following symlinks would let any
// directory entry rewrite or rename files anywhere on the filesystem the
// running user can write — see issue #2.
func TestSymlinkNotFollowed(t *testing.T) {
	root := t.TempDir()

	// Build a "victim" tree outside of the area find-replace will be told to
	// process.
	victimDir := t.TempDir()
	victimFile := filepath.Join(victimDir, "secret.txt")
	const victimContents = "secret data"
	if err := os.WriteFile(victimFile, []byte(victimContents), 0o600); err != nil {
		t.Fatal(err)
	}

	// Inside the work tree, plant a symlink that points at the victim
	// directory.
	if err := os.Symlink(victimDir, filepath.Join(root, "escape")); err != nil {
		t.Fatal(err)
	}

	// Also include a regular file with content that should be rewritten so we
	// can confirm the run did something inside `root`.
	regularFile := filepath.Join(root, "regular.txt")
	if err := os.WriteFile(regularFile, []byte("secret in root"), 0o600); err != nil {
		t.Fatal(err)
	}

	fr := findReplace{find: "secret", replace: "PWNED"}
	fr.WalkDir(NewFile(root))

	// The victim file must be untouched.
	got, err := os.ReadFile(victimFile)
	if err != nil {
		t.Fatalf("victim file disappeared: %v", err)
	}
	if string(got) != victimContents {
		t.Errorf("victim file was rewritten through symlink: got %q want %q",
			string(got), victimContents)
	}

	// The regular file inside root should have been rewritten.
	got, err = os.ReadFile(regularFile)
	if err != nil {
		t.Fatalf("regular file disappeared: %v", err)
	}
	if !strings.Contains(string(got), "PWNED") {
		t.Errorf("regular file was not rewritten: got %q", string(got))
	}
}

// TestSymlinkNotRenamed verifies that a symlink whose name matches the find
// string is not itself renamed (which would still be safe, but we want to be
// explicit) AND that it is not chased to rename its target.
func TestSymlinkTargetNotRenamed(t *testing.T) {
	root := t.TempDir()

	victimDir := t.TempDir()
	victimFile := filepath.Join(victimDir, "alpha-target")
	if err := os.WriteFile(victimFile, []byte(""), 0o600); err != nil {
		t.Fatal(err)
	}

	// Symlink whose own name does NOT match the find string but whose target
	// directory contains a file whose name DOES match the find string.
	if err := os.Symlink(victimDir, filepath.Join(root, "via")); err != nil {
		t.Fatal(err)
	}

	fr := findReplace{find: "alpha", replace: "beta"}
	fr.WalkDir(NewFile(root))

	// File inside the symlinked target must not have been renamed.
	if _, err := os.Stat(victimFile); err != nil {
		t.Errorf("victim file %v was renamed/removed: %v", victimFile, err)
	}
	renamed := filepath.Join(victimDir, "beta-target")
	if _, err := os.Stat(renamed); err == nil {
		t.Errorf("symlink target was renamed to %v", renamed)
	}
}

// TestTempfileSymlinkAttack verifies that even if the same directory contains
// pre-planted files with names matching the temp-file pattern, the rewrite
// uses an O_EXCL-style creation that does not follow attacker-planted
// symlinks. See issue #3.
func TestTempfileSymlinkAttackRefuses(t *testing.T) {
	root := t.TempDir()
	target := filepath.Join(root, "target.txt")
	if err := os.WriteFile(target, []byte("alpha"), 0o600); err != nil {
		t.Fatal(err)
	}

	// Plant a victim file outside of the working tree.
	victimDir := t.TempDir()
	victim := filepath.Join(victimDir, "victim.txt")
	if err := os.WriteFile(victim, []byte("victim contents"), 0o600); err != nil {
		t.Fatal(err)
	}

	// Pre-plant 200 symlinks named like our temp-file pattern that point at
	// the victim. With O_EXCL the temp-file creation must reject every
	// pre-existing name and ultimately settle on a unique one (or error
	// out); without O_EXCL the open would follow the symlink and clobber
	// the victim.
	for i := 0; i < 200; i++ {
		linkName := filepath.Join(root, ".find-replace-attack-"+filepath.Base(target)+"-"+strings.Repeat("x", i+1))
		_ = os.Symlink(victim, linkName)
	}

	changed, err := rewriteFile(target, []byte("alpha"), []byte("BETA"), statOrFail(t, target))
	if err != nil {
		t.Fatal(err)
	}
	if !changed {
		t.Error("expected changed=true")
	}

	// Victim must be untouched.
	got, err := os.ReadFile(victim)
	if err != nil {
		t.Fatalf("victim missing: %v", err)
	}
	if !bytes.Equal(got, []byte("victim contents")) {
		t.Errorf("victim was rewritten via tempfile: %q", got)
	}

	// Target was rewritten correctly.
	got, _ = os.ReadFile(target)
	if string(got) != "BETA" {
		t.Errorf("target = %q, want BETA", got)
	}
}

// TestRenameRefusesOverwrite verifies that RenameFile does not silently
// overwrite an existing destination, even one created concurrently.
// See issue #4.
func TestRenameRefusesOverwrite(t *testing.T) {
	root := t.TempDir()

	src := filepath.Join(root, "alpha")
	if err := os.WriteFile(src, []byte("source"), 0o600); err != nil {
		t.Fatal(err)
	}
	// Pre-plant the destination so the rename must refuse.
	dst := filepath.Join(root, "BETA")
	if err := os.WriteFile(dst, []byte("destination"), 0o600); err != nil {
		t.Fatal(err)
	}

	fr := findReplace{find: "alpha", replace: "BETA"}
	fr.WalkDir(NewFile(root))

	// The destination must not have been overwritten.
	got, _ := os.ReadFile(dst)
	if string(got) != "destination" {
		t.Errorf("destination overwritten: %q", got)
	}
	// And we should have recorded an error.
	if fr.errors == 0 {
		t.Error("expected fr.errors > 0 when refusing rename")
	}
}

// TestRewritePreservesMtime verifies the rewrite preserves the original
// modification time. See issue #23.
func TestRewritePreservesMtime(t *testing.T) {
	d := t.TempDir()
	path := filepath.Join(d, "f.txt")
	if err := os.WriteFile(path, []byte("alpha"), 0o600); err != nil {
		t.Fatal(err)
	}
	stat0, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	// Roll mtime back so the test catches a real preservation rather than
	// a coincidence (writing now and rewriting now are likely close enough
	// to pass a coarser-grained check).
	past := stat0.ModTime().Add(-2 * time.Hour)
	if err := os.Chtimes(path, past, past); err != nil {
		t.Fatal(err)
	}

	changed, err := rewriteFile(path, []byte("alpha"), []byte("BETA"), statOrFail(t, path))
	if err != nil {
		t.Fatal(err)
	}
	if !changed {
		t.Fatal("expected change=true")
	}

	stat1, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if !stat1.ModTime().Equal(past) {
		t.Errorf("mtime not preserved: got %v want %v", stat1.ModTime(), past)
	}
}

// TestRewritePreservesOwner verifies that on Linux the rewrite preserves the
// original Uid and Gid. Only meaningful when running as root over files owned
// by other uids; otherwise we just check that the uid/gid are unchanged.
// See issue #17.
func TestRewritePreservesOwner(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("ownership semantics differ on windows")
	}
	d := t.TempDir()
	path := filepath.Join(d, "f.txt")
	if err := os.WriteFile(path, []byte("alpha"), 0o600); err != nil {
		t.Fatal(err)
	}

	stat0 := statSysOrSkip(t, path)

	if os.Getuid() == 0 {
		// Chown to a non-root uid we can use to make the test meaningful.
		// 'nobody' (65534) is universally available.
		if err := os.Chown(path, 65534, 65534); err != nil {
			t.Skipf("cannot chown to nobody: %v", err)
		}
		stat0 = statSysOrSkip(t, path)
	}

	changed, err := rewriteFile(path, []byte("alpha"), []byte("BETA"), statOrFail(t, path))
	if err != nil {
		t.Fatal(err)
	}
	if !changed {
		t.Fatal("expected change=true")
	}

	stat1 := statSysOrSkip(t, path)
	if stat1.Uid != stat0.Uid || stat1.Gid != stat0.Gid {
		t.Errorf("ownership not preserved: got uid=%d gid=%d want uid=%d gid=%d",
			stat1.Uid, stat1.Gid, stat0.Uid, stat0.Gid)
	}
}

// TestRefusesSetuidFile verifies that files with setuid/setgid bits are not
// rewritten. See issue #18.
func TestRefusesSetuidFile(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("setuid not applicable on windows")
	}
	d := t.TempDir()
	path := filepath.Join(d, "setuid")
	if err := os.WriteFile(path, []byte("alpha"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(path, 0o755|os.ModeSetuid); err != nil {
		t.Skipf("cannot set setuid bit: %v", err)
	}
	// Verify the bit actually stuck (some sandboxes silently strip setuid).
	if info, err := os.Stat(path); err != nil || info.Mode()&os.ModeSetuid == 0 {
		t.Skipf("setuid bit was stripped by the filesystem")
	}

	_, err := rewriteFile(path, []byte("alpha"), []byte("BETA"), statOrFail(t, path))
	if err == nil {
		t.Error("expected an error for setuid file")
	}
	got, _ := os.ReadFile(path)
	if string(got) != "alpha" {
		t.Errorf("setuid file was rewritten: %q", got)
	}
}
