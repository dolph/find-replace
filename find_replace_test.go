package main

import (
	"errors"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func createTestFile(path string, baseName string, content string) (string, fs.DirEntry, string) {
	f, err := os.CreateTemp(path, baseName)
	if err != nil {
		log.Fatal(err)
	}

	fileInfo, err := f.Stat()
	if err != nil {
		log.Fatal(err)
	}

	if _, err := f.Write([]byte(content)); err != nil {
		log.Fatal(err)
	}
	if err := f.Close(); err != nil {
		log.Fatal(err)
	}

	dirName := filepath.Dir(f.Name())

	// There has to be a better way to get `fInfo` directly for `f`?
	files, err := os.ReadDir(dirName)
	if err != nil {
		log.Fatal(err)
	}
	fInfo := files[0]
	for _, file := range files {
		if file.Name() == fileInfo.Name() {
			fInfo = file
			break
		}
	}

	return dirName, fInfo, f.Name()
}

func assertFileExistsBeforeRename(t *testing.T, path string) {
	// Ensure file exists as expected before renaming
	if _, err := os.Stat(path); errors.Is(err, os.ErrNotExist) {
		t.Errorf("test file %v does not exist", path)
	}
}

func assertFileExistsAfterRename(t *testing.T, oldPath string, newPath string) {
	if _, err := os.Stat(oldPath); err == nil {
		t.Errorf("test file %v still exists after it was supposed to be renamed to %v", oldPath, newPath)
	}
	if _, err := os.Stat(newPath); errors.Is(err, os.ErrNotExist) {
		t.Errorf("renamed test file %v does not exist", newPath)
	}
}

func TestHandleFileWithFile(t *testing.T) {
	initial := "alpha"
	find := "ph"
	replace := "f"
	want := "alfa"

	dirName, fInfo, path := createTestFile("", initial, initial)
	defer os.Remove(path)
	expectedName := strings.Replace(fInfo.Name(), find, replace, -1)
	expectedPath := dirName + string(os.PathSeparator) + expectedName
	defer os.Remove(expectedPath)
	fr := findReplace{find: find, replace: replace}

	assertFileExistsBeforeRename(t, path)
	fr.HandleFile(dirName, fInfo)
	assertFileExistsAfterRename(t, path, expectedPath)

	got := readFile(expectedPath)
	if got != want {
		t.Errorf("replace %v with %v in %v, but got %v; want %v", find, replace, initial, got, want)
	}
}

func TestRenameFile(t *testing.T) {
	initial := "alpha"
	find := "ph"
	replace := "f"

	dirName, fInfo, path := createTestFile("", initial, "")
	defer os.Remove(path)
	expectedName := strings.Replace(fInfo.Name(), find, replace, -1)
	expectedPath := dirName + string(os.PathSeparator) + expectedName
	defer os.Remove(expectedPath)
	fr := findReplace{find: find, replace: replace}

	assertFileExistsBeforeRename(t, path)
	fr.RenameFile(dirName, fInfo)
	assertFileExistsAfterRename(t, path, expectedPath)
}

func assertNewContentsOfFile(t *testing.T, path string, initial string, find string, replace string, want string) {
	got := readFile(path)
	if got != want {
		t.Errorf("replace %v with %v in %v, but got %v; want %v", find, replace, initial, got, want)
	}
}

func TestReplaceContents(t *testing.T) {
	initial := "alpha"
	find := "ph"
	replace := "f"
	want := "alfa"

	dirName, fInfo, path := createTestFile("", "*", initial)
	defer os.Remove(path)
	fr := findReplace{find: find, replace: replace}
	fr.ReplaceContents(dirName, fInfo)
	assertNewContentsOfFile(t, path, initial, find, replace, want)
}

func TestReplaceContentsEntireFile(t *testing.T) {
	initial := "alpha"
	find := "alpha"
	replace := "beta"
	want := "beta"

	dirName, fInfo, path := createTestFile("", "*", initial)
	defer os.Remove(path)
	fr := findReplace{find: find, replace: replace}
	fr.ReplaceContents(dirName, fInfo)
	assertNewContentsOfFile(t, path, initial, find, replace, want)
}

func TestReplaceContentsMultipleMatchesSingleLine(t *testing.T) {
	initial := "alphaalpha"
	find := "ph"
	replace := "f"
	want := "alfaalfa"

	dirName, fInfo, path := createTestFile("", "*", initial)
	defer os.Remove(path)
	fr := findReplace{find: find, replace: replace}
	fr.ReplaceContents(dirName, fInfo)
	assertNewContentsOfFile(t, path, initial, find, replace, want)
}

func TestReplaceContentsMultipleMatchesMultipleLines(t *testing.T) {
	initial := "alpha\nalpha"
	find := "ph"
	replace := "f"
	want := "alfa\nalfa"

	dirName, fInfo, path := createTestFile("", "*", initial)
	defer os.Remove(path)
	fr := findReplace{find: find, replace: replace}
	fr.ReplaceContents(dirName, fInfo)
	assertNewContentsOfFile(t, path, initial, find, replace, want)
}

func TestReplaceContentsNoMatches(t *testing.T) {
	initial := "alpha"
	find := "abc"
	replace := "xyz"
	want := "alpha"

	dirName, fInfo, path := createTestFile("", "*", initial)
	defer os.Remove(path)
	fr := findReplace{find: find, replace: replace}
	fr.ReplaceContents(dirName, fInfo)
	assertNewContentsOfFile(t, path, initial, find, replace, want)
}

func assertRandomStringLength(t *testing.T, ask int, want int) {
	got := len(randomString(ask))
	if got != want {
		t.Errorf("len(RandomString(%v)) = %v; want %v", ask, got, want)
	}
}

func TestRandomStringLengthNegativeOne(t *testing.T) {
	assertRandomStringLength(t, -1, 0)
}

func TestRandomStringLengthZero(t *testing.T) {
	assertRandomStringLength(t, 0, 0)
}

func TestRandomStringLengthOne(t *testing.T) {
	assertRandomStringLength(t, 1, 1)
}

func TestRandomStringLengthTen(t *testing.T) {
	assertRandomStringLength(t, 10, 10)
}

func TestRandomStringLengthTwenty(t *testing.T) {
	assertRandomStringLength(t, 20, 20)
}
