// PROTOTYPE · entry for the manual search sheet prototype (#107). Owns the sheet's state, the simulated per-Indexer streaming and the switcher. Throwaway.

import { loadIcons, mount, closeMenus, menuOpen } from './shell.js';
import { fixtures, indexers, clients, admin } from './data.js';
import { evaluate, normScope, SORTS } from './model.js';
import { wireCommon } from './common.js';
import { workDesk, workPhone } from './work.js';

const VARIANTS = { A: { file: './a.js', name: 'Verdict strip' }, B: { file: './b.js', name: 'Target columns' }, C: { file: './c.js', name: 'Cell first' }, D: { file: './d.js', name: 'The plan' }, E: { file: './e.js', name: 'Shelves by outcome' }, F: { file: './f.js', name: 'Needs, then choices' } };
const SCOPES = ['work', 'level', 'item', 'cell'];
const keys = Object.keys(VARIANTS);
const params = new URLSearchParams(location.search);
const P = {
  variant: VARIANTS[params.get('variant')] ? params.get('variant') : 'D',
  fixture: fixtures[params.get('fixture')] ? params.get('fixture') : 'movie',
  device: params.get('device') === 'phone' ? 'phone' : 'wide',
  scope: SCOPES.includes(params.get('scope')) ? params.get('scope') : 'work',
};
document.body.classList.add(`device-${P.device}`, 'surface-admin');

function navigate(patch, keep = ['settled', 'at']) {
  const next = new URLSearchParams();
  for (const [k, v] of Object.entries({ ...P, ...patch })) next.set(k, v);
  for (const k of keep) if (params.has(k)) next.set(k, params.get(k));
  location.search = next.toString();
}

const bar = document.getElementById('proto-bar');
const btn = (label, on, pressed, title) => { const b = document.createElement('button'); b.textContent = label; b.onclick = on; if (pressed !== undefined) b.setAttribute('aria-pressed', pressed); if (title) b.title = title; return b; };
const sep = () => Object.assign(document.createElement('span'), { className: 'sep' });
const cycle = (d) => navigate({ variant: keys[(keys.indexOf(P.variant) + d + keys.length) % keys.length] });
bar.append(
  btn('←', () => cycle(-1)),
  Object.assign(document.createElement('span'), { className: 'label', textContent: `${P.variant} · ${VARIANTS[P.variant].name}` }),
  btn('→', () => cycle(1)), sep(),
  btn('Movie', () => navigate({ fixture: 'movie' }), P.fixture === 'movie'),
  btn('Series', () => navigate({ fixture: 'series' }), P.fixture === 'series'), sep(),
  ...SCOPES.map((s) => btn(s[0].toUpperCase() + s.slice(1), () => navigate({ scope: s }), P.scope === s, `?scope=${s}`)), sep(),
  btn('Wide', () => navigate({ device: 'wide' }), P.device === 'wide'),
  btn('Phone', () => navigate({ device: 'phone' }), P.device === 'phone'), sep(),
  btn('Replay', () => navigate({}, []), undefined, 'Replay the streaming search'),
);
addEventListener('keydown', (e) => {
  const t = e.target; if (t.closest && t.closest('input, textarea, select, [contenteditable]')) return;
  if (menuOpen()) return;
  if (e.key === 'ArrowLeft' && !e.metaKey && !e.ctrlKey) cycle(-1);
  if (e.key === 'ArrowRight' && !e.metaKey && !e.ctrlKey) cycle(1);
  if (e.key === '\\') bar.hidden = !bar.hidden;
});

await loadIcons();
const fx = fixtures[P.fixture];
const root = document.getElementById('app');
const host = mount(root, {
  device: P.device, admin, work: fx.work,
  crumbs: `<span class="crumb">Library</span><span class="t3">/</span><span class="crumb">${fx.work.kind === 'movie' ? 'Movies' : 'Series'}</span><span class="t3">/</span><span class="crumb cur">${fx.work.title}</span>`,
  body: P.device === 'phone' ? workPhone(fx) : workDesk(fx),
});
const variant = await import(VARIANTS[P.variant].file);

let cache = null;
const app = {
  fx, indexers, clients, device: P.device, phone: P.device === 'phone',
  sc: normScope(fx, { kind: P.scope, cell: params.get('cell') || undefined }),
  filter: params.get('filter') === 'all' ? 'all' : 'wanted',
  sort: SORTS[params.get('sort')] ? params.get('sort') : 'best',
  selected: new Set(), arrived: new Set(), grabbed: new Map(), clientFor: new Map(),
  open: params.get('open') || null,
  row: params.get('row') || params.get('open') || null,
  isOpen: true,
  rows() {
    const k = `${JSON.stringify(this.sc)}|${[...this.arrived].join()}`;
    if (!cache || cache.k !== k) {
      const ok = new Set(indexers.filter((x) => this.arrived.has(x.id) && x.outcome === 'ok').map((x) => x.id));
      cache = { k, rows: evaluate(fx, this.sc).filter((x) => ok.has(x.r.indexer)) };
    }
    return cache.rows;
  },
  visible() {
    const rows = this.rows().filter((x) => this.filter === 'all' || x.wanted);
    return rows.sort(SORTS[this.sort].cmp);
  },
  rowById(id) { return this.rows().find((x) => x.r.id === id) || evaluate(fx, normScope(fx, { kind: 'work' })).find((x) => x.r.id === id); },
  countFor(ix) { return this.rows().filter((x) => x.r.indexer === ix).length; },
  setScope(patch) { this.sc = normScope(fx, { ...this.sc, ...patch }); if (variant.onScope) variant.onScope(this); this.render(); },
  openAt(patch) { this.sc = normScope(fx, { ...this.sc, ...patch }); this.isOpen = true; this.open = null; this.row = null; if (variant.onScope) variant.onScope(this); startStream(); this.render(); },
  close() { this.isOpen = false; closeMenus(); this.render(); },
  render() {
    const keep = {};
    host.querySelectorAll('[data-keep]').forEach((el) => { keep[el.dataset.keep] = el.scrollTop; });
    host.innerHTML = this.isOpen ? variant.render(this) : '';
    host.querySelectorAll('[data-keep]').forEach((el) => { if (keep[el.dataset.keep] != null) el.scrollTop = keep[el.dataset.keep]; });
    if (variant.after) variant.after(this, host);
  },
};
if (variant.init) variant.init(app);

const timers = [];
function startStream() {
  timers.forEach(clearTimeout);
  app.arrived.clear();
  const at = params.has('at') ? +params.get('at') : null;
  for (const x of indexers) {
    if (params.get('settled') === '1') { app.arrived.add(x.id); continue; }
    if (at != null) { if (x.delay <= at) app.arrived.add(x.id); continue; }
    timers.push(setTimeout(() => { app.arrived.add(x.id); app.render(); }, x.delay));
  }
}

wireCommon(app, host);
root.addEventListener('click', (e) => {
  const el = e.target.closest('[data-ms]');
  if (!el || host.contains(el)) return;
  const kind = el.dataset.ms;
  app.openAt({ kind, ...(el.dataset.level && { level: el.dataset.level }), ...(el.dataset.item && { item: el.dataset.item }), ...(el.dataset.cell && { cell: el.dataset.cell }) });
});
document.addEventListener('keydown', (e) => {
  if (e.key !== 'Escape') return;
  if (menuOpen()) { closeMenus(); return; }
  if (variant.escape && variant.escape(app)) return;
  if (app.isOpen) app.close();
});

startStream();
app.render();
