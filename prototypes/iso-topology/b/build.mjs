import { readFileSync, writeFileSync } from "node:fs";
import { dirname, join } from "node:path";
import { fileURLToPath } from "node:url";
import * as k from "../../anatomy/src/kit/iso-kit.mjs";
import * as G from "../../anatomy/src/kit/lathe.mjs";
import { tubePieces, tubeSvg } from "../../anatomy/src/kit/tube.mjs";
import { kitScript } from "../../anatomy/src/scripts/inline-kit.mjs";

const HERE = dirname(fileURLToPath(import.meta.url));
const W = 1100;
const H = 540;

const HOME = { x: -170, y: -120, w: 340, d: 240, r: 12 };
const RDY = 150;
const RELAY = { x: 250, y: -330 + RDY, w: 190, d: 170, r: 10 };
const PEERS = { x: -440, y: -60, w: 120, d: 110, r: 8 };
const SLAB = 14;
const FOOT = 10;
const GROUND = -SLAB - FOOT;
const PEER_TOP = -10;

const P = k.fitProjection(
  [
    ...k.boxCorners(HOME, GROUND, 0),
    ...k.boxCorners(RELAY, GROUND, 0),
    ...k.boxCorners(PEERS, GROUND, PEER_TOP),
    [-150, -60, 162],
    [82, -87, 104],
    [308, -288 + RDY, 104],
  ],
  W,
  H,
  { pad: 26, azimuth: 45 },
);

const at = (p) => k.iso(p, P);
const S = (paths, style) => k.solidSvg(paths, style);
const L = (d, style) => k.lineSvg(d, style);
const D = (points, style) => k.dotsSvg(points, style);
const box = (plan, z, h, tone = "mid", steps = 3, bevel = 0.5, extra = {}) => S(k.slabOf(plan, z, h, P, steps, bevel), { tone, ...extra });
const cyl = (x, y, r, z, h, tone = "mid", steps = 24, bevel = 0, extra = {}) => S(k.cylinder(x, y, r, z, h, P, steps, bevel), { tone, ...extra });
const byDepth = (points) => [...points].sort((a, b) => k.depthOf([a[0], a[1], 0], P) - k.depthOf([b[0], b[1], 0], P));
const faceY = (y) => k.sideMatrix([0, y], 0, P, "left");
const faceX = (x) => k.sideMatrix([x, 0], 0, P, "right");
const rr = (u0, z0, u1, z1, r, steps = 4) => k.pathOf(k.roundedPlan({ x: u0, y: z0, w: u1 - u0, d: z1 - z0, r }, steps).map(([u, z]) => [u, -z]), true);
const screwsOn = (points, z) => L(points.map(([x, y]) => k.ring(x, y, 1.3, z, P, 12)).join(""), { tone: "lo" }) + D(points.map(([x, y]) => at([x, y, z])), { size: 0.42 });
const halo = (plan, z, h, blur = 9) => `<path class="iso-halo" d="${k.haloOf(plan, z, h, P)}" style="filter:blur(${blur}px)"/>`;
const glow = ([x, y]) => `<g class="lamp-glow"><circle class="gh" cx="${x.toFixed(1)}" cy="${y.toFixed(1)}" r="4.6"/><circle class="gc" cx="${x.toFixed(1)}" cy="${y.toFixed(1)}" r="1.7"/></g>`;
const r1 = (v) => Math.round(v * 100) / 100;

const tubeItems = (route, r, opts = {}, tone = "mid") =>
  tubePieces(route, r, P, { maxLength: 18, ...opts }).map((piece) => ({ key: G.dot3(piece.mid, G.viewOf(P)), svg: tubeSvg(piece, { tone }) }));
const cable = (points, r = 0.85, opts = {}, tone = "mid") => {
  const route = G.fillet(points, Math.max(3, r * 3.2), 7, { P, tube: r });
  return { route, svg: tubePieces(route, r, P, { maxLength: 400, ...opts }).map((piece) => tubeSvg(piece, { tone })).join("") };
};

function tile(T, { label, plateAt, feet = 22, top = 0, slab = SLAB, part }) {
  const z0 = top - slab;
  const out = [halo(T, GROUND, slab + FOOT, 12)];
  const pts = [...k.corners(T, feet)];
  if (T.w > 200) pts.push([T.x + T.w / 2, T.y + feet], [T.x + T.w / 2, T.y + T.d - feet]);
  for (const [x, y] of byDepth(pts)) {
    out.push(cyl(x, y, 8, GROUND, 1.8, "lo", 28));
    out.push(cyl(x, y, 6, GROUND + 1.8, z0 - GROUND - 1.8, "mid", 28));
    out.push(L(k.sideArc(x, y, 6, GROUND + 5, P, 16), { tone: "faint" }));
  }
  const seam = L(k.sideSeam(T, z0 + slab * 0.38, P, 8), { tone: "faint" });
  out.push(S(k.slabOf(T, z0, slab, P, 8, 1.6), { tone: "mid", inner: seam }));
  out.push(L(k.planOutline(k.insetPlan(T, 5), top, P), { tone: "lo" }));
  out.push(L(k.planOutline(k.insetPlan(T, 6.6), top, P), { tone: "faint" }));
  const screws = k.corners(T, 11);
  out.push(screwsOn(screws, top));
  if (label) {
    const [px, py] = plateAt;
    const plan = { x: px, y: py, w: 54, d: 13, r: 1.4 };
    out.push(box(plan, top, 0.9, "lo", 3, 0, { crease: "none" }));
    out.push(L(k.planOutline(k.insetPlan(plan, 1.4), top + 0.9, P, 3), { tone: "faint" }));
    out.push(k.faceTextSvg(k.topMatrix([px + 4, py + 6.4], top + 0.9, P, "x"), label, { size: 4.2, tone: "mid" }));
    out.push(L([26, 18].map((w, i) => k.lineOnTop([px + 4, py + 8.6 + i * 2], [px + 4 + w, py + 8.6 + i * 2], top + 0.9, P)).join(""), { tone: "lo", free: true }));
    out.push(D([at([px + plan.w - 3, py + 3, top + 0.9]), at([px + plan.w - 3, py + plan.d - 3, top + 0.9])], { size: 0.42 }));
  }
  return `<g class="part" data-part="${part}">${out.join("")}</g>`;
}

function homeMat() {
  const inner = k.insetPlan(HOME, 6.6);
  const lines = [];
  for (let x = HOME.x + 40; x < HOME.x + HOME.w - 20; x += 40) lines.push(k.lineOnTop([x, inner.y], [x, inner.y + inner.d], 0, P));
  for (let y = HOME.y + 40; y < HOME.y + HOME.d - 20; y += 40) lines.push(k.lineOnTop([inner.x, y], [inner.x + inner.w, y], 0, P));
  return L(lines.join(""), { tone: "faint" });
}

function homeRuler() {
  const strip = { x: HOME.x + 70, y: HOME.y + HOME.d - 12, w: 180, d: 6, r: 1 };
  const out = [box(strip, 0, 0.6, "lo", 3, 0, { crease: "none" })];
  const t = k.topTicks(strip.x + 4, strip.x + strip.w - 4, 4, 5, strip.y + 0.6, 0.6, [1.8, 3.4], P);
  out.push(L(t.minor, { tone: "lo" }), L(t.major, { tone: "mid" }));
  out.push(D([at([strip.x + 2, strip.y + 3, 0.6]), at([strip.x + strip.w - 2, strip.y + 3, 0.6])], { size: 0.4 }));
  const vents = [];
  for (let i = 0; i < 10; i++) {
    const y0 = HOME.y + 14 + i * 6;
    vents.push(k.pathOf(k.roundedPlan({ x: y0, y: -10, w: 2.2, d: 6, r: 1 }, 3).map(([u, z]) => at([HOME.x + HOME.w, u, z])), true));
  }
  out.push(L(vents.join(""), { tone: "lo" }));
  return out.join("");
}

const ANT = [-150, -60];
const ANT_TOP = 146;
const BOOM = { from: -178, to: -142, z: ANT_TOP };
const RX = { x: -158, y: -68, w: 16, d: 14, r: 1.5, z: 100, h: 16 };
const YAGI_TIP = [BOOM.from, ANT[1], ANT_TOP + 1.2];

