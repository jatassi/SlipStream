# Add a movie

An Admin searches the metadata catalog from the Library's `Add` menu and adds a movie as a monitored Work. The Work page shows its fields and a cell for each Target, and the Library lists it.

## Sub-features

- `add-search` a title search lists catalog matches from the Fake world's TMDB with year and an `Add` button, from the `home` Scenario's catalog (P-256).
- `add-no-match` a search with no match reads `Nothing matches “<query>”.`
- `add-work` `Add The Quiet Orchard` opens the add sheet, whose `Profile` reads `Movies (default)`, and its `Add Movie` creates the Work and opens its Work page.
- `add-work-page` the Work page shows Year `2021`, Released `2021-04-16`, Runtime `1h 44m` and the table `Any cut cells` with the starter `1080p` Target reading `Missing` (P-259).
- `add-open-existing` a Work already in the library is a link, `The Quiet Orchard 2021 In the library`, in place of `Add`, and opens its Work page.

## How to get to it (user POV)

- Desktop shell: on the Library (the sidebar's `Library`), the top bar's `Add`, then `Add a Movie`; on the Movies library (the sidebar's `Movies`), the top bar's `Add a Movie`. Both open `/add?contentType=movie`.
- Phone shell: on the `Library` tab, `Add` beside the large title opens the action sheet `Add a Work` with `Add a Movie`.
- Both shells: the palette's `Add a Movie` (Ctrl+K or ⌘K on desktop, the `Search` tab on phone).
- Both shells: `/add?contentType=movie&query=<title>` runs the search directly. The Library at `/library` (`/` redirects there) lists the added Work.
- API: `GET /api/v1/content-types/{contentType}/catalog?query=`, `POST /api/v1/works`, `GET /api/v1/works`.

## Driving it with fakeworld and playwright cli

Preconditions:

- The baseline: a `home` run, doctor passing, `{runId}-admin-desktop` on the Library, which reads `Nothing in the library yet.`

Steps:

- **Search the catalog.** Choose `Add`, then `Add a Movie`, fill `Title` with `quiet orchard` and choose `Search`. The `Results` list reads `The Quiet Orchard 2021` with `Add`.

  ```sh
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop click 'getByRole("banner").getByRole("button", { name: "Add" })'
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop click 'getByRole("menuitem", { name: "Add a Movie" })'
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop fill 'getByRole("searchbox", { name: "Title" })' "quiet orchard"
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop click 'getByRole("main").getByRole("button", { name: "Search", exact: true })'
  $FW exec $RUN --proof add-a-movie/add-search -- $PW cli -s={runId}-admin-desktop snapshot --filename={runDir}/proof/add-a-movie/add-search/results.aria.yml
  ```

- **No match.** Go to `/add?contentType=movie&query=volcano`. The page reads `Nothing matches “volcano”.`

  ```sh
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop goto '{nodes.home.browserUrl}/add?contentType=movie&query=volcano'
  $FW exec $RUN --proof add-a-movie/add-no-match -- $PW cli -s={runId}-admin-desktop snapshot --filename={runDir}/proof/add-a-movie/add-no-match/results.aria.yml
  ```

- **Add the Work.** Search again, choose `Add The Quiet Orchard`, then `Add Movie` in the sheet. The Work page opens, headed `The Quiet Orchard`, with its fields and the `1080p` cell reading `Missing`.

  ```sh
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop goto '{nodes.home.browserUrl}/add?contentType=movie&query=quiet+orchard'
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop click 'getByRole("button", { name: "Add The Quiet Orchard" })'
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop click 'getByRole("dialog", { name: "Add The Quiet Orchard" }).getByRole("button", { name: "Add Movie" })'
  $FW exec $RUN --proof add-a-movie/add-work-page -- $PW cli -s={runId}-admin-desktop snapshot --filename={runDir}/proof/add-a-movie/add-work-page/work.aria.yml
  $FW exec $RUN --proof add-a-movie/add-work-page -- $PW cli -s={runId}-admin-desktop screenshot --filename={runDir}/proof/add-a-movie/add-work-page/work.png
  $FW exec $RUN --proof add-a-movie/add-work -- curl -sS -H 'X-Api-Key: {apiKey}' '{nodes.home.url}/api/v1/works'
  ```

- **Open an existing Work.** Go back to `/add?contentType=movie&query=quiet+orchard`. The result is the link `The Quiet Orchard 2021 In the library`, and the Library lists `The Quiet Orchard 2021`.

  ```sh
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop goto '{nodes.home.browserUrl}/add?contentType=movie&query=quiet+orchard'
  $FW exec $RUN --proof add-a-movie/add-open-existing -- $PW cli -s={runId}-admin-desktop snapshot --filename={runDir}/proof/add-a-movie/add-open-existing/results.aria.yml
  ```

## Gotchas

- The Add page's search button and the Work page's search button are both named `Search`; the first searches the catalog, the second the Indexers. The top bar's capsule `Search Works or run a command` matches `Search` too. Scope the catalog's to `getByRole("main")` with `exact: true`; on desktop the Work page's sits in `getByRole("banner")` as the page's action.
- `/add` needs `contentType`: without it, or with a type that isn't enabled, the page reads `There's nothing at this address.`
- `home`'s catalog holds one movie, The Quiet Orchard (2021). Other titles find nothing.
- Adding a Work is permanent for the run: a recipe that needs an empty library wants a fresh run.
