package main

import (
	"bytes"
	"errors"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

/*
 * Testing utilities
 */

// newTestFile creates a file in the given directory path, with the given name
// and content. If a directory path is not provided, a temp directory is used.
// If a baseName is not provided, a random file name is generated. Returns the
// directory where the file was created, the file's directory entry, and the
// actual name of the file.
func newTestFile(tb testing.TB, path string, baseName string, content string) *File {
	tb.Helper()
	f, err := os.CreateTemp(path, baseName)
	if err != nil {
		tb.Fatalf("CreateTemp(%q, %q): %v", path, baseName, err)
	}
	if _, err := f.Write([]byte(content)); err != nil {
		defer os.Remove(f.Name())
		tb.Fatalf("write to %v: %v", f.Name(), err)
	}
	if err := f.Close(); err != nil {
		defer os.Remove(f.Name())
		tb.Fatalf("close %v: %v", f.Name(), err)
	}

	return newFileOrFatal(tb, f.Name())
}

// newTestDir creates a directory in the given directory path, with the given
// base name. If a directory path is not provided, a temp directory is used. If
// a baseName is not provided, a random file name is generated. Returns the
// directory where the file was created, the file's directory entry, and the
// actual name of the file.
func newTestDir(tb testing.TB, path string, baseName string) *File {
	tb.Helper()
	dirPath, err := os.MkdirTemp(path, baseName)
	if err != nil {
		tb.Fatalf("MkdirTemp(%q, %q): %v", path, baseName, err)
	}
	return newFileOrFatal(tb, dirPath)
}

// newFileOrFatal wraps NewFile for tests that should never see the
// (vanishingly rare) filepath.Abs error.
func newFileOrFatal(tb testing.TB, path string) *File {
	tb.Helper()
	f, err := NewFile(path)
	if err != nil {
		tb.Fatalf("NewFile(%q): %v", path, err)
	}
	return f
}

// readOrFatal returns the contents of f or fails the test.
func readOrFatal(tb testing.TB, f *File) string {
	tb.Helper()
	s, err := f.Read()
	if err != nil {
		tb.Fatalf("Read(%q): %v", f.Path, err)
	}
	return s
}

func expectedPathAfterRename(f *File, fr *findReplace) string {
	return filepath.Join(f.Dir(), strings.ReplaceAll(f.Base(), fr.find, fr.replace))
}

/*
 * Assertions
 */

// assertFileExists ensures that the given File exists
func assertFileExists(t *testing.T, f *File) {
	t.Helper()
	if _, err := os.Stat(f.Path); errors.Is(err, os.ErrNotExist) {
		t.Errorf("test file %v does not exist", f.Path)
	}
}

// assertFileNonexistent ensures that the File does not exist
func assertFileNonexistent(t *testing.T, f *File) {
	t.Helper()
	if _, err := os.Stat(f.Path); !errors.Is(err, os.ErrNotExist) {
		if err == nil {
			t.Errorf("test file %v exists", f.Path)
		} else {
			t.Errorf("test file %v exists (got %v)", f.Path, err)
		}
	}
}

func assertPathExistsAfterRename(t *testing.T, f *File, expectedPath string) *File {
	t.Helper()
	assertFileNonexistent(t, f)
	newFile := newFileOrFatal(t, expectedPath)
	assertFileExists(t, newFile)
	return newFile
}

/*
 * Tests
 */

// TestWalkDir is the most important test of the entire suite, because it
// exercises all the basic functionality of the app. It builds a directory tree
// of temporary files and directories, walks the entire tree, and ensures that
// all files and directories are appropriately renamed at at the end, and all
// files contain the correct contents.
func TestWalkDir(t *testing.T) {
	find := "wh"
	replace := "f"

	d := newTestDir(t, "", "*")
	defer os.Remove(d.Path)

	// d1: who/
	d1 := newTestDir(t, d.Path, "who")
	defer os.Remove(d1.Path)

	// d1d1: who/what/
	d1d1 := newTestDir(t, d1.Path, "what")
	defer os.Remove(d1d1.Path)

	// d1d1f1: who/what/when (contains "where")
	d1d1f1Contents := "where"
	d1d1f1 := newTestFile(t, d1d1.Path, "when", d1d1f1Contents)
	defer os.Remove(d1d1f1.Path)

	// d2: what/
	d2 := newTestDir(t, d.Path, "what")
	defer os.Remove(d2.Path)

	// d2d1: what/when/
	d2d1 := newTestDir(t, d2.Path, "when")
	defer os.Remove(d2d1.Path)

	// d2d1d1: what/when/where (directories with no files)
	d2d1d1 := newTestDir(t, d2d1.Path, "where")
	defer os.Remove(d2d1d1.Path)

	// d3: when/
	d3 := newTestDir(t, d.Path, "when")
	defer os.Remove(d3.Path)

	// d3f1: when/where (contains "why")
	d3f1Contents := "why"
	d3f1 := newTestFile(t, d3.Path, "where", d3f1Contents)
	defer os.Remove(d3f1.Path)

	// d4: where/ (empty directory in base dir)
	d4 := newTestDir(t, d.Path, "where")
	defer os.Remove(d4.Path)

	// f1: why (file in base dir contains "wh")
	f1Contents := "wh\nwh\nwh\n"
	f1 := newTestFile(t, d.Path, "why", f1Contents)
	defer os.Remove(f1.Path)

	fr := findReplace{find: find, replace: replace}
	fr.WalkDir(d)
	if err := fr.errs.err(); err != nil {
		t.Fatalf("WalkDir reported errors: %v", err)
	}

	// d1: who/ > fo/
	d1ExpectedPath := expectedPathAfterRename(d1, &fr)
	assertPathExistsAfterRename(t, d1, d1ExpectedPath)

	// d1d1: who/what/ > fo/foat/
	d1d1ExpectedPath := filepath.Join(d1ExpectedPath, strings.ReplaceAll(d1d1.Base(), fr.find, fr.replace))
	assertPathExistsAfterRename(t, d1d1, d1d1ExpectedPath)

	// d1d1f1: who/what/when > fo/fat/fen (contains "fere")
	d1d1f1ExpectedPath := filepath.Join(d1d1ExpectedPath, strings.ReplaceAll(d1d1f1.Base(), fr.find, fr.replace))
	assertPathExistsAfterRename(t, d1d1f1, d1d1f1ExpectedPath)
	assertNewContentsOfFile(t, d1d1f1ExpectedPath, d1d1f1Contents, find, replace, "fere")

	// d2: what/ > fat/
	d2ExpectedPath := expectedPathAfterRename(d2, &fr)
	assertPathExistsAfterRename(t, d2, d2ExpectedPath)

	// d2d1: what/when/
	d2d1ExpectedPath := filepath.Join(d2ExpectedPath, strings.ReplaceAll(d2d1.Base(), fr.find, fr.replace))
	assertPathExistsAfterRename(t, d2d1, d2d1ExpectedPath)

	// d2d1d1: what/when/where (directories with no files)
	d2d1d1ExpectedPath := filepath.Join(d2d1ExpectedPath, strings.ReplaceAll(d2d1d1.Base(), fr.find, fr.replace))
	assertPathExistsAfterRename(t, d2d1d1, d2d1d1ExpectedPath)

	// d3: when/
	d3ExpectedPath := expectedPathAfterRename(d3, &fr)
	assertPathExistsAfterRename(t, d3, d3ExpectedPath)

	// d3f1: when/where (contains "why")
	d3f1ExpectedPath := filepath.Join(d3ExpectedPath, strings.ReplaceAll(d3f1.Base(), fr.find, fr.replace))
	assertPathExistsAfterRename(t, d3f1, d3f1ExpectedPath)
	assertNewContentsOfFile(t, d3f1ExpectedPath, d3f1Contents, find, replace, "fy")

	// d4: where/ (empty directory in base dir)
	d4ExpectedPath := expectedPathAfterRename(d4, &fr)
	assertPathExistsAfterRename(t, d4, d4ExpectedPath)

	// f1: why (file in base dir contains "wh\nwh\nwh\n")
	f1ExpectedPath := expectedPathAfterRename(f1, &fr)
	assertPathExistsAfterRename(t, f1, f1ExpectedPath)
	assertNewContentsOfFile(t, f1ExpectedPath, f1Contents, find, replace, "f\nf\nf\n")
}

func TestHandleFileWithDir(t *testing.T) {
	initial := "alpha"
	find := "ph"
	replace := "f"

	f := newTestDir(t, "", initial)
	defer os.Remove(f.Path)
	expectedPath := filepath.Join(f.Dir(), strings.ReplaceAll(f.Base(), find, replace))
	defer os.Remove(expectedPath)
	fr := findReplace{find: find, replace: replace}

	assertFileExists(t, f)
	if err := fr.HandleFile(f); err != nil {
		t.Fatalf("HandleFile(%q): %v", f.Path, err)
	}
	assertPathExistsAfterRename(t, f, expectedPath)
}

func TestHandleFileWithIgnoredDir(t *testing.T) {
	initial := ".git"
	find := "git"
	replace := "got"

	dirPath := filepath.Join(t.TempDir(), initial)
	if err := os.Mkdir(dirPath, 0700); err != nil {
		t.Fatalf("Mkdir(%q): %v", dirPath, err)
	}
	f := newFileOrFatal(t, dirPath)
	// Just in case it's unexpectedly renamed, let's make sure we cleanup the
	// anticipated name.
	unexpectedName := strings.ReplaceAll(f.Base(), find, replace)
	unexpectedPath := filepath.Join(f.Dir(), unexpectedName)
	defer os.Remove(unexpectedPath)
	fr := findReplace{find: find, replace: replace}

	assertFileExists(t, f)
	if err := fr.HandleFile(f); err != nil {
		t.Fatalf("HandleFile(%q): %v", f.Path, err)
	}
	assertFileExists(t, f)
}

func TestHandleFileWithFile(t *testing.T) {
	initial := "alpha"
	find := "ph"
	replace := "f"
	want := "alfa"

	f := newTestFile(t, "", initial, initial)
	defer os.Remove(f.Path)
	expectedName := strings.ReplaceAll(f.Base(), find, replace)
	expectedPath := filepath.Join(f.Dir(), expectedName)
	defer os.Remove(expectedPath)
	fr := findReplace{find: find, replace: replace}

	assertFileExists(t, f)
	if err := fr.HandleFile(f); err != nil {
		t.Fatalf("HandleFile(%q): %v", f.Path, err)
	}
	assertPathExistsAfterRename(t, f, expectedPath)

	got := readOrFatal(t, newFileOrFatal(t, expectedPath))
	if got != want {
		t.Errorf("replace %v with %v in %v, but got %v; want %v", find, replace, initial, got, want)
	}
}

func TestRenameFile(t *testing.T) {
	initial := "alpha"
	find := "ph"
	replace := "f"

	f := newTestFile(t, "", initial, "")
	defer os.Remove(f.Path)
	expectedName := strings.ReplaceAll(f.Base(), find, replace)
	expectedPath := filepath.Join(f.Dir(), expectedName)
	defer os.Remove(expectedPath)
	fr := findReplace{find: find, replace: replace}

	assertFileExists(t, f)
	if err := fr.RenameFile(f); err != nil {
		t.Fatalf("RenameFile(%q): %v", f.Path, err)
	}
	assertPathExistsAfterRename(t, f, expectedPath)
}

// assertNewContentsOfFile ensures that the contents of the file at the given
// path exactly match the desired string.
func assertNewContentsOfFile(t *testing.T, path string, initial string, find string, replace string, want string) {
	t.Helper()
	got := readOrFatal(t, newFileOrFatal(t, path))
	if got != want {
		t.Errorf("replace %v with %v in %v, but got %v; want %v", find, replace, initial, got, want)
	}
}

func TestReplaceContents(t *testing.T) {
	initial := "alpha"
	find := "ph"
	replace := "f"
	want := "alfa"

	f := newTestFile(t, "", "*", initial)
	defer os.Remove(f.Path)
	fr := findReplace{find: find, replace: replace}
	if err := fr.ReplaceContents(f); err != nil {
		t.Fatalf("ReplaceContents(%q): %v", f.Path, err)
	}
	assertNewContentsOfFile(t, f.Path, initial, find, replace, want)
}

func TestReplaceContentsEntireFile(t *testing.T) {
	initial := "alpha"
	find := "alpha"
	replace := "beta"
	want := "beta"

	f := newTestFile(t, "", "*", initial)
	defer os.Remove(f.Path)
	fr := findReplace{find: find, replace: replace}
	if err := fr.ReplaceContents(f); err != nil {
		t.Fatalf("ReplaceContents(%q): %v", f.Path, err)
	}
	assertNewContentsOfFile(t, f.Path, initial, find, replace, want)
}

func TestReplaceContentsMultipleMatchesSingleLine(t *testing.T) {
	initial := "alphaalpha"
	find := "ph"
	replace := "f"
	want := "alfaalfa"

	f := newTestFile(t, "", "*", initial)
	defer os.Remove(f.Path)
	fr := findReplace{find: find, replace: replace}
	if err := fr.ReplaceContents(f); err != nil {
		t.Fatalf("ReplaceContents(%q): %v", f.Path, err)
	}
	assertNewContentsOfFile(t, f.Path, initial, find, replace, want)
}

func TestReplaceContentsMultipleMatchesMultipleLines(t *testing.T) {
	initial := "alpha\nalpha"
	find := "ph"
	replace := "f"
	want := "alfa\nalfa"

	f := newTestFile(t, "", "*", initial)
	defer os.Remove(f.Path)
	fr := findReplace{find: find, replace: replace}
	if err := fr.ReplaceContents(f); err != nil {
		t.Fatalf("ReplaceContents(%q): %v", f.Path, err)
	}
	assertNewContentsOfFile(t, f.Path, initial, find, replace, want)
}

func TestReplaceContentsNoMatches(t *testing.T) {
	initial := "alpha"
	find := "abc"
	replace := "xyz"
	want := "alpha"

	f := newTestFile(t, "", "*", initial)
	defer os.Remove(f.Path)
	fr := findReplace{find: find, replace: replace}
	if err := fr.ReplaceContents(f); err != nil {
		t.Fatalf("ReplaceContents(%q): %v", f.Path, err)
	}
	assertNewContentsOfFile(t, f.Path, initial, find, replace, want)
}

// TestWalkDir_PermissionDeniedSubdirContinues ensures that an unreadable
// subdirectory does not abort the walk. The sibling subtree must still be
// rewritten, and the walker must record an error referencing the failing
// subdirectory.
func TestWalkDir_PermissionDeniedSubdirContinues(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("permission semantics differ on Windows")
	}
	if os.Geteuid() == 0 {
		t.Skip("test requires non-root: chmod 0 directories are still readable as root")
	}

	root := t.TempDir()

	// Build root/a/inside.txt (unreadable parent) and root/b/inside.txt
	// (normal). After WalkDir we expect b's file to be rewritten and a's
	// directory to surface an error.
	denied := filepath.Join(root, "a")
	if err := os.Mkdir(denied, 0700); err != nil {
		t.Fatalf("Mkdir(%q): %v", denied, err)
	}
	deniedChild := filepath.Join(denied, "inside.txt")
	if err := os.WriteFile(deniedChild, []byte("alpha"), 0600); err != nil {
		t.Fatalf("WriteFile(%q): %v", deniedChild, err)
	}

	siblingDir := filepath.Join(root, "b")
	if err := os.Mkdir(siblingDir, 0700); err != nil {
		t.Fatalf("Mkdir(%q): %v", siblingDir, err)
	}
	siblingFile := filepath.Join(siblingDir, "inside.txt")
	if err := os.WriteFile(siblingFile, []byte("alpha"), 0600); err != nil {
		t.Fatalf("WriteFile(%q): %v", siblingFile, err)
	}

	// Remove read+exec from the denied directory only. Restore the bit at
	// cleanup so t.TempDir's RemoveAll can succeed.
	if err := os.Chmod(denied, 0); err != nil {
		t.Fatalf("Chmod(%q, 0): %v", denied, err)
	}
	t.Cleanup(func() { _ = os.Chmod(denied, 0700) })

	rootFile := newFileOrFatal(t, root)
	fr := findReplace{find: "alpha", replace: "beta"}
	fr.WalkDir(rootFile)

	// The sibling file should have been rewritten despite the denied subtree.
	got, err := os.ReadFile(siblingFile)
	if err != nil {
		t.Fatalf("ReadFile(%q): %v", siblingFile, err)
	}
	if string(got) != "beta" {
		t.Errorf("sibling file contents = %q; want %q (denied sibling aborted the walk)", string(got), "beta")
	}

	// The walker must have recorded an error referencing the denied subtree.
	err = fr.errs.err()
	if err == nil {
		t.Fatalf("WalkDir succeeded; want error mentioning %q", denied)
	}
	if !strings.Contains(err.Error(), denied) {
		t.Errorf("WalkDir error = %v; want one mentioning %q", err, denied)
	}
	// errors.Is should walk the joined chain and find the permission error.
	if !errors.Is(err, fs.ErrPermission) {
		t.Errorf("WalkDir error = %v; want errors.Is(_, fs.ErrPermission) == true", err)
	}
}

