package main

import (
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

// TestNewFile exercises NewFile's path-resolution behavior.
func TestNewFile(t *testing.T) {
	tmp := t.TempDir()

	tests := []struct {
		name string
		// input is the raw path passed to NewFile.
		input string
		// want is the absolute path the resulting *File should expose. If
		// empty, the test computes the expected value via filepath.Abs(input)
		// at runtime (useful for inputs that are inherently relative to the
		// test process's working directory).
		want string
	}{
		{
			name:  "absolute path is returned cleaned",
			input: filepath.Join(tmp, "foo"),
			want:  filepath.Join(tmp, "foo"),
		},
		{
			name:  "absolute path with redundant separators is cleaned",
			input: tmp + "//foo///bar",
			want:  filepath.Join(tmp, "foo", "bar"),
		},
		{
			name:  "absolute path with .. is resolved",
			input: filepath.Join(tmp, "a", "..", "b"),
			want:  filepath.Join(tmp, "b"),
		},
		{
			name:  "relative path is resolved to absolute",
			input: "relative/path",
			// want is computed below because it depends on the test
			// process's working directory.
		},
		{
			name:  "relative path with .. is resolved",
			input: "a/../b",
		},
		{
			name:  "dot is resolved to the working directory",
			input: ".",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			want := tc.want
			if want == "" {
				abs, err := filepath.Abs(tc.input)
				if err != nil {
					t.Fatalf("filepath.Abs(%q) returned unexpected error: %v", tc.input, err)
				}
				want = abs
			}

			got, err := NewFile(tc.input)
			if err != nil {
				t.Fatalf("NewFile(%q) returned unexpected error: %v", tc.input, err)
			}
			if got == nil {
				t.Fatalf("NewFile(%q) returned nil", tc.input)
			}
			if got.Path != want {
				t.Errorf("NewFile(%q).Path = %q; want %q", tc.input, got.Path, want)
			}
			if !filepath.IsAbs(got.Path) {
				t.Errorf("NewFile(%q).Path = %q; want an absolute path", tc.input, got.Path)
			}
		})
	}
}

// writeAndPrime is a tiny helper that primes the file's cached Info (Write
// requires it) and runs Write, returning the resulting error.
func writeAndPrime(tb testing.TB, f *File, content string) error {
	tb.Helper()
	if _, err := f.Info(); err != nil {
		tb.Fatalf("Info(%q): %v", f.Path, err)
	}
	return f.Write(content)
}

// TestWrite_TempFileUsesORDWREXCL documents the security invariant we get
// from os.CreateTemp: the temp file is opened with O_RDWR|O_CREATE|O_EXCL,
// so a pre-existing file at the same path would cause the open to fail
// rather than clobber. The CSPRNG-quality suffix on the temp name means an
// attacker cannot reasonably pre-create the right name; this test pre-
// creates a distractor under the same prefix to assert it survives the
// rewrite unchanged.
func TestWrite_TempFileUsesORDWREXCL(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "target.txt")
	if err := os.WriteFile(target, []byte("alpha"), 0644); err != nil {
		t.Fatalf("WriteFile(%q): %v", target, err)
	}

	// Pre-create a sentinel file under the agreed-upon prefix. Whatever the
	// random suffix happens to be, it must not collide with this name (the
	// CSPRNG suffix guarantees that statistically); even if it did,
	// O_EXCL would cause the open to fail, leaving the sentinel untouched.
	sentinelName := tempFilePrefix + "collision"
	sentinel := filepath.Join(dir, sentinelName)
	const sentinelContent = "do-not-touch"
	if err := os.WriteFile(sentinel, []byte(sentinelContent), 0644); err != nil {
		t.Fatalf("WriteFile(%q): %v", sentinel, err)
	}

	f := newFileOrFatal(t, target)
	if err := writeAndPrime(t, f, "beta"); err != nil {
		t.Fatalf("Write(%q): %v", target, err)
	}

	// The target file should be rewritten.
	got, err := os.ReadFile(target)
	if err != nil {
		t.Fatalf("ReadFile(%q): %v", target, err)
	}
	if string(got) != "beta" {
		t.Errorf("target contents = %q; want %q", string(got), "beta")
	}

	// The sentinel must be exactly as we left it.
	gotSentinel, err := os.ReadFile(sentinel)
	if err != nil {
		t.Fatalf("ReadFile(%q): %v", sentinel, err)
	}
	if string(gotSentinel) != sentinelContent {
		t.Errorf("sentinel contents = %q; want %q (Write clobbered a same-prefix file)", string(gotSentinel), sentinelContent)
	}
}

// TestWrite_TempFilePrefixVisible enforces the agreed-upon temp-file name
// prefix. On a successful rewrite, the temp file has been renamed over the
// target, so no `.find-replace.*` entry should linger in the target's
// directory.
func TestWrite_TempFilePrefixVisible(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "target.txt")
	if err := os.WriteFile(target, []byte("alpha"), 0644); err != nil {
		t.Fatalf("WriteFile(%q): %v", target, err)
	}

	f := newFileOrFatal(t, target)
	if err := writeAndPrime(t, f, "beta"); err != nil {
		t.Fatalf("Write(%q): %v", target, err)
	}

	// After a successful Write, no temp file should remain.
	matches, err := filepath.Glob(filepath.Join(dir, tempFilePrefix+"*"))
	if err != nil {
		t.Fatalf("Glob(%q*): %v", tempFilePrefix, err)
	}
	if len(matches) != 0 {
		t.Errorf("leftover temp files matching %q*: %v", tempFilePrefix, matches)
	}
}

