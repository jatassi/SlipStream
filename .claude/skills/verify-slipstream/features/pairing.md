# Pairing

An Admin issues a Join Code on the Controller, and an Unclaimed Node joins with it from the host CLI. The joined Node becomes a Worker: its own page sends people to the Controller, and the Controller lists it with its Node link up and no Roles yet.

## Sub-features

- `pairing-roster` the Controller lists every paired Node with its link state and Roles over `GET /api/v1/nodes`: a Worker reads `up` while its Sync session is attached, and once it stops, `down` since its last Sync frame.
- `pairing-wipe` a stopped, wiped seedbox starts Unclaimed on its first address: `{"claim":"unclaimed"}`.
- `pairing-join-code` the Controller issues a single-use Join Code, `SLP1-…`, over `POST /api/v1/nodes/join-codes`.
- `pairing-join` `slipstream join` pairs the stopped Node and prints `joined home as seedbox-2 (node <id>); link up to <address>`.
- `pairing-worker-page` the Worker's own URL lands on `/worker`, headed `This Node is a Worker. Manage it from its Controller.`

## How to get to it (user POV)

- Host CLI on the joining machine: `slipstream join --data-dir <dir> <code>` while that Node is stopped.
- API on the Controller: `POST /api/v1/nodes/join-codes`, `GET /api/v1/nodes`. The skeleton has no Nodes screen.
- API on an Unclaimed Node: `POST /api/v1/setup/worker` with the Join Code and the Node's Setup code.
- Desktop and phone: the Worker's own URL, which shows only the Worker page.

## Driving it with fakeworld and the host CLI

Preconditions:

- A `home` run with doctor passing. Run this last or on its own run: it replaces the seedbox's Pairing, so the old row stays `down`.

Steps:

- **Read the roster.** `home` reads link `local` and `seedbox` reads `up` with Roles `searching` and `downloading`.

  ```sh
  $FW exec $RUN --proof pairing/pairing-roster -- curl -sS -H 'X-Api-Key: {apiKey}' '{nodes.home.url}/api/v1/nodes'
  ```

- **Stop the seedbox.** It prints the Node's manifest entry with `"state":"stopped"`. The roster then reads `seedbox` `down`, with `since` at or before the stop: its last Sync frame, not the moment the Controller noticed.

  ```sh
  $FW node stop $RUN seedbox
  $FW exec $RUN --proof pairing/pairing-roster -- curl -sS -H 'X-Api-Key: {apiKey}' '{nodes.home.url}/api/v1/nodes'
  ```

- **Wipe the seedbox.** Empty its data dir. It prints the Node's manifest entry with `"state":"stopped"`.

  ```sh
  $FW node wipe $RUN seedbox
  ```

- **Issue a Join Code.** The Controller answers 201 with `{"code":"SLP1-…"}`.

  ```sh
  CODE=$($FW exec $RUN --proof pairing/pairing-join-code -- curl -sS -X POST -H 'X-Api-Key: {apiKey}' '{nodes.home.url}/api/v1/nodes/join-codes' | jq -r .code)
  ```

- **Join from the host CLI.** The command exits 0 and prints `joined home as seedbox-2 (node <id>); link up to 127.0.0.1:<port>`, the link front.

  ```sh
  $FW exec $RUN --proof pairing/pairing-join -- {runDir}/bin/slipstream join --data-dir {nodes.seedbox.dataDir} "$CODE"
  $FW node start $RUN seedbox
  $FW exec $RUN --proof pairing/pairing-join -- curl -sS '{nodes.seedbox.url}/api/v1/status'
  $FW exec $RUN --proof pairing/pairing-join -- curl -sS -H 'X-Api-Key: {apiKey}' '{nodes.home.url}/api/v1/nodes'
  ```

  The status reads `{"claim":"worker"}`. The roster lists `seedbox` `down` with its old Roles and `seedbox-2` `up` with none.

- **Open the Worker page.** The URL ends `/worker` and the heading reads `This Node is a Worker. Manage it from its Controller.`

  ```sh
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop goto '{nodes.seedbox.browserUrl}/'
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop run-code 'async page => { await page.waitForURL(/\/worker$/); }'
  $FW exec $RUN --proof pairing/pairing-worker-page -- $PW cli -s={runId}-admin-desktop screenshot --filename={runDir}/proof/pairing/pairing-worker-page/worker.png
  ```

## Gotchas

- A wiped Node has a new key, so the Controller pairs it as a new Node named `seedbox-2`, and the old row stays down with its Roles. Doctor's check 3 then fails with `node seedbox's link is down, not up`, which is expected after this recipe.
- `slipstream join` needs the Node stopped: it writes the data dir the running Node holds.
- The Join Code is single-use and short-lived. Mint it right before the join.
- A wiped Node logs a new Setup code in `nodes/seedbox.log`, which the API path `POST /api/v1/setup/worker` takes in `X-Setup-Code`.
