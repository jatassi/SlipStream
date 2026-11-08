# Trackers and seeding goals

Settings > Trackers holds the default seeding goal, the margin past a rule and the warning delay, then every Tracker with whether it is private, its rule, whether it follows the default goal, its torrents, how many still owe it seeding and a warning dot for torrents at risk. Each Tracker's page leads with its rule, where it stands with it and its totals, then a Torrents tab of only its torrents, a Seeding tab that edits its rule and its own goal, and an Indexers tab of the Indexers that reach it. Its menu renames, merges and removes it, and Remove says what still uses a Tracker it refuses. On phone the page lists Seeding and Indexers as rows above its torrents, and each is a pushed page.

## Sub-features

- `trackers-list` on `indexers`, Settings > Trackers lists every Tracker, each row `<privacy> · Rule <rule> · Default goal` with its torrents, and the row opens `/settings/trackers/<id>`.
- `trackers-default-goal` the `Seeding goal` form under `Default goal` saves the whole goal: `GET /api/v1/trackers/default-goal` answers what was saved, and `Forever` sends `ratio`, `seedSeconds` and `action` as `null`.
- `trackers-settings` `Margin past a rule` and `Warn when a torrent that owes hasn't announced for` save by key through `PATCH /api/v1/trackers/settings`.
- `tracker-page` the page reads `<name> requires … for each torrent.` with `From <name>'s Releases` or `Set by you`, the standing sentence with the Tracker's keys and hosts under `Details`, and its totals; the `Torrents` tab lists only the torrents whose Tracker it is.
- `tracker-seeding` the Seeding tab's `Seeding rule` form saves the whole rule and reads `Set by you`; `Use what <name> states` drops it. Its `Seeding goal` form saves the Tracker's own goal and `Use the default goal` drops it. A refused value reads under the field that `seeding_value_refused` names.
- `tracker-remove` `<name> actions`, `Remove…` and `Remove` on a Tracker an Indexer still reaches reads `<name> is still in use. Delete its Indexer … first.` and keeps it; on one nothing uses it goes back to Settings > Trackers without it.

## How to get to it (user POV)

- Desktop shell: the sidebar's `Settings`, then `Trackers` at `/settings/trackers`, between Indexers and Download Clients. A Tracker's heading on Settings > Indexers links to its page at `/settings/trackers/<id>`, whose tabs are `Torrents`, `Seeding` (`/seeding`) and `Indexers` (`/indexers`); a torrent opens its Inspector at `/settings/trackers/<id>/torrents/<torrentId>`.
- Phone shell: `More`, `Settings`, `Trackers`. A Tracker's page lists `Seeding` and `Indexers` as rows above its torrents; each is a pushed page whose back link names the Tracker.
- API: `GET /api/v1/trackers`, `GET /api/v1/trackers/{id}`, `PUT` and `DELETE /api/v1/trackers/{id}/rule`, `PUT` and `DELETE /api/v1/trackers/{id}/goal`, `GET` and `PUT /api/v1/trackers/default-goal`, `GET` and `PATCH /api/v1/trackers/settings`, `DELETE /api/v1/trackers/{id}`.

## Driving it with fakeworld and playwright cli

Preconditions:

- An `indexers` run with doctor passing and the `{runId}-admin-desktop` session.

Steps:

- **The list and a Tracker's page.** Open Settings > Trackers, then the first private Tracker's row.

  ```sh
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop goto '{nodes.home.browserUrl}/settings/trackers'
  $FW exec $RUN --proof trackers/trackers-list -- $PW cli -s={runId}-admin-desktop screenshot --filename={runDir}/proof/trackers/trackers-list/list.png
  ```

- **Set the rule and the goal.** On the Seeding tab choose `Either one`, `Ratio` `1`, `Seeding time` `8` and `Save rule`, then the goal's `Ratio` `3` and `Save goal`, and read the Tracker back.

  ```sh
  $FW exec $RUN -- curl -sk -H 'X-Api-Key: {apiKey}' '{nodes.home.url}/api/v1/trackers'
  ```

  The Tracker answers `rule` `{"clears":"either","ratio":1,"seedSeconds":691200,"source":"admin"}` and its own `goal`.

## Gotchas

- The Torrents tab counts and filters the one live Torrents list by each row's `tracker`, so a torrent with no known Tracker shows on no Tracker's page.
- The windows list, the routes and the Health `seeding` category aren't on this branch, so the Seeding tab shows no open windows and the page has no Routing tab.
