# Roles

An Admin changes which Node holds which Role through one plan, and writes off the work a Node still holds for a Role, live or offline. The Role board shows every Node's Roles and the plan's verdict on every drag. A plan answers what a set would do in words, and a set stores it once the guardrails allow it. A Role that SlipStream always needs goes back to the Controller when its last holder lets it go. A set planned on rows that changed since, or one a guardrail blocks, answers 409 with the plan as it stands, and nothing is stored. A Node unticked from Downloading while it seeds keeps seeding, draining, until its torrents are written off; a write-off lets them go and keeps every file. Each Node reports its probes, and a probe that fails blocks the Role it guards there. A Role with a target, such as Notifying and its mail server, is reach tested from the Node that gains it, and mail goes out through the Node that holds Notifying.

## Sub-features

- `roles-board` `GET /api/v1/nodes/roles` answers the `basis`, each Node with `assigned` and its `roles` with a `state` each, and `gestures`, the verdict on every move and add.
- `roles-plan` `POST /api/v1/nodes/roles/plan` answers the Outcome, `effects` and `checks` in words, and stores nothing.
- `roles-set` `PUT /api/v1/nodes/roles` stores the set and answers once the Nodes in step it changed hold their new Roles. Unticking seedbox's Downloading hands it to home: `home takes over Downloading, since SlipStream always needs a Node that downloads.`
- `roles-stale` a plan or set on an old `basis` answers 409 `roles_stale` with `outcome`, the same change planned on the rows now.
- `roles-blocked` adding Relaying to a second Node answers 409 `roles_blocked`, whose check reads `seedbox already relays. Drag Relaying to move it.`
- `roles-probes` `GET /api/v1/nodes` lists each Node's `probes`, `engine-starts`, `data-writable` and `listener-up`, each `fits` with its time. The board's Nodes carry the same `probes` and `suggested`, `searching` and `downloading` while engine-starts fits.
- `roles-reach` a plan with `"test": true` moving Notifying to seedbox checks `seedbox reached 127.0.0.1:<port> in 0 ms.`, the `smtp` front, and a plan without it checks nothing. With the front refused the set answers 409 `roles_blocked`, whose `reach_failed` check reads `seedbox can't reach 127.0.0.1:<port> (connect: connection refused).`
- `roles-notifying` with Notifying on seedbox alone, `POST /api/v1/mail/test` answers `{"outcome":"sent"}` and the `mail` capture holds it. With seedbox stopped it answers `failed` with `seedbox is reconnecting, and no other Node sends mail`, and nothing reaches the capture.
- `roles-roster` `GET /api/v1/nodes` lists each Node's `roles` as `{role, state}`, `holding` while it is in step.
- `roles-drain` unticking Downloading on seedbox while it seeds an imported Grab answers a `drains` effect, `seedbox takes no new Downloading work. It keeps its 1 torrent (3.5 MB) until you move or write it off.`, and the roster lists seedbox `draining` in Downloading with `held`, its facts running `downloading` with only `searching` assigned.
- `roles-write-off` `POST /api/v1/nodes/{id}/write-offs` answers the Outcome with a `writes-off` effect, `The 1 torrent (3.5 MB) seedbox holds for Downloading is written off. No files are deleted.` seedbox announces `stopped`, the roster lists no Downloading on it, and the downloaded file in seedbox's data dir still matches the release.
- `roles-write-off-nothing` writing off a Role the Node holds no work for answers 409 `nothing_held` with `outcome`, whose check reads `seedbox holds nothing to write off for Downloading.`
- `roles-write-off-offline` writing off Downloading on a stopped seedbox with a Grab mid-download answers at once and hands Downloading to home. The Grab fails at `downloading` with `node_written_off`, and Activity reads `What the Node downloading it held was written off. Its files are kept.` Once seedbox is back it runs no Downloading.

## How to get to it (user POV)

- API on the Controller (Admin): `GET /api/v1/nodes/roles`, `POST /api/v1/nodes/roles/plan` with `{"basis": …, "set": [{"nodeId": …, "roles": […]}], "test": true}` and `PUT /api/v1/nodes/roles` with `{"basis": …, "set": [{"nodeId": …, "roles": […]}]}`, `POST /api/v1/mail/test` in an Admin's session, `POST /api/v1/nodes/{id}/write-offs` with `{"roles": […]}`, and `GET /api/v1/nodes`.
- Desktop shell: the sidebar's `Activity` shows a written-off Grab's failure. Phone shell: the `Activity` tab.
- Desktop and phone: no Roles screen or write-off action yet. The Roles section and its Review come later in the Nodes, Pairing and Roles build.

