# Custom Formats

An Admin reads what SlipStream reads from release names and steers which Release a search grabs. Settings > Files to keep > Custom Formats lists the library's formats, pastes TRaSH Guides', Radarr's or Sonarr's JSON as written, and holds the tester, which reads pasted names, matches each to a Work, lists every format with each condition's result and gives the Score and verdict at each Target. Each Target scores each format and holds a minimum, an upgrade-until score and an increment. A search keeps every Release it saw with its outcome, the Work page lists them under Details, and a Release that lost names the step it lost at. A repack replaces the held file only from its Release group, and a release whose title or year doesn't match the Work is refused as `wrong_title`.

## Sub-features

- `formats-paste-trash` pasting TRaSH Guides' `FLUX` format, which names no app, asks `Say whether this was written for Radarr or Sonarr.`; with `Radarr` the preview reads `Read as Radarr formats.` and `Add 1 Custom Format` lists `FLUX` with `Matches a Release when its release group matches "^(FLUX)$".` and `Radarr` (P-358).
- `formats-score` the `1080p` Target's `Scores` region takes `FLUX score` 51 and `Upgrade until` 51, the `Grabs` sentence ends `and a score of 51, then stops.`, and the save has no Impact (P-358).
- `formats-grab-changes` with FLUX's WEB-DL listed beside FKW's, the first search, before the score, grabs FKW's (12 seeders) and lists FLUX's as outranked at seeders, `3` against `12`; after the score, the next search grabs FLUX's at Score 51 and refuses FKW's held release with `not_an_upgrade` (P-071, P-072, P-128).
- `formats-releases-list` the Work page's search lists its Releases under `Details`: the grabbed one first with `Grabbed for 1080p.`, each in words such as `WEBDL-1080p · DD+ 5.1 · FLUX` with the raw title under it, FLUX's outranked one with `It has 3 seeders. The grabbed Release has 12 seeders.`, and each one's `Open in tester` opens the `Tester` dialog over the Work page with its title, judged against the Work (P-071).
- `formats-wrong-title` an untagged Bluray whose name gives 1987 is refused as `wrong_title` with the value `The Quiet Orchard (1987)`, though it ranks first, and the search grabs the WEB-DL (P-129).
- `formats-repack` with FKW's WEB-DL held at both Targets, a search grabs FKW's REPACK, refuses NTb's REPACK with `repack_other_group` at each Target, and the REPACK replaces the held file at the same path (P-072).
- `formats-freeleech` with a freeleech WEB-DL of 2 seeders listed beside FKW's of 12, the search grabs the freeleech one and FKW's lost at freeleech, `1` against `0` (P-071).
- `formats-tester` the tester reads `The.Quiet.Orchard.2021.1080p.WEB-DL.DDP5.1.Atmos.DV.HDR10.x265-FLUX` as `Reads as WEBDL-1080p from FLUX. It matches The Quiet Orchard and would fill 1080p and HD.`, with HDR `DV` and `HDR10`, `FLUX` matched by its one condition, a Score of 51 at `1080p` and 0 at `HD`, and reads the 1987 Bluray as matching no Work (P-262, P-130).

## How to get to it (user POV)

- Desktop shell: the sidebar's `Settings`, then `Custom Formats` under Files to keep, `/settings/custom-formats`. `Paste` opens `Paste Custom Formats`, `New Custom Format` opens `/settings/custom-formats/new`, and a format's name opens `/settings/custom-formats/<id>`. The `Tester` region sits under the section's intro, above `New Custom Format` and `Paste`. A format editor's `Try it` and a Release's `Open in tester` open the same tester as the `Tester` dialog. The Target editor's `Scores` region and `Propers and repacks` choice are on `/settings/targets/<id>`.
- Phone shell: `More`, then `Settings`, then `Custom Formats`; the editor and the paste dialog are pushed pages.
- The Work page: the `Searches` list, where each Item's latest search lists its `Releases` under its `Details`.
- API (Admin): `GET`, `POST /api/v1/custom-formats`, `GET /api/v1/custom-formats/schema`, `GET`, `PUT`, `DELETE /api/v1/custom-formats/{formatId}`, `POST /api/v1/custom-formats/{formatId}/duplicate`, `POST /api/v1/custom-formats/paste`, `POST /api/v1/releases/parse`, `PUT /api/v1/targets/{targetId}` with `scores` and `propers`, `GET /api/v1/searches/{searchId}/releases`.

## Driving it with fakeworld, playwright cli and curl

Preconditions:

- A fresh `targets` run with doctor passing and the `{runId}-admin-desktop` session. Its movie, The Quiet Orchard, wants `1080p` and `HD` through `Films`. fixture-site lists FKW's 1080p WEB-DL (12 seeders) and holds back, until `fixture-site-list` lists each, FLUX's WEB-DL (3 seeders), CMRG's freeleech WEB-DL (2 seeders), FKW's and NTb's REPACKs, and an untagged 1987 Bluray (40 seeders).
- `formats-wrong-title` and `formats-freeleech` each need a fresh `targets` run of their own, since each lists its release before the Work's first search.

Steps:

- **Keep Films to `1080p` alone.** One Target keeps the walk to one file.

  ```sh
  FILMS=$($FW exec $RUN -- curl -sS -H 'X-Api-Key: {apiKey}' '{nodes.home.url}/api/v1/profiles' | jq -r '.profiles[] | select(.name == "Films") | .id')
  STARTER=$($FW exec $RUN -- curl -sS -H 'X-Api-Key: {apiKey}' '{nodes.home.url}/api/v1/targets' | jq -r '.targets[] | select(.name == "1080p") | .id')
  $FW exec $RUN -- curl -sS -X PUT -H 'X-Api-Key: {apiKey}' -H 'Content-Type: application/json' -d "{\"name\":\"Films\",\"contentType\":\"movie\",\"targets\":[{\"targetId\":\"$STARTER\",\"sourcing\":\"download\"}],\"editions\":{\"kind\":\"any\"}}" "{nodes.home.url}/api/v1/profiles/$FILMS"
  ```

- **Paste TRaSH's FLUX format.** Open Custom Formats, choose `Paste`, paste the pinned TRaSH file and choose `Preview`. The dialog asks which app, and `Radarr` reads `Read as Radarr formats.` Choose `Add 1 Custom Format`. The list's `FLUX` row reads `Radarr` and `None`.

  ```sh
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop goto '{nodes.home.browserUrl}/settings/custom-formats'
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop click 'getByRole("button", { name: "Paste", exact: true })'
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop fill 'getByRole("dialog", { name: "Paste Custom Formats" }).getByRole("textbox", { name: "JSON" })' "$(cat internal/formats/testdata/trash/radarr/cf/flux.json)"
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop click 'getByRole("dialog", { name: "Paste Custom Formats" }).getByRole("button", { name: "Preview" })'
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop click 'getByRole("dialog", { name: "Paste Custom Formats" }).getByRole("radio", { name: "Radarr" })'
  $FW exec $RUN --proof custom-formats/formats-paste-trash -- $PW cli -s={runId}-admin-desktop screenshot --filename={runDir}/proof/custom-formats/formats-paste-trash/preview.png
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop click 'getByRole("dialog", { name: "Paste Custom Formats" }).getByRole("button", { name: "Add 1 Custom Format" })'
  $FW exec $RUN --proof custom-formats/formats-paste-trash -- $PW cli -s={runId}-admin-desktop snapshot --filename={runDir}/proof/custom-formats/formats-paste-trash/list.aria.yml
  $FW exec $RUN --proof custom-formats/formats-paste-trash -- curl -sS -H 'X-Api-Key: {apiKey}' '{nodes.home.url}/api/v1/custom-formats'
  ```

- **List FLUX's WEB-DL, add the Work and search.** Add The Quiet Orchard as in [add-a-movie.md](add-a-movie.md) and choose `Search`. The search grabs FKW's WEB-DL of 2 Releases. Its Releases list FKW's as `grabbed` at Score 0 and FLUX's as `outranked` with `lostAt` `seeders`, `3` against `12`. Wait for the Grab to import before the next search.

  ```sh
  $FW cue $RUN fixture-site-list '{"release": "The.Quiet.Orchard.2021.1080p.WEB-DL.DDP5.1.H264-FLUX"}'
  W=$($FW exec $RUN -- curl -sS -X POST -H 'X-Api-Key: {apiKey}' -H 'Content-Type: application/json' -d '{"contentType":"movie","key":"9000001","preset":"all"}' '{nodes.home.url}/api/v1/works' | jq -r .work.id)
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop goto "{nodes.home.browserUrl}/works/$W"
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop click 'getByRole("main").getByRole("button", { name: "Search", exact: true })'
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop run-code 'async page => { await page.getByRole("list", { name: "Searches" }).getByText("Grabbed a Release").waitFor({ timeout: 30000 }); }'
  S1=$($FW exec $RUN -- curl -sS -H 'X-Api-Key: {apiKey}' "{nodes.home.url}/api/v1/works/$W/searches" | jq -r '.searches[0].id')
  $FW exec $RUN --proof custom-formats/formats-grab-changes -- curl -sS -H 'X-Api-Key: {apiKey}' "{nodes.home.url}/api/v1/searches/$S1/releases"
  ```

