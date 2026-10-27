# Targets and Profiles

An Admin decides what each file should be. A Target ranks the qualities it accepts in tiers, with a cutoff, hard filters and sizes per quality. A Profile is the ordered Targets a Work wants and its Edition list, and every Work has one. One file fills every Target it meets and ranks best at, a search grabs for every Want an upgrade included, and a release or file a Target refuses says why. Every change that can leave files filling nothing, rename files or make new cells opens its Impact first, with Delete chosen and Keep as Kept files beside it. A fresh Controller starts with the starter Target `1080p` and a default Profile per Content Type, and Settings gains Targets, Profiles and Editions.

## Sub-features

- `targets-starter` a fresh Controller lists the starter Target `1080p`, accepting `HDTV-720p to Remux-1080p` and held by `Movies`, and the default Profile `Movies`, marked `Default`; a Work added with no Profile choice reads `Missing` at `1080p` at once (P-044, P-070).
- `targets-in-use` deleting `1080p` while `Movies` holds it reads `1080p can't be deleted. Movies uses it.` and the API answers 409 `target_in_use` with the detail `Movies uses it`; `Duplicate` makes `1080p copy`, held by no Profile, which deletes.
- `profiles-default-fixed` the default Profile has no `Actions` menu, and deleting it over the API, whatever `moveTo` names, answers 409 `profile_is_default` (P-044).
- `add-sheet-profile` the add sheet's `Profile` picker reads the type's default with `(default)`, and a Work added on another Profile wants that Profile's Targets (P-044).
- `targets-one-file` one WEB-DL fills both `1080p` and `HD`, the cells read `Upgradable` at both, and one file, `The Quiet Orchard (2021) - 1080p.mkv`, named after the first Target, sits on disk once (P-200, P-202, P-209).
- `targets-upgrade` once fixture-site lists the Bluray, a search grabs it, the searches view refuses the held WEB-DL with `not_an_upgrade` at each Target, the Grab records both Targets, and the Bluray replaces the WEB-DL at the same path with both cells `Met` (P-203, P-210, P-211).
- `targets-editor` a new Target filled from `Upgrade for disc or resolution` ticks and ranks its qualities with ties, moves its cutoff, says what its order means in a sentence and `Sample moves`, and keeps `CAM` apart under `Cinema sources`, off, with its warning (P-065, P-066, P-067).
- `targets-filters-sizes` `Filters` sets `Any`, `Only these` or `Not these` per attribute with a value picker, and `Sizes` takes a minimum and a maximum in MiB per minute per accepted quality with what each means for a two-hour film (P-069).
- `profiles-editor` Profiles lists each Content Type's Profiles with the default and the Works counted; a Profile's Targets show a disk estimate and a line naming a Target that contains another; a new Profile ranks its Targets by keyboard, its Editions sentence follows `Choose cuts` and warns `No Standard`, and `Make default` moves the default (P-064, P-201, P-044).
- `work-wants-sheet` the Work page reads `Profile Films. Editions from the Profile.`, and `Profile and Editions` opens a sheet with the `Profile` picker, the Editions it takes from the Profile or `set for this Work` with `Reset to Profile`; the Work page's cells table sits under each wanted cut's heading (P-208).
- `impact-keep` moving the Work to `UHD` opens the Impact, `1 Work changes.`, `1 file would fill nothing`, `1 cell becomes wanted.`, with `Delete` chosen; `Keep as Kept files` and `Save and keep 1 file` leave the file as `The Quiet Orchard (2021) - 1080p (extra 1).mkv`, its old label plus ` (extra 1)` (P-205, P-202, P-207).
- `profiles-delete-move` deleting `UHD` asks where its Works go; the Work moves to `Films` and the Kept file fills again under `The Quiet Orchard (2021) - 1080p.mkv` (P-064).
- `impact-delete` taking both Targets out of `Films` opens the Impact with `Delete` chosen and `Save and delete 1 file`, which removes the file from the media root and leaves the Work with no cells (P-199, P-204, P-206).
- `sizes-refusals` on the `sizes` Scenario a search refuses the 14 GiB encode with `size_above` and the 1.5 GiB one with `size_below`, grabs the 8.2 GiB one, and a Work whose runtime TMDB doesn't know grabs with the note `size_not_checked` (P-203).
- `sizes-probed-codec` a release whose title names no video codec is grabbed for `Encoded`, which refuses x264, and its file, probed as H.264, fills nothing. The Grab fails at probing with `fills_nothing`, no file lands, and the cell reads `Failed` (P-069).
- `editions-cuts` Settings > Editions lists every cut with the phrases it is read as and where it comes from, and `Add cut` adds `Superfan`, also read as `Fan Cut`, as `Custom`.
- `editions-cut-rename` renaming `Superfan` to `Superfan Edition` opens the Impact, `1 Work changes.` and `1 file is renamed.`, with no Delete or Keep, and its file is renamed to `{edition-Superfan Edition}` (P-202).
- `editions-cut-in-use` deleting `Superfan Edition` while `Films` lists it and a file holds it reads `Superfan Edition can't be deleted. Films and 1 file use it.`, and the API answers 409 `cut_in_use`.

## How to get to it (user POV)

