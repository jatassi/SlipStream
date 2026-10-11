// PROTOTYPE · Matrix variant A, Grid. The matrix stays a matrix at every scale: a rich table for one Item, squares per Item for many.
import { esc, I, md, ago } from '../../variants/lit-console/variant.js';
import { STATES, MISSING_WHY, items, cells, counts, summaryLine, targetsOf, editionsOf, levelNodes, profileOf, STATE_ORDER } from '../model.js';

let X = null;

const num = (v) => `<span class="num">${esc(v)}</span>`;
const mono = (v) => `<span class="mono">${esc(v)}</span>`;
const chip = (s, extra = '') => `<span class="st st-${s}">${I(STATES[s].icon)}${esc(extra || STATES[s].short || STATES[s].label)}</span>`;
const sq = (s, job, label = '') => `<span class="sq sq-${s}"${job ? ` data-live-progress="${job}" style="--p:${X.job(job)?.progress ?? 0}"` : ''}${label ? ` title="${esc(label)}"` : ''}></span>`;
const plural = (n, w) => `${n} ${w}${n === 1 ? '' : 's'}`;

function liveText(kind, j) {
  const f = X.format;
  const v = kind === 'pct' ? f.pct(j.progress) : kind === 'eta' ? f.eta(j.etaMin) : f.rate(j);
  return `<span data-live-${kind}="${j.id}">${esc(v)}</span>`;
}

const SOURCING = {
  downloaded: { icon: 'download', short: () => 'Downloaded' },
  'download-then-derive': { icon: 'timer', short: (t) => `Download, ${t.grace} grace` },
  derived: { icon: 'replace', short: (t, all) => `From ${all.find((x) => x.id === t.from).short}` },
};
const srcShort = (t) => SOURCING[t.sourcing].short(t, X.targets);
const srcIcon = (t) => I(SOURCING[t.sourcing].icon);

const MISSING_ACTION = {
  searching: ['search', 'Search now'],
  'not-found': ['search', 'Search now'],
  grace: ['replace', 'Derive now'],
  queued: ['play', 'Convert now'],
  waiting: null,
};

const btn = (icon, label, cls = '') => `<button class="btn sm ${cls}" data-toast="${esc(label)} is not in this prototype">${I(icon)}${esc(label)}</button>`;

// One entry per state: what a person needs to read and the one thing they can do. Every view reads this.
const DETAIL = {
  met: (c) => ({
    head: c.score >= c.cutoff ? 'Cutoff reached' : 'Below cutoff',
    facts: [['Score', `${num(c.score)} <span class="t3">/ ${c.cutoff}</span>`], ['Size', num(c.size)], ['Video', mono(c.video)], ['Audio', mono(c.audio)], ['Source', c.source === 'derived' ? 'Derived' : 'Downloaded']],
    release: c.file,
  }),
  upgrade: (c) => ({
    head: `Found ${c.found.score} on ${c.found.indexer}`,
    facts: [['Score', `${num(c.score)} ${I('arrow-right', 'gx-arr')} ${num(c.found.score)} <span class="t3">/ ${c.cutoff}</span>`], ['Size', `${num(c.size)} ${I('arrow-right', 'gx-arr')} ${num(c.found.size)}`], ['Video', mono(c.video)], ['Audio', mono(c.audio)]],
    release: c.found.release,
    action: btn('circle-arrow-up', 'Grab upgrade', 'pri'),
  }),
  missing: (c) => {
    const a = MISSING_ACTION[c.why];
    return { head: MISSING_WHY[c.why], note: c.detail, facts: [], action: a ? btn(a[0], a[1]) : '' };
  },
  downloading: (c) => {
    const j = X.job(c.job);
    return {
      head: `${j.stage} on ${j.node}`,
      job: j,
      facts: [['Rate', liveText('rate', j)], ['Left', liveText('eta', j)], ['Size', num(c.size)], ['Score', num(c.score)]],
      release: c.release,
      action: btn('pause', 'Pause'),
    };
  },
  converting: (c) => {
    const j = X.job(c.job);
    return {
      head: `Converting on ${c.node}`,
      job: j,
      facts: [['Rate', liveText('rate', j)], ['Left', liveText('eta', j)], ['Source', esc(c.from)], ['Settings', mono(c.settings)], ['Size', num(c.estSize)]],
      action: btn('pause', 'Pause Conversion'),
    };
  },
  failed: (c) => ({
    head: `At ${c.stage} · ${ago(c.at)}`,
    facts: [],
    note: c.cause,
    recovery: c.recovery,
    release: c.release,
    action: btn('search', 'Manual search', 'fail'),
  }),
  unaired: (c) => ({ head: `Airs ${md(c.airs)} · search starts then`, facts: [] }),
};
const detailOf = (e) => {
  const d = DETAIL[e.cell.state](e.cell);
  if (d.note && d.note.startsWith(d.head)) d.head = '';
  return d;
};

