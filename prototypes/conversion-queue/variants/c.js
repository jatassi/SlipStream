// PROTOTYPE variant C · Timeline: the queue as a schedule across Nodes (time is the main axis); the backfill dialog as one editable sentence.

const LANE_H = 36, LANE_GAP = 4, ROW_PAD = 8, SPAN = 690, PX_MIN = 2.4;
let S, D, T0, NOW;
const q = { zoom: 'near', band: null, node: 'forge', pinned: null };
let st = null;

const clock = (m) => new Date(T0 + m * 60000).toTimeString().slice(0, 5);
const mins = (iso) => (new Date(iso) - T0) / 60000;
const A0 = 15, SPLIT = 105;
const frac = (m) => { m = Math.max(A0, Math.min(SPAN, m)); return q.zoom === 'even' ? (m - A0) / (SPAN - A0) : m <= SPLIT ? 0.5 * (m - A0) / (SPLIT - A0) : 0.5 + 0.5 * (m - SPLIT) / (SPAN - SPLIT); };
const pct = (m) => `${(frac(m) * 100).toFixed(3)}%`;
const wid = (a, b) => `${((frac(b) - frac(a)) * 100).toFixed(3)}%`;
const conv = (id) => D.conversions.find((c) => c.id === id);
const prio = (id) => D.priorities.find((p) => p.id === id).label;
const nodeName = (id) => S.node(id).name;
const rowH = (n) => Math.max(68, n * LANE_H + (n - 1) * LANE_GAP + ROW_PAD * 2);

function setup(api) {
  S = api; D = api.D;
  const t = new Date(D.now); t.setMinutes(0, 0, 0);
  T0 = +t; NOW = mins(D.now);
}

// ───────────────────────────── schedule model: Node → lanes of blocks, plus off ranges and backfill bands

function blocksFor(c) {
  const s = c.state;
  if (s.is === 'running') { const end = NOW + s.etaMin; return [{ c, kind: 'run', a: NOW - (s.etaMin * s.progress) / (1 - s.progress), b: end }]; }
  if (s.is === 'suspended') {
    const since = mins(s.since), remain = 22, done = (remain / (1 - s.progress)) * s.progress;
    return [{ c, kind: 'paused', a: since - done, b: NOW + remain, since, resume: NOW }];
  }
  if (s.is !== 'waiting') return [];
  if (s.reason === 'queued') return [{ c, kind: 'queued', a: mins(s.startsAt), b: mins(s.finishesAt) }];
  if (s.reason === 'source-conversion') return [{ c, kind: 'dep', a: mins(s.startsAt), b: mins(s.finishesAt) }];
  if (s.reason === 'disk-space') {
    const n = S.node(s.node), open = n.activeHours ? openAt(n) : NOW;
    return [{ c, kind: 'disk', a: open, b: open + 150 }];
  }
  return [];
}

function openAt(n) {
  const [h, m] = n.activeHours.from.split(':').map(Number);
  const d = new Date(T0); d.setHours(h, m, 0, 0); if (+d <= T0) d.setDate(d.getDate() + 1);
  return (d - T0) / 60000;
}
function offRanges(n) {
  if (!n.activeHours) return [];
  const a = openAt(n), [h] = n.activeHours.to.split(':').map(Number), b = a + ((h - Number(n.activeHours.from.slice(0, 2)) + 24) % 24) * 60;
  return [[0, a], [b, SPAN]].filter(([x, y]) => y > x);
}

function laneUp(blocks, slots) {
  const lanes = Array.from({ length: slots }, () => []);
  [...blocks].sort((x, y) => x.a - y.a).forEach((bk) => {
    const free = lanes.find((l) => !l.length || l[l.length - 1].b <= bk.a + 0.5) || lanes.reduce((m, l) => (l[l.length - 1].b < m[m.length - 1].b ? l : m));
    free.push(bk);
  });
  return lanes.sort((x, y) => x.some((b) => b.kind === 'paused') - y.some((b) => b.kind === 'paused'));
}

function schedule() {
  return D.nodes.map((n) => {
    const blocks = D.conversions.filter((c) => c.state.node === n.id).flatMap(blocksFor);
    const lanes = laneUp(blocks, n.slots);
    let cursor = Math.max(NOW, ...blocks.map((b) => b.b));
    const bands = [];
    if (n.status === 'online') {
      D.backfill.forEach((g) => {
        const count = g.nodes[n.id];
        if (!count) return;
        const total = Object.values(g.nodes).reduce((a, b) => a + b, 0);
        const dur = n.slots > 1 ? (count * (g.works ? 27 : 3)) / n.slots : 230;
        bands.push({ g, count, a: cursor, b: cursor + dur, gb: Math.round((g.estGB * count) / total) });
        cursor += dur;
      });
    }
    return { n, lanes, bands, off: offRanges(n) };
  });
}

// ───────────────────────────── shared bits

function unitLabel(g, count) {
  if (g.works) return `${count} ${count === 1 ? 'Work' : 'Works'}`;
  const w = S.W(g.items[0].work);
  return `${count} ${S.ct(w).levels.at(-1).label.toLowerCase()}s`;
}

function kindLine(c) {
  const t = S.target(c.target);
  if (c.kind === 'adjust') return { icon: 'wrench', short: 'Adjust', text: `Adjusts the downloaded file: ${c.steps.join(', ')}. The torrent keeps seeding the original.` };
  return { icon: 'replace', short: 'Derive', text: `Derives ${t.name} from the ${S.target(t.from).name} file.` };
}

function stateLine(c, bk) {
  const s = c.state, n = s.node && nodeName(s.node);
  if (s.is === 'running') return `Running on ${n} · <span class="num" data-live-pct="${c.id}"></span> · <span class="num" data-live-fps="${c.id}"></span>`;
  if (s.is === 'suspended') return `Paused on ${n} since ${S.hm(s.since)}. ${S.esc(s.cause)}. It resumes on its own when that stops; this is not a failure.`;
  if (s.reason === 'queued') return `Queued on ${n} · starts <span class="num">${S.hm(s.startsAt)}</span>`;
  if (s.reason === 'source-conversion') { const w = conv(s.waitsOn); return `Waiting for the source's Conversion (${S.esc(S.target(w.target).name)} adjust) · starts <span class="num">${S.hm(s.startsAt)}</span>`; }
  if (s.reason === 'disk-space') return `${S.esc(s.detail)}`;
  return '';
}

function finishLine(c, bk) {
  const s = c.state;
  if (s.is === 'running') return `<span class="num" data-live-eta="${c.id}"></span> left · ≈ <span class="num">${clock(bk.b)}</span>`;
  if (s.is === 'suspended') return `≈ <span class="num">${clock(bk.b)}</span> if Jellyfin stops now`;
  if (s.reason === 'disk-space') return `≈ <span class="num">${clock(bk.b)}</span> once space frees (atlas opens ${clock(bk.a)})`;
  return `≈ <span class="num">${S.hm(s.finishesAt)}</span>`;
}