function antenna() {
  const [x, y] = ANT;
  const out = [halo({ x: x - 12, y: y - 12, w: 24, d: 24, r: 12 }, 0, 1, 5)];
  out.push(cyl(x, y, 11, 0, 2, "lo", 40, 0.5));
  out.push(D(G.dotsOf(G.frameAlong([x, y, 0], [0, 0, 1]), 2, 8.4, 6, P, { all: true }), { size: 0.45 }));
  out.push(cyl(x, y, 5, 2, 5, "mid", 28, 0.4));
  out.push(cyl(x, y, 2.6, 7, RX.z - 7, "mid", 20, 0, { crease: "none", inner: L([30, 60].map((z) => k.sideArc(x, y, 2.6, z, P, 12)).join(""), { tone: "faint" }) }));
  for (const z of [40, 70]) out.push(cyl(x, y, 3.6, z - 1.5, 3, "mid", 20, 0.3));
  out.push(box(RX, RX.z, RX.h, "mid", 3, 0.6));
  out.push(`<g transform="${faceY(RX.y + RX.d)}">${L(rr(RX.x + 2.4, RX.z + 3, RX.x + RX.w - 2.4, RX.z + RX.h - 3, 1.2), { tone: "faint" })}${L(rr(RX.x + 3.4, RX.z + 9.5, RX.x + 9.5, RX.z + 11.8, 1), { tone: "lo" })}</g>`);
  out.push(`<g data-led="rx">${D([at([RX.x + RX.w - 4, RX.y + RX.d, RX.z + 10.6])], { size: 0.85, tone: "hi" })}</g>`);
  out.push(D([at([RX.x + 2.2, RX.y + RX.d, RX.z + 1.8]), at([RX.x + RX.w - 2.2, RX.y + RX.d, RX.z + 1.8])], { size: 0.4 }));
  out.push(cyl(x, y, 2.2, RX.z + RX.h, ANT_TOP - RX.z - RX.h - 3, "mid", 18, 0, { crease: "none" }));
  out.push(cyl(x, y, 3.4, ANT_TOP - 3, 3, "mid", 18, 0.3));
  const boom = { x: BOOM.from, y: y - 1.2, w: BOOM.to - BOOM.from, d: 2.4, r: 0.6 };
  const els = [];
  [-174, -167, -160, -153, -146].forEach((ex, i) => {
    const half = [7, 8.5, 10, 11.5, 13][i];
    els.push({ x: ex, half });
  });
  const back = els.map((e) => box({ x: e.x - 0.6, y: y - e.half, w: 1.2, d: e.half - 1.2, r: 0.5 }, ANT_TOP + 0.2, 1.2, "mid", 2, 0, { crease: "none" }));
  const front = els.map((e) => box({ x: e.x - 0.6, y: y + 1.2, w: 1.2, d: e.half - 1.2, r: 0.5 }, ANT_TOP + 0.2, 1.2, "mid", 2, 0, { crease: "none" }));
  out.push(back.join(""));
  out.push(box(boom, ANT_TOP, 2.4, "mid", 2, 0));
  out.push(front.join(""));
  out.push(D(els.map((e) => at([e.x, y, ANT_TOP + 2.4])), { size: 0.4 }));
  out.push(cyl(x, y, 1, ANT_TOP + 2.4, 10, "mid", 12, 0, { crease: "none" }));
  out.push(`<g data-led="beacon">${S(k.cylinder(x, y, 2, ANT_TOP + 12.4, 3, P, 16), { tone: "hi", lit: true })}</g>`);
  return out.join("");
}

const LOOM = { x: -122, y: -106, w: 120, d: 20, r: 2 };
const BOARD_Y = -90;
const PANELS = [
  { x0: -112, cols: 5, rows: 7 },
  { x0: -60, cols: 5, rows: 7 },
];
const CELL = { w: 9, h: 9.4, gap: 0.8, z0: 13 };
const BEAM_Z = 86;
const HOPPER = { x: -108, y: -101, w: 14, d: 10, r: 1 };

function loom() {
  const out = [halo(LOOM, 0, 1, 6)];
  out.push(box(LOOM, 0, 6, "mid", 3, 0.6));
  out.push(screwsOn([[LOOM.x + 4, LOOM.y + LOOM.d - 4], [LOOM.x + LOOM.w - 4, LOOM.y + LOOM.d - 4]], 6));
  out.push(box({ x: LOOM.x + 6, y: -104, w: LOOM.w - 12, d: 14, r: 1 }, 6, BEAM_Z - 6, "mid", 3, 0.4));
  for (const px of [LOOM.x, LOOM.x + LOOM.w - 7]) {
    out.push(box({ x: px, y: -105, w: 7, d: 16, r: 1 }, 6, BEAM_Z - 6, "mid", 3, 0.5));
    out.push(D([at([px + 3.5, -89, 14]), at([px + 3.5, -89, BEAM_Z - 8])], { size: 0.42 }));
  }
  const cells = [];
  PANELS.forEach((panel, n) => {
    const x1 = panel.x0 + panel.cols * (CELL.w + CELL.gap) - CELL.gap;
    const z1 = CELL.z0 + panel.rows * (CELL.h + CELL.gap) - CELL.gap;
    const frame = [L(rr(panel.x0 - 2, CELL.z0 - 2, x1 + 2, z1 + 2, 1.4), { tone: "lo" })];
    const inner = [];
    for (let row = 0; row < panel.rows; row++) {
      for (let col = 0; col < panel.cols; col++) {
        const u = panel.x0 + col * (CELL.w + CELL.gap);
        const z = z1 - (row + 1) * CELL.h - row * CELL.gap;
        const i = row * panel.cols + col;
        inner.push(`<path class="cell-well" d="${rr(u, z, u + CELL.w, z + CELL.h, 0.8, 2)}"/>`);
        inner.push(`<rect class="cell" data-cell="${n}-${i}" x="${r1(u + 0.9)}" y="${r1(-(z + CELL.h - 0.9))}" width="${r1(CELL.w - 1.8)}" height="${r1(CELL.h - 1.8)}" rx="0.5"/>`);
      }
    }
    cells.push(`<g data-panel="${n}">${frame.join("")}${inner.join("")}<path class="iso-line" data-tone="lo" d="${Array.from({ length: panel.rows * panel.cols }, (_, i) => { const row = Math.floor(i / panel.cols); const col = i % panel.cols; const u = panel.x0 + col * (CELL.w + CELL.gap); const z = z1 - (row + 1) * CELL.h - row * CELL.gap; return rr(u, z, u + CELL.w, z + CELL.h, 0.8, 2); }).join("")}"/><g data-drops="${n}"></g></g>`);
  });
  out.push(`<g transform="${faceY(BOARD_Y)}">${cells.join("")}${L(rr(-64.8, 10, -63.2, 84, 0.8), { tone: "faint" })}</g>`);
  out.push(box({ x: LOOM.x - 2, y: -107, w: LOOM.w + 4, d: 18, r: 1.5 }, BEAM_Z, 6, "mid", 3, 0.6));
  out.push(L(k.lineOnTop([LOOM.x - 2, -93], [LOOM.x + LOOM.w + 2, -93], BEAM_Z + 6, P), { tone: "lo" }));
  out.push(screwsOn([[LOOM.x + 2, -98], [LOOM.x + LOOM.w - 2, -98]], BEAM_Z + 6));
  const prism = G.prismOf(G.frameOf([HOPPER.x, HOPPER.y + HOPPER.d / 2, BEAM_Z + 6], [1, 0, 0], [0, 1, 0], [0, 0, 1]), [[-5, 0], [5, 0], [7.5, 9], [-7.5, 9]], 0, HOPPER.w, P);
  out.push(S(prism, { tone: "mid" }));
  out.push(L(k.onTop([[HOPPER.x + 1.5, HOPPER.y - 1.2], [HOPPER.x + HOPPER.w - 1.5, HOPPER.y - 1.2], [HOPPER.x + HOPPER.w - 1.5, HOPPER.y + HOPPER.d + 1.2], [HOPPER.x + 1.5, HOPPER.y + HOPPER.d + 1.2]], BEAM_Z + 15, P, true), { tone: "lo" }));
  out.push(`<g data-part-move="shuttle">${box({ x: -6, y: -95, w: 12, d: 6, r: 1 }, BEAM_Z - 6, 5, "hi", 2, 0.4, { crease: "lo" })}${D([at([0, -89, BEAM_Z - 3.5])], { size: 0.55, tone: "hi" })}</g>`);
  PANELS.forEach((panel, n) => {
    const x1 = panel.x0 + panel.cols * (CELL.w + CELL.gap) - CELL.gap;
    const rail = { x: panel.x0, y: LOOM.y + LOOM.d - 5, w: x1 - panel.x0, d: 3, r: 0.8 };
    out.push(box(rail, 6, 0.6, "lo", 2, 0, { crease: "none" }));
    out.push(`<g data-progress="${n}">${box({ x: -1.4, y: rail.y - 0.6, w: 2.8, d: 4.2, r: 0.6 }, 6.6, 1.6, "hi", 2, 0, { crease: "lo" })}</g>`);
  });
  return out.join("");
}

