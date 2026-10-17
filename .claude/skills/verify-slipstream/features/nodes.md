# Nodes

Nodes is where an Admin runs a Node in the cluster. `/nodes` lands on the Node the browser is talking to, at `/nodes/<id>`, titled with the Node's name, and a Node's pages are its sections: Overview, Web and API, Logs and Log settings. The Overview lists the Node's build version, when it started, its data folder and the addresses of its web, torrent and link listeners and its control socket, for any Node through the per-Node API. Web and API and Log settings edit the Node's own `config.yaml`, one field at a time. Logs streams the Node's log live, and Download logs saves its log folder as a zip. Restart restarts any Node, and the browser's own Node from the account menu, More and the palette too. `GET /api/v1/system` returns the facts of the Node that answers, plus the build stamp. A Worker that holds Serving answers its own address with its Controller's app, the Controller's sign-in and passkeys included, and while its Controller can't be reached it shows a status card of what it holds.

## Sub-features

- `nodes-overview` `/nodes` and the old `/system` land on `/nodes/<id>` for the Node that served the page, whose Overview lists Version, Running since, Data folder, Web and API, Torrents, Node link and Control socket, and the data folder sits in the run's own scratch dir, not a developer database (P-255).
- `nodes-sections` in the desktop shell the page tabs, a nav named for the Node, link Overview, Web and API, Logs and Log settings and mark the current one; on phone the Node's page, titled `Nodes`, lists the other sections, and each is a pushed page whose back link reads `Nodes`.
- `nodes-picker` with a Worker paired, the `Node` select lists every Node with an online, offline or other-version dot, and choosing one keeps the section.
- `nodes-system-api` `GET /api/v1/system` with the API key returns the PID, the data dir, the listeners and `build` with `stamped`, `commit` and `fingerprint`.
- `nodes-web-settings` Web and API shows `Port` with `Default 8484.` and `Applies after restart.`, and `Ports to try`; a saved port toasts `Web and API settings saved` and shows the warning line `Restart to use port <port>` with `Restart now`, a port out of range answers 422 with `Between 1 and 65535.`, and a port an environment variable sets is locked with `Set by SLIPSTREAM_HTTP_LISTEN` (P-244). The running port is the one the Node bound, so saving the port it listens on, after a fallback or on a Node that started on port 0, waits for no restart.
- `nodes-port-fallback` a Node whose HTTP port another program holds at start binds one of the next ports, and its Overview reads `Port <configured> was taken, so SlipStream is on <bound>.` with `Change port` (P-246).
- `nodes-locked-key` a key that `.env` in the Node's data dir or its environment sets reads `Set by <variable>` with the control disabled, `PATCH .../config` answers 409 `set_by_environment` and `slipstream config set` exits 3 with `<key>: Set by <variable>.` (P-244).
- `nodes-log-settings` Log settings shows `Level`, `Rotate at`, `Files kept` and `Days kept` (0 reads `No limit`), `Compress rotated files` and `Lines kept for the viewer`; choosing a level saves it at once and toasts `Log settings saved`, the Node logs at the new level with no restart, and the `Log folder` reads the Node's log dir with `Copy` (P-247, P-248).
- `nodes-logs-viewer` Logs shows the Node's recent lines, then each new one: the time, a level chip, the subsystem and the message; it filters by minimum level, subsystem and search, `Pause` holds the list and reads `Paused, <n> new` until `Resume`, `Clear` empties the view only, a row expands to every attribute with `Copy as JSON`, scrolling up reads `Not following,` with `Jump to latest`, and a restarted Node draws `SlipStream restarted at <time>` (P-249).
- `nodes-logs-download` `Download logs` saves a zip of the Node's log folder holding `slipstream.log`, and a Node that doesn't answer toasts `Couldn't download the logs.` instead (P-250).
- `nodes-restart` the account menu's, More's and the palette's `Restart SlipStream…` and the Nodes header's `Restart` ask `Restart SlipStream?` (or `Restart <name>?` for a Worker); the browser's own Node shows `Restarting SlipStream…` over the page, and the page reloads onto the same address once a later `startedAt` answers, while a Worker's sections read `<name> is restarting…` until its link is up again. A restart onto a saved port reads `SlipStream is moving to port <port>` with `Open SlipStream on port <port>`, which opens the same page on the new port (P-010).
- `nodes-serving` on a Worker holding Serving, its own address serves the Controller's app: the Admin signs in there with the Controller's PIN and lands on the Library, `GET /api/v1/system` there reports the Controller's data dir, the live stream carries the Controller's events, and a passkey added there signs in there.
- `nodes-serving-unreachable` with the Controller stopped, every page at the Serving Worker's address lands on `/serving`, headed `home can't be reached from seedbox since <time>.` over `seedbox keeps seeding and finishes what it started.` and `Holding Searching and Downloading · seeding 0 torrents.`; its API answers 503 `controller_unreachable`, and `GET /api/v1/serving` answers the card's facts and no request. Once the Controller answers again the card goes back to the app.
- `nodes-cli` `slipstream config set --data-dir <dir> <key> <value>` prints `<key>: <value>`, ending `, applies after restart` for a restart key, the Node's page shows the change, `config unset` prints the default, and `slipstream restart --data-dir <dir>` prints the Node's web address once its successor answers.

## How to get to it (user POV)

- Desktop shell: the sidebar's `Nodes`, below the separator and above `Tasks`, which opens `/nodes` and lands on `/nodes/<id>`; the page tabs `Overview`, `Web and API`, `Logs` and `Log settings` sit under the Node's name, and `Restart` is the page's action in the top bar. The `Nodes` row stays current on every Node page. The account menu holds `Restart SlipStream…` between the Startup sequence switch and `Sign out`.
- Phone shell: the `More` tab, then the `Nodes` button in the sheet `More`, listed above `Tasks`, then a section's row in the list named for the Node; the Node's page carries `Restart`. More holds `Restart SlipStream…` above `Sign out`. On Logs the filters sit behind `Filter`.
- Both shells: the palette's `Go to Nodes`, `Go to Overview`, `Go to Web and API`, `Go to Logs` and `Go to Log settings` for the browser's own Node, and `Restart SlipStream…`.
- API (Admin): `GET /api/v1/system`, `GET /api/v1/nodes` for the Node ids, then per Node `GET /api/v1/nodes/{id}/system`, `GET` and `PATCH /api/v1/nodes/{id}/config` (a key to value map, `null` resets), `POST /api/v1/nodes/{id}/restart` (202 with `startedAt` and `boundPort`), `GET /api/v1/nodes/{id}/logs/stream` (server-sent `log.record` and `log.reset`, resuming with `?after=<seq>`) and `GET /api/v1/nodes/{id}/logs/archive`; the live event `node.settings-changed`.
- Serving: a Worker that holds Serving, at its own address (`{nodes.seedbox.browserUrl}` on `serving`); `/serving` is its status card, and `GET /api/v1/serving` (no sign-in) answers any Worker's facts.
- Host CLI: `slipstream version` prints the same build stamp; `slipstream config set --data-dir <dir> <key> <value>`, `config unset` and `restart --data-dir <dir>`.

## Driving it with fakeworld and playwright cli

Preconditions:

- The baseline: a `home` run, doctor passing, `{runId}-admin-desktop` signed in.
- The Serving steps want their own `serving` run, where seedbox holds Serving too, with doctor passing and `$FW browser $RUN` run; they open their own session at seedbox's address.

Steps:

- **Open Nodes.** Choose `Nodes` in the sidebar. The URL ends `/nodes/<id>`, the page is headed `home`, the tab `Overview` is current, Version reads the binary's short commit, and Data folder reads the manifest's `{nodes.home.dataDir}`.

  ```sh
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop click 'getByRole("navigation", { name: "Main" }).getByRole("link", { name: "Nodes" })'
  $FW exec $RUN --proof nodes/nodes-overview -- $PW cli -s={runId}-admin-desktop snapshot --filename={runDir}/proof/nodes/nodes-overview/overview.aria.yml
  $FW exec $RUN --proof nodes/nodes-overview -- $PW cli -s={runId}-admin-desktop screenshot --filename={runDir}/proof/nodes/nodes-overview/overview.png
  ```

- **Walk the sections.** Choose `Logs` in the page tabs named `home`. It becomes current and the breadcrumbs read `Nodes`, `home`, `Logs`.

  ```sh
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop click 'getByRole("navigation", { name: "home" }).getByRole("link", { name: "Logs" })'
  $FW exec $RUN --proof nodes/nodes-sections -- $PW cli -s={runId}-admin-desktop snapshot --filename={runDir}/proof/nodes/nodes-sections/logs.aria.yml
  ```

- **Read it over the API and the host CLI.** The API's `dataDir` and listeners equal the Overview's rows, and `slipstream version` prints the same commit.

  ```sh
  $FW exec $RUN --proof nodes/nodes-system-api -- curl -sS -H 'X-Api-Key: {apiKey}' '{nodes.home.url}/api/v1/system'
  $FW exec $RUN --proof nodes/nodes-system-api -- {runDir}/bin/slipstream version
  ```

- **Save a port, then put it back.** Choose `Web and API`. `Port` reads the manifest's HTTP port. Enter 9000 and `Save`: the toast reads `Web and API settings saved` and the warning line `Restart to use port 9000` appears with `Restart now`. Over the API, 70000 answers 422 with `Between 1 and 65535.`. Save the manifest's port again over the API, and the warning line goes.

  ```sh
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop click 'getByRole("navigation", { name: "home" }).getByRole("link", { name: "Web and API" })'
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop fill 'getByRole("textbox", { name: "Port", exact: true })' 9000
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop click 'getByRole("button", { name: "Save" })'
  $FW exec $RUN --proof nodes/nodes-web-settings -- $PW cli -s={runId}-admin-desktop snapshot 'getByRole("main")' --filename={runDir}/proof/nodes/nodes-web-settings/pending.aria.yml
  $FW exec $RUN --proof nodes/nodes-web-settings -- $PW cli -s={runId}-admin-desktop screenshot --filename={runDir}/proof/nodes/nodes-web-settings/pending.png
  $FW exec $RUN --proof nodes/nodes-web-settings -- curl -sS -X PATCH -H 'X-Api-Key: {apiKey}' -H 'Content-Type: application/json' -d '{"http.listen":70000}' '{nodes.home.url}/api/v1/nodes/{nodes.home.nodeId}/config'
  $FW exec $RUN --proof nodes/nodes-web-settings -- sh -c 'a={nodes.home.listeners.http}; curl -sS -X PATCH -H "X-Api-Key: {apiKey}" -H "Content-Type: application/json" -d "{\"http.listen\":${a##*:}}" "{nodes.home.url}/api/v1/nodes/{nodes.home.nodeId}/config"'
  ```

- **See it on phone.** In `{runId}-admin-phone`, choose `More`, then `Nodes`: the page is titled `Nodes`, lists the Node under `Node` and its sections under `Sections`. Choose `Logs`: it is a pushed page headed `Logs` with the back link `Nodes`, and its filters sit behind `Filter`.

  ```sh
  $FW exec $RUN -- $PW cli -s={runId}-admin-phone click 'getByRole("navigation", { name: "Tabs" }).getByRole("button", { name: "More" })'
  $FW exec $RUN -- $PW cli -s={runId}-admin-phone click 'getByRole("dialog", { name: "More" }).getByRole("button", { name: "Nodes" })'
  $FW exec $RUN --proof nodes/nodes-overview -- $PW cli -s={runId}-admin-phone screenshot --full-page --filename={runDir}/proof/nodes/nodes-overview/overview-phone.png
  $FW exec $RUN -- $PW cli -s={runId}-admin-phone click 'getByRole("list", { name: "home" }).getByRole("link", { name: "Logs" })'
  $FW exec $RUN --proof nodes/nodes-sections -- $PW cli -s={runId}-admin-phone snapshot --filename={runDir}/proof/nodes/nodes-sections/logs-phone.aria.yml
  $FW exec $RUN -- $PW cli -s={runId}-admin-phone click 'getByRole("button", { name: "Filter" })'
  $FW exec $RUN --proof nodes/nodes-logs-viewer -- $PW cli -s={runId}-admin-phone screenshot --filename={runDir}/proof/nodes/nodes-logs-viewer/filter-phone.png
  $FW exec $RUN -- $PW cli -s={runId}-admin-phone press Escape
  ```

- **Log at Debug.** Choose `Log settings`, then `Level` and `Debug`. The toast reads `Log settings saved`, `Reset to default` appears, and the API reads `debug` as both value and running.

  ```sh
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop click 'getByRole("navigation", { name: "home" }).getByRole("link", { name: "Log settings" })'
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop click 'getByRole("combobox", { name: "Level" })'
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop click 'getByRole("option", { name: "Debug" })'
  $FW exec $RUN --proof nodes/nodes-log-settings -- $PW cli -s={runId}-admin-desktop screenshot --filename={runDir}/proof/nodes/nodes-log-settings/debug.png
  $FW exec $RUN --proof nodes/nodes-log-settings -- curl -sS -H 'X-Api-Key: {apiKey}' '{nodes.home.url}/api/v1/nodes/{nodes.home.nodeId}/config'
  ```

- **Watch a Debug line arrive, through Pause.** Choose `Logs`, then knock on the link port with bytes that aren't TLS: a row `link handshake failed` arrives with the `Debug` chip. Choose `Pause` and knock again: the status line reads `Paused, <n> new` and the list holds still. `Resume` shows the second row. Choosing a row expands its attributes with `Copy as JSON`.

  ```sh
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop click 'getByRole("navigation", { name: "home" }).getByRole("link", { name: "Logs" })'
  $FW exec $RUN -- bash -c 'a={nodes.home.listeners.link}; exec 3<>/dev/tcp/${a%:*}/${a##*:}; printf "not tls\r\n\r\n" >&3'
  $FW exec $RUN --proof nodes/nodes-logs-viewer -- $PW cli -s={runId}-admin-desktop run-code 'async page => { await page.getByRole("list", { name: "Log" }).getByText("link handshake failed").waitFor({ timeout: 15000 }); }'
  $FW exec $RUN --proof nodes/nodes-logs-viewer -- $PW cli -s={runId}-admin-desktop screenshot --filename={runDir}/proof/nodes/nodes-logs-viewer/debug-line.png
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop click 'getByRole("button", { name: "Pause" })'
  $FW exec $RUN -- bash -c 'a={nodes.home.listeners.link}; exec 3<>/dev/tcp/${a%:*}/${a##*:}; printf "not tls\r\n\r\n" >&3'
  $FW exec $RUN --proof nodes/nodes-logs-viewer -- $PW cli -s={runId}-admin-desktop run-code 'async page => { await page.getByText(/^Paused, \d+ new$/).waitFor({ timeout: 15000 }); }'
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop click 'getByRole("button", { name: "Resume" })'
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop click 'getByRole("list", { name: "Log" }).getByRole("button", { name: /link handshake failed/ }).last()'
  $FW exec $RUN --proof nodes/nodes-logs-viewer -- $PW cli -s={runId}-admin-desktop snapshot 'getByRole("main")' --filename={runDir}/proof/nodes/nodes-logs-viewer/expanded.aria.yml
  ```

- **Search, filter, follow and clear.** Type `handshake` in `Search the log`: only the `link handshake failed` rows stay. Clear it and choose `Warn` in `Minimum level`: the Debug and Info rows go. Choose `Debug` again, knock 60 times, and scroll the list up: the status reads `Not following,` with `Jump to latest`. `Clear` empties the view, while the stream still serves every line.

  ```sh
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop fill 'getByRole("searchbox", { name: "Search the log" })' handshake
  $FW exec $RUN --proof nodes/nodes-logs-viewer -- $PW cli -s={runId}-admin-desktop screenshot --filename={runDir}/proof/nodes/nodes-logs-viewer/search.png
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop fill 'getByRole("searchbox", { name: "Search the log" })' ''
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop click 'getByRole("radiogroup", { name: "Minimum level" }).getByRole("radio", { name: "Warn" })'
  $FW exec $RUN --proof nodes/nodes-logs-viewer -- $PW cli -s={runId}-admin-desktop screenshot --filename={runDir}/proof/nodes/nodes-logs-viewer/min-warn.png
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop click 'getByRole("radiogroup", { name: "Minimum level" }).getByRole("radio", { name: "Debug" })'
  $FW exec $RUN -- bash -c 'a={nodes.home.listeners.link}; for i in $(seq 60); do exec 3<>/dev/tcp/${a%:*}/${a##*:}; printf "not tls\r\n\r\n" >&3; exec 3>&-; done'
  $FW exec $RUN --proof nodes/nodes-logs-viewer -- $PW cli -s={runId}-admin-desktop run-code 'async page => { await page.waitForTimeout(2000); await page.getByRole("list", { name: "Log" }).hover(); await page.mouse.wheel(0, -3000); await page.getByText(/Not following/).waitFor({ timeout: 10000 }); }'
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop click 'getByRole("button", { name: "Jump to latest" })'
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop click 'getByRole("button", { name: "Clear" })'
  $FW exec $RUN --proof nodes/nodes-logs-viewer -- $PW cli -s={runId}-admin-desktop screenshot --filename={runDir}/proof/nodes/nodes-logs-viewer/cleared.png
  ```

- **Download the logs.** Choose `Download logs` and keep the download: the zip lists `slipstream.log`. The API's archive is the same zip.

  ```sh
  $FW exec $RUN --proof nodes/nodes-logs-download -- $PW cli -s={runId}-admin-desktop run-code 'async page => { const saved = page.waitForEvent("download"); await page.getByRole("button", { name: "Download logs" }).click(); await (await saved).saveAs("{runDir}/proof/nodes/nodes-logs-download/logs.zip"); }'
  $FW exec $RUN --proof nodes/nodes-logs-download -- unzip -l {runDir}/proof/nodes/nodes-logs-download/logs.zip
  $FW exec $RUN --proof nodes/nodes-logs-download -- curl -sS -o {runDir}/proof/nodes/nodes-logs-download/api.zip -w '%{http_code} %{content_type}\n' -H 'X-Api-Key: {apiKey}' '{nodes.home.url}/api/v1/nodes/{nodes.home.nodeId}/logs/archive'
  ```

- **Choose a Worker in the picker.** On `Logs`, choose `Node`, then `seedbox`: the URL keeps `/logs`, the heading reads `seedbox` and its lines stream through the Controller. Its Overview equals the per-Node API's facts.

  ```sh
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop click 'getByRole("combobox", { name: "Node" })'
  $FW exec $RUN --proof nodes/nodes-picker -- $PW cli -s={runId}-admin-desktop snapshot 'getByRole("listbox")' --filename={runDir}/proof/nodes/nodes-picker/options.aria.yml
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop click 'getByRole("option", { name: /seedbox/ })'
  $FW exec $RUN --proof nodes/nodes-picker -- $PW cli -s={runId}-admin-desktop run-code 'async page => { await page.getByRole("heading", { name: "seedbox", level: 1 }).waitFor({ timeout: 10000 }); await page.getByRole("list", { name: "Log" }).getByRole("listitem").first().waitFor({ timeout: 15000 }); return page.url(); }'
  $FW exec $RUN --proof nodes/nodes-picker -- curl -sS -H 'X-Api-Key: {apiKey}' '{nodes.home.url}/api/v1/nodes/{nodes.seedbox.nodeId}/system'
  ```

- **Restart the Worker.** The header's `Restart` asks `Restart seedbox?`; `Restart` shows `seedbox is restarting…` and the sections come back with a later `Running since`. The Worker is back in under a second, so click and wait in one step.

  ```sh
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop click 'getByRole("banner").getByRole("button", { name: "Restart" })'
  $FW exec $RUN --proof nodes/nodes-restart -- $PW cli -s={runId}-admin-desktop run-code 'async page => { await page.getByRole("dialog", { name: "Restart seedbox?" }).getByRole("button", { name: "Restart" }).click(); await page.getByText("seedbox is restarting…").first().waitFor({ timeout: 15000 }); await page.screenshot({ path: "{runDir}/proof/nodes/nodes-restart/worker-restarting.png" }); }'
  $FW exec $RUN --proof nodes/nodes-restart -- curl -sS -H 'X-Api-Key: {apiKey}' '{nodes.home.url}/api/v1/nodes/{nodes.seedbox.nodeId}/system'
  ```

- **Lock a key.** As the host operator, write `SLIPSTREAM_LOG_LEVEL=debug` to the seedbox's `.env`. `slipstream config set` exits 3 with `log.level: Set by SLIPSTREAM_LOG_LEVEL.`, the API answers 409, and after a reload seedbox's `Log settings` shows `Level` disabled with `Set by SLIPSTREAM_LOG_LEVEL`. Remove the file after.

  ```sh
  $FW exec $RUN --proof nodes/nodes-locked-key -- sh -c 'printf "SLIPSTREAM_LOG_LEVEL=debug\n" > {nodes.seedbox.dataDir}/.env'
  $FW exec $RUN --proof nodes/nodes-locked-key -- {runDir}/bin/slipstream config set --data-dir {nodes.seedbox.dataDir} log.level warn
  $FW exec $RUN --proof nodes/nodes-locked-key -- curl -sS -X PATCH -H 'X-Api-Key: {apiKey}' -H 'Content-Type: application/json' -d '{"log.level":"warn"}' '{nodes.home.url}/api/v1/nodes/{nodes.seedbox.nodeId}/config'
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop click 'getByRole("navigation", { name: "seedbox" }).getByRole("link", { name: "Log settings" })'
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop reload
  $FW exec $RUN --proof nodes/nodes-locked-key -- $PW cli -s={runId}-admin-desktop run-code 'async page => { await page.getByText("Set by SLIPSTREAM_LOG_LEVEL").waitFor({ timeout: 15000 }); }'
  $FW exec $RUN --proof nodes/nodes-locked-key -- $PW cli -s={runId}-admin-desktop screenshot --filename={runDir}/proof/nodes/nodes-locked-key/locked.png
  $FW exec $RUN --proof nodes/nodes-locked-key -- rm {nodes.seedbox.dataDir}/.env
  ```

- **Fail a download.** On seedbox's `Logs`, stall the Node link, then choose `Download logs`: the toast reads `Couldn't download the logs. The Node is offline.` Clear the fault after.

  ```sh
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop click 'getByRole("navigation", { name: "seedbox" }).getByRole("link", { name: "Logs" })'
  $FW fault set $RUN hold-link link/home stall
  $FW exec $RUN --proof nodes/nodes-logs-download -- $PW cli -s={runId}-admin-desktop run-code 'async page => { await page.getByRole("button", { name: "Download logs" }).click(); await page.getByText(/^Couldn.t download the logs/).waitFor({ timeout: 90000 }); }'
  $FW fault clear $RUN hold-link
  ```

