# Torrents

Activity's Torrents view lists every torrent the downloading Nodes' built-in clients hold, live, problems first: a status sentence, the signal rows of what needs the Admin, tabs by state, filters, search and a table that drops from eight columns to four and then to cards as the window narrows. Selecting a row opens the Inspector beside the table on a desktop and as a pushed page on a phone: what the torrent is doing in a sentence, its numbers, announce URLs, files, Grabs and details. The Admin pauses, resumes, rechecks, marks failed or removes one torrent or a picked set, from the row menu, the Inspector, the picked toolbar or a Job's menu. A resume that a private tracker would reject waits and says when it starts. Removing asks first, says library files stay, lists what it cancels and can delete the download copy. Each Node runs a set number of downloads at once and queues the rest, and a restart resumes every torrent without losing its totals. A Grab of a magnet link gets the torrent's metadata from peers while it is fetching, and its Torrent appears only once the metadata arrives.

## Sub-features

- `torrents-queue` with `Downloads at once` 1, a second and third Grab read `Queued` with `1st in line` and `2nd in line`, and start once the limit is 3 again.
- `torrents-view` `/activity/torrents` leads with the sentence (`1 downloading · 1 failure · 1 warning`), then `Needs you` with a signal row per failure and warning (`Unregistered` Harbor Lights, `Stalled` Salt Meadow), the `Show` tabs `Downloading`, `Seeding`, `Problems` and `All` with counts, the search box and the table; each row leads with its Work's poster (a blank raised tile for Harbor Lights, which has none, and for a torrent with no Work) and names its Work with its quality, a state word over a line (`Downloading` over `1 peer`), the live bar with percent, time left and speed, ratio, speeds, the announce host over its status, and the Node over `Built-in` (P-164).
- `torrents-columns` the table shows Work, State, Progress, Ratio, Speed, Announce and Node · client at 1280 px, drops to Work, State and Progress at 980 px, and stacks each row as a card at 620 px and on a phone, each row led by its Work's 32 × 48 poster at every width.
- `torrents-live` the Admin's stream carries `torrents.live` each second while a torrent moves, and the row's percent and the Activity count follow without a reload (P-167).
- `torrents-filters` a tab, `?type=movie` (the `Content Type` select, shown once set) and a search narrow the table, and the URL keeps them (P-165).
- `torrents-inspector` a row opens the Inspector at `/activity/torrents/<id>` with `Downloading from 1 peer, <n> min left at 16 KB/s.`, `Pause`, `Remove` and `More actions for <Work>`, then Numbers, Announce URLs with the passkey hidden as `/announce/•••`, Files, Grabs and Details, whose save folder is `<downloads>/Movies/<info hash>` (P-146); ArrowDown steps to the next row with the Inspector following; on a phone it is a pushed page with the back link `Torrents`.
- `torrents-pause-resume` `Pause` reads `Paused` and the tracker gets `stopped`; `Resume` inside the tracker's window reads `Resumes at <HH:MM>`, then starts, and the UNIT3D tracker accepts every announce of it (P-166).
- `torrents-unregistered` a release the tracker calls unregistered reads `Announce error` over `Unregistered by 127.0.0.1` and is a failure in `Needs you`; once the tracker says so twice the unregistered gap apart, the Inspector adds `The release's tracker reports that the torrent is no longer registered.`
- `torrents-bulk` picking two rows shows the toolbar `Picked torrents` with `2 torrents picked`, and its `Pause` and `Resume` act on both and keep them picked; a paused torrent with an announce error still reads `Announce error`.
- `torrents-mark-failed` a download fed only corrupt pieces by a bad peer reads `Stalled` with `No data is arriving` after `Stalled after`; `Mark failed` asks `Mark Salt Meadow failed?`, removes it with its download copy and fails its Grab `download_failed`.
- `torrents-jobs` Jobs shows each Grab's quality and Target chips and its live percent, with time left and speed only while its torrent downloads (a paused one reads `<done> of <size>` as its speed winds down, as on the Work page's Grabs); its menu offers `Open its torrent`, which opens the Inspector (P-164).
- `torrents-disk-guard` on a `home` run while the disk guard holds seedbox (see [download-clients.md](./download-clients.md)), a new grab reads `Queued` over `Waiting for disk space on seedbox` with `hold` `disk_guard`, `Needs you` holds `Waiting for disk` with `New downloads on seedbox wait for disk space.`, and the sentence counts it as a warning; once the guard lifts the torrent downloads and the signal row goes. A download already running keeps running.
- `torrents-restart` restarting home stops each torrent with `stopped`, and the seeding torrent comes back `Seeding` with a `started` that reports nothing left and the same totals.
- `torrents-magnet-search` on a `magnets` run, a Grab of a magnet link reads `Fetching` with `Getting metadata from 1 peer` (or `Looking for peers`) over `Fails at <HH:MM> if no peer sends the metadata` in its Progress cell, on Jobs and on the Work page, and the API's Grab carries `metadata` with `peers` and `endsAt`.
- `torrents-magnet-torrent` a magnet link that names the tracker, and a .torrent URL that redirects to one, each import, and their Torrent appears on Torrents named for its file at the release's real size (`100% of 3.0 MB`), not the Indexer's advertised one.
- `torrents-magnet-no-metadata` with `A magnet link with no metadata fails after` at 2 min under Settings > Download Clients > `Magnet links`, Cedar Hollow's private magnet link fails 2 minutes after its grab: `Needs you` holds its card with `No peer sent the magnet link's metadata in time.`, `Search again to pick another release.` and the detail `No peer sent the metadata in 2 min, and 1 peer connected`, Torrents never lists it, and fake-dht never sees its info hash.
- `torrents-dht-switch` home's Torrent client lists `DHT for public torrents` under `Peers`, on by default. Off, a Grab of the trackerless Quiet Orchard fails at once with `no_peer_source` and the recovery `Turn on DHT for public torrents in that Node's torrent client settings, or search again.` Back on, `Search again` imports it through fake-dht alone: `lookups` holds home's `get_peers` for its hash and `announces` none from home.
- `torrents-remove` `Remove` asks `Remove The Quiet Orchard?`, says `Files in your library stay.` and lists the Work that stops seeding; with `Also delete the download copy` the download copy goes, the library file still matches the release, the tracker gets `stopped`, and the row leaves once home confirms (P-166).

