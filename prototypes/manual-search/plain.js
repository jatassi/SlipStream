// PROTOTYPE · round 2 · pieces D, E and F share: the sheet frame, the one-line search status, a Release in plain words, its outcome, Details one step down, and the folded "wouldn't grab" list. Throwaway.

import { I, esc, toast, openMenu } from './shell.js';
import { rname, breakdown, overrides, clientFor, doGrab } from './common.js';
import { describe, outcome, wouldGrab, hiddenGroups, needKeys, scopeOptions, normScope, fmt, fills, itemById, parseCell, targetById, KINDS } from './model.js';

export { I, esc };

// ───────────────────────────── frame

export function scopeName(fx, sc) {
  if (sc.kind === 'work') return fx.work.title;
  if (sc.kind === 'level') return fx.levels.find((l) => l.id === sc.level).label;
  const it = itemById(fx, sc.item);
  if (sc.kind === 'item') return `Episode ${it.ep}`;
  const c = parseCell(sc.cell);
  const t = targetById(fx, c.target).name;
  return fx.work.kind === 'movie' ? `${t}${fx.editions.length > 1 ? ` (${c.edition})` : ''}` : `Episode ${it.ep} in ${t}`;
}

function scopeMenu(app, anchor) {
  const opts = scopeOptions(app.fx, app.sc);
  openMenu(anchor, 'Search for', opts.map((o) => ({ label: scopeName(app.fx, normScope(app.fx, { ...app.sc, kind: o.kind })), sub: o.kind === 'work' && app.fx.work.kind !== 'movie' ? 'Every season' : '', icon: o.kind === app.sc.kind ? 'Tick02' : null, run: () => app.openAt({ kind: o.kind }) })));
}

export function header(app, opts = {}) {
  const { fx, sc } = app;
  const scope = opts.scope === false ? '' : `${app.phone ? '' : '<span class="s-sep">·</span>'}<button class="r2-scope" data-scope-menu aria-haspopup="menu">${esc(sc.kind === 'work' ? (fx.work.kind === 'movie' ? 'Everything' : 'Every season') : scopeName(fx, sc))}${I('ArrowDown01', 'ic-s')}</button>`;
  return `<header class="r2-h"><div class="r2-hl"><p class="r2-ctx">${app.phone ? `<span class="r2-kick">Manual search</span><b>${esc(fx.work.title)}</b>` : `<b>${esc(fx.work.title)}</b><span class="s-sep">·</span>Manual search`}${scope}</p>${searchLine(app)}</div><button class="ib${app.phone ? ' tap' : ''}" data-close-ms aria-label="Close">${I('Cancel01')}</button></header>`;
}

export function frame(app, { head, body, foot = '', cls = '' }) {
  const box = app.phone ? `psheet r2-sheet ${cls}` : `dlg2 ${cls}`;
  return `<div class="ms-wrap"><div class="ms-scrim" data-close-ms></div><div class="${box}" role="dialog" aria-modal="true" aria-label="Manual search">${app.phone ? '<span class="grab" aria-hidden="true"></span>' : ''}${head}<div class="r2-scroll" data-keep="r2">${body}</div>${foot}</div></div>`;
}

// ───────────────────────────── search status, one line

export const searching = (app) => app.indexers.some((x) => !app.arrived.has(x.id));

export function searchLine(app) {
  const ix = app.indexers;
  const done = ix.filter((x) => app.arrived.has(x.id)).length;
  const failed = ix.filter((x) => app.arrived.has(x.id) && x.outcome !== 'ok').length;
  if (done < ix.length) return `<p class="r2-status" aria-live="polite"><span class="dot dot-draft r2-pulse" aria-hidden="true"></span><span class="draft-t">Searching ${ix.length} Indexers…</span><span class="t3">${done} answered</span></p>`;
  return `<p class="r2-status" aria-live="polite"><span>Searched ${ix.length} Indexers</span>${failed ? `<span class="s-sep">·</span><span class="r2-sig"><span class="dot dot-fail" aria-hidden="true"></span>${failed} didn’t answer</span>` : ''}<span class="s-sep">·</span><button class="lnk r2-lnk" data-ix-pop><b>Details</b></button></p>`;
}

