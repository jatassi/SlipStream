// PROTOTYPE · Settings > Download Clients and Seeding goals on Direction C's shell. Throwaway.

import { mount, h, openSheet, closeAll, toast } from './shell.js';

const { esc, I, art } = h;
let D, ctx, root, phone;

const MARK = `<svg class="mark" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" aria-hidden="true"><path d="M2.5 6.5c6 0 9.5 0 12.5 3.2"/><path d="M2.5 12h9.5"/><path d="M2.5 17.5c6 0 9.5 0 12.5-3.2"/><circle cx="18.6" cy="12" r="2.6" fill="currentColor" stroke="none"/></svg>`;
const dot = (k) => `<span class="dot dot-${k}" aria-hidden="true"></span>`;
const hm = (iso) => new Date(iso).toTimeString().slice(0, 5);
const ago = (iso) => {
  const m = Math.round((D.now - new Date(iso)) / 60000);
  if (m < 60) return `${m} min ago`;
  const hh = Math.floor(m / 60);
  return hh < 24 ? `${hh} h ago` : `${Math.floor(hh / 24)} days ago`;
};
const hrs = (x) => (x % 24 === 0 && x >= 168 ? `${x / 24} days` : `${+x.toFixed(1)} h`);
const r1 = (x) => Number(x).toFixed(1);
const gh = (x) => (x % 24 === 0 && x >= 24 ? (x === 24 ? '1 day' : `${x / 24} days`) : `${+x.toFixed(1)} h`);
const plural = (n, one, many) => `${n} ${n === 1 ? one : many}`;

// ───────────────────────────── extra fixture: what v1 kept per client (P-142…P-147), invented values

const EXT = {
  'qbit-ams': { hostName: '127.0.0.1', port: 8080, ssl: false, urlBase: '', user: 'slipstream', priority: 1, delay: 0, enabled: true, check: { ok: true, at: '2026-10-08T21:35:00', ms: 3 } },
  'tr-atlas': { hostName: '192.168.1.20', port: 9091, ssl: false, urlBase: '/transmission/', user: 'jack', priority: 2, delay: 5, enabled: true, check: { ok: false, at: '2026-10-08T21:35:00' } },
};
const BI = {
  'builtin-ams': { checkedAt: '2026-10-08T21:30:00' },
  'builtin-atlas': { checkedAt: '2026-10-08T21:30:00', fix: 'Forward TCP and UDP 6881 on your router to atlas at 192.168.1.10, or turn on UPnP there.' },
};
const KIND = {
  qbittorrent: { short: 'qB', port: 8080, urlBase: '', version: '4.6.7' },
  transmission: { short: 'Tr', port: 9091, urlBase: '/transmission/', version: '4.0.6' },
  deluge: { short: 'De', port: 8112, urlBase: '', version: '2.1.1' },
  rtorrent: { short: 'rT', port: 8000, urlBase: '/RPC2', version: '0.9.8' },
};
const WHEN_SHORT = { remove: 'Remove', stop: 'Stop seeding', keep: 'Keep seeding' };
const WHEN_ICON = { remove: 'trash-2', stop: 'circle-pause', keep: 'infinity' };
const WHEN_THEN = { remove: 'remove the torrent and its download copy', stop: 'stop seeding and keep the torrent', keep: 'keep seeding' };
const MODES = [['either', 'Whichever first'], ['both', 'Both'], ['forever', 'Forever']];

let CL, routes, lim, tests;
let G, units, baseline, lastImpact;
let F = null;
let menuFrom = null;

function initState() {
  CL = D.clients.map((c) => ({ ...c, ...(EXT[c.id] || {}), ...(BI[c.id] || {}) }));
  routes = Object.fromEntries(D.trackers.map((t) => [t.id, t.route]));
  lim = Object.fromEntries(CL.filter((c) => c.kind === 'built-in').map((c) => [c.id, { ...c.limits, port: c.listen.port }]));
  tests = {};
  const d = D.seedingGoals.default;
  G = {
    def: { ...d },
    per: {
      beacon: { override: false, ...d },
      cinder: { override: true, ...D.seedingGoals.perTracker.cinder },
      public: { override: true, ...D.seedingGoals.perTracker.public },
    },
    disk: { minFreePct: D.seedingGoals.diskGuard.minFreePct, mode: 'pause' },
  };
  units = { def: 'd', beacon: 'd', cinder: 'd', public: 'd' };
  baseline = JSON.stringify(G);
}

const client = (id) => CL.find((c) => c.id === id);
const tracker = (id) => D.trackers.find((t) => t.id === id);
const nodeOf = (id) => D.nodes.find((n) => n.id === id);
const kindName = (k) => D.externalKinds.find((x) => x.kind === k)?.name || k;
const freePct = (n) => Math.round((n.disk.freeTB / n.disk.totalTB) * 100);
const isDown = (c) => c.status === 'unreachable';

// ───────────────────────────── small controls

function seg(label, opts, cur, attrs, { disabled = false, cls = '' } = {}) {
  return `<div class="seg-c ${cls}" role="radiogroup" aria-label="${esc(label)}"${disabled ? ' aria-disabled="true"' : ''}>${opts.map(([v, l]) => `<button type="button" role="radio" class="seg-o" aria-checked="${v === cur}" tabindex="${v === cur ? 0 : -1}" ${disabled ? 'disabled' : ''} ${attrs(v)}>${l}</button>`).join('')}</div>`;
}
const swBtn = (on, attrs, label) => `<button type="button" class="btn tog" role="switch" aria-checked="${on}" ${attrs}><span>${esc(label)}</span><span class="sw" aria-hidden="true"><span></span></span></button>`;
const swBare = (on, attrs, label) => `<button type="button" class="swb" role="switch" aria-checked="${on}" aria-label="${esc(label)}" ${attrs}><span class="sw" aria-hidden="true"><span></span></span></button>`;
const unitInput = (id, value, unit, attrs, ph = 'Unlimited') => `<span class="inp-u"><input class="inp num" id="${id}" type="text" inputmode="decimal" autocomplete="off" value="${value ?? ''}" placeholder="${ph}" ${attrs}><span class="u" aria-hidden="true">${unit}</span></span>`;

function tres(t) {
  if (!t) return '';
  if (t.state === 'busy') return `<span class="tres busy">${I('loader-circle', 'spin')}<span>${esc(t.msg || 'Testing…')}</span></span>`;
  if (t.state === 'ok') return `<span class="tres ok">${I('circle-check')}<span>${esc(t.msg)}</span></span>`;
  if (t.state === 'warn') return `<span class="tres warn">${I('triangle-alert')}<span>${esc(t.msg)}</span></span>`;
  return `<span class="tres bad">${I('circle-x')}<span>${esc(t.msg)}</span></span>`;
}

// ═════════════════════════════ Download Clients

function summary() {
  const down = CL.filter(isDown).length;
  const closed = CL.filter((c) => c.kind === 'built-in' && !c.listen.reachable);
  const dn = CL.reduce((n, c) => n + (isDown(c) ? 0 : c.totals.downRate), 0);
  const up = CL.reduce((n, c) => n + (isDown(c) ? 0 : c.totals.upRate), 0);
  const parts = [`<span>${plural(CL.length, 'Download Client', 'Download Clients')} on ${plural(D.downloadingNodes().length, 'Node', 'Nodes')}</span>`];
  if (down) parts.push(`<span class="s-fail">${down} unreachable</span>`);
  closed.forEach((c) => parts.push(`<span class="s-warn">${esc(c.node)} port closed</span>`));
  parts.push(`<span class="s-live">${dot('draft')}<span class="num">↓ ${r1(dn)} ↑ ${r1(up)} MB/s</span></span>`);
  return parts.join('<span class="s-sep">·</span>');
}

