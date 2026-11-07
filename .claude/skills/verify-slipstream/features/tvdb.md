# TVDB

TVDB is the series' metadata source and a core provider. Settings > Metadata lists it with its status, the key in use, Set key with an optional subscriber PIN, Remove key, Test and TheTVDB's attribution. A key and PIN are tested by a login before they are kept, and a refusal shows on the field it is about. SlipStream logs in once per key, reuses the token, and logs in again once when a token expires.

## Sub-features

- `tvdb-card` the `TVDB` card reads `TVDB is working.`, `Your key, set <day month>` (the world sets an Admin key), `Change key`, `Remove key`, `Test`, `Get a key from TVDB` and `Metadata provided by TheTVDB.` (P-055).
- `tvdb-key-pin` `Change key` opens `Set your TVDB key` with `Key` and `PIN`; a key and PIN fake-tvdb takes close it and toast `SlipStream now calls TVDB with your key and PIN.` (P-055).
- `tvdb-key-refused` once fake-tvdb rejects a key, saving it keeps the dialog open with `The metadata provider refused that key, so SlipStream didn't save it` on `Key`, and nothing is kept.
- `tvdb-pin-refused` once fake-tvdb rejects PINs, saving a key with a PIN shows `The metadata provider refused that PIN, so SlipStream didn't save it` on `PIN`, and the card's `Test` toasts `TVDB refused your PIN. Check it and save it again with your key, or remove your key.`
- `tvdb-relogin` after `tvdb-expire-tokens`, the next series search logs in again once and answers its results.

## How to get to it (user POV)

- Desktop shell: Settings in the sidebar, then Metadata, the Providers block's `TVDB` card.
- Phone shell: More, then Settings, then Metadata.
- API: `GET /api/v1/metadata/providers`, `PUT` and `DELETE /api/v1/metadata/providers/tvdb/key {key, pin}`, `POST /api/v1/metadata/providers/tvdb/test`.

## Driving it with fakeworld, curl and playwright cli

Preconditions:

- A `series` run, doctor passing, `{runId}-admin-desktop` signed in.

Steps:

- **The card and a key with a PIN.**

  ```sh
  KEYS='getByRole("dialog", { name: "Set your TVDB key" })'
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop goto '{nodes.home.browserUrl}/settings/metadata'
  $FW exec $RUN --proof tvdb/tvdb-card -- $PW cli -s={runId}-admin-desktop snapshot --filename={runDir}/proof/tvdb/tvdb-card/card.aria.yml
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop click 'getByRole("article", { name: "TVDB" }).getByRole("button", { name: "Change key" })'
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop fill "$KEYS"'.getByRole("textbox", { name: "Key" })' e2e-tvdb-key
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop fill "$KEYS"'.getByRole("textbox", { name: "PIN" })' 1234
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop click "$KEYS"'.getByRole("button", { name: "Test and save" })'
  $FW exec $RUN --proof tvdb/tvdb-key-pin -- $PW cli -s={runId}-admin-desktop snapshot --filename={runDir}/proof/tvdb/tvdb-key-pin/saved.aria.yml
  ```

- **A refused key, then a refused PIN.**

  ```sh
  $FW cue $RUN tvdb-reject-key '{"key":"e2e-refused-key"}'
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop click 'getByRole("article", { name: "TVDB" }).getByRole("button", { name: "Change key" })'
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop fill "$KEYS"'.getByRole("textbox", { name: "Key" })' e2e-refused-key
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop click "$KEYS"'.getByRole("button", { name: "Test and save" })'
  $FW exec $RUN --proof tvdb/tvdb-key-refused -- $PW cli -s={runId}-admin-desktop snapshot --filename={runDir}/proof/tvdb/tvdb-key-refused/dialog.aria.yml
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop click "$KEYS"'.getByRole("button", { name: "Cancel" })'
  $FW cue $RUN tvdb-reject-pin '{}'
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop click 'getByRole("article", { name: "TVDB" }).getByRole("button", { name: "Change key" })'
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop fill "$KEYS"'.getByRole("textbox", { name: "Key" })' e2e-tvdb-key
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop fill "$KEYS"'.getByRole("textbox", { name: "PIN" })' 9999
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop click "$KEYS"'.getByRole("button", { name: "Test and save" })'
  $FW exec $RUN --proof tvdb/tvdb-pin-refused -- $PW cli -s={runId}-admin-desktop snapshot --filename={runDir}/proof/tvdb/tvdb-pin-refused/dialog.aria.yml
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop click "$KEYS"'.getByRole("button", { name: "Cancel" })'
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop click 'getByRole("article", { name: "TVDB" }).getByRole("button", { name: "Test" })'
  $FW exec $RUN --proof tvdb/tvdb-pin-refused -- $PW cli -s={runId}-admin-desktop snapshot --filename={runDir}/proof/tvdb/tvdb-pin-refused/test.aria.yml
  ```

- **Token expiry.** Watch logins with a pass-through fault rule, end every token, and search.

  ```sh
  $FW fault set $RUN logins tvdb slow-body=1GiB/s --route login
  $FW cue $RUN tvdb-expire-tokens '{}'
  $FW exec $RUN --proof tvdb/tvdb-relogin -- curl -sS -H 'X-Api-Key: {apiKey}' '{nodes.home.url}/api/v1/content-types/series/catalog?query=harbor'
  $FW fault wait $RUN --rule logins --timeout 10s
  $FW fault clear $RUN logins
  ```

## Gotchas

- `tvdb-reject-pin` with `{}` rejects the PIN beside every key and ends their tokens, so run it after the key steps and expect the card's status to turn.
- The field errors read the problem's title, which names the field but not TVDB; the problem's `detail` holds TVDB's words.
- A fault rule with `slow-body=1GiB/s` changes nothing a test can see except that it fires, which is what counts the logins.
