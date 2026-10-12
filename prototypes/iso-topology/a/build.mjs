import { readFileSync, writeFileSync } from "node:fs";
import { dirname, join } from "node:path";
import { fileURLToPath } from "node:url";

const HERE = dirname(fileURLToPath(import.meta.url));
const SRC = join(HERE, "..", "..", "anatomy", "src");
const k = await import(join(SRC, "kit", "iso-kit.mjs"));
const G = await import(join(SRC, "kit", "lathe.mjs"));
const { tubePieces, tubeSvg } = await import(join(SRC, "kit", "tube.mjs"));
const { kitScript } = await import(join(SRC, "scripts", "inline-kit.mjs"));
const MODEL_SRC = readFileSync(join(HERE, "..", "model.mjs"), "utf8");
const { NODES, sample } = await import(join(HERE, "..", "model.mjs"));
const THEME = readFileSync(join(HERE, "..", "theme.css"), "utf8");

const W = 1100;
const H = 560;
const AZ = 58;
const EL = 30;

const PL = { x: -470, y: -78, w: 300, d: 190, r: 8 };
const PL_TOP = 15;
const T = { x0: -452, x1: -298, y0: -64, y1: 4, z0: 19, z1: 177 };
const WALL = 3;
const BOARD = { x0: -382, x1: -306, z0: 66, z1: 168, y: -58 };
const CPU = { x: -347, z: 135 };
const FAN_Y = -34;
const CAGE = { x0: -448, x1: -388, z0: 24, z1: 116 };
const SLED_H = 19.5;
const SLED_Z = [28, 50.5, 73, 95.5];
const SLED_Y = 0.5;
const PSU = { x0: -386, x1: -302, z0: 22, z1: 60 };
const PB = { x0: -444, x1: -306, y0: -58, y1: -4, z: 186 };
const COLS = 16;
const ROWS = 6;
const CELL = 7.6;
const ROUTER = { x: -262, y: -26, w: 44, d: 30 };
const LRTV = { x: -252, x1: -192, y: 92 };
const RDX = -300;
const RDY = -430;
const sh = ([x, y, z]) => [x + RDX, y + RDY, z];
const TRAY = { x0: -166, xa: -104, xb: -80, y0: -30, y1: -6, yEnd: -358, z0: 4, z1: 11 };
const TILE = { x: 140 + RDX, y: -78 + RDY, w: 150, d: 130, r: 4 };
const TILE_TOP = 10;
const RACK = { x0: 150 + RDX, x1: 282 + RDX, post: 10, y0: -24 + RDY, y1: -14 + RDY, z1: 104 };
const U = 14;
const PEER = { x: -720, y: -250, w: 110, d: 120, r: 6 };
const MAST = [-686, -212];
const MAST_TOP = 168;

const P = k.fitProjection(
  [
    ...k.boxCorners(PEER, 0, MAST_TOP + 8),
    ...k.boxCorners(PL, 0, PB.z + 12),
    ...k.boxCorners(TILE, 0, RACK.z1),
    sh([400, -60, 0]), sh([300, 80, 0]), sh([390, -40, 80]),
  ],
  W,
  H,
  { pad: 26, azimuth: AZ, elevation: EL },
);
const at = (p) => k.iso(p, P);
const S = (paths, style) => k.solidSvg(paths, style);
const L = (d, style) => k.lineSvg(d, style);
const D = (points, style) => k.dotsSvg(points, style);
const path3 = (points, closed = false) => k.pathOf(points.map(at), closed);
const f2 = (v) => Math.round(v * 100) / 100;
const box = (x0, y0, z0, x1, y1, z1, { r = 0.5, steps = 2, bevel = 0 } = {}) => k.slabOf({ x: x0, y: y0, w: x1 - x0, d: y1 - y0, r }, z0, z1 - z0, P, steps, bevel);
const byDepth = (points) => [...points].sort((a, b) => k.depthOf([a[0], a[1], a[2] ?? 0], P) - k.depthOf([b[0], b[1], b[2] ?? 0], P));
const depth = (p) => k.depthOf(p, P);

function plane(o, ex, ey) {
  const p0 = at(o);
  const px = at(G.add3(o, ex));
  const py = at(G.add3(o, ey));
  const v = (x) => x.toFixed(4);
  return `matrix(${v(px[0] - p0[0])} ${v(px[1] - p0[1])} ${v(py[0] - p0[0])} ${v(py[1] - p0[1])} ${p0[0].toFixed(2)} ${p0[1].toFixed(2)})`;
}
const frontPlane = (y) => plane([0, y, 0], [1, 0, 0], [0, 0, -1]);
const sidePlane = (x) => plane([x, 0, 0], [0, -1, 0], [0, 0, -1]);
const topPlane = (z) => plane([0, 0, z], [1, 0, 0], [0, 1, 0]);
const rr = (x0, y0, x1, y1, r = 0.6, steps = 4) => k.pathOf(k.roundedPlan({ x: Math.min(x0, x1), y: Math.min(y0, y1), w: Math.abs(x1 - x0), d: Math.abs(y1 - y0), r }, steps), true);
const circle = (cx, cy, r) => `M${f2(cx + r)} ${f2(cy)}A${r} ${r} 0 1 1 ${f2(cx - r)} ${f2(cy)}A${r} ${r} 0 1 1 ${f2(cx + r)} ${f2(cy)}Z`;
const flatLine = (d, tone = "lo", extra = "") => `<path class="iso-line" data-tone="${tone}"${extra} d="${d}"/>`;

function cable(points, r, { tone = "mid", bend = 4, max = 60 } = {}) {
  const route = G.fillet(points, Math.max(bend, r * 2.6), 8, { P, tube: r });
  return tubePieces(route, r, P, { maxLength: max }).map((piece) => tubeSvg(piece, { tone })).join("");
}

function feetUnder(plan, z0, h, r = 6) {
  const out = [];
  for (const [x, y] of byDepth(k.corners(plan, 18))) {
    out.push(S(k.cylinder(x, y, r + 1.4, z0, 1.4, P, 24), { tone: "lo" }));
    out.push(S(k.cylinder(x, y, r, z0 + 1.4, h - 1.4, P, 24), { tone: "mid" }));
  }
  return out.join("");
}

function labelPlate(plan, z, lines = [0.7, 0.45]) {
  const out = [S(k.slabOf(plan, z, 0.8, P, 3), { tone: "lo", crease: "none" })];
  out.push(L(k.planOutline(k.insetPlan(plan, 1.2), z + 0.8, P, 3), { tone: "faint" }));
  out.push(L(lines.map((share, i) => k.lineOnTop([plan.x + 2.6, plan.y + 2.8 + i * 2.6], [plan.x + 2.6 + (plan.w - 7) * share, plan.y + 2.8 + i * 2.6], z + 0.8, P)).join(""), { tone: "lo", free: true }));
  out.push(D([at([plan.x + plan.w - 2.4, plan.y + plan.d / 2, z + 0.8])], { size: 0.45 }));
  return out.join("");
}

const fanBlades = (r0, r1, count) =>
  Array.from({ length: count }, (_, i) => {
    const a = (i / count) * Math.PI * 2;
    const pt = (r, da) => [r * Math.cos(a + da), r * Math.sin(a + da)].map(f2).join(" ");
    return `M${pt(r0, 0)}Q${pt(r1 * 0.72, 0.28)} ${pt(r1, 0.14)}L${pt(r1, 0.7)}Q${pt(r1 * 0.68, 0.8)} ${pt(r0, 0.78)}Z`;
  }).join("");

function fanFace(cx, cy, r, part, { spokes = 4, rings = [0.96, 0.66, 0.36], blades = 7 } = {}) {
  const guard = rings.map((s) => circle(cx, cy, f2(r * s))).join("");
  const spoke = Array.from({ length: spokes }, (_, i) => {
    const a = (i / spokes) * Math.PI * 2 + 0.4;
    return `M${f2(cx + r * 0.36 * Math.cos(a))} ${f2(cy + r * 0.36 * Math.sin(a))}L${f2(cx + r * 0.96 * Math.cos(a))} ${f2(cy + r * 0.96 * Math.sin(a))}`;
  }).join("");
  return [
    `<path class="t-hole" d="${circle(cx, cy, r)}"/>`,
    `<circle class="t-blur" data-part="${part}-blur" cx="${cx}" cy="${cy}" r="${f2(r * 0.94)}" opacity="0"/>`,
    `<g data-part="${part}" data-cx="${cx}" data-cy="${cy}" transform="translate(${cx} ${cy})"><path class="t-blade" d="${fanBlades(r * 0.3, r * 0.92, blades)}"/></g>`,
    `<circle class="t-hub" cx="${cx}" cy="${cy}" r="${f2(r * 0.3)}"/>`,
    `<circle class="t-hub-dot" cx="${cx}" cy="${cy}" r="${f2(r * 0.09)}"/>`,
    `<path class="t-guard" d="${guard}${spoke}"/>`,
  ].join("");
}

function slots(x0, x1, y0, y1, count, vertical = true, r = 0.7) {
  const out = [];
  if (vertical) {
    const step = (x1 - x0) / count;
    for (let i = 0; i < count; i++) out.push(rr(x0 + i * step + step * 0.3, y0, x0 + i * step + step * 0.7, y1, Math.min(r, step * 0.2)));
  } else {
    const step = (y1 - y0) / count;
    for (let i = 0; i < count; i++) out.push(rr(x0, y0 + i * step + step * 0.3, x1, y0 + i * step + step * 0.7, Math.min(r, step * 0.2)));
  }
  return out.join("");
}

const groups = [];
const put = (res, svg, extra = "") => groups.push(`<g class="it" data-res="${res}"${extra}>${svg}</g>`);

