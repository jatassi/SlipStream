# Prowlarr sync routine

The weekly scheduled task that keeps SlipStream's ported site indexers in step with Prowlarr (spec S27, Upstream sync). Jackson creates the scheduled task with everything under **Prompt** as its prompt. Change the routine by changing this file in a PR, then pasting the new prompt into the task.

## Prompt

Weekly Prowlarr sync for github.com/jatassi/SlipStream. This is a fresh run. Read the state from GitHub and the tool, and don't reuse anything from earlier in this conversation. Goal: every upstream change to the Prowlarr code a port came from reaches SlipStream as one draft PR per change set, which ports the diff, bumps the manifest and updates the fixtures, so Jackson only reviews and merges. Jackson (GitHub `jatassi`) reviews and merges. Never merge yourself.

Load pstack:poteto-mode with the Skill tool first and follow it alongside these steps. Use the GitHub MCP tools (mcp__github__*) for all GitHub access. There is no gh CLI. Work only in this cloud container (Bash, the local clone, GitHub tools). Never use mcp__remote-devices__* tools, Remote Control, or anything on Jackson's computers. This run must work when they're off.

### Hard rules

- Never merge, push to `main`, or force-push a branch you didn't create in this run.
- One draft PR per change set. Never fold two change sets into one PR, and never stack one routine PR on another.
- Port from Prowlarr only. Never read or copy Jackett, which is GPL-2.0-only (AGENTS.md).
- Keep every `// SlipStream:` line and the behaviour it marks. When an upstream hunk conflicts with one, keep SlipStream's line and say so in the PR body.
- Never skip, disable or loosen a test, a fixture case, a lint rule or a check, and never edit a golden by hand. Goldens change only through `-update`, and every changed golden is explained by the upstream diff.
- Never use `git stash` or any command that touches another worktree.
- At most five PRs in one run. The rest wait for next week and are named in the log comment.

### 1. Read the state

Find the open issue titled "Prowlarr sync routine log". If there is none, open it with the body "Each comment is one run of the weekly Prowlarr sync routine (docs/agents/prowlarr-sync-routine.md)." The newest comment starting with `prowlarr-sync-run` is the previous run.

Start from the latest `origin/main`. Clone Prowlarr without blobs, then write the report both ways:

```sh
git clone --quiet --filter=blob:none --no-checkout https://github.com/Prowlarr/Prowlarr "$TMPDIR/prowlarr"
go run ./cmd/prowlarr-sync report --prowlarr "$TMPDIR/prowlarr" --json > "$TMPDIR/report.json"
go run ./cmd/prowlarr-sync report --prowlarr "$TMPDIR/prowlarr" > "$TMPDIR/report.md"
```

If `report` fails because a manifest disagrees with its code, that is a bug on `main`. Open one draft PR that fixes it with `prowlarr-sync check`'s output in its body, record the run, and stop.

Search open PRs whose body contains "Opened by the weekly Prowlarr sync routine". Each one's title is `prowlarr-sync: <key>`. A change set's key is `<subject>@<short sha>`, and the subject is everything before the `@`.

- A change set whose key is an open PR's key already has its PR. Skip it.
- A change set whose subject is an open PR's subject, at another sha, waits for that PR. Skip it, and name it in the log as waiting on that PR.

If no change set is left, comment on the log issue `prowlarr-sync-run <run start time> ref=<report ref> outcome=no-change` with one sentence and the list of open routine PRs, and stop.

### 2. Port each change set

Take the change sets in the report's order. For each, branch `prowlarr-sync/<subject>-<short sha>` from `origin/main`, with each `:` in the subject written as `-`. Read the change set's commits, files table and diff in `report.md`, and the port author's README at `internal/ports/README.md`.

- **upstream, a port** (`port:<id>`). Port each hunk into the Go files the files table maps it to, line for line, with the vocabulary map in `internal/ports/port`'s package doc. A hunk in an added C# file gets a new Go file, mapped in the unit's `upstream.json`. Run `go run ./cmd/prowlarr-sync bump --prowlarr "$TMPDIR/prowlarr" --unit <unit> --to <to>`. Run `go test ./fakes/site/ports -run 'TestPorts/<id>' -update` and read every changed `want/*.json`. When an upstream commit also changed Prowlarr's recording under `src/NzbDrone.Core.Test/Files/Indexers/`, replace the fixture's body with the new recording through the scrub rewrite first.
- **upstream, a family** (`family:<name>`). As for a port, then re-run the cases of every port whose `upstream.json` names that family, and read each changed golden.
- **upstream, the kit or a shared parser** (`kit`, `prowlarr:<name>`). As for a port, then re-run every port's cases (`go test ./fakes/site/ports -update`) and the Cardigann runner's tests for a shared parser.
- **new-site**. Run `go run ./cmd/prowlarr-sync new --prowlarr "$TMPDIR/prowlarr" --site <name> --auth <login>` (`--auth` as the README says, and `--family <name>` for a site on a family base), translate each C# file into its Go file, and fill the fixture as the README says. A site whose media is one v2 has no Content Type for is still ported. A generic engine or a usenet indexer is added to `internal/ports/skipped.json` with why, instead of being ported.
- **obsolete** and **deleted**. Delete the port's package, its `internal/ports/all/<id>.go` and its fixtures, and add a `port.Retired` record to `internal/ports/all/all.go` whose reason sentence follows WRITING.md and says why SlipStream no longer supports the site.
- **moved**. Find the site's definition in the Cardigann catalog and add a runner-tier test for it against a `fixture-site` fake. When it passes, add a `port.Moved` record with the port's field map to `internal/ports/all/all.go` and delete the port, as the spec's Lifecycle section says. When it fails, keep the port, and the PR body says what fails.
- **hidden-list**. For each added or removed id, check the `Replaces` and `Aliases` of the port for that site, and change them to match. Run `go run ./cmd/prowlarr-sync bump --prowlarr "$TMPDIR/prowlarr" --unit hidden-list --to <to>`.

Then, on every branch:

1. `go run ./cmd/prowlarr-sync check --prowlarr "$TMPDIR/prowlarr"`, `make test` and `go run ./cmd/release lint`. Fix what your change broke. When a failure is a pre-existing flake, say so in the PR with the evidence.
2. Run the deslop and no-comments skills on the diff, and read every new or changed string against WRITING.md.
3. Commit, push, and open a DRAFT PR to `main` titled `prowlarr-sync: <key>`. The body follows `.github/pull_request_template.md`. Its Summary opens with what upstream changed, as the commits table with each commit linked at `https://github.com/Prowlarr/Prowlarr/commit/<sha>`, then what changed in SlipStream, every hunk kept or changed on purpose with why, and the goldens that changed and why. Its Verify section runs verify-slipstream when the change is observable to an Admin, or says "No observable change" and why. The body ends with "Opened by the weekly Prowlarr sync routine; log in #<log issue>." Assign it to jatassi, request his review, and subscribe to its PR activity.

### 3. Record

Comment on the log issue `prowlarr-sync-run <run start time> ref=<report ref> outcome=<prs-opened|no-change|waiting>`, then a table with one row per change set in the report. Each row has its key, its kind, and the PR it opened, the PR it waits on, or why it was left for next week. The next run reads this comment.

Then post one short reply in this thread with how many change sets the report had, the draft PRs opened with their links, and what waits. Keep it under 80 words and link the log issue for detail.