// TestRenameFile_ReturnsErrorOnExistingDestination ensures a clobbering
// rename is refused (returning an error) rather than crashing the process.
func TestRenameFile_ReturnsErrorOnExistingDestination(t *testing.T) {
	tmp := t.TempDir()
	src := filepath.Join(tmp, "alpha")
	if err := os.WriteFile(src, nil, 0600); err != nil {
		t.Fatalf("WriteFile(%q): %v", src, err)
	}
	dst := filepath.Join(tmp, "beta")
	if err := os.WriteFile(dst, nil, 0600); err != nil {
		t.Fatalf("WriteFile(%q): %v", dst, err)
	}

	f := newFileOrFatal(t, src)
	fr := findReplace{find: "alpha", replace: "beta"}
	err := fr.RenameFile(f)
	if err == nil {
		t.Fatalf("RenameFile(%q): err = nil; want an error referencing the occupied destination", src)
	}
	if !strings.Contains(err.Error(), "beta") {
		t.Errorf("RenameFile error = %v; want one mentioning %q", err, "beta")
	}
	// The source must still be present — RenameFile must not have clobbered
	// the destination either.
	if _, statErr := os.Stat(src); statErr != nil {
		t.Errorf("Stat(%q) after refused rename: %v", src, statErr)
	}
	if _, statErr := os.Stat(dst); statErr != nil {
		t.Errorf("Stat(%q) after refused rename: %v", dst, statErr)
	}
}

