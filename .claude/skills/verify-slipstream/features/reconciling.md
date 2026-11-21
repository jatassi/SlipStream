# Reconciling

A newly paired Node keeps every torrent its client holds until the Controller adopts them. While it waits, the Controller's Nodes page reads it `keeping` for Downloading, never `draining`, and its Nodes Health item links to Reconciling. Once its Holdings are in, the Controller records each torrent it doesn't know as adopted with its folder, and a Node whose list held torrents gains the Downloading Role in the same Write. The restore's own Reconciling, its page and its banner land in later PRs.

## Sub-features

- `repair-seeding` on `repair`, 40 movies added on home and searched each leave seedbox seeding 40 torrents: home's `GET /api/v1/torrents` lists 40 rows `seeding` on seedbox.
- `repair-keeps` office claimed as a second Controller and seedbox joined to it with office's Join Code by `slipstream join`: office's `GET /api/v1/nodes` reads seedbox `downloading:keeping` until its Holdings are adopted, then `downloading:holding`, never `draining`, and office's `GET /api/v1/torrents` lists the same 40 torrents on seedbox, seeding or briefly paused by the announce window, none removed.

## How to get to it (user POV)

- Desktop shell: the Nodes page, seedbox's Downloading cell; Health's Nodes item and its `Open Reconciling` link while seedbox keeps everything.
- API: `GET /api/v1/nodes` (`roles[].state` `keeping`), `GET /api/v1/torrents`, `GET /api/v1/system/health`.
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

## Gotchas

- Adopting 40 torrents takes well under a second, so a read after the join usually sees `holding` already. Poll from before the join to catch `keeping`; the in-process proof that holds Metainfo until it looks is `TestANewPairingKeepsEverythingUntilAdopted` in `fakes/world/sim`.
- An adopted torrent lists `contentType` `""` and `tracker` null on office: SlipStream adopted it without a Grab.
- Right after the restart a few torrents read `paused` with `hold` `window`, waiting for the tracker's announce window. They resume on their own.
- `{…}` in an `exec` argument is a manifest token, so curl's `-w '%{http_code}'` fails there.
