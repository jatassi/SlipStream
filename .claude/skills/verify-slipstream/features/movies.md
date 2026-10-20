# Movies

A movie added from the metadata catalog carries its whole record: titles in the Admin's language, English and the original, overview, tagline, runtime, genres, certification, studio, collection, cast and crew, trailer, every release date, TMDB's rating, OMDb's IMDb, Rotten Tomatoes and Metacritic ratings with the awards line, and its artwork. Previewing a movie reads TMDB once and never OMDb, and adding it right after reuses that read. A refresh keeps the TMDB id and what SlipStream already knew. A provider that is down, rate limited, refusing the key or out of its daily allowance answers in its status and in a plain problem, and a Worker holding Metadata makes every call from its own network and keeps no key on its disk. This file covers the lifecycle end to end on the `movies` Scenario; [Add a movie](./add-a-movie.md), [Artwork](./artwork.md), [Movie availability](./movie-availability.md), [Metadata providers](./metadata-providers.md), [Metadata settings](./metadata-settings.md) and [Metadata refresh](./metadata-refresh.md) cover each part on `home`.

## Sub-features

- `movies-whole-record` `POST /api/v1/works` for The Salt Road (`9000005`) answers every field from `english_title` to `awards` (`1 win & 3 nominations`), with `original_title` `La Route du Sel`, `certification` `PG-13` from `US`, four people with a `photoUrl` each, ratings from `imdb`, `metacritic`, `rottentomatoes` and `tmdb` each with its `fetchedAt`, and the `backdrop`, `poster`, `studio_logo` and `title_treatment` artwork. Its Item's availability reads `2023-04-04`, `Digital release in United States`, and the Work page on desktop and phone shows the record (P-286, P-054, P-056, P-058).
- `movies-preview-then-add` `GET /api/v1/content-types/movie/catalog/9000005` opens a connection to fake-tmdb and none to fake-omdb, and answers the TMDB rating alone. With every TMDB `movie` request then failing, the add still answers 200 with the whole record, the fault never fires, and fake-omdb sees one connection (P-054, P-060).
- `movies-refresh` after the `tmdb-change` cue, `POST /api/v1/works/{id}/refresh` answers the overview `Two sisters walk the salt road. Revision 1.` and TMDB's 411 votes with the OMDb ratings kept; `POST /api/v1/content-types/movie/refresh` answers 202 and its run ends `succeeded` with `1 Work`. After the `tmdb-delete` cue, a refresh answers 409 `work_gone_upstream` and the Work keeps its record with `goneUpstreamAt` set (P-038, P-231).
- `movies-unreachable` with fake-tmdb answering 503 (or refusing connections, as the end-to-end test plays it), a search answers 502 `metadata_unavailable` naming TMDB and TMDB reads `unreachable`, `SlipStream can't reach TMDB. It tries again on its own; …`. Once the fault clears, TMDB reads `working` again with no call from anyone, from its own retest (P-253, P-061).
- `movies-rate-limited` a TMDB 429 with `Retry-After: 30` answers 502 `metadata_unavailable` with `TMDB: is rate limited until <time>`, TMDB reads `rate_limited` until that time, and a second search inside it answers the same without reaching fake-tmdb (P-061).
- `movies-key-refused` after `tmdb-revoke-key`, a search answers 502 `metadata_refused`, `TMDB refused SlipStream's key`, and TMDB reads `key_refused`, `This build has no TMDB key. Add your own.`; an Admin key fake-tmdb takes turns it `working` on `yours` (P-063).
- `movies-omdb-cap` after `omdb-cap`, adding a movie still answers 200, and OMDb reads `out_of_allowance` until the next UTC midnight, `OMDb is out of today's allowance. It resets on its own.` (P-056, P-061).
- `movies-half-image` with fake-image-cdn cutting every image off mid-answer, a size never fetched answers 404 `artwork_unavailable` with `no-store`, and a size already stored still answers the same bytes. Once image-cdn answers whole again, the size answers 200 after `timings.metadata.artwork_retry` (5 seconds in `movies`) (P-059).
- `movies-worker` with Metadata moved to seedbox alone and home's `tmdb` front refused, saving an Admin key is tested from seedbox, the add answers through seedbox's `tmdb` front, a fault on seedbox's front fails the search, and no file in seedbox's data dir holds the key or a `library.db`.

