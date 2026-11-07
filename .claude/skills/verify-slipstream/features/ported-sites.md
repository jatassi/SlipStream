# Ported sites

A ported site is a Tracker whose Indexer runs SlipStream's own port of Prowlarr's code for that site, and it sits in the Add Indexer list beside the catalog's definitions. Its form shows only the fields the site takes, such as an API key, a password, a PID, a browser cookie with its User-Agent or an authenticator key, and says that fixes to it come with SlipStream updates. A sign-in that needs a two-factor code shows a code step in the form and on the Indexer's page. Edit shows each sealed setting as `Saved` with `Replace`, and a new setting or cookie is tested before it saves. The Indexer's page says under Details which Prowlarr code the port came from and how far apart its requests are, and a stop names the setting to fix.

## Sub-features

- `ported-picker` on `ported-sites`, `/settings/indexers/new` searched for `norbits` lists `NorBits` with its media, `Norwegian Bokmål (Norway)` and `Private`; its form reads `SlipStream includes support for NorBits. Fixes to it come with SlipStream updates.` above `Address`, `Username`, `Password`, `Use full search`, `Freeleech only` and `Authenticator key` (P-364)
- `ported-add-code` NorBits added as `NorBits by code` at `Another address` with the authenticator key left empty: `Save` shows `NorBits needs the current code from your authenticator app to sign in.` with `Two-factor code` and the alert `NorBits by code needs a two-factor code to sign in again. Type the current code and test again.`; `Save` with the code from `$FW site $RUN norbits` opens its page reading `NorBits by code is working.` (P-364)
- `ported-page-port` PassThePopcorn's page, the last `Details`, reads `Support` `SlipStream's PassThePopcorn support is ported from Prowlarr's code of <date>.`, `Spacing` `Requests to PassThePopcorn are at least 4 seconds apart.`, `Session` `Doesn't sign in` and `Searches` `0 of 150 per hour`, its port's default limit (P-364)
- `ported-edit-sealed` `Edit Torrent Bytes` shows `Password` as `Saved` with `Replace Password`; a wrong password reads `Torrent Bytes refused the sign-in with 'Username or password incorrect'. Check the username, password and any code, then test again.` and the right one saves, and `GET /api/v1/indexers/{id}` answers `revision` `2` with `sealed` `["password"]`. `Edit IPTorrents`, `Replace Cookie`, the cookie with `User-Agent` `Mozilla/5.0 (another)` reads `IPTorrents no longer accepts the browser cookie. Paste a new one with that browser's User-Agent and test again.`, and with `Mozilla/5.0 (fixture)` it saves, toasting `Saved IPTorrents` (P-364)
- `ported-stopped` `fixture-site-expire` `{"site": "iptorrents"}` and `Test` on IPTorrents' page lead with `Stopped because IPTorrents no longer accepts the browser cookie.` and `Paste a new browser cookie and its User-Agent with Edit.`; `{"site": "norbits"}` and `Test` on NorBits by code's page lead with `Stopped because NorBits by code needs a two-factor code to sign in again.` and show `Two-factor code`, and Health's Fail reads `NorBits by code needs a two-factor code to sign in again. Open the Indexer and type the current code.` with `Type code` linking to `#code`. `Test` with the code reads `NorBits by code passed the Test and listed 5 items.` and clears the item, while NorBits, which keeps its authenticator key, passes its Test with no code (P-364)

## How to get to it (user POV)

- Desktop shell: the sidebar's `Settings`, `Indexers`, then `Add Indexer`, and a ported site's row in the Indexer catalog. An Indexer's row opens its page at `/settings/indexers/<id>`, with `Test`, `Edit`, `Move` and `Delete` under its lead. Phone shell: `More`, `Settings`, `Indexers`.
- Direct: `/settings/indexers/new/catalog/<port id>`, such as `/settings/indexers/new/catalog/norbits`, and a code stop's Health link `/settings/indexers/<id>#code`.
- API: `GET /api/v1/indexer-catalog`, `POST /api/v1/indexers/test` and `POST /api/v1/indexers` with `kind` `native`, `port`, `baseAddress`, `values`, `cookie` and `code`; `PATCH /api/v1/indexers/{id}` with `values`, `cookie` and `code`, where a sealed setting sent empty keeps its saved value; `POST /api/v1/indexers/{id}/test` with `code`; `GET /api/v1/system/health`.
- Host CLI: `$FW site $RUN <port id>` prints what an Admin enters for a family site: its `origin` and `settings`, IPTorrents' `cookie` and `userAgent`, and NorBits' two-factor `code` of now.

