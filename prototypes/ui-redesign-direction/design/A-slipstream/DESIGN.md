---
name: SlipStream · Direction A · Slipstream
description: A dark wind tunnel that work flows through; laminar when it moves, an eddy where it breaks.
colors:
  tunnel-ground: "#08090B"
  tunnel-raised: "#0D0F12"
  tunnel-high: "#13161A"
  tunnel-higher: "#1A1E23"
  pop-surface: "#101216"
  smoke-ink: "#E9EDF1"
  smoke-ink-2: "rgba(233, 237, 241, 0.68)"
  smoke-ink-3: "rgba(233, 237, 241, 0.5)"
  smoke-ink-4: "rgba(233, 237, 241, 0.28)"
  hairline: "rgba(233, 237, 241, 0.07)"
  hairline-2: "rgba(233, 237, 241, 0.12)"
  filament-smoke: "#D6E2EE"
  light-sheet: "#9ED8FF"
  rhodamine: "#FF6F7D"
  fluorescein: "#C6F36B"
typography:
  display-sentence:
    fontFamily: "Archivo, ui-sans-serif, system-ui, sans-serif"
    fontSize: "38px"
    fontWeight: 300
    lineHeight: 1.12
    letterSpacing: "-0.03em"
    fontVariation: "'wdth' 125"
  display-page:
    fontFamily: "Archivo, ui-sans-serif, system-ui, sans-serif"
    fontSize: "34px"
    fontWeight: 300
    lineHeight: 1.1
    letterSpacing: "-0.03em"
    fontVariation: "'wdth' 125"
  display-work:
    fontFamily: "Archivo, ui-sans-serif, system-ui, sans-serif"
    fontSize: "40px"
    fontWeight: 400
    lineHeight: 1.06
    letterSpacing: "-0.034em"
    fontVariation: "'wdth' 125"
  headline:
    fontFamily: "Archivo, ui-sans-serif, system-ui, sans-serif"
    fontSize: "30px"
    fontWeight: 600
    lineHeight: 1.08
    letterSpacing: "-0.028em"
    fontVariation: "'wdth' 112"
  title:
    fontFamily: "Archivo, ui-sans-serif, system-ui, sans-serif"
    fontSize: "15px"
    fontWeight: 600
    lineHeight: 1.45
    letterSpacing: "-0.005em"
    fontVariation: "'wdth' 112"
  body-strong:
    fontFamily: "Archivo, ui-sans-serif, system-ui, sans-serif"
    fontSize: "14px"
    fontWeight: 550
    lineHeight: 1.45
  body:
    fontFamily: "Archivo, ui-sans-serif, system-ui, sans-serif"
    fontSize: "14px"
    fontWeight: 400
    lineHeight: 1.45
  body-prose:
    fontFamily: "Archivo, ui-sans-serif, system-ui, sans-serif"
    fontSize: "15px"
    fontWeight: 400
    lineHeight: 1.55
  secondary:
    fontFamily: "Archivo, ui-sans-serif, system-ui, sans-serif"
    fontSize: "12.5px"
    fontWeight: 400
    lineHeight: 1.45
  label:
    fontFamily: "Archivo, ui-sans-serif, system-ui, sans-serif"
    fontSize: "12px"
    fontWeight: 500
    lineHeight: 1.45
  label-rake:
    fontFamily: "Archivo, ui-sans-serif, system-ui, sans-serif"
    fontSize: "11px"
    fontWeight: 500
    lineHeight: "16px"
    letterSpacing: "0.02em"
  button:
    fontFamily: "Archivo, ui-sans-serif, system-ui, sans-serif"
    fontSize: "13px"
    fontWeight: 550
    lineHeight: 1
    letterSpacing: "0.003em"
  mono:
    fontFamily: "'Martian Mono', ui-monospace, 'SF Mono', monospace"
    fontSize: "11px"
    fontWeight: 400
    letterSpacing: "-0.01em"
    fontVariation: "'wdth' 87.5"
