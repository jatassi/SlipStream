# Subtitles

An Admin adds a Wanted Language to a Profile, and SlipStream records a want for every library file of that Profile that no embedded text track, no audio in the language and no Sidecar meets yet. A subtitle file found beside a library file at its name, such as `Copper Kettle (2012).en.srt`, meets its language as a found Sidecar, also when the language is wanted only after the scan. Deleting a file removes the Sidecars beside it from the Storage Node that holds it, and only bytes SlipStream recorded: a file the Admin put at that name since stays.

## Sub-features

- `subtitles-wants` a Profile `PUT` that adds a Wanted Language previews and saves `subtitles.gained` with that language's `searches`, the files nothing meets yet, and its `needScan`, the files whose audio is scanned first; files with a text track in the language, or audio in it for a `no_audio` language, aren't counted (P-368).
- `subtitles-maps` a film with an unmet want gets its audio scanned on the Storage Node that holds it, one file at a time: its Speech Map lands in that Node's `speechmaps` folder, and a film whose audio can't be decoded is asked for again later (P-368).
- `subtitles-delete-removes` deleting a Work with its files removes the found Sidecar beside its video from the Storage Node's disk, and keeps one whose bytes changed after the survey read them (P-368).
- `subtitles-settings` `GET /api/v1/subtitles/settings` answers every setting with its default, the Timing match at 30%, a `lead` whose `kind` follows the first that holds of no Wanted Language, a Content Type no provider searches, search off and ready, and each Storage Node's `scans` left; a `PATCH` that puts `subtitles.longestGap` under the Subtitle search cadence answers 422 `gap_too_short` on that key (P-368).
- `subtitles-node-settings` `GET` and `PATCH /api/v1/subtitles/nodes/{nodeId}/settings` read and store `subtitles.mapSlots`, 1 to 8, on a Storage Node, and answer 404 `not_a_storage_node` for any other id (P-368).
- `subtitles-providers` `GET /api/v1/subtitles/providers` lists SubDL, SubSource and Gestdown in search order with each one's state; `PUT .../order` reorders them, `PATCH .../{id}` turns one off and on, and `PUT .../{id}/key` checks the key on the Subtitles holder first, so a refused key answers 422 `subtitle_key_rejected` with the provider's own words and is never stored (P-368).
- `subtitles-work` `GET /api/v1/works/{workId}/subtitles` answers each file's wants with their state, its embedded tracks and its companions; `PUT .../files/{fileId}/subtitles/tracks/{stream}/ignored` stops a text track from counting, so its want reads missing, and `DELETE` counts it again (P-368).
- `subtitles-health` Health's Subtitles category, after Indexers, raises no holder, a rejected key, a failing provider, a Content Type with Wanted Languages that no provider searches, and a Node that can't scan audio, and clears each when its rows do (P-368).
- `subtitles-settings-screen` Settings > Subtitles, in the `Metadata and subtitles` group, leads with one sentence for the first `lead` that holds and its next step (`Open Profiles`, or `Add SubDL key`), lists the Profiles with subtitle languages, then `Subtitle Providers`, `Subtitle search` with the rest under `Details`, and a card per Storage Node with its `Audio scans at once on this Node`; a gap under the cadence shows the 422 under the field (P-368).
- `subtitles-key-dialog` a provider's `Add key` opens `Add your SubDL key`: numbered steps whose `Open SubDL` link ticks its step, and `SlipStream never asks for your SubDL password.`; a refused key replaces the steps with a `Not saved` card quoting SubDL's words and `Try another key`, and an accepted key with a card naming the account, plan, downloads left and place in the search order (P-368).
- `subtitles-provider-order` `Subtitle Providers` ranks SubDL, SubSource and Gestdown with each one's state in a sentence and a downloads Meter, moves a row by `Move up` and `Move down` (an action sheet on phone), and turns one off with `Search <name>` (P-368).
- `subtitles-search-task` the `subtitle-search` task, "Subtitle search", is listed once a Profile has a Wanted Language; Run now ends Succeeded with "Queued N subtitles." while its drain goes on searching, and with `subtitles.searchOn` false the run is skipped with "Subtitle search is off in Settings." while audio scans go on (P-368).
- `subtitles-search-now` `POST .../files/{fileId}/subtitles/{want}/search` answers 202 for a want the file searches and 404 `subtitle_not_searched` for a language its Profile doesn't list or a track already has (P-368).
- `subtitles-manual-search` `POST .../{want}/candidates` answers 202 with a `searchId`; `GET .../candidates/{searchId}` reads `searching`, then `done` with the ranked candidates, each one's ranking steps and earlier try, and each provider's outcome; an unknown `searchId` answers 410 `subtitle_search_expired`, and a pick on a file whose audio isn't scanned answers 409 `audio_not_scanned` (P-368).
- `subtitles-upload` an upload over 5 MB answers 422 `subtitle_too_large` and a body over 7 MiB 413; one on a file whose audio isn't scanned answers 409 `audio_not_scanned`; a failing upload's `GET .../attempts/{attemptId}` reads `failed_timing` with an `uploadId`, whose `POST .../uploads/{uploadId}/keep` writes it beside the video, and an unknown upload or attempt id answers 410 (P-368).
- `subtitles-mark-bad` `POST .../{want}/mark-bad` on an adopted `.en.srt` answers 202, deletes the file from the Storage Node's disk, records "Subtitle marked bad" in History and searches again; a second Mark bad answers 404 `subtitle_not_found` (P-368).
- `subtitles-library-mark` each Library row's `subtitles` reads `missing` or `none_passed` for a Work with an unmet want, and null otherwise (P-368).
- `subtitles-work-rows` the Work page's Files section shows, inside each file's row, one subtitle row per Wanted Language in Profile order: a language chip with its state icon, one sentence on what the subtitle is or why there is none, Search, Manual search and Upload, and a "…" menu with Mark bad… and Don't count this track; a timed subtitle's Details list its Timing sync numbers, and the file's Details list its embedded tracks and the subtitle files beside it (P-368).
- `subtitles-season-fold` a series' files fold per season, the first season with a missing subtitle open, each episode's language chips opening into the same rows; the open seasons are kept per Work in the browser (P-368).
- `subtitles-library-filter` the Library's Filter offers Subtitles, "Subtitles missing" and "None matched the timing", at the address key `subtitles=missing,none-passed`; a Work none of whose found subtitles matched counts as missing too (P-368).
- `subtitles-manual-sheets` a row's Manual search opens a 940 dialog on desktop and a full sheet on phone: the candidates ranked with the first as Best and each one's ranking steps, earlier tries folded under Tried before, and one line for each provider that found nothing or is paused, such as "SubDL is out of downloads. SlipStream searches it again tonight at 00:00."; a pick shows "Downloading from SubDL…" and "Checking timing…", then toasts "English subtitle added." and closes, or stays open with why it failed: its timing match against the threshold, or for a download the provider didn't serve, "SubDL is out of downloads. SlipStream searches it again tonight at 00:00. Pick another subtitle."; Upload is a dialog with a drop target whose passing file toasts "English subtitle added." and whose failing file offers Keep anyway under Details; Mark bad… asks a question naming the file version, or the Admin's own file for an upload, then toasts "English subtitle marked bad. SlipStream is searching for another.", also while Subtitle search is off The Work page's row reads the added, kept or removed subtitle without a reload (P-368).
- `subtitles-sidecars-land` adding English in a Profile's editor previews "Saving searches English subtitles for 3 files. SlipStream scans their audio first.", each file's Speech Map then lands in its Storage Node's `speechmaps` folder, and a row's Search lands the `.en.srt` beside the video, the row reading "From Gestdown. Subtitle search looks for a better match."; one Gestdown season pack lands each searched episode's subtitle (P-368).
- `subtitles-key-paused` once SubDL answers a download with its daily limit, `GET /api/v1/subtitles/providers` reads SubDL `out` until the next UTC midnight, also after the Controller restarts, and its Settings row reads "SubDL is out of downloads. SlipStream searches it again tonight at 00:00." (P-368).
- `subtitles-key-revoked` once a provider refuses a stored key on a search, it reads `rejected` with the provider's own words, its Settings row reads `SubSource rejected the key today at <time> and responded "<words>". Replace the key.`, and Health says to add a new key in Settings > Subtitles, with when under Details (P-368).
- `subtitles-upgrade-carry` an Import that replaces a file with a fetched Sidecar moves the Sidecar to the new file's name and times it again against the new file's Speech Map: it stays when the new file speaks as the old one did, and leaves the disk with the want missing when it doesn't (P-368).

