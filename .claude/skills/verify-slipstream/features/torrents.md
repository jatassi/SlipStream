# Torrents

Activity's Torrents view lists every torrent the downloading Nodes' built-in clients hold, live, problems first: a status sentence, the signal rows of what needs the Admin, tabs by state, filters, search and a table that drops from eight columns to four and then to cards as the window narrows. Selecting a row opens the Inspector beside the table on a desktop and as a pushed page on a phone: what the torrent is doing in a sentence, its numbers, trackers, files, Grabs and details. The Admin pauses, resumes, rechecks, marks failed or removes one torrent or a picked set, from the row menu, the Inspector, the picked toolbar or a Job's menu. A resume that a private tracker would reject waits and says when it starts. Removing asks first, says library files stay, lists what it cancels and can delete the download copy. Each Node runs a set number of downloads at once and queues the rest, and a restart resumes every torrent without losing its totals.

## Sub-features

- `torrents-queue` with `Downloads at once` 1, a second and third Grab read `Queued` with `1st in line` and `2nd in line`, and start once the limit is 3 again.
- `torrents-view` `/activity/torrents` leads with the sentence (`1 downloading · 1 failure · 1 warning`), then `Needs you` with a signal row per failure and warning (`Unregistered` Harbor Lights, `Stalled` Salt Meadow), the `Show` tabs `Downloading`, `Seeding`, `Problems` and `All` with counts, the search box and the table; each row names its Work with its quality, a state word over a line (`Downloading` over `1 peer`), the live bar with percent, time left and speed, ratio, speeds, the tracker host over its status, and the Node over `Built-in` (P-164).
- `torrents-columns` the table shows Work, State, Progress, Ratio, Speed, Tracker and Node · client at 1280 px, drops to Work, State and Progress at 980 px, and stacks each row as a card at 620 px and on a phone.
- `torrents-live` the Admin's stream carries `torrents.live` each second while a torrent moves, and the row's percent and the Activity count follow without a reload (P-167).
- `torrents-filters` a tab, `?type=movie` (the `Content Type` select, shown once set) and a search narrow the table, and the URL keeps them (P-165).
- `torrents-inspector` a row opens the Inspector at `/activity/torrents/<id>` with `Downloading from 1 peer, <n> min left at 16 KB/s.`, `Pause`, `Remove` and `More actions for <Work>`, then Numbers, Trackers with the passkey hidden as `/announce/•••`, Files, Grabs and Details, whose save folder is `<downloads>/Movies/<info hash>` (P-146); ArrowDown steps to the next row with the Inspector following; on a phone it is a pushed page with the back link `Torrents`.
- `torrents-pause-resume` `Pause` reads `Paused` and the tracker gets `stopped`; `Resume` inside the tracker's window reads `Resumes at <HH:MM>`, then starts, and the UNIT3D tracker accepts every announce of it (P-166).
- `torrents-unregistered` a release the tracker calls unregistered reads `Tracker error` over `Unregistered by 127.0.0.1` and is a failure in `Needs you`; once the tracker says so twice the unregistered gap apart, the Inspector adds `The release looks bad: its tracker says the torrent is no longer registered.`
- `torrents-bulk` picking two rows shows the toolbar `Picked torrents` with `2 picked`, and its `Pause` and `Resume` act on both.
- `torrents-mark-failed` a download fed only corrupt pieces by a bad peer reads `Stalled` with `No data is arriving` after `Stalled after`; `Mark failed` asks `Mark Salt Meadow failed?`, removes it with its download copy and fails its Grab `download_failed`.
- `torrents-jobs` Jobs shows each Grab's quality and Target chips and its live percent; its menu offers `Open its torrent`, which opens the Inspector (P-164).
- `torrents-restart` restarting home stops each torrent with `stopped`, and the seeding torrent comes back `Seeding` with a `started` that reports nothing left and the same totals.
- `torrents-remove` `Remove` asks `Remove The Quiet Orchard?`, says `Files in your library stay.` and lists the Work that stops seeding; with `Also delete the download copy` its save folder goes, the library file still matches the release, the tracker gets `stopped`, and the row leaves once home confirms (P-166).

