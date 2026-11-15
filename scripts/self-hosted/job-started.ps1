# Runner pre-job hook: refuse any job not triggered by the repository owner
# from the repository itself (no forks). A non-zero exit fails the job before
# any workflow step runs.
$ErrorActionPreference = 'Stop'

$Owner = 'jatassi'
$Repo = 'jatassi/SlipStream'

function Deny([string]$Reason) {
    Write-Output "::error::Self-hosted runner refused this job: $Reason"
    exit 1
}

if ($env:GITHUB_REPOSITORY -ne $Repo) { Deny "repository '$env:GITHUB_REPOSITORY' is not $Repo" }
if ($env:GITHUB_ACTOR -ne $Owner) { Deny "actor '$env:GITHUB_ACTOR' is not $Owner" }
if ($env:GITHUB_TRIGGERING_ACTOR -ne $Owner) { Deny "triggering actor '$env:GITHUB_TRIGGERING_ACTOR' is not $Owner" }
if (-not $env:GITHUB_EVENT_PATH -or -not (Test-Path $env:GITHUB_EVENT_PATH)) { Deny 'event payload missing' }

try {
    $event = Get-Content -Raw -Encoding UTF8 $env:GITHUB_EVENT_PATH | ConvertFrom-Json
} catch {
    Deny 'event payload unreadable'
}

if ($event.sender.login -ne $Owner) { Deny 'sender is not the owner' }
if ($event.pull_request -and $event.pull_request.head.repo.full_name -ne $Repo) {
    Deny 'pull request head is not in the repository (fork)'
}
if ($event.workflow_run -and $event.workflow_run.head_repository.full_name -ne $Repo) {
    Deny 'workflow_run head is not in the repository (fork)'
}

Write-Output "Job authorized for $Owner on $Repo"
exit 0