- **Take seedbox's port, then use the host CLI.** Stop seedbox and reload: its sections read `seedbox is offline. Its settings and logs come back when it reconnects.` Hold its HTTP port from a second shell for 150 seconds and start it: its Overview reads `Port <port> was taken, so SlipStream is on <next port>.` with `Change port`. Then `slipstream config set` and `unset` change seedbox's `log.level`, and once the holder exits `slipstream restart` brings it back on its own port.

  ```sh
  $FW node stop $RUN seedbox
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop reload
  $FW exec $RUN --proof nodes/nodes-picker -- $PW cli -s={runId}-admin-desktop run-code 'async page => { await page.getByText("seedbox is offline. Its settings and logs come back when it reconnects.").waitFor({ timeout: 30000 }); }'
  $FW exec $RUN --proof nodes/nodes-port-fallback -- timeout 150 python3 -c 'import socket,sys,time; h,p=sys.argv[1].rsplit(":",1); s=socket.socket(); s.bind((h,int(p))); s.listen(); time.sleep(150)' '{nodes.seedbox.listeners.http}'   # in a second shell
  $FW node start $RUN seedbox
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop click 'getByRole("navigation", { name: "seedbox" }).getByRole("link", { name: "Overview" })'
  $FW exec $RUN --proof nodes/nodes-port-fallback -- $PW cli -s={runId}-admin-desktop run-code 'async page => { await page.getByText(/was taken, so SlipStream is on/).waitFor({ timeout: 60000 }); }'
  $FW exec $RUN --proof nodes/nodes-port-fallback -- curl -sS -H 'X-Api-Key: {apiKey}' '{nodes.home.url}/api/v1/nodes/{nodes.seedbox.nodeId}/system'
  $FW exec $RUN --proof nodes/nodes-cli -- {runDir}/bin/slipstream config set --data-dir {nodes.seedbox.dataDir} log.level debug
  $FW exec $RUN --proof nodes/nodes-cli -- {runDir}/bin/slipstream config unset --data-dir {nodes.seedbox.dataDir} log.level
  $FW exec $RUN --proof nodes/nodes-cli -- {runDir}/bin/slipstream restart --data-dir {nodes.seedbox.dataDir}   # after the holder exits
  $FW doctor $RUN
  ```

