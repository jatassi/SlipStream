# SlipStream verification map

This directory is the maintained source for verifying what a SlipStream user or script can observe. Each feature file is one user-facing feature, named in [GLOSSARY.md](../../../../GLOSSARY.md) words, with the recipe that proves it. Read this index, then the feature file for what you changed.

## Baseline

- The `home` Scenario launched from a `go run ./cmd/release build` binary, with `RUN`, `FW` and `PW` set as in the skill's Launch.
- `$FW doctor $RUN` passing all six checks.
- The `{runId}-admin-desktop` session open, signed in as the manifest's Admin (Ada Okafor, `ada@example.test`) and on the Library, as in the skill's Drive.
- A feature that needs another Scenario names it in its Preconditions. `first-run` is a fresh Controller before setup, and `serving` is `home` with Serving on seedbox too.

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

- [Shell](./shell.md) covers the desktop sidebar and top bar, the phone tab bar and More sheet, the palette, toasts, titles, errors and not-found, the portal and the startup sequence.
- [Setup](./setup.md) covers the Setup code from the Controller's log, a wrong code, the setup link that fills it in, and creating the first Admin on `first-run`.
- [Sign-in](./sign-in.md) covers Email and PIN for an Admin and a Member, a wrong PIN and the wait it earns, the per-address limit, returning to the page asked for, saved state and sign-out.
- [Signup links](./signup-links.md) covers issuing a Signup link, its email, the pending list, a dead link and joining on desktop and phone.
- [Passkeys](./passkeys.md) covers the prompt after sign-in, adding one with Chromium's virtual authenticator, signing in with it, the deep link and the IP address that can't hold one.
- [Sessions](./sessions.md) covers the devices list on Your Account, signing out one or every other device, and an Admin ending a person's sessions.
- [Re-confirm](./reconfirm.md) covers `Confirm it's you` past the window, cancel, a wrong PIN and the API key that skips it.
- [Reset links](./reset-links.md) covers sending a Reset link, the open tab it signs out, setting a new PIN, its email and `slipstream admin reset` on the host.
- [Accounts](./accounts.md) covers the Accounts list, making an Admin a Member and the re-gated tab, the last Admin, turning an Account off and the Member's portal.
- [Settings](./settings.md) covers Settings > Address, Email with its test email, and Sign-in.
- [API key](./api-key.md) covers Settings > API key, regenerating it and the old key refused.
- [Pairing](./pairing.md) covers a Join Code, the Add a Node dialog and its open codes, wiping the seedbox with `fakeworld node`, `slipstream join` and the Worker page.
- [Roles](./roles.md) covers the Role board, a Node's Roles section with its Review and write-off dialog, planning and applying a set of Roles, the handback to home, a stale or blocked set, probes, reach tests, Notifying, drain and write-off.
- [Offline Nodes](./offline.md) covers the offline grace on the API and Settings > Nodes, the Nodes nav alert and its dot, each Node's standing, an Indexer waiting for its offline home, the Controller covering Searching past the grace, and a Grab that waits for a Node that downloads.
- [Metadata providers](./metadata-providers.md) covers each provider's status with its summary and Details, an Admin key refused or tested and kept, a provider test, removing the key, and a release's bundled keys.
- [Metadata settings](./metadata-settings.md) covers Settings > Metadata with its language and region sentence and Details, the metadata settings API, `works.type-changed` on a new region, and clearing every holder's cache.
- [Add a movie](./add-a-movie.md) covers the Library's Add menu, catalog search, adding a Work and the Work page.
- [Artwork](./artwork.md) covers the artwork URLs the Work API hands out, each size served immutable, the poster and backdrop prefetched at add, a miss answered until its retry, a forged token refused and the route kept to signed-in Accounts.
- [Movie availability](./movie-availability.md) covers each movie Item's availability date and basis on the Work API, the movie settings with their `works.type-changed` event, and the availability timer's announced day.
- [Search and grab](./search-and-grab.md) covers a 429 from the Indexer, Search, a held download, completing it now, the Transfer, import, Activity and Files.
- [Targets and Profiles](./targets-and-profiles.md) covers the starter Target and default Profile, the Target and Profile editors, the add sheet's Profile, one file filling two Targets, an upgrade, the Work's Profile and Editions sheet, the Impact with Delete and Keep, size refusals, a file refused by its probed video codec, and custom cuts in Settings > Editions.
- [Library roots](./library-roots.md) covers Settings > Library roots, adding a root on seedbox through its folder browser with a refusal in words, an import on that Worker by hard link, the Storage block, `Check storage every` and the Storage Inspector, a drain and write-off that mark the root `Node written off`, re-homing it onto home, and deleting a root while its folder stays.
- [Nodes](./nodes.md) covers the Nodes landing with its Diagram and its Roster, renaming and removing a Node, a Node's Overview, its sections and Node picker, Web and API and Log settings, the live Logs and Download logs, Restart, `GET /api/v1/system`, and Serving on a Worker: its address serving the Controller's app and the status card while the Controller can't be reached.
- [Tasks](./tasks.md) covers the Tasks page, a task's detail page, Run now and Cancel, the header capsule and the phone's running list, cadence and Schedule settings, and Housekeeping going Partial and back when seedbox stops and starts.
- [Metadata refresh](./metadata-refresh.md) covers the Metadata refresh task listed once a Work exists, its tiers and changes switch under `When it runs`, a TMDB change refreshed by the next run, refresh all over the API with 409 `task_running` while a run is open, and a Work gone upstream failing with its cause.
- [Live updates](./live-updates.md) covers the reconnecting line, reconnecting after `fakeworld node kill` and `start`, and the captured stream.
