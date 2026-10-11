// PROTOTYPE · C · Exceptions first. What isn't met leads; met cells fold into a summary that expands to the full matrix.
import { esc, I, ago, md, hm } from '../../variants/lit-console/variant.js';
import { items, cells, counts, STATES, MISSING_WHY, targetsOf, editionsOf, levelNodes } from '../model.js';

const ui = { lv: null, metOpen: false, sel: null };

const LIVE = new Set(['downloading', 'converting']);
const NEEDS = ['failed', 'upgrade', 'not-found'];
const sectionOf = (c) => {
  if (c.state === 'failed' || c.state === 'upgrade') return 'needs';
  if (c.state === 'missing') return c.why === 'not-found' ? 'needs' : 'waiting';
  if (LIVE.has(c.state)) return 'moving';
  return c.state;
};
const needRank = (c) => NEEDS.indexOf(c.state === 'missing' ? c.why : c.state);
const plural = (n, noun) => `${n} ${noun.toLowerCase()}${n === 1 ? '' : 's'}`;
const day = (d) => md(d.length === 10 ? `${d}T12:00` : d);
const chipLabel = (c) => (c.state === 'missing' ? MISSING_WHY[c.why] : STATES[c.state].short ?? STATES[c.state].label);
const chip = (c) => `<span class="st st-${c.state}">${I(STATES[c.state].icon)}${esc(chipLabel(c))}</span>`;
const sq = (state, extra = '') => `<span class="sq sq-${state}"${extra}></span>`;

function shortEnd(a, b) {
  let i = 0;
  while (i < a.length && a[i] === b[i]) i++;
  while (i > 0 && !/[A-Za-z]/.test(b[i - 1])) i--;
  return b.slice(Math.max(0, i - 1));
}

function model(vctx) {
  const { work, desc } = vctx;
  const lvs = levelNodes(work);
  const all = items(work, desc).map((it, idx) => ({ ...it, idx, lv: it.path.length ? lvs.indexOf(it.path[0]) : -1 }));
  const eds = editionsOf(work), tgs = targetsOf(work);
  const rows = cells(work, desc, all).map((x) => ({ ...x, ei: eds.indexOf(x.edition), ti: tgs.findIndex((t) => t.id === x.target.id), id: `${x.idx}|${x.edition.id}|${x.target.id}` }));
  return { work, desc, lvs, all, eds, tgs, rows, many: desc.layout.items === 'many' };
}

function runs(list) {
  const sorted = [...list].sort((a, b) => a.lv - b.lv || a.ei - b.ei || a.ti - b.ti || a.idx - b.idx);
  const out = [];
  for (const x of sorted) {
    const last = out[out.length - 1];
    const key = `${x.lv}|${x.ei}|${x.ti}|${x.cell.state}|${x.cell.why}|${x.cell.detail}`;
    if (x.cell.state === 'missing' && last && last.key === key && last.to.idx === x.idx - 1) { last.to = x; last.n += 1; continue; }
    out.push({ key, from: x, to: x, n: 1, cell: x.cell, edition: x.edition, target: x.target, lv: x.lv, ei: x.ei, ti: x.ti });
  }
  return out.sort((a, b) => a.lv - b.lv || a.from.idx - b.from.idx || a.ei - b.ei || a.ti - b.ti);
}

function where(m, r, ph) {
  const et = `<b>${esc(r.edition.name)}</b><span class="s-sep">·</span><b>${esc(r.target.short)}</b>`;
  if (!m.many) return `<div class="tw"><span class="tw-one">${et}</span></div>`;
  const code = r.n > 1 ? `${r.from.code}–${shortEnd(r.from.code, r.to.code)}` : r.from.code;
  const sub = r.n > 1 ? plural(r.n, m.desc.item.noun) : r.from.label;
  return `<div class="tw"><span class="tw-it"><span class="mono tw-code">${esc(code)}</span><span class="t3 ell">${esc(sub)}</span></span><span class="tw-et${ph ? ' t2' : ''}">${et}</span></div>`;
}

