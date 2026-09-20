# 10: Search tab and wide header search

Spec: `docs/admin-native-overhaul-spec.md` (stories 58–62)

**What to build:** The Search tab shows a 16 px input with a magnifier and a clear button that never zooms the page. Library results are rows with poster thumbnail, title, year, media type in its colour and a status dot, opening the detail on tap. Before the admin types, an empty state says how many titles are searchable. The external metadata search (adding new media) is reachable from the Search tab and hands off to the add flow. On a wide screen the header search bar is retained, styled with the same field, and results open the same way from any page.

**Blocked by:** 07 (Library tab: module segmented control, chip filters and PosterCell)

**Status:** done

- [x] Phone project: focusing the Search tab input leaves the visual viewport scale unchanged; the clear button empties the field and restores the empty state
- [x] Both projects: the empty state names the number of searchable titles from developer-mode data
- [x] Both projects: typing a known title shows a row with thumbnail, title, year, media type and status dot; activating it opens the detail
- [x] Both projects: the "add new" entry from Search opens the external metadata search and continues into the add flow
- [x] Wide project: the header field is present on the Dashboard, uses the design-system field styling and produces the same results view

## Comments

The Search tab is a `Screen` on both shells: `search` joins the screen-filling panes in
`tab-panes.tsx` and `push-routes.ts`, so the phone pane and the wide content area let the page own
its scrolling and its large title. `isScreenFillPath` now tests a set of pane ids instead of a
chain of comparisons — same paths, one place to add the next one.

The field is the `SearchField` the wide header already used, so both shells show the same 16 px
input with a magnifier and a clear button, and the page's state is seeded from the `q` search param
the header writes. That is how the header keeps "the same results view from any page": it navigates
to `/search?q=…` and the page picks the query up. There was no mobile fallback left in the header to
remove — the header only renders in the wide shell — so the removal on the spec's list had already
happened with the two-shell layout.

Library search moved into the browser: the page reads the whole library once with `useMovies()` /
`useSeries()` (the same queries the Library tab has already cached) and filters by title, which is
also where the empty state's count of searchable titles comes from. The server-side `search` filter
is no longer used here.

"Add new" is a row while the library has matches and the external metadata search opens in place
under an "Add new" group; with no library match the external search still runs on its own, as
before. Each external result is a row that navigates to the module's add route with its `tmdbId`,
so the add flow (ticket 08) is unchanged. The card-based `ExternalSearchSection` is gone;
`SearchResultsSection`, `ExpandableMediaGrid` and `ExternalMediaCard` stay because the portal's
search still uses them.

E2E: `e2e/search.spec.ts` is five tests — the phone zoom-and-clear check, the count in the empty
state (read back from `/movies` and `/series`), a library row's status, year, media type and detail
push, the "Add new" hand-off into `/movies/add?tmdbId=…`, and the wide header field searching from
the Dashboard. The page field is scoped by the `Screen` region and the header field by the app
banner, because on wide both are searchboxes named "Search". `library.spec.ts`'s existing
`/search?q=matrix` check still passes: the result is still a link to the detail carrying a status
mark, it is a row now rather than a poster cell.