- **Move home to another port (last but one).** Choose `home` in the picker, then `Web and API`, enter 9000 and `Save`, then `Restart now` and `Restart`: the page reads `SlipStream is moving to port 9000`, and `Open SlipStream on port 9000` opens the same page on port 9000. The manifest follows the move.

  ```sh
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop fill 'getByRole("textbox", { name: "Port", exact: true })' 9000
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop click 'getByRole("button", { name: "Save" })'
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop click 'getByRole("button", { name: "Restart now" })'
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop click 'getByRole("dialog", { name: "Restart SlipStream?" }).getByRole("button", { name: "Restart" })'
  $FW exec $RUN --proof nodes/nodes-restart -- $PW cli -s={runId}-admin-desktop run-code 'async page => { await page.getByText("SlipStream is moving to port 9000").waitFor({ timeout: 90000 }); }'
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop click 'getByRole("button", { name: "Open SlipStream on port 9000" })'
  $FW exec $RUN --proof nodes/nodes-restart -- $PW cli -s={runId}-admin-desktop run-code 'async page => { await page.waitForURL(/:9000\//, { timeout: 30000 }); return page.url(); }'
  $FW doctor $RUN
  ```

- **Restart from the account menu (last).** Choose `Account`, then `Restart SlipStream…`, then `Restart` in `Restart SlipStream?`. `Restarting SlipStream…` covers the page, the page reloads onto the same address, and the API's `startedAt` is later than before. The event log records `node.restarted` with the same PID on Linux and macOS.

  ```sh
  $FW exec $RUN --proof nodes/nodes-restart -- curl -sS -H 'X-Api-Key: {apiKey}' '{nodes.home.url}/api/v1/system'
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop click 'getByRole("button", { name: "Account" })'
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop click 'getByRole("menuitem", { name: "Restart SlipStream…" })'
  $FW exec $RUN --proof nodes/nodes-restart -- $PW cli -s={runId}-admin-desktop screenshot --filename={runDir}/proof/nodes/nodes-restart/confirm.png
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop click 'getByRole("dialog", { name: "Restart SlipStream?" }).getByRole("button", { name: "Restart" })'
  $FW exec $RUN --proof nodes/nodes-restart -- $PW cli -s={runId}-admin-desktop screenshot --filename={runDir}/proof/nodes/nodes-restart/restarting.png
  $FW exec $RUN --proof nodes/nodes-restart -- $PW cli -s={runId}-admin-desktop run-code 'async page => { await page.waitForEvent("load", { timeout: 90000 }); await page.getByRole("heading", { name: "home", level: 1 }).waitFor({ timeout: 30000 }); }'
  $FW exec $RUN --proof nodes/nodes-restart -- curl -sS -H 'X-Api-Key: {apiKey}' '{nodes.home.url}/api/v1/system'
  $FW doctor $RUN
  ```

