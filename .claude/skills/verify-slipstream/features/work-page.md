# Work page

One Work on one scroll. The hero sets the Work's backdrop behind its poster, its title treatment as the heading, the facts line with its first rating, pills for what its cells add up to, its overview, and its actions: Search, Editions, Monitored and the `More actions` menu with Edit, Refresh metadata, Rename files and Delete. Needs you lists what only an Admin can resolve. The Edition by Target matrix shows each wanted Edition's cells with their files, progress or failure, then Files and Kept files, About with the fields its type places there, and its Grabs and searches.

## Sub-features

- `work-hero` The Salt Road's page on `movies` shows the backdrop, the poster and the title treatment as an image inside the `h1` named `The Salt Road`, the facts line `2022 · Movie · 1h 52m · PG-13 · Drama, Adventure` with its first rating, `IMDb 7.1`, the `Cells` pills, and the overview clamped to three lines with `More` (P-045).
- `work-hero-steady` while the title treatment is still loading, its place in the hero already has its full height, so `More actions for <title>` is at the same spot before and after the image arrives.
- `work-needs-you` a Work with no library root reads `No library root for this Work` with `Open Library roots` under `Needs you`; a failed download reads one compact failure card named for the cell that failed, with `Search again`, and the pills read `failed`.
- `work-needs-you-client-away` while a Grab's Torrent is in an external client that isn't responding, refused the sign-in or needs attention, and its Node is in step, `needs` holds `{kind: "waiting", grabId, code: "client_away"}`; an out-of-step Node reads `node_away` instead. Needs you shows it as `Waiting for Transmission on seedbox` with `Its download carries on once Transmission responds again.` and `Send to another client`, which opens `Send to another client`: every other client that can take it (`Built-in client on home`, each on, responding external as `qBittorrent on home`), and `Send` posts `{builtInOn}` or `{clientId}` to `POST /api/v1/grabs/{grabId}/send-to`, toasting `Sent <release> to the built-in client on home`. A client on the same Node that already has the torrent shows `grab_torrent_in_client`'s detail in the sheet.
- `work-needs-you-routed` on a `seeding-routing` run, a private Tracker's Grab whose route names a client on a Node out of step waits at fetching, whether or not the Indexer's own Node is in step: `needs` holds `{kind: "waiting", grabId, code: "node_away"}` and the Grab carries `waitsFor` `{nodeId, client}`. Needs you shows `Waiting to send to the built-in client on <node>` (or `<client> on <node>`) with `<node> is offline. This download goes there once <node> is back online.` and `Send to another client`, whose sheet leaves out the client the Grab waits for. A send fetches and downloads it on the picked client, and the row leaves.
- `work-matrix` the table `Editions and Targets` has one column per Target of the Work's Profile with its sourcing, `Downloaded`, and one row per wanted Edition reading where its list comes from, `From Profile <name>` or `Set for this Work`; each cell has its state chip and, while it is wanted, `Search for <Target>, <Edition>`; a downloading cell that holds a file reads `Replaces <quality>` and a failed one shows its cause and the file it kept. On phone it is one card per Edition (P-208).
- `work-editions` `Editions` (desktop) or `Edit Editions` (phone, in the actions sheet) opens `Editions of <title>` with the heading `Editions from Profile <name>`; `Choose cuts` and `Add cut` make `Editions set for this Work`, and `Save` opens the Impact `Save the Editions of <title>`. Saving toasts `Saved the Editions of <title>.`, the rows read `Set for this Work`, and a cut no release has shown yet sits in its own row reading `Not seen yet. It's wanted once a release of it is found.`
- `work-files` `Files` reads the Work's library root and folder, then each file that fills a cell with its fills, quality, size and dimensions, and `Kept files` apart, each with `Kept file` and `Fills no wanted Target`; with none it reads `No files yet. A Grab that imports puts its file here.` (P-048).
- `work-files-media` on `home` with the shelves added as roots ([Pending imports](./pending-imports.md)), Lanterns at Low Tide's file row reads `Lanterns at Low Tide (2016) {tmdb-9000101} · Fills 1080p` and `HDTV-1080p · 1920×1080 · 524 KB`, and its `Details` read `Video` `x265 · 1920×1080 · 10-bit · Dolby Vision profile 8, HDR10`, `Audio` `TrueHD · 5.1 · English` and `AAC · 2.0 · English · Commentary`, `Subtitles` `English` and `Duration` `1 s`. `GET /api/v1/works/{id}/files` answers each file's `media` record with its `codec`, `cut`, `label`, `kept`, `modifiedAt`, `goneAt` and `probeError`: Saltmarsh Radio `HDR10+` and `HDR10` with English and French E-AC-3 and a forced French subtitle, The Tin Signal's episodes `SDR`, `HDR10` and `HLG`, and Brass Meridian's `DV` profile 5 (P-029, P-052).
- `work-files-gone` after `fakeworld fs rm` of The Tin Signal's S01E01 and Scan now, `Gone files` lists `The Tin Signal - S01E01.mkv` with `Gone since <date>` and `Forget`, its cell reads missing, and `Forget The Tin Signal - S01E01.mkv` removes the row with no dialog, leaving `Files 2` and no Library Health item (P-026).
- `work-about` About shows the tagline, Certification `PG-13 (US)` with its `With` country, Studio `Fennel Films` beside the studio logo, Collection and Awards, one strip of photos per department of Cast and crew, each rating with its votes and fetch time, the TMDB page and trailer as links that open a new tab, the movie Item's `Counts as out on April 4, 2023, Digital release in United States.`, and `TMDB lists this movie as 9000005.` over `Details` (P-048, P-057).
- `work-edit` `Edit…` opens `Edit <title>` with `Profile` and `Library root`; a new Profile saves through the Impact `Give <title> the Profile <name>` and toasts `Changed the Profile of <title>.` (P-047).
- `work-refresh` `Refresh metadata` toasts `Refreshed the metadata of <title>.` and the page shows the new record; once TMDB drops the Work it toasts `TMDB no longer lists <title>, so it wasn't refreshed.`, the hero reads `Gone from TMDB`, `Refresh metadata is unavailable` and `TMDB no longer lists this movie. SlipStream keeps it as it is.` with `Delete`, the record stays, and `Refresh metadata` in the menu is disabled (P-047).
- `work-delete` `Delete…`, or the gone-upstream warning's `Delete`, opens `Delete <title>?`; deleting toasts once, `Deleted <title> and its files.`, or `Deleted <title>.` for a Work with no files, and the page goes back to its Library. A Work deleted elsewhere while its page is open toasts `<title> was deleted.` and leaves the same way (P-047).
- `work-page-many-items` a Work of a Content Type with many Items shows its levels and Items between Needs you and Files in place of the Edition by Target matrix, and its hero's Monitored switch becomes a Monitoring button; [series-work-page.md](series-work-page.md) and [series-monitoring.md](series-monitoring.md) drive them on `series`.

