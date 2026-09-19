# 03: Two-shell layout: phone tab bar, More index, wide sidebar, Screen

Spec: `docs/admin-native-overhaul-spec.md` (stories 1–3, 6–19, 21–22, 61, 97–98, 100–102)

**What to build:** The admin app renders inside one `ShellLayout` chosen by the viewport hook. On a phone the admin sees a translucent five-tab bottom bar (Dashboard, Library, Activity, Search, More) that respects the home indicator, with the Activity tab carrying the active download count. On a wide screen the admin sees a translucent sidebar with the same groups the phone's More tab lists, collapsible to icons with the choice remembered. The same URL opens correctly in either shell. Every existing page keeps working inside the new shell unchanged, and the Dashboard becomes the first page to use the new `Screen` pattern: a large title that scrolls with content and collapses into a compact centred title in a translucent bar with a hairline that appears only once content is beneath it.

Concretely:

- `ShellLayout` replaces the body of the current root layout. Auth gating, WebSocket lifecycle, theme and dev-mode effects are unchanged. The layout uses dynamic viewport height so bottom-pinned bars stay above browser chrome and the keyboard.
- Phone: tab bar mapping Dashboard to `/`, Library to the last-used module's list route (a new UI-store preference, defaulting to the first enabled module), Activity to `/downloads`, Search to `/search`, More to a new `/more` route. Tab switches are instant, tab screens stay mounted and hidden when inactive so scroll position survives. The Activity badge reads the same queue query the sidebar's downloads link uses today. The header search bar is not rendered on phones.
- `/more` renders the grouped index: Calendar, Requests, Missing (with separate movie and episode counts in their media colours), Manual Import, History; Settings (Media, Download Pipeline, General); System; Developer Tools; Log out and Restart. Rows have icon tiles and chevrons. The existing log-out and restart confirmations keep working from here for now (they move to the action-sheet presenter in a later ticket).
- Wide: translucent sidebar with the same grouping as `/more`, the collapse-to-icons preference kept in the UI store, route changes instant with no transition, the header search field retained and restyled with the design-system field, Developer Tools kept in the header. The existing `DownloadsProgressOverlay` stays on the wide sidebar's downloads row and is not rendered on phones.
- `Screen`: scroll container plus top bar with props for title, back, trailing actions, large-title on/off and transparent-until offset; owns the collapse behaviour. Page actions live in the top bar's trailing slot on phones and beside the title on wide screens. Content uses a 16 px gutter on phones and a readable maximum width on wide screens. The Dashboard adopts `Screen`; other pages adopt it in their own tickets.
- The current tab and sidebar item are marked `aria-current`; every icon-only control in the shell has an accessible name; focus rings are visible for keyboard users. Toasts render above the tab bar on phones and bottom-right on wide screens.
- Skeleton for the tab bar badge and More counts while queries load.

**Blocked by:** 01 (Token layer, platform baseline and unified status rendering), 02 (Playwright end-to-end harness)

**Status:** ready-for-agent

- [ ] Phone project: the five tabs are visible with accessible names, tapping each opens its destination, the active tab is `aria-current`, and the Activity tab shows the developer-mode queue count
- [ ] Phone project: scrolling a tab, switching away and back restores the scroll position; the tab bar remains visible with the on-screen keyboard open and above simulated safe-area insets
- [ ] Phone project: `/more` lists every destination above as grouped rows; Missing shows movie and episode counts; each row navigates to its route
- [ ] Wide project: sidebar shows the same groups, collapses to icons, the choice survives reload, the header search field is present and Developer Tools are reachable from the header; the downloads progress overlay renders only here
- [ ] Both projects: opening any existing route directly by URL renders it inside the correct shell; the portal under `/requests` is unaffected
- [ ] Both projects: on the Dashboard the large title is visible at rest; after scrolling, the compact title and bar hairline appear and the large title is out of view
- [ ] Wide project: keyboard tab traversal shows a visible focus ring on shell controls; every icon-only control has a name
- [ ] Toasts appear above the tab bar on `phone` and bottom-right on `wide`
- [ ] Reduced-motion run of the shell tests passes