## How to get to it (user POV)

- Desktop shell: the sidebar's `Activity`, then the page tab `Torrents` at `/activity/torrents`, or `?tab=all|seeding|problems`. A Node's `Torrent client` section links `Torrents` filtered to that Node (`?node=<id>&tab=all`).
- Phone shell: the `Activity` tab, then `Torrents` in the segmented control; the tabs list All first, the filters sit behind `Filter`, a row is a card, and the Inspector is a pushed page.
- Jobs: `/activity`, a Job's `<Work> actions` menu, `Open its torrent`.
- Magnet links: a Grab's Progress cell on Jobs or the Work page while it is fetching, its card under `Needs you` once it fails, the `DHT for public torrents` switch under `Peers` on a Node's `Torrent client` page, and `A magnet link with no metadata fails after` under `Magnet links` on Settings > Download Clients.
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

- For the magnet steps, a `magnets` run (`--scenario magnets`) instead: home is the Controller and the only Node, its Indexer lists most releases by magnet link, and every release is seeded. The Quiet Orchard's release is public and its magnet link names no tracker, so home finds its seeders only through fake-dht. Harbor Lights' magnet link names the tracker, Salt Meadow's .torrent URL redirects to its magnet link, Cedar Hollow's is a private magnet link whose seeders never send the metadata, and Juniper Falls' is a private .torrent. The same session and handles, plus:

  ```sh
  grab() { id=$(curl -sS -X POST -H "$KEY" -H 'Content-Type: application/json' -d "{\"contentType\":\"movie\",\"key\":\"$1\",\"monitor\":true}" "$API/works" | jq -r .work.id); curl -sS -X POST -H "$KEY" "$API/works/$id/search"; echo "$id"; }
  hashOf() { jq -r --arg n "$1" '.releases[] | select(.name | startswith($n)) | .infoHash' $RUN/manifest.json; }
  ```

Steps:

- **Slow home down and queue the downloads.** On home's Torrent client, save `Download limit` 16 with `KB/s` picked, then `Downloads at once` 1 and `Stalled after` 1 (min), each with Enter, then grab the three Works through the API. The Quiet Orchard downloads, and the others read `Queued` with `1st in line` and `2nd in line`.

  ```sh
  $FW exec $RUN -- $PW cli $D goto '{nodes.home.browserUrl}/nodes/{nodes.home.nodeId}/torrent-client'
  $FW exec $RUN -- $PW cli $D fill 'getByRole("textbox", { name: "Download limit" })' 16
  $FW exec $RUN -- $PW cli $D click 'getByRole("radiogroup", { name: "Download limit unit" }).getByRole("radio", { name: "KB/s" })'
  $FW exec $RUN -- $PW cli $D click 'getByRole("region", { name: "Speed" }).getByRole("button", { name: "Save" })'
  for field in 'Downloads at once:1' 'Stalled after:1'; do
    $FW exec $RUN -- $PW cli $D fill "getByRole(\"textbox\", { name: \"${field%:*}\" })" "${field##*:}"
    $FW exec $RUN -- $PW cli $D press Enter
  done
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
  $FW exec $RUN -- $PW cli $D press Enter
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

- **Narrow the window.** At 1280 px all seven named columns show; at 980 px only Work, State and Progress; at 620 px the header goes and each row is a card whose cells carry their labels. At every width The Quiet Orchard and Salt Meadow lead with their loaded posters and Harbor Lights, which has no poster, with a blank tile.

  ```sh
  for w in 1280 980 620; do
    $FW exec $RUN -- $PW cli $D resize $w 900
    $FW exec $RUN --proof torrents/torrents-columns -- $PW cli $D run-code 'async page => { const headers = page.getByRole("table", { name: "Torrents" }).getByRole("columnheader"); const shown = []; for (const h of await headers.all()) { if (await h.isVisible()) shown.push((await h.textContent()).trim()); } return shown; }'
    $FW exec $RUN --proof torrents/torrents-columns -- $PW cli $D run-code 'async page => page.getByRole("table", { name: "Torrents" }).locator("[data-torrent-id]").evaluateAll(rows => rows.map(row => { const img = row.querySelector("[data-poster] img"); return [row.querySelector("[data-col=work]").textContent, img === null ? "blank" : img.naturalWidth > 0 ? "poster" : "loading"]; }))'
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

