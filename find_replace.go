package main

import (
	"errors"
	"io"
	"log"
	"math/rand"
	"os"
	"path/filepath"
	"strings"
	"time"

	"golang.org/x/tools/godoc/util"
)

// findReplace is a struct used to provide context to all find & replace
// operations, including the strings to search for & replace.
type findReplace struct {
	find    string
	replace string
}

type File struct {
	Path string
	info os.FileInfo
}

func NewFile(path string) *File {
	absPath, err := filepath.Abs(path)
	if err != nil {
		log.Fatalf("Unable to resolve absolute path of %v: %v", path, err)
	}
	return &File{Path: absPath}
}

func (f *File) Base() string {
	return filepath.Base(f.Path)
}

func (f *File) Dir() string {
	return filepath.Dir(f.Path)
}

func (f *File) Info() os.FileInfo {
	if f.info == nil {
		stat, err := os.Stat(f.Path)
		if err != nil {
			log.Fatalf("Failed to stat %v: %v", f.Path, err)
		}
		f.info = stat
	}
	return f.info
}

func (f *File) Mode() os.FileMode {
	return f.Info().Mode()
}

// main processes command line arguments, builds the context struct, and begins
// the process of walking the current working directory.
//
// Variable terminology used throughout this module:
//
// • dirName: the name of a directory, without a trailing separator
// • baseName: the relative name of a file, without a directory
// • path: the relative path to a specific file or directory, including both dirName and baseName
func main() {
	// Remove date/time from logging output
	log.SetFlags(0)
	rand.Seed(time.Now().UnixNano())

	if len(os.Args) != 3 {
		log.Fatal("Usage: find-replace FIND REPLACE")
	}

	find := os.Args[1]
	replace := os.Args[2]

	fr := findReplace{find: find, replace: replace}

	// Recursively explore the hierarchy depth first, rewrite files as needed,
	// and rename files last (after we don't have to revisit them).
	// path.filepath.WalkDir() won't work here because it walks files
	// alphabetically, breadth-first (and you'd be renaming files that you
	// haven't explored yet).

	fr.WalkDir(NewFile("."))
}

// Walks files in the directory given by dirName, which is a relative path to a
// directory. Calls HandleFile for each file it finds, if it's not ignored.
func (fr *findReplace) WalkDir(f *File) {
	// List the files in this directory.
	files, err := os.ReadDir(f.Path)
	if err != nil {
		log.Fatalf("Unable to read directory: %v", err)
	}

	for _, file := range files {
		if file.Name() != ".git" {
			fr.HandleFile(NewFile(f.Path + string(os.PathSeparator) + file.Name()))
		}
	}
}

// HandleFile immediately recurses depth-first into directories it finds,
// otherwise calls ReplaceContents for regular files. When either operation is
// complete, the file is renamed (if necessary) since no subsequent operations
// will need to access it again.
func (fr *findReplace) HandleFile(f *File) {
	// If file is a directory, recurse immediately (depth-first).
	if f.Info().IsDir() {
		fr.WalkDir(f)
	} else {
		// Replace the contents of regular files
		fr.ReplaceContents(f)
	}

	// Rename the file now that we're otherwise done with it
	fr.RenameFile(f)
}

// RenameFile renames a file if the destination file name does not already
// exist.
func (fr *findReplace) RenameFile(f *File) {
	newBaseName := strings.Replace(f.Base(), fr.find, fr.replace, -1)
	newPath := f.Dir() + string(os.PathSeparator) + newBaseName

	if f.Base() != newBaseName {
		if _, err := os.Stat(newPath); errors.Is(err, os.ErrNotExist) {
			log.Printf("Renaming %v to %v", f.Path, newBaseName)
			if err := os.Rename(f.Path, newPath); err != nil {
				log.Fatalf("Unable to rename %v to %v: %v", f.Path, newBaseName, err)
			}
		} else {
			log.Fatalf("Refusing to rename %v to %v: %v already exists", f.Path, newBaseName, newPath)
		}
	}
}

// Replaces the contents of the given file, using the find & replace values in
// context.
func (fr *findReplace) ReplaceContents(f *File) {
	// Find & replace the contents of file.
	content := readFile(f.Path)
	if util.IsText([]byte(content)) && strings.Contains(content, fr.find) {
		newContent := strings.Replace(content, fr.find, fr.replace, -1)
		writeFile(f, newContent)
	}
}

// readFile reads a file at the given path into a string.
func readFile(path string) string {
	f, err := os.Open(path)
	if err != nil {
		log.Fatalf("Unable to open %v: %v", path, err)
	}
	defer f.Close()
	builder := new(strings.Builder)
	if _, err := io.Copy(builder, f); err != nil {
		log.Fatalf("Failed to read %v to a string: %v", path, err)
	}
	return builder.String()
}

// writeFile atomically write content to file by writing it to a temporary file
// first, and then moving it to the destination, overwriting the original.
func writeFile(f *File, content string) {
	tempName := f.Dir() + string(os.PathSeparator) + randomString(20)
	if err := os.WriteFile(tempName, []byte(content), f.Mode()); err != nil {
		log.Fatalf("Error creating tempfile in %v: %v", f.Dir(), err)
	}

	log.Printf("Rewriting %v", f.Path)
	if err := os.Rename(tempName, f.Path); err != nil {
		log.Fatalf("Unable to atomically move temp file %v to %v: %v", tempName, f.Path, err)
	}
}

var characters = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789")

// randomString generates a random base-62 string of the given length (or returns an
// empty string).
func randomString(n int) string {
	if n <= 0 {
		return ""
	}
	b := make([]rune, n)
	for i := range b {
		b[i] = characters[rand.Intn(len(characters))]
	}
	return string(b)
}
