#!/bin/sh
set -e
[ "$1" -eq 0 ] || exit 0
systemctl disable slipstream.service
if [ -d /run/systemd/system ]; then
	systemctl stop slipstream.service
fi
