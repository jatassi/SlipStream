# Pairing

An Admin issues a Join Code on the Controller, and an Unclaimed Node joins with it from the host CLI. The joined Node becomes a Worker: its own page sends people to the Controller, and the Controller lists it with its Node link up and no Roles yet. A Node that can't reach the Controller prints its Node address and waits; the Admin pastes it on the Controller, which dials the Node, pairs it over that session and keeps dialing it from then on.

## Sub-features

- `pairing-roster` the Controller lists every paired Node with its link state and Roles over `GET /api/v1/nodes`: a Worker reads `up` while its Sync session is attached, and once it stops, `down` since its last Sync frame.
- `pairing-wipe` a stopped, wiped seedbox starts Unclaimed on its first address: `{"claim":"unclaimed"}`.
- `pairing-join-code` the Controller issues a single-use Join Code, `SLP1-…`, over `POST /api/v1/join-codes`.
- `pairing-join` `slipstream join` pairs the stopped Node and prints `joined home as seedbox-2 (node <id>), linked to <address>`.
- `pairing-replace` after a wiped seedbox joins again as `seedbox-2`, `DELETE /api/v1/nodes/{id}` on the old `seedbox` row, down and holding nothing, answers 204 and hands Searching and Downloading back to home; `PATCH /api/v1/nodes/{id}` renames `seedbox-2` to `seedbox`, so the Roster reads home and `seedbox` `Online · seedbox dials home` with `No Roles yet`, and the Worker page's `Name` reads `seedbox`.
- `pairing-join-codes` `GET /api/v1/join-codes` lists each open code's `id`, `createdAt` and `expiresAt`, never its text; `GET /api/v1/join-codes/{id}` reads it `open`, `used` with `nodeId`, or `expired`, with `dialing{address, addresses, since, unreachable}` once a Node address is pasted; `DELETE` answers 204, and the code then reads 404 `join_code_not_found`.
- `pairing-reverse-dial` on `cgnat-home`, `slipstream join` prints `Can't reach the Controller at <address>.`, `Paste this Node address into the Add a Node dialog on the Controller:`, the indented `SLN1-…` and `Waiting for the Controller to dial until <HH:MM>. Press Ctrl-C to stop.`; `POST /api/v1/join-codes/{id}/dial` with that `nodeAddress` answers 202, the command exits 0 with the joined line, and the roster lists `seedbox` `up` with `dial` `{"by":"controller","controllerDials":["<seedbox's link front>"]}`, also after both Nodes restart.
- `pairing-setup-pending` on `cgnat-home`, `POST /api/v1/setup/worker` answers 202 with `state` `awaiting-dial`, `nodeAddress` and `until`; `POST /api/v1/setup/controller` then answers 409 `join_pending`; once the Controller dials, `GET /api/v1/setup/worker` reads `state` `joined` with `nodeId`.
- `pairing-add-dialog` **Add a Node** on the Nodes landing opens the `Add a Node` dialog at `?add=<code id>`, with the `Join Code` `SLP1-…`, `Expires in 59:42. It pairs one Node.` ticking, Copy, the three ways (`First-run page.`, `slipstream join <code>`, `SLIPSTREAM_JOIN_CODE=<code>`), `Waiting for the new Node to pair…` and the `Open Join Codes` list; after a reload it reads `A Join Code shows only when it is made. This one still works for <left>.`
- `pairing-add-dialog-paired` once a Node joins with the dialog's code, the dialog reads `seedbox paired.` with its `Key` (the Worker's fingerprint) and `Reached by` `seedbox dials home`; **Choose its Roles** shows the Role Groups with Searching and Downloading ticked, and **Review** then **Apply** lands on seedbox's Roles section with both held.
- `pairing-add-dialog-reverse-dial` on `cgnat-home`, **The new Node can't reach home?** opens a `Node address` field; **home dials it** with the `nodeAddress` from `POST /api/v1/setup/worker` reads `home is dialing the new Node at <address>…` until it pairs, then `Reached by` `home dials seedbox`. An address nothing answers at reads `home can't reach the new Node at <address>. Check that the port is open.` after `timings.nodes.dial_report`.
- `pairing-add-dialog-revoke` each open code's row has **Revoke**: another code's row (`Made <time>`) leaves the list and can no longer pair (`POST /api/v1/setup/worker` answers 403 `join_code_unknown`), and revoking `This code` reads `This Join Code was revoked, so no Node can use it.` with **New Join Code**.
- `pairing-node-address` `slipstream node-address --data-dir <dir>` prints the Node's `SLN1-…` from its config and key, and `slipstream join --node-port <port>` writes `link.listen` at that port before it joins.
- `pairing-worker-page` the Worker's own URL lands on `/worker`, headed `This Node is a Worker. Manage it from its Controller.` over its `Name`, its `Controller` and its `Key`, the fingerprint of its Node key.
- `pairing-rename` `PATCH /api/v1/nodes/{id}` with `{"name":"attic"}` answers the Node under its new name once the Worker has it, and the Worker page's `Name` row reads `attic`; a name another Node has answers 409 `node_name_taken`, and one ending in a space 422 `node_name_invalid`.
- `pairing-remove-blocked` `DELETE /api/v1/nodes/{id}` on home answers 409 `node_is_controller`, and on seedbox while it seeds an imported Grab 409 `node_holds_work`; each carries `outcome`, whose check reads `home is the Controller, so it can't be removed.` or `seedbox still holds 1 torrent (3.5 MB) for Downloading. Untick its Roles and remove it when they finish, or remove it now and write them off.`
- `pairing-remove` `DELETE /api/v1/nodes/{id}?writeOff=true` answers 204 and the roster lists only home, holding Downloading. seedbox lets its torrent go, announcing `stopped`, and restarts Unclaimed: `GET /api/v1/status` reads `"claim":"unclaimed"` with `"removed":{"at":…,"controller":"home"}`, its log holds `This Node was removed from home on <d Mon>. Join with a new Join Code to use it again.`, and the file it downloaded still matches the release.
- `pairing-remove-rejoin` the removed seedbox joins again through `POST /api/v1/setup/worker` with a new Join Code, as a new Node named `seedbox` with a new id and no Roles; with Downloading ticked on it, it runs Downloading and announces no `started` for the old torrent.

