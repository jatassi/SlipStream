---
name: SlipStream
description: A self-hosted media manager whose pages take their colour from the Work in focus.
colors:
  page: "hsl(214 7% 5%)"
  glass: "hsl(214 7.7% 8% / 0.56)"
  glass-raised: "hsl(214 8.4% 13% / 0.6)"
  chrome: "hsl(214 7% 6% / 0.64)"
  popover: "hsl(214 7.7% 12% / 0.84)"
  sheet: "hsl(214 5.6% 10% / 0.88)"
  well: "hsl(214 30% 4% / 0.5)"
  fill: "hsl(214 40% 92% / 0.1)"
  fill-hover: "hsl(214 40% 92% / 0.16)"
  selected: "hsl(214 30% 92% / 0.22)"
  live-fill: "hsl(214 50% 26% / 0.7)"
  hairline: "hsl(214 40% 88% / 0.11)"
  hairline-strong: "hsl(214 40% 88% / 0.18)"
  ink-1: "hsl(214 25% 96%)"
  ink-2: "hsl(214 18% 83%)"
  ink-3: "hsl(214 14% 71%)"
  ink-4: "hsl(214 12% 60%)"
  glow: "hsl(214 95% 72%)"
  halo: "hsl(214 80% 58% / 0.5)"
  solid: "#f6f4f0"
  on-solid: "#111111"
  fail: "#ff6b61"
  fail-ink: "#ffb0a9"
  fail-text: "#fff3f1"
  fail-dim: "rgba(255, 226, 223, 0.74)"
  fail-deep: "#2a0603"
  fail-card: "hsl(3 48% 11% / 0.97)"
  fail-line: "rgba(255, 107, 97, 0.46)"
  warn: "#ffb547"
  warn-ink: "#ffd391"
  warn-text: "#fff6e9"
  warn-dim: "rgba(255, 238, 212, 0.76)"
  warn-deep: "#2b1a00"
  warn-card: "hsl(37 50% 9% / 0.97)"
  warn-line: "rgba(255, 181, 71, 0.36)"
  ok: "#5fd394"
  ok-deep: "#03230f"
typography:
  display:
    fontFamily: "Geist, 'Helvetica Neue', sans-serif"
    fontSize: "56px"
    fontWeight: 700
    lineHeight: 1.02
    letterSpacing: "-0.035em"
  title-large:
    fontFamily: "Geist, 'Helvetica Neue', sans-serif"
    fontSize: "34px"
    fontWeight: 700
    lineHeight: 1.15
    letterSpacing: "-0.035em"
  numeral:
    fontFamily: "Geist, 'Helvetica Neue', sans-serif"
    fontSize: "34px"
    fontWeight: 600
    lineHeight: 1
    letterSpacing: "-0.04em"
    fontFeature: "'tnum'"
  sentence:
    fontFamily: "Geist, 'Helvetica Neue', sans-serif"
    fontSize: "24px"
    fontWeight: 600
    lineHeight: 1.25
    letterSpacing: "-0.025em"
  title:
    fontFamily: "Geist, 'Helvetica Neue', sans-serif"
    fontSize: "20px"
    fontWeight: 700
    lineHeight: 1.2
    letterSpacing: "-0.025em"
  sentence-touch:
    fontFamily: "Geist, 'Helvetica Neue', sans-serif"
    fontSize: "17px"
    fontWeight: 600
    lineHeight: 1.35
    letterSpacing: "-0.015em"
  card-title:
    fontFamily: "Geist, 'Helvetica Neue', sans-serif"
    fontSize: "15px"
    fontWeight: 600
    lineHeight: 1.3
    letterSpacing: "-0.01em"
  body-touch:
    fontFamily: "Geist, 'Helvetica Neue', sans-serif"
    fontSize: "15px"
    fontWeight: 400
    lineHeight: 1.45
  input-touch:
    fontFamily: "Geist, 'Helvetica Neue', sans-serif"
    fontSize: "17px"
    fontWeight: 400
    lineHeight: 1.3
  heading:
    fontFamily: "Geist, 'Helvetica Neue', sans-serif"
    fontSize: "14px"
    fontWeight: 600
    lineHeight: 1.35
    letterSpacing: "-0.01em"
  body:
    fontFamily: "Geist, 'Helvetica Neue', sans-serif"
    fontSize: "13px"
    fontWeight: 400
    lineHeight: 1.45
  button:
    fontFamily: "Geist, 'Helvetica Neue', sans-serif"
    fontSize: "13px"
    fontWeight: 600
    lineHeight: 1
  meta:
    fontFamily: "Geist, 'Helvetica Neue', sans-serif"
    fontSize: "12px"
    fontWeight: 400
    lineHeight: 1.4
  chip:
    fontFamily: "Geist, 'Helvetica Neue', sans-serif"
    fontSize: "12px"
    fontWeight: 600
    lineHeight: 1
  label:
    fontFamily: "Geist, 'Helvetica Neue', sans-serif"
    fontSize: "11px"
    fontWeight: 500
    lineHeight: 1.3
  label-caps:
    fontFamily: "Geist, 'Helvetica Neue', sans-serif"
    fontSize: "11px"
    fontWeight: 600
    lineHeight: 1.3
    letterSpacing: "0.06em"
  data:
    fontFamily: "'Geist Mono', ui-monospace, monospace"
    fontSize: "0.92em"
    fontWeight: 400
    lineHeight: 1.45
    letterSpacing: "-0.01em"
    fontFeature: "'tnum'"