## How to get to it (user POV)

- Desktop shell: the sidebar's `Activity`, then the page tab `Torrents` at `/activity/torrents`, or `?tab=all|seeding|problems`. A Node's `Torrent client` section links `Torrents` filtered to that Node (`?node=<id>&tab=all`).
- Phone shell: the `Activity` tab, then `Torrents` in the segmented control; the tabs list All first, the filters sit behind `Filter`, a row is a card, and the Inspector is a pushed page.
- Jobs: `/activity`, a Job's `<Work> actions` menu, `Open its torrent`.
- API: `GET /api/v1/torrents` (`?tab=&nodeId=&contentType=&q=&ids=`), `GET /api/v1/torrents/{id}`, `POST /api/v1/torrents/actions` with `{action, ids, deleteFiles?, dryRun?}`, and the events `torrents.changed` and `torrents.live`.

## Driving it with fakeworld and playwright cli

Preconditions:

- A `torrents` run (`--scenario torrents`): home is the Controller and the only Node, downloading through a tracker that plays UNIT3D's rules with a 2 minute min interval. The Quiet Orchard's release has seeders, Harbor Lights' is unregistered at the tracker, and Salt Meadow's comes only from a bad peer that sends corrupt pieces. Doctor passes.
- The `{runId}-admin-desktop` and `{runId}-admin-phone` sessions open and signed in, as in the skill's Drive, and these handles:

  ```sh
  D=-s={runId}-admin-desktop
  P=-s={runId}-admin-phone
  API="$(jq -r '.nodes[] | select(.name == "home").url' $RUN/manifest.json)/api/v1"
  KEY="X-Api-Key: $(jq -r .apiKey $RUN/manifest.json)"
  idOf() { curl -sS -H "$KEY" "$API/torrents" | jq -r --arg n "$1" '.rows[] | select(.name | startswith($n)) | .id'; }
  ```

Steps:

- **Slow home down and queue the downloads.** On home's Torrent client, save `Download limit` 16 (KB/s), `Downloads at once` 1 and `Stalled after` 1 (min), then grab the three Works through the API. The Quiet Orchard downloads, and the others read `Queued` with `1st in line` and `2nd in line`.

  ```sh
  $FW exec $RUN -- $PW cli $D goto '{nodes.home.browserUrl}/nodes/{nodes.home.nodeId}/torrent-client'
  $FW exec $RUN -- $PW cli $D fill 'getByRole("textbox", { name: "Download limit" })' 16
  $FW exec $RUN -- $PW cli $D click 'getByRole("region", { name: "Speed" }).getByRole("button", { name: "Save" })'
  $FW exec $RUN -- $PW cli $D fill 'getByRole("textbox", { name: "Downloads at once" })' 1
  $FW exec $RUN -- $PW cli $D fill 'getByRole("textbox", { name: "Stalled after" })' 1
  $FW exec $RUN -- $PW cli $D click 'getByRole("region", { name: "Queue" }).getByRole("button", { name: "Save" })'
  $FW exec $RUN --proof torrents/torrents-queue -- curl -sS -H 'X-Api-Key: {apiKey}' '{nodes.home.url}/api/v1/download-clients/{nodes.home.nodeId}/settings'
  for key in 9000001 9000002 9000003; do $FW exec $RUN --proof torrents/torrents-queue -- sh -c "id=\$(curl -sS -X POST -H 'X-Api-Key: {apiKey}' -H 'Content-Type: application/json' -d '{\"contentType\":\"movie\",\"key\":\"$key\",\"monitor\":true}' '{nodes.home.url}/api/v1/works' | jq -r .work.id); curl -sS -X POST -H 'X-Api-Key: {apiKey}' \"{nodes.home.url}/api/v1/works/\$id/search\""; sleep 2; done
  $FW exec $RUN -- $PW cli $D goto '{nodes.home.browserUrl}/activity/torrents'
  $FW exec $RUN --proof torrents/torrents-queue -- $PW cli $D run-code 'async page => { await page.getByRole("table", { name: "Torrents" }).getByText("2nd in line").waitFor({ timeout: 30000 }); }'
  $FW exec $RUN --proof torrents/torrents-queue -- $PW cli $D snapshot 'getByRole("table", { name: "Torrents" })' --filename={runDir}/proof/torrents/torrents-queue/queued.aria.yml
  $FW exec $RUN --proof torrents/torrents-queue -- $PW cli $D screenshot --filename={runDir}/proof/torrents/torrents-queue/queued.png
  ```