## How to get to it (user POV)

- Desktop shell: the sidebar's `Settings`, then `Subtitles` under `Metadata and subtitles` (`/settings/subtitles`). A Profile's `Subtitles` section, under `Settings` > `Profiles` > the Profile, holds its Wanted Languages.
- Desktop shell: a Work page's Files section, each file's subtitle rows, its Details, and a row's Manual search, Upload and "…" menu; Library, Filter, Subtitles.
- Phone shell: `More`, then `Settings`, then `Subtitles`; the key dialog is a bottom sheet, and a ranked provider's `Actions for <name>` opens an action sheet.
- Phone shell: the same Work page, where a subtitle row's actions open as an action sheet and Manual search is a full sheet; Library, Filter opens as a sheet.
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

- **Scan the audio.** French stays unmet on every film without French audio, so once the Keeper has surveyed them the map drain asks `home`, which holds the Movies root, to scan each one's audio. Wait a minute, then list the Node's maps: one `.ssmp` file per film that decoded.

  ```sh
  $FW exec $RUN --proof subtitles/subtitles-maps -- find '{nodes.home.dataDir}/speechmaps' -type f
  ```

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

- **The screens.** Sign in as `{runId}-admin-desktop` and `{runId}-admin-phone`, open `/settings/subtitles`, press `Add key` on SubDL, paste `not-a-key` for the refusal, `Try another key`, then the fake's valid key for the accepted card. Move Gestdown up with its row actions, and screenshot each state under `proof/subtitles/<sub-feature>/`.

