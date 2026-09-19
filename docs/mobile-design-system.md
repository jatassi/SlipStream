# SlipStream Mobile — Layout & Design System

Status: **exploration**. Three shell directions are prototyped behind a picker at
`http://localhost:3000/prototypes/mobile/` (run `cd web && bun run dev`). Nothing here
is wired into the production admin app yet; the prototype surface lives in
`web/src/prototypes/mobile/` and `web/prototypes/mobile/index.html` and imports nothing
into production code.

The craft bar comes from Emil Kowalski's design-engineering skills installed in
`.agents/skills/` (`emil-design-eng`, `mobile-native`, `apple-design`, `prototype`).

## 1. Foundations (shared by every direction)

These live in `web/src/prototypes/mobile/styles/mobile-tokens.css` and layer on top of the
existing admin tokens in `web/src/index.css`. Variants never change these; they differ in
layout, navigation model and personality only. When a direction is chosen, this file moves
next to `index.css` unchanged.

### Motion

| Token | Value | Use |
| --- | --- | --- |
| `--ease-out-strong` | `cubic-bezier(0.23, 1, 0.32, 1)` | Every entrance, every press. Never `ease-in` on UI. |
| `--ease-in-out-strong` | `cubic-bezier(0.77, 0, 0.175, 1)` | Things that move on screen (segment thumb, dock highlight). |
| `--ease-drawer` | `cubic-bezier(0.32, 0.72, 0, 1)` | Sheets and drawers only. |
| `--dur-press` | `120ms` | `:active` scale. |
| `--dur-fast` | `160ms` | Bar/title crossfades, chips, tooltips. |
| `--dur-base` | `220ms` | Push transitions, list entrances, overlays. |
| `--dur-sheet` | `320ms` | Sheet open. Exit is always shorter than enter. |

Rules that fall out of the frequency table:

- Tab and scope switches happen 100+ times a session: **no transition**. Screens stay mounted and are toggled with `hidden`, so scroll position survives.
- Entrances animate once on first mount (`m-enter-*` with a 40 ms stagger, capped at 200 ms). Re-visiting a tab does not replay them.
- Only `transform` and `opacity` animate. Progress bars are the exception — their `width` transitions 700 ms linear so the value creeps rather than jumps.
- `prefers-reduced-motion` swaps slides/scales for cross-fades and removes press scale; `prefers-reduced-transparency` makes materials solid.

### Type scale (`text-m-*`)

Size, leading and tracking travel together. Tracking tightens as size grows.

| Utility | Size / leading | Tracking | Where |
| --- | --- | --- | --- |
| `text-m-display` | 34 / 40 | −0.022em | Large titles (Native), screen titles (Cinematic) |
| `text-m-heading` | 22 / 28 | −0.015em | Detail titles, action sheet buttons |
| `text-m-title` | 17 / 22 | −0.01em | Compact bar titles, card titles |
| `text-m-body` | 15 / 20 | 0 | Rows, overview copy |
| `text-m-footnote` | 13 / 18 | 0 | Secondary row text, chips |
| `text-m-caption` | 12 / 16 | +0.01em | Metadata, section headers |

Numbers that change (progress, speeds, counts) use `.m-nums` (`tabular-nums`). Release
names and paths are monospace at 11–12 px.

### Touch geometry

| Token | Value |
| --- | --- |
| `--spacing-tap` (`size-tap`, `min-h-tap`) | 44 px minimum hit area |
| `--spacing-tap-lg` | 52 px for primary actions |
| `--spacing-screen` (`px-screen`) | 16 px horizontal gutter |
| `--spacing-tab-bar` | 56 px bar height, excluding safe area |
| `--radius-sheet` / `--radius-card` / `--radius-thumb` | 20 / 14 / 6 px |

Inputs are never below 16 px (`.m-root input`), which is what stops iOS Safari zooming the
page. Controls get `touch-action: manipulation` and `user-select: none`; content never does.