function detail(c, bk) {
  const w = S.W(c.work), t = S.target(c.target), k = kindLine(c), sub = S.itemLabel(c);
  const settings = t.settings ? `<span class="mono">${S.esc(t.settings.summary)}</span> <span class="chip">v${c.settingsVersion}</span>` : '';
  return `<div class="cq-pd">
    <div class="cq-pd-h">${S.art(w, 'thumb-md')}<div><b>${S.esc(w.title)}</b>${sub ? `<span class="t2">${S.esc(sub)}</span>` : ''}<span class="t2">${S.esc(t.name)} · <span class="chip">${prio(c.priority)}</span></span></div></div>
    <p class="cq-pd-state">${stateLine(c, bk)}</p>
    <dl class="cq-kv">
      <dt>${S.I(k.icon)}${k.short}</dt><dd>${S.esc(k.text)}</dd>
      <dt>Settings</dt><dd>${settings}</dd>
      <dt>Source</dt><dd><span class="t2">${S.esc(c.source.name)}</span> · <span class="mono">${S.esc(c.source.spec)}</span></dd>
      <dt>Finishes</dt><dd>${finishLine(c, bk)}</dd>
    </dl>
  </div>`;
}

const actions = (c) => [
  ['Move up', 'arrow-up-to-line', '', `${S.W(c.work).title} moved up. It starts next on ${c.state.node ? nodeName(c.state.node) : 'the first capable Node'}.`],
  ['Pin Target to Node', 'pin', '', null],
  ['Cancel Conversion', 'x', 'danger', `${S.W(c.work).title} · ${S.target(c.target).name} Conversion cancelled`],
];

// ───────────────────────────── queue: wide timeline

function block(bk, top) {
  const c = bk.c, w = S.W(c.work), sub = S.itemLabel(c), k = kindLine(c);
  const status = {
    paused: `${S.I('pause', 'cq-kic')}paused at ${Math.floor(c.state.progress * 100)}%`,
    dep: `${S.I('link', 'cq-kic')}waits for source`,
    disk: bk.kind === 'disk' ? `${S.I('hard-drive', 'cq-kic')}needs ${c.state.needGB} GB, ${c.state.freeGB} free` : '',
  }[bk.kind];
  const line = status || `${S.I(k.icon, 'cq-kic')}${S.esc(sub || S.target(c.target).name)}${c.priority === 'bump' ? ` · ${S.I('arrow-up', 'cq-kic')}Moved up` : ''}`;
  const split = (m) => (((m - bk.a) / (bk.b - bk.a)) * 100).toFixed(2);
  const pausedPart = bk.kind === 'paused'
    ? `<span class="cq-done" style="width:${split(bk.since)}%"></span><span class="cq-hatch" style="left:${split(bk.since)}%;width:${(split(bk.resume) - split(bk.since)).toFixed(2)}%"></span>` : '';
  const live = bk.kind === 'run' ? `<div class="bar cq-bbar" role="progressbar" aria-valuemin="0" aria-valuemax="100" aria-label="${S.esc(w.title)} progress" data-live-progress="${c.id}"><span class="fill"></span></div>` : '';
  const label = `${w.title}${sub ? `, ${sub}` : ''}, ${S.target(c.target).name}, ${k.short.toLowerCase()}, ${bk.kind === 'run' ? 'running' : bk.kind}`;
  return `<button class="cq-blk k-${bk.kind} c-${c.kind}" style="top:${top}px;left:${pct(bk.a)};width:${wid(bk.a, bk.b)}" data-conv="${c.id}" data-work="${w.id}" aria-label="${S.esc(label)}">
    ${pausedPart}
    <span class="cq-bin">${S.art(w, 'thumb')}<span class="cq-bt"><b>${S.esc(w.title)}</b><span class="cq-bs">${line}</span></span></span>
    ${live}
  </button>`;
}

function laneTop(i) { return ROW_PAD + i * (LANE_H + LANE_GAP); }

function nodeTrack(r) {
  const { n, lanes, bands, off } = r, h = rowH(n.slots);
  const offline = n.status !== 'online';
  const offHatch = off.map(([a, b]) => `<div class="cq-off" style="left:${pct(a)};width:${wid(a, b)}"><span>${a === 0 ? `starts nothing new until ${n.activeHours.from}` : `ends ${n.activeHours.to}`}</span></div>`).join('');
  const offlineHatch = offline ? `<div class="cq-off cq-offline" style="left:0;width:100%"><span>${S.I('power-off')}offline since ${S.hm(n.offlineSince)} · 3 Conversions with no capable Node could run here</span></div>` : '';
  const laneLines = lanes.map((_, i) => `<div class="cq-lane" style="top:${laneTop(i)}px;height:${LANE_H}px"></div>`).join('');
  const blocks = lanes.flatMap((l, i) => l.map((bk) => block(bk, laneTop(i)))).join('');
  const arrows = [];
  lanes.forEach((l, i) => l.forEach((bk) => {
    if (bk.kind !== 'dep') return;
    const src = conv(bk.c.state.waitsOn);
    const j = lanes.findIndex((ll) => ll.some((x) => x.c === src));
    if (j < 0) return;
    const top = Math.min(laneTop(i), laneTop(j)) + LANE_H - 4, bot = Math.max(laneTop(i), laneTop(j)) + 4;
    arrows.push(`<div class="cq-dep ${j < i ? 'down' : 'up'}" style="left:${pct(bk.a)};top:${top}px;height:${bot - top}px" title="waiting for the source's Conversion"></div>`);
  }));
  const bandHtml = bands.map((b) => `<button class="cq-band${q.band === b.g.id ? ' on' : ''}" data-band="${b.g.id}" style="left:${pct(b.a)};width:${wid(b.a, b.b)};top:${ROW_PAD}px;height:${h - ROW_PAD * 2}px" aria-expanded="${q.band === b.g.id}" aria-label="Backfill ${S.esc(b.g.label)}, ${unitLabel(b.g, b.count)} on ${n.name}">
      <span class="cq-band-in"><span class="cq-band-1"><b>${unitLabel(b.g, b.count)}</b><span class="t2 cq-band-l">${S.esc(b.g.label)}</span></span><span class="cq-band-2 num t2">${S.fmtHours((b.b - b.a) / 60)} · ${b.gb} GB${S.I(q.band === b.g.id ? 'chevron-up' : 'chevron-down')}</span></span>
    </button>`).join('');
  return `<div class="cq-row${offline ? ' is-off' : ''}" style="height:${h}px">${laneLines}${offHatch}${offlineHatch}${bandHtml}${blocks}${arrows.join('')}</div>`;
}

function nodeLabel(r) {
  const n = r.n, h = rowH(n.slots), ms = n.mediaServer && n.mediaServer.transcoding;
  const status = n.status !== 'online' ? `${S.dot('idle')}<span class="t3">offline</span>` : ms ? `${S.dot('warn')}<span class="t2">Jellyfin transcoding</span>` : n.activeHours ? `${S.dot('idle')}<span class="t2">${n.activeHours.from}–${n.activeHours.to}</span>` : `${S.dot('draft')}<span class="t2">converting</span>`;
  return `<div class="cq-lab" style="height:${h}px">
    <div class="cq-lab-n"><b>${n.name}</b><span class="t3">${n.kind}</span></div>
    <div class="cq-lab-s">${status}</div>
    <div class="cq-lab-m t3"><span class="num">${n.cap}</span> at once</div>
    ${n.slots > 1 ? `<div class="cq-lab-lanes" aria-hidden="true">${r.lanes.map((_, i) => `<span style="top:${laneTop(i)}px;height:${LANE_H}px">${i + 1}</span>`).join('')}</div>` : ''}
  </div>`;
}

