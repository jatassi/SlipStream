# Subtitles

An Admin adds a Wanted Language to a Profile, and SlipStream records a want for every library file of that Profile that no embedded text track, no audio in the language and no Sidecar meets yet. A subtitle file found beside a library file at its name, such as `Copper Kettle (2012).en.srt`, meets its language as a found Sidecar, also when the language is wanted only after the scan. Deleting a file removes the Sidecars beside it from the Storage Node that holds it, and only bytes SlipStream recorded: a file the Admin put at that name since stays.

## Sub-features

- `subtitles-wants` a Profile `PUT` that adds a Wanted Language previews and saves `subtitles.gained` with that language's `searches`, the files nothing meets yet, and its `needScan`, the files whose audio is scanned first; files with a text track in the language, or audio in it for a `no_audio` language, aren't counted (P-368).
- `subtitles-delete-removes` deleting a Work with its files removes the found Sidecar beside its video from the Storage Node's disk, and keeps one whose bytes changed after the survey read them (P-368).
- `subtitles-settings` `GET /api/v1/subtitles/settings` answers every setting with its default, the Timing match at 30%, a `lead` whose `kind` follows the first that holds of no Wanted Language, a Content Type no provider searches, search off and ready, and each Storage Node's `scans` left; a `PATCH` that puts `subtitles.longestGap` under the Subtitle search cadence answers 422 `gap_too_short` on that key (P-368).
- `subtitles-node-settings` `GET` and `PATCH /api/v1/subtitles/nodes/{nodeId}/settings` read and store `subtitles.mapSlots`, 1 to 8, on a Storage Node, and answer 404 `not_a_storage_node` for any other id (P-368).
- `subtitles-providers` `GET /api/v1/subtitles/providers` lists SubDL, SubSource and Gestdown in search order with each one's state; `PUT .../order` reorders them, `PATCH .../{id}` turns one off and on, and `PUT .../{id}/key` checks the key on the Subtitles holder first, so a refused key answers 422 `subtitle_key_rejected` with the provider's own words and is never stored (P-368).
- `subtitles-work` `GET /api/v1/works/{workId}/subtitles` answers each file's wants with their state, its embedded tracks and its companions; `PUT .../files/{fileId}/subtitles/tracks/{stream}/ignored` stops a text track from counting, so its want reads missing, and `DELETE` counts it again (P-368).
- `subtitles-health` Health's Subtitles category, after Indexers, raises no holder, a rejected key, a failing provider, a Content Type with Wanted Languages that no provider searches, and a Node that can't scan audio, and clears each when its rows do (P-368).
- `subtitles-actions-wait` search, Manual search, pick, upload, Mark bad and keep answer 501 `subtitle_action_not_built` until Subtitle search lands, except that an upload over 5 MB answers 422 `subtitle_too_large` and a body over 7 MiB answers 413 (P-368).
- `subtitles-library-mark` each Library row's `subtitles` reads `missing` or `none_passed` for a Work with an unmet want, and null otherwise (P-368).

## How to get to it (user POV)

- Desktop shell: none yet. The Profile editor gains Wanted Languages in a later PR.
- Phone shell: none yet.
- API (Admin): `PUT /api/v1/profiles/{profileId}` with `languages`, `?preview=true` for the Impact alone and `?confirmSubtitles=<n>` on the save; `POST /api/v1/works/delete` with `{"workIds":[...],"files":"delete"}` for the delete; `POST /api/v1/roots/{rootId}/scan` for a Library scan.
- API (Admin): `/api/v1/subtitles/settings`, `/api/v1/subtitles/nodes/{nodeId}/settings`, `/api/v1/subtitles/providers` with `/order`, `/{id}` and `/{id}/key`, `/api/v1/works/{workId}/subtitles`, and `/api/v1/files/{fileId}/subtitles/...` for tracks and the manual actions. `GET /api/v1/system/health` for the Subtitles category, and `GET /api/v1/library` for the row mark.
- Host CLI: none.

## Driving it with fakeworld and curl

Preconditions:

- A fresh `home` run, doctor passing. Its Movies root (`movies`) starts empty; the films sit in `shelf-movies`, which no root names.
- Copper Kettle copied into the Movies root, with two companions beside it, before the scan. A Scenario folder holds `.mkv` files only, so each `.srt` goes on disk with `fakeworld exec -- cp` from a local file, and the braces of a `{tmdb-...}` folder are doubled there.

  ```sh
  $FW fs cp $RUN 'shelf-movies/Copper Kettle (2012).mkv' 'movies/Copper Kettle (2012) {tmdb-9000105}/Copper Kettle (2012).mkv'
  printf '1\n00:00:01,000 --> 00:00:02,000\nHello\n' > "$SCRATCH/en.srt"
  printf '1\n00:00:01,000 --> 00:00:02,000\n[kettle whistles] Hello\n' > "$SCRATCH/en.hi.srt"
  for t in en en.hi; do
    $FW exec $RUN -- cp "$SCRATCH/$t.srt" "{mediaDir}/movies/Copper Kettle (2012) {{tmdb-9000105}/Copper Kettle (2012).$t.srt"
  done
  ROOT=$($FW exec $RUN -- curl -sS -H 'X-Api-Key: {apiKey}' '{nodes.home.url}/api/v1/roots' | jq -r '.roots[] | select(.path | endswith("/movies")) | .id')
  $FW exec $RUN -- curl -sS -X POST -H 'X-Api-Key: {apiKey}' "{nodes.home.url}/api/v1/roots/$ROOT/scan"
  ```

Steps:

- **Gain a language.** Put French, full, `no_audio`, on `Movies` as a preview, then as a save: `subtitles.gained[0]` reads `fr` with `searches` counting the films without French audio or a French text track, and `needScan` the same count, since no Speech Map is built yet. A JSON body goes in as `-d @file`: `fakeworld exec` reads single braces as placeholders.

  ```sh
  MOVIES=$($FW exec $RUN -- curl -sS -H 'X-Api-Key: {apiKey}' '{nodes.home.url}/api/v1/profiles' | jq -c '.profiles[] | select(.name == "Movies")')
  ID=$(echo "$MOVIES" | jq -r .id)
  FR='{"language":"fr","forced":false,"hearingImpaired":"allow","audio":"no_audio"}'
  echo "$MOVIES" | jq -c "{name, contentType, targets, editions, languages: [$FR]}" > "$SCRATCH/fr.json"
  $FW exec $RUN --proof subtitles/subtitles-wants -- curl -sS -X PUT -H 'X-Api-Key: {apiKey}' -H 'Content-Type: application/json' -d @"$SCRATCH/fr.json" "{nodes.home.url}/api/v1/profiles/$ID?preview=true"
  $FW exec $RUN --proof subtitles/subtitles-wants -- curl -sS -X PUT -H 'X-Api-Key: {apiKey}' -H 'Content-Type: application/json' -d @"$SCRATCH/fr.json" "{nodes.home.url}/api/v1/profiles/$ID?confirm=0&confirmSubtitles=0"
  ```

- **Adopt English.** The scan surveyed Copper Kettle while nothing wanted English, so both companions are extras. Put English, full, `always`, beside French: its preview already counts Copper Kettle as met, since a companion its last survey listed meets English. After the save, the bulk pass marks the film for a survey and the Keeper adopts `.en.srt` as found, `.en.hi.srt` when the language prefers hearing impaired. A second preview of the same body then gains nothing, and a preview without English drops `en` with `deleted` 0, since a found Sidecar stays.

- **Delete the Work.** Delete Copper Kettle with its files: the video and the adopted `.srt` both leave `{mediaDir}`, which the `find` after it shows.

  ```sh
  WORK=$($FW exec $RUN -- curl -sS -H 'X-Api-Key: {apiKey}' '{nodes.home.url}/api/v1/library' | jq -r '.rows[] | select(.label == "Copper Kettle") | .id')
  jq -nc --arg w "$WORK" '{workIds: [$w], files: "delete"}' > "$SCRATCH/delete.json"
  $FW exec $RUN --proof subtitles/subtitles-delete-removes -- curl -sS -X POST -H 'X-Api-Key: {apiKey}' -H 'Content-Type: application/json' -d @"$SCRATCH/delete.json" '{nodes.home.url}/api/v1/works/delete'
  $FW exec $RUN --proof subtitles/subtitles-delete-removes -- find '{mediaDir}/movies' -type f -not -name .slipstream-root
  ```

