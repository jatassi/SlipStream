---
name: SlipStream · House Lights (direction B)
description: A dark auditorium where the only colour comes from the Works on screen.
colors:
  velvet: "#0b0a0a"
  house: "#121010"
  raised: "#191616"
  raised-2: "#221e1d"
  overlay: "#1c1817"
  ink: "#f3ece4"
  ink-2: "rgba(243, 236, 228, 0.7)"
  ink-3: "rgba(243, 236, 228, 0.52)"
  ink-4: "rgba(243, 236, 228, 0.3)"
  on-ink: "#15110f"
  line: "rgba(243, 236, 228, 0.08)"
  line-2: "rgba(243, 236, 228, 0.14)"
  fill: "rgba(243, 236, 228, 0.07)"
  fill-2: "rgba(243, 236, 228, 0.12)"
  amber: "#e9a54b"
  exit: "#e5534b"
  glow-default: "#3a3330"
typography:
  logo:
    fontFamily: "Gloock, serif"
    fontSize: "92px"
    fontWeight: 400
    lineHeight: 0.98
    letterSpacing: "-0.01em"
  headline:
    fontFamily: "Jost, ui-sans-serif, system-ui, sans-serif"
    fontSize: "34px"
    fontWeight: 500
    lineHeight: 1.1
    letterSpacing: "-0.02em"
  title:
    fontFamily: "Jost, ui-sans-serif, system-ui, sans-serif"
    fontSize: "20px"
    fontWeight: 500
    lineHeight: 1.25
    letterSpacing: "-0.01em"
  row-title:
    fontFamily: "Jost, ui-sans-serif, system-ui, sans-serif"
    fontSize: "14.5px"
    fontWeight: 500
    lineHeight: 1.3
  nav:
    fontFamily: "Jost, ui-sans-serif, system-ui, sans-serif"
    fontSize: "15px"
    fontWeight: 450
    lineHeight: 1
  body:
    fontFamily: "Jost, ui-sans-serif, system-ui, sans-serif"
    fontSize: "15px"
    fontWeight: 400
    lineHeight: 1.5
    fontFeature: "\"kern\""
  overview:
    fontFamily: "Jost, ui-sans-serif, system-ui, sans-serif"
    fontSize: "15.5px"
    fontWeight: 400
    lineHeight: 1.5
  label:
    fontFamily: "Jost, ui-sans-serif, system-ui, sans-serif"
    fontSize: "13px"
    fontWeight: 500
    lineHeight: 1.3
    letterSpacing: "0.01em"
  caption:
    fontFamily: "Jost, ui-sans-serif, system-ui, sans-serif"
    fontSize: "12.5px"
    fontWeight: 400
    lineHeight: 1.35
    fontFeature: "\"tnum\""
  mono:
    fontFamily: "ui-monospace, SF Mono, JetBrains Mono, Menlo, monospace"
    fontSize: "12.5px"
    fontWeight: 400
    lineHeight: 1.45
rounded:
  thumb: "4px"
  poster: "6px"
  poster-lg: "8px"
  chip: "6px"
  control: "10px"
  segment: "11px"
  card: "14px"
  palette: "16px"
  sheet: "22px"
  pill: "999px"
spacing:
  phone-gut: "20px"
  desk-gut-narrow: "40px"
  desk-gut: "48px"
  desk-gut-wide: "64px"
  row: "12px"
  tile-gap: "22px"
  tile-gap-phone: "14px"
  column-gap: "64px"
  bar: "60px"