- **Let all three run.** Save `Downloads at once` 3. Harbor Lights' started is refused as unregistered, and Salt Meadow stalls on the bad peer's corrupt pieces a minute later.

  ```sh
  $FW exec $RUN -- $PW cli $D goto '{nodes.home.browserUrl}/nodes/{nodes.home.nodeId}/torrent-client'
  $FW exec $RUN -- $PW cli $D fill 'getByRole("textbox", { name: "Downloads at once" })' 3
  $FW exec $RUN -- $PW cli $D click 'getByRole("region", { name: "Queue" }).getByRole("button", { name: "Save" })'
  $FW announces wait $RUN --node home --where rule=unregistered --timeout 60s
  ```

- **Read the view.** Torrents leads with `1 downloading · 1 failure · 1 warning` once Salt Meadow stalls, and `Needs you` holds Harbor Lights `Unregistered` and Salt Meadow `Stalled`. The API reads the same states. On the phone, `Torrents` is a segment, `All` is the first tab and each row is a card.

  ```sh
  $FW exec $RUN -- $PW cli $D goto '{nodes.home.browserUrl}/activity/torrents?tab=all'
  $FW exec $RUN --proof torrents/torrents-view -- $PW cli $D run-code 'async page => { await page.getByText("1 warning").waitFor({ timeout: 120000 }); return page.getByRole("main").getByRole("paragraph").first().textContent(); }'
  $FW exec $RUN --proof torrents/torrents-view -- $PW cli $D snapshot 'getByRole("main")' --filename={runDir}/proof/torrents/torrents-view/desktop.aria.yml
  $FW exec $RUN --proof torrents/torrents-view -- $PW cli $D screenshot --filename={runDir}/proof/torrents/torrents-view/desktop.png
  $FW exec $RUN --proof torrents/torrents-view -- curl -sS -H 'X-Api-Key: {apiKey}' '{nodes.home.url}/api/v1/torrents'
  $FW exec $RUN -- $PW cli $P goto '{nodes.home.browserUrl}/activity/torrents'
  $FW exec $RUN --proof torrents/torrents-view -- $PW cli $P snapshot 'getByRole("main")' --filename={runDir}/proof/torrents/torrents-view/phone.aria.yml
  $FW exec $RUN --proof torrents/torrents-view -- $PW cli $P screenshot --full-page --filename={runDir}/proof/torrents/torrents-view/phone.png
  ```

- **Narrow the window.** At 1280 px all seven named columns show; at 980 px only Work, State and Progress; at 620 px the header goes and each row is a card whose cells carry their labels.

  ```sh
  for w in 1280 980 620; do
    $FW exec $RUN -- $PW cli $D resize $w 900
    $FW exec $RUN --proof torrents/torrents-columns -- $PW cli $D run-code 'async page => { const headers = page.getByRole("table", { name: "Torrents" }).getByRole("columnheader"); const shown = []; for (const h of await headers.all()) { if (await h.isVisible()) shown.push((await h.textContent()).trim()); } return shown; }'
    $FW exec $RUN --proof torrents/torrents-columns -- $PW cli $D screenshot --filename={runDir}/proof/torrents/torrents-columns/$w.png
  done
  $FW exec $RUN -- $PW cli $D resize 1440 900
  ```