function builtinWide(c) {
  const n = nodeOf(c.node), L = lim[c.id], t = c.totals, closed = !c.listen.reachable;
  const reach = closed
    ? `<div class="warnbox" data-reach="${c.id}"><span class="chip-warn">${I('triangle-alert')}Port closed from outside</span><p>${esc(c.listen.note)}</p><p class="warn-dim">${esc(c.fix)}</p><div class="warnbox-a"><button class="btn warn sm" data-recheck="${c.id}">Check again</button><span data-tres="port-${c.id}">${tres(tests['port-' + c.id])}</span></div></div>`
    : `<p class="reach" data-reach="${c.id}">${I('circle-check')}<span>Reachable from outside <span class="t3">· checked ${hm(c.checkedAt)}</span></span><button class="btn ghost sm" data-recheck="${c.id}">Check again</button></p><span data-tres="port-${c.id}">${tres(tests['port-' + c.id])}</span>`;
  const f = (k, label, unit, ph) => `<div class="fld"><label for="${c.id}-${k}">${label}</label>${unitInput(`${c.id}-${k}`, L[k], unit, `data-lim="${c.id}" data-f="${k}"`, ph)}</div>`;
  return `<article class="cc cc-bi" aria-label="Built-in client on ${esc(c.node)}">
    <header class="cc-h">
      <span class="cc-logo bi">${MARK}</span>
      <div class="cc-id"><h3>Built-in <span class="t3 num">${esc(c.version)}</span></h3><p class="t3">Peer ID <span class="mono t2">${esc(c.peerId)}</span><span class="s-sep">·</span>Seeds by <button class="lnk" data-go="goals"><b>Seeding goals</b></button></p></div>
      <span class="cc-rate num">${t.downRate + t.upRate > 0 ? dot('draft') : dot('idle')}↓ ${r1(t.downRate)} <span class="t3">·</span> ↑ ${r1(t.upRate)} MB/s</span>
      <button class="btn sm" data-go="torrents">${I('magnet')}Open torrents</button>
    </header>
    <div class="cc-grid">
      <section class="cc-col" aria-label="Connection"><h4>Connection</h4>
        <div class="fld"><label for="${c.id}-port">Listen port</label><input class="inp num w-port" id="${c.id}-port" type="text" inputmode="numeric" value="${L.port}" data-lim="${c.id}" data-f="port"></div>
        ${reach}
      </section>
      <section class="cc-col" aria-label="Limits"><h4>Limits <span class="saved" data-saved="${c.id}" aria-live="polite"></span></h4>
        <div class="f2">${f('downMBs', 'Download speed', 'MB/s')}${f('upMBs', 'Upload speed', 'MB/s')}${f('activeDownloads', 'Download slots', 'at once')}${f('activeSeeds', 'Seed slots', 'at once')}</div>
      </section>
      <section class="cc-col" aria-label="Totals"><h4>Totals</h4>
        <dl class="kv3 kv-t">
          <div><dt>Torrents</dt><dd class="num">${t.torrents}</dd></div>
          <div><dt>Seeding</dt><dd class="num">${t.seeding}</dd></div>
          <div><dt>Downloading</dt><dd class="num">${t.downloading}<span class="t3"> +${t.queued} queued</span></dd></div>
          <div><dt>Ratio</dt><dd class="num">${r1(t.ratio)}</dd></div>
          <div><dt>Uploaded</dt><dd class="num">${t.uploadedTB} TB</dd></div>
          <div><dt>Problems</dt><dd class="num${t.problems ? ' warn-t' : ''}">${t.problems}</dd></div>
        </dl>
      </section>
    </div>
    <footer class="cc-f t3">${I('lock')}<span>Every downloading Node has one built-in client. It goes only when ${esc(n.name)} leaves the Downloading Role.</span><span class="grow"></span>${I('folder')}<span class="mono">${esc(n.disk.path)}/SlipStream/&lt;Content Type&gt;</span></footer>
  </article>`;
}

const whyFor = (c) => c.why && D.trackers.some((t) => t.status === 'not-allowed' && routes[t.id] === c.id);

function routedHere(c) {
  const ts = D.trackers.filter((t) => routes[t.id] === c.id);
  return `<span class="rh" data-rh="${c.id}">${ts.length ? ts.map((t) => `<span class="chip">${I('route')}${esc(t.name)}</span>`).join('') : '<span class="t3">No tracker routes here</span>'}</span>`;
}

function extKv(c) {
  const ck = c.check;
  const health = ck ? (ck.ok ? `<span class="hc ok">${I('circle-check')}OK <span class="t3 num">· ${hm(ck.at)} · ${ck.ms} ms</span></span>` : `<span class="hc bad">${I('circle-x')}Failed <span class="num">· ${hm(ck.at)}</span></span>`) : '<span class="t3">Not checked yet</span>';
  return `<dl class="cc-kv">
    <div><dt>Category</dt><dd><span class="mono">${esc(c.category)}</span><span class="t3"> /&lt;Content Type&gt;</span></dd></div>
    <div><dt>Priority</dt><dd class="num">${c.priority}</dd></div>
    <div><dt>Import delay</dt><dd class="num">${c.delay ? `${c.delay} min` : 'None'}</dd></div>
    <div><dt>Health check</dt><dd>${health}<span class="t3"> · every 5 min</span></dd></div>
    <div><dt>Routes here</dt><dd>${routedHere(c)}</dd></div>
  </dl>`;
}

function externalWide(c) {
  const t = c.totals, off = !c.enabled;
  if (isDown(c)) {
    return `<article class="cc cc-ext cc-down" data-cc="${c.id}" aria-label="${esc(c.name)} on ${esc(c.node)}, unreachable">
      <header class="cc-h">
        <span class="cc-logo">${KIND[c.kind].short}</span>
        <div class="cc-id"><h3>${esc(c.name)} <span class="fail-dim num">${esc(c.version.replace(c.name + ' ', ''))}</span></h3><p class="mono fail-dim">${esc(c.host)}</p></div>
        <span class="chip-fail">${I('plug-zap')}Unreachable</span>
        <span class="grow"></span>
        <button class="btn fail sm" data-test="${c.id}">${I('refresh-cw')}Test connection</button>
        <button class="btn sm" data-edit="${c.id}">${I('pencil')}Edit</button>
        <button class="ib" data-menu="ext:${c.id}" aria-haspopup="menu" aria-label="More for ${esc(c.name)}">${I('ellipsis')}</button>
      </header>
      <dl class="fail-kv cc-err">
        <dt>Error</dt><dd class="mono">${esc(c.error)}</dd>
        <dt>Last seen</dt><dd><span class="num">${hm(c.lastSeen)}</span> <span class="fail-dim">· ${ago(c.lastSeen)}</span> <span class="chip chip-onfail">${I('history')}Showing last known data</span></dd>
      </dl>
      <p class="cc-test" data-tres="${c.id}">${tres(tests[c.id])}</p>
      ${extKv(c)}
      <footer class="cc-f fail-dim"><span class="num">${t.torrents} torrents · ${t.seeding} seeding · ratio ${r1(t.ratio)} · ${t.uploadedTB} TB uploaded</span><span class="grow"></span><span>Keeps its own seeding rules</span></footer>
    </article>`;
  }
  return `<article class="cc cc-ext${off ? ' off' : ''}" data-cc="${c.id}" aria-label="${esc(c.name)} on ${esc(c.node)}">
    <header class="cc-h">
      <span class="cc-logo">${KIND[c.kind].short}</span>
      <div class="cc-id"><h3>${esc(c.name)} <span class="t3 num">${esc(c.version.replace(c.name + ' ', ''))}</span></h3><p class="mono t3">${esc(c.host)}</p></div>
      <span class="cc-rate num">${off ? '' : `${t.downRate + t.upRate > 0 ? dot('draft') : dot('idle')}↓ ${r1(t.downRate)} <span class="t3">·</span> ↑ ${r1(t.upRate)} MB/s`}</span>
      <span class="grow"></span>
      ${swBtn(!off, `data-enable="${c.id}" aria-label="${esc(c.name)} enabled"`, off ? 'Off' : 'Enabled')}
      <button class="ib" data-menu="ext:${c.id}" aria-haspopup="menu" aria-label="More for ${esc(c.name)}">${I('ellipsis')}</button>
    </header>
    ${whyFor(c) ? `<p class="cc-why">${I('info')}<span>${esc(c.why)}</span></p>` : ''}
    ${extKv(c)}
    <footer class="cc-f t3"><span class="num">${t.torrents} torrents · ${t.seeding} seeding · ratio ${r1(t.ratio)} · ${t.uploadedTB} TB uploaded</span><span class="sep-dot">·</span><span>Keeps its own seeding rules</span><span class="grow"></span><span data-tres="${c.id}">${tres(tests[c.id])}</span><button class="btn sm" data-test="${c.id}">Test connection</button><button class="btn sm" data-edit="${c.id}">${I('pencil')}Edit</button></footer>
  </article>`;
}

function nodeHead(n) {
  const p = freePct(n);
  return `<header class="ng-h">
    <span class="ng-ic">${I('server')}</span>
    <div class="ng-id"><h2 class="mono" id="ng-${n.id}">${esc(n.name)}</h2><p class="t3">${n.kind} · ${esc(n.where)}</p></div>
    <span class="chip node">${I('download')}Downloading</span>
    <span class="grow"></span>
    <div class="ng-disk"><p><span class="mono t2">${esc(n.disk.path)}</span></p><div class="meter" aria-hidden="true"><span style="width:${100 - p}%"></span></div><p class="t3 num">${n.disk.freeTB} TB free of ${n.disk.totalTB} TB · ${p}%</p></div>
  </header>`;
}

function nodeGroupWide(n) {
  const mine = CL.filter((c) => c.node === n.id);
  const bi = mine.find((c) => c.kind === 'built-in');
  const ext = mine.filter((c) => c.kind !== 'built-in');
  return `<section class="ng" data-ng="${n.id}" aria-labelledby="ng-${n.id}">
    ${nodeHead(n)}
    <div class="ng-b">
      ${builtinWide(bi)}
      ${ext.length ? `<div class="ext-h"><h3>External on ${esc(n.name)}</h3><p class="t3">${esc(n.name)} talks to ${ext.length === 1 ? 'it' : 'them'} and reads ${ext.length === 1 ? 'its' : 'their'} files.</p></div>${ext.map(externalWide).join('')}` : ''}
    </div>
  </section>`;
}

// ───────────────────────────── Tracker routing

