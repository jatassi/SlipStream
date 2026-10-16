# Tasks

The Controller runs each Scheduled task on its cadence and records every run with a plain result. The Tasks page opens with one sentence about what went wrong, then lists the shown tasks by group with cadence, last result, next run and Run now, and ends with the Schedule settings. Each task has a detail page with its units and its runs. A run an Admin starts, or a scheduled one open past the header delay, shows in a capsule in the desktop top bar and atop Activity on phone. `home` shows one task, Housekeeping, which prunes runs, Join Codes and sessions on the Controller and tidies each Node's database, so a stopped seedbox makes a run Partial and its return runs the seedbox's unit again.

## Sub-features

- `tasks-page` the Tasks page reads `Every task ran as scheduled.`, then the `Controller` group's table with Housekeeping, its cadence `Daily in the Maintenance window`, its last run, its next run `Tonight at 04:00` and its Nodes `home, seedbox` in a `Nodes` column (P-228).
- `tasks-run-now` `Run now` starts a run at once, the row shows `Running since <time>` with live progress and `Cancel` in its place, a second Run now over the API answers 409 `task_running`, and the row ends `Succeeded` with `6 steps` (P-228).
- `tasks-capsule` the desktop top bar shows `1 task running: Housekeeping` between the connection line and the page action, and its popover `Running tasks` lists the run with `home, seedbox · <n> of 6 steps`, a meter and `Cancel`; a scheduled run shows only after the header delay (P-007, P-008).
- `tasks-phone-running` on phone the same list, `Running tasks`, heads Activity while the run is open, and Activity's count and Home's colour never count it (P-007, P-008).
- `tasks-detail` `/tasks/housekeeping` reads the last result and the next run, the cadence and the overdue rule under `When it runs`, the `Units` table with each unit's last outcome, last success and cause, and the `Runs` list newest first, each run expanding to its units (P-228).
- `tasks-partial` with seedbox stopped, Run now ends `Partial` with `5 of 6 steps, seedbox offline`, and the unit `Local database on seedbox` reads `Skipped` with `seedbox is offline. Runs when seedbox returns.` (P-228).
- `tasks-returned` when seedbox starts again a `Node returned` run of `1 step` runs its unit, which ends `Succeeded` (P-228).
- `tasks-cadence` the row's `Runs every: Daily in the Maintenance window` opens the cadence; 2 days reads `Every 2 days in the Maintenance window` on the row and the detail page, a cadence outside 1 to 7 days answers 422 with `Between 1 day and 7 days.`, and `Reset to default` brings back daily (P-229).
- `tasks-schedule` the `Schedule` section edits the `Maintenance window` with the Controller's zone beside it, `Overdue tasks`, `Runs kept per task`, and under `Details` the jitter, the header delay and the Partial rule; a window of 01:00 to 03:00 moves Housekeeping's next run to 01:00 (P-229).
- `tasks-housekeeping` a Housekeeping run lists its units in order: `Run history`, `Join Codes`, `Sessions`, `Local database on home`, `Local database on seedbox`, `Library database`.

## How to get to it (user POV)

- Desktop shell: the sidebar's `Tasks`, below `Nodes`, which opens `/tasks`; a task's name opens `/tasks/<id>`. The capsule sits in the top bar while a run is shown.
- Phone shell: the `More` tab, then `Tasks`; the `Running tasks` list heads `Activity`.
- Both shells: the palette's `Go to Tasks`.
- API (Admin): `GET /api/v1/tasks`, `GET /api/v1/tasks/{taskId}`, `POST /api/v1/tasks/{taskId}/runs`, `DELETE /api/v1/tasks/runs/{runId}`, `PATCH /api/v1/tasks/{taskId}`, `GET` and `PATCH /api/v1/tasks/settings`, `GET /api/v1/tasks/runs/open`; live events `tasks.changed` and `tasks.progress`.
- Host CLI: none.

## Driving it with fakeworld and playwright cli

Preconditions:

- The baseline: a `home` run, doctor passing, `{runId}-admin-desktop` signed in. Housekeeping finishes in well under a second, so the steps that show an open run first slow the Node link with a latency fault and clear it after. Run the seedbox steps after the others or on their own run: they stop and start a Node.

