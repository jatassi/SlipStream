# Passkeys

At an address that can hold one (a name, over HTTPS or at `localhost` and `*.localhost`), a person adds a passkey after signing in, while joining or on Your Account, and then signs in with it alone. Each passkey is labelled with the address it works at and can be renamed and deleted. At an IP address SlipStream offers no passkey and says why.

## Sub-features

- `passkey-prompt` after a PIN sign-in at `{nodes.home.browserUrl}` the sheet `Sign in faster here with a passkey` offers `Add a passkey`, `Not now` and `Don't ask again` (P-314).
- `passkey-add` `Add a passkey` on Your Account runs registration with Chromium's virtual authenticator and lists the new passkey under `Passkeys` with the address `home.localhost` (P-236).
- `passkey-sign-in` the sign-in page offers the passkey from the email field's autofill and signs in with no email or PIN typed; `Sign in with a passkey` is the same for browsers without autofill (P-313).
- `passkey-deep-link` `/account/passkeys/new` opens Your Account, `/portal/account?add=passkey` for a Member, with the dialog `Add a passkey`, whose `Continue` starts registration; an authenticator that already holds one reads `This device already has a passkey for you here.` (P-315).
- `passkey-rename-delete` a passkey's `<name> actions` menu has `Rename` and `Delete` (P-236).
- `passkey-ip-address` at `{nodes.home.url}`, an IP address, there is no prompt, no `Sign in with a passkey`, and Your Account explains why.

## How to get to it (user POV)

- Desktop and phone: the prompt after signing in; Your Account at `/account` (the sidebar's `Account` menu, or `More` on the phone) and `/portal/account` for a Member; the sign-in page's `Sign in with a passkey`; the deep link `/account/passkeys/new`.
- The signup and Reset flows' passkey step ([signup links](./signup-links.md)).
- API: `POST /api/v1/passkeys/registration[/{ceremony}]`, `POST /api/v1/passkeys/assertion[/{ceremony}]`, `GET /api/v1/passkeys`, `PATCH` and `DELETE /api/v1/passkeys/{id}`, and `GET /api/v1/status` for `passkeys.available`.

## Driving it with fakeworld and playwright cli

Preconditions:

- A `home` run with doctor passing.
- A Chromium session with no saved state for Grace, `{runId}-grace-desktop`, and Chromium's virtual authenticator on its page. `playwright cli` reaches CDP through `run-code`:

  ```sh
  $FW exec $RUN -- $PW cli -s={runId}-grace-desktop open --config={runDir}/browser/desktop/playwright-cli.json '{nodes.home.browserUrl}/sign-in'
  $FW exec $RUN -- $PW cli -s={runId}-grace-desktop run-code 'async page => { const cdp = await page.context().newCDPSession(page); await cdp.send("WebAuthn.enable"); await cdp.send("WebAuthn.addVirtualAuthenticator", { options: { protocol: "ctap2", transport: "internal", hasResidentKey: true, hasUserVerification: true, isUserVerified: true, automaticPresenceSimulation: true } }); }'
  ```

Steps:

- **The prompt.** Sign in as `grace@example.test` with PIN `1357`. The sheet `Sign in faster here with a passkey` opens over the portal's Home.

  ```sh
  $FW exec $RUN -- $PW cli -s={runId}-grace-desktop fill 'getByRole("textbox", { name: "Email" })' "grace@example.test"
  $FW exec $RUN -- $PW cli -s={runId}-grace-desktop click 'getByRole("textbox", { name: "PIN", exact: true })'
  $FW exec $RUN -- $PW cli -s={runId}-grace-desktop type 1357
  $FW exec $RUN --proof passkeys/passkey-prompt -- $PW cli -s={runId}-grace-desktop screenshot --filename={runDir}/proof/passkeys/passkey-prompt/sheet.png
  ```

- **Add one.** Choose `Add a passkey`. The virtual authenticator answers, the sheet closes, and Your Account's `Passkeys` list holds `Chrome on Linux` with the meta `home.localhost · Added <time> · Never used`.

  ```sh
  $FW exec $RUN -- $PW cli -s={runId}-grace-desktop click 'getByRole("dialog", { name: "Sign in faster here with a passkey" }).getByRole("button", { name: "Add a passkey" })'
  $FW exec $RUN -- $PW cli -s={runId}-grace-desktop goto '{nodes.home.browserUrl}/portal/account'
  $FW exec $RUN --proof passkeys/passkey-add -- $PW cli -s={runId}-grace-desktop snapshot --filename={runDir}/proof/passkeys/passkey-add/account.aria.yml
  ```

- **Sign in with it.** Sign out. The sign-in page starts a conditional ceremony for the email field's autofill, the virtual authenticator answers it at once as a person picking the passkey would, and the portal's Home opens again with nothing typed. `requests` shows `DELETE /api/v1/session` answering 204, then `POST /api/v1/passkeys/assertion` and `POST /api/v1/passkeys/assertion/<ceremony>`, each 200.

  ```sh
  $FW exec $RUN -- $PW cli -s={runId}-grace-desktop click 'getByRole("banner").getByRole("button", { name: "Account" })'
  $FW exec $RUN -- $PW cli -s={runId}-grace-desktop click 'getByRole("menuitem", { name: "Sign out" })'
  $FW exec $RUN -- $PW cli -s={runId}-grace-desktop click 'getByRole("dialog", { name: "Sign out of this device?" }).getByRole("button", { name: "Sign out" })'
  $FW exec $RUN -- $PW cli -s={runId}-grace-desktop run-code 'async page => { await page.getByRole("heading", { name: "Home" }).waitFor({ timeout: 10000 }); }'
  $FW exec $RUN --proof passkeys/passkey-sign-in -- $PW cli -s={runId}-grace-desktop requests
  $FW exec $RUN --proof passkeys/passkey-sign-in -- $PW cli -s={runId}-grace-desktop screenshot --filename={runDir}/proof/passkeys/passkey-sign-in/home.png
  ```

- **At the IP address.** Read `GET /api/v1/status` at `{nodes.home.url}`; it answers `"passkeys":{"available":false,"reason":"ip_address"}`, the sign-in page there has no passkey button, and after a PIN sign-in there Your Account reads `Passkeys need SlipStream's name, not an IP address. Open SlipStream by its name to use one.`

  ```sh
  $FW exec $RUN --proof passkeys/passkey-ip-address -- curl -sS '{nodes.home.url}/api/v1/status'
  ```

## Gotchas

- Use `{nodes.home.browserUrl}`. Chromium treats `*.localhost` as a secure context, so passkeys work there over plain HTTP; the bare IP can't hold one.
- The virtual authenticator belongs to the page it was added on and lives as long as the session's browser. Closing the session drops its passkeys, so a passkey added in one run can't sign in after `close`.
- Firefox and WebKit have no virtual authenticator; drive passkeys in Chromium only.
- With a passkey on the authenticator, every visit to the sign-in page signs straight in through autofill, so `Sign in with a passkey` can't be clicked there. Report it as not driven; the web's Vitest covers the button.
- The Node prompts only while the Account holds no passkey at the address, so after `passkey-add` Grace's PIN sign-ins elsewhere get no sheet.
- Adding a passkey is a Confirmed action: past the re-confirm window it asks for the PIN first ([re-confirm](./reconfirm.md)).