const CHUTE_IN = [[RX.x + RX.w, RX.y + 4, RX.z + 6], [HOPPER.x, HOPPER.y + 3, BEAM_Z + 12]];
const CHUTE_OUT = [[HOPPER.x, HOPPER.y + 8, BEAM_Z + 12], [RX.x + RX.w, RX.y + 10, RX.z + 12]];

function chutes() {
  const items = [];
  for (const route of [CHUTE_IN, CHUTE_OUT]) {
    tubePieces(route, 2.1, P, { maxLength: 60 }).forEach((piece) => items.push(`<g class="glass">${tubeSvg(piece, { tone: "mid" })}</g>`));
  }
  return items.join("");
}

const CAB = { x: 40, y: -114, w: 84, d: 50, r: 3 };
const CAB_FRONT = CAB.y + CAB.d;
const CAB_TOP = 100;
const DRAWER_H = 21;
const DRAWER_Z = (j) => 7 + j * 23;
const DISK_OF = (j) => 3 - j;

function cabinet(disks) {
  const out = [halo(CAB, 0, 4, 7)];
  out.push(box(k.insetPlan(CAB, 2.5), 0, 4, "lo", 3, 0));
  const side = [];
  for (let i = 0; i < 7; i++) {
    const y0 = CAB.y + 8 + i * 4.6;
    side.push(k.pathOf(k.roundedPlan({ x: y0, y: 78, w: 2, d: 14, r: 1 }, 3).map(([u, z]) => at([CAB.x + CAB.w, u, z])), true));
  }
  const seam = L(k.sideSeam(CAB, 50, P, 3), { tone: "faint" });
  out.push(S(k.slabOf(CAB, 4, CAB_TOP - 4, P, 3, 0.8), { tone: "mid", inner: seam }));
  out.push(L(side.join(""), { tone: "lo" }));
  out.push(box(k.insetPlan(CAB, -1.2), CAB_TOP, 3, "mid", 3, 0.8));
  out.push(L(k.planOutline(k.insetPlan(CAB, 4), CAB_TOP + 3, P, 3), { tone: "faint" }));
  out.push(screwsOn(k.corners(CAB, 3), CAB_TOP + 3));
  out.push(`<g transform="${faceY(CAB_FRONT)}">${L(rr(CAB.x + 2, 5, CAB.x + CAB.w - 2, CAB_TOP - 2, 1.5), { tone: "faint" })}</g>`);
  for (let j = 0; j < 4; j++) {
    const disk = disks[DISK_OF(j)];
    const z0 = DRAWER_Z(j);
    const clipId = `dr-clip-${j}`;
    const near = [
      [CAB.x + 2, CAB_FRONT - 0.2, z0 - 1],
      [CAB.x + CAB.w - 2, CAB_FRONT - 0.2, z0 - 1],
      [CAB.x + CAB.w - 2, CAB_FRONT - 0.2, z0 + DRAWER_H + 1],
      [CAB.x + 2, CAB_FRONT - 0.2, z0 + DRAWER_H + 1],
    ];
    const far = near.map(([x, y, z]) => [x, y + 30, z]);
    const hull = G.hullOf([...near, ...far].map(at));
    const g = [];
    g.push(box({ x: CAB.x + 6, y: CAB_FRONT - 12, w: CAB.w - 12, d: 12, r: 0.6 }, z0 + 1.5, DRAWER_H - 4, "mid", 2, 0));
    const plan = { x: CAB.x + 3.5, y: CAB_FRONT, w: CAB.w - 7, d: 2.2, r: 0.8 };
    g.push(box(plan, z0, DRAWER_H, "hi", 3, 0.5, { crease: "lo" }));
    const fy = CAB_FRONT + 2.2;
    const share = disk.used / disk.size;
    const gx0 = CAB.x + 50, gx1 = CAB.x + CAB.w - 10;
    const face = [];
    face.push(L(rr(CAB.x + 8, z0 + 9, CAB.x + 26, z0 + 16, 0.8), { tone: "mid" }));
    face.push(`<path class="card" d="${rr(CAB.x + 9.2, z0 + 10.2, CAB.x + 24.8, z0 + 14.8, 0.4)}"/>`);
    face.push(L([0, 1].map((i) => `M${CAB.x + 10.6} ${-(z0 + 13.6 - i * 2)}H${CAB.x + 10.6 + [11, 7][i]}`).join(""), { tone: "lo", free: true }));
    face.push(L(rr(gx0, z0 + 4, gx1, z0 + 7.4, 1.2), { tone: "lo" }));
    face.push(`<path class="gauge" d="${rr(gx0 + 0.8, z0 + 4.8, gx0 + 0.8 + (gx1 - gx0 - 1.6) * share, z0 + 6.6, 0.8)}"/>`);
    face.push(L(rr(gx0 + (gx1 - gx0) * 0.9 - 0.3, z0 + 3.4, gx0 + (gx1 - gx0) * 0.9 + 0.3, z0 + 8, 0.3), { tone: "mid" }));
    face.push(`<text class="iso-face-text" data-tone="lo" x="${gx0}" y="${-(z0 + 10)}" font-size="3">${disk.label.toUpperCase()}</text>`);
    g.push(`<g transform="${faceY(fy)}">${face.join("")}</g>`);
    g.push(`<g class="led" data-drive-led="${DISK_OF(j)}">${D([at([CAB.x + CAB.w - 7, fy, z0 + 15])], { size: 0.9, tone: "hi" })}</g>`);
    g.push(D([at([CAB.x + 6, fy, z0 + 3]), at([CAB.x + CAB.w - 6, fy, z0 + 3])], { size: 0.38 }));
    const px = CAB.x + 36;
    g.push(box({ x: px - 7, y: fy, w: 2.4, d: 3, r: 0.6 }, z0 + 11, 2.4, "mid", 2, 0, { crease: "none" }));
    g.push(box({ x: px + 4.6, y: fy, w: 2.4, d: 3, r: 0.6 }, z0 + 11, 2.4, "mid", 2, 0, { crease: "none" }));
    g.push(box({ x: px - 8, y: fy + 3, w: 16, d: 2, r: 0.8 }, z0 + 10.6, 3.2, "hi", 2, 0.3, { crease: "lo" }));
    out.push(`<clipPath id="${clipId}"><path d="${k.pathOf(hull, true)}"/></clipPath><g clip-path="url(#${clipId})"><g data-drawer="${DISK_OF(j)}">${g.join("")}</g></g>`);
  }
  return out.join("");
}

const PRESS = { x: -8, y: 26, w: 150, d: 68, r: 4 };
const NIP_X = 76;
const PRESS_SCALE = 1.15;
const PRESS_AT = [-24, 22, 0];
const ROLL_R = 7;
const ROLL_LOW_Z = 22;
const ROLL_HIGH_Z = ROLL_LOW_Z + ROLL_R * 2 + 0.8;
const NIP_Z = ROLL_LOW_Z + ROLL_R + 0.4;
const ROLL_Y = [36, 84];
const LANES = [72, 62, 52, 42].map((y) => y + 1);
const IN_W = 8.4;
const OUT_W = 5;
const IN_X = [-2, NIP_X];
const OUT_X = [NIP_X, 128];
const RIB_COUNT = 24;
const RIB_FRAMES = 16;
const GEAR_Y = ROLL_Y[1] + 7;

