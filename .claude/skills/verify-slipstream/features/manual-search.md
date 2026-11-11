# Manual search

An Admin opens Manual search from any Search button to see the Releases for part of a Work and choose what to grab. The sheet asks every on Indexer, Automatic search off included, and shows Releases as each Indexer responds. It leads with a plan in plain words: the fewest Grabs that fill the missing, failed and upgradable cells in scope, chosen as automatic search would choose them. Each row can be swapped through Other choices, grabbed past a soft refusal, grabbed as another Edition, sent to a chosen client, or taken off the Blocklist and grabbed. A Grab records who grabbed it and what they overrode, and Grab Details shows the record. Cancel stops a Grab, and with `Also blocklist this Release` adds a Blocklist entry. A release fault and Mark failed add entries too, and Activity's Blocklist tab lists and removes them.

## Sub-features

- `manual-open` Manual search opens from the hero's `Manual search`, a cell's `Manual search for <Target>, <Edition>`, a season header's and an episode row's `More actions for <name>` menu, a Needs you failure card, and on phone from the hero's action sheet, a cell's `Actions for <Target>, <Edition>` menu and the episode sheet. The sheet is a dialog titled with the scope in words, such as `The Quiet Orchard` or `E10 of Season 2 of The Lantern Keepers`, with the Work's title alone when the scope covers every Item of the Work, as Season 2 does here, and a full-height sheet on phone (P-123).
- `manual-reopen` closing the sheet and opening it again on the same scope shows the same Manual search, with `Started <age> by Ada Okafor` and `Search again`, and starts no new one (P-123).
- `manual-streaming` while Heron answers 20 seconds late, the Searching line reads `Searching 4 Indexers. 3 have responded.` and the plan already lists Fixture Site's Releases. Once Heron responds it reads `SlipStream searched 4 Indexers.`, and its Details lists each Indexer's outcome with `Open Indexer` and `Retry` (P-126).
- `manual-plan` the Quiet Orchard's plan reads `2 Grabs get you everything.`: Tamarack's Bluray for `1080p (standard)` (Tamarack's automatic search is off, so an automatic Search would grab Fixture Site's WEB-DL instead), and FKW's 1080p WEB-DL for `HD (standard)` as an upgrade. `SlipStream wouldn't grab N more Releases.` folds the rest into sentences, such as `1 names another Work.` and `1 fails a filter of every Target here.` (P-127).
- `manual-all-releases` `All Releases (N)` lists every Release with `Sort by` Rank, Score, Quality, Indexer, Size, Age or Seeders, a `Freeleech` chip on CMRG's WEB-DL, and each Release's Details with its raw title, a verdict per Target, Indexers, Size, Published, Seeders and `Open in tester` (P-125).
- `manual-query` a scope of one Item has a `Query` field. A typed query starts a new Manual search whose `query` echoes it, and fixture-site's `searches` capture records it as `q`. A season scope has no `Query` field (P-124).
- `manual-swap-override` `Other choices` on the HD row lists LQ's WEB-DL with the `Override` chip `Score -5 is below the minimum of 0 at HD.`. `Use this` pins it, the plan is worked out again around it, and the pin survives later refreshes (P-127).
- `manual-client` a row's Details `Send to` takes `Built-in client on home`, the row then reads `SlipStream sends it to the built-in client on home.`, and `Send all to` sets every row without its own (P-127).
- `manual-grab` `Grab 2` grabs both ticked rows in one press, each row reads `Grabbed.`, and each Grab's Details on Activity reads `Manual search by Ada Okafor, HH:MM.` with its overrides as sentences, such as `Ada Okafor grabbed it although score -5 is below the minimum of 0 at HD.` and `Ada Okafor sent it to the built-in client on home.` (P-127).
- `manual-cancel` Cancel Grab on a downloading Grab asks `Cancel this Grab?` with the Torrent's preview and `Also blocklist this Release`. Confirming fails the Grab `canceled`, its cell reads missing with no Needs you card, and the Blocklist gains `Ada Okafor blocklisted it when canceling its Grab.` (P-356).
- `manual-mark-failed` Mark failed on the Torrent a downloading Grab holds adds `Ada Okafor marked its download failed.` to the Blocklist (P-356).
- `manual-release-fault` grabbing S02E10's Release, which the tracker reports unregistered, fails its Grab `release_fault` and adds `Fixture Site reported it unregistered.` to the Blocklist (P-356).
- `manual-unblock-edition` searching again after those entries, the Bluray reads `It's been on the Blocklist since <day> because Ada Okafor marked its download failed.` with `Remove from Blocklist and grab`, and the Extended WEB-DL is grabbed as the Standard Edition. The Grabs read `Ada Okafor removed it from the Blocklist to grab it.` and `Ada Okafor grabbed it as the Standard Edition.`, the Bluray's entry leaves the Blocklist, and both import (P-127, P-356).
- `manual-blocklist-tab` Activity's `Blocklist` tab lists each entry with its Release in words and raw title, its cause in a sentence, its Indexer, its Work and when. `Search the Blocklist` filters by title, and picking entries and `Remove` empties it (P-356).
- `manual-series-plan` Season 2's plan reads `5 Grabs get you 5 of the 6 files you're missing. SlipStream hasn't found a Release for 1080p yet.`, with one row per episode from E05 to E10 but E09, `Missing: 1080p for E09.` with `No Indexer listed a 1080p Release.`, and folded lines for the season pack, `SlipStream can't import packs yet.`, and for the Release named S02E07 listed under E05, `1 names another episode.` (P-123).

## How to get to it (user POV)

- Desktop shell: a Work page's hero has `Manual search` beside `Search`. Each Edition by Target cell has `Manual search for <Target>, <Edition>` beside its `Search for <Target>, <Edition>`, and a filled cell's details popover has `Manual search`. On a series, each season header's and each episode row's `More actions for <name>` menu has `Manual search`, and each Needs you failure card has `Manual search` beside `Details`.
- Phone shell: the hero's `More actions for <title>` action sheet lists `Manual search` first. A cell's action opens the menu `Actions for <Target>, <Edition>` with `Search` and `Manual search`. The episode sheet lists `Manual search`. Other choices and Details open in place inside the full-height sheet.
- Activity: the `Grabs` table's `<Work> actions` menu has `Details` and `Cancel Grab`. The `Blocklist` tab sits after `Jobs` and `Torrents` at `/activity/blocklist`. Torrents' `Mark failed` blocklists.
- API: `POST /api/v1/works/{workId}/manual-search` with `{levelIds, itemIds, cell, query, again}`, `GET /api/v1/works/{workId}/manual-search?pins=`, `GET /api/v1/works/{workId}/manual-search/releases`, `POST /api/v1/works/{workId}/manual-search/grabs` with `{manualSearchId, picks}`, `POST /api/v1/grabs/{grabId}/cancel` with `{blocklist}`, `GET /api/v1/blocklist?offset=`, `POST /api/v1/blocklist/remove` with `{infoHashes}`, and `GET /api/v1/grabs/{grabId}` for `source`, `grabbedBy`, `manualSearchId` and `overrides`.

## Driving it with fakeworld, playwright cli and curl

Preconditions:

- A fresh `manual-search` run with doctor passing, `{runId}-admin-desktop` signed in and, for the phone step, `{runId}-admin-phone`. The one Node, home, holds every Role and has a Challenge solver for Cinder.
- The first library scans add The Quiet Orchard, with a 720p WEB-DL that leaves `1080p` missing and `HD` upgradable, and The Lantern Keepers, with S02E01 to S02E04, so E05 to E10 are missing.
- The Scenario's fault `heron-answers-late` holds every Heron search 20 seconds. FKW's 1080p REPACK stays unlisted until `fixture-site-list`, so this walk never sees it.

Steps:

- **Find the Works.** Wait for both scans, then name the sessions and the sheet.

  ```sh
  D=-s={runId}-admin-desktop
  P=-s={runId}-admin-phone
  KEY='X-Api-Key: {apiKey}'
  $FW exec $RUN -- sh -c "until curl -sS -H '$KEY' '{nodes.home.url}/api/v1/library' | jq -e '.rows | length == 2' >/dev/null; do sleep 1; done"
  ORCHARD=$($FW exec $RUN -- curl -sS -H "$KEY" '{nodes.home.url}/api/v1/library' | jq -r '.rows[] | select(.label == "The Quiet Orchard") | .id')
  KEEPERS=$($FW exec $RUN -- curl -sS -H "$KEY" '{nodes.home.url}/api/v1/library' | jq -r '.rows[] | select(.label == "The Lantern Keepers") | .id')
  SHEET='getByRole("dialog", { name: "The Quiet Orchard" })'
  $FW exec $RUN -- $PW cli $D tracing-start
  ```

- **Open it from the hero and watch it stream.** Choose `Manual search` in the hero. Within 20 seconds the Searching line reads `Searching 4 Indexers. 3 have responded.` and the `Grabs` list already has rows. Once Heron responds it reads `SlipStream searched 4 Indexers.` Open the line's `Details`: `Indexers` lists Fixture Site, Heron, Tamarack and Cinder, each with its outcome, `Open Indexer` and `Retry`.

  ```sh
  $FW exec $RUN -- $PW cli $D goto "{nodes.home.browserUrl}/works/$ORCHARD"
  $FW exec $RUN -- $PW cli $D click 'getByRole("main").getByRole("button", { name: "Manual search", exact: true }).first()'
  $FW exec $RUN -- $PW cli $D run-code 'async page => { await page.getByRole("dialog", { name: "The Quiet Orchard" }).getByText("Searching 4 Indexers. 3 have responded.").waitFor({ timeout: 15000 }); }'
  $FW exec $RUN --proof manual-search/manual-streaming -- $PW cli $D snapshot "$SHEET" --filename={runDir}/proof/manual-search/manual-streaming/searching.aria.yml
  $FW exec $RUN --proof manual-search/manual-streaming -- $PW cli $D screenshot --filename={runDir}/proof/manual-search/manual-streaming/searching.png
  $FW exec $RUN -- $PW cli $D run-code 'async page => { await page.getByRole("dialog", { name: "The Quiet Orchard" }).getByText("SlipStream searched 4 Indexers.").waitFor({ timeout: 60000 }); }'
  $FW exec $RUN -- $PW cli $D click "$SHEET"'.getByRole("button", { name: "Details" }).first()'
  $FW exec $RUN --proof manual-search/manual-streaming -- $PW cli $D snapshot "$SHEET"'.getByRole("list", { name: "Indexers" })' --filename={runDir}/proof/manual-search/manual-streaming/indexers.aria.yml
  $FW exec $RUN --proof manual-search/manual-open -- curl -sS -H "$KEY" "{nodes.home.url}/api/v1/works/$ORCHARD/manual-search"
  $FW fault clear $RUN heron-answers-late
  ```

- **Read the plan.** The headline reads `2 Grabs get you everything.` The first row is `Bluray-1080p · FKW · 9.8 GB` with `It gets you 1080p (standard) for The Quiet Orchard.`, the second FKW's 1080p WEB-DL with `It gets you HD (standard) for The Quiet Orchard as an upgrade.` Open the folded line's `Details` for its groups.

  ```sh
  $FW exec $RUN -- $PW cli $D click "$SHEET"'.getByText(/^SlipStream wouldn.t grab/).locator("..").getByRole("button", { name: "Details" })'
  $FW exec $RUN --proof manual-search/manual-plan -- $PW cli $D snapshot "$SHEET" --filename={runDir}/proof/manual-search/manual-plan/plan.aria.yml
  $FW exec $RUN --proof manual-search/manual-plan -- $PW cli $D screenshot --filename={runDir}/proof/manual-search/manual-plan/plan.png
  ```

- **Browse All Releases.** Open `All Releases (N)`, sort by `Seeders`, and open CMRG's Details. It shows the `Freeleech` chip, its raw title, a verdict per Target and its facts. `Open in tester` opens the `Tester` dialog with its title.

  ```sh
  ALL="$SHEET"'.getByRole("list", { name: "All Releases" })'
  $FW exec $RUN -- $PW cli $D click "$SHEET"'.getByRole("button", { name: /^All Releases \(\d+\)$/ })'
  $FW exec $RUN -- $PW cli $D click "$SHEET"'.getByRole("combobox", { name: "Sort by" })'
  $FW exec $RUN -- $PW cli $D click 'getByRole("option", { name: "Seeders" })'
  $FW exec $RUN -- $PW cli $D click "$ALL"'.getByRole("listitem").filter({ hasText: "Freeleech" }).getByRole("button", { name: "Details" })'
  $FW exec $RUN --proof manual-search/manual-all-releases -- $PW cli $D snapshot "$ALL" --filename={runDir}/proof/manual-search/manual-all-releases/releases.aria.yml
  $FW exec $RUN --proof manual-search/manual-all-releases -- $PW cli $D screenshot --filename={runDir}/proof/manual-search/manual-all-releases/releases.png
  $FW exec $RUN --proof manual-search/manual-all-releases -- curl -sS -H "$KEY" "{nodes.home.url}/api/v1/works/$ORCHARD/manual-search/releases"
  ```

- **Reopen it.** Press Escape and choose `Manual search` again. The sheet reads `Started <age> by Ada Okafor`, and the API's `id` is the one read before.

  ```sh
  $FW exec $RUN -- $PW cli $D press Escape
  $FW exec $RUN -- $PW cli $D click 'getByRole("main").getByRole("button", { name: "Manual search", exact: true }).first()'
  $FW exec $RUN --proof manual-search/manual-reopen -- $PW cli $D snapshot "$SHEET" --filename={runDir}/proof/manual-search/manual-reopen/sheet.aria.yml
  $FW exec $RUN --proof manual-search/manual-reopen -- curl -sS -H "$KEY" "{nodes.home.url}/api/v1/works/$ORCHARD/manual-search"
  ```

- **Swap, choose a client and grab.** Hold the swarm so both Grabs stay at downloading. On the HD row open `Other choices` and `Use this` on LQ's WEB-DL: the row now shows the `Override` chip `Score -5 is below the minimum of 0 at HD.` On the Bluray row open `Details` and set `Send to` to `Built-in client on home`. Choose `Grab 2`. Each row reads `Grabbed.`

  ```sh
  $FW fault set $RUN swarm-hold seeders@home throttle=16KiB/s
  ROW1="$SHEET"'.getByRole("list", { name: "Grabs" }).getByRole("listitem").filter({ hasText: "It gets you 1080p" })'
  ROW2="$SHEET"'.getByRole("list", { name: "Grabs" }).getByRole("listitem").filter({ hasText: "It gets you HD" })'
  $FW exec $RUN -- $PW cli $D click "$ROW2"'.getByRole("button", { name: "Other choices" })'
  $FW exec $RUN -- $PW cli $D click "$ROW2"'.getByRole("list", { name: "Other choices" }).getByRole("listitem").filter({ hasText: "· LQ ·" }).getByRole("button", { name: "Use this" })'
  $FW exec $RUN -- $PW cli $D run-code 'async page => { await page.getByRole("dialog", { name: "The Quiet Orchard" }).getByText("Score -5 is below the minimum of 0 at HD.").first().waitFor({ timeout: 15000 }); }'
  $FW exec $RUN --proof manual-search/manual-swap-override -- $PW cli $D snapshot "$ROW2" --filename={runDir}/proof/manual-search/manual-swap-override/row.aria.yml
  $FW exec $RUN -- $PW cli $D click "$ROW1"'.getByRole("button", { name: "Details", exact: true })'
  $FW exec $RUN -- $PW cli $D click "$ROW1"'.getByRole("combobox", { name: "Send to" })'
  $FW exec $RUN -- $PW cli $D click 'getByRole("option", { name: "Built-in client on home", exact: true })'
  $FW exec $RUN --proof manual-search/manual-client -- $PW cli $D snapshot "$ROW1" --filename={runDir}/proof/manual-search/manual-client/row.aria.yml
  $FW exec $RUN -- $PW cli $D click "$SHEET"'.getByRole("button", { name: "Grab 2", exact: true })'
  $FW exec $RUN -- $PW cli $D run-code 'async page => { await page.getByRole("dialog", { name: "The Quiet Orchard" }).getByText("Grabbed.").nth(1).waitFor({ timeout: 15000 }); }'
  $FW exec $RUN --proof manual-search/manual-grab -- $PW cli $D screenshot --filename={runDir}/proof/manual-search/manual-grab/grabbed.png
  $FW exec $RUN --proof manual-search/manual-grab -- curl -sS -H "$KEY" '{nodes.home.url}/api/v1/grabs'
  ```

- **Read Grab Details.** On Activity, open the Bluray Grab's `The Quiet Orchard actions` menu and `Details`. The dialog `Grab of The Quiet Orchard` reads `Manual search by Ada Okafor, HH:MM.`, `Ada Okafor sent it to the built-in client on home.`, the stages with their times and the facts. The LQ Grab's reads `Ada Okafor grabbed it although score -5 is below the minimum of 0 at HD.`

  ```sh
  GRABS='getByRole("table", { name: "Grabs" }).getByRole("row")'
  $FW exec $RUN -- $PW cli $D press Escape
  $FW exec $RUN -- $PW cli $D goto '{nodes.home.browserUrl}/activity'
  $FW exec $RUN -- $PW cli $D click "$GRABS"'.filter({ hasText: "BluRay" }).getByRole("button", { name: "The Quiet Orchard actions" })'
  $FW exec $RUN -- $PW cli $D click 'getByRole("menuitem", { name: "Details" })'
  $FW exec $RUN --proof manual-search/manual-grab -- $PW cli $D snapshot 'getByRole("dialog", { name: "Grab of The Quiet Orchard" })' --filename={runDir}/proof/manual-search/manual-grab/details-bluray.aria.yml
  $FW exec $RUN -- $PW cli $D press Escape
  $FW exec $RUN -- $PW cli $D click "$GRABS"'.filter({ hasText: "-LQ" }).getByRole("button", { name: "The Quiet Orchard actions" })'
  $FW exec $RUN -- $PW cli $D click 'getByRole("menuitem", { name: "Details" })'
  $FW exec $RUN --proof manual-search/manual-grab -- $PW cli $D snapshot 'getByRole("dialog", { name: "Grab of The Quiet Orchard" })' --filename={runDir}/proof/manual-search/manual-grab/details-lq.aria.yml
  $FW exec $RUN -- $PW cli $D press Escape
  ```

- **Cancel with Also blocklist.** From the LQ row's menu choose `Cancel Grab`. The dialog `Cancel this Grab?` previews the Torrent's removal. Tick `Also blocklist this Release` and choose `Cancel Grab`. The Grab fails `canceled`, the Work's HD cell reads upgradable again with no Needs you card for it, and the Blocklist holds one `by_hand` entry.

  ```sh
  $FW exec $RUN -- $PW cli $D click "$GRABS"'.filter({ hasText: "-LQ" }).getByRole("button", { name: "The Quiet Orchard actions" })'
  $FW exec $RUN -- $PW cli $D click 'getByRole("menuitem", { name: "Cancel Grab" })'
  $FW exec $RUN -- $PW cli $D click 'getByRole("dialog", { name: "Cancel this Grab?" }).getByRole("checkbox", { name: "Also blocklist this Release" })'
  $FW exec $RUN --proof manual-search/manual-cancel -- $PW cli $D snapshot 'getByRole("dialog", { name: "Cancel this Grab?" })' --filename={runDir}/proof/manual-search/manual-cancel/dialog.aria.yml
  $FW exec $RUN -- $PW cli $D click 'getByRole("dialog", { name: "Cancel this Grab?" }).getByRole("button", { name: "Cancel Grab" })'
  $FW exec $RUN --proof manual-search/manual-cancel -- curl -sS -H "$KEY" '{nodes.home.url}/api/v1/grabs'
  $FW exec $RUN --proof manual-search/manual-cancel -- curl -sS -H "$KEY" '{nodes.home.url}/api/v1/blocklist'
  ```

- **Mark the Bluray failed.** On Torrents open the row's `The Quiet Orchard actions` menu, choose `Mark failed` and confirm `Mark The Quiet Orchard failed?`. The Blocklist gains `Ada Okafor marked its download failed.`, and the Work page's Needs you shows the failed 1080p cell.

  ```sh
  $FW exec $RUN -- $PW cli $D goto '{nodes.home.browserUrl}/activity/torrents?tab=all'
  $FW exec $RUN -- $PW cli $D click 'getByRole("table", { name: "Torrents" }).getByRole("button", { name: "The Quiet Orchard actions" })'
  $FW exec $RUN -- $PW cli $D click 'getByRole("menuitem", { name: "Mark failed" })'
  $FW exec $RUN -- $PW cli $D click 'getByRole("dialog", { name: "Mark The Quiet Orchard failed?" }).getByRole("button", { name: "Mark failed" })'
  $FW exec $RUN --proof manual-search/manual-mark-failed -- curl -sS -H "$KEY" '{nodes.home.url}/api/v1/blocklist'
  ```

- **Plan a season and fault a Release.** Shorten `Unregistered twice, this far apart` to 1 min on Settings > Download Clients, or the default 30 minutes passes before the release fault. Open The Lantern Keepers, open `More actions for Season 2` and `Manual search`, and capture the season's plan. Season 2 is every Item of the Work, so the sheet is titled `The Lantern Keepers`. Then open `More actions for S02E10` and `Manual search`, and choose `Grab 1`. Clear the hold. The tracker answers the Grab's announce `Unregistered torrent`, the Grab fails `release_fault`, and the Blocklist gains `Fixture Site reported it unregistered.`

  ```sh
  $FW exec $RUN -- $PW cli $D goto '{nodes.home.browserUrl}/settings/download-clients'
  $FW exec $RUN -- $PW cli $D fill 'getByRole("textbox", { name: "Unregistered twice, this far apart" })' 1
  $FW exec $RUN -- $PW cli $D click 'getByRole("main").getByRole("button", { name: "Save" })'
  $FW exec $RUN -- $PW cli $D goto "{nodes.home.browserUrl}/works/$KEEPERS"
  $FW exec $RUN -- $PW cli $D click 'getByRole("button", { name: "More actions for Season 2" })'
  $FW exec $RUN -- $PW cli $D click 'getByRole("menuitem", { name: "Manual search" })'
  $FW exec $RUN -- $PW cli $D run-code 'async page => { await page.getByRole("dialog", { name: "The Lantern Keepers" }).getByText(/^SlipStream searched/).waitFor({ timeout: 60000 }); }'
  $FW exec $RUN --proof manual-search/manual-series-plan -- $PW cli $D snapshot 'getByRole("dialog", { name: "The Lantern Keepers" })' --filename={runDir}/proof/manual-search/manual-series-plan/plan.aria.yml
  $FW exec $RUN --proof manual-search/manual-series-plan -- $PW cli $D screenshot --filename={runDir}/proof/manual-search/manual-series-plan/plan.png
  $FW exec $RUN -- $PW cli $D press Escape
  $FW exec $RUN -- $PW cli $D click 'getByRole("button", { name: "More actions for S02E10" })'
  $FW exec $RUN -- $PW cli $D click 'getByRole("menuitem", { name: "Manual search" })'
  $FW exec $RUN -- $PW cli $D run-code 'async page => { await page.getByRole("dialog").getByRole("button", { name: "Grab 1", exact: true }).waitFor({ timeout: 60000 }); }'
  $FW exec $RUN --proof manual-search/manual-open -- $PW cli $D snapshot 'getByRole("dialog")' --filename={runDir}/proof/manual-search/manual-open/episode.aria.yml
  $FW exec $RUN -- $PW cli $D click 'getByRole("dialog").getByRole("button", { name: "Grab 1", exact: true })'
  $FW fault clear $RUN swarm-hold
  $FW exec $RUN -- sh -c "until curl -sS -H '$KEY' '{nodes.home.url}/api/v1/blocklist' | jq -e '.entries[] | select(.cause == \"unregistered\")' >/dev/null; do sleep 2; done"
  $FW exec $RUN --proof manual-search/manual-release-fault -- curl -sS -H "$KEY" '{nodes.home.url}/api/v1/grabs'
  $FW exec $RUN --proof manual-search/manual-release-fault -- curl -sS -H "$KEY" '{nodes.home.url}/api/v1/blocklist'
  ```

- **Remove from Blocklist and grab, as another Edition.** On The Quiet Orchard, choose the failure card's `Manual search` under Needs you. The new search's 1080p row has `Other choices` with the Bluray, which reads `It's been on the Blocklist since <day> because Ada Okafor marked its download failed.` Choose its `Remove from Blocklist and grab`: the row now reads `... Grabbing it removes it from the Blocklist.` On the HD row, `Use this` on the Extended WEB-DL, then choose `Grab 2`. Both Grabs import, the Bluray's entry leaves the Blocklist, and their Details read `Ada Okafor removed it from the Blocklist to grab it.` and `Ada Okafor grabbed it as the Standard Edition.`

  ```sh
  $FW exec $RUN -- $PW cli $D goto "{nodes.home.browserUrl}/works/$ORCHARD"
  $FW exec $RUN -- $PW cli $D click 'getByRole("region", { name: /^Needs you/ }).getByRole("button", { name: "Manual search" })'
  $FW exec $RUN -- $PW cli $D run-code 'async page => { await page.getByRole("dialog", { name: "The Quiet Orchard" }).getByText(/^SlipStream searched/).waitFor({ timeout: 60000 }); }'
  $FW exec $RUN -- $PW cli $D click "$ROW1"'.getByRole("button", { name: "Other choices" })'
  $FW exec $RUN --proof manual-search/manual-unblock-edition -- $PW cli $D snapshot "$ROW1" --filename={runDir}/proof/manual-search/manual-unblock-edition/blocked.aria.yml
  $FW exec $RUN -- $PW cli $D click "$ROW1"'.getByRole("list", { name: "Other choices" }).getByRole("listitem").filter({ hasText: "Bluray-1080p" }).getByRole("button", { name: "Remove from Blocklist and grab" })'
  $FW exec $RUN -- $PW cli $D run-code 'async page => { await page.getByRole("dialog", { name: "The Quiet Orchard" }).getByText(/Grabbing it removes it from the Blocklist\.$/).waitFor({ timeout: 15000 }); }'
  $FW exec $RUN -- $PW cli $D click "$ROW2"'.getByRole("button", { name: "Other choices" })'
  $FW exec $RUN -- $PW cli $D click "$ROW2"'.getByRole("list", { name: "Other choices" }).getByRole("listitem").filter({ hasText: "Extended" }).getByRole("button", { name: "Use this" })'
  $FW exec $RUN --proof manual-search/manual-unblock-edition -- $PW cli $D snapshot "$SHEET" --filename={runDir}/proof/manual-search/manual-unblock-edition/plan.aria.yml
  $FW exec $RUN -- $PW cli $D click "$SHEET"'.getByRole("button", { name: "Grab 2", exact: true })'
  $FW exec $RUN -- sh -c "until curl -sS -H '$KEY' '{nodes.home.url}/api/v1/grabs' | jq -e '[.grabs[] | select(.stage == \"imported\")] | length == 2' >/dev/null; do sleep 2; done"
  $FW exec $RUN --proof manual-search/manual-unblock-edition -- curl -sS -H "$KEY" '{nodes.home.url}/api/v1/grabs'
  $FW exec $RUN --proof manual-search/manual-unblock-edition -- curl -sS -H "$KEY" '{nodes.home.url}/api/v1/blocklist'
  ```

