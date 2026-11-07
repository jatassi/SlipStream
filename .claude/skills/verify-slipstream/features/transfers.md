# Transfers

A Transfer copies a finished download's files from the Node that downloaded them to the Node that holds the Work's library root, verified chunk by chunk and resumable. On `seedbox-nas` a Grab downloads on the Worker seedbox and lands in the Movies root on the Worker nas, with the Controller home in neither. Each Node's Transfers section opens with one sentence, such as `Receiving The Quiet Orchard from seedbox at 540 KB/s` or `Nothing moving`, then the Node's send limit, the Transfers it sends and receives with their state and pace, and the recent speed to and from each other Node. A copying Transfer reads `Transferring from seedbox` over `7.6 MB of 36 MB · 540 KB/s · 1 min left` there and in Activity's stage and progress cells. A Transfer that can't copy says why in the same place: `Connecting to seedbox.`, `Waiting for seedbox to come back.` or `Waiting for 242.3 GB free on nas (/).`, and it resumes from its verified chunks once the cause clears.

## Sub-features

- `transfers-section` P-363 Nodes > a Node > `Transfers` appears once SlipStream has more than one Node. On a Node with nothing moving it reads `Nothing moving`, the `Settings` region holds `Send limit` (`Default No limit. Up to 2000 MB/s. 0 means no limit.`), and `Recent speeds` reads `No Transfer has moved bytes between nas and another Node yet.` Covered by `web/e2e/transfers.spec.ts` (`seedbox's Transfers say nothing moves…`) and the section's unit tests under `web/src/transfers/`.
- `transfers-send-limit` P-363 seedbox's `Send limit` of 512 with `KB/s` picked saves with the toast `Transfer settings saved`, applies at once, and `GET /api/v1/transfers/nodes/{nodeId}/settings` reads `value` and `running` 524288. A Transfer from seedbox then copies near 512 KB/s across every lane. Covered by `TestTheSendLimitPacesChunks` and `TestASendLimitTooSlowForAChunkPerLaneWithinTheChunkTimeoutStillArrives` in `internal/transfers`, `TestANodesSendLimitChangesAndResets` in `internal/transfersapi`, and both browser tests.
- `transfers-copying` P-363 the Grab of The Quiet Orchard reads, on nas, `Receiving The Quiet Orchard from seedbox at <rate>` and a row `Transferring from seedbox` over `<bytes> of 36 MB · <rate> · <n> min left`, whose Details list `Route` `nas dials seedbox`, `Lanes` 4, `Round trip`, `Retried chunks` and `Files` `0 of 1 verified`. seedbox reads `Sending The Quiet Orchard to nas at <rate>` and `Transferring to nas`. Activity's Grabs row reads `Transferring from seedbox` and the same pace. `GET /api/v1/transfers?nodeId=` lists it with `state` `copying`, `route` `receiver_dials` and `details`. Covered by `web/e2e/transfers.spec.ts` (`a Grab on seedbox reads Transferring from seedbox…`), `TestAGrabOnSeedboxLandsInNassRootThroughATransfer` in `fakes/world/sim`, and `TestATransferReadsWhatItWaitsForThenItsPace` in `internal/transfersapi`.
- `transfers-cut` P-363 a `cut=1MiB` on `link/seedbox` mid-Transfer closes one lane (`front.close` with `cause` `cut`), nas dials it again (`front.open`), `Retried chunks` counts the chunks it lost, and the Transfer copies on. Covered by `TestALinkCutMidTransferResumes` in `e2e`, `TestATransferBetweenWorkersResumesAcrossCutLanes` in `fakes/world/sim`, and `TestACutLaneResumesFromTheVerifiedChunks` and `TestALaneStalledPastTheChunkTimeoutIsRedialed` in `internal/transfers`.
- `transfers-needs-space` P-363 with nas's volume given its own levels of 100%, the Transfer goes to `needs_space` with `reason` `Waiting for <bytes> free on nas (/).`; nas's row reads `From seedbox`, the chip `Needs space` and that reason, its sentence reads `1 waiting`, and Activity's stage cell reads the reason. Removing the levels sends it back to copying. Covered by `TestJudge` in `internal/transfers` (the space cases) and `TestADiskThatFillsWaitsAndGoesOn` for a disk that fills mid-copy.
- `transfers-node-away` P-363 with seedbox stopped past the blip (`standing` `offline`), the Transfer reads `Waiting for seedbox to come back.`; once seedbox starts it reads `Connecting to seedbox.`, then copies on from nas's verified chunks and imports. Covered by `TestATransferBetweenWorkersSurvivesASeedboxKillAndAControllerRestart` in `fakes/world/sim` and `TestJudge` in `internal/transfers`.
- `transfers-landed` P-363 the Grab imports: Activity reads `Imported`, the placed file under `movies` matches the release by BLAKE3, `movies/.slipstream-staging` is empty, nas reads `Nothing moving` again, `Recent speeds` lists `From seedbox` on nas and `To nas` on seedbox, `GET /api/v1/transfers/speeds` lists the pair with `route` `receiver_dials`, and the Admin's stream carries `transfers.changed`. Covered by `TestAGrabOnSeedboxLandsInNassRootThroughATransfer` in `fakes/world/sim`, `TestSpeedsListEachPairsLastSpeedAndRoute` in `internal/transfersapi` and the browser test.