## How to get to it (user POV)

- Desktop and phone shells: the Work page at `/works/{id}` shows the stored record. The add sheet, the provider key screen and posters in lists belong to S20 (Library, the add flow and the Work page).
- API: `GET /api/v1/content-types/movie/catalog?query=`, `GET /api/v1/content-types/movie/catalog/{key}`, `POST /api/v1/works`, `GET /api/v1/works/{id}`, `POST /api/v1/works/{id}/refresh`, `POST /api/v1/content-types/movie/refresh`, `GET /api/v1/tasks/metadata-refresh`, `GET /api/v1/metadata/providers`, `PUT /api/v1/metadata/providers/{provider}/key`, `GET /api/v1/artwork/{token}?size=` with the session cookie.
- Host CLI: none.

## Driving it with fakeworld and playwright cli

Preconditions:

- A fresh `movies` run (`--scenario movies`), doctor passing, and `{runId}-admin-desktop` and `{runId}-admin-phone` signed in as in the skill's Drive.
- `one.js` and `one()` from [Artwork](./artwork.md)'s Preconditions.
- The Go end-to-end tier proves the same lifecycle in `e2e/movie_test.go`: `TestAMovieIsAddedWithItsWholeRecordFromThePreviewsOneFetch`, `TestArtworkIsServedAtEachSizeAndNeverFromAHalfImage`, `TestAMovieRefreshesAloneAndWithEveryMovieAndKeepsItsRecordOnceGone`, `TestAProviderFaultEndsInItsStatusAndProblem` and `TestMetadataOnAWorkerCallsFromItsFrontAndLeavesNoKeyOnItsDisk`.

Steps:

- **Preview, then add on that one read.** The preview names no OMDb rating, and home's count of `front.open` records on `omdb` is the same before and after it (doctor's `every fake answers` opens one through each front). Fail TMDB's `movie` route, then add: 200 with the whole record, and `fault list` shows the rule never fired. Set `W` to the reply's `work.id`.

  ```sh
  grep '"front.open","node":"home"' $RUN/events.jsonl | grep -c '"omdb"'
  $FW exec $RUN --proof movies/movies-preview-then-add -- curl -sS -H 'X-Api-Key: {apiKey}' '{nodes.home.url}/api/v1/content-types/movie/catalog/9000005'
  grep '"front.open","node":"home"' $RUN/events.jsonl | grep -c '"omdb"'
  $FW fault set $RUN second-fetch tmdb status=500 --route movie
  $FW exec $RUN --proof movies/movies-whole-record -- curl -sS -X POST -H 'X-Api-Key: {apiKey}' -H 'Content-Type: application/json' -d '{"contentType":"movie","key":"9000005","monitor":true}' '{nodes.home.url}/api/v1/works'
  grep -c '"rule":"second-fetch","target"' $RUN/events.jsonl
  $FW fault clear $RUN second-fetch
  ```

- **The Work page.** Open the Work on desktop and phone.

  ```sh
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop goto "{nodes.home.browserUrl}/works/$W"
  $FW exec $RUN --proof movies/movies-whole-record -- $PW cli -s={runId}-admin-desktop screenshot --full-page --filename={runDir}/proof/movies/movies-whole-record/desktop.png
  $FW exec $RUN -- $PW cli -s={runId}-admin-phone goto "{nodes.home.browserUrl}/works/$W"
  $FW exec $RUN --proof movies/movies-whole-record -- $PW cli -s={runId}-admin-phone screenshot --full-page --filename={runDir}/proof/movies/movies-whole-record/phone.png
  ```