- Desktop shell: the sidebar's `Settings`, then `Targets`, `Profiles` or `Editions` in the `Settings` navigation (`/settings/targets`, `/settings/profiles`, `/settings/editions`). A Target's name opens `/settings/targets/<id>` and `New Target` opens `/settings/targets/new`; Profiles likewise.
- Phone shell: `More`, then `Settings`, then the section in the `Settings` list; each section and editor is a pushed page. A ranked row's `Actions for <name>` opens a sheet of that name holding `Move up`, `Move down` and, where a tie can change, `Tie with the one above` or `Untie from the one above`.
- Both shells: the Work page's `Profile and Editions`, above its cells; the add sheet that `Add <title>` opens on `/add`.
- API (Admin): `GET /api/v1/qualities`; `GET`, `POST /api/v1/targets`, `GET`, `PUT`, `DELETE /api/v1/targets/{targetId}`, `POST /api/v1/targets/{targetId}/duplicate`; `GET`, `POST /api/v1/profiles`, `GET`, `PUT /api/v1/profiles/{profileId}`, `DELETE /api/v1/profiles/{profileId}?moveTo=<profileId>`, `POST /api/v1/profiles/{profileId}/default`; `GET`, `POST /api/v1/cuts`, `PUT`, `DELETE /api/v1/cuts/{cutKey}`; `GET`, `PUT /api/v1/works/{workId}/wants`; `GET /api/v1/works/{workId}/cells` and `/searches`. Every change takes `?preview=true` for its Impact alone, `?unwanted=delete|keep` and `?confirm=<n>`, and publishes `targets.changed`.
- Host CLI: none.

## Driving it with fakeworld and playwright cli

Preconditions:

- Three runs, each launched as in the skill's Launch with its own `RUN`, doctor passing, and `{runId}-admin-desktop` signed in:
  - `home`, fresh, for the starter set. The rest of `home`'s features run on it after.
  - `targets`, fresh, for everything from the add sheet to the Impact. Its movies want `1080p` and `HD`, a second Target of the same qualities, through the default Profile `Films`. fixture-site lists a 1080p WEB-DL, and holds back a 1080p Bluray, an Extended WEB-DL and a Superfan WEB-DL until `fixture-site-list` lists each.
  - `sizes`, fresh, whose `Films` wants `Sized` (WEBDL-1080p at 20 to 100 MiB per minute, TrueHD only, no 2.0 channels) and whose `Encodes` wants `Encoded` (WEBDL-1080p, not x264).
- The Editions steps need a fresh `targets` run of their own, since they list Superfan in `Films` before the Work is added.

Steps on `home`:

- **The starter set.** Open `/settings/targets`. The `Targets` table's row reads `1080p`, `HDTV-720p to Remux-1080p`, `Bluray-1080p` and `Movies`. Open `Profiles`. The `Movies Profiles` row reads `Movies`, `Default` and `1080p`, with no `Actions for Movies`.

  ```sh
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop goto '{nodes.home.browserUrl}/settings/targets'
  $FW exec $RUN --proof targets-and-profiles/targets-starter -- $PW cli -s={runId}-admin-desktop snapshot --filename={runDir}/proof/targets-and-profiles/targets-starter/targets.aria.yml
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop click 'getByRole("navigation", { name: "Settings" }).getByRole("link", { name: "Profiles" })'
  $FW exec $RUN --proof targets-and-profiles/targets-starter -- $PW cli -s={runId}-admin-desktop screenshot --filename={runDir}/proof/targets-and-profiles/targets-starter/profiles.png
  $FW exec $RUN --proof targets-and-profiles/targets-starter -- curl -sS -H 'X-Api-Key: {apiKey}' '{nodes.home.url}/api/v1/profiles'
  ```

- **The default Profile stays.** Delete `Movies` over the API: 409 `profile_is_default`. The route requires `moveTo`, and the default check comes first, so naming `Movies` itself reaches it.

  ```sh
  MOVIES=$($FW exec $RUN -- curl -sS -H 'X-Api-Key: {apiKey}' '{nodes.home.url}/api/v1/profiles' | jq -r '.profiles[] | select(.name == "Movies") | .id')
  $FW exec $RUN --proof targets-and-profiles/profiles-default-fixed -- curl -sS -i -X DELETE -H 'X-Api-Key: {apiKey}' "{nodes.home.url}/api/v1/profiles/$MOVIES?moveTo=$MOVIES"
  ```

