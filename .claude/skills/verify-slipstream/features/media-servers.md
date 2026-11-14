# Media Servers

An Admin adds Plex, Jellyfin and Emby servers in Settings > Media Servers and gives each library root the folder the server sees it at. SlipStream then sends every change to a library file to each server that shows its root, through the Node that reaches the server, with the Controller as fallback. The `media-servers` Scenario starts with all three added the way an Admin adds them: Living room Plex reached from home through a plex.tv sign-in the world approves, Den Jellyfin by its key and Den Emby by an administrator sign-in, both reached from den with the Controller's front as fallback. Each server holds its own Node's movie and series roots and shows only those.

## Sub-features

- `media-servers-add-plex` Settings > Media Servers lists `Living room Plex` with `Plex · Reached from home` and `Living room Plex responds to home.`, and the `Plex account` section reads `plex-admin` and `Plex Pass` (P-189).
- `media-servers-add-jellyfin` `Den Jellyfin` reads `Jellyfin · Reached from den` and `Den Jellyfin responds to den.`; the API lists its fallback address at home's front and `fallback: available` (P-189).
- `media-servers-add-emby` `Den Emby` reads `Emby · Reached from den` and `Den Emby responds to den.`, signed in with the key its administrator sign-in created (P-189).
- `media-servers-mappings` a server's `Path mappings` tab shows each root its Node holds at `/media/<folder>` with `Shown on <server>` on, and every other root not shown (P-189).
- `media-servers-import` an import into a shown root sends the server its change: a fresh library is scanned whole first, then Plex refreshes the folder (`plex-refreshes`) and Jellyfin and Emby take a path report (`jellyfin-reports`, `emby-reports`); the server's `Changes` tab reads the change sent (P-189).
- `media-servers-rename` a rename on Den Jellyfin keeps every user's watch data, and its change reads `Kept every user's watch data` (P-189).
- `media-servers-fallback` with den stopped, the Controller sends Den Jellyfin's changes through home's front (P-189).
- `media-servers-health` a server that stops responding reads `hasn't responded` on its row and raises a Health item after Mark unhealthy after (P-189).

## How to get to it (user POV)

- Desktop shell: `Settings`, then `Media Servers` under `System`, at `/settings/media-servers`; a server's row opens its page with the tabs `Now playing`, `Changes`, `Path mappings`, `Settings` and `Details`. A Node's page lists the servers it reaches in its `Media Servers` section.
- Phone shell: the `More` tab, `Settings`, then `Media Servers`; a server's page lists its sections as rows.
- API (Admin): `GET /api/v1/media-servers`, `GET /api/v1/media-servers/{id}/libraries`, `GET /api/v1/media-servers/{id}/changes`, `GET /api/v1/media-servers/{id}/sessions`, `POST /api/v1/media-servers/{id}/test`; live events `media-servers.changed`, `media-servers.sessions-changed`.
- Host CLI: none.

## Driving it with fakeworld and playwright cli

Preconditions:

- A `media-servers` run (`dist/fakeworld start --binary dist/slipstream --scenario media-servers --run-dir $RUN`), doctor passing, `{runId}-admin-desktop` signed in as in the skill's Drive. Doctor's check 5 reads every server's address and fallback from `GET /api/v1/media-servers` and fails one that isn't the fake's front for the Node that dials it.

Steps:

- **Read the servers.** Open Settings > Media Servers. The list reads all three servers responding, and the API agrees.

  ```sh
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop goto {nodes.home.browserUrl}/settings/media-servers
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop run-code 'async page => { await page.getByRole("list", { name: "Media Servers" }).getByText("Den Emby responds to den.").waitFor(); }'
  $FW exec $RUN --proof media-servers/media-servers-add-plex -- $PW cli -s={runId}-admin-desktop snapshot 'getByRole("main")' --filename={runDir}/proof/media-servers/media-servers-add-plex/list.aria.yml
  $FW exec $RUN --proof media-servers/media-servers-add-plex -- $PW cli -s={runId}-admin-desktop screenshot --filename={runDir}/proof/media-servers/media-servers-add-plex/list.png
  $FW exec $RUN --proof media-servers/media-servers-add-jellyfin -- curl -sS -H 'X-Api-Key: {apiKey}' '{nodes.home.url}/api/v1/media-servers'
  ```

- **Read the Path mappings.** Open Den Jellyfin's `Path mappings` tab. den's roots read shown at `/media/den-movies` and `/media/den-series`, home's not shown.

  ```sh
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop click 'getByRole("list", { name: "Media Servers" }).getByRole("link", { name: /^Den Jellyfin/ })'
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop click 'getByRole("navigation", { name: "Den Jellyfin" }).getByRole("link", { name: "Path mappings" })'
  $FW exec $RUN --proof media-servers/media-servers-mappings -- $PW cli -s={runId}-admin-desktop snapshot 'getByRole("main")' --filename={runDir}/proof/media-servers/media-servers-mappings/jellyfin.aria.yml
  $FW exec $RUN --proof media-servers/media-servers-mappings -- $PW cli -s={runId}-admin-desktop screenshot --filename={runDir}/proof/media-servers/media-servers-mappings/jellyfin.png
  ```

- **Import and read what the server took.** Grab and import into a shown root as [Search and grab](./search-and-grab.md) does, then read the fake's capture and the server's changes.

  ```sh
  $FW plex-refreshes list $RUN
  $FW jellyfin-reports list $RUN
  $FW exec $RUN --proof media-servers/media-servers-import -- curl -sS -H 'X-Api-Key: {apiKey}' '{nodes.home.url}/api/v1/media-servers/<id>/changes'
  ```

- **Stop a server.** Reset every connection to Den Jellyfin; its row reads it hasn't responded once den reports, and Health raises it after Mark unhealthy after. Clear the rule to bring it back.

  ```sh
  $FW fault set $RUN jellyfin-down jellyfin reset
  $FW fault clear $RUN jellyfin-down
  ```

## Gotchas

- Settings > Media Servers lives at `/settings/media-servers`; there is no `/settings/system/...` path.
- Every library starts empty, so the first change into each root starts a full scan and waits for it before the folder refresh or path report. Expect the library refresh in the captures before the folder's.
- Jellyfin takes a path report after its monitor delay, 60 s; Emby after 90 s. Wait on the capture, not on a fixed sleep.
- `playwright cli screenshot` writes at CSS scale whatever the config's `deviceScaleFactor`; for a 2x image use `run-code` with `page.screenshot({ scale: "device" })`.
- The world plays at most one server of each kind, which every Node's front answers as the same server. A fault rule `jellyfin@den` hits den's route only, and `jellyfin@home` the Controller's fallback.