function peers() {
  const out = [`<path class="iso-halo" d="${k.haloOf(PEER, 0, 6, P)}"/>`];
  out.push(feetUnder(PEER, 0, 4, 4));
  out.push(S(k.slabOf(PEER, 4, 6, P, 6, 1), { tone: "mid" }));
  out.push(L(k.planOutline(k.insetPlan(PEER, 4), 10, P), { tone: "lo" }));
  const screws = k.corners(PEER, 9);
  out.push(L(screws.map(([x, y]) => k.ring(x, y, 1.8, 10, P, 14)).join(""), { tone: "lo" }));
  out.push(D(screws.map(([x, y]) => at([x, y, 10])), { size: 0.45 }));
  const [mx, my] = MAST;
  const base = { x: mx - 10, y: my - 10, w: 20, d: 20, r: 2 };
  out.push(S(k.slabOf(base, 10, 4, P, 3, 0.5), { tone: "mid" }));
  out.push(D(k.corners(base, 3).map(([x, y]) => at([x, y, 14])), { size: 0.45 }));
  const legs = [[-5, -5], [5, -5], [5, 5], [-5, 5]];
  const topW = 1.4;
  const legAt = ([lx, ly], z) => {
    const share = (z - 14) / (MAST_TOP - 14);
    const s = 1 - share * (1 - topW / 5);
    return [mx + lx * s, my + ly * s, z];
  };
  const lattice = [];
  const rungs = [];
  for (let z = 14; z < MAST_TOP - 6; z += 14) rungs.push(z);
  const visible = [[0, 1], [1, 2], [2, 3]];
  const backLegs = [legs[0]];
  lattice.push(backLegs.map((l) => path3([legAt(l, 14), legAt(l, MAST_TOP - 4)])).join(""));
  out.push(L(lattice.join(""), { tone: "lo" }));
  const front = [];
  for (const [a, b] of visible) {
    for (let i = 0; i < rungs.length - 1; i++) {
      front.push(path3([legAt(legs[a], rungs[i]), legAt(legs[b], rungs[i + 1])]));
      front.push(path3([legAt(legs[a], rungs[i + 1]), legAt(legs[b], rungs[i + 1])]));
    }
    front.push(path3([legAt(legs[a], rungs[0]), legAt(legs[b], rungs[0])]));
  }
  out.push(L(front.join(""), { tone: "lo" }));
  out.push(L([1, 2, 3].map((i) => path3([legAt(legs[i], 14), legAt(legs[i], MAST_TOP - 4)])).join(""), { tone: "mid" }));
  out.push(S(k.cylinder(mx, my, 3, MAST_TOP - 4, 4, P, 20), { tone: "mid" }));
  out.push(S(k.cylinder(mx, my, 1, MAST_TOP, 10, P, 12), { tone: "mid", crease: "none" }));
  out.push(S(k.cylinder(mx, my, 2.2, MAST_TOP + 10, 2, P, 16), { tone: "hi" }));
  const dish = G.frameAlong([mx + 3, my, MAST_TOP - 22], [1, 0.3, 0.15]);
  out.push(S(k.slabOf({ x: mx + 1.3, y: my - 1, w: 3, d: 2, r: 0.5 }, MAST_TOP - 24, 4, P, 2), { tone: "mid", crease: "none" }));
  out.push(S(G.solidOf([[0, 9], [1.2, 9], [2.6, 5, 1], [3.6, 1.8, 1], [4.4, 0.6]], dish, P), { tone: "mid" }));
  out.push(L(G.arcOf(dish, 0.6, 8.3, P), { tone: "faint" }));
  const nodes = [
    { x: -700, y: -168, s: 1 },
    { x: -668, y: -156, s: 2 },
    { x: -646, y: -200, s: 3 },
  ];
  const peerCables = [
    [[nodes[0].x + 11, nodes[0].y - 0.3, 14], [nodes[0].x + 11, nodes[0].y - 6, 10.7], [MAST[0] + 4, MAST[1] + 14, 10.7], [MAST[0] + 4, MAST[1] + 10.3, 11.4]],
    [[nodes[1].x + 11, nodes[1].y - 0.3, 14], [nodes[1].x + 11, nodes[1].y - 8, 10.7], [MAST[0] + 8, MAST[1] + 16, 10.7], [MAST[0] + 8, MAST[1] + 10.3, 11.4]],
    [[nodes[2].x - 0.3, nodes[2].y + 7, 14], [nodes[2].x - 6, nodes[2].y + 7, 10.7], [MAST[0] + 16, MAST[1] + 5, 10.7], [MAST[0] + 10.3, MAST[1] + 5, 11.4]],
  ];
  out.push(peerCables.map((r) => cable(r, 0.6, { tone: "mid", bend: 3 })).join(""));
  nodes.forEach((n, i) => {
    const plan = { x: n.x, y: n.y, w: 22, d: 14, r: 2 };
    out.push(`<path class="iso-halo" d="${k.haloOf(plan, 10, 3, P)}" style="filter:blur(4px)"/>`);
    out.push(S(k.slabOf(plan, 10.5, 9, P, 3, 0.6), { tone: "mid" }));
    out.push(`<g transform="${frontPlane(n.y + 14)}">${flatLine(slots(n.x + 3, n.x + 13, -18, -12, 4))}<circle class="t-led" data-peer="${i}" cx="${n.x + 18}" cy="-15" r="1"/></g>`);
    out.push(L(k.planOutline({ x: n.x + 3, y: n.y + 3, w: 16, d: 8, r: 1 }, 19.5, P, 3), { tone: "faint" }));
  });
  return out.join("");
}

function plinth() {
  const out = [`<path class="iso-halo" d="${k.haloOf(PL, 0, 10, P)}"/>`];
  out.push(feetUnder(PL, 0, 4, 6));
  out.push(S(k.slabOf(PL, 4, 10, P, 8, 1.4), { tone: "mid" }));
  const mat = k.insetPlan(PL, 6);
  out.push(S(k.slabOf(mat, 14, 1, P, 8), { tone: "lo", crease: "none" }));
  out.push(L(k.planOutline(k.insetPlan(mat, 4), PL_TOP, P), { tone: "faint" }));
  const screws = k.corners(PL, 3.2);
  out.push(D(screws.map(([x, y]) => at([x, y, 14])), { size: 0.45 }));
  const snap = [mat.x + mat.w - 12, mat.y + mat.d - 10];
  out.push(S(k.cylinder(snap[0], snap[1], 2.4, PL_TOP, 1.2, P, 18), { tone: "mid" }));
  out.push(D([at([snap[0], snap[1], PL_TOP + 1.2])], { size: 0.5, tone: "hi" }));
  const strip = { x: mat.x + 10, y: mat.y + mat.d - 9, w: 120, d: 5, r: 0.8 };
  out.push(S(k.slabOf(strip, PL_TOP, 0.5, P, 2), { tone: "lo", crease: "none" }));
  const ticks = k.topTicks(strip.x + 3, strip.x + strip.w - 3, 4, 5, strip.y + 0.6, PL_TOP + 0.5, [1.6, 3], P);
  out.push(L(ticks.minor, { tone: "lo" }), L(ticks.major, { tone: "mid" }));
  out.push(labelPlate({ x: mat.x + 136, y: mat.y + mat.d - 12, w: 30, d: 8, r: 1 }, PL_TOP));
  return out.join("");
}

function caseFeet() {
  const plan = { x: T.x0, y: T.y0, w: T.x1 - T.x0, d: T.y1 - T.y0, r: 2 };
  const out = [`<path class="iso-halo" d="${k.haloOf(plan, PL_TOP, 6, P)}" style="filter:blur(5px)"/>`];
  for (const [x, y] of byDepth(k.corners(plan, 10))) {
    out.push(S(k.cylinder(x, y, 5, PL_TOP, 4, P, 20), { tone: "lo" }));
  }
  return out.join("");
}

function shellBack() {
  const out = [];
  out.push(S(box(T.x0 - 5, T.y0, T.z0, T.x0, T.y1 + 1.5, T.z1, { r: 1.5, steps: 3 }), { tone: "mid" }));
  out.push(S(box(T.x0, T.y0, T.z0, T.x1, T.y0 + WALL, T.z1, { r: 0.4 }), { tone: "mid" }));
  out.push(S(box(T.x0, T.y0, T.z0, T.x1, T.y1, T.z0 + WALL, { r: 0.4 }), { tone: "mid" }));
  out.push(S(box(T.x0, T.y0, T.z0, T.x0 + WALL, T.y1, T.z1, { r: 0.4 }), { tone: "mid" }));
  const inner = T.y0 + WALL;
  const grommets = [
    rr(BOARD.x0 - 7, -(BOARD.z1 - 6), BOARD.x0 - 3, -(BOARD.z1 - 30), 2),
    rr(BOARD.x0 - 7, -(BOARD.z0 + 34), BOARD.x0 - 3, -(BOARD.z0 + 10), 2),
    rr(BOARD.x0 + 6, -(PSU.z1 + 3.4), BOARD.x0 + 30, -(PSU.z1 + 1.2), 1),
  ].join("");
  out.push(`<g transform="${frontPlane(inner)}"><path class="t-recess" d="${grommets}"/>${flatLine(grommets, "lo")}</g>`);
  out.push(`<g transform="${plane([T.x0 + WALL, 0, 0], [0, 1, 0], [0, 0, -1])}">${flatLine(rr(T.y0 + 10, -(T.z1 - 10), T.y1 - 10, -(T.z1 - 40), 2) + slots(T.y0 + 12, T.y1 - 12, -(T.z1 - 36), -(T.z1 - 14), 10), "faint")}</g>`);
  return out.join("");
}

function motherboard() {
  const out = [];
  const plan = { x: BOARD.x0, y: BOARD.y, w: BOARD.x1 - BOARD.x0, d: 1.6, r: 0.4 };
  for (const [x, z] of [[BOARD.x0 + 4, BOARD.z0 + 4], [BOARD.x1 - 4, BOARD.z0 + 4], [BOARD.x0 + 4, BOARD.z1 - 4], [BOARD.x1 - 4, BOARD.z1 - 4], [BOARD.x0 + 4, (BOARD.z0 + BOARD.z1) / 2], [BOARD.x1 - 4, (BOARD.z0 + BOARD.z1) / 2]]) {
    out.push(D([at([x, BOARD.y + 1.6, z])], { size: 0.55, tone: "hi" }));
  }
  out.push(S(box(plan.x, plan.y, BOARD.z0, plan.x + plan.w, plan.y + plan.d, BOARD.z1), { tone: "mid" }));
  const face = BOARD.y + 1.6;
  const traces = [];
  const trace = (pts) => traces.push(k.pathOf(pts.map(([x, z]) => [x, -z])));
  for (let i = 0; i < 6; i++) trace([[CPU.x - 12, CPU.z - 18 - i * 2], [CPU.x - 22 - i * 2, CPU.z - 28 - i * 2], [CPU.x - 22 - i * 2, BOARD.z0 + 30]]);
  for (let i = 0; i < 5; i++) trace([[CPU.x + 14, CPU.z + 12 - i * 2.2], [BOARD.x1 - 18, CPU.z + 12 - i * 2.2]]);
  for (let i = 0; i < 4; i++) trace([[BOARD.x1 - 30 + i * 2, BOARD.z0 + 14], [BOARD.x1 - 30 + i * 2, BOARD.z0 + 40], [BOARD.x1 - 20 + i * 2, BOARD.z0 + 50]]);
  const pads = [];
  const ends = (d) => d;
  const tracePads = [];
  for (let i = 0; i < 6; i++) tracePads.push([CPU.x - 12, -(CPU.z - 18 - i * 2)], [CPU.x - 22 - i * 2, -(BOARD.z0 + 30)]);
  for (let i = 0; i < 5; i++) tracePads.push([CPU.x + 14, -(CPU.z + 12 - i * 2.2)], [BOARD.x1 - 18, -(CPU.z + 12 - i * 2.2)]);
  for (let i = 0; i < 4; i++) tracePads.push([BOARD.x1 - 30 + i * 2, -(BOARD.z0 + 14)], [BOARD.x1 - 20 + i * 2, -(BOARD.z0 + 50)]);
  const padSvg = tracePads.map(([x, y]) => `<circle cx="${f2(x)}" cy="${f2(y)}" r=".45"/>`).join("");
  out.push(`<g transform="${frontPlane(face)}">${flatLine(traces.join(""), "faint")}<g class="t-pads">${padSvg}</g>${flatLine(rr(BOARD.x0 + 2, -(BOARD.z1 - 2), BOARD.x1 - 2, -(BOARD.z0 + 2), 1), "faint")}</g>`);
  void pads; void ends;
  for (const z of [92, 80]) {
    out.push(S(box(BOARD.x0 + 6, face, z, BOARD.x1 - 14, face + 2.4, z + 3), { tone: "lo" }));
    out.push(`<g transform="${frontPlane(face + 2.4)}">${flatLine(rr(BOARD.x0 + 7.5, -(z + 2.2), BOARD.x1 - 15.5, -(z + 0.8), 0.4), "faint")}</g>`);
  }
  out.push(S(box(BOARD.x1 - 26, face, BOARD.z0 + 6, BOARD.x1 - 8, face + 4, BOARD.z0 + 20, { r: 1 }), { tone: "mid" }));
  out.push(`<g transform="${frontPlane(face + 4)}">${flatLine(slots(BOARD.x1 - 24, BOARD.x1 - 10, -(BOARD.z0 + 18), -(BOARD.z0 + 8), 6), "lo")}</g>`);
  out.push(S(box(BOARD.x0 + 2, face, BOARD.z0 + 20, BOARD.x0 + 6, face + 6, BOARD.z0 + 30), { tone: "lo" }));
  out.push(S(box(BOARD.x0 + 2, face, BOARD.z0 + 42, BOARD.x0 + 6, face + 9, BOARD.z0 + 70), { tone: "mid" }));
  out.push(`<g transform="${plane([BOARD.x0 + 6, 0, 0], [0, -1, 0], [0, 0, -1])}">${flatLine(Array.from({ length: 12 }, (_, i) => circle(-(face + 4.5), -(BOARD.z0 + 44 + i * 2.1), 0.6)).join(""), "lo")}</g>`);
  return out.join("");
}

function vrm() {
  const out = [];
  const face = BOARD.y + 1.6;
  out.push(S(box(CPU.x - 24, face, BOARD.z1 - 11, CPU.x + 14, face + 7, BOARD.z1 - 3, { r: 1 }), { tone: "mid" }));
  out.push(`<g transform="${frontPlane(face + 7)}">${flatLine(slots(CPU.x - 22, CPU.x + 12, -(BOARD.z1 - 5), -(BOARD.z1 - 9), 9), "lo")}</g>`);
  out.push(S(box(CPU.x - 32, face, CPU.z - 18, CPU.x - 25, face + 7, BOARD.z1 - 13, { r: 1 }), { tone: "mid" }));
  out.push(`<g transform="${frontPlane(face + 7)}">${flatLine(slots(CPU.x - 31, CPU.x - 26, -(BOARD.z1 - 15), -(CPU.z - 16), 7, false), "lo")}</g>`);
  return out.join("");
}

