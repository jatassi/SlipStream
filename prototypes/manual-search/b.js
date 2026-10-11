// PROTOTYPE · B · Target columns + Inspector. A full-width large sheet; each Target in the Profile is a column whose header shows its cells' state. Selecting a row opens the Inspector, which pushes the table. Phone: a segmented control picks one Target; tapping a row pushes the detail page. Throwaway.

import { I, esc, rname, vchip, sq, facts, stats, edTag, covTag, ixName, grabbedChip, scopeSeg, filterSeg, sortBtn, searchStatus, ixFooter, verdictBlocks, breakdown, grabBlock, bulkBar, check } from './common.js';
import { fmt, scopeCells, scopeTargets, cellStateLine, STATES, fills, rankForTarget, SORTS } from './model.js';

function targetHead(app, t) {
  const inScope = scopeTargets(app.fx, app.sc).some((x) => x.id === t.id);
  const cells = scopeCells(app.fx, app.sc).filter((c) => c.target.id === t.id);
  let line;
  if (!inScope) line = 'Outside this scope';
  else if (cells.length === 1) line = cellStateLine(app.fx, cells[0]);
  else {
    const n = {};
    cells.forEach((c) => { n[c.state] = (n[c.state] || 0) + 1; });
    line = Object.keys(STATES).filter((s) => n[s]).map((s) => `${n[s]} ${STATES[s].word}`).join(' · ');
  }
  const squares = cells.length > 1 ? `<span class="b-sqs">${cells.slice(0, 20).map((c) => sq(c.state)).join('')}</span>` : cells.length ? sq(cells[0].state) : '';
  return `<span class="b-th${inScope ? '' : ' out'}"><b>${esc(t.name)}</b><span class="b-th-s">${squares}</span><span class="t3 b-th-l" title="${esc(line)}">${esc(line)}</span></span>`;
}

function deskRow(app, row) {
  const { r, v } = row;
  return `<div class="b-row${app.row === r.id ? ' sel' : ''}${row.allHard ? ' dim' : ''}" data-pick="${r.id}" tabindex="0" role="row" aria-selected="${app.row === r.id}">
    ${check(app, r)}
    <span class="b-rel"><span class="b-name mono ell">${esc(r.name)}</span><span class="b-facts t3 ell">${grabbedChip(app, r)}${facts(r)}</span></span>
    <span class="b-cov">${covTag(row)}${edTag(app, r)}</span>
    <span class="b-ix ell">${esc(ixName(app, r))}${r.freeleech ? `<span class="fl" title="Freeleech">${I('Gift', 'ic-s')}</span>` : ''}</span>
    <span class="num r b-size">${fmt.size(r.size)}</span>
    <span class="num r b-peers">${r.seeders}<span class="t3">/${r.leechers}</span></span>
    <span class="num r b-age">${fmt.age(r.age)}</span>
    ${app.fx.targets.map((t) => `<span class="b-cell">${vchip(v[t.id], 'xs', v[t.id].kind === 'meets' ? 'Meets, not better' : undefined)}${v[t.id].kind === 'hard' ? '' : `<span class="num t3">${fmt.score(v[t.id].score)}</span>`}</span>`).join('')}
  </div>`;
}

function inspector(app, row) {
  const { r } = row;
  return `<aside class="insp" aria-label="Release detail">
    <div class="insp-h"><div class="insp-t"><span class="t3">Release</span><button class="ib sm" data-unpick aria-label="Close the Inspector">${I('Cancel01')}</button></div>
      <p class="insp-name mono">${rname(r.name)}</p>
      <p class="t3 insp-f">${facts(r)}</p>
      <p class="t3 insp-f">${covTag(row)}${edTag(app, r)}<span>${esc(ixName(app, r))}</span><span class="s-sep">·</span>${stats(r)}</p>
    </div>
    <div class="insp-b" data-keep="insp">
      <section class="well">${grabBlock(app, row)}</section>
      <section class="well"><h3 class="x-h">Verdict per Target</h3>${verdictBlocks(app, row)}</section>
      <section class="well">${breakdown(app, row)}</section>
    </div>
  </aside>`;
}

