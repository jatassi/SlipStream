# Automatic search

SlipStream searches a Work by itself when it is added with `Search after adding`, and an Admin's `Search` goes ahead of everything automatic. A season searches as one query per Indexer and fills its empty cells with the fewest Releases, a pack winning over better singles, while upgrades go episode by episode. A second press on a group that is queued or searching merges into it and says so, and a search decides at its deadline when an Indexer is late, with that Indexer's line reading `Didn't finish in time`.

## Sub-features

- `auto-search-on-add-pack` on `pack-beats-singles`, adding The Copper Coast with `Search after adding` at `Everything monitored` toasts `Added The Copper Coast. Searching now.`, and the hero reads `Grabbed 2 Releases`: the pack `The.Copper.Coast.S01E01-E09.1080p.WEB-DL.H264-NTB`, whose `chosenBecause` is `fewer_releases` filling 9, and the Bluray single of S01E10, both through to `imported` (P-138, P-132, P-210).
- `auto-upgrade-singles` the hero's `Search` on the same Work, every cell filled by the pack's WEB-DL under the Bluray cutoff, reads `Grabbed 9 Releases`: the Bluray singles of S01E01 to S01E09, one Grab each (P-132).
- `auto-press-merges` on `five-indexers-one-late`, a second `Search` while the first waits for Tamarack toasts `Already searching The Quiet Orchard`, and the hero reads `Searching` until the run decides (P-046, P-133).
- `auto-late-indexer` with `Wait for each Indexer up to` set to 15 s under Settings > Automatic search's Details, the search decides 15 seconds after it started: the Searches list reads `Grabbed 1 Release`, four Indexers `Found 1 Release` and Tamarack `Didn't finish in time`, and the API's line for Tamarack reads `late` (P-132).

## How to get to it (user POV)

- Desktop shell: the add sheet's `Search after adding` (a switch for a movie, a choice of `Everything monitored`, `First season`, `Latest season`, `First episode` or `Don't search` for a series); `Search` in a Work's hero, `Search Season N` on a season header and `Search S01E01` on an episode row; Settings > Automatic search with `Details` for the search wait.
- Phone shell: the same add sheet and hero; the season header's search button; an episode's sheet.
- API: `POST /api/v1/works` with `search`, `POST /api/v1/works/{workId}/search`, `GET /api/v1/works/{workId}/searching` (each group's `state`, `lane`, `trigger` and `behind`), `GET /api/v1/works/{workId}/searches`, `GET /api/v1/grabs` (`source`, `chosenBecause`), `GET` and `PATCH /api/v1/search-settings`.

## Driving it with fakeworld and playwright cli

Preconditions:

- A `pack-beats-singles` run for the first two sub-features and a `five-indexers-one-late` run for the last two, each launched and doctor passing, with `RUN`, `FW` and `PW` set as in the skill's Launch, and `{runId}-admin-desktop` signed in.

Steps:

- **Add a series that searches on add.** On `pack-beats-singles`, open the add sheet and add with its defaults. The hero reads `Grabbed 2 Releases`, and the Grabs read the pack with `fewer_releases` and the single of S01E10.

  ```sh
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop goto '{nodes.home.browserUrl}/add?contentType=series&query=copper'
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop click 'getByRole("button", { name: "Add The Copper Coast" })'
  $FW exec $RUN --proof automatic-search/auto-search-on-add-pack -- $PW cli -s={runId}-admin-desktop screenshot --filename={runDir}/proof/automatic-search/auto-search-on-add-pack/sheet.png
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop click 'getByRole("dialog", { name: "Add The Copper Coast" }).getByRole("button", { name: "Add Series" })'
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop run-code 'async page => { await page.getByRole("status", { name: "Search" }).getByText(/^Grabbed/).waitFor({ timeout: 60000 }); }'
  $FW exec $RUN --proof automatic-search/auto-search-on-add-pack -- $PW cli -s={runId}-admin-desktop screenshot --filename={runDir}/proof/automatic-search/auto-search-on-add-pack/work.png
  $FW exec $RUN --proof automatic-search/auto-search-on-add-pack -- curl -sS -H 'X-Api-Key: {apiKey}' '{nodes.home.url}/api/v1/grabs'
  ```

- **Upgrade episode by episode.** Choose the hero's `Search` once the pack is imported. The hero reads `Grabbed 9 Releases`.

  ```sh
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop click 'getByRole("main").getByRole("button", { name: "Search", exact: true }).first()'
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop run-code 'async page => { await page.getByRole("status", { name: "Search" }).getByText("Grabbed 9 Releases").waitFor({ timeout: 30000 }); }'
  $FW exec $RUN --proof automatic-search/auto-upgrade-singles -- $PW cli -s={runId}-admin-desktop screenshot --filename={runDir}/proof/automatic-search/auto-upgrade-singles/work.png
  $FW exec $RUN --proof automatic-search/auto-upgrade-singles -- curl -sS -H 'X-Api-Key: {apiKey}' '{nodes.home.url}/api/v1/grabs'
  ```

- **Shorten the search wait.** On `five-indexers-one-late`, open Settings > Automatic search, open `Details`, set `Wait for each Indexer up to` to 15 s and `Save`. The API reads `indexers.searchWait` 15.

  ```sh
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop goto '{nodes.home.browserUrl}/settings/automatic-search'
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop click 'getByRole("main").getByRole("button", { name: "Details" })'
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop click 'getByRole("radiogroup", { name: "Wait for each Indexer up to unit" }).getByRole("radio", { name: "s" })'
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop fill 'getByRole("textbox", { name: "Wait for each Indexer up to" })' 15
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop click 'getByRole("main").getByRole("button", { name: "Save" })'
  $FW exec $RUN --proof automatic-search/auto-late-indexer -- curl -sS -H 'X-Api-Key: {apiKey}' '{nodes.home.url}/api/v1/search-settings'
  ```

- **Press twice while Tamarack is late.** Add The Quiet Orchard with `Search after adding` off, then choose `Search` twice. The second toasts `Already searching The Quiet Orchard`, and after 15 seconds the Searches list reads Tamarack `Didn't finish in time`.

  ```sh
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop goto '{nodes.home.browserUrl}/add?contentType=movie&query=quiet+orchard'
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop click 'getByRole("button", { name: "Add The Quiet Orchard" })'
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop click 'getByRole("dialog", { name: "Add The Quiet Orchard" }).getByRole("switch", { name: "Search after adding" })'
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop click 'getByRole("dialog", { name: "Add The Quiet Orchard" }).getByRole("button", { name: "Add Movie" })'
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop click 'getByRole("main").getByRole("button", { name: "Search", exact: true }).first()'
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop click 'getByRole("main").getByRole("button", { name: "Search", exact: true }).first()'
  $FW exec $RUN --proof automatic-search/auto-press-merges -- $PW cli -s={runId}-admin-desktop screenshot --filename={runDir}/proof/automatic-search/auto-press-merges/merged.png
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop run-code 'async page => { await page.getByRole("status", { name: "Search" }).getByText(/^Grabbed/).waitFor({ timeout: 60000 }); }'
  $FW exec $RUN --proof automatic-search/auto-late-indexer -- $PW cli -s={runId}-admin-desktop screenshot --filename={runDir}/proof/automatic-search/auto-late-indexer/work.png
  $FW exec $RUN --proof automatic-search/auto-late-indexer -- curl -sS -H 'X-Api-Key: {apiKey}' '{nodes.home.url}/api/v1/grabs'
  ```

  Read the Work's id from that answer's `workId`, then `GET /api/v1/works/<id>/searches`: `finishedAt` is 15 seconds after `startedAt`, and Tamarack's line reads `late`.

- **Phone.** Open `{runId}-admin-phone`, go to the Work's page and screenshot it into `proof/automatic-search/auto-late-indexer/work-phone.png`; on `pack-beats-singles`, the series into `proof/automatic-search/auto-search-on-add-pack/work-phone.png`.

## Gotchas

- A movie's Work page has a second `Search` in main once Needs you lists `Upgrades available`; take `.first()`, the hero's.
- On `pack-beats-singles` the search ends within a second, so two presses don't overlap. Prove the merge on `five-indexers-one-late`, whose search waits for Tamarack.
- A press that merges into a queued or bulk group answers `Already searching …` too, since the press promotes the group and it starts at once.
- The Tamarack fault arms at `run.ready` and delays every Tamarack request by 25 s, Test and caps included. Keep the search wait below 25 s, and never above 30 s, the site's HTTP timeout.
- Backlog search and RSS sync are on in `pack-beats-singles`. RSS sync's first run comes about 15 minutes in and can grab, so finish the recipe before then or read Grabs by `source`.