## How to get to it (user POV)

- Host CLI on the joining machine: `slipstream join --data-dir <dir> [--node-port <port>] <code>`, stopped or running; `slipstream node-address --data-dir <dir>`.
- API on the Controller: `POST` and `GET /api/v1/join-codes`, `GET` and `DELETE /api/v1/join-codes/{id}`, `POST /api/v1/join-codes/{id}/dial`, `PUT /api/v1/nodes/{id}/dial` (`{"nodeAddress":"SLN1-…"}`, or `{"addresses":[]}` to stop dialing), `GET /api/v1/nodes`, `PATCH /api/v1/nodes/{id}` (`{"name":…}`) and `DELETE /api/v1/nodes/{id}[?writeOff=true]`. The Nodes Roster and a Node's Overview rename and remove a Node too; see nodes.md.
- API on a removed Node: `GET /api/v1/status` carries `removed`, for the first-run page to say who removed it.
- API on an Unclaimed Node: `POST /api/v1/setup/worker` with the Join Code and the Node's Setup code, then `GET /api/v1/setup/worker` while it awaits the dial.
- Desktop and phone: the Worker's own URL, which shows only the Worker page.
- Desktop and phone on the Controller: **Add a Node**, the Nodes landing's action, or the `Add a Node` row under a lone Node on the Roster, or `/nodes?add=<code id>`, which keeps a code's dialog open across a reload.

## Driving it with fakeworld and the host CLI

Preconditions:

- A `home` run with doctor passing for the steps up to the Worker page. Run them last or on their own run: they replace the seedbox's Pairing and hand its Roles back to home.
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

