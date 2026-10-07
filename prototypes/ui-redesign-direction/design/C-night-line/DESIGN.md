---
name: SlipStream · Night Line (direction C)
description: A midnight transit diagram for a media network; Role Groups are lines, Nodes are stations, every Work rides a route.
colors:
  midnight-enamel: "#0A0F1E"
  night-deck: "#0E1527"
  raised-deck: "#131C33"
  lit-deck: "#1B2643"
  overlay-deck: "#141D35"
  porcelain-ink: "#EDF1F7"
  platform-grey: "#B4BED1"
  timetable-grey: "#8390A8"
  conduit-grey: "#4E5A73"
  rule: "rgba(237, 241, 247, 0.075)"
  rule-strong: "rgba(237, 241, 247, 0.13)"
  acquire-red: "#F0524F"
  library-blue: "#3D7BFF"
  process-amber: "#F4A62A"
  deliver-green: "#22C08A"
  notify-violet: "#A99BFF"
  delay-amber: "#F6B13A"
  failure-red: "#FF4A5B"
  suspended-slate: "#66728C"
typography:
  page-title:
    fontFamily: "'Barlow Condensed', 'Barlow', ui-sans-serif, sans-serif"
    fontSize: "27px"
    fontWeight: 600
    lineHeight: 1
    letterSpacing: "0.06em"
  phone-large-title:
    fontFamily: "'Barlow Condensed', 'Barlow', ui-sans-serif, sans-serif"
    fontSize: "36px"
    fontWeight: 600
    lineHeight: 1.05
    letterSpacing: "0.05em"
  work-title:
    fontFamily: "'Barlow', ui-sans-serif, system-ui, sans-serif"
    fontSize: "36px"
    fontWeight: 600
    lineHeight: 1.04
    letterSpacing: "-0.018em"
  portal-headline:
    fontFamily: "'Barlow', ui-sans-serif, system-ui, sans-serif"
    fontSize: "30px"
    fontWeight: 600
    lineHeight: 1.1
    letterSpacing: "-0.016em"
  portal-section:
    fontFamily: "'Barlow', ui-sans-serif, system-ui, sans-serif"
    fontSize: "20px"
    fontWeight: 600
    letterSpacing: "-0.005em"
  status-line:
    fontFamily: "'Barlow', ui-sans-serif, system-ui, sans-serif"
    fontSize: "clamp(15.5px, 1.25vw, 18px)"
    fontWeight: 500
    lineHeight: 1.35
    letterSpacing: "-0.003em"
  sign-label:
    fontFamily: "'Barlow Condensed', 'Barlow', ui-sans-serif, sans-serif"
    fontSize: "14px"
    fontWeight: 600
    lineHeight: 1
    letterSpacing: "0.1em"
  station-label:
    fontFamily: "'Barlow Condensed', 'Barlow', ui-sans-serif, sans-serif"
    fontSize: "14px"
    fontWeight: 600
    lineHeight: 1
    letterSpacing: "0.09em"
  board-time:
    fontFamily: "'Barlow Condensed', 'Barlow', ui-sans-serif, sans-serif"
    fontSize: "15px"
    fontWeight: 600
    lineHeight: 1
    letterSpacing: "0.02em"
    fontFeature: "tnum"
  arrival-clock:
    fontFamily: "'Barlow Condensed', 'Barlow', ui-sans-serif, sans-serif"
    fontSize: "34px"
    fontWeight: 600
    lineHeight: 1
    letterSpacing: "0.02em"
    fontFeature: "tnum"
  body-desk:
    fontFamily: "'Barlow', ui-sans-serif, system-ui, sans-serif"
    fontSize: "13px"
    fontWeight: 400
    lineHeight: 1.45
    letterSpacing: "0.006em"
  body-touch:
    fontFamily: "'Barlow', ui-sans-serif, system-ui, sans-serif"
    fontSize: "15px"
    fontWeight: 400
    lineHeight: 1.45
  column-label:
    fontFamily: "'Barlow Condensed', 'Barlow', ui-sans-serif, sans-serif"
    fontSize: "12px"
    fontWeight: 600
    lineHeight: 1
    letterSpacing: "0.07em"
  meta:
    fontFamily: "'Barlow', ui-sans-serif, system-ui, sans-serif"
    fontSize: "12px"
    fontWeight: 400
    lineHeight: 1.45
  bullet-letter:
    fontFamily: "'Barlow Condensed', 'Barlow', ui-sans-serif, sans-serif"
    fontSize: "12px"
    fontWeight: 700
    lineHeight: 1
    letterSpacing: "0"