Steps:

- **Open Tasks.** Choose `Tasks` in the sidebar. The page reads `Every task ran as scheduled.`, and the `Controller` table's Housekeeping row reads `Daily in the Maintenance window`, `Not run yet`, `Tonight at 04:00` and `home, seedbox`. The API returns the same row.

  ```sh
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop click 'getByRole("navigation", { name: "Main" }).getByRole("link", { name: "Tasks" })'
  $FW exec $RUN --proof tasks/tasks-page -- $PW cli -s={runId}-admin-desktop snapshot 'getByRole("main")' --filename={runDir}/proof/tasks/tasks-page/tasks.aria.yml
  $FW exec $RUN --proof tasks/tasks-page -- $PW cli -s={runId}-admin-desktop screenshot --filename={runDir}/proof/tasks/tasks-page/tasks.png
  $FW exec $RUN --proof tasks/tasks-page -- curl -sS -H 'X-Api-Key: {apiKey}' '{nodes.home.url}/api/v1/tasks'
  ```

- **Run now with the capsule.** Slow the link, then choose `Run now: Housekeeping`. The capsule reads `1 task running: Housekeeping` at once, and its popover lists `home, seedbox · <n> of 6 steps` with `Cancel`. A second Run now over the API answers 409 `task_running`.

  ```sh
  $FW fault set $RUN slow-link link/home latency=3s
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop click 'getByRole("button", { name: "Run now: Housekeeping" })'
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop click 'getByRole("banner").getByRole("button", { name: "1 task running: Housekeeping" })'
  $FW exec $RUN --proof tasks/tasks-capsule -- $PW cli -s={runId}-admin-desktop snapshot 'getByRole("dialog", { name: "Running tasks" })' --filename={runDir}/proof/tasks/tasks-capsule/popover.aria.yml
  $FW exec $RUN --proof tasks/tasks-capsule -- $PW cli -s={runId}-admin-desktop screenshot --filename={runDir}/proof/tasks/tasks-capsule/capsule.png
  $FW exec $RUN --proof tasks/tasks-run-now -- curl -sS -X POST -H 'X-Api-Key: {apiKey}' '{nodes.home.url}/api/v1/tasks/housekeeping/runs'
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop press Escape
  ```

- **Let it finish.** Clear the fault. The capsule goes, and the row reads `Succeeded` with `6 steps · <time>`. The stream carries `tasks.progress` and `tasks.changed`.

  ```sh
  $FW fault clear $RUN slow-link
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop run-code 'async page => { await page.getByRole("table", { name: "Controller" }).getByText("6 steps").waitFor({ timeout: 30000 }); }'
  $FW exec $RUN --proof tasks/tasks-run-now -- $PW cli -s={runId}-admin-desktop screenshot --filename={runDir}/proof/tasks/tasks-run-now/succeeded.png
  $FW sse list $RUN --as admin --event tasks.progress
  ```

- **Open the detail page.** Choose `Housekeeping`. The `Runs` list's first run reads `Succeeded`, `6 steps` and `Run now`, and expanding it lists the six units in order.

  ```sh
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop click 'getByRole("link", { name: "Housekeeping", exact: true })'
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop click 'getByRole("list", { name: "Runs" }).getByRole("listitem").first().getByRole("button")'
  $FW exec $RUN --proof tasks/tasks-detail -- $PW cli -s={runId}-admin-desktop snapshot 'getByRole("main")' --filename={runDir}/proof/tasks/tasks-detail/detail.aria.yml
  $FW exec $RUN --proof tasks/tasks-detail -- $PW cli -s={runId}-admin-desktop screenshot --filename={runDir}/proof/tasks/tasks-detail/detail.png
  ```

