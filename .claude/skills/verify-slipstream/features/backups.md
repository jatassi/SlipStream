# Backups

The Controller keeps Backups of its library in its Backups folder, `backups` in the data dir unless the Node's `backups.folder` names another. `slipstream backup` takes one on the running Controller, checks that it opens and prints its size and path. A Backup is an age file sealed with a scrypt passphrase around a zstd tar, so stock `age`, `zstd` and `tar` open it with that passphrase. A Backups folder without room for the Backup refuses before anything is written, naming the folder and both sizes. A Backup destination is a folder on any Node that gets a copy of every Backup: a Low Transfer from the Controller's Backups folder that waits behind every Normal Transfer on its pair for up to `A copy waits behind downloads for up to` (24 hours), lands under the Backup's own name, and catches up when its Node comes back.

## Sub-features

- `backup-cli` `slipstream backup --data-dir <dir>` answers `Took a <size> Backup and checked it: <path>`, and the file lands in the Backups folder (P-346).
- `backup-stock-tools` the file opens with age's header, `age-encryption.org/v1`, and a `-> scrypt` stanza. `age -d` with the passphrase gives a zstd stream (`28 b5 2f fd`), and `tar t` lists `manifest.json`, `library.db`, `library.key`, `backup.key` and `backups.json`.
- `destination-add` `POST /api/v1/backup-destinations` with `nodeId` and `folder` answers 201 with the destination and `room` `null`, or a `room` of `need` and `free` bytes when the folder's volume can't hold every listed Backup. A folder that isn't there answers 422 `destination_unusable` with `refusal` `{reason: "not_found"}`, and the same folder again 409 `destination_exists`.
- `destination-copy` once added, every verified Backup is copied: `GET /api/v1/backup-destinations` counts `copies.copied` and names the `newest`, and the folder holds each Backup under its own name, the same bytes as the Controller's (`cmp`), an age file, beside a hidden `.slipstream-staging` folder.
- `destination-low` on `offline-destination` with home's send limit at 512 KB/s, a Backup taken while a Grab's import copies to vps lists its copy `queued` with `Queued behind 1 other Transfer from home.` and `destination` `{kind: "backup", backupDestinationId, folder}`, and the copy lands once the import is done.
- `destination-offline` on `offline-destination` with vps stopped, a Backup taken queues no copy and the Backup task's unit `<folder> on vps` reads `skipped`. Once vps starts, the unit runs again and the copy lands.
- `backups-screen` Settings > Backups, under System after Nodes, leads with one sentence naming the schedule, the newest good Backup and the Nodes with a copy, lists each Backup with its time, `Checked`, its kind, version and size and its copies, and on phone opens from Settings' System list.
- `back-up-now` `Back up now` toasts `Backing up…`, and the list gains a `Manual` row that reads `Checked` once the Backup is checked.
- `show-passphrase` `Show passphrase` past the re-confirm window opens `Confirm it's you`; the PIN shows the ten words in the `Backup passphrase` field with `Copy Backup passphrase`. `Download recovery sheet` asks the same way, then downloads the sheet.
- `destination-add-screen` `Add a Backup destination` with a Node and a folder lists the destination as `<Node>: <folder>`, and its first copy reads `Newest copy <time>, checked <time>.`
- `backup-full-folder` on `full-backups-folder`, with a small volume mounted on the Backups folder, `slipstream backup` exits 1 with `slipstream backup: backup: failed: <folder> has <free> free, and the Backup needs about <size>` and lists nothing in `backups.json`.
- `update-backup` on `home` launched on a copy of the release binary stamped `v2.1.0-aa`, home stopped and started by a copy stamped `v2.2.0-aa` takes an `update` Backup of that version before it is ready, and `backups.json` reads `startedVersion` `v2.2.0-aa` (P-270).
- `update-backup-full` the same data dir with a 16 KB volume mounted on the Backups folder: a copy stamped `v2.3.0-aa` exits 7, logs `start failed` with `back up library.db: <folder> has 16 KB free, and the Backup needs about <size>`, prints `Free space in <folder>, or start the previous version of SlipStream.`, and leaves `startedVersion` at `v2.2.0-aa` (P-270).

## How to get to it (user POV)

- Desktop: Settings > Backups (System, after Nodes). `Back up now` in the Backups section's header, the passphrase panel, `Add a Backup destination`, and each Backup's menu with Download, Restore… and Delete….
- Phone: Settings, then Backups in the System list. The same page, one column.
- Host CLI: `slipstream backup --data-dir <dir>` on the Controller's host, while the Controller runs.
- Node settings: `backups.folder` in the Controller's Node settings, a folder with the `backups` purpose, on Nodes > the Controller > Backups.
- API on the Controller (Admin): `GET` and `POST /api/v1/backup-destinations`, `DELETE /api/v1/backup-destinations/{id}`, and `GET /api/v1/transfers` for the copies. `GET /api/v1/backups` lists the Backups, `POST /api/v1/backups` takes one, and `GET /api/v1/backups/passphrase` needs a re-confirmed Admin session; the API key gets 403.
- Start: a start that would migrate `library.db` first takes an update Backup while status answers `backing up`. Its failure exits 7 with `Free space in <folder>, or start the previous version of SlipStream.` or `Start the previous version of SlipStream, then check the log for the cause.` A release build carries no pending migration, so a verify run drives the update Backup a newer version takes at its first start, with copies of the binary stamped newer, as **A newer version** shows.

