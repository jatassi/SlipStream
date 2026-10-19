# Library roots

An Admin keeps every place SlipStream stores files in Settings > Library roots. A Content Type has any number of roots on any Node that holds Storage, one of them its default, where new Works go. A root is added through a folder browser that reads the Node's own drives, mounts and folders and judges each folder in words before it can be used. The section lists each root's Node, free space, Works and bytes, and a Storage block shows every volume that holds a root, counted once per volume, with how full it is. A Grab into a root on a Worker that downloads imports there by hard link. Writing off a Storage Node keeps its roots, marked `Node written off`, and their files count as missing until the root is re-homed onto a live Node, which first checks every recorded file at the new place. A root that holds files can't be deleted, and deleting one never touches its folder.

## Sub-features

- `roots-section` `/settings/library-roots` groups roots by Content Type under `Movies`, each row reading its folder, `Default` on the default, its Node with its status dot, `<size> free` and `<n> Movies, <bytes>`, with `Actions for <path>` holding `Make default`, `Edit` and `Delete` (P-021, P-022).
- `roots-add-browser` `Add a library root` on seedbox, once it holds Storage, opens `Choose a folder on seedbox`, read by seedbox. Its data folder reads `Can't use this folder because it's inside SlipStream's data folder.` with `Use seedbox` disabled; a typed path goes there, `New folder` makes `worker-movies` and opens it, the footer reads `This folder is on /, <size> free.`, and `Use worker-movies` fills the dialog's `Folder`. With `Make this the default for Movies` ticked, `Add` toasts `Added <path>. New Movies go there.` (P-027, P-021, P-022).
- `roots-import-worker` The Quiet Orchard, added and searched with the seedbox root as the default, downloads on seedbox and imports there with no Transfer: its Grab's history runs fetching, downloading, probing, importing, imported, and the placed file under `worker-movies` matches the release with 2 links, a hard link to seedbox's download.
- `roots-storage` the Storage block lists `Volumes on home` and `Volumes on seedbox`, each volume with `<free> free of <total>`, a meter named `<volume> used` and the roots on it; the Diagram's Storage cabinet on seedbox has one drawer with `data-used`, and its Inspector `Storage on seedbox` reads `1 root on 1 volume, <size> free.` and links `Library roots` (P-028).
- `roots-sample-every` the Storage block's Details sets `Check storage every` to 10 minutes, and the summary reads `Storage Nodes check each volume every 10 min, and again after each import.` (P-028).
- `roots-write-off` unticking Storage on seedbox drains it, its Roster chip reading `Storage · Draining, 1 root (<size>) stays`; writing seedbox's Storage off marks the root `Node written off` with `Re-home` first in its menu, the Work's cell reads `missing`, and its search answers 409 `root_unreachable`, `seedbox's Storage was written off, so the library root there takes no new files.`
- `roots-rehome` `Re-home` onto home at the same folder checks the recorded file, saves with `The Movies root is now <path> on home.`, and the cell reads `upgradable` again with no new download.
- `roots-delete` deleting the worker root, which holds the file, answers 409 `root_in_use` with its file count; deleting the empty `movies` root toasts `Deleted the <path> root. The folder is still on disk.`, and its `.slipstream-root` marker is still in the folder (P-021).

## How to get to it (user POV)

- Desktop shell: the sidebar's `Settings`, then `Library roots` in the `Settings` navigation (`/settings/library-roots`). `Add a library root`, `Add a root for Movies` and a row's `Actions for <path>` open the dialogs; the dialog's `Browse` opens the folder browser.
- Phone shell: More, `Settings`, then `Library roots`, a pushed page. The dialogs and the folder browser are sheets; the browser's `Folder actions` holds `New folder` and `Show hidden`, and its back link, named after the parent folder, goes up.
- Both shells: `Nodes` with `?view=diagram`, then the Storage machine on a tile, opens `Storage on <Node>`, whose `Library roots` link opens the section.
- API (Admin): `GET`, `POST /api/v1/roots`, `POST /api/v1/roots/check`, `PATCH`, `DELETE /api/v1/roots/{rootId}?worksTo=<rootId>`, `POST /api/v1/roots/{rootId}/default`, `GET`, `PATCH /api/v1/roots/settings`, `GET /api/v1/volumes`, `GET /api/v1/nodes/{nodeId}/folders?path=&hidden=`, `GET /api/v1/nodes/{nodeId}/folders/suggest?text=`, `POST /api/v1/nodes/{nodeId}/folders`, `PATCH /api/v1/works/{workId} {"rootId": …}`. Changes publish `roots.changed` and `volumes.changed`.
- Host CLI: none.

