# Setup

A fresh Node opens on its first-run page. A person types the Setup code from the Node's console or log, or opens the setup link that carries it, creates the first Admin with an email, display name and PIN, and lands on the first step of [First setup](./first-run.md) signed in as that Admin.

## Sub-features

- `setup-code-read` the Setup code is read from the Controller's log, as a real Admin reads it.
- `setup-wrong-code` a wrong Setup code marks the field invalid with `The Setup code is wrong` and keeps the rest of the form.
- `setup-link-prefill` the setup link `/setup#code=<code>` fills in the `Setup code` grouped, says `Setup code filled in` under it, and drops the fragment from the address bar (P-282).
- `setup-create-admin` the right code creates the first Admin and shows no API key.
- `setup-signed-in` the Admin lands signed in on first setup's first step at `/setup/address`, headed `Where people reach SlipStream`, and `GET /api/v1/status` answers `{"claim":"controller"}`.

## How to get to it (user POV)

- Desktop and phone: any URL of an Unclaimed Node redirects to `/setup`, headed `Set up SlipStream`.
- Setup link: `/setup#code=<code>`, the link a Node opens or logs for its first run.
- API: `POST /api/v1/setup/controller` with the Setup code; `GET /api/v1/status` reads the claim.

## Driving it with fakeworld and playwright cli

Preconditions:

- A `first-run` run: `dist/fakeworld start --binary dist/slipstream --scenario first-run --run-dir $RUN`, then `FW` and `PW` as in the skill's Launch, and doctor passing.
- The manifest names no Admin and no API key, so the supervisor captures no SSE and `bin/sign-in.ts` has no identity to sign in. Open the session without `state-load`.

Steps:

- **Open the first-run page.** The page reads `Set up SlipStream` at `/setup`.

  ```sh
  $FW browser $RUN
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop open --config={runDir}/browser/desktop/playwright-cli.json '{nodes.home.browserUrl}/'
  ```

- **Read the Setup code.** It is the `code` of the log record whose `msg` is `setup code`, such as `Q7G9-2NRV-7TG7`.

  ```sh
  CODE=$($FW exec $RUN --proof setup/setup-code-read -- jq -r 'select(.msg == "setup code") | .code' '{nodes.home.log}')
  ```

- **Try a wrong code.** Fill the form with `0000-0000-0000`. The `Setup code` textbox turns invalid with `The Setup code is wrong`; the other fields keep their values.

  ```sh
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop fill 'getByRole("textbox", { name: "Setup code" })' "0000-0000-0000"
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop fill 'getByRole("textbox", { name: "Email" })' "grace@example.test"
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop fill 'getByRole("textbox", { name: "Display name" })' "Grace Hopper"
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop click 'getByRole("textbox", { name: "PIN", exact: true })'
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop type 1357
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop click 'getByRole("textbox", { name: "PIN again", exact: true })'
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop type 1357
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop click 'getByRole("button", { name: "Create Admin" })'
  $FW exec $RUN --proof setup/setup-wrong-code -- $PW cli -s={runId}-admin-desktop snapshot --filename={runDir}/proof/setup/setup-wrong-code/form.aria.yml
  ```

- **Open the setup link.** A new tab at `/setup#code=$CODE` shows the `Setup code` textbox holding `$CODE`, described `Setup code filled in`, and its address reads `/setup` with no fragment.

  ```sh
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop tab-new "{nodes.home.browserUrl}/setup#code=$CODE"
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop run-code 'async page => { await page.waitForURL((url) => url.hash === "", { timeout: 10000 }); }'
  $FW exec $RUN --proof setup/setup-link-prefill -- $PW cli -s={runId}-admin-desktop eval 'location.href'
  $FW exec $RUN --proof setup/setup-link-prefill -- $PW cli -s={runId}-admin-desktop snapshot --filename={runDir}/proof/setup/setup-link-prefill/form.aria.yml
  $FW exec $RUN --proof setup/setup-link-prefill -- $PW cli -s={runId}-admin-desktop screenshot --filename={runDir}/proof/setup/setup-link-prefill/form.png
  ```

- **Create the Admin.** In the setup link's tab, fill the rest of the form and choose `Create Admin`. The page goes straight to first setup at `/setup/address`, headed `Where people reach SlipStream`, and shows no API key.

  ```sh
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop fill 'getByRole("textbox", { name: "Email" })' "grace@example.test"
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop fill 'getByRole("textbox", { name: "Display name" })' "Grace Hopper"
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop click 'getByRole("textbox", { name: "PIN", exact: true })'
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop type 1357
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop click 'getByRole("textbox", { name: "PIN again", exact: true })'
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop type 1357
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop click 'getByRole("button", { name: "Create Admin" })'
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop run-code 'async page => { await page.getByRole("heading", { name: "Where people reach SlipStream", level: 1 }).waitFor({ timeout: 10000 }); }'
  $FW exec $RUN --proof setup/setup-create-admin -- $PW cli -s={runId}-admin-desktop screenshot --filename={runDir}/proof/setup/setup-create-admin/first-step.png
  ```

- **Read where it landed.** The address bar reads `/setup/address` and the status reads `{"claim":"controller"}`. [First setup](./first-run.md) carries on from here.

  ```sh
  $FW exec $RUN --proof setup/setup-signed-in -- $PW cli -s={runId}-admin-desktop eval 'location.href'
  $FW exec $RUN --proof setup/setup-signed-in -- curl -sS '{nodes.home.url}/api/v1/status'
  ```

## Gotchas

- The Setup code is single-use. After the Admin exists, `/setup` sends the browser to first setup's current step, or to the Library once first setup is finished; relaunch `first-run` to drive setup again.
- Open the setup link with `tab-new`, as the Node's browser hand-off does. A `goto` from `/setup` to `/setup#code=...` is a same-document fragment change, so the page never reads the fragment and the field stays as it was.
- The PIN fields split into four digit boxes named `PIN`, `PIN digit 2 of 4` and so on. Click the first box and `type` the digits.
- The manifest's `admin` and `apiKey` stay empty for the whole run, so `exec` refuses `{apiKey}` with `{apiKey} names nothing in the manifest`, and `bin/sign-in.ts` fails on `first-run`.