const jobOf = (vctx, c) => vctx.data.jobs.find((j) => j.id === c.job);

function liveBits(vctx, c) {
  const j = jobOf(vctx, c);
  const { format } = vctx;
  const stage = c.state === 'converting' ? `Converting on ${c.node}` : `${j.stage} on ${j.node}`;
  const bar = `<div class="bar" role="progressbar" aria-valuemin="0" aria-valuemax="100" aria-label="${esc(stage)}" data-live-progress="${j.id}" style="--p:${j.progress}"><span class="fill"></span></div>`;
  const stats = `<span class="draft-t" data-live-pct="${j.id}">${format.pct(j.progress)}</span><span class="s-sep">·</span><span data-live-rate="${j.id}">${format.rate(j)}</span><span class="s-sep">·</span><span data-live-eta="${j.id}">${format.eta(j.etaMin)}</span> left`;
  const what = c.state === 'converting' ? `from the ${esc(c.from)} · <span class="mono">${esc(c.settings)}</span> · ${esc(c.estSize)}` : `<span class="mono">${esc(c.release)}</span> · ${esc(c.size)}`;
  return { stage, bar, stats, what };
}

function failCard(m, r, ph) {
  const c = r.cell;
  return `<div class="tri-fail${ph ? ' ph' : ''}">
    <div class="tf-h"><span class="chip-fail">${I('circle-x')}Failed at ${esc(c.stage)}</span>${where(m, r, ph)}<span class="fail-dim num tf-at">${md(c.at)} ${hm(c.at)} · ${ago(c.at)}</span></div>
    <dl class="fail-kv"><dt>Cause</dt><dd>${esc(c.cause)}</dd><dt>Recovery</dt><dd>${esc(c.recovery)}</dd><dt>Release</dt><dd><span class="mono rel">${esc(c.release)}</span><span class="chip chip-onfail">${I('ban')}Blocklisted</span></dd></dl>
    <div class="tf-act"><button class="btn fail${ph ? '' : ' sm'}" data-toast="Manual search is not in this prototype">${I('search')}Manual search</button><button class="btn${ph ? '' : ' sm'}" data-toast="Removing from the Blocklist is not in this prototype">${I('ban')}Unblock release</button></div>
  </div>`;
}

function rowHtml(vctx, m, r, ph) {
  const c = r.cell;
  if (c.state === 'failed') return failCard(m, r, ph);
  let detail = '', sub = '', act = '';
  if (c.state === 'upgrade') {
    detail = `Score <span class="num">${c.score}</span> ${I('arrow-right', 'tr-arr')} <span class="num draft-u">${c.found.score}</span> <span class="t3">found · cutoff <span class="num">${c.cutoff}</span> · <span class="num">${esc(c.size)}</span> now, <span class="num">${esc(c.found.size)}</span> on ${esc(c.found.indexer)}</span>`;
    sub = `<span class="mono">${esc(c.found.release)}</span>`;
    act = `<button class="btn pri sm" data-toast="Grabbing an upgrade is not in this prototype">${I('circle-arrow-up')}Grab upgrade</button>`;
  } else if (c.state === 'missing') {
    detail = esc(c.detail);
    if (c.why === 'not-found') act = `<button class="btn sm" data-toast="Manual search is not in this prototype">${I('search')}Manual search</button>`;
  } else if (LIVE.has(c.state)) {
    const b = liveBits(vctx, c);
    return `<li class="tr tr-live${ph ? ' ph' : ''}">${where(m, r, ph)}<div class="tr-st">${chip(c)}</div>
      <div class="tr-d"><div class="tr-lv"><span class="tr-stage">${esc(b.stage)}</span><span class="num tr-stats">${b.stats}</span></div>${b.bar}<p class="tr-sub t3 ell">${b.what}</p></div>
      <div class="tr-a"><button class="btn ghost sm" data-toast="Pausing is not in this prototype">${I('pause')}Pause</button></div></li>`;
  }
  return `<li class="tr${ph ? ' ph' : ''}">${where(m, r, ph)}<div class="tr-st">${chip(c)}</div><div class="tr-d"><p class="tr-l">${detail}</p>${sub ? `<p class="tr-sub t3 ell">${sub}</p>` : ''}</div>${act ? `<div class="tr-a">${act}</div>` : ''}</li>`;
}