## Driving it with fakeworld and playwright cli

Preconditions:

- A fresh `home` run launched, doctor passing and `{runId}-admin-desktop` signed in, as in the skill's Launch and Drive. The library holds no Work.
- `S='-s={runId}-admin-desktop'`, and `PICK='getByRole("dialog", { name: "Choose a folder on seedbox" })'` and `ADD='getByRole("dialog", { name: "Add a library root" })'` for the long locators.

Steps:

- **The section.** Open `/settings/library-roots`. `Movies roots` lists `{mediaDir}/movies` on home with `Default` and `0 Movies, 0 bytes`.

  ```sh
  $FW exec $RUN -- $PW cli $S tracing-start
  $FW exec $RUN -- $PW cli $S goto '{nodes.home.browserUrl}/settings/library-roots'
  $FW exec $RUN --proof library-roots/roots-section -- $PW cli $S snapshot --filename={runDir}/proof/library-roots/roots-section/section.aria.yml
  $FW exec $RUN --proof library-roots/roots-section -- $PW cli $S screenshot --filename={runDir}/proof/library-roots/roots-section/section.png
  $FW exec $RUN --proof library-roots/roots-section -- curl -sS -H 'X-Api-Key: {apiKey}' '{nodes.home.url}/api/v1/roots'
  ```

- **Give seedbox Storage.** Set seedbox's Roles to Searching, Downloading and Storage. The set answers `seedbox starts Storage.`

  ```sh
  BASIS=$($FW exec $RUN -- curl -sS -H 'X-Api-Key: {apiKey}' '{nodes.home.url}/api/v1/nodes/roles' | jq -r .basis)
  $FW exec $RUN --proof library-roots/roots-add-browser -- curl -sS -X PUT -H 'X-Api-Key: {apiKey}' -H 'Content-Type: application/json' --data-binary "{\"basis\":\"$BASIS\",\"set\":[{\"nodeId\":\"{nodes.seedbox.nodeId}\",\"roles\":[\"searching\",\"downloading\",\"storage\"]}]}" '{nodes.home.url}/api/v1/nodes/roles'
  ```