## How to get to it (user POV)

- Desktop shell: a Work's poster or row in the Library, a Work in the palette, or `/works/{id}`. The hero's `More actions for <title>` holds `Edit…`, `Refresh metadata`, `Rename files…` and `Delete…`, and `Rename files…` opens the Work's rename plan ([naming.md](naming.md)).
- Both shells: a gone file's `Forget` sits on its row in `Gone files`, and each file's `Details` hold its media info.
- Phone shell: the same Work, as a pushed page whose nav bar holds the title. `More actions for <title>` opens an action sheet with `Monitored`, `Edit Editions`, `Edit…`, `Refresh metadata`, `Rename files…` and `Delete…`.
- API: `GET /api/v1/works/{id}`, `GET /api/v1/works/{id}/cells` (the summary and Needs you), `GET /api/v1/works/{id}/files`, `GET` and `PUT /api/v1/works/{id}/wants` (`{"editions": ...}` or `{"editions": null}`, with `?preview=true`), `PUT /api/v1/works/profile`, `POST /api/v1/works/refresh` with one id, `POST /api/v1/works/delete`, `POST /api/v1/files/forget {fileIds}`, which answers 409 `files_not_gone` for a file that isn't gone.
- Host CLI: none.

## Driving it with fakeworld and playwright cli

Preconditions:

- A fresh `movies` run, launched as in the skill's Launch, doctor passing, with `{runId}-admin-desktop` and `{runId}-admin-phone` signed in. It has no library root, so every Work reads `No library root for this Work`.
- The failed download and a Work's own Editions are proved by the browser tier's `web/e2e/work-page.spec.ts` on `targets`; drive them by hand on a `targets` run as that spec does.

Steps:

- **Add The Salt Road and open it.** Set `W` to the reply's `work.id`. The page is first opened on phone with image-cdn slowed, because the title treatment is fetched on the first view and not when the Work is added. The first `run-code` reads `loading: true` and the box of `More actions` while the treatment is held. The second reads its box once the treatment has loaded, and the two boxes are equal.

  ```sh
  $FW fault set $RUN cdn-slow image-cdn latency=3s
  $FW exec $RUN -- curl -sS -X POST -H 'X-Api-Key: {apiKey}' -H 'Content-Type: application/json' -d '{"contentType":"movie","key":"9000005","preset":"all"}' '{nodes.home.url}/api/v1/works'
  $FW exec $RUN -- $PW cli -s={runId}-admin-phone goto "{nodes.home.browserUrl}/works/$W"
  $FW exec $RUN --proof work-page/work-hero-steady -- $PW cli -s={runId}-admin-phone run-code 'async page => { const more = page.getByRole("button", { name: "More actions for The Salt Road" }); await more.waitFor({ timeout: 10000 }); const loading = await page.locator("header :is(h1, p) > img").evaluate((img) => !img.complete || img.naturalWidth === 0); return { loading, box: await more.boundingBox() }; }'
  $FW exec $RUN --proof work-page/work-hero-steady -- $PW cli -s={runId}-admin-phone screenshot --filename={runDir}/proof/work-page/work-hero-steady/loading.png
  $FW fault clear $RUN cdn-slow
  $FW exec $RUN --proof work-page/work-hero-steady -- $PW cli -s={runId}-admin-phone run-code 'async page => { await page.waitForFunction(() => { const img = document.querySelector("header :is(h1, p) > img"); return img !== null && img.complete && img.naturalWidth > 0; }, undefined, { timeout: 30000 }); return { loading: false, box: await page.getByRole("button", { name: "More actions for The Salt Road" }).boundingBox() }; }'
  $FW exec $RUN --proof work-page/work-hero-steady -- $PW cli -s={runId}-admin-phone screenshot --filename={runDir}/proof/work-page/work-hero-steady/loaded.png
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop goto "{nodes.home.browserUrl}/works/$W"
  $FW exec $RUN --proof work-page/work-hero -- $PW cli -s={runId}-admin-desktop snapshot --filename={runDir}/proof/work-page/work-hero/desktop.aria.yml
  $FW exec $RUN --proof work-page/work-hero -- $PW cli -s={runId}-admin-desktop screenshot --full-page --filename={runDir}/proof/work-page/work-hero/desktop.png
  $FW exec $RUN -- $PW cli -s={runId}-admin-phone goto "{nodes.home.browserUrl}/works/$W"
  $FW exec $RUN --proof work-page/work-hero -- $PW cli -s={runId}-admin-phone screenshot --full-page --filename={runDir}/proof/work-page/work-hero/phone.png
  ```

- **Needs you, the matrix, Files and About.** Read them from the desktop snapshot above: `Needs you` with `No library root for this Work`, the `Editions and Targets` table, `No files yet.`, and About's values.

  ```sh
  $FW exec $RUN --proof work-page/work-about -- $PW cli -s={runId}-admin-desktop run-code 'async page => page.getByRole("region", { name: "About" }).ariaSnapshot()'
  $FW exec $RUN --proof work-page/work-about -- $PW cli -s={runId}-admin-desktop run-code 'async page => page.getByRole("list", { name: "Links" }).getByRole("link").evaluateAll((links) => links.map((a) => [a.textContent, a.href, a.target, a.rel]))'
  ```

- **Edit.** Open `Edit…`, read the dialog, and close it.

  ```sh
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop click 'getByRole("button", { name: "More actions for The Salt Road" })'
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop click 'getByRole("menuitem", { name: "Edit…" })'
  $FW exec $RUN --proof work-page/work-edit -- $PW cli -s={runId}-admin-desktop screenshot --filename={runDir}/proof/work-page/work-edit/edit.png
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop press Escape
  ```

- **Refresh, then gone.**

  ```sh
  $FW cue $RUN tmdb-change '{"id":9000005}'
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop click 'getByRole("button", { name: "More actions for The Salt Road" })'
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop click 'getByRole("menuitem", { name: "Refresh metadata" })'
  $FW exec $RUN --proof work-page/work-refresh -- $PW cli -s={runId}-admin-desktop screenshot --filename={runDir}/proof/work-page/work-refresh/refreshed.png
  $FW cue $RUN tmdb-delete '{"id":9000005}'
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop click 'getByRole("button", { name: "More actions for The Salt Road" })'
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop click 'getByRole("menuitem", { name: "Refresh metadata" })'
  $FW exec $RUN --proof work-page/work-refresh -- $PW cli -s={runId}-admin-desktop screenshot --full-page --filename={runDir}/proof/work-page/work-refresh/gone.png
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop click 'getByRole("button", { name: "More actions for The Salt Road" })'
  $FW exec $RUN --proof work-page/work-refresh -- $PW cli -s={runId}-admin-desktop snapshot --filename={runDir}/proof/work-page/work-refresh/menu.aria.yml
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop press Escape
  ```