function section(title, count, inner, ph, extra = '', cls = '') {
  if (ph) return `<section class="psec tri-sec ${cls}"><div class="psec-h"><h2>${title}</h2>${count}</div>${extra}${inner}</section>`;
  return `<section class="wk-sec tri-sec ${cls}"><div class="sec-h"><h2>${title}</h2>${count}${extra}</div>${inner}</section>`;
}

const empty = (text, ph) => `<p class="tri-empty${ph ? ' ph' : ''} t3">${I('check')}${esc(text)}</p>`;

function sourcing(m, ph) {
  return `<p class="tri-src${ph ? ' ph' : ''} t3"><span class="tri-src-h">Sourcing</span>${m.tgs.map((t) => `<span>${I(t.sourcing === 'downloaded' ? 'download' : 'replace')}<b>${esc(t.short)}</b> ${esc(t.label)}</span>`).join('')}</p>`;
}

function strip(m, ph) {
  const unairedRow = (lvRows, node) => {
    const un = m.all.filter((it) => it.lv === m.lvs.indexOf(node) && m.eds.every((e) => m.tgs.every((t) => it.item.cells[e.id][t.id].state === 'unaired')));
    if (!un.length) return '';
    const dates = un.map((it) => it.item.cells[m.eds[0].id][m.tgs[0].id].airs).sort();
    const when = dates.length === 1 ? `airs ${day(dates[0])}` : `air ${day(dates[0])} to ${day(dates[dates.length - 1])}`;
    return `<span class="ts-un t3">${I('calendar-clock')}${plural(un.length, m.desc.item.noun)} ${when}</span>`;
  };
  const lines = m.lvs.map((node, i) => {
    const lr = m.rows.filter((r) => r.lv === i);
    const c = counts(lr);
    const wanted = lr.length - c.unaired;
    const bar = Object.keys(STATES).filter((s) => c[s]).map((s) => `<span class="sq sq-${s}" style="flex-grow:${c[s]}" title="${c[s]} ${esc(STATES[s].label)}"></span>`).join('');
    const tally = Object.keys(STATES).filter((s) => c[s] && s !== 'unaired' && s !== 'met').map((s) => `<span>${sq(s)}<span class="num">${c[s]}</span> ${esc((STATES[s].short ?? STATES[s].label).toLowerCase())}</span>`).join('');
    const nIt = m.all.filter((it) => it.lv === i).length;
    return `<button class="ts${ui.lv === i ? ' on' : ''}" data-tri-lv="${i}" aria-pressed="${ui.lv === i}">
      <span class="ts-n"><b>${esc(m.desc.levels[0].label(node.n))}</b><span class="t3">${plural(nIt, m.desc.item.noun)}</span></span>
      <span class="ts-bar">${bar}</span>
      <span class="ts-c"><span class="ts-met num">${c.met}/${wanted}</span><span class="t3 ts-metl">met</span>${tally ? `<span class="ts-tally">${tally}</span>` : ''}</span>
      ${unairedRow(lr, node)}
    </button>`;
  }).join('');
  const lvNoun = m.desc.levels[0].noun;
  const head = `<span class="t3">${ui.lv == null ? `All ${lvNoun.toLowerCase()}s · tap one to filter` : `Showing ${esc(m.desc.levels[0].label(m.lvs[ui.lv].n))}`}</span>${ui.lv == null ? '' : `<button class="btn ghost sm" data-tri-lv="">${I('x')}Show all</button>`}`;
  return ph
    ? `<section class="psec tri-sec"><div class="psec-h"><h2>${esc(lvNoun)}s</h2>${ui.lv == null ? '' : '<button class="btn ghost sm" data-tri-lv="">Show all</button>'}</div><div class="glist tri-strip ph">${lines}</div></section>`
    : `<section class="tri-sec"><div class="sec-h"><h2>${esc(lvNoun)}s</h2>${head}</div><div class="tri-strip">${lines}</div></section>`;
}