function rollerFrames(z, dir, name) {
  const F = G.frameAlong([NIP_X, ROLL_Y[0], z], [0, 1, 0]);
  const frames = [];
  for (let f = 0; f < RIB_FRAMES; f++) {
    const phase = dir * (f / RIB_FRAMES) * ((Math.PI * 2) / RIB_COUNT);
    const ribs = G.ribsOf(F, 0.4, ROLL_Y[1] - ROLL_Y[0] - 0.4, ROLL_R, RIB_COUNT, P, { phase, fade: [0.08, 0.5] });
    frames.push(`<g data-roll="${name}" data-f="${f}"${f ? ' style="display:none"' : ""}>${k.fadedSvg(ribs, { tone: "lo" })}</g>`);
  }
  const body = G.disc(0, ROLL_Y[1] - ROLL_Y[0], ROLL_R, F, P);
  const seams = L([8, 40].map((s) => G.arcOf(F, s, ROLL_R, P)).join(""), { tone: "faint" });
  return S(body, { tone: "hi", crease: "lo", inner: seams }) + frames.join("");
}

function gearPath(cx, cz, r, teeth) {
  const pts = [];
  for (let i = 0; i < teeth * 4; i++) {
    const a = (i / (teeth * 4)) * Math.PI * 2;
    const rad = i % 4 < 2 ? r + 1.3 : r;
    pts.push([cx + rad * Math.cos(a), -(cz + rad * Math.sin(a))]);
  }
  return k.pathOf(pts, true);
}

const FRAME_TOP = 47;
function frame(y) {
  const out = [box({ x: NIP_X - 13, y, w: 26, d: 7, r: 1.4 }, 10, FRAME_TOP - 10, "mid", 3, 0.6)];
  out.push(screwsOn([[NIP_X - 9, y + 3.5], [NIP_X + 9, y + 3.5]], FRAME_TOP));
  const knob = [NIP_X, y + 3.5];
  out.push(cyl(knob[0], knob[1], 1.6, FRAME_TOP, 3, "mid", 14));
  out.push(cyl(knob[0], knob[1], 3.4, FRAME_TOP + 3, 3, "mid", 24, 0.3));
  out.push(L(k.knurl(knob[0], knob[1], 3.4, FRAME_TOP + 3, FRAME_TOP + 6, 22, P), { tone: "lo" }));
  return out.join("");
}

function press() {
  const out = [halo(PRESS, 0, 4, 7)];
  for (const [x, y] of byDepth(k.corners(PRESS, 6))) out.push(cyl(x, y, 3.4, 0, 3, "lo", 18));
  out.push(box(PRESS, 3, 7, "mid", 4, 0.8));
  out.push(L(k.planOutline(k.insetPlan(PRESS, 3), 10, P, 4), { tone: "faint" }));
  out.push(screwsOn(k.corners(PRESS, 6), 10));
  const fy = PRESS.y + PRESS.d;
  const panel = [];
  panel.push(L(rr(PRESS.x + 6, 4.6, PRESS.x + 34, 8.6, 1), { tone: "lo" }));
  panel.push(`<path class="window" d="${rr(PRESS.x + 7, 5.2, PRESS.x + 22, 8, 0.6)}"/>`);
  panel.push(`<text class="digits" data-fps x="${PRESS.x + 21}" y="${-5.8}" font-size="2.6" text-anchor="end">--</text>`);
  panel.push(`<text class="iso-face-text" data-tone="lo" x="${PRESS.x + 24}" y="${-5.7}" font-size="2.2">FPS</text>`);
  LANES.forEach((_, i) => panel.push(`<circle class="lamp-dot" data-lane-lamp="${i}" cx="${PRESS.x + 44 + i * 5}" cy="-6.6" r="1.1"/>`));
  panel.push(L(rr(PRESS.x + 40.6, 4.6, PRESS.x + 62.4, 8.6, 1), { tone: "lo" }));
  out.push(`<g transform="${faceY(fy)}">${panel.join("")}</g>`);
  out.push(frame(ROLL_Y[0] - 7));
  for (const x of [2, 40]) out.push(box({ x, y: 38, w: 6, d: 46, r: 1 }, 10, 15.6, "lo", 2, 0.3));
  out.push(box({ x: IN_X[0], y: 37, w: NIP_X - 8 - IN_X[0], d: 48, r: 1.2 }, 25.6, 2.6, "mid", 3, 0.5));
  for (let i = 0; i <= 4; i++) out.push(L(k.lineOnTop([IN_X[0], 37.6 + i * 10 + 0.4], [NIP_X - 8, 37.6 + i * 10 + 0.4], 28.2, P), { tone: "faint" }));
  out.push(box({ x: 116, y: 40, w: 6, d: 42, r: 1 }, 10, 15.6, "lo", 2, 0.3));
  out.push(box({ x: NIP_X + 8, y: 39, w: OUT_X[1] - NIP_X - 8, d: 44, r: 1.2 }, 25.6, 2.6, "mid", 3, 0.5));
  out.push(rollerFrames(ROLL_LOW_Z, 1, "low"));
  const lanes = [];
  LANES.forEach((cy, i) => {
    const inPlan = { x: IN_X[0] + 2, y: cy - IN_W / 2, w: IN_X[1] - IN_X[0] - 2, d: IN_W, r: 0.4 };
    const outPlan = { x: OUT_X[0], y: cy - OUT_W / 2, w: OUT_X[1] - OUT_X[0] - 2, d: OUT_W, r: 0.4 };
    const strips = [];
    for (const [plan, pitch, frameW, name, zTop] of [[inPlan, 10, 7.4, "in", 28.6], [outPlan, 6, 4.2, "out", 28.6]]) {
      const id = `strip-${name}-${i}`;
      const topRing = k.ringOf(plan, 2).map(([x, y]) => at([x, y, zTop + 0.5]));
      const marks = [];
      for (let x = plan.x - pitch * 2; x < plan.x + plan.w + pitch * 2; x += pitch) {
        marks.push(k.pathOf([[x + 0.9, plan.y + 1.1], [x + 0.9 + frameW, plan.y + 1.1], [x + 0.9 + frameW, plan.y + plan.d - 1.1], [x + 0.9, plan.y + plan.d - 1.1]].map(([u, v]) => at([u, v, zTop + 0.5])), true));
      }
      const holes = [];
      for (let x = plan.x - pitch * 2; x < plan.x + plan.w + pitch * 2; x += pitch / 2) {
        holes.push(at([x, plan.y + 0.5, zTop + 0.5]), at([x, plan.y + plan.d - 0.5, zTop + 0.5]));
      }
      strips.push(`<g data-strip="${name}" data-lane="${i}">${box(plan, zTop, 0.5, "hi", 2, 0, { crease: "none" })}<clipPath id="${id}"><path d="${k.pathOf(G.hullOf(topRing), true)}"/></clipPath><g clip-path="url(#${id})"><g data-marks="${name}-${i}" data-pitch="${pitch}">${L(marks.join(""), { tone: "lo" })}${D(holes, { size: 0.3, tone: "mid" })}</g></g></g>`);
    }
    lanes.push(`<g class="lane" data-lane="${i}">${strips.join("")}</g>`);
  });
  out.push(lanes.join(""));
  out.push(rollerFrames(ROLL_HIGH_Z, -1, "high"));
  out.push(frame(ROLL_Y[1]));
  const gears = [];
  gears.push(`<path class="gear-back" d="${gearPath(NIP_X, ROLL_LOW_Z, 6.4, 14)}"/>`);
  gears.push(`<g data-gear="low"><path class="gear" d="${gearPath(NIP_X, ROLL_LOW_Z, 6.4, 14)}"/>${L(rr(NIP_X - 4.4, ROLL_LOW_Z - 0.5, NIP_X + 4.4, ROLL_LOW_Z + 0.5, 0.5, 2), { tone: "mid" })}</g>`);
  gears.push(`<g data-gear="high"><path class="gear" d="${gearPath(NIP_X, ROLL_HIGH_Z, 6.4, 14)}"/>${L(rr(NIP_X - 4.4, ROLL_HIGH_Z - 0.5, NIP_X + 4.4, ROLL_HIGH_Z + 0.5, 0.5, 2), { tone: "mid" })}</g>`);
  gears.push(`<circle class="hub" cx="${NIP_X}" cy="${-ROLL_LOW_Z}" r="1.6"/><circle class="hub" cx="${NIP_X}" cy="${-ROLL_HIGH_Z}" r="1.6"/>`);
  out.push(`<g transform="${faceY(ROLL_Y[1] + 7)}">${gears.join("")}</g>`);
  const dial = [NIP_X + 30, 0];
  out.push(`<g transform="${faceY(fy)}"><circle class="window" cx="${PRESS.x + 104}" cy="-6.6" r="2.4"/><g data-load-needle><path class="iso-line" data-tone="hi" data-free="end" d="M${PRESS.x + 104} -6.6v-2"/></g><circle class="hub" cx="${PRESS.x + 104}" cy="-6.6" r="0.5"/></g>`);
  const OUTBOX = { x: 128, y: 44, w: 12, d: 34, r: 1.5 };
  out.push(box(OUTBOX, 10, 26, "mid", 3, 0.6));
  out.push(`<g transform="${faceX(OUTBOX.x + OUTBOX.w)}">${L(rr(-(OUTBOX.y + OUTBOX.d - 3), 13, -(OUTBOX.y + 3), 33, 1), { tone: "faint" })}</g>`);
  out.push(`<g class="led" data-out-led>${D([at([OUTBOX.x + OUTBOX.w, OUTBOX.y + 8, 30])], { size: 0.85, tone: "hi" })}</g>`);
  out.push(screwsOn([[OUTBOX.x + 3, OUTBOX.y + 3], [OUTBOX.x + 3, OUTBOX.y + OUTBOX.d - 3]], 36));
  return out.join("");
}

