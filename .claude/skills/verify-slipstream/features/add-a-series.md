# Add a series

An Admin finds a series on TVDB from the add page or the palette, by its title or any alternative title, and adds it as a Work. The add sheet opens on the series with its facts, its season and episode counts and one sentence saying what Add will do. Below it sit the Profile, which episodes to monitor (each preset with its count), Include specials, Season folders and Remember these choices. When TVDB search fails, results come from TMDB, and one TVDB doesn't list says so and offers no Add.

## Sub-features

- `series-search` `/add?contentType=series` searches fake-tvdb: `Nights at the Harbor`, an alternative title, lists the one result `Harbor Nights` reading `2015 · Harbor TV` and its overview, with `Add Harbor Nights` (P-055, P-042).
- `series-sheet` `Add The Lantern Keepers` opens the dialog of that name reading `3 seasons · 9 episodes` and `Adds The Lantern Keepers (2020) to TV on <the TV root>, monitoring all 9 episodes and new ones as they air, wanting TV's 1 Target.` `Episodes to monitor` lists `All · 9 episodes`, `Future · 0 episodes`, `Missing · 9 episodes`, `Existing · 0 episodes`, `First season · 3 episodes`, `Latest season · 3 episodes` and `None · 0 episodes` (P-042, P-051).
- `series-presets` each preset rephrases the sentence: Future `monitoring only new episodes as they air`, Existing and None `monitoring no episodes`, First season `monitoring 3 episodes and no new ones`, Latest season `monitoring 3 episodes and new ones as they air`; `Include specials`, off by default, adds the two specials, and `Season folders` starts on (P-042).
- `series-add` `Add Series` with Latest season, Include specials on and Season folders off toasts `Added The Lantern Keepers`, opens its Work page on `Monitoring 5 of 11 episodes`, and `GET /api/v1/works/{id}` reads `monitorNew` true, `monitorAsides` true and `levelFolders` false (P-042, P-287).
- `series-remember` with Remember on, the next sheet, `Add Harbor Nights`, opens on `Latest season · 2 episodes` with Include specials on and Season folders off, and a search for `lantern` shows `The Lantern Keepers … In the library` as a link (P-042).
- `series-fallback` while fake-tvdb answers 500 on search, `/add?contentType=series&query=lights` lists TMDB's `Harbor Lights` with `TVDB doesn't list this series. SlipStream needs TVDB to keep it up to date.` and no Add (P-055).

## How to get to it (user POV)

- Desktop shell: the TV library's `Add a Series` in the top bar or its empty state, or the Library's `Add` menu's `Add a Series`. Both open `/add?contentType=series`.
- Phone shell: the `Library` tab's `Add` action sheet, or the TV library's empty state. The add sheet is a bottom sheet.
- Both shells: the palette's `Add a Series` command and its group of TVDB results; first setup's done screen has an `Add a Series` button.
- API: `GET /api/v1/content-types/series/catalog?query=`, `GET /api/v1/content-types/series/catalog/{key}`, `POST /api/v1/works {contentType: "series", key, preset, includeAsides, levelFolders, remember}`.

## Driving it with fakeworld, curl and playwright cli

Preconditions:

- A `series` run, doctor passing, `{runId}-admin-desktop` signed in. The run's TV library is empty, its TV root is `{mediaDir}/tv` and the world sets an Admin TVDB key.

Steps:

- **Search by an alternative title.** The Results list holds `Harbor Nights` alone.

  ```sh
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop goto '{nodes.home.browserUrl}/add?contentType=series&query=Nights+at+the+Harbor'
  $FW exec $RUN --proof add-a-series/series-search -- $PW cli -s={runId}-admin-desktop snapshot --filename={runDir}/proof/add-a-series/series-search/results.aria.yml
  ```

- **Open the sheet and pick choices.** Search `lantern`, choose `Add The Lantern Keepers`, read the sentence and the preset list, then pick `Latest season`, turn `Include specials` on and `Season folders` off.

  ```sh
  SHEET='getByRole("dialog", { name: "Add The Lantern Keepers" })'
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop goto '{nodes.home.browserUrl}/add?contentType=series&query=lantern'
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop click 'getByRole("button", { name: "Add The Lantern Keepers" })'
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop click "$SHEET"'.getByRole("combobox", { name: "Episodes to monitor" })'
  $FW exec $RUN --proof add-a-series/series-sheet -- $PW cli -s={runId}-admin-desktop snapshot --filename={runDir}/proof/add-a-series/series-sheet/sheet.aria.yml
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop click 'getByRole("option", { name: /^Latest season/ })'
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop click "$SHEET"'.getByRole("switch", { name: "Include specials" })'
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop click "$SHEET"'.getByRole("switch", { name: "Season folders" })'
  $FW exec $RUN --proof add-a-series/series-presets -- $PW cli -s={runId}-admin-desktop snapshot --filename={runDir}/proof/add-a-series/series-presets/chosen.aria.yml
  $FW exec $RUN --proof add-a-series/series-presets -- $PW cli -s={runId}-admin-desktop screenshot --filename={runDir}/proof/add-a-series/series-presets/chosen.png
  ```

- **Add it.** `Add Series` lands on the Work page reading `Monitoring 5 of 11 episodes`; the API reads the stored switches.

  ```sh
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop click "$SHEET"'.getByRole("button", { name: "Add Series" })'
  $FW exec $RUN --proof add-a-series/series-add -- $PW cli -s={runId}-admin-desktop snapshot --filename={runDir}/proof/add-a-series/series-add/work.aria.yml
  $FW exec $RUN --proof add-a-series/series-add -- curl -sS -H 'X-Api-Key: {apiKey}' '{nodes.home.url}/api/v1/library?contentType=series'
  ```

- **Remember.** The next sheet opens on the remembered choices.

  ```sh
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop goto '{nodes.home.browserUrl}/add?contentType=series&query=harbor'
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop click 'getByRole("button", { name: "Add Harbor Nights" })'
  $FW exec $RUN --proof add-a-series/series-remember -- $PW cli -s={runId}-admin-desktop snapshot --filename={runDir}/proof/add-a-series/series-remember/next.aria.yml
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop press Escape
  ```

- **TMDB fallback.** Make fake-tvdb's search fail, search `lights`, then clear the fault.

  ```sh
  $FW fault set $RUN tvdb-search tvdb status=500 --route search
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop goto '{nodes.home.browserUrl}/add?contentType=series&query=lights'
  $FW exec $RUN --proof add-a-series/series-fallback -- $PW cli -s={runId}-admin-desktop snapshot --filename={runDir}/proof/add-a-series/series-fallback/results.aria.yml
  $FW fault clear $RUN tvdb-search
  ```

## Gotchas

- The sentence's root is the run's scratch path, so match it by its `tv` ending, never in full.
- The preset list is a Select: its options exist only while it is open, and `Escape` closes it without picking.
- The fallback resolves each TMDB result through TMDB's `external_ids`; the series Scenario's TMDB lists only `Harbor Lights` among series, so a fallback search for another title lists nothing.
- SlipStream keeps a search's answer for a while, so a query already searched before the fault reads TVDB's cached results. Search a query the run hasn't searched yet.
- On phone, the empty TV library holds two `Add a Series` buttons, the large title's and the empty state's.
