# Sessions and devices

Your Account lists every device signed in as you, with its browser, system, address and last activity, and signs out one device or every device but this one. A tab whose session ended from somewhere else lands on Sign in with `This device was signed out.` An Admin sees and ends another Account's sessions from its Account page.

## Sub-features

- `sessions-devices` Your Account's `Devices` list shows each session with its device label and address, and marks this one (P-234).
- `sessions-sign-out-one` `Sign out <device>` asks `Sign out <device>?` and ends that session; its open tab lands on `/sign-in?reason=revoked` reading `This device was signed out.`
- `sessions-sign-out-others` `Sign out every other device` asks `Sign out every other device?` and keeps only this session.
- `sessions-admin-ends` on `/accounts/<id>`, `Sign out everywhere` asks `Sign <name> out everywhere?` and ends every session the Account holds.
- `sessions-sign-out-confirm` `Sign out` asks `Sign out of this device?` in both shells and the palette (P-011).

## How to get to it (user POV)

- Desktop shell: the sidebar's `Account` menu, then Your Account, `/account`. Phone shell: `More`, then the Admin's name. A Member: the portal's `Account`, `/portal/account`.
- An Admin: `Accounts`, then the person's name, `/accounts/<id>`.
- API: `GET /api/v1/sessions`, `DELETE /api/v1/sessions/{id}`, `DELETE /api/v1/sessions?others=true`; `GET` and `DELETE /api/v1/accounts/{id}/sessions[/{session}]`.

## Driving it with fakeworld and playwright cli

Preconditions:

- A `home` run with doctor passing, and `$FW browser $RUN` run once.
- Grace signed in twice: `bun web/harness/bin/sign-in.ts {runDir} member desktop` and `... member phone`, each loaded into its own session, `{runId}-member-desktop` and `{runId}-member-phone`, as the skill's Drive does for `admin`.

Steps:

- **List the devices.** On the desktop session open `/portal/account`. `Devices` lists two sessions, `Chrome on Linux` from `127.0.0.1`, and the desktop's is this device.

  ```sh
  $FW exec $RUN -- $PW cli -s={runId}-member-desktop goto '{nodes.home.browserUrl}/portal/account'
  $FW exec $RUN --proof sessions/sessions-devices -- $PW cli -s={runId}-member-desktop snapshot --filename={runDir}/proof/sessions/sessions-devices/devices.aria.yml
  ```

- **Sign out the phone.** Choose `Sign out every other device`, then `Sign out others`. The phone session's tab lands on Sign in with `This device was signed out.`

  ```sh
  $FW exec $RUN -- $PW cli -s={runId}-member-desktop click 'getByRole("button", { name: "Sign out every other device" })'
  $FW exec $RUN -- $PW cli -s={runId}-member-desktop click 'getByRole("dialog", { name: "Sign out every other device?" }).getByRole("button", { name: "Sign out others" })'
  $FW exec $RUN -- $PW cli -s={runId}-member-phone run-code 'async page => { await page.getByText("This device was signed out.").waitFor({ timeout: 10000 }); }'
  $FW exec $RUN --proof sessions/sessions-sign-out-others -- $PW cli -s={runId}-member-phone screenshot --filename={runDir}/proof/sessions/sessions-sign-out-others/phone.png
  ```

- **An Admin ends them all.** In the Admin session open Accounts, then `Grace Hopper`, choose `Sign out everywhere` and confirm. Grace's desktop tab lands on Sign in the same way, and the Account's sessions read empty.

  ```sh
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop goto '{nodes.home.browserUrl}/accounts'
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop click 'getByRole("table", { name: "Accounts" }).getByRole("link", { name: "Grace Hopper" })'
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop click 'getByRole("button", { name: "Sign out everywhere" })'
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop click 'getByRole("dialog", { name: "Sign Grace Hopper out everywhere?" }).getByRole("button", { name: "Sign out everywhere" })'
  $FW exec $RUN -- $PW cli -s={runId}-member-desktop run-code 'async page => { await page.getByText("This device was signed out.").waitFor({ timeout: 10000 }); }'
  ```

## Gotchas

- Every session comes from loopback, so every device reads `127.0.0.1`; tell them apart by the `current` mark and the device label.
- The world signs the session it redeemed Grace's Signup link with out again, so Grace starts with no devices.
- Ending another Account's sessions is Reconfirmed: an Admin session older than the re-confirm window gets the `Confirm it's you` dialog.
- A session loaded from `auth/member-desktop.json` is the same server session as the file, so ending it ends the saved state. Run `bin/sign-in.ts` again before reusing the file.
