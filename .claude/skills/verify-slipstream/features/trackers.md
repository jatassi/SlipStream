# Trackers and seeding goals

Settings > Trackers holds the default seeding goal, the margin past a rule and the warning delay, then every Tracker with whether it is private, its rule, whether it follows the default goal, its torrents, how many still owe it seeding and a warning dot for torrents at risk. Each Tracker's page leads with its rule, where it stands with it and its totals, then a Torrents tab of only its torrents, a Seeding tab that edits its rule and its own goal, a Routing tab that picks the client a private Tracker's Grabs download on, and an Indexers tab of the Indexers that reach it. Its menu renames, merges and removes it, and Remove says what still uses a Tracker it refuses. On phone the page lists Seeding, Routing and Indexers as rows above its torrents, and each is a pushed page.

## Sub-features

- `trackers-list` on `indexers`, Settings > Trackers lists every Tracker, each row `<privacy> · Rule <rule> · Default goal` with its torrents, and the row opens `/settings/trackers/<id>`.
- `trackers-default-goal` the `Seeding goal` form under `Default goal` saves the whole goal: `GET /api/v1/trackers/default-goal` answers what was saved, and `Forever` sends `ratio`, `seedSeconds` and `action` as `null`.
- `trackers-settings` `Margin past a rule` and `Warn when a torrent that owes hasn't announced for` save by key through `PATCH /api/v1/trackers/settings`.
- `tracker-page` the page reads `<name> requires … for each torrent.` with `From <name>'s Releases` or `Set by you`, the standing sentence with the Tracker's keys and hosts under `Details`, and its totals; the `Torrents` tab lists only the torrents whose Tracker it is.
- `tracker-seeding` the Seeding tab's `Seeding rule` form saves the whole rule and reads `Set by you`; `Use what <name> states` drops it. Its `Seeding goal` form saves the Tracker's own goal and `Use the default goal` drops it. A refused value reads under the field that `seeding_value_refused` names.
- `tracker-routing` a private Tracker's `Routing` tab holds a section `Indexers on <home>` for each Node an Indexer reaching it searches from, with the select `Grabs go to` (each client that can take a Grab, the default marked `(default)`, or `Where the Work's library root is` while that Node doesn't hold Downloading) over `These Grabs go to the built-in client on <home>, the default.` and `The built-in client on <home> announces as libtorrent <version>.`; picking a client reads `These Grabs go to <client>, which you picked.` and `Use the default` drops the pick. A public Tracker's tab reads `<name> is public. Its Grabs download where the Work's library root is.`, and one read as unknown says SlipStream reads it from its first torrent.
- `tracker-remove` `<name> actions`, `Remove…` and `Remove` on a Tracker an Indexer still reaches reads `<name> is still in use. Delete its Indexer … first.` and keeps it; on one nothing uses it goes back to Settings > Trackers without it.

## How to get to it (user POV)

- Desktop shell: the sidebar's `Settings`, then `Trackers` at `/settings/trackers`, between Indexers and Download Clients. A Tracker's heading on Settings > Indexers links to its page at `/settings/trackers/<id>`, whose tabs are `Torrents`, `Seeding` (`/seeding`), `Routing` (`/routing`) and `Indexers` (`/indexers`); a torrent opens its Inspector at `/settings/trackers/<id>/torrents/<torrentId>`.
- Phone shell: `More`, `Settings`, `Trackers`. A Tracker's page lists `Seeding`, `Routing` and `Indexers` as rows above its torrents; each is a pushed page whose back link names the Tracker.
- API: `GET /api/v1/trackers`, `GET /api/v1/trackers/{id}`, `PUT` and `DELETE /api/v1/trackers/{id}/rule`, `PUT` and `DELETE /api/v1/trackers/{id}/goal`, `GET` and `PUT /api/v1/trackers/default-goal`, `GET` and `PATCH /api/v1/trackers/settings`, `GET /api/v1/trackers/{id}/routes`, `PUT` and `DELETE /api/v1/trackers/{id}/routes/{nodeId}` (`{"builtInOn": <nodeId>}` or `{"clientId": <id>}`), `DELETE /api/v1/trackers/{id}`.

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
- A Tracker reads `unknown` until a fetch shows whether its torrents are private, so its first Grab downloads by the Work's library root and only later Grabs follow its route. The `seeding-routing` Scenario plays this with Heron, whose Indexer searches from seedbox while the Movies root is on den.
- A Grab routed to a client whose Node is offline waits at fetching; [work-page.md](./work-page.md) `work-needs-you-routed` drives the send.
