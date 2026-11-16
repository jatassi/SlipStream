# Backlog search

Backlog search runs every 8 hours by default and queues each search group whose due time has come, missing before upgrades, at Backlog pace behind an Indexer's limit. The task's page holds its interval and `Run now`, each run's outcome reads `Queued N searches.`, and turned off in Settings > Automatic search its run is skipped with `Backlog search is off in Settings.` Wanted's status line names the next run, or that Backlog search is off.

## Sub-features

- `backlog-run-now` on `limited-tracker`, `Run now: Backlog search` on the task's page ends a run `succeeded` whose one unit `Due searches` reads `Queued 0 searches.` while every group's due time is still to come (P-136, P-137).
- `backlog-off` with `search.backlogOn` off through `PATCH /api/v1/search-settings`, `Run now` ends the run `skipped` with the cause `Backlog search is off in Settings.`, and Wanted's status line reads `Backlog search is off` in place of `Next Backlog search …` (P-136).

## How to get to it (user POV)

- Desktop shell: the sidebar's `Tasks`, then `Backlog search`, with `Run now: Backlog search` in the header and `Backlog search runs every` under `When it runs`; Settings > Automatic search's `Backlog search` and `Backlog search looks for upgrades` switches and the same interval; Library > Wanted's status line.
- Phone shell: `More`, then `Tasks`, and the same page; Settings > Automatic search.
- API: `GET /api/v1/tasks/backlog-search`, `POST /api/v1/tasks/backlog-search/runs` (202 `{runId}`), `GET` and `PATCH /api/v1/search-settings` (`search.backlogOn`, `search.backlogUpgrades`, `tasks.backlog-search.cadence`), `GET /api/v1/wanted` (`status.nextBacklogAt`).

## Driving it with fakeworld and playwright cli

Preconditions:

- A `limited-tracker` run launched and doctor passing, with `RUN`, `FW` and `PW` set as in the skill's Launch, and `{runId}-admin-desktop` signed in. Its ten movies are added with no search, so each is wanted and not searched yet.

Steps:

- **Run it now.** Open the task and choose `Run now: Backlog search`. The run reads `Queued 0 searches.`: a first Backlog search spreads new groups' due times over its interval, so none is due yet.

  ```sh
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop goto '{nodes.home.browserUrl}/tasks/backlog-search'
  $FW exec $RUN --proof backlog-search/backlog-run-now -- $PW cli -s={runId}-admin-desktop screenshot --filename={runDir}/proof/backlog-search/backlog-run-now/before.png
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop click 'getByRole("button", { name: "Run now: Backlog search" })'
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop run-code 'async page => { await page.getByText("Queued 0 searches.").first().waitFor({ timeout: 30000 }); }'
  $FW exec $RUN --proof backlog-search/backlog-run-now -- $PW cli -s={runId}-admin-desktop screenshot --filename={runDir}/proof/backlog-search/backlog-run-now/ran.png
  $FW exec $RUN --proof backlog-search/backlog-run-now -- curl -sS -H 'X-Api-Key: {apiKey}' '{nodes.home.url}/api/v1/tasks/backlog-search'
  ```

- **Turn it off and run it.** The run ends `skipped` with its cause, and Wanted reads `Backlog search is off`.

  ```sh
  $FW exec $RUN --proof backlog-search/backlog-off -- curl -sS -X PATCH -H 'X-Api-Key: {apiKey}' -H 'Content-Type: application/json' -d '{"values":{"search.backlogOn":false}}' '{nodes.home.url}/api/v1/search-settings'
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop click 'getByRole("button", { name: "Run now: Backlog search" })'
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop run-code 'async page => { await page.getByText("Backlog search is off in Settings.").first().waitFor({ timeout: 30000 }); }'
  $FW exec $RUN --proof backlog-search/backlog-off -- $PW cli -s={runId}-admin-desktop screenshot --filename={runDir}/proof/backlog-search/backlog-off/skipped.png
  $FW exec $RUN --proof backlog-search/backlog-off -- curl -sS -H 'X-Api-Key: {apiKey}' '{nodes.home.url}/api/v1/tasks/backlog-search'
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop goto '{nodes.home.browserUrl}/library/wanted'
  $FW exec $RUN --proof backlog-search/backlog-off -- $PW cli -s={runId}-admin-desktop screenshot --filename={runDir}/proof/backlog-search/backlog-off/wanted.png
  ```

## Gotchas

- Backlog search's first scheduled run comes 8 hours after the Controller starts, and its first reconcile spreads new groups' due times over the interval, so a group's due time can fall after the next run. Use `Run now` and read the note; a run that queues groups needs a group whose due time has passed, which a verify run of minutes rarely holds.
- A run's note counts groups, not Works: a series' season is one group.
- `Cancel queued searches` on Wanted keeps each group's due time, so the next Backlog search queues them again.
