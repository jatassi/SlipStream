# Search and grab

An Admin searches a Work's Indexers from the Work page, and SlipStream grabs the best release, downloads it on the Node that holds Downloading, transfers it to the Node that holds Storage and imports it into the library. An Indexer that answers 429 ends one search with Indexer unavailable, and the next search grabs.

## Sub-features

- `grab-indexer-429` a search whose Indexer answers 429 ends `Indexer unavailable · 0 releases` with no Grab.
- `grab-search` a search that reaches the fake Indexer ends `Grabbed · 2 releases` and starts a Grab on the seedbox, through the Fake world's fake Indexer, tracker and seeders (P-256).
- `grab-held` a throttled swarm holds the Grab at `Downloading` with a progress bar, and the seedbox's `started` announce reaches the tracker.
- `grab-complete-now` clearing the throttle completes the torrent at once while the seedbox's libtorrent still hashes every piece (P-257).
- `grab-transfer-import` the Grab moves through probing, transferring and importing to `Imported`, and the Admin's live stream carries `grabs.changed`.
- `grab-activity` Activity lists the Grab with its Work, release and stage.
- `grab-files` the Work page lists the imported file, and the file in the real media root matches the release byte for byte (P-258).

## How to get to it (user POV)

- Desktop shell: the sidebar's `Library`, a Work's link, then the `Search` button on the Work page; or the palette (Ctrl+K, ⌘K on macOS) with part of the Work's name.
- Phone shell: the `Library` tab, a Work's link (a pushed page with the back link `Library`), then `Search`; or the `Search` tab's palette.
- Desktop shell: the sidebar's `Activity`, which reads `Activity 1 live` while a Grab moves. Phone shell: the `Activity` tab, badged with the same count.
- API: `POST /api/v1/works/{workId}/search`, `GET /api/v1/works/{workId}/searches`, `GET /api/v1/grabs`.


## Driving it with fakeworld and playwright cli

Preconditions:

- A `home` run launched and doctor passing, with `RUN`, `FW` and `PW` set as in the skill's Launch.
- The `{runId}-admin-desktop` session open and signed in, on the Library (the skill's Drive).
- The library holds no Work: run this on a fresh `home`.

Steps:

- **Hold the swarm and arm a 429.** Throttle the seedbox's seeder fronts and make the seedbox's next Torznab request answer 429. Each command prints its `fault.set` record.

  ```sh
  $FW fault set $RUN swarm-hold seeders@seedbox throttle=16KiB/s
  $FW fault set $RUN indexer-429 fixture-site@seedbox status=429,retry-after=30s --route torznab --times 1
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop tracing-start
  ```

- **Add the Work.** Add The Quiet Orchard as in [add-a-movie.md](add-a-movie.md). The Work page shows `No searches yet.`

  ```sh
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop click 'getByRole("banner").getByRole("button", { name: "Add" })'
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop click 'getByRole("menuitem", { name: "Add a Movie" })'
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop fill 'getByRole("searchbox", { name: "Title" })' "quiet orchard"
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop click 'getByRole("main").getByRole("button", { name: "Search", exact: true })'
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop click 'getByRole("button", { name: "Add The Quiet Orchard" })'
  ```

- **Search into the 429.** Choose `Search` on the Work page. The rule fires once for the seedbox, and the Searches list reads `Indexer unavailable · 0 releases` with no Grab.

  ```sh
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop click 'getByRole("main").getByRole("button", { name: "Search", exact: true })'
  FIRED=$($FW fault wait $RUN --rule indexer-429 --timeout 30s | jq .seq)
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop run-code 'async page => { await page.getByRole("list", { name: "Searches" }).getByText("Indexer unavailable · 0 releases").waitFor({ timeout: 30000 }); }'
  $FW exec $RUN --proof search-and-grab/grab-indexer-429 -- $PW cli -s={runId}-admin-desktop snapshot --filename={runDir}/proof/search-and-grab/grab-indexer-429/work.aria.yml
  $FW exec $RUN --proof search-and-grab/grab-indexer-429 -- $PW cli -s={runId}-admin-desktop screenshot --filename={runDir}/proof/search-and-grab/grab-indexer-429/work.png
  ```

- **Search again.** Choose `Search`. The list gains `Grabbed · 2 releases`, and the seedbox announces `started` to the tracker.

  ```sh
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop click 'getByRole("main").getByRole("button", { name: "Search", exact: true })'
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop run-code 'async page => { await page.getByRole("list", { name: "Searches" }).getByText("Grabbed · 2 releases").waitFor({ timeout: 30000 }); }'
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

- **See it imported.** Activity reads `Imported`, and the API's Grab history runs fetching, downloading, probing, transferring, importing, imported.

  ```sh
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop run-code 'async page => { await page.getByRole("list", { name: "Grabs" }).getByText("Imported").waitFor({ timeout: 60000 }); }'
  $FW exec $RUN --proof search-and-grab/grab-activity -- $PW cli -s={runId}-admin-desktop screenshot --filename={runDir}/proof/search-and-grab/grab-activity/activity.png
  $FW exec $RUN --proof search-and-grab/grab-transfer-import -- curl -sS -H 'X-Api-Key: {apiKey}' '{nodes.home.url}/api/v1/grabs'
  ```

- **Prove the file.** Open the Work from Activity. Files lists `The Quiet Orchard (2021)/The Quiet Orchard (2021) - 1080p.mkv 1920×1080`, and `fs match` exits 0 on the placed file.

  ```sh
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop click 'getByRole("main").getByRole("link", { name: "The Quiet Orchard" })'
  $FW exec $RUN --proof search-and-grab/grab-files -- $PW cli -s={runId}-admin-desktop snapshot --filename={runDir}/proof/search-and-grab/grab-files/work.aria.yml
  $FW exec $RUN --proof search-and-grab/grab-files -- $PW cli -s={runId}-admin-desktop screenshot --filename={runDir}/proof/search-and-grab/grab-files/work.png
  $FW exec $RUN --proof search-and-grab/grab-files -- $FW fs match {runDir} '{mediaDir}/movies/The Quiet Orchard (2021)/The Quiet Orchard (2021) - 1080p.mkv' --release The.Quiet.Orchard.2021.1080p.WEB-DL.H264-FKW
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop tracing-stop
  ```

- **Phone.** Open `{runId}-admin-phone` as in the skill's Drive with `phone`, choose the `Activity` tab, and screenshot the same Grab into `proof/search-and-grab/grab-activity/activity-phone.png`.

## Gotchas

- `announces wait` and `sse wait` default `--after` to the previous step's start. The `started` announce arrives while the second search's click or wait step runs, so pass `--after $FIRED`, the seq of the 429's `fault.fired`.
- The Work page's `Search` shares its name with the top bar's capsule `Search Works or run a command`; scope it to `getByRole("main")` with `exact: true`.
- Search outcomes arrive after the click returns. Wait for the Searches text with `run-code`; a snapshot taken at once may read `Searching`.
- The skeleton doesn't hold off after a 429's `Retry-After`: the second Search grabs at once.
- At 16 KiB/s the 3.7 MB release takes about four minutes, so clear the throttle once the held state is captured.
- libtorrent announces again only every five minutes. Wait for `started` and `completed`, never a regular announce.
- Grab and Work ids are drawn fresh in each run. Address the Work by its link name and read ids from `GET /api/v1/grabs`.
- The Indexer lives on the seedbox, which holds Searching, so Torznab faults target `fixture-site@seedbox`. A rule on `fixture-site@home` never fires.
