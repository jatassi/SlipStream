# Restore

A Backup restores onto a machine that has none of the old Controller's files, onto an Unclaimed Node from its first-run page, or in place over the running Controller. Each way stages the Backup first, refuses with one sentence when it can't be restored, and asks to confirm that the old Controller stays off when the Backup is of another machine. The Node then restarts or starts, swaps the library, and begins Reconciling: History records Backup restored, and Controller replaced on a new machine, and every Node keeps every torrent until its Holdings are adopted.

## Sub-features

- `dead-controller` on `dead-controller`, with home killed, `slipstream restore --data-dir <spare> --passphrase-file <file> <backup>` refuses with `this Backup is of home, another machine. Make sure home stays off, then run slipstream restore again with --old-controller-off to confirm it`, and with `--old-controller-off` prints `Prepared the restore of the Backup from <day>, <time>. Start SlipStream to finish it.` Started, spare answers `GET /api/v1/status` `controller`, lists home as self and seedbox `down` in `GET /api/v1/nodes` with the Backup's API key, and History reads `backup_restored` and `controller_replaced`. Once seedbox joins spare by `slipstream join`, spare logs `adopted a Node's torrents` with `"torrents":2` for seedbox and lists both torrents.
- `first-run-upload` on `dead-controller`, spare's `POST /api/v1/setup/restore` with the Setup code refuses a JSON body with 415 `not_an_upload` and keeps an `application/octet-stream` upload, answering 201 with its id. `POST /api/v1/setup/restore/<id>/open` answers 422 `backup_wrong_passphrase` for a wrong passphrase, and the summary for the right one: `takenAt`, `version`, `controller` `home`, `nodes` home then seedbox, `newMachine` true. `POST /api/v1/setup/restore/<id>/confirm` answers 422 `old_controller_on` without `oldControllerOff`, then 202, and spare restarts as the Controller with the Backup kept in its Backups folder as `restored`.
- `in-place-after-grab` on `restore-in-place-after-grab`, a Backup taken before a Grab, then `slipstream restore` on the running home prints `Prepared the restore of the Backup from <day>, <time>. SlipStream is restarting to finish it.` home comes back with a `before-restore` Backup in its folder, History reads `backup_restored`, and seedbox's Holdings are adopted with the newer torrent still seeding.

## How to get to it (user POV)

- Host CLI: `slipstream restore [--data-dir <dir>] [--passphrase-file <file>] [--old-controller-off] [--remove <Node name>]... <backup>`. With no Node running on the data dir it stages offline and the next start finishes it; with one running it hands the restore to that Node, which restarts. Without `--passphrase-file` it asks for the passphrase on the terminal.
- API: `POST /api/v1/setup/restore` (the file as `application/octet-stream`), then `POST /api/v1/setup/restore/{id}/open` with `{"passphrase": ...}` and `POST /api/v1/setup/restore/{id}/confirm` with `{"oldControllerOff": ..., "remove": [...]}`, each with `X-Setup-Code`.
- Docker: `SLIPSTREAM_RESTORE=<backup>`, `SLIPSTREAM_RESTORE_PASSPHRASE_FILE=<file>` and `SLIPSTREAM_RESTORE_OLD_CONTROLLER_OFF=1` on a data dir no Node set up restore in that same start. On a data dir already set up the start logs `SlipStream ignores SLIPSTREAM_RESTORE because this data dir is already set up`. Only Go tests drive it.

## Driving it with fakeworld and the host CLI

Preconditions:

- A `dead-controller` run (`--scenario dead-controller`) with doctor passing, for **Restore a dead Controller** and, in a second run, **Restore from the first-run page**.
- A `restore-in-place-after-grab` run with doctor passing, for **Restore in place**.
- The passphrase in `$RUN/reveal/pass`, read with the helper [backups.md](./backups.md) describes, after the Backup is taken. Delete it before `fakeworld stop`.

Steps:

- **Restore a dead Controller.** Add and search both movies on home, wait until `GET /api/v1/torrents` lists two `seeding`, then take a Backup, kill home and stop spare.

  ```sh
  $FW exec $RUN --proof restore/dead-controller -- dist/slipstream backup --data-dir {nodes.home.dataDir}
  $FW node kill $RUN home
  $FW node stop $RUN spare
  $FW exec $RUN --proof restore/dead-controller -- env -u SLIPSTREAM_TMDB_KEY dist/slipstream restore --data-dir {nodes.spare.dataDir} --passphrase-file {runDir}/reveal/pass <backup>
  $FW exec $RUN --proof restore/dead-controller -- env -u SLIPSTREAM_TMDB_KEY dist/slipstream restore --data-dir {nodes.spare.dataDir} --passphrase-file {runDir}/reveal/pass --old-controller-off <backup>
  $FW node start $RUN spare
  $FW exec $RUN --proof restore/dead-controller -- curl -sS '{nodes.spare.url}/api/v1/status'
  $FW exec $RUN --proof restore/dead-controller -- curl -sS -H 'X-Api-Key: {apiKey}' '{nodes.spare.url}/api/v1/nodes'
  $FW exec $RUN --proof restore/dead-controller -- curl -sS -H 'X-Api-Key: {apiKey}' '{nodes.spare.url}/api/v1/history?limit=5'
  ```

  Then mint a Join Code on spare and join seedbox to it; spare's log then reads `adopted a Node's torrents` for seedbox.

  ```sh
  $FW exec $RUN --proof restore/dead-controller -- curl -sS -X POST -H 'X-Api-Key: {apiKey}' '{nodes.spare.url}/api/v1/join-codes'
  $FW exec $RUN --proof restore/dead-controller -- {runDir}/bin/slipstream join --data-dir {nodes.seedbox.dataDir} <join code>
  $FW exec $RUN --proof restore/dead-controller -- curl -sS -H 'X-Api-Key: {apiKey}' '{nodes.spare.url}/api/v1/torrents'
  ```

