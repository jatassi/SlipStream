# Accounts

Accounts lists every Account with its name, email, role, passkey count and last sign-in, and an Account's page holds what an Admin does to it: Make Admin or Make Member, Turn off or Turn on, Change email, Send a Reset link, Sign out everywhere and Delete. A change to someone's role reaches their open tabs at once, so a demoted Admin lands on the portal without signing out, and the last Admin can't be demoted, turned off or deleted.

## Sub-features

- `accounts-list` `Accounts` lists Ada as an Admin and Grace as a Member in the table `Accounts`.
- `accounts-make-admin` `Make Admin` on a Member's page gives the admin flag; their open portal tab can now open Admin pages.
- `accounts-demote-regate` `Make Member`, confirmed in `Make <name> a Member?`, moves the person's open Admin tab to `/portal` with no sign-in.
- `accounts-last-admin` demoting the only Admin is refused inside the dialog with `SlipStream needs at least one Admin who is turned on`, as the API's 409 `last_admin`.
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
  $FW exec $RUN -- $PW cli -s={runId}-member-desktop goto '{nodes.home.browserUrl}/nodes'
  $FW exec $RUN --proof accounts/accounts-make-admin -- $PW cli -s={runId}-member-desktop screenshot --filename={runDir}/proof/accounts/accounts-make-admin/nodes.png
  ```

- **Make her a Member again.** Choose `Make Member` and confirm. Grace's tab, still on Nodes, moves to the portal's Home by itself.

  ```sh
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop click 'getByRole("button", { name: "Make Member" })'
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop click 'getByRole("dialog", { name: "Make Grace Hopper a Member?" }).getByRole("button", { name: "Make Member" })'
  $FW exec $RUN -- $PW cli -s={runId}-member-desktop run-code 'async page => { await page.getByRole("heading", { name: "Home" }).waitFor({ timeout: 10000 }); }'
  $FW exec $RUN --proof accounts/accounts-demote-regate -- $PW cli -s={runId}-member-desktop screenshot --filename={runDir}/proof/accounts/accounts-demote-regate/portal.png
  ```

- **The last Admin.** Open `Ada Okafor`, choose `Make Member` and confirm. The dialog stays open with the alert `SlipStream needs at least one Admin who is turned on`; `Cancel` closes it.

  ```sh
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop goto '{nodes.home.browserUrl}/accounts'
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop click 'getByRole("table", { name: "Accounts" }).getByRole("link", { name: "Ada Okafor" })'
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop click 'getByRole("button", { name: "Make Member" })'
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop click 'getByRole("dialog", { name: "Make Ada Okafor a Member?" }).getByRole("button", { name: "Make Member" })'
  $FW exec $RUN --proof accounts/accounts-last-admin -- $PW cli -s={runId}-admin-desktop snapshot --filename={runDir}/proof/accounts/accounts-last-admin/refused.aria.yml
  ```

- **Turn one off.** On a Member who isn't needed later, such as Lin from the [signup links](./signup-links.md) recipe, choose `Turn off` and confirm `Turn off Lin Wei?`. The page offers `Turn on`, and Lin's sign-in answers 403 `account_disabled`.

  ```sh
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop goto '{nodes.home.browserUrl}/accounts'
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop click 'getByRole("table", { name: "Accounts" }).getByRole("link", { name: "Lin Wei" })'
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop click 'getByRole("button", { name: "Turn off" })'
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop click 'getByRole("dialog", { name: "Turn off Lin Wei?" }).getByRole("button", { name: "Turn off" })'
  $FW exec $RUN --proof accounts/accounts-turn-off -- curl -sS -i -X POST -H 'Content-Type: application/json' -d '{"email":"lin@example.test","pin":"8642"}' '{nodes.home.url}/api/v1/session'
  ```

## Gotchas

- Every Account write is Reconfirmed ([re-confirm](./reconfirm.md)).
- `Turn off` and `Delete` end Grace's sessions; run them last, or reload her saved state after `Turn on`.
