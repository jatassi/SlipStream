# E. React frontend libraries beyond ADR 0013

> Spec numbers follow the feature graph as renumbered on 2026-10-11, with seven foundation specs (S01 to S07) ahead of the walking skeleton (S09). "S01-S09" means the foundations plus the skeleton.


Area: the embedded React app's primitives that ADR 0013 does not pin. ADR 0013 already fixes React 19.3, TS 7, Vite 8 with React Compiler, TanStack Router, Query and Virtual, Tailwind 4, Motion, Oxlint, Oxfmt, Vitest, Orval (Query hooks, Zod, MSW mocks). `docs/offline-boundaries.md` row 43 already fixes **Geist and Lucide, bundled**. Row 22 fixes **Playwright's virtual authenticator** as the WebAuthn fake. Row 47 fixes **Orval MSW mocks** for component tests.

## How this was verified

- **npm registry**, queried this session (2026-10-11): latest version, its publish date, license, React peer range and deprecation flag for every package below. "Released" dates are npm publish dates of the latest version.
- **@mdn/browser-compat-data 8.1.5** (2026-10-08), queried this session, for every browser-support claim.
- **React Compiler source** (`DefaultModuleTypeProvider.ts` on `main`), read this session, for the incompatible-library list.
- **Sizes** are bundlephobia whole-package min+gzip, fetched this session. They are before tree-shaking, so read them as an upper bound. Blank means bundlephobia failed to build it.
- **The GitHub API was not reachable** from this session (403). Stars, open-issue counts and last-commit months are therefore *not* recorded, except where a repo page was read with WebFetch (vaul, cmdk). Treat "maintenance" below as "release cadence on npm".

## What the prototypes used

