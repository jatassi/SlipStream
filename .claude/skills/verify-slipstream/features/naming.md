# Naming

An Admin sets how SlipStream names each Content Type's files and folders, and renames the library when they choose. Settings > Naming holds one card per enabled Content Type. Each card leads with what to do and its one button, or says that Plex, Jellyfin and Emby show each Work's files as one item. Its `Details` holds the rename switch, each folder and file template with its live preview of full paths, the colon replacement and, for TV, the multi-episode style. A template field suggests tokens on `{`, and `Tokens` opens a builder. The save check refuses a template that would split a Work's files, in words under the field. Saving a change that renames files first shows the rename plan, old and new paths side by side, with `Save and rename` and `Save only`. The section's header holds `Review all renames` and `Test file names`, a Work's `More actions` menu holds `Rename files…`, and an Impact lists the files it renames under `Details`. A rename moves each file's companions and removes the folders it empties.

## Sub-features

- `naming-cards` Settings > Naming lists `Movies` and `TV` as cards, under `Review all renames` and `Test file names`. On `home` with its shelves scanned, `Movies` leads with `3 movie files don't match the templates.` and `TV` with `4 series files don't match the templates.`, each with `Review renames`. Each type's row in Settings > Content Types ends `The names of <type> files and folders are set in Naming.`, linking to `/settings/naming#<type>` (P-091).
- `naming-template` the Movies card's `Details` holds `Rename files` on, `Movie folder` and `Movie file` with their recommended templates and a list of samples under each, such as `Heat with a cut` reading `Heat (1995)/Heat (1995) - {edition-Director's Cut} 4K HDR.mkv` and `A colon with a Standard file` reading `Star Wars - Episode IV - A New Hope (1977)/…`. A file template without `{Target}` re-renders the samples as typed and reads `The file name has no {Target}. Add {Target} so each file of a Work gets its own name.` under the field; `Save` puts `Movie file: ` and that sentence under the lead, and `PATCH /api/v1/naming/movie` answers 422 `naming_template_refused` with it. `Use recommended` puts the template back and the `Save` goes (P-092, P-098).
- `naming-modes` Colon replacement offers `Delete`, `Dash (-)`, `Space dash ( -)`, `Space dash space ( - )`, `Smart` and `Custom`. Custom shows `Custom colon replacement`: ` ?` saves to `Custom colon replacement: The colon text can't contain '?' because a name on Windows can't hold it. Remove it.`, and ` ~` renders `Star Wars ~ Episode IV - A New Hope (1977)`. The TV card's `Multi-episode style` offers `Extend`, `Duplicate`, `Repeat`, `Scene`, `Range` and `Prefixed range`; its `Two episodes with a Standard file` sample reads `S01E01-E02` and, with `Repeat`, `S01E01E02`, and `A gap` reads `S01E01E03` under both. `Discard` drops the draft (P-094, P-095).
- `naming-tokens` `Tokens for Movie file` opens `Movie file tokens`, grouped `Movie`, `Target and cut`, `Release` and `Media`. `Quality Full` shows its sentence and the controls `Text around it`, `Case`, `Between words`, `Longest, in bytes` and `Keep the end`; `Square brackets` makes the `Token text` `{[Quality Full]}` and `Insert` puts it at the caret. The preview renders `{[Custom Formats]}` for `Heat with every release fact` as `[IMAX HDR10+]` (P-096, P-097, P-073).
- `naming-save-and-rename` saving `{Movie Title} {(Year)} - {Edition Tag} {Target} {[Quality Full]}` opens `Save movie naming`, `Saving renames 3 files in 3 Works.`, each Work ticked with its files' old and new paths marked, Saltmarsh Radio's with `Moves with it: Saltmarsh Radio (2018)/Saltmarsh Radio (2018).en.srt`. `Save and rename 3 files` renames each on disk, the subtitle with its video, Copper Kettle into its own folder, and removes the folder `Lanterns at Low Tide (2016) {tmdb-9000101}`; the card then reads `Plex, Jellyfin and Emby show the files of each movie as one item.` (P-099).
- `naming-save-only` the TV card's `Episode file` without `{Episode Title}` opens `Save series naming`, `Saving renames 4 files in 2 Works.`; `Save only` stores the template, no file moves, and the card still reads `4 series files don't match the templates.` (P-093).
- `naming-review-all` `Review all renames` opens `Rename files`, `This renames 4 files in 2 Works.`; unticking The Tin Signal reads `This renames 1 file in 1 Work.` and `0 of 3 files`, and `Rename 1 file` moves Brass Meridian's `2x05` into `Season 02/Brass Meridian - S02E05 - 1080p.mkv` alone (P-099).
- `naming-tester` `Test file names` opens a sheet with `File paths`, one per line, up to 100. `Test` lists each name's result: The Tin Signal's S02E06 `It matches The Tin Signal and would be imported as:` its path in the Work's folder on disk, with `What it reads as` under `Details`; S07E01 `It matches The Tin Signal. This series has no episode S07E01.`; and an unknown film `It matches no Work. A library scan would send it to Pending imports.` (P-098).
- `naming-work-rename` The Tin Signal's `More actions for The Tin Signal` lists `Edit…`, `Refresh metadata`, `Rename files…` and `Delete…`. `Rename files…` opens `Rename files of The Tin Signal`, `This renames 3 files.`, with its files open, and `Rename 3 files` moves them from `The Tin Signal [tvdbid-9100201]` to `The Tin Signal (2019)` and removes the old folder (P-099).
- `naming-air-date` `Edit The Tin Signal` holds `Name by air date`, `Default Off.`, which saves when switched and then reads `Files keep their names and folders until you rename them.` with `Rename files`. Its plan names each episode by its air date, such as `The Tin Signal - 2019-01-07 - Static 1080p.mkv`, and confirming renames them on disk. TV's row in Content Types holds its add default (P-093).
- `naming-impact` renaming the `1080p` Target to `Full HD` opens `Save 1080p` with `5 Works change.` and `7 files are renamed.`, and its `Details` lists `Renamed files` by Work with each old and new path. `Save` renames all seven on disk, the subtitle with its video (P-099).
- `naming-switch` turning the TV card's `Rename files` off and saving makes it lead with `Turn on renaming so Jellyfin and Emby show the files of each series as one item.` and `Turn on renaming`, which turns it back on (P-091).

