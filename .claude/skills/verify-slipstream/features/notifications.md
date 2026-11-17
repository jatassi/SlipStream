# Notifications

An Admin adds notification connections in Settings > System > Notifications: Discord, Telegram, Slack, Pushover, Gotify, ntfy, Pushbullet, Apprise, Email, a Webhook or a Custom Script. Each gets the events it is set for, recorded in the Write that makes each true, and the Node holding Notifying sends them in order, at least once. A connection a provider refuses pauses and holds its messages until a Test of it passes. The bell in the top bar shows each Admin the same events from a feed. Every provider but Email and Custom Script posts to `notify-sink` in the Fake world, Email goes to the `smtp` fake and a Custom Script runs `capture-script`.

## Sub-features

- `notifications-add` adding a Discord connection to the sink answers it with the recommended events (`grab_failed`, `health_item_cleared`, `imported`, `new_health_item`, `upgraded`), `sealed: ["webhookUrl"]`, `shown.webhookUrl` the front's host and no secret value, and doctor's check 5 reads its webhook as the Controller's sink front (P-171, P-172, P-183).
- `notifications-imported` a Grab that imports sends `Imported The Quiet Orchard` from the Node holding Notifying, with the poster as `files[0]`, kept as a file of the `notify` record (P-176, P-183).
- `notifications-paused` with `discord-gone` on, the next message is answered 404 and the connection reads `shown: paused` with `Discord responded that this webhook no longer exists. Paste a new webhook URL.`; later messages wait (P-174).
- `notifications-drain` with the cue off, a Test of the saved connection passes and sends the messages it held, oldest first, after the Test message (P-174).
- `notifications-bell` the feed lists each event newest first with `unread`, `GET /notifications/bell` answers the bell's own events, and `POST /notifications/feed/seen` with the API key answers 403 `needs_account`, since the cursor belongs to an Account; an empty bell reads `No notifications in the last 90 days.`, from Keep history for, and the settings hold no Keep field (P-009).
- `notifications-health-pair` a Download Client fault raises a Health item and its clear clears it, and the bell shows the pair: the item after `Wait before sending a new Health item`, then `Cleared: <item>` (P-009).
- `notifications-scenario` a Scenario's `notify` adds one connection per listed provider at launch, each on with posters and the recommended events, and `script` runs `<run>/bin/capture-script` on the Controller (P-195).

## How to get to it (user POV)

- Desktop shell: `Settings`, then `Notifications` under `System`, at `/settings/notifications`; the bell in the top bar. The section, its sheet and the bell are in [Shell](./shell.md).
- Phone shell: the `More` tab, `Settings`, then `Notifications`; the bell among the bar's actions.
- API (Admin): `GET /api/v1/notifications/providers`, `GET /api/v1/notifications/events`, `GET` and `POST /api/v1/notifications/connections`, `POST /api/v1/notifications/connections/test` (a draft), `GET`, `PATCH` and `DELETE /api/v1/notifications/connections/{id}`, `POST /api/v1/notifications/connections/{id}/test`, `GET` and `PATCH /api/v1/notifications/bell`, `GET /api/v1/notifications/feed`, `POST /api/v1/notifications/feed/seen` (an Account only), `GET` and `PATCH /api/v1/notifications/settings` (no `notifications.keepFor`); live events `notifications.changed`, `notifications.seen` and `history.changed`, which refreshes the feed.
- Host CLI: none.

## Driving it with fakeworld and curl

Preconditions:

- A fresh `home` run, doctor passing. The connection is added through the API here; the sheet's path is in [Shell](./shell.md).
- `SINK` is the Controller's sink front: `SINK=$(jq -r '.fronts[] | select(.target == "notify-sink" and .node == "home").url' $RUN/manifest.json)`.

Steps:

- **Add a Discord connection to the sink.** It answers 201 with the recommended events and no secret, and doctor's check 5 passes.

  ```sh
  C=$($FW exec $RUN --proof notifications/notifications-add -- curl -sS -X POST -H 'X-Api-Key: {apiKey}' -H 'Content-Type: application/json' --data-binary "{\"name\":\"Family\",\"provider\":\"discord\",\"on\":true,\"posters\":true,\"secrets\":{\"webhookUrl\":\"$SINK/api/webhooks/1180000000000000000/discord-token\"}}" '{nodes.home.url}/api/v1/notifications/connections' | jq -r .id)
  $FW doctor $RUN
  ```

