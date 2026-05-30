//go:build !linux && !darwin && !windows

package main

import (
	"errors"
	"fmt"
	"os"
)

func atomicRenameNoReplace(oldpath, newpath string) error {
	if _, err := os.Stat(newpath); err == nil {
		return fmt.Errorf("refusing to rename %v to %v: %v already exists", oldpath, newpath, newpath)
	} else if !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("stat rename destination %v: %w", newpath, err)
	}
	return os.Rename(oldpath, newpath)
}