// TestWalkDir_BadRenameTargetDoesNotAbortSiblings sets up two sibling files
// whose post-rename names would collide with already-existing files. The
// walker must rename what it can, record errors for what it cannot, and not
// abort the rest of the tree.
func TestWalkDir_BadRenameTargetDoesNotAbortSiblings(t *testing.T) {
	root := t.TempDir()

	// Files that will be renamed alpha -> beta. The "occupied" path already
	// has a beta target so its rename must fail. The "free" path has a
	// distinct prefix and should succeed.
	occupied := filepath.Join(root, "occupied-alpha")
	if err := os.WriteFile(occupied, nil, 0600); err != nil {
		t.Fatalf("WriteFile(%q): %v", occupied, err)
	}
	occupiedTarget := filepath.Join(root, "occupied-beta")
	if err := os.WriteFile(occupiedTarget, nil, 0600); err != nil {
		t.Fatalf("WriteFile(%q): %v", occupiedTarget, err)
	}

	free := filepath.Join(root, "free-alpha")
	if err := os.WriteFile(free, nil, 0600); err != nil {
		t.Fatalf("WriteFile(%q): %v", free, err)
	}

	rootFile := newFileOrFatal(t, root)
	fr := findReplace{find: "alpha", replace: "beta"}
	fr.WalkDir(rootFile)

	// The free file should have been renamed.
	freeRenamed := filepath.Join(root, "free-beta")
	if _, err := os.Stat(freeRenamed); err != nil {
		t.Errorf("Stat(%q) after walk: %v (free-alpha should have been renamed despite occupied-alpha's failure)", freeRenamed, err)
	}

	// The walker must have recorded an error referencing the occupied target.
	err := fr.errs.err()
	if err == nil {
		t.Fatalf("WalkDir succeeded; want a 'refusing to rename' error for occupied-alpha")
	}
	if !strings.Contains(err.Error(), "occupied-beta") {
		t.Errorf("WalkDir error = %v; want one mentioning %q", err, "occupied-beta")
	}
}

