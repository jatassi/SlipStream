const P = DATA.P;
const stage = document.querySelector(".stage");
const svg = stage.querySelector("svg");
const NS = "http://www.w3.org/2000/svg";
const still = matchMedia("(prefers-reduced-motion: reduce)");
const $ = (s, root = svg) => root.querySelector(s);
const $$ = (s, root = svg) => [...root.querySelectorAll(s)];
const make = (tag, attrs, parent) => { const el = document.createElementNS(NS, tag); for (const k in attrs) el.setAttribute(k, attrs[k]); parent.appendChild(el); return el; };
const clamp = (v, a, b) => Math.min(b, Math.max(a, v));
const smooth = (a, b, x) => { const t = clamp((x - a) / (b - a), 0, 1); return t * t * (3 - 2 * t); };
const burst = (age, rise, fall) => (age < 0 ? 0 : age < rise ? smooth(0, rise, age) : Math.exp(-(age - rise) / fall));
const fmt = (v, d = 1) => (v == null ? "–" : (+v).toFixed(d));

const pick = () => {
  const q = new URLSearchParams(location.search).get("state");
  const h = location.hash.replace("#", "").replace("state=", "");
  return STATES[q] ? q : STATES[h] ? h : "busy";
};
let stateId = pick();

function track(points) {
  const acc = [0];
  for (let i = 1; i < points.length; i++) acc.push(acc[i - 1] + Math.hypot(points[i][0] - points[i - 1][0], points[i][1] - points[i - 1][1], points[i][2] - points[i - 1][2]));
  const len = acc[acc.length - 1];
  const at = (s) => {
    s = clamp(s, 0, len);
    let lo = 0, hi = acc.length - 1;
    while (hi - lo > 1) { const mid = (lo + hi) >> 1; if (acc[mid] <= s) lo = mid; else hi = mid; }
    const f = (s - acc[lo]) / (acc[hi] - acc[lo] || 1);
    const a = points[lo], b = points[hi];
    return [a[0] + (b[0] - a[0]) * f, a[1] + (b[1] - a[1]) * f, a[2] + (b[2] - a[2]) * f];
  };
  return { len, at };
}

const E = {};
let calm = still.matches;
const ease = (name, target, dt, tau = 0.4) => {
  const v = E[name] ?? target;
  E[name] = calm ? target : v + (target - v) * (1 - Math.exp(-dt / tau));
  return E[name];
};

const tunnel = track(DATA.tunnel);
const capLayer = $("[data-capsules]");
const capsules = Array.from({ length: 28 }, () => {
  const g = make("g", { opacity: 0 }, capLayer);
  return { g, halo: make("path", { class: "capsule capsule-halo" }, g), core: make("path", { class: "capsule capsule-core" }, g), s: -1 };
});
let capAcc = 0;

const pieceLayer = $('[data-layer="pieces"]');
const cube = (() => {
  const c = [[-1.1, -1.1, 0], [1.1, -1.1, 0], [1.1, 1.1, 0], [-1.1, 1.1, 0]];
  const top = c.map(([x, y]) => iso([x, y, 2.2], { ...P, origin: [0, 0] }));
  const bot = c.map(([x, y]) => iso([x, y, 0], { ...P, origin: [0, 0] }));
  const hull = [top[0], top[1], bot[1], bot[2], bot[3], top[3]];
  return pathOf(hull, true);
})();
const pieces = Array.from({ length: 70 }, () => ({ el: make("path", { class: "piece", d: cube, opacity: 0 }, pieceLayer), route: null, s: 0, alive: false }));
const downTracks = DATA.down.map(track);
const seedTracks = DATA.seed.map(track);
let downAcc = 0, seedAcc = 0;
const houseFlash = new Map();

const panels = DATA.panels.map((p, n) => ({
  ...p,
  cells: Array.from({ length: p.cols * p.rows }, (_, i) => $(`[data-cell="${n}-${i}"]`)),
  drops: $(`[data-drops="${n}"]`),
  progress: $(`[data-progress="${n}"]`),
  flash: new Map(),
  shown: -1,
}));
const dropEls = [];
const C = DATA.cell;
const cellCenter = (panel, i) => {
  const row = Math.floor(i / panel.cols), col = i % panel.cols;
  const z1 = C.z0 + panel.rows * (C.h + C.gap) - C.gap;
  return { u: panel.x0 + col * (C.w + C.gap) + C.w / 2, z: z1 - (row + 1) * C.h - row * C.gap + C.h / 2 };
};
const shuttle = $('[data-part-move="shuttle"]');
let nextPanel = 0;

