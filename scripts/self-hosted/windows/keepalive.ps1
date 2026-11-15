# Keeps the SlipStream-CI distro running unless the runners are paused. The
# "SlipStream CI (WSL)" task starts it at startup. WSL stops a distro once no
# Windows process is attached to it, which would stop the runners.
param([Parameter(Mandatory)] [string]$PauseFlag)

while ($true) {
    if (-not (Test-Path $PauseFlag)) {
        & "$env:SystemRoot\System32\wsl.exe" -d SlipStream-CI -u root -- sleep infinity
    }
    Start-Sleep -Seconds 30
}
