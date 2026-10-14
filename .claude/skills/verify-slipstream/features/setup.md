# Setup

A fresh Node opens on its first-run page. A person types the Setup code from the Node's console or log, creates the first Admin with an email, display name and PIN, sees the API key once, and lands on the Library as that Admin.

## Sub-features

- `setup-code-read` the Setup code is read from the Controller's log, as a real Admin reads it.
- `setup-wrong-code` a wrong Setup code marks the field invalid with `The Setup code is wrong` and keeps the rest of the form.
- `setup-create-admin` the right code creates the first Admin and shows the `API key` once.
- `setup-signed-in` `Continue` lands on the Library signed in, the header shows the display name, and `GET /api/v1/status` answers `{"claim":"controller"}`.

## How to get to it (user POV)

- Desk and phone: any URL of an Unclaimed Node redirects to `/setup`, headed `Set up SlipStream`.
- API: `POST /api/v1/setup/controller` with the Setup code; `GET /api/v1/status` reads the claim.

## Driving it with fakeworld and playwright cli

Preconditions:

- A `first-run` run: `dist/fakeworld start --binary dist/slipstream --scenario first-run --run-dir $RUN`, then `FW` and `PW` as in the skill's Launch, and doctor passing.
- The manifest names no Admin and no API key, so the supervisor captures no SSE and `bin/sign-in.ts` has no identity to sign in. Open the session without `state-load`.

Steps:

- **Open the first-run page.** The page reads `Set up SlipStream` at `/setup`.

  ```sh
  $FW browser $RUN
  $FW exec $RUN -- $PW cli -s={runId}-admin-desk open --config={runDir}/browser/desk/playwright-cli.json '{nodes.home.browserUrl}/'
  ```

- **Read the Setup code.** It is the `code` of the log record whose `msg` is `setup code`, such as `Q7G9-2NRV-7TG7`.

  ```sh
  CODE=$($FW exec $RUN --proof setup/setup-code-read -- jq -r 'select(.msg == "setup code") | .code' '{nodes.home.log}')
  ```

- **Try a wrong code.** Fill the form with `0000-0000-0000`. The `Setup code` textbox turns invalid with `The Setup code is wrong`; the other fields keep their values.

  ```sh
  $FW exec $RUN -- $PW cli -s={runId}-admin-desk fill 'getByRole("textbox", { name: "Setup code" })' "0000-0000-0000"
  $FW exec $RUN -- $PW cli -s={runId}-admin-desk fill 'getByRole("textbox", { name: "Email" })' "grace@example.test"
  $FW exec $RUN -- $PW cli -s={runId}-admin-desk fill 'getByRole("textbox", { name: "Display name" })' "Grace Hopper"
  $FW exec $RUN -- $PW cli -s={runId}-admin-desk click 'getByRole("textbox", { name: "PIN", exact: true })'
  $FW exec $RUN -- $PW cli -s={runId}-admin-desk type 1357
  $FW exec $RUN -- $PW cli -s={runId}-admin-desk click 'getByRole("textbox", { name: "PIN again", exact: true })'
  $FW exec $RUN -- $PW cli -s={runId}-admin-desk type 1357
  $FW exec $RUN -- $PW cli -s={runId}-admin-desk click 'getByRole("button", { name: "Create Admin" })'
  $FW exec $RUN --proof setup/setup-wrong-code -- $PW cli -s={runId}-admin-desk snapshot --filename={runDir}/proof/setup/setup-wrong-code/form.aria.yml
  ```

- **Create the Admin.** Fill the real code and choose `Create Admin`. The page shows a read-only `API key` (`ssk_` and 43 characters) and `Continue`.

  ```sh
  $FW exec $RUN -- $PW cli -s={runId}-admin-desk fill 'getByRole("textbox", { name: "Setup code" })' "$CODE"
  $FW exec $RUN -- $PW cli -s={runId}-admin-desk click 'getByRole("button", { name: "Create Admin" })'
  $FW exec $RUN -- $PW cli -s={runId}-admin-desk run-code 'async page => { await page.getByRole("textbox", { name: "API key" }).waitFor({ timeout: 10000 }); }'
  $FW exec $RUN --proof setup/setup-create-admin -- $PW cli -s={runId}-admin-desk screenshot --filename={runDir}/proof/setup/setup-create-admin/api-key.png
  ```

- **Continue.** The Library opens at `/`, the header shows `Grace Hopper` and `Live updates: on`, and the status reads `{"claim":"controller"}`.

  ```sh
  $FW exec $RUN -- $PW cli -s={runId}-admin-desk click 'getByRole("button", { name: "Continue" })'
  $FW exec $RUN --proof setup/setup-signed-in -- $PW cli -s={runId}-admin-desk screenshot --filename={runDir}/proof/setup/setup-signed-in/library.png
  $FW exec $RUN --proof setup/setup-signed-in -- curl -sS '{nodes.home.url}/api/v1/status'
  ```

## Gotchas

- The Setup code is single-use. After the Admin exists, `/setup` sends the browser to the Library; relaunch `first-run` to drive setup again.
- The API key appears only once. Copy it into the proof by the screenshot; nothing reads it back.
- The PIN fields split into four digit boxes named `PIN`, `PIN digit 2 of 4` and so on. Click the first box and `type` the digits.
- The manifest's `admin` and `apiKey` stay empty for the whole run, so `{apiKey}` and `bin/sign-in.ts` fail on `first-run`.