- **Read the Releases list.** Open the search's `Details`, which loads its `Releases` list. The grabbed Release comes first with its facts in words and `Grabbed for 1080p.`, and FLUX's reads `WEBDL-1080p · DD+ 5.1 · FLUX` with its raw title under it and `It has 3 seeders. The grabbed Release has 12 seeders.` FLUX's `Open in tester` opens the `Tester` dialog over the Work page, its `Release names` filled with FLUX's title and `Each name is judged against The Quiet Orchard.` under it.

  ```sh
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop click 'getByRole("list", { name: "Searches" }).getByRole("button", { name: "Details" }).first()'
  $FW exec $RUN --proof custom-formats/formats-releases-list -- $PW cli -s={runId}-admin-desktop snapshot --filename={runDir}/proof/custom-formats/formats-releases-list/releases.aria.yml
  $FW exec $RUN --proof custom-formats/formats-releases-list -- $PW cli -s={runId}-admin-desktop screenshot --filename={runDir}/proof/custom-formats/formats-releases-list/releases.png
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop click 'getByRole("list", { name: "Releases" }).getByRole("listitem").filter({ hasText: "FLUX" }).first().getByRole("button", { name: "Open in tester" })'
  $FW exec $RUN --proof custom-formats/formats-releases-list -- $PW cli -s={runId}-admin-desktop snapshot --filename={runDir}/proof/custom-formats/formats-releases-list/tester.aria.yml
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop press Escape
  ```

- **Score FLUX at `1080p`.** Open the `1080p` Target, open `Scores`' `Details`, set `FLUX score` to 51 and `Upgrade until` to 51, and save. The `Grabs` sentence ends `and a score of 51, then stops.`, and `Saved 1080p.` shows with no Impact.

  ```sh
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop goto "{nodes.home.browserUrl}/settings/targets/$STARTER"
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop click 'getByRole("region", { name: "Scores" }).getByRole("button", { name: "Details" })'
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop fill 'getByRole("region", { name: "Scores" }).getByRole("textbox", { name: "FLUX score" })' 51
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop fill 'getByRole("region", { name: "Scores" }).getByRole("textbox", { name: "Upgrade until" })' 51
  $FW exec $RUN --proof custom-formats/formats-score -- $PW cli -s={runId}-admin-desktop screenshot --filename={runDir}/proof/custom-formats/formats-score/scores.png
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop click 'getByRole("button", { name: "Save", exact: true })'
  $FW exec $RUN --proof custom-formats/formats-score -- curl -sS -H 'X-Api-Key: {apiKey}' "{nodes.home.url}/api/v1/targets/$STARTER"
  ```

- **Search again.** The search grabs FLUX's WEB-DL, which reads `grabbed` at Score 51, and refuses FKW's held release with `not_an_upgrade`. Once imported, the file at `movies/The Quiet Orchard (2021)/The Quiet Orchard (2021) - 1080p.mkv` is FLUX's payload.

  ```sh
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop goto "{nodes.home.browserUrl}/works/$W"
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop click 'getByRole("main").getByRole("button", { name: "Search", exact: true })'
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop run-code 'async page => { await page.getByRole("list", { name: "Searches" }).getByText("Grabbed a Release").first().waitFor({ timeout: 30000 }); }'
  S2=$($FW exec $RUN -- curl -sS -H 'X-Api-Key: {apiKey}' "{nodes.home.url}/api/v1/works/$W/searches" | jq -r '.searches[0].id')
  $FW exec $RUN --proof custom-formats/formats-grab-changes -- curl -sS -H 'X-Api-Key: {apiKey}' "{nodes.home.url}/api/v1/searches/$S2/releases"
  $FW exec $RUN --proof custom-formats/formats-grab-changes -- $FW fs match {runDir} '{mediaDir}/movies/The Quiet Orchard (2021)/The Quiet Orchard (2021) - 1080p.mkv' --release The.Quiet.Orchard.2021.1080p.WEB-DL.DDP5.1.H264-FLUX
  ```

