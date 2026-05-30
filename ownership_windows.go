//go:build windows

package main

import "os"

func chownTempFromInfo(tempPath string, info os.FileInfo) error {
	return nil
}