- **Join from the host CLI.** The command exits 0 and prints `joined home as seedbox-2 (node <id>), linked to 127.0.0.1:<port>`, the link front.

  ```sh
  $FW exec $RUN --proof pairing/pairing-join -- {runDir}/bin/slipstream join --data-dir {nodes.seedbox.dataDir} "$CODE"
  $FW node start $RUN seedbox
  $FW exec $RUN --proof pairing/pairing-join -- curl -sS '{nodes.seedbox.url}/api/v1/status'
  $FW exec $RUN --proof pairing/pairing-join -- curl -sS -H 'X-Api-Key: {apiKey}' '{nodes.home.url}/api/v1/nodes'
  ```

  The status reads `{"claim":"worker"}`. The roster lists `seedbox` `down` with its old Roles and `seedbox-2` `up` with none.

- **Replace the old row.** Remove the old `seedbox` row: it holds no work, so it answers 204 with no write-off, and home holds Searching and Downloading again. Rename `seedbox-2` to `seedbox`: the roster lists home and `seedbox` `up` with no Roles.

  ```sh
  $FW exec $RUN --proof pairing/pairing-replace -- curl -sS -w '%{{http_code}\n' -X DELETE -H 'X-Api-Key: {apiKey}' '{nodes.home.url}/api/v1/nodes/{nodes.seedbox.nodeId}'
  NEW=$($FW exec $RUN -- curl -sS -H 'X-Api-Key: {apiKey}' '{nodes.home.url}/api/v1/nodes' | jq -r '.nodes[] | select(.name == "seedbox-2") | .id')
  $FW exec $RUN --proof pairing/pairing-replace -- curl -sS -X PATCH -H 'X-Api-Key: {apiKey}' -H 'Content-Type: application/json' -d '{"name":"seedbox"}' "{nodes.home.url}/api/v1/nodes/$NEW"
  $FW exec $RUN --proof pairing/pairing-replace -- curl -sS -H 'X-Api-Key: {apiKey}' '{nodes.home.url}/api/v1/nodes'
  ```

- **Open the Worker page.** The URL ends `/worker`, the heading reads `This Node is a Worker. Manage it from its Controller.`, and the rows read `Name` `seedbox`, `Controller` `home` and its `Key`.

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

- **Dial the Node address.** Paste it into `dialForCode`. It answers 202, the join exits 0 with `joined home as seedbox (node <id>), linked to <address>`, and the roster lists seedbox `up` with `dial.by` `controller` and `dial.controllerDials` its link front.

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

### Rename and remove

Preconditions:

- A fresh `home` run with doctor passing and the `{runId}-admin-desktop` session signed in, used for nothing else: it grabs The Quiet Orchard and removes seedbox.

Steps:

- **Rename seedbox.** It answers 200 with `name` `attic`, and seedbox's Worker page reads `Name` `attic`. Renaming it `home` answers 409 `node_name_taken` and `attic ` 422 `node_name_invalid`. Rename it back.

  ```sh
  $FW exec $RUN --proof pairing/pairing-rename -- curl -sS -X PATCH -H 'X-Api-Key: {apiKey}' -H 'Content-Type: application/json' -d '{"name":"attic"}' '{nodes.home.url}/api/v1/nodes/{nodes.seedbox.nodeId}'
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop goto '{nodes.seedbox.browserUrl}/worker'
  $FW exec $RUN --proof pairing/pairing-rename -- $PW cli -s={runId}-admin-desktop snapshot 'getByRole("main")' --filename={runDir}/proof/pairing/pairing-rename/worker.aria.yml
  $FW exec $RUN --proof pairing/pairing-rename -- $PW cli -s={runId}-admin-desktop screenshot --filename={runDir}/proof/pairing/pairing-rename/worker.png
  $FW exec $RUN --proof pairing/pairing-rename -- curl -sS -X PATCH -H 'X-Api-Key: {apiKey}' -H 'Content-Type: application/json' -d '{"name":"home"}' '{nodes.home.url}/api/v1/nodes/{nodes.seedbox.nodeId}'
  $FW exec $RUN --proof pairing/pairing-rename -- curl -sS -X PATCH -H 'X-Api-Key: {apiKey}' -H 'Content-Type: application/json' -d '{"name":"attic "}' '{nodes.home.url}/api/v1/nodes/{nodes.seedbox.nodeId}'
  $FW exec $RUN -- curl -sS -X PATCH -H 'X-Api-Key: {apiKey}' -H 'Content-Type: application/json' -d '{"name":"seedbox"}' '{nodes.home.url}/api/v1/nodes/{nodes.seedbox.nodeId}'
  ```

