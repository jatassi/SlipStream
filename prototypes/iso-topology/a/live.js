const D = __DATA__;
const P = D.P;
const stage = document.querySelector(".stage");
const svg = stage.querySelector("svg");
const $ = (s) => svg.querySelector(s);
const $$ = (s) => [...svg.querySelectorAll(s)];
const at = (p) => iso(p, P);
const still = matchMedia("(prefers-reduced-motion: reduce)");
const readout = document.querySelector("[data-readout]");
const sentenceEl = document.querySelector("[data-sentence]");
const sentenceText = document.querySelector("[data-sentence-text]");
const panel = document.querySelector("[data-panel]");
const stateButtons = [...document.querySelectorAll("[data-state]")];

const smooth = (a, b, x) => {
  const t = Math.min(1, Math.max(0, (x - a) / (b - a)));
  return t * t * (3 - 2 * t);
};
const burst = (age, rise, fall) => (age < 0 ? 0 : age < rise ? smooth(0, rise, age) : Math.exp(-(age - rise) / fall));
const follow = (x, target, dt, tau, calm) => (calm ? target : x + (target - x) * (1 - Math.exp(-dt / tau)));
const lerp = (a, b, t) => a + (b - a) * t;
const lerp3 = (a, b, t) => [lerp(a[0], b[0], t), lerp(a[1], b[1], t), lerp(a[2], b[2], t)];
const DIM = [50, 55, 61];
const WHITE = [255, 255, 255];
const BLUE = [116, 175, 251];
const mix = (c, t) => `rgb(${Math.round(lerp(DIM[0], c[0], t))},${Math.round(lerp(DIM[1], c[1], t))},${Math.round(lerp(DIM[2], c[2], t))})`;

const validState = (s) => (s && STATES[s] ? s : null);
const params = new URLSearchParams(location.search);
let stateId = validState(params.get("state")) || validState(location.hash.slice(1).replace(/^state=/, "")) || "busy";

const ease = {
  power: 1, tunnel: 1, fan: 0.4, rear: 0.5, lamps: [0, 0, 0, 0], read: [0, 0, 0, 0], write: [0, 0, 0, 0],
  devices: Object.fromEntries([...D.devices.map((d) => d.id), "lrtv"].map((id) => [id, 0])),
  regions: [0, 0], frac: [0, 0],
};
const clock = { t: 0, fan: 0, rear: 0, pan: 0 };
let S = sample(stateId, 0);

const el = {
  cpufan: $('[data-part="cpufan"]'),
  cpublur: $('[data-part="cpufan-blur"]'),
  rearfan: $('[data-part="rearfan"]'),
  rearblur: $('[data-part="rearfan-blur"]'),
  power: $('[data-part="power-led"]'),
  lamps: [0, 1, 2, 3].map((i) => $(`[data-lamp="${i}"]`)),
  reads: [0, 1, 2, 3].map((i) => $(`[data-disk-read="${i}"]`)),
  writes: [0, 1, 2, 3].map((i) => $(`[data-disk-write="${i}"]`)),
  router: $$("[data-router-led]"),
  net: $$("[data-net-led]"),
  peers: $$("[data-peer]"),
  relayCpu: $('[data-relay-led="cpu"]'),
  relayNet: $('[data-relay-led="net"]'),
  relayTls: $('[data-relay-led="tls"]'),
  badges: $$("[data-badge]"),
  status: Object.fromEntries($$("[data-status]").map((n) => [n.dataset.status, n])),
  cells: [],
  caps: $('[data-layer="caps"]'),
  flow: $('[data-layer="flow"]'),
  pieces: $('[data-layer="pieces"]'),
  screens: Object.fromEntries($$("[data-screen]").map((n) => [n.dataset.screen, n])),
  pans: Object.fromEntries($$("[data-pan]").map((n) => [n.dataset.pan, n])),
  progs: Object.fromEntries($$("[data-prog]").map((n) => [n.dataset.prog, n])),
  standby: Object.fromEntries($$("[data-standby]").map((n) => [n.dataset.standby, n])),
  items: $$(".it[data-res]"),
};
for (const n of $$("[data-cell]")) el.cells[Number(n.dataset.cell)] = n;

const rotateIn = (node, deg) => node.setAttribute("transform", `translate(${node.dataset.cx} ${node.dataset.cy}) rotate(${deg.toFixed(2)})`);

