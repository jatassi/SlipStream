# Imports

A Grab stops at its download and hands off to an Import, which takes every video of the download, reads which Items each file holds, decides once which cells each file beats, and lands each file in its own Write. On `packs` a season pack downloads on the Worker seedbox and its episodes land one by one in the Controller home's Series root through a Transfer, while Activity reads `Importing 4 of 10 episodes`. A file the library refuses waits in Pending imports with its sentence and a recommended action instead of failing the Grab, and holds its cells, whose facts read `Waiting for you`. On `imports` an Admin forces a file that isn't an upgrade into its Target from a confirm dialog that names the file it deletes, and imports a folder of the Node's download folder by hand with Import files.

## Sub-features

- `import-pack-two-nodes` on `packs`, `Search Season 1` on The Copper Coast grabs `The.Copper.Coast.S01.720p.WEB-DL.H264-FKW` on seedbox. The Grab runs `fetching`, `downloading`, `importing`, `imported` as its client reports the download complete, and its Import sends the 10 episodes to home in one Transfer (`purpose` `import`, `job` the Import's id) and lands them as `The Copper Coast (2022)/Season 01/The Copper Coast - S01E01 - Low Tide 1080p.mkv` and so on, `webdl-720p`. The pack's 2-second sample and its nfo land nowhere. Each placed file's size and BLAKE3 equal its download's on seedbox, with one link, since a Transfer copies (P-074, P-075, P-169).
- `import-pack-progress` the pack's files land as they arrive: the Grab's `import` reads `Importing 0 of 10 episodes` with `sending` 10, then `Importing 1 of 10 episodes` once the first file is verified, and ends `Imported 10 of 10 episodes`. Activity's Grabs row reads `Importing` in `Stage`, its track lighting Transfer, and the Import's words in `Progress`, on desktop and on the phone's cards (P-079).
- `import-held-step` a held cell's facts name the step its Grab or Import is at. While the pack sends, each Season 1 episode's cell popover, and on the phone its episode sheet, reads the chip `Downloading` over `Importing 0 of 10 episodes` and `Sending from seedbox`. On `imports`, The Quiet Orchard's 1080p cell reads `Downloading`, `Importing 0 of 1 movie` with its `Waiting for you` link, the step `Waiting for you`, and `Replaces WEBDL-1080p` (P-081).
- `import-movie-pack` on `packs`, The Glass Meridian's release of a 1080p and a 2160p HDR10 file places the 1080p file in the 1080p cell and leaves the 2160p one out with `meets_nothing`. Long Division's `CD1` and `CD2` wait together with `This movie is split into 2 files. SlipStream imports a movie as one file.`, recommending `Don't import` and offering `Not this Release`. A two-episode file covers both its Items, covered by `TestAFileOfTwoItemsCountsOnceTowardItsTargetsEstimate` in `internal/files` and `TestPackLandsInAnyOrder` in `fakes/world/sim` (P-080).
- `import-no-number` on `packs`, `Search Season 2` grabs `The.Copper.Coast.S02.720p.WEB-DL.H264-QOS`, whose files are named `01.mkv` to `03.mkv`. Each waits `no_number`, the Grab stays at `importing` reading `Importing 0 of 3 files` with `Waiting for you`, and nothing is blocklisted. The Pending import reads `The file name has no episode number.` and, in this build, offers no action (P-089).
- `import-not-an-upgrade` on `imports`, Search on The Quiet Orchard grabs `The.Quiet.Orchard.2021.1080p.BluRay.x264-FKW`, whose video is 720p. Its file waits `not_an_upgrade`, the Library reads `1 pending import needs you`, and Pending imports' `From downloads` lists the Release with `The Quiet Orchard · Downloaded on home` and the file's `This isn't better than the file you have for 1080p.`, `Don't import` and `Other actions for <file>` holding `Force into 1080p` (P-083).
- `import-force` `Force into 1080p` opens `Force into 1080p?` with `This deletes The.Quiet.Orchard.2021.1080p.WEB-DL.H264-FKW.mkv, because it fills no other Target.`; confirming it lands the forced file as `The Quiet Orchard (2021) {tmdb-9000001} - 1080p.mkv`, `bluray-720p`, deletes the WEB-DL file from disk, ends the Grab `Imported 1 of 1 movie`, and the Work page's 1080p cell reads `Upgrade available` over `Bluray-720p · 1280×720` with one file (P-078).
- `import-activity-waiting` Activity's Grabs row of a Grab whose file waits reads `Importing 0 of 1 movie` and a `Waiting for you` link, which opens `/library/pending/<import id>` with the file's sentence; the phone's card reads the same (P-083).
- `import-by-hand` with Harbor Lights added and its video copied into `Harbor Lights (2019)` in home's download folder, Pending imports' `Import files`, `Choose a folder`, the download folder's `Harbor Lights (2019)` and `Open Harbor Lights (2019)` list the file under `Harbor Lights` with `Fills 1080p for Harbor Lights.`, `Import`, `Other actions` and the `Move files` switch, off. `Import all` places `Harbor Lights (2019)/Harbor Lights (2019) - 1080p.mkv`, `webdl-1080p`, by hard link: the library file and the download share one inode with two links (P-077, P-084, P-085).
- `import-by-hand-outside` the folder browser refuses a folder in neither a library root nor a download folder, reading `The folder isn't in a library root or a download folder on the Node` with `Go up` and `Try again`, and the API answers 422 (P-084).
- `import-settings` Settings > Library > Importing reads the rule under `Details`: `Video file extensions`, `Smallest video file` with its unit and `Sample when shorter than`. There is no validation level or match conflict setting: the probe before import is the one check, and a file whose reading conflicts waits for review (P-087, P-088, P-089).
- `import-file-details` a library file's `Details` read `Placed` with how its import put it there, `Hard linked. The download and the library share the same data on disk.` for the by-hand import and for the forced file on one Node, `placedBy` `transfer` for the pack's episodes, and `From` `Grab The.Quiet.Orchard.2021.1080p.BluRay.x264-FKW` for a file a Grab brought, on desktop and phone. The files API answers `placedBy` `link` and `grab {id, release}`, null for a file no import recorded. Covered by `TestAnImportedFileListsHowItWasPlacedAndAFoundFileNothing` in `internal/files` (P-082).
- `import-match-retry` Change the match on a waiting file, Retry on a held or failed one and the Import's backoff aren't driven by this file's recipe. Covered by `TestEveryOfferedHoldOffersRetryAndDontImport` and `TestEveryMoveIsBetweenOutcomesAndNoFinalOutcomeMovesButARetryOrASettingsWait` in `internal/imports` and `web/src/imports/pending-page.test.tsx` (P-081, P-086).
- `import-client-removed` a torrent its client removes before anything is placed fails its Grab `client_removed`, and a completed download hands off only once its client reports it complete. Not driven by this recipe; covered by `TestAGrabFailsAtAnyStageOnceItsTorrentIsGoneWithoutAnAdminsRemove` in `internal/grabs`, `TestAVanishedTorrentPartsOnceAsRemovedByItsClient` in `internal/downloadclients` and [Download Clients](./download-clients.md) (P-076, P-170).

## How to get to it (user POV)

- Desktop shell: Activity's `Jobs` tab, the `Grabs` table's `Stage` and `Progress` cells and their `Waiting for you` link; the Library's `<n> pending imports need you` row, which opens `/library/pending`, `Import files` in its top bar and each download's `Other actions for <file>` menu; a Work page's `Editions and Targets` cell for a movie, and a series' episode cell button, whose popover reads the held step; Settings > Library > `Importing`.
- Phone shell: the `Activity` tab's cards; the `Library` tab's row, which pushes `Pending imports`, each download a row that pushes its own page; a series episode's row opens its sheet with the cell's facts.
- API (Admin): `GET /api/v1/grabs` with each Grab's `import {id, placed, planned, sending, waiting, words}`, `GET /api/v1/imports?source=grab&waiting=true`, `GET /api/v1/imports/{id}` with each file's `outcome`, `reason {code, sentence}`, `recommended`, `actions` and `forces`, `POST /api/v1/imports/{id}/files/{position}` with `action` and for `force` a `cell`, `POST /api/v1/imports {nodeId, folder}` and `GET /api/v1/works/{id}/files` with `placedBy` and `grab`.
- Host CLI: none.

## Driving it with fakeworld, curl and playwright cli

Preconditions:

- A `packs` run for the pack, launched with `--scenario packs` in place of `home`, and an `imports` run for the waiting download, Force and the manual import. Each with doctor passing and `{runId}-admin-desktop` and `{runId}-admin-phone` signed in as in the skill's Drive.
- For screenshots at twice the pixels, open each session from a copy of its config with `deviceScaleFactor` 2, and pass `--hires` to `screenshot`, which otherwise saves CSS pixels:

  ```sh
  for sh in desktop phone; do jq '.browser.contextOptions.deviceScaleFactor = 2' $RUN/browser/$sh/playwright-cli.json > $RUN/browser/$sh/dsf2.json; done
  D=-s={runId}-admin-desktop
  P=-s={runId}-admin-phone
  ```

Steps on `packs`:

- **Slow the Transfer.** Without a send limit the 64 MB pack crosses loopback in about a second. 256 KB/s gives a minute or more to read each step.

  ```sh
  SB=$(jq -r '.nodes[] | select(.name == "seedbox").nodeId' $RUN/manifest.json)
  $FW exec $RUN --proof imports/import-pack-two-nodes -- curl -sS -X PATCH -H 'X-Api-Key: {apiKey}' -H 'Content-Type: application/json' -d '{"sendLimit":262144}' "{nodes.home.url}/api/v1/transfers/nodes/$SB/settings"
  W=$($FW exec $RUN -- curl -sS -X POST -H 'X-Api-Key: {apiKey}' -H 'Content-Type: application/json' -d '{"contentType":"series","key":"1101"}' '{nodes.home.url}/api/v1/works' | jq -r .work.id)
  ```

- **Search Season 1 and watch it import.** The Grab reaches `importing` within about 15 seconds. Read Activity, then an episode's cell while it sends.

  ```sh
  $FW exec $RUN -- $PW cli $D goto "{nodes.home.browserUrl}/works/$W"
  $FW exec $RUN -- $PW cli $D click 'getByRole("main").getByRole("button", { name: "Search Season 1", exact: true })'
  $FW exec $RUN -- $PW cli $D goto '{nodes.home.browserUrl}/activity'
  $FW exec $RUN --proof imports/import-pack-progress -- $PW cli $D run-code 'async page => { await page.getByRole("table", { name: "Grabs" }).getByText(/^Importing \d+ of 10 episodes$/).waitFor({ timeout: 60000 }); }'
  $FW exec $RUN --proof imports/import-pack-progress -- $PW cli $D screenshot --hires --filename={runDir}/proof/imports/import-pack-progress/activity-desktop.png
  $FW exec $RUN -- $PW cli $D goto "{nodes.home.browserUrl}/works/$W"
  $FW exec $RUN -- $PW cli $D click 'getByRole("main").getByRole("button", { name: "S01E01 1080p, Downloading" })'
  $FW exec $RUN --proof imports/import-held-step -- $PW cli $D snapshot 'getByRole("dialog", { name: "S01E01 1080p" })' --filename={runDir}/proof/imports/import-held-step/pack-cell.aria.yml
  $FW exec $RUN -- $PW cli $P goto "{nodes.home.browserUrl}/works/$W"
  $FW exec $RUN -- $PW cli $P click 'getByRole("main").getByRole("button", { name: "S01E02 The Wreckers" })'
  $FW exec $RUN --proof imports/import-held-step -- $PW cli $P screenshot --hires --filename={runDir}/proof/imports/import-held-step/pack-sheet-phone.png
  ```

- **See the pack landed.** Lift the limit, wait for `imported`, list the files, and compare an episode with its download on seedbox. `fs match --release` compares a file with the whole Release, so for a pack compare `fs stat` of both sides.

  ```sh
  $FW exec $RUN -- curl -sS -X PATCH -H 'X-Api-Key: {apiKey}' -H 'Content-Type: application/json' -d '{"sendLimit":0}' "{nodes.home.url}/api/v1/transfers/nodes/$SB/settings"
  $FW exec $RUN -- bash -c "for i in \$(seq 120); do curl -sS -H 'X-Api-Key: {apiKey}' '{nodes.home.url}/api/v1/grabs' | jq -e '.grabs[0].stage == \"imported\"' >/dev/null && exit 0; sleep 1; done; exit 1"
  $FW exec $RUN --proof imports/import-pack-two-nodes -- curl -sS -H 'X-Api-Key: {apiKey}' "{nodes.home.url}/api/v1/works/$W/files"
  SRC=$(find "$(jq -r '.nodes[] | select(.name == "seedbox").dataDir' $RUN/manifest.json)/downloads" -name '*S01E05*.mkv')
  $FW exec $RUN --proof imports/import-pack-two-nodes -- $FW fs stat {runDir} "$SRC"
  $FW exec $RUN --proof imports/import-pack-two-nodes -- $FW fs stat {runDir} '{mediaDir}/tv/The Copper Coast (2022)/Season 01/The Copper Coast - S01E05 - The Harbor Master 1080p.mkv'
  ```

- **Bare-numbered files.** Search Season 2 from the Work page; its section sits below the fold, so scroll before the click. The Grab stops with 3 files waiting, and Activity's `Waiting for you` opens its Pending import.

  ```sh
  $FW exec $RUN -- $PW cli $D run-code 'async page => { await page.mouse.wheel(0, 1600); await page.getByRole("button", { name: "Search Season 2", exact: true }).click({ timeout: 10000 }); }'
  $FW exec $RUN -- $PW cli $D goto '{nodes.home.browserUrl}/activity'
  $FW exec $RUN -- $PW cli $D click 'getByRole("table", { name: "Grabs" }).getByRole("link", { name: "Waiting for you" })'
  $FW exec $RUN --proof imports/import-no-number -- $PW cli $D snapshot 'getByRole("main")' --filename={runDir}/proof/imports/import-no-number/pending.aria.yml
  ```

- **Movie packs.** Add The Glass Meridian (`9100001`) and Long Division (`9100002`) and search each; read each Grab's Import with `GET /api/v1/imports/{id}`.

Steps on `imports`:

- **A download that isn't an upgrade.** Search The Quiet Orchard from its hero. The Grab stops at `importing` with one file waiting, usually within 30 seconds.

  ```sh
  Q=$($FW exec $RUN -- curl -sS -H 'X-Api-Key: {apiKey}' '{nodes.home.url}/api/v1/library' | jq -r '.rows[] | select(.label == "The Quiet Orchard") | .id')
  $FW exec $RUN -- $PW cli $D goto "{nodes.home.browserUrl}/works/$Q"
  $FW exec $RUN -- $PW cli $D click 'getByRole("main").getByRole("button", { name: "Search", exact: true }).first()'
  $FW exec $RUN -- bash -c "for i in \$(seq 60); do curl -sS -H 'X-Api-Key: {apiKey}' '{nodes.home.url}/api/v1/grabs' | jq -e '.grabs[0].import.waiting == 1' >/dev/null && exit 0; sleep 2; done; exit 1"
  ```

- **Activity and the held cell.** Activity's row reads the Import and its link; the Work page's cell names the step.

  ```sh
  $FW exec $RUN -- $PW cli $D goto '{nodes.home.browserUrl}/activity'
  $FW exec $RUN --proof imports/import-activity-waiting -- $PW cli $D snapshot 'getByRole("table", { name: "Grabs" })' --filename={runDir}/proof/imports/import-activity-waiting/activity.aria.yml
  $FW exec $RUN --proof imports/import-activity-waiting -- $PW cli $D screenshot --hires --filename={runDir}/proof/imports/import-activity-waiting/activity-desktop.png
  $FW exec $RUN -- $PW cli $D goto "{nodes.home.browserUrl}/works/$Q"
  $FW exec $RUN --proof imports/import-held-step -- $PW cli $D snapshot 'getByRole("table", { name: "Editions and Targets" })' --filename={runDir}/proof/imports/import-held-step/cell.aria.yml
  ```

- **Force into 1080p.** From the Library's row, open the download's `Other actions`, pick `Force into 1080p` and confirm. The download leaves the list and the forced file replaces the WEB-DL one.

  ```sh
  V=The.Quiet.Orchard.2021.1080p.BluRay.x264-FKW.mkv
  $FW exec $RUN -- $PW cli $D goto '{nodes.home.browserUrl}/library'
  $FW exec $RUN -- $PW cli $D click 'getByRole("link", { name: /^1 pending import needs you/ })'
  $FW exec $RUN --proof imports/import-not-an-upgrade -- $PW cli $D snapshot 'getByRole("main")' --filename={runDir}/proof/imports/import-not-an-upgrade/pending.aria.yml
  $FW exec $RUN -- $PW cli $D click "getByRole(\"button\", { name: \"Other actions for $V\" })"
  $FW exec $RUN -- $PW cli $D click 'getByRole("menuitem", { name: "Force into 1080p", exact: true })'
  $FW exec $RUN --proof imports/import-force -- $PW cli $D snapshot 'getByRole("dialog", { name: "Force into 1080p?" })' --filename={runDir}/proof/imports/import-force/confirm.aria.yml
  $FW exec $RUN -- $PW cli $D click 'getByRole("dialog", { name: "Force into 1080p?" }).getByRole("button", { name: "Force into 1080p", exact: true })'
  $FW exec $RUN --proof imports/import-force -- $PW cli $D run-code 'async page => { await page.getByRole("listitem", { name: "The.Quiet.Orchard.2021.1080p.BluRay.x264-FKW" }).waitFor({ state: "hidden", timeout: 60000 }); }'
  $FW exec $RUN --proof imports/import-force -- curl -sS -H 'X-Api-Key: {apiKey}' "{nodes.home.url}/api/v1/works/$Q/files"
  ```

- **Import a folder by hand.** Add Harbor Lights, copy the shelf's video into a folder of home's download folder, then Import files, choose that folder and Import all. Wait for the toast in the same `run-code` as the click, since it can be gone by the next step.

  ```sh
  H=$($FW exec $RUN -- curl -sS -X POST -H 'X-Api-Key: {apiKey}' -H 'Content-Type: application/json' -d '{"contentType":"movie","key":"9000002"}' '{nodes.home.url}/api/v1/works' | jq -r .work.id)
  $FW exec $RUN --proof imports/import-by-hand -- mkdir -p '{nodes.home.dataDir}/downloads/Harbor Lights (2019)'
  $FW exec $RUN --proof imports/import-by-hand -- cp '{mediaDir}/shelf/Harbor.Lights.2019.1080p.WEB-DL.H264-FKW.mkv' '{nodes.home.dataDir}/downloads/Harbor Lights (2019)/'
  B='getByRole("dialog", { name: "Choose a folder on home" })'
  $FW exec $RUN -- $PW cli $D goto '{nodes.home.browserUrl}/library/pending'
  $FW exec $RUN -- $PW cli $D click 'getByRole("button", { name: "Import files", exact: true })'
  $FW exec $RUN -- $PW cli $D click 'getByRole("dialog", { name: "Import files" }).getByRole("button", { name: "Choose a folder", exact: true })'
  $FW exec $RUN -- $PW cli $D click "$B"'.getByRole("listbox", { name: "Folders" }).getByRole("option").filter({ hasText: "/home/downloads" })'
  $FW exec $RUN -- $PW cli $D click "$B"'.getByRole("listbox", { name: "Folders" }).getByRole("option", { name: "Harbor Lights (2019)", exact: true })'
  $FW exec $RUN -- $PW cli $D click "$B"'.getByRole("button", { name: "Open Harbor Lights (2019)", exact: true })'
  $FW exec $RUN -- $PW cli $D run-code 'async page => { await page.getByRole("dialog", { name: "Import files" }).getByRole("button", { name: "Import all", exact: true }).waitFor({ timeout: 30000 }); }'
  $FW exec $RUN --proof imports/import-by-hand -- $PW cli $D snapshot 'getByRole("dialog", { name: "Import files" })' --filename={runDir}/proof/imports/import-by-hand/sheet.aria.yml
  $FW exec $RUN --proof imports/import-by-hand -- $PW cli $D run-code 'async page => { await page.getByRole("dialog", { name: "Import files" }).getByRole("button", { name: "Import all", exact: true }).click(); await page.getByRole("region", { name: "Notifications" }).getByText("Importing 1 file.").waitFor({ timeout: 10000 }); }'
  $FW exec $RUN --proof imports/import-by-hand -- curl -sS -H 'X-Api-Key: {apiKey}' "{nodes.home.url}/api/v1/works/$H/files"
  $FW exec $RUN --proof imports/import-by-hand -- $FW fs stat {runDir} '{mediaDir}/movies/Harbor Lights (2019)/Harbor Lights (2019) - 1080p.mkv'
  ```

- **How a file was placed.** Open the file's `Details` on each Work page: Harbor Lights' reads `Placed` `Hard linked. …`, and The Quiet Orchard's forced file adds `From` `Grab The.Quiet.Orchard.2021.1080p.BluRay.x264-FKW`.

  ```sh
  $FW exec $RUN -- $PW cli $D goto "{nodes.home.browserUrl}/works/$Q"
  $FW exec $RUN -- $PW cli $D click 'getByRole("list", { name: "Files" }).getByRole("button", { name: "Details" })'
  $FW exec $RUN --proof imports/import-file-details -- $PW cli $D snapshot 'getByRole("list", { name: "Files" })' --filename={runDir}/proof/imports/import-file-details/details.aria.yml
  ```

- **Phone.** Open `{nodes.home.browserUrl}/library/pending/<import id>` and the Activity tab on `{runId}-admin-phone`: the download is a pushed page with the same sentence and actions, and the Grab's card reads the Import's words and `Waiting for you`.

## Gotchas

- `fakeworld fs cp` writes only inside the media dir, and Import files browses only library roots and download folders, so a folder to import by hand goes into `{nodes.<n>.dataDir}/downloads` with `cp` through `exec`. A folder under the media dir that no root names reads `The folder isn't in a library root or a download folder on the Node`.
- `screenshot` saves CSS pixels whatever the session's `deviceScaleFactor`; only `--hires` saves device pixels.
- `fs match --release` compares a file with the Release's whole payload, so it fails on every file of a multi-file Release. Compare `fs stat` of the placed file and its download instead.
- A `{...}` in an `exec` argument is a manifest token, so `curl -w '%{http_code}'` fails with `names nothing in the manifest`; write `%{{http_code}`, since `{{` is a literal brace.
- A series' Seasons list renders only what is near the viewport, so a later season's `Search Season <n>` matches nothing until the page scrolls to it.
- A held cell reads `Downloading` as its chip through the whole Import, sending and waiting included; the step is in the facts under it.
- Known gaps a run will meet. These are not recipe failures:
  - A Grab's file that waits `no_number` or reads as an episode the series lacks carries the Library scan's reason: it reads `The file name has no episode number.` without the folder's season, offers no action, and the Grab waits at `importing` until the Admin removes its torrent.
  - A movie cell held by a waiting file reads `Waiting for you` twice: the Import's link and the step under it.
  - Activity's `For` cell lists the Grab's Target once per Item, so a 10-episode pack reads `1080p` ten times.