function routeNotes(t) {
  const c = client(routes[t.id]);
  const notes = [];
  if (t.status === 'not-allowed' && c.kind === 'built-in') notes.push(['warn', `${t.name} rejects ${c.peerId} today, so its grabs will fail until staff add it to the allow list.`]);
  else if (t.status === 'not-allowed') notes.push(['why', `${t.name}'s allow list doesn't have -SS2000- yet, so its grabs go to ${c.name}. Move it back to the built-in once ${t.name} allows -SS2000-.`]);
  if (isDown(c)) notes.push(['fail', `${c.name} on ${c.node} is unreachable. ${t.name}'s grabs fail until it answers.`]);
  else if (c.enabled === false) notes.push(['warn', `${c.name} is off. ${t.name}'s grabs wait until it's on.`]);
  return notes;
}
const noteHtml = ([k, msg]) => `<p class="rt-n rt-${k}">${I(k === 'why' ? 'info' : k === 'fail' ? 'circle-x' : 'triangle-alert')}<span>${esc(msg)}</span></p>`;

function routeRow(t) {
  const c = client(routes[t.id]);
  const notes = routeNotes(t);
  const pick = `<button class="${phone ? 'rt-pick-p' : 'btn rt-pick'}" data-route="${t.id}" aria-haspopup="${phone ? 'dialog' : 'menu'}" aria-label="${esc(t.name)} grabs go to ${esc(c.name)} on ${esc(c.node)}. Change">${phone ? '' : `<span class="rt-pl">${c.kind === 'built-in' ? MARK : `<span class="rt-k">${KIND[c.kind].short}</span>`}<span>${esc(c.name)}</span><span class="t3 mono">${esc(c.node)}</span></span>`}${I('chevrons-up-down')}</button>`;
  if (phone) {
    return `<li class="gl rt-gl" data-rt="${t.id}"><div class="rt-top"><span class="ltxt"><b>${esc(t.name)}</b><span class="t3">${t.private ? 'Private' : 'Public'} · ${esc(t.software)}</span></span><span class="rt-to t2">${esc(c.name)}<span class="t3 mono">${esc(c.node)}</span></span>${pick}</div>${notes.map(noteHtml).join('')}</li>`;
  }
  return `<li class="rt-row" data-rt="${t.id}">
    <div class="rt-id"><b>${esc(t.name)}</b><span class="t3">${t.private ? 'Private' : 'Public'} · ${esc(t.software)}</span></div>
    <span class="t3 num rt-st">${t.stats.torrents} torrents · ${t.stats.uploadedTB} TB up</span>
    <span class="t3 rt-arrow">${I('arrow-right')}</span>
    ${pick}
    ${notes.length ? `<div class="rt-notes">${notes.map(noteHtml).join('')}</div>` : ''}
  </li>`;
}

function routing() {
  return `<section class="rt" data-routing aria-labelledby="rt-h">
    <div class="sec-h"><h2 id="rt-h">Tracker routing</h2><span class="t3">Which Download Client each tracker's grabs go to</span></div>
    <ul class="rt-list">${D.trackers.map(routeRow).join('')}</ul>
    <p class="st-foot t3">A tracker you add later goes to the built-in client on the Node that searches it, until you route it elsewhere.</p>
  </section>`;
}

function routeOptions(tid) {
  const t = tracker(tid);
  return D.downloadingNodes().map((n) => {
    const opts = CL.filter((c) => c.node === n.id).map((c) => {
      const on = routes[tid] === c.id;
      let note = c.kind === 'built-in' ? `Peer ID ${c.peerId}` : 'External · its own seeding rules';
      let cls = '';
      if (c.kind === 'built-in' && t.status === 'not-allowed') { note = `Not on ${t.name}'s allow list yet`; cls = ' warn'; }
      if (isDown(c)) { note = `Unreachable since ${hm(c.lastSeen)}`; cls = ' bad'; }
      else if (c.enabled === false) note = 'Off';
      return `<button class="inb" role="menuitemradio" aria-checked="${on}" data-pick="${tid}" data-c="${c.id}"><span class="inb-l"><b>${esc(c.name)}</b><span class="inb-n${cls}">${note}</span></span>${on ? I('check', 'inb-ck') : ''}</button>`;
    }).join('');
    return `<p class="inb-g mono" role="presentation">${esc(n.name)}</p>${opts}`;
  }).join('');
}

function openRoute(btn) {
  const tid = btn.dataset.route, t = tracker(tid);
  if (phone) {
    openSheet(`<p class="sheet-t"><b>${esc(t.name)}'s grabs go to</b><span class="t3">${t.private ? 'Private' : 'Public'} · ${esc(t.software)}</span></p><div class="sheet-grp rt-sheet" role="menu" aria-label="${esc(t.name)} grabs go to">${routeOptions(tid)}</div><button class="sheet-x" data-close>Cancel</button>`, btn);
    return;
  }
  closeAll();
  menuFrom = btn;
  const m = root.querySelector('.menu');
  m.innerHTML = `<p class="inb-h">${esc(t.name)}'s grabs go to</p>${routeOptions(tid)}`;
  m.setAttribute('aria-label', `${t.name} grabs go to`);
  m.classList.add('rt-menu');
  m.hidden = false;
  ctx.icons();
  const r = btn.getBoundingClientRect(), rr = root.getBoundingClientRect();
  m.style.top = `${r.bottom - rr.top + 6}px`;
  m.style.left = `${Math.max(8, r.right - rr.left - m.offsetWidth)}px`;
  (m.querySelector('.inb[aria-checked="true"]') || m.querySelector('.inb')).focus({ preventScroll: true });
}

function pickRoute(el) {
  const tid = el.dataset.pick;
  routes[tid] = el.dataset.c;
  closeAll();
  repaint(`[data-rt="${tid}"]`, routeRow(tracker(tid)));
  root.querySelectorAll('[data-rh]').forEach((x) => { x.outerHTML = routedHere(client(x.dataset.rh)); });
  ctx.icons();
  const b = root.querySelector(`[data-route="${tid}"]`);
  if (b) b.focus({ preventScroll: true });
  const c = client(routes[tid]);
  toast(`${tracker(tid).name}'s grabs now go to ${c.name} on ${c.node}`);
}

// ───────────────────────────── phone: Download Clients

const glRow = (label, right, cls = '') => `<li class="gl ${cls}"><span class="gl-k">${label}</span><span class="gl-v">${right}</span></li>`;
const glField = (id, label, input) => `<li class="gl gl-f"><label class="gl-k" for="${id}">${label}</label>${input}</li>`;

function builtinPhone(c) {
  const n = nodeOf(c.node), L = lim[c.id], t = c.totals;
  const f = (k, label, unit) => glField(`${c.id}-${k}`, label, unitInput(`${c.id}-${k}`, L[k], unit, `data-lim="${c.id}" data-f="${k}"`));
  return `<p class="cap">Built-in client</p>
    <ul class="glist">
      <li class="gl gl-head"><span class="cc-logo bi">${MARK}</span><span class="ltxt"><b>Built-in</b><span class="t3 num">${esc(c.version)} · Peer ID ${esc(c.peerId)}</span></span><span class="gl-r num">${t.downRate + t.upRate > 0 ? dot('draft') : ''}↓${r1(t.downRate)} ↑${r1(t.upRate)}</span></li>
      ${glField(`${c.id}-port`, 'Listen port', `<input class="inp num gl-in" id="${c.id}-port" type="text" inputmode="numeric" value="${L.port}" data-lim="${c.id}" data-f="port">`)}
      ${c.listen.reachable ? glRow('Reachable', `<span class="reach">${I('circle-check')}<span>from outside · ${hm(c.checkedAt)}</span></span>`) : ''}
      ${f('downMBs', 'Download speed', 'MB/s')}${f('upMBs', 'Upload speed', 'MB/s')}${f('activeDownloads', 'Download slots', 'at once')}${f('activeSeeds', 'Seed slots', 'at once')}
      <li class="gl gl-col"><dl class="kv3 kv4 kv-t"><div><dt>Torrents</dt><dd class="num">${t.torrents}</dd></div><div><dt>Seeding</dt><dd class="num">${t.seeding}</dd></div><div><dt>Ratio</dt><dd class="num">${r1(t.ratio)}</dd></div><div><dt>Problems</dt><dd class="num${t.problems ? ' warn-t' : ''}">${t.problems}</dd></div></dl></li>
      <li class="gl gl-link"><button class="gl-btn" data-go="goals"><span>Seeds by Seeding goals</span>${I('chevron-right')}</button></li>
      <li class="gl gl-link"><button class="gl-btn" data-go="torrents"><span>Open torrents</span>${I('chevron-right')}</button></li>
    </ul>
    ${c.listen.reachable ? '' : `<section class="pwarn" aria-label="Port closed"><span class="chip-warn">${I('triangle-alert')}Port closed from outside</span><p>${esc(c.listen.note)} <span class="warn-dim">${esc(c.fix)}</span></p><button class="btn warn" data-recheck="${c.id}">Check again</button><span data-tres="port-${c.id}">${tres(tests['port-' + c.id])}</span></section>`}
    <p class="gfoot">Can't be removed. It goes only when ${esc(n.name)} leaves the Downloading Role. Downloads land in <span class="mono">${esc(n.disk.path)}/SlipStream/&lt;Content Type&gt;</span>.</p>`;
}

