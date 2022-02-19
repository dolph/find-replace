package main

import (
	"errors"
	"log"
	"os"
	"strings"
	"testing"
)

// createTestFile creates a file in the given directory path, with the given
// name and content. If a directory path is not provided, a temp directory is
// used. If a baseName is not provided, a random file name is generated.
// Returns the directory where the file was created, the file's directory
// entry, and the actual name of the file.
func createTestFile(path string, baseName string, content string) *File {
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

// createTestDir creates a directory in the given directory path, with the
// given base name. If a directory path is not provided, a temp directory is
// used. If a baseName is not provided, a random file name is generated.
// Returns the directory where the file was created, the file's directory
// entry, and the actual name of the file.
func createTestDir(path string, baseName string) *File {
	dirPath, err := os.MkdirTemp(path, baseName)
	if err != nil {
		log.Fatal(err)
	}
	return NewFile(dirPath)
}

// assertPathExists ensures that the file at the given path exists
// prior to being renamed.
func assertPathExists(t *testing.T, path string) {
	// Ensure file exists as expected before renaming
	if _, err := os.Stat(path); errors.Is(err, os.ErrNotExist) {
		t.Errorf("test file %v does not exist", path)
	}
}

// assertPathExistsAfterRename ensures that the file at oldPath no longer
// exists, and that a file at newPath exists instead.
func assertPathExistsAfterRename(t *testing.T, oldPath string, newPath string) {
	if _, err := os.Stat(oldPath); err == nil {
		t.Errorf("test file %v still exists after it was supposed to be renamed to %v", oldPath, newPath)
	}
	if _, err := os.Stat(newPath); errors.Is(err, os.ErrNotExist) {
		t.Errorf("renamed test file %v does not exist", newPath)
	}
}

func TestWalkDir(t *testing.T) {
	find := "wh"
	replace := "f"

	d := createTestDir("", "*")
	defer os.Remove(d.Path)

	// d1: who/
	d1 := createTestDir(d.Path, "who")
	defer os.Remove(d1.Path)

	// d1d1: who/what/
	d1d1 := createTestDir(d1.Path, "what")
	defer os.Remove(d1d1.Path)

	// d1d1f1: who/what/when (contains "where")
	d1d1f1Contents := "where"
	d1d1f1 := createTestFile(d1d1.Path, "when", d1d1f1Contents)
	defer os.Remove(d1d1f1.Path)

	// d2: what/
	d2 := createTestDir(d.Path, "what")
	defer os.Remove(d2.Path)

	// d2d1: what/when/
	d2d1 := createTestDir(d2.Path, "when")
	defer os.Remove(d2d1.Path)

	// d2d1d1: what/when/where (directories with no files)
	d2d1d1 := createTestDir(d2d1.Path, "where")
	defer os.Remove(d2d1d1.Path)

	// d3: when/
	d3 := createTestDir(d.Path, "when")
	defer os.Remove(d3.Path)

	// d3f1: when/where (contains "why")
	d3f1Contents := "why"
	d3f1 := createTestFile(d3.Path, "where", d3f1Contents)
	defer os.Remove(d3f1.Path)

	// d4: where/ (empty directory in base dir)
	d4 := createTestDir(d.Path, "where")
	defer os.Remove(d4.Path)

	// f1: why (file in base dir contains "wh")
	f1Contents := "wh\nwh\nwh\n"
	f1 := createTestFile(d.Path, "why", f1Contents)
	defer os.Remove(f1.Path)

	fr := findReplace{find: find, replace: replace}
	fr.WalkDir(d)

	// d1: who/ > fo/
	d1ExpectedPath := d1.Dir() + string(os.PathSeparator) + strings.Replace(d1.Base(), find, replace, -1)
	assertPathExistsAfterRename(t, d1.Path, d1ExpectedPath)

	// d1d1: who/what/ > fo/foat/
	d1d1ExpectedPath := d1ExpectedPath + string(os.PathSeparator) + strings.Replace(d1d1.Base(), find, replace, -1)
	assertPathExistsAfterRename(t, d1d1.Path, d1d1ExpectedPath)

	// d1d1f1: who/what/when > fo/fat/fen (contains "fere")
	d1d1f1ExpectedPath := d1d1ExpectedPath + string(os.PathSeparator) + strings.Replace(d1d1f1.Base(), find, replace, -1)
	assertPathExistsAfterRename(t, d1d1f1.Path, d1d1f1ExpectedPath)
	assertNewContentsOfFile(t, d1d1f1ExpectedPath, d1d1f1Contents, find, replace, "fere")

	// d2: what/ > fat/
	d2ExpectedPath := d2.Dir() + string(os.PathSeparator) + strings.Replace(d2.Base(), find, replace, -1)
	assertPathExistsAfterRename(t, d2.Path, d2ExpectedPath)

	// d2d1: what/when/
	d2d1ExpectedPath := d2ExpectedPath + string(os.PathSeparator) + strings.Replace(d2d1.Base(), find, replace, -1)
	assertPathExistsAfterRename(t, d2d1.Path, d2d1ExpectedPath)

	// d2d1d1: what/when/where (directories with no files)
	d2d1d1ExpectedPath := d2d1ExpectedPath + string(os.PathSeparator) + strings.Replace(d2d1d1.Base(), find, replace, -1)
	assertPathExistsAfterRename(t, d2d1d1.Path, d2d1d1ExpectedPath)

	// d3: when/
	d3ExpectedPath := d3.Dir() + string(os.PathSeparator) + strings.Replace(d3.Base(), find, replace, -1)
	assertPathExistsAfterRename(t, d3.Path, d3ExpectedPath)

	// d3f1: when/where (contains "why")
	d3f1ExpectedPath := d3ExpectedPath + string(os.PathSeparator) + strings.Replace(d3f1.Base(), find, replace, -1)
	assertPathExistsAfterRename(t, d3f1.Path, d3f1ExpectedPath)
	assertNewContentsOfFile(t, d3f1ExpectedPath, d3f1Contents, find, replace, "fy")

	// d4: where/ (empty directory in base dir)
	d4ExpectedPath := d4.Dir() + string(os.PathSeparator) + strings.Replace(d4.Base(), find, replace, -1)
	assertPathExistsAfterRename(t, d4.Path, d4ExpectedPath)

	// f1: why (file in base dir contains "wh\nwh\nwh\n")
	f1ExpectedPath := f1.Dir() + string(os.PathSeparator) + strings.Replace(f1.Base(), find, replace, -1)
	assertPathExistsAfterRename(t, f1.Path, f1ExpectedPath)
	assertNewContentsOfFile(t, f1ExpectedPath, f1Contents, find, replace, "f\nf\nf\n")
}

func TestHandleFileWithDir(t *testing.T) {
	initial := "alpha"
	find := "ph"
	replace := "f"

	f := createTestDir("", initial)
	defer os.Remove(f.Path)
	expectedName := strings.Replace(f.Base(), find, replace, -1)
	expectedPath := f.Dir() + string(os.PathSeparator) + expectedName
	defer os.Remove(expectedPath)
	fr := findReplace{find: find, replace: replace}

	assertPathExists(t, f.Path)
	fr.HandleFile(f)
	assertPathExistsAfterRename(t, f.Path, expectedPath)
}

func TestHandleFileWithIgnoredDir(t *testing.T) {
	initial := ".git"
	find := "git"
	replace := "got"

	dirPath := os.TempDir() + string(os.PathSeparator) + initial
	if err := os.Mkdir(dirPath, 0700); err != nil {
		log.Fatal(err)
	}
	f := NewFile(dirPath)
	defer os.Remove(f.Path)
	// Just in case it's unexpectedly renamed, let's make sure we cleanup the
	// anticipated name.
	unexpectedName := strings.Replace(f.Base(), find, replace, -1)
	unexpectedPath := f.Dir() + string(os.PathSeparator) + unexpectedName
	defer os.Remove(unexpectedPath)
	fr := findReplace{find: find, replace: replace}

	assertPathExists(t, f.Path)
	fr.HandleFile(f)
	assertPathExists(t, f.Path)
}

func TestHandleFileWithFile(t *testing.T) {
	initial := "alpha"
	find := "ph"
	replace := "f"
	want := "alfa"

	f := createTestFile("", initial, initial)
	defer os.Remove(f.Path)
	expectedName := strings.Replace(f.Base(), find, replace, -1)
	expectedPath := f.Dir() + string(os.PathSeparator) + expectedName
	defer os.Remove(expectedPath)
	fr := findReplace{find: find, replace: replace}

	assertPathExists(t, f.Path)
	fr.HandleFile(f)
	assertPathExistsAfterRename(t, f.Path, expectedPath)

	got := NewFile(expectedPath).Read()
	if got != want {
		t.Errorf("replace %v with %v in %v, but got %v; want %v", find, replace, initial, got, want)
	}
}

func TestRenameFile(t *testing.T) {
	initial := "alpha"
	find := "ph"
	replace := "f"

	f := createTestFile("", initial, "")
	defer os.Remove(f.Path)
	expectedName := strings.Replace(f.Base(), find, replace, -1)
	expectedPath := f.Dir() + string(os.PathSeparator) + expectedName
	defer os.Remove(expectedPath)
	fr := findReplace{find: find, replace: replace}

	assertPathExists(t, f.Path)
	fr.RenameFile(f)
	assertPathExistsAfterRename(t, f.Path, expectedPath)
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

	f := createTestFile("", "*", initial)
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

	f := createTestFile("", "*", initial)
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

	f := createTestFile("", "*", initial)
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

	f := createTestFile("", "*", initial)
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

	f := createTestFile("", "*", initial)
	defer os.Remove(f.Path)
	fr := findReplace{find: find, replace: replace}
	fr.ReplaceContents(f)
	assertNewContentsOfFile(t, f.Path, initial, find, replace, want)
}
