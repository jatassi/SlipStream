// PROTOTYPE · Variant B · Ranked list, guided sheet. Throwaway: one thing at a time, setup in a sheet.

let C, D, S, root, phone, scen;
let st;

const esc = (s) => String(s ?? '').replace(/[&<>"]/g, (c) => ({ '&': '&amp;', '<': '&lt;', '>': '&gt;', '"': '&quot;' }[c]));
const I = (n, c = '') => `<i data-lucide="${n}" class="ic ${c}" aria-hidden="true"></i>`;
const P = (id) => D.providers[id];
const ORD = ['1st', '2nd', '3rd', '4th'];
const CODE = { English: 'en', Spanish: 'es', French: 'fr', German: 'de', 'Portuguese (BR)': 'pt-BR', Japanese: 'ja', Korean: 'ko', Italian: 'it', Dutch: 'nl', Arabic: 'ar' };
const clone = (o) => structuredClone(o);
const NOW = '21:40';

const HI_TXT = { avoid: 'hearing impaired avoided', allow: 'hearing impaired allowed', prefer: 'hearing impaired preferred' };
const HI_SHORT = { avoid: 'HI avoided', allow: 'HI allowed', prefer: 'HI preferred' };
const KIND_TXT = { full: 'full', forced: 'forced only' };
const REASON_SHORT = { hash: 'hash match', group: 'same group', source: 'same source and resolution', copied: 'copied from the 2160p HDR file' };

// ───────────────────────────── entry

export function render(ctx) {
  C = ctx; D = ctx.data; S = ctx.shell; root = ctx.root; phone = ctx.phone; scen = ctx.scenario;
  root.classList.add('vB');
  st = {
    sheet: null, pop: null,
    profile: ctx.screen === 'series' ? 'tv' : 'cinema',
    saved: { cinema: clone(D.profiles.cinema.wanted), tv: clone(D.profiles.tv.wanted) },
    draft: null,
    files: buildFiles(),
    open: new Set(),
    blocked: {},
    ep: null,
    upTarget: null,
  };
  st.draft = clone(st.saved[st.profile]);
  if (ctx.screen === 'movie') st.open.add('f-dune-1080|w-es');
  if (ctx.screen === 'series') { st.ep = 6; st.open.add('ep-6|t-en'); }

  const pages = {
    providers: { active: 'subtitles', crumbs: ['Settings', 'Subtitles'], phoneTitle: 'Subtitles' },
    profile: { active: 'profiles', crumbs: ['Settings', 'Profiles', 'Wanted Languages'], phoneTitle: 'Profiles' },
    movie: { active: 'movies', crumbs: ['Library', 'Movies', 'Dune: Part Two', 'Subtitles'], phoneTitle: 'Subtitles' },
    series: { active: 'series', crumbs: ['Library', 'Series', 'Severance', 'Season 2'], phoneTitle: 'Season 2' },
  };
  const pg = pages[ctx.screen];
  root.innerHTML = S.AMBIENT
    + S.page({ ...pg, body: `<div id="vb-main" class="vb-${ctx.screen}"></div>`, phone, back: true })
    + S.OVERLAYS
    + `<div id="vb-ov"></div><div id="vb-pop"></div><p id="vb-live" class="sr" aria-live="polite"></p>
       <input type="file" id="vb-file" accept=".srt,.ass,.ssa,.vtt,.sub" hidden>`;
  if (ctx.screen === 'movie') S.lightBy(root, D.works['dune-2']);
  if (ctx.screen === 'series') S.lightBy(root, D.works.severance);
  wire();
  paint();
}

function paint(focusSel) {
  const v = { providers: providersView, profile: profileView, movie: movieView, series: seriesView }[C.screen];
  root.querySelector('#vb-main').innerHTML = v();
  C.icons();
  if (focusSel) root.querySelector(focusSel)?.focus({ preventScroll: true });
}

const announce = (msg) => { const l = root.querySelector('#vb-live'); l.textContent = ''; setTimeout(() => { l.textContent = msg; }, 30); };
const toast = (msg) => S.toast(root, msg);

// ───────────────────────────── providers

function stateLine(pid) {
  const p = P(pid), s = scen.status[pid], a = p.allowance;
  if (s.state === 'off') {
    return { tone: 'off', text: 'Needs your free key', sub: p.covers === 'TV only' ? '' : `Not searched until it has one. Getting a key takes about a minute.` };
  }
  if (s.state === 'invalid') {
    return { tone: 'fail', text: `Key rejected at ${s.since}: ${s.error}`, sub: `${p.name} answered ${p.reject.status} when SlipStream used the key. Paste a new one to resume.` };
  }
  if (s.state === 'paused') {
    const others = scen.order.filter((id) => id !== pid && scen.status[id].state === 'ready').map((id) => P(id).name);
    return { tone: 'warn', text: `Paused until ${s.resumesAt}, out of downloads`, sub: `Resumes in ${s.resumesIn}.${others.length ? ` ${others.join(' and ')} carr${others.length > 1 ? 'y' : 'ies'} on.` : ''}`, bar: { p: 0, warn: true, label: `0 of ${a.limit} ${a.unit} left` } };
  }
  if (!a) return { tone: 'ok', text: 'Ready · no key and no daily limit', sub: `Last answered ${s.lastOk}. Its order only matters for TV.` };
  const left = a.limit - (s.used || 0);
  if (pid === 'subdl') return { tone: 'ok', text: `Ready · ${left} of ${a.limit} downloads left today, resets at 00:00 UTC`, sub: `Last answered ${s.lastOk}.`, bar: { p: left / a.limit, label: `${left} of ${a.limit} downloads left` } };
  return { tone: 'ok', text: `Ready · ${S.fmt(left)} of ${S.fmt(a.limit)} requests left over the last 24 hours`, sub: `Up to ${a.perMinute} a minute. Last answered ${s.lastOk}.`, bar: { p: left / a.limit, label: `${left} of ${a.limit} requests left` } };
}

function provSummary() {
  const ids = scen.order, s = scen.status;
  const ready = ids.filter((id) => s[id].state === 'ready');
  if (ready.length === ids.length) return 'All three Subtitle Providers are ready.';
  const movieReady = ready.some((id) => P(id).covers !== 'TV only');
  const issues = ids.filter((id) => s[id].state !== 'ready').map((id) => {
    const n = P(id).name, x = s[id].state;
    return x === 'off' ? `${n} needs your free key.` : x === 'paused' ? `${n} resumes at ${s[id].resumesAt}.` : `${n} needs a new key.`;
  });
  if (!movieReady && ids.every((id) => s[id].state !== 'paused' && s[id].state !== 'invalid')) return 'Gestdown already finds TV subtitles with no key. Movies need one free key, from SubDL or SubSource.';
  if (!movieReady) return `Only Gestdown is ready, so only TV gets subtitles right now. ${issues.join(' ')}`;
  return `${ready.length} of ${ids.length} ready. ${issues.join(' ')}`;
}

function provRow(pid, i) {
  const p = P(pid), s = scen.status[pid], L = stateLine(pid), n = scen.order.length;
  const keyBtn = !p.needsKey ? ''
    : !s.key ? `<button class="btn pri sm" data-key="${pid}">${I('key-round')}Add key</button>`
    : `<button class="btn sm${s.state === 'invalid' ? ' pri' : ''}" data-key="${pid}">${I('key-round')}Replace key</button>`;
  const more = p.needsKey && s.key ? `<button class="ib sm" data-pmenu="${pid}" aria-haspopup="menu" aria-label="More for ${p.name}">${I('ellipsis')}</button>` : '';
  const dotK = { ok: 'dot-draft', warn: 'dot-warn', fail: 'dot-fail', off: 'dot-idle' }[L.tone];
  return `<li class="rk rk-${L.tone}" data-pid="${pid}">
    <span class="rk-n" aria-hidden="true">${i + 1}</span>
    <button class="rk-h" data-handle="${pid}" aria-label="Reorder ${p.name}, ${ORD[i]} of ${n}. Use arrow keys." title="Drag to reorder">${I('grip-vertical')}</button>
    <div class="rk-b">
      <p class="rk-name"><b>${p.name}</b>${p.covers === 'TV only' ? '<span class="vb-tag">TV only</span>' : `<span class="t3">${p.covers}</span>`}${s.key ? `<span class="mono t3 rk-key">${I('key-round')}${esc(s.key)}</span>` : ''}</p>
      <p class="rk-s"><span class="dot ${dotK}" aria-hidden="true"></span><span>${esc(L.text)}</span></p>
      ${L.sub ? `<p class="rk-sub t3">${esc(L.sub)}</p>` : ''}
      ${L.bar ? `<div class="al${L.bar.warn ? ' al-warn' : ''}" role="meter" aria-valuemin="0" aria-valuemax="1" aria-valuenow="${L.bar.p.toFixed(2)}" aria-label="${L.bar.label}"><span style="width:${Math.max(L.bar.p * 100, 0)}%"></span></div>` : ''}
    </div>
    <div class="rk-a">${keyBtn}${more}
      <span class="rk-mv"><button class="ib sm" data-mv="-1" data-pid="${pid}" aria-label="Move ${p.name} up" ${i === 0 ? 'disabled' : ''}>${I('chevron-up')}</button><button class="ib sm" data-mv="1" data-pid="${pid}" aria-label="Move ${p.name} down" ${i === n - 1 ? 'disabled' : ''}>${I('chevron-down')}</button></span>
    </div>
  </li>`;
}

function providersView() {
  return `<header class="vb-ph">${phone ? '' : '<h1>Subtitles</h1>'}<p class="vb-lede">${esc(provSummary())}</p></header>
    <section class="vb-sec" aria-labelledby="rk-h">
      <div class="vb-sec-h"><h2 id="rk-h">Subtitle Providers</h2><span class="t3">${scen.order.length} in order</span></div>
      <p class="vb-hint">Candidates rank by hash match, then same group, then same source and resolution. After that, the provider higher on this list wins.</p>
      <ol class="rk-list" aria-label="Subtitle Provider order">${scen.order.map(provRow).join('')}</ol>
      <p class="vb-foot t3">${I('info')}<span>Which languages SlipStream looks for is set per Profile. <button class="lnk vb-a" data-go="profile">Edit Wanted Languages</button></span></p>
    </section>`;
}

function move(pid, d, focusDir) {
  const o = scen.order, i = o.indexOf(pid), j = i + d;
  if (j < 0 || j >= o.length) return;
  [o[i], o[j]] = [o[j], o[i]];
  paint(focusDir ? `[data-mv="${d}"][data-pid="${pid}"]:not([disabled]), [data-handle="${pid}"]` : `[data-handle="${pid}"]`);
  announce(`${P(pid).name} moved to ${ORD[j]}.`);
}

function dragStart(e, h) {
  const row = h.closest('.rk'), list = row.parentElement;
  e.preventDefault();
  h.setPointerCapture(e.pointerId);
  let y0 = e.clientY;
  const before = scen.order.join();
  row.classList.add('drag');
  const renumber = () => [...list.children].forEach((li, k) => { li.querySelector('.rk-n').textContent = k + 1; });
  const mv = (ev) => {
    const prev = row.previousElementSibling, next = row.nextElementSibling;
    let dy = ev.clientY - y0;
    if (next && dy > next.offsetHeight / 2) { list.insertBefore(next, row); y0 += next.offsetHeight; renumber(); }
    else if (prev && dy < -prev.offsetHeight / 2) { list.insertBefore(row, prev); y0 -= prev.offsetHeight; renumber(); }
    dy = ev.clientY - y0;
    row.style.transform = `translateY(${dy}px)`;
  };
  const up = () => {
    h.removeEventListener('pointermove', mv); h.removeEventListener('pointerup', up); h.removeEventListener('pointercancel', up);
    scen.order = [...list.children].map((li) => li.dataset.pid);
    paint(`[data-handle="${row.dataset.pid}"]`);
    if (scen.order.join() !== before) announce(`${P(row.dataset.pid).name} moved to ${ORD[scen.order.indexOf(row.dataset.pid)]}.`);
  };
  h.addEventListener('pointermove', mv); h.addEventListener('pointerup', up); h.addEventListener('pointercancel', up);
}

// ───────────────────────────── key sheet

const mask = (v) => `${v.slice(0, 3)}…${v.slice(-4)}`;

function keyStepsFor(p) {
  if (p.id === 'subdl') return { open: 'Open SubDL’s API page', openSub: 'Sign in, or create a free account, if it asks.', copy: 'Generate a key there and copy it.' };
  return { open: `Open ${p.name}`, openSub: 'Sign in, or create a free account, then open My Profile from your account menu.', copy: 'Generate an API key there and copy it. You can regenerate it any time.' };
}

function openKey(pid, opener) {
  const s = scen.status[pid];
  st.sheet = { kind: 'key', pid, step: 1, phase: 'idle', replace: !!s.key, opener };
  paintSheet();
  root.querySelector('.rs [data-step-link]')?.focus({ preventScroll: true });
}

function keySheetHTML() {
  const sh = st.sheet, p = P(sh.pid), k = keyStepsFor(p);
  const head = `${sh.replace ? 'Replace' : 'Add'} your ${p.name} key`;
  if (sh.phase === 'ok') return sheetFrame(head, 'Checked with ' + p.name, keyResultHTML());
  const stepCls = (n) => (sh.step > n ? 'done' : sh.step === n ? 'cur' : '');
  const num = (n) => `<span class="gs-n" aria-hidden="true">${sh.step > n ? I('check') : n}</span>`;
  const body = `<p class="rs-lede">Free, and about a minute. SlipStream checks the key the moment you paste it.</p>
    <ol class="gs">
      <li class="gs-i ${stepCls(1)}" data-step="1">${num(1)}<div class="gs-c">
        <a class="gs-link" href="${p.keyPage}" target="_blank" rel="noopener" data-step-link>${esc(k.open)}${I('arrow-up-right')}</a>
        <p class="t3">${esc(k.openSub)}</p><p class="mono gs-url">${esc(p.keyPage.replace('https://', ''))}</p></div></li>
      <li class="gs-i ${stepCls(2)}" data-step="2" tabindex="-1">${num(2)}<div class="gs-c"><p class="gs-t">Copy the key there</p><p class="t3">${esc(k.copy)}</p></div></li>
      <li class="gs-i ${stepCls(3)}" data-step="3">${num(3)}<div class="gs-c"><label class="gs-t" for="vb-key">Paste it here</label>
        <input id="vb-key" class="gs-in mono" type="text" autocomplete="off" spellcheck="false" placeholder="${p.id === 'subdl' ? 'Your SubDL API key' : 'Your SubSource API key'}" aria-describedby="vb-key-res">
        <div id="vb-key-res" class="gs-res" aria-live="polite"></div></div></li>
    </ol>
    <div class="gs-foot"><p class="never">${I('lock-keyhole')}<span>${esc(p.never)}</span></p><p class="t3 gs-how">${esc(p.check)}</p></div>`;
  return sheetFrame(head, `${p.name} · ${p.covers}`, body);
}

function keyResultHTML() {
  const sh = st.sheet, p = P(sh.pid), s = scen.status[sh.pid], a = p.allowance;
  const left = a.limit - (s.used || 0);
  const paused = s.state === 'paused';
  const allow = sh.pid === 'subdl'
    ? `${left} of ${a.limit} downloads left today · resets at 00:00 UTC`
    : `${S.fmt(left)} of ${S.fmt(a.limit)} requests left · rolling 24 hours`;
  const rank = scen.order.indexOf(sh.pid);
  return `<div class="gs-ok">
      <p class="gs-ok-h">${I('badge-check')}<b>Key works</b></p>
      <dl class="vb-kv">
        <dt>Account</dt><dd>${esc(s.account)}</dd>
        <dt>Plan</dt><dd>${esc(s.plan || 'Free')}</dd>
        <dt>Allowance</dt><dd><span>${allow}</span><div class="al${paused ? ' al-warn' : ''}"><span style="width:${(left / a.limit) * 100}%"></span></div>${paused ? `<p class="warn-t gs-note">Still out of downloads until ${s.resumesAt}. The allowance belongs to the account, not the key.</p>` : ''}</dd>
        <dt>Key</dt><dd class="gs-keyrow"><span class="mono">${esc(s.key)}</span><span class="grow"></span><button class="btn ghost sm" data-key-replace>Replace</button><button class="btn ghost sm" data-key-remove="${sh.pid}">Remove</button></dd>
      </dl>
    </div>
    <p class="rs-lede">${p.name} is ${ORD[rank]} in the order. ${paused ? `It starts again at ${s.resumesAt}.` : 'SlipStream starts searching it now for every file missing a Wanted Language.'}</p>
    <p class="gs-done"><button class="btn pri" data-done>Done</button></p>
    <p class="never">${I('lock-keyhole')}<span>${esc(p.never)}</span></p>`;
}

let checkT, typeT;
function onKeyInput(inp, pasted) {
  const sh = st.sheet;
  clearTimeout(checkT); clearTimeout(typeT);
  if (sh.step < 3) setStep(3);
  const v = inp.value.trim();
  const res = root.querySelector('#vb-key-res');
  if (!v) { res.innerHTML = ''; inp.removeAttribute('aria-invalid'); return; }
  const run = () => {
    const p = P(sh.pid);
    sh.phase = 'checking';
    res.innerHTML = `<p class="gs-chk"><span class="spin" aria-hidden="true"></span>Checking with ${p.name}…</p>`;
    inp.removeAttribute('aria-invalid');
    checkT = setTimeout(() => {
      if (/bad/i.test(v) || v.length < 12) {
        sh.phase = 'fail';
        inp.setAttribute('aria-invalid', 'true');
        res.innerHTML = `<div class="gs-fail"><p><b>${p.name} rejected this key</b></p><p class="mono">${p.reject.status} · ${esc(p.reject.body)}</p><p class="gs-fail-d">Copy it again from the ${p.name} page. Keys are long and case-sensitive, so take the whole thing.</p></div>`;
        C.icons();
        return;
      }
      const prev = scen.status[sh.pid];
      const used = prev.used || 0;
      scen.status[sh.pid] = used >= (p.allowance.limit)
        ? { ...prev, key: mask(v), account: 'jackson', plan: 'Free' }
        : { key: mask(v), state: 'ready', account: 'jackson', plan: 'Free', used, lastOk: 'just now' };
      sh.phase = 'ok';
      paintSheet();
      paint();
      root.querySelector('.rs [data-done]')?.focus({ preventScroll: true });
    }, 900);
  };
  if (pasted) run(); else typeT = setTimeout(run, 650);
}

function setStep(n) {
  const sh = st.sheet;
  if (!sh || sh.kind !== 'key' || sh.phase === 'ok' || n <= sh.step) return;
  sh.step = n;
  root.querySelectorAll('.rs .gs-i').forEach((li) => {
    const k = +li.dataset.step;
    li.classList.toggle('done', k < n);
    li.classList.toggle('cur', k === n);
    const num = li.querySelector('.gs-n');
    num.innerHTML = k < n ? I('check') : String(k);
  });
  C.icons();
}

function removeKey(pid) {
  scen.status[pid] = { key: null, state: 'off' };
  closeSheet();
  paint();
  const left = scen.order.filter((id) => id !== pid && P(id).covers !== 'TV only' && scen.status[id].state === 'ready').map((id) => P(id).name);
  toast(`${P(pid).name} key removed. ${left.length ? `Movies now search ${left.join(' and ')}.` : 'Movies have no Subtitle Provider now.'}`);
}

// ───────────────────────────── sheet frame

function sheetFrame(title, sub, body, foot = '') {
  return `<div class="rs-scrim" data-close></div>
    <div class="rs" role="dialog" aria-modal="true" aria-labelledby="rs-t">
      ${phone ? '<span class="grab" aria-hidden="true"></span>' : ''}
      <header class="rs-h"><div class="grow"><h2 id="rs-t">${esc(title)}</h2><p class="t3">${esc(sub)}</p></div><button class="ib" data-close aria-label="Close">${I('x')}</button></header>
      <div class="rs-b">${body}</div>
      ${foot ? `<footer class="rs-f">${foot}</footer>` : ''}
    </div>`;
}

function paintSheet() {
  const ov = root.querySelector('#vb-ov');
  if (!st.sheet) { ov.innerHTML = ''; return; }
  const scroll = ov.querySelector('.rs-b')?.scrollTop || 0;
  ov.innerHTML = st.sheet.kind === 'key' ? keySheetHTML() : searchSheetHTML();
  const b = ov.querySelector('.rs-b'); if (b) b.scrollTop = scroll;
  C.icons();
}

function closeSheet() {
  const op = st.sheet?.opener;
  clearTimeout(checkT); clearTimeout(typeT);
  st.sheet = null;
  paintSheet();
  if (op && document.contains(op)) op.focus({ preventScroll: true });
}

// ───────────────────────────── library files and wants

function buildFiles() {
  const fresh = C.scenarioId === 'fresh', strained = C.scenarioId === 'strained';
  const movieFiles = clone(D.works['dune-2'].files).map((f) => {
    for (const [wid, w] of Object.entries(f.wants)) {
      if (fresh && ['final', 'present', 'nosync', 'missing'].includes(w.state)) {
        f.wants[wid] = { state: 'missing', note: 'No Subtitle Provider covers movies yet. Gestdown is TV only.', next: 'Add a SubDL or SubSource key to start', noProvider: true };
      } else if (strained && (w.state === 'missing' || w.state === 'nosync')) {
        w.next = 'Waits for 00:00 UTC, when SubDL’s downloads reset';
        if (w.state === 'missing') w.note = `${w.note} SubDL is out of downloads and SubSource rejected its key.`;
      }
    }
    return f;
  });
  const sev = D.works.severance;
  const eps = clone(sev.episodes).map((e) => {
    const f = { id: `ep-${e.n}`, ep: e.n, title: e.title, target: '1080p', name: `Severance.S02E0${e.n}.1080p.ATVP.WEB-DL.DDP5.1.H.264-FLUX.mkv`, size: `${(2.9 + e.n * 0.13).toFixed(1)} GB`, wants: e.wants, fresh: e.fresh };
    for (const [wid, w] of Object.entries(f.wants)) {
      if (w.state === 'notneeded') w.note = 'Wanted only without Spanish audio. This file carries a Spanish audio track.';
      if (fresh && w.sub && w.sub.provider !== 'gestdown') f.wants[wid] = { state: 'missing', note: 'Gestdown had nothing that passed sync. SubDL and SubSource need your free key.', next: 'Searched again on schedule' };
      if (w.sub && !w.sub.file) w.sub.file = `Severance.S02E0${e.n}.1080p.ATVP.WEB-DL.DDP5.1.H.264-FLUX.${CODE.English}.srt`;
      if (w.state === 'missing' && !w.next) w.next = `Searching ${scen.order.filter((id) => scen.status[id].state === 'ready').map((id) => P(id).name).join(', ')}`;
    }
    return f;
  });
  return [...movieFiles, ...eps];
}

const fileOf = (id) => st.files.find((f) => f.id === id);
const wantedFor = (f) => (f.ep ? st.saved.tv : st.saved.cinema);
const wlOf = (f, wid) => wantedFor(f).find((w) => w.id === wid);
const provName = (id) => (id === 'upload' ? 'Upload' : P(id)?.name || id);

function nosyncShort(w) {
  const m = /Best alignment (\d+%)/.exec(w.note || '');
  if (m) return `${w.tried} tried, best alignment ${m[1]}`;
  const tail = (w.note || '').split('. ').slice(1).join('. ').replace(/\.$/, '');
  return `${w.tried} tried${tail ? `, ${tail.charAt(0).toLowerCase()}${tail.slice(1)}` : ''}`;
}

function stateParts(f, wid) {
  const w = f.wants[wid], wl = wlOf(f, wid);
  if (w.busy) return { label: w.busy, rest: [], live: true };
  const sub = w.sub;
  switch (w.state) {
    case 'final':
    case 'present': {
      const r = sub.reasons.filter((x) => REASON_SHORT[x]).map((x) => REASON_SHORT[x]);
      const parts = [provName(sub.provider)];
      if (r.length) parts.push(r[0]);
      else if (sub.provider !== 'upload') parts.push('no release match');
      parts.push(sub.by === 'ai' ? { warn: 'AI-translated' } : 'human-made');
      if (w.state === 'present' && sub.provider === 'upload') parts.splice(1, 1, `synced ${sub.sync.split(',')[0]}`);
      return { label: D.subStates[w.state].label, rest: parts };
    }
    case 'embedded': return { label: 'Embedded', rest: w.track.split(' · ') };
    case 'missing': return { label: 'Missing', rest: [lower(w.noProvider ? 'no Subtitle Provider covers movies yet' : w.next || w.note)], live: /^Searching/.test(w.next || '') && !w.noProvider };
    case 'nosync': return { label: 'No candidate passed sync', rest: [nosyncShort(w), lower(w.next || 'searched again later')] };
    case 'notneeded': return { label: 'Not needed', rest: [`this file has ${wl.lang} audio`] };
    default: return { label: w.state, rest: [] };
  }
}
const lower = (s) => (s ? s.charAt(0).toLowerCase() + s.slice(1) : s);

function stateIcon(w) {
  if (w.busy) return '<span class="spin" aria-hidden="true"></span>';
  return I(D.subStates[w.state].icon);
}

function wantRow(f, wid) {
  const w = f.wants[wid], wl = wlOf(f, wid), key = `${f.id}|${wid}`, open = st.open.has(key);
  const tone = w.busy ? 'live' : D.subStates[w.state].tone;
  const sp = stateParts(f, wid);
  const rest = sp.rest.map((x) => (typeof x === 'object' ? `<span class="warn-t">${esc(x.warn)}</span>` : esc(x))).join('<span class="s-sep">·</span>');
  const hasSub = (w.state === 'present' || w.state === 'final') && !w.busy;
  const can = w.state !== 'notneeded' && !w.busy;
  const did = `wd-${key.replace(/[^a-z0-9]/gi, '-')}`;
  return `<li class="wr wr-${tone}${open ? ' open' : ''}" data-row="${key}">
    <div class="wr-row">
      <button class="wr-tg" data-toggle="${key}" aria-expanded="${open}" aria-controls="${did}">
        <span class="wr-ic">${stateIcon(w)}</span>
        <span class="wr-l"><b>${esc(wl.lang)}${wl.kind === 'forced' ? ' forced' : ''}</b><span class="t3">${KIND_TXT[wl.kind]} · ${HI_SHORT[wl.hi]}</span></span>
        <span class="wr-s">${sp.live && !w.busy ? '<span class="dot dot-draft" aria-hidden="true"></span>' : ''}<b>${esc(sp.label)}</b>${rest ? `<span class="s-sep">·</span>${rest}` : ''}</span>
        ${I('chevron-down', 'wr-chev')}
      </button>
      <div class="wr-a">
        ${can ? `<button class="btn sm${w.state === 'missing' || w.state === 'nosync' ? ' pri' : ''}" data-search="${key}">${I('search')}<span>Search</span></button><button class="btn sm" data-upload="${key}">${I('upload')}<span>Upload</span></button>` : ''}
        ${hasSub ? `<button class="ib sm" data-wmenu="${key}" aria-haspopup="menu" aria-label="More for ${esc(wl.lang)} on ${esc(f.target)}">${I('ellipsis')}</button>` : can ? '<span class="ib-sp" aria-hidden="true"></span>' : ''}
      </div>
    </div>
    <div class="wr-d" id="${did}" ${open ? '' : 'hidden'}>${open ? wantDetail(f, wid) : ''}</div>
  </li>`;
}

function wantDetail(f, wid) {
  const w = f.wants[wid], wl = wlOf(f, wid), meaning = D.subStates[w.state].means;
  if (w.busy) return `<p class="t2">${esc(w.busy)}</p>`;
  const kv = (rows) => `<dl class="vb-kv">${rows.filter(Boolean).map(([k, v]) => `<dt>${k}</dt><dd>${v}</dd>`).join('')}</dl>`;
  if (w.state === 'final' || w.state === 'present') {
    const s = w.sub;
    const chips = s.reasons.length ? s.reasons.map((r) => `<span class="chip">${esc(D.reasonLabels[r].label)}</span>`).join(' ') : '<span class="t3">No hash, group or source match</span>';
    return `${kv([
      ['Sidecar', `<span class="mono">${esc(s.file)}</span>`],
      ['From', s.provider === 'upload' ? `Your upload <span class="t3">· ${esc(s.release)}</span>` : esc(provName(s.provider))],
      s.provider !== 'upload' && ['Release', `<span class="mono">${esc(s.release)}</span>`],
      ['Match', `<span class="vb-chips">${chips}</span>`],
      ['Timing sync', /skipped|copied/.test(s.sync) ? esc(s.sync.charAt(0).toUpperCase() + s.sync.slice(1)) : `Lined up with the Speech Map · ${esc(s.sync)}`],
      ['Made by', s.by === 'ai' ? '<span class="vb-warnchip">AI-translated</span> <span class="t3">ranks below every human-made one</span>' : 'Human'],
      s.at && ['Fetched', new Date(s.at).toLocaleString('en-US', { month: 'short', day: 'numeric', hour: '2-digit', minute: '2-digit', hour12: false, timeZone: 'UTC' }) + ' UTC'],
      w.next && ['Next', esc(w.next)],
    ])}<p class="wr-mean t3">${esc(meaning)}</p>`;
  }
  if (w.state === 'embedded') {
    const others = (f.embedded || []).filter((t) => t.note).map((t) => `${t.lang} ${t.codec}: ${t.note}`);
    return `${kv([['Track', esc(w.track)], others.length && ['Also inside', esc(others.join(' '))]])}<p class="wr-mean t3">${esc(meaning)}</p>`;
  }
  if (w.state === 'missing') {
    return `${kv([['Why', esc(w.note)], w.next && ['Next', esc(w.next)]])}${w.noProvider ? `<p class="wr-cta"><button class="btn sm pri" data-go="providers">${I('key-round')}Add a key</button><span class="t3">Or upload one you have. Uploads are synced like any other.</span></p>` : `<p class="wr-mean t3">${esc(meaning)}</p>`}`;
  }
  if (w.state === 'nosync') {
    return `${kv([['What happened', esc(w.note)], ['Next', esc(w.next || 'Searched again later')]])}<p class="wr-mean t3">${esc(meaning)}</p><p class="wr-cta"><button class="btn sm" data-search="${f.id}|${wid}" data-tried>${I('history')}See what was tried</button></p>`;
  }
  return `${kv([['Why', esc(w.note || '')], ['Wanted', `Only when no audio track is in ${esc(wl.lang)}`]])}<p class="wr-mean t3">${esc(meaning)}</p>`;
}

function tally(files) {
  let met = 0, att = 0, all = 0;
  for (const f of files) for (const w of Object.values(f.wants)) { all++; if (['missing', 'nosync'].includes(w.state)) att++; else met++; }
  return { met, att, all };
}

function banner(kind) {
  const s = scen.status;
  const movieReady = scen.order.some((id) => P(id).covers !== 'TV only' && s[id].state === 'ready');
  if (kind === 'movie' && !movieReady) {
    const strained = Object.values(s).some((x) => x.state === 'paused' || x.state === 'invalid');
    const msg = strained
      ? '<b>Movie searches are on hold until 00:00 UTC.</b> SubDL is out of downloads and SubSource rejected its key. Gestdown covers TV only.'
      : '<b>No Subtitle Provider covers movies yet.</b> Gestdown is TV only. SubDL and SubSource each need your own free key.';
    return `<div class="vb-banner">${I('triangle-alert')}<p>${msg}</p><button class="btn sm" data-go="providers">${strained ? 'Open Subtitle Providers' : 'Add a key'}</button></div>`;
  }
  if (kind === 'series') {
    const down = scen.order.filter((id) => s[id].state !== 'ready').map((id) => P(id).name);
    if (down.length) return `<p class="vb-quiet">${I('info')}<span>Gestdown finds TV subtitles with no key. ${down.join(' and ')} ${down.length > 1 ? 'are' : 'is'} not searched right now. <button class="lnk vb-a" data-go="providers">Subtitle Providers</button></span></p>`;
  }
  return '';
}

function workHead(w, line, sub) {
  return `<div class="vb-wk">
    <div class="vb-wk-bd" aria-hidden="true"><img src="${w.art.backdrop}" alt="" onerror="this.remove()"></div>
    ${S.art(w, phone ? 'poster-sm' : 'poster-md')}
    <div class="vb-wk-id">${phone ? '' : '<p class="t3 vb-eyebrow">Subtitles</p>'}<h1>${esc(w.title)}</h1><p class="t2">${line}</p><p class="vb-wk-sum">${sub}</p></div>
  </div>`;
}

function movieView() {
  const w = D.works['dune-2'], files = st.files.filter((f) => !f.ep), t = tally(files);
  const wanted = st.saved.cinema;
  const sum = t.att ? `${t.met} of ${t.all} wants met across ${files.length} files · <span class="warn-t">${t.att} need${t.att === 1 ? 's' : ''} attention</span>` : `All ${t.all} wants met across ${files.length} files`;
  return `${workHead(w, `<span class="num">${w.year}</span> · ${w.edition} Edition · Profile Cinema · ${wanted.length} Wanted Languages`, sum)}
    ${banner('movie')}
    ${files.map((f) => `<section class="fs" aria-labelledby="h-${f.id}">
      <header class="fs-h"><h2 id="h-${f.id}">${esc(f.target)}</h2><span class="t3">${f.derived ? 'Target · converted' : 'Target · downloaded'}</span><span class="grow"></span><span class="num t3">${esc(f.size)} · ${esc(f.node)}</span></header>
      <p class="fs-f mono">${I('file-video')}<span>${esc(f.name)}</span></p>
      ${f.derived ? `<p class="fs-note t3">${esc(f.derived)}</p>` : ''}
      <ul class="wr-list">${wanted.map((wl) => wantRow(f, wl.id)).join('')}</ul>
    </section>`).join('')}`;
}

function seriesView() {
  const w = D.works.severance, eps = st.files.filter((f) => f.ep), t = tally(eps), wanted = st.saved.tv;
  const chip = (f, wl) => {
    const x = f.wants[wl.id], s = D.subStates[x.state];
    const tone = x.busy ? 'live' : s.tone;
    return `<span class="ec ec-${tone}" title="${esc(wl.lang)}: ${esc(x.busy || s.label)}">${x.busy ? '<span class="spin" aria-hidden="true"></span>' : I(s.icon)}<b>${(CODE[wl.lang] || '').toUpperCase()}</b><span>${esc(x.busy ? 'Working' : s.label === 'No candidate passed sync' ? 'No sync' : s.label)}</span></span>`;
  };
  return `${workHead(w, `Season ${w.season} · ${eps.length} episodes · Profile TV Standard`, t.att ? `${t.met} of ${t.all} wants met · <span class="warn-t">${t.att} need attention</span>` : `All ${t.all} wants met`)}
    ${banner('series')}
    <section class="vb-sec" aria-labelledby="eps-h">
      <div class="vb-sec-h"><h2 id="eps-h">Episodes</h2><span class="t3">${wanted.map((wl) => `${wl.lang} ${KIND_TXT[wl.kind]}`).join(' · ')}</span></div>
      <ul class="ep-list">${eps.map((f) => {
        const open = st.ep === f.ep;
        return `<li class="ep${open ? ' open' : ''}">
          <button class="ep-row" data-ep="${f.ep}" aria-expanded="${open}" aria-controls="ep-d-${f.ep}">
            <span class="num ep-n">E${String(f.ep).padStart(2, '0')}</span><span class="ep-t">${esc(f.title)}${f.fresh ? '<span class="vb-tag">New</span>' : ''}</span>
            <span class="ep-chips">${wanted.map((wl) => chip(f, wl)).join('')}</span>${I('chevron-down', 'wr-chev')}
          </button>
          <div class="ep-d" id="ep-d-${f.ep}" ${open ? '' : 'hidden'}>${open ? `<p class="fs-f mono">${I('file-video')}<span>${esc(f.name)}</span><span class="t3 num">${esc(f.size)}</span></p><ul class="wr-list">${wanted.map((wl) => wantRow(f, wl.id)).join('')}</ul>` : ''}</div>
        </li>`;
      }).join('')}</ul>
    </section>`;
}

// ───────────────────────────── manual search

function candidatesFor(f, wid) {
  const wl = wlOf(f, wid);
  let list;
  if (f.id === D.searchFor.file && wid === D.searchFor.want) list = clone(D.candidates);
  else if (f.ep) {
    const n = `S02E0${f.ep}`;
    list = [
      { id: `${f.id}-g1`, provider: 'gestdown', release: `Severance.${n}.1080p.ATVP.WEB-DL.DDP5.1.H.264-FLUX`, lang: wl.lang, reasons: ['group'], downloads: 4210, rating: null, uploader: 'Addic7ed', status: 'new' },
      { id: `${f.id}-s1`, provider: 'subdl', release: `Severance.${n}.1080p.WEB.H264-SuccessfulCrab`, lang: wl.lang, reasons: ['source'], downloads: 2984, rating: 8.8, uploader: 'GoldenBeard', status: 'new' },
      { id: `${f.id}-s2`, provider: 'subsource', release: `Severance.${n}.720p.WEB.h264-ETHEL`, lang: wl.lang, reasons: [], downloads: 1530, rating: 8.1, uploader: 'msub', status: 'new' },
      { id: `${f.id}-s3`, provider: 'subdl', release: `Severance.${n}.1080p.WEB.H264-SuccessfulCrab`, lang: wl.lang, reasons: ['source', 'ai'], downloads: 310, rating: null, uploader: 'SubDL AI', status: 'new' },
    ];
  } else {
    list = clone(D.candidates).map((c) => ({ ...c, id: `${f.id}-${wid}-${c.id}`, lang: wl.lang, status: c.status === 'blocked' ? 'blocked' : 'new', syncNote: c.status === 'blocked' ? c.syncNote : undefined }));
  }
  for (const b of st.blocked[`${f.id}|${wid}`] || []) {
    const hit = list.find((c) => c.release === b.release && c.provider === b.provider);
    if (hit) Object.assign(hit, { status: 'blocked', syncNote: b.syncNote });
    else list.push(b);
  }
  return list;
}

const reasonsKey = (c, order) => [c.reasons.includes('ai') || c.reasons.includes('machine') ? 1 : 0, c.reasons.includes('hash') ? 0 : 1, c.reasons.includes('group') ? 0 : 1, c.reasons.includes('source') ? 0 : 1, order.indexOf(c.provider), -c.downloads];
function rank(list, order) {
  return list.slice().sort((a, b) => { const x = reasonsKey(a, order), y = reasonsKey(b, order); for (let i = 0; i < x.length; i++) if (x[i] !== y[i]) return x[i] - y[i]; return 0; });
}

function searchCtx() {
  const sh = st.sheet, f = fileOf(sh.fileId), wl = wlOf(f, sh.wid);
  const order = scen.order.filter((id) => !(f.ep ? false : P(id).covers === 'TV only'));
  const searched = order.filter((id) => ['ready', 'paused'].includes(scen.status[id].state));
  const skipped = order.filter((id) => !searched.includes(id));
  const all = candidatesFor(f, sh.wid).filter((c) => searched.includes(c.provider) || c.status !== 'new');
  const live = rank(all.filter((c) => c.status === 'new'), scen.order);
  const tried = all.filter((c) => c.status !== 'new');
  return { f, wl, order, searched, skipped, live, tried };
}

function openSearch(key, opener, tried) {
  const [fileId, wid] = key.split('|');
  st.sheet = { kind: 'search', fileId, wid, pick: null, tried: !!tried, opener };
  paintSheet();
  const rb = root.querySelector('.rs-b');
  if (tried) root.querySelector('.rs details.tb')?.scrollIntoView({ block: 'start' });
  root.querySelector('.rs [data-pick]:not([disabled]), .rs [data-close]')?.focus({ preventScroll: true });
  if (rb && !tried) rb.scrollTop = 0;
}

function checkItem(ok, label) {
  return `<li class="${ok ? 'y' : 'n'}">${I(ok ? 'check' : 'x')}<span>${label}</span><span class="sr">${ok ? 'yes' : 'no'}</span></li>`;
}

function card(c, i, x) {
  const sh = st.sheet, pk = sh.pick, mine = pk && pk.id === c.id;
  const paused = scen.status[c.provider].state === 'paused';
  const ai = c.reasons.includes('ai') || c.reasons.includes('machine');
  const ordIdx = scen.order.indexOf(c.provider);
  const btn = paused
    ? `<button class="btn sm" data-queue="${c.id}" ${pk ? 'disabled' : ''}>${I('clock')}Queue for ${scen.status[c.provider].resumesAt}</button><span class="t3 cd-why">${P(c.provider).name} is out of downloads.</span>`
    : `<button class="btn sm${i === 0 ? ' pri' : ''}" data-pick="${c.id}" ${pk ? 'disabled' : ''}>${I('download')}Use this</button>`;
  return `<li class="cd${i === 0 ? ' best' : ''}${mine ? ' running' : ''}">
    <div class="cd-h"><span class="cd-n num" aria-hidden="true">${i + 1}</span>
      <div class="cd-id"><p class="mono cd-rel">${esc(c.release)}</p><p class="t3">${esc(c.lang)} · ${P(c.provider).name} · by ${esc(c.uploader)}</p></div>
      ${i === 0 ? '<span class="vb-best">Best</span>' : ''}</div>
    <ul class="cd-ck" aria-label="Match reasons, in ranking order">
      ${checkItem(c.reasons.includes('hash'), 'Hash match')}
      ${checkItem(c.reasons.includes('group'), 'Same group')}
      ${checkItem(c.reasons.includes('source'), 'Same source and resolution')}
      <li class="i">${I('list-ordered')}<span>Provider order: ${ORD[ordIdx]}</span></li>
      <li class="i">${I('download')}<span>Downloads <span class="num">${S.fmt(c.downloads)}</span>${c.rating ? ` · rating <span class="num">${c.rating}</span>` : ''}</span></li>
    </ul>
    ${ai || c.hi ? `<p class="cd-tags">${ai ? `<span class="vb-warnchip">${I('sparkles')}${c.reasons.includes('ai') ? 'AI-translated' : 'Machine-translated'}</span><span class="t3">Ranks below every human-made one.</span>` : ''}${c.hi ? '<span class="chip">Hearing impaired</span>' : ''}</p>` : ''}
    ${mine ? runHTML(c, x) : `<div class="cd-a">${btn}</div>`}
  </li>`;
}

function runHTML(c, x) {
  const ph = st.sheet.pick.phase;
  const hash = c.reasons.includes('hash');
  const step = (on, done, label) => `<li class="${done ? 'done' : on ? 'cur' : ''}">${done ? I('check') : on ? '<span class="spin" aria-hidden="true"></span>' : I('circle')}<span>${label}</span></li>`;
  const wl = x.wl;
  return `<div class="cd-run" aria-live="polite">
    <ol class="run">
      ${step(ph === 'dl', ph !== 'dl', `Download from ${P(c.provider).name}`)}
      ${step(ph === 'sync', ph === 'ok', hash ? 'Hash match, Timing sync skipped' : 'Timing sync against the Speech Map')}
    </ol>
    ${ph === 'ok' ? `<div class="run-ok"><p><b>Passed.</b> ${hash ? 'Made for this exact file.' : 'Lined up 96% with the Speech Map, offset +80 ms. The gate is 70%.'}</p><p class="mono t3">${esc(sidecarName(x.f, wl))}</p><button class="btn pri sm" data-pick-done>Done</button></div>` : ''}
  </div>`;
}

function sidecarName(f, wl) {
  const base = f.ep ? f.name.replace(/\.mkv$/, '') : f.name.replace(/\.mkv$/, '');
  return `${base}.${CODE[wl.lang] || 'xx'}${wl.kind === 'forced' ? '.forced' : ''}.srt`;
}

function searchSheetHTML() {
  const x = searchCtx(), { f, wl } = x, names = (ids) => ids.map((id) => P(id).name).join(' and ');
  const title = `${wl.lang}${wl.kind === 'forced' ? ' forced' : ''} for ${f.ep ? `E${String(f.ep).padStart(2, '0')}` : f.target}`;
  const sub = f.ep ? `Severance · ${f.title}` : `Dune: Part Two · ${f.name}`;
  const paused = x.searched.filter((id) => scen.status[id].state === 'paused');
  const cover = [
    x.searched.length ? `Searched ${names(x.searched)}.` : '',
    ...x.skipped.map((id) => `${P(id).name} was skipped because ${scen.status[id].state === 'invalid' ? 'its key was rejected' : 'it has no key'}.`),
    !f.ep ? 'Gestdown covers TV only.' : '',
    paused.length ? `${names(paused)} is out of downloads until ${scen.status[paused[0]].resumesAt}, so its candidates can only be queued.` : '',
  ].filter(Boolean).join(' ');
  let body;
  if (!x.searched.length) {
    body = `<div class="vb-empty">${I('key-round')}<p><b>No Subtitle Provider covers movies yet.</b></p><p class="t2">Gestdown is TV only. SubDL and SubSource each need your own free key, which takes about a minute.</p><p class="vb-empty-a"><button class="btn pri sm" data-go="providers">Add a key</button><button class="btn sm" data-upload="${f.id}|${wl.id}">${I('upload')}Upload instead</button></p></div>`;
  } else {
    body = `<p class="rs-lede">Ranked by hash match, same group, same source and resolution, provider order, then downloads. AI-translated ones always rank below human-made ones.</p>
      <p class="vb-cover t3">${esc(cover)}</p>
      <ol class="cd-list" aria-label="Candidates, best first">${x.live.map((c, i) => card(c, i, x)).join('')}</ol>
      ${x.live.length ? '' : '<p class="t2">No new candidates.</p>'}`;
  }
  if (x.tried.length && x.searched.length) {
    body += `<details class="tb" ${st.sheet.tried ? 'open' : ''}><summary>${I('chevron-right', 'tb-chev')}<b>Tried before</b><span class="t3 num">${x.tried.length}</span><span class="t3 tb-why">Never picked again for this file</span></summary>
      <ul>${x.tried.map((c) => `<li class="tb-i"><span class="tb-ic ${c.status === 'blocked' ? 'fail-t' : 'warn-t'}">${I(c.status === 'blocked' ? 'ban' : 'audio-waveform')}</span><div class="grow"><p class="mono cd-rel">${esc(c.release)}</p><p class="t3">${P(c.provider).name} · <b class="${c.status === 'blocked' ? 'fail-t' : 'warn-t'}">${c.status === 'blocked' ? 'Marked bad' : 'Failed sync'}</b> · ${esc(c.syncNote || '')}</p></div>${c.status === 'blocked' ? `<button class="btn ghost sm" data-unblock="${esc(c.release)}|${c.provider}">Unblock</button>` : ''}</li>`).join('')}</ul></details>`;
  }
  return sheetFrame(title, sub, body, `<p class="t3 rs-f-row">${I('upload')}<span>Have a better one?</span><button class="btn sm" data-upload="${f.id}|${wl.id}">Upload a subtitle</button></p>`);
}

let pickT = [];
function pick(cid) {
  const x = searchCtx(), c = x.live.find((k) => k.id === cid);
  st.sheet.pick = { id: cid, phase: 'dl' };
  paintSheet();
  pickT.push(setTimeout(() => { if (!st.sheet?.pick) return; st.sheet.pick.phase = 'sync'; paintSheet(); }, 800));
  pickT.push(setTimeout(() => {
    if (!st.sheet?.pick) return;
    st.sheet.pick.phase = 'ok';
    const f = fileOf(st.sheet.fileId), wl = wlOf(f, st.sheet.wid);
    const human = !c.reasons.includes('ai') && !c.reasons.includes('machine');
    f.wants[wl.id] = {
      state: human && (c.reasons.includes('hash') || c.reasons.includes('group')) ? 'final' : 'present',
      sub: { provider: c.provider, release: c.release, file: sidecarName(f, wl), reasons: c.reasons.filter((r) => r !== 'hi'), sync: c.reasons.includes('hash') ? 'skipped, hash match' : '+80 ms, framerate unchanged', by: human ? 'human' : 'ai', at: '2026-10-08T21:40:00Z' },
      next: human && (c.reasons.includes('hash') || c.reasons.includes('group')) ? null : 'Upgrade search tonight at 03:00',
    };
    st.open.add(`${f.id}|${wl.id}`);
    paintSheet();
    paint();
    root.querySelector('.rs [data-pick-done]')?.focus({ preventScroll: true });
  }, 2300));
}

// ───────────────────────────── upload and mark bad

function startUpload(key, file) {
  const [fid, wid] = key.split('|'), f = fileOf(fid), wl = wlOf(f, wid);
  const name = file?.name || 'subtitle.srt';
  if (st.sheet) closeSheet();
  const prev = f.wants[wid];
  f.wants[wid] = { ...prev, busy: `Timing sync against the Speech Map · ${name}` };
  st.open.add(key);
  paint();
  setTimeout(() => {
    f.wants[wid] = { state: 'present', sub: { provider: 'upload', release: name, file: sidecarName(f, wl), reasons: [], sync: '+120 ms, framerate unchanged', by: 'human', at: '2026-10-08T21:40:00Z' }, next: 'Upgrade search tonight at 03:00' };
    paint();
    toast(`${name} synced and saved as the ${wl.lang} Sidecar.`);
  }, 1600);
}

function markBad(key) {
  const [fid, wid] = key.split('|'), f = fileOf(fid), wl = wlOf(f, wid), w = f.wants[wid];
  const k = `${fid}|${wid}`;
  (st.blocked[k] ||= []).push({ id: `blk-${Date.now()}`, provider: w.sub.provider === 'upload' ? 'subdl' : w.sub.provider, release: w.sub.release, lang: wl.lang, reasons: w.sub.reasons, downloads: 0, rating: null, uploader: '', status: 'blocked', syncNote: `Marked bad at ${NOW} by Jackson.` });
  f.wants[wid] = { state: 'missing', busy: 'Searching again' };
  closePop();
  paint(`[data-toggle="${k}"]`);
  toast(`${wl.lang} Sidecar removed and blocked for this file. Searching again.`);
  setTimeout(() => {
    f.wants[wid] = { state: 'missing', note: `Marked bad at ${NOW}. The search that followed found nothing new that passed sync.`, next: 'Searched again tomorrow at 03:00' };
    paint();
  }, 1800);
}

// ───────────────────────────── popovers (menu on wide, small sheet on phone)

function openPop(anchor, html, label) {
  closePop();
  const el = root.querySelector('#vb-pop');
  st.pop = { anchor };
  if (phone) {
    el.innerHTML = `<div class="sheet-wrap"><div class="sheet-scrim" data-pop-close></div><div class="sheet vb-psheet" role="dialog" aria-modal="true" aria-label="${esc(label)}"><span class="grab" aria-hidden="true"></span><p class="sheet-t"><b>${esc(label)}</b></p>${html}<button class="sheet-x" data-pop-close>Cancel</button></div></div>`;
  } else {
    el.innerHTML = `<div class="menu vb-menu" role="dialog" aria-label="${esc(label)}">${html}</div>`;
    const m = el.querySelector('.menu'), r = anchor.getBoundingClientRect();
    m.style.position = 'fixed';
    m.style.top = `${Math.min(r.bottom + 6, innerHeight - m.offsetHeight - 12)}px`;
    m.style.left = `${Math.max(12, Math.min(r.left, innerWidth - m.offsetWidth - 12))}px`;
  }
  C.icons();
  el.querySelector('.mi[aria-checked="true"], .mi, .btn')?.focus({ preventScroll: true });
}

function closePop(refocus) {
  const a = st.pop?.anchor;
  root.querySelector('#vb-pop').innerHTML = '';
  st.pop = null;
  if (refocus && a && document.contains(a)) a.focus({ preventScroll: true });
}

function provMenu(pid, anchor) {
  openPop(anchor, `<div class="vb-mlist" role="menu"><button class="mi" role="menuitem" data-key="${pid}">${I('key-round')}<span>Replace key</span></button><a class="mi" role="menuitem" href="${P(pid).keyPage}" target="_blank" rel="noopener">${I('arrow-up-right')}<span>Open ${P(pid).name}’s key page</span></a><button class="mi danger" role="menuitem" data-key-remove="${pid}">${I('trash-2')}<span>Remove key</span></button></div>`, P(pid).name);
}

function wantMenu(key, anchor, confirm) {
  const [fid, wid] = key.split('|'), f = fileOf(fid), wl = wlOf(f, wid);
  if (!confirm) {
    openPop(anchor, `<div class="vb-mlist" role="menu"><button class="mi danger" role="menuitem" data-bad-ask="${key}">${I('thumbs-down')}<span>Mark bad</span></button><button class="mi" role="menuitem" data-search="${key}" data-tried>${I('history')}<span>What was tried</span></button></div>`, `${wl.lang} on ${f.ep ? `E0${f.ep}` : f.target}`);
    return;
  }
  openPop(anchor, `<div class="vb-confirm"><p><b>Mark this ${esc(wl.lang)} subtitle bad?</b></p><p class="t2">SlipStream deletes the Sidecar, never picks this subtitle again for this file, and searches again.</p><div class="vb-confirm-a"><button class="btn sm ghost" data-pop-close>Cancel</button><button class="btn sm fail" data-bad="${key}">Mark bad</button></div></div>`, 'Mark bad');
}

const TOKENS = {
  when: (w) => [['always', 'Always', 'Searched for every file.'], ['noAudio', `Without ${w.lang} audio`, `Only when no audio track is in ${w.lang}. Otherwise Not needed.`]],
  kind: () => [['full', 'full', 'Every line of dialogue.'], ['forced', 'forced only', 'Only foreign-language lines and on-screen text.']],
  hi: () => [['avoid', 'hearing impaired avoided', 'Picks ones without sound descriptions when there is a choice.'], ['allow', 'hearing impaired allowed', 'Either kind, ranked the same.'], ['prefer', 'hearing impaired preferred', 'Picks ones with sound descriptions when there is a choice.']],
  lang: (w) => D.languages.map((l) => [l, l, '']),
};

function tokenMenu(wid, field, anchor) {
  const w = st.draft.find((x) => x.id === wid);
  const opts = TOKENS[field](w);
  const html = `<div class="vb-mlist${field === 'lang' ? ' vb-mgrid' : ''}" role="menu">${opts.map(([v, l, sub]) => `<button class="mi${sub ? ' mi-2' : ''}" role="menuitemradio" aria-checked="${w[field] === v}" data-tok-set="${wid}|${field}|${esc(v)}">${w[field] === v ? I('check') : '<span class="ic"></span>'}<span class="mi-txt"><span>${esc(l)}</span>${sub ? `<span class="t3">${esc(sub)}</span>` : ''}</span></button>`).join('')}</div>`;
  openPop(anchor, html, { when: 'When', kind: 'Which lines', hi: 'Hearing impaired', lang: 'Language' }[field]);
}

// ───────────────────────────── profile

const FILES = { cinema: '428 library files', tv: 'every episode file of 61 series' };

function sentenceHTML(w, state) {
  const tk = (field, label) => `<button class="tk" data-tok="${w.id}|${field}" aria-haspopup="menu" ${state === 'gone' ? 'disabled' : ''}>${esc(label)}</button>`;
  const when = w.when === 'always' ? tk('when', 'Always') : tk('when', `Without ${w.lang} audio,`);
  return `${when} want ${tk('lang', w.lang)} subtitles, ${tk('kind', KIND_TXT[w.kind])}, <span class="nw">${tk('hi', HI_TXT[w.hi])}.</span>`;
}

function diffLines() {
  const saved = st.saved[st.profile], draft = st.draft, files = FILES[st.profile];
  const out = [];
  const desc = (w) => `${w.lang}${w.kind === 'forced' ? ' forced' : ''}`;
  for (const d of draft) {
    const s = saved.find((x) => x.id === d.id);
    if (d.gone) { if (s) out.push(['minus', `Drops ${desc(s)}. The ${s.lang} Sidecars SlipStream fetched for it are deleted from ${files}.`]); continue; }
    if (!s) { out.push(['plus', `Adds ${desc(d)}. SlipStream searches for it beside ${files}, starting now.`]); continue; }
    if (s.lang !== d.lang) { out.push(['minus', `Drops ${desc(s)}. Its fetched Sidecars are deleted.`]); out.push(['plus', `Adds ${desc(d)}. Searched for ${files}, starting now.`]); continue; }
    if (s.kind !== d.kind) out.push(['refresh-cw', `${d.lang} becomes ${KIND_TXT[d.kind]}. Every file is searched again for it, and the old Sidecars are replaced once a new one passes sync.`]);
    if (s.hi !== d.hi) out.push(['refresh-cw', `${d.lang}: ${HI_TXT[d.hi]}. Applies from the next search. Nothing is deleted.`]);
    if (s.when !== d.when) out.push(['refresh-cw', d.when === 'always' ? `${d.lang} is searched for files that have ${d.lang} audio too.` : `${d.lang} becomes Not needed on files that have ${d.lang} audio. Their fetched Sidecars are deleted.`]);
  }
  return out;
}

function dupes() {
  const seen = new Map(), out = [];
  for (const d of st.draft.filter((x) => !x.gone)) {
    const k = `${d.lang}|${d.kind}`;
    if (seen.has(k)) out.push(`${d.lang} ${KIND_TXT[d.kind]} is wanted twice.`); else seen.set(k, 1);
  }
  return out;
}

function profileView() {
  const p = D.profiles[st.profile], saved = st.saved[st.profile];
  const lines = diffLines(), dup = dupes();
  const items = st.draft.map((w) => {
    const s = saved.find((x) => x.id === w.id);
    const state = w.gone ? 'gone' : !s ? 'new' : JSON.stringify({ ...s }) !== JSON.stringify({ id: w.id, lang: w.lang, kind: w.kind, hi: w.hi, when: w.when }) ? 'changed' : '';
    return `<li class="wl wl-${state || 'same'}" data-wl="${w.id}">
      <p class="wl-s">${sentenceHTML(w, state)}</p>
      ${state === 'gone' ? `<button class="btn ghost sm" data-wl-undo="${w.id}">${I('undo-2')}Undo</button>` : `<button class="ib sm" data-wl-rm="${w.id}" aria-label="Remove ${esc(w.lang)} ${KIND_TXT[w.kind]}">${I('x')}</button>`}
    </li>`;
  }).join('');
  const dirty = lines.length > 0;
  return `<header class="vb-ph">${phone ? '' : `<h1>${esc(p.name)}</h1>`}<p class="vb-lede">${phone ? `<b>${esc(p.name)}</b> · ` : ''}${esc(p.contentType)} · ${p.targets.join(' and ')} · <span class="num">${p.works}</span> Works${st.profile === 'cinema' ? `, ${FILES.cinema}` : ''}</p></header>
    <div class="seg-c vb-seg" role="radiogroup" aria-label="Profile">${Object.values(D.profiles).map((x) => `<button class="seg-o" role="radio" aria-checked="${x.id === st.profile}" data-prof="${x.id}">${esc(x.name)}</button>`).join('')}</div>
    <section class="vb-sec" aria-labelledby="wl-h">
      <div class="vb-sec-h"><h2 id="wl-h">Wanted Languages</h2><span class="t3">${st.draft.filter((x) => !x.gone).length} wanted</span></div>
      <p class="vb-hint">SlipStream looks for each one beside every library file under ${esc(p.name)}. Click a word to change it.</p>
      <ol class="wl-list">${items}</ol>
      <button class="btn vb-add" data-wl-add>${I('plus')}Add a language</button>
      <div class="wl-live" aria-live="polite">
        ${dup.map((d) => `<p class="wl-ln warn-t">${I('triangle-alert')}<span>${esc(d)} Remove one to save.</span></p>`).join('')}
        ${dirty ? lines.map(([ic, t]) => `<p class="wl-ln">${I(ic)}<span>${esc(t)}</span></p>`).join('') : `<p class="wl-ln t3">${I('info')}<span>No changes. Adding a language searches ${FILES[st.profile]}. Dropping one deletes the Sidecars SlipStream fetched for it.</span></p>`}
      </div>
      <div class="wl-act"${dirty ? '' : ' hidden'}><button class="btn ghost" data-wl-discard>Discard</button><button class="btn pri" data-wl-save ${dup.length ? 'disabled' : ''}>Save ${esc(p.name)}</button></div>
    </section>`;
}

// ───────────────────────────── wiring

function wire() {
  root.addEventListener('click', (e) => {
    const t = e.target;
    const q = (s) => t.closest(s);
    let el;
    if ((el = q('[data-go]'))) { e.preventDefault(); return C.go(el.dataset.go); }
    if ((el = q('[data-pop-close]'))) return closePop(true);
    if (st.pop && !q('#vb-pop') && !q('[data-tok],[data-wmenu],[data-pmenu]')) closePop();
    if ((el = q('[data-close]'))) return closeSheet();
    if ((el = q('[data-step-link]'))) {
      setStep(2);
      setTimeout(() => root.querySelector('.rs .gs-i[data-step="2"]')?.focus({ preventScroll: false }), 0);
      st.sheet.awaitReturn = true;
      return;
    }
    if ((el = q('[data-key-replace]'))) { st.sheet = { ...st.sheet, phase: 'idle', step: 3, replace: true }; paintSheet(); root.querySelector('#vb-key')?.focus(); return; }
    if ((el = q('[data-key-remove]'))) { closePop(); return removeKey(el.dataset.keyRemove); }
    if ((el = q('[data-key]'))) { closePop(); return openKey(el.dataset.key, root.querySelector(`.rk [data-key="${el.dataset.key}"]`) || el); }
    if ((el = q('[data-done]'))) return closeSheet();
    if ((el = q('[data-pmenu]'))) return st.pop?.anchor === el ? closePop() : provMenu(el.dataset.pmenu, el);
    if ((el = q('[data-mv]'))) return move(el.dataset.pid, +el.dataset.mv, true);

    if ((el = q('[data-toggle]'))) { const k = el.dataset.toggle; st.open.has(k) ? st.open.delete(k) : st.open.add(k); return paint(`[data-toggle="${k}"]`); }
    if ((el = q('[data-ep]'))) { const n = +el.dataset.ep; st.ep = st.ep === n ? null : n; return paint(`[data-ep="${n}"]`); }
    if ((el = q('[data-search]'))) { closePop(); return openSearch(el.dataset.search, root.querySelector(`[data-search="${el.dataset.search}"]`) || el, el.hasAttribute('data-tried')); }
    if ((el = q('[data-upload]'))) { st.upTarget = el.dataset.upload; const inp = root.querySelector('#vb-file'); inp.value = ''; return inp.click(); }
    if ((el = q('[data-wmenu]'))) return st.pop?.anchor === el ? closePop() : wantMenu(el.dataset.wmenu, el);
    if ((el = q('[data-bad-ask]'))) return wantMenu(el.dataset.badAsk, st.pop.anchor, true);
    if ((el = q('[data-bad]'))) return markBad(el.dataset.bad);
    if ((el = q('[data-pick]'))) return pick(el.dataset.pick);
    if ((el = q('[data-pick-done]'))) { pickT.forEach(clearTimeout); pickT = []; return closeSheet(); }
    if ((el = q('[data-queue]'))) { const c = searchCtx().live.find((k) => k.id === el.dataset.queue); toast(`Queued. ${P(c.provider).name} downloads it at ${scen.status[c.provider].resumesAt}, then Timing sync runs.`); return; }
    if ((el = q('[data-unblock]'))) {
      const [rel, prov] = el.dataset.unblock.split('|'), k = `${st.sheet.fileId}|${st.sheet.wid}`;
      st.blocked[k] = (st.blocked[k] || []).filter((b) => b.release !== rel);
      if (st.sheet.fileId === D.searchFor.file && st.sheet.wid === D.searchFor.want) { const c = D.candidates.find((x) => x.release === rel && x.provider === prov && x.status === 'blocked'); if (c) c.status = 'new'; }
      paintSheet(); toast('Unblocked. It ranks with the others again.'); return;
    }

    if ((el = q('[data-prof]'))) { st.profile = el.dataset.prof; st.draft = clone(st.saved[st.profile]); return paint(`[data-prof="${st.profile}"]`); }
    if ((el = q('[data-tok]'))) { const [wid, field] = el.dataset.tok.split('|'); return st.pop?.anchor === el ? closePop() : tokenMenu(wid, field, el); }
    if ((el = q('[data-tok-set]'))) {
      const [wid, field, v] = el.dataset.tokSet.split('|');
      st.draft.find((x) => x.id === wid)[field] = v;
      closePop();
      return paint(`[data-tok="${wid}|${field}"]`);
    }
    if ((el = q('[data-wl-rm]'))) {
      const id = el.dataset.wlRm, w = st.draft.find((x) => x.id === id);
      if (!st.saved[st.profile].some((x) => x.id === id)) st.draft = st.draft.filter((x) => x.id !== id); else w.gone = true;
      return paint(st.draft.some((x) => x.id === id) ? `[data-wl-undo="${id}"]` : '[data-wl-add]');
    }
    if ((el = q('[data-wl-undo]'))) { delete st.draft.find((x) => x.id === el.dataset.wlUndo).gone; return paint(`[data-wl-rm="${el.dataset.wlUndo}"]`); }
    if ((el = q('[data-wl-add]'))) {
      const used = new Set(st.draft.filter((x) => !x.gone).map((x) => x.lang));
      const lang = D.languages.find((l) => !used.has(l)) || 'German';
      const id = `new-${Date.now()}`;
      st.draft.push({ id, lang, kind: 'full', hi: 'avoid', when: 'always' });
      paint();
      const tok = root.querySelector(`[data-tok="${id}|lang"]`);
      tok.focus({ preventScroll: true });
      tok.scrollIntoView({ block: 'nearest' });
      return tokenMenu(id, 'lang', tok);
    }
    if ((el = q('[data-wl-discard]'))) { st.draft = clone(st.saved[st.profile]); return paint('[data-wl-add]'); }
    if ((el = q('[data-wl-save]'))) {
      const added = diffLines().filter(([ic]) => ic === 'plus').length;
      st.saved[st.profile] = st.draft.filter((x) => !x.gone).map(({ id, lang, kind, hi, when }) => ({ id, lang, kind, hi, when }));
      st.draft = clone(st.saved[st.profile]);
      paint('[data-wl-add]');
      return toast(`${D.profiles[st.profile].name} saved.${added ? ' Searching for the new language now.' : ''}`);
    }
  });

  root.addEventListener('pointerdown', (e) => { const h = e.target.closest('[data-handle]'); if (h && e.button === 0) dragStart(e, h); });

  root.addEventListener('keydown', (e) => {
    if (e.key === 'Escape') {
      if (st.pop) { e.preventDefault(); return closePop(true); }
      if (st.sheet) { e.preventDefault(); return closeSheet(); }
    }
    const h = e.target.closest && e.target.closest('[data-handle]');
    if (h && (e.key === 'ArrowUp' || e.key === 'ArrowDown')) { e.preventDefault(); return move(h.dataset.handle, e.key === 'ArrowUp' ? -1 : 1, false); }
    const m = e.target.closest && e.target.closest('.vb-mlist');
    if (m && (e.key === 'ArrowDown' || e.key === 'ArrowUp')) {
      e.preventDefault();
      const items = [...m.querySelectorAll('.mi')], i = items.indexOf(document.activeElement);
      items[(i + (e.key === 'ArrowDown' ? 1 : -1) + items.length) % items.length].focus();
    }
    if (e.key === 'Tab' && st.sheet) {
      const box = root.querySelector('.rs'), f = [...box.querySelectorAll('a[href], button:not([disabled]), input, summary, [tabindex="0"]')];
      if (!f.length) return;
      if (e.shiftKey && document.activeElement === f[0]) { e.preventDefault(); f[f.length - 1].focus(); }
      else if (!e.shiftKey && document.activeElement === f[f.length - 1]) { e.preventDefault(); f[0].focus(); }
    }
  });

  root.addEventListener('paste', (e) => { if (e.target.id === 'vb-key') setTimeout(() => onKeyInput(e.target, true), 0); });
  root.addEventListener('input', (e) => { if (e.target.id === 'vb-key' && e.inputType !== 'insertFromPaste') onKeyInput(e.target, false); });
  root.addEventListener('focusin', (e) => { if (e.target.id === 'vb-key') setStep(3); });

  // Coming back from the provider's tab means the key is probably on the clipboard: step 2 is done, focus the field.
  addEventListener('focus', () => {
    const sh = st.sheet;
    if (sh?.kind === 'key' && sh.awaitReturn && sh.phase !== 'ok') { sh.awaitReturn = false; setStep(3); root.querySelector('#vb-key')?.focus(); }
  });

  root.querySelector('#vb-file').addEventListener('change', (e) => { const file = e.target.files[0]; if (file && st.upTarget) startUpload(st.upTarget, file); });

  root.addEventListener('dragover', (e) => { const r = e.target.closest('.wr'); if (r && e.dataTransfer?.types.includes('Files')) { e.preventDefault(); r.classList.add('drop'); } });
  root.addEventListener('dragleave', (e) => { const r = e.target.closest('.wr'); if (r && !r.contains(e.relatedTarget)) r.classList.remove('drop'); });
  root.addEventListener('drop', (e) => {
    const r = e.target.closest('.wr'); if (!r) return;
    e.preventDefault(); r.classList.remove('drop');
    const file = e.dataTransfer.files[0], key = r.dataset.row;
    const w = fileOf(key.split('|')[0]).wants[key.split('|')[1]];
    if (file && w.state !== 'notneeded') startUpload(key, file);
  });
}