rounded:
  poster: "5px"
  sm: "7px"
  md: "8px"
  row: "10px"
  lg: "12px"
  panel: "14px"
  card: "16px"
  sheet: "24px"
  round: "50%"
spacing:
  hair: "4px"
  xs: "8px"
  sm: "12px"
  md: "16px"
  lg: "24px"
  page: "32px"
  section: "56px"
components:
  button-primary:
    backgroundColor: "{colors.smoke-ink}"
    textColor: "{colors.tunnel-ground}"
    typography: "{typography.button}"
    rounded: "{rounded.md}"
    padding: "0 14px"
    height: "34px"
  button-primary-hover:
    backgroundColor: "#FFFFFF"
    textColor: "{colors.tunnel-ground}"
  button-secondary:
    backgroundColor: "{colors.tunnel-high}"
    textColor: "{colors.smoke-ink}"
    typography: "{typography.button}"
    rounded: "{rounded.md}"
    padding: "0 14px"
    height: "34px"
  button-secondary-hover:
    backgroundColor: "{colors.tunnel-higher}"
  button-ghost:
    backgroundColor: "transparent"
    textColor: "{colors.smoke-ink-2}"
    rounded: "{rounded.md}"
    padding: "0 14px"
    height: "34px"
  button-sm:
    rounded: "{rounded.sm}"
    padding: "0 11px"
    height: "28px"
  button-lg:
    rounded: "{rounded.row}"
    padding: "0 18px"
    height: "40px"
  icon-button:
    backgroundColor: "transparent"
    textColor: "{colors.smoke-ink-2}"
    rounded: "{rounded.md}"
    size: "32px"
  input:
    backgroundColor: "{colors.tunnel-raised}"
    textColor: "{colors.smoke-ink}"
    rounded: "{rounded.md}"
    padding: "0 12px"
    height: "34px"
  segmented:
    backgroundColor: "{colors.tunnel-raised}"
    textColor: "{colors.smoke-ink-3}"
    rounded: "9px"
    padding: "2px"
  segmented-selected:
    backgroundColor: "{colors.tunnel-high}"
    textColor: "{colors.smoke-ink}"
    rounded: "{rounded.sm}"
    height: "28px"
  chip:
    backgroundColor: "rgba(233, 237, 241, 0.05)"
    textColor: "{colors.smoke-ink-2}"
    typography: "{typography.secondary}"
    rounded: "{rounded.sm}"
    padding: "0 9px"
    height: "24px"
  chip-filter-on:
    backgroundColor: "{colors.smoke-ink}"
    textColor: "{colors.tunnel-ground}"
    rounded: "16px"
    height: "32px"
  rail:
    backgroundColor: "{colors.tunnel-raised}"
    textColor: "{colors.smoke-ink-3}"
    width: "72px"
  rail-item-current:
    backgroundColor: "{colors.tunnel-higher}"
    textColor: "{colors.smoke-ink}"
    rounded: "9px"
  top-bar:
    backgroundColor: "{colors.tunnel-ground}"
    height: "56px"
  panel:
    backgroundColor: "{colors.tunnel-raised}"
    rounded: "{rounded.panel}"
    padding: "22px 24px"
  popover:
    backgroundColor: "{colors.pop-surface}"
    rounded: "{rounded.panel}"
    padding: "6px"
  toast:
    backgroundColor: "{colors.smoke-ink}"
    textColor: "{colors.tunnel-ground}"
    rounded: "{rounded.row}"
    padding: "10px 16px"
  tunnel:
    backgroundColor: "{colors.tunnel-ground}"
    height: "340px"
---

# Design System: SlipStream · Direction A · Slipstream

## Overview

**Creative North Star: "The Wind Tunnel"**