## How to get to it (user POV)

- Desktop shell: `Nodes`, the Node in the Roster or the picker, then its `Transfers` page tab, `/nodes/<id>/transfers`. The palette's `Go to Transfers` opens it for the browser's own Node, home.
- Phone shell: `More`, then `Nodes`, the Node's row in the Roster, then `Transfers` in the list named for the Node, a pushed page.
- Activity: a transferring Grab's row in `Grabs` reads the Transfer's sentence in `Stage` and its pace in `Progress`.
- API on the Controller (Admin): `GET /api/v1/transfers` with `nodeId` or `source` (an info hash), `GET /api/v1/transfers/{id}`, `GET /api/v1/transfers/speeds`, `GET` and `PATCH /api/v1/transfers/nodes/{nodeId}/settings` with `{"sendLimit": <bytes a second>}`, and the event `transfers.changed`. `GET /api/v1/grabs` carries only `progress.done` and `progress.total` for a transferring Grab.

## Driving it with fakeworld, curl and playwright cli

Preconditions:

- A fresh `seedbox-nas` run with doctor passing, launched with `--scenario seedbox-nas` in place of `home`. Its release is a 60 second video of 37,373,411 bytes, nine 4 MiB chunks.
- The `{runId}-admin-desktop` and `{runId}-admin-phone` sessions open and signed in, and these handles:

  ```sh
  D=-s={runId}-admin-desktop
  P=-s={runId}-admin-phone
  SB=$(jq -r '.nodes[] | select(.name == "seedbox").nodeId' $RUN/manifest.json)
  NAS=$(jq -r '.nodes[] | select(.name == "nas").nodeId' $RUN/manifest.json)
  ```

Steps:

- **Read nas's empty Transfers.** Desktop and phone.

  ```sh
  $FW exec $RUN -- $PW cli $D goto "{nodes.home.browserUrl}/nodes/$NAS/transfers"
  $FW exec $RUN --proof transfers/transfers-section -- $PW cli $D run-code 'async page => { await page.getByText("Nothing moving", { exact: true }).waitFor({ timeout: 10000 }); return "Nothing moving"; }'
  $FW exec $RUN --proof transfers/transfers-section -- $PW cli $D snapshot 'getByRole("main")' --filename={runDir}/proof/transfers/transfers-section/desktop.aria.yml
  $FW exec $RUN --proof transfers/transfers-section -- $PW cli $D screenshot --full-page --filename={runDir}/proof/transfers/transfers-section/desktop.png
  $FW exec $RUN -- $PW cli $P goto "{nodes.home.browserUrl}/nodes/$NAS/transfers"
  $FW exec $RUN --proof transfers/transfers-section -- $PW cli $P screenshot --full-page --filename={runDir}/proof/transfers/transfers-section/phone.png
  ```