## How to get to it (user POV)

- Desktop shell: the sidebar's `Settings`, then `Naming` in the Library group, `/settings/naming`. Each card is an `article` named for its type, and `/settings/naming#<type>` scrolls to it. A Work's `More actions for <title>` menu holds `Rename files…`, and its `Edit…` dialog holds the type's options, such as `Name by air date`, with the link to the Work's plan once one saves. A Target, Profile or cut change's Impact lists its renames under `Details`.
- Phone shell: `More`, then `Settings`, then `Naming`; the plan is a pushed page.
- API (Admin): `GET /api/v1/naming`, `PATCH /api/v1/naming/{type}` with `{"values": {...}}`, `GET /api/v1/naming/{type}/tokens`, `POST /api/v1/naming/{type}/preview` with `{"draft": {...}}`, `POST /api/v1/renames/plan` with a `scope` of `work`, `type` or `library` and an optional `draft`, `POST /api/v1/renames/plan/{workId}`, `POST /api/v1/renames` with each Work's digest, `GET /api/v1/renames/unmatched`, `PUT /api/v1/works/{id}/options` with `{"type":{"nameByAirDate":true}}`. Live: `naming.changed`, `files.moved`.

## Driving it with fakeworld, playwright cli and curl

Preconditions:

- A fresh `home` run with doctor passing and `{runId}-admin-desktop` signed in. `S='-s={runId}-admin-desktop'`. Run the steps in order: each builds on the files the one before renamed.
- The two series added before their root, the shelves added as roots, and a subtitle beside Saltmarsh Radio's video. Each root's first Ready report scans it within seconds; the card's count says when.

  ```sh
  for key in 9100201 9100202; do $FW exec $RUN -- curl -sS -X POST -H 'X-Api-Key: {apiKey}' -H 'Content-Type: application/json' --data-binary "{\"contentType\":\"series\",\"key\":\"$key\"}" '{nodes.home.url}/api/v1/works'; done
  $FW exec $RUN -- curl -sS -X POST -H 'X-Api-Key: {apiKey}' -H 'Content-Type: application/json' --data-binary '{"contentType":"series","nodeId":"{nodes.home.nodeId}","path":"{mediaDir}/shelf-series"}' '{nodes.home.url}/api/v1/roots'
  $FW exec $RUN -- curl -sS -X POST -H 'X-Api-Key: {apiKey}' -H 'Content-Type: application/json' --data-binary '{"contentType":"movie","nodeId":"{nodes.home.nodeId}","path":"{mediaDir}/shelf-movies"}' '{nodes.home.url}/api/v1/roots'
  $FW exec $RUN -- touch '{mediaDir}/shelf-movies/Saltmarsh Radio (2018)/Saltmarsh Radio (2018).en.srt'
  TIN=$($FW exec $RUN -- curl -sS -H 'X-Api-Key: {apiKey}' '{nodes.home.url}/api/v1/library' | jq -r '.rows[] | select(.label == "The Tin Signal") | .id')
  M='getByRole("article", { name: "Movies" })'
  T='getByRole("article", { name: "TV" })'
  ```