- **Half an image.** Set `T` to the `title_treatment` URL and `P` to the poster's. Fetch the poster small, cut image-cdn's answers off, then fetch the title treatment small (404) and the poster small (200). Clear the fault, wait five seconds and fetch the title treatment again (200).

  ```sh
  $FW exec $RUN --proof movies/movies-half-image -- $PW cli -s={runId}-admin-desktop run-code "$(one "$P?size=small")"
  $FW fault set $RUN half-image image-cdn truncate --route image
  $FW exec $RUN --proof movies/movies-half-image -- $PW cli -s={runId}-admin-desktop run-code "$(one "$T?size=small")"
  $FW exec $RUN --proof movies/movies-half-image -- $PW cli -s={runId}-admin-desktop run-code "$(one "$P?size=small")"
  $FW fault clear $RUN half-image
  $FW exec $RUN -- sleep 5
  $FW exec $RUN --proof movies/movies-half-image -- $PW cli -s={runId}-admin-desktop run-code "$(one "$T?size=small")"
  ```

- **Refresh one and all, then gone.**

  ```sh
  $FW cue $RUN tmdb-change '{"id":9000005}'
  $FW exec $RUN --proof movies/movies-refresh -- curl -sS -X POST -H 'X-Api-Key: {apiKey}' "{nodes.home.url}/api/v1/works/$W/refresh"
  $FW exec $RUN --proof movies/movies-refresh -- curl -sS -i -X POST -H 'X-Api-Key: {apiKey}' '{nodes.home.url}/api/v1/content-types/movie/refresh'
  $FW exec $RUN --proof movies/movies-refresh -- curl -sS -H 'X-Api-Key: {apiKey}' '{nodes.home.url}/api/v1/tasks/metadata-refresh'
  $FW cue $RUN tmdb-delete '{"id":9000005}'
  $FW exec $RUN --proof movies/movies-refresh -- curl -sS -i -X POST -H 'X-Api-Key: {apiKey}' "{nodes.home.url}/api/v1/works/$W/refresh"
  $FW exec $RUN --proof movies/movies-refresh -- curl -sS -H 'X-Api-Key: {apiKey}' "{nodes.home.url}/api/v1/works/$W"
  ```

- **Unreachable, then back on its own.**

  ```sh
  $FW fault set $RUN down tmdb status=503
  $FW exec $RUN --proof movies/movies-unreachable -- curl -sS -H 'X-Api-Key: {apiKey}' '{nodes.home.url}/api/v1/content-types/movie/catalog?query=orchard'
  $FW exec $RUN --proof movies/movies-unreachable -- curl -sS -H 'X-Api-Key: {apiKey}' '{nodes.home.url}/api/v1/metadata/providers'
  $FW fault clear $RUN down
  $FW exec $RUN -- sleep 10
  $FW exec $RUN --proof movies/movies-unreachable -- curl -sS -H 'X-Api-Key: {apiKey}' '{nodes.home.url}/api/v1/metadata/providers'
  ```

