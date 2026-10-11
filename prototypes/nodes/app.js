// PROTOTYPE. Mounts a variant inside the Lit console admin shell, owns URL state, sheets, popovers, palette and toast.
// Everything renders from state: a click mutates state, then refresh() redraws the page and any open sheet.

import { scenario } from './data.js';
import { startLive } from '../ui-direction/live.js';
import { esc, I, AMBIENT, overlays, setAmbient } from './shell.js';
import { apply, key, controllerOf } from './model.js';
import { pairing, startPairing, pairActions, pairSim, tickCountdown, handover, startHandover, hoActions, removeHtml, forceHtml, joinHtml, joinActions, joinSim, workerStatusHtml } from './flows.js';
import * as A from './a.js';
import * as B from './b.js';
import * as C from './c.js';
import * as D from './d.js';
import * as E from './flow.js';
import * as F from './f.js';

export const VARIANTS = { A, B, C, D, E, F };

let app;

function sharedModal(modal) {
  const n = app.S.nodes.find((m) => m.id === modal.node);
  if (modal.kind === 'remove') return removeHtml(app, n);
  if (modal.kind === 'force') return forceHtml(app, n);
  return null;
}

const SELF_DISABLED = ['handover', 'b-cell', 'c-pick', 'a-tick', 'pair-role'];

const SHARED = {
  pair: () => { startPairing(); app.go({ view: 'pair' }); },
  handover: (a, el) => {
    if (el.getAttribute('aria-disabled') === 'true') return app.toast(el.title);
    startHandover(el.dataset.node || null);
    app.go({ view: 'handover' });
  },
  'close-flow': () => app.go({ view: 'nodes' }),
  'test-conn': (a, el) => {
    const n = app.S.nodes.find((m) => m.id === el.dataset.node);
    if (n.state === 'offline') return app.toast(`${n.name} did not answer. Last heartbeat ${n.lastSeen.slice(11, 16)}`);
    app.toast(`${n.name} answered in ${n.link.latency} ms over the Node link${n.warn ? `. ${n.warn} still` : ''}`);
  },
  remove: (a, el) => { app.modal = { kind: 'remove', node: el.dataset.node, size: 'sm' }; },
  'remove-yes': (a, el) => {
    const n = app.S.nodes.find((m) => m.id === el.dataset.node);
    const d = new Map(Object.keys(n.roles).map((r) => [key(n.id, r), false]));
    apply(app.S.nodes, d);
    app.S.nodes.splice(app.S.nodes.indexOf(n), 1);
    app.modal = null;
    app.toast(`${n.name} removed. Its key is deleted`);
    app.go({ node: '' });
  },
  force: (a, el) => { app.modal = { kind: 'force', node: el.dataset.node, size: 'sm' }; },
  'force-yes': (a, el) => {
    const n = app.S.nodes.find((m) => m.id === el.dataset.node);
    app.modal = null;
    app.toast(`Marked ${n.name}'s 2 downloads and ${n.seeding?.count ?? 0} torrents lost. Searching again for the 2 downloads`);
  },
  'add-role': (a, el) => {
    const n = app.S.nodes.find((m) => m.id === el.dataset.node);
    apply(app.S.nodes, new Map([[key(n.id, el.dataset.role), true]]));
    app.toast(`${n.name} holds ${el.dataset.role}. The search requeues there at the end of the grace period`);
  },
  copy: (a, el) => { try { navigator.clipboard.writeText(el.dataset.copy); } catch { /* clipboard unavailable */ } app.toast('Copied'); },
  close: () => { app.close(); return 'keep'; },
  palette: () => { openPalette(); return 'keep'; },
  more: () => app.toast('The other sections are not in this prototype'),
};

function actionFor(name) {
  return SHARED[name] || pairActions[name] || hoActions[name] || joinActions[name] || app.V.actions?.[name];
}

function params() {
  const p = { variant: app.cfg.variant, device: app.cfg.device, state: app.cfg.state, view: app.cfg.view };
  if (app.cfg.node) p.node = app.cfg.node;
  return new URLSearchParams(p).toString();
}

function scrollEl() { return app.phone ? app.root : document.scrollingElement; }