- **Watch it move.** The stream carries `torrents.live` with The Quiet Orchard's `done` climbing, and its row's percent follows.

  ```sh
  $FW sse wait $RUN --as admin --event torrents.live --timeout 30s
  $FW exec $RUN --proof torrents/torrents-live -- $PW cli $D run-code 'async page => { const cell = page.getByRole("table", { name: "Torrents" }).getByRole("row").filter({ hasText: "The Quiet Orchard" }).getByRole("progressbar"); const before = await cell.getAttribute("aria-valuenow"); await page.waitForFunction(([b]) => document.querySelector("[data-torrent-row] [role=progressbar]")?.getAttribute("aria-valuenow") !== b, [before], { timeout: 30000 }); return [before, await cell.getAttribute("aria-valuenow")]; }'
  $FW exec $RUN --proof torrents/torrents-live -- $FW sse list $RUN --as admin --event torrents.live
  ```

- **Filter and search.** `Problems` lists Harbor Lights and Salt Meadow; `?type=movie` shows the `Content Type` select reading `Movies` with every row; searching `harbor` leaves Harbor Lights alone, and the URL keeps `q=harbor`.

  ```sh
  $FW exec $RUN -- $PW cli $D click 'getByRole("radiogroup", { name: "Show" }).getByRole("radio", { name: /^Problems/ })'
  $FW exec $RUN --proof torrents/torrents-filters -- $PW cli $D snapshot 'getByRole("table", { name: "Torrents" })' --filename={runDir}/proof/torrents/torrents-filters/problems.aria.yml
  $FW exec $RUN -- $PW cli $D goto '{nodes.home.browserUrl}/activity/torrents?tab=all&type=movie'
  $FW exec $RUN --proof torrents/torrents-filters -- $PW cli $D snapshot 'getByRole("main")' --filename={runDir}/proof/torrents/torrents-filters/type.aria.yml
  $FW exec $RUN -- $PW cli $D fill 'getByRole("searchbox", { name: "Search torrents" })' harbor
  $FW exec $RUN --proof torrents/torrents-filters -- $PW cli $D run-code 'async page => { await page.waitForURL(/q=harbor/, { timeout: 10000 }); return [page.url(), await page.getByRole("table", { name: "Torrents" }).getByRole("row").count()]; }'
  $FW exec $RUN --proof torrents/torrents-filters -- $PW cli $D screenshot --filename={runDir}/proof/torrents/torrents-filters/search.png
  ```

- **Inspect a torrent.** On `All`, choose The Quiet Orchard's row. The Inspector leads with `Downloading from 1 peer, …`, lists the tracker as `127.0.0.1:<port>` with its URL under `Details` ending `/announce/•••`, and Details' `Save folder` ends `downloads/Movies/<info hash>`. ArrowDown moves the selection and the URL to the next row. On the phone the row opens a pushed page with the back link `Torrents`.

  ```sh
  $FW exec $RUN -- $PW cli $D goto '{nodes.home.browserUrl}/activity/torrents?tab=all'
  $FW exec $RUN -- $PW cli $D click 'getByRole("table", { name: "Torrents" }).getByRole("button", { name: /^The Quiet Orchard(?! actions)/ })'
  $FW exec $RUN -- $PW cli $D click 'getByRole("complementary", { name: "The Quiet Orchard" }).getByRole("listitem").filter({ hasText: "127.0.0.1" }).getByRole("button", { name: "Details" })'
  $FW exec $RUN --proof torrents/torrents-inspector -- $PW cli $D snapshot 'getByRole("complementary", { name: "The Quiet Orchard" })' --filename={runDir}/proof/torrents/torrents-inspector/desktop.aria.yml
  $FW exec $RUN --proof torrents/torrents-inspector -- $PW cli $D screenshot --filename={runDir}/proof/torrents/torrents-inspector/desktop.png
  $FW exec $RUN --proof torrents/torrents-inspector -- curl -sS -H "$KEY" "$API/torrents/$(idOf The.Quiet)"
  $FW exec $RUN -- $PW cli $D press ArrowDown
  $FW exec $RUN --proof torrents/torrents-inspector -- $PW cli $D run-code 'async page => [page.url(), await page.getByRole("complementary").getByRole("heading", { level: 2 }).first().textContent()]'
  $FW exec $RUN -- $PW cli $P goto '{nodes.home.browserUrl}/activity/torrents?tab=all'
  $FW exec $RUN -- $PW cli $P click 'getByRole("button", { name: /^The Quiet Orchard(?! actions)/ })'
  $FW exec $RUN --proof torrents/torrents-inspector -- $PW cli $P snapshot 'getByRole("banner")' --filename={runDir}/proof/torrents/torrents-inspector/phone-banner.aria.yml
  $FW exec $RUN --proof torrents/torrents-inspector -- $PW cli $P screenshot --full-page --filename={runDir}/proof/torrents/torrents-inspector/phone.png
  ```

