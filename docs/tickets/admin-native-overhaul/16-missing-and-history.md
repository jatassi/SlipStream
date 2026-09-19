# 16: Missing and History pages

Spec: `docs/admin-native-overhaul-spec.md` (stories 77–78)

**What to build:** The Missing page uses a segmented control across enabled modules and a secondary Missing or Upgradable toggle, listing items as rows with a trailing Search action that triggers a search in one tap. History renders as rows with an event tile, title, detail and relative time, grouped by day, matching the Dashboard's Recent section. Both pages adopt `Screen` and open from More on phones and the sidebar on wide screens.

**Blocked by:** 07 (Library tab: module segmented control, chip filters and PosterCell)

**Status:** ready-for-agent

- [ ] Both projects: Missing shows one segment per enabled module and a Missing/Upgradable toggle; switching either changes the visible rows
- [ ] Both projects: activating a row's Search action starts a search against the developer-mode indexer and the row reflects the searching state; activating the row itself opens the detail
- [ ] Both projects: History groups rows by day with an event tile, title, detail and relative time; a filter change updates the list
- [ ] Phone project: the back label from either page reads "More"
