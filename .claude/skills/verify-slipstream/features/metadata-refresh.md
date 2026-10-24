# Metadata refresh

The Metadata refresh task keeps every Work's record current. Each run first reads the metadata source's changes list since the last good run and makes every Work it names due, then refreshes each due Work on the Node that holds Metadata: a Work is due when its tier's days have passed since it was last answered (Unreleased, Growing, Every other, with a Work gone upstream riding Every other), when the changes list named it, or when an Admin asked to refresh the whole Content Type. The task is listed on Tasks under `Connect` once the library holds a Work, its detail page carries the tiers and the changes switch under `When it runs`, and a Work the source no longer has fails with that cause.

## Sub-features

- `refresh-hidden` with an empty library `GET /api/v1/tasks` lists Housekeeping alone, `POST /api/v1/content-types/movie/refresh` answers 404 `task_unknown` with `Metadata refresh is listed once the library holds a Work`, and an unknown Content Type answers 404 `content_type_not_found` (P-038).
- `refresh-listed` once a movie is added, Tasks lists `Metadata refresh` in the `Connect` table with `Daily in the Maintenance window`, `Not run yet` and `Run now: Metadata refresh` (P-231).
- `refresh-detail` `/tasks/metadata-refresh` reads `Runs every` 1 day (between 1 day and 7 days), `When overdue`, `Unreleased Works refresh every` 1 day, `Growing Works refresh every` 1 day, `Every other Work refresh every` 30 days and the switch `Refresh what the provider reports changed` on, each with the floor `No faster than 1 day. Every SlipStream install shares the bundled keys.`; a first Run now with nothing due ends `Succeeded` with `0 Works` (P-231).
- `refresh-changes` after the `tmdb-change` cue, Run now ends `Succeeded` with `1 Work` on `home`, its unit `The Quiet Orchard` `Succeeded`, and the Work's `overview` field reads `Revision 1.` (P-231).
- `refresh-all` `POST /api/v1/content-types/movie/refresh` answers 202 with `{"runId":n}`, and run n lists `The Quiet Orchard` refreshed on `home` (P-038).
- `refresh-all-running` with TMDB slowed, a second refresh all while the first run is open answers 409 `task_running` with `Metadata refresh has been running since <time>; the next run refreshes every Work`, and the capsule's `Running tasks` lists `Metadata refresh` with `Cancel` (P-038).
- `refresh-gone` after the `tmdb-delete` cue, Run now ends `Failed` with `1 of 1 Work failed`, the unit's cause reads `the metadata source no longer has this Work`, the Work's `goneUpstreamAt` is set, and Tasks opens with `Metadata refresh failed at <time>.` (P-231).
- `refresh-tuning` saving `Every other Work refresh every` 14 on the detail page stores `metadata.refresh.others` as 1209600 in the task's `tuning`; a cadence of one hour answers 422 with the floor and eight days with `Between 1 day and 7 days.` (P-231).

## How to get to it (user POV)

- Desktop shell: the sidebar's `Tasks`, then `Metadata refresh` in the `Connect` table, which opens `/tasks/metadata-refresh`; Run now sits on the row and in the detail page's top bar.
- Phone shell: the `More` tab, then `Tasks`, then `Metadata refresh` in the `Connect` list.
- Both shells: the palette's `Go to Tasks`.
- API (Admin): `POST /api/v1/content-types/{contentType}/refresh` (refresh all), `GET /api/v1/tasks/metadata-refresh`, `POST /api/v1/tasks/metadata-refresh/runs`, `PATCH /api/v1/tasks/metadata-refresh`.
- Host CLI: none.

## Driving it with fakeworld and playwright cli

Preconditions:

- The baseline on a fresh run: the library must start empty for `refresh-hidden`, and the `tmdb-delete` step leaves the Work gone for the rest of the run, so it comes last but for tuning.

Steps:

- **Before any Work.** Tasks lists Housekeeping alone, and refresh all is refused.

  ```sh
  $FW exec $RUN --proof metadata-refresh/refresh-hidden -- curl -sS -H 'X-Api-Key: {apiKey}' '{nodes.home.url}/api/v1/tasks'
  $FW exec $RUN --proof metadata-refresh/refresh-hidden -- curl -sS -i -X POST -H 'X-Api-Key: {apiKey}' '{nodes.home.url}/api/v1/content-types/movie/refresh'
  $FW exec $RUN --proof metadata-refresh/refresh-hidden -- curl -sS -i -X POST -H 'X-Api-Key: {apiKey}' '{nodes.home.url}/api/v1/content-types/show/refresh'
  ```

