# API

A script's path into SlipStream is `/api/v1` with the API key in `X-Api-Key`. Recipes call it with literal `curl` lines through `fakeworld exec`, so a reviewer can rerun the exact request.

```sh
$FW exec $RUN --proof <f>/<s> -- curl -sS -H 'X-Api-Key: {apiKey}' '{nodes.home.url}/api/v1/grabs'
$FW exec $RUN --proof <f>/<s> -- curl -sS -i -X POST -H 'X-Api-Key: {apiKey}' '{nodes.home.url}/api/v1/join-codes'
```

- `{apiKey}` is the manifest's key for the Scenario's Admin. `{nodes.<n>.url}` is the Node's HTTP address; the API needs no `.localhost` name.
- An API read is the second view of every UI proof: after the screen shows a state, read the same state over the API with `--proof`, so the response sits beside the screenshot.
- `-i` keeps the status line and headers in the proof. Leave it off when the body is the proof.
- The routes are in `web/openapi.json`. Reads the skeleton ships: `GET /api/v1/system` (the Nodes Overview), `GET /api/v1/status` (`{"claim":...}`, no key needed), `GET /api/v1/nodes`, `GET /api/v1/works`, `GET /api/v1/works/{id}/searches`, `GET /api/v1/grabs`, `GET /api/v1/grabs/{id}`, `GET /api/v1/indexers`.
- Ids are drawn fresh in each run (UUIDv7), so a recipe never hard-codes one. Find an id from a list read, and address the UI by accessible name.

## The live stream

The supervisor signs the Admin in once at launch, opens `GET /api/v1/events` with that session, and logs every message as an `sse.message` record. Read it with the `sse` reader:

```sh
$FW sse list $RUN --as admin --event grabs.changed
$FW sse wait $RUN --as admin --event grabs.changed --after <seq> --timeout 60s
```

`wait` prints the first matching record after `--after`, by default after the previous step's start, and exits 1 on timeout with the last five records. The record's `data.event` is the event name and `data.data` its payload as text. The stream opens with `live.resync`; the stream re-opens with `Last-Event-ID` after its Node restarts. A browser's reaction to the stream is proved on the screen, not here.