components:
  button-primary:
    backgroundColor: "{colors.ink}"
    textColor: "{colors.on-ink}"
    rounded: "{rounded.pill}"
    padding: "0 16px"
    height: "36px"
    typography: "{typography.label}"
  button-primary-hover:
    backgroundColor: "#fffaf4"
    textColor: "{colors.on-ink}"
  button-secondary:
    backgroundColor: "{colors.fill-2}"
    textColor: "{colors.ink}"
    rounded: "{rounded.pill}"
    padding: "0 16px"
    height: "36px"
  button-secondary-hover:
    backgroundColor: "rgba(243, 236, 228, 0.18)"
  button-veil:
    backgroundColor: "rgba(243, 236, 228, 0.16)"
    textColor: "{colors.ink}"
    rounded: "{rounded.pill}"
    padding: "0 22px"
    height: "46px"
  button-quiet:
    backgroundColor: "transparent"
    textColor: "{colors.ink-2}"
    rounded: "{rounded.pill}"
    padding: "0 16px"
    height: "36px"
  icon-button:
    backgroundColor: "transparent"
    textColor: "{colors.ink-2}"
    rounded: "{rounded.pill}"
    size: "36px"
  nav-item:
    backgroundColor: "transparent"
    textColor: "{colors.ink-2}"
    rounded: "{rounded.pill}"
    padding: "0 11px"
    height: "36px"
    typography: "{typography.nav}"
  nav-item-active:
    textColor: "{colors.ink}"
  search-trigger:
    backgroundColor: "{colors.fill}"
    textColor: "{colors.ink-3}"
    rounded: "{rounded.control}"
    padding: "0 8px 0 12px"
    height: "36px"
    width: "240px"
  segmented:
    backgroundColor: "{colors.fill}"
    textColor: "{colors.ink-2}"
    rounded: "{rounded.segment}"
    padding: "3px"
  segmented-selected:
    backgroundColor: "rgba(243, 236, 228, 0.13)"
    textColor: "{colors.ink}"
    rounded: "8px"
    height: "30px"
  input:
    backgroundColor: "{colors.fill}"
    textColor: "{colors.ink}"
    rounded: "{rounded.control}"
    padding: "0 12px"
    height: "38px"
  chip:
    backgroundColor: "{colors.fill}"
    textColor: "{colors.ink-2}"
    rounded: "{rounded.chip}"
    padding: "0 8px"
    height: "22px"
  target-chip-met:
    backgroundColor: "{colors.fill-2}"
    textColor: "{colors.ink}"
    rounded: "{rounded.chip}"
    padding: "0 7px"
    height: "22px"
  count:
    backgroundColor: "{colors.fill-2}"
    textColor: "{colors.ink}"
    rounded: "9px"
    padding: "0 5px"
    height: "18px"
  count-amber:
    backgroundColor: "rgba(233, 165, 75, 0.16)"
    textColor: "{colors.amber}"
  poster-tile:
    backgroundColor: "{colors.raised}"
    rounded: "{rounded.poster}"
    width: "140px"
  ticket:
    backgroundColor: "{colors.raised}"
    textColor: "{colors.ink}"
    rounded: "{rounded.card}"
    padding: "16px 18px 16px 16px"
  popover:
    backgroundColor: "{colors.overlay}"
    rounded: "{rounded.card}"
    padding: "8px"
  sheet:
    backgroundColor: "{colors.overlay}"
    rounded: "{rounded.sheet}"
    padding: "6px 14px"
  tab-bar:
    backgroundColor: "rgba(24, 20, 18, 0.74)"
    textColor: "{colors.ink-3}"
    height: "54px"
---

# Design System: SlipStream · House Lights (direction B)

<!-- PROTOTYPE DIRECTION. One of three competing directions in prototypes/ui-redesign-direction; not the project's adopted system. Extracted from src/variants/b-house-lights/ and the built dist/index.html on 2026-10-07. -->

## Overview

**Creative North Star: "House Lights"**

The library is the light source. Every surface sits in a warm velvet dark, every word is set in screen-light ink, and the only colour on any screen is spill from the art: a heavily blurred copy of the current Work's backdrop that lights the top of the page, a per-Work glow behind lit tiles and activity rows, and each Work's title set in its own poster face. The chrome owns no hue. Two signal colours exist, amber for "needs you" and red for failure, and they are spent like aisle lights: few, small, and only where the admin must act.

Focus dims the house. Hovering or focusing a poster lifts it (scale 1.03), deepens its shadow, raises its glow and reveals its caption while every sibling in the same row drops to 40% opacity. Work in progress is a print developing: a grained monochrome poster brings up its colour left to right to its acquisition fraction. The house lights rise slightly on Tonight when nothing needs the admin and stay down while anything does. Requests, for Members, are ticket stubs that tear when the Work is ready.