rounded:
  chip: "6px"
  item: "8px"
  card: "14px"
  dialog: "20px"
  sheet: "24px"
  capsule: "999px"
spacing:
  "0.5": "2px"
  "1": "4px"
  "2": "8px"
  "3": "12px"
  "4": "16px"
  "5": "20px"
  "6": "24px"
  "7": "28px"
  "8": "32px"
components:
  button:
    backgroundColor: "{colors.fill}"
    textColor: "{colors.ink-1}"
    typography: "{typography.button}"
    rounded: "{rounded.capsule}"
    height: "32px"
    padding: "0 14px"
  button-hover:
    backgroundColor: "{colors.fill-hover}"
  button-primary:
    backgroundColor: "{colors.solid}"
    textColor: "{colors.on-solid}"
    rounded: "{rounded.capsule}"
    height: "32px"
  button-ghost:
    backgroundColor: "transparent"
    textColor: "{colors.ink-2}"
  button-fail:
    backgroundColor: "{colors.fail}"
    textColor: "{colors.fail-deep}"
  button-warn:
    backgroundColor: "{colors.warn}"
    textColor: "{colors.warn-deep}"
  button-small:
    height: "28px"
    padding: "0 12px"
  button-touch:
    typography: "{typography.body-touch}"
    height: "44px"
    padding: "0 18px"
  button-touch-primary:
    backgroundColor: "{colors.solid}"
    textColor: "{colors.on-solid}"
    height: "50px"
  icon-button:
    textColor: "{colors.ink-2}"
    rounded: "{rounded.capsule}"
    size: "32px"
  icon-button-touch:
    size: "44px"
  segmented-track:
    backgroundColor: "{colors.fill}"
    rounded: "{rounded.capsule}"
    padding: "2px"
  segmented-option:
    textColor: "{colors.ink-2}"
    rounded: "{rounded.capsule}"
    height: "28px"
    padding: "0 12px"
  segmented-option-selected:
    backgroundColor: "{colors.selected}"
    textColor: "{colors.ink-1}"
  chip:
    backgroundColor: "{colors.fill}"
    textColor: "{colors.ink-2}"
    typography: "{typography.chip}"
    rounded: "{rounded.chip}"
    height: "24px"
    padding: "0 8px"
  chip-fail:
    backgroundColor: "{colors.fail}"
    textColor: "{colors.fail-deep}"
  chip-warn:
    backgroundColor: "{colors.warn}"
    textColor: "{colors.warn-deep}"
  chip-ok:
    backgroundColor: "{colors.ok}"
    textColor: "{colors.ok-deep}"
  chip-live:
    backgroundColor: "{colors.live-fill}"
    textColor: "{colors.ink-1}"
  pill-met:
    backgroundColor: "{colors.solid}"
    textColor: "{colors.on-solid}"
    rounded: "{rounded.capsule}"
    height: "28px"
    padding: "0 12px"
  failure-card:
    backgroundColor: "{colors.fail-card}"
    textColor: "{colors.fail-text}"
    rounded: "{rounded.card}"
    padding: "16px 20px 20px"
  warning-line:
    backgroundColor: "{colors.warn-card}"
    textColor: "{colors.warn-text}"
    padding: "12px 20px"
  input:
    backgroundColor: "{colors.well}"
    textColor: "{colors.ink-1}"
    rounded: "{rounded.item}"
    height: "32px"
    padding: "0 12px"
  input-touch:
    typography: "{typography.input-touch}"
    height: "48px"
  menu:
    backgroundColor: "{colors.popover}"
    textColor: "{colors.ink-1}"
    rounded: "{rounded.card}"
    padding: "4px"
  menu-item:
    rounded: "{rounded.item}"
    height: "36px"
    padding: "0 12px"
  dialog:
    backgroundColor: "{colors.popover}"
    rounded: "{rounded.dialog}"
  sheet:
    backgroundColor: "{colors.sheet}"
    rounded: "{rounded.sheet}"
    padding: "8px 16px 44px"
  toast:
    backgroundColor: "{colors.solid}"
    textColor: "{colors.on-solid}"
    rounded: "{rounded.capsule}"
    padding: "12px 16px"
---