## Driving it with fakeworld and curl

Preconditions:

- A `home` run with doctor passing. Run this last or on its own run: it changes seedbox's Roles, so doctor's check 3 then reports them.

Steps:

- **Read the board.** `home` is the Controller, `seedbox` has `assigned` `searching` and `downloading`, both `holding`.

  ```sh
  BASIS=$($FW exec $RUN --proof roles/roles-board -- curl -sS -H 'X-Api-Key: {apiKey}' '{nodes.home.url}/api/v1/nodes/roles' | jq -r .basis)
  ```

- **Plan unticking Downloading on seedbox.** The Outcome's effects read `seedbox stops Downloading.` and the handback to home.

  ```sh
  $FW exec $RUN --proof roles/roles-plan -- curl -sS -X POST -H 'X-Api-Key: {apiKey}' -H 'Content-Type: application/json' --data-binary "{\"basis\":\"$BASIS\",\"set\":[{\"nodeId\":\"{nodes.seedbox.nodeId}\",\"roles\":[\"searching\"]}]}" '{nodes.home.url}/api/v1/nodes/roles/plan'
  ```

- **Apply it.** The answer is the same Outcome. The roster then lists `seedbox` holding only `searching` and `home` holding `downloading`, each with its facts at its `intentVersion`.

  ```sh
  $FW exec $RUN --proof roles/roles-set -- curl -sS -X PUT -H 'X-Api-Key: {apiKey}' -H 'Content-Type: application/json' --data-binary "{\"basis\":\"$BASIS\",\"set\":[{\"nodeId\":\"{nodes.seedbox.nodeId}\",\"roles\":[\"searching\"]}]}" '{nodes.home.url}/api/v1/nodes/roles'
  $FW exec $RUN --proof roles/roles-roster -- curl -sS -H 'X-Api-Key: {apiKey}' '{nodes.home.url}/api/v1/nodes'
  ```

- **Set again on the old basis.** 409 `roles_stale`, whose `outcome.basis` is the new one.

  ```sh
  $FW exec $RUN --proof roles/roles-stale -- curl -sS -i -X PUT -H 'X-Api-Key: {apiKey}' -H 'Content-Type: application/json' --data-binary "{\"basis\":\"$BASIS\",\"set\":[{\"nodeId\":\"{nodes.seedbox.nodeId}\",\"roles\":[]}]}" '{nodes.home.url}/api/v1/nodes/roles'
  ```

- **Add Relaying to seedbox, then to home.** The first set answers 200. The second answers 409 `roles_blocked` with the `role_taken` check, and home holds no Relaying.

  ```sh
  BASIS=$($FW exec $RUN -- curl -sS -H 'X-Api-Key: {apiKey}' '{nodes.home.url}/api/v1/nodes/roles' | jq -r .basis)
  $FW exec $RUN --proof roles/roles-blocked -- curl -sS -X PUT -H 'X-Api-Key: {apiKey}' -H 'Content-Type: application/json' --data-binary "{\"basis\":\"$BASIS\",\"set\":[{\"nodeId\":\"{nodes.seedbox.nodeId}\",\"roles\":[\"searching\",\"relaying\"]}]}" '{nodes.home.url}/api/v1/nodes/roles'
  BASIS=$($FW exec $RUN -- curl -sS -H 'X-Api-Key: {apiKey}' '{nodes.home.url}/api/v1/nodes/roles' | jq -r .basis)
  $FW exec $RUN --proof roles/roles-blocked -- curl -sS -i -X PUT -H 'X-Api-Key: {apiKey}' -H 'Content-Type: application/json' --data-binary "{\"basis\":\"$BASIS\",\"set\":[{\"nodeId\":\"{nodes.home.nodeId}\",\"roles\":[\"downloading\",\"storage\",\"converting\",\"subtitles\",\"metadata\",\"media-server-link\",\"notifying\",\"relaying\",\"serving\"]}]}" '{nodes.home.url}/api/v1/nodes/roles'
  ```

### Probes, reach tests and Notifying

Preconditions:

- A fresh `home` run with doctor passing, used for nothing else: it moves Notifying to seedbox.