- **Sign in at the Serving Worker and add a passkey (`serving` run).** Open a session at seedbox's address with Chromium's virtual authenticator, sign in with the Admin's email and PIN, and choose `Add a passkey` in `Sign in faster here with a passkey`: the page lands on `/library` at seedbox's address, and its `GET /api/v1/system` reports `{nodes.home.dataDir}`.

  ```sh
  $FW exec $RUN -- $PW cli -s={runId}-seedbox-desktop open --config={runDir}/browser/desktop/playwright-cli.json
  $FW exec $RUN -- $PW cli -s={runId}-seedbox-desktop goto '{nodes.seedbox.browserUrl}/'
  $FW exec $RUN -- $PW cli -s={runId}-seedbox-desktop run-code 'async page => { const cdp = await page.context().newCDPSession(page); await cdp.send("WebAuthn.enable"); await cdp.send("WebAuthn.addVirtualAuthenticator", { options: { protocol: "ctap2", transport: "internal", hasResidentKey: true, hasUserVerification: true, isUserVerified: true, automaticPresenceSimulation: true } }); }'
  $FW exec $RUN --proof nodes/nodes-serving -- $PW cli -s={runId}-seedbox-desktop run-code 'async page => { await page.getByLabel("Email").fill("ada@example.test"); await page.getByRole("textbox", { name: "PIN", exact: true }).pressSequentially("2468"); const prompt = page.getByRole("dialog", { name: "Sign in faster here with a passkey" }); await prompt.getByRole("button", { name: "Add a passkey" }).click(); await prompt.waitFor({ state: "hidden" }); await page.getByRole("heading", { name: "Library" }).waitFor(); return page.url(); }'
  $FW exec $RUN --proof nodes/nodes-serving -- $PW cli -s={runId}-seedbox-desktop screenshot --filename={runDir}/proof/nodes/nodes-serving/library.png
  $FW exec $RUN --proof nodes/nodes-serving -- $PW cli -s={runId}-seedbox-desktop run-code 'async page => (await page.evaluate(async () => (await fetch("/api/v1/system")).json())).dataDir'
  $FW exec $RUN --proof nodes/nodes-serving -- curl -sS -H 'X-Api-Key: {apiKey}' '{nodes.seedbox.url}/api/v1/system'
  ```

