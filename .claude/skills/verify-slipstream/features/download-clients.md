# Download Clients

Settings > Download Clients lists a card for the built-in client of each Node that downloads, and the thresholds that make a release count as bad. A card leads with the client's listen-port check: listening on its address, and whether a peer from the internet has reached the port or it is still unproven. It shows the torrents and totals the client holds, its limits, its peer ID prefix and user agent under Details, and no Remove button, since it goes away only when the Node stops downloading. While a Node's downloads volume is below Health's error level for it, the disk guard holds that Node's new downloads: its card leads with `Waiting for disk`, the section carries the Health item, and the guard lifts on the first sample above the level or once the level moves below it. Each card links to its Node's Torrent client section, where the per-Node limits save one group at a time and apply live, and from there to Torrents filtered to that Node.

## Sub-features

- `download-clients-cards` `/settings/download-clients` lists `Built-in client on seedbox` reading `Listening on <address>:<port>. No peer from the internet has connected to port <port> yet, so it may not be reachable.` with the router hint, `0 torrents, 0 bytes uploaded and 0 bytes downloaded.`, `This card is removed once this Node stops downloading.` and no Remove; its Details list the limits, the peer ID prefix `-LT20F0-` and the user agent `libtorrent/2.0.15.0`.
- `download-clients-reachable` once the Scenario's leecher connects in after seedbox's first announce, the card reads `Peers from the internet have reached port <port>.` and the API's `reach` reads `reachable`.
- `download-clients-thresholds` `When a release counts as bad` saves `A download with no data at all is bad after` and `Unregistered twice, this far apart` one at a time with the toast `Download Clients settings saved`, and the API refuses a value outside its bounds (422 `setting_out_of_range`).
- `download-clients-disk-guard` with seedbox's downloads volume (`/` in a process world) given its own error level above its free space, the API's card carries `diskGuard` `{sentence, hint, errorBelow}` and the downloads volume as `downloads` `{mount, name, total, free, sampledAt}`, its `sentence` and `hint` the words Health's item says, the section's tab reads `Download Clients 1 warning` and opens with the Health line `Warning New downloads on seedbox …` and `Open Health`, and seedbox's card leads with the chip `Waiting for disk`, `New downloads on seedbox wait for disk space.` and `<volume> has <free> left, below its error level of <bytes>. Free up space there, or move the downloads folder to a volume with more room. Downloads already running continue.`, as does seedbox's Torrent client page. A grab then queues with `hold` `disk_guard` (see [torrents.md](./torrents.md)). Removing the volume's own levels lifts the guard at once: `diskGuard` goes, the torrent starts, and each change carries `download-clients.changed`. With a level in bytes just above the free space, freeing space lifts it at seedbox's next sample, at Storage's `Check storage every` cadence or a Storage Check now.
- `download-clients-limits` the card's `Torrent client settings on seedbox` opens seedbox's Torrent client; a `Download limit` of 16 saved there applies live, holding a running download near 16 KB/s, and the section's `Torrents` link opens Torrents on `All` filtered to seedbox with the `Node` select reading `seedbox`.

## How to get to it (user POV)

- Desktop shell: the sidebar's `Settings`, then `Download Clients` in the section nav, `/settings/download-clients`. Phone shell: `More`, then `Settings`, then `Download Clients`, a pushed page.
- A card's `Torrent client settings on <Node>` link, or Nodes, a Node, then its `Torrent client` page tab (only on a Node that stands in Downloading); the Nodes Inspector's Downloading chip links `Torrent client` and `Torrents`.
- API: `GET /api/v1/download-clients` (cards and thresholds), `PATCH /api/v1/download-clients/thresholds`, `GET` and `PATCH /api/v1/download-clients/{nodeId}/settings`, and the event `download-clients.changed`.

## Driving it with fakeworld and playwright cli

Preconditions:

- A fresh `home` run with doctor passing: seedbox holds Downloading and the Scenario's leecher connects to it once it announces the 1080p release. The `{runId}-admin-desktop` and `{runId}-admin-phone` sessions open and signed in, and these handles:

  ```sh
  D=-s={runId}-admin-desktop
  P=-s={runId}-admin-phone
  ```