- **Grab and import.** Add The Quiet Orchard and search it, then wait for the Discord request. The Imported message is the first, since Grabbed isn't a recommended event. `open` copies its poster into the proof dir.

  ```sh
  W=$($FW exec $RUN -- curl -sS -X POST -H 'X-Api-Key: {apiKey}' -H 'Content-Type: application/json' --data-binary '{"contentType":"movie","key":"9000001","preset":"all"}' '{nodes.home.url}/api/v1/works' | jq -r .work.id)
  $FW exec $RUN -- curl -sS -X POST -H 'X-Api-Key: {apiKey}' "{nodes.home.url}/api/v1/works/$W/search"
  SEQ=$($FW notify wait $RUN --where profile=discord --where status=200 --timeout 5m | jq .seq)
  $FW notify list $RUN --after $((SEQ - 1)) | jq -r .data.dump | head -12
  $FW notify open $RUN $SEQ --proof notifications/notifications-imported
  ```

- **Pause it.** Add `works_added` to the connection's events, turn `discord-gone` on and add Lanterns at Low Tide. The sink answers 404, and the connection reads paused with one message waiting. Add Saltmarsh Radio: two wait.

  ```sh
  $FW exec $RUN -- curl -sS -X PATCH -H 'X-Api-Key: {apiKey}' -H 'Content-Type: application/json' --data-binary '{"events":["grab_failed","health_item_cleared","imported","new_health_item","upgraded","works_added"]}' "{nodes.home.url}/api/v1/notifications/connections/$C"
  $FW cue $RUN discord-gone
  $FW exec $RUN -- curl -sS -o /dev/null -w '%{{http_code}\n' -X POST -H 'X-Api-Key: {apiKey}' -H 'Content-Type: application/json' --data-binary '{"contentType":"movie","key":"9000101","preset":"none"}' '{nodes.home.url}/api/v1/works'
  GONE=$($FW notify wait $RUN --where profile=discord --where status=404 --timeout 5m | jq .seq)
  $FW exec $RUN --proof notifications/notifications-paused -- curl -sS -H 'X-Api-Key: {apiKey}' "{nodes.home.url}/api/v1/notifications/connections/$C"
  $FW exec $RUN -- curl -sS -o /dev/null -w '%{{http_code}\n' -X POST -H 'X-Api-Key: {apiKey}' -H 'Content-Type: application/json' --data-binary '{"contentType":"movie","key":"9000102","preset":"none"}' '{nodes.home.url}/api/v1/works'
  ```

- **Test it.** Poll the connection until `status.waiting` is 2, turn the cue off and Test. The Test answers `{"node":"home"}`, and the sink then takes `Test from SlipStream`, `Added Lanterns at Low Tide …` and `Added Saltmarsh Radio …` in that order, each 200. The connection reads `ok` with nothing waiting.

  ```sh
  $FW cue $RUN discord-gone '{"off": true}'
  $FW exec $RUN --proof notifications/notifications-drain -- curl -sS -X POST -H 'X-Api-Key: {apiKey}' -H 'Content-Type: application/json' --data-binary '{}' "{nodes.home.url}/api/v1/notifications/connections/$C/test"
  $FW notify list $RUN --after $GONE | jq -r '[.seq, .data.status, (.data.dump | capture("\"title\": \"(?<t>[^\"]*)\"").t)] | @tsv'
  ```

- **Read the bell.** The feed lists the events newest first. Marking them seen with the API key is refused; the signed-in bell does it, as in [Shell](./shell.md).

  ```sh
  $FW exec $RUN --proof notifications/notifications-bell -- curl -sS -H 'X-Api-Key: {apiKey}' '{nodes.home.url}/api/v1/notifications/feed'
  $FW exec $RUN --proof notifications/notifications-bell -- curl -sS -X POST -H 'X-Api-Key: {apiKey}' -H 'Content-Type: application/json' --data-binary '{"through":1}' '{nodes.home.url}/api/v1/notifications/feed/seen'
  ```

- **A Scenario's connections.** A Scenario with `notify: [discord, script]` launches with both added. A Test of each lands a `notify` record and a `script` record from home. `fakes/world/notify_test.go` drives it.

## Gotchas

- `fakeworld exec` fills `{...}` tokens, so curl's `-w '%{http_code}'` must be written `'%{{http_code}'`, or exec refuses it as a token the manifest doesn't name.
- Works added waits out its grouping window before it sends, a few minutes in a `home` run, so poll the connection's `waiting` rather than sleeping a fixed time.
- A cue stays on until `{"off": true}`. A paused connection stays paused after the cue is off, until a Test of the saved connection passes or the Admin saves it.
- `discord` captures answer 200, not 204, because SlipStream posts with `?wait=true`.
- `feed/seen` belongs to an Account. Drive it from the signed-in session, never with the API key.
- The `script` capture holds runs only on a Scenario that lists `script`, since the world builds `capture-script` only then.
