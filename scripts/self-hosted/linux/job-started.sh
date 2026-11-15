#!/usr/bin/env bash
# Runner pre-job hook: refuse any job not triggered by the repository owner
# from the repository itself (no forks). A non-zero exit fails the job before
# any workflow step runs.
set -euo pipefail

OWNER="jatassi"
REPO="jatassi/SlipStream"

deny() { echo "::error::Self-hosted runner refused this job: $1"; exit 1; }

[ "${GITHUB_REPOSITORY:-}" = "$REPO" ] || deny "repository '${GITHUB_REPOSITORY:-}' is not $REPO"
[ "${GITHUB_ACTOR:-}" = "$OWNER" ] || deny "actor '${GITHUB_ACTOR:-}' is not $OWNER"
[ "${GITHUB_TRIGGERING_ACTOR:-}" = "$OWNER" ] || deny "triggering actor '${GITHUB_TRIGGERING_ACTOR:-}' is not $OWNER"
[ -f "${GITHUB_EVENT_PATH:-}" ] || deny "event payload missing"

python3 - "$GITHUB_EVENT_PATH" "$OWNER" "$REPO" <<'PY' || deny "event payload check failed"
import json, sys
path, owner, repo = sys.argv[1:]
event = json.load(open(path, encoding="utf-8"))

def fail(msg):
    print(msg)
    sys.exit(1)

if (event.get("sender") or {}).get("login") != owner:
    fail("sender is not the owner")
pr = event.get("pull_request")
if pr and ((pr.get("head") or {}).get("repo") or {}).get("full_name") != repo:
    fail("pull request head is not in the repository (fork)")
wr = event.get("workflow_run")
if wr and (wr.get("head_repository") or {}).get("full_name") != repo:
    fail("workflow_run head is not in the repository (fork)")
PY

echo "Job authorized for $OWNER on $REPO"
