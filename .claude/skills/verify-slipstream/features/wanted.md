# Wanted

Wanted, under Library, lists every Work SlipStream is looking for, with its Items, why, when it last searched and when it searches next, and leads with a sentence saying what the queue is doing and what it waits on. `Search` on a row goes ahead of everything automatic and skips an Indexer at its limit, `Search N Works` queues every listed Work behind each Indexer's limit, and `Cancel queued searches` takes back what is still queued. An Indexer whose whole search limit RSS sync needs is named in the sentence, so a stalled queue is explained.

## Sub-features

- `wanted-rows` on `limited-tracker`, Wanted lists ten rows `Missing`, `Not searched yet`, with the next search time, a `Search <Work>` and `More for <Work>` per row, `Search 10 Works` and the status line `Nothing queued · Next Backlog search <time>`; on phone each row is a button reading `<Work> Missing. Next search <time>.` (P-053).
- `wanted-search-all` `Search 10 Works` queues them: the status reads `2 searching · 7 queued · Next Backlog search <time> ·` and the link `Waiting for Heron's search limit until <time>` to Heron's page, and the API's `status.limits` names Heron with its `until` (P-135).
- `wanted-row-search` `Search Copper Kettle` while it is queued toasts `Already searching Copper Kettle`, and it searches at once in the `now` lane: Fixture Site and Heron each read `No Releases`, spending Heron's second search this hour (P-037, P-132).
- `wanted-press-skips` `Search Nightfall Ferry` with Heron at its limit searches at once, Heron's line reading `Limit reached until <time>` (`limit_reached`) (P-132).
- `wanted-cancel` `Cancel queued searches` toasts `Canceled 5 queued searches.`, the status loses its queued count, and each canceled row's next search reads the time of the next Backlog search, which queues it again.
- `wanted-starved` after Heron's limit drops to 1 an hour, which RSS sync's reserve takes whole, the status line links `RSS sync uses all of Heron's search limit`, `status.limits` reads Heron `starved` with no `until`, and an open bulk run's group reads `behind` Heron `starved`.

## How to get to it (user POV)

- Desktop shell: `/library/wanted`, from `Wanted` in the `Library views` switch on the Library or the Calendar, and the command menu (`Open Wanted`), with `All`, `Movies` and `TV` under `Content Types`, the `Reasons` filter (`Failed`, `Missing`, `Upgrade available`, `File gone`) and `Sort`.
- Phone shell: the same page from the Library tab's `Library views` switch, one button per row that opens the row's actions.
- API: `GET /api/v1/wanted` (`status {searching, queued, nextBacklogAt, limits}`, `rows`), `POST /api/v1/works/search` (Search all, by ids or the Wanted filter), `DELETE /api/v1/search-queue` (Cancel queued searches), `POST /api/v1/works/{workId}/search`, `PATCH /api/v1/indexers/{id}` with `searches {count, per}`.

## Driving it with fakeworld and playwright cli

Preconditions:

- A `limited-tracker` run launched and doctor passing, with `RUN`, `FW` and `PW` set as in the skill's Launch, and `{runId}-admin-desktop` signed in. Its ten movies were added with no search, and Heron allows 2 searches an hour.

Steps:

- **Read Wanted.**

  ```sh
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop goto '{nodes.home.browserUrl}/library/wanted'
  $FW exec $RUN --proof wanted/wanted-rows -- $PW cli -s={runId}-admin-desktop snapshot --filename={runDir}/proof/wanted/wanted-rows/wanted.aria.yml
  $FW exec $RUN --proof wanted/wanted-rows -- $PW cli -s={runId}-admin-desktop screenshot --filename={runDir}/proof/wanted/wanted-rows/wanted.png
  ```

