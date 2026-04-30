package main

import (
	"log"
	"path/filepath"
)

// File holds a single absolute path. It exists primarily to keep call sites
// readable (`f.Base()` reads better than `filepath.Base(path)`) and is
// otherwise a thin wrapper around a string.
type File struct {
	Path string
}

// NewFile resolves path to an absolute path and returns a *File for it. Used
// for the initial root passed in from main().
func NewFile(path string) *File {
	if filepath.IsAbs(path) {
		return &File{Path: filepath.Clean(path)}
	}
	abs, err := filepath.Abs(path)
	if err != nil {
		log.Fatalf("Unable to resolve absolute path of %v: %v", path, err)
	}
	return &File{Path: abs}
}

// newChildFile builds a *File for the given child of an already-absolute
// parent path, skipping the redundant filepath.Abs call performed by NewFile.
func newChildFile(parentAbs, name string) *File {
	return &File{Path: filepath.Join(parentAbs, name)}
}

// Base returns the base name of f's path.
func (f *File) Base() string {
	return filepath.Base(f.Path)
}

// Dir returns the directory of f's path.
func (f *File) Dir() string {
	return filepath.Dir(f.Path)
}