- **Pause, resume, then resume inside the window.** From The Quiet Orchard's row menu choose `Pause`: the row reads `Paused` and the tracker gets `stopped`. `Resume` starts it again with a `started`, since its last one is older than the window. Half a minute after that `stopped`, `Pause` again, then `Resume` at once: the row reads `Resumes at <HH:MM>`, 2 minutes after the `started`, with no time left. When the window passes it starts, and the tracker accepted every announce of it.

  ```sh
  pick() { $FW exec $RUN -- $PW cli $D click 'getByRole("button", { name: "The Quiet Orchard actions" })'; $FW exec $RUN -- $PW cli $D click "getByRole(\"menuitem\", { name: \"$1\" })"; }
  $FW exec $RUN -- $PW cli $D goto '{nodes.home.browserUrl}/activity/torrents?tab=all'
  pick Pause
  $FW exec $RUN --proof torrents/torrents-pause-resume -- $PW cli $D run-code 'async page => { await page.getByRole("table", { name: "Torrents" }).getByRole("row").filter({ hasText: "The Quiet Orchard" }).getByRole("cell", { name: /^Paused/ }).waitFor({ timeout: 15000 }); }'
  STOPPED=$($FW announces wait $RUN --node home --where event=stopped --timeout 30s | jq .seq)
  pick Resume
  STARTED=$($FW announces wait $RUN --node home --where event=started --after $STOPPED --timeout 30s | jq .seq)
  sleep 30
  pick Pause
  $FW announces wait $RUN --node home --where event=stopped --after $STARTED --timeout 30s
  pick Resume
  $FW exec $RUN --proof torrents/torrents-pause-resume -- $PW cli $D run-code 'async page => { const line = page.getByRole("table", { name: "Torrents" }).getByText(/^Resumes at/); await line.waitFor({ timeout: 15000 }); return line.textContent(); }'
  $FW exec $RUN --proof torrents/torrents-pause-resume -- $PW cli $D screenshot --filename={runDir}/proof/torrents/torrents-pause-resume/window.png
  $FW announces wait $RUN --node home --where event=started --after $STARTED --timeout 180s
  $FW exec $RUN --proof torrents/torrents-pause-resume -- sh -c "$FW announces list {runDir} --node home | jq -c 'select(.data.infoHash == \"'$(curl -sS -H "$KEY" "$API/torrents/$(idOf The.Quiet)" | jq -r .infoHash)'\") | {at, event: .data.event, accepted: .data.accepted}'"
  ```