- **Search all.** The status names Heron's limit.

  ```sh
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop click 'getByRole("button", { name: "Search 10 Works" })'
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop run-code 'async page => { await page.getByRole("link", { name: /^Waiting for Heron's search limit until / }).waitFor({ timeout: 30000 }); }'
  $FW exec $RUN --proof wanted/wanted-search-all -- $PW cli -s={runId}-admin-desktop screenshot --filename={runDir}/proof/wanted/wanted-search-all/after.png
  $FW exec $RUN --proof wanted/wanted-search-all -- curl -sS -H 'X-Api-Key: {apiKey}' '{nodes.home.url}/api/v1/wanted'
  ```

- **Press a queued row, then another.** The first spends Heron's last search; the second skips Heron.

  ```sh
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop click 'getByRole("button", { name: "Search Copper Kettle" })'
  $FW exec $RUN --proof wanted/wanted-row-search -- $PW cli -s={runId}-admin-desktop screenshot --filename={runDir}/proof/wanted/wanted-row-search/after.png
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop click 'getByRole("button", { name: "Search Nightfall Ferry" })'
  $FW exec $RUN --proof wanted/wanted-press-skips -- $PW cli -s={runId}-admin-desktop screenshot --filename={runDir}/proof/wanted/wanted-press-skips/wanted.png
  $FW exec $RUN --proof wanted/wanted-press-skips -- curl -sS -H 'X-Api-Key: {apiKey}' '{nodes.home.url}/api/v1/wanted'
  ```

  Read Nightfall Ferry's `work.id` from that answer and `GET /api/v1/works/<id>/searches` with `--proof wanted/wanted-press-skips`: Heron's line reads `limit_reached`.

- **Cancel what is queued.**

  ```sh
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop click 'getByRole("button", { name: "Cancel queued searches" })'
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop run-code 'async page => { await page.getByText(/^Canceled \d+ queued searches\.$/).waitFor({ timeout: 10000 }); }'
  $FW exec $RUN --proof wanted/wanted-cancel -- $PW cli -s={runId}-admin-desktop screenshot --filename={runDir}/proof/wanted/wanted-cancel/canceled.png
  $FW exec $RUN --proof wanted/wanted-cancel -- curl -sS -H 'X-Api-Key: {apiKey}' '{nodes.home.url}/api/v1/wanted'
  ```

- **Starve Backlog search at Heron.** Read Heron's id from `GET /api/v1/indexers`, drop its limit to 1 an hour, and reload Wanted.

  ```sh
  $FW exec $RUN --proof wanted/wanted-starved -- curl -sS -X PATCH -H 'X-Api-Key: {apiKey}' -H 'Content-Type: application/json' -d '{"searches":{"count":1,"per":"hour"}}' '{nodes.home.url}/api/v1/indexers/<heron id>'
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop goto '{nodes.home.browserUrl}/library/wanted'
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop run-code 'async page => { await page.getByRole("link", { name: "RSS sync uses all of Heron's search limit" }).waitFor({ timeout: 30000 }); }'
  $FW exec $RUN --proof wanted/wanted-starved -- $PW cli -s={runId}-admin-desktop screenshot --filename={runDir}/proof/wanted/wanted-starved/wanted.png
  $FW exec $RUN --proof wanted/wanted-starved -- curl -sS -H 'X-Api-Key: {apiKey}' '{nodes.home.url}/api/v1/wanted'
  ```

- **Phone.** Open `{runId}-admin-phone`, go to `{nodes.home.browserUrl}/library/wanted` and snapshot and screenshot it into `proof/wanted/wanted-rows/wanted-phone.*`.

## Gotchas

- A search that finds nothing ends in milliseconds in the Fake world, so a press on a queued row reads `Already searching …`: the press promotes the queued group, which then starts at once.
- Search all at a limited Indexer leaves runs open in the `bulk` lane until the Indexer has room; they count under `searching`, not `queued`, and Cancel queued searches leaves them.
- RSS sync keeps one search of Heron's hour for its feed, so a limit of 2 still lets Backlog search and Search all spend one. Only a limit of 1 starves them.
- `until` times count from each search Heron took, so a line's `Limit reached until` and the status line's `until` can differ by a minute.
