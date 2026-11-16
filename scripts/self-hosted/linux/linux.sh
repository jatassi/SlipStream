#!/usr/bin/env bash
# Provisions the Linux runners. provision.ps1 runs it as root in the
# SlipStream-CI WSL distro:
#   linux.sh wsl-conf
#   linux.sh <runners> <kit> <runner-version> <runner-sha256> <registration-token>
set -euo pipefail

if [ "${1:-}" = wsl-conf ]; then
  # Linux can't start Windows programs or see the Windows drives through
  # automount. Root can still mount a drive by hand, and the runner users
  # have sudo, so this limits mistakes, not an attacker.
  cat > /etc/wsl.conf <<'EOF'
[boot]
systemd=true

[automount]
enabled=false
mountFsTab=false

[interop]
enabled=false
appendWindowsPath=false

[time]
useWindowsTimezone=false

[user]
default=root
EOF
  exit 0
fi

RUNNERS=$1
KIT=$2
VERSION=$3
SHA=$4
TOKEN=$5
KIT_DIR=$(cd "$(dirname "$0")" && pwd)
HOOKS=/opt/slipstream-ci/hooks
PACKAGE=/opt/slipstream-ci/actions-runner-linux-x64-$VERSION.tar.gz
export DEBIAN_FRONTEND=noninteractive

# Hosted runners run in UTC, and tests that count days rely on it.
ln -sf /usr/share/zoneinfo/Etc/UTC /etc/localtime
echo Etc/UTC > /etc/timezone
# WSL sets the distro's clock from Windows every few seconds. With NTP on as
# well, the clock stepped back and forth by however far Windows was off.
systemctl disable --now systemd-timesyncd > /dev/null 2>&1 || true

# Jobs on different runners run apt at the same time, so apt waits for the
# lock instead of failing.
echo 'DPkg::Lock::Timeout "900";' > /etc/apt/apt.conf.d/90slipstream-ci-lock
apt-get update -qq
apt-get install -y -qq --no-install-recommends \
  build-essential ca-certificates curl gawk gh git iproute2 jq python3 sudo unzip xz-utils zip zstd > /dev/null

install -d -m 755 "$HOOKS"
for hook in job-started.sh job-completed.sh; do
  sed 's/\r$//' "$KIT_DIR/$hook" > "$HOOKS/$hook"
  chmod 755 "$HOOKS/$hook"
done

if [ ! -f "$PACKAGE" ]; then
  curl -fsSL -o "$PACKAGE.part" "https://github.com/actions/runner/releases/download/v$VERSION/actions-runner-linux-x64-$VERSION.tar.gz"
  echo "$SHA  $PACKAGE.part" | sha256sum -c --quiet
  mv "$PACKAGE.part" "$PACKAGE"
fi

# actions/cache versions a cache by its absolute paths, which name
# runner.temp, and some caches hold paths relative to the workspace, such as
# ../../../.bun. Every runner's service sees its own home at the hosted
# runners' /home/runner, so caches saved on hosted runners restore here and
# the other way around.
install -d -m 755 /home/runner

groupadd -f slipstream-ci
echo '%slipstream-ci ALL=(ALL) NOPASSWD:ALL' > /etc/sudoers.d/slipstream-ci
chmod 440 /etc/sudoers.d/slipstream-ci
visudo -cqf /etc/sudoers.d/slipstream-ci

cat > /etc/systemd/system/slipstream-ci-runner@.service <<'EOF'
[Unit]
Description=GitHub Actions runner %i for jatassi/SlipStream
Wants=network-online.target
After=network-online.target

[Service]
User=gha%i
WorkingDirectory=/home/gha%i/actions-runner
BindPaths=/home/gha%i:/home/runner
Environment=HOME=/home/runner
ExecStart=/home/gha%i/actions-runner/run.sh
KillMode=mixed
KillSignal=SIGINT
TimeoutStopSec=60
Restart=always
RestartSec=10
PrivateTmp=yes

[Install]
WantedBy=multi-user.target
EOF
# Each runner gets its own CPUs and an equal share of memory, less 2 GiB for
# the distro, so it runs its job like a hosted runner of that size. Go, Node
# and Playwright size their parallelism from the CPUs a process may use. A job
# that outgrows its memory is killed alone, instead of the VM running out.
cpus=$(( $(nproc) / RUNNERS ))
memory_mib=$(( ($(awk '/^MemTotal:/ {print $2}' /proc/meminfo) / 1024 - 2048) / RUNNERS ))
for i in $(seq 1 "$RUNNERS"); do
  install -d "/etc/systemd/system/slipstream-ci-runner@$i.service.d"
  cat > "/etc/systemd/system/slipstream-ci-runner@$i.service.d/limits.conf" <<EOF