const bar = (j, label) => `<div class="bar gx-bar" role="progressbar" aria-valuemin="0" aria-valuemax="100" aria-label="${esc(label)}" data-live-progress="${j.id}" style="--p:${j.progress}"><span class="fill"></span></div>`;
const kv = (facts) => (facts.length ? `<dl class="kv gx-kv">${facts.map(([k, v]) => `<dt>${k}</dt><dd>${v}</dd>`).join('')}</dl>` : '');

const entryLabel = (e) => [e.code, e.edition.name, e.target.name].filter(Boolean).join(' · ');


const COMPACT = { met: 3, upgrade: 2, missing: 0, downloading: 2, converting: 2, failed: 0, unaired: 0 };

function cellCard(e) {
  const d = detailOf(e), s = e.cell.state;
  return `<td class="gx-cell gx-${s}">
    <div class="gx-cell-h">${chip(s)}${d.job ? `<span class="gx-pct num">${liveText('pct', d.job)}</span>` : ''}</div>
    ${d.head ? `<p class="gx-head ${s === 'failed' ? 'fail-t' : 't2'}">${esc(d.head)}</p>` : ''}
    ${d.job ? bar(d.job, `${entryLabel(e)} progress`) : ''}
    ${kv(d.facts.slice(0, COMPACT[s]))}
    ${d.note ? `<p class="gx-note-t">${esc(d.note)}</p>` : ''}
    ${d.recovery ? `<p class="gx-rec t3">${esc(d.recovery)}</p>` : ''}
    <div class="gx-act">${d.action || ''}<button class="btn ghost sm gx-more" data-gx-cell="${e.edition.id}|${e.target.id}|${e.item.id}" aria-label="All details for ${esc(entryLabel(e))}">${I('info')}Details</button></div>
  </td>`;
}

function targetHead(t) {
  return `<th scope="col"><div class="gx-th"><b>${esc(t.name)}</b><span class="t3">${srcIcon(t)}${esc(t.label)}</span></div></th>`;
}

function oneWide() {
  const all = cells(X.work, X.desc);
  const at = (ed, t) => all.find((e) => e.edition.id === ed.id && e.target.id === t.id);
  return `<table class="mx gx-mx">
    <caption class="sr">Edition by Target state for ${esc(X.work.title)}</caption>
    <thead><tr><th scope="col" class="mx-corner"><span>Edition</span>${I('move-right')}<span>Target</span></th>${X.targets.map(targetHead).join('')}</tr></thead>
    <tbody>${X.editions.map((ed) => `<tr><th scope="row" class="mx-ed"><b>${esc(ed.name)}</b><span class="t3">${esc(ed.note)}</span></th>${X.targets.map((t) => cellCard(at(ed, t))).join('')}</tr>`).join('')}
    <tr class="mx-add"><td colspan="${X.targets.length + 1}"><button class="btn ghost sm" data-toast="Editing Editions is not in this prototype">${I('plus')}Want another Edition</button><span class="t3">Each Edition is wanted at every Target in ${esc(profileOf(X.work).name)}</span></td></tr>
    </tbody></table>`;
}