- **Review and clear the Blocklist.** Open Activity's `Blocklist` tab. Its table lists LQ's entry and S02E10's. Search `LQ` to leave one, clear the search, pick both and choose `Remove`. The toast reads `Removed 2 Releases from the Blocklist.`, and the tab is empty.

  ```sh
  $FW exec $RUN -- $PW cli $D goto '{nodes.home.browserUrl}/activity/blocklist'
  $FW exec $RUN --proof manual-search/manual-blocklist-tab -- $PW cli $D snapshot 'getByRole("table", { name: "Blocklist" })' --filename={runDir}/proof/manual-search/manual-blocklist-tab/table.aria.yml
  $FW exec $RUN --proof manual-search/manual-blocklist-tab -- $PW cli $D screenshot --filename={runDir}/proof/manual-search/manual-blocklist-tab/table.png
  $FW exec $RUN -- $PW cli $D fill 'getByRole("searchbox", { name: "Search the Blocklist" })' LQ
  $FW exec $RUN --proof manual-search/manual-blocklist-tab -- $PW cli $D snapshot 'getByRole("table", { name: "Blocklist" })' --filename={runDir}/proof/manual-search/manual-blocklist-tab/search.aria.yml
  $FW exec $RUN -- $PW cli $D fill 'getByRole("searchbox", { name: "Search the Blocklist" })' ''
  $FW exec $RUN -- $PW cli $D click 'getByRole("checkbox", { name: "Pick all" })'
  $FW exec $RUN -- $PW cli $D click 'getByRole("toolbar", { name: "Picked entries" }).getByRole("button", { name: "Remove" })'
  $FW exec $RUN --proof manual-search/manual-blocklist-tab -- curl -sS -H "$KEY" '{nodes.home.url}/api/v1/blocklist'
  ```

