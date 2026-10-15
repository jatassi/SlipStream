#!/bin/sh
set -e
[ "$1" = configure ] || exit 0
systemd-sysusers /usr/lib/sysusers.d/slipstream.conf
if [ -z "$2" ]; then
	systemctl enable slipstream.service
fi
if [ -d /run/systemd/system ]; then
	systemctl daemon-reload
	if [ -z "$2" ]; then
		systemctl start slipstream.service
	else
		systemctl try-restart slipstream.service
	fi
fi
