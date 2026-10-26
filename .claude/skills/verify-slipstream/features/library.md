# Library

The Library shows every Work of a Content Type, or of every one on `/library`, as a poster grid or a table, from one projection the browser holds and keeps live. Sort cuts it into sections named by the value it sorts by, filters keep the Works that pass every group and any value picked inside one, and the search box matches any title a Work is known by. The view lives in the address and the Library opens on the last one used in that browser. Each poster carries its first fact and its status, a red Failed chip whenever a wanted cell failed, and goes quiet when nothing of the Work is monitored. Bulk edit over picked Works is [Bulk edit](./bulk-edit.md).

## Sub-features

- `library-grid` `/library/movie` draws a poster per Work under a heading per section with its count, each poster linking to its Work page with the year and a status such as `Missing` under it, two to a row on a phone 16px from each edge (P-033).
- `library-sort-sections` the `Sort: Title` menu offers Title, Date added, Availability date, Size on disk, Status, Profile, Library root, Monitored and each listed field such as Year; each cuts the Library into sections named by its value, such as `This month` or `2023` for a date and `No date` last, and `Reverse` flips the sort with Works lacking the value still last. The address carries `sort=` and `rev=1` (P-034).
- `library-filters` `Filter` lists Status, Monitored, Profile, Library root, Needs you and each filterable field with a count per value; picks inside a group keep Works with any of them and groups narrow together. `x of y Movies` and `Clear filters` show while narrowed, and a filter that keeps nothing reads `No Movies match these filters.` (P-035).
- `library-search` `Search Movies` keeps the Works `GET /api/v1/works/search?limit=0` matches by any title; a search that keeps nothing reads `No Movies match "<words>".`
- `library-table` `Library options` > `Table` lists one row per Work with the Status, Profile, Size on disk, Added and field columns, each a toggle in the same menu; below 620px of its width each row stacks its values under their labels (P-033).
- `library-remembered` a changed view goes into the address (`view=table&sort=added`), and opening `/library/movie` with no view in the address redirects to the last one used for that Library in that browser, keeping `q=`.
- `library-status` a poster or row shows its status in words and an icon, `Failed <n>` in red when a wanted cell failed, `Not monitored` and a quieter poster when nothing is monitored, and for a Work of many Items a tally such as `12 of 40 met` (P-030, P-040).
- `library-keyboard` one Work takes Tab; the arrow keys move between posters, up and down keeping the column across section headings, Home and End go to the first and last, Enter opens the Work and Space picks it.
- `library-live` a Work added, changed or deleted through the API appears, changes or leaves without a reload (P-242).

## How to get to it (user POV)

- Desktop shell: the sidebar's `Library` row for every type, and the Content Type row under it (`Movies`) for one.
- Phone shell: the `Library` tab, and `More` > `Movies`.
- API: `GET /api/v1/library` (the projection), `GET /api/v1/library?workIds=` (named rows), `GET /api/v1/works/search?q=&limit=0&contentType=`.
- Host CLI: none.

## Driving it with fakeworld and playwright cli

Preconditions:

- A fresh `movies` run, doctor passing, and `{runId}-admin-desktop` and `{runId}-admin-phone` signed in as in the skill's Drive.
- The Quiet Orchard and The Salt Road added, The Salt Road unmonitored:

  ```sh
  $FW exec $RUN -- curl -sS -X POST -H 'X-Api-Key: {apiKey}' -H 'Content-Type: application/json' -d '{"contentType":"movie","key":"9000001","monitor":true}' '{nodes.home.url}/api/v1/works'
  $FW exec $RUN -- curl -sS -X POST -H 'X-Api-Key: {apiKey}' -H 'Content-Type: application/json' -d '{"contentType":"movie","key":"9000005","monitor":false}' '{nodes.home.url}/api/v1/works'
  ```

- The browser tier proves the same paths in `web/e2e/library.spec.ts`, and the component tests in `web/src/routes/_app/library/*.test.tsx` and `web/src/library/*.test.ts`. `web/e2e/library-scale.spec.ts` measures 20,000 Works on `scale`.

Steps:

- **The grid.** Open the Movies Library on desktop and phone. Two `T 1`-style headings, two posters, The Salt Road quieter and reading `Not monitored`.

  ```sh
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop goto '{nodes.home.browserUrl}/library/movie'
  $FW exec $RUN --proof library/library-grid -- $PW cli -s={runId}-admin-desktop snapshot --filename={runDir}/proof/library/library-grid/desktop.aria.yml
  $FW exec $RUN --proof library/library-grid -- $PW cli -s={runId}-admin-desktop screenshot --filename={runDir}/proof/library/library-grid/desktop.png
  $FW exec $RUN -- $PW cli -s={runId}-admin-phone goto '{nodes.home.browserUrl}/library/movie'
  $FW exec $RUN --proof library/library-grid -- $PW cli -s={runId}-admin-phone screenshot --filename={runDir}/proof/library/library-grid/phone.png
  ```