- **Change the cadence on the row.** On Tasks choose `Runs every: Daily in the Maintenance window`, enter 2 and Save. The row reads `Every 2 days in the Maintenance window`. Over the API a cadence of one hour answers 422 with `Between 1 day and 7 days.`.

  ```sh
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop goto {nodes.home.browserUrl}/tasks
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop click 'getByRole("button", { name: "Runs every: Daily in the Maintenance window" })'
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop fill 'getByRole("dialog", { name: "Runs every" }).getByRole("textbox", { name: "Runs every" })' 2
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop click 'getByRole("dialog", { name: "Runs every" }).getByRole("button", { name: "Save" })'
  $FW exec $RUN --proof tasks/tasks-cadence -- $PW cli -s={runId}-admin-desktop screenshot --filename={runDir}/proof/tasks/tasks-cadence/two-days.png
  $FW exec $RUN --proof tasks/tasks-cadence -- curl -sS -X PATCH -H 'X-Api-Key: {apiKey}' -H 'Content-Type: application/json' -d '{"cadence":3600}' '{nodes.home.url}/api/v1/tasks/housekeeping'
  ```

- **Reset it on the detail page.** Open Housekeeping: `When it runs` reads `Every 2 days in the Maintenance window`. Choose `Reset to default`: it reads `Daily in the Maintenance window` and the field `1 day`.

  ```sh
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop goto {nodes.home.browserUrl}/tasks/housekeeping
  $FW exec $RUN --proof tasks/tasks-cadence -- $PW cli -s={runId}-admin-desktop snapshot 'getByRole("region", { name: "When it runs" })' --filename={runDir}/proof/tasks/tasks-cadence/two-days.aria.yml
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop click 'getByRole("region", { name: "When it runs" }).getByRole("button", { name: "Reset to default" })'
  $FW exec $RUN --proof tasks/tasks-cadence -- $PW cli -s={runId}-admin-desktop snapshot 'getByRole("region", { name: "When it runs" })' --filename={runDir}/proof/tasks/tasks-cadence/reset.aria.yml
  ```