const TR = D.tray;
const glowEllipse = (() => {
  const [cx, cy] = at([0, 0, 0]);
  return `<ellipse class="t-glow" cx="${cx.toFixed(1)}" cy="${cy.toFixed(1)}" rx="8" ry="4"/>`;
})();
const capPool = Array.from({ length: 60 }, () => {
  const g = document.createElementNS("http://www.w3.org/2000/svg", "g");
  g.setAttribute("class", "t-cap");
  g.style.display = "none";
  g.innerHTML = glowEllipse + `<g data-axis="x">${D.capsule}</g><g data-axis="y" style="display:none">${D.capsuleY}</g>`;
  el.caps.appendChild(g);
  return g;
});
const piecePool = Array.from({ length: 40 }, () => {
  const g = document.createElementNS("http://www.w3.org/2000/svg", "g");
  g.setAttribute("class", "t-piece");
  g.style.display = "none";
  g.innerHTML = D.piece;
  el.pieces.appendChild(g);
  return g;
});
const makeDots = (parent, n) => Array.from({ length: n }, () => {
  const c = document.createElementNS("http://www.w3.org/2000/svg", "circle");
  c.setAttribute("class", "t-dot");
  c.setAttribute("r", "1.1");
  c.style.display = "none";
  parent.appendChild(c);
  return c;
});
const dotPools = { relay: makeDots(el.flow, 40), home: makeDots($('[data-layer="flow-home"]'), 16) };

const polyOf = (pts) => {
  const lengths = [0];
  for (let i = 1; i < pts.length; i++) lengths.push(lengths[i - 1] + Math.hypot(pts[i][0] - pts[i - 1][0], pts[i][1] - pts[i - 1][1], pts[i][2] - pts[i - 1][2]));
  const total = lengths[lengths.length - 1];
  return (u) => {
    const s = Math.max(0, Math.min(1, u)) * total;
    let i = 1;
    while (i < pts.length - 1 && lengths[i] < s) i++;
    const t = (s - lengths[i - 1]) / (lengths[i] - lengths[i - 1] || 1);
    return lerp3(pts[i - 1], pts[i], t);
  };
};
const wireAt = polyOf(D.wire);
const laneAt = { video: polyOf(D.lanes.video), req: polyOf(D.lanes.req) };
const laneAxis = (lane, p) => {
  const pts = D.lanes[lane];
  const corner = pts[1];
  const along = lane === "video" ? p[0] < corner[0] - 0.01 : p[1] > corner[1] + 0.01;
  return along ? "x" : "y";
};
const outAt = Object.fromEntries(Object.entries(D.out).map(([id, pts]) => [id, polyOf(pts.map((p) => [p[0], p[1], p[2] + 1.2]))]));

let caps = [];
let pieces = [];
let dots = [];
const acc = { video: 0.5, req: 0.3, down: [0, 0], seed: 0, dots: {} };
const led = { read: [-9, -9, -9, -9], write: [-9, -9, -9, -9], router: [-9, -9, -9, -9, -9], peer: [-9, -9, -9], net: -9, cpu: -9, tls: -9, badge: [-9, -9] };
const cellPop = new Map();
const shownCells = [new Set(), new Set()];

const latencyOf = () => S.tunnel.latencyMs ?? 20;
const travel = () => 1.3 + latencyOf() / 45;
const videoRate = () => (S.tunnel.mbps / 2.2) * Math.min(1, 30 / latencyOf()) * ease.tunnel;
const reqRate = () => (S.health.tunnel === "down" ? 0 : (0.5 + S.remote.length * 0.6) * Math.min(1, 30 / latencyOf()));

function spawnCap(lane, age = 0) {
  caps.push({ lane, born: clock.t - age, dur: travel() });
}
function drawCaps() {
  let i = 0;
  caps = caps.filter((c) => (clock.t - c.born) / c.dur < 1);
  for (const c of caps) {
    if (i >= capPool.length) break;
    const u = (clock.t - c.born) / c.dur;
    const p = laneAt[c.lane](u);
    const g = capPool[i++];
    g.style.display = "";
    const axis = laneAxis(c.lane, p);
    if (g.dataset.axis !== axis) {
      g.dataset.axis = axis;
      g.children[1].style.display = axis === "x" ? "" : "none";
      g.children[2].style.display = axis === "y" ? "" : "none";
    }
    g.setAttribute("transform", translateAlong(P, p));
    g.style.opacity = (smooth(0, 0.05, u) * smooth(1, 0.95, u) * (c.lane === "req" ? 0.75 : 1)).toFixed(2);
  }
  for (; i < capPool.length; i++) if (capPool[i].style.display !== "none") capPool[i].style.display = "none";
}

