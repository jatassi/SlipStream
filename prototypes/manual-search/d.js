// PROTOTYPE · D · The plan. SlipStream proposes the smallest set of grabs that fills everything in scope, as a checklist with one Grab button. Each pick can be swapped for another choice; what it wouldn't grab folds to one line. Throwaway.

import { I, esc, header, frame, facts, outLines, details, detBtn, hiddenBlock, judgeAll, wouldGrab, needKeys, wire, escapeShared, toast, searching } from './plain.js';
import { buildPlan, planHeadline, outcome, keysPhrase, fmt } from './model.js';

function plan(app) {
  return buildPlan(app.fx, app.sc, app.rows(), app.forced || [], app.excluded);
}

function altsList(app, step) {
  const alts = step.alts.slice(0, 5);
  const fewer = alts.some((row) => outcome(app.fx, row, step.covers, app.sc).keys.length < step.covers.length);
  const note = fewer ? `<p class="r2-alts-n">${I('InformationCircle', 'ic-s')}Some of these cover less. If you pick one, SlipStream adds grabs for the rest.</p>` : '';
  return `<div class="r2-alts" role="region" aria-label="Other choices">${note}${alts.map((row) => {
    const o = outcome(app.fx, row, step.covers, app.sc);
    return `<div class="r2-alt"><div class="r2-alt-b">${facts(app, row.r)}${outLines(o, { also: false })}</div><button class="btn sm" data-swap="${step.row.r.id}:${row.r.id}">Use this instead</button></div>`;
  }).join('')}</div>`;
}

function stepRow(app, step) {
  const { r } = step.row;
  const o = outcome(app.fx, step.row, step.covers, app.sc);
  const grabbed = app.grabbed.get(r.id);
  const on = !app.off.has(r.id);
  const altsOpen = app.alts === r.id;
  const n = Math.min(step.alts.length, 5);
  const check = grabbed ? `<span class="r2-ck-done">${I('Tick02')}</span>` : `<label class="ck r2-ck"><input type="checkbox" data-pick="${r.id}" ${on ? 'checked' : ''} aria-label="Include this grab"><span aria-hidden="true">${I('Tick02')}</span></label>`;
  return `<li class="r2-step${on || grabbed ? '' : ' off'}">${check}<div class="r2-step-b">
    <span class="r2-step-t">${facts(app, r)}${grabbed ? '<span class="chip chip-live r2-onway"><span class="dot dot-draft"></span>On its way</span>' : ''}</span>
    ${outLines(o)}
    <div class="r2-acts">${n ? `<button class="btn sm ghost" data-alts="${r.id}" aria-expanded="${altsOpen}">Other choices (${n})${I(altsOpen ? 'ArrowUp01' : 'ArrowDown01', 'ic-s')}</button>` : ''}${detBtn(app, r)}</div>
    ${altsOpen ? altsList(app, step) : ''}
    ${app.det === r.id ? details(app, step.row) : ''}
  </div></li>`;
}

function missingRow(app, keys) {
  return `<li class="r2-step r2-none"><span class="r2-none-i">${I('Search01')}</span><div class="r2-step-b"><span class="r2-out">Nothing found yet for ${esc(keysPhrase(app.fx, keys, app.sc))}</span><span class="t3">SlipStream keeps looking on its own, and grabs it when one turns up.</span></div></li>`;
}

export function render(app) {
  const p = plan(app);
  const rows = app.rows();
  const pickedIds = new Set(p.steps.map((s) => s.row.r.id));
  const hidden = judgeAll(app, rows, needKeys(app.fx, app.sc)).filter((x) => !wouldGrab(x.out) && !pickedIds.has(x.row.r.id));
  const live = p.steps.filter((s) => !app.off.has(s.row.r.id) && !app.grabbed.has(s.row.r.id));
  const size = live.reduce((n, s) => n + s.row.r.size, 0);
  const waiting = searching(app) && !rows.length;
  const sentence = waiting ? 'Looking for releases…' : planHeadline(app.fx, p, app.sc);
  const sub = waiting ? '' : p.steps.length ? `Picked from ${rows.length} releases${searching(app) ? ' so far' : ''}. Uncheck any you’d rather skip.` : '';
  const body = `<section class="r2-lead"><h2 class="r2-sent">${esc(sentence)}</h2>${sub ? `<p class="r2-sub">${esc(sub)}</p>` : ''}</section>
    ${p.steps.length || p.missing.length ? `<ol class="r2-plan">${p.steps.map((s) => stepRow(app, s)).join('')}${!waiting && p.missing.length ? missingRow(app, p.missing) : ''}</ol>` : ''}
    ${hiddenBlock(app, hidden)}`;
  const allDone = p.steps.length && !live.length && p.steps.every((s) => app.grabbed.has(s.row.r.id) || app.off.has(s.row.r.id));
  const foot = p.steps.length ? `<div class="r2-foot"><span class="r2-foot-t">${allDone ? 'Sent. Follow them in Activity.' : live.length ? `${live.length} of ${p.steps.length} picked<span class="s-sep">·</span><span class="num">${fmt.size(size)}</span> in all` : 'Nothing picked'}</span>${allDone ? '<button class="btn r2-go" data-close-ms>Done</button>' : `<button class="btn pri r2-go" data-plan-grab ${live.length ? '' : 'disabled'}>${I('Download04')}Grab ${live.length || ''}</button>`}</div>` : '';
  return frame(app, { head: header(app), body, foot, cls: 'r2-d' });
}

export function init(app) {
  const q = new URLSearchParams(location.search);
  app.forced = null;
  app.excluded = new Set();
  app.off = new Set();
  app.alts = q.get('alts');
  app.det = q.get('open');
  app.hiddenOpen = q.get('hidden') === '1' ? 'all' : null;
  wire(app, (e, qq) => {
    let el;
    if ((el = qq('[data-alts]'))) { app.alts = app.alts === el.dataset.alts ? null : el.dataset.alts; app.render(); return true; }
    if ((el = qq('[data-swap]'))) {
      const [from, to] = el.dataset.swap.split(':');
      app.forced = plan(app).steps.map((s) => (s.row.r.id === from ? to : s.row.r.id));
      app.excluded.add(from);
      app.excluded.delete(to);
      if (app.off.delete(from)) app.off.add(to);
      app.alts = null;
      app.render();
      return true;
    }
    if ((el = qq('[data-plan-grab]'))) {
      const go = plan(app).steps.filter((s) => !app.off.has(s.row.r.id) && !app.grabbed.has(s.row.r.id));
      go.forEach((s) => app.grabbed.set(s.row.r.id, s.row.fillTargets.map((t) => t.name).join(', ')));
      toast(`Sent ${go.length} release${go.length > 1 ? 's' : ''} to download. Follow ${go.length > 1 ? 'them' : 'it'} in Activity`);
      app.render();
      return true;
    }
    return false;
  });
  document.addEventListener('change', (e) => {
    const el = e.target.closest && e.target.closest('[data-pick]');
    if (!el) return;
    if (el.checked) app.off.delete(el.dataset.pick); else app.off.add(el.dataset.pick);
    app.render();
  });
}

export function onScope(app) { app.forced = null; app.excluded = new Set(); app.off = new Set(); app.alts = null; app.det = null; }

export function escape(app) {
  if (app.alts) { app.alts = null; app.render(); return true; }
  return escapeShared(app);
}

export function after(app, host) {
  if (app.scrolled) return;
  const el = host.querySelector('.r2-alts, .r2-det, .r2-hidden.open');
  if (el) { (el.closest('.r2-step, .r2-hr') || el).scrollIntoView({ block: 'start' }); app.scrolled = true; }
}
