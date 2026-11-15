#!/usr/bin/env bash
# Runner post-job hook. Each runner's service has its own /tmp, so everything
# in it was left by this runner's jobs, some of them as root.
sudo find /tmp -mindepth 1 -maxdepth 1 -exec rm -rf {} + 2> /dev/null || true
