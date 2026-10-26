# Add a movie

An Admin finds a movie in the metadata catalog from the add page or the palette and adds it as a Work. The add sheet opens on the movie itself and one sentence saying what Add will do, with the Profile, the root, Monitor and Remember these choices below it. Remember on makes a changed choice the Content Type's default for the next add. Adding lands on the Work page, and a result already in the library opens its Work page instead.

## Sub-features

- `add-search` a title search on `/add?contentType=movie` lists up to 25 catalog matches from the Fake world's TMDB, each with a poster slot, its year and original title, a line of overview when the record has one, and an `Add` button (P-256, P-041).
- `add-search-year` the API's search for `quiet orchard 2021` reads the title before the year filtered to 2021, then the whole query, and answers The Quiet Orchard once, with `original` and `overview` empty because the `home` record has none (P-054).
- `add-preview` the API's `GET /api/v1/content-types/movie/catalog/9000001` answers `inLibrary` false and the tree adding would make, with every field `add-work-page` lists and the Item unmonitored; `GET /api/v1/library?contentType=movie` still lists nothing, and fake-omdb sees no connection (P-054). Once added, the same call answers `inLibrary` true and the stored Work's id.
- `add-no-match` a search with no match reads `Nothing matches "<query>". Try another spelling, or the Movie's original title.`
- `add-sheet` `Add The Quiet Orchard` opens the dialog `Add The Quiet Orchard` on the preview: the poster, the facts line starting `2021 · 1h 44m`, then `Adds The Quiet Orchard (2021) to Movies on <the movie root's path>, monitored, wanting Movies' 1 Target.`, then `Profile` reading `Movies (default)`, `Monitor` on, `Remember these choices` on, and `Adding doesn't search. To search now, use Search on its Work page.` With one movie root there is no `Library root` choice (P-041).
- `add-sheet-remember` turning `Monitor` off changes the sentence to `..., not monitored, ...` and shows `New Movies will start with these.`; turning `Remember these choices` off hides that line, and on again shows it.
- `add-work` `Add Movie` posts every choice with `remember`, toasts `Added The Quiet Orchard` and opens its Work page (P-041).
- `add-remember-default` an add with `Monitor` off and Remember on leaves `movie.add.monitor` false in `GET /api/v1/content-types/movie/settings`, the value Settings > Content Types shows as `New movies are monitored`.
- `add-remember` by API on a fresh run, `GET /api/v1/content-types/movie/settings` leads with `movie.add.monitor`, `New movies are monitored`, on by default. `POST /api/v1/works` with `remember` true, a Profile `Small` and `monitor` false answers `created` true with the Item unmonitored and makes `Small` the type's default Profile and `movie.add.monitor` off, publishing one `targets.changed`, after the one creating `Small` sent. The same add again with the `Movies` Profile and `monitor` true answers `created` false with the Work as it was, and moves no default. The in-process tests in `internal/worksapi/add_test.go` prove the root, a racing add, an add without `remember`, and a monitor left out taking `movie.add.monitor`.
- `add-work-page` the Work page shows the movie's record from fake-tmdb: English title and Original title `The Quiet Orchard`, Original language `en`, Year `2021`, Release date `2021-04-16`, Runtime `1h 44m`, Status `Released` and IMDb id `tt9000001`, and the table `Any cut cells` with the starter `1080p` Target reading `Missing` (P-259). `GET /api/v1/works/{id}` carries the same fields, and no ratings: `home`'s fake-omdb knows no title and its fake-tmdb lists no votes.
- `add-open-existing` a Work already in the library is a link, `The Quiet Orchard 2021 In the library`, in place of `Add`, and opens its Work page; adding it again by API answers `"created":false` with the same Work.
- `add-preview-unreachable` with fake-tmdb refusing the movie route, the sheet reads TMDB's problem detail in an alert and offers only `Close`.

## How to get to it (user POV)

