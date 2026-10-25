# Reset links and the host command

An Admin sends a Reset link from an Account's page. Issuing it ends every session the Account holds, so the person's open tabs land on Sign in, and the link is emailed and shown once to copy. The person opens it, sets a new PIN twice, deletes any passkey they don't recognise, and is signed in. When no Admin can sign in, `slipstream admin reset <email>` on the host prints a Reset link, whether the Node runs or not.

## Sub-features

- `reset-link-issue` `Send a Reset link` on `/accounts/<id>` asks with `Also remove passkeys`, answers `Emailed it to <email>.` and shows the field `Reset link` with `SlipStream shows this link only once.`
- `reset-link-ends-sessions` issuing it sends the person's open tab to `/sign-in?reason=revoked`, reading `This device was signed out.`
- `reset-link-redeem` the link asks for a `New PIN` and `Type it again`, then reads `Your new PIN is set, and you're signed in.`; the old PIN stops working (P-235).
- `reset-link-email` the email reaches the `mail` capture with the subject `Reset your PIN for <host>`.
- `reset-host-command` `slipstream admin reset --data-dir <dir> <email>` against the running Controller prints one link, `{nodes.home.browserUrl}/reset#token=…`, and nothing else on stdout; its dead-link card names `your Admin`.
- `reset-host-command-refusals` an unknown email exits 1 with `identity.reset: failed: no Account has the email <email>` against the running Node; the Worker's data dir exits 3 with `this machine isn't a Controller, so it has no Accounts to reset`.

## How to get to it (user POV)

- Desktop and phone: `Accounts`, the person's name, then `Send a Reset link`.
- The person: the link from the dialog or the email, `{nodes.home.browserUrl}/reset#token=…`.
- Host CLI on the Controller's machine: `slipstream admin reset [--data-dir <dir>] [--address <origin>] <email>`.
- API: `POST /api/v1/accounts/{id}/reset-link {removePasskeys}`; the page's `GET /api/v1/reset-links/by-token/{token}` and `POST .../pin`.

## Driving it with fakeworld, playwright cli and the host CLI

Preconditions:

- A `home` run with doctor passing, the `{runId}-admin-desktop` session, and Grace signed in through `bin/sign-in.ts {runDir} member desktop` and loaded into `{runId}-member-desktop` on the portal's Home.

Steps:

- **Send the link.** As the Admin open Accounts, then `Grace Hopper`, choose `Send a Reset link` and `Send the Reset link`. The dialog reads `Emailed it to grace@example.test.` above the field `Reset link`.

  ```sh
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop goto '{nodes.home.browserUrl}/accounts'
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop click 'getByRole("table", { name: "Accounts" }).getByRole("link", { name: "Grace Hopper" })'
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop click 'getByRole("button", { name: "Send a Reset link" })'
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop click 'getByRole("button", { name: "Send the Reset link" })'
  $FW exec $RUN --proof reset-links/reset-link-issue -- $PW cli -s={runId}-admin-desktop screenshot --filename={runDir}/proof/reset-links/reset-link-issue/dialog.png
  ```

- **Grace's tab signs out.** Her session lands on Sign in with `This device was signed out.`

  ```sh
  $FW exec $RUN -- $PW cli -s={runId}-member-desktop run-code 'async page => { await page.getByText("This device was signed out.").waitFor({ timeout: 10000 }); }'
  $FW exec $RUN --proof reset-links/reset-link-ends-sessions -- $PW cli -s={runId}-member-desktop screenshot --filename={runDir}/proof/reset-links/reset-link-ends-sessions/sign-in.png
  ```

- **Read the email and use the link.** List the `mail` record to Grace and open its link in her session: `Set a new PIN`, `Step 1 of 3`. Type `9753` twice. When Grace holds a passkey here, as she does after the [passkeys](./passkeys.md) recipe, step 2 lists it to delete and `Continue` moves on; with none, step 2 offers one and `Skip for now` moves on. Then `Open SlipStream`. Signing in with `1357` over the API now answers 401, and `9753` answers 200.

  ```sh
  $FW mail list $RUN --where 'to=["grace@example.test"]'
  $FW mail open $RUN <seq> --proof reset-links/reset-link-email
  $FW exec $RUN -- $PW cli -s={runId}-member-desktop goto '<link from message.txt>'
  $FW exec $RUN -- $PW cli -s={runId}-member-desktop click 'getByRole("textbox", { name: "New PIN", exact: true })'
  $FW exec $RUN -- $PW cli -s={runId}-member-desktop type 9753
  $FW exec $RUN -- $PW cli -s={runId}-member-desktop click 'getByRole("textbox", { name: "Type it again", exact: true })'
  $FW exec $RUN -- $PW cli -s={runId}-member-desktop type 9753
  $FW exec $RUN -- $PW cli -s={runId}-member-desktop click 'getByRole("button", { name: "Continue" })'
  $FW exec $RUN --proof reset-links/reset-link-redeem -- $PW cli -s={runId}-member-desktop screenshot --filename={runDir}/proof/reset-links/reset-link-redeem/done.png
  $FW exec $RUN --proof reset-links/reset-link-redeem -- curl -sS -i -X POST -H 'Content-Type: application/json' -d '{"email":"grace@example.test","pin":"1357"}' '{nodes.home.url}/api/v1/session'
  ```

- **From the host.** Run the command against the running Controller's data dir. It prints one Reset link and exits 0, and Grace's open tab signs out. Once used, the link's card reads `This Reset link was already used, and each link works only once. Sign in, or ask your Admin for a new one.`, naming no Admin since the host issued it. Setting `1357` again through it puts Grace back on the manifest's PIN.

  ```sh
  $FW exec $RUN --proof reset-links/reset-host-command -- {runDir}/bin/slipstream admin reset --data-dir {nodes.home.dataDir} grace@example.test
  $FW exec $RUN --proof reset-links/reset-host-command-refusals -- {runDir}/bin/slipstream admin reset --data-dir {nodes.home.dataDir} nobody@example.test
  $FW exec $RUN --proof reset-links/reset-host-command-refusals -- {runDir}/bin/slipstream admin reset --data-dir {nodes.seedbox.dataDir} grace@example.test
  ```

## Gotchas

- A Reset link is shown once and never kept: copy it from the dialog or the email before closing either.
- Opening a Reset link while signed in reads `You're signed in as <name>. Sign out to use this link.` Sign out from that card first.
- The host command and a running Node share `library.db`; against a running Node the command hands the session ending to it, so Grace's tabs sign out as they do for the dialog.
- `--address` sets the address only while none is set. `home` sets it at launch, so a different `--address` exits 3 with `SlipStream's address is already <address>. Leave out --address, or change it in Settings > Address`.
- Every Reset link ends Grace's sessions, and her new PIN isn't the manifest's, so `bin/sign-in.ts {runDir} member` fails afterwards. Sign her in by hand with the new PIN, or run reset recipes last.
