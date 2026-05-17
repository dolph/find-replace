package main

import (
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

type operationScope int

const (
	scopeAll operationScope = iota
	scopeContentOnly
	scopeRenameOnly
)

// findReplace is a struct used to provide context to all find & replace
// operations, including the strings to search for & replace.
type findReplace struct {
	find    string
	replace string
	scope   operationScope
}

// main processes command line arguments, builds the context struct, and begins
// the process of walking the current working directory.
//
// Variable terminology used throughout this module:
//
// • dirName: the name of a directory, without a trailing separator
// • baseName: the relative name of a file, without a directory
// • path: the relative path to a specific file or directory, including both dirName and baseName
func parseArgs(args []string) (findReplace, error) {
	scope := scopeAll
	i := 0
	for i < len(args) && strings.HasPrefix(args[i], "-") {
		switch args[i] {
		case "--content-only":
			if scope != scopeAll {
				return findReplace{}, fmt.Errorf("cannot pass both --content-only and --rename-only")
			}
			scope = scopeContentOnly
		case "--rename-only":
			if scope != scopeAll {
				return findReplace{}, fmt.Errorf("cannot pass both --content-only and --rename-only")
			}
			scope = scopeRenameOnly
		default:
			return findReplace{}, fmt.Errorf("unknown flag: %s", args[i])
		}
		i++
	}

	rest := args[i:]
	if len(rest) != 2 {
		return findReplace{}, fmt.Errorf("usage: find-replace [--content-only | --rename-only] FIND REPLACE")
	}

	return findReplace{find: rest[0], replace: rest[1], scope: scope}, nil
}

func main() {
	// Remove date/time from logging output
	log.SetFlags(0)

	fr, err := parseArgs(os.Args[1:])
	if err != nil {
		log.Fatal(err)
	}

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
	var wg sync.WaitGroup

	// List the files in this directory.
	files, err := os.ReadDir(f.Path)
	if err != nil {
		log.Fatalf("Unable to read directory: %v", err)
	}

	for _, file := range files {
		childFile := NewFile(filepath.Join(f.Path, file.Name()))
		wg.Add(1)
		go func() {
			defer wg.Done()
			fr.HandleFile(childFile)
		}()
	}

	wg.Wait() // for (potentially recursive) calls to return
}

// HandleFile immediately recurses depth-first into directories it finds,
// otherwise calls ReplaceContents for regular files. When either operation is
// complete, the file is renamed (if necessary) since no subsequent operations
// will need to access it again.
func (fr *findReplace) HandleFile(f *File) {
	// If file is a directory, recurse immediately (depth-first).
	if f.Info().IsDir() {
		// Ignore certain directories
		if f.Base() == ".git" {
			return
		}
		fr.WalkDir(f)
	} else if fr.scope != scopeRenameOnly {
		// Replace the contents of regular files
		fr.ReplaceContents(f)
	}

	if fr.scope != scopeContentOnly {
		// Rename the file now that we're otherwise done with it
		fr.RenameFile(f)
	}
}

// RenameFile renames a file if the destination file name does not already
// exist.
func (fr *findReplace) RenameFile(f *File) {
	newBaseName := strings.ReplaceAll(f.Base(), fr.find, fr.replace)
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
		newContent := strings.ReplaceAll(content, fr.find, fr.replace)
		f.Write(newContent)
	}
}
