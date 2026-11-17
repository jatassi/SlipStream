# Calendar

An Admin asks the Calendar what comes out in a span of days and whether the library has it. `GET /calendar` answers each day's rows: a Work's Items of one level and one dated kind on that day, worded by their place in the level, with the most urgent state among them and how many are met. A sentence leads with Items that are out and still missing, else with what comes out in the next 7 days. Each Content Type declares its dated kinds on `GET /content-types`. The Calendar page at `/library/calendar` and `/library/calendar/<id>` draws the same answer as a Month of six weeks or a List of six weeks, opens from an always-shown row above the Library and the palette's `Open Calendar`, keeps its view, month, kinds and Show unmonitored in the address, and opens on the List on a phone.

## Sub-features

- `calendar-rows` a span holding a dated album answers one row for its Tracks, `Harbor Lights · Tracks 1 to 3` under `release`, with `state`, `met` and `of`, and leaves out the albums dated only by a year or a month (P-219, P-221).
- `calendar-sentence` with the album out and its Tracks missing, the sentence reads `3 tracks are out and still missing.` with `needsYou: true` (P-219).
- `calendar-kinds` `GET /content-types` lists music's dated kind, `{"key":"release","label":"Release date"}` (P-221).
- `calendar-page-month` `/library/calendar/music?view=month&month=2001-03` reads `March 2001` between `Previous month` and `Next month`, and the cell for Monday 12 March lists The Lanterns with `Harbor Lights · Tracks 1 to 3 · Release date` and the Missing mark. The sentence `3 tracks are out and still missing.` links to `/library/wanted/music` (P-219).
- `calendar-page-list` `List` reads the six weeks from the address's `from`, one section per day with rows, each row The Lanterns' poster and label linking to its Work page, `Harbor Lights · Tracks 1 to 3 · Release date` and `Missing · 0 of 3 met`; `Tracks` opens to `Track 1 · Tidewater` and on. `Show later` and `Show earlier` move `from` by 42 days, and a span with no rows reads `Nothing in your library comes out between <first day> and <last day>.` (P-220, P-221).
- `calendar-page-unmonitored` with The Lanterns' Tracks 2 and 3 not monitored, the month shows only `Harbor Lights · Track 1`. `Show unmonitored` puts `unmonitored=true` in the address and adds `Harbor Lights · Tracks 2 and 3` with the `Not monitored` mark (P-219).
- `calendar-page-entry` the Library reads a `Calendar` row over `Release dates in your library, day by day.`, which opens `/library/calendar` on `All` and `/library/calendar/music` on Music, and the palette's `Open Calendar` opens `/library/calendar`.
- `calendar-page-phone` on a phone `/library/calendar` opens as `view=list`; its Month shows one mark a day, and tapping a day lists that day's rows under the grid.
- `calendar-refusals` a span of more than 42 days, a `to` not after `from`, an unknown `contentType` and an unknown kind each answer 422 `invalid` at their `query.` field in words (P-219).

## How to get to it (user POV)

- Desktop shell: the `Calendar` row above the Library's grid, or ⌘K `Open Calendar`.
- Phone shell: the `Calendar` row above the Library's grid, or Search > `Open Calendar`; it opens on the List.
- API (Admin, Controller only): `GET /api/v1/calendar?from=&to=` with optional `contentType`, `kinds` (a comma list) and `unmonitored=true`; `GET /api/v1/content-types` for each type's `dated` kinds. A Worker answers 409 `worker_node`.
- Host CLI: none.

## Driving it with fakeworld and curl

Preconditions:

- A fresh `music` run on `dist/slipstream-e2e`, which registers music beside movies, doctor passing (its build check names the e2e main, which is expected).
- The Lanterns added, whose Harbor Lights first came out on 2001-03-12 and Night Ferry only in 2003 (a year, so it never reaches a day):

  ```sh
  $FW exec $RUN -- curl -sS -X POST -H 'X-Api-Key: {apiKey}' -H 'Content-Type: application/json' --data-binary '{"contentType":"music","key":"a1000000-0000-4000-8000-000000000001","preset":"all"}' '{nodes.home.url}/api/v1/works'
  ```

