# Library

The Library shows every Work of a Content Type, or of every one on `/library`, as a poster grid or a table, from one projection the browser holds and keeps live. Sort cuts it into sections named by the value it sorts by, filters keep the Works that pass every group and any value picked inside one, and the search box matches any title a Work is known by. The view lives in the address and the Library opens on the last one used in that browser. Each poster carries its first fact and its status, which puts what needs attention first, so any failed cell makes it a red `Failed <n>`, and goes quiet when nothing of the Work is monitored. Bulk edit over picked Works is [Bulk edit](./bulk-edit.md).

## Sub-features

- `library-grid` `/library/movie` draws a poster per Work under a heading per section with its count, each poster linking to its Work page with the year and a status such as `Missing` under it, two to a row on a phone 16px from each edge (P-033).
- `library-sort-sections` the `Sort: Title` menu offers Title, Date added, Availability date, Size on disk, Status, Profile, Library root, Monitored and each listed field such as Year; each cuts the Library into sections named by its value, such as `This month` or `2023` for a date and `No date` last, and Title by its first letter after a leading English article, so The Quiet Orchard sits under `Q`. `Reverse` flips the sort with Works lacking the value still last. The address carries `sort=` and `rev=1` (P-034).
- `library-filters` `Filter` lists Status, Monitored, Profile, Library root, Needs you and each filterable field with a count per value; picks inside a group keep Works with any of them and groups narrow together. `x of y Movies` and `Clear filters` show while narrowed, and a filter that keeps nothing reads `No Movies match these filters.` (P-035).
- `library-search` `Search Movies` keeps the Works `GET /api/v1/works/search?limit=0` matches by any title; a search that keeps nothing reads `No Movies match "<words>".`
- `library-table` `Library options` > `Table` lists one row per Work with the Status, Profile, Size on disk, Added and field columns, each a toggle in the same menu; below 620px of its width each row stacks its values under their labels (P-033).
- `library-remembered` a changed view goes into the address (`view=table&sort=added`), and opening `/library/movie` with no view in the address redirects to the last one used for that Library in that browser, keeping `q=`.
- `library-status` a poster or row shows its status in words and an icon, in the order failed, missing, downloading, upgrade available, unreleased, met, so `Failed <n>` in red with its count of failed cells whenever any cell failed, `Unreleased` only when nothing else applies, `Not monitored` alone and a quieter poster when nothing is monitored, since the status counts only monitored Items its source still lists, and for a Work of many Items a tally such as `12 of 40 met` (P-030, P-040).
- `library-keyboard` one Work takes Tab; the arrow keys move between posters, up and down keeping the column across section headings, Home and End go to the first and last, Enter opens the Work and Space picks it.
- `library-type-switch` with two or more types on, a `Content Types` switch beside the title links `All` and each enabled type by its name in sidebar order to `/library` and `/library/<id>`, marks the one shown, and each segment opens on the view its own Library last used; with one type on there is no switch. A type's availability column and sort take its own word, such as `Next airing`, and `All` reads `Available` (P-036).
- `library-live` a Work added, changed or deleted through the API appears, changes or leaves without a reload (P-242).
- `library-tv` on `series`, TV's posters read their status and tally over monitored episodes, such as `Missing · 0 of 3 met` for The Lantern Keepers added with First season; its filters add `Network` and `Production status`, `Ended (1)` leaves only Harbor Nights, and the filter is still set after a visit to Movies and back (P-035, P-036).

- `library-pending-link` on `home` with the shelves added as roots ([Pending imports](./pending-imports.md)), the Library reads `3 folders need you` over `A library scan found them and can't match them with confidence.` above the filters, linking to `/library/pending`, and the sidebar's `Library` row carries the count; the row leaves once no folder needs the Admin (P-024).
- `library-scan-filters` on the same run, `Filter` adds `Files gone (<n>)` and `Added by a library scan (<n>)` beside `Needs you`. `Added by a library scan (3)` keeps Copper Kettle, Lanterns at Low Tide and Saltmarsh Radio as `scanned=1`, and once a scan marks files gone `Files gone (2)` keeps their Works as `gone=1`, which Health's `Show the Works` opens with the root, `?root=<root id>&gone=1`. `GET /api/v1/library` answers `goneFiles` and `addedBy` (`admin` or `scan`) per Work (P-024, P-026).

## How to get to it (user POV)

- Desktop shell: the sidebar's `Library` row for every type, and the Content Type row under it (`Movies`) for one.
- Phone shell: the `Library` tab, and `More` > `Movies`.
- Desktop and phone shells: the `<n> folders need you` row opens Pending imports, and the Pending imports lead's `Show the movies it added` opens the Library filtered to `scanned=1` and its root.
- API: `GET /api/v1/library` (the projection, with `goneFiles` and `addedBy` per Work), `GET /api/v1/library?workIds=` (named rows), `GET /api/v1/works/search?q=&limit=0&contentType=`.
- Host CLI: none.

