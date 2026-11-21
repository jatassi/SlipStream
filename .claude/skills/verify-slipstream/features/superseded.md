# Superseded

A Controller is Superseded once a Worker it pins carries the new Controller's proof that a Backup of its library was restored on another machine, or once an Admin stops or retires it. A Superseded Controller keeps every torrent on every Node, pauses its own, refuses every Admin operation but its own page with 409 `superseded`, and records no History row. Every page in the browser opens the Superseded page, which leads with Wipe and join as a Worker. A Worker paired with another Controller that carries no proof only raises a Nodes Health Error on the old one, which keeps running.

## Sub-features

- `fence-proven` on `dead-controller` with home left running, a Backup restored on spare with `--old-controller-off` and seedbox joined to spare: within seconds home's `GET /api/v1/status` reads `superseded` true, `GET /api/v1/superseded` reads `state` `proven` with `replacement` `{key, address}` and `worker` `seedbox`, `GET /api/v1/nodes` and every other Admin read answer 409 `{"code":"superseded","title":"This Controller was replaced"}`, and home logs `seedbox sent proof that a Backup of this library was restored on another Controller`.
- `superseded-desktop` the same run on desktop: any page on home opens `/superseded`, titled `This Controller was replaced`; signed out it reads `Sign in as an Admin to wipe this Controller and join the new one.` with Sign in, and signed in it reads `At <time>, a Backup of this library was restored on the Controller at <address>. This Controller's torrents are paused and its searches have stopped.`, the `Join Code` field with `Create a Join Code on the new Controller, then enter it here.`, `Wipe and join as a Worker`, and under Details the new Controller's key, `Reported by` seedbox and `Resume this Controller…`.
- `superseded-phone` the same page on phone, Wipe and join first.
- `wipe-and-join` a Join Code minted on spare, entered on home's page and confirmed in `Wipe this Controller?`: home restarts, its own page reads `This Node is a Worker. Manage it from its Controller.` as `home-2`, spare's `GET /api/v1/nodes` lists `home-2` `up`, and home's data dir keeps `node.key`, `local.db` and its Backups folder with a `superseded` Backup but holds no `library.db`, `library.key`, `backup.key` or `backups.json`.
- `wipe-review-list` on `dead-controller` with home given Searching and Downloading before The Quiet Harbor's Grab, so home's own client seeds it and the restored library lists it on the review list: once home-2 joins, spare logs `adopted a Node's torrents` with `"torrents":1` and `"downloading_granted":true`, the review list is empty, and `GET /api/v1/torrents` lists Harbor on home-2, `paused` with `hold` `window` and then `seeding`.
- `hostile-worker` on `hostile-worker`, office claimed as a second Controller, seedbox joined to it and home set to dial seedbox's Node address: home's `GET /api/v1/superseded` reads `unproven` with `worker` `seedbox`, `GET /api/v1/status` has no `superseded`, `GET /api/v1/nodes` answers 200, and `GET /api/v1/system/health` lists the Error `seedbox is paired with another Controller` with the other Controller's key under its detail.

## How to get to it (user POV)

- Browser: every page of a Superseded Controller opens `/superseded`. Signed out, Sign in leads back to it.
- API on the old Controller (Admin): `GET /api/v1/superseded`; `POST /api/v1/superseded/wipe` with `{"joinCode": ...}`, `POST /api/v1/superseded/resume` and `POST /api/v1/superseded/stop`, each with a reconfirmed session. `GET /api/v1/status` carries `superseded`.
- A paired Worker out of reach of its Controller for `unreachable_after` (15 minutes) logs `To pair this Node again, open <address> and enter Setup code <code>.`, and its page then takes that Setup code and a Join Code to pair again.

## Driving it with fakeworld and the host CLI

Preconditions:

- A `dead-controller` run with doctor passing, for **Fence home**, **The page** and **Wipe and join**, in that order.
- The passphrase in `$RUN/reveal/pass`, read with the helper [backups.md](./backups.md) describes, after the Backup is taken. Delete it before `fakeworld stop`.
- A separate `hostile-worker` run for **A Worker with no proof**.

Steps:

- **Fence home.** Take a Backup on home and keep home running. Restore it on the stopped spare, start spare, mint a Join Code on it and join seedbox, then read home.

  ```sh
  $FW exec $RUN --proof superseded/fence-proven -- dist/slipstream backup --data-dir {nodes.home.dataDir}
  $FW node stop $RUN spare
  $FW exec $RUN --proof superseded/fence-proven -- env -u SLIPSTREAM_TMDB_KEY dist/slipstream restore --data-dir {nodes.spare.dataDir} --passphrase-file {runDir}/reveal/pass --old-controller-off <backup>
  $FW node start $RUN spare
  $FW exec $RUN --proof superseded/fence-proven -- curl -sS -X POST -H 'X-Api-Key: {apiKey}' '{nodes.spare.url}/api/v1/join-codes'
  $FW exec $RUN --proof superseded/fence-proven -- {runDir}/bin/slipstream join --data-dir {nodes.seedbox.dataDir} <join code>
  $FW exec $RUN --proof superseded/fence-proven -- curl -sS '{nodes.home.url}/api/v1/status'
  $FW exec $RUN --proof superseded/fence-proven -- curl -sS -H 'X-Api-Key: {apiKey}' '{nodes.home.url}/api/v1/superseded'
  $FW exec $RUN --proof superseded/fence-proven -- curl -sS -H 'X-Api-Key: {apiKey}' '{nodes.home.url}/api/v1/nodes'
  ```