- **Read the unregistered release.** Shorten `Unregistered twice, this far apart` to 1 min on Settings > Download Clients. Harbor Lights' row reads `Tracker error` over `Unregistered by 127.0.0.1`; after the tracker's next refusal, 3 minutes after the first, its Inspector adds the release fault.

  ```sh
  $FW exec $RUN -- $PW cli $D goto '{nodes.home.browserUrl}/settings/download-clients'
  $FW exec $RUN -- $PW cli $D fill 'getByRole("textbox", { name: "Unregistered twice, this far apart" })' 1
  $FW exec $RUN -- $PW cli $D click 'getByRole("main").getByRole("button", { name: "Save" })'
  $FW exec $RUN -- $PW cli $D goto '{nodes.home.browserUrl}/activity/torrents?tab=all'
  $FW exec $RUN -- $PW cli $D click 'getByRole("table", { name: "Torrents" }).getByRole("button", { name: /^Harbor Lights(?! actions)/ })'
  $FW exec $RUN --proof torrents/torrents-unregistered -- $PW cli $D run-code 'async page => { await page.getByText("The release looks bad: its tracker says the torrent is no longer registered.").waitFor({ timeout: 300000 }); }'
  $FW exec $RUN --proof torrents/torrents-unregistered -- $PW cli $D snapshot 'getByRole("complementary", { name: "Harbor Lights" })' --filename={runDir}/proof/torrents/torrents-unregistered/inspector.aria.yml
  $FW exec $RUN --proof torrents/torrents-unregistered -- $PW cli $D screenshot --filename={runDir}/proof/torrents/torrents-unregistered/inspector.png
  $FW exec $RUN --proof torrents/torrents-unregistered -- curl -sS -H "$KEY" "$API/torrents/$(idOf Harbor)"
  ```

- **Pause two at once.** Pick Harbor Lights and Salt Meadow: the toolbar `Picked torrents` reads `2 picked`. Its `Pause` pauses both, and its `Resume` resumes both.

  ```sh
  $FW exec $RUN -- $PW cli $D goto '{nodes.home.browserUrl}/activity/torrents?tab=all'
  $FW exec $RUN -- $PW cli $D click 'getByRole("checkbox", { name: "Pick Harbor Lights" })'
  $FW exec $RUN -- $PW cli $D click 'getByRole("checkbox", { name: "Pick Salt Meadow" })'
  $FW exec $RUN --proof torrents/torrents-bulk -- $PW cli $D snapshot 'getByRole("toolbar", { name: "Picked torrents" })' --filename={runDir}/proof/torrents/torrents-bulk/picked.aria.yml
  $FW exec $RUN -- $PW cli $D click 'getByRole("toolbar", { name: "Picked torrents" }).getByRole("button", { name: "Pause" })'
  $FW exec $RUN --proof torrents/torrents-bulk -- $PW cli $D run-code 'async page => { const rows = page.getByRole("table", { name: "Torrents" }).getByRole("row"); for (const work of ["Harbor Lights", "Salt Meadow"]) await rows.filter({ hasText: work }).getByRole("cell", { name: /^Paused/ }).waitFor({ timeout: 15000 }); }'
  $FW exec $RUN --proof torrents/torrents-bulk -- $PW cli $D screenshot --filename={runDir}/proof/torrents/torrents-bulk/paused.png
  $FW exec $RUN -- $PW cli $D click 'getByRole("checkbox", { name: "Pick Harbor Lights" })'
  $FW exec $RUN -- $PW cli $D click 'getByRole("checkbox", { name: "Pick Salt Meadow" })'
  $FW exec $RUN -- $PW cli $D click 'getByRole("toolbar", { name: "Picked torrents" }).getByRole("button", { name: "Resume" })'
  $FW exec $RUN --proof torrents/torrents-bulk -- $PW cli $D run-code 'async page => { await page.getByRole("table", { name: "Torrents" }).getByRole("row").filter({ hasText: "Salt Meadow" }).getByRole("cell", { name: /^(Downloading|Stalled|Paused Resumes)/ }).waitFor({ timeout: 15000 }); }'
  ```