- **OMDb's cap, a revoked key, a 429.** Each search uses a new query, so the holder's cache never answers it.

  ```sh
  $FW cue $RUN omdb-cap '{}'
  $FW exec $RUN --proof movies/movies-omdb-cap -- curl -sS -o /dev/null -w '%{{http_code}\n' -X POST -H 'X-Api-Key: {apiKey}' -H 'Content-Type: application/json' -d '{"contentType":"movie","key":"9000001"}' '{nodes.home.url}/api/v1/works'
  $FW cue $RUN tmdb-revoke-key '{}'
  $FW exec $RUN --proof movies/movies-key-refused -- curl -sS -H 'X-Api-Key: {apiKey}' '{nodes.home.url}/api/v1/content-types/movie/catalog?query=salt'
  $FW exec $RUN --proof movies/movies-key-refused -- curl -sS -X PUT -H 'X-Api-Key: {apiKey}' -H 'Content-Type: application/json' -d '{"key":"ada-own-key"}' '{nodes.home.url}/api/v1/metadata/providers/tmdb/key'
  $FW fault set $RUN limited tmdb status=429,retry-after=30s --route search
  $FW exec $RUN --proof movies/movies-rate-limited -- curl -sS -H 'X-Api-Key: {apiKey}' '{nodes.home.url}/api/v1/content-types/movie/catalog?query=quiet'
  $FW exec $RUN --proof movies/movies-rate-limited -- curl -sS -H 'X-Api-Key: {apiKey}' '{nodes.home.url}/api/v1/content-types/movie/catalog?query=harbor'
  $FW exec $RUN --proof movies/movies-rate-limited -- curl -sS -H 'X-Api-Key: {apiKey}' '{nodes.home.url}/api/v1/metadata/providers'
  $FW fault clear $RUN limited
  ```

- **Metadata on seedbox.** Hand Metadata to seedbox alone (the API form of [Roles](./roles.md)'s `roles-metadata`), refuse home's TMDB front, save a key, which seedbox tests, and search: the new `front.open` records on `tmdb` are seedbox's. No file in seedbox's data dir holds the key.

  ```sh
  M=$RUN/manifest.json
  curl -sS -H "X-Api-Key: $(jq -r .apiKey $M)" "$(jq -r '.nodes[0].url' $M)/api/v1/nodes/roles" | jq --arg h "$(jq -r '.nodes[0].nodeId' $M)" --arg s "$(jq -r '.nodes[1].nodeId' $M)" \
    '{basis, set: [{nodeId: $h, roles: ["storage","converting","subtitles","media-server-link","notifying","serving"]}, {nodeId: $s, roles: ["searching","downloading","metadata"]}]}' > $RUN/roles.json
  $FW exec $RUN --proof movies/movies-worker -- curl -sS -X PUT -H 'X-Api-Key: {apiKey}' -H 'Content-Type: application/json' -d @{runDir}/roles.json '{nodes.home.url}/api/v1/nodes/roles'
  $FW fault set $RUN home-cut-off tmdb@home refuse
  $FW exec $RUN --proof movies/movies-worker -- curl -sS -X PUT -H 'X-Api-Key: {apiKey}' -H 'Content-Type: application/json' -d '{"key":"seedbox-never-keeps-this"}' '{nodes.home.url}/api/v1/metadata/providers/tmdb/key'
  $FW exec $RUN --proof movies/movies-worker -- curl -sS -H 'X-Api-Key: {apiKey}' '{nodes.home.url}/api/v1/content-types/movie/catalog?query=quiet+orchard'
  grep '"front.open"' $RUN/events.jsonl | grep '"tmdb"' | tail -3
  $FW exec $RUN --proof movies/movies-worker -- grep -rIl seedbox-never-keeps-this {nodes.seedbox.dataDir}
  $FW exec $RUN --proof movies/movies-worker -- ls {nodes.seedbox.dataDir}
  $FW fault clear $RUN home-cut-off
  ```

## Gotchas

- Run the recipe on a fresh `movies` run in the order above: the key and cap steps change every provider's status for the rest of the run, and `tmdb-delete` takes The Salt Road away.
- The holder caches answers for an hour: a search or preview repeated with the same query never reaches the fake, so a fault set after it never fires. Use a new query, or `DELETE /api/v1/metadata/cache`.
- A `refuse` fault refuses new connections only: a holder that already keeps a connection open to the fake goes on using it. Play `status=503` to fail every call.
- A TMDB key fake-tmdb revoked stays refused for the run; `tmdb-revoke-key '{}'` revokes every key it has seen, none included, so save a new Admin key to call TMDB again.
- `grep -rIl` exits 1 when no file holds the key, which is the pass; exec reports the exit code with the step.
- OMDb's cap lasts until the next UTC midnight, and a world's clock never crosses it.
