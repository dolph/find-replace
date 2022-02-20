package main

import (
	"errors"
	"log"
	"os"
	"path/filepath"
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
func newTestFile(path string, baseName string, content string) *File {
	f, err := os.CreateTemp(path, baseName)
	if err != nil {
		log.Fatal(err)
	}
	if _, err := f.Write([]byte(content)); err != nil {
		defer os.Remove(f.Name())
		log.Fatal(err)
	}
	if err := f.Close(); err != nil {
		defer os.Remove(f.Name())
		log.Fatal(err)
	}

	return NewFile(f.Name())
}

// newTestDir creates a directory in the given directory path, with the given
// base name. If a directory path is not provided, a temp directory is used. If
// a baseName is not provided, a random file name is generated. Returns the
// directory where the file was created, the file's directory entry, and the
// actual name of the file.
func newTestDir(path string, baseName string) *File {
	dirPath, err := os.MkdirTemp(path, baseName)
	if err != nil {
		log.Fatal(err)
	}
	return NewFile(dirPath)
}

func expectedPathAfterRename(f *File, fr *findReplace) string {
	return filepath.Join(f.Dir(), strings.Replace(f.Base(), fr.find, fr.replace, -1))
}

/*
 * Assertions
 */

// assertFileExists ensures that the given File exists
func assertFileExists(t *testing.T, f *File) {
	if _, err := os.Stat(f.Path); errors.Is(err, os.ErrNotExist) {
		t.Errorf("test file %v does not exist", f.Path)
	}
}

// assertFileNonexistent ensures that the File does not exist
func assertFileNonexistent(t *testing.T, f *File) {
	if _, err := os.Stat(f.Path); !errors.Is(err, os.ErrNotExist) {
		if err == nil {
			t.Errorf("test file %v exists", f.Path)
		} else {
			t.Errorf("test file %v exists (got %v)", f.Path, err)
		}
	}
}

func assertPathExistsAfterRename(t *testing.T, f *File, expectedPath string) *File {
	assertFileNonexistent(t, f)
	newFile := NewFile(expectedPath)
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

	d := newTestDir("", "*")
	defer os.Remove(d.Path)

	// d1: who/
	d1 := newTestDir(d.Path, "who")
	defer os.Remove(d1.Path)

	// d1d1: who/what/
	d1d1 := newTestDir(d1.Path, "what")
	defer os.Remove(d1d1.Path)

	// d1d1f1: who/what/when (contains "where")
	d1d1f1Contents := "where"
	d1d1f1 := newTestFile(d1d1.Path, "when", d1d1f1Contents)
	defer os.Remove(d1d1f1.Path)

	// d2: what/
	d2 := newTestDir(d.Path, "what")
	defer os.Remove(d2.Path)

	// d2d1: what/when/
	d2d1 := newTestDir(d2.Path, "when")
	defer os.Remove(d2d1.Path)

	// d2d1d1: what/when/where (directories with no files)
	d2d1d1 := newTestDir(d2d1.Path, "where")
	defer os.Remove(d2d1d1.Path)

	// d3: when/
	d3 := newTestDir(d.Path, "when")
	defer os.Remove(d3.Path)

	// d3f1: when/where (contains "why")
	d3f1Contents := "why"
	d3f1 := newTestFile(d3.Path, "where", d3f1Contents)
	defer os.Remove(d3f1.Path)

	// d4: where/ (empty directory in base dir)
	d4 := newTestDir(d.Path, "where")
	defer os.Remove(d4.Path)

	// f1: why (file in base dir contains "wh")
	f1Contents := "wh\nwh\nwh\n"
	f1 := newTestFile(d.Path, "why", f1Contents)
	defer os.Remove(f1.Path)

	fr := findReplace{find: find, replace: replace}
	fr.WalkDir(d)

	// d1: who/ > fo/
	d1ExpectedPath := expectedPathAfterRename(d1, &fr)
	assertPathExistsAfterRename(t, d1, d1ExpectedPath)

	// d1d1: who/what/ > fo/foat/
	d1d1ExpectedPath := filepath.Join(d1ExpectedPath, strings.Replace(d1d1.Base(), fr.find, fr.replace, -1))
	d1d1 = assertPathExistsAfterRename(t, d1d1, d1d1ExpectedPath)

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

func TestHandleFileWithDir(t *testing.T) {
	initial := "alpha"
	find := "ph"
	replace := "f"

	f := newTestDir("", initial)
	defer os.Remove(f.Path)
	expectedPath := filepath.Join(f.Dir(), strings.Replace(f.Base(), find, replace, -1))
	defer os.Remove(expectedPath)
	fr := findReplace{find: find, replace: replace}

	assertFileExists(t, f)
	fr.HandleFile(f)
	assertPathExistsAfterRename(t, f, expectedPath)
}

func TestHandleFileWithIgnoredDir(t *testing.T) {
	initial := ".git"
	find := "git"
	replace := "got"

	dirPath := filepath.Join(os.TempDir(), initial)
	if err := os.Mkdir(dirPath, 0700); err != nil {
		log.Fatal(err)
	}
	f := NewFile(dirPath)
	defer os.Remove(f.Path)
	// Just in case it's unexpectedly renamed, let's make sure we cleanup the
	// anticipated name.
	unexpectedName := strings.Replace(f.Base(), find, replace, -1)
	unexpectedPath := filepath.Join(f.Dir(), unexpectedName)
	defer os.Remove(unexpectedPath)
	fr := findReplace{find: find, replace: replace}

	assertFileExists(t, f)
	fr.HandleFile(f)
	assertFileExists(t, f)
}

func TestHandleFileWithFile(t *testing.T) {
	initial := "alpha"
	find := "ph"
	replace := "f"
	want := "alfa"

	f := newTestFile("", initial, initial)
	defer os.Remove(f.Path)
	expectedName := strings.Replace(f.Base(), find, replace, -1)
	expectedPath := filepath.Join(f.Dir(), expectedName)
	defer os.Remove(expectedPath)
	fr := findReplace{find: find, replace: replace}

	assertFileExists(t, f)
	fr.HandleFile(f)
	assertPathExistsAfterRename(t, f, expectedPath)

	got := NewFile(expectedPath).Read()
	if got != want {
		t.Errorf("replace %v with %v in %v, but got %v; want %v", find, replace, initial, got, want)
	}
}

func TestRenameFile(t *testing.T) {
	initial := "alpha"
	find := "ph"
	replace := "f"

	f := newTestFile("", initial, "")
	defer os.Remove(f.Path)
	expectedName := strings.Replace(f.Base(), find, replace, -1)
	expectedPath := filepath.Join(f.Dir(), expectedName)
	defer os.Remove(expectedPath)
	fr := findReplace{find: find, replace: replace}

	assertFileExists(t, f)
	fr.RenameFile(f)
	assertPathExistsAfterRename(t, f, expectedPath)
}

// assertNewContentsOfFile ensures that the contents of the file at the given
// path exactly match the desired string.
func assertNewContentsOfFile(t *testing.T, path string, initial string, find string, replace string, want string) {
	got := NewFile(path).Read()
	if got != want {
		t.Errorf("replace %v with %v in %v, but got %v; want %v", find, replace, initial, got, want)
	}
}

func TestReplaceContents(t *testing.T) {
	initial := "alpha"
	find := "ph"
	replace := "f"
	want := "alfa"

	f := newTestFile("", "*", initial)
	defer os.Remove(f.Path)
	fr := findReplace{find: find, replace: replace}
	fr.ReplaceContents(f)
	assertNewContentsOfFile(t, f.Path, initial, find, replace, want)
}

func TestReplaceContentsEntireFile(t *testing.T) {
	initial := "alpha"
	find := "alpha"
	replace := "beta"
	want := "beta"

	f := newTestFile("", "*", initial)
	defer os.Remove(f.Path)
	fr := findReplace{find: find, replace: replace}
	fr.ReplaceContents(f)
	assertNewContentsOfFile(t, f.Path, initial, find, replace, want)
}

func TestReplaceContentsMultipleMatchesSingleLine(t *testing.T) {
	initial := "alphaalpha"
	find := "ph"
	replace := "f"
	want := "alfaalfa"

	f := newTestFile("", "*", initial)
	defer os.Remove(f.Path)
	fr := findReplace{find: find, replace: replace}
	fr.ReplaceContents(f)
	assertNewContentsOfFile(t, f.Path, initial, find, replace, want)
}

func TestReplaceContentsMultipleMatchesMultipleLines(t *testing.T) {
	initial := "alpha\nalpha"
	find := "ph"
	replace := "f"
	want := "alfa\nalfa"

	f := newTestFile("", "*", initial)
	defer os.Remove(f.Path)
	fr := findReplace{find: find, replace: replace}
	fr.ReplaceContents(f)
	assertNewContentsOfFile(t, f.Path, initial, find, replace, want)
}

func TestReplaceContentsNoMatches(t *testing.T) {
	initial := "alpha"
	find := "abc"
	replace := "xyz"
	want := "alpha"

	f := newTestFile("", "*", initial)
	defer os.Remove(f.Path)
	fr := findReplace{find: find, replace: replace}
	fr.ReplaceContents(f)
	assertNewContentsOfFile(t, f.Path, initial, find, replace, want)
}
