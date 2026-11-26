# Member access to Media Servers

An Admin offers Members the Media Servers shown to Members, on a Signup link or on a Member's Account page. On Plex the Member signs in to Plex on plex.tv's page from the Watch step, which emails every Admin, and the Admin's Share on Plex shares the Libraries for Members and accepts it with the Member's sign-in. On Jellyfin and Emby the Member makes their own user. The Member then reads How to watch in the Watch step and under Where to watch on Your Account. Remove access deletes a Plex share and turns a Jellyfin or Emby user off, and Offer again turns it back on. The `media-servers` Scenario seeds a Member in each state: Lin offered, Mae waiting, Ravi shared, Tomas active on Plex and Jellyfin, Una removed on Jellyfin, plus the spare plex.tv account `noor`.

## Sub-features

- `member-access-needs-you` Accounts' `Needs you` lists Mae Carter's Living room Plex with `Signed in through Plex as mae-plex, mae@example.test.` and `Share on Plex`, `Add to Plex Home`, `Share by hand instead` and `Remove access`, and Ravi Menon's with `Waiting for Ravi Menon to accept the share in Plex's email.` and `Check again` (P-366).
- `member-access-share-plex` `Share on Plex` moves the row to `active` and out of Needs you, plex.tv records the share (`plex-invites`), and the Member gets `You can watch Living room Plex on Plex` in the `mail` capture (P-366).
- `member-access-signup-offer` `New Signup link` lists every server shown to Members under `Media Servers`, ticked, with `Add on Plex as` for Plex, and the link carries the servers left ticked (P-366).
- `member-access-watch-step` the person invited reads the `Watch` step, `Step 5 of 6`, with a block per offered server: `Make my Jellyfin account` for Jellyfin and `Continue with Plex` for Plex (P-366).
- `member-access-make-account` `Make my Jellyfin account` makes a user that isn't an administrator and sees only the Libraries for Members (`jellyfin-users`), and the block turns into `How to watch Den Jellyfin` (P-366).
- `member-access-plex-sign-in` `Continue with Plex` opens plex.tv's page in a new tab; allowing it as `noor` reads `Signed in to Plex as noor.`, leaves the row `waiting`, and emails Ada `Noor Haddad wants to watch on Living room Plex` (P-366).
- `member-access-where-to-watch` Your Account's `Where to watch` lists `How to watch Living room Plex` with `Sign in as tomas-plex` (P-366).
- `member-access-remove` `Remove access` on Tomas Reyes's Den Jellyfin asks `Remove Tomas Reyes's access to Den Jellyfin?`, then reads `Access removed. The account on Den Jellyfin is turned off.` with the user disabled in `jellyfin-users` (P-366).
- `member-access-reoffer` `Offer again` on Una Byrne's removed Den Jellyfin turns her user back on and reads `Can watch as una.` (P-366).

## How to get to it (user POV)

- Desktop shell, Admin: the sidebar's `Accounts`, its `Needs you` list and `New Signup link`; a Member's row opens their Account page with its `Media Servers` section. Phone shell: `More`, then `Accounts`.
- The person invited: the Signup link, whose `Watch` step follows the passkey step at `/welcome/watch`; plex.tv's page opens in a new tab on a wide screen and in the same tab on a phone.
- A Member: `Your Account` (`/portal/account`), its `Where to watch` section.
- API: `GET /api/v1/media-servers/access`, `PUT /api/v1/accounts/{id}/media-servers/{serverId}`, `POST /api/v1/media-servers/access/{id}/share`, `DELETE /api/v1/media-servers/access/{id}`, the Member's `GET /api/v1/account/media-servers`, `POST /api/v1/account/media-servers/plex/pin` and `POST /api/v1/account/media-servers/{accessId}/account`; live events `media-servers.access-changed` and `media-servers.my-access-changed`.
- Host CLI: none.

## Driving it with fakeworld and playwright cli

Preconditions:

- A `media-servers` run (`dist/fakeworld start --binary dist/slipstream --scenario media-servers --run-dir $RUN`), doctor passing, `{runId}-admin-desktop` signed in on Accounts as in the skill's Drive.

Steps:

- **Read Needs you, then share.** Accounts lists Mae's and Ravi's Plex rows under `Needs you`. Choose `Share on Plex` on Mae's; her row leaves the list and reads `active`.

  ```sh
  $FW exec $RUN --proof member-media-access/member-access-needs-you -- $PW cli -s={runId}-admin-desktop snapshot 'getByRole("main")' --filename={runDir}/proof/member-media-access/member-access-needs-you/accounts.aria.yml
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop click 'getByRole("list", { name: "Needs you" }).getByRole("listitem").filter({ hasText: "Mae Carter" }).getByRole("button", { name: "Share on Plex" })'
  $FW exec $RUN --proof member-media-access/member-access-share-plex -- curl -sS -H 'X-Api-Key: {apiKey}' '{nodes.home.url}/api/v1/media-servers/access'
  $FW mail list $RUN
  $FW plex-invites list $RUN
  ```

- **Remove, then offer again.** Open Tomas Reyes from the `Accounts` table, choose `Remove access` on Den Jellyfin and confirm in the dialog. Then open Una Byrne and choose `Offer again` on Den Jellyfin.

  ```sh
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop click 'getByRole("table", { name: "Accounts" }).getByRole("link", { name: "Tomas Reyes" })'
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop click 'getByRole("list", { name: "Media Servers" }).getByRole("listitem").filter({ hasText: "Den Jellyfin" }).getByRole("button", { name: "Remove access" })'
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop click 'getByRole("dialog").getByRole("button", { name: "Remove access" })'
  $FW jellyfin-users list $RUN
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop click 'getByRole("list", { name: "Media Servers" }).getByRole("listitem").filter({ hasText: "Den Jellyfin" }).getByRole("button", { name: "Offer again" })'
  ```

- **Where to watch.** Sign in as Tomas (`bun web/harness/bin/sign-in.ts {runDir} tomas@example.test desktop`, then `open` and `state-load` a `{runId}-tomas-desktop` session) and open `{nodes.home.browserUrl}/portal/account`.

- **Join with Plex and Jellyfin.** Issue `New Signup link` for `noor@example.test` with Den Emby unticked, then open the link in a fresh `{runId}-noor-desktop` session and join as `Noor Haddad`, typing the PIN into `PIN` twice and choosing `Skip for now` on the passkey step. On `Watch`, make the Jellyfin user, then choose `Continue with Plex`, pick `noor` and `Allow` on plex.tv's tab, close it and read the Plex block.

  ```sh
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop uncheck 'getByRole("dialog").getByRole("checkbox", { name: "Den Emby" })'
  $FW exec $RUN -- $PW cli -s={runId}-noor-desktop fill 'getByRole("region", { name: "Den Jellyfin" }).getByRole("textbox", { name: "Username" })' noor
  $FW exec $RUN -- $PW cli -s={runId}-noor-desktop fill 'getByRole("region", { name: "Den Jellyfin" }).getByRole("textbox", { name: "Password" })' orchard-lamp-41
  $FW exec $RUN -- $PW cli -s={runId}-noor-desktop click 'getByRole("button", { name: "Make my Jellyfin account" })'
  $FW exec $RUN -- $PW cli -s={runId}-noor-desktop run-code 'async page => { await Promise.all([page.context().waitForEvent("page"), page.getByRole("button", { name: "Continue with Plex" }).click()]); }'
  $FW exec $RUN -- $PW cli -s={runId}-noor-desktop tab-select 1
  $FW exec $RUN -- $PW cli -s={runId}-noor-desktop select 'getByRole("combobox", { name: "Plex account" })' noor
  $FW exec $RUN -- $PW cli -s={runId}-noor-desktop click 'getByRole("button", { name: "Allow" })'
  $FW exec $RUN -- $PW cli -s={runId}-noor-desktop tab-close 1
  $FW exec $RUN --proof member-media-access/member-access-plex-sign-in -- curl -sS -H 'X-Api-Key: {apiKey}' '{nodes.home.url}/api/v1/media-servers/access'
  ```

  Share Noor's row from Accounts as above; the Watch step's Plex block turns into `How to watch Living room Plex` live.

## Gotchas

- The launch itself sends email: `wants to watch` to Ada for Mae, Ravi and Tomas, and `You can watch` to Ravi and Tomas. Read the `mail` capture by recipient.
- Ravi's share stays `shared`: the world signed his SlipStream token out on plex.tv (`plextv-sign-out-member`), so neither Check again nor the renewal's check can accept it.
- plex.tv's page lists every account the world holds, `plex-admin` first and selected. Pick the account before `Allow`, or the sign-in is refused as the account that owns the servers.
- Removing Plex access deletes the row, so `removed` exists only on Jellyfin and Emby.
- A Member's Jellyfin or Emby user from the Scenario is the local part of their email, with their PIN as its password.
