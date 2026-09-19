# 17: Calendar and Manual Import pages

Spec: `docs/admin-native-overhaul-spec.md` (stories 76, 79)

**What to build:** Calendar's month and list toggle is a segmented control; the list view is grouped rows by day so upcoming releases read well on a phone, and the month view fits the phone width without horizontal scroll. Manual Import's file browser renders as a pushed list of rows with folder chevrons: entering a folder pushes a new list with the parent folder as the back label, files show their detected match and status, and the import action works from the row list. Both pages adopt `Screen`.

**Blocked by:** 04 (Push navigation), 06 (Activity queue rows, segmented filter and the action presenter)

**Status:** ready-for-agent

- [ ] Both projects: Calendar shows the segmented toggle; the list view groups developer-mode releases by day; the month view shows the same items on their dates without horizontal overflow on `phone`
- [ ] Phone project: entering a folder in Manual Import pushes a new list whose back label is the parent folder's name; back returns to the parent at the same scroll position
- [ ] Both projects: selecting a file and importing completes as today and the file leaves the list
- [ ] Both projects: rows in the browser are at least 44 px and folders show a chevron