- **Settings and the gap.** Read the section, then put the longest gap under a 2 day cadence: the reply is 422 `gap_too_short` at `body.values.subtitles.longestGap`, reading "Use 2 days or longer. Subtitle search runs every 2 days."

  ```sh
  $FW exec $RUN --proof subtitles/subtitles-settings -- curl -sS -H 'X-Api-Key: {apiKey}' '{nodes.home.url}/api/v1/subtitles/settings'
  $FW exec $RUN --proof subtitles/subtitles-settings -- curl -sS -X PATCH -H 'X-Api-Key: {apiKey}' -H 'Content-Type: application/json' -d '{"values":{"tasks.subtitle-search.cadence":172800,"subtitles.longestGap":86400}}' '{nodes.home.url}/api/v1/subtitles/settings'
  ```

- **Node slots.** Patch `subtitles.mapSlots` to 4 on the `home` Node's id from `GET /api/v1/nodes`, read it back, then read a random id for the 404.

  ```sh
  $FW exec $RUN --proof subtitles/subtitles-node-settings -- curl -sS -X PATCH -H 'X-Api-Key: {apiKey}' -H 'Content-Type: application/json' -d '{"values":{"subtitles.mapSlots":4}}' "{nodes.home.url}/api/v1/subtitles/nodes/$NODE/settings"
  ```

- **Providers.** List them, move Gestdown first, turn it off and on, then put a key SubDL refuses: the reply quotes SubDL's words, and the list still shows `needs_key`.

  ```sh
  $FW exec $RUN --proof subtitles/subtitles-providers -- curl -sS -X PUT -H 'X-Api-Key: {apiKey}' -H 'Content-Type: application/json' -d '{"ids":["gestdown","subdl","subsource"]}' '{nodes.home.url}/api/v1/subtitles/providers/order'
  $FW exec $RUN --proof subtitles/subtitles-providers -- curl -sS -X PUT -H 'X-Api-Key: {apiKey}' -H 'Content-Type: application/json' -d '{"key":"not-a-key"}' '{nodes.home.url}/api/v1/subtitles/providers/subdl/key'
  ```

- **A Work's subtitles.** With English wanted on `Movies`, read The Quiet Orchard's subtitles, then ignore its English text track by its stream number and read again: the want reads missing and the Library row's `subtitles` reads `missing`.

  ```sh
  $FW exec $RUN --proof subtitles/subtitles-work -- curl -sS -H 'X-Api-Key: {apiKey}' "{nodes.home.url}/api/v1/works/$WORK/subtitles"
  $FW exec $RUN --proof subtitles/subtitles-library-mark -- curl -sS -H 'X-Api-Key: {apiKey}' '{nodes.home.url}/api/v1/library?contentType=movie'
  ```

- **Health.** With no provider keyed and English wanted on `Movies`, `GET /api/v1/system/health` shows `subtitles.uncovered` under Subtitles; a key the holder accepts clears it.

- **Actions wait.** `POST .../subtitles/en/search` answers 501 with "This build of SlipStream can't search for, download or upload subtitles yet."; an upload of 5 MB and 1 byte, base64 in `bytes`, answers 422 `subtitle_too_large`.

  ```sh
  $FW exec $RUN --proof subtitles/subtitles-actions-wait -- curl -sS -X POST -H 'X-Api-Key: {apiKey}' "{nodes.home.url}/api/v1/files/$FILE/subtitles/en/search"
  ```

## Gotchas

- The Keeper wakes a second after the Write that names a file and sweeps every minute, so a survey or a removal shows a moment after the API answers. Wait for it before reading the disk.
- `home`'s Movies root starts empty: The Quiet Orchard isn't in it until a step copies a film in and scans.
- A Sidecar's removal names its hash. Overwriting the `.en.srt` before the delete keeps it on disk, which is the second half of `subtitles-delete-removes`, not a failure.
- No Subtitle Provider searches yet, so a want stays missing, and the manual actions answer 501.
- A key check needs a Node with the Subtitles Role. Without one, `PUT .../key` answers 409 `no_subtitles_holder`.
