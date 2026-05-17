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

func isSkippedTempName(name string) bool {
	return strings.HasPrefix(name, ".find-replace.tmp.")
}

// StreamFindReplace rewrites the file in place when find occurs. Binary or
// non-text files are skipped. Memory use is bounded by streamBufferSize.
func (f *File) StreamFindReplace(find, replace string) {
	if len(find) == 0 {
		return
	}

	in, err := os.Open(f.Path)
	if err != nil {
		log.Fatalf("Unable to open %v: %v", f.Path, err)
	}
	defer in.Close()

	var head [1024]byte
	n, err := in.Read(head[:])
	if err != nil && err != io.EOF {
		log.Fatalf("Unable to read %v: %v", f.Path, err)
	}
	if n == 0 || !util.IsText(head[:n]) {
		return
	}
	if _, err := in.Seek(0, io.SeekStart); err != nil {
		log.Fatalf("Failed to seek back to beginning of %v: %v", f.Path, err)
	}

	tempName := filepath.Join(f.Dir(), fmt.Sprintf(".find-replace.tmp.%d.%s", os.Getpid(), RandomString(8)))
	out, err := os.OpenFile(tempName, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, f.Mode())
	if err != nil {
		log.Fatalf("Error creating tempfile in %v: %v", f.Dir(), err)
	}

	changed, err := streamReplace(in, out, []byte(find), []byte(replace))
	closeErr := out.Close()
	if err != nil {
		_ = os.Remove(tempName)
		log.Fatalf("Failed to rewrite %v: %v", f.Path, err)
	}
	if closeErr != nil {
		_ = os.Remove(tempName)
		log.Fatalf("Failed to close tempfile for %v: %v", f.Path, closeErr)
	}
	if !changed {
		_ = os.Remove(tempName)
		return
	}

	log.Printf("Rewriting %v", f.Path)
	if err := os.Rename(tempName, f.Path); err != nil {
		_ = os.Remove(tempName)
		log.Fatalf("Unable to atomically move temp file %v to %v: %v", tempName, f.Path, err)
	}
}

// Read the file into a string (used by tests).
func (f *File) Read() string {
	handle, err := os.Open(f.Path)
	if err != nil {
		log.Fatalf("Unable to open %v: %v", f.Path, err)
	}
	defer handle.Close()

	var buf [1024]byte
	n, err := handle.Read(buf[0:])
	if err != nil || !util.IsText(buf[0:n]) {
		return ""
	}

	if _, err := handle.Seek(0, io.SeekStart); err != nil {
		log.Fatalf("Failed to seek back to beginning of %v: %v", f.Path, err)
	}

	builder := new(strings.Builder)
	if _, err := io.Copy(builder, handle); err != nil {
		log.Fatalf("Failed to read %v to a string: %v", f.Path, err)
	}
	return builder.String()
}

// Write content to file atomically (used by tests).
func (f *File) Write(content string) {
	tempName := filepath.Join(f.Dir(), fmt.Sprintf(".find-replace.tmp.%d.%s", os.Getpid(), RandomString(8)))
	if err := os.WriteFile(tempName, []byte(content), f.Mode()); err != nil {
		log.Fatalf("Error creating tempfile in %v: %v", f.Dir(), err)
	}

	log.Printf("Rewriting %v", f.Path)
	if err := os.Rename(tempName, f.Path); err != nil {
		log.Fatalf("Unable to atomically move temp file %v to %v: %v", tempName, f.Path, err)
	}
}