const PLEX = { x: -142, y: 18, w: 42, d: 30, r: 3 };
const TV = { x: -164, y: 92, w: 66, d: 4, r: 1 };
const TV_Z = [12, 52];

function plex() {
  const out = [halo(PLEX, 0, 2, 6)];
  for (const [x, y] of byDepth(k.corners(PLEX, 5))) out.push(cyl(x, y, 2.2, 0, 2, "lo", 14));
  out.push(S(k.slabOf(PLEX, 2, 16, P, 4, 0.8), { tone: "mid", inner: L(k.sideSeam(PLEX, 14, P, 4), { tone: "faint" }) }));
  const slots = [];
  for (let i = 0; i < 8; i++) slots.push(k.planOutline({ x: PLEX.x + 8 + i * 3.4, y: PLEX.y + 7, w: 1.4, d: 16, r: 0.7 }, 18, P, 3));
  out.push(L(slots.join(""), { tone: "lo" }));
  out.push(screwsOn([[PLEX.x + 3.5, PLEX.y + 3.5], [PLEX.x + PLEX.w - 3.5, PLEX.y + PLEX.d - 3.5]], 18));
  out.push(`<g transform="${faceY(PLEX.y + PLEX.d)}">${L(rr(PLEX.x + 4, 6, PLEX.x + 20, 10, 1), { tone: "lo" })}</g>`);
  out.push(`<g class="led" data-plex-led>${D([at([PLEX.x + PLEX.w - 7, PLEX.y + PLEX.d, 8])], { size: 0.85, tone: "hi" })}</g>`);
  return out.join("");
}

function tv() {
  const out = [];
  const stand = { x: TV.x + 20, y: TV.y - 6, w: 26, d: 14, r: 2 };
  out.push(halo(stand, 0, 1, 4));
  out.push(box(stand, 0, 2, "lo", 3, 0.4));
  out.push(box({ x: TV.x + 30, y: TV.y - 2, w: 6, d: 3, r: 0.8 }, 2, TV_Z[0] - 2, "mid", 2, 0.3));
  out.push(box(TV, TV_Z[0], TV_Z[1] - TV_Z[0], "mid", 3, 0.5));
  const fy = TV.y + TV.d;
  const sx0 = TV.x + 2.6, sx1 = TV.x + TV.w - 2.6, sz0 = TV_Z[0] + 3.4, sz1 = TV_Z[1] - 2.6;
  const id = "tv-home-clip";
  out.push(`<g transform="${faceY(fy)}"><clipPath id="${id}"><path d="${rr(sx0, sz0, sx1, sz1, 0.8)}"/></clipPath><path class="screen" d="${rr(sx0, sz0, sx1, sz1, 0.8)}"/><g clip-path="url(#${id})" class="picture" data-screen="home">${pictureSvg(sx0, sz0, sx1, sz1)}</g>${L(rr(sx0, sz0, sx1, sz1, 0.8), { tone: "lo" })}</g>`);
  out.push(`<g class="led" data-tv-led>${D([at([TV.x + TV.w - 6, fy, TV_Z[0] + 1.6])], { size: 0.6, tone: "hi" })}</g>`);
  return out.join("");
}

function pictureSvg(x0, z0, x1, z1) {
  const w = x1 - x0, h = z1 - z0;
  const hills = `M${x0} ${-(z0 + h * 0.3)}Q${x0 + w * 0.25} ${-(z0 + h * 0.62)} ${x0 + w * 0.5} ${-(z0 + h * 0.38)}T${x1} ${-(z0 + h * 0.46)}V${-z0}H${x0}Z`;
  return [
    `<rect class="pic-sky" x="${x0}" y="${-z1}" width="${w}" height="${h}"/>`,
    `<g data-pan><path class="pic-hill" d="${hills}"/><path class="pic-hill2" transform="translate(${w * 0.18} 0)" d="M${x0} ${-(z0 + h * 0.18)}Q${x0 + w * 0.3} ${-(z0 + h * 0.4)} ${x0 + w * 0.6} ${-(z0 + h * 0.2)}V${-z0}H${x0}Z"/></g>`,
    `<circle class="pic-sun" cx="${x0 + w * 0.72}" cy="${-(z0 + h * 0.72)}" r="${h * 0.12}"/>`,
    `<rect class="pic-bar-bg" x="${x0 + w * 0.08}" y="${-(z0 + h * 0.1)}" width="${w * 0.84}" height="${Math.max(0.5, h * 0.035)}"/>`,
    `<rect class="pic-bar" data-progress-bar x="${x0 + w * 0.08}" y="${-(z0 + h * 0.1)}" width="${w * 0.3}" height="${Math.max(0.5, h * 0.035)}" data-w="${w * 0.84}"/>`,
    `<rect class="pic-off" x="${x0}" y="${-z1}" width="${w}" height="${h}"/>`,
  ].join("");
}

const PORT_HOME = [HOME.x + HOME.w, 62, -7];
const PORT_RELAY = [330, RELAY.y + RELAY.d, -7];
const TUBE_R = 5;
const TUNNEL_ROUTE = G.fillet([PORT_HOME, [330, 62, -7], PORT_RELAY], 22, 14);
const TUNNEL_LEN = G.pathLength(TUNNEL_ROUTE);

function tunnel() {
  const items = [];
  const flangeT = 2.2;
  const bands = [];
  const supports = [];
  const bendFrom = 160 - 22 - 4, bendTo = 160 - 22 + 22 * Math.PI / 2 + 4;
  [46, 116, TUNNEL_LEN - 26].forEach((s) => supports.push(s));
  for (let s = 22; s < TUNNEL_LEN - 12; s += 23) if (!(s > bendFrom && s < bendTo) && supports.every((t) => Math.abs(t - s) > 12)) bands.push(s);
  const gaps = [[0, flangeT], [TUNNEL_LEN - flangeT, TUNNEL_LEN], ...bands.map((s) => [s - 1.2, s + 1.2]), ...supports.map((s) => [s - 2.2, s + 2.2])];
  tubePieces(TUNNEL_ROUTE, TUBE_R, P, { maxLength: 16, gaps }).forEach((piece) => items.push({ key: G.dot3(piece.mid, G.viewOf(P)), svg: `<g class="glass tunnel-glass">${tubeSvg(piece, { tone: "mid" })}</g>` }));
  const ring = (s, half, r, tone) => {
    const c = G.pointAlong(TUNNEL_ROUTE, s);
    const a = G.tangentAlong(TUNNEL_ROUTE, s);
    const F = G.frameAlong(c, a);
    return { key: G.dot3(c, G.viewOf(P)) + 0.5, svg: S(G.disc(-half, half, r, F, P), { tone, crease: "lo" }) };
  };
  bands.forEach((s) => items.push(ring(s, 1.2, TUBE_R + 0.9, "mid")));
  supports.forEach((s) => {
    const c = G.pointAlong(TUNNEL_ROUTE, s);
    const post = cyl(c[0], c[1], 2, GROUND + 2, c[2] - TUBE_R - 1.4 - GROUND - 2, "mid", 16, 0, { crease: "none" });
    const foot = cyl(c[0], c[1], 5, GROUND, 2, "lo", 24, 0.3);
    items.push({ key: G.dot3([c[0], c[1], GROUND], G.viewOf(P)) - 0.2, svg: `<path class="iso-halo" d="${k.haloOf({ x: c[0] - 6, y: c[1] - 6, w: 12, d: 12, r: 6 }, GROUND, 1, P)}" style="filter:blur(4px)"/>` + foot + post });
    items.push(ring(s, 2.2, TUBE_R + 1.6, "mid"));
  });
  for (const [s, dir] of [[0, 1], [TUNNEL_LEN, -1]]) {
    const c = G.pointAlong(TUNNEL_ROUTE, s);
    const a = G.tangentAlong(TUNNEL_ROUTE, s);
    const F = G.frameAlong(c, G.mul3(a, dir));
    const body = S(G.disc(0, flangeT, TUBE_R + 2.6, F, P), { tone: "mid", crease: "lo" });
    const bolts = D(G.dotsOf(F, flangeT, TUBE_R + 1.6, 8, P, { fade: [0.04, 0.42] }), { size: 0.42 });
    items.push({ key: G.dot3(c, G.viewOf(P)) + 1, svg: body + bolts });
  }
  return items;
}

