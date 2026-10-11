// PROTOTYPE · Direction C's Lit console shell, cut down from prototypes/torrent-client/shell.js. Hugeicons (ADR 0021) instead of lucide. Throwaway.

let ICONS = {};
export async function loadIcons() {
  ICONS = await import('https://cdn.jsdelivr.net/npm/@hugeicons/core-free-icons@4.3.5/+esm');
}
const kebab = (k) => k.replace(/[A-Z]/g, (c) => `-${c.toLowerCase()}`);
export function I(name, cls = '') {
  const node = ICONS[`${name}Icon`];
  if (!node) { console.warn('missing icon', name); return ''; }
  const kids = node.map(([tag, attrs]) => `<${tag} ${Object.entries(attrs).filter(([k]) => k !== 'key').map(([k, v]) => `${kebab(k)}="${v}"`).join(' ')}/>`).join('');
  return `<svg class="ic ${cls}" viewBox="0 0 24 24" fill="none" aria-hidden="true">${kids}</svg>`;
}

export const esc = (s) => String(s ?? '').replace(/[&<>"]/g, (c) => ({ '&': '&amp;', '<': '&lt;', '>': '&gt;', '"': '&quot;' }[c]));
const initials = (t) => t.replace(/^The /, '').split(/[\s:]+/).filter(Boolean).slice(0, 2).map((s) => s[0]).join('').toUpperCase();

export function art(w, cls = 'thumb', kind = 'poster') {
  return `<span class="art ${cls}"><img src="${w.art[kind]}" alt="" decoding="async" onerror="this.parentNode.classList.add('art-x');this.remove()"><span class="art-fb" aria-hidden="true"><b>${esc(initials(w.title))}</b><span>${esc(w.title)}</span></span></span>`;
}

const MARK = `<svg class="mark" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" aria-hidden="true"><path d="M2.5 6.5c6 0 9.5 0 12.5 3.2"/><path d="M2.5 12h9.5"/><path d="M2.5 17.5c6 0 9.5 0 12.5-3.2"/><circle cx="18.6" cy="12" r="2.6" fill="currentColor" stroke="none"/></svg>`;

const NAV = [
  { id: 'home', label: 'Home', icon: 'Home03' },
  { id: 'library', label: 'Library', icon: 'Library', kids: [{ id: 'movies', label: 'Movies', count: 412 }, { id: 'series', label: 'Series', count: 96 }] },
  { id: 'activity', label: 'Activity', icon: 'Activity01', live: 4 },
  { id: 'torrents', label: 'Torrents', icon: 'Magnet', count: 612 },
  { id: 'calendar', label: 'Calendar', icon: 'Calendar03' },
  { id: 'wanted', label: 'Wanted', icon: 'Target02', count: 38 },
  { id: 'requests', label: 'Requests', icon: 'Inbox', count: 3 },
  { id: 'history', label: 'History', icon: 'Clock01' },
  { sep: true },
  { id: 'nodes', label: 'Nodes', icon: 'Hierarchy', count: 4 },
  { id: 'settings', label: 'Settings', icon: 'Settings01' },
  { id: 'system', label: 'System', icon: 'DashboardSpeed01', count: 2, flag: 'warn' },
];

const dot = (k) => `<span class="dot dot-${k}" aria-hidden="true"></span>`;

function sidebar(active, admin) {
  const item = (n, sub = false) => {
    const on = n.id === active;
    const right = n.live ? `<span class="nav-live">${dot('draft')}<span class="num">${n.live}</span></span>`
      : n.count ? `<span class="nav-count num">${n.flag ? dot(n.flag) : ''}${n.count}</span>` : '';
    return `<button class="nav-i${sub ? ' sub' : ''}${on ? ' on' : ''}" data-toast="${esc(n.label)} is not in this prototype" ${on ? 'aria-current="page"' : ''}>${sub ? '' : I(n.icon)}<span class="nav-l">${esc(n.label)}</span>${right}</button>`;
  };
  return `<aside class="side" aria-label="Main">
    <div class="side-top"><span class="brand">${MARK}<span class="brand-w">SlipStream</span></span></div>
    <nav class="nav">${NAV.map((n) => (n.sep ? '<div class="nav-sep" role="separator"></div>' : item(n) + (n.kids ? n.kids.map((k) => item(k, true)).join('') : ''))).join('')}</nav>
    <div class="side-foot"><span class="avatar">${admin.initials}</span><span class="side-who"><b>${admin.name}</b><span>Admin · atlas</span></span></div>
  </aside>`;
}

function topbar(crumbs) {
  return `<header class="top">
    <div class="crumbs">${crumbs}</div>
    <button class="search-btn" data-toast="The palette is not in this prototype">${I('Search01')}<span>Search Works or run a command</span><kbd><span>⌘</span>K</kbd></button>
    <div class="top-r">
      <span class="rates num">${I('ArrowDown01')}41.2 MB/s<span class="rates-sep"></span>${I('ArrowUp01')}6.8 MB/s</span>
      <button class="ib bell" data-toast="Notifications are not in this prototype" aria-label="Notifications">${I('Notification01')}<span class="bell-n num">2</span></button>
      <button class="btn" data-toast="Adding a Work is not in this prototype">${I('PlusSign')}Add Work</button>
    </div>
  </header>`;
}

const TABS = [
  { id: 'home', label: 'Home', icon: 'Home03' },
  { id: 'library', label: 'Library', icon: 'Library' },
  { id: 'activity', label: 'Activity', icon: 'Activity01', badge: 4 },
  { id: 'search', label: 'Search', icon: 'Search01' },
  { id: 'more', label: 'More', icon: 'Menu01' },
];
const tabs = (active) => `<nav class="tabs" aria-label="Tabs">${TABS.map((t) => `<button class="tab${t.id === active ? ' on' : ''}" data-toast="${t.label} is not in this prototype">${I(t.icon)}<span>${t.label}</span>${t.badge ? `<span class="tab-b num">${t.badge}</span>` : ''}</button>`).join('')}</nav>`;

let root, isPhone, lastFocus = null;

export function closeMenus() {
  root.querySelectorAll('.menu, .sheet-wrap').forEach((el) => { el.hidden = true; });
  if (lastFocus && document.contains(lastFocus)) lastFocus.focus({ preventScroll: true });
  lastFocus = null;
}
export const menuOpen = () => [...root.querySelectorAll('.menu, .sheet-wrap')].some((el) => !el.hidden);

let toastT;
export function toast(msg) {
  const t = root.querySelector('.toast');
  t.innerHTML = `${I('InformationCircle')}<span>${esc(msg)}</span>`;
  t.classList.add('on');
  clearTimeout(toastT);
  toastT = setTimeout(() => t.classList.remove('on'), 2800);
}

// items: [{ label, icon, run, danger, sub, disabled, sep }]. Desk: anchored menu. Phone: action sheet.
let menuItems = [];
export function openMenu(anchor, title, items, sub = '') {
  closeMenus();
  lastFocus = anchor;
  menuItems = typeof items === 'string' ? [] : items;
  const html = typeof items === 'string' ? items : items.map((x, i) => (x.sep ? '<div class="mi-sep" role="separator"></div>'
    : x.head ? `<p class="mi-h">${esc(x.head)}</p>`
      : `<button class="mi${x.danger ? ' danger' : ''}" role="menuitem" data-mi="${i}" ${x.disabled ? 'disabled' : ''}>${x.icon ? I(x.icon) : ''}<span class="mi-l"><span>${esc(x.label)}</span>${x.sub ? `<span class="mi-s">${esc(x.sub)}</span>` : ''}</span>${x.right ? `<span class="mi-r t3">${x.right}</span>` : ''}</button>`)).join('');
  if (isPhone) {
    const s = root.querySelector('.sheet-wrap');
    s.querySelector('.sheet').innerHTML = `<span class="grab" aria-hidden="true"></span><p class="sheet-t"><b>${esc(title)}</b>${sub ? `<span class="t3 sheet-sub">${esc(sub)}</span>` : ''}</p><div class="sheet-grp" ${typeof items === 'string' ? '' : 'role="menu"'} aria-label="${esc(title)}">${html}</div><button class="sheet-x" data-close-menu>${typeof items === 'string' ? 'Done' : 'Cancel'}</button>`;
    s.hidden = false;
    const f = s.querySelector('.mi:not([disabled])');
    if (f) f.focus({ preventScroll: true });
    return;
  }
  const m = root.querySelector('.menu');
  m.classList.toggle('menu-pop', typeof items === 'string');
  m.innerHTML = (title ? `<p class="mi-t">${esc(title)}${sub ? `<span class="t3">${esc(sub)}</span>` : ''}</p>` : '') + html;
  m.hidden = false;
  const r = anchor.getBoundingClientRect();
  const vw = innerWidth, vh = innerHeight;
  let top = r.bottom + 6;
  if (top + m.offsetHeight > vh - 12) top = Math.max(12, r.top - 6 - m.offsetHeight);
  m.style.top = `${top}px`;
  m.style.left = `${Math.max(12, Math.min(r.left, vw - m.offsetWidth - 12))}px`;
  const f = m.querySelector('.mi:not([disabled])');
  if (f) f.focus({ preventScroll: true });
}

const AMBIENT = '<div class="amb" aria-hidden="true"><div class="amb-i on"></div><div class="amb-i"></div><div class="amb-tint"></div></div>';

function sampleTint(src) {
  return new Promise((res) => {
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
}

// screen: { device, admin, work, crumbs, body, tab }
export function mount(el, screen) {
  root = el;
  isPhone = screen.device === 'phone';
  root.classList.add('lc', `lc-${screen.device}`, 'lc-work');
  document.documentElement.classList.add('lc-doc');
  const body = isPhone
    ? `<div class="statusbar" aria-hidden="true"></div>${screen.body}${tabs('library')}`
    : `<div class="shell">${sidebar(screen.work.kind === 'movie' ? 'movies' : 'series', screen.admin)}<div class="mainc">${topbar(screen.crumbs)}<main class="content">${screen.body}</main></div></div>`;
  root.innerHTML = `${AMBIENT}${body}<div class="ms-host"></div><div class="menu" role="menu" hidden></div><div class="sheet-wrap" hidden><div class="sheet-scrim" data-close-menu></div><div class="sheet"></div></div><div class="toast" role="status" aria-live="polite"></div>`;
  root.querySelector('.amb-i').style.backgroundImage = `url('${screen.work.art.backdrop}')`;
  sampleTint(screen.work.art.poster).then((t) => {
    if (!t) return;
    root.style.setProperty('--h', t.h.toFixed(1));
    root.style.setProperty('--s', `${t.s}%`);
  });
  root.addEventListener('click', (e) => {
    const q = (s) => e.target.closest(s);
    let el2;
    if (q('[data-close-menu]')) return closeMenus();
    if ((el2 = q('.mi[data-mi]'))) { const x = menuItems[+el2.dataset.mi]; closeMenus(); if (x && x.run) x.run(); return; }
    if (!q('.menu') && !root.querySelector('.menu').hidden) closeMenus();
    if ((el2 = q('[data-toast]'))) return toast(el2.dataset.toast);
  });
  document.addEventListener('keydown', (e) => {
    const menu = e.target.closest && e.target.closest('.menu, .sheet-grp');
    if (menu && (e.key === 'ArrowDown' || e.key === 'ArrowUp')) {
      e.preventDefault();
      const items = [...menu.querySelectorAll('.mi:not([disabled])')];
      const i = items.indexOf(document.activeElement);
      items[(i + (e.key === 'ArrowDown' ? 1 : -1) + items.length) % items.length].focus();
    }
  });
  return root.querySelector('.ms-host');
}
