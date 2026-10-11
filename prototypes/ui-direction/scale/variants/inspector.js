// PROTOTYPE · B · Inspector. A dense overview on the left, one selection's full detail on the right (a bottom sheet on phone).
import { esc, I, md } from '../../variants/lit-console/variant.js';
import { STATES, MISSING_WHY, items, worst, counts, summaryLine, targetsOf, editionsOf, needsYou } from '../model.js';

let V, M, S = { key: null, filter: 'all', phone: false }, root;

const WHY_SHORT = { searching: 'Searching', 'not-found': 'Not found', waiting: 'Waiting', grace: 'Grace', queued: 'Queued' };
const SOURCING_ICON = { downloaded: 'download', 'download-then-derive': 'hourglass', derived: 'replace' };
const nameOf = (id) => M.tgs.find((t) => t.id === id)?.name ?? id;
const srcLine = (t) => (t.sourcing === 'downloaded' ? 'Downloaded' : t.sourcing === 'derived' ? `Derived from ${nameOf(t.from)}` : `Downloaded, or derived after ${t.grace}`);
const jobOf = (c) => V.data.jobs.find((j) => j.id === c.job);

function build(vctx) {
  V = vctx;
  const its = items(vctx.work, vctx.desc);
  const groups = [];
  its.forEach((it, i) => {
    const node = it.path[it.path.length - 1];
    let g = groups.find((x) => x.node === node);
    if (!g) groups.push((g = { node, idx: [] }));
    g.idx.push(i);
  });
  M = { its, groups, eds: editionsOf(vctx.work), tgs: targetsOf(vctx.work), many: vctx.desc.layout.items === 'many', level: vctx.desc.levels[0] };
  if (!S.key) S.key = readHash() ?? urgentKey(M.its.map((_, i) => i));
}

const at = (i, ed, tg) => {
  const edition = M.eds.find((e) => e.id === ed), target = M.tgs.find((t) => t.id === tg);
  return { ...M.its[i], i, edition, target, cell: M.its[i].item.cells[ed][tg] };
};
const cellsOf = (idx) => idx.flatMap((i) => M.eds.flatMap((e) => M.tgs.map((t) => at(i, e.id, t.id))));
const keyOf = (x) => `c:${x.i}:${x.edition.id}:${x.target.id}`;
function urgentKey(idx) {
  const list = cellsOf(idx);
  const best = list.reduce((a, x) => (STATES[x.cell.state].rank < STATES[a.cell.state].rank ? x : a), list[0]);
  return keyOf(best);
}
function readHash() {
  const m = /^#ix=(.+)$/.exec(location.hash);
  if (!m) return null;
  const k = decodeURIComponent(m[1]);
  const [kind, a, ed, tg] = k.split(':');
  if (kind === 'c' && M.its[a]?.item.cells[ed]?.[tg]) return k;
  if (kind === 'l' && M.groups[a]) return k;
  return null;
}
const parse = (k) => { const [kind, a, ed, tg] = k.split(':'); return kind === 'c' ? { kind, x: at(+a, ed, tg) } : { kind, g: +a }; };
const groupLabel = (g) => M.level.label(M.groups[g].node.n);
const groupOfItem = (i) => M.groups.findIndex((g) => g.idx.includes(i));


const chip = (s, iconOnly = false) => `<span class="st st-${s}"${iconOnly ? ` title="${STATES[s].label}"` : ''}>${I(STATES[s].icon)}${iconOnly ? '' : (STATES[s].short ?? STATES[s].label)}</span>`;

function tileNum(x, compact) {
  const c = x.cell;
  switch (c.state) {
    case 'met': return `<span class="ix-n num">${c.score}</span>`;
    case 'upgrade': return compact || S.phone ? `<span class="ix-n num"><span class="t3">→ </span>${c.found.score}</span>` : `<span class="ix-n num">${c.score}<span class="t3"> → </span>${c.found.score}</span>`;
    case 'downloading': case 'converting': return `<span class="ix-n num draft-t" data-live-pct="${c.job}"></span>`;
    case 'failed': return `<span class="ix-w">${compact ? esc(c.stage) : `at ${esc(c.stage)}`}</span>`;
    case 'missing': return `<span class="ix-w">${esc(compact ? WHY_SHORT[c.why] : MISSING_WHY[c.why])}</span>`;
    case 'unaired': return `<span class="ix-n num t2">${md(c.airs)}</span>`;
    default: return '';
  }
}

