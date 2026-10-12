# Brief: isometric Nodes topology, one variant

You are building ONE variant of a throwaway prototype: an isometric, animated, alive version of SlipStream's Nodes topology view. The owner (Jackson) asked: "design beautiful representations of nodes and their resources for the nodes topology view ... an isometric version of the topology. The isometric figures should be highly detailed, have playful but crisp motion, and the topology view should feel alive." He named the **anatomy** skill as the method.

Scratch root: `$S = /tmp/claude-0/-home-user/3df9f621-c15d-5ca0-bd63-91a832856c4e/scratchpad`. Work only in your variant folder (`$S/topo/a/` or `$S/topo/b/`). Never touch git repos, never run git stash or anything that affects a worktree. Do not call any `mcp__hearthbot__` tool.

## Read first, in full
1. `$S/anatomy/src/SKILL.md` (the anatomy skill: workflow, craft rules). Follow its workflow and craft rules; this is the quality bar.
2. `$S/anatomy/src/references/craft.md`, `references/kit.md`, `references/motion.md`. Skim `references/verify.md` for the capture and line checks.
3. Closest examples: `$S/anatomy/src/examples/dial-indicator/build.mjs` (smallest full page), `examples/desk-computer/build.mjs` (screens on faces), `examples/arcade-cabinet/` (cutaway, live parts, dense Hero).
4. `$S/topo/model.mjs`: the shared truth. Every number shown and every speed of motion comes from `sample(state, t)`. Do not invent other numbers; if you need one, add it to your own copy logic but derive from sample().
5. `$S/topo/theme.css`: the kit's tokens resolved to SlipStream's dark "Lit Console" palette. Load it after the kit CSS. One theme only: dark. Skip the light theme entirely.

