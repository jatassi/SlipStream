#!/bin/sh
set -e
[ "$1" = remove ] || exit 0
systemctl disable slipstream.service
if [ -d /run/systemd/system ]; then
	systemctl stop slipstream.service
fi
