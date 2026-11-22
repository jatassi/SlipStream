# Home

An Admin's Home reads the moving Grabs nearest Available first, what needs the Admin (failed Grabs and Grabs that wait), and the newest finished Imports in words, with today's totals from the browser's midnight. Each Downloading Node's Torrents are counted for its card. `Rows on Home` sets how many rows each list shows. This PR adds the API; the page comes later.

## Sub-features

- `home-summary` `GET /home?since=` answers `jobs` (its rows and `moving`, every moving Grab), `needsYou` (rows newest first, `failed` counted up to 100, `waiting`) and `recent` (each Import's `words`, such as `Imported 10 episodes · 2 upgrades`, and `today`'s `imported` and `upgraded`), each row with its Work's label, artwork URLs and `failed`; a job's `torrent` is null until the .torrent is fetched.
- `home-counts` `GET /torrents/counts` lists each Downloading Node in Node order with `torrents`, `downloading`, `seeding`, `failures` and `warnings`.
- `home-settings` `GET /home-settings` lists `Rows on Home`; `PATCH /home-settings` with `{"values":{"home.rows":3}}` saves it, cuts every list of `GET /home` to 3 while the counts stay whole, and publishes `home.settings-changed`; a value out of range answers 422 at its key.

## How to get to it (user POV)

- Desktop shell: none yet.
- Phone shell: none yet.
- API (Admin, Controller only): `GET /api/v1/home?since=<RFC 3339>`, `GET /api/v1/torrents/counts`, `GET /api/v1/home-settings`, `PATCH /api/v1/home-settings`. A Worker answers 409 `worker_node`.
- Host CLI: none.

## Driving it with fakeworld and playwright cli

Preconditions:

- The `home` Scenario, doctor passing, after one Grab made as in [Search and grab](./search-and-grab.md).

Steps:

- **Read Home.** While the Grab moves it is in `jobs.rows` with its torrent; once imported, its Import leads `recent.rows` and `today.imported` is 1.

  ```sh
  $FW exec $RUN --proof home/home-summary -- curl -sS -H 'X-Api-Key: {apiKey}' "{nodes.home.url}/api/v1/home?since=$(date -u +%Y-%m-%dT00:00:00Z)"
  ```

- **Count Torrents.** The one row is seedbox's, the Node that holds Downloading, and counts the Grab's torrent as downloading, then seeding.

  ```sh
  $FW exec $RUN --proof home/home-counts -- curl -sS -H 'X-Api-Key: {apiKey}' '{nodes.home.url}/api/v1/torrents/counts'
  ```

- **Change Rows on Home.** Save 3, read Home again, then try 2 for the 422.

  ```sh
  $FW exec $RUN --proof home/home-settings -- curl -sS -X PATCH -H 'X-Api-Key: {apiKey}' -H 'Content-Type: application/json' -d '{"values":{"home.rows":3}}' '{nodes.home.url}/api/v1/home-settings'
  $FW exec $RUN --proof home/home-settings -- curl -sS -X PATCH -H 'X-Api-Key: {apiKey}' -H 'Content-Type: application/json' -d '{"values":{"home.rows":2}}' '{nodes.home.url}/api/v1/home-settings'
  ```

## Gotchas

- `since` is required and is the browser's midnight; a later instant makes `today` read 0.
- Torrent rows on Home carry `inLine` 0, because a read of a few Torrents can't place them in the queue.
- With one Grab every list holds one row, so saving 3 changes nothing a read can see. The cut shows only once a list has more than 3 rows.