- **Sort and Reverse.** Choose `Sort: Title`, then `Availability date`, then `Reverse`. The headings follow the release dates, and the address reads `?sort=available&rev=1`.

  ```sh
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop click 'getByRole("button", { name: "Sort: Title" })'
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop click 'getByRole("menuitemcheckbox", { name: "Availability date" })'
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop click 'getByRole("menuitemcheckbox", { name: "Reverse" })'
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop press Escape
  $FW exec $RUN --proof library/library-sort-sections -- $PW cli -s={runId}-admin-desktop snapshot --filename={runDir}/proof/library/library-sort-sections/available-reversed.aria.yml
  $FW exec $RUN --proof library/library-sort-sections -- $PW cli -s={runId}-admin-desktop eval 'location.search'
  ```

- **Filter.** Choose `Filter`, then `Not monitored (1)`. The Library reads `1 of 2 Movies` with The Salt Road alone. Choose `Clear filters`.

  ```sh
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop click 'getByRole("button", { name: "Filter" })'
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop click 'getByRole("checkbox", { name: "Not monitored (1)" })'
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop press Escape
  $FW exec $RUN --proof library/library-filters -- $PW cli -s={runId}-admin-desktop screenshot --filename={runDir}/proof/library/library-filters/not-monitored.png
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop click 'getByRole("main").getByRole("button", { name: "Clear filters" })'
  ```

- **Search.** Fill `Search Movies` with `route du sel`, The Salt Road's original title. The Salt Road stays alone; the API read answers its id.

  ```sh
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop fill 'getByRole("searchbox", { name: "Search Movies" })' "route du sel"
  $FW exec $RUN --proof library/library-search -- $PW cli -s={runId}-admin-desktop run-code 'async page => { await page.getByText("1 of 2 Movies").waitFor({ timeout: 5000 }); }'
  $FW exec $RUN --proof library/library-search -- $PW cli -s={runId}-admin-desktop screenshot --filename={runDir}/proof/library/library-search/route-du-sel.png
  $FW exec $RUN --proof library/library-search -- curl -sS -H 'X-Api-Key: {apiKey}' '{nodes.home.url}/api/v1/works/search?q=route+du+sel&limit=0'
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop fill 'getByRole("searchbox", { name: "Search Movies" })' ""
  ```

- **Table and the remembered view.** Choose `Library options`, then `Table`. Go to `/library/movie` with nothing after it: the address comes back as the table view, sorted as before.

  ```sh
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop click 'getByRole("button", { name: "Library options" })'
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop click 'getByRole("menuitemcheckbox", { name: "Table" })'
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop press Escape
  $FW exec $RUN --proof library/library-table -- $PW cli -s={runId}-admin-desktop screenshot --filename={runDir}/proof/library/library-table/desktop.png
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop goto '{nodes.home.browserUrl}/library/movie'
  $FW exec $RUN --proof library/library-remembered -- $PW cli -s={runId}-admin-desktop eval 'location.search'
  $FW exec $RUN -- $PW cli -s={runId}-admin-phone goto '{nodes.home.browserUrl}/library/movie?view=table'
  $FW exec $RUN --proof library/library-table -- $PW cli -s={runId}-admin-phone screenshot --filename={runDir}/proof/library/library-table/phone.png
  ```

- **Keyboard.** Back on the grid, focus the first poster and press `ArrowRight`; the second poster takes focus. `Space` picks it and the `Picked Works` bar opens.

  ```sh
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop goto '{nodes.home.browserUrl}/library/movie?view=grid'
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop run-code 'async page => { await page.getByRole("main").getByRole("link").first().focus(); }'
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop press ArrowRight
  $FW exec $RUN --proof library/library-keyboard -- $PW cli -s={runId}-admin-desktop run-code 'async page => page.evaluate(() => document.activeElement?.textContent)'
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop press Space
  $FW exec $RUN --proof library/library-keyboard -- $PW cli -s={runId}-admin-desktop snapshot --filename={runDir}/proof/library/library-keyboard/picked.aria.yml
  ```

- **Live.** With the page open, monitor The Salt Road through the API. Its poster loses `Not monitored` without a reload.

  ```sh
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop goto '{nodes.home.browserUrl}/library/movie?view=grid'
  $FW exec $RUN --proof library/library-live -- curl -sS -H 'X-Api-Key: {apiKey}' '{nodes.home.url}/api/v1/library'
  ```

  Set `S` to The Salt Road's `id` from that read, then:

  ```sh
  $FW exec $RUN --proof library/library-live -- curl -sS -X PUT -H 'X-Api-Key: {apiKey}' -H 'Content-Type: application/json' -d "{\"workIds\":[\"$S\"],\"on\":true}" '{nodes.home.url}/api/v1/works/monitored'
  $FW exec $RUN --proof library/library-live -- $PW cli -s={runId}-admin-desktop run-code 'async page => { await page.getByText("Not monitored").waitFor({ state: "detached", timeout: 10000 }); }'
  ```

## Gotchas

- Menu toggles stay open after a choice, as a column chooser should; press `Escape` before the next click outside the menu, or that click only closes it.
- The remembered view lives in the browser's storage per session; a fresh session opens on the default grid sorted by Title.
- `movies` adds no library root, so every Work reads `No library root` in the table until one is made; [Bulk edit](./bulk-edit.md) makes two.
- The grid draws only the rows near the window; a snapshot of a long Library holds a few dozen posters, not all of them.