function tile(x, mode, r, cIdx) {
  const k = keyOf(x), on = S.key === k, s = x.cell.state;
  const live = STATES[s].live ? `<span class="ix-tbar" data-live-progress="${x.cell.job}"></span>` : '';
  return `<button class="ix-tile ix-${s}${on ? ' on' : ''}" data-ix="${k}" data-r="${r}" data-c="${cIdx}" aria-pressed="${on}" aria-label="${esc(`${x.edition.name} at ${x.target.name}: ${STATES[s].label}`)}">${chip(s, mode === 'sm')}${tileNum(x, mode === 'sm')}${live}</button>`;
}

function matrix(i, mode) {
  const head = M.tgs.map((t) => (mode === 'sm'
    ? `<div class="ix-th" title="${esc(t.label)}">${I(SOURCING_ICON[t.sourcing])}<b>${esc(t.short)}</b></div>`
    : `<div class="ix-th" title="${esc(t.label)}"><b>${esc(t.name)}</b><span class="t3">${I(SOURCING_ICON[t.sourcing])}${esc(srcLine(t))}</span></div>`)).join('');
  const rows = M.eds.map((e, r) => `<div class="ix-ed"><b class="ell">${esc(e.name)}</b>${mode === 'sm' ? '' : `<span class="t3">${esc(e.note)}</span>`}</div>${M.tgs.map((t, c) => tile(at(i, e.id, t.id), mode, r, c)).join('')}`).join('');
  return `<div class="ix-mx ix-mx-${mode}" data-grid style="--n:${M.tgs.length}"><div class="ix-corner t3">${mode === 'sm' ? '' : `Edition ${I('move-right')} Target`}</div>${head}${rows}</div>`;
}


const FILTERS = () => [
  ['all', 'All', () => true],
  ['attn', 'Needs you', (x) => needsYou(x.cell)],
  ['live', 'In flight', (x) => STATES[x.cell.state].live],
  ...M.tgs.map((t) => [`nm:${t.id}`, `${t.short} not met`, (x) => x.target.id === t.id && !['met', 'unaired'].includes(x.cell.state)]),
];
const filterFn = () => FILTERS().find((f) => f[0] === S.filter)[2];

function filterBar() {
  const all = cellsOf(M.its.map((_, i) => i));
  return `<div class="seg-c ix-filter" role="radiogroup" aria-label="Show">${FILTERS().map(([id, l, fn]) => `<button class="seg-o" role="radio" data-ixf="${id}" aria-checked="${S.filter === id}">${esc(l)}${id === 'all' ? '' : `<span class="seg-n num">${all.filter(fn).length}</span>`}</button>`).join('')}</div>`;
}

function epTile(i, r, cIdx, sel, fn) {
  const it = M.its[i];
  const sq = M.eds.map((e) => M.tgs.map((t) => {
    const x = at(i, e.id, t.id), s = x.cell.state;
    return `<span class="sq sq-${s}${fn(x) ? '' : ' dim'}${sel === keyOf(x) ? ' cur' : ''}"${STATES[s].live ? ` data-live-progress="${x.cell.job}"` : ''}></span>`;
  }).join('')).join('');
  const list = cellsOf([i]);
  const hit = list.some(fn);
  const on = sel.startsWith(`c:${i}:`);
  return `<button class="ix-ep${on ? ' on' : ''}${hit ? '' : ' dim'}" data-ix="e:${i}" data-r="${r}" data-c="${cIdx}" aria-pressed="${on}" aria-label="${esc(`${it.code}, ${it.label}: ${summaryLine(counts(list)).join(', ')}`)}"><span class="ix-micro" style="--n:${M.tgs.length}">${sq}</span><span class="ix-epn num">${it.item.n}</span></button>`;
}

