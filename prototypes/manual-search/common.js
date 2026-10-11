// PROTOTYPE · pieces every variant of the manual search sheet shares: chips, controls, breakdown, grab and overrides, Indexer footer. Throwaway.

import { I, esc, toast, openMenu } from './shell.js';
import { KINDS, fmt, judge, fills, grabLabel, rankForTarget, qualityName, qualityRank, cellStateLine, scopeOptions, SORTS, parseCell, itemById, targetById } from './model.js';

export { I, esc };
export const rname = (s) => esc(s).replace(/([.\-_])/g, '$1<wbr>');

// ───────────────────────────── chips and squares

export function vchip(v, size = 'sm', label) {
  const k = KINDS[v.kind];
  const icon = v.hold != null ? 'Clock01' : v.shape ? 'Remove01' : k.icon;
  const word = label || (v.kind === 'fill' ? 'Would fill' : v.kind === 'meets' ? 'Meets, not better' : v.short);
  return `<span class="vc ${vcls(v)} vc-${size}" title="${esc(v.full)}">${I(icon)}<span>${esc(word)}</span></span>`;
}
export const vstrip = (v) => vchip(v, 'xs', v.kind === 'fill' ? (v.fillItems.length > 1 ? `Fills ${v.fillItems.length}` : 'Fills') : v.kind === 'meets' ? 'Meets' : v.short);

export const sq = (state, cls = '') => `<span class="sq sq-${state} ${cls}" aria-hidden="true"></span>`;
export const vcls = (v) => (v.shape ? 'v-off' : KINDS[v.kind].cls);
export const vsq = (v) => `<span class="vsq ${vcls(v)}" aria-hidden="true"></span>`;

export function stateBadge(c) {
  if (c.state === 'failed') return `<span class="chip-fail">${I('Cancel01')}Failed</span>`;
  if (c.state === 'met') return `<span class="badge b-met">${I('Tick02')}Met</span>`;
  if (c.state === 'upgrade') return `<span class="badge b-up">${sq('upgrade')}Upgrade</span>`;
  return `<span class="badge b-miss">${sq('missing')}Missing</span>`;
}

export const facts = (r) => [r.res, r.source, r.codec, r.hdr || 'SDR', r.audio, r.group].map(esc).join('<span class="s-sep">·</span>');

export function stats(r, opts = {}) {
  const seeds = `<span class="num" title="Seeders and leechers">${I('ArrowUp01', 'ic-s')}${r.seeders}<span class="t3"> / ${r.leechers}</span></span>`;
  const fl = r.freeleech ? `<span class="fl">${I('Gift', 'ic-s')}Freeleech</span>` : '';
  return `<span class="num">${fmt.size(r.size)}</span><span class="s-sep">·</span>${seeds}<span class="s-sep">·</span><span class="num">${fmt.age(r.age)}</span>${opts.indexer ? `<span class="s-sep">·</span><span>${esc(opts.indexer)}</span>` : ''}${fl ? `<span class="s-sep">·</span>${fl}` : ''}`;
}

export const edTag = (app, r) => (app.fx.work.kind === 'movie' || r.edition !== 'Standard' ? `<span class="ed${r.misparsed ? ' ed-x' : ''}" title="Parsed Edition">${esc(r.edition)}</span>` : '');
export const covTag = (row) => `<span class="cov">${esc(row.coverage)}</span>`;
export const ixName = (app, r) => app.indexers.find((x) => x.id === r.indexer).name;

export function grabbedChip(app, r) {
  const g = app.grabbed.get(r.id);
  return g ? `<span class="chip chip-live"><span class="dot dot-draft"></span>Grabbed · ${esc(g)}</span>` : '';
}

// ───────────────────────────── controls

export function scopeSeg(app, cls = '') {
  const opts = scopeOptions(app.fx, app.sc);
  return `<div class="seg-c scope-seg ${cls}" role="radiogroup" aria-label="Scope">${opts.map((o) => `<button class="seg-o" role="radio" aria-checked="${o.kind === app.sc.kind}" data-scope="${o.kind}">${esc(o.label)}</button>`).join('')}</div>`;
}

export function filterSeg(app, cls = '') {
  const all = app.rows();
  const w = all.filter((x) => x.wanted).length;
  return `<div class="seg-c ${cls}" role="radiogroup" aria-label="Filter"><button class="seg-o" role="radio" aria-checked="${app.filter === 'wanted'}" data-filter="wanted">Wanted<span class="seg-n num">${w}</span></button><button class="seg-o" role="radio" aria-checked="${app.filter === 'all'}" data-filter="all">All<span class="seg-n num">${all.length}</span></button></div>`;
}

export const sortBtn = (app) => `<button class="btn sm ghost sort-b" data-sort-menu>${I('Sorting01')}${esc(SORTS[app.sort].label)}</button>`;