function onePhone() {
  const all = cells(X.work, X.desc);
  return X.editions.map((ed) => {
    const row = all.filter((e) => e.edition.id === ed.id);
    return `<section class="psec gx-psec"><div class="psec-h"><h2>${esc(ed.name)}</h2><span class="t3">${esc(ed.note.split(' · ')[0])}</span></div>
      <ul class="glist">${row.map((e) => {
        const d = detailOf(e), s = e.cell.state, facts = d.facts.slice(0, COMPACT[s]);
        return `<li class="gx-pi gx-${s}">
          <div class="gx-pi-h"><b>${esc(e.target.name)}</b><span class="t3 gx-src">${srcIcon(e.target)}${esc(srcShort(e.target))}</span><span class="grow"></span>${chip(s)}</div>
          ${d.head ? `<p class="${s === 'failed' ? 'fail-t' : 't2'} gx-pi-l">${esc(d.head)}${d.job ? ` · <b class="num draft-t">${liveText('pct', d.job)}</b>` : ''}</p>` : ''}
          ${d.job ? bar(d.job, `${entryLabel(e)} progress`) : ''}
          ${facts.length ? `<p class="gx-pi-f t2">${facts.map(([k, v]) => `<span><span class="t3">${k}</span> ${v}</span>`).join('')}</p>` : ''}
          ${d.note ? `<p class="gx-note-t">${esc(d.note)}</p>` : ''}
          ${d.recovery ? `<p class="gx-rec t3">${esc(d.recovery)}</p>` : ''}
          ${d.action ? `<div class="gx-act">${d.action}</div>` : ''}
        </li>`;
      }).join('')}</ul></section>`;
  }).join('');
}


// Grouped by Edition so each group reads left to right along the sourcing chain: download, then derive.
function groups() {
  return X.editions.map((ed) => ({ key: ed.id, label: ed.name, cols: X.targets.map((t) => ({ ed, t, label: t.short, title: `${t.name} · ${t.label}` })) }));
}

const cellOf = (it, col) => it.item.cells[col.ed.id][col.t.id];
const isUnaired = (it) => cells(X.work, X.desc, [it]).every((e) => e.cell.state === 'unaired');

function squares(it, interactive) {
  return groups().map((g) => `<span class="gx-g">${g.cols.map((col) => {
    const c = cellOf(it, col), label = `${it.code} · ${col.ed.name} · ${col.t.name}: ${STATES[c.state].label}`;
    const inner = sq(c.state, c.job);
    return interactive ? `<button class="gx-sq" data-gx-cell="${col.ed.id}|${col.t.id}|${it.item.id}" aria-label="${esc(label)}">${inner}</button>` : `<span class="gx-sq" aria-hidden="true">${inner}</span>`;
  }).join('')}</span>`).join('');
}

const PHRASE = {
  failed: (c) => `<span class="fail-t">failed at ${esc(c.stage)}</span>`,
  upgrade: (c) => `upgrade <span class="num">${c.score} → ${c.found.score}</span>`,
  downloading: (c) => { const j = X.job(c.job); return `<span class="draft-t">${esc(j.stage.toLowerCase())} <span class="num">${liveText('pct', j)}</span></span>`; },
  converting: (c) => `<span class="draft-t">converting <span class="num">${liveText('pct', X.job(c.job))}</span></span>`,
};

function rowNote(it, compact) {
  const es = cells(X.work, X.desc, [it]).filter((e) => PHRASE[e.cell.state]).sort((a, b) => STATES[a.cell.state].rank - STATES[b.cell.state].rank);
  if (es.length) {
    const shown = es.slice(0, compact ? 1 : 2).map((e) => `<span class="gx-ph">${compact ? '' : `<b>${esc(e.edition.name)} ${esc(e.target.short)}</b> `}${PHRASE[e.cell.state](e.cell)}</span>`).join('');
    const more = es.length - (compact ? 1 : 2);
    return `${shown}${more > 0 ? `<span class="t3"> +${more}</span>` : ''}`;
  }
  const miss = cells(X.work, X.desc, [it]).filter((e) => e.cell.state === 'missing');
  if (!miss.length) return '';
  const whys = [...new Set(miss.map((e) => MISSING_WHY[e.cell.why].toLowerCase()))];
  return `<span class="t3">${miss.length} missing · ${esc(whys.join(', '))}</span>`;
}