Steps:

- **The cards.** Open Naming once both shelves are scanned. Each card leads with its count and `Review renames`.

  ```sh
  $FW exec $RUN -- $PW cli $S goto '{nodes.home.browserUrl}/settings/naming'
  $FW exec $RUN -- $PW cli $S run-code 'async page => { await page.getByRole("article", { name: "TV" }).getByText("4 series files don'"'"'t match the templates.").waitFor({ timeout: 60000 }); }'
  $FW exec $RUN --proof naming/naming-cards -- $PW cli $S snapshot 'getByRole("main")' --filename={runDir}/proof/naming/naming-cards/section.aria.yml
  $FW exec $RUN --proof naming/naming-cards -- $PW cli $S screenshot --full-page --filename={runDir}/proof/naming/naming-cards/section.png
  $FW exec $RUN --proof naming/naming-cards -- curl -sS -H 'X-Api-Key: {apiKey}' '{nodes.home.url}/api/v1/naming'
  ```

- **A refused template.** Open the Movies card's `Details`, drop `{Target}` from `Movie file`, read the check under it, then `Save`. Prove the refusal over the API, then `Use recommended`.

  ```sh
  $FW exec $RUN -- $PW cli $S click "$M"'.getByRole("button", { name: "Details" })'
  $FW exec $RUN --proof naming/naming-template -- $PW cli $S snapshot "$M" --filename={runDir}/proof/naming/naming-template/details.aria.yml
  $FW exec $RUN -- $PW cli $S fill "$M"'.getByRole("combobox", { name: "Movie file" })' '{{Movie Title} {{(Year)} - {{Edition Tag}'
  $FW exec $RUN -- $PW cli $S press Escape
  $FW exec $RUN -- $PW cli $S run-code 'async page => { await page.getByRole("article", { name: "Movies" }).getByRole("alert").first().waitFor(); }'
  $FW exec $RUN --proof naming/naming-template -- $PW cli $S screenshot --filename={runDir}/proof/naming/naming-template/refused-draft.png
  $FW exec $RUN -- $PW cli $S click "$M"'.getByRole("button", { name: "Save", exact: true })'
  $FW exec $RUN -- $PW cli $S run-code 'async page => { await page.getByRole("article", { name: "Movies" }).getByText("Movie file: The file name has no {Target}.", { exact: false }).waitFor(); }'
  $FW exec $RUN --proof naming/naming-template -- $PW cli $S snapshot "$M" --filename={runDir}/proof/naming/naming-template/refused-save.aria.yml
  $FW exec $RUN --proof naming/naming-template -- curl -sS -X PATCH -H 'X-Api-Key: {apiKey}' -H 'Content-Type: application/json' -d '{{"values":{{"movie.naming.file.standard":"{{Movie Title} {{(Year)} - {{Edition Tag}"}}' '{nodes.home.url}/api/v1/naming/movie'
  $FW exec $RUN -- $PW cli $S click "$M"'.getByRole("button", { name: "Use recommended" })'
  ```

