# RSS sync

RSS sync reads each Indexer's new uploads every 15 minutes by default, one unit per Indexer with RSS on, and judges each Release as a search would, so a wanted Work grabs from the feed. A busy feed is paged back to the position the last fetch stored, and a feed that can't list older uploads leaves a gap that Health reports with `Open Wanted`. A gone file's cell waits for the Admin's own Search, so RSS sync never grabs for it.

## Sub-features

- `rss-first-fetch` on `busy-feed`, `Run now: RSS sync` runs one unit per Indexer, each `succeeded` with `Read no new releases.`, which stores each feed's position (P-139).
- `rss-gap` after the `fixture-site-feed` cue posts 250 uploads on each site, the next run reads Fixture Site `Read 250 new releases. None matched a Work.` over three pages, and Heron, which lists its newest page at every offset, `Read 100 new releases. None matched a Work.` Heron's `rssFetch.gapKind` reads `no_paging`, and Health's Indexers category warns `RSS sync may have missed releases on Heron. More releases were added to Heron between two RSS syncs than one fetch returns, and Heron can't list older ones. Search from Wanted to find what was missed, or run RSS sync more often.` with an `Open Wanted` link to Wanted filtered to Missing and Failed (P-141).
- `rss-gone-waits` on `gone-with-rss`, with The Quiet Orchard's and The Salt Road's files marked gone and both Releases listed by `fixture-site-list`, the run reads `Read 2 new releases. 2 matched a Work, and 1 was grabbed.`: it grabs Long Division, `source` `rss`, and leaves The Quiet Orchard's gone cell, which Wanted lists under `File gone` (P-140).
- `rss-gone-admin-search` the hero's `Search` on The Quiet Orchard then grabs its Bluray at once, `source` `search` (P-132).

## How to get to it (user POV)

- Desktop shell: the sidebar's `Tasks`, then `RSS sync`, with `Run now: RSS sync`, its interval and its units per Indexer; Settings > Automatic search's `RSS sync` switch and `RSS sync runs every`; each Indexer's `RSS` switch in Settings > Indexers; Health's Indexers category.
- Phone shell: `More`, then `Tasks`; the `Health` page.
- API: `GET /api/v1/tasks/rss-sync`, `POST /api/v1/tasks/rss-sync/runs`, `GET /api/v1/indexers` (`rssFetch {lastFetchAt, gapKind, gapSince}`), `GET /api/v1/system/health`, `GET /api/v1/grabs` (`source` `rss`).

## Driving it with fakeworld and playwright cli

Preconditions:

- A `busy-feed` run for the first two sub-features and a `gone-with-rss` run for the last two, each launched and doctor passing, with `RUN`, `FW` and `PW` set as in the skill's Launch, and `{runId}-admin-desktop` signed in.

Steps:

- **Store each feed's position.** On `busy-feed`, open the task and run it.

  ```sh
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop goto '{nodes.home.browserUrl}/tasks/rss-sync'
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop click 'getByRole("button", { name: "Run now: RSS sync" })'
  $FW exec $RUN --proof rss-sync/rss-first-fetch -- curl -sS -H 'X-Api-Key: {apiKey}' '{nodes.home.url}/api/v1/tasks/rss-sync'
  ```

- **Post more than a page and run again.** Cue 250 uploads on each site, Heron's without paging, then run the task again. Health warns about Heron.

  ```sh
  $FW cue $RUN fixture-site-feed '{"site":"torznab","uploads":250}'
  $FW cue $RUN fixture-site-feed '{"site":"heron","uploads":250,"noPaging":true}'
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop reload
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop click 'getByRole("button", { name: "Run now: RSS sync" })'
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop run-code 'async page => { await page.getByText(/^Read 250 new releases/).first().waitFor({ timeout: 60000 }); }'
  $FW exec $RUN --proof rss-sync/rss-gap -- $PW cli -s={runId}-admin-desktop screenshot --filename={runDir}/proof/rss-sync/rss-gap/ran.png
  $FW exec $RUN --proof rss-sync/rss-gap -- curl -sS -H 'X-Api-Key: {apiKey}' '{nodes.home.url}/api/v1/tasks/rss-sync'
  $FW exec $RUN --proof rss-sync/rss-gap -- curl -sS -H 'X-Api-Key: {apiKey}' '{nodes.home.url}/api/v1/indexers'
  $FW exec $RUN --proof rss-sync/rss-gap -- curl -sS -H 'X-Api-Key: {apiKey}' '{nodes.home.url}/api/v1/system/health'
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop goto '{nodes.home.browserUrl}/health'
  $FW exec $RUN --proof rss-sync/rss-gap -- $PW cli -s={runId}-admin-desktop screenshot --filename={runDir}/proof/rss-sync/rss-gap/health.png
  ```