- **Add the movie and open Tasks.** Add The Quiet Orchard as in [add-a-movie.md](./add-a-movie.md), then open Tasks. The `Connect` table lists Metadata refresh.

  ```sh
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop goto '{nodes.home.browserUrl}/add?contentType=movie&query=quiet+orchard'
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop click 'getByRole("button", { name: "Add The Quiet Orchard" })'
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop click 'getByRole("dialog", { name: "Add The Quiet Orchard" }).getByRole("button", { name: "Add Movie" })'
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop goto '{nodes.home.browserUrl}/tasks'
  $FW exec $RUN --proof metadata-refresh/refresh-listed -- $PW cli -s={runId}-admin-desktop snapshot 'getByRole("main")' --filename={runDir}/proof/metadata-refresh/refresh-listed/tasks.aria.yml
  $FW exec $RUN --proof metadata-refresh/refresh-listed -- $PW cli -s={runId}-admin-desktop screenshot --filename={runDir}/proof/metadata-refresh/refresh-listed/tasks.png
  ```

- **Run now with nothing due, then open the detail page.** The first run ends `0 Works`; it starts the changes list's cursor. `When it runs` holds the tiers and the switch.

  ```sh
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop click 'getByRole("button", { name: "Run now: Metadata refresh" })'
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop click 'getByRole("link", { name: "Metadata refresh", exact: true })'
  $FW exec $RUN --proof metadata-refresh/refresh-detail -- $PW cli -s={runId}-admin-desktop snapshot 'getByRole("main")' --filename={runDir}/proof/metadata-refresh/refresh-detail/detail.aria.yml
  $FW exec $RUN --proof metadata-refresh/refresh-detail -- $PW cli -s={runId}-admin-desktop screenshot --full-page --filename={runDir}/proof/metadata-refresh/refresh-detail/detail.png
  ```

- **A change upstream.** Read the Work, play `tmdb-change`, and choose `Run now: Metadata refresh` in the top bar. The new run reads `1 Work`, and the Work's `overview` field reads `Revision 1.`. Take the Work's id from `GET /api/v1/works`.

  ```sh
  $FW exec $RUN --proof metadata-refresh/refresh-changes -- curl -sS -H 'X-Api-Key: {apiKey}' '{nodes.home.url}/api/v1/works/<id>'
  $FW cue $RUN tmdb-change '{"id":9000001}'
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop click 'getByRole("banner").getByRole("button", { name: "Run now: Metadata refresh" })'
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop run-code 'async page => { await page.getByRole("list", { name: "Runs" }).getByText("1 Work", { exact: true }).waitFor({ timeout: 30000 }); }'
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop click 'getByRole("list", { name: "Runs" }).getByRole("listitem").first().getByRole("button")'
  $FW exec $RUN --proof metadata-refresh/refresh-changes -- $PW cli -s={runId}-admin-desktop snapshot 'getByRole("main")' --filename={runDir}/proof/metadata-refresh/refresh-changes/detail.aria.yml
  $FW exec $RUN --proof metadata-refresh/refresh-changes -- curl -sS -H 'X-Api-Key: {apiKey}' '{nodes.home.url}/api/v1/works/<id>'
  ```

- **Refresh all.** The route answers 202 with the run's id, and that run lists the Work.

  ```sh
  $FW exec $RUN --proof metadata-refresh/refresh-all -- curl -sS -i -X POST -H 'X-Api-Key: {apiKey}' '{nodes.home.url}/api/v1/content-types/movie/refresh'
  $FW exec $RUN --proof metadata-refresh/refresh-all -- curl -sS -H 'X-Api-Key: {apiKey}' '{nodes.home.url}/api/v1/tasks/metadata-refresh'
  ```