function cellDetail(c) {
  if (c.state === 'met') return `<dl class="kv tri-kv"><dt>Score</dt><dd><span class="num">${c.score}</span> <span class="t3">cutoff <span class="num">${c.cutoff}</span></span></dd><dt>Size</dt><dd class="num">${esc(c.size)}</dd><dt>Video</dt><dd class="mono">${esc(c.video)}</dd><dt>Audio</dt><dd class="mono">${esc(c.audio)}</dd><dt>Source</dt><dd>${esc(c.source)}</dd></dl>`;
  if (c.state === 'unaired') return `<p class="t2">Airs ${day(c.airs)}. Search starts then.</p>`;
  if (c.state === 'missing') return `<p class="t2">${esc(c.detail)}</p>`;
  if (c.state === 'upgrade') return `<p class="t2">Score <span class="num">${c.score}</span> of cutoff <span class="num">${c.cutoff}</span>; found <span class="num">${c.found.score}</span>.</p>`;
  if (c.state === 'failed') return `<p class="fail-t">${esc(c.cause)}</p>`;
  return '<p class="t2">Moving. See the list above for progress.</p>';
}

function cellShort(vctx, c) {
  if (c.state === 'met') return `<span class="num">${c.score}</span>`;
  if (c.state === 'upgrade') return `<span class="num">${c.score}→${c.found.score}</span>`;
  if (c.state === 'missing') return `<span class="t3">${esc(MISSING_WHY[c.why].split(' ')[0].replace('In', 'Grace').replace('Not', 'Not found'))}</span>`;
  if (c.state === 'unaired') return `<span class="t3 num">${day(c.airs)}</span>`;
  if (LIVE.has(c.state)) { const j = jobOf(vctx, c); return `<span class="num draft-t" data-live-pct="${j.id}">${vctx.format.pct(j.progress)}</span>`; }
  return '<span class="fail-t">Failed</span>';
}