const empty = (app) => `<div class="ms-empty">${app.arrived.size < app.indexers.length ? '<span class="spin"></span>Waiting for Indexers' : app.filter === 'wanted' ? 'Nothing here would fill a wanted cell. <button class="lnk" data-filter="all"><b>Show all Releases</b></button>' : 'No Releases came back.'}</div>`;

function desk(app) {
  const rows = app.visible();
  const picked = app.row && app.rowById(app.row);
  const cols = `--tcols:${app.fx.targets.length}`;
  return `<div class="ms-wrap"><div class="ms-scrim" data-close-ms></div><div class="bpage${picked ? ' has-insp' : ''}" role="dialog" aria-modal="true" aria-label="Manual search">
    <header class="ms-h"><div class="ms-t"><h2>Manual search</h2><p class="t3">${esc(app.fx.work.title)}<span class="s-sep">·</span>Profile ${esc(app.fx.profile.name)}</p></div>${scopeSeg(app)}<button class="ib" data-close-ms aria-label="Close">${I('Cancel01')}</button></header>
    <div class="ms-tools">${filterSeg(app)}${searchStatus(app)}<span class="grow"></span>${sortBtn(app)}</div>
    <div class="b-body">
      <div class="b-main"><div class="b-tablewrap"><div class="b-table" style="${cols}">
        <div class="b-row b-head" role="row"><span></span><span>Release</span><span>Coverage</span><span class="b-ix">Indexer</span><span class="r b-size">Size</span><span class="r b-peers">Peers</span><span class="r b-age">Age</span>${app.fx.targets.map((t) => targetHead(app, t)).join('')}</div>
        <div class="b-rows" data-keep="b">${rows.map((x) => deskRow(app, x)).join('') || empty(app)}</div>
      </div></div>
      ${bulkBar(app)}
      ${ixFooter(app)}</div>
      ${picked ? inspector(app, picked) : ''}
    </div>
  </div></div>`;
}

