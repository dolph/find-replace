package main

import "fmt"

// Build metadata injected by build.sh via -ldflags -X.
var (
	GitTag         string
	GitCommit      string
	GoVersion      string
	BuildTimestamp string
	BuildOS        string
	BuildArch      string
	BuildTainted   string
)

func versionString() string {
	tag := GitTag
	if tag == "" {
		tag = "dev"
	}
	commit := GitCommit
	if commit == "" {
		commit = "unknown"
	}
	return fmt.Sprintf(
		"find-replace %s (%s) go=%s built=%s os=%s arch=%s tainted=%s",
		tag, commit, GoVersion, BuildTimestamp, BuildOS, BuildArch, BuildTainted,
	)
}