// TestWriteCleansUpTempFileOnRenameFailure ensures that File.Write does not
// leak a temp file when the rename step fails. It forces the rename to fail
// (after the temp file has been created) by making the destination a
// non-empty directory; os.Rename of a regular file onto a non-empty
// directory returns ENOTEMPTY ("file exists") on Linux regardless of the
// running user, so this exercises the deferred-cleanup path under both root
// and non-root.
func TestWriteCleansUpTempFileOnRenameFailure(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("rename-over-directory semantics differ on Windows")
	}

	dir := t.TempDir()
	target := filepath.Join(dir, "target")

	// Create the target as a non-empty directory. Write will succeed in
	// creating its tempfile next to it, then fail the rename step.
	if err := os.Mkdir(target, 0700); err != nil {
		t.Fatalf("Mkdir(%q): %v", target, err)
	}
	if err := os.WriteFile(filepath.Join(target, "sentinel"), nil, 0600); err != nil {
		t.Fatalf("WriteFile sentinel: %v", err)
	}

	// Snapshot the directory contents before the Write so we can confirm no
	// stray files survive afterwards.
	beforeEntries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("ReadDir(%q): %v", dir, err)
	}
	before := make(map[string]struct{}, len(beforeEntries))
	for _, e := range beforeEntries {
		before[e.Name()] = struct{}{}
	}

	// We need f.Mode() to succeed, so prime the cached info.
	f := newFileOrFatal(t, target)
	if _, err := f.Info(); err != nil {
		t.Fatalf("Info(%q): %v", target, err)
	}

	if err := f.Write("beta"); err == nil {
		t.Fatalf("Write succeeded over a non-empty directory; expected an error")
	}

	// Confirm no new entries (other than the existing target directory)
	// linger in the parent.
	afterEntries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("ReadDir(%q): %v", dir, err)
	}
	for _, e := range afterEntries {
		if _, ok := before[e.Name()]; ok {
			continue
		}
		t.Errorf("leftover entry %q in %q after Write failure (tempfile was not cleaned up)", e.Name(), dir)
	}
}

