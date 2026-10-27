# Offline Nodes

When a Worker stops, SlipStream waits for it through the offline grace before it moves anything. Until the grace passes, the Indexer's `waits` names the Node it waits for, a search's line for that Indexer reads `Waiting for seedbox`, and a Grab it downloads waits rather than failing. Past the grace the Node has lapsed: the Nodes nav alert turns from `warn` to `fail`, the Controller covers Searching, so a search on that Indexer grabs through the Controller, and a Grab with no Node that may download waits at `fetching`. Once the Node is back and has settled, its work routes to it again and the alert clears. The Nodes nav row carries the alert as an amber or red dot, and each Node's `standing` says how it stands. An Admin sets the grace between 1 min and 1 day under Settings > Nodes.

## Sub-features

- `offline-grace` `GET /api/v1/nodes/settings` answers `offlineGrace` as a duration field, 900 s by default between 60 and 86400; `PATCH /api/v1/nodes/settings` with `{"offlineGrace":120}` stores it, and 30 answers 422 with `Between 1 min and 1 day.`
- `offline-settings-page` Settings > Nodes shows `Join Codes` with `Join Code lifetime` and `Offline Nodes` with `Offline grace` (`Default 15 min. Between 1 min and 1 day.`); saving `2` minutes stores 120 and offers `Reset to default`.
- `offline-alert` `GET /api/v1/counts` answers `alerts.nodes` `null` while every Node is in step or reconnecting, `warn` once a Node is away past the blip, `fail` once it is away past the grace, and `null` again once it is back and settled.
- `offline-nav-dot` the sidebar's Nodes row (More's on phone) reads `Nodes` while every Node is in step, `Nodes needs attention` with an amber dot on `warn`, and `Nodes failed` with a red dot on `fail`, with no number, and follows `nodes.changed` without a reload. The collapsed rail's icon carries the dot and the link is named `Nodes, failed`.
- `offline-standing` `GET /api/v1/nodes` gives each Node `standing`: `in-step`, `reconnecting` through the blip, `offline`, `lapsed` past the grace or `mismatched`. The Node picker's dot reads `reconnecting`, `offline` (amber), `offline past the grace` (red) or `on another version`.
- `offline-indexer-waits` within the grace `GET /api/v1/indexers` lists the Indexer homed on seedbox with `waits` `{kind: "waiting", nodeId, since, lapsed: false}` while seedbox's `standing` is `reconnecting` then `offline`, and a search of a wanted Work finishes `nothing_acceptable` with no Grab, its Fixture Site line `waiting`. Its row on the Work page reads `Found nothing acceptable` over the line `Fixture Site` `Waiting for seedbox`.
- `offline-covering` past the grace `GET /api/v1/nodes` lists seedbox `lapsed` in Searching and Downloading and home `covering` Searching, the Indexer's `waits` is `null`, a search finishes `grabbed` through home, and its Grab waits at `fetching` with `fetchNodeId` `null`, since no Node may download.
- `offline-back` once seedbox starts and settles, the waiting Grab is fetched and downloaded on seedbox, `fetchNodeId` and `nodeId` both seedbox's id, and imports.
- `offline-session-waits` on `indexers` with seedbox lapsed, Quarry, which keeps no session, searches from home and its line reads `found`, while Larkspur, Marsh and Cinder, which sign in, read `waiting` `Waiting for seedbox` and list `waits` `{kind: "waiting", lapsed: true}`. Health's Indexers holds a `warn` item `waiting.<indexer id>` for each, `Larkspur is waiting for seedbox, which has been offline since <HH:MM>. Allow failover to search from another Node.`, linking `Open Larkspur` to `/settings/indexers/<id>#allow-failover`. An RSS feed with a pasted cookie signs in the same way (`signsIn` `true`).
- `offline-allow-failover` once `PATCH /api/v1/indexers/<id>` sets `allowFailover` `true` the warns go, and the next search sends the three to home: Marsh signs in there (a `fixture-site.signin` record from `home`) and reads `found`, while Larkspur's captcha and Cinder's browser challenge make each wait for seedbox again, `status` `home_only` with `cause` `captcha_needed` or `challenge_unsolved` and `nodeId` home's. The search after that skips both without a request to home, and Marsh reuses its session there.
- `offline-covering-sessions` on `indexers` with Searching taken from home, Fixture Site and Brook move to seedbox. Past the grace home covers Searching: Quarry and Fixture Site read `found` through home, and the three that sign in read `waiting`.
- `offline-write-off` `DELETE /api/v1/nodes/<seedbox id>?writeOff=true` answers 204, and every Indexer is homed on home with `session` `null`: the sessions seedbox earned are deleted with it.