- **Refresh all during a run.** Slow TMDB, refresh all twice: the first answers 202, the second 409 `task_running`. The capsule lists the run. Clear the fault once the run ends.

  ```sh
  $FW fault set $RUN slow-tmdb tmdb latency=3s
  $FW exec $RUN --proof metadata-refresh/refresh-all-running -- curl -sS -X POST -H 'X-Api-Key: {apiKey}' '{nodes.home.url}/api/v1/content-types/movie/refresh'
  $FW exec $RUN --proof metadata-refresh/refresh-all-running -- curl -sS -i -X POST -H 'X-Api-Key: {apiKey}' '{nodes.home.url}/api/v1/content-types/movie/refresh'
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop click 'getByRole("banner").getByRole("button", { name: /task running/ })'
  $FW exec $RUN --proof metadata-refresh/refresh-all-running -- $PW cli -s={runId}-admin-desktop snapshot 'getByRole("dialog", { name: "Running tasks" })' --filename={runDir}/proof/metadata-refresh/refresh-all-running/popover.aria.yml
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop press Escape
  $FW fault clear $RUN slow-tmdb
  ```

- **Gone upstream.** Play `tmdb-delete` and Run now. The run reads `Failed` and `1 of 1 Work failed`, the unit's cause `the metadata source no longer has this Work`, and Tasks opens with `Metadata refresh failed at <time>.`.

  ```sh
  $FW cue $RUN tmdb-delete '{"id":9000001}'
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop goto '{nodes.home.browserUrl}/tasks/metadata-refresh'
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop click 'getByRole("banner").getByRole("button", { name: "Run now: Metadata refresh" })'
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop click 'getByRole("list", { name: "Runs" }).getByRole("listitem").first().getByRole("button").first()'
  $FW exec $RUN --proof metadata-refresh/refresh-gone -- $PW cli -s={runId}-admin-desktop snapshot 'getByRole("main")' --filename={runDir}/proof/metadata-refresh/refresh-gone/detail.aria.yml
  $FW exec $RUN --proof metadata-refresh/refresh-gone -- curl -sS -H 'X-Api-Key: {apiKey}' '{nodes.home.url}/api/v1/works/<id>'
  ```

- **Tune a tier and the cadence.** Fill `Every other Work refresh every` with 14 and Save; the task's `tuning` reads 1209600. Over the API one hour and eight days are refused.

  ```sh
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop fill 'getByRole("region", { name: "When it runs" }).getByRole("textbox", { name: "Every other Work refresh every" })' 14
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop click 'getByRole("region", { name: "When it runs" }).getByRole("button", { name: "Save" })'
  $FW exec $RUN --proof metadata-refresh/refresh-tuning -- curl -sS -H 'X-Api-Key: {apiKey}' '{nodes.home.url}/api/v1/tasks/metadata-refresh'
  $FW exec $RUN --proof metadata-refresh/refresh-tuning -- curl -sS -X PATCH -H 'X-Api-Key: {apiKey}' -H 'Content-Type: application/json' -d '{"cadence":3600}' '{nodes.home.url}/api/v1/tasks/metadata-refresh'
  $FW exec $RUN --proof metadata-refresh/refresh-tuning -- curl -sS -X PATCH -H 'X-Api-Key: {apiKey}' -H 'Content-Type: application/json' -d '{"cadence":691200}' '{nodes.home.url}/api/v1/tasks/metadata-refresh'
  ```

## Gotchas

- The first run after a Work is added reads no changes list: it only starts the cursor at its own start. A `tmdb-change` played before that run is never seen, so play it after the first run.
- `Units` and the row's Nodes cell list the Works due now, so both read empty between runs; the Works a run refreshed are under its entry in `Runs`.
- A refresh all sent while a run is open is kept, but a Work that run answers after the request counts as refreshed, so with one Work the next run has nothing due. The in-process test `TestRefreshAllDuringARunKeepsTheRequestAndAnswersTaskRunning` proves a Work answered before the request runs again.
- Run lists on `home` place every unit on `home`, the Metadata holder; moving Metadata to seedbox moves them.
- Between 04:00 and 06:00 the Maintenance window is open, so a scheduled run can land between steps; the row's next run then reads `Tomorrow at 04:<mm>`.
- A Work gone upstream is answered by its gone time, so it rides the `Every other Work` tier and leaves `Units` until that tier comes round.
