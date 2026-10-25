# Metadata settings

Settings > Metadata says in one sentence which language titles read in and which country release dates and certifications come from, with those two choices beside it, and keeps the cache time, OMDb's ratings interval, the artwork store size and the Clear metadata cache button under Details. The same settings are on the API. A new region sends one `works.type-changed` per Content Type, and clearing the cache makes every Node reading metadata ask the providers again.

## Sub-features

- `metadata-settings-read` `GET /api/v1/metadata/settings` lists `Metadata language` (choice, `en-US`, TMDB's languages), `Metadata region` (choice, `US`, ISO 3166-1 countries), `Metadata cache keeps responses for` (duration, 3600 seconds, 0 to 1 day), `OMDb ratings no more often than every` (duration, 7 days) and `Artwork store size` (count, 2048 MB), each at its default. The refresh tiers are not here: they are the Metadata refresh task's Tuning.
- `metadata-settings-change` `PATCH` with `{"values":{"metadata.region":"GB"}}` answers the region changed, and the live stream carries one `works.type-changed` per Content Type. The same `PATCH` again sends none. A region `ZZ` answers 422 `setting_out_of_range` with "ZZ is not one of the 250 choices for Metadata region." and changes nothing, `metadata.refresh.others` answers 422 `unknown_setting`, and `null` resets the region with one more `works.type-changed` per Content Type (P-058).
- `metadata-cache-clear` a preview read again within the cache time is answered from the holder's cache, so a TMDB fault set between the two reads never fires. `DELETE /api/v1/metadata/cache` answers 204, and the next preview reaches TMDB and meets the fault (P-060).
- `metadata-settings-screen` Settings > Metadata on desktop and phone leads with "Titles and overviews are in English (United States), and release dates and certifications come from United States." and the language and region choices; Details opens the rest and the Clear metadata cache button, whose toast reads "Cleared the metadata cache." (P-060).

## How to get to it (user POV)

- Desktop shell: Settings in the sidebar, then Metadata.
- Phone shell: More, then Settings, then Metadata.
- API: `GET` and `PATCH /api/v1/metadata/settings`, `DELETE /api/v1/metadata/cache` (Admin), and `works.type-changed` on `GET /api/v1/events`.

## Driving it with fakeworld, curl and playwright cli

Preconditions:

- The baseline: a `home` run with doctor passing and the `{runId}-admin-desktop` session open. Open a `{runId}-admin-phone` session the same way for the phone shell.

Steps:

- **Read the settings.**

  ```sh
  $FW exec $RUN --proof metadata-settings/metadata-settings-read -- curl -sS -H 'X-Api-Key: {apiKey}' '{nodes.home.url}/api/v1/metadata/settings'
  ```

- **Change the region, repeat, refuse and reset.** Count `works.type-changed` records after each step with `sse list`.

  ```sh
  $FW exec $RUN --proof metadata-settings/metadata-settings-change -- curl -sS -X PATCH -H 'X-Api-Key: {apiKey}' -H 'Content-Type: application/json' -d '{"values":{"metadata.region":"GB"}}' '{nodes.home.url}/api/v1/metadata/settings'
  $FW sse wait $RUN --as admin --event works.type-changed --timeout 10s
  $FW exec $RUN --proof metadata-settings/metadata-settings-change -- curl -sS -X PATCH -H 'X-Api-Key: {apiKey}' -H 'Content-Type: application/json' -d '{"values":{"metadata.region":"GB"}}' '{nodes.home.url}/api/v1/metadata/settings'
  $FW exec $RUN --proof metadata-settings/metadata-settings-change -- curl -sS -i -X PATCH -H 'X-Api-Key: {apiKey}' -H 'Content-Type: application/json' -d '{"values":{"metadata.language":"fr-FR","metadata.region":"ZZ"}}' '{nodes.home.url}/api/v1/metadata/settings'
  $FW exec $RUN --proof metadata-settings/metadata-settings-change -- curl -sS -i -X PATCH -H 'X-Api-Key: {apiKey}' -H 'Content-Type: application/json' -d '{"values":{"metadata.refresh.others":86400}}' '{nodes.home.url}/api/v1/metadata/settings'
  $FW exec $RUN --proof metadata-settings/metadata-settings-change -- curl -sS -X PATCH -H 'X-Api-Key: {apiKey}' -H 'Content-Type: application/json' -d '{"values":{"metadata.region":null}}' '{nodes.home.url}/api/v1/metadata/settings'
  $FW sse list $RUN --as admin --event works.type-changed
  ```

- **Clear the cache.** Preview The Quiet Orchard, set a 404 on TMDB's movie route, preview again (200, no `fault.fired`), clear the cache, preview again (502 `metadata_unavailable`, "TMDB: responded with 404 Not Found but no not-found body", one `fault.fired`), then clear the fault.

  ```sh
  $FW exec $RUN --proof metadata-settings/metadata-cache-clear -- curl -sS -o /dev/null -w '%{{http_code}\n' -H 'X-Api-Key: {apiKey}' '{nodes.home.url}/api/v1/content-types/movie/catalog/9000001'
  $FW fault set $RUN tmdb-gone tmdb status=404 --route movie
  $FW exec $RUN --proof metadata-settings/metadata-cache-clear -- curl -sS -o /dev/null -w '%{{http_code}\n' -H 'X-Api-Key: {apiKey}' '{nodes.home.url}/api/v1/content-types/movie/catalog/9000001'
  $FW exec $RUN --proof metadata-settings/metadata-cache-clear -- curl -sS -i -X DELETE -H 'X-Api-Key: {apiKey}' '{nodes.home.url}/api/v1/metadata/cache'
  $FW exec $RUN --proof metadata-settings/metadata-cache-clear -- curl -sS -i -H 'X-Api-Key: {apiKey}' '{nodes.home.url}/api/v1/content-types/movie/catalog/9000001'
  $FW fault clear $RUN tmdb-gone
  ```

- **The screen.** Open Settings > Metadata, then Details, then Clear metadata cache.

  ```sh
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop goto {nodes.home.browserUrl}/settings/metadata
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop screenshot --filename={runDir}/proof/metadata-settings/metadata-settings-screen/desktop.png
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop click 'getByRole("main").getByRole("button", { name: "Details" })'
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop click 'getByRole("button", { name: "Clear metadata cache" })'
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop snapshot --filename={runDir}/proof/metadata-settings/metadata-settings-screen/desktop-details.aria.yml
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop screenshot --filename={runDir}/proof/metadata-settings/metadata-settings-screen/desktop-details.png
  $FW exec $RUN -- $PW cli -s={runId}-admin-phone goto {nodes.home.browserUrl}/settings/metadata
  $FW exec $RUN -- $PW cli -s={runId}-admin-phone screenshot --filename={runDir}/proof/metadata-settings/metadata-settings-screen/phone.png
  ```

## Gotchas

- The registered Content Types on `home` decide how many `works.type-changed` one region change sends; read `contentType` on each record rather than counting a fixed number.
- A new language or region asks for every Work to be refreshed at the next Metadata refresh run; no fetch happens at the change, so nothing on the Work changes until then.
- exec fills `{...}` tokens, so curl's `-w '%{http_code}'` is written `%{{http_code}` there.
- The cache is per holder and in memory. Setting `metadata.cacheFor` to 0 turns it off, so every preview reaches TMDB.
