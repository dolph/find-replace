//go:build windows

package main

import "os"

func chownToOriginal(path string, info os.FileInfo) error {
	// Windows does not have a meaningful equivalent.
	return nil
}
