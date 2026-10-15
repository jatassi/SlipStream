#!/bin/sh
set -e
{{- if .Deb}}
[ "$1" = configure ] || exit 0
{{- end}}
systemd-sysusers {{.Sysusers}}
if {{.Fresh}}; then
	systemctl enable {{.Unit}}
fi
if [ -d /run/systemd/system ]; then
	systemctl daemon-reload
	if {{.Fresh}}; then
		systemctl start {{.Unit}}
	else
		systemctl try-restart {{.Unit}}
	fi
fi
