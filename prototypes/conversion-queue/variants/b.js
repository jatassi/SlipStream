// PROTOTYPE variant B · Ledger: one run-ordered list of every Conversion, and a stepped backfill dialog.

let S, D, st, bf, root;
const PR = ['bump', 'new', 'rederive', 'backfill'];
const MIN = 60000;
const T = (iso) => new Date(iso).getTime();
const prLabel = (p) => D.priorities.find((x) => x.id === p).label;
const fmtT = (ms) => {
  const d = new Date(ms);
  const day = d.getDate() !== D.now.getDate() ? `${d.toLocaleDateString('en-US', { weekday: 'short' })} ` : '';
  return `${day}${d.toTimeString().slice(0, 5)}`;
};
const nextAt = (hhmm) => {
  const [h, m] = hhmm.split(':').map(Number);
  const d = new Date(D.now); d.setHours(h, m, 0, 0);
  if (d <= D.now) d.setDate(d.getDate() + 1);
  return d.getTime();
};
const conv = (id) => D.conversions.find((c) => c.id === id);
const leaf = (t) => t.levels[t.levels.length - 1];
const plural = (n, one, many) => `${n} ${n === 1 ? one : many || `${one}s`}`;

// ───────────────────────────── queue model

function initQueue() {
  const waiting = D.conversions.filter((c) => c.state.is === 'waiting' && c.state.reason !== 'no-capable-node');
  const entries = [
    ...waiting.map((c) => ({ id: c.id, kind: 'conv', c, prio: c.priority })),
    ...D.backfill.map((g) => ({ id: g.id, kind: 'group', g, prio: 'backfill' })),
  ];
  const at = (e) => (e.c && e.c.state.startsAt ? T(e.c.state.startsAt) : 9e15);
  entries.sort((a, b) => PR.indexOf(a.prio) - PR.indexOf(b.prio) || at(a) - at(b));
  st = { entries, open: new Set(), band: 'all', node: '', flash: null, drag: null };
}

function slotsNow() {
  const slots = {};
  for (const n of D.nodes) {
    if (n.status !== 'online') { slots[n.id] = []; continue; }
    const queued = D.conversions.filter((c) => c.state.is === 'waiting' && c.state.node === n.id && c.state.reason === 'queued').map((c) => T(c.state.startsAt)).sort((a, b) => a - b);
    const held = D.conversions.filter((c) => c.state.is === 'suspended' && c.state.node === n.id).length;
    const opens = n.activeHours ? nextAt(n.activeHours.from) : D.now.getTime() + 5 * MIN;
    slots[n.id] = queued.length ? queued.slice(0, n.cap - held) : Array(n.cap).fill(opens);
  }
  return slots;
}

function plan() {
  const slots = slotsNow(), res = {};
  for (const e of st.entries) {
    let n, dur, dep;
    if (e.kind === 'conv') {
      const s = e.c.state;
      n = s.node;
      if (s.reason === 'disk-space') { res[e.id] = { node: n, start: null, finish: null }; continue; }
      dur = s.finishesAt ? T(s.finishesAt) - T(s.startsAt) : 150 * MIN;
      dep = s.waitsOn && res[s.waitsOn];
    } else {
      n = Object.entries(e.g.nodes).sort((a, b) => b[1] - a[1])[0][0];
      dur = e.g.estHours * 60 * MIN;
    }
    const arr = slots[n].sort((a, b) => a - b);
    const start = Math.max(arr[0], dep ? dep.finish : 0);
    arr[0] = start + dur;
    res[e.id] = { node: n, start, finish: start + dur };
  }
  return res;
}

const runningNow = () => D.conversions.filter((c) => c.state.is === 'running' || c.state.is === 'suspended');
const failedNow = () => D.conversions.filter((c) => c.state.is === 'failed');
function noCapGroups() {
  const by = {};
  for (const c of D.conversions) if (c.state.is === 'waiting' && c.state.reason === 'no-capable-node') (by[c.state.cause] ||= []).push(c);
  return Object.entries(by).map(([cause, convs]) => {
    const offline = D.nodes.filter((n) => n.status !== 'online' && n.proven.includes(cause));
    const failedTrial = D.nodes.filter((n) => n.status === 'online' && n.trials.some((t) => t.cap === cause && t.result === 'failed'));
    const notified = convs.map((c) => T(c.state.since)).sort((a, b) => a - b)[0];
    return { cause, convs, offline, failedTrial, notified };
  });
}

function moveUp(id, toIndex = 0) {
  const before = plan();
  const i = st.entries.findIndex((e) => e.id === id);
  const [e] = st.entries.splice(i, 1);
  const was = e.prio;
  e.prio = 'bump';
  const moved = [e];
  const dep = e.kind === 'conv' && e.c.state.waitsOn && st.entries.find((x) => x.id === e.c.state.waitsOn);
  if (dep) { st.entries.splice(st.entries.indexOf(dep), 1); dep.prio = 'bump'; moved.unshift(dep); }
  const bumpEnd = st.entries.findIndex((x) => x.prio !== 'bump');
  const at = Math.min(toIndex, bumpEnd === -1 ? st.entries.length : bumpEnd);
  st.entries.splice(at, 0, ...moved);
  const after = plan();
  st.flash = id;
  const name = entryName(e);
  const b = before[id], a = after[id];
  const when = a.start ? `Estimated start ${fmtT(a.start)} on ${a.node}${b.start && b.start !== a.start ? `, was ${fmtT(b.start)}` : ''}.` : 'It starts when disk space frees.';
  S.toast(`${name} moved up from ${prLabel(was)}. ${when}${dep ? ' Its source\'s Conversion moved up with it.' : ''}`);
  render();
}

function entryName(e) {
  if (e.kind === 'group') return e.g.label;
  const w = S.W(e.c.work), il = S.itemLabel(e.c);
  return `${w.title}${il ? ` ${il.split(' · ')[0]}` : ''} · ${S.target(e.c.target).name}`;
}

// ───────────────────────────── queue cells

function kindCell(c) {
  const t = S.target(c.target);
  if (c.kind === 'adjust') return `<span class="cq-kind" title="Runs ${esc(t.name)}'s download steps on a copy. The torrent keeps seeding the original.">${S.I('sliders-horizontal')}<span>Adjust</span><span class="t3 ell">${esc((c.steps || []).join(', '))}</span></span>`;
  return `<span class="cq-kind" title="Makes ${esc(t.name)}'s file from ${esc(S.target(t.from).name)}">${S.I('replace')}<span>Derive</span><span class="t3 ell">from ${esc(S.target(t.from).name)}</span></span>`;
}
const esc = (s) => S.esc(s);

function workCell(c, { thumb = true, sub } = {}) {
  const w = S.W(c.work), il = S.itemLabel(c), line = sub || (il ? `<span class="t3 ell">${esc(il)}</span>` : '');
  return `<div class="c-work" role="cell">${thumb ? S.art(w) : ''}<span class="cq-wt"><b class="ell">${esc(w.title)}</b>${line}</span></div>`;
}

function waitChip(c, p) {
  const s = c.state;
  if (s.reason === 'queued') return `<span class="chip">${S.I('list-ordered')}Queued</span>`;
  if (s.reason === 'source-conversion') {
    const src = conv(s.waitsOn);
    return `<span class="chip" title="Waits for ${esc(S.target(src.target).name)} to finish its own Conversion">${S.I('link-2')}Its source first</span><span class="t3 cq-wsub">${esc(S.target(src.target).name)}, #${st.entries.findIndex((e) => e.id === src.id) + 1}</span>`;
  }
  if (s.reason === 'disk-space') return `<span class="chip cq-chip-disk" title="${esc(s.detail)}">${S.I('hard-drive')}Disk space</span><span class="t3 cq-wsub"><span class="num">${s.freeGB}</span> of <span class="num">${s.needGB} GB</span> free</span>`;
  if (s.reason === 'active-hours') return activeChip(p.node);
  return '';
}
function activeChip(nid) {
  const ah = S.node(nid).activeHours;
  return `<span class="chip">${S.I('moon')}Active hours</span><span class="t3 cq-wsub num">${ah.from}–${ah.to}</span>`;
}

const nodeCell = (nid) => (nid ? `<span class="cq-node mono">${esc(nid)}</span>` : '<span class="t3 cq-np">not placed</span>');

// ───────────────────────────── wide

