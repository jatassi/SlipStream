# Provisions this PC's self-hosted runners for jatassi/SlipStream from the
# committed scripts/self-hosted, and labels them with that kit's tree hash.
# The plan job routes a commit's jobs to them only when the commit's kit has
# the same hash. Run it again after changing the kit. It's safe to run more
# than once. Run it from PowerShell 7 as yourself. The Windows half asks for
# administrator rights once.
#   pwsh scripts/self-hosted/provision.ps1 [-LinuxRunners 8] [-WindowsRunners 4] [-Only linux|windows]
param(
    [int]$LinuxRunners = 8,
    [int]$WindowsRunners = 4,
    [ValidateSet('', 'linux', 'windows')] [string]$Only = ''
)
$ErrorActionPreference = 'Stop'
$Repo = 'jatassi/SlipStream'
$Distro = 'SlipStream-CI'
$env:WSL_UTF8 = '1'

if (git -C $PSScriptRoot status --porcelain -- .) {
    throw 'scripts/self-hosted has uncommitted changes. Commit them first, because the runners are labelled with the committed kit.'
}
$kit = (git -C $PSScriptRoot rev-parse HEAD:scripts/self-hosted).Substring(0, 12)
$release = gh api repos/actions/runner/releases/latest | ConvertFrom-Json
$version = $release.tag_name.TrimStart('v')
function Get-RunnerSha([string]$Platform) {
    if ($release.body -notmatch "<!-- BEGIN SHA $Platform -->([0-9a-f]{64})") { throw "The runner release notes have no SHA-256 for $Platform." }
    $Matches[1]
}
$token = gh api -X POST "repos/$Repo/actions/runners/registration-token" --jq .token
Write-Host "Kit $kit, runner $version"

if ($Only -ne 'windows') {
    if (-not ((wsl.exe -l -q) -contains $Distro)) {
        wsl.exe --install Ubuntu-24.04 --name $Distro --location "C:\slipstream-ci\wsl" --no-launch
    }
    $stage = "\\wsl.localhost\$Distro\root\slipstream-ci-kit"
    wsl.exe -d $Distro -u root -- mkdir -p /root/slipstream-ci-kit
    Copy-Item "$PSScriptRoot\linux.sh", "$PSScriptRoot\job-started.sh", "$PSScriptRoot\job-completed.sh" $stage -Force
    wsl.exe -d $Distro -u root -- bash -c "sed -i 's/\r$//' /root/slipstream-ci-kit/*.sh && bash /root/slipstream-ci-kit/linux.sh wsl-conf"
    wsl.exe --terminate $Distro | Out-Null
    wsl.exe -d $Distro -u root -- bash /root/slipstream-ci-kit/linux.sh $LinuxRunners $kit $version (Get-RunnerSha 'linux-x64') $token
    if ($LASTEXITCODE -ne 0) { throw "Provisioning the Linux runners failed with exit code $LASTEXITCODE." }
}

if ($Only -ne 'linux') {
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
        & "$PSScriptRoot\windows.ps1" @winArgs
    } else {
        $log = Join-Path ([IO.Path]::GetTempPath()) 'slipstream-ci-provision.log'
        $argList = @('-NoProfile', '-ExecutionPolicy', 'Bypass', '-Command',
            "Start-Transcript -Force '$log' | Out-Null; try { & '$PSScriptRoot\windows.ps1' " +
            (($winArgs.GetEnumerator() | ForEach-Object { "-$($_.Key) '$($_.Value)'" }) -join ' ') +
            "; 'PROVISION-OK' } finally { Stop-Transcript | Out-Null }")
        Start-Process pwsh -Verb RunAs -Wait -ArgumentList $argList
        Get-Content $log | Select-Object -Last 25
        if (-not (Select-String -Quiet 'PROVISION-OK' $log)) { throw "Provisioning the Windows runners failed. See $log." }
    }
}

& "$PSScriptRoot\runners.ps1" status
