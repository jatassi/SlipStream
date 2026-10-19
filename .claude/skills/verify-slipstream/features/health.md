# Health

Health is one report of what needs a look, built from the features that already know: an Admin or a script reads it at `GET /api/v1/system/health`, by category in page order (Storage, Nodes, Scheduled tasks), each item with its level, subject, what to do first, its facts, its one action and how long it has stood. Nodes says when a Worker is offline past the grace and when another Node covers its Roles; Scheduled tasks says when a task failed. `GET /api/v1/counts` carries the worst level as `alerts.health` and Scheduled tasks' as `alerts.tasks`, and the live stream says which categories changed with `health.changed`. Check now re-asks a category and says what it couldn't reach without touching those items. The Storage levels, `Warn when free space is below` and `Show an error when free space is below`, take bytes or a percent of the volume. The Health page, the warnings on Settings rows and the Storage category's volumes arrive with later waves; this file covers the API until then.

## Sub-features

- `health-report` with nothing wrong, `GET /api/v1/system/health` answers `level` `null`, `first` `null`, and the categories `storage`, `nodes` and `scheduled-tasks` in that order, each with its `label`, `phrase`, `level` `null`, `canCheck` (`true` for Storage and Nodes) and `items` `[]`: what the page's `Everything is working.` and `Checked storage, Nodes and scheduled tasks` are read from. `GET /api/v1/counts` answers `alerts.health` and `alerts.tasks` `null` (P-213).
- `health-node-offline` with seedbox stopped past the offline grace, Nodes holds a `fail` item keyed by seedbox's Node id, subject `seedbox`, `seedbox has been offline past the offline grace. Start SlipStream on it, or remove it if it is gone.`, the fact `Offline since` and a `link` action `Open seedbox` to `/nodes/<id>`, and a `warn` item `covering.searching`, subject `Searching on home`, `home searches in place of seedbox. Bring seedbox back, or give Searching to another Node.`; the report's `level` is `fail` and `first` is seedbox's item, `/counts` answers `alerts.health` and `alerts.nodes` `fail`, and the stream carries `health.changed` with `{"categories":["nodes"]}`. Once seedbox starts its `fail` item clears at once while home's covering warning stays and both alerts read `warn`; once seedbox settles the warning clears too and the alerts are `null` again, each change with a `health.changed` naming `nodes` (P-213).
- `health-check-now` `POST /api/v1/system/health/nodes/check` answers `report` and `unreached`: `[]` with every Node in step, and `[{"words":"seedbox is offline."}]` with seedbox away, its item's `since` unchanged. `POST /api/v1/system/health/check` checks every category the same way. An item with a link, and a category with no Check now, answer 404 `no_check`, and an unknown category 404 `health_category_not_found` (P-214).
- `health-levels-api` `GET /api/v1/system/health/settings` answers `warnBelow` and `errorBelow` as space fields, 100 GB and 20 GB in bytes by default; `PATCH` takes `{"amount":<n>,"unit":"bytes"}` or `"unit":"percent"`, and units may mix. An error level above a warning level in the same unit answers 422 `setting_out_of_range` with `No more than the warning level, <level>.`, whether both are in the request or the warning is the stored one, and stores nothing. `null` resets a level to its default. Each saved change carries `health.changed` naming `storage` (P-216).

## How to get to it (user POV)

- Desktop and phone shells: none yet; the Health page and its Settings rows arrive with a later wave.
- API (Admin): `GET /api/v1/system/health`, `POST /api/v1/system/health/check`, `POST /api/v1/system/health/{category}/check`, `POST /api/v1/system/health/{category}/items/{key}/check`, `GET` and `PATCH /api/v1/system/health/settings`, and `GET /api/v1/counts` for `alerts.health` and `alerts.tasks`; the live event `health.changed`.
- Host CLI: none.

## Driving it with fakeworld and curl

Preconditions:

- A fresh `home` run with doctor passing. No browser session is needed. Run the steps in order: the offline steps stop and start seedbox, and take about three minutes of waiting.

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

- **Set the levels in bytes and percent.** The warning saves as 50 GB and the error as 5%. A same-unit error above the warning, 15% over 10%, answers 422 `No more than the warning level, 10%.` and the read still shows 50 GB and 5%. Reset both with `null`, then an error of 200 GB over the default 100 GB answers 422 `No more than the warning level, 100 GB.`. The stream carries `health.changed` naming `storage` for the two saves.

  ```sh
  $FW exec $RUN --proof health/health-levels-api -- curl -sS -H 'X-Api-Key: {apiKey}' '{nodes.home.url}/api/v1/system/health/settings'
  $FW exec $RUN --proof health/health-levels-api -- curl -sS -X PATCH -H 'X-Api-Key: {apiKey}' -H 'Content-Type: application/json' --data-binary '{"warnBelow":{"amount":53687091200,"unit":"bytes"},"errorBelow":{"amount":5,"unit":"percent"}}' '{nodes.home.url}/api/v1/system/health/settings'
  $FW exec $RUN --proof health/health-levels-api -- curl -sS -i -X PATCH -H 'X-Api-Key: {apiKey}' -H 'Content-Type: application/json' --data-binary '{"warnBelow":{"amount":10,"unit":"percent"},"errorBelow":{"amount":15,"unit":"percent"}}' '{nodes.home.url}/api/v1/system/health/settings'
  $FW exec $RUN --proof health/health-levels-api -- curl -sS -H 'X-Api-Key: {apiKey}' '{nodes.home.url}/api/v1/system/health/settings'
  $FW exec $RUN --proof health/health-levels-api -- curl -sS -X PATCH -H 'X-Api-Key: {apiKey}' -H 'Content-Type: application/json' --data-binary '{"warnBelow":null,"errorBelow":null}' '{nodes.home.url}/api/v1/system/health/settings'
  $FW exec $RUN --proof health/health-levels-api -- curl -sS -X PATCH -H 'X-Api-Key: {apiKey}' -H 'Content-Type: application/json' --data-binary '{"errorBelow":{"amount":214748364800,"unit":"bytes"}}' '{nodes.home.url}/api/v1/system/health/settings'
  $FW exec $RUN --proof health/health-levels-api -- $FW sse list {runDir} --as admin --event health.changed
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

## Gotchas

- The offline grace's floor is 60 s, and the blip is 1 min too, so with the grace at 60 s seedbox goes from in step straight to a `fail` item and never reads `warn`. The item's `since` is the moment past the grace; when seedbox went away is its `Offline since` fact.
- Home keeps covering Searching until seedbox has settled, about a minute after `node start`, so the Nodes category reads `warn` through that minute after seedbox's own item has gone. Poll `/counts` for `null` rather than reading at once.
- `unreached` from a category's Check now has no `key`: it names the whole category, so read `words`, not a key.
- Storage is listed with `canCheck` `true` and no items at this head, because no Node reports volumes to it yet. Its levels still save, and each save sends `health.changed` naming `storage`, so wait on `nodes` by reading the event's data, not by its name alone.
- The 422 for an inverted pair compares levels only in one unit. Mixed units such as 50 GB over 5% save, because they can't be compared without a volume.