The admin reads SlipStream as a flow visualisation. The ground is near-black tunnel graphite; everything that moves is smoke. Each live job is a filament of smoke crossing six rakes (Wanted, Searching, Downloading, Moving, Importing, Library); its speed comes from its rate, its thickness from its size, and a failure lifts it off the line into a rhodamine vortex. The rest of the interface is a quiet instrument around that tunnel: monochrome smoke-white ink at four strengths, 1px hairlines, inset rings instead of borders, and three dyes held back for flow, separation and arrival.

Type does two jobs. Archivo, opened to its widest (125%) and lightest (300), writes the system sentence ("Five jobs are moving at 68 MB/s. Two need you.") and every page head; Archivo at normal width and medium weights carries the dense rows. Martian Mono, condensed to 87.5%, is kept for releases, paths and addresses. The density is that of a control room: 12.5 to 14px rows, tabular numerals, right-aligned rates, and generous 56px gutters between the two columns that sit under the tunnel.

Motion behaves like air: a single strong ease-out (cubic-bezier(0.23, 1, 0.32, 1)), presses that compress to 0.97, rows that change text in place with a short blur-and-rise rather than re-rendering, and continuous canvas motion only where work is actually moving. Reduced motion collapses every one of these to fades.

**Key Characteristics:**
- Tunnel graphite ground in four tonal steps; no colour fills on chrome.
- Smoke-white ink at 100 / 68 / 50 / 28% carries all hierarchy.
- Three dyes, each with one meaning: light-sheet for flow and focus, rhodamine for separation, fluorescein for arrival.
- Extended light Archivo for sentences and page heads; mono only for releases and paths.
- The Tunnel and its filaments are the signature; the same painter draws a job at row scale.
- Depth from inset 1px rings and one diffuse popover shadow; never hard shadows.

## Colors

A monochrome graphite-and-smoke instrument with three laboratory dyes used sparingly.

### Primary
- **Smoke Ink** (smoke-ink): the voice of the system. Body text, titles, the filled primary button, the selected filter chip, toasts, badge pills, the filament core and the progress head. Primary emphasis is ink on graphite, never a hue.

### Secondary
- **Light Sheet** (light-sheet): the laser sheet that lights the flow. Keyboard focus ring (at 85%), input focus ring (55% ring plus 12% halo), the live head of a path strip, a hovered filament, the switch when on, the middle stroke of the mark, the caret, and the text selection (30%). On phone it also tints the back button in the navigation bar.

### Tertiary
- **Rhodamine** (rhodamine): separation only. Failed jobs, the vortex glyph and the vortex in the Tunnel, the "Two need you." clause of the system sentence (with a 32% underline), offline or failed Node states, and any row line marked as separated.
- **Fluorescein** (fluorescein): arrival only. The "1 new" count beside Arrived tonight, the arrival dye a finished filament takes in the Tunnel, and on the member portal the ready state of a request (its sentence, state line and strip head with a 14% halo).

### Neutral
- **Tunnel Ground** (tunnel-ground): page background, top bar, text on filled ink controls.
- **Tunnel Raised** (tunnel-raised): the icon rail, inputs, segmented controls, inset lists, panels and member request cards.
- **Tunnel High** (tunnel-high): secondary buttons, the selected segment, poster placeholders, phone sheet groups, hover on portal cards.
- **Tunnel Higher** (tunnel-higher): secondary-button hover and the current rail item.
- **Pop Surface** (pop-surface): popovers and the command palette.
- **Smoke Ink 2 / 3 / 4** (smoke-ink-2, smoke-ink-3, smoke-ink-4): secondary text and live values (68%), metadata, column heads and inactive navigation (50%), separators such as the "·" between metadata and the "/" after a back link (28%).
- **Hairline / Hairline 2** (hairline, hairline-2): row dividers and panel rings (7%); control rings for buttons, inputs and the command field (12%).
- **Filament Smoke** (filament-smoke): the colour of smoke on canvas and the glow around progress heads; the cooler sibling of Smoke Ink used only in drawn flow.

