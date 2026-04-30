package main

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

/*
 * Testing utilities
 */

// newTestFile creates a file in the given directory with the given name and
// content. If baseName is empty a unique random name is used.
func newTestFile(t testing.TB, dir, baseName, content string) *File {
	t.Helper()
	pattern := baseName
	if pattern == "" {
		pattern = "*"
	}
	f, err := os.CreateTemp(dir, pattern)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := f.Write([]byte(content)); err != nil {
		_ = f.Close()
		t.Fatal(err)
	}
	if err := f.Close(); err != nil {
		t.Fatal(err)
	}
	return NewFile(f.Name())
}

// newTestDir creates a directory in the given directory path with the given
// base name (or a random name if empty).
func newTestDir(t testing.TB, dir, baseName string) *File {
	t.Helper()
	pattern := baseName
	if pattern == "" {
		pattern = "*"
	}
	dirPath, err := os.MkdirTemp(dir, pattern)
	if err != nil {
		t.Fatal(err)
	}
	return NewFile(dirPath)
}

func expectedPathAfterRename(f *File, fr *findReplace) string {
	return filepath.Join(f.Dir(), strings.Replace(f.Base(), fr.find, fr.replace, -1))
}

/*
 * Assertions
 */

func assertFileExists(t *testing.T, f *File) {
	t.Helper()
	if _, err := os.Stat(f.Path); errors.Is(err, os.ErrNotExist) {
		t.Errorf("test file %v does not exist", f.Path)
	}
}

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
	newFile := NewFile(expectedPath)
	assertFileExists(t, newFile)
	return newFile
}

func readFile(t *testing.T, path string) string {
	t.Helper()
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %v: %v", path, err)
	}
	return string(b)
}

func assertNewContentsOfFile(t *testing.T, path, initial, find, replace, want string) {
	t.Helper()
	got := readFile(t, path)
	if got != want {
		t.Errorf("replace %v with %v in %v, but got %v; want %v", find, replace, initial, got, want)
	}
}

/*
 * Tests
 */