function metSection(vctx, m, rows, ph) {
  const c = counts(rows);
  const wanted = rows.length - c.unaired;
  const btn = `<button class="btn ghost sm tri-tog" data-tri-met aria-expanded="${ui.metOpen}">${I(ui.metOpen ? 'chevron-up' : 'chevron-down')}${ui.metOpen ? 'Collapse' : 'Full matrix'}</button>`;
  const count = `<span class="t3 num">${c.met} of ${wanted}</span>`;
  const sel = ui.sel && rows.find((r) => r.id === ui.sel);
  const detail = sel ? `<div class="tri-det"><div class="tri-det-h">${chip(sel.cell)}<b>${m.many ? `<span class="mono">${esc(sel.code)}</span> ${esc(sel.label)} · ` : ''}${esc(sel.edition.name)} · ${esc(sel.target.short)}</b><button class="ib" data-tri-cell="" aria-label="Close">${I('x')}</button></div>${cellDetail(sel.cell)}</div>` : '';
  let inner;
  if (!ui.metOpen) {
    const its = m.all.filter((it) => ui.lv == null || it.lv === ui.lv);
    const strip = (e, t) => {
      const list = its.map((it) => ({ it, cell: it.item.cells[e.id][t.id] })).filter((x) => x.cell.state !== 'unaired');
      const met = list.filter((x) => x.cell.state === 'met').length;
      const sqs = list.map((x, k) => `${k && x.it.lv !== list[k - 1].it.lv ? '<i class="tm-gap"></i>' : ''}${sq(x.cell.state)}`).join('');
      return { sqs, met, n: list.length };
    };
    if (ph) {
      inner = `<ul class="glist">${m.eds.map((e) => `<li class="gl tm-ph"><b class="tm-ed">${esc(e.name)}</b><span class="tm-ts">${m.tgs.map((t) => { const s = strip(e, t); return `<span class="tm-t"><span class="t3">${esc(t.short)}</span><span class="num">${s.met}/${s.n}</span></span>`; }).join('')}</span></li>`).join('')}</ul>`;
    } else {
      inner = `<div class="tri-mini"><table class="tm"><thead><tr><th></th>${m.tgs.map((t) => `<th scope="col">${esc(t.name)}</th>`).join('')}</tr></thead><tbody>${m.eds.map((e) => `<tr><th scope="row">${esc(e.name)}</th>${m.tgs.map((t) => { const s = strip(e, t); return `<td><span class="tm-sqs">${s.sqs}</span><span class="num tm-n${s.met === s.n ? '' : ' t3'}">${s.met}/${s.n}</span></td>`; }).join('')}</tr>`).join('')}</tbody></table></div>`;
    }
  } else if (!m.many) {
    inner = ph
      ? `<ul class="glist">${m.eds.map((e) => `<li class="tmx-ph"><b>${esc(e.name)}</b>${m.tgs.map((t) => { const c2 = rows.find((r) => r.edition === e && r.target.id === t.id).cell; return `<div class="tmx-pr"><span class="t2">${esc(t.short)}</span>${chip(c2)}${c2.state === 'met' ? `<span class="num t3">${c2.score} / ${c2.cutoff} · ${esc(c2.size)}</span>` : ''}</div>`; }).join('')}</li>`).join('')}</ul>`
      : `<table class="tmx"><thead><tr><th class="tmx-corner t3">Edition ${I('move-right')} Target</th>${m.tgs.map((t) => `<th scope="col"><b>${esc(t.name)}</b><span class="t3">${I(t.sourcing === 'downloaded' ? 'download' : 'replace')}${esc(t.label)}</span></th>`).join('')}</tr></thead><tbody>${m.eds.map((e) => `<tr><th scope="row"><b>${esc(e.name)}</b><span class="t3">${esc(e.note)}</span></th>${m.tgs.map((t) => { const c2 = rows.find((r) => r.edition === e && r.target.id === t.id).cell; return `<td class="${c2.state === 'met' ? '' : 'dim'}">${chip(c2)}${c2.state === 'met' ? cellDetail(c2) : `<p class="t3 tmx-l">${c2.state === 'missing' ? esc(c2.detail) : 'Listed above'}</p>`}</td>`; }).join('')}</tr>`).join('')}</tbody></table>`;
  } else {
    const its = m.all.filter((it) => ui.lv == null || it.lv === ui.lv);
    const head = `<thead><tr><th rowspan="2"></th>${m.eds.map((e) => `<th colspan="${m.tgs.length}" class="tg-ed">${esc(e.name)}</th>`).join('')}</tr><tr>${m.eds.map(() => m.tgs.map((t, k) => `<th class="tg-t${k === 0 ? ' g0' : ''}">${esc(t.short)}</th>`).join('')).join('')}</tr></thead>`;
    let lastLv = null;
    const body = its.map((it) => {
      const lvRow = it.lv !== lastLv ? `<tr class="tg-lv"><th colspan="${1 + m.eds.length * m.tgs.length}">${esc(m.desc.levels[0].label(m.lvs[it.lv].n))}</th></tr>` : '';
      lastLv = it.lv;
      return `${lvRow}<tr><th scope="row" class="tg-it"><span class="mono">${esc(it.code)}</span>${ph ? '' : `<span class="t3 ell">${esc(it.label)}</span>`}</th>${m.eds.map((e) => m.tgs.map((t, k) => { const cl = it.item.cells[e.id][t.id]; const id = `${it.idx}|${e.id}|${t.id}`; return `<td class="${k === 0 ? 'g0' : ''}${ui.sel === id ? ' sel' : ''}"><button class="tg-c" data-tri-cell="${id}" aria-label="${esc(`${it.code} ${e.name} ${t.short}: ${chipLabel(cl)}`)}">${sq(cl.state)}${ph ? '' : cellShort(vctx, cl)}</button></td>`; }).join('')).join('')}</tr>`;
    }).join('');
    inner = `<div class="tg-wrap${ph ? ' ph' : ''}"><table class="tg">${head}<tbody>${body}</tbody></table></div>${detail}`;
  }
  return section('Met', count, inner, ph, ph ? `<div class="tri-ph-tog">${btn}</div>` : `<span class="grow"></span>${btn}`, 'tri-met');
}

