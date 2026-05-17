package main

import (
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"

	"golang.org/x/tools/godoc/util"
)

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

// Read the file into a string.
func (f *File) Read() string {
	handle, err := os.Open(f.Path)
	if err != nil {
		log.Fatalf("Unable to open %v: %v", f.Path, err)
	}
	defer handle.Close()

	// Check if the file looks like text before reading the entire file.
	var buf [1024]byte
	n, err := handle.Read(buf[0:])
	if err != nil || !util.IsText(buf[0:n]) {
		return ""
	}

	// Reset file handle so we can read the entire file.
	if _, err := handle.Seek(0, io.SeekStart); err != nil {
		log.Fatalf("Failed to seek back to beginning of %v: %v", f.Path, err)
	}

	builder := new(strings.Builder)
	if _, err := io.Copy(builder, handle); err != nil {
		log.Fatalf("Failed to read %v to a string: %v", f.Path, err)
	}
	return builder.String()
}

// writeContent writes bytes via a temp file in the same directory, then renames
// over the target. The temp file is removed if rename fails.
func (f *File) writeContent(content []byte) error {
	tempName := filepath.Join(f.Dir(), RandomString(20))
	if err := os.WriteFile(tempName, content, f.Mode()); err != nil {
		return fmt.Errorf("error creating tempfile in %v: %w", f.Dir(), err)
	}

	log.Printf("Rewriting %v", f.Path)
	if err := os.Rename(tempName, f.Path); err != nil {
		_ = os.Remove(tempName)
		return fmt.Errorf("unable to atomically move temp file %v to %v: %w", tempName, f.Path, err)
	}
	return nil
}

// Write content to file atomically, by writing it to a temporary file first,
// and then moving it to the destination, overwriting the original.
func (f *File) Write(content string) {
	if err := f.writeContent([]byte(content)); err != nil {
		log.Fatalf("%v", err)
	}
}