// TestWalkDir is the most important test of the entire suite, because it
// exercises all the basic functionality of the app. It builds a directory
// tree of temporary files and directories, walks the entire tree, and ensures
// that all files and directories are appropriately renamed and contain the
// correct contents.
func TestWalkDir(t *testing.T) {
	find := "wh"
	replace := "f"

	d := NewFile(t.TempDir())

	// d1: who/
	d1 := newTestDir(t, d.Path, "who")

	// d1d1: who/what/
	d1d1 := newTestDir(t, d1.Path, "what")

	// d1d1f1: who/what/when (contains "where")
	d1d1f1Contents := "where"
	d1d1f1 := newTestFile(t, d1d1.Path, "when", d1d1f1Contents)

	// d2: what/
	d2 := newTestDir(t, d.Path, "what")

	// d2d1: what/when/
	d2d1 := newTestDir(t, d2.Path, "when")

	// d2d1d1: what/when/where (directories with no files)
	d2d1d1 := newTestDir(t, d2d1.Path, "where")

	// d3: when/
	d3 := newTestDir(t, d.Path, "when")

	// d3f1: when/where (contains "why")
	d3f1Contents := "why"
	d3f1 := newTestFile(t, d3.Path, "where", d3f1Contents)

	// d4: where/ (empty directory in base dir)
	d4 := newTestDir(t, d.Path, "where")

	// f1: why (file in base dir contains "wh")
	f1Contents := "wh\nwh\nwh\n"
	f1 := newTestFile(t, d.Path, "why", f1Contents)

	fr := findReplace{find: find, replace: replace}
	fr.WalkDir(d)
	if fr.errors != 0 {
		t.Fatalf("walk reported %d errors", fr.errors)
	}

	// d1: who/ > fo/
	d1ExpectedPath := expectedPathAfterRename(d1, &fr)
	assertPathExistsAfterRename(t, d1, d1ExpectedPath)

	// d1d1: who/what/ > fo/foat/
	d1d1ExpectedPath := filepath.Join(d1ExpectedPath, strings.Replace(d1d1.Base(), fr.find, fr.replace, -1))
	assertPathExistsAfterRename(t, d1d1, d1d1ExpectedPath)

	// d1d1f1: who/what/when > fo/fat/fen (contains "fere")
	d1d1f1ExpectedPath := filepath.Join(d1d1ExpectedPath, strings.Replace(d1d1f1.Base(), fr.find, fr.replace, -1))
	assertPathExistsAfterRename(t, d1d1f1, d1d1f1ExpectedPath)
	assertNewContentsOfFile(t, d1d1f1ExpectedPath, d1d1f1Contents, find, replace, "fere")

	// d2: what/ > fat/
	d2ExpectedPath := expectedPathAfterRename(d2, &fr)
	assertPathExistsAfterRename(t, d2, d2ExpectedPath)

	// d2d1: what/when/
	d2d1ExpectedPath := filepath.Join(d2ExpectedPath, strings.Replace(d2d1.Base(), fr.find, fr.replace, -1))
	assertPathExistsAfterRename(t, d2d1, d2d1ExpectedPath)

	// d2d1d1: what/when/where (directories with no files)
	d2d1d1ExpectedPath := filepath.Join(d2d1ExpectedPath, strings.Replace(d2d1d1.Base(), fr.find, fr.replace, -1))
	assertPathExistsAfterRename(t, d2d1d1, d2d1d1ExpectedPath)

	// d3: when/
	d3ExpectedPath := expectedPathAfterRename(d3, &fr)
	assertPathExistsAfterRename(t, d3, d3ExpectedPath)

	// d3f1: when/where (contains "why")
	d3f1ExpectedPath := filepath.Join(d3ExpectedPath, strings.Replace(d3f1.Base(), fr.find, fr.replace, -1))
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

func TestIgnoredGitDir(t *testing.T) {
	root := t.TempDir()
	gitDir := filepath.Join(root, ".git")
	if err := os.Mkdir(gitDir, 0o700); err != nil {
		t.Fatal(err)
	}
	gitFile := filepath.Join(gitDir, "config")
	if err := os.WriteFile(gitFile, []byte("git contents"), 0o600); err != nil {
		t.Fatal(err)
	}

	fr := findReplace{find: "git", replace: "got"}
	fr.WalkDir(NewFile(root))
	if fr.errors != 0 {
		t.Fatalf("walk reported %d errors", fr.errors)
	}

	// The .git directory must remain unchanged.
	if _, err := os.Stat(gitDir); err != nil {
		t.Errorf(".git was renamed/removed: %v", err)
	}
	if got := readFile(t, gitFile); got != "git contents" {
		t.Errorf(".git/config was rewritten: %q", got)
	}
}

func TestRenameSingleFile(t *testing.T) {
	d := t.TempDir()
	f := newTestFile(t, d, "alpha", "")

	fr := findReplace{find: "ph", replace: "f"}
	expectedPath := filepath.Join(f.Dir(), strings.Replace(f.Base(), fr.find, fr.replace, -1))

	fr.WalkDir(NewFile(d))
	assertPathExistsAfterRename(t, f, expectedPath)
}

func TestReplaceContents(t *testing.T) {
	d := t.TempDir()
	f := newTestFile(t, d, "*", "alpha")

	fr := findReplace{find: "ph", replace: "f"}
	fr.WalkDir(NewFile(d))
	assertNewContentsOfFile(t, f.Path, "alpha", "ph", "f", "alfa")
}

func TestReplaceContentsEntireFile(t *testing.T) {
	d := t.TempDir()
	f := newTestFile(t, d, "*", "alpha")

	fr := findReplace{find: "alpha", replace: "beta"}
	fr.WalkDir(NewFile(d))
	assertNewContentsOfFile(t, f.Path, "alpha", "alpha", "beta", "beta")
}

func TestReplaceContentsMultipleMatchesSingleLine(t *testing.T) {
	d := t.TempDir()
	f := newTestFile(t, d, "*", "alphaalpha")

	fr := findReplace{find: "ph", replace: "f"}
	fr.WalkDir(NewFile(d))
	assertNewContentsOfFile(t, f.Path, "alphaalpha", "ph", "f", "alfaalfa")
}

func TestReplaceContentsMultipleMatchesMultipleLines(t *testing.T) {
	d := t.TempDir()
	f := newTestFile(t, d, "*", "alpha\nalpha")

	fr := findReplace{find: "ph", replace: "f"}
	fr.WalkDir(NewFile(d))
	assertNewContentsOfFile(t, f.Path, "alpha\nalpha", "ph", "f", "alfa\nalfa")
}

func TestReplaceContentsNoMatches(t *testing.T) {
	d := t.TempDir()
	f := newTestFile(t, d, "*", "alpha")

	fr := findReplace{find: "abc", replace: "xyz"}
	fr.WalkDir(NewFile(d))
	assertNewContentsOfFile(t, f.Path, "alpha", "abc", "xyz", "alpha")
}

// BenchmarkSyntheticTree benchmarks find-replace against a synthetic tree of
// files with controlled size so the benchmark is reproducible and not network-
// bound.
func BenchmarkSyntheticTree(b *testing.B) {
	const dirs = 10
	const filesPerDir = 100
	const fileBytes = 4 * 1024

	root := b.TempDir()
	body := strings.Repeat("alpha beta gamma\n", fileBytes/16)
	for i := 0; i < dirs; i++ {
		dir := filepath.Join(root, fmt.Sprintf("dir-alpha-%d", i))
		if err := os.Mkdir(dir, 0o755); err != nil {
			b.Fatal(err)
		}
		for j := 0; j < filesPerDir; j++ {
			path := filepath.Join(dir, fmt.Sprintf("file-alpha-%d.txt", j))
			if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
				b.Fatal(err)
			}
		}
	}

	b.ResetTimer()
	for n := 0; n < b.N; n++ {
		// We can't safely rerun with rename in place, so use a fresh
		// subtree per iteration.
		b.StopTimer()
		work := filepath.Join(b.TempDir(), "tree")
		if err := copyTree(root, work); err != nil {
			b.Fatal(err)
		}
		b.StartTimer()

		fr := findReplace{find: "alpha", replace: "BETA"}
		fr.WalkDir(NewFile(work))
		if fr.errors != 0 {
			b.Fatalf("benchmark reported %d errors", fr.errors)
		}
	}
}

// copyTree recursively copies the regular files and directories under src
// into dst. Used by benchmarks to set up a fresh working tree without
// requiring os.CopyFS (Go 1.23+).
func copyTree(src, dst string) error {
	return filepath.Walk(src, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		target := filepath.Join(dst, rel)
		if info.IsDir() {
			return os.MkdirAll(target, info.Mode().Perm())
		}
		if !info.Mode().IsRegular() {
			return nil
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		return os.WriteFile(target, data, info.Mode().Perm())
	})
}