function axis() {
  const ticks = [];
  for (let m = 30; m <= SPAN; m += 15) {
    const dense = q.zoom !== 'even' && m <= SPLIT;
    if (!dense && m % 60) continue;
    if (dense && m % 30) continue;
    const mid = new Date(T0 + m * 60000).getHours() === 0;
    ticks.push(`<span class="cq-tick${mid ? ' mid' : ''}" style="left:${pct(m)}"><span class="num">${clock(m)}</span>${mid ? '<span class="t3">Fri</span>' : ''}</span>`);
  }
  return `<div class="cq-axis">${ticks.join('')}${q.zoom !== 'even' ? `<span class="cq-split" style="left:${pct(SPLIT)}" title="The scale changes here"></span>` : ''}</div>`;
}

function gridLines() {
  const l = [];
  for (let m = 30; m <= SPAN; m += 15) {
    const dense = q.zoom !== 'even' && m <= SPLIT;
    if (!dense && m % 30) continue;
    l.push(`<span class="cq-gl${m % 60 ? ' half' : ''}" style="left:${pct(m)}"></span>`);
  }
  return `<div class="cq-grid" aria-hidden="true">${l.join('')}</div>`;
}

function bandList() {
  if (!q.band) return '';
  const g = D.backfill.find((b) => b.id === q.band);
  let rows;
  if (g.works) {
    rows = g.works.map((id) => { const w = S.W(id); return `<li class="cq-bl" data-work="${id}">${S.art(w)}<span class="ell"><b>${S.esc(w.title)}</b></span><span class="t3">${S.esc(S.ct(w).size(w))}</span></li>`; }).join('');
  } else {
    const byWork = new Map();
    g.items.forEach((it) => { if (!byWork.has(it.work)) byWork.set(it.work, []); byWork.get(it.work).push(it); });
    rows = [...byWork].map(([id, its]) => {
      const w = S.W(id), t = S.ct(w), key = t.layout.collapse;
      const folds = its.map((it) => `<span class="chip">${S.esc(t.group({ [key]: it[key] }))} · <span class="num">${it.episodes}</span> ${t.levels.at(-1).label.toLowerCase()}s</span>`).join('');
      return `<li class="cq-bl wide" data-work="${id}">${S.art(w)}<span class="ell"><b>${S.esc(w.title)}</b></span><span class="cq-folds">${folds}</span></li>`;
    }).join('');
  }
  const split = Object.entries(g.nodes).map(([n, c]) => `${c} on ${n}`).join(' · ');
  return `<div class="cq-bandlist pnl-glass">
    <div class="sec-h"><span class="chip">Backfill</span><h2>${S.esc(g.label)}</h2><span class="t2 num">${split} · ≈ ${S.fmtHours(g.estHours)} · ${g.estGB} GB</span><span class="grow"></span><button class="btn sm ghost" data-toast="Cancelling a backfill group is not in this prototype">Cancel group</button><button class="ib sm" data-band="${g.id}" aria-label="Close">${S.I('x')}</button></div>
    <p class="t2">${S.esc(g.reason)} Backfill runs at the lowest priority, after everything else on the schedule.</p>
    <ul class="cq-bls">${rows}</ul>
  </div>`;
}

function summary() {
  const cs = D.conversions, by = (f) => cs.filter(f).length;
  const bf = D.backfill.reduce((n, g) => n + (g.works ? g.works.length : g.items.reduce((m, i) => m + i.episodes, 0)), 0);
  const end = new Date(+D.now + D.today.hoursQueued * 3600000); end.setMinutes(Math.ceil(end.getMinutes() / 10) * 10, 0, 0);
  const parts = [
    [by((c) => c.state.is === 'running'), 'running'],
    [by((c) => c.state.is === 'suspended'), 'paused'],
    [by((c) => c.state.is === 'waiting' && ['queued', 'source-conversion'].includes(c.state.reason)), 'queued'],
    [by((c) => c.state.reason === 'disk-space'), 'waiting for disk space'],
    [bf, 'backfill'],
  ].filter(([n]) => n).map(([n, l]) => `<span><b class="num">${n}</b> ${l}</span>`).join('<span class="s-sep">·</span>');
  const failed = by((c) => c.state.is === 'failed'), nocap = by((c) => c.state.reason === 'no-capable-node');
  return `<div class="cq-sum">
    <p class="cq-sum-h">Everything queued finishes by about <span class="num draft-t">${end.toTimeString().slice(0, 5)}</span> tomorrow.</p>
    <p class="cq-sum-c t2">${parts}${failed ? `<span class="s-sep">·</span><a href="#cq-off" class="fail-t"><b class="num">${failed}</b> failed</a>` : ''}${nocap ? `<span class="s-sep">·</span><a href="#cq-off" class="warn-t"><b class="num">${nocap}</b> with no capable Node</a>` : ''}<span class="s-sep">·</span><span><b class="num">${D.today.finished}</b> finished today</span></p>
  </div>`;
}

function schedHtml() {
  const rows = schedule();
  return `<div class="cq-tl">
    <div class="cq-labs"><div class="cq-axis-pad"></div>${rows.map(nodeLabel).join('')}</div>
    <div class="cq-scroll"><div class="cq-inner">
      ${axis()}${gridLines()}
      ${rows.map(nodeTrack).join('')}
      <div class="cq-now" style="left:${pct(NOW)}"><span class="num">${clock(NOW)}</span></div>
    </div></div>
  </div>
  ${bandList()}`;
}

function legend() {
  const sw = (cls, l) => `<span class="cq-lg"><span class="cq-sw ${cls}"></span>${l}</span>`;
  return `<div class="cq-legend t2">${sw('k-run', 'Running')}${sw('k-queued', 'Queued, estimated')}${sw('k-paused', 'Paused or outside active hours')}${sw('k-disk', 'Waiting for disk space')}${sw('k-band', 'Backfill')}<span class="cq-lg">${S.I('replace', 'cq-kic')}Derive</span><span class="cq-lg">${S.I('wrench', 'cq-kic')}Adjust</span></div>`;
}

// ───────────────────────────── Not on the schedule

function failCard() {
  const c = D.conversions.find((x) => x.state.is === 'failed');
  if (!c) return '';
  const w = S.W(c.work), t = S.target(c.target), s = c.state, last = s.attempts.at(-1);
  const nodesTried = [...new Set(s.attempts.map((a) => a.node))];
  const att = s.attempts.map((a, i) => `<li><span class="num cq-an">${i + 1}</span><span class="cq-anode"><b>${a.node}</b><span class="fail-dim num">${S.hm(a.at)}</span></span><code class="mono">${S.esc(a.error)}</code></li>`).join('');
  return `<div class="fail cq-failc">
    <div class="fail-main">
      <div class="fail-h"><span class="chip-fail">${S.I('circle-x')}Failed on ${nodesTried.length} Nodes</span><span class="fail-dim num">${S.hm(last.at)} · ${S.ago(last.at)}</span>
        <div class="fail-act"><button class="btn fail" data-toast="Retrying ${S.esc(w.title)} on forge">${S.I('rotate-ccw')}Retry</button><button class="btn" data-toast="Searching for a new source is not in this prototype">${S.I('search')}Search for a new source</button></div></div>
      <div class="cq-fail-b">
        <div class="fail-id">${S.art(w, 'thumb-md')}<div><b>${S.esc(w.title)}</b><p class="t2">${S.esc(t.name)} · ${kindLine(c).short} · ${S.esc(c.source.spec)}</p><p class="t2">${S.esc(prio(c.priority))} · settings v${c.settingsVersion}</p></div></div>
        <div>${S.isPhone ? `<details class="cq-attd"><summary>${s.attempts.length} attempts · ${s.attempts.map((a) => a.node).join(', ')}</summary><ol class="cq-att">${att}</ol></details>` : `<ol class="cq-att">${att}</ol>`}<p class="cq-verdict"><b>Verdict.</b> ${S.esc(s.verdict)} Nothing retries until the source changes or you press Retry.</p></div>
      </div>
    </div>
  </div>`;
}

