# Changelog

All notable changes to this project are documented here.

## Unreleased

### Changed

- README documents exit-code semantics, rename refusal, and security expectations.
- Worker pool, streaming rewrites, and other reliability/performance fixes are landing via open PRs—see GitHub issues #7–#23.

### Fixed

- Non-fatal walk errors are collected and surfaced as a non-zero exit code (#6).
