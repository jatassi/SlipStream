# Pairing

An Admin issues a Join Code on the Controller, and an Unclaimed Node joins with it from the host CLI. The joined Node becomes a Worker: its own page sends people to the Controller, and the Controller lists it with its Node link up and no Roles yet. A Node that can't reach the Controller prints its Node address and waits; the Admin pastes it on the Controller, which dials the Node, pairs it over that session and keeps dialing it from then on.

## Sub-features

- `pairing-roster` the Controller lists every paired Node with its link state and Roles over `GET /api/v1/nodes`: a Worker reads `up` while its Sync session is attached, and once it stops, `down` since its last Sync frame.
- `pairing-wipe` a stopped, wiped seedbox starts Unclaimed on its first address: `{"claim":"unclaimed"}`.
- `pairing-join-code` the Controller issues a single-use Join Code, `SLP1-…`, over `POST /api/v1/join-codes`.
- `pairing-join` `slipstream join` pairs the stopped Node and prints `joined home as seedbox-2 (node <id>); link up to <address>`.
- `pairing-join-codes` `GET /api/v1/join-codes` lists each open code's `id`, `createdAt` and `expiresAt`, never its text; `GET /api/v1/join-codes/{id}` reads it `open`, `used` with `nodeId`, or `expired`; `DELETE` answers 204, and the code then reads 404 `join_code_not_found`.
- `pairing-reverse-dial` on `cgnat-home`, `slipstream join` prints `Can't reach the Controller at <address>.`, `Paste this Node address into the Add a Node dialog on the Controller:`, the indented `SLN1-…` and `Waiting for the Controller to dial until <HH:MM>. Ctrl-C stops.`; `POST /api/v1/join-codes/{id}/dial` with that `nodeAddress` answers 202, the command exits 0 with the joined line, and the roster lists `seedbox` `up` with `dial` `{"by":"controller","controllerDials":["<seedbox's link front>"]}`, also after both Nodes restart.
- `pairing-setup-pending` on `cgnat-home`, `POST /api/v1/setup/worker` answers 202 with `state` `awaiting-dial`, `nodeAddress` and `until`; `POST /api/v1/setup/controller` then answers 409 `join_pending`; once the Controller dials, `GET /api/v1/setup/worker` reads `state` `joined` with `nodeId`.
- `pairing-node-address` `slipstream node-address --data-dir <dir>` prints the Node's `SLN1-…` from its config and key, and `slipstream join --node-port <port>` writes `link.listen` at that port before it joins.
- `pairing-worker-page` the Worker's own URL lands on `/worker`, headed `This Node is a Worker. Manage it from its Controller.` over its `Name`, its `Controller` and its `Key`, the fingerprint of its Node key.

## How to get to it (user POV)

- Host CLI on the joining machine: `slipstream join --data-dir <dir> [--node-port <port>] <code>`, stopped or running; `slipstream node-address --data-dir <dir>`.
- API on the Controller: `POST` and `GET /api/v1/join-codes`, `GET` and `DELETE /api/v1/join-codes/{id}`, `POST /api/v1/join-codes/{id}/dial`, `PUT /api/v1/nodes/{id}/dial` (`{"nodeAddress":"SLN1-…"}`, or `{"addresses":[]}` to stop dialing), `GET /api/v1/nodes`. The skeleton has no Nodes screen.
- API on an Unclaimed Node: `POST /api/v1/setup/worker` with the Join Code and the Node's Setup code, then `GET /api/v1/setup/worker` while it awaits the dial.
- Desktop and phone: the Worker's own URL, which shows only the Worker page.

## Driving it with fakeworld and the host CLI

Preconditions:

- A `home` run with doctor passing for the steps up to the Worker page. Run them last or on their own run: they replace the seedbox's Pairing, so the old row stays `down`.
- A separate `cgnat-home` run (`--scenario cgnat-home`) for the steps from **Join behind CGNAT**: home's link front refuses every dial, and seedbox starts Unclaimed and running.

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
  CODE=$($FW exec $RUN --proof pairing/pairing-join-code -- curl -sS -X POST -H 'X-Api-Key: {apiKey}' '{nodes.home.url}/api/v1/join-codes' | jq -r .code)
  ```

- **Join from the host CLI.** The command exits 0 and prints `joined home as seedbox-2 (node <id>); link up to 127.0.0.1:<port>`, the link front.

  ```sh
  $FW exec $RUN --proof pairing/pairing-join -- {runDir}/bin/slipstream join --data-dir {nodes.seedbox.dataDir} "$CODE"
  $FW node start $RUN seedbox
  $FW exec $RUN --proof pairing/pairing-join -- curl -sS '{nodes.seedbox.url}/api/v1/status'
  $FW exec $RUN --proof pairing/pairing-join -- curl -sS -H 'X-Api-Key: {apiKey}' '{nodes.home.url}/api/v1/nodes'
  ```

  The status reads `{"claim":"worker"}`. The roster lists `seedbox` `down` with its old Roles and `seedbox-2` `up` with none.

- **Open the Worker page.** The URL ends `/worker`, the heading reads `This Node is a Worker. Manage it from its Controller.`, and the rows read the Worker's `Name`, `Controller` `home` and its `Key`.

  ```sh
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop goto '{nodes.seedbox.browserUrl}/'
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop run-code 'async page => { await page.waitForURL(/\/worker$/); }'
  $FW exec $RUN --proof pairing/pairing-worker-page -- $PW cli -s={runId}-admin-desktop screenshot --filename={runDir}/proof/pairing/pairing-worker-page/worker.png
  $FW exec $RUN --proof pairing/pairing-worker-page -- $PW cli -s={runId}-admin-desktop snapshot 'getByRole("main")' --filename={runDir}/proof/pairing/pairing-worker-page/worker.aria.yml
  ```

- **List and revoke a Join Code.** Mint one, list it, then revoke it. The list holds its `id` without its text; after the 204 the code reads 404 `join_code_not_found`.

  ```sh
  ID=$($FW exec $RUN --proof pairing/pairing-join-codes -- curl -sS -X POST -H 'X-Api-Key: {apiKey}' '{nodes.home.url}/api/v1/join-codes' | jq -r .id)
  $FW exec $RUN --proof pairing/pairing-join-codes -- curl -sS -H 'X-Api-Key: {apiKey}' '{nodes.home.url}/api/v1/join-codes'
  $FW exec $RUN --proof pairing/pairing-join-codes -- curl -sS -o /dev/null -w '%{{http_code}\n' -X DELETE -H 'X-Api-Key: {apiKey}' "{nodes.home.url}/api/v1/join-codes/$ID"
  $FW exec $RUN --proof pairing/pairing-join-codes -- curl -sS -H 'X-Api-Key: {apiKey}' "{nodes.home.url}/api/v1/join-codes/$ID"
  ```

- **Join behind CGNAT.** On the `cgnat-home` run, mint a code and start `slipstream join` for the running seedbox in the background. Its output holds the four waiting lines, the third the indented Node address.

  ```sh
  MINTED=$($FW exec $RUN --proof pairing/pairing-reverse-dial -- curl -sS -X POST -H 'X-Api-Key: {apiKey}' '{nodes.home.url}/api/v1/join-codes')
  $FW exec $RUN --proof pairing/pairing-reverse-dial -- {runDir}/bin/slipstream join --data-dir {nodes.seedbox.dataDir} "$(jq -r .code <<<"$MINTED")" > $RUN/join.out &
  until grep -q '^  SLN1-' $RUN/join.out; do sleep 1; done
  cat $RUN/join.out
  ```

- **Dial the Node address.** Paste it into `dialForCode`. It answers 202, the join exits 0 with `joined home as seedbox (node <id>); link up to <address>`, and the roster lists seedbox `up` with `dial.by` `controller` and `dial.controllerDials` its link front.

  ```sh
  $FW exec $RUN --proof pairing/pairing-reverse-dial -- curl -sS -w '%{{http_code}\n' -X POST -H 'X-Api-Key: {apiKey}' -H 'Content-Type: application/json' -d "{\"nodeAddress\":\"$(sed -n 's/^  //p' $RUN/join.out)\"}" "{nodes.home.url}/api/v1/join-codes/$(jq -r .id <<<"$MINTED")/dial"
  wait
  cat $RUN/join.out
  $FW exec $RUN --proof pairing/pairing-reverse-dial -- curl -sS -H 'X-Api-Key: {apiKey}' '{nodes.home.url}/api/v1/nodes'
  ```

- **Restart both Nodes.** The roster lists seedbox `up` again, dialed by the controller.

  ```sh
  $FW node restart $RUN home
  $FW node restart $RUN seedbox
  $FW exec $RUN --proof pairing/pairing-reverse-dial -- curl -sS -H 'X-Api-Key: {apiKey}' '{nodes.home.url}/api/v1/nodes'
  ```

- **Read the Node address from the host CLI.** On a fresh `cgnat-home` run, `slipstream node-address` prints seedbox's `SLN1-…`, naming its link front.

  ```sh
  $FW exec $RUN --proof pairing/pairing-node-address -- {runDir}/bin/slipstream node-address --data-dir {nodes.seedbox.dataDir}
  ```

- **Join from the first-run page's API.** On the same fresh run, claim seedbox as a Worker with the Setup code its log holds. It answers 202 `awaiting-dial`; a Controller claim answers 409 `join_pending`; after `dialForCode` with the `nodeAddress`, as in **Dial the Node address**, `GET /api/v1/setup/worker` reads `joined`.

  ```sh
  SETUP=$($FW exec $RUN -- jq -r 'select(.msg == "setup code") | .code' '{nodes.seedbox.log}' | tail -1)
  MINTED=$($FW exec $RUN -- curl -sS -X POST -H 'X-Api-Key: {apiKey}' '{nodes.home.url}/api/v1/join-codes')
  $FW exec $RUN --proof pairing/pairing-setup-pending -- curl -sS -X POST -H "X-Setup-Code: $SETUP" -H 'Content-Type: application/json' -d "{\"code\":\"$(jq -r .code <<<"$MINTED")\"}" '{nodes.seedbox.url}/api/v1/setup/worker'
  $FW exec $RUN --proof pairing/pairing-setup-pending -- curl -sS -X POST -H "X-Setup-Code: $SETUP" -H 'Content-Type: application/json' -d '{"email":"olu@example.test","displayName":"Olu Bello","pin":"1357"}' '{nodes.seedbox.url}/api/v1/setup/controller'
  $FW exec $RUN --proof pairing/pairing-setup-pending -- curl -sS -H "X-Setup-Code: $SETUP" '{nodes.seedbox.url}/api/v1/setup/worker'
  ```

## Gotchas

- A wiped Node has a new key, so the Controller pairs it as a new Node named `seedbox-2`, and the old row stays down with its Roles. Doctor's check 3 then fails with `node seedbox's link is down, not up`, which is expected after this recipe.
- `slipstream join` needs the Node stopped: it writes the data dir the running Node holds.
- The Join Code is single-use and short-lived. Mint it right before the join.
- A wiped Node logs a new Setup code in `nodes/seedbox.log`, which the API path `POST /api/v1/setup/worker` takes in `X-Setup-Code`.
- `slipstream join` waits for the dial in the foreground until `timings.link.await_dial` (10 minutes) passes. Background it, as above, or the recipe blocks.
- `dialForCode` answers 202 before the Node pairs. A paste that names no waiting Node reads `dialing.unreachable` `true` on `GET /api/v1/join-codes/{id}` once `timings.nodes.dial_report` (30 seconds) passes.
- On `cgnat-home`, seedbox's own dials to home keep failing at home's link front, by design. Doctor lists `home-behind-cgnat` as an active fault, not a failure.
- `$FW exec` reads `{...}` as a manifest token, so curl's write-out is spelled `%{{http_code}`.