// TestRun_ExitsZeroOnSuccess confirms run() returns 0 for a clean walk.
func TestRun_ExitsZeroOnSuccess(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "alpha.txt"), []byte("alpha"), 0600); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	withWorkingDir(t, dir)

	var stderr bytes.Buffer
	got := run([]string{"find-replace", "alpha", "beta"}, &stderr)
	if got != 0 {
		t.Errorf("run = %d; want 0 (stderr: %q)", got, stderr.String())
	}
}

// TestRun_ExitsNonZeroOnTraversalError confirms run() returns a non-zero
// exit code when any file failed to be processed. We force a failure by
// putting a file whose rename target is occupied.
func TestRun_ExitsNonZeroOnTraversalError(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "occupied-alpha")
	if err := os.WriteFile(src, nil, 0600); err != nil {
		t.Fatalf("WriteFile(%q): %v", src, err)
	}
	dst := filepath.Join(dir, "occupied-beta")
	if err := os.WriteFile(dst, nil, 0600); err != nil {
		t.Fatalf("WriteFile(%q): %v", dst, err)
	}

	withWorkingDir(t, dir)

	var stderr bytes.Buffer
	got := run([]string{"find-replace", "alpha", "beta"}, &stderr)
	if got == 0 {
		t.Errorf("run = 0; want non-zero (stderr: %q)", stderr.String())
	}
}

