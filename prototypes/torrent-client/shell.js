// PROTOTYPE · torrent client screens on Direction C's shell (Lit console). Copied from ui-direction/variants/lit-console/variant.js and cut down to the chrome. Throwaway.

let D, C, root;
let isPhone;

const esc = (s) => String(s ?? '').replace(/[&<>"]/g, (c) => ({ '&': '&amp;', '<': '&lt;', '>': '&gt;', '"': '&quot;' }[c]));
const W = (id) => D.works[id];
const I = (n, c = '') => `<i data-lucide="${n}" class="ic ${c}" aria-hidden="true"></i>`;
const kindLabel = (w) => D.contentTypes[w.type].label;
const initials = (t) => t.replace(/^The /, '').split(/[\s:]+/).filter(Boolean).slice(0, 2).map((s) => s[0]).join('').toUpperCase();
const minutesAgo = (iso) => Math.round((D.now - new Date(iso)) / 60000);
const ago = (iso) => {
  const m = minutesAgo(iso);
  if (m < 60) return `${m} min ago`;
  const h = Math.floor(m / 60);
  if (h < 24) return `${h} h ago`;
  const d = Math.floor(h / 24);
  return d === 1 ? 'yesterday' : `${d} days ago`;
};
const hm = (iso) => new Date(iso).toTimeString().slice(0, 5);
const md = (iso) => new Date(iso).toLocaleDateString('en-US', { month: 'short', day: 'numeric' });

const MARK = `<svg class="mark" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" aria-hidden="true"><path d="M2.5 6.5c6 0 9.5 0 12.5 3.2"/><path d="M2.5 12h9.5"/><path d="M2.5 17.5c6 0 9.5 0 12.5-3.2"/><circle cx="18.6" cy="12" r="2.6" fill="currentColor" stroke="none"/></svg>`;

function art(w, cls = 'thumb', kind = 'poster') {
  return `<span class="art ${cls}"><img src="${w.art[kind]}" alt="" decoding="async" onerror="this.parentNode.classList.add('art-x');this.remove()"><span class="art-fb" aria-hidden="true"><b>${esc(initials(w.title))}</b><span>${esc(w.title)}</span></span></span>`;
}

const dot = (k) => `<span class="dot dot-${k}" aria-hidden="true"></span>`;
const running = () => D.jobs.filter((j) => j.status === 'running');
const degraded = () => D.nodes.filter((n) => n.status === 'degraded');

export const h = { esc, W, I, art, dot, ago, hm, md, initials, minutesAgo };

// ───────────────────────────── admin shell

const NAV = [
  { id: 'home', label: 'Home', icon: 'house' },
  { id: 'library', label: 'Library', icon: 'library', kids: [{ id: 'movies', label: 'Movies', count: () => D.library.movies }, { id: 'series', label: 'Series', count: () => D.library.series }] },
  { id: 'activity', label: 'Activity', icon: 'activity', live: () => running().length },
  { id: 'torrents', label: 'Torrents', icon: 'magnet', count: () => D.clients.reduce((n, c) => n + c.totals.torrents, 0), flag: () => (D.trackers.some((t) => t.status !== 'ok') ? 'warn' : '') },
  { id: 'calendar', label: 'Calendar', icon: 'calendar' },
  { id: 'wanted', label: 'Wanted', icon: 'circle-dashed' },
  { id: 'requests', label: 'Requests', icon: 'inbox', count: () => D.requests.filter((r) => r.state === 'pending').length },
  { id: 'history', label: 'History', icon: 'history' },
  { id: 'manual-import', label: 'Manual import', icon: 'folder-input' },
  { sep: true },
  { id: 'nodes', label: 'Nodes', icon: 'network', count: () => D.nodes.length, flag: () => (degraded().length ? 'warn' : '') },
  { id: 'settings', label: 'Settings', icon: 'settings', kids: [{ id: 'clients', label: 'Download Clients', count: () => D.clients.length, flag: () => (D.clients.some((c) => c.status !== 'ok') ? 'warn' : '') }, { id: 'goals', label: 'Seeding goals' }] },
  { id: 'system', label: 'System', icon: 'gauge', flag: () => 'fail', count: () => D.health.length },
];

function sidebar(active) {
  const item = (n, sub = false) => {
    const on = n.id === active;
    const right = n.live ? `<span class="nav-live">${dot('draft')}<span class="num">${n.live()}</span></span>`
      : n.count ? `<span class="nav-count num">${n.flag && n.flag() ? dot(n.flag()) : ''}${n.count()}</span>` : '';
    return `<button class="nav-i${sub ? ' sub' : ''}${on ? ' on' : ''}" data-nav="${n.id}" data-label="${esc(n.label)}" ${on ? 'aria-current="page"' : ''} title="${esc(n.label)}">${sub ? '<span class="nav-ind" aria-hidden="true"></span>' : I(n.icon)}<span class="nav-l">${esc(n.label)}</span>${right}</button>`;
  };
  return `<aside class="side" aria-label="Main">
    <div class="side-top"><span class="brand">${MARK}<span class="brand-w">SlipStream</span></span><button class="ib side-col" data-collapse aria-label="Collapse sidebar" aria-expanded="true">${I('panel-left-close')}</button></div>
    <nav class="nav">${NAV.map((n) => (n.sep ? '<div class="nav-sep" role="separator"></div>' : item(n) + (n.kids ? n.kids.map((k) => item(k, true)).join('') : ''))).join('')}</nav>
    <div class="side-foot"><span class="avatar">${D.admin.initials}</span><span class="side-who"><b>${D.admin.name}</b><span>Admin · atlas</span></span></div>
  </aside>`;
}

function topbar(crumbs) {
  return `<header class="top">
    <div class="crumbs">${crumbs}</div>
    <button class="search-btn" data-open="palette" aria-haspopup="dialog">${I('search')}<span>Search Works or run a command</span><kbd><span>⌘</span>K</kbd></button>
    <div class="top-r">
      <span class="rates num" title="Download and upload across all Nodes">${I('arrow-down')}${D.today.downloadRate} MB/s<span class="rates-sep"></span>${I('arrow-up')}${D.today.uploadRate} MB/s</span>
      <button class="ib bell" data-toast="Notifications are not in this prototype" aria-label="Notifications, ${D.health.length} unread">${I('bell')}<span class="bell-n num">${D.health.length}</span></button>
      <button class="btn" data-toast="Adding a Work is not in this prototype">${I('plus')}Add Work</button>
    </div>
  </header>`;
}

const TABS_ADMIN = [
  { id: 'home', label: 'Home', icon: 'house' },
  { id: 'library', label: 'Library', icon: 'library' },
  { id: 'activity', label: 'Activity', icon: 'activity', badge: () => running().length },
  { id: 'search', label: 'Search', icon: 'search' },
  { id: 'more', label: 'More', icon: 'menu' },
];
function tabs(list, active) {
  return `<nav class="tabs" aria-label="Tabs">${list.map((t) => `<button class="tab${t.id === active ? ' on' : ''}" data-tab="${t.id}" data-label="${t.label}" ${t.id === active ? 'aria-current="page"' : ''}>${I(t.icon)}<span>${t.label}</span>${t.badge ? `<span class="tab-b num">${t.badge()}</span>` : ''}</button>`).join('')}</nav>`;
}

export function phoneSection(title, inner, extra = '') {
  return `<section class="psec"><div class="psec-h"><h2>${title}</h2>${extra}</div>${inner}</section>`;
}
// ───────────────────────────── overlays

function paletteItems(q) {
  q = q.trim().toLowerCase();
  const works = Object.values(D.works).map((w) => ({ kind: 'work', id: w.id, label: w.title, meta: `${w.year} · ${kindLabel(w)}`, w }));
  const cmds = [
    { label: 'Add a Work', icon: 'plus', meta: 'Library' },
    { label: 'Open Torrents', icon: 'magnet', meta: 'Activity', go: 'torrents' },
    { label: 'Add a Download Client', icon: 'plus', meta: 'Settings', go: 'clients' },
    { label: 'Edit seeding goals', icon: 'target', meta: 'Settings', go: 'goals' },
    { label: 'Manual search for The Batman', icon: 'search', meta: 'Failed at Import' },
    { label: 'Search every missing Item', icon: 'circle-dashed', meta: 'Wanted' },
    { label: 'Open Activity', icon: 'activity', meta: `${running().length} running` },
    { label: 'Create a Join Code to pair a Node', icon: 'network', meta: 'Nodes' },
    { label: 'Issue a Signup link', icon: 'user-round-plus', meta: 'Accounts' },
    { label: 'Test connection on relay-fra', icon: 'radio-tower', meta: 'Degraded' },
    ...(isPhone ? [] : [{ label: 'Toggle sidebar', icon: 'panel-left', meta: 'View', act: 'collapse' }]),
  ].map((c) => ({ ...c, kind: 'cmd' }));
  const hit = (s) => !q || s.toLowerCase().includes(q);
  const ws = works.filter((x) => hit(x.label)).slice(0, q ? 8 : 5);
  const cs = cmds.filter((x) => hit(x.label));
  return { ws, cs };
}

function renderPaletteList(q) {
  const { ws, cs } = paletteItems(q);
  const list = root.querySelector('.pal-list');
  let n = 0;
  const row = (x) => {
    const id = `pal-o${n++}`;
    const lead = x.kind === 'work' ? art(x.w) : `<span class="pal-ic">${I(x.icon)}</span>`;
    const tail = `<span class="t3 pal-meta">${esc(x.meta)}</span>`;
    return `<li role="option" id="${id}" class="pal-o" data-kind="${x.kind}" data-label="${esc(x.label)}" ${x.act ? `data-act="${x.act}"` : ''} ${x.go ? `data-go="${x.go}"` : ''} aria-selected="false">${lead}<span class="pal-l">${esc(x.label)}</span>${tail}</li>`;
  };
  const groups = [];
  if (ws.length) groups.push(`<li role="presentation" class="pal-g">Works</li>${ws.map(row).join('')}`);
  if (cs.length) groups.push(`<li role="presentation" class="pal-g">Commands</li>${cs.map(row).join('')}`);
  list.innerHTML = groups.join('') || `<li class="pal-empty" role="presentation">Nothing matches “${esc(q)}”. Try a Work, or a command such as Add a Work.</li>`;
  C.icons();
  setSel(0);
}

let sel = 0;
function setSel(i) {
  const opts = [...root.querySelectorAll('.pal-o')];
  if (!opts.length) return;
  sel = (i + opts.length) % opts.length;
  opts.forEach((o, k) => o.setAttribute('aria-selected', String(k === sel)));
  root.querySelector('.pal-in').setAttribute('aria-activedescendant', opts[sel].id);
  opts[sel].scrollIntoView({ block: 'nearest' });
}

function activate(o) {
  if (!o) return;
  if (o.dataset.kind === 'work') return C.go('work');
  if (o.dataset.go) return C.go(o.dataset.go);
  closeAll();
  if (o.dataset.act === 'collapse') return toggleSidebar();
  toast(`${o.dataset.label} is not in this prototype`);
}

let lastFocus = null;
function openPalette() {
  closeAll();
  lastFocus = document.activeElement;
  const pal = root.querySelector('.pal');
  pal.hidden = false;
  const inp = pal.querySelector('.pal-in');
  inp.value = '';
  renderPaletteList('');
  requestAnimationFrame(() => inp.focus({ preventScroll: true }));
}

export function closeAll() {
  root.querySelectorAll('.pal, .sheet-wrap, .menu, .pop').forEach((el) => { el.hidden = true; });
  if (lastFocus && document.contains(lastFocus)) lastFocus.focus({ preventScroll: true });
  lastFocus = null;
}
const anyOpen = () => [...root.querySelectorAll('.pal, .sheet-wrap, .menu, .pop')].some((el) => !el.hidden);

let toastT;
export function toast(msg) {
  const t = root.querySelector('.toast');
  t.innerHTML = `${I('info')}<span>${esc(msg)}.</span>`;
  C.icons();
  t.classList.add('on');
  clearTimeout(toastT);
  toastT = setTimeout(() => t.classList.remove('on'), 2800);
}

export function openMenu(btn) {
  const a = S.actions(btn.dataset.menu);
  closeAll();
  lastFocus = btn;
  const items = a.items.map(([l, ic, k, to]) => `<button class="mi${k === 'danger' || k === 'act-danger' ? ' danger' : ''}" role="menuitem" data-mi="${k === 'go' ? to : ''}"${k && k.startsWith('act') ? ` data-act="${esc(to)}"` : ''} data-label="${esc(l)}">${I(ic)}<span>${esc(l)}</span></button>`).join('');
  if (isPhone) {
    const s = root.querySelector('.sheet-wrap');
    s.querySelector('.sheet').innerHTML = `<span class="grab" aria-hidden="true"></span><p class="sheet-t"><b>${esc(a.title)}</b>${a.sub ? `<span class="t3">${esc(a.sub)}</span>` : ''}</p><div class="sheet-grp" role="menu" aria-label="${esc(a.title)}">${items}</div><button class="sheet-x" data-close>Cancel</button>`;
    s.hidden = false;
    C.icons();
    s.querySelector('.mi').focus({ preventScroll: true });
    return;
  }
  const m = root.querySelector('.menu');
  m.innerHTML = items;
  m.setAttribute('aria-label', a.title);
  m.hidden = false;
  C.icons();
  const r = btn.getBoundingClientRect(), rr = root.getBoundingClientRect();
  m.style.top = `${r.bottom - rr.top + 6}px`;
  m.style.left = `${Math.max(8, r.right - rr.left - m.offsetWidth)}px`;
  m.querySelector('.mi').focus({ preventScroll: true });
}

function toggleSidebar() {
  const c = root.classList.toggle('collapsed');
  const b = root.querySelector('[data-collapse]');
  if (b) { b.setAttribute('aria-expanded', String(!c)); b.setAttribute('aria-label', c ? 'Expand sidebar' : 'Collapse sidebar'); b.innerHTML = I(c ? 'panel-left-open' : 'panel-left-close'); C.icons(); }
  try { localStorage.setItem('lc-collapsed', c ? '1' : ''); } catch { /* storage unavailable */ }
}

function overlays() {
  return `<div class="pal" hidden><div class="pal-scrim" data-close></div>
      <div class="pal-box" role="dialog" aria-modal="true" aria-label="Search Works and commands">
        <div class="pal-top">${I('search')}<input class="pal-in" type="text" role="combobox" aria-expanded="true" aria-controls="pal-list" aria-autocomplete="list" placeholder="Search Works or type a command" autocomplete="off" spellcheck="false">${isPhone ? '<button class="pal-cancel" data-close>Cancel</button>' : '<kbd>Esc</kbd>'}</div>
        <ul class="pal-list" id="pal-list" role="listbox" aria-label="Results"></ul>
        ${isPhone ? '' : `<div class="pal-foot t3"><span><kbd>↑</kbd><kbd>↓</kbd> move</span><span><kbd>↵</kbd> open</span><span><kbd>Esc</kbd> close</span></div>`}
      </div></div>
    <div class="menu" role="menu" hidden></div>
    <div class="pop" hidden></div>
    <div class="sheet-wrap" hidden><div class="sheet-scrim" data-close></div><div class="sheet"></div></div>
    <div class="toast" role="status" aria-live="polite"></div>`;
}

// ───────────────────────────── wiring

const NAV_SCREENS = { torrents: 'torrents', settings: 'clients', clients: 'clients', goals: 'goals', home: 'home', activity: 'home', movies: 'work' };

function wire() {
  root.addEventListener('click', (e) => {
    const t = e.target;
    const q = (sel) => t.closest(sel);
    let el;
    if ((el = q('[data-close]'))) return closeAll();
    if ((el = q('.mi'))) { if (el.dataset.mi) return C.go(el.dataset.mi); if (el.dataset.act && S.act) { closeAll(); return S.act(el.dataset.act, el); } closeAll(); return toast(`${el.dataset.label} is not in this prototype`); }
    if ((el = q('.pal-o'))) return activate(el);
    if ((el = q('[data-open="palette"]'))) return openPalette();
    if (S.click && S.click(e, q)) return;
    if ((el = q('[data-menu]'))) { e.stopPropagation(); return openMenu(el); }
    if ((el = q('[data-collapse]'))) return toggleSidebar();
    if ((el = q('[data-toast]'))) return toast(el.dataset.toast);
    if ((el = q('[data-go]'))) return C.go(el.dataset.go);
    if ((el = q('[data-nav]'))) {
      const to = NAV_SCREENS[el.dataset.nav];
      if (to) return C.go(to);
      return toast(`${el.dataset.label} is not in this prototype`);
    }
    if ((el = q('[data-tab]'))) {
      const id = el.dataset.tab;
      if (id === 'home') return C.go('home');
      if (id === 'activity') return C.go('torrents');
      if (id === 'search') return openPalette();
      if (id === 'more') return openMore(el);
      return toast(`${el.dataset.label} is not in this prototype`);
    }
    if (!q('.menu, .pop')) { const open = root.querySelector('.menu:not([hidden]), .pop:not([hidden])'); if (open) closeAll(); }
  });

  root.querySelector('.pal-in').addEventListener('input', (e) => renderPaletteList(e.target.value));
  root.querySelector('.pal-in').addEventListener('keydown', (e) => {
    if (e.key === 'ArrowDown') { e.preventDefault(); setSel(sel + 1); }
    if (e.key === 'ArrowUp') { e.preventDefault(); setSel(sel - 1); }
    if (e.key === 'Enter') { e.preventDefault(); activate(root.querySelectorAll('.pal-o')[sel]); }
  });

  document.addEventListener('keydown', (e) => {
    if ((e.metaKey || e.ctrlKey) && e.key.toLowerCase() === 'k') { e.preventDefault(); e.stopPropagation(); return root.querySelector('.pal').hidden ? openPalette() : closeAll(); }
    if (e.key === 'Escape' && anyOpen()) { e.preventDefault(); return closeAll(); }
    if (anyOpen() && (e.key === 'ArrowLeft' || e.key === 'ArrowRight')) e.stopPropagation();
    const menu = e.target.closest && e.target.closest('.menu, .sheet-grp');
    if (menu && (e.key === 'ArrowDown' || e.key === 'ArrowUp')) {
      e.preventDefault();
      const items = [...menu.querySelectorAll('.mi, .inb')];
      const i = items.indexOf(document.activeElement);
      items[(i + (e.key === 'ArrowDown' ? 1 : -1) + items.length) % items.length].focus();
    }
  });
}

function openMore(btn) {
  closeAll();
  lastFocus = btn;
  const rest = NAV.filter((n) => !n.sep && !['home', 'library', 'activity'].includes(n.id));
  const s = root.querySelector('.sheet-wrap');
  const flat = rest.flatMap((n) => [n, ...(n.kids || []).map((k) => ({ ...k, icon: 'corner-down-right' }))]);
  s.querySelector('.sheet').innerHTML = `<span class="grab" aria-hidden="true"></span><p class="sheet-t"><b>More</b></p><div class="sheet-grp" role="menu" aria-label="More">${flat.map((n) => `<button class="mi" role="menuitem" data-mi="${NAV_SCREENS[n.id] || ''}" data-label="${n.label}">${I(n.icon)}<span>${n.label}</span>${n.count ? `<span class="t3 num mi-r">${n.flag && n.flag() ? dot(n.flag()) : ''}${n.count()}</span>` : ''}</button>`).join('')}</div><button class="sheet-x" data-close>Cancel</button>`;
  s.hidden = false;
  C.icons();
  s.querySelector('.mi').focus({ preventScroll: true });
}

// A phone sheet or desk dialog with any content. `html` is the inside; the shell adds grab handle and scrim.
export function openSheet(html, from) {
  closeAll();
  lastFocus = from || document.activeElement;
  const sw = root.querySelector('.sheet-wrap');
  sw.querySelector('.sheet').innerHTML = `<span class="grab" aria-hidden="true"></span>${html}`;
  sw.hidden = false;
  C.icons();
  const f = sw.querySelector('button, input, select, [tabindex]');
  if (f) f.focus({ preventScroll: true });
}

// ───────────────────────────── ambient: the page takes its colour from the Work in focus

const tintCache = new Map();
function sampleTint(src) {
  if (tintCache.has(src)) return tintCache.get(src);
  const p = new Promise((res) => {
    const im = new Image();
    im.onload = () => {
      try {
        const c = document.createElement('canvas');
        c.width = 24; c.height = 36;
        const g = c.getContext('2d', { willReadFrequently: true });
        g.drawImage(im, 0, 0, 24, 36);
        const d = g.getImageData(0, 0, 24, 36).data;
        let x = 0, y = 0, ws = 0, ss = 0;
        for (let i = 0; i < d.length; i += 4) {
          const r = d[i] / 255, gg = d[i + 1] / 255, b = d[i + 2] / 255;
          const mx = Math.max(r, gg, b), mn = Math.min(r, gg, b), l = (mx + mn) / 2, dd = mx - mn;
          if (dd < 0.04) continue;
          const sat = dd / (1 - Math.abs(2 * l - 1));
          let h = mx === r ? ((gg - b) / dd) % 6 : mx === gg ? (b - r) / dd + 2 : (r - gg) / dd + 4;
          h = (h * 60 + 360) % 360;
          const w = sat * sat * (1 - Math.abs(l - 0.5) * 1.6);
          if (w <= 0) continue;
          x += Math.cos((h * Math.PI) / 180) * w; y += Math.sin((h * Math.PI) / 180) * w; ws += w; ss += sat * w;
        }
        if (!ws) return res({ h: 220, s: 12 });
        res({ h: ((Math.atan2(y, x) * 180) / Math.PI + 360) % 360, s: Math.round(Math.min(70, Math.max(18, (ss / ws) * 80))) });
      } catch { res(null); }
    };
    im.onerror = () => res(null);
    im.src = src;
  });
  tintCache.set(src, p);
  return p;
}

let ambOn = 0, ambWork = null, ambLead = null;
export function setLead(id) { ambLead = id; setAmbient(id); }
function setAmbient(id) {
  if (ambWork === id) return;
  ambWork = id;
  const w = W(id);
  const layers = [...root.querySelectorAll('.amb-i')];
  const next = layers[1 - ambOn];
  const swap = () => {
    if (ambWork !== id) return;
    next.style.backgroundImage = `url('${w.art.backdrop}')`;
    next.classList.add('on');
    layers[ambOn].classList.remove('on');
    ambOn = 1 - ambOn;
  };
  const pre = new Image();
  pre.onload = swap;
  pre.onerror = () => { if (ambWork === id) { next.style.backgroundImage = ''; swap(); } };
  pre.src = w.art.backdrop;
  sampleTint(w.art.poster).then((t) => {
    if (!t || ambWork !== id) return;
    root.style.setProperty('--h', t.h.toFixed(1));
    root.style.setProperty('--s', `${t.s}%`);
  });
  root.querySelectorAll('[data-work].focus').forEach((el) => el.classList.remove('focus'));
  root.querySelectorAll(`[data-work="${id}"]`).forEach((el) => el.classList.add('focus'));
}

// Rows carrying data-work recolour the page on hover or focus (desk) or when scrolled to the reading line (phone), and fall back to `lead`.
function wireAmbient(lead) {
  ambLead = lead;
  setAmbient(lead);
  const rows = root.querySelectorAll('[data-work][data-amb]');
  if (isPhone) {
    const io = new IntersectionObserver((es) => {
      const hit = es.filter((x) => x.isIntersecting).sort((a, b) => a.boundingClientRect.top - b.boundingClientRect.top)[0];
      if (hit) setAmbient(hit.target.dataset.work);
    }, { root, rootMargin: '-30% 0px -55% 0px' });
    rows.forEach((el) => io.observe(el));
    return;
  }
  let enterT, leaveT;
  const enter = (el) => { clearTimeout(leaveT); clearTimeout(enterT); enterT = setTimeout(() => setAmbient(el.dataset.work), 90); };
  const leave = () => { clearTimeout(enterT); clearTimeout(leaveT); leaveT = setTimeout(() => setAmbient(ambLead), 450); };
  rows.forEach((el) => {
    el.addEventListener('mouseenter', () => enter(el));
    el.addEventListener('mouseleave', leave);
    el.addEventListener('focusin', () => enter(el));
    el.addEventListener('focusout', leave);
  });
}
export { setAmbient };

const AMBIENT = '<div class="amb" aria-hidden="true"><div class="amb-i"></div><div class="amb-i"></div><div class="amb-tint"></div></div>';

let S;

// screen: { id, title, nav, tab, crumbs (desk html), body (html), lead (work id for the tint), actions(key) → {title, sub, items: [label, icon, kind, goScreen | actKey]}, act(key, el) for kind 'act'/'act-danger' items, click(e, q) → true when handled, after() }
export function mount(ctx, screen) {
  C = ctx; D = ctx.data; root = ctx.root; S = screen;
  isPhone = ctx.device === 'phone';
  root.classList.add('lc', `lc-${ctx.device}`, `lc-${screen.id}`);
  document.documentElement.classList.add('lc-doc');
  document.title = `${screen.title} · SlipStream`;
  try { if (!isPhone && localStorage.getItem('lc-collapsed')) root.classList.add('collapsed'); } catch { /* storage unavailable */ }
  const body = isPhone
    ? `<div class="statusbar" aria-hidden="true"></div>${screen.body}${tabs(TABS_ADMIN, screen.tab)}`
    : `<div class="shell">${sidebar(screen.nav)}<div class="mainc">${topbar(screen.crumbs)}<main class="content">${screen.body}</main></div></div>`;
  root.innerHTML = AMBIENT + body + overlays();
  if (root.classList.contains('collapsed')) { const b = root.querySelector('[data-collapse]'); if (b) { b.innerHTML = I('panel-left-open'); b.setAttribute('aria-expanded', 'false'); b.setAttribute('aria-label', 'Expand sidebar'); } }
  ctx.icons();
  wire();
  wireAmbient(screen.lead);
  if (screen.after) screen.after(root);
}