- **Move the Maintenance window.** On Tasks set `From` 01:00 and `to` 03:00 and Save. The zone reads beside the times (the window's `zone` from `GET /api/v1/tasks/settings`), and Housekeeping's next run reads `at 01:00`. `Reset to default` brings back 04:00 to 06:00.

  ```sh
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop goto {nodes.home.browserUrl}/tasks
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop fill 'getByRole("group", { name: "Maintenance window" }).getByRole("textbox", { name: "From" })' 01:00
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop fill 'getByRole("group", { name: "Maintenance window" }).getByRole("textbox", { name: "to" })' 03:00
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop click 'getByRole("group", { name: "Maintenance window" }).getByRole("button", { name: "Save" })'
  $FW exec $RUN --proof tasks/tasks-schedule -- $PW cli -s={runId}-admin-desktop snapshot 'getByRole("main")' --filename={runDir}/proof/tasks/tasks-schedule/window.aria.yml
  $FW exec $RUN --proof tasks/tasks-schedule -- $PW cli -s={runId}-admin-desktop screenshot --filename={runDir}/proof/tasks/tasks-schedule/window.png
  $FW exec $RUN --proof tasks/tasks-schedule -- curl -sS -H 'X-Api-Key: {apiKey}' '{nodes.home.url}/api/v1/tasks/settings'
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop click 'getByRole("region", { name: "Schedule" }).getByRole("button", { name: "Reset to default" })'
  ```

- **Run now on phone.** Sign in on phone and open Tasks. Slow the link, choose `Run now: Housekeeping` in the `Controller` list, then `Activity`: `Running tasks` heads the page with `home, seedbox · <n> of 6 steps`. Clear the fault and the list goes.

  ```sh
  $FW exec $RUN -- bun web/harness/bin/sign-in.ts {runDir} admin phone
  $FW exec $RUN -- $PW cli -s={runId}-admin-phone open --config={runDir}/browser/phone/playwright-cli.json
  $FW exec $RUN -- $PW cli -s={runId}-admin-phone state-load {runDir}/auth/admin-phone.json
  $FW exec $RUN -- $PW cli -s={runId}-admin-phone goto {nodes.home.browserUrl}/tasks
  $FW fault set $RUN slow-link link/home latency=3s
  $FW exec $RUN -- $PW cli -s={runId}-admin-phone click 'getByRole("list", { name: "Controller" }).getByRole("button", { name: "Run now: Housekeeping" })'
  $FW exec $RUN -- $PW cli -s={runId}-admin-phone click 'getByRole("navigation", { name: "Tabs" }).getByRole("link", { name: /^Activity/ })'
  $FW exec $RUN -- $PW cli -s={runId}-admin-phone run-code 'async page => { await page.getByRole("region", { name: "Running tasks" }).waitFor({ timeout: 10000 }); }'
  $FW exec $RUN --proof tasks/tasks-phone-running -- $PW cli -s={runId}-admin-phone snapshot 'getByRole("main")' --filename={runDir}/proof/tasks/tasks-phone-running/activity.aria.yml
  $FW exec $RUN --proof tasks/tasks-phone-running -- $PW cli -s={runId}-admin-phone screenshot --filename={runDir}/proof/tasks/tasks-phone-running/activity.png
  $FW fault clear $RUN slow-link
  ```

- **Stop seedbox and Run now.** Stop seedbox, open Housekeeping and choose `Run now: Housekeeping` in the top bar. The first run reads `Partial` with `5 of 6 steps, seedbox offline`, and the `Units` row `Local database on seedbox` reads `Skipped` with `seedbox is offline. Runs when seedbox returns.`.

  ```sh
  $FW node stop $RUN seedbox
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop goto {nodes.home.browserUrl}/tasks/housekeeping
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop click 'getByRole("banner").getByRole("button", { name: "Run now: Housekeeping" })'
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop run-code 'async page => { await page.getByRole("list", { name: "Runs" }).getByText("5 of 6 steps, seedbox offline").waitFor({ timeout: 30000 }); }'
  $FW exec $RUN --proof tasks/tasks-partial -- $PW cli -s={runId}-admin-desktop snapshot 'getByRole("main")' --filename={runDir}/proof/tasks/tasks-partial/partial.aria.yml
  $FW exec $RUN --proof tasks/tasks-partial -- $PW cli -s={runId}-admin-desktop screenshot --filename={runDir}/proof/tasks/tasks-partial/partial.png
  ```

- **Start seedbox.** A `Node returned` run of `1 step` heads the list and ends `Succeeded`, and the seedbox unit reads `Succeeded` with its time. Doctor passes again.

  ```sh
  $FW node start $RUN seedbox
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop run-code 'async page => { await page.getByRole("list", { name: "Runs" }).getByText("Node returned").waitFor({ timeout: 60000 }); }'
  $FW exec $RUN --proof tasks/tasks-returned -- $PW cli -s={runId}-admin-desktop snapshot 'getByRole("main")' --filename={runDir}/proof/tasks/tasks-returned/returned.aria.yml
  $FW exec $RUN --proof tasks/tasks-returned -- $PW cli -s={runId}-admin-desktop screenshot --filename={runDir}/proof/tasks/tasks-returned/returned.png
  $FW exec $RUN --proof tasks/tasks-returned -- curl -sS -H 'X-Api-Key: {apiKey}' '{nodes.home.url}/api/v1/tasks/housekeeping'
  $FW doctor $RUN
  ```

## Gotchas

- Without the latency fault a Housekeeping run opens and closes between two snapshots, so the capsule and the phone list are never seen. With `latency=3s` a run stays open about 5 s and ends `Succeeded`, with the link up throughout. The link's keepalive must still answer inside its 10 s write timeout, so keep the latency well under 5 s. Clear the fault before the next step, or every later run crawls.
- Next-run words follow the clock: between 04:00 and 06:00 the window is open, so the row reads `Tomorrow at 04:00`, and a moved window that covers now makes Housekeeping due at once.
- The zone beside the window is the Controller's: an IANA name when the process knows one (`TZ`), else its abbreviation, which reads `UTC` in a container. Time words use that zone only when it is an IANA name or `UTC`, else the browser's.
- Run now on the detail page sits in the top bar on desktop and in the page on phone; scope it to `getByRole("banner")` on desktop, since the Tasks row's button has the same name.
- `fakeworld node stop seedbox` makes doctor's cluster check fail until `node start`; a Run now in between is Partial by design.
