# 08: Library actions: sort and view presenter, edit mode toolbar, add flow presentation

Spec: `docs/admin-native-overhaul-spec.md` (stories 36–37, 39)

**What to build:** The Library's secondary options (sort, sort direction, view mode on wide, poster size, table columns on wide) sit behind a single trailing action in the top bar that opens an action sheet on phones and a menu on wide screens. Edit mode keeps working for bulk monitor, quality profile and delete: checkboxes overlay poster cells and a bottom toolbar sits above the tab bar on phones (and at the bottom of the content column on wide screens). The Add action opens the existing add flow (search, configure, add) as a pushed full screen on phones and a dialog on wide screens.

**Blocked by:** 07 (Library tab: module segmented control, chip filters and PosterCell)

**Status:** ready-for-agent

- [ ] Both projects: the top bar shows one trailing options action; on `phone` it opens an action sheet, on `wide` a menu; choosing a sort reorders the grid and the choice persists across reload
- [ ] Wide project: view mode, poster size and column options are present in the menu; phone project: view mode and column options are absent
- [ ] Both projects: entering edit mode overlays checkboxes on cells; selecting two items enables the toolbar; a bulk monitor change updates both items' status; leaving edit mode hides the toolbar
- [ ] Phone project: the edit toolbar is fully visible above the tab bar and not obscured by safe-area insets
- [ ] Both projects: Add opens the add flow; searching the developer-mode metadata provider, configuring and adding a title results in it appearing in the grid; on `phone` the flow is a pushed screen with a back button, on `wide` a dialog