function externalPhone(c) {
  const t = c.totals, ver = esc(c.version.replace(c.name + ' ', ''));
  if (isDown(c)) {
    return `<section class="pfail cc-p" data-cc="${c.id}" aria-label="${esc(c.name)} unreachable">
      <div class="pfail-h"><span class="pfail-id"><span class="cc-logo">${KIND[c.kind].short}</span><span class="ltxt"><b>${esc(c.name)} <span class="fail-dim num">${ver}</span></b><span class="fail-dim mono">${esc(c.host)}</span></span></span><span class="chip-fail">${I('plug-zap')}Unreachable</span></div>
      <dl class="fail-kv"><dt>Error</dt><dd class="mono">${esc(c.error)}</dd><dt>Last seen</dt><dd><span class="num">${hm(c.lastSeen)}</span> <span class="fail-dim">· ${ago(c.lastSeen)}</span></dd></dl>
      <p class="fail-dim">${I('history')} Showing last known data: <span class="num">${t.torrents} torrents · ratio ${r1(t.ratio)}</span></p>
      <p data-tres="${c.id}">${tres(tests[c.id])}</p>
      <div class="pfail-act"><button class="btn fail" data-test="${c.id}">Test connection</button><button class="btn" data-edit="${c.id}">Edit</button></div>
    </section>`;
  }
  const ck = c.check;
  return `<ul class="glist cc-p${c.enabled ? '' : ' off'}" data-cc="${c.id}" aria-label="${esc(c.name)} on ${esc(c.node)}">
    <li class="gl gl-head"><span class="cc-logo">${KIND[c.kind].short}</span><span class="ltxt"><b>${esc(c.name)} <span class="t3 num">${ver}</span></b><span class="t3 mono">${esc(c.host)}</span></span>${swBare(c.enabled, `data-enable="${c.id}"`, `${c.name} enabled`)}</li>
    ${whyFor(c) ? `<li class="gl gl-col"><p class="cc-why">${I('info')}<span>${esc(c.why)}</span></p></li>` : ''}
    ${glRow('Category', `<span class="mono">${esc(c.category)}</span>`)}
    ${glRow('Priority', `<span class="num">${c.priority}</span>`)}
    ${glRow('Import delay', `<span class="num">${c.delay ? `${c.delay} min` : 'None'}</span>`)}
    ${glRow('Health check', ck.ok ? `<span class="hc ok">${I('circle-check')}OK · <span class="num">${hm(ck.at)}</span></span>` : '<span class="hc bad">Failed</span>')}
    ${glRow('Routes here', routedHere(c))}
    ${glRow('Seeding', '<span class="t2">Its own rules</span>')}
    <li class="gl gl-act"><span data-tres="${c.id}">${tres(tests[c.id])}</span><span class="grow"></span><button class="btn sm" data-test="${c.id}">Test</button><button class="btn sm" data-edit="${c.id}">Edit</button></li>
  </ul>`;
}

function nodeGroupPhone(n) {
  const mine = CL.filter((c) => c.node === n.id);
  const bi = mine.find((c) => c.kind === 'built-in');
  const ext = mine.filter((c) => c.kind !== 'built-in');
  const p = freePct(n);
  return `<section class="psec" data-ng="${n.id}" aria-labelledby="ng-${n.id}">
    <div class="psec-h"><h2 class="mono" id="ng-${n.id}">${esc(n.name)}</h2><span class="t3">${n.kind} · ${esc(n.where.split(' · ')[0])}</span></div>
    <ul class="glist"><li class="gl gl-col ng-disk-p"><span class="mono t2">${esc(n.disk.path)}</span><div class="meter" aria-hidden="true"><span style="width:${100 - p}%"></span></div><span class="t3 num">${n.disk.freeTB} TB free of ${n.disk.totalTB} TB · ${p}%</span></li></ul>
    ${builtinPhone(bi)}
    ${ext.length ? `<p class="cap">External</p>${ext.map(externalPhone).join('<div class="gap"></div>')}<p class="gfoot">External clients keep their own seeding rules. ${esc(n.name)} talks to them and reads their files.</p>` : ''}
  </section>`;
}

function routingPhone() {
  return `<section class="psec" data-routing aria-labelledby="rt-h"><div class="psec-h"><h2 id="rt-h">Tracker routing</h2></div>
    <ul class="glist">${D.trackers.map(routeRow).join('')}</ul>
    <p class="gfoot">A tracker you add later goes to the built-in client on the Node that searches it.</p>
  </section>`;
}

function clientsBody() {
  if (phone) {
    return `<div data-page class="st-p">
      <header class="ph-nav"><button class="back tap" data-more>${I('chevron-left')}More</button><span class="ph-nav-t" aria-hidden="true">Download Clients</span><button class="ib tap" data-add aria-label="Add a Download Client">${I('plus')}</button></header>
      <header class="ph-head ph-head-push"><h1>Download Clients</h1></header>
      <p class="ph-sentence st-sent">${summary()}</p>
      <p class="ph-lede t2">Each downloading Node has a built-in client. External clients sit beside it for trackers that don't allow the built-in yet; they keep their own seeding rules.</p>
      ${D.downloadingNodes().map(nodeGroupPhone).join('')}
      ${routingPhone()}
      <div class="ph-pad st-addp"><button class="btn pri wide" data-add>${I('plus')}Add a Download Client</button></div>
    </div>`;
  }
  return `<div data-page class="st">
    <header class="st-h">
      <div class="st-ht"><h1>Download Clients</h1><p class="t2">Each downloading Node has a built-in client. External clients sit beside it for trackers that don't allow the built-in yet. SlipStream only grabs to them and imports from them; they keep their own seeding rules.</p></div>
      <button class="btn pri" data-add>${I('plus')}Add a Download Client</button>
    </header>
    <p class="st-sent">${summary()}</p>
    ${D.downloadingNodes().map(nodeGroupWide).join('')}
    ${routing()}
  </div>`;
}

// ───────────────────────────── Add / Edit a Download Client

function newForm(c) {
  const node = D.downloadingNodes()[1] || D.downloadingNodes()[0];
  if (!c) return { mode: 'add', node: node.id, kind: 'qbittorrent', host: '', port: String(KIND.qbittorrent.port), ssl: false, urlBase: '', user: '', pass: '', category: 'slipstream', priority: '1', delay: '0', test: null, err: {} };
  return { mode: 'edit', id: c.id, node: c.node, kind: c.kind, host: c.hostName, port: String(c.port), ssl: c.ssl, urlBase: c.urlBase, user: c.user, pass: '', saved: true, category: c.category, priority: String(c.priority), delay: String(c.delay), test: null, err: {} };
}

function formBody() {
  const nodes = D.downloadingNodes();
  const n = nodeOf(F.node), kn = kindName(F.kind);
  const err = (k) => (F.err[k] ? `<p class="ferr" id="err-${k}">${I('circle-alert')}${esc(F.err[k])}</p>` : '');
  const inp = (k, label, attrs = '', hint = '') => `<div class="fld${F.err[k] ? ' bad' : ''}"><label for="f-${k}">${label}</label><input class="inp" id="f-${k}" type="text" autocomplete="off" spellcheck="false" value="${esc(F[k])}" data-ff="${k}" ${F.err[k] ? `aria-invalid="true" aria-describedby="err-${k}"` : ''} ${attrs}>${err(k)}${hint ? `<p class="hint">${hint}</p>` : ''}</div>`;
  const nodeCards = `<div class="rc rc-2" role="radiogroup" aria-labelledby="lg-node">${nodes.map((x) => `<button type="button" class="rc-o" role="radio" aria-checked="${x.id === F.node}" tabindex="${x.id === F.node ? 0 : -1}" data-ffr="node" data-v="${x.id}"><span class="rc-t mono">${esc(x.name)}</span><span class="t3">${x.kind} · ${esc(x.where)}</span><span class="rc-ck" aria-hidden="true">${I('check')}</span></button>`).join('')}</div>`;
  const kindCards = `<div class="rc rc-4" role="radiogroup" aria-labelledby="lg-kind">${D.externalKinds.map((k) => `<button type="button" class="rc-o" role="radio" aria-checked="${k.kind === F.kind}" tabindex="${k.kind === F.kind ? 0 : -1}" data-ffr="kind" data-v="${k.kind}"><span class="rc-t"><span class="cc-logo sm">${KIND[k.kind].short}</span>${esc(k.name)}</span><span class="t3">${esc(k.note)}</span><span class="rc-ck" aria-hidden="true">${I('check')}</span></button>`).join('')}</div>`;
  const pass = F.saved
    ? `<div class="fld"><label for="f-pass">Password</label><input class="inp" id="f-pass" type="password" autocomplete="new-password" value="" placeholder="Saved, hidden" data-ff="pass"><p class="hint">Leave empty to keep the saved password.</p></div>`
    : `<div class="fld"><label for="f-pass">Password</label><input class="inp" id="f-pass" type="password" autocomplete="new-password" value="${esc(F.pass)}" data-ff="pass"></div>`;
  const down = F.mode === 'edit' && isDown(client(F.id)) ? client(F.id) : null;
  return `${down ? `<p class="rt-n rt-fail">${I('plug-zap')}<span>Unreachable since ${hm(down.lastSeen)}: ${esc(down.error)}. Fix the address or start ${esc(down.name)}, then test.</span></p>` : ''}
    <fieldset class="fs"><legend id="lg-node">Node</legend>${nodeCards}<p class="hint">${esc(n.name)} talks to ${esc(kn)} and reads its files, so it must run where ${esc(n.name)} can reach it. A remote seedbox needs a SlipStream Node on it.</p></fieldset>
    <fieldset class="fs"><legend id="lg-kind">Client</legend>${kindCards}</fieldset>
    <fieldset class="fs"><legend>Connection</legend>
      <div class="fg fg-conn">
        ${inp('host', 'Host', `placeholder="e.g. ${F.node === 'atlas' ? '192.168.1.20' : '127.0.0.1'}"`)}
        ${inp('port', 'Port', 'inputmode="numeric"')}
        <div class="fld fld-sw"><span class="lbl" id="lb-ssl">SSL</span>${swBtn(F.ssl, 'data-ssl aria-labelledby="lb-ssl"', F.ssl ? 'On' : 'Off')}</div>
        ${inp('urlBase', 'URL base', `placeholder="${KIND[F.kind].urlBase || 'None'}"`)}
        ${inp('user', 'Username')}
        ${pass}
      </div>
    </fieldset>
    <fieldset class="fs"><legend>Grabs and import</legend>
      <div class="fg fg-imp">
        ${inp('category', 'Category', '', `Lands in <span class="mono">${esc(F.category || 'category')}/movie</span> and <span class="mono">/series</span>`)}
        ${inp('priority', 'Priority', 'inputmode="numeric"', 'Lower goes first')}
        <div class="fld"><label for="f-delay">Import delay</label>${unitInput('f-delay', esc(F.delay), 'min', 'data-ff="delay"', '0')}<p class="hint">Wait after it finishes</p></div>
      </div>
    </fieldset>`;
}

