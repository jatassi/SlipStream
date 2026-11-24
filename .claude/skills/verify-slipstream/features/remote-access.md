# Remote access

An Admin sets the UI and portal, or a Media Server, up for use away from home, and SlipStream checks each one's Route: from a paired Node outside the home it connects to the public address as a player would, and judges each Route Reachable, Fixable or Needs a relay with the check that decided it and what to do. Each Node's network is read for the check, and a Node watches its network only while something is wanted. The Reachability check runs as a Scheduled task, again after a network change, and on Run again. A Route that rests on a failing check for "Warn in Health after a route fails for" raises a warning under Health's Remote access. Nodes › Remote access is the page: a pushed page of Nodes with the status sentence, one recommendation card, each route and the suggestions, and each Node has a Network section.

## Sub-features

- `remote-access-nothing-wanted` with nothing wanted, `GET /api/v1/remote-access` answers no routes, `POST /api/v1/remote-access/run` answers 409 `nothing_wanted`, and no Node watches its network.
- `remote-access-want` `POST /api/v1/remote-access/exposures` with `{"kind":"ui-portal"}`, or `{"kind":"media-server","serverId":"<id>","public":"https://…"}`, answers 201 with the Exposure and starts its check; the same again answers 409 `exposure_exists`, a Media Server with no public address 422 `public_required`, one that isn't an http or https URL 422 `public_not_url`, and an unknown server 404 `server_not_found`.
- `remote-access-route` once the check ran, the route in `GET /api/v1/remote-access` carries its verdict, whether it was tested from outside and by which Node, its deciding check and fix ids, and the status counts. The UI and portal with no Address is skipped, with the Scheduled task's outcome saying so.
- `remote-access-run-again` `POST /api/v1/remote-access/run` answers 202 with the task run, `?exposure=<id>` checks one route, and an unknown id answers 404 `exposure_not_found`.
- `remote-access-settings` `GET` and `PATCH /api/v1/remote-access/settings` read and change "Warn in Health after a route fails for", "Look for network changes every", "Wait after a network change" and "STUN servers"; per Node, `GET` and `PATCH /api/v1/remote-access/nodes/{id}/settings` read and change "Open router ports automatically" and "Stable IPv6 address", which refuses an address starting with fd with 422 `setting_out_of_range`.
- `remote-access-node-network` `GET /api/v1/remote-access/nodes/{id}` answers a Node's interfaces, listeners and firewall read now with nothing sent outside the machine, or its last stored read while it is offline.
- `remote-access-page` `/nodes/remote-access`, titled `Remote access` under the crumbs `Nodes › Remote access` (on phone a pushed page with the back link `Nodes`), leads with the status sentence of counts, such as `1 route works · Living room Plex needs a fix`, whose warning and failure parts link to the card or the route, then `Last run <time> · Tested from <Node>` with `Run again`. Each route reads its subject, public address, `IPv4`, `IPv4 and IPv6` or `IPv6 only`, its route kind (`Direct`, `Through your own proxy`) and its verdict chip, `Reachable`, `Fixable` or `Needs a relay`, in signal colour only when a failing check decides it, with a `Not tested` chip beside an untested one; each opens its definition on hover, tap and Enter. A route that works and was tested is one line; any other is drawn hop by hop (`Player`, `DNS`, `Proxy`, `TLS`, `Origin`, `Media Server`), a failing hop in its signal card and the hops past it dashed `not reached`. The route's menu holds `Run again`, `Change public address…` for a Media Server and `Stop checking…`.
- `remote-access-card` the one recommendation: a route that needs a relay is a red `Needs a relay` card with no button, a route a failing check makes Fixable an amber `Fixable` card with its fix's button (`Turn on Open router ports automatically` turns that setting on for the Node it names), and a route nothing outside the home tested a neutral line with `Pair a Node`, which opens `Add a Node`. With every route Reachable and tested, no card shows. On phone a card is a row that opens a sheet.
- `remote-access-hop` selecting a hop opens `<hop> · <subject>` in the Inspector, or a sheet on phone: each check headed by its result and family, such as `Fails over IPv4`, then what it means and its fix, and under `Details` what was seen and `Seen by <Node> at <time>`.
- `remote-access-empty` with nothing wanted the page reads `Nothing is set up for use away from home.` and `Choose what to use away from home` lists the suggestions: a Media Server with its public address filled in from what it publishes, editable, with `Add` and `Dismiss`, and the UI and portal at Address, or first the Address field when outsiders can't use it.
- `remote-access-class-lines` the page's `Each Node's network` lists one line per Node, the Controller first, such as `home is behind a router that doesn't open ports for SlipStream.`, each opening that Node's Network section.
- `remote-access-network-section` a Node's `Network` section, after `Web and API`, leads with its class line in no signal colour, then `Open router ports automatically` and `Stable IPv6 address`, and under `Details` its addresses, router, what listens on each port per family, the firewall's answer per port and family and its router mappings; an offline Node reads `<name> is offline. This is its network at <time>.` (P-218).
- `remote-access-settings-section` Settings › Access › Remote access, after Address, holds `Open Remote access`, "Warn in Health after a route fails for", "Check every route again every" (while a route is set up), and under `Network changes` "Look for network changes every", "Wait after a network change" and "STUN servers"; the section's top shows Remote access's worst Health item.
- `remote-access-nodes-line` the Nodes landing reads `Remote access · <n> of <m> routes work`, or `Remote access · Nothing set up`, under the topology sentence, opening the page.
- `remote-access-first-setup` first setup's `Remote access` step, after the Targets step, shows only while there is a suggestion or something is set up; each suggestion has a switch, a Media Server asks for its public address, and Continue adds the switched-on ones, checks them and marks the step done.
- `remote-access-remove` `DELETE /api/v1/remote-access/exposures/{id}` answers 204 and forgets the route's result; `POST /api/v1/remote-access/suggestions/{id}/dismiss` hides a suggestion, `ui-portal` or a Media Server's id.

## How to get to it (user POV)

- Desktop shell: the Nodes landing's `Remote access · …` line, or a Node's class line, opens `/nodes/remote-access`; a Node's page tab `Network`; Settings › Access › `Remote access`; first setup's `Remote access` step; a Remote access Health item's link.
- Phone shell: the same line on the Roster's page, pushing `Remote access` with the back link `Nodes`; a hop opens a sheet; a Node's section list holds `Network`.
- Both shells: the palette's `Go to Network` for the browser's own Node.
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

- **The page.** Open it, read the sentence and the card, select a hop and read its checks; on the phone session the hop opens a sheet.

  ```sh
  $FW exec $RUN --proof remote-access/remote-access-page -- $PW cli -s={runId}-admin-desktop goto '{nodes.home.browserUrl}/nodes/remote-access'
  $FW exec $RUN --proof remote-access/remote-access-page -- $PW cli -s={runId}-admin-desktop snapshot
  $FW exec $RUN --proof remote-access/remote-access-network-section -- $PW cli -s={runId}-admin-desktop goto '{nodes.home.browserUrl}/nodes/<home id>/network'
  ```

## Gotchas

- Every verdict on `home` is untested, because its Nodes reach nothing outside. Verdicts tested from outside need a Scenario with a `network` block, which none in the catalog has yet.
- A route is judged by a Scheduled task run, so read it after the run ends: `GET /api/v1/tasks` shows the Reachability check's last run.
- The UI and portal's check is skipped until Settings > Address holds the https:// name outsiders use.