## Product context (plain)
- SlipStream v2 runs on **Nodes**. Jackson has two: the **Media server** at home (Windows 11 PC, Intel UHD 750 with Quick Sync, four data drives, the embedded libtorrent torrent engine, Plex next to it) and the **Relay Node** (a small Hetzner VPS in Ashburn) that fronts `slipstream.atassi.org` and `plex.atassi.org` over TLS and reaches home through a **WireGuard tunnel**. Remote viewers come in through the Relay Node; the living-room TV watches at home directly.
- Resources to represent: **Storage** (four drives with fill and read/write activity), **Transcoder** (Quick Sync sessions, load, fps), **Torrent engine** (downloads arriving from peers via trackers, seeding out), **Network / relay traffic** (tunnel throughput and latency, TLS front, viewers).
- The Nodes screen decision is "Topology canvas plus traffic": a canvas of Nodes with the live traffic drawn on the links its bytes cross. Your figure is that canvas, reimagined isometrically.
- Design direction (DESIGN.md "Lit Console"): one dark theme, tinted graphite at hue 214. Live is luminance: anything in flight is white with a blue glow (`--glow`, #74affb family). Fixed signal hues only for status: green ok, amber warning, red failure, always beside words or as status LEDs. No other colour.
- Jackson's standing preference: approachable first. The page opens on one plain sentence (the `sentence` from the model), the figure, and a side panel that explains a selected Node or resource in plain words with numbers in words ("Writing 31 MB/s to the Downloads drive"). Machinery (host names, addresses, engines, ports) goes under a "Details" disclosure in that panel.

## What to build
A standalone `index.html` in your folder, produced by a `build.mjs` (Node, imports `$S/anatomy/src/kit/iso-kit.mjs`, and `lathe.mjs`/`tube.mjs` if you use them; inline the kit with `$S/anatomy/src/scripts/inline-kit.mjs` `kitScript()`; inline the model by reading `model.mjs` source and stripping `export`). No external files at runtime except Google Fonts (Geist, Geist Mono) if you want them.

Page layout (desktop ~1440 wide): a slim header "Nodes" with Topology (active) and Roster (inert) tabs, the status sentence in large type (24px/600), then the isometric stage filling the width (roughly 1100 x 620 viewBox, adjust), and a right-hand panel (~320px) for the selection. Keep the anatomy card conventions in spirit: a small hint bottom-left of the stage ("Hover a part to see what it is doing") and a live readout bottom-right (e.g. "↓ 26.4 MB/s · ↑ 14.2 Mbit/s to 2 viewers"). At 390 px wide the page must not scroll sideways (stack the panel under the stage; the stage may scale down).

The figure: both Nodes, all four resources, the tunnel between them, the trackers/peers feeding the torrent engine, and the viewers fed by the Relay Node, in one isometric scene (azimuth/elevation of your choosing). Highly detailed per the skill: every part sits on something, screws, seams, bevels, vents, label plates, LEDs, cables that land on connectors. "The object is the explanation": each resource is a recognisable physical mechanism whose motion IS the metric (see your variant concept below).

Alive (the core ask):
- Continuous calm motion driven by `sample(state, t)` each frame: packets/capsules travel the tunnel at a rate and spacing proportional to `tunnel.mbps`; drive LEDs flicker with `disks[i].reading/writing`; the transcoder's mechanism runs at a speed set by `transcoder.load`/`fps`; torrent pieces arrive at a rate set by `downMBps` and leave (seeding) by `seedUpMBps`; viewer devices show they are receiving.
- Playful but crisp: eased, springy, never jittery. Use the skill's motion rules (time-based dt, exponential follow or springs, envelopes with attack/decay for one-shot events like a piece landing or a drive blink, nothing pops).
- State changes happen in place without reload. Listen for `window.addEventListener('message', e => e.data?.state && setState(e.data.state))`, and read `?state=` (calm | busy | warn | down) on load (default `busy`). On change, ease every quantity to the new targets (settle over ~0.6–1.5 s): e.g. going to `down`, the media server's mechanisms spin down, its LEDs fade, packets stop entering the tunnel and the ones in flight drain out, the tunnel's home end shows a red status lamp, the Relay Node shows it is waiting. `warn` shows an amber lamp on the Relay Node and visibly slower, sparser tunnel traffic. Also update the sentence and panel.
- Selection: hovering or clicking a Node or resource selects it. Selected parts become the one lit accent (the skill's `lit` treatment), everything else steps down a tone (CSS on a data attribute, as in motion.md "Selection"). The side panel shows its plain-language summary. Keyboard: Tab focuses the stage, arrows cycle through Nodes and resources, Escape clears. With nothing selected the panel shows the overall summary (both Nodes, one line each).
- An idle tour: after ~4 s with no input, gently cycle the selection through the resources every ~4 s; stop on hover/focus, resume 4 s after the last input.
- Reduced motion: no tour, no travel; still frame that still reads, still responds to input.
- Stop the rAF loop when the stage is offscreen (IntersectionObserver).
- Performance: keep a frame under ~3 ms of script. Build static parts once; per frame only move transforms and rebuild the few paths that change shape (rotating parts, capsules).

## Verify like a critic (required)
Chrome: `CHROME_PATH=$S/chrome` (a wrapper that adds --no-sandbox). Commands, run from `$S/anatomy/src`:
- `node scripts/capture.mjs $S/topo/<v>/index.html --select .stage --out $S/topo/shots/<v>-busy.png --times 600,1200` and one per state (`?state=` works with a file URL? if not, add a `#state` hash read too). Capture hover on a resource (`--hover fx,fy`), phone width (`--width 390`), and 4x zooms (`--zoom x,y,w,h`) over every resource and junction.
- `node scripts/lines.mjs $S/topo/<v>/index.html --out $S/topo/shots/<v>-lines.png` must print `line ends: 0`. Read the ringed PNG if not.
- Read every PNG you make. Ask the skill's question: is each Node instantly a real object, rich in small parts with volume, every line ending on something, rewarding at 4x, no bare face or empty corner? Iterate until yes. Console must be clean.
- The kit audit (`A.recorder` + `auditOrExit`) is optional for this prototype; run it if you use tubes and have time.

Budget: about two hours of focused work. Have a working page that opens within the first 30 minutes, then deepen. Do not ask questions; pick reasonable defaults and note them.

## Report back (under 300 words)
Path to index.html and build.mjs, the parts count, what each resource's mechanism is and what its motion means, the screenshots you made (paths) and which you consider best, the line-check result, known faults you did not fix.
