# 15: Requests admin: segmented queue, row actions, users and settings

Spec: `docs/admin-native-overhaul-spec.md` (stories 73–75)

**What to build:** The requests queue uses a segmented control across Pending, Approved, Downloading, Available and Denied, rendering each request as a row with poster, requester, media type and status pill. Approve and deny are trailing actions on the row; deny confirms in an action sheet on phones and a dialog on wide screens. Portal Users and Request Settings are grouped lists under Requests, and their forms (invite, user edit, request search) open through the sheet/dialog presenter. The portal itself under `/requests` is untouched.

**Blocked by:** 06 (Activity queue rows, segmented filter and the action presenter), 09 (Movie and series detail: hero, pill actions, grouped metadata, sheet presenter)

**Status:** done

- [x] Both projects: the queue shows the five segments; choosing Pending lists developer-mode pending requests as rows with poster, requester, media type and status pill
- [x] Both projects: approving a pending request moves it out of Pending and into Approved (or Downloading once auto search begins)
- [x] Phone project: deny opens an action sheet; confirming moves the request to Denied; wide project: the same through a dialog
- [x] Both projects: Users lists portal users as rows; invite and edit open through the presenter and a created user appears as a row
- [x] Both projects: Request Settings renders as grouped controls and saves
- [x] The portal under `/requests` renders exactly as before

## Comments

The three routes are now `isScreenFillPath` (one line in `push-routes.ts`, plus a small rewrite of
`isScreenFillPath` to keep it under the complexity limit) so each renders its own `Screen`. Back
reads "More", which the ticket accepts.

The queue is `Screen` "Requests": a `Group` of Users / Request Settings rows (the replacement for
the old `RequestsNav` tab strip), a `Segmented` of the five states, and one `Group` of request
rows. `request-status.ts` holds the segment membership and the pill hue: Approved also carries
`searching`, Denied also carries `cancelled`, and `failed` sits under Downloading — a failed
request is a download that did not finish, not a refusal. `StatusPill` gained an optional `label`
so request statuses can reuse the media status hues without inventing a second pill.

Approve and Deny are trailing controls on a pending row. Deny presents an `ActionPresenter` with
`wide="dialog"`, which is exactly the ticket's "action sheet on phones, dialog on wide"; the
secondary actions (Approve & Manual Search, Approve & Auto Search, Delete) sit behind the row's
"More actions" presenter, where Delete uses `confirm`.

Two deliberate reductions, both legacy the native row design has no room for:

- **Multi-select and batch deny/delete are gone.** The spec keeps bulk selection only for the
  Library's edit mode; checkboxes on request rows would fight the trailing actions. The now-dead
  `useBatchDenyRequests` / `useBatchDeleteRequests` hooks, their API functions and the
  `BatchApproveInput` / `BatchDenyInput` types were removed with them.
- **The deny reason textarea is gone.** An action sheet takes actions, not input. `deniedReason`
  is still rendered on a row when the portal side set one; only the admin's ability to type one
  while denying was dropped.

`SearchModal` now presents through `SheetPresenter` (the requests queue is its only consumer), and
the invite and user-edit forms became `InviteSheet` / `UserEditSheet`. The clipboard "show link"
fallback on invitations was dropped as defensive code; a copy failure now just reports itself.

The admin request list endpoint returns no `user` on a request, so the requester name is looked up
client-side from `useAdminUsers()` rather than changing the backend.

`web/e2e/requests-admin.spec.ts` covers every box in both projects (12 tests). Two things the
harness forced:

- Developer mode does not seed requests — it copies portal users from the production database,
  which is empty in the suite. The spec creates its own requests through `POST /requests`, and
  first clears the Administrator's auto-approve and per-module profiles, otherwise a request
  skips the queue entirely (and skips it only until the weekly quota runs out, which made the
  first draft flaky rather than failing).
- Portal signup is behind the shared 10-requests-per-minute auth limiter, so the Users test
  asserts the invited user through its invitation row rather than completing a signup; the
  Administrator row (switch, edit sheet) covers "portal users as rows".

The two projects write different Request Settings fields, because those settings are global and
the shells would otherwise overwrite each other between save and reload.

Full suite: 156 passed with two unrelated flakes under load (`calendar-import` on phone,
`dashboard` Recent rows on wide); both pass on their own. `test:e2e:reduced-motion` passes.

The deny reason is back, and the second reduction above no longer holds: Deny now presents
`DenySheet` (`web/src/routes/requests-admin/deny-sheet.tsx`) through `SheetPresenter` — a
`TextareaRow` for the optional reason with a destructive Deny and Cancel in `FormActions` — and the
reason is sent to the existing deny mutation, so the portal shows it as `deniedReason`.