function pulse(p) {
  const run = runningNow(), live = run.filter((c) => c.state.is === 'running').length, paused = run.length - live;
  const nc = noCapGroups().reduce((n, g) => n + g.convs.length, 0);
  const upNext = st.entries.reduce((n, e) => n + (e.kind === 'group' ? groupCount(e.g) : 1), 0);
  const end = Math.max(...Object.values(p).map((x) => x.finish || 0));
  return `<span class="s-live">${S.dot('draft')}${live} converting</span><span class="s-sep">·</span><span class="t2">${paused} paused</span><span class="s-sep">·</span><span class="s-fail">${failedNow().length} failed</span><span class="s-sep">·</span><span class="s-warn">${nc} need a capable Node</span><span class="s-sep">·</span><span class="t2"><span class="num">${upNext}</span> up next, clears about <span class="num">${fmtT(end)}</span></span>`;
}

function seg(name, opts, cur, label) {
  return `<div class="seg-c" role="radiogroup" aria-label="${label}">${opts.map(([v, l, n]) => `<button role="radio" class="seg-o" data-q-${name}="${v}" aria-checked="${cur === v}"><span>${esc(l)}</span>${n != null ? `<span class="seg-n num">${n}</span>` : ''}</button>`).join('')}</div>`;
}

function tools() {
  const nNeeds = failedNow().length + noCapGroups().length, nRun = runningNow().length;
  const nNext = st.entries.length;
  return `<div class="cq-tools">
    ${seg('band', [['all', 'All'], ['needs', 'Needs you', nNeeds], ['now', 'Converting', nRun], ['next', 'Up next', nNext]], st.band, 'Show')}
    ${seg('node', [['', 'All Nodes'], ...D.nodes.map((n) => [n.id, n.id])], st.node, 'Node')}
    <span class="grow"></span>
    <span class="t3 cq-today">Today <span class="num t2">${D.today.finished}</span> finished · <span class="num fail-t">${D.today.failed}</span> failed</span>
  </div>`;
}

const hidden = (nodes) => (st.node && !nodes.includes(st.node) ? ' hidden' : '');

function failRow(c) {
  const w = S.W(c.work), s = c.state, last = s.attempts[s.attempts.length - 1];
  const nodes = [...new Set(s.attempts.map((a) => a.node))];
  const open = st.open.has(c.id);
  const att = s.attempts.map((a, i) => {
    const prev = s.attempts[i - 1];
    const how = i === 0 ? 'First run' : prev.node === a.node ? 'Retry, same Node' : 'Another Node';
    return `<li class="cq-att"><span class="num cq-att-n">${i + 1}</span><span class="cq-att-m"><b class="mono">${esc(a.node)}</b><span class="num fail-dim">${S.hm(a.at)}</span><span class="fail-dim">${how}</span></span><code class="mono cq-err">${esc(a.error)}</code></li>`;
  }).join('');
  return `<div class="cq-sig cq-sig-fail${open ? ' open' : ''}" data-nodes="${nodes.join(' ')}"${hidden(nodes)}>
    <div class="cq-sig-row" role="button" tabindex="0" data-q-toggle="${c.id}" aria-expanded="${open}">
      ${S.I('chevron-right', 'cq-chev')}
      <span class="chip-fail">${S.I('circle-x')}Failed on ${nodes.length} Nodes</span>
      <span class="cq-sig-id">${S.art(w)}<span class="cq-wt"><b class="ell">${esc(w.title)}</b><span class="fail-dim ell">${esc(S.target(c.target).name)} · ${esc(prLabel(c.priority))}</span></span></span>
      <span class="cq-sig-what">${esc(s.verdict)}</span>
      <span class="num fail-dim cq-sig-t">${S.hm(last.at)} · ${S.ago(last.at)}</span>
    </div>
    <div class="cq-sig-act"><button class="btn fail sm" data-q-retry="${c.id}">${S.I('rotate-ccw')}Retry</button><button class="btn sm" data-toast="Searching for a new source for ${esc(w.title)} is not in this prototype">${S.I('search')}Search for a new source</button></div>
    ${open ? `<div class="cq-sig-more"><ol class="cq-atts">${att}</ol><p class="cq-verdict"><b>Verdict.</b> ${esc(s.verdict)} Nothing retries until the source changes or you press Retry.</p></div>` : ''}
  </div>`;
}