- **A held Target can't go.** On `/settings/targets`, choose `Actions for 1080p`, then `Delete`, then `Delete` in `Delete 1080p?`. The dialog reads `1080p can't be deleted. Movies uses it.` Cancel it; the API answers 409 `target_in_use` with the detail `Movies uses it`.

  ```sh
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop goto '{nodes.home.browserUrl}/settings/targets'
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop click 'getByRole("button", { name: "Actions for 1080p" })'
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop click 'getByRole("menuitem", { name: "Delete" })'
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop click 'getByRole("dialog", { name: "Delete 1080p?" }).getByRole("button", { name: "Delete" })'
  $FW exec $RUN --proof targets-and-profiles/targets-in-use -- $PW cli -s={runId}-admin-desktop screenshot --filename={runDir}/proof/targets-and-profiles/targets-in-use/refused.png
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop press Escape
  STARTER=$($FW exec $RUN -- curl -sS -H 'X-Api-Key: {apiKey}' '{nodes.home.url}/api/v1/targets' | jq -r '.targets[] | select(.name == "1080p") | .id')
  $FW exec $RUN --proof targets-and-profiles/targets-in-use -- curl -sS -i -X DELETE -H 'X-Api-Key: {apiKey}' "{nodes.home.url}/api/v1/targets/$STARTER"
  ```

- **Duplicate.** Choose `Actions for 1080p`, then `Duplicate`. The toast reads `Made 1080p copy from 1080p.`, and the new row's Profiles read `None`. Delete `1080p copy` the same way, and the toast reads `Deleted 1080p copy.`

  ```sh
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop click 'getByRole("button", { name: "Actions for 1080p" })'
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop click 'getByRole("menuitem", { name: "Duplicate" })'
  $FW exec $RUN --proof targets-and-profiles/targets-in-use -- $PW cli -s={runId}-admin-desktop snapshot --filename={runDir}/proof/targets-and-profiles/targets-in-use/duplicated.aria.yml
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop click 'getByRole("button", { name: "Actions for 1080p copy" })'
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop click 'getByRole("menuitem", { name: "Delete" })'
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop click 'getByRole("dialog", { name: "Delete 1080p copy?" }).getByRole("button", { name: "Delete" })'
  ```

- **Add with the default.** Open `/add?contentType=movie&query=quiet+orchard` and choose `Add The Quiet Orchard`. The sheet `Add The Quiet Orchard` reads `Movies (default)` in `Profile`. Choose `Add Movie`. The Work page reads `Profile Movies. Editions from the Profile.`, and the `Any cut cells` table reads `Movie`, `Status`, `1080p` with `Missing` and `Missing`.

  ```sh
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop goto '{nodes.home.browserUrl}/add?contentType=movie&query=quiet+orchard'
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop click 'getByRole("button", { name: "Add The Quiet Orchard" })'
  $FW exec $RUN --proof targets-and-profiles/add-sheet-profile -- $PW cli -s={runId}-admin-desktop screenshot --filename={runDir}/proof/targets-and-profiles/add-sheet-profile/sheet.png
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop click 'getByRole("dialog", { name: "Add The Quiet Orchard" }).getByRole("button", { name: "Add Movie" })'
  $FW exec $RUN --proof targets-and-profiles/targets-starter -- $PW cli -s={runId}-admin-desktop snapshot --filename={runDir}/proof/targets-and-profiles/targets-starter/work.aria.yml
  ```

Steps on `targets`:

- **Add on Films and search.** Add The Quiet Orchard as above; the picker reads `Films (default)`. Choose `Search`. Once the WEB-DL imports, both cells read `Upgradable`, and one file sits in the media root.

  ```sh
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop goto '{nodes.home.browserUrl}/add?contentType=movie&query=quiet+orchard'
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop click 'getByRole("button", { name: "Add The Quiet Orchard" })'
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop click 'getByRole("dialog", { name: "Add The Quiet Orchard" }).getByRole("button", { name: "Add Movie" })'
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop click 'getByRole("banner").getByRole("button", { name: "Search", exact: true })'
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop run-code 'async page => { await page.getByRole("table", { name: "Any cut cells" }).getByText("Upgradable").first().waitFor({ timeout: 60000 }); }'
  $FW exec $RUN --proof targets-and-profiles/targets-one-file -- $PW cli -s={runId}-admin-desktop screenshot --filename={runDir}/proof/targets-and-profiles/targets-one-file/work.png
  WORK=$($FW exec $RUN -- curl -sS -H 'X-Api-Key: {apiKey}' '{nodes.home.url}/api/v1/works' | jq -r '.works[0].id')
  $FW exec $RUN --proof targets-and-profiles/targets-one-file -- curl -sS -H 'X-Api-Key: {apiKey}' "{nodes.home.url}/api/v1/works/$WORK/cells"
  $FW exec $RUN --proof targets-and-profiles/targets-one-file -- $FW fs match {runDir} '{mediaDir}/movies/The Quiet Orchard (2021)/The Quiet Orchard (2021) - 1080p.mkv' --release The.Quiet.Orchard.2021.1080p.WEB-DL.H264-FKW
  ```

- **Upgrade to the Bluray.** List the Bluray and search again. The searches view's newest search reads `grabbed` with 2 candidates and refuses the WEB-DL with `not_an_upgrade` at each Target; the Grab's `targetIds` name both. Both cells read `Met`, and the same path now holds the Bluray.

  ```sh
  $FW cue $RUN fixture-site-list '{"release": "The.Quiet.Orchard.2021.1080p.BluRay.x264-FKW"}'
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop click 'getByRole("banner").getByRole("button", { name: "Search", exact: true })'
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop run-code 'async page => { await page.getByRole("table", { name: "Any cut cells" }).getByText("Met").first().waitFor({ timeout: 60000 }); }'
  $FW exec $RUN --proof targets-and-profiles/targets-upgrade -- curl -sS -H 'X-Api-Key: {apiKey}' "{nodes.home.url}/api/v1/works/$WORK/searches"
  $FW exec $RUN --proof targets-and-profiles/targets-upgrade -- curl -sS -H 'X-Api-Key: {apiKey}' '{nodes.home.url}/api/v1/grabs'
  $FW exec $RUN --proof targets-and-profiles/targets-upgrade -- $FW fs match {runDir} '{mediaDir}/movies/The Quiet Orchard (2021)/The Quiet Orchard (2021) - 1080p.mkv' --release The.Quiet.Orchard.2021.1080p.BluRay.x264-FKW
  ```

- **Build a Target.** On `/settings/targets` choose `New Target`, name it `2160p`, choose `Fill from a preset`, then `Upgrade for disc or resolution`, and untick every 1080p, 720p and 480p quality, `DVD` and `SDTV`. The sentence reads `Grabs HDTV-2160p, WEBRip-2160p or WEBDL-2160p or better. Upgrades until Bluray-2160p or Remux-2160p, then stops.`, and `Sample moves` holds `HDTV-2160p and WEBRip-2160p are tied.` `CAM` sits unticked under `Cinema sources` with `Cinema recordings (CAM, TS, TELESYNC, HDCAM) are filmed in a theatre.`

  ```sh
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop goto '{nodes.home.browserUrl}/settings/targets/new'
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop fill 'getByRole("textbox", { name: "Name" })' 2160p
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop click 'getByRole("button", { name: "Fill from a preset" })'
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop click 'getByRole("menuitem", { name: "Upgrade for disc or resolution" })'
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop run-code 'async page => { for (const q of ["Remux-1080p", "Bluray-1080p", "WEBDL-1080p", "WEBRip-1080p", "HDTV-1080p", "Bluray-720p", "WEBDL-720p", "WEBRip-720p", "HDTV-720p", "WEBRip-480p", "DVD", "SDTV"]) { await page.getByRole("list", { name: "Qualities", exact: true }).getByRole("checkbox", { name: q, exact: true }).click(); } }'
  $FW exec $RUN --proof targets-and-profiles/targets-editor -- $PW cli -s={runId}-admin-desktop snapshot --filename={runDir}/proof/targets-and-profiles/targets-editor/preset.aria.yml
  $FW exec $RUN --proof targets-and-profiles/targets-editor -- $PW cli -s={runId}-admin-desktop screenshot --full-page --filename={runDir}/proof/targets-and-profiles/targets-editor/preset.png
  ```

- **Ties and the cutoff.** A tier's qualities list in the table's order, lowest first. Choose `Tie WEBDL-2160p with the one above` to untie WEBDL-2160p from WEBRip-2160p. It becomes a tier of its own at the bottom, and the sentence starts `Grabs WEBDL-2160p or better.` Choose it again to tie it back. Focus `Move Cutoff` and press ArrowDown. The cutoff sits on the bottom tier and the sentence ends `Keeps the first file it grabs.`, the No upgrades case. Press ArrowUp to put it back.

  ```sh
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop click 'getByRole("button", { name: "Tie WEBDL-2160p with the one above" })'
  $FW exec $RUN --proof targets-and-profiles/targets-editor -- $PW cli -s={runId}-admin-desktop snapshot --filename={runDir}/proof/targets-and-profiles/targets-editor/untied.aria.yml
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop click 'getByRole("button", { name: "Tie WEBDL-2160p with the one above" })'
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop run-code 'async page => { await page.getByRole("button", { name: "Move Cutoff", exact: true }).focus(); await page.keyboard.press("ArrowDown"); }'
  $FW exec $RUN --proof targets-and-profiles/targets-editor -- $PW cli -s={runId}-admin-desktop snapshot --filename={runDir}/proof/targets-and-profiles/targets-editor/no-upgrades.aria.yml
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop press ArrowUp
  ```

- **Filters and sizes.** In `Filters`, set `HDR` to `Only these` and pick `DV` and `HDR10`; the picker reads `HDR: DV, HDR10`. In `Sizes`, set `Remux-2160p minimum` to `200` and `Remux-2160p maximum` to `400`, and the row reads `A two-hour film takes 23 GB to 47 GB.` Choose `Create Target`. The list's `2160p` row reads `HDTV-2160p to Remux-2160p`, and the API's Target holds the filter and the size.

  ```sh
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop click 'getByRole("radiogroup", { name: "HDR" }).getByRole("radio", { name: "Only these" })'
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop click 'getByRole("button", { name: "HDR: Pick values" })'
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop click 'getByRole("menuitemcheckbox", { name: "DV" })'
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop click 'getByRole("menuitemcheckbox", { name: "HDR10", exact: true })'
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop press Escape
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop fill 'getByRole("textbox", { name: "Remux-2160p minimum" })' 200
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop fill 'getByRole("textbox", { name: "Remux-2160p maximum" })' 400
  $FW exec $RUN --proof targets-and-profiles/targets-filters-sizes -- $PW cli -s={runId}-admin-desktop screenshot --full-page --filename={runDir}/proof/targets-and-profiles/targets-filters-sizes/editor.png
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop click 'getByRole("button", { name: "Create Target" })'
  $FW exec $RUN --proof targets-and-profiles/targets-filters-sizes -- curl -sS -H 'X-Api-Key: {apiKey}' '{nodes.home.url}/api/v1/targets'
  ```

- **Profiles.** Open `/settings/profiles`, where `Films` reads `Default` and 1 Work. Open `Films`. Its `Targets` list shows an estimate for each Target, `About <size> across its Works, at <rate> MiB per minute`, and `Every file 1080p takes also meets HD.` Leave without saving. Choose `New Profile`, name it `UHD`, choose `Add Target`, then `2160p`: the row reads `No estimate yet`. Add `1080p` the same way, focus `Move 2160p` and press ArrowDown: `1080p` ranks 1, and the page says `2160p moved to rank 2 of 2.` Choose `Remove 1080p`. Choose `Choose cuts`, `Add cut`, `Extended`, then `Remove Standard`: the chip `No Standard` shows and the sentence reads `Every Item wants Extended outright, even where no release of it has been seen. A release that names no cut is Standard, which this list doesn't want.` Choose `Any cut` again and `Create Profile`.

  ```sh
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop goto '{nodes.home.browserUrl}/settings/profiles'
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop click 'getByRole("table", { name: "Movies Profiles" }).getByRole("link", { name: "Films", exact: true })'
  $FW exec $RUN --proof targets-and-profiles/profiles-editor -- $PW cli -s={runId}-admin-desktop snapshot --filename={runDir}/proof/targets-and-profiles/profiles-editor/films.aria.yml
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop goto '{nodes.home.browserUrl}/settings/profiles/new'
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop fill 'getByRole("textbox", { name: "Name" })' UHD
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop click 'getByRole("button", { name: "Add Target" })'
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop click 'getByRole("menuitem", { name: "2160p", exact: true })'
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop click 'getByRole("button", { name: "Add Target" })'
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop click 'getByRole("menuitem", { name: "1080p", exact: true })'
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop run-code 'async page => { await page.getByRole("button", { name: "Move 2160p", exact: true }).focus(); await page.keyboard.press("ArrowDown"); }'
  $FW exec $RUN --proof targets-and-profiles/profiles-editor -- $PW cli -s={runId}-admin-desktop snapshot --filename={runDir}/proof/targets-and-profiles/profiles-editor/keyboard-rank.aria.yml
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop click 'getByRole("button", { name: "Remove 1080p" })'
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop click 'getByRole("radio", { name: "Choose cuts" })'
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop click 'getByRole("button", { name: "Add cut" })'
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop click 'getByRole("menuitem", { name: "Extended", exact: true })'
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop click 'getByRole("button", { name: "Remove Standard" })'
  $FW exec $RUN --proof targets-and-profiles/profiles-editor -- $PW cli -s={runId}-admin-desktop screenshot --full-page --filename={runDir}/proof/targets-and-profiles/profiles-editor/no-standard.png
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop click 'getByRole("radio", { name: "Any cut" })'
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop click 'getByRole("button", { name: "Create Profile" })'
  ```

