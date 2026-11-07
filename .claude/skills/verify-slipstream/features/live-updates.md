# Live updates

Every signed-in page keeps a live stream open to the Controller and says nothing about it while it works. When the Controller goes away a quiet line reads `Reconnecting to SlipStream…`, in the top bar in the desktop shell and above the tab bar on phone, and when the Controller comes back on its address the stream resumes and the line goes.

## Sub-features

- `live-on` while the stream is open, no connection line shows.
- `live-reconnect` killing the Controller shows the status `Reconnecting to SlipStream…`, and starting it again on the same port removes it.
- `live-stream-captured` the Admin's stream is captured from launch: it opens with `live.resync`, and opens again with `live.resync` after the Controller restarts.

## How to get to it (user POV)

- Desktop shell: the top bar's right side, which holds the line only while reconnecting.
- Phone shell: just above the tab bar, only while reconnecting.
- API: `GET /api/v1/events` with a session cookie; the run's capture reads it as `sse.message` records.

## Driving it with fakeworld and playwright cli

Preconditions:

- The baseline: a `home` run, doctor passing, `{runId}-admin-desktop` signed in on the Library. Run this last or on its own run: the Controller restarts.

Steps:

- **See it on.** The top bar holds the capsule and the page's action, and no status.

  ```sh
  $FW exec $RUN --proof live-updates/live-on -- $PW cli -s={runId}-admin-desktop snapshot 'getByRole("banner")' --filename={runDir}/proof/live-updates/live-on/banner.aria.yml
  ```

- **Kill the Controller.** The kill prints the entry with `"state":"killed"`, and the top bar shows `Reconnecting to SlipStream…`.

  ```sh
  $FW node kill $RUN home
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop run-code 'async page => { await page.getByText("Reconnecting to SlipStream…").waitFor({ timeout: 10000 }); }'
  $FW exec $RUN --proof live-updates/live-reconnect -- $PW cli -s={runId}-admin-desktop screenshot --filename={runDir}/proof/live-updates/live-reconnect/reconnecting.png
  ```

- **Start it again.** The start prints the same `http` listener as before, the line goes, and doctor passes once the seedbox has relinked.

  ```sh
  $FW node start $RUN home
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop run-code 'async page => { await page.getByText("Reconnecting to SlipStream…").waitFor({ state: "hidden", timeout: 30000 }); }'
  $FW exec $RUN --proof live-updates/live-reconnect -- $PW cli -s={runId}-admin-desktop screenshot --filename={runDir}/proof/live-updates/live-reconnect/on.png
  $FW doctor $RUN
  ```

- **Read the captured stream.** The list holds `live.resync` once at launch and once after the restart.

  ```sh
  $FW sse list $RUN --as admin --event live.resync
  ```

## Gotchas

- The line is an empty `status` while the stream is open, so prove `live-on` by its absence in the banner's snapshot, not by a text.
- Right after `node start home`, doctor's check 3 can read `node seedbox's link is down, not up` until the seedbox relinks; run doctor again.
- The page reconnects on its own; reloading it would prove a fresh connection, not the reconnect.
- The supervisor's capture re-opens on the Node's next `node.ready`, never on a timer, so a killed Node with no start leaves the capture closed.
- `first-run` has no Admin until setup, so its run captures no stream.
- `works.changed`, `works.removed` and `targets.changed` name their Works in `workIds`, so a bulk action sends one event. `grabs.changed` carries `progress`, true when only a download's progress moved.