function warnCards() {
  const groups = new Map();
  D.conversions.filter((c) => c.state.reason === 'no-capable-node').forEach((c) => { const k = c.state.cause; if (!groups.has(k)) groups.set(k, []); groups.get(k).push(c); });
  return [...groups].map(([cap, cs]) => {
    const could = D.nodes.filter((n) => n.proven.includes(cap));
    const failedTrial = D.nodes.map((n) => [n, (n.trials || []).find((t) => t.cap === cap && t.result === 'failed')]).filter(([, t]) => t);
    const first = cs.map((c) => c.state.since).sort()[0];
    const works = cs.map((c) => { const w = S.W(c.work); return `<li class="cq-wk" data-work="${w.id}">${S.art(w)}<span class="ell"><b>${S.esc(w.title)}</b><span class="warn-dim"> · ${S.esc(S.target(c.target).name)} · ${prio(c.priority)}</span></span><span class="warn-dim num">since ${S.hm(c.state.since)}</span></li>`; }).join('');
    const why = [
      ...could.map((n) => `<li><b>${n.name}</b> has proven it${n.status !== 'online' ? `, but it is offline since ${S.hm(n.offlineSince)}.` : '.'}</li>`),
      ...failedTrial.map(([n, t]) => `<li><b>${n.name}</b>'s trial failed ${S.ago(t.at)}: ${S.esc(t.note)}.</li>`),
    ].join('');
    return `<div class="cq-warnc">
      <div class="cq-warn-h"><span class="chip-warn">${S.I('triangle-alert')}No capable Node</span><b>${S.esc(D.capabilities[cap])}</b><span class="grow"></span><span class="warn-dim">Notified you once at <span class="num">${S.hm(first)}</span></span></div>
      <div class="cq-warn-b">
        <ul class="cq-wks">${works}</ul>
        <div class="cq-why"><p class="warn-dim">Which Node could</p><ul>${why}</ul>
          <div class="cq-warn-act">${could.filter((n) => n.status !== 'online').map((n) => `<button class="btn warn sm" data-toast="Wake-on-LAN sent to ${n.name}">${S.I('power')}Wake ${n.name}</button>`).join('')}${failedTrial.map(([n]) => `<button class="btn sm" data-toast="Trial run queued on ${n.name}">${S.I('flask-conical')}Rerun trial on ${n.name}</button>`).join('')}</div>
        </div>
      </div>
    </div>`;
  }).join('');
}

function offSchedule() {
  return `<section class="cq-offs" id="cq-off" aria-labelledby="cq-off-h">
    <div class="sec-h"><h2 id="cq-off-h">Not on the schedule</h2><span class="t3">These wait for you or for a Node, not for time.</span></div>
    <div class="cq-offs-g">${failCard()}${warnCards()}</div>
  </section>`;
}

// ───────────────────────────── queue: phone agenda

function phoneAgenda() {
  const rows = schedule(), r = rows.find((x) => x.n.id === q.node), n = r.n;
  const items = [...r.lanes.flat(), ...r.bands], lead = r.off.find(([a]) => a === 0);
  const last = items.length ? Math.max(...items.map((b) => b.b)) : NOW + 120;
  const gap = lead && items.length && Math.min(...items.map((b) => b.a)) >= lead[1] ? lead : null;
  const from = gap ? gap[1] - 30 : 20, to = Math.min(SPAN, last + 30), y = (m) => (m - from) * PX_MIN, H = y(to);
  const skip = gap ? `<div class="cq-ph-skip">${S.dot('idle')}<span>Now <span class="num">${clock(NOW)}</span>. ${n.name} starts nothing new until ${n.activeHours.from}, in ${S.fmtHours((gap[1] - NOW) / 60)}.</span></div>` : '';
  const hours = []; for (let m = Math.ceil(from / 60) * 60; m <= to; m += 60) hours.push(`<span class="cq-ph-hr" style="top:${y(m)}px"><span class="num">${clock(m)}</span></span>`);
  const cols = r.lanes.length;
  const colStyle = (i) => `left:calc(${(i / cols) * 100}% + 2px);width:calc(${100 / cols}% - 4px)`;
  const blocks = r.lanes.flatMap((l, i) => l.map((bk) => {
    const c = bk.c, w = S.W(c.work), k = kindLine(c);
    const top = Math.max(0, y(bk.a)), h = Math.max(30, y(bk.b) - top);
    const tag = bk.kind === 'paused' ? `${S.I('pause', 'cq-kic')}paused` : bk.kind === 'dep' ? `${S.I('link', 'cq-kic')}waits for source` : bk.kind === 'disk' ? `${S.I('hard-drive', 'cq-kic')}needs ${c.state.needGB} GB, ${c.state.freeGB} free` : `${S.I(k.icon, 'cq-kic')}<span class="num">${clock(bk.a)}</span>`;
    return `<button class="cq-pb k-${bk.kind}" style="top:${top}px;height:${h}px;${colStyle(i)}" data-conv="${c.id}" data-work="${w.id}">
      ${S.art(w)}<span class="cq-pb-t"><b>${S.esc(w.title)}</b><span class="t2">${tag}</span></span>
      ${bk.kind === 'run' ? `<div class="bar cq-bbar" data-live-progress="${c.id}"><span class="fill"></span></div>` : ''}
    </button>`;
  })).join('');
  const bands = r.bands.map((b) => `<button class="cq-pband" data-band="${b.g.id}" style="top:${y(b.a) + 4}px;height:${y(b.b) - y(b.a) - 8}px"><span class="cq-pband-1"><span class="chip">Backfill</span><b>${unitLabel(b.g, b.count)}</b><span class="num t2">${S.fmtHours((b.b - b.a) / 60)} · ${b.gb} GB</span></span><span class="t2 cq-pband-l">${S.esc(b.g.label)}</span></button>`).join('');
  const off = r.off.filter(([a, b]) => b > from && a < to).map(([a, b]) => `<div class="cq-poff" style="top:${y(Math.max(a, from))}px;height:${y(Math.min(b, to)) - y(Math.max(a, from))}px"><span>${a === 0 ? `starts nothing new until ${n.activeHours.from}` : `active hours end ${n.activeHours.to}`}</span></div>`).join('');
  const offline = n.status !== 'online' ? `<div class="cq-poff cq-offline" style="top:0;height:${H}px"><span>${S.I('power-off')}offline since ${S.hm(n.offlineSince)}</span></div>` : '';
  const seg = rows.map((x) => { const cnt = x.lanes.flat().length; return `<button role="radio" class="seg-o" data-pnode="${x.n.id}" aria-checked="${x.n.id === q.node}"><span>${x.n.name}</span><span class="seg-n num">${x.n.status !== 'online' ? 'off' : cnt}</span></button>`; }).join('');
  const ms = n.mediaServer && n.mediaServer.transcoding ? `<p class="cq-ph-note t2">${S.dot('warn')} Jellyfin on ${n.name} is ${S.esc(n.mediaServer.detail)}. Conversions here are paused and everything behind them slides.</p>` : '';
  return `<div class="psec">
    <div class="psec-h"><h2>Schedule</h2><span class="t3"><span class="num">${n.cap}</span> at once</span></div>
    <div class="seg-c" role="radiogroup" aria-label="Node">${seg}</div>
    ${ms}${skip}
    <div class="cq-ph-ag" style="height:${H + 20}px">
      <div class="cq-ph-times">${hours.join('')}</div>
      <div class="cq-ph-track" style="background-position:0 ${y(Math.ceil(from / 60) * 60)}px">${off}${offline}${bands}${blocks}${NOW >= from ? `<div class="cq-ph-now" style="top:${y(NOW)}px"><span class="num">${clock(NOW)}</span></div>` : ''}</div>
    </div>
    ${q.band ? `<div class="cq-ph-bl">${bandList()}</div>` : ''}
  </div>`;
}