- **Make default.** Choose `Actions for UHD`, then `Make default`. The toast reads `New Movies now take UHD.` and `UHD` reads `Default`. Do the same for `Films` to put the default back.

  ```sh
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop click 'getByRole("button", { name: "Actions for UHD" })'
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop click 'getByRole("menuitem", { name: "Make default" })'
  $FW exec $RUN --proof targets-and-profiles/profiles-editor -- $PW cli -s={runId}-admin-desktop snapshot --filename={runDir}/proof/targets-and-profiles/profiles-editor/uhd-default.aria.yml
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop click 'getByRole("button", { name: "Actions for Films" })'
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop click 'getByRole("menuitem", { name: "Make default" })'
  ```

- **Move the Work to UHD and keep its file.** On the Work page choose `Profile and Editions`, pick `UHD` in `Profile`, and the sheet's heading reads `Editions from Profile UHD`. Choose `Save`. The Impact `Save The Quiet Orchard` lists `1 Work changes.`, `1 file would fill nothing (<size>).` and `1 cell becomes wanted.`, with `Delete` chosen. Choose `Keep as Kept files`, then `Save and keep 1 file`. The toast reads `Saved what The Quiet Orchard wants.`, the page reads `Profile UHD. Editions from the Profile.`, and the Bluray now sits under its old label plus ` (extra 1)`.

  ```sh
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop goto "{nodes.home.browserUrl}/works/$WORK"
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop click 'getByRole("button", { name: "Profile and Editions" })'
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop click 'getByRole("dialog", { name: "Profile and Editions" }).getByRole("combobox", { name: "Profile" })'
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop click 'getByRole("option", { name: "UHD" })'
  $FW exec $RUN --proof targets-and-profiles/work-wants-sheet -- $PW cli -s={runId}-admin-desktop screenshot --filename={runDir}/proof/targets-and-profiles/work-wants-sheet/sheet.png
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop click 'getByRole("dialog", { name: "Profile and Editions" }).getByRole("button", { name: "Save", exact: true })'
  $FW exec $RUN --proof targets-and-profiles/impact-keep -- $PW cli -s={runId}-admin-desktop snapshot --filename={runDir}/proof/targets-and-profiles/impact-keep/impact.aria.yml
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop click 'getByRole("dialog", { name: "Save The Quiet Orchard" }).getByRole("radio", { name: "Keep as Kept files" })'
  $FW exec $RUN --proof targets-and-profiles/impact-keep -- $PW cli -s={runId}-admin-desktop screenshot --filename={runDir}/proof/targets-and-profiles/impact-keep/keep.png
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop click 'getByRole("button", { name: "Save and keep 1 file" })'
  $FW exec $RUN --proof targets-and-profiles/impact-keep -- $FW fs match {runDir} '{mediaDir}/movies/The Quiet Orchard (2021)/The Quiet Orchard (2021) - 1080p (extra 1).mkv' --release The.Quiet.Orchard.2021.1080p.BluRay.x264-FKW
  $FW exec $RUN --proof targets-and-profiles/impact-keep -- curl -sS -H 'X-Api-Key: {apiKey}' "{nodes.home.url}/api/v1/works/$WORK/wants"
  ```

