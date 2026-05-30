package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestParseRunArgs(t *testing.T) {
	cases := []struct {
		name        string
		args        []string
		wantFind    string
		wantReplace string
		contentOnly bool
		renameOnly  bool
		wantErr     bool
	}{
		{name: "default", args: []string{"find-replace", "a", "b"}, wantFind: "a", wantReplace: "b"},
		{name: "content only", args: []string{"find-replace", "--content-only", "a", "b"}, wantFind: "a", wantReplace: "b", contentOnly: true},
		{name: "rename only", args: []string{"find-replace", "--rename-only", "a", "b"}, wantFind: "a", wantReplace: "b", renameOnly: true},
		{name: "both flags", args: []string{"find-replace", "--content-only", "--rename-only", "a", "b"}, wantErr: true},
		{name: "missing args", args: []string{"find-replace", "--content-only"}, wantErr: true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			find, replace, contentOnly, renameOnly, err := parseRunArgs(tc.args)
			if tc.wantErr {
				if err == nil {
					t.Fatal("expected error")
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if find != tc.wantFind || replace != tc.wantReplace || contentOnly != tc.contentOnly || renameOnly != tc.renameOnly {
				t.Fatalf("parseRunArgs() = (%q,%q,%v,%v); want (%q,%q,%v,%v)",
					find, replace, contentOnly, renameOnly, tc.wantFind, tc.wantReplace, tc.contentOnly, tc.renameOnly)
			}
		})
	}
}

func TestContentOnlySkipsRename(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "needle.txt")
	if err := os.WriteFile(path, []byte("needle content"), 0o644); err != nil {
		t.Fatal(err)
	}
	f := newFileOrFatal(t, path)

	fr := findReplace{find: "needle", replace: "hay", contentOnly: true}
	if err := fr.HandleFile(f); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("file was renamed; want %q kept: %v", path, err)
	}
	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "hay content" {
		t.Fatalf("content = %q; want rewritten content", got)
	}
}

func TestRenameOnlySkipsContent(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "needle.txt")
	if err := os.WriteFile(path, []byte("needle content"), 0o644); err != nil {
		t.Fatal(err)
	}
	f := newFileOrFatal(t, path)

	fr := findReplace{find: "needle", replace: "hay", renameOnly: true}
	if err := fr.HandleFile(f); err != nil {
		t.Fatal(err)
	}
	renamed := filepath.Join(dir, "hay.txt")
	if _, err := os.Stat(path); err == nil {
		t.Fatal("original file still exists; expected rename")
	}
	got, err := os.ReadFile(renamed)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "needle content" {
		t.Fatalf("content = %q; want unchanged", got)
	}
}