- `prototype/ui-direction` (variant C, "Lit console", is the settled reference per PRODUCT.md) is plain HTML, CSS and ES modules. Its only library is `lucide@0.469.0` from unpkg, plus Geist and Geist Mono from Google Fonts. It hand-builds the phone shell: fixed bottom `.tabs`, `.ph-nav` push header, `.sheet` action sheets sliding up on `cubic-bezier(.32,.72,0,1)`, and heavy `backdrop-filter` materials.
- `prototype/portal-onboarding` uses only Lucide too. `prototype/release-judging-settings` is vanilla JS; its variant A orders priorities by dragging.
- That branch also carries a `pick-ui-library` skill (Emil Kowalski's curated list): base-ui, cmdk, Sonner, input-otp, motion, recharts, dnd kit, Virtuoso, zustand, clsx, cva, next-themes. It is a taste list, not a decision. Its Virtuoso pick conflicts with ADR 0013's TanStack Virtual, and next-themes is a Next.js tool.
- **There is no DESIGN.md yet.** PRODUCT.md says DESIGN.md records variant C once the build ships (S11, [#94](https://github.com/jatassi/SlipStream/issues/94)).

## Summary table

| # | Primitive | Specs | Verdict | Pick | Criticality |
|---|---|---|---|---|---|
| 1 | Headless accessible primitives | S11 and every UI spec | adopt | Base UI 1.9 | critical |
| 2 | Copy-in component layer | S11 | borrow | shadcn/ui (Base UI flavour) as reading material | convenience |
| 3 | Bottom sheets and action sheets | S11, S20, S32, S41 | adopt (via 1) | Base UI Drawer; not vaul | important |
| 4 | Push navigation, swipe-back, view transitions | S11, S20, S41 | build | React 19.3 `<ViewTransition>` + Motion | important |
| 5 | Forms + Orval Zod | S12, S17, S23, S24, S25, every settings spec | adopt | TanStack Form 1.x | important |
| 6 | Command palette / global search | S20 (P-006, P-019, P-020) | build on 1 | Base UI Dialog + Autocomplete | important |
| 7 | Toasts | S11 (P-013) | adopt (via 1) | Base UI Toast | convenience |
| 8 | Dates and calendar | S38, S57, S28, S34 | adopt + build | `Intl` + date-fns 4; native `<input type="date">` | convenience |
| 9 | Drag to reorder Targets and priorities | S17, S23, S53 | build on Motion | Motion `Reorder` + Move up/down actions | important |
| 10 | Charts, sparklines, meters | S29, S36, S39, S28, S15 | build (SVG) + wrap visx if needed | hand SVG; visx for S36 | convenience |
| 11 | Icons and fonts | S11 | adopt (maintainer's pick) | Hugeicons, Geist via Fontsource | convenience |
| 12 | Markdown for release notes | S57 | adopt, lazy | react-markdown 10 (or server-rendered) | convenience |
| 13 | Virtualized grids and tables | S20, S28, S14 | adopt (decided) + build | TanStack Virtual; no table lib | important |
| 14 | Client state | S11, S20 | build | `@tanstack/react-store` (already shipped by Router) or `useSyncExternalStore` | convenience |
| 15 | SSE client | S01-S09, S13, S14, S28, S39, every live page | build | native `EventSource` + Orval Zod per event | critical |
| 16 | Passkeys in the browser | S12 | build | native `PublicKeyCredential` JSON methods | critical (security) |
| 17 | PIN entry | S12, S25 | adopt (via 1) | Base UI OTP Field | important |
| 18 | Tests and accessibility checks | S10, S11, all | adopt | Playwright, Vitest browser mode, axe-core | important |

---

## 1. Headless accessible primitives

**Specs.** S11 owns them. Every UI spec consumes them: S12, S20, S32, S41 most heavily.
**Current plan.** Unspecified. ADR 0013 pins styling (Tailwind 4) and motion (Motion) only. PRODUCT.md requires WCAG 2.2 AA with full keyboard paths.

| Candidate | Repo | License | Latest (npm) | Released | Size gz (whole pkg) | React 19 / Compiler | Notes | Verified |
|---|---|---|---|---|---|---|---|---|
| Base UI `@base-ui/react` | github.com/mui/base-ui | MIT | 1.9.0 | 2026-10-09 | bundlephobia failed | peer `^17‖^18‖^19`; nothing on the compiler's incompatible list | 1.0.0 shipped 2025-12-11. Old `@base-ui-components/react` is deprecated ("renamed"). Exports include Dialog, AlertDialog, **Drawer**, **Toast**, Menu, Popover, Select, **Combobox**, **Autocomplete**, **OTP Field**, NumberField, Tabs, Field, Form, Toolbar, ScrollArea, plus internal Temporal adapters (date-fns, luxon) | yes |
| Radix `radix-ui` | github.com/radix-ui/primitives | MIT | 1.7.0 | 2026-10-05 | 73 KB | peer up to `^19` | Mature and widely used. No Drawer, Combobox or OTP field. Many Radix authors now work on Base UI | yes (npm) |
| React Aria Components | github.com/adobe/react-spectrum | Apache-2.0 | 1.22.1 | 2026-10-09 | 279 KB | peer up to `^19` | Strongest a11y and i18n (Adobe), with keyboard drag and drop, date pickers and a calendar. Heaviest by far. Render-prop and data-attribute styling idiom | yes (npm) |
| Ariakit `@ariakit/react` | github.com/ariakit/ariakit | MIT | 0.4.41 | 2026-10-05 | 65 KB | peer up to `^19` | Still 0.x. Essentially a single maintainer | yes (npm); bus factor inferred |

**Verdict: adopt Base UI.** It is the only candidate that covers the phone shell's own pieces (Drawer with swipe and snap points, Toast, OTP field, Combobox) under one team and one idiom, so S11 needs no vaul, Sonner, cmdk or input-otp. v1 already used it, so the maintainer knows it. React Aria is the a11y gold standard, but at roughly 4x Radix's size it costs a lot for a self-hosted app's phone users, and its date and DnD extras are things this app barely needs. Building our own primitives would mean re-deriving focus traps, dismiss layers, typeahead, `aria-activedescendant` combobox rules and the iOS VoiceOver quirks. That is several thousand lines of the riskiest a11y code, against WCAG 2.2 AA, for no product gain. The dependency's cost is a large API surface tied to MUI's release cadence. Base UI's 1.x has shipped about monthly since December 2025.
**Criticality.** Critical: a broken focus trap or dialog fails the AA requirement app-wide.
**ADR conflict.** None.

## 2. shadcn/ui as a copy-in layer

**Specs.** S11.
**Current plan.** Unspecified. v1 used shadcn.

| Candidate | Repo | License | Latest | Released | Notes | Verified |
|---|---|---|---|---|---|---|
| `shadcn` CLI and registry | github.com/shadcn-ui/ui | MIT | 4.21.4 | 2026-10-07 | The docs site serves Base UI variants under `/docs/components/base/…`. How a project selects Base UI vs Radix (style name or init flag) was **not** confirmed on ui.shadcn.com: third-party pages disagree | npm yes; Base UI mode partly |

**Verdict: borrow.** shadcn's value is pre-styled Tailwind wrappers in a neutral look. Variant C's materials (blur, hue-tinted translucency, Geist Mono numerals) will replace nearly every class anyway. Running its CLI would bring `cva`, `tailwind-merge`, `clsx` conventions and a generic aesthetic that the impeccable skill warns against. Read its Base UI wrappers for structure (slot names, `data-*` styling hooks) while writing S11's own components. The custom version is about 30 thin wrapper components over Base UI, each 30 to 100 lines, and that work is S11's actual deliverable.
**Criticality.** Convenience.

## 3. Bottom sheets and action sheets

**Specs.** S11 (action sheets, P-016), S20 (the add sheet), S32 (manual search sheet, P-123), S41 (Member request flows), S40 (deny sheet, P-299).
**Current plan.** Unspecified. v1 used vaul.

| Candidate | Repo | License | Latest | Released | Notes | Verified |
|---|---|---|---|---|---|---|
| Base UI Drawer | (see 1) | MIT | 1.9.0 | 2026-10-09 | Extends Dialog. Swipe to dismiss by default, `swipeDirection`, `snapPoints`, nested drawers, `Drawer.VirtualKeyboardProvider` for form sheets, and an action-sheet example in the docs. Not marked preview | yes (docs read) |
| vaul | github.com/emilkowalski/vaul | MIT | 1.1.2 | 2024-12-14 | **The README says "This repo is unmaintained"** | yes |

**Verdict: adopt Base UI Drawer.** vaul is abandoned by its author, and Base UI now ships the same gesture model inside the primitive set already chosen. A custom sheet is about 300 lines of pointer, velocity and focus code, and the keyboard-avoidance case on iOS is where it usually breaks.
**Criticality.** Important: action sheets are the phone shell's main control (P-016).

## 4. Push navigation, swipe-back and view transitions

**Specs.** S11 (phone shell), S20 (Library to Work page), S41.
**Current plan.** ADR 0013 Consequences: "The walking skeleton checks that React's `<ViewTransition>` works with TanStack Router's navigation."

Facts checked this session:
- React **19.3.0** (npm, 2026-09-09) exports a stable `ViewTransition`.
- TanStack Router (1.170.42, 2026-10-10) has `defaultViewTransition` (`true` or `{ types }`), which calls `document.startViewTransition` itself.
- `document.startViewTransition`: Chrome 111, Firefox 144, Safari 18. View transition `types`: Chrome 125, Firefox 147, Safari 18.2.
- `PopStateEvent.hasUAVisualTransition`: Chrome 118, Firefox 149, Safari 18. It tells the app that the browser already animated a back swipe, so the app must not animate again.
- Navigation API: Safari 26.2, Firefox 147.

No library does iOS-style push with edge swipe-back for the web. The pieces are Motion (decided) for the drag, and the history stack from the Router.

**Verdict: build.** Use React's `<ViewTransition>` with directional types (`push` and `pop`) and leave the Router's `defaultViewTransition` off, so only one layer calls `startViewTransition`. In the browser, iOS Safari's own edge swipe already goes back. Skip our transition when `hasUAVisualTransition` is true. In an installed PWA (standalone) there is no system swipe, so add a Motion edge-drag that calls `router.history.back()`. Cost is about 200 to 400 lines. The risk is the double-animation bug, which the walking skeleton should test on a real iPhone, as the `mobile-native` skill insists. Nothing to depend on.
**Criticality.** Important.

## 5. Forms with Orval's Zod schemas

**Specs.** S12 (sign-in, Signup, PIN change), S17 (Targets, Profiles), S23 (Custom Format editor and tester), S24 (naming token builder with live preview), S25 (first run), and every settings page.
**Current plan.** Zod via Orval is decided (ADR 0013). The form library is unspecified. v1 used react-hook-form.

| Candidate | Repo | License | Latest | Released | Compiler | Notes | Verified |
|---|---|---|---|---|---|---|---|
| TanStack Form `@tanstack/react-form` | github.com/TanStack/form | MIT | 1.33.5 | 2026-08-11 | not on the incompatible list | Takes Standard Schema validators directly, and Zod 4 (4.6.6) is one, so Orval's schemas plug in with no resolver. Depends on `@tanstack/react-store`, which TanStack Router already ships | yes |
| react-hook-form | github.com/react-hook-form/react-hook-form | MIT | 7.90.0 | 2026-10-11 | **`useForm().watch` is on React Compiler's incompatible list** (use `useWatch`) | Mature. Needs `@hookform/resolvers` 5.9.1 for Zod | yes |
| Plain React 19 actions (`useActionState`, `FormData`, `schema.safeParse`) | n/a | n/a | n/a | n/a | fine | Enough for the PIN and Signup forms. Weak for nested arrays (Target qualities, CF conditions) and field-level async checks | n/a |

**Verdict: adopt TanStack Form.** It stays in the TanStack family the ADR already chose, consumes Orval's Zod output with no adapter, adds almost no bundle because its store is already present, and has no compiler footgun. Pair it with Base UI `Field` for labels, descriptions and error wiring. Plain actions would be fine for the five or six simple forms, but S17, S23 and S24 are array-heavy editors where hand-rolled dirty and touched state turns into custom framework code.
**Criticality.** Important: a wrong form library spreads across every settings page and is costly to swap.

## 6. Command palette and global search

**Specs.** S20 (P-006 header search, P-019 instant library match, P-020 Add New from TMDB).
**Current plan.** Unspecified. v1 used cmdk.

| Candidate | Repo | License | Latest | Released | Size gz | Notes | Verified |
|---|---|---|---|---|---|---|---|
| cmdk | github.com/pacocoursey/cmdk | MIT | 1.1.1 | 2025-03-14 | 15 KB | No release in 19 months. 52 open issues and 24 open PRs on the repo page | yes |
| Base UI Autocomplete / Combobox in a Dialog | (see 1) | MIT | 1.9.0 | 2026-10-09 | in 1 | Gives the listbox, `aria-activedescendant` and grouping | yes |

**Verdict: build on Base UI.** cmdk's main value is client-side fuzzy ranking over a static command list. SlipStream's global search is mostly server results (library match and TMDB Add New), streamed through TanStack Query. A Dialog plus Autocomplete covers keyboard and screen reader behaviour, and the ⌘K binding is a 10-line `keydown` listener. `@tanstack/react-hotkeys` (0.13.0) is still 0.x, so skip it. The custom part is about 150 lines.
**Criticality.** Important (P-006, P-019 and P-020 are parity rows).

## 7. Toasts

**Specs.** S11 (P-013, "toasts placed per shell"). Used by S34 (P-135 Search All) and S40 (P-301).
**Current plan.** Unspecified. v1 used Sonner.

| Candidate | Repo | License | Latest | Released | Notes | Verified |
|---|---|---|---|---|---|---|
| Base UI Toast | (see 1) | MIT | 1.9.0 | 2026-10-09 | Headless. Live region, swipe dismiss, stacking | yes (export present) |
| Sonner | github.com/emilkowalski/sonner | MIT | 2.0.8 | 2026-08-09 | Opinionated look, with promise toasts out of the box | yes |

**Verdict: adopt Base UI Toast** behind a small `toast()` module. One primitive family, and per-shell placement is just two viewports. Sonner is good and still released, so it is the fallback if Base UI's toast proves rough. Its styling would still need overriding to match variant C.
**Criticality.** Convenience.

## 8. Dates, times and the calendar

**Specs.** S38 (month and list calendar, History by day), S57 (install window), S28 and S34 (relative times, ETAs).

| Candidate | Repo | License | Latest | Released | Size gz | Notes | Verified |
|---|---|---|---|---|---|---|---|
| `Intl.DateTimeFormat`, `RelativeTimeFormat`, `DurationFormat` | platform | n/a | n/a | n/a | 0 | DurationFormat: Chrome 129, Firefox 136, Safari 16.4 | yes (BCD) |
| date-fns | github.com/date-fns/date-fns | MIT | 4.4.0 | 2026-05-29 | tree-shaken | Grid maths: `startOfMonth`, `eachDayOfInterval`, week starts. `@date-fns/tz` for zones | yes (npm) |
| Native Temporal | platform | n/a | n/a | n/a | 0 | Chrome 144, Firefox 139, **Safari: preview only, iOS none** | yes (BCD) |
| `temporal-polyfill` (FullCalendar) | github.com/fullcalendar/temporal-polyfill | MIT | 1.0.5 | 2026-09-11 | 20 KB | Needed until Safari ships Temporal | yes |
| `@js-temporal/polyfill` | github.com/js-temporal/temporal-polyfill | ISC | 0.5.1 | 2025-03-31 | — | Stale | yes |
| react-day-picker | github.com/gpbl/react-day-picker | MIT | 10.0.2 | 2026-09-30 | 20 KB | Date-picker widget | yes |

**Verdict: adopt date-fns 4 for calendar grid maths, and use `Intl` for all formatting.** Build the S38 month grid with Base UI-free semantic markup: a `table`/`grid` with day buttons. Use a native `<input type="date">` or `type="datetime-local"` for the few pickers (History range, install window). Native inputs are the most accessible option on phones, so skip react-day-picker. Temporal is the better model, but it would ship a 20 KB polyfill on every iPhone with no end date, so revisit when Safari ships it.
**Criticality.** Convenience.

## 9. Drag to reorder Targets, qualities and priorities

**Specs.** S17 (Targets with ordered qualities, Profiles with ordered Targets), S23 and S53 (priority ladders; the release-judging prototype's variant A is drag-to-order).

| Candidate | Repo | License | Latest | Released | Size gz | Notes | Verified |
|---|---|---|---|---|---|---|---|
| Motion `Reorder` | (decided, ADR 0013) | MIT | motion 14.1.0 | 2026-10-09 | in Motion | Pointer and touch list reorder with layout animation. No keyboard path | npm yes; `Reorder` from prior knowledge |
| `@dnd-kit/react` | github.com/clauderic/dnd-kit | MIT | 0.5.0 | 2026-06-11 | 33 KB | The rewrite, still 0.x. `@dnd-kit/core` 6.3.1 last published 2024-12-05 | yes |
| pragmatic-drag-and-drop | github.com/atlassian/pragmatic-drag-and-drop | Apache-2.0 | 4.0.0 | 2026-09-24 | tiny core | Built on native HTML5 DnD. Atlassian's own guidance pairs it with explicit keyboard actions rather than keyboard dragging | npm yes; a11y guidance inferred |
| React Aria `useDragAndDrop` | (see 1) | Apache-2.0 | 1.22.1 | 2026-10-09 | in RAC | Full keyboard and screen-reader DnD, but only inside RAC collections | yes (npm) |

**Verdict: build on Motion.** The lists are short (under 20 rows) and single-container. Motion `Reorder` handles pointer and touch, and every row also gets **Move up / Move down / Move to top** in its row menu (an action sheet on phones), with an `aria-live` announcement. That keyboard path is what WCAG 2.2's 2.5.7 (Dragging Movements, AA) requires anyway, whichever library drags. The cost is about 150 lines. dnd-kit's stable line has gone stale and its successor is pre-1.0, so a dependency buys little here.
**Criticality.** Important: Target order decides grabs (ADR 0002).

## 10. Charts, sparklines and meters

**Specs.** S39 (Home: live pipeline, Downloading, Storage), S29 (two-lane goal meter), S28 (torrent speed), S36 (playback report), S15 (Nodes canvas).
**Accessibility.** PRODUCT.md requires "screen-reader equivalents for any visual that carries state, such as a canvas".

| Candidate | Repo | License | Latest | Released | Size gz | Notes | Verified |
|---|---|---|---|---|---|---|---|
| visx | github.com/airbnb/visx | MIT | 4.0.0 (`@visx/shape`) | 2026-06-11 | per module | Low-level SVG primitives and d3 scales. Easy to style with tokens | yes |
| recharts | github.com/recharts/recharts | MIT | 3.10.1 | 2026-07-25 | 152 KB | Batteries included, heavy | yes |
| uPlot | github.com/leeoniya/uPlot | MIT | 1.6.32 | 2025-03-14 | 22 KB | Canvas, very fast for streams. Needs its own table equivalent | yes |
| `@xyflow/react` (Nodes canvas) | github.com/xyflow/xyflow | MIT | 12.12.0 | 2026-09-24 | — | Graph canvas, if S15's canvas needs pan, zoom and edges | yes |

**Verdict: build sparklines and meters as hand-written SVG components** (each about 50 to 100 lines, themed by tokens, with an `aria-label` and a visually hidden summary). **Wrap visx** only if S36's report needs real axes. Recharts' 150 KB is unjustified for a few charts, and uPlot's canvas adds SR-equivalent work. For S15's Nodes canvas, try a hand SVG first: an install has a handful of Nodes. Revisit xyflow only if pan and zoom over many Nodes becomes a requirement, and pair it with the Roster table as the SR equivalent.
**Criticality.** Convenience.

## 11. Icons and fonts

**Specs.** S11.
**Current plan.** Decided in offline-boundaries row 43: Geist and Lucide bundled.

| Candidate | License | Latest | Released | Verified |
|---|---|---|---|---|
| lucide-react | ISC | 1.55.0 | 2026-10-10 | yes |
| `@fontsource-variable/geist` (and `geist-mono`) | OFL-1.1 | 5.3.0 | 2026-07-19 | yes |
| @phosphor-icons/react (alternative) | MIT | 2.1.10 | 2025-05-22 | yes |

**Superseded on 2026-10-11: the maintainer picked Hugeicons** (`@hugeicons/react` 1.1.10 and `@hugeicons/core-free-icons` 4.3.5, both MIT on npm). Only the free set ships. Pro icons are paid and not MIT, so they likely cannot be redistributed in a GPL-3.0 repository (inferred, license not read). The change reopens offline-boundaries row 43 and the S06 row of the feature graph, which both name Lucide.

**Original verdict: adopt lucide-react and Fontsource Geist** (self-hosted woff2 through Vite, no Google Fonts call, as the prototype made). OFL fonts ship alongside GPL-3.0 code as separate data, which is fine, but list them in the notices file. Phosphor is staler and would reopen a settled boundary row.
**Criticality.** Convenience.
**ADR conflict.** Switching icons would reopen offline-boundaries row 43 (from [#26](https://github.com/jatassi/SlipStream/issues/26) Q15), not an ADR.

## 12. Markdown for release notes

**Specs.** S57 (P-266, release notes on the Updates page).

| Candidate | Repo | License | Latest | Released | Size gz | Notes | Verified |
|---|---|---|---|---|---|---|---|
| react-markdown | github.com/remarkjs/react-markdown | MIT | 10.1.0 | 2025-03-07 | 34 KB | Ignores raw HTML by default, so it needs no sanitizer | yes |
| markdown-to-jsx | github.com/quantizor/markdown-to-jsx | MIT | 9.10.3 | 2026-09-15 | 28 KB | Actively released | yes |
| marked + DOMPurify | marked MIT; DOMPurify MPL-2.0 or Apache-2.0 | 18.1.0 / 3.4.16 | 2026-10 / 2026-09 | — | Needs `dangerouslySetInnerHTML` | yes |

**Verdict: adopt react-markdown in a lazy-loaded route chunk.** It is safe by default and costs nothing outside the Updates page. Notes are maintainer-authored and come with minisign-signed manifests (ADR 0015), so the XSS risk is low but not zero. Rendering Markdown to HTML in Go (goldmark) and shipping it is the alternative, but that puts HTML into the API. Area B or C's Go findings can decide that instead.
**Criticality.** Convenience.

## 13. Virtualized posters grid, tables and the log viewer

**Specs.** S20 (Library grid and table), S28 (Torrents table), S14 (P-249 live log viewer), S38 (History).
**Current plan.** TanStack Virtual is decided (ADR 0013).

**Finding (verified in React Compiler source).** `useVirtualizer` and `useWindowVirtualizer` from `@tanstack/react-virtual` **and** `useReactTable` from `@tanstack/react-table` are on React Compiler's known-incompatible list. The compiler skips memoizing components that call them, and the `incompatible-library` lint flags them. Keep each virtualizer in a small leaf component (`VirtualGrid`, `VirtualRows`) so the bail-out doesn't spread to whole pages.

| Candidate | License | Latest | Released | Notes | Verified |
|---|---|---|---|---|---|
| `@tanstack/react-virtual` (decided) | MIT | 3.14.15 | 2026-10-11 | | yes |
| `@tanstack/react-table` | MIT | 9.2.8 | 2026-10-08 | v9 is new. Not checked whether v9 leaves the compiler list | yes (npm) |
| react-virtuoso (taste list) | MIT | 4.18.16 | 2026-09-29 | Would duplicate the decided Virtual | yes |

**Verdict: adopt TanStack Virtual (decided), and build tables without a table library.** SlipStream's tables have fixed columns, server-side sort and filter via Router search params, and bulk selection, which together is about 200 lines over Virtual. A headless table engine adds a second compiler-incompatible hook for little gain.
**Criticality.** Important (the Library is the main screen).

## 14. Client state

**Specs.** S11 (theme, shell prefs, SSE status), S20 (bulk selection).
**Current plan.** Unspecified. v1 used zustand.

| Candidate | License | Latest | Released | Notes | Verified |
|---|---|---|---|---|---|
| `@tanstack/react-store` | MIT | 0.11.2 | 2026-09-29 | **Already a dependency of TanStack Router 1.170.42 and TanStack Form**, so it costs zero bytes. Still 0.x | yes |
| zustand | MIT | 5.0.15 | 2026-08-13 | Under 1 KB | yes |
| none (`useSyncExternalStore` + `localStorage`) | n/a | n/a | n/a | | n/a |

**Verdict: build (none), or use the transitive TanStack Store.** Server state lives in Query, filters and sorts in Router search params, and form state in Form. What remains is the theme, the shell layout and SSE connection status: three tiny stores. Use `@tanstack/react-store` since it is already bundled, and pin it to the Router's range. Adding zustand would be a second store idiom for the same job.
**Criticality.** Convenience.

## 15. SSE client

**Specs.** S01-S09 (the skeleton proves the live stream), S13 (running-tasks pill), S14 (live logs), S28, S33, S39, and every live page.
**Current plan.** ADR 0013: "One Server-Sent Events stream per session. Its typed events are declared in the OpenAPI document." The client is unspecified.

| Candidate | Repo | License | Latest | Released | Notes | Verified |
|---|---|---|---|---|---|---|
| Native `EventSource` | platform | n/a | n/a | n/a | Universal support. Sends same-origin cookies, auto-reconnects and resends `Last-Event-ID` | yes (BCD) |
| `@microsoft/fetch-event-source` | github.com/Azure/fetch-event-source | MIT | 2.0.1 | **2021-04-25** | Effectively abandoned | yes |
| eventsource-parser (+ `fetch`) | github.com/rexxars/eventsource-parser | MIT | 4.1.1 | 2026-09-15 | Use only if custom headers or POST are needed | yes |

Sessions are opaque cookies (P-234, S12), so the browser needs none of what fetch-based clients add (custom headers, POST). Orval generates nothing for `text/event-stream`, but it does generate Zod schemas for each event payload the document declares.

**Verdict: build** one `LiveStream` module of about 150 lines. It wraps one `EventSource`, dispatches each `event:` name to its Orval Zod schema, then updates or invalidates TanStack Query caches. It exposes connection state to the shell. Native `EventSource` hides HTTP status on failure, so when `readyState` is `CLOSED`, it re-checks the session through a Query and routes to sign-in on 401. A CI check should assert that every SSE event type in the OpenAPI document has a handler (an exhaustive `switch` over the generated union).
**Criticality.** Critical: if this breaks, every live page goes stale.

## 16. Passkeys in the browser

**Specs.** S12 (P-236, P-313 to P-315; passkeys per address under ADR 0005).
**Current plan.** ADR 0005 sets passkeys as optional, one relying party per address and never at an IP. Offline-boundaries row 22 fixes Playwright's virtual authenticator as the fake. The browser library is unspecified. v1 used `@simplewebauthn/browser`.

| Candidate | Repo | License | Latest | Released | Notes | Verified |
|---|---|---|---|---|---|---|
| Native `PublicKeyCredential.parseCreationOptionsFromJSON`, `parseRequestOptionsFromJSON`, `credential.toJSON()` | platform | n/a | n/a | n/a | **Chrome 129, Firefox 119, Safari and iOS 18.4** (Baseline 2025). `getClientCapabilities`: Chrome 133, Firefox 135, Safari 17.4 | yes (BCD) |
| `@simplewebauthn/browser` | github.com/MasterKale/SimpleWebAuthn | MIT | 14.0.0 | 2026-09-02 | Base64URL handling, error-code mapping, conditional-UI helper | yes (npm) |

**Verdict: build on the native JSON methods** (about 60 lines). Feature-detect `parseCreationOptionsFromJSON` and `window.isSecureContext`, and simply don't offer a passkey where either is missing. ADR 0005 already makes passkeys optional, with the PIN always available, so this is capability gating rather than fallback code. Devices that can't reach iOS 18.4 lose only the passkey option. Conditional UI uses `mediation: 'conditional'` after `PublicKeyCredential.isConditionalMediationAvailable()`. The library mostly polyfills what the platform now has. Its remaining value is friendlier error names, which a 20-line `DOMException` mapper covers. The server side's JSON must match the WebAuthn Level 3 JSON shape, so check this with the Go-side WebAuthn finding.
**Criticality.** Critical (security), but the surface is small. The cryptographic checks all live on the server.

## 17. PIN entry

**Specs.** S12 (4-digit PIN sign-in, re-confirm), S25 (first Admin).

| Candidate | License | Latest | Released | Verified |
|---|---|---|---|---|
| Base UI OTP Field | MIT | 1.9.0 | 2026-10-09 | yes (export present) |
| input-otp | MIT | 1.5.0 | 2026-08-18 | yes |

**Verdict: adopt Base UI OTP Field**, using `inputmode="numeric"` and `autocomplete="one-time-code"` only where that is appropriate. PRODUCT.md stresses that the PIN must be quick to type on a TV. Test it with a remote's D-pad.
**Criticality.** Important.

## 18. Test and accessibility tooling

**Specs.** S10 (Fake world harness, end-to-end suite), S11, and all UI specs.
**Current plan.** Vitest and MSW via Orval are decided (ADR 0013, offline-boundaries row 47). Playwright is implied by offline-boundaries row 22.

| Candidate | Repo | License | Latest | Released | Verified |
|---|---|---|---|---|---|
| `@playwright/test` | github.com/microsoft/playwright | Apache-2.0 | 1.64.0 | 2026-10-07 | yes |
| `@vitest/browser-playwright` (Vitest browser mode) | github.com/vitest-dev/vitest | MIT | 5.0.3 | 2026-09-30 | yes |
| msw | github.com/mswjs/msw | MIT | 3.0.3 | 2026-10-10 | yes |
| `@axe-core/playwright` | github.com/dequelabs/axe-core-npm | MPL-2.0 | 4.13.0 | 2026-08-11 | yes |
| axe-core | github.com/dequelabs/axe-core | MPL-2.0 | 4.14.0 | 2026-10-05 | yes |
| vitest-axe | MIT | 0.1.0 | 2022-10-21 (stale) | yes |

**Verdict: adopt.**
- Use Playwright for end-to-end runs against the release binary. Its WebKit project stands in for iOS Safari but does not replace a real iPhone.
- Run component tests in Vitest **browser mode** (Playwright provider) rather than jsdom. Base UI's focus, layout and pointer behaviour need a real engine, and MSW runs there too.
- Add `@axe-core/playwright` scans per route, in both themes and both shells. Skip vitest-axe, which is stale.
- axe catches only the automatable share of WCAG. Add two custom Playwright checks: 44 px minimum targets in the phone viewport (P-016, stricter than WCAG 2.5.8's 24 px), and scripted keyboard-only journeys for the main flows.
- MPL-2.0 is fine for dev-only tools.

**Criticality.** Important.

---

## Dependency picture if every verdict holds

Added beyond ADR 0013 and the boundary rows: `@base-ui/react`, `@tanstack/react-form`, `date-fns`, `react-markdown` (lazy), `@hugeicons/react` and `@hugeicons/core-free-icons`, `@fontsource-variable/geist`, plus `visx` modules only if S36 needs them. Dev only: `@playwright/test`, `@vitest/browser-playwright`, `@axe-core/playwright`.

Dropped from v1's list: vaul (unmaintained), cmdk (stale, replaced by Base UI), Sonner (replaced by Base UI Toast), react-hook-form (compiler footgun, replaced by TanStack Form), zustand (TanStack Store is already bundled), shadcn as a dependency, and `@simplewebauthn/browser` (now native).

None of these reopens an ADR. Two items touch settled non-ADR decisions: icons and fonts (offline-boundaries row 43, which the Hugeicons pick reopens) and the WebAuthn fake (row 22, compatible with native methods because Playwright's virtual authenticator runs in Chromium 129+).

## Open questions for the maintainer

1. **Primitives: Base UI, React Aria Components, or Radix?** Recommend **Base UI**. It is the only one with Drawer, Toast, OTP field and Combobox in one family, and v1 already used it.
2. **Run the shadcn CLI, or write S11's ~30 wrappers by hand while reading shadcn's Base UI versions?** Recommend **by hand**, because variant C restyles everything.
3. **Toasts: Base UI Toast or keep Sonner?** Recommend **Base UI Toast**. Sonner stays as the fallback if the primitive disappoints.
4. **Forms: TanStack Form, react-hook-form, or plain React 19 actions?** Recommend **TanStack Form**: Orval's Zod plugs in directly and it has no React Compiler incompatibility.
5. **Reordering: Motion `Reorder` plus Move up/down actions, or adopt dnd-kit?** Recommend **Motion plus actions**. The keyboard path is needed for WCAG 2.5.7 regardless.
6. **Passkeys: native JSON methods with feature detection (no passkey offer before Safari 18.4), or keep `@simplewebauthn/browser`?** Recommend **native**. PIN sign-in covers older devices under ADR 0005.
7. **View transitions: React `<ViewTransition>` only (Router's `defaultViewTransition` off), or the Router's option only?** Recommend **React's only**, and confirm it in the walking skeleton on a real iPhone, including the browser's own back swipe.
8. **Release notes: render Markdown in the browser (react-markdown, lazy) or in Go (goldmark) and ship HTML?** Recommend **the browser**, which keeps HTML out of the API.
9. **Client state: TanStack Store (already bundled via Router) or add zustand?** Recommend **TanStack Store**.
10. **Temporal: wait for Safari before adopting it, using date-fns and `Intl` meanwhile?** Recommend **yes, wait**.
11. **Component tests: Vitest browser mode (Playwright provider) instead of jsdom?** Recommend **yes**.