const drawers = [0, 1, 2, 3].map((i) => ({ el: $(`[data-drawer="${i}"]`), led: $(`[data-drive-led="${i}"] circle`) }));
const rollFrames = { low: $$('[data-roll="low"]'), high: $$('[data-roll="high"]') };
let rollAngle = 0, rollShown = { low: 0, high: 0 };
const marks = $$("[data-marks]").map((el) => ({ el, name: el.dataset.marks.split("-")[0], lane: +el.dataset.marks.split("-")[1], pitch: +el.dataset.pitch }));
const lanes = $$(".lane");
const laneLamps = $$("[data-lane-lamp]");
const gears = { low: $('[data-gear="low"]'), high: $('[data-gear="high"]') };
const needle = $("[data-load-needle]");
const fpsText = $("[data-fps]");
let stripIn = 0, stripOut = 0;

const glowOf = (sel) => { const g = $(sel + " .lamp-glow"); return g; };
const lamps = { home: glowOf('[data-lamp="home"]'), relay: glowOf('[data-lamp="relay"]'), ok: glowOf('[data-tower-lamp="ok"]'), warn: glowOf('[data-tower-lamp="warn"]'), fail: glowOf('[data-tower-lamp="fail"]') };
const leds = { rx: $('[data-led="rx"] circle'), beacon: $('[data-led="beacon"]'), out: $("[data-out-led] circle"), plex: $("[data-plex-led] circle"), tv: $("[data-tv-led] circle"), trackers: $$("[data-tracker] circle") };
const cpuCells = $$("[data-cpu]");
const portLeds = $$("[data-port-led]");
const tls = $$("[data-tls]");
const houseWins = $$("[data-house-win]");

const screens = Object.fromEntries($$("[data-screen]").map((el) => [el.dataset.screen, { el, off: el.querySelector(".pic-off"), pan: el.querySelector("[data-pan]"), bar: el.querySelector("[data-progress-bar]") }]));

const flowLayer = $('[data-flow-dots="relay"]');
const flows = Object.fromEntries(Object.entries(DATA.relayRoutes).map(([name, pts]) => {
  const tr = track(pts);
  const dots = Array.from({ length: 10 }, () => make("circle", { class: "flow-dot", r: 0.9, opacity: 0 }, name === "plex" ? $("[data-device]").parentNode : flowLayer));
  return [name, { tr, dots, phase: Math.random() * 100 }];
}));

const PARTS = ["home", "storage", "transcoder", "torrent", "tunnel", "relay"];
let selected = null;
const groups = $$(".part[data-part]");
function select(name) {
  if (name === selected) return;
  selected = name;
  if (name) stage.dataset.focus = name; else delete stage.dataset.focus;
  for (const g of groups) g.toggleAttribute("data-on", g.dataset.part === name);
  renderDetails();
  renderPanel(lastSample);
  live.textContent = name ? titleOf(name) : "Overview";
}

let touring = !still.matches;
let tourClock = 0;
let idleTimer = 0;
const hold = () => {
  touring = false;
  clearTimeout(idleTimer);
  idleTimer = setTimeout(() => { touring = !still.matches; tourClock = 3.2; run(); }, 4000);
};

svg.addEventListener("pointermove", (e) => {
  const g = e.target.closest?.("[data-part]");
  hold();
  if (g) select(g.dataset.part);
  run();
});
svg.addEventListener("click", (e) => { const g = e.target.closest?.("[data-part]"); hold(); select(g ? g.dataset.part : null); run(); });
stage.addEventListener("pointerleave", () => hold());
stage.addEventListener("focus", () => { hold(); run(); });
stage.addEventListener("keydown", (e) => {
  const i = PARTS.indexOf(selected);
  let to;
  if (e.key === "ArrowRight" || e.key === "ArrowDown") to = PARTS[(i + 1) % PARTS.length];
  else if (e.key === "ArrowLeft" || e.key === "ArrowUp") to = PARTS[(i - 1 + PARTS.length) % PARTS.length];
  else if (e.key === "Home") to = PARTS[0];
  else if (e.key === "End") to = PARTS[PARTS.length - 1];
  else if (e.key === "Escape") to = null;
  else return;
  e.preventDefault();
  hold();
  select(to);
  run();
});