- **Limit seedbox to 512 KB/s.** On seedbox's Transfers, type 512, pick `KB/s` and save. Without a limit the Transfer is over in about a second on loopback.

  ```sh
  $FW exec $RUN -- $PW cli $D goto "{nodes.home.browserUrl}/nodes/$SB/transfers"
  $FW exec $RUN -- $PW cli $D fill 'getByRole("textbox", { name: "Send limit" })' 512
  $FW exec $RUN -- $PW cli $D click 'getByRole("radiogroup", { name: "Send limit unit" }).getByRole("radio", { name: "KB/s" })'
  $FW exec $RUN -- $PW cli $D click 'getByRole("region", { name: "Settings" }).getByRole("button", { name: "Save" })'
  $FW exec $RUN --proof transfers/transfers-send-limit -- $PW cli $D run-code 'async page => { await page.getByRole("region", { name: "Notifications" }).getByText("Transfer settings saved").waitFor({ timeout: 10000 }); return "saved"; }'
  $FW exec $RUN --proof transfers/transfers-send-limit -- curl -sS -H 'X-Api-Key: {apiKey}' "{nodes.home.url}/api/v1/transfers/nodes/$SB/settings"
  ```

- **Grab The Quiet Orchard and watch it copy.** Add and search it over the API, wait for `copying`, then read nas with Details open, seedbox and Activity. The rate reads near 512 KB/s.

  ```sh
  $FW exec $RUN -- sh -c "id=\$(curl -sS -X POST -H 'X-Api-Key: {apiKey}' -H 'Content-Type: application/json' -d '{\"contentType\":\"movie\",\"key\":\"9000001\",\"monitor\":true}' '{nodes.home.url}/api/v1/works' | jq -r .work.id); curl -sS -X POST -H 'X-Api-Key: {apiKey}' \"{nodes.home.url}/api/v1/works/\$id/search\""
  $FW exec $RUN -- bash -c "for i in \$(seq 120); do curl -sS -H 'X-Api-Key: {apiKey}' '{nodes.home.url}/api/v1/transfers' | jq -e '.transfers[0].state == \"copying\"' >/dev/null && exit 0; sleep 1; done; exit 1"
  $FW exec $RUN --proof transfers/transfers-copying -- curl -sS -H 'X-Api-Key: {apiKey}' "{nodes.home.url}/api/v1/transfers?nodeId=$NAS"
  $FW exec $RUN -- $PW cli $D goto "{nodes.home.browserUrl}/nodes/$NAS/transfers"
  $FW exec $RUN -- $PW cli $D click 'getByRole("list", { name: "Transfers of nas" }).getByRole("button", { name: "Details" })'
  $FW exec $RUN --proof transfers/transfers-copying -- $PW cli $D snapshot 'getByRole("main")' --filename={runDir}/proof/transfers/transfers-copying/nas.aria.yml
  $FW exec $RUN --proof transfers/transfers-copying -- $PW cli $D screenshot --full-page --filename={runDir}/proof/transfers/transfers-copying/nas.png
  $FW exec $RUN -- $PW cli $P goto "{nodes.home.browserUrl}/nodes/$NAS/transfers"
  $FW exec $RUN --proof transfers/transfers-copying -- $PW cli $P screenshot --full-page --filename={runDir}/proof/transfers/transfers-copying/nas-phone.png
  $FW exec $RUN -- $PW cli $D goto "{nodes.home.browserUrl}/nodes/$SB/transfers"
  $FW exec $RUN --proof transfers/transfers-copying -- $PW cli $D snapshot 'getByRole("main")' --filename={runDir}/proof/transfers/transfers-copying/seedbox.aria.yml
  $FW exec $RUN -- $PW cli $D goto '{nodes.home.browserUrl}/activity'
  $FW exec $RUN --proof transfers/transfers-copying -- $PW cli $D snapshot 'getByRole("table", { name: "Grabs" })' --filename={runDir}/proof/transfers/transfers-copying/activity.aria.yml
  $FW exec $RUN --proof transfers/transfers-copying -- $PW cli $D screenshot --full-page --filename={runDir}/proof/transfers/transfers-copying/activity.png
  ```

