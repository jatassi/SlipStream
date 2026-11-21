# Backups

The Controller keeps Backups of its library in its Backups folder, `backups` in the data dir unless the Node's `backups.folder` names another. `slipstream backup` takes one on the running Controller, checks that it opens and prints its size and path. A Backup is an age file sealed with a scrypt passphrase around a zstd tar, so stock `age`, `zstd` and `tar` open it with that passphrase. A Backups folder without room for the Backup refuses before anything is written, naming the folder and both sizes.

## Sub-features

- `backup-cli` `slipstream backup --data-dir <dir>` answers `Took a <size> Backup and checked it: <path>`, and the file lands in the Backups folder.
- `backup-stock-tools` the file opens with age's header, `age-encryption.org/v1`, and a `-> scrypt` stanza. `age -d` with the passphrase gives a zstd stream (`28 b5 2f fd`), and `tar t` lists `manifest.json`, `library.db`, `library.key`, `backup.key` and `backups.json`.
- `backup-full-folder` on `full-backups-folder`, with a small volume mounted on the Backups folder, `slipstream backup` exits 1 with `slipstream backup: backup: failed: <folder> has <free> free, and the Backup needs about <size>` and lists nothing in `backups.json`.

## How to get to it (user POV)

- Host CLI: `slipstream backup --data-dir <dir>` on the Controller's host, while the Controller runs.
- Node settings: `backups.folder` in the Controller's Node settings, a folder with the `backups` purpose.
- Start: a start that would migrate `library.db` first takes an update Backup while status answers `backing up`. Its failure exits 7 with `Free space in <folder>, or start the previous version of SlipStream.` or `Start the previous version of SlipStream, then check the log for the cause.` Only Go tests drive it, since a release build carries no pending migration.

## Driving it with fakeworld and the host CLI

Preconditions:

- A `home` run with doctor passing, for **Take one** and **Open it with stock tools**.
- A separate `full-backups-folder` run (`--scenario full-backups-folder`), as root on Linux, for **A full Backups folder**.

Steps:

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

- **A full Backups folder.** Mount a 64 KB volume on the Backups folder, then take one.

  ```sh
  $FW exec $RUN -- mkdir -p {nodes.home.dataDir}/full-backups
  $FW exec $RUN -- mount -t tmpfs -o size=64k tmpfs {nodes.home.dataDir}/full-backups
  $FW exec $RUN --proof backups/backup-full-folder -- dist/slipstream backup --data-dir {nodes.home.dataDir}
  $FW exec $RUN --proof backups/backup-full-folder -- cat {nodes.home.dataDir}/backups.json
  $FW exec $RUN -- umount {nodes.home.dataDir}/full-backups
  ```

## Gotchas

- No screen shows the passphrase yet; it lives sealed in `backup.key`, so the run reads it with a helper. Keep it out of the journal: run the helper outside `$FW exec` and pass the file.
- `tar --zstd` needs the `zstd` CLI. Without it, decompress with any zstd decoder first, then `tar t` the result.
- Give every `$FW exec` a path: a `head -c` with an empty path reads stdin and never ends.
- Unmount the volume before `fakeworld stop`, or the scratch dir under it can't be deleted.
- A dev build never takes the update Backup at start, since only a release version compares as newer. A start that finds the window passed takes a scheduled Backup, so the folder already holds one.
- `slipstream backup` answers only on the Controller. On a Worker's data dir it fails with `this Node isn't the Controller, so it keeps no Backups`.
