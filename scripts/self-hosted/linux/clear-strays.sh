#!/usr/bin/env bash
# Kills what earlier jobs left running in this runner's service. A cancelled
# job's test binaries outlive it, and a hosted runner's VM goes away with
# them, but this service keeps them: they starved the runner until it
# couldn't hand the next job to its worker. Every process in the service is
# a job's, apart from the runner's own processes, which run this hook.
cg=$(cut -d: -f3- /proc/self/cgroup)
case "$cg" in */slipstream-ci-runner@*.service) ;; *) exit 0 ;; esac
keep=" "
pid=$$
while [ "$pid" -gt 1 ]; do
  keep+="$pid "
  pid=$(awk '/^PPid:/ {print $2}' "/proc/$pid/status")
done
strays=()
while read -r pid; do
  case "$keep" in *" $pid "*) ;; *) strays+=("$pid") ;; esac
done < "/sys/fs/cgroup$cg/cgroup.procs"
if [ "${#strays[@]}" -gt 0 ]; then
  echo "Stopping ${#strays[@]} processes earlier jobs left running"
  sudo kill -KILL "${strays[@]}" 2> /dev/null || true
fi