- **A Work's own Editions.** Open `Profile and Editions` again and choose `Choose cuts`. The heading reads `Editions set for this Work` beside `Reset to Profile`. Choose `Reset to Profile`, then close the sheet without saving.

  ```sh
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop click 'getByRole("button", { name: "Profile and Editions" })'
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop click 'getByRole("dialog", { name: "Profile and Editions" }).getByRole("radio", { name: "Choose cuts" })'
  $FW exec $RUN --proof targets-and-profiles/work-wants-sheet -- $PW cli -s={runId}-admin-desktop snapshot --filename={runDir}/proof/targets-and-profiles/work-wants-sheet/own-list.aria.yml
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop click 'getByRole("button", { name: "Reset to Profile" })'
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop press Escape
  ```

- **Delete UHD.** On `/settings/profiles` choose `Actions for UHD`, then `Delete`. The dialog `Delete UHD?` reads `Its 1 Work moves to the Profile you pick.` with `Films` in `Move its Works to`. Choose `Delete UHD`, and `Delete UHD` again on its Impact, which lists `1 Work changes.` and `1 file is renamed.` The toast reads `Deleted UHD. Its Works now take Films.`, and the Kept file fills again under the plain label.

  ```sh
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop goto '{nodes.home.browserUrl}/settings/profiles'
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop click 'getByRole("button", { name: "Actions for UHD" })'
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop click 'getByRole("menuitem", { name: "Delete" })'
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop click 'getByRole("dialog", { name: "Delete UHD?" }).getByRole("button", { name: "Delete UHD" })'
  $FW exec $RUN --proof targets-and-profiles/profiles-delete-move -- $PW cli -s={runId}-admin-desktop snapshot --filename={runDir}/proof/targets-and-profiles/profiles-delete-move/impact.aria.yml
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop click 'getByRole("dialog", { name: "Delete UHD" }).getByRole("button", { name: "Delete UHD" })'
  $FW exec $RUN --proof targets-and-profiles/profiles-delete-move -- $FW fs match {runDir} '{mediaDir}/movies/The Quiet Orchard (2021)/The Quiet Orchard (2021) - 1080p.mkv' --release The.Quiet.Orchard.2021.1080p.BluRay.x264-FKW
  ```