export function queue(api) {
  setup(api);
  if (S.isPhone) {
    return `<div class="cq-c cq-ph">
      <div class="ph-pad">${summary()}</div>
      <div class="psec cq-ph-offs"><div class="psec-h"><h2>Not on the schedule</h2></div><div class="ph-pad cq-offs-g">${failCard()}${warnCards()}</div></div>
      <div id="cq-sched">${phoneAgenda()}</div>
    </div>`;
  }
  return `<div class="cq-c">
    ${summary()}
    <section class="cq-sched-s" aria-labelledby="cq-s-h">
      <div class="sec-h"><h2 id="cq-s-h">Schedule</h2><span class="t3">Placed on the Node with the earliest estimated finish</span><span class="grow"></span>
        <div class="seg-c" role="radiogroup" aria-label="Zoom"><button role="radio" class="seg-o" data-zoom="near" aria-checked="${q.zoom === 'near'}">Next 90 min wide</button><button role="radio" class="seg-o" data-zoom="even" aria-checked="${q.zoom === 'even'}">Even hours</button></div></div>
      <div id="cq-sched">${schedHtml()}</div>
      ${legend()}
    </section>
    ${offSchedule()}
  </div>`;
}

// ───────────────────────────── backfill dialog

const ask = () => D.backfillAsk;
const presets = () => [
  ['mobile', 'Played on phones', ask().playedOnMobile],
  ['requested', 'Requested', Object.keys(ask().requestedBy)],
  ['recent', 'Added recently', ask().addedRecently],
  ['all', 'All', D.cinemaWorks],
  ['none', 'None', []],
].map(([id, label, ids]) => ({ id, label, ids: ids.filter((x) => D.cinemaWorks.includes(x)) }));

function initState() {
  const a = ask();
  st = { sel: new Set(presets()[0].ids), dest: a.existingMatch ? { kind: 'profile', id: a.existingMatch } : { kind: 'create', name: a.destination.name }, makeDefault: false, editing: false, open: true };
}

const profile = () => D.profiles.find((p) => p.id === ask().profile);
const destName = () => (st.dest.kind === 'create' ? st.dest.name : D.profiles.find((p) => p.id === st.dest.id).name);
const activePreset = () => presets().find((p) => p.ids.length === st.sel.size && p.ids.every((x) => st.sel.has(x)));
const finishAt = (h) => { const d = new Date(+D.now + h * 3600000); return `${d.toLocaleDateString('en-US', { weekday: 'short' })} ${d.toTimeString().slice(0, 5)}`; };

function sentence() {
  const n = st.sel.size, total = D.cinemaWorks.length, rest = total - n, e = D.estimate(n), p = profile();
  const tgt = S.target(ask().addedTarget).name;
  const used = Object.entries(e.byNode).filter(([, v]) => v > 0).map(([k]) => k).join(' and ');
  const destTok = st.editing
    ? `<span class="cq-tok cq-tok-dest editing"><input class="cq-rename" value="${S.esc(st.dest.name)}" aria-label="Name of the new Profile" size="${Math.max(12, st.dest.name.length)}"></span>`
    : `<span class="cq-tok cq-tok-dest">${st.dest.kind === 'create' ? `<button class="cq-tok-b" data-rename aria-label="Rename the new Profile, ${S.esc(destName())}">${S.esc(destName())}${S.I('pencil', 'cq-tic')}</button>` : `<button class="cq-tok-b" data-destmenu aria-haspopup="menu">${S.esc(destName())}${S.I('chevron-down', 'cq-tic')}</button>`}</span>`;
  const head = `Make <b>${S.esc(tgt)}</b> for <button class="cq-tok cq-tok-b" data-countmenu aria-haspopup="menu" aria-label="${n} Works selected, change">${n}${S.I('chevron-down', 'cq-tic')}</button> of ${total} ${S.esc(p.name)} Works.`;
  const cost = n ? ` That's <span class="cq-ro num">≈ ${S.fmtHours(e.hours - D.today.hoursQueued)}</span> on ${used} and <span class="cq-ro num">${e.gb} GB</span> of ${ask().disk.freeTB} TB free.` : ' No Conversions are made.';
  const move = rest === 0 ? ` Every Work stays in ${S.esc(p.name)}.` : ` ${n === 0 ? `All ${total}` : `The other ${rest}`} move to ${destTok}.`;
  return `<p class="cq-sent" aria-live="polite">${head}${cost}${move}</p>`;
}

