# Search and grab

An Admin searches a Work's Indexers from the Work page. SlipStream asks every Indexer at once, decides once over every Release they found, grabs the best one, downloads it on the Node that holds Downloading, transfers it to the Node that holds Storage and imports it into the library. Each search lists one line per Indexer in the server's words. An Indexer that answers 429 reads `Limit reached until …` on its line, and the next search after its Retry-After grabs.

## Sub-features

- `grab-indexer-429` a search whose Indexer answers 429 ends `Found nothing acceptable`, its Fixture Site line reading `Limit reached until HH:MM`, with no Grab.
- `grab-search` a search that reaches the fake Indexer ends `Grabbed 1 release` with `2 Releases found`, the stage its Grab is at now, such as `Downloading`, `Imported` or `Failed`, and the line `Fixture Site` `Found 2 Releases`, and starts a Grab on the seedbox, through the Fake world's fake Indexer, tracker and seeders (P-256).
- `grab-held` a throttled swarm holds the Grab at `Downloading` with a progress bar, and the seedbox's `started` announce reaches the tracker.
- `grab-complete-now` clearing the throttle completes the torrent at once while the seedbox's libtorrent still hashes every piece (P-257).
- `grab-transfer-import` the finished download hands off to its Import, which reads the file on seedbox, sends it to home in a Transfer and places it. The Grab moves from `downloading` to `importing` to `Imported`, its `import` reads `placed` 1, and the Admin's live stream carries `grabs.changed` (P-074).
- `grab-activity` Activity lists the Grab with its Work, Release and stage.
- `grab-files` the Work page lists the imported file, and the file in the real media root matches the Release byte for byte (P-258).
- `grab-fault-loop` on `fault-loop`, adding The Quiet Orchard with `Search after adding` grabs one of its four Releases, each only rar parts, which fails `no_video_file`. SlipStream searches again at once with trigger `failed_grab`, skipping the Releases already tried, until the third failure in 7 days stops the cell: Needs you reads `3 Grabs failed for this file in 1 day.` with `Backlog search is off, so SlipStream searches for it only when you press Search again.` and `Search again`, which clears the count and grabs the fourth. The fourth fails too, and the search again after it finds only the four Releases already tried, so the hero reads `Found 4 releases, but none fits a Target.` (P-134).

## How to get to it (user POV)

- Desktop shell: the sidebar's `Library`, a Work's link, then `Search` in the Work page's hero; or the palette (Ctrl+K, ⌘K on macOS) with part of the Work's name. `Manual search` beside it opens the sheet in [manual-search.md](manual-search.md).
- Phone shell: the `Library` tab, a Work's link (a pushed page with the back link `Movies`), then `Search` in the hero; or the `Search` tab's palette.
- Desktop shell: the sidebar's `Activity`, which reads `Activity 1 live` while a Grab moves. Phone shell: the `Activity` tab, badged with the same count.
- API: `POST /api/v1/works/{workId}/search`, `GET /api/v1/works/{workId}/searches`, `GET /api/v1/grabs`, and `POST /api/v1/grabs/{grabId}/send-to` with `{clientId}` or `{builtInOn}` for a downloading Grab. `GET /api/v1/works/{workId}/searching` lists a stopped cell under `stops`; Settings > Automatic search's Details holds `Stop searching again after` and its window.


## Driving it with fakeworld and playwright cli

Preconditions:

- A `home` run launched and doctor passing, with `RUN`, `FW` and `PW` set as in the skill's Launch.
- The `{runId}-admin-desktop` session open and signed in, on the Library (the skill's Drive).
- The library holds no Work: run this on a fresh `home`.

Steps:

- **Hold the swarm and arm a 429.** Throttle the seedbox's seeder fronts and make the seedbox's next Torznab request answer 429. Each command prints its `fault.set` record.

  ```sh
  $FW fault set $RUN swarm-hold seeders@seedbox throttle=16KiB/s
  $FW fault set $RUN indexer-429 fixture-site@seedbox status=429,retry-after=2s --route torznab.api --times 1
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop tracing-start
  ```

- **Add the Work.** Add The Quiet Orchard as in [add-a-movie.md](add-a-movie.md). The Work page shows `No searches yet.`

  ```sh
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop click 'getByRole("banner").getByRole("button", { name: "Add" })'
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop click 'getByRole("menuitem", { name: "Add a Movie" })'
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop fill 'getByRole("searchbox", { name: "Title" })' "quiet orchard"
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop click 'getByRole("main").getByRole("button", { name: "Search", exact: true })'
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop click 'getByRole("button", { name: "Add The Quiet Orchard" })'
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop click 'getByRole("dialog", { name: "Add The Quiet Orchard" }).getByRole("button", { name: "Add Movie" })'
  ```

- **Search into the 429.** Choose `Search` on the Work page. The rule fires once for the seedbox, and the Searches list reads `Found nothing acceptable` with Fixture Site's line `Limit reached until HH:MM` and no Grab.

  ```sh
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop click 'getByRole("main").getByRole("button", { name: "Search", exact: true })'
  FIRED=$($FW fault wait $RUN --rule indexer-429 --timeout 30s | jq .seq)
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop run-code 'async page => { await page.getByRole("list", { name: "Searches" }).getByText(/^Limit reached until \d\d:\d\d$/).waitFor({ timeout: 30000 }); }'
  $FW exec $RUN --proof search-and-grab/grab-indexer-429 -- $PW cli -s={runId}-admin-desktop snapshot --filename={runDir}/proof/search-and-grab/grab-indexer-429/work.aria.yml
  $FW exec $RUN --proof search-and-grab/grab-indexer-429 -- $PW cli -s={runId}-admin-desktop screenshot --filename={runDir}/proof/search-and-grab/grab-indexer-429/work.png
  ```

- **Search again.** Wait out the 2s Retry-After, then choose `Search`. The list gains `Grabbed 1 release` with `2 Releases found` and Fixture Site's line `Found 2 Releases`, and the seedbox announces `started` to the tracker.

  ```sh
  sleep 2
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop click 'getByRole("main").getByRole("button", { name: "Search", exact: true })'
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop run-code 'async page => { await page.getByRole("list", { name: "Searches" }).getByText("Grabbed 1 release").waitFor({ timeout: 30000 }); }'
  $FW exec $RUN --proof search-and-grab/grab-search -- $PW cli -s={runId}-admin-desktop screenshot --filename={runDir}/proof/search-and-grab/grab-search/work.png
  $FW announces wait $RUN --node seedbox --where event=started --after $FIRED --timeout 60s
  ```

- **Watch the held download.** Choose `Activity` in the sidebar. The Grab reads `Downloading` with a progress bar named `Downloading The.Quiet.Orchard.2021.1080p.WEB-DL.H264-FKW`, and the API reads stage `downloading` with `progress.done` below `progress.total`.

  ```sh
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop click 'getByRole("navigation", { name: "Main" }).getByRole("link", { name: "Activity" })'
  $FW exec $RUN --proof search-and-grab/grab-held -- $PW cli -s={runId}-admin-desktop snapshot --filename={runDir}/proof/search-and-grab/grab-held/activity.aria.yml
  $FW exec $RUN --proof search-and-grab/grab-held -- $PW cli -s={runId}-admin-desktop screenshot --filename={runDir}/proof/search-and-grab/grab-held/activity.png
  $FW exec $RUN --proof search-and-grab/grab-held -- curl -sS -H 'X-Api-Key: {apiKey}' '{nodes.home.url}/api/v1/grabs'
  ```

- **Complete this torrent now.** Clear the throttle. The seedbox announces `completed`, and the Admin's stream carries `grabs.changed`.

  ```sh
  $FW fault clear $RUN swarm-hold
  $FW announces wait $RUN --node seedbox --where event=completed --timeout 60s
  $FW sse wait $RUN --as admin --event grabs.changed --timeout 60s
  ```

- **See it imported.** Activity's `Grabs` table reads `Imported`, and the API's Grab history runs fetching, downloading, importing, imported.

  ```sh
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop run-code 'async page => { await page.getByRole("table", { name: "Grabs" }).getByText("Imported").waitFor({ timeout: 60000 }); }'
  $FW exec $RUN --proof search-and-grab/grab-activity -- $PW cli -s={runId}-admin-desktop screenshot --filename={runDir}/proof/search-and-grab/grab-activity/activity.png
  $FW exec $RUN --proof search-and-grab/grab-transfer-import -- curl -sS -H 'X-Api-Key: {apiKey}' '{nodes.home.url}/api/v1/grabs'
  ```

- **Prove the file.** Open the Work from Activity. Files lists `The Quiet Orchard (2021) - 1080p.mkv` in `The Quiet Orchard (2021)` at `1920×1080`, and `fs match` exits 0 on the placed file.

  ```sh
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop click 'getByRole("main").getByRole("link", { name: "The Quiet Orchard" })'
  $FW exec $RUN --proof search-and-grab/grab-files -- $PW cli -s={runId}-admin-desktop snapshot --filename={runDir}/proof/search-and-grab/grab-files/work.aria.yml
  $FW exec $RUN --proof search-and-grab/grab-files -- $PW cli -s={runId}-admin-desktop screenshot --filename={runDir}/proof/search-and-grab/grab-files/work.png
  $FW exec $RUN --proof search-and-grab/grab-files -- $FW fs match {runDir} '{mediaDir}/movies/The Quiet Orchard (2021)/The Quiet Orchard (2021) - 1080p.mkv' --release The.Quiet.Orchard.2021.1080p.WEB-DL.H264-FKW
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop tracing-stop
  ```

- **Phone.** Open `{runId}-admin-phone` as in the skill's Drive with `phone`, choose the `Activity` tab, and screenshot the same Grab into `proof/search-and-grab/grab-activity/activity-phone.png`.

- **Search again after bad Releases.** On a fresh `fault-loop` run, add The Quiet Orchard with its defaults. Each Grab fails `no_video_file` within seconds; wait for the third, then read the Work.

  ```sh
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop goto '{nodes.home.browserUrl}/add?contentType=movie&query=quiet+orchard'
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop click 'getByRole("button", { name: "Add The Quiet Orchard" })'
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop click 'getByRole("dialog", { name: "Add The Quiet Orchard" }).getByRole("button", { name: "Add Movie" })'
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop run-code 'async page => { await page.getByText(/^3 Grabs failed for this file/).first().waitFor({ timeout: 120000 }); }'
  $FW exec $RUN --proof search-and-grab/grab-fault-loop -- $PW cli -s={runId}-admin-desktop snapshot --filename={runDir}/proof/search-and-grab/grab-fault-loop/work.aria.yml
  $FW exec $RUN --proof search-and-grab/grab-fault-loop -- $PW cli -s={runId}-admin-desktop screenshot --filename={runDir}/proof/search-and-grab/grab-fault-loop/work.png
  $FW exec $RUN --proof search-and-grab/grab-fault-loop -- curl -sS -H 'X-Api-Key: {apiKey}' '{nodes.home.url}/api/v1/grabs'
  ```

  The answer lists three failed Grabs, the latest with `foundBy` `failed_grab` and a `searchId`. The older two read `searchId` null, because their runs were pruned, and keep what found them: `foundBy` `search` for the add's Grab and `failed_grab` for the second. Read the Work's id from its `workId`, and `GET /api/v1/works/<id>/searches` and `/searching` with `--proof search-and-grab/grab-fault-loop`. Searches lists one run, the group's latest, with trigger `failed_grab` and one Grab, because each finished run replaces the earlier runs of its group. Searching lists the cell under `stops` with its three failures.

## Gotchas

- `announces wait` and `sse wait` default `--after` to the previous step's start. The `started` announce arrives while the second search's click or wait step runs, so pass `--after $FIRED`, the seq of the 429's `fault.fired`.
- The Work page's `Search` sits in the hero inside `getByRole("main")` on both shells. Use `exact: true`, which keeps it apart from `Manual search` beside it and from the top bar's capsule `Search Works or run a command`. The Add page's catalog `Search` is in `getByRole("main")` too.
- Search outcomes arrive after the click returns. Wait for the Searches text with `run-code`; a snapshot taken at once may read `Searching`.
- A 429 limits the Indexer until its `Retry-After` passes, and a search before then skips it with `Limit reached until …` without asking the site. Keep the fault's `retry-after` short and wait it out before searching again.
- A Grab whose Indexer stops, is deleted or whose home Node lapses before the .torrent is fetched fails with `indexer_stopped`, `indexer_removed` or `home_offline`, and the search's decision reruns over the Releases it kept: the run's `grabIds` gains the rerun's Grab. The rerun's Grab holds the dropped Grab's cells, so its failure reads `current: false` and leaves Needs you on Activity and the Work page alike. Read it from `GET /api/v1/grabs`.
- At 16 KiB/s the 3.7 MB Release takes about four minutes, so clear the throttle once the held state is captured.
- libtorrent announces again only every five minutes. Wait for `started` and `completed`, never a regular announce.
- Grab and Work ids are drawn fresh in each run. Address the Work by its link name and read ids from `GET /api/v1/grabs`.
- The Indexer lives on the seedbox, which holds Searching, so Torznab faults target `fixture-site@seedbox`. A rule on `fixture-site@home` never fires.
- `fault-loop`'s Releases download whole and fail at importing, so each failure lands within seconds of its Grab. Its Backlog search and RSS sync are off, so every run after the first is a search again.