- **Cut seedbox's front mid-Transfer.** One lane closes on the cut and nas dials again; `retried` counts the lost chunks.

  ```sh
  $FW fault set $RUN lane-cut link/seedbox cut=1MiB --times 1
  $FW exec $RUN -- bash -c "for i in \$(seq 30); do grep -q '\"cause\":\"cut\"' {runDir}/events.jsonl && exit 0; sleep 1; done; exit 1"
  $FW exec $RUN --proof transfers/transfers-cut -- bash -c "jq -c 'select(.data.target == \"link/seedbox\" and (.kind == \"front.close\" or .kind == \"front.open\"))' {runDir}/events.jsonl | tail -4"
  $FW exec $RUN --proof transfers/transfers-cut -- curl -sS -H 'X-Api-Key: {apiKey}' '{nodes.home.url}/api/v1/transfers'
  $FW fault clear $RUN lane-cut
  ```

- **Hold it for space.** Give nas's volume its own levels of 100%: the Transfer reads Needs space on nas and in Activity. Then remove them and it copies again.

  ```sh
  $FW exec $RUN --proof transfers/transfers-needs-space -- curl -sS -X PATCH -H 'X-Api-Key: {apiKey}' -H 'Content-Type: application/json' --data-binary "{\"nodeId\":\"$NAS\",\"mount\":\"/\",\"warnBelow\":{\"amount\":100,\"unit\":\"percent\"},\"errorBelow\":{\"amount\":100,\"unit\":\"percent\"}}" '{nodes.home.url}/api/v1/system/health/volumes'
  $FW exec $RUN -- bash -c "for i in \$(seq 30); do curl -sS -H 'X-Api-Key: {apiKey}' '{nodes.home.url}/api/v1/transfers' | jq -e '.transfers[0].state == \"needs_space\"' >/dev/null && exit 0; sleep 1; done; exit 1"
  $FW exec $RUN --proof transfers/transfers-needs-space -- curl -sS -H 'X-Api-Key: {apiKey}' '{nodes.home.url}/api/v1/transfers'
  $FW exec $RUN --proof transfers/transfers-needs-space -- $PW cli $D run-code 'async page => { const reason = page.getByRole("table", { name: "Grabs" }).getByText(/^Waiting for .+ free on nas/u); await reason.waitFor({ timeout: 10000 }); return reason.textContent(); }'
  $FW exec $RUN --proof transfers/transfers-needs-space -- $PW cli $D snapshot 'getByRole("table", { name: "Grabs" })' --filename={runDir}/proof/transfers/transfers-needs-space/activity.aria.yml
  $FW exec $RUN -- $PW cli $D goto "{nodes.home.browserUrl}/nodes/$NAS/transfers"
  $FW exec $RUN --proof transfers/transfers-needs-space -- $PW cli $D snapshot 'getByRole("main")' --filename={runDir}/proof/transfers/transfers-needs-space/nas.aria.yml
  $FW exec $RUN --proof transfers/transfers-needs-space -- $PW cli $D screenshot --full-page --filename={runDir}/proof/transfers/transfers-needs-space/nas.png
  $FW exec $RUN --proof transfers/transfers-needs-space -- curl -sS -i -X DELETE -H 'X-Api-Key: {apiKey}' "{nodes.home.url}/api/v1/system/health/volumes?nodeId=$NAS&mount=%2F"
  $FW exec $RUN -- bash -c "for i in \$(seq 60); do curl -sS -H 'X-Api-Key: {apiKey}' '{nodes.home.url}/api/v1/transfers' | jq -e '.transfers[0].state == \"copying\"' >/dev/null && exit 0; sleep 1; done; exit 1"
  ```

- **Stop seedbox, then start it.** Once seedbox is `offline` the Transfer waits for it; started again, it copies on and the Grab imports.

  ```sh
  $FW node stop $RUN seedbox
  $FW exec $RUN -- bash -c "for i in \$(seq 120); do curl -sS -H 'X-Api-Key: {apiKey}' '{nodes.home.url}/api/v1/transfers' | jq -e '.transfers[0].reason == \"Waiting for seedbox to come back.\"' >/dev/null && exit 0; sleep 1; done; exit 1"
  $FW exec $RUN --proof transfers/transfers-node-away -- curl -sS -H 'X-Api-Key: {apiKey}' '{nodes.home.url}/api/v1/transfers'
  $FW exec $RUN --proof transfers/transfers-node-away -- $PW cli $D snapshot 'getByRole("main")' --filename={runDir}/proof/transfers/transfers-node-away/nas.aria.yml
  $FW node start $RUN seedbox
  $FW doctor $RUN
  $FW exec $RUN -- bash -c "for i in \$(seq 120); do curl -sS -H 'X-Api-Key: {apiKey}' '{nodes.home.url}/api/v1/grabs' | jq -e '.grabs[0].stage == \"imported\"' >/dev/null && exit 0; sleep 1; done; exit 1"
  $FW exec $RUN --proof transfers/transfers-node-away -- curl -sS -H 'X-Api-Key: {apiKey}' '{nodes.home.url}/api/v1/grabs'
  ```