Steps:

- **Read seedbox's card.** Open Settings > Download Clients. seedbox's card reads unproven, since nothing has connected in yet, and its Details hold the limits, the peer ID prefix and the user agent. It has no Remove button. On the phone it is a pushed page under Settings.

  ```sh
  $FW exec $RUN -- $PW cli $D goto '{nodes.home.browserUrl}/settings/download-clients'
  $FW exec $RUN -- $PW cli $D click 'getByRole("article", { name: "Built-in client on seedbox" }).getByRole("button", { name: "Details" })'
  $FW exec $RUN --proof download-clients/download-clients-cards -- $PW cli $D snapshot 'getByRole("main")' --filename={runDir}/proof/download-clients/download-clients-cards/desktop.aria.yml
  $FW exec $RUN --proof download-clients/download-clients-cards -- $PW cli $D screenshot --full-page --filename={runDir}/proof/download-clients/download-clients-cards/desktop.png
  $FW exec $RUN --proof download-clients/download-clients-cards -- curl -sS -H 'X-Api-Key: {apiKey}' '{nodes.home.url}/api/v1/download-clients'
  $FW exec $RUN -- $PW cli $P goto '{nodes.home.browserUrl}/settings/download-clients'
  $FW exec $RUN --proof download-clients/download-clients-cards -- $PW cli $P screenshot --full-page --filename={runDir}/proof/download-clients/download-clients-cards/phone.png
  ```

- **Save a threshold and refuse one.** Set `A download with no data at all is bad after` to 2 (h) and save: the toast reads `Download Clients settings saved`, and the API's `faultStall` reads 7200. Over the API, 60 s answers 422.

  ```sh
  $FW exec $RUN -- $PW cli $D fill 'getByRole("textbox", { name: "A download with no data at all is bad after" })' 2
  $FW exec $RUN -- $PW cli $D click 'getByRole("main").getByRole("button", { name: "Save" })'
  $FW exec $RUN --proof download-clients/download-clients-thresholds -- $PW cli $D run-code 'async page => { await page.getByRole("region", { name: "Notifications" }).getByText("Download Clients settings saved").waitFor({ timeout: 10000 }); }'
  $FW exec $RUN --proof download-clients/download-clients-thresholds -- curl -sS -H 'X-Api-Key: {apiKey}' '{nodes.home.url}/api/v1/download-clients'
  $FW exec $RUN --proof download-clients/download-clients-thresholds -- curl -sS -i -X PATCH -H 'X-Api-Key: {apiKey}' -H 'Content-Type: application/json' -d '{"faultStall":60}' '{nodes.home.url}/api/v1/download-clients/thresholds'
  ```

