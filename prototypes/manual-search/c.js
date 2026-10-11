// PROTOTYPE · C · Cell first. A rail lists the Edition × Target cells in scope with their state squares; picking one ranks the Releases for that cell, its verdict large and the other Targets as small squares. Phone: the cells become a sideways chip row. Throwaway.

import { I, esc, rname, vchip, vsq, sq, facts, stats, edTag, covTag, ixName, grabbedChip, scopeSeg, searchStatus, ixFooter, verdictBlocks, breakdown, grabBlock, bulkBar, check, cellName } from './common.js';
import { scopeCells, cellVerdict, rankForCell, cellStateLine, STATES, KINDS, parseCell, SORTS, fills } from './model.js';

const fillsCell = fills;

function cellsWithCounts(app) {
  const rows = app.rows();
  return scopeCells(app.fx, app.sc).map((c) => ({ ...c, n: rows.filter((x) => fillsCell(cellVerdict(app.fx, x, c.key))).length }));
}

function ranked(app) {
  const rows = app.rows();
  if (app.cell === 'all') {
    const list = [...rows].sort(SORTS.best.cmp).map((row) => ({ row, cv: null }));
    return app.filter === 'all' ? list : list.filter((x) => x.row.wanted);
  }
  const list = rankForCell(app.fx, rows, app.cell);
  return app.filter === 'all' ? list : list.filter((x) => fillsCell(x.cv));
}

function bigVerdict(app, x) {
  if (x.cv) return vchip(x.cv, 'lg');
  const n = new Set(x.row.fillTargets.flatMap((t) => x.row.v[t.id].fillItems.map((i) => `${i}|${t.id}`))).size;
  if (n) return `<span class="vc v-fill vc-lg">${I('Tick02')}<span>Fills ${n} cell${n > 1 ? 's' : ''}</span></span>`;
  const best = app.fx.targets.map((t) => x.row.v[t.id]).sort((a, b) => KINDS[a.kind].rank - KINDS[b.kind].rank)[0];
  return vchip(best, 'lg');
}

function others(app, x) {
  const skip = x.cv ? parseCell(app.cell).target : null;
  return `<span class="c-oth">${app.fx.targets.filter((t) => t.id !== skip).map((t) => `<span class="c-o" title="${esc(`${t.name}: ${x.row.v[t.id].full}`)}">${vsq(x.row.v[t.id])}<span>${esc(t.name)}</span><span class="c-ow">${esc(x.row.v[t.id].kind === 'fill' ? 'Fills' : x.row.v[t.id].kind === 'meets' ? 'Meets' : x.row.v[t.id].short)}</span></span>`).join('')}</span>`;
}

function rowHtml(app, x, i) {
  const { r } = x.row;
  const open = app.open === r.id;
  const body = `<span class="c-rel"><span class="c-name mono">${rname(r.name)}</span><span class="t3 c-facts">${grabbedChip(app, r)}${covTag(x.row)}${edTag(app, r)}<span>${esc(ixName(app, r))}</span><span class="s-sep">·</span>${stats(r)}</span>${others(app, x)}</span>`;
  const exp = open ? `<div class="c-x">${app.phone ? `<p class="t3 pc-facts">${facts(r)}</p>` : ''}${grabBlock(app, x.row, { touch: app.phone })}<div class="c-x-grid"><div><h3 class="x-h">Verdict per Target</h3>${verdictBlocks(app, x.row)}</div><div>${breakdown(app, x.row)}</div></div></div>` : '';
  return `<div class="c-item${open ? ' open' : ''}${app.row === r.id ? ' sel' : ''}${x.row.allHard ? ' dim' : ''}"><div class="c-row" data-toggle="${r.id}" aria-expanded="${open}" tabindex="0">${app.phone ? '' : check(app, r)}<span class="c-rank${i === 0 ? ' first' : ''}">${i + 1}</span><span class="c-big">${bigVerdict(app, x)}</span>${body}</div>${exp}</div>`;
}

