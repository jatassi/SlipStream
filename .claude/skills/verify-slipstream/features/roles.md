# Roles

An Admin changes which Node holds which Role through one plan. The Role board shows every Node's Roles and the plan's verdict on every drag. A plan answers what a set would do in words, and a set stores it once the guardrails allow it. A Role that SlipStream always needs goes back to the Controller when its last holder lets it go. A set planned on rows that changed since, or one a guardrail blocks, answers 409 with the plan as it stands, and nothing is stored.

## Sub-features

- `roles-board` `GET /api/v1/nodes/roles` answers the `basis`, each Node with `assigned` and its `roles` with a `state` each, and `gestures`, the verdict on every move and add.
- `roles-plan` `POST /api/v1/nodes/roles/plan` answers the Outcome, `effects` and `checks` in words, and stores nothing.
- `roles-set` `PUT /api/v1/nodes/roles` stores the set and answers once the Nodes in step it changed hold their new Roles. Unticking seedbox's Downloading hands it to home: `home takes over Downloading, since SlipStream always needs a Node that downloads.`
- `roles-stale` a plan or set on an old `basis` answers 409 `roles_stale` with `outcome`, the same change planned on the rows now.
- `roles-blocked` adding Relaying to a second Node answers 409 `roles_blocked`, whose check reads `seedbox already relays. Drag Relaying to move it.`
- `roles-roster` `GET /api/v1/nodes` lists each Node's `roles` as `{role, state}`, `holding` while it is in step.

## How to get to it (user POV)

- API on the Controller (Admin): `GET /api/v1/nodes/roles`, `POST /api/v1/nodes/roles/plan` and `PUT /api/v1/nodes/roles` with `{"basis": …, "set": [{"nodeId": …, "roles": […]}]}`, and `GET /api/v1/nodes`.
- Desktop and phone: no Roles screen yet. The Roles section and its Review come later in the Nodes, Pairing and Roles build.

## Driving it with fakeworld and curl

Preconditions:

- A `home` run with doctor passing. Run this last or on its own run: it changes seedbox's Roles, so doctor's check 3 then reports them.

Steps:

- **Read the board.** `home` is the Controller, `seedbox` has `assigned` `searching` and `downloading`, both `holding`.

  ```sh
  BASIS=$($FW exec $RUN --proof roles/roles-board -- curl -sS -H 'X-Api-Key: {apiKey}' '{nodes.home.url}/api/v1/nodes/roles' | jq -r .basis)
  ```

- **Plan unticking Downloading on seedbox.** The Outcome's effects read `seedbox stops Downloading.` and the handback to home.

  ```sh
  $FW exec $RUN --proof roles/roles-plan -- curl -sS -X POST -H 'X-Api-Key: {apiKey}' -H 'Content-Type: application/json' --data-binary "{\"basis\":\"$BASIS\",\"set\":[{\"nodeId\":\"{nodes.seedbox.nodeId}\",\"roles\":[\"searching\"]}]}" '{nodes.home.url}/api/v1/nodes/roles/plan'
  ```

- **Apply it.** The answer is the same Outcome. The roster then lists `seedbox` holding only `searching` and `home` holding `downloading`, each with its facts at its `intentVersion`.

  ```sh
  $FW exec $RUN --proof roles/roles-set -- curl -sS -X PUT -H 'X-Api-Key: {apiKey}' -H 'Content-Type: application/json' --data-binary "{\"basis\":\"$BASIS\",\"set\":[{\"nodeId\":\"{nodes.seedbox.nodeId}\",\"roles\":[\"searching\"]}]}" '{nodes.home.url}/api/v1/nodes/roles'
  $FW exec $RUN --proof roles/roles-roster -- curl -sS -H 'X-Api-Key: {apiKey}' '{nodes.home.url}/api/v1/nodes'
  ```

- **Set again on the old basis.** 409 `roles_stale`, whose `outcome.basis` is the new one.

  ```sh
  $FW exec $RUN --proof roles/roles-stale -- curl -sS -i -X PUT -H 'X-Api-Key: {apiKey}' -H 'Content-Type: application/json' --data-binary "{\"basis\":\"$BASIS\",\"set\":[{\"nodeId\":\"{nodes.seedbox.nodeId}\",\"roles\":[]}]}" '{nodes.home.url}/api/v1/nodes/roles'
  ```

- **Add Relaying to seedbox, then to home.** The first set answers 200. The second answers 409 `roles_blocked` with the `role_taken` check, and home holds no Relaying.

  ```sh
  BASIS=$($FW exec $RUN -- curl -sS -H 'X-Api-Key: {apiKey}' '{nodes.home.url}/api/v1/nodes/roles' | jq -r .basis)
  $FW exec $RUN --proof roles/roles-blocked -- curl -sS -X PUT -H 'X-Api-Key: {apiKey}' -H 'Content-Type: application/json' --data-binary "{\"basis\":\"$BASIS\",\"set\":[{\"nodeId\":\"{nodes.seedbox.nodeId}\",\"roles\":[\"searching\",\"relaying\"]}]}" '{nodes.home.url}/api/v1/nodes/roles'
  BASIS=$($FW exec $RUN -- curl -sS -H 'X-Api-Key: {apiKey}' '{nodes.home.url}/api/v1/nodes/roles' | jq -r .basis)
  $FW exec $RUN --proof roles/roles-blocked -- curl -sS -i -X PUT -H 'X-Api-Key: {apiKey}' -H 'Content-Type: application/json' --data-binary "{\"basis\":\"$BASIS\",\"set\":[{\"nodeId\":\"{nodes.home.nodeId}\",\"roles\":[\"downloading\",\"storage\",\"converting\",\"subtitles\",\"metadata\",\"media-server-link\",\"notifying\",\"relaying\",\"serving\"]}]}" '{nodes.home.url}/api/v1/nodes/roles'
  ```

## Gotchas

- Every set needs the `basis` of the board or Outcome it was planned on. Read it again after each set; the old one answers `roles_stale`.
- A Node the set doesn't name keeps its Roles. A Role SlipStream always needs comes back to home whenever no Node in the set holds it, and Serving stays on home whatever the set says.
- A set naming a Node that is down answers at once; the Node gets its Roles when its Sync session attaches again.
