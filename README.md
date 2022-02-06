# `find-replace`

A fast find &amp; replace shell command.

## Usage

Recursively find and replace the string `alpha` with `beta` in both file names and file contents:

```bash
$ find-replace alpha beta
Rewriting ./hello-world
Renaming ./alphabet to betabet
```

* Searches are case sensitive.
* `.git/` directories are skipped.
* File types are ignored.

## Goal

The goal of this project is to improve on a bash snippet that I've relied on for years, by making it faster. The bash:

```bash
#!/bin/bash
set -ex
find . -type f -not -path './.git/*' -exec sed -i "s/$1/$2/g" '{}' \;
find . -iname "*$1*" -not -path "./.git/*" -exec rename "$1" "$2" '{}' \;
```
