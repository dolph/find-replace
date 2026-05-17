package main

import (
	"errors"
	"log"
	"math/rand"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// findReplace is a struct used to provide context to all find & replace
// operations, including the strings to search for & replace.
type findReplace struct {
	find    string
	replace string
	workers chan struct{}
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

	fr := findReplace{
		find:    find,
		replace: replace,
		workers: make(chan struct{}, workerLimit()),
	}

	// Recursively explore the hierarchy depth first, rewrite files as needed,
	// and rename files last (after we don't have to revisit them).
	// path.filepath.WalkDir() won't work here because it walks files
	// alphabetically, breadth-first (and you'd be renaming files that you
	// haven't explored yet).

	fr.WalkDir(NewFile("."))
}

func (fr *findReplace) acquireWorker() {
	fr.workers <- struct{}{}
}

func (fr *findReplace) releaseWorker() {
	<-fr.workers
}

func (fr *findReplace) processFile(f *File) {
	fr.acquireWorker()
	defer fr.releaseWorker()
	fr.ReplaceContents(f)
	fr.RenameFile(f)
}

// Walks files in the directory given by dirName, which is a relative path to a
// directory. Calls HandleFile for each file it finds, if it's not ignored.
func (fr *findReplace) WalkDir(f *File) {
	files, err := os.ReadDir(f.Path)
	if err != nil {
		log.Fatalf("Unable to read directory: %v", err)
	}

	var wg sync.WaitGroup
	for _, entry := range files {
		childFile := NewFile(filepath.Join(f.Path, entry.Name()))
		if entry.IsDir() {
			if childFile.Base() == ".git" {
				continue
			}
			fr.WalkDir(childFile)
			fr.RenameFile(childFile)
			continue
		}

		wg.Add(1)
		go func(file *File) {
			defer wg.Done()
			fr.processFile(file)
		}(childFile)
	}

	wg.Wait()
}

// HandleFile supports unit tests and mirrors the walk order: directories
// synchronously, files through the bounded worker pool.
func (fr *findReplace) HandleFile(f *File) {
	if f.Info().IsDir() {
		if f.Base() == ".git" {
			return
		}
		fr.WalkDir(f)
		fr.RenameFile(f)
		return
	}
	fr.processFile(f)
}

// RenameFile renames a file if the destination file name does not already
// exist.
func (fr *findReplace) RenameFile(f *File) {
	newBaseName := strings.Replace(f.Base(), fr.find, fr.replace, -1)
	newPath := filepath.Join(f.Dir(), newBaseName)

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
	// Find & replace the contents of text files. Binary-looking files return
	// an empty string and will be skipped here.
	content := f.Read()
	if strings.Contains(content, fr.find) {
		newContent := strings.Replace(content, fr.find, fr.replace, -1)
		f.Write(newContent)
	}
}