[Service]
AllowedCPUs=$(( (i - 1) * cpus ))-$(( i * cpus - 1 ))
MemoryMax=${memory_mib}M
MemorySwapMax=0
EOF
done
# Each runner's memory every 2 s, to size the runners from what jobs use:
# n=current/anon in MiB, where anon is what its processes allocated and
# current also counts the file cache.
install -d -m 755 /var/log/slipstream-ci
cat > /usr/local/bin/slipstream-ci-memory <<'EOF'
#!/bin/bash
while sleep 2; do
  line=$(date -u +%FT%TZ)
  for d in /sys/fs/cgroup/system.slice/system-slipstream*runner.slice/slipstream-ci-runner@*.service; do
    [ -e "$d/memory.current" ] || continue
    n=${d%.service}
    anon=$(awk '$1 == "anon" {print $2}' "$d/memory.stat")
    line+=" ${n##*@}=$(( $(cat "$d/memory.current") / 1048576 ))/$(( anon / 1048576 ))"
  done
  echo "$line"
done >> /var/log/slipstream-ci/memory.log
EOF
chmod 755 /usr/local/bin/slipstream-ci-memory
cat > /etc/systemd/system/slipstream-ci-memory.service <<'EOF'
[Unit]
Description=Log each SlipStream CI runner's memory

[Service]
ExecStart=/usr/local/bin/slipstream-ci-memory
Restart=always

[Install]
WantedBy=multi-user.target
EOF
systemctl daemon-reload
systemctl enable --now slipstream-ci-memory > /dev/null 2>&1
systemctl restart slipstream-ci-memory

deps_installed=false
for i in $(seq 1 "$RUNNERS"); do
  user=gha$i
  dir=/home/$user/actions-runner
  id "$user" > /dev/null 2>&1 || useradd --create-home --shell /bin/bash "$user"
  usermod -aG slipstream-ci "$user"
  systemctl stop "slipstream-ci-runner@$i" 2> /dev/null || true
  install -d -o "$user" -g "$user" "/home/$user/work"
  if [ ! -f "$dir/config.sh" ]; then
    install -d -o "$user" -g "$user" "$dir"
    sudo -u "$user" tar -xzf "$PACKAGE" -C "$dir"
  fi
  if [ "$deps_installed" = false ]; then
    "$dir/bin/installdependencies.sh" > /dev/null
    deps_installed=true
  fi
  # A runner moves its settings to the _migrated files once it starts.
  rm -f "$dir/.runner" "$dir/.credentials" "$dir/.credentials_rsaparams" "$dir/.runner_migrated" "$dir/.credentials_migrated"
  cd "$dir"
  sudo -u "$user" ./config.sh --unattended --replace \
    --url https://github.com/jatassi/SlipStream \
    --token "$TOKEN" \
    --name "slipstream-linux-$i" \
    --labels "slipstream-linux,slipstream-linux-$KIT" \
    --work /home/runner/work > /dev/null
  printf 'ACTIONS_RUNNER_HOOK_JOB_STARTED=%s\nACTIONS_RUNNER_HOOK_JOB_COMPLETED=%s\n' \
    "$HOOKS/job-started.sh" "$HOOKS/job-completed.sh" | sudo -u "$user" tee "$dir/.env" > /dev/null
  systemctl enable --now "slipstream-ci-runner@$i" > /dev/null 2>&1
  echo "slipstream-linux-$i configured"
done

# Runners above the count from an earlier provisioning stop. GitHub lists
# them as offline, and the plan job ignores offline runners.
for d in /etc/systemd/system/multi-user.target.wants/slipstream-ci-runner@*.service; do
  [ -e "$d" ] || continue
  n=${d##*@}
  n=${n%.service}
  if [ "$n" -gt "$RUNNERS" ]; then
    systemctl disable --now "slipstream-ci-runner@$n" > /dev/null 2>&1 || true
    rm -f "/etc/systemd/system/slipstream-ci-runner@$n.service.d/limits.conf"
  fi
done
