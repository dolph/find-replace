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

However, in order to recursively rename files & directories with this snippet, you have to run it until it stops failing (because it's renaming directories that it has not traversed yet):

First attempt:

```
+ find . -type f -not -path './.git/*' -exec sed -i s/virt/subvert/g '{}' ';'
+ + find . -iname '*virt*' -not -path './.git/*' -exec rename virt subvert '{}' ';'
+ find: ‘./doc/api_samples/os-virtual-interfaces’: No such file or directory
+ find: ‘./nova/tests/functional/libvirt’: No such file or directory
+ find: ‘./nova/tests/unit/virt’: No such file or directory
+ find: ‘./nova/virt’: No such file or directory
real    0m5.755s
user    0m1.651s
sys     0m3.893s
```

Second attempt:

```
+ find . -type f -not -path './.git/*' -exec sed -i s/virt/subvert/g '{}' ';'
+ + find . -iname '*virt*' -not -path './.git/*' -exec rename virt subvert '{}' ';'
+ find: ‘./nova/tests/unit/subvert/libvirt’: No such file or directory
+ find: ‘./nova/subvert/libvirt’: No such file or directory
real    0m6.680s
user    0m1.593s
sys     0m4.864s
```

Third attempt:

```
+ find . -type f -not -path './.git/*' -exec sed -i s/virt/subvert/g '{}' ';'
+ + find . -iname '*virt*' -not -path './.git/*' -exec rename virt subvert '{}' ';'
real    0m6.802s
user    0m1.705s
sys     0m4.866s
```

So, it effectively takes 3 attempts and a sum total of 19.237 seconds to find and replace "virt" with "libvirt" in this example.