function frontierCell(region) {
  const dl = S.downloads[region];
  if (!dl) return null;
  const filled = Math.floor(dl.done * 48);
  return D.order[region][Math.min(47, filled)];
}
function spawnPiece(region) {
  const peer = Math.floor(Math.random() * 3);
  led.peer[peer] = clock.t;
  pieces.push({ kind: "in", region, born: clock.t, wire: 1.1, hop: 0.42 });
}
function spawnSeed() {
  pieces.push({ kind: "out", born: clock.t, wire: 1.6 });
}
function drawPieces() {
  let i = 0;
  const keep = [];
  for (const p of pieces) {
    const age = clock.t - p.born;
    let pos = null;
    let scale = 1;
    if (p.kind === "in") {
      if (age < p.wire) pos = wireAt(smooth(0, 1, age / p.wire) * 0.6 + (age / p.wire) * 0.4);
      else if (age < p.wire + p.hop) {
        const u = (age - p.wire) / p.hop;
        const target = p.target ?? (p.target = frontierCell(p.region));
        if (target == null) continue;
        const to = D.cells[target].center;
        const from = D.boardHop;
        const flat = lerp3(from, [to[0], to[1], to[2] + 3.6], u);
        pos = [flat[0], flat[1], flat[2] + 18 * 4 * u * (1 - u)];
      } else {
        const target = p.target ?? frontierCell(p.region);
        if (target != null) cellPop.set(target, clock.t);
        continue;
      }
    } else {
      if (age > p.wire) continue;
      pos = wireAt(1 - age / p.wire);
      scale = 0.8;
    }
    keep.push(p);
    if (i >= piecePool.length) continue;
    const g = piecePool[i++];
    g.style.display = "";
    g.setAttribute("class", p.kind === "in" ? "t-piece" : "t-piece t-seed");
    const [dx, dy] = axisVector(P, pos);
    const [ox, oy] = P.origin;
    g.setAttribute("transform", scale !== 1 ? `translate(${(dx + ox).toFixed(2)} ${(dy + oy).toFixed(2)}) scale(${scale}) translate(${-ox} ${-oy})` : `translate(${dx.toFixed(2)} ${dy.toFixed(2)})`);
  }
  pieces = keep;
  for (; i < piecePool.length; i++) if (piecePool[i].style.display !== "none") piecePool[i].style.display = "none";
}

function drawCells(calm) {
  for (const region of [0, 1]) {
    const dl = S.downloads[region];
    const target = dl && ease.power > 0.5 ? Math.floor(dl.done * 48) : 0;
    const frontier = dl && ease.power > 0.5 ? D.order[region][Math.min(47, target)] : null;
    const shown = shownCells[region];
    for (let n = 0; n < 48; n++) {
      const id = D.order[region][n];
      const node = el.cells[id];
      const want = n < target;
      if (want && !shown.has(id)) {
        shown.add(id);
        if (!cellPop.has(id)) cellPop.set(id, clock.t);
      }
      if (!want && shown.has(id)) {
        shown.delete(id);
        node.dataset.leave = String(clock.t + (n % 12) * 0.03);
      }
      let opacity = 0;
      let lift = 0;
      if (shown.has(id)) {
        const age = calm ? 9 : clock.t - (cellPop.get(id) ?? -9);
        opacity = 0.75;
        lift = age < 0.35 ? 5 * (1 - smooth(0, 0.35, age)) - 1.2 * Math.sin(Math.PI * smooth(0.2, 0.35, age)) : 0;
        opacity = 0.75 + 0.25 * burst(age, 0.05, 0.35);
      } else if (id === frontier) {
        const age = clock.t - (cellPop.get(id) ?? -9);
        opacity = 0.18 + 0.4 * (dl.done * 48 - target) + 0.4 * burst(age, 0.04, 0.25);
      } else if (node.dataset.leave) {
        const age = clock.t - Number(node.dataset.leave);
        opacity = 0.75 * (1 - smooth(0, 0.5, age));
        lift = age > 0 ? smooth(0, 0.5, age) * 4 : 0;
        if (age > 0.5) delete node.dataset.leave;
      }
      const o = opacity.toFixed(2);
      if (node.style.opacity !== o) node.style.opacity = o;
      const tr = lift ? translateAlong(P, [0, 0, lift]) : "";
      if ((node.getAttribute("transform") || "") !== tr) tr ? node.setAttribute("transform", tr) : node.removeAttribute("transform");
    }
  }
}

