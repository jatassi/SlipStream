// PROTOTYPE · F · Needs, then choices. What this Work needs, in words and worst first; pick one to see its choices as cards, the top one recommended. Desk: master-detail. Phone: a pushed page. Opening from one cell lands on that need. Throwaway.

import { I, esc, header, frame, facts, outLines, rawName, details, hiddenBlock, wouldGrab, wire, escapeShared, searching, grabBtn, recTag, scopeName } from './plain.js';
import { needGroups, needKeys, metKeys, outcome, rankCmp, normScope, parseCell, describeFile, choiceCmp } from './model.js';
import { sq } from './common.js';

function needs(app) { return needGroups(app.fx, needKeys(app.fx, app.sc), app.sc); }

function choicesFor(app, need) {
  const rank = new Map([...app.rows()].sort(rankCmp).map((row, i) => [row.r.id, i]));
  const t = app.fx.targets.find((x) => x.id === need.tid);
  const its = new Set(need.keys.map((k) => parseCell(k).item));
  const pick = [], hidden = [];
  for (const row of app.rows()) {
    const o = outcome(app.fx, row, need.keys, app.sc);
    if (wouldGrab(o)) pick.push({ row, out: o });
    else if (row.r.res === t.res && row.r.coverage.some((id) => its.has(id))) hidden.push({ row, out: o });
  }
  pick.sort(choiceCmp(app.fx, rank));
  return { pick, hidden };
}

const onWay = (app, need) => {
  const got = new Set([...app.grabbed.keys()].flatMap((id) => { const row = app.rowById(id); return outcome(app.fx, row, need.keys, app.sc).keys; }));
  return need.keys.every((k) => got.has(k));
};

function needSub(app, need, ch) {
  if (onWay(app, need)) return '<span class="r2-sub-live"><span class="dot dot-draft"></span>On its way</span>';
  const c = app.fx.cells[need.keys[0]];
  const why = need.state === 'failed' && c.plain ? `Last try failed: ${c.plain}` : need.state === 'upgrade' && c.file ? `You have a ${describeFile(c.file)} copy` : '';
  const n = ch.pick.length;
  const what = n ? `${n} choice${n > 1 ? 's' : ''}` : searching(app) ? 'Looking…' : 'Nothing found yet';
  return `${why ? `${esc(why)}<span class="s-sep">·</span>` : ''}${what}`;
}

function needRow(app, need) {
  const ch = choicesFor(app, need);
  const on = app.need === need.id && !app.phone;
  return `<button class="r2-need${on ? ' on' : ''}" data-need="${need.id}" aria-pressed="${on}">${sq(need.state, 'sq-lg')}<span class="r2-need-b"><span class="r2-need-t"><b>${esc(need.word)}:</b> ${esc(need.label)}</span><span class="r2-need-s t3">${needSub(app, need, ch)}</span></span>${app.phone ? I('ArrowRight01', 'r2-chev') : ''}</button>`;
}

function fine(app) {
  const keys = metKeys(app.fx, app.sc);
  if (!keys.length) return '';
  const open = app.fine;
  return `<div class="r2-fine"><button class="r2-fold" data-fine aria-expanded="${open}">${I(open ? 'ArrowDown01' : 'ArrowRight01', 'ic-s')}<span>Already fine: ${keys.length}</span><span class="s-sep">·</span><b>${open ? 'Hide' : 'Show'}</b></button>${open ? needGroups(app.fx, keys, app.sc).map((g) => `<p class="r2-fine-r">${sq('met')}<span>${esc(g.label)}</span></p>`).join('') : ''}</div>`;
}

function needsList(app, list) {
  const name = app.sc.kind === 'work' ? app.fx.work.title : scopeName(app.fx, app.sc);
  const head = searching(app) && !app.rows().length && !list.length ? 'Looking…' : list.length ? `${name} needs ${list.length} thing${list.length > 1 ? 's' : ''}` : `${name} has everything it needs`;
  return `<div class="r2-needs"><h2 class="r2-sent r2-sent-f">${esc(head)}</h2><div class="r2-nlist">${list.map((n) => needRow(app, n)).join('')}</div>${fine(app)}</div>`;
}

function card(app, x, i) {
  const { r } = x.row;
  const rec = i === 0;
  const open = app.det === r.id;
  return `<div class="r2-card${rec ? ' rec' : ''}${open ? ' open' : ''}"><div class="r2-card-m"><button class="r2-card-b" data-det="${r.id}" aria-expanded="${open}">${facts(app, r, { tag: rec ? recTag() : '', chev: open ? 'ArrowUp01' : 'ArrowDown01' })}${outLines(x.out)}${rec ? rawName(r) : ''}</button><span class="r2-card-a">${grabBtn(app, x.row, { primary: rec, sm: !rec })}</span></div>${open ? details(app, x.row) : ''}</div>`;
}