// TestRun_BadArgCountPrintsUsage confirms the usage message goes to stderr
// and the exit code is non-zero.
func TestRun_BadArgCountPrintsUsage(t *testing.T) {
	var stderr bytes.Buffer
	got := run([]string{"find-replace"}, &stderr)
	if got == 0 {
		t.Errorf("run = 0; want non-zero")
	}
	if !strings.Contains(stderr.String(), "Usage: find-replace") {
		t.Errorf("stderr = %q; want it to contain a usage line", stderr.String())
	}
}

// withWorkingDir chdirs to dir for the duration of the test and restores the
// previous working directory at cleanup.
func withWorkingDir(t *testing.T, dir string) {
	t.Helper()
	prev, err := os.Getwd()
	if err != nil {
		t.Fatalf("Getwd: %v", err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatalf("Chdir(%q): %v", dir, err)
	}
	t.Cleanup(func() { _ = os.Chdir(prev) })
}

func CloneRepoToTestDir(b *testing.B, repoUrl string) *File {
	b.Helper()
	d := newTestDir(b, "", "*")
	defer os.Remove(d.Path)

	cmd := exec.Command("git", "clone", "--depth=1", "--single-branch", repoUrl, ".")
	cmd.Dir = d.Path
	out, err := cmd.CombinedOutput()
	if err != nil {
		b.Errorf("failed to clone repo: %s", out)
	}

	return d
}

func BenchmarkNova(b *testing.B) {
	for n := 0; n < b.N; n++ {
		b.StopTimer()
		d := CloneRepoToTestDir(b, "git@github.com:openstack/nova.git")
		fr := findReplace{find: RandomString(2), replace: RandomString(2)}
		b.StartTimer()
		fr.WalkDir(d)
	}
}

// TestWalkDir_SymlinkToFileOutsideCWDNotFollowed ensures the walker does not
// rewrite a file reached via a symlink whose target lives outside the search
// root. Following the symlink would violate find-replace's documented contract
// ("Searches are performed recursively from the current working directory")
// and is a known privilege-escalation primitive (see issue #2).
func TestWalkDir_SymlinkToFileOutsideCWDNotFollowed(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("creating symlinks on Windows typically requires admin privileges")
	}

	// outside holds the file that must NOT be touched by the walk.
	outside := t.TempDir()
	outsideFile := filepath.Join(outside, "outside.txt")
	const outsideContent = "alpha-outside"
	if err := os.WriteFile(outsideFile, []byte(outsideContent), 0600); err != nil {
		t.Fatalf("WriteFile(%q): %v", outsideFile, err)
	}

	// root is the search root. It contains a real file and a symlink that
	// escapes to outsideFile.
	root := t.TempDir()
	insideFile := filepath.Join(root, "inside.txt")
	if err := os.WriteFile(insideFile, []byte("alpha-inside"), 0600); err != nil {
		t.Fatalf("WriteFile(%q): %v", insideFile, err)
	}
	escape := filepath.Join(root, "escape.txt")
	if err := os.Symlink(outsideFile, escape); err != nil {
		t.Fatalf("Symlink(%q, %q): %v", outsideFile, escape, err)
	}

	rootFile := newFileOrFatal(t, root)
	fr := findReplace{find: "alpha", replace: "beta"}
	fr.WalkDir(rootFile)

	if err := fr.errs.err(); err != nil {
		t.Errorf("WalkDir reported unexpected error(s): %v", err)
	}

	// The real file inside the root must have been rewritten.
	got, err := os.ReadFile(insideFile)
	if err != nil {
		t.Fatalf("ReadFile(%q): %v", insideFile, err)
	}
	if string(got) != "beta-inside" {
		t.Errorf("inside.txt = %q; want %q", string(got), "beta-inside")
	}

	// The symlink target outside the root must NOT have been rewritten.
	got, err = os.ReadFile(outsideFile)
	if err != nil {
		t.Fatalf("ReadFile(%q): %v", outsideFile, err)
	}
	if string(got) != outsideContent {
		t.Errorf("outside.txt = %q; want %q (symlink was followed)", string(got), outsideContent)
	}

	// The symlink itself should still be present as a symlink (the walker
	// should have skipped it, not replaced it with a regular file or renamed
	// it).
	info, err := os.Lstat(escape)
	if err != nil {
		t.Fatalf("Lstat(%q) after walk: %v (symlink was removed/renamed)", escape, err)
	}
	if info.Mode()&fs.ModeSymlink == 0 {
		t.Errorf("Lstat(%q).Mode() = %v; want a symlink (was replaced with a regular file)", escape, info.Mode())
	}
}

