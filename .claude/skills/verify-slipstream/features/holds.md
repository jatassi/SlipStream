# Holds

A Target can hold an automatic grab for a set time after a Release is published, so a better Release posted soon after wins. While a cell waits, the hero reads `Waiting for a better release until …` with `Grab now`, the cell says how many Releases it has so far and when it grabs, and a season header reads `Waiting`. A wait survives a restart of the Controller. The Admin's own Search never waits, and `Grab now` grabs the best Release the wait holds at once.

## Sub-features

- `hold-wait` on `hold`, adding The Quiet Orchard with `Search after adding` reads `Waiting for a better release until <time>` with `Grab now`, the cell reads `1 release found so far. SlipStream grabs the best one <time> unless a better one arrives.`, and the API's `waits` lists the cell with `candidates` 1 and its `until`.
- `hold-restart` after `fakeworld node restart home`, which runs `slipstream restart` on its data dir, the wait reads the same `until`, and the hero still reads `Waiting for a better release until …`.
- `hold-grab-now` the hero's `Grab now` toasts `Grabbing 1 release` and the hero reads `Grabbed 1 release`, with the WEB-DL through to `imported`.
- `hold-press-skips` adding The Lantern Keepers waits too, its season header reading `Waiting`, and `Search Season 1` grabs the season pack at once: the hero reads `Grabbed 1 release`.

## How to get to it (user POV)

- Desktop shell: Settings > Targets, a Target's `Hold`; a Work's hero line and its `Grab now`; a cell's popover; a season header's `Waiting` and its menu's `Grab now`.
- Phone shell: the same hero line and `Grab now`; the season row's `Waiting`; the episode sheet.
- API: `GET /api/v1/works/{workId}/searching` (`waits [{cell, until, candidates}]`), `POST /api/v1/works/{workId}/waits/grab`, `GET /api/v1/grabs`.
- Host CLI: `slipstream restart --data-dir <dir>`, through `fakeworld node restart`.

## Driving it with fakeworld and playwright cli

Preconditions:

- A `hold` run launched and doctor passing, with `RUN`, `FW` and `PW` set as in the skill's Launch, and `{runId}-admin-desktop` signed in. Its 1080p Target holds 2 hours, and both Releases sit below the Bluray cutoff.

Steps:

- **Add a movie that waits.**

  ```sh
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop goto '{nodes.home.browserUrl}/add?contentType=movie&query=quiet+orchard'
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop click 'getByRole("button", { name: "Add The Quiet Orchard" })'
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop click 'getByRole("dialog", { name: "Add The Quiet Orchard" }).getByRole("button", { name: "Add Movie" })'
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop run-code 'async page => { await page.getByRole("status", { name: "Search" }).getByText(/^Waiting for a better release until /).waitFor({ timeout: 30000 }); }'
  $FW exec $RUN --proof holds/hold-wait -- $PW cli -s={runId}-admin-desktop snapshot --filename={runDir}/proof/holds/hold-wait/work.aria.yml
  $FW exec $RUN --proof holds/hold-wait -- $PW cli -s={runId}-admin-desktop screenshot --filename={runDir}/proof/holds/hold-wait/work.png
  ```

  Read the Work's id from the page's address and `GET /api/v1/works/<id>/searching` with `--proof holds/hold-wait`.

- **Restart the Controller.** The wait keeps its `until`.

  ```sh
  $FW node restart $RUN home
  $FW doctor $RUN
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop reload
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop run-code 'async page => { await page.getByRole("status", { name: "Search" }).getByText(/^Waiting for a better release until /).waitFor({ timeout: 30000 }); }'
  $FW exec $RUN --proof holds/hold-restart -- $PW cli -s={runId}-admin-desktop screenshot --filename={runDir}/proof/holds/hold-restart/work.png
  ```

  `GET /api/v1/works/<id>/searching` with `--proof holds/hold-restart` reads the same wait.

- **Grab now.**

  ```sh
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop click 'getByRole("status", { name: "Search" }).getByRole("button", { name: "Grab now" })'
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop run-code 'async page => { await page.getByRole("status", { name: "Search" }).getByText("Grabbed 1 release").waitFor({ timeout: 30000 }); }'
  $FW exec $RUN --proof holds/hold-grab-now -- $PW cli -s={runId}-admin-desktop screenshot --filename={runDir}/proof/holds/hold-grab-now/work.png
  $FW exec $RUN --proof holds/hold-grab-now -- curl -sS -H 'X-Api-Key: {apiKey}' '{nodes.home.url}/api/v1/grabs'
  ```

- **A press never waits.** Add The Lantern Keepers with its defaults; Season 1 reads `Waiting`. Choose `Search Season 1`.

  ```sh
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop goto '{nodes.home.browserUrl}/add?contentType=series&query=lantern'
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop click 'getByRole("button", { name: "Add The Lantern Keepers" })'
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop click 'getByRole("dialog", { name: "Add The Lantern Keepers" }).getByRole("button", { name: "Add Series" })'
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop run-code 'async page => { await page.getByRole("heading", { name: /^Season 1/, level: 3 }).getByText("Waiting").first().waitFor({ timeout: 30000 }); }'
  $FW exec $RUN --proof holds/hold-press-skips -- $PW cli -s={runId}-admin-desktop screenshot --filename={runDir}/proof/holds/hold-press-skips/waiting.png
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop click 'getByRole("button", { name: "Search Season 1" })'
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop run-code 'async page => { await page.getByRole("status", { name: "Search" }).getByText(/^Grabbed/).waitFor({ timeout: 30000 }); }'
  $FW exec $RUN --proof holds/hold-press-skips -- $PW cli -s={runId}-admin-desktop screenshot --filename={runDir}/proof/holds/hold-press-skips/grabbed.png
  $FW exec $RUN --proof holds/hold-press-skips -- curl -sS -H 'X-Api-Key: {apiKey}' '{nodes.home.url}/api/v1/grabs'
  ```

- **Phone.** Open `{runId}-admin-phone` on The Lantern Keepers while Season 1 waits and screenshot it into `proof/holds/hold-press-skips/waiting-phone.png`: the hero reads `Waiting for a better release until …` above `Grab now`.

## Gotchas

- The Scenario's Releases are dated 2099, which SlipStream clamps to the moment it hears them, so each wait ends 2 hours after the search, not after a publish time.
- `Search after adding` is remembered per Content Type. A run that turned it off for movies keeps it off for the next add.
- The hero line is `getByRole("status", { name: "Search" })`; its `Grab now` sits inside it, and the season menu's `Grab now` is a menu item on desktop and a button on phone.