function formHtml() {
  const title = F.mode === 'add' ? 'Add a Download Client' : `Edit ${kindName(F.kind)}`;
  const foot = `<span class="dlg-res" data-ftres aria-live="polite">${tres(F.test)}</span>`;
  if (phone) {
    return `<div class="dlg dlg-p" role="dialog" aria-modal="true" aria-labelledby="dlg-t">
      <header class="dlg-ph"><button class="dlg-pb" data-close>Cancel</button><h2 id="dlg-t">${title}</h2><button class="dlg-pb pri" data-fsave>Save</button></header>
      <div class="dlg-b" data-fbody>${formBody()}</div>
      <div class="dlg-pf">${foot}<button class="btn wide" data-ftest>${I('plug')}Test connection</button></div>
    </div>`;
  }
  return `<div class="dlg" role="dialog" aria-modal="true" aria-labelledby="dlg-t">
    <header class="dlg-h"><div><h2 id="dlg-t">${title}</h2><p class="t3">External clients keep their own seeding rules. SlipStream grabs to them and imports from them.</p></div><button class="ib" data-close aria-label="Close">${I('x')}</button></header>
    <div class="dlg-b" data-fbody>${formBody()}</div>
    <footer class="dlg-f">${foot}<span class="grow"></span><button class="btn" data-close>Cancel</button><button class="btn" data-ftest>${I('plug')}Test</button><button class="btn pri" data-fsave>Save</button></footer>
  </div>`;
}

function openForm(c, from) {
  F = newForm(c);
  openSheet(formHtml(), from);
  root.querySelector('.sheet').classList.add('sheet-dlg');
  const first = root.querySelector('.sheet .rc-o[aria-checked="true"]');
  if (first) first.focus({ preventScroll: true });
}

function refreshForm(focusSel) {
  const b = root.querySelector('[data-fbody]');
  if (!b) return;
  const st = b.scrollTop;
  b.innerHTML = formBody();
  b.scrollTop = st;
  ctx.icons();
  if (focusSel) { const el = b.querySelector(focusSel); if (el) el.focus({ preventScroll: true }); }
}
const setFtres = () => { const el = root.querySelector('[data-ftres]'); if (el) { el.innerHTML = tres(F.test); ctx.icons(); } };

function validate() {
  F.err = {};
  if (!F.host.trim()) F.err.host = `Enter the address ${F.node} reaches ${kindName(F.kind)} at`;
  if (!/^\d{2,5}$/.test(F.port) || +F.port > 65535) F.err.port = 'Use a port from 1 to 65535';
  return !Object.keys(F.err).length;
}

function testForm() {
  if (!validate()) { F.test = null; refreshForm(F.err.host ? '#f-host' : '#f-port'); setFtres(); return; }
  refreshForm();
  F.test = { state: 'busy', msg: `Asking ${kindName(F.kind)} through ${F.node}…` };
  setFtres();
  setTimeout(() => {
    if (!F) return;
    F.test = F.host.includes('192.168.1.20')
      ? { state: 'bad', msg: `Connection refused at ${F.host}:${F.port}. Is ${kindName(F.kind)} running?` }
      : { state: 'ok', msg: `Connected to ${kindName(F.kind)} ${KIND[F.kind].version}. ${F.node} can read its download folder.` };
    setFtres();
  }, 900);
}

function saveForm() {
  if (!validate()) { refreshForm(F.err.host ? '#f-host' : '#f-port'); return; }
  const host = `${F.ssl ? 'https' : 'http'}://${F.host}:${F.port}${F.urlBase && F.urlBase !== '/' ? F.urlBase.replace(/\/$/, '') : ''}`;
  const patch = { hostName: F.host, port: +F.port, ssl: F.ssl, urlBase: F.urlBase, user: F.user, category: F.category, priority: +F.priority || 1, delay: +F.delay || 0, host };
  let c;
  if (F.mode === 'add') {
    c = { id: `new-${Date.now()}`, node: F.node, kind: F.kind, name: kindName(F.kind), status: 'ok', version: `${kindName(F.kind)} ${KIND[F.kind].version}`, enabled: true, check: { ok: true, at: D.now.toISOString(), ms: 4 }, totals: { torrents: 0, downloading: 0, seeding: 0, queued: 0, problems: 0, downRate: 0, upRate: 0, ratio: 0, uploadedTB: 0 }, ...patch };
    CL.push(c);
  } else {
    c = client(F.id);
    Object.assign(c, patch, { node: F.node, kind: F.kind });
    if (F.test && F.test.state === 'ok') Object.assign(c, { status: 'ok', check: { ok: true, at: D.now.toISOString(), ms: 5 } });
  }
  const msg = F.mode === 'add' ? `${c.name} added to ${c.node}` : `${c.name} on ${c.node} saved`;
  F = null;
  closeAll();
  paintClients();
  toast(msg);
}

function paintClients() {
  D.downloadingNodes().forEach((n) => repaint(`[data-ng="${n.id}"]`, phone ? nodeGroupPhone(n) : nodeGroupWide(n)));
  repaint('[data-routing]', phone ? routingPhone() : routing());
  const s = root.querySelector('.st-sent');
  if (s) s.innerHTML = summary();
  ctx.icons();
}

// ═════════════════════════════ Seeding goals

const effGoal = (tid) => (G.per[tid] && G.per[tid].override ? G.per[tid] : G.def);
const goalOf = (key) => (key === 'def' ? G.def : G.per[key]);

function floorText(t) {
  const r = t.rule;
  if (!r) return 'No tracker rule. Public torrents can stop any time.';
  if (!r.minHours && r.minRatio) return `${t.name} requires an account ratio above ${r.minRatio}.`;
  const parts = [`${hrs(r.minHours)} of seeding`];
  if (r.minRatio) parts.push(`ratio ${r1(r.minRatio)}`);
  return `${t.name} requires ${parts.join(' or ')} within ${r.windowDays} days of finishing.`;
}

function goalWords(g) {
  if (g.mode === 'forever') return '<b>Seed until you remove it.</b>';
  const sides = [];
  if (g.ratio != null) sides.push(`<b class="num">ratio ${r1(g.ratio)}</b>`);
  if (g.hours != null) sides.push(`<b class="num">${gh(g.hours)}</b>`);
  const then = `Then ${WHEN_THEN[g.whenMet]}.`;
  if (!sides.length) return `Seed only until the tracker's rule is met. ${then}`;
  if (sides.length === 1) return `Seed until ${sides[0]} and the tracker's rule is met. ${then}`;
  return g.mode === 'both'
    ? `Seed until ${sides[0]} and ${sides[1]}, and the tracker's rule is met. ${then}`
    : `Seed until ${sides[0]} or ${sides[1]}, whichever comes first, and the tracker's rule is met. ${then}`;
}
const goalShort = (g) => (g.mode === 'forever' ? 'Seed forever' : `${[g.ratio != null ? `ratio ${r1(g.ratio)}` : '', g.hours != null ? gh(g.hours) : ''].filter(Boolean).join(g.mode === 'both' ? ' and ' : ' or ') || "the tracker's rule"}, then ${WHEN_SHORT[g.whenMet].toLowerCase()}`);

