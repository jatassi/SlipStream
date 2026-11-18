# Subtitles

An Admin adds a Wanted Language to a Profile, and SlipStream records a want for every library file of that Profile that no embedded text track, no audio in the language and no Sidecar meets yet. A subtitle file found beside a library file at its name, such as `Copper Kettle (2012).en.srt`, meets its language as a found Sidecar, also when the language is wanted only after the scan. Deleting a file removes the Sidecars beside it from the Storage Node that holds it, and only bytes SlipStream recorded: a file the Admin put at that name since stays.

## Sub-features

- `subtitles-wants` a Profile `PUT` that adds a Wanted Language previews and saves `subtitles.gained` with that language's `searches`, the files nothing meets yet, and its `needScan`, the files whose audio is scanned first; files with a text track in the language, or audio in it for a `no_audio` language, aren't counted (P-368).
- `subtitles-delete-removes` deleting a Work with its files removes the found Sidecar beside its video from the Storage Node's disk, and keeps one whose bytes changed after the survey read them (P-368).

## How to get to it (user POV)

- Desktop shell: none yet. The Profile editor gains Wanted Languages in a later PR.
- Phone shell: none yet.
- API (Admin): `PUT /api/v1/profiles/{profileId}` with `languages`, `?preview=true` for the Impact alone and `?confirmSubtitles=<n>` on the save; `POST /api/v1/works/delete` with `{"workIds":[...],"files":"delete"}` for the delete; `POST /api/v1/roots/{rootId}/scan` for a Library scan.
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

## Gotchas

- The Keeper wakes a second after the Write that names a file and sweeps every minute, so a survey or a removal shows a moment after the API answers. Wait for it before reading the disk.
- `home`'s Movies root starts empty: The Quiet Orchard isn't in it until a step copies a film in and scans.
- A Sidecar's removal names its hash. Overwriting the `.en.srt` before the delete keeps it on disk, which is the second half of `subtitles-delete-removes`, not a failure.
- No Subtitle Provider searches in this PR, so a want stays missing. Its rows show only through the Profile Impact.