- **Take the Targets away and delete.** Open `Films`, choose `Remove 1080p` and `Remove HD`, then `Save`. The Impact `Save Films` lists `1 Work changes.` and `1 file would fill nothing (<size>).`, `Delete` is chosen, and the button reads `Save and delete 1 file`. Choose it. The toast reads `Saved Films.`, the Work page reads `Its Profile has no Targets, so this Movie wants nothing.`, and the file is gone from the media root.

  ```sh
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop click 'getByRole("table", { name: "Movies Profiles" }).getByRole("link", { name: "Films", exact: true })'
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop click 'getByRole("button", { name: "Remove 1080p" })'
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop click 'getByRole("button", { name: "Remove HD" })'
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop click 'getByRole("button", { name: "Save", exact: true })'
  $FW exec $RUN --proof targets-and-profiles/impact-delete -- $PW cli -s={runId}-admin-desktop snapshot --filename={runDir}/proof/targets-and-profiles/impact-delete/impact.aria.yml
  $FW exec $RUN --proof targets-and-profiles/impact-delete -- $PW cli -s={runId}-admin-desktop screenshot --filename={runDir}/proof/targets-and-profiles/impact-delete/impact.png
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop click 'getByRole("button", { name: "Save and delete 1 file" })'
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop goto "{nodes.home.browserUrl}/works/$WORK"
  $FW exec $RUN --proof targets-and-profiles/impact-delete -- $PW cli -s={runId}-admin-desktop snapshot --filename={runDir}/proof/targets-and-profiles/impact-delete/work.aria.yml
  $FW exec $RUN --proof targets-and-profiles/impact-delete -- curl -sS -H 'X-Api-Key: {apiKey}' "{nodes.home.url}/api/v1/works/$WORK/cells"
  $FW exec $RUN --proof targets-and-profiles/impact-delete -- $FW fs stat {runDir} '{mediaDir}/movies/The Quiet Orchard (2021)/The Quiet Orchard (2021) - 1080p.mkv'
  ```

  The last `fs stat` exits non-zero, since no file is left at that path.

Steps on `sizes`:

- **Sizes.** Add The Quiet Orchard on `Films` and choose `Search`. The Searches list reads `Grabbed a Release`; the searches view's `refused` lists `The.Quiet.Orchard.2021.1080p.WEB-DL.H264-BLOAT` with `size_above` and `The.Quiet.Orchard.2021.1080p.WEB-DL.H264-TINY` with `size_below`, both at `Sized`, and the file lands as `The Quiet Orchard (2021) - Sized.mkv`. Add The Lantern Keeper the same way and search it: its search grabs with `notes` holding `size_not_checked`.

  ```sh
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop goto '{nodes.home.browserUrl}/add?contentType=movie&query=quiet+orchard'
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop click 'getByRole("button", { name: "Add The Quiet Orchard" })'
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop click 'getByRole("dialog", { name: "Add The Quiet Orchard" }).getByRole("button", { name: "Add Movie" })'
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop click 'getByRole("banner").getByRole("button", { name: "Search", exact: true })'
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop run-code 'async page => { await page.getByRole("list", { name: "Searches" }).getByText("Grabbed a Release").waitFor({ timeout: 30000 }); }'
  ORCHARD=$($FW exec $RUN -- curl -sS -H 'X-Api-Key: {apiKey}' '{nodes.home.url}/api/v1/works' | jq -r '.works[] | select(.label == "The Quiet Orchard") | .id')
  $FW exec $RUN --proof targets-and-profiles/sizes-refusals -- curl -sS -H 'X-Api-Key: {apiKey}' "{nodes.home.url}/api/v1/works/$ORCHARD/searches"
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop run-code 'async page => { await page.getByRole("list", { name: "Files" }).getByText("The Quiet Orchard (2021) - Sized.mkv").waitFor({ timeout: 90000 }); }'
  $FW exec $RUN --proof targets-and-profiles/sizes-refusals -- $FW fs match {runDir} '{mediaDir}/movies/The Quiet Orchard (2021)/The Quiet Orchard (2021) - Sized.mkv' --release The.Quiet.Orchard.2021.1080p.WEB-DL.H264-FKW
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop goto '{nodes.home.browserUrl}/add?contentType=movie&query=lantern+keeper'
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop click 'getByRole("button", { name: "Add The Lantern Keeper" })'
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop click 'getByRole("dialog", { name: "Add The Lantern Keeper" }).getByRole("button", { name: "Add Movie" })'
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop click 'getByRole("banner").getByRole("button", { name: "Search", exact: true })'
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop run-code 'async page => { await page.getByRole("list", { name: "Searches" }).getByText("Grabbed a Release").waitFor({ timeout: 30000 }); }'
  KEEPER=$($FW exec $RUN -- curl -sS -H 'X-Api-Key: {apiKey}' '{nodes.home.url}/api/v1/works' | jq -r '.works[] | select(.label == "The Lantern Keeper") | .id')
  $FW exec $RUN --proof targets-and-profiles/sizes-refusals -- curl -sS -H 'X-Api-Key: {apiKey}' "{nodes.home.url}/api/v1/works/$KEEPER/searches"
  ```

