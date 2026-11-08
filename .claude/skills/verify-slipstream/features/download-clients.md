# Download Clients

Settings > Download Clients lists a card for each Node that downloads, and the thresholds that make a release count as bad. A card holds the Node's built-in client first, then its external clients (qBittorrent, Transmission, Deluge or rTorrent the Admin already runs), then `Add a Download Client`; while the Node has an external client that is on, the card's header holds `New Grabs go to`. External clients on Nodes that no longer download sit under `Nodes that no longer download` with Delete only. A card leads with the client's listen-port check: listening on its address, and whether a peer from the internet has reached the port or it is still unproven. It shows the torrents and totals the client holds, its limits, its peer ID prefix and user agent under Details, and no Remove button, since it goes away only when the Node stops downloading. While a Node's downloads volume is below Health's error level for it, the disk guard holds that Node's new downloads: its card leads with `Waiting for disk`, the section carries the Health item, and the guard lifts on the first sample above the level or once the level moves below it. Each card links to its Node's Torrent client section, where the per-Node limits save one group at a time and apply live, and from there to Torrents filtered to that Node.

## Sub-features

- `download-clients-cards` `/settings/download-clients` lists `Built-in client on seedbox` reading `Listening on <address>:<port>. No peer from the internet has connected to port <port> yet, so it may not be reachable.` with the router hint, `0 torrents, 0 bytes uploaded and 0 bytes downloaded.`, `This card is removed once this Node stops downloading.` and no Remove; its Details list the limits, the peer ID prefix `-LT20F0-` and the user agent `libtorrent/2.0.15.0`.
- `download-clients-reachable` once the Scenario's leecher connects in after seedbox's first announce, the card reads `Peers from the internet have reached port <port>.` and the API's `reach` reads `reachable`.
- `download-clients-thresholds` `When a release counts as bad` saves `A download with no data at all is bad after` and `Unregistered twice, this far apart` one at a time with the toast `Download Clients settings saved`, and the API refuses a value outside its bounds (422 `setting_out_of_range`).
- `download-clients-disk-guard` with seedbox's downloads volume (`/` in a process world) given its own error level above its free space, the API's card carries `diskGuard` `{sentence, hint, errorBelow}` and the downloads volume as `downloads` `{mount, name, total, free, sampledAt}`, its `sentence` and `hint` the words Health's item says, the section's tab reads `Download Clients 1 warning` and opens with the Health line `Warning New downloads on seedbox …` and `Open Health`, and seedbox's card leads with the chip `Waiting for disk`, `New downloads on seedbox wait for disk space.` and `<volume> has <free> left, below its error level of <bytes>. Free up space there, or move the downloads folder to a volume with more room. Downloads already running continue.`, as does seedbox's Torrent client page. A grab then queues with `hold` `disk_guard` (see [torrents.md](./torrents.md)). Removing the volume's own levels lifts the guard at once: `diskGuard` goes, the torrent starts, and each change carries `download-clients.changed`. With a level in bytes just above the free space, freeing space lifts it at seedbox's next sample, at Storage's `Check storage every` cadence or a Storage Check now.
- `download-clients-external` on an `external-clients` run the world has added home's Transmission (fake-transmission 4.1) through the API as an Admin would, with the Client folder pairing `/downloads` with the folder home sees and `grabsTo` set to it. `GET /api/v1/download-clients` lists it under home's card's `externals` with `reach` `responding` and `productVersion` `4.1.1` once home polled it. A draft Test with the saved settings answers 200 with `product` `Transmission` and the `nodeFolder`; with a wrong password it answers 422 `download_client_test_failed`, `Transmission rejected the username or password. Check them and test again.` on `username` and `password`; with the Client folder pointed at the movie root it answers 422 on `folder`, `Transmission saves to <root>, which overlaps a library root on home. Choose a download folder outside every library root and test again.` A Grab of The Quiet Orchard goes to Transmission, which alone can download it, and imports a hard link of Transmission's file (see [torrents.md](./torrents.md)).
- `download-clients-external-kinds` on an `external-kinds` run the world has added home's qBittorrent 5.2, Transmission 4.1, Deluge 2.1 and rTorrent 0.16 through the API, each with the Client folder pairing its own name for its folder (`D:\Torrents`, `/downloads`, `/downloads`, `/srv/torrents`) with the folder home sees, and `grabsTo` set to qBittorrent. `GET /api/v1/download-clients` lists all four under home's card's `externals` with `reach` `responding` at `5.2.3`, `4.1.1`, `2.1.1` and `0.16.27`. Deluge takes no username: a draft Test with a wrong password answers 422 `download_client_test_failed`, `Deluge rejected the password. Check it and test again.` on `password` alone. rTorrent's passing Test warns `rTorrent doesn't report when it moves finished downloads. If rTorrent moves them, set an Import delay so SlipStream imports after the move.` A Deluge Test that needs a daemon answers 422 `download_client_choose_daemon` with the `daemons` its web UI lists, which a Scenario's single-daemon Deluge never plays. With `grabsTo` set to rTorrent, a Grab imports from it, and removing the Torrent with its files empties rTorrent's folder, since home deletes the files rTorrent's remove keeps (see [torrents.md](./torrents.md)).
- `download-clients-external-add` a card's `Add a Download Client` opens the `Add a Download Client` sheet: the On switch, then `Node` (only with more than one downloading Node), `Client` (qBittorrent first), `Name` (the kind's name until renamed), `Address` with the kind's example under it (`The whole address, such as http://192.168.1.20:9091/transmission/rpc. With no path, SlipStream adds /transmission/rpc.` for Transmission) and only the sign-in fields the kind takes (Transmission: `Username` and `Password`; qBittorrent adds `API key`; Deluge only `Password`). Details reads `SlipStream labels its torrents slipstream, and <kind> saves them in its own download folder. <Node> opens the paths <kind> reports as they are.` and holds `Label`, `Download folder`, `Client folders` (`Add a pair`) and, for Deluge after a Test that lists more than one daemon, `Daemon`. A Test or Save refused with 422 `download_client_choose_daemon` reads `Deluge's web UI lists more than one daemon. Choose the daemon SlipStream uses and test again.` and opens Details on `Daemon`, listing the `daemons` the refusal carries; the next Test sends the one chosen. `Test` posts the draft to `/external-download-clients/test` and reads `The Test passed. <Node> reached Transmission <version>.` with each warning as an amber line; `Save` posts it to `/external-download-clients`, which tests on the Node first, toasts `Added <name>` and adds the row to the card.
- `download-clients-external-row` an external client's row names it, its kind and torrent count, its status as a chip and a sentence (`Responding` with `<Node> reached Transmission <version>.`; `Not responding`, `Sign-in refused` or `Needs attention` in red with the API's `problem.sentence`; `Turned off`, `Node offline`, `Not checked yet`), `Open <name>` to its own web UI in a new tab (none for rTorrent), `Edit` and `Delete`.
- `download-clients-external-edit` `Edit` opens `Edit <name>` with the username shown, each saved secret reading `Saved` with `Replace <label>`, and Check every and Import delay under Details saving on their own; an empty secret is never sent, `Test` posts the unsaved changes to `/external-download-clients/{id}/test`, and `Save` PATCHes only what changed, toasting `Saved <name>`. A failed Test (422 `download_client_test_failed`) shows its sentence above Save, such as `Transmission rejected the username or password. Check them and test again.`, and marks each field in its `fields` with `Change this, then test again.`, opening Details when one sits there; a Test that can't run (502 `download_client_unavailable`) shows `SlipStream couldn't reach <Node> to run the Test. Test again once <Node> is back.` and keeps the form.
- `download-clients-grabs-to` `New Grabs go to` lists `Built-in client` and the Node's external clients that are on, reads `Grabs for Works whose library root is on <Node> go to this client.`, and saves `grabsTo` through `PATCH /download-clients/{nodeId}/settings` with the toast `New Grabs on <Node> go to <name>`; a refusal shows the API's sentence under it.
- `download-clients-external-delete` `Delete` asks `Delete <name>?`, saying `SlipStream stops checking <name>. Nothing in <name> changes.` or, while it holds torrents, `<name> holds <n> torrents SlipStream tracks. They stay in <name>, and SlipStream stops tracking them.` with `Stop tracking and delete`, which sends `?stopTracking=true`; a refusal shows the API's sentence in the dialog, such as `<Node>'s new Grabs go to this client. Choose another client for them first.`. A delete toasts `Deleted <name>` and takes the row off.
- `download-clients-unattached` `Nodes that no longer download` reads `SlipStream doesn't check these clients, because their Nodes no longer hold Downloading. Delete each one, or give its Node Downloading again.` and lists each such client as `<kind> on <Node>, <n> torrents` with Delete only.
- `download-clients-limits` the card's `Torrent client settings on seedbox` opens seedbox's Torrent client; a `Download limit` of 16 saved there applies live, holding a running download near 16 KB/s, and the section's `Torrents` link opens Torrents on `All` filtered to seedbox with the `Node` select reading `seedbox`.

