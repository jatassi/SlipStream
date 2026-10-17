# Signup links and joining

An Admin issues a Signup link for an email on Accounts. SlipStream emails it when email is set up and always shows it to copy, lists it as pending until it is used, and lets the Admin copy it again, send it again or withdraw it. The person invited opens the link, is welcomed by the Admin's name, chooses a name and a PIN typed twice, may add a passkey, and lands signed in on the portal. A dead link says in one sentence why.

## Sub-features

- `signup-link-issue` `New Signup link` on Accounts takes an email, answers `Emailed it to <email>.` with the link in the read-only field `Signup link`, and lists it under `Pending Signup links` (P-307).
- `signup-link-email` the email reaches the `mail` capture with the subject `Ada Okafor invited you to <host>`, the button and the raw link in its text part.
- `signup-link-pending` a pending link's `Actions for <email>` menu has `Copy link`, `Send again` (a new link, the old one withdrawn) and `Withdraw` (P-307).
- `signup-link-account-exists` an email that already has an Account is refused on the field with `An Account already has that email`.
- `signup-link-address` with no address set, the dialog asks for `SlipStream's address` inline, saves it and then makes the link.
- `signup-join` the link opens the guided flow, `Step 1 of 5` on the Node's name and `Step 1 of 4` on its IP address, and ends on the portal's Home signed in as the new Member (P-310).
- `signup-join-phone` the same flow on the phone shell takes the PIN on the 80px keypad.
- `signup-dead-link` a used link says `This link was already used, and each link works once, so sign in or ask Ada Okafor for a new one.` with `Sign in`; a made-up one names nobody.
- `signup-signed-in-visitor` a signed-in visitor reads `You're signed in as <name>. Sign out to use this link.`

## How to get to it (user POV)

- Desktop shell: the sidebar's `Accounts`, then `New Signup link`. Phone shell: `More`, then `Accounts`. Both shells: the palette's `New Signup link`.
- The person invited: the link from the dialog's `Signup link` field or from the email, `{nodes.home.browserUrl}/signup#token=…`, on desktop or phone.
- API: `POST /api/v1/signup-links {email}`, `GET /api/v1/signup-links`, `GET /api/v1/signup-links/{id}`, `POST /api/v1/signup-links/{id}/resend`, `DELETE /api/v1/signup-links/{id}`; the page's `GET` and `POST /api/v1/signup-links/by-token/{token}[/account]`.

## Driving it with fakeworld and playwright cli

Preconditions:

- A `home` run with doctor passing, its address set to `{nodes.home.browserUrl}` and email going to the `smtp` front, which the Scenario does at launch.
- The `{runId}-admin-desktop` session from the skill's Drive, and a second session for the person invited, opened with no saved state: `{runId}-lin-phone` from `browser/phone/playwright-cli.json`.

Steps:

- **Issue a link.** On Accounts choose `New Signup link`, enter `lin@example.test` and choose `Make the link`. The dialog reads `Emailed it to lin@example.test.` and the field `Signup link` holds `{nodes.home.browserUrl}/signup#token=…`.

  ```sh
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop goto '{nodes.home.browserUrl}/accounts'
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop click 'getByRole("button", { name: "New Signup link" })'
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop fill 'getByRole("dialog", { name: "New Signup link" }).getByRole("textbox", { name: "Email" })' "lin@example.test"
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop click 'getByRole("button", { name: "Make the link" })'
  $FW exec $RUN --proof signup-links/signup-link-issue -- $PW cli -s={runId}-admin-desktop screenshot --filename={runDir}/proof/signup-links/signup-link-issue/dialog.png
  $FW exec $RUN --proof signup-links/signup-link-issue -- $PW cli -s={runId}-admin-desktop --raw eval 'document.querySelector("input[readonly]").value'
  ```

- **Read the email.** The capture holds one `smtp.message` to `lin@example.test`, sent before the dialog says `Emailed it to`, so `list` finds it. `open` copies `message.eml`, `message.html` and `message.txt`; the text part holds the same link.

  ```sh
  $FW mail list $RUN --where 'to=["lin@example.test"]'
  $FW mail open $RUN <seq> --proof signup-links/signup-link-email
  ```