function preview() {
  const n = st.sel.size, e = D.estimate(n), qh = D.today.hoursQueued;
  const forgeH = (e.byNode.forge * 9) / 3 / 60, atlasW = (e.byNode.atlas * 34) / 60;
  const R = Math.max(e.hours, qh + forgeH) + 0.8, x = (h) => `${((h / R) * 100).toFixed(2)}%`;
  const atlasN = S.node('atlas'), open = (openAt(atlasN) - NOW) / 60;
  const wins = []; for (let a = open; a < R; a += 24) wins.push([a, a + 6]);
  const off = []; let c = 0; wins.forEach(([a, b]) => { off.push([c, a]); c = b; }); off.push([c, R]);
  const atlasSegs = []; let left = atlasW;
  if (e.byNode.atlas) { const end = e.hours; [...wins].reverse().forEach(([a, b]) => { if (left <= 0 || a > end) return; const bEnd = Math.min(b, end), take = Math.min(left, bEnd - a); atlasSegs.push([bEnd - take, bEnd]); left -= take; }); }
  const ticks = []; const step = R > 16 ? 6 : 2;
  for (let h = Math.ceil((D.now.getHours() + D.now.getMinutes() / 60) / step) * step - (D.now.getHours() + D.now.getMinutes() / 60); h < R; h += step) {
    const d = new Date(+D.now + h * 3600000); ticks.push(`<span style="left:${x(h)}" class="num">${d.toTimeString().slice(0, 5)}</span>`);
  }
  const seg = (a, b, cls, label = '') => `<span class="cq-mseg ${cls}" style="left:${x(a)};width:${x(b - a)}">${label}</span>`;
  return `<div class="cq-mini">
    <div class="cq-mini-h"><b>Where it lands</b><span class="t2">${n ? `After the <span class="num">${qh} h</span> already queued. Done by about <span class="num draft-t">${finishAt(e.hours)}</span>.` : 'Nothing is added to the schedule.'}</span></div>
    <div class="cq-mini-g">
      <span class="cq-mini-l">forge</span><div class="cq-mini-t">${seg(0, qh, 'q', 'already queued')}${e.byNode.forge ? seg(qh, qh + forgeH, 'bf', `${e.byNode.forge}`) : ''}</div>
      <span class="cq-mini-l">atlas</span><div class="cq-mini-t">${off.map(([a, b]) => seg(a, b, 'off')).join('')}${atlasSegs.map(([a, b]) => seg(a, b, 'bf', '')).join('')}${e.byNode.atlas ? '' : `<span class="cq-mini-none t3">not needed for ${n} Works; atlas joins past 20</span>`}</div>
      <span></span><div class="cq-mini-ax">${ticks.join('')}</div>
    </div>
  </div>`;
}

function wall() {
  const a = ask(), act = activePreset();
  const segs = presets().map((p) => `<button role="radio" class="seg-o" data-preset="${p.id}" aria-checked="${act && act.id === p.id}">${S.esc(p.label)}<span class="seg-n num">${p.ids.length}</span></button>`).join('');
  const noCap = new Set(a.noCapable.map((x) => x.work));
  const posters = D.cinemaWorks.map((id) => {
    const w = S.W(id), on = st.sel.has(id), req = a.requestedBy[id];
    const badges = [
      a.playedOnMobile.includes(id) ? `<span class="cq-pbadge" title="Played on a phone or tablet in the last 30 days">${S.I('smartphone')}</span>` : '',
      req ? `<span class="cq-pbadge" title="Requested by ${S.esc(D.members[req])}">${S.esc(D.members[req].split(' ')[0])}</span>` : '',
      a.addedRecently.includes(id) ? `<span class="cq-pbadge" title="Added recently">New</span>` : '',
      noCap.has(id) ? `<span class="cq-pbadge warn" title="Dolby Vision Profile 5 source, no online Node can convert it">DV P5</span>` : '',
    ].join('');
    return `<button class="cq-post${on ? ' on' : ''}" data-pick="${id}" aria-pressed="${on}" ${on ? `data-work="${id}"` : ''} style="--ratio:${S.ct(w).artwork.ratio}">
      ${S.art(w, 'cq-pimg')}<span class="cq-pbadges">${badges}</span><span class="cq-pcheck">${S.I(on ? 'check' : 'plus')}</span><span class="cq-pname">${S.esc(w.title)}</span>
    </button>`;
  }).join('');
  return `<div class="cq-wall-s">
    <div class="cq-wall-h"><b>Works</b><span class="t2 num">${st.sel.size} of ${D.cinemaWorks.length} selected</span><span class="grow"></span><div class="seg-c" role="radiogroup" aria-label="Select">${segs}</div></div>
    <div class="cq-wall">${posters}</div>
  </div>`;
}

function destSection() {
  const a = ask(), p = profile(), rest = D.cinemaWorks.length - st.sel.size;
  const ctype = D.contentTypes[a.contentType];
  const match = a.existingMatch && D.profiles.find((x) => x.id === a.existingMatch);
  const opt = (on, attrs, title, body) => `<div class="cq-dopt${on ? ' on' : ''}" ${attrs}><span class="cq-radio" aria-hidden="true"></span><div><b>${title}</b><p class="t2">${body}</p></div></div>`;
  const tnames = (ids) => ids.map((t) => S.target(t).name).join(' and ');
  const other = st.dest.kind === 'profile' && st.dest.id !== a.existingMatch ? D.profiles.find((x) => x.id === st.dest.id) : null;
  const sel = new Set(st.sel), dv = a.noCapable.filter((x) => sel.has(x.work));
  const dvNote = dv.length ? `<div class="cq-dv"><span class="chip-warn">${S.I('triangle-alert')}DV P5</span><p>${dv.map((x) => `<b>${S.esc(S.W(x.work).title)}</b>`).join(', ')} ${dv.length === 1 ? 'has a' : 'have'} Dolby Vision Profile 5 source${dv.length === 1 ? '' : 's'} no online Node can convert. If selected, ${dv.length === 1 ? 'it waits' : 'they wait'} as no capable Node until studio is back or forge passes its trial.</p></div>` : '';
  return `<div class="cq-dest">
    <div class="cq-dest-h"><b>Where the other ${rest} go</b><span class="t2">They keep what they have: ${tnames(p.targets)}.</span></div>
    ${rest === 0 ? '<p class="t2">Every Work is selected, so nothing moves.</p>' : `<div class="cq-dopts" role="radiogroup" aria-label="Destination">
      ${match ? `<button role="radio" aria-checked="${st.dest.kind === 'profile' && st.dest.id === match.id}" class="cq-dbtn" data-dest="${match.id}">${opt(st.dest.kind === 'profile' && st.dest.id === match.id, '', `Reuse ${S.esc(match.name)}`, `Already matches: ${tnames(match.targets)}. ${match.works} Works use it.`)}</button>` : ''}
      <button role="radio" aria-checked="${st.dest.kind === 'create'}" class="cq-dbtn" data-dest="create">${opt(st.dest.kind === 'create', '', `Create ${S.esc(st.dest.kind === 'create' ? st.dest.name : a.destination.name)}`, `${S.esc(p.name)} as it was: ${tnames(p.targets)}. Rename it in the sentence above.`)}</button>
      ${other ? `<button role="radio" aria-checked="true" class="cq-dbtn" data-dest="${other.id}">${opt(true, '', S.esc(other.name), `Has ${tnames(other.targets)} only. These Works would lose what ${S.esc(p.name)} gave them.`)}</button>` : ''}
    </div>
    <button class="btn sm ghost" data-destmenu aria-haspopup="menu">${S.I('list')}Pick another Profile</button>`}
    <div class="cq-defrow" role="switch" tabindex="0" aria-checked="${st.makeDefault}" data-default><span class="cq-def-t"><b>Make ${S.esc(rest ? destName() : 'the new Profile')} the default for ${S.esc(ctype.plural)}</b><span class="t2">${S.esc(p.name)} is the default now, so new ${S.esc(ctype.plural)} get ${S.esc(S.target(a.addedTarget).name)}${st.makeDefault ? '. With this on, they won\'t.' : '.'}</span></span><span class="sw"><span></span></span></div>
    ${dvNote}
  </div>`;
}