function headRow() {
  return `<div class="gx-row gx-hrow" aria-hidden="true"><span></span><span></span>
    <span class="gx-cells">${groups().map((g) => `<span class="gx-g gx-gh"><b class="ell">${esc(g.label)}</b><span class="gx-gcols">${g.cols.map((c) => `<span class="gx-gc" title="${esc(c.title)}">${esc(c.label)}</span>`).join('')}</span></span>`).join('')}</span>
    <span class="gx-nh">Needs a look</span></div>`;
}

function runs(list) {
  const out = [];
  list.forEach((it) => {
    const un = isUnaired(it), last = out[out.length - 1];
    if (un && last && last.unaired) last.items.push(it);
    else out.push({ unaired: un, items: [it] });
  });
  return out;
}

const dated = (it) => it.item[X.desc.item.dated];

function unairedLine(run, phone) {
  const a = run.items[0], b = run.items[run.items.length - 1];
  const noun = X.desc.item.noun;
  const text = `${noun}s ${a.item.n}–${b.item.n} air ${md(dated(a))} to ${md(dated(b))}`;
  return `<button class="gx-un" data-gx-un aria-expanded="false">${I('calendar-clock')}<span>${esc(text)}</span><span class="grow"></span><span class="t3 gx-un-n">${plural(run.items.length, noun.toLowerCase())}</span>${I('chevron-down', 'gx-chev')}</button>
    <div class="gx-un-rows" hidden>${run.items.map((it) => (phone ? rowPhone(it) : rowWide(it))).join('')}</div>`;
}

function rowWide(it) {
  const un = isUnaired(it);
  return `<div class="gx-row${un ? ' gx-dim' : ''}" data-gx-item="${it.item.id}">
    <span class="gx-code mono">${esc(it.code)}</span>
    <span class="gx-ttl ell">${esc(it.label)}</span>
    <span class="gx-cells">${squares(it, true)}</span>
    <span class="gx-note ell">${un ? `<span class="t3">Airs ${md(dated(it))}</span>` : rowNote(it)}</span>
  </div>`;
}

function rowPhone(it) {
  const un = isUnaired(it);
  return `<button class="gx-prow${un ? ' gx-dim' : ''}" data-gx-row="${it.item.id}" aria-label="${esc(`${it.code} ${it.label}`)}, show all cells">
    <span class="gx-prow-t"><span class="gx-code mono">${esc(it.code)}</span><span class="ell">${esc(it.label)}</span>${I('chevron-right', 'gx-pchev')}</span>
    <span class="gx-prow-s"><span class="gx-cells">${squares(it, false)}</span><span class="gx-pnote ell">${un ? `<span class="t3">Airs ${md(dated(it))}</span>` : rowNote(it, true)}</span></span>
  </button>`;
}

function tally(c) {
  return STATE_ORDER.filter((s) => c[s]).map((s) => `<span class="gx-t">${sq(s)}<span class="num">${c[s]}</span><span class="t3">${esc((STATES[s].short || STATES[s].label).toLowerCase())}</span></span>`).join('');
}

function stack(c) {
  return `<span class="gx-stack" aria-hidden="true">${STATE_ORDER.filter((s) => c[s]).map((s) => `<span class="sq sq-${s}" style="flex:${c[s]}"></span>`).join('')}</span>`;
}

function startsOpen(level, list) {
  if (level.open === 'all') return true;
  if (level.open !== 'wanted') return false;
  return list.some((e) => !['met', 'unaired'].includes(e.cell.state));
}