export function sortMenu(app, anchor) {
  openMenu(anchor, 'Sort Releases', Object.entries(SORTS).map(([id, s]) => ({ label: s.label, icon: id === app.sort ? 'Tick02' : null, run: () => { app.sort = id; app.render(); } })));
}

export function searchStatus(app) {
  const ix = app.indexers;
  const pending = ix.filter((x) => !app.arrived.has(x.id));
  const failed = ix.filter((x) => app.arrived.has(x.id) && x.outcome !== 'ok');
  const n = app.rows().length;
  if (pending.length) return `<span class="ms-status"><span class="spin" aria-hidden="true"></span><span><b class="num-t">${n}</b> Releases so far · ${ix.length - pending.length} of ${ix.length} Indexers answered</span></span>`;
  return `<span class="ms-status"><span><b class="num-t">${n}</b> Releases from ${ix.length - failed.length} Indexers${failed.length ? `<span class="s-sep">·</span><span class="fail-t">${failed.length} failed</span>` : ''}</span></span>`;
}

export function ixFooter(app) {
  const parts = app.indexers.map((x) => {
    if (!app.arrived.has(x.id)) return `<span class="ix ix-wait"><span class="spin" aria-hidden="true"></span>${esc(x.name)}</span>`;
    if (x.outcome === 'ok') return `<span class="ix">${esc(x.name)}<span class="num t3">${app.countFor(x.id)}</span></span>`;
    return '';
  }).join('');
  const errs = app.indexers.filter((x) => app.arrived.has(x.id) && x.outcome !== 'ok').map((x) => `<div class="ix-err"><span class="chip-fail">${I(x.outcome === 'challenge' ? 'Shield01' : 'Alert02')}${x.outcome === 'challenge' ? 'Challenge' : 'Error'}</span><b>${esc(x.name)}</b><span class="ix-msg">${esc(x.msg)}</span><button class="btn sm ix-b" data-toast="${x.outcome === 'challenge' ? 'Opening the Indexer is not in this prototype' : `Retrying ${x.name} at 21:52`}">${x.outcome === 'challenge' ? 'Open Indexer' : 'Retry'}</button></div>`).join('');
  return `<div class="ix-foot"><div class="ix-row">${parts}</div>${errs ? `<div class="ix-errs">${errs}</div>` : ''}</div>`;
}

// ───────────────────────────── per-Target verdicts and the ranking breakdown

export function verdictBlocks(app, row) {
  return app.fx.targets.map((t) => {
    const v = row.v[t.id];
    return `<div class="vb ${vcls(v)}-b"><div class="vb-h"><b>${esc(t.name)}</b>${vchip(v, 'sm')}</div><p class="vb-r">${esc(v.full)}.</p></div>`;
  }).join('');
}

