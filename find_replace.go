package main

import (
	"errors"
	"fmt"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"strings"
)

// findReplace carries the parameters and counters for a single run.
type findReplace struct {
	find    string
	replace string
	errors  int
}

// Skipped directory base names. Kept simple and easy to extend.
var skipDirs = map[string]struct{}{
	".git": {},
}

func main() {
	log.SetFlags(0)

	if len(os.Args) != 3 {
		log.Fatal("Usage: find-replace FIND REPLACE")
	}

	find := os.Args[1]
	replace := os.Args[2]

	if find == "" {
		log.Fatal("FIND must be non-empty")
	}
	if find == replace {
		// Nothing to do, but it's a user error worth flagging.
		log.Fatal("FIND and REPLACE are identical; nothing to do")
	}

	fr := findReplace{find: find, replace: replace}
	fr.WalkDir(NewFile("."))

	if fr.errors > 0 {
		os.Exit(1)
	}
}

// WalkDir traverses the directory tree rooted at f depth-first, rewriting file
// contents and renaming entries on the way back up. It runs single-threaded
// to keep memory and file-descriptor usage bounded and to avoid races between
// concurrent renames in the same parent directory.
func (fr *findReplace) WalkDir(f *File) {
	entries, err := os.ReadDir(f.Path)
	if err != nil {
		fr.recordErr(fmt.Errorf("read directory %s: %w", f.Path, err))
		return
	}

	for _, entry := range entries {
		// Use the DirEntry directly instead of stat'ing each child; this
		// avoids both a redundant syscall and the symlink-following side
		// effect of os.Stat. (Issues #2, #13.)
		fr.handleEntry(f, entry)
	}
}

func (fr *findReplace) handleEntry(parent *File, entry fs.DirEntry) {
	name := entry.Name()
	mode := entry.Type()

	// Symlinks are skipped entirely (issue #2). We never follow them and we
	// don't rewrite or rename them — renaming a symlink only renames the
	// link itself, which is harmless, but skipping is clearer and avoids
	// subtle interactions with the rename phase below.
	if mode&os.ModeSymlink != 0 {
		return
	}

	child := newChildFile(parent.Path, name)

	if entry.IsDir() {
		if _, skip := skipDirs[name]; skip {
			return
		}
		fr.WalkDir(child)
	} else if mode.IsRegular() {
		fr.rewriteContents(child, entry)
	} else {
		// Devices, named pipes, sockets, etc.: leave alone.
		return
	}

	fr.RenameFile(child)
}

// rewriteContents replaces fr.find with fr.replace in the file's bytes,
// streaming through a bounded buffer. The original file's mode is preserved.
func (fr *findReplace) rewriteContents(f *File, entry fs.DirEntry) {
	info, err := entry.Info()
	if err != nil {
		// Race with another process; treat as missing and continue.
		if errors.Is(err, fs.ErrNotExist) {
			return
		}
		fr.recordErr(fmt.Errorf("stat %s: %w", f.Path, err))
		return
	}

	changed, err := rewriteFile(f.Path, []byte(fr.find), []byte(fr.replace), info.Mode())
	if err != nil {
		fr.recordErr(fmt.Errorf("rewrite %s: %w", f.Path, err))
		return
	}
	if changed {
		log.Printf("Rewriting %v", f.Path)
	}
}

// RenameFile renames f to the same path with fr.find replaced by fr.replace
// in the basename, only if the destination does not already exist. Uses
// renameNoReplace to close the TOCTOU window in the existence check.
func (fr *findReplace) RenameFile(f *File) {
	newBase := strings.Replace(f.Base(), fr.find, fr.replace, -1)
	if newBase == f.Base() {
		return
	}
	newPath := filepath.Join(f.Dir(), newBase)

	if err := renameNoReplace(f.Path, newPath); err != nil {
		if errors.Is(err, os.ErrExist) {
			fr.recordErr(fmt.Errorf("refusing to rename %s to %s: %s already exists",
				f.Path, newBase, newPath))
			return
		}
		fr.recordErr(fmt.Errorf("rename %s to %s: %w", f.Path, newBase, err))
		return
	}
	log.Printf("Renaming %v to %v", f.Path, newBase)
}

func (fr *findReplace) recordErr(err error) {
	fr.errors++
	log.Printf("error: %v", err)
}