function spawnDot(id, mbps) {
  dots.push({ id, born: clock.t, dur: id === "lrtv" ? 0.9 : 1.1 + latencyOf() / 200 });
}
function drawDots() {
  const used = { relay: 0, home: 0 };
  dots = dots.filter((d) => clock.t - d.born < d.dur);
  for (const d of dots) {
    const pool = d.id === "lrtv" ? "home" : "relay";
    const dotPool = dotPools[pool];
    if (used[pool] >= dotPool.length) continue;
    const u = (clock.t - d.born) / d.dur;
    const [x, y] = at(outAt[d.id](u));
    const c = dotPool[used[pool]++];
    c.style.display = "";
    c.setAttribute("cx", x.toFixed(1));
    c.setAttribute("cy", y.toFixed(1));
    c.style.opacity = (smooth(0, 0.08, u) * smooth(1, 0.9, u)).toFixed(2);
  }
  for (const pool of ["relay", "home"]) for (let i = used[pool]; i < dotPools[pool].length; i++) if (dotPools[pool][i].style.display !== "none") dotPools[pool][i].style.display = "none";
}

const streamFor = (id) => {
  if (id === "lrtv") return S.streams.find((s) => s.where === "home");
  const d = D.devices.find((x) => x.id === id);
  if (!d || d.who < 0) return null;
  return S.streams.find((s) => s.where === "remote" && s.device === (id === "tv" ? "TV" : "phone"));
};

const HEALTH_HUE = { ok: "ok", warn: "warn", slow: "warn", down: "down" };
function drawStatus() {
  const h = S.health;
  const set = (n, v) => n && n.dataset.h !== v && (n.dataset.h = v);
  set(el.status.home, HEALTH_HUE[h.tunnel]);
  set(el.status.relay, HEALTH_HUE[h.relay]);
  set(el.status["relay-end"], h.tunnel === "down" ? "idle" : HEALTH_HUE[h.tunnel]);
  if (h.tunnel === "down" && el.status["relay-end"]) el.status["relay-end"].style.fill = mix(WHITE, 0.25 + 0.35 * (0.5 + 0.5 * Math.sin(clock.t * 2.2)));
  else if (el.status["relay-end"]) el.status["relay-end"].style.fill = "";
}