function board() {
  const fn = filterFn();
  return `<div class="ix-board" data-grid>${M.groups.map((g, r) => {
    const list = cellsOf(g.idx), c = counts(list), w = worst(list);
    const wanted = list.length - c.unaired;
    const on = S.key === `l:${r}`;
    return `<div class="ix-season"><button class="ix-sh${on ? ' on' : ''}" data-ix="l:${r}" aria-pressed="${on}"><b>${esc(groupLabel(r))}</b><span class="t3"><span class="num">${c.met}/${wanted}</span> met</span>${chip(w, true)}</button>
      <div class="ix-eps">${g.idx.map((i, cIdx) => epTile(i, r, cIdx + 1, S.key, fn)).join('')}</div></div>`;
  }).join('')}</div>`;
}

function axisKey() {
  const edl = M.eds.map((e) => esc(e.name)).join(' · ');
  const tgl = M.tgs.map((t) => `<span title="${esc(t.label)}">${I(SOURCING_ICON[t.sourcing])}${esc(t.short)}</span>`).join('');
  const legend = Object.entries(STATES).map(([s, d]) => `<span><span class="sq sq-${s}"></span>${d.short ?? d.label}</span>`).join('');
  return `<div class="ix-key"><div class="ix-axis"><span class="ix-micro ix-micro-key" style="--n:${M.tgs.length}">${'<span class="sq"></span>'.repeat(M.eds.length * M.tgs.length)}</span><span class="t3"><b>Rows</b> ${edl}</span><span class="t3 ix-cols"><b>Columns</b> ${tgl}</span></div><div class="sx-legend">${legend}</div></div>`;
}

function overview() {
  if (!M.many) return S.phone ? phoneMatrix() : `<div class="ix-panel">${matrix(0, 'lg')}</div>`;
  return `<div class="ix-ovh">${filterBar()}</div><div class="ix-panel">${board()}</div>${axisKey()}`;
}

function phoneMatrix() {
  const rows = M.eds.map((e, r) => `<div class="ix-ped"><b>${esc(e.name)}</b><span class="t3">${esc(e.note)}</span></div><div class="ix-prow" style="--n:${M.tgs.length}">${M.tgs.map((t, c) => tile(at(0, e.id, t.id), 'ph', r, c)).join('')}</div>`).join('');
  const head = `<div class="ix-prow ix-phead" style="--n:${M.tgs.length}">${M.tgs.map((t) => `<span>${I(SOURCING_ICON[t.sourcing])}${esc(t.short)}</span>`).join('')}</div>`;
  const src = M.tgs.map((t) => `<li>${I(SOURCING_ICON[t.sourcing])}<b>${esc(t.name)}</b><span class="t3">${esc(t.label)}</span></li>`).join('');
  return `<div class="ix-panel ix-pmx" data-grid>${head}${rows}</div><ul class="ix-src-list">${src}</ul>`;
}


function meter(c) {
  const top = Math.max(c.score, c.cutoff, c.found?.score ?? 0) * 1.08;
  const pct = (v) => `${((v / top) * 100).toFixed(1)}%`;
  return `<div class="ix-meter" role="img" aria-label="Score ${c.score}${c.found ? `, found ${c.found.score}` : ''}, cutoff ${c.cutoff}">
    ${c.found ? `<span class="ix-m-found" style="width:${pct(c.found.score)}"></span>` : ''}<span class="ix-m-cur" style="width:${pct(c.score)}"></span><span class="ix-m-cut" style="left:${pct(c.cutoff)}"><span class="num">cutoff ${c.cutoff}</span></span></div>`;
}

const kv = (rows) => `<dl class="kv ix-kv">${rows.filter(Boolean).map(([k, v]) => `<dt>${k}</dt><dd>${v}</dd>`).join('')}</dl>`;
const file = (label, name, extra = '') => `<div class="ix-file"><p class="t3">${label}</p><p class="mono">${esc(name)}</p>${extra}</div>`;
const btn = (label, icon, cls = '', msg = `${label} is not in this prototype`) => `<button class="btn sm ${cls}" data-toast="${esc(msg)}">${I(icon)}${esc(label)}</button>`;