## How to get to it (user POV)

- API on the Controller (Admin): `GET` and `PATCH /api/v1/nodes/settings`, `GET /api/v1/counts`, `GET /api/v1/indexers`, `GET /api/v1/nodes`, `POST /api/v1/works/{id}/search`, `GET /api/v1/works/{id}/searches` and `GET /api/v1/grabs`.
- Host: `fakeworld node stop` and `start` take seedbox away and bring it back.
- Desktop shell: the sidebar's Nodes row carries the dot, `/settings/nodes` holds the grace, the Work page's Searches show the Indexer's line, and the Node picker on `/nodes/<id>` shows each Node's standing. Phone shell: More's Nodes row carries the dot, and `More`, `Settings`, `Nodes` holds the grace.

## Driving it with fakeworld and curl

Preconditions:

- A fresh `home` run with doctor passing. Run this last or on its own run: it stops seedbox, and the Work it adds is grabbed.

Steps:

- **Set the grace to 2 min.** A grace under 1 min is refused. With seedbox in step, the alert is `null`.

  ```sh
  $FW exec $RUN --proof offline/offline-grace -- curl -sS -X PATCH -H 'X-Api-Key: {apiKey}' -H 'Content-Type: application/json' --data-binary '{"offlineGrace":30}' '{nodes.home.url}/api/v1/nodes/settings'
  $FW exec $RUN --proof offline/offline-grace -- curl -sS -X PATCH -H 'X-Api-Key: {apiKey}' -H 'Content-Type: application/json' --data-binary '{"offlineGrace":120}' '{nodes.home.url}/api/v1/nodes/settings'
  $FW exec $RUN --proof offline/offline-alert -- curl -sS -H 'X-Api-Key: {apiKey}' '{nodes.home.url}/api/v1/counts'
  WORK=$($FW exec $RUN -- curl -sS -X POST -H 'X-Api-Key: {apiKey}' -H 'Content-Type: application/json' --data-binary '{"contentType":"movie","key":"9000001","monitor":true}' '{nodes.home.url}/api/v1/works' | jq -r .work.id)
  ```

- **Stop seedbox and search within the grace.** The Indexer reads `waits` of kind `waiting` while seedbox's `standing` is `reconnecting`, and the search finishes `nothing_acceptable` with its line `waiting`.

  ```sh
  $FW node stop $RUN seedbox
  $FW exec $RUN --proof offline/offline-indexer-waits -- curl -sS -H 'X-Api-Key: {apiKey}' '{nodes.home.url}/api/v1/indexers'
  $FW exec $RUN -- curl -sS -X POST -H 'X-Api-Key: {apiKey}' "{nodes.home.url}/api/v1/works/$WORK/search"
  $FW exec $RUN -- bash -c "for i in \$(seq 30); do curl -sS -H 'X-Api-Key: {apiKey}' '{nodes.home.url}/api/v1/works/$WORK/searches' | jq -e '.searches[0].outcome != null' >/dev/null && exit 0; sleep 1; done; exit 1"
  $FW exec $RUN --proof offline/offline-indexer-waits -- curl -sS -H 'X-Api-Key: {apiKey}' "{nodes.home.url}/api/v1/works/$WORK/searches"
  ```

- **Wait past the blip.** A minute after the stop the alert reads `warn` and seedbox's `standing` `offline`, with the Indexer still waiting.

  ```sh
  $FW exec $RUN -- bash -c "for i in \$(seq 90); do curl -sS -H 'X-Api-Key: {apiKey}' '{nodes.home.url}/api/v1/counts' | jq -e '.alerts.nodes == \"warn\"' >/dev/null && exit 0; sleep 1; done; exit 1"
  $FW exec $RUN --proof offline/offline-alert -- curl -sS -H 'X-Api-Key: {apiKey}' '{nodes.home.url}/api/v1/counts'
  $FW exec $RUN --proof offline/offline-indexer-waits -- curl -sS -H 'X-Api-Key: {apiKey}' '{nodes.home.url}/api/v1/indexers'
  ```