- **Limit seedbox's download speed.** From the card, open `Torrent client settings on seedbox`, save `Download limit` 16 with `KB/s` picked, then search The Quiet Orchard through the API. On Torrents its row's speed stays near `16 KB/s`.

  ```sh
  $FW exec $RUN -- $PW cli $D click 'getByRole("link", { name: "Torrent client settings on seedbox" })'
  $FW exec $RUN -- $PW cli $D fill 'getByRole("textbox", { name: "Download limit" })' 16
  $FW exec $RUN -- $PW cli $D click 'getByRole("radiogroup", { name: "Download limit unit" }).getByRole("radio", { name: "KB/s" })'
  $FW exec $RUN -- $PW cli $D click 'getByRole("region", { name: "Speed" }).getByRole("button", { name: "Save" })'
  $FW exec $RUN --proof download-clients/download-clients-limits -- $PW cli $D run-code 'async page => { await page.getByRole("region", { name: "Notifications" }).getByText("Torrent client settings saved").waitFor({ timeout: 10000 }); return page.url(); }'
  $FW exec $RUN --proof download-clients/download-clients-limits -- sh -c "id=\$(curl -sS -X POST -H 'X-Api-Key: {apiKey}' -H 'Content-Type: application/json' -d '{\"contentType\":\"movie\",\"key\":\"9000001\",\"monitor\":true}' '{nodes.home.url}/api/v1/works' | jq -r .work.id); curl -sS -X POST -H 'X-Api-Key: {apiKey}' \"{nodes.home.url}/api/v1/works/\$id/search\""
  $FW exec $RUN -- $PW cli $D click 'getByRole("main").getByRole("link", { name: "Torrents" })'
  $FW exec $RUN --proof download-clients/download-clients-limits -- $PW cli $D run-code 'async page => { const speed = page.getByRole("table", { name: "Torrents" }).getByRole("row").filter({ hasText: "The Quiet Orchard" }).getByRole("cell", { name: /^Down \d/ }); await speed.waitFor({ timeout: 30000 }); await page.waitForTimeout(5000); return [page.url(), await speed.textContent()]; }'
  $FW exec $RUN --proof download-clients/download-clients-limits -- $PW cli $D snapshot 'getByRole("main")' --filename={runDir}/proof/download-clients/download-clients-limits/torrents.aria.yml
  $FW exec $RUN --proof download-clients/download-clients-limits -- $PW cli $D screenshot --filename={runDir}/proof/download-clients/download-clients-limits/torrents.png
  ```

- **See the port proven.** The leecher says hello once seedbox announces the release, and the card reads reachable.

  ```sh
  $FW exec $RUN -- $PW cli $D goto '{nodes.home.browserUrl}/settings/download-clients'
  $FW exec $RUN --proof download-clients/download-clients-reachable -- $PW cli $D run-code 'async page => { const line = page.getByRole("article", { name: "Built-in client on seedbox" }).getByText(/^Listening on .+\. Peers from the internet have reached port \d+\.$/); await line.waitFor({ timeout: 60000 }); return line.textContent(); }'
  $FW exec $RUN --proof download-clients/download-clients-reachable -- $PW cli $D screenshot --full-page --filename={runDir}/proof/download-clients/download-clients-reachable/desktop.png
  $FW exec $RUN --proof download-clients/download-clients-reachable -- curl -sS -H 'X-Api-Key: {apiKey}' '{nodes.home.url}/api/v1/download-clients'
  ```