function refresh() {
  const { root } = app;
  const pg = root.querySelector('.pg');
  const fid = document.activeElement?.dataset?.fid;
  const keep = ['.nd-sheet .fl-body', '.insp'].map((s) => [s, root.querySelector(s)?.scrollTop || 0]);
  let out;
  if (app.cfg.view === 'worker-join') out = { html: joinHtml() };
  else if (app.cfg.view === 'worker-status') out = { html: workerStatusHtml(app) };
  else out = app.V.render(app);
  pg.innerHTML = out.html;

  const wrap = root.querySelector('.nd-sheet-wrap'), sh = wrap.querySelector('.nd-sheet');
  const modal = app.modal ? (app.V.modalHtml(app, app.modal) || sharedModal(app.modal)) : null;
  const sheet = modal ?? out.sheet ?? null;
  app.sheetClose = modal ? null : out.sheetClose || null;
  if (sheet) {
    const fresh = wrap.hidden;
    wrap.hidden = false;
    sh.className = `nd-sheet sz-${(app.modal && app.modal.size) || out.sheetSize || 'md'}${fresh ? ' enter' : ''}`;
    sh.innerHTML = sheet;
  } else wrap.hidden = true;

  window.lucide && window.lucide.createIcons();
  setAmbient(root, out.ambient || null);
  keep.forEach(([s, t]) => { const el = root.querySelector(s); if (el) el.scrollTop = t; });
  if (fid) { const el = root.querySelector(`[data-fid="${CSS.escape(fid)}"]`); if (el) el.focus({ preventScroll: true }); }
  out.mounted?.();
}

function go(patch) {
  const pageChange = ('node' in patch && patch.node !== app.cfg.node) || ('view' in patch && patch.view !== app.cfg.view);
  Object.assign(app.cfg, patch);
  history.replaceState(null, '', `?${params()}`);
  window.dispatchEvent(new CustomEvent('proto-url', { detail: { view: app.cfg.view } }));
  app.modal = null;
  closePop();
  const inPlace = app.cfg.variant === 'C' || (app.cfg.variant !== 'A' && !app.phone);
  if (pageChange && !inPlace) scrollEl().scrollTop = 0;
  refresh();
}

function close() {
  if (!app.root.querySelector('.nd-pop').hidden) return closePop();
  if (!app.root.querySelector('.pal').hidden) return closePalette();
  if (app.modal) { app.modal = null; return refresh(); }
  if (app.sheetClose) return go(app.sheetClose);
}

let toastT;
function toast(msg) {
  const t = app.root.querySelector('.toast');
  t.innerHTML = `${I('info')}<span>${esc(msg)}.</span>`;
  window.lucide && window.lucide.createIcons();
  t.classList.add('on');
  clearTimeout(toastT);
  toastT = setTimeout(() => t.classList.remove('on'), 3200);
}

function popover(anchor, html, cls = '') {
  const pop = app.root.querySelector('.nd-pop');
  pop.className = `pop nd-pop ${cls}`;
  pop.innerHTML = html;
  pop.hidden = false;
  window.lucide && window.lucide.createIcons();
  const r = anchor.getBoundingClientRect(), rr = app.root.getBoundingClientRect();
  const sTop = app.phone ? app.root.scrollTop : 0;
  const w = pop.offsetWidth, h = pop.offsetHeight;
  let top = r.bottom - rr.top + sTop + 6;
  const viewH = app.phone ? app.root.clientHeight : window.innerHeight;
  if (r.bottom + h + 12 > viewH && r.top - h - 6 > 0) top = r.top - rr.top + sTop - h - 6;
  const left = Math.min(Math.max(8, r.left - rr.left), rr.width - w - 8);
  pop.style.top = `${top}px`;
  pop.style.left = `${left}px`;
  const first = pop.querySelector('button:not([aria-disabled="true"])');
  if (first) first.focus({ preventScroll: true });
}
function closePop() { const p = app.root.querySelector('.nd-pop'); if (p) p.hidden = true; }

// ── ⌘K palette, scoped to what the Nodes screens can do
function paletteItems(q) {
  const ctl = controllerOf(app.S.nodes);
  const items = [
    { label: 'Pair a Node', icon: 'plus', meta: 'Issue a Join Code', act: 'pair' },
    ...(app.S.nodes.length > 1 ? [{ label: 'Hand over the Controller', icon: 'crown', meta: `from ${ctl.name}`, act: 'handover' }] : []),
    ...app.S.nodes.map((n) => ({ label: `Open ${n.name}`, icon: 'server', meta: n.kind, go: { node: n.id, view: 'nodes' } })),
    { label: 'Add a Work', icon: 'film', meta: 'Library', toast: 'Adding a Work is not in this prototype' },
  ];
  q = q.trim().toLowerCase();
  return items.filter((x) => !q || x.label.toLowerCase().includes(q));
}
let sel = 0, palItems = [];
function renderPalette(q) {
  palItems = paletteItems(q);
  const list = app.root.querySelector('.pal-list');
  list.innerHTML = palItems.length ? `<li role="presentation" class="pal-g">Nodes</li>${palItems.map((x, i) => `<li role="option" id="pal-o${i}" class="pal-o" data-i="${i}" aria-selected="${i === 0}"><span class="pal-ic">${I(x.icon)}</span><span class="pal-l">${esc(x.label)}</span><span class="t3 pal-meta">${esc(x.meta)}</span></li>`).join('')}` : `<li class="pal-empty" role="presentation">Nothing matches “${esc(q)}”.</li>`;
  window.lucide && window.lucide.createIcons();
  sel = 0;
}
function setSel(i) {
  const opts = [...app.root.querySelectorAll('.pal-o')];
  if (!opts.length) return;
  sel = (i + opts.length) % opts.length;
  opts.forEach((o, k) => o.setAttribute('aria-selected', String(k === sel)));
}
function runPal(i) {
  const x = palItems[i];
  if (!x) return;
  closePalette();
  if (x.act) return SHARED[x.act](app, document.createElement('button'));
  if (x.go) return go(x.go);
  toast(x.toast);
}
function openPalette() {
  const pal = app.root.querySelector('.pal');
  pal.hidden = false;
  const inp = pal.querySelector('.pal-in');
  inp.value = '';
  renderPalette('');
  requestAnimationFrame(() => inp.focus({ preventScroll: true }));
}
function closePalette() { app.root.querySelector('.pal').hidden = true; }

