// PROTOTYPE. The canvas the D, E and F variants share: a dot-grid world that pans, zooms and fits, boxes you can move,
// and the geometry for edges.

import { I } from './shell.js';

const persistedViews = new Map();
const persistedMoves = new Map();
const hooks = new Map();

export const reduceMotion = () => matchMedia('(prefers-reduced-motion: reduce)').matches;
export const isNarrow = (app) => app.phone || innerWidth < 1000;

export function viewOf(id) {
  if (!persistedViews.has(id)) persistedViews.set(id, { tx: 0, ty: 0, k: 1, fitted: false });
  return persistedViews.get(id);
}
export function posOf(id, boxId, fallback) {
  return persistedMoves.get(`${id}|${boxId}`) || fallback;
}

export function register(id, h) { hooks.set(id, h); }

export function canvasHtml(id, world, ui = '') {
  const v = viewOf(id);
  return `<div class="cv" data-cv="${id}" tabindex="-1">
    <div class="cv-world" style="transform:translate(${v.tx}px,${v.ty}px) scale(${v.k})">${world}</div>
    ${ui}
    <div class="cv-zoom cv-ui" role="toolbar" aria-label="Canvas zoom">
      <button class="ib" data-cv-zoom="out" aria-label="Zoom out">${I('minus')}</button>
      <span class="num cv-k">${Math.round(v.k * 100)}%</span>
      <button class="ib" data-cv-zoom="in" aria-label="Zoom in">${I('plus')}</button>
      <span class="cv-zsep"></span>
      <button class="btn ghost sm" data-cv-zoom="fit">${I('scan')}Fit</button>
    </div>
  </div>`;
}

function apply(cv) {
  const v = viewOf(cv.dataset.cv);
  cv.querySelector('.cv-world').style.transform = `translate(${v.tx}px,${v.ty}px) scale(${v.k})`;
  const k = cv.querySelector('.cv-k');
  if (k) k.textContent = `${Math.round(v.k * 100)}%`;
  cv.style.backgroundPosition = `${v.tx}px ${v.ty}px`;
  cv.style.backgroundSize = `${22 * v.k}px ${22 * v.k}px`;
}

function zoomAt(cv, factor, cx, cy) {
  const v = viewOf(cv.dataset.cv);
  const k = Math.min(2, Math.max(0.3, v.k * factor));
  v.tx = cx - ((cx - v.tx) * k) / v.k;
  v.ty = cy - ((cy - v.ty) * k) / v.k;
  v.k = k;
  apply(cv);
}

export function bounds(cv) {
  let x0 = Infinity, y0 = Infinity, x1 = -Infinity, y1 = -Infinity;
  cv.querySelectorAll('.cv-world [data-bound]').forEach((el) => {
    x0 = Math.min(x0, el.offsetLeft); y0 = Math.min(y0, el.offsetTop);
    x1 = Math.max(x1, el.offsetLeft + el.offsetWidth); y1 = Math.max(y1, el.offsetTop + el.offsetHeight);
  });
  return { x0, y0, x1, y1 };
}

export function fit(cv) {
  const v = viewOf(cv.dataset.cv);
  const b = bounds(cv);
  if (!isFinite(b.x0)) return;
  const pad = { l: 40, r: cv.dataset.insp ? 520 : Number(cv.dataset.padr || 40), t: Number(cv.dataset.padt || 88), b: Number(cv.dataset.padb || 84) };
  const w = cv.clientWidth - pad.l - pad.r, h = cv.clientHeight - pad.t - pad.b;
  v.k = Math.min(1.15, w / (b.x1 - b.x0), h / (b.y1 - b.y0));
  v.tx = pad.l + (w - (b.x1 - b.x0) * v.k) / 2 - b.x0 * v.k;
  v.ty = pad.t + (h - (b.y1 - b.y0) * v.k) / 2 - b.y0 * v.k;
  v.fitted = true;
  apply(cv);
}

export function mounted(id) {
  return () => {
    const cv = document.querySelector(`.cv[data-cv="${id}"]`);
    if (!cv) return;
    hooks.get(id)?.draw(cv);
    if (!viewOf(id).fitted) fit(cv);
    else apply(cv);
  };
}

