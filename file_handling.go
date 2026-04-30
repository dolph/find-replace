package main

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"unicode/utf8"
)

// sniffSize is the number of leading bytes inspected when deciding whether a
// file looks textual. Matches the historical behavior.
const sniffSize = 1024

// rewriteBufSize is the working buffer size used by the streaming rewriter.
// Sized to balance syscall count against per-call memory.
const rewriteBufSize = 64 * 1024

// looksBinary reports whether the given prefix appears to be binary content.
// A file is considered binary if it contains a NUL byte or an invalid UTF-8
// sequence in its sniffed prefix. Empty content is treated as text (a no-op
// rewrite is harmless).
func looksBinary(prefix []byte) bool {
	if bytes.IndexByte(prefix, 0) >= 0 {
		return true
	}
	return !utf8.Valid(prefix)
}

// renameNoReplace renames src to dst, returning os.ErrExist if dst already
// exists. Implemented as a hardlink-then-remove so the existence check and
// the directory-entry creation are a single atomic step. Falls back to a
// best-effort exists-check + Rename when hardlinks are not supported (for
// example renaming across filesystems or onto filesystems without link
// support).
func renameNoReplace(src, dst string) error {
	err := os.Link(src, dst)
	if err == nil {
		// Link succeeded, source can be unlinked.
		if rmErr := os.Remove(src); rmErr != nil {
			// Try to undo the link so we don't end up with two copies.
			_ = os.Remove(dst)
			return rmErr
		}
		return nil
	}
	if errors.Is(err, os.ErrExist) {
		return err
	}
	// Hardlinks unsupported (EXDEV across filesystems, EPERM on certain
	// filesystems). Fall back to a TOCTOU-prone but functional rename.
	if _, statErr := os.Lstat(dst); statErr == nil {
		return os.ErrExist
	}
	return os.Rename(src, dst)
}

// rewriteFile rewrites the contents of path, replacing every occurrence of
// find with replace. Returns true if the file was modified. The rewrite is
// atomic with respect to readers: a temp file is written under O_EXCL in the
// same directory and renamed over the original on success.
//
// Files that look binary in their first sniffSize bytes are skipped. Files
// that contain no occurrences of find are also skipped (no temp file is
// written).
func rewriteFile(path string, find, replace []byte, mode os.FileMode) (changed bool, err error) {
	// We have to know whether the content needs rewriting before we create a
	// temp file (otherwise we'd thrash the filesystem on every no-op file).
	// Read the prefix, sniff it, and remember it so we don't have to seek
	// back later.
	in, err := os.Open(path)
	if err != nil {
		return false, err
	}
	defer in.Close()

	prefix := make([]byte, sniffSize)
	n, err := io.ReadFull(in, prefix)
	prefix = prefix[:n]
	switch {
	case errors.Is(err, io.EOF), errors.Is(err, io.ErrUnexpectedEOF):
		// Short file; prefix contains the entire content.
		err = nil
	case err != nil:
		return false, err
	}
	if looksBinary(prefix) {
		return false, nil
	}

	// Cheap pre-check: if the file is entirely contained in our prefix and
	// it doesn't contain `find`, there's nothing to do. The streaming path
	// below would also detect this, but it would still create a temp file.
	// For larger files we let the streaming pass discover "no match" by
	// observing that no replacement was performed.
	if n < sniffSize && !bytes.Contains(prefix, find) {
		return false, nil
	}

	// Atomic write: create a temp file with O_EXCL via os.CreateTemp, stream
	// the rewrite into it, fsync, rename over the original. If anything
	// fails, the deferred remove cleans up the temp file.
	tmp, err := os.CreateTemp(filepath.Dir(path), ".find-replace-*")
	if err != nil {
		return false, err
	}
	tmpName := tmp.Name()
	// Always attempt to clean up the temp file. A successful rename consumes
	// the path, so the remove becomes a harmless no-op in that case.
	cleanup := true
	defer func() {
		if cleanup {
			_ = os.Remove(tmpName)
		}
	}()

	// Restore the original mode on the temp file (CreateTemp uses 0600).
	if err = os.Chmod(tmpName, mode.Perm()); err != nil {
		_ = tmp.Close()
		return false, err
	}

	// Stream the rest of the file, prepending the prefix we already read.
	rest := io.MultiReader(bytes.NewReader(prefix), in)
	wrote, err := streamReplace(tmp, rest, find, replace)
	if err != nil {
		_ = tmp.Close()
		return false, err
	}
	if err = tmp.Sync(); err != nil {
		_ = tmp.Close()
		return false, err
	}
	if err = tmp.Close(); err != nil {
		return false, err
	}
	if !wrote.changed {
		// Nothing actually replaced; leave the original alone.
		return false, nil
	}

	if err = os.Rename(tmpName, path); err != nil {
		return false, fmt.Errorf("rename %s -> %s: %w", tmpName, path, err)
	}
	cleanup = false
	return true, nil
}

type rewriteStats struct {
	changed bool
}

// streamReplace copies r to w, replacing every occurrence of find with
// replace. Memory usage is bounded by the size of the working buffer plus the
// length of `find`. Returns rewriteStats.changed=true if at least one
// replacement was made.
func streamReplace(w io.Writer, r io.Reader, find, replace []byte) (rewriteStats, error) {
	var stats rewriteStats
	if len(find) == 0 {
		// Defensive: callers should reject empty find before this point.
		// Behave as a plain copy to avoid pathological output.
		_, err := io.Copy(w, r)
		return stats, err
	}

	// Buffer is sized so that a full `find` plus a non-trivial chunk fits
	// after we carry up to (len(find)-1) bytes from the previous iteration.
	bufSize := rewriteBufSize
	if bufSize < 2*len(find) {
		bufSize = 2 * len(find)
	}
	buf := make([]byte, bufSize)
	keep := 0
	for {
		n, readErr := io.ReadFull(r, buf[keep:])
		end := keep + n
		eof := errors.Is(readErr, io.EOF) || errors.Is(readErr, io.ErrUnexpectedEOF)
		if readErr != nil && !eof {
			return stats, readErr
		}

		// Scan the whole buffer for matches. A match found here is fully
		// contained in buf[0:end] (bytes.Index requires the entire pattern
		// to fit inside the search slice).
		i := 0
		for i < end {
			j := bytes.Index(buf[i:end], find)
			if j < 0 {
				break
			}
			if _, err := w.Write(buf[i : i+j]); err != nil {
				return stats, err
			}
			if _, err := w.Write(replace); err != nil {
				return stats, err
			}
			stats.changed = true
			i += j + len(find)
		}

		if eof {
			// Emit anything left over and we're done.
			if i < end {
				if _, err := w.Write(buf[i:end]); err != nil {
					return stats, err
				}
			}
			return stats, nil
		}

		// Determine how much of the unmatched tail is safe to emit. The
		// last (len(find)-1) bytes might be the start of a match that
		// completes after the next read, so they must be carried forward.
		safeEnd := end - (len(find) - 1)
		if safeEnd < i {
			safeEnd = i
		}
		if i < safeEnd {
			if _, err := w.Write(buf[i:safeEnd]); err != nil {
				return stats, err
			}
		}

		copy(buf, buf[safeEnd:end])
		keep = end - safeEnd
	}
}