### Named Rules
**The One Meaning per Dye Rule.** Light-sheet means flow or focus, rhodamine means something has separated from the flow, fluorescein means something has arrived. A dye that appears without that meaning is a defect.

**The Monochrome Canon Rule.** Outside the Tunnel, chrome is graphite and smoke. The contract budgets four dye uses per screen; the build stays near that (the admin home carries the rhodamine clause, two vortex glyphs and the fluorescein "1 new"). Artwork and member avatars (hsl at 22% saturation, 26% lightness, per-person hue) are content, not chrome, and sit outside the budget.

**The Stay-Marked Rule.** Arrivals carry a 6px ink-2 dot and brighter time until seen; failures keep their rhodamine glyph until resolved.

## Typography

**Display Font:** Archivo (variable width 62 to 125, weight 100 to 900), falling back to ui-sans-serif, system-ui
**Body Font:** Archivo at 100% width
**Label/Mono Font:** Martian Mono (width 75 to 112.5, weight 300 to 500), falling back to ui-monospace, SF Mono

**Character:** A single grotesque stretched in two directions: wide and light where the system speaks a sentence, normal and medium where it lists facts. The condensed mono is a lab-label voice reserved for literal strings.

### Hierarchy
- **Display, sentence** (display-sentence): the system sentence on admin Flow. Phone 23px; member portal greeting and request sentence 29px phone, 46px wide.
- **Display, page** (display-page): every admin page head, one step below the sentence.
- **Display, Work** (display-work): the Work title over its backdrop; weight 400 rather than 300.
- **Headline** (headline): phone large titles (32px, 112% width) and member portal titles (30px phone, 50px wide). Semibold, semi-extended.
- **Title** (title): section heads ("Needs you", "Arrived tonight", "Nodes") with a 450-weight count in ink-3 at normal width. Phone section heads 18px at 108%; top-bar title 15px at 108%; Node names 17px at 110%.
- **Body strong** (body-strong): row titles in Needs you, Arrived, jobs, tables. Phone rows 15px/500.
- **Body** (body): the base 14px/1.45 setting.
- **Body prose** (body-prose): Work overviews, capped at 64ch (portal 60ch).
- **Secondary** (secondary): row detail lines, metadata, rates.
- **Label** (label): table and column heads, palette groups, footnotes. Sentence case, never uppercase.
- **Label, rake** (label-rake): Tunnel rake names with tabular counts in ink-2; rail and tab labels at 10.5px/500, 0.01em.
- **Mono** (mono): release names, file paths, addresses; 10.5px inside Tunnel cards and side panels.

### Named Rules
**The Two Stretches Rule.** Width is the hierarchy axis: 125% light for anything that reads as a sentence or a page name, 106 to 112% semibold for titles, 100% for everything in a row.

**The Literal Mono Rule.** Martian Mono is set only on strings a machine would print: releases, paths, addresses. Words about a job stay in Archivo.

**The Tabular Rule.** Every live number uses tabular numerals so rates and percentages update without jitter.

## Layout

Admin wide is a two-column shell: a 72px labelled icon rail and a main column whose first row is a 56px top bar (title left, a 300 to 420px ⌘K command field centred, live throughput and two icon buttons right). Pages pad 24px top, 32px sides, 72px bottom. Flow stacks the sentence, a full-bleed Tunnel (it bleeds the 32px page padding and gives up height, clamp(200px, 100cqh − 530px, 340px), before Nodes leave the first viewport), two equal columns with a 56px gutter (Needs you, Arrived tonight), and a six-cell Nodes strip divided by hairlines.

Work pages open on a 430px backdrop hero masked to transparent and shaded left to right, a 220px poster beside the head, then a body of main content and a 300px side list with a 56px gutter. Library grids fill columns at 126, 164 or 220px minimums with 30px by 20px gaps at medium.

