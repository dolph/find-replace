package main

import (
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"testing"
)

func createTestFile(path string, baseName string, content string) (string, fs.DirEntry, string) {
	f, err := os.CreateTemp(path, baseName)
	if err != nil {
		log.Fatal(err)
	}

	file_info, err := f.Stat()
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

	// There has to be a better way to get `f_info` directly for `f`?
	files, err := os.ReadDir(dirName)
	if err != nil {
		log.Fatal(err)
	}
	f_info := files[0]
	for _, file := range files {
		if file.Name() == file_info.Name() {
			f_info = file
			break
		}
	}

	return dirName, f_info, f.Name()
}

func TestReplaceContents(t *testing.T) {
	initial := "alpha"
	find := "ph"
	replace := "f"
	want := "alfa"

	dirName, f_info, path := createTestFile("", "*", initial)
	defer os.Remove(path)
	fr := findReplace{find: find, replace: replace}
	fr.ReplaceContents(dirName, f_info)
	got := readFile(path)
	if got != want {
		t.Errorf("replace %v with %v in %v, but got %v; want %v", find, replace, initial, got, want)
	}
}

func TestReplaceContentsEntireFile(t *testing.T) {
	initial := "alpha"
	find := "alpha"
	replace := "beta"
	want := "beta"

	dirName, f_info, path := createTestFile("", "*", initial)
	defer os.Remove(path)
	fr := findReplace{find: find, replace: replace}
	fr.ReplaceContents(dirName, f_info)
	got := readFile(path)
	if got != want {
		t.Errorf("replace %v with %v in %v, but got %v; want %v", find, replace, initial, got, want)
	}
}

func TestReplaceContentsMultipleMatchesSingleLine(t *testing.T) {
	initial := "alphaalpha"
	find := "ph"
	replace := "f"
	want := "alfaalfa"

	dirName, f_info, path := createTestFile("", "*", initial)
	defer os.Remove(path)
	fr := findReplace{find: find, replace: replace}
	fr.ReplaceContents(dirName, f_info)
	got := readFile(path)
	if got != want {
		t.Errorf("replace %v with %v in %v, but got %v; want %v", find, replace, initial, got, want)
	}
}

func TestReplaceContentsMultipleMatchesMultipleLines(t *testing.T) {
	initial := "alpha\nalpha"
	find := "ph"
	replace := "f"
	want := "alfa\nalfa"

	dirName, f_info, path := createTestFile("", "*", initial)
	defer os.Remove(path)
	fr := findReplace{find: find, replace: replace}
	fr.ReplaceContents(dirName, f_info)
	got := readFile(path)
	if got != want {
		t.Errorf("replace %v with %v in %v, but got %v; want %v", find, replace, initial, got, want)
	}
}

func TestReplaceContentsNoMatches(t *testing.T) {
	initial := "alpha"
	find := "abc"
	replace := "xyz"
	want := "alpha"

	dirName, f_info, path := createTestFile("", "*", initial)
	defer os.Remove(path)
	fr := findReplace{find: find, replace: replace}
	fr.ReplaceContents(dirName, f_info)
	got := readFile(path)
	if got != want {
		t.Errorf("replace %v with %v in %v, but got %v; want %v", find, replace, initial, got, want)
	}
}

func TestRandomStringLengthNegativeOne(t *testing.T) {
	got := randomString(-1)
	if len(got) != 0 {
		t.Errorf("len(RandomString(-1)) = %v; want 0", got)
	}
}

func TestRandomStringLengthZero(t *testing.T) {
	got := randomString(0)
	if len(got) != 0 {
		t.Errorf("len(RandomString(0)) = %v; want 0", got)
	}
}

func TestRandomStringLengthOne(t *testing.T) {
	got := randomString(1)
	if len(got) != 1 {
		t.Errorf("len(RandomString(1)) = %v; want 1", got)
	}
}

func TestRandomStringLengthTen(t *testing.T) {
	got := randomString(10)
	if len(got) != 10 {
		t.Errorf("len(RandomString(10)) = %v; want 10", got)
	}
}

func TestRandomStringLengthTwenty(t *testing.T) {
	got := randomString(20)
	if len(got) != 20 {
		t.Errorf("len(RandomString(20)) = %v; want 20", got)
	}
}