<!-- Derived from the reference specs named in Design system and DESIGN.md (https://github.com/jatassi/SlipStream/issues/94), before any v2 code exists. Re-run /impeccable document once the walking skeleton ships, and keep `.impeccable/contrast.py` passing. -->

# Design System: SlipStream

## Overview

**Creative North Star: "The Lit Console"**

SlipStream is a console lit by the Work it is showing. A Work's artwork sits blurred behind the page, and its hue runs through every surface and every ink, so a page about *Dune: Part Two* is a page in Dune's colour. Everything else holds still against that light. Red and amber are fixed alarms, green is a fixed all-clear, and live progress is white with a glow in the Work's hue. Glass blur belongs only to things that float. Lists, tables and cards are tinted translucent sheets with hairline edges, and nothing is drawn as a heavy box.

There is one theme, and it is dark. The Admin console is dense, built from A's layout in the direction ticket: a collapsible sidebar with live counts, a top bar with a ⌘K palette, and pages that open on a status sentence. The portal shares every token and component but runs at the Touch density, with B's poster-led layout and no machinery on screen. Geist sets everything. Geist Mono appears only in single values someone copies or compares down a column.

Motion is slow where light moves and quick where controls answer. The tint crossfades over 1.1s when focus moves to another Work. Buttons, switches and menus respond in about 200ms. Sheets and dialogs arrive on a spring.

**Key Characteristics:**
- The Work in focus colours the page through a blurred ambient layer and two CSS properties, a hue and a saturation.
- Fixed signal hues: red for failure, amber for warning, green for a passed check. None of them ever takes the tint.
- Live is luminance. A running job is white with the Work's glow, and Met is solid white.
- Translucent tinted surfaces, with glass blur only on floating chrome and overlays.
- Two densities: Console for the Admin desk shell, Touch for the Admin phone shell and the whole portal.
- WCAG 2.2 AA is checked for every tint, and an ambient ceiling holds it.

## Colors

The palette is a tinted graphite whose hue follows the Work in focus, under three fixed signal hues and one solid white.

Every `hsl(214 …)` value in the frontmatter is the neutral resolution of a formula. Two registered CSS properties drive the tint: `--h` (the Work's hue, 0 to 360) and `--s` (its saturation, 18% to 70%). Inks keep their own saturation and take only the hue. Surfaces scale the ambient saturation. The canonical formulas follow, and `.impeccable/design.json` repeats them per token.

| Token | Formula |
|---|---|
| `page` | `hsl(var(--h) calc(var(--s) * .5) 5%)` |
| `glass` | `hsl(var(--h) calc(var(--s) * .55) 8% / .56)` |
| `glass-raised` | `hsl(var(--h) calc(var(--s) * .6) 13% / .6)` |
| `chrome` | `hsl(var(--h) calc(var(--s) * .5) 6% / .64)` |
| `popover` | `hsl(var(--h) calc(var(--s) * .55) 12% / .84)` |
| `sheet` | `hsl(var(--h) calc(var(--s) * .4) 10% / .88)` |
| `well` | `hsl(var(--h) 30% 4% / .5)` |
| `fill`, `fill-hover` | `hsl(var(--h) 40% 92% / .1)`, `/ .16` |
| `selected` | `hsl(var(--h) 30% 92% / .22)` |
| `live-fill` | `hsl(var(--h) 50% 26% / .7)` |
| `hairline`, `hairline-strong` | `hsl(var(--h) 40% 88% / .11)`, `/ .18` |
| `ink-1` to `ink-4` | `hsl(var(--h) 25% 96%)`, `18% 83%`, `14% 71%`, `12% 60%` |
| `glow`, `halo` | `hsl(var(--h) 95% 72%)`, `hsl(var(--h) 80% 58% / .5)` |

### Primary

- **Lit Graphite** (`page`, `glass`, `glass-raised`, `chrome`): The ground and the sheets laid on it. `page` paints under the ambient. `glass` carries tables, lists, the matrix and every card that is not a signal. `glass-raised` lifts a thing inside glass, such as a poster placeholder or a step disc. `chrome` is the sidebar and the bars.
- **Work Glow** (`glow`, `halo`, `live-fill`): The Work's own colour at full voice, and the only place it is bright. `glow` is the halo on live progress and the caret. `halo` is the outer focus ring. `live-fill` is the body of a chip that names something in flight.

### Secondary

- **Signal Red** (`fail` and its family): Failure only. `fail` is the solid chip, button, dot and failed stage segment, with `fail-deep` ink on it. `fail-card` is the near-opaque failure card, with `fail-text` for its copy, `fail-dim` for its secondary copy and `fail-ink` for its labels and for failure text set on glass. `fail-line` is the card's edge.
- **Signal Amber** (`warn` and its family): Warnings only, built the same way as red. `warn-line` is the one amber hairline on warning lines and amber rows.

### Tertiary

- **Signal Green** (`ok`, `ok-deep`): A check that passed, or a thing that is healthy. Examples are a Node online, a route hop passing, a provider key accepted, "Checked by Plex", Granted, a guardrail that allows a drop, Test connection passing and a seeding goal met. Green appears as a solid chip with `ok-deep` ink, a dot, or a check icon, always beside its words.

### Neutral

- **Ink tiers** (`ink-1` to `ink-4`): `ink-1` for primary text and values, `ink-2` for secondary text and unselected controls, `ink-3` for meta, counts and column headers. `ink-4` is not for text. It marks separators, chevrons, idle dots and disabled glyphs.
- **Fills and edges** (`fill`, `fill-hover`, `selected`, `hairline`, `hairline-strong`, `well`): Controls sit on `fill`. `selected` marks the chosen segment. Hairlines are 1px inset rings, never borders that change layout. `well` is the dark inset under inputs, code and attempt lists.
- **Solid White** (`solid`, `on-solid`): The primary action, Met, the toast and count badges. Dark `on-solid` ink always sits on it.

### The ambient

The ambient is a fixed layer behind each page, made of the Work's backdrop and a tint wash. The backdrop is blurred with `blur(64px) saturate(1.8)` and crossfades between two layers. The wash is two radial gradients at the Work's hue (26% and 22% lightness, alpha .55 and .45) over a vertical darkening at 4% to 5% lightness.

A sampler draws the poster onto a 24 × 36 canvas. It skips pixels with less than 4% chroma and weights each pixel by `sat² × (1 − |l − .5| × 1.6)`, which favours saturated mid-tones. It takes the circular mean of the hues, and sets `--s` to the weighted mean saturation × 80, clamped to 18% to 70%. A poster with no colourful pixels gives 220 / 12%.

Who sets the ambient:
- **A Work's page.** That Work.
- **Admin Home.** The most important job, meaning the job closest to Available, with a Member's request breaking ties. The pipeline sorts in that order, so the top row and the tint agree. On desk, hovering or focusing a pipeline row moves the tint to its Work after 90ms, and leaving returns it after 450ms. On phone, the topmost job in the middle band of the viewport sets it.
- **Portal Home.** The poster under the pointer or focus on desk, with the rest of the wall dimming. At rest, the first Work of the first shelf.
- **Any screen with no Work.** Nodes, Remote access, Settings, Providers and System use the neutral, hue 214 at 14%.
- **A failed Work never drives the tint.** Failure cards and rows carry no Work for the ambient.

### Named Rules

**The Ambient Ceiling Rule.** After its layers composite, the brightest region of the ambient stays at or below relative luminance 0.053. Text sits directly on the ambient (the status sentence, breadcrumbs, section counts), and 0.053 is the brightest ground on which `ink-3` still clears 4.5:1 at every hue. Before showing a backdrop, measure its brightest region in the 24 × 36 sample and set the image's `brightness()` to hold the ceiling. A fixed `brightness(.6)`, as the prototypes used, does not hold it: a white backdrop under that filter composites to about 0.16.

**The Fixed Signal Rule.** Red, amber and green never take the tint, and the tint never takes their meaning. A signal is a solid chip, a dot, a solid card (red and amber only) or an icon. Never a thin coloured word on a tinted ground.

**The Luminance Is Live Rule.** Running work is white with the Work's glow. Met is solid white. A hue on screen means trouble or a passed check, never motion.

**The Never Alone Rule.** Every signal colour comes with an icon and a word. Red and green are the pair most often confused, so colour alone never carries the state.

**The Contrast Proof Rule.** Change a colour token only together with `.impeccable/contrast.py`, and only if the script still passes. It sweeps 120 hues at six saturations, finds the ambient ceiling and checks 32 ink and surface pairs at it.

## Typography

**Display Font:** Geist (with Helvetica Neue, sans-serif)
**Body Font:** Geist (with Helvetica Neue, sans-serif)
**Label/Mono Font:** Geist Mono (with ui-monospace, monospace), for data only

**Character:** One grotesque at every size, tightened as it grows. Display and titles run at 700 with negative tracking. Body runs at 400 to 600 with none. Mono is a tool for values, not a voice.

### Hierarchy

The scale has ten sizes: 11, 12, 13, 14, 15, 17, 20, 24, 34 and 56px. The weights are 400, 500, 600 and 700. Load static instances, never an in-between weight such as 650.

- **Display** (700, 56px, 1.02): The portal's search hero headline and a Work's logo fallback on the portal.
- **Title large** (700, 34px, 1.15): The phone shell's large title.
- **Numeral** (600, 34px, 1, tabular): Big ordinals and headline figures, such as a provider's rank. Sans, never Mono.
- **Sentence** (600, 24px, 1.25): The desk status sentence that opens Home, Torrents, Conversions and Remote access. Also the portal's shelf titles.
- **Title** (700, 20px, 1.2): Phone section titles, the inspector title and dialog titles.
- **Sentence touch** (600, 17px, 1.35): The phone status sentence, sheet items and a recommendation card's title.
- **Card title** (600, 15px, 1.3): A failure card's subject, a list row's title in Touch, and a sheet's title.
- **Body touch** (400, 15px, 1.45): Body copy at the Touch density.
- **Input touch** (400, 17px, 1.3): Text in Touch inputs. iOS zooms the page into any input set below 16px.
- **Heading** (600, 14px, 1.35): Section and panel headings in Console, and matrix row and column heads.
- **Body** (400, 13px, 1.45): Console body, table cells and buttons (600). Prose runs to 64ch at most.
- **Button** (600, 13px, 1): Button labels in Console. Touch buttons use Body touch at 600.
- **Meta** (400, 12px, 1.4): Secondary lines under titles, timestamps in prose and legends.
- **Chip** (600, 12px): Chips, badges and ribbons. Signal chips are 700.
- **Label** (500, 11px, 1.3): Column headers, group labels and nav counts.
- **Label caps** (600, 11px, 0.06em, uppercase): Only the key column of a key and value list on a failure card. No other text is uppercase.
- **Data** (Geist Mono 400, 0.92em of its context, tabular): Values under the Copy or Compare Rule.

### Named Rules

**The Copy or Compare Rule.** Geist Mono sets a single value that someone copies or compares down a column: rates, sizes, durations, ETAs, percentages, timestamps in tables, release names, file names, file specs, paths, hashes, keys, addresses and ports. Everything else is Geist. Counts inside sentences, section counts, ordinals, step numbers and weekday labels are Geist with tabular numerals (`font-variant-numeric: tabular-nums`). Apply Mono to the value's own element, never to a container that holds words.

**The Names Are Words Rule.** Names are never Mono. A Work, a Node, a Member, a Profile and a Target are names someone gave a thing, so they read as words, even when a Node's name looks like a hostname.

## Layout

The system has two shells and two densities. The shell follows the viewport and the density follows the audience.

### Shells

- **Desk shell (768px and up).**
  - **Sidebar.** 228px wide, collapsing to a 60px rail. It collapses on its own below 1100px, and the Admin can collapse it at any width (⌘K "Toggle sidebar", remembered per browser).
  - **Nav.** Items are 32px tall with 16px icons, a live count on Activity (a white glowing dot and a number) and plain counts elsewhere. A count can carry an amber or red dot. A separator sits above Nodes, Settings and System.
  - **Top bar.** Sticky, 56px tall, laid out as `1fr minmax(320px, 440px) 1fr`. Breadcrumbs sit on the left, the ⌘K search capsule in the middle, and live rates, the bell and the page's main action on the right.
  - **Content.** Padding is 24px top, 28px sides and 120px bottom, with a 1480px maximum width and 24px between sections.
  - **Narrow desks.** At 1360px and below, tables tighten their columns, and the legend, the status sentence's right-hand totals and the top bar's rates hide.
- **Phone shell (below 768px).** A large-title header (34px title, search and avatar), sections 28px apart with 20px titles, and 16px list gutters.
  - **Admin tab bar.** Home, Library, Activity, Search and More. More opens an action sheet with the rest of the nav and its counts.
  - **Portal tab bar.** Home, Search, Requests and Account.
  - **Navigation.** Push navigation adds a 44px glass nav bar with a back label. Every menu becomes an action sheet, and every Dialog becomes a bottom sheet.
- **Safe areas.** Bars and sheets pad with `env(safe-area-inset-*)`. Never hard-code the status bar or home indicator height.
- **Portal on desk.** A 64px top nav that sits transparent over the hero and turns to glass after 24px of scroll. The gutter is `clamp(24px, 3.6vw, 56px)`, and pages run full bleed.

### Densities

- **Console.** The Admin desk shell. 13px body, 32px buttons (28px small), 44px single-line table rows and 60px two-line rows.
- **Touch.** The Admin phone shell and the whole portal on any device. 15px body, 44px minimum targets, 50px primary buttons, 48px inputs and 54px list rows.

### Rhythm

Spacing runs on a 4px grid (2, 4, 8, 12, 16, 20, 24, 28, 32), the same as Tailwind's default `--spacing`. Cards pad 16px, list rows pad 12px vertically, and sections sit 24px apart (28px on a Work page and on phone).

### Named Rules

**The Container Rule.** A component reflows by container queries, not by viewport breakpoints. The Torrents table drops to four columns at 980px and becomes stacked cards at 620px, wherever it is drawn. An open Inspector narrows the page, and the table answers without a breakpoint of its own.

## Elevation & Depth

Depth comes from translucency over the ambient, not from shadows. Surfaces at rest are flat tinted sheets edged by a 1px inset hairline. Shadows appear only under things that float, under posters, and under signal cards so they lift off the tint.

### Materials

- **Tinted translucent, no blur.** The sidebar (`chrome`), tables, lists, matrix, files, rails and board panels (`glass`), and things raised inside them (`glass-raised`).
- **Glass, blurred.** Only things that float.
  - The top bar, the phone tab bar, the nav bar and the status bar use `blur(24px) saturate(1.6)` over `chrome`.
  - Menus, popovers, the palette, the Inspector, Dialogs and sheets use `blur(30px) saturate(1.7)` over `popover` or `sheet`.
- **Solid.** Failure cards and warning lines (`fail-card`, `warn-card` at .97), and every signal chip and button. Solid white on the primary action, Met, toasts and badges.
- **Artwork.** The blurred ambient behind every page. A sharp, edge-masked backdrop sits behind the header of a Work page and of the portal's Work page.

### Shadow Vocabulary

- **Float** (`box-shadow: inset 0 1px 0 rgba(255,255,255,.1), inset 0 0 0 1px var(--hairline), 0 18px 40px rgba(0,0,0,.55)`): Menus and popovers.
- **Float large** (`box-shadow: inset 0 1px 0 rgba(255,255,255,.1), inset 0 0 0 1px var(--hairline), 0 30px 60px rgba(0,0,0,.6)`): The palette, Dialogs and the Inspector.
- **Signal lift** (`box-shadow: inset 0 0 0 1px var(--fail-line), 0 18px 40px -24px rgba(0,0,0,.8)`): Failure cards. Warning lines use `--warn-line` with no drop.
- **Poster** (`box-shadow: 0 24px 48px -16px rgba(0,0,0,.8), inset 0 0 0 1px rgba(255,255,255,.08)`): Large posters and portal poster cards.
- **Toast** (`box-shadow: 0 12px 28px rgba(0,0,0,.5)`).
- **Live glow** (`box-shadow: 0 0 10px 0 hsl(var(--h) 95% 70% / .7)`): The fill of a live bar. The live dot adds a 3px ring at `hsl(var(--h) 90% 65% / .32)`.
- **Focus** (`outline: 2px solid #fff; outline-offset: 2px; box-shadow: 0 0 0 6px var(--halo)`): Every focusable thing except menu items and the palette input, which mark focus with `fill-hover` and a 2px glow underline.

### Named Rules

**The Floating Glass Rule.** Blur means "this floats above the page". Nothing that scrolls with the content is blurred. That covers live pills and portal ribbons too.

**The Reduce Transparency Rule.** Under `prefers-reduced-transparency: reduce`, every glass and translucent surface turns opaque at the same tint and lightness, and the ambient image hides, leaving the tint wash. Under `prefers-contrast: more`, `ink-2` and `ink-3` rise to `ink-1`, and hairlines rise to `hairline-strong` at full opacity. Under `forced-colors: active`, surfaces drop to system colours, and every state keeps its icon and its word.

## Shapes

Controls are capsules and containers are softly squared. A control a finger or pointer presses (a button, segmented control, switch, search field, toast or pill) is a capsule at 999px. Containers step up in radius as they float higher: 6px chips, 8px menu items, inputs and small art, 14px cards, tables, menus and large posters, 20px Dialogs, and a 24px top edge on sheets. Marks smaller than 16px (state squares, stage segments, bars) round to a quarter of their height. Network zones on the Nodes canvas are the one dashed shape, a 20px rounded rectangle with a 1px dashed edge.

## Components

### Buttons

Capsules that press in.
- **Shape:** Capsule (999px). Console is 32px tall with 14px sides, small is 28px. Touch is 44px with 18px sides, and the primary is 50px.
- **Default:** `fill` with a `hairline-strong` ring and `ink-1` text at 600. Hover turns `fill-hover`. Press scales to .97. Both transitions run 200ms on the standard ease.
- **Primary:** Solid white with `on-solid` ink and no ring. One per view.
- **Ghost:** Transparent with `ink-2`, turning `fill` and `ink-1` on hover.
- **Fail and Warn:** Solid `fail` or `warn` with deep ink, for the one fix a signal offers. Inside a failure card, the other buttons use `rgba(255,220,216,.1)` with a `.16` ring.
- **Disabled:** 40% opacity, no hover, no press.
- **Busy:** The label stays and a 14px spinner replaces the icon.
- **Icon buttons:** 32px circles in Console and 44px in Touch, `ink-2` until hovered.

### Segmented control

An iOS-style choice of view or filter.
- **Track:** Capsule `fill` with a hairline ring and 2px padding. Options are 28px tall and 500 weight.
- **Selected:** `selected` fill with `ink-1` and a small lift shadow. Never solid white.
- **Counts:** Every option shows its count, All included, in tabular Geist.
- **Phone:** Full width with equal options, scrolling sideways when they overflow.
- **Semantics:** `role="radiogroup"`.

### Switch

A Console switch is 34 × 20 and a Touch switch is 51 × 31. The knob is white. The track is `hsl(var(--h) 20% 90% / .22)` off and `hsl(var(--h) 62% 46%)` on. The track fades over 250ms and the knob slides over 280ms. The whole labelled capsule is the switch target.

### Chips, pills and ribbons

- **Chip:** 24px, radius 6, `fill`, `ink-2`, 12px at 600. A Node chip is outlined.
- **Signal chips:** Solid `fail`, `warn` or `ok` with deep ink at 700 and a 14px icon. They name a state and its stage ("Failed at Import"), and they never take the tint.
- **Live chip:** `live-fill` with a hue ring and a white glowing dot.
- **Met pill:** Solid white capsule, 28px, with a check.
- **Request ribbons (portal):** Requested, Coming and Ready, as solid capsules at 12px and 600, with no blur. Ready is solid white, Coming is `live-fill`, and Requested is near-black at .9 with a light ring. On a poster, the ribbon sits as a full-width band at the poster's foot, with a 3px live line above it while a Coming job runs.

### Status sentence

The first line of every page that watches live work. It reads as one sentence of counts and links ("5 jobs moving · 1 failure · relay-fra degraded"). It is set in Sentence type on desk and Sentence touch on phone, over the ambient with a soft text shadow. The live part leads with a glowing white dot. Failure and warning parts are links to the cards below, in `fail-ink` and `warn`. Separators are `ink-4`. A right-hand line of the day's totals shows on wide desks.

### Stage track

A job's path from Searching to Available, one segment per stage (Searching, Downloading, Transfer, Import, Conversion, Available). Segments are 6px tall (8px on phone and in the Inspector) with 3px gaps.
- **Done:** Flat ink.
- **Current:** Holds a live fill.
- **Ahead:** A faint wash.
- **Not needed:** A hairline outline at 45% width.
- **Failed:** Solid `fail`.

Running tracks are `role="progressbar"` with a stage-and-title label. Others are `role="img"`.

### Progress

There are two kinds, and every bar maps to one.
- **Live.** A white fill rising from the Work's hue, with the live glow. A sheen sweeps it every 2.4s. Width moves linearly each second. The track is a 6px capsule at `hsl(var(--h) 60% 80% / .14)` with a hue ring. Used for downloads, Conversions, Transfers and live matrix cells.
- **Meter.** Flat ink with no glow and no sheen. Used for storage, provider allowances and seeding goals.
  - **Seeding goal.** Two lanes, ratio and seeded time, 4px each in tables and 8px in the Inspector. A 2px tick marks the tracker's minimum. It turns `warn` when a hit and run is at risk, and the goal-met state carries a green chip.
  - **Allowance.** Thin by design.
- **Spinner.** One spinner: 14px, a white arc on a faint ring, turning every 0.8s.

### Failure card

The answer to "what broke". A near-opaque red-black card (`fail-card`, radius 14, signal lift). Its header row is the signal chip with the failed stage, the time in Mono `fail-dim`, and the actions on the right (the fix as a Fail button first). Its body puts the Work and its stage track beside a key and value list. The labels are Label caps in `fail-ink` (Cause, Recovery, Release), and values run to 78ch.
- **Full:** Home's lead failures and Torrents' problems, which tile at `minmax(360px, 1fr)`.
- **Compact:** The Work page's Needs you list. It drops the stage track, and its actions stack at the bottom right.
- **Phone:** A separate 16px-radius card with its kv in one column and two actions (the fix and More).

### Signal row

The compact red or amber row for lists that can grow long: the Conversion queue's Needs you band, problems inside a table, and Needs you rows past the first few cards. It is one line of chevron, signal chip, subject, a two-line clamped verdict and the time. It expands in place to the full detail, such as each attempt with its Node, time and error in a `well`. Enter or Space toggles it, with `aria-expanded`.

### Warning line

Warnings are amber lines, never solid cards. A warning line is `warn-card` with a `warn-line` hairline, an amber chip, a bold subject, `warn-dim` detail, "since" time and one Warn button. Attached under a failure card, it is that card's last strip. On its own, it is a Signal row at rest. On phone, it is a row that opens a sheet.

### Lists and tables

Tables are glass sheets with radius 14 and hairlines between rows. Console rows are 44px for one line and 60px for two. Touch list rows are 54px, in iOS inset groups with 16px gutters. Headers are Label at `ink-3`. Hover and focus wash a row at `hsl(var(--h) 50% 70% / .1)` over 350ms. A selected row adds a 3px white glowing bar on its left edge. A whole row is one target, with arrow keys moving between rows.

### Edition × Target matrix and state squares

The Work page's grid of wanted cuts and versions. Every cell is in one of seven states, ranked failed, upgrade, missing, downloading, converting, Met and not aired. Each state has a chip and a square.
- **Met:** Solid white.
- **Upgrade:** A white ring with a centre dot.
- **Missing:** A hatch.
- **Live:** A fill rising from the bottom with the glow.
- **Failed:** Solid `fail`.
- **Not aired:** A hairline ring.

There are three layouts.
- **One Item.** A full table with a 168px first column. Each cell shows its state chip, its facts and its one action. Each Target header names its sourcing.
- **Many Items.** One collapsible glass section per level. The header holds the level name, a stacked state bar (180 × 8px), a tally and a level search. A section opens when anything in it is not Met or not aired. Rows are 36px, with 16px squares in 42 × 30 buttons grouped by Edition. Runs of not-aired Items fold into one line.
- **Needs you.** A list above the matrix with compact failure cards first, then upgrades and Not found rows. Consecutive identical Items merge into ranges.

Hovering or focusing a square opens a Popover with its detail, and clicking pins it. On phone, one Item becomes one card per Edition, and many Items keep their rows, with each row opening a sheet.

### Inspector

A non-modal panel on the right, for the detail of a selected row, hop or Node. It is `min(480px, 40vw)` wide with radius 20, glass with float large. It pushes the content left rather than covering it. It enters over 280ms from 16px right. Arrow keys move the selection and the Inspector follows. Its sections are wells at `hsl(var(--h) 40% 92% / .045)` with radius 14. On phone, the Inspector's content is a pushed page.

### Dialog

A centred modal for a decision or a guided task: provider key setup, backfill, removing torrents, Pairing and Handover. Widths are 460, 680 and 940px, with radius 20 on glass with float large, over a .5 black scrim. It enters on the spring over 360ms. Focus is trapped and returns to the opener on close. On phone, a Dialog is a bottom sheet.

### Popover and menus

Anchored to their trigger, 6px away, flipping to fit with a 12px viewport margin. A menu is at least 220px wide on glass with float, radius 14 and 4px padding. Its items are 36px with radius 8, and a danger item is `fail-ink`. Detail popovers are 340px wide. Both enter over 180ms, fading up from 8px and 98.5% scale. Hover popovers hide 160ms after the pointer leaves, and Esc closes either kind.

### Sheet

The phone's bottom sheet. It is glass at `sheet` with a 24px top edge, a 38 × 5 grabber and grouped items 52px tall with 20px icons. Cancel is a separate 52px block. It rises on the spring (`cubic-bezier(.32,.72,0,1)`, 340ms) and falls over 260ms. It scrolls when taller than the screen, and its padding includes the bottom safe area.

### Command palette

⌘K or Ctrl+K on desk. It is 640px wide at 12vh from the top, glass with float large and radius 20, with a 60px input row (17px text) and a glow underline while focused. Results are grouped (Works, then Commands), with 44px options and thumbnails. Footer hints show ↑↓, ↵ and Esc. On phone it is full screen with 56px options and a Cancel button.

### Toast

A solid white capsule at 13px and 500. It sits at the top right on desk and the top centre on phone. It enters over 220ms from 8px and stays for 2.8s. A toast says what happened and, when it matters, what happens next ("moved up from New. Estimated start 21:48 on forge").

### Guided steps

A numbered list with a 30px disc per step on a 1px connector.
- **Current step:** A white disc with dark ink and the Work glow.
- **Done step:** A check on `selected`.
- **Pending step:** `ink-2`.

Following a step's real link ticks it, and a pasted key is checked at once. The answer replaces the steps, either the provider's own reply in a failure card or a result card with a glowing check. Portal onboarding and provider key setup both use this pattern.

### Ranked list

A list whose order is its meaning, such as subtitle providers in search order. Each row has a Numeral rank (`ink-3`, the first in `ink-1`), a drag handle, the body with a one-sentence state and a meter, then actions with up and down buttons. Arrow keys move a row from its handle. A line above the list says what the order does.

### Route

Remote access draws each exposure as a lane of hops: Player, DNS, Relay, TLS, Node link, Origin and Media Server. A hop is a small `fill` tile joined by a 2px connector, and it takes the worst status of its checks. A failing or warning hop takes its signal card and a 3px connector. A hop past a failure shows "not reached", with a dashed outline and a dashed connector. A passing hop carries a green check. Selecting a hop opens its checks in the Inspector on desk and a sheet on phone. Each check shows what was observed, what was expected, what it means and its fix.

### Node canvas

Desktop Nodes is a pan and zoom diagram (30% to 200%). Its ground is a 4.5% lightness field with a 22px dot grid. Network zones (Internet, Home LAN, a remote site) are dashed rounded rectangles with a label pill. Node boxes are near-opaque tinted cards with a drop shadow, and arrows point from the Node that dials. Role chips drag between Nodes, holding ⌥ to add instead of move. While a chip is dragged, each Node shows its guardrail verdict as a chip: green allows the drop, amber means a weak probe, red blocks it, and neutral means no change. The Traffic toggle draws each running job on the links its bytes cross. Canvas controls float on glass. Below 1000px and on phone, Nodes shows the Roster list instead.

### PIN keypad

Portal sign-in and onboarding take a 4-digit PIN on a keypad of 80px circular keys in a three-column grid. A key is `hsl(var(--h) 30% 92% / .1)` with a hairline ring, and its digit is 34px Geist at 400. A press lights the key at once and fades it back over 250ms. Four 18px dots show entry. A filled dot turns `ink-1` with the Work glow and scales to 108% on the spring. A status line below counts the digits ("0 of 4"). A new PIN is entered twice, and a mismatch shakes the dots once (420ms, 9px each way) and says so in `fail-ink`.

### Emails

Emails are the one light surface, because mail clients force it. They are table layouts with no images and `color-scheme: light only`, set in Helvetica and Arial at hex and px. The page is `#f4f2ee`, the card is white with a `#e3e0da` border, and the button is a bulletproof `#1d1d1f` capsule with a VML fallback for Outlook. A callout is `#fff4dc` with `#5c3d00` text and a `#e0a020` left rule. Each email has a plain-text part built from the same blocks.

### Logo

The mark is Two Nodes: a stream S whose two ends are dots, one for each kind of Node, the Controller and the Relay. It is drawn on a 64-unit grid. The S is one stroke, `M40 14H25a9 9 0 0 0 0 18h14a9 9 0 0 1 0 18H24`, 8 units wide with round caps and joins. The dots are 5.5-unit circles at (52, 14) and (12, 50). The masters live in `brand/`.

- **In the app.** The mark is inline SVG. The S takes `ink-1` and the dots take `glow`, so the dots follow the Work in focus like every other light on the page.
- **Outside the app.** The favicon, the tray icon, the installer and anything else SlipStream cannot tint use the neutral resolution in `brand/mark.svg` (`#f2f4f7` and `#74affb`). `brand/app-icon.svg` puts that mark at 70% on a `page` tile with a 14-unit radius.
- **On light ground.** Emails and the light Windows taskbar take `brand/mark-light.svg`, with the S in `#1d1d1f`, the email button's ink, and the dots in `#1464cc`.
- **Wordmark.** The mark sits left of "SlipStream" in Geist, "Slip" at 600 and "Stream" at 400, with -0.03em tracking. The mark's 64-unit box is twice the cap height, and the gap between them is a third of that box.
- **Size and space.** The smallest size is 16px, where the stroke lands on 2px. Keep clear space of one dot's width on every side.

**The Two Dots Rule.** The dots are always the Work's glow or the neutral blue. They never take a signal colour, and the mark never stands in for a status.

### Motion

- **Easing.** The standard ease is `cubic-bezier(.16, 1, .3, 1)`, and the spring is `cubic-bezier(.32, .72, 0, 1)`.
- **Ambient.** The tint (`--h`, `--s`) and the backdrop crossfade over 1.1s on the standard ease. The incoming backdrop settles from 104% scale over 1.6s.
- **Controls.** Buttons, segments and switches respond in 200 to 280ms. Disclosure chevrons rotate over 250ms. Sections open and close at once, with no height animation.
- **Overlays.** Menus take 180ms, the palette 220ms, the Inspector 280ms, sheets 340ms in and 260ms out, and Dialogs 360ms.
- **Live.** Fills move linearly once a second, and the sheen loops every 2.4s.

**The Reduce Motion Rule.** Under `prefers-reduced-motion: reduce`, the tint and backdrop swap at once, fills jump without the sheen, and overlays appear without travel. Spinners, steps, the PIN shake and canvas animations stop. No animation is exempt.

## Do's and Don'ts

### Do:
- **Do** take every surface and ink from the formulas in Colors, so a Work's hue reaches the whole page.
- **Do** hold the ambient at or below relative luminance 0.053, measured per backdrop.
- **Do** run `python3 .impeccable/contrast.py` after changing any colour token, and keep it passing.
- **Do** lead every watching page with a status sentence, then failures with their stage, cause and recovery, then live work.
- **Do** show live work as white with the Work's glow, and Met as solid white.
- **Do** pair every signal colour with an icon and a word.
- **Do** use capsules for every control, at 32px in Console and 44px or more in Touch.
- **Do** blur only what floats: bars, menus, popovers, the palette, the Inspector, Dialogs and sheets.
- **Do** set rates, sizes, ETAs, timestamps, release names, paths and keys in Geist Mono, and everything else in Geist.
- **Do** use the neutral ambient (214 / 14%) on every screen without a Work.
- **Do** pad bars and sheets with `env(safe-area-inset-*)`.

### Don't:
- **Don't** draw a light theme. v2 has one dark theme (parity row P-012 is excepted). Emails are the only light surface.
- **Don't** tint red, amber or green, and don't let a failed Work drive the ambient.
- **Don't** use green for Met, for live work or for a solid card. Green marks a passed check or a healthy thing, as a chip, dot or icon.
- **Don't** add a fourth red. `fail-ink` covers failure text on glass, including the status sentence's link.
- **Don't** set a name, a sentence, a count in prose or a step number in Geist Mono.
- **Don't** use solid white for a selected segment or tab. Solid white means the primary action or Met.
- **Don't** blur anything that scrolls with the content, such as a live pill or a ribbon on a poster.
- **Don't** put text over a sharp backdrop unless a mask or scrim holds the ambient ceiling beneath it.
- **Don't** use `ink-4` for text.
- **Don't** use uppercase outside a failure card's key labels.
- **Don't** use B's condensed display face (Big Shoulders Display). Geist sets the portal too.
- **Don't** show Members any machinery: no Nodes, Targets, stages, releases or Conversions on the portal.