function railItem(app, c) {
  return `<button class="c-cell${app.cell === c.key ? ' on' : ''}" data-cell-pick="${c.key}" aria-pressed="${app.cell === c.key}">${sq(c.state)}<span class="c-cell-b"><b>${esc(cellName(app, c.key))}</b><span class="t3">${esc(cellStateLine(app.fx, c))}</span></span><span class="c-n num${c.n ? '' : ' t3'}">${c.n}</span></button>`;
}

function rail(app, cells) {
  const want = cells.filter((c) => c.state !== 'met').sort((a, b) => STATES[a.state].rank - STATES[b.state].rank);
  const met = cells.filter((c) => c.state === 'met');
  const allN = app.rows().filter((x) => x.wanted).length;
  return `<nav class="c-rail" aria-label="Cells" data-keep="rail">
    <button class="c-cell c-all${app.cell === 'all' ? ' on' : ''}" data-cell-pick="all" aria-pressed="${app.cell === 'all'}">${I('GridView')}<span class="c-cell-b"><b>All cells</b><span class="t3">${cells.length} in scope</span></span><span class="c-n num">${allN}</span></button>
    ${want.length ? `<p class="c-g">Wanted · ${want.length}</p>${want.map((c) => railItem(app, c)).join('')}` : ''}
    ${met.length ? `<button class="c-g c-fold" data-fold>${I(app.foldOpen ? 'ArrowDown01' : 'ArrowRight01', 'ic-s')}Met, cutoff reached · ${met.length}</button>${app.foldOpen ? met.map((c) => railItem(app, c)).join('') : ''}` : ''}
  </nav>`;
}

function cellHead(app, cells) {
  if (app.cell === 'all') return `<div class="c-head"><span class="c-head-t">${I('GridView')}<b>All cells</b></span><span class="t3">Ranked by how many cells each Release fills</span></div>`;
  const c = cells.find((x) => x.key === app.cell);
  return `<div class="c-head"><span class="c-head-t">${sq(c.state, 'sq-lg')}<b>${esc(cellName(app, c.key))}</b></span><span class="t3">${esc(cellStateLine(app.fx, c))}${c.file ? `<span class="s-sep">·</span><span class="mono">${esc(c.file)}</span>` : ''}</span></div>`;
}

function filter(app) {
  const all = app.cell === 'all' ? app.rows().length : app.rows().length;
  const w = app.cell === 'all' ? app.rows().filter((x) => x.wanted).length : rankForCell(app.fx, app.rows(), app.cell).filter((x) => fillsCell(x.cv)).length;
  return `<div class="seg-c${app.phone ? ' seg-full' : ''}" role="radiogroup" aria-label="Filter"><button class="seg-o" role="radio" aria-checked="${app.filter === 'wanted'}" data-filter="wanted">Would fill<span class="seg-n num">${w}</span></button><button class="seg-o" role="radio" aria-checked="${app.filter === 'all'}" data-filter="all">All<span class="seg-n num">${all}</span></button></div>`;
}

const empty = (app) => `<div class="ms-empty">${app.arrived.size < app.indexers.length ? '<span class="spin"></span>Waiting for Indexers' : app.filter === 'wanted' ? 'Nothing here would fill this cell. <button class="lnk" data-filter="all"><b>Show all Releases</b></button>' : 'No Releases came back.'}</div>`;