- **Type a query.** Open the 1080p cell's `Manual search for 1080p, Standard`. Its scope is one Item, so the sheet has `Query`. Type `quiet orchard` and choose the field's `Search`. The new Manual search's `query` reads `quiet orchard`, and fixture-site records the search's `q`. The Season 2 sheet captured earlier has no `Query`.

  ```sh
  $FW exec $RUN -- $PW cli $D goto "{nodes.home.browserUrl}/works/$ORCHARD"
  $FW exec $RUN -- $PW cli $D click 'getByRole("button", { name: "Manual search for 1080p, Standard" })'
  $FW exec $RUN -- $PW cli $D fill 'getByRole("dialog").getByRole("searchbox", { name: "Query" })' 'quiet orchard'
  $FW exec $RUN -- $PW cli $D click 'getByRole("dialog").getByRole("button", { name: "Search", exact: true })'
  $FW exec $RUN --proof manual-search/manual-query -- curl -sS -H "$KEY" "{nodes.home.url}/api/v1/works/$ORCHARD/manual-search"
  $FW exec $RUN --proof manual-search/manual-query -- $FW searches list {runDir} --where 'q=quiet orchard'
  $FW exec $RUN -- $PW cli $D tracing-stop
  ```

- **Phone.** Open `{runId}-admin-phone` as in the skill's Drive with `phone`. On The Quiet Orchard open `More actions for The Quiet Orchard` and `Manual search`: the sheet fills the screen. Both cells are filled by now, so the plan has no rows. Open `All Releases` and CMRG's `Details`, which opens in place. Close it, then open the cell's `Actions for 1080p, Standard` menu, which lists `Manual search` (and `Search` too while the cell is missing). On The Lantern Keepers, tap `S02E05` for its episode sheet and choose its `Manual search`, then open its row's `Details` in place.

  ```sh
  $FW exec $RUN -- $PW cli $P goto "{nodes.home.browserUrl}/works/$ORCHARD"
  $FW exec $RUN -- $PW cli $P click 'getByRole("button", { name: "More actions for The Quiet Orchard" })'
  $FW exec $RUN -- $PW cli $P click 'getByRole("dialog", { name: "Actions for The Quiet Orchard" }).getByRole("button", { name: "Manual search" })'
  $FW exec $RUN -- $PW cli $P run-code 'async page => { await page.getByRole("dialog", { name: "The Quiet Orchard" }).getByText(/^SlipStream searched/).waitFor({ timeout: 60000 }); }'
  $FW exec $RUN -- $PW cli $P click "$SHEET"'.getByRole("button", { name: /^All Releases \(\d+\)$/ })'
  $FW exec $RUN -- $PW cli $P click "$SHEET"'.getByRole("list", { name: "All Releases" }).getByRole("listitem").filter({ hasText: "Freeleech" }).getByRole("button", { name: "Details" })'
  $FW exec $RUN --proof manual-search/manual-open -- $PW cli $P screenshot --filename={runDir}/proof/manual-search/manual-open/sheet-phone.png
  $FW exec $RUN -- $PW cli $P press Escape
  $FW exec $RUN -- $PW cli $P click 'getByRole("button", { name: "Search for 1080p, Standard" })'
  $FW exec $RUN --proof manual-search/manual-open -- $PW cli $P snapshot 'getByRole("dialog", { name: "Actions for 1080p, Standard" })' --filename={runDir}/proof/manual-search/manual-open/cell-phone.aria.yml
  $FW exec $RUN -- $PW cli $P press Escape
  $FW exec $RUN -- $PW cli $P goto "{nodes.home.browserUrl}/works/$KEEPERS"
  $FW exec $RUN -- $PW cli $P click 'getByRole("region", { name: "Seasons" }).getByRole("button", { name: /^S02E05 / })'
  $FW exec $RUN -- $PW cli $P click 'getByRole("dialog", { name: /^S02E05 / }).getByRole("button", { name: "Manual search" })'
  $FW exec $RUN -- $PW cli $P run-code 'async page => { await page.getByRole("dialog").getByRole("button", { name: "Grab 1", exact: true }).waitFor({ timeout: 60000 }); }'
  $FW exec $RUN -- $PW cli $P click 'getByRole("dialog").getByRole("list", { name: "Grabs" }).getByRole("button", { name: "Details" })'
  $FW exec $RUN --proof manual-search/manual-open -- $PW cli $P screenshot --filename={runDir}/proof/manual-search/manual-open/episode-phone.png
  ```