export function breakdown(app, row) {
  const ts = app.fx.targets;
  const r = row.r;
  const all = app.rows();
  const td = (t, s) => `<td class="num${row.v[t.id].kind === 'hard' ? ' bd-x' : ''}">${s}</td>`;
  const cfRows = r.cfs.length ? r.cfs.map((cf) => `<tr><th>${esc(cf)}</th>${ts.map((t) => { const s = (app.fx.cf[cf] || {})[t.id] || 0; return td(t, s ? fmt.signed(s) : '0'); }).join('')}</tr>`).join('') : `<tr><th class="t3">No Custom Formats matched</th>${ts.map((t) => td(t, '0')).join('')}</tr>`;
  const rank = (t) => { const list = rankForTarget(all, t.id); return `${list.findIndex((x) => x.r.id === r.id) + 1} of ${list.length}`; };
  return `<div class="bd-wrap"><table class="bd">
    <thead><tr><th>Ranking</th>${ts.map((t) => `<th>${esc(t.name)}</th>`).join('')}</tr></thead>
    <tbody>${cfRows}
      <tr class="bd-sum"><th>Custom Format score</th>${ts.map((t) => td(t, fmt.score(row.v[t.id].score))).join('')}</tr>
      <tr><th>Quality</th>${ts.map((t) => `<td>${esc(qualityName(r))}<span class="t3"> · #${qualityRank(r)}</span></td>`).join('')}</tr>
      <tr><th>Seeders</th>${ts.map((t) => td(t, r.seeders)).join('')}</tr>
      <tr><th>Size</th>${ts.map((t) => td(t, fmt.size(r.size))).join('')}</tr>
      <tr><th>Age</th>${ts.map((t) => td(t, fmt.age(r.age))).join('')}</tr>
      <tr><th>Freeleech</th>${ts.map((t) => `<td>${r.freeleech ? 'Yes' : 'No'}</td>`).join('')}</tr>
      <tr class="bd-sum"><th>Place among these Releases</th>${ts.map((t) => `<td class="num">${rank(t)}</td>`).join('')}</tr>
    </tbody></table>
    <p class="bd-note t3">Ranked by verdict, then Custom Format score, quality, seeders, freeleech and age.</p></div>`;
}

// ───────────────────────────── grab and overrides

const clientName = (app, id) => app.clients.find((c) => c.id === id).name;
const defaultClient = (app, r) => app.indexers.find((x) => x.id === r.indexer).client;
export const clientFor = (app, r) => app.clientFor.get(r.id) || defaultClient(app, r);

export function doGrab(app, r, what, extra = '') {
  app.grabbed.set(r.id, what);
  app.selected.delete(r.id);
  toast(`Grabbed for ${what}${extra}. Sent to ${clientName(app, clientFor(app, r))}`);
  app.render();
}

function softWord(v) {
  const w = v.reasons[0][0];
  return w === 'Low score' ? 'the score' : w.startsWith('Too') ? 'the size' : 'the title match';
}

export function overrides(app, row) {
  const { r, v } = row;
  const fx = app.fx;
  const out = [];
  for (const t of fx.targets) {
    if (v[t.id].kind === 'meets') out.push({ label: `Grab for ${t.name}`, sub: v[t.id].full, icon: 'Target02', run: () => doGrab(app, r, t.name, ' as a forced grab') });
    if (v[t.id].kind === 'soft' && v[t.id].hold == null) out.push({ label: `Grab past ${softWord(v[t.id])} for ${t.name}`, sub: v[t.id].full, icon: 'Alert02', run: () => doGrab(app, r, t.name, ` past ${softWord(v[t.id])}`) });
  }
  if (r.blocklisted) out.push({ label: 'Remove from Blocklist and grab', sub: v[fx.targets[0].id].full, icon: 'Unavailable', danger: true, run: () => doGrab(app, r, fx.targets.filter((t) => { const j = judge(fx, { ...r, blocklisted: null }, t); return fills(j); }).map((t) => t.name).join(', ') || 'nothing', ' after removing it from the Blocklist') });
  if (fx.editions.length > 1 || r.misparsed) out.push({ label: 'Grab as another Edition…', sub: `Parsed as ${r.edition}`, icon: 'Film01', run: () => editionMenu(app, row) });
  if (!row.allHard || r.misparsed) out.push({ label: 'Send to another Download Client…', sub: `Now ${clientName(app, clientFor(app, r))}`, icon: 'Download04', run: () => clientMenu(app, row) });
  return out;
}

let menuAnchor = null;
export function moreMenu(app, row, anchor) {
  menuAnchor = anchor;
  openMenu(anchor, 'Grab options', overrides(app, row), row.r.name);
}

function editionMenu(app, row) {
  const fx = app.fx;
  const items = fx.editions.filter((e) => e !== row.r.edition).map((ed) => {
    const ts = fx.targets.filter((t) => fills(judge(fx, { ...row.r, edition: ed }, t)));
    return { label: `Grab as ${ed}`, sub: ts.length ? `Fills ${ts.map((t) => t.name).join(', ')}` : 'Fills nothing for this Edition', icon: 'Film01', disabled: !ts.length, run: () => doGrab(app, row.r, `${ed} · ${ts.map((t) => t.name).join(', ')}`) };
  });
  openMenu(menuAnchor, `Grab as another Edition · parsed as ${row.r.edition}`, items, row.r.name);
}

function clientMenu(app, row) {
  const r = row.r;
  const def = defaultClient(app, r);
  openMenu(menuAnchor, 'Send to another Download Client', app.clients.map((c) => ({ label: c.name, sub: c.id === def ? `Tracker routing for ${ixName(app, r)}` : '', icon: c.id === clientFor(app, r) ? 'Tick02' : 'Download04', run: () => { app.clientFor.set(r.id, c.id); toast(`This grab goes to ${c.name}`); app.render(); } })), r.name);
}

export function grabBlock(app, row, opts = {}) {
  const { r } = row;
  const label = grabLabel(app.fx, row);
  const held = app.fx.targets.some((t) => row.v[t.id].hold != null && fills(row.v[t.id]));
  const ov = overrides(app, row);
  const grabbed = app.grabbed.get(r.id);
  const pri = grabbed ? `<span class="chip chip-live grab-done"><span class="dot dot-draft"></span>Grabbed for ${esc(grabbed)}</span>`
    : label ? `<button class="btn pri grab-b" data-grab="${r.id}">${I('Download04')}${esc(label)}</button>`
      : `<button class="btn pri grab-b" disabled>${I('Download04')}Grab</button>`;
  const whyText = r.blocklisted ? `Blocklisted ${r.blocklisted.on}: ${r.blocklisted.cause}. Remove it from the Blocklist to grab it.`
    : r.misparsed ? `Parsed as ${r.edition}, which this Work doesn’t want. If the parse is wrong, grab it as another Edition.`
    : row.allHard ? `Meets no wanted Target’s rules, so it can’t be grabbed. ${app.fx.targets.map((t) => `${t.name}: ${row.v[t.id].short}`).join(' · ')}.`
      : 'Fills nothing as it stands. A grab option can force it.';
  const why = !label && !grabbed ? `<p class="grab-why${r.misparsed || !row.allHard ? ' t2' : ''}">${esc(whyText)}</p>` : '';
  const client = `<p class="grab-c t3">${I('Download04', 'ic-s')}<span>${esc(clientName(app, clientFor(app, r)))}${app.clientFor.has(r.id) ? '' : ` · Tracker routing for ${esc(ixName(app, r))}`}</span>${held ? '<span class="s-sep">·</span><span>A manual grab skips the Hold</span>' : ''}</p>`;
  const more = ov.length ? `<button class="btn${opts.touch ? '' : ' sm'} more-b" data-more="${r.id}" aria-haspopup="menu">${I('MoreHorizontal')}Grab options</button>` : '';
  return `<div class="grabx${opts.touch ? ' grab-touch' : ''}"><div class="grab-row">${pri}${more}</div>${why}${client}</div>`;
}

export function bulkBar(app) {
  if (!app.selected.size) return '';
  const rows = app.rows().filter((x) => app.selected.has(x.r.id));
  const ts = [...new Set(rows.flatMap((x) => x.fillTargets.map((t) => t.name)))];
  const none = rows.filter((x) => !x.fillTargets.length).length;
  return `<div class="bulk"><span class="bulk-n num-t">${rows.length} selected</span><span class="bulk-t">${ts.length ? `Fills ${esc(ts.join(', '))}` : 'Fills nothing'}${none ? `<span class="s-sep">·</span><span class="warn-t">${none} fill nothing and are skipped</span>` : ''}</span><button class="btn sm ghost" data-bulk-clear>Clear</button><button class="btn sm pri" data-bulk-grab ${rows.length - none ? '' : 'disabled'}>${I('Download04')}Grab ${rows.length - none}</button></div>`;
}

export const check = (app, r) => `<label class="ck" title="Select for bulk grab"><input type="checkbox" data-sel="${r.id}" ${app.selected.has(r.id) ? 'checked' : ''} aria-label="Select ${esc(r.name)}"><span aria-hidden="true">${I('Tick02')}</span></label>`;

// ───────────────────────────── shared event wiring; variants handle their own data-* first

export function wireCommon(app, host) {
  host.addEventListener('click', (e) => {
    const q = (s) => e.target.closest(s);
    let el;
    if (app.variantClick && app.variantClick(e, q)) return;
    if ((el = q('[data-scope]'))) { app.setScope({ kind: el.dataset.scope }); return; }
    if ((el = q('[data-filter]'))) { app.filter = el.dataset.filter; app.render(); return; }
    if ((el = q('[data-sort-menu]'))) { e.stopPropagation(); sortMenu(app, el); return; }
    if ((el = q('[data-grab]'))) { const row = app.rowById(el.dataset.grab); doGrab(app, row.r, row.fillTargets.map((t) => t.name).join(', ')); return; }
    if ((el = q('[data-more]'))) { e.stopPropagation(); moreMenu(app, app.rowById(el.dataset.more), el); return; }
    if ((el = q('[data-bulk-clear]'))) { app.selected.clear(); app.render(); return; }
    if ((el = q('[data-bulk-grab]'))) {
      const rows = app.rows().filter((x) => app.selected.has(x.r.id) && x.fillTargets.length);
      rows.forEach((x) => app.grabbed.set(x.r.id, x.fillTargets.map((t) => t.name).join(', ')));
      app.selected.clear();
      toast(`Grabbed ${rows.length} Releases. Each goes to its Tracker routing’s Download Client`);
      app.render();
      return;
    }
    if ((el = q('[data-close-ms]'))) { app.close(); return; }
  });
  host.addEventListener('change', (e) => {
    const el = e.target.closest('[data-sel]');
    if (!el) return;
    if (el.checked) app.selected.add(el.dataset.sel); else app.selected.delete(el.dataset.sel);
    app.render();
  });
}

export function cellName(app, key) {
  const c = parseCell(key);
  const t = targetById(app.fx, c.target);
  if (app.fx.work.kind === 'movie') return `${c.edition} · ${t.name}`;
  return `${itemById(app.fx, c.item).short} · ${app.fx.editions.length > 1 ? `${c.edition} · ` : ''}${t.name}`;
}
export { cellStateLine };