- **Grab a release onto seedbox.** As in [Roles](./roles.md)' **Drain and write off**: add The Quiet Orchard, search it, wait for `imported` and keep its `HASH`.

- **Try to remove home, then seedbox.** Both answer 409 with `outcome` and its check: `node_is_controller` for home, `node_holds_work` for seedbox.

  ```sh
  $FW exec $RUN --proof pairing/pairing-remove-blocked -- curl -sS -X DELETE -H 'X-Api-Key: {apiKey}' '{nodes.home.url}/api/v1/nodes/{nodes.home.nodeId}'
  $FW exec $RUN --proof pairing/pairing-remove-blocked -- curl -sS -X DELETE -H 'X-Api-Key: {apiKey}' '{nodes.home.url}/api/v1/nodes/{nodes.seedbox.nodeId}'
  ```

- **Remove seedbox, writing its work off.** 204. The roster lists only home, holding Downloading. seedbox announces `stopped` for the torrent and restarts Unclaimed: its status reads `removed` by `home`, its log holds the Removed sentence, and the file it downloaded still matches the release.

  ```sh
  $FW exec $RUN --proof pairing/pairing-remove -- curl -sS -o /dev/null -w '%{{http_code}\n' -X DELETE -H 'X-Api-Key: {apiKey}' '{nodes.home.url}/api/v1/nodes/{nodes.seedbox.nodeId}?writeOff=true'
  $FW exec $RUN --proof pairing/pairing-remove -- curl -sS -H 'X-Api-Key: {apiKey}' '{nodes.home.url}/api/v1/nodes'
  $FW exec $RUN -- bash -c 'for i in $(seq 30); do curl -sSf "{nodes.seedbox.url}/api/v1/status" | grep -q "\"removed\"" && exit 0; sleep 1; done; exit 1'
  $FW exec $RUN --proof pairing/pairing-remove -- curl -sS '{nodes.seedbox.url}/api/v1/status'
  $FW exec $RUN --proof pairing/pairing-remove -- grep 'This Node was removed from home' '{nodes.seedbox.log}'
  $FW exec $RUN --proof pairing/pairing-remove -- $FW fs match {runDir} "{nodes.seedbox.dataDir}/downloads/$HASH/The.Quiet.Orchard.2021.1080p.WEB-DL.H264-FKW.mkv" --release The.Quiet.Orchard.2021.1080p.WEB-DL.H264-FKW
  ```

- **Join seedbox again.** With the new Setup code its log holds and a new Join Code, `POST /api/v1/setup/worker` answers 201 with `state` `joined` and a new `nodeId`. The roster lists `seedbox` `up` under that id with no Roles. Tick Searching and Downloading on it: its facts run `downloading`, and the tracker's last seedbox announce is still the `stopped` from the removal.

  ```sh
  SETUP=$($FW exec $RUN -- jq -r 'select(.msg == "setup code") | .code' '{nodes.seedbox.log}' | tail -1)
  CODE=$($FW exec $RUN -- curl -sS -X POST -H 'X-Api-Key: {apiKey}' '{nodes.home.url}/api/v1/join-codes' | jq -r .code)
  NEW=$($FW exec $RUN --proof pairing/pairing-remove-rejoin -- curl -sS -X POST -H "X-Setup-Code: $SETUP" -H 'Content-Type: application/json' -d "{\"code\":\"$CODE\"}" '{nodes.seedbox.url}/api/v1/setup/worker' | jq -r .nodeId)
  $FW exec $RUN --proof pairing/pairing-remove-rejoin -- curl -sS -H 'X-Api-Key: {apiKey}' '{nodes.home.url}/api/v1/nodes'
  BASIS=$($FW exec $RUN -- curl -sS -H 'X-Api-Key: {apiKey}' '{nodes.home.url}/api/v1/nodes/roles' | jq -r .basis)
  $FW exec $RUN --proof pairing/pairing-remove-rejoin -- curl -sS -X PUT -H 'X-Api-Key: {apiKey}' -H 'Content-Type: application/json' --data-binary "{\"basis\":\"$BASIS\",\"set\":[{\"nodeId\":\"$NEW\",\"roles\":[\"searching\",\"downloading\"]}]}" '{nodes.home.url}/api/v1/nodes/roles'
  $FW exec $RUN --proof pairing/pairing-remove-rejoin -- curl -sS -H 'X-Api-Key: {apiKey}' '{nodes.home.url}/api/v1/nodes'
  $FW exec $RUN --proof pairing/pairing-remove-rejoin -- $FW announces list $RUN --node seedbox
  ```

