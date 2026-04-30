package main

import (
	"path/filepath"
	"testing"
)

func TestNewFileAbsolutePath(t *testing.T) {
	f := NewFile("/tmp/find-replace/example")
	if !filepath.IsAbs(f.Path) {
		t.Errorf("expected absolute path, got %v", f.Path)
	}
}

func TestNewFileRelativePath(t *testing.T) {
	f := NewFile("example")
	if !filepath.IsAbs(f.Path) {
		t.Errorf("expected absolute path, got %v", f.Path)
	}
}

func TestNewChildFileSkipsAbs(t *testing.T) {
	parent := "/tmp/find-replace"
	child := newChildFile(parent, "kid")
	if child.Path != "/tmp/find-replace/kid" {
		t.Errorf("unexpected path: %v", child.Path)
	}
}

func TestBaseDir(t *testing.T) {
	f := NewFile("/tmp/find-replace/example")
	if f.Base() != "example" {
		t.Errorf("Base = %v, want example", f.Base())
	}
	if f.Dir() != "/tmp/find-replace" {
		t.Errorf("Dir = %v, want /tmp/find-replace", f.Dir())
	}
}
