# Metadata providers

Each metadata provider, TMDB and OMDb, says how it stands in one plain sentence, whose key it is called with (SlipStream's, the Admin's, or none in this build) and, under Details, its last success, its last error and which Nodes reached it. An Admin can set their own key, which a holder tests before it is sealed and kept, remove it to go back to SlipStream's, and test a provider on demand. Settings > Metadata leads with a Providers block that does all of this, links to each provider's key page and shows TMDB's required attribution with TMDB's logo.

## Sub-features

- `providers-status` `GET /api/v1/metadata/providers` lists TMDB then OMDb, each with its summary, key in use, state and Details, `unused` until a call or test (P-061).
- `key-refused` a key the provider refuses answers 422 `metadata_key_refused` with the provider's words, and nothing is kept (P-063).
- `key-saved` a key the provider takes is kept and in use at once: the provider reads `yours` and `working`, and `metadata.changed` fires (P-063).
- `provider-test` `Test` makes one cheap call with the key in use and answers the state it found, such as OMDb out of today's allowance until UTC midnight (P-061).
- `key-cleared` removing the key goes back to SlipStream's, or none in a build without one, and starts the status afresh (P-063).
- `providers-wire` each provider also carries `keySetAt` (null unless the key is the Admin's), `keyPage` (TMDB's and OMDb's key pages, under `.invalid` hosts in a Fake world) and `attribution`, which is TMDB's sentence "This product uses the TMDB API but is not endorsed or certified by TMDB" with logo `tmdb`, and null for OMDb.
- `providers-screen` Settings > Metadata on desktop and phone shows a Providers block above the Metadata settings: one card per provider with its summary sentence, "No key", "SlipStream's key" or "Your key, set <day month>", Set key (Change key while the key is the Admin's), Remove key only then, Test, and "Get a TMDB key" linking to `keyPage` in a new tab. TMDB's card ends with the attribution sentence beside TMDB's logo, bundled with the web app (P-061).
- `key-dialog` Set key opens "Set your TMDB key". Test and save with a key the provider refuses keeps the dialog open with "The metadata provider refused that key, so SlipStream didn't save it" and nothing is kept; a key it takes closes the dialog, toasts "TMDB now uses your key." and the card reads "Your key, set <today>". Remove key asks "Remove your TMDB key?" first. Test toasts the provider's summary after the call (P-063).
- `bundled-key` a release built with `-bundle-keys` reads `bundled` for both providers and sends that key: revoking it at the fake turns TMDB to `key_refused` with "TMDB refused SlipStream's key. Add your own." (P-063).

## How to get to it (user POV)

- Desktop shell: Settings in the sidebar, then Metadata.
- Phone shell: More, then Settings, then Metadata.
- API: `GET /api/v1/metadata/providers`, `PUT` and `DELETE /api/v1/metadata/providers/{provider}/key`, `POST /api/v1/metadata/providers/{provider}/test` (Admin). The live stream carries `metadata.changed`.

## Driving it with fakeworld, curl and playwright cli

Preconditions:

- A `home` run with doctor passing, and the `{runId}-admin-desktop` and `{runId}-admin-phone` sessions for the screen steps. `bundled-key` needs its own run from a keyed build: `SLIPSTREAM_TMDB_KEY=bundledtmdb7 SLIPSTREAM_OMDB_KEY=bundledomdb7 go run ./cmd/release build -bundle-keys -o <dir>`, then `dist/fakeworld start --binary <dir>/slipstream`.

Steps:

- **Read the status.** Both read `"key":"none"`, `"state":"unused"` and "This build has no TMDB key. Add your own."

  ```sh
  $FW exec $RUN --proof metadata-providers/providers-status -- curl -sS -H 'X-Api-Key: {apiKey}' '{nodes.home.url}/api/v1/metadata/providers'
  ```

- **A refused key.** Revoke `typo` at fake-tmdb, then save it: 422 `metadata_key_refused`, detail "TMDB refused the key: Invalid API key: You must be granted a valid key."

  ```sh
  $FW cue $RUN tmdb-revoke-key '{"key":"typo"}'
  $FW exec $RUN --proof metadata-providers/key-refused -- curl -sS -i -X PUT -H 'X-Api-Key: {apiKey}' -H 'Content-Type: application/json' -d '{"key":"typo"}' '{nodes.home.url}/api/v1/metadata/providers/tmdb/key'
  ```

- **A good key.** 200 with `"key":"yours","state":"working"` and home in Details' nodes; the stream carries one `metadata.changed` for `tmdb`.

  ```sh
  $FW exec $RUN --proof metadata-providers/key-saved -- curl -sS -i -X PUT -H 'X-Api-Key: {apiKey}' -H 'Content-Type: application/json' -d '{"key":"mine"}' '{nodes.home.url}/api/v1/metadata/providers/tmdb/key'
  $FW sse list $RUN --as admin --event metadata.changed
  ```

- **Test OMDb out of its allowance.** `"state":"out_of_allowance"`, `until` the next UTC midnight.

  ```sh
  $FW cue $RUN omdb-cap '{}'
  $FW exec $RUN --proof metadata-providers/provider-test -- curl -sS -X POST -H 'X-Api-Key: {apiKey}' '{nodes.home.url}/api/v1/metadata/providers/omdb/test'
  ```

- **Remove the key.** TMDB reads `"key":"none","state":"unused"` again.

  ```sh
  $FW exec $RUN --proof metadata-providers/key-cleared -- curl -sS -X DELETE -H 'X-Api-Key: {apiKey}' '{nodes.home.url}/api/v1/metadata/providers/tmdb/key'
  ```

- **The screen.** Open Settings > Metadata, refuse a key in the dialog, then save a good one, test OMDb and remove the key.

  ```sh
  $FW exec $RUN --proof metadata-providers/providers-wire -- curl -sS -H 'X-Api-Key: {apiKey}' '{nodes.home.url}/api/v1/metadata/providers'
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop goto {nodes.home.browserUrl}/settings/metadata
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop snapshot --filename={runDir}/proof/metadata-providers/providers-screen/desktop.aria.yml
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop screenshot --filename={runDir}/proof/metadata-providers/providers-screen/desktop.png
  $FW cue $RUN tmdb-revoke-key '{"key":"typo"}'
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop click 'getByRole("article", { name: "TMDB" }).getByRole("button", { name: "Set key" })'
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop fill 'getByRole("dialog").getByRole("textbox", { name: "Key" })' typo
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop click 'getByRole("button", { name: "Test and save" })'
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop snapshot --filename={runDir}/proof/metadata-providers/key-dialog/refused.aria.yml
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop fill 'getByRole("dialog").getByRole("textbox", { name: "Key" })' mine
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop click 'getByRole("button", { name: "Test and save" })'
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop snapshot --filename={runDir}/proof/metadata-providers/key-dialog/saved.aria.yml
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop click 'getByRole("article", { name: "TMDB" }).getByRole("button", { name: "Remove key" })'
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop click 'getByRole("dialog").getByRole("button", { name: "Remove key" })'
  $FW exec $RUN -- $PW cli -s={runId}-admin-phone goto {nodes.home.browserUrl}/settings/metadata
  $FW exec $RUN -- $PW cli -s={runId}-admin-phone screenshot --filename={runDir}/proof/metadata-providers/providers-screen/phone.png
  ```

- **A bundled key (keyed run).** Both read `"key":"bundled"`; after revoking the bundled key at fake-tmdb, a test reads `key_refused`.

  ```sh
  $FW exec $RUN --proof metadata-providers/bundled-key -- curl -sS -H 'X-Api-Key: {apiKey}' '{nodes.home.url}/api/v1/metadata/providers'
  $FW cue $RUN tmdb-revoke-key '{"key":"bundledtmdb7"}'
  $FW exec $RUN --proof metadata-providers/bundled-key -- curl -sS -X POST -H 'X-Api-Key: {apiKey}' '{nodes.home.url}/api/v1/metadata/providers/tmdb/test'
  ```

## Gotchas

- Status lives in the Controller's memory: a restart starts every provider `unused`, while a saved key survives in library.db.
- An OMDb test spends one call of the day's allowance, so drive it against the fake only.
- fake-tmdb's `tmdb-revoke-key` with `{}` revokes only keys it has already seen; name `{"key":""}` to refuse keyless calls before any call.
- `keySetAt`, `keyPage` and `attribution` are filled by B3 (Add with Remember, metadata screens API). Before it merges they read null, "" and null, so drive the screen steps on a build with B3.
