# Movie availability

Each movie Item carries the day it counts as out and the rule that chose it, in words. An Admin sets which release counts and the cinema delay as the movie Content Type's own settings, and a change sends one `works.type-changed` for the type.

## Sub-features

- `availability-on-the-work` `GET /api/v1/works/{id}` for The Quiet Orchard reads `"availability":{"date":"2021-04-16","basis":"Digital release in United States"}` on its one Item, from `home`'s fake-tmdb US digital date.
- `movie-settings-read` `GET /api/v1/content-types/movie/settings` lists `Movies count as out on` (choice, `home`, options Home release and Cinema release) and `A cinema release counts as out after` (duration, 7776000 seconds, 0 to 365 days in days), both at their defaults.
- `movie-settings-change` `PATCH` with `{"values":{"movie.countsAs":"cinema","movie.cinemaDelay":2592000}}` answers both fields changed, and the live stream carries one `works.type-changed` with `{"contentType":"movie"}`. The same `PATCH` again sends none. A delay of 400 days answers 422 `setting_out_of_range` with `Between 0 s and 365 days.`, an unknown key 422 `unknown_setting`, and nulls reset both.

## How to get to it (user POV)

- No screen yet: S20 (Library, the add flow and the Work page) builds the Work page's availability line and the movie settings screen.
- API: `GET /api/v1/works/{id}` (each Item's `availability`), `GET` and `PATCH /api/v1/content-types/{contentType}/settings`, and the `works.type-changed` event on `GET /api/v1/events`.

## Driving it with fakeworld and curl

Preconditions:

- The baseline: a `home` run with doctor passing. No browser session is needed.

Steps:

- **Add The Quiet Orchard and read its availability.** The Item's `availability` is the US digital date.

  ```sh
  $FW exec $RUN -- curl -sS -X POST -H 'X-Api-Key: {apiKey}' -H 'Content-Type: application/json' -d '{"contentType":"movie","key":"9000001","monitor":true}' '{nodes.home.url}/api/v1/works'
  $FW exec $RUN --proof movie-availability/availability-on-the-work -- curl -sS -H 'X-Api-Key: {apiKey}' '{nodes.home.url}/api/v1/works/<id from the add>'
  ```

- **Read the movie settings.**

  ```sh
  $FW exec $RUN --proof movie-availability/movie-settings-read -- curl -sS -H 'X-Api-Key: {apiKey}' '{nodes.home.url}/api/v1/content-types/movie/settings'
  ```

- **Change them, repeat, refuse and reset.** Count `works.type-changed` records after each step with `sse list`.

  ```sh
  $FW exec $RUN --proof movie-availability/movie-settings-change -- curl -sS -X PATCH -H 'X-Api-Key: {apiKey}' -H 'Content-Type: application/json' -d '{"values":{"movie.countsAs":"cinema","movie.cinemaDelay":2592000}}' '{nodes.home.url}/api/v1/content-types/movie/settings'
  $FW sse wait $RUN --as admin --event works.type-changed --timeout 10s
  $FW exec $RUN --proof movie-availability/movie-settings-change -- curl -sS -X PATCH -H 'X-Api-Key: {apiKey}' -H 'Content-Type: application/json' -d '{"values":{"movie.countsAs":"cinema","movie.cinemaDelay":2592000}}' '{nodes.home.url}/api/v1/content-types/movie/settings'
  $FW exec $RUN --proof movie-availability/movie-settings-change -- curl -sS -i -X PATCH -H 'X-Api-Key: {apiKey}' -H 'Content-Type: application/json' -d '{"values":{"movie.cinemaDelay":34560000}}' '{nodes.home.url}/api/v1/content-types/movie/settings'
  $FW exec $RUN --proof movie-availability/movie-settings-change -- curl -sS -i -X PATCH -H 'X-Api-Key: {apiKey}' -H 'Content-Type: application/json' -d '{"values":{"movie.region":"GB"}}' '{nodes.home.url}/api/v1/content-types/movie/settings'
  $FW exec $RUN --proof movie-availability/movie-settings-change -- curl -sS -X PATCH -H 'X-Api-Key: {apiKey}' -H 'Content-Type: application/json' -d '{"values":{"movie.countsAs":null,"movie.cinemaDelay":null}}' '{nodes.home.url}/api/v1/content-types/movie/settings'
  $FW sse list $RUN --as admin --event works.type-changed
  ```

## Gotchas

- `home`'s catalog holds one movie with one US digital date, so neither setting moves its date. The rule's other branches (any country, cinema plus the delay, the primary date plus the delay, and the cinema setting) are proven by `TestAvailabilityRule` in `internal/contenttypes/movie`.
- The delay is in seconds on the wire and set in whole days.
- The reset in the last step changes both settings, so it sends a second `works.type-changed`.