Phone admin is a pushed-stack shell: 44px navigation bar that fills with blurred graphite on scroll, 16px side padding, a 5-item tab bar of 50px items, a compact 172px Tunnel bleeding to the edges, and bottom action sheets inset 8px. The member portal is one container that switches at 760px: single column with a 16px gutter below, a 64px top bar with text navigation and a 300px search field above, content capped at 1120px with 40px sides.

Spacing is a loose 4px rhythm; the reused steps are 4, 8, 12, 16, 24, 32 and 56px, with 10, 14 and 18px filling inside rows. Hover styles are gated to fine pointers; touch gets press scales instead.

## Elevation & Depth

Depth is tonal and ringed. Surfaces step up through four graphite tones, and edges are drawn with inset 1px rings of smoke at 7 or 12% rather than borders. Only floating layers cast a shadow, and it is a single diffuse, deep, low-contrast falloff with its own 9% ring. Light, not shadow, marks the live: filament glows, a 4px halo around path-strip heads, the soft glow under progress heads.

### Shadow Vocabulary
- **Pop** (`0 0 0 1px rgba(233,237,241,0.09), 0 24px 60px -18px rgba(0,0,0,0.85), 0 4px 14px -6px rgba(0,0,0,0.6)`): popovers, command palette, Tunnel cards.
- **Ring** (`inset 0 0 0 1px` hairline or hairline-2): buttons, inputs, segmented controls, panels, cards.
- **Artwork drop** (`0 40px 80px -30px rgba(0,0,0,0.95), 0 0 0 1px rgba(233,237,241,0.08)`): hero posters; tiles use `0 14px 30px -20px rgba(0,0,0,0.9)`.
- **Toast** (`0 16px 40px -14px rgba(0,0,0,0.9)`).
- **Selected segment** (`inset 0 0 0 1px` hairline-2 plus `0 1px 2px rgba(0,0,0,0.4)`).

### Named Rules
**The Ring Not Border Rule.** Controls and containers are outlined with inset rings; 1px solid borders are kept for row dividers and structural seams (rail edge, top bar).

**The Light Marks Life Rule.** Glow is reserved for something live: a moving filament, a live strip head, an arrival. Static chrome never glows.

## Shapes

Gently rounded, never pill-shaped except for true pills (switch, filter chips, badges). Posters 5px (3 to 4px at thumbnail sizes, 8px at hero size). Small buttons and chips 7px, buttons and inputs 8px, rows that highlight on hover 10px, the rail mark and portal buttons 12px, panels and popovers 14px, portal cards and the palette 16px, phone sheets 24px. Rows bleed 12px past their column so their hover fill aligns text with the column edge. Lines are 1px everywhere; drawn strokes in glyphs are 1.5px, the mark 1.9px.

## Components

### Buttons
Quiet graphite controls with one smoke-white primary.
- **Shape:** gently rounded (8px; 7px small, 10px large, 12px on portal and phone hero actions at 46px tall).
- **Primary:** filled smoke ink with ground text, no ring; hover lifts to pure white.
- **Secondary:** tunnel-high with a hairline-2 ring; hover steps to tunnel-higher and a 16% ring.
- **Ghost:** transparent, ink-2 text; hover gets a 6% smoke fill.
- **Press / Disabled / Loading:** press scales to 0.97 over 140ms; disabled sits at 38% opacity; loading blurs the label away and runs three short flow-by strokes (900ms linear, staggered).

### Chips
- **Style:** static chips are 24px, 7px radius, 5% smoke fill with a hairline ring; phone filter chips are 32px pills on tunnel-raised with a hairline-2 ring.
- **State:** a pressed filter chip inverts to filled smoke ink with ground text.

### Segmented control
A tunnel-raised track (9px, 2px inset) holding 28px segments; the selected segment lifts to tunnel-high with a ring and a 1px drop.

