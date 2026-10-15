#!/bin/sh
set -e
{{- if .Deb}}
[ "$1" = remove ] || exit 0
{{- else}}
[ "$1" -eq 0 ] || exit 0
{{- end}}
systemctl disable {{.Unit}}
if [ -d /run/systemd/system ]; then
	systemctl stop {{.Unit}}
fi
