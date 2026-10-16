# This Node

This Node shows the Node the browser is talking to: its build version, data dir and the addresses of its HTTP, torrent and link listeners and its control socket. `GET /api/v1/system` returns the same facts plus the build stamp.

## Sub-features

- `this-node-page` the This Node page lists Version, Data dir, HTTP, Torrent, Link and Control socket, and the data dir sits in the run's own scratch dir, not a developer database (P-255).
- `this-node-api` `GET /api/v1/system` with the API key returns the PID, the data dir, the listeners and `build` with `stamped`, `commit` and `fingerprint`.

## How to get to it (user POV)

- Desktop shell: the sidebar's `This Node`, below the separator and above `Tasks`, which opens `/system`.
- Phone shell: the `More` tab, then `This Node`, listed above `Tasks`.
- Both shells: the palette's `Go to This Node`.
- API: `GET /api/v1/system`.
- Host CLI: `slipstream version` prints the same build stamp.

## Driving it with fakeworld and playwright cli

Preconditions:

- The baseline: a `home` run, doctor passing, `{runId}-admin-desktop` signed in.

Steps:

- **Open This Node.** Choose `This Node` in the sidebar. The page is headed `This Node`, Version reads the binary's short commit, and Data dir reads the manifest's `{nodes.home.dataDir}`.

  ```sh
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop click 'getByRole("navigation", { name: "Main" }).getByRole("link", { name: "This Node" })'
  $FW exec $RUN --proof this-node/this-node-page -- $PW cli -s={runId}-admin-desktop snapshot --filename={runDir}/proof/this-node/this-node-page/system.aria.yml
  $FW exec $RUN --proof this-node/this-node-page -- $PW cli -s={runId}-admin-desktop screenshot --filename={runDir}/proof/this-node/this-node-page/system.png
  ```

- **Read it over the API and the host CLI.** The API's `dataDir` and listeners equal the page's rows, and `slipstream version` prints the same commit.

  ```sh
  $FW exec $RUN --proof this-node/this-node-api -- curl -sS -H 'X-Api-Key: {apiKey}' '{nodes.home.url}/api/v1/system'
  $FW exec $RUN --proof this-node/this-node-api -- {runDir}/bin/slipstream version
  ```

## Gotchas

- The page shows the Node that served it. On the seedbox's URL a Worker shows only the Worker page, so read a Worker's facts with `{nodes.seedbox.url}/api/v1/system`.
- Ports are drawn at launch, so compare the page with the manifest, never with a remembered value.