function tick(dt, calm) {
  clock.t += dt;
  S = sample(stateId, clock.t);
  const up = S.health.home !== "down";
  ease.power = follow(ease.power, up ? 1 : 0, dt, up ? 0.5 : 0.9, calm);
  ease.tunnel = follow(ease.tunnel, S.health.tunnel === "down" ? 0 : 1, dt, 0.6, calm);
  const fanTarget = up ? 0.12 + 1.9 * S.transcoder.load : 0;
  ease.fan = follow(ease.fan, fanTarget, dt, up ? 0.9 : 1.4, calm);
  ease.rear = follow(ease.rear, up ? 0.55 : 0, dt, up ? 0.8 : 1.6, calm);
  if (!calm) {
    clock.fan = (clock.fan + ease.fan * 360 * dt) % 360;
    clock.rear = (clock.rear + ease.rear * 360 * dt) % 360;
  }
  rotateIn(el.cpufan, clock.fan);
  rotateIn(el.rearfan, clock.rear);
  el.cpublur.setAttribute("opacity", (Math.min(1, ease.fan / 2) * 0.16).toFixed(3));
  el.rearblur.setAttribute("opacity", (ease.rear * 0.06).toFixed(3));
  const pw = ease.power;
  el.power.style.fill = mix(WHITE, 0.95 * pw);

  for (let i = 0; i < 4; i++) {
    ease.lamps[i] = follow(ease.lamps[i], i < S.transcoder.sessions ? 1 : 0, dt, 0.35, calm);
    const flick = calm ? 0 : 0.08 * Math.sin(clock.t * 23 + i * 2) * ease.lamps[i];
    el.lamps[i].style.fill = mix(WHITE, Math.max(0, ease.lamps[i] * pw * (0.85 + 0.15 * S.transcoder.load) + flick));
  }

  S.disks.forEach((d, i) => {
    if (!calm) {
      if (Math.random() < d.reading * 9 * dt) led.read[i] = clock.t;
      if (Math.random() < d.writing * 9 * dt) led.write[i] = clock.t;
    }
    const r = calm ? d.reading * 0.7 : burst(clock.t - led.read[i], 0.025, 0.09);
    const w = calm ? d.writing * 0.7 : burst(clock.t - led.write[i], 0.025, 0.09);
    el.reads[i].style.opacity = (r * pw).toFixed(2);
    el.writes[i].style.opacity = (w * pw).toFixed(2);
  });

  const traffic = (S.tunnel.mbps + S.downMBps * 0.3 + S.streams.filter((s) => s.where === "home").reduce((a, s) => a + s.mbps, 0) * 0.3) * pw;
  el.router.forEach((n, i) => {
    if (!calm && Math.random() < traffic * 0.12 * dt * (i === 4 ? 1.4 : 1)) led.router[i] = clock.t;
    n.style.fill = mix(i === 0 ? WHITE : BLUE, i === 0 ? 0.85 * pw : calm ? 0.5 * Math.min(1, traffic / 20) : burst(clock.t - led.router[i], 0.02, 0.07));
  });
  el.net[0].style.fill = mix(WHITE, 0.8 * pw);
  if (!calm && Math.random() < traffic * 0.2 * dt) led.net = clock.t;
  el.net[1].style.fill = mix(BLUE, calm ? 0.4 * pw : burst(clock.t - led.net, 0.02, 0.07));
  el.peers.forEach((n, i) => (n.style.fill = mix(BLUE, 0.2 + 0.8 * burst(clock.t - led.peer[i], 0.02, 0.18))));

  const relayLoad = S.relay.cpu / 100;
  if (!calm && Math.random() < relayLoad * 25 * dt) led.cpu = clock.t;
  el.relayCpu.style.fill = mix(WHITE, calm ? 0.6 : 0.35 + 0.65 * burst(clock.t - led.cpu, 0.02, 0.08));
  el.relayNet.style.fill = mix(BLUE, calm ? 0.5 * ease.tunnel : burst(clock.t - led.net, 0.02, 0.07) * ease.tunnel);
  if (!calm && Math.random() < S.relay.tls * 0.25 * dt) led.tls = clock.t;
  el.relayTls.style.fill = mix(WHITE, 0.25 + 0.75 * (calm ? 0.4 : burst(clock.t - led.tls, 0.04, 0.3)));
  el.badges.forEach((n, i) => {
    if (!calm && Math.random() < (i === 1 ? S.remote.length * 1.2 : 0.6) * ease.tunnel * dt) led.badge[i] = clock.t;
    n.style.fill = mix(WHITE, 0.3 + 0.7 * (calm ? 0.3 : burst(clock.t - led.badge[i], 0.02, 0.12)));
  });
  drawStatus();

  if (calm) {
    caps = [];
    const n = Math.round(videoRate() * travel());
    for (let i = 0; i < n; i++) caps.push({ lane: "video", born: clock.t - ((i + 0.5) / n) * travel(), dur: travel() });
    const m = Math.round(reqRate() * travel());
    for (let i = 0; i < m; i++) caps.push({ lane: "req", born: clock.t - ((i + 0.3) / m) * travel(), dur: travel() });
  } else {
    acc.video += videoRate() * dt;
    while (acc.video >= 1) { acc.video -= 1 + (Math.random() - 0.5) * 0.3; spawnCap("video"); }
    acc.req += reqRate() * dt;
    while (acc.req >= 1) { acc.req -= 1 + (Math.random() - 0.5) * 0.4; spawnCap("req"); }
  }
  drawCaps();

  if (!calm) {
    S.downloads.forEach((dl, n) => {
      acc.down[n] += (dl.MBps / 7) * pw * dt;
      while (acc.down[n] >= 1) { acc.down[n] -= 1; spawnPiece(n); }
    });
    acc.seed += (S.seedUpMBps / 2.4) * pw * dt;
    while (acc.seed >= 1) { acc.seed -= 1; spawnSeed(); }
  } else pieces = [];
  drawPieces();
  drawCells(calm);

  for (const id of Object.keys(ease.devices)) {
    const s = streamFor(id);
    ease.devices[id] = follow(ease.devices[id], s ? 1 : 0, dt, 0.5, calm);
    const scr = el.screens[id];
    if (scr) scr.setAttribute("opacity", ease.devices[id].toFixed(2));
    if (el.pans[id]) el.pans[id].setAttribute("transform", `translate(${(Math.sin(clock.t * 0.13 + id.length) * 14).toFixed(2)} 0)`);
    const prog = el.progs[id];
    if (prog) prog.setAttribute("width", (Number(prog.dataset.w) * ((0.18 + id.length * 0.11 + clock.t * 0.002) % 1)).toFixed(2));
    if (el.standby[id]) el.standby[id].style.fill = s ? mix(WHITE, 0.15) : mix(WHITE, 0.55);
    if (s && !calm) {
      acc.dots[id] = (acc.dots[id] ?? 0) + (s.mbps / (id === "lrtv" ? 6 : 3)) * (id === "lrtv" ? ease.power : ease.tunnel) * dt;
      while (acc.dots[id] >= 1) { acc.dots[id] -= 1; spawnDot(id, s.mbps); }
    }
  }
  if (calm) dots = [];
  drawDots();
}