- **Inspect a torrent.** On `All`, choose The Quiet Orchard's row. The Inspector leads with `Downloading from 1 peer, …`, lists the tracker as `127.0.0.1:<port>` with its URL under `Details` ending `/announce/•••`, and ArrowDown on the chosen row moves the selection and the URL to the next row, and ArrowUp back. Details' `Save folder` ends `downloads/Movies/<info hash>`. On the phone the row opens a pushed page with the back link `Torrents`.

  ```sh
  $FW exec $RUN -- $PW cli $D goto '{nodes.home.browserUrl}/activity/torrents?tab=all'
  $FW exec $RUN -- $PW cli $D click 'getByRole("table", { name: "Torrents" }).getByRole("button", { name: /^The Quiet Orchard(?! actions)/ })'
  $FW exec $RUN -- $PW cli $D press ArrowDown
  $FW exec $RUN --proof torrents/torrents-inspector -- $PW cli $D run-code 'async page => [page.url(), await page.getByRole("complementary").getByRole("heading", { level: 2 }).first().textContent()]'
  $FW exec $RUN -- $PW cli $D press ArrowUp
  $FW exec $RUN -- $PW cli $D click 'getByRole("complementary", { name: "The Quiet Orchard" }).getByRole("listitem").filter({ hasText: "127.0.0.1" }).getByRole("button", { name: "Details" })'
  $FW exec $RUN --proof torrents/torrents-inspector -- $PW cli $D snapshot 'getByRole("complementary", { name: "The Quiet Orchard" })' --filename={runDir}/proof/torrents/torrents-inspector/desktop.aria.yml
  $FW exec $RUN --proof torrents/torrents-inspector -- $PW cli $D screenshot --filename={runDir}/proof/torrents/torrents-inspector/desktop.png
  $FW exec $RUN --proof torrents/torrents-inspector -- curl -sS -H "$KEY" "$API/torrents/$(idOf The.Quiet)"
  $FW exec $RUN -- $PW cli $P goto '{nodes.home.browserUrl}/activity/torrents?tab=all'
  $FW exec $RUN -- $PW cli $P click 'getByRole("button", { name: /^The Quiet Orchard(?! actions)/ })'
  $FW exec $RUN --proof torrents/torrents-inspector -- $PW cli $P snapshot 'getByRole("banner")' --filename={runDir}/proof/torrents/torrents-inspector/phone-banner.aria.yml
  $FW exec $RUN --proof torrents/torrents-inspector -- $PW cli $P screenshot --full-page --filename={runDir}/proof/torrents/torrents-inspector/phone.png
  ```

- **Pause, resume, then resume inside the window.** From The Quiet Orchard's row menu choose `Pause`: the row reads `Paused` and the tracker gets `stopped`. `Resume` starts it again with a `started`, since its last one is older than the window. Half a minute after that `stopped`, `Pause` again, then `Resume` at once: the row reads `Resumes at <HH:MM>`, 2 minutes after the `started`, with no time left. When the window passes it starts, and the tracker accepted every announce of it.

  ```sh
  pick() { $FW exec $RUN -- $PW cli $D click 'getByRole("button", { name: "The Quiet Orchard actions" })'; $FW exec $RUN -- $PW cli $D click "getByRole(\"menuitem\", { name: \"$1\" })"; }
  IH=$(curl -sS -H "$KEY" "$API/torrents/$(idOf The.Quiet)" | jq -r .infoHash)
  $FW exec $RUN -- $PW cli $D goto '{nodes.home.browserUrl}/activity/torrents?tab=all'
  pick Pause
  $FW announces wait $RUN --node home --where infoHash=$IH --where event=stopped --timeout 30s
  $FW exec $RUN --proof torrents/torrents-pause-resume -- $PW cli $D run-code 'async page => { await page.getByRole("table", { name: "Torrents" }).getByRole("row").filter({ hasText: "The Quiet Orchard" }).getByRole("cell", { name: /^Paused/ }).waitFor({ timeout: 15000 }); }'
  pick Resume
  $FW announces wait $RUN --node home --where infoHash=$IH --where event=started --timeout 30s
  sleep 30
  pick Pause
  $FW announces wait $RUN --node home --where infoHash=$IH --where event=stopped --timeout 30s
  pick Resume
  $FW exec $RUN --proof torrents/torrents-pause-resume -- $PW cli $D run-code 'async page => { const line = page.getByRole("table", { name: "Torrents" }).getByText(/^Resumes at/); await line.waitFor({ timeout: 15000 }); return line.textContent(); }'
  $FW exec $RUN --proof torrents/torrents-pause-resume -- $PW cli $D screenshot --filename={runDir}/proof/torrents/torrents-pause-resume/window.png
  $FW announces wait $RUN --node home --where infoHash=$IH --where event=started --timeout 180s
  $FW exec $RUN --proof torrents/torrents-pause-resume -- sh -c "$FW announces list {runDir} --node home --where infoHash=$IH | jq -c '{at, event: .data.event, accepted: .data.accepted}'"
  ```