function ram() {
  const out = [];
  const face = BOARD.y + 1.6;
  for (const x of [CPU.x + 22, CPU.x + 27, CPU.x + 32, CPU.x + 37]) {
    out.push(S(box(x - 1, face, CPU.z - 26, x + 1, face + 2, CPU.z + 26), { tone: "lo" }));
  }
  for (const x of [CPU.x + 22, CPU.x + 32]) {
    out.push(S(box(x - 1.3, face + 2, CPU.z - 24, x + 1.3, face + 12, CPU.z + 24, { r: 0.6 }), { tone: "mid", inner: L(k.sideSeam({ x: x - 1.3, y: face + 2, w: 2.6, d: 10, r: 0.6 }, CPU.z + 16, P, 2), { tone: "faint" }) }));
    out.push(D([at([x, face + 12, CPU.z - 20]), at([x, face + 12, CPU.z + 20])], { size: 0.4 }));
  }
  return out.join("");
}

function cooler() {
  const out = [];
  const face = BOARD.y + 1.6;
  out.push(S(box(CPU.x - 17, face, CPU.z - 17, CPU.x + 17, face + 2.2, CPU.z + 17, { r: 1.5 }), { tone: "lo" }));
  out.push(D([[-14, -14], [14, -14], [-14, 14], [14, 14]].map(([dx, dz]) => at([CPU.x + dx, face + 2.2, CPU.z + dz])), { size: 0.5, tone: "hi" }));
  const F = G.frameAlong([CPU.x, face + 2.2, CPU.z], [0, 1, 0]);
  const sinkTo = FAN_Y - 3.4 - (face + 2.2);
  const core = G.solidOf([[0, 6], [sinkTo, 6]], F, P);
  const sink = G.solidOf([[0, 15], [sinkTo, 15]], F, P);
  const ribs = G.ribsOf(F, 0, sinkTo, 15, 40, P, { fade: [0.05, 0.5], seams: true });
  out.push(S(sink, { tone: "mid", inner: k.fadedSvg(ribs, { tone: "lo" }) }));
  void core;
  const frame = { x0: CPU.x - 18, x1: CPU.x + 18, z0: CPU.z - 18, z1: CPU.z + 18 };
  out.push(S(box(frame.x0, FAN_Y - 3.4, frame.z0, frame.x1, FAN_Y, frame.z1, { r: 2.4, steps: 4 }), { tone: "hi", crease: "lo" }));
  const holes = [[-14.6, -14.6], [14.6, -14.6], [-14.6, 14.6], [14.6, 14.6]];
  out.push(`<g data-res-part="fan" transform="${frontPlane(FAN_Y)}">${fanFace(CPU.x, -CPU.z, 16.4, "cpufan", { blades: 9 })}${holes.map(([dx, dz]) => `<circle class="t-screw" cx="${f2(CPU.x + dx)}" cy="${f2(-(CPU.z + dz))}" r=".8"/>`).join("")}</g>`);
  return out.join("");
}

function sessionBar() {
  const out = [];
  const z1 = CPU.z - 18.4;
  const z0 = z1 - 6;
  out.push(S(box(CPU.x - 14, FAN_Y - 3, z0, CPU.x + 14, FAN_Y + 0.6, z1, { r: 0.8 }), { tone: "mid", crease: "lo" }));
  const lamps = [0, 1, 2, 3].map((i) => {
    const cx = CPU.x - 9 + i * 6;
    return `<circle class="t-bezel" cx="${f2(cx)}" cy="${f2(-(z0 + 3))}" r="1.9"/><circle class="t-lamp" data-lamp="${i}" cx="${f2(cx)}" cy="${f2(-(z0 + 3))}" r="1.3"/>`;
  }).join("");
  out.push(`<g transform="${frontPlane(FAN_Y + 0.6)}">${lamps}<circle class="t-screw" cx="${CPU.x - 12.4}" cy="${f2(-(z0 + 3))}" r=".6"/><circle class="t-screw" cx="${CPU.x + 12.4}" cy="${f2(-(z0 + 3))}" r=".6"/></g>`);
  return out.join("");
}

function atxCable() {
  const route = G.fillet([[-372, -47, PSU.z1], [-372, -47, 84], [BOARD.x0 + 4, -51.5, 100], [BOARD.x0 + 4, -51.5, BOARD.z0 + 42]], 7, 8, { P, tube: 2.2 });
  return tubePieces(route, 2.2, P, { maxLength: 80, rings: { pitch: 1.6, twist: 0.6, cross: true } }).map((piece) => tubeSvg(piece, { tone: "mid" })).join("") +
    S(box(-376, -51, PSU.z1, -368, -43, PSU.z1 + 1, { r: 1.5 }), { tone: "lo" });
}

function psuShroud() {
  const out = [];
  const y0 = T.y0 + WALL;
  const y1 = T.y1 - 3;
  out.push(S(box(PSU.x0, y0, T.z0 + WALL, PSU.x1, y1, PSU.z1, { r: 1.2, steps: 3 }), { tone: "mid", inner: L(k.sideSeam({ x: PSU.x0, y: y0, w: PSU.x1 - PSU.x0, d: y1 - y0, r: 1.2 }, PSU.z1 - 6, P, 3), { tone: "faint" }) }));
  out.push(`<g transform="${topPlane(PSU.z1)}">${flatLine(slots(PSU.x0 + 10, PSU.x0 + 40, y0 + 8, y1 - 8, 8), "lo")}</g>`);
  out.push(`<g transform="${frontPlane(y1)}">${flatLine(rr(PSU.x0 + 50, -(PSU.z1 - 14), PSU.x1 - 6, -(PSU.z0 + 12), 1.2), "lo")}${flatLine(Array.from({ length: 3 }, (_, i) => `M${PSU.x0 + 54} ${-(PSU.z1 - 18 - i * 3)}H${PSU.x0 + 54 + [18, 12, 15][i]}`).join(""), "lo", ' data-free=""')}</g>`);
  out.push(D([at([PSU.x0 + 4, y1, PSU.z1 - 3]), at([PSU.x1 - 4, y1, PSU.z1 - 3])], { size: 0.45 }));
  return out.join("");
}

function driveCage() {
  const out = [];
  const y0 = T.y0 + WALL;
  out.push(S(box(CAGE.x0, y0, CAGE.z0, CAGE.x0 + 2, SLED_Y - 1, CAGE.z1), { tone: "mid" }));
  SLED_Z.forEach((z, i) => out.push(sled(i, z)));
  out.push(S(box(CAGE.x1 - 2, y0, CAGE.z0, CAGE.x1, SLED_Y - 1, CAGE.z1), { tone: "mid" }));
  out.push(`<g transform="${plane([CAGE.x1, 0, 0], [0, -1, 0], [0, 0, -1])}">${flatLine(Array.from({ length: 4 }, (_, i) => rr(-(SLED_Y - 6), -(SLED_Z[i] + 8), -(y0 + 6), -(SLED_Z[i] + 10), 0.8)).join(""), "faint")}</g>`);
  out.push(S(box(CAGE.x0, y0, CAGE.z1, CAGE.x1, SLED_Y - 1, CAGE.z1 + 2), { tone: "mid" }));
  out.push(D([CAGE.x0 + 4, CAGE.x1 - 4].map((x) => at([x, SLED_Y - 4, CAGE.z1 + 2])), { size: 0.45 }));
  return out.join("");
}

function sled(i, z) {
  const out = [];
  const x0 = CAGE.x0 + 2.2;
  const x1 = CAGE.x1 - 2.2;
  const y0 = T.y0 + WALL + 4;
  out.push(S(box(x0 + 1, y0, z + 2, x1 - 1, SLED_Y - 2, z + SLED_H - 2), { tone: "lo" }));
  out.push(S(box(x0, SLED_Y - 2, z, x1, SLED_Y, z + SLED_H, { r: 0.8, steps: 3, bevel: 0.4 }), { tone: "mid", crease: "lo" }));
  const fy = -z;
  const disk = NODES.home.disks[i];
  const g0 = x0 + 15;
  const g1 = x1 - 9;
  const share = disk.used / disk.size;
  const face = [
    `<path class="t-recess" d="${rr(x0 + 2, fy - 5, x0 + 6.6, fy - 15, 1.4)}"/>`,
    flatLine(rr(x0 + 2, fy - 5, x0 + 6.6, fy - 15, 1.4), "lo"),
    flatLine(rr(x0 + 8.5, fy - 12.2, x0 + 13, fy - 15.6, 0.4), "faint"),
    flatLine(`M${x0 + 9.4} ${fy - 14.4}H${x0 + 12.2}`, "lo", ' data-free=""'),
    `<path class="t-gauge-well" d="${rr(g0, fy - 4.4, g1, fy - 7.4, 0.5)}"/>`,
    `<path class="t-gauge" d="${rr(g0 + 0.5, fy - 4.9, g0 + 0.5 + (g1 - g0 - 1) * share, fy - 6.9, 0.3)}"/>`,
    flatLine(rr(g0, fy - 4.4, g1, fy - 7.4, 0.5), "lo"),
    flatLine(Array.from({ length: 11 }, (_, t) => `M${f2(g0 + ((g1 - g0) * t) / 10)} ${f2(fy - 4.4)}V${f2(fy - (t % 5 === 0 ? 3 : 3.6))}`).join(""), "lo"),
    `<path class="t-plate" d="${rr(g0, fy - 10, g0 + 18, fy - 15.4, 0.6)}"/>`,
    flatLine(rr(g0, fy - 10, g0 + 18, fy - 15.4, 0.6), "lo"),
    flatLine(`M${g0 + 2} ${fy - 11.8}H${g0 + 2 + [13, 11, 9, 12][i]}M${g0 + 2} ${fy - 13.6}H${g0 + 2 + [8, 9, 6, 10][i]}`, "lo", ' data-free=""'),
    flatLine(slots(g0 + 21, g1, fy - 10.2, fy - 15.4, 6), "faint"),
    `<circle class="t-bezel" cx="${f2(x1 - 4.4)}" cy="${f2(fy - 5.9)}" r="1.9"/>`,
    `<circle class="t-led-read" data-disk-read="${i}" cx="${f2(x1 - 4.4)}" cy="${f2(fy - 5.9)}" r="1.25"/>`,
    `<circle class="t-led-write" data-disk-write="${i}" cx="${f2(x1 - 4.4)}" cy="${f2(fy - 5.9)}" r="1.25"/>`,
    `<circle class="t-screw" cx="${f2(x1 - 4.4)}" cy="${f2(fy - 13.4)}" r=".7"/>`,
  ];
  out.push(`<g transform="${frontPlane(SLED_Y)}">${face.join("")}</g>`);
  return out.join("");
}

