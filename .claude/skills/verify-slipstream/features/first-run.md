# First setup

After the claim, the new Admin lands on first setup at `/setup/<step>`: a short run of steps that each say why they matter, with `Step n of m`, the steps ahead, `Back`, `Skip`, `Continue` and `Finish setup` on every one. The place is stored on the Controller, so a reload or another browser resumes on the same step, and any app URL brings the Admin back to it. A last screen lists each step as set or skipped, and `Go to Library` finishes setup for good.

## Sub-features

- `first-setup-frame` each step shows its title, one sentence on why it matters, `Step n of m` and `After this:` with the names of the steps ahead. `Skip`, `Continue`, `Back` and `Finish setup` work on every step, and `Finish setup` lands on the Library (P-106).
- `first-setup-resume` a reload, a second browser or any app URL such as `/settings/address` opens the step the Admin is on, and a step completed elsewhere reads done (P-106).
- `first-setup-address` the address step suggests the address in use, warns that a plain-HTTP address can't hold a passkey, and `Continue` saves it (P-106).
- `first-setup-passkey` the passkey step shows only where the browser and address can hold a passkey, its `Add a passkey` runs registration, and the frame's `Skip` is its only Skip (P-106).
- `first-setup-media` the kinds of media step shows only when the binary registers more than one Content Type, turns one off, keeps the last one on, and shows `content_type_in_use` on its switch. The API refuses turning off the only type on with 409 `content_type_last`, which the switch shows as `At least one kind of media stays on. Turn on another before turning this one off.` The same switches lead Settings > Library roots (P-106).
- `first-setup-language` the language and region step fills in both from the browser's languages while they hold the defaults, says file names keep the English title, and `Continue` saves both at once (P-106).
- `first-setup-roots` the library roots step lists one row per enabled Content Type, `Add a folder` opens `Add a library root` with the type chosen and no Node picker while one Node holds Storage, and says Works wait for a root while one is missing (P-106).
- `first-setup-downloads` the downloads step lists each Downloading Node with its `torrent.downloads` folder, its free space, the different-drive recommendation when the Node's roots sit on another volume, and the restart notice after a change. A Node whose settings don't load says so in its own row with `Try again`, and the step keeps `Back`, `Skip` and `Finish setup` (P-106).
- `first-setup-targets` the Targets step says in one sentence what each enabled type's default Profile grabs, holds the Profile, qualities, cutoff and sizes under `Details`, and `Edit` opens the Target editor in place (P-106).
- `first-setup-done` the last screen lists each shown step as `Set` or `Skipped`, a skipped step's `Set it in Settings` finishes and opens its place in Settings, while email isn't set up `Email isn't set up.` links `/settings/email` the same way, and after `Go to Library` no URL opens first setup again (P-106).

## How to get to it (user POV)

- Desktop and phone: claim a fresh Node as in [Setup](./setup.md), and the app opens `/setup/address`. While first setup is open, every app URL redirects to `/setup/<step>`. The last screen is `/setup/done`.
- A Member never sees first setup.
- API: `GET /api/v1/first-run` reads the steps and whether setup is open, `PUT /api/v1/first-run/steps/{id}` stores `done` or `skipped`, and `POST /api/v1/first-run/finish` ends it. `GET /api/v1/session` carries `firstRun` for an Admin.

## Driving it with fakeworld and playwright cli

Preconditions:

- A `first-run` run claimed through the page as in [Setup](./setup.md), so `{runId}-admin-desktop` is signed in as Grace Hopper on `/setup/address`. The release binary registers one Content Type, so the media step is hidden and the steps read `Address`, `Passkey`, `Language and region`, `Library roots`, `Downloads`, `What SlipStream grabs`.
- For the passkey step, Chromium's virtual authenticator on the session's page, added with `run-code` as in [Passkeys](./passkeys.md).

Steps:

- **Read the frame.** The address step reads `Step 1 of 6` and `After this: Passkey, Language and region, Library roots, Downloads and What SlipStream grabs`, with the `Address` textbox holding `{nodes.home.browserUrl}`.

  ```sh
  $FW exec $RUN --proof first-run/first-setup-frame -- $PW cli -s={runId}-admin-desktop snapshot --filename={runDir}/proof/first-run/first-setup-frame/address.aria.yml
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop click 'getByRole("button", { name: "Continue" })'
  ```