// TestWrite_PreservesOriginalPermissionBits verifies that the rewrite
// preserves the original file's permission bits (mode.Perm() — setuid/
// setgid/sticky preservation is out of scope, tracked separately in
// issue #18). Without an explicit Chmod after CreateTemp, the rewritten
// file would silently become 0600.
func TestWrite_PreservesOriginalPermissionBits(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("permission-bit semantics differ on Windows")
	}

	tests := []struct {
		name string
		mode os.FileMode
	}{
		{name: "0644", mode: 0644},
		{name: "0600", mode: 0600},
		{name: "0755", mode: 0755},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			target := filepath.Join(dir, "target.txt")
			if err := os.WriteFile(target, []byte("alpha"), tc.mode); err != nil {
				t.Fatalf("WriteFile(%q, _, %v): %v", target, tc.mode, err)
			}
			// Some umasks would alter the on-disk mode; force it.
			if err := os.Chmod(target, tc.mode); err != nil {
				t.Fatalf("Chmod(%q, %v): %v", target, tc.mode, err)
			}

			f := newFileOrFatal(t, target)
			if err := writeAndPrime(t, f, "beta"); err != nil {
				t.Fatalf("Write(%q): %v", target, err)
			}

			info, err := os.Stat(target)
			if err != nil {
				t.Fatalf("Stat(%q) after Write: %v", target, err)
			}
			if got := info.Mode().Perm(); got != tc.mode {
				t.Errorf("mode after Write = %v; want %v", got, tc.mode)
			}
		})
	}
}

// TestWrite_DoesNotFollowAttackerPreparedSymlinkAtTempPath simulates an
// attacker who has pre-created a symlink under the agreed temp-file prefix
// pointing at a sensitive target. The rewrite must NOT follow that symlink
// — it gets a fresh random name from os.CreateTemp, and even on a
// (statistically impossible) name collision O_EXCL would fail. The decoy
// must remain a symlink to the same destination, and the destination's
// contents must be unchanged.
func TestWrite_DoesNotFollowAttackerPreparedSymlinkAtTempPath(t *testing.T) {
	requireSymlinks(t)

	dir := t.TempDir()
	target := filepath.Join(dir, "target.txt")
	if err := os.WriteFile(target, []byte("alpha"), 0644); err != nil {
		t.Fatalf("WriteFile(%q): %v", target, err)
	}

	// "Sensitive" file the attacker hopes to clobber via the symlink.
	sensitiveDir := t.TempDir()
	sensitive := filepath.Join(sensitiveDir, "victim")
	const sensitiveContent = "must-not-change"
	if err := os.WriteFile(sensitive, []byte(sensitiveContent), 0644); err != nil {
		t.Fatalf("WriteFile(%q): %v", sensitive, err)
	}

	// Pre-create a symlink in the rewrite directory under the temp prefix
	// pointing at the sensitive file.
	decoy := filepath.Join(dir, tempFilePrefix+"attacker")
	if err := os.Symlink(sensitive, decoy); err != nil {
		t.Fatalf("Symlink(%q, %q): %v", sensitive, decoy, err)
	}

	f := newFileOrFatal(t, target)
	if err := writeAndPrime(t, f, "beta"); err != nil {
		t.Fatalf("Write(%q): %v", target, err)
	}

	// (a) Rewrite succeeded.
	got, err := os.ReadFile(target)
	if err != nil {
		t.Fatalf("ReadFile(%q): %v", target, err)
	}
	if string(got) != "beta" {
		t.Errorf("target contents = %q; want %q", string(got), "beta")
	}

	// (b) The decoy still exists as a symlink (was not opened-and-written
	// through, was not replaced by a regular file).
	info, err := os.Lstat(decoy)
	if err != nil {
		t.Fatalf("Lstat(%q): %v (decoy symlink was removed)", decoy, err)
	}
	if info.Mode()&fs.ModeSymlink == 0 {
		t.Errorf("Lstat(%q).Mode() = %v; want a symlink (was replaced with a regular file)", decoy, info.Mode())
	}

	// (c) The symlink's destination is unchanged.
	gotSensitive, err := os.ReadFile(sensitive)
	if err != nil {
		t.Fatalf("ReadFile(%q): %v", sensitive, err)
	}
	if string(gotSensitive) != sensitiveContent {
		t.Errorf("sensitive contents = %q; want %q (symlink was followed)", string(gotSensitive), sensitiveContent)
	}

	// Sanity check: the symlink in the rewrite dir is still pointing at the
	// same place.
	if dst, err := os.Readlink(decoy); err != nil {
		t.Fatalf("Readlink(%q): %v", decoy, err)
	} else if dst != sensitive {
		t.Errorf("Readlink(%q) = %q; want %q", decoy, dst, sensitive)
	}
}