function shellFront() {
  const out = [];
  out.push(S(box(T.x0, T.y0, T.z1 - WALL, T.x1, T.y1, T.z1, { r: 0.6 }), { tone: "mid" }));
  out.push(`<g transform="${topPlane(T.z1)}">${flatLine(slots(T.x0 + 8, T.x0 + 30, T.y0 + 8, T.y1 - 8, 6) + slots(T.x1 - 30, T.x1 - 8, T.y0 + 8, T.y1 - 8, 6), "faint")}</g>`);
  out.push(S(box(T.x1 - WALL, T.y0, T.z0, T.x1, T.y1, T.z1 - WALL, { r: 0.4 }), { tone: "mid" }));
  const X = T.x1;
  const u = (y) => -y;
  const v = (z) => -z;
  const io = { y0: -60, y1: -46, z0: 112, z1: 166 };
  const parts = [];
  parts.push(`<path class="t-recess" d="${rr(u(io.y1), v(io.z1), u(io.y0), v(io.z0), 0.6)}"/>`);
  parts.push(flatLine(rr(u(io.y1), v(io.z1), u(io.y0), v(io.z0), 0.6), "lo"));
  const port = (y, z, w, h) => flatLine(rr(u(y + w), v(z + h), u(y), v(z), 0.4), "mid");
  parts.push(port(-58, 158, 5, 4), port(-52, 158, 5, 4), port(-58, 152, 5, 4), port(-52, 152, 5, 4));
  parts.push(port(-58, 144, 8, 3.4), port(-58, 138, 8, 3.4));
  parts.push(port(-58, 124, 8, 6));
  for (const [i, z] of [116, 113].entries()) void i, parts.push(flatLine(circle(u(-56), v(z), 1), "lo"), flatLine(circle(u(-51), v(z), 1), "lo"));
  parts.push(`<circle class="t-led" data-net-led="0" cx="${u(-57)}" cy="${v(129)}" r=".7"/><circle class="t-led" data-net-led="1" cx="${u(-51.6)}" cy="${v(129)}" r=".7"/>`);
  const fan = { y: -24, z: 145, r: 15 };
  parts.push(flatLine(rr(u(fan.y + 17), v(fan.z + 17), u(fan.y - 17), v(fan.z - 17), 2.4), "lo"));
  parts.push(fanFace(u(fan.y), v(fan.z), fan.r, "rearfan", { blades: 7 }));
  parts.push([[-14, -14], [14, -14], [-14, 14], [14, 14]].map(([dy, dz]) => `<circle class="t-screw" cx="${u(fan.y + dy)}" cy="${v(fan.z + dz)}" r=".8"/>`).join(""));
  for (let i = 0; i < 4; i++) {
    const z = 72 + i * 9;
    parts.push(flatLine(rr(u(-46), v(z + 6.6), u(-60), v(z), 0.4), "lo"));
    parts.push(flatLine(slots(u(-47.5), u(-58.5), v(z + 5), v(z + 1.6), 1, false), "faint"));
    parts.push(`<circle class="t-screw" cx="${u(-44)}" cy="${v(z + 3.3)}" r=".6"/>`);
  }
  parts.push(flatLine(rr(u(-6), v(PSU.z1 - 2), u(-60), v(PSU.z0 + 1), 1), "lo"));
  const psuFan = { y: -42, z: 41, r: 14 };
  parts.push(`<path class="t-hole" d="${circle(u(psuFan.y), v(psuFan.z), psuFan.r)}"/>`);
  parts.push(flatLine([0.95, 0.75, 0.55, 0.35, 0.15].map((s) => circle(u(psuFan.y), v(psuFan.z), f2(psuFan.r * s))).join(""), "mid"));
  parts.push(flatLine(`M${u(psuFan.y + psuFan.r * 0.95)} ${v(psuFan.z)}H${u(psuFan.y - psuFan.r * 0.95)}M${u(psuFan.y)} ${v(psuFan.z + psuFan.r * 0.95)}V${v(psuFan.z - psuFan.r * 0.95)}`, "mid"));
  parts.push(`<path class="t-recess" d="${rr(u(-12), v(52), u(-24), v(44), 0.8)}"/>`, flatLine(rr(u(-12), v(52), u(-24), v(44), 0.8), "lo"));
  parts.push(flatLine(rr(u(-14), v(37), u(-22), v(30), 0.5), "lo"));
  parts.push(`<path class="t-plate" d="${rr(u(-9), v(37), u(-25), v(26), 0.6)}" opacity="0"/>`);
  parts.push(`<circle class="t-screw" cx="${u(-8)}" cy="${v(PSU.z1 - 5)}" r=".7"/><circle class="t-screw" cx="${u(-58)}" cy="${v(PSU.z1 - 5)}" r=".7"/><circle class="t-screw" cx="${u(-8)}" cy="${v(PSU.z0 + 4)}" r=".7"/><circle class="t-screw" cx="${u(-58)}" cy="${v(PSU.z0 + 4)}" r=".7"/>`);
  out.push(`<g transform="${sidePlane(X)}">${parts.join("")}</g>`);
  out.push(S(box(X, -51, 124.6, X + 5, -53 + 7, 129.4, { r: 0.6 }), { tone: "hi" }));
  return out.join("");
}

function powerLed() {
  return `<g transform="${frontPlane(T.y1)}"><circle class="t-bezel" cx="${T.x0 - 2.5}" cy="${-(T.z1 - 8)}" r="1.5"/><circle class="t-power" data-part="power-led" cx="${T.x0 - 2.5}" cy="${-(T.z1 - 8)}" r="1"/></g>`;
}

const STRIP = { x: -290, y: -72, w: 44, d: 10, r: 2 };
function powerStrip() {
  const out = [];
  out.push(`<path class="iso-halo" d="${k.haloOf(STRIP, PL_TOP, 3, P)}" style="filter:blur(3px)"/>`);
  out.push(S(k.slabOf(STRIP, PL_TOP, 5, P, 3, 0.5), { tone: "mid", inner: L(k.sideSeam(STRIP, PL_TOP + 1.6, P, 3), { tone: "faint" }) }));
  const sockets = [0, 1, 2, 3].map((i) => k.planOutline({ x: STRIP.x + 4 + i * 8.6, y: STRIP.y + 2, w: 6, d: 6, r: 1.4 }, PL_TOP + 5, P, 3)).join("");
  out.push(L(sockets, { tone: "lo" }));
  out.push(D([0, 1, 2, 3].flatMap((i) => [at([STRIP.x + 6.2 + i * 8.6, STRIP.y + 5, PL_TOP + 5]), at([STRIP.x + 7.8 + i * 8.6, STRIP.y + 5, PL_TOP + 5])]), { size: 0.35, tone: "lo" }));
  out.push(`<g transform="${frontPlane(STRIP.y + STRIP.d)}"><circle class="t-bezel" cx="${STRIP.x + STRIP.w - 3}" cy="${-(PL_TOP + 2.6)}" r="1.2"/><circle class="t-power-strip" cx="${STRIP.x + STRIP.w - 3}" cy="${-(PL_TOP + 2.6)}" r=".8"/></g>`);
  return out.join("");
}
const plugOn = (i) => [STRIP.x + 7 + i * 8.6, STRIP.y + 5, PL_TOP + 5];
function plugs() {
  const out = [];
  for (const i of [0, 2]) {
    const [x, y, z] = plugOn(i);
    out.push(S(box(x - 3, y - 3, z, x + 3, y + 3, z + 4, { r: 1 }), { tone: "mid" }));
  }
  return out.join("");
}
function powerCords() {
  const X = T.x1;
  const a = plugOn(0);
  const b = plugOn(2);
  return S(box(X, -21, 44, X + 4, -15, 52, { r: 0.8 }), { tone: "mid" }) +
    cable([[X + 4, -18, 48], [X + 10, -18, 48], [X + 10, -18, 21], [X + 10, -40, PL_TOP + 1.3], [a[0], -40, PL_TOP + 1.3], [a[0], a[1] + 9, PL_TOP + 7], [a[0], a[1], a[2] + 10], [a[0], a[1], a[2] + 4]], 1.2, { tone: "mid", bend: 5 }) +
    cable([[ROUTER.x + 34, ROUTER.y - 0.4, PL_TOP + 6], [ROUTER.x + 34, ROUTER.y - 8, PL_TOP + 1.2], [b[0], ROUTER.y - 12, PL_TOP + 1.2], [b[0], b[1] + 9, PL_TOP + 7], [b[0], b[1], b[2] + 10], [b[0], b[1], b[2] + 4]], 0.8, { tone: "mid", bend: 4 });
}

const ETH_PORT = [T.x1 + 5, -48.5, 127];
const ROUTER_LAN = [ROUTER.x - 0.5, ROUTER.y + 10, PL_TOP + 5];
const ROUTER_WAN = [ROUTER.x + ROUTER.w + 0.5, ROUTER.y + 15, PL_TOP + 5];
const COUPLER_HOME = { x: TRAY.x0, y: (TRAY.y0 + TRAY.y1) / 2, z: 13 };
const COUPLER_RELAY = { x: (TRAY.xa + TRAY.xb) / 2, y: TRAY.yEnd - 9, z: 13, axis: "y" };

function ethernet() {
  return cable([ETH_PORT, [ETH_PORT[0] + 5, ETH_PORT[1], ETH_PORT[2]], [ETH_PORT[0] + 7, ETH_PORT[1], PL_TOP + 9], [ETH_PORT[0] + 10, ROUTER_LAN[1] - 4, PL_TOP + 1.4], [ROUTER_LAN[0] - 6, ROUTER_LAN[1], PL_TOP + 1.4], [ROUTER_LAN[0] - 2, ROUTER_LAN[1], ROUTER_LAN[2]], ROUTER_LAN], 1.1, { tone: "mid", bend: 5 });
}

function router() {
  const out = [];
  const plan = { ...ROUTER, r: 3 };
  out.push(`<path class="iso-halo" d="${k.haloOf(plan, PL_TOP, 4, P)}" style="filter:blur(5px)"/>`);
  for (const [x, y] of byDepth(k.corners(plan, 5))) out.push(S(k.cylinder(x, y, 2.4, PL_TOP, 1.6, P, 14), { tone: "lo" }));
  const z0 = PL_TOP + 1.6;
  const z1 = z0 + 9;
  for (const [x, y] of [[plan.x + 7, plan.y + 3], [plan.x + plan.w - 7, plan.y + 3]]) {
    out.push(S(k.cylinder(x, y, 2.2, z1, 3, P, 16), { tone: "mid" }));
    out.push(S(k.cylinder(x, y, 1.3, z1 + 3, 30, P, 14), { tone: "mid", crease: "none" }));
    out.push(L(k.sideArc(x, y, 1.3, z1 + 12, P, 10), { tone: "faint" }));
    out.push(S(k.cylinder(x, y, 1.5, z1 + 33, 1.4, P, 14), { tone: "mid" }));
  }
  out.push(S(k.slabOf(plan, z0, 9, P, 4, 1), { tone: "mid", inner: L(k.sideSeam(plan, z0 + 2.4, P, 4), { tone: "faint" }) }));
  out.push(`<g transform="${topPlane(z1)}">${flatLine(slots(plan.x + 12, plan.x + plan.w - 12, plan.y + 9, plan.y + plan.d - 6, 9), "faint")}</g>`);
  const leds = [0, 1, 2, 3].map((i) => `<circle class="t-bezel" cx="${plan.x + 8 + i * 6}" cy="${-(z0 + 5.4)}" r="1.2"/><circle class="t-led" data-router-led="${i}" cx="${plan.x + 8 + i * 6}" cy="${-(z0 + 5.4)}" r=".8"/>`).join("");
  out.push(`<g transform="${frontPlane(plan.y + plan.d)}">${leds}${flatLine(`M${plan.x + 34} ${-(z0 + 6.5)}H${plan.x + 40}M${plan.x + 34} ${-(z0 + 4.4)}H${plan.x + 38}`, "lo", ' data-free=""')}</g>`);
  const wan = ROUTER_WAN;
  out.push(`<g transform="${sidePlane(plan.x + plan.w)}">${flatLine(rr(-(wan[1] + 3.4), -(wan[2] + 2.6), -(wan[1] - 3.4), -(wan[2] - 2.6), 0.4), "mid")}${flatLine(rr(-(wan[1] - 6), -(wan[2] + 2.6), -(wan[1] - 12.8), -(wan[2] - 2.6), 0.4), "lo")}<circle class="t-led" data-router-led="4" cx="${-(wan[1] - 9.4)}" cy="${-(wan[2] + 4.4)}" r=".6"/></g>`);
  return out.join("");
}

function wanCable() {
  const c = COUPLER_HOME;
  return S(box(ROUTER_WAN[0], ROUTER_WAN[1] - 2, ROUTER_WAN[2] - 1.8, ROUTER_WAN[0] + 4, ROUTER_WAN[1] + 2, ROUTER_WAN[2] + 1.8, { r: 0.5 }), { tone: "hi" }) +
    cable([[ROUTER_WAN[0] + 4, ROUTER_WAN[1], ROUTER_WAN[2]], [ROUTER_WAN[0] + 12, ROUTER_WAN[1], ROUTER_WAN[2]], [PL.x + PL.w + 4, c.y, PL_TOP + 3], [c.x - 18, c.y, c.z], [c.x - 9, c.y, c.z]], 1.2, { tone: "mid", bend: 6 });
}