- **Mark the bad release failed.** Once Salt Meadow reads `Stalled` again, on the phone open its card's menu and `Mark failed`; the dialog `Mark Salt Meadow failed?` lists its Grab. Confirm: the row leaves, the tracker gets `stopped`, and its Grab fails `download_failed`.

  ```sh
  $FW exec $RUN -- $PW cli $P goto '{nodes.home.browserUrl}/activity/torrents?tab=all'
  $FW exec $RUN -- $PW cli $P run-code 'async page => { await page.getByRole("table", { name: "Torrents" }).getByRole("row").filter({ hasText: "Salt Meadow" }).getByText("No data is arriving").waitFor({ timeout: 150000 }); }'
  $FW exec $RUN -- $PW cli $P click 'getByRole("button", { name: "Salt Meadow actions" })'
  $FW exec $RUN -- $PW cli $P click 'getByRole("dialog", { name: "Salt Meadow actions" }).getByRole("button", { name: "Mark failed" })'
  $FW exec $RUN --proof torrents/torrents-mark-failed -- $PW cli $P snapshot 'getByRole("dialog", { name: "Mark Salt Meadow failed?" })' --filename={runDir}/proof/torrents/torrents-mark-failed/dialog.aria.yml
  $FW exec $RUN --proof torrents/torrents-mark-failed -- $PW cli $P screenshot --filename={runDir}/proof/torrents/torrents-mark-failed/dialog-phone.png
  $FW exec $RUN -- $PW cli $P click 'getByRole("dialog", { name: "Mark Salt Meadow failed?" }).getByRole("button", { name: "Mark failed" })'
  $FW exec $RUN --proof torrents/torrents-mark-failed -- $PW cli $P run-code 'async page => { await page.getByRole("table", { name: "Torrents" }).getByRole("row").filter({ hasText: "Salt Meadow" }).waitFor({ state: "detached", timeout: 30000 }); }'
  $FW exec $RUN --proof torrents/torrents-mark-failed -- curl -sS -H 'X-Api-Key: {apiKey}' '{nodes.home.url}/api/v1/grabs'
  ```

- **Finish The Quiet Orchard and read Jobs.** Save `Download limit` 0. The Quiet Orchard completes, imports and reads `Seeding` on `Seeding`. Jobs shows its quality `WEBDL-1080p` and Target `1080p` chips, and its menu's `Open its torrent` opens its Inspector.

  ```sh
  $FW exec $RUN -- $PW cli $D goto '{nodes.home.browserUrl}/nodes/{nodes.home.nodeId}/torrent-client'
  $FW exec $RUN -- $PW cli $D fill 'getByRole("textbox", { name: "Download limit" })' 0
  $FW exec $RUN -- $PW cli $D click 'getByRole("region", { name: "Speed" }).getByRole("button", { name: "Save" })'
  $FW exec $RUN -- $PW cli $D goto '{nodes.home.browserUrl}/activity'
  $FW exec $RUN --proof torrents/torrents-jobs -- $PW cli $D run-code 'async page => { await page.getByRole("table", { name: "Grabs" }).getByRole("row").filter({ hasText: "The Quiet Orchard" }).getByText("Imported").waitFor({ timeout: 120000 }); }'
  $FW exec $RUN --proof torrents/torrents-jobs -- $PW cli $D snapshot 'getByRole("table", { name: "Grabs" })' --filename={runDir}/proof/torrents/torrents-jobs/jobs.aria.yml
  $FW exec $RUN --proof torrents/torrents-jobs -- $PW cli $D screenshot --filename={runDir}/proof/torrents/torrents-jobs/jobs.png
  $FW exec $RUN -- $PW cli $D click 'getByRole("button", { name: "The Quiet Orchard actions" })'
  $FW exec $RUN -- $PW cli $D click 'getByRole("menuitem", { name: "Open its torrent" })'
  $FW exec $RUN --proof torrents/torrents-jobs -- $PW cli $D run-code 'async page => { await page.getByRole("complementary", { name: "The Quiet Orchard" }).getByText(/^Seeding/).waitFor({ timeout: 15000 }); return page.url(); }'
  ```

