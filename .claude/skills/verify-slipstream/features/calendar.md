# Calendar

An Admin asks the Calendar what comes out in a span of days and whether the library has it. `GET /calendar` answers each day's rows: a Work's Items of one level and one dated kind on that day, worded by their place in the level, with the most urgent state among them and how many are met. A sentence leads with Items that are out and still missing, else with what comes out in the next 7 days. Each Content Type declares its dated kinds on `GET /content-types`. This file holds the API half; the Calendar page comes with its own recipe.

## Sub-features

- `calendar-rows` a span holding a dated album answers one row for its Tracks, `Harbor Lights · Tracks 1 to 3` under `release`, with `state`, `met` and `of`, and leaves out the albums dated only by a year or a month (P-219, P-221).
- `calendar-sentence` with the album out and its Tracks missing, the sentence reads `3 tracks are out and still missing.` with `needsYou: true` (P-219).
- `calendar-kinds` `GET /content-types` lists music's dated kind, `{"key":"release","label":"Release date"}` (P-221).
- `calendar-refusals` a span of more than 42 days, a `to` not after `from`, an unknown `contentType` and an unknown kind each answer 422 `invalid` at their `query.` field in words (P-219).

## How to get to it (user POV)

- Desktop shell: none yet.
- Phone shell: none yet.
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

## Gotchas

- `to` is the day after the last day: `from=2001-03-01&to=2001-04-01` is March, and 42 days is the most a span can hold.
- The `music` Scenario's dates are fixed in 2001 and 2003, so its rows are always in the past and the sentence always leads with what is missing. A row for this week needs a date set at run time.
- `kinds` is one comma list, such as `kinds=release,cinema`.
- Days are the Controller's. `today` is its day, and `nextDayAt` is when its next day starts, in UTC.