const sentence = document.querySelector("[data-sentence]");
const readout = document.querySelector("[data-readout]");
const live = document.querySelector("[data-live]");
const ui = {
  kicker: document.querySelector("[data-kicker]"),
  title: document.querySelector("[data-title]"),
  pill: document.querySelector("[data-pill]"),
  pillText: document.querySelector("[data-pill-text]"),
  lines: document.querySelector("[data-lines]"),
  details: document.querySelector("[data-details]"),
  dl: document.querySelector("[data-dl]"),
};
const H = NODES.home, R = NODES.relay;
const plural = (n, one, many = one + "s") => `${n} ${n === 1 ? one : many}`;
const words = ["no", "one", "two", "three", "four", "five", "six"];
const count = (n, one, many = one + "s") => `${words[n] ?? n} ${n === 1 ? one : many}`;
const pct = (v) => Math.round(v * 100);
const titleOf = (name) => ({ home: "Media server", storage: "Storage", transcoder: "Transcoder", torrent: "Torrent engine", tunnel: "Tunnel", relay: "Relay Node" })[name];
const kickerOf = (name) => ({ home: "Node · Home", storage: "Resource · Media server", transcoder: "Resource · Media server", torrent: "Resource · Media server", tunnel: "Link · Media server to Relay Node", relay: "Node · Ashburn" })[name];
const detailsOf = (name) => ({
  home: [["Host", H.host], ["Role", H.kind], ["System", H.os], ["Hardware", H.hw], ["Address", H.addr]],
  storage: H.disks.map((d) => [d.label, `${d.used} of ${d.size} TB`]),
  transcoder: [["Engine", H.transcoder.engine], ["Sessions", `up to ${H.transcoder.maxSessions}`]],
  torrent: [["Engine", H.torrent.engine], ["Trackers", H.torrent.trackers.join(", ")]],
  tunnel: [["Kind", TUNNEL.name], ["Port", `UDP ${TUNNEL.port}`], ["Home upload", `${TUNNEL.upstreamMbps} Mbit/s`]],
  relay: [["Host", R.host], ["Where", R.where], ["System", R.os], ["Hardware", R.hw], ["Address", R.addr], ["Fronts", R.fronts.join(", ")]],
  null: [["Media server", `${H.host} · ${H.addr}`], ["Relay Node", `${R.host} · ${R.addr}`], ["Tunnel", `${TUNNEL.name} · UDP ${TUNNEL.port}`]],
})[name];

function renderDetails() {
  ui.dl.innerHTML = detailsOf(selected).map(([k, v]) => `<dt>${k}</dt><dd>${v}</dd>`).join("");
}