## How to get to it (user POV)

- Desktop shell: the sidebar's `Settings`, then `Download Clients` in the section nav, `/settings/download-clients`. Phone shell: `More`, then `Settings`, then `Download Clients`, a pushed page.
- A card's `Torrent client settings on <Node>` link, or Nodes, a Node, then its `Torrent client` page tab (only on a Node that stands in Downloading); the Nodes Inspector's Downloading chip links `Torrent client` and `Torrents`.
- API: `GET /api/v1/download-clients` (cards with each Node's external clients and `grabsTo`, the clients on Nodes that no longer download as `orphans`, the `kinds`, and the thresholds), `PATCH /api/v1/download-clients/thresholds`, `GET` and `PATCH /api/v1/download-clients/{nodeId}/settings` (with `grabsTo`, an external client's id or null for the built-in client), `POST /api/v1/external-download-clients` (saved only once its Test passes on its Node), `PATCH` and `DELETE /api/v1/external-download-clients/{id}` (`?stopTracking=true` parts the torrents it holds and leaves them in the client), `POST /api/v1/external-download-clients/test` for a draft and `POST /api/v1/external-download-clients/{id}/test` for a saved client with unsaved changes, and the event `download-clients.changed`, which names `nodeIds` and `clientIds`.

## Driving it with fakeworld and playwright cli