function season(node, phone) {
  const level = X.desc.levels.find((l) => l.id === node.level);
  const its = items(X.work, X.desc).filter((it) => it.path.includes(node));
  const list = cells(X.work, X.desc, its), c = counts(list);
  const open = startsOpen(level, list);
  const label = level.label(node.n);
  const id = `gx-s-${node.level}-${node.n}`;
  const body = runs(its).map((r) => (r.unaired && r.items.length > 1 ? unairedLine(r, phone) : r.items.map((it) => (phone ? rowPhone(it) : rowWide(it))).join(''))).join('');
  const sum = summaryLine(c);
  return `<section class="gx-season${open ? '' : ' closed'}" aria-labelledby="${id}">
    <div class="gx-sh">
      <button class="gx-tog" data-gx-tog aria-expanded="${open}" aria-controls="${id}-b">${I('chevron-down', 'gx-chev')}<h3 id="${id}">${esc(label)}</h3><span class="t3 num gx-sh-n">${plural(its.length, X.desc.item.noun.toLowerCase())}</span></button>
      ${phone ? '' : stack(c)}
      <span class="gx-tally">${phone ? `<span class="t2">${esc(sum[0])}</span>${c.failed ? `<span class="gx-t">${sq('failed')}<span class="num fail-t">${c.failed}</span></span>` : ''}${c.upgrade ? `<span class="gx-t">${sq('upgrade')}<span class="num">${c.upgrade}</span></span>` : ''}${c.downloading + c.converting ? `<span class="gx-t">${sq('downloading', null)}<span class="num">${c.downloading + c.converting}</span></span>` : ''}` : tally(c)}</span>
      ${phone ? '' : `<button class="btn ghost sm" data-toast="Searching a whole ${esc(level.noun)} is not in this prototype">${I('search')}Search ${esc(level.short(node.n))}</button>`}
    </div>
    ${phone ? stack(c) : ''}
    <div class="gx-sb" id="${id}-b">${phone ? '' : headRow()}${body}</div>
  </section>`;
}

function sourcingStrip() {
  return `<p class="gx-srcs">${X.targets.map((t) => `<span>${srcIcon(t)}<b>${esc(t.name)}</b><span class="t3">${esc(t.label)}</span></span>`).join('')}</p>`;
}

function legend() {
  return `<span class="sx-legend">${STATE_ORDER.map((s) => `<span>${sq(s)}${esc(STATES[s].short || STATES[s].label)}</span>`).join('')}</span>`;
}

function manyWide() {
  return `${sourcingStrip()}
    <div class="gx-seasons">${levelNodes(X.work).map((n) => season(n, false)).join('')}</div>`;
}

function manyPhone() {
  const lv = X.desc.levels[0];
  return `<section class="psec gx-psec"><div class="psec-h"><h2>${esc(lv.noun)}s</h2><span class="t3">${esc(groups().map((g) => g.label).join(' · '))}</span></div>
    <p class="gx-pkey t3">Each ${esc(X.desc.item.noun.toLowerCase())} shows ${X.editions.length} Editions × ${X.targets.length} Targets, ${esc(X.targets.map((t) => t.short).join(', '))} left to right. Tap one for detail.</p>
    <div class="gx-seasons">${levelNodes(X.work).map((n) => season(n, true)).join('')}</div></section>`;
}


function setup(vctx) {
  const { work, desc, data } = vctx;
  X = { ...vctx, targets: targetsOf(work), editions: editionsOf(work), job: (id) => data.jobs.find((j) => j.id === id) };
}

function section(inner) {
  const many = X.desc.layout.items === 'many';
  const n = many ? items(X.work, X.desc).length : 0;
  return `<section class="wk-sec gx" aria-labelledby="gx-h"><div class="sec-h"><h2 id="gx-h">Editions and Targets</h2><span class="t3">Profile ${esc(profileOf(X.work).name)} · ${X.editions.length} Editions × ${X.targets.length} Targets${many ? ` × ${n} ${X.desc.item.noun}s` : ''}</span>${many ? `<span class="grow"></span>${legend()}` : ''}</div>${inner}</section>`;
}

export function wide(vctx) {
  setup(vctx);
  return section(X.desc.layout.items === 'one' ? oneWide() : manyWide());
}