- **Read the unregistered release.** Shorten `Unregistered twice, this far apart` to 1 min on Settings > Download Clients. Harbor Lights' row reads `Announce error` over `Unregistered by 127.0.0.1`; after the tracker's next refusal, 3 minutes after the first, its Inspector adds the release fault.

  ```sh
  $FW exec $RUN -- $PW cli $D goto '{nodes.home.browserUrl}/settings/download-clients'
  $FW exec $RUN -- $PW cli $D fill 'getByRole("textbox", { name: "Unregistered twice, this far apart" })' 1
  $FW exec $RUN -- $PW cli $D click 'getByRole("main").getByRole("button", { name: "Save" })'
  $FW exec $RUN -- $PW cli $D goto '{nodes.home.browserUrl}/activity/torrents?tab=all'
  $FW exec $RUN -- $PW cli $D click 'getByRole("table", { name: "Torrents" }).getByRole("button", { name: /^Harbor Lights(?! actions)/ })'
  $FW exec $RUN --proof torrents/torrents-unregistered -- $PW cli $D run-code 'async page => { await page.getByText("The release's tracker reports that the torrent is no longer registered.").waitFor({ timeout: 300000 }); }'
  $FW exec $RUN --proof torrents/torrents-unregistered -- $PW cli $D snapshot 'getByRole("complementary", { name: "Harbor Lights" })' --filename={runDir}/proof/torrents/torrents-unregistered/inspector.aria.yml
  $FW exec $RUN --proof torrents/torrents-unregistered -- $PW cli $D screenshot --filename={runDir}/proof/torrents/torrents-unregistered/inspector.png
  $FW exec $RUN --proof torrents/torrents-unregistered -- curl -sS -H "$KEY" "$API/torrents/$(idOf Harbor)"
  ```

- **Pause two at once.** Pick Harbor Lights and Salt Meadow: the toolbar `Picked torrents` reads `2 torrents picked`. Its `Pause` pauses both and keeps them picked: Salt Meadow reads `Paused`, while Harbor Lights still reads `Announce error`, which outranks `Paused`, and the API's `hold` reads `wanted` for both. The toolbar's `Resume` then resumes both: `hold` clears, or reads `window` with `Resumes at` on a torrent whose last `started` is inside its tracker's window.

  ```sh
  $FW exec $RUN -- $PW cli $D goto '{nodes.home.browserUrl}/activity/torrents?tab=all'
  $FW exec $RUN -- $PW cli $D click 'getByRole("checkbox", { name: "Pick Harbor Lights" })'
  $FW exec $RUN -- $PW cli $D click 'getByRole("checkbox", { name: "Pick Salt Meadow" })'
  $FW exec $RUN --proof torrents/torrents-bulk -- $PW cli $D snapshot 'getByRole("toolbar", { name: "Picked torrents" })' --filename={runDir}/proof/torrents/torrents-bulk/picked.aria.yml
  $FW exec $RUN -- $PW cli $D click 'getByRole("toolbar", { name: "Picked torrents" }).getByRole("button", { name: "Pause" })'
  $FW exec $RUN --proof torrents/torrents-bulk -- $PW cli $D run-code 'async page => { await page.getByRole("table", { name: "Torrents" }).getByRole("row").filter({ hasText: "Salt Meadow" }).getByRole("cell", { name: /^Paused/ }).waitFor({ timeout: 15000 }); }'
  $FW exec $RUN --proof torrents/torrents-bulk -- sh -c "curl -sS -H 'X-Api-Key: {apiKey}' '{nodes.home.url}/api/v1/torrents' | jq -c '.rows[] | {name, state, hold}'"
  $FW exec $RUN --proof torrents/torrents-bulk -- $PW cli $D screenshot --filename={runDir}/proof/torrents/torrents-bulk/paused.png
  $FW exec $RUN -- $PW cli $D click 'getByRole("toolbar", { name: "Picked torrents" }).getByRole("button", { name: "Resume" })'
  $FW exec $RUN --proof torrents/torrents-bulk -- $PW cli $D run-code 'async page => { await page.getByRole("table", { name: "Torrents" }).getByRole("row").filter({ hasText: "Salt Meadow" }).getByRole("cell", { name: /^(Downloading|Stalled|Queued|Paused Resumes at)/ }).waitFor({ timeout: 15000 }); }'
  $FW exec $RUN --proof torrents/torrents-bulk -- sh -c "curl -sS -H 'X-Api-Key: {apiKey}' '{nodes.home.url}/api/v1/torrents' | jq -c '.rows[] | {name, state, hold}'"
  $FW exec $RUN -- $PW cli $D click 'getByRole("toolbar", { name: "Picked torrents" }).getByRole("button", { name: "Clear" })'
  ```