The external-client steps need a Scenario with an external client fake (the `external-clients` Scenario, Transmission first). Until a run has one, drive `download-clients-external-*`, `download-clients-grabs-to` and `download-clients-unattached` from the web unit tests in `web/src/download-clients/` and the API.

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

- **Add, test and save Transmission.** On the `external-clients` Scenario, open seedbox's card's `Add a Download Client`, pick `Transmission`, fill the fake's address, Test, then Save. The row reads `Responding`; then pick it in `New Grabs go to`.

  ```sh
  $FW exec $RUN -- $PW cli $D goto '{nodes.home.browserUrl}/settings/download-clients'
  $FW exec $RUN -- $PW cli $D click 'getByRole("button", { name: "Add a Download Client" })'
  $FW exec $RUN -- $PW cli $D click 'getByRole("dialog").getByRole("combobox", { name: "Client" })'
  $FW exec $RUN -- $PW cli $D click 'getByRole("option", { name: "Transmission" })'
  $FW exec $RUN -- $PW cli $D fill 'getByRole("dialog").getByRole("textbox", { name: "Address" })' '{fakes.transmission.url}'
  $FW exec $RUN -- $PW cli $D click 'getByRole("dialog").getByRole("button", { name: "Test" })'
  $FW exec $RUN --proof download-clients/download-clients-external-add -- $PW cli $D run-code 'async page => { const line = page.getByRole("dialog").getByText(/^The Test passed\. seedbox reached Transmission /); await line.waitFor({ timeout: 30000 }); return line.textContent(); }'
  $FW exec $RUN -- $PW cli $D click 'getByRole("dialog").getByRole("button", { name: "Save" })'
  $FW exec $RUN --proof download-clients/download-clients-external-row -- $PW cli $D run-code 'async page => { const row = page.getByRole("list", { name: "External clients on seedbox" }).getByRole("listitem", { name: "Transmission" }); await row.getByText("Responding").waitFor({ timeout: 60000 }); return row.textContent(); }'
  $FW exec $RUN -- $PW cli $D click 'getByRole("combobox", { name: "New Grabs go to" })'
  $FW exec $RUN -- $PW cli $D click 'getByRole("option", { name: "Transmission" })'
  $FW exec $RUN --proof download-clients/download-clients-grabs-to -- curl -sS -H 'X-Api-Key: {apiKey}' '{nodes.home.url}/api/v1/download-clients/{nodes.seedbox.nodeId}/settings'
  $FW exec $RUN --proof download-clients/download-clients-external-add -- $PW cli $D screenshot --full-page --filename={runDir}/proof/download-clients/download-clients-external-add/desktop.png
  ```

