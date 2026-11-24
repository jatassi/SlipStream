# Remote access

An Admin sets the UI and portal, or a Media Server, up for use away from home, and SlipStream checks each one's Route: from a paired Node outside the home it connects to the public address as a player would, and judges each Route Reachable, Fixable or Needs a relay with the check that decided it and what to do. Each Node's network is read for the check, and a Node watches its network only while something is wanted. The Reachability check runs as a Scheduled task, again after a network change, and on Run again. A Route that rests on a failing check for "Warn in Health after a route fails for" raises a warning under Health's Remote access. Until the Remote access page ships, everything here is read and driven through the API.

## Sub-features

- `remote-access-nothing-wanted` with nothing wanted, `GET /api/v1/remote-access` answers no routes, `POST /api/v1/remote-access/run` answers 409 `nothing_wanted`, and no Node watches its network.
- `remote-access-want` `POST /api/v1/remote-access/exposures` with `{"kind":"ui-portal"}`, or `{"kind":"media-server","serverId":"<id>","public":"https://…"}`, answers 201 with the Exposure and starts its check; the same again answers 409 `exposure_exists`, a Media Server with no public address 422 `public_required`, one that isn't an http or https URL 422 `public_not_url`, and an unknown server 404 `server_not_found`.
- `remote-access-route` once the check ran, the route in `GET /api/v1/remote-access` carries its verdict, whether it was tested from outside and by which Node, its deciding check and fix ids, and the status counts. The UI and portal with no Address is skipped, with the Scheduled task's outcome saying so.
- `remote-access-run-again` `POST /api/v1/remote-access/run` answers 202 with the task run, `?exposure=<id>` checks one route, and an unknown id answers 404 `exposure_not_found`.
- `remote-access-settings` `GET` and `PATCH /api/v1/remote-access/settings` read and change "Warn in Health after a route fails for", "Look for network changes every", "Wait after a network change" and "STUN servers"; per Node, `GET` and `PATCH /api/v1/remote-access/nodes/{id}/settings` read and change "Open router ports automatically" and "Stable IPv6 address", which refuses an address starting with fd with 422 `setting_out_of_range`.
- `remote-access-node-network` `GET /api/v1/remote-access/nodes/{id}` answers a Node's interfaces, listeners and firewall read now with nothing sent outside the machine, or its last stored read while it is offline.
- `remote-access-remove` `DELETE /api/v1/remote-access/exposures/{id}` answers 204 and forgets the route's result; `POST /api/v1/remote-access/suggestions/{id}/dismiss` hides a suggestion, `ui-portal` or a Media Server's id.

## How to get to it (user POV)

- Desktop shell and phone shell: none yet. Settings > Access will hold the Remote access section and each Node's page its Network section.
- API (Admin): `GET /api/v1/remote-access`, `POST /api/v1/remote-access/exposures`, `PATCH` and `DELETE /api/v1/remote-access/exposures/{id}`, `POST /api/v1/remote-access/suggestions/{id}/dismiss`, `POST /api/v1/remote-access/run`, `GET /api/v1/remote-access/nodes/{id}`, `GET` and `PATCH /api/v1/remote-access/nodes/{id}/settings`, `GET` and `PATCH /api/v1/remote-access/settings`; the live event `reachability.changed`. `GET /api/v1/remote-access/echo/{id}` is public: a test fetches it once.
- Host CLI: none.

## Driving it with fakeworld and curl

Preconditions:

- A `home` run with doctor passing. `home` has no `network` block, so its Nodes reach no STUN server, gateway or resolver outside the world's sink, and every route reads untested.

Steps:

- **Nothing wanted.**

  ```sh
  $FW exec $RUN --proof remote-access/remote-access-nothing-wanted -- curl -sS -H 'X-Api-Key: {apiKey}' '{nodes.home.url}/api/v1/remote-access'
  $FW exec $RUN --proof remote-access/remote-access-nothing-wanted -- curl -sS -i -X POST -H 'X-Api-Key: {apiKey}' '{nodes.home.url}/api/v1/remote-access/run'
  ```

- **Want the UI and portal.** The answer is 201 with the Exposure's id; a second POST answers 409 `exposure_exists`.

  ```sh
  $FW exec $RUN --proof remote-access/remote-access-want -- curl -sS -i -X POST -H 'X-Api-Key: {apiKey}' -H 'Content-Type: application/json' -d '{"kind":"ui-portal"}' '{nodes.home.url}/api/v1/remote-access/exposures'
  ```

- **Read the route.** Once the Reachability check's run ends, the route has a verdict and `tested` false.

  ```sh
  $FW exec $RUN --proof remote-access/remote-access-route -- curl -sS -H 'X-Api-Key: {apiKey}' '{nodes.home.url}/api/v1/remote-access'
  ```

- **Run again, read a Node's network, change a setting, remove the route.**

  ```sh
  $FW exec $RUN --proof remote-access/remote-access-run-again -- curl -sS -i -X POST -H 'X-Api-Key: {apiKey}' '{nodes.home.url}/api/v1/remote-access/run'
  $FW exec $RUN --proof remote-access/remote-access-node-network -- curl -sS -H 'X-Api-Key: {apiKey}' '{nodes.home.url}/api/v1/remote-access/nodes/<home id>'
  $FW exec $RUN --proof remote-access/remote-access-settings -- curl -sS -X PATCH -H 'X-Api-Key: {apiKey}' -H 'Content-Type: application/json' -d '{"values":{"reachability.warnAfter":3600}}' '{nodes.home.url}/api/v1/remote-access/settings'
  $FW exec $RUN --proof remote-access/remote-access-remove -- curl -sS -i -X DELETE -H 'X-Api-Key: {apiKey}' '{nodes.home.url}/api/v1/remote-access/exposures/<id>'
  ```

## Gotchas

- Every verdict on `home` is untested, because its Nodes reach nothing outside. Verdicts tested from outside need a `remote-*` Scenario, whose `network` block the network fakes play. Ports are drawn at launch, so a public address names the port atlas listens on: `http://slipstream.open.localhost:<atlas's http port>` for Address on `remote-open`, and Plex's front for atlas as the Plex route's port. `remote-own-proxy` takes `https://slipstream.proxy.fake.localhost` and `https://plex.proxy.fake.localhost` as they are.
- `fakeworld cue $RUN network '{"site":"home","wan":"203.0.113.10"}'` moves a site's address; a Node sees it on its next watch pass, at least a minute away, so set "Look for network changes every" to 60 and "Wait after a network change" to 0 first. Plex's own check runs every 15 seconds and leaves a `plextv.connect-back` event; `fakeworld cue $RUN plex-mapping '{"server":"Plex","state":"mapped"}'` sets its result instead, and `fakeworld fault set $RUN the-vps proxy@atlas tls=expired` (or `tls=wrong-name`, `status=401`) breaks the Admin's proxy as atlas reaches it.
- A route is judged by a Scheduled task run, so read it after the run ends: `GET /api/v1/tasks` shows the Reachability check's last run.
- The UI and portal's check is skipped until Settings > Address holds the https:// name outsiders use.
