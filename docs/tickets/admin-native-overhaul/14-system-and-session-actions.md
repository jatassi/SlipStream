# 14: System screens, session actions and Developer Tools

Spec: `docs/admin-native-overhaul-spec.md` (stories 70–71, 100)

**What to build:** System shows Health issues as rows, scheduled Tasks as rows with last and next run and a running indicator, and Logs and Update as sub-screens, all inside `Screen`. Restart and Log out present as an action sheet on phones and a dialog on wide screens with the same countdown behaviour as today, reachable from More on phones and the sidebar on wide screens. Developer Tools (dev-mode toggle, global loading) are available from More on phones and remain in the header on wide screens.

**Blocked by:** 06 (Activity queue rows, segmented filter and the action presenter), 11 (Settings index and list pages as grouped rows)

**Status:** ready-for-agent

- [ ] Both projects: `/system/health` lists developer-mode health issues as rows; `/system/tasks` lists tasks with last and next run; running a task shows the running indicator and updates last run
- [ ] Both projects: Logs and Update open as sub-screens with correct back labels
- [ ] Phone project: Log out from More opens an action sheet; confirming signs out; Restart opens an action sheet showing the countdown
- [ ] Wide project: the same actions confirm in dialogs with the countdown
- [ ] Phone project: Developer Tools in More toggles developer mode and the global loading state; wide project: the header controls do the same