let wired = false;
let suppress = false;
export function suppressClick() { suppress = true; setTimeout(() => { suppress = false; }, 0); }
export function wire(root) {
  if (wired) return;
  wired = true;
  window.addEventListener('click', (e) => {
    if (suppress) { suppress = false; e.stopImmediatePropagation(); e.preventDefault(); return; }
    const z = e.target.closest?.('[data-cv-zoom]');
    if (!z) return;
    e.stopImmediatePropagation();
    const cv = z.closest('.cv');
    const r = cv.getBoundingClientRect();
    if (z.dataset.cvZoom === 'fit') return fit(cv);
    zoomAt(cv, z.dataset.cvZoom === 'in' ? 1.2 : 1 / 1.2, r.width / 2, r.height / 2);
  }, true);

  root.addEventListener('wheel', (e) => {
    const cv = e.target.closest('.cv');
    if (!cv || e.target.closest('.cv-ui, .insp')) return;
    e.preventDefault();
    const r = cv.getBoundingClientRect();
    const pinchOrWheel = e.ctrlKey || e.metaKey || (e.deltaMode === 1) || (Math.abs(e.deltaY) >= 40 && e.deltaX === 0 && Number.isInteger(e.deltaY));
    if (pinchOrWheel) return zoomAt(cv, Math.exp(-e.deltaY * (e.ctrlKey ? 0.01 : 0.0015)), e.clientX - r.left, e.clientY - r.top);
    const v = viewOf(cv.dataset.cv);
    v.tx -= e.deltaX; v.ty -= e.deltaY;
    apply(cv);
  }, { passive: false });

  root.addEventListener('pointerdown', (e) => {
    const cv = e.target.closest('.cv');
    if (!cv || e.button !== 0 || e.target.closest('.cv-ui, [data-chip], button:not([data-move]), a, input')) return;
    const mover = e.target.closest('[data-move]');
    const id = cv.dataset.cv;
    const v = viewOf(id);
    const sx = e.clientX, sy = e.clientY;
    const start = mover ? { x: mover.offsetLeft, y: mover.offsetTop } : { x: v.tx, y: v.ty };
    let live = false;
    const move = (ev) => {
      const dx = ev.clientX - sx, dy = ev.clientY - sy;
      if (!live && Math.hypot(dx, dy) < 4) return;
      if (!live) { live = true; cv.classList.add(mover ? 'moving' : 'panning'); }
      if (mover) {
        const p = { x: Math.round(start.x + dx / v.k), y: Math.round(start.y + dy / v.k) };
        persistedMoves.set(`${id}|${mover.dataset.move}`, p);
        mover.style.left = `${p.x}px`; mover.style.top = `${p.y}px`;
        hooks.get(id)?.draw(cv);
      } else { v.tx = start.x + dx; v.ty = start.y + dy; apply(cv); }
    };
    const up = () => {
      removeEventListener('pointermove', move); removeEventListener('pointerup', up);
      cv.classList.remove('moving', 'panning');
      if (live) { suppress = true; setTimeout(() => { suppress = false; }, 0); }
    };
    addEventListener('pointermove', move); addEventListener('pointerup', up);
  });
}

export function rect(el) { return { x: el.offsetLeft, y: el.offsetTop, w: el.offsetWidth, h: el.offsetHeight }; }
export const center = (r) => ({ x: r.x + r.w / 2, y: r.y + r.h / 2 });

export function port(r, side, t = 0.5) {
  if (side === 'l') return { x: r.x, y: r.y + r.h * t, side };
  if (side === 'r') return { x: r.x + r.w, y: r.y + r.h * t, side };
  if (side === 't') return { x: r.x + r.w * t, y: r.y, side };
  return { x: r.x + r.w * t, y: r.y + r.h, side };
}

export function facing(a, b) {
  const v = Math.min(a.y + a.h, b.y + b.h) - Math.max(a.y, b.y);
  const h = Math.min(a.x + a.w, b.x + b.w) - Math.max(a.x, b.x);
  const gapY = Math.max(b.y - (a.y + a.h), a.y - (b.y + b.h));
  if (v > 30) return b.x > a.x ? ['r', 'l'] : ['l', 'r'];
  if (h > 30 || gapY > 40) return b.y > a.y ? ['b', 't'] : ['t', 'b'];
  return center(b).x > center(a).x ? ['r', 'l'] : ['l', 'r'];
}

export function ports(a, b) {
  const [sa, sb] = facing(a, b);
  const horiz = sa === 'l' || sa === 'r';
  const lo = horiz ? Math.max(a.y, b.y) : Math.max(a.x, b.x);
  const hi = horiz ? Math.min(a.y + a.h, b.y + b.h) : Math.min(a.x + a.w, b.x + b.w);
  const along = (r, other) => {
    if (hi - lo > 30) return horiz ? (lo + hi) / 2 - r.y : (lo + hi) / 2 - r.x;
    const c = center(other), len = horiz ? r.h : r.w;
    return Math.max(0.2, Math.min(0.8, ((horiz ? c.y - r.y : c.x - r.x)) / len)) * len;
  };
  return [port(a, sa, along(a, b) / (horiz ? a.h : a.w)), port(b, sb, along(b, a) / (horiz ? b.h : b.w))];
}

const DIR = { l: [-1, 0], r: [1, 0], t: [0, -1], b: [0, 1] };
export function curve(p, q) {
  const d = Math.max(40, Math.min(160, Math.hypot(q.x - p.x, q.y - p.y) * 0.45));
  const [ax, ay] = DIR[p.side], [bx, by] = DIR[q.side];
  const c1 = { x: p.x + ax * d, y: p.y + ay * d }, c2 = { x: q.x + bx * d, y: q.y + by * d };
  const at = (t) => {
    const u = 1 - t;
    return { x: u * u * u * p.x + 3 * u * u * t * c1.x + 3 * u * t * t * c2.x + t * t * t * q.x, y: u * u * u * p.y + 3 * u * u * t * c1.y + 3 * u * t * t * c2.y + t * t * t * q.y };
  };
  return { d: `M${p.x},${p.y} C${c1.x},${c1.y} ${c2.x},${c2.y} ${q.x},${q.y}`, at };
}
