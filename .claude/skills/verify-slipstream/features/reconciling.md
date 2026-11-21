# Reconciling

A newly paired Node keeps every torrent its client holds until the Controller adopts them. While it waits, the Controller's Nodes page reads it `keeping` for Downloading, never `draining`, and its Nodes Health item links to Reconciling. Once its Holdings are in, the Controller records each torrent it doesn't know as adopted with its folder, and a Node whose list held torrents gains the Downloading Role in the same Write. While a restore's Reconciling runs, every gate refuses with 409 `reconciling` naming what it waits on, `GET /api/v1/reconciling` lists the restore, each Node and each root, and the re-home step's Save all starts each saved root's scan. Every admin page opens the re-home step until Save all or Later, then carries a banner naming what SlipStream is doing and the next step, which opens the Reconciling page.

## Sub-features

- `repair-seeding` on `repair`, 40 movies added on home and searched each leave seedbox seeding 40 torrents: home's `GET /api/v1/torrents` lists 40 rows `seeding` on seedbox.
- `repair-keeps` office claimed as a second Controller and seedbox joined to it with office's Join Code by `slipstream join`: office's `GET /api/v1/nodes` reads seedbox `downloading:keeping` until its Holdings are adopted, then `downloading:holding`, never `draining`, and office's `GET /api/v1/torrents` lists the same 40 torrents on seedbox, seeding or briefly paused by the announce window, none removed.

