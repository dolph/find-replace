# AGENTS.md

Contributor guide for humans and AI agents working on `find-replace`. This is
the single source of truth — `CLAUDE.md` imports it.

**Don't make assertions in this file about the current or future state of the
repo** — "currently does X", "once issue #N lands", "this bug exists in
foo.go:42". They go stale fast and create a docs-rot tax. Keep the content
here to durable rules, language-level gotchas, and process conventions. If a
claim only makes sense relative to a specific point in time, it belongs in a
PR description or a commit message, not here.

## What this project is

`find-replace` is a small Go CLI (single binary) that recursively
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

Adding a flag or subcommand requires updating the README in the same PR and
applying at least `release:minor`. Backwards-incompatible changes to that
surface require `release:major`.

## Development loop

Run these in order before sending a PR:

```bash
gofmt -l .                       # must print nothing
go vet ./...                     # zero output
go build ./...                   # zero output
go test -race ./...              # zero output beyond PASS
./build.sh                       # stamps version metadata into the binary
```

`go test -race` is non-negotiable. Anywhere the codebase fans out work
across goroutines, silent data races are the most likely class of
regression.

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
  real working directory, the user's `$HOME`, or the real network from a
  test or benchmark.
- **`t.Helper()` is mandatory** at the top of every helper function that
  takes a `*testing.T`. Without it, failure lines point inside the helper
  instead of the call site.
- **Use `t.Fatalf`, never `log.Fatalf`, in test code.** `log.Fatalf` calls
  `os.Exit(1)`, which kills the test binary, leaks any temp files (deferred
  cleanup doesn't run on `os.Exit`), and prevents subsequent tests from
  running.
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
  - `net.JoinHostPort` for `host:port` strings.
  - `errors.Is` / `errors.As` instead of `==` / type-assertion on errors.
- **HTTP / file resources** must always have `defer resp.Body.Close()` /
  `defer f.Close()` immediately after the open call, never later.
- **`context.Context` plumbing.** If a function does long-running work or
  makes external calls, take a `ctx context.Context` as its first parameter.
  Short single-shot operations don't need one.
- **`log/slog`** for any new structured logging. Don't mix `log/slog` and
  `log.Printf` within a single emit site; for user-facing status output
  documented in the README, follow the existing style.
- **No `math/rand` for anything that ends up in a filesystem path.** Use
  `os.CreateTemp` or `crypto/rand`.
- **No `log.Fatal` from goroutines.** Bubble errors up. The only allowed
  `log.Fatal` site is `main`, after the walker has fully drained.

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
  schema, exit codes, or emitted log/metric names. Tightening input
  validation that rejects a previously-accepted invocation is also a
  breaking change.

Precedence when multiple labels are set: `skip > major > minor > patch`.
`release:skip` always wins, so a PR can be parked mid-flight.

**Apply the label *before* merging.** The workflow reads it at merge time.

## Priority labels (mandatory on every issue)

Apply exactly one of these to every issue. (Note: repo convention uses a
space after the colon, `priority: critical`.)

- **`priority: critical`** — Drop everything. Production-impacting bug:
  data loss, security vulnerability shipped in a release, the tool's
  stated purpose is broken.
- **`priority: high`** — Significant correctness, security, or
  reliability concern. Fix in the next release cycle.
- **`priority: medium`** — Important quality-of-life or prevention work.
- **`priority: low`** — Nice to have, backlog.

Type labels (`bug`, `security`, `reliability`, `performance`,
`enhancement`) are orthogonal — preserve them.

## Language and stdlib gotchas

Durable traps that bite anyone writing filesystem-walking Go code. None of
these are specific to this codebase — they're properties of the language
and stdlib.

- **`os.Stat` follows symlinks.** Anywhere code decides "is this a
  directory" via `os.Stat`, the answer is a lie for symlinks. Use
  `os.Lstat` or the `fs.DirEntry` returned by `os.ReadDir`.
- **`os.Rename` silently overwrites.** Linux/POSIX `rename(2)` clobbers
  the destination by default. A `Stat`-then-`Rename` "does the
  destination exist?" guard is a TOCTOU race; the answer can change
  between the two calls. Use `os.Link`+`os.Remove`, or
  `unix.Renameat2(RENAME_NOREPLACE)`.
- **`os.WriteFile` uses `O_CREATE`, not `O_CREATE|O_EXCL`.** If the
  target path already exists (e.g., as an attacker-pre-created symlink),
  the write follows it. Use `os.OpenFile` with `O_CREATE|O_EXCL` for any
  temp file written into a shared directory.
- **`log.Fatal` from a goroutine does not run deferreds.** It calls
  `os.Exit(1)`, which kills the process without flushing deferred temp
  file cleanup. Worker goroutines that hit a fatal error must return it,
  not log-and-exit.
- **Build-time `-ldflags -X` against undefined variables is silent.**
  `go build` accepts `-ldflags="-X 'main.Foo=bar'"` even when no
  package-level `Foo` exists; the metadata is dropped. Always declare
  the target variable in source before adding a `-X` injection.

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
