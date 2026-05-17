package main

import (
	"os"
	"testing"
)

func TestHasSpecialFileModeBits(t *testing.T) {
	if !hasSpecialFileModeBits(os.ModeSetuid | 0o755) {
		t.Error("expected setuid bit to be detected")
	}
	if !hasSpecialFileModeBits(os.ModeSetgid | 0o755) {
		t.Error("expected setgid bit to be detected")
	}
	if !hasSpecialFileModeBits(os.ModeSticky | 0o755) {
		t.Error("expected sticky bit to be detected")
	}
	if hasSpecialFileModeBits(0o644) {
		t.Error("expected plain mode to pass through")
	}
}