- **A Work's subtitles.** With English wanted on `Movies`, copy Lanterns at Low Tide (it has an embedded English SubRip track) into the Movies root, scan, read its subtitles, then ignore its English text track by its stream number and read again: the want reads missing and the Library row's `subtitles` reads `missing`.

  ```sh
  $FW exec $RUN --proof subtitles/subtitles-work -- curl -sS -H 'X-Api-Key: {apiKey}' "{nodes.home.url}/api/v1/works/$WORK/subtitles"
  $FW exec $RUN --proof subtitles/subtitles-library-mark -- curl -sS -H 'X-Api-Key: {apiKey}' '{nodes.home.url}/api/v1/library?contentType=movie'
  ```

- **Health.** With no provider keyed and English wanted on `Movies`, `GET /api/v1/system/health` shows `uncovered.movie` under Subtitles; a key the holder accepts clears it.

- **Subtitle search.** With English wanted on `Movies`, `GET /api/v1/tasks` lists `subtitle-search`; `POST /api/v1/tasks/subtitle-search/runs` ends Succeeded with "Queued N subtitles.". Patch `subtitles.searchOn` to false and run it again: the run reads skipped, "Subtitle search is off in Settings.".

  ```sh
  $FW exec $RUN --proof subtitles/subtitles-search-task -- curl -sS -X POST -H 'X-Api-Key: {apiKey}' '{nodes.home.url}/api/v1/tasks/subtitle-search/runs'
  $FW exec $RUN --proof subtitles/subtitles-search-task -- curl -sS -H 'X-Api-Key: {apiKey}' '{nodes.home.url}/api/v1/tasks/subtitle-search'
  ```