function panelOf(m, name) {
  const down = m.health.home === "down";
  const tone = (h) => (h === "ok" ? ["ok", "Running"] : h === "warn" || h === "slow" ? ["warn", h === "slow" ? "Slow" : "Needs attention"] : ["fail", "Offline"]);
  const cap = (t) => t[0].toUpperCase() + t.slice(1);
  const homeLine = down ? "Offline. Last heard from at 21:31." : `Running. ${m.streams.length ? cap(`${count(m.streams.length, "stream")} playing`) : "Nothing playing"}${m.downloads.length ? `, ${count(m.downloads.length, "download")} coming in` : ""}.`;
  const relayLine = m.health.relay === "warn" ? `Plex has not confirmed remote access since 19:20. ${cap(count(m.relay.viewers, "remote viewer"))} still connected.` : down ? "Running, waiting for the media server to come back." : `Fronting two sites for ${count(m.relay.viewers, "remote viewer")}.`;
  if (!name) {
    return {
      kicker: "Overview", title: "Both Nodes", pill: down ? ["fail", "One Node offline"] : m.health.relay === "warn" ? ["warn", "One warning"] : ["ok", "All running"],
      lines: [
        `<div class="row"><div class="row-head"><span>Media server</span><span>Home</span></div><p>${homeLine}</p></div>`,
        `<div class="row"><div class="row-head"><span>Relay Node</span><span>Ashburn</span></div><p>${relayLine}</p></div>`,
        `<div class="row"><div class="row-head"><span>Tunnel</span><span>${m.tunnel.latencyMs == null ? "down" : m.tunnel.latencyMs + " ms"}</span></div><p>${down ? "No traffic. The relay end is waiting." : `Carrying <b>${fmt(m.tunnel.mbps)} Mbit/s</b> between them.`}</p></div>`,
      ],
    };
  }
  if (name === "home") {
    const used = m.disks.reduce((a, d) => a + d.used, 0), size = m.disks.reduce((a, d) => a + d.size, 0);
    const local = m.streams.filter((s) => s.where === "home").length;
    return {
      pill: tone(m.health.home),
      lines: down ? [`<p>The Relay Node cannot reach this machine. Last heard from at 21:31.</p>`, `<p>Remote viewers cannot start anything until it is back.</p>`] : [
        `<p>${m.streams.length ? `Serving <b>${count(m.streams.length, "stream")}</b>: ${count(m.remote.length, "goes", "go")} out through the Relay Node${local ? `, ${words[local]} ${local === 1 ? "plays" : "play"} on the living-room TV` : ""}.` : "Nothing is playing."}</p>`,
        `<p>${m.downloads.length ? `${count(m.downloads.length, "download")} coming in at <b>${fmt(m.downMBps)} MB/s</b>.` : "No downloads right now."}</p>`,
        `<p>Drives hold <b>${fmt(used)} of ${size} TB</b>.</p>`,
      ],
    };
  }
  if (name === "storage") {
    const dl = m.disks[3];
    return {
      pill: tone(m.health.home),
      lines: [
        ...m.disks.map((d) => `<div class="row"><div class="row-head"><span>${d.label}</span><span>${d.used} / ${d.size} TB</span></div><p>${down ? "Not reachable." : d.writing > 0.05 ? (d.id === "d4" ? `Writing <b>${fmt(m.downMBps)} MB/s</b> of new downloads.` : "Taking in finished downloads.") : d.reading > 0.2 ? "Reading for a stream." : "Idle."}</p></div>`),
      ],
    };
  }
  if (name === "transcoder") {
    const tc = m.transcodes[0];
    return {
      pill: tone(m.health.home),
      lines: down ? ["<p>Not reachable.</p>"] : tc ? [
        `<p>Converting <b>${count(m.transcoder.sessions, "stream")}</b>: ${tc.title} from ${tc.from} to ${tc.to} for ${tc.who}'s ${tc.device}.</p>`,
        `<p>Running at <b>${m.transcoder.fps} frames a second</b>, about <b>${pct(m.transcoder.load)} percent</b> busy. Room for ${count(H.transcoder.maxSessions - m.transcoder.sessions, "more stream")}.</p>`,
      ] : [`<p>Idle. Every stream plays as it is, with no converting needed.</p>`],
    };
  }
  if (name === "torrent") {
    return {
      pill: tone(m.health.home),
      lines: down ? ["<p>Not reachable.</p>"] : [
        `<p>${m.downloads.length ? `${count(m.downloads.length, "download")} coming in at <b>${fmt(m.downMBps)} MB/s</b>.` : "No downloads right now."}</p>`,
        ...m.downloads.map((d) => `<div class="row"><div class="row-head"><span>${d.title}</span><span>${pct(d.done)}%</span></div><p>${fmt(d.MBps)} MB/s from ${d.peers} peers on ${d.tracker}.</p></div>`),
        `<p>Sharing <b>${m.seeding} finished torrents</b> back at <b>${fmt(m.seedUpMBps)} MB/s</b>.</p>`,
      ],
    };
  }
  if (name === "tunnel") {
    return {
      pill: tone(m.health.tunnel),
      lines: m.tunnel.latencyMs == null ? ["<p>Down. Nothing is getting through.</p>", "<p>The Relay Node keeps the far end open and waits.</p>"] : [
        `<p>Carrying <b>${fmt(m.tunnel.mbps)} Mbit/s</b> to Ashburn, <b>${pct(m.tunnel.share)} percent</b> of the home upload.</p>`,
        `<p>Each trip takes <b>${m.tunnel.latencyMs} ms</b>${m.health.tunnel === "slow" ? ", much slower than usual" : ""}.</p>`,
      ],
    };
  }
  if (name === "relay") {
    return {
      pill: tone(m.health.relay),
      lines: [
        `<p>${m.health.relay === "warn" ? "Plex has not confirmed remote access since 19:20. Remote viewers may fall back to Plex Relay." : down ? "Running, but waiting for the media server. Viewers see nothing new." : `Fronting <b>two sites</b> over a locked connection.`}</p>`,
        ...m.remote.map((s) => `<div class="row"><div class="row-head"><span>${s.who}</span><span>${fmt(s.mbps)} Mbit/s</span></div><p>Watching ${s.title} on a ${s.device}.</p></div>`),
        `<p>Using <b>${m.relay.cpu} percent</b> of its processor.</p>`,
      ],
    };
  }
}

