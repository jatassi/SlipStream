# Moves

An Admin moves a Work with files to another library root of its Content Type, or a seeding torrent to another Download Client, from the Work page, the Library bulk bar, the torrent table and Inspector, a root's row, an external client and the drain Review. One Dialog previews each Move first: it leads with a sentence such as `Move 1 movie and 1 torrent to den.`, groups each Work and torrent under `Moves`, `Moves, with a note` or `Can't move` with its reason, and its button names how many start. Files copy while the source keeps playing and seeding, each Work or torrent switches in one step, and the source is deleted after the switch. A Work whose files are hard links of a seeding torrent takes the torrent with it, so the data crosses once. Activity's `Moves` section lists each batch with its units, `Switch now`, `Cancel` and `Move back…`.

## Sub-features

- `move-work` the Work page's `More actions for <Work>` > `Move to…` opens `Move <Work>` with `Move to` (each other root of its Content Type, `<path> on <Node> · <free>`), `Also move its torrent` ticked, and the preview; the button `Move <n>` starts it and toasts `Moving 1 movie to <Node>.` (P-370)
- `move-work-linked-torrent` a Work imported by hard link from a seeding torrent moves to another Node with that torrent: the preview counts both, the torrent's unit runs first, the Work links its files from the payload the torrent placed, and at the destination the library file is a hard link of the payload the destination client seeds. (P-370)
- `move-bulk` the Library bulk bar's `Move to…` opens `Move <n> <Works>` for the picked Works of one Content Type. (P-370)
- `move-torrent` the torrent table's and Inspector's `Move to…` picks a Node and a client; a moving torrent carries its Move's tag, and a Move into an external client notes that SlipStream stops counting its Hit and run window. (P-370)
- `move-root` a root's `Move root…` edits a moved disk's root or moves its Works, with `Delete this root when it's empty`. (P-370)
- `move-client` an external client's `Move client…` changes its Node or moves its torrents. (P-370)
- `move-drain` the drain Review offers `Also move <Node>'s library files and torrents`, then starts the Move after Apply. (P-370)
- `move-activity` Activity's `Moves` section lists each batch, open ones first, with each unit's sentence, `Switch now` on a playing or Media Server wait, `Cancel` before a unit switches, `Cancel the remaining <n>` and `Move back…` once a batch finished. (P-370)
- `move-waits` a unit waiting at its switch past the threshold under Settings > Library roots' `Moves` heading reaches Home's Needs you with why, such as `An Import into it hasn't finished.` for a Work or `An Import from it hasn't finished.` for a torrent. (P-370)

## How to get to it (user POV)

- Desktop shell: a Work page's `More actions for <Work>` menu, `Move to…`; Library, pick Works, the bulk bar's `More actions` > `Move to…`; Activity's Torrents table and a torrent's Inspector, `Move to…`; Settings > Library roots, a root's menu, `Move root…`; Settings > Download Clients, an external client's sheet, `Move client…`; a Node's Roles Review while it drains Storage or Downloading; Activity's `Moves` section.
- Phone shell: the same pages from `More`; a Work's `More actions for <Work>` opens an action sheet holding `Move to…`.
- API: `POST /api/v1/moves/preview` (`works`, `torrents`, `roots`, `clients`, `options.includeLinkedTorrents`, `options.repointRouting`, `options.deleteEmptiedRoots`), `POST /api/v1/moves?confirm=<movable>` with the preview's `token`, `GET /api/v1/moves`, `GET /api/v1/moves/{batchId}`, `POST /api/v1/moves/{batchId}/cancel`, `GET /api/v1/moves/{batchId}/back`, `POST /api/v1/moves/units/{unitId}/retry`, `.../cancel`, `.../switch-now`, `GET` and `PATCH /api/v1/moves/settings`.

## Driving it with fakeworld and playwright cli

Preconditions:

- A `moves` run with doctor passing and the `{runId}-admin-desktop` session. `moves` has home, seedbox (movies-d, the built-in client, qBittorrent Box and Deluge) and den (movies-e, movies-f, the built-in client, Transmission and rTorrent), and fake-tracker plays UNIT3D's Rust announce with a 5 second guard.

Steps:

- **Grab The Quiet Orchard into seedbox.** Add it and search, as a script would, and wait until its cells read `upgradable`.

  ```sh
  $FW exec $RUN -- curl -sk -H 'X-Api-Key: {apiKey}' -X POST -d '{"contentType":"movie","key":"9000001","preset":"all"}' '{nodes.home.url}/api/v1/works'
  $FW exec $RUN -- curl -sk -H 'X-Api-Key: {apiKey}' -X POST '{nodes.home.url}/api/v1/works/<workId>/search'
  $FW exec $RUN -- stat -c '%h %n' {mediaDir}/movies-d/*/*.mkv
  ```

  The film in movies-d has 2 links: it and seedbox's payload.

- **Move it to den with its torrent.** Open the Work, `More actions for The Quiet Orchard`, `Move to…`, pick `movies-e on den` and read the preview.

  ```sh
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop goto '{nodes.home.browserUrl}/works/<workId>'
  $FW exec $RUN --proof moves/move-work-linked-torrent -- $PW cli -s={runId}-admin-desktop screenshot --filename={runDir}/proof/moves/move-work-linked-torrent/preview.png
  ```

  The Dialog reads `Move 1 movie and 1 torrent to den.` with `Also move its torrent` ticked and the button `Move 2`. Press it: the toast reads `Moving 1 movie to den.`

- **Read the batch and the links at den.**

  ```sh
  $FW exec $RUN -- curl -sk -H 'X-Api-Key: {apiKey}' '{nodes.home.url}/api/v1/moves'
  $FW exec $RUN -- curl -sk -H 'X-Api-Key: {apiKey}' '{nodes.home.url}/api/v1/moves/<batchId>'
  $FW exec $RUN -- stat -c '%h %n' {mediaDir}/movies-e/*/*.mkv
  ```

  Within about 30 seconds the units read `torrent` `done` then `work` `done`, movies-d holds no film, and movies-e's film has 2 links: it and den's payload.

## Gotchas

- The process world refuses a root's volume figures, so `moves` puts movies-f on den's one disk. Only the sim plays movies-f as a second volume (`movestest.Scenario`); a Move to movies-f here links or clones like any same-disk Move.
- `Also move its torrent` counts only when the Work crosses Nodes and the destination Node holds Downloading. A same-Node Move links its files, and the preview's torrent unit is absent.
- The preview's `Media Servers` note on movies-e is expected on `moves`: a Media Server hasn't confirmed the root mapping, so the unit sits under `Moves, with a note`.
- The guard waits out the source's last announce, so a torrent that started seconds ago waits up to the guard before it stops. A refused start leaves the source seeding and den's copy paused, and the Work stays moved.
- `GET /api/v1/moves` lists batches open ones first; poll a batch by id rather than its position.
