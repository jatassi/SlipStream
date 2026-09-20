# 14: System screens, session actions and Developer Tools

Spec: `docs/admin-native-overhaul-spec.md` (stories 70–71, 100)

**What to build:** System shows Health issues as rows, scheduled Tasks as rows with last and next run and a running indicator, and Logs and Update as sub-screens, all inside `Screen`. Restart and Log out present as an action sheet on phones and a dialog on wide screens with the same countdown behaviour as today, reachable from More on phones and the sidebar on wide screens. Developer Tools (dev-mode toggle, global loading) are available from More on phones and remain in the header on wide screens.

**Blocked by:** 06 (Activity queue rows, segmented filter and the action presenter), 11 (Settings index and list pages as grouped rows)

**Status:** done

- [x] Both projects: `/system/health` lists developer-mode health issues as rows; `/system/tasks` lists tasks with last and next run; running a task shows the running indicator and updates last run
- [x] Both projects: Logs and Update open as sub-screens with correct back labels
- [x] Phone project: Log out from More opens an action sheet; confirming signs out; Restart opens an action sheet showing the countdown
- [x] Wide project: the same actions confirm in dialogs with the countdown
- [x] Phone project: Developer Tools in More toggles developer mode and the global loading state; wide project: the header controls do the same

## Comments

`/system/health` is the System index, not a peer of the other three. The tabbed `SystemNav` is gone:
the screen is titled "System", carries a `Group` of rows for Scheduled Tasks, Logs and Update, and
then the health groups. Tasks, Logs and Update are the sub-screens and all three read "System" as
their back label; `/system/health` reads "More", the default. The Scheduled Tasks row shows an
"n running" indicator, as the Native prototype's More screen does.

Health is one `Group` per category, header = category name, `action` = the existing "Test All"
(`shell.spec.ts` still drives it for the toast placement check). Rows carry a status `IconTile`
(emerald / amber / destructive), the message and relative age as subtitle, and a 44 px per-item test
button; storage has neither, as before. The Prowlarr tree is flattened: in Prowlarr mode the Prowlarr
item is simply the first row of the Indexers group. A grouped list has no indentation to spend on a
tree, and the parent/child relationship is already clear from the ordering.

The task rows dropped the frequency column. On a 390 px phone the subtitle fits "Last … · Next …" and
little else, and the prototype shows exactly that, so `cronToPlainEnglish` went with the table it
served. `useScheduledTasks` now polls every 2 s while any task is running (60 s otherwise) so the
running indicator reflects reality instead of waiting out a minute.

Session actions go through `ActionPresenter`, which gained three things to make that possible without
a second hand-rolled surface: `wide="dialog"` (the wide surface is the confirmation itself, not a
one-item menu), `ActionItem.keepOpen` (the restart surface stays up so its label can count down, the
way the old dialog's button did), and `locked` (Cancel disabled and dismissal refused while the
countdown runs). `ActionItem.disabled` came along for the pending state. `sidebar-dialogs.tsx` is
deleted; More and the sidebar both render `SessionActionPresenters`.

Log out is styled destructive rather than the old amber: an action sheet has one emphatic tone and
iOS spends it on sign-out. The row on More keeps its amber `warning` tone.

The Developer Tools e2e checks toggle Force Loading for real but only assert that the `Developer mode`
switch is present, checked and enabled. Toggling developer mode off swaps the backend database, which
would break every test running in parallel against the same server; that is not a check worth buying
at that price.

Logs is a fixed-height (`60vh`, min 16 rem) `role="log"` panel inside the `Screen` scroller rather
than a `flex-1` fill, because `Screen` owns the scrolling. Its toolbar buttons became 44 px
icon-buttons with real accessible names. Update keeps its state machine unchanged; only its chrome
moved into `Screen`, with the developer-mode debug button in the `trailing` slot.

E2E: `e2e/system.spec.ts`, 17 checks across both projects. The running indicator is served from a
route stub — a real task finishes long before a poll can catch it — while "updates last run" runs
History Cleanup for real and waits for "Last just now". The restart check stubs
`POST /api/v1/system/restart`, cancels once, confirms once, asserts `Restarting (Ns)` and then
navigates away before the reload. Log out signs out for real; storage state is re-seeded per context
from `e2e/.auth/admin.json`, so no other test is affected and no serial mode is needed.