- **Search now.** On a film's file from `GET /api/v1/works/{workId}/subtitles`, search English (202), then a language the Profile doesn't list (404 `subtitle_not_searched`).

  ```sh
  $FW exec $RUN --proof subtitles/subtitles-search-now -- curl -sS -o /dev/null -w '%{http_code}\n' -X POST -H 'X-Api-Key: {apiKey}' "{nodes.home.url}/api/v1/files/$FILE/subtitles/en/search"
  $FW exec $RUN --proof subtitles/subtitles-search-now -- curl -sS -X POST -H 'X-Api-Key: {apiKey}' "{nodes.home.url}/api/v1/files/$FILE/subtitles/de/search"
  ```

- **Manual subtitle search.** Start one, wait a second, read it, then read a random id for the 410. Pick a candidate's `key`, as `subdl:<id>`, on a film whose audio scan failed: 409 `audio_not_scanned`.

  ```sh
  SEARCH=$($FW exec $RUN -- curl -sS -X POST -H 'X-Api-Key: {apiKey}' "{nodes.home.url}/api/v1/files/$FILE/subtitles/en/candidates" | jq -r .searchId)
  $FW exec $RUN --proof subtitles/subtitles-manual-search -- curl -sS -H 'X-Api-Key: {apiKey}' "{nodes.home.url}/api/v1/files/$FILE/subtitles/en/candidates/$SEARCH"
  ```

- **Upload.** An upload of 5 MB and 1 byte, base64 in `bytes`, answers 422 `subtitle_too_large`; a small `.srt` on a film whose audio scan failed answers 409 `audio_not_scanned`; `POST .../uploads/<random uuid>/keep` answers 410 `subtitle_upload_expired`. On a film with a `speech` audio track, an `.srt` off its speech reads `failed_timing` with an `uploadId` on the attempt, and keeping it writes it beside the video.

  ```sh
  printf '{"name":"Film.en.srt","bytes":"%s"}' "$(base64 -w0 "$SCRATCH/en.srt")" > "$SCRATCH/upload.json"
  $FW exec $RUN --proof subtitles/subtitles-upload -- curl -sS -X POST -H 'X-Api-Key: {apiKey}' -H 'Content-Type: application/json' -d @"$SCRATCH/upload.json" "{nodes.home.url}/api/v1/files/$FILE/subtitles/en/upload"
  ```

- **Mark bad.** After **Adopt English**, Mark bad Copper Kettle's English subtitle: the reply is 202, the `find` shows `.en.srt` gone, and History lists "Subtitle marked bad". Mark bad again: 404 `subtitle_not_found`.

  ```sh
  $FW exec $RUN --proof subtitles/subtitles-mark-bad -- curl -sS -o /dev/null -w '%{http_code}\n' -X POST -H 'X-Api-Key: {apiKey}' "{nodes.home.url}/api/v1/files/$FILE/subtitles/en/mark-bad"
  $FW exec $RUN --proof subtitles/subtitles-mark-bad -- find '{mediaDir}/movies' -name '*.srt'
  ```

- **The Work page.** With English wanted on `Movies` and on `TV`, copy Lanterns at Low Tide into the Movies root and The Tin Signal's three episodes into the TV root, scan both, and open each Work page at desktop and phone width. The film's row shows English as embedded; Don't count this track turns it missing. The series folds its files into Season 1 and Season 2, the first season with a missing subtitle open; its chips open into the rows. The sheets themselves need films with speech and a provider catalog: see **The sheets** below.

  ```sh
  $FW exec $RUN --proof subtitles/subtitles-work-rows -- $PW cli -s={runId}-admin-desktop goto "{nodes.home.browserUrl}/works/$WORK"
  $FW exec $RUN --proof subtitles/subtitles-work-rows -- $PW cli -s={runId}-admin-desktop screenshot --full-page --filename={runDir}/proof/subtitles/subtitles-work-rows/movie-desktop.png
  ```

