# Nodes

Nodes is where an Admin runs a Node in the cluster. `/nodes` lands on the Node the browser is talking to, at `/nodes/<id>`, titled with the Node's name, and a Node's pages are its sections: Overview, Web and API, Logs and Log settings. The Overview lists the Node's build version, when it started, its data folder and the addresses of its web, torrent and link listeners and its control socket. `GET /api/v1/system` returns the same facts plus the build stamp.

## Sub-features

- `nodes-overview` `/nodes` and the old `/system` land on `/nodes/<id>` for the Node that served the page, whose Overview lists Version, Running since, Data folder, Web and API, Torrents, Node link and Control socket, and the data folder sits in the run's own scratch dir, not a developer database (P-255).
- `nodes-sections` in the desktop shell the page tabs, a nav named for the Node, link Overview, Web and API, Logs and Log settings and mark the current one; on phone the Node's page, titled `Nodes`, lists the other sections, and each is a pushed page whose back link reads `Nodes`.
- `nodes-picker` with a Worker paired, the `Node` select lists every Node with an online, offline or other-version dot, and choosing one keeps the section.
- `nodes-system-api` `GET /api/v1/system` with the API key returns the PID, the data dir, the listeners and `build` with `stamped`, `commit` and `fingerprint`.

## How to get to it (user POV)

- Desktop shell: the sidebar's `Nodes`, below the separator, which opens `/nodes` and lands on `/nodes/<id>`; the page tabs `Overview`, `Web and API`, `Logs` and `Log settings` sit under the Node's name. The `Nodes` row stays current on every Node page.
- Phone shell: the `More` tab, then `Nodes`, then a section's row.
- Both shells: the palette's `Go to Nodes`, and `Go to Overview`, `Go to Web and API`, `Go to Logs` and `Go to Log settings` for the browser's own Node.
- API: `GET /api/v1/system`, and `GET /api/v1/nodes` for the Node ids.
- Host CLI: `slipstream version` prints the same build stamp.

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

## Gotchas

- The Overview's facts are the Node that served the page. A Worker chosen in the picker shows only a sentence until the per-Node API lands, and on the seedbox's own URL a Worker shows only the Worker page, so read a Worker's facts with `{nodes.seedbox.url}/api/v1/system`.
- Ports are drawn at launch, so compare the page with the manifest, never with a remembered value.
- Web and API, Logs and Log settings show only their titles for now.