- **Hold new downloads on low disk.** Give seedbox's downloads volume its own levels of 100%, which any free space is below. The card and the API read the guard; grab The Quiet Orchard and it queues `disk_guard`. Then remove the volume's own levels: the guard lifts, the torrent downloads, and the card reads the listen verdict first again. On a fresh run do this before **Limit seedbox's download speed**, or after it with the limit back at 0.

  ```sh
  SB=$(jq -r '.nodes[] | select(.name == "seedbox").nodeId' $RUN/manifest.json)
  $FW exec $RUN --proof download-clients/download-clients-disk-guard -- curl -sS -X PATCH -H 'X-Api-Key: {apiKey}' -H 'Content-Type: application/json' --data-binary "{\"nodeId\":\"$SB\",\"mount\":\"/\",\"warnBelow\":{\"amount\":100,\"unit\":\"percent\"},\"errorBelow\":{\"amount\":100,\"unit\":\"percent\"}}" '{nodes.home.url}/api/v1/system/health/volumes'
  $FW exec $RUN --proof download-clients/download-clients-disk-guard -- curl -sS -H 'X-Api-Key: {apiKey}' '{nodes.home.url}/api/v1/download-clients'
  $FW exec $RUN -- $PW cli $D goto '{nodes.home.browserUrl}/settings/download-clients'
  $FW exec $RUN --proof download-clients/download-clients-disk-guard -- $PW cli $D run-code 'async page => { const s = page.getByRole("article", { name: "Built-in client on seedbox" }).getByText("New downloads on seedbox wait for disk space."); await s.waitFor({ timeout: 30000 }); return s.textContent(); }'
  $FW exec $RUN --proof download-clients/download-clients-disk-guard -- $PW cli $D snapshot 'getByRole("main")' --filename={runDir}/proof/download-clients/download-clients-disk-guard/held.aria.yml
  $FW exec $RUN --proof download-clients/download-clients-disk-guard -- $PW cli $D screenshot --full-page --filename={runDir}/proof/download-clients/download-clients-disk-guard/held.png
  $FW exec $RUN -- $PW cli $P goto '{nodes.home.browserUrl}/settings/download-clients'
  $FW exec $RUN --proof download-clients/download-clients-disk-guard -- $PW cli $P screenshot --full-page --filename={runDir}/proof/download-clients/download-clients-disk-guard/held-phone.png
  $FW exec $RUN --proof download-clients/download-clients-disk-guard -- sh -c "id=\$(curl -sS -X POST -H 'X-Api-Key: {apiKey}' -H 'Content-Type: application/json' -d '{\"contentType\":\"movie\",\"key\":\"9000001\",\"monitor\":true}' '{nodes.home.url}/api/v1/works' | jq -r .work.id); curl -sS -X POST -H 'X-Api-Key: {apiKey}' \"{nodes.home.url}/api/v1/works/\$id/search\""
  $FW exec $RUN -- bash -c "for i in \$(seq 60); do curl -sS -H 'X-Api-Key: {apiKey}' '{nodes.home.url}/api/v1/torrents?tab=all' | jq -e '.rows[0].hold == \"disk_guard\"' >/dev/null && exit 0; sleep 1; done; exit 1"
  $FW exec $RUN --proof download-clients/download-clients-disk-guard -- curl -sS -i -X DELETE -H 'X-Api-Key: {apiKey}' "{nodes.home.url}/api/v1/system/health/volumes?nodeId=$SB&mount=%2F"
  $FW exec $RUN -- bash -c "for i in \$(seq 60); do curl -sS -H 'X-Api-Key: {apiKey}' '{nodes.home.url}/api/v1/torrents?tab=all' | jq -e '.rows[0].state == \"downloading\"' >/dev/null && exit 0; sleep 1; done; exit 1"
  $FW exec $RUN --proof download-clients/download-clients-disk-guard -- curl -sS -H 'X-Api-Key: {apiKey}' '{nodes.home.url}/api/v1/download-clients'
  $FW exec $RUN --proof download-clients/download-clients-disk-guard -- $FW sse list {runDir} --as admin --event download-clients.changed
  ```

- **Put the limit back.** Save `Download limit` 0 over the API so later recipes in the run download at full speed.

  ```sh
  $FW exec $RUN --proof download-clients/download-clients-limits -- curl -sS -X PATCH -H 'X-Api-Key: {apiKey}' -H 'Content-Type: application/json' -d '{"downloadLimit":0}' '{nodes.home.url}/api/v1/download-clients/{nodes.seedbox.nodeId}/settings'
  ```

## Gotchas

- The card reads unproven until the leecher connects, which happens only after seedbox's first accepted announce of the 1080p release. Read the card before any search to see unproven.
- `Download limit` shows its unit only once it holds a number, and starts at MB/s, so pick `KB/s` before saving; 0 means no limit. Each setting saves on its own, with the Save beside it or Enter in its field. The limit holds every peer, the fake seeders on loopback included.
- A process world's Nodes share the host's disk, so a level in bytes judges whatever the host has free. 100% fires on any host; to see the guard lift on a sample rather than a level change, set a byte level a little above the free space, free that much on the host, and wait for seedbox's next sample or run `POST /api/v1/system/health/storage/check`.
- The guard follows Health's resolved error level for (seedbox, mount), so a global level moves it too, along with home's Storage items.
- A threshold's unit follows the value the field holds (min, h or days); type the number in the unit it shows.
- A Node sends its torrents' totals to the Controller every 5 minutes, so a card's bytes and a torrent's `downloaded` lag the live speed and percent.
- The Torrent client page's per-Node limits are Controller settings carried to the Node live; its listen address and downloads folder are Node settings that apply after restart ([nodes.md](./nodes.md)).
