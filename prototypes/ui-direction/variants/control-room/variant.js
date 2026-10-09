// PROTOTYPE · Direction A · Control room. Throwaway sketch for the UI direction pick.

let D, C, root;
let isPhone, isAdmin;
const requested = new Set();

const esc = (s) => String(s ?? '').replace(/[&<>"]/g, (c) => ({ '&': '&amp;', '<': '&lt;', '>': '&gt;', '"': '&quot;' }[c]));
const W = (id) => D.works[id];
const I = (n, c = '') => `<i data-lucide="${n}" class="ic ${c}" aria-hidden="true"></i>`;
const kindLabel = (w) => (w.type === 'movie' ? 'Movie' : 'Series');
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
const member = (id) => D.members.find((m) => m.id === id);

const MARK = `<svg class="mark" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" aria-hidden="true"><path d="M2.5 6.5c6 0 9.5 0 12.5 3.2"/><path d="M2.5 12h9.5"/><path d="M2.5 17.5c6 0 9.5 0 12.5-3.2"/><circle cx="18.6" cy="12" r="2.6" fill="currentColor" stroke="none"/></svg>`;

function art(w, cls = 'thumb', kind = 'poster') {
  return `<span class="art ${cls}"><img src="${w.art[kind]}" alt="" decoding="async" onerror="this.parentNode.classList.add('art-x');this.remove()"><span class="art-fb" aria-hidden="true"><b>${esc(initials(w.title))}</b><span>${esc(w.title)}</span></span></span>`;
}

const running = () => D.jobs.filter((j) => j.status === 'running');
const failed = () => D.jobs.filter((j) => j.status === 'failed');
const degraded = () => D.nodes.filter((n) => n.status === 'degraded');
const isDerivedJob = (j) => j.stage === 'Conversion' || (D.targets.find((t) => t.name === j.target) || {}).sourcing === 'derived';

function track(j, big = false) {
  const si = D.stages.indexOf(j.stage);
  const conv = isDerivedJob(j);
  const segs = D.stages.map((s, i) => {
    let st = i < si ? 'done' : i === si ? (j.status === 'failed' ? 'fail' : 'cur') : 'todo';
    if (s === 'Conversion' && !conv && i !== si) st += ' skip';
    const tip = s === 'Conversion' && !conv ? 'Conversion: not needed for this Target' : `${s}${i < si ? ': done' : i === si ? (j.status === 'failed' ? ': failed' : ': now') : ''}`;
    return `<span class="seg ${st}" title="${tip}">${st === 'cur' ? '<span class="fill"></span>' : ''}</span>`;
  }).join('');
  const w = W(j.work);
  const label = j.status === 'failed' ? `${w.title}, failed at ${j.stage}` : `${w.title}, ${j.stage}, stage ${si + 1} of ${D.stages.length}`;
  const live = j.status === 'running' ? `role="progressbar" aria-valuemin="0" aria-valuemax="100" data-live-progress="${j.id}"` : 'role="img"';
  return `<div class="trk${big ? ' trk-lg' : ''}" ${live} aria-label="${esc(label)}">${segs}</div>`;
}

const dot = (k) => `<span class="dot dot-${k}" aria-hidden="true"></span>`;

function sentence() {
  const n = running().length, f = failed().length, dg = degraded();
  return `<span class="s-live">${dot('draft')}${n} jobs moving</span><span class="s-sep" aria-hidden="true">·</span><a class="s-fail" href="#fail">${f} failure</a>${dg.length ? `<span class="s-sep" aria-hidden="true">·</span><a class="s-warn" href="#nodes">${esc(dg[0].name)} degraded</a>` : ''}`;
}

function rateCell(j) {
  if (j.rate != null) return `<span data-live-rate="${j.id}"></span>`;
  const m = /(\d+) releases/.exec(j.release || '');
  return m ? `<span class="t2">${m[1]} found</span>` : '<span class="t3">none</span>';
}
const etaCell = (j) => (j.etaMin != null ? `<span data-live-eta="${j.id}"></span>` : '<span class="t3">open</span>');

// ───────────────────────────── admin shell

const NAV = [
  { id: 'home', label: 'Home', icon: 'house' },
  { id: 'library', label: 'Library', icon: 'library', kids: [{ id: 'movies', label: 'Movies', count: () => D.library.movies }, { id: 'series', label: 'Series', count: () => D.library.series }] },
  { id: 'activity', label: 'Activity', icon: 'activity', live: () => running().length },
  { id: 'calendar', label: 'Calendar', icon: 'calendar' },
  { id: 'wanted', label: 'Wanted', icon: 'circle-dashed' },
  { id: 'requests', label: 'Requests', icon: 'inbox', count: () => D.requests.filter((r) => r.state === 'pending').length },
  { id: 'history', label: 'History', icon: 'history' },
  { id: 'manual-import', label: 'Manual import', icon: 'folder-input' },
  { sep: true },
  { id: 'nodes', label: 'Nodes', icon: 'network', count: () => D.nodes.length, flag: () => (degraded().length ? 'warn' : '') },
  { id: 'settings', label: 'Settings', icon: 'settings' },
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

// ───────────────────────────── admin home

function failureBlock() {
  const j = failed()[0];
  const w = W(j.work);
  const h = D.health.find((x) => x.level === 'error');
  const warn = D.health.find((x) => x.level === 'warn');
  return `<div class="fail" id="fail">
    <div class="fail-main">
      <div class="fail-h">
        <span class="fail-tag">${I('circle-x')}Failed at ${esc(j.stage)}</span>
        <span class="t3 num">${hm(h.at)} · ${ago(h.at)}</span>
        <div class="fail-act">
          <button class="btn" data-toast="Manual search is not in this prototype">${I('search')}Manual search</button>
          <button class="btn ghost" data-toast="Removing from the Blocklist is not in this prototype">${I('ban')}Remove from Blocklist</button>
          <button class="btn ghost" data-go="work">Open Work</button>
        </div>
      </div>
      <div class="fail-body">
        <div class="fail-id" data-row-go>${art(w, 'thumb-md')}<div><button class="lnk" data-go="work"><b>${esc(w.title)}</b></button><p class="t2">${esc(j.item)} · ${esc(j.target)}</p>${track(j)}</div></div>
        <dl class="fail-kv">
          <dt>Cause</dt><dd>${esc(j.cause)}</dd>
          <dt>Recovery</dt><dd>${esc(j.recovery)}</dd>
          <dt>Release</dt><dd><span class="mono rel">${esc(j.release)}</span><span class="chip chip-fail">${I('ban')}Blocklisted</span></dd>
        </dl>
      </div>
    </div>
    <div class="warn-line">
      <span class="warn-tag">${I('triangle-alert')}${esc(warn.stage)}</span>
      <p><b>${esc(warn.title)}.</b> <span class="t2">${esc(warn.body)}</span></p>
      <span class="t3 num">since ${hm(warn.at)}</span>
      <button class="btn ghost sm" data-toast="Testing a Node's connection is not in this prototype">Test connection</button>
    </div>
  </div>`;
}

function pipeline() {
  const rows = running().map((j) => {
    const w = W(j.work);
    return `<div class="jrow" role="row" data-row-go title="${esc(j.release || '')}">
      <div class="c-work" role="cell">${art(w)}<button class="lnk" data-go="work"><b>${esc(w.title)}</b></button><span class="t3 ell">${esc(j.item)}</span></div>
      <div class="c-tgt" role="cell">${esc(j.target)}</div>
      <div class="c-stage" role="cell">${esc(j.stage)}</div>
      <div class="c-trk" role="cell">${track(j)}</div>
      <div class="c-pct num" role="cell" data-live-pct="${j.id}"></div>
      <div class="c-node" role="cell"><span class="ell">${esc(j.node)}</span></div>
      <div class="c-rate num" role="cell">${rateCell(j)}</div>
      <div class="c-eta num" role="cell">${etaCell(j)}</div>
      <div class="c-menu" role="cell"><button class="ib" data-menu="${j.id}" aria-haspopup="menu" aria-label="Actions for ${esc(w.title)} ${esc(j.item)}">${I('ellipsis')}</button></div>
    </div>`;
  }).join('');
  return `<section class="pipe" aria-labelledby="pipe-h">
    <div class="sec-h"><h2 id="pipe-h">Live pipeline</h2><span class="t3 num">${running().length} running · 1 failed above</span><span class="grow"></span>
      <span class="legend" aria-hidden="true"><span class="lg"><span class="seg done"></span>Done</span><span class="lg"><span class="seg cur"></span>Now</span><span class="lg"><span class="seg todo"></span>Ahead</span><span class="lg"><span class="seg skip"></span>Not needed</span></span>
      <button class="btn ghost sm" data-nav="activity" data-label="Activity">Open Activity</button></div>
    <div class="ptable" role="table" aria-label="Running jobs">
      <div class="jrow jhead" role="row"><div role="columnheader">Work</div><div role="columnheader">Target</div><div role="columnheader">Stage</div><div role="columnheader">Searching to Available</div><div role="columnheader" class="num">%</div><div role="columnheader">Node</div><div role="columnheader">Rate</div><div role="columnheader">ETA</div><div role="columnheader"><span class="sr">Actions</span></div></div>
      ${rows}
    </div>
  </section>`;
}

function nodesPanel() {
  return `<section class="pnl pnl-nodes" id="nodes" aria-labelledby="nodes-h">
    <div class="pnl-h"><h2 id="nodes-h">Nodes</h2><span class="t3 num">${D.nodes.length} paired · ${degraded().length} degraded</span><span class="grow"></span><button class="btn ghost sm" data-toast="Pairing a Node is not in this prototype">${I('plus')}Pair a Node</button></div>
    <div class="nlist">${D.nodes.map((n) => {
      const busy = D.jobs.some((j) => j.status === 'running' && j.node.includes(n.name));
      const k = n.status === 'degraded' ? 'warn' : busy ? 'draft' : 'idle';
      return `<div class="nrow">
        <span class="nstate" title="${n.status === 'degraded' ? 'Degraded' : busy ? 'Working' : 'Online, idle'}">${dot(k)}</span>
        <div class="nname"><b>${esc(n.name)}</b><span class="t3">${n.kind} · ${esc(n.where)}</span></div>
        <div class="nroles">${n.roles.map((r) => `<span class="chip">${esc(r)}</span>`).join('')}</div>
        <div class="ncpu num"><span class="t3">CPU</span> ${n.cpu}%</div>
        <div class="ndet ${n.status === 'degraded' ? 'warn-t' : 't2'}">${esc(n.detail)}${n.gpu ? `<span class="t3 mono">${esc(n.gpu)}</span>` : ''}</div>
      </div>`;
    }).join('')}</div>
  </section>`;
}

function storagePanel() {
  const s = D.storage, pct = s.usedTB / s.totalTB, t = D.today;
  return `<section class="pnl pnl-store" aria-labelledby="store-h">
    <div class="pnl-h"><h2 id="store-h">Storage</h2><span class="t3 mono">${esc(s.root)}</span></div>
    <div class="store">
      <p class="store-n num"><b>${s.usedTB} TB</b><span class="t3"> of ${s.totalTB} TB · ${(s.totalTB - s.usedTB).toFixed(1)} TB free</span></p>
      <div class="meter" role="meter" aria-valuemin="0" aria-valuemax="${s.totalTB}" aria-valuenow="${s.usedTB}" aria-label="Library storage used"><span style="width:${(pct * 100).toFixed(1)}%"></span></div>
      <dl class="kv3 num"><div><dt>Movies</dt><dd>${D.library.movies}</dd></div><div><dt>Series</dt><dd>${D.library.series}</dd></div><div><dt>Episodes</dt><dd>${D.library.episodes.toLocaleString('en-US')}</dd></div></dl>
    </div>
    <div class="pnl-h pnl-h2"><h2>Today</h2></div>
    <dl class="kv3 kv4 num"><div><dt>Imported</dt><dd>${t.imported}</dd></div><div><dt>Grabbed</dt><dd>${t.grabbed}</dd></div><div><dt>Conversions</dt><dd>${t.conversions}</dd></div><div><dt>Failed</dt><dd class="fail-t">${t.failed}</dd></div></dl>
  </section>`;
}

function upNextPanel() {
  return `<section class="pnl" aria-labelledby="up-h"><div class="pnl-h"><h2 id="up-h">Up next</h2><span class="t3">airing this week</span><span class="grow"></span><button class="btn ghost sm" data-nav="calendar" data-label="Calendar">Calendar</button></div>
    <ul class="list">${D.upNext.map((u) => { const w = W(u.work); return `<li class="lrow" data-row-go><span class="when num"><b>${u.label}</b><span class="t3">${hm(u.at)}</span></span>${art(w)}<span class="ltxt"><button class="lnk" data-go="work"><b>${esc(w.title)}</b></button><span class="t3 ell">${esc(u.item)}</span></span></li>`; }).join('')}</ul></section>`;
}

function requestsPanel() {
  const rs = [...D.requests].sort((a, b) => (a.state === 'pending' ? -1 : 1) - (b.state === 'pending' ? -1 : 1));
  return `<section class="pnl" aria-labelledby="req-h"><div class="pnl-h"><h2 id="req-h">Requests</h2><span class="t3 num">${D.requests.filter((r) => r.state === 'pending').length} waiting</span><span class="grow"></span><button class="btn ghost sm" data-nav="requests" data-label="Requests">All requests</button></div>
    <ul class="list">${rs.map((r) => { const w = W(r.work), m = member(r.by); return `<li class="lrow req" data-req-row="${r.id}"><span class="avatar sm" title="${esc(m.name)}">${m.initials}</span><span class="ltxt"><span class="ell"><b>${esc(w.title)}</b>${r.note && r.state === 'pending' ? ` <span class="t2">${esc(r.note)}</span>` : ''}</span><span class="t3 ell">${esc(m.name)} · ${ago(r.at)}${r.state === 'approved' ? ` · ${esc(r.note || 'Approved')}` : ''}</span></span>${r.state === 'pending' ? `<span class="req-act"><button class="btn sm" data-approve="${r.id}">Approve</button><button class="ib sm" data-decline="${r.id}" aria-label="Decline ${esc(w.title)} for ${esc(m.name)}">${I('x')}</button></span>` : `<span class="t3 sm-tag">${I('check')}Approved</span>`}</li>`; }).join('')}</ul></section>`;
}

function recentPanel() {
  return `<section class="pnl" aria-labelledby="rec-h"><div class="pnl-h"><h2 id="rec-h">Recently added</h2><span class="grow"></span><button class="btn ghost sm" data-nav="library" data-label="Library">Library</button></div>
    <ul class="list list-2">${D.recentlyAdded.slice(0, 4).map((id) => { const w = W(id); return `<li class="lrow" data-row-go>${art(w)}<span class="ltxt"><button class="lnk" data-go="work"><b class="ell">${esc(w.title)}</b></button><span class="t3 num">${w.year} · ${kindLabel(w)}</span></span></li>`; }).join('')}</ul></section>`;
}

function adminHomeWide() {
  return `<h1 class="sr">Home</h1>
    <section class="pulse" aria-label="System status"><p class="sentence">${sentence()}</p>
      <p class="pulse-r t3">Today <b class="num">${D.today.imported}</b> imported<span class="s-sep">·</span><b class="num">${D.today.grabbed}</b> grabbed<span class="s-sep">·</span><b class="num">${D.today.conversions}</b> Conversions<span class="s-sep">·</span><b class="num fail-t">${D.today.failed}</b> failed</p></section>
    ${failureBlock()}
    ${pipeline()}
    <div class="board">${nodesPanel()}${storagePanel()}${upNextPanel()}${requestsPanel()}${recentPanel()}</div>`;
}

// ───────────────────────────── admin work

function matrixWide(w, job) {
  const [met, conv] = w.targets;
  const via = conv.via.split(' · ');
  return `<table class="mx">
    <caption class="sr">Edition by Target state for ${esc(w.title)}</caption>
    <thead><tr><th scope="col" class="mx-corner"><span>Edition</span>${I('move-right')}<span>Target</span></th>
      <th scope="col"><div class="mx-th"><b>2160p HDR</b><span class="t3">${I('download')}Downloaded</span></div></th>
      <th scope="col"><div class="mx-th"><b>1080p SDR</b><span class="t3">${I('replace')}Derived from 2160p HDR</span></div></th></tr></thead>
    <tbody>
      <tr><th scope="row" class="mx-ed"><b>Standard</b><span class="t3">Profile default</span></th>
        <td class="cell met">
          <div class="cell-h"><span class="badge b-met">${I('check')}Met</span><span class="t2">Cutoff reached, no upgrades</span></div>
          <dl class="kv">
            <dt>Score</dt><dd class="num">${met.score}</dd>
            <dt>Video</dt><dd class="mono">${esc(met.video)}</dd>
            <dt>Audio</dt><dd class="mono">${esc(met.audio)}</dd>
            <dt>Size</dt><dd class="num">${esc(met.size)}</dd>
          </dl>
          <p class="cell-foot t3">${I('arrow-up')}${esc(met.seeding)}</p>
        </td>
        <td class="cell conv">
          <div class="cell-h"><span class="badge b-live">${dot('draft')}Converting on ${esc(via[0])}</span><span class="num cell-pct" data-live-pct="${job.id}"></span></div>
          <div class="bar" role="progressbar" aria-valuemin="0" aria-valuemax="100" aria-label="1080p SDR Conversion progress" data-live-progress="${job.id}"><span class="fill"></span></div>
          <p class="cell-stats num"><span data-live-rate="${job.id}"></span><span class="s-sep">·</span><span data-live-eta="${job.id}"></span> left<span class="s-sep">·</span>≈ ${esc(conv.estSize)}</p>
          <dl class="kv">
            <dt>Source</dt><dd>${esc(conv.from)}</dd>
            <dt>Settings</dt><dd class="mono">${esc(via.slice(1).join(' · '))}</dd>
          </dl>
          <p class="cell-foot"><button class="btn ghost sm" data-toast="Pausing a Conversion is not in this prototype">${I('pause')}Pause Conversion</button></p>
        </td></tr>
      <tr class="mx-add"><td colspan="3"><button class="btn ghost sm" data-toast="Editing Editions is not in this prototype">${I('plus')}Want another Edition</button><span class="t3">Theatrical, Extended or Director's, each wanted at both Targets in Cinema</span></td></tr>
    </tbody></table>`;
}

function filesSection(w, job) {
  const met = w.targets[0];
  return `<section class="wk-sec" aria-labelledby="files-h"><div class="sec-h"><h2 id="files-h">Files</h2><span class="t3 num">1 on disk · 1 being written</span></div>
    <div class="files">
      <div class="frow">
        <span class="ficon">${I('file-video')}</span>
        <div class="fmain"><p class="mono fname">${esc(met.file)}</p>
          <p class="fspec mono"><span>${esc(met.video)}</span><span>${esc(met.audio)}</span><span>Subtitles EN, ES</span></p></div>
        <div class="fmeets"><span class="chip">${I('check')}2160p HDR</span></div>
        <div class="fsize num">${esc(met.size)}</div>
      </div>
      <div class="frow pending">
        <span class="ficon">${I('file-clock')}</span>
        <div class="fmain"><p class="fname">1080p SDR file <span class="t3">· being written by the Conversion on forge</span></p>
          <p class="fspec mono"><span>HEVC · SDR</span><span>from the 2160p HDR file</span><span class="draft-t" data-live-pct="${job.id}"></span></p></div>
        <div class="fmeets"><span class="chip chip-live">${dot('draft')}1080p SDR</span></div>
        <div class="fsize num">≈ ${esc(w.targets[1].estSize)}</div>
      </div>
    </div>
    <div class="subs"><h3>Subtitles</h3>${w.subtitles.map((s) => `<span class="sub">${I('captions')}<b>${esc(s.lang)}</b><span class="mono t2">${esc(s.note)}</span></span>`).join('')}<button class="btn ghost sm" data-toast="Subtitle search is not in this prototype">${I('search')}Find subtitles</button></div>
  </section>`;
}

const HIST_ICON = { grabbed: 'magnet', downloaded: 'download', transfer: 'arrow-right-left', imported: 'folder-check', conversion: 'replace' };

function historyTimeline(w) {
  const hs = [...w.history].reverse();
  return `<ol class="tl">${hs.map((h, i) => `<li class="tl-i${i === 0 ? ' live' : ''}"><span class="tl-dot">${I(HIST_ICON[h.kind])}</span><div class="tl-b"><p>${esc(h.text)}</p><p class="tl-m"><time class="num t3" datetime="${h.at}">${md(h.at)} · ${hm(h.at)}</time><span class="chip node">${I('server')}${esc(h.node)}</span>${i === 0 ? `<span class="draft-t num" data-live-pct="j-dune-conv"></span>` : ''}</p></div></li>`).join('')}</ol>`;
}

function profileBox() {
  return `<section class="rail-sec" aria-labelledby="prof-h"><div class="sec-h"><h2 id="prof-h">Profile</h2><span class="grow"></span><button class="btn ghost sm" data-toast="Editing a Profile is not in this prototype">Edit Profile</button></div>
    <p class="prof-n"><b>Cinema</b><span class="t3">Editions wanted: Standard</span></p>
    <ul class="prof-t"><li>${I('download')}<b>2160p HDR</b><span class="t3">downloaded</span></li><li>${I('replace')}<b>1080p SDR</b><span class="t3">derived from 2160p HDR</span></li></ul></section>`;
}

function workHeadWide(w, job) {
  return `<div class="wk-head">
    <div class="wk-bd" aria-hidden="true"><img src="${w.art.backdrop}" alt="" onerror="this.remove()"></div>
    ${art(w, 'poster-md')}
    <div class="wk-id">
      <h1>${esc(w.title)}</h1>
      <p class="meta t2"><span class="num">${w.year}</span> · ${kindLabel(w)} · ${w.runtime} · ${w.rating} · ${w.genres.join(', ')}</p>
      <p class="wk-state"><span>${I('check')}1 of 2 Targets met</span><span class="s-sep">·</span><span class="s-live">${dot('draft')}1080p SDR converting on forge, <span class="num" data-live-eta="${job.id}"></span> left</span></p>
      <p class="overview t2">${esc(w.overview)}</p>
    </div>
    <div class="wk-act">
      <button class="btn pri" data-toast="Searching every Target is not in this prototype">${I('search')}Search</button>
      <button class="btn" data-toast="Manual search is not in this prototype">${I('list-filter')}Manual search</button>
      <button class="btn" data-toast="Editing a Profile is not in this prototype">${I('sliders-horizontal')}Edit Profile</button>
      <button class="btn tog" data-monitor aria-pressed="true">${I('eye')}<span>Monitored</span></button>
      <button class="ib bordered" data-menu="work" aria-haspopup="menu" aria-label="More actions for ${esc(w.title)}">${I('ellipsis')}</button>
    </div>
  </div>`;
}

function adminWorkWide() {
  const w = W('dune-2'), job = D.jobs.find((j) => j.id === w.targets[1].job);
  return `${workHeadWide(w, job)}
    <div class="wk-grid">
      <div class="wk-main">
        <section class="wk-sec" aria-labelledby="mx-h"><div class="sec-h"><h2 id="mx-h">Editions and Targets</h2><span class="t3">Profile Cinema · 1 Edition × 2 Targets</span></div>${matrixWide(w, job)}</section>
        ${filesSection(w, job)}
      </div>
      <aside class="wk-rail">
        <section class="rail-sec" aria-labelledby="hist-h"><div class="sec-h"><h2 id="hist-h">History</h2><span class="grow"></span><button class="btn ghost sm" data-nav="history" data-label="History">Full history</button></div>${historyTimeline(w)}</section>
        ${profileBox()}
      </aside>
    </div>`;
}

// ───────────────────────────── admin phone

function phoneJob(j) {
  const w = W(j.work);
  return `<li class="pj" data-row-go>
    <div class="pj-top">${art(w, 'thumb-md')}<div class="pj-id"><button class="lnk" data-go="work"><b>${esc(w.title)}</b></button><span class="t3 ell">${esc(j.item)} · ${esc(j.target)}</span></div><span class="pj-pct num" data-live-pct="${j.id}"></span>
      <button class="ib tap" data-menu="${j.id}" aria-haspopup="dialog" aria-label="Actions for ${esc(w.title)}">${I('ellipsis')}</button></div>
    ${track(j, true)}
    <div class="pj-meta num"><span class="pj-stage">${esc(j.stage)}</span><span class="t3 ell">${esc(j.node)}</span><span class="grow"></span>${rateCell(j)}<span class="s-sep">·</span>${etaCell(j)}</div>
  </li>`;
}

function phoneFail() {
  const j = failed()[0], w = W(j.work), h = D.health.find((x) => x.level === 'error'), warn = D.health.find((x) => x.level === 'warn');
  return `<section class="pfail" id="fail" aria-label="Failure">
    <div class="pfail-h"><span class="fail-tag">${I('circle-x')}Failed at ${esc(j.stage)}</span><span class="t3 num">${ago(h.at)}</span></div>
    <div class="pfail-id" data-row-go>${art(w, 'thumb-md')}<div><button class="lnk" data-go="work"><b>${esc(w.title)}</b></button><p class="t2">${esc(j.item)} · ${esc(j.target)}</p></div></div>
    ${track(j, true)}
    <dl class="fail-kv"><dt>Cause</dt><dd>${esc(j.cause)}</dd><dt>Recovery</dt><dd>${esc(j.recovery)}</dd></dl>
    <p class="mono rel">${esc(j.release)}</p>
    <div class="pfail-act"><button class="btn" data-toast="Manual search is not in this prototype">${I('search')}Manual search</button><button class="btn ghost" data-menu="fail" aria-haspopup="dialog">More</button></div>
  </section>
  <section class="pwarn" aria-label="Warning"><span class="warn-tag">${I('triangle-alert')}${esc(warn.stage)}</span><p><b>${esc(warn.title)}.</b> <span class="t2">${esc(warn.body)}</span></p></section>`;
}

const TABS_ADMIN = [
  { id: 'home', label: 'Home', icon: 'house' },
  { id: 'library', label: 'Library', icon: 'library' },
  { id: 'activity', label: 'Activity', icon: 'activity', badge: () => running().length },
  { id: 'search', label: 'Search', icon: 'search' },
  { id: 'more', label: 'More', icon: 'menu' },
];
const TABS_PORTAL = [
  { id: 'home', label: 'Home', icon: 'house' },
  { id: 'search', label: 'Search', icon: 'search' },
  { id: 'requests', label: 'Requests', icon: 'list-checks' },
  { id: 'inbox', label: 'Inbox', icon: 'bell', badge: () => D.portal.inbox.length },
];

function tabs(list, active) {
  return `<nav class="tabs" aria-label="Tabs">${list.map((t) => `<button class="tab${t.id === active ? ' on' : ''}" data-tab="${t.id}" data-label="${t.label}" ${t.id === active ? 'aria-current="page"' : ''}>${I(t.icon)}<span>${t.label}</span>${t.badge ? `<span class="tab-b num">${t.badge()}</span>` : ''}</button>`).join('')}</nav>`;
}

function phoneSection(title, inner, extra = '') {
  return `<section class="psec"><div class="psec-h"><h2>${title}</h2>${extra}</div>${inner}</section>`;
}

function adminHomePhone() {
  return `<div class="statusbar" aria-hidden="true"></div>
  <header class="ph-head"><h1>Home</h1><span class="grow"></span><button class="ib tap" data-open="palette" aria-label="Search">${I('search')}</button><span class="avatar">${D.admin.initials}</span></header>
  <p class="ph-sentence">${sentence()}</p>
  ${phoneFail()}
  ${phoneSection('Live pipeline', `<ul class="pjs">${running().map(phoneJob).join('')}</ul>`, `<span class="t3 num">${running().length} running</span>`)}
  ${phoneSection('Nodes', `<ul class="glist">${D.nodes.map((n) => { const busy = D.jobs.some((j) => j.status === 'running' && j.node.includes(n.name)); const k = n.status === 'degraded' ? 'warn' : busy ? 'draft' : 'idle'; return `<li class="gl">${dot(k)}<span class="ltxt"><b>${esc(n.name)}</b><span class="t3 ell">${n.roles.join(', ')}</span></span><span class="${n.status === 'degraded' ? 'warn-t' : 't3'} gl-r num">${n.status === 'degraded' ? 'Degraded' : `CPU ${n.cpu}%`}</span></li>`; }).join('')}</ul>`, `<span class="t3">${degraded().length} degraded</span>`)}
  ${phoneSection('Requests', `<ul class="glist">${D.requests.filter((r) => r.state === 'pending').map((r) => { const w = W(r.work), m = member(r.by); return `<li class="gl" data-req-row="${r.id}"><span class="avatar sm">${m.initials}</span><span class="ltxt"><b class="ell">${esc(w.title)}${r.note ? ` <span class="t2">${esc(r.note)}</span>` : ''}</b><span class="t3">${esc(m.name)} · ${ago(r.at)}</span></span><button class="btn sm tapx" data-approve="${r.id}">Approve</button></li>`; }).join('')}</ul>`, `<span class="t3">2 waiting</span>`)}
  ${phoneSection('Up next', `<ul class="glist">${D.upNext.map((u) => { const w = W(u.work); return `<li class="gl" data-row-go>${art(w)}<span class="ltxt"><button class="lnk" data-go="work"><b>${esc(w.title)}</b></button><span class="t3 ell">${esc(u.item)}</span></span><span class="t2 gl-r num">${u.label} ${hm(u.at)}</span></li>`; }).join('')}</ul>`)}
  ${phoneSection('Recently added', `<div class="strip">${D.recentlyAdded.map((id) => { const w = W(id); return `<button class="strip-i" data-go="work">${art(w, 'poster-sm')}<span class="ell">${esc(w.title)}</span></button>`; }).join('')}</div>`)}
  ${phoneSection('Storage', `<div class="pstore"><p class="store-n num"><b>${D.storage.usedTB} TB</b><span class="t3"> of ${D.storage.totalTB} TB used</span></p><div class="meter" role="meter" aria-valuemin="0" aria-valuemax="${D.storage.totalTB}" aria-valuenow="${D.storage.usedTB}" aria-label="Library storage used"><span style="width:${((D.storage.usedTB / D.storage.totalTB) * 100).toFixed(1)}%"></span></div><p class="t3">Today <span class="num t2">${D.today.imported}</span> imported · <span class="num t2">${D.today.grabbed}</span> grabbed · <span class="num t2">${D.today.conversions}</span> Conversions · <span class="num fail-t">${D.today.failed}</span> failed</p></div>`)}
  ${tabs(TABS_ADMIN, 'home')}`;
}

function adminWorkPhone() {
  const w = W('dune-2'), job = D.jobs.find((j) => j.id === w.targets[1].job), [met, conv] = w.targets;
  const via = conv.via.split(' · ');
  return `<div class="statusbar" aria-hidden="true"></div>
  <header class="ph-nav"><button class="back tap" data-go="home">${I('chevron-left')}Home</button><span class="grow"></span><button class="ib tap" data-menu="work" aria-haspopup="dialog" aria-label="More actions for ${esc(w.title)}">${I('ellipsis')}</button></header>
  <div class="pw-top">
    <div class="wk-bd" aria-hidden="true"><img src="${w.art.backdrop}" alt="" onerror="this.remove()"></div>
    ${art(w, 'poster-md')}
    <div><h1>${esc(w.title)}</h1><p class="meta t2"><span class="num">${w.year}</span> · ${w.runtime} · ${w.rating}</p><p class="t2">${w.genres.join(', ')}</p></div>
  </div>
  <p class="wk-state ph-pad"><span>${I('check')}1 of 2 Targets met</span><span class="s-sep">·</span><span class="s-live">${dot('draft')}converting, <span class="num" data-live-eta="${job.id}"></span> left</span></p>
  <div class="pw-act ph-pad"><button class="btn pri" data-toast="Searching every Target is not in this prototype">${I('search')}Search</button><button class="btn" data-toast="Manual search is not in this prototype">${I('list-filter')}Manual search</button><button class="btn tog" data-monitor aria-pressed="true" aria-label="Monitored">${I('eye')}</button></div>
  ${phoneSection('Standard <span class="t3 h-sub">Edition</span>', `<ul class="glist mxl">
    <li class="mxi"><div class="mxi-h"><b>2160p HDR</b><span class="t3">${I('download')}downloaded</span><span class="grow"></span><span class="badge b-met">${I('check')}Met</span></div>
      <p class="t2">Cutoff reached · score <span class="num">${met.score}</span> · <span class="num">${esc(met.size)}</span></p><p class="mono t3 sm-l">${esc(met.video)} · ${esc(met.audio)}</p></li>
    <li class="mxi"><div class="mxi-h"><b>1080p SDR</b><span class="t3">${I('replace')}from 2160p HDR</span><span class="grow"></span><span class="badge b-live">${dot('draft')}Converting</span></div>
      <div class="bar" role="progressbar" aria-valuemin="0" aria-valuemax="100" aria-label="1080p SDR Conversion progress" data-live-progress="${job.id}"><span class="fill"></span></div>
      <p class="num t2 cell-stats"><span data-live-pct="${job.id}"></span><span class="s-sep">·</span><span data-live-rate="${job.id}"></span><span class="s-sep">·</span><span data-live-eta="${job.id}"></span> left<span class="s-sep">·</span>${esc(via[0])}</p><p class="mono t3 sm-l">${esc(via.slice(1).join(' · '))}</p></li>
  </ul>`, '<span class="t3">Profile Cinema</span>')}
  ${phoneSection('Files', `<ul class="glist"><li class="pfile"><p class="mono fname">${esc(met.file)}</p><p class="t3 num">${esc(met.size)} · meets 2160p HDR · subtitles ${w.subtitles.map((s) => `${s.lang} ${s.note.replace('synced · ', '')}`).join(', ')}</p></li></ul>`)}
  ${phoneSection('History', historyTimeline(w))}
  ${tabs(TABS_ADMIN, 'home')}`;
}

// ───────────────────────────── portal

function memberState(id) {
  if (requested.has(id)) return 'requested';
  const mine = D.portal.mine.find((m) => m.work === id);
  if (mine) return mine.state;
  const pend = D.requests.find((r) => r.work === id);
  if (pend) return pend.state === 'pending' ? 'requested' : 'coming';
  if (D.portal.newInLibrary.includes(id) || D.recentlyAdded.includes(id)) return 'ready';
  if (D.jobs.some((j) => j.work === id && j.status === 'running')) return 'coming';
  return { 'the-batman': 'ready', got: 'ready', 'the-boys': 'ready' }[id] || 'none';
}

function statePill(id) {
  const s = memberState(id);
  if (s === 'ready') return `<span class="state st-ready">${I('check')}Ready</span>`;
  if (s === 'coming') return `<span class="state st-coming">${dot('draft')}Coming</span>`;
  if (s === 'requested') return `<span class="state st-req">${I('clock')}Requested</span>`;
  return `<button class="btn req" data-req="${id}" aria-label="Request ${esc(W(id).title)}">${I('plus')}Request</button>`;
}

function portalTop(active) {
  const nav = [['home', 'Home'], ['requests', 'My requests'], ['library', 'Library']];
  return `<header class="p-top"><div class="p-in p-top-in">
    <span class="brand">${MARK}<span class="brand-w">SlipStream</span></span>
    <nav class="p-nav" aria-label="Main">${nav.map(([id, l]) => `<button class="p-nav-i${id === active ? ' on' : ''}" data-nav="${id}" data-label="${l}" ${id === active ? 'aria-current="page"' : ''}>${l}</button>`).join('')}</nav>
    <span class="grow"></span>
    <button class="ib bell" data-inbox aria-haspopup="dialog" aria-label="Inbox, ${D.portal.inbox.length} new">${I('bell')}<span class="bell-n num">${D.portal.inbox.length}</span></button>
    <span class="avatar" title="${esc(D.portal.member.name)}">${D.portal.member.initials}</span>
  </div></header>`;
}

function portalRow(m) {
  const w = W(m.work);
  const bar = m.state === 'coming' && m.job ? `<div class="bar thin" role="progressbar" aria-valuemin="0" aria-valuemax="100" aria-label="${esc(w.title)}: how close the next episode is" data-live-progress="${m.job}"><span class="fill"></span></div>` : '';
  return `<li class="prow" data-row-go>${art(w, 'poster-p')}<div class="prow-t"><button class="lnk" data-go="work"><b>${esc(w.title)}</b></button><p class="prow-line">${esc(m.line)}</p><p class="t3">${esc(m.detail)}</p>${bar}</div>${statePill(m.work)}</li>`;
}

function miniRow(id, sub) {
  const w = W(id);
  return `<li class="mrow" data-row-go>${art(w, 'poster-s')}<span class="ltxt"><button class="lnk" data-go="work"><b>${esc(w.title)}</b></button><span class="t3 num">${sub || `${w.year} · ${kindLabel(w)}`}</span></span>${statePill(id)}</li>`;
}

function portalSearchField() {
  return `<button class="p-search" data-open="palette" aria-haspopup="dialog">${I('search')}<span>Search movies and series</span><kbd>/</kbd></button>`;
}

function portalHomeWide() {
  return `${portalTop('home')}<main class="p-in p-main">
    <section class="p-hero"><h1>Request a movie or series</h1><p class="t2">Ask for anything. You will hear when it is ready to watch.</p>${portalSearchField()}</section>
    <section class="p-sec" aria-labelledby="mine-h"><div class="sec-h"><h2 id="mine-h">Your requests</h2><span class="t3 num">${D.portal.mine.length}</span></div><ul class="plist">${D.portal.mine.map(portalRow).join('')}</ul></section>
    <div class="p-cols">
      <section class="p-sec" aria-labelledby="new-h"><div class="sec-h"><h2 id="new-h">New in the library</h2></div><ul class="mlist">${D.portal.newInLibrary.map((id) => miniRow(id)).join('')}</ul></section>
      <section class="p-sec" aria-labelledby="pop-h"><div class="sec-h"><h2 id="pop-h">Popular right now</h2></div><ul class="mlist">${D.portal.popular.map((id) => miniRow(id)).join('')}</ul></section>
    </div>
  </main>`;
}

function portalWorkBody(w, phone) {
  const pw = D.portalWork;
  const like = ['interstellar', 'oppenheimer', 'the-matrix', 'the-prestige', 'the-dark-knight', 'eeaao'];
  return `<div class="pw-state"><span class="state st-ready lg">${I('check')}Ready</span><div><p class="pw-line">${esc(pw.line)}</p><p class="t2">${esc(pw.detail)}</p></div></div>
    <div class="pw-cta"><button class="btn pri lg" data-toast="Opening Plex is not in this prototype">${I('play')}Watch on Plex</button>${phone ? '' : '<button class="btn lg" data-toast="Reporting a problem is not in this prototype">Report a problem</button>'}</div>
    <section class="p-sec"><h2 class="h2s">About</h2><p class="pw-ov">${esc(w.overview)}</p></section>
    <section class="p-sec" aria-labelledby="like-h"><div class="sec-h"><h2 id="like-h">More like this</h2></div><ul class="mlist${phone ? '' : ' mlist-2'}">${like.map((id) => miniRow(id)).join('')}</ul></section>`;
}

function portalWorkWide() {
  const w = W('dune-2');
  return `${portalTop('library')}
    <div class="pw-hero"><div class="wk-bd" aria-hidden="true"><img src="${w.art.backdrop}" alt="" onerror="this.remove()"></div>
      <div class="p-in pw-head">${art(w, 'poster-lg')}<div class="pw-id"><button class="back-link" data-go="home">${I('chevron-left')}Home</button><h1>${esc(w.title)}</h1><p class="t2 meta"><span class="num">${w.year}</span> · Movie · ${w.runtime} · ${w.rating} · ${w.genres.join(', ')}</p></div></div></div>
    <main class="p-in p-main pw-main">${portalWorkBody(w, false)}</main>`;
}

function portalHomePhone() {
  return `<div class="statusbar" aria-hidden="true"></div>
  <header class="ph-head"><h1>Home</h1><span class="grow"></span><button class="ib tap bell" data-inbox aria-haspopup="dialog" aria-label="Inbox, ${D.portal.inbox.length} new">${I('bell')}<span class="bell-n num">${D.portal.inbox.length}</span></button><span class="avatar">${D.portal.member.initials}</span></header>
  <div class="ph-pad">${portalSearchField()}</div>
  ${phoneSection('Your requests', `<ul class="plist">${D.portal.mine.map(portalRow).join('')}</ul>`)}
  ${phoneSection('New in the library', `<div class="strip strip-p">${D.portal.newInLibrary.map((id) => { const w = W(id); return `<button class="strip-i" data-go="work">${art(w, 'poster-p2')}<span class="ell">${esc(w.title)}</span></button>`; }).join('')}</div>`)}
  ${phoneSection('Popular right now', `<ul class="mlist">${D.portal.popular.map((id) => miniRow(id)).join('')}</ul>`)}
  ${tabs(TABS_PORTAL, 'home')}`;
}

function portalWorkPhone() {
  const w = W('dune-2');
  return `<div class="statusbar" aria-hidden="true"></div>
  <header class="ph-nav"><button class="back tap" data-go="home">${I('chevron-left')}Home</button></header>
  <div class="pw-top">
    <div class="wk-bd" aria-hidden="true"><img src="${w.art.backdrop}" alt="" onerror="this.remove()"></div>
    ${art(w, 'poster-lg')}
    <div><h1>${esc(w.title)}</h1><p class="meta t2"><span class="num">${w.year}</span> · Movie · ${w.runtime}</p><p class="t2">${w.genres.join(', ')}</p></div>
  </div>
  <div class="ph-pad pw-ph">${portalWorkBody(w, true)}</div>
  ${tabs(TABS_PORTAL, 'home')}`;
}

// ───────────────────────────── overlays

function paletteItems(q) {
  q = q.trim().toLowerCase();
  const works = Object.values(D.works).map((w) => ({ kind: 'work', id: w.id, label: w.title, meta: `${w.year} · ${kindLabel(w)}`, w }));
  const cmds = isAdmin ? [
    { label: 'Add a Work', icon: 'plus', meta: 'Library' },
    { label: 'Manual search for The Batman', icon: 'search', meta: 'Failed at Import' },
    { label: 'Search every missing Item', icon: 'circle-dashed', meta: 'Wanted' },
    { label: 'Open Activity', icon: 'activity', meta: `${running().length} running` },
    { label: 'Create a Join Code to pair a Node', icon: 'network', meta: 'Nodes' },
    { label: 'Issue a Signup link', icon: 'user-round-plus', meta: 'Accounts' },
    { label: 'Test connection on relay-fra', icon: 'radio-tower', meta: 'Degraded' },
    ...(isPhone ? [] : [{ label: 'Toggle sidebar', icon: 'panel-left', meta: 'View', act: 'collapse' }]),
  ].map((c) => ({ ...c, kind: 'cmd' })) : [];
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
    const tail = x.kind === 'work' && !isAdmin ? `<span class="pal-state">${statePill(x.id).replace(/<button[^>]*>/, '<span class="state st-none">').replace('</button>', '</span>')}</span>` : x.kind === 'work' ? '' : `<span class="t3 pal-meta">${esc(x.meta)}</span>`;
    return `<li role="option" id="${id}" class="pal-o" data-kind="${x.kind}" data-label="${esc(x.label)}" ${x.act ? `data-act="${x.act}"` : ''} aria-selected="false">${lead}<span class="pal-l">${esc(x.label)}</span>${tail}${x.kind === 'work' && isAdmin ? `<span class="t3 pal-meta">${esc(x.meta)}</span>` : ''}</li>`;
  };
  const groups = [];
  if (ws.length) groups.push(`<li role="presentation" class="pal-g">${isAdmin ? 'Works' : q ? 'Results' : 'Try one of these'}</li>${ws.map(row).join('')}`);
  if (cs.length) groups.push(`<li role="presentation" class="pal-g">Commands</li>${cs.map(row).join('')}`);
  list.innerHTML = groups.join('') || `<li class="pal-empty" role="presentation">Nothing matches “${esc(q)}”. ${isAdmin ? 'Try a Work, or a command such as Add a Work.' : 'Try another spelling, or search by year.'}</li>`;
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

function closeAll() {
  root.querySelectorAll('.pal, .sheet-wrap, .menu, .pop').forEach((el) => { el.hidden = true; });
  if (lastFocus && document.contains(lastFocus)) lastFocus.focus({ preventScroll: true });
  lastFocus = null;
}
const anyOpen = () => [...root.querySelectorAll('.pal, .sheet-wrap, .menu, .pop')].some((el) => !el.hidden);

let toastT;
function toast(msg) {
  const t = root.querySelector('.toast');
  t.innerHTML = `${I('info')}<span>${esc(msg)}.</span>`;
  C.icons();
  t.classList.add('on');
  clearTimeout(toastT);
  toastT = setTimeout(() => t.classList.remove('on'), 2800);
}

function actionsFor(key) {
  if (key === 'work') return { title: 'Dune: Part Two', items: [['Rename files', 'file-pen'], ['Edit Editions', 'layers'], ['Refresh metadata', 'refresh-cw'], ['Remove Work', 'trash-2', 'danger']] };
  if (key === 'fail') return { title: 'The Batman · Import failed', items: [['Open Work', 'arrow-up-right', 'go'], ['Remove from Blocklist', 'ban'], ['Show log', 'scroll-text'], ['Dismiss failure', 'x']] };
  const j = D.jobs.find((x) => x.id === key), w = W(j.work);
  return { title: `${w.title} · ${j.item}`, sub: `${j.stage} on ${j.node}`, items: [['Open Work', 'arrow-up-right', 'go'], ['Pause job', 'pause'], ['Show log', 'scroll-text'], ['Cancel job', 'circle-x', 'danger']] };
}

function openMenu(btn) {
  const a = actionsFor(btn.dataset.menu);
  closeAll();
  lastFocus = btn;
  const items = a.items.map(([l, ic, k]) => `<button class="mi${k === 'danger' ? ' danger' : ''}" role="menuitem" data-mi="${k === 'go' ? 'go' : ''}" data-label="${esc(l)}">${I(ic)}<span>${esc(l)}</span></button>`).join('');
  if (isPhone) {
    const s = root.querySelector('.sheet-wrap');
    s.querySelector('.sheet').innerHTML = `<div class="sheet-grp" role="menu" aria-label="${esc(a.title)}"><p class="sheet-t"><b>${esc(a.title)}</b>${a.sub ? `<span class="t3">${esc(a.sub)}</span>` : ''}</p>${items}</div><button class="sheet-x" data-close>Cancel</button>`;
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

function openInbox(btn) {
  closeAll();
  lastFocus = btn;
  const body = `<p class="sheet-t"><b>Inbox</b><span class="t3">${D.portal.inbox.length} new</span></p>${D.portal.inbox.map((n) => `<button class="inb" data-go="work">${I('circle-check')}<span><b>${esc(n.text)}</b><span class="t3">${ago(n.at)}</span></span></button>`).join('')}`;
  if (isPhone) {
    const s = root.querySelector('.sheet-wrap');
    s.querySelector('.sheet').innerHTML = `<div class="sheet-grp" role="dialog" aria-label="Inbox">${body}</div><button class="sheet-x" data-close>Done</button>`;
    s.hidden = false;
  } else {
    const p = root.querySelector('.pop');
    p.innerHTML = body;
    p.hidden = false;
    const r = btn.getBoundingClientRect(), rr = root.getBoundingClientRect();
    p.style.top = `${r.bottom - rr.top + 8}px`;
    p.style.left = `${r.right - rr.left - 340}px`;
  }
  C.icons();
  root.querySelector(isPhone ? '.sheet .inb' : '.pop .inb').focus({ preventScroll: true });
}

function toggleSidebar() {
  const c = root.classList.toggle('collapsed');
  const b = root.querySelector('[data-collapse]');
  if (b) { b.setAttribute('aria-expanded', String(!c)); b.setAttribute('aria-label', c ? 'Expand sidebar' : 'Collapse sidebar'); b.innerHTML = I(c ? 'panel-left-open' : 'panel-left-close'); C.icons(); }
  try { localStorage.setItem('cr-collapsed', c ? '1' : ''); } catch { /* storage unavailable */ }
}

function overlays() {
  return `<div class="pal" hidden><div class="pal-scrim" data-close></div>
      <div class="pal-box" role="dialog" aria-modal="true" aria-label="${isAdmin ? 'Search Works and commands' : 'Search movies and series'}">
        <div class="pal-top">${I('search')}<input class="pal-in" type="text" role="combobox" aria-expanded="true" aria-controls="pal-list" aria-autocomplete="list" placeholder="${isAdmin ? 'Search Works or type a command' : 'Search movies and series'}" autocomplete="off" spellcheck="false">${isPhone ? '<button class="pal-cancel" data-close>Cancel</button>' : '<kbd>Esc</kbd>'}</div>
        <ul class="pal-list" id="pal-list" role="listbox" aria-label="Results"></ul>
        ${isPhone ? '' : `<div class="pal-foot t3"><span><kbd>↑</kbd><kbd>↓</kbd> move</span><span><kbd>↵</kbd> open</span><span><kbd>Esc</kbd> close</span></div>`}
      </div></div>
    <div class="menu" role="menu" hidden></div>
    <div class="pop" hidden></div>
    <div class="sheet-wrap" hidden><div class="sheet-scrim" data-close></div><div class="sheet"></div></div>
    <div class="toast" role="status" aria-live="polite"></div>`;
}

// ───────────────────────────── wiring

function wire() {
  root.addEventListener('click', (e) => {
    const t = e.target;
    const q = (s) => t.closest(s);
    let el;
    if ((el = q('[data-close]'))) return closeAll();
    if ((el = q('.mi'))) { if (el.dataset.mi === 'go') return C.go('work'); closeAll(); return toast(`${el.dataset.label} is not in this prototype`); }
    if ((el = q('.pal-o'))) return activate(el);
    if ((el = q('[data-open="palette"]'))) return openPalette();
    if ((el = q('[data-menu]'))) { e.stopPropagation(); return openMenu(el); }
    if ((el = q('[data-inbox]'))) return openInbox(el);
    if ((el = q('[data-collapse]'))) return toggleSidebar();
    if ((el = q('[data-monitor]'))) {
      const on = el.getAttribute('aria-pressed') !== 'true';
      el.setAttribute('aria-pressed', String(on));
      const l = el.querySelector('span'); if (l) l.textContent = on ? 'Monitored' : 'Not monitored';
      return toast(on ? 'Dune: Part Two is monitored again' : 'Dune: Part Two is no longer monitored; nothing new will be grabbed');
    }
    if ((el = q('[data-req]'))) {
      const id = el.dataset.req; requested.add(id);
      root.querySelectorAll(`[data-req="${id}"]`).forEach((b) => { b.outerHTML = `<span class="state st-req">${I('clock')}Requested</span>`; });
      C.icons();
      return toast(`Requested ${W(id).title}. You will hear when it is coming`);
    }
    if ((el = q('[data-approve]'))) {
      const r = D.requests.find((x) => x.id === el.dataset.approve);
      el.closest('.req-act, .btn').outerHTML = `<span class="t3 sm-tag">${I('check')}Approved</span>`;
      C.icons();
      return toast(`Approved ${W(r.work).title} for ${member(r.by).name}. Searching now`);
    }
    if ((el = q('[data-decline]'))) { const r = D.requests.find((x) => x.id === el.dataset.decline); return toast(`Declining needs a reason for ${member(r.by).name}; not in this prototype`); }
    if ((el = q('[data-toast]'))) return toast(el.dataset.toast);
    if ((el = q('[data-go]'))) return C.go(el.dataset.go);
    if ((el = q('[data-nav]'))) {
      if (el.dataset.nav === 'home') return C.go('home');
      return toast(`${el.dataset.label} is not in this prototype`);
    }
    if ((el = q('[data-tab]'))) {
      const id = el.dataset.tab;
      if (id === 'home') return C.go('home');
      if (id === 'search') return openPalette();
      if (id === 'inbox') return openInbox(el);
      if (id === 'more') return openMore(el);
      return toast(`${el.dataset.label} is not in this prototype`);
    }
    if ((el = q('[data-row-go]')) && !q('button, a')) return C.go('work');
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
    if (e.key === '/' && !isAdmin && !e.target.closest('input')) { e.preventDefault(); return openPalette(); }
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
  s.querySelector('.sheet').innerHTML = `<div class="sheet-grp" role="menu" aria-label="More">${rest.map((n) => `<button class="mi" role="menuitem" data-label="${n.label}">${I(n.icon)}<span>${n.label}</span>${n.count ? `<span class="t3 num mi-r">${n.flag && n.flag() ? dot(n.flag()) : ''}${n.count()}</span>` : ''}</button>`).join('')}</div><button class="sheet-x" data-close>Cancel</button>`;
  s.hidden = false;
  C.icons();
  s.querySelector('.mi').focus({ preventScroll: true });
}

export function render(ctx) {
  C = ctx; D = ctx.data; root = ctx.root;
  isPhone = ctx.device === 'phone';
  isAdmin = ctx.surface === 'admin';
  root.classList.add('cr', `cr-${ctx.surface}`, `cr-${ctx.device}`, `cr-${ctx.screen}`);
  document.documentElement.classList.add('cr-doc', `cr-doc-${ctx.surface}`);
  document.title = `${ctx.screen === 'work' ? 'Dune: Part Two' : 'Home'} · SlipStream`;
  try { if (!isPhone && isAdmin && localStorage.getItem('cr-collapsed')) root.classList.add('collapsed'); } catch { /* storage unavailable */ }

  let body;
  if (isAdmin && !isPhone) {
    const work = ctx.screen === 'work';
    const crumbs = work ? `<button class="crumb" data-nav="library" data-label="Library">Library</button>${I('chevron-right', 'crumb-sep')}<button class="crumb" data-nav="movies" data-label="Movies">Movies</button>${I('chevron-right', 'crumb-sep')}<span class="crumb cur">Dune: Part Two</span>` : '<span class="crumb cur">Home</span>';
    body = `<div class="shell">${sidebar(work ? 'movies' : 'home')}<div class="mainc">${topbar(crumbs)}<main class="content">${work ? adminWorkWide() : adminHomeWide()}</main></div></div>`;
  } else if (isAdmin) body = ctx.screen === 'work' ? adminWorkPhone() : adminHomePhone();
  else if (!isPhone) body = ctx.screen === 'work' ? portalWorkWide() : portalHomeWide();
  else body = ctx.screen === 'work' ? portalWorkPhone() : portalHomePhone();

  root.innerHTML = body + overlays();
  if (root.classList.contains('collapsed')) { const b = root.querySelector('[data-collapse]'); if (b) { b.innerHTML = I('panel-left-open'); b.setAttribute('aria-expanded', 'false'); b.setAttribute('aria-label', 'Expand sidebar'); } }
  ctx.icons();
  wire();
}
