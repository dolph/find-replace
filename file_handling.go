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
// files. An error indicates the file could not be opened or fully read; the
// caller should log-and-skip rather than abort.
func (f *File) Read() (string, error) {
	handle, err := os.Open(f.Path)
	if err != nil {
		return "", fmt.Errorf("open %v: %w", f.Path, err)
	}
	defer handle.Close()

	// Check if the file looks like text before reading the entire file.
	var buf [1024]byte
	n, err := handle.Read(buf[0:])
	if err != nil || !util.IsText(buf[0:n]) {
		return "", nil
	}

	// Reset file handle so we can read the entire file.
	if _, err := handle.Seek(0, io.SeekStart); err != nil {
		return "", fmt.Errorf("seek to start of %v: %w", f.Path, err)
	}

	builder := new(strings.Builder)
	if _, err := io.Copy(builder, handle); err != nil {
		return "", fmt.Errorf("read %v: %w", f.Path, err)
	}
	return builder.String(), nil
}

// tempFilePrefix is the prefix used for the atomic-rewrite temp file
// created in the target's directory. The leading dot keeps the file hidden
// from `ls`, and the shared prefix gives a future "skip orphan temp files
// on a subsequent walk" pass (issue #21) a stable glob to filter on.
const tempFilePrefix = ".find-replace."

// Write atomically replaces the file with content, via a temp file + rename.
//
// The temp file is created with os.CreateTemp, which opens with
// O_RDWR|O_CREATE|O_EXCL|0600 and a CSPRNG-quality random suffix. The
// combination defends against the symlink/race attack in issue #3: a
// co-resident attacker cannot predict the name, and even if they could,
// O_EXCL would refuse to clobber an attacker-pre-created path (which
// os.WriteFile + O_CREATE used to silently follow as a symlink).
//
// A deferred os.Remove(tempName) ensures the temp file is cleaned up if any
// step after its creation fails (including the rename); on success the
// remove is a no-op because the file has already been renamed away.
func (f *File) Write(content string) error {
	mode, err := f.Mode()
	if err != nil {
		return err
	}

	tmp, err := os.CreateTemp(f.Dir(), tempFilePrefix+"*")
	if err != nil {
		return fmt.Errorf("create tempfile in %v: %w", f.Dir(), err)
	}
	tempName := tmp.Name()
	// Make sure the temp file is removed if any step below fails. On
	// success, the rename has already moved the file to f.Path so this is
	// a no-op (we deliberately ignore the not-exist error).
	defer os.Remove(tempName)

	if _, err := tmp.WriteString(content); err != nil {
		tmp.Close()
		return fmt.Errorf("write tempfile %v: %w", tempName, err)
	}
	// Preserve the original file's permission bits. os.CreateTemp creates
	// the file 0600; without this Chmod the rewrite would silently downgrade
	// permissions. Setuid/setgid/sticky bits are intentionally NOT preserved
	// in this PR — see issue #18 for the separate threat model.
	if err := tmp.Chmod(mode.Perm()); err != nil {
		tmp.Close()
		return fmt.Errorf("chmod tempfile %v: %w", tempName, err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("close tempfile %v: %w", tempName, err)
	}

	log.Printf("Rewriting %v", f.Path)
	if err := os.Rename(tempName, f.Path); err != nil {
		return fmt.Errorf("atomically move temp file %v to %v: %w", tempName, f.Path, err)
	}
	return nil
}