## Driving it with fakeworld, playwright cli and the host CLI

Preconditions:

- A `home` run with doctor passing and the `{runId}-admin-desktop` and `{runId}-admin-phone` sessions, for **The screen**, **Take one** and **Open it with stock tools**.
- A separate `offline-destination` run (`--scenario offline-destination`) for **Add a destination**, **A copy waits behind a download** and **An offline destination catches up**, in that order. Read the Nodes' ids from `GET /api/v1/nodes`.
- A separate `full-backups-folder` run (`--scenario full-backups-folder`), as root on Linux, for **A full Backups folder**.

Steps:

- **The screen.** Open Settings > Backups. The lead names the schedule; press `Back up now` and wait for the `Manual` row to read `Checked`. On phone open Settings and tap Backups in the System list.

  ```sh
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop goto '{nodes.home.browserUrl}/settings/backups'
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop click 'getByRole("button", { name: "Back up now" })'
  $FW exec $RUN --proof backups/back-up-now -- $PW cli -s={runId}-admin-desktop screenshot --filename={runDir}/proof/backups/back-up-now/desktop.png
  $FW exec $RUN -- $PW cli -s={runId}-admin-phone goto '{nodes.home.browserUrl}/settings'
  $FW exec $RUN -- $PW cli -s={runId}-admin-phone click 'getByRole("list", { name: "System" }).getByRole("link", { name: "Backups" })'
  $FW exec $RUN --proof backups/backups-screen -- $PW cli -s={runId}-admin-phone screenshot --filename={runDir}/proof/backups/backups-screen/phone.png
  ```

- **Show the passphrase.** Shorten the re-confirm window and wait past it as [reconfirm.md](./reconfirm.md) does, then press `Show passphrase` and type the PIN. Keep the screenshot out of the PR: it shows the words.

  ```sh
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop click 'getByRole("button", { name: "Show passphrase" })'
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop type 2468
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop click 'getByRole("button", { name: "Download recovery sheet" })'
  ```

- **Take one.** The command prints the size and the path, and the folder holds the file.

  ```sh
  $FW exec $RUN --proof backups/backup-cli -- dist/slipstream backup --data-dir {nodes.home.dataDir}
  $FW exec $RUN --proof backups/backup-cli -- ls -l {nodes.home.dataDir}/backups
  $FW exec $RUN --proof backups/backup-cli -- cat {nodes.home.dataDir}/backups.json
  ```

- **Open it with stock tools.** Copy the path from the first step. The header names age's format and a scrypt stanza. Then write the passphrase to `$RUN/reveal/pass` with a helper in the run dir that calls `secret.ReadKeys` and `backups.OpenVault` on the data dir and prints `Passphrase(1).Reveal()`, and type it into `age -d` on a pty (age reads it only from a terminal).

  ```sh
  $FW exec $RUN --proof backups/backup-stock-tools -- head -c 120 <path>
  $FW exec $RUN --proof backups/backup-stock-tools -- python3 -I $RUN/reveal/age-open.py $RUN/reveal/pass $RUN/reveal/b.tar.zst <path>
  $FW exec $RUN --proof backups/backup-stock-tools -- od -A x -t x1 -N 4 $RUN/reveal/b.tar.zst
  $FW exec $RUN --proof backups/backup-stock-tools -- tar --zstd -tvf $RUN/reveal/b.tar.zst
  ```

- **Add a destination.** Make the folder on vps, add a missing folder, the folder, and the folder again, then wait for the first copy.

  ```sh
  $FW exec $RUN -- mkdir -p {nodes.vps.dataDir}/destination
  $FW exec $RUN --proof backups/destination-add -- curl -sS -X POST -H 'X-Api-Key: {apiKey}' -H 'Content-Type: application/json' --data-binary "{\"nodeId\":\"$VPS\",\"folder\":\"$VPSDIR/destination\"}" '{nodes.home.url}/api/v1/backup-destinations'
  $FW exec $RUN -- bash -c "for i in \$(seq 60); do curl -sS -H 'X-Api-Key: {apiKey}' '{nodes.home.url}/api/v1/backup-destinations' | jq -e '.destinations[0].copies.copied >= 1' >/dev/null && exit 0; sleep 0.5; done; exit 1"
  $FW exec $RUN --proof backups/destination-copy -- cmp {nodes.home.dataDir}/backups/<name> {nodes.vps.dataDir}/destination/<name>
  ```

