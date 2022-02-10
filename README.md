# `find-replace`

A fast find &amp; replace shell command.

## Usage

Recursively find and replace the string `alpha` with `beta` in both file names and file contents:

```bash
$ find-replace alpha beta
Rewriting ./hello-world
Renaming ./alphabet to betabet
```

* Files with matching contents in the current working directory are atomically rewritten.
* Files and directories are renamed.
* Searches are performed recursively from the current working directory.
* Searches are case sensitive.
* `.git/` directories are skipped.
* Binary files are ignored.

## Goal

The goal of this project is to improve on a bash snippet that I've relied on for years, by making it faster. The bash:

```bash
#!/bin/bash
set -ex
find . -type f -not -path './.git/*' -exec sed -i "s/$1/$2/g" '{}' \;
find . -iname "*$1*" -not -path "./.git/*" -exec rename "$1" "$2" '{}' \;
```
