# Provisions this PC's self-hosted runners for jatassi/SlipStream from the
# committed kit. The Linux runners are labelled with the tree hash of
# scripts/self-hosted/linux and the Windows runners with that of
# scripts/self-hosted/windows. The plan job routes a commit's jobs to a pool
# only when the commit's folder for it has the same hash, so only a change to
# a folder needs the pool provisioned again. It's safe to run more than once.
# Run it from PowerShell 7 as yourself. The Windows half asks for
# administrator rights once.
#   pwsh scripts/self-hosted/provision.ps1 [-LinuxRunners 4] [-WindowsRunners 2] [-WslMemory 24GB] [-Only linux|windows]
param(
    [int]$LinuxRunners = 4,
    [int]$WindowsRunners = 2,
    # The WSL VM's memory, which the Linux runners share. WSL defaults to half
    # of the PC's memory.
    [string]$WslMemory = '24GB',
    [ValidateSet('', 'linux', 'windows')] [string]$Only = ''
)
$ErrorActionPreference = 'Stop'
$Repo = 'jatassi/SlipStream'
$Distro = 'SlipStream-CI'
$env:WSL_UTF8 = '1'

function Get-Kit([string]$Pool) {
    if (git -C $PSScriptRoot status --porcelain -- $Pool) {
        throw "scripts/self-hosted/$Pool has uncommitted changes. Commit them first, because the runners are labelled with the committed kit."
    }
    (git -C $PSScriptRoot rev-parse "HEAD:scripts/self-hosted/$Pool").Substring(0, 12)
}
$release = gh api repos/actions/runner/releases/latest | ConvertFrom-Json
$version = $release.tag_name.TrimStart('v')
function Get-RunnerSha([string]$Platform) {
    if ($release.body -notmatch "<!-- BEGIN SHA $Platform -->([0-9a-f]{64})") { throw "The runner release notes have no SHA-256 for $Platform." }
    $Matches[1]
}
$token = gh api -X POST "repos/$Repo/actions/runners/registration-token" --jq .token

if ($Only -ne 'windows') {
    $kit = Get-Kit linux
    Write-Host "Linux kit $kit, runner $version"
    # .wslconfig sets the VM every distro shares, and WSL reads it only when
    # the VM starts, so a change shuts every distro down.
    $wslConfig = Join-Path $env:USERPROFILE '.wslconfig'
    $before = if (Test-Path $wslConfig) { Get-Content -Raw $wslConfig } else { '' }
    $after = if ($before -match '(?m)^\s*memory\s*=') {
        $before -replace '(?m)^\s*memory\s*=.*$', "memory=$WslMemory"
    } elseif ($before -match '(?m)^\[wsl2\]') {
        $before -replace '(?m)^\[wsl2\]\s*$', "[wsl2]`nmemory=$WslMemory"
    } else {
        $before.TrimEnd() + "`n[wsl2]`nmemory=$WslMemory`n"
    }
    if ($after.Trim() -ne $before.Trim()) {
        Set-Content -Path $wslConfig -Value $after.TrimStart() -NoNewline
        Write-Host "Set memory=$WslMemory in $wslConfig. Shutting WSL down to apply it."
        wsl.exe --shutdown
    }

    if (-not ((wsl.exe -l -q) -contains $Distro)) {
        wsl.exe --install Ubuntu-24.04 --name $Distro --location "C:\slipstream-ci\wsl" --no-launch
    }
    # Without interop there's no \\wsl.localhost share, so the files go in on
    # the command line.
    foreach ($file in 'linux.sh', 'job-started.sh', 'job-completed.sh') {
        $b64 = [Convert]::ToBase64String([IO.File]::ReadAllBytes("$PSScriptRoot\linux\$file"))
        wsl.exe -d $Distro -u root -- bash -c "mkdir -p /root/slipstream-ci-kit && echo $b64 | base64 -d | tr -d '\r' > /root/slipstream-ci-kit/$file"
    }
    wsl.exe -d $Distro -u root -- bash /root/slipstream-ci-kit/linux.sh wsl-conf
    wsl.exe --terminate $Distro | Out-Null
    wsl.exe -d $Distro -u root -- bash /root/slipstream-ci-kit/linux.sh $LinuxRunners $kit $version (Get-RunnerSha 'linux-x64') $token
    if ($LASTEXITCODE -ne 0) { throw "Provisioning the Linux runners failed with exit code $LASTEXITCODE." }
}

if ($Only -ne 'linux') {
    $kit = Get-Kit windows
    Write-Host "Windows kit $kit, runner $version"
    $me = [Security.Principal.WindowsIdentity]::GetCurrent()
    $winArgs = @{
        Runners     = $WindowsRunners
        Kit         = $kit
        Version     = $version
        Sha         = Get-RunnerSha 'win-x64'
        Token       = $token
        RemoveToken = gh api -X POST "repos/$Repo/actions/runners/remove-token" --jq .token
        Owner       = $me.Name
        OwnerSid    = $me.User.Value
        PauseFlag   = Join-Path $env:LOCALAPPDATA 'SlipStreamCI\paused'
    }
    New-Item -ItemType Directory -Force (Split-Path $winArgs.PauseFlag) | Out-Null
    if ([Security.Principal.WindowsPrincipal]::new($me).IsInRole([Security.Principal.WindowsBuiltInRole]::Administrator)) {
        & "$PSScriptRoot\windows\windows.ps1" @winArgs
    } else {
        $log = Join-Path ([IO.Path]::GetTempPath()) 'slipstream-ci-provision.log'
        $argList = @('-NoProfile', '-ExecutionPolicy', 'Bypass', '-Command',
            "Start-Transcript -Force '$log' | Out-Null; try { & '$PSScriptRoot\windows\windows.ps1' " +
            (($winArgs.GetEnumerator() | ForEach-Object { "-$($_.Key) '$($_.Value)'" }) -join ' ') +
            "; 'PROVISION-OK' } finally { Stop-Transcript | Out-Null }")
        Start-Process pwsh -Verb RunAs -Wait -ArgumentList $argList
        Get-Content $log | Select-Object -Last 25
        if (-not (Select-String -Quiet 'PROVISION-OK' $log)) { throw "Provisioning the Windows runners failed. See $log." }
    }
}

# Runners above the counts from an earlier provisioning are offline. Their
# registrations go too.
$counts = @{ linux = $LinuxRunners; windows = $WindowsRunners }
$registered = (gh api "repos/$Repo/actions/runners?per_page=100" | ConvertFrom-Json).runners
foreach ($r in $registered) {
    if ($r.name -match '^slipstream-(linux|windows)-(\d+)$' -and ($Only -eq '' -or $Only -eq $Matches[1]) -and [int]$Matches[2] -gt $counts[$Matches[1]]) {
        gh api -X DELETE "repos/$Repo/actions/runners/$($r.id)" | Out-Null
        Write-Host "Removed the registration of $($r.name)."
    }
}

& "$PSScriptRoot\windows\runners.ps1" status