- **Browse seedbox.** Choose `Add a library root`, pick `seedbox` in `Node` and choose `Browse`. Type seedbox's data dir into `Filter, or go to a path` and press Enter: the footer reads `Can't use this folder because it's inside SlipStream's data folder.` and `Use seedbox` is disabled.

  ```sh
  $FW exec $RUN -- $PW cli $S click 'getByRole("button", { name: "Add a library root" })'
  $FW exec $RUN -- $PW cli $S click "$ADD"'.getByRole("combobox", { name: "Node" })'
  $FW exec $RUN -- $PW cli $S click 'getByRole("option", { name: "seedbox" })'
  $FW exec $RUN -- $PW cli $S click "$ADD"'.getByRole("button", { name: "Browse" })'
  $FW exec $RUN --proof library-roots/roots-add-browser -- $PW cli $S screenshot --filename={runDir}/proof/library-roots/roots-add-browser/places.png
  $FW exec $RUN -- $PW cli $S fill "$PICK"'.getByRole("combobox", { name: "Filter, or go to a path" })' '{nodes.seedbox.dataDir}'
  $FW exec $RUN -- $PW cli $S press Enter
  $FW exec $RUN -- $PW cli $S run-code 'async page => { await page.getByText("Can'"'"'t use this folder because it'"'"'s inside SlipStream'"'"'s data folder.").waitFor(); }'
  $FW exec $RUN --proof library-roots/roots-add-browser -- $PW cli $S snapshot --filename={runDir}/proof/library-roots/roots-add-browser/data-folder.aria.yml
  $FW exec $RUN --proof library-roots/roots-add-browser -- $PW cli $S screenshot --filename={runDir}/proof/library-roots/roots-add-browser/data-folder.png
  ```

- **Make a folder and use it.** Go to `{mediaDir}`, choose `New folder`, type `worker-movies` and press Enter. The browser opens it, reads `This folder is on /, <size> free.` and `Use worker-movies` fills `Folder`. Tick `Make this the default for Movies` and choose `Add`: the toast reads `Added {mediaDir}/worker-movies. New Movies go there.` and the row moves `Default` to it.

  ```sh
  $FW exec $RUN -- $PW cli $S fill "$PICK"'.getByRole("combobox", { name: "Filter, or go to a path" })' '{mediaDir}'
  $FW exec $RUN -- $PW cli $S press Enter
  $FW exec $RUN -- $PW cli $S click "$PICK"'.getByRole("button", { name: "New folder" })'
  $FW exec $RUN -- $PW cli $S fill "$PICK"'.getByRole("textbox", { name: "New folder name" })' worker-movies
  $FW exec $RUN -- $PW cli $S press Enter
  $FW exec $RUN -- $PW cli $S run-code 'async page => { await page.getByRole("button", { name: "Use worker-movies" }).waitFor(); }'
  $FW exec $RUN --proof library-roots/roots-add-browser -- $PW cli $S screenshot --filename={runDir}/proof/library-roots/roots-add-browser/new-folder.png
  $FW exec $RUN -- $PW cli $S click "$PICK"'.getByRole("button", { name: "Use worker-movies" })'
  $FW exec $RUN -- $PW cli $S check "$ADD"'.getByRole("checkbox", { name: "Make this the default for Movies" })'
  $FW exec $RUN --proof library-roots/roots-add-browser -- $PW cli $S screenshot --filename={runDir}/proof/library-roots/roots-add-browser/dialog.png
  $FW exec $RUN -- $PW cli $S click "$ADD"'.getByRole("button", { name: "Add", exact: true })'
  $FW exec $RUN -- $PW cli $S run-code 'async page => { await page.getByRole("region", { name: "Notifications" }).getByText("New Movies go there.").waitFor(); }'
  $FW exec $RUN --proof library-roots/roots-add-browser -- $PW cli $S screenshot --filename={runDir}/proof/library-roots/roots-add-browser/added.png
  $FW exec $RUN --proof library-roots/roots-add-browser -- curl -sS -H 'X-Api-Key: {apiKey}' '{nodes.home.url}/api/v1/roots'
  ```

- **Import on the Worker.** Add The Quiet Orchard from the Library's `Add` menu and choose `Search`. Activity reads `Imported`, the Grab's history has no `transferring`, and the file under `worker-movies` matches the release with `"links":2`.

  ```sh
  $FW exec $RUN -- $PW cli $S goto '{nodes.home.browserUrl}/library'
  $FW exec $RUN -- $PW cli $S click 'getByRole("banner").getByRole("button", { name: "Add" })'
  $FW exec $RUN -- $PW cli $S click 'getByRole("menuitem", { name: "Add a Movie" })'
  $FW exec $RUN -- $PW cli $S fill 'getByRole("searchbox", { name: "Title" })' "quiet orchard"
  $FW exec $RUN -- $PW cli $S click 'getByRole("main").getByRole("button", { name: "Search", exact: true })'
  $FW exec $RUN -- $PW cli $S click 'getByRole("button", { name: "Add The Quiet Orchard" })'
  $FW exec $RUN -- $PW cli $S click 'getByRole("dialog", { name: "Add The Quiet Orchard" }).getByRole("button", { name: "Add Movie" })'
  $FW exec $RUN -- $PW cli $S click 'getByRole("banner").getByRole("button", { name: "Search", exact: true })'
  $FW exec $RUN -- $PW cli $S click 'getByRole("navigation", { name: "Main" }).getByRole("link", { name: "Activity" })'
  $FW exec $RUN -- $PW cli $S run-code 'async page => { await page.getByRole("table", { name: "Grabs" }).getByText("Imported").waitFor({ timeout: 60000 }); }'
  $FW exec $RUN --proof library-roots/roots-import-worker -- $PW cli $S screenshot --filename={runDir}/proof/library-roots/roots-import-worker/activity.png
  $FW exec $RUN --proof library-roots/roots-import-worker -- curl -sS -H 'X-Api-Key: {apiKey}' '{nodes.home.url}/api/v1/grabs'
  $FW exec $RUN --proof library-roots/roots-import-worker -- $FW fs match {runDir} '{mediaDir}/worker-movies/The Quiet Orchard (2021)/The Quiet Orchard (2021) - 1080p.mkv' --release The.Quiet.Orchard.2021.1080p.WEB-DL.H264-FKW
  $FW exec $RUN --proof library-roots/roots-import-worker -- $FW fs stat {runDir} '{mediaDir}/worker-movies/The Quiet Orchard (2021)/The Quiet Orchard (2021) - 1080p.mkv'
  ```

- **Storage.** Open the section again. The worker root reads `1 Movie, 3.5 MB`, and the Storage block lists `Volumes on home` and `Volumes on seedbox`, each with `/`, `<free> free of <total>` and its root. Set `Check storage every` to 10 under Details.

  ```sh
  $FW exec $RUN -- $PW cli $S goto '{nodes.home.browserUrl}/settings/library-roots'
  $FW exec $RUN -- $PW cli $S run-code 'async page => { await page.getByRole("list", { name: "Volumes on seedbox" }).waitFor(); }'
  $FW exec $RUN --proof library-roots/roots-storage -- $PW cli $S snapshot --filename={runDir}/proof/library-roots/roots-storage/section.aria.yml
  $FW exec $RUN --proof library-roots/roots-storage -- $PW cli $S screenshot --full-page --filename={runDir}/proof/library-roots/roots-storage/section.png
  $FW exec $RUN --proof library-roots/roots-storage -- curl -sS -H 'X-Api-Key: {apiKey}' '{nodes.home.url}/api/v1/volumes'
  $FW exec $RUN -- $PW cli $S click 'getByRole("main").getByRole("button", { name: "Details" })'
  $FW exec $RUN -- $PW cli $S fill 'getByRole("textbox", { name: "Check storage every" })' 10
  $FW exec $RUN -- $PW cli $S click 'getByRole("main").getByRole("button", { name: "Save", exact: true })'
  $FW exec $RUN -- $PW cli $S run-code 'async page => { await page.getByText("Storage Nodes check each volume every 10 min, and again after each import.").waitFor(); }'
  $FW exec $RUN --proof library-roots/roots-sample-every -- $PW cli $S screenshot --filename={runDir}/proof/library-roots/roots-sample-every/setting.png
  $FW exec $RUN --proof library-roots/roots-sample-every -- curl -sS -H 'X-Api-Key: {apiKey}' '{nodes.home.url}/api/v1/roots/settings'
  ```

- **The Diagram.** Open the Diagram and choose seedbox's Storage cabinet. Drawer 3 alone carries `data-used`, and `Storage on seedbox` reads `1 root on 1 volume, <size> free.` with the `Library roots` link.

  ```sh
  $FW exec $RUN -- $PW cli $S goto '{nodes.home.browserUrl}/nodes?view=diagram'
  $FW exec $RUN --proof library-roots/roots-storage -- $PW cli $S eval 'Array.from(document.querySelectorAll("[data-node=\"{nodes.seedbox.nodeId}\"] [data-drawer]")).map((drawer) => drawer.dataset.drawer + ":" + (drawer.dataset.used ?? "-")).join(" ")'
  $FW exec $RUN -- $PW cli $S click 'locator("[data-node=\"{nodes.seedbox.nodeId}\"] [data-role=storage]:not([data-chip])").first()'
  $FW exec $RUN -- $PW cli $S run-code 'async page => { await page.getByRole("complementary", { name: "Storage on seedbox" }).getByText(/1 root on 1 volume/).waitFor(); }'
  $FW exec $RUN --proof library-roots/roots-storage -- $PW cli $S snapshot --filename={runDir}/proof/library-roots/roots-storage/inspector.aria.yml
  $FW exec $RUN --proof library-roots/roots-storage -- $PW cli $S screenshot --filename={runDir}/proof/library-roots/roots-storage/inspector.png
  ```

- **Drain and write off seedbox's Storage.** Untick Storage on seedbox: the set's effect reads `seedbox takes no new Storage work. It keeps its 1 root (3.5 MB) until you move or write it off.` and the Roster reads `Storage · Draining, 1 root (3.5 MB) stays`. Write seedbox's Storage off: the root reads `written_off`, the cell `missing`, and its search 409 `root_unreachable`.

  ```sh
  BASIS=$($FW exec $RUN -- curl -sS -H 'X-Api-Key: {apiKey}' '{nodes.home.url}/api/v1/nodes/roles' | jq -r .basis)
  $FW exec $RUN --proof library-roots/roots-write-off -- curl -sS -X PUT -H 'X-Api-Key: {apiKey}' -H 'Content-Type: application/json' --data-binary "{\"basis\":\"$BASIS\",\"set\":[{\"nodeId\":\"{nodes.seedbox.nodeId}\",\"roles\":[\"searching\",\"downloading\"]}]}" '{nodes.home.url}/api/v1/nodes/roles'
  $FW exec $RUN -- $PW cli $S goto '{nodes.home.browserUrl}/nodes?view=roster'
  $FW exec $RUN -- $PW cli $S run-code 'async page => { await page.getByText(/^Storage · Draining, 1 root \(.+\) stays$/).waitFor(); }'
  $FW exec $RUN --proof library-roots/roots-write-off -- $PW cli $S screenshot --filename={runDir}/proof/library-roots/roots-write-off/draining.png
  $FW exec $RUN --proof library-roots/roots-write-off -- curl -sS -X POST -H 'X-Api-Key: {apiKey}' -H 'Content-Type: application/json' --data-binary '{"roles":["storage"]}' '{nodes.home.url}/api/v1/nodes/{nodes.seedbox.nodeId}/write-offs'
  WORK=$($FW exec $RUN -- curl -sS -H 'X-Api-Key: {apiKey}' '{nodes.home.url}/api/v1/works' | jq -r '.works[0].id')
  $FW exec $RUN --proof library-roots/roots-write-off -- curl -sS -H 'X-Api-Key: {apiKey}' '{nodes.home.url}/api/v1/roots'
  $FW exec $RUN --proof library-roots/roots-write-off -- curl -sS -H 'X-Api-Key: {apiKey}' "{nodes.home.url}/api/v1/works/$WORK/cells"
  $FW exec $RUN --proof library-roots/roots-write-off -- curl -sS -i -X POST -H 'X-Api-Key: {apiKey}' "{nodes.home.url}/api/v1/works/$WORK/search"
  ```

- **Re-home onto home.** On the section the worker root reads `Node written off`, and its menu leads with `Re-home`. The dialog `Edit the Movies root` already names home, the only Storage Node; keep the folder and choose `Save`. The toast reads `The Movies root is now {mediaDir}/worker-movies on home.`, and the cell reads `upgradable`.

  ```sh
  $FW exec $RUN -- $PW cli $S goto '{nodes.home.browserUrl}/settings/library-roots'
  $FW exec $RUN -- $PW cli $S click 'getByRole("button", { name: "Actions for {mediaDir}/worker-movies" })'
  $FW exec $RUN --proof library-roots/roots-write-off -- $PW cli $S snapshot --filename={runDir}/proof/library-roots/roots-write-off/menu.aria.yml
  $FW exec $RUN --proof library-roots/roots-write-off -- $PW cli $S screenshot --filename={runDir}/proof/library-roots/roots-write-off/menu.png
  $FW exec $RUN -- $PW cli $S click 'getByRole("menuitem", { name: "Re-home" })'
  $FW exec $RUN -- $PW cli $S run-code 'async page => { await page.getByRole("dialog", { name: "Edit the Movies root" }).getByText(/^This folder is on \//).waitFor(); }'
  $FW exec $RUN --proof library-roots/roots-rehome -- $PW cli $S screenshot --filename={runDir}/proof/library-roots/roots-rehome/dialog.png
  $FW exec $RUN -- $PW cli $S click 'getByRole("dialog", { name: "Edit the Movies root" }).getByRole("button", { name: "Save", exact: true })'
  $FW exec $RUN -- $PW cli $S run-code 'async page => { await page.getByRole("region", { name: "Notifications" }).getByText(/^The Movies root is now .+ on home\.$/).waitFor(); }'
  $FW exec $RUN --proof library-roots/roots-rehome -- $PW cli $S screenshot --filename={runDir}/proof/library-roots/roots-rehome/saved.png
  $FW exec $RUN --proof library-roots/roots-rehome -- curl -sS -H 'X-Api-Key: {apiKey}' '{nodes.home.url}/api/v1/roots'
  $FW exec $RUN --proof library-roots/roots-rehome -- curl -sS -H 'X-Api-Key: {apiKey}' "{nodes.home.url}/api/v1/works/$WORK/cells"
  ```

- **Delete.** Deleting the worker root over the API answers 409 `root_in_use`, since it holds a file. Delete `{mediaDir}/movies` from its menu: the dialog says the folder stays, and the toast reads `Deleted the {mediaDir}/movies root. The folder is still on disk.`; `fs stat` still finds its marker.

  ```sh
  HELD=$($FW exec $RUN -- curl -sS -H 'X-Api-Key: {apiKey}' '{nodes.home.url}/api/v1/roots' | jq -r '.roots[] | select(.works > 0) | .id')
  $FW exec $RUN --proof library-roots/roots-delete -- curl -sS -i -X DELETE -H 'X-Api-Key: {apiKey}' "{nodes.home.url}/api/v1/roots/$HELD"
  $FW exec $RUN -- $PW cli $S click 'getByRole("button", { name: "Actions for {mediaDir}/movies" })'
  $FW exec $RUN -- $PW cli $S click 'getByRole("menuitem", { name: "Delete" })'
  $FW exec $RUN --proof library-roots/roots-delete -- $PW cli $S screenshot --filename={runDir}/proof/library-roots/roots-delete/confirm.png
  $FW exec $RUN -- $PW cli $S click 'getByRole("dialog", { name: "Delete {mediaDir}/movies?" }).getByRole("button", { name: "Delete", exact: true })'
  $FW exec $RUN -- $PW cli $S run-code 'async page => { await page.getByRole("region", { name: "Notifications" }).getByText(/The folder is still on disk\.$/).waitFor(); }'
  $FW exec $RUN --proof library-roots/roots-delete -- $PW cli $S screenshot --filename={runDir}/proof/library-roots/roots-delete/deleted.png
  $FW exec $RUN --proof library-roots/roots-delete -- $FW fs stat {runDir} '{mediaDir}/movies/.slipstream-root'
  $FW exec $RUN -- $PW cli $S tracing-stop
  ```

- **Phone.** Open `{runId}-admin-phone` as in the skill's Drive with `phone` and go to `/settings/library-roots`: one pushed page with the worker root on home and the Storage block.

  ```sh
  $FW exec $RUN -- bun web/harness/bin/sign-in.ts {runDir} admin phone
  $FW exec $RUN -- $PW cli -s={runId}-admin-phone open --config={runDir}/browser/phone/playwright-cli.json
  $FW exec $RUN -- $PW cli -s={runId}-admin-phone state-load {runDir}/auth/admin-phone.json
  $FW exec $RUN -- $PW cli -s={runId}-admin-phone goto '{nodes.home.browserUrl}/settings/library-roots'
  $FW exec $RUN -- $PW cli -s={runId}-admin-phone run-code 'async page => { await page.getByRole("list", { name: "Volumes on home" }).waitFor(); }'
  $FW exec $RUN --proof library-roots/roots-section -- $PW cli -s={runId}-admin-phone screenshot --full-page --filename={runDir}/proof/library-roots/roots-section/section-phone.png
  ```

## Gotchas

- Overlap is judged per Node. `{mediaDir}/movies` is home's root, so seedbox's browser reads it as a usable folder; a refusal on seedbox needs seedbox's own data or downloads folder, or a root on seedbox.
- exec fills every `{...}` in a command from the manifest, so a script passed to `eval` or `run-code` can't hold a template literal's `${...}`; join strings with `+` instead.
- The Library's `Add` button sits in the Library's banner only, so go to `/library` before the Add steps.
- The completed announce lands before any step that follows `Search` returns, so wait on Activity's `Imported` rather than `announces wait` without an `--after` seq.
- After seedbox lets Storage go, the Edit dialog defaults its `Node` to home, the only Storage Node left; the row still names seedbox until Save.
- A re-home with one file checks it too fast to see `Checked <n> of <m> files at the new place`; the progress line shows only on a large root.
- The Storage block counts a volume once per Node. home and seedbox share the container's `/`, so each Node lists its own `/`.
- `Re-home` with missing files and `Save anyway, forget <n> files`, deleting a root whose Works move to another, the empty state and the phone sheets are driven by `web/e2e/library-roots.spec.ts` on the browser tier.
