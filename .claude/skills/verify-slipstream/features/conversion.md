# Conversion

A Target with Conversion settings gets its file made from another Target's file on a Node that holds the Converting Role, instead of downloading one. When a Profile makes a Target from another, each new source file queues a Conversion, which Activity's Conversions tab lists by band: Needs attention, Converting with live progress, and Up next by lane. A Conversion runs on its Node's processor or a proven GPU, pauses while a Media Server that runs on that Node transcodes video, and lands its made file in the Work's folder through an Import. The `conversion` Scenario's movie Profile Films makes 720p mobile from the starter 1080p file, home converts on its processor, and Living room Plex runs on home.

## Sub-features

- `conversion-target-editor` a Target's `Conversion settings` section turns on with `Make this Target's files from another Target's`, takes a codec, a largest resolution and a bitrate, and saves them: `GET /api/v1/targets` reads SD's `conversion` with `maxHeight` 480 and `kbps` 1200.
- `conversion-pause` with a `plex-play` video transcode running on Living room Plex, each new Conversion is placed on home and waits: the tab's row and the Work's 720p mobile cell read `Converting`, `Ready on home` and `Paused while Living room Plex transcodes on home.`, and `GET /api/v1/conversions` lists `wait.code` `transcode` with `servers` `[Living room Plex]`.
- `conversion-tab` the tab's `Show` control counts `All`, `Needs attention`, `Converting` and `Up next`, Up next lists the `New` lane, and `Move up` on the one queued row moves it to `Moved up`, which the API's `next` reads as lane `moved_up`.
- `conversion-run` after `plex-stop`, the Conversions run one at a time on home's processor: the Converting band shows a row with a live progress bar, seconds left and speed, and the tab ends empty.
- `conversion-landed` each Work's 720p mobile cell reads `Met`, its Files list `<Work> - 720p mobile.mkv`, the API's file reads `placedBy` `converted`, `madeFrom` the 1080p file and a 1280x720 H.264 probe, and `fs stat` finds it in the movies root.
- `conversion-node` Node › Converting reads `home converts with its processor.`, with Conversions at once, the hours switch, the processor switch, the scratch folder with its free space, and the passed processor test run under `Test runs`.
- `conversion-backfill` adding SD `Made from 1080p` to Films and pressing `Save` opens `Convert existing movies to SD` with the steps `Cost`, `Movies` and `The rest`, the cost of 3 files on home, `Convert all`, `Choose movies` and `Convert none`, and a restating sentence; `Save` queues the 3 Conversions in the Backfill lane and each Work's SD cell reads `met`.

## How to get to it (user POV)

- Desktop shell: `Activity`, then the `Conversions` tab, at `/activity/conversions`; `Settings`, `Targets`, a Target's `Conversion settings`; `Settings`, `Profiles`, a Profile's Target row's `How <Target> gets its file`; a Node's `Converting` tab at `/nodes/<id>/converting`; a Media Server's `Settings` tab; a Work page's `Editions and Targets` cell and `Files`.
- Phone shell: the `Activity` tab's `Conversions` view, two-line rows with an action sheet; the same Settings and Node pages through `More`.
- API (Admin): `GET /api/v1/conversions` (`band`, `work`), `POST /api/v1/conversions/{id}/move`, `GET /api/v1/nodes/{id}/converting`, `PUT /api/v1/nodes/{id}/converting`, `POST /api/v1/targets` and `PUT /api/v1/targets/{id}` with `conversion`, `PUT /api/v1/profiles/{id}` with `sourcing`, `fromTargetId` and `backfill`, `GET /api/v1/works/{id}/files`; live `conversions.live`.
- Host CLI: none.

## Driving it with fakeworld and playwright cli

Preconditions:

- A `conversion` run launched and doctor passing, with `RUN`, `FW` and `PW` set as in the skill's Launch, and `{runId}-admin-desktop` signed in. For screenshots at device pixels, open the session from a copy of its config with `jq '.browser.contextOptions.deviceScaleFactor = 2'` and pass `--hires` to `screenshot`.

Steps:

- **Turn on Conversion settings.** Open SD's editor, turn the switch on, pick `480p`, type a bitrate and save.

  ```sh
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop goto '{nodes.home.browserUrl}/settings/targets/<SD id>'
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop click "getByRole(\"switch\", { name: \"Make this Target's files from another Target's\" })"
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop click 'getByRole("radio", { name: "480p" })'
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop fill 'getByRole("textbox", { name: "Video bitrate in kbps" })' 1200
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop click 'getByRole("button", { name: "Save", exact: true })'
  $FW exec $RUN --proof conversion/conversion-target-editor -- curl -sS -H 'X-Api-Key: {apiKey}' '{nodes.home.url}/api/v1/targets'
  ```

- **Pause, then queue.** Start a video transcode on Plex, then add and search The Quiet Orchard (9000001), The Salt Road (9000005) and Meadowlark (9000006).

  ```sh
  $FW cue $RUN plex-play "{\"path\":\"/media/shelf/Grandma's 80th Birthday (2019).mkv\",\"user\":\"plex-admin\",\"player\":\"Office PC\",\"playerKey\":\"office-pc\",\"video\":\"transcode\",\"audio\":\"direct\",\"subtitles\":\"none\"}"
  $FW exec $RUN --proof conversion/conversion-pause -- curl -sS -H 'X-Api-Key: {apiKey}' -H 'Content-Type: application/json' -X POST '{nodes.home.url}/api/v1/works' -d '{"contentType":"movie","key":"9000001","preset":"all"}'
  $FW exec $RUN --proof conversion/conversion-pause -- curl -sS -H 'X-Api-Key: {apiKey}' -X POST '{nodes.home.url}/api/v1/works/<work id>/search'
  $FW exec $RUN --proof conversion/conversion-pause -- curl -sS -H 'X-Api-Key: {apiKey}' '{nodes.home.url}/api/v1/conversions'
  ```

  `counts.waiting` reads 3 within about 30 seconds. Open `/activity/conversions` and a Work page, and screenshot each.

- **Move up.** The one queued row has the button.

  ```sh
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop click 'getByRole("button", { name: "Move up Meadowlark" })'
  $FW exec $RUN --proof conversion/conversion-tab -- curl -sS -H 'X-Api-Key: {apiKey}' '{nodes.home.url}/api/v1/conversions'
  ```

- **Run.** Stop the play with the session `plex-play` answered, and wait on the tab for the progress bar.

  ```sh
  $FW cue $RUN plex-stop '{"session":"<session>"}'
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop run-code 'async page => { await page.getByRole("region", { name: "Converting", exact: true }).getByRole("progressbar").first().waitFor({ timeout: 60000 }); }'
  ```

- **Landed.** Read each Work's files and stat the made file under `mediaDir`.

  ```sh
  $FW exec $RUN --proof conversion/conversion-landed -- curl -sS -H 'X-Api-Key: {apiKey}' '{nodes.home.url}/api/v1/works/<work id>/files'
  $FW fs stat $RUN "<mediaDir>/movies/The Quiet Orchard (2021)/The Quiet Orchard (2021) - 720p mobile.mkv"
  ```

- **Node and server.** Open `/nodes/<home id>/converting` and press `Test runs`; open `/settings/media-servers/<Plex id>/settings` for the pause switch, which [Media Servers](./media-servers.md) covers as `media-servers-pause-conversions`.

- **Backfill.** On Films' editor, `Add Target` then `SD`, `How SD gets its file` then `Made from 1080p`, `Save`. Walk the dialog with `Next`, `Save`, then read each Work's cells with `--proof conversion/conversion-backfill`.

## Gotchas

- Plex lists the shelf only because the Scenario plays Grandma's 80th Birthday at launch; a `plex-play` of a file Plex hasn't listed answers `no item`.
- The two 6-second movies convert in about a second each, so only Meadowlark's 60-second run stays on the Converting band long enough to screenshot.
- Turning on Conversion settings leaves the bitrate empty until you type one.
- The tab's rows read the Work's title twice for a movie.
- A made file's `Made from` reaches the API's `madeFrom` but not the Work page yet.
