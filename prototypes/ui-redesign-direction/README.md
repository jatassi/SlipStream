# UI redesign direction prototype

**Throwaway.** This never merges. It answers one question for [UI redesign direction](https://github.com/jatassi/SlipStream/issues/25): what does SlipStream v2 look and feel like? It renders three design directions across the four shells that later per-surface prototypes build on.

Open `dist/index.html` by double-clicking it. It is one self-contained file. Fonts load from Google Fonts, so it needs a network connection. Everything else is inline, and all state lives in memory.

## The directions

- **A · Slipstream.** The system reads as airflow in a wind tunnel. Each job is a smoke filament on the home screen's Tunnel, and a failure curls into a vortex at its stage.
- **B · House Lights.** A dark auditorium whose only colour comes from the Works on screen. Artwork lights its surroundings, hovering dims the house, and a Work in progress develops from monochrome into colour.
- **C · Night Line.** The Nodes form a transit network. Role Groups are lines, Nodes are stations, jobs are trains, and a Work's detail shows its route as an itinerary.

## Use the switcher

The bar at the bottom switches directions and shells.

- Press ← and → to switch directions.
- Choose **Admin Desk**, **Admin Phone**, **Portal Phone** or **Portal Desk** to switch shells. The phone shells render inside a device frame on a desk browser and full screen on a phone.
- Press `\` to hide the bar.

Every screen opens from a URL, for example `dist/index.html?variant=B&shell=phone&view=work/brightwater`. All three directions support these `view` routes:

- **Admin Desk.** `home`, `library`, `series`, `work/<id>`, `activity`, `nodes`, `requests`, and ⌘K over any of them.
- **Admin Phone.** `home`, `library`, `work/<id>`, `work/<id>/actions`, `activity`, `search`, `more`.
- **Portal.** `home`, `search`, `library`, `work/<id>`, `request/<id>`, `requests`.

A Work's id is its title in kebab case, such as `halcyon-drift` or `brightwater`. Request ids run from `r1` to `r8`; `r3` is June's Halcyon Drift.

## What is real and what is invented

- **Invented.** Every title, poster, person, Node, release and number. The prop department in `src/core/props.js` draws each poster, backdrop and episode still on a canvas from a palette and a motif, so no copyrighted artwork ships. The library is 46 movies and 12 series in `src/core/data.js`.
- **Live.** `src/core/sim.js` advances jobs nine times faster than real rates. Over a few minutes, Halcyon Drift finishes downloading, transfers to vault, imports, and becomes available, and June's request turns available with it. Displayed times stay at real rates.
- **Domain.** Labels follow `GLOSSARY.md`. The Role Group names Library, Process, Deliver and Notify are proposals for this prototype. Only Acquire is in the glossary.

## Known limits

- Dark only, by the chef's instruction. Parity row P-012 still keeps light, dark and system themes.
- Actions that would change a Node, such as pause or delete, show a confirmation and change little or nothing.
- Gestures (edge-swipe back, sheet flick) were tested with a mouse in Chrome, not on a phone.

## Rebuild and capture

```bash
cd tools && npm install && cd ..
node build.mjs
```

`build.mjs` writes `dist/index.html` and `dist/artifact.html`, the copy published as a private link. It bundles only the Lucide icons the sources name.

```bash
node tools/shoot.mjs --out media/shots A:wide:home B:phone:work/brightwater C:portal:home
tools/matrix.sh media/review A B C
node tools/record.mjs A B C
```

`shoot.mjs` captures screens and prints console errors. `matrix.sh` captures the 23 review views of each direction. `record.mjs` records one screencast per direction from the tours in `tools/tours.mjs`.