Density is a calm desk register (15px body, 12px row padding, 64px column gaps) over full-bleed 21:9 heroes; the phone and the Member portal are the same room at a larger, softer size.

**Key Characteristics:**
- Warm near-black ground; text is one ink at four opacities, never a second hue.
- Colour comes only from the art: spill, glow, develop, poster faces.
- Amber and red are state lights, never decoration.
- Hover and focus are theatrical: one thing lifts, the rest recede.
- Pill-shaped actions, soft-cornered posters, hairline rules instead of boxes.

## Colors

A monochrome warm-dark house whose only chroma is borrowed from the Works, plus two reserved state lights.

### Primary
- **Screen-Light Ink** (ink): every heading, active nav item, primary-button fill, focus ring, met-state chip text and filled meter. The primary action is literally the screen colour inverted, set in **Projector Dark** (on-ink) text.
- **Ink tiers** (ink-2, ink-3, ink-4): secondary text and inactive nav at 70%; captions, column heads and placeholders at 52%; disclosure chevrons only at 30%.

### Secondary
- **Aisle Amber** (amber): needs-you only. Stalled jobs, pending requests (aisle dot, the Requests nav count on a 16% amber wash, the phone More-tab dot), degraded Node dots, and the bottom-edge aisle glow under a stalled Activity row.

### Tertiary
- **Exit Red** (exit): failure only. Failed imports, offline Nodes, failed Target chips (red text on a 50% red hairline), the 1px inset ring on a failed developing poster, and the red aisle glow under a failed row.

### Neutral
- **Velvet** (velvet): the page ground on every shell; also the base of every hero shade gradient (at 50–84% alpha).
- **House** (house): defined as the house layer but in the build it only beds ticket-stub art. See drift note below.
- **Raised** (raised): posters before they load, More-screen groups, Member season pickers, ticket bodies.
- **Raised Light** (raised-2): ticket hover, avatar fill.
- **Overlay** (overlay): popovers, menus and the phone action sheet.
- **Line / Line Strong** (line, line-2): hairline row rules (8%) and stronger dividers, inactive chip outlines, ticket perforation (14%).
- **Fill / Fill Strong** (fill, fill-2): resting and hover fills for search, segmented controls, secondary buttons and fields (7% and 12%).
- **Glow Default** (glow-default): the fallback when no Work tint is in scope. At runtime every tinted element receives its own glow from the Work's palette (a 60/40 mix of the poster's mid ground and accent), plus a 16% version for ticket washes.

### Named Rules
**The Art Is The Palette Rule.** No chrome element carries a hue of its own. Any colour beyond ink, amber and red must come from a Work's palette in scope (spill, glow, develop, poster face).

**The Aisle Light Rule.** Amber means "needs you" and red means "failed or offline". Neither marks selection, brand, progress or emphasis. Selection is ink; progress is the Work's own glow.

## Typography

**Display Font:** each Work's own poster face (Gloock, Bebas Neue, Italiana, Bodoni Moda, Unbounded, Big Shoulders Display, Limelight, Archivo Black, Poiret One), with serif fallback
**Body Font:** Jost (with ui-sans-serif, system-ui)
**Label/Mono Font:** ui-monospace stack, for release names and file paths only

**Character:** a quiet geometric sans for everything the house says, against titles that arrive in the voice of their own posters. The UI never competes with the title.

