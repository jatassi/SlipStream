# Sign-in

A person signs in on the Controller with their email and four-digit PIN, or with a passkey at an address that can hold one. The browser remembers the email. A wrong PIN says so and clears the PIN; five in a row bring a wait and ten turn the PIN off, and each address counts its own failures. A sign-in from a deep link returns to the page asked for, and Sign out ends the session in every tab.

## Sub-features

- `sign-in-redirect` an anonymous visit to `/system` lands on `/sign-in?redirect=%2Fsystem`, and signing in returns to This Node.
- `sign-in-wrong-pin` a wrong PIN shows the alert `The email or PIN is wrong` and empties the PIN.
- `sign-in-pin` the right email and PIN sign in; the email is matched without case or surrounding spaces.
- `sign-in-saved-state` `bin/sign-in.ts` signs in through the real page, and its saved state opens a signed-in session.
- `sign-out` `Sign out` asks `Sign out of this device?`, then lands on `/sign-in?reason=signed-out` with the status `You signed out.` (P-011)
- `sign-in-member` a Member's sign-in lands on the portal's Home at `/portal`, and `bin/sign-in.ts {runDir} member <shell>` saves Grace's state (P-233).
- `sign-in-remembered-email` the email that last signed in fills the Email field on the next visit (P-233).
- `sign-in-pin-wait` past the wrong-PIN limit the alert names when to try again, and the right PIN waits too (P-238).
- `sign-in-per-address` past the per-address limit every sign-in from that address reads `Too many failed attempts from this address. Wait before trying again.` with the time (P-238).

## How to get to it (user POV)

- Desktop and phone: any page of a signed-out browser on the Controller redirects to `/sign-in`, headed `Sign in to SlipStream`.
- Desktop shell: the sidebar's `Account` button, then `Sign out`. Phone shell: the `More` tab, then `Sign out`. Both shells: the palette's `Sign out`.
- Phone shell: the PIN field sits above the keypad at the Touch density.
- API: `POST /api/v1/session` with email and PIN; `DELETE /api/v1/session`.

## Driving it with fakeworld and playwright cli

Preconditions:

- A `home` run with doctor passing, and `$FW browser $RUN` run once.
- A session with no saved state, named for the person signing in: `{runId}-admin-desktop` before `state-load`, or a second one such as `{runId}-ada-desktop`.

Steps:

- **Deep link signed out.** Open `/system`. The page reads `Sign in to SlipStream` at `/sign-in?redirect=%2Fsystem`.

  ```sh
  $FW exec $RUN -- $PW cli -s={runId}-ada-desktop open --config={runDir}/browser/desktop/playwright-cli.json '{nodes.home.browserUrl}/system'
  ```

- **Wrong PIN.** Enter `ada@example.test` and PIN `1357`. The PIN submits itself on the fourth digit, the alert reads `The email or PIN is wrong`, and the PIN boxes are empty.

  ```sh
  $FW exec $RUN -- $PW cli -s={runId}-ada-desktop fill 'getByRole("textbox", { name: "Email" })' "ada@example.test"
  $FW exec $RUN -- $PW cli -s={runId}-ada-desktop click 'getByRole("textbox", { name: "PIN", exact: true })'
  $FW exec $RUN -- $PW cli -s={runId}-ada-desktop type 1357
  $FW exec $RUN --proof sign-in/sign-in-wrong-pin -- $PW cli -s={runId}-ada-desktop snapshot --filename={runDir}/proof/sign-in/sign-in-wrong-pin/form.aria.yml
  ```

- **Right PIN.** Type the manifest's PIN, `2468`. This Node opens at `/system`.

  ```sh
  $FW exec $RUN -- $PW cli -s={runId}-ada-desktop click 'getByRole("textbox", { name: "PIN", exact: true })'
  $FW exec $RUN -- $PW cli -s={runId}-ada-desktop type 2468
  $FW exec $RUN -- $PW cli -s={runId}-ada-desktop run-code 'async page => { await page.getByRole("heading", { name: "This Node" }).waitFor({ timeout: 10000 }); }'
  $FW exec $RUN --proof sign-in/sign-in-redirect -- $PW cli -s={runId}-ada-desktop screenshot --filename={runDir}/proof/sign-in/sign-in-redirect/this-node.png
  ```

