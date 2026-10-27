# Indexers and Trackers

Settings > Indexers lists every Indexer under each Tracker it reaches, with its status sentence and its On, Automatic search and RSS switches. Every Tracker shows as a group, also one no Indexer reaches any more. A group leads with whether the Tracker is private and holds its keys and announce hosts under Details. Its menu renames the Tracker or merges it into another Tracker that is the same site, and both stick. An Indexer's page leads with its status sentence and the fix, then what it reaches, where it searches from, its limits with how much is used and its switches, then its Recent queries and Recent grabs, with its session, backoff and wire facts under Details. Edit, Test, Move and Delete sit under the lead; Test shows a captcha its sign-in meets, and Move asks for what the new home needs. A Searching Node's page sets its Challenge solver, saved only once the Node reaches it.

## Sub-features

- `trackers-groups` on `home`, Settings > Indexers reads `Searches ask all 2 Indexers that are on.` once an RSS Indexer naming the Tracker `Fixture RSS` is added beside Fixture Site, with the groups `Fixture RSS` and `Fixture Site`, each led by `Its first torrent shows whether <name> is private.` and a `Details` holding `Keys` (`named:<Indexer id>`) and `Announce hosts` `None`.
- `trackers-merge` `Fixture RSS actions`, `Merge into…` opens `Merge Fixture RSS` with `Merge into` offering only `Fixture Site`; `Merge` leaves one group, `Fixture Site`, listing both Indexers, and `GET /api/v1/trackers` holds one Tracker with both keys.
- `trackers-unreached` deleting the only Indexer of a Tracker keeps the Tracker: its group reads `No Indexer reaches this Tracker. Its Grabs and torrents stay with it. If another Tracker is the same site, merge this one into it.` with no rows.
- `trackers-rename` `Rename` with another Tracker's name, in any case or punctuation, reads `Another Tracker has that name` on the field and keeps the dialog open; a free name saves and the group takes it.
- `indexer-edit-test` on `indexers`, Quarry's page, `Edit`, `Priority` `5` and `Save` close the dialog and `GET /api/v1/indexers/{id}` answers `priority` `5`; `Test` reads `Quarry passed the Test and listed <n> items.` (P-107, P-121)
- `indexer-stop-back` on `indexers`, `fixture-site-refuse` `{"code": 101}` and `Test` on Fixture Site's page read `Fixture Site refused the API key with 'Account suspended'. Check the account, then test again.` in an alert and lead with `Stopped because Fixture Site refused the API key with 'Account suspended'.`; `{"code": 0}` and `Test` again read `Fixture Site passed the Test and listed <n> items.` and the stop line is gone (P-114, P-120)
- `indexer-page-captcha` on `indexers`, `fixture-site-expire` `{"site": "larkspur"}` signs Larkspur out; `Test` on its page shows the image `The captcha Larkspur shows` and a `Code` field, and `Test` with the code from the `captchas` capture reads `Larkspur passed the Test and listed <n> items.` (P-111)
- `indexer-move-cookie` on `indexers`, Cinder's cookie was pasted for seedbox; `Move` with `Search from` `home` reads `The browser cookie pasted for Cinder works only from the Node it was pasted for. Paste the cookie of a browser signed in to Cinder on home's network.`, and `Move` with no cookie reads `Paste the cookie of a browser signed in to Cinder.` (P-111)
- `indexer-solver` on `indexers`, home's Searching page saves `Challenge solver` `http://127.0.0.1:9` only to read `home can't reach a Challenge solver at http://127.0.0.1:9. Check the address and that the solver is running beside home.`; seedbox's solver address reads `Saved home's Challenge solver` and shows `Remove`
- `indexer-limits` on `indexers`, PATCH Quarry `{"priority":1,"searches":{"count":2,"per":"hour"},"grabs":{"count":1,"per":"day"}}`; the first search grabs from Quarry, the second lists Quarry `Found 1 Release` yet grabs from another Indexer, and the third reads Quarry `Limit reached until HH:MM`. Quarry's page reads `2 of 2 per hour` and `1 of 1 per day`, and the counts hold across a restart of home (P-115)
- `indexer-backoff` on `indexers`, `fixture-site-maintenance` `{"site": "quarry", "on": true}` makes each search past Quarry's backoff read `Isn't responding. SlipStream retries at HH:MM.` and climb a rung of the 1 s, 2 s, 4 s ladder (`status.rung`); out of maintenance the next search reads `Found 1 Release` and steps down one rung (P-114)
- `indexer-grabs` on `indexers`, after a search grabs, the Indexer it grabbed from lists the Release under `Recent grabs` with its stage, linking to the Work, and `GET /api/v1/indexers/{id}/grabs` answers it with `nextOffset` `null`; the Grab's `fetchNodeId` is that Indexer's home, which fetched the .torrent. Deleting the Indexer keeps the Grab with `indexerId` `null` under its Tracker (P-117, P-131)
- `indexer-catalog-update` on `indexers`, `indexer-catalog-update` and Run now on `indexer-catalog-refresh` move Larkspur to the new `definition.version`; `indexer-catalog-bad-update` and Run now keep it and raise the Indexers `warn` `definition.<id>` `The catalog's new version of Larkspur doesn't load. The Indexer keeps running the version from …`, linking to its page (P-110)
- `indexer-categories` an Indexer with only non-movie categories, such as PATCH `{"categories":[5000]}`, reads `Skipped because it has no movie categories` on a movie search's line (P-113)

## How to get to it (user POV)

- Desktop shell: the sidebar's `Settings`, then `Indexers` at `/settings/indexers`. Each Tracker group's `<name> actions` menu holds `Rename` and, while another Tracker exists, `Merge into…`. An Indexer's row opens its page at `/settings/indexers/<id>`. Phone shell: `More`, `Settings`, `Indexers`; the menu is an action sheet.
- A Node's Challenge solver: `Nodes`, the Node, `Searching` at `/nodes/<id>/searching`. Only a Node that holds Searching has the section.
- API: `GET /api/v1/trackers`, `GET` and `PATCH /api/v1/trackers/{id}`, `POST /api/v1/trackers/{id}/merge` with `into`; `POST /api/v1/indexers`, `GET`, `PATCH` and `DELETE /api/v1/indexers/{id}`, `POST /api/v1/indexers/{id}/test`, `POST /api/v1/indexers/{id}/captcha`, `GET /api/v1/indexers/{id}/queries` and `GET /api/v1/indexers/{id}/grabs` (50 a page, `?offset=`); `GET` and `PUT /api/v1/nodes/{id}/challenge-solver`.

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

### The `indexers` Scenario

Preconditions:

- An `indexers` run with doctor passing and the `{runId}-admin-desktop` session. home and seedbox hold Searching; Quarry, Larkspur, Marsh and Cinder search from seedbox, which alone has a Challenge solver, and the Torznab feed Fixture Site and an RSS feed search from home. Add The Quiet Orchard (`9000001`) as in [add-a-movie.md](./add-a-movie.md) before a search step. Read an Indexer's id by name:

  ```sh
  IX() { curl -sk -H 'X-Api-Key: {apiKey}' '{nodes.home.url}/api/v1/indexers' | jq -r --arg n "$1" '.indexers[] | select(.name == $n) | .id'; }
  ```

Steps:

- **Edit and test.** Open Quarry's page, `Edit`, fill `Priority` with `5`, `Save`, then `Test`.

  ```sh
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop goto "{nodes.home.browserUrl}/settings/indexers/$(IX Quarry)"
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop click 'getByRole("button", { name: "Edit", exact: true })'
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop fill 'getByRole("dialog").getByRole("textbox", { name: "Priority" })' '5'
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop click 'getByRole("dialog").getByRole("button", { name: "Save" })'
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop click 'getByRole("button", { name: "Test", exact: true })'
  $FW exec $RUN --proof indexers/indexer-edit-test -- $PW cli -s={runId}-admin-desktop screenshot --filename={runDir}/proof/indexers/indexer-edit-test/tested.png
  ```

- **A stop and its return.** Have the site refuse Fixture Site's account, `Test` on its page, then serve again and `Test`.

  ```sh
  $FW cue $RUN fixture-site-refuse '{"code": 101}'
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop goto "{nodes.home.browserUrl}/settings/indexers/$(IX 'Fixture Site')"
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop click 'getByRole("button", { name: "Test", exact: true })'
  $FW exec $RUN --proof indexers/indexer-stop-back -- $PW cli -s={runId}-admin-desktop screenshot --filename={runDir}/proof/indexers/indexer-stop-back/stopped.png
  $FW cue $RUN fixture-site-refuse '{"code": 0}'
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop click 'getByRole("button", { name: "Test", exact: true })'
  ```

- **A captcha on the page.** Sign Larkspur out, `Test`, read the code, fill `Code` and `Test` again.

  ```sh
  $FW cue $RUN fixture-site-expire '{"site": "larkspur"}'
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop goto "{nodes.home.browserUrl}/settings/indexers/$(IX Larkspur)"
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop click 'getByRole("button", { name: "Test", exact: true })'
  $FW captchas wait $RUN --where site=larkspur --timeout 30s
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop fill 'getByRole("textbox", { name: "Code" })' '<text>'
  $FW exec $RUN --proof indexers/indexer-page-captcha -- $PW cli -s={runId}-admin-desktop screenshot --filename={runDir}/proof/indexers/indexer-page-captcha/captcha.png
  ```

- **Move Cinder.** `Move`, choose `home` under `Search from`, then `Move` with the cookie empty.

  ```sh
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop goto "{nodes.home.browserUrl}/settings/indexers/$(IX Cinder)"
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop click 'getByRole("button", { name: "Move", exact: true })'
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop click 'getByRole("dialog").getByRole("combobox", { name: "Search from" })'
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop click 'getByRole("option", { name: "home" })'
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop click 'getByRole("dialog").getByRole("button", { name: "Move", exact: true })'
  $FW exec $RUN --proof indexers/indexer-move-cookie -- $PW cli -s={runId}-admin-desktop screenshot --filename={runDir}/proof/indexers/indexer-move-cookie/cookie.png
  ```

- **home's Challenge solver.** Save an address home can't reach, then seedbox's.

  ```sh
  curl -sk -H 'X-Api-Key: {apiKey}' '{nodes.home.url}/api/v1/nodes/{nodes.seedbox.nodeId}/challenge-solver'
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop goto '{nodes.home.browserUrl}/nodes/{nodes.home.nodeId}/searching'
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop fill 'getByRole("textbox", { name: "Address" })' 'http://127.0.0.1:9'
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop click 'getByRole("button", { name: "Save" })'
  $FW exec $RUN --proof indexers/indexer-solver -- $PW cli -s={runId}-admin-desktop screenshot --filename={runDir}/proof/indexers/indexer-solver/unreachable.png
  ```

- **Limits, backoff and Recent grabs.** PATCH Quarry's limits, search the Work three times from its page, and read Quarry's page; put Quarry in maintenance for the backoff. The Go tests `TestSearchAndGrabLimitsHoldAcrossAControllerRestart`, `TestAFailingIndexerClimbsTheLadderAndStepsDownOnASuccess` and `TestDeletingAnIndexerKeepsItsGrabsUnderItsTracker` in `e2e/` drive the same steps.

  ```sh
  curl -sk -X PATCH -H 'X-Api-Key: {apiKey}' -H 'Content-Type: application/json' -d '{"priority":1,"searches":{"count":2,"per":"hour"},"grabs":{"count":1,"per":"day"}}' "{nodes.home.url}/api/v1/indexers/$(IX Quarry)"
  $FW cue $RUN fixture-site-maintenance '{"site": "quarry", "on": true}'
  $FW exec $RUN --proof indexers/indexer-grabs -- curl -sk -H 'X-Api-Key: {apiKey}' "{nodes.home.url}/api/v1/indexers/$(IX Quarry)/grabs"
  ```

- **A catalog update.** Cue an update, Run now on the catalog refresh and read Larkspur's version; then a bad update.

  ```sh
  $FW cue $RUN indexer-catalog-update '{}'
  curl -sk -X POST -H 'X-Api-Key: {apiKey}' '{nodes.home.url}/api/v1/tasks/indexer-catalog-refresh/runs'
  $FW exec $RUN --proof indexers/indexer-catalog-update -- curl -sk -H 'X-Api-Key: {apiKey}' "{nodes.home.url}/api/v1/indexers/$(IX Larkspur)"
  $FW cue $RUN indexer-catalog-bad-update '{}'
  curl -sk -X POST -H 'X-Api-Key: {apiKey}' '{nodes.home.url}/api/v1/tasks/indexer-catalog-refresh/runs'
  $FW exec $RUN --proof indexers/indexer-catalog-update -- curl -sk -H 'X-Api-Key: {apiKey}' '{nodes.home.url}/api/v1/system/health'
  ```

- **The browser spec.** `web/e2e/indexers.spec.ts` drives edit and test, the stop and return, the page captcha, Cinder's Move and home's Challenge solver at both widths.

  ```sh
  cd web && TEST_SLIPSTREAM_BINARY=../dist/slipstream TEST_FAKEWORLD_BINARY=../dist/fakeworld bunx playwright test -c e2e indexers.spec.ts --project=chromium-desktop --project=chromium-phone
  ```

## Gotchas

- The RSS feed is served on the Torznab host over https, `https://torznab.fixture-site.fake.localhost/rss`; over http the feed's Test fails with status 400.
- A Tracker's name is taken by another that differs only in case or punctuation: `fixture site` beside `Fixture Site` reads `Another Tracker has that name`.
- Each captcha is answered once. A Test that fails hands back a new image, so read the newest `captchas` record for the site.
- The `indexers` Scenario's backoff ladder is 1 s, 2 s, 4 s, so wait out `status.until` before the next search or it reads Quarry's backoff instead of asking the site.
- A Move's Test runs from the new home, so a cookie pasted for seedbox fails from home: paste one from home's network.
- `exec` fills `{...}` tokens, so a curl `-w '%{http_code}'` fails as an unknown token; write `%{{http_code}`.
- Name a group's menu with `exact: true` once two Tracker names share words, or the locator matches both.
