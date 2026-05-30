package main

import "os"

func hasSpecialFileModeBits(mode os.FileMode) bool {
	return mode&os.ModeSetuid != 0 || mode&os.ModeSetgid != 0 || mode&os.ModeSticky != 0
}
