# Provisions the Windows runners and the WSL keep-alive task. provision.ps1
# runs it as administrator.
param(
    [Parameter(Mandatory)] [int]$Runners,
    [Parameter(Mandatory)] [string]$Kit,
    [Parameter(Mandatory)] [string]$Version,
    [Parameter(Mandatory)] [string]$Sha,
    [Parameter(Mandatory)] [string]$Token,
    [Parameter(Mandatory)] [string]$RemoveToken,
    [Parameter(Mandatory)] [string]$Owner,
    [Parameter(Mandatory)] [string]$OwnerSid,
    [Parameter(Mandatory)] [string]$PauseFlag
)
$ErrorActionPreference = 'Stop'
$Root = 'C:\slipstream-ci'
$Hooks = "$Root\hooks"
$Package = "$Root\packages\actions-runner-win-x64-$Version.zip"

# GitHub's Windows images allow long paths, and node_modules needs them.
Set-ItemProperty 'HKLM:\SYSTEM\CurrentControlSet\Control\FileSystem' LongPathsEnabled 1 -Type DWord

# Administrators write the hooks. The runner accounts only read them.
New-Item -ItemType Directory -Force $Hooks, "$Root\packages" | Out-Null
foreach ($file in 'job-started.ps1', 'job-completed.ps1', 'keepalive.ps1', 'runners.ps1') {
    Copy-Item "$PSScriptRoot\$file" "$Hooks\$file" -Force
}
icacls $Hooks /inheritance:r /grant:r '*S-1-5-32-544:(OI)(CI)F' '*S-1-5-18:(OI)(CI)F' '*S-1-5-32-545:(OI)(CI)RX' | Out-Null

if (-not (Test-Path $Package)) {
    Invoke-WebRequest "https://github.com/actions/runner/releases/download/v$Version/actions-runner-win-x64-$Version.zip" -OutFile "$Package.part"
    if ((Get-FileHash "$Package.part" -Algorithm SHA256).Hash -ne $Sha) { throw "The runner package's SHA-256 doesn't match the release." }
    Move-Item "$Package.part" $Package
}

# Jobs default to bash, which is Git Bash on GitHub's images. It goes ahead
# of System32, whose bash.exe starts WSL.
$path = "C:\Program Files\Git\bin;C:\Program Files\Git\usr\bin;" + [Environment]::GetEnvironmentVariable('Path', 'Machine')
$hidden = 'HKLM:\SOFTWARE\Microsoft\Windows NT\CurrentVersion\Winlogon\SpecialAccounts\UserList'
New-Item -Force $hidden | Out-Null

foreach ($i in 1..$Runners) {
    $name = "slipstream-windows-$i"
    $user = "slipstream-ci-$i"
    $dir = "$Root\windows-$i"
    $bytes = [byte[]]::new(24)
    [Security.Cryptography.RandomNumberGenerator]::Fill($bytes)
    $password = [Convert]::ToBase64String($bytes)
    $secure = ConvertTo-SecureString $password -AsPlainText -Force
    if (Get-LocalUser $user -ErrorAction SilentlyContinue) {
        Set-LocalUser $user -Password $secure
    } else {
        New-LocalUser $user -Password $secure -PasswordNeverExpires -UserMayNotChangePassword -AccountNeverExpires -Description 'SlipStream CI runner' | Out-Null
        Add-LocalGroupMember -Group (Get-LocalGroup -SID 'S-1-5-32-545') -Member $user
    }
    Set-ItemProperty $hidden $user 0 -Type DWord

    if (Test-Path "$dir\.runner") {
        & "$dir\config.cmd" remove --token $RemoveToken
        if ($LASTEXITCODE -ne 0) { throw "Removing $name failed with exit code $LASTEXITCODE." }
    }
    if (-not (Test-Path "$dir\config.cmd")) {
        Expand-Archive $Package -DestinationPath $dir -Force
    }
    @(
        "PATH=$path"
        "ACTIONS_RUNNER_HOOK_JOB_STARTED=$Hooks\job-started.ps1"
        "ACTIONS_RUNNER_HOOK_JOB_COMPLETED=$Hooks\job-completed.ps1"
        'GIT_CONFIG_COUNT=1'
        'GIT_CONFIG_KEY_0=core.longpaths'
        'GIT_CONFIG_VALUE_0=true'
    ) | Set-Content "$dir\.env" -Encoding ascii
    Push-Location $dir
    & .\config.cmd --unattended --replace `
        --url https://github.com/jatassi/SlipStream `
        --token $Token `
        --name $name `
        --labels "slipstream-windows,slipstream-windows-$Kit" `
        --work _work `
        --runasservice `
        --windowslogonaccount "$env:COMPUTERNAME\$user" `
        --windowslogonpassword $password
    if ($LASTEXITCODE -ne 0) { throw "Configuring $name failed with exit code $LASTEXITCODE." }
    Pop-Location

    # The owner may start and stop the service without administrator rights.
    $service = "actions.runner.jatassi-SlipStream.$name"
    $sddl = (sc.exe sdshow $service | Where-Object { $_ }) -join ''
    $ace = "(A;;RPWPLCRC;;;$OwnerSid)"
    if ($sddl -notlike "*$ace*" -and $sddl -match '^(D:.*?)(S:.*)?$') {
        sc.exe sdset $service ($Matches[1] + $ace + $Matches[2]) | Out-Null
    }
    if (Test-Path $PauseFlag) { Stop-Service $service }
}

# Runners above the count from an earlier provisioning are removed.
Get-ChildItem $Root -Directory -Filter 'windows-*' | Where-Object { [int]($_.Name -replace 'windows-') -gt $Runners } | ForEach-Object {
    if (Test-Path "$($_.FullName)\.runner") { & "$($_.FullName)\config.cmd" remove --token $RemoveToken }
    Remove-Item -Recurse -Force $_.FullName
    Remove-LocalUser "slipstream-ci-$($_.Name -replace 'windows-')" -ErrorAction SilentlyContinue
}

# Keeps the SlipStream-CI distro running, and with it the Linux runners'
# systemd services, whether or not the owner is logged on.
$task = 'SlipStream CI (WSL)'
Stop-ScheduledTask -TaskName $task -ErrorAction SilentlyContinue
$action = New-ScheduledTaskAction -Execute "$env:SystemRoot\System32\WindowsPowerShell\v1.0\powershell.exe" `
    -Argument "-NoProfile -NonInteractive -ExecutionPolicy Bypass -WindowStyle Hidden -File `"$Hooks\keepalive.ps1`" -PauseFlag `"$PauseFlag`""
$triggers = @((New-ScheduledTaskTrigger -AtStartup), (New-ScheduledTaskTrigger -AtLogOn -User $Owner))
$principal = New-ScheduledTaskPrincipal -UserId $Owner -LogonType S4U -RunLevel Limited
$settings = New-ScheduledTaskSettingsSet -MultipleInstances IgnoreNew -ExecutionTimeLimit ([TimeSpan]::Zero) `
    -AllowStartIfOnBatteries -DontStopIfGoingOnBatteries -StartWhenAvailable `
    -RestartCount 999 -RestartInterval (New-TimeSpan -Minutes 1)
Register-ScheduledTask -TaskName $task -Action $action -Trigger $triggers -Principal $principal -Settings $settings -Force | Out-Null
Start-ScheduledTask -TaskName $task