- **The page.** Open a session per shell, go to any page on home, screenshot the signed-out page, then click Sign in and sign in by hand (`bin/sign-in.ts` waits for the passkey prompt and the Library, which a Superseded Controller never shows). Open Details and screenshot.

  ```sh
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop goto {nodes.home.browserUrl}/nodes
  $FW exec $RUN --proof superseded/superseded-desktop -- $PW cli -s={runId}-admin-desktop screenshot --filename={runDir}/proof/superseded/superseded-desktop/signed-out.png
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop click 'getByRole("button", { name: "Sign in" })'
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop fill 'getByLabel("Email")' ada@example.test
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop click 'getByRole("textbox", { name: "PIN", exact: true })'
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop type 2468
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop click 'getByRole("button", { name: "Details" })'
  $FW exec $RUN --proof superseded/superseded-desktop -- $PW cli -s={runId}-admin-desktop screenshot --filename={runDir}/proof/superseded/superseded-desktop/admin.png
  ```

- **Wipe and join.** Mint another Join Code on spare, enter it, confirm, then read both Controllers and home's data dir.

  ```sh
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop fill 'getByLabel("Join Code")' <join code>
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop click 'getByRole("button", { name: "Wipe and join as a Worker" })'
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop click 'getByRole("dialog").getByRole("button", { name: "Wipe and join" })'
  $FW exec $RUN --proof superseded/wipe-and-join -- curl -sS -H 'X-Api-Key: {apiKey}' '{nodes.spare.url}/api/v1/nodes'
  $FW exec $RUN --proof superseded/wipe-and-join -- ls {nodes.home.dataDir} {nodes.home.dataDir}/backups
  $FW exec $RUN --proof superseded/wipe-and-join -- curl -sS '{nodes.home.url}/api/v1/status'
  ```

- **A Worker with no proof.** On `hostile-worker`, claim office as in [reconciling.md](./reconciling.md), mint its Join Code and join seedbox to it. Then make home dial seedbox and wait for the claim.

  ```sh
  $FW exec $RUN -- {runDir}/bin/slipstream node-address --data-dir {nodes.seedbox.dataDir}
  $FW exec $RUN --proof superseded/hostile-worker -- curl -sS -X PUT -H 'X-Api-Key: {apiKey}' -H 'Content-Type: application/json' -d '{"nodeAddress":"<SLN1-…>"}' '{nodes.home.url}/api/v1/nodes/<seedbox id>/dial'
  $FW exec $RUN --proof superseded/hostile-worker -- curl -sS -H 'X-Api-Key: {apiKey}' '{nodes.home.url}/api/v1/superseded'
  $FW exec $RUN --proof superseded/hostile-worker -- curl -sS '{nodes.home.url}/api/v1/status'
  $FW exec $RUN --proof superseded/hostile-worker -- curl -sS -H 'X-Api-Key: {apiKey}' '{nodes.home.url}/api/v1/system/health'
  ```

## Gotchas

- A Worker tells the old Controller only when it carries a proof. A Worker joined to a Controller no restore made says so only when the old Controller dials it, so set home's dial to seedbox's Node address first.
- In `dead-controller` home holds no Downloading, so its own paused client holds nothing to show, unless the run gives it Searching and Downloading (`PUT /api/v1/nodes/roles`) before a Grab, as `wipe-review-list` does. The Go test `TestSupersededKeepsAllAndPausesOnlyItself` proves its intent pauses every torrent.
- `bin/sign-in.ts` signs in to the Superseded home: it waits only to leave `/sign-in` and declines no passkey prompt on the Superseded page.
- History is an Admin read, so it answers 409 `superseded` too. `TestARestoreOnANewMachineFencesTheOldControllerThroughAWorker` reads home's library and proves no row was added.
- The restored Controller keeps the Backup's name, so spare lists itself as home and the wiped Node joins as `home-2`.
- The page's address is the new Controller's link address as its Worker last reached it, `127.0.0.1` behind fakeworld's front.
- The Health Error's link opens seedbox's Overview, whose "Another Controller" section offers Stop this Controller. Retire is `POST /api/v1/backups/retire`; its screen belongs to Settings > Backups.
