// PROTOTYPE · A · Verdict strip. A dense, row-first table in a 940px Dialog; each row ends in one compact verdict chip per Target and expands in place. Phone: a bottom sheet of stacked cards. Throwaway.

import { I, esc, rname, vstrip, facts, stats, edTag, covTag, ixName, grabbedChip, scopeSeg, filterSeg, sortBtn, searchStatus, ixFooter, verdictBlocks, breakdown, grabBlock, bulkBar, check } from './common.js';
import { fmt } from './model.js';

const COLS = [['size', 'Size'], ['seeders', 'Peers'], ['age', 'Age']];

function head(app) {
  const s = (id, label, cls = '') => `<button class="a-hc ${cls}${app.sort === id ? ' on' : ''}" data-sort="${id}">${label}${app.sort === id ? I('ArrowDown01', 'ic-s') : ''}</button>`;
  return `<div class="a-row a-head" role="row"><span></span>${s('best', 'Release')}<span class="a-hc">Coverage</span>${s('indexer', 'Indexer')}${COLS.map(([id, l]) => s(id, l, 'r')).join('')}<span class="a-strip a-strip-h">${app.fx.targets.map((t) => `<span class="a-th" title="${esc(t.name)}">${esc(t.name)}</span>`).join('')}</span></div>`;
}

function expansion(app, row) {
  return `<div class="a-x" role="region" aria-label="Details">
    ${grabBlock(app, row)}
    <div class="a-x-grid"><div class="a-x-v"><h3 class="x-h">Verdict per Target</h3>${verdictBlocks(app, row)}</div><div>${breakdown(app, row)}</div></div>
  </div>`;
}

function deskRow(app, row) {
  const { r, v } = row;
  const open = app.open === r.id;
  return `<div class="a-item${open ? ' open' : ''}${app.row === r.id ? ' sel' : ''}${row.allHard ? ' dim' : ''}">
    <div class="a-row" role="row" data-toggle="${r.id}" aria-expanded="${open}" tabindex="0">
      ${check(app, r)}
      <span class="a-rel"><span class="a-name mono">${rname(r.name)}</span><span class="a-facts t3">${grabbedChip(app, r)}${facts(r)}</span></span>
      <span class="a-cov">${covTag(row)}${edTag(app, r)}</span>
      <span class="a-ix ell">${esc(ixName(app, r))}${r.freeleech ? `<span class="fl" title="Freeleech">${I('Gift', 'ic-s')}</span>` : ''}</span>
      <span class="num r">${fmt.size(r.size)}</span>
      <span class="num r">${r.seeders}<span class="t3">/${r.leechers}</span></span>
      <span class="num r">${fmt.age(r.age)}</span>
      <span class="a-strip">${app.fx.targets.map((t) => vstrip(v[t.id])).join('')}</span>
    </div>
    ${open ? expansion(app, row) : ''}
  </div>`;
}

function phoneCard(app, row) {
  const { r, v } = row;
  const open = app.open === r.id;
  return `<div class="pc${open ? ' open' : ''}${row.allHard ? ' dim' : ''}">
    <div class="pc-top" data-toggle="${r.id}" aria-expanded="${open}" tabindex="0">
      ${check(app, r)}
      <div class="pc-b"><span class="pc-name mono">${rname(r.name)}</span>
        <span class="pc-meta t3">${covTag(row)}${edTag(app, r)}<span>${esc(ixName(app, r))}</span></span>
        <span class="pc-meta t3">${stats(r)}</span>
        <span class="pc-strip">${app.fx.targets.map((t) => `<span class="pc-v"><span class="pc-t">${esc(t.name)}</span>${vstrip(v[t.id])}</span>`).join('')}</span>
        ${grabbedChip(app, r)}
      </div>
    </div>
    ${open ? `<div class="pc-x"><p class="t3 pc-facts">${facts(r)}</p>${grabBlock(app, row, { touch: true })}<h3 class="x-h">Verdict per Target</h3>${verdictBlocks(app, row)}${breakdown(app, row)}</div>` : ''}
  </div>`;
}

const empty = (app) => `<div class="ms-empty">${app.arrived.size < app.indexers.length ? '<span class="spin"></span>Waiting for Indexers' : app.filter === 'wanted' ? 'Nothing here would fill a wanted cell. <button class="lnk" data-filter="all"><b>Show all Releases</b></button>' : 'No Releases came back.'}</div>`;

export function render(app) {
  const rows = app.visible();
  const t = `<div class="ms-t"><h2>Manual search</h2><p class="t3">${esc(app.fx.work.title)}<span class="s-sep">·</span>Profile ${esc(app.fx.profile.name)}</p></div>`;
  if (app.phone) {
    return `<div class="ms-wrap"><div class="ms-scrim" data-close-ms></div><div class="psheet" role="dialog" aria-modal="true" aria-label="Manual search">
      <span class="grab" aria-hidden="true"></span>
      <header class="ms-h">${t}<button class="ib tap" data-close-ms aria-label="Close">${I('Cancel01')}</button></header>
      <div class="ms-scope-p">${scopeSeg(app, 'seg-scroll')}</div>
      <div class="ms-tools-p">${filterSeg(app, 'seg-full')}</div>
      <div class="ms-tools-p2">${searchStatus(app)}${sortBtn(app)}</div>
      <div class="ms-list-p" data-keep="a">${rows.map((x) => phoneCard(app, x)).join('') || empty(app)}${ixFooter(app)}</div>
      ${bulkBar(app)}
    </div></div>`;
  }
  return `<div class="ms-wrap"><div class="ms-scrim" data-close-ms></div><div class="dlg" role="dialog" aria-modal="true" aria-label="Manual search">
    <header class="ms-h">${t}${scopeSeg(app)}<button class="ib" data-close-ms aria-label="Close">${I('Cancel01')}</button></header>
    <div class="ms-tools">${filterSeg(app)}${searchStatus(app)}<span class="grow"></span>${sortBtn(app)}</div>
    <div class="ms-list a-list" data-keep="a" role="table">${head(app)}${rows.map((x) => deskRow(app, x)).join('') || empty(app)}</div>
    ${bulkBar(app)}
    ${ixFooter(app)}
  </div></div>`;
}

export function init(app) {
  app.variantClick = (e, q) => {
    let el;
    if ((el = q('[data-sort]'))) { app.sort = el.dataset.sort; app.render(); return true; }
    if (q('.ck, button, a')) return false;
    if ((el = q('[data-toggle]'))) { const id = el.dataset.toggle; app.open = app.open === id ? null : id; app.row = id; app.render(); return true; }
    return false;
  };
  document.addEventListener('keydown', (e) => {
    const el = e.target.closest && e.target.closest('[data-toggle]');
    if (el && (e.key === 'Enter' || e.key === ' ')) { e.preventDefault(); el.click(); }
  });
}

export function after(app, host) {
  if (app.scrolled || !app.open) return;
  const el = host.querySelector('.a-item.open, .pc.open');
  if (el) { el.scrollIntoView({ block: 'start' }); app.scrolled = true; }
}