- **Sign out, then in with the passkey.** Choose `Account`, `Sign out` and `Sign out` in `Sign out of this device?`: the sign-in page's passkey ceremony answers 200 at once and the page is back on the Library at seedbox's address.

  ```sh
  $FW exec $RUN -- $PW cli -s={runId}-seedbox-desktop click 'getByRole("button", { name: "Account" })'
  $FW exec $RUN -- $PW cli -s={runId}-seedbox-desktop click 'getByRole("menuitem", { name: "Sign out" })'
  $FW exec $RUN --proof nodes/nodes-serving -- $PW cli -s={runId}-seedbox-desktop run-code 'async page => { const asserted = page.waitForResponse((r) => r.request().method() === "POST" && /\/api\/v1\/passkeys\/assertion\/[^/]+$/u.test(new URL(r.url()).pathname)); await page.getByRole("dialog", { name: "Sign out of this device?" }).getByRole("button", { name: "Sign out" }).click(); const status = (await asserted).status(); await page.getByRole("heading", { name: "Library" }).waitFor(); return [status, page.url()]; }'
  ```

- **Stop the Controller, then start it.** Stop home and reload: the page lands on `/serving` with the card's three lines, the API at seedbox's address answers 503, and `GET /api/v1/serving` names home, `since` and the Roles held. Start home: within a few seconds the card goes back to the Library.

  ```sh
  $FW node stop $RUN home
  $FW exec $RUN -- $PW cli -s={runId}-seedbox-desktop reload
  $FW exec $RUN --proof nodes/nodes-serving-unreachable -- $PW cli -s={runId}-seedbox-desktop run-code 'async page => { await page.waitForURL(/\/serving$/, { timeout: 15000 }); return await page.getByRole("main").innerText(); }'
  $FW exec $RUN --proof nodes/nodes-serving-unreachable -- $PW cli -s={runId}-seedbox-desktop snapshot --filename={runDir}/proof/nodes/nodes-serving-unreachable/card.aria.yml
  $FW exec $RUN --proof nodes/nodes-serving-unreachable -- $PW cli -s={runId}-seedbox-desktop screenshot --filename={runDir}/proof/nodes/nodes-serving-unreachable/card.png
  $FW exec $RUN --proof nodes/nodes-serving-unreachable -- curl -sS -H 'X-Api-Key: {apiKey}' '{nodes.seedbox.url}/api/v1/system'
  $FW exec $RUN --proof nodes/nodes-serving-unreachable -- curl -sS '{nodes.seedbox.url}/api/v1/serving'
  $FW node start $RUN home
  $FW exec $RUN --proof nodes/nodes-serving-unreachable -- $PW cli -s={runId}-seedbox-desktop run-code 'async page => { await page.getByRole("heading", { name: "Library" }).waitFor({ timeout: 30000 }); return page.url(); }'
  $FW exec $RUN -- $PW cli -s={runId}-seedbox-desktop close
  ```

