# Subtitles

An Admin adds a Wanted Language to a Profile, and SlipStream records a want for every library file of that Profile that no embedded text track, no audio in the language and no Sidecar meets yet. A subtitle file found beside a library file at its name, such as `Heat (1995).en.srt`, meets its language as a found Sidecar. Deleting a file removes the Sidecars beside it from the Storage Node that holds it, and only bytes SlipStream recorded: a file the Admin put at that name since stays.

## Sub-features

- `subtitles-wants` a Profile `PUT` that adds a Wanted Language previews and saves `subtitles.gained` with that language's `searches`, the files nothing meets yet, and its `needScan`, the files whose audio is scanned first; files with a text track in the language, or audio in it for a `no_audio` language, aren't counted (P-368).
- `subtitles-delete-removes` deleting a Work with its files removes the found Sidecar beside its video from the Storage Node's disk, and keeps one whose bytes changed after the survey read them (P-368).

## How to get to it (user POV)

- Desktop shell: none yet. The Profile editor gains Wanted Languages in a later PR.
- Phone shell: none yet.
- API (Admin): `PUT /api/v1/profiles/{profileId}` with `languages`, `?preview=true` for the Impact alone and `?confirmSubtitles=<n>` on the save; `DELETE /api/v1/works/{workId}?deleteFiles=true` for the delete.
- Host CLI: none.

## Driving it with fakeworld and curl

Preconditions:

- A fresh `home` run, doctor passing, whose Movies root holds The Quiet Orchard from its first library scan.
- Before the run, a companion `The Quiet Orchard (2021).en.srt` beside the film's video in the Scenario's Movies root, so the first scan adopts it.

Steps:

- **Gain a language.** Put French, full, `no_audio`, on `Movies` as a preview, then as a save: `subtitles.gained[0]` reads `fr` with `searches` counting the films without French audio or a French text track, and `needScan` the same count, since no Speech Map is built yet.

  ```sh
  MOVIES=$($FW exec $RUN -- curl -sS -H 'X-Api-Key: {apiKey}' '{nodes.home.url}/api/v1/profiles' | jq -c '.profiles[] | select(.name == "Movies")')
  ID=$(echo "$MOVIES" | jq -r .id)
  FR='{"language":"fr","forced":false,"hearingImpaired":"allow","audio":"no_audio"}'
  BODY=$(echo "$MOVIES" | jq -c "{name, contentType, targets, editions, languages: [$FR]}")
  $FW exec $RUN --proof subtitles/subtitles-wants -- curl -sS -X PUT -H 'X-Api-Key: {apiKey}' -H 'Content-Type: application/json' -d "$BODY" "{nodes.home.url}/api/v1/profiles/$ID?preview=true"
  $FW exec $RUN --proof subtitles/subtitles-wants -- curl -sS -X PUT -H 'X-Api-Key: {apiKey}' -H 'Content-Type: application/json' -d "$BODY" "{nodes.home.url}/api/v1/profiles/$ID?confirm=0&confirmSubtitles=0"
  ```

- **Adopt English.** Put English, full, `always`, beside French. Within a minute the Keeper's sweep surveys the film and adopts the `.en.srt` as found: a second preview of the same body gains nothing, and a preview without English drops `en` with `deleted` 0, since a found Sidecar stays.

- **Delete the Work.** Delete The Quiet Orchard with its files: the video and `The Quiet Orchard (2021).en.srt` both leave `{mediaDir}`, which `fs match` proves absent.

  ```sh
  $FW exec $RUN --proof subtitles/subtitles-delete-removes -- curl -sS -X DELETE -H 'X-Api-Key: {apiKey}' "{nodes.home.url}/api/v1/works/$WORK?deleteFiles=true"
  $FW exec $RUN --proof subtitles/subtitles-delete-removes -- ls '{mediaDir}/movies/The Quiet Orchard (2021)'
  ```

## Gotchas

- The Keeper wakes a second after the Write that names a file and sweeps every minute, so a survey or a removal shows a moment after the API answers. Wait for it before reading the disk.
- A Sidecar's removal names its hash. Overwriting the `.en.srt` before the delete keeps it on disk, which is the second half of `subtitles-delete-removes`, not a failure.
- No Subtitle Provider searches in this PR, so a want stays missing. Its rows show only through the Profile Impact.