- **Join on the phone.** Open the link from the email in the phone session. The page reads `Join SlipStream`, `Step 1 of 5` and `Ada Okafor invited you to home.localhost.` Choose `Get started`, type `Lin Wei` in `Your name`, choose `Continue`, tap `8642` on the keypad twice, then `Skip for now` on the passkey step and `Open SlipStream`. The portal's Home opens with the tab bar `Tabs`.

  ```sh
  $FW exec $RUN -- $PW cli -s={runId}-lin-phone open --config={runDir}/browser/phone/playwright-cli.json '<link from message.txt>'
  $FW exec $RUN --proof signup-links/signup-join-phone -- $PW cli -s={runId}-lin-phone screenshot --filename={runDir}/proof/signup-links/signup-join-phone/welcome.png
  $FW exec $RUN -- $PW cli -s={runId}-lin-phone click 'getByRole("button", { name: "Get started" })'
  $FW exec $RUN -- $PW cli -s={runId}-lin-phone fill 'getByRole("textbox", { name: "Your name" })' "Lin Wei"
  $FW exec $RUN -- $PW cli -s={runId}-lin-phone click 'getByRole("button", { name: "Continue" })'
  $FW exec $RUN -- $PW cli -s={runId}-lin-phone run-code 'async page => { const keypad = page.getByRole("group", { name: "Keypad" }); for (const d of "86428642") { await keypad.getByRole("button", { name: d, exact: true }).click(); } }'
  $FW exec $RUN --proof signup-links/signup-join-phone -- $PW cli -s={runId}-lin-phone screenshot --filename={runDir}/proof/signup-links/signup-join-phone/passkey-step.png
  $FW exec $RUN -- $PW cli -s={runId}-lin-phone click 'getByRole("button", { name: "Skip for now" })'
  $FW exec $RUN -- $PW cli -s={runId}-lin-phone click 'getByRole("button", { name: "Open SlipStream" })'
  $FW exec $RUN --proof signup-links/signup-join -- $PW cli -s={runId}-lin-phone screenshot --filename={runDir}/proof/signup-links/signup-join/portal.png
  ```

- **See it used.** Accounts lists `Lin Wei` as a Member and the pending list no longer holds the link. Opening the link again in the phone session reads the signed-in visitor sentence; after `Sign out` it reads the used-link card.

  ```sh
  $FW exec $RUN --proof signup-links/signup-join -- curl -sS -H 'X-Api-Key: {apiKey}' '{nodes.home.url}/api/v1/accounts'
  $FW exec $RUN --proof signup-links/signup-link-pending -- curl -sS -H 'X-Api-Key: {apiKey}' '{nodes.home.url}/api/v1/signup-links'
  ```

- **Withdraw and resend.** Issue a link for `alan@example.test`, then from its `Actions for alan@example.test` menu choose `Send again`. The dialog `Send a new Signup link?` reads `A new link goes to alan@example.test, and the old one stops working.`; confirm `Send again` and a second email arrives, and the first link's page reads `This link was replaced or cancelled, so ask Ada Okafor whether a new one is on its way.` `Withdraw` empties the list.

## Gotchas

- The link is built on the configured address, `{nodes.home.browserUrl}`, never the Node's IP. Open it in a Chromium session from the run's config, whose host-resolver rules resolve `home.localhost`.
- The token sits in the fragment, so it never shows in a request log. Read it from the dialog's field or from `message.txt`.
- `mail wait` waits for a record after the last step's start, and SlipStream sends the email before the dialog answers, so after the click use `mail list`.
- The launch makes Grace through a Signup link before it saves the email setting, so the `mail` capture starts empty.
- Issuing a link is a Reconfirmed action. A session signed in under 15 minutes ago is confirmed; an older one gets the `Confirm it's you` dialog ([re-confirm](./reconfirm.md)).
- `signup-link-address` needs a Controller with no address: run it on `single-machine`, which has no Members, so the world leaves the address unset.
