# Re-confirm

Actions that change who can get in or what a credential is ask for the PIN or a passkey again when the session's last confirm is older than the re-confirm window, 15 minutes by default. One dialog, `Confirm it's you`, serves every such action, and on success the action runs once more by itself. The API key never re-confirms.

## Sub-features

- `reconfirm-dialog` past the window, a Reconfirmed action opens `Confirm it's you` reading `Enter your PIN to continue.`, and the right PIN runs the action with nothing to redo (P-235).
- `reconfirm-cancel` `Cancel` leaves the action undone with the Node's `needs_reconfirm` answer.
- `reconfirm-wrong-pin` a wrong PIN in the dialog says so and counts toward the PIN's limits.
- `reconfirm-api-key` the same action with `X-Api-Key` succeeds with no confirm.

## How to get to it (user POV)

- Desktop and phone: any Reconfirmed action once the window has passed: `New Signup link`, `Send a Reset link`, `Make Member`, `Regenerate` on Settings > API key, `Save` on Settings > Email, Address or Sign-in. On the phone the dialog is a sheet with the keypad.
- API: `POST /api/v1/session/reconfirm {pin}` and `POST /api/v1/session/reconfirm/passkey[/{ceremony}]`; a Reconfirmed call outside the window answers 403 `needs_reconfirm`.

## Driving it with fakeworld and playwright cli

Preconditions:

- A `home` run with doctor passing and the `{runId}-admin-desktop` session.
- The re-confirm window shortened to its floor, one minute, through the API, then a wait past it. Sign-in stamps the confirm, so the wait starts after the session's sign-in.

  ```sh
  $FW exec $RUN --proof reconfirm/reconfirm-dialog -- curl -sS -X PATCH -H 'X-Api-Key: {apiKey}' -H 'Content-Type: application/json' -d '{"reconfirmWindow":60}' '{nodes.home.url}/api/v1/identity/settings'
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop run-code 'async page => { await page.waitForTimeout(65000); }'
  ```

Steps:

- **Ask.** On Accounts choose `New Signup link`, enter `alan@example.test` and choose `Make the link`. `Confirm it's you` opens over the dialog with `Enter your PIN to continue.`, the PIN, `Use a passkey`, `Cancel` and `Confirm`.

  ```sh
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop goto '{nodes.home.browserUrl}/accounts'
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop click 'getByRole("button", { name: "New Signup link" })'
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop fill 'getByRole("dialog", { name: "New Signup link" }).getByRole("textbox", { name: "Email" })' "alan@example.test"
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop click 'getByRole("button", { name: "Make the link" })'
  $FW exec $RUN --proof reconfirm/reconfirm-dialog -- $PW cli -s={runId}-admin-desktop screenshot --filename={runDir}/proof/reconfirm/reconfirm-dialog/ask.png
  ```

- **Confirm.** The dialog's PIN has focus; type `2468`. The dialog closes and the Signup link dialog shows the link and `Emailed it to alan@example.test.`, with nothing typed again.

  ```sh
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop type 2468
  $FW exec $RUN --proof reconfirm/reconfirm-dialog -- $PW cli -s={runId}-admin-desktop screenshot --filename={runDir}/proof/reconfirm/reconfirm-dialog/done.png
  ```

- **The API key skips it.** A Reconfirmed call with the key succeeds at once: put the window back to its default.

  ```sh
  $FW exec $RUN --proof reconfirm/reconfirm-api-key -- curl -sS -i -X PATCH -H 'X-Api-Key: {apiKey}' -H 'Content-Type: application/json' -d '{"reconfirmWindow":null}' '{nodes.home.url}/api/v1/identity/settings'
  ```

## Gotchas

- Prove the dialog on an action that leaves the run usable. `Regenerate` on Settings > API key asks too, but it replaces `{apiKey}` for every later step ([API key](./api-key.md)).
- `{"reconfirmWindow":null}` resets the window to its default; leave it short and every later recipe asks again.
- The dialog's name holds an apostrophe; in bash write it as `'"'"'` inside a single-quoted locator: `'getByRole("dialog", { name: "Confirm it'"'"'s you" })'`.
