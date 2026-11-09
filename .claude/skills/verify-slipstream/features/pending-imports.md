# Pending imports

Pending imports lists each folder a Library scan found that joins no Work with confidence, and waits for the Admin to choose. It sits under the Library at `/library/pending`, a pushed page on phone, and the Library links to it while any folder needs the Admin. The page leads, for each library root, with what that root's last scan added and how many folders need the Admin, linking to the Works it added. Each waiting folder, a Pending import, shows why it waits and its suggested match, with Add, Change, Import into and Ignore, and its hint, parsed quality and files under Details. The Ignored view lists the folders the Admin set aside, including the folder of a Work deleted with its files left on disk, so the next scan never adds that Work back.

## Sub-features

- `pending-list` on `home` with the two series added and the shelves added as roots, `/library/pending` reads, for each root by its words, such as `Movies library root at <path> on home` when home holds two Movies roots and `Series library root on home` when it holds one, `The last scan of the <root> added 3 movies. 2 folders need you.` with `Show the movies it added`, or `didn't add any series. 1 folder needs you.` Each folder reads its name, `1 file · 789 KB · HDTV-720p`, its reason, its suggestion and its actions: `Brass Meridian (2015)` `This series has no episode S03E09.` with `Suggested match Brass Meridian (2015)` and `In the library`, `Grandma's 80th Birthday (2019)` `No movie matches Grandma's 80th Birthday (2019).` with `No match yet.` and no Add, and `Nightfall` `2 movies match Nightfall. The name has no year.` with `Suggested match Nightfall (1998)`. `Add all 2` sits in the top bar. The Library's link reads `3 folders need you` over `A library scan found them and can't match them with confidence.`, and `GET /api/v1/counts` answers `pendingImports` (P-024).
- `pending-change` `Change` on Nightfall opens `Change the match for Nightfall`, `Search TMDB for the movie in this folder.`, with `Nightfall (1998)` and `Nightfall (2014)`; `Match Nightfall (2014)` toasts `Nightfall now matches Nightfall (2014).` and the suggestion reads `Nightfall (2014)` (P-024).
- `pending-add` `Add` on Nightfall toasts `Added Nightfall (2014).`, the folder leaves the list, and the Library holds Nightfall with its file, added by the Admin (P-024).
- `pending-import-into` `Import into` on Brass Meridian (2015) opens `Import into a Series` on Brass Meridian with `Pick another series` and one picker per file, `Brass Meridian - S03E09.mkv` reading `Leave it waiting` and offering `S01E01 Noon` and `S02E05 Longitude`; `Import` with S01E01 toasts `Imported 1 file into Brass Meridian.`, and the folder leaves the list (P-024).
- `pending-ignore` `Ignore` on Grandma's 80th Birthday (2019) toasts `Ignored Grandma's 80th Birthday (2019). A library scan lists it again if its files change.` with `Undo`, and the root's lead reads `No folders need you.`; `Ignored` (`?view=ignored`) lists it with `Restore` and `Import into`, and its `Details` read `Name reads` and each file's quality and size with its reason.
- `pending-leave-files` deleting Lanterns at Low Tide with `Leave files on disk` reads `Its file (524 KB) stays on disk and shows under Ignored in Pending imports.` and toasts `Deleted Lanterns at Low Tide. Its file stays on disk.`; `Ignored` lists its folder with `This movie was deleted from the library and its files were left on disk.`, and Scan now on its root reads `Nothing changed.` and adds nothing (P-024).

## How to get to it (user POV)

- Desktop shell: the Library's `<n> folders need you` row, above the filters, opens `/library/pending`; the `Needs you` and `Ignored` switch beside the title swaps the views.
- Phone shell: the `Library` tab, then the same row, which pushes `Pending imports` with a back link `Library`. Change and Import into open as sheets.
- API (Admin): `GET /api/v1/imports/pending?rootId=&ignored=`, `POST /api/v1/imports/pending/add {ids}`, `POST /api/v1/imports/pending/{id}/match {key}`, `POST /api/v1/imports/pending/{id}/import {workId, files}` answering `{entry}` with `null` once every file is imported, `POST /api/v1/imports/pending/ignore {ids}` and `/unignore {ids}`, and `pendingImports` in `GET /api/v1/counts`. Each entry carries `root {id, label}`. Changes publish `imports.changed {rootIds}`.
- Host CLI: none.

## Driving it with fakeworld, curl and playwright cli

Preconditions:

- A fresh `home` run with doctor passing and `{runId}-admin-desktop` signed in, as in the skill's Launch and Drive. `home`'s `shelf-movies` and `shelf-series` are on disk under `{mediaDir}`, and no root names them.
- `S='-s={runId}-admin-desktop'`.
- The two series added before their root, so their folders join them, then the shelves added as roots. Each root's first Ready report scans it within seconds.

  ```sh
  for key in 9100201 9100202; do $FW exec $RUN --proof pending-imports/pending-list -- curl -sS -X POST -H 'X-Api-Key: {apiKey}' -H 'Content-Type: application/json' --data-binary "{\"contentType\":\"series\",\"key\":\"$key\"}" '{nodes.home.url}/api/v1/works'; done
  $FW exec $RUN --proof pending-imports/pending-list -- curl -sS -X POST -H 'X-Api-Key: {apiKey}' -H 'Content-Type: application/json' --data-binary '{"contentType":"series","nodeId":"{nodes.home.nodeId}","path":"{mediaDir}/shelf-series"}' '{nodes.home.url}/api/v1/roots'
  $FW exec $RUN --proof pending-imports/pending-list -- curl -sS -X POST -H 'X-Api-Key: {apiKey}' -H 'Content-Type: application/json' --data-binary '{"contentType":"movie","nodeId":"{nodes.home.nodeId}","path":"{mediaDir}/shelf-movies"}' '{nodes.home.url}/api/v1/roots'
  ```

Steps:

- **The list.** Open the Library and follow `3 folders need you`. The page reads the three leads and three folders above, and the API answers the same entries with their reasons.

  ```sh
  $FW exec $RUN -- $PW cli $S goto '{nodes.home.browserUrl}/library'
  $FW exec $RUN -- $PW cli $S click 'getByRole("list", { name: "Pending imports" }).getByRole("link")'
  $FW exec $RUN -- $PW cli $S run-code 'async page => { await page.getByRole("list", { name: /^Pending imports in the/ }).first().waitFor(); }'
  $FW exec $RUN --proof pending-imports/pending-list -- $PW cli $S snapshot --filename={runDir}/proof/pending-imports/pending-list/page.aria.yml
  $FW exec $RUN --proof pending-imports/pending-list -- $PW cli $S screenshot --full-page --filename={runDir}/proof/pending-imports/pending-list/page.png
  $FW exec $RUN --proof pending-imports/pending-list -- curl -sS -H 'X-Api-Key: {apiKey}' '{nodes.home.url}/api/v1/imports/pending'
  ```

- **Change, then Add.** Change Nightfall's match to the 2014 film, then Add it.

  ```sh
  NF='getByRole("listitem", { name: "Nightfall", exact: true })'
  $FW exec $RUN -- $PW cli $S click "$NF"'.getByRole("button", { name: "Change" })'
  $FW exec $RUN --proof pending-imports/pending-change -- $PW cli $S screenshot --filename={runDir}/proof/pending-imports/pending-change/sheet.png
  $FW exec $RUN -- $PW cli $S click 'getByRole("dialog", { name: "Change the match for Nightfall" }).getByRole("button", { name: "Match Nightfall (2014)" })'
  $FW exec $RUN -- $PW cli $S run-code 'async page => { await page.getByRole("listitem", { name: "Nightfall", exact: true }).getByText("Nightfall (2014)").waitFor(); }'
  $FW exec $RUN --proof pending-imports/pending-change -- $PW cli $S screenshot --filename={runDir}/proof/pending-imports/pending-change/changed.png
  $FW exec $RUN -- $PW cli $S click "$NF"'.getByRole("button", { name: "Add", exact: true })'
  $FW exec $RUN -- $PW cli $S run-code 'async page => { await page.getByRole("region", { name: "Notifications" }).getByText("Added Nightfall (2014).").waitFor(); }'
  $FW exec $RUN --proof pending-imports/pending-add -- $PW cli $S screenshot --filename={runDir}/proof/pending-imports/pending-add/added.png
  $FW exec $RUN --proof pending-imports/pending-add -- curl -sS -H 'X-Api-Key: {apiKey}' '{nodes.home.url}/api/v1/library'
  ```

- **Import into.** Import Brass Meridian's S03E09 file into Brass Meridian as S01E01. The Work's files list it.

  ```sh
  INTO='getByRole("dialog", { name: "Import into a Series" })'
  $FW exec $RUN -- $PW cli $S click 'getByRole("listitem", { name: "Brass Meridian (2015)" }).getByRole("button", { name: "Import into" })'
  $FW exec $RUN -- $PW cli $S click "$INTO"'.getByRole("combobox", { name: "Brass Meridian - S03E09.mkv" })'
  $FW exec $RUN -- $PW cli $S click 'getByRole("option", { name: "S01E01 Noon" })'
  $FW exec $RUN --proof pending-imports/pending-import-into -- $PW cli $S screenshot --filename={runDir}/proof/pending-imports/pending-import-into/picked.png
  $FW exec $RUN -- $PW cli $S click "$INTO"'.getByRole("button", { name: "Import", exact: true })'
  $FW exec $RUN -- $PW cli $S run-code 'async page => { await page.getByRole("region", { name: "Notifications" }).getByText("Imported 1 file into Brass Meridian.").waitFor(); }'
  BRASS=$($FW exec $RUN -- curl -sS -H 'X-Api-Key: {apiKey}' '{nodes.home.url}/api/v1/library' | jq -r '.rows[] | select(.label == "Brass Meridian") | .id')
  $FW exec $RUN --proof pending-imports/pending-import-into -- curl -sS -H 'X-Api-Key: {apiKey}' "{nodes.home.url}/api/v1/works/$BRASS/files"
  ```