- **See it landed.** The file matches the release, staging is empty, nas and seedbox list the pair's speed, and the stream carried `transfers.changed`.

  ```sh
  $FW exec $RUN --proof transfers/transfers-landed -- $FW fs match {runDir} '{mediaDir}/movies/The Quiet Orchard (2021)/The Quiet Orchard (2021) - 1080p.mkv' --release The.Quiet.Orchard.2021.1080p.WEB-DL.H264-FKW
  $FW exec $RUN --proof transfers/transfers-landed -- ls -A '{mediaDir}/movies/.slipstream-staging'
  $FW exec $RUN --proof transfers/transfers-landed -- curl -sS -H 'X-Api-Key: {apiKey}' '{nodes.home.url}/api/v1/transfers/speeds'
  $FW exec $RUN -- $PW cli $D goto "{nodes.home.browserUrl}/nodes/$NAS/transfers"
  $FW exec $RUN --proof transfers/transfers-landed -- $PW cli $D snapshot 'getByRole("main")' --filename={runDir}/proof/transfers/transfers-landed/nas.aria.yml
  $FW exec $RUN --proof transfers/transfers-landed -- $PW cli $D screenshot --full-page --filename={runDir}/proof/transfers/transfers-landed/nas.png
  $FW exec $RUN -- $PW cli $D goto "{nodes.home.browserUrl}/nodes/$SB/transfers"
  $FW exec $RUN --proof transfers/transfers-landed -- $PW cli $D snapshot 'getByRole("region", { name: "Recent speeds" })' --filename={runDir}/proof/transfers/transfers-landed/seedbox-speeds.aria.yml
  $FW exec $RUN --proof transfers/transfers-landed -- $FW sse list {runDir} --as admin --event transfers.changed
  ```

## Gotchas

- Pages follow `transfers.changed`, published at most once a second, so a snapshot taken just after the API changes can show the state before. Wait for the words with `run-code` first.
- A Transfer reads `Connecting to seedbox.` until its first whole 4 MiB chunk is verified, even while bytes move. At 512 KB/s over four lanes that is about 30 seconds, and the pace line appears only then.
- `link/seedbox` is keyed by the Node that listens, so a fault there holds every connection to seedbox's link listener. On `seedbox-nas` nas dials seedbox for lanes and no other Node does, so it holds the Transfer's lanes alone. Faults on one lane or one pair, damaged chunks and a route refused in one direction stay in the simulation and package tiers.
- Without a send limit the 36 MB Transfer finishes in about a second on loopback, too fast to read. Set the limit before the search. `Send limit` starts at MB/s once it holds a number, so pick `KB/s`; 0 means no limit.
- A process world's Nodes share the host's disk, so 100% levels put any host in Needs space. The reason names the bytes needed above the error level, which on a big host is most of the disk.
- While a Transfer is in Needs space or waits for a Node, the API's `verified` reads 0 and Activity's progress 0%, though nas keeps its verified chunks and resumes from them once it copies again.
- `Waiting for seedbox to come back.` needs seedbox `offline`. A stop reads `reconnecting` first, through the blip, and a reconnecting Node stops nothing, so wait on the reason, not on the stop.
- Lanes between two Workers are not drawn on the Diagram. On `home`, seedbox's cable counts the lane bytes of a Transfer to home ([nodes.md](./nodes.md)).
- A write-off of seedbox's Downloading, a re-home of the root, a source gone or failed, and damaged chunks are proved in `fakes/world/sim` and `internal/transfers`, not driven here; the Scenario has one release, so a run proves one Grab.