function phone(app) {
  const t = app.fx.targets.find((x) => x.id === app.tab);
  if (app.page) {
    const row = app.rowById(app.page);
    const { r } = row;
    return `<div class="ppage" role="dialog" aria-label="Release">
      <div class="ph-nav"><button class="back" data-back>${I('ArrowLeft01')}Results</button></div>
      <div class="ppage-b" data-keep="bp">
        <p class="pd-name mono">${rname(r.name)}</p>
        <p class="t3 pd-f">${facts(r)}</p>
        <p class="t3 pd-f">${covTag(row)}${edTag(app, r)}<span>${esc(ixName(app, r))}</span></p>
        <p class="t3 pd-f">${stats(r)}</p>
        ${grabBlock(app, row, { touch: true })}
        <h3 class="x-h">Verdict per Target</h3>${verdictBlocks(app, row)}
        ${breakdown(app, row)}
      </div></div>`;
  }
  const all = app.rows();
  const fillN = (tid) => all.filter((x) => fills(x.v[tid])).length;
  const sorted = app.sort === 'best' ? rankForTarget(all, t.id) : [...all].sort(SORTS[app.sort].cmp);
  const rows = app.filter === 'all' ? sorted : sorted.filter((x) => fills(x.v[t.id]));
  const cells = scopeCells(app.fx, app.sc).filter((c) => c.target.id === t.id);
  const seg = `<div class="seg-c seg-full" role="radiogroup" aria-label="Filter"><button class="seg-o" role="radio" aria-checked="${app.filter === 'wanted'}" data-filter="wanted">Would fill<span class="seg-n num">${fillN(t.id)}</span></button><button class="seg-o" role="radio" aria-checked="${app.filter === 'all'}" data-filter="all">All<span class="seg-n num">${all.length}</span></button></div>`;
  return `<div class="ppage" role="dialog" aria-label="Manual search">
    <div class="ph-nav"><button class="back" data-close-ms>${I('ArrowLeft01')}${esc(app.fx.work.title)}</button><span class="grow"></span>${sortBtn(app)}</div>
    <div class="ppage-b" data-keep="bl">
      <div class="ph-head"><h1>Manual search</h1></div>
      <div class="ms-scope-p">${scopeSeg(app, 'seg-scroll')}</div>
      <div class="ms-tools-p"><div class="seg-c seg-full" role="radiogroup" aria-label="Target">${app.fx.targets.map((x) => `<button class="seg-o" role="radio" aria-checked="${x.id === t.id}" data-tab="${x.id}">${esc(x.name)}<span class="seg-n num">${fillN(x.id)}</span></button>`).join('')}</div></div>
      <p class="b-ptgt t3">${cells.length === 1 ? sq(cells[0].state) : ''}<span>${esc(t.name)} · ${cells.length === 1 ? esc(cellStateLine(app.fx, cells[0])) : `${cells.length} cells in scope`}</span></p>
      <div class="ms-tools-p">${seg}</div>
      <div class="ms-tools-p2">${searchStatus(app)}</div>
      <div class="glist b-plist">${rows.map((x) => `<button class="b-prow${x.allHard ? ' dim' : ''}" data-push="${x.r.id}">
          <span class="b-prow-b"><span class="pc-name mono">${rname(x.r.name)}</span><span class="pc-meta t3">${covTag(x)}${edTag(app, x.r)}<span>${esc(ixName(app, x.r))}</span><span class="s-sep">·</span><span class="num">${fmt.size(x.r.size)}</span></span><span class="b-prow-v">${vchip(x.v[t.id], 'sm')}${x.v[t.id].kind === 'hard' ? '' : `<span class="num t3">${fmt.score(x.v[t.id].score)}</span>`}${grabbedChip(app, x.r)}</span></span>${I('ArrowRight01', 'b-chev')}</button>`).join('') || empty(app)}</div>
      ${ixFooter(app)}
    </div></div>`;
}

export const render = (app) => (app.phone ? phone(app) : desk(app));

export function init(app) {
  app.tab = app.fx.targets[0].id;
  if (app.phone && app.open) app.page = app.open;
  app.variantClick = (e, q) => {
    let el;
    if ((el = q('[data-unpick]'))) { app.row = null; app.render(); return true; }
    if ((el = q('[data-tab]'))) { app.tab = el.dataset.tab; app.render(); return true; }
    if ((el = q('[data-push]'))) { app.page = el.dataset.push; app.render(); return true; }
    if ((el = q('[data-back]'))) { app.page = null; app.render(); return true; }
    if (q('.ck, button, a')) return false;
    if ((el = q('[data-pick]'))) { app.row = el.dataset.pick; app.render(); return true; }
    return false;
  };
  document.addEventListener('keydown', (e) => {
    if (app.phone || !app.row || (e.key !== 'ArrowDown' && e.key !== 'ArrowUp')) return;
    if (e.target.closest && e.target.closest('.menu')) return;
    const ids = app.visible().map((x) => x.r.id);
    const i = ids.indexOf(app.row);
    if (i < 0) return;
    e.preventDefault();
    app.row = ids[Math.max(0, Math.min(ids.length - 1, i + (e.key === 'ArrowDown' ? 1 : -1)))];
    app.render();
    const el = document.querySelector(`[data-pick="${app.row}"]`);
    if (el) { el.scrollIntoView({ block: 'nearest' }); el.focus({ preventScroll: true }); }
  });
}

export function escape(app) {
  if (app.page) { app.page = null; app.render(); return true; }
  if (app.row && !app.phone) { app.row = null; app.render(); return true; }
  return false;
}

export function after(app, host) {
  if (app.scrolled || !app.row) return;
  const el = host.querySelector('.b-row.sel');
  if (el) { el.scrollIntoView({ block: 'nearest' }); app.scrolled = true; }
}