### Colour

Media-type theming carries over from the admin app: movies `movie-*` (orange), series
`tv-*` (blue), mixed content `bg-media-gradient`. Text on dark uses the 400 shade, fills
and progress use 500.

Six status hues are shared so a state always looks the same in every shell:

| Status | Token |
| --- | --- |
| Available | `--status-available` (green) |
| Missing | `--status-missing` (amber) |
| Downloading | `--status-downloading` (violet) |
| Upgradable | `--status-upgradable` (cyan) |
| Unreleased | `--status-unreleased` (blue) |
| Failed | `--status-failed` (red) |

`StatusDot` / `StatusPill` / `StatusTag` (one per density) are the only components that
paint these.

### Materials and safe areas

- `.m-material` — bar material (78 % background, 20 px blur, 180 % saturate). Content scrolls under it; the hairline is a `box-shadow`, not a border, and appears only once content is beneath the bar.
- `.m-material-heavy` — sheets, overlays, command bar.
- `.m-material-chip` — floating chips over artwork (one light surface, never stacked on another).
- `.m-safe-top` / `.m-safe-bottom` pad by `--safe-top` / `--safe-bottom`, which resolve to `env(safe-area-inset-*)` on a device. The prototype phone frame overrides them (54 / 34 px) so the stage shows real insets.

### Platform baseline

`prototypes/mobile/index.html` ships the floor from the `mobile-native` skill:
`viewport-fit=cover`, `interactive-widget=resizes-content`, one `theme-color` per colour
scheme, `-webkit-tap-highlight-color: transparent`, `overscroll-behavior: none` on the
root and `contain` on every inner scroller (`.m-scroll`), hover styles gated behind
`(hover: hover) and (pointer: fine)`.

## 2. Interaction primitives

| Class | Behaviour |
| --- | --- |
| `.m-press` | `scale(0.97)` on `:active`, 120 ms ease-out. For buttons and cards. |
| `.m-press-row` | Background tint on `:active` (and hover on fine pointers). For list rows. |
| `.m-press-dim` | Opacity 0.6 on `:active`. For text links and back buttons. |
| `.m-scroll` / `.m-scroll-x` | Contained scrollers; horizontal rails snap. |
| `.m-enter-fade-up` / `.m-enter-scale` / `.m-enter-fade` | One-shot entrances; combine with `.m-stagger` on the parent. |

Feedback lands on pointer-down, never on click. Where JavaScript is needed (the sheet),
Pointer Events with `setPointerCapture` track 1:1, keep the grab offset, rubber-band past
the top, and dismiss on either distance (> 35 %) or velocity (> 0.6 px/ms) — a flick is
enough. The sheet reads its current transform on grab so it can be caught mid-animation
without a jump.

## 3. The three directions

Each is a complete mobile shell for the same key screens — dashboard/home, library,
activity (queue), detail, search, settings/system — with real interactions (pause/resume,
monitored toggle, filters, expand/collapse, remove) against shared realistic data.

### Native — *platform familiarity*

`web/src/prototypes/mobile/variants/native/`

- **Navigation**: five-tab translucent bottom bar (Dashboard, Library, Activity, Search, More). Detail and settings push in from the right (260 ms), the screen behind slides 28 % and dims; pop is faster (200 ms). Back button carries the previous screen's name.
- **Header**: large 34 px title in the content flow; a compact 17 px title and material background fade in once the title scrolls under the bar.
- **Lists**: inset grouped lists (`Group` / `Row`), 44 px rows, hairline dividers, coloured icon tiles, chevrons.
- **Library**: segmented control with sliding thumb, three-column poster grid with title, year, quality and status dot under each poster.
- **Activity**: rows with progress, release name in mono, trailing pause/resume; tapping a row opens an iOS-style action sheet (Pause/Resume, Remove, Remove & blocklist, Cancel).
- **Detail**: backdrop hero with overlapping poster, three pill actions (Search, Auto, Monitored toggle), Overview / File / Details groups.