- **Colon and multi-episode modes.** On Movies pick `Custom`, try ` ?` and `Save`, then ` ~` and read `A colon with a Standard file`, then `Discard`. On TV open `Details`, pick `Repeat` and read the episode samples, then `Discard`.

  ```sh
  $FW exec $RUN -- $PW cli $S click "$M"'.getByRole("combobox", { name: "Colon replacement" })'
  $FW exec $RUN -- $PW cli $S click 'getByRole("option", { name: "Custom" })'
  $FW exec $RUN -- $PW cli $S fill "$M"'.getByRole("textbox", { name: "Custom colon replacement" })' ' ?'
  $FW exec $RUN -- $PW cli $S click "$M"'.getByRole("button", { name: "Save", exact: true })'
  $FW exec $RUN -- $PW cli $S run-code 'async page => { await page.getByRole("article", { name: "Movies" }).getByText("Custom colon replacement: The colon text", { exact: false }).waitFor(); }'
  $FW exec $RUN --proof naming/naming-modes -- $PW cli $S screenshot --filename={runDir}/proof/naming/naming-modes/colon-refused.png
  $FW exec $RUN -- $PW cli $S fill "$M"'.getByRole("textbox", { name: "Custom colon replacement" })' ' ~'
  $FW exec $RUN -- $PW cli $S run-code 'async page => { await page.getByRole("list", { name: "Movie file samples" }).getByText("Star Wars ~ Episode IV - A New Hope (1977)/", { exact: false }).first().waitFor(); }'
  $FW exec $RUN --proof naming/naming-modes -- $PW cli $S snapshot "$M"'.getByRole("list", { name: "Movie file samples" })' --filename={runDir}/proof/naming/naming-modes/colon-custom.aria.yml
  $FW exec $RUN -- $PW cli $S click "$M"'.getByRole("button", { name: "Discard" })'
  $FW exec $RUN -- $PW cli $S click "$T"'.getByRole("button", { name: "Details" })'
  $FW exec $RUN -- $PW cli $S click "$T"'.getByRole("combobox", { name: "Multi-episode style" })'
  $FW exec $RUN -- $PW cli $S click 'getByRole("option", { name: "Repeat" })'
  $FW exec $RUN --proof naming/naming-modes -- $PW cli $S snapshot "$T"'.getByRole("list", { name: "Episode file samples" })' --filename={runDir}/proof/naming/naming-modes/repeat.aria.yml
  $FW exec $RUN -- $PW cli $S click "$T"'.getByRole("button", { name: "Discard" })'
  ```

