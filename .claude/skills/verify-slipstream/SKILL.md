---
name: verify-slipstream
description: Prove a SlipStream change the way a user or a script sees it, by driving the release binary in a Fake world through fakeworld, playwright cli, curl and the slipstream host CLI, and collect the evidence for the PR's Verify section. Use before a PR that changes anything observable in the UI, the API, the host CLI or between Nodes, or to reproduce a bug end to end.
---

# Verify SlipStream

A verify run launches a Fake world (`fakeworld`) against the worktree's release binary, drives one or more features from [features/README.md](features/README.md) the way a user would, and leaves a run dir that holds the proof, the journal of every command and `summary.md`. Read the README, then the feature file for what your change touches. The references hold the detail each surface needs:

- [references/browser.md](references/browser.md) for `playwright cli` sessions, locators, screenshots, traces and video.
- [references/api.md](references/api.md) for `curl` with the API key and the captured SSE stream.
- [references/host-cli.md](references/host-cli.md) for `slipstream` commands and `fakeworld node`.
- [references/world.md](references/world.md) for fronts, faults, captures, `fs`, replay, summary and the four isolation rules.

Run every command from the worktree root. The run dir lives under `.verify/`, which git ignores.

## Launch

Build once per worktree change. `release build` stamps the binary with the worktree's fingerprint, which doctor checks.

```sh
go run ./cmd/release deps fetch      # once per machine: the pinned native libraries and ffmpeg
go run ./cmd/release build           # dist/slipstream with the web app embedded; installs web/node_modules
eval "$(go run ./cmd/release deps env)" && go build -o dist/fakeworld ./cmd/fakeworld
bun run --cwd web e2e:install        # the browsers web/'s pinned Playwright drives
```

On macOS, run `sudo dist/fakeworld setup-loopback` once per machine, which aliases 127.0.0.2 to 127.0.0.32. Linux needs nothing.

Start the world and name the three handles every recipe uses:

```sh
RUN=.verify/runs/$(date +%Y%m%d-%H%M%S)-search-and-grab
dist/fakeworld start --binary dist/slipstream --scenario home --run-dir $RUN
FW=$RUN/bin/fakeworld
PW=web/node_modules/.bin/playwright
```

`start` returns once every Node is ready and prints `$RUN/manifest.json`. It exits non-zero with the reason when launch fails, and `$RUN/world.log` and `$RUN/nodes/*.log` say more. Every listener binds port 0, so two runs in one worktree share nothing. `$FW` is the run's own copy of fakeworld, and the rest of this skill uses it.

`home` is the default Scenario: the Controller `home` and the Worker `seedbox`, which holds Searching and Downloading. `first-run` is a fresh Controller before setup. A feature file names its Scenario.

## Doctor

```sh
$FW doctor $RUN
```

Six read-only checks, one `ok` line each, exit 0 when all pass: the run is ours, the build matches the worktree, the cluster matches the Scenario, every fake answers, every address an Admin entered is a fake's front, and the API key works. Run it after launch, after a drive that surprised you, and after any `fakeworld node` action. A failure names its fix. `worktree changed since the build` means rebuild and relaunch: the run would prove an old binary. Fault rules you set are listed as active faults, not failures.

## Drive

Every action is one journaled step: `$FW exec $RUN [--proof <feature>/<sub-feature>] -- <command>`. exec fills `{path}` tokens from the manifest (`{runId}`, `{runDir}`, `{apiKey}`, `{mediaDir}`, `{nodes.home.browserUrl}`, `{nodes.seedbox.dataDir}`), runs the command with no shell, keeps its output under `journal/` or the proof dir, and names any ERROR a Node logged during the step. World actions (`fault`, `node`, `announces`, `sse`, `fs`) are fakeworld verbs, which journal themselves. A step that isn't journaled can't be replayed or summarised, so run nothing against the world outside these two forms.

Open the browser once per identity and shell, signed in through the real page:

```sh
$FW browser $RUN
$FW exec $RUN -- bun web/harness/bin/sign-in.ts {runDir} admin desktop
$FW exec $RUN -- $PW cli -s={runId}-admin-desktop open --config={runDir}/browser/desktop/playwright-cli.json
$FW exec $RUN -- $PW cli -s={runId}-admin-desktop state-load {runDir}/auth/admin-desktop.json
$FW exec $RUN -- $PW cli -s={runId}-admin-desktop goto {nodes.home.browserUrl}/
```