- **Put the limit back.** Save `Download limit` 0 over the API so later recipes in the run download at full speed.

  ```sh
  $FW exec $RUN --proof download-clients/download-clients-limits -- curl -sS -X PATCH -H 'X-Api-Key: {apiKey}' -H 'Content-Type: application/json' -d '{"downloadLimit":0}' '{nodes.home.url}/api/v1/download-clients/{nodes.seedbox.nodeId}/settings'
  ```

- **Test home's Transmission.** On an `external-clients` run, read the client, then Test a draft with the saved settings, a wrong password, and the Client folder on the movie root. `transmission-adds list` reads empty until a Grab.

  ```sh
  $FW exec $RUN --proof download-clients/download-clients-external -- curl -sS -H 'X-Api-Key: {apiKey}' '{nodes.home.url}/api/v1/download-clients'
  TR=$(curl -sS -H 'X-Api-Key: {apiKey}' '{nodes.home.url}/api/v1/download-clients' | jq -c '.cards[0].externals[0] | {nodeId, kind, address, username, clientFolders, password: "quiet-orchard"}')
  $FW exec $RUN --proof download-clients/download-clients-external -- curl -sS -X POST -H 'X-Api-Key: {apiKey}' -H 'Content-Type: application/json' --data-binary "$TR" '{nodes.home.url}/api/v1/external-download-clients/test'
  $FW exec $RUN --proof download-clients/download-clients-external -- curl -sS -i -X POST -H 'X-Api-Key: {apiKey}' -H 'Content-Type: application/json' --data-binary "$(jq -c '.password = "wrong"' <<<"$TR")" '{nodes.home.url}/api/v1/external-download-clients/test'
  $FW exec $RUN --proof download-clients/download-clients-external -- curl -sS -i -X POST -H 'X-Api-Key: {apiKey}' -H 'Content-Type: application/json' --data-binary "$(jq -c --arg root "$(jq -r .mediaDir $RUN/manifest.json)/movies" '.clientFolders[0].node = $root' <<<"$TR")" '{nodes.home.url}/api/v1/external-download-clients/test'
  $FW exec $RUN --proof download-clients/download-clients-external -- $FW transmission-adds list $RUN
  ```