rounded:
  hairline: "3px"
  key: "4px"
  small: "6px"
  control: "7px"
  field: "8px"
  panel: "11px"
  overlay: "12px"
  touch-card: "14px"
  portal-card: "16px"
  feature: "18px"
  sheet: "26px"
  full: "9999px"
spacing:
  hair: "4px"
  tight: "8px"
  row: "10px"
  inset: "14px"
  gutter: "16px"
  desk-margin: "24px"
  work-margin: "32px"
components:
  button:
    backgroundColor: "{colors.raised-deck}"
    textColor: "{colors.porcelain-ink}"
    rounded: "{rounded.control}"
    padding: "0 12px"
    height: "32px"
  button-hover:
    backgroundColor: "{colors.lit-deck}"
  button-primary:
    backgroundColor: "{colors.porcelain-ink}"
    textColor: "{colors.midnight-enamel}"
    rounded: "{rounded.control}"
    padding: "0 12px"
    height: "32px"
  button-primary-hover:
    backgroundColor: "#FFFFFF"
  button-danger:
    backgroundColor: "rgba(255, 74, 91, 0.14)"
    textColor: "#FF9AA4"
    rounded: "{rounded.control}"
    padding: "0 12px"
    height: "32px"
  button-small:
    rounded: "{rounded.small}"
    padding: "0 9px"
    height: "26px"
  button-touch:
    rounded: "11px"
    padding: "0 16px"
    height: "44px"
  button-large:
    rounded: "{rounded.overlay}"
    padding: "0 20px"
    height: "48px"
  icon-button:
    textColor: "{colors.platform-grey}"
    rounded: "{rounded.control}"
    size: "30px"
  line-bullet:
    textColor: "{colors.midnight-enamel}"
    typography: "{typography.bullet-letter}"
    rounded: "{rounded.full}"
    size: "18px"
  segmented:
    backgroundColor: "{colors.night-deck}"
    rounded: "{rounded.field}"
    padding: "2px"
  segmented-option:
    textColor: "{colors.timetable-grey}"
    rounded: "{rounded.small}"
    padding: "0 10px"
    height: "26px"
  segmented-option-selected:
    backgroundColor: "{colors.lit-deck}"
    textColor: "{colors.porcelain-ink}"
  select:
    backgroundColor: "{colors.night-deck}"
    textColor: "{colors.porcelain-ink}"
    rounded: "{rounded.field}"
    height: "30px"
  search-field-desk:
    backgroundColor: "{colors.night-deck}"
    textColor: "{colors.timetable-grey}"
    rounded: "{rounded.field}"
    padding: "0 7px 0 11px"
    height: "32px"
  search-field-touch:
    backgroundColor: "{colors.raised-deck}"
    textColor: "{colors.porcelain-ink}"
    rounded: "{rounded.overlay}"
    padding: "0 12px"
    height: "42px"
  nav-item:
    textColor: "{colors.platform-grey}"
    rounded: "{rounded.control}"
    padding: "0 10px"
    height: "34px"
  nav-item-active:
    backgroundColor: "{colors.lit-deck}"
    textColor: "{colors.porcelain-ink}"
  disruption-pill:
    backgroundColor: "rgba(246, 177, 58, 0.11)"
    textColor: "{colors.delay-amber}"
    rounded: "{rounded.full}"
    padding: "0 12px 0 10px"
    height: "28px"
  chip:
    textColor: "{colors.platform-grey}"
    rounded: "{rounded.full}"
    padding: "0 14px"
    height: "32px"
  chip-selected:
    backgroundColor: "{colors.porcelain-ink}"
    textColor: "{colors.midnight-enamel}"
  panel:
    backgroundColor: "{colors.night-deck}"
    rounded: "{rounded.panel}"
  panel-header:
    typography: "{typography.sign-label}"
    padding: "0 14px 0 16px"
    height: "42px"
  table-cell:
    typography: "{typography.body-desk}"
    padding: "0 10px"
    height: "32px"
  board-row:
    padding: "0 7px"
    height: "31px"
  touch-card:
    backgroundColor: "{colors.night-deck}"
    rounded: "{rounded.touch-card}"
  popover:
    backgroundColor: "rgba(21, 31, 56, 0.985)"
    rounded: "{rounded.overlay}"
    width: "372px"
  toast:
    backgroundColor: "#18233F"
    rounded: "10px"
    padding: "11px 14px"
  sheet:
    backgroundColor: "{colors.overlay-deck}"
    rounded: "{rounded.sheet}"
    padding: "6px 12px 10px"
