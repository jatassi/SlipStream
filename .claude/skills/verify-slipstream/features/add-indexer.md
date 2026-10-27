# Add Indexer

Settings > Indexers > Add Indexer leads with the Indexer catalog: a search that keeps focus, Privacy and Language filters with counts, and rows for the definitions SlipStream can't run greyed with the reason a click away. Beside it sit a Torznab feed, an RSS feed and a custom definition. A definition's form draws its own settings, then the rest under Details; a definition whose sign-in shows a captcha loads one from its home Node before it saves. A Torznab feed is tested before it saves, so its report can warn about many sources or ask for the Tracker its items don't name. Every refusal is Go's sentence, shown as is.

## Sub-features

- `add-indexer-picker` on `catalog-sites`, `/settings/indexers/new` lists the catalog with Quarry, Larkspur, Cinder and the rest, filters by Privacy and Language with counts, and a search that matches nothing reads `No definition matches <words>.` (P-109)
- `add-indexer-torznab` `Torznab feed` with a wrong API key reads `<Name> refused the API key with 'Invalid API Key'. Check the account, then test again.`; with the run's key, `Test` reads `The feed passed the Test and listed <n> items.` and shows `Tracker` filled with the Name, and `Save` opens the new Indexer's page. (P-108)
- `add-indexer-captcha` Larkspur's form signs in from home; `Continue` loads the captcha home fetched, a wrong password reads `Larkspur refused the sign-in with '<site's words>'. Check the username, password and any code, then test again.` with a new image, and the right password with the new code saves. (P-111)
- `add-indexer-rss` `RSS feed` saves Brook's feed with its cookie, or reads the feed's refusal such as `Some items have no size. Turn on Allow items with no size to use the feed anyway.` (P-112)
- `add-indexer-custom` `Custom definition` takes pasted YAML or a file; one SlipStream can't run reads `SlipStream can't run this definition yet.` with the reason under Details, and one that loads opens its form. (P-110)

## How to get to it (user POV)

- Desktop shell: the sidebar's `Settings`, `Indexers`, then `Add Indexer` in the section's header (or its empty state). Phone shell: `More`, `Settings`, `Indexers`, `Add Indexer`.
- Direct: `/settings/indexers/new`, `/settings/indexers/new/catalog/<definition id>`, `/settings/indexers/new/torznab`, `/settings/indexers/new/rss`, `/settings/indexers/new/custom`.
- API: `GET /api/v1/indexer-catalog`, `GET /api/v1/indexer-catalog/{id}`, `POST /api/v1/indexer-catalog/custom`, `POST /api/v1/indexers/test`, `POST /api/v1/indexers/captcha`, `POST /api/v1/indexers`.
- Host CLI: `$FW site $RUN <label>` prints what an Admin enters for one of fixture-site's sites: a feed's address and API key, or a catalog site's settings and pasted cookie.

## Driving it with fakeworld and playwright cli

Preconditions:

- A `catalog-sites` run with doctor passing and the `{runId}-admin-desktop` session. Setup has refreshed the catalog and added Quarry.

Steps:

- **The picker.** Open Add Indexer and narrow the catalog.

  ```sh
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop goto '{nodes.home.browserUrl}/settings/indexers/new'
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop fill 'getByRole("combobox", { name: "Search the Indexer catalog" })' 'lark'
  $FW exec $RUN --proof add-indexer/add-indexer-picker -- $PW cli -s={runId}-admin-desktop screenshot --filename={runDir}/proof/add-indexer/add-indexer-picker/picker.png
  ```

- **A Torznab feed.** Read the feed's address and key, then fill `Name`, `Feed address` and `API key`, choose `Test`, then `Save`.

  ```sh
  $FW site $RUN torznab
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop goto '{nodes.home.browserUrl}/settings/indexers/new/torznab'
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop fill 'getByRole("textbox", { name: "Feed address" })' '<feed>'
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop click 'getByRole("button", { name: "Test" })'
  ```

- **Larkspur with its captcha.** Read Larkspur's username and password, pick Larkspur, fill both, choose `Continue`, then read the code the captcha shows from the `captchas` capture and fill `Code`.

  ```sh
  $FW site $RUN larkspur
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop goto '{nodes.home.browserUrl}/settings/indexers/new/catalog/larkspur'
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop click 'getByRole("button", { name: "Continue" })'
  $FW captchas wait $RUN --where site=larkspur --timeout 30s
  $FW exec $RUN --proof add-indexer/add-indexer-captcha -- $PW cli -s={runId}-admin-desktop screenshot --filename={runDir}/proof/add-indexer/add-indexer-captcha/captcha.png
  ```

- **The browser spec.** `web/e2e/add-indexer.spec.ts` drives the Torznab and Larkspur steps on chromium-desktop.

  ```sh
  cd web && TEST_SLIPSTREAM_BINARY=../dist/slipstream TEST_FAKEWORLD_BINARY=../dist/fakeworld bunx playwright test -c e2e add-indexer.spec.ts --project=chromium-desktop --workers=1
  ```

## Gotchas

- Each captcha is answered once. A failed save hands back a new image, so read the next `captchas` record after the first one's seq, not the first again.
- Changing the username or the home Node drops the captcha: the button goes back to `Continue`.
- The Torznab fixture's items name no source, so `Tracker` always shows after its Test. A feed whose Test lists many sources shows `Many sources` instead.
- A definition SlipStream can't run stays in the list; its page shows only the sentence and `Details`, with no form.