const LRTV_ROUTE = [[ROUTER.x + 30, ROUTER.y + ROUTER.d + 2.4, PL_TOP + 5.4], [ROUTER.x + 30, ROUTER.y + ROUTER.d + 8, PL_TOP + 1.2], [(LRTV.x + LRTV.x1) / 2, LRTV.y - 24, PL_TOP + 1.2], [(LRTV.x + LRTV.x1) / 2, LRTV.y - 8.4, PL_TOP + 1.2]];
function lrtvCable() {
  return S(box(ROUTER.x + 27, ROUTER.y + ROUTER.d, PL_TOP + 3.4, ROUTER.x + 33, ROUTER.y + ROUTER.d + 2.4, PL_TOP + 7.4, { r: 0.5 }), { tone: "hi" }) + cable(LRTV_ROUTE, 0.8, { tone: "mid", bend: 5 });
}

function livingRoomTv() {
  const out = [];
  const tv = { x0: LRTV.x, x1: LRTV.x1, y0: LRTV.y, y1: LRTV.y + 4, z0: PL_TOP + 16, z1: PL_TOP + 52 };
  const foot = { x: (tv.x0 + tv.x1) / 2 - 14, y: tv.y0 - 8, w: 28, d: 16, r: 3 };
  out.push(`<path class="iso-halo" d="${k.haloOf(foot, PL_TOP, 3, P)}" style="filter:blur(4px)"/>`);
  out.push(S(k.slabOf(foot, PL_TOP, 2, P, 4, 0.5), { tone: "mid" }));
  out.push(S(box((tv.x0 + tv.x1) / 2 - 3, tv.y0 - 3, PL_TOP + 2, (tv.x0 + tv.x1) / 2 + 3, tv.y0, tv.z0 + 6, { r: 0.6 }), { tone: "mid" }));
  out.push(S(box(tv.x0 + 8, tv.y0 - 4, tv.z0 + 4, tv.x1 - 8, tv.y0, tv.z1 - 6, { r: 1 }), { tone: "lo" }));
  out.push(S(box(tv.x0, tv.y0, tv.z0, tv.x1, tv.y1, tv.z1, { r: 0.8, bevel: 0.4 }), { tone: "mid" }));
  out.push(screen("lrtv", tv.y1, tv.x0 + 2, tv.z0 + 2.5, tv.x1 - 2, tv.z1 - 2));
  return out.join("");
}

function screen(id, y, x0, z0, x1, z1) {
  const w = x1 - x0;
  const h = z1 - z0;
  const cid = `clip-${id}`;
  const rect = rr(x0, -z1, x1, -z0, 0.6);
  const sky = [];
  return `<g transform="${frontPlane(y)}"><clipPath id="${cid}"><path d="${rect}"/></clipPath><path class="t-glass" d="${rect}"/><g clip-path="url(#${cid})" class="t-pic" data-screen="${id}" opacity="0">` +
    `<rect class="t-pic-sky" x="${x0}" y="${-z1}" width="${w}" height="${f2(h * 0.62)}"/>` +
    `<g data-pan="${id}"><path class="t-pic-hill" d="M${x0 - w} ${f2(-z0 - h * 0.2)}${Array.from({ length: 9 }, (_, i) => `Q${f2(x0 - w + (i + 0.5) * (w / 3))} ${f2(-z0 - h * (0.42 + 0.12 * Math.sin(i * 1.7)))} ${f2(x0 - w + (i + 1) * (w / 3))} ${f2(-z0 - h * (0.22 + 0.06 * Math.cos(i)))}`).join("")}V${-z0}H${x0 - w}Z"/>` +
    `<path class="t-pic-hill2" d="M${x0 - w} ${f2(-z0 - h * 0.12)}${Array.from({ length: 12 }, (_, i) => `Q${f2(x0 - w + (i + 0.5) * (w / 4))} ${f2(-z0 - h * (0.26 + 0.08 * Math.cos(i * 2.1)))} ${f2(x0 - w + (i + 1) * (w / 4))} ${f2(-z0 - h * 0.12)}`).join("")}V${-z0}H${x0 - w}Z"/></g>` +
    `<circle class="t-pic-sun" cx="${f2(x0 + w * 0.7)}" cy="${f2(-z1 + h * 0.3)}" r="${f2(h * 0.12)}"/>` +
    `<rect class="t-pic-bar" x="${f2(x0 + w * 0.08)}" y="${f2(-z0 - h * 0.1)}" width="${f2(w * 0.84)}" height="${f2(Math.max(0.6, h * 0.035))}"/>` +
    `<rect class="t-pic-prog" data-prog="${id}" x="${f2(x0 + w * 0.08)}" y="${f2(-z0 - h * 0.1)}" width="0" height="${f2(Math.max(0.6, h * 0.035))}" data-w="${f2(w * 0.84)}"/>` +
    `</g><path class="t-glare" d="M${f2(x0 + w * 0.06)} ${f2(-z1 + h * 0.1)}L${f2(x0 + w * 0.3)} ${f2(-z1 + h * 0.1)}L${f2(x0 + w * 0.12)} ${f2(-z1 + h * 0.6)}Z"/>` +
    `<circle class="t-standby" data-standby="${id}" cx="${f2(x1 - 1.6)}" cy="${f2(-z0 + 1.2)}" r=".5"/>` +
    flatLine(rect, "lo") + `</g>`;
}

function trayBack() {
  const out = [];
  const { x0, xa, xb, y0, y1, yEnd, z0, z1 } = TRAY;
  const legX = [];
  for (let x = x0 + 20; x < xa - 6; x += 44) legX.push(x);
  const legY = [];
  for (let y = y0 - 16; y > yEnd + 4; y -= 34) legY.push(y);
  for (const y of legY) {
    out.push(S(box(xa - 3, y - 2, 0, xb + 3, y + 2, 1.5, { r: 0.5 }), { tone: "lo" }));
    out.push(S(box(xa + 2, y - 1.2, 1.5, xb - 2, y + 1.2, z0, { r: 0.3 }), { tone: "mid" }));
  }
  out.push(S(box(xa, yEnd, z0, xb, y0, z0 + 1.2, { r: 0.4 }), { tone: "lo" }));
  out.push(S(box(xa, yEnd, z0 + 1.2, xa + 1.2, y0, z1, { r: 0.3 }), { tone: "mid" }));
  for (const x of legX) {
    out.push(S(box(x - 2, y0 - 3, 0, x + 2, y1 + 3, 1.5, { r: 0.5 }), { tone: "lo" }));
    out.push(S(box(x - 1.2, y0 + 2, 1.5, x + 1.2, y1 - 2, z0, { r: 0.3 }), { tone: "mid" }));
  }
  out.push(S(box(xa + 2, y0 + 2, 0, xb - 2, y1 - 2, z0, { r: 0.6 }), { tone: "lo" }));
  out.push(S(box(x0, y0, z0, xb, y1, z0 + 1.2, { r: 0.4 }), { tone: "lo" }));
  out.push(S(box(x0, y0, z0 + 1.2, xa, y0 + 1.2, z1, { r: 0.3 }), { tone: "mid" }));
  const rungs = [];
  for (let x = x0 + 8; x < xa - 3; x += 10) rungs.push(rr(x, y0 + 2.4, x + 1.6, y1 - 2.4, 0.5));
  for (let y = y0 - 6; y > yEnd + 3; y -= 10) rungs.push(rr(xa + 2.4, y - 1.6, xb - 2.4, y, 0.5));
  const mid = (y0 + y1) / 2;
  const midX = (xa + xb) / 2;
  out.push(`<g transform="${topPlane(z0 + 1.2)}">${flatLine(rungs.join(""), "faint")}${flatLine(`M${x0 + 1} ${mid}H${midX}V${yEnd + 1}`, "faint")}</g>`);
  return out.join("");
}

function trayFront() {
  const out = [];
  const { x0, xa, xb, y0, y1, yEnd, z0, z1 } = TRAY;
  out.push(S(box(xb - 1.2, yEnd, z0 + 1.2, xb, y1, z1, { r: 0.3 }), { tone: "mid" }));
  out.push(S(box(x0, y1 - 1.2, z0 + 1.2, xb, y1, z1, { r: 0.3 }), { tone: "mid" }));
  const bolts = [];
  for (let x = x0 + 20; x < xa - 6; x += 44) bolts.push(at([x, y1, z0 + 4]));
  for (let y = y0 - 16; y > yEnd + 4; y -= 34) bolts.push(at([xb, y, z0 + 4]));
  out.push(D(bolts, { size: 0.5 }));
  const strip = [x0 + 34];
  out.push(`<g transform="${frontPlane(y1)}">${strip.map((x) => `<path class="t-plate" d="${rr(x, -(z1 - 1.6), x + 22, -(z0 + 2.6), 0.4)}"/>${flatLine(rr(x, -(z1 - 1.6), x + 22, -(z0 + 2.6), 0.4), "lo")}${flatLine(`M${x + 2} ${-(z1 - 3.6)}H${x + 14}M${x + 2} ${-(z1 - 5.6)}H${x + 10}`, "lo", ' data-free=""')}`).join("")}</g>`);
  out.push(`<g transform="${sidePlane(xb)}"><path class="t-plate" d="${rr(-(y0 - 12), -(z1 - 1.6), -(y0 - 34), -(z0 + 2.6), 0.4)}"/>${flatLine(rr(-(y0 - 12), -(z1 - 1.6), -(y0 - 34), -(z0 + 2.6), 0.4), "lo")}${flatLine(`M${-(y0 - 14)} ${-(z1 - 3.6)}H${-(y0 - 26)}M${-(y0 - 14)} ${-(z1 - 5.6)}H${-(y0 - 22)}`, "lo", ' data-free=""')}</g>`);
  return out.join("");
}

function coupler(c, side) {
  const out = [];
  const F = c.axis === "y" ? G.frameAlong([c.x, c.y + 9, c.z], [0, -1, 0]) : G.frameAlong([c.x - 9, c.y, c.z], [1, 0, 0]);
  const body = [[0, 3.2], [3, 3.2], [3, 6.2], [4.4, 7.4], [13.6, 7.4], [15, 6.2], [15, 3.2], [18, 3.2]];
  out.push(S(box(c.x - 6, c.y - 7, 0, c.x + 6, c.y + 7, 2, { r: 1 }), { tone: "lo" }));
  out.push(S(box(c.x - 3, c.y - 3, 2, c.x + 3, c.y + 3, c.z - 6, { r: 0.6 }), { tone: "mid" }));
  out.push(S(G.solidOf(body, F, P), { tone: "hi", crease: "lo", inner: L(G.arcOf(F, 9, 7.4, P), { tone: "faint" }) }));
  out.push(D(G.dotsOf(F, 4.4, 6.8, 8, P, { fade: [0.04, 0.42] }), { size: 0.45, tone: "mid" }));
  out.push(D(G.dotsOf(F, 13.6, 6.8, 8, P, { fade: [0.04, 0.42] }), { size: 0.45, tone: "mid" }));
  const lampAt = [c.x, c.y, c.z + 7.4];
  out.push(S(k.cylinder(lampAt[0], lampAt[1], 1.8, lampAt[2] - 0.4, 1.6, P, 14), { tone: "mid" }));
  out.push(`<ellipse class="t-status" data-status="${side}" cx="${f2(at([lampAt[0], lampAt[1], lampAt[2] + 1.2])[0])}" cy="${f2(at([lampAt[0], lampAt[1], lampAt[2] + 1.2])[1])}" rx="2.3" ry="1.3"/>`);
  return out.join("");
}

