# Series monitoring

An Admin chooses which episodes of a series are wanted: a Monitor switch per season and per episode on the Work page, and the Monitoring dialog with the presets All, Future, Missing, Existing, First season, Latest season and None, Include specials and Monitor new episodes. The dialog previews how many episodes start and stop being monitored before Apply. Monitoring never searches. Status and counts follow only monitored episodes.

## Sub-features

- `monitoring-dialog` on The Lantern Keepers added with First season, `Monitoring 3 of 11 episodes` opens `Set monitoring for The Lantern Keepers` with one radio per preset and its count (`All · 9 episodes` to `None · 0 episodes`), `Include specials` and `Monitor new episodes` (P-051).
- `monitoring-preview` `Latest season` reads `3 episodes start being monitored and 3 stop.`, and Include specials on `5 episodes start being monitored and 3 stop.` (P-051).
- `monitoring-apply` `Apply` closes the dialog: the hero reads `Monitoring 5 of 11 episodes`, Season 1 reads `Not monitored`, `Monitor Specials` is on, and the API reads `monitorNew` and `monitorAsides` true (P-051).
- `season-switch` `Monitor Season 2` then turns on all three of its episodes: the hero reads `Monitoring 8 of 11 episodes` and Season 2 no longer reads `Not monitored` (P-050).
- `episode-switch` `Monitor S01E02` on reads `Monitoring 9 of 11 episodes`, and `Monitor Season 1` reads mixed (`aria-checked="mixed"`), which a reload keeps (P-050).

## How to get to it (user POV)

- Desktop shell: a series' Work page: each season header's `Monitor <season>` switch, each row's `Monitor <code>` switch, and the hero's `Monitoring <n> of <m> episodes` button.
- Phone shell: each season card's switch; an episode's sheet holds its `Monitor` switch; the hero's `More actions for <series>` sheet holds `Set monitoring…`.
- Library: bulk Monitor and Unmonitor apply All and None to picked series, and `Set monitoring…` opens the same dialog for picked series of one type (see [bulk-edit.md](bulk-edit.md)).
- API: `PUT /api/v1/works/{id}/monitored {levelIds, itemIds, on}`, `PUT /api/v1/works/monitoring {workIds, preset, includeAsides, monitorNew, preview}`.

## Driving it with fakeworld, curl and playwright cli

Preconditions:

- A `series` run, doctor passing, `{runId}-admin-desktop` signed in.

Steps:

- **The dialog.** Add The Lantern Keepers with First season, open Monitoring, pick Latest season, turn on Include specials, then Apply.

  ```sh
  WORK=$($FW exec $RUN -- curl -sS -X POST -H 'X-Api-Key: {apiKey}' -H 'Content-Type: application/json' -d '{"contentType":"series","key":"1001","preset":"first"}' '{nodes.home.url}/api/v1/works' | jq -r .work.id)
  DIALOG='getByRole("dialog", { name: "Set monitoring for The Lantern Keepers" })'
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop goto "{nodes.home.browserUrl}/works/$WORK"
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop click 'getByRole("button", { name: /^Monitoring / })'
  $FW exec $RUN --proof series-monitoring/monitoring-dialog -- $PW cli -s={runId}-admin-desktop snapshot --filename={runDir}/proof/series-monitoring/monitoring-dialog/open.aria.yml
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop click "$DIALOG"'.getByRole("radio", { name: /^Latest season/ })'
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop click "$DIALOG"'.getByRole("switch", { name: "Include specials" })'
  $FW exec $RUN --proof series-monitoring/monitoring-preview -- $PW cli -s={runId}-admin-desktop snapshot --filename={runDir}/proof/series-monitoring/monitoring-preview/preview.aria.yml
  $FW exec $RUN --proof series-monitoring/monitoring-preview -- $PW cli -s={runId}-admin-desktop screenshot --filename={runDir}/proof/series-monitoring/monitoring-preview/preview.png
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop click "$DIALOG"'.getByRole("button", { name: "Apply" })'
  $FW exec $RUN --proof series-monitoring/monitoring-apply -- $PW cli -s={runId}-admin-desktop snapshot --filename={runDir}/proof/series-monitoring/monitoring-apply/after.aria.yml
  $FW exec $RUN --proof series-monitoring/monitoring-apply -- curl -sS -H 'X-Api-Key: {apiKey}' "{nodes.home.url}/api/v1/works/$WORK"
  ```

- **A season and an episode.** Turn Season 2 on, then S01E02 on, then reload.

  ```sh
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop click 'getByRole("switch", { name: "Monitor Season 2" })'
  $FW exec $RUN --proof series-monitoring/season-switch -- $PW cli -s={runId}-admin-desktop snapshot --filename={runDir}/proof/series-monitoring/season-switch/after.aria.yml
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop click 'getByRole("button", { name: "Season 1", exact: true })'
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop click 'getByRole("switch", { name: "Monitor S01E02" })'
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop reload
  $FW exec $RUN --proof series-monitoring/episode-switch -- $PW cli -s={runId}-admin-desktop snapshot --filename={runDir}/proof/series-monitoring/episode-switch/reloaded.aria.yml
  $FW exec $RUN --proof series-monitoring/episode-switch -- $PW cli -s={runId}-admin-desktop screenshot --filename={runDir}/proof/series-monitoring/episode-switch/reloaded.png
  ```

## Gotchas

- The dialog opens on All whatever the series was added with: a Work stores which episodes are monitored, not the preset that set them.
- The hero's count reads live episodes only. An episode TVDB dropped keeps its `monitored` flag in the API's tree, so count Items with `withdrawnAt` null when checking by API.
- A season that holds nothing wanted starts closed, so open Season 1 before its episode switches exist.
- On phone the hero has no Monitoring button; `More actions for <series>` holds `Set monitoring…`.
