# Metadata providers

Each metadata provider, TMDB and OMDb, says how it stands in one plain sentence, whose key it is called with (SlipStream's, the Admin's, or none in this build) and, under Details, its last success, its last error and which Nodes reached it. An Admin can set their own key, which a holder tests before it is sealed and kept, remove it to go back to SlipStream's, and test a provider on demand. The provider key screen belongs to S20 (Library, the add flow and the Work page); until then this feature is API only.

## Sub-features

- `providers-status` `GET /api/v1/metadata/providers` lists TMDB then OMDb, each with its summary, key in use, state and Details, `unused` until a call or test, `keySetAt` null, its `keyPage` (`https://www.tmdb.invalid/settings/api` and `https://www.omdb.invalid/apikey.aspx` in the Fake world), and TMDB's `attribution` `This product uses the TMDB API but is not endorsed or certified by TMDB.` with logo `tmdb`, OMDb's null (P-061).
- `key-refused` a key the provider refuses answers 422 `metadata_key_refused` with the provider's words, and nothing is kept (P-063).
- `key-saved` a key the provider takes is kept and in use at once: the provider reads `yours` and `working` with `keySetAt` the time it was saved, and `metadata.changed` fires (P-063).
- `provider-test` `Test` makes one cheap call with the key in use and answers the state it found, such as OMDb out of today's allowance until UTC midnight (P-061).
- `key-cleared` removing the key goes back to SlipStream's, or none in a build without one, with `keySetAt` null, and starts the status afresh (P-063).
- `bundled-key` a release built with `-bundle-keys` reads `bundled` for both providers and sends that key: revoking it at the fake turns TMDB to `key_refused` with "TMDB refused SlipStream's key. Add your own." (P-063).

## How to get to it (user POV)

- Desktop and phone shells: no screen yet; S20 (Library, the add flow and the Work page) builds it.
- API: `GET /api/v1/metadata/providers`, `PUT` and `DELETE /api/v1/metadata/providers/{provider}/key`, `POST /api/v1/metadata/providers/{provider}/test` (Admin). The live stream carries `metadata.changed`.

## Driving it with fakeworld and curl

Preconditions:

- A `home` run with doctor passing. `bundled-key` needs its own run from a keyed build: `SLIPSTREAM_TMDB_KEY=bundledtmdb7 SLIPSTREAM_OMDB_KEY=bundledomdb7 go run ./cmd/release build -bundle-keys -o <dir>`, then `dist/fakeworld start --binary <dir>/slipstream`.

Steps:

- **Read the status.** Both read `"key":"none"`, `"keySetAt":null`, `"state":"unused"` and "This build has no TMDB key. Add your own.", with each `keyPage` and TMDB's `attribution`.

  ```sh
  $FW exec $RUN --proof metadata-providers/providers-status -- curl -sS -H 'X-Api-Key: {apiKey}' '{nodes.home.url}/api/v1/metadata/providers'
  ```

- **A refused key.** Revoke `typo` at fake-tmdb, then save it: 422 `metadata_key_refused`, detail "TMDB refused the key: Invalid API key: You must be granted a valid key."

  ```sh
  $FW cue $RUN tmdb-revoke-key '{"key":"typo"}'
  $FW exec $RUN --proof metadata-providers/key-refused -- curl -sS -i -X PUT -H 'X-Api-Key: {apiKey}' -H 'Content-Type: application/json' -d '{"key":"typo"}' '{nodes.home.url}/api/v1/metadata/providers/tmdb/key'
  ```

- **A good key.** 200 with `"key":"yours"`, `keySetAt` now, `"state":"working"` and home in Details' nodes; the stream carries one `metadata.changed` for `tmdb`.

  ```sh
  $FW exec $RUN --proof metadata-providers/key-saved -- curl -sS -i -X PUT -H 'X-Api-Key: {apiKey}' -H 'Content-Type: application/json' -d '{"key":"mine"}' '{nodes.home.url}/api/v1/metadata/providers/tmdb/key'
  $FW sse list $RUN --as admin --event metadata.changed
  ```

- **Test OMDb out of its allowance.** `"state":"out_of_allowance"`, `until` the next UTC midnight.

  ```sh
  $FW cue $RUN omdb-cap '{}'
  $FW exec $RUN --proof metadata-providers/provider-test -- curl -sS -X POST -H 'X-Api-Key: {apiKey}' '{nodes.home.url}/api/v1/metadata/providers/omdb/test'
  ```

- **Remove the key.** TMDB reads `"key":"none","keySetAt":null` and `"state":"unused"` again.

  ```sh
  $FW exec $RUN --proof metadata-providers/key-cleared -- curl -sS -X DELETE -H 'X-Api-Key: {apiKey}' '{nodes.home.url}/api/v1/metadata/providers/tmdb/key'
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
