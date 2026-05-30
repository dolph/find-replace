package main

import (
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"
	"unicode/utf8"
	"bytes"
)

type File struct {
	Path string
	info os.FileInfo
}

// NewFile resolves path to an absolute path and wraps it in a *File. It
// returns an error if the working directory cannot be determined (the only
// failure mode of filepath.Abs).
func NewFile(path string) (*File, error) {
	absPath, err := filepath.Abs(path)
	if err != nil {
		return nil, fmt.Errorf("resolve absolute path of %v: %w", path, err)
	}
	return &File{Path: absPath}, nil
}

func (f *File) Base() string {
	return filepath.Base(f.Path)
}

func (f *File) Dir() string {
	return filepath.Dir(f.Path)
}

// Info lazily stats the file and caches the result. It returns an error if
// the underlying os.Stat fails.
func (f *File) Info() (os.FileInfo, error) {
	if f.info == nil {
		stat, err := os.Stat(f.Path)
		if err != nil {
			return nil, fmt.Errorf("stat %v: %w", f.Path, err)
		}
		f.info = stat
	}
	return f.info, nil
}

// Mode returns the cached mode bits. It is only safe to call after Info() has
// succeeded; callers that have a *File handed to them by the walker can rely
// on that precondition because the walker calls Info() before dispatching.
func (f *File) Mode() (os.FileMode, error) {
	info, err := f.Info()
	if err != nil {
		return 0, err
	}
	return info.Mode(), nil
}

// Read reads the file into a string, or returns the empty string for binary
// files (NUL bytes or invalid UTF-8). An error indicates the file could not be
// opened or fully read; the caller should log-and-skip rather than abort.
func (f *File) Read() (string, error) {
	handle, err := os.Open(f.Path)
	if err != nil {
		return "", fmt.Errorf("open %v: %w", f.Path, err)
	}
	defer handle.Close()

	var buf [1024]byte
	n, err := handle.Read(buf[:])
	if err != nil && err != io.EOF {
		return "", fmt.Errorf("read %v: %w", f.Path, err)
	}
	if n == 0 {
		return "", nil
	}
	if !isTextBytes(buf[:n]) {
		return "", nil
	}
	if err == io.EOF {
		return string(buf[:n]), nil
	}

	builder := new(strings.Builder)
	if _, wErr := builder.Write(buf[:n]); wErr != nil {
		return "", fmt.Errorf("read %v: %w", f.Path, wErr)
	}

	chunk := make([]byte, 32*1024)
	for {
		readN, readErr := handle.Read(chunk)
		if readN > 0 {
			if bytes.IndexByte(chunk[:readN], 0) >= 0 {
				return "", nil
			}
			if !utf8.Valid(chunk[:readN]) {
				return "", nil
			}
			if _, wErr := builder.Write(chunk[:readN]); wErr != nil {
				return "", fmt.Errorf("read %v: %w", f.Path, wErr)
			}
		}
		if readErr == io.EOF {
			break
		}
		if readErr != nil {
			return "", fmt.Errorf("read %v: %w", f.Path, readErr)
		}
	}
	return builder.String(), nil
}

// Write atomically replaces the file with content, via a temp file + rename.
// A deferred os.Remove(tempName) ensures the temp file is cleaned up if any
// step after its creation fails (including the rename); on success the remove
// is a no-op because the file has already been renamed away.
func (f *File) Write(content string) error {
	mode, err := f.Mode()
	if err != nil {
		return err
	}

	tempName := filepath.Join(f.Dir(), RandomString(20))
	if err := os.WriteFile(tempName, []byte(content), mode); err != nil {
		return fmt.Errorf("create tempfile in %v: %w", f.Dir(), err)
	}
	// Make sure the temp file is removed if the rename below fails. On
	// success, the rename has already moved the file to f.Path so this is
	// a no-op (we deliberately ignore the not-exist error).
	defer os.Remove(tempName)

	log.Printf("Rewriting %v", f.Path)
	if err := os.Rename(tempName, f.Path); err != nil {
		return fmt.Errorf("atomically move temp file %v to %v: %w", tempName, f.Path, err)
	}
	return nil
}