- **The token builder.** End `Movie file` with a space, open its `Tokens`, pick `Quality Full` with `Square brackets` and `Insert`. Read the Custom Formats token's render from the preview API.

  ```sh
  $FW exec $RUN -- $PW cli $S fill "$M"'.getByRole("combobox", { name: "Movie file" })' '{{Movie Title} {{(Year)} - {{Edition Tag} {{Target} '
  $FW exec $RUN -- $PW cli $S press Escape
  $FW exec $RUN -- $PW cli $S click "$M"'.getByRole("button", { name: "Tokens for Movie file" })'
  $FW exec $RUN -- $PW cli $S click 'getByRole("dialog", { name: "Movie file tokens" }).getByRole("button", { name: "Quality Full" })'
  $FW exec $RUN -- $PW cli $S click 'getByRole("dialog", { name: "Movie file tokens" }).getByRole("combobox", { name: "Text around it" })'
  $FW exec $RUN -- $PW cli $S click 'getByRole("option", { name: "Square brackets" })'
  $FW exec $RUN --proof naming/naming-tokens -- $PW cli $S snapshot 'getByRole("dialog", { name: "Movie file tokens" })' --filename={runDir}/proof/naming/naming-tokens/builder.aria.yml
  $FW exec $RUN --proof naming/naming-tokens -- $PW cli $S screenshot --filename={runDir}/proof/naming/naming-tokens/builder.png
  $FW exec $RUN -- $PW cli $S click 'getByRole("dialog", { name: "Movie file tokens" }).getByRole("button", { name: "Insert" })'
  $FW exec $RUN --proof naming/naming-tokens -- curl -sS -X POST -H 'X-Api-Key: {apiKey}' -H 'Content-Type: application/json' -d '{{"draft":{{"movie.naming.file.standard":"{{Movie Title} {{(Year)} - {{Edition Tag} {{Target} {{[Custom Formats]}"}}' '{nodes.home.url}/api/v1/naming/movie/preview'
  ```

- **Save and rename.** `Save` opens the draft's plan. Open Saltmarsh Radio's and Copper Kettle's files, then `Save and rename 3 files`. The card's lead turns, and the shelf holds the new names.

  ```sh
  D='getByRole("dialog", { name: "Save movie naming" })'
  $FW exec $RUN -- $PW cli $S click "$M"'.getByRole("button", { name: "Save", exact: true })'
  $FW exec $RUN -- $PW cli $S click "$D"'.getByRole("button", { name: "Files of Saltmarsh Radio" })'
  $FW exec $RUN -- $PW cli $S click "$D"'.getByRole("button", { name: "Files of Copper Kettle" })'
  $FW exec $RUN --proof naming/naming-save-and-rename -- $PW cli $S snapshot "$D" --filename={runDir}/proof/naming/naming-save-and-rename/plan.aria.yml
  $FW exec $RUN --proof naming/naming-save-and-rename -- $PW cli $S screenshot --filename={runDir}/proof/naming/naming-save-and-rename/plan.png
  $FW exec $RUN -- $PW cli $S click "$D"'.getByRole("button", { name: "Save and rename 3 files" })'
  $FW exec $RUN -- $PW cli $S run-code 'async page => { await page.getByRole("article", { name: "Movies" }).getByText("Plex, Jellyfin and Emby show the files of each movie as one item.").waitFor({ timeout: 30000 }); }'
  $FW exec $RUN --proof naming/naming-save-and-rename -- $PW cli $S screenshot --filename={runDir}/proof/naming/naming-save-and-rename/after.png
  $FW exec $RUN --proof naming/naming-save-and-rename -- find '{mediaDir}/shelf-movies'
  ```

- **Save only.** On TV, drop `{Episode Title}` from `Episode file` and `Save`; choose `Save only`. Nothing moves and the count stays.

  ```sh
  $FW exec $RUN -- $PW cli $S fill "$T"'.getByRole("combobox", { name: "Episode file", exact: true })' '{{Series Title} - S{{season:00}E{{episode:00} - {{Edition} {{Target}'
  $FW exec $RUN -- $PW cli $S press Escape
  $FW exec $RUN -- $PW cli $S click "$T"'.getByRole("button", { name: "Save", exact: true })'
  $FW exec $RUN --proof naming/naming-save-only -- $PW cli $S snapshot 'getByRole("dialog", { name: "Save series naming" })' --filename={runDir}/proof/naming/naming-save-only/plan.aria.yml
  $FW exec $RUN -- $PW cli $S click 'getByRole("dialog", { name: "Save series naming" }).getByRole("button", { name: "Save only" })'
  $FW exec $RUN -- $PW cli $S run-code 'async page => { await page.getByRole("dialog").waitFor({ state: "detached" }); }'
  $FW exec $RUN --proof naming/naming-save-only -- $PW cli $S snapshot "$T" --filename={runDir}/proof/naming/naming-save-only/after.aria.yml
  $FW exec $RUN --proof naming/naming-save-only -- find '{mediaDir}/shelf-series' -type f
  ```

