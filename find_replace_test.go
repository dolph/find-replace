package main

import (
	"log"
	"os"
	"path/filepath"
	"testing"
)

func TestReplaceContents(t *testing.T) {
	initial := "alpha"

	f, err := os.CreateTemp("", "*")
	if err != nil {
		log.Fatal(err)
	}
	defer os.Remove(f.Name())

	file_info, err := f.Stat()
	if err != nil {
		log.Fatal(err)
	}
	baseName := file_info.Name()

	if _, err := f.Write([]byte(initial)); err != nil {
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
		if file.Name() == baseName {
			f_info = file
			break
		}
	}

	find := "ph"
	replace := "f"
	want := "alfa"
	fr := findReplace{find: "ph", replace: "f"}
	fr.ReplaceContents(dirName, f_info)

	got := readFile(f.Name())
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