- Desktop shell: on the Library (the sidebar's `Library`), the top bar's `Add`, then `Add a Movie`; on the Movies library (the sidebar's `Movies`), the top bar's `Add a Movie`. Both open `/add?contentType=movie`.
- Phone shell: on the `Library` tab, `Add` beside the large title opens the action sheet `Add a Work` with `Add a Movie`. The add sheet is a bottom sheet.
- Both shells: the palette's `Add a Movie` command, and its `Add a Movie` group of catalog results (see [shell.md](shell.md)): a result opens the add sheet over the current page, and `See all results` opens `/add?contentType=movie&query=<query>`.
- Both shells: `/add?contentType=movie&query=<title>` runs the search directly.
- API: `GET /api/v1/content-types/{contentType}/catalog?query=`, `GET /api/v1/content-types/{contentType}/catalog/{key}` (the sheet's preview), `POST /api/v1/works {contentType, key, profileId, rootId, monitor, remember}`, `GET` and `PATCH /api/v1/content-types/{contentType}/settings` (the `<type>.add.monitor` default), `GET /api/v1/library?contentType=`.

## Driving it with fakeworld and playwright cli

Preconditions:

- The baseline: a `home` run, doctor passing, `{runId}-admin-desktop` on the Library, which reads `Nothing in the library yet.`
- B1 (`GET /api/v1/library`) and B3 (`addWork` with Remember, `movie.add.monitor`) merged, so every route below is real.

Steps:

- **Search the catalog.** Choose `Add`, then `Add a Movie`, fill `Title` with `quiet orchard` and choose `Search`. The `Results` list reads `The Quiet Orchard 2021` with `Add`.

  ```sh
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop click 'getByRole("banner").getByRole("button", { name: "Add" })'
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop click 'getByRole("menuitem", { name: "Add a Movie" })'
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop fill 'getByRole("searchbox", { name: "Title" })' "quiet orchard"
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop click 'getByRole("main").getByRole("button", { name: "Search", exact: true })'
  $FW exec $RUN --proof add-a-movie/add-search -- $PW cli -s={runId}-admin-desktop snapshot --filename={runDir}/proof/add-a-movie/add-search/results.aria.yml
  ```

- **Search with a year and preview, by API.** On a fresh run, search with a trailing year, preview the record, and list the Library. The search answers The Quiet Orchard once; the preview answers `"inLibrary":false` with the record's fields; the Library's rows are empty. `events.jsonl` holds no front open to the `omdb` fake up to here.

  ```sh
  $FW exec $RUN --proof add-a-movie/add-search-year -- curl -sS -H 'X-Api-Key: {apiKey}' '{nodes.home.url}/api/v1/content-types/movie/catalog?query=quiet+orchard+2021'
  $FW exec $RUN --proof add-a-movie/add-preview -- curl -sS -H 'X-Api-Key: {apiKey}' '{nodes.home.url}/api/v1/content-types/movie/catalog/9000001'
  $FW exec $RUN --proof add-a-movie/add-preview -- curl -sS -H 'X-Api-Key: {apiKey}' '{nodes.home.url}/api/v1/library?contentType=movie'
  ```

- **Remember the choices, by API.** On a fresh run, read the type's settings, make a Profile `Small`, then add The Quiet Orchard remembering `Small` and `monitor` false. The add answers `"created":true` with `"monitored":false`; the Profiles list `Small` with `"default":true`; the settings read `movie.add.monitor` `"value":false`; the stream carries one `targets.changed` after the one creating `Small` sent. Add it again with the `Movies` Profile and `monitor` true: `"created":false`, and the Profiles and settings read as before.

  ```sh
  $FW exec $RUN --proof add-a-movie/add-remember -- curl -sS -H 'X-Api-Key: {apiKey}' '{nodes.home.url}/api/v1/content-types/movie/settings'
  MOVIES=$($FW exec $RUN -- curl -sS -H 'X-Api-Key: {apiKey}' '{nodes.home.url}/api/v1/profiles' | jq -r '.profiles[] | select(.name == "Movies") | .id')
  SMALL=$($FW exec $RUN -- curl -sS -X POST -H 'X-Api-Key: {apiKey}' -H 'Content-Type: application/json' -d '{"name":"Small","contentType":"movie","targets":[],"editions":{"kind":"any"}}' '{nodes.home.url}/api/v1/profiles' | jq -r .id)
  $FW exec $RUN --proof add-a-movie/add-remember -- curl -sS -X POST -H 'X-Api-Key: {apiKey}' -H 'Content-Type: application/json' -d '{"contentType":"movie","key":"9000001","profileId":"'$SMALL'","monitor":false,"remember":true}' '{nodes.home.url}/api/v1/works'
  $FW exec $RUN --proof add-a-movie/add-remember -- curl -sS -H 'X-Api-Key: {apiKey}' '{nodes.home.url}/api/v1/profiles'
  $FW exec $RUN --proof add-a-movie/add-remember -- curl -sS -H 'X-Api-Key: {apiKey}' '{nodes.home.url}/api/v1/content-types/movie/settings'
  $FW exec $RUN --proof add-a-movie/add-remember -- curl -sS -X POST -H 'X-Api-Key: {apiKey}' -H 'Content-Type: application/json' -d '{"contentType":"movie","key":"9000001","profileId":"'$MOVIES'","monitor":true,"remember":true}' '{nodes.home.url}/api/v1/works'
  $FW exec $RUN --proof add-a-movie/add-remember -- curl -sS -H 'X-Api-Key: {apiKey}' '{nodes.home.url}/api/v1/profiles'
  $FW exec $RUN --proof add-a-movie/add-remember -- curl -sS -H 'X-Api-Key: {apiKey}' '{nodes.home.url}/api/v1/content-types/movie/settings'
  $FW sse list $RUN --as admin --event targets.changed
  ```

- **No match.** Go to `/add?contentType=movie&query=volcano`. The page reads `Nothing matches "volcano".`

  ```sh
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop goto '{nodes.home.browserUrl}/add?contentType=movie&query=volcano'
  $FW exec $RUN --proof add-a-movie/add-no-match -- $PW cli -s={runId}-admin-desktop snapshot --filename={runDir}/proof/add-a-movie/add-no-match/results.aria.yml
  ```

- **Open the sheet and change a choice.** Search again and choose `Add The Quiet Orchard`. The sheet shows the preview and the sentence. Turn `Monitor` off: the sentence reads `not monitored` and `New Movies will start with these.` shows. Turn `Remember these choices` off and on again: the line hides and returns.

  ```sh
  SHEET='getByRole("dialog", { name: "Add The Quiet Orchard" })'
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop goto '{nodes.home.browserUrl}/add?contentType=movie&query=quiet+orchard'
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop click 'getByRole("button", { name: "Add The Quiet Orchard" })'
  $FW exec $RUN --proof add-a-movie/add-sheet -- $PW cli -s={runId}-admin-desktop snapshot "$SHEET" --filename={runDir}/proof/add-a-movie/add-sheet/sheet.aria.yml
  $FW exec $RUN --proof add-a-movie/add-sheet -- $PW cli -s={runId}-admin-desktop screenshot --filename={runDir}/proof/add-a-movie/add-sheet/sheet.png
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop click "$SHEET"'.getByRole("switch", { name: "Monitor" })'
  $FW exec $RUN --proof add-a-movie/add-sheet-remember -- $PW cli -s={runId}-admin-desktop snapshot "$SHEET" --filename={runDir}/proof/add-a-movie/add-sheet-remember/moved.aria.yml
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop click "$SHEET"'.getByRole("switch", { name: "Remember these choices" })'
  $FW exec $RUN --proof add-a-movie/add-sheet-remember -- $PW cli -s={runId}-admin-desktop snapshot "$SHEET" --filename={runDir}/proof/add-a-movie/add-sheet-remember/forget.aria.yml
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop click "$SHEET"'.getByRole("switch", { name: "Remember these choices" })'
  ```

- **Add the Work, remembering.** Choose `Add Movie`. The toast `Added The Quiet Orchard` shows and the Work page opens, headed `The Quiet Orchard`, with its fields and the `1080p` Target reading `Missing`. The type's settings now hold `movie.add.monitor` false.

  ```sh
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop click "$SHEET"'.getByRole("button", { name: "Add Movie" })'
  $FW exec $RUN --proof add-a-movie/add-work-page -- $PW cli -s={runId}-admin-desktop snapshot --filename={runDir}/proof/add-a-movie/add-work-page/work.aria.yml
  $FW exec $RUN --proof add-a-movie/add-work-page -- $PW cli -s={runId}-admin-desktop screenshot --filename={runDir}/proof/add-a-movie/add-work-page/work.png
  $FW exec $RUN --proof add-a-movie/add-work -- curl -sS -H 'X-Api-Key: {apiKey}' '{nodes.home.url}/api/v1/library?contentType=movie'
  $FW exec $RUN --proof add-a-movie/add-remember-default -- curl -sS -H 'X-Api-Key: {apiKey}' '{nodes.home.url}/api/v1/content-types/movie/settings'
  ```

- **Open an existing Work.** Go back to `/add?contentType=movie&query=quiet+orchard`. The result is the link `The Quiet Orchard 2021 In the library`; choosing it opens the Work page. Adding it again by API answers `"created":false` with the same Work id.

  ```sh
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop goto '{nodes.home.browserUrl}/add?contentType=movie&query=quiet+orchard'
  $FW exec $RUN --proof add-a-movie/add-open-existing -- $PW cli -s={runId}-admin-desktop snapshot --filename={runDir}/proof/add-a-movie/add-open-existing/results.aria.yml
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop click 'getByRole("link", { name: "The Quiet Orchard 2021 In the library" })'
  $FW exec $RUN --proof add-a-movie/add-open-existing -- $PW cli -s={runId}-admin-desktop eval 'location.pathname'
  $FW exec $RUN --proof add-a-movie/add-open-existing -- curl -sS -H 'X-Api-Key: {apiKey}' -H 'Content-Type: application/json' -d '{"contentType":"movie","key":"9000001"}' '{nodes.home.url}/api/v1/works'
  ```

- **A preview TMDB refuses.** On a fresh run, make fake-tmdb answer 500 on the movie route, then open the sheet. It shows an alert with the problem's detail and only `Close`.

  ```sh
  $FW fault set $RUN preview-gone tmdb status=500 --route movie
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop goto '{nodes.home.browserUrl}/add?contentType=movie&query=quiet+orchard'
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop click 'getByRole("button", { name: "Add The Quiet Orchard" })'
  $FW exec $RUN --proof add-a-movie/add-preview-unreachable -- $PW cli -s={runId}-admin-desktop snapshot "$SHEET" --filename={runDir}/proof/add-a-movie/add-preview-unreachable/sheet.aria.yml
  $FW fault clear $RUN preview-gone
  ```

## Gotchas

- The Add page's search button and the Work page's search button are both named `Search`; the first searches the catalog, the second the Indexers. The top bar's capsule `Search Works or run a command` matches `Search` too. Scope the catalog's to `getByRole("main")` with `exact: true`; on desktop the Work page's sits in `getByRole("banner")` as the page's action.
- `/add` needs `contentType`: without it, or with a type that isn't enabled, the page reads `There's nothing at this address.`
- `home`'s catalog holds one movie, The Quiet Orchard (2021). Other titles find nothing, so the sheet opens only before the add, and `add-preview-unreachable` wants a fresh run.
- fake-tmdb matches a query as a substring of a title, so the whole query `quiet orchard 2021` finds nothing there; only the year-filtered read finds the movie.
- Adding a Work is permanent for the run: a recipe that needs an empty library wants a fresh run.
