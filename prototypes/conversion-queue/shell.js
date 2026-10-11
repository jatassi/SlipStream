// PROTOTYPE shell: Lit console's Admin chrome (sidebar, top bar, phone header and tabs, ambient tint, menus, sheets, toast),
// lifted from prototypes/ui-direction/variants/lit-console/variant.js so the variants only draw the queue and the dialog. Throwaway.

let D, root, isPhone, go;

export const esc = (s) => String(s ?? '').replace(/[&<>"]/g, (c) => ({ '&': '&amp;', '<': '&lt;', '>': '&gt;', '"': '&quot;' }[c]));
export const I = (n, c = '') => `<i data-lucide="${n}" class="ic ${c}" aria-hidden="true"></i>`;
export const dot = (k) => `<span class="dot dot-${k}" aria-hidden="true"></span>`;
const initials = (t) => t.replace(/^The /, '').split(/[\s:]+/).filter(Boolean).slice(0, 2).map((s) => s[0]).join('').toUpperCase();
export const hm = (iso) => new Date(iso).toTimeString().slice(0, 5);
export const ago = (iso) => {
  const m = Math.round((D.now - new Date(iso)) / 60000);
  if (m < 60) return `${m} min ago`;
  const h = Math.floor(m / 60);
  if (h < 24) return `${h} h ago`;
  const d = Math.floor(h / 24);
  return d === 1 ? 'yesterday' : `${d} days ago`;
};
export const fmtEta = (m) => (m == null ? '' : m <= 0 ? 'now' : m < 60 ? `${Math.ceil(m)} min` : `${Math.floor(m / 60)} h ${Math.round(m % 60)} min`);
export const fmtHours = (h) => (h < 1 ? `${Math.round(h * 60)} min` : h < 48 ? `${Math.floor(h)} h${Math.round((h % 1) * 60) ? ` ${Math.round((h % 1) * 60)} min` : ''}` : `${Math.floor(h / 24)} d ${Math.round(h % 24)} h`);

// Content Type lookups. Every Work, Item and group label goes through the Work's descriptor.
export const W = (id) => D.works[id];
export const ct = (w) => D.contentTypes[w.type];
export const itemLabel = (conv) => {
  const w = W(conv.work), t = ct(w);
  const parts = [];
  if (t.layout.itemInRow === 'show' && conv.item) parts.push(t.item(conv.item));
  if (conv.edition && conv.edition !== 'Standard') parts.push(conv.edition);
  return parts.join(' · ');
};
export const node = (id) => D.nodes.find((n) => n.id === id);
export const target = (id) => D.targets[id];
export const capable = (cap) => D.nodes.filter((n) => n.status === 'online' && n.proven.includes(cap));

export function art(w, cls = 'thumb', kind) {
  const k = kind || ct(w).artwork.thumb;
  return `<span class="art ${cls}"><img src="${w.art[k]}" alt="" decoding="async" onerror="this.parentNode.classList.add('art-x');this.remove()"><span class="art-fb" aria-hidden="true"><b>${esc(initials(w.title))}</b><span>${esc(w.title)}</span></span></span>`;
}

const MARK = `<svg class="mark" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" aria-hidden="true"><path d="M2.5 6.5c6 0 9.5 0 12.5 3.2"/><path d="M2.5 12h9.5"/><path d="M2.5 17.5c6 0 9.5 0 12.5-3.2"/><circle cx="18.6" cy="12" r="2.6" fill="currentColor" stroke="none"/></svg>`;

const convCount = () => D.conversions.length + D.backfill.reduce((n, b) => n + (b.works ? b.works.length : b.items.reduce((m, i) => m + i.episodes, 0)), 0);
const failedCount = () => D.conversions.filter((c) => c.state.is === 'failed').length;
const noCapCount = () => D.conversions.filter((c) => c.state.is === 'waiting' && c.state.reason === 'no-capable-node').length;

const NAV = [
  { id: 'home', label: 'Home', icon: 'house' },
  { id: 'library', label: 'Library', icon: 'library', kids: [{ id: 'movies', label: 'Movies', count: () => 812 }, { id: 'series', label: 'Series', count: () => 96 }] },
  { id: 'activity', label: 'Activity', icon: 'activity', live: () => 7, kids: [{ id: 'jobs', label: 'Jobs', count: () => 5 }, { id: 'conversions', label: 'Conversions', count: convCount, flag: () => (failedCount() ? 'fail' : '') }] },
  { id: 'calendar', label: 'Calendar', icon: 'calendar' },
  { id: 'wanted', label: 'Wanted', icon: 'circle-dashed' },
  { id: 'requests', label: 'Requests', icon: 'inbox', count: () => 2 },
  { id: 'history', label: 'History', icon: 'history' },
  { sep: true },
  { id: 'nodes', label: 'Nodes', icon: 'network', count: () => D.nodes.length, flag: () => 'warn' },
  { id: 'settings', label: 'Settings', icon: 'settings', kids: [{ id: 'profiles', label: 'Profiles', count: () => D.profiles.length }, { id: 'targets', label: 'Targets', count: () => Object.keys(D.targets).length }] },
  { id: 'system', label: 'System', icon: 'gauge' },
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

const crumbsHtml = (list) => list.map((c, i) => (i === list.length - 1 ? `<span class="crumb cur">${esc(c)}</span>` : `<button class="crumb" data-nav="${esc(c.toLowerCase())}" data-label="${esc(c)}">${esc(c)}</button>${I('chevron-right', 'crumb-sep')}`)).join('');

function topbar(crumbs) {
  return `<header class="top">
    <div class="crumbs">${crumbsHtml(crumbs)}</div>
    <button class="search-btn" data-toast="The palette is in the direction prototype" aria-haspopup="dialog">${I('search')}<span>Search Works or run a command</span><kbd><span>⌘</span>K</kbd></button>
    <div class="top-r">
      <button class="ib bell" data-toast="Notifications are not in this prototype" aria-label="Notifications, 2 unread">${I('bell')}<span class="bell-n num">2</span></button>
      <button class="btn" data-toast="Adding a Work is not in this prototype">${I('plus')}Add Work</button>
    </div>
  </header>`;
}

const TABS = [
  { id: 'home', label: 'Home', icon: 'house' },
  { id: 'library', label: 'Library', icon: 'library' },
  { id: 'activity', label: 'Activity', icon: 'activity', badge: () => 7 },
  { id: 'search', label: 'Search', icon: 'search' },
  { id: 'more', label: 'More', icon: 'menu' },
];
const tabs = (active) => `<nav class="tabs" aria-label="Tabs">${TABS.map((t) => `<button class="tab${t.id === active ? ' on' : ''}" data-tab="${t.id}" data-label="${t.label}" ${t.id === active ? 'aria-current="page"' : ''}>${I(t.icon)}<span>${t.label}</span>${t.badge ? `<span class="tab-b num">${t.badge()}</span>` : ''}</button>`).join('')}</nav>`;

// Mounts a screen. wide: sidebar + top bar + content. phone: status bar, push-nav header with a back label, content, tabs.
export function page({ nav, crumbs, back, title, tab, body, overlay = '' }) {
  const AMBIENT = '<div class="amb" aria-hidden="true"><div class="amb-i"></div><div class="amb-i"></div><div class="amb-tint"></div></div>';
  const inner = isPhone
    ? `<div class="statusbar" aria-hidden="true"></div>
       <header class="ph-nav"><button class="back tap" data-toast="${esc(back)} is not in this prototype">${I('chevron-left')}${esc(back)}</button><span class="grow"></span></header>
       <div class="ph-gap" aria-hidden="true" style="height:44px"></div><header class="ph-head"><h1>${esc(title)}</h1></header>
       ${body}
       ${tabs(tab)}`
    : `<div class="shell">${sidebar(nav)}<div class="mainc">${topbar(crumbs)}<main class="content">${body}</main></div></div>`;
  root.innerHTML = AMBIENT + inner + overlays() + overlay;
  window.lucide && window.lucide.createIcons();
}

function overlays() {
  return `<div class="menu" role="menu" hidden></div>
    <div class="sheet-wrap" hidden><div class="sheet-scrim" data-close></div><div class="sheet"></div></div>
    <div class="toast" role="status" aria-live="polite"></div>`;
}

let toastT;
export function toast(msg) {
  const t = root.querySelector('.toast');
  t.innerHTML = `${I('info')}<span>${esc(msg)}</span>`;
  window.lucide && window.lucide.createIcons();
  t.classList.add('on');
  clearTimeout(toastT);
  toastT = setTimeout(() => t.classList.remove('on'), 3000);
}

let lastFocus = null;
export function closeMenus() {
  root.querySelectorAll('.sheet-wrap, .menu').forEach((el) => { el.hidden = true; });
  if (lastFocus && document.contains(lastFocus)) lastFocus.focus({ preventScroll: true });
  lastFocus = null;
}

// items: [label, lucide icon, 'danger' | undefined, toast message | undefined]
export function openMenu(btn, { title, sub, items }) {
  closeMenus();
  lastFocus = btn;
  const html = items.map(([l, ic, k, msg]) => `<button class="mi${k === 'danger' ? ' danger' : ''}" role="menuitem" data-msg="${esc(msg || `${l} is not in this prototype`)}">${I(ic)}<span>${esc(l)}</span></button>`).join('');
  if (isPhone) {
    const s = root.querySelector('.sheet-wrap');
    s.querySelector('.sheet').innerHTML = `<span class="grab" aria-hidden="true"></span><p class="sheet-t"><b>${esc(title)}</b>${sub ? `<span class="t3">${esc(sub)}</span>` : ''}</p><div class="sheet-grp" role="menu" aria-label="${esc(title)}">${html}</div><button class="sheet-x" data-close>Cancel</button>`;
    s.hidden = false;
    window.lucide && window.lucide.createIcons();
    s.querySelector('.mi').focus({ preventScroll: true });
    return;
  }
  const m = root.querySelector('.menu');
  m.innerHTML = html;
  m.setAttribute('aria-label', title);
  m.hidden = false;
  window.lucide && window.lucide.createIcons();
  const r = btn.getBoundingClientRect(), rr = root.getBoundingClientRect();
  m.style.top = `${r.bottom - rr.top + 6}px`;
  m.style.left = `${Math.max(8, r.right - rr.left - m.offsetWidth)}px`;
  m.querySelector('.mi').focus({ preventScroll: true });
}

function wireShell() {
  root.addEventListener('click', (e) => {
    const q = (s) => e.target.closest(s);
    let el;
    if ((el = q('[data-close]')) && q('.sheet-wrap')) return closeMenus();
    if ((el = q('.mi'))) { closeMenus(); return toast(el.dataset.msg); }
    if ((el = q('[data-collapse]'))) { root.classList.toggle('collapsed'); return; }
    if ((el = q('[data-toast]'))) return toast(el.dataset.toast);
    if ((el = q('[data-nav]'))) {
      const id = el.dataset.nav;
      if (id === 'conversions' || id === 'activity') return go('queue');
      if (id === 'profiles' || id === 'settings') return go('backfill');
      return toast(`${el.dataset.label} is not in this prototype`);
    }
    if ((el = q('[data-tab]'))) {
      if (el.dataset.tab === 'activity') return go('queue');
      if (el.dataset.tab === 'more') return go('backfill');
      return toast(`${el.dataset.label} is not in this prototype`);
    }
    if (!q('.menu, [data-menu]')) { const m = root.querySelector('.menu:not([hidden])'); if (m) closeMenus(); }
  });
  document.addEventListener('keydown', (e) => {
    if (e.key === 'Escape') closeMenus();
    const menu = e.target.closest && e.target.closest('.menu, .sheet-grp');
    if (menu && (e.key === 'ArrowDown' || e.key === 'ArrowUp')) {
      e.preventDefault();
      const items = [...menu.querySelectorAll('.mi')];
      const i = items.indexOf(document.activeElement);
      items[(i + (e.key === 'ArrowDown' ? 1 : -1) + items.length) % items.length].focus();
    }
  });
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

let ambOn = 0, ambWork = null;
// A failed Work never drives the tint; pass a running or waiting Work.
export function setAmbient(id) {
  if (ambWork === id) return;
  ambWork = id;
  const w = W(id), hero = w.art[ct(w).artwork.hero];
  const layers = [...root.querySelectorAll('.amb-i')];
  const next = layers[1 - ambOn];
  const swap = () => {
    if (ambWork !== id) return;
    next.style.backgroundImage = `url('${hero}')`;
    next.classList.add('on');
    layers[ambOn].classList.remove('on');
    ambOn = 1 - ambOn;
  };
  const pre = new Image();
  pre.onload = swap;
  pre.onerror = () => { if (ambWork === id) { next.style.backgroundImage = ''; swap(); } };
  pre.src = hero;
  sampleTint(w.art.poster).then((t) => {
    if (!t || ambWork !== id) return;
    root.style.setProperty('--h', t.h.toFixed(1));
    root.style.setProperty('--s', `${t.s}%`);
  });
  root.querySelectorAll('[data-work].focus').forEach((el) => el.classList.remove('focus'));
  root.querySelectorAll(`[data-work="${id}"]`).forEach((el) => el.classList.add('focus'));
}

// Rows carrying data-work retint the page on hover or focus (wide) or when scrolled to the reading line (phone).
export function wireAmbient(selector, lead) {
  ambWork = null;
  setAmbient(lead);
  const rows = root.querySelectorAll(selector);
  if (isPhone) {
    const io = new IntersectionObserver((es) => {
      const hit = es.filter((e) => e.isIntersecting).sort((a, b) => a.boundingClientRect.top - b.boundingClientRect.top)[0];
      if (hit) setAmbient(hit.target.dataset.work);
    }, { root, rootMargin: '-30% 0px -55% 0px' });
    rows.forEach((el) => io.observe(el));
    return;
  }
  let enterT, leaveT;
  const enter = (el) => { clearTimeout(leaveT); clearTimeout(enterT); enterT = setTimeout(() => setAmbient(el.dataset.work), 90); };
  const leave = () => { clearTimeout(enterT); clearTimeout(leaveT); leaveT = setTimeout(() => setAmbient(lead), 450); };
  rows.forEach((el) => {
    el.addEventListener('mouseenter', () => enter(el));
    el.addEventListener('mouseleave', leave);
    el.addEventListener('focusin', () => enter(el));
    el.addEventListener('focusout', leave);
  });
}

// ───────────────────────────── live ticker
// Annotate [data-live-progress="<convId>"] (gets --p and aria-valuenow), [data-live-pct], [data-live-eta], [data-live-fps].
function patch() {
  for (const c of D.conversions) {
    if (c.state.is !== 'running' && c.state.is !== 'suspended') continue;
    const s = c.state;
    root.querySelectorAll(`[data-live-progress="${c.id}"]`).forEach((el) => { el.style.setProperty('--p', s.progress); el.setAttribute('aria-valuenow', Math.floor(s.progress * 100)); });
    root.querySelectorAll(`[data-live-pct="${c.id}"]`).forEach((el) => { el.textContent = `${Math.floor(s.progress * 100)}%`; });
    root.querySelectorAll(`[data-live-eta="${c.id}"]`).forEach((el) => { el.textContent = s.is === 'running' ? fmtEta(s.etaMin) : 'paused'; });
    root.querySelectorAll(`[data-live-fps="${c.id}"]`).forEach((el) => { el.textContent = s.is === 'running' ? `${Math.round(s.fps).toLocaleString('en-US')} fps` : '0 fps'; });
  }
}
function startLive() {
  patch();
  setInterval(() => {
    for (const c of D.conversions) {
      const s = c.state;
      if (s.is !== 'running') continue;
      const step = s.etaMin > 0 ? (1 - s.progress) / (s.etaMin * 60) : 0;
      s.progress = Math.min(0.995, s.progress + step * (0.8 + Math.random() * 0.4));
      s.fps = Math.max(1, s.fps * (0.96 + Math.random() * 0.08));
      s.etaMin = Math.max(0, s.etaMin - 1 / 60);
    }
    patch();
  }, 1000);
}

export function init(ctx) {
  D = ctx.data; root = ctx.root; isPhone = ctx.device === 'phone'; go = ctx.go;
  root.classList.add('lc', `lc-${ctx.device}`, 'cq');
  document.documentElement.classList.add('lc-doc');
  wireShell();
  return { startLive };
}
