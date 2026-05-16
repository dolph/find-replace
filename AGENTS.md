# AGENTS.md

Contributor guide for humans and AI agents working on `find-replace`. This is
the single source of truth — `CLAUDE.md` imports it.

## What this project is

`find-replace` is a small Go CLI (~400 LoC, single binary) that recursively
finds and replaces a string across both file contents *and* file/directory
names, rooted at `$PWD`. Files with matching contents are atomically
rewritten via a temp-file + rename; `.git/` is skipped; binary files are
ignored. The competitor it benchmarks against is the
`find … -exec sed -i …` / `rename` bash idiom — speed and single-traversal
correctness are the value proposition.

The CLI surface is intentionally minimal:

```
find-replace FIND REPLACE
```

No subcommands, no flags (yet — see #26). Adding flags is allowed but
requires updating the README and bumping `release:minor` or higher.

## Development loop

Run these in order before sending a PR:

```bash
gofmt -l .                       # must print nothing
go vet ./...                     # zero output
go build ./...                   # zero output
go test -race ./...              # zero output beyond PASS

# The full build script also stamps version metadata (Linux only today; see #32):
./build.sh
```

`go test -race` is non-negotiable. The walker is concurrent (and will get
*more* concurrent once #7 lands); silent data races are this project's most
likely class of regression.

## Test-driven development

Default workflow for any behavior change:

1. **Write the failing test first.** Run `go test -race ./...`, *confirm
   the test fails for the right reason* — wrong-output failure, not a
   compile error or a setup error. A test that fails to compile is not a
   failing test.
2. **Minimal fix.** Land just enough code to make the test pass. Don't
   refactor in the same change.
3. **Refactor green.** If structural cleanup is warranted, do it after the
   test is passing, and keep it passing on every save.

### Go-specific test discipline

- **Table-driven** wherever the test inputs/outputs are uniform. Use
  `t.Run(tc.name, …)` so failures are addressable individually.
- **Hermetic by default.** Use `t.TempDir()` instead of `os.TempDir()` —
  `t.TempDir` is auto-cleaned and unique per test. Do **not** touch the
  real working directory, the user's `$HOME`, or the real network. The
  network-bound `BenchmarkNova` (#16) is the cautionary tale.
- **`t.Helper()` is mandatory** at the top of every helper function that
  takes a `*testing.T`. Without it, failure lines point inside the helper
  instead of the call site.
- **Use `t.Fatalf`, never `log.Fatalf`, in test code.** `log.Fatalf` calls
  `os.Exit(1)`, which kills the test binary, leaks any temp files (deferred
  cleanup doesn't run on `os.Exit`), and prevents subsequent tests from
  running. This bug currently exists in several helpers — see #33.
- **Descriptive test names.** `TestRenameFile_RefusesOverwriteOfExisting`
  reads better than `TestRenameFile2`.
- **Beware of checklist theater.** Tests that look like:

  ```go
  if err := fr.HandleFile(f); err != nil {
      t.Fatal(err)
  }
  ```

  …assert nothing about the *outcome*. They check only that the call
  returned. A real test asserts on file contents, file paths, exit code,
  emitted log lines — the *observable behavior* under test, not the fact
  that the function ran.

## Code style

- **`fmt.Errorf` with `%w`** for wrapped errors:
  `fmt.Errorf("rename %s → %s: %w", src, dst, err)`. Don't use `errors.New`
  for formatted strings; don't lose the inner error.
- **Modern stdlib.** Prefer:
  - `os.ReadFile` / `os.WriteFile` over hand-rolled `os.Open`+`io.Copy`.
  - `strings.ReplaceAll(s, old, new)` over `strings.Replace(s, old, new, -1)`.
  - `net.JoinHostPort` for `host:port` strings (not relevant here, listed
    for completeness).
  - `errors.Is` / `errors.As` instead of `==` / type-assertion on errors.
- **HTTP / file resources** must always have `defer resp.Body.Close()` /
  `defer f.Close()` immediately after the open call, never later.
- **`context.Context` plumbing** is not required today (the tool is a
  short-lived single-traversal process), but if you add a long-running
  goroutine or external call, take a `ctx context.Context` as the first
  parameter.
- **`log/slog`** for any *new* structured logging. The current code uses
  `log.Printf` with `log.SetFlags(0)` to emit human-readable status. Don't
  mix the two within a single emit; keep the existing user-facing output
  exactly as documented in the README until the README is updated.
- **No `math/rand` for anything that ends up in a filesystem path.** Use
  `os.CreateTemp` or `crypto/rand` — see #3.
- **No `log.Fatal` from goroutines.** Bubble errors up. The only allowed
  `log.Fatal` site is `main`, after the walker has fully drained — see #6.

## Repo conventions

- **Branch naming.** Feature branches: `<author>/<short-description>` or
  `claude/fix-issue-N-<slug>` for AI-driven work. Don't push directly to
  `main`.
- **Commit subjects.** Imperative mood, ≤72 chars: `fix walker leak on
  read error`, not `fixed a bug in the walker`. Body wrapped at 72 cols.
- **PR description must include `Fixes #N`** (or `Closes #N`) when the
  PR resolves an open issue. GitHub will auto-close on merge.
- **One PR per issue.** If you spot an adjacent defect while working on
  one issue, **file a new issue** rather than expanding the PR. Scope
  creep is the failure mode that kills small-tool maintainability.

## Release labels (mandatory on every PR)

The release workflow (`.github/workflows/release.yml`) is triggered after
every successful CI run on `main`. It inspects the merged PR's labels to
decide the semver bump. **If no `release:*` label is set, the workflow
defaults to `release:patch` and will cut an empty patch release for
docs/test PRs.** Always apply one of:

- **`release:skip`** — No user-visible behavior change. Docs, tests,
  CI, internal refactors that don't change the binary's behavior.
- **`release:patch`** — Bug fix, security patch, dependency bump with no
  CLI/config/output change. Default if nothing is set.
- **`release:minor`** — Additive change: new flag, new subcommand, new
  optional config key, new emitted metric. No existing surface changes.
- **`release:major`** — Breaking change to the CLI surface, config
  schema, exit codes, or emitted log/metric names. Example: rejecting
  empty `FIND` (#10) is technically a breaking change.

Precedence when multiple labels are set: `skip > major > minor > patch`.
`release:skip` always wins, so a PR can be parked mid-flight.

**Apply the label *before* merging.** The workflow reads it at merge time.

## Priority labels (mandatory on every issue)

Apply exactly one of these to every issue. (Note: existing repo convention
uses a space after the colon, `priority: critical`.)

- **`priority: critical`** — Drop everything. Production-impacting bug:
  data loss, security vulnerability shipped in a release, the tool's
  stated purpose is broken. Examples: #2 (symlink traversal),
  #3 (predictable temp-file names).
- **`priority: high`** — Significant correctness, security, or
  reliability concern. Fix in the next release cycle. Examples:
  #4 (TOCTOU), #6 (`log.Fatal` from goroutines), #27 (stale CI actions).
- **`priority: medium`** — Important quality-of-life or prevention work.
  Examples: #11 (exit code), #13 (redundant `Stat`), #30 (CI lacks
  `staticcheck`/`govulncheck`).
- **`priority: low`** — Nice to have, backlog. Examples: #14 (double
  scan), #21 (stale temp files), #29 (Dependabot/SECURITY.md).

Type labels (`bug`, `security`, `reliability`, `performance`,
`enhancement`) are orthogonal — preserve them.

## Known traps

High-impact landmines a contributor is likely to hit. Cross-reference the
relevant issue for full repro/fix.

- **Concurrent walker shares no state safely.** Every `*File` is owned by
  exactly one goroutine today. The lazy `File.Info()` cache (`file_handling.go:34`)
  is *not* safe to share. Don't pass a `*File` between goroutines without
  taking ownership. See #12.
- **`os.Stat` follows symlinks.** Anywhere the walker decides "is this a
  directory" via `os.Stat`, the answer is a lie for symlinks. Use
  `os.Lstat` or the `fs.DirEntry` from `os.ReadDir`. See #2 — critical
  security bug.
- **`os.Rename` silently overwrites.** Linux/POSIX `rename(2)` clobbers
  the destination by default. The `os.Stat`-then-`os.Rename` pattern in
  `RenameFile` is a TOCTOU and is not safe. Use `os.Link`+`os.Remove`,
  or `unix.Renameat2(RENAME_NOREPLACE)`. See #4.
- **`os.WriteFile` uses `O_CREATE`, not `O_CREATE|O_EXCL`.** If the temp
  path already exists (attacker-pre-created symlink), the write follows
  the symlink. See #3.
- **`log.Fatal` from a goroutine does not run deferreds.** Worker
  goroutines that `log.Fatal` leak temp files. Tests that `log.Fatal`
  leak temp dirs. See #6, #33.
- **`golang.org/x/tools/godoc/util.IsText`** samples only the first 1024
  bytes. Files with a text prefix and binary tail will be silently
  rewritten. See #9.
- **Build-time `-ldflags -X` against undefined variables.** `go build`
  accepts ldflags that target package-level variables that don't exist;
  the metadata is dropped on the floor. See #26.

## Scope discipline

Single-purpose PRs. The repo is small; the *change* should be small too:

- A bug fix should change behavior, add a test, and nothing else.
- Don't bundle a refactor with a fix. If you want to refactor, do it in
  a separate PR after the fix lands.
- If you see something else broken while working on your PR, **file a
  new issue with a `priority:*` label**, then keep moving. The next
  contributor (or your future self) will pick it up.

This applies doubly to AI agents driving fixes: if you find an adjacent
defect, file the issue, don't expand the diff.