const H = (icon, text, when, live) => ({ icon, text, when, live });
function events(x) {
  const c = x.cell, t = x.target;
  switch (c.state) {
    case 'met': return c.source === 'derived'
      ? [H('replace', `Derived from the ${nameOf(t.from ?? M.tgs[0].id)} file on forge`, 'Oct 2 · 04:18'), H('folder-check', `Imported · score ${c.score} reached cutoff ${c.cutoff}`, 'Oct 2 · 04:31')]
      : [H('magnet', 'Grabbed from Aurora', 'Sep 29 · 22:40'), H('download', 'Downloaded on seedbox-ams', 'Sep 29 · 23:12'), H('folder-check', `Imported · score ${c.score} reached cutoff ${c.cutoff}`, 'Sep 29 · 23:20')];
    case 'upgrade': return [H('folder-check', `Imported · score ${c.score}, below cutoff ${c.cutoff}`, 'Sep 21 · 19:02'), H('search', 'Searched 4 indexers · nothing above cutoff', 'Oct 6 · 06:00'), H('circle-arrow-up', `Found ${c.found.score} on ${c.found.indexer}`, 'Oct 10 · 06:00', true)];
    case 'missing': return {
      searching: [H('search', 'Searched 4 indexers · nothing matching this Edition', 'Oct 10 · 06:00'), H('clock', 'Next search', 'Oct 11 · 06:00', true)],
      'not-found': [H('search', 'Searched 4 indexers · no releases', 'Oct 10 · 18:00'), H('search', 'Searched 4 indexers · no releases', 'Oct 11 · 00:00', true)],
      waiting: [H('hourglass', `Waiting on ${nameOf(t.from ?? M.tgs[0].id)}`, 'Oct 8 · 20:52', true)],
      grace: [H('search', 'Searched for a release at this Target · none yet', 'Oct 9 · 12:00'), H('hourglass', `Grace period of ${t.grace ?? '3 days'} running`, 'Oct 8 · 09:00', true)],
      queued: [H('folder-check', 'Source file imported', 'Oct 9 · 21:40'), H('list-ordered', 'Queued for Conversion on forge', 'Oct 9 · 21:41', true)],
    }[c.why];
    case 'downloading': return [H('magnet', `Grabbed · score ${c.score}`, 'Oct 11 · 01:58'), H('download', `${jobOf(c)?.stage ?? 'Downloading'} on ${jobOf(c)?.node ?? 'seedbox-ams'}`, 'Oct 11 · 02:01', true)];
    case 'converting': return [H('folder-check', `Imported the ${esc(c.from)}`, 'Oct 11 · 01:40'), H('replace', `Conversion started on ${c.node}`, 'Oct 11 · 02:06', true)];
    case 'failed': return [H('magnet', 'Grabbed from Torrentleech', `${md(c.at)} · 14:20`), H('circle-x', `Failed at ${c.stage}`, `${md(c.at)} · ${c.at.slice(11, 16)}`), H('ban', 'Blocklisted the release · searching again', `${md(c.at)} · ${c.at.slice(11, 16)}`, true)];
    case 'unaired': return [H('calendar-plus', 'Added to the calendar', 'Sep 12 · 10:00'), H('calendar-clock', `Search starts when it airs on ${md(c.airs)}`, md(c.airs), true)];
    default: return [];
  }
}

function lead(x) {
  const c = x.cell;
  switch (c.state) {
    case 'met': return 'Cutoff reached, no upgrades wanted';
    case 'upgrade': return `A release scoring ${c.found.score} beats the file's ${c.score}`;
    case 'missing': return MISSING_WHY[c.why];
    case 'downloading': return `${jobOf(c)?.stage ?? 'Downloading'} on ${jobOf(c)?.node ?? ''}`;
    case 'converting': return `Converting on ${c.node}`;
    case 'failed': return `Failed at ${c.stage} on ${md(c.at)}`;
    case 'unaired': return `Airs ${md(c.airs)}`;
    default: return '';
  }
}

