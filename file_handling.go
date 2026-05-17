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
	Path    string
	info    os.FileInfo
	onError func(string, ...interface{})
}

func NewFile(path string) *File {
	absPath, err := filepath.Abs(path)
	if err != nil {
		log.Fatalf("Unable to resolve absolute path of %v: %v", path, err)
	}
	return &File{Path: absPath}
}

func (f *File) logError(format string, args ...interface{}) {
	if f.onError != nil {
		f.onError(format, args...)
		return
	}
	log.Fatalf(format, args...)
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
			f.logError("Failed to stat %v: %v", f.Path, err)
			return nil
		}
		f.info = stat
	}
	return f.info
}

func (f *File) Mode() os.FileMode {
	return f.Info().Mode()
}

// Read returns file contents and whether the read succeeded. Binary-looking files return "", true.
func (f *File) Read() (string, bool) {
	handle, err := os.Open(f.Path)
	if err != nil {
		f.logError("Unable to open %v: %v", f.Path, err)
		return "", false
	}
	defer handle.Close()

	var buf [1024]byte
	n, err := handle.Read(buf[0:])
	if err != nil || !util.IsText(buf[0:n]) {
		return "", true
	}

	if _, err := handle.Seek(0, io.SeekStart); err != nil {
		f.logError("Failed to seek back to beginning of %v: %v", f.Path, err)
		return "", false
	}

	builder := new(strings.Builder)
	if _, err := io.Copy(builder, handle); err != nil {
		f.logError("Failed to read %v to a string: %v", f.Path, err)
		return "", false
	}
	return builder.String(), true
}

// Write content to file atomically, by writing it to a temporary file first,
// and then moving it to the destination, overwriting the original.
func (f *File) Write(content string) {
	tempName := filepath.Join(f.Dir(), RandomString(20))
	if err := os.WriteFile(tempName, []byte(content), f.Mode()); err != nil {
		f.logError("Error creating tempfile in %v: %v", f.Dir(), err)
		return
	}

	log.Printf("Rewriting %v", f.Path)
	if err := os.Rename(tempName, f.Path); err != nil {
		f.logError("Unable to atomically move temp file %v to %v: %v", tempName, f.Path, err)
	}
}