function underFloor(g, t) {
  const r = t.rule;
  if (!r || g.mode === 'forever') return null;
  const lows = [];
  if (r.minHours != null && g.hours != null && g.hours < r.minHours) lows.push(`${gh(g.hours)} is under ${t.name}'s ${hrs(r.minHours)}`);
  if (r.minRatio != null && g.ratio != null && g.ratio < r.minRatio) lows.push(`ratio ${r1(g.ratio)} is under ${t.name}'s ${r1(r.minRatio)}`);
  if (!lows.length) return null;
  return `${lows.join(', and ')}. ${t.name}'s rule still holds each torrent until it's met.`;
}

function goalEditor(key, compact) {
  const g = goalOf(key), u = units[key], forever = g.mode === 'forever';
  const tv = g.hours == null ? '' : u === 'd' ? +(g.hours / 24).toFixed(2) : g.hours;
  const unitSeg = seg('Time unit', [['d', 'days'], ['h', 'hours']], u, (v) => `data-g="${key}" data-f="unit" data-v="${v}"`, { disabled: forever, cls: 'seg-u' });
  return `<div class="ge${compact ? ' ge-c' : ''}${forever ? ' forever' : ''}" data-ge="${key}">
    <div class="fld"><label for="${key}-ratio">Ratio</label>${unitInput(`${key}-ratio`, g.ratio == null ? '' : r1(g.ratio), 'ratio', `data-g="${key}" data-f="ratio" ${forever ? 'disabled' : ''}`, 'None')}</div>
    <div class="fld fld-time"><label for="${key}-time">Time seeded</label><span class="time-w">${unitInput(`${key}-time`, tv, '', `data-g="${key}" data-f="time" ${forever ? 'disabled' : ''}`, 'None')}${unitSeg}</span></div>
    <div class="fld fld-mode"><span class="lbl" id="${key}-mode-l">Goal is met at</span>${seg('Goal is met at', MODES, g.mode, (v) => `data-g="${key}" data-f="mode" data-v="${v}"`, { cls: 'seg-mode' })}</div>
    ${compact ? `<div class="fld fld-wm"><span class="lbl">When met</span>${seg('When met', Object.entries(WHEN_SHORT), g.whenMet, (v) => `data-g="${key}" data-f="whenMet" data-v="${v}" title="${esc(D.whenMetLabel[v])}"`, { disabled: forever })}</div>` : ''}
    <p class="ferr" data-ferr="${key}" hidden></p>
  </div>`;
}

function whenMetList() {
  const g = G.def, forever = g.mode === 'forever';
  return `<div class="wm" role="radiogroup" aria-label="When the default goal is met"${forever ? ' aria-disabled="true"' : ''}>${Object.entries(D.whenMetLabel).map(([v, l]) => `<button type="button" class="wm-o" role="radio" aria-checked="${g.whenMet === v}" tabindex="${g.whenMet === v ? 0 : -1}" ${forever ? 'disabled' : ''} data-g="def" data-f="whenMet" data-v="${v}">${I(WHEN_ICON[v])}<span>${esc(l)}</span><span class="wm-ck" aria-hidden="true">${I('check')}</span></button>`).join('')}</div>`;
}

const defNote = () => {
  const lows = D.trackers.filter((t) => t.rule && !(G.per[t.id] && G.per[t.id].override) && t.id !== 'aurora' && underFloor(G.def, t));
  return lows.length ? `<p class="info-n">${I('shield')}<span>${lows.map((t) => esc(underFloor(G.def, t))).join(' ')}</span></p>` : '';
};

function trackerGoal(t) {
  if (t.id === 'aurora') {
    const c = client(routes.aurora);
    return `<article class="tg tg-ext" data-tg="${t.id}" aria-labelledby="tg-${t.id}">
      <div class="tg-h"><h3 id="tg-${t.id}">${esc(t.name)}</h3><span class="chip">Private · ${esc(t.software)}</span><span class="grow"></span><span class="chip node">${I('route')}${esc(c.name)} · ${esc(c.node)}</span></div>
      <p class="tg-floor">${I('shield')}<span>${esc(floorText(t))}</span></p>
      <p class="tg-x">${I('info')}<span>Aurora's torrents live in ${esc(c.name)}, whose own rules apply, so goals don't apply here. If Aurora moves back to the built-in client, the default goal applies.</span><button class="lnk" data-go="clients"><b>Tracker routing</b></button></p>
    </article>`;
  }
  const p = G.per[t.id], on = p.override;
  const body = on
    ? `${goalEditor(t.id, true)}<p class="tg-sent" data-sent="${t.id}">${goalWords(p)}</p>${p.why ? `<p class="t3 tg-why">${I('message-square')}${esc(p.why)}</p>` : ''}`
    : `<p class="tg-inh" data-inh="${t.id}">Uses the default: <b>${esc(goalShort(G.def))}</b>.</p>`;
  return `<article class="tg${on ? ' own' : ''}" data-tg="${t.id}" aria-labelledby="tg-${t.id}">
    <div class="tg-h"><h3 id="tg-${t.id}">${esc(t.name)}</h3><span class="chip">${t.private ? 'Private' : 'Public'} · ${esc(t.private ? t.software : 'DHT, PEX, open')}</span><span class="grow"></span>${phone ? swBare(on, `data-ovr="${t.id}"`, `${t.name} has its own goal`) : swBtn(on, `data-ovr="${t.id}"`, 'Own goal')}</div>
    <p class="tg-floor">${I(t.rule ? 'shield' : 'shield-off')}<span>${esc(floorText(t))}</span></p>
    ${body}
    <div data-note="${t.id}">${noteFor(t)}</div>
  </article>`;
}
const noteFor = (t) => { const n = underFloor(effGoal(t.id), t); return n ? `<p class="info-n">${I('info')}<span>${esc(n)}</span></p>` : ''; };

function goalMet(t, g) {
  const R = g.ratio, T = g.hours;
  if (R == null && T == null) return true;
  if (g.mode === 'both') return (R == null || t.ratio >= R) && (T == null || t.seededH >= T);
  return (R != null && t.ratio >= R) || (T != null && t.seededH >= T);
}

function impact() {
  const o = { now: [], held: [], toward: 0, forever: 0, pending: 0, external: 0, hnr: 0, risk: 0, gone: 0 };
  for (const t of D.torrents) {
    const c = client(t.client);
    if (c.kind !== 'built-in') { o.external++; continue; }
    if (t.state === 'error') { o.gone++; continue; }
    if (['downloading', 'queued', 'checking'].includes(t.state)) { o.pending++; continue; }
    const hn = D.hnr(t);
    if (hn) { o.hnr++; if (hn.atRisk) o.risk++; }
    const g = effGoal(t.tracker);
    if (g.mode === 'forever') { o.forever++; continue; }
    if (!goalMet(t, g)) o.toward++;
    else if (!D.ruleMet(t, tracker(t.tracker).rule)) o.held.push(t);
    else o.now.push({ t, whenMet: g.whenMet });
  }
  return o;
}

function impactHtml() {
  const o = impact(), prev = lastImpact;
  lastImpact = { now: o.now.length, held: o.held.length, toward: o.toward, forever: o.forever, hnr: o.hnr };
  const by = (k) => o.now.filter((x) => x.whenMet === k).length;
  const acts = [[by('remove'), 'will be removed'], [by('stop'), 'will stop seeding'], [by('keep'), 'will keep seeding']].filter(([n]) => n);
  const verb = acts.map(([n, v]) => `${n} ${v}`).join(', ');
  const nowP = o.now.length ? `<b>${plural(o.now.length, 'torrent meets its', 'torrents meet their')} goal now</b>${acts.length === 1 ? ` and ${acts[0][1]}` : `: ${verb}`}` : '<b>no torrent meets its goal now</b>';
  const sent = [nowP, o.held.length ? `${o.held.length} ${o.held.length === 1 ? 'is' : 'are'} held by tracker rules` : '', o.hnr ? `${o.hnr} ${o.hnr === 1 ? 'is' : 'are'} inside a hit-and-run window` : ''].filter(Boolean);
  const bump = (k) => (prev && prev[k] !== lastImpact[k] ? ' bump' : '');
  const row = (k, n, label, sub = '') => `<li class="imp-r"><span class="imp-n num${bump(k)}">${n}</span><span class="imp-l">${label}${sub ? `<span class="t3">${sub}</span>` : ''}</span></li>`;
  const list = (arr, why) => (arr.length ? `<ul class="imp-w">${arr.map((x) => { const t = x.t || x, w = D.works[t.work]; return `<li data-work="${w.id}" data-amb>${art(w, 'thumb')}<span class="ltxt"><b class="ell">${esc(w.title)}</b><span class="t3">${esc(D.coverageLabel(t.work, t.coverage))} · ${esc(tracker(t.tracker).name)}</span></span><span class="t3 imp-why">${why(x)}</span></li>`; }).join('')}</ul>` : '');
  return `<p class="imp-s">With these goals, ${sent.join(', ')}.</p>
    <ul class="imp-rows">
      ${row('now', o.now.length, 'Meet their goal now', verb ? ` · ${verb}` : '')}
      ${row('held', o.held.length, 'Held by a tracker rule', ' · goal met, rule not yet')}
      ${row('toward', o.toward, 'Seeding toward a goal')}
      ${row('forever', o.forever, 'Seed forever')}
      ${row('hnr', o.hnr, 'Inside a hit-and-run window', o.risk ? ` · <span class="warn-t">${o.risk} at risk</span>` : '')}
    </ul>
    ${list(o.now, (x) => esc(WHEN_SHORT[x.whenMet]))}
    ${list(o.held, (t) => { const r = tracker(t.tracker).rule; return `${Math.ceil(r.minHours - t.seededH)} h to go`; })}
    <p class="t3 imp-f">${plural(o.pending, 'torrent is', 'torrents are')} still downloading. ${plural(o.external, 'torrent', 'torrents')} in external clients follow their own rules.</p>`;
}