export function phone(vctx) {
  setup(vctx);
  return `<div class="gx gx-phone">${X.desc.layout.items === 'one' ? onePhone() : manyPhone()}</div>`;
}

function popHtml(e, pinned) {
  const d = detailOf(e), s = e.cell.state;
  return `<div class="gx-pop-h"><span class="gx-pop-t">${e.code ? `<span class="mono">${esc(e.code)}</span> ` : ''}<b>${esc(e.edition.name)}</b> <span class="t3">·</span> <b>${esc(e.target.name)}</b></span>${pinned ? `<button class="ib gx-pop-x" data-gx-unpin aria-label="Close">${I('x')}</button>` : ''}</div>
    <div class="gx-cell-h">${chip(s)}<span class="${s === 'failed' ? 'fail-t' : 't2'} gx-pop-head">${esc(d.head)}</span>${d.job ? `<span class="gx-pct num">${liveText('pct', d.job)}</span>` : ''}</div>
    ${d.job ? bar(d.job, `${entryLabel(e)} progress`) : ''}
    ${kv(d.facts)}
    ${d.note ? `<p class="gx-note-t">${esc(d.note)}</p>` : ''}
    ${d.recovery ? `<p class="gx-rec t3">${esc(d.recovery)}</p>` : ''}
    ${d.release ? `<p class="mono gx-rel">${esc(d.release)}</p>` : ''}
    <p class="gx-pop-src t3">${srcIcon(e.target)}<span><b>${esc(e.target.name)}</b> · ${esc(e.target.label)}</span></p>
    ${d.action ? `<div class="gx-act">${d.action}</div>` : ''}`;
}

function sheetHtml(it) {
  const list = cells(X.work, X.desc, [it]);
  const block = groups().map((g) => `<p class="gx-sh-g">${esc(g.label)}</p><div class="sheet-grp gx-sgrp">${g.cols.map((col) => {
    const e = list.find((x) => x.edition.id === col.ed.id && x.target.id === col.t.id), d = detailOf(e), s = e.cell.state;
    const facts = d.facts.slice(0, COMPACT[s]);
    return `<div class="gx-si gx-${s}">
      <div class="gx-pi-h"><b>${esc(e.target.name)}</b><span class="t3 gx-src">${srcIcon(e.target)}${esc(srcShort(e.target))}</span><span class="grow"></span>${chip(s)}</div>
      ${d.head ? `<p class="${s === 'failed' ? 'fail-t' : 't2'} gx-pi-l">${esc(d.head)}${d.job ? ` · <b class="num draft-t">${liveText('pct', d.job)}</b>` : ''}</p>` : ''}
      ${d.job ? bar(d.job, `${entryLabel(e)} progress`) : ''}
      ${facts.length ? `<p class="gx-pi-f t2">${facts.map(([k, v]) => `<span><span class="t3">${k}</span> ${v}</span>`).join('')}</p>` : ''}
      ${d.note ? `<p class="gx-note-t">${esc(d.note)}</p>` : ''}
      ${d.recovery ? `<p class="gx-rec t3">${esc(d.recovery)}</p>` : ''}
      ${d.action ? `<div class="gx-act">${d.action}</div>` : ''}
    </div>`;
  }).join('')}</div>`).join('');
  return `<span class="grab" aria-hidden="true"></span><p class="sheet-t"><b>${esc(it.code)} · ${esc(it.label)}</b><span class="t3">${esc(summaryLine(counts(list)).join(' · '))}</span></p>
    <div class="gx-sheet-body">${block}</div><button class="sheet-x" data-close>Done</button>`;
}

