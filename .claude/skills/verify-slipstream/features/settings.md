# Settings: its groups, Address, Email, Sign-in and Library scan

Settings holds the Controller-wide settings an Admin tunes, with its sections in groups ([docs/settings-groups.md](../../../../docs/settings-groups.md)). Address is the origin every link is built on, refusing an IP address and warning about plain HTTP. Email is the one SMTP server every SlipStream email goes through, from the Node that holds Notifying, with a write-only password and Send a test email. Sign-in holds every PIN limit and lifetime, each bounded by a floor and a ceiling. Library scan holds how often the scan runs, the share of a root's files gone that stops it, and how many files each Node reads media info from at once.

## Sub-features

- `settings-groups` On a desktop browser, Settings opens on Content Types with a column named `Settings` holding the groups `Library` (Content Types, Library roots, Library scan, Metadata), `Files to keep` (Targets, Profiles, Editions), `Search and download` (Indexers, Download Clients), `Access` (Address, Sign-in, API key) and `System` (Nodes, Email), in that order. On phone, `/settings` lists one group of rows per group under the same labels, and each row pushes its section with a back link `Settings`.
- `settings-address` Settings > Address shows the saved address, suggests the one in use while none is set, warns `Passkeys don't work on a plain-HTTP address. Use an https:// address.` for a plain-HTTP address other than `localhost` or `*.localhost`, refuses an IP address with `The address must be a name, not an IP address` and saves with `Saved the address`.
- `settings-email` Settings > Email reads `Every email SlipStream sends goes through <host>.` over the fields `Server`, `Port`, `Security`, `Username`, `Password`, `From address` and `From name`, keeps the password write-only and saves with `Saved the email settings`.
- `settings-email-test` `Send a test email` reads `Sent a test email to ada@example.test.` and the `mail` capture holds it; under `smtp-reject` it reads `The test email failed: 550 5.1.1 Mailbox unavailable`.
- `settings-sign-in` Settings > Sign-in groups its settings under `Wrong PINs`, `Failed attempts from one address`, `Sessions and re-confirm` and `Link lifetimes`, saves one and refuses a value outside its bounds on the field, as `Between 3 and 9 wrong PINs.` while the PIN turns off after 10 (P-238).
- `settings-library-scan` Settings > Library scan (`/settings/library-scan`) reads `Library scan compares each library root with SlipStream's records. …`, `Runs every` `1` `day` with `Default 1 day. Between 1 day and 7 days.`, and under `Details` `Stop a scan when this share of a root's files is gone` `25` `%` and `Files each Node reads media info from at once` `2` `files`, then `Each root's last scan and Scan now are in Library roots.` The cadence and the two settings each save with their own `Save`: 2 days reads `Every 2 days in the Maintenance window` on the Tasks page, and 50% saves `library.scan.stopShare` 50 at `GET /api/v1/imports/settings`. The section carries Library's Health dot, `Library scan 1 warning` on phone's list, with its worst item at its top and `Open Health` (P-230).
- `settings-tv` on `series`, Settings > Content Types' TV row holds TV's add defaults (Profile, root, `Episodes to monitor`, `Include specials`, `Season folders`), then `Delay after an air date`, 0 days by default and at most 30, then Refresh all; the add sheet's Remember writes the same values.

## How to get to it (user POV)

- Desktop shell: the sidebar's `Settings`, `/settings/address`, `/settings/email`, `/settings/sign-in`. Phone shell: `More`, then `Settings`. The palette's `Go to Settings`.
- API: `GET` and `PUT /api/v1/identity/address`; `GET` and `PATCH /api/v1/mail/settings`, `POST /api/v1/mail/test`; `GET` and `PATCH /api/v1/identity/settings`; `GET` and `PATCH /api/v1/imports/settings` (`library.scan.stopShare`, 1 to 100, and `library.scan.probes`, 1 to 8) and `PATCH /api/v1/tasks/library-scan` for the cadence.

## Driving it with fakeworld and playwright cli

Preconditions:

- A `home` run with doctor passing and the `{runId}-admin-desktop` and `{runId}-admin-phone` sessions. The Scenario's `mail: true` saved the `smtp` front for `home` as the server, without TLS or sign-in.

Steps:

- **Groups.** Open `/settings`. It lands on `/settings/content-types`, and the `Settings` column shows the five groups with their links. Open `/settings` in the `{runId}-admin-phone` session and it lists the same groups.

  ```sh
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop goto '{nodes.home.browserUrl}/settings'
  $FW exec $RUN --proof settings/settings-groups -- $PW cli -s={runId}-admin-desktop screenshot --filename={runDir}/proof/settings/settings-groups/desktop.png
  $FW exec $RUN -- $PW cli -s={runId}-admin-phone goto '{nodes.home.browserUrl}/settings'
  $FW exec $RUN --proof settings/settings-groups -- $PW cli -s={runId}-admin-phone screenshot --filename={runDir}/proof/settings/settings-groups/phone.png
  ```

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

- **A failing server.** Play `smtp-reject` and send the test again. The page reads `The test email failed: 550 5.1.1 Mailbox unavailable`. Turn the cue off after.

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

- **Library scan.** Open `/settings/library-scan` and its `Details`. Set `Runs every` to 2 and save that form, then the stop share to 50 and save the Details form. Read both back.

  ```sh
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop goto '{nodes.home.browserUrl}/settings/library-scan'
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop click 'getByRole("main").getByRole("button", { name: "Details" })'
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop fill 'getByRole("textbox", { name: "Runs every" })' 2
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop click 'locator("form").filter({ hasText: "Runs every" }).getByRole("button", { name: "Save" })'
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop fill 'getByRole("textbox", { name: "Stop a scan when this share of a root'"'"'s files is gone" })' 50
  $FW exec $RUN -- $PW cli -s={runId}-admin-desktop click 'locator("form").filter({ hasText: "Stop a scan" }).getByRole("button", { name: "Save" })'
  $FW exec $RUN --proof settings/settings-library-scan -- $PW cli -s={runId}-admin-desktop screenshot --filename={runDir}/proof/settings/settings-library-scan/saved.png
  $FW exec $RUN --proof settings/settings-library-scan -- curl -sS -H 'X-Api-Key: {apiKey}' '{nodes.home.url}/api/v1/imports/settings'
  $FW exec $RUN --proof settings/settings-library-scan -- curl -sS -H 'X-Api-Key: {apiKey}' '{nodes.home.url}/api/v1/tasks/library-scan'
  ```

## Gotchas

- Every address an Admin enters must be a fake's front (the skill's isolation rules). Change the email server only to the `smtp` front the manifest lists for `home`; doctor's check 5 fails on any other.
- Saving Address, Email or Sign-in is Reconfirmed ([re-confirm](./reconfirm.md)).
- A cue stays on until turned off; leave `smtp-reject` on and every later Signup and Reset email fails with its reply.
- Library scan holds two forms, the cadence and the Details settings, each with its own `Save`, so scope a click on `Save` to its form.
