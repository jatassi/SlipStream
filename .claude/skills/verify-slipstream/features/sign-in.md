# Sign-in

A person signs in on the Controller with their email and four-digit PIN. A wrong PIN says so and clears the PIN, a sign-in from a deep link returns to the page asked for, and Sign out ends the session in every tab.

## Sub-features

- `sign-in-redirect` an anonymous visit to `/system` lands on `/sign-in?redirect=%2Fsystem`, and signing in returns there, which lands on Nodes at `/nodes/<id>`.
- `sign-in-wrong-pin` a wrong PIN shows the alert `The email or PIN is wrong` and empties the PIN.
- `sign-in-pin` the right email and PIN sign in; the email is matched without case or surrounding spaces.
- `sign-in-saved-state` `bin/sign-in.ts` signs in through the real page, and its saved state opens a signed-in session.
- `sign-out` `Sign out` lands on `/sign-in?reason=signed-out` with the status `You signed out.`

## How to get to it (user POV)

- Desktop and phone: any page of a signed-out browser on the Controller redirects to `/sign-in`, headed `Sign in to SlipStream`.
- Desktop shell: the sidebar's `Account` button, then `Sign out`. Phone shell: the `More` tab, then `Sign out`. Both shells: the palette's `Sign out`.
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

- **Right PIN.** Type the manifest's PIN, `2468`. `/system` opens and lands on Nodes at `/nodes/<id>`, headed `home`.

  ```sh
  $FW exec $RUN -- $PW cli -s={runId}-ada-desktop click 'getByRole("textbox", { name: "PIN", exact: true })'
  $FW exec $RUN -- $PW cli -s={runId}-ada-desktop type 2468
  $FW exec $RUN -- $PW cli -s={runId}-ada-desktop run-code 'async page => { await page.getByRole("heading", { name: "home", level: 1 }).waitFor({ timeout: 10000 }); }'
  $FW exec $RUN --proof sign-in/sign-in-redirect -- $PW cli -s={runId}-ada-desktop screenshot --filename={runDir}/proof/sign-in/sign-in-redirect/nodes.png
  ```

- **Sign out.** Choose `Account`, then `Sign out`. The page reads `Sign in to SlipStream` with the status `You signed out.` at `/sign-in?reason=signed-out`.

  ```sh
  $FW exec $RUN -- $PW cli -s={runId}-ada-desktop click 'getByRole("button", { name: "Account" })'
  $FW exec $RUN -- $PW cli -s={runId}-ada-desktop click 'getByRole("menuitem", { name: "Sign out" })'
  $FW exec $RUN --proof sign-in/sign-out -- $PW cli -s={runId}-ada-desktop snapshot --filename={runDir}/proof/sign-in/sign-out/sign-in.aria.yml
  $FW exec $RUN -- $PW cli -s={runId}-ada-desktop close
  ```

- **Saved state.** Run the skill's Drive block for `admin desktop`. `goto {nodes.home.browserUrl}/` lands on `/library` with the Library heading and `Ada Okafor` on the sidebar's `Account` button, without a sign-in page.

## Gotchas

- The PIN field submits on its fourth digit; there is no need to choose `Sign in`.
- The PIN is four digit boxes named `PIN`, `PIN digit 2 of 4` and so on. Click the first and `type` the digits, as a person does.
- Every sign-in comes from loopback, and sign-in is rate limited per IP. Sign each identity in once per run and reuse the saved state.
- A session loaded from `auth/admin-desktop.json` carries the same server session as the saved file, so signing out there ends the saved state too. Sign out from a separate session, as above, or run `bin/sign-in.ts` again afterwards.
