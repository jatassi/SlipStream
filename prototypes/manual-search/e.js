// PROTOTYPE · E · Shelves by outcome. One list, each Release once, shelved by what it gets you: something new, something better, then the folded "wouldn't grab". A chip row narrows to one Target or one episode in plain words. Throwaway.

import { I, esc, header, frame, facts, outLines, details, hiddenBlock, judgeAll, wouldGrab, needKeys, wire, escapeShared, searching, grabBtn, recTag } from './plain.js';
import { rankCmp, parseCell, itemById, normScope, outcome, choiceCmp } from './model.js';
import { openMenu } from './shell.js';

function focusKeys(app) {
  return needKeys(app.fx, app.sc).filter((k) => { const c = parseCell(k); return (app.ft === 'all' || c.target === app.ft) && (app.fi === 'all' || c.item === app.fi); });
}

function shelve(app) {
  const all = needKeys(app.fx, app.sc);
  const focus = focusKeys(app);
  const rank = new Map([...app.rows()].sort(rankCmp).map((row, i) => [row.r.id, i]));
  const shelves = { new: [], better: [], hidden: [] };
  const t = app.ft === 'all' ? null : app.fx.targets.find((x) => x.id === app.ft);
  for (const row of app.rows()) {
    const full = outcome(app.fx, row, all, app.sc);
    if (wouldGrab(full)) {
      const o = outcome(app.fx, row, focus, app.sc);
      if (wouldGrab(o)) shelves[o.kind].push({ row, out: o });
    } else if ((!t || row.r.res === t.res) && (app.fi === 'all' || row.r.coverage.includes(app.fi))) {
      shelves.hidden.push({ row, out: outcome(app.fx, row, focus, app.sc) });
    }
  }
  const cmp = choiceCmp(app.fx, rank);
  shelves.new.sort(cmp);
  shelves.better.sort(cmp);
  return shelves;
}

function countFor(app, patch) {
  const keep = { ft: app.ft, fi: app.fi };
  Object.assign(app, patch);
  const s = shelve(app);
  Object.assign(app, keep);
  return s.new.length + s.better.length;
}

function chips(app) {
  const ts = [{ id: 'all', name: 'All' }, ...app.fx.targets];
  const seg = `<div class="seg-c r2-seg" role="radiogroup" aria-label="For">${ts.map((t) => `<button class="seg-o" role="radio" aria-checked="${app.ft === t.id}" data-ft="${t.id}">${esc(t.name)}<span class="seg-n num">${countFor(app, { ft: t.id })}</span></button>`).join('')}</div>`;
  const ep = app.fx.work.kind === 'movie' || app.sc.kind === 'item' ? '' : `<button class="btn sm r2-epb" data-ep-menu aria-haspopup="menu">${esc(app.fi === 'all' ? 'All episodes' : `Episode ${itemById(app.fx, app.fi).ep}`)}${I('ArrowDown01', 'ic-s')}</button>`;
  return `<div class="r2-chips"><span class="r2-for t3">For</span>${seg}${ep}</div>`;
}

function epMenu(app, anchor) {
  const ids = [...new Set(needKeys(app.fx, app.sc).map((k) => parseCell(k).item))].map((id) => itemById(app.fx, id)).sort((a, b) => a.season - b.season || a.ep - b.ep);
  openMenu(anchor, 'Episode', [{ id: 'all', label: 'All episodes' }, ...ids.map((it) => ({ id: it.id, label: `Episode ${it.ep}`, sub: it.label }))].map((x) => ({ label: x.label, sub: x.sub, right: x.id === 'all' ? '' : `${countFor(app, { fi: x.id })}`, icon: app.fi === x.id ? 'Tick02' : null, run: () => { app.fi = x.id; app.render(); } })));
}

function row(app, x, opts) {
  const { r } = x.row;
  const open = app.det === r.id;
  return `<div class="r2-row${open ? ' open' : ''}${opts.primary ? ' r2-row-pri' : ''}"><div class="r2-row-m"><button class="r2-row-b" data-det="${r.id}" aria-expanded="${open}">${facts(app, r, { tag: opts.rec ? recTag() : '', chev: open ? 'ArrowUp01' : 'ArrowDown01' })}${outLines(x.out)}</button><span class="r2-row-a">${grabBtn(app, x.row, { primary: opts.primary, sm: !opts.primary })}</span></div>${open ? details(app, x.row) : ''}</div>`;
}

function shelf(app, title, list, primary) {
  if (!list.length) return '';
  return `<section class="r2-shelf"><h3 class="r2-sh">${esc(title)}<span class="num-t t3">${list.length}</span></h3><div class="r2-list">${list.map((x, i) => row(app, x, { rec: i === 0, primary: primary && i === 0 })).join('')}</div></section>`;
}

export function render(app) {
  const s = shelve(app);
  const waiting = searching(app) && !app.rows().length;
  const none = !waiting && !s.new.length && !s.better.length;
  const body = `${chips(app)}
    ${waiting ? '<p class="r2-empty"><span class="spin"></span>Looking for releases…</p>' : ''}
    ${none ? `<p class="r2-empty">${needKeys(app.fx, app.sc).length ? 'Nothing found yet that SlipStream would grab here. It keeps looking on its own.' : 'Nothing here is missing.'}</p>` : ''}
    ${shelf(app, 'Gets you something new', s.new, true)}
    ${shelf(app, 'Better than what you have', s.better, !s.new.length)}
    ${hiddenBlock(app, s.hidden)}`;
  return frame(app, { head: header(app), body, cls: 'r2-e' });
}

export function init(app) {
  const q = new URLSearchParams(location.search);
  app.det = q.get('open');
  app.hiddenOpen = q.get('hidden') === '1' ? 'all' : null;
  app.ft = 'all';
  app.fi = 'all';
  onScope(app);
  wire(app, (e, qq) => {
    let el;
    if ((el = qq('[data-ft]'))) { app.ft = el.dataset.ft; app.render(); return true; }
    if ((el = qq('[data-ep-menu]'))) { e.stopPropagation(); epMenu(app, el); return true; }
    return false;
  });
}

// Opening from a cell or an Item widens to its season and narrows with the chips instead, so "All" is one tap away.
export function onScope(app) {
  const { sc, fx } = app;
  app.ft = 'all';
  app.fi = 'all';
  if (sc.kind === 'cell') {
    const c = parseCell(sc.cell);
    app.ft = c.target;
    if (fx.work.kind !== 'movie') app.fi = c.item;
  }
  if (sc.kind === 'item') app.fi = sc.item;
  if (sc.kind === 'cell' || sc.kind === 'item') app.sc = normScope(fx, { ...sc, kind: fx.work.kind === 'movie' ? 'work' : 'level' });
}

export const escape = escapeShared;

export function after(app, host) {
  if (app.scrolled) return;
  const el = host.querySelector('.r2-row.open, .r2-hr.open, .r2-hidden.open');
  if (el) { el.scrollIntoView({ block: 'start' }); app.scrolled = true; }
}