Wins when the app is a daily-use utility and users should never think about the UI. Costs
memorability and shows less artwork.

### Cinematic — *immersion*

`web/src/prototypes/mobile/variants/cinematic/`

- **Navigation**: floating glass dock (Now, Library, Activity, Search) with a sliding white highlight and a media-gradient glow; content is edge-to-edge underneath.
- **Now**: horizontal scroll-snap hero cards for active downloads (4:5 backdrop, display title, quality tokens, glowing progress), then "Recently added" and "Missing" poster rails, health as a floating chip.
- **Library**: two-column large posters with on-art title treatments, sticky material filter chips (All / Movies / Series / Missing / Downloading), status pills only where the state is not "available".
- **Activity**: full-bleed backdrop cards with glass pause/remove buttons.
- **Detail**: a draggable bottom sheet — backdrop header is the drag surface, 1:1 tracking, rubber-band at the top, velocity dismissal, scrim tied to drag progress.
- **Search**: material overlay that scales in (blur + opacity + scale together), autofocused input, suggested titles before typing.

Wins for a poster-forward product people open to enjoy their collection. Costs vertical
density (fewer items per screen) and GPU cost from blur on older phones.

### Console — *density and speed*

`web/src/prototypes/mobile/variants/console/`

- **Navigation**: thumb-reach command bar pinned to the bottom — scrollable scope chips (Board, Library, Activity, Missing, System) with live counts, plus a 16 px filter field that narrows whatever scope is active. On the Board it searches the whole library.
- **Header**: a 40 px status strip — wordmark, health LED with count, download LED with `n↓ speed`, WebSocket LED.
- **Board**: four stat tiles, health issues, the first four queue rows, recent history, scheduled tasks.
- **Rows**: 44 px `DenseRow` with a 28 px thumb, monospace tags for quality/event, tabular numbers right-aligned, a 2 px progress line along the bottom edge of queue rows.
- **Activity**: inline pause/resume; tapping a row expands it instantly (no animation) to show the release, client and Remove / Blocklist actions.
- **Detail**: full-screen replace, no transition; compact header, key/value file table, inline "In queue" rows.

Wins for operators checking in many times a day who want everything one tap away. Costs
warmth and approachability; artwork is reduced to a thumbnail.

## 4. Prototype harness

- Isolated Vite page at `web/prototypes/mobile/index.html` → `web/src/prototypes/mobile/main.tsx`. Not part of `vite build` output; `tsc -b` and ESLint still cover it.
- Renders one variant at a time inside a 390 × 844 phone frame (status bar, Dynamic Island, home indicator) on desktop, or full-viewport on a real phone (`max-width: 520px`).
- Picker chrome is copied verbatim from the `prototype` skill's `PICKER.md` and positioned top so it never covers a tab bar or dock. Keys: `1–3`, `←` / `→`, `R` to re-mount. Selection persists in `?v=`.
- Shared mock data (`shared/mock-data.ts`) and a reducer-backed store (`shared/state.tsx`) that ticks download progress every second so the queue is alive.

## 5. Promotion checklist

When a direction is picked:

1. Move `styles/mobile-tokens.css` beside `web/src/index.css` and import it there.
2. Add the platform baseline (`viewport-fit=cover`, `theme-color` per scheme) to `web/index.html`.
3. Port the chosen shell into `web/src/components/layout/` behind a `(max-width: 768px)` breakpoint or a standalone `/m` route, replacing mock data with the existing TanStack Query hooks (`useQueue`, `useHistory`, `useStorage`, module registries for Library).
4. Poster art uses `PosterImage` / `BackdropImage`; the gradient stand-ins in `shared/poster.tsx` are deleted.
5. Delete `web/prototypes/` and `web/src/prototypes/`.
6. Verify on hardware: sticky hover, tap delay, safe areas, keyboard, and the sheet gesture cannot be judged in device emulation.
