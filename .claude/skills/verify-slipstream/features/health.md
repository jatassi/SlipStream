# Health

Health is one list of what needs a look, built from the features that already know. The Health row in the system nav carries an amber or red dot while anything is wrong, and the Health page opens with the sentence that says what to do first, or `Everything is working.` with what it checked, beside `Check all`. Each category, Storage, Nodes, Download Clients and Scheduled tasks, is a section with its level and count, errors first, or one quiet line with its Check now; each item says what to do first, how long it has stood, its facts under the row and its one action, in a sheet on phone. Storage judges every volume a Storage Node reports, and each downloading Node's downloads volume, against a Warning level and an Error level, in bytes or a percent of the volume, globally on the Health page and in Library roots and per volume from Library roots, and raises an item per library root in trouble. Library roots carries the Storage dot on its settings row and its worst item at its top, with `Open Health`. An Admin or a script reads the same report at `GET /api/v1/system/health`, the worst level as `alerts.health` and Scheduled tasks' as `alerts.tasks` in `GET /api/v1/counts`, and which categories changed as `health.changed` on the live stream.

## Sub-features

- `health-report` with nothing wrong, `GET /api/v1/system/health` answers `level` `null`, `first` `null`, and the categories `storage`, `nodes`, `download-clients` and `scheduled-tasks` in that order, each with its `label`, `phrase`, `level` `null`, `canCheck` (`true` for Storage and Nodes) and `items` `[]`: what the page's `Everything is working.` and `Checked storage, Nodes, download clients and scheduled tasks` are read from. `GET /api/v1/counts` answers `alerts.health` and `alerts.tasks` `null` (P-213).
- `health-node-offline` with seedbox stopped past the offline grace, Nodes holds a `fail` item keyed by seedbox's Node id, subject `seedbox`, `seedbox has been offline past the offline grace. Start SlipStream on it, or remove it if it is gone.`, the fact `Offline since` and a `link` action `Open seedbox` to `/nodes/<id>`, and a `warn` item `covering.searching`, subject `Searching on home`, `home searches in place of seedbox. Bring seedbox back, or give Searching to another Node.`; the report's `level` is `fail` and `first` is seedbox's item, `/counts` answers `alerts.health` and `alerts.nodes` `fail`, and the stream carries `health.changed` with `{"categories":["nodes"]}`. Once seedbox starts its `fail` item clears at once while home's covering warning stays and both alerts read `warn`; once seedbox settles the warning clears too and the alerts are `null` again, each change with a `health.changed` naming `nodes` (P-213).
- `health-check-now` `POST /api/v1/system/health/nodes/check` answers `report` and `unreached`: `[]` with every Node in step, and `[{"words":"seedbox is offline."}]` with seedbox away, its item's `since` unchanged. `POST /api/v1/system/health/check` checks every category the same way. An item with a link, and a category with no Check now, answer 404 `no_check`, and an unknown category 404 `health_category_not_found` (P-214).
- `health-levels-api` `GET /api/v1/system/health/settings` answers `warnBelow` and `errorBelow` as space fields, 100 GB and 20 GB in bytes by default; `PATCH` takes `{"amount":<n>,"unit":"bytes"}` or `"unit":"percent"`, and units may mix. An error level above a warning level in the same unit answers 422 `setting_out_of_range` with `No more than the warning level, <level>.`, whether both are in the request or the warning is the stored one, and stores nothing. `null` resets a level to its default. Each saved change carries `health.changed` naming `storage` (P-216).
- `health-download-clients` on `home`, Download Clients holds a `fail` item `<node id>.listen`, subject `Torrent client on seedbox`, `The torrent client on seedbox can't listen on <listen>, so no peer can connect in. If another program holds the port, free it or choose another, then restart seedbox.`, with the facts `Listen address` and `Error` and a `link` action `Open Torrent client` to `/nodes/<id>/torrent-client`, while seedbox's listen port is held by another program; Nodes' `engine-starts` probe raises its own `fail` beside it. While the disk guard holds (seedbox's downloads volume below its error level) it holds a `warn` item `<node id>.disk-guard`, subject `New downloads on seedbox`, `New downloads on seedbox wait for disk space. <volume> has <free> left, below its error level of <level>. …`, with `Volume`, `Free`, `Total`, `Error level` and `Sampled` and the same link; Storage judges the same downloads volume as `<volume> on seedbox`. Each change carries `health.changed` naming `download-clients`, and both items clear once the cause goes.
- `health-page-calm` on `home` with nothing wrong, the Health nav row reads `Health` with no dot, and `/health` reads `Everything is working.` over `Checked storage, Nodes, download clients and scheduled tasks`, then `Storage. All good.`, `Nodes. All good.`, `Download Clients. All good.` and `Scheduled tasks. All good.`, the first two with `Check now: Storage` and `Check now: Nodes` and the last two with none. `Check all` and each Check now answer `Checked.` (P-213, P-214).
- `health-page-node-offline` with seedbox offline past the grace, the page moves Nodes first as a section `Nodes Error 2`, the status sentence links `seedbox has been offline past the offline grace. Start SlipStream on it, or remove it if it is gone.`, the row `Error seedbox … since <HH:MM>` unfolds `Offline since` and `Open seedbox`, which opens seedbox's page, and the `Warning Searching on home` row follows it. The nav row reads `Health failed` with a red dot. `Check now: Nodes` answers `Checked. seedbox is offline.`, and once seedbox is back the rows go without a reload and the page reads `Everything is working.` again (P-213, P-214).
- `health-page-levels` the Storage section ends with `A volume shows a warning below <level> free and an error below <level>. Storage Nodes check each volume every 5 min, and again after each import.`, whose Details hold `Warn when free space is below`, `Show an error when free space is below`, each with a GB, TB or % unit, `Check storage every` and `Set a volume's own levels in Library roots`. An error level above the warning in the same unit reads `No more than the warning level, <level>.` beside the field and saves nothing; a warning of 100% raises a Storage warning on the volume and the nav row reads `Health needs attention` (P-215, P-216).
- `health-volume-levels` in Settings > Library roots, a volume's `Actions for <volume>`, then `Set warning levels`, opens `Warning levels for <volume>`, reading `<volume> on home follows the global levels until you change one here.`; a warning of 100% there reads `<volume> on home has its own levels.`, the volume's row gains `Own levels` and a `Warning` chip with an amber bar, the `Library roots 1 warning` tab and a warning line with `Open Health` appear, and `Follow the global levels` takes them all away (P-215, P-217).
- `health-settings-row` on `low-space`, whose Scenario sets both levels to 1024 TB, the nav row reads `Health failed`, the Settings tab reads `Library roots 1 error` (the Settings list row on phone), and the section opens with `Error <volume> on home Free up space on <volume> on home, or add a library root on another volume. It has <free> left. since <HH:MM>` and `Open Health`, which lands on `/health#health-storage-<key>` with that item. Its row unfolds `Free`, `Total`, `Warning level` and `Error level` `1024 TB (global)` and `Sampled`, and `Check now: <volume> on home` answers `Checked.` with the item kept (P-215, P-217).

## How to get to it (user POV)

- Desktop shell: the sidebar's `Health` row (`/health`), between Nodes and Accounts. An item's row unfolds its facts and action. Settings > Library roots carries the Storage dot on its tab, its worst item at the top with `Open Health`, and `Actions for <volume>` on each volume of its Storage block.
- Phone shell: More, then `Health`, a pushed page; an item opens a sheet named for its subject with its facts and action. Settings lists `Library roots` with the dot, and its warning line opens a sheet with `Open Health`.
- API (Admin): `GET /api/v1/system/health`, `POST /api/v1/system/health/check`, `POST /api/v1/system/health/{category}/check`, `POST /api/v1/system/health/{category}/items/{key}/check`, `GET` and `PATCH /api/v1/system/health/settings`, `GET`, `PATCH` and `DELETE /api/v1/system/health/volumes`, and `GET /api/v1/counts` for `alerts.health` and `alerts.tasks`; the live event `health.changed`.
- Host CLI: none.

## Driving it with fakeworld, curl and playwright cli

Preconditions:

- A fresh `home` run with doctor passing and `{runId}-admin-desktop` signed in, as in the skill's Launch and Drive. Run the API steps in order: the offline steps stop and start seedbox, and take about three minutes of waiting. The page steps follow them on the same run.
- A separate `low-space` run (`--scenario low-space`) with its own `{runId}-admin-desktop` and `{runId}-admin-phone` for **The settings row on low space**.
- `S='-s={runId}-admin-desktop'`.

Steps:

- **Read the report with nothing wrong.** Every category is clear and both alerts are `null`.

  ```sh
  $FW exec $RUN --proof health/health-report -- curl -sS -H 'X-Api-Key: {apiKey}' '{nodes.home.url}/api/v1/system/health'
  $FW exec $RUN --proof health/health-report -- curl -sS -H 'X-Api-Key: {apiKey}' '{nodes.home.url}/api/v1/counts'
  ```

- **Check Nodes now with every Node in step.** `unreached` is `[]` and Nodes has no items. A category with no Check now and an unknown one answer 404.

  ```sh
  $FW exec $RUN --proof health/health-check-now -- curl -sS -X POST -H 'X-Api-Key: {apiKey}' '{nodes.home.url}/api/v1/system/health/nodes/check'
  $FW exec $RUN --proof health/health-check-now -- curl -sS -X POST -H 'X-Api-Key: {apiKey}' '{nodes.home.url}/api/v1/system/health/scheduled-tasks/check'
  $FW exec $RUN --proof health/health-check-now -- curl -sS -X POST -H 'X-Api-Key: {apiKey}' '{nodes.home.url}/api/v1/system/health/indexers/check'
  ```

- **Set the levels in bytes and percent.** The warning saves as 50 GB and the error as 5%. A same-unit error above the warning, 15% over 10%, answers 422 `No more than the warning level, 10%.` and the read still shows 50 GB and 5%. Reset both with `null`, then an error of 200 GB over the default 100 GB answers 422 `No more than the warning level, 100 GB.`. The stream carries `health.changed` naming `storage` for the two saves. Set both back to 0B after: the defaults judge the host's disk and may raise a Storage item that holds `alerts.health` through the offline steps.

  ```sh
  $FW exec $RUN --proof health/health-levels-api -- curl -sS -H 'X-Api-Key: {apiKey}' '{nodes.home.url}/api/v1/system/health/settings'
  $FW exec $RUN --proof health/health-levels-api -- curl -sS -X PATCH -H 'X-Api-Key: {apiKey}' -H 'Content-Type: application/json' --data-binary '{"warnBelow":{"amount":53687091200,"unit":"bytes"},"errorBelow":{"amount":5,"unit":"percent"}}' '{nodes.home.url}/api/v1/system/health/settings'
  $FW exec $RUN --proof health/health-levels-api -- curl -sS -i -X PATCH -H 'X-Api-Key: {apiKey}' -H 'Content-Type: application/json' --data-binary '{"warnBelow":{"amount":10,"unit":"percent"},"errorBelow":{"amount":15,"unit":"percent"}}' '{nodes.home.url}/api/v1/system/health/settings'
  $FW exec $RUN --proof health/health-levels-api -- curl -sS -H 'X-Api-Key: {apiKey}' '{nodes.home.url}/api/v1/system/health/settings'
  $FW exec $RUN --proof health/health-levels-api -- curl -sS -X PATCH -H 'X-Api-Key: {apiKey}' -H 'Content-Type: application/json' --data-binary '{"warnBelow":null,"errorBelow":null}' '{nodes.home.url}/api/v1/system/health/settings'
  $FW exec $RUN --proof health/health-levels-api -- curl -sS -X PATCH -H 'X-Api-Key: {apiKey}' -H 'Content-Type: application/json' --data-binary '{"errorBelow":{"amount":214748364800,"unit":"bytes"}}' '{nodes.home.url}/api/v1/system/health/settings'
  $FW exec $RUN --proof health/health-levels-api -- $FW sse list {runDir} --as admin --event health.changed
  $FW exec $RUN -- curl -sS -X PATCH -H 'X-Api-Key: {apiKey}' -H 'Content-Type: application/json' --data-binary '{"warnBelow":{"amount":0,"unit":"bytes"},"errorBelow":{"amount":0,"unit":"bytes"}}' '{nodes.home.url}/api/v1/system/health/settings'
  ```

- **Stop seedbox and wait past the grace.** Set the grace to its 60 s floor and stop seedbox. About a minute later `/counts` reads `alerts.health` and `alerts.nodes` `fail`, the report holds seedbox's `fail` item with its link and home's `covering.searching` warning, and the stream's newest `health.changed` reads `{"categories":["nodes"]}`.

  ```sh
  $FW exec $RUN --proof health/health-node-offline -- curl -sS -X PATCH -H 'X-Api-Key: {apiKey}' -H 'Content-Type: application/json' --data-binary '{"offlineGrace":60}' '{nodes.home.url}/api/v1/nodes/settings'
  $FW node stop $RUN seedbox
  $FW exec $RUN -- bash -c "for i in \$(seq 120); do curl -sS -H 'X-Api-Key: {apiKey}' '{nodes.home.url}/api/v1/counts' | jq -e '.alerts.health == \"fail\"' >/dev/null && exit 0; sleep 1; done; exit 1"
  $FW exec $RUN --proof health/health-node-offline -- curl -sS -H 'X-Api-Key: {apiKey}' '{nodes.home.url}/api/v1/counts'
  $FW exec $RUN --proof health/health-node-offline -- curl -sS -H 'X-Api-Key: {apiKey}' '{nodes.home.url}/api/v1/system/health'
  $FW exec $RUN --proof health/health-node-offline -- $FW sse list {runDir} --as admin --event health.changed
  ```

- **Check now with seedbox offline.** Nodes and Check all each answer `unreached` `[{"words":"seedbox is offline."}]`, and seedbox's item keeps its `since` and `changedAt`. Its own Check now answers 404 `no_check`, since its action is a link.

  ```sh
  $FW exec $RUN --proof health/health-check-now -- curl -sS -X POST -H 'X-Api-Key: {apiKey}' '{nodes.home.url}/api/v1/system/health/nodes/check'
  $FW exec $RUN --proof health/health-check-now -- curl -sS -X POST -H 'X-Api-Key: {apiKey}' '{nodes.home.url}/api/v1/system/health/check'
  $FW exec $RUN --proof health/health-check-now -- curl -sS -X POST -H 'X-Api-Key: {apiKey}' '{nodes.home.url}/api/v1/system/health/nodes/items/{nodes.seedbox.nodeId}/check'
  ```

- **Start seedbox and see the items clear.** Right after the start Nodes holds only `covering.searching` and both alerts read `warn`. About a minute later, once seedbox has settled, Nodes has no items, both alerts are `null`, and two more `health.changed` name `nodes`.

  ```sh
  $FW node start $RUN seedbox
  $FW exec $RUN --proof health/health-node-offline -- curl -sS -H 'X-Api-Key: {apiKey}' '{nodes.home.url}/api/v1/system/health'
  $FW exec $RUN --proof health/health-node-offline -- curl -sS -H 'X-Api-Key: {apiKey}' '{nodes.home.url}/api/v1/counts'
  $FW exec $RUN -- bash -c "for i in \$(seq 150); do curl -sS -H 'X-Api-Key: {apiKey}' '{nodes.home.url}/api/v1/counts' | jq -e '.alerts.health == null' >/dev/null && exit 0; sleep 1; done; exit 1"
  $FW exec $RUN --proof health/health-node-offline -- curl -sS -H 'X-Api-Key: {apiKey}' '{nodes.home.url}/api/v1/system/health'
  $FW exec $RUN --proof health/health-node-offline -- curl -sS -H 'X-Api-Key: {apiKey}' '{nodes.home.url}/api/v1/counts'
  $FW exec $RUN --proof health/health-node-offline -- $FW sse list {runDir} --as admin --event health.changed
  $FW doctor $RUN
  ```

- **Read Download Clients.** The disk guard steps of [download-clients.md](./download-clients.md) raise and clear its warning; read it there. For the bind failure, hold a port on seedbox's address with another program, point seedbox's listener at it and restart seedbox: Download Clients and Nodes each hold a `fail`. Put the listener back and restart, and the report reads `level` `null` again.

  ```sh
  T=$(jq -r '.nodes[] | select(.name == "seedbox").listeners.torrent' $RUN/manifest.json)
  $FW exec $RUN -- timeout 900 python3 -I -c 'import socket,time; s=socket.socket(); s.bind(("127.0.0.3", 41999)); s.listen(); time.sleep(900)' &
  $FW exec $RUN --proof health/health-download-clients -- curl -sS -X PATCH -H 'X-Api-Key: {apiKey}' -H 'Content-Type: application/json' -d '{"torrent.listen":"127.0.0.3:41999"}' '{nodes.home.url}/api/v1/nodes/{nodes.seedbox.nodeId}/config'
  $FW node restart $RUN seedbox
  $FW exec $RUN -- bash -c "for i in \$(seq 90); do curl -sS -H 'X-Api-Key: {apiKey}' '{nodes.home.url}/api/v1/system/health' | jq -e '.categories[] | select(.id==\"download-clients\") | .level == \"fail\"' >/dev/null && exit 0; sleep 1; done; exit 1"
  $FW exec $RUN --proof health/health-download-clients -- curl -sS -H 'X-Api-Key: {apiKey}' '{nodes.home.url}/api/v1/system/health'
  $FW exec $RUN -- $PW cli $S goto '{nodes.home.browserUrl}/health'
  $FW exec $RUN -- $PW cli $S click 'getByRole("region", { name: "Download Clients" }).getByRole("button").filter({ hasText: "Torrent client on seedbox" })'
  $FW exec $RUN --proof health/health-download-clients -- $PW cli $S screenshot --full-page --filename={runDir}/proof/health/health-download-clients/bind.png
  $FW exec $RUN -- curl -sS -X PATCH -H 'X-Api-Key: {apiKey}' -H 'Content-Type: application/json' -d "{\"torrent.listen\":\"$T\"}" '{nodes.home.url}/api/v1/nodes/{nodes.seedbox.nodeId}/config'
  $FW node restart $RUN seedbox
  ```

- **Read the calm page.** On the `home` run, after the API steps, the nav row reads `Health` and the page reads `Everything is working.` with three quiet lines. `Check all` answers `Checked.`.

  ```sh
  $FW exec $RUN -- $PW cli $S goto '{nodes.home.browserUrl}/health'
  $FW exec $RUN -- $PW cli $S run-code 'async page => { await page.getByText("Everything is working.").waitFor(); }'
  $FW exec $RUN --proof health/health-page-calm -- $PW cli $S snapshot --filename={runDir}/proof/health/health-page-calm/calm.aria.yml
  $FW exec $RUN --proof health/health-page-calm -- $PW cli $S screenshot --filename={runDir}/proof/health/health-page-calm/calm.png
  $FW exec $RUN -- $PW cli $S click 'getByRole("button", { name: "Check all" })'
  $FW exec $RUN -- $PW cli $S run-code 'async page => { await page.getByRole("status").filter({ hasText: /^Checked\./ }).waitFor(); }'
  $FW exec $RUN --proof health/health-page-calm -- $PW cli $S screenshot --filename={runDir}/proof/health/health-page-calm/checked.png
  ```

- **Watch seedbox go offline on the page.** With the grace still at 60 s, stop seedbox and keep the page open. Within about a minute and a half the Nodes section leads with seedbox's Error row and the nav row reads `Health failed`. Unfold the row, then `Check now: Nodes`. Start seedbox and the rows leave without a reload.

  ```sh
  $FW node stop $RUN seedbox
  $FW exec $RUN -- $PW cli $S run-code 'async page => { await page.getByRole("region", { name: "Nodes" }).getByText("seedbox has been offline past the offline grace.", { exact: false }).first().waitFor({ timeout: 150000 }); }'
  $FW exec $RUN -- $PW cli $S click 'getByRole("region", { name: "Nodes" }).getByRole("button").filter({ hasText: "seedbox has been offline" })'
  $FW exec $RUN --proof health/health-page-node-offline -- $PW cli $S snapshot --filename={runDir}/proof/health/health-page-node-offline/offline.aria.yml
  $FW exec $RUN --proof health/health-page-node-offline -- $PW cli $S screenshot --filename={runDir}/proof/health/health-page-node-offline/offline.png
  $FW exec $RUN -- $PW cli $S click 'getByRole("button", { name: "Check now: Nodes" })'
  $FW exec $RUN -- $PW cli $S run-code 'async page => { await page.getByText("Checked. seedbox is offline.").waitFor(); }'
  $FW exec $RUN --proof health/health-page-node-offline -- $PW cli $S screenshot --filename={runDir}/proof/health/health-page-node-offline/checked.png
  $FW node start $RUN seedbox
  $FW exec $RUN -- $PW cli $S run-code 'async page => { await page.getByText("Everything is working.").waitFor({ timeout: 150000 }); }'
  $FW exec $RUN --proof health/health-page-node-offline -- $PW cli $S screenshot --filename={runDir}/proof/health/health-page-node-offline/back.png
  $FW doctor $RUN
  ```

- **Set the levels on the page.** Open the Storage section's Details. A warning of 2% saves; an error of 5% is refused beside the field with `No more than the warning level, 2%.`; a warning of 100% saves and raises a Storage warning row on home's volume, and the nav row reads `Health needs attention`. Reset both levels to `0B` through the API after.

  ```sh
  $FW exec $RUN -- $PW cli $S click 'getByRole("region", { name: "Storage" }).getByRole("button", { name: "Details" })'
  $FW exec $RUN -- $PW cli $S click 'getByRole("region", { name: "Storage" }).getByRole("radio", { name: "%" }).first()'
  $FW exec $RUN -- $PW cli $S fill 'getByRole("textbox", { name: "Warn when free space is below" })' 2
  $FW exec $RUN -- $PW cli $S click 'getByRole("region", { name: "Storage" }).getByRole("button", { name: "Save" })'
  $FW exec $RUN -- $PW cli $S click 'getByRole("region", { name: "Storage" }).getByRole("radio", { name: "%" }).nth(1)'
  $FW exec $RUN -- $PW cli $S fill 'getByRole("textbox", { name: "Show an error when free space is below" })' 5
  $FW exec $RUN -- $PW cli $S click 'getByRole("region", { name: "Storage" }).getByRole("button", { name: "Save" })'
  $FW exec $RUN --proof health/health-page-levels -- $PW cli $S screenshot --filename={runDir}/proof/health/health-page-levels/refused.png
  $FW exec $RUN -- $PW cli $S fill 'getByRole("textbox", { name: "Warn when free space is below" })' 100
  $FW exec $RUN -- $PW cli $S click 'getByRole("region", { name: "Storage" }).getByRole("button", { name: "Save" }).first()'
  $FW exec $RUN -- $PW cli $S run-code 'async page => { await page.getByRole("link", { name: /^Free up space on / }).waitFor(); }'
  $FW exec $RUN --proof health/health-page-levels -- $PW cli $S snapshot --filename={runDir}/proof/health/health-page-levels/warning.aria.yml
  $FW exec $RUN --proof health/health-page-levels -- $PW cli $S screenshot --filename={runDir}/proof/health/health-page-levels/warning.png
  $FW exec $RUN -- curl -sS -X PATCH -H 'X-Api-Key: {apiKey}' -H 'Content-Type: application/json' --data-binary '{"warnBelow":{"amount":0,"unit":"bytes"},"errorBelow":{"amount":0,"unit":"bytes"}}' '{nodes.home.url}/api/v1/system/health/settings'
  ```

- **Give a volume its own levels.** In Library roots, open `Actions for <volume>` on `Volumes on home` (`/` on Linux; read `name` from `GET /api/v1/system/health/volumes`), then `Set warning levels`. Set the warning to 100% and choose `Done`: the row reads `Own levels` and `Warning`, and the tab reads `Library roots 1 warning`. Open the dialog again and choose `Follow the global levels`.

  ```sh
  $FW exec $RUN -- $PW cli $S goto '{nodes.home.browserUrl}/settings/library-roots'
  $FW exec $RUN -- $PW cli $S click 'getByRole("button", { name: "Actions for /", exact: true })'
  $FW exec $RUN -- $PW cli $S click 'getByRole("menuitem", { name: "Set warning levels" })'
  $FW exec $RUN -- $PW cli $S click 'getByRole("dialog", { name: "Warning levels for /" }).getByRole("radio", { name: "%" }).first()'
  $FW exec $RUN -- $PW cli $S fill 'getByRole("dialog", { name: "Warning levels for /" }).getByRole("textbox", { name: "Warn when free space is below" })' 100
  $FW exec $RUN -- $PW cli $S click 'getByRole("dialog", { name: "Warning levels for /" }).getByRole("button", { name: "Save" })'
  $FW exec $RUN --proof health/health-volume-levels -- $PW cli $S screenshot --filename={runDir}/proof/health/health-volume-levels/dialog.png
  $FW exec $RUN -- $PW cli $S click 'getByRole("button", { name: "Done" })'
  $FW exec $RUN -- $PW cli $S run-code 'async page => { await page.getByRole("link", { name: "Library roots 1 warning" }).waitFor(); }'
  $FW exec $RUN --proof health/health-volume-levels -- $PW cli $S snapshot --filename={runDir}/proof/health/health-volume-levels/own.aria.yml
  $FW exec $RUN --proof health/health-volume-levels -- $PW cli $S screenshot --filename={runDir}/proof/health/health-volume-levels/own.png
  $FW exec $RUN -- $PW cli $S click 'getByRole("button", { name: "Actions for /", exact: true })'
  $FW exec $RUN -- $PW cli $S click 'getByRole("menuitem", { name: "Set warning levels" })'
  $FW exec $RUN -- $PW cli $S click 'getByRole("button", { name: "Follow the global levels" })'
  $FW exec $RUN -- $PW cli $S click 'getByRole("button", { name: "Done" })'
  $FW exec $RUN --proof health/health-volume-levels -- $PW cli $S screenshot --filename={runDir}/proof/health/health-volume-levels/global.png
  ```

- **The settings row on low space.** On the `low-space` run, Settings > Library roots carries `Library roots 1 error` and opens with the volume's Error line; `Open Health` lands on the item, whose row unfolds its facts and `Check now: <volume> on home`. On phone, the Settings list row carries the dot, and the line opens a sheet with `Open Health`.

  ```sh
  $FW exec $RUN -- $PW cli $S goto '{nodes.home.browserUrl}/settings/library-roots'
  $FW exec $RUN -- $PW cli $S run-code 'async page => { await page.getByRole("link", { name: "Library roots 1 error" }).waitFor({ timeout: 60000 }); }'
  $FW exec $RUN --proof health/health-settings-row -- $PW cli $S snapshot --filename={runDir}/proof/health/health-settings-row/section.aria.yml
  $FW exec $RUN --proof health/health-settings-row -- $PW cli $S screenshot --filename={runDir}/proof/health/health-settings-row/section.png
  $FW exec $RUN -- $PW cli $S click 'getByRole("button", { name: "Open Health" })'
  $FW exec $RUN -- $PW cli $S click 'getByRole("region", { name: "Storage" }).getByRole("button").filter({ hasText: "Free up space on" })'
  $FW exec $RUN --proof health/health-settings-row -- $PW cli $S snapshot --filename={runDir}/proof/health/health-settings-row/item.aria.yml
  $FW exec $RUN --proof health/health-settings-row -- $PW cli $S screenshot --filename={runDir}/proof/health/health-settings-row/item.png
  $FW exec $RUN -- $PW cli -s={runId}-admin-phone goto '{nodes.home.browserUrl}/settings'
  $FW exec $RUN --proof health/health-settings-row -- $PW cli -s={runId}-admin-phone screenshot --filename={runDir}/proof/health/health-settings-row/phone-list.png
  $FW exec $RUN -- $PW cli -s={runId}-admin-phone click 'getByRole("link", { name: "Library roots 1 error" })'
  $FW exec $RUN -- $PW cli -s={runId}-admin-phone click 'getByRole("main").getByRole("button").filter({ hasText: "on home" }).first()'
  $FW exec $RUN --proof health/health-settings-row -- $PW cli -s={runId}-admin-phone screenshot --filename={runDir}/proof/health/health-settings-row/phone-sheet.png
  ```

## Gotchas

- The offline grace's floor is 60 s, and the blip is 1 min too, so with the grace at 60 s seedbox goes from in step straight to a `fail` item and never reads `warn`. The item's `since` is the moment past the grace; when seedbox went away is its `Offline since` fact.
- Home keeps covering Searching until seedbox has settled, about a minute after `node start`, so the Nodes category reads `warn` through that minute after seedbox's own item has gone. Poll `/counts` for `null` rather than reading at once.
- `unreached` from a category's Check now has no `key`: it names the whole category, so read `words`, not a key.
- The torrent listen setting refuses a port the Node's own web or link listener uses (422 `port_used_by_node`), so hold the port with another program. Read seedbox's torrent listener from the manifest before moving it: a restart that can't bind leaves it empty there.
- Download Clients has no Check now: its items follow the clients' facts and the downloads volume samples, which Storage's Check now refreshes.
- A process world's Nodes report the host's own disk, so a catalog Scenario sets both Storage levels to 0B (`library.storage_levels`), which never fires. A level in bytes saved on `home` judges the host's disk and may or may not fire; use 100% to raise a warning on any host, and `low-space`, whose levels are 1024 TB, for an error. The disk faults (`volume=…`, `full`, `unmounted`) play only in the simulation tier.
- Each levels save sends `health.changed` naming `storage`, so wait on `nodes` by reading the event's data, not by its name alone.
- A section with an item replaces its quiet line, so a Storage warning raised from the Details folds them again; open Details again to keep editing.
- The Health page keeps sections and rows in the order it first showed them while it is open, so Nodes moves first only on a reload; a reload sorts them errors first, then oldest.
- The 422 for an inverted pair compares levels only in one unit. Mixed units such as 50 GB over 5% save, because they can't be compared without a volume.
