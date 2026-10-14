# Live updates

Every signed-in page keeps a live stream open to the Controller and shows its state in the header as `Live updates: on`. When the Controller goes away the header reads `Live updates: reconnecting`, and when it comes back on its address the stream resumes and the header reads `on` again.

## Sub-features

- `live-on` a signed-in page's header reads `Live updates: on`.
- `live-reconnect` killing the Controller turns it to `Live updates: reconnecting`, and starting it again on the same port turns it back to `on`.
- `live-stream-captured` the Admin's stream is captured from launch: it opens with `live.resync`, and opens again with `live.resync` after the Controller restarts.

## How to get to it (user POV)

- Desk and phone: the status text in the header of every signed-in page.
- API: `GET /api/v1/events` with a session cookie; the run's capture reads it as `sse.message` records.

## Driving it with fakeworld and playwright cli

Preconditions:

- The baseline: a `home` run, doctor passing, `{runId}-admin-desk` signed in on the Library. Run this last or on its own run: the Controller restarts.

Steps:

- **See it on.** The header's status reads `Live updates: on`.

  ```sh
  $FW exec $RUN --proof live-updates/live-on -- $PW cli -s={runId}-admin-desk snapshot 'getByRole("banner")' --filename={runDir}/proof/live-updates/live-on/banner.aria.yml
  ```

- **Kill the Controller.** The kill prints the entry with `"state":"killed"`, and the header turns to `Live updates: reconnecting`.

  ```sh
  $FW node kill $RUN home
  $FW exec $RUN -- $PW cli -s={runId}-admin-desk run-code 'async page => { await page.getByText("Live updates: reconnecting").waitFor({ timeout: 10000 }); }'
  $FW exec $RUN --proof live-updates/live-reconnect -- $PW cli -s={runId}-admin-desk screenshot --filename={runDir}/proof/live-updates/live-reconnect/reconnecting.png
  ```

- **Start it again.** The start prints the same `http` listener as before, the header turns back to `Live updates: on`, and doctor passes.

  ```sh
  $FW node start $RUN home
  $FW exec $RUN -- $PW cli -s={runId}-admin-desk run-code 'async page => { await page.getByText("Live updates: on").waitFor({ timeout: 30000 }); }'
  $FW exec $RUN --proof live-updates/live-reconnect -- $PW cli -s={runId}-admin-desk screenshot --filename={runDir}/proof/live-updates/live-reconnect/on.png
  $FW doctor $RUN
  ```

- **Read the captured stream.** The list holds `live.resync` once at launch and once after the restart.

  ```sh
  $FW sse list $RUN --as admin --event live.resync
  ```

## Gotchas

- The page reconnects on its own; reloading it would prove a fresh connection, not the reconnect.
- The supervisor's capture re-opens on the Node's next `node.ready`, never on a timer, so a killed Node with no start leaves the capture closed.
- `first-run` has no Admin until setup, so its run captures no stream.
