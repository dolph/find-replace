//go:build !windows

package main

import (
	"os"
	"syscall"
)

func chownTempFromInfo(tempPath string, info os.FileInfo) error {
	sys, ok := info.Sys().(*syscall.Stat_t)
	if !ok {
		return nil
	}
	return os.Chown(tempPath, int(sys.Uid), int(sys.Gid))
}