- **Test home's other kinds.** On an `external-kinds` run, read the four clients, then Test Deluge with a wrong password and rTorrent with its saved settings.

  ```sh
  $FW exec $RUN --proof download-clients/download-clients-external-kinds -- sh -c "curl -sS -H 'X-Api-Key: {apiKey}' '{nodes.home.url}/api/v1/download-clients' | jq -c '.cards[0].externals[] | {name, reach, productVersion}'"
  DL=$(curl -sS -H 'X-Api-Key: {apiKey}' '{nodes.home.url}/api/v1/download-clients' | jq -c '.cards[0].externals[] | select(.kind == "deluge") | {nodeId, kind, address, clientFolders, password: "wrong"}')
  $FW exec $RUN --proof download-clients/download-clients-external-kinds -- curl -sS -i -X POST -H 'X-Api-Key: {apiKey}' -H 'Content-Type: application/json' --data-binary "$DL" '{nodes.home.url}/api/v1/external-download-clients/test'
  RT=$(curl -sS -H 'X-Api-Key: {apiKey}' '{nodes.home.url}/api/v1/download-clients' | jq -c '.cards[0].externals[] | select(.kind == "rtorrent") | {nodeId, kind, address, username, clientFolders, password: "quiet-orchard"}')
  $FW exec $RUN --proof download-clients/download-clients-external-kinds -- curl -sS -X POST -H 'X-Api-Key: {apiKey}' -H 'Content-Type: application/json' --data-binary "$RT" '{nodes.home.url}/api/v1/external-download-clients/test'
  ```

## Gotchas

- fake-qbittorrent, fake-deluge and fake-rtorrent download the same way into `fakes/<kind>/<node>`, and play the same cues under their own prefix (`qbittorrent-pause`, `deluge-late-add`, `rtorrent-remove`). fake-deluge adds `deluge-switch` and `deluge-disconnect`. A Scenario's Deluge lists one daemon, so its Test never needs one chosen.
- fake-transmission downloads 1 MiB a second into the run's `fakes/transmission/<node>` folder under its scratch dir, calling it by the Scenario's `folder` (`/downloads`). An external Torrent lands in a subfolder named for its Content Type, as the built-in client's do. Its faults are fault rules on its front for the Node (`reset` for a client that has stopped; `refuse` turns away only new connections, and the Node's kept-alive one still answers) and its cues: `transmission-remove`, `-pause`, `-stall`, `-error`, `-tracker`, `-empty`, `-drop-add`, `-late-add` and `-restart`.
- Save is the Test: create and an edit that changes the Node, kind, address, sign-in, label, folder, Client folders or daemon test on the Node first and save only on a pass, so a Save with the fake stopped answers `download_client_test_failed` and nothing is saved. An edit of only the name or the On switch saves without a Test.
- Check every and Import delay show only on a saved client's sheet; a new client starts at their defaults.
- Delete has no Stop tracking yet, so a client holding torrents refuses with `download_client_has_torrents`.

- The card reads unproven until the leecher connects, which happens only after seedbox's first accepted announce of the 1080p release. Read the card before any search to see unproven.
- `Download limit` shows its unit only once it holds a number, and starts at MB/s, so pick `KB/s` before saving; 0 means no limit. Each setting saves on its own, with the Save beside it or Enter in its field. The limit holds every peer, the fake seeders on loopback included.
- A process world's Nodes share the host's disk, so a level in bytes judges whatever the host has free. 100% fires on any host; to see the guard lift on a sample rather than a level change, set a byte level a little above the free space, free that much on the host, and wait for seedbox's next sample or run `POST /api/v1/system/health/storage/check`.
- The guard follows Health's resolved error level for (seedbox, mount), so a global level moves it too, along with home's Storage items.
- A threshold's unit follows the value the field holds (min, h or days); type the number in the unit it shows.
- A Node sends its torrents' totals to the Controller every 5 minutes, so a card's bytes and a torrent's `downloaded` lag the live speed and percent.
- The Torrent client page's per-Node limits are Controller settings carried to the Node live; its listen address and downloads folder are Node settings that apply after restart ([nodes.md](./nodes.md)).