- **Wait past the grace and search again.** The alert reads `fail`, seedbox is `lapsed`, home `covering` Searching, and the search finishes `grabbed`. Its Grab stays at `fetching` with no `fetchNodeId`.

  ```sh
  $FW exec $RUN -- bash -c "for i in \$(seq 90); do curl -sS -H 'X-Api-Key: {apiKey}' '{nodes.home.url}/api/v1/counts' | jq -e '.alerts.nodes == \"fail\"' >/dev/null && exit 0; sleep 1; done; exit 1"
  $FW exec $RUN --proof offline/offline-alert -- curl -sS -H 'X-Api-Key: {apiKey}' '{nodes.home.url}/api/v1/counts'
  $FW exec $RUN --proof offline/offline-covering -- curl -sS -H 'X-Api-Key: {apiKey}' '{nodes.home.url}/api/v1/nodes'
  $FW exec $RUN --proof offline/offline-covering -- curl -sS -H 'X-Api-Key: {apiKey}' '{nodes.home.url}/api/v1/indexers'
  $FW exec $RUN -- curl -sS -X POST -H 'X-Api-Key: {apiKey}' "{nodes.home.url}/api/v1/works/$WORK/search"
  $FW exec $RUN -- bash -c "for i in \$(seq 30); do curl -sS -H 'X-Api-Key: {apiKey}' '{nodes.home.url}/api/v1/works/$WORK/searches' | jq -e '.searches[0].outcome != null' >/dev/null && exit 0; sleep 1; done; exit 1"
  $FW exec $RUN --proof offline/offline-covering -- curl -sS -H 'X-Api-Key: {apiKey}' "{nodes.home.url}/api/v1/works/$WORK/searches"
  $FW exec $RUN --proof offline/offline-covering -- curl -sS -H 'X-Api-Key: {apiKey}' '{nodes.home.url}/api/v1/grabs'
  ```

- **See it in the browser.** On a fresh run, set the grace to 2 min on `/settings/nodes`, add The Quiet Orchard, stop seedbox, and once the Nodes row reads `Nodes needs attention` choose Search: the newest row reads `Found nothing acceptable0 Releases foundFixture SiteWaiting for seedbox`. Past the grace the row reads `Nodes failed`. The browser tier's `web/e2e/nodes.spec.ts` drives the same path on desktop and phone.

  ```sh
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop goto '{nodes.home.browserUrl}/settings/nodes'
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop fill 'getByRole("region", { name: "Offline Nodes" }).getByRole("textbox", { name: "Offline grace" })' 2
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop click 'getByRole("region", { name: "Offline Nodes" }).getByRole("button", { name: "Save" })'
  $FW exec $RUN --proof offline/offline-settings-page -- $PW cli -s={runId}-admin-desktop screenshot --filename={runDir}/proof/offline/offline-settings-page/saved.png
  $FW node stop $RUN seedbox
  $FW exec $RUN --proof offline/offline-nav-dot -- $PW cli -s={runId}-admin-desktop run-code 'async page => { await page.getByRole("navigation", { name: "Main" }).getByRole("link", { name: "Nodes needs attention" }).waitFor({ timeout: 90000 }); }'
  $FW exec $RUN --proof offline/offline-nav-dot -- $PW cli -s={runId}-admin-desktop screenshot --filename={runDir}/proof/offline/offline-nav-dot/warn.png
  ```

- **Start seedbox.** Once it has settled, a minute after its link is up, the Grab imports with seedbox's id in `fetchNodeId` and `nodeId`, and the alert is `null`.

  ```sh
  $FW node start $RUN seedbox
  $FW exec $RUN -- bash -c "for i in \$(seq 180); do curl -sS -H 'X-Api-Key: {apiKey}' '{nodes.home.url}/api/v1/grabs' | jq -e '.grabs[0].stage == \"imported\"' >/dev/null && exit 0; sleep 1; done; exit 1"
  $FW exec $RUN --proof offline/offline-back -- curl -sS -H 'X-Api-Key: {apiKey}' '{nodes.home.url}/api/v1/grabs'
  $FW exec $RUN --proof offline/offline-alert -- curl -sS -H 'X-Api-Key: {apiKey}' '{nodes.home.url}/api/v1/counts'
  ```

### Login-session Indexers on `indexers`

Preconditions:

- A fresh `indexers` run with doctor passing. It stops seedbox and writes it off, so run it on its own run.

Steps:

- **Take seedbox past the grace and search.** Set the grace to 1 min, add Harbor Lights, stop seedbox and wait until Larkspur's `waits.lapsed` is `true`. Read Health, then search.

  ```sh
  $FW exec $RUN -- curl -sS -X PATCH -H 'X-Api-Key: {apiKey}' -H 'Content-Type: application/json' --data-binary '{"offlineGrace":60}' '{nodes.home.url}/api/v1/nodes/settings'
  WORK=$($FW exec $RUN -- curl -sS -X POST -H 'X-Api-Key: {apiKey}' -H 'Content-Type: application/json' --data-binary '{"contentType":"movie","key":"9000002","monitor":true}' '{nodes.home.url}/api/v1/works' | jq -r .work.id)
  $FW node stop $RUN seedbox
  $FW exec $RUN -- bash -c "for i in \$(seq 120); do curl -sS -H 'X-Api-Key: {apiKey}' '{nodes.home.url}/api/v1/indexers' | jq -e '.indexers[] | select(.name == \"Larkspur\") | .waits.lapsed' >/dev/null && exit 0; sleep 1; done; exit 1"
  $FW exec $RUN --proof offline/offline-session-waits -- curl -sS -H 'X-Api-Key: {apiKey}' '{nodes.home.url}/api/v1/system/health'
  $FW exec $RUN -- curl -sS -X POST -H 'X-Api-Key: {apiKey}' "{nodes.home.url}/api/v1/works/$WORK/search"
  $FW exec $RUN -- bash -c "for i in \$(seq 60); do curl -sS -H 'X-Api-Key: {apiKey}' '{nodes.home.url}/api/v1/works/$WORK/searches' | jq -e '.searches[0].outcome != null' >/dev/null && exit 0; sleep 1; done; exit 1"
  $FW exec $RUN --proof offline/offline-session-waits -- curl -sS -H 'X-Api-Key: {apiKey}' "{nodes.home.url}/api/v1/works/$WORK/searches"
  ```

- **Allow failover and search another Work.** Allow failover on Larkspur, Marsh and Cinder, add Salt Meadow (`9000003`) and search it as above. Read the Indexers and the sign-ins home's front recorded.

  ```sh
  for IX in $($FW exec $RUN -- curl -sS -H 'X-Api-Key: {apiKey}' '{nodes.home.url}/api/v1/indexers' | jq -r '.indexers[] | select(.signsIn) | .id'); do
    $FW exec $RUN -- curl -sS -X PATCH -H 'X-Api-Key: {apiKey}' -H 'Content-Type: application/json' --data-binary '{"allowFailover":true}' "{nodes.home.url}/api/v1/indexers/$IX"
  done
  $FW exec $RUN --proof offline/offline-allow-failover -- curl -sS -H 'X-Api-Key: {apiKey}' '{nodes.home.url}/api/v1/indexers'
  $FW exec $RUN --proof offline/offline-allow-failover -- $FW sign-ins list {runDir} --node home --where site=marsh
  ```

- **Write seedbox off.** Every Indexer moves to home and holds no session.

  ```sh
  $FW exec $RUN --proof offline/offline-write-off -- curl -sS -X DELETE -H 'X-Api-Key: {apiKey}' "{nodes.home.url}/api/v1/nodes/{nodes.seedbox.nodeId}?writeOff=true"
  $FW exec $RUN --proof offline/offline-write-off -- curl -sS -H 'X-Api-Key: {apiKey}' '{nodes.home.url}/api/v1/indexers'
  ```

`TestALoginSessionIndexerWaitsForItsLapsedHomeUntilItAllowsFailover` and `TestWhileTheControllerOnlyCoversSearchingASessionWaitsAndAWriteOffDeletesSeedboxsSessions` in `e2e/failover_test.go` drive the same paths, the second with Searching taken from home first.

## Gotchas

- A search's Indexer line says how the Indexer stood when the search ran. It doesn't change once seedbox is back.

- The blip is 1 min, the same as the grace's floor, so a 1 min grace goes from `reconnecting` straight to `lapsed` and never reads `warn`. Use 2 min to see all three.
- The grace runs from seedbox's last Sync frame, or from the Controller's start when that is later: restarting home restarts every away Node's grace.
- The alert stays `warn` while seedbox settles after it is back, since home still covers Searching; it clears a minute after the link is up.
- A Worker whose link answers pings but whose Sync stream goes silent for `timings.role.stall` (45 s) loses its link and dials again; this recipe can't drive that, and `TestAWorkerSilentOnSyncLosesItsLink` proves it.
- `fakeworld node stop seedbox` makes doctor's cluster check fail until `node start`.
- Larkspur shows a captcha at every sign-in, so allowing it to fail over makes it wait for seedbox again at its first search from home, as does Cinder's browser challenge, cleared for seedbox only. Marsh is the one that keeps searching from home. A wait for home ends only once seedbox answers again.
- The waiting warn needs the home past the grace and the Indexer signing in without Allow failover. One that allows failover and still waits for home, as Larkspur does after its captcha, raises none.