- `restore-refusal` on `home` held Reconciling for a restore onto a new machine: `POST /api/v1/works/{id}/search` answers 409 `reconciling` with the detail "SlipStream is matching the restored library with what your Nodes hold. It's waiting for the old Controller's folders, seedbox and 1 library root. Try again once it finishes."
- `restore-page` the same run: `GET /api/v1/reconciling` lists the restore (`newMachine` and `rehome` true), home and seedbox with their `link`, and the Movies root by its label, not saved; `GET /api/v1/counts` has `reconciling` `{"rehome":true,"nodes":["seedbox"],"roots":1}`.
- `restore-rehome` `POST /api/v1/reconciling/rehome` with the Movies root and `done`: the root reads `refused` null, the Library scan's last run is `event` and succeeded for that root, and `GET /api/v1/reconciling` reads it saved and scanned with `rehome` false.
- `rehome-save-anyway` on `dead-controller` restored on spare from its first-run page, with The Quiet Orchard's folder moved out of the Movies root by `fakeworld fs mv`: Save all on the re-home step reads `0 of 1 folder is at the same path on this machine.` and `1 file isn't in this folder.` with `Save anyway, forget 1 file`, which lands on `/reconciling` with the root `Scanned.` and the review list's `1 torrent has its files on disk. Re-add it to keep seeding from those files.`
- `rehome-desktop` the same run on desktop: `/library` opens "Find the old Controller's folders" with "The folder is at the same path on this machine. Save to start the scans." and the Movies root's "The folder is there."; Save all lands on `/reconciling`, which reads "Pair seedbox again and wait for 1 library root to finish scanning." with seedbox's "Create Join Code" and "Continue without seedbox".
- `rehome-phone` the same step on phone, Save all above Later.
- `banner-desktop` after Save all, `/library` carries the Reconciling banner: "SlipStream is matching the restored library with the torrents and files on your Nodes. Searches and removals wait until it finishes." then "Pair seedbox again.", and Open Reconciling.
- `banner-phone` the same banner as a card above the tab bar that opens the Reconciling page.
- `page-desktop` hovering Reconciling in the lead shows its definition; Create Join Code opens "Pair seedbox again" with the code and, under Details, `slipstream join <code>`.
- `review-readd` `POST /api/v1/reconciling/torrents/readd` with `{"reason":"not_held","found":"files"}` and `/forget` with `{"reason":"not_held","found":"no_files"}` each answer 204.
- `end-phone` Continue without seedbox on the phone page: the page reads only seedbox, "SlipStream stopped waiting for seedbox. It keeps its torrents until it reports them.", and the desktop `/library` carries no banner.
- `restore-end` `POST /api/v1/reconciling/nodes/{seedbox}/continue`: home logs `Reconciling finished`, `GET /api/v1/reconciling` reads `restore` null with seedbox still keeping its torrents (`continued` true), `counts.reconciling` is null, and the search no longer answers `reconciling`.
- `restore-waits-end` on the `rehome-save-anyway` run, once seedbox joins spare and is adopted, Reconciling ends without `continue`: History reads `reconciling_finished`, and the RSS sync that waited runs at once, its Grab of the forgotten Orchard `imported` within a second.

## How to get to it (user POV)

- Desktop shell: any admin page while Reconciling, through the re-home step and the banner's Open Reconciling; the Nodes page, seedbox's Downloading cell; Health's Nodes item and its `Open Reconciling` link while seedbox keeps everything.
- Phone shell: the banner card above the tab bar opens Reconciling.
- API: `GET /api/v1/nodes` (`roles[].state` `keeping`), `GET /api/v1/torrents`, `GET /api/v1/system/health`. After a restore, `GET /api/v1/reconciling`, `POST /api/v1/reconciling/rehome`, `/nodes/{id}/continue`, `/roots/{id}/continue`, `/finish`, `GET /api/v1/reconciling/torrents`, `POST /api/v1/reconciling/torrents/readd` and `/forget`, and `GET /api/v1/counts`'s `reconciling`.
- Host CLI: `slipstream join --data-dir <dir> <code>` on the running seedbox, which re-pairs it live.

## Driving it with fakeworld and the host CLI

Preconditions:

- A `repair` run launched and doctor passing, with `RUN` and `FW` set as in the skill's Launch.

Steps:

- **Seed 40 torrents on seedbox.** Add each movie, 9300001 to 9300040, then search each Work. Wait until `GET /api/v1/grabs` reads 40 `imported`.

  ```sh
  $FW exec $RUN --proof reconciling/repair-seeding -- curl -sS -o /dev/null -X POST -H 'X-Api-Key: {apiKey}' -H 'Content-Type: application/json' -d '{"contentType":"movie","key":"9300001"}' '{nodes.home.url}/api/v1/works'
  $FW exec $RUN --proof reconciling/repair-seeding -- curl -sS -o /dev/null -X POST -H 'X-Api-Key: {apiKey}' '{nodes.home.url}/api/v1/works/<workId>/search'
  $FW exec $RUN --proof reconciling/repair-seeding -- curl -sS -H 'X-Api-Key: {apiKey}' '{nodes.home.url}/api/v1/torrents'
  ```

- **Claim office and mint its Join Code.** Read office's Setup code from `$RUN/nodes/office.log` (`setup_code`), claim it as a Controller with a cookie jar in the run dir, regenerate its API key and mint a code.

  ```sh
  $FW exec $RUN --proof reconciling/repair-keeps -- curl -sS -i -c {runDir}/office.cookies -X POST -H 'X-Setup-Code: <code>' -H 'Content-Type: application/json' -d '{"email":"olu@example.test","displayName":"Olu Bello","pin":"1357"}' '{nodes.office.url}/api/v1/setup/controller'
  $FW exec $RUN --proof reconciling/repair-keeps -- curl -sS -b {runDir}/office.cookies -X POST '{nodes.office.url}/api/v1/api-key'
  $FW exec $RUN --proof reconciling/repair-keeps -- curl -sS -X POST -H 'X-Api-Key: <office key>' '{nodes.office.url}/api/v1/join-codes'
  ```

- **Join seedbox to office.** It prints `joined office as seedbox (node <id>), linked to <address>` and restarts.

  ```sh
  $FW exec $RUN --proof reconciling/repair-keeps -- {runDir}/bin/slipstream join --data-dir {nodes.seedbox.dataDir} <join code>
  ```

- **Watch seedbox keep, then hold.** Poll office's `GET /api/v1/nodes` every 50 ms from before the join: seedbox reads `downloading:keeping` for under a second, then `downloading:holding`. Then read office's `GET /api/v1/nodes` and `GET /api/v1/torrents` with `--proof`.

- **Hold home Reconciling for a restore.** Until the restore itself lands, stop home, write the rows a restore onto a new machine writes into its stopped `library.db`, and start it again. The write is test-only: a few lines of Go on `modernc.org/sqlite`, built into `$RUN/bin/begin`, that run `INSERT INTO reconciling_state (id, restore_id, started_at, cause, backup_name, backup_taken_at, new_machine) VALUES (1, 'live-check', <now µs>, 'restore', 'nightly', <now µs>, 1)`, then one `reconciling_node` row (`cause` `restore`, `round` 1) per `nodes_node` row and one `reconciling_root` row per `roots_root` row.

  ```sh
  $FW node stop $RUN home
  $FW exec $RUN --proof reconciling/restore-begin -- {runDir}/bin/begin {nodes.home.dataDir}/library.db
  $FW node start $RUN home
  ```

- **For the screens, sign in first and stop seedbox too.** `bin/sign-in.ts` waits for the Library heading, which the re-home step replaces, so open and sign in each browser session before holding home. Stop seedbox before home so it reads as still paired with the old Controller (`Pair seedbox again`); left running, it reconnects and reads `Wait for seedbox to report its torrents.` A Node script on `node:sqlite` writes the same rows as `begin`; delete the `reconciling_node` rows pairing left first.

- **Read the refusal and the page**, then Save all and continue without seedbox.

  ```sh
  $FW exec $RUN --proof reconciling/restore-refusal -- curl -sS -X POST -H 'X-Api-Key: {apiKey}' '{nodes.home.url}/api/v1/works/<any id>/search'
  $FW exec $RUN --proof reconciling/restore-page -- curl -sS -H 'X-Api-Key: {apiKey}' '{nodes.home.url}/api/v1/reconciling'
  $FW exec $RUN --proof reconciling/restore-rehome -- curl -sS -X POST -H 'X-Api-Key: {apiKey}' -H 'Content-Type: application/json' -d '{"roots":[{"rootId":"<root id>"}],"done":true}' '{nodes.home.url}/api/v1/reconciling/rehome'
  $FW exec $RUN --proof reconciling/restore-end -- curl -sS -i -X POST -H 'X-Api-Key: {apiKey}' '{nodes.home.url}/api/v1/reconciling/nodes/<seedbox id>/continue'
  ```

## Gotchas

- seedbox didn't report its Holdings for the held restore, even after `$FW node restart $RUN seedbox`, so `restore-end` continues without it. The cause isn't found yet, and the restore's own PR checks it.

- Adopting 40 torrents takes well under a second, so a read after the join usually sees `holding` already. Poll from before the join to catch `keeping`; the in-process proof that holds Metainfo until it looks is `TestANewPairingKeepsEverythingUntilAdopted` in `fakes/world/sim`.
- An adopted torrent lists `contentType` `""` and `tracker` null on office: SlipStream adopted it without a Grab.
- Right after the restart a few torrents read `paused` with `hold` `window`, waiting for the tracker's announce window. They resume on their own.
- `{…}` in an `exec` argument is a manifest token, so curl's `-w '%{http_code}'` fails there.