### Cards / Containers
- **Corner Style:** 14px panels, 16px member request cards.
- **Background:** tunnel-raised, hover to tunnel-high on fine pointers.
- **Shadow Strategy:** ring only (see Elevation).
- **Internal Padding:** 22px by 24px panels; 14px phone, 18px wide portal cards.

### Inputs / Fields
- **Style:** 34px, tunnel-raised, hairline-2 ring, 8px radius; phone search 42px on tunnel-high with 16px text.
- **Focus:** the ring turns light-sheet at 55% with a 3px light-sheet halo at 12%.

### Navigation
- **Rail:** 72px tunnel-raised column, 64px items with a 40 by 30px icon well and a 10.5px label; current item is ink on tunnel-higher with a hairline-2 ring; badges are ink pills cut out with a 2px rail-coloured ring.
- **Top bar:** 56px, back link in ink-3 followed by a 28% "/", centred command field with a `kbd` hint, live throughput in ink-2.
- **Phone:** translucent graphite tab bar with 20px blur; current tab in ink, others ink-3; the navigation title fades in when the large title scrolls away.

### The Tunnel (signature)
A canvas, 340px tall (172px compact), bounded by two hairlines that fade at the ends. Six rakes divide it, labelled at the bottom with counts. Each live job is a filament: a blurred smoke body plus 2 to 4 grain strands, speed from rate, thickness from log size, a short stub when stalled. A failed job lifts off its line and spirals into a three-armed rhodamine vortex that pins its card for resolution. Wanted items are faint wisps stacked at the first rake. Hover tints a filament toward light-sheet and shows a 300px pop card (title, line, mono release, stats, actions). The same painter draws a single job at row scale inside tables, tiles and the Work status line.

### Path strip
A streamline across stops: past segment in 78% smoke at 1.5px, future as a dotted 20% line, a 9px head that is ink at rest, light-sheet when live, an outlined ground dot when idle, a rhodamine vortex when separated, fluorescein when a member's request is ready.

### Node body
Each Node is drawn as an aerofoil in tunnel-high with a 42% ring and streamlines puffing past at a speed set by load; degraded strengthens the outline, offline turns everything dotted and stops the puffs.

### Rows that mutate in place
Needs you, Arrived, jobs and requests are patched by key. Text that changes state re-enters with a 240ms blur-and-rise (40% down, 2px blur) on the system ease; new rows slide in 6px over 280ms; leaving rows fade in 140ms.

## Do's and Don'ts

### Do:
- **Do** let ink strength carry hierarchy: smoke ink, then 68%, 50% and 28%.
- **Do** keep each dye to its one meaning: light-sheet for flow and focus, rhodamine for separation, fluorescein for arrival.
- **Do** write state as a sentence in extended light Archivo (125% width, weight 300) before showing any number.
- **Do** outline controls with inset 1px rings at 7 or 12% smoke, and save the pop shadow for floating layers.
- **Do** set releases, paths and addresses in Martian Mono at 87.5% width, and everything else in Archivo.
- **Do** draw job progress as a filament with the shared painter; keep the 2px hairline bar only for tasks that are not jobs.
- **Do** use cubic-bezier(0.23, 1, 0.32, 1) for state changes, scale presses to 0.94 to 0.97, and give every motion a fade fallback under reduced motion.

### Don't:
- **Don't** use rhodamine or fluorescein as decoration, emphasis or brand colour; they are reserved for separation and arrival.
- **Don't** fill chrome with colour; graphite tones and smoke are the only chrome surfaces.
- **Don't** add stat cards or a poster wall to Flow; the Tunnel and the sentence are its first viewport.
- **Don't** use hard or offset shadows; depth is tonal, ringed, and diffuse.
- **Don't** set labels or headings in uppercase or as small tracked kickers; labels are sentence case at 12px/500.
- **Don't** set words below 10.5px; the only smaller type is the count inside a 15px ink badge pill (9.5px, weight 650).
