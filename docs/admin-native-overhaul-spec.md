# Admin App Overhaul: Native Design System and Patterns

Status: ready-for-agent

Origin: `/to-spec` synthesis of the mobile design exploration on branch
`cursor/mobile-design-system-prototypes-20af` (PR #17). The Native direction was chosen from
three prototyped shells; see `docs/mobile-design-system.md` for the token layer.

Reference implementation: the Native prototype on branch
`cursor/mobile-design-system-prototypes-20af`, under `web/src/prototypes/mobile/` (harness,
shared primitives and mock data, `variants/native/`). It is kept on that branch for reference
throughout implementation and is the source of truth for the look and behaviour of every
pattern named below. Run it with `cd web && bun run dev` and open
`http://localhost:3000/prototypes/mobile/?v=1`.

## Problem Statement

The SlipStream admin app is a desktop-only React shell: a fixed 256 px sidebar, a 56 px header
with a search bar, and pages that assume a wide viewport. On a phone it is unusable as an app:
the sidebar eats the screen, nothing is reachable by thumb, hover states stick after a tap,
inputs zoom the page, tables overflow, and dialogs are sized for a monitor. Someone checking
whether last night's episode downloaded, pausing a queue item, or approving a request from
their phone has to pinch and scroll through a desktop layout.

Beyond mobile, the app has no shared design language. Each page invents its own header, card
grid, status colour, hover glow and transition. Status is painted six different ways (solid
badges, coloured text, dots, glows), motion is a mix of `transition-all` and 300 ms fades,
and the type scale is whatever Tailwind default a given file reached for. The result works but
does not feel like one product, and every new page adds more variance.

## Solution

Rebuild the admin app's presentation layer on the Native design system that was prototyped
and chosen: a single token layer (motion, type, touch geometry, status hues, materials, safe
areas) and a fixed set of patterns (large collapsing titles, inset grouped lists, segmented
controls, poster cells, pill actions, action sheets, a translucent bottom tab bar on phones,
a translucent sidebar on wide screens).

The app becomes one responsive codebase with two shells:

- **Phone (below 768 px)**: the Native shell. Five-tab bottom bar (Dashboard, Library,
  Activity, Search, More). Detail and settings screens push in from the right with the
  previous screen's name on the back button. Forms and confirmations present as bottom
  sheets and action sheets. Everything reachable by thumb; every control at least 44 px.
- **Wide (768 px and up)**: a translucent sidebar carrying the same grouping the More tab
  uses, the same large titles, grouped lists, segmented controls and cells. No slide
  transitions (keyboard and mouse users navigate hundreds of times a day). Tables remain
  available here.

Functionality is unchanged: same routes, same data hooks, same backend. This is a
presentation-layer overhaul with feature parity, plus the platform baseline that makes a web
app feel installed on a phone.

## User Stories

### Shell and navigation

1. As an admin on my phone, I want a bottom tab bar with Dashboard, Library, Activity, Search and More, so that the five things I do most are one thumb-tap away.
2. As an admin on my phone, I want the Activity tab to show a count of active downloads, so that I can see at a glance whether anything is downloading without opening it.
3. As an admin on my phone, I want switching tabs to be instant with no animation and to preserve each tab's scroll position, so that flipping between tabs never feels slow.
4. As an admin on my phone, I want detail and settings screens to slide in from the right and slide back out when I go back, so that I always know where I am in the hierarchy.
5. As an admin on my phone, I want the back button to name the screen I came from (for example "Library" or "More"), so that I can predict where back takes me.
6. As an admin on my phone, I want the tab bar and every top bar to respect the notch, Dynamic Island and home indicator, so that nothing is hidden behind system UI.
7. As an admin on my phone, I want the tab bar and top bars to be translucent with content scrolling beneath them, so that the app reads as a native surface rather than fixed opaque strips.
8. As an admin on a wide screen, I want a sidebar with the same sections as the phone's More tab (Library modules, Discover, Settings, System, session actions), so that the two shells share one mental model.
9. As an admin on a wide screen, I want the sidebar to collapse to icons and remember that choice, so that I can reclaim width on smaller laptops.
10. As an admin on a wide screen, I want route changes to be instant with no transition, so that mouse and keyboard navigation never waits on motion.
11. As an admin, I want the current tab or sidebar item to be marked as current for assistive technology, so that screen readers announce where I am.
12. As an admin, I want the same URL to work on both shells, so that a link shared from a desktop opens correctly on a phone.
13. As an admin on my phone, I want a More tab that lists Calendar, Requests, Missing, Manual Import, History, Settings (Media, Download Pipeline, General), System, Log out and Restart as grouped rows with icon tiles and chevrons, so that every secondary destination is discoverable in one place.
14. As an admin on my phone, I want the Missing row in More to show separate movie and episode counts in their media colours, so that I know which library needs attention.
15. As an admin on my phone, I want the Downloads progress ring that the sidebar shows today to be represented by the Activity tab badge instead, so that the tab bar stays uncluttered.

### Page structure

16. As an admin, I want every top-level page to open with a large title that scrolls with the content, so that the page name is unmistakable.
17. As an admin, I want that large title to collapse into a compact centred title in a translucent bar once it scrolls out of view, so that I keep context without losing space.
18. As an admin, I want the top bar to gain a hairline only once content is scrolled beneath it, so that a resting page has no unnecessary dividers.
19. As an admin, I want page actions (add, filter, edit) to live in the top bar's trailing slot on phones and beside the title on wide screens, so that primary actions are always in the same place.
20. As an admin, I want detail screens to use a compact bar that is transparent over the hero and becomes translucent once the hero scrolls away, so that artwork is shown edge to edge.
21. As an admin, I want a consistent 16 px screen gutter and inset grouped containers on phones, so that lists align across every page.
22. As an admin on a wide screen, I want content constrained to a readable maximum width and laid out in two columns where the phone stacks, so that wide monitors do not produce metre-long rows.

### Dashboard

23. As an admin, I want the Dashboard to show Health, Storage, Downloading and Recent as grouped sections, so that the first screen answers "is anything wrong, how full is disk, what is downloading, what just happened".
24. As an admin, I want health issues to appear as rows with an amber warning tile and open System when tapped, so that I can act on a problem from the Dashboard.
25. As an admin, I want storage shown as a single bar split by media type with used/total and per-folder figures, so that I see movie versus series consumption at a glance.
26. As an admin, I want the Downloading section to show up to three active items with progress, speed and remaining time and a "See all" action that opens Activity, so that the Dashboard stays short.
27. As an admin, I want the Recent section to show history events with a coloured tile per event type (Imported, Grabbed, Upgraded, Failed) and a relative time, so that I can scan what happened.
28. As an admin, I want tapping a Downloading or Recent row to open that title's detail, so that the Dashboard is a launchpad, not a dead end.

### Library

29. As an admin, I want the Library tab to show a segmented control of enabled modules (Movies, Series, and any future module) with a sliding thumb, so that I switch libraries without leaving the tab.
30. As an admin, I want the Library to remember which module I last viewed, so that returning to the tab shows the same library.
31. As an admin on my phone, I want a three-column poster grid with the title, year, quality and a status dot under each poster, so that posters are unobstructed and status is readable.
32. As an admin on a wide screen, I want the poster grid to grow to as many columns as fit at my chosen poster size, so that large monitors show more of the library.
33. As an admin on a wide screen, I want the table view to remain available with the existing column preferences, so that power users keep their dense view.
34. As an admin on my phone, I want the table view hidden and the grid used exclusively, so that I never see a horizontally overflowing table.
35. As an admin, I want filters (Monitored, Missing, Upgradable, status, and the module's own options) presented as a horizontally scrollable chip row, so that filtering is one tap.
36. As an admin, I want sort and view options behind a single trailing action in the top bar that opens an action sheet on phones and a menu on wide screens, so that the toolbar does not crowd the title.
37. As an admin, I want edit mode (multi-select for bulk monitor, quality, delete) to keep working with checkboxes overlaid on poster cells and a bottom toolbar above the tab bar, so that bulk actions are reachable on a phone.
38. As an admin, I want tapping a poster cell to give immediate press feedback and open the detail screen, so that the grid feels responsive to touch.
39. As an admin, I want the Add action to open the existing add flow (search, configure, add) presented as a full screen on phones and a dialog on wide screens, so that adding media works everywhere.

### Detail screens (movie and series)

40. As an admin, I want a detail screen with a backdrop hero, the poster overlapping the hero, the title, year, runtime or season count, genres, rating and a status pill, so that the top of the screen tells me what and how complete the title is.
41. As an admin, I want three pill actions under the hero (Search, Auto Search, Monitored toggle) that take the media colour when active, so that the primary actions are obvious and thumb-sized.
42. As an admin, I want the Monitored pill to toggle immediately and reflect the new state, so that monitoring is a one-tap decision.
43. As an admin, I want the existing search and monitor control states (default, searching manual or auto, progress with pause, completed flash, error with dismiss) preserved inside the pill row, so that I keep live feedback during a search.
44. As an admin, I want the overview clamped to three lines and expandable by tap, so that long synopses do not push the important sections down.
45. As an admin, I want File, Details and (for series) Seasons presented as grouped lists with trailing values, so that metadata is scannable.
46. As an admin, I want version slots and quality profile shown in the File group, so that I can see which slot each file occupies.
47. As an admin on a series detail, I want seasons as collapsible groups with episode rows showing air date, status dot and a per-episode search action, so that I can act on individual episodes.
48. As an admin, I want Edit and Delete available from a trailing menu in the compact bar, presenting a sheet on phones and a dialog on wide screens, so that destructive actions are deliberate but reachable.

### Activity (downloads queue)

49. As an admin, I want Activity to open with a summary line (n downloading, total speed, n in queue) under the large title, so that the state of the queue is one glance.
50. As an admin, I want each queue row to show the poster thumbnail, title, episode label, release name in monospace, a media-coloured progress line, and percent, speed and time remaining in tabular figures, so that numbers do not jitter as they update.
51. As an admin, I want a 44 px pause or resume control at the trailing edge of each row that toggles without opening anything, so that the most common queue action is one tap.
52. As an admin, I want importing and queued items to show their state in place of the pause control, so that I never see a control that cannot act.
53. As an admin, I want tapping a queue row to open an action sheet with Pause or Resume, Remove from queue, Remove and blocklist release, and Cancel, so that every queue action is available without a long-press or swipe gesture.
54. As an admin, I want destructive actions in the action sheet coloured red and separated from the safe ones, so that I do not remove something by accident.
55. As an admin, I want the filter across All, Movies and Series to be a segmented control, so that it matches the Library.
56. As an admin, I want the download client unreachable banner preserved as an amber row at the top of the queue, so that I know when data is stale.
57. As an admin, I want progress to advance smoothly between updates rather than jumping, so that the queue feels alive.

### Search

58. As an admin on my phone, I want a Search tab with a 16 px input, a magnifier and a clear button, so that the page never zooms when I focus the field.
59. As an admin, I want library results as rows with poster thumbnail, title, year, media type in its colour and a status dot, so that results are scannable and open the detail on tap.
60. As an admin, I want an empty state that tells me how many titles are searchable, so that the tab is not blank before I type.
61. As an admin on a wide screen, I want the header search bar retained and styled with the same field, so that search remains available from every page.
62. As an admin, I want the existing external metadata search (add new media) reachable from the Search tab, so that adding from a phone starts from the same place as searching.

### Settings and System

63. As an admin, I want Media, Download Pipeline and General settings presented as grouped lists of sub-sections with counts (for example "Indexers 4 · 1 warning"), so that I can see the shape of my configuration before opening it.
64. As an admin, I want Root Folders, Quality Profiles, Version Slots, Indexers, Download Clients and Notifications listed as rows with an icon tile, a primary line and a trailing detail, replacing today's card grid, so that lists are dense and consistent.
65. As an admin, I want the Add action for each list to be a trailing plus in the top bar, so that adding is in the same place on every settings page.
66. As an admin, I want create and edit forms (indexer, download client, quality profile, notification, root folder) to open as a bottom sheet on phones and a dialog on wide screens, so that forms are usable on a phone.
67. As an admin, I want every form field to be at least 44 px tall with a 16 px input font, so that fields are tappable and never trigger zoom.
68. As an admin, I want toggles rendered as switches in rows with the label leading and the switch trailing, so that settings screens read like system settings.
69. As an admin, I want Import & Naming, Auto Search, RSS Sync, Server and Authentication settings laid out as grouped lists of controls with section footers for explanatory text, so that long-form settings are navigable.
70. As an admin, I want System to show Health issues as rows, scheduled Tasks with last and next run and a running indicator, Logs and Update as sub-screens, so that operations are grouped in one place.
71. As an admin, I want the Restart and Log out actions presented as an action sheet on phones and a dialog on wide screens with the same countdown behaviour as today, so that dangerous actions are confirmed the same way everywhere.
72. As an admin, I want the Migrate from *arr flow to remain accessible as a settings sub-screen, so that import is not lost in the overhaul.

### Requests admin, Calendar, Missing, History, Manual Import

73. As an admin, I want the Requests queue to use a segmented control across Pending, Approved, Downloading, Available and Denied and to render each request as a row with poster, requester, media type and status pill, so that approvals work from a phone.
74. As an admin, I want approve and deny to be trailing actions on the row with deny confirming in an action sheet, so that triaging requests is fast but safe.
75. As an admin, I want portal Users and Request Settings as grouped lists under Requests, so that the requests area matches the rest of settings.
76. As an admin, I want Calendar's month and list toggle as a segmented control and the list view as grouped rows by day, so that upcoming releases are readable on a phone.
77. As an admin, I want the Missing page to use a segmented control across enabled modules and a secondary Missing or Upgradable toggle, listing items as rows with a trailing Search action, so that I can trigger a search for a missing item in one tap.
78. As an admin, I want History as rows with event tile, title, detail and relative time, grouped by day, so that history matches the Dashboard's Recent section.
79. As an admin, I want Manual Import's file browser to render as a pushed list of rows with folder chevrons, so that browsing the filesystem works by thumb.

### Design system: status, colour, type, motion

80. As an admin, I want every status (Available, Missing, Downloading, Upgradable, Unreleased, Failed) to use one hue everywhere it appears, so that a colour always means the same thing.
81. As an admin, I want status shown as a dot beside text in dense contexts and as a tinted pill with a dot in prominent contexts, replacing solid coloured badges, so that status never shouts.
82. As an admin, I want movie content to keep its orange and series content its blue for progress, active pills and media marks, so that media type stays legible.
83. As an admin, I want a single type scale (display, heading, title, body, footnote, caption) with tighter tracking at larger sizes, so that typography is consistent across pages.
84. As an admin, I want numbers that update live rendered in tabular figures, so that neighbouring text does not shift.
85. As an admin, I want release names and file paths in a monospace face at a small size, so that technical strings are distinguishable from titles.
86. As an admin, I want every pressable element to scale down slightly on press with an ease-out curve, so that the interface confirms every touch.
87. As an admin, I want list rows to tint on press rather than scale, so that long lists do not wobble.
88. As an admin, I want hover styles only on devices with a precise pointer, so that a tapped element never stays highlighted.
89. As an admin, I want entrances (list stagger, sheet, pushed screen) to use a strong ease-out and stay under 300 ms, with exits faster than entrances, so that motion feels quick and intentional.
90. As an admin, I want lists to animate in only on first mount and not when I return to a tab, so that revisiting never replays motion.
91. As an admin who prefers reduced motion, I want slides and scales replaced by short cross-fades and press scale removed, so that the app respects my system setting.
92. As an admin who prefers reduced transparency, I want translucent bars to become solid, so that text stays legible.
93. As an admin, I want the light theme preserved with the same tokens defined for it, so that choosing light mode does not break status colours or materials.
94. As an admin, I want hover glows on cards and nav items removed and glow reserved for active download progress and the completion flash, so that emphasis means something.

### Platform baseline and feedback

95. As an admin on my phone, I want no grey tap flash, no page zoom on inputs, no pull-to-refresh hijack inside scrolling lists, and no text selection when I long-press a control, so that the app feels installed rather than embedded.
96. As an admin on my phone, I want the browser status bar colour to match the app's top bar in both colour schemes, so that the chrome blends in.
97. As an admin on my phone, I want the layout to use the dynamic viewport height so bottom-pinned bars stay above the browser chrome and the keyboard, so that the tab bar never disappears.
98. As an admin, I want toasts positioned above the tab bar on phones and bottom-right on wide screens, so that notifications never cover navigation.
99. As an admin, I want loading states to use skeletons shaped like the final grouped rows and cells, so that the layout does not jump when data arrives.
100. As an admin, I want Developer Tools (dev mode toggle, global loading) available from More on phones and the header on wide screens, so that the dev workflow is preserved in both shells.
101. As an admin, I want focus rings visible for keyboard users on every interactive element in both shells, so that the overhaul does not regress accessibility.
102. As an admin, I want all icon-only controls to carry accessible labels, so that screen readers can name every button.

## Implementation Decisions

### Scope and strategy

- This is a presentation-layer overhaul with feature parity. Routes, TanStack Query hooks, WebSocket invalidation, Zustand stores, the module registry and the backend are unchanged except where noted below.
- One codebase, two shells, selected by a single breakpoint at 768 px. The shell decision (tab bar versus sidebar, sheet versus dialog, push versus instant) is made by one viewport hook; every other responsive difference is CSS.
- The portal (`/requests` for portal users) is untouched.
- The Native prototype on branch `cursor/mobile-design-system-prototypes-20af` is the reference implementation. Its shell, screens and primitives are ported into the production component tree; the Native variant, the harness (phone frame, picker) and the shared primitives and mock data it depends on are kept on the branch for reference. The Cinematic and Console variants are removed when the port lands.

### Token layer

- The mobile token layer becomes the app-wide token layer, imported into the global stylesheet. The `m-` prefixes on type and utility names are dropped in production (for example `text-display`, `.press`, `.material`), since they are no longer mobile-specific.
- Motion tokens: three easing curves (strong ease-out, strong ease-in-out, drawer), four durations (press 120, fast 160, base 220, sheet 320). No UI motion exceeds 320 ms. `transition-all` is removed from the codebase.
- Type tokens: six steps with paired leading and tracking as documented in `docs/mobile-design-system.md`.
- Touch tokens: 44 px minimum tap target, 52 px large, 16 px screen gutter, 56 px tab bar height, sheet radius 20, card radius 14, thumbnail radius 6.
- Status tokens: six hues. Light-theme values are added alongside the dark ones. All status rendering goes through the shared StatusDot and StatusPill components; MediaStatusBadge is replaced by StatusPill and removed.
- Material tokens: bar, heavy and chip materials with reduced-transparency fallbacks, defined for both themes.
- Safe-area variables resolve from `env()`; the platform baseline (viewport-fit, theme-color per scheme, tap highlight, text-size-adjust, overscroll, 16 px inputs, touch-action on controls) moves into the production HTML entry and global stylesheet.
- Glow utilities are retained only for active download progress and the download completion flash; hover-glow and card-glow usages are removed.

### Shell and navigation

- A ShellLayout replaces the current RootLayout body. On phones it renders the tab bar and a screen stack; on wide screens it renders the sidebar and a content column. Auth gating, WebSocket lifecycle, theme and dev-mode effects are unchanged.
- Tab bar destinations map to existing routes: Dashboard `/`, Library (last-used module's list route), Activity `/downloads`, Search `/search`, More (new `/more` route rendering the grouped index). The Activity badge reads the same queue query the sidebar's downloads link uses today.
- Tab screens stay mounted and are hidden when inactive so scroll position survives. Tab switches have no transition.
- Detail routes (`/movies/$id`, `/series/$id`), add routes, settings sub-routes, requests admin, system and `/more` sub-screens are "pushed" screens on phones: they render above the tab root with a slide-in (260 ms) and a faster slide-out (200 ms), the layer beneath shifts and dims. The back button label is derived from the route hierarchy (parent list or "More"). On wide screens the same routes render instantly in the content column with a back button but no motion.
- The DownloadsProgressOverlay (ring, glow, completion flash) is kept on the wide-screen sidebar row and is not rendered on phones; the tab badge carries the count instead.
- The `useUIStore` keeps `sidebarCollapsed`, `theme`, view and column preferences, and gains the last-used Library module.

### Page patterns (shared components)

- Screen: the scroll container plus top bar. Props: title, back, trailing actions, large-title on/off, transparent-until offset. Owns the collapse behaviour (bar background and compact title fade in once content scrolls past a threshold). PageHeader is replaced by Screen and removed.
- Group and Row: inset grouped list with optional header and header action; rows with leading (icon tile or thumbnail), title, subtitle, trailing value, chevron, tone (default, warning, destructive) and 44 px minimum height. Card-based ListSection and AddPlaceholderCard in settings are replaced by Group/Row and removed.
- Segmented: sliding-thumb control for two to five options; replaces the Tabs primitive in Library module switching, Activity filter, Missing, Calendar toggle and Requests queue states. Tabs remains available for content that is genuinely tabbed.
- Chip row: horizontally scrolling filter chips; replaces the FilterDropdown for the primary filter set. Secondary options (sort, view, poster size, columns) move behind a single trailing action.
- PosterCell: poster with title, year, quality and status dot beneath; replaces the per-module card components (MovieCard, series card). Modules no longer supply a card component; the cell reads title, year, quality and status from the module's existing item type. The `cardComponent` field is removed from ModuleConfig.
- Pill action: 44 px rounded action with icon and label, media-tinted when active; wraps the existing MediaSearchMonitorControls states so searching, progress, completed and error states render inside the pill row. The `xs`/`sm`/`lg`/`responsive` size matrix collapses to one pill size plus a compact row variant for lists.
- Action sheet: bottom-anchored list of actions with title, optional monospace subtitle, destructive tone and Cancel; used on phones for row actions and confirmations. On wide screens the same action list renders as a dropdown menu or alert dialog through one presenter component.
- Sheet: draggable bottom sheet for forms on phones (1:1 drag, rubber-band, velocity dismiss, catchable mid-animation), using vaul; the same form renders inside Dialog on wide screens through the same presenter. All existing dialogs (media edit, delete, indexer, download client, quality profile, notification, token builder, dry-run, invite, user edit, request search, confirm) adopt the presenter.
- Skeletons are reshaped to match Group/Row and PosterCell.

### Motion policy

- Entrances only on first mount, staggered 40 ms and capped at 200 ms; never replayed on tab return.
- Press feedback on pointer-down via `:active`: scale 0.97 for buttons and cells, background tint for rows, opacity for text links.
- Only `transform` and `opacity` animate, with the progress bar width (700 ms linear) as the sole exception.
- Reduced motion swaps slides and scales for 160 ms fades and removes press scale; reduced transparency makes materials solid.
- Keyboard-initiated navigation and tab switches never animate.

### Data and modules

- No API or schema changes. No backend changes.
- ModuleConfig loses `cardComponent`; it keeps `listComponent`, `detailComponent`, filter and sort options, table columns, query keys and WS rules. The Library segmented control and the Missing page read `getEnabledModules()` as the sidebar does today.
- Detail screens keep the existing loaders and prefetches.

### Removal of legacy

When the port lands, the following are removed: PageHeader, MediaStatusBadge, ListSection, AddPlaceholderCard, per-module card components, the header search bar's mobile fallback, hover-glow utilities and usages, `transition-all` usages, and the Cinematic and Console prototype variants. The Native prototype and its harness stay on branch `cursor/mobile-design-system-prototypes-20af` for reference; whether they ship to `main` or are dropped once the production shell is complete is a decision for the final ticket, not this spec.

## Testing Decisions

### The seam

There are no frontend tests today; the only feedback loops are TypeScript, ESLint and manual
checks. The Go backend is tested at the package level under `internal/` and is not touched by
this work.

One new seam, at the highest point available: **browser-level end-to-end tests with
Playwright against the running app in developer mode**. Developer mode gives a separate
database and mock metadata, indexer, download client and notification services, so the
suite is deterministic without network access. Two Playwright projects run the same tests:
a phone profile (390 × 844, touch, device scale 2) and a wide profile (1440 × 900, mouse).
Where a behaviour exists in only one shell (tab bar, sidebar) the test is scoped to that
project.

No component or unit tests of the presentation layer. Tokens, class names, transitions and
component boundaries are implementation details that will keep changing; the tests assert
what the admin sees and can do.

**Confirm this seam before implementation begins.** It is the one decision here that adds
tooling to the repo (a Playwright dev dependency, a config, a test script and a CI step).

### What makes a good test here

- It starts from a URL, uses roles and visible text to find things, and asserts on visible
  outcomes: a screen is shown, a count changes, a status label changes, a row disappears.
- It performs the interaction the way the admin does: tap on the phone project, click on the
  wide project, keyboard where the story is about keyboard.
- It never asserts on class names, animation timing or DOM structure. Motion is tested only by
  its outcome (the pushed screen is visible; the previous screen is not) and by a reduced-motion
  smoke run to confirm the app remains operable with animations disabled.
- Screenshot comparison is a secondary, opt-in signal for the key screens (Dashboard, Library
  grid, Detail, Activity, Settings list) in both projects and both themes, used to catch
  layout regressions rather than to pin pixels.

### Coverage

- Shell: tab bar destinations and badge; sidebar sections and collapse; push and back with
  correct back label; same URL renders in both shells; current-page marking.
- Dashboard: four groups render with dev-mode data; health row opens System; "See all" opens
  Activity.
- Library: module segmented switch; grid on phone, grid and table on wide; chip filters change
  the visible set; edit mode bulk toolbar; cell opens detail.
- Detail: hero content; Monitored pill toggles and persists after reload; search control states
  using the dev-mode mock indexer; overview expand; edit and delete through sheet or dialog.
- Activity: summary line; pause and resume toggle the row's state text; action sheet actions;
  segmented filter; unreachable-client banner when the mock client is toggled off.
- Search: 16 px input does not zoom (asserted by viewport scale unchanged after focus on the
  phone project); results open detail.
- Settings: each list renders rows; add opens sheet or dialog; a created item appears as a row;
  toggles persist.
- Requests admin: segmented states; approve and deny flows.
- Platform: safe-area padding applied when insets are simulated; toasts positioned above the
  tab bar; reduced-motion run passes the shell tests.
- Accessibility: every icon-only control has a name; focus is visible on tab traversal in the
  wide project.

### Prior art

- No frontend prior art in the repo. The headless Chrome tour used to verify the prototypes on
  this branch (role-based locators, per-step assertions, console-error capture) is the model
  for test style.
- Go tests under `internal/` are table-driven behaviour tests against public package
  functions; the same posture (behaviour, not internals) applies here.
- Lint and `tsc -b` remain gates and run before the browser suite.

## Out of Scope

- The portal for external requesters under `/requests` (its own layout and auth).
- Backend, API, database or module-framework changes beyond removing `cardComponent` from the frontend ModuleConfig.
- New features. Every story above is a re-presentation of an existing capability; anything not in the app today (swipe-to-go-back, offline support, push notifications, home-screen widgets) is a follow-up.
- Native iOS or Android apps and app-store packaging. Full PWA installability (manifest, service worker, offline cache) is a follow-up; only the theme-color and viewport baseline are in scope.
- The Cinematic and Console directions from the exploration. Their prototype variants are deleted; the Native variant and harness are kept for reference.
- Tablet-specific layouts between the two shells. Tablets get the wide shell.
- Migrating existing forms to a single form library. Forms keep their current state management (react-hook-form with zod where present, local state elsewhere) and only change presentation.
- Redesigning the `/dev/controls` showcase beyond re-skinning the controls it shows.

## Further Notes

- The chosen direction and its trade-offs are recorded in `docs/mobile-design-system.md`
  section 3; the promotion checklist in section 5 is superseded by this spec.
- Poster and backdrop artwork uses the existing PosterImage, BackdropImage and TitleTreatment
  components with their cache fallbacks; the gradient stand-ins in the prototype do not ship.
- Suggested tracer-bullet order for `/to-tickets`: token layer and platform baseline; shell
  (tab bar, sidebar, Screen, push navigation) around the existing Dashboard; Dashboard groups;
  Library (segmented, chips, PosterCell, edit mode); Detail (hero, pills, control states,
  groups); Activity (rows, action sheet, presenter); Search; Settings lists and the
  sheet/dialog presenter across all forms; System and Requests admin; Calendar, Missing,
  History, Manual Import; legacy removal (Cinematic and Console variants). Each slice is shippable on
  its own because the shell is route-based and unaffected pages keep working inside it.
- When a production pattern and the Native prototype disagree during implementation, the
  prototype on `cursor/mobile-design-system-prototypes-20af` wins unless this spec says
  otherwise; if the spec is changed, note it under Comments below.
- Real-hardware verification is required before calling the phone shell done: sticky hover,
  tap delay, safe areas, keyboard behaviour and sheet drag cannot be judged in device
  emulation or in the Playwright phone profile.
- The Base UI conventions in `web/CLAUDE.md` (render prop composition, SelectValue label
  rendering) continue to apply to every new primitive.

## Comments

### The prototypes' fate (ticket 18)

**Decision: the whole `web/src/prototypes/` tree and the `web/prototypes/mobile/index.html`
harness entry point are removed from `main`.** The Cinematic and Console variants were already
slated for deletion; the Native variant and its harness go with them.

Reasoning. The production shell is the reference now: every part the Native prototype was drawn to
answer — the tab bar and sidebar, `Screen` and push navigation, the grouped list, the action sheet,
the segmented control, the poster grid — exists in `web/src/components/` with e2e tests and
screenshot baselines behind it, and reads better there than in a mocked copy. Nothing in production
imported from the tree (the dependency ran the other way: the prototype imported `@/lib/utils`), and
the harness was not in the Vite build, so it was reachable only by typing its path into a dev
server — a page nobody would find and nobody would keep true. A second, mock-data implementation of
the same shell is a standing invitation to drift. The reference is preserved where the spec already
says it is: branch `cursor/mobile-design-system-prototypes-20af`, plus
`docs/mobile-design-system.md` section 3 for the direction and its trade-offs.

### Real-hardware checklist (ticket 18)

Not run: it needs a physical iOS and a physical Android device. See the ticket's Comments for what
to check and how to reach the dev server from a phone.