const NUMBER = ["no", "one", "two", "three", "four", "five", "six"];
const words = (n) => NUMBER[n] ?? String(n);
const plural = (n, one, many) => `${words(n)} ${n === 1 ? one : many}`;
const pct = (x) => `${Math.round(x * 100)}%`;
const TB = (x) => `${x.toFixed(1)} TB`;

function overall() {
  const home = S.health.home === "down" ? "Offline since 21:31. Nothing is streaming or downloading." : `Running. Serving ${plural(S.streams.length, "stream", "streams")}${S.downloads.length ? `, downloading ${plural(S.downloads.length, "file", "files")}` : ""}.`;
  const relay = S.health.home === "down" ? "Running, waiting for the media server to come back." : S.health.relay === "warn" ? `Plex has not confirmed remote access. Forwarding ${plural(S.remote.length, "viewer", "viewers")} on a slow tunnel.` : `Running. Forwarding ${plural(S.remote.length, "remote viewer", "remote viewers")} through the tunnel.`;
  return {
    eyebrow: "Overview",
    title: "Two Nodes",
    body: `<ul class="rows"><li><span class="dot" data-h="${S.health.home === "down" ? "down" : "ok"}"></span><span><strong>Media server</strong> at home. ${home}</span></li><li><span class="dot" data-h="${S.health.relay}"></span><span><strong>Relay Node</strong> in Ashburn. ${relay}</span></li></ul>`,
    details: [["Media server", NODES.home.host], ["Relay Node", NODES.relay.host], ["Tunnel", `${TUNNEL.name} · UDP ${TUNNEL.port}`]],
    hint: "Hover a part of either machine, or focus the drawing and use the arrow keys.",
  };
}

