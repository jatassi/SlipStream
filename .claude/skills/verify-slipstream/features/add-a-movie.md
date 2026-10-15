# Add a movie

An Admin searches the metadata catalog from the Library's Add movie link and adds a movie as a monitored Work. The Work page shows its fields and a cell for each Target, and the Library lists it.

## Sub-features

- `add-search` a title search lists catalog matches from the Fake world's TMDB with year and an `Add` button, from the `home` Scenario's catalog (P-256).
- `add-no-match` a search with no match reads `Nothing matches “<query>”.`
- `add-work` `Add The Quiet Orchard` creates the Work and opens its Work page.
- `add-work-page` the Work page shows Year `2021`, Released `2021-04-16`, Runtime `1h 44m` and a Cells table with the Scenario's `1080p` Target reading `Missing` (P-259).
- `add-open-existing` a Work already in the library shows an `Open The Quiet Orchard` link in place of `Add`.

## How to get to it (user POV)

- Desktop and phone: the `Add movie` link beside the Library heading, which opens `/add`.
- Desktop and phone: `/add?query=<title>` runs the search directly.
- API: `GET /api/v1/content-types/{contentType}/catalog?query=`, `POST /api/v1/works`, `GET /api/v1/works`.

## Driving it with fakeworld and playwright cli

Preconditions:

- The baseline: a `home` run, doctor passing, `{runId}-admin-desktop` on the Library, which reads `Nothing in the library yet.`

Steps:

- **Search the catalog.** Choose `Add movie`, fill `Title` with `quiet orchard` and choose `Search`. The `Results` list reads `The Quiet Orchard 2021` with `Add`.

  ```sh
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop click 'getByRole("link", { name: "Add movie" })'
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop fill 'getByRole("searchbox", { name: "Title" })' "quiet orchard"
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop click 'getByRole("button", { name: "Search" })'
  $FW exec $RUN --proof add-a-movie/add-search -- $PW cli -s={runId}-admin-desktop snapshot --filename={runDir}/proof/add-a-movie/add-search/results.aria.yml
  ```

- **No match.** Go to `/add?query=volcano`. The page reads `Nothing matches “volcano”.`

  ```sh
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop goto '{nodes.home.browserUrl}/add?query=volcano'
  $FW exec $RUN --proof add-a-movie/add-no-match -- $PW cli -s={runId}-admin-desktop snapshot --filename={runDir}/proof/add-a-movie/add-no-match/results.aria.yml
  ```

- **Add the Work.** Search again and choose `Add The Quiet Orchard`. The Work page opens, headed `The Quiet Orchard`, with its fields and the `1080p` cell reading `Missing`.

  ```sh
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop goto '{nodes.home.browserUrl}/add?query=quiet+orchard'
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop click 'getByRole("button", { name: "Add The Quiet Orchard" })'
  $FW exec $RUN --proof add-a-movie/add-work-page -- $PW cli -s={runId}-admin-desktop snapshot --filename={runDir}/proof/add-a-movie/add-work-page/work.aria.yml
  $FW exec $RUN --proof add-a-movie/add-work-page -- $PW cli -s={runId}-admin-desktop screenshot --filename={runDir}/proof/add-a-movie/add-work-page/work.png
  $FW exec $RUN --proof add-a-movie/add-work -- curl -sS -H 'X-Api-Key: {apiKey}' '{nodes.home.url}/api/v1/works'
  ```

- **Open an existing Work.** Go back to `/add?query=quiet+orchard`. The result shows the link `Open The Quiet Orchard`, and the Library lists `The Quiet Orchard 2021`.

  ```sh
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop goto '{nodes.home.browserUrl}/add?query=quiet+orchard'
  $FW exec $RUN --proof add-a-movie/add-open-existing -- $PW cli -s={runId}-admin-desktop snapshot --filename={runDir}/proof/add-a-movie/add-open-existing/results.aria.yml
  ```

## Gotchas

- The Add page's search button and the Work page's search button are both named `Search`; the first searches the catalog, the second the Indexers.
- `home`'s catalog holds one movie, The Quiet Orchard (2021). Other titles find nothing.
- Adding a Work is permanent for the run: a recipe that needs an empty library wants a fresh run.
