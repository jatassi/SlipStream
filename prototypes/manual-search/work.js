// PROTOTYPE · a simplified Work page behind the sheet, so each Manual search entry point (Work, level, Item, cell) is visible. Throwaway.

import { I, esc, art } from './shell.js';
import { sq, stateBadge } from './common.js';
import { cellKey, fmt, STATES } from './model.js';

const search = (attrs, label, cls = 'btn sm ghost') => `<button class="${cls}" ${attrs}>${I('Search01')}${label}</button>`;

function tally(fx, cells) {
  const n = {};
  for (const c of cells) n[c.state] = (n[c.state] || 0) + 1;
  return Object.keys(STATES).filter((s) => n[s]).map((s) => `${n[s]} ${STATES[s].word}`).join(' · ');
}

function stateBar(cells) {
  const order = ['met', 'upgrade', 'missing', 'failed'];
  return `<span class="sbar" aria-hidden="true">${order.map((s) => { const k = cells.filter((c) => c.state === s).length; return k ? `<span class="sbar-${s}" style="flex:${k}"></span>` : ''; }).join('')}</span>`;
}

function movieMatrix(fx) {
  return `<table class="mx"><thead><tr><th class="mx-corner">Edition ${I('ArrowRight01')} Target</th>${fx.targets.map((t) => `<th><div class="mx-th"><b>${esc(t.name)}</b><span class="t3">${esc(t.sourcing)}</span></div></th>`).join('')}</tr></thead>
  <tbody>${fx.editions.map((ed) => `<tr><td class="mx-ed"><b>${esc(ed)}</b><span class="t3">${ed === 'Standard' ? 'Theatrical release' : 'Listed by TMDB, 3h 18m'}</span></td>${fx.targets.map((t) => {
    const key = cellKey('movie', ed, t.id);
    const c = fx.cells[key];
    const facts = c.file ? `<p class="mx-file mono">${esc(c.file)}</p><p class="t3 mx-sub">Score <span class="num">${fmt.score(c.score)}</span>, upgrading until <span class="num">${fmt.score(t.upgradeUntil)}</span></p>` : '<p class="t3 mx-sub">Nothing grabbed yet</p>';
    return `<td class="cell"><div class="cell-h">${stateBadge(c)}</div>${facts}<div class="cell-foot">${search(`data-ms="cell" data-cell="${key}"`, 'Manual search')}</div></td>`;
  }).join('')}</tr>`).join('')}</tbody></table>`;
}

function seriesMatrix(fx) {
  return fx.levels.map((l) => {
    const items = fx.items.filter((i) => i.level === l.id);
    const cells = items.flatMap((i) => fx.targets.map((t) => ({ ...fx.cells[cellKey(i.id, 'Standard', t.id)] })));
    const open = cells.some((c) => c.state !== 'met');
    const rows = open ? `<div class="lv-rows"><div class="lv-head"><span></span><span></span>${fx.targets.map((t) => `<span class="lv-th">${esc(t.name)}</span>`).join('')}<span></span></div>${items.map((it) => `<div class="lv-row"><span class="num t3">${it.short.slice(3)}</span><span class="ell">${esc(it.label)}</span>${fx.targets.map((t) => { const key = cellKey(it.id, 'Standard', t.id); const c = fx.cells[key]; return `<button class="sqb" data-ms="cell" data-cell="${key}" title="${esc(`${it.short} · ${t.name} · ${STATES[c.state].word}. Manual search this cell`)}">${sq(c.state)}</button>`; }).join('')}<button class="ib sm" data-ms="item" data-item="${it.id}" aria-label="Manual search ${it.short}" title="Manual search ${it.short}">${I('Search01')}</button></div>`).join('')}</div>` : '';
    return `<section class="lv${open ? ' open' : ''}"><div class="lv-h">${I(open ? 'ArrowDown01' : 'ArrowRight01', 'lv-chev')}<b>${esc(l.label)}</b>${stateBar(cells)}<span class="t3 lv-tally">${tally(fx, cells)}</span>${search(`data-ms="level" data-level="${l.id}"`, 'Manual search')}</div>${rows}</section>`;
  }).join('');
}

function legend() {
  return `<span class="mx-legend t3">${['met', 'upgrade', 'missing', 'failed'].map((s) => `<span>${sq(s)}${STATES[s].word}</span>`).join('')}</span>`;
}

export function workDesk(fx) {
  const w = fx.work;
  const cells = Object.values(fx.cells);
  const met = cells.filter((c) => c.state === 'met').length;
  return `<section class="wk-head"><div class="wk-bd" aria-hidden="true"><img src="${w.art.backdrop}" alt="" onerror="this.remove()"></div>
      ${art(w, 'poster-lg')}
      <div class="wk-id"><span class="wk-logo-t">${esc(w.title)}</span><p class="meta t2">${w.meta.map(esc).join('<span class="s-sep">·</span>')}<span class="s-sep">·</span>Profile ${esc(fx.profile.name)}</p>
        <p class="wk-state">${met ? `<span class="pill-met">${I('Tick02')}${met} of ${cells.length} cells Met</span>` : ''}<span class="t2">${tally(fx, cells)}</span></p>
        <p class="overview t2">${esc(w.overview)}</p>
        <div class="wk-act"><button class="btn pri" data-toast="Search now queues an automatic search">${I('Search01')}Search now</button>${search('data-ms="work"', 'Manual search', 'btn')}</div></div></section>
    <section><div class="sec-h"><h2>${fx.work.kind === 'movie' ? 'Editions and Targets' : 'Seasons'}</h2>${legend()}</div>${w.kind === 'movie' ? movieMatrix(fx) : seriesMatrix(fx)}</section>`;
}

export function workPhone(fx) {
  const w = fx.work;
  const list = w.kind === 'movie'
    ? fx.editions.map((ed) => `<div class="mxi"><div class="mxi-h"><b>${esc(ed)}</b></div>${fx.targets.map((t) => { const key = cellKey('movie', ed, t.id); const c = fx.cells[key]; return `<button class="gl-cell" data-ms="cell" data-cell="${key}">${sq(c.state)}<span>${esc(t.name)}</span><span class="t3">${STATES[c.state].word}</span>${I('Search01', 'gl-go')}</button>`; }).join('')}</div>`).join('')
    : fx.levels.map((l) => { const items = fx.items.filter((i) => i.level === l.id); const cells = items.flatMap((i) => fx.targets.map((t) => fx.cells[cellKey(i.id, 'Standard', t.id)])); return `<button class="gl-cell" data-ms="level" data-level="${l.id}"><b>${esc(l.label)}</b>${stateBar(cells)}<span class="t3">${tally(fx, cells)}</span>${I('Search01', 'gl-go')}</button>`; }).join('');
  return `<div class="pw-top"><div class="pw-bd" aria-hidden="true"><img src="${w.art.backdrop}" alt="" onerror="this.remove()"></div>
      <div class="pw-id">${art(w, 'poster-md')}<div><span class="wk-logo-t">${esc(w.title)}</span><p class="meta t2">${w.meta.slice(0, 3).map(esc).join('<span class="s-sep">·</span>')}</p><p class="t3">Profile ${esc(fx.profile.name)}</p></div></div>
      <div class="pw-act"><button class="btn pri" data-toast="Search now queues an automatic search">${I('Search01')}Search now</button><button class="btn" data-ms="work">${I('Search01')}Manual search</button></div></div>
    <section class="psec"><div class="psec-h"><h2>${w.kind === 'movie' ? 'Editions' : 'Seasons'}</h2></div><div class="glist">${list}</div></section>`;
}