### Hierarchy
- **Logo** (poster face weight and tracking, fitted 15–92px, 0.98 line-height): a Work's title on its hero, Work page, ticket and phone stage. Size is computed per Work to a max width in one or two balanced lines (desk hero 40–88px at 600px wide; Work page up to 92px; tickets 15–46px). The frontmatter entry is the default face at the desk ceiling.
- **Headline** (500, 34px, 1.1, -0.02em): page titles (Activity, Requests, Library). The phone large title uses the same size at 600.
- **Title** (500, 20px, 1.25, -0.01em): section heads (Developing, Needs you, Coming soon, Targets and Editions). Phone section heads use 600.
- **Row title** (500, 14.5px, 1.3): Activity rows, aisle items, episode titles, palette results.
- **Nav** (450, 15px): house-bar and portal links.
- **Body** (400, 15px, 1.5; 16px on phone): default text.
- **Overview** (400, 15.5px, 1.5, max 54–62ch, clamped to 2–3 lines): Work synopses on heroes.
- **Label** (500, 13px, 0.01em, ink-2): sub-section heads in Work side columns; never uppercase.
- **Caption** (400, 12.5px, tabular figures, ink-3): tile subtitles, column heads, timestamps.
- **Mono** (12.5px; 11.5px in Activity rows): release names and file paths.

### Named Rules
**The Two Voices Rule.** Jost speaks for the house; the poster face speaks for the Work. A poster face never sets UI text, and Jost never sets a Work's title on its own hero.

**The Tabular Rule.** Every live number (rates, percentages, counts, times) uses tabular figures so ticking values do not jitter.

## Layout

The desk is a full-bleed stage with content inset by a container-responsive gutter (40px under 1360px wide, 48px default, 64px over 1700px). A 60px house bar floats over the art: clear with a velvet fade while the hero is under it, then solid (80% dark, 22px blur) once the page scrolls.

Tonight is a 21:9 hero (height clamps 440–900px against the viewport) whose copy sits bottom-left over three directional shades, with the Needs-you aisle in a 340–400px right column. Below it, a two-column lower deck (fluid main, 320–380px side) pulls up 36px into the hero's fade. Section columns are separated by 64px; poster rows scroll horizontally at 132–150px per tile with 22px gaps. Work pages repeat the hero, then sink a 228px poster 104px below the stage edge beside the logo and a 260–300px state column.

Lists are rule-separated grids, never cards: Activity rows (42px poster, then fixed columns), Library table rows (64px min, 8px radius hover fill), Requests (64px poster), Nodes (four columns, 32px gaps). Row padding is 10–14px with a 1px hairline between rows.

The phone uses a 20px gutter, a 44px nav bar that gains its material as the large title scrolls under it, a 548px Tonight hero, 3-up poster grids, 116px snap-scrolling rows and a 54px blurred tab bar. Every small control keeps a 44px hit target via an invisible pseudo-element. The Member portal is the phone grammar at desk scale: a 1180px centred column with a 48px gutter, 64px header, tickets in a 460px auto-fill grid.

**The Stage And Stalls Rule.** Art occupies the top of the page full-bleed and fades out by mask; content begins in that fade, not below a hard edge.

## Elevation & Depth

Depth comes from light, not from stacked surfaces. The page is one velvet plane; what reads as depth is the spill (a 72px-blurred, 1.45-saturated copy of the backdrop at 56% opacity, 36% on Library, Series and Search, masked to fade out by 96% of its height), per-Work glows behind tiles and rows, and tonal raised fills. Shadows exist only on things that physically sit above the stage: posters, overlays, sheets and tickets.

### Shadow Vocabulary
- **Poster rest** (`box-shadow: 0 1px 2px rgba(0,0,0,0.5), 0 10px 26px -16px rgba(0,0,0,0.9)`): every poster tile.
- **Poster lifted** (`box-shadow: 0 2px 4px rgba(0,0,0,0.5), 0 26px 50px -18px rgba(0,0,0,0.95)`): hovered or focused tile.
- **Work poster** (`box-shadow: 0 2px 6px rgba(0,0,0,0.5), 0 40px 80px -24px rgba(0,0,0,0.95)`): the large poster on a Work page.
- **Popover** (`box-shadow: 0 0 0 1px rgba(243,236,228,0.08), 0 24px 60px -12px rgba(0,0,0,0.75), 0 4px 12px rgba(0,0,0,0.35)`): menus, notifications, tasks.
- **Ticket** (`filter: drop-shadow(0 14px 22px rgba(0,0,0,0.4))`): drop-shadow so the notched silhouette casts correctly.
- **Glass bar** (`backdrop-filter: blur(22–24px) saturate(1.3–1.7)` over 74–80% dark): solid house bar, phone nav bar, tab bar.

