package main

import "os"

const specialFileModeBits = os.ModeSetuid | os.ModeSetgid | os.ModeSticky

func hasSpecialFileModeBits(mode os.FileMode) bool {
	return mode&specialFileModeBits != 0
}
