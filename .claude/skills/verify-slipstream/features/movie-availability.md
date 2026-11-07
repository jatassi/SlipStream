# Movie availability

Each movie Item carries the day it counts as released and the rule that chose it, in words. An Admin sets which release counts and the cinema delay as the movie Content Type's own settings, on the API or in Movies' row of Settings > Content Types, and a change sends one `works.type-changed` for the type. The Controller's availability timer announces each monitored Item on the day it counts as released, and on start catches up every day since the last one it announced, kept in the Controller's `local.db`.

## Sub-features

- `availability-on-the-work` `GET /api/v1/works/{id}` for The Quiet Orchard reads `"availability":{"date":"2021-04-16","basis":"Digital release in United States"}` on its one Item, from `home`'s fake-tmdb US digital date.
- `movie-settings-read` `GET /api/v1/content-types/movie/settings` lists `Movies count as released on` (choice, `home`, options Home release and Cinema release) and `Delay after a cinema release` (duration, 7776000 seconds, 0 to 365 days in days), both at their defaults.
- `announced-day` as `home` starts, its log carries one `announced released Items` line, `after` yesterday `through` today with `items` 0: a Controller with no announced day in its `local.db` reads it as yesterday. Adding The Quiet Orchard monitored adds no line (2021 is before the day announced), and after `home` stops and starts the log holds two `ready` lines and still one announcement, because `local.db` kept the day (P-031).
- `movie-settings-change` `PATCH` with `{"values":{"movie.countsAs":"cinema","movie.cinemaDelay":2592000}}` answers both fields changed, and the live stream carries one `works.type-changed` with `{"contentType":"movie"}`. The same `PATCH` again sends none. A delay of 400 days answers 422 `setting_out_of_range` with `Between 0 s and 365 days.`, an unknown key 422 `unknown_setting`, and nulls reset both.
- `movie-settings-screen` Settings > Content Types on desktop and phone has a Movies row with its switch on, then the monitor-on-add switch, `Movies count as released on` and `Delay after a cinema release` through the shared setting control, then Default Profile and Default root. Choosing Cinema release saves at once and the stream carries one `works.type-changed` for `movie`.
- `movie-type-row` with The Quiet Orchard added, turning Movies off is refused: the switch stays on and the row reads "The Library still holds Movies. Delete them in the Movies Library, then turn Movies off." with `Movies Library` linking to `/library/movie`. Default Profile and Default root list Movies' Profiles and roots, and choosing one makes it the default at once, toasting "New Movies now take <Profile>." Refresh all reads "It hasn't run yet." on a fresh run, toasts "Refreshing every Movie from TMDB." and then reads "It's running now." or the run's time.

## How to get to it (user POV)

- Desktop shell: Settings in the sidebar, then Content Types. Phone shell: More, then Settings, then Content Types.
- The Work page's availability line is S20's Work page (W5), in `work-page.md`.
- No surface for the timer: its hook belongs to S34 (Automatic search, Backlog search and RSS sync), and its `works.changed` on a release day needs the clock to cross a midnight.
- API: `GET /api/v1/works/{id}` (each Item's `availability`), `GET` and `PATCH /api/v1/content-types/{contentType}/settings`, and the `works.type-changed` event on `GET /api/v1/events`.

## Driving it with fakeworld, curl and playwright cli

Preconditions:

- The baseline: a `home` run with doctor passing. The screen steps need the `{runId}-admin-desktop` session, and a `{runId}-admin-phone` session for the phone shell.

Steps:

- **Add The Quiet Orchard and read its availability.** The Item's `availability` is the US digital date.

  ```sh
  $FW exec $RUN -- curl -sS -X POST -H 'X-Api-Key: {apiKey}' -H 'Content-Type: application/json' -d '{"contentType":"movie","key":"9000001","monitor":true}' '{nodes.home.url}/api/v1/works'
  $FW exec $RUN --proof movie-availability/availability-on-the-work -- curl -sS -H 'X-Api-Key: {apiKey}' '{nodes.home.url}/api/v1/works/<id from the add>'
  ```

- **The announced day survives a restart.** After the add above, read `home`'s announcements, restart it and read them again.

  ```sh
  $FW exec $RUN --proof movie-availability/announced-day -- grep "announced released Items" {runDir}/nodes/home.log
  $FW node stop $RUN home
  $FW node start $RUN home
  $FW exec $RUN --proof movie-availability/announced-day -- grep -c '"msg":"ready"' {runDir}/nodes/home.log
  $FW exec $RUN --proof movie-availability/announced-day -- grep "announced released Items" {runDir}/nodes/home.log
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

- **The settings screen.** With The Quiet Orchard added, open Settings > Content Types, choose Cinema release, try to turn Movies off, change the default root and start Refresh all.

  ```sh
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop goto {nodes.home.browserUrl}/settings/content-types
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop snapshot --filename={runDir}/proof/movie-availability/movie-settings-screen/desktop.aria.yml
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop screenshot --filename={runDir}/proof/movie-availability/movie-settings-screen/desktop.png
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop click 'getByRole("article", { name: "Movies" }).getByRole("radio", { name: "Cinema release" })'
  $FW sse wait $RUN --as admin --event works.type-changed --timeout 10s
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop click 'getByRole("switch", { name: "Movies", exact: true })'
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop snapshot --filename={runDir}/proof/movie-availability/movie-type-row/refused.aria.yml
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop screenshot --filename={runDir}/proof/movie-availability/movie-type-row/refused.png
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop click 'getByRole("button", { name: "Refresh all" })'
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop snapshot --filename={runDir}/proof/movie-availability/movie-type-row/refresh-all.aria.yml
  $FW exec $RUN -- $PW cli -s={runId}-admin-phone goto {nodes.home.browserUrl}/settings/content-types
  $FW exec $RUN -- $PW cli -s={runId}-admin-phone screenshot --filename={runDir}/proof/movie-availability/movie-settings-screen/phone.png
  ```

  Reset the setting afterwards with the `null` PATCH above.

## Gotchas

- `home`'s catalog holds one movie with one US digital date, so neither setting moves its date. The rule's other branches (any country, cinema plus the delay, the primary date plus the delay, and the cinema setting) are proven by `TestAvailabilityRule` in `internal/contenttypes/movie`.
- The delay is in seconds on the wire and set in whole days.
- A world's clock doesn't cross a midnight and nothing may write a Node's `local.db`, so firing on the day, catching up after days off, re-arming on monitoring and settings, the hook getting only monitored Items and a failed hook getting them again are proven on a fake clock in `internal/metadata/availability_test.go`.
- The reset in the last step changes both settings, so it sends a second `works.type-changed`.
- `GET /api/v1/content-types/{contentType}/refresh` is a 501 stub until B3 (Add with Remember, metadata screens API) merges, and the Content Types screen loads it, so run the screen steps on a build with B3.
- Movies count as released on is a Segmented control of two options, so its options are radios.