function ixPop(app, anchor) {
  const rows = [...app.indexers].sort((a, b) => (a.outcome === 'ok' ? 0 : 1) - (b.outcome === 'ok' ? 0 : 1)).map((x) => {
    if (x.outcome === 'ok') return `<div class="ixp-r"><span class="ixp-n">${esc(x.name)}</span><span class="t3">${app.countFor(x.id)} found</span></div>`;
    const ch = x.outcome === 'challenge';
    return `<div class="ixp-r ixp-bad"><span class="chip-fail">${I(ch ? 'Shield01' : 'Alert02')}${ch ? 'Blocked' : 'Error'}</span><span class="ixp-n">${esc(x.name)}</span><button class="btn sm" data-toast="${ch ? 'Opening the Indexer is not in this prototype' : `Retrying ${x.name} at 21:52`}">${ch ? 'Open Indexer' : 'Retry'}</button><span class="ixp-m">${esc(x.msg)}</span></div>`;
  }).join('');
  openMenu(anchor, 'Indexers', `<div class="ixp">${rows}</div>`);
}

// ───────────────────────────── a Release in words

export function facts(app, r, opts = {}) {
  const d = describe(app.fx, r);
  const bits = [...d.parts.map((x, i) => (i === 0 ? `<b>${esc(x)}</b>` : `<b class="r2-b2">${esc(x)}</b>`)), `<span class="num r2-size">${esc(d.size)}</span>`];
  if (d.health.low) bits.push(`<span class="r2-low">${I('Alert02', 'ic-s')}${esc(d.health.word)}</span>`);
  return `${opts.tag || ''}<span class="r2-facts">${bits.map((b, i) => `<span class="r2-fp">${b}${i < bits.length - 1 ? '<span class="s-sep">·</span>' : opts.chev ? I(opts.chev, 'r2-fchev') : ''}</span>`).join('')}</span>`;
}

export function outLines(o, opts = {}) {
  return `<span class="r2-out${o.kind === 'cant' || o.kind === 'blocked' ? ' r2-out-no' : ''}">${esc(o.text)}</span>${o.also && opts.also !== false ? `<span class="r2-also">${I('PlusSign', 'ic-s')}${esc(o.also)}</span>` : ''}${o.note ? `<span class="r2-note">${I('Clock01', 'ic-s')}${esc(o.note)}</span>` : ''}`;
}

export const rawName = (r) => `<span class="r2-raw mono" title="${esc(r.name)}">${esc(r.name)}</span>`;

export function grabBtn(app, row, opts = {}) {
  const g = app.grabbed.get(row.r.id);
  if (g) return `<span class="chip chip-live r2-onway"><span class="dot dot-draft"></span>On its way</span>`;
  return `<button class="btn${opts.primary ? ' pri' : ''}${opts.sm && !app.phone ? ' sm' : ''} r2-grab" data-grab="${row.r.id}">${opts.primary ? I('Download04') : ''}${esc(opts.label || 'Grab')}</button>`;
}

export const recTag = () => `<span class="chip r2-rec">${I('ThumbsUp', 'ic-s')}Recommended</span>`;

// ───────────────────────────── Details, one step down

function perTarget(app, row) {
  return app.fx.targets.map((t) => {
    const v = row.v[t.id];
    const word = v.kind === 'fill' ? 'Would grab' : v.hold != null ? `Would wait ${v.hold} min` : v.kind === 'soft' ? 'Would skip' : v.kind === 'meets' ? 'No better than yours' : v.kind === 'block' ? 'Blocked' : 'Can’t use';
    return `<div class="r2-pt"><span class="r2-pt-h"><b>${esc(t.name)}</b><span class="r2-pt-w${v.kind === 'hard' || v.kind === 'block' ? ' t3' : ''}">${esc(word)}</span><span class="t3 num">Score ${fmt.score(v.score)}</span></span><span class="t3">${esc(v.full)}.</span></div>`;
  }).join('');
}

export function details(app, row) {
  const { r } = row;
  const ix = app.indexers.find((x) => x.id === r.indexer);
  const client = app.clients.find((c) => c.id === clientFor(app, r));
  const kv = [
    ['Found on', esc(ix.name)],
    ['Sharing', `<span class="num">${r.seeders}</span> seeders · <span class="num">${r.leechers}</span> downloading`],
    ['Uploaded', esc(r.age < 60 ? `${r.age} min ago` : r.age < 60 * 48 ? `${Math.round(r.age / 60)} hours ago` : `${Math.round(r.age / 1440)} days ago`)],
    ['Size', `<span class="num">${fmt.size(r.size)}</span>`],
    ['Video', `${esc(r.res)} · ${esc(r.source)} · ${esc(r.codec)} · ${esc(r.hdr || 'SDR')}`],
    ['Audio', esc(r.audio)],
    ...(r.freeleech ? [['Freeleech', 'Yes, it won’t count against your ratio']] : []),
    ['Sends to', `${esc(client.name)}${app.clientFor.has(r.id) ? '' : ` <span class="t3">(usual for ${esc(ix.name)})</span>`}`],
  ].map(([k, v]) => `<div><dt>${k}</dt><dd>${v}</dd></div>`).join('');
  const more = overrides(app, row).length ? `<button class="btn${app.phone ? '' : ' sm'}" data-more="${r.id}" aria-haspopup="menu">${I('MoreHorizontal')}More options</button>` : '';
  return `<div class="r2-det"><p class="r2-raw-full mono">${rname(r.name)}</p><dl class="r2-kv">${kv}</dl>
    <h4 class="r2-dh">How SlipStream judged it</h4><div class="r2-pts">${perTarget(app, row)}</div>
    <details class="r2-bd"><summary>Ranking breakdown</summary>${breakdown(app, row)}</details>
    ${more ? `<div class="r2-det-a">${more}</div>` : ''}</div>`;
}

