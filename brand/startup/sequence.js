export const DURATION = 4.4;

export const PHASES = {
  gridIn: [0, 0.7],
  popA: 0.14,
  popB: 0.3,
  launch: 0.66,
  depart: 0.7,
  arrive: 1.56,
  tau: 0.17,
  bloom: [1.56, 2.66],
  ack: [1.66, 2.04],
  lockup: [2.02, 2.92],
  wipe: [2.24, 3.08],
  settle: [2.6, 3.6],
  exit: [3.96, 4.36],
};

export const STILL = PHASES.settle[1];

const MARKUP = `
<div class="startup-ambient"></div>
<div class="startup-grid"></div>
<svg viewBox="0 0 1920 1080" preserveAspectRatio="xMidYMid meet" aria-hidden="true">
  <defs>
    <filter id="startup-soft" x="-50%" y="-50%" width="200%" height="200%"><feGaussianBlur stdDeviation="2.2"/></filter>
    <filter id="startup-softer" x="-100%" y="-100%" width="300%" height="300%"><feGaussianBlur stdDeviation="4.5"/></filter>
    <linearGradient id="startup-wipe" gradientUnits="userSpaceOnUse" x1="0" x2="1920" y1="0" y2="0">
      <stop id="startup-w0" offset="0" stop-color="#fff"/>
      <stop id="startup-w1" offset="0" stop-color="#000"/>
    </linearGradient>
    <mask id="startup-wordMask" maskUnits="userSpaceOnUse" x="0" y="0" width="1920" height="1080">
      <rect width="1920" height="1080" fill="url(#startup-wipe)"/>
    </mask>
  </defs>
  <g id="startup-lockup">
    <g id="startup-mark">
      <path id="startup-E" d="M52 14H25a9 9 0 0 0 0 18h14a9 9 0 0 1 0 18H12" fill="none" stroke="none"/>
      <path id="startup-S" d="M40 14H25a9 9 0 0 0 0 18h14a9 9 0 0 1 0 18H24" fill="none" stroke-width="8" stroke-linecap="round" stroke-linejoin="round"/>
      <g id="startup-draft" fill="none" stroke-linecap="round"></g>
      <g id="startup-trailGlow" filter="url(#startup-softer)" style="mix-blend-mode: screen"></g>
      <g id="startup-trailCore" style="mix-blend-mode: screen"></g>
      <circle id="startup-ringA" cx="52" cy="14" r="5.5" fill="none"/>
      <circle id="startup-ringB" cx="12" cy="50" r="5.5" fill="none"/>
      <circle id="startup-ringB2" cx="12" cy="50" r="5.5" fill="none"/>
      <circle id="startup-ringA2" cx="52" cy="14" r="5.5" fill="none"/>
      <circle id="startup-haloA" cx="52" cy="14" r="5.5" filter="url(#startup-softer)"/>
      <circle id="startup-haloB" cx="12" cy="50" r="5.5" filter="url(#startup-softer)"/>
      <circle id="startup-dotA" cx="52" cy="14" r="5.5"/>
      <circle id="startup-dotB" cx="12" cy="50" r="5.5"/>
      <path id="startup-ackGlow" d="M12 50H39a9 9 0 0 0 0-18H25a9 9 0 0 1 0-18H52" fill="none" stroke-width="12" stroke-linecap="round" filter="url(#startup-softer)" style="mix-blend-mode: screen"/>
      <path id="startup-ackCore" d="M12 50H39a9 9 0 0 0 0-18H25a9 9 0 0 1 0-18H52" fill="none" stroke-width="5" stroke-linecap="round" filter="url(#startup-soft)" style="mix-blend-mode: screen"/>
      <circle id="startup-headGlow" r="9" filter="url(#startup-softer)"/>
      <circle id="startup-head" r="4" fill="#fff"/>
    </g>
    <g id="startup-word" mask="url(#startup-wordMask)" style="letter-spacing: -0.03em; font-kerning: normal">
      <text id="startup-wordText" x="0" y="0"><tspan font-weight="600">Slip</tspan><tspan font-weight="400">Stream</tspan></text>
    </g>
  </g>
</svg>
<div class="startup-grain"></div>`;