function body(x) {
  const c = x.cell, j = jobOf(c);
  const spec = (s) => [['Video', `<span class="mono">${esc(s.video)}</span>`], ['Audio', `<span class="mono">${esc(s.audio)}</span>`], ['Size', `<span class="num">${esc(s.size)}</span>`], ['Source', s.source === 'derived' ? 'Derived' : 'Downloaded']];
  const liveBlock = (rateLabel) => `<div class="ix-live"><span class="ix-pct num" data-live-pct="${c.job}"></span><span class="t2 num"><span data-live-rate="${c.job}"></span> · <span data-live-eta="${c.job}"></span> left</span></div><div class="bar" role="progressbar" aria-valuemin="0" aria-valuemax="100" aria-label="${rateLabel} progress" data-live-progress="${c.job}"><span class="fill"></span></div>`;
  switch (c.state) {
    case 'met': return `${meter(c)}${kv(spec(c))}${c.file ? file('File', c.file) : ''}
      <div class="ix-act">${btn('Manual search', 'list-filter', 'ghost')}${btn('Show file', 'folder-open', 'ghost')}</div>`;
    case 'upgrade': return `${meter(c)}${kv([['Now', `<span class="num">${c.score}</span> <span class="t3">of cutoff ${c.cutoff}</span>`], ['Found', `<span class="num draft-t">${c.found.score}</span> <span class="t3">· ${esc(c.found.size)} · ${esc(c.found.indexer)}</span>`], ...spec(c).slice(0, 2)])}
      ${file('Found release', c.found.release)}${c.file ? file('Current file', c.file, `<p class="t3 num">${esc(c.size)}</p>`) : ''}
      <div class="ix-act">${btn('Grab upgrade', 'download', 'pri', `Grabbing ${c.found.release} is not in this prototype`)}${btn('Manual search', 'list-filter')}</div>`;
    case 'missing': return `<p class="ix-why">${esc(c.detail)}</p>
      <div class="ix-act">${['searching', 'not-found'].includes(c.why) ? btn('Search now', 'search', 'pri') : ''}${btn('Manual search', 'list-filter')}${c.why === 'grace' ? btn('Derive now', 'replace', 'ghost') : ''}</div>`;
    case 'downloading': return `${liveBlock(j?.stage ?? 'Download')}${kv([['Stage', esc(j?.stage ?? '')], ['Node', `<span class="mono">${esc(j?.node ?? '')}</span>`], ['Size', `<span class="num">${esc(c.size)}</span>`], ['Score', `<span class="num">${c.score}</span>`]])}${file('Release', c.release)}
      <div class="ix-act">${btn('Pause', 'pause')}${btn('Cancel', 'circle-x', 'ghost')}</div>`;
    case 'converting': return `${liveBlock('Conversion')}${kv([['Source', esc(c.from)], ['Node', `<span class="mono">${esc(c.node)}</span>`], ['Settings', `<span class="mono">${esc(c.settings)}</span>`], ['Size', `<span class="num">${esc(c.estSize)}</span>`]])}
      <div class="ix-act">${btn('Pause Conversion', 'pause')}</div>`;
    case 'failed': return `<div class="ix-fail"><p><b>${esc(c.cause)}</b></p><p class="ix-rec">${I('rotate-cw')}${esc(c.recovery)}</p></div>
      ${kv([['Stage', esc(c.stage)], ['When', `<span class="num">${md(c.at)} · ${c.at.slice(11, 16)}</span>`]])}${file('Release', c.release)}
      <div class="ix-act">${btn('Manual search', 'search', 'fail')}${btn('Remove from Blocklist', 'ban')}${btn('Show log', 'scroll-text', 'ghost')}</div>`;
    case 'unaired': return `<p class="ix-why">Search is gated by the air date. It starts automatically on <b class="num">${md(c.airs)}</b>.</p>
      <div class="ix-act">${btn('Search anyway', 'search', 'ghost')}</div>`;
    default: return '';
  }
}

