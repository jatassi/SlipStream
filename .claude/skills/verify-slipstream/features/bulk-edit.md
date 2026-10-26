# Bulk edit

An Admin picks Works in the Library, by each poster's or row's checkbox, by a click once picking has started or `Library options` > `Select` was chosen, by Shift for a range, Space from the keyboard, or `Select all` for every Work the filters show. The `Picked Works` bar counts the picks, says how many the filters hide, and acts on every pick in one request that answers per Work: Monitor, Unmonitor, Change Profile through one Impact, Set library root, Refresh metadata and Delete with or without the files. Profile and library root belong to one Content Type, so they need picks of one type. Each action toasts what happened to which Works, naming the ones skipped and why.

## Sub-features

- `bulk-pick` the checkbox on a poster or row, a click while picking, Shift for a range and `Select all` pick Works; the bar reads `2 Movies picked` and `· 1 hidden by filters` when a filter hides a pick, `Done` ends picking, and an action acts on hidden picks too (P-039).
- `bulk-monitor` `Monitor` and `Unmonitor` answer per Work, such as `Monitored The Salt Road. The Quiet Orchard was already monitored.`, and `PUT /api/v1/works/monitored` stores it; a poster goes quiet when nothing of it is monitored (P-039, P-242).
- `bulk-profile` `Change Profile…` picks a Profile, `Next` opens one Impact for every pick, and saving answers `Changed the Profile of …` (P-039).
- `bulk-root` `Set library root…` moves the picks without files, and a Work with files stays, as the dialog and the toast say: `The Quiet Orchard has files, so it stays in its library root. Moving a Work with files arrives with Moves.`
- `bulk-refresh` `Refresh metadata` refreshes every pick and toasts per Work (P-038).
- `bulk-delete` `Delete…` deletes with the files by default, `Deleted … and their files.`, the files leaving the disk; `Leave files on disk` answers `Deleted …. Their files stay on disk.`, Works with no files answer `Deleted ….` whichever is chosen, and the Library empties without a reload (P-039, P-242).

## How to get to it (user POV)

- Desktop and phone shells: the Library at `/library` or `/library/movie`; the checkbox on each poster shows on hover, on focus and while picking, and on each table row always.
- API: `PUT /api/v1/works/monitored`, `PUT /api/v1/works/profile` (with `preview`), `PUT /api/v1/works/root`, `POST /api/v1/works/refresh`, `POST /api/v1/works/delete` (with `preview`), each taking `{"workIds": [...]}`.
- Host CLI: none.

## Driving it with fakeworld and playwright cli

Preconditions:

- A fresh `movies` run with [Library](./library.md)'s two Works added, monitored off, and `{runId}-admin-desktop` signed in.
- Two library roots and a UHD Profile, made through the API:

  ```sh
  M=$(jq -r .scratchDir $RUN/manifest.json)
  HOME_ID=$(jq -r '.nodes[] | select(.name == "home") | .nodeId' $RUN/manifest.json)
  mkdir -p "$M/movies" "$M/archive"
  $FW exec $RUN -- curl -sS -X POST -H 'X-Api-Key: {apiKey}' -H 'Content-Type: application/json' -d "{\"contentType\":\"movie\",\"nodeId\":\"$HOME_ID\",\"path\":\"$M/movies\"}" '{nodes.home.url}/api/v1/roots'
  $FW exec $RUN -- curl -sS -X POST -H 'X-Api-Key: {apiKey}' -H 'Content-Type: application/json' -d "{\"contentType\":\"movie\",\"nodeId\":\"$HOME_ID\",\"path\":\"$M/archive\"}" '{nodes.home.url}/api/v1/roots'
  ```

  then the 2160p Target and the UHD Profile, as `makeUHD` in `web/e2e/library.ts` posts them.
- `web/e2e/bulk-edit.spec.ts` proves the same on `movies`, and `web/e2e/bulk-edit-files.spec.ts` the Work with files on `targets`, whose catalog adds The Salt Road with no releases.

Steps:

- **Pick and Monitor.** Open the Movies Library, pick both posters and choose `Monitor`. The Quiet Orchard was added monitored, so the toast says so.

  ```sh
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop goto '{nodes.home.browserUrl}/library/movie'
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop click 'getByRole("checkbox", { name: "Pick The Quiet Orchard" })'
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop click 'getByRole("checkbox", { name: "Pick The Salt Road" })'
  $FW exec $RUN --proof bulk-edit/bulk-pick -- $PW cli -s={runId}-admin-desktop screenshot --filename={runDir}/proof/bulk-edit/bulk-pick/two-picked.png
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop click 'getByRole("toolbar", { name: "Picked Works" }).getByRole("button", { name: "Monitor", exact: true })'
  $FW exec $RUN --proof bulk-edit/bulk-monitor -- $PW cli -s={runId}-admin-desktop run-code 'async page => { await page.getByText("Monitored The Salt Road. The Quiet Orchard was already monitored.").waitFor({ timeout: 10000 }); }'
  $FW exec $RUN --proof bulk-edit/bulk-monitor -- curl -sS -H 'X-Api-Key: {apiKey}' '{nodes.home.url}/api/v1/library'
  ```

- **Hidden by the search.** Fill `Search Movies` with `salt`; the bar adds `· 1 hidden by filters`. Empty the box again.

  ```sh
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop fill 'getByRole("searchbox", { name: "Search Movies" })' "salt"
  $FW exec $RUN --proof bulk-edit/bulk-pick -- $PW cli -s={runId}-admin-desktop snapshot --filename={runDir}/proof/bulk-edit/bulk-pick/hidden.aria.yml
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop fill 'getByRole("searchbox", { name: "Search Movies" })' ""
  ```

- **Refresh.** Choose `Refresh metadata`; the toast names both Works.

  ```sh
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop click 'getByRole("toolbar", { name: "Picked Works" }).getByRole("button", { name: "Refresh metadata" })'
  $FW exec $RUN --proof bulk-edit/bulk-refresh -- $PW cli -s={runId}-admin-desktop screenshot --filename={runDir}/proof/bulk-edit/bulk-refresh/toast.png
  ```

- **Profile through one Impact.** Choose `Change Profile…`, pick `UHD`, `Next`, then the Impact's `Save`.

  ```sh
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop click 'getByRole("button", { name: "Change Profile…" })'
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop click 'getByRole("combobox", { name: "Profile" })'
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop click 'getByRole("option", { name: "UHD" })'
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop click 'getByRole("button", { name: "Next" })'
  $FW exec $RUN --proof bulk-edit/bulk-profile -- $PW cli -s={runId}-admin-desktop screenshot --filename={runDir}/proof/bulk-edit/bulk-profile/impact.png
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop click 'getByRole("dialog", { name: /the Profile UHD/ }).getByRole("button", { name: /^Save/ })'
  $FW exec $RUN --proof bulk-edit/bulk-profile -- curl -sS -H 'X-Api-Key: {apiKey}' '{nodes.home.url}/api/v1/library'
  ```

- **Library root.** Choose `Set library root…`, pick the archive root and `Set library root`. Both rows' `rootId` read the archive's id.

- **Delete leaving files.** Choose `Delete…`, `Leave files on disk`, then `Delete`. The Library reads `No Movies in the library yet.`

  ```sh
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop click 'getByRole("button", { name: "Delete…" })'
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop click 'getByRole("radio", { name: "Leave files on disk" })'
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop click 'getByRole("dialog").getByRole("button", { name: "Delete", exact: true })'
  $FW exec $RUN --proof bulk-edit/bulk-delete -- $PW cli -s={runId}-admin-desktop screenshot --filename={runDir}/proof/bulk-edit/bulk-delete/empty.png
  $FW exec $RUN --proof bulk-edit/bulk-delete -- curl -sS -H 'X-Api-Key: {apiKey}' '{nodes.home.url}/api/v1/library'
  ```

- **A Work with files.** On a fresh `targets` run, add The Quiet Orchard and search it until its WEB-DL is imported (as [Search and grab](./search-and-grab.md)), add The Salt Road, and make an archive root. Pick both, `Set library root…` to the archive: the dialog and the toast say The Quiet Orchard stays. `Delete…` keeps `Delete files from disk`; the file leaves `{mediaDir}`, which `fs match` proves.

## Gotchas

- `movies` has no library root until one is made, and its Works then sit in no root; a root change needs two roots to move between.
- The poster's checkbox is invisible until hover or focus, but it still takes a click; `click` on it works without a hover.
- While picking, a click on a poster picks it rather than opening it; `Done` brings links back.