Steps:

- **Read the probes.** Each Node lists its three probes `fits`, and the board suggests `searching` and `downloading` for each.

  ```sh
  $FW exec $RUN --proof roles/roles-probes -- curl -sS -H 'X-Api-Key: {apiKey}' '{nodes.home.url}/api/v1/nodes'
  $FW exec $RUN --proof roles/roles-probes -- curl -sS -H 'X-Api-Key: {apiKey}' '{nodes.home.url}/api/v1/nodes/roles'
  ```

- **Plan Notifying on seedbox, with and without its reach test.** Without `test` the plan checks nothing. With it, the check reads `seedbox reached 127.0.0.1:<port> in 0 ms.`, home's `smtp` front from the Scenario's email setting.

  ```sh
  BASIS=$($FW exec $RUN -- curl -sS -H 'X-Api-Key: {apiKey}' '{nodes.home.url}/api/v1/nodes/roles' | jq -r .basis)
  SET='[{"nodeId":"{nodes.home.nodeId}","roles":["storage","converting","subtitles","metadata","media-server-link","serving"]},{"nodeId":"{nodes.seedbox.nodeId}","roles":["searching","downloading","notifying"]}]'
  $FW exec $RUN --proof roles/roles-reach -- curl -sS -X POST -H 'X-Api-Key: {apiKey}' -H 'Content-Type: application/json' --data-binary "{\"basis\":\"$BASIS\",\"set\":$SET}" '{nodes.home.url}/api/v1/nodes/roles/plan'
  $FW exec $RUN --proof roles/roles-reach -- curl -sS -X POST -H 'X-Api-Key: {apiKey}' -H 'Content-Type: application/json' --data-binary "{\"basis\":\"$BASIS\",\"set\":$SET,\"test\":true}" '{nodes.home.url}/api/v1/nodes/roles/plan'
  ```

- **Set it while the mail server refuses.** 409 `roles_blocked` with the `reach_failed` check, and nothing is stored.

  ```sh
  $FW fault set $RUN smtp-down smtp refuse
  $FW exec $RUN --proof roles/roles-reach -- curl -sS -i -X PUT -H 'X-Api-Key: {apiKey}' -H 'Content-Type: application/json' --data-binary "{\"basis\":\"$BASIS\",\"set\":$SET}" '{nodes.home.url}/api/v1/nodes/roles'
  $FW fault clear $RUN smtp-down
  ```

- **Set it and send a test email.** The set answers with the passing check, and the roster's facts have seedbox holding Notifying. Sign in for a session, since the test email goes to the signed-in Admin, then send it: `{"outcome":"sent"}`, and `mail list` holds it.

  ```sh
  $FW exec $RUN --proof roles/roles-notifying -- curl -sS -X PUT -H 'X-Api-Key: {apiKey}' -H 'Content-Type: application/json' --data-binary "{\"basis\":\"$BASIS\",\"set\":$SET}" '{nodes.home.url}/api/v1/nodes/roles'
  $FW exec $RUN --proof roles/roles-notifying -- curl -sS -H 'X-Api-Key: {apiKey}' '{nodes.home.url}/api/v1/nodes'
  $FW exec $RUN -- curl -sS -c {runDir}/admin.jar -X POST -H 'Content-Type: application/json' --data-binary '{"email":"{admin.email}","pin":"{admin.pin}"}' '{nodes.home.url}/api/v1/session'
  $FW exec $RUN --proof roles/roles-notifying -- curl -sS -b {runDir}/admin.jar -X POST -H 'Origin: {nodes.home.url}' '{nodes.home.url}/api/v1/mail/test'
  $FW mail list $RUN
  ```

- **Stop seedbox and send it again.** It answers `failed` with `seedbox is reconnecting, and no other Node sends mail`, and the capture holds no new message. Start seedbox and wait for its link: the next test email is `sent`.

  ```sh
  $FW node stop $RUN seedbox
  $FW exec $RUN --proof roles/roles-notifying -- curl -sS -b {runDir}/admin.jar -X POST -H 'Origin: {nodes.home.url}' '{nodes.home.url}/api/v1/mail/test'
  $FW node start $RUN seedbox
  $FW exec $RUN -- bash -c 'for i in $(seq 30); do curl -sS -H "X-Api-Key: {apiKey}" "{nodes.home.url}/api/v1/nodes" | jq -e ".nodes[] | select(.name == \"seedbox\") | .link.state == \"up\"" >/dev/null && exit 0; sleep 1; done; exit 1'
  $FW exec $RUN --proof roles/roles-notifying -- curl -sS -b {runDir}/admin.jar -X POST -H 'Origin: {nodes.home.url}' '{nodes.home.url}/api/v1/mail/test'
  ```