### Add a Node

Preconditions:

- A fresh `cgnat-home` run with doctor passing and the `{runId}-admin-desktop` and `{runId}-admin-phone` sessions signed in.
- `SETUP` holds seedbox's Setup code, read as in **Join from the first-run page's API**. The browser stands in for the first-run page's Join form through `POST /api/v1/setup/worker`.

Steps:

- **Open the dialog.** The code, its countdown, the three ways and the open codes show; desktop and phone.

  ```sh
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop goto '{nodes.home.browserUrl}/nodes'
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop click 'getByRole("button", { name: "Add a Node", exact: true })'
  $FW exec $RUN --proof pairing/pairing-add-dialog -- $PW cli -s={runId}-admin-desktop snapshot 'getByRole("dialog")' --filename={runDir}/proof/pairing/pairing-add-dialog/open.aria.yml
  $FW exec $RUN --proof pairing/pairing-add-dialog -- $PW cli -s={runId}-admin-desktop screenshot --filename={runDir}/proof/pairing/pairing-add-dialog/open.png
  CODE=$(grep -o 'SLP1-[0-9A-Z-]*' $RUN/proof/pairing/pairing-add-dialog/open.aria.yml | head -1)
  ```

  On phone, the same from the Nodes landing; the dialog is a sheet.

- **Reverse dial through the dialog.** Claim seedbox with the code: 202 with `nodeAddress`. Open **The new Node can't reach home?**, paste it and press **home dials it**. The dialog reads `seedbox paired.` and `home dials seedbox`.

  ```sh
  $FW exec $RUN --proof pairing/pairing-add-dialog-reverse-dial -- curl -sS -X POST -H "X-Setup-Code: $SETUP" -H 'Content-Type: application/json' -d "{\"code\":\"$CODE\"}" '{nodes.seedbox.url}/api/v1/setup/worker'
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop click 'getByRole("dialog").getByRole("button", { name: "The new Node can'"'"'t reach home?" })'
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop fill 'getByRole("dialog").getByRole("textbox", { name: "Node address" })' "$ADDRESS"
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop click 'getByRole("dialog").getByRole("button", { name: "home dials it" })'
  $FW exec $RUN --proof pairing/pairing-add-dialog-paired -- $PW cli -s={runId}-admin-desktop snapshot 'getByRole("dialog")' --filename={runDir}/proof/pairing/pairing-add-dialog-paired/key-check.aria.yml
  ```

  `ADDRESS` is the `nodeAddress` of the 202. With the fault cleared (`$FW fault clear $RUN home-behind-cgnat`), the claim answers 201 instead and the dialog reads `seedbox dials home`.

- **Choose its Roles.** Searching and Downloading start ticked; **Review**, then **Apply**, lands on `/nodes/<seedbox>/roles`, and `GET /api/v1/nodes/roles` lists seedbox `assigned` `["searching","downloading"]`.

  ```sh
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop click 'getByRole("dialog").getByRole("button", { name: "Choose its Roles" })'
  $FW exec $RUN --proof pairing/pairing-add-dialog-paired -- $PW cli -s={runId}-admin-desktop screenshot --filename={runDir}/proof/pairing/pairing-add-dialog-paired/roles-step.png
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop click 'getByRole("dialog").getByRole("button", { name: "Review", exact: true })'
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop click 'getByRole("dialog", { name: "Review the change" }).getByRole("button", { name: "Apply" })'
  $FW exec $RUN --proof pairing/pairing-add-dialog-paired -- curl -sS -H 'X-Api-Key: {apiKey}' '{nodes.home.url}/api/v1/nodes/roles'
  ```

