# Series Work page

A series' Work page shows the series' facts in the hero, then one collapsible section per season in order with Specials last. Each season's header has its name, a state bar, a tally and a Monitor switch, and a season with anything wanted and not met opens. Each episode is a row with its code, title, air date, its cells its own Monitor switch and a Search, and each season's header has a Search too. The hero's Search searches every wanted episode, a season's its wanted episodes and an episode's that episode, one Torznab `tvsearch` by TVDB id, season and episode each. Unmonitored episodes read `Not monitored`, a run of announced episodes folds into one line, and an episode TVDB drops leaves its season. On a phone each row opens a sheet with the episode's cells and actions. When TVDB's order stops listing an episode that holds a file along with its season, the episode trails the list in one section named `No longer listed by TVDB`, whose header counts the episodes it holds.

## Sub-features

- `series-hero` The Lantern Keepers added with First season reads `2020 to present · Series · Lantern One · Fantasy, Drama · Continuing` and `Monitoring 3 of 11 episodes` (P-049, P-287).
- `series-seasons` `Seasons` holds `Season 1`, `Season 2`, `Season 3` and `Specials` in that order. Season 1 is open with `0 of 3 met`, and the unmonitored seasons are closed and read `Not monitored` (P-050).
- `series-rows` `S01E01` reads `First Flame`, `January 10, 2020` and its cell `S01E01 1080p, Missing`; opening Season 2 shows `S02E01` `The Northern Lamp` with `Not monitored` (P-050).
- `series-popover` hovering `S01E01 1080p, Missing` opens the detail `S01E01 1080p` reading `Missing`, clicking pins it while the pointer leaves, and `Escape` closes it.
- `series-fold` after fake-tvdb announces a fourth season and a refresh, Season 4 reads one line `Episodes 1 to 3 air <day> to <day>` in place of three rows, and dropping `Fog Signal` leaves Season 2 with `S02E01` and `S02E03` and no row for it (P-050).
- `series-work-search` with First season, the hero's `Search` toasts `Searching 3 episodes`, `Searches` reads `Grabbed a Release` twice, for S01E01 and S01E03, and `searches` records three `tvsearch` searches with `tvdbid` 1001, `season` 1 and `ep` 1, 2 and 3 (P-050).
- `series-level-search` with only S02E01 of Season 2 monitored, `Search Season 2` toasts `Searching 1 episode` and searches `season` 2 `ep` 1 alone (P-050).
- `series-episode-search` with S00E02 monitored, `Search S00E02` toasts `Searching 1 episode`, sends `season` 0 and `ep` 2, and A Winter Light imports (P-050).
- `series-withdrawn-file` after the order changes to DVD, which doesn't list A Winter Light, the last heading is `No longer listed by TVDB` with `1 episode` and a `1 Met` bar, after one `Specials`, and holds S00E02 with its file (P-050).
- `series-phone` on the phone shell a row is one button, and `S01E01` opens the sheet `S01E01 First Flame` with its `Cells`, a `Monitor` switch, `Search` and `Details`.

## How to get to it (user POV)

- Desktop shell: the TV library's poster or table row, or the palette's Works group, opens `/works/<id>`.
- Phone shell: the same, pushed with a back link labelled `TV`; seasons are cards, and a row opens the episode's sheet.
- API: `GET /api/v1/works/{id}` (the tree with levels and Items), `GET /api/v1/works/{id}/cells`, `POST /api/v1/works/refresh {workIds}`, `POST /api/v1/works/{id}/search` with an optional `{levelIds, itemIds}` body, `PUT /api/v1/works/{id}/options`.

## Driving it with fakeworld, curl and playwright cli

Preconditions:

- A `series` run, doctor passing, `{runId}-admin-desktop` signed in, and The Lantern Keepers added by API with `preset` `first`.

Steps:

- **Open the page.** Read the hero and the seasons, then open Season 2.

  ```sh
  WORK=$($FW exec $RUN -- curl -sS -X POST -H 'X-Api-Key: {apiKey}' -H 'Content-Type: application/json' -d '{"contentType":"series","key":"1001","preset":"first"}' '{nodes.home.url}/api/v1/works' | jq -r .work.id)
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop goto "{nodes.home.browserUrl}/works/$WORK"
  $FW exec $RUN --proof series-work-page/series-seasons -- $PW cli -s={runId}-admin-desktop snapshot --filename={runDir}/proof/series-work-page/series-seasons/page.aria.yml
  $FW exec $RUN --proof series-work-page/series-seasons -- $PW cli -s={runId}-admin-desktop screenshot --filename={runDir}/proof/series-work-page/series-seasons/page.png
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop click 'getByRole("region", { name: "Seasons" }).getByRole("button", { name: "Season 2", exact: true })'
  $FW exec $RUN --proof series-work-page/series-rows -- $PW cli -s={runId}-admin-desktop snapshot --filename={runDir}/proof/series-work-page/series-rows/season-2.aria.yml
  ```

- **A cell's detail.** Hover the cell, then click it.

  ```sh
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop hover 'getByRole("button", { name: "S01E01 1080p, Missing" })'
  $FW exec $RUN --proof series-work-page/series-popover -- $PW cli -s={runId}-admin-desktop snapshot --filename={runDir}/proof/series-work-page/series-popover/hover.aria.yml
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop click 'getByRole("button", { name: "S01E01 1080p, Missing" })'
  $FW exec $RUN --proof series-work-page/series-popover -- $PW cli -s={runId}-admin-desktop screenshot --filename={runDir}/proof/series-work-page/series-popover/pinned.png
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop press Escape
  ```

- **A fold and a dropped episode.** Announce a season, drop Fog Signal, refresh, then open Season 4.

  ```sh
  $FW cue $RUN tvdb-add-season '{"id":1001}'
  $FW cue $RUN tvdb-remove-episode '{"id":100122}'
  $FW exec $RUN --proof series-work-page/series-fold -- curl -sS -X POST -H 'X-Api-Key: {apiKey}' -H 'Content-Type: application/json' -d '{"workIds":["'$WORK'"]}' '{nodes.home.url}/api/v1/works/refresh'
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop goto "{nodes.home.browserUrl}/works/$WORK"
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop click 'getByRole("region", { name: "Seasons" }).getByRole("button", { name: "Season 4", exact: true })'
  $FW exec $RUN --proof series-work-page/series-fold -- $PW cli -s={runId}-admin-desktop snapshot --filename={runDir}/proof/series-work-page/series-fold/season-4.aria.yml
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop run-code 'async page => { const season = page.getByRole("region", { name: "Seasons" }).getByRole("button", { name: "Season 2", exact: true }); if (await season.getAttribute("aria-expanded") !== "true") await season.click(); }'
  $FW exec $RUN --proof series-work-page/series-fold -- $PW cli -s={runId}-admin-desktop snapshot --filename={runDir}/proof/series-work-page/series-fold/season-2.aria.yml
  ```

- **Search.** On the `preset` `first` Work, search the series, wait for the two grabs, read the searches the site answered, then monitor S02E01 alone and search Season 2.

  ```sh
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop goto "{nodes.home.browserUrl}/works/$WORK"
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop click 'getByRole("main").getByRole("button", { name: "Search", exact: true })'
  $FW exec $RUN --proof series-work-page/series-work-search -- $PW cli -s={runId}-admin-desktop screenshot --filename={runDir}/proof/series-work-page/series-work-search/toast.png
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop run-code 'async page => { await page.getByRole("list", { name: "Searches" }).getByText("Grabbed a Release").nth(1).waitFor(); }'
  $FW exec $RUN --proof series-work-page/series-work-search -- $PW cli -s={runId}-admin-desktop snapshot --filename={runDir}/proof/series-work-page/series-work-search/searches.aria.yml
  $FW searches list $RUN --where t=tvsearch
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop click 'getByRole("region", { name: "Seasons" }).getByRole("button", { name: "Season 2", exact: true })'
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop click 'getByRole("region", { name: "Seasons" }).getByRole("switch", { name: "Monitor S02E01" })'
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop click 'getByRole("region", { name: "Seasons" }).getByRole("button", { name: "Search Season 2" })'
  $FW exec $RUN --proof series-work-page/series-level-search -- $PW cli -s={runId}-admin-desktop snapshot --filename={runDir}/proof/series-work-page/series-level-search/toast.aria.yml
  ```