const NS = "http://www.w3.org/2000/svg";
const TRAIL_N = 16;
const STEP = 0.25;
const DRAFT = [8, -8, 12, -12];
const DRAFT_SEG = 14;
const ARC = 9 * Math.PI;
const ARCS = [
  [27, 27 + ARC],
  [41 + ARC, 41 + 2 * ARC],
];
const CENTER = { x: 960, y: 540 };
const BIG = 300 / 64;
const SMALL = 152 / 64;
const S_START = 12;

const glow = "hsl(214 95% 72%)";
const ink = "hsl(214 25% 96%)";
const hot = "hsl(214 100% 88%)";

function bezier(x1, y1, x2, y2) {
  const cx = 3 * x1,
    bx = 3 * (x2 - x1) - cx,
    ax = 1 - cx - bx;
  const cy = 3 * y1,
    by = 3 * (y2 - y1) - cy,
    ay = 1 - cy - by;
  const sx = (t) => ((ax * t + bx) * t + cx) * t;
  const sy = (t) => ((ay * t + by) * t + cy) * t;
  const dx = (t) => (3 * ax * t + 2 * bx) * t + cx;
  return (x) => {
    if (x <= 0) return 0;
    if (x >= 1) return 1;
    let t = x;
    for (let i = 0; i < 8; i++) {
      const e = sx(t) - x;
      if (Math.abs(e) < 1e-6) break;
      const d = dx(t);
      if (Math.abs(d) < 1e-6) break;
      t -= e / d;
    }
    let lo = 0,
      hi = 1;
    for (let i = 0; i < 30 && Math.abs(sx(t) - x) > 1e-6; i++) {
      if (sx(t) < x) lo = t;
      else hi = t;
      t = (lo + hi) / 2;
    }
    return sy(t);
  };
}
const standard = bezier(0.16, 1, 0.3, 1);
const spring = bezier(0.32, 0.72, 0, 1);
const travel = bezier(0.62, 0, 0.18, 1);
const ackEase = bezier(0.4, 0, 0.2, 1);
const clamp01 = (x) => Math.min(1, Math.max(0, x));
const span = (t, a, b) => clamp01((t - a) / (b - a));
const lerp = (a, b, k) => a + (b - a) * k;

function bounce(t, { stiffness = 260, damping = 16 } = {}) {
  if (t <= 0) return 0;
  const w0 = Math.sqrt(stiffness),
    z = damping / (2 * w0);
  if (z >= 1) return 1 - Math.exp(-w0 * t) * (1 + w0 * t);
  const wd = w0 * Math.sqrt(1 - z * z);
  return 1 - Math.exp(-z * w0 * t) * (Math.cos(wd * t) + ((z * w0) / wd) * Math.sin(wd * t));
}

function straightness(l) {
  let k = 1;
  for (const [a, b] of ARCS) {
    if (l > a - 3 && l < b + 3)
      k = Math.min(
        k,
        l < a + 2 ? clamp01((a + 2 - l) / 5) : l > b - 2 ? clamp01((l - b + 2) / 5) : 0,
      );
  }
  return k * k * (3 - 2 * k);
}

const stages = new WeakMap();

