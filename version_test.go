package main

import (
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
	code := run([]string{"find-replace", "--version"}, ioDiscard{t})
	if code != 0 {
		t.Fatalf("run --version exit = %d; want 0", code)
	}
}

type ioDiscard struct{ t *testing.T }

func (d ioDiscard) Write(p []byte) (int, error) {
	if !strings.Contains(string(p), "find-replace") {
		d.t.Fatalf("version output = %q; want find-replace", p)
	}
	return len(p), nil
}