- **A codec the probe reads.** Add The Copper Bell, picking `Encodes` in the sheet's `Profile`, and search it. The search grabs, but the Grab ends `failed` at `probing` with `fills_nothing` and `a WEBDL-1080p file fills no Target the Work still wants`. The `Encoded` cell reads `Failed`, `Needs you` lists the Grab, and no file lands.

  ```sh
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop goto '{nodes.home.browserUrl}/add?contentType=movie&query=copper+bell'
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop click 'getByRole("button", { name: "Add The Copper Bell" })'
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop click 'getByRole("dialog", { name: "Add The Copper Bell" }).getByRole("combobox", { name: "Profile" })'
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop click 'getByRole("option", { name: "Encodes" })'
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop click 'getByRole("dialog", { name: "Add The Copper Bell" }).getByRole("button", { name: "Add Movie" })'
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop click 'getByRole("banner").getByRole("button", { name: "Search", exact: true })'
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop run-code 'async page => { await page.getByRole("table", { name: "Any cut cells" }).getByText("Failed").first().waitFor({ timeout: 60000 }); }'
  $FW exec $RUN --proof targets-and-profiles/sizes-probed-codec -- $PW cli -s={runId}-admin-desktop screenshot --full-page --filename={runDir}/proof/targets-and-profiles/sizes-probed-codec/work.png
  $FW exec $RUN --proof targets-and-profiles/sizes-probed-codec -- curl -sS -H 'X-Api-Key: {apiKey}' '{nodes.home.url}/api/v1/grabs'
  ```

Steps on a second fresh `targets` run:

- **Add a cut.** Open `/settings/editions`. The `Cuts` table's `Standard` row reads `Any title that names no cut`, and `Extended` reads `Built in`. Choose `Add cut`, fill `Name` with `Superfan` and `Also read as` with `Fan Cut`, and choose `Add cut` in the dialog. The toast reads `Added Superfan. Profiles and Works can list it now.`, and the row reads `Superfan, Fan Cut` and `Custom`.

  ```sh
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop goto '{nodes.home.browserUrl}/settings/editions'
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop click 'getByRole("button", { name: "Add cut" })'
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop fill 'getByRole("dialog", { name: "Add a cut" }).getByRole("textbox", { name: "Name" })' Superfan
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop fill 'getByRole("dialog", { name: "Add a cut" }).getByRole("textbox", { name: "Also read as" })' "Fan Cut"
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop click 'getByRole("dialog", { name: "Add a cut" }).getByRole("button", { name: "Add cut" })'
  $FW exec $RUN --proof targets-and-profiles/editions-cuts -- $PW cli -s={runId}-admin-desktop snapshot --filename={runDir}/proof/targets-and-profiles/editions-cuts/cuts.aria.yml
  $FW exec $RUN --proof targets-and-profiles/editions-cuts -- curl -sS -H 'X-Api-Key: {apiKey}' '{nodes.home.url}/api/v1/cuts'
  ```

- **List it and grab it.** Open `Films`, choose `Choose cuts`, `Add cut`, `Superfan`, then `Remove Standard`, and `Save`. The toast reads `Saved Films.` List the Superfan release, add The Quiet Orchard and search it. Once it imports, `Files` lists `The Quiet Orchard (2021) - {edition-Superfan} 1080p.mkv`.

  ```sh
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop goto '{nodes.home.browserUrl}/settings/profiles'
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop click 'getByRole("table", { name: "Movies Profiles" }).getByRole("link", { name: "Films", exact: true })'
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop click 'getByRole("radio", { name: "Choose cuts" })'
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop click 'getByRole("button", { name: "Add cut" })'
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop click 'getByRole("menuitem", { name: "Superfan", exact: true })'
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop click 'getByRole("button", { name: "Remove Standard" })'
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop click 'getByRole("button", { name: "Save", exact: true })'
  $FW cue $RUN fixture-site-list '{"release": "The.Quiet.Orchard.2021.Superfan.1080p.WEB-DL.H264-FKW"}'
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop goto '{nodes.home.browserUrl}/add?contentType=movie&query=quiet+orchard'
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop click 'getByRole("button", { name: "Add The Quiet Orchard" })'
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop click 'getByRole("dialog", { name: "Add The Quiet Orchard" }).getByRole("button", { name: "Add Movie" })'
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop click 'getByRole("banner").getByRole("button", { name: "Search", exact: true })'
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop run-code 'async page => { await page.getByRole("list", { name: "Files" }).getByText("{{edition-Superfan}", { exact: false }).waitFor({ timeout: 60000 }); }'
  ```

- **Rename it through the Impact.** On `/settings/editions` choose `Actions for Superfan`, then `Edit`, set `Name` to `Superfan Edition` and choose `Save`. The Impact `Save Superfan` lists `1 Work changes.` and `1 file is renamed.` with no Delete or Keep. Choose `Save`. The toast reads `Saved Superfan Edition.`, and the file is now `The Quiet Orchard (2021) - {edition-Superfan Edition} 1080p.mkv`.

  ```sh
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop goto '{nodes.home.browserUrl}/settings/editions'
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop click 'getByRole("button", { name: "Actions for Superfan" })'
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop click 'getByRole("menuitem", { name: "Edit" })'
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop fill 'getByRole("dialog", { name: "Edit Superfan" }).getByRole("textbox", { name: "Name" })' "Superfan Edition"
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop click 'getByRole("dialog", { name: "Edit Superfan" }).getByRole("button", { name: "Save" })'
  $FW exec $RUN --proof targets-and-profiles/editions-cut-rename -- $PW cli -s={runId}-admin-desktop snapshot --filename={runDir}/proof/targets-and-profiles/editions-cut-rename/impact.aria.yml
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop click 'getByRole("dialog", { name: "Save Superfan" }).getByRole("button", { name: "Save", exact: true })'
  $FW exec $RUN --proof targets-and-profiles/editions-cut-rename -- $FW fs match {runDir} '{mediaDir}/movies/The Quiet Orchard (2021)/The Quiet Orchard (2021) - {{edition-Superfan Edition} 1080p.mkv' --release The.Quiet.Orchard.2021.Superfan.1080p.WEB-DL.H264-FKW
  ```