function tile() {
  const out = [`<path class="iso-halo" d="${k.haloOf(TILE, 0, 8, P)}"/>`];
  for (const [x, y] of byDepth(k.corners(TILE, 12))) {
    out.push(S(k.cylinder(x, y, 5, 0, 1.2, P, 18), { tone: "lo" }));
    out.push(S(k.cylinder(x, y, 2, 1.2, 4.8, P, 14), { tone: "mid", crease: "none" }));
    out.push(S(k.cylinder(x, y, 4, 6, 0.8, P, 16), { tone: "mid" }));
  }
  out.push(S(k.slabOf(TILE, 6.8, TILE_TOP - 6.8, P, 4, 1), { tone: "mid" }));
  const half = { x: TILE.x + TILE.w / 2 };
  out.push(L(k.lineOnTop([half.x, TILE.y], [half.x, TILE.y + TILE.d], TILE_TOP, P) + k.lineOnTop([TILE.x, TILE.y + TILE.d / 2], [TILE.x + TILE.w, TILE.y + TILE.d / 2], TILE_TOP, P), { tone: "faint" }));
  const grid = { x: TILE.x + TILE.w / 2 + 8, y: TILE.y + TILE.d / 2 + 8 };
  const holes = [];
  for (let i = 0; i < 9; i++) for (let j = 0; j < 6; j++) holes.push(at([grid.x + 3 + i * 6.4, grid.y + 3 + j * 6.4, TILE_TOP]));
  out.push(D(holes, { size: 0.7, tone: "lo" }));
  out.push(L(k.planOutline({ x: grid.x, y: grid.y, w: 58, d: 39, r: 1 }, TILE_TOP, P), { tone: "faint" }));
  out.push(D(k.corners(TILE, 3.4).map(([x, y]) => at([x, y, TILE_TOP])), { size: 0.45 }));
  return out.join("");
}

function rack() {
  const out = [];
  const { x0, x1, post, y0, y1, z1 } = RACK;
  for (const px of [x0, x1 - post]) {
    const foot = { x: px - 8, y: y0 - 16, w: post + 16, d: 34, r: 2 };
    out.push(S(k.slabOf(foot, TILE_TOP, 3, P, 3, 0.5), { tone: "mid" }));
    out.push(D(k.corners(foot, 3).map(([x, y]) => at([x, y, TILE_TOP + 3])), { size: 0.5 }));
  }
  for (const px of [x0, x1 - post]) {
    out.push(S(box(px, y0, TILE_TOP + 3, px + post, y1, z1, { r: 0.6 }), { tone: "mid" }));
    const holes = [];
    for (let z = TILE_TOP + 8; z < z1 - 4; z += U / 3) holes.push(rr(px + post * 0.35, -(z + 0.8), px + post * 0.65, -(z - 0.8), 0.4));
    out.push(`<g transform="${frontPlane(y1)}">${flatLine(holes.join(""), "lo")}</g>`);
  }
  out.push(S(box(x0 - 3, y0 - 2, z1, x1 + 3, y1 + 2, z1 + 4, { r: 1 }), { tone: "mid" }));
  out.push(D([at([x0 + post / 2, y1 + 2, z1 + 4]), at([x1 - post / 2, y1 + 2, z1 + 4])], { size: 0.5 }));
  return out.join("");
}

const RU = (n) => TILE_TOP + 14 + n * U;

function rackUnits() {
  const out = [];
  const { x0, x1, post, y1 } = RACK;
  const bx0 = x0 + post;
  const bx1 = x1 - post;
  const front = y1 + 1.4;
  const unit = (n, h = 1, tone = "mid") => {
    const z0 = RU(n) + 0.3;
    const z1 = RU(n + h) - 0.3;
    out.push(S(box(bx0 + 1, y1 - 46, z0 + 0.4, bx1 - 1, y1, z1 - 0.4, { r: 0.4 }), { tone: "lo" }));
    out.push(S(box(x0 - 1, y1, z0, x1 + 1, front, z1, { r: 0.6, bevel: 0.3 }), { tone }));
    return { z0, z1 };
  };
  const blank = (n) => {
    const { z0, z1 } = unit(n);
    out.push(`<g transform="${frontPlane(front)}">${flatLine(slots(bx0 + 8, bx1 - 8, -(z1 - 3), -(z0 + 3), 22), "faint")}${ears(z0, z1)}</g>`);
  };
  const ears = (z0, z1) => [x0 + post / 2, x1 - post / 2].map((x) => `<circle class="t-screw" cx="${x}" cy="${f2(-(z0 + z1) / 2)}" r=".9"/>`).join("");
  blank(0);
  blank(1);
  {
    const { z0, z1 } = unit(2, 1, "hi");
    const face = [];
    face.push(flatLine(slots(bx0 + 4, bx0 + 30, -(z1 - 2.4), -(z0 + 2.4), 9), "lo"));
    face.push(`<circle class="t-bezel" cx="${bx0 + 36}" cy="${f2(-(z0 + z1) / 2)}" r="1.6"/><circle class="t-led" data-relay-led="cpu" cx="${bx0 + 36}" cy="${f2(-(z0 + z1) / 2)}" r="1.05"/>`);
    face.push(`<circle class="t-bezel" cx="${bx0 + 41}" cy="${f2(-(z0 + z1) / 2)}" r="1.6"/><circle class="t-led" data-relay-led="net" cx="${bx0 + 41}" cy="${f2(-(z0 + z1) / 2)}" r="1.05"/>`);
    face.push(`<circle class="t-bezel" cx="${bx0 + 46}" cy="${f2(-(z0 + z1) / 2)}" r="1.6"/><circle class="t-status" data-status="relay" cx="${bx0 + 46}" cy="${f2(-(z0 + z1) / 2)}" r="1.05"/>`);
    face.push(`<path class="t-recess" d="${rr(bx0 + 52, -(z1 - 2), bx1 - 4, -(z0 + 2), 0.6)}"/>`, flatLine(rr(bx0 + 52, -(z1 - 2), bx1 - 4, -(z0 + 2), 0.6), "lo"));
    face.push(flatLine(Array.from({ length: 6 }, (_, i) => rr(bx0 + 54 + i * 4.4, -(z1 - 3.4), bx0 + 57.4 + i * 4.4, -(z0 + 3.4), 0.3)).join(""), "faint"));
    face.push(ears(z0, z1));
    out.push(`<g transform="${frontPlane(front)}">${face.join("")}</g>`);
    out.push(S(box(bx1 - 22, front, z0 + 1.6, bx1 - 4, front + 3, z1 - 1.6, { r: 0.6 }), { tone: "hi", crease: "lo" }));
    const tlsFace = `<g transform="${frontPlane(front + 3)}"><path class="t-plate" d="${rr(bx1 - 20, -(z1 - 3), bx1 - 10, -(z0 + 3), 0.4)}"/>${flatLine(rr(bx1 - 20, -(z1 - 3), bx1 - 10, -(z0 + 3), 0.4), "lo")}<path class="t-key" d="M${bx1 - 7} ${f2(-(z0 + z1) / 2 + 1.4)}a1.4 1.4 0 1 1 0.01 0Z"/><circle class="t-led" data-relay-led="tls" cx="${bx1 - 7}" cy="${f2(-(z0 + z1) / 2 - 2.2)}" r=".6"/>${flatLine(`M${bx1 - 18.4} ${f2(-(z0 + z1) / 2 + 1)}H${bx1 - 12}M${bx1 - 18.4} ${f2(-(z0 + z1) / 2 - 1.2)}H${bx1 - 14}`, "lo", ' data-free=""')}</g>`;
    out.push(tlsFace);
  }
  {
    const { z0, z1 } = unit(3, 1, "mid");
    const face = [ears(z0, z1)];
    const badge = (bx, i) => {
      face.push(`<path class="t-recess" d="${rr(bx, -(z1 - 2.4), bx + 7, -(z0 + 2.4), 0.5)}"/>`, flatLine(rr(bx, -(z1 - 2.4), bx + 7, -(z0 + 2.4), 0.5), "mid"));
      face.push(`<path class="t-plate" d="${rr(bx + 9, -(z1 - 2.4), bx + 31, -(z0 + 2.4), 0.5)}"/>`, flatLine(rr(bx + 9, -(z1 - 2.4), bx + 31, -(z0 + 2.4), 0.5), "lo"));
      face.push(flatLine(`M${bx + 11} ${f2(-(z0 + z1) / 2 - 0.9)}H${bx + 11 + [17, 12][i]}M${bx + 11} ${f2(-(z0 + z1) / 2 + 1.3)}H${bx + 11 + [10, 9][i]}`, "lo", ' data-free=""'));
      face.push(`<circle class="t-led" data-badge="${i}" cx="${bx + 3.5}" cy="${f2(-(z1 - 3.6))}" r=".55"/>`);
    };
    badge(bx0 + 4, 0);
    badge(bx0 + 42, 1);
    out.push(`<g transform="${frontPlane(front)}">${face.join("")}</g>`);
  }
  blank(4);
  const top = RU(5) - 0.7;
  out.push(`<g transform="${topPlane(top)}">${flatLine(slots(bx0 + 10, bx1 - 30, RACK.y1 - 40, RACK.y1 - 8, 16), "faint")}<path class="t-plate" d="${rr(bx1 - 26, RACK.y1 - 30, bx1 - 6, RACK.y1 - 16, 0.6)}"/>${flatLine(rr(bx1 - 26, RACK.y1 - 30, bx1 - 6, RACK.y1 - 16, 0.6), "lo")}${flatLine(`M${bx1 - 23} ${RACK.y1 - 26}H${bx1 - 11}M${bx1 - 23} ${RACK.y1 - 23}H${bx1 - 14}M${bx1 - 23} ${RACK.y1 - 20}H${bx1 - 16}`, "lo", ' data-free=""')}</g>`);
  out.push(D([[bx0 + 4, RACK.y1 - 42], [bx1 - 4, RACK.y1 - 42], [bx0 + 4, RACK.y1 - 4], [bx1 - 4, RACK.y1 - 4]].map(([x, y]) => at([x, y, top])), { size: 0.45 }));
  return out.join("");
}

const RELAY_IN = [RACK.x0 - 1, RACK.y1 - 6, RU(2) + 5.4];
function relayInCable() {
  const c = COUPLER_RELAY;
  return cable([[c.x, c.y - 9, c.z], [c.x, c.y - 16, c.z], [c.x, TILE.y + TILE.d - 8, TILE_TOP + 1.6], [RACK.x0 - 6, RACK.y1 + 10, TILE_TOP + 1.6], [RACK.x0 - 6, RACK.y1 - 6, TILE_TOP + 5], [RACK.x0 - 6, RACK.y1 - 6, RELAY_IN[2]], [RACK.x0 - 1, RACK.y1 - 6, RELAY_IN[2]]], 1.2, { tone: "mid", bend: 5 });
}

const DEVICES = [
  { id: "tv", who: 0, x0: 318 + RDX, x1: 384 + RDX, y: -34 + RDY, z0: 32, z1: 70, foot: true },
  { id: "laptop", who: -1, x0: 336 + RDX, x1: 376 + RDX, y: 22 + RDY, z0: 12, z1: 38 },
  { id: "phone", who: 1, x0: 312 + RDX, x1: 326 + RDX, y: 68 + RDY, z0: 10, z1: 36 },
];
const OUT_PORT = [RACK.x1 + 1, RACK.y1 - 4, RU(2) + 5.4];
const EDGE = TILE.x + TILE.w;
const down = (y) => [[RACK.x1 + 8, RACK.y1 - 4, RU(2) + 5.4], [RACK.x1 + 8, RACK.y1 - 4, TILE_TOP + 3], [RACK.x1 + 8, y, TILE_TOP + 1.2], [EDGE - 3, y, TILE_TOP + 1.2], [EDGE + 2, y, TILE_TOP - 1], [EDGE + 5, y, 1.4]];
const OUT_ROUTES = {
  tv: [OUT_PORT, ...down(RACK.y1 - 6), sh([326, -35, 1.4]), sh([335, -35, 1.2])],
  laptop: [OUT_PORT, ...down(RACK.y1 + 10), sh([326, 32, 1.4]), sh([336, 32, 1.5])],
  phone: [OUT_PORT, ...down(RACK.y1 + 24), sh([300, 77, 1.4]), sh([306, 77, 1.8])],
};

function outCables() {
  return Object.values(OUT_ROUTES).map((r) => cable(r, 0.8, { tone: "mid", bend: 5 })).join("") +
    S(box(RACK.x1 + 1, RACK.y1 - 6.5, RU(2) + 3, RACK.x1 + 5, RACK.y1 - 1.5, RU(2) + 7.8, { r: 0.5 }), { tone: "hi" });
}