- **Delete.** Choose the warning's `Delete`, then the dialog's `Delete`. The page goes to `/library/movie` with one toast. Then add The Quiet Orchard, open its page, set `Q` to its id and delete it over the API: the page toasts `The Quiet Orchard was deleted.` and goes to `/library/movie`.

  ```sh
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop click 'getByRole("button", { name: "Delete", exact: true })'
  $FW exec $RUN --proof work-page/work-delete -- $PW cli -s={runId}-admin-desktop screenshot --filename={runDir}/proof/work-page/work-delete/dialog.png
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop click 'getByRole("dialog").getByRole("button", { name: "Delete", exact: true })'
  $FW exec $RUN --proof work-page/work-delete -- $PW cli -s={runId}-admin-desktop screenshot --filename={runDir}/proof/work-page/work-delete/after.png
  $FW exec $RUN --proof work-page/work-delete -- curl -sS -X POST -H 'X-Api-Key: {apiKey}' -H 'Content-Type: application/json' -d "{\"workIds\":[\"$Q\"],\"files\":\"leave\"}" '{nodes.home.url}/api/v1/works/delete'
  $FW exec $RUN --proof work-page/work-delete -- $PW cli -s={runId}-admin-desktop screenshot --filename={runDir}/proof/work-page/work-delete/elsewhere.png
  ```

- **Media info and Forget.** On a separate `home` run with the shelves added as roots, open Lanterns at Low Tide and unfold its file's `Details`. Then remove The Tin Signal's S01E01 on disk and Scan now as in [Library roots](./library-roots.md)' **Scan now finds a gone file**, open The Tin Signal and choose `Forget`.

  ```sh
  LANT=$($FW exec $RUN -- curl -sS -H 'X-Api-Key: {apiKey}' '{nodes.home.url}/api/v1/library' | jq -r '.rows[] | select(.label == "Lanterns at Low Tide") | .id')
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop goto "{nodes.home.browserUrl}/works/$LANT"
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop click 'getByRole("list", { name: "Files" }).getByRole("button", { name: "Details" })'
  $FW exec $RUN --proof work-page/work-files-media -- $PW cli -s={runId}-admin-desktop snapshot --filename={runDir}/proof/work-page/work-files-media/details.aria.yml
  $FW exec $RUN --proof work-page/work-files-media -- $PW cli -s={runId}-admin-desktop screenshot --filename={runDir}/proof/work-page/work-files-media/details.png
  $FW exec $RUN --proof work-page/work-files-media -- curl -sS -H 'X-Api-Key: {apiKey}' "{nodes.home.url}/api/v1/works/$LANT/files"
  TIN=$($FW exec $RUN -- curl -sS -H 'X-Api-Key: {apiKey}' '{nodes.home.url}/api/v1/library' | jq -r '.rows[] | select(.label == "The Tin Signal") | .id')
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop goto "{nodes.home.browserUrl}/works/$TIN"
  $FW exec $RUN --proof work-page/work-files-gone -- $PW cli -s={runId}-admin-desktop snapshot --filename={runDir}/proof/work-page/work-files-gone/gone.aria.yml
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop click 'getByRole("button", { name: "Forget The Tin Signal - S01E01.mkv" })'
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop run-code 'async page => { await page.getByRole("list", { name: "Gone files" }).waitFor({ state: "detached" }); }'
  $FW exec $RUN --proof work-page/work-files-gone -- curl -sS -H 'X-Api-Key: {apiKey}' "{nodes.home.url}/api/v1/works/$TIN/files"
  ```

## Gotchas

- On phone the nav bar holds the page's `h1`, so the hero's title is not a heading there: find the Work by `getByRole("heading", { name: "<title>", exact: true })`, which matches one element in either shell.
- The hero's backdrop shows only once the ambient has sampled it, so a screenshot taken at once can lack it. Wait for the `img` inside the backdrop before capturing.
- `Search` is exact: each cell's own button is named `Search for <Target>, <Edition>`, and the upgrades list has its own `Search` per row inside `Needs you`.
- `Refresh metadata` while gone upstream is disabled, not hidden; the warning beside the actions says why.
- `Files`' `Folder` reads the first listed file's folder, so a series with season folders reads one season's folder, such as `The Tin Signal [tvdbid-9100201]/Season 02`.
- `Gone since` reads the day in the browser's locale, such as `Nov 8` for `en-US` or `8 Nov` for `en-GB`.
