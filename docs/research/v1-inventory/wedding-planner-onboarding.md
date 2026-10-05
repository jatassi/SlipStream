# wedding-planner onboarding, end to end

Source: `~/Git/wedding-planner` at `37e720bb` (main, read 2026-10-04). Read-only survey. Every claim below
cites the file it came from. "Inferred" marks anything I did not read directly in code.

## 0. Vocabulary (what wedding-planner calls things)

wedding-planner does not call it an "invite" or "signup link". Its domain terms (`GLOSSARY.md`, People section):

- **Operator.** Login to the server, never to a wedding. Creates a wedding and appoints its first owner.
- **Organizer.** Login to one or more weddings. Owners and helpers are organizers.
- **Owner.** Max two per wedding. Holds every permission.
- **Helper.** Organizer with a set of explicit Grants (`guests:read`, `budget:write`, ...).
- **Pending appointment.** The DB row created when someone is appointed. Holds email, role, permissions,
  expiry, token.
- **Appointment link.** The emailed link. Works once, lasts seven days, grants to whoever follows it and
  signs in or up, not to the address it was sent to.
- **Guest "invite".** A different thing entirely (a household's RSVP link and code). Not organizer onboarding.

Governing decision: `docs/adr/0085-an-appointment-is-taken-through-its-link.md` (issue #977). Amends
`docs/adr/0002-two-account-types.md`.

Key ruling in ADR 0085: the address grants nothing. Organizer sign-up is open and unverified, so an
account with a matching email proves nothing about the inbox. Only possession of the link token proves it.
Every appointment goes through the link, whether or not the address already has an account. A typo in the
address cannot hand the seat to whoever registered that typo, because they would also need the link.

## 1. Creating the link (who, how, data)

### Who can appoint

| Appointer | Can appoint | Gate | File |
| --- | --- | --- | --- |
| Operator | First owner only | `operatorMayAppoint`: wedding has 0 owners and no live pending owner appointment to a different address | `packages/domain/src/AppointmentSeats.ts` |
| Owner | Second owner | `ownerMayAppoint`: owners + live pending owner appointments to other addresses < 2 | same |
| Owner | Helper with a permission set | Always allowed; unlimited pending helpers | `packages/domain/src/Appointments.ts` (`helperMayAppoint`) |

A pending owner appointment holds a seat until it expires, so it counts toward the cap of two.

Owner appoint/cancel/list are `Operation`s under the Owner-reserved `grants` resource, which ADR 0005's
closed list keeps off MCP. So no AI agent can appoint or cancel (ADR 0085, amended by #977).

### The row

Table `appointment` (migrations 0065, 0067 in `packages/domain/src/AppointmentMigration.ts`):

```
id TEXT PK            -- UUIDv7
wedding_id TEXT       -- FK wedding
email TEXT
role TEXT             -- 'owner' | 'helper'
permissions TEXT      -- JSON array of GrantPermission; '[]' for owner
made_by TEXT          -- organizer id, or '' when the Operator made it (Operator is never named)
made_at TEXT, expires_at TEXT   -- ISO; expires = made + 7 days
token TEXT            -- UNIQUE index, server-wide
```

No `state` column and no version. Making inserts. Taking, cancelling or replacing deletes. History lives in
the Activity row each act writes, plus the change feed.

Companion table `spent_link` (migration 0066): `token PK, wedding_id, ended ('used'|'withdrawn'), ended_at`.
When an appointment row is deleted its token is copied here, so a link followed later can say *why* it is
dead instead of a generic "invalid".

### Token format

`packages/domain/src/Appointments.ts` `drawToken`: 32 random bytes from Effect's `Crypto.randomBytes`, hex
encoded. 64 hex chars, 256 bits. Branded type `AppointmentToken` in `packages/api/src/Appointment.ts`.

Stored in plaintext in `appointment.token` and `spent_link.token`, not hashed (see Rough edges).

URL: `appointmentLink(baseUrl, token)` = `${baseUrl}/plan/appointment/${encodeURIComponent(token)}`
(`packages/api/src/Appointment.ts`). `baseUrl` comes from the `WeddingHost` service, so the link is on the
wedding's own host.

### Expiry and single use

- `appointmentDays = 7` (`packages/api/src/Appointment.ts`). Liveness is `now < expires_at`
  (`isLive` in `AppointmentSeats.ts`). Nothing writes when it expires; it is computed on read (ADR 0026,
  "nothing is written because time passed").
- Single use. A successful take deletes the row and writes `spent_link.ended = 'used'` in the same
  transaction (`takeAppointment` in `packages/domain/src/AppointmentTaking.ts`).
- Archiving the wedding disables all links (refusal `archived`) but the seven-day clock keeps running.
- A refusal never consumes the link. If the taker already holds everything it grants (`held`) or seats are
  full (`full`), the row stays so someone else can still take it.

### Make flow (`appointThrough` in `Appointments.ts`)

One SQL transaction:
1. Read owners + all pending appointments + now (`seatsNow`).
2. Check the role's `may` predicate, else refuse `state: vacant`.
3. `replacedBy`: delete same-role appointments to the same address (case-insensitive, trimmed), live or
   expired, plus for owner seats every expired owner appointment. Each deleted token goes to `spent_link`
   as `withdrawn`.
4. Insert the new row with a fresh token. Write the change-feed row and Activity row.
5. Return the `HeldAppointment` (with token) so the UI can show a copyable link.

The email is sent *after* the transaction, and a mail failure never rolls back the appointment.

## 2. The email

### Sending pipeline

1. The appoint act (`owner_appoint` for the Operator in `packages/domain/src/OperatorActs.ts`; `owner_appoint` and
   `helper_appoint` for owners in `packages/domain/src/grantBindings.ts`) commits the appointment, then calls
   `db.sendAppointmentEmail(held.id)`.
2. `appointmentEmailSenderFor` (`packages/domain/src/AppointmentEmail.ts`):
   - Archived wedding sends nothing (`archived`). Appointment row missing sends nothing (`gone`).
   - Resolves the appointer. Empty `made_by` is the Operator, described and never named. Otherwise the
     organizer's account name, falling back to their email.
   - Builds the link from `WeddingHost.baseUrl` + token. Formats expiry as a local date in the wedding timezone.
   - Calls `Mailer.send`. `MailerRefused` / `MailerUnreachable` / `MailerUnconfigured` are logged as a warning and
     mapped to `failed`. The appointment stays.
3. The send runs inline in the request, after the DB transaction. Callers discard the `sent|failed` result
   (see Rough edges).

### Provider and library

- Provider: **SMTP2GO**, over its HTTP API, not SMTP. `POST https://api.smtp2go.com/v3/email/send` with header
  `X-Smtp2go-Api-Key` (`packages/domain/src/Mailer.ts`). No mail library; a hand-written Effect `HttpClient` call
  with `Schema`-encoded body and decoded answer. Retries: `Schedule.exponential("1 second")` up to 2 times.
- Why SMTP2GO: free plan of 1,000/month and 200/day, chosen over Postmark on 2026-09-10
  (`docs/research/invite-delivery.md`, `docs/adr/0013-the-digest-is-three-sections.md`).
- Body sent: `sender`, `to`, `subject`, `html_body`, `text_body`, `fastaccept: false`, and custom headers
  `Reply-To`, `X-Wedding: <weddingId>`, `X-Wedding-Recipient: appointment:<appointmentId>`
  (`packages/domain/src/MailHeaders.ts`). The two `X-Wedding*` headers let the bounce webhook route a report
  to the right wedding and record without a cross-wedding address lookup.

### From address config

Env vars, read with Effect `Config` (`packages/domain/src/Mailer.ts`; documented in `deploy/README.md`):

| Var | Purpose |
| --- | --- |
| `WEDDING_MAIL_API_KEY` | SMTP2GO key. Optional. Absent means the `Mailer` is the unconfigured variant, logs `WEDDING_MAIL_API_KEY is not set, so no email will be sent`, and every send fails `MailerUnconfigured`. |
| `WEDDING_MAIL_FROM` | `From:` address, on the wedding's own domain. Required only when the key is set. |
| `WEDDING_MAIL_REPLY_TO` | `Reply-To:` (the couple's iCloud mailbox). |
| `WEDDING_MAIL_WEBHOOK_SECRET` | Bearer secret for `POST /hooks/mail` (bounce reports). |

The display name is per wedding: `senderNameOf` in `packages/domain/src/CommunicationDraft.ts` builds
"First and Second" from the guest site's names, falling back to one name, then the wedding name. So the
From line reads e.g. `Julia and Jackson <mail@atassi.org>`.

### What it looks like

Rendered by a pure function, `renderAppointmentEmail` (`packages/domain/src/AppointmentEmailRender.ts`), on the
shared table layout in `packages/domain/src/EmailFrame.ts`. All words come from the message catalog
(`packages/ui/src/messages/catalogOrganizer.ts`, keys `email.appointment.*`).

- **Subject.** "You have been added to {wedding}'s wedding"
- **HTML body rows.**
  1. Wedding name as page title.
  2. Who appointed you, one of three sentences:
     - Operator to owner. "The person who runs the server for {wedding} has appointed you an owner."
     - Organizer to owner. "{name} has appointed you an owner of {wedding}."
     - Organizer to helper. "{name} has added you as a helper on {wedding}."
  3. "Follow the link to sign up, or to sign in if you already have an account."
  4. Link styled as text: "Join {wedding}". It is not a bulletproof button.
  5. Caption. "The link works once, until {date}."
  6. Caption. "If you did not expect this email, you can ignore it. Nothing changes until the link is used."
  7. Inset footer with the full raw URL as a muted link, for copy-paste.
- **Plain-text part.** The same sentences, with "Join {wedding}: {link}".
- **Theming.** Organizer tokens from `@wedding/theme`, plus the wedding's two chosen colours as a rule pair.
  `EmailFrame.ts` enforces the email rules from `DESIGN.md` in one place: light scheme only, explicit
  bg/colour on every element, sRGB hex and px only (no oklch/rem/calc), no alpha, `mso-` font fallbacks so
  Outlook does not fall back to Times, gold never as text colour, no `<img>`, plain-text part with no markup.

### Bounces

`POST /hooks/mail` (`packages/web/src/MailHooks.ts` → `packages/domain/src/MailDoor.ts`) decodes
`X-Wedding-Recipient`. For `kind: appointment` it only logs ("The provider reported on an appointment email") and
records nothing. The rationale in the code is that **Copy link** is the fallback channel.

## 3. What the invitee walks through

URL: `https://<wedding-host>/plan/appointment/<64-hex-token>`. React route `appointmentRoute = "/appointment/$token"`
under the `/plan` base (`packages/plan/src/router.tsx`, `packages/plan/src/routes/appointment.tsx`).

### Screen 0. Loading

`AppointmentCard` renders nothing until both reads answer. The reads are `appointment_read` (what the link grants) and
`caller_read` (is a session open). No skeleton or spinner (`packages/plan/src/routes/AppointmentCard.tsx`).

`appointment_read` is unauthenticated: `POST /api/appointment/appointment_read {token}`. It returns
`{email, role, weddingName}` or `409 LinkRefusal{reason}` (`packages/api/src/Appointment.ts`,
`packages/web/src/AppointmentApi.ts`). The wedding comes from the request's `Host`, and an unknown host answers
`unknown`, the same as a bad token. So the endpoint tells a stranger nothing about which hosts exist.

### Screen 1a. Dead link (one sentence per reason)

The card heading is "Appointment link" and an `ErrorForm` shows the sentence for the reason
(`plan.appointment.refused.*`):

| Reason | Sentence |
| --- | --- |
| `unknown` | This link does not work. Check that you have all of it, or ask the person who sent it for a new one. |
| `withdrawn` | This link was withdrawn. Ask the person who sent it whether a new one is on its way. |
| `expired` | This link has expired. Links work for seven days. Ask the person who sent it for a new one. |
| `used` | This link was already used. Each link works once. Ask the person who sent it. |
| `archived` | This wedding is archived, so its links do not work for now. Ask the person who sent this one. |
| `held` | You already have everything this link gives, so nothing changed. The link was not used, so someone else can still take it. |
| `full` | This wedding has no free owner seat. The link was not used. Ask the person who sent it. |
| `unreached` (client only) | The link was not read or taken. The server did not answer. Try again. |

Lookup order in `openLink` (`packages/domain/src/AppointmentTaking.ts`): live table by token, then archived check,
then expiry. If the token is not in the live table, `spent_link` is checked and returns `used` or `withdrawn`;
otherwise `unknown`.

### Screen 1b. Live link, signed out: Sign up (default)

The heading is "Join {wedding}" and the remark is "You have been appointed an owner of {wedding}. Sign up to start
planning." (helper variant: "...added as a helper... Sign up to start helping.").

Form (`packages/plan/src/routes/AppointmentSignUp.tsx`):
- **Name.** Caption: "This name appears beside your changes."
- **Email address.** Prefilled with the appointed address and **editable**. The link grants, not the address.
- **Password.** Caption: "At least 8 characters." (`passwordLength = {min: 8, max: 128}`,
  `packages/api/src/OrganizerLogin.ts`). Has a show/hide toggle (`LoginPassword`).
- Primary button: **Sign up and join**.
- Tertiary button: **Sign in instead** swaps to 1c.

Field values live in local React state, so they survive a refused submit. Field ids match the normal login form
(`sign-up-name`, `sign-up-email`, `sign-up-password`), so the same test recipes drive both.

### Screen 1c. Live link, signed out: Sign in instead

The remark is "...Sign in to start planning." It shows the prefilled editable email and a password, the button
**Sign in and join**, and **Sign up instead** goes back (`AppointmentSignIn.tsx`).

### Screen 1d. Live link, already signed in: Confirm

The text is "You have been appointed an owner of {wedding}. Join with the account you are signed in with." The single
button is **Join as an owner** / **Join as a helper** (`AppointmentConfirm.tsx`). The code names a removed owner who
is appointed again as the typical user.

### Submit sequence (`packages/plan/src/appointment.ts`)

`signUpAndTake`:
1. `POST /api/auth/sign-up/email {email, name, password}` to better-auth (`Login.posted`). A failure maps
   better-auth codes `USER_ALREADY_EXISTS` / `PASSWORD_TOO_SHORT` / `PASSWORD_TOO_LONG` to specific sentences and
   anything else to a generic one. better-auth sets the session cookie on success. (Inferred: better-auth's default
   `autoSignIn` after sign-up. The code relies on the session existing for step 2.)
2. `POST /api/appointment/appointment_take {token}`. The server reads the organizer from the better-auth
   session (`organizerOfSession`, `packages/web/src/Authorization.ts`), then `takeAppointment` in one
   transaction:
   - Re-runs `openLink` (archived / expired / spent).
   - The caller is already an owner here → `held`.
   - Owner link: `appointOwner` with `SEATS = 2`; no free seat → `full`.
   - Helper link: writes only the Grants the caller lacks. All held → `held`. First grant begins membership.
   - Deletes the appointment, writes `spent_link(used)`, writes Activity row `appointment_take`
     ("Joined through an appointment link").
3. On success the client refreshes `caller_read` and the summary figures, then sets `landed = route "/"`.
   `LoginResume` navigates only once `caller_read` says owner/helper.
4. On take refusal the account **stays**, the link stays unused, and `AppointmentUntaken` shows the reason sentence
   under the form.

`signInAndTake` is the same with `/api/auth/sign-in/email`. `takeSignedIn` skips step 1.

### Screen 2. First run (first owner only)

The app root redirects an owner to Settings while `wedding.firstRun === "pending"` (`packages/plan/src/firstRun.ts`,
ADR 0043). `FirstRun.tsx` shows an `EmptyState` ("Time zone and currency", Sunrise icon) with the two prefilled
fields and two chips: **Confirm these** (writes both plus `firstRun: confirmed`) or **Keep the defaults** (writes
only the `firstRun` stamp). Both navigate to `/` (Today). The state lives on the wedding, not per owner, so the second
owner never sees it. A version conflict, where the other owner confirmed simultaneously, shows "The other owner already
confirmed these settings." Helpers skip first run.

### Screen 3. Today

The normal app.

### Stranger state (no link)

Organizer sign-up at `/plan/login` is open to anyone. A signed-in account with no seat sees
"You are signed in, but you are not part of this wedding yet. Ask one of its owners to add you." (or, before any
owner exists, "...The person who runs this server must appoint the first owner...").
`plan.login.stranger.*`. There is a **Check again** button (`plan.login.checkAgain`).

## 4. Auth and session

- **Mechanism.** Email and password only, via **better-auth 1.7.7** `emailAndPassword` (`packages/web/src/Auth.ts`).
  There are no passkeys, no magic links, no email verification, and no password reset or "forgot password" flow (no
  matches for forgot/reset/verification/passkey/magic in `packages/*/src`).
- **Two separate better-auth instances.** The organizer one uses the default tables and the
  `__Host-wedding.session_token` cookie. The operator one (`packages/web/src/OperatorAuth.ts`) uses tables renamed to
  `operator`, `operator_session`, `operator_account`, `operator_verification`, the cookie `__Host-operator`, and no
  plugins. Its routes are never mounted; server-rendered forms call its handler directly. Operator sign-up exists only
  on a setup page shown while the `operator` table is empty, under a semaphore so two concurrent setup posts cannot
  both create an account.
- **Cookies.** The `__Host-` prefix with `Secure`, `Path=/` and no `Domain` (ADR 0060), so a parent-domain cookie
  cannot be planted. `useSecureCookies: false` avoids better-auth's double prefix.
- **Session lifetime.** Not configured, so it runs on better-auth defaults (inferred: 7-day expiry, refreshed daily).
  Secret `BETTER_AUTH_SECRET` has no default, so a deploy without it fails to boot.
- **Authorization per call.** The session gives identity only. Grants are read from the DB on every call, so
  revoking takes effect immediately (ADR 0004).
- **The same better-auth instance is also the OAuth/MCP authorization server** (`@better-auth/mcp`, CIMD,
  `jwt()` plugin). Agents act as the organizer. Not part of onboarding, but the login page also shows the MCP
  endpoint.

## 5. Resend, revoke, expiry, edge cases

### Appointer-side UI

- **Operator.** A server-rendered HTML page at `/operator/wedding` (`packages/web/src/OperatorWedding.ts`,
  `OperatorOwners.ts`). It has an "Appoint an owner" form (one Email field). After appointing it shows a "Pending
  appointment" block with Email, Sent, Expires, the link in a readonly `<input type=url>`, **Copy link** (the only
  inline script on operator pages, `navigator.clipboard.writeText`), and **Cancel** (a POST form).
- **Owner.** The Helpers screen `/plan/grants`. The second owner is appointed from the Owners section ("Appoint as
  owner", "Their email"). A helper is added by typing an email, ticking View/Edit boxes per resource in a permission
  matrix, then **Send link** or **Discard** (`plan.grants.unsent`). Pending rows are titled "Waiting to join" and show
  email, "Sent {date}", "Expires {date}", the helper's carried permissions, the full link (`break-all` so phones do not
  scroll sideways), **Copy link** (toast "The link was copied.") and **Cancel**
  (`packages/plan/src/routes/GrantPending.tsx`, `packages/plan/src/grantAppointments.ts`). `appointments_read`
  is the only read that carries tokens. The change feed carries appointments without the token, so a change event
  re-runs that read.

### Resend

There is no resend button. **Appointing the same address again is the resend.** `replacedBy` deletes the old
appointment (`spent_link: withdrawn`), mints a new token, and emails the new link. The UI says so above the form
while a link waits: "Appointing the same address again sends a new link, and the old one stops working. To appoint
someone else, cancel the pending appointment first." (`operator.owner.again`, `plan.grants.appoint.again`).

The remark under every pending link tells the appointer about the out-of-band fallback: "This link is emailed to the
address above. If it does not arrive, copy it and send it another way. Whoever follows it and signs up becomes the
owner. It works once, for seven days."

### Revoke

**Cancel** deletes the row, writes `spent_link(withdrawn)`, frees the seat, and writes an Activity row. Either owner
cancels any appointment. The Operator cancels only first-owner appointments: its own, or any owner appointment while
the wedding has no owner (`mayCancel` in `Appointments.ts`). Cancelling an already-used or cancelled appointment
refuses `gone` ("This appointment was already used or cancelled.").

### Expiry

Computed, never written. An expired appointment stays visible as "Expired {date}" with no link and no Copy link,
holds no seat, and shows the remark "This link has expired. It no longer works or holds the seat. Appoint again to send
a new link, or cancel it." For an owner seat, the next appointment to *any* address sweeps expired owner appointments.
An expired helper appointment to another address stays until it is cancelled or replaced.

### Edge cases handled

- **Address typo.** Harmless. The typo'd recipient never gets the link. Cancel and re-appoint.
- **Signing up with a different email than invited.** Allowed by design.
- **Invitee already has an account.** Use **Sign in instead**, or the confirm screen if already signed in.
- **Invitee already an owner or holding every grant.** Refused `held`, and the link stays live for someone else.
- **Race for the last owner seat.** Seat checks and writes run in one SQL transaction (`appointThrough`,
  `takeAppointment`). Refused `full`, and the account stays.
- **Two owners finishing first run together.** A version-conflict message, with nothing to resolve.
- **Wedding archived.** No email is sent and every link answers `archived`. The seven days keep counting.
- **Bad host.** Answers `unknown`, which hides which weddings exist.
- **Agents (MCP).** They cannot appoint or cancel. `grants:*` is closed to MCP.
- **Removed owner re-appointed.** Takes the seat through the signed-in confirm path.
- **A refusal after sign-up** never rolls back the new account (ADR 0085, `LinkRefusal` doc comment).

## 6. Testing and offline mail

- **The `Mailer` is an Effect `Context.Service` with three layers** (`packages/domain/src/Mailer.ts`):
  - `Mailer.layer`. Live when `WEDDING_MAIL_API_KEY` is set. Otherwise an **unconfigured** variant whose sends fail
    `MailerUnconfigured`, which the appointment flow logs and ignores. Local dev and the `drive` harness run this way
    on purpose. The repo's session-start hook and run-the-app skill say never to start dev with the key set.
  - `Mailer.layerTest(script, recorder, {configured?})`. A **scripted, recording fake**. `script.send` is a
    queue of answers (`ok`, `refused{text}`, `unreachable`), and every message is pushed to `recorder.messages` with
    subject, HTML and text. Tests assert on the recorded email and can force a provider failure on the Nth send.
- **Getting the link without email in dev.** Read it from the Operator page's readonly field or the owner's
  pending row (Copy link). `.claude/skills/run-the-app/features/open-the-wedding.md` scripts the whole flow with the
  repo's `drive` CLI (`bun run drive up --scenario cold`, then fill/click steps, read the token out of the captured
  aria tree).
- **Time.** `TestClock.adjust(Duration.days(8))` from `effect/testing` proves expiry with no wall-clock waits.
- **Tests** (Vitest + `@effect/vitest`, full HTTP handler in-process with throwaway SQLite):
  - `test/appointmentEmail.test.ts`. The email goes to the right address with the right subject and link,
    re-appointing re-sends a new link and withdraws the old, archived sends nothing, and an unreachable provider
    keeps the appointment.
  - `test/appointmentLink.test.ts`. The end-to-end Operator path, from sign-up to owner.
  - `test/appointmentPaths.test.ts`. Replace, expiry, archived, `held`, `full`, withdrawn and used.
  - `test/ownerAppointment.test.ts`, `test/helperAppointment.test.ts`. The owner-side paths.
  - `packages/domain/src/appointmentEmailRender.test.ts`. Pure render.
  - `packages/plan/src/transport/GrantsAppoint.test.tsx`, `GrantsHelperLinks.test.tsx`. The React screens over
    fixture transport.
  - `packages/ui/src/messages/catalogOperator.test.ts`. Copy checks, for example no em dashes in strings.
- **Real-client proof scripts** (run on the server, where the key lives):
  - `bun run mail:proof <addr>` (`scripts/mail-proof.ts`). One plain message to check the key, domain and DNS.
  - `bun run mail:appointment <addr> [--colours=a,b]` (`scripts/appointment-proof.ts`). Renders the real appointment
    email through the real `renderAppointmentEmail` + `appointmentLink`, with a **dead token** (`"0".repeat(64)`), so
    clicking it shows the `unknown` refusal and never grants a seat. The recipient id `appointment-proof` matches no
    row, so a bounce touches nothing. The run-the-app doc then says to read it in Gmail, Outlook.com and Apple Mail
    via the sanitized DOM.
  - `bun run mail:templates` (`scripts/template-proof.ts`). The same idea for the other templates.

## 7. Tech stack for this slice

- **Runtime and backbone.** Bun 1.4, TypeScript 7, Effect v4 everywhere (services, `Schema`, `HttpApi`, SQL, `Config`,
  `TestClock`). Decided in `docs/adr/0007-the-tech-stack.md`.
- **Data.** SQLite (WAL) through `effect/sql` with hand-written SQL. The tables are `appointment` and `spent_link`.
- **Auth.** better-auth 1.7.7, email+password, two instances (organizer, operator). `@better-auth/mcp` and
  `@better-auth/cimd` for agent OAuth.
- **Mail.** SMTP2GO HTTP API, hand-rolled client, with a hand-rolled table-layout HTML email (`EmailFrame.ts`) and
  design tokens from `@wedding/theme`.
- **Organizer UI.** React 19, Vite 8, TanStack Router, Effect atoms (`@effect/atom-react`) for state, Tailwind 4,
  shadcn on Base UI. Copy comes from a typed message catalog (`@wedding/ui/messages`).
- **Operator UI.** Server-rendered HTML strings with plain `<form method=post>`, almost no JS.

### Key files

Domain rules:
- `docs/adr/0085-an-appointment-is-taken-through-its-link.md`
- `GLOSSARY.md` (Pending appointment, Appointment link)
- `packages/api/src/Appointment.ts`. Types, `appointmentDays`, `appointmentLink`, `LinkRefusalReason`.
- `packages/domain/src/Appointments.ts`. Make, replace, cancel, token draw.
- `packages/domain/src/AppointmentSeats.ts`. Seat math and `replacedBy`.
- `packages/domain/src/AppointmentTaking.ts`. `openLink`, `takeAppointment`.
- `packages/domain/src/AppointmentQueries.ts`, `AppointmentMigration.ts`. SQL and schema.
- `packages/domain/src/AppointmentLink.ts`. The read and take entry points.
- `packages/domain/src/OperatorActs.ts`, `packages/domain/src/grantBindings.ts`. The appoint acts and the email
  trigger.

Mail:
- `packages/domain/src/AppointmentEmail.ts`, `AppointmentEmailRender.ts`, `EmailFrame.ts`
- `packages/domain/src/Mailer.ts`, `MailHeaders.ts`, `MailDoor.ts`, `packages/web/src/MailHooks.ts`
- `packages/ui/src/messages/catalogOrganizer.ts`. All copy (`email.appointment.*`, `plan.appointment.*`,
  `plan.grants.*`, `operator.*`).

Web and auth:
- `packages/web/src/AppointmentApi.ts`, `Auth.ts`, `OperatorAuth.ts`, `Authorization.ts`, `AuthRoutes.ts`
- `packages/web/src/OperatorWedding.ts`, `OperatorOwners.ts`

Organizer client:
- `packages/plan/src/appointment.ts`. State machine and the sign-up/sign-in/take effects.
- `packages/plan/src/routes/appointment.tsx`, `AppointmentCard.tsx`, `AppointmentOpen.tsx`, `AppointmentSignUp.tsx`,
  `AppointmentSignIn.tsx`, `AppointmentConfirm.tsx`, `AppointmentUntaken.tsx`
- `packages/plan/src/login.ts`, `resume.ts`, `routes/LoginResume.tsx`
- `packages/plan/src/firstRun.ts`, `routes/FirstRun.tsx`, `FirstRunActions.tsx`, `FirstRunLeave.tsx`
- `packages/plan/src/grantAppointments.ts`, `routes/GrantPending.tsx`

Ops and test:
- `deploy/README.md` (mail env)
- `scripts/appointment-proof.ts`, `scripts/mail-proof.ts`
- `.claude/skills/run-the-app/features/open-the-wedding.md`
- the `test/appointment*.test.ts` files above

## 8. Polished and worth copying

1. **The link grants, the address does not** (ADR 0085). This closes the hole "open sign-up + match by email =
   whoever registers the address gets the seat". The ADR records the two rejected alternatives and why. This is the
   strongest idea in the slice.
2. **`spent_link` tombstones mean a refusal names its reason.** Each of the seven reasons has an actionable sentence
   that says what happened and who to ask. `unknown` is returned for a bad host too, so nothing leaks.
3. **Refusals never consume the link and never undo sign-up.** Someone who already has access cannot burn a link
   meant for someone else.
4. **Re-appoint is resend, and it rotates the token.** There is only ever one live link per seat or address. The UI
   explains this in one sentence above the form.
5. **Copy link is always there, and the email is not the only channel.** Mail failure never blocks or rolls back.
   The full URL is visible and selectable (readonly input, `break-all`), for when the clipboard API is refused.
6. **The invitee page covers three entry states with one card**: signed out new (sign up, email prefilled and
   editable), signed out existing (**Sign in instead** toggle), and signed in (one-button confirm). Form values
   survive a refused submit.
7. **Email rendering is a pure function over a shared frame** that encodes the Outlook and Gmail rules once. It has a
   plain-text part and no images, is themed with the wedding's colours, puts the raw URL in the footer, and says "If
   you did not expect this email, you can ignore it. Nothing changes until the link is used."
8. **The Mailer as a swappable service** with live, unconfigured and scripted-fake layers. The unconfigured mode makes
   dev mail-safe by default. The scripted fake lets one test cover "the 4th send fails".
9. **The real-render proof script with a dead token**, for checking deliverability in real clients without minting
   a live credential.
10. **`X-Wedding` / `X-Wedding-Recipient` headers**, so bounce webhooks route without an address lookup.
11. **Seat math is pure functions** (`operatorMayAppoint`, `ownerMayAppoint`, `replacedBy`, `isLive`) applied
    inside one transaction. Expiry is computed, never written by a cron.
12. **First run after joining** is tiny: two prefilled fields, Confirm or Keep defaults, stored per wedding so only
    one owner sees it.
13. **Every act writes an Activity row** ("Appointed an owner", "Joined through an appointment link", "Cancelled
    an appointment"), and the Operator is never named.
14. **Copy discipline.** Every string lives in a catalog with comments, and tests assert on copy (for example no em
    dashes).

## 9. Rough edges

1. **Tokens are stored in plaintext** (`appointment.token`, `spent_link.token`, unique index). Anyone with a DB read
   (backup, operator) can take any live link. SlipStream should store a hash (SHA-256 of a 256-bit token is enough)
   and show the raw link only once, or keep it re-showable deliberately as wedding-planner does for Copy link. That is
   a real tradeoff: re-showable Copy link *requires* storing the recoverable token.
2. **No password reset at all.** A forgotten password has no in-app recovery. I found no forgot/reset code or copy.
   The fallback is presumably the operator or direct DB access (inferred).
3. **Password-only auth, no email verification.** better-auth sign-up at `/api/auth/sign-up/email` is open to the
   internet, and strangers get a "not part of this wedding yet" screen. This is safe because of point 8.1, but it
   lets anyone create accounts. No passkey or magic-link option exists.
4. **The email send result is thrown away.** `sendAppointmentEmail` returns `sent|failed|archived|gone`, but both
   call sites ignore it, so the appointer never sees "the email failed". Bounces for appointment emails are only
   logged. The design leans on the always-visible Copy link and its remark ("If it does not arrive...") instead.
5. **The send is synchronous in the request**, with up to two retries on exponential backoff from 1 s. A down
   provider makes the Appoint click slow (inferred: about 3 s or more). There is no outbox for this email, unlike the
   guest communications.
6. **Orphan accounts.** A sign-up that is then refused (`full`, `held`, a race) leaves an account with no
   membership. This is intended, but nothing cleans it up.
7. **No loading state.** The link page renders blank until both reads answer.
8. **The expiry shown is a date, but the real expiry is an instant.** The email says "until {date}" in the wedding
   timezone, while the link dies at `made_at + 7d`, which may be mid-day on that date (inferred from
   `todayIn(timezone, expires_at)`).
9. **There is no explicit "Resend" affordance.** Resend is discoverable only through the remark text above the form.
10. **The email's CTA is a text link, not a button** (`link(...)` row). That is fine for deliverability but less
    prominent.
11. **Dev cookie gotcha.** The Operator door answers only on `127.0.0.1` and the wedding only on `localhost`, so a
    copied link must be opened on `localhost` (`open-the-wedding.md` Gotchas).

## 10. Mapping notes for SlipStream (observations, not decisions)

- wedding-planner's "Operator appoints first owner" corresponds to SlipStream's admin inviting a portal user. Its
  "owner appoints helper with permission set" corresponds to an admin inviting a requester with a quota or
  permission set. The pending-appointment and spent-link model ports directly.
- The parts that carry over unchanged: link-grants-not-address, a 256-bit token, 7-day TTL computed on read,
  single use, re-appoint-rotates, tombstone reasons, Copy link fallback, an unconfigured mailer by default in dev, and
  a scripted fake mailer in tests.
- The parts SlipStream would need to decide fresh: auth mechanism (wedding-planner is password-only with no reset),
  token hashing versus a re-showable link, mail transport (SMTP2GO HTTP is specific to a hosted deploy; a
  self-hosted media manager likely needs generic SMTP), and whether send failure surfaces to the admin.
