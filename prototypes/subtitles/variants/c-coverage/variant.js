// PROTOTYPE · Variant C, coverage first. Throwaway. Leads with what the library is missing and why; setup is the answer to a blocker.

let C, D, sh, root, S;
const esc = (s) => String(s ?? '').replace(/[&<>"]/g, (c) => ({ '&': '&amp;', '<': '&lt;', '>': '&gt;', '"': '&quot;' }[c]));
const I = (n, c = '') => `<i data-lucide="${n}" class="ic ${c}" aria-hidden="true"></i>`;
const fmt = (n) => Number(n).toLocaleString('en-US');
const P = (id) => D.providers[id];
const $ = (s) => root.querySelector(s);
const $$ = (s) => [...root.querySelectorAll(s)];

const LANG_CODE = { English: 'en', Spanish: 'es', French: 'fr', German: 'de', 'Portuguese (BR)': 'pt-BR', Japanese: 'ja', Korean: 'ko', Italian: 'it', Dutch: 'nl', Arabic: 'ar' };
const MOVIE_PROVIDERS = ['subdl', 'subsource'];

// Library-wide counts per scenario: one entry per (library file, Wanted Language). Invented, but they add up.
const COVER = {
  fresh: { final: 248, present: 138, embedded: 232, missing: 1384, nosync: 38, notneeded: 214,
    kinds: [{ k: 'Movies', total: 1410, met: 202, missing: 1208, nosync: 0 }, { k: 'Series', total: 630, met: 416, missing: 176, nosync: 38 }] },
  running: { final: 1104, present: 516, embedded: 292, missing: 96, nosync: 32, notneeded: 214,
    kinds: [{ k: 'Movies', total: 1410, met: 1318, missing: 70, nosync: 22 }, { k: 'Series', total: 630, met: 594, missing: 26, nosync: 10 }] },
  strained: { final: 1098, present: 490, embedded: 292, missing: 128, nosync: 32, notneeded: 214,
    kinds: [{ k: 'Movies', total: 1410, met: 1290, missing: 98, nosync: 22 }, { k: 'Series', total: 630, met: 590, missing: 30, nosync: 10 }] },
};
const BAR = [
  ['final', 'Final'], ['present', 'Present'], ['embedded', 'Embedded'], ['missing', 'Missing'], ['nosync', 'No candidate passed sync'],
];

// Profile numbers the consequence lines quote.
const PROF = {
  cinema: { files: 428, filesNote: '214 Works × 2 Targets', work: 'dune-2',
    sidecars: { 'w-en': 401, 'w-en-f': 58, 'w-es': 377, 'w-fr': 36 },
    audioHas: { English: 402, French: 387, Spanish: 61, German: 44, Italian: 39, Japanese: 12 },
    met: { fresh: { 'w-en': 0, 'w-en-f': 96, 'w-es': 0, 'w-fr': 0 }, running: { 'w-en': 412, 'w-en-f': 380, 'w-es': 377, 'w-fr': 36 }, strained: { 'w-en': 410, 'w-en-f': 378, 'w-es': 336, 'w-fr': 36 } } },
  tv: { files: 1284, filesNote: '61 Series, 1,284 episodes', work: 'severance',
    sidecars: { 't-en': 1190, 't-es': 22 },
    audioHas: { English: 1270, Spanish: 1240, French: 310, German: 288 },
    met: { fresh: { 't-en': 1010, 't-es': 4 }, running: { 't-en': 1236, 't-es': 22 }, strained: { 't-en': 1229, 't-es': 22 } } },
};

const SEV_FILE = (n) => `Severance.S02E0${n}.1080p.ATVP.WEB-DL.DDP5.1.H.264-FLUX.mkv`;

// ───────────────────────────── state

function initState(ctx) {
  const works = structuredClone(D.works);
  const dune = works['dune-2'], sev = works.severance;
  if (ctx.scenarioId === 'fresh') {
    for (const f of dune.files) for (const [wid, w] of Object.entries(f.wants)) {
      if (['final', 'present', 'nosync'].includes(w.state)) f.wants[wid] = { state: 'missing', note: 'No Subtitle Provider covers movies yet. Gestdown is TV only.', next: 'Searches the moment a SubDL or SubSource key works', nokey: true };
    }
    dune.files[1].wants['w-en-f'] = { state: 'missing', note: 'The forced track was inside the 2160p file. Conversion keeps only the main subtitle track. No Subtitle Provider covers movies yet.', next: 'Searches the moment a SubDL or SubSource key works', nokey: true };
    sev.episodes[4].wants['t-en'] = { state: 'missing', note: 'Gestdown has no English subtitle for this release yet. SubDL and SubSource are searched too once one has a key.', next: 'Gestdown again in 6 hours' };
  }
  if (ctx.scenarioId === 'strained') {
    dune.files[1].wants['w-en-f'] = { state: 'missing', note: 'Not searched since 06:12. SubSource rejected its key and SubDL is out of downloads.', next: 'First in line at 00:00 UTC: this file was imported today' };
    sev.episodes[6].wants['t-en'] = { state: 'missing', note: 'Imported 4 min ago. Gestdown has nothing yet. SubDL resumes at 00:00 UTC and SubSource needs a new key.', next: 'Gestdown again in 30 min' };
  }
  const prof = {};
  for (const [id, p] of Object.entries(D.profiles)) prof[id] = { saved: structuredClone(p.wanted), draft: structuredClone(p.wanted) };
  return {
    scen: ctx.scenarioId,
    order: [...ctx.scenario.order],
    status: ctx.scenario.status,
    keyUI: {},
    how: new Set(),
    replacing: new Set(),
    blkTab: 'subdl',
    resolved: [],
    works,
    sel: ctx.screen === 'series' ? { ep: 6, want: 't-en' } : { file: 'f-dune-1080', want: 'w-es' },
    confirmBad: false,
    busy: null,
    blocked: {},
    profile: 'cinema',
    prof,
    pop: null,
    modal: null,
  };
}

// ───────────────────────────── entry

export function render(ctx) {
  C = ctx; D = ctx.data; sh = ctx.shell; root = ctx.root;
  root.classList.add('vC');
  S = initState(ctx);
  const sev = D.works.severance;
  const chrome = {
    providers: { active: 'subtitles', crumbs: ['Settings', 'Subtitles'], phoneTitle: 'Subtitles' },
    profile: { active: 'profiles', crumbs: ['Settings', 'Profiles', 'Cinema'], phoneTitle: 'Profile', back: true },
    movie: { active: 'movies', crumbs: ['Library', 'Movies', 'Dune: Part Two'], phoneTitle: 'Subtitles', back: true },
    series: { active: 'series', crumbs: ['Library', 'Series', 'Severance', 'Season 2'], phoneTitle: 'Subtitles', back: true },
  }[ctx.screen];
  document.title = `${chrome.crumbs[chrome.crumbs.length - 1]} · Subtitles · SlipStream`;
  root.innerHTML = sh.AMBIENT + sh.page({ ...chrome, phone: ctx.phone, body: '<div class="vc" id="vc-main"></div>' }) + sh.OVERLAYS
    + '<div id="vc-ov"></div><p class="sr" aria-live="polite" id="vc-live"></p><input type="file" id="vc-file" accept=".srt,.ass,.ssa,.vtt,.sub" hidden>';
  if (ctx.screen === 'movie') sh.lightBy(root, D.works['dune-2']);
  if (ctx.screen === 'series') sh.lightBy(root, sev);
  paint();
  wire();
}

const VIEWS = { providers: viewProviders, profile: viewProfile, movie: viewMovie, series: viewSeries };

function paint() {
  const fk = document.activeElement && document.activeElement.dataset ? document.activeElement.dataset.fk : null;
  $('#vc-main').innerHTML = VIEWS[C.screen]();
  C.icons();
  if (fk) { const el = $(`[data-fk="${fk}"]`); if (el && !el.disabled) el.focus({ preventScroll: true }); }
}

const announce = (m) => { $('#vc-live').textContent = m; };
const toast = (m) => sh.toast(root, m);

// ───────────────────────────── providers: coverage, blockers, table

function coverage() {
  const c = COVER[S.scen];
  const met = c.final + c.present + c.embedded, total = met + c.missing + c.nosync;
  const pct = (n) => `${((n / total) * 100).toFixed(2)}%`;
  const searching = S.resolved.length ? `<span class="vc-searching">${sh.dot('draft')}searching now</span>` : '';
  return `<section class="vc-cov" aria-labelledby="cov-h">
    <p class="vc-eyebrow" id="cov-h">Library coverage</p>
    <p class="vc-sentence"><span>${fmt(met)} of ${fmt(total)} wants met</span><span class="s-sep" aria-hidden="true">·</span><span class="s-warn">${fmt(c.missing)} missing</span><span class="s-sep" aria-hidden="true">·</span><span class="s-warn">${fmt(c.nosync)} no candidate passed sync</span>${searching}</p>
    <div class="vc-bar" role="img" aria-label="${BAR.map(([k, l]) => `${l} ${fmt(c[k])}`).join(', ')}">${BAR.map(([k]) => `<span class="vc-seg vs-${k}" style="width:${pct(c[k])}"></span>`).join('')}</div>
    <div class="vc-cov-foot">
      <ul class="vc-legend">${BAR.map(([k, l]) => `<li><span class="vc-sw vs-${k}" aria-hidden="true"></span>${l}<b class="num">${fmt(c[k])}</b></li>`).join('')}<li class="t3"><span class="vc-sw vs-notneeded" aria-hidden="true"></span>Not needed<b class="num">${fmt(c.notneeded)}</b></li></ul>
      <ul class="vc-kinds">${c.kinds.map((k) => `<li><span class="vc-kind-l">${k.k}</span><span class="vc-mini" aria-hidden="true"><span class="vs-met" style="width:${(k.met / k.total) * 100}%"></span><span class="vs-missing" style="width:${(k.missing / k.total) * 100}%"></span><span class="vs-nosync" style="width:${(k.nosync / k.total) * 100}%"></span></span><span class="num t2">${fmt(k.met)} of ${fmt(k.total)}</span></li>`).join('')}</ul>
    </div>
  </section>`;
}

function blockers() {
  const out = [];
  if (!MOVIE_PROVIDERS.some((id) => S.status[id].key)) out.push({ kind: 'setup' });
  for (const id of S.order) {
    if (S.status[id].state === 'invalid') out.push({ kind: 'invalid', id });
    if (S.status[id].state === 'paused') out.push({ kind: 'paused', id });
  }
  return out;
}

function steps(id) {
  const p = P(id);
  return `<ol class="vc-steps">${p.steps.map((s, i) => `<li><span class="vc-n num">${i + 1}</span><span>${esc(s)}</span></li>`).join('')}</ol>
    <a class="btn sm vc-out" href="${p.keyPage}" target="_blank" rel="noopener">${I('external-link')}Open ${esc(p.keyPage.replace(/^https:\/\//, '').replace(/\/$/, ''))}</a>`;
}

let kbN = 0;
function keybox(id, { replace = false, label } = {}) {
  const p = P(id), ui = S.keyUI[id] || {};
  const n = ++kbN;
  return `<div class="vc-keybox" data-keybox="${id}">
    <label class="vc-key-l" for="kb-${id}-${n}">${label || `${p.name} API key`}</label>
    <div class="vc-key-row"><span class="vc-key-f">${I('key-round')}<input id="kb-${id}-${n}" class="vc-key-in mono" data-key-in="${id}" data-fk="kin-${id}-${replace ? 'r' : 'a'}" type="text" autocomplete="off" spellcheck="false" placeholder="Paste your ${p.name} key" value="${esc(ui.phase === 'fail' ? ui.value : '')}"></span>${replace ? `<button class="btn ghost sm" data-cancel-replace="${id}">Cancel</button>` : ''}</div>
    <p class="vc-key-stat" data-keystat="${id}">${keyStat(id)}</p>
  </div>`;
}

function keyStat(id) {
  const ui = S.keyUI[id] || {}, p = P(id);
  if (ui.phase === 'checking') return `<span class="vc-spin" aria-hidden="true"></span><span>Checking with ${p.name}…</span>`;
  if (ui.phase === 'fail') return `<span class="fail-t">${I('circle-x')}<span>${p.name} answered <b class="mono">${p.reject.status} ${esc(p.reject.body)}</b>. Copy the whole key from the ${p.name} page and paste it again.</span></span>`;
  return `<span class="t3">Checked the moment you paste it. ${esc(p.check.split('. ')[1] || '')}</span>`;
}

function never(id) {
  return `<p class="vc-never">${I('lock-keyhole')}<span>${esc(P(id).never)}</span></p>`;
}

function blockerCard(b) {
  if (b.kind === 'setup') {
    const tab = S.blkTab, p = P(tab);
    const left = MOVIE_PROVIDERS.map((id) => `<button role="radio" class="seg-o" data-blk-tab="${id}" data-fk="tab-${id}" aria-checked="${id === tab}">${P(id).name}<span class="seg-n">${id === 'subdl' ? '50 downloads a day' : '7,200 requests a day'}</span></button>`).join('');
    return `<article class="vc-blk vc-blk-setup" aria-labelledby="blk-setup">
      <div class="vc-blk-h"><span class="chip-warn">${I('triangle-alert')}Movies blocked</span><span class="t3">${fmt(COVER[S.scen].kinds[0].missing)} movie wants waiting</span></div>
      <h3 id="blk-setup">Movies get no subtitles until you add a SubDL or SubSource key. TV already runs on Gestdown.</h3>
      <p class="t2 vc-blk-sub">Both keys are free and either one unblocks movies. Adding both gives more candidates to rank.</p>
      <div class="seg-c vc-blk-tabs" role="radiogroup" aria-label="Provider to set up">${left}</div>
      <div class="vc-fix">
        <div class="vc-fix-steps"><p class="vc-fix-h">Get a ${p.name} key</p>${steps(tab)}</div>
        <div class="vc-fix-key">${keybox(tab)}${never(tab)}</div>
      </div>
    </article>`;
  }
  const p = P(b.id), s = S.status[b.id];
  if (b.kind === 'invalid') {
    return `<article class="vc-blk fail" aria-labelledby="blk-${b.id}">
      <div class="vc-blk-h"><span class="chip-fail">${I('circle-x')}Key rejected</span><span class="fail-dim">since ${esc(s.since)}, 15 h 28 min ago</span></div>
      <h3 id="blk-${b.id}">${p.name} stopped accepting its key at ${esc(s.since)} (${esc(s.error)}). 41 Spanish wants were searching it.</h3>
      <p class="fail-dim vc-blk-sub">Keys stop working when they are regenerated or the account is closed. ${p.name} replied <span class="mono">${p.reject.status} ${esc(p.reject.body)}</span> to every request since.</p>
      <div class="vc-fix">
        <div class="vc-fix-steps"><p class="vc-fix-h">Get a fresh key</p>${steps(b.id)}</div>
        <div class="vc-fix-key">${keybox(b.id, { label: `Replace the ${p.name} key` })}${never(b.id)}</div>
      </div>
    </article>`;
  }
  const others = S.order.filter((id) => id !== b.id && S.status[id].state === 'ready').map((id) => P(id).name);
  return `<article class="vc-blk warn" aria-labelledby="blk-${b.id}">
    <div class="vc-blk-h"><span class="chip-warn">${I('hourglass')}Out of downloads</span><span class="warn-dim">${s.used} of ${p.allowance.limit} used today on the free key</span></div>
    <h3 id="blk-${b.id}">${p.name} is out of downloads until ${esc(s.resumesAt)} (${esc(s.resumesIn)}). Missing wants on the newest imports go first when it resumes.</h3>
    <p class="warn-dim vc-blk-sub">Nothing to fix. It resumes on its own at the reset ${p.name} reported.${others.length ? ` ${others.join(' and ')} keep${others.length === 1 ? 's' : ''} searching meanwhile${others.length === 1 && others[0] === 'Gestdown' ? ', for TV only' : ''}.` : ''}</p>
  </article>`;
}

function blockersSection() {
  const bs = blockers();
  const res = S.resolved.map((r) => `<p class="vc-calm vc-res">${I('check')}<span>${esc(r)}</span></p>`).join('');
  const body = bs.length ? bs.map(blockerCard).join('')
    : `<p class="vc-calm">${sh.dot('draft')}<span><b>Nothing is holding subtitles back.</b> <span class="t3">${S.order.filter((id) => S.status[id].state === 'ready').length} Subtitle Providers ready · next sweep 03:00</span></span></p>`;
  return `<section class="vc-hold" aria-labelledby="hold-h">
    <div class="sec-h"><h2 id="hold-h">What’s holding subtitles back</h2>${bs.length ? `<span class="t3">${bs.length} blocker${bs.length > 1 ? 's' : ''}</span>` : ''}</div>
    <div class="vc-blks">${res}${body}</div>
  </section>`;
}

function stateLine(id) {
  const s = S.status[id], p = P(id);
  if (!p.needsKey) return '<span class="t3">Ready · no key</span>';
  return { ready: '<span class="t2">Ready</span>', paused: '<span class="warn-t">Paused</span>', invalid: '<span class="fail-t">Key rejected</span>', off: '<span class="t3">Needs a key</span>' }[s.state];
}

function keyCell(id) {
  const s = S.status[id], p = P(id);
  if (!p.needsKey) return '<span class="t3">No key needed</span>';
  if (!s.key) return `<button class="btn sm" data-addkey="${id}" data-fk="add-${id}">${I('key-round')}Add key</button>`;
  return `<span class="vc-mask mono${s.state === 'invalid' ? ' is-bad' : ''}">${esc(s.key)}</span><button class="btn ghost sm" data-replace="${id}" data-fk="rep-${id}">Replace</button>`;
}

function allowCell(id) {
  const s = S.status[id], p = P(id), a = p.allowance;
  if (!a) return '<span class="t3">No daily limit</span>';
  if (s.state === 'off') return '<span class="t3">Shown once a key works</span>';
  if (s.state === 'invalid') return '<span class="t3">Unknown while the key is rejected</span>';
  const left = Math.max(0, a.limit - s.used);
  const unit = a.unit;
  const txt = s.state === 'paused'
    ? `<span class="warn-t">0 of ${fmt(a.limit)} left</span><span class="warn-dim vc-allow-s">${unit} · back at ${esc(s.resumesAt)}</span>`
    : `<span class="t1">${fmt(left)} of ${fmt(a.limit)} left</span><span class="t3 vc-allow-s">${unit} · ${a.perMinute ? `${a.perMinute} a minute max` : `resets ${a.resets === 'midnight UTC' ? '00:00 UTC' : a.resets}`}</span>`;
  return `<span class="vc-allow"><span class="vc-meter${s.state === 'paused' ? ' is-out' : ''}" role="meter" aria-valuemin="0" aria-valuemax="${a.limit}" aria-valuenow="${left}" aria-label="${p.name} ${unit} left today"><span style="width:${(left / a.limit) * 100}%"></span></span><span class="vc-allow-t">${txt}</span></span>`;
}

function lastOk(id) {
  const s = S.status[id];
  if (s.state === 'invalid') return `<span class="fail-t">Failing since ${esc(s.since)}</span>`;
  if (s.state === 'off') return '<span class="t3">Never</span>';
  return `<span class="t2">${esc(s.lastOk)}</span>`;
}

function howStrip(id) {
  const p = P(id), s = S.status[id];
  if (!p.needsKey) {
    return `<div class="vc-how" id="how-${id}"><div class="vc-how-steps"><p class="vc-fix-h">No key to get</p><p class="t2">${esc(p.never)}</p><p class="t3">${esc(p.check)}</p></div>
      <div class="vc-how-key"><p class="t2">Covers ${esc(p.covers)}. Movies are searched on SubDL and SubSource only.</p></div></div>`;
  }
  const editing = !s.key || S.replacing.has(id);
  const cur = s.key && !editing
    ? `<div class="vc-cur"><p class="t2">${s.state === 'invalid' ? '<span class="fail-t">Rejected.</span> ' : ''}Key <span class="mono">${esc(s.key)}</span>${s.account ? ` · account <b>${esc(s.account)}</b>${s.plan ? ` · ${esc(s.plan)} plan` : ''}` : ''}</p>
        <div class="vc-cur-act"><button class="btn sm" data-replace="${id}">Replace key</button><button class="btn ghost sm vc-danger" data-remove="${id}">${I('trash-2')}Remove</button></div></div>`
    : keybox(id, { replace: !!s.key });
  return `<div class="vc-how" id="how-${id}">
    <div class="vc-how-steps"><p class="vc-fix-h">How to get a key</p>${steps(id)}</div>
    <div class="vc-how-key">${cur}${never(id)}<p class="t3 vc-check">${esc(p.check)}</p></div>
  </div>`;
}

function provRow(id, i) {
  const p = P(id), s = S.status[id], n = S.order.length, open = S.how.has(id);
  const tone = s.state === 'invalid' ? ' is-fail' : s.state === 'paused' ? ' is-warn' : '';
  return `<li class="vc-prov${tone}${open ? ' is-open' : ''}" data-prov="${id}">
    <div class="vc-prow">
      <div class="vc-c-rank">
        <span class="vc-grip" draggable="true" data-grip="${id}" title="Drag to reorder" aria-hidden="true">${I('grip-vertical')}</span>
        <span class="vc-rank num" aria-label="Rank ${i + 1}">${i + 1}</span>
        <span class="vc-ud"><button class="ib sm" data-move="${id}" data-dir="-1" data-fk="mv-${id}-up" ${i === 0 ? 'disabled' : ''} aria-label="Move ${p.name} up">${I('chevron-up')}</button><button class="ib sm" data-move="${id}" data-dir="1" data-fk="mv-${id}-dn" ${i === n - 1 ? 'disabled' : ''} aria-label="Move ${p.name} down">${I('chevron-down')}</button></span>
      </div>
      <div class="vc-c-prov"><b>${p.name}</b><span class="vc-pl">${stateLine(id)}<span class="vc-cov-in t3"> · ${esc(p.covers)}</span></span></div>
      <div class="vc-c-cov t2">${esc(p.covers)}</div>
      <div class="vc-c-key">${keyCell(id)}</div>
      <div class="vc-c-allow">${allowCell(id)}</div>
      <div class="vc-c-ok">${lastOk(id)}</div>
      <div class="vc-c-tog"><button class="btn ghost sm vc-howbtn" data-how="${id}" data-fk="how-${id}" aria-expanded="${open}" aria-controls="how-${id}">${p.needsKey ? 'How to get a key' : 'About'}${I('chevron-down', 'vc-chev')}</button></div>
    </div>
    ${open ? howStrip(id) : ''}
  </li>`;
}

function providersTable() {
  return `<section class="vc-provs" aria-labelledby="prov-h">
    <div class="sec-h"><h2 id="prov-h">Subtitle Providers</h2><span class="t3">The order ranks candidates after hash match, release group, and source and resolution.</span></div>
    <div class="vc-ptable">
      <div class="vc-prow vc-phead" aria-hidden="true"><span>Rank</span><span>Provider</span><span>Covers</span><span>Key</span><span>Allowance today</span><span>Last OK</span><span></span></div>
      <ol class="vc-plist" aria-label="Subtitle Providers in rank order">${S.order.map(provRow).join('')}</ol>
    </div>
  </section>`;
}

function viewProviders() {
  kbN = 0;
  return `<h1 class="sr">Subtitles</h1>${coverage()}${blockersSection()}${providersTable()}`;
}

// ───────────────────────────── key check (simulated)

let typeT;
function onKeyInput(inp, pasted) {
  const id = inp.dataset.keyIn;
  clearTimeout(typeT);
  const v = inp.value.trim();
  if (!v) { S.keyUI[id] = {}; return updateStat(id); }
  if (pasted) return checkKey(id, v);
  typeT = setTimeout(() => checkKey(id, v), 650);
}

function updateStat(id) {
  $$(`[data-keystat="${id}"]`).forEach((el) => { el.innerHTML = keyStat(id); });
  C.icons();
}

function checkKey(id, value) {
  const ui = { phase: 'checking', value };
  S.keyUI[id] = ui;
  updateStat(id);
  setTimeout(() => {
    if (S.keyUI[id] !== ui) return;
    const p = P(id);
    if (value.includes('bad') || value.length < 12) {
      ui.phase = 'fail';
      updateStat(id);
      announce(`${p.name} rejected the key: ${p.reject.status} ${p.reject.body}`);
      return;
    }
    const was = S.status[id];
    const masked = `${value.slice(0, 4)}…${value.slice(-4)}`;
    S.status[id] = { key: masked, state: 'ready', account: 'jackson', plan: 'Free', used: 0, lastOk: 'just now' };
    S.keyUI[id] = {};
    S.replacing.delete(id);
    const left = p.allowance.unit === 'downloads' ? `${p.allowance.limit} downloads` : `${fmt(p.allowance.limit)} requests`;
    let msg;
    if (was.state === 'invalid') msg = `${p.name} key works (account jackson). The 41 Spanish wants that were searching it are queued again. ${left} a day.`;
    else if (!was.key && !MOVIE_PROVIDERS.some((x) => x !== id && S.status[x].key)) msg = `${p.name} key works (account jackson, Free plan). Searching ${fmt(COVER[S.scen].kinds[0].missing)} movie wants now, newest imports first. ${left} a day.`;
    else msg = `${p.name} key works (account jackson, Free plan). ${left} a day.`;
    S.resolved = [msg];
    paint();
    announce(msg);
  }, 900);
}

// ───────────────────────────── profile

const kindTxt = (w) => (w.kind === 'forced' ? 'forced' : 'full');
function wantLabel(w) {
  const bits = [w.lang, kindTxt(w)];
  if (w.kind === 'full' && w.hi !== 'avoid') bits.push(D.hiLabels[w.hi]);
  if (w.when === 'noAudio') bits.push(`only without ${w.lang} audio`);
  return bits.join(' · ');
}
const laneSub = (w) => [D.kindLabels[w.kind], w.kind === 'full' ? D.hiLabels[w.hi] : null, w.when === 'noAudio' ? `only without ${w.lang} audio` : 'always'].filter(Boolean).join(' · ');
const shortName = (w) => (w.kind === 'forced' ? `${w.lang} forced` : w.lang);
const andList = (a) => (a.length < 2 ? a.join('') : `${a.slice(0, -1).join(', ')} and ${a[a.length - 1]}`);

function applicable(pid, w) {
  const pr = PROF[pid];
  return w.when === 'noAudio' ? pr.files - (pr.audioHas[w.lang] ?? 20) : pr.files;
}

function diff(pid) {
  const { saved, draft } = S.prof[pid];
  const out = [];
  for (const d of draft) {
    const s = saved.find((x) => x.id === d.id);
    if (!s) out.push({ t: 'add', w: d });
    else if (s.lang !== d.lang) out.push({ t: 'remove', w: s }, { t: 'add', w: d });
    else {
      if (s.kind !== d.kind) out.push({ t: 'kind', w: d, s });
      if (s.hi !== d.hi) out.push({ t: 'hi', w: d, s });
      if (s.when !== d.when) out.push({ t: 'when', w: d, s });
    }
  }
  for (const s of saved) if (!draft.find((x) => x.id === s.id)) out.push({ t: 'remove', w: s });
  return out;
}

function conseqLine(pid, c) {
  const pr = PROF[pid], name = D.profiles[pid].name;
  const sc = (w) => pr.sidecars[w.id] ?? 0;
  const L = c.w.lang;
  switch (c.t) {
    case 'add': return { ic: 'search', txt: `Adding ${shortName(c.w)} searches ${c.w.when === 'noAudio' ? `the ${fmt(applicable(pid, c.w))} files under ${name} that have no ${L} audio` : `all ${fmt(pr.files)} files under ${name} (${pr.filesNote})`}.` };
    case 'remove': return { ic: 'trash-2', warn: true, txt: `Dropping ${shortName(c.w)} deletes the ${fmt(sc(c.w))} ${L} Sidecars SlipStream fetched for it. Embedded tracks and your uploads stay.` };
    case 'kind': return c.w.kind === 'forced'
      ? { ic: 'replace', warn: true, txt: `${L} becomes forced only. Its ${fmt(sc(c.w))} full Sidecars are deleted and ${fmt(applicable(pid, c.w))} files are searched for forced ${L}.` }
      : { ic: 'replace', warn: true, txt: `${L} becomes full. Its ${fmt(sc(c.w))} forced Sidecars are deleted and ${fmt(applicable(pid, c.w))} files are searched for full ${L}.` };
    case 'hi': return { ic: 'ear', txt: `${L} now ${{ avoid: 'avoids', allow: 'allows', prefer: 'prefers' }[c.w.hi]} hearing-impaired subtitles. Sidecars stay; the ones that aren’t final get an upgrade search tonight.` };
    case 'when': return c.w.when === 'noAudio'
      ? { ic: 'volume-2', warn: true, txt: `${L} is wanted only without ${L} audio. The Sidecars on the ${fmt(pr.audioHas[L] ?? 20)} files with ${L} audio are deleted.` }
      : { ic: 'search', txt: `${L} is wanted always. ${fmt(pr.audioHas[L] ?? 20)} files with ${L} audio are searched too.` };
  }
  return null;
}

function previewFor(pid) {
  const w = S.works[PROF[pid].work], draft = S.prof[pid].draft;
  const on = [], skipped = [];
  for (const x of draft) (x.when === 'noAudio' && w.audio.includes(x.lang) ? skipped : on).push(x);
  const series = w.type === 'series';
  const where = series ? `the ${w.episodes.length} Season ${w.season} episodes` : 'the file';
  let s = `On ${w.title} this means ${on.length ? andList(on.map(shortName)) : 'no subtitles at all'}.`;
  if (skipped.length) s += ` ${andList([...new Set(skipped.map((x) => x.lang))])} ${skipped.length > 1 ? 'are' : 'is'} skipped: ${where} ${series ? 'have' : 'has'} ${andList([...new Set(skipped.map((x) => x.lang))])} audio.`;
  const d = diff(pid);
  const units = series ? w.episodes.length : w.files.length;
  const unitN = series ? 'episodes' : 'files';
  const local = [];
  for (const c of d) {
    if (c.t === 'add' || (c.t === 'when' && c.w.when === 'always') || c.t === 'kind') {
      if (c.w.when === 'noAudio' && w.audio.includes(c.w.lang)) continue;
      local.push(`searches ${units === 2 ? 'both' : `all ${units}`} ${unitN} for ${shortName(c.w)}`);
    }
    if (c.t === 'remove' || c.t === 'kind' || (c.t === 'when' && c.w.when === 'noAudio')) {
      const id = c.w.id;
      const items = series ? w.episodes.map((e) => e.wants[id]) : w.files.map((f) => f.wants[id]);
      const n = items.filter((x) => x && x.sub).length;
      if (n) local.push(`deletes ${n} ${c.w.lang} Sidecar${n > 1 ? 's' : ''}`);
    }
  }
  return { s, local: [...new Set(local)] };
}

function chip(pid, w) {
  const met = PROF[pid].met[S.scen][w.id];
  const app = applicable(pid, w);
  const isNew = met == null;
  const saved = S.prof[pid].saved.find((x) => x.id === w.id);
  const changed = saved && ['lang', 'kind', 'hi', 'when'].some((k) => saved[k] !== w[k]);
  const editing = S.pop && S.pop.id === w.id;
  return `<button class="vc-wchip${editing ? ' is-on' : ''}" data-want-edit="${w.id}" data-fk="wc-${w.id}" aria-haspopup="dialog" aria-expanded="${editing}">
    <span class="vc-wchip-l">${esc(wantLabel(w))}</span>
    <span class="vc-wchip-m">${isNew ? '<span class="t3">New · searched when you save</span>' : changed ? '<span class="warn-t">Changed · see what saving does</span>' : `<span class="vc-meter" aria-hidden="true"><span style="width:${Math.min(100, (met / app) * 100)}%"></span></span><span class="num t3">${fmt(met)} of ${fmt(app)} files</span>`}</span>
  </button>`;
}

function viewProfile() {
  const pid = S.profile, pr = D.profiles[pid], draft = S.prof[pid].draft;
  const w = S.works[PROF[pid].work];
  const pv = previewFor(pid);
  const d = diff(pid).map((c) => conseqLine(pid, c)).filter(Boolean);
  const nokey = S.scen === 'fresh' && pid === 'cinema';
  return `<h1 class="sr">Profile ${esc(pr.name)}</h1>
  <header class="vc-phead2">
    <div class="seg-c" role="radiogroup" aria-label="Profile">${Object.values(D.profiles).map((p) => `<button role="radio" class="seg-o" data-prof="${p.id}" data-fk="prof-${p.id}" aria-checked="${p.id === pid}">${esc(p.name)}</button>`).join('')}</div>
    <p class="t3">${esc(pr.contentType)} · <span class="num">${pr.works}</span> Works · Targets ${pr.targets.join(', ')}</p>
  </header>
  <div class="vc-pgrid">
    <section class="vc-wl" aria-labelledby="wl-h">
      <div class="sec-h"><h2 id="wl-h">Wanted Languages</h2><span class="t3">Every library file under ${esc(pr.name)} is checked for each one.</span></div>
      ${nokey ? `<p class="vc-note-warn">${I('triangle-alert')}<span>Movies have no Subtitle Provider yet, so these wants wait. <button class="lnk" data-go="providers"><b>Add a SubDL or SubSource key</b></button></span></p>` : ''}
      <div class="vc-wchips" id="vc-chips">${draft.map((x) => chip(pid, x)).join('')}<button class="vc-wchip vc-wadd" data-want-add data-fk="wadd">${I('plus')}<span>Add a Wanted Language</span></button></div>
      <p class="t3 vc-wl-note">Order doesn’t matter. A forced want is met only by a forced-only subtitle; hearing-impaired preference ranks candidates, it never rules one out.</p>
    </section>
    <aside class="vc-prev" aria-labelledby="prev-h">
      <div class="vc-prev-w">${sh.art(w, 'thumb-md')}<div><p class="vc-eyebrow" id="prev-h">Preview</p><b>${esc(w.title)}</b><p class="t3">${w.type === 'series' ? `Season ${w.season} · ${w.episodes.length} episodes` : `${w.files.length} files`} · audio ${andList(w.audio)}</p></div></div>
      <p class="vc-prev-s">${esc(pv.s)}</p>
      ${pv.local.length ? `<p class="vc-prev-c">${I('arrow-right')}<span>This change ${esc(andList(pv.local))} here.</span></p>` : ''}
    </aside>
  </div>
  ${d.length ? `<section class="vc-pending" aria-labelledby="pend-h">
    <div class="vc-pend-h"><h2 id="pend-h">What saving does</h2><span class="grow"></span><button class="btn ghost sm" data-discard>Discard</button><button class="btn pri sm" data-save>Save Profile</button></div>
    <ul class="vc-conseq">${d.map((c) => `<li class="${c.warn ? 'is-warn' : ''}">${I(c.ic)}<span>${esc(c.txt)}</span></li>`).join('')}</ul>
  </section>` : ''}`;
}

function popover() {
  if (!S.pop) return '';
  const pid = S.profile, w = S.prof[pid].draft.find((x) => x.id === S.pop.id);
  if (!w) return '';
  const seg = (field, opts) => `<div class="seg-c" role="radiogroup" aria-label="${field}">${opts.map(([v, l]) => `<button role="radio" class="seg-o" data-wf="${field}" data-v="${v}" data-fk="wf-${field}-${v}" aria-checked="${w[field] === v}">${l}</button>`).join('')}</div>`;
  const used = S.prof[pid].draft.filter((x) => x.id !== w.id && x.kind === w.kind).map((x) => x.lang);
  return `<div class="vc-popwrap">${C.phone ? '<div class="sheet-scrim" data-pop-close></div>' : ''}
    <div class="vc-pop" role="dialog" aria-label="${S.pop.isNew ? 'Add' : 'Edit'} Wanted Language">
      ${C.phone ? '<span class="grab" aria-hidden="true"></span>' : ''}
      <div class="vc-pop-h"><b>${S.pop.isNew ? 'New Wanted Language' : esc(wantLabel(w))}</b><button class="ib sm" data-pop-close aria-label="Close">${I('x')}</button></div>
      <label class="vc-f"><span>Language</span><select class="vc-sel" data-wf-sel="lang" data-fk="wf-lang">${D.languages.map((l) => `<option ${l === w.lang ? 'selected' : ''} ${used.includes(l) ? 'disabled' : ''}>${l}</option>`).join('')}</select></label>
      <div class="vc-f"><span>Type</span>${seg('kind', [['full', 'Full'], ['forced', 'Forced only']])}</div>
      <div class="vc-f"><span>Hearing impaired</span>${seg('hi', [['avoid', 'Avoid'], ['allow', 'Allow'], ['prefer', 'Prefer']])}</div>
      <div class="vc-f"><span>When</span>${seg('when', [['always', 'Always'], ['noAudio', `Only without ${esc(w.lang)} audio`]])}</div>
      <div class="vc-pop-f">${S.pop.isNew ? `<button class="btn ghost sm" data-pop-cancel>Cancel</button>` : `<button class="btn ghost sm vc-danger" data-want-remove="${w.id}">${I('trash-2')}Remove</button>`}<span class="grow"></span><button class="btn pri sm" data-pop-close data-fk="pop-done">${S.pop.isNew ? 'Add' : 'Done'}</button></div>
    </div></div>`;
}

function openPop(id, isNew, anchor) {
  S.pop = { id, isNew };
  paintProfileParts();
  if (!C.phone && anchor) placePop(anchor);
  const f = $('.vc-pop [data-fk="wf-lang"]');
  if (f) f.focus({ preventScroll: true });
}

function placePop(anchor) {
  const pop = $('.vc-pop');
  if (!pop) return;
  const r = anchor.getBoundingClientRect(), rr = root.getBoundingClientRect();
  pop.style.top = `${r.bottom - rr.top + 8}px`;
  pop.style.left = `${Math.max(12, Math.min(r.left - rr.left, rr.width - pop.offsetWidth - 12))}px`;
}

function paintProfileParts() {
  const fk = document.activeElement && document.activeElement.dataset ? document.activeElement.dataset.fk : null;
  const anchor = S.pop && $(`[data-want-edit="${S.pop.id}"]`);
  const pos = $('.vc-pop') ? { top: $('.vc-pop').style.top, left: $('.vc-pop').style.left } : null;
  $('#vc-main').innerHTML = viewProfile();
  $('#vc-ov').innerHTML = popover();
  C.icons();
  if (S.pop && !C.phone) {
    const pop = $('.vc-pop');
    if (pos && pos.top) { pop.style.top = pos.top; pop.style.left = pos.left; } else if (anchor) placePop(anchor);
    const a = $(`[data-want-edit="${S.pop.id}"]`);
    if (a && !(pos && pos.top)) placePop(a);
  }
  if (fk) { const el = $(`[data-fk="${fk}"]`); if (el) el.focus({ preventScroll: true }); }
}

function closePop() {
  const id = S.pop && S.pop.id;
  S.pop = null;
  paintProfileParts();
  const a = id && $(`[data-want-edit="${id}"]`);
  if (a) a.focus({ preventScroll: true });
}

// ───────────────────────────── a Work's subtitles

const stTone = (st) => D.subStates[st].tone;
function stateBadge(st, extra = '') {
  const s = D.subStates[st];
  return `<span class="vc-badge vb-${st}${extra}">${I(s.icon)}${esc(s.label)}</span>`;
}
const provName = (id) => (id === 'upload' ? 'Upload' : P(id).name);

function chipSub(x) {
  if (x.searching) return 'Searching now';
  if (x.sub && x.sub.provider === 'upload') return 'Uploaded by you · synced';
  if (x.sub) return `${x.sub.reasons.length ? D.reasonLabels[x.sub.reasons[0]].label : x.sub.byHand ? 'Picked by hand' : 'Best that synced'} · ${provName(x.sub.provider)}`;
  if (x.state === 'embedded') return x.track.split(' · ').slice(0, 2).join(' · ');
  if (x.state === 'nosync') return `${x.tried} tried · none lined up`;
  if (x.state === 'notneeded') return 'Has that audio';
  return x.nokey ? 'Waiting for a key' : x.next || 'Not found yet';
}

function wantsFor(work) {
  return S.prof[work.profile].saved;
}

function cellKey(a, b) { return `${a}|${b}`; }

function movieLanes(w) {
  const wants = wantsFor(w);
  return `<div class="vc-lanes" role="list">${wants.map((wt) => {
    const items = w.files.map((f) => ({ f, x: f.wants[wt.id] }));
    const met = items.filter(({ x }) => ['final', 'present', 'embedded'].includes(x.state)).length;
    const nn = items.filter(({ x }) => x.state === 'notneeded').length;
    const sum = nn === items.length ? 'Not needed here' : `Met on ${met} of ${items.length - nn} file${items.length - nn > 1 ? 's' : ''}`;
    return `<div class="vc-lane" role="listitem">
      <div class="vc-lane-l"><b>${esc(shortName(wt))}</b><span class="t3">${esc(laneSub(wt))}</span><span class="vc-lane-sum ${met < items.length - nn ? 'warn-t' : 't2'}">${sum}</span></div>
      <div class="vc-lane-chips">${items.map(({ f, x }) => {
        const on = S.sel.file === f.id && S.sel.want === wt.id;
        const busy = S.busy && S.busy.k === cellKey(f.id, wt.id);
        return `<button class="vc-fchip vf-${x.searching ? 'searching' : x.state}${on ? ' is-on' : ''}" data-pick="${f.id}" data-want="${wt.id}" data-fk="fc-${f.id}-${wt.id}" aria-pressed="${on}" aria-label="${esc(shortName(wt))} on the ${f.target} file: ${esc(D.subStates[x.state].label)}">
          <span class="vc-fchip-t">${esc(f.target)}${f.derived ? '<span class="t3"> · converted</span>' : ''}</span>
          <span class="vc-fchip-s">${busy ? `<span class="vc-spin" aria-hidden="true"></span>Timing sync…` : x.searching ? `${sh.dot('draft')}Searching` : `${I(D.subStates[x.state].icon)}${esc(D.subStates[x.state].label)}`}</span>
          <span class="vc-fchip-m">${esc(chipSub(x))}</span>
        </button>`;
      }).join('')}</div>
    </div>`;
  }).join('')}</div>`;
}

function seriesLanes(w) {
  const wants = wantsFor(w);
  return `<div class="vc-lanes vc-lanes-ep" role="list">${wants.map((wt) => {
    const xs = w.episodes.map((e) => e.wants[wt.id]);
    const nn = xs.filter((x) => x.state === 'notneeded').length;
    const met = xs.filter((x) => ['final', 'present', 'embedded'].includes(x.state)).length;
    const short = xs.length - nn - met;
    const sum = nn === xs.length ? `Not needed on all ${xs.length}: each has ${wt.lang} audio` : `Met on ${met} of ${xs.length - nn}${short ? ` · ${short} short` : ''}`;
    return `<div class="vc-lane" role="listitem">
      <div class="vc-lane-l"><b>${esc(shortName(wt))}</b><span class="t3">${esc(laneSub(wt))}</span><span class="vc-lane-sum ${short ? 'warn-t' : 't2'}">${sum}</span></div>
      <div class="vc-eps">${w.episodes.map((e) => {
        const x = e.wants[wt.id];
        const on = S.sel.ep === e.n && S.sel.want === wt.id;
        const busy = S.busy && S.busy.k === cellKey(e.n, wt.id);
        return `<button class="vc-ep vf-${x.searching ? 'searching' : x.state}${on ? ' is-on' : ''}" data-pick-ep="${e.n}" data-want="${wt.id}" data-fk="ep-${e.n}-${wt.id}" aria-pressed="${on}" aria-label="Episode ${e.n}, ${esc(e.title)}: ${esc(D.subStates[x.state].label)}" title="E${e.n} · ${esc(e.title)} · ${esc(D.subStates[x.state].label)}">
          <span class="vc-ep-n num">E${e.n}</span>${busy ? '<span class="vc-spin" aria-hidden="true"></span>' : x.searching ? sh.dot('draft') : I(D.subStates[x.state].icon)}
        </button>`;
      }).join('')}</div>
    </div>`;
  }).join('')}</div>`;
}

function legend() {
  return `<ul class="vc-slegend" aria-label="States">${Object.entries(D.subStates).map(([k, s]) => `<li title="${esc(s.means)}"><span class="vc-lg vf-${k}">${I(s.icon)}</span>${esc(s.label)}</li>`).join('')}</ul>`;
}

function selected() {
  const w = C.screen === 'series' ? S.works.severance : S.works['dune-2'];
  const wt = wantsFor(w).find((x) => x.id === S.sel.want);
  if (C.screen === 'series') {
    const e = w.episodes.find((x) => x.n === S.sel.ep);
    return { w, wt, x: e.wants[wt.id], key: cellKey(e.n, wt.id), where: `S02E0${e.n} · ${e.title}`, file: SEV_FILE(e.n), target: '1080p', set: (v) => { e.wants[wt.id] = v; }, ep: e };
  }
  const f = w.files.find((x) => x.id === S.sel.file);
  return { w, wt, x: f.wants[wt.id], key: cellKey(f.id, wt.id), where: `the ${f.target} file`, file: f.name, target: f.target, set: (v) => { f.wants[wt.id] = v; }, f };
}

function sidecarName(sel) {
  const base = sel.file.replace(/\.mkv$/, '');
  return `${base}.${LANG_CODE[sel.wt.lang] || 'xx'}${sel.wt.kind === 'forced' ? '.forced' : ''}.srt`;
}

function reasonMeans(r, sel) {
  if (r === 'group') return C.screen === 'series' ? 'Made for a release by FLUX, the group that made this file.' : D.reasonLabels.group.means;
  if (r === 'source') return C.screen === 'series' ? 'WEB-DL 1080p, like this file.' : D.reasonLabels.source.means;
  return D.reasonLabels[r].means;
}

function detail() {
  const sel = selected(), { x, wt } = sel;
  const st = D.subStates[x.state];
  const busy = S.busy && S.busy.k === sel.key;
  const title = `${esc(shortName(wt))} on ${esc(sel.where)}`;
  const acts = [];
  if (x.state !== 'notneeded') {
    const pri = ['missing', 'nosync'].includes(x.state);
    acts.push(`<button class="btn ${pri ? 'pri' : ''} sm" data-msearch data-fk="msearch">${I('list-filter')}Search by hand</button>`);
    acts.push(`<button class="btn sm" data-upload data-fk="upload">${I('upload')}Upload</button>`);
    if (x.sub) acts.push(`<button class="btn ghost sm vc-danger" data-bad data-fk="bad">${I('flag')}Mark bad</button>`);
  }
  let body = '';
  if (busy) {
    body = `<div class="vc-prog"><p>${esc(S.busy.label)}</p><div class="bar"><span class="fill" style="--p:${S.busy.p}"></span></div></div>`;
  } else if (x.sub) {
    const s = x.sub;
    const reasons = s.reasons.length
      ? s.reasons.map((r) => `<li><span class="vc-rchip vr-${D.reasonLabels[r].tone}">${esc(D.reasonLabels[r].label)}</span><span class="t2">${esc(reasonMeans(r, sel))}</span></li>`).join('')
      : `<li><span class="t2">No hash, group or source match. It was the best-ranked candidate that passed Timing sync.</span></li>`;
    body = `<dl class="vc-kv">
      <dt>Provider</dt><dd>${esc(provName(s.provider))}${s.provider === 'upload' ? ' <span class="t3">· uploaded by Jackson</span>' : ''}</dd>
      <dt>Release</dt><dd class="mono vc-wrap">${esc(s.release)}</dd>
      ${s.file ? `<dt>Sidecar</dt><dd class="mono vc-wrap t2">${esc(s.file)}</dd>` : ''}
      <dt>Match</dt><dd><ul class="vc-reasons">${reasons}</ul></dd>
      <dt>Timing sync</dt><dd>${esc(s.sync)}</dd>
      <dt>Made by</dt><dd>${s.by === 'ai' ? '<span class="warn-t">AI-translated</span> <span class="t3">· ranks below every human-made subtitle, so it is never final</span>' : 'Human'}</dd>
      ${x.next ? `<dt>Next</dt><dd class="t2">${esc(x.next)}</dd>` : ''}
    </dl>`;
  } else if (x.state === 'embedded') {
    body = `<dl class="vc-kv"><dt>Track</dt><dd class="mono">${esc(x.track)}</dd><dt>Download</dt><dd class="t2">Nothing is downloaded. The track inside the file meets this want.</dd></dl>`;
  } else if (x.state === 'notneeded') {
    body = `<dl class="vc-kv"><dt>Why</dt><dd>${esc(x.note || `Wanted only without ${wt.lang} audio. This episode has a ${wt.lang} audio track.`)}</dd></dl>
      <p class="t3 vc-det-foot">Want it anyway? <button class="lnk" data-go="profile"><b>Set it to Always on the Profile</b></button>.</p>`;
  } else {
    body = `<dl class="vc-kv">
      <dt>Why</dt><dd>${esc(x.note || 'Not found yet.')}${x.nokey ? ` <button class="lnk" data-go="providers"><b>Add a key</b></button>` : ''}</dd>
      ${x.next ? `<dt>Next</dt><dd class="t2">${x.searching ? sh.dot('draft') + ' ' : ''}${esc(x.next)}</dd>` : ''}
    </dl>`;
  }
  const confirm = S.confirmBad && x.sub ? `<div class="vc-confirm" role="alertdialog" aria-labelledby="cb-t">
      <p id="cb-t">Mark it bad? The Sidecar is deleted, this subtitle is blocked for ${esc(sel.where)}, and SlipStream searches again.</p>
      <div><button class="btn sm" data-bad-cancel>Keep it</button><button class="btn fail sm" data-bad-go data-fk="bad-go">${I('flag')}Mark bad</button></div>
    </div>` : '';
  return `<section class="vc-detail" id="vc-detail" aria-labelledby="det-h">
    <div class="vc-det-h">
      ${x.searching ? `<span class="vc-badge vb-searching">${sh.dot('draft')}Searching</span>` : stateBadge(x.state)}
      <div class="vc-det-t"><h3 id="det-h">${title}</h3><p class="mono t3 vc-wrap">${esc(sel.file)}</p></div>
      <div class="vc-det-act">${busy ? '' : acts.join('')}</div>
    </div>
    <p class="t2 vc-det-means">${x.searching ? 'Marked bad just now. Searching the next candidate.' : esc(st.means)}</p>
    ${confirm}
    ${body}
  </section>`;
}

function workHead(w) {
  const series = w.type === 'series';
  const list = series ? w.episodes.flatMap((e) => Object.values(e.wants)) : w.files.flatMap((f) => Object.values(f.wants));
  const want = list.filter((x) => x.state !== 'notneeded');
  const met = want.filter((x) => ['final', 'present', 'embedded'].includes(x.state) && !x.searching).length;
  const miss = want.filter((x) => x.state === 'missing').length, ns = want.filter((x) => x.state === 'nosync').length;
  let meta = series
    ? `<span class="num">${w.year}</span> · Series · Season ${w.season} · ${w.episodes.length} episodes · Profile ${esc(D.profiles[w.profile].name)}`
    : `<span class="num">${w.year}</span> · Movie · ${esc(w.edition)} Edition · ${w.files.length} files · Profile ${esc(D.profiles[w.profile].name)}`;
  if (C.phone) meta = series ? `<span class="num">${w.year}</span> · Season ${w.season} · ${w.episodes.length} episodes` : `<span class="num">${w.year}</span> · ${w.files.length} files · Profile ${esc(D.profiles[w.profile].name)}`;
  return `<header class="vc-whead">
    <div class="wk-bd" aria-hidden="true"><img src="${w.art.backdrop}" alt="" onerror="this.remove()"></div>
    ${sh.art(w, C.phone ? 'poster-sm' : 'poster-md')}
    <div class="wk-id">
      <h1 class="wk-logo-t">${esc(w.title)}</h1>
      <p class="meta t2">${meta}</p>
      <p class="wk-state"><span class="pill-met">${I('captions')}${met} of ${want.length} wants met</span>${miss ? `<span class="vc-pill-warn">${miss} missing</span>` : ''}${ns ? `<span class="vc-pill-warn">${ns} no candidate passed sync</span>` : ''}</p>
    </div>
  </header>`;
}

function subsSection(w) {
  const series = w.type === 'series';
  const wants = wantsFor(w);
  const nokey = S.scen === 'fresh' && !series;
  return `<section class="vc-subs" aria-labelledby="subs-h">
    <div class="sec-h"><h2 id="subs-h">Subtitles</h2><span class="t3">${wants.length} Wanted Languages × ${series ? `${w.episodes.length} episodes` : `${w.files.length} files`}</span><span class="grow"></span><button class="btn ghost sm" data-go="profile">${I('sliders-horizontal')}Wanted Languages</button></div>
    ${nokey ? `<p class="vc-note-warn">${I('triangle-alert')}<span>No Subtitle Provider covers movies yet. Gestdown is TV only. <button class="lnk" data-go="providers"><b>Add a SubDL or SubSource key</b></button></span></p>` : ''}
    ${series ? seriesLanes(w) : movieLanes(w)}
    ${legend()}
    ${detail()}
  </section>`;
}

function viewMovie() { const w = S.works['dune-2']; return workHead(w) + subsSection(w); }
function viewSeries() { const w = S.works.severance; return workHead(w) + subsSection(w); }

// ───────────────────────────── upload and mark bad (simulated)

function runSteps(stepsArr, done) {
  let i = 0;
  const tick = () => {
    if (i >= stepsArr.length) return done();
    const [label, ms] = stepsArr[i++];
    S.busy = { ...S.busy, label, p: i / stepsArr.length };
    paint();
    setTimeout(tick, ms);
  };
  tick();
}

function doUpload(fileName) {
  const sel = selected();
  S.busy = { k: sel.key, label: '', p: 0 };
  S.confirmBad = false;
  runSteps([[`Reading ${fileName}`, 500], ['Timing sync against the Speech Map', 1300]], () => {
    S.busy = null;
    sel.set({ state: 'present', sub: { provider: 'upload', release: fileName, file: sidecarName(sel), reasons: [], sync: '+310 ms offset, alignment 88%. Passed the gate.', by: 'human', at: D.now.toISOString() }, next: 'Upgrade search tonight at 03:00' });
    paint();
    toast(`${shortName(sel.wt)} uploaded and synced for ${sel.where}`);
  });
}

function doMarkBad() {
  const sel = selected();
  const rel = sel.x.sub.release, prov = provName(sel.x.sub.provider);
  (S.blocked[sel.key] ||= []).push(rel);
  const searching = MOVIE_PROVIDERS.concat(C.screen === 'series' ? ['gestdown'] : []).filter((id) => S.status[id].state === 'ready').map((id) => P(id).name);
  sel.set({ state: 'missing', searching: true, note: `Marked bad just now. The ${prov} subtitle (${rel}) is blocked for this file.`, next: searching.length ? `Searching ${andList(searching)} now` : 'No Subtitle Provider can search right now' });
  S.confirmBad = false;
  paint();
  toast('Sidecar deleted and blocked for this file. Searching again');
}

// ───────────────────────────── manual search

function candsFor(sel) {
  if (C.screen === 'series') {
    const n = sel.ep.n, ns = sel.x.state === 'nosync';
    return [
      { id: 's1', provider: 'gestdown', release: `Severance.S02E0${n}.1080p.ATVP.WEB-DL.DDP5.1.H.264-FLUX`, lang: 'English', hi: true, reasons: ['group', 'hi'], downloads: 4210, rating: null, uploader: 'Addic7ed · elderman', status: ns ? 'failed-sync' : 'new', syncNote: 'Tried at 03:00. Drifted past the framerate limit.' },
      { id: 's2', provider: 'subdl', release: `Severance.S02E0${n}.1080p.WEB.H264-SuccessfulCrab`, lang: 'English', hi: false, reasons: ['source'], downloads: 9120, rating: 8.8, uploader: 'GoldenBeard', status: ns ? 'failed-sync' : 'new', syncNote: 'Tried at 03:10. Drifted past the framerate limit.' },
      { id: 's3', provider: 'subsource', release: `Severance.S02E0${n}.720p.WEB.h264-ETHEL`, lang: 'English', hi: true, reasons: ['hi'], downloads: 1880, rating: 8.1, uploader: 'explosiveskull', status: 'new' },
      { id: 's4', provider: 'subdl', release: `Severance.S02E0${n}.1080p.WEB.H264-SuccessfulCrab`, lang: 'English', hi: false, reasons: ['source', 'ai'], downloads: 310, rating: null, uploader: 'SubDL AI', status: 'new' },
    ];
  }
  return structuredClone(D.candidates);
}

// Rank by the rule in #20 with the live provider order, so reordering providers is felt here too.
function rankCands(list) {
  const ai = (c) => c.reasons.some((r) => r === 'ai' || r === 'machine');
  const key = (c) => [c.status === 'blocked' ? 1 : 0, ai(c) ? 1 : 0, c.reasons.includes('hash') ? 0 : 1, c.reasons.includes('group') ? 0 : 1, c.reasons.includes('source') ? 0 : 1, S.order.indexOf(c.provider), -c.downloads];
  return [...list].sort((a, b) => { const ka = key(a), kb = key(b); for (let i = 0; i < ka.length; i++) if (ka[i] !== kb[i]) return ka[i] - kb[i]; return 0; });
}

function provAvail(id) {
  const s = S.status[id];
  if (C.screen === 'movie' && id === 'gestdown') return 'tv';
  if (!s.key && P(id).needsKey) return 'nokey';
  return s.state === 'paused' ? 'paused' : s.state === 'invalid' ? 'invalid' : 'ok';
}

function openSearch() {
  const sel = selected();
  const blocked = S.blocked[sel.key] || [];
  const all = candsFor(sel).map((c) => (blocked.includes(c.release) ? { ...c, status: 'blocked', syncNote: 'Marked bad just now by Jackson.' } : c));
  const skipped = [...new Set(all.map((c) => c.provider))].filter((id) => ['nokey', 'invalid'].includes(provAvail(id)));
  const list = rankCands(all.filter((c) => !skipped.includes(c.provider)));
  S.modal = { list, skipped, selId: list[0] && list[0].id, run: null, view: 'list', sel };
  paintModal();
  const f = $('.vc-ms-row[aria-selected="true"]') || $('.vc-modal [data-close-modal]');
  if (f) f.focus({ preventScroll: true });
}

function closeSearch() {
  S.modal = null;
  $('#vc-ov').innerHTML = '';
  const b = $('[data-msearch]');
  if (b) b.focus({ preventScroll: true });
}

function rchips(c) {
  return c.reasons.map((r) => `<span class="vc-rchip vr-${D.reasonLabels[r].tone}">${esc(D.reasonLabels[r].label)}</span>`).join('')
    + (c.status === 'failed-sync' ? `<span class="vc-rchip vr-hist">${I('audio-waveform')}Failed Timing sync</span>` : '')
    + (c.status === 'blocked' ? `<span class="vc-rchip vr-block">${I('ban')}Blocked</span>` : '')
    + (provAvail(c.provider) === 'paused' ? `<span class="vc-rchip vr-hist">${I('hourglass')}${P(c.provider).name} paused</span>` : '');
}

function msRow(c, i) {
  const on = S.modal.selId === c.id;
  return `<li class="vc-ms-row${on ? ' is-on' : ''}${c.status === 'blocked' ? ' is-blocked' : ''}" role="option" tabindex="${on ? 0 : -1}" aria-selected="${on}" data-cand="${c.id}">
    <span class="vc-ms-rank num">${i + 1}</span>
    <span class="vc-ms-main"><span class="mono vc-ms-rel">${esc(c.release)}</span><span class="t3 vc-ms-meta">${esc(P(c.provider).name)} · ${esc(c.lang)} · <span class="num">${fmt(c.downloads)}</span> downloads${c.rating != null ? ` · <span class="num">${c.rating}</span>` : ''}</span><span class="vc-ms-chips">${rchips(c)}</span></span>
  </li>`;
}

function msDetail(c, rank) {
  const sel = S.modal.sel, run = S.modal.run, p = P(c.provider);
  const ai = c.reasons.includes('ai') || c.reasons.includes('machine');
  const why = c.reasons.filter((r) => r !== 'hi');
  const whyList = why.length
    ? why.map((r) => `<li><span class="vc-rchip vr-${D.reasonLabels[r].tone}">${esc(D.reasonLabels[r].label)}</span><span class="t2">${esc(reasonMeans(r, sel))}</span></li>`).join('')
    : '';
  const orderTxt = `${p.name} is ${['1st', '2nd', '3rd'][S.order.indexOf(c.provider)]} in your provider order`;
  const rest = why.length ? '' : `<li><span class="t2">No hash, group or source match. It ranks here because ${orderTxt}, then on its ${fmt(c.downloads)} downloads.</span></li>`;
  const hiLine = c.hi ? `<li><span class="vc-rchip vr-info">Hearing impaired</span><span class="t2">${esc(D.reasonLabels.hi.means)} This want ${sel.wt.hi === 'avoid' ? 'avoids HI, so it ranks below a plain one with the same match' : sel.wt.hi === 'prefer' ? 'prefers HI' : 'allows HI'}.</span></li>` : '';
  const hist = [];
  if (c.status === 'failed-sync') hist.push(`<li class="warn-t">${I('audio-waveform')}<span>Failed Timing sync before. ${esc(c.syncNote)} It will most likely fail again.</span></li>`);
  if (c.status === 'blocked') hist.push(`<li class="fail-t">${I('ban')}<span>Blocked for this file. ${esc(c.syncNote)}</span></li>`);
  if (!hist.length) hist.push('<li class="t3"><span>Never tried on this file.</span></li>');
  const av = provAvail(c.provider);
  let action;
  if (run) {
    const stp = [[`Download from ${p.name}`, 'download'], ['Timing sync against the Speech Map', 'audio-waveform'], ['Save as Sidecar', 'captions']];
    const skipSync = c.reasons.includes('hash');
    action = `<ol class="vc-run">${stp.map(([l, ic], i) => {
      const stt = i < run.step ? 'done' : i === run.step ? (run.fail ? 'fail' : 'cur') : 'todo';
      const label = i === 1 && skipSync ? 'Timing sync skipped: hash match' : l;
      return `<li class="vr-${stt}">${stt === 'done' ? I('check') : stt === 'cur' ? '<span class="vc-spin" aria-hidden="true"></span>' : stt === 'fail' ? I('circle-x') : I(ic)}<span>${label}</span></li>`;
    }).join('')}</ol>
    <div class="bar${run.fail ? ' is-fail' : ''}"><span class="fill" style="--p:${run.fail ? 1 : run.step / 3}"></span></div>
    ${run.msg ? `<p class="${run.fail ? 'fail-t' : 't2'} vc-run-msg" role="status">${esc(run.msg)}</p>` : ''}
    ${run.fail ? `<div class="vc-ms-act"><button class="btn sm" data-run-reset>Back to candidates</button></div>` : ''}`;
  } else if (c.status === 'blocked') {
    action = `<div class="vc-ms-act"><button class="btn pri" disabled>${I('ban')}Blocked for this file</button><button class="btn ghost sm" data-unblock="${c.id}">Unblock</button></div>`;
  } else if (av === 'paused') {
    action = `<div class="vc-ms-act"><button class="btn pri" data-queue="${c.id}">${I('hourglass')}Use at ${esc(S.status[c.provider].resumesAt)}</button></div><p class="t3 vc-ms-note">${p.name} is out of downloads until ${esc(S.status[c.provider].resumesAt)}. Queued picks go first when it resumes.</p>`;
  } else {
    action = `<div class="vc-ms-act"><button class="btn pri" data-use="${c.id}" data-fk="use">${I('captions')}Use this subtitle</button></div>
      <p class="t3 vc-ms-note">${c.reasons.includes('hash') ? 'Downloads it and saves it as a Sidecar. Hash matches skip Timing sync.' : 'Downloads it, then runs Timing sync against the Speech Map. If it fails the gate it is dropped and nothing changes.'}${ai ? ' AI-translated, so it is never final.' : ''}</p>`;
  }
  return `<div class="vc-ms-det-in">
    ${C.phone ? `<button class="btn ghost sm vc-ms-back" data-ms-back>${I('chevron-left')}All candidates</button>` : ''}
    <p class="vc-eyebrow">Candidate ${rank + 1} of ${S.modal.list.length}</p>
    <h3 class="mono vc-wrap">${esc(c.release)}</h3>
    <p class="t2">${esc(p.name)} · uploaded by <b>${esc(c.uploader)}</b> · ${esc(c.lang)}</p>
    <dl class="vc-ms-stats"><div><dt>Downloads</dt><dd class="num">${fmt(c.downloads)}</dd></div><div><dt>Rating</dt><dd class="num">${c.rating ?? '<span class="t3">none</span>'}</dd></div><div><dt>Made by</dt><dd>${ai ? '<span class="warn-t">AI</span>' : 'Human'}</dd></div></dl>
    <h4>Why it ranks here</h4>
    <ul class="vc-reasons">${whyList}${rest}${hiLine}${ai ? '' : ''}</ul>
    <h4>On this file before</h4>
    <ul class="vc-hist">${hist.join('')}</ul>
    <div class="vc-ms-go">${action}</div>
  </div>`;
}

function paintModal() {
  const m = S.modal;
  if (!m) return;
  const sel = m.sel;
  const fk = document.activeElement && document.activeElement.dataset ? document.activeElement.dataset.fk || (document.activeElement.dataset.cand ? `cand-${document.activeElement.dataset.cand}` : null) : null;
  const c = m.list.find((x) => x.id === m.selId);
  const notices = [];
  for (const id of m.skipped) notices.push(provAvail(id) === 'invalid' ? `${P(id).name} wasn’t searched: it rejected its key at ${S.status[id].since}.` : `${P(id).name} wasn’t searched: it has no key yet.`);
  if (C.screen === 'movie') notices.push('Gestdown wasn’t searched: it covers TV only.');
  const empty = !m.list.length;
  const phoneDetail = C.phone && m.view === 'detail';
  $('#vc-ov').innerHTML = `<div class="vc-modal-wrap"><div class="sheet-scrim" data-close-modal></div>
    <div class="vc-modal" role="dialog" aria-modal="true" aria-labelledby="ms-h">
      <header class="vc-ms-h"><div class="vc-ms-ht"><p class="vc-eyebrow">Search by hand</p><h2 id="ms-h">${esc(shortName(sel.wt))} for ${esc(sel.w.title)}${C.screen === 'series' ? ` S02E0${sel.ep.n}` : ''}</h2><p class="t3">${esc(sel.target)} file · <span class="mono">${esc(sel.file)}</span></p></div><button class="ib" data-close-modal aria-label="Close search">${I('x')}</button></header>
      <p class="vc-ms-rule t3">Ranked by hash match, then release group, then source and resolution, then your provider order, then downloads and rating. AI-translated subtitles rank below every human-made one.</p>
      ${notices.length ? `<p class="vc-ms-notice">${I('info')}<span>${notices.map(esc).join(' ')}</span></p>` : ''}
      ${empty ? `<div class="vc-ms-empty"><b>No Subtitle Provider can search ${C.screen === 'movie' ? 'movies' : 'this'} yet.</b><p class="t2">Add a SubDL or SubSource key and this list fills in. You can still upload a subtitle you have.</p><div class="vc-ms-act"><button class="btn pri" data-go="providers">${I('key-round')}Add a key</button><button class="btn" data-upload-ms>${I('upload')}Upload instead</button></div></div>`
      : `<div class="vc-ms-body${phoneDetail ? ' is-detail' : ''}">
        <ol class="vc-ms-list" role="listbox" aria-label="Candidates, best first">${m.list.map(msRow).join('')}</ol>
        <section class="vc-ms-det" aria-label="Selected candidate">${c ? msDetail(c, m.list.indexOf(c)) : ''}</section>
      </div>`}
    </div></div>`;
  C.icons();
  if (fk) { const el = fk.startsWith('cand-') ? $(`[data-cand="${fk.slice(5)}"]`) : $(`.vc-modal [data-fk="${fk}"]`); if (el) el.focus({ preventScroll: true }); }
}

function useCand(id) {
  const m = S.modal, c = m.list.find((x) => x.id === id), sel = m.sel, p = P(c.provider);
  const willFail = c.status === 'failed-sync';
  const hash = c.reasons.includes('hash');
  m.run = { step: 0, fail: false, msg: '' };
  paintModal();
  const go = (fn, ms) => setTimeout(() => { if (S.modal !== m) return; fn(); paintModal(); }, ms);
  go(() => { m.run.step = 1; }, 750);
  if (willFail) {
    go(() => { m.run.fail = true; m.run.msg = `Alignment ${c.provider === 'subdl' ? '38' : '43'}%, the gate is 70%. Dropped; the Sidecar you had is unchanged.`; c.syncNote = `Tried just now. ${m.run.msg.split('.')[0]}.`; }, 2200);
    return;
  }
  go(() => { m.run.step = 2; m.run.msg = hash ? 'Hash match, so no sync needed.' : 'Offset +1.84 s, alignment 91%. Passed the gate.'; }, 2200);
  go(() => {
    m.run.step = 3;
    const ai = c.reasons.includes('ai') || c.reasons.includes('machine');
    const final = !ai && (c.reasons.includes('hash') || c.reasons.includes('group'));
    sel.set({ state: final ? 'final' : 'present', sub: { provider: c.provider, release: c.release, file: sidecarName(sel), reasons: c.reasons.filter((r) => r !== 'hi'), sync: hash ? 'skipped, hash match' : '+1.84 s offset, alignment 91%. Passed the gate.', by: ai ? 'ai' : 'human', byHand: true, at: D.now.toISOString() }, next: final ? null : 'Upgrade search tonight at 03:00' });
  }, 2900);
  setTimeout(() => {
    if (S.modal !== m) return;
    S.modal = null;
    $('#vc-ov').innerHTML = '';
    paint();
    toast(`${shortName(sel.wt)} from ${p.name} synced and saved for ${sel.where}`);
    const d = $('#vc-detail');
    if (d) d.scrollIntoView({ block: 'nearest', behavior: 'smooth' });
  }, 3700);
}

// ───────────────────────────── wiring

function move(id, dir) {
  const i = S.order.indexOf(id), j = i + dir;
  if (j < 0 || j >= S.order.length) return;
  [S.order[i], S.order[j]] = [S.order[j], S.order[i]];
  paint();
  announce(`${P(id).name} is now rank ${j + 1}`);
}

function wire() {
  root.addEventListener('click', (e) => {
    const q = (s) => e.target.closest(s);
    let el;
    if ((el = q('[data-go]'))) { if (S.modal) S.modal = null; return C.go(el.dataset.go); }
    if ((el = q('[data-move]'))) return move(el.dataset.move, +el.dataset.dir);
    if ((el = q('[data-how]'))) { const id = el.dataset.how; S.how.has(id) ? S.how.delete(id) : S.how.add(id); return paint(); }
    if ((el = q('[data-addkey]'))) { const id = el.dataset.addkey; S.how.add(id); paint(); const f = $(`[data-key-in="${id}"]`); return f && f.focus(); }
    if ((el = q('[data-replace]'))) {
      const id = el.dataset.replace; S.how.add(id); S.replacing.add(id); S.keyUI[id] = {}; paint();
      const f = $(`.vc-how [data-key-in="${id}"]`); return f && f.focus();
    }
    if ((el = q('[data-cancel-replace]'))) { const id = el.dataset.cancelReplace; S.replacing.delete(id); S.keyUI[id] = {}; return paint(); }
    if ((el = q('[data-remove]'))) {
      const id = el.dataset.remove, p = P(id);
      S.status[id] = { key: null, state: 'off' };
      S.replacing.delete(id);
      S.resolved = [];
      paint();
      return toast(`${p.name} key removed. ${p.name} is no longer searched`);
    }
    if ((el = q('[data-blk-tab]'))) { S.blkTab = el.dataset.blkTab; return paint(); }

    if ((el = q('[data-prof]'))) { S.profile = el.dataset.prof; S.pop = null; $('#vc-ov').innerHTML = ''; return paint(); }
    if ((el = q('[data-want-edit]'))) {
      if (S.pop && S.pop.id === el.dataset.wantEdit) return closePop();
      return openPop(el.dataset.wantEdit, false, el);
    }
    if ((el = q('[data-want-add]'))) {
      const d = S.prof[S.profile].draft;
      const lang = D.languages.find((l) => !d.some((x) => x.lang === l)) || 'German';
      const id = `new-${Date.now()}`;
      d.push({ id, lang, kind: 'full', hi: 'avoid', when: 'always' });
      paintProfileParts();
      return openPop(id, true, $(`[data-want-edit="${id}"]`));
    }
    if ((el = q('[data-wf]'))) {
      const w = S.prof[S.profile].draft.find((x) => x.id === S.pop.id);
      w[el.dataset.wf] = el.dataset.v;
      return paintProfileParts();
    }
    if ((el = q('[data-pop-cancel]'))) {
      const d = S.prof[S.profile].draft; const i = d.findIndex((x) => x.id === S.pop.id); if (i >= 0) d.splice(i, 1);
      S.pop = null; paintProfileParts(); const a = $('[data-want-add]'); return a && a.focus();
    }
    if ((el = q('[data-want-remove]'))) {
      const d = S.prof[S.profile].draft; const i = d.findIndex((x) => x.id === el.dataset.wantRemove); d.splice(i, 1);
      S.pop = null; paintProfileParts(); const a = $('[data-want-add]'); return a && a.focus();
    }
    if ((el = q('[data-pop-close]'))) return closePop();
    if ((el = q('[data-discard]'))) { const p = S.prof[S.profile]; p.draft = structuredClone(p.saved); S.pop = null; $('#vc-ov').innerHTML = ''; return paint(); }
    if ((el = q('[data-save]'))) {
      const p = S.prof[S.profile];
      const lines = diff(S.profile).length;
      p.saved = structuredClone(p.draft);
      for (const w of p.saved) if (PROF[S.profile].met[S.scen][w.id] == null) PROF[S.profile].met[S.scen][w.id] = 0;
      S.pop = null; $('#vc-ov').innerHTML = '';
      paint();
      return toast(`${D.profiles[S.profile].name} saved. ${lines} change${lines > 1 ? 's' : ''} queued`);
    }

    if ((el = q('[data-pick]'))) { S.sel = { file: el.dataset.pick, want: el.dataset.want }; S.confirmBad = false; return paint(); }
    if ((el = q('[data-pick-ep]'))) { S.sel = { ep: +el.dataset.pickEp, want: el.dataset.want }; S.confirmBad = false; return paint(); }
    if ((el = q('[data-msearch]'))) return openSearch();
    if ((el = q('[data-upload]')) || (el = q('[data-upload-ms]'))) { if (S.modal) closeSearch(); return $('#vc-file').click(); }
    if ((el = q('[data-bad]'))) { S.confirmBad = true; paint(); const b = $('[data-bad-go]'); return b && b.focus(); }
    if ((el = q('[data-bad-cancel]'))) { S.confirmBad = false; return paint(); }
    if ((el = q('[data-bad-go]'))) return doMarkBad();

    if ((el = q('[data-close-modal]'))) return closeSearch();
    if ((el = q('[data-cand]'))) { if (S.modal.run && !S.modal.run.fail) return; S.modal.selId = el.dataset.cand; S.modal.run = null; S.modal.view = 'detail'; paintModal(); const r = $(`[data-cand="${el.dataset.cand}"]`); return r && !C.phone && r.focus({ preventScroll: true }); }
    if ((el = q('[data-ms-back]'))) { S.modal.view = 'list'; return paintModal(); }
    if ((el = q('[data-use]'))) return useCand(el.dataset.use);
    if ((el = q('[data-run-reset]'))) { const c = S.modal.list.find((x) => x.id === S.modal.selId); c.status = 'failed-sync'; S.modal.run = null; return paintModal(); }
    if ((el = q('[data-unblock]'))) { const c = S.modal.list.find((x) => x.id === el.dataset.unblock); c.status = 'new'; S.modal.list = rankCands(S.modal.list); return paintModal(); }
    if ((el = q('[data-queue]'))) { const c = S.modal.list.find((x) => x.id === el.dataset.queue); closeSearch(); return toast(`Queued. ${P(c.provider).name} downloads it first at ${S.status[c.provider].resumesAt}`); }
    if (S.pop && !q('.vc-pop')) return closePop();
  });

  root.addEventListener('input', (e) => {
    const t = e.target;
    if (t.matches('[data-key-in]')) return onKeyInput(t, e.inputType === 'insertFromPaste');
  });
  root.addEventListener('change', (e) => {
    const t = e.target;
    if (t.matches('[data-wf-sel]')) { const w = S.prof[S.profile].draft.find((x) => x.id === S.pop.id); w.lang = t.value; return paintProfileParts(); }
    if (t.id === 'vc-file' && t.files[0]) { const n = t.files[0].name; t.value = ''; return doUpload(n); }
  });

  root.addEventListener('keydown', (e) => {
    if ((S.modal || S.pop) && (e.key === 'ArrowLeft' || e.key === 'ArrowRight')) e.stopPropagation();
    if (e.key === 'Escape') {
      if (S.modal) { e.preventDefault(); return closeSearch(); }
      if (S.pop) { e.preventDefault(); return closePop(); }
      if (S.confirmBad) { S.confirmBad = false; return paint(); }
    }
    if (e.target.matches('[data-cand]') && (e.key === 'ArrowDown' || e.key === 'ArrowUp')) {
      e.preventDefault();
      const l = S.modal.list, i = l.findIndex((x) => x.id === S.modal.selId);
      const n = l[(i + (e.key === 'ArrowDown' ? 1 : -1) + l.length) % l.length];
      S.modal.selId = n.id; S.modal.run = null; paintModal();
      $(`[data-cand="${n.id}"]`).focus({ preventScroll: false });
    }
    if (S.modal && e.key === 'Tab') {
      const f = $$('.vc-modal button:not([disabled]), .vc-modal [tabindex="0"], .vc-modal a');
      if (!f.length) return;
      if (e.shiftKey && document.activeElement === f[0]) { e.preventDefault(); f[f.length - 1].focus(); }
      else if (!e.shiftKey && document.activeElement === f[f.length - 1]) { e.preventDefault(); f[0].focus(); }
    }
  });

  let dragId = null;
  root.addEventListener('dragstart', (e) => {
    const g = e.target.closest && e.target.closest('[data-grip]');
    if (!g) return;
    dragId = g.dataset.grip;
    const row = g.closest('.vc-prov');
    e.dataTransfer.effectAllowed = 'move';
    e.dataTransfer.setData('text/plain', dragId);
    try { e.dataTransfer.setDragImage(row, 24, 20); } catch { /* older engines */ }
    row.classList.add('is-drag');
  });
  root.addEventListener('dragover', (e) => {
    const row = e.target.closest && e.target.closest('.vc-prov');
    if (!dragId || !row) return;
    e.preventDefault();
    $$('.vc-prov.is-over').forEach((r) => r.classList.remove('is-over'));
    if (row.dataset.prov !== dragId) row.classList.add('is-over');
  });
  root.addEventListener('drop', (e) => {
    const row = e.target.closest && e.target.closest('.vc-prov');
    if (!dragId || !row) return;
    e.preventDefault();
    const to = S.order.indexOf(row.dataset.prov);
    S.order.splice(S.order.indexOf(dragId), 1);
    S.order.splice(to, 0, dragId);
    const id = dragId; dragId = null;
    paint();
    announce(`${P(id).name} is now rank ${S.order.indexOf(id) + 1}`);
  });
  root.addEventListener('dragend', () => { dragId = null; $$('.vc-prov.is-drag, .vc-prov.is-over').forEach((r) => r.classList.remove('is-drag', 'is-over')); });
}