Steps:

- **Read music's dated kinds.** `/content-types` needs a signed-in Admin (the API key answers 403 `needs_account`), so sign Ada in with a cookie jar first. The music type lists `dated: [{"key":"release","label":"Release date"}]`.

  ```sh
  $FW exec $RUN -- curl -sS -c '{runDir}/admin-cookies.txt' -X POST -H 'Content-Type: application/json' --data-binary '{"email":"ada@example.test","pin":"2468"}' '{nodes.home.url}/api/v1/session'
  $FW exec $RUN --proof calendar/calendar-kinds -- curl -sS -b '{runDir}/admin-cookies.txt' '{nodes.home.url}/api/v1/content-types'
  ```

- **Read a span holding Harbor Lights.** `days` holds `2001-03-12` alone, with one row for The Lanterns: `kind: release`, `kindLabel: Release date`, `itemsWords: Harbor Lights · Tracks 1 to 3`, three `items` labeled `Track 1 · Tidewater` and on, `state: missing`, `met: 0`, `of: 3`. The sentence reads `3 tracks are out and still missing.` with `needsYou: true`. Spans over February 2001 and January 2003 answer no days.

  ```sh
  $FW exec $RUN --proof calendar/calendar-rows -- curl -sS -H 'X-Api-Key: {apiKey}' '{nodes.home.url}/api/v1/calendar?from=2001-03-01&to=2001-04-01'
  $FW exec $RUN -- curl -sS -H 'X-Api-Key: {apiKey}' '{nodes.home.url}/api/v1/calendar?from=2001-02-01&to=2001-03-01'
  $FW exec $RUN -- curl -sS -H 'X-Api-Key: {apiKey}' '{nodes.home.url}/api/v1/calendar?from=2003-01-01&to=2003-02-01'
  ```

- **Ask for too much.** A 43-day span answers 422 at `query.to` with `A calendar request can span at most 42 days. Ask for a shorter span.`; `contentType=boats` answers `There's no kind of media called boats.` at `query.contentType`; `kinds=boxed` answers `There's no release date kind called boxed. Choose one from the Calendar's filter.` at `query.kinds`.

  ```sh
  $FW exec $RUN --proof calendar/calendar-refusals -- curl -sS -H 'X-Api-Key: {apiKey}' '{nodes.home.url}/api/v1/calendar?from=2001-03-01&to=2001-04-13'
  $FW exec $RUN -- curl -sS -H 'X-Api-Key: {apiKey}' '{nodes.home.url}/api/v1/calendar?from=2001-03-01&to=2001-04-01&contentType=boats'
  $FW exec $RUN -- curl -sS -H 'X-Api-Key: {apiKey}' '{nodes.home.url}/api/v1/calendar?from=2001-03-01&to=2001-04-01&kinds=boxed'
  $FW exec $RUN -- curl -sS -H 'X-Api-Key: {apiKey}' '{nodes.home.url}/api/v1/calendar?from=2001-03-01&to=2001-03-01'
  ```

  The last answers `The span has no days. Ask for an end day after the start day.` at `query.to`.

- **Open the page.** Sign `{runId}-admin-desktop` and `{runId}-admin-phone` in as in the skill's Drive. On desktop, open the Library and choose the `Calendar` row, then go to March 2001 on Music. The cell for Monday 12 March lists The Lanterns, and the sentence leads with `3 tracks are out and still missing.`

  ```sh
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop goto '{nodes.home.browserUrl}/library'
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop click 'getByRole("list", { name: "Calendar" }).getByRole("link")'
  $FW exec $RUN --proof calendar/calendar-page-entry -- $PW cli -s={runId}-admin-desktop snapshot --filename={runDir}/proof/calendar/calendar-page-entry/calendar.aria.yml
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop goto '{nodes.home.browserUrl}/library/calendar/music?view=month&month=2001-03'
  $FW exec $RUN --proof calendar/calendar-page-month -- $PW cli -s={runId}-admin-desktop snapshot --filename={runDir}/proof/calendar/calendar-page-month/month.aria.yml
  $FW exec $RUN --proof calendar/calendar-page-month -- $PW cli -s={runId}-admin-desktop screenshot --filename={runDir}/proof/calendar/calendar-page-month/month.png
  ```

