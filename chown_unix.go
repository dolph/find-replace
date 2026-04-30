//go:build unix

package main

import (
	"errors"
	"io/fs"
	"os"
	"syscall"
)

// chownToOriginal applies the original uid/gid from info to path. Best-effort:
// permission errors (typical for non-root users) are silently ignored. Other
// errors are returned.
func chownToOriginal(path string, info os.FileInfo) error {
	st, ok := info.Sys().(*syscall.Stat_t)
	if !ok {
		return nil
	}
	err := os.Chown(path, int(st.Uid), int(st.Gid))
	if err == nil {
		return nil
	}
	// Non-root processes lack CAP_CHOWN and will get EPERM here. That's
	// expected: the temp file already has the correct uid (the running
	// user), and the running user is the owner of the original file too
	// (otherwise they couldn't have rewritten it). Don't fail.
	if errors.Is(err, fs.ErrPermission) {
		return nil
	}
	return err
}