- **Sign out.** Choose `Account`, then `Sign out`. The page reads `Sign in to SlipStream` with the status `You signed out.` at `/sign-in?reason=signed-out`.

  ```sh
  $FW exec $RUN -- $PW cli -s={runId}-ada-desktop click 'getByRole("button", { name: "Account" })'
  $FW exec $RUN -- $PW cli -s={runId}-ada-desktop click 'getByRole("menuitem", { name: "Sign out" })'
  $FW exec $RUN --proof sign-in/sign-out -- $PW cli -s={runId}-ada-desktop snapshot --filename={runDir}/proof/sign-in/sign-out/sign-in.aria.yml
  $FW exec $RUN -- $PW cli -s={runId}-ada-desktop close
  ```

- **Saved state.** Run the skill's Drive block for `admin desktop`. `goto {nodes.home.browserUrl}/` lands on `/library` with the Library heading and `Ada Okafor` on the sidebar's `Account` button, without a sign-in page.

- **A Member on the phone.** Sign Grace in and open the `member-phone` session. `{nodes.home.browserUrl}/` lands on the portal's Home with the tab bar `Tabs`.

  ```sh
  $FW exec $RUN -- bun web/harness/bin/sign-in.ts {runDir} member phone
  $FW exec $RUN -- $PW cli -s={runId}-member-phone open --config={runDir}/browser/phone/playwright-cli.json
  $FW exec $RUN -- $PW cli -s={runId}-member-phone state-load {runDir}/auth/member-phone.json
  $FW exec $RUN -- $PW cli -s={runId}-member-phone goto {nodes.home.browserUrl}/
  $FW exec $RUN --proof sign-in/sign-in-member -- $PW cli -s={runId}-member-phone screenshot --filename={runDir}/proof/sign-in/sign-in-member/portal.png
  ```

- **The wait.** Lower the limit to its floor, three wrong PINs, through the API, then type three wrong PINs for Grace over the API. The fourth answer is 429 `pin_wait` with `Retry-After`, and the sign-in page shows the time to try again.

  ```sh
  $FW exec $RUN -- curl -sS -X PATCH -H 'X-Api-Key: {apiKey}' -H 'Content-Type: application/json' -d '{"pinWaitAfter":3,"pinWait":60}' '{nodes.home.url}/api/v1/identity/settings'
  $FW exec $RUN -- curl -sS -X POST -H 'Content-Type: application/json' -d '{"email":"grace@example.test","pin":"0000"}' '{nodes.home.url}/api/v1/session'
  $FW exec $RUN -- curl -sS -X POST -H 'Content-Type: application/json' -d '{"email":"grace@example.test","pin":"0000"}' '{nodes.home.url}/api/v1/session'
  $FW exec $RUN -- curl -sS -X POST -H 'Content-Type: application/json' -d '{"email":"grace@example.test","pin":"0000"}' '{nodes.home.url}/api/v1/session'
  $FW exec $RUN --proof sign-in/sign-in-pin-wait -- curl -sS -i -X POST -H 'Content-Type: application/json' -d '{"email":"grace@example.test","pin":"1357"}' '{nodes.home.url}/api/v1/session'
  ```

## Gotchas

- The PIN field submits on its fourth digit; there is no need to choose `Sign in`.
- The PIN is four digit boxes named `PIN`, `PIN digit 2 of 4` and so on. Click the first and `type` the digits, as a person does.
- Every sign-in comes from loopback, and each address counts its failures: 30 failed sign-ins in 15 minutes, by default, refuse every sign-in from loopback until the window passes, whoever signs in. Successes never count. Sign each identity in once per run, reuse the saved state, and put `sign-in-pin-wait` last or on its own run.
- `bin/sign-in.ts` signs in with the manifest's PIN. After a Reset link or Change PIN has replaced it, sign that person in by hand.
- A session loaded from `auth/admin-desktop.json` carries the same server session as the saved file, so signing out there ends the saved state too. Sign out from a separate session, as above, or run `bin/sign-in.ts` again afterwards.