// TestWalkDir_SymlinkToDirOutsideCWDNotTraversed ensures the walker does not
// recurse through a symlink whose target is a directory outside the search
// root. This is the directory variant of the issue #2 reproducer.
func TestWalkDir_SymlinkToDirOutsideCWDNotTraversed(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("creating symlinks on Windows typically requires admin privileges")
	}

	// outside is a directory tree we must not touch.
	outside := t.TempDir()
	outsideFile := filepath.Join(outside, "victim.txt")
	const outsideContent = "alpha-outside"
	if err := os.WriteFile(outsideFile, []byte(outsideContent), 0600); err != nil {
		t.Fatalf("WriteFile(%q): %v", outsideFile, err)
	}

	root := t.TempDir()
	escape := filepath.Join(root, "escape")
	if err := os.Symlink(outside, escape); err != nil {
		t.Fatalf("Symlink(%q, %q): %v", outside, escape, err)
	}

	rootFile := newFileOrFatal(t, root)
	fr := findReplace{find: "alpha", replace: "beta"}
	fr.WalkDir(rootFile)

	if err := fr.errs.err(); err != nil {
		t.Errorf("WalkDir reported unexpected error(s): %v", err)
	}

	// The outside file must NOT have been rewritten.
	got, err := os.ReadFile(outsideFile)
	if err != nil {
		t.Fatalf("ReadFile(%q): %v", outsideFile, err)
	}
	if string(got) != outsideContent {
		t.Errorf("victim.txt = %q; want %q (symlinked directory was traversed)", string(got), outsideContent)
	}

	// The outside file must NOT have been renamed.
	renamed := filepath.Join(outside, "beta-victim.txt")
	if _, err := os.Lstat(renamed); err == nil {
		t.Errorf("Lstat(%q) succeeded; file inside symlinked dir was renamed", renamed)
	}

	// The symlink itself should still be present.
	if _, err := os.Lstat(escape); err != nil {
		t.Errorf("Lstat(%q) after walk: %v (symlink was removed/renamed)", escape, err)
	}
}

// TestWalkDir_SymlinkToFileInsideCWDNotFollowed verifies that even when a
// symlink's target is inside the search root, the symlink itself is still
// skipped. The target is still picked up via the normal walker pass over its
// real path, so it is rewritten exactly once — never via the symlink.
func TestWalkDir_SymlinkToFileInsideCWDNotFollowed(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("creating symlinks on Windows typically requires admin privileges")
	}

	root := t.TempDir()
	real := filepath.Join(root, "real.txt")
	if err := os.WriteFile(real, []byte("alpha"), 0600); err != nil {
		t.Fatalf("WriteFile(%q): %v", real, err)
	}
	link := filepath.Join(root, "link.txt")
	if err := os.Symlink(real, link); err != nil {
		t.Fatalf("Symlink(%q, %q): %v", real, link, err)
	}

	rootFile := newFileOrFatal(t, root)
	fr := findReplace{find: "alpha", replace: "beta"}
	fr.WalkDir(rootFile)

	if err := fr.errs.err(); err != nil {
		t.Errorf("WalkDir reported unexpected error(s): %v", err)
	}

	// The real file should be rewritten via the normal pass — exactly once
	// (not twice — that is, the symlink was not also followed and rewritten).
	got, err := os.ReadFile(real)
	if err != nil {
		t.Fatalf("ReadFile(%q): %v", real, err)
	}
	if string(got) != "beta" {
		t.Errorf("real.txt = %q; want %q", string(got), "beta")
	}

	// The symlink should still be a symlink pointing to the (now-rewritten)
	// real file. The walker must not have renamed or removed it.
	info, err := os.Lstat(link)
	if err != nil {
		t.Fatalf("Lstat(%q) after walk: %v (symlink was removed/renamed)", link, err)
	}
	if info.Mode()&fs.ModeSymlink == 0 {
		t.Errorf("Lstat(%q).Mode() = %v; want a symlink (was replaced with a regular file)", link, info.Mode())
	}
}