---

# Design System: SlipStream · Night Line (direction C)

## Overview

**Creative North Star: "The Night Line"**

SlipStream drawn as a transit system after dark. The surface is midnight enamel, the text is porcelain, and the only saturated colour on the page belongs to five lines: Acquire, Library, Process, Deliver and Notify, one per Role Group. Nodes are stations, Works are trains, and a Work's journey from search to screen is a route with stops. The admin home is a status line, a network map and a departures board, never a wall of stat cards or posters.

The world borrows the material honesty of signage rather than of a dashboard. Labels that name places (stations, lines, panels, page titles, column heads) are set in tracked, uppercase Barlow Condensed, the way a platform sign is. Everything a person reads as a sentence (the status line, Work titles, disruptions, overview copy) stays in sentence-case Barlow. Density is high on the desk (13px body, 31 to 32px rows) and relaxes to 15px body with 44 to 54px rows on touch surfaces. The member portal keeps the enamel and the route device but speaks in plain sentence-case headlines and leads with the Library line's blue.

State is told through form before colour: a delay is an amber ring around a bullet or station, a failure a red double ring, a suspension a dashed slate line, a paused Node a dashed grey ring, and a train is a white rounded car riding the line. Motion is functional: departures flip in place, changed rows stay lit until noticed, the chosen line burns while the rest dims.