function footer() {
  const n = st.sel.size, rest = D.cinemaWorks.length - n, e = D.estimate(n);
  const parts = [];
  parts.push(n ? `<b class="num">${n}</b> ${n === 1 ? 'Work gets' : 'Works get'} ${S.esc(S.target(ask().addedTarget).name)} as Backfill, done ≈ <span class="num">${finishAt(e.hours)}</span>` : 'No Conversions');
  if (rest) parts.push(`<b class="num">${rest}</b> move to ${S.esc(destName())}${st.dest.kind === 'create' ? ' (new)' : ''}`);
  return `<div class="cq-foot"><p class="cq-restate t2">${parts.join('<span class="s-sep">·</span>')}</p><div class="cq-foot-b"><button class="btn ghost" data-dlg-cancel>Cancel</button><button class="btn pri" data-dlg-confirm>${n ? `Queue ${n} and save` : 'Save'}${rest ? ` · move ${rest}` : ''}</button></div></div>`;
}

function dlgBody() {
  return `<div class="cq-dlg-sc">
      ${sentence()}
      ${preview()}
      ${wall()}
      ${destSection()}
    </div>
    ${footer()}`;
}

export function dialog(api) {
  setup(api);
  initState();
  const p = profile();
  return `<div class="cq-dlg-wrap${S.isPhone ? ' ph' : ''}">
    <div class="cq-scrim" data-dlg-cancel></div>
    <div class="cq-dlg" role="dialog" aria-modal="true" aria-labelledby="cq-dlg-h">
      ${S.isPhone ? '<span class="grab" aria-hidden="true"></span>' : ''}
      <div class="cq-dlg-top"><span class="chip chip-live">New Target</span><h2 id="cq-dlg-h">${S.esc(p.name)} gains ${S.esc(S.target(ask().addedTarget).name)}</h2><span class="grow"></span><button class="ib" data-dlg-cancel aria-label="Cancel">${S.I('x')}</button></div>
      <div class="cq-dlg-body">${dlgBody()}</div>
    </div>
  </div>`;
}

// ───────────────────────────── wiring

function popup(root, anchor, html) {
  let m = root.querySelector('.cq-menu');
  if (!m) { m = document.createElement('div'); m.className = 'menu cq-menu'; m.setAttribute('role', 'menu'); root.append(m); }
  m.innerHTML = html; m.hidden = false;
  window.lucide && window.lucide.createIcons();
  const r = anchor.getBoundingClientRect(), rr = root.getBoundingClientRect();
  m.style.top = `${r.bottom - rr.top + root.scrollTop + 6}px`;
  m.style.left = `${Math.max(8, Math.min(r.left - rr.left, rr.width - m.offsetWidth - 8))}px`;
  m.querySelector('button')?.focus({ preventScroll: true });
  return m;
}
const closePopup = (root) => { const m = root.querySelector('.cq-menu'); if (m) m.hidden = true; };

function wireBackfill(root) {
  const wrap = root.querySelector('.cq-dlg-wrap');
  const body = wrap.querySelector('.cq-dlg-body');
  const render = (focusSel) => {
    const sc = body.querySelector('.cq-dlg-sc'), top = sc ? sc.scrollTop : 0;
    body.innerHTML = dlgBody();
    window.lucide && window.lucide.createIcons();
    body.querySelector('.cq-dlg-sc').scrollTop = top;
    S.wireAmbient('.cq-post.on', [...st.sel][0] || 'dune-2');
    if (focusSel) body.querySelector(focusSel)?.focus({ preventScroll: true });
    if (st.editing) { const i = body.querySelector('.cq-rename'); i.focus(); i.select(); }
  };
  const close = () => { wrap.hidden = true; closePopup(root); };
  const commitRename = () => { const i = body.querySelector('.cq-rename'); if (!i) return; st.dest.name = i.value.trim() || ask().destination.name; st.editing = false; render('[data-rename]'); };

  root.addEventListener('click', (e) => {
    const qq = (s) => e.target.closest(s);
    let el;
    if ((el = qq('[data-open-backfill]'))) { initState(); render(); wrap.hidden = false; return; }
    if ((el = qq('.cq-mi'))) {
      closePopup(root);
      const { kind, val } = el.dataset;
      if (kind === 'preset') st.sel = new Set(presets().find((p) => p.id === val).ids);
      if (kind === 'count') st.sel = new Set(D.cinemaWorks.slice(0, Number(val)));
      if (kind === 'dest') st.dest = val === 'create' ? { kind: 'create', name: st.dest.kind === 'create' ? st.dest.name : ask().destination.name } : { kind: 'profile', id: val };
      return render(kind === 'dest' ? '[data-destmenu]' : '[data-countmenu]');
    }
    if (wrap.hidden) return;
    if (!qq('.cq-menu') && !qq('[data-countmenu],[data-destmenu]')) closePopup(root);
    if ((el = qq('[data-dlg-cancel]'))) { close(); return S.toast('Back to the editor. Nothing was saved.'); }
    if ((el = qq('[data-dlg-confirm]'))) {
      const n = st.sel.size, rest = D.cinemaWorks.length - n;
      close();
      return S.toast(`Saved ${profile().name}. ${n} Conversions queued as Backfill${rest ? `; ${rest} Works moved to ${destName()}` : ''}${st.makeDefault ? `, now the default` : ''}.`);
    }
    if ((el = qq('[data-pick]'))) { const id = el.dataset.pick; st.sel.has(id) ? st.sel.delete(id) : st.sel.add(id); return render(`[data-pick="${id}"]`); }
    if ((el = qq('[data-preset]'))) { st.sel = new Set(presets().find((p) => p.id === el.dataset.preset).ids); return render(`[data-preset="${el.dataset.preset}"]`); }
    if ((el = qq('[data-dest]'))) { const v = el.dataset.dest; st.dest = v === 'create' ? { kind: 'create', name: st.dest.kind === 'create' ? st.dest.name : ask().destination.name } : { kind: 'profile', id: v }; return render(`[data-dest="${v}"]`); }
    if ((el = qq('[data-default]'))) { st.makeDefault = !st.makeDefault; return render('[data-default]'); }
    if ((el = qq('[data-rename]'))) { st.editing = true; return render(); }
    if ((el = qq('[data-countmenu]'))) {
      const items = presets().map((p) => `<button class="cq-mi" role="menuitem" data-kind="preset" data-val="${p.id}">${S.I(p.id === 'none' ? 'circle-off' : p.id === 'all' ? 'check-check' : 'filter')}<span>${S.esc(p.label)}</span><span class="mi-r num t3">${p.ids.length}</span></button>`).join('');
      return popup(root, el, `<p class="cq-mh t3">Select Works</p>${items}`);
    }
    if ((el = qq('[data-destmenu]'))) {
      const a = ask(), p = profile();
      const others = D.profiles.filter((x) => x.contentType === p.contentType && x.id !== p.id);
      const items = [
        `<button class="cq-mi" role="menuitem" data-kind="dest" data-val="create">${S.I('plus')}<span>Create ${S.esc(a.destination.name)}</span></button>`,
        ...others.map((x) => `<button class="cq-mi" role="menuitem" data-kind="dest" data-val="${x.id}">${S.I(x.id === a.existingMatch ? 'equal' : 'layers')}<span>${S.esc(x.name)}</span><span class="mi-r t3">${x.id === a.existingMatch ? 'matches' : x.targets.map((t) => S.target(t).name).join(', ')}</span></button>`),
      ].join('');
      return popup(root, el, `<p class="cq-mh t3">Move the other Works to</p>${items}`);
    }
  });
  body.addEventListener('keydown', (e) => {
    if (e.target.matches('.cq-rename')) {
      if (e.key === 'Enter') { e.preventDefault(); commitRename(); }
      if (e.key === 'Escape') { e.stopPropagation(); st.editing = false; render('[data-rename]'); }
    }
    if (e.target.matches('[data-default]') && (e.key === ' ' || e.key === 'Enter')) { e.preventDefault(); st.makeDefault = !st.makeDefault; render('[data-default]'); }
  });
  body.addEventListener('focusout', (e) => { if (e.target.matches('.cq-rename')) setTimeout(() => st.editing && commitRename(), 0); });
  body.addEventListener('input', (e) => { if (e.target.matches('.cq-rename')) e.target.size = Math.max(12, e.target.value.length); });
  document.addEventListener('keydown', (e) => { if (e.key === 'Escape' && !wrap.hidden && !root.querySelector('.cq-menu:not([hidden])') && !st.editing) { close(); S.toast('Back to the editor. Nothing was saved.'); } });
  S.wireAmbient('.cq-post.on', [...st.sel][0] || 'dune-2');
}