function body(vctx, ph) {
  const m = model(vctx);
  const rows = m.rows.filter((r) => ui.lv == null || r.lv === ui.lv);
  const by = (s) => rows.filter((r) => sectionOf(r.cell) === s);
  const needs = runs(by('needs')).sort((a, b) => needRank(a.cell) - needRank(b.cell));
  const moving = runs(by('moving'));
  const waiting = runs(by('waiting'));
  const nCells = (list) => list.reduce((n, r) => n + r.n, 0);
  const cnt = (list) => `<span class="t3 num">${nCells(list)} ${nCells(list) === 1 ? 'cell' : 'cells'}${list.length !== nCells(list) ? ` in ${list.length} row${list.length === 1 ? '' : 's'}` : ''}</span>`;
  const list = (rs, cls = '') => `<ul class="tri-list ${cls}${ph ? ' glist ph' : ''}">${rs.map((r) => rowHtml(vctx, m, r, ph)).join('')}</ul>`;
  const fails = needs.filter((r) => r.cell.state === 'failed');
  const rest = needs.filter((r) => r.cell.state !== 'failed');
  const needsInner = needs.length ? `${fails.map((r) => failCard(m, r, ph)).join('')}${rest.length ? list(rest) : ''}` : empty('Nothing needs you', ph);
  const legend = ph ? '' : `<span class="grow"></span><span class="sx-legend">${Object.keys(STATES).filter((s) => m.many || s !== 'unaired').map((s) => `<span>${sq(s)}${esc(STATES[s].short ?? STATES[s].label)}</span>`).join('')}</span>`;
  return `${m.many ? strip(m, ph) : ''}
    ${section('Needs you', cnt(needs), needsInner, ph, legend, 'tri-needs')}
    ${section('Moving', cnt(moving), moving.length ? list(moving) : empty('Nothing is moving', ph), ph)}
    ${section('Waiting', cnt(waiting), waiting.length ? `${list(waiting, 'tri-wait')}${sourcing(m, ph)}` : empty('Nothing is waiting', ph), ph)}
    ${metSection(vctx, m, rows, ph)}`;
}

export const wide = (vctx) => `<div class="tri" data-tri>${body(vctx, false)}</div>`;
export const phone = (vctx) => `<div class="tri tri-p" data-tri>${body(vctx, true)}</div>`;

export function wire(root, vctx) {
  const host = root.querySelector('[data-tri]');
  const ph = host.classList.contains('tri-p');
  host.addEventListener('click', (e) => {
    const lv = e.target.closest('[data-tri-lv]');
    const met = e.target.closest('[data-tri-met]');
    const cell = e.target.closest('[data-tri-cell]');
    if (!lv && !met && !cell) return;
    if (lv) { const v = lv.dataset.triLv === '' ? null : Number(lv.dataset.triLv); ui.lv = ui.lv === v ? null : v; ui.sel = null; }
    if (met) { ui.metOpen = !ui.metOpen; ui.sel = null; }
    if (cell) ui.sel = cell.dataset.triCell === '' || ui.sel === cell.dataset.triCell ? null : cell.dataset.triCell;
    host.innerHTML = body(vctx, ph);
    window.lucide && window.lucide.createIcons();
  });
}