- **Mark two films gone.** On `gone-with-rss`, run RSS sync once to store the position, remove two of the four films and settle the stopped scan with `Mark them gone`, as in [library-roots.md](library-roots.md).

  ```sh
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop goto '{nodes.home.browserUrl}/tasks/rss-sync'
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop click 'getByRole("button", { name: "Run now: RSS sync" })'
  $FW fs rm $RUN 'movies/The Quiet Orchard (2021) {tmdb-9000001}/The Quiet Orchard (2021).mkv'
  $FW fs rm $RUN 'movies/The Salt Road (2022) {tmdb-9000005}/The Salt Road (2022).mkv'
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop goto '{nodes.home.browserUrl}/settings/library-roots'
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop click 'getByRole("button", { name: "Actions for {mediaDir}/movies" })'
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop click 'getByRole("menuitem", { name: "Scan now" })'
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop run-code 'async page => { await page.getByText(/^Library scan stopped/).waitFor({ timeout: 30000 }); }'
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop click 'getByRole("button", { name: "Actions for {mediaDir}/movies" })'
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop click 'getByRole("menuitem", { name: "Mark them gone" })'
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop click 'getByRole("dialog", { name: "Mark 2 files gone?" }).getByRole("button", { name: "Mark them gone" })'
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop run-code 'async page => { await page.getByText("It found 2 gone files.").waitFor({ timeout: 30000 }); }'
  ```

- **Add Long Division, list both Releases and run RSS sync.** Add it with `Search after adding` off, list both held-back Releases and run the task. Only Long Division is grabbed.

  ```sh
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop goto '{nodes.home.browserUrl}/add?contentType=movie&query=long+division'
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop click 'getByRole("button", { name: "Add Long Division" })'
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop click 'getByRole("dialog", { name: "Add Long Division" }).getByRole("switch", { name: "Search after adding" })'
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop click 'getByRole("dialog", { name: "Add Long Division" }).getByRole("button", { name: "Add Movie" })'
  $FW cue $RUN fixture-site-list '{"release":"The.Quiet.Orchard.2021.1080p.BluRay.x264-FKW"}'
  $FW cue $RUN fixture-site-list '{"release":"Long.Division.2019.1080p.BluRay.x264-FKW"}'
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop goto '{nodes.home.browserUrl}/tasks/rss-sync'
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop click 'getByRole("button", { name: "Run now: RSS sync" })'
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop run-code 'async page => { await page.getByText(/and 1 was grabbed/).first().waitFor({ timeout: 60000 }); }'
  $FW exec $RUN --proof rss-sync/rss-gone-waits -- curl -sS -H 'X-Api-Key: {apiKey}' '{nodes.home.url}/api/v1/tasks/rss-sync'
  $FW exec $RUN --proof rss-sync/rss-gone-waits -- curl -sS -H 'X-Api-Key: {apiKey}' '{nodes.home.url}/api/v1/grabs'
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop goto '{nodes.home.browserUrl}/library/wanted'
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop click 'getByRole("button", { name: "File gone" })'
  $FW exec $RUN --proof rss-sync/rss-gone-waits -- $PW cli -s={runId}-admin-desktop screenshot --filename={runDir}/proof/rss-sync/rss-gone-waits/wanted-gone.png
  ```

- **Search the gone film.** Open The Quiet Orchard from Wanted and choose `Search`. The hero reads `Grabbed 1 Release`.

  ```sh
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop click 'getByRole("main").getByRole("link", { name: "The Quiet Orchard" })'
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop click 'getByRole("main").getByRole("button", { name: "Search", exact: true }).first()'
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop run-code 'async page => { await page.getByRole("status", { name: "Search" }).getByText(/^Grabbed/).waitFor({ timeout: 60000 }); }'
  $FW exec $RUN --proof rss-sync/rss-gone-admin-search -- $PW cli -s={runId}-admin-desktop screenshot --filename={runDir}/proof/rss-sync/rss-gone-admin-search/work.png
  $FW exec $RUN --proof rss-sync/rss-gone-admin-search -- curl -sS -H 'X-Api-Key: {apiKey}' '{nodes.home.url}/api/v1/grabs'
  ```

- **Phone.** Open `{runId}-admin-phone` and screenshot Health on `busy-feed` into `proof/rss-sync/rss-gap/health-phone.png`, and Wanted on `gone-with-rss` into `proof/rss-sync/rss-gone-waits/wanted-phone.png`.

## Gotchas

- RSS sync's first scheduled run comes about 15 minutes after the Controller starts. Use `Run now`, and run it once before any cue, since a first fetch only stores the position and reports no gap.
- A Feed cue without `site` plays on every Torznab site and clears the Scenario's own listings, `no_paging` included. Name each site to keep the other's listing.
- The run's unit note doesn't name a gap. Read it from the Indexer's `rssFetch` and Health.
- A position covers every upload published no later than its newest one, so a Release a cue lists must be published after the feed's newest, as `gone-with-rss` dates its two.
- A run started from the Tasks page right after a reload may find the button gone while the previous run ends; wait for its note first.