function wire() {
  const { root } = app;
  root.addEventListener('click', (e) => {
    const q = (s) => e.target.closest(s);
    const popEl = root.querySelector('.nd-pop');
    if (!popEl.hidden && !q('.nd-pop')) closePop();
    let el;
    if ((el = q('.pal-o'))) return runPal(Number(el.dataset.i));
    if ((el = q('[data-act]'))) {
      const fn = actionFor(el.dataset.act);
      if (!fn) return toast(`${el.dataset.act} is not in this prototype`);
      if (el.getAttribute('aria-disabled') === 'true' && !SELF_DISABLED.includes(el.dataset.act)) return el.title ? toast(el.title) : undefined;
      if (fn(app, el) === 'keep') return;
      return refresh();
    }
    if ((el = q('[data-go]'))) return go(JSON.parse(el.dataset.go));
    if ((el = q('[data-toast]'))) return toast(el.dataset.toast);
  });
  root.addEventListener('input', (e) => {
    if (e.target.matches('[data-pin]')) { hoActions['ho-pin'](app, e.target.value); refresh(); const p = root.querySelector('[data-pin]'); if (p) { p.focus(); p.setSelectionRange(p.value.length, p.value.length); } }
    if (e.target.matches('.pal-in')) renderPalette(e.target.value);
  });
  root.addEventListener('keydown', (e) => {
    if (e.target.matches('.pal-in')) {
      if (e.key === 'ArrowDown') { e.preventDefault(); setSel(sel + 1); }
      if (e.key === 'ArrowUp') { e.preventDefault(); setSel(sel - 1); }
      if (e.key === 'Enter') { e.preventDefault(); runPal(sel); }
    }
  });
  document.addEventListener('keydown', (e) => {
    const anyOpen = !root.querySelector('.nd-sheet-wrap').hidden || !root.querySelector('.pal').hidden || !root.querySelector('.nd-pop').hidden;
    if ((e.metaKey || e.ctrlKey) && e.key.toLowerCase() === 'k' && !app.cfg.view.startsWith('worker-')) { e.preventDefault(); e.stopPropagation(); return root.querySelector('.pal').hidden ? openPalette() : closePalette(); }
    if (e.key === 'Escape' && anyOpen) { e.preventDefault(); return close(); }
    if (anyOpen && (e.key === 'ArrowLeft' || e.key === 'ArrowRight')) e.stopPropagation();
  });
  window.addEventListener('proto-sim', (e) => {
    const what = e.detail;
    if (app.cfg.view === 'worker-join') { joinSim(what); return refresh(); }
    if (app.cfg.view !== 'pair') { startPairing(); app.cfg.view = 'pair'; app.cfg.node = ''; history.replaceState(null, '', `?${params()}`); window.dispatchEvent(new CustomEvent('proto-url', { detail: { view: 'pair' } })); }
    pairSim(app, what);
    refresh();
  });
}

export function mount({ root, cfg }) {
  const workerView = cfg.view.startsWith('worker-');
  app = {
    root, cfg, phone: cfg.device === 'phone', V: VARIANTS[cfg.variant],
    S: scenario(workerView ? 'normal' : cfg.state), draft: new Map(), modal: null, sheetClose: null, flowClose: '',
    go, refresh, toast, close, popover, closePop,
  };
  root.classList.add('lc', `lc-${cfg.device}`, 'nd', `nd-${cfg.variant.toLowerCase()}`);
  if (workerView) root.classList.add('nd-solo', `nd-${cfg.view}`);
  document.documentElement.classList.add('lc-doc');
  root.innerHTML = `${AMBIENT}<div class="pg"></div>${overlays()}`;
  if (cfg.view === 'pair') startPairing();
  if (cfg.view === 'handover') startHandover(null);
  if (cfg.node && !app.S.nodes.some((n) => n.id === cfg.node)) cfg.node = '';
  document.title = `${workerView ? (cfg.view === 'worker-join' ? 'Join a SlipStream' : 'seedbox-ams status') : 'Nodes'} · SlipStream`;
  wire();
  refresh();
  startLive(root);
  setInterval(() => tickCountdown(app), 1000);
  window.__nodesApp = app;
}
