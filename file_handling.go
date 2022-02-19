package main

import (
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
	handle.Seek(0, 0)

	builder := new(strings.Builder)
	if _, err := io.Copy(builder, handle); err != nil {
		log.Fatalf("Failed to read %v to a string: %v", f.Path, err)
	}
	return builder.String()
}

// Write content to file atomically, by writing it to a temporary file first,
// and then moving it to the destination, overwriting the original.
func (f *File) Write(content string) {
	tempName := filepath.Join(f.Dir(), RandomString(20))
	if err := os.WriteFile(tempName, []byte(content), f.Mode()); err != nil {
		log.Fatalf("Error creating tempfile in %v: %v", f.Dir(), err)
	}

	log.Printf("Rewriting %v", f.Path)
	if err := os.Rename(tempName, f.Path); err != nil {
		log.Fatalf("Unable to atomically move temp file %v to %v: %v", tempName, f.Path, err)
	}
}
