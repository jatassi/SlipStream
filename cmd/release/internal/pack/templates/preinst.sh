#!/bin/sh
set -e
at_least() {
	[ "$(printf '%s\n%s\n' "$2" "$1" | sort -V | head -n 1)" = "$2" ]
}
glibc=$(getconf GNU_LIBC_VERSION | cut -d ' ' -f 2)
if ! at_least "$glibc" {{.Glibc}}; then
	echo "{{.Product}} needs glibc {{.Glibc}} or later, and this host has glibc $glibc." >&2
	exit 1
fi
kernel=$(uname -r | cut -d - -f 1)
if ! at_least "$kernel" {{.Kernel}}; then
	echo "{{.Product}} needs Linux {{.Kernel}} or later, and this host runs Linux $kernel." >&2
	exit 1
fi