- **The sheets.** On the `subtitles` Scenario (see **The subtitles Scenario**), with SubDL and SubSource keyed and `subtitles.searchOn` false so nothing fetches first, open a film's Manual search at desktop and phone width: both candidates list, Best first. Pick the failing one: the sheet stays open with its timing match. Pick the passing one: "Downloading from SubSource…", then the toast, and the sheet closes. On another film, Upload a file whose cues follow its speech: the toast. On a third, Upload one off its speech, open Details and press Keep anyway. Mark bad… the uploaded subtitle: the question names the file, and its `.srt` leaves the disk. Open Manual search on a film with a SubDL candidate, run `$FW cue $RUN subdl-daily-limit '{}'`, then pick it: the sheet reads "SubDL is out of downloads. SlipStream searches it again tonight at 00:00. Pick another subtitle.", and its attempt reads `unfetched` with `answer.until`. A new search then lists SubDL's pause among `outcomes`, one entry per provider that searches films.

  ```sh
  $FW exec $RUN --proof subtitles/subtitles-manual-sheets -- $PW cli -s={runId}-admin-desktop click 'getByRole("list", { name: "Subtitles" }).getByRole("button", { name: "Manual search" })'
  $FW cue $RUN subdl-daily-limit '{}'
  $FW exec $RUN --proof subtitles/subtitles-manual-sheets -- $PW cli -s={runId}-admin-desktop screenshot --filename={runDir}/proof/subtitles/subtitles-manual-sheets/unfetched-desktop.png
  ```

- **The Library filter.** Open `{nodes.home.browserUrl}/library?subtitles=missing`: only Works with a missing subtitle stay, and Filter shows Subtitles with both choices and their counts.

- **The subtitles Scenario.** Launch `--scenario subtitles` for the steps below and **The sheets**. Its Movies root holds Amber Signal, Brick Lane Echo and Cinder Bloom, its TV root The Lamplighters' season 1, and `shelf` a 1080p Amber Signal that speaks as the 720p one. Every file has a speech track. SubDL lists a failing subtitle for Amber and Brick and a passing one for Cinder, SubSource a passing one for Amber and Brick, and Gestdown a season pack of The Lamplighters. `$FW subtitle $RUN <id>` prints a catalog subtitle to upload: `brick-speech` passes on Brick and fails on Cinder, `cinder-speech` passes on Cinder. Key SubDL and SubSource with any non-empty key, as in **Providers**.

- **Land Sidecars.** In Movies' editor press Add language, then English, then Save: the impact counts 3 files. Save TV the same way at phone width. A minute later `{nodes.home.dataDir}/speechmaps` holds six maps. With Subtitle search on, open The Lamplighters, open an episode's chip and press Search in its row (on phone, the "…" button, then Search): the `.en.srt` lands beside the episode, from Gestdown's season pack, and each episode searched after it lands its own.

  ```sh
  $FW exec $RUN --proof subtitles/subtitles-sidecars-land -- find '{nodes.home.dataDir}/speechmaps' -type f
  $FW exec $RUN --proof subtitles/subtitles-sidecars-land -- find '{mediaDir}/tv' -name '*.srt'
  ```

- **A key out of downloads.** Spend SubDL's downloads, then search Cinder Bloom, whose one subtitle is SubDL's (Mark bad on a kept upload searches it too). The providers read SubDL `out` until the next UTC midnight, and a `$FW node restart $RUN home` keeps it.

  ```sh
  $FW cue $RUN subdl-daily-limit '{}'
  $FW exec $RUN --proof subtitles/subtitles-key-paused -- curl -sS -X POST -H 'X-Api-Key: {apiKey}' "{nodes.home.url}/api/v1/files/$FILE/subtitles/en/search"
  $FW exec $RUN --proof subtitles/subtitles-key-paused -- curl -sS -H 'X-Api-Key: {apiKey}' '{nodes.home.url}/api/v1/subtitles/providers'
  ```