- **Add a passkey.** `Add a passkey` registers with the virtual authenticator and moves to `Language and region`.

  ```sh
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop click 'getByRole("button", { name: "Add a passkey" })'
  $FW exec $RUN --proof first-run/first-setup-passkey -- $PW cli -s={runId}-admin-desktop run-code 'async page => { await page.getByRole("heading", { name: "Language and region", level: 1 }).waitFor({ timeout: 10000 }); }'
  ```

- **Save the language and region.** The comboboxes `Metadata language` and `Metadata region` hold what the browser's languages suggest. `Continue` saves both, and `GET /api/v1/metadata/settings` reads them back.

  ```sh
  $FW exec $RUN --proof first-run/first-setup-language -- $PW cli -s={runId}-admin-desktop snapshot --filename={runDir}/proof/first-run/first-setup-language/before.aria.yml
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop click 'getByRole("button", { name: "Continue" })'
  $FW exec $RUN --proof first-run/first-setup-language -- $PW cli -s={runId}-admin-desktop eval 'fetch("/api/v1/metadata/settings").then((r) => r.json()).then((b) => b.settings.map((s) => s.key + "=" + s.value).join(" "))'
  ```

- **Resume it.** Reload on `Library roots`, then open `/settings/address`. Both land on `/setup/roots`, whose `Movies` row reads `No folder yet` with `Add a folder`.

  ```sh
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop reload
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop goto '{nodes.home.browserUrl}/settings/address'
  $FW exec $RUN --proof first-run/first-setup-resume -- $PW cli -s={runId}-admin-desktop eval 'location.href'
  $FW exec $RUN --proof first-run/first-setup-roots -- $PW cli -s={runId}-admin-desktop snapshot --filename={runDir}/proof/first-run/first-setup-roots/roots.aria.yml
  ```

- **Skip the roots and read the downloads.** `Skip` stores the step skipped. The downloads step's `Downloading Nodes` list holds `home` with its folder and free space.

  ```sh
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop click 'getByRole("button", { name: "Skip" })'
  $FW exec $RUN --proof first-run/first-setup-downloads -- $PW cli -s={runId}-admin-desktop snapshot --filename={runDir}/proof/first-run/first-setup-downloads/downloads.aria.yml
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop click 'getByRole("button", { name: "Continue" })'
  ```

- **Read the Targets.** The `Movies` row's sentence starts `Grabs`. `Details` opens the Profile, qualities, cutoff and sizes, and `Edit` opens the Target editor in a dialog.

  ```sh
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop click 'getByRole("button", { name: "Details" })'
  $FW exec $RUN --proof first-run/first-setup-targets -- $PW cli -s={runId}-admin-desktop screenshot --filename={runDir}/proof/first-run/first-setup-targets/details.png
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop click 'getByRole("button", { name: "Continue" })'
  ```

- **Finish.** The last screen at `/setup/done` lists `Library roots` as `Skipped` with `Set it in Settings` and every other step as `Set`, then `Email isn't set up.` with its own `Set it in Settings`. `Go to Library` opens `/library`, a later `/setup/roots` lands on the Library too, and `GET /api/v1/first-run` reads `"open":false`.

  ```sh
  $FW exec $RUN --proof first-run/first-setup-done -- $PW cli -s={runId}-admin-desktop snapshot --filename={runDir}/proof/first-run/first-setup-done/summary.aria.yml
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop click 'getByRole("button", { name: "Go to Library" })'
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop goto '{nodes.home.browserUrl}/setup/roots'
  $FW exec $RUN --proof first-run/first-setup-done -- $PW cli -s={runId}-admin-desktop eval 'location.href'
  $FW exec $RUN --proof first-run/first-setup-done -- $PW cli -s={runId}-admin-desktop eval 'fetch("/api/v1/first-run").then((r) => r.text())'
  ```

## Gotchas

- `first-run` is the only Scenario that leaves first setup open. Every Scenario that claims its Controller through the API finishes it, so recipes elsewhere land in the app.
- The passkey step needs `home.localhost` in Chromium. Firefox and WebKit reach the Node by IP address, so the step is hidden there and the steps read `Step 1 of 5`.
- The language step's suggestion follows the browser's languages, so what it fills in depends on the session's locale. It fills in only while the settings still hold the defaults.
- The media step and the Settings switches show only on a binary that registers more than one Content Type. The release binary registers Movies alone, so prove `first-setup-media` with the component tests.
- `Continue` on a step that already reads done moves on without a write. `Back` to a skipped step and `Continue` stores it done.