function noCapRow(g) {
  const cap = D.capabilities[g.cause], key = `nc-${g.cause}`, open = st.open.has(key);
  const nodes = [...g.offline, ...g.failedTrial].map((n) => n.id);
  const why = [
    ...g.offline.map((n) => `${n.name} has proven it but is offline since ${S.hm(n.offlineSince)}.`),
    ...g.failedTrial.map((n) => { const t = n.trials.find((x) => x.cap === g.cause); return `${n.name}'s trial failed ${new Date(t.at).toLocaleDateString('en-US', { month: 'short', day: 'numeric' })}: ${t.note}.`; }),
  ].join(' ');
  const rows = g.convs.map((c) => `<li class="cq-nc">${workCell(c)}<span class="t2">${esc(S.target(c.target).name)}</span><span class="t3">${esc(prLabel(c.priority))}</span><span class="mono cq-nc-src">${esc(c.source.spec)}</span><span class="num warn-dim">waiting since ${S.hm(c.state.since)}</span></li>`).join('');
  return `<div class="cq-sig cq-sig-warn${open ? ' open' : ''}" data-nodes="${nodes.join(' ')}"${hidden(nodes)}>
    <div class="cq-sig-row" role="button" tabindex="0" data-q-toggle="${key}" aria-expanded="${open}">
      ${S.I('chevron-right', 'cq-chev')}
      <span class="chip-warn">${S.I('triangle-alert')}No capable Node</span>
      <span class="cq-sig-id"><span class="cq-stack">${g.convs.slice(0, 3).map((c) => S.art(S.W(c.work))).join('')}</span><span class="cq-wt"><b>${plural(g.convs.length, 'Work')} need ${esc(cap.replace(/ \(.*\)$/, ''))}</b><span class="warn-dim ell">${g.convs.map((c) => S.W(c.work).title).join(', ')}</span></span></span>
      <span class="cq-sig-what">${esc(why)}</span>
      <span class="num warn-dim cq-sig-t">Notified once · ${S.hm(g.notified)}</span>
    </div>
    <div class="cq-sig-act">${g.offline.map((n) => `<button class="btn warn sm" data-toast="Wake-on-LAN sent to ${esc(n.name)}. The three Conversions start there when it comes online.">${S.I('power')}Wake ${esc(n.name)}</button>`).join('')}${g.failedTrial.map((n) => `<button class="btn sm" data-toast="Trial run for ${esc(cap)} queued on ${esc(n.name)}">${S.I('flask-conical')}Rerun trial on ${esc(n.name)}</button>`).join('')}</div>
    ${open ? `<div class="cq-sig-more"><ul class="cq-ncs">${rows}</ul><p class="cq-verdict warn-dim">Each one is placed as soon as a Node that has proven ${esc(cap)} is online. You were notified once for this cause and won't be again.</p></div>` : ''}
  </div>`;
}

function bandNeeds() {
  const body = failedNow().map(failRow).join('') + noCapGroups().map(noCapRow).join('');
  return band('needs', 'Needs you', `${failedNow().length} failed · ${noCapGroups().length} cause with no capable Node`, `<div class="cq-sigs">${body}</div>`);
}

function band(id, title, sub, inner) {
  const off = st.band !== 'all' && st.band !== id;
  return `<section class="cq-band" data-band="${id}" aria-labelledby="cqh-${id}"${off ? ' hidden' : ''}><div class="sec-h"><h2 id="cqh-${id}">${title}</h2><span class="t3">${sub}</span></div>${inner}</section>`;
}

function nowRow(c) {
  const s = c.state, w = S.W(c.work), paused = s.is === 'suspended', n = S.node(s.node);
  const pauseTxt = paused ? `Paused · ${n.mediaServer.name} is transcoding on ${n.name}` : '';
  return `<div class="jrow cq-now${paused ? ' cq-paused' : ''}" role="row" data-work="${w.id}" data-nodes="${s.node}"${hidden([s.node])}>
    ${workCell(c, paused ? { sub: `<span class="cq-psub-l ell" title="${esc(s.cause)}">${S.I('pause')}${esc(pauseTxt)}</span>` } : {})}
    <div class="c-tgt" role="cell">${esc(S.target(c.target).name)}</div>
    <div role="cell" class="cq-kc">${kindCell(c)}</div>
    <div role="cell"><div class="bar${paused ? ' cq-bar-p' : ''}" role="progressbar" aria-valuemin="0" aria-valuemax="100" aria-label="${esc(w.title)} progress" data-live-progress="${c.id}"><span class="fill"></span></div></div>
    <div class="c-pct num" role="cell" data-live-pct="${c.id}"></div>
    ${paused ? `<div class="c-rate t3" role="cell">Paused</div><div class="c-eta t3" role="cell">–</div>` : `<div class="c-rate num" role="cell" data-live-fps="${c.id}"></div><div class="c-eta num" role="cell" data-live-eta="${c.id}"></div>`}
    <div role="cell">${nodeCell(s.node)}</div>
    <div class="c-menu" role="cell"><button class="ib" data-q-menu="${c.id}" data-menu aria-haspopup="menu" aria-label="Actions for ${esc(w.title)}">${S.I('ellipsis')}</button></div>
  </div>`;
}

function bandNow() {
  const run = runningNow();
  const head = `<div class="jrow jhead cq-now" role="row"><div role="columnheader">Work</div><div role="columnheader">Target</div><div role="columnheader">Kind</div><div role="columnheader">Progress</div><div role="columnheader" class="num">%</div><div role="columnheader">Rate</div><div role="columnheader">ETA</div><div role="columnheader">Node</div><div role="columnheader"><span class="sr">Actions</span></div></div>`;
  const paused = run.filter((c) => c.state.is === 'suspended').length;
  const none = st.node && !run.some((c) => c.state.node === st.node) ? `<p class="cq-none t3">Nothing is converting on ${esc(st.node)}${S.node(st.node).activeHours ? `. Its active hours are ${S.node(st.node).activeHours.from}–${S.node(st.node).activeHours.to}` : S.node(st.node).status !== 'online' ? '. It is offline' : ''}.</p>` : '';
  return band('now', 'Converting now', `${run.length - paused} running · ${paused} paused, resumes on its own`, `<div class="ptable" role="table" aria-label="Converting now">${head}${run.map(nowRow).join('')}${none}</div>`);
}

function groupCount(g) { return g.works ? g.works.length : g.items.reduce((n, i) => n + i.episodes, 0); }

function groupLabel(g) {
  if (g.works) return plural(g.works.length, S.ct(S.W(g.works[0])).label, S.ct(S.W(g.works[0])).plural);
  const t = S.ct(S.W(g.items[0].work)), key = t.layout.collapse;
  const folds = new Set(g.items.map((i) => `${i.work}:${i[key]}`)).size;
  const lvl = t.levels.find((l) => l.key === key);
  return `${plural(groupCount(g), leaf(t).label.toLowerCase())} in ${plural(folds, lvl.label.toLowerCase())}`;
}

function groupKids(g) {
  if (g.works) {
    const order = Object.entries(g.nodes).flatMap(([n, k]) => Array(k).fill(n));
    return g.works.map((id, i) => {
      const w = S.W(id), nid = order[i], ah = S.node(nid).activeHours;
      return `<div class="jrow cq-next cq-kid" role="row" data-work="${id}" data-nodes="${nid}"${hidden([nid])}><div></div><div></div><div class="c-work" role="cell">${S.art(w)}<span class="cq-wt"><b class="ell">${esc(w.title)}</b></span></div><div class="c-tgt">${esc(S.target(g.target).name)}</div><div>${kindCell({ kind: 'derive', target: g.target })}</div><div>${nodeCell(nid)}</div><div class="cq-wait">${ah ? activeChip(nid) : '<span class="chip">' + S.I('list-ordered') + 'Queued</span>'}</div><div></div><div></div><div></div></div>`;
    }).join('');
  }
  return g.items.map((it) => {
    const w = S.W(it.work), t = S.ct(w), nid = Object.keys(g.nodes)[0];
    return `<div class="jrow cq-next cq-kid" role="row" data-work="${w.id}" data-nodes="${nid}"${hidden([nid])}><div></div><div></div><div class="c-work" role="cell">${S.art(w)}<span class="cq-wt"><b class="ell">${esc(w.title)}</b><span class="t3 ell">${esc(t.group(it))} · ${plural(it.episodes, leaf(t).label.toLowerCase())}</span></span></div><div class="c-tgt">${esc(S.target(g.target).name)}</div><div>${kindCell({ kind: 'adjust', target: g.target, steps: ['Add stereo track'] })}</div><div>${nodeCell(nid)}</div><div class="cq-wait"><span class="chip">${S.I('list-ordered')}Queued</span></div><div></div><div></div><div></div></div>`;
  }).join('');
}

function nextRow(e, i, p) {
  const r = p[e.id], fl = st.flash === e.id ? ' cq-flash' : '';
  const grip = `<div class="cq-grip" data-q-grip title="Drag to reorder" aria-hidden="true">${S.I('grip-vertical')}</div><div class="num t3 cq-pos" role="cell">${i + 1}</div>`;
  const acts = (label) => `<div class="cq-acts" role="cell">${e.prio === 'bump' && i === 0 ? '<span class="t3 cq-top">Next</span>' : `<button class="btn ghost sm cq-up" data-q-up="${e.id}" aria-label="Move ${esc(label)} up">${S.I('arrow-up-to-line')}Move up</button>`}<button class="ib" data-q-menu="${e.id}" data-menu aria-haspopup="menu" aria-label="Actions for ${esc(label)}">${S.I('ellipsis')}</button></div>`;
  if (e.kind === 'group') {
    const g = e.g, open = st.open.has(g.id), nodes = Object.keys(g.nodes);
    const first = g.works ? g.works : g.items.map((x) => x.work);
    return `<div class="jrow cq-next cq-grp${open ? ' open' : ''}${fl}" role="row" data-entry="${e.id}" draggable="false" data-nodes="${nodes.join(' ')}"${hidden(nodes)}>
      ${grip}
      <div class="c-work" role="cell"><button class="cq-gt" data-q-toggle="${g.id}" aria-expanded="${open}">${S.I('chevron-right', 'cq-chev')}<span class="cq-stack">${[...new Set(first)].slice(0, 3).map((id) => S.art(S.W(id))).join('')}</span><span class="cq-wt"><b class="ell">${esc(groupLabel(g))}</b><span class="t3 ell">${esc(g.label)}</span></span></button></div>
      <div class="c-tgt" role="cell">${esc(S.target(g.target).name)}</div>
      <div role="cell">${kindCell(g.works ? { kind: 'derive', target: g.target } : { kind: 'adjust', target: g.target, steps: ['Add stereo track'] })}</div>
      <div role="cell" class="cq-gnodes">${Object.entries(g.nodes).map(([n, k]) => `<span><span class="mono">${n}</span> <span class="num t3">${k}</span></span>`).join('')}</div>
      <div role="cell" class="cq-wait"><span class="chip">${S.I('list-ordered')}Queued</span><span class="t3 cq-wsub"><span class="num">${S.fmtHours(g.estHours)}</span> · <span class="num">${g.estGB} GB</span></span></div>
      <div class="num cq-when" role="cell">${fmtT(r.start)}</div>
      <div class="num cq-when" role="cell">${fmtT(r.finish)}</div>
      ${acts(g.label)}
    </div>${open ? `<div class="cq-kids"${hidden(nodes)}><p class="cq-reason t3">${esc(g.reason)}</p>${groupKids(g)}</div>` : ''}`;
  }
  const c = e.c, w = S.W(c.work);
  return `<div class="jrow cq-next${fl}" role="row" data-entry="${e.id}" data-work="${w.id}" data-nodes="${r.node || ''}"${hidden([r.node])}>
    ${grip}
    ${workCell(c)}
    <div class="c-tgt" role="cell">${esc(S.target(c.target).name)}</div>
    <div role="cell">${kindCell(c)}</div>
    <div role="cell">${nodeCell(r.node)}</div>
    <div role="cell" class="cq-wait">${waitChip(c, r)}</div>
    <div class="num cq-when" role="cell">${r.start ? fmtT(r.start) : '<span class="t3">when space frees</span>'}</div>
    <div class="num cq-when" role="cell">${r.finish ? fmtT(r.finish) : '<span class="t3">–</span>'}</div>
    ${acts(entryName(e))}
  </div>`;
}

function bandNext(p) {
  let i = 0, html = '';
  for (const pr of PR) {
    const es = st.entries.filter((e) => e.prio === pr);
    if (!es.length) continue;
    const n = es.reduce((k, e) => k + (e.kind === 'group' ? groupCount(e.g) : 1), 0);
    const vis = !st.node || es.some((e) => (e.kind === 'group' ? Object.keys(e.g.nodes) : [p[e.id].node]).includes(st.node));
    const hint = { bump: 'you moved these up', new: 'new Items', rederive: 'source upgraded, remade from the new file', backfill: 'lowest priority, runs after everything above' }[pr];
    html += `<div class="cq-sub" role="row" data-prio="${pr}"${vis ? '' : ' hidden'}><span role="cell"><b>${prLabel(pr)}</b><span class="num t3">${n}</span><span class="t3">${hint}</span></span></div>`;
    html += es.map((e) => nextRow(e, i++, p)).join('');
  }
  const head = `<div class="jrow jhead cq-next" role="row"><div></div><div role="columnheader">#</div><div role="columnheader">Work</div><div role="columnheader">Target</div><div role="columnheader">Kind</div><div role="columnheader">Node</div><div role="columnheader">Waiting for</div><div role="columnheader">Starts</div><div role="columnheader">Finishes</div><div role="columnheader"><span class="sr">Actions</span></div></div>`;
  return band('next', 'Up next', 'in the order the Controller will run them · drag or Move up to reorder', `<div class="ptable cq-ledger" role="table" aria-label="Up next">${head}${html}</div>`);
}

