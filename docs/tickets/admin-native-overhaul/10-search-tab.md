# 10: Search tab and wide header search

Spec: `docs/admin-native-overhaul-spec.md` (stories 58–62)

**What to build:** The Search tab shows a 16 px input with a magnifier and a clear button that never zooms the page. Library results are rows with poster thumbnail, title, year, media type in its colour and a status dot, opening the detail on tap. Before the admin types, an empty state says how many titles are searchable. The external metadata search (adding new media) is reachable from the Search tab and hands off to the add flow. On a wide screen the header search bar is retained, styled with the same field, and results open the same way from any page.

**Blocked by:** 07 (Library tab: module segmented control, chip filters and PosterCell)

**Status:** ready-for-agent

- [ ] Phone project: focusing the Search tab input leaves the visual viewport scale unchanged; the clear button empties the field and restores the empty state
- [ ] Both projects: the empty state names the number of searchable titles from developer-mode data
- [ ] Both projects: typing a known title shows a row with thumbnail, title, year, media type and status dot; activating it opens the detail
- [ ] Both projects: the "add new" entry from Search opens the external metadata search and continues into the add flow
- [ ] Wide project: the header field is present on the Dashboard, uses the design-system field styling and produces the same results view