function device(d) {
  const out = [];
  if (d.id === "tv") {
    const foot = { x: (d.x0 + d.x1) / 2 - 16, y: d.y - 10, w: 32, d: 18, r: 3 };
    out.push(`<path class="iso-halo" d="${k.haloOf(foot, 0, 3, P)}" style="filter:blur(4px)"/>`);
    out.push(S(k.slabOf(foot, 0, 2.4, P, 4, 0.5), { tone: "mid" }));
    out.push(S(box((d.x0 + d.x1) / 2 - 3, d.y - 4, 2.4, (d.x0 + d.x1) / 2 + 3, d.y, d.z0 + 6, { r: 0.6 }), { tone: "mid" }));
    out.push(S(box(d.x0 + 8, d.y - 4, d.z0 + 4, d.x1 - 8, d.y, d.z1 - 6, { r: 1 }), { tone: "lo" }));
    out.push(S(box(d.x0, d.y, d.z0, d.x1, d.y + 4, d.z1, { r: 0.8, bevel: 0.4 }), { tone: "mid" }));
    out.push(screen(d.id, d.y + 4, d.x0 + 2, d.z0 + 2.5, d.x1 - 2, d.z1 - 2));
  } else if (d.id === "laptop") {
    const base = { x: d.x0, y: d.y - 4, w: d.x1 - d.x0, d: 28, r: 2 };
    out.push(`<path class="iso-halo" d="${k.haloOf(base, 0, 3, P)}" style="filter:blur(4px)"/>`);
    out.push(S(box(d.x0, d.y - 3, d.z0 - 1, d.x1, d.y, d.z1, { r: 1 }), { tone: "mid" }));
    out.push(screen(d.id, d.y, d.x0 + 2, d.z0 + 1.5, d.x1 - 2, d.z1 - 2));
    out.push(S(k.slabOf(base, 0, 3, P, 4, 0.5), { tone: "mid" }));
    const keys = [];
    for (let r = 0; r < 4; r++) for (let c = 0; c < 11; c++) keys.push(k.planOutline({ x: d.x0 + 3 + c * 3.2, y: d.y + 1 + r * 3.2, w: 2.5, d: 2.5, r: 0.4 }, 3, P, 2));
    out.push(L(keys.join(""), { tone: "faint" }));
    out.push(L(k.planOutline({ x: (d.x0 + d.x1) / 2 - 7, y: d.y + 15, w: 14, d: 8, r: 1 }, 3, P, 3), { tone: "lo" }));
  } else {
    const dock = { x: d.x0 - 6, y: d.y - 6, w: d.x1 - d.x0 + 12, d: 18, r: 3 };
    out.push(`<path class="iso-halo" d="${k.haloOf(dock, 0, 3, P)}" style="filter:blur(4px)"/>`);
    out.push(S(k.slabOf(dock, 0, 4, P, 4, 0.6), { tone: "mid" }));
    out.push(S(box(d.x0 - 2, d.y - 4, 4, d.x1 + 2, d.y, d.z0 + 14, { r: 1.2 }), { tone: "mid" }));
    out.push(S(box(d.x0, d.y, d.z0 - 2, d.x1, d.y + 2.2, d.z1, { r: 1.6, steps: 4 }), { tone: "mid", crease: "lo" }));
    out.push(screen(d.id, d.y + 2.2, d.x0 + 1, d.z0 + 1, d.x1 - 1, d.z1 - 2.4));
    out.push(S(box(d.x0 - 1, d.y, 4, d.x1 + 1, d.y + 4, d.z0, { r: 0.6 }), { tone: "lo" }));
  }
  return out.join("");
}

function pieceBoard() {
  const out = [];
  const plan = { x: PB.x0, y: PB.y0, w: PB.x1 - PB.x0, d: PB.y1 - PB.y0, r: 2 };
  for (const [x, y] of byDepth(k.corners(plan, 6))) {
    out.push(S(k.cylinder(x, y, 2.6, T.z1, 1.2, P, 14), { tone: "lo" }));
    out.push(S(k.cylinder(x, y, 1.6, T.z1 + 1.2, PB.z - 4 - T.z1 - 1.2, P, 12), { tone: "mid", crease: "none" }));
  }
  out.push(S(k.slabOf(plan, PB.z - 4, 4, P, 4, 0.8), { tone: "mid" }));
  const gx = PB.x0 + 12;
  const gy = PB.y0 + (PB.y1 - PB.y0 - ROWS * CELL) / 2;
  const wells = [];
  for (let r = 0; r < ROWS; r++) for (let c = 0; c < COLS; c++) wells.push(rr(gx + c * CELL + 0.8, gy + r * CELL + 0.8, gx + (c + 1) * CELL - 0.8, gy + (r + 1) * CELL - 0.8, 0.8));
  out.push(`<g transform="${topPlane(PB.z)}"><path class="t-well" d="${wells.join("")}"/>${flatLine(wells.join(""), "lo")}${flatLine(`M${gx - 2} ${gy + 3 * CELL}H${gx + COLS * CELL + 2}`, "mid")}${flatLine(rr(gx - 2, gy - 2, gx + COLS * CELL + 2, gy + ROWS * CELL + 2, 1), "faint")}</g>`);
  out.push(D(k.corners(plan, 3).map(([x, y]) => at([x, y, PB.z])), { size: 0.45 }));
  out.push(S(k.cylinder(PB.x0 + 5.5, PB.y1 - 9, 3, PB.z, 2, P, 18), { tone: "mid" }));
  out.push(S(k.cylinder(PB.x0 + 5.5, PB.y1 - 9, 1, PB.z + 2, 10, P, 12), { tone: "mid", crease: "none" }));
  out.push(S(k.cylinder(PB.x0 + 5.5, PB.y1 - 9, 2, PB.z + 12, 2.4, P, 14), { tone: "hi" }));
  out.push(labelPlate({ x: PB.x0 + 2, y: PB.y0 + 2, w: 7, d: 14, r: 0.8 }, PB.z, [0.5]));
  return { svg: out.join(""), gx, gy };
}

const CELLS = [];
function cellBlocks(gx, gy) {
  const out = [];
  const order = [];
  for (let r = ROWS - 1; r >= 0; r--) for (let c = 0; c < COLS; c++) order.push([r, c]);
  order.sort((a, b) => depth([gx + a[1] * CELL, gy + a[0] * CELL, 0]) - depth([gx + b[1] * CELL, gy + b[0] * CELL, 0]));
  for (const [r, c] of order) {
    const x = gx + c * CELL + 1.3;
    const y = gy + r * CELL + 1.3;
    const id = r * COLS + c;
    CELLS[id] = { center: [x + (CELL - 2.6) / 2, y + (CELL - 2.6) / 2, PB.z], region: r < 3 ? 0 : 1 };
    out.push(`<g class="t-cell" data-cell="${id}" style="opacity:0">${S(box(x, y, PB.z, x + CELL - 2.6, y + CELL - 2.6, PB.z + 2.6, { r: 0.6 }), { tone: "hi", crease: "lo" })}</g>`);
  }
  return out.join("");
}

const WIRE = (() => {
  const a = [MAST[0] + 1, MAST[1], MAST_TOP + 10];
  const b = [PB.x0 + 5.5, PB.y1 - 9, PB.z + 14];
  const pts = [];
  for (let i = 0; i <= 24; i++) {
    const t = i / 24;
    pts.push([a[0] + (b[0] - a[0]) * t, a[1] + (b[1] - a[1]) * t, a[2] + (b[2] - a[2]) * t - 14 * Math.sin(Math.PI * t)]);
  }
  return pts;
})();

const RNG = (() => {
  let s = 7;
  return () => ((s = (s * 16807) % 2147483647) / 2147483647);
})();
const shuffled = (n) => {
  const list = Array.from({ length: n }, (_, i) => i);
  for (let i = n - 1; i > 0; i--) {
    const j = Math.floor(RNG() * (i + 1));
    [list[i], list[j]] = [list[j], list[i]];
  }
  return list;
};
const REGION_ORDER = [0, 1].map((region) => shuffled(48).map((i) => region * 48 + i));

const capsuleY = (() => {
  const F = G.frameAlong([0, -5, 0], [0, 1, 0]);
  return S(G.solidOf([[0, 0.6, 1], [0.6, 1.4, 1], [1.6, 1.9, 1], [2.6, 2, 1], [7.4, 2, 1], [8.4, 1.9, 1], [9.4, 1.4, 1], [10, 0.6, 1]], F, P), { tone: "hi", lit: true });
})();
const capsule = (() => {
  const F = G.frameAlong([-5, 0, 0], [1, 0, 0]);
  return S(G.solidOf([[0, 0.6, 1], [0.6, 1.4, 1], [1.6, 1.9, 1], [2.6, 2, 1], [7.4, 2, 1], [8.4, 1.9, 1], [9.4, 1.4, 1], [10, 0.6, 1]], F, P), { tone: "hi", lit: true });
})();
const piece = S(box(-1.5, -1.5, -1.2, 1.5, 1.5, 1.2, { r: 0.4 }), { tone: "hi", lit: true });

const pb = pieceBoard();
const RES = {
  home: { name: "Media server" },
  storage: { name: "Storage" },
  transcoder: { name: "Transcoder" },
  torrent: { name: "Torrent engine" },
  tunnel: { name: "Tunnel" },
  relay: { name: "Relay Node" },
};

put("relay", tile());
put("tunnel", relayInCable());
put("relay", rack() + rackUnits());
put("relay", outCables());
groups.push(`<g data-layer="flow"></g>`);
for (const d of byDepth(DEVICES.map((d) => [d.x0, d.y, 0, d])).map((p) => p[3])) put("relay", device(d), ` data-device="${d.id}"`);
put("torrent", peers());
put("home", plinth());
put("home", powerStrip());
put("home", caseFeet());
put("home", shellBack());
put("home", motherboard());
put("transcoder", vrm() + cooler() + sessionBar());
put("home", ram());
put("home", psuShroud() + atxCable());
put("storage", driveCage());
put("home", shellFront() + powerLed());
put("home", powerCords() + plugs());
put("torrent", pb.svg);
groups.push(`<g class="it" data-res="torrent" data-layer="cells">${cellBlocks(pb.gx, pb.gy)}</g>`);
put("tunnel", ethernet());
put("tunnel", router());
put("home", lrtvCable());
groups.push(`<g data-layer="flow-home"></g>`);
put("home", livingRoomTv());
put("tunnel", trayBack());
groups.push(`<g data-layer="caps"></g>`);
put("tunnel", trayFront() + wanCable() + coupler(COUPLER_HOME, "home"));
put("tunnel", coupler(COUPLER_RELAY, "relay-end"));
groups.push(`<g data-layer="wire">${L(k.pathOf(WIRE.map(at)), { tone: "mid" })}<circle class="iso-wire-end" cx="${f2(at(WIRE[0])[0])}" cy="${f2(at(WIRE[0])[1])}" r="1.1"/><circle class="iso-wire-end" cx="${f2(at(WIRE[WIRE.length - 1])[0])}" cy="${f2(at(WIRE[WIRE.length - 1])[1])}" r="1.1"/></g>`);
groups.push(`<g data-layer="pieces"></g>`);

const svg = k.figureSvg({
  width: W,
  height: H,
  label: "An isometric bench with two machines. On the left, the home media server: a mid-tower PC with its side panel off, showing four drive sleds with fill gauges and activity lights, a CPU cooler whose fan spins with the transcoding load, and a piece board on top where torrent pieces from a peer mast land. A cable runs from its rear port to a router, and from there a cable tray carries lit packets across the floor to the Relay Node: a small rack on a raised-floor tile whose server feeds a TV, a laptop and a phone.",
  body: groups,
});

const sampleNow = sample("busy", 0);
const DATA = {
  P,
  tray: TRAY,
  couplers: { home: COUPLER_HOME, relay: COUPLER_RELAY },
  wire: WIRE,
  cells: CELLS,
  order: REGION_ORDER,
  boardHop: [PB.x0 + 5.5, PB.y1 - 9, PB.z + 14],
  out: Object.fromEntries([...Object.entries(OUT_ROUTES), ["lrtv", LRTV_ROUTE]].map(([id, r]) => [id, G.fillet(r, 5, 8)])),
  devices: DEVICES.map((d) => ({ id: d.id, who: d.who })),
  capsule,
  capsuleY,
  lanes: {
    video: [[COUPLER_HOME.x + 9, TRAY.y1 - 7.5, TRAY.z0 + 3.4], [TRAY.xb - 7.5, TRAY.y1 - 7.5, TRAY.z0 + 3.4], [TRAY.xb - 7.5, COUPLER_RELAY.y + 9, TRAY.z0 + 3.4]],
    req: [[TRAY.xa + 7.5, COUPLER_RELAY.y + 9, TRAY.z0 + 3.4], [TRAY.xa + 7.5, TRAY.y0 + 7.5, TRAY.z0 + 3.4], [COUPLER_HOME.x + 9, TRAY.y0 + 7.5, TRAY.z0 + 3.4]],
  },
  piece,
  res: Object.keys(RES),
};