- **Mark the bad release failed.** Once Salt Meadow reads `Stalled` again, a minute after it starts (which can wait out its tracker's window first), on the phone open its card's menu and `Mark failed`; the dialog `Mark Salt Meadow failed?` lists its Grab. Confirm: the row leaves, the tracker gets `stopped`, and its Grab fails `download_failed`.

  ```sh
  $FW exec $RUN -- $PW cli $P goto '{nodes.home.browserUrl}/activity/torrents?tab=all'
  $FW exec $RUN -- $PW cli $P run-code 'async page => { await page.getByRole("table", { name: "Torrents" }).getByRole("row").filter({ hasText: "Salt Meadow" }).getByText("No data is arriving").waitFor({ timeout: 240000 }); }'
  $FW exec $RUN -- $PW cli $P click 'getByRole("button", { name: "Salt Meadow actions" })'
  $FW exec $RUN -- $PW cli $P click 'getByRole("dialog", { name: "Salt Meadow actions" }).getByRole("button", { name: "Mark failed" })'
  $FW exec $RUN --proof torrents/torrents-mark-failed -- $PW cli $P snapshot 'getByRole("dialog", { name: "Mark Salt Meadow failed?" })' --filename={runDir}/proof/torrents/torrents-mark-failed/dialog.aria.yml
  $FW exec $RUN --proof torrents/torrents-mark-failed -- $PW cli $P screenshot --filename={runDir}/proof/torrents/torrents-mark-failed/dialog-phone.png
  $FW exec $RUN -- $PW cli $P click 'getByRole("dialog", { name: "Mark Salt Meadow failed?" }).getByRole("button", { name: "Mark failed" })'
  $FW announces wait $RUN --node home --where event=stopped --timeout 30s
  $FW exec $RUN --proof torrents/torrents-mark-failed -- $PW cli $P run-code 'async page => { await page.getByRole("table", { name: "Torrents" }).getByRole("row").filter({ hasText: "Salt Meadow" }).waitFor({ state: "detached", timeout: 30000 }); }'
  $FW exec $RUN --proof torrents/torrents-mark-failed -- curl -sS -H 'X-Api-Key: {apiKey}' '{nodes.home.url}/api/v1/grabs'
  ```

- **Finish The Quiet Orchard and read Jobs.** Save `Download limit` 0. The Quiet Orchard completes, imports and reads `Seeding` on `Seeding`. Jobs shows its quality `WEBDL-1080p` and Target `1080p` chips, and its menu's `Open its torrent` opens its Inspector.

  ```sh
  $FW exec $RUN -- $PW cli $D goto '{nodes.home.browserUrl}/nodes/{nodes.home.nodeId}/torrent-client'
  $FW exec $RUN -- $PW cli $D fill 'getByRole("textbox", { name: "Download limit" })' 0
  $FW exec $RUN -- $PW cli $D press Enter
  $FW exec $RUN -- $PW cli $D goto '{nodes.home.browserUrl}/activity'
  $FW exec $RUN --proof torrents/torrents-jobs -- $PW cli $D run-code 'async page => { await page.getByRole("table", { name: "Grabs" }).getByRole("row").filter({ hasText: "The Quiet Orchard" }).getByText("Imported").waitFor({ timeout: 120000 }); }'
  $FW exec $RUN --proof torrents/torrents-jobs -- $PW cli $D snapshot 'getByRole("table", { name: "Grabs" })' --filename={runDir}/proof/torrents/torrents-jobs/jobs.aria.yml
  $FW exec $RUN --proof torrents/torrents-jobs -- $PW cli $D screenshot --filename={runDir}/proof/torrents/torrents-jobs/jobs.png
  $FW exec $RUN -- $PW cli $D click 'getByRole("button", { name: "The Quiet Orchard actions" })'
  $FW exec $RUN -- $PW cli $D click 'getByRole("menuitem", { name: "Open its torrent" })'
  $FW exec $RUN --proof torrents/torrents-jobs -- $PW cli $D run-code 'async page => { await page.getByRole("complementary", { name: "The Quiet Orchard" }).getByText(/^Seeding/).waitFor({ timeout: 15000 }); return page.url(); }'
  ```

- **Watch the disk guard hold a grab.** This runs on a `home` run, in the steps of [download-clients.md](./download-clients.md)'s **Hold new downloads on low disk**: between the grab and removing the levels, read Torrents.

  ```sh
  $FW exec $RUN -- $PW cli $D goto '{nodes.home.browserUrl}/activity/torrents'
  $FW exec $RUN --proof torrents/torrents-disk-guard -- $PW cli $D run-code 'async page => { const s = page.getByText("New downloads on seedbox wait for disk space.").first(); await s.waitFor({ timeout: 30000 }); return s.textContent(); }'
  $FW exec $RUN --proof torrents/torrents-disk-guard -- $PW cli $D snapshot 'getByRole("main")' --filename={runDir}/proof/torrents/torrents-disk-guard/held.aria.yml
  $FW exec $RUN --proof torrents/torrents-disk-guard -- $PW cli $D screenshot --filename={runDir}/proof/torrents/torrents-disk-guard/held.png
  $FW exec $RUN --proof torrents/torrents-disk-guard -- sh -c "curl -sS -H 'X-Api-Key: {apiKey}' '{nodes.home.url}/api/v1/torrents?tab=all' | jq -c '.rows[] | {name, state, hold}'"
  ```

- **Restart home.** Note The Quiet Orchard's totals, then restart home. Its torrents announce `stopped` as it stops. Within the 2 minute start spread The Quiet Orchard announces `started` again with `left` 0, which UNIT3D accepts from the new peer, and it reads `Seeding` with the same `downloaded`.

  ```sh
  $FW exec $RUN --proof torrents/torrents-restart -- curl -sS -H "$KEY" "$API/torrents/$(idOf The.Quiet)"
  $FW node restart $RUN home
  $FW exec $RUN --proof torrents/torrents-restart -- sh -c "$FW announces wait {runDir} --node home --where infoHash=$(curl -sS -H "$KEY" "$API/torrents/$(idOf The.Quiet)" | jq -r .infoHash) --where event=started --timeout 180s | jq -c '{at, event: .data.event, left: .data.left, accepted: .data.accepted}'"
  $FW exec $RUN --proof torrents/torrents-restart -- curl -sS -H "$KEY" "$API/torrents/$(idOf The.Quiet)"
  ```

- **Remove The Quiet Orchard with its download copy.** From its Inspector choose `Remove`. The dialog says `Files in your library stay.`, that The Quiet Orchard stops seeding, and offers `Also delete the download copy (frees <size>)`. The download copy and the library file are one file under two links, so it frees 0 bytes and says the library shares the files. Tick it and `Remove`: the tracker gets `stopped`, the row leaves, the download copy is gone, and the library file still matches the release, now with one link.

  ```sh
  COPY="$(curl -sS -H "$KEY" "$API/torrents/$(idOf The.Quiet)" | jq -r .saveFolder)/The.Quiet.Orchard.2021.1080p.WEB-DL.H264-FKW.mkv"
  $FW fs stat $RUN "$COPY"
  $FW exec $RUN -- $PW cli $D goto "{nodes.home.browserUrl}/activity/torrents/$(idOf The.Quiet)"
  $FW exec $RUN -- $PW cli $D click 'getByRole("complementary", { name: "The Quiet Orchard" }).getByRole("button", { name: "Remove", exact: true })'
  $FW exec $RUN --proof torrents/torrents-remove -- $PW cli $D snapshot 'getByRole("dialog", { name: "Remove The Quiet Orchard?" })' --filename={runDir}/proof/torrents/torrents-remove/dialog.aria.yml
  $FW exec $RUN -- $PW cli $D check 'getByRole("dialog", { name: "Remove The Quiet Orchard?" }).getByRole("checkbox", { name: /^Also delete the download copy/ })'
  $FW exec $RUN --proof torrents/torrents-remove -- $PW cli $D screenshot --filename={runDir}/proof/torrents/torrents-remove/dialog.png
  $FW exec $RUN -- $PW cli $D click 'getByRole("dialog", { name: "Remove The Quiet Orchard?" }).getByRole("button", { name: "Remove", exact: true })'
  $FW announces wait $RUN --node home --where event=stopped --timeout 30s
  $FW exec $RUN --proof torrents/torrents-remove -- $PW cli $D run-code 'async page => { await page.goto(page.url().replace(/\/activity\/torrents.*/, "/activity/torrents?tab=all")); await page.getByRole("table", { name: "Torrents" }).getByRole("row").filter({ hasText: "The Quiet Orchard" }).waitFor({ state: "detached", timeout: 30000 }); }'
  $FW fs stat $RUN "$COPY"
  $FW exec $RUN --proof torrents/torrents-remove -- $FW fs match {runDir} '{mediaDir}/movies/The Quiet Orchard (2021)/The Quiet Orchard (2021) - 1080p.mkv' --release The.Quiet.Orchard.2021.1080p.WEB-DL.H264-FKW
  ```

- **Turn DHT off and grab the trackerless release.** On home's Torrent client, `DHT for public torrents` under `Peers` reads on. Turn it off, then grab The Quiet Orchard: its Grab fails at once, with no metadata search, and `Needs you` holds its card with `The magnet link names no tracker, and DHT is off on the Node that downloads it.`

  ```sh
  $FW exec $RUN -- $PW cli $D goto '{nodes.home.browserUrl}/nodes/{nodes.home.nodeId}/torrent-client'
  $FW exec $RUN --proof torrents/torrents-dht-switch -- $PW cli $D snapshot 'getByRole("region", { name: "Peers" })' --filename={runDir}/proof/torrents/torrents-dht-switch/on.aria.yml
  $FW exec $RUN -- $PW cli $D click 'getByRole("switch", { name: "DHT for public torrents" })'
  $FW exec $RUN --proof torrents/torrents-dht-switch -- $PW cli $D screenshot --filename={runDir}/proof/torrents/torrents-dht-switch/off.png
  grab 9000001
  $FW exec $RUN -- $PW cli $D goto '{nodes.home.browserUrl}/activity'
  $FW exec $RUN --proof torrents/torrents-dht-switch -- $PW cli $D run-code 'async page => { await page.getByText("Turn on DHT for public torrents").waitFor({ timeout: 15000 }); }'
  $FW exec $RUN --proof torrents/torrents-dht-switch -- $PW cli $D screenshot --filename={runDir}/proof/torrents/torrents-dht-switch/no-peer-source.png
  $FW exec $RUN --proof torrents/torrents-dht-switch -- sh -c "curl -sS -H 'X-Api-Key: {apiKey}' '{nodes.home.url}/api/v1/grabs' | jq -c '.grabs[] | {releaseTitle, stage, failure}'"
  ```

- **Turn DHT on and search again.** Turn the switch back on, then choose the card's `Search again`. The Grab imports through fake-dht: `lookups` holds home's `get_peers` for The Quiet Orchard's hash, the tracker saw no announce of it from home, and its Torrent reads `No announce URL`.

  ```sh
  $FW exec $RUN -- $PW cli $D goto '{nodes.home.browserUrl}/nodes/{nodes.home.nodeId}/torrent-client'
  $FW exec $RUN -- $PW cli $D click 'getByRole("switch", { name: "DHT for public torrents" })'
  $FW exec $RUN -- $PW cli $D goto '{nodes.home.browserUrl}/activity'
  $FW exec $RUN -- $PW cli $D click 'getByRole("main").getByRole("button", { name: "Search again" })'
  $FW exec $RUN --proof torrents/torrents-dht-switch -- $PW cli $D run-code 'async page => { await page.getByRole("table", { name: "Grabs" }).getByRole("row").filter({ hasText: "The Quiet Orchard" }).getByText("Imported").waitFor({ timeout: 60000 }); }'
  $FW exec $RUN --proof torrents/torrents-dht-switch -- $FW lookups list {runDir} --node home --where method=get_peers --where infoHash=$(hashOf The.Quiet)
  $FW exec $RUN --proof torrents/torrents-dht-switch -- $FW announces list {runDir} --node home --where infoHash=$(hashOf The.Quiet)
  ```

- **Shorten the metadata wait and watch a search.** On Settings > Download Clients, save `A magnet link with no metadata fails after` 2 (min) under `Magnet links`, then grab Cedar Hollow. Its Jobs row reads `Fetching` with `Getting metadata from 1 peer` over `Fails at <HH:MM> if no peer sends the metadata`, 2 minutes after the grab, and the Work page's Grab reads the same.

  ```sh
  $FW exec $RUN -- $PW cli $D goto '{nodes.home.browserUrl}/settings/download-clients'
  $FW exec $RUN -- $PW cli $D fill 'getByRole("textbox", { name: "A magnet link with no metadata fails after" })' 2
  $FW exec $RUN -- $PW cli $D press Enter
  $FW exec $RUN --proof torrents/torrents-magnet-no-metadata -- $PW cli $D screenshot --filename={runDir}/proof/torrents/torrents-magnet-no-metadata/wait.png
  CEDAR=$(grab 9000004 | tail -1)
  $FW exec $RUN -- $PW cli $D goto '{nodes.home.browserUrl}/activity'
  $FW exec $RUN --proof torrents/torrents-magnet-search -- $PW cli $D run-code 'async page => { const row = page.getByRole("table", { name: "Grabs" }).getByRole("row").filter({ hasText: "Cedar Hollow" }); await row.getByText(/^Getting metadata from|^Looking for peers/).waitFor({ timeout: 30000 }); return row.textContent(); }'
  $FW exec $RUN --proof torrents/torrents-magnet-search -- $PW cli $D screenshot --filename={runDir}/proof/torrents/torrents-magnet-search/jobs.png
  $FW exec $RUN --proof torrents/torrents-magnet-search -- sh -c "curl -sS -H 'X-Api-Key: {apiKey}' '{nodes.home.url}/api/v1/grabs' | jq -c '.grabs[] | select(.releaseTitle | startswith(\"Cedar\")) | {stage, metadata}'"
  $FW exec $RUN -- $PW cli $D goto "{nodes.home.browserUrl}/works/$CEDAR"
  $FW exec $RUN --proof torrents/torrents-magnet-search -- $PW cli $D screenshot --filename={runDir}/proof/torrents/torrents-magnet-search/work.png
  ```

- **Read the no_metadata card.** 2 minutes after the grab, Cedar Hollow's Grab fails at fetching, and `Needs you` holds its card. Torrents never lists it, and fake-dht never saw its hash.

  ```sh
  $FW exec $RUN -- $PW cli $D goto '{nodes.home.browserUrl}/activity'
  $FW exec $RUN --proof torrents/torrents-magnet-no-metadata -- $PW cli $D run-code 'async page => { await page.getByText(/^No peer sent the magnet link.s metadata in time\.$/).waitFor({ timeout: 180000 }); }'
  $FW exec $RUN --proof torrents/torrents-magnet-no-metadata -- $PW cli $D snapshot 'getByRole("main")' --filename={runDir}/proof/torrents/torrents-magnet-no-metadata/card.aria.yml
  $FW exec $RUN --proof torrents/torrents-magnet-no-metadata -- $PW cli $D screenshot --filename={runDir}/proof/torrents/torrents-magnet-no-metadata/card.png
  $FW exec $RUN --proof torrents/torrents-magnet-no-metadata -- sh -c "curl -sS -H 'X-Api-Key: {apiKey}' '{nodes.home.url}/api/v1/grabs' | jq -c '.grabs[] | select(.releaseTitle | startswith(\"Cedar\")) | {stage, failure, history}'"
  $FW exec $RUN --proof torrents/torrents-magnet-no-metadata -- $FW lookups list {runDir} --where infoHash=$(hashOf Cedar)
  ```

- **Grab the magnet links that name a tracker.** Grab Harbor Lights, Salt Meadow and Juniper Falls. All three import, and Torrents lists each at 3.0 MB, the payload's size, while the Indexer advertised about 7 GB. Harbor Lights' Inspector lists the tracker and `Private` `No`, and fake-dht never saw Juniper Falls' private hash.

  ```sh
  for key in 9000002 9000003 9000005; do grab $key; done
  $FW exec $RUN -- $PW cli $D goto '{nodes.home.browserUrl}/activity/torrents?tab=all'
  $FW exec $RUN --proof torrents/torrents-magnet-torrent -- $PW cli $D run-code 'async page => { await page.getByRole("table", { name: "Torrents" }).getByRole("row").filter({ hasText: "Salt Meadow" }).waitFor({ timeout: 60000 }); }'
  $FW exec $RUN --proof torrents/torrents-magnet-torrent -- $PW cli $D screenshot --filename={runDir}/proof/torrents/torrents-magnet-torrent/torrents.png
  $FW exec $RUN -- $PW cli $D click 'getByRole("table", { name: "Torrents" }).getByRole("button", { name: /^Harbor Lights(?! actions)/ })'
  $FW exec $RUN --proof torrents/torrents-magnet-torrent -- $PW cli $D snapshot 'getByRole("complementary", { name: "Harbor Lights" })' --filename={runDir}/proof/torrents/torrents-magnet-torrent/inspector.aria.yml
  $FW exec $RUN --proof torrents/torrents-magnet-torrent -- sh -c "curl -sS -H 'X-Api-Key: {apiKey}' '{nodes.home.url}/api/v1/torrents?tab=all' | jq -c '.rows[] | {name, size, state}'"
  $FW exec $RUN --proof torrents/torrents-magnet-torrent -- $FW fs match {runDir} '{mediaDir}/movies/Salt Meadow (2020)/Salt Meadow (2020) - 1080p.mkv' --release Salt.Meadow.2020.1080p.WEB-DL.H264-FKW
  $FW exec $RUN --proof torrents/torrents-magnet-torrent -- $FW lookups list {runDir} --where infoHash=$(hashOf Juniper)
  ```

## Gotchas

- Each setting on the Torrent client page saves on its own, with the Save that appears beside it once changed or with Enter in its field. `Download limit` shows its unit only once it holds a number, and starts at MB/s; pick `KB/s` before saving.
- `announces wait` matches only announces after the last step's start, so wait right after the click that causes one, before any other step, and name the torrent's `infoHash`: the tracker refuses Harbor Lights again every few minutes.
- UNIT3D rejects any `started` inside a window of 85 to 95% of the tracker's min interval, so a resume within 2 minutes of the last `started` waits. That is the behaviour to prove, not a hang. A `started` within 30 s of the last one waits too, on every tracker.
- The rate limit, not a fault rule, holds The Quiet Orchard at 16 KB/s here, so the live numbers are home's own. Save the limit back to 0 before waiting on an import, or it takes about 4 minutes.
- `Stalled after` and `Unregistered twice, this far apart` stay at the values this recipe saves for the rest of the run.
- The torrents Scenario has one Node, so the Node select never shows; the Node filter is in [download clients](./download-clients.md) on `home`. Only Movies exist in a release build, so `Content Type` narrows nothing it shows.
- `Pick <name>` checkboxes sit in each row's pick column; on a phone the card has no pick-all.
- `fs stat` reads only regular files. Before the remove the download copy shows `links` 2; after it, `fs stat` exits non-zero with `no such file`, which is the proof.
- A public magnet link's metadata arrives within a second or two, so its search is seldom on screen. Cedar Hollow's private magnet link is the one to watch, since its seeders never send the metadata.
- A Grab that failed `no_peer_source` searches again from the card's `Search again`, which picks the same release once DHT is back on.
- `lookups` also lists libtorrent's own DHT lookups when the DHT starts, `get_peers` for hashes no release has. Match on `infoHash` to read a torrent's.
- A change to the metadata wait moves the deadline of a search already running, which still counts from when that search started. Save the wait before the grab, as the recipe does, so the `Fails at` time is the one to expect.
