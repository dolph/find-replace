package main

import (
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

// findReplace is a struct used to provide context to all find & replace
// operations, including the strings to search for & replace.
type findReplace struct {
	find    string
	replace string

	// errs accumulates non-fatal errors that occurred during a walk. The
	// walker logs each error at the point of failure (preserving the
	// operator-visible UX) and appends it here so main can surface a
	// non-zero exit code at the end.
	errs errAccumulator
}

// errAccumulator is a tiny thread-safe collector for errors that occur in
// concurrent walker goroutines. It is intentionally small: just enough to
// preserve "log everything, exit non-zero if anything failed" semantics
// without committing the codebase to a particular concurrency primitive
// (see issue #7 for the eventual bounded worker pool).
type errAccumulator struct {
	mu   sync.Mutex
	errs []error
}

// add records err. A nil err is ignored so callers can write
// `acc.add(fn())` without a guard.
func (a *errAccumulator) add(err error) {
	if err == nil {
		return
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	a.errs = append(a.errs, err)
}

// err returns the accumulated errors joined with errors.Join, or nil if
// nothing was recorded. The returned error is safe to unwrap with errors.Is
// / errors.As over every accumulated error.
func (a *errAccumulator) err() error {
	a.mu.Lock()
	defer a.mu.Unlock()
	if len(a.errs) == 0 {
		return nil
	}
	return errors.Join(a.errs...)
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
	os.Exit(run(os.Args, os.Stderr))
}

// run is the testable body of main. It returns the process exit code: 0 on
// clean success, 1 if argument parsing failed or any traversal error was
// recorded. Output documented in the README (Renaming/Rewriting lines) still
// goes to log.Default(); usage and aggregated error summaries go to stderr.
func run(args []string, stderr io.Writer) int {
	// Remove date/time from logging output.
	log.SetFlags(0)

	if len(args) != 3 {
		fmt.Fprintln(stderr, "Usage: find-replace FIND REPLACE")
		return 1
	}

	fr := findReplace{find: args[1], replace: args[2]}

	// Recursively explore the hierarchy depth first, rewrite files as needed,
	// and rename files last (after we don't have to revisit them).
	// filepath.WalkDir won't work here because it walks files alphabetically,
	// breadth-first (and you'd be renaming files that you haven't explored
	// yet).
	root, err := NewFile(".")
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	fr.WalkDir(root)

	if err := fr.errs.err(); err != nil {
		// Each individual error has already been printed at the point of
		// failure; the join here is for completeness in case a caller is
		// scraping stderr.
		fmt.Fprintln(stderr, err)
		return 1
	}
	return 0
}

// WalkDir lists files in the directory given by f and dispatches each child
// to HandleFile in its own goroutine. Per-child errors are logged at their
// failure site and recorded on fr so main can surface a non-zero exit code.
// A failure to read the directory itself is recorded and returned to the
// caller, but does not abort the rest of the walk in any other subtree.
func (fr *findReplace) WalkDir(f *File) {
	var wg sync.WaitGroup

	// List the files in this directory.
	files, err := os.ReadDir(f.Path)
	if err != nil {
		wrapped := fmt.Errorf("read directory %v: %w", f.Path, err)
		log.Print(wrapped)
		fr.errs.add(wrapped)
		return
	}

	for _, file := range files {
		if strings.HasPrefix(file.Name(), ".find-replace-") {
			continue
		}
		childPath := filepath.Join(f.Path, file.Name())
		childFile, err := NewFile(childPath)
		if err != nil {
			log.Print(err)
			fr.errs.add(err)
			continue
		}
		wg.Add(1)
		go func() {
			defer wg.Done()
			if err := fr.HandleFile(childFile); err != nil {
				log.Print(err)
				fr.errs.add(err)
			}
		}()
	}

	wg.Wait() // for (potentially recursive) calls to return
}

// HandleFile immediately recurses depth-first into directories it finds,
// otherwise calls ReplaceContents for regular files. When either operation is
// complete, the file is renamed (if necessary) since no subsequent operations
// will need to access it again. Errors from ReplaceContents are not fatal to
// the rename step; the failure is returned so the walker can log it and
// continue with siblings.
func (fr *findReplace) HandleFile(f *File) error {
	info, err := f.Info()
	if err != nil {
		return err
	}

	if strings.HasPrefix(f.Base(), ".find-replace-") {
		return nil
	}

	// If file is a directory, recurse immediately (depth-first).
	if info.IsDir() {
		// Ignore certain directories
		if f.Base() == ".git" {
			return nil
		}
		fr.WalkDir(f)
	} else {
		// Replace the contents of regular files.
		if err := fr.ReplaceContents(f); err != nil {
			return err
		}
	}

	// Rename the file now that we're otherwise done with it.
	return fr.RenameFile(f)
}

// RenameFile renames f to its post-replacement name if (a) the name actually
// changes and (b) no file already exists at the destination. It returns an
// error if the destination is occupied or if the os.Rename itself fails.
func (fr *findReplace) RenameFile(f *File) error {
	newBaseName := strings.ReplaceAll(f.Base(), fr.find, fr.replace)
	if f.Base() == newBaseName {
		return nil
	}

	newPath := filepath.Join(f.Dir(), newBaseName)
	if _, err := os.Stat(newPath); err == nil {
		return fmt.Errorf("refusing to rename %v to %v: %v already exists", f.Path, newBaseName, newPath)
	} else if !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("stat rename destination %v: %w", newPath, err)
	}

	log.Printf("Renaming %v to %v", f.Path, newBaseName)
	if err := os.Rename(f.Path, newPath); err != nil {
		return fmt.Errorf("rename %v to %v: %w", f.Path, newBaseName, err)
	}
	return nil
}

// ReplaceContents rewrites the file at f if its contents contain the find
// string. Binary-looking files (where Read returns "") are skipped silently.
func (fr *findReplace) ReplaceContents(f *File) error {
	content, err := f.Read()
	if err != nil {
		return err
	}
	if !strings.Contains(content, fr.find) {
		return nil
	}
	newContent := strings.ReplaceAll(content, fr.find, fr.replace)
	return f.Write(newContent)
}