const LAMP_HOME = [158, 102];
const LAMP_RELAY = [348, RELAY.y + RELAY.d - 12];

function lamp([x, y], name) {
  const out = [cyl(x, y, 4, 0, 2, "lo", 24, 0.3), cyl(x, y, 2.6, 2, 3, "mid", 20, 0.3)];
  out.push(`<g class="lamp" data-lamp="${name}">${S(k.cylinder(x, y, 2.2, 5, 3.2, P, 20), { tone: "hi" })}${glow(at([x, y, 8.2]))}</g>`);
  out.push(D([at([x - 3.2, y + 0.6, 2]), at([x + 3.2, y - 0.6, 2])], { size: 0.38 }));
  return out.join("");
}

const TOWER = { x: 280, y: -312 + RDY, w: 58, d: 44, r: 3 };
const TOWER_FRONT = TOWER.y + TOWER.d;
const TOWER_TOP = 96;
const SOCKETS = [{ name: "slipstream", x: 294, z: 44 }, { name: "plex", x: 324, z: 44 }];

function raisedFloor() {
  const out = [];
  const inner = k.insetPlan(RELAY, 6.6);
  const lines = [];
  for (let x = RELAY.x + 30; x < RELAY.x + RELAY.w - 20; x += 24) lines.push(k.lineOnTop([x, inner.y], [x, inner.y + inner.d], 0, P));
  for (let y = RELAY.y + 26; y < RELAY.y + RELAY.d - 18; y += 24) lines.push(k.lineOnTop([inner.x, y], [inner.x + inner.w, y], 0, P));
  out.push(L(lines.join(""), { tone: "faint" }));
  const dots = [];
  for (const [cx, cy] of [[RELAY.x + 42, RELAY.y + 136], [RELAY.x + 90, RELAY.y + 136], [RELAY.x + 138, RELAY.y + 136], [RELAY.x + 114, RELAY.y + 112], [RELAY.x + 42, RELAY.y + 112]]) {
    dots.push(...k.dotGrid(cx - 8, cy - 8, 5, 5, 4, 0, P));
  }
  out.push(D(dots, { size: 0.36, tone: "lo" }));
  return out.join("");
}

function tower(fronts) {
  const out = [halo(TOWER, 0, 4, 8)];
  out.push(box(k.insetPlan(TOWER, 2), 0, 5, "lo", 3, 0));
  const seam = L([24, 48, 72].map((z) => k.sideSeam(TOWER, z, P, 3)).join(""), { tone: "faint" });
  out.push(S(k.slabOf(TOWER, 5, TOWER_TOP - 5, P, 3, 0.8), { tone: "mid", inner: seam }));
  out.push(box(k.insetPlan(TOWER, -1), TOWER_TOP, 3, "mid", 3, 0.8));
  out.push(screwsOn(k.corners(TOWER, 3), TOWER_TOP + 3));
  const vents = [];
  for (let i = 0; i < 6; i++) vents.push(k.pathOf(k.roundedPlan({ x: TOWER.y + 7 + i * 4.6, y: 78, w: 2, d: 12, r: 1 }, 3).map(([u, z]) => at([TOWER.x + TOWER.w, u, z])), true));
  out.push(L(vents.join(""), { tone: "lo" }));
  const face = [];
  face.push(L(rr(TOWER.x + 3, 8, TOWER.x + TOWER.w - 3, TOWER_TOP - 3, 1.5), { tone: "faint" }));
  face.push(L(rr(TOWER.x + 6, 62, TOWER.x + 30, 88, 1.2), { tone: "lo" }));
  for (let i = 0; i < 8; i++) face.push(`<rect class="cpu-cell" data-cpu="${i}" x="${TOWER.x + 8 + i * 2.7}" y="-86" width="1.9" height="22"/>`);
  face.push(`<text class="iso-face-text" data-tone="lo" x="${TOWER.x + 34}" y="-84" font-size="2.6">CPU</text>`);
  face.push(L(`M${TOWER.x + 34} -80H${TOWER.x + 52}M${TOWER.x + 34} -77H${TOWER.x + 46}`, { tone: "lo", free: true }));
  SOCKETS.forEach((sock) => {
    face.push(L(rr(sock.x - 9, sock.z - 9, sock.x + 9, sock.z + 9, 1.6), { tone: "lo" }));
    face.push(`<circle class="socket" cx="${sock.x}" cy="${-sock.z}" r="6"/>`);
    face.push(L(`M${sock.x - 2.4} ${-sock.z - 1.6}v3.2M${sock.x + 2.4} ${-sock.z - 1.6}v3.2`, { tone: "mid" }));
  });
  face.push(L(rr(TOWER.x + 6, 14, TOWER.x + TOWER.w - 6, 28, 1), { tone: "lo" }));
  for (let i = 0; i < 4; i++) face.push(`<circle class="port-led" data-port-led="${i}" cx="${TOWER.x + 12 + i * 11}" cy="-24" r="0.9"/>`, L(rr(TOWER.x + 8 + i * 11, 15.6, TOWER.x + 16 + i * 11, 21, 0.6), { tone: "mid" }));
  out.push(`<g transform="${faceY(TOWER_FRONT)}">${face.join("")}</g>`);
  out.push(D(SOCKETS.flatMap((s) => [at([s.x - 7, TOWER_FRONT, s.z - 7]), at([s.x + 7, TOWER_FRONT, s.z + 7])]), { size: 0.4 }));
  const lampsAt = [TOWER.x + 12, TOWER.x + 22, TOWER.x + 32];
  ["ok", "warn", "fail"].forEach((name, i) => {
    const x = lampsAt[i], y = TOWER_FRONT - 6;
    out.push(cyl(x, y, 3, TOWER_TOP + 3, 1.4, "lo", 20));
    out.push(`<g class="tower-lamp" data-tower-lamp="${name}">${S(k.cylinder(x, y, 2.2, TOWER_TOP + 4.4, 2.6, P, 20), { tone: "mid" })}${glow(at([x, y, TOWER_TOP + 7]))}</g>`);
  });
  const lx = TOWER.x + TOWER.w;
  const lock = { x: lx, y: TOWER.y + 18, w: 4, d: 14, r: 1.4 };
  out.push(box({ x: lx, y: lock.y + 3, w: 1.4, d: 8, r: 0.6 }, 52, 4, "mid", 2, 0));
  const shackle = [[lx + 2.2, lock.y + 3.4, 66], [lx + 2.2, lock.y + 3.4, 71], [lx + 2.2, lock.y + lock.d - 3.4, 71], [lx + 2.2, lock.y + lock.d - 3.4, 66]];
  const sh = G.fillet(shackle, 3.2, 8);
  out.push(`<g data-tls>${tubePieces(sh, 1.1, P, { maxLength: 400 }).map((p) => tubeSvg(p, { tone: "hi" })).join("")}</g>`);
  out.push(`<g data-tls>${box(lock, 52, 14, "hi", 3, 0.5, { crease: "lo" })}</g>`);
  out.push(`<g transform="${faceX(lx + 4)}"><circle class="keyhole" cx="${-(lock.y + 7)}" cy="-59.5" r="1.3"/><path class="keyhole" d="M${-(lock.y + 7) - 0.5} -59.2h1l0.4 3h-1.8z"/></g>`);
  out.push(D([at([lx + 1.4, lock.y + 4, 56]), at([lx + 1.4, lock.y + 10, 56])], { size: 0.36 }));
  return out.join("");
}