- **A revoked key.** Revoke SubSource's key and search Amber Signal, whose met subtitle is SubSource's: SubSource reads `rejected` with its words, Settings > Subtitles shows them at desktop and phone width, and so does Health. A paused provider isn't searched, so revoke a provider that is ready.

  ```sh
  $FW cue $RUN subsource-revoke-key '{}'
  $FW exec $RUN --proof subtitles/subtitles-key-revoked -- curl -sS -o /dev/null -w '%{{http_code}\n' -X POST -H 'X-Api-Key: {apiKey}' "{nodes.home.url}/api/v1/files/$AMBER/subtitles/en/search"
  ```

- **An upgrade's carry.** With Amber Signal met from a pick, copy the shelf's 1080p file into a folder under `{nodes.home.dataDir}/downloads`, import that folder by hand and confirm its Import: the 1080p file replaces the 720p one, and `<new name>.en.srt` sits beside it, `met` from SubSource with a fresh timing match.

  ```sh
  $FW exec $RUN -- mkdir -p '{nodes.home.dataDir}/downloads/Amber.Signal.2020.1080p.WEB-DL.H264-FKW'
  $FW exec $RUN -- cp '{mediaDir}/shelf/Amber.Signal.2020.1080p.WEB-DL.H264-FKW.mkv' '{nodes.home.dataDir}/downloads/Amber.Signal.2020.1080p.WEB-DL.H264-FKW/'
  $FW exec $RUN --proof subtitles/subtitles-upgrade-carry -- curl -sS -X POST -H 'X-Api-Key: {apiKey}' -H 'Content-Type: application/json' -d @"$SCRATCH/import.json" '{nodes.home.url}/api/v1/imports'
  $FW exec $RUN --proof subtitles/subtitles-upgrade-carry -- curl -sS -X POST -H 'X-Api-Key: {apiKey}' -H 'Content-Type: application/json' -d @"$SCRATCH/confirm.json" '{nodes.home.url}/api/v1/imports/confirm'
  $FW exec $RUN --proof subtitles/subtitles-upgrade-carry -- find '{mediaDir}/movies/Amber Signal (2020) {{tmdb-9000201}' -type f
  ```

  `import.json` is `{"nodeId":"<home's id>","folder":"<home's dataDir>/downloads/Amber.Signal.2020.1080p.WEB-DL.H264-FKW"}` and `confirm.json` `{"importIds":["<the Import's id>"]}`, once `GET /api/v1/imports/<id>` reads `waiting`.

## Gotchas

- The Keeper wakes a second after the Write that names a file and sweeps every minute, so a survey or a removal shows a moment after the API answers. Wait for it before reading the disk.
- `home`'s Movies root starts empty: The Quiet Orchard isn't in it until a step copies a film in and scans.
- A Sidecar's removal names its hash. Overwriting the `.en.srt` before the delete keeps it on disk, which is the second half of `subtitles-delete-removes`, not a failure.
- A Scenario film whose bytes aren't real audio can't be decoded: its map reads `decode_failed` and is asked for again after `map_retry_min`, so it adds no `.ssmp` file. That is the backoff, not a failure.
- The Profile API needs the whole Profile back with `languages`: `targets` and `editions` as `GET /api/v1/profiles` lists them.
- Only the `subtitles` Scenario has files with a `speech` audio track and a `subtitle_providers` catalog. On `home` every audio scan reads `decode_failed`, nothing reaches the search queue, a pick or an upload answers 409 `audio_not_scanned`, and a Manual subtitle search lists no candidates, which is how to reach that 409.
- Subtitle search queues only wants whose audio is scanned. A new want's first search is spread over the Subtitle search cadence, so Run now right after adding a language queues few or none: search a want now with its row's Search or `POST .../{want}/search` instead.
- While Subtitle search is off, a row's Search and Mark bad still search that want at once, ahead of fresh imports, which wait until it is on again. Manual search and Upload work while it is off.
- A replacement needs an Import. Copy the file into a folder under `{nodes.home.dataDir}/downloads`, which `POST /api/v1/imports` with `nodeId` and `folder` reads, then confirm it once it reads `waiting`.
- A key check needs a Node with the Subtitles Role. Without one, `PUT .../key` answers 409 `no_subtitles_holder`.
