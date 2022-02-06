# `find-replace`

A fast find &amp; replace shell command.

## Usage

Recursively find and replace the string `"a"` with `"b"` in both file names and file contents:

```bash
find-replace "a" "b"
```

`.git/` directories are ignored.

## Goal

The goal of this project is to improve on a bash snippet that I've relied on for years, by making it faster. The bash:

```bash
#!/bin/bash
set -ex
find . -type f -not -path './.git/*' -exec sed -i "s/$1/$2/g" '{}' \;
find . -iname "*$1*" -not -path "./.git/*" -exec rename "$1" "$2" '{}' \;
```