- **Try names in the tester.** Paste the Dolby Vision name and the 1987 Bluray into the `Tester` region's `Release names` and choose `Test`. The first reads as WEBDL-1080p from FLUX, matches The Quiet Orchard and would fill `1080p` at Score 51. The second matches no Work.

  ```sh
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop goto '{nodes.home.browserUrl}/settings/custom-formats'
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop fill 'getByRole("region", { name: "Tester" }).getByRole("textbox", { name: "Release names" })' "$(printf '%s\n%s' The.Quiet.Orchard.2021.1080p.WEB-DL.DDP5.1.Atmos.DV.HDR10.x265-FLUX The.Quiet.Orchard.1987.1080p.BluRay.x264-FKW)"
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop click 'getByRole("region", { name: "Tester" }).getByRole("button", { name: "Test", exact: true })'
  $FW exec $RUN --proof custom-formats/formats-tester -- curl -sS -X POST -H 'X-Api-Key: {apiKey}' -H 'Content-Type: application/json' -d '{"names":["The.Quiet.Orchard.2021.1080p.WEB-DL.DDP5.1.Atmos.DV.HDR10.x265-FLUX","The.Quiet.Orchard.1987.1080p.BluRay.x264-FKW"]}' '{nodes.home.url}/api/v1/releases/parse'
  $FW exec $RUN --proof custom-formats/formats-tester -- $PW cli -s={runId}-admin-desktop screenshot --filename={runDir}/proof/custom-formats/formats-tester/section.png
  ```

- **A REPACK of the held file.** On a fresh `targets` run, add the Work and search as in [targets-and-profiles.md](targets-and-profiles.md) so FKW's WEB-DL fills both Targets, then list both REPACKs and search. FKW's REPACK is grabbed, NTb's is refused with `repack_other_group` at each Target, and the REPACK replaces the file.

  ```sh
  $FW cue $RUN fixture-site-list '{"release": "The.Quiet.Orchard.2021.REPACK.1080p.WEB-DL.H264-FKW"}'
  $FW cue $RUN fixture-site-list '{"release": "The.Quiet.Orchard.2021.REPACK.1080p.WEB-DL.H264-NTb"}'
  $FW exec $RUN -- curl -sS -X POST -H 'X-Api-Key: {apiKey}' "{nodes.home.url}/api/v1/works/$W/search"
  S3=$($FW exec $RUN -- curl -sS -H 'X-Api-Key: {apiKey}' "{nodes.home.url}/api/v1/works/$W/searches" | jq -r '.searches[0].id')
  $FW exec $RUN --proof custom-formats/formats-repack -- curl -sS -H 'X-Api-Key: {apiKey}' "{nodes.home.url}/api/v1/searches/$S3/releases"
  $FW exec $RUN --proof custom-formats/formats-repack -- $FW fs match {runDir} '{mediaDir}/movies/The Quiet Orchard (2021)/The Quiet Orchard (2021) - 1080p.mkv' --release The.Quiet.Orchard.2021.REPACK.1080p.WEB-DL.H264-FKW
  ```

- **Another year, and freeleech.** On a fresh `targets` run, list the 1987 Bluray, add the Work and search: the Bluray reads `refused` with `wrong_title` `The Quiet Orchard (1987)`. On another, list `The.Quiet.Orchard.2021.1080p.WEB-DL.H264-CMRG` instead: it reads `grabbed`, and FKW's `outranked` with `lostAt` `freeleech`, `1` against `0`. Read each run's Releases as above, with `--proof custom-formats/formats-wrong-title` and `--proof custom-formats/formats-freeleech`.

  ```sh
  $FW cue $RUN fixture-site-list '{"release": "The.Quiet.Orchard.1987.1080p.BluRay.x264-FKW"}'
  ```

## Gotchas

- `GET /api/v1/searches/{searchId}/releases` lists a run's Releases only while it is its Item's latest finished run. Read it before the next search finishes, which empties it.
- The list orders ranked Releases first. A Release refused only softly, such as `wrong_title`, still ranks, so the 1987 Bluray comes before the grabbed WEB-DL with `rank` 1.
- TRaSH's group formats name neither app. A paste over the API without `"app": "radarr"` answers 422 `format_app_needed`.
- `GET /api/v1/custom-formats` answers `matchLimitMilliseconds` beside `formats`, so a reader that maps every member to a list fails on it.
- The untagged Bluray lists no `tmdbid`, so only its name says which film it is. A tagged release whose id matches the Work settles the match and is never refused as `wrong_title`.
- The tester reads with no size, so every Target's verdict carries the note `size_not_checked`.
- `e2e/formats_test.go` drives the same walks with the API alone and runs in CI.
