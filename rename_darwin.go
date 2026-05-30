//go:build darwin

package main

import (
	"errors"
	"fmt"

	"golang.org/x/sys/unix"
)

func atomicRenameNoReplace(oldpath, newpath string) error {
	err := unix.RenameatxNp(unix.AT_FDCWD, oldpath, unix.AT_FDCWD, newpath, unix.RENAME_EXCL)
	if errors.Is(err, unix.EEXIST) {
		return fmt.Errorf("refusing to rename %v to %v: %v already exists", oldpath, newpath, newpath)
	}
	return err
}