function diskGuard() {
  const nodes = D.downloadingNodes();
  const g = G.disk;
  return `<div class="dg">
    <div class="dg-f"><label for="dg-pct">Pause new downloads on a Node when its free space drops below</label>${unitInput('dg-pct', g.minFreePct, '%', 'data-dg="minFreePct"', '10')}</div>
    <ul class="dg-n">${nodes.map((n) => { const p = freePct(n), low = p < g.minFreePct; return `<li><span class="mono">${esc(n.name)}</span><div class="meter dg-m" aria-hidden="true"><span style="width:${100 - p}%"></span><i style="left:${100 - g.minFreePct}%"></i></div><span class="num ${low ? 'warn-t' : 't3'}">${p}% free${low ? ' · paused' : ''}</span></li>`; }).join('')}</ul>
    <p class="t3">Seeding carries on while downloads pause. Torrents leave only by their goals or by hand.</p>
  </div>`;
}

const drainNote = () => `<p class="drain">${I('log-out')}<span>A Node leaving the Downloading Role drains first. Its built-in client takes no new grabs, and seeding torrents count as unfinished until their goals are met.</span></p>`;

function saveBar() {
  return `<div class="savebar" data-savebar hidden role="region" aria-label="Unsaved changes"><span>${dot('draft')}Unsaved changes to seeding goals</span><button class="btn ghost sm" data-revert>Revert</button><button class="btn pri sm" data-save>Save</button></div>`;
}

function goalsBody() {
  const trs = D.trackers.filter((t) => t.private).concat(D.trackers.filter((t) => !t.private));
  if (phone) {
    return `<div data-page class="st-p st-goals">
      <header class="ph-nav"><button class="back tap" data-more>${I('chevron-left')}More</button><span class="ph-nav-t" aria-hidden="true">Seeding goals</span><span class="ph-nav-sp" aria-hidden="true"></span></header>
      <header class="ph-head ph-head-push"><h1>Seeding goals</h1></header>
      <p class="ph-lede t2">For the built-in client on every downloading Node. External clients keep their own seeding rules.</p>
      <section class="psec"><div class="psec-h"><h2>Default goal</h2></div>
        <div class="glist gpad">${goalEditor('def', false)}</div>
        <p class="gfoot gsent" data-sent="def">${goalWords(G.def)}</p>
        <div class="ph-pad" data-defnote>${defNote()}</div>
      </section>
      <section class="psec"><div class="psec-h"><h2>When met</h2></div><div class="glist" data-wm>${whenMetList()}</div><p class="gfoot">Library files are never touched. Import hard-linked or copied them.</p></section>
      <section class="psec"><div class="psec-h"><h2>Right now</h2></div><div class="glist gpad imp" data-imp aria-live="polite">${impactHtml()}</div></section>
      <section class="psec"><div class="psec-h"><h2>Tracker goals</h2></div><p class="gfoot gfoot-top">A tracker's own rule is a floor. A goal is met only when the tracker's rule is met too.</p><ul class="glist tgs">${trs.map((t) => `<li>${trackerGoal(t)}</li>`).join('')}</ul></section>
      <section class="psec"><div class="psec-h"><h2>Disk guard</h2></div><div class="glist gpad">${diskGuard()}</div></section>
      <div class="ph-pad psec">${drainNote()}</div>
      ${saveBar()}
    </div>`;
  }
  return `<div data-page class="st st-goals">
    <header class="st-h"><div class="st-ht"><h1>Seeding goals</h1><p class="t2">How long the built-in client on every downloading Node seeds what it downloads. External clients keep their own seeding rules.</p></div></header>
    <div class="gg">
      <div class="gg-main">
        <section class="gcard" aria-labelledby="def-h">
          <div class="sec-h"><h2 id="def-h">Default goal</h2><span class="t3">For every tracker without its own goal</span></div>
          ${goalEditor('def', false)}
          <p class="gsent" data-sent="def">${goalWords(G.def)}</p>
          <div data-defnote>${defNote()}</div>
          <h3 class="sub-h">When met</h3>
          <div data-wm>${whenMetList()}</div>
          <p class="t3 lib-n">${I('library')}Library files are never touched. Import hard-linked or copied them.</p>
        </section>
        <section aria-labelledby="tg-hh">
          <div class="sec-h"><h2 id="tg-hh">Tracker goals</h2><span class="t3">A tracker's own rule is a floor. A goal is met only when the tracker's rule is met too.</span></div>
          <div class="tgs">${trs.map(trackerGoal).join('')}</div>
        </section>
        <section class="gcard" aria-labelledby="dg-h"><div class="sec-h"><h2 id="dg-h">Disk guard</h2></div>${diskGuard()}</section>
      </div>
      <aside class="gg-rail">
        <section class="gcard imp-c" aria-labelledby="imp-h"><div class="sec-h"><h2 id="imp-h">Right now</h2><span class="t3">Updates as you edit</span></div><div class="imp" data-imp aria-live="polite">${impactHtml()}</div></section>
        ${drainNote()}
      </aside>
    </div>
    ${saveBar()}
  </div>`;
}

function refreshGoals() {
  root.querySelectorAll('[data-sent]').forEach((el) => { el.innerHTML = goalWords(goalOf(el.dataset.sent)); });
  root.querySelectorAll('[data-inh]').forEach((el) => { el.innerHTML = `Uses the default: <b>${esc(goalShort(G.def))}</b>.`; });
  root.querySelectorAll('[data-note]').forEach((el) => { el.innerHTML = noteFor(tracker(el.dataset.note)); });
  const dn = root.querySelector('[data-defnote]');
  if (dn) dn.innerHTML = defNote();
  const imp = root.querySelector('[data-imp]');
  if (imp) imp.innerHTML = impactHtml();
  const sb = root.querySelector('[data-savebar]');
  if (sb) sb.hidden = JSON.stringify(G) === baseline;
  ctx.icons();
}

function setGoalField(key, f, v) {
  const g = goalOf(key);
  if (f === 'unit') {
    units[key] = v;
    const inp = root.querySelector(`#${key}-time`);
    if (inp && g.hours != null) inp.value = v === 'd' ? +(g.hours / 24).toFixed(2) : g.hours;
    return;
  }
  g[f] = v;
  if (f === 'mode') {
    const ge = root.querySelector(`[data-ge="${key}"]`);
    const forever = v === 'forever';
    ge.classList.toggle('forever', forever);
    ge.querySelectorAll('input').forEach((i) => { i.disabled = forever; });
    ge.querySelectorAll('.seg-u .seg-o, .fld-wm .seg-o').forEach((b) => { b.disabled = forever; });
    if (key === 'def') { const wm = root.querySelector('[data-wm]'); wm.innerHTML = whenMetList(); }
  }
}

function onGoalInput(el) {
  const key = el.dataset.g, f = el.dataset.f, g = goalOf(key);
  const raw = el.value.trim().replace(',', '.');
  const errEl = root.querySelector(`[data-ferr="${key}"]`);
  const num = raw === '' ? null : Number(raw);
  const bad = num != null && (!Number.isFinite(num) || num < 0);
  el.setAttribute('aria-invalid', String(bad));
  if (errEl) { errEl.hidden = !bad; errEl.innerHTML = bad ? `${I('circle-alert')}${f === 'ratio' ? 'Use a number like 1.5, or leave it empty' : 'Use a number of days or hours, or leave it empty'}` : ''; if (bad) ctx.icons(); }
  if (bad) return;
  if (f === 'ratio') g.ratio = num;
  if (f === 'time') g.hours = num == null ? null : Math.round(units[key] === 'd' ? num * 24 : num);
  refreshGoals();
}

// ═════════════════════════════ wiring

function repaint(sel, html) {
  const el = root.querySelector(sel);
  if (!el) return null;
  const t = document.createElement('div');
  t.innerHTML = html.trim();
  const n = t.firstElementChild;
  el.replaceWith(n);
  ctx.icons();
  return n;
}

