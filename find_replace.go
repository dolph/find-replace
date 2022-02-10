package main

import (
	"errors"
	"io"
	"io/fs"
	"log"
	"math/rand"
	"os"
	"strings"
	"time"

	"golang.org/x/tools/godoc/util"
)

type findReplace struct {
	find    string
	replace string
}

// ** Variable terminology **
// dirName: the name of a directory, without a trailing separator
// baseName: the relative name of a file, without a directory
// path: the relative path to a specific file or directory, including both dirName and baseName

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
	fr.WalkDir(".")
}

func (fr *findReplace) WalkDir(dirName string) {
	// List the files in this directory.
	files, err := os.ReadDir(dirName)
	if err != nil {
		log.Fatalf("Unable to read directory: %v", err)
	}

	for _, file := range files {
		if file.Name() != ".git" {
			fr.HandleFile(dirName, file)
		}
	}
}

func (fr *findReplace) HandleFile(dirName string, file fs.DirEntry) {
	// If file is a directory, recurse immediately (depth-first).
	if file.IsDir() {
		fr.WalkDir(dirName + string(os.PathSeparator) + file.Name())
	} else {
		// Replace the contents of regular files
		fr.ReplaceContents(dirName, file)
	}

	// Rename the file now that we're otherwise done with it
	fr.RenameFile(dirName, file)
}

// Renames a file if the destination does not already exist.
func (fr *findReplace) RenameFile(dirName string, file fs.DirEntry) {
	oldPath := dirName + string(os.PathSeparator) + file.Name()
	newBaseName := strings.Replace(file.Name(), fr.find, fr.replace, -1)
	newPath := dirName + string(os.PathSeparator) + newBaseName

	if file.Name() != newBaseName {
		if _, err := os.Stat(newPath); errors.Is(err, os.ErrNotExist) {
			log.Printf("Renaming %v to %v", oldPath, newBaseName)
			if err := os.Rename(oldPath, newPath); err != nil {
				log.Fatalf("Unable to rename %v to %v: %v", oldPath, newBaseName, err)
			}
		} else {
			log.Fatalf("Refusing to rename %v to %v: %v already exists", oldPath, newBaseName, newPath)
		}
	}
}

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

// Atomically write file.
func writeFile(dirName string, file fs.DirEntry, content string) {
	path := dirName + string(os.PathSeparator) + file.Name()

	info, err := os.Stat(path)
	if err != nil {
		log.Fatalf("Error getting stats on %v: %v", path, err)
	}

	tempName := dirName + string(os.PathSeparator) + randomString(20)
	if err := os.WriteFile(tempName, []byte(content), info.Mode()); err != nil {
		log.Fatalf("Error creating tempfile in %v: %v", dirName, err)
	}

	log.Printf("Rewriting %v", path)
	os.Rename(tempName, path)
}

func (fr *findReplace) ReplaceContents(dirName string, file fs.DirEntry) {
	path := dirName + string(os.PathSeparator) + file.Name()

	// Find & replace the contents of file.
	content := readFile(path)
	if util.IsText([]byte(content)) && strings.Contains(content, fr.find) {
		newContent := strings.Replace(content, fr.find, fr.replace, -1)
		writeFile(dirName, file, newContent)
	}
}

var characters = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789")

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