let lastSample = null;
let lastPanel = 0;
let lastState = null;
function renderPanel(m) {
  if (!m) return;
  const p = panelOf(m, selected);
  ui.kicker.textContent = selected ? kickerOf(selected) : p.kicker;
  ui.title.textContent = selected ? titleOf(selected) : p.title;
  ui.pill.dataset.tone = p.pill[0];
  ui.pillText.textContent = p.pill[1];
  ui.lines.innerHTML = p.lines.join("");
}

for (const b of document.querySelectorAll("[data-state]")) b.addEventListener("click", () => setState(b.dataset.state));
function setState(id) {
  if (!STATES[id]) return;
  stateId = id;
  for (const b of document.querySelectorAll("[data-state]")) b.setAttribute("aria-pressed", String(b.dataset.state === id));
  sentence.textContent = STATES[id].sentence;
  renderPanel(sample(id, T));
  run();
}
window.addEventListener("message", (e) => e.data?.state && setState(e.data.state));
window.addEventListener("hashchange", () => setState(pick()));

let T = 40;
let frame = 0, last = 0, visible = true;
const iso3 = (p) => iso(p, P);

function tick(now) {
  const t0 = performance.now();
  const dt = last ? Math.min((now - last) / 1000, 1 / 30) : 1 / 60;
  last = now;
  calm = still.matches;
  const move = calm ? 0 : dt;
  T += dt;
  const m = sample(stateId, T);
  lastSample = m;
  const homeOn = ease("home", m.health.home === "down" ? 0 : 1, dt, 0.5);

  if (touring) {
    tourClock += dt;
    if (tourClock > 4) { tourClock = 0; const i = PARTS.indexOf(selected); select(PARTS[(i + 1) % PARTS.length]); }
  }

  const lat = m.tunnel.latencyMs;
  const eff = lat == null ? 0 : Math.min(m.tunnel.mbps, 0.5 / (lat / 1000));
  const capRate = ease("capRate", eff * 0.25, dt, 0.6);
  const capSpeed = ease("capSpeed", lat == null ? (E.capSpeed ?? 150) : 40 + 1900 / lat, dt, 0.6);
  if (calm) {
    const spacing = capRate > 0.05 ? capSpeed / capRate : Infinity;
    capsules.forEach((c, i) => { c.s = spacing === Infinity ? -1 : 12 + i * spacing; if (c.s > tunnel.len - 12) c.s = -1; });
  } else {
    capAcc += capRate * dt;
    while (capAcc >= 1) { capAcc -= 1; const c = capsules.find((c) => c.s < 0); if (c) c.s = 0; }
    for (const c of capsules) if (c.s >= 0) { c.s += capSpeed * move; if (c.s > tunnel.len) c.s = -1; }
  }
  for (const c of capsules) {
    if (c.s < 0) { if (c.vis) { c.g.setAttribute("opacity", 0); c.vis = false; } continue; }
    const a = iso3(tunnel.at(c.s - 3.2)), b = iso3(tunnel.at(c.s + 3.2));
    const d = `M${a[0].toFixed(2)} ${a[1].toFixed(2)}L${b[0].toFixed(2)} ${b[1].toFixed(2)}`;
    c.halo.setAttribute("d", d); c.core.setAttribute("d", d);
    c.g.setAttribute("opacity", (smooth(0, 8, c.s) * smooth(0, 8, tunnel.len - c.s)).toFixed(3));
    c.vis = true;
  }

  const downRate = ease("downRate", (m.downMBps / 6) * homeOn, dt, 0.6);
  const seedRate = ease("seedRate", (m.seedUpMBps / 2.5) * homeOn, dt, 0.6);
  const spawn = (tracks, kind) => {
    const p = pieces.find((p) => !p.alive);
    if (!p) return;
    const h = Math.floor(Math.random() * tracks.length);
    Object.assign(p, { alive: true, route: tracks[h], s: 0, kind, house: h, speed: 120 + Math.random() * 30 });
    if (kind === "seed") houseFlash.set(h, null);
    else houseFlash.set(h, T);
  };
  if (!calm) {
    downAcc += downRate * dt; seedAcc += seedRate * dt;
    while (downAcc >= 1) { downAcc -= 1; spawn(downTracks, "down"); }
    while (seedAcc >= 1) { seedAcc -= 1; spawn(seedTracks, "seed"); }
  }
  for (const p of pieces) {
    if (!p.alive) { if (p.vis) { p.el.setAttribute("opacity", 0); p.vis = false; } continue; }
    p.s += p.speed * move;
    if (p.s >= p.route.len) {
      p.alive = false;
      if (p.kind === "down") landInHopper(m);
      else houseFlash.set(p.house, T);
      continue;
    }
    const q = iso3(p.route.at(p.s));
    p.el.setAttribute("transform", `translate(${q[0].toFixed(2)} ${q[1].toFixed(2)})`);
    p.el.setAttribute("opacity", (smooth(0, 10, p.s) * smooth(0, 6, p.route.len - p.s)).toFixed(3));
    p.vis = true;
  }
  houseWins.forEach((w) => {
    const at = houseFlash.get(+w.dataset.houseWin);
    const v = at == null ? 0 : burst(T - at, 0.08, 0.5);
    w.style.fill = v > 0.01 ? `hsl(214 70% ${Math.round(5 + 60 * v)}%)` : "";
  });

  panels.forEach((panel, n) => {
    const dl = m.downloads[n];
    const present = ease("dl" + n, dl && homeOn > 0.5 ? 1 : 0, dt, 0.5);
    const done = dl ? dl.done : E["done" + n] ?? 0;
    E["done" + n] = done;
    const total = panel.cells.length;
    const filled = done * total;
    panel.cells.forEach((cell, i) => {
      const part = clamp(filled - i, 0, 1);
      const flashAt = panel.flash.get(i);
      const fl = flashAt == null ? 0 : burst(T - flashAt, 0.06, 0.35);
      const o = present * (part >= 1 ? 0.55 : part * 0.35) + fl * 0.6 * present;
      const key = o.toFixed(2);
      if (cell._o !== key) { cell.style.opacity = key; cell._o = key; }
    });
    const rail = panel.cols * (C.w + C.gap) - C.gap;
    panel.progress.setAttribute("transform", translateAlong(P, [panel.x0 + done * rail, 0, 0]));
    panel.progress.style.opacity = (0.3 + 0.7 * present).toFixed(2);
  });
  const active = m.downloads.length ? panels[nextPanel % Math.max(1, m.downloads.length)] : null;
  if (active) {
    const i = Math.min(active.cells.length - 1, Math.floor((m.downloads[panels.indexOf(active)]?.done ?? 0) * active.cells.length));
    ease("shuttle", cellCenter(active, i).u, dt, 0.35);
  } else ease("shuttle", DATA.board.hopperX, dt, 0.8);
  shuttle.setAttribute("transform", translateAlong(P, [E.shuttle, 0, 0]));
  for (let j = dropEls.length - 1; j >= 0; j--) {
    const d = dropEls[j];
    d.age += move;
    const z = d.z0 - 0.5 * 520 * d.age * d.age;
    if (z <= d.z1) {
      d.panel.flash.set(d.cell, T);
      d.el.remove();
      dropEls.splice(j, 1);
      continue;
    }
    d.el.setAttribute("y", (-z - 1.6).toFixed(2));
  }

  const fps = (m.transcoder.fps ?? 0) * homeOn;
  const inSpeed = ease("press", (fps / 24) * 10, dt, 0.5);
  stripIn = (stripIn + inSpeed * move) % 1000;
  stripOut = (stripOut + inSpeed * 0.6 * move) % 1000;
  rollAngle += (inSpeed / DATA.press.rollR) * move;
  const step = (Math.PI * 2) / DATA.press.ribCount;
  const fi = Math.floor(((rollAngle % step) / step) * DATA.press.ribFrames);
  for (const [name, dir] of [["low", 1], ["high", -1]]) {
    const want = dir > 0 ? fi : (DATA.press.ribFrames - fi) % DATA.press.ribFrames;
    if (rollShown[name] !== want) {
      rollFrames[name][rollShown[name]].style.display = "none";
      rollFrames[name][want].style.display = "";
      rollShown[name] = want;
    }
  }
  const deg = (rollAngle * 180) / Math.PI;
  gears.low.setAttribute("transform", `rotate(${(-deg).toFixed(2)} ${DATA.press.nipX} ${-DATA.press.low})`);
  gears.high.setAttribute("transform", `rotate(${deg.toFixed(2)} ${DATA.press.nipX} ${-DATA.press.high})`);
  const sessions = homeOn > 0.5 ? m.transcoder.sessions : 0;
  lanes.forEach((lane, i) => {
    const v = ease("lane" + i, i < sessions ? 1 : 0, dt, 0.45);
    lane.style.opacity = v.toFixed(2);
    laneLamps[i].toggleAttribute("data-on", i < sessions);
  });
  for (const mk of marks) mk.el.setAttribute("transform", translateAlong(P, [mk.name === "in" ? stripIn % mk.pitch : stripOut % mk.pitch, 0, 0]));
  const load = ease("load", m.transcoder.load * homeOn, dt, 0.5);
  needle.setAttribute("transform", `rotate(${(-120 + 240 * load).toFixed(1)} ${DATA.loadDial[0]} ${DATA.loadDial[1]})`);
  const fpsTextNow = fps > 1 ? String(Math.round(fps)) : "--";
  if (fpsText.textContent !== fpsTextNow) fpsText.textContent = fpsTextNow;

  drawers.forEach((d, i) => {
    const disk = m.disks[i];
    const reading = ease("rd" + i, disk.reading, dt, 0.35);
    const writing = ease("wr" + i, disk.writing, dt, 0.35);
    const pulse = 0.5 + 0.5 * Math.sin(T * 5.2);
    const open = DATA.drawerOpen * clamp(reading, 0, 1) + writing * 1.6 * pulse;
    d.el.setAttribute("transform", translateAlong(P, [0, open, 0]));
    const flick = 0.55 + 0.45 * Math.sin(T * 31 + i * 7.1) * Math.sin(T * 13.7 + i * 2.3);
    const v = clamp(Math.max(reading * flick, writing * (0.35 + 0.65 * pulse)), 0, 1);
    d.led.style.opacity = (0.15 + 0.85 * v).toFixed(2);
    d.led.style.fill = v > 0.1 ? "var(--anatomy-lit)" : "";
  });

  const set = (el, v, color) => { if (!el) return; el.style.opacity = v.toFixed(2); if (color) el.style.fill = color; };
  const homeLamp = ease("lampHome", 1, dt);
  const tunnelState = m.health.tunnel;
  set(lamps.home, homeLamp, m.health.home === "down" ? "var(--fail)" : "var(--ok)");
  const blink = 0.55 + 0.45 * Math.sin(T * 3.2);
  set(lamps.relay, tunnelState === "ok" ? 1 : tunnelState === "slow" ? 1 : blink, tunnelState === "ok" ? "var(--ok)" : tunnelState === "slow" ? "var(--warn)" : "var(--fail)");
  set(lamps.ok, ease("tOk", m.health.relay === "ok" ? 1 : 0, dt), "var(--ok)");
  set(lamps.warn, ease("tWarn", m.health.relay === "warn" ? 1 : 0, dt) + (m.health.home === "down" ? 0.5 + 0.5 * Math.sin(T * 2.4) : 0) * 0.9, "var(--warn)");
  set(lamps.fail, 0, "var(--fail)");
  const rxv = clamp(downRate / 5, 0, 1);
  set(leds.rx, 0.2 + 0.8 * rxv * (0.6 + 0.4 * Math.sin(T * 17)));
  if (leds.rx) leds.rx.style.fill = rxv > 0.1 ? "var(--anatomy-lit)" : "";
  set(leds.out, 0.2 + 0.8 * clamp(inSpeed / 20, 0, 1) * (0.6 + 0.4 * Math.sin(T * 21)));
  set(leds.plex, 0.25 + 0.75 * homeOn);
  leds.trackers.forEach((t, i) => set(t, 0.25 + 0.75 * homeOn * (0.5 + 0.5 * Math.sin(T * 1.7 + i * 2))));
  if (leds.beacon) leds.beacon.style.opacity = (0.5 + 0.5 * homeOn * (0.5 + 0.5 * Math.sin(T * 2))).toFixed(2);
  const cpuOn = Math.round((m.relay.cpu / 40) * cpuCells.length);
  cpuCells.forEach((c, i) => c.toggleAttribute("data-on", i < cpuOn));
  portLeds.forEach((p, i) => { const v = i < m.remote.length + 1 ? 0.4 + 0.6 * Math.abs(Math.sin(T * (9 + i * 3.1))) : 0.15; p.style.fill = i < m.remote.length + 1 ? "var(--glow)" : ""; p.style.opacity = v.toFixed(2); });
  const tlsOn = ease("tls", m.relay.tls > 0 ? 1 : 0.35, dt);
  tls.forEach((t) => (t.style.opacity = tlsOn.toFixed(2)));

  const who = (name) => m.streams.find((s) => s.who === name);
  const screenLevel = { home: who("Living room") ? 1 : 0, tv: who("Mom") ? 1 : 0, phone: who("Sam") ? 1 : 0, laptop: 0 };
  for (const [name, s] of Object.entries(screens)) {
    const v = ease("scr" + name, screenLevel[name] * homeOn, dt, 0.45);
    s.off.style.opacity = (1 - v * 0.92).toFixed(2);
    s.pan.setAttribute("transform", `translate(${((T * 1.4) % 20 - 10).toFixed(2)} 0)`);
    s.bar.setAttribute("width", (+s.bar.dataset.w * (((T / 90) + name.length * 0.13) % 1)).toFixed(2));
  }
  if (leds.tv) set(leds.tv, screenLevel.home ? 0.2 : 0.8, screenLevel.home ? "" : "var(--fail)");

  const flowLevel = { tv: E.scrtv ?? 0, phone: E.scrphone ?? 0, laptop: 0, plex: Math.max(E.scrtv ?? 0, E.scrphone ?? 0) };
  for (const [name, f] of Object.entries(flows)) {
    const lvl = flowLevel[name] ?? 0;
    const spacing = 14;
    f.phase += 34 * move;
    f.dots.forEach((dot, i) => {
      const s = (f.phase + i * spacing) % (spacing * f.dots.length);
      if (s > f.tr.len || lvl < 0.02) { dot.setAttribute("opacity", 0); return; }
      const q = iso3(f.tr.at(s));
      dot.setAttribute("cx", q[0].toFixed(2)); dot.setAttribute("cy", q[1].toFixed(2));
      dot.setAttribute("opacity", (lvl * smooth(0, 4, s) * smooth(0, 4, f.tr.len - s)).toFixed(2));
    });
  }

  if (now - lastPanel > 250 || lastState !== stateId) {
    lastPanel = now;
    lastState = stateId;
    const text = `↓ ${fmt(m.downMBps)} MB/s · ↑ ${fmt(m.tunnel.mbps)} Mbit/s to ${plural(m.relay.viewers, "viewer")}`;
    if (readout.textContent !== text) readout.textContent = text;
    renderPanel(m);
  }
  window.__tickMs = (window.__tickMs ?? 0) * 0.95 + (performance.now() - t0) * 0.05;
  frame = visible && !calm ? requestAnimationFrame(tick) : 0;
}

function landInHopper(m) {
  if (!m.downloads.length) return;
  const n = nextPanel % m.downloads.length;
  nextPanel++;
  const panel = panels[n];
  const i = Math.min(panel.cells.length - 1, Math.floor(m.downloads[n].done * panel.cells.length));
  const c = cellCenter(panel, i);
  const el = make("rect", { class: "drop", x: (c.u - 1.6).toFixed(2), y: "0", width: 3.2, height: 3.2, rx: 0.6 }, panel.drops);
  dropEls.push({ el, panel, cell: i, age: 0, z0: DATA.board.beam - 4, z1: c.z + C.h / 2 - 1 });
}

const run = () => { if (!frame && visible) { last = 0; frame = requestAnimationFrame(tick); } };
new IntersectionObserver((entries) => { visible = entries[entries.length - 1].isIntersecting; if (visible) run(); }, { rootMargin: "120px 0px" }).observe(stage);
still.addEventListener?.("change", () => { calm = still.matches; touring = !calm; run(); });
renderDetails();
setState(stateId);
run();