function runTest(id) {
  const c = client(id);
  tests[id] = { state: 'busy', msg: `Asking ${c.name}…` };
  const slot = () => root.querySelector(`[data-tres="${id}"]`);
  slot().innerHTML = tres(tests[id]); ctx.icons();
  setTimeout(() => {
    tests[id] = isDown(c)
      ? { state: 'bad', msg: `Still refused at ${c.hostName}:${c.port} · ${hm(D.now)}` }
      : { state: 'ok', msg: `Connected · ${c.check?.ms || 4} ms` };
    const s = slot();
    if (s) { s.innerHTML = tres(tests[id]); ctx.icons(); }
  }, 900);
}

function runRecheck(id) {
  const c = client(id), key = 'port-' + id;
  tests[key] = { state: 'busy', msg: `Checking port ${lim[id].port} from outside…` };
  const slot = () => root.querySelector(`[data-tres="${key}"]`);
  slot().innerHTML = tres(tests[key]); ctx.icons();
  setTimeout(() => {
    tests[key] = c.listen.reachable ? { state: 'ok', msg: `Reachable · ${hm(D.now)}` } : { state: 'warn', msg: `Still closed · ${hm(D.now)}` };
    const s = slot();
    if (s) { s.innerHTML = tres(tests[key]); ctx.icons(); }
  }, 1100);
}

function moveRadio(group, dir) {
  const opts = [...group.querySelectorAll('[role="radio"]:not([disabled])')];
  if (!opts.length) return;
  const i = opts.indexOf(document.activeElement);
  const n = opts[(i + dir + opts.length) % opts.length];
  n.click();
  const again = n.isConnected ? n : null;
  const target = again || root.querySelector(`[data-ffr="${n.dataset.ffr}"][data-v="${n.dataset.v}"]`);
  if (target) target.focus();
}

function selectRadio(el) {
  const grp = el.closest('[role="radiogroup"]');
  grp.querySelectorAll('[role="radio"]').forEach((b) => { const on = b === el; b.setAttribute('aria-checked', String(on)); b.tabIndex = on ? 0 : -1; });
}

function click(e, q) {
  let el;
  if ((el = q('[data-more]'))) { root.querySelector('[data-tab="more"]').click(); return true; }
  if ((el = q('[data-add]'))) { openForm(null, el); return true; }
  if ((el = q('[data-edit]'))) { openForm(client(el.dataset.edit), el); return true; }
  if ((el = q('[data-test]'))) { runTest(el.dataset.test); return true; }
  if ((el = q('[data-recheck]'))) { runRecheck(el.dataset.recheck); return true; }
  if ((el = q('[data-route]'))) { e.stopPropagation(); openRoute(el); return true; }
  if ((el = q('[data-pick]'))) { pickRoute(el); return true; }
  if ((el = q('[data-enable]'))) {
    const c = client(el.dataset.enable);
    c.enabled = !c.enabled;
    paintClients();
    const b = root.querySelector(`[data-enable="${c.id}"]`);
    if (b) b.focus({ preventScroll: true });
    return true;
  }
  if ((el = q('[data-ffr]'))) {
    F[el.dataset.ffr] = el.dataset.v;
    if (el.dataset.ffr === 'kind' && F.mode === 'add') { F.port = String(KIND[F.kind].port); F.urlBase = KIND[F.kind].urlBase; }
    F.test = null;
    refreshForm(`[data-ffr="${el.dataset.ffr}"][data-v="${el.dataset.v}"]`);
    setFtres();
    return true;
  }
  if ((el = q('[data-ssl]'))) { F.ssl = !F.ssl; el.setAttribute('aria-checked', String(F.ssl)); el.querySelector('span').textContent = F.ssl ? 'On' : 'Off'; return true; }
  if ((el = q('[data-ftest]'))) { testForm(); return true; }
  if ((el = q('[data-fsave]'))) { saveForm(); return true; }
  if ((el = q('[data-g][data-v]'))) {
    if (el.disabled) return true;
    selectRadio(el);
    setGoalField(el.dataset.g, el.dataset.f, el.dataset.v);
    refreshGoals();
    return true;
  }
  if ((el = q('[data-ovr]'))) {
    const tid = el.dataset.ovr, p = G.per[tid];
    p.override = !p.override;
    if (p.override && p.mode !== 'forever' && p.ratio == null && p.hours == null) Object.assign(p, { ...G.def });
    repaint(`[data-tg="${tid}"]`, trackerGoal(tracker(tid)));
    root.querySelector(`[data-ovr="${tid}"]`).focus({ preventScroll: true });
    refreshGoals();
    return true;
  }
  if ((el = q('[data-revert]'))) {
    G = JSON.parse(baseline);
    repaint('[data-page]', goalsBody());
    toast('Seeding goals reverted');
    return true;
  }
  if ((el = q('[data-save]'))) {
    baseline = JSON.stringify(G);
    refreshGoals();
    toast('Seeding goals saved. The built-in clients apply them on their next check');
    return true;
  }
  return false;
}

function actions(key) {
  const c = client(key.split(':')[1]);
  return { title: `${c.name} on ${c.node}`, sub: c.host, items: [[`Open ${c.name}'s web UI`, 'external-link', ''], ['Show its torrents', 'magnet', 'go', 'torrents'], [`Remove ${c.name} from SlipStream`, 'trash-2', 'danger']] };
}

function after() {
  if (phone) root.addEventListener('scroll', () => { const n = root.querySelector('.ph-nav'); if (n) n.classList.toggle('titled', root.scrollTop > 64); }, { passive: true });
  root.addEventListener('input', (e) => {
    const el = e.target;
    if (el.dataset.ff && F) {
      F[el.dataset.ff] = el.value;
      if (F.err[el.dataset.ff]) { delete F.err[el.dataset.ff]; el.removeAttribute('aria-invalid'); const m = root.querySelector(`#err-${el.dataset.ff}`); if (m) m.remove(); el.closest('.fld').classList.remove('bad'); }
      if (el.dataset.ff === 'category') { const hint = el.closest('.fld').querySelector('.hint'); if (hint) hint.innerHTML = `Lands in <span class="mono">${esc(F.category || 'category')}/movie</span> and <span class="mono">/series</span>`; }
      if (F.test && F.test.state !== 'busy') { F.test = null; setFtres(); }
    }
    if (el.dataset.g) onGoalInput(el);
    if (el.dataset.dg) {
      const v = Number(el.value);
      if (Number.isFinite(v) && v >= 0 && v <= 90) { G.disk.minFreePct = v; root.querySelectorAll('.dg-n').forEach((ul) => { const t = document.createElement('div'); t.innerHTML = diskGuard(); ul.replaceWith(t.querySelector('.dg-n')); }); refreshGoals(); }
    }
  });
  root.addEventListener('change', (e) => {
    const el = e.target;
    if (el.dataset.lim) {
      const raw = el.value.trim();
      lim[el.dataset.lim][el.dataset.f] = raw === '' ? null : Number(raw);
      const s = root.querySelector(`[data-saved="${el.dataset.lim}"]`);
      if (s) { s.textContent = 'Saved'; s.classList.add('on'); setTimeout(() => s.classList.remove('on'), 1600); }
      if (phone) toast(`${el.closest('.gl').querySelector('.gl-k').textContent} saved`);
    }
  });
  root.addEventListener('keydown', (e) => {
    const grp = e.target.closest && e.target.closest('[role="radiogroup"]');
    if (grp && ['ArrowLeft', 'ArrowRight', 'ArrowUp', 'ArrowDown'].includes(e.key)) {
      e.preventDefault(); e.stopPropagation();
      moveRadio(grp, e.key === 'ArrowLeft' || e.key === 'ArrowUp' ? -1 : 1);
    }
    if (e.target.matches && e.target.matches('input') && (e.key === 'ArrowLeft' || e.key === 'ArrowRight')) e.stopPropagation();
  });
  document.addEventListener('keydown', (e) => {
    if (e.key === 'Escape' && menuFrom) { const b = menuFrom; menuFrom = null; setTimeout(() => b.isConnected && b.focus({ preventScroll: true })); }
  }, true);
  const m = root.querySelector('.menu');
  new MutationObserver(() => { if (m.hidden) m.classList.remove('rt-menu'); }).observe(m, { attributes: true, attributeFilter: ['hidden'] });
  const sw = root.querySelector('.sheet-wrap');
  new MutationObserver(() => { if (sw.hidden) { root.querySelector('.sheet').classList.remove('sheet-dlg'); F = null; } }).observe(sw, { attributes: true, attributeFilter: ['hidden'] });
}

export function render(c) {
  ctx = c; D = c.data; root = c.root; phone = c.device === 'phone';
  initState();
  const goals = c.screen === 'goals';
  const title = goals ? 'Seeding goals' : 'Download Clients';
  mount(c, {
    id: goals ? 'goals' : 'clients',
    title,
    nav: goals ? 'goals' : 'clients',
    tab: 'more',
    lead: 'dune-2',
    crumbs: `<button class="crumb" data-toast="Settings overview is not in this prototype">Settings</button>${I('chevron-right', 'crumb-sep')}<span class="crumb cur">${title}</span>`,
    body: goals ? goalsBody() : clientsBody(),
    actions,
    click,
    after,
  });
}
