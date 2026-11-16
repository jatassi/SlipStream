# Pauses, resumes or shows the self-hosted runners. It needs no administrator
# rights. Provisioning copies it to C:\slipstream-ci\hooks.
#   runners.ps1 pause    stop every runner. CI runs every job on hosted runners.
#   runners.ps1 resume   start the runners again.
#   runners.ps1 status
param([ValidateSet('pause', 'resume', 'status')] [string]$Action = 'status')

$Repo = 'jatassi/SlipStream'
$PauseFlag = Join-Path $env:LOCALAPPDATA 'SlipStreamCI\paused'
$Services = 'actions.runner.jatassi-SlipStream.slipstream-windows-*'

function Set-Routing([string]$Value) {
    gh variable set SELF_HOSTED_RUNNERS --repo $Repo --body $Value 2>&1 | Out-Null
    if ($LASTEXITCODE -ne 0) { Write-Warning 'Setting the SELF_HOSTED_RUNNERS variable failed. Run gh auth login and try again.' }
}

switch ($Action) {
    'pause' {
        Set-Routing 'paused'
        New-Item -ItemType File -Force $PauseFlag | Out-Null
        Get-Service $Services | Stop-Service
        & wsl.exe --terminate SlipStream-CI 2>&1 | Out-Null
        Write-Host 'The runners are paused. CI runs on hosted runners until you resume them.' -ForegroundColor Yellow
    }
    'resume' {
        Remove-Item -Force -ErrorAction SilentlyContinue $PauseFlag
        Get-Service $Services | Start-Service
        Set-Routing 'enabled'
        Write-Host 'The runners are starting. The Linux runners connect within a minute.' -ForegroundColor Green
    }
}

Write-Host ''
Write-Host "Paused: $(Test-Path $PauseFlag)"
Get-Service $Services | Sort-Object Name | ForEach-Object { Write-Host "$($_.Name -replace '^actions\.runner\.jatassi-SlipStream\.'): $($_.Status)" }
$runners = gh api "repos/$Repo/actions/runners?per_page=100" | ConvertFrom-Json
foreach ($r in $runners.runners | Sort-Object name) {
    Write-Host ("GitHub lists {0} as {1}{2}" -f $r.name, $r.status, ($r.busy ? ', busy' : ''))
}
if ($Action -ne 'status') { Start-Sleep -Seconds 3 }