- **The list.** Choose `List` with `from=2001-03-01` in the address, open `Tracks`, then `Show later`. The address reads `from=2001-04-12`, and the empty span names its days.

  ```sh
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop goto '{nodes.home.browserUrl}/library/calendar/music?view=list&from=2001-03-01'
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop click 'getByRole("button", { name: "Tracks" })'
  $FW exec $RUN --proof calendar/calendar-page-list -- $PW cli -s={runId}-admin-desktop snapshot --filename={runDir}/proof/calendar/calendar-page-list/list.aria.yml
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop click 'getByRole("button", { name: "Show later" })'
  $FW exec $RUN --proof calendar/calendar-page-list -- $PW cli -s={runId}-admin-desktop screenshot --filename={runDir}/proof/calendar/calendar-page-list/later.png
  ```

- **Open Calendar and the phone.** Press `ControlOrMeta+k`, type `Open Calendar` and press Enter: the address is `/library/calendar?view=month`. On the phone session `/library/calendar` comes back as `view=list`; choose `Month` and tap a day.

  ```sh
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop press 'ControlOrMeta+k'
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop type 'Open Calendar'
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop press 'Enter'
  $FW exec $RUN -- $PW cli -s={runId}-admin-phone goto '{nodes.home.browserUrl}/library/calendar/music?from=2001-03-01'
  $FW exec $RUN --proof calendar/calendar-page-phone -- $PW cli -s={runId}-admin-phone screenshot --filename={runDir}/proof/calendar/calendar-page-phone/list.png
  ```

- **Show unmonitored.** On a fresh `music` run, add The Lanterns with `"preset":"all"` as above, then turn off Tracks 2 and 3 (no add preset monitors a single track; `first` monitors the first album). Read their ids and the Work's id from the March span's row, open March 2001 and turn on `Show unmonitored`. The second row carries `Not monitored`.

  ```sh
  $FW exec $RUN -- curl -sS -H 'X-Api-Key: {apiKey}' '{nodes.home.url}/api/v1/calendar?from=2001-03-01&to=2001-04-01'
  $FW exec $RUN -- curl -sS -X PUT -H 'X-Api-Key: {apiKey}' -H 'Content-Type: application/json' --data-binary '{"levelIds":[],"itemIds":["<track 2 id>","<track 3 id>"],"on":false}' '{nodes.home.url}/api/v1/works/<work id>/monitored'
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop goto '{nodes.home.browserUrl}/library/calendar/music?view=month&month=2001-03'
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop click 'getByRole("switch", { name: "Show unmonitored" })'
  $FW exec $RUN --proof calendar/calendar-page-unmonitored -- $PW cli -s={runId}-admin-desktop snapshot --filename={runDir}/proof/calendar/calendar-page-unmonitored/month.aria.yml
  ```

  The Go e2e test `e2e/calendar_test.go` dates Harbor Lights on the first day of the run's month, sees it missing, moves its tracks into the music root and sees it met after Scan now. The component tests in `web/src/calendar/*.test.tsx` and `web/src/live/calendar-effects.test.tsx` prove the grid, paging, the address and the live rules.

## Gotchas

- `to` is the day after the last day: `from=2001-03-01&to=2001-04-01` is March, and 42 days is the most a span can hold.
- The `music` Scenario's dates are fixed in 2001 and 2003, so its rows are always in the past and the sentence always leads with what is missing. A row for this week needs a date set at run time.
- `kinds` is one comma list, such as `kinds=release,cinema`.
- Days are the Controller's. `today` is its day, and `nextDayAt` is when its next day starts, in UTC.
- The page guesses the Controller's day from the browser's until its first answer, then moves a month or list the address leaves out to the Controller's day.
- A narrow container, below 640px, draws the month as one mark a day whatever the shell, and lists the picked day's rows under it.