**Key Characteristics:**
- Five line inks are the only chromatic colour; every surface and text tone is a blue-tinted neutral.
- Tonal layering on enamel, outlined by a 1px inset hairline, instead of drop shadows on resting surfaces.
- Barlow Condensed uppercase for signage, Barlow sentence case for prose.
- Line bullets (a coloured disc with the line's letter) wherever a Role appears.
- State shapes: ring for delay, double ring for failure, dash for suspension, white car for a train.
- Dark only in this build; light and system themes are an unresolved obligation (P-012).

## Colors

Midnight blue-black neutrals stepped by a few points of lightness, with five line inks and three state colours as the only chroma.

### Primary
- **Porcelain Ink** (`porcelain-ink`): primary text, the primary button fill, the selected chip and switch, the train car, station rings, the focus ring. On this world it is the action colour; there is no separate brand accent on the admin desk.

### Secondary
- **The five line inks** (`acquire-red`, `library-blue`, `process-amber`, `deliver-green`, `notify-violet`): each paints only its own line: map strokes, line bullets, progress bars and route rails of Work on that line. They are passed to components as a per-element `--ink` custom property rather than hard-coded into each component.
- **Library Blue** doubles as the interaction tint: caret and accent colour, text selection (at 40%), the lit-row wash (`rgba(61,123,255,0.09)`) and its marker dot, the portal's active-nav underline, bell dot and request route.

### Tertiary
- **Delay Amber** (`delay-amber`): delays and stalls; ring marks, the disruptions pill, delayed departure status, delayed station subtitles.
- **Failure Red** (`failure-red`): failures; double rings, notification dot, failure text. Danger text on tinted backgrounds uses a lighter pink tint for legibility.
- **Suspended Slate** (`suspended-slate`): suspended lines and stations, always as a dashed stroke, never as text.

### Neutral
- **Midnight Enamel** (`midnight-enamel`): the page; the map panel and phone strip card sit directly on it.
- **Night Deck** (`night-deck`): panels, sidebar, tables, cards, segmented track, select and search fields.
- **Raised Deck** (`raised-deck`): default buttons, touch search fields, sheet groups, hover on portal cards.
- **Lit Deck** (`lit-deck`): the selected or current item: active nav, selected segment, pressed legend item, button hover.
- **Overlay Deck** (`overlay-deck`): command palette and action sheet; popovers and the map hover card use the translucent `rgba(21,31,56,0.97–0.985)` equivalent, toasts `#18233F`.
- **Platform Grey** (`platform-grey`): secondary text: nav labels, overview copy, route stops, icon buttons.
- **Timetable Grey** (`timetable-grey`): tertiary text and metadata, column labels, placeholders, icons at rest.
- **Conduit Grey** (`conduit-grey`): non-text marks only: status-line separators, route arrows, hover outline on search.
- **Rule / Rule Strong** (`rule`, `rule-strong`): 7.5% and 13% porcelain hairlines; `rule` divides rows and outlines panels, `rule-strong` outlines controls and overlays.

### Named Rules
**The Line Ink Rule.** A line colour appears only on things that belong to that line. No line ink is used as decoration, as a section accent, or as a generic status colour.

**The Shape Before Colour Rule.** Every state carries a shape (ring, double ring, dashed stroke, car) as well as a colour, so the map and the bullets remain readable without hue.

## Typography

**Display / Signage Font:** Barlow Condensed (with Barlow, ui-sans-serif)
**Body Font:** Barlow (with ui-sans-serif, system-ui)

**Character:** Barlow is a slightly rounded grotesk drawn from road and rail signage; its condensed cut, set in tracked caps, is the station sign, and the regular cut carries everything a person reads as a sentence.

### Hierarchy
- **Page title** (Condensed 600, 27px, line-height 1, 0.06em, uppercase): desk page headings (Movies, Activity, Nodes). On phones the large title is 36px at 0.05em.
- **Work title** (Barlow 600, 36px desk / 28px phone, 1.04, -0.018em): a Work's own name, always sentence case.
- **Portal headline** (Barlow 600, 30px, 40px on wide portal, -0.016em): the member greeting and page titles; portal section heads at 20px.
- **Status line** (Barlow 500, clamp 15.5–18px, 1.35): the one-sentence network summary on admin home; 17px on phones.
- **Sign label** (Condensed 600, 14px, 0.1em, uppercase): panel headers (Disruptions, Departures, Activity), section heads at 15px, sidebar line names at 14px / 0.07em.
- **Station label** (Condensed 600, 14px, 0.09em, uppercase): map station names; phone strip station names at 17px / 0.08em.
- **Board time** (Condensed 600, 15px, tabular): departure times; the portal arrival clock runs the same face at 34px (46px wide).
- **Body** (Barlow 400, 13px desk / 15px touch, 1.45, 0.006em): rows, controls, sentences. Overview prose runs 15–16px at 1.6, capped at 64–68ch.
- **Column label** (Condensed 600, 12px, 0.07em, uppercase, timetable grey): table heads and the coverage corner cell.
- **Meta** (Barlow 400, 12–12.5px, timetable grey): timestamps, subtitles, panel meta.

### Named Rules
**The Signage Rule.** Condensed uppercase is for names of places and furniture: stations, lines, panels, columns, page titles. Work titles, sentences and the portal's member-facing headings stay in sentence-case Barlow.

**The Tabular Clock Rule.** Every time, rate, count and percentage that updates in place uses tabular numerals so split-flap updates never shift the row.

## Layout

The desk admin is a fixed two-column shell: a 232px labelled sidebar (nav with counts, then a Lines list with per-line status) and a 52px header holding the page title, a search field at `min(440px, 38%)`, live rate, the disruptions pill and the bell. Admin home stacks the status line, then a grid of the network map beside a side column of `clamp(348px, 27vw, 448px)` (Disruptions over Departures), then the activity table. Page gutters are 24px, panel gaps 16px; Work detail uses 32px margins, a 400px backdrop hero and a 372px side column. Below 1360px the route column, map legend key and departure platform column collapse.

Phones use a navigation bar of 44px that only becomes a blurred material once content scrolls beneath it, a five-tab bar of 50px, 16px side margins and grouped cards. The map becomes a vertical strip with stations listed beside it.

The member portal centres content at 1120px (720px narrow), 16px margins on phones and 32px on wide, with 36–44px between sections. Request cards run one column on phones and two on wide.

Poster grids use `repeat(auto-fill, minmax(150px, 1fr))` with 24/18px gaps (small 112px, large 208px).

## Elevation & Depth

Resting surfaces are flat: depth comes from stepping lightness (enamel, night deck, raised deck, lit deck) and a 1px inset porcelain hairline at 7.5% or 13%. Shadows exist only on things that float above the page (popovers, the palette, toasts, sheets, the map hover card) and on poster art, where a long, soft, offset-down shadow lifts the image off the enamel. Translucent bars (phone nav, tab bar, portal header) use `blur(20–22px) saturate(1.5–1.6)` over 80–86% enamel.

### Shadow Vocabulary
- **Hairline outline** (`box-shadow: inset 0 0 0 1px rgba(237,241,247,0.075)`): panels, cards, tables; the 13% variant on controls and overlays.
- **Poster lift** (`box-shadow: 0 0 0 1px rgba(237,241,247,0.06), 0 12px 26px -16px rgba(0,0,0,0.9)`): poster tiles at rest; deepens on hover with a 3px rise.
- **Popover** (`box-shadow: 0 22px 54px -14px rgba(0,0,0,0.78), inset 0 0 0 1px rgba(237,241,247,0.13)`): menus, disruption and activity popovers.
- **Palette** (`box-shadow: 0 32px 90px -24px rgba(0,0,0,0.85), inset 0 0 0 1px rgba(237,241,247,0.13)`): the command palette.
- **Toast** (`box-shadow: 0 16px 36px -12px rgba(0,0,0,0.75), inset 0 0 0 1px rgba(237,241,247,0.13)`).
- **Cut-out halo** (`box-shadow: 0 0 0 2px <surface>`): a ring of the underlying surface colour that separates trains, badges and dots from the line or poster under them, followed by a state ring at 3.5–4.5px when needed.

### Named Rules
**The Floats Only Rule.** Nothing at rest casts a shadow except poster art. A panel that needs separation gets a lighter deck or a hairline, not a shadow.

## Shapes

Corners are soft and graded by scale: 6px on small controls and posters, 7px on buttons and nav items, 8px on fields and segmented tracks, 11px on desk panels, 12px on popovers, 14px on phone cards and toasts, 16–18px on portal cards and the feature card, 26px on the top of action sheets. Pills (disruptions pill, chips, line filters, legend items) are fully rounded. Bullets, avatars and status dots are circles.

The transit geometry is the recurring silhouette: lines are 6px butt-capped strokes running only at 0, 45 and 90 degrees with rounded joins; stations are porcelain rings (3.5px stroke) filled with enamel, or a perpendicular tick on a line terminus; trains are rounded white cars (20×9px on the map and request routes, 13×6px on poster progress, 8×17px on route rails); suspension is a dashed 9/7 slate stroke. Progress bars are 4px (3px thin) tracks at 9% porcelain with a line-ink fill.

## Components

### Buttons
Quiet, compact and tactile; one primary per group.
- **Shape:** gently rounded (7px; 6px small; 11px touch; 12px large).
- **Default:** raised deck with a 13% inset hairline, porcelain text at 500, 32px tall, icons in platform grey.
- **Primary:** porcelain fill with enamel text at 600, no outline; hover goes to pure white.
- **Danger:** 14% failure-red wash with a 35% red hairline and a pink text tint.
- **Hover / Press:** hover lifts the fill one deck step (fine pointers only); press scales to 0.97 over 160ms on the house ease. Disabled is 38% opacity. Loading swaps the icon for a 12px spinner.
- **Icon button:** 30px square, transparent, platform grey; hover adds a 7% porcelain wash. Bordered variant is 32px on raised deck.

### Chips
- **Style:** fully rounded, 32px tall, 13% inset hairline, platform grey text at 500.
- **State:** selected inverts to porcelain fill with enamel text. Desk line filters are 30px with a leading bullet and select to lit deck.

### Cards / Containers
- **Corner Style:** 11px desk panels, 14px phone cards, 16–18px portal cards.
- **Background:** night deck; the map panel and phone strip card sit on midnight enamel with a stronger hairline so the diagram reads as a printed sheet.
- **Shadow Strategy:** none; see the Floats Only Rule.
- **Border:** 1px inset hairline at 7.5%.
- **Internal Padding:** 42px header row padded 16px left / 14px right; rows 14–16px inset.

### Inputs / Fields
- **Style:** desk search and select sit on night deck with a hairline, 8px radius, 30–32px; touch search and portal fields sit on raised deck without a hairline, 42–48px, 12–14px radius, 17px text.
- **Focus:** fields swap to a timetable- or conduit-grey inset hairline; every other focusable gets a 2px porcelain outline offset 2px.
- **Segmented control:** 2px-inset track on night deck; the selected option moves to lit deck with a 13% hairline and a 1px drop.
- **Switch:** 30×18 track; checked is porcelain with an enamel knob that slides 12px.

### Navigation
- **Desk sidebar:** 34px items, 14px Barlow 500 in platform grey with counts right-aligned; current item is lit deck with a hairline and porcelain text and icon. Below a hairline, a Lines list pairs each bullet with its condensed line name and a coloured status word.
- **Phone:** five-tab bar with 10.5px labels, current tab in porcelain, badges as porcelain pills with an enamel halo; large condensed titles collapse into a 17px centred bar title on scroll; back and action buttons become 36px blurred discs over hero images.
- **Portal:** sentence-case text nav in timetable grey; current link turns porcelain with a 2px Library-blue underline.

### Line Bullet (signature)
An 18px disc in the line's ink with the line letter in 12px Condensed 700 enamel. It stands in for a Role wherever one appears: sidebar, tables, disruptions, departures, coverage. State is a ring outside a 2px halo: amber for delays or stalls, red for failure, slate for partial or suspended, grey for paused. 16px in strip rows and search results.

### Network Map (signature)
An SVG diagram fitted to its panel: line runs at 45° and 90° only, interchange stations as a single porcelain ring sized to span every line through it (the strip map draws them as capsules across its parallel strokes), station labels in condensed caps with a sentence-case Barlow subtitle (amber when delayed). Trains move along runs. Choosing a line from the legend or sidebar dims every other line, station and train to 30% opacity. The legend lists each line with its running count and a key for train, delayed, failed and suspended.

### Strip Map (signature, phone)
The network as a vertical strip: 5px line strokes on the left, stations as rows (17px condensed name, 13px subtitle, bullets right). Stalled and failed trains turn amber and red in place.

### Route (signature)
A Work's itinerary: a 4px rail in the line ink with 10px porcelain stop dots; future segments drop to 32% opacity and hollow stops; the current stop is a pulsing white car; delayed adds an amber ring, failed a red double ring. The portal's request route is the same device horizontally across Requested, Approved, On its way and Arrived, in Library blue, ending in a green ring on arrival and a dashed slate line when denied.

### Departures Board (signature)
A fixed-layout table: condensed tabular time, bullet, Work, Node in timetable grey, and a condensed status word right-aligned (amber when delayed). Values that change flip in character by character.

## Do's and Don'ts

### Do:
- **Do** give every Role a line bullet and paint line-owned marks (strokes, bars, rails) through the element's own line ink.
- **Do** carry every state as a shape plus a colour: amber ring for delay, red double ring for failure, dashed slate for suspended, dashed grey for paused.
- **Do** set station, line, panel, column and page-title labels in Barlow Condensed 600 uppercase with 0.06–0.12em tracking, and everything sentence-like in Barlow.
- **Do** separate resting surfaces with deck steps and the 7.5% or 13% inset hairline.
- **Do** use the house ease `cubic-bezier(0.23, 1, 0.32, 1)` at 150–240ms for state changes and a 0.97 press scale on buttons, chips and segments.
- **Do** use tabular numerals for every value that updates in place.
- **Do** keep hover treatments behind `(hover: hover) and (pointer: fine)` and give touch surfaces an `:active` wash or scale instead.

### Don't:
- **Don't** use a line ink outside its own line, as decoration, or as a generic accent.
- **Don't** compose admin home from stat cards or a poster wall; the network is the home.
- **Don't** cast drop shadows from panels, cards or tables; shadows belong to floating layers and poster art.
- **Don't** set Work titles, disruption sentences or portal headlines in condensed caps.
- **Don't** set text in conduit grey or suspended slate; they are for marks and strokes.
- **Don't** draw map lines at angles other than 0°, 45° and 90°.
