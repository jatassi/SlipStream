// PROTOTYPE, throwaway. Variant A, "Provider cards": every Subtitle Provider is a self-contained card in a grid,
// a file's subtitles are a matrix of library files × Wanted Languages with a popover per cell.

let C, D, sh, root, S;
const esc = (s) => String(s ?? '').replace(/[&<>"]/g, (c) => ({ '&': '&amp;', '<': '&lt;', '>': '&gt;', '"': '&quot;' }[c]));
const I = (n, c = '') => `<i data-lucide="${n}" class="ic ${c}" aria-hidden="true"></i>`;
const fmt = (n) => Number(n).toLocaleString('en-US');
const st = (pid) => S.sc.status[pid];
const prov = (pid) => D.providers[pid];
const pname = (pid) => (pid === 'upload' ? 'Upload' : prov(pid).name);
const mono2 = { subdl: 'SD', subsource: 'SS', gestdown: 'GD' };
const SPIN = '<span class="vA-spin" aria-hidden="true"></span>';
const when = (iso) => new Date(iso).toLocaleString('en-US', { month: 'short', day: 'numeric', hour: '2-digit', minute: '2-digit', hour12: false, timeZone: 'UTC' });

function untilMidnight() {
  const n = D.now, m = Date.UTC(n.getUTCFullYear(), n.getUTCMonth(), n.getUTCDate() + 1);
  const min = Math.round((m - n) / 60000);
  return `${Math.floor(min / 60)} h ${min % 60} min`;
}
const mask = (v) => `${v.slice(0, 7)}…${v.slice(-4)}`;
const coversMovies = (pid) => prov(pid).covers !== 'TV only';
const readyFor = (kind) => S.sc.order.filter((id) => st(id).state === 'ready' && (kind === 'tv' || coversMovies(id)));

// ───────────────────────────── state

function init() {
  const w = C.screen === 'series' ? D.works.severance : D.works['dune-2'];
  const profile = D.profiles[w.profile];
  const rows = w.type === 'movie'
    ? w.files.map((f) => ({ key: f.id, title: f.target, file: f.name, size: f.size, note: f.derived, wants: f.wants }))
    : w.episodes.map((e) => ({ key: `ep-${e.n}`, title: `E${String(e.n).padStart(2, '0')}`, name: e.title, file: `Severance.S02E${String(e.n).padStart(2, '0')}.1080p.ATVP.WEB-DL.DDP5.1.H.264-FLUX.mkv`, size: `${(1.8 + ((e.n * 37) % 9) / 10).toFixed(1)} GB`, wants: e.wants, n: e.n }));
  return {
    sc: C.scenario,
    work: w, profile, rows,
    wants: Object.fromEntries(rows.map((r) => [r.key, structuredClone(r.wants)])),
    check: {}, replace: {}, steps: {},
    base: structuredClone(D.profiles.cinema.wanted),
    draft: structuredClone(D.profiles.cinema.wanted).map((x) => ({ ...x })),
    blocked: { 'f-dune-1080|w-es': [] },
    pop: null, modal: null, pick: null, confirmBad: false, upload: null,
  };
}

// ───────────────────────────── providers screen

function orderStrip() {
  const o = S.sc.order;
  const dotFor = (id) => { const s = st(id).state; return s === 'ready' ? 'draft' : s === 'paused' ? 'warn' : s === 'invalid' ? 'fail' : 'idle'; };
  const stateWord = (id) => ({ ready: 'ready', paused: 'paused', invalid: 'key rejected', off: 'no key yet' }[st(id).state]);
  const caps = o.map((id, i) => `<li class="vA-cap" draggable="true" data-drag="${id}">
      <span class="vA-grip" aria-hidden="true">${I('grip-vertical')}</span>
      <span class="vA-rank num" aria-hidden="true">${i + 1}</span>
      <span class="vA-cap-n"><b>${esc(pname(id))}</b><span class="t3">${C.phone ? stateWord(id) : ''}</span></span>
      <span class="dot dot-${dotFor(id)}" title="${stateWord(id)}"></span>
      <span class="vA-cap-btns">
        <button class="ib sm" data-move="${id}" data-d="-1" ${i === 0 ? 'disabled' : ''} aria-label="Move ${pname(id)} earlier" data-fid="mv-${id}-1">${I(C.phone ? 'arrow-up' : 'arrow-left')}</button>
        <button class="ib sm" data-move="${id}" data-d="1" ${i === o.length - 1 ? 'disabled' : ''} aria-label="Move ${pname(id)} later" data-fid="mv-${id}+1">${I(C.phone ? 'arrow-down' : 'arrow-right')}</button>
      </span>
    </li>${!C.phone && i < o.length - 1 ? `<li class="vA-cap-sep" aria-hidden="true">${I('chevron-right')}</li>` : ''}`).join('');
  return `<section class="vA-order" aria-labelledby="ord-h">
    <div class="vA-order-h"><h2 id="ord-h">Search order</h2><p class="t3">Ranks candidates after hash match, same group, and same source and resolution. Drag, or use the arrows.</p></div>
    <ol class="vA-caps" aria-label="Subtitle Providers in search order">${caps}</ol>
  </section>`;
}

function statusPill(pid) {
  const s = st(pid), p = prov(pid);
  if (S.check[pid]?.phase === 'checking') return `<span class="vA-pill">${SPIN}Checking</span>`;
  if (!p.needsKey) return `<span class="vA-pill live"><span class="dot dot-draft"></span>Ready, no key</span>`;
  if (s.state === 'ready') return `<span class="vA-pill live"><span class="dot dot-draft"></span>Ready</span>`;
  if (s.state === 'paused') return `<span class="chip-warn">${I('pause')}Paused</span>`;
  if (s.state === 'invalid') return `<span class="chip-fail">${I('key-round')}Key rejected</span>`;
  return `<span class="vA-pill quiet">${I('circle-dashed')}No key yet</span>`;
}

function steps(pid) {
  const p = prov(pid);
  const linkAt = pid === 'subdl' ? 1 : 0;
  const label = pid === 'subdl' ? 'Open SubDL’s API page' : 'Open subsource.net';
  return `<ol class="vA-steps">${p.steps.map((t, i) => `<li><span class="vA-sn num">${i + 1}</span><div><p>${esc(t)}</p>${i === linkAt ? `<a class="btn sm vA-keylink" href="${p.keyPage}" target="_blank" rel="noopener">${label}<span aria-hidden="true">↗</span><span class="sr"> (opens in a new tab)</span></a>` : ''}</div></li>`).join('')}</ol>`;
}

function verdict(pid) {
  const c = S.check[pid] || {}, p = prov(pid);
  if (c.phase === 'checking') return `<span class="vA-v-chk">${SPIN}Checking with ${p.name}…</span>`;
  if (c.phase === 'bad') return `<span class="vA-v-bad">${I('circle-x')}<span>${p.name} answered <span class="mono">${p.reject.status} “${esc(p.reject.body)}”</span>. Check you copied the whole key, or generate a new one.</span></span>`;
  return `<span class="t3">${esc(p.check)}</span>`;
}

function keyField(pid, cancel = false) {
  const p = prov(pid), c = S.check[pid] || {};
  return `<div class="vA-key">
    <label class="vA-lbl" for="k-${pid}">${p.name} API key</label>
    <div class="vA-in${c.phase ? ` is-${c.phase}` : ''}">${I('key-round')}<input id="k-${pid}" data-key="${pid}" data-fid="k-${pid}" class="mono" type="text" placeholder="Paste your key" autocomplete="off" spellcheck="false" value="${esc(c.value || '')}" aria-describedby="v-${pid}"${c.phase === 'bad' ? ' aria-invalid="true"' : ''}>${cancel ? `<button class="btn ghost sm" data-cancel-replace="${pid}">Cancel</button>` : ''}</div>
    <p class="vA-verdict" id="v-${pid}" aria-live="polite">${verdict(pid)}</p>
  </div>`;
}

function allowance(pid) {
  const a = prov(pid).allowance, s = st(pid);
  const left = a.limit - (s.used || 0);
  const span = pid === 'subdl' ? 'today' : 'in 24 hours';
  const paused = s.state === 'paused';
  const others = readyFor('tv').filter((id) => id !== pid).map(pname);
  const foot = paused
    ? `Out of ${a.unit}. Paused until <b>${esc(s.resumesAt)}</b>, in ${esc(s.resumesIn)}. ${others.length ? `${others.join(' and ')} ${others.length > 1 ? 'continue' : 'continues'} meanwhile.` : ''}`
    : pid === 'subdl' ? `Resets at midnight UTC, in ${untilMidnight()}.` : `Rolling 24 hours, up to ${a.perMinute} a minute.`;
  return `<div class="vA-allow${paused ? ' is-out' : ''}">
    <p class="vA-allow-n"><b class="num">${fmt(left)}</b><span class="t2">left of ${fmt(a.limit)} ${a.unit} ${span}</span></p>
    <div class="vA-meter" role="meter" aria-valuemin="0" aria-valuemax="${a.limit}" aria-valuenow="${left}" aria-label="${pname(pid)} ${a.unit} left"><span style="width:${(left / a.limit) * 100}%"></span></div>
    <p class="vA-allow-f ${paused ? 'warn-t' : 't3'}">${foot}</p>
  </div>`;
}

function keyRow(pid) {
  const s = st(pid), p = prov(pid);
  if (S.replace[pid]) return `<div class="vA-sec">${keyField(pid, true)}<p class="t3 vA-small">The current key keeps working until the new one passes.</p></div>`;
  return `<div class="vA-sec vA-keyrow">
    <div class="vA-keyrow-l"><span class="vA-lbl">Key</span><span><span class="mono">${esc(s.key)}</span><span class="t3"> · ${esc(s.account)}${s.plan ? `, ${esc(s.plan)}` : ''}</span></span></div>
    <div class="vA-keyrow-b"><button class="btn ghost sm" data-replace="${pid}">Replace</button><button class="btn ghost sm" data-remove="${pid}">Remove</button></div>
    ${s.justChecked ? `<p class="vA-v-ok">${I('check')}Key works. Account ${esc(s.account)}${s.plan ? `, ${esc(s.plan)} plan` : ''}. Checked just now.</p>` : `<p class="t3 vA-small">Last answer ${esc(s.lastOk)}.</p>`}
  </div>`;
}

function never(pid) { return `<p class="vA-never">${I('lock')}<span>${esc(prov(pid).never)}</span></p>`; }

function howTo(pid, open) {
  return `<details class="vA-how" data-how="${pid}"${open ? ' open' : ''}><summary>${I('chevron-right')}How to get a key</summary>${steps(pid)}</details>`;
}

function card(pid, i) {
  const p = prov(pid), s = st(pid);
  const head = `<header class="vA-card-h">
      <span class="vA-mono" aria-hidden="true">${mono2[pid]}<span class="vA-mono-r num" title="Search order">${i + 1}</span></span>
      <div class="vA-card-id"><h3>${p.name}</h3><p class="t3"><a href="https://${p.site}" target="_blank" rel="noopener">${p.site}</a><span class="s-sep">·</span>${p.covers}</p></div>
      ${statusPill(pid)}
    </header>`;
  let body;
  if (!p.needsKey) {
    const movieGap = readyFor('movie').length === 0;
    body = `<div class="vA-sec"><p class="vA-lead">Works without a key, so TV subtitles are searched from the first minute.</p></div>
      <div class="vA-sec"><dl class="vA-kv"><dt>Covers</dt><dd>TV only. ${movieGap ? '<span class="warn-t">No provider covers movies yet.</span>' : 'Movies come from SubDL and SubSource.'}</dd>
      <dt>Check</dt><dd>${esc(p.check)} Last answer ${esc(s.lastOk)}.</dd><dt>Allowance</dt><dd>No daily limit to track.</dd></dl></div>
      ${never(pid)}`;
  } else if (s.state === 'off') {
    body = `<div class="vA-sec"><p class="vA-sec-h">Get a free key <span class="t3">· about a minute</span></p>${steps(pid)}</div>
      <div class="vA-sec">${keyField(pid)}</div>
      ${never(pid)}
      <div class="vA-sec vA-allow-pre"><p class="t3">${I('gauge')}<span>A free key allows ${fmt(p.allowance.limit)} ${p.allowance.unit} a day${pid === 'subdl' ? ', reset at midnight UTC' : `, ${p.allowance.perMinute} a minute`}. The meter appears once the key works.</span></p></div>`;
  } else if (s.state === 'invalid') {
    body = `<div class="vA-sec vA-rej">
        <p class="vA-rej-h">${I('octagon-x')}<b>${p.name} rejected the key</b><span class="vA-rej-t">since ${esc(s.since)}</span></p>
        <p class="vA-quote mono">${p.reject.status} · “${esc(s.error)}”</p>
        <p class="vA-small vA-rej-d">${p.name} isn’t searched until a working key is pasted. Generate a new one on your profile and paste it below.</p>
      </div>
      <div class="vA-sec">${keyField(pid)}</div>
      ${howTo(pid, S.steps[pid] ?? true)}
      ${never(pid)}`;
  } else {
    body = `<div class="vA-sec">${allowance(pid)}</div>
      ${keyRow(pid)}
      ${howTo(pid, S.steps[pid] ?? false)}
      ${never(pid)}`;
  }
  return `<article class="vA-card is-${p.needsKey ? s.state : 'free'}" data-card="${pid}" aria-labelledby="c-${pid}">${head.replace('<h3>', `<h3 id="c-${pid}">`)}${body}</article>`;
}

function providersScreen() {
  const movies = readyFor('movie');
  const s = S.sc.status;
  let banner = '';
  if (s.subdl.state === 'off' && s.subsource.state === 'off') {
    banner = `<div class="vA-banner">${I('info')}<p><b>TV subtitles already run through Gestdown.</b> Movies have no Subtitle Provider until you add a SubDL or SubSource key. Each key is free and takes about a minute.</p></div>`;
  } else if (!movies.length) {
    const why = S.sc.order.filter(coversMovies).map((id) => (st(id).state === 'paused' ? `${pname(id)} is paused until ${st(id).resumesAt}` : st(id).state === 'invalid' ? `${pname(id)} rejected its key` : `${pname(id)} has no key`));
    banner = `<div class="vA-banner warn">${I('triangle-alert')}<p><b>Movie searches are waiting.</b> ${why.join(', and ')}. Gestdown continues, so TV subtitles still run.</p></div>`;
  }
  const head = C.phone ? '' : `<div class="vA-pageh"><h1>Subtitles</h1><p class="t2">Subtitle Providers SlipStream searches for every Wanted Language, and in which order.</p></div>`;
  return `${head}${banner}${orderStrip()}
    <div class="vA-grid">${S.sc.order.map((id, i) => card(id, i)).join('')}</div>`;
}

// ───────────────────────────── profile screen

const wantLabel = (w) => `${w.lang}${w.kind === 'forced' ? ' forced' : ''}`;

function seg(name, id, cur, opts) {
  return `<span class="seg-c vA-seg" role="radiogroup" aria-label="${name}">${opts.map(([v, l]) => `<button class="seg-o" role="radio" aria-checked="${cur === v}" data-set="${id}" data-f="${name}" data-v="${v}" data-fid="${id}-${name}-${v}">${l}</button>`).join('')}</span>`;
}

function profileRow(w) {
  const orig = S.base.find((b) => b.id === w.id);
  const changed = orig && ['lang', 'kind', 'hi', 'when'].some((k) => orig[k] !== w[k]);
  const dup = !w.removed && S.draft.some((o) => o !== w && !o.removed && o.lang === w.lang && o.kind === w.kind);
  if (w.removed) {
    return `<tr class="vA-pr is-removed"><td colspan="4"><span class="vA-strike">${esc(wantLabel(w))}</span><span class="t3"> · removed when you save</span></td><td class="vA-pr-x"><button class="btn ghost sm" data-undo="${w.id}" data-fid="undo-${w.id}">Undo</button></td></tr>`;
  }
  const flag = !orig ? '<span class="vA-tag">New</span>' : changed ? '<span class="vA-tag">Edited</span>' : '';
  return `<tr class="vA-pr${!orig ? ' is-new' : changed ? ' is-changed' : ''}">
    <td data-th="Language"><div class="vA-langc"><span class="vA-sel"><select data-lang="${w.id}" data-fid="lang-${w.id}" aria-label="Language">${D.languages.map((l) => `<option${l === w.lang ? ' selected' : ''}>${l}</option>`).join('')}</select>${I('chevron-down')}</span>${flag}</div>${dup ? `<p class="vA-dup">${I('triangle-alert')}Already wanted as ${w.kind === 'forced' ? 'forced only' : 'full'}.</p>` : ''}</td>
    <td data-th="Subtitle">${seg('kind', w.id, w.kind, [['full', 'Full'], ['forced', 'Forced only']])}</td>
    <td data-th="Hearing impaired">${seg('hi', w.id, w.hi, [['avoid', 'Avoid'], ['allow', 'Allow'], ['prefer', 'Prefer']])}</td>
    <td data-th="When">${seg('when', w.id, w.when, [['always', 'Always'], ['noAudio', `Only without ${w.lang} audio`]])}</td>
    <td class="vA-pr-x"><button class="ib" data-rm="${w.id}" aria-label="Remove ${esc(wantLabel(w))}" data-fid="rm-${w.id}">${I('trash-2')}</button></td>
  </tr>`;
}

function consequences() {
  const p = D.profiles.cinema, out = [];
  for (const w of S.draft) {
    const o = S.base.find((b) => b.id === w.id);
    if (!o && !w.removed) out.push(`Adding <b>${esc(wantLabel(w))}</b> searches all ${p.works} Works under Cinema, at both Targets.${w.when === 'noAudio' ? ` Files with ${esc(w.lang)} audio show it as Not needed.` : ''}`);
    else if (o && w.removed) out.push(`Removing <b>${esc(wantLabel(o))}</b> deletes the ${esc(o.lang)}${o.kind === 'forced' ? ' forced' : ''} Sidecars SlipStream fetched for these Works. Ones you uploaded stay.`);
    else if (o) {
      if (o.lang !== w.lang) out.push(`<b>${esc(wantLabel(o))}</b> becomes <b>${esc(wantLabel(w))}</b>. The ${esc(o.lang)} Sidecars SlipStream fetched are deleted and ${p.works} Works are searched for ${esc(w.lang)}.`);
      if (o.kind !== w.kind) out.push(`<b>${esc(w.lang)}</b> is now wanted ${w.kind === 'forced' ? 'as forced only. Full Sidecars SlipStream fetched for it are replaced on the next search' : 'in full. Works that only have a forced one are searched again'}.`);
      if (o.hi !== w.hi) out.push(`<b>${esc(wantLabel(w))}</b> now ${w.hi === 'prefer' ? 'prefers' : w.hi === 'avoid' ? 'avoids' : 'allows'} hearing-impaired subtitles. The next upgrade search may swap existing Sidecars.`);
      if (o.when !== w.when) out.push(w.when === 'noAudio' ? `<b>${esc(wantLabel(w))}</b> is only wanted without ${esc(w.lang)} audio. Files with it show Not needed and keep what they have.` : `<b>${esc(wantLabel(w))}</b> is wanted always. Files with ${esc(w.lang)} audio are searched too.`);
    }
  }
  return out;
}

function profileScreen() {
  const p = D.profiles.cinema;
  const notes = consequences();
  const tabs = `<div class="seg-c vA-tabs" role="tablist">${['Targets', 'Editions', 'Subtitles'].map((t) => `<button class="seg-o" role="tab" aria-selected="${t === 'Subtitles'}" aria-checked="${t === 'Subtitles'}" ${t === 'Subtitles' ? '' : `data-toast="${t} are not in this prototype"`}>${t}</button>`).join('')}</div>`;
  return `<div class="vA-pageh">${C.phone ? '' : '<h1>Cinema</h1>'}<p class="t2">Profile · ${p.contentType} · <span class="num">${p.works}</span> Works · Targets ${p.targets.join(', ')}</p></div>
    ${tabs}
    <section class="vA-panel" aria-labelledby="wl-h">
      <div class="vA-panel-h"><h2 id="wl-h">Wanted Languages</h2><p class="t3">Each one is searched for every library file under Cinema. “Only without that audio” skips files that already have the language as an audio track.</p></div>
      <table class="vA-ptable">
        <thead><tr><th scope="col">Language</th><th scope="col">Subtitle</th><th scope="col">Hearing impaired</th><th scope="col">When</th><th scope="col"><span class="sr">Remove</span></th></tr></thead>
        <tbody>${S.draft.map(profileRow).join('')}</tbody>
        <tfoot><tr><td colspan="5"><button class="btn ghost sm" data-add data-fid="add">${I('plus')}Add language</button></td></tr></tfoot>
      </table>
      ${notes.length ? `<div class="vA-conseq" role="region" aria-label="What saving does">
        <p class="vA-sec-h">${I('info')}What saving does</p>
        <ul>${notes.map((n) => `<li>${n}</li>`).join('')}</ul>
        <div class="vA-conseq-b"><button class="btn ghost" data-discard>Discard</button><button class="btn pri" data-save>Save changes</button></div>
      </div>` : ''}
    </section>`;
}

// ───────────────────────────── a Work's subtitles

const wants = () => S.profile.wanted;

function wantHead(w) {
  const bits = [D.kindLabels[w.kind], w.hi === 'avoid' ? 'avoid HI' : w.hi === 'prefer' ? 'prefer HI' : 'HI allowed', w.when === 'always' ? 'always' : `only without ${w.lang} audio`];
  return `<div class="vA-wh"><b>${esc(w.lang)}</b><span class="t3">${bits.join(' · ')}</span></div>`;
}

function waitingNote() {
  if (S.work.type === 'series') return readyFor('tv').length ? null : 'Waits for a Subtitle Provider';
  if (readyFor('movie').length) return null;
  const s = S.sc.status;
  if (s.subdl.state === 'off' && s.subsource.state === 'off') return 'Waits for a movie Subtitle Provider';
  return `Waits until ${s.subdl.resumesAt || 'a key works'}`;
}

function cellLine(x, w) {
  if (x.searching) return '<span class="vA-live"><span class="dot dot-draft"></span>Searching now</span>';
  if (x.state === 'final' || x.state === 'present') {
    const s = x.sub;
    const why = s.reasons.length ? D.reasonLabels[s.reasons[0]].label : s.provider === 'upload' ? `synced ${s.sync.split(',')[0]}` : 'no release match';
    return `${esc(pname(s.provider))} · ${esc(why)}${s.by === 'ai' ? ' · AI' : ''}`;
  }
  if (x.state === 'embedded') return esc(x.track.split(' · ').slice(0, 2).join(' · '));
  if (x.state === 'missing') return esc(waitingNote() || x.next || x.note || 'Searched on schedule');
  if (x.state === 'nosync') return `${x.tried} tried${x.note && x.note.includes('%') ? `, best ${x.note.match(/(\d+)%/)[1]}%` : ''}`;
  return `${esc(w.lang)} audio on file`;
}

function stateBadge(state, big = false) {
  const s = D.subStates[state];
  return `<span class="vA-st st-${state}${big ? ' big' : ''}">${I(s.icon)}${esc(s.label)}</span>`;
}

function matrix() {
  const ws = wants();
  const series = S.work.type === 'series';
  const head = `<thead><tr><th scope="col" class="vA-mx-corner">${series ? 'Episode' : 'Library file'}</th>${ws.map((w) => `<th scope="col">${wantHead(w)}</th>`).join('')}</tr></thead>`;
  const rows = S.rows.map((r) => `<tr>
      <th scope="row" class="vA-mx-f"><div class="vA-ft"><b>${esc(r.title)}</b>${r.name ? `<span class="t2">${esc(r.name)}</span>` : ''}<span class="num t3 vA-fs">${esc(r.size)}</span></div><p class="mono t3 vA-fn" title="${esc(r.file)}">${esc(r.file)}</p>${r.note ? `<p class="t3 vA-fnote">${I('replace')}${esc(r.note.split('.')[0])}</p>` : ''}</th>
      ${ws.map((w) => { const x = S.wants[r.key][w.id]; return `<td><button class="vA-cell" data-cell="${r.key}|${w.id}" aria-haspopup="dialog" aria-label="${esc(w.lang)}${w.kind === 'forced' ? ' forced' : ''} for ${esc(r.title)}: ${D.subStates[x.state].label}">${stateBadge(x.state)}<span class="vA-cell-d">${cellLine(x, w)}</span></button></td>`; }).join('')}
    </tr>`).join('');
  return `<div class="vA-mx-wrap"><table class="vA-mx${series ? ' is-series' : ''}"><caption class="sr">Subtitle state per library file and Wanted Language</caption>${head}<tbody>${rows}</tbody></table></div>`;
}

function phoneMatrix() {
  const ws = wants();
  return S.rows.map((r) => `<article class="vA-pcard">
      <header class="vA-pcard-h"><div><b>${esc(r.title)}</b>${r.name ? ` <span class="t2">${esc(r.name)}</span>` : ''}</div><span class="num t3">${esc(r.size)}</span></header>
      <p class="mono t3 vA-fn">${esc(r.file)}</p>
      <ul class="vA-plist">${ws.map((w) => { const x = S.wants[r.key][w.id]; return `<li><button class="vA-prow" data-cell="${r.key}|${w.id}" aria-haspopup="dialog"><span class="vA-wh"><b>${esc(w.lang)}${w.kind === 'forced' ? ' forced' : ''}</b><span class="t3">${cellLine(x, w)}</span></span>${stateBadge(x.state)}${I('chevron-right', 'vA-chev')}</button></li>`; }).join('')}</ul>
    </article>`).join('');
}

function legend() {
  return `<ul class="vA-legend" aria-label="Legend">${Object.entries(D.subStates).map(([k, s]) => `<li title="${esc(s.means)}">${stateBadge(k)}</li>`).join('')}</ul>`;
}

function workBanner() {
  const s = S.sc.status, series = S.work.type === 'series';
  if (series) {
    if (s.subdl.state === 'paused' || s.subsource.state === 'invalid') return `<div class="vA-banner warn">${I('triangle-alert')}<p><b>Gestdown carries TV for now.</b> SubDL is paused until ${esc(s.subdl.resumesAt)} and SubSource rejected its key.</p><button class="btn sm" data-go="providers">Open Subtitle Providers</button></div>`;
    if (s.subdl.state === 'off' && s.subsource.state === 'off') return `<div class="vA-banner">${I('info')}<p><b>Gestdown searches these without a key.</b> SubDL and SubSource add more candidates once their keys are in.</p><button class="btn sm" data-go="providers">Add keys</button></div>`;
    return '';
  }
  if (s.subdl.state === 'off' && s.subsource.state === 'off') return `<div class="vA-banner">${I('info')}<p><b>No Subtitle Provider covers movies yet.</b> Gestdown is TV only. Add a free SubDL or SubSource key and SlipStream searches the missing ones at once.</p><button class="btn sm pri" data-go="providers">Add a key</button></div>`;
  if (!readyFor('movie').length) return `<div class="vA-banner warn">${I('triangle-alert')}<p><b>Movie searches wait until ${esc(s.subdl.resumesAt)}.</b> SubDL is out of downloads and SubSource rejected its key. Sidecars already here are untouched.</p><button class="btn sm" data-go="providers">Open Subtitle Providers</button></div>`;
  return '';
}

function workScreen() {
  const w = S.work, series = w.type === 'series';
  const counts = {};
  for (const r of S.rows) for (const x of Object.values(S.wants[r.key])) counts[x.state] = (counts[x.state] || 0) + 1;
  const need = (counts.missing || 0) + (counts.nosync || 0);
  const summary = need ? `${need} ${need === 1 ? 'want needs' : 'wants need'} a subtitle` : 'Every Wanted Language is met';
  const art = C.shell.art(w, C.phone ? 'poster-sm' : 'poster-md');
  const head = `<div class="vA-wk">
      <div class="vA-wk-bd" aria-hidden="true"><img src="${w.art.backdrop}" alt="" onerror="this.remove()"></div>
      ${art}
      <div class="vA-wk-id">
        <h1>${esc(w.title)}${series ? ` <span class="t2">Season ${w.season}</span>` : ''}</h1>
        <p class="t2"><span class="num">${w.year}</span> · ${series ? 'Series' : `Movie · Edition ${esc(w.edition)}`} · Audio ${w.audio.join(', ')}</p>
        ${C.phone ? '' : `<div class="seg-c vA-tabs" role="tablist">${['Overview', series ? 'Episodes' : 'Files', 'Subtitles', 'History'].map((t) => `<button class="seg-o" role="tab" aria-selected="${t === 'Subtitles'}" aria-checked="${t === 'Subtitles'}" ${t === 'Subtitles' ? '' : `data-toast="${t} is not in this prototype"`}>${t}</button>`).join('')}</div>`}
      </div>
    </div>`;
  const ready = readyFor(series ? 'tv' : 'movie').map(pname);
  const order = ready.length ? `Searches ${ready.join(', then ')}.` : 'No Subtitle Provider is searching.';
  return `${head}
    ${workBanner()}
    <section class="vA-panel" aria-labelledby="subs-h">
      <div class="vA-panel-h vA-panel-hr">
        <div><h2 id="subs-h">Subtitles</h2><p class="t3">${summary}. Profile ${esc(S.profile.name)}, ${wants().length} Wanted Languages. ${order}</p></div>
        <span class="grow"></span>
        <button class="btn sm ghost" data-go="profile">${I('sliders-horizontal')}Edit Wanted Languages</button>
      </div>
      ${C.phone ? phoneMatrix() : matrix()}
      ${legend()}
    </section>`;
}

// ───────────────────────────── cell detail: popover on wide, sheet on phone

function cellCtx(key) {
  const [rk, wid] = key.split('|');
  const r = S.rows.find((x) => x.key === rk), w = wants().find((x) => x.id === wid);
  return { rk, wid, r, w, x: S.wants[rk][wid] };
}

function detailBody(key) {
  const { r, w, x } = cellCtx(key);
  const s = D.subStates[x.state];
  const title = `${w.lang}${w.kind === 'forced' ? ' forced' : ''} · ${r.title}${r.name ? ` ${r.name}` : ''}`;
  let kv = '';
  if (x.sub) {
    const sub = x.sub;
    const reasons = sub.reasons.length ? sub.reasons.map((k) => `<span class="vA-rc rc-${D.reasonLabels[k].tone}" title="${esc(D.reasonLabels[k].means)}">${esc(D.reasonLabels[k].label)}</span>`).join('') : '<span class="t3">No release match. Ranked by provider order, downloads and rating.</span>';
    kv = `<dl class="vA-kv">
      <dt>Source</dt><dd>${sub.provider === 'upload' ? 'Upload by Jackson' : esc(pname(sub.provider))}</dd>
      <dt>Release</dt><dd class="mono vA-brk">${esc(sub.release)}</dd>
      ${sub.file ? `<dt>Sidecar</dt><dd class="mono vA-brk">${esc(sub.file)}</dd>` : ''}
      <dt>Matched</dt><dd class="vA-rcs">${reasons}</dd>
      <dt>Timing sync</dt><dd>${esc(sub.sync[0].toUpperCase() + sub.sync.slice(1))}</dd>
      <dt>Made by</dt><dd>${sub.by === 'ai' ? '<span class="vA-rc rc-weak">AI-translated</span> <span class="t3">Searched again for a human-made one.</span>' : 'Human'}</dd>
      ${sub.at ? `<dt>Saved</dt><dd class="num">${when(sub.at)} UTC</dd>` : ''}
    </dl>`;
  } else if (x.state === 'embedded') {
    kv = `<dl class="vA-kv"><dt>Track</dt><dd class="mono">${esc(x.track)}</dd></dl>`;
  } else if (x.state === 'missing') {
    const wait = waitingNote();
    kv = `<dl class="vA-kv"><dt>Why</dt><dd>${esc(x.note || 'No candidate found yet.')}</dd><dt>Next</dt><dd>${x.searching ? '<span class="vA-live"><span class="dot dot-draft"></span>Searching now</span>' : esc(wait || x.next || 'Searched again on schedule.')}</dd></dl>`;
  } else if (x.state === 'nosync') {
    kv = `<dl class="vA-kv"><dt>Why</dt><dd>${esc(x.note)}</dd><dt>Next</dt><dd>${esc(waitingNote() || x.next || 'Searched again later.')}</dd></dl>`;
  } else {
    kv = `<dl class="vA-kv"><dt>Why</dt><dd>${esc(x.note || `This file has a ${w.lang} audio track.`)}</dd></dl>`;
  }
  let foot;
  if (S.confirmBad) {
    foot = `<div class="vA-confirm"><p>SlipStream deletes this Sidecar, blocks this release from ${esc(pname(x.sub.provider))} for this file, and searches again.</p><div class="vA-row"><button class="btn sm ghost" data-bad-cancel>Cancel</button><button class="btn sm fail" data-bad-go>${I('flag')}Mark bad</button></div></div>`;
  } else if (S.upload) {
    foot = `<div class="vA-upl-run">${S.upload.phase === 'sync' ? `${SPIN}<span>Timing sync of <span class="mono">${esc(S.upload.name)}</span> against the Speech Map…</span>` : `${I('circle-x')}<span>${esc(S.upload.msg)}</span>`}</div>`;
  } else {
    foot = `<div class="vA-acts">
      <button class="btn sm" data-search="${key}">${I('list-filter')}Search by hand</button>
      <label class="btn sm vA-upl">${I('upload')}Upload<input type="file" accept=".srt,.ass,.ssa,.vtt,.sub" data-upload="${key}" class="sr"></label>
      ${x.sub && x.sub.provider !== 'upload' ? `<button class="btn sm ghost vA-bad" data-bad>${I('flag')}Mark bad</button>` : ''}
    </div>
    <p class="vA-drop t3" data-drop="${key}">${I('file-up')}Or drop an .srt or .ass file here. It is Timing-synced like any other.</p>`;
  }
  return `<div class="vA-dh"><span class="t2">${esc(title)}</span>${stateBadge(x.state, true)}</div>
    <p class="vA-means t2">${esc(s.means)}</p>${kv}${foot}`;
}

function openCell(key, anchor) {
  S.pop = { key, anchor }; S.confirmBad = false; S.upload = null;
  drawPop();
}

function drawPop() {
  const layer = root.querySelector('.vA-layer');
  if (!S.pop) { layer.querySelector('.vA-pop')?.remove(); layer.querySelector('.vA-sheet')?.remove(); return; }
  const html = detailBody(S.pop.key);
  if (C.phone) {
    let sw = layer.querySelector('.vA-sheet');
    if (!sw) { sw = document.createElement('div'); sw.className = 'sheet-wrap vA-sheet'; sw.innerHTML = '<div class="sheet-scrim" data-close></div><div class="sheet" role="dialog" aria-modal="true" aria-label="Subtitle detail"></div>'; layer.append(sw); }
    sw.querySelector('.sheet').innerHTML = `<span class="grab" aria-hidden="true"></span>${html}<button class="sheet-x" data-close>Close</button>`;
  } else {
    let p = layer.querySelector('.vA-pop');
    const fresh = !p;
    if (!p) { p = document.createElement('div'); p.className = 'pop vA-pop'; p.setAttribute('role', 'dialog'); p.setAttribute('aria-label', 'Subtitle detail'); layer.append(p); }
    p.innerHTML = html;
    const a = root.querySelector(`[data-cell="${S.pop.key}"]`);
    const r = a.getBoundingClientRect(), rr = root.getBoundingClientRect();
    const wdt = p.offsetWidth;
    p.style.left = `${Math.min(Math.max(12, r.left - rr.left + r.width / 2 - wdt / 2), rr.width - wdt - 12)}px`;
    const h = p.offsetHeight, below = innerHeight - r.bottom, above = r.top - 64;
    p.style.top = `${(below < h + 16 && above > h + 8 ? r.top - rr.top - h - 8 : r.bottom - rr.top + 8)}px`;
    if (fresh && below < h + 16 && above <= h + 8) p.scrollIntoView({ block: 'nearest', behavior: 'smooth' });
    if (fresh) p.querySelector('button, label')?.focus({ preventScroll: true });
  }
  C.icons();
}

// ───────────────────────────── manual search modal

function candsFor(key) {
  const { rk, w } = cellCtx(key);
  let base;
  if (key === 'f-dune-1080|w-es') base = structuredClone(D.candidates);
  else if (S.work.type === 'movie') base = structuredClone(D.candidates).map((c) => ({ ...c, lang: `${w.lang}${w.kind === 'forced' ? ' forced' : ''}`, status: 'new', syncNote: null })).filter((c) => c.id !== 'c6');
  else {
    const e = String(S.rows.find((r) => r.key === rk).n).padStart(2, '0');
    base = [
      { id: 's1', provider: 'gestdown', release: `Severance.S02E${e}.1080p.ATVP.WEB-DL.DDP5.1.H.264-FLUX`, lang: w.lang, hi: true, reasons: ['group', 'hi'], downloads: 4210, rating: null, uploader: 'Addic7ed', status: 'new' },
      { id: 's2', provider: 'subdl', release: `Severance.S02E${e}.1080p.WEB.H264-SuccessfulCrab`, lang: w.lang, hi: false, reasons: ['source'], downloads: 9120, rating: 8.8, uploader: 'tvsubs', status: 'new' },
      { id: 's3', provider: 'subsource', release: `Severance.S02E${e}.720p.WEB.h264-ETHEL`, lang: w.lang, hi: false, reasons: [], downloads: 3302, rating: 8.1, uploader: 'kaito', status: 'new' },
      { id: 's4', provider: 'subdl', release: `Severance.S02E${e}.1080p.WEB.H264-SuccessfulCrab`, lang: w.lang, hi: false, reasons: ['source', 'ai'], downloads: 410, rating: null, uploader: 'SubDL AI', status: 'new' },
    ];
  }
  for (const b of S.blocked[key] || []) {
    const hit = base.find((c) => c.release === b.release && c.provider === b.provider);
    if (hit) Object.assign(hit, { status: 'blocked', syncNote: b.note });
    else base.push({ id: `b-${b.release}`, provider: b.provider, release: b.release, lang: b.lang, reasons: [], downloads: null, rating: null, uploader: '', status: 'blocked', syncNote: b.note });
  }
  return base;
}

function modalHtml() {
  const key = S.modal;
  const { r, w } = cellCtx(key);
  const kind = S.work.type === 'movie' ? 'movie' : 'tv';
  const all = candsFor(key);
  const usable = (c) => { const s = st(c.provider).state; return (s === 'ready' || s === 'paused') && (kind === 'tv' || coversMovies(c.provider)); };
  const skipped = S.sc.order.filter((id) => (kind === 'tv' || coversMovies(id)) && !['ready', 'paused'].includes(st(id).state));
  const notices = [
    ...skipped.map((id) => `<li>${I(st(id).state === 'invalid' ? 'key-round' : 'circle-dashed')}${pname(id)} not searched: ${st(id).state === 'invalid' ? `its key was rejected (${prov(id).reject.status} “${esc(prov(id).reject.body)}”)` : 'no key yet'}.</li>`),
    ...S.sc.order.filter((id) => st(id).state === 'paused').map((id) => `<li class="warn-t">${I('pause')}${pname(id)} is out of downloads until ${esc(st(id).resumesAt)}. Its candidates show, but can’t be downloaded yet.</li>`),
  ];
  const list = all.filter(usable);
  const live = list.filter((c) => c.status === 'new'), dead = list.filter((c) => c.status !== 'new');
  const row = (c, i) => {
    const dl = c.downloads != null ? `<span class="num">${fmt(c.downloads)}</span>` : '<span class="t3">–</span>';
    const rt = c.rating != null ? `<span class="num">${c.rating.toFixed(1)}</span>` : '<span class="t3">–</span>';
    const chips = c.reasons.map((k) => `<span class="vA-rc rc-${D.reasonLabels[k].tone}" title="${esc(D.reasonLabels[k].means)}">${esc(D.reasonLabels[k].label)}</span>`).join('') || '<span class="t3 vA-small">No release match</span>';
    const p = S.pick && S.pick.id === c.id ? S.pick : null;
    let act;
    if (p) act = p.phase === 'fail' ? `<span class="vA-pk fail-t">${I('circle-x')}${esc(p.msg)}</span>` : p.phase === 'ok' ? `<span class="vA-pk">${I('check')}${esc(p.msg)}</span>` : `<span class="vA-pk">${SPIN}${esc(p.msg)}</span>`;
    else if (st(c.provider).state === 'paused') act = `<span class="vA-pk warn-t vA-nw">${I('pause')}Paused until ${esc(st(c.provider).resumesAt)}</span>`;
    else if (c.status === 'failed-sync') act = `<button class="btn sm ghost" data-pick="${c.id}" ${S.pick ? 'disabled' : ''}>Try anyway</button>`;
    else if (c.status === 'blocked') act = `<button class="btn sm ghost" data-unblock="${esc(c.release)}" ${S.pick ? 'disabled' : ''}>Unblock</button>`;
    else act = `<button class="btn sm${i === 0 ? ' pri' : ''}" data-pick="${c.id}" ${S.pick ? 'disabled' : ''}>${I('download')}Download</button>`;
    const why = c.status === 'failed-sync' ? `<span class="vA-dead">${I('audio-waveform')}Failed Timing sync. ${esc(c.syncNote)}</span>` : c.status === 'blocked' ? `<span class="vA-dead">${I('ban')}Blocked for this file. ${esc(c.syncNote)}</span>` : '';
    return `<tr class="${c.status === 'new' ? '' : 'is-dead'}${p ? ' is-picking' : ''}">
      <td class="vA-rk num">${c.status === 'new' ? i + 1 : '–'}</td>
      <td class="vA-rel"><p class="mono">${esc(c.release)}</p><p class="t3 vA-small">${esc(c.lang)}${c.hi ? ' · HI' : ''}${c.uploader ? ` · by ${esc(c.uploader)}` : ''}</p>${why}</td>
      <td>${esc(pname(c.provider))}</td>
      <td><div class="vA-rcs">${chips}</div></td>
      <td class="vA-nums">${dl}<span class="t3 vA-small">${c.rating != null ? `rated ${c.rating.toFixed(1)}` : 'no rating'}</span></td>
      <td class="vA-act">${act}</td>
    </tr>`;
  };
  const empty = !list.length ? `<div class="vA-empty">${I('search-x')}<p><b>Nothing to search.</b> ${kind === 'movie' ? 'No Subtitle Provider that covers movies is ready.' : 'No Subtitle Provider is ready.'}</p><button class="btn sm pri" data-go="providers">Open Subtitle Providers</button></div>` : '';
  return `<div class="vA-modal"><div class="vA-scrim" data-close></div>
    <div class="vA-box" role="dialog" aria-modal="true" aria-labelledby="ms-h">
      <header class="vA-box-h"><div><h2 id="ms-h">Search ${esc(w.lang)}${w.kind === 'forced' ? ' forced' : ''} by hand</h2><p class="t2">${esc(S.work.title)} · ${esc(r.title)}${r.name ? ` ${esc(r.name)}` : ''} · <span class="mono">${esc(r.file)}</span></p></div><button class="ib" data-close aria-label="Close">${I('x')}</button></header>
      <p class="vA-rule t3">Ranked by hash match, then same group, then same source and resolution, then your search order, then downloads and rating. AI-translated ranks below every human-made one. Each pick is Timing-synced against this file’s Speech Map, unless it is a hash match.</p>
      ${notices.length ? `<ul class="vA-notes">${notices.join('')}</ul>` : ''}
      ${empty || `<div class="vA-cands-wrap"><table class="vA-cands"><thead><tr><th scope="col">#</th><th scope="col">Release</th><th scope="col">Provider</th><th scope="col">Why it ranks here</th><th scope="col">Downloads</th><th scope="col"><span class="sr">Action</span></th></tr></thead>
        <tbody>${live.map(row).join('')}${dead.length ? `<tr class="vA-sep"><td colspan="6">Tried before or blocked</td></tr>${dead.map(row).join('')}` : ''}</tbody></table></div>`}
    </div></div>`;
}

function drawModal() {
  const layer = root.querySelector('.vA-layer');
  layer.querySelector('.vA-modal')?.remove();
  if (!S.modal) return;
  layer.insertAdjacentHTML('beforeend', modalHtml());
  C.icons();
}

function pick(id) {
  const key = S.modal, c = candsFor(key).find((x) => x.id === id);
  const set = (phase, msg) => { S.pick = { id, phase, msg }; drawModal(); };
  set('dl', `Downloading from ${pname(c.provider)}…`);
  setTimeout(() => {
    if (c.reasons.includes('hash')) return done('Hash match, no sync needed');
    set('sync', 'Timing sync against the Speech Map…');
    setTimeout(() => {
      if (c.status === 'failed-sync') { set('fail', 'Alignment 38%, the gate is 70%. Not saved.'); setTimeout(() => { S.pick = null; drawModal(); }, 2200); return; }
      done('Synced, +120 ms');
    }, 1300);
  }, 700);
  function done(msg) {
    set('ok', msg);
    setTimeout(() => {
      const { rk, wid, r, w } = cellCtx(key);
      const human = !c.reasons.includes('ai') && !c.reasons.includes('machine');
      const final = human && (c.reasons.includes('hash') || c.reasons.includes('group'));
      S.wants[rk][wid] = { state: final ? 'final' : 'present', sub: { provider: c.provider, release: c.release, file: `${r.file.replace(/\.mkv$/, '')}.${w.lang.slice(0, 2).toLowerCase()}${w.kind === 'forced' ? '.forced' : ''}.srt`, reasons: c.reasons.filter((k) => k !== 'hi'), sync: msg.startsWith('Hash') ? 'skipped, hash match' : '+120 ms, framerate unchanged', by: human ? 'human' : 'ai', at: D.now.toISOString() } };
      S.modal = null; S.pick = null; S.pop = null;
      drawModal(); drawPop(); drawBody();
      C.shell.toast(root, `${w.lang} for ${r.title} saved from ${pname(c.provider)}.`);
    }, 800);
  }
}

// ───────────────────────────── upload and mark bad

function upload(key, name) {
  S.upload = { phase: 'sync', name };
  drawPop();
  setTimeout(() => {
    if (/bad/i.test(name)) { S.upload = { phase: 'fail', msg: `${name} failed Timing sync. Alignment 22%, the gate is 70%. Not saved.` }; drawPop(); setTimeout(() => { S.upload = null; drawPop(); }, 2600); return; }
    const { rk, wid, r, w } = cellCtx(key);
    S.wants[rk][wid] = { state: 'present', sub: { provider: 'upload', release: name, file: `${r.file.replace(/\.mkv$/, '')}.${w.lang.slice(0, 2).toLowerCase()}${w.kind === 'forced' ? '.forced' : ''}.srt`, reasons: [], sync: '+310 ms, framerate unchanged', by: 'human', at: D.now.toISOString() } };
    S.upload = null;
    drawBody(); drawPop();
    C.shell.toast(root, `${name} synced and saved as a Sidecar.`);
  }, 1400);
}

function markBad() {
  const key = S.pop.key, { rk, wid, w, r } = cellCtx(key), sub = S.wants[rk][wid].sub;
  (S.blocked[key] ||= []).push({ release: sub.release, provider: sub.provider, lang: w.lang, note: 'Marked bad just now by Jackson.' });
  S.wants[rk][wid] = { state: 'missing', searching: true, note: `Marked bad just now. ${sub.release} is blocked for this file.` };
  S.confirmBad = false;
  drawBody(); drawPop();
  C.shell.toast(root, `Sidecar removed. Searching ${w.lang} for ${r.title} again.`);
  setTimeout(() => {
    const x = S.wants[rk][wid];
    if (x.state !== 'missing' || !x.searching) return;
    S.wants[rk][wid] = { ...x, searching: false, next: 'Searched just now. Next try tonight at 03:00.' };
    drawBody(); if (S.pop?.key === key) drawPop();
  }, 3500);
}

// ───────────────────────────── key checks

const timers = {};
function runCheck(pid, value) {
  clearTimeout(timers[pid]);
  if (!value.trim()) { S.check[pid] = null; paintVerdict(pid); return; }
  S.check[pid] = { phase: 'checking', value };
  paintVerdict(pid);
  timers[pid] = setTimeout(() => {
    const cur = root.querySelector(`[data-key="${pid}"]`);
    if (!cur || cur.value !== value) return;
    if (/bad/i.test(value) || value.trim().length < 12) { S.check[pid] = { phase: 'bad', value }; paintVerdict(pid); return; }
    const was = st(pid).state;
    S.check[pid] = null; S.replace[pid] = false;
    S.sc.status[pid] = { key: mask(value.trim()), state: 'ready', account: 'jackson', plan: pid === 'subdl' ? 'Free' : null, used: 0, lastOk: 'just now', justChecked: true };
    drawBody();
    C.shell.toast(root, was === 'ready' ? `${pname(pid)} key replaced.` : `${pname(pid)} is ready. Movie searches start now.`);
  }, 900);
}

function paintVerdict(pid) {
  const card = root.querySelector(`[data-card="${pid}"]`);
  if (!card) return;
  const box = card.querySelector('.vA-in');
  box.className = `vA-in${S.check[pid]?.phase ? ` is-${S.check[pid].phase}` : ''}`;
  const inp = box.querySelector('input');
  if (S.check[pid]?.phase === 'bad') inp.setAttribute('aria-invalid', 'true'); else inp.removeAttribute('aria-invalid');
  card.querySelector('.vA-verdict').innerHTML = verdict(pid);
  const pill = card.querySelector('.vA-card-h > :last-child');
  pill.outerHTML = statusPill(pid);
  C.icons();
}

// ───────────────────────────── render + wiring

function drawBody() {
  const fid = document.activeElement?.dataset?.fid;
  const b = root.querySelector('.vA-body');
  b.innerHTML = C.screen === 'providers' ? providersScreen() : C.screen === 'profile' ? profileScreen() : workScreen();
  C.icons();
  if (fid) root.querySelector(`[data-fid="${fid}"]`)?.focus({ preventScroll: true });
}

function move(pid, d) {
  const o = S.sc.order, i = o.indexOf(pid), j = i + d;
  if (j < 0 || j >= o.length) return;
  [o[i], o[j]] = [o[j], o[i]];
  drawBody();
  C.shell.toast(root, `Search order: ${o.map(pname).join(', ')}.`);
}

function closeAll() {
  if (S.modal) { S.modal = null; S.pick = null; drawModal(); return; }
  if (S.pop) { const k = S.pop.key; S.pop = null; drawPop(); root.querySelector(`[data-cell="${k}"]`)?.focus({ preventScroll: true }); }
}

function wire() {
  root.addEventListener('click', (e) => {
    const t = e.target;
    const q = (s) => t.closest(s);
    let el;
    if ((el = q('[data-go]'))) { C.go(el.dataset.go); return; }
    if ((el = q('[data-move]'))) { move(el.dataset.move, +el.dataset.d); return; }
    if ((el = q('[data-replace]'))) { S.replace[el.dataset.replace] = true; S.check[el.dataset.replace] = null; drawBody(); root.querySelector(`[data-key="${el.dataset.replace}"]`)?.focus(); return; }
    if ((el = q('[data-cancel-replace]'))) { S.replace[el.dataset.cancelReplace] = false; S.check[el.dataset.cancelReplace] = null; drawBody(); return; }
    if ((el = q('[data-remove]'))) { const pid = el.dataset.remove; S.sc.status[pid] = { key: null, state: 'off' }; S.steps[pid] = undefined; drawBody(); C.shell.toast(root, `${pname(pid)} key removed. ${pname(pid)} is no longer searched.`); return; }
    if ((el = q('[data-set]'))) { const w = S.draft.find((x) => x.id === el.dataset.set); w[el.dataset.f] = el.dataset.v; drawBody(); return; }
    if ((el = q('[data-rm]'))) { const i = S.draft.findIndex((x) => x.id === el.dataset.rm); if (S.base.some((b) => b.id === el.dataset.rm)) S.draft[i].removed = true; else S.draft.splice(i, 1); drawBody(); root.querySelector('[data-fid="add"]')?.focus(); return; }
    if ((el = q('[data-undo]'))) { S.draft.find((x) => x.id === el.dataset.undo).removed = false; drawBody(); return; }
    if ((el = q('[data-add]'))) {
      const used = new Set(S.draft.filter((x) => !x.removed).map((x) => x.lang));
      const lang = D.languages.find((l) => !used.has(l)) || 'English';
      const id = `w-new-${Date.now()}`;
      S.draft.push({ id, lang, kind: 'full', hi: 'avoid', when: 'always' });
      drawBody(); root.querySelector(`[data-fid="lang-${id}"]`)?.focus(); return;
    }
    if (q('[data-discard]')) { S.draft = structuredClone(S.base); drawBody(); C.shell.toast(root, 'Changes discarded.'); return; }
    if (q('[data-save]')) {
      const adds = S.draft.filter((x) => !x.removed && !S.base.some((b) => b.id === x.id));
      S.draft = S.draft.filter((x) => !x.removed); S.base = structuredClone(S.draft);
      drawBody(); C.shell.toast(root, adds.length ? `Saved. Searching ${D.profiles.cinema.works} Works for ${adds.map(wantLabel).join(', ')}.` : 'Saved. Cinema’s Wanted Languages are updated.'); return;
    }
    if ((el = q('[data-cell]'))) { if (S.pop?.key === el.dataset.cell) { closeAll(); return; } openCell(el.dataset.cell, el); return; }
    if ((el = q('[data-search]'))) { S.modal = el.dataset.search; S.pop = null; drawPop(); drawModal(); root.querySelector('.vA-box [data-pick], .vA-box [data-close]')?.focus(); return; }
    if (q('[data-bad]')) { S.confirmBad = true; drawPop(); root.querySelector('[data-bad-go]')?.focus(); return; }
    if (q('[data-bad-cancel]')) { S.confirmBad = false; drawPop(); return; }
    if (q('[data-bad-go]')) { markBad(); return; }
    if ((el = q('[data-pick]'))) { pick(el.dataset.pick); return; }
    if ((el = q('[data-unblock]'))) { S.blocked[S.modal] = (S.blocked[S.modal] || []).filter((b) => b.release !== el.dataset.unblock); drawModal(); C.shell.toast(root, 'Unblocked for this file.'); return; }
    if (q('[data-close]')) { closeAll(); return; }
    if (S.pop && !C.phone && !q('.vA-pop')) { S.pop = null; drawPop(); }
  });
  root.addEventListener('change', (e) => {
    const t = e.target;
    if (t.dataset.lang) { S.draft.find((x) => x.id === t.dataset.lang).lang = t.value; drawBody(); }
    if (t.dataset.upload && t.files[0]) upload(t.dataset.upload, t.files[0].name);
  });
  root.addEventListener('paste', (e) => {
    const t = e.target;
    if (!t.dataset?.key) return;
    setTimeout(() => runCheck(t.dataset.key, t.value), 0);
  });
  root.addEventListener('input', (e) => {
    const t = e.target;
    if (!t.dataset?.key || e.inputType === 'insertFromPaste') return;
    clearTimeout(timers[`t-${t.dataset.key}`]);
    if (!t.value.trim()) { runCheck(t.dataset.key, ''); return; }
    timers[`t-${t.dataset.key}`] = setTimeout(() => runCheck(t.dataset.key, t.value), 450);
  });
  root.addEventListener('toggle', (e) => { const d = e.target; if (d.dataset?.how) S.steps[d.dataset.how] = d.open; }, true);

  let drag = null;
  root.addEventListener('dragstart', (e) => { const c = e.target.closest?.('[data-drag]'); if (!c) return; drag = c.dataset.drag; c.classList.add('is-drag'); e.dataTransfer.effectAllowed = 'move'; e.dataTransfer.setData('text/plain', drag); });
  root.addEventListener('dragover', (e) => { const c = e.target.closest?.('[data-drag]'); if (!c || !drag) return; e.preventDefault(); root.querySelectorAll('.vA-cap.is-over').forEach((x) => x.classList.remove('is-over')); if (c.dataset.drag !== drag) c.classList.add('is-over'); });
  root.addEventListener('drop', (e) => {
    const dz = e.target.closest?.('[data-drop]');
    if (dz) { e.preventDefault(); const f = e.dataTransfer.files[0]; if (f) upload(dz.dataset.drop, f.name); return; }
    const c = e.target.closest?.('[data-drag]'); if (!c || !drag) return;
    e.preventDefault();
    const o = S.sc.order, from = o.indexOf(drag), to = o.indexOf(c.dataset.drag);
    o.splice(from, 1); o.splice(to, 0, drag);
    drag = null; drawBody();
    C.shell.toast(root, `Search order: ${o.map(pname).join(', ')}.`);
  });
  root.addEventListener('dragend', () => { drag = null; root.querySelectorAll('.is-drag, .is-over').forEach((x) => x.classList.remove('is-drag', 'is-over')); });
  root.addEventListener('dragover', (e) => { if (e.target.closest?.('[data-drop]')) e.preventDefault(); });
  addEventListener('keydown', (e) => { if (e.key === 'Escape') closeAll(); });
  addEventListener('resize', () => { if (S.pop && !C.phone) drawPop(); });
}

export function render(ctx) {
  C = ctx; D = ctx.data; sh = ctx.shell; root = ctx.root;
  root.classList.add('vA', `vA-${ctx.screen}`);
  S = init();
  const crumbs = { providers: ['Settings', 'Subtitles'], profile: ['Settings', 'Profiles', 'Cinema'], movie: ['Library', 'Movies', 'Dune: Part Two'], series: ['Library', 'Series', 'Severance', 'Season 2'] }[ctx.screen];
  const active = { providers: 'subtitles', profile: 'profiles', movie: 'movies', series: 'series' }[ctx.screen];
  const phoneTitle = { providers: 'Subtitles', profile: 'Cinema', movie: 'Subtitles', series: 'Subtitles' }[ctx.screen];
  root.innerHTML = sh.AMBIENT + sh.page({ active, crumbs, phone: ctx.phone, phoneTitle, back: true, body: '<div class="vA-body"></div>' }) + sh.OVERLAYS + '<div class="vA-layer"></div>';
  drawBody();
  wire();
  if (ctx.screen === 'movie' || ctx.screen === 'series') sh.lightBy(root, S.work);
  document.title = `${phoneTitle === 'Subtitles' && ctx.screen !== 'providers' ? S.work.title : phoneTitle} · SlipStream`;
}
