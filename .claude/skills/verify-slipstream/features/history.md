# History

An Admin opens Activity > History and reads what happened to the library, newest first under each day's heading. Filters narrow it by Content Type, group, kind, date and Work and stay in the address; Load more (desktop) or scrolling (phone) pages back with no limit. The opening sentence leads with the failed Grabs that need attention, else counts the week's Grabs and Imports. Details opens an entry in full, linking to its Work and its Health item while each exists. History options set Keep history for and clear History. A Work page lists its ten newest entries.

## Sub-features

- `history-days` entries sit under `Today`, `Yesterday` or `Tuesday 10 November` (with the year when it isn't this one), each with its time of day such as `14:02` and its catalog words, such as `Added The Salt Road to Movies` (P-223).
- `history-paging` with more than 50 entries, `Load more` reads the page before the last one shown, by `before`, with no entry twice; a phone loads the next page as the list's end scrolls into view (P-223).
- `history-filters` the Content Type select, the group chips `Grabs`, `Imports`, `Library` and `Health` with each chosen group's kinds, `When` (`Any time`, `Today`, `Last 7 days`, `Last 30 days`) and a Work chip with `Show every Work` narrow the list and stay in the address; with nothing matching it reads `No entries match these filters.` with `Clear filters`, or `History is empty.` when History holds no entry at all (P-224, P-225).
- `history-sentence` the sentence reads `There were 2 Grabs and 1 Import in the last 7 days.`, or with a failed Grab that needs attention `1 Grab failed and needs attention.`, linking to Activity's Jobs.
- `history-entry` each row's words start at one edge, after its Work's poster or the empty poster column, and a failed or warning entry leads its time with the fail or warn icon, poster or not. `Details, <words>` opens an entry in full in place on desktop, its lines under the row's words, and a phone's whole row opens it in a sheet: its lines, Files, Stages and Health items, with `Open <Work>` while the Work exists and `Open on Health` while its item is on Health; a deleted Work's entry keeps its words and draws no link.
- `history-clear` `History options` > `Clear History…` opens `Clear History?`; `Clear History` toasts `Clearing History.`, the list and the bell empty at once, and a message still being sent stays in the bell until it's sent.
- `history-keep` `History options` > `Keep history for…` opens the `History` dialog, which edits Housekeeping's `Keep history for`, between 30 and 1,095 days.
- `history-api` `GET /history` refuses an unknown kind and a kind whose group isn't chosen at `query.kinds`, and `GET /history/{id}` an unknown zone at `query.zone`; a cleared entry answers 404 `history_entry_gone`.

## How to get to it (user POV)

- Desktop shell: Activity's `History` tab, `/activity/history`; a Work page's `History` section and its `See all in History`.
- Phone shell: Activity's `View` control, `History`; the same Work page section.
- API (Admin, Controller only): `GET /api/v1/history` with `before`, `limit`, `contentType`, `groups`, `kinds` (comma lists), `since` and `work`; `GET /api/v1/history/{id}?zone=`; `GET /api/v1/history/summary?since=`; `GET /api/v1/history/kinds`; `DELETE /api/v1/history` (202). A Worker answers 409 `worker_node`.
- Host CLI: none.

## Driving it with fakeworld and playwright cli

Preconditions:

- A fresh `imports` run, doctor passing, with `{runId}-admin-desktop` and `{runId}-admin-phone` signed in. Its The Quiet Orchard search grabs and imports a file, so History holds `Works added`, `Grabbed` and `Imported` entries.

Steps:

- **Read the days.** Open History and snapshot it: `Today` holds the entries newest first, each with its time of day.

  ```sh
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop goto "{nodes.home.browserUrl}/activity/history"
  $FW exec $RUN --proof history/history-days -- $PW cli -s={runId}-admin-desktop snapshot --filename={runDir}/proof/history/history-days/desktop.aria.yml
  $FW exec $RUN --proof history/history-days -- $PW cli -s={runId}-admin-desktop screenshot --full-page --filename={runDir}/proof/history/history-days/desktop.png
  ```

- **Filter.** Choose `Grabs`, then `Grabbed`, then `Last 7 days`: the address reads `groups=["grabs"]`, `kinds=["grabbed"]` and `since=7d`, and only the Grab is listed. `Clear filters` on an empty match lists everything again.

  ```sh
  $FW exec $RUN --proof history/history-filters -- $PW cli -s={runId}-admin-desktop run-code 'async page => { await page.getByRole("button", { name: "Grabs", exact: true }).click(); await page.getByRole("button", { name: "Grabbed" }).click(); await page.getByRole("radio", { name: "Last 7 days" }).click(); return page.url(); }'
  ```

- **Open an entry.** `Details` on the Import shows its lines and Files and `Open The Quiet Orchard`. On the phone the row itself opens a sheet named for the entry.

  ```sh
  $FW exec $RUN --proof history/history-entry -- $PW cli -s={runId}-admin-desktop run-code 'async page => { const row = page.getByRole("listitem").filter({ hasText: "Imported" }).first(); await row.getByRole("button", { name: /^Details, /u }).click(); return row.ariaSnapshot(); }'
  ```

- **Read the API.** The list, one entry's long form in a zone, and the refusals.

  ```sh
  $FW exec $RUN --proof history/history-api -- curl -sS -H 'X-Api-Key: {apiKey}' '{nodes.home.url}/api/v1/history?limit=2'
  $FW exec $RUN --proof history/history-api -- curl -sS -H 'X-Api-Key: {apiKey}' '{nodes.home.url}/api/v1/history?groups=imports&kinds=grabbed'
  $FW exec $RUN --proof history/history-api -- curl -sS -H 'X-Api-Key: {apiKey}' '{nodes.home.url}/api/v1/history/1?zone=Mars/Olympus'
  ```

- **Clear.** `History options` > `Clear History…` > `Clear History`: the toast reads `Clearing History.`, History reads `History is empty. Grabs, Imports and Works added or deleted show here as they happen.`, and the bell is empty. `GET /history/1?zone=UTC` then answers 404 `history_entry_gone`.

  ```sh
  $FW exec $RUN --proof history/history-clear -- $PW cli -s={runId}-admin-desktop run-code 'async page => { await page.getByRole("button", { name: "History options" }).click(); await page.getByRole("menuitem", { name: "Clear History…" }).click(); await page.getByRole("button", { name: "Clear History", exact: true }).click(); await page.getByText("Clearing History.").waitFor(); return page.getByRole("main").ariaSnapshot(); }'
  ```

## Gotchas

- The date chips and the day headings use the browser's zone, so a run near midnight can move an entry between `Today` and `Yesterday` between two reads.
- History draws no link for a Grab or an Import: no page shows one Grab or one finished Import yet, while `links.grab` and `links.import` stay on the wire.
- A Clear empties the list at once and sweeps the rows behind it; `Keep history for` prunes only when Housekeeping runs.