function describe(res) {
  const h = NODES.home;
  const r = NODES.relay;
  const down = S.health.home === "down";
  if (res === "home") return {
    eyebrow: "Node · Controller",
    title: "Media server",
    body: down ? `<p>Offline. The Relay Node last heard from it at 21:31. Its drives, transcoder and downloads are all stopped.</p>` :
      `<p>Running at home. Serving <span class="num">${words(S.streams.length)}</span> ${S.streams.length === 1 ? "stream" : "streams"}${S.transcodes.length ? `, ${words(S.transcodes.length)} of them converted on the fly` : ""}.</p><p>${S.downloads.length ? `Downloading ${plural(S.downloads.length, "file", "files")} at <span class="num">${S.downMBps.toFixed(1)} MB/s</span> and seeding` : "Seeding"} <span class="num">${S.seeding}</span> finished downloads.</p>`,
    details: [["Host", h.host], ["System", h.os], ["Hardware", h.hw], ["Address", h.addr], ["Role", h.kind]],
  };
  if (res === "storage") {
    const used = S.disks.reduce((a, d) => a + d.used, 0);
    const size = S.disks.reduce((a, d) => a + d.size, 0);
    const dl = S.disks[3];
    return {
      eyebrow: "Media server · Resource",
      title: "Storage",
      body: `<p>Four drives, <span class="num">${Math.round(used)} of ${size} TB</span> used.${!down && dl.writing > 0.05 ? ` Writing <span class="num">${(S.downMBps).toFixed(1)} MB/s</span> to the Downloads drive.` : down ? " The drives are spun down." : ""}</p><ul class="rows">${S.disks.map((d) => `<li><span class="dot" data-h="${down ? "idle" : d.used / d.size > 0.82 ? "warn" : "ok"}"></span><span><strong>${d.label}</strong> <span class="num">${TB(d.used)}</span> of ${d.size} TB${down ? "" : d.writing > 0.05 ? ", writing" : d.reading > 0.1 ? ", reading" : ", quiet"}</span></li>`).join("")}</ul>`,
      details: S.disks.map((d) => [d.id, `${d.label} · ${pct(d.used / d.size)} full`]),
    };
  }
  if (res === "transcoder") {
    const t = S.transcodes[0];
    return {
      eyebrow: "Media server · Resource",
      title: "Transcoder",
      body: down ? `<p>Stopped with the media server.</p>` : t ? `<p>Converting <span class="num">${words(S.transcoder.sessions)}</span> ${S.transcoder.sessions === 1 ? "stream" : "streams"}: ${t.title} for ${t.who}, ${t.from} to ${t.to} at <span class="num">${S.transcoder.fps} fps</span>.</p><p>The graphics chip is <span class="num">${pct(S.transcoder.load)}</span> busy. The lamps under the cooler count the sessions, up to four.</p>` : `<p>Idle. Every stream plays in its original format, so nothing needs converting.</p>`,
      details: [["Engine", h.transcoder.engine], ["Sessions", `${S.transcoder.sessions} of ${h.transcoder.maxSessions}`]],
    };
  }
  if (res === "torrent") return {
    eyebrow: "Media server · Resource",
    title: "Torrent engine",
    body: down ? `<p>Stopped with the media server. Nothing is downloading or seeding.</p>` : `${S.downloads.length ? `<p>Downloading ${plural(S.downloads.length, "file", "files")} at <span class="num">${S.downMBps.toFixed(1)} MB/s</span>. Each square on the board is a slice of the file; pieces arrive from peers and drop into place.</p><ul class="rows">${S.downloads.map((d) => `<li><span class="dot"></span><span><strong>${d.title}</strong> <span class="num">${pct(d.done)}</span> from ${d.peers} peers</span></li>`).join("")}</ul>` : `<p>Nothing downloading.</p>`}<p>Seeding <span class="num">${S.seeding}</span> finished downloads at <span class="num">${S.seedUpMBps.toFixed(1)} MB/s</span>.</p>`,
    details: [["Engine", h.torrent.engine], ["Trackers", h.torrent.trackers.join(", ")]],
  };
  if (res === "tunnel") return {
    eyebrow: "Network",
    title: "Tunnel",
    body: S.health.tunnel === "down" ? `<p>Down. The Relay Node cannot reach home, so remote viewers cannot play anything.</p>` : `<p>Carrying <span class="num">${S.tunnel.mbps.toFixed(1)} Mbit/s</span> of video to the Relay Node, ${pct(S.tunnel.share)} of the home upload.</p><p>Each trip takes <span class="num">${S.tunnel.latencyMs} ms</span>${S.health.tunnel === "slow" ? ", far slower than usual" : ""}.</p>`,
    details: [["Kind", TUNNEL.name], ["Port", `UDP ${TUNNEL.port}`], ["Upload", `${TUNNEL.upstreamMbps} Mbit/s`], ["Home", h.addr], ["Relay", r.addr]],
  };
  return {
    eyebrow: "Node · Relay",
    title: "Relay Node",
    body: `<p>${S.health.relay === "warn" ? "Running, with a warning. Plex has not confirmed remote access since 19:20." : down ? "Running, waiting for the media server to answer." : "Running in Ashburn."} ${S.remote.length ? `Sending video to ${plural(S.remote.length, "remote viewer", "remote viewers")} over TLS:` : "No remote viewers right now."}</p>${S.remote.length ? `<ul class="rows">${S.remote.map((s) => `<li><span class="dot"></span><span><strong>${s.who}</strong> on a ${s.device}, ${s.title}, <span class="num">${s.mbps.toFixed(1)} Mbit/s</span></span></li>`).join("")}</ul>` : ""}<p>The processor is <span class="num">${S.relay.cpu}%</span> busy.</p>`,
    details: [["Host", r.host], ["Where", r.where], ["System", r.os], ["Hardware", r.hw], ["Address", r.addr], ["Fronts", r.fronts.join(", ")]],
  };
}

let selected = null;
let panelKey = "";
function renderPanel(force) {
  const info = selected ? describe(selected) : overall();
  const html = `<div class="eyebrow">${info.eyebrow}</div><h2>${info.title}</h2>${info.body}<details${panel.querySelector("details")?.open ? " open" : ""}><summary>Details</summary><dl>${info.details.map(([a, b]) => `<dt>${a}</dt><dd>${b}</dd>`).join("")}</dl></details>${info.hint ? `<p class="hint">${info.hint}</p>` : ""}`;
  if (force || html !== panelKey) {
    panelKey = html;
    panel.innerHTML = html;
  }
  const text = S.health.home === "down" ? "↓ 0 MB/s · tunnel down" : `↓ ${S.downMBps.toFixed(1)} MB/s · ↑ ${S.tunnel.mbps.toFixed(1)} Mbit/s to ${S.remote.length} ${S.remote.length === 1 ? "viewer" : "viewers"}`;
  if (readout.textContent !== text) readout.textContent = text;
}

