// PROTOTYPE. The Lit console admin shell (sidebar, top bar, phone header and tabs, sheets, popovers, toast, ambient tint)
// plus the primitives every variant shares: workRow from Content Type descriptors, Node state, Node link lines.

import { works, now, admin, contentTypes, jobs, ROLES } from './data.js';
import { format } from '../ui-direction/live.js';
import { jobsOn, controllerOf } from './model.js';

export const esc = (s) => String(s ?? '').replace(/[&<>"]/g, (c) => ({ '&': '&amp;', '<': '&lt;', '>': '&gt;', '"': '&quot;' }[c]));
export const I = (n, c = '') => `<i data-lucide="${n}" class="ic ${c}" aria-hidden="true"></i>`;
export const dot = (k) => `<span class="dot dot-${k}" aria-hidden="true"></span>`;
export const hm = (iso) => new Date(iso).toTimeString().slice(0, 5);
export const minsFromNow = (iso) => Math.round((new Date(iso) - now) / 60000);
export const plural = (n, w, ws = `${w}s`) => `${n} ${n === 1 ? w : ws}`;
const initials = (t) => t.replace(/^The /, '').split(/[\s:]+/).filter(Boolean).slice(0, 2).map((s) => s[0]).join('').toUpperCase();

const MARK = `<svg class="mark" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" aria-hidden="true"><path d="M2.5 6.5c6 0 9.5 0 12.5 3.2"/><path d="M2.5 12h9.5"/><path d="M2.5 17.5c6 0 9.5 0 12.5-3.2"/><circle cx="18.6" cy="12" r="2.6" fill="currentColor" stroke="none"/></svg>`;
export const mark = () => MARK;

export function art(w, kind, cls) {
  return `<span class="art ${cls}"><img src="${w.art[kind]}" alt="" decoding="async" onerror="this.parentNode.classList.add('art-x');this.remove()"><span class="art-fb" aria-hidden="true"><b>${esc(initials(w.title))}</b><span>${esc(w.title)}</span></span></span>`;
}

// ── Content Type descriptors drive every Work row. Nothing below names a Content Type.
export function fill(fmt, fields) {
  return fmt.replace(/\{(\w+)(?::(\d))?\}/g, (_, k, pad) => {
    const v = fields[k];
    return v == null ? '' : pad ? String(v).padStart(Number(pad), '0') : String(v);
  }).replace(/^ · | · $/g, '');
}
export function itemLabel(workId, itemKey) {
  const w = works[workId];
  return fill(contentTypes[w.type].item, itemKey || {});
}
const ART_CLASS = { poster: 'thumb', backdrop: 'still' };

export function workRow(j, opts = {}) {
  const w = works[j.work], ct = contentTypes[w.type], L = ct.layout;
  const fields = { work: w.title, item: itemLabel(j.work, j.itemKey), year: w.year };
  const live = j.status === 'running' && j.etaMin != null;
  const failed = j.status === 'failed';
  const held = opts.held;
  const meter = failed ? `<span class="wr-fail">${I('circle-x')}Failed</span>`
    : `<div class="bar wr-bar${held ? ' held' : ''}" ${live && !held ? `role="progressbar" aria-valuemin="0" aria-valuemax="100" aria-label="${esc(w.title)} ${esc(j.stage)} progress" data-live-progress="${j.id}"` : 'role="img" aria-label="Paused"'} style="--p:${j.progress}"><span class="fill"></span></div>`;
  const pct = failed ? '' : `<span class="wr-pct num" ${held ? '' : `data-live-pct="${j.id}"`}>${format.pct(j.progress)}</span>`;
  const rate = held || failed ? '' : j.rate != null ? `<span class="num t2" data-live-rate="${j.id}">${format.rate(j)}</span>` : `<span class="t3">${esc((/(\d+ releases)/.exec(j.release || '') || [])[1] || 'open')}</span>`;
  const eta = held || failed || j.etaMin == null ? '' : `<span class="num t2" data-live-eta="${j.id}">${format.eta(j.etaMin)}</span>`;
  return `<li class="wr${opts.compact ? ' wr-c' : ''}${failed ? ' wr-failed' : ''}" data-work="${w.id}" data-type-art="${L.artwork}">
    ${art(w, L.artwork, ART_CLASS[L.artwork])}
    <span class="wr-id"><b class="ell">${esc(fill(L.title, fields))}</b><span class="t3 ell">${esc(fill(L.sub, fields))}</span></span>
    <span class="wr-st"><span class="wr-stage">${esc(j.stage)}</span><span class="t3"> · ${esc(j.target)}</span></span>
    <span class="wr-m">${meter}${pct}</span>
    <span class="wr-r">${rate}${eta ? `<span class="s-sep">·</span>${eta}` : ''}</span>
    ${opts.note ? `<span class="wr-note ${opts.noteCls || 't3'}">${opts.note}</span>` : ''}
  </li>`;
}

// ── Node state, one table for every variant.
export function nodeState(n) {
  if (n.state === 'offline') return { k: 'fail', label: 'Offline', cls: 'st-fail' };
  if (n.state === 'updating') return { k: 'warn', label: 'Updating', cls: 'st-warn' };
  if (n.state === 'behind') return { k: 'warn', label: `On ${n.version}`, cls: 'st-warn' };
  if (n.state === 'degraded') return { k: 'warn', label: 'Degraded', cls: 'st-warn' };
  const busy = jobsOn(n).some((j) => j.status === 'running');
  return busy ? { k: 'draft', label: 'Working', cls: 'st-live' } : { k: 'idle', label: 'Online', cls: 'st-ok' };
}
export const statePill = (n) => { const s = nodeState(n); return `<span class="npill ${s.cls}">${dot(s.k)}${esc(s.label)}</span>`; };
export const kindChip = (n) => `<span class="chip ${n.kind === 'Controller' ? 'chip-ctl' : ''}">${n.kind === 'Controller' ? I('crown') : ''}${n.kind}</span>`;

export function dialText(n, ctl) {
  if (n.link.dial === 'self') return n.single ? 'Every Role runs in process. No Pairing, no listener besides the UI.' : 'This is the Controller. Workers link to it.';
  if (n.link.dial === 'controller') return `${ctl.name} dials ${n.name}`;
  if (n.link.dial === 'worker') return `${n.name} dials ${ctl.name}`;
  return `Both dial ${n.link.relay}`;
}
export function heartbeatText(n) {
  if (n.state === 'offline') return `Last seen ${hm(n.lastSeen)}, ${-minsFromNow(n.lastSeen)} min ago`;
  if (n.link.heartbeat == null) return '';
  return `Heartbeat ${n.link.heartbeat} s ago`;
}

export function loadLine(n) {
  if (n.state === 'offline') return `<span class="fail-t">No heartbeat since ${hm(n.lastSeen)}</span>`;
  const js = jobsOn(n).filter((j) => j.status === 'running');
  const bits = [`CPU ${n.cpu}%`];
  if (n.gpu) bits.push(`GPU ${n.gpu}%`);
  if (js.length) bits.push(plural(js.length, 'job'));
  if (n.seeding) bits.push(`seeding ${n.seeding.count}`);
  return bits.join(' · ');
}

export function roleChip(n, r) {
  const st = n.roles[r];
  if (st.state === 'draining') return `<span class="chip rchip draining" title="Draining: ${esc(st.left)}">${I('log-out')}${esc(r)}<span class="t3">draining</span></span>`;
  return `<span class="chip rchip${st.handedBack ? ' handed' : ''}" title="${esc(ROLES[r].blurb)}">${esc(r)}</span>`;
}

// ── Shell
const NAV = [
  { id: 'home', label: 'Home', icon: 'house' },
  { id: 'library', label: 'Library', icon: 'library', kids: [{ id: 'movies', label: 'Movies', count: () => 812 }, { id: 'series', label: 'Series', count: () => 96 }] },
  { id: 'activity', label: 'Activity', icon: 'activity', live: () => jobs.filter((j) => j.status === 'running').length },
  { id: 'calendar', label: 'Calendar', icon: 'calendar' },
  { id: 'wanted', label: 'Wanted', icon: 'circle-dashed' },
  { id: 'requests', label: 'Requests', icon: 'inbox', count: () => 2 },
  { id: 'history', label: 'History', icon: 'history' },
  { id: 'manual-import', label: 'Manual import', icon: 'folder-input' },
  { sep: true },
  { id: 'nodes', label: 'Nodes', icon: 'network' },
  { id: 'settings', label: 'Settings', icon: 'settings' },
  { id: 'system', label: 'System', icon: 'gauge', flag: () => 'fail', count: () => 2 },
];

function nodesNav(nodes) {
  const bad = nodes.filter((n) => n.state !== 'online');
  const worst = bad.some((n) => n.state === 'offline') ? 'fail' : bad.length ? 'warn' : '';
  return { count: nodes.length, flag: worst };
}

export function sidebar(nodes) {
  const nn = nodesNav(nodes);
  const item = (n, sub = false) => {
    const on = n.id === 'nodes';
    let right = '';
    if (n.live) right = `<span class="nav-live">${dot('draft')}<span class="num">${n.live()}</span></span>`;
    else if (n.id === 'nodes') right = `<span class="nav-count num">${nn.flag ? dot(nn.flag) : ''}${nn.count}</span>`;
    else if (n.count) right = `<span class="nav-count num">${n.flag ? dot(n.flag()) : ''}${n.count()}</span>`;
    return `<button class="nav-i${sub ? ' sub' : ''}${on ? ' on' : ''}" data-toast="${esc(n.label)} is not in this prototype" ${on ? 'aria-current="page"' : ''}>${sub ? '<span class="nav-ind" aria-hidden="true"></span>' : I(n.icon)}<span class="nav-l">${esc(n.label)}</span>${right}</button>`;
  };
  return `<aside class="side" aria-label="Main">
    <div class="side-top"><span class="brand">${MARK}<span class="brand-w">SlipStream</span></span></div>
    <nav class="nav">${NAV.map((n) => (n.sep ? '<div class="nav-sep" role="separator"></div>' : item(n) + (n.kids ? n.kids.map((k) => item(k, true)).join('') : ''))).join('')}</nav>
    <div class="side-foot"><span class="avatar">${admin.initials}</span><span class="side-who"><b>${admin.name}</b><span>Admin · ${esc(controllerOf(nodes).name)}</span></span></div>
  </aside>`;
}

export function topbar(crumbs) {
  return `<header class="top">
    <div class="crumbs">${crumbs}</div>
    <button class="search-btn" data-act="palette" aria-haspopup="dialog">${I('search')}<span>Search Works or run a command</span><kbd><span>⌘</span>K</kbd></button>
    <div class="top-r">
      <button class="ib bell" data-toast="Notifications are not in this prototype" aria-label="Notifications, 2 unread">${I('bell')}<span class="bell-n num">2</span></button>
      <button class="btn" data-toast="Adding a Work is not in this prototype">${I('plus')}Add Work</button>
    </div>
  </header>`;
}

export const crumbs = (...parts) => parts.map((p, i) => (i === parts.length - 1 ? `<span class="crumb cur">${esc(p.label)}</span>` : `<button class="crumb" data-go='${JSON.stringify(p.go)}'>${esc(p.label)}</button>${I('chevron-right', 'crumb-sep')}`)).join('');

export function wideShell(nodes, crumbHtml, body, aside = '') {
  return `<div class="shell">${sidebar(nodes)}<div class="mainc">${topbar(crumbHtml)}${aside ? `<div class="nd-split"><main class="content">${body}</main>${aside}</div>` : `<main class="content">${body}</main>`}</div></div>`;
}

const TABS = [['home', 'Home', 'house'], ['library', 'Library', 'library'], ['activity', 'Activity', 'activity'], ['search', 'Search', 'search'], ['more', 'More', 'menu']];
export function tabs() {
  return `<nav class="tabs" aria-label="Tabs">${TABS.map(([id, l, ic]) => `<button class="tab${id === 'more' ? ' on' : ''}" ${id === 'search' ? 'data-act="palette"' : id === 'more' ? 'data-act="more"' : `data-toast="${l} is not in this prototype"`} ${id === 'more' ? 'aria-current="page"' : ''}>${I(ic)}<span>${l}</span>${id === 'activity' ? `<span class="tab-b num">${jobs.filter((j) => j.status === 'running').length}</span>` : ''}</button>`).join('')}</nav>`;
}

export function phoneTop(title, right = '') {
  return `<div class="statusbar" aria-hidden="true"></div><header class="ph-head"><h1>${esc(title)}</h1><span class="grow"></span>${right}<button class="ib tap" data-act="palette" aria-label="Search">${I('search')}</button><span class="avatar">${admin.initials}</span></header>`;
}
export function phonePushed(backLabel, backGo, right = '') {
  return `<div class="statusbar" aria-hidden="true"></div><header class="ph-nav"><button class="back tap" data-go='${JSON.stringify(backGo)}'>${I('chevron-left')}${esc(backLabel)}</button><span class="grow"></span>${right}</header><div class="ph-pushed-gap"></div>`;
}

export const AMBIENT = '<div class="amb" aria-hidden="true"><div class="amb-i"></div><div class="amb-i"></div><div class="amb-tint"></div></div>';

export function overlays() {
  return `<div class="nd-sheet-wrap" hidden><div class="sheet-scrim" data-act="close"></div><div class="nd-sheet" role="dialog" aria-modal="true"></div></div>
    <div class="pop nd-pop" hidden></div>
    <div class="pal" hidden><div class="pal-scrim" data-act="close"></div>
      <div class="pal-box" role="dialog" aria-modal="true" aria-label="Search Works and commands">
        <div class="pal-top">${I('search')}<input class="pal-in" type="text" placeholder="Search Nodes, Works or type a command" autocomplete="off" spellcheck="false" aria-controls="pal-list"><kbd>Esc</kbd></div>
        <ul class="pal-list" id="pal-list" role="listbox" aria-label="Results"></ul>
      </div></div>
    <div class="toast" role="status" aria-live="polite"></div>`;
}

// ── Ambient: no Work in focus on the Nodes list, so the page sits on a calm slate tint.
// A Node detail tints from the Work behind its focused job when that artwork loads.
export const NEUTRAL = { h: 214, s: 14 };
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
        if (!ws) return res(null);
        res({ h: ((Math.atan2(y, x) * 180) / Math.PI + 360) % 360, s: Math.round(Math.min(48, Math.max(14, (ss / ws) * 60))) });
      } catch { res(null); }
    };
    im.onerror = () => res(null);
    im.src = src;
  });
  tintCache.set(src, p);
  return p;
}

let ambKey = null;
export function setAmbient(root, workId) {
  if (ambKey === (workId || '')) return;
  ambKey = workId || '';
  const layer = root.querySelector('.amb-i');
  const setTint = (t) => { root.style.setProperty('--h', t.h.toFixed(1)); root.style.setProperty('--s', `${t.s}%`); };
  if (!workId) { if (layer) layer.classList.remove('on'); return setTint(NEUTRAL); }
  setTint(NEUTRAL);
  const w = works[workId];
  const pre = new Image();
  pre.onload = () => { if (ambKey !== workId || !layer) return; layer.style.backgroundImage = `url('${w.art.backdrop}')`; layer.classList.add('on'); };
  pre.src = w.art.backdrop;
  sampleTint(w.art.poster).then((t) => { if (t && ambKey === workId) setTint(t); });
}
export const resetAmbient = () => { ambKey = null; };