// ───────────────────────────── phone

function phRow({ id, work, art, title, sub, right, right2, cls = '', act }) {
  return `<li class="gl cq-pl ${cls}"${work ? ` data-work="${work}"` : ''}${act ? ` data-q-sheet="${act}" role="button" tabindex="0"` : ''}>${art}<span class="ltxt"><b class="ell">${title}</b><span class="t3 cq-pl-sub">${sub}</span></span><span class="cq-pl-r">${right}${right2 ? `<span class="t3 num">${right2}</span>` : ''}</span></li>`;
}

function phoneQueue(p) {
  const off = (b) => (st.band !== 'all' && st.band !== b ? ' hidden' : '');
  const needs = failedNow().map((c) => {
    const w = S.W(c.work), s = c.state, open = st.open.has(c.id), nodes = [...new Set(s.attempts.map((a) => a.node))];
    return `<li class="cq-pfail${open ? ' open' : ''}"><div class="cq-pf-h" role="button" tabindex="0" data-q-toggle="${c.id}" aria-expanded="${open}"><span class="chip-fail">${S.I('circle-x')}Failed on ${nodes.length} Nodes</span><span class="grow"></span>${S.I('chevron-down', 'cq-chev')}</div>
      <div class="cq-pf-id">${S.art(w, 'thumb-md')}<span class="ltxt"><b>${esc(w.title)}</b><span class="fail-dim">${esc(S.target(c.target).name)} · ${S.hm(s.attempts.at(-1).at)}</span></span></div>
      <p class="cq-pf-v">${esc(s.verdict)}</p>
      ${open ? `<ol class="cq-atts">${s.attempts.map((a, i) => `<li class="cq-att"><span class="num cq-att-n">${i + 1}</span><span class="cq-att-m"><b class="mono">${a.node}</b><span class="num fail-dim">${S.hm(a.at)}</span></span><code class="mono cq-err">${esc(a.error)}</code></li>`).join('')}</ol>` : ''}
      <div class="cq-pf-act"><button class="btn fail" data-q-retry="${c.id}">Retry</button><button class="btn" data-toast="Searching for a new source is not in this prototype">New source</button></div></li>`;
  }).join('') + noCapGroups().map((g) => {
    const key = `nc-${g.cause}`, open = st.open.has(key);
    return `<li class="cq-pwarn${open ? ' open' : ''}"><div class="cq-pf-h" role="button" tabindex="0" data-q-toggle="${key}" aria-expanded="${open}"><span class="chip-warn">${S.I('triangle-alert')}No capable Node</span><span class="grow"></span><span class="num warn-dim">notified ${S.hm(g.notified)}</span>${S.I('chevron-down', 'cq-chev')}</div>
      <p><b>${plural(g.convs.length, 'Work')} need ${esc(D.capabilities[g.cause].replace(/ \(.*\)$/, ''))}.</b> <span class="warn-dim">${g.offline.map((n) => `${n.name} could, but is offline since ${S.hm(n.offlineSince)}.`).join(' ')} ${g.failedTrial.map((n) => `${n.name}'s trial failed after a driver update.`).join(' ')}</span></p>
      ${open ? `<ul class="cq-pnc">${g.convs.map((c) => `<li>${S.art(S.W(c.work))}<span class="ltxt"><b class="ell">${esc(S.W(c.work).title)}</b><span class="warn-dim">${esc(S.target(c.target).name)} · since ${S.hm(c.state.since)}</span></span></li>`).join('')}</ul>` : ''}
      <div class="cq-pf-act">${g.offline.map((n) => `<button class="btn warn" data-toast="Wake-on-LAN sent to ${n.name}">Wake ${n.name}</button>`).join('')}${g.failedTrial.map((n) => `<button class="btn" data-toast="Trial run queued on ${n.name}">Rerun trial on ${n.name}</button>`).join('')}</div></li>`;
  }).join('');

  const now = runningNow().map((c) => {
    const w = S.W(c.work), s = c.state, paused = s.is === 'suspended', il = S.itemLabel(c);
    return `<li class="pj cq-pj" data-work="${w.id}"${hidden([s.node])}><div class="pj-top">${S.art(w)}<span class="pj-id"><b class="ell">${esc(w.title)}</b><span class="t3 ell">${il ? `${esc(il)} · ` : ''}${esc(S.target(c.target).name)} · ${c.kind === 'adjust' ? 'Adjust' : 'Derive'}</span></span><span class="pj-pct num" data-live-pct="${c.id}"></span></div>
      <div class="bar${paused ? ' cq-bar-p' : ''}" role="progressbar" aria-label="${esc(w.title)} progress" data-live-progress="${c.id}"><span class="fill"></span></div>
      <div class="pj-meta">${paused ? `${S.I('pause')}<span class="ell t2">Paused · ${S.node(s.node).mediaServer.name} is transcoding on ${s.node}</span>` : `<span class="mono t2">${s.node}</span><span class="s-sep">·</span><span class="num" data-live-fps="${c.id}"></span><span class="s-sep">·</span><span class="num" data-live-eta="${c.id}"></span> left`}</div></li>`;
  }).join('');

  let i = 0, next = '';
  for (const pr of PR) {
    const es = st.entries.filter((e) => e.prio === pr);
    if (!es.length) continue;
    next += `<li class="cq-psub">${prLabel(pr)}</li>`;
    next += es.map((e) => {
      const r = p[e.id], pos = ++i, fl = st.flash === e.id ? ' cq-flash' : '';
      if (e.kind === 'group') {
        const g = e.g;
        return phRow({ act: e.id, cls: fl, art: `<span class="cq-stack">${[...new Set(g.works || g.items.map((x) => x.work))].slice(0, 2).map((id) => S.art(S.W(id))).join('')}</span>`, title: esc(groupLabel(g)), sub: `${esc(S.target(g.target).name)} · Backfill · ${Object.entries(g.nodes).map(([n, k]) => `${n} ${k}`).join(', ')}`, right: `<span class="num">${fmtT(r.start)}</span>`, right2: `${S.fmtHours(g.estHours)}` });
      }
      const c = e.c, w = S.W(c.work), il = S.itemLabel(c), s = c.state;
      const why = { queued: 'Queued', 'source-conversion': 'Waiting for source', 'disk-space': `Disk space · ${s.freeGB} of ${s.needGB} GB` }[s.reason];
      return phRow({ act: e.id, cls: fl, work: w.id, art: S.art(w), title: `<span class="num t3 cq-ppos">${pos}</span>${esc(w.title)}`, sub: `${il ? `${esc(il.split(' · ')[0])} · ` : ''}${esc(S.target(c.target).name)} · ${c.kind === 'adjust' ? 'Adjust' : 'Derive'} · ${r.node || 'not placed'} · <span class="${s.reason === 'disk-space' ? 'warn-t' : 't2'}">${why}</span>`, right: r.start ? `<span class="num">${fmtT(r.start)}</span>` : '<span class="t3">when space frees</span>', right2: r.finish ? `→ ${fmtT(r.finish)}` : '' });
    }).join('');
  }
  return `<p class="ph-sentence cq-ph-pulse">${pulse(p)}</p>
    <div class="cq-ph-seg">${seg('band', [['all', 'All'], ['needs', 'Needs you'], ['now', 'Converting'], ['next', 'Up next']], st.band, 'Show')}</div>
    <section class="psec" data-band="needs"${off('needs')}><div class="psec-h"><h2>Needs you</h2></div><ul class="cq-pneeds">${needs}</ul></section>
    <section class="psec" data-band="now"${off('now')}><div class="psec-h"><h2>Converting now</h2></div><ul class="pjs">${now}</ul></section>
    <section class="psec" data-band="next"${off('next')}><div class="psec-h"><h2>Up next</h2><span class="t3">tap to reorder</span></div><ul class="glist cq-pnext">${next}</ul></section>`;
}

function phoneSheet(id) {
  const e = st.entries.find((x) => x.id === id), name = entryName(e);
  const sheet = root.querySelector('.sheet-wrap');
  const top = st.entries[0].id === id;
  const items = [
    ...(top ? [] : [['arrow-up-to-line', 'Move up', `data-q-up="${id}"`]]),
    ...(e.kind === 'group' ? [['list', 'Show what is in this group', `data-toast="The group's Works are on the wide layout in this prototype"`]] : [['pin', `Pin ${S.target(e.c.target).name} to a Node`, 'data-toast="Pinning a Target to a Node is not in this prototype"']]),
    ['file-video', 'Open the Work', 'data-toast="The Work page is not in this prototype"'],
    ['x', e.kind === 'group' ? 'Cancel this backfill' : 'Cancel this Conversion', 'data-toast="Cancelling is not in this prototype"', 'danger'],
  ];
  const r = plan()[id];
  sheet.querySelector('.sheet').innerHTML = `<span class="grab" aria-hidden="true"></span><p class="sheet-t"><b>${esc(name)}</b><span class="t3">${prLabel(e.prio)} · ${r.start ? `starts about ${fmtT(r.start)}` : 'starts when disk space frees'}</span></p><div class="sheet-grp" role="menu">${items.map(([ic, l, a, k]) => `<button class="mi cq-mi${k ? ' danger' : ''}" role="menuitem" ${a}>${S.I(ic)}<span>${esc(l)}</span></button>`).join('')}</div><button class="sheet-x" data-close>Cancel</button>`;
  sheet.hidden = false;
  window.lucide.createIcons();
}

// ───────────────────────────── queue render + wire

export const queue = (s) => {
  S = s; D = s.D;
  initQueue();
  return `<div class="cq-b${S.isPhone ? ' cq-ph' : ''}" id="cq-b">${body()}</div>`;
};

function body() {
  const p = plan();
  if (S.isPhone) return phoneQueue(p);
  return `<div class="cq-head"><p class="sentence cq-pulse">${pulse(p)}</p>${tools()}</div>${bandNeeds()}${bandNow()}${bandNext(p)}`;
}

function render() {
  const el = root.querySelector('#cq-b');
  el.innerHTML = body();
  window.lucide.createIcons();
  if (S.isPhone) S.wireAmbient('#cq-b [data-work]', 'dune-2');
  setTimeout(() => { st.flash = null; root.querySelectorAll('.cq-flash').forEach((x) => x.classList.remove('cq-flash')); }, 1600);
}

function rowMenu(btn, id) {
  const e = st.entries.find((x) => x.id === id);
  const c = e ? e.c : conv(id);
  if (e && e.kind === 'group') return S.openMenu(btn, { title: e.g.label, items: [['Move the whole group up', 'arrow-up-to-line', null, 'Use Move up on the row'], ['Cancel this backfill', 'x', 'danger']] });
  const tn = S.target(c.target).name;
  const live = c.state.is === 'running' || c.state.is === 'suspended';
  S.openMenu(btn, { title: S.W(c.work).title, items: [
    [`Pin ${tn} to a Node`, 'pin', null, `Pinning ${tn} to a Node is not in this prototype`],
    ['Open the Work', 'file-video'],
    ...(live ? [['Show the ffmpeg log', 'terminal']] : []),
    [live ? 'Stop and requeue' : 'Cancel this Conversion', 'x', 'danger'],
  ] });
}

function wireQueue() {
  root.addEventListener('click', (ev) => {
    const q = (s) => ev.target.closest(s);
    let el;
    if ((el = q('[data-q-up]'))) { ev.stopPropagation(); S.closeMenus(); return moveUp(el.dataset.qUp); }
    if ((el = q('[data-q-toggle]'))) { const k = el.dataset.qToggle; st.open.has(k) ? st.open.delete(k) : st.open.add(k); return render(); }
    if ((el = q('[data-q-band]'))) { st.band = el.dataset.qBand; return render(); }
    if ((el = q('[data-q-node]'))) { st.node = el.dataset.qNode; return render(); }
    if ((el = q('[data-q-retry]'))) { ev.stopPropagation(); return S.toast(`Retry queued for ${S.W(conv(el.dataset.qRetry).work).title}. It starts on forge first, then atlas.`); }
    if ((el = q('[data-q-menu]'))) { ev.stopPropagation(); return rowMenu(el, el.dataset.qMenu); }
    if ((el = q('.cq-mi[data-toast]'))) { ev.stopPropagation(); S.closeMenus(); return S.toast(el.dataset.toast); }
    if ((el = q('[data-q-sheet]'))) { ev.stopPropagation(); return phoneSheet(el.dataset.qSheet); }
  }, true);
  root.addEventListener('keydown', (ev) => {
    const el = ev.target.closest('[data-q-toggle], [data-q-sheet]');
    if (el && (ev.key === 'Enter' || ev.key === ' ')) { ev.preventDefault(); el.click(); }
  });
  if (S.isPhone) return S.wireAmbient('#cq-b [data-work]', 'dune-2');
  S.setAmbient('dune-2');
  let t;
  root.addEventListener('mouseover', (ev) => {
    const r = ev.target.closest('#cq-b [data-work]');
    clearTimeout(t);
    t = setTimeout(() => S.setAmbient(r ? r.dataset.work : 'dune-2'), r ? 90 : 450);
  });
  root.addEventListener('pointerdown', (ev) => {
    const g = ev.target.closest('[data-q-grip]');
    if (g) g.closest('[data-entry]').draggable = true;
  });
  root.addEventListener('dragstart', (ev) => {
    const r = ev.target.closest('[data-entry]');
    if (!r) return;
    st.drag = r.dataset.entry;
    ev.dataTransfer.effectAllowed = 'move';
    ev.dataTransfer.setData('text/plain', st.drag);
    r.classList.add('cq-dragging');
  });
  root.addEventListener('dragover', (ev) => {
    const r = ev.target.closest('.cq-ledger [data-entry]');
    if (!st.drag || !r) return;
    ev.preventDefault();
    root.querySelectorAll('.cq-drop').forEach((x) => x.classList.remove('cq-drop'));
    r.classList.add('cq-drop');
  });
  root.addEventListener('drop', (ev) => {
    const r = ev.target.closest('.cq-ledger [data-entry]');
    if (!st.drag || !r) return;
    ev.preventDefault();
    const to = st.entries.findIndex((e) => e.id === r.dataset.entry), from = st.entries.findIndex((e) => e.id === st.drag);
    const id = st.drag; st.drag = null;
    if (to < from) moveUp(id, to); else render();
  });
  root.addEventListener('dragend', () => { st.drag = null; root.querySelectorAll('.cq-drop, .cq-dragging').forEach((x) => x.classList.remove('cq-drop', 'cq-dragging')); root.querySelectorAll('[data-entry][draggable="true"]').forEach((x) => { x.draggable = false; }); });
}

// ───────────────────────────── backfill dialog

const mins = (rt) => { const m = rt.match(/(\d+)h\s*(\d+)m/); return m ? +m[1] * 60 + +m[2] : 120; };

function initBackfill() {
  const A = D.backfillAsk;
  const all = D.cinemaWorks;
  const totalMin = all.reduce((n, id) => n + mins(S.W(id).runtime), 0);
  const gb48 = D.estimate(all.length).gb;
  const meta = {};
  all.forEach((id, i) => {
    const w = S.W(id), r = A.addedRecently.indexOf(id);
    const daysAgo = r >= 0 ? r + 1 : 40 + ((i * 37) % 600);
    meta[id] = {
      gb: (mins(w.runtime) / totalMin) * gb48,
      mobile: A.playedOnMobile.includes(id),
      req: A.requestedBy[id] ? D.members[A.requestedBy[id]] : '',
      daysAgo,
      dv5: A.noCapable.some((x) => x.work === id),
    };
  });
  bf = {
    step: 1, dir: 1, open: true,
    sel: new Set(),
    q: '', sort: { key: 'added', dir: 1 }, filt: '',
    dest: A.existingMatch ? 'match' : 'create', name: A.destination.name,
    other: D.profiles.find((p) => p.contentType === A.contentType && p.id !== A.profile && p.id !== A.existingMatch).id,
    makeDefault: false, meta,
  };
}

const ask = () => D.backfillAsk;
const prof = (id) => D.profiles.find((p) => p.id === id);
const ctOf = () => D.contentTypes[ask().contentType];
const tn = (id) => S.target(id).name;
const destName = () => (bf.dest === 'match' ? prof(ask().existingMatch).name : bf.dest === 'create' ? bf.name : prof(bf.other).name);
const selGB = () => Math.round([...bf.sel].reduce((n, id) => n + bf.meta[id].gb, 0));
const addedTxt = (d) => (d <= 1 ? 'yesterday' : d < 14 ? `${d} days ago` : new Date(D.now.getTime() - d * 864e5).toLocaleDateString('en-US', { month: 'short', year: 'numeric' }));

function costLine(n) {
  if (!n) return 'Nothing to convert';
  const e = D.estimate(n);
  return `${plural(n, 'Conversion')} · about ${S.fmtHours(e.hours - D.today.hoursQueued)} of work, done in ≈ ${S.fmtHours(e.hours)} · ${selGB() || e.gb} GB`;
}

function totalLine() {
  const n = bf.sel.size, rest = D.cinemaWorks.length - n;
  return `<p class="cq-total" aria-live="polite"><b class="num">${n}</b> of ${D.cinemaWorks.length} convert<span class="s-sep">·</span><span class="num">${n ? S.fmtHours(D.estimate(n).hours) : '0 min'}</span> until done<span class="s-sep">·</span><span class="num">${selGB()} GB</span><span class="s-sep">·</span><span class="num">${rest}</span> move to <b>${esc(destName())}</b></p>`;
}

function rail() {
  const steps = [['Cost', 'What all of them would take'], ['Works', `${bf.sel.size} of ${D.cinemaWorks.length} chosen`], ['The rest', `${D.cinemaWorks.length - bf.sel.size} move to ${destName()}`]];
  return `<ol class="cq-rail">${steps.map(([t, s], i) => `<li><button class="cq-rs${bf.step === i + 1 ? ' on' : ''}${bf.step > i + 1 ? ' done' : ''}" data-b-step="${i + 1}" ${bf.step === i + 1 ? 'aria-current="step"' : ''}><span class="cq-rn num">${bf.step > i + 1 ? S.I('check') : i + 1}</span><span class="cq-rt"><b>${t}</b><span class="t3 ell">${esc(s)}</span></span></button></li>`).join('')}</ol>`;
}

function step1() {
  const A = ask(), all = D.cinemaWorks.length, e = D.estimate(all), p = prof(A.profile), ct = ctOf();
  const work = e.hours - D.today.hoursQueued;
  return `<div class="cq-s1">
    <p class="cq-lede">${esc(p.name)} now wants <b>${esc(tn(A.addedTarget))}</b>, derived from ${esc(tn(S.target(A.addedTarget).from))}. Its ${all} ${esc(ct.plural)} have no ${esc(tn(A.addedTarget))} file yet.</p>
    <dl class="cq-cost">
      <div><dt>Time</dt><dd class="num">+${S.fmtHours(work)}</dd><span class="t3">added to the queue across Nodes. Done about <span class="num t2">${fmtT(D.now.getTime() + e.hours * 3600000)}</span>, after the <span class="num">${S.fmtHours(D.today.hoursQueued)}</span> already queued.</span></div>
      <div><dt>Disk</dt><dd class="num">${e.gb} GB</dd><span class="t3">≈ ${A.perWork.gb} GB a Work on <span class="mono">${esc(A.disk.root)}</span>, ${A.disk.freeTB} TB free</span></div>
      <div><dt>Nodes</dt><dd class="cq-nodes">${A.nodes.map((n) => `<span class="cq-ns${n.share ? '' : ' off'}"><span class="mono">${n.node}</span><span class="cq-nbar"><span style="width:${n.share * 100}%"></span></span><span class="num t2">${Math.round(n.share * 100)}%</span><span class="t3">${esc(n.note)}</span></span>`).join('')}</dd></div>
    </dl>
    <p class="cq-note">${S.I('layers')}<span>They run at Backfill priority, after every new Item and upgrade, so nothing already waiting gets pushed back.</span></p>
    <p class="cq-note cq-note-warn">${S.I('triangle-alert')}<span>${A.noCapable.map((x) => esc(S.W(x.work).title)).join(', ')} have Dolby Vision Profile 5 sources. No online Node can convert them, so if you include them they wait as no capable Node.</span></p>
    <div class="cq-paths" role="group" aria-label="Choose a path">
      <button class="cq-path" data-b-path="all"><b>Convert all ${all}</b><span class="t3">Every ${esc(ct.label)} in ${esc(p.name)} gets ${esc(tn(A.addedTarget))}. Nothing moves.</span>${S.I('arrow-right', 'cq-pgo')}</button>
      <button class="cq-path pri" data-b-path="pick"><b>Choose Works</b><span>Pick by playback on phones, Member requests or date added. The rest move to a Profile without ${esc(tn(A.addedTarget))}.</span>${S.I('arrow-right', 'cq-pgo')}</button>
      <button class="cq-path" data-b-path="none"><b>Convert none</b><span class="t3">All ${all} move to a Profile without ${esc(tn(A.addedTarget))}. New ${esc(ct.plural)} still get it.</span>${S.I('arrow-right', 'cq-pgo')}</button>
    </div>
  </div>`;
}

const FILTERS = [['mobile', 'Played on phones', (m) => m.mobile], ['req', 'Requested by Members', (m) => !!m.req], ['recent', 'Added in the last 2 weeks', (m) => m.daysAgo < 14], ['dv5', 'DV Profile 5', (m) => m.dv5]];

function shownWorks() {
  const f = FILTERS.find((x) => x[0] === bf.filt);
  const q = bf.q.trim().toLowerCase();
  const rows = D.cinemaWorks.filter((id) => (!f || f[2](bf.meta[id])) && (!q || S.W(id).title.toLowerCase().includes(q)));
  const k = bf.sort.key, d = bf.sort.dir;
  const val = (id) => { const w = S.W(id), m = bf.meta[id]; return { title: w.title.replace(/^The /, ''), year: w.year, gb: m.gb, mobile: m.mobile ? 0 : 1, req: m.req || '~', added: m.daysAgo, sel: bf.sel.has(id) ? 0 : 1 }[k]; };
  return rows.sort((a, b) => { const x = val(a), y = val(b); return (x < y ? -1 : x > y ? 1 : 0) * d; });
}

function step2() {
  const rows = shownWorks();
  const allOn = rows.length && rows.every((id) => bf.sel.has(id));
  const th = (k, l, cls = '') => `<button class="cq-th ${cls}" data-b-sort="${k}" aria-sort="${bf.sort.key === k ? (bf.sort.dir === 1 ? 'ascending' : 'descending') : 'none'}">${l}${bf.sort.key === k ? S.I(bf.sort.dir === 1 ? 'chevron-up' : 'chevron-down') : ''}</button>`;
  const ct = ctOf();
  const list = rows.map((id) => {
    const w = S.W(id), m = bf.meta[id], on = bf.sel.has(id);
    return `<label class="cq-wr${on ? ' on' : ''}" data-work="${id}"><input type="checkbox" class="cq-ck" data-b-pick="${id}" ${on ? 'checked' : ''} aria-label="${esc(w.title)}">${S.art(w)}<span class="cq-wr-t"><b class="ell">${esc(w.title)}</b>${m.dv5 ? '<span class="chip cq-dv">DV P5</span>' : ''}</span><span class="num t2">${w.year}</span><span class="num t2">${m.gb.toFixed(1)} GB</span><span>${m.mobile ? `<span class="cq-yes">${S.I('smartphone')}Yes</span>` : '<span class="t3">–</span>'}</span><span class="${m.req ? '' : 't3'} ell">${m.req ? esc(m.req) : '–'}</span><span class="t2">${addedTxt(m.daysAgo)}</span></label>`;
  }).join('');
  const f = FILTERS.find((x) => x[0] === bf.filt);
  return `<div class="cq-s2">
    <div class="cq-s2-tools">
      <label class="cq-search">${S.I('search')}<input type="search" placeholder="Search ${esc(ct.plural)}" value="${esc(bf.q)}" data-b-q aria-label="Search ${esc(ct.plural)}"></label>
      <div class="cq-filts" role="group" aria-label="Show only">${FILTERS.map(([k, l, fn]) => `<button class="cq-filt" data-b-filt="${k}" aria-pressed="${bf.filt === k}">${esc(l)}<span class="num">${D.cinemaWorks.filter((id) => fn(bf.meta[id])).length}</span></button>`).join('')}</div>
    </div>
    <div class="cq-s2-sel">
      ${f || bf.q ? `<span class="t2">${rows.length} shown</span><button class="btn sm" data-b-selshown="1">${S.I('check-check')}Select these ${rows.length}</button><button class="btn ghost sm" data-b-selshown="0">Deselect these</button>` : `<button class="btn sm" data-b-selshown="1">Select all ${rows.length}</button><button class="btn ghost sm" data-b-selshown="0">Clear</button>`}
      <span class="grow"></span><span class="t3">Played on phones comes from the Playback report, last 30 days</span>
    </div>
    <div class="cq-wt-wrap" role="table" aria-label="${esc(ct.plural)} in ${esc(prof(ask().profile).name)}">
      <div class="cq-wr cq-wh"><input type="checkbox" class="cq-ck" data-b-all ${allOn ? 'checked' : ''} aria-label="Select all shown"><span></span>${th('title', 'Title')}${th('year', 'Year')}${th('gb', 'Est. size')}${th('mobile', 'Played on phones')}${th('req', 'Requested by')}${th('added', 'Added')}</div>
      ${list || `<p class="cq-empty t2">No ${esc(ct.plural)} match.</p>`}
    </div>
  </div>`;
}

function step3() {
  const A = ask(), p = prof(A.profile), ct = ctOf(), rest = D.cinemaWorks.length - bf.sel.size, n = bf.sel.size;
  const oldT = p.targets.map(tn).join(' and ');
  const others = D.profiles.filter((x) => x.contentType === A.contentType && x.id !== p.id && x.id !== A.existingMatch);
  const lose = (pid) => p.targets.filter((t) => !prof(pid).targets.includes(t)).map(tn);
  const opt = (k, title, sub, extra = '') => `<label class="cq-opt${bf.dest === k ? ' on' : ''}"><input type="radio" name="cq-dest" value="${k}" data-b-dest ${bf.dest === k ? 'checked' : ''}><span class="cq-opt-b"><b>${title}</b><span class="t3">${sub}</span>${extra}</span></label>`;
  const dv = A.noCapable.filter((x) => bf.sel.has(x.work));
  const dest = rest ? `<fieldset class="cq-fs"><legend>Where the other ${rest} go</legend>
      ${A.existingMatch ? opt('match', `Move to ${esc(prof(A.existingMatch).name)}`, `Already wants ${esc(prof(A.existingMatch).targets.map(tn).join(' and '))}, the same Targets ${esc(p.name)} had. ${prof(A.existingMatch).works} Works use it today.`) : ''}
      ${opt('create', A.existingMatch ? 'Create a new Profile instead' : 'Create a matching Profile', `Wants ${esc(oldT)}, like ${esc(p.name)} before this change.`, bf.dest === 'create' ? `<span class="cq-name"><input type="text" value="${esc(bf.name)}" data-b-name aria-label="New Profile name" spellcheck="false">${S.I('pencil')}</span>` : '')}
      ${opt('other', 'Move to another Profile', 'Pick one you already have.', bf.dest === 'other' ? `<span class="cq-other"><select data-b-other aria-label="Profile">${others.map((o) => `<option value="${o.id}" ${bf.other === o.id ? 'selected' : ''}>${esc(o.name)} · ${esc(o.targets.map(tn).join(', '))}</option>`).join('')}</select>${lose(bf.other).length ? `<span class="warn-t cq-lose">${S.I('triangle-alert')}They would lose ${esc(lose(bf.other).join(' and '))}.</span>` : ''}</span>` : '')}
    </fieldset>` : `<p class="cq-note">${S.I('check')}<span>All ${D.cinemaWorks.length} convert, so nothing moves.</span></p>`;
  const defName = rest ? destName() : null;
  const def = `<fieldset class="cq-fs"><legend>Default for new ${esc(ct.plural)}</legend>
      <label class="cq-opt${!bf.makeDefault ? ' on' : ''}"><input type="radio" name="cq-def" value="0" data-b-def ${!bf.makeDefault ? 'checked' : ''}><span class="cq-opt-b"><b>Keep ${esc(p.name)} as the default</b><span class="t3">New ${esc(ct.plural)} get ${esc(tn(A.addedTarget))} too.</span></span></label>
      ${defName ? `<label class="cq-opt${bf.makeDefault ? ' on' : ''}"><input type="radio" name="cq-def" value="1" data-b-def ${bf.makeDefault ? 'checked' : ''}><span class="cq-opt-b"><b>Make ${esc(defName)} the default</b><span class="t3">New ${esc(ct.plural)} skip ${esc(tn(A.addedTarget))} unless you change their Profile.</span></span></label>` : ''}
    </fieldset>`;
  return `<div class="cq-s3">${dest}${def}
    ${dv.length ? `<p class="cq-note cq-note-warn">${S.I('triangle-alert')}<span>${dv.map((x) => esc(S.W(x.work).title)).join(', ')} ${dv.length === 1 ? 'has a' : 'have'} Dolby Vision Profile 5 source${dv.length === 1 ? '' : 's'}. ${dv.length === 1 ? 'It waits' : 'They wait'} as no capable Node until studio is back online or forge passes a new trial.</span></p>` : ''}
    <p class="cq-confirm">${confirmSentence()}</p>
  </div>`;
}

function confirmSentence() {
  const A = ask(), p = prof(A.profile), ct = ctOf(), n = bf.sel.size, rest = D.cinemaWorks.length - n;
  const conv = n ? `Convert <b>${plural(n, 'Work')}</b> to ${esc(tn(A.addedTarget))} at Backfill priority, about <span class="num">${S.fmtHours(D.estimate(n).hours)}</span> until done and <span class="num">${selGB()} GB</span>.` : `Convert nothing.`;
  const mv = rest ? ` Move <b>${rest}</b> to <b>${esc(destName())}</b>${bf.dest === 'create' ? ', a new Profile with ' + esc(p.targets.map(tn).join(' and ')) : ''}.` : '';
  const def = ` ${bf.makeDefault && rest ? `<b>${esc(destName())}</b> becomes` : `<b>${esc(p.name)}</b> stays`} the default for ${esc(ct.plural)}.`;
  return `Save ${esc(p.name)} with ${esc(tn(A.addedTarget))}. ${conv}${mv}${def}`;
}

function stepBody() { return bf.step === 1 ? step1() : bf.step === 2 ? step2() : step3(); }
const stepTitle = () => ['What converting all of them costs', `Choose which ${ctOf().plural} to convert`, 'Where the rest go'][bf.step - 1];

function foot() {
  const n = bf.sel.size;
  const save = `<button class="btn pri" data-b-save>${n ? `Save and convert ${n}` : `Save and move ${D.cinemaWorks.length}`}</button>`;
  const next = bf.step === 1 ? '' : bf.step === 2 ? '<button class="btn pri" data-b-next>Next</button>' : save;
  return `${bf.step > 1 ? totalLine() : '<span class="t3 cq-esc">Cancel returns to the editor without saving.</span>'}<span class="grow"></span><button class="btn ghost" data-b-cancel>Cancel</button>${bf.step > 1 ? '<button class="btn" data-b-back>Back</button>' : ''}${next}`;
}

function dialogWide() {
  const A = ask(), p = prof(A.profile);
  return `<div class="cq-modal${bf.step === 2 ? ' tall' : ''}" role="dialog" aria-modal="true" aria-labelledby="cq-dt">
    <aside class="cq-side"><p class="cq-kicker t3">Saving ${esc(p.name)}</p><h2 id="cq-dt" class="cq-dt">Backfill ${esc(tn(A.addedTarget))}</h2>${rail()}</aside>
    <div class="cq-main">
      <header class="cq-mh"><h3>${stepTitle()}</h3><span class="t3 num">Step ${bf.step} of 3</span></header>
      <div class="cq-mb cq-slide-${bf.dir > 0 ? 'f' : 'b'}">${stepBody()}</div>
      <footer class="cq-mf">${foot()}</footer>
    </div>
  </div>`;
}

function dialogPhone() {
  const back = bf.step === 1 ? '<button class="cq-sh-b" data-b-cancel>Cancel</button>' : `<button class="cq-sh-b" data-b-back>${S.I('chevron-left')}Back</button>`;
  const nxt = bf.step === 2 ? '<button class="cq-sh-b r" data-b-next>Next</button>' : bf.step === 3 ? '<span></span>' : '<span></span>';
  return `<div class="cq-sheet" role="dialog" aria-modal="true" aria-labelledby="cq-dt">
    <span class="grab" aria-hidden="true"></span>
    <header class="cq-sh-h">${back}<span class="cq-sh-t"><b id="cq-dt">${['Cost', 'Works', 'The rest'][bf.step - 1]}</b><span class="t3 num">${bf.step} of 3 · ${esc(tn(ask().addedTarget))}</span></span>${nxt}</header>
    <div class="cq-sh-b0 cq-slide-${bf.dir > 0 ? 'f' : 'b'}"><h3 class="cq-sh-h3">${stepTitle()}</h3>${bf.step === 2 ? step2Phone() : stepBody()}</div>
    ${bf.step > 1 ? `<footer class="cq-sh-f">${totalLine()}${bf.step === 3 ? `<button class="btn pri" data-b-save>${bf.sel.size ? `Save and convert ${bf.sel.size}` : `Save and move ${D.cinemaWorks.length}`}</button>` : '<button class="btn pri" data-b-next>Next</button>'}</footer>` : ''}
  </div>`;
}

function step2Phone() {
  const rows = shownWorks(), ct = ctOf();
  return `<div class="cq-s2p">
    <label class="cq-search">${S.I('search')}<input type="search" placeholder="Search ${esc(ct.plural)}" value="${esc(bf.q)}" data-b-q aria-label="Search ${esc(ct.plural)}"></label>
    <div class="cq-filts cq-filts-p" role="group" aria-label="Show only">${FILTERS.map(([k, l, fn]) => `<button class="cq-filt" data-b-filt="${k}" aria-pressed="${bf.filt === k}">${esc(l)}<span class="num">${D.cinemaWorks.filter((id) => fn(bf.meta[id])).length}</span></button>`).join('')}</div>
    <div class="cq-s2-sel"><button class="btn sm" data-b-selshown="1">Select ${bf.filt || bf.q ? `these ${rows.length}` : `all ${rows.length}`}</button><button class="btn ghost sm" data-b-selshown="0">${bf.filt || bf.q ? 'Deselect' : 'Clear'}</button><span class="grow"></span>
      <label class="cq-sort"><span class="sr">Sort by</span><select data-b-sortsel>${[['added', 'Newest'], ['title', 'Title'], ['mobile', 'Played on phones'], ['gb', 'Size'], ['sel', 'Selected first']].map(([k, l]) => `<option value="${k}" ${bf.sort.key === k ? 'selected' : ''}>${l}</option>`).join('')}</select>${S.I('arrow-down-up')}</label></div>
    <ul class="glist cq-wl">${rows.map((id) => {
      const w = S.W(id), m = bf.meta[id], on = bf.sel.has(id);
      return `<li><label class="gl cq-wlr${on ? ' on' : ''}" data-work="${id}"><input type="checkbox" class="cq-ck" data-b-pick="${id}" ${on ? 'checked' : ''} aria-label="${esc(w.title)}">${S.art(w, 'thumb-md')}<span class="ltxt"><b class="ell">${esc(w.title)}</b><span class="t3">${w.year} · <span class="num">${m.gb.toFixed(1)} GB</span> · ${addedTxt(m.daysAgo)}</span>${m.mobile || m.req || m.dv5 ? `<span class="cq-tags">${m.mobile ? `<span class="cq-yes">${S.I('smartphone')}Phones</span>` : ''}${m.req ? `<span class="t2">${S.I('user')}${esc(m.req)}</span>` : ''}${m.dv5 ? '<span class="chip cq-dv">DV P5</span>' : ''}</span>` : ''}</span></label></li>`;
    }).join('')}</ul>
  </div>`;
}

export const dialog = (s) => {
  S = s; D = s.D;
  initBackfill();
  return `<div class="cq-dlg${S.isPhone ? ' ph' : ''}" id="cq-dlg"><div class="cq-scrim" data-b-cancel></div><div id="cq-dlg-in">${S.isPhone ? dialogPhone() : dialogWide()}</div></div>`;
};

function renderDlg(keepFocus) {
  const box = root.querySelector('#cq-dlg-in');
  const sc = box.querySelector('.cq-wt-wrap, .cq-sh-b0');
  const top = sc ? sc.scrollTop : 0;
  const active = document.activeElement && document.activeElement.matches('[data-b-q], [data-b-name]') ? document.activeElement.dataset.bQ !== undefined ? '[data-b-q]' : '[data-b-name]' : null;
  const pos = active ? document.activeElement.selectionStart : 0;
  box.innerHTML = S.isPhone ? dialogPhone() : dialogWide();
  window.lucide.createIcons();
  if (keepFocus) {
    const sc2 = box.querySelector('.cq-wt-wrap, .cq-sh-b0');
    if (sc2) sc2.scrollTop = top;
    box.querySelectorAll('.cq-slide-f, .cq-slide-b').forEach((x) => x.classList.remove('cq-slide-f', 'cq-slide-b'));
    if (active) { const i = box.querySelector(active); i.focus(); i.setSelectionRange(pos, pos); }
  }
}

function go(step) { bf.dir = step > bf.step ? 1 : -1; bf.step = step; renderDlg(false); }

function wireBackfill() {
  const dlg = root.querySelector('#cq-dlg');
  S.setAmbient('dune-2');
  root.addEventListener('click', (ev) => {
    const q = (s) => ev.target.closest(s);
    let el;
    if (q('[data-open-backfill]')) { ev.stopPropagation(); initBackfill(); dlg.hidden = false; renderDlg(false); return; }
    if (!q('#cq-dlg')) return;
    ev.stopPropagation();
    if ((el = q('[data-b-cancel]'))) { dlg.hidden = true; return; }
    if ((el = q('[data-b-step]'))) return go(+el.dataset.bStep);
    if ((el = q('[data-b-path]'))) {
      const k = el.dataset.bPath;
      if (k === 'all') { bf.sel = new Set(D.cinemaWorks); return go(3); }
      if (k === 'none') { bf.sel = new Set(); return go(3); }
      if (!bf.sel.size) bf.sel = new Set(D.backfillAsk.playedOnMobile.filter((id) => D.cinemaWorks.includes(id)));
      return go(2);
    }
    if ((el = q('[data-b-next]'))) return go(bf.step + 1);
    if ((el = q('[data-b-back]'))) return go(bf.step - 1);
    if ((el = q('[data-b-sort]'))) { const k = el.dataset.bSort; bf.sort = { key: k, dir: bf.sort.key === k ? -bf.sort.dir : 1 }; return renderDlg(true); }
    if ((el = q('[data-b-filt]'))) { bf.filt = bf.filt === el.dataset.bFilt ? '' : el.dataset.bFilt; return renderDlg(true); }
    if ((el = q('[data-b-selshown]'))) { const on = el.dataset.bSelshown === '1'; shownWorks().forEach((id) => (on ? bf.sel.add(id) : bf.sel.delete(id))); return renderDlg(true); }
    if ((el = q('[data-b-save]'))) {
      const n = bf.sel.size, rest = D.cinemaWorks.length - n;
      dlg.hidden = true;
      return S.toast(`Cinema saved. ${n ? `${plural(n, 'Conversion')} queued at Backfill priority` : 'Nothing queued'}${rest ? `, ${rest} Works moved to ${destName()}` : ''}.`);
    }
  }, true);
  root.addEventListener('change', (ev) => {
    const t = ev.target;
    if (t.matches('[data-b-pick]')) { t.checked ? bf.sel.add(t.dataset.bPick) : bf.sel.delete(t.dataset.bPick); return renderDlg(true); }
    if (t.matches('[data-b-all]')) { shownWorks().forEach((id) => (t.checked ? bf.sel.add(id) : bf.sel.delete(id))); return renderDlg(true); }
    if (t.matches('[data-b-dest]')) { bf.dest = t.value; return renderDlg(true); }
    if (t.matches('[data-b-def]')) { bf.makeDefault = t.value === '1'; return renderDlg(true); }
    if (t.matches('[data-b-other]')) { bf.other = t.value; return renderDlg(true); }
    if (t.matches('[data-b-sortsel]')) { bf.sort = { key: t.value, dir: 1 }; return renderDlg(true); }
  });
  root.addEventListener('input', (ev) => {
    const t = ev.target;
    if (t.matches('[data-b-q]')) { bf.q = t.value; return renderDlg(true); }
    if (t.matches('[data-b-name]')) {
      bf.name = t.value;
      root.querySelectorAll('.cq-total, .cq-confirm, .cq-rail').forEach((x) => { x.outerHTML = x.classList.contains('cq-total') ? totalLine() : x.classList.contains('cq-rail') ? rail() : `<p class="cq-confirm">${confirmSentence()}</p>`; });
      window.lucide.createIcons();
    }
  });
  root.addEventListener('keydown', (ev) => { if (ev.key === 'Escape' && !dlg.hidden) dlg.hidden = true; });
  let t;
  root.addEventListener('mouseover', (ev) => {
    const r = ev.target.closest('#cq-dlg [data-work]');
    if (!r) return;
    clearTimeout(t);
    t = setTimeout(() => S.setAmbient(r.dataset.work), 120);
  });
}

export function wire(s, screen) {
  S = s; D = s.D; root = document.getElementById('app');
  if (screen === 'queue') wireQueue(); else wireBackfill();
}