### Named Rules
**The Light Not Lift Rule.** Content on the stage is never boxed or raised to separate it; use a hairline or a glow. Shadows belong to objects (posters, tickets, overlays).

**The House Dims Rule.** When one thing is under the pointer or focus, its siblings in the same group drop to 40% (Coming soon rows to 50%). Hover dimming belongs behind a hover-and-fine-pointer query; focus dims on every device.

## Shapes

Actions are pills (999px): buttons, nav items, icon buttons, the back control, the switch. Posters keep a soft 6px corner (8px at Work-page size, 3–5px at thumbnail size) with a 6% inner hairline so dark art holds its edge. Fields, search and selects use a 10px corner; segmented controls nest an 8px selection inside an 11px track. Overlays round to 14px, the command palette to 16px, the phone sheet to 22px on its top edge. Ticket stubs are the one cut shape: a 14px card whose seam is notched by 9px circular masks top and bottom with a 1.5×9px dashed perforation, the stub tilting 4 degrees off when it tears.

## Components

### Buttons
Soft, luminous pills; the primary is a lit screen, everything else a veil of ink.
- **Shape:** full pill; 36px default, 30px small, 46px large and block.
- **Primary:** ink fill with on-ink text, 500 14px. One per decision.
- **Secondary:** 12% ink fill; hover 18%.
- **Veil:** 16% ink fill over art (hero Details); hover 24%; an opaque #2a2422 under reduced transparency.
- **Quiet:** transparent ink-2 text; hover gains the 7% fill and full ink.
- **Press:** scale 0.97 over 140ms on the house ease. **Busy:** label goes transparent and a 16px spinner turns at 700ms.
- **Disabled:** 42% opacity.

### Chips
- **Status chip:** 22px, 6px corner, 7% fill, ink-2 500 12px.
- **Target chips:** outlined in line-2 at rest (ink-3 text); met is filled 12% with ink text; upgradable 7% fill; downloading or deriving takes a 1px ring in the Work's glow; failed takes red text on a 50% red ring.
- **Count:** 18px pill with tabular figures; the amber variant (16% amber wash, amber text) only on Requests when requests wait.
- **Filter chips (phone):** 34px pills; selected inverts to ink fill with on-ink text.

### Cards / Containers
There are almost no cards. Lists are hairline-ruled rows. The exceptions are the ticket stub (below), the phone More groups and Member season picker (raised, 14–16px corners) and the phone Work state panel (5% ink fill, 14px corner).

### Inputs / Fields
- **Style:** 38px, 10px corner, 7% fill, transparent 1px border, 15px ink text, ink-3 placeholder.
- **Search fields:** phone and portal search sit at 42–50px with a 12–14px corner and take a 2px ink ring at 35% on focus.
- **Selects:** 36px, 10px corner, 7% fill (12% on hover), custom chevron.
- **Switch:** 34×20 track at 20% ink; on, the track fills ink and the knob turns on-ink and slides 14px on the house ease.
- **Checkbox:** 22px, 7px corner, 1.5px ink-3 inset ring; checked fills ink.

### Navigation
- **House bar (desk):** wordmark (a 20×16 screen-and-seats glyph plus "SlipStream" at 500 18px), then Tonight, Movies, Series, Requests, Activity, Nodes as 36px pills in ink-2. Hover adds the 7% fill; current turns ink and an 18×2px ink underline glides beneath it over 340ms. Right side: a 240px search trigger with a kbd hint, tasks, bell (ink dot for unread), avatar.
- **Phone:** 44px nav bar with a back pill that is a 42% velvet veil over art and loses it as the bar fills; five-tab bar at 10.5px labels, ink-3 inactive and ink current, an ink count badge, and an amber dot only for needs-you.
- **Portal:** wordmark, search, Library and Requests links, bell, account; same pill links at 450 15px.

