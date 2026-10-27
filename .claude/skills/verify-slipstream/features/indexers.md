# Indexers and Trackers

Settings > Indexers lists every Indexer under each Tracker it reaches, with its status sentence and its On, Automatic search and RSS switches. Every Tracker shows as a group, also one no Indexer reaches any more. A group leads with whether the Tracker is private and holds its keys and announce hosts under Details. Its menu renames the Tracker or merges it into another Tracker that is the same site, and both stick.

## Sub-features

- `trackers-groups` on `home`, Settings > Indexers reads `Searches ask all 2 Indexers that are on.` once an RSS Indexer naming the Tracker `Fixture RSS` is added beside Fixture Site, with the groups `Fixture RSS` and `Fixture Site`, each led by `Its first torrent shows whether <name> is private.` and a `Details` holding `Keys` (`named:<Indexer id>`) and `Announce hosts` `None`.
- `trackers-merge` `Fixture RSS actions`, `Merge into…` opens `Merge Fixture RSS` with `Merge into` offering only `Fixture Site`; `Merge` leaves one group, `Fixture Site`, listing both Indexers, and `GET /api/v1/trackers` holds one Tracker with both keys.
- `trackers-unreached` deleting the only Indexer of a Tracker keeps the Tracker: its group reads `No Indexer reaches this Tracker. Its Grabs and torrents stay with it. If another Tracker is the same site, merge this one into it.` with no rows.
- `trackers-rename` `Rename` with another Tracker's exact name reads `Another Tracker has that name` on the field and keeps the dialog open; a free name saves and the group takes it.

## How to get to it (user POV)

- Desktop shell: the sidebar's `Settings`, then `Indexers` at `/settings/indexers`. Each Tracker group's `<name> actions` menu holds `Rename` and, while another Tracker exists, `Merge into…`. Phone shell: `More`, `Settings`, `Indexers`; the menu is an action sheet.
- API: `GET /api/v1/trackers`, `GET` and `PATCH /api/v1/trackers/{id}`, `POST /api/v1/trackers/{id}/merge` with `into`; `POST` and `DELETE /api/v1/indexers`.

## Driving it with fakeworld and playwright cli

Preconditions:

- A `home` run with doctor passing and the `{runId}-admin-desktop` session. The Fake world adds the Torznab Indexer Fixture Site on seedbox, which reaches the Tracker `Fixture Site`.

Steps:

- **A second Tracker.** Add an RSS Indexer on Fixture Site's feed that names its own Tracker, then open Settings > Indexers and a group's Details.

  ```sh
  $FW exec $RUN -- curl -sk -X POST -H 'X-Api-Key: {apiKey}' -H 'Content-Type: application/json' -d '{"kind":"rss","name":"Fixture Site RSS","address":"https://torznab.fixture-site.fake.localhost/rss","trackerName":"Fixture RSS","media":["movie"]}' '{nodes.home.url}/api/v1/indexers'
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop goto '{nodes.home.browserUrl}/settings/indexers'
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop click 'getByRole("region", { name: "Fixture Site", exact: true }).getByRole("button", { name: "Details" })'
  $FW exec $RUN --proof indexers/trackers-groups -- $PW cli -s={runId}-admin-desktop screenshot --filename={runDir}/proof/indexers/trackers-groups/groups.png
  ```

- **Merge.** Open `Fixture RSS actions`, `Merge into…`, and choose `Merge`. Reload: one group, `Fixture Site`, lists both Indexers.

  ```sh
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop click 'getByRole("button", { name: "Fixture RSS actions" })'
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop click 'getByRole("menuitem", { name: "Merge into…" })'
  $FW exec $RUN --proof indexers/trackers-merge -- $PW cli -s={runId}-admin-desktop screenshot --filename={runDir}/proof/indexers/trackers-merge/dialog.png
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop click 'getByRole("dialog").getByRole("button", { name: "Merge", exact: true })'
  $FW exec $RUN --proof indexers/trackers-merge -- curl -sk -H 'X-Api-Key: {apiKey}' '{nodes.home.url}/api/v1/trackers'
  ```

- **A Tracker no Indexer reaches.** Add the RSS Indexer `Old Mirror` naming the Tracker `Old Mirror` as above, delete it by its id, and reload Settings > Indexers: the `Old Mirror` group says no Indexer reaches it.

  ```sh
  $FW exec $RUN -- curl -sk -X DELETE -H 'X-Api-Key: {apiKey}' '{nodes.home.url}/api/v1/indexers/<Old Mirror id>'
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop goto '{nodes.home.browserUrl}/settings/indexers'
  $FW exec $RUN --proof indexers/trackers-unreached -- $PW cli -s={runId}-admin-desktop screenshot --filename={runDir}/proof/indexers/trackers-unreached/unreached.png
  ```

- **Rename.** Open `Old Mirror actions`, `Rename`, fill `Fixture Site` and choose `Rename`: the field reads `Another Tracker has that name`. Fill `Mirror` and choose `Rename`: the group reads `Mirror`.

  ```sh
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop click 'getByRole("button", { name: "Old Mirror actions", exact: true })'
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop click 'getByRole("menuitem", { name: "Rename" })'
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop fill 'getByRole("dialog").getByRole("textbox", { name: "Name" })' 'Fixture Site'
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop click 'getByRole("dialog").getByRole("button", { name: "Rename" })'
  $FW exec $RUN --proof indexers/trackers-rename -- $PW cli -s={runId}-admin-desktop screenshot --filename={runDir}/proof/indexers/trackers-rename/taken.png
  ```

## Gotchas

- The RSS feed is served on the Torznab host over https, `https://torznab.fixture-site.fake.localhost/rss`; over http the feed's Test fails with status 400.
- A Tracker's name is taken only by the exact same name: `fixture site` beside `Fixture Site` saves.
- `exec` fills `{...}` tokens, so a curl `-w '%{http_code}'` fails as an unknown token; write `%{{http_code}`.
- Name a group's menu with `exact: true` once two Tracker names share words, or the locator matches both.