- **Review all renames.** Open it, open Brass Meridian's files, untick The Tin Signal and rename the one file left.

  ```sh
  R='getByRole("dialog", { name: "Rename files" })'
  $FW exec $RUN -- $PW cli $S click 'getByRole("button", { name: "Review all renames" })'
  $FW exec $RUN -- $PW cli $S click "$R"'.getByRole("button", { name: "Files of Brass Meridian" })'
  $FW exec $RUN -- $PW cli $S click "$R"'.getByRole("checkbox", { name: "The Tin Signal" })'
  $FW exec $RUN --proof naming/naming-review-all -- $PW cli $S snapshot "$R" --filename={runDir}/proof/naming/naming-review-all/unticked.aria.yml
  $FW exec $RUN --proof naming/naming-review-all -- $PW cli $S screenshot --filename={runDir}/proof/naming/naming-review-all/unticked.png
  $FW exec $RUN -- $PW cli $S click "$R"'.getByRole("button", { name: "Rename 1 file" })'
  $FW exec $RUN -- $PW cli $S run-code 'async page => { await page.getByRole("article", { name: "TV" }).getByText("3 series files don'"'"'t match the templates.").waitFor({ timeout: 30000 }); }'
  $FW exec $RUN --proof naming/naming-review-all -- find '{mediaDir}/shelf-series' -type f
  ```

- **Test file names.** Paste four names and `Test`, then open the first result's `Details`.

  ```sh
  X='getByRole("dialog", { name: "Test file names" })'
  $FW exec $RUN -- $PW cli $S click 'getByRole("button", { name: "Test file names" })'
  $FW exec $RUN -- $PW cli $S fill "$X"'.getByRole("textbox", { name: "File paths" })' "$(printf '%s\n' The.Tin.Signal.S02E06.1080p.WEB-DL.DDP5.1.H264-FKW.mkv Saltmarsh.Radio.2018.2160p.UHD.BluRay.x265-FLUX.mkv The.Tin.Signal.S07E01.1080p.WEB-DL.H264-FKW.mkv Unheard.Of.Film.1999.1080p.BluRay.x264-NOPE.mkv)"
  $FW exec $RUN -- $PW cli $S click "$X"'.getByRole("button", { name: "Test", exact: true })'
  $FW exec $RUN -- $PW cli $S click "$X"'.getByRole("list", { name: "Results" }).getByRole("button", { name: "Details" }).first()'
  $FW exec $RUN --proof naming/naming-tester -- $PW cli $S snapshot "$X" --filename={runDir}/proof/naming/naming-tester/results.aria.yml
  $FW exec $RUN --proof naming/naming-tester -- $PW cli $S screenshot --filename={runDir}/proof/naming/naming-tester/results.png
  $FW exec $RUN -- $PW cli $S press Escape
  ```