The session name carries the run id, so another agent's sessions in the same worktree stay apart. Swap `desktop` for `phone` for the phone shell. `{nodes.home.browserUrl}/` redirects to `/library`. Then follow the feature file's recipe. Target elements by role and accessible name, scoped to a landmark where a name repeats, as in `click 'getByRole("main").getByRole("button", { name: "Search", exact: true })'`; snapshot refs change on every snapshot.

Every new tab plays the 4.4s startup sequence over the page. Verify sessions start past it: `fakeworld browser` writes `browser/startup-played.js` (a copy of `web/harness/startup-played.js`) and names it as each config's `initScript`, which marks every tab as played before the app loads, as the browser tier's fixtures and `bin/sign-in.ts` do. To see the sequence on purpose, open a session from a copy of the config without the init script; the tab then plays it once, and any key, click or tap skips it:

```sh
jq 'del(.browser.initScript)' $RUN/browser/desktop/playwright-cli.json > $RUN/browser/desktop/startup-on.json
$FW exec $RUN -- $PW cli -s={runId}-startup-desktop open --config={runDir}/browser/desktop/startup-on.json
```

[features/shell.md](features/shell.md) drives it from there.

## Evidence

Proof lives under `$RUN/proof/<feature>/<sub-feature>/`. A proof step:

- Captures the state before and after the action: an ARIA snapshot (`snapshot --filename={runDir}/proof/<f>/<s>/<name>.aria.yml`) and a screenshot (`screenshot --filename={runDir}/proof/<f>/<s>/<name>.png`), desktop and phone where the feature file lists a phone entry point.
- Proves each side effect on the receiving side: the placed file with `fs match`, announces through `announces`, the live stream through `sse`, stored state through an API read with `--proof`.
- Drives the real user path. Cues act only on the world outside SlipStream: fakes, fault rules and machine events. Nothing writes a Node's database or files.
- Reports a feature it couldn't reach with the command it tried and the precondition it lacked, never as verified through another path.

Traces: wrap each recipe in `tracing-start` and `tracing-stop`; each pair writes its own trace under `$RUN/browser/output/traces/`. Record video only for a recipe that proves motion ([references/browser.md](references/browser.md)).

Delivery. Paste `$RUN/summary.md` into the PR's Verify section. Show a few screenshots and MP4s to the maintainer inline in your reply, or publish them on one private claude.ai Artifact when there are more than a few, and put only that link in the PR. Media stays out of the PR and the repo: no `gh pr edit --attach`, nothing under `.verify/` committed.

## Cleanup

```sh
$FW exec $RUN -- $PW cli -s={runId}-admin-desktop close    # each session the run opened; $PW cli list shows them
$FW stop $RUN
$FW summary $RUN
ls $RUN/proof $RUN/summary.md
```

`stop` ends the supervisor, every Node and every fake, and deletes the scratch dir that held the Nodes' data dirs and the media. The run dir stays: `proof/`, `summary.md`, `journal/`, `events.jsonl`, `world.log`, `nodes/*.log` and the traces. Close only the sessions your run opened; `close-all` and `kill-all` would close other agents' browsers. Stop only through `fakeworld stop` and `fakeworld node`, which act on the manifest's PIDs. Run cleanup after a failed attempt too, then relaunch.

`summary` is a pure function of the run dir, so it works after `stop`, and `fakeworld replay $RUN --to N` starts a fresh world and runs steps 1 to N again ([references/world.md](references/world.md)).

## Helpers

- `bun .claude/skills/verify-slipstream/bin/check-map.ts` checks the feature map: the README's index against the files, each file's H1, paragraph and four H2s, unique sub-feature ids, and that every cited `P-###` row exists and is neither out of scope nor excepted. It prints shipped parity rows no sub-feature cites and exits 1 on a problem. Run it after editing anything under `features/`.
- `bun web/harness/bin/sign-in.ts <run> <identity> <shell>` signs the identity in through the real sign-in page, past the startup sequence, and writes `<run>/auth/<identity>-<shell>.json` for `state-load`.
- `$FW` with no arguments prints every verb, including the capture readers.
- `web/harness/fakeworld.ts` (`fw(runDir)`) drives the same verbs from TypeScript, for a script too long for the shell.