- **A held cut stays.** Choose `Actions for Superfan Edition`, then `Delete`, then `Delete` in `Delete Superfan Edition?`. The dialog reads `Superfan Edition can't be deleted. Films and 1 file use it.`, and the API answers 409 `cut_in_use` with the same detail.

  ```sh
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop click 'getByRole("button", { name: "Actions for Superfan Edition" })'
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop click 'getByRole("menuitem", { name: "Delete" })'
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop click 'getByRole("dialog", { name: "Delete Superfan Edition?" }).getByRole("button", { name: "Delete" })'
  $FW exec $RUN --proof targets-and-profiles/editions-cut-in-use -- $PW cli -s={runId}-admin-desktop screenshot --filename={runDir}/proof/targets-and-profiles/editions-cut-in-use/refused.png
  $FW exec $RUN --proof targets-and-profiles/editions-cut-in-use -- curl -sS -i -X DELETE -H 'X-Api-Key: {apiKey}' '{nodes.home.url}/api/v1/cuts/superfan'
  ```

- **Phone.** On the first `targets` run, open `{runId}-admin-phone` as in the skill's Drive with `phone`. Choose `More`, then `Settings`, `Targets` in the `Settings` list and `2160p`. Screenshot the Qualities list, then choose `Actions for WEBRip-2160p`: the sheet holds `Move up`, `Move down` and `Untie from the one above`, since WEBRip-2160p is tied with HDTV-2160p.

  ```sh
  $FW exec $RUN -- $PW cli -s={runId}-admin-phone click 'getByRole("button", { name: "More" })'
  $FW exec $RUN -- $PW cli -s={runId}-admin-phone click 'getByRole("button", { name: "Settings" })'
  $FW exec $RUN -- $PW cli -s={runId}-admin-phone click 'getByRole("list", { name: "Settings" }).getByRole("link", { name: "Targets" })'
  $FW exec $RUN -- $PW cli -s={runId}-admin-phone click 'getByRole("link", { name: "2160p" })'
  $FW exec $RUN --proof targets-and-profiles/targets-editor -- $PW cli -s={runId}-admin-phone screenshot --filename={runDir}/proof/targets-and-profiles/targets-editor/phone-editor.png
  $FW exec $RUN -- $PW cli -s={runId}-admin-phone click 'getByRole("button", { name: "Actions for WEBRip-2160p" })'
  $FW exec $RUN --proof targets-and-profiles/targets-editor -- $PW cli -s={runId}-admin-phone snapshot --filename={runDir}/proof/targets-and-profiles/targets-editor/phone-menu.aria.yml
  $FW exec $RUN --proof targets-and-profiles/targets-editor -- $PW cli -s={runId}-admin-phone screenshot --filename={runDir}/proof/targets-and-profiles/targets-editor/phone.png
  ```

## Gotchas

- `fakeworld exec` reads `{word}` as a manifest token, so a cut's tag in a path or a locator is written `{{edition-Superfan}`, which reaches the command as `{edition-Superfan}`.
- A cut's key is fixed when it is made, so `Superfan Edition` is still `superfan` on the API's paths.
- The Work page's Searches list shows each search's outcome but not its refusals. Read the reasons from `GET /api/v1/works/{workId}/searches`, whose `refused` lists each refused release with a `code` and `targetId` per Target, and whose `notes` hold the grabbed release's `size_not_checked`.
- No page or API list shows Kept files yet. The Work page's Files come from the cells, and a Kept file fills none. Prove one on disk with `fs match` or `fs stat`.
- Every scenario Profile wants Any cut, so a Work's cells sit under one heading, `Any cut`, in the table `Any cut cells`. A named list makes one table per wanted cut.
- `DELETE /api/v1/profiles/{profileId}` without `moveTo` answers 422 `invalid` before any Profile check.
- In `Any cut cells` the `Status` column and each Target's column read the same state, so `getByText("Failed")` or `getByText("Met")` matches more than one cell; add `.first()`.
- The second `Delete UHD` is the Impact's button, in the dialog named `Delete UHD` without the question mark.
- The payloads are a few seconds of video, so the Profile editor's estimates are tiny. Read their wording, not their numbers.
- The add sheet keeps `Monitor` on; an unmonitored Work has no Wants and its Search finds nothing to do.
- Known gaps a run will meet. These are not recipe failures:
  - A file SlipStream doesn't know, sitting at the path a rename wants, blocks that rename. The Node logs it and retries, and no page says so.
  - A cell whose upgrade Grab is downloading still reads `Upgradable`, not `Downloading`.
  - When two files of a Work would take one name as an import commits, the Grab retries instead of failing.
  - A save refused with 409 `impact_changed` carries no Impact. The web app previews again to show the new one.
  - A Grab under Any cut that is still downloading when the Work's list switches to named cuts leaves the new cell reading `Missing` until it imports.
  - Nothing reads HDR, the audio codec, channels or audio languages from a title or a probe yet, so a filter on them passes every release and file. Only the video codec is judged, at the probe, which is why `Sized`'s TrueHD rule lets every `sizes` release through.
  - The `fills_nothing` message doesn't name the filter that refused the file.
  - A film title that holds a cut's words, such as Uncut Gems or The Final Cut, reads as that cut.