// build fills stage with the sequence's layers and measures the wordmark in
// the stage's inherited font, which it waits for.
export async function build(stage) {
  stage.innerHTML = MARKUP;
  const $ = (id) => stage.querySelector(`#startup-${id}`);
  const doc = stage.ownerDocument;
  const E = $("E");
  const LE = E.getTotalLength();
  const LS = $("S").getTotalLength();

  const trailGlow = [];
  const trailCore = [];
  for (let i = 0; i < TRAIL_N; i++) {
    for (const [list, parent] of [
      [trailGlow, $("trailGlow")],
      [trailCore, $("trailCore")],
    ]) {
      const p = doc.createElementNS(NS, "path");
      p.setAttribute("d", E.getAttribute("d"));
      p.setAttribute("fill", "none");
      p.setAttribute("stroke-linecap", "round");
      p.setAttribute("stroke-linejoin", "round");
      parent.appendChild(p);
      list.push(p);
    }
  }

  const samples = [];
  for (let l = 0; l <= LE + STEP; l += STEP) {
    const a = E.getPointAtLength(Math.max(0, l - 0.5)),
      b = E.getPointAtLength(Math.min(LE, l + 0.5)),
      p = E.getPointAtLength(Math.min(LE, l));
    const dx = b.x - a.x,
      dy = b.y - a.y,
      n = Math.hypot(dx, dy) || 1;
    samples.push({ x: p.x, y: p.y, nx: -dy / n, ny: dx / n });
  }
  const draftLines = DRAFT.map(() =>
    Array.from({ length: DRAFT_SEG }, () => {
      const el = doc.createElementNS(NS, "polyline");
      $("draft").appendChild(el);
      return el;
    }),
  );

  const text = $("wordText");
  const family = doc.defaultView.getComputedStyle(text).fontFamily;
  await Promise.all([
    doc.fonts.load(`600 91px ${family}`),
    doc.fonts.load(`400 91px ${family}`),
  ]);
  const fontSize = 91;
  const c = doc.createElement("canvas").getContext("2d");
  c.font = `600 ${fontSize}px ${family}`;
  const cap = c.measureText("H").actualBoundingBoxAscent;
  const scale = 64 / 2 / cap;
  text.setAttribute("font-size", fontSize * (SMALL * scale));
  const textW = text.getComputedTextLength();
  const box = 64 * SMALL,
    gap = box / 3;
  const left = CENTER.x - (box + gap + textW) / 2;
  const layout = {
    markEnd: { x: left + box / 2, y: CENTER.y },
    textX: left + box + gap,
    baseline: CENTER.y + (cap * SMALL * scale) / 2,
    textW,
  };
  text.setAttribute("x", layout.textX);
  text.setAttribute("y", layout.baseline);

  stages.set(stage, {
    $,
    ambient: stage.querySelector(".startup-ambient"),
    grid: stage.querySelector(".startup-grid"),
    E,
    S: $("S"),
    LE,
    LS,
    trailGlow,
    trailCore,
    samples,
    draftLines,
    layout,
  });
}