### Poster Tile (signature)
A 2:3 poster with a 6px corner and the rest shadow. Its caption (500 13.5px title, 12.5px ink-3 subtitle) is hidden until the tile is live, hovered or focused on desk; developing and live tiles always show it. Hover or focus scales the tile to 1.03, raises a 30px-blurred ellipse of the Work's glow behind it to 55%, deepens the shadow and dims siblings to 40%. Focus adds a 2px ink outline offset 3px.

### Develop (signature)
A monochrome print (grayscale, 0.9 brightness, 0.86 contrast) under a 42% overlay fractal grain, with the colour copy masked in from the left to the acquisition fraction across a 12% soft edge. Download owns 0–90% of the fraction, Transfer 90–98%, import the last 2%, so colour never runs backwards. Queued and failed show no colour; stalled colour desaturates to 55%; failed adds a 1px exit-red inner ring.

### Ticket Stub (signature)
A Member's request: poster, title in its poster face, who asked, a three-segment progress (Requested, Approved, Developing; 2px bars, current segment label in ink 500) and a stub reading "Showing / Not yet". Both halves are washed in the Work's 16% glow. When the request becomes available the stub tears: a spring (0.55s response, 0.62 damping) translates it 7px by 4px and rotates it 4 degrees, and the stub face swaps to "Ready to watch". Denied tickets fall to 60% with a grayscale poster.

### Aisle (Needs you)
A ruled list: 7px state dot (amber or red), 500 14.5px title, 13px ink-2 reason, and a quiet action at the right (stacked under the text on phone). Activity rows that need the admin carry the same light as a 1px gradient underline and a radial bloom in amber or red along their bottom edge.

### Overlays and Motion
Popovers enter from scale 0.97 and -4px over 180ms; toasts rise 8px over 260ms; the command palette opens instantly with no animation over a 60% scrim. Opening a Work runs a view transition: the old page dims to 30% brightness over 300ms ("house down"), the new one fades up over 520ms after 180ms, and the poster morphs between positions over 560ms on the drawer curve. The house curve is cubic-bezier(0.23, 1, 0.32, 1); phone pushes, swipes and sheets use sampled springs (0.36–0.5s response). Reduced motion removes tile scaling, the hero copy blur-in and the stub's vertical drop.

## Do's and Don'ts

### Do:
- **Do** take every hue from the Work in scope: spill behind the page top, the Work's glow behind its tile or row, its poster face for its title.
- **Do** keep amber for needs-you and red for failed or offline, and keep them small: a 7–8px dot, a count wash, a row-edge glow.
- **Do** separate list content with 1px hairlines at 8% ink, not boxes or cards.
- **Do** make the single primary action an ink pill with on-ink text; everything else is a veil, fill or quiet pill.
- **Do** lift one thing and dim its siblings to 40% on hover and focus, and show a 2px ink focus ring on every interactive element.
- **Do** use tabular figures for every live number.
- **Do** keep a 44px hit target on phone for every small control.

### Don't:
- **Don't** introduce a brand or accent hue in the chrome; the art is the palette.
- **Don't** use amber or red for selection, progress, emphasis or decoration.
- **Don't** set UI text in a poster face, or a Work's hero title in Jost.
- **Don't** box stage content in raised cards or shadows; shadows belong to posters, tickets and overlays.
- **Don't** uppercase or letter-space UI labels; uppercase appears only where a Work's poster face is uppercase.
- **Don't** animate the command palette; it opens instantly.

<!-- Gaps between build and direction contract, recorded rather than repaired:
- House layer (#121010) is defined but used only to bed ticket-stub art; pages sit on velvet directly.
- The contract's amber budget of three per screen is not enforced in code; it holds in the reviewed screens by data, not by rule.
- Overlay surfaces are hard-coded near-duplicates outside the token set: palette #1a1615, popover and sheet #1c1817 (recorded as overlay), toast #231e1c.
- The phone register raises headline and section weights to 600 where the desk uses 500.
- Coming soon's sibling dimming (to 50%) is not behind the hover-and-fine-pointer query, so it can fire on a touch tap.
- Per-Work tint sets --glow-a and --glow-b, which no style reads.
- The desk text field's focus is only a border shift to 14% ink plus a 12% fill, weaker than the 2px ink ring used elsewhere; not recorded as a rule.
-->
