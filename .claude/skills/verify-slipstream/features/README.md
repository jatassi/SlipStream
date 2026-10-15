# SlipStream verification map

This directory is the maintained source for verifying what a SlipStream user or script can observe. Each feature file is one user-facing feature, named in [GLOSSARY.md](../../../../GLOSSARY.md) words, with the recipe that proves it. Read this index, then the feature file for what you changed.

## Baseline

- The `home` Scenario launched from a `go run ./cmd/release build` binary, with `RUN`, `FW` and `PW` set as in the skill's Launch.
- `$FW doctor $RUN` passing all six checks.
- The `{runId}-admin-desktop` session open, signed in as the manifest's Admin (Ada Okafor, `ada@example.test`) and on the Library, as in the skill's Drive.
- A feature that needs another Scenario names it in its Preconditions. `first-run` is a fresh Controller before setup.

## Conventions

- Start each recipe from the baseline unless its Preconditions say otherwise. Recipes that change the library or a Node (search-and-grab, pairing, live-updates) want a fresh run, or a run where they come last.
- Run every command through `$FW exec $RUN` or a fakeworld verb, from the worktree root, so the journal and `summary.md` hold it.
- Target the UI by role and accessible name. Treat every quoted name and command as literal.
- Address the API and the browser through manifest tokens: `{nodes.home.url}`, `{nodes.home.browserUrl}`, `{apiKey}`.
- Recipes are written for `bash`; quote `{...}` tokens the way they appear.

## Proof standards

- Capture the action and the resulting state: an ARIA snapshot and a screenshot before and after, under `proof/<feature>/<sub-feature>/`.
- Prove each side effect where it lands: `fs match` for files, `announces` for the tracker, `sse` for the live stream, and an API read for stored state.
- Drive the real user path. Cues and fault rules act on the world outside every Node and never on a Node's database or files.
- Record the sub-feature id and the entry point with each proof by its proof dir.
- Report an entry point you couldn't drive with the command you tried and the precondition it lacked, never as verified through another one.

## Feature entry contract

Each feature file starts with an H1 title and one paragraph describing the user-visible behaviour, then exactly four H2s in this order:

1. `Sub-features` lists each behaviour as a bullet that starts with its unique `` `id` `` and cites the `P-###` parity rows it proves.
2. `How to get to it (user POV)` lists every entry point, per shell where desktop and phone differ, plus the API and host CLI.
3. `Driving it with <harness>` starts with `Preconditions:` and pairs each user action with exact commands and the observable result.
4. `Gotchas` lists traps that waste or invalidate a run.

Run `bun .claude/skills/verify-slipstream/bin/check-map.ts` after any edit here.

## Features

- [Setup](./setup.md) covers the Setup code from the Controller's log, a wrong code, and creating the first Admin on `first-run`.
- [Sign-in](./sign-in.md) covers Email and PIN, a wrong PIN, returning to the page asked for, saved state and sign-out.
- [Pairing](./pairing.md) covers a Join Code, wiping the seedbox with `fakeworld node`, `slipstream join` and the Worker page.
- [Add a movie](./add-a-movie.md) covers Library, Add movie, catalog search, adding a Work and the Work page.
- [Search and grab](./search-and-grab.md) covers a 429 from the Indexer, Search, a held download, completing it now, the Transfer, import, Activity and Files.
- [This Node](./this-node.md) covers the This Node page and `GET /api/v1/system`.
- [Live updates](./live-updates.md) covers the live status, reconnecting after `fakeworld node kill` and `start`, and the captured stream.
