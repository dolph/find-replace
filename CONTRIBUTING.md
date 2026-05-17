# Contributing

Thanks for helping improve find-replace.

## Development setup

```bash
git clone https://github.com/dolph/find-replace.git
cd find-replace
go test ./...
go vet ./...
```

`./build.sh` runs vet, a release-style build, and tests. It works on Linux and macOS.

## Pull requests

- One logical change per PR; link the issue (`Fixes #N`).
- Run `go test ./...` and `go vet ./...` before pushing.
- Match existing style: minimal comments, straightforward Go.
- PR titles use conventional prefixes when possible: `fix:`, `feat:`, `docs:`, `test:`, `chore:`.

## Labels

Maintainers use `release:*` labels on PRs that should appear in release notes. If your change is user-visible, mention the desired release note in the PR body.

## Security

See [SECURITY.md](SECURITY.md) for reporting vulnerabilities privately.