### Drain and write off

Preconditions:

- A fresh `home` run with doctor passing, used for nothing else: it grabs The Quiet Orchard and changes seedbox's Roles.

Steps:

- **Write off what seedbox doesn't hold.** 409 `nothing_held`, and nothing is stored.

  ```sh
  $FW exec $RUN --proof roles/roles-write-off-nothing -- curl -sS -i -X POST -H 'X-Api-Key: {apiKey}' -H 'Content-Type: application/json' --data-binary '{"roles":["downloading"]}' '{nodes.home.url}/api/v1/nodes/{nodes.seedbox.nodeId}/write-offs'
  ```

- **Grab a release onto seedbox.** Add The Quiet Orchard and search it, then wait for the Grab to read `imported`.

  ```sh
  WORK=$($FW exec $RUN -- curl -sS -X POST -H 'X-Api-Key: {apiKey}' -H 'Content-Type: application/json' --data-binary '{"contentType":"movie","key":"9000001","monitor":true}' '{nodes.home.url}/api/v1/works' | jq -r .work.id)
  $FW exec $RUN -- curl -sS -X POST -H 'X-Api-Key: {apiKey}' "{nodes.home.url}/api/v1/works/$WORK/search"
  $FW exec $RUN -- bash -c 'for i in $(seq 60); do curl -sS -H "X-Api-Key: {apiKey}" "{nodes.home.url}/api/v1/grabs" | jq -e ".grabs[0].stage == \"imported\"" >/dev/null && exit 0; sleep 2; done; exit 1'
  HASH=$($FW exec $RUN -- curl -sS -H 'X-Api-Key: {apiKey}' '{nodes.home.url}/api/v1/grabs' | jq -r '.grabs[0].infoHash')
  ```

- **Untick Downloading on seedbox.** The Outcome drains seedbox and hands Downloading to home. The roster lists seedbox `draining` with its 1 torrent held.

  ```sh
  BASIS=$($FW exec $RUN -- curl -sS -H 'X-Api-Key: {apiKey}' '{nodes.home.url}/api/v1/nodes/roles' | jq -r .basis)
  $FW exec $RUN --proof roles/roles-drain -- curl -sS -X PUT -H 'X-Api-Key: {apiKey}' -H 'Content-Type: application/json' --data-binary "{\"basis\":\"$BASIS\",\"set\":[{\"nodeId\":\"{nodes.seedbox.nodeId}\",\"roles\":[\"searching\"]}]}" '{nodes.home.url}/api/v1/nodes/roles'
  $FW exec $RUN --proof roles/roles-drain -- curl -sS -H 'X-Api-Key: {apiKey}' '{nodes.home.url}/api/v1/nodes'
  ```

- **Write it off.** The answer carries the `writes-off` effect. seedbox announces `stopped`, the roster lists no Downloading on seedbox, and the file it downloaded still matches the release.

  ```sh
  $FW exec $RUN --proof roles/roles-write-off -- curl -sS -X POST -H 'X-Api-Key: {apiKey}' -H 'Content-Type: application/json' --data-binary '{"roles":["downloading"]}' '{nodes.home.url}/api/v1/nodes/{nodes.seedbox.nodeId}/write-offs'
  $FW announces wait $RUN --node seedbox --where event=stopped --timeout 60s
  $FW exec $RUN --proof roles/roles-write-off -- curl -sS -H 'X-Api-Key: {apiKey}' '{nodes.home.url}/api/v1/nodes'
  $FW exec $RUN --proof roles/roles-write-off -- $FW fs match {runDir} "{nodes.seedbox.dataDir}/downloads/$HASH/The.Quiet.Orchard.2021.1080p.WEB-DL.H264-FKW.mkv" --release The.Quiet.Orchard.2021.1080p.WEB-DL.H264-FKW
  ```

### Write off an offline Node

Preconditions:

- Another fresh `home` run with doctor passing and the `{runId}-admin-desktop` session signed in.

Steps:

- **Hold a Grab mid-download.** Throttle seedbox's seeders, add The Quiet Orchard and search it. seedbox announces `started`.

  ```sh
  $FW fault set $RUN swarm-hold seeders@seedbox throttle=16KiB/s
  WORK=$($FW exec $RUN -- curl -sS -X POST -H 'X-Api-Key: {apiKey}' -H 'Content-Type: application/json' --data-binary '{"contentType":"movie","key":"9000001","monitor":true}' '{nodes.home.url}/api/v1/works' | jq -r .work.id)
  $FW exec $RUN -- curl -sS -X POST -H 'X-Api-Key: {apiKey}' "{nodes.home.url}/api/v1/works/$WORK/search"
  $FW announces wait $RUN --node seedbox --where event=started --timeout 60s
  ```

- **Stop seedbox and write off its Downloading.** The answer comes at once with the `writes-off` effect and home taking over Downloading. The Grab reads stage `failed` with failure `{stage: downloading, code: node_written_off}`, and Activity shows its words.

  ```sh
  $FW node stop $RUN seedbox
  $FW exec $RUN --proof roles/roles-write-off-offline -- curl -sS -X POST -H 'X-Api-Key: {apiKey}' -H 'Content-Type: application/json' --data-binary '{"roles":["downloading"]}' '{nodes.home.url}/api/v1/nodes/{nodes.seedbox.nodeId}/write-offs'
  $FW exec $RUN --proof roles/roles-write-off-offline -- curl -sS -H 'X-Api-Key: {apiKey}' '{nodes.home.url}/api/v1/grabs'
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop goto '{nodes.home.browserUrl}/activity'
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop run-code 'async page => { await page.getByText("What the Node downloading it held was written off. Its files are kept.").waitFor({ timeout: 15000 }); }'
  $FW exec $RUN --proof roles/roles-write-off-offline -- $PW cli -s={runId}-admin-desktop snapshot --filename={runDir}/proof/roles/roles-write-off-offline/activity.aria.yml
  $FW exec $RUN --proof roles/roles-write-off-offline -- $PW cli -s={runId}-admin-desktop screenshot --filename={runDir}/proof/roles/roles-write-off-offline/activity.png
  ```

- **Start seedbox again.** Once its link is `up`, its facts run no Downloading.

  ```sh
  $FW node start $RUN seedbox
  $FW exec $RUN -- bash -c 'for i in $(seq 30); do curl -sS -H "X-Api-Key: {apiKey}" "{nodes.home.url}/api/v1/nodes" | jq -e ".nodes[] | select(.name == \"seedbox\") | .link.state == \"up\" and .facts.running == []" >/dev/null && exit 0; sleep 1; done; exit 1'
  $FW exec $RUN --proof roles/roles-write-off-offline -- curl -sS -H 'X-Api-Key: {apiKey}' '{nodes.home.url}/api/v1/nodes'
  ```

## Gotchas

- Every set needs the `basis` of the board or Outcome it was planned on. Read it again after each set; the old one answers `roles_stale`.
- A Node the set doesn't name keeps its Roles. A Role SlipStream always needs comes back to home whenever no Node in the set holds it, and Serving stays on home whatever the set says.
- A set naming a Node that is down answers at once; the Node gets its Roles when its Sync session attaches again.
- A write-off unticks a Role still ticked on the Node, so the Always handback to home follows it; the drain recipe unticks first to show the draining state on its own.
- A Node that comes back after an offline write-off resumes its cached torrents for a moment, out of step, and lets them go once its intent arrives: the tracker sees a second `started` and no `stopped`. Wait on the roster's facts, not on announces.
- After a write-off doctor's check 3 reports seedbox's Roles, since the Scenario gives it Downloading.
- The `mail` capture attributes a message to the Node whose front it came through, and every Node dials the one `smtp` front the email setting names, home's. A message seedbox sent reads `node: home`; the stopped-seedbox step is what shows the route. The simulation tier's `TestMailGoesThroughTheNodeThatSendsIt` proves the sending address.
- A cookie-session `POST` needs an `Origin` header naming the Node's own URL, or it answers 403 `cross_origin`.
- A probe that fails, such as a downloads folder that can't be written, needs a Node started with other config, which no `fakeworld node` verb does; `TestADownloadsFolderSeedboxCantWriteBlocksDownloadingThere` covers it on the simulation tier.
