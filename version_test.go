package main

import (
	"bytes"
	"strings"
	"testing"
)

func TestVersionString(t *testing.T) {
	GitTag = "v1.2.3"
	GitCommit = "abc1234"
	GoVersion = "go1.22.0"
	BuildTimestamp = "2026-01-01T00:00:00Z"
	BuildOS = "linux"
	BuildArch = "amd64"
	BuildTainted = "false"

	got := versionString()
	for _, want := range []string{"v1.2.3", "abc1234", "go1.22.0", "linux", "amd64"} {
		if !strings.Contains(got, want) {
			t.Fatalf("versionString() = %q; want substring %q", got, want)
		}
	}
}

func TestRunVersionFlag(t *testing.T) {
	var stderr bytes.Buffer
	code := run([]string{"find-replace", "--version"}, &stderr)
	if code != 0 {
		t.Fatalf("run --version exit = %d; want 0", code)
	}
	if !strings.Contains(stderr.String(), "find-replace") {
		t.Fatalf("version output = %q; want find-replace", stderr.String())
	}
}

func TestRunHelpFlag(t *testing.T) {
	var stderr bytes.Buffer
	code := run([]string{"find-replace", "--help"}, &stderr)
	if code != 0 {
		t.Fatalf("run --help exit = %d; want 0", code)
	}
	if !strings.Contains(stderr.String(), "Usage: find-replace") {
		t.Fatalf("help output = %q; want usage line", stderr.String())
	}
}