function plugs() {
  const out = [];
  const cables = [];
  SOCKETS.forEach((sock, i) => {
    const F = G.frameAlong([sock.x, TOWER_FRONT, sock.z], [0, 1, 0]);
    out.push(S(G.disc(0, 5, 4.4, F, P), { tone: "mid", crease: "lo" }));
    out.push(L(G.ribsOf(F, 0.4, 4.6, 4.4, 14, P), { tone: "lo" }));
    out.push(S(G.disc(5, 8, 2, F, P), { tone: "mid", crease: "lo" }));
  });
  return out.join("");
}

const SPLIT = { x: 360, y: -258 + RDY, w: 14, d: 10, r: 1.4 };
const DEVICES = {
  tv: { x: 404, y: -318 + RDY, w: 4, d: 52, r: 1, z: [12, 44], stream: 0 },
  phone: { x: 410, y: -246 + RDY, w: 2.4, d: 11, r: 1, z: [6, 26], stream: 1 },
  laptop: { x: 398, y: -214 + RDY, w: 2, d: 30, r: 0.8, z: [3, 24], stream: -1 },
};
const PLUG_ROUTES = [
  [[SOCKETS[0].x, TOWER_FRONT + 8, SOCKETS[0].z], [SOCKETS[0].x, TOWER_FRONT + 14, SOCKETS[0].z], [SOCKETS[0].x, TOWER_FRONT + 14, 0.9], [SOCKETS[0].x, TOWER_FRONT + 30, 0.9]],
  [[SOCKETS[1].x, TOWER_FRONT + 8, SOCKETS[1].z], [SOCKETS[1].x, TOWER_FRONT + 14, SOCKETS[1].z], [SOCKETS[1].x, TOWER_FRONT + 14, 0.9], [SOCKETS[1].x, SPLIT.y + 5, 0.9], [SPLIT.x, SPLIT.y + 5, 0.9]],
];
const FEED_ROUTES = {
  tv: [[SPLIT.x + SPLIT.w, SPLIT.y + 3, 0.9], [392, SPLIT.y + 3, 0.9], [392, -292 + RDY, 0.9], [398, -292 + RDY, 0.9]],
  phone: [[SPLIT.x + SPLIT.w, SPLIT.y + 7, 0.9], [392, SPLIT.y + 7, 0.9], [392, -240 + RDY, 0.9], [404, -240 + RDY, 0.9]],
  laptop: [[SPLIT.x + 7, SPLIT.y + SPLIT.d, 0.9], [SPLIT.x + 7, -205 + RDY, 0.9], [396, -205 + RDY, 0.9]],
};

function deviceScreen(xFace, y0, y1, z0, z1, name) {
  const id = `scr-${name}`;
  const u0 = -y1, u1 = -y0;
  return `<g transform="${faceX(xFace)}"><clipPath id="${id}"><path d="${rr(u0, z0, u1, z1, 0.8)}"/></clipPath><path class="screen" d="${rr(u0, z0, u1, z1, 0.8)}"/><g clip-path="url(#${id})" class="picture" data-screen="${name}">${pictureSvg(u0, z0, u1, z1)}</g>${L(rr(u0, z0, u1, z1, 0.8), { tone: "lo" })}</g>`;
}

function devices() {
  const out = {};
  {
    const d = DEVICES.tv;
    const g = [];
    const stand = { x: d.x - 8, y: d.y + 14, w: 18, d: 24, r: 2 };
    g.push(halo(stand, 0, 1, 4), box(stand, 0, 2, "lo", 3, 0.4));
    g.push(box({ x: d.x - 1, y: d.y + 23, w: 3, d: 6, r: 0.8 }, 2, d.z[0] - 2, "mid", 2, 0.3));
    g.push(box(d, d.z[0], d.z[1] - d.z[0], "mid", 3, 0.5));
    g.push(deviceScreen(d.x + d.w, d.y + 2.4, d.y + d.d - 2.4, d.z[0] + 3, d.z[1] - 2.4, "tv"));
    out.tv = g.join("");
  }
  {
    const d = DEVICES.phone;
    const g = [];
    const dock = { x: d.x - 6, y: d.y - 3, w: 12, d: 17, r: 2 };
    g.push(halo(dock, 0, 1, 3), box(dock, 0, 3.4, "lo", 3, 0.4));
    g.push(box({ x: d.x - 1.6, y: d.y - 1, w: 1.6, d: d.d + 2, r: 0.5 }, 3.4, 8, "mid", 2, 0));
    g.push(box(d, d.z[0], d.z[1] - d.z[0], "mid", 3, 0.4));
    g.push(deviceScreen(d.x + d.w, d.y + 1.2, d.y + d.d - 1.2, d.z[0] + 2.6, d.z[1] - 2.6, "phone"));
    g.push(D([at([d.x + d.w, d.y + d.d / 2, d.z[1] - 1.3])], { size: 0.4 }));
    out.phone = g.join("");
  }
  {
    const d = DEVICES.laptop;
    const g = [];
    const base = { x: d.x, y: d.y, w: 24, d: d.d, r: 1.4 };
    g.push(halo(base, 0, 1, 3), box(base, 0, 2.4, "mid", 3, 0.4));
    g.push(L(k.planOutline({ x: d.x + 5, y: d.y + 3, w: 12, d: d.d - 6, r: 0.8 }, 2.4, P, 3), { tone: "lo" }));
    g.push(D(k.dotGrid(d.x + 6.4, d.y + 4.6, 4, 9, 2.6, 2.4, P), { size: 0.3, tone: "lo" }));
    g.push(L(k.planOutline({ x: d.x + 18.5, y: d.y + 10, w: 3.6, d: 10, r: 0.6 }, 2.4, P, 3), { tone: "faint" }));
    g.push(box(d, d.z[0], d.z[1] - d.z[0], "mid", 2, 0.3));
    g.push(deviceScreen(d.x + d.w, d.y + 1.6, d.y + d.d - 1.6, d.z[0] + 2.2, d.z[1] - 1.6, "laptop"));
    out.laptop = g.join("");
  }
  return out;
}

const HOUSES = [
  [-424, -44], [-392, -50], [-428, -6], [-370, -16], [-402, 20],
];
const BEACONS = [[-350, 26], [-336, -36]];

function house([x, y], i) {
  const out = [];
  const plan = { x, y, w: 18, d: 14, r: 0.8 };
  const z = PEER_TOP;
  out.push(box(plan, z, 11, "mid", 2, 0.4));
  const F = G.frameOf([x - 1, y + 7, z + 11], [1, 0, 0], [0, 1, 0], [0, 0, 1]);
  out.push(S(G.prismOf(F, [[-8.6, 0], [8.6, 0], [0, 7]], 0, 20, P), { tone: "mid" }));
  out.push(box({ x: x + 12, y: y + 3, w: 3, d: 3, r: 0.4 }, z + 13, 6, "mid", 2, 0));
  out.push(`<g transform="${faceY(y + 14)}"><path class="house-win" data-house-win="${i}" d="${rr(x + 3, z + 4, x + 8, z + 8.4, 0.4)}"/>${L(rr(x + 3, z + 4, x + 8, z + 8.4, 0.4), { tone: "lo" })}${L(rr(x + 11, z, x + 15, z + 7, 0.4), { tone: "lo" })}</g>`);
  out.push(`<g transform="${faceX(x + 18)}"><path class="house-win" data-house-win="${i}" d="${rr(-(y + 10), z + 4, -(y + 4), z + 8.4, 0.4)}"/>${L(rr(-(y + 10), z + 4, -(y + 4), z + 8.4, 0.4), { tone: "lo" })}</g>`);
  return out.join("");
}

function peers() {
  const out = [];
  const items = HOUSES.map((h, i) => ({ key: h[0] + h[1], svg: house(h, i) }));
  BEACONS.forEach((BEACON, bi) => items.push({
    key: BEACON[0] + BEACON[1],
    svg: [
      cyl(BEACON[0], BEACON[1], 4.4, PEER_TOP, 2, "lo", 24),
      cyl(BEACON[0], BEACON[1], 1.4, PEER_TOP + 2, 34 - bi * 8, "mid", 12, 0, { crease: "none" }),
      box({ x: BEACON[0] - 5, y: BEACON[1] - 4, w: 10, d: 8, r: 1 }, PEER_TOP + 36 - bi * 8, 6, "mid", 2, 0.4),
      `<g class="led" data-tracker="${bi}">${D([at([BEACON[0], BEACON[1] + 4, PEER_TOP + 39 - bi * 8])], { size: 0.8, tone: "hi" })}</g>`,
      D([at([BEACON[0] - 3, BEACON[1] + 4, PEER_TOP + 39 - bi * 8]), at([BEACON[0] + 3, BEACON[1] + 4, PEER_TOP + 39 - bi * 8])], { size: 0.35 }),
    ].join(""),
  }));
  items.sort((a, b) => a.key - b.key);
  out.push(items.map((i) => i.svg).join(""));
  return out.join("");
}