- **Ignore.** Ignore Grandma's 80th Birthday (2019), then open `Ignored` and its `Details`.

  ```sh
  $FW exec $RUN -- $PW cli $S click 'getByRole("listitem", { name: "Grandma'"'"'s 80th Birthday (2019)" }).getByRole("button", { name: "Ignore" })'
  $FW exec $RUN -- $PW cli $S run-code 'async page => { await page.getByRole("region", { name: "Notifications" }).getByText(/^Ignored /).waitFor(); }'
  $FW exec $RUN --proof pending-imports/pending-ignore -- $PW cli $S screenshot --filename={runDir}/proof/pending-imports/pending-ignore/ignored-toast.png
  $FW exec $RUN -- $PW cli $S click 'getByRole("navigation", { name: "Pending imports views" }).getByRole("link", { name: "Ignored" })'
  $FW exec $RUN -- $PW cli $S click 'getByRole("listitem", { name: "Grandma'"'"'s 80th Birthday (2019)" }).getByRole("button", { name: "Details" })'
  $FW exec $RUN --proof pending-imports/pending-ignore -- $PW cli $S snapshot --filename={runDir}/proof/pending-imports/pending-ignore/details.aria.yml
  $FW exec $RUN --proof pending-imports/pending-ignore -- $PW cli $S screenshot --filename={runDir}/proof/pending-imports/pending-ignore/details.png
  ```

- **Leave files.** Delete Lanterns at Low Tide from its Work page's `More actions for Lanterns at Low Tide` with `Leave files on disk`. Ignored lists its folder, and Scan now on `shelf-movies` adds nothing back.

  ```sh
  LANT=$($FW exec $RUN -- curl -sS -H 'X-Api-Key: {apiKey}' '{nodes.home.url}/api/v1/library' | jq -r '.rows[] | select(.label == "Lanterns at Low Tide") | .id')
  DEL='getByRole("dialog", { name: "Delete Lanterns at Low Tide?" })'
  $FW exec $RUN -- $PW cli $S goto "{nodes.home.browserUrl}/works/$LANT"
  $FW exec $RUN -- $PW cli $S click 'getByRole("button", { name: "More actions for Lanterns at Low Tide" })'
  $FW exec $RUN -- $PW cli $S click 'getByRole("menuitem", { name: /^Delete/ })'
  $FW exec $RUN -- $PW cli $S check "$DEL"'.getByRole("radio", { name: "Leave files on disk" })'
  $FW exec $RUN --proof pending-imports/pending-leave-files -- $PW cli $S screenshot --filename={runDir}/proof/pending-imports/pending-leave-files/dialog.png
  $FW exec $RUN -- $PW cli $S click "$DEL"'.getByRole("button", { name: "Delete", exact: true })'
  SHELF=$($FW exec $RUN -- curl -sS -H 'X-Api-Key: {apiKey}' '{nodes.home.url}/api/v1/roots' | jq -r '.roots[] | select(.path | endswith("shelf-movies")) | .id')
  $FW exec $RUN --proof pending-imports/pending-leave-files -- curl -sS -X POST -H 'X-Api-Key: {apiKey}' "{nodes.home.url}/api/v1/roots/$SHELF/scan"
  $FW exec $RUN -- $PW cli $S goto '{nodes.home.browserUrl}/library/pending?view=ignored'
  $FW exec $RUN --proof pending-imports/pending-leave-files -- $PW cli $S snapshot --filename={runDir}/proof/pending-imports/pending-leave-files/ignored.aria.yml
  $FW exec $RUN --proof pending-imports/pending-leave-files -- curl -sS -H 'X-Api-Key: {apiKey}' '{nodes.home.url}/api/v1/library'
  ```

- **Phone.** Open `{runId}-admin-phone` as in the skill's Drive with `phone` and follow the Library's row: `Pending imports` is a pushed page, and `Change` opens a sheet.

## Gotchas

- exec fills every `{...}` in a command, so a folder tagged with an id, such as `Lanterns at Low Tide (2016) {tmdb-9000101}`, needs `{{tmdb-9000101}}` inside exec.
- A folder whose name matches no title exactly still shows the provider's first search result as its suggestion, while its reason reads `No movie matches <name>.` A folder with no search result shows `No match yet.` and offers no Add.
- An `{imdb-tt…}` tag matches only a Work already in the library. At the provider, a scan looks up only the type's own id, `{tmdb-…}` for a movie and `{tvdb-…}` or `[tvdbid-…]` for a series, and an other tag falls back to the title and year.
- A search that ends with a year, such as `Brass Meridian 2015`, filters a series to that year as it does a movie, in Change and in the add flow alike.
- A folder of a Work in another root of its type offers only `Ignore`, since a Work's files all live in one root.
- A file the disk queue was still moving into the Work's folder when the Work was deleted with its files left isn't listed under Ignored. The holding sweep removes those bytes, a window of seconds that predates Library scan.
- Pending imports never raise a Health item. They are decisions about Works, so they stay in the Library's count.
