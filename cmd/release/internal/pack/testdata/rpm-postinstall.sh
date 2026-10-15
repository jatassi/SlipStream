#!/bin/sh
set -e
systemd-sysusers /usr/lib/sysusers.d/slipstream.conf
if [ "$1" -eq 1 ]; then
	systemctl enable slipstream.service
fi
if [ -d /run/systemd/system ]; then
	systemctl daemon-reload
	if [ "$1" -eq 1 ]; then
		systemctl start slipstream.service
	else
		systemctl try-restart slipstream.service
	fi
fi