function detail(x) {
  const t = x.target, hs = events(x);
  return `<section class="ix-d ix-d-${x.cell.state}" aria-label="Selected cell">
    <p class="ix-dt"><b>${esc(x.edition.name)}</b><span class="t3"> at </span><b>${esc(t.name)}</b></p>
    <div class="ix-dh">${chip(x.cell.state)}<span class="t2">${esc(lead(x))}</span></div>
    <p class="ix-src t3">${I(SOURCING_ICON[t.sourcing])}<span><b>${esc(t.short)} sourcing</b> ${esc(t.label)}</span></p>
    ${body(x)}
    <h4 class="ix-h4">History</h4>
    <ol class="tl ix-tl">${[...hs].reverse().map((h) => `<li class="tl-i${h.live ? ' live' : ''}"><span class="tl-dot">${I(h.icon)}</span><div class="tl-b"><p>${h.text}</p><p class="tl-m"><time class="num t3">${esc(h.when)}</time></p></div></li>`).join('')}</ol>
  </section>`;
}

function rollup(g) {
  const grp = M.groups[g], list = cellsOf(grp.idx), c = counts(list);
  const shown = Object.keys(STATES).filter((s) => c[s]);
  const rows = M.tgs.map((t) => {
    const tc = counts(list.filter((x) => x.target.id === t.id));
    return `<tr><th scope="row"><b>${esc(t.short)}</b><span class="t3" title="${esc(t.label)}">${I(SOURCING_ICON[t.sourcing])}${esc(srcLine(t))}</span></th>${shown.map((s) => `<td class="num${tc[s] ? '' : ' t3'}">${tc[s] || '·'}</td>`).join('')}</tr>`;
  }).join('');
  const attn = list.filter((x) => (STATES[x.cell.state].attention || STATES[x.cell.state].live) && x.cell.why !== 'waiting').sort((a, b) => STATES[a.cell.state].rank - STATES[b.cell.state].rank).slice(0, 7);
  return `<section class="ix-d" aria-label="${esc(groupLabel(g))} roll-up">
    <p class="ix-crumb t3">${esc(V.desc.levels[0].noun)} · ${grp.idx.length} ${esc(V.desc.item.noun.toLowerCase())}s</p>
    <h3 class="ix-title">${esc(groupLabel(g))}</h3>
    <p class="t2 ix-sum">${summaryLine(c).map(esc).join(' · ')}</p>
    <div class="ix-act">${btn(`Search ${groupLabel(g)}`, 'search', 'pri')}${btn('Manual search', 'list-filter')}${btn('Unmonitor', 'eye-off', 'ghost')}</div>
    <h4 class="ix-h4">By Target</h4>
    <table class="ix-roll"><thead><tr><th></th>${shown.map((s) => `<th scope="col" title="${STATES[s].label}"><span class="sq sq-${s}"></span></th>`).join('')}</tr></thead><tbody>${rows}</tbody></table>
    <h4 class="ix-h4">Needs a look</h4>
    <ul class="ix-attn">${attn.map((x) => `<li><button data-ix="${keyOf(x)}">${chip(x.cell.state, true)}<span class="num t2">${esc(x.code)}</span><span class="ell">${esc(x.edition.name)} · ${esc(x.target.short)}</span></button></li>`).join('')}</ul>
  </section>`;
}

function inspector() {
  const p = parse(S.key);
  if (p.kind === 'l') return rollup(p.g);
  const x = p.x;
  if (!M.many) return detail(x);
  const it = M.its[x.i];
  return `<div class="ix-item">
      <p class="ix-crumb t3"><button data-ix="l:${groupOfItem(x.i)}">${esc(groupLabel(groupOfItem(x.i)))}</button>${I('chevron-right')}<span class="num">${esc(it.code)}</span>${it.item.airs ? `<span> · aired ${md(it.item.airs)}</span>` : ''}</p>
      <h3 class="ix-title">${esc(it.label)}</h3>
      ${matrix(x.i, 'sm')}
    </div>${detail(x)}`;
}


export function wide(vctx) {
  S.phone = false;
  build(vctx);
  return `<section class="ix" aria-label="Editions by Targets"><div class="ix-ov">${overview()}</div><aside class="ix-insp" aria-label="Inspector">${inspector()}</aside></section>`;
}

