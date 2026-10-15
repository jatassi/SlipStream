#!/bin/sh
set -e
at_least() {
	[ "$(printf '%s\n%s\n' "$2" "$1" | sort -V | head -n 1)" = "$2" ]
}
glibc=$(getconf GNU_LIBC_VERSION | cut -d ' ' -f 2)
if ! at_least "$glibc" 2.28; then
	echo "SlipStream needs glibc 2.28 or later, and this host has glibc $glibc." >&2
	exit 1
fi
kernel=$(uname -r | cut -d - -f 1)
if ! at_least "$kernel" 4.18; then
	echo "SlipStream needs Linux 4.18 or later, and this host runs Linux $kernel." >&2
	exit 1
fi
