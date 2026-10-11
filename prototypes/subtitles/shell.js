// PROTOTYPE shell. Lit console's admin chrome (sidebar, top bar, ambient tint, toast), lifted from
// ../ui-direction/variants/lit-console/variant.js so every variant here sits inside the settled direction.
// Variants own everything inside <main>, plus any modal, sheet or popover they open.

export const esc = (s) => String(s ?? '').replace(/[&<>"]/g, (c) => ({ '&': '&amp;', '<': '&lt;', '>': '&gt;', '"': '&quot;' }[c]));
export const I = (n, c = '') => `<i data-lucide="${n}" class="ic ${c}" aria-hidden="true"></i>`;
export const dot = (k) => `<span class="dot dot-${k}" aria-hidden="true"></span>`;
export const fmt = (n) => Number(n).toLocaleString('en-US');

const initials = (t) => t.replace(/^The /, '').split(/[\s:]+/).filter(Boolean).slice(0, 2).map((s) => s[0]).join('').toUpperCase();
export function art(w, cls = 'thumb', kind = 'poster') {
  return `<span class="art ${cls}"><img src="${w.art[kind]}" alt="" decoding="async" onerror="this.parentNode.classList.add('art-x');this.remove()"><span class="art-fb" aria-hidden="true"><b>${esc(initials(w.title))}</b><span>${esc(w.title)}</span></span></span>`;
}

const MARK = `<svg class="mark" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" aria-hidden="true"><path d="M2.5 6.5c6 0 9.5 0 12.5 3.2"/><path d="M2.5 12h9.5"/><path d="M2.5 17.5c6 0 9.5 0 12.5-3.2"/><circle cx="18.6" cy="12" r="2.6" fill="currentColor" stroke="none"/></svg>`;

const NAV = [
  { id: 'home', label: 'Home', icon: 'house' },
  { id: 'library', label: 'Library', icon: 'library', kids: [{ id: 'movies', label: 'Movies', count: 812 }, { id: 'series', label: 'Series', count: 96 }] },
  { id: 'activity', label: 'Activity', icon: 'activity' },
  { id: 'calendar', label: 'Calendar', icon: 'calendar' },
  { id: 'wanted', label: 'Wanted', icon: 'circle-dashed' },
  { id: 'requests', label: 'Requests', icon: 'inbox', count: 2 },
  { id: 'history', label: 'History', icon: 'history' },
  { sep: true },
  { id: 'nodes', label: 'Nodes', icon: 'network', count: 4 },
  { id: 'settings', label: 'Settings', icon: 'settings', kids: [{ id: 'profiles', label: 'Profiles' }, { id: 'subtitles', label: 'Subtitles' }, { id: 'indexers', label: 'Indexers' }, { id: 'media-servers', label: 'Media Servers' }] },
  { id: 'system', label: 'System', icon: 'gauge' },
];

// `active` is a NAV id. Settings' children only show while a settings page is open, like a real nav would.
export function sidebar(active, extra = {}) {
  const inSettings = ['settings', 'profiles', 'subtitles', 'indexers', 'media-servers'].includes(active);
  const item = (n, sub = false) => {
    const on = n.id === active;
    const c = extra[n.id] ?? n.count;
    const right = c != null && c !== '' ? `<span class="nav-count num">${c}</span>` : '';
    return `<button class="nav-i${sub ? ' sub' : ''}${on ? ' on' : ''}" data-nav="${n.id}" ${on ? 'aria-current="page"' : ''} title="${esc(n.label)}">${sub ? '<span class="nav-ind" aria-hidden="true"></span>' : I(n.icon)}<span class="nav-l">${esc(n.label)}</span>${right}</button>`;
  };
  const kids = (n) => (n.kids && (n.id !== 'settings' || inSettings) ? n.kids.map((k) => item(k, true)).join('') : '');
  return `<aside class="side" aria-label="Main">
    <div class="side-top"><span class="brand">${MARK}<span class="brand-w">SlipStream</span></span></div>
    <nav class="nav">${NAV.map((n) => (n.sep ? '<div class="nav-sep" role="separator"></div>' : item(n) + kids(n))).join('')}</nav>
    <div class="side-foot"><span class="avatar">JA</span><span class="side-who"><b>Jackson</b><span>Admin · atlas</span></span></div>
  </aside>`;
}

// crumbs: array of labels, last one is the current page.
export function topbar(crumbs, right = '') {
  const c = crumbs.map((l, i) => (i === crumbs.length - 1 ? `<span class="crumb cur">${esc(l)}</span>` : `<span class="crumb">${esc(l)}</span>${I('chevron-right', 'crumb-sep')}`)).join('');
  return `<header class="top">
    <div class="crumbs">${c}</div>
    <button class="search-btn" data-toast="The palette is not in this prototype">${I('search')}<span>Search Works or run a command</span><kbd><span>⌘</span>K</kbd></button>
    <div class="top-r">${right}</div>
  </header>`;
}

// Wraps a variant's page in the admin chrome. On phone it drops the sidebar and keeps a compact header.
export function page({ active, crumbs, body, phone, phoneTitle, back }) {
  if (phone) {
    return `<div class="statusbar" aria-hidden="true"></div><header class="ph-head">${back ? `<button class="ib tap" data-toast="Back is not in this prototype" aria-label="Back">${I('chevron-left')}</button>` : ''}<h1 class="grow">${esc(phoneTitle || crumbs[crumbs.length - 1])}</h1><span class="avatar">JA</span></header><main class="ph-main">${body}</main>`;
  }
  return `<div class="shell">${sidebar(active)}<div class="mainc">${topbar(crumbs)}<main class="content">${body}</main></div></div>`;
}

export const AMBIENT = '<div class="amb" aria-hidden="true"><div class="amb-i"></div><div class="amb-i"></div><div class="amb-tint"></div></div>';
export const OVERLAYS = '<div class="toast" role="status" aria-live="polite"></div>';

let toastT;
export function toast(root, msg) {
  const t = root.querySelector('.toast');
  t.innerHTML = `${I('info')}<span>${esc(msg)}</span>`;
  window.lucide && window.lucide.createIcons();
  t.classList.add('on');
  clearTimeout(toastT);
  toastT = setTimeout(() => t.classList.remove('on'), 2800);
}

// The page takes its colour from the Work in focus (Lit console). Settings pages have no Work and stay neutral.
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

export function lightBy(root, w) {
  const layer = root.querySelector('.amb-i');
  const pre = new Image();
  pre.onload = () => { layer.style.backgroundImage = `url('${w.art.backdrop}')`; layer.classList.add('on'); };
  pre.src = w.art.backdrop;
  sampleTint(w.art.poster).then((t) => {
    if (!t) return;
    root.style.setProperty('--h', t.h.toFixed(1));
    root.style.setProperty('--s', `${t.s}%`);
  });
}