const live = readFileSync(join(HERE, "live.js"), "utf8").replace("__DATA__", JSON.stringify(DATA));
const model = MODEL_SRC.replace(/^export /gm, "");
const kit = kitScript();

const CSS = `
*{box-sizing:border-box}
html{color-scheme:dark}
body{margin:0;background:var(--anatomy-page);color:var(--anatomy-text);font:14px/1.55 Geist,ui-sans-serif,system-ui,-apple-system,"Segoe UI",sans-serif;-webkit-font-smoothing:antialiased}
.app{max-width:1440px;margin:0 auto;padding:0 24px 48px}
.top{display:flex;align-items:center;gap:28px;height:56px;border-bottom:1px solid var(--anatomy-faint)}
.top h1{margin:0;font-size:15px;font-weight:600;color:var(--anatomy-strong);letter-spacing:.01em}
.tabs{display:flex;gap:4px}
.tab{font:inherit;font-size:13px;color:var(--anatomy-muted);background:none;border:0;padding:6px 10px;border-radius:6px;cursor:default}
.tab[aria-selected="true"]{color:var(--anatomy-strong);background:hsl(214 9% 11%);box-shadow:inset 0 0 0 1px hsl(214 9% 16%)}
.states{margin-left:auto;display:flex;gap:4px}
.states button{font:inherit;font-size:12px;color:var(--anatomy-muted);background:none;border:1px solid hsl(214 9% 14%);padding:4px 9px;border-radius:999px;cursor:pointer}
.states button[aria-pressed="true"]{color:var(--anatomy-strong);border-color:hsl(214 9% 30%)}
.sentence{margin:26px 0 18px;font-size:24px;font-weight:600;line-height:1.3;color:var(--anatomy-strong);letter-spacing:-.01em;max-width:60ch;min-height:1.3em}
.sentence .lamp{display:inline-block;width:9px;height:9px;border-radius:50%;margin-right:12px;vertical-align:.12em;background:var(--ok);box-shadow:0 0 10px var(--ok)}
.sentence[data-health="warn"] .lamp{background:var(--warn);box-shadow:0 0 10px var(--warn)}
.sentence[data-health="down"] .lamp{background:var(--fail);box-shadow:0 0 10px var(--fail)}
.layout{display:grid;grid-template-columns:minmax(0,1fr) 320px;gap:20px;align-items:start}
.stage.iso-plate{padding:34px 16px 40px;box-shadow:0 0 0 1px hsl(214 9% 14%);border-radius:10px;overflow:hidden;touch-action:pan-y}
.stage:focus{outline:none}
.stage:focus-visible{outline:1px solid var(--anatomy-muted);outline-offset:4px}
.stage .iso-plate-corners{inset:14px 18px}
.panel{border:1px solid hsl(214 9% 14%);border-radius:10px;background:var(--anatomy-card);padding:18px 18px 16px;min-height:300px}
.panel .eyebrow{font-family:var(--anatomy-mono);font-size:10.5px;letter-spacing:.1em;text-transform:uppercase;color:var(--anatomy-muted)}
.panel h2{margin:6px 0 10px;font-size:17px;font-weight:600;color:var(--anatomy-strong)}
.panel p{margin:0 0 10px}
.panel .rows{display:grid;gap:8px;margin:12px 0 4px;padding:0;list-style:none}
.panel .rows li{display:flex;gap:10px;align-items:baseline}
.panel .rows .dot{flex:none;width:7px;height:7px;border-radius:50%;background:var(--ok);transform:translateY(-1px)}
.panel .rows .dot[data-h="warn"]{background:var(--warn)}.panel .rows .dot[data-h="down"]{background:var(--fail)}.panel .rows .dot[data-h="idle"]{background:var(--anatomy-lo)}
.panel .rows strong{color:var(--anatomy-strong);font-weight:600}
.panel .num{font-variant-numeric:tabular-nums;color:var(--anatomy-strong)}
.panel details{margin-top:14px;border-top:1px solid hsl(214 9% 13%);padding-top:10px}
.panel summary{cursor:pointer;color:var(--anatomy-muted);font-size:12.5px}
.panel dl{display:grid;grid-template-columns:auto 1fr;gap:4px 12px;margin:10px 0 0;font-family:var(--anatomy-mono);font-size:11.5px}
.panel dt{color:var(--anatomy-muted)}.panel dd{margin:0;color:var(--anatomy-text);overflow-wrap:anywhere}
.panel .hint{margin-top:14px;color:var(--anatomy-muted);font-size:12px}
.sr{position:absolute;width:1px;height:1px;overflow:hidden;clip-path:inset(50%);white-space:nowrap}
@media (max-width:560px){.stage.iso-plate{padding:30px 0 52px}.stage .iso-svg{width:114%;margin-left:-7%}.stage .iso-plate-corners{inset:12px 14px}}
@media (max-width:900px){.layout{grid-template-columns:minmax(0,1fr)}.sentence{font-size:20px}.app{padding:0 16px 40px}.states{display:none}}

.t-hole{fill:var(--anatomy-shade-0)}
.t-recess{fill:var(--anatomy-shade-0)}
.t-well{fill:var(--anatomy-shade-1)}
.t-plate{fill:var(--anatomy-shade-3)}
.t-glass{fill:hsl(214 12% 4%)}
.t-blade{fill:var(--anatomy-shade-3);stroke:var(--anatomy-mid);stroke-width:.4px;vector-effect:non-scaling-stroke}
.t-blur{fill:var(--anatomy-hi)}
.t-hub{fill:var(--anatomy-top);stroke:var(--anatomy-mid);stroke-width:.5px;vector-effect:non-scaling-stroke}
.t-hub-dot{fill:var(--anatomy-mid)}
.t-guard{fill:none;stroke:var(--anatomy-mid);stroke-width:.5px;vector-effect:non-scaling-stroke}
.t-screw{fill:var(--anatomy-dot);opacity:.5}
.t-pads circle{fill:var(--anatomy-lo)}
.t-bezel{fill:var(--anatomy-shade-0);stroke:var(--anatomy-lo);stroke-width:.4px;vector-effect:non-scaling-stroke}
.t-lamp,.t-led,.t-power,.t-led-read,.t-led-write,.t-standby{fill:hsl(214 10% 22%)}
.t-led-read{fill:var(--glow)}.t-led-write{fill:#fff}
.t-led-read,.t-led-write{opacity:0}
.t-gauge-well{fill:var(--anatomy-shade-0)}
.t-gauge{fill:var(--anatomy-hi)}
.t-key{fill:var(--anatomy-mid)}
.t-status{fill:var(--ok);filter:drop-shadow(0 0 1.6px currentColor);color:var(--ok)}
.t-status[data-h="warn"]{color:var(--warn)}.t-status[data-h="down"]{color:var(--fail)}.t-status[data-h="idle"]{color:transparent}
.t-power-strip{fill:var(--anatomy-hi)}
.t-status[data-h="warn"]{fill:var(--warn)}
.t-status[data-h="down"]{fill:var(--fail)}
.t-status[data-h="idle"]{fill:hsl(214 10% 22%)}
.t-pic-sky{fill:hsl(214 30% 16%)}
.t-pic-hill{fill:hsl(214 18% 30%)}
.t-pic-hill2{fill:hsl(214 14% 46%)}
.t-pic-sun{fill:hsl(214 60% 92%)}
.t-pic-bar{fill:hsl(214 10% 30%)}
.t-pic-prog{fill:#fff}
.t-glare{fill:#fff;opacity:.04;pointer-events:none}
.t-cap .iso-top,.t-cap .iso-shade,.t-piece .iso-top,.t-piece .iso-shade{fill:hsl(214 60% 90%)!important;stroke:hsl(214 60% 90%)!important}
.t-cap .iso-edge,.t-piece .iso-edge{stroke:#fff}
.t-glow{fill:var(--glow);opacity:.28;filter:blur(2.5px)}
.t-seed .iso-top,.t-seed .iso-shade{fill:var(--anatomy-mid)!important;stroke:var(--anatomy-mid)!important}
.t-seed .iso-edge{stroke:var(--anatomy-hi)}
.t-dot{fill:#fff}

.it{transition:opacity .4s var(--anatomy-ease)}
.it[data-on] .iso-edge{stroke:var(--anatomy-lit)}
.it[data-on] .iso-line[data-tone="mid"]{stroke:var(--anatomy-hi)}
.it[data-on] .iso-line[data-tone="lo"]{stroke:var(--anatomy-mid)}
.it[data-on] .iso-top{fill:var(--anatomy-lit-top)}
.it[data-on] .iso-shade{fill:var(--anatomy-lit-shade)}
.it[data-on] .iso-solid[data-seamless]>.iso-shade{stroke:var(--anatomy-lit-shade)}
.it[data-on] .tb-body{fill:var(--anatomy-lit-shade)}
.it[data-on] .tb-shine{fill:var(--anatomy-lit-top)}
[data-focus] .it:not([data-on]) .iso-edge[data-tone="hi"]{stroke:var(--anatomy-mid)}
[data-focus] .it:not([data-on]) .iso-edge[data-tone="mid"],[data-focus] .it:not([data-on]) .iso-line[data-tone="mid"]{stroke:var(--anatomy-lo)}
[data-focus] .it:not([data-on]) .iso-line[data-tone="lo"]{stroke:var(--anatomy-faint)}
[data-focus] .it:not([data-on]) .t-guard,[data-focus] .it:not([data-on]) .t-blade{stroke:var(--anatomy-lo)}
`;

const PANEL = `<aside class="panel" data-panel aria-live="polite"></aside>`;
const html = `<!doctype html><html lang="en"><head><meta charset="utf-8"><meta name="viewport" content="width=device-width, initial-scale=1"><title>Nodes topology</title>
<link rel="preconnect" href="https://fonts.googleapis.com"><link rel="stylesheet" href="https://fonts.googleapis.com/css2?family=Geist:wght@400;500;600&family=Geist+Mono:wght@400;500&display=swap">
<style>${k.ISO_CSS}${THEME}${CSS}</style></head>
<body class="iso" data-theme="dark"><div class="app">
<header class="top"><h1>Nodes</h1><nav class="tabs" role="tablist"><button class="tab" role="tab" aria-selected="true">Topology</button><button class="tab" role="tab" aria-selected="false" tabindex="-1">Roster</button></nav>
<div class="states" role="group" aria-label="Prototype states">${["calm", "busy", "warn", "down"].map((s) => `<button data-state="${s}" aria-pressed="false">${s}</button>`).join("")}</div></header>
<p class="sentence" data-sentence><span class="lamp"></span><span data-sentence-text></span></p>
<div class="layout"><div class="stage iso-plate" tabindex="0" role="group" aria-label="Nodes topology. Arrow keys move between the Nodes and their resources, Escape clears.">
<div class="iso-plate-corners" aria-hidden="true"><span class="iso-plate-corner" data-corner="fig">Topology</span><span class="iso-plate-corner" data-corner="title">2 Nodes</span><span class="iso-plate-corner" data-corner="hint">Hover a part to see what it is doing</span><span class="iso-plate-corner" data-corner="readout" data-readout></span></div>
${svg}</div>${PANEL}</div></div>
<script type="module">${(kit + "\n" + model + "\n" + live).replace(/<\/(script)/gi, "<\\/$1")}</script></body></html>`;

writeFileSync(join(HERE, "index.html"), html);
const solids = (html.match(/class="iso-solid/g) || []).length;
const paths = (html.match(/<path/g) || []).length;
console.log("wrote index.html", { solids, paths, bytes: html.length, sample: sampleNow.sentence.slice(0, 20) });