- **Restore from the first-run page.** In a fresh `dead-controller` run, take a Backup on home and kill it. Read spare's Setup code from `$RUN/nodes/spare.log` (`setup code`). Write the open bodies to `$RUN/reveal/open.json` and `$RUN/reveal/wrong.json` outside `$FW exec`.

  ```sh
  $FW exec $RUN --proof restore/first-run-upload -- curl -sS -X POST -H 'X-Setup-Code: <code>' -H 'Content-Type: application/json' -d '{"path":"/etc/passwd"}' '{nodes.spare.url}/api/v1/setup/restore'
  $FW exec $RUN --proof restore/first-run-upload -- curl -sS -X POST -H 'X-Setup-Code: <code>' -H 'Content-Type: application/octet-stream' --data-binary @<backup> '{nodes.spare.url}/api/v1/setup/restore'
  $FW exec $RUN --proof restore/first-run-upload -- curl -sS -X POST -H 'X-Setup-Code: <code>' -H 'Content-Type: application/json' -d @{runDir}/reveal/open.json '{nodes.spare.url}/api/v1/setup/restore/<id>/open'
  $FW exec $RUN --proof restore/first-run-upload -- curl -sS -X POST -H 'X-Setup-Code: <code>' -H 'Content-Type: application/json' -d '{"oldControllerOff":true}' '{nodes.spare.url}/api/v1/setup/restore/<id>/confirm'
  $FW exec $RUN --proof restore/first-run-upload -- ls {nodes.spare.dataDir}/backups
  ```

- **Restore in place.** Take a Backup on home, then add and search The Quiet Orchard and wait for it to seed on seedbox. Restore with home running.

  ```sh
  $FW exec $RUN --proof restore/in-place-after-grab -- env -u SLIPSTREAM_TMDB_KEY dist/slipstream restore --data-dir {nodes.home.dataDir} --passphrase-file {runDir}/reveal/pass <backup>
  $FW exec $RUN --proof restore/in-place-after-grab -- curl -sS -H 'X-Api-Key: {apiKey}' '{nodes.home.url}/api/v1/torrents'
  $FW exec $RUN --proof restore/in-place-after-grab -- ls {nodes.home.dataDir}/backups
  ```

## Gotchas

- A host command loads the Node's config from its environment too, so a `SLIPSTREAM_` variable your shell exports that SlipStream doesn't know, such as an API key for a script, stops it with `env: <name>: unknown key`. Unset each one with `env -u` in the step.
- spare's manifest row keeps `claim` `unclaimed` after the restore; `GET /api/v1/status` is the truth.
- The restored Controller row keeps the Backup's name, so spare lists itself as home.
- After the join, seedbox's torrents read `paused` with `hold` `window` for a while: they wait for the tracker's announce window, as in [reconciling.md](./reconciling.md).
- The restart after an online restore cuts Health's evaluations, which log `evaluate a Health category` at ERROR with `context canceled`. `$FW exec` names them; they aren't the restore's failure.
- No route lists Backups yet; read `backups.json` and the Backups folder.
