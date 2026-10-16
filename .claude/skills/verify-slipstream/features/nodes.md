# Nodes

Nodes is where an Admin runs a Node in the cluster. `/nodes` lands on the Node the browser is talking to, at `/nodes/<id>`, titled with the Node's name, and a Node's pages are its sections: Overview, Web and API, Logs and Log settings. The Overview lists the Node's build version, when it started, its data folder and the addresses of its web, torrent and link listeners and its control socket, for any Node through the per-Node API. Web and API and Log settings edit the Node's own `config.yaml`, one field at a time. Logs streams the Node's log live, and Download logs saves its log folder as a zip. Restart restarts any Node, and the browser's own Node from the account menu, More and the palette too. `GET /api/v1/system` returns the facts of the Node that answers, plus the build stamp.

## Sub-features

- `nodes-overview` `/nodes` and the old `/system` land on `/nodes/<id>` for the Node that served the page, whose Overview lists Version, Running since, Data folder, Web and API, Torrents, Node link and Control socket, and the data folder sits in the run's own scratch dir, not a developer database (P-255).
- `nodes-sections` in the desktop shell the page tabs, a nav named for the Node, link Overview, Web and API, Logs and Log settings and mark the current one; on phone the Node's page, titled `Nodes`, lists the other sections, and each is a pushed page whose back link reads `Nodes`.
- `nodes-picker` with a Worker paired, the `Node` select lists every Node with an online, offline or other-version dot, and choosing one keeps the section.
- `nodes-system-api` `GET /api/v1/system` with the API key returns the PID, the data dir, the listeners and `build` with `stamped`, `commit` and `fingerprint`.
- `nodes-web-settings` Web and API shows `Port` with `Default 8484.` and `Applies after restart.`, and `Ports to try`; a saved port toasts `Web and API settings saved` and shows the warning line `Restart to use port <port>` with `Restart now`, a port out of range answers 422 with `Between 1 and 65535.`, and a port an environment variable sets is locked with `Set by SLIPSTREAM_HTTP_LISTEN` (P-244).
- `nodes-port-fallback` a Node whose HTTP port another program holds at start binds one of the next ports, and its Overview reads `Port <configured> was taken, so SlipStream is on <bound>.` with `Change port` (P-246).
- `nodes-log-settings` Log settings shows `Level`, `Rotate at`, `Files kept` and `Days kept` (0 reads `No limit`), `Compress rotated files` and `Lines kept for the viewer`; choosing a level saves it at once and toasts `Log settings saved`, the Node logs at the new level with no restart, and the `Log folder` reads the Node's log dir with `Copy` (P-247, P-248).
- `nodes-logs-viewer` Logs shows the Node's recent lines, then each new one: the time, a level chip, the subsystem and the message; it filters by minimum level, subsystem and search, `Pause` holds the list and reads `Paused, <n> new` until `Resume`, `Clear` empties the view only, a row expands to every attribute with `Copy as JSON`, scrolling up reads `Not following,` with `Jump to latest`, and a restarted Node draws `SlipStream restarted at <time>` (P-249).
- `nodes-logs-download` `Download logs` saves a zip of the Node's log folder holding `slipstream.log`, and a Node that doesn't answer toasts `Couldn't download the logs.` instead (P-250).
- `nodes-restart` the account menu's, More's and the palette's `Restart SlipStream…` and the Nodes header's `Restart` ask `Restart SlipStream?` (or `Restart <name>?` for a Worker); the browser's own Node shows `Restarting SlipStream…` over the page, and the page reloads onto the same address once a later `startedAt` answers, while a Worker's sections read `<name> is restarting…` until its link is up again (P-010).

## How to get to it (user POV)

- Desktop shell: the sidebar's `Nodes`, below the separator and above `Tasks`, which opens `/nodes` and lands on `/nodes/<id>`; the page tabs `Overview`, `Web and API`, `Logs` and `Log settings` sit under the Node's name, and `Restart` is the page's action in the top bar. The `Nodes` row stays current on every Node page. The account menu holds `Restart SlipStream…` between the Startup sequence switch and `Sign out`.
- Phone shell: the `More` tab, then `Nodes`, listed above `Tasks`, then a section's row; the Node's page carries `Restart`. More holds `Restart SlipStream…` above `Sign out`. On Logs the filters sit behind `Filter`.
- Both shells: the palette's `Go to Nodes`, `Go to Overview`, `Go to Web and API`, `Go to Logs` and `Go to Log settings` for the browser's own Node, and `Restart SlipStream…`.
- API (Admin): `GET /api/v1/system`, `GET /api/v1/nodes` for the Node ids, then per Node `GET /api/v1/nodes/{id}/system`, `GET` and `PATCH /api/v1/nodes/{id}/config` (a key to value map, `null` resets), `POST /api/v1/nodes/{id}/restart` (202 with `startedAt` and `boundPort`), `GET /api/v1/nodes/{id}/logs/stream` (server-sent `log.record` and `log.reset`, resuming with `?after=<seq>`) and `GET /api/v1/nodes/{id}/logs/archive`; the live event `node.settings-changed`.
- Host CLI: `slipstream version` prints the same build stamp; `slipstream config set --data-dir <dir> <key> <value>`, `config unset` and `restart --data-dir <dir>`.

## Driving it with fakeworld and playwright cli

Preconditions:

- The baseline: a `home` run, doctor passing, `{runId}-admin-desktop` signed in.

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

- **Download the logs.** Choose `Download logs` and keep the download: the zip lists `slipstream.log`. The API's archive is the same zip.

  ```sh
  $FW exec $RUN --proof nodes/nodes-logs-download -- $PW cli -s={runId}-admin-desktop run-code 'async page => { const saved = page.waitForEvent("download"); await page.getByRole("button", { name: "Download logs" }).click(); await (await saved).saveAs("{runDir}/proof/nodes/nodes-logs-download/logs.zip"); }'
  $FW exec $RUN --proof nodes/nodes-logs-download -- unzip -l {runDir}/proof/nodes/nodes-logs-download/logs.zip
  $FW exec $RUN --proof nodes/nodes-logs-download -- curl -sS -o {runDir}/proof/nodes/nodes-logs-download/api.zip -w '%{http_code} %{content_type}\n' -H 'X-Api-Key: {apiKey}' '{nodes.home.url}/api/v1/nodes/{nodes.home.nodeId}/logs/archive'
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

## Gotchas

- A Worker chosen in the picker shows its own facts and settings through the Controller; an offline Worker's sections read `<name> is offline. Its settings and logs come back when it reconnects.` On the seedbox's own URL a Worker shows only the Worker page.
- Ports are drawn at launch, so compare the page with the manifest, never with a remembered value. fakeworld pins each listen key to the port it bound, so `Port` reads that port with `Reset to default`, and a saved or reset port moves the Node off the manifest's port at its next restart. Put the port back before any restart, and restart last.
- A choice such as `Level` saves as soon as it is chosen; a number field shows `Save` once it differs.
- Nothing logs at Debug on an idle `home`, so a recipe makes a Debug line itself: bytes that aren't TLS on the link port log `link handshake failed`.
- `fakeworld` starts each Node on port 0 and pins the bound port later, so `POST .../restart` can answer a `nextPort` equal to its `boundPort`; the page treats that as no move.
- Port fallback needs the HTTP port held before a Node starts: stop seedbox, hold `{nodes.seedbox.listeners.http}` with another program, start seedbox, and read its Overview through the picker; free the port and restart seedbox after.