- **Revoke open codes.** On a fresh `cgnat-home` run, since every **Add a Node** press mints a code and the stray row must be the only other one: mint a stray code over the API, open the dialog, reload, and revoke the stray row, then `This code`.

  ```sh
  $FW exec $RUN -- curl -sS -X POST -H 'X-Api-Key: {apiKey}' '{nodes.home.url}/api/v1/join-codes'
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop click 'getByRole("button", { name: "Add a Node", exact: true })'
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop reload
  $FW exec $RUN --proof pairing/pairing-add-dialog-revoke -- $PW cli -s={runId}-admin-desktop snapshot 'getByRole("dialog")' --filename={runDir}/proof/pairing/pairing-add-dialog-revoke/two-open.aria.yml
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop click 'getByRole("dialog").getByRole("button", { name: /^Revoke the code made / })'
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop click 'getByRole("dialog").getByRole("button", { name: "Revoke this code" })'
  $FW exec $RUN --proof pairing/pairing-add-dialog-revoke -- $PW cli -s={runId}-admin-desktop screenshot --filename={runDir}/proof/pairing/pairing-add-dialog-revoke/revoked.png
  $FW exec $RUN --proof pairing/pairing-add-dialog-revoke -- curl -sS -H 'X-Api-Key: {apiKey}' '{nodes.home.url}/api/v1/join-codes'
  ```

## Gotchas

- A wiped Node has a new key, so the Controller pairs it as a new Node named `seedbox-2`, and the old row stays down with its Roles until **Replace the old row** removes it. The manifest still names the old id, so doctor's check 3 then fails with `node seedbox (<old id>) is not paired` and home holding Searching and Downloading, and check 5 with the Indexer's address no longer being a front for home. Both are expected after this recipe.
- `slipstream join` takes a stopped or a running Node. A wipe needs the Node stopped, so this recipe joins it stopped and starts it after.
- The Join Code is single-use and short-lived. Mint it right before the join.
- A wiped Node logs a new Setup code in `nodes/seedbox.log`, which the API path `POST /api/v1/setup/worker` takes in `X-Setup-Code`.
- `slipstream join` waits for the dial in the foreground until `timings.link.await_dial` (10 minutes) passes. Background it, as above, or the recipe blocks.
- `dialForCode` answers 202 before the Node pairs. A paste that names no waiting Node reads `dialing.unreachable` `true` on `GET /api/v1/join-codes/{id}` once `timings.nodes.dial_report` (30 seconds) passes.
- The dialog shows a code's text only in the tab that made it: the Controller stores its hash. A reload, or `/nodes?add=<id>` in another tab, follows the code without its text.
- The dialog asks `getJoinCode` every 2 seconds only while a pasted address is dialing, since `unreachable` is worked out on read; pairing itself arrives by `nodes.changed`.
- On `cgnat-home`, seedbox's own dials to home keep failing at home's link front, by design. Doctor lists `home-behind-cgnat` as an active fault, not a failure.
- `$FW exec` reads `{...}` as a manifest token, so curl's write-out is spelled `%{{http_code}`.
- A removed Node restarts itself, so the manifest's seedbox PID is stale after **Remove seedbox**; read its state through its URL, not `fakeworld node`. After the rejoin `{nodes.seedbox.nodeId}` still names the tombstone, and doctor's check 3 reports seedbox's Roles.
- The removed Worker logs the Removed sentence twice: once as `pairing` when home refuses its Hello, and once as `node` when its successor starts Unclaimed.
- Removing a Node that is down answers at once; the Node learns it was removed the next time it says Hello, which home refuses as removed.