## Gotchas

- A Worker chosen in the picker shows its own facts and settings through the Controller; an offline Worker's sections read `<name> is offline. Its settings and logs come back when it reconnects.` On the seedbox's own URL a Worker shows only the Worker page.
- Ports are drawn at launch, so compare the page with the manifest, never with a remembered value. fakeworld pins each listen key to the port it bound, so `Port` reads that port with `Reset to default`, and a saved or reset port moves the Node off the manifest's port at its next restart. Put the port back before any restart, and restart last.
- A choice such as `Level` saves as soon as it is chosen; a number field shows `Save` once it differs.
- Nothing logs at Debug on an idle `home`, so a recipe makes a Debug line itself: bytes that aren't TLS on the link port log `link handshake failed`.
- `fakeworld` starts each Node on port 0 and pins the bound port later, so `POST .../restart` can answer a `nextPort` equal to its `boundPort`; the page treats that as no move. A build before the bound-port fix shows `Restart to use port <port>` with `SlipStream listens on 0 until it restarts.` on every Node from launch.
- A Node's environment can't be set in a Fake world, so a locked key comes from the data dir's `.env`, an operator's hand edit. A `SLIPSTREAM_` variable in the CLI's own environment locks nothing on a running Node, which checks its own. An open page doesn't learn of a `.env` edit, so reload it.
- `link/home` is the one front every peer dials, so a link fault can't take `@seedbox`.
- `Open SlipStream on port <port>` is a button, not a link, and the page doesn't follow a move by itself.
- After a Worker restart, the page keeps reading `<name> is restarting…` through any later outage of that Worker until it reloads, so reload before checking the offline sentence.
- Port fallback needs the HTTP port held before a Node starts: stop seedbox, hold `{nodes.seedbox.listeners.http}` with another program, start seedbox, and read its Overview through the picker; free the port and restart seedbox after.
- A Serving Worker's address is its own origin: a session or a passkey made at `{nodes.home.browserUrl}` doesn't carry to `{nodes.seedbox.browserUrl}`, so `bin/sign-in.ts`, which signs in at the Controller, can't open a session there. Sign in through the page instead.
- Over plain HTTP only a `.localhost` name can hold a passkey, so a Serving Worker reached by IP address offers none, as any Node does.
