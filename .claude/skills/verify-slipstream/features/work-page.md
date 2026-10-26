# Work page

One Work on one scroll. The hero sets the Work's backdrop behind its poster, its title treatment as the heading, the facts line with its first rating, pills for what its cells add up to, its overview, and its actions: Search, Editions, Monitored and the `More actions` menu with Edit, Refresh metadata and Delete. Needs you lists what only an Admin can resolve. The Edition by Target matrix shows each wanted Edition's cells with their files, progress or failure, then Files and Kept files, About with the fields its type places there, and its Grabs and searches.

## Sub-features

- `work-hero` The Salt Road's page on `movies` shows the backdrop, the poster and the title treatment as an image inside the `h1` named `The Salt Road`, the facts line `2023 · Movie · 1h 52m · PG-13 · Drama, Adventure` with its first rating, the `Cells` pills, and the overview clamped to three lines with `More` (P-045).
- `work-needs-you` a Work with no library root reads `No library root for this Work` with `Open Library roots` under `Needs you`; a failed download reads one compact failure card named for the cells it holds back, with `Search again`, and the pills read `failed`.
- `work-matrix` the table `Editions and Targets` has one column per Target of the Work's Profile with its sourcing, `Downloaded`, and one row per wanted Edition reading where its list comes from, `From Profile <name>` or `Set for this Work`; each cell has its state chip and, while it lacks a file, `Search for <Target>, <Edition>`. On phone it is one card per Edition (P-208).
- `work-editions` `Editions` (desktop) or `Edit Editions` (phone, in the actions sheet) opens `Editions of <title>` with the heading `Editions from Profile <name>`; `Choose cuts` and `Add cut` make `Editions set for this Work`, and `Save` opens the Impact `Save the Editions of <title>`. Saving toasts `Saved the Editions of <title>.`, the rows read `Set for this Work`, and a cut no release has shown yet sits in its own row reading `Not seen yet. It's wanted once a release of it is found.`
- `work-files` `Files` reads the Work's library root and folder, then each file that fills a cell with its fills, quality, size and dimensions, and `Kept files` apart, each with `Kept file` and `Fills no wanted Target`; with none it reads `No files yet. A Grab that imports puts its file here.` (P-048).
- `work-about` About shows the tagline, Certification `PG-13 (United States)` with its `With` country, Studio `Fennel Films` beside the studio logo, Collection and Awards, one strip of photos per department of Cast and crew, each rating with its votes and fetch time, the TMDB page and trailer as links that open a new tab, the movie Item's `Counts as out on April 4, 2023, Digital release in United States.`, and `TMDB lists this movie as 9000005.` over `Details` (P-048, P-057).
- `work-edit` `Edit…` opens `Edit <title>` with `Profile` and `Library root`; a new Profile saves through the Impact `Give <title> the Profile <name>` and toasts `Changed the Profile of <title>.` (P-047).
- `work-refresh` `Refresh metadata` toasts `Refreshed the metadata of <title>.` and the page shows the new record; once TMDB drops the Work it toasts `TMDB no longer lists <title>, so it wasn't refreshed.`, the hero reads `Gone from TMDB`, `Refresh metadata is unavailable` and `TMDB no longer lists this movie. SlipStream keeps it as it is.` with `Delete`, the record stays, and `Refresh metadata` in the menu is disabled (P-047).
- `work-delete` `Delete…` opens the delete dialog; deleting toasts once and the page goes back to the Library, and a Work deleted elsewhere while its page is open toasts `<title> was deleted.` and leaves the same way (P-047).

## How to get to it (user POV)

- Desktop shell: a Work's poster or row in the Library, a Work in the palette, or `/works/{id}`. The hero's `More actions for <title>` holds `Edit…`, `Refresh metadata` and `Delete…`.
- Phone shell: the same Work, as a pushed page whose nav bar holds the title. `More actions for <title>` opens an action sheet with `Monitored`, `Edit Editions`, `Edit…`, `Refresh metadata` and `Delete…`.
- API: `GET /api/v1/works/{id}`, `GET /api/v1/works/{id}/cells` (the summary and Needs you), `GET /api/v1/works/{id}/files`, `GET` and `PUT /api/v1/works/{id}/wants` (`{"editions": ...}` or `{"editions": null}`, with `?preview=true`), `PUT /api/v1/works/profile`, `POST /api/v1/works/refresh` with one id, `POST /api/v1/works/delete`.
- Host CLI: none.

## Driving it with fakeworld and playwright cli

Preconditions:

- A fresh `movies` run, launched as in the skill's Launch, doctor passing, with `{runId}-admin-desktop` and `{runId}-admin-phone` signed in. It has no library root, so every Work reads `No library root for this Work`.
- The failed download and a Work's own Editions are proved by the browser tier's `web/e2e/work-page.spec.ts` on `targets`; drive them by hand on a `targets` run as that spec does.

Steps:

- **Add The Salt Road and open it.** Set `W` to the reply's `work.id`.

  ```sh
  $FW exec $RUN -- curl -sS -X POST -H 'X-Api-Key: {apiKey}' -H 'Content-Type: application/json' -d '{"contentType":"movie","key":"9000005","monitor":true}' '{nodes.home.url}/api/v1/works'
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop goto "{nodes.home.browserUrl}/works/$W"
  $FW exec $RUN --proof work-page/work-hero -- $PW cli -s={runId}-admin-desktop snapshot --filename={runDir}/proof/work-page/work-hero/desktop.aria.yml
  $FW exec $RUN --proof work-page/work-hero -- $PW cli -s={runId}-admin-desktop screenshot --full-page --filename={runDir}/proof/work-page/work-hero/desktop.png
  $FW exec $RUN -- $PW cli -s={runId}-admin-phone goto "{nodes.home.browserUrl}/works/$W"
  $FW exec $RUN --proof work-page/work-hero -- $PW cli -s={runId}-admin-phone screenshot --full-page --filename={runDir}/proof/work-page/work-hero/phone.png
  ```

- **Needs you, the matrix, Files and About.** Read them from the desktop snapshot above: `Needs you` with `No library root for this Work`, the `Editions and Targets` table, `No files yet.`, and About's values.

  ```sh
  $FW exec $RUN --proof work-page/work-about -- $PW cli -s={runId}-admin-desktop run-code 'async page => page.getByRole("region", { name: "About" }).ariaSnapshot()'
  $FW exec $RUN --proof work-page/work-about -- $PW cli -s={runId}-admin-desktop run-code 'async page => page.getByRole("list", { name: "Links" }).getByRole("link").evaluateAll((links) => links.map((a) => [a.textContent, a.href, a.target, a.rel]))'
  ```

- **Edit.** Open `Edit…`, read the dialog, and close it.

  ```sh
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop click 'getByRole("button", { name: "More actions for The Salt Road" })'
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop click 'getByRole("menuitem", { name: "Edit…" })'
  $FW exec $RUN --proof work-page/work-edit -- $PW cli -s={runId}-admin-desktop screenshot --filename={runDir}/proof/work-page/work-edit/edit.png
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop press Escape
  ```

- **Refresh, then gone.**

  ```sh
  $FW cue $RUN tmdb-change '{"id":9000005}'
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop click 'getByRole("button", { name: "More actions for The Salt Road" })'
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop click 'getByRole("menuitem", { name: "Refresh metadata" })'
  $FW exec $RUN --proof work-page/work-refresh -- $PW cli -s={runId}-admin-desktop screenshot --filename={runDir}/proof/work-page/work-refresh/refreshed.png
  $FW cue $RUN tmdb-delete '{"id":9000005}'
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop click 'getByRole("button", { name: "More actions for The Salt Road" })'
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop click 'getByRole("menuitem", { name: "Refresh metadata" })'
  $FW exec $RUN --proof work-page/work-refresh -- $PW cli -s={runId}-admin-desktop screenshot --full-page --filename={runDir}/proof/work-page/work-refresh/gone.png
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop click 'getByRole("button", { name: "More actions for The Salt Road" })'
  $FW exec $RUN --proof work-page/work-refresh -- $PW cli -s={runId}-admin-desktop snapshot --filename={runDir}/proof/work-page/work-refresh/menu.aria.yml
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop press Escape
  ```

- **Delete.** Choose `Delete…`, then the dialog's delete button. The page goes back to the Library, and `GET /api/v1/works/$W` answers 404.

  ```sh
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop click 'getByRole("main").getByRole("button", { name: "Delete" })'
  $FW exec $RUN --proof work-page/work-delete -- $PW cli -s={runId}-admin-desktop screenshot --filename={runDir}/proof/work-page/work-delete/dialog.png
  ```

## Gotchas

- On phone the nav bar holds the page's `h1`, so the hero's title is not a heading there: find the Work by `getByRole("heading", { name: "<title>", exact: true })`, which matches one element in either shell.
- The hero's backdrop shows only once the ambient has sampled it, so a screenshot taken at once can lack it. Wait for the `img` inside the backdrop before capturing.
- `Search` is exact: each cell's own button is named `Search for <Target>, <Edition>`, and the upgrades list has its own `Search` per row inside `Needs you`.
- `Refresh metadata` while gone upstream is disabled, not hidden; the warning beside the actions says why.
