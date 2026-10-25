# Settings: Address, Email and Sign-in

Settings holds the Controller-wide settings an Admin tunes. Address is the origin every link is built on, refusing an IP address and warning about plain HTTP. Email is the one SMTP server every SlipStream email goes through, from the Node that holds Notifying, with a write-only password and Send a test email. Sign-in holds every PIN limit and lifetime, each bounded by a floor and a ceiling.

## Sub-features

- `settings-address` Settings > Address shows the saved address, suggests the one in use while none is set, warns `Passkeys don't work on a plain-HTTP address. Use an https:// address.` for a plain-HTTP address other than `localhost` or `*.localhost`, refuses an IP address with `The address must be a name, not an IP address` and saves with `Saved the address`.
- `settings-email` Settings > Email reads `Every email SlipStream sends goes through <host>.` over the fields `Server`, `Port`, `Security`, `Username`, `Password`, `From address` and `From name`, keeps the password write-only and saves with `Saved the email settings`.
- `settings-email-test` `Send a test email` reads `Sent a test email to ada@example.test.` and the `mail` capture holds it; under `smtp-reject` it reads `The test email failed. The mail server responded: 550 5.1.1 Mailbox unavailable`.
- `settings-sign-in` Settings > Sign-in groups its settings under `Wrong PINs`, `Failed attempts from one address`, `Sessions and re-confirm` and `Link lifetimes`, saves one and refuses a value outside its bounds on the field, as `Between 3 and 9 wrong PINs.` while the PIN turns off after 10 (P-238).

## How to get to it (user POV)

- Desktop shell: the sidebar's `Settings`, `/settings/address`, `/settings/email`, `/settings/sign-in`. Phone shell: `More`, then `Settings`. The palette's `Go to Settings`.
- API: `GET` and `PUT /api/v1/identity/address`; `GET` and `PATCH /api/v1/mail/settings`, `POST /api/v1/mail/test`; `GET` and `PATCH /api/v1/identity/settings`.

## Driving it with fakeworld and playwright cli

Preconditions:

- A `home` run with doctor passing and the `{runId}-admin-desktop` session. The Scenario's `mail: true` saved the `smtp` front for `home` as the server, without TLS or sign-in.

Steps:

- **Address.** Open `/settings/address`. The field `Address` holds `{nodes.home.browserUrl}` with `Signup, Reset and email links use this address.` Typing `http://127.0.0.2:8484` and choosing `Save` is refused on the field; leave the page without saving another.

  ```sh
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop goto '{nodes.home.browserUrl}/settings/address'
  $FW exec $RUN --proof settings/settings-address -- $PW cli -s={runId}-admin-desktop screenshot --filename={runDir}/proof/settings/settings-address/address.png
  $FW exec $RUN --proof settings/settings-address -- curl -sS -H 'X-Api-Key: {apiKey}' '{nodes.home.url}/api/v1/identity/address'
  ```

- **Email and a test.** Open `/settings/email`. It reads `Every email SlipStream sends goes through 127.0.0.1.`, the `smtp` front's address, with `Security` set to `None` and `No password is saved.` Choose `Send a test email`: the page reads `Sent a test email to ada@example.test.` and the capture holds the message to Ada.

  ```sh
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop goto '{nodes.home.browserUrl}/settings/email'
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop click 'getByRole("button", { name: "Send a test email" })'
  $FW exec $RUN --proof settings/settings-email-test -- $PW cli -s={runId}-admin-desktop screenshot --filename={runDir}/proof/settings/settings-email-test/sent.png
  $FW mail list $RUN --where 'to=["ada@example.test"]'
  ```

- **A failing server.** Play `smtp-reject` and send the test again. The page reads `The test email failed. The mail server responded: 550 5.1.1 Mailbox unavailable`. Turn the cue off after.

  ```sh
  $FW cue $RUN smtp-reject
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop click 'getByRole("button", { name: "Send a test email" })'
  $FW exec $RUN --proof settings/settings-email-test -- $PW cli -s={runId}-admin-desktop screenshot --filename={runDir}/proof/settings/settings-email-test/rejected.png
  $FW cue $RUN smtp-reject '{"off": true}'
  ```

- **Sign-in.** Open `/settings/sign-in`, set `Wrong PINs before a wait` to `4` and save its group. The API reads the new value back; `2` is refused as out of range, on the field and over the API with the same bounds.

  ```sh
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop goto '{nodes.home.browserUrl}/settings/sign-in'
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop fill 'getByRole("textbox", { name: "Wrong PINs before a wait" })' 4
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop click 'getByRole("region", { name: "Wrong PINs" }).getByRole("button", { name: "Save" })'
  $FW exec $RUN --proof settings/settings-sign-in -- curl -sS -H 'X-Api-Key: {apiKey}' '{nodes.home.url}/api/v1/identity/settings'
  $FW exec $RUN --proof settings/settings-sign-in -- curl -sS -i -X PATCH -H 'X-Api-Key: {apiKey}' -H 'Content-Type: application/json' -d '{"pinWaitAfter":2}' '{nodes.home.url}/api/v1/identity/settings'
  ```

## Gotchas

- Every address an Admin enters must be a fake's front (the skill's isolation rules). Change the email server only to the `smtp` front the manifest lists for `home`; doctor's check 5 fails on any other.
- Saving Address, Email or Sign-in is Reconfirmed ([re-confirm](./reconfirm.md)).
- A cue stays on until turned off; leave `smtp-reject` on and every later Signup and Reset email fails with its reply.