## Gotchas

- Each Work keeps one Manual search, shared by every Admin. Opening another scope of the same Work, a cell or the Needs you card's Item, starts a new one and replaces the old. A Grab from a sheet still showing the old one answers `manual_search_replaced`, and the sheet offers `Open Manual search again`.
- The plan's first row is Tamarack's Bluray only because Manual search asks Indexers whose Automatic search is off. An automatic `Search` on the same Work grabs Fixture Site's WEB-DL. Don't read that difference as a bug.
- Heron's delay is a Scenario fault that holds every Heron search 20 seconds. Clear `heron-answers-late` once the streaming proof is captured, or every later search waits for it.
- The sheet's list of Grabs and Activity's table are both named `Grabs`. Scope the sheet's to its dialog and Activity's to `getByRole("table")`.
- `Grab 2` and `Grab 1` need `exact: true`, since each Release under All Releases has its own `Grab`.
- The hero's `Manual search` shares its name with the Needs you card's, so take the hero's with `.first()`. The folded line's and each row's disclosure are all named `Details`, so reach the folded line's through its sentence.
- After the Needs you search, LQ's WEB-DL is on the Blocklist too, so the 1080p row's Other choices has two `Remove from Blocklist and grab` buttons. Filter the list item by `Bluray-1080p`.
- `Remove from Blocklist and grab` under a row's Other choices pins the Release. The `Grab N` button grabs it. The same button under All Releases grabs at once.
- Torrents names each row by its Work, so two held torrents of The Quiet Orchard share `Pick The Quiet Orchard`. Cancel the LQ Grab before you mark the Bluray failed.
- `manual-search` has one Node, so no Grab transfers. Cancel during a Transfer needs `seedbox-nas` ([transfers.md](transfers.md)). Cancel at probing or importing answers `grab_busy`.
- Grabbing doesn't monitor an Item, and Manual search ignores monitoring and release dates, so an unmonitored Item still plans.
- A Release on the Blocklist stays out of every plan and every automatic search until removed, whatever Work lists it.
- The code says `canceled`, with one l, in the failure code, the toast and the Blocklist sentence.