// render draws frame t (seconds) of the sequence into a stage build filled.
export function render(t, stage) {
  const { $, ambient, grid, E, S, LE, LS, trailGlow, trailCore, samples, draftLines, layout } =
    stages.get(stage);
  const T = PHASES;
  const cometAt = (time) => LE * travel(span(time, T.depart, T.arrive));
  const offsetPoints = (a, b, off) => {
    const pts = [];
    for (let i = Math.floor(a / STEP); i <= Math.ceil(b / STEP) && i < samples.length; i++) {
      const q = samples[i];
      pts.push(`${(q.x + q.nx * off).toFixed(3)},${(q.y + q.ny * off).toFixed(3)}`);
    }
    return pts.join(" ");
  };
  const ring = (el, start, dur, grow, width) => {
    const k = span(t, start, start + dur);
    const e = standard(k);
    el.setAttribute("r", 5.5 + grow * e);
    el.setAttribute("stroke", glow);
    el.setAttribute("stroke-width", width * (1 - e * 0.7));
    el.setAttribute("opacity", k <= 0 || k >= 1 ? 0 : (1 - e) * 0.75);
  };
  const dot = (el, halo, popAt, cx, cy, extraScale, haloBoost) => {
    const k = bounce(t - popAt, { stiffness: 320, damping: 15 });
    const s = Math.max(0, k) * extraScale;
    el.setAttribute("transform", `translate(${cx} ${cy}) scale(${s}) translate(${-cx} ${-cy})`);
    el.setAttribute("fill", glow);
    halo.setAttribute(
      "transform",
      `translate(${cx} ${cy}) scale(${s * (1.25 + haloBoost * 0.9)}) translate(${-cx} ${-cy})`,
    );
    halo.setAttribute("fill", glow);
    halo.setAttribute("opacity", (0.35 + haloBoost * 0.65) * clamp01(k));
  };

  grid.style.opacity = standard(span(t, ...T.gridIn)) * (1 - 0.65 * standard(span(t, ...T.lockup)));
  grid.style.transform = `translate3d(${-t * 6}px, ${t * 3}px, 0)`;
  ambient.style.opacity = standard(span(t, ...T.bloom));

  const m = spring(span(t, ...T.lockup));
  const lift = 1 - standard(span(t, 0, 1.4));
  const markX = lerp(CENTER.x, layout.markEnd.x, m);
  const markY = lerp(CENTER.y, layout.markEnd.y, m) + lift * 10;
  const markK = lerp(BIG, SMALL, m) * (1 - lift * 0.03);
  $("mark").setAttribute("transform", `translate(${markX} ${markY}) scale(${markK}) translate(-32 -32)`);

  const d = cometAt(t),
    tail = cometAt(t - T.tau);
  const moving = t >= T.depart && t < T.arrive + T.tau + 0.05;

  const drawn = clamp01((d - S_START) / LS) * LS;
  S.setAttribute("stroke", ink);
  S.setAttribute("stroke-dasharray", `${drawn} ${LS + 20}`);
  S.setAttribute("opacity", drawn > 0.01 ? 1 : 0);

  for (let i = 0; i < TRAIL_N; i++) {
    const f = i / (TRAIL_N - 1);
    const from = lerp(d, tail, f);
    const len = Math.max(0, d - from);
    for (const [p, w, o, c] of [
      [trailGlow[i], 10 * (1 - f) + 2, 0.5, glow],
      [trailCore[i], 6.5 * (1 - f * 0.9), 0.22, hot],
    ]) {
      p.setAttribute("stroke", c);
      p.setAttribute("stroke-width", w);
      p.setAttribute("stroke-dasharray", `${len} ${LE * 2}`);
      p.setAttribute("stroke-dashoffset", -from);
      p.setAttribute("opacity", moving && len > 0.05 ? o : 0);
    }
  }

  const dTail = cometAt(t - T.tau * 1.25),
    dLead = Math.max(dTail, d - 4);
  DRAFT.forEach((off, j) => {
    const far = Math.abs(off) > 10;
    for (let i = 0; i < DRAFT_SEG; i++) {
      const el = draftLines[j][i];
      const f0 = i / DRAFT_SEG,
        f1 = (i + 1) / DRAFT_SEG;
      const a = lerp(dLead, dTail, f1),
        b = lerp(dLead, dTail, f0);
      const vis = moving && b - a > 0.05;
      el.setAttribute("points", vis ? offsetPoints(a, b, off) : "");
      el.setAttribute("stroke", glow);
      el.setAttribute("stroke-width", far ? 0.45 : 0.7);
      el.setAttribute(
        "opacity",
        vis ? (1 - f0) * (far ? 0.4 : 0.65) * straightness((a + b) / 2) : 0,
      );
    }
  });

  const head = E.getPointAtLength(d);
  const headVisible = t >= T.depart && t < T.arrive;
  const headIn = standard(span(t, T.depart, T.depart + 0.12));
  for (const id of ["head", "headGlow"]) {
    $(id).setAttribute("cx", head.x);
    $(id).setAttribute("cy", head.y);
  }
  $("head").setAttribute("opacity", headVisible ? headIn : 0);
  $("headGlow").setAttribute("fill", glow);
  $("headGlow").setAttribute("opacity", headVisible ? headIn : 0);

  const recoil =
    t < T.launch ? 1 : 1 - 0.2 * Math.sin(Math.PI * span(t, T.launch, T.launch + 0.22));
  const flare = t < T.arrive ? 0 : Math.exp(-(t - T.arrive) * 6);
  const reply = t < T.ack[1] ? 0 : Math.exp(-(t - T.ack[1]) * 7);
  const charge = Math.sin(Math.PI * span(t, 0.42, T.launch + 0.06)) ** 2;
  const settle = 1 - standard(span(t, ...T.settle));
  dot(
    $("dotA"),
    $("haloA"),
    T.popA,
    52,
    14,
    recoil * (1 + 0.22 * reply),
    (t < 0.9 ? 0.6 : 0) * settle + reply * 0.8 + charge * 0.9 + 0.25 * settle,
  );
  dot(
    $("dotB"),
    $("haloB"),
    T.popB,
    12,
    50,
    1 + 0.38 * flare,
    (t < 0.9 ? 0.6 : 0) * settle + flare + 0.25 * settle,
  );
  ring($("ringA"), T.popA, 0.8, 9, 1.4);
  ring($("ringB"), T.popB, 0.8, 9, 1.4);
  ring($("ringB2"), T.arrive, 0.95, 18, 2);
  ring($("ringA2"), T.ack[1], 0.85, 12, 1.4);

  const ak = span(t, ...T.ack);
  const ad = LE * ackEase(ak);
  const seg = 16;
  for (const [id, o] of [
    ["ackGlow", 0.9],
    ["ackCore", 0.55],
  ]) {
    const el = $(id);
    el.setAttribute("stroke", glow);
    el.setAttribute("stroke-dasharray", `${seg} ${LE * 2}`);
    el.setAttribute("stroke-dashoffset", -(ad - seg));
    el.setAttribute("opacity", ak > 0 && ak < 1 ? o * Math.sin(Math.PI * ak) ** 0.4 : 0);
  }

  const w = standard(span(t, ...T.wipe));
  const edge = lerp(layout.textX - 60, layout.textX + layout.textW + 80, w);
  $("w0").setAttribute("offset", (edge - 70) / 1920);
  $("w1").setAttribute("offset", edge / 1920);
  $("word").setAttribute("transform", `translate(${markX - layout.markEnd.x + lerp(-24, 0, w)} 0)`);
  $("wordText").setAttribute("fill", ink);

  const x = standard(span(t, ...T.exit));
  $("lockup").setAttribute("opacity", 1 - x);
  $("lockup").setAttribute(
    "transform",
    `translate(960 540) scale(${1 + x * 0.025}) translate(-960 -540)`,
  );
}

// liftMark hides the drawn mark and returns a copy of it as a fixed element
// over the same box, for a host to hand to another element.
export function liftMark(stage) {
  const { $ } = stages.get(stage);
  const doc = stage.ownerDocument;
  const mark = $("mark");
  const ctm = mark.getScreenCTM();
  const scale = Math.hypot(ctm.a, ctm.b);
  const holder = doc.createElement("div");
  Object.assign(holder.style, {
    position: "fixed",
    left: `${ctm.e}px`,
    top: `${ctm.f}px`,
    width: `${64 * scale}px`,
    height: `${64 * scale}px`,
  });
  const copy = doc.createElementNS(NS, "svg");
  copy.setAttribute("viewBox", "0 0 64 64");
  copy.setAttribute("width", "100%");
  copy.setAttribute("height", "100%");
  copy.setAttribute("overflow", "visible");
  for (const id of ["haloA", "haloB", "S", "dotA", "dotB"]) {
    const part = $(id).cloneNode();
    part.removeAttribute("id");
    copy.append(part);
  }
  holder.append(copy);
  stage.append(holder);
  mark.setAttribute("opacity", 0);
  return holder;
}