- **An episode's Search and a withdrawn file.** On the same Work, open Specials, monitor and search S00E02, wait for its file, then change the order to DVD.

  ```sh
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop goto "{nodes.home.browserUrl}/works/$WORK"
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop run-code 'async page => { const b = page.getByRole("region", { name: "Seasons" }).getByRole("button", { name: "Specials", exact: true }); for (let i = 0; i < 30 && await b.count() === 0; i++) await page.mouse.wheel(0, 400); await b.scrollIntoViewIfNeeded(); if (await b.getAttribute("aria-expanded") !== "true") await b.click(); }'
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop run-code 'async page => { const s = page.getByRole("region", { name: "Seasons" }).getByRole("switch", { name: "Monitor S00E02" }); await s.scrollIntoViewIfNeeded(); await s.click(); }'
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop click 'getByRole("region", { name: "Seasons" }).getByRole("button", { name: "Search S00E02" })'
  $FW exec $RUN --proof series-work-page/series-episode-search -- $PW cli -s={runId}-admin-desktop screenshot --filename={runDir}/proof/series-work-page/series-episode-search/toast.png
  $FW searches list $RUN --where t=tvsearch --where season=0
  $FW exec $RUN --proof series-work-page/series-episode-search -- curl -sS -H 'X-Api-Key: {apiKey}' "{nodes.home.url}/api/v1/works/$WORK/cells"
  $FW exec $RUN --proof series-work-page/series-withdrawn-file -- curl -sS -X PUT -H 'X-Api-Key: {apiKey}' -H 'Content-Type: application/json' -d '{"type":{"order":"dvd"},"preview":false}' "{nodes.home.url}/api/v1/works/$WORK/options"
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop goto "{nodes.home.browserUrl}/works/$WORK"
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop run-code 'async page => { const b = page.getByRole("region", { name: "Seasons" }).getByRole("button", { name: "No longer listed by TVDB", exact: true }); for (let i = 0; i < 30 && await b.count() === 0; i++) await page.mouse.wheel(0, 400); await b.scrollIntoViewIfNeeded(); if (await b.getAttribute("aria-expanded") !== "true") await b.click(); }'
  $FW exec $RUN --proof series-work-page/series-withdrawn-file -- $PW cli -s={runId}-admin-desktop snapshot --filename={runDir}/proof/series-work-page/series-withdrawn-file/page.aria.yml
  $FW exec $RUN --proof series-work-page/series-withdrawn-file -- $PW cli -s={runId}-admin-desktop screenshot --filename={runDir}/proof/series-work-page/series-withdrawn-file/page.png
  ```

- **Phone.** Open the phone session on the page and tap `S01E01`.

  ```sh
  $FW exec $RUN -- $PW cli -s={runId}-admin-phone goto "{nodes.home.browserUrl}/works/$WORK"
  $FW exec $RUN -- $PW cli -s={runId}-admin-phone click 'getByRole("region", { name: "Seasons" }).getByRole("button", { name: /^S01E01 / })'
  $FW exec $RUN --proof series-work-page/series-phone -- $PW cli -s={runId}-admin-phone snapshot --filename={runDir}/proof/series-work-page/series-phone/sheet.aria.yml
  $FW exec $RUN --proof series-work-page/series-phone -- $PW cli -s={runId}-admin-phone screenshot --filename={runDir}/proof/series-work-page/series-phone/sheet.png
  ```

## Gotchas

- The seasons are one window virtualizer: a header or row below the window is not in the DOM until the page scrolls to it, so scroll before you look for it.
- Fold dates follow the browser's locale: `en-US` reads `November 8 to November 22`, `en-GB` `8 November to 22 November`.
- An episode TVDB drops shows only while it holds a file; with no file it leaves the page and the counts.
- `tvdb-add-season` dates the new episodes from tomorrow on the fake's clock, so a run near midnight UTC can read a day off the browser's.
- A Search counts the wanted episodes no unfinished Grab holds. The catalog's 1080p WEB-DL files read `Upgrade available`, so an episode is wanted again once its Grab imports, and a count taken after a Search depends on how far its Grabs got.
- The Prowlarr host's caps turn `tv-search` off, so an Indexer there searches free text such as `The Lantern Keepers S01E03`, and The Lantern Keepers' S01E03 release listed under Harbor Nights comes back refused as `wrong_work`.
- The world's Indexer probe sends one `t=search` with an empty `q`. Skip it when you count searches.
