# Home

An Admin's Home reads the moving Grabs nearest Available first, what needs the Admin (failed Grabs and Grabs that wait), and the newest finished Imports in words, with today's totals from the browser's midnight. Each Downloading Node's Torrents are counted for its card. `Rows on Home` sets how many rows each list shows. `/` opens the page: the status sentence, Needs you, Jobs tinting the page, then Recent beside Downloading, Health and Storage, one column on phone. It reads only those bounded lists, never the Grabs, torrents, Library or Transfers.

## Sub-features

- `home-summary` `GET /home?since=` answers `jobs` (its rows and `moving`, every moving Grab), `needsYou` (rows newest first, `failed` counted up to 100, `waiting`) and `recent` (each Import's `words`, such as `Imported 10 episodes · 2 upgrades`, and `today`'s `imported` and `upgraded`), each row with its Work's label, artwork URLs and `failed`; a job's `torrent` is null until the .torrent is fetched.
- `home-counts` `GET /torrents/counts` lists each Downloading Node in Node order with `torrents`, `downloading`, `seeding`, `failures` and `warnings`.
- `home-page` `/` opens Home, titled `Home · SlipStream`, as the sidebar's first row, the phone's first tab and the palette's `Go to Home`. Opening it requests `GET /home`, `GET /torrents/counts` and bounded reads (counts, Health, Health's volumes, nodes, roots, content types, Housekeeping), never `GET /grabs`, `/torrents`, `/works` or `/transfers`, and it opens in about half a second on `scale`'s 20,000 Works (P-018, P-347).
- `home-status` the status sentence reads `Nothing moving` or `<n> jobs moving` with a live dot, `<n> jobs paused while Reconciling` during a restore, then `<n> failures` (`100+ failures` at the cap) linking to the first failure, `<n> waiting for you` linking to Needs you and Health's worst as `<Category> warning` or `<Category> error` linking to Home's Health; above 1360px `Today: <n> imported · <n> upgraded` sits at its right, leaving out a zero and hidden at both zero.
- `home-needs-you` Needs you, left out while nothing needs the Admin, shows the first three failed Grabs as full failure cards, the rest as Signal rows, each waiting Grab as its Work page's Callout and the pending imports row, with `See all <n> in Activity` past `Rows on Home`.
- `home-jobs` Jobs lists every moving Grab nearest Available first with its Work, poster and release, stage track, progress and menu, a Node column only while two Nodes download, `See all <n> jobs` past the rows, and `No jobs are moving. Open a Work and choose Search to find a release for it.` when none moves; a stage change moves its row with no reload.
- `home-tint` the top Job whose Work hasn't failed tints the page from its backdrop; on desktop a row under the pointer or focus takes the tint after a short pause and gives it back a moment after leaving; a failed Work or one with no backdrop leaves the page neutral.
- `home-recent` Recent lists the newest Imports as `<Work>` over `<words> · <age>`, each linking to its Work, with `See all in History` to History's imports; empty, it reads `Nothing was imported in the last <Keep history for>. Each download or folder Import shows here once its files are in the library.`
- `home-downloading` Downloading, left out with no Downloading Node, shows every Node's rates while bytes move, the `Every download` bar, `All downloads are paused.` when paused, and one line per Downloading Node such as `seedbox · 3 downloading · 1 seeding` or `home · no torrents`, its failures and warnings linking to Torrents' problems for that Node; the top bar shows the same rates above 1360px while bytes move.
- `home-health` Health shows the Health page's verdict and one line per category with a problem, such as `Storage · 1 error`, or Health's words for a check that couldn't run, each linking to `/health#<category>` (P-217).
- `home-storage` Storage shows one meter per volume Health watches, grouped by Node with its dot, the worst Node and volume first, a level's chip and colour, and `Library roots` to Settings.
- `home-settings` `GET /home-settings` lists `Rows on Home`; `PATCH /home-settings` with `{"values":{"home.rows":3}}` saves it, cuts every list of `GET /home` to 3 while the counts stay whole, and publishes `home.settings-changed`; a value out of range answers 422 at its key.

## How to get to it (user POV)

- Desktop shell: `/`, the sidebar's `Home`, the brand `SlipStream`, the palette's `Go to Home`, `Back to Home` from the portal and `Go to Home` on a dead address. `Home options` › `Rows on Home…` opens the `Home` dialog.
- Phone shell: the tab bar's `Home`, the first tab; `Home options` sits beside the large title.
- API (Admin, Controller only): `GET /api/v1/home?since=<RFC 3339>`, `GET /api/v1/torrents/counts`, `GET /api/v1/home-settings`, `PATCH /api/v1/home-settings`. A Worker answers 409 `worker_node`.
- Host CLI: none.

## Driving it with fakeworld and playwright cli

Preconditions:

- The `home` Scenario, doctor passing, after one Grab made as in [Search and grab](./search-and-grab.md), for the API steps.
- For the page: `single-machine` (one Downloading Node, so no Node column), `showcase` (real posters and backdrops, for the tint), `fault-loop` (a failed Grab in Needs you), `low-space` (a Storage error), `low-space-warning` (a Storage warning) and `scale` (20,000 Works, for the open time), each with `{runId}-admin-desktop` and `{runId}-admin-phone` signed in.

Steps:

- **Open Home.** On `single-machine`, `/` reads `Home`, `Nothing moving`, the Jobs and Recent empty sentences and `home · no torrents`. Record the API requests while it opens: none is `/grabs`, `/torrents`, `/works` or `/transfers`.

  ```sh
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop goto '{nodes.home.browserUrl}/'
  $FW exec $RUN --proof home/home-page -- $PW cli -s={runId}-admin-desktop run-code 'async page => { const reads = []; page.on("request", (r) => { const u = new URL(r.url()); if (u.pathname.startsWith("/api/")) reads.push(u.pathname); }); const t = Date.now(); await page.reload(); await page.getByRole("heading", { name: "Home", level: 1 }).waitFor({ timeout: 30000 }); const ms = Date.now() - t; await page.waitForLoadState("networkidle"); return { ms, reads: [...new Set(reads)].sort() }; }'
  $FW exec $RUN --proof home/home-page -- $PW cli -s={runId}-admin-desktop screenshot --filename={runDir}/proof/home/home-page/empty.png
  ```

- **Watch a Job move.** Stall the seeders, add and search The Quiet Orchard, and the row reads `Downloading` with `1 job moving`; clear the fault and it reads `Importing`, then leaves Jobs while Recent gains `The Quiet Orchard` `Imported in WEBDL-1080p · just now` and the right of the sentence reads `Today: 1 imported`.

  ```sh
  $FW fault set $RUN swarm-stalled seeders stall
  $FW exec $RUN --proof home/home-jobs -- curl -sS -X POST -H 'X-Api-Key: {apiKey}' -H 'Content-Type: application/json' -d '{"contentType":"movie","key":"9000001","preset":"all"}' '{nodes.home.url}/api/v1/works'
  $FW exec $RUN --proof home/home-jobs -- curl -sS -X POST -H 'X-Api-Key: {apiKey}' '{nodes.home.url}/api/v1/works/<workId>/search'
  $FW exec $RUN --proof home/home-jobs -- $PW cli -s={runId}-admin-desktop run-code 'async page => { await page.getByRole("table", { name: "Jobs" }).getByRole("row", { name: /Quiet/ }).getByText("Downloading", { exact: true }).waitFor({ timeout: 90000 }); }'
  $FW fault clear $RUN swarm-stalled
  $FW exec $RUN --proof home/home-recent -- $PW cli -s={runId}-admin-desktop run-code 'async page => { await page.getByRole("list", { name: "Recent" }).getByRole("link", { name: /^The Quiet Orchard/ }).waitFor({ timeout: 90000 }); }'
  $FW exec $RUN --proof home/home-recent -- $PW cli -s={runId}-admin-desktop screenshot --filename={runDir}/proof/home/home-recent/arrived.png
  ```

- **See the tint.** On `showcase`, import Inception, then stall the seeders and search four more movies. Home takes The Dark Knight's backdrop; hovering Mad Max: Fury Road's row tints the page from it after a pause, and moving away gives the tint back.

  ```sh
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop hover 'getByRole("table", { name: "Jobs" }).getByRole("row", { name: /^Mad Max/ })'
  $FW exec $RUN --proof home/home-tint -- $PW cli -s={runId}-admin-desktop screenshot --filename={runDir}/proof/home/home-tint/hover.png
  ```

- **Fail a Grab.** On `fault-loop`, add and search The Quiet Orchard. Once its Releases fail, the sentence reads `1 failure`, Needs you holds its card `Failed at Import` with `Search again`, and Jobs is empty.
- **Read Health and Storage.** On `low-space` the sentence reads `Storage error`, Health opens with the volume's Error line over `Storage · 1 error`, and the volume's meter is red with `Error`. On `low-space-warning` the same read `Storage warning`, `Storage · 1 warning` and an amber `Warning`.
- **Time it at scale.** On `scale`, open `/` three times as in **Open Home** and report each time.

- **Read Home.** While the Grab moves it is in `jobs.rows` with its torrent; once imported, its Import leads `recent.rows` and `today.imported` is 1.

  ```sh
  $FW exec $RUN --proof home/home-summary -- curl -sS -H 'X-Api-Key: {apiKey}' "{nodes.home.url}/api/v1/home?since=$(date -u +%Y-%m-%dT00:00:00Z)"
  ```

- **Count Torrents.** The one row is seedbox's, the Node that holds Downloading, and counts the Grab's torrent as downloading, then seeding.

  ```sh
  $FW exec $RUN --proof home/home-counts -- curl -sS -H 'X-Api-Key: {apiKey}' '{nodes.home.url}/api/v1/torrents/counts'
  ```

- **Change Rows on Home.** Save 3, read Home again, then try 2 for the 422.

  ```sh
  $FW exec $RUN --proof home/home-settings -- curl -sS -X PATCH -H 'X-Api-Key: {apiKey}' -H 'Content-Type: application/json' -d '{"values":{"home.rows":3}}' '{nodes.home.url}/api/v1/home-settings'
  $FW exec $RUN --proof home/home-settings -- curl -sS -X PATCH -H 'X-Api-Key: {apiKey}' -H 'Content-Type: application/json' -d '{"values":{"home.rows":2}}' '{nodes.home.url}/api/v1/home-settings'
  ```

## Gotchas

- `since` is required and is the browser's midnight; a later instant makes `today` read 0.
- Torrent rows on Home carry `inLine` 0, because a read of a few Torrents can't place them in the queue.
- A process world's seeders send a file in under a second, so a recipe that wants a Job on screen stalls them first (`fault set … seeders stall`) and clears the rule to let it finish. Traffic samples flow only while bytes move, so a stalled download shows no rates.
- Every Scenario's Home reads `Backups warning` until the Backup passphrase is confirmed.
- With one Grab every list holds one row, so saving 3 changes nothing a read can see. The cut shows only once a list has more than 3 rows.