// TestWalkDir_BrokenSymlinkSkippedWithoutError ensures a dangling symlink is
// silently skipped, not reported as an error and not "rewritten" or "renamed".
func TestWalkDir_BrokenSymlinkSkippedWithoutError(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("creating symlinks on Windows typically requires admin privileges")
	}

	root := t.TempDir()
	broken := filepath.Join(root, "alpha-broken")
	target := filepath.Join(root, "does-not-exist")
	if err := os.Symlink(target, broken); err != nil {
		t.Fatalf("Symlink(%q, %q): %v", target, broken, err)
	}

	rootFile := newFileOrFatal(t, root)
	fr := findReplace{find: "alpha", replace: "beta"}
	fr.WalkDir(rootFile)

	if err := fr.errs.err(); err != nil {
		t.Errorf("WalkDir reported unexpected error(s) for a broken symlink: %v", err)
	}

	// The symlink must still be present under its original name (it was not
	// renamed alpha-broken -> beta-broken, nor was it followed).
	info, err := os.Lstat(broken)
	if err != nil {
		t.Fatalf("Lstat(%q) after walk: %v (broken symlink was removed/renamed)", broken, err)
	}
	if info.Mode()&fs.ModeSymlink == 0 {
		t.Errorf("Lstat(%q).Mode() = %v; want a symlink (was replaced with a regular file)", broken, info.Mode())
	}

	// The renamed name should not have been created either.
	renamed := filepath.Join(root, "beta-broken")
	if _, err := os.Lstat(renamed); err == nil {
		t.Errorf("Lstat(%q) succeeded; broken symlink was renamed", renamed)
	}
}

// TestWalkDir_RegularFileStillProcessed is a sanity check that the
// symlink-skipping fix does not break the common case for regular files.
func TestWalkDir_RegularFileStillProcessed(t *testing.T) {
	root := t.TempDir()
	src := filepath.Join(root, "alpha.txt")
	if err := os.WriteFile(src, []byte("alpha"), 0600); err != nil {
		t.Fatalf("WriteFile(%q): %v", src, err)
	}

	rootFile := newFileOrFatal(t, root)
	fr := findReplace{find: "alpha", replace: "beta"}
	fr.WalkDir(rootFile)

	if err := fr.errs.err(); err != nil {
		t.Errorf("WalkDir reported unexpected error(s): %v", err)
	}

	renamed := filepath.Join(root, "beta.txt")
	got, err := os.ReadFile(renamed)
	if err != nil {
		t.Fatalf("ReadFile(%q): %v", renamed, err)
	}
	if string(got) != "beta" {
		t.Errorf("beta.txt = %q; want %q", string(got), "beta")
	}
}

// TestWalkDir_RegularDirStillTraversed is the directory variant of the sanity
// check.
func TestWalkDir_RegularDirStillTraversed(t *testing.T) {
	root := t.TempDir()
	sub := filepath.Join(root, "alpha-dir")
	if err := os.Mkdir(sub, 0700); err != nil {
		t.Fatalf("Mkdir(%q): %v", sub, err)
	}
	child := filepath.Join(sub, "alpha.txt")
	if err := os.WriteFile(child, []byte("alpha"), 0600); err != nil {
		t.Fatalf("WriteFile(%q): %v", child, err)
	}

	rootFile := newFileOrFatal(t, root)
	fr := findReplace{find: "alpha", replace: "beta"}
	fr.WalkDir(rootFile)

	if err := fr.errs.err(); err != nil {
		t.Errorf("WalkDir reported unexpected error(s): %v", err)
	}

	renamedDir := filepath.Join(root, "beta-dir")
	renamedChild := filepath.Join(renamedDir, "beta.txt")
	got, err := os.ReadFile(renamedChild)
	if err != nil {
		t.Fatalf("ReadFile(%q): %v", renamedChild, err)
	}
	if string(got) != "beta" {
		t.Errorf("beta-dir/beta.txt = %q; want %q", string(got), "beta")
	}
}