- **Rename files… on a Work.** Open The Tin Signal, its menu and `Rename files…`. The one Work's files are open. Rename them.

  ```sh
  W='getByRole("dialog", { name: "Rename files of The Tin Signal" })'
  $FW exec $RUN -- $PW cli $S goto "{nodes.home.browserUrl}/works/$TIN"
  $FW exec $RUN -- $PW cli $S click 'getByRole("button", { name: "More actions for The Tin Signal" })'
  $FW exec $RUN --proof naming/naming-work-rename -- $PW cli $S snapshot 'getByRole("menu")' --filename={runDir}/proof/naming/naming-work-rename/menu.aria.yml
  $FW exec $RUN -- $PW cli $S click 'getByRole("menuitem", { name: "Rename files…" })'
  $FW exec $RUN -- $PW cli $S run-code 'async page => { await page.getByRole("dialog", { name: "Rename files of The Tin Signal" }).getByRole("insertion").first().waitFor(); }'
  $FW exec $RUN --proof naming/naming-work-rename -- $PW cli $S snapshot "$W" --filename={runDir}/proof/naming/naming-work-rename/plan.aria.yml
  $FW exec $RUN --proof naming/naming-work-rename -- $PW cli $S screenshot --filename={runDir}/proof/naming/naming-work-rename/plan.png
  $FW exec $RUN -- $PW cli $S click "$W"'.getByRole("button", { name: "Rename 3 files" })'
  $FW exec $RUN -- $PW cli $S run-code 'async page => { await page.getByRole("dialog").waitFor({ state: "detached" }); }'
  $FW exec $RUN --proof naming/naming-work-rename -- find '{mediaDir}/shelf-series'
  ```

- **Name by air date.** Open `Edit…`, switch `Name by air date` on, which saves at once, and follow `Rename files` to the plan. Rename.

  ```sh
  E='getByRole("dialog", { name: "Edit The Tin Signal" })'
  $FW exec $RUN -- $PW cli $S click 'getByRole("button", { name: "More actions for The Tin Signal" })'
  $FW exec $RUN -- $PW cli $S click 'getByRole("menuitem", { name: "Edit…" })'
  $FW exec $RUN -- $PW cli $S click "$E"'.getByRole("switch", { name: "Name by air date" })'
  $FW exec $RUN -- $PW cli $S run-code 'async page => { await page.getByText("Files keep their names and folders until you rename them.").waitFor(); }'
  $FW exec $RUN --proof naming/naming-air-date -- $PW cli $S screenshot --filename={runDir}/proof/naming/naming-air-date/saved.png
  $FW exec $RUN -- $PW cli $S click "$E"'.getByRole("button", { name: "Rename files" })'
  $FW exec $RUN -- $PW cli $S run-code 'async page => { await page.getByRole("dialog", { name: "Rename files of The Tin Signal" }).getByRole("insertion").first().waitFor(); }'
  $FW exec $RUN --proof naming/naming-air-date -- $PW cli $S snapshot "$W" --filename={runDir}/proof/naming/naming-air-date/plan.aria.yml
  $FW exec $RUN --proof naming/naming-air-date -- $PW cli $S screenshot --filename={runDir}/proof/naming/naming-air-date/plan.png
  $FW exec $RUN -- $PW cli $S click "$W"'.getByRole("button", { name: "Rename 3 files" })'
  $FW exec $RUN -- $PW cli $S run-code 'async page => { await page.getByRole("dialog").waitFor({ state: "detached" }); }'
  $FW exec $RUN --proof naming/naming-air-date -- curl -sS -H 'X-Api-Key: {apiKey}' "{nodes.home.url}/api/v1/works/$TIN"
  $FW exec $RUN --proof naming/naming-air-date -- find '{mediaDir}/shelf-series' -type f
  ```