export function wire(root, vctx) {
  setup(vctx);
  const byId = new Map(items(X.work, X.desc).map((it) => [it.item.id, it]));
  const entry = (key) => {
    const [ed, t, id] = key.split('|');
    return cells(X.work, X.desc, [byId.get(id)]).find((e) => e.edition.id === ed && e.target.id === t);
  };
  const icons = () => window.lucide && window.lucide.createIcons();

  root.addEventListener('click', (ev) => {
    const q = (s) => ev.target.closest(s);
    let el;
    if ((el = q('[data-gx-tog]'))) {
      const sec = el.closest('.gx-season');
      const closed = sec.classList.toggle('closed');
      el.setAttribute('aria-expanded', String(!closed));
      return;
    }
    if ((el = q('[data-gx-un]'))) {
      const rows = el.nextElementSibling, open = rows.hidden;
      rows.hidden = !open;
      el.setAttribute('aria-expanded', String(open));
      return;
    }
    if ((el = q('[data-gx-row]'))) {
      const s = root.querySelector('.sheet-wrap');
      s.querySelector('.sheet').innerHTML = sheetHtml(byId.get(el.dataset.gxRow));
      s.querySelector('.sheet').classList.add('gx-sheet');
      s.hidden = false;
      icons();
    }
  });

  if (root.classList.contains('lc-phone')) return;

  const pop = document.createElement('div');
  pop.className = 'gx-pop';
  pop.setAttribute('role', 'dialog');
  pop.hidden = true;
  root.append(pop);
  let pinned = null, hoverT = null;

  const place = (anchor) => {
    const r = anchor.getBoundingClientRect(), w = pop.offsetWidth, h = pop.offsetHeight;
    let left = r.left + r.width / 2 - w / 2;
    left = Math.max(12, Math.min(window.innerWidth - w - 12, left));
    let top = r.bottom + 8;
    if (top + h > window.innerHeight - 12) top = r.top - h - 8;
    pop.style.left = `${left}px`;
    pop.style.top = `${Math.max(12, top)}px`;
  };
  const show = (anchor, pin) => {
    clearTimeout(hoverT);
    root.querySelectorAll('.gx-sq.on').forEach((b) => b.classList.remove('on'));
    anchor.classList.add('on');
    pop.innerHTML = popHtml(entry(anchor.dataset.gxCell), pin);
    pop.classList.toggle('pinned', pin);
    pop.setAttribute('aria-label', anchor.getAttribute('aria-label') || 'Cell detail');
    pop.hidden = false;
    icons();
    place(anchor);
    pinned = pin ? anchor : null;
  };
  const hide = () => {
    pop.hidden = true;
    pinned = null;
    root.querySelectorAll('.gx-sq.on').forEach((b) => b.classList.remove('on'));
  };
  const later = () => { clearTimeout(hoverT); hoverT = setTimeout(() => { if (!pinned && !pop.matches(':hover')) hide(); }, 160); };

  root.addEventListener('mouseover', (ev) => {
    const sqb = ev.target.closest('.gx-sq[data-gx-cell]');
    if (sqb && !pinned) show(sqb, false);
    else if (ev.target.closest('.gx-pop')) clearTimeout(hoverT);
  });
  root.addEventListener('mouseout', (ev) => {
    if (pinned) return;
    if (ev.target.closest('.gx-sq[data-gx-cell], .gx-pop')) later();
  });
  root.addEventListener('focusin', (ev) => {
    const sqb = ev.target.closest('.gx-sq[data-gx-cell]');
    if (sqb && !pinned) show(sqb, false);
  });
  root.addEventListener('focusout', (ev) => {
    if (!pinned && ev.target.closest('.gx-sq[data-gx-cell]') && !(ev.relatedTarget && ev.relatedTarget.closest('.gx-pop'))) later();
  });
  root.addEventListener('click', (ev) => {
    const b = ev.target.closest('[data-gx-cell]');
    if (b) { if (pinned === b) return hide(); return show(b, true); }
    if (ev.target.closest('[data-gx-unpin]')) { const a = pinned; hide(); a && a.focus(); return; }
    if (pinned && !ev.target.closest('.gx-pop')) hide();
  });
  document.addEventListener('keydown', (ev) => { if (ev.key === 'Escape' && !pop.hidden) { const a = pinned; hide(); a && a.focus(); } });
  root.closest('#app')?.addEventListener('scroll', () => pinned && place(pinned), { passive: true });
  window.addEventListener('scroll', () => { if (pinned) place(pinned); else if (!pop.hidden) hide(); }, { passive: true });
}
