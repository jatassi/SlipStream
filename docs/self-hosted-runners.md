# Self-hosted runners

The maintainer's PC runs self-hosted GitHub Actions runners for the linux-amd64 and windows-amd64 targets. CI uses them when it can and runs everything else on GitHub's hosted runners. Nothing in CI needs them.

| Pool | Runners | Where | Account |
|---|---|---|---|
| Linux | `slipstream-linux-1` to `-4` | systemd services in the `SlipStream-CI` WSL distro | `gha1` to `gha4`, with passwordless sudo |
| Windows | `slipstream-windows-1` and `-2` | services in `C:\slipstream-ci\windows-N` | local standard users `slipstream-ci-N` |

Each Linux runner has its own CPUs and an equal share of the WSL VM's memory, less 2 GiB for the distro. A job that outgrows its share is killed alone. The distro runs in UTC, as hosted runners do. Memory is what limits the count. Eight Linux jobs at once filled a 15 GB VM and hung it.

Each runner has two labels: the pool's (`slipstream-linux`) and the pool's with its kit's hash (`slipstream-linux-0a1b2c3d4e5f`). A pool's kit is `scripts/self-hosted/linux` or `scripts/self-hosted/windows`, and its hash is the first 12 characters of `git rev-parse HEAD:scripts/self-hosted/<os>`.

Each Linux runner's service sees its own home at `/home/runner`, the hosted runners' path. `actions/cache` versions a cache by its absolute paths, and the Bun cache holds paths relative to the workspace, so with any other home no cache a hosted runner saved would restore on these runners. The distro takes its clock from Windows with NTP off, because the two corrected each other every few seconds when Windows was a minute off. Windows has to keep its own clock in time.

## Which jobs they take

The plan job's "Find the self-hosted runners" step counts the online runners whose label carries the hash of the commit's kit, and passes them to `release plan -self-hosted`. It counts none in any of these cases:

- someone other than `jatassi` started or re-ran the run, or a pull request comes from a fork
- the `SELF_HOSTED_RUNNERS` repository variable is `paused`
- the `RUNNER_STATUS_TOKEN` secret is missing. It's a fine-grained token for this repository with Administration read-only, because the workflow token can't list runners.

The planner moves up to one job per runner onto a pool. The lanes that start with the run take runners first. The e2e and browser lanes start when the build ends, so they get the build's runner and whatever is left. A packaging build always stays on its hosted runner, because its smoke tests install the packages system-wide. Every other job runs on the target's hosted runner, as it would without the pools.

The plan job itself runs on a hosted runner, so a run falls back to hosted runners when this machine is off. A dispatch with `plan-self-hosted` runs the plan and the required check on a Linux runner instead, and skips the hosted queue. It waits while the Linux runners are offline.

A commit that changes a pool's kit gets no runners from that pool until the pool is provisioned from it. Everything else, including new tools and new versions of the ones CI uses, needs no provisioning. CI installs them in every job, as it does on hosted runners.

## Who can run jobs on them

1. The repository requires approval to run workflows from every outside contributor.
2. Each runner runs `job-started.sh` or `job-started.ps1` before every job. It fails the job before its first step unless `jatassi` started the run from this repository, so a fork's workflow can't use the runners even if it names their labels.
3. Fork pull requests get no secrets, so their plan job finds no runners.

The Linux runners have sudo, as on hosted runners. WSL doesn't mount the Windows drives or let Linux start Windows programs, but root can mount a drive by hand. A compromised dependency in one of the maintainer's own runs could read the maintainer's files. The Windows runners' accounts can't read the maintainer's profile.

## Provisioning

From PowerShell 7 in a checkout with the kit committed:

```powershell
pwsh scripts/self-hosted/provision.ps1
```

It sets the WSL VM's memory to 24 GB in `%USERPROFILE%\.wslconfig`, which every distro shares, and shuts WSL down when that changes. It installs the `SlipStream-CI` distro if it's missing, registers 4 Linux and 2 Windows runners with the kit's hash, removes the registrations of runners above those counts, and asks for administrator rights once for the Windows half. Run it again after changing the kit. `-LinuxRunners`, `-WindowsRunners`, `-WslMemory` and `-Only linux|windows` change what it does.

The "SlipStream CI (WSL)" scheduled task keeps the distro running from startup, whether or not anyone is logged on.

## Pausing

```powershell
pwsh C:\slipstream-ci\hooks\runners.ps1 pause    # or resume, or status
```

Pausing sets `SELF_HOSTED_RUNNERS` to `paused`, stops the Windows services and stops the distro. A job that is running at the time fails. The pause lasts across restarts until `resume`. While the PC sleeps, the runners are offline and CI uses hosted runners. A job already queued on a runner waits for it to come back.

## Removing them

From an administrator PowerShell:

```powershell
$token = gh api -X POST repos/jatassi/SlipStream/actions/runners/remove-token --jq .token
Get-ChildItem C:\slipstream-ci -Directory -Filter 'windows-*' | ForEach-Object { & "$($_.FullName)\config.cmd" remove --token $token }
Get-LocalUser 'slipstream-ci-*' | Remove-LocalUser
Unregister-ScheduledTask 'SlipStream CI (WSL)' -Confirm:$false
wsl --unregister SlipStream-CI
Remove-Item -Recurse -Force C:\slipstream-ci
```

Then delete the Linux runners, which are offline by then, under Settings, Actions, Runners.