function blockFor(id) {
  for (const r of schedule()) for (const l of r.lanes) for (const bk of l) if (bk.c.id === id) return bk;
  return null;
}

function wireQueue(root) {
  const sched = root.querySelector('#cq-sched');
  let pop = null, hideT;
  const ensurePop = () => {
    if (!pop) { pop = document.createElement('div'); pop.className = 'pop cq-pop'; pop.setAttribute('role', 'dialog'); pop.hidden = true; root.append(pop);
      pop.addEventListener('mouseenter', () => clearTimeout(hideT));
      pop.addEventListener('mouseleave', () => { if (!q.pinned) hidePop(); });
    }
    return pop;
  };
  const showPop = (btn) => {
    clearTimeout(hideT);
    const id = btn.dataset.conv, c = conv(id), bk = blockFor(id), p = ensurePop();
    if (p.dataset.conv === id && !p.hidden) return;
    p.dataset.conv = id;
    p.setAttribute('aria-label', `${S.W(c.work).title} details`);
    p.innerHTML = `${detail(c, bk)}<div class="cq-pd-act">${actions(c).map(([l, ic, k], i) => `<button class="btn sm${k === 'danger' ? ' ghost cq-danger' : ''}" data-pact="${i}">${S.I(ic)}${l}</button>`).join('')}</div>`;
    p.hidden = false;
    window.lucide && window.lucide.createIcons();
    const r = btn.getBoundingClientRect(), rr = root.getBoundingClientRect(), row = btn.closest('.cq-row').getBoundingClientRect();
    let top = row.bottom - rr.top + 6;
    if (r.bottom + p.offsetHeight + 16 > window.innerHeight && r.top - p.offsetHeight - 8 > 0) top = r.top - rr.top - p.offsetHeight - 8;
    p.style.top = `${top}px`;
    p.style.left = `${Math.max(8, Math.min(r.left - rr.left, rr.width - p.offsetWidth - 16))}px`;
  };
  const hidePop = (now) => { clearTimeout(hideT); hideT = setTimeout(() => { if (pop) { pop.hidden = true; pop.dataset.conv = ''; } q.pinned = null; }, now ? 0 : 220); };

  const rerender = () => {
    sched.innerHTML = S.isPhone ? phoneAgenda() : schedHtml();
    window.lucide && window.lucide.createIcons();
    S.wireAmbient('[data-work]', 'dune-2');
  };

  if (!S.isPhone) {
    root.addEventListener('mouseover', (e) => { const b = e.target.closest('.cq-blk[data-conv]'); if (b && !q.pinned) showPop(b); });
    root.addEventListener('mouseout', (e) => { const b = e.target.closest('.cq-blk[data-conv]'); if (b && !q.pinned && !(e.relatedTarget && (b.contains(e.relatedTarget) || e.relatedTarget.closest('.cq-pop')))) hidePop(); });
    root.addEventListener('focusin', (e) => { const b = e.target.closest('.cq-blk[data-conv]'); if (b) { q.pinned = null; showPop(b); } });
    root.addEventListener('focusout', (e) => { const b = e.target.closest('.cq-blk[data-conv]'); if (b && !(e.relatedTarget && e.relatedTarget.closest('.cq-pop'))) hidePop(); });
  }
  document.addEventListener('keydown', (e) => { if (e.key === 'Escape' && pop && !pop.hidden) { const id = pop.dataset.conv; hidePop(true); root.querySelector(`.cq-blk[data-conv="${id}"]`)?.focus({ preventScroll: true }); } });

  root.addEventListener('click', (e) => {
    const qq = (s) => e.target.closest(s);
    let el;
    if ((el = qq('[data-pact]'))) {
      const c = conv(pop.dataset.conv), [l, , , msg] = actions(c)[Number(el.dataset.pact)];
      if (msg) { hidePop(true); return S.toast(msg); }
      const t = S.target(c.target);
      return S.openMenu(el, { title: `Pin ${t.name} to a Node`, items: D.nodes.map((n) => [n.name, n.status === 'online' ? 'pin' : 'power-off', '', `${t.name} is pinned to ${n.name}. Its Conversions always run there${n.status === 'online' ? '' : ' once it is back'}.`]) });
    }
    if ((el = qq('.cq-blk[data-conv]'))) { if (q.pinned === el.dataset.conv) { hidePop(true); } else { showPop(el); q.pinned = el.dataset.conv; } return; }
    if ((el = qq('.cq-pb[data-conv]'))) {
      const c = conv(el.dataset.conv), bk = blockFor(c.id), s = root.querySelector('.sheet-wrap');
      s.querySelector('.sheet').innerHTML = `<span class="grab" aria-hidden="true"></span><div class="cq-psheet">${detail(c, bk)}</div><div class="sheet-grp" role="menu">${actions(c).map(([l, ic, k, msg]) => `<button class="mi${k === 'danger' ? ' danger' : ''}" role="menuitem" data-msg="${S.esc(msg || `Pinning ${S.target(c.target).name} to a Node is not in this prototype`)}">${S.I(ic)}<span>${l}</span></button>`).join('')}</div><button class="sheet-x" data-close>Close</button>`;
      s.hidden = false;
      window.lucide && window.lucide.createIcons();
      S.setAmbient(c.work);
      return;
    }
    if (pop && !pop.hidden && !qq('.cq-pop') && !qq('.menu')) hidePop(true);
    if ((el = qq('[data-zoom]'))) { q.zoom = el.dataset.zoom; root.querySelectorAll('[data-zoom]').forEach((b) => b.setAttribute('aria-checked', b === el)); return rerender(); }
    if ((el = qq('[data-band]'))) { q.band = q.band === el.dataset.band ? null : el.dataset.band; rerender(); if (q.band) root.querySelector('.cq-bandlist')?.scrollIntoView({ block: 'nearest', behavior: 'smooth' }); return; }
    if ((el = qq('[data-pnode]'))) { q.node = el.dataset.pnode; q.band = null; return rerender(); }
  });
}

export function wire(api, screen) {
  setup(api);
  const root = document.getElementById('app');
  if (screen === 'backfill') return wireBackfill(root);
  wireQueue(root);
  S.wireAmbient('[data-work]', 'dune-2');
}