function arcRoute(from, to, lift) {
  const pts = [];
  for (let i = 0; i <= 24; i++) {
    const t = i / 24;
    const p = G.lerp3(from, to, t);
    p[2] += lift * 4 * t * (1 - t);
    pts.push(p);
  }
  return pts;
}

const MAST_DOWN = [[ANT[0], ANT[1], ANT_TOP - 1], [ANT[0], ANT[1], RX.z + RX.h + 1]];
const DOWN_ROUTES = HOUSES.map(([x, y]) => [
  ...arcRoute([x + 13.5, y + 4.5, PEER_TOP + 20], YAGI_TIP, 46),
  [ANT[0], ANT[1], ANT_TOP + 1.2],
  ...MAST_DOWN,
  [RX.x + RX.w / 2, RX.y + 4, RX.z + 6],
  ...CHUTE_IN,
]);
const SEED_ROUTES = DOWN_ROUTES.map((_, i) => {
  const [x, y] = HOUSES[i];
  return [...CHUTE_OUT, [RX.x + RX.w / 2, RX.y + 10, RX.z + 12], ...[...MAST_DOWN].reverse(), [ANT[0], ANT[1], ANT_TOP + 1.2], ...arcRoute(YAGI_TIP, [x + 13.5, y + 4.5, PEER_TOP + 20], 40)];
});

const floorCable = cable([[PLEX.x + PLEX.w, PLEX.y + 12, 6], [PLEX.x + PLEX.w + 10, PLEX.y + 12, 0.9], [-80, 8, 0.9], [CAB.x + 10, 8, 0.9], [CAB.x + 10, CAB_FRONT + 1.5, 0.9], [CAB.x + 10, CAB_FRONT - 0.6, 2]]);
const tvCable = cable([[PLEX.x + 10, PLEX.y + PLEX.d, 6], [PLEX.x + 10, PLEX.y + PLEX.d + 8, 0.9], [PLEX.x + 10, TV.y - 6, 0.9]]);

function homeLayer() {
  const order = [
    ["torrent", antenna()],
    ["torrent", loom()],
    ["torrent", chutes()],
    ["home", plex()],
    ["home", tv()],
    ["storage", cabinet(MODEL_DISKS)],
    ["transcoder", `<g transform="translate(${at(PRESS_AT).map((v) => v.toFixed(2)).join(" ")}) scale(${PRESS_SCALE}) translate(${at([PRESS.x, PRESS.y, 0]).map((v) => (-v).toFixed(2)).join(" ")})">${press()}</g>`],
  ];
  return [
    tile(HOME, { label: "HOME", plateAt: [HOME.x + 12, HOME.y + HOME.d - 22], part: "home" }),
    `<g class="part" data-part="home">${homeMat()}${homeRuler()}${floorCable.svg}${tvCable.svg}${lamp(LAMP_HOME, "home")}</g>`,
    ...order.map(([part, svg]) => `<g class="part" data-part="${part}">${svg}</g>`),
  ].join("");
}

function relayLayer() {
  const dev = devices();
  const feeds = Object.fromEntries(Object.entries(FEED_ROUTES).map(([n, pts]) => [n, cable(pts, 0.7)]));
  const plugCables = PLUG_ROUTES.map((pts) => cable(pts, 0.9));
  const pieces = [
    tile(RELAY, { label: "ASHBURN", plateAt: [RELAY.x + 12, RELAY.y + RELAY.d - 22], part: "relay" }),
    `<g class="part" data-part="relay">${raisedFloor()}`,
    `<path class="iso-halo" d="${k.haloOf(SPLIT, 0, 1, P)}" style="filter:blur(3px)"/>`,
    Object.values(feeds).map((c) => c.svg).join(""),
    `<g data-flow-dots="relay"></g>`,
    tower(),
    plugs(),
    plugCables.map((c) => c.svg).join(""),
    box(SPLIT, 0, 6, "mid", 2, 0.4),
    D([at([SPLIT.x + 3, SPLIT.y + SPLIT.d, 4]), at([SPLIT.x + 7, SPLIT.y + SPLIT.d, 4]), at([SPLIT.x + 11, SPLIT.y + SPLIT.d, 4])], { size: 0.5, tone: "mid" }),
    `<g class="grommet">${cyl(SOCKETS[0].x, TOWER_FRONT + 33, 3.4, 0, 0.8, "lo", 20)}</g>`,
    lamp(LAMP_RELAY, "relay"),
    `<g data-device="tv">${dev.tv}</g>`,
    `<g data-device="phone">${dev.phone}</g>`,
    `<g data-device="laptop">${dev.laptop}</g>`,
    `</g>`,
  ];
  return { svg: pieces.join(""), routes: { plex: plugCables[1].route, ...Object.fromEntries(Object.entries(feeds).map(([n, c]) => [n, c.route])) } };
}

const MODEL_SRC = readFileSync(join(HERE, "..", "model.mjs"), "utf8").replace(/^export /gm, "");
const { NODES } = await import(join(HERE, "..", "model.mjs"));
const MODEL_DISKS = NODES.home.disks;

const peerTile = tile(PEERS, { top: PEER_TOP, slab: 8, feet: 14, part: "torrent" });
const tunnelItems = tunnel().sort((a, b) => a.key - b.key);
const relay = relayLayer();

const svg = k.figureSvg({
  width: W,
  height: H,
  label: "Two Nodes drawn as isometric site tiles. On the home tile: a four-drawer steel cabinet for the drives, a film press for the transcoder, a sorting board fed by an antenna for the torrent engine, and a Plex box feeding the living-room TV. A glass pneumatic tunnel carries lit capsules to the Ashburn relay tile, whose switchboard tower feeds the remote viewers' screens.",
  body: [
    `<g class="part" data-part="torrent">${peerTile}<g class="part" data-part="torrent">${peers()}</g></g>`,
    homeLayer(),
    `<g data-layer="pieces"></g>`,
    relay.svg,
    `<g class="part" data-part="tunnel"><g data-capsules></g>${tunnelItems.map((i) => i.svg).join("")}</g>`,
  ],
});

const DATA = {
  P,
  tunnel: TUNNEL_ROUTE,
  down: DOWN_ROUTES,
  seed: SEED_ROUTES,
  panels: PANELS.map((p) => ({ x0: p.x0, cols: p.cols, rows: p.rows })),
  cell: CELL,
  board: { y: BOARD_Y, beam: BEAM_Z, hopperX: HOPPER.x + HOPPER.w / 2 },
  press: { nipX: NIP_X, rollR: ROLL_R, low: ROLL_LOW_Z, high: ROLL_HIGH_Z, ribFrames: RIB_FRAMES, ribCount: RIB_COUNT },
  drawerOpen: 7,
  relayRoutes: relay.routes,
  loadDial: [PRESS.x + 104, -6.6],
};

const CSS = readFileSync(join(HERE, "..", "theme.css"), "utf8") + readFileSync(join(HERE, "page.css"), "utf8");
const BODY = readFileSync(join(HERE, "body.html"), "utf8").replace("__SVG__", svg);
const LIVE = readFileSync(join(HERE, "live.js"), "utf8");

const html = `<!doctype html><html lang="en"><head><meta charset="utf-8"><meta name="viewport" content="width=device-width, initial-scale=1"><title>Nodes topology</title>
<link rel="preconnect" href="https://fonts.googleapis.com"><link href="https://fonts.googleapis.com/css2?family=Geist:wght@400;500;600&family=Geist+Mono:wght@400;500&display=swap" rel="stylesheet">
<style>${k.ISO_CSS}${CSS}</style></head><body class="iso">${BODY}<script type="module">${(kitScript() + "\n" + MODEL_SRC + "\nconst DATA = " + JSON.stringify(DATA) + ";\n" + LIVE).replace(/<\/(script)/gi, "<\\/$1")}</script></body></html>`;
writeFileSync(join(HERE, "index.html"), html);
console.log("wrote index.html", Math.round(html.length / 1024), "KB");