function select(res) {
  if (res === selected) return;
  selected = res;
  for (const n of el.items) n.toggleAttribute("data-on", n.dataset.res === res);
  stage.toggleAttribute("data-focus", !!res);
  renderPanel(true);
}

function setState(next) {
  if (!STATES[next] || next === stateId) return;
  stateId = next;
  S = sample(stateId, clock.t);
  sentenceText.textContent = S.sentence;
  sentenceEl.dataset.health = S.health.home === "down" ? "down" : S.health.relay === "warn" ? "warn" : "ok";
  stateButtons.forEach((b) => b.setAttribute("aria-pressed", String(b.dataset.state === next)));
  renderPanel(true);
  kick();
}

let frame = 0, last = 0, visible = false, panelAt = 0;
let touring = !still.matches, tourClock = 0, idleTimer = 0;
const TOUR = ["storage", "transcoder", "torrent", "tunnel", "relay", "home"];
function loop(now) {
  const dt = last ? Math.min((now - last) / 1000, 1 / 30) : 1 / 60;
  last = now;
  const calm = still.matches;
  tick(dt, calm);
  if (touring && !calm) {
    tourClock += dt;
    if (tourClock > 4) {
      const i = Math.floor((tourClock - 4) / 4) % TOUR.length;
      select(TOUR[i]);
    }
  }
  if (now - panelAt > 250) { panelAt = now; renderPanel(false); }
  frame = visible && !calm ? requestAnimationFrame(loop) : 0;
}
function kick() {
  if (still.matches) { tick(0, true); renderPanel(false); return; }
  if (!frame && visible) { last = 0; frame = requestAnimationFrame(loop); }
}
function takeOver() {
  touring = false;
  clearTimeout(idleTimer);
  idleTimer = setTimeout(() => { if (still.matches) return; touring = true; tourClock = 0; select(null); }, 4000);
}

new IntersectionObserver((entries) => { visible = entries[entries.length - 1].isIntersecting; if (visible) kick(); }, { rootMargin: "120px 0px" }).observe(stage);
stage.addEventListener("pointermove", (e) => {
  takeOver();
  const it = e.target.closest?.(".it[data-res]");
  select(it ? it.dataset.res : null);
});
stage.addEventListener("pointerdown", (e) => {
  takeOver();
  const it = e.target.closest?.(".it[data-res]");
  select(it ? it.dataset.res : null);
});
stage.addEventListener("pointerleave", () => { takeOver(); select(null); });
stage.addEventListener("focus", () => takeOver());
stage.addEventListener("keydown", (e) => {
  const order = D.res;
  let i = order.indexOf(selected);
  if (e.key === "ArrowRight" || e.key === "ArrowDown") i = (i + 1) % order.length;
  else if (e.key === "ArrowLeft" || e.key === "ArrowUp") i = i < 0 ? order.length - 1 : (i - 1 + order.length) % order.length;
  else if (e.key === "Escape") i = -1;
  else return;
  e.preventDefault();
  takeOver();
  select(i < 0 ? null : order[i]);
});
stateButtons.forEach((b) => b.addEventListener("click", () => setState(b.dataset.state)));
window.addEventListener("message", (e) => e.data?.state && setState(e.data.state));
window.addEventListener("hashchange", () => { const s = validState(location.hash.slice(1).replace(/^state=/, "")); if (s) setState(s); });
still.addEventListener?.("change", () => { touring = !still.matches; kick(); });

sentenceText.textContent = S.sentence;
sentenceEl.dataset.health = S.health.home === "down" ? "down" : S.health.relay === "warn" ? "warn" : "ok";
stateButtons.forEach((b) => b.setAttribute("aria-pressed", String(b.dataset.state === stateId)));
ease.power = S.health.home === "down" ? 0 : 1;
ease.tunnel = S.health.tunnel === "down" ? 0 : 1;
tick(0, true);
if (!still.matches) {
  for (let i = 0; i < Math.round(videoRate() * travel()); i++) spawnCap("video", Math.random() * travel());
  for (let i = 0; i < Math.round(reqRate() * travel()); i++) spawnCap("req", Math.random() * travel());
}
renderPanel(true);