export const detBtn = (app, r) => `<button class="btn sm ghost r2-db" data-det="${r.id}" aria-expanded="${app.det === r.id}">${esc(app.det === r.id ? 'Hide details' : 'Details')}${I(app.det === r.id ? 'ArrowUp01' : 'ArrowDown01', 'ic-s')}</button>`;

// ───────────────────────────── what SlipStream wouldn't grab, folded to one line

function secondary(app, row, o) {
  if (app.grabbed.get(row.r.id)) return grabBtn(app, row);
  if (o.kind === 'soft') return `<button class="btn sm" data-anyway="${row.r.id}">Grab anyway</button>`;
  if (o.kind === 'blocked') return `<button class="btn sm" data-unblock="${row.r.id}">Unblock and grab</button>`;
  return '';
}

function hiddenRow(app, x) {
  const { row, out } = x;
  const open = app.det === row.r.id;
  return `<div class="r2-hr${open ? ' open' : ''}"><div class="r2-hr-m"><button class="r2-hr-b" data-det="${row.r.id}" aria-expanded="${open}">${facts(app, row.r, { health: false })}${outLines(out, { also: false })}</button><span class="r2-hr-a">${secondary(app, row, out)}</span></div>${open ? details(app, row) : ''}</div>`;
}

export function hiddenBlock(app, items, key = 'all', label = 'SlipStream wouldn’t grab') {
  if (!items.length) return '';
  const open = app.hiddenOpen === key;
  const head = `<button class="r2-fold" data-hidden="${key}" aria-expanded="${open}">${I(open ? 'ArrowDown01' : 'ArrowRight01', 'ic-s')}<span>${items.length} more ${esc(label)}</span><span class="s-sep">·</span><b>${open ? 'Hide' : 'Show'}</b></button>`;
  if (!open) return `<div class="r2-hidden">${head}</div>`;
  const groups = hiddenGroups(app.fx, items).map((g) => `<section class="r2-hg"><h4>${esc(g.sentence)}</h4>${g.rows.map((x) => hiddenRow(app, x)).join('')}</section>`).join('');
  return `<div class="r2-hidden open">${head}${groups}</div>`;
}

export const judgeAll = (app, rows, focus) => rows.map((row) => ({ row, out: outcome(app.fx, row, focus, app.sc) }));
export { wouldGrab, needKeys };

// ───────────────────────────── shared clicks; variants handle their own first

export function wire(app, own) {
  app.variantClick = (e, q) => {
    let el;
    if (own && own(e, q)) return true;
    if ((el = q('[data-scope-menu]'))) { e.stopPropagation(); scopeMenu(app, el); return true; }
    if ((el = q('[data-ix-pop]'))) { e.stopPropagation(); ixPop(app, el); return true; }
    if ((el = q('[data-det]'))) { app.det = app.det === el.dataset.det ? null : el.dataset.det; app.render(); return true; }
    if ((el = q('[data-hidden]'))) { app.hiddenOpen = app.hiddenOpen === el.dataset.hidden ? null : el.dataset.hidden; app.render(); return true; }
    if ((el = q('[data-anyway]'))) { const row = app.rowById(el.dataset.anyway); const ts = app.fx.targets.filter((t) => row.v[t.id].kind === 'soft'); doGrab(app, row.r, ts.map((t) => t.name).join(', '), ' past the limit you set'); return true; }
    if ((el = q('[data-unblock]'))) { const row = app.rowById(el.dataset.unblock); const o = overrides(app, row).find((x) => x.icon === 'Unavailable'); o.run(); return true; }
    return false;
  };
}

export function escapeShared(app) {
  if (!app.det) return false;
  app.det = null;
  app.render();
  return true;
}

export { toast, fills, KINDS };