- **A copy waits behind a download.** Limit home to 512 KB/s, add and search The Quiet Orchard, and take a Backup as soon as its import Transfer is listed.

  ```sh
  $FW exec $RUN -- curl -sS -X PATCH -H 'X-Api-Key: {apiKey}' -H 'Content-Type: application/json' --data-binary '{"sendLimit":524288}' "{nodes.home.url}/api/v1/transfers/nodes/$HOME_ID/settings"
  $FW exec $RUN -- bash -c "for i in \$(seq 240); do curl -sS -H 'X-Api-Key: {apiKey}' '{nodes.home.url}/api/v1/transfers' | jq -e '[.transfers[] | select(.purpose==\"import\")] | length > 0' >/dev/null && exit 0; sleep 0.5; done; exit 1"
  $FW exec $RUN --proof backups/destination-low -- dist/slipstream backup --data-dir {nodes.home.dataDir}
  $FW exec $RUN --proof backups/destination-low -- curl -sS -H 'X-Api-Key: {apiKey}' '{nodes.home.url}/api/v1/transfers'
  ```

- **An offline destination catches up.** Stop vps, take a Backup, read the task, start vps and wait for the copy.

  ```sh
  $FW node stop $RUN vps
  $FW exec $RUN --proof backups/destination-offline -- dist/slipstream backup --data-dir {nodes.home.dataDir}
  $FW exec $RUN --proof backups/destination-offline -- curl -sS -H 'X-Api-Key: {apiKey}' '{nodes.home.url}/api/v1/tasks/backup'
  $FW node start $RUN vps
  $FW exec $RUN -- bash -c "for i in \$(seq 180); do curl -sS -H 'X-Api-Key: {apiKey}' '{nodes.home.url}/api/v1/backup-destinations' | jq -e '.destinations[0].copies.copied >= 3' >/dev/null && exit 0; sleep 1; done; exit 1"
  $FW exec $RUN --proof backups/destination-offline -- ls -la {nodes.vps.dataDir}/destination
  ```

- **A full Backups folder.** Mount a 64 KB volume on the Backups folder, then take one.

  ```sh
  $FW exec $RUN -- mkdir -p {nodes.home.dataDir}/full-backups
  $FW exec $RUN -- mount -t tmpfs -o size=64k tmpfs {nodes.home.dataDir}/full-backups
  $FW exec $RUN --proof backups/backup-full-folder -- dist/slipstream backup --data-dir {nodes.home.dataDir}
  $FW exec $RUN --proof backups/backup-full-folder -- cat {nodes.home.dataDir}/backups.json
  $FW exec $RUN -- umount {nodes.home.dataDir}/full-backups
  ```

- **A newer version.** Copy `dist/slipstream` three times with its stamp's `"version":"<v>"` replaced by `v2.1.0-aa`, `v2.2.0-aa` and `v2.3.0-aa`, the same length as the build's own version, so the stamp still parses (a short Python `bytes.replace` does it). Launch `home` with `--binary` the first copy; doctor passes, since the commit and fingerprint are unchanged. Stop home, then start the second copy on its data dir for 15 s, and the third on a full Backups folder, as root.

  ```sh
  $FW node stop $RUN home
  $FW exec $RUN --proof backups/update-backup -- timeout -s INT 15 dist/slipstream-semver-b run --data-dir {nodes.home.dataDir}
  $FW exec $RUN --proof backups/update-backup -- jq -c '{startedVersion, backups:[.backups[]|{trigger,version}]}' {nodes.home.dataDir}/backups.json
  $FW exec $RUN -- mount -t tmpfs -o size=16k tmpfs {nodes.home.dataDir}/backups
  $FW exec $RUN --proof backups/update-backup-full -- dist/slipstream-semver-c run --data-dir {nodes.home.dataDir}
  $FW exec $RUN -- umount {nodes.home.dataDir}/backups
  ```

## Gotchas

- The passphrase lives sealed in `backup.key`. Read it in a signed-in Admin session on Settings > Backups, or with the helper, outside `$FW exec`, and keep it out of the journal and of screenshots that land on a PR.
- `tar --zstd` needs the `zstd` CLI. Without it, decompress with any zstd decoder first, then `tar t` the result.
- Give every `$FW exec` a path: a `head -c` with an empty path reads stdin and never ends.
- Unmount the volume before `fakeworld stop`, or the scratch dir under it can't be deleted.
- A dev build never takes the update Backup at start, since only a release version compares as newer. A start that finds the window passed takes a scheduled Backup, so the folder already holds one.
- Without a send limit the import is over in about a second on loopback, before the Backup's copy is queued.
- `$FW exec` fills every `{...}`, so `curl -w '%{http_code}'` fails as a token the manifest lacks. Write `{{http_code}` for a literal brace.
- `slipstream backup` answers only on the Controller. On a Worker's data dir it fails with `this Node isn't the Controller, so it keeps no Backups`.
