# API key

A script reaches SlipStream with the API key in `X-Api-Key`. Settings > API key shows when the key was made and regenerates it: the new key is shown once with Copy, and the old one is refused from that moment.

## Sub-features

- `api-key-made` Settings > API key reads when the key was made, and `GET /api/v1/api-key` answers the same time (P-240).
- `api-key-regenerate` `Regenerate` asks `Regenerate the API key?`, then shows the new key once in a read-only field with `Copy New API key` (P-240).
- `api-key-old-refused` after regenerating, the old key answers 401 and the new one 200.

## How to get to it (user POV)

- Desktop shell: the sidebar's `Settings`, then `API key`, `/settings/api-key`. Phone shell: `More`, then `Settings`.
- API: `GET /api/v1/api-key` (Admin) and `POST /api/v1/api-key` (Reconfirmed), which answers the new key once.

## Driving it with fakeworld and playwright cli

Preconditions:

- A `home` run with doctor passing and the `{runId}-admin-desktop` session. Run this recipe last: it replaces the manifest's `{apiKey}`.

Steps:

- **Read when it was made.**

  ```sh
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop goto '{nodes.home.browserUrl}/settings/api-key'
  $FW exec $RUN --proof api-key/api-key-made -- $PW cli -s={runId}-admin-desktop screenshot --filename={runDir}/proof/api-key/api-key-made/settings.png
  $FW exec $RUN --proof api-key/api-key-made -- curl -sS -H 'X-Api-Key: {apiKey}' '{nodes.home.url}/api/v1/api-key'
  ```

- **Regenerate.** Choose `Regenerate`, then `Regenerate` in the dialog. The page shows the new key with `Copy New API key`.

  ```sh
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop click 'getByRole("main").getByRole("button", { name: "Regenerate" })'
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop click 'getByRole("dialog", { name: "Regenerate the API key?" }).getByRole("button", { name: "Regenerate" })'
  $FW exec $RUN --proof api-key/api-key-regenerate -- $PW cli -s={runId}-admin-desktop screenshot --filename={runDir}/proof/api-key/api-key-regenerate/new-key.png
  ```

- **The old key stops.** The manifest's key now answers 401.

  ```sh
  $FW exec $RUN --proof api-key/api-key-old-refused -- curl -sS -i -H 'X-Api-Key: {apiKey}' '{nodes.home.url}/api/v1/api-key'
  ```

## Gotchas

- `{apiKey}` is the manifest's key from launch and nothing rewrites it, so every recipe after a regenerate that uses it fails with 401, and doctor's check 6 fails. Regenerate last, or on its own run.
- Regenerating is Reconfirmed; past the re-confirm window it asks for the PIN first ([re-confirm](./reconfirm.md)).