- **Restart home.** Note The Quiet Orchard's totals, then restart home. Its torrents announce `stopped` as it stops. Within the 2 minute start spread The Quiet Orchard announces `started` again with `left` 0, which UNIT3D accepts from the new peer, and it reads `Seeding` with the same `downloaded`.

  ```sh
  $FW exec $RUN --proof torrents/torrents-restart -- curl -sS -H "$KEY" "$API/torrents/$(idOf The.Quiet)"
  $FW node restart $RUN home
  $FW exec $RUN --proof torrents/torrents-restart -- sh -c "$FW announces wait {runDir} --node home --where infoHash=$(curl -sS -H "$KEY" "$API/torrents/$(idOf The.Quiet)" | jq -r .infoHash) --where event=started --timeout 180s | jq -c '{at, event: .data.event, left: .data.left, accepted: .data.accepted}'"
  $FW exec $RUN --proof torrents/torrents-restart -- curl -sS -H "$KEY" "$API/torrents/$(idOf The.Quiet)"
  ```

- **Remove The Quiet Orchard with its download copy.** From its Inspector choose `Remove`. The dialog says `Files in your library stay.`, that The Quiet Orchard stops seeding, and offers `Also delete the download copy (frees <size>)`. Tick it and `Remove`: the tracker gets `stopped`, the row leaves, the save folder is gone and the library file still matches the release.

  ```sh
  SAVE=$(curl -sS -H "$KEY" "$API/torrents/$(idOf The.Quiet)" | jq -r .saveFolder)
  $FW fs stat $RUN "$SAVE"
  $FW exec $RUN -- $PW cli $D goto "{nodes.home.browserUrl}/activity/torrents/$(idOf The.Quiet)"
  $FW exec $RUN -- $PW cli $D click 'getByRole("complementary", { name: "The Quiet Orchard" }).getByRole("button", { name: "Remove", exact: true })'
  $FW exec $RUN --proof torrents/torrents-remove -- $PW cli $D snapshot 'getByRole("dialog", { name: "Remove The Quiet Orchard?" })' --filename={runDir}/proof/torrents/torrents-remove/dialog.aria.yml
  $FW exec $RUN -- $PW cli $D check 'getByRole("dialog", { name: "Remove The Quiet Orchard?" }).getByRole("checkbox", { name: /^Also delete the download copy/ })'
  $FW exec $RUN --proof torrents/torrents-remove -- $PW cli $D screenshot --filename={runDir}/proof/torrents/torrents-remove/dialog.png
  $FW exec $RUN -- $PW cli $D click 'getByRole("dialog", { name: "Remove The Quiet Orchard?" }).getByRole("button", { name: "Remove", exact: true })'
  $FW announces wait $RUN --node home --where event=stopped --timeout 30s
  $FW exec $RUN --proof torrents/torrents-remove -- $PW cli $D run-code 'async page => { await page.goto(page.url().replace(/\/activity\/torrents.*/, "/activity/torrents?tab=all")); await page.getByRole("table", { name: "Torrents" }).getByRole("row").filter({ hasText: "The Quiet Orchard" }).waitFor({ state: "detached", timeout: 30000 }); }'
  $FW fs stat $RUN "$SAVE"
  $FW exec $RUN --proof torrents/torrents-remove -- $FW fs match {runDir} '{mediaDir}/movies/The Quiet Orchard (2021)/The Quiet Orchard (2021) - 1080p.mkv' --release The.Quiet.Orchard.2021.1080p.WEB-DL.H264-FKW
  ```

## Gotchas

- UNIT3D rejects any `started` inside a window of 85 to 95% of the tracker's min interval, so a resume within 2 minutes of the last `started` waits. That is the behaviour to prove, not a hang. A `started` within 30 s of the last one waits too, on every tracker.
- The rate limit, not a fault rule, holds The Quiet Orchard at 16 KB/s here, so the live numbers are home's own. Save the limit back to 0 before waiting on an import, or it takes about 4 minutes.
- `Stalled after` and `Unregistered twice, this far apart` stay at the values this recipe saves for the rest of the run.
- The torrents Scenario has one Node, so the Node select never shows; the Node filter is in [download clients](./download-clients.md) on `home`. Only Movies exist in a release build, so `Content Type` narrows nothing it shows.
- `Pick <name>` checkboxes sit in each row's pick column; on a phone the card has no pick-all.
- `fs stat` of the removed save folder exits non-zero with `no such file`. That is the proof.