- **An Impact's renames.** Rename the `1080p` Target to `Full HD`, open the Impact's `Details`, and `Save`.

  ```sh
  T1080=$($FW exec $RUN -- curl -sS -H 'X-Api-Key: {apiKey}' '{nodes.home.url}/api/v1/targets' | jq -r '.targets[] | select(.name == "1080p") | .id')
  $FW exec $RUN -- $PW cli $S goto "{nodes.home.browserUrl}/settings/targets/$T1080"
  $FW exec $RUN -- $PW cli $S fill 'getByRole("main").getByRole("textbox", { name: "Name" })' 'Full HD'
  $FW exec $RUN -- $PW cli $S click 'getByRole("button", { name: "Save", exact: true })'
  $FW exec $RUN -- $PW cli $S click 'getByRole("dialog", { name: "Save 1080p" }).getByRole("button", { name: "Details" })'
  $FW exec $RUN --proof naming/naming-impact -- $PW cli $S snapshot 'getByRole("dialog", { name: "Save 1080p" })' --filename={runDir}/proof/naming/naming-impact/impact.aria.yml
  $FW exec $RUN --proof naming/naming-impact -- $PW cli $S screenshot --filename={runDir}/proof/naming/naming-impact/impact.png
  $FW exec $RUN -- $PW cli $S click 'getByRole("dialog", { name: "Save 1080p" }).getByRole("button", { name: "Save", exact: true })'
  $FW exec $RUN -- $PW cli $S run-code 'async page => { await page.getByText("Saved Full HD.").waitFor(); }'
  $FW exec $RUN --proof naming/naming-impact -- find '{mediaDir}/shelf-movies' '{mediaDir}/shelf-series' -type f
  ```

- **The rename switch.** On TV, switch `Rename files` off and `Save`; read the lead, then `Turn on renaming`.

  ```sh
  $FW exec $RUN -- $PW cli $S goto '{nodes.home.browserUrl}/settings/naming'
  $FW exec $RUN -- $PW cli $S click "$T"'.getByRole("button", { name: "Details" })'
  $FW exec $RUN -- $PW cli $S click "$T"'.getByRole("switch", { name: "Rename files" })'
  $FW exec $RUN -- $PW cli $S click "$T"'.getByRole("button", { name: "Save", exact: true })'
  $FW exec $RUN -- $PW cli $S run-code 'async page => { await page.getByRole("article", { name: "TV" }).getByRole("button", { name: "Turn on renaming" }).waitFor(); }'
  $FW exec $RUN --proof naming/naming-switch -- $PW cli $S snapshot "$T" --filename={runDir}/proof/naming/naming-switch/off.aria.yml
  $FW exec $RUN --proof naming/naming-switch -- $PW cli $S screenshot --filename={runDir}/proof/naming/naming-switch/off.png
  $FW exec $RUN -- $PW cli $S click "$T"'.getByRole("button", { name: "Turn on renaming" })'
  $FW exec $RUN -- $PW cli $S run-code 'async page => { await page.getByRole("article", { name: "TV" }).getByText("Plex, Jellyfin and Emby show the files of each series as one item.").waitFor(); }'
  $FW exec $RUN --proof naming/naming-switch -- curl -sS -H 'X-Api-Key: {apiKey}' '{nodes.home.url}/api/v1/naming'
  ```

## Gotchas

- `fakeworld exec` fills `{name}` tokens from the manifest and refuses one it doesn't know, so every literal `{` in a template, a JSON body or a `fill` value is written `{{`. `}` needs no escape.
- `PATCH /api/v1/naming/{type}` takes `{"values": {...}}` and the preview takes `{"draft": {...}}`; each key is the full setting key, such as `movie.naming.file.standard`.
- `Movie file` and `Episode file` are comboboxes that suggest tokens as you type. Press `Escape` after a `fill` so the suggestions close before the next click. TV has three episode templates, so `Episode file` needs `exact: true`.
- A Work's own plan opens its files already; `Files of <title>` collapses them. A plan of several Works opens each on its button.
- A type option such as `Name by air date` saves when it switches, and the edit dialog's `Save` then only closes it, losing the `Rename files` link. Follow the link before `Save`.
- The home Scenario also holds the empty `movies` and `tv` roots, so wait on the card's count, not on a number of finished scans.
- The plan's Standard file samples carry only a Target, so a quality token prints in the `every release fact` sample and not in the Standard one.
- `e2e/naming_test.go` drives the save plan with a file appearing at its new path, two files that trade names after an episode order change, Name by air date, a Target rename's Impact, renaming off and a name too long for a Windows Node, through the API, in CI.