export function phone(vctx) {
  S.phone = true;
  build(vctx);
  return `<section class="ix ix-ph psec" aria-label="Editions by Targets"><div class="psec-h"><h2>${M.many ? `${M.its.length} ${esc(V.desc.item.noun)}s` : 'Editions'}</h2><span class="t3 num">${M.eds.length} × ${M.tgs.length}</span></div><div class="ix-ov">${overview()}</div></section>`;
}

function prime(el) {
  for (const j of V.data.jobs) {
    el.querySelectorAll(`[data-live-progress="${j.id}"]`).forEach((n) => n.style.setProperty('--p', j.progress));
    el.querySelectorAll(`[data-live-pct="${j.id}"]`).forEach((n) => { n.textContent = V.format.pct(j.progress); });
    el.querySelectorAll(`[data-live-eta="${j.id}"]`).forEach((n) => { n.textContent = V.format.eta(j.etaMin); });
    el.querySelectorAll(`[data-live-rate="${j.id}"]`).forEach((n) => { n.textContent = V.format.rate(j); });
  }
}

function paint(focusKey) {
  root.querySelector('.ix .ix-ov').innerHTML = overview();
  if (S.phone) {
    const wrap = root.querySelector('.sheet-wrap');
    if (!wrap.hidden) wrap.querySelector('.sheet').innerHTML = sheetHtml();
  } else root.querySelector('.ix-insp').innerHTML = inspector();
  window.lucide && window.lucide.createIcons();
  prime(root);
  if (focusKey) root.querySelector(`[data-ix="${CSS.escape(focusKey)}"]`)?.focus({ preventScroll: true });
}

const sheetHtml = () => `<span class="grab" aria-hidden="true"></span><div class="ix-sheet-b">${inspector()}</div><button class="sheet-x" data-close>Done</button>`;

function select(k, focusKey) {
  S.key = k.startsWith('e:') ? urgentKey([+k.slice(2)]) : k;
  history.replaceState(null, '', `#ix=${encodeURIComponent(S.key)}`);
  if (S.phone) {
    const wrap = root.querySelector('.sheet-wrap');
    const s = wrap.querySelector('.sheet');
    s.classList.add('ix-sheet');
    s.innerHTML = sheetHtml();
    const opening = wrap.hidden;
    wrap.hidden = false;
    paint(focusKey);
    if (opening) s.scrollTop = 0;
    return;
  }
  paint(focusKey);
  if (!S.phone) root.querySelector('.ix-insp').scrollTop = 0;
}

export function wire(r, vctx) {
  root = r;
  build(vctx);
  prime(root);
  root.addEventListener('click', (e) => {
    const f = e.target.closest('[data-ixf]');
    if (f) { S.filter = f.dataset.ixf; return paint(); }
    const b = e.target.closest('[data-ix]');
    if (b) select(b.dataset.ix, b.dataset.ix);
  });
  root.addEventListener('keydown', (e) => {
    const g = e.target.closest && e.target.closest('[data-grid]');
    const dirs = { ArrowLeft: [0, -1], ArrowRight: [0, 1], ArrowUp: [-1, 0], ArrowDown: [1, 0] };
    if (!g || !dirs[e.key] || !e.target.dataset.r) return;
    e.preventDefault();
    e.stopPropagation();
    const [dr, dc] = dirs[e.key];
    const r0 = +e.target.dataset.r, c0 = +e.target.dataset.c;
    const all = [...g.querySelectorAll('[data-r]')];
    const rows = [...new Set(all.map((n) => +n.dataset.r))];
    const r1 = rows[Math.max(0, Math.min(rows.length - 1, rows.indexOf(r0) + dr))];
    const inRow = all.filter((n) => +n.dataset.r === r1);
    const cs = inRow.map((n) => +n.dataset.c);
    const want = dr ? c0 : c0 + dc;
    const c1 = Math.max(Math.min(...cs), Math.min(Math.max(...cs), want));
    const next = inRow.find((n) => +n.dataset.c === c1);
    if (next && next !== e.target) select(next.dataset.ix, next.dataset.ix);
  });
}