function choicesPane(app, need, total) {
  const ch = choicesFor(app, need);
  const c = app.fx.cells[need.keys[0]];
  const back = app.phone || app.landed ? `<button class="r2-back" data-back>${I('ArrowLeft01')}All needs (${total})</button>` : '';
  const lead = need.state === 'failed' ? `The last try failed: ${c.plain || 'the download broke'}.` : need.state === 'upgrade' ? (c.file ? `You have a ${describeFile(c.file)} copy. SlipStream keeps looking for a better one.` : 'You have a copy. SlipStream keeps looking for a better one.') : 'Nothing downloaded yet.';
  const list = ch.pick.length ? `<div class="r2-cards">${ch.pick.map((x, i) => card(app, x, i)).join('')}</div>`
    : `<p class="r2-empty">${searching(app) ? '<span class="spin"></span>Looking…' : `${I('Search01')}Nothing found yet that SlipStream would grab. It keeps looking on its own.`}</p>`;
  return `<div class="r2-pane">${back}<div class="r2-pane-h"><h3 class="r2-pane-t">${sq(need.state, 'sq-lg')}<span><b>${esc(need.word)}:</b> ${esc(need.label)}</span></h3><p class="r2-sub">${esc(lead)}</p></div>${list}${hiddenBlock(app, ch.hidden, need.id, 'SlipStream wouldn’t grab for this')}</div>`;
}

export function render(app) {
  const list = needs(app);
  if (app.need && !list.some((n) => n.id === app.need)) app.need = null;
  const sel = list.find((n) => n.id === app.need);
  if (app.phone) {
    const body = app.page === 'choices' && sel ? choicesPane(app, sel, list.length) : needsList(app, list);
    return frame(app, { head: header(app), body, cls: `r2-f${app.page === 'choices' ? ' r2-pushed' : ''}` });
  }
  const pane = sel || list[0];
  if (!app.need && pane) app.need = pane.id;
  const body = app.landed && sel ? `<div class="r2-solo">${choicesPane(app, sel, list.length)}</div>`
    : `<div class="r2-md"><nav class="r2-master" aria-label="Needs" data-keep="master">${needsList(app, list)}</nav><div class="r2-detail" data-keep="detail">${pane ? choicesPane(app, pane, list.length) : ''}</div></div>`;
  return frame(app, { head: header(app), body, cls: 'r2-f' });
}

// A need is a run of cells sharing state, Target and Edition; this finds the one holding a cell.
function needOf(app, key) {
  return needs(app).find((n) => n.keys.includes(key));
}

export function init(app) {
  const q = new URLSearchParams(location.search);
  app.det = q.get('open');
  app.hiddenOpen = q.get('hidden') === '1' ? q.get('need') : null;
  app.fine = q.get('fine') === '1';
  app.need = q.get('need');
  app.page = app.need ? 'choices' : 'needs';
  app.landed = false;
  onScope(app);
  wire(app, (e, qq) => {
    let el;
    if ((el = qq('[data-need]'))) { app.need = el.dataset.need; app.page = 'choices'; app.det = null; app.render(); return true; }
    if ((el = qq('[data-back]'))) { app.page = 'needs'; app.landed = false; app.det = null; app.render(); return true; }
    if ((el = qq('[data-fine]'))) { app.fine = !app.fine; app.render(); return true; }
    return false;
  });
}

// Opening from a cell widens to the season (or the movie) so "All needs" has somewhere to go, and lands on that cell's need.
export function onScope(app) {
  const { sc, fx } = app;
  if (sc.kind !== 'cell') return;
  app.sc = normScope(fx, { ...sc, kind: fx.work.kind === 'movie' ? 'work' : 'level' });
  const n = needOf(app, sc.cell);
  if (n) { app.need = n.id; app.page = 'choices'; app.landed = true; }
}

export function escape(app) {
  if (escapeShared(app)) return true;
  if ((app.phone && app.page === 'choices') || app.landed) { app.page = 'needs'; app.landed = false; app.render(); return true; }
  return false;
}

export function after(app, host) {
  if (app.scrolled) return;
  const el = host.querySelector('.r2-card.open, .r2-hr.open, .r2-hidden.open');
  if (el) { el.scrollIntoView({ block: 'start' }); app.scrolled = true; }
}