export function render(app) {
  const cells = cellsWithCounts(app);
  if (app.cell !== 'all' && !cells.some((c) => c.key === app.cell)) app.cell = 'all';
  const list = ranked(app);
  const t = `<div class="ms-t"><h2>Manual search</h2><p class="t3">${esc(app.fx.work.title)}<span class="s-sep">·</span>Profile ${esc(app.fx.profile.name)}</p></div>`;
  if (app.phone) {
    const want = cells.filter((c) => c.state !== 'met').sort((a, b) => STATES[a.state].rank - STATES[b.state].rank);
    const chips = [{ key: 'all' }, ...want, ...cells.filter((c) => c.state === 'met')].map((c) => (c.key === 'all'
      ? `<button class="c-chip${app.cell === 'all' ? ' on' : ''}" data-cell-pick="all">${I('GridView')}All cells<span class="num">${app.rows().filter((x) => x.wanted).length}</span></button>`
      : `<button class="c-chip${app.cell === c.key ? ' on' : ''}" data-cell-pick="${c.key}">${sq(c.state)}${esc(cellName(app, c.key))}<span class="num">${c.n}</span></button>`)).join('');
    return `<div class="ms-wrap"><div class="ms-scrim" data-close-ms></div><div class="psheet" role="dialog" aria-modal="true" aria-label="Manual search">
      <span class="grab" aria-hidden="true"></span>
      <header class="ms-h">${t}<button class="ib tap" data-close-ms aria-label="Close">${I('Cancel01')}</button></header>
      <div class="ms-scope-p">${scopeSeg(app, 'seg-scroll')}</div>
      <div class="c-chips" data-keep="chips">${chips}</div>
      ${cellHead(app, cells)}
      <div class="ms-tools-p">${filter(app)}</div>
      <div class="ms-tools-p2">${searchStatus(app)}</div>
      <div class="ms-list-p c-list" data-keep="c">${list.map((x, i) => rowHtml(app, x, i)).join('') || empty(app)}${ixFooter(app)}</div>
      ${bulkBar(app)}
    </div></div>`;
  }
  return `<div class="ms-wrap"><div class="ms-scrim" data-close-ms></div><div class="dlg dlg-c" role="dialog" aria-modal="true" aria-label="Manual search">
    <header class="ms-h">${t}${scopeSeg(app)}<button class="ib" data-close-ms aria-label="Close">${I('Cancel01')}</button></header>
    <div class="c-body">${rail(app, cells)}
      <div class="c-main">${cellHead(app, cells)}
        <div class="ms-tools">${filter(app)}${searchStatus(app)}</div>
        <div class="ms-list c-list" data-keep="c">${list.map((x, i) => rowHtml(app, x, i)).join('') || empty(app)}</div>
        ${bulkBar(app)}
      </div>
    </div>
    ${ixFooter(app)}
  </div></div>`;
}

export function init(app) {
  const p = new URLSearchParams(location.search);
  app.cell = p.get('cell') || (app.sc.kind === 'cell' ? app.sc.cell : 'all');
  app.variantClick = (e, q) => {
    let el;
    if ((el = q('[data-cell-pick]'))) { app.cell = el.dataset.cellPick; app.open = null; app.render(); return true; }
    if ((el = q('[data-fold]'))) { app.foldOpen = !app.foldOpen; app.render(); return true; }
    if (q('.ck, button, a')) return false;
    if ((el = q('[data-toggle]'))) { const id = el.dataset.toggle; app.open = app.open === id ? null : id; app.row = id; app.render(); return true; }
    return false;
  };
  document.addEventListener('keydown', (e) => {
    const el = e.target.closest && e.target.closest('[data-toggle]');
    if (el && (e.key === 'Enter' || e.key === ' ')) { e.preventDefault(); el.click(); }
  });
}

export function onScope(app) { app.cell = app.sc.kind === 'cell' ? app.sc.cell : 'all'; }

export function after(app, host) {
  const on = host.querySelector('.c-chip.on');
  if (on && !app.chipScrolled) { on.scrollIntoView({ inline: 'center', block: 'nearest' }); app.chipScrolled = true; }
  if (app.scrolled || !app.open) return;
  const el = host.querySelector('.c-item.open');
  if (el) { el.scrollIntoView({ block: 'start' }); app.scrolled = true; }
}