## Driving it with fakeworld and playwright cli

Preconditions:

- A `ported-sites` run with doctor passing and the `{runId}-admin-desktop` session. Setup has added PassThePopcorn, Torrent Bytes and AvistaZ on home, and IPTorrents and NorBits on seedbox, NorBits with its authenticator key. Read each Indexer's id from `GET /api/v1/indexers`.

Steps:

- **The picker.** Search the catalog for NorBits.

  ```sh
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop goto '{nodes.home.browserUrl}/settings/indexers/new'
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop fill 'getByRole("combobox", { name: "Search the Indexer catalog" })' 'norbits'
  $FW exec $RUN --proof ported-sites/ported-picker -- $PW cli -s={runId}-admin-desktop screenshot --filename={runDir}/proof/ported-sites/ported-picker/picker.png
  ```

- **NorBits with a typed code.** Read NorBits' settings, open its form, choose `Another address` and fill `New address` with the `origin` and a `/`, then `Name`, `Username` and `Password`. Leave `Authenticator key` empty and choose `Save`, then fill `Two-factor code` and `Save` again.

  ```sh
  $FW site $RUN norbits
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop click 'getByRole("option", { name: /^NorBits/ })'
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop click 'getByRole("combobox", { name: "Address" })'
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop click 'getByRole("option", { name: "Another address" })'
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop fill 'getByRole("textbox", { name: "New address" })' 'https://norbits.fixture-site.fake.localhost/'
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop fill 'getByRole("textbox", { name: "Name", exact: true })' 'NorBits by code'
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop click 'getByRole("main").getByRole("button", { name: "Save" })'
  $FW exec $RUN --proof ported-sites/ported-add-code -- $PW cli -s={runId}-admin-desktop screenshot --filename={runDir}/proof/ported-sites/ported-add-code/code-step.png
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop fill 'getByRole("textbox", { name: "Two-factor code" })' '<code>'
  ```

- **The port's facts.** Open PassThePopcorn's page and its last `Details`.

  ```sh
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop goto '{nodes.home.browserUrl}/settings/indexers/<PassThePopcorn id>'
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop click 'getByRole("main").getByRole("button", { name: "Details" }).last()'
  $FW exec $RUN --proof ported-sites/ported-page-port -- $PW cli -s={runId}-admin-desktop screenshot --full-page --filename={runDir}/proof/ported-sites/ported-page-port/details.png
  ```

- **Edit with Replace.** On Torrent Bytes' page choose `Edit`, `Replace Password`, fill `Password` with `wrong` and `Save`, then with the `password` from `$FW site $RUN torrentbytes`. On IPTorrents' page choose `Edit`, `Replace Cookie`, fill `Cookie` and `User-Agent` from `$FW site $RUN iptorrents`, the first time with another User-Agent.

  ```sh
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop click 'getByRole("dialog").getByRole("button", { name: "Replace Password" })'
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop fill 'getByRole("dialog").getByRole("textbox", { name: "Password" })' 'wrong'
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop click 'getByRole("dialog").getByRole("button", { name: "Save" })'
  $FW exec $RUN --proof ported-sites/ported-edit-sealed -- curl -sS -H 'X-Api-Key: {apiKey}' '{nodes.home.url}/api/v1/indexers/<Torrent Bytes id>'
  ```

- **A stopped Indexer.** Stop IPTorrents' cookie and test it, then sign NorBits out and test NorBits by code, read Health, and test again with the code.

  ```sh
  $FW cue $RUN fixture-site-expire '{"site": "iptorrents"}'
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop click 'getByRole("main").getByRole("button", { name: "Test" })'
  $FW cue $RUN fixture-site-expire '{"site": "norbits"}'
  $FW exec $RUN --proof ported-sites/ported-stopped -- curl -sS -H 'X-Api-Key: {apiKey}' '{nodes.home.url}/api/v1/system/health'
  ```

## Gotchas

- The form's Name and Username are both found by `name: "Name"`; fill Name with `exact: true`, or the fill fails and the save is refused with `Another Indexer has that name`.
- A two-factor code holds for its 30 s step and the step either side. Read it from `$FW site $RUN norbits` just before you type it.
- `fixture-site-expire` stops IPTorrents' cookie for the rest of the run, because the account has one cookie. Paste a cookie in Edit before the cue, not after.
- NorBits keeps its authenticator key, so a signed-out NorBits signs in again by itself. Only an Indexer added without the key stops for a code.