## Driving it with fakeworld and playwright cli

Preconditions:

- A fresh `movies` run, doctor passing, and `{runId}-admin-desktop` and `{runId}-admin-phone` signed in as in the skill's Drive.
- The Quiet Orchard and The Salt Road added, The Salt Road unmonitored:

  ```sh
  $FW exec $RUN -- curl -sS -X POST -H 'X-Api-Key: {apiKey}' -H 'Content-Type: application/json' -d '{"contentType":"movie","key":"9000001","preset":"all"}' '{nodes.home.url}/api/v1/works'
  $FW exec $RUN -- curl -sS -X POST -H 'X-Api-Key: {apiKey}' -H 'Content-Type: application/json' -d '{"contentType":"movie","key":"9000005","preset":"none"}' '{nodes.home.url}/api/v1/works'
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

- **Type switch.** This one needs two types on: a fresh `music` run on `dist/slipstream-e2e`, which registers music beside movies (doctor's build check names the e2e main, which is expected), with The Lanterns added (`{"contentType":"music","key":"a1000000-0000-4000-8000-000000000001","preset":"all"}`). On `/library` the switch reads `All`, `Movies`, `Music`. Choose `Table`, then `Music`: the Music Library opens on its own grid, its poster reading `Missing · 0 of 5 met`. Choose `All`: the address comes back as `/library?view=table`.

  ```sh
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop goto '{nodes.home.browserUrl}/library'
  $FW exec $RUN --proof library/library-type-switch -- $PW cli -s={runId}-admin-desktop run-code 'async page => page.getByRole("navigation", { name: "Content Types" }).ariaSnapshot()'
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop click 'getByRole("button", { name: "Library options" })'
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop click 'getByRole("menuitemcheckbox", { name: "Table" })'
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop press Escape
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop click 'getByRole("navigation", { name: "Content Types" }).getByRole("link", { name: "Music" })'
  $FW exec $RUN --proof library/library-type-switch -- $PW cli -s={runId}-admin-desktop screenshot --filename={runDir}/proof/library/library-type-switch/music-grid.png
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop click 'getByRole("navigation", { name: "Content Types" }).getByRole("link", { name: "All" })'
  $FW exec $RUN --proof library/library-type-switch -- $PW cli -s={runId}-admin-desktop run-code 'async page => { await page.getByRole("table").waitFor(); return page.url(); }'
  $FW exec $RUN -- $PW cli -s={runId}-admin-phone goto '{nodes.home.browserUrl}/library/movie'
  $FW exec $RUN --proof library/library-type-switch -- $PW cli -s={runId}-admin-phone screenshot --filename={runDir}/proof/library/library-type-switch/phone.png
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

- **Library scan's filters.** On a separate `home` run with the shelves added as roots, the Library leads with `3 folders need you`. Choose `Filter`, then `Added by a library scan (3)`. After [Library roots](./library-roots.md)' **A mass vanish stops, and Mark them gone settles it**, `/library/movie` with `Files gone (2)` keeps Copper Kettle and Saltmarsh Radio.

  ```sh
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop goto '{nodes.home.browserUrl}/library'
  $FW exec $RUN --proof library/library-pending-link -- $PW cli -s={runId}-admin-desktop screenshot --filename={runDir}/proof/library/library-pending-link/library.png
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop click 'getByRole("main").getByRole("button", { name: "Filter" })'
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop check 'getByRole("dialog", { name: "Filters" }).getByRole("checkbox", { name: /^Added by a library scan/ })'
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop press Escape
  $FW exec $RUN --proof library/library-scan-filters -- $PW cli -s={runId}-admin-desktop snapshot --filename={runDir}/proof/library/library-scan-filters/added-by-scan.aria.yml
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop goto '{nodes.home.browserUrl}/library/movie?gone=1'
  $FW exec $RUN --proof library/library-scan-filters -- $PW cli -s={runId}-admin-desktop snapshot --filename={runDir}/proof/library/library-scan-filters/files-gone.aria.yml
  $FW exec $RUN --proof library/library-scan-filters -- curl -sS -H 'X-Api-Key: {apiKey}' '{nodes.home.url}/api/v1/library'
  ```

## Gotchas

- Menu toggles stay open after a choice, as a column chooser should; press `Escape` before the next click outside the menu, or that click only closes it.
- The remembered view lives in the browser's storage per session; a fresh session opens on the default grid sorted by Title.
- `movies` adds no library root, so every Work reads `No library root` in the table until one is made; [Bulk edit](./bulk-edit.md) makes two.
- The grid draws only the rows near the window; a snapshot of a long Library holds a few dozen posters, not all of them.
