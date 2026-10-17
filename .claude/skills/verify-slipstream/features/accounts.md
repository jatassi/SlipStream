# Accounts

Accounts lists every Account with its name, email, role, passkey count and last sign-in, and an Account's page holds what an Admin does to it: Make Admin or Make Member, Turn off or Turn on, Change email, Send a Reset link, Sign out everywhere and Delete. A change to someone's role reaches their open tabs at once, so a demoted Admin lands on the portal without signing out, and the last Admin can't be demoted, turned off or deleted.

## Sub-features

- `accounts-list` `Accounts` lists Ada as an Admin and Grace as a Member in the table `Accounts`.
- `accounts-make-admin` `Make Admin` on a Member's page gives the admin flag; their open portal tab can now open Admin pages.
- `accounts-demote-regate` `Make Member`, confirmed in `Make <name> a Member?`, moves the person's open Admin tab to `/portal` with no sign-in.
- `accounts-last-admin` demoting the only Admin is refused inside the dialog.
- `accounts-turn-off` `Turn off` ends the Account's sessions, and its sign-in reads `This Account is turned off. Ask an Admin to turn it back on.`
- `accounts-portal` an Admin opens the portal's pages as a Member does (P-317).

## How to get to it (user POV)

- Desktop shell: the sidebar's `Accounts`, then a name. Phone shell: `More`, then `Accounts`. The palette's `Go to Accounts`.
- API: `GET /api/v1/accounts`, `GET`, `PATCH` and `DELETE /api/v1/accounts/{id}`.

## Driving it with fakeworld and playwright cli

Preconditions:

- A `home` run with doctor passing, the `{runId}-admin-desktop` session, and Grace in `{runId}-member-desktop` from `bin/sign-in.ts {runDir} member desktop`.

Steps:

- **List.** Accounts shows both Accounts; the API agrees.

  ```sh
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop goto '{nodes.home.browserUrl}/accounts'
  $FW exec $RUN --proof accounts/accounts-list -- $PW cli -s={runId}-admin-desktop snapshot --filename={runDir}/proof/accounts/accounts-list/accounts.aria.yml
  $FW exec $RUN --proof accounts/accounts-list -- curl -sS -H 'X-Api-Key: {apiKey}' '{nodes.home.url}/api/v1/accounts'
  ```

- **Make Grace an Admin.** Open `Grace Hopper` and choose `Make Admin`, then confirm. Grace's tab can open `/system`.

  ```sh
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop click 'getByRole("table", { name: "Accounts" }).getByRole("link", { name: "Grace Hopper" })'
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop click 'getByRole("button", { name: "Make Admin" })'
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop click 'getByRole("dialog", { name: "Make Grace Hopper an Admin?" }).getByRole("button", { name: "Make Admin" })'
  $FW exec $RUN -- $PW cli -s={runId}-member-desktop goto '{nodes.home.browserUrl}/system'
  $FW exec $RUN --proof accounts/accounts-make-admin -- $PW cli -s={runId}-member-desktop screenshot --filename={runDir}/proof/accounts/accounts-make-admin/this-node.png
  ```

- **Make her a Member again.** Choose `Make Member` and confirm. Grace's tab, still on This Node, moves to the portal's Home by itself.

  ```sh
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop click 'getByRole("button", { name: "Make Member" })'
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop click 'getByRole("dialog", { name: "Make Grace Hopper a Member?" }).getByRole("button", { name: "Make Member" })'
  $FW exec $RUN -- $PW cli -s={runId}-member-desktop run-code 'async page => { await page.getByRole("heading", { name: "Home" }).waitFor({ timeout: 10000 }); }'
  $FW exec $RUN --proof accounts/accounts-demote-regate -- $PW cli -s={runId}-member-desktop screenshot --filename={runDir}/proof/accounts/accounts-demote-regate/portal.png
  ```

## Gotchas

- Every Account write is Reconfirmed ([re-confirm](./reconfirm.md)).
- `Turn off` and `Delete` end Grace's sessions; run them last, or reload her saved state after `Turn on`.
