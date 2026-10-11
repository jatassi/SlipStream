// PROTOTYPE variant A · Lanes: what is each Node doing, and what's stuck?

const CAP_SHORT = { 'hevc10-hdr10-sdr': 'HDR10 → SDR', 'dv5-sdr': 'DV P5 → SDR', 'dv7-81': 'DV P7 → 8.1', 'h264-720p': 'H.264 720p', 'remux-audio': 'Remux + audio' };
const PRI = { bump: 0, new: 1, rederive: 2, backfill: 3 };

const mins = (a, b) => Math.round((b - a) / 60000);
const atClock = (now, hhmm) => {
  const [h, m] = hhmm.split(':').map(Number);
  const d = new Date(now); d.setHours(h, m, 0, 0);
  if (d <= now) d.setDate(d.getDate() + 1);
  return d;
};
const plural = (n, one, many = `${one}s`) => `${n} ${n === 1 ? one : many}`;

function model(S) {
  const { D } = S;
  const convs = D.conversions;
  const failed = convs.filter((c) => c.state.is === 'failed');
  const noCap = convs.filter((c) => c.state.is === 'waiting' && c.state.reason === 'no-capable-node');
  const causes = [...new Set(noCap.map((c) => c.state.cause))].map((cause) => {
    const list = noCap.filter((c) => c.state.cause === cause);
    return {
      cause, list,
      notified: list.map((c) => c.state.since).sort()[0],
      offline: D.nodes.filter((n) => n.status !== 'online' && n.proven.includes(cause)),
      trialFailed: D.nodes.flatMap((n) => n.trials.filter((t) => t.cap === cause && t.result === 'failed').map((t) => ({ n, t }))),
    };
  });
  const bfCount = (b) => (b.works ? b.works.length : b.items.reduce((m, i) => m + i.episodes, 0));
  const lanes = D.nodes.map((n) => {
    const mine = convs.filter((c) => c.state.node === n.id);
    return {
      n,
      running: mine.filter((c) => c.state.is === 'running'),
      suspended: mine.filter((c) => c.state.is === 'suspended'),
      queued: mine.filter((c) => c.state.is === 'waiting').sort((a, b) => PRI[a.priority] - PRI[b.priority]),
      backfill: D.backfill.filter((b) => b.nodes[n.id]).map((b) => ({ b, share: b.nodes[n.id], total: bfCount(b) })),
      couldTake: n.status !== 'online' ? noCap.filter((c) => n.proven.includes(c.needs)) : [],
    };
  });
  const counts = {
    running: convs.filter((c) => c.state.is === 'running').length,
    suspended: convs.filter((c) => c.state.is === 'suspended').length,
    queued: convs.filter((c) => c.state.is === 'waiting' && c.state.reason !== 'no-capable-node').length,
    backfill: D.backfill.reduce((m, b) => m + bfCount(b), 0),
    noCap: noCap.length,
    failed: failed.length,
  };
  return { failed, causes, lanes, counts };
}

function backfillSlice(b, nodeId) {
  if (b.works) {
    let start = 0;
    for (const [id, k] of Object.entries(b.nodes)) { if (id === nodeId) return { works: b.works.slice(start, start + k) }; start += k; }
  }
  return { items: b.items };
}

// ───────────────────────────── queue

export function queue(S) {
  const { D, esc, I, isPhone } = S;
  const M = model(S);
  const c = M.counts;
  const sentence = [
    `<span class="s-live">${S.dot('draft')}${c.running} converting</span>`,
    `<span>${c.suspended} paused for ${esc(D.nodes.find((n) => n.mediaServer)?.mediaServer.name || 'a Media Server')}</span>`,
    `<span>${c.queued} queued <span class="t3">+ ${c.backfill} backfill</span></span>`,
    `<span class="s-warn">${c.noCap} waiting for a capable Node</span>`,
    `<span class="s-fail">${c.failed} failed</span>`,
  ].join('<span class="s-sep">·</span>');

  const needs = [...M.failed.map((f) => failCard(S, f)), ...M.causes.map((k) => causeCard(S, k))].join('');

  if (isPhone) {
    return `<div class="cqa cqa-ph">
      <p class="ph-sentence cqa-sent">${sentence}</p>
      <section class="psec cqa-needs-ph" aria-label="Needs you"><div class="psec-h"><h2>Needs you</h2><span class="t3">${M.failed.length + M.causes.length}</span></div>${needs}</section>
      <section class="psec" aria-label="Nodes">
        <div class="psec-h"><h2>Nodes</h2></div>
        <div class="seg-c cqa-pseg" role="radiogroup" aria-label="Show Node">${M.lanes.map((l, i) => `<button role="radio" class="seg-o" data-lane="${l.n.id}" aria-checked="${i === 0}">${S.dot(laneDot(l))}<span>${esc(l.n.name)}</span></button>`).join('')}</div>
        <div class="cqa-plane" data-plane>${phoneLane(S, M.lanes[0])}</div>
      </section>
    </div>`;
  }

  return `<div class="cqa">
    <p class="cqa-sent">${sentence}<span class="grow"></span><span class="t3 cqa-today num">Today ${D.today.finished} finished · ${fmtQueued(S)} queued across Nodes</span></p>
    <section aria-labelledby="cqa-needs-h">
      <div class="sec-h"><h2 id="cqa-needs-h">Needs you</h2><span class="t3">Nothing here retries or moves on until you act or the source changes.</span></div>
      <div class="cqa-needs">${needs}</div>
    </section>
    <section aria-labelledby="cqa-lanes-h">
      <div class="sec-h"><h2 id="cqa-lanes-h">Converting Nodes</h2><span class="t3">Each Node runs its lane top to bottom. The Controller places a Conversion on the capable Node with the earliest finish.</span></div>
      <div class="cqa-lanes">${M.lanes.map((l) => wideLane(S, l)).join('')}</div>
    </section>
  </div>`;
}

const fmtQueued = (S) => S.fmtHours(S.D.today.hoursQueued);
const laneDot = (l) => (l.n.status !== 'online' ? 'fail' : l.running.length ? 'draft' : 'idle');

function failCard(S, c) {
  const { esc, I, isPhone, W, target, node, hm, ago } = S;
  const w = W(c.work), s = c.state, last = s.attempts[s.attempts.length - 1];
  const nodesTried = [...new Set(s.attempts.map((a) => a.node))];
  const sub = [S.itemLabel(c), target(c.target).name].filter(Boolean).join(' · ');
  const attempts = s.attempts.map((a, i) => `<li class="cqa-att"><span class="cqa-att-n num">${i + 1}</span><div><p class="cqa-att-h"><b>${esc(node(a.node).name)}</b><span class="fail-dim num">${hm(a.at)}</span>${i === 1 && a.node === s.attempts[0].node ? '<span class="fail-dim">retry on the same Node</span>' : i > 0 ? '<span class="fail-dim">moved to another Node</span>' : ''}</p><p class="mono cqa-err">${esc(a.error)}</p></div></li>`).join('');
  const log = `<pre class="mono cqa-log" data-log hidden>${s.attempts.map((a, i) => `# attempt ${i + 1} · ${node(a.node).name} · ${a.at.replace('T', ' ')}\n$ jellyfin-ffmpeg -hwaccel ${a.node === 'forge' ? 'cuda' : 'none'} -i "${w.title} (${w.year}).mkv" -map 0 -c:v hevc_${a.node === 'forge' ? 'nvenc' : 'x265'} -vf tonemap…\n${a.error}\n`).join('\n')}</pre>`;
  const actions = `<button class="btn fail" data-toast="Retrying ${esc(w.title)} on ${esc(node(s.attempts[0].node).name)}. Nothing else retries on its own.">${I('rotate-ccw')}Retry</button>
    <button class="btn" data-toast="Searching for a new 2160p HDR source for ${esc(w.title)} is not in this prototype">${I('search')}Search for a new source</button>
    <button class="btn" data-show-log aria-expanded="false">${I('scroll-text')}Show log</button>`;
  if (isPhone) {
    return `<article class="pfail cqa-pfail">
      <div class="pfail-h"><span class="chip-fail">${I('circle-x')}Failed on ${nodesTried.length} Nodes</span><span class="fail-dim num">${ago(last.at)}</span></div>
      <div class="pfail-id">${S.art(w, 'thumb-md')}<div class="ltxt"><b>${esc(w.title)}</b><span class="fail-dim">${esc(sub)}</span></div></div>
      <p class="cqa-verdict">${esc(s.verdict)}</p>
      <ol class="cqa-atts">${attempts}</ol>
      ${log}
      <div class="cqa-pacts">${actions}</div>
    </article>`;
  }
  return `<article class="fail cqa-fail">
    <div class="fail-main">
      <div class="fail-h"><span class="chip-fail">${I('circle-x')}Failed on ${nodesTried.length} Nodes · stopped</span><span class="fail-dim num">last try ${hm(last.at)} · ${ago(last.at)}</span></div>
      <div class="cqa-fail-b">
        <div class="cqa-fail-id">${S.art(w, 'thumb-md')}<div class="ltxt"><b>${esc(w.title)}</b><span class="fail-dim">${esc(sub)} · ${esc(c.source.spec)}</span><p class="cqa-verdict">${esc(s.verdict)}</p></div></div>
        <ol class="cqa-atts">${attempts}</ol>
      </div>
      ${log}
      <div class="cqa-fail-act">${actions}<span class="grow"></span><span class="fail-dim cqa-small">Retries once on the same Node, then once on another. It stops after two Nodes.</span></div>
    </div>
  </article>`;
}

function causeCard(S, k) {
  const { D, esc, I, isPhone, W, hm, node } = S;
  const lines = [
    ...k.offline.map((n) => `<b>${esc(n.name)}</b> has proven it but is offline since <span class="num">${hm(n.offlineSince)}</span>`),
    ...k.trialFailed.map(({ n, t }) => `<b>${esc(n.name)}</b>'s trial failed after driver <span class="num">${esc((n.ffmpeg.match(/driver ([\d.]+)/) || [])[1] || '')}</span>`),
  ].join(' · ');
  const posters = k.list.map((c) => { const w = W(c.work); return `<li class="cqa-cposter" data-work="${w.id}" tabindex="0">${S.art(w, 'poster-sm')}<span class="cqa-cp-t">${esc(w.title)}</span><span class="warn-dim cqa-cp-s">${esc(D.priorities.find((p) => p.id === c.priority).label)} · ${esc(S.target(c.target).name)}</span></li>`; }).join('');
  const actions = [
    ...k.offline.map((n) => `<button class="btn warn" data-toast="Sent a wake request to ${esc(n.name)}. Its 3 Conversions start when it comes online.">${I('power')}Wake ${esc(n.name)}</button>`),
    ...k.trialFailed.map(({ n }) => `<button class="btn" data-toast="Rerunning the ${esc(CAP_SHORT[k.cause])} trial on ${esc(n.name)}…">${I('flask-conical')}Rerun trial on ${esc(n.name)}</button>`),
  ].join('');
  return `<article class="cqa-cause${isPhone ? ' cqa-cause-ph' : ''}">
    <div class="cqa-cause-h"><span class="chip-warn">${I('triangle-alert')}No capable Node</span><b>${esc(D.capabilities[k.cause])}</b></div>
    <p class="cqa-cause-why">${plural(k.list.length, 'Conversion')} wait. ${lines}.</p>
    <ul class="cqa-cposters">${posters}</ul>
    <div class="cqa-cause-f">${actions}<span class="grow"></span><span class="warn-dim cqa-small">${I('bell')}You were notified once at <span class="num">${hm(k.notified)}</span>, for this cause</span></div>
  </article>`;
}

function slotPips(l) {
  const used = l.running.length + l.suspended.length;
  const pips = Array.from({ length: l.n.cap }, (_, i) => `<span class="cqa-pip ${i < l.running.length ? 'on' : i < used ? 'paused' : ''}"></span>`).join('');
  return `<span class="cqa-slots" title="Runs as many at once as its trials sustain, capped at ${l.n.cap}"><span class="cqa-pips">${pips}</span><span class="num">${used} of ${l.n.cap}</span><span class="t3">slots</span></span>`;
}

function capsHtml(S, n) {
  const { D, esc, I } = S;
  return `<div class="cqa-caps">${Object.keys(D.capabilities).map((k) => {
    const has = n.proven.includes(k), failed = n.trials.find((t) => t.cap === k && t.result === 'failed');
    if (!has && !failed) return '';
    return `<span class="cqa-cap${failed ? ' x' : ''}" title="${esc(D.capabilities[k])}${failed ? ` · trial failed: ${esc(failed.note)}` : ''}">${failed ? I('x') : I('check')}${esc(CAP_SHORT[k])}</span>`;
  }).join('')}</div>`;
}

function hoursHtml(S, n) {
  const { esc, I } = S;
  const hours = n.activeHours ? `${I('clock')}Active <span class="num">${n.activeHours.from}–${n.activeHours.to}</span>` : `${I('clock')}Any time`;
  const ms = n.mediaServer ? `${I('tv')}Pauses for ${esc(n.mediaServer.name)}${n.mediaServer.transcoding ? ' <span class="cqa-now">transcoding now</span>' : ''}` : `${I('tv')}No Media Server`;
  return `<p class="cqa-lh3 t3"><span>${hours}</span><span>${ms}</span></p>`;
}

function laneHeader(S, l) {
  const { esc, I } = S;
  const n = l.n, off = n.status !== 'online';
  return `<header class="cqa-lh">
    <div class="cqa-lh1">${S.dot(laneDot(l))}<h3>${esc(n.name)}</h3><span class="t3">${esc(n.kind)}</span><span class="grow"></span>${off ? '<span class="chip cqa-off">Offline</span>' : slotPips(l)}<button class="ib sm" data-node-menu="${n.id}" aria-label="Actions for ${esc(n.name)}">${I('ellipsis')}</button></div>
    <p class="cqa-hw t2">${esc(n.hw)}</p>
    <p class="cqa-speed mono t3">${esc(n.speed)}</p>
    ${capsHtml(S, n)}
    ${hoursHtml(S, n)}
  </header>`;
}

function kindLine(S, c) {
  const { esc, I, target } = S;
  if (c.kind === 'adjust') return `<span class="cqa-kind" title="Runs the downloaded Target's steps on a copy">${I('sliders-horizontal')}<span>Adjust · ${esc((c.steps || []).join(', '))} · the torrent keeps seeding the original</span></span>`;
  const from = target(target(c.target).from);
  return `<span class="cqa-kind" title="Makes a derived Target's file">${I('replace')}<span>Derive from ${esc(from ? from.name : c.source.name)}</span></span>`;
}

function priChip(S, c) {
  const p = S.D.priorities.find((x) => x.id === c.priority);
  return c.priority === 'bump' ? `<span class="chip cqa-bump">${S.I('arrow-up')}${p.label}</span>` : `<span class="chip">${p.label}</span>`;
}

function runCard(S, c) {
  const { esc, I, W, target, hm } = S;
  const w = W(c.work), s = c.state, paused = s.is === 'suspended';
  const sub = [S.itemLabel(c), target(c.target).name].filter(Boolean).join(' · ');
  return `<article class="cqa-run${paused ? ' paused' : ''}" data-work="${w.id}" tabindex="0">
    <div class="cqa-run-top">${S.art(w, 'thumb-md')}<div class="ltxt"><b class="ell">${esc(w.title)}</b><span class="t2 ell">${esc(sub)} <span class="mono t3">v${c.settingsVersion}</span></span>${kindLine(S, c)}</div>${menuBtn(S, c)}</div>
    ${paused ? `<p class="cqa-paused">${I('pause')}<span><b>Paused</b> while ${esc(s.cause.replace(/^(\w+) on \w+ is/, '$1 is'))}. Resumes on its own; not a failure.</span></p>` : ''}
    <div class="bar${paused ? ' cqa-bar-p' : ''}" data-live-progress="${c.id}" role="progressbar" aria-label="${esc(w.title)} progress" aria-valuemin="0" aria-valuemax="100"><span class="fill"></span></div>
    <div class="cqa-run-m num"><span class="cqa-pct" data-live-pct="${c.id}"></span>${paused ? `<span class="t3">since ${hm(s.since)}</span>` : `<span class="t2" data-live-fps="${c.id}"></span><span class="t2">ETA <span data-live-eta="${c.id}"></span></span>`}<span class="grow"></span><span class="t3">→ ${esc(c.estSize)}</span></div>
  </article>`;
}

function waitReason(S, c, l) {
  const { esc, I, D, W, target } = S;
  const s = c.state;
  const R = {
    queued: () => [c.priority === 'bump' ? 'arrow-up' : 'list-ordered', c.priority === 'bump' ? 'Moved up · takes the next free slot' : 'Queued'],
    'source-conversion': () => { const o = D.conversions.find((x) => x.id === s.waitsOn); return ['link-2', `Waiting for the source's Conversion · ${target(o.target).name} ${o.kind === 'adjust' ? 'adjust' : 'derive'} above`]; },
    'disk-space': () => ['hard-drive', `Disk space · needs <span class="num">${s.needGB} GB</span>, scratch has <span class="num">${s.freeGB} GB</span> free · requeues when space frees`],
    'active-hours': () => ['moon', `Waiting for active hours · <span class="num">${l.n.activeHours.from}</span>`],
  };
  const [ic, txt] = R[s.reason]();
  return `<span class="cqa-why${s.reason === 'disk-space' ? ' disk' : ''}">${I(ic)}<span>${txt}</span></span>`;
}

function whenText(S, c, l) {
  const s = c.state;
  if (s.startsAt) return `<span class="num">${S.hm(s.startsAt)} → ${S.hm(s.finishesAt)}</span>`;
  if (l.n.activeHours) return `<span class="num">≥ ${l.n.activeHours.from}</span>`;
  return '<span class="t3">—</span>';
}

function queueRow(S, c, i, l) {
  const { esc, W, target } = S;
  const w = W(c.work);
  const sub = [S.itemLabel(c), target(c.target).name].filter(Boolean).join(' · ');
  return `<li class="cqa-q" data-work="${w.id}" tabindex="0">
    <span class="cqa-q-n num t3">${i + 1}</span>${S.art(w, 'thumb')}
    <div class="cqa-q-b">
      <div class="cqa-q-t"><b class="ell">${esc(w.title)}</b>${priChip(S, c)}<span class="grow"></span>${menuBtn(S, c)}</div>
      <div class="cqa-q-l2"><span class="t3 ell">${esc(sub)} · ${c.kind === 'adjust' ? 'Adjust' : 'Derive'}</span><span class="cqa-when">${whenText(S, c, l)}</span></div>
      ${waitReason(S, c, l)}
    </div>
  </li>`;
}

function bfCard(S, x, l) {
  const { esc, I, D, W, target } = S;
  const { b, share, total } = x;
  const slice = backfillSlice(b, l.n.id);
  const hours = (b.estHours * share) / total, gb = Math.round((b.estGB * share) / total);
  const firstWork = W(slice.works ? slice.works[0] : slice.items[0].work);
  const body = slice.works
    ? `<ul class="cqa-bf-works">${slice.works.map((id) => { const w = W(id); return `<li data-work="${id}" title="${esc(w.title)}">${S.art(w, 'thumb-md')}<span class="ell">${esc(w.title)}</span></li>`; }).join('')}</ul>`
    : `<ul class="cqa-bf-fold">${foldItems(S, slice.items).map((g) => `<li data-work="${g.w.id}">${S.art(g.w, 'thumb')}<div class="ltxt"><b class="ell">${esc(g.w.title)}</b><span class="t3">${g.rows.map((r) => `${esc(r.label)} · ${r.count}`).join('<span class="s-sep">·</span>')}</span></div></li>`).join('')}</ul>`;
  const unit = slice.works ? 'Works' : `${S.ct(firstWork).levels.at(-1).label.toLowerCase()}s`;
  return `<li class="cqa-bf" data-bf="${b.id}">
    <button class="cqa-bf-h" data-bf-toggle aria-expanded="false">
      <span class="cqa-bf-stack">${(slice.works || slice.items.map((i) => i.work)).filter((v, i, a) => a.indexOf(v) === i).slice(0, 3).map((id) => S.art(W(id), 'thumb')).join('')}</span>
      <span class="ltxt"><b>${esc(b.label)}</b><span class="t3"><span class="num">${share}</span> of <span class="num">${total}</span> ${unit} on this Node · ${esc(target(b.target).name)}</span></span>
      <span class="cqa-bf-r num"><span>≈ ${S.fmtHours(hours)}</span><span class="t3">${gb} GB</span></span>
      ${I('chevron-down', 'cqa-chev')}
    </button>
    <div class="cqa-bf-b" hidden><p class="t3 cqa-bf-why">${esc(b.reason)}</p>${body}</div>
  </li>`;
}

function foldItems(S, items) {
  const by = new Map();
  for (const it of items) {
    const w = S.W(it.work), t = S.ct(w);
    if (!by.has(w.id)) by.set(w.id, { w, rows: [] });
    const unit = t.levels.at(-1).label.toLowerCase();
    by.get(w.id).rows.push({ label: t.layout.collapse ? t.group(it) : t.item(it), count: plural(it.episodes, unit) });
  }
  return [...by.values()];
}

function laneNotice(S, l) {
  const { D, esc, I, hm, ago, fmtHours } = S;
  const n = l.n;
  if (n.status !== 'online') return `<div class="cqa-notice off">${I('power-off')}<div><b>Offline since <span class="num">${hm(n.offlineSince)}</span></b><span class="t3">${ago(n.offlineSince)} · nothing is placed here until it returns</span></div></div>`;
  if (n.activeHours) {
    const opens = atClock(D.now, n.activeHours.from);
    const nowHm = D.now.toTimeString().slice(0, 5);
    const inside = n.activeHours.from < n.activeHours.to ? nowHm >= n.activeHours.from && nowHm < n.activeHours.to : nowHm >= n.activeHours.from || nowHm < n.activeHours.to;
    if (!inside) return `<div class="cqa-notice hours">${I('moon')}<div><b>Starts nothing until <span class="num">${n.activeHours.from}</span></b><span class="t3">Opens in ${fmtHours(mins(D.now, opens) / 60)} · active <span class="num">${n.activeHours.from}–${n.activeHours.to}</span>; outside them it finishes running work only</span></div></div>`;
  }
  return '';
}

function wideLane(S, l) {
  const { esc, I, W, D } = S;
  const off = l.n.status !== 'online';
  const sect = (title, extra, inner) => (inner ? `<div class="cqa-ls"><p class="cqa-ls-h">${title}${extra ? `<span class="t3">${extra}</span>` : ''}</p>${inner}</div>` : '');
  const running = [...l.running, ...l.suspended].map((c) => runCard(S, c)).join('');
  const queued = l.queued.length ? `<ol class="cqa-ql">${l.queued.map((c, i) => queueRow(S, c, i, l)).join('')}</ol>` : '';
  const bf = l.backfill.length ? `<ul class="cqa-bfl">${l.backfill.map((x) => bfCard(S, x, l)).join('')}</ul>` : '';
  const could = off ? couldTake(S, l) : '';
  const idle = !off && !running ? `<p class="cqa-empty t3">${I('circle-dashed')}Nothing running${l.n.activeHours ? '' : ' · a slot is free'}</p>` : '';
  return `<section class="cqa-lane${off ? ' off' : ''}" aria-label="${esc(l.n.name)}">
    ${laneHeader(S, l)}
    ${laneNotice(S, l)}
    ${sect('Running', running ? `${l.running.length} running${l.suspended.length ? ` · ${l.suspended.length} paused` : ''}` : '', running)}${idle}
    ${sect('Up next', queued ? 'in run order · start → finish' : '', queued)}
    ${sect('Backfill', bf ? 'lowest priority · after everything above' : '', bf)}
    ${could}
  </section>`;
}

function couldTake(S, l) {
  const { esc, I, W } = S;
  const n = l.n;
  const list = l.couldTake.map((c) => { const w = W(c.work); return `<li class="cqa-q cqa-could" data-work="${w.id}" tabindex="0">${S.art(w, 'thumb')}<div class="ltxt"><b class="ell">${esc(w.title)}</b><span class="t3 ell">${esc(S.target(c.target).name)} · only ${esc(n.name)} has proven ${esc(CAP_SHORT[c.needs])}</span></div></li>`; }).join('');
  return `<div class="cqa-ls"><p class="cqa-ls-h">Could take if it were online</p>
    <ol class="cqa-ql">${list}</ol>
    <p class="t3 cqa-could-more">${I('zap')}<span>Plus ${n.cap} slots at <span class="num">${esc(n.speed.split(' on ')[0])}</span>, which would take Up next work from forge.</span></p>
    <button class="btn warn sm cqa-wake" data-toast="Sent a wake request to ${esc(n.name)}.">${I('power')}Wake ${esc(n.name)}</button>
  </div>`;
}

function menuBtn(S, c) {
  const w = S.W(c.work);
  return `<button class="ib sm cqa-mb" data-menu="${c.id}" aria-label="Actions for ${S.esc(w.title)}" aria-haspopup="menu">${S.I('ellipsis')}</button>`;
}

function phoneLane(S, l) {
  const { esc, I, W, target } = S;
  const n = l.n, off = n.status !== 'online';
  const head = `<div class="pstore cqa-plh">
    <div class="cqa-lh1">${S.dot(laneDot(l))}<b>${esc(n.name)}</b><span class="t3">${esc(n.kind)}</span><span class="grow"></span>${off ? '<span class="chip cqa-off">Offline</span>' : slotPips(l)}</div>
    <p class="t2">${esc(n.hw)}</p><p class="mono t3 cqa-speed">${esc(n.speed)}</p>
    ${capsHtml(S, n)}
    ${hoursHtml(S, n)}
  </div>`;
  const notice = laneNotice(S, l);
  const grp = (title, inner) => (inner ? `<p class="cqa-pg-h">${title}</p><ul class="glist">${inner}</ul>` : '');
  const running = [...l.running, ...l.suspended].map((c) => {
    const w = W(c.work), paused = c.state.is === 'suspended';
    return `<li class="pj cqa-pj${paused ? ' paused' : ''}" data-work="${w.id}">
      <div class="pj-top">${S.art(w, 'thumb-md')}<div class="pj-id"><b class="ell">${esc(w.title)}</b><span class="t3 ell">${esc([S.itemLabel(c), target(c.target).name].filter(Boolean).join(' · '))}</span></div><span class="num pj-pct" data-live-pct="${c.id}"></span>${menuBtn(S, c)}</div>
      <div class="bar${paused ? ' cqa-bar-p' : ''}" data-live-progress="${c.id}" role="progressbar" aria-label="${esc(w.title)} progress"><span class="fill"></span></div>
      <div class="pj-meta">${paused ? `<span class="cqa-paused-t">${I('pause')}Paused for ${esc(n.mediaServer.name)} · resumes on its own</span>` : `${kindLine(S, c)}<span class="grow"></span><span class="num t2" data-live-eta="${c.id}"></span>`}</div>
    </li>`;
  }).join('');
  const queued = l.queued.map((c, i) => {
    const w = W(c.work);
    return `<li class="gl cqa-pq" data-work="${w.id}">${S.art(w, 'thumb')}<div class="ltxt"><span class="cqa-q-t"><b class="ell">${esc(w.title)}</b>${c.priority === 'bump' ? priChip(S, c) : ''}</span><span class="t3 ell">${esc([S.itemLabel(c), target(c.target).name].filter(Boolean).join(' · '))}</span>${waitReason(S, c, l)}</div><span class="gl-r t2">${whenText(S, c, l)}</span></li>`;
  }).join('');
  const bf = l.backfill.map((x) => bfCard(S, x, l)).join('');
  const could = off ? l.couldTake.map((c) => { const w = W(c.work); return `<li class="gl" data-work="${w.id}">${S.art(w, 'thumb')}<div class="ltxt"><b class="ell">${esc(w.title)}</b><span class="t3 ell">only ${esc(n.name)} has proven ${esc(CAP_SHORT[c.needs])}</span></div></li>`; }).join('') : '';
  return `${head}${notice ? `<div class="cqa-pn">${notice}</div>` : ''}
    ${grp('Running', running)}
    ${!off && !running ? `<p class="cqa-empty t3">${I('circle-dashed')}Nothing running</p>` : ''}
    ${grp('Up next', queued)}
    ${bf ? `<p class="cqa-pg-h">Backfill</p><ul class="glist cqa-pbf">${bf}</ul>` : ''}
    ${could ? `${grp('Could take if online', could)}<div class="cqa-pwake"><button class="btn warn" data-toast="Sent a wake request to ${esc(n.name)}.">${I('power')}Wake ${esc(n.name)}</button></div>` : ''}`;
}

// ───────────────────────────── backfill dialog

const bf = { sel: null, q: '', dest: null, def: 'keep', open: true };

function bfInit(S) {
  if (bf.sel) return;
  const A = S.D.backfillAsk;
  bf.sel = new Set(A.playedOnMobile.filter((id) => S.D.cinemaWorks.includes(id)));
  bf.dest = A.existingMatch ? { kind: 'reuse', id: A.existingMatch, name: A.destination.name, other: null } : { kind: 'create', name: A.destination.name, other: null };
}

const FILTERS = (A, works) => [
  { id: 'mobile', label: 'Played on phones', icon: 'smartphone', ids: A.playedOnMobile },
  { id: 'req', label: 'Requested', icon: 'inbox', ids: Object.keys(A.requestedBy) },
  { id: 'recent', label: 'Added recently', icon: 'sparkles', ids: A.addedRecently },
  { id: 'all', label: 'All', ids: works },
  { id: 'none', label: 'None', ids: [] },
];

export function dialog(S) {
  const { D, esc, I, isPhone } = S;
  bfInit(S);
  const A = D.backfillAsk, p = D.profiles.find((x) => x.id === A.profile), tg = D.targets[A.addedTarget];
  const works = D.cinemaWorks;
  const chips = FILTERS(A, works).map((f) => `<button class="cqa-fchip" data-filter="${f.id}" aria-pressed="false">${f.icon ? I(f.icon) : ''}${f.label}${f.id !== 'none' ? `<span class="num t3">${f.ids.filter((id) => works.includes(id)).length}</span>` : ''}</button>`).join('');
  const tiles = works.map((id) => tile(S, id)).join('');
  const pick = `<section class="cqa-pick" aria-label="Works to backfill">
    <div class="cqa-pick-bar">
      <label class="cqa-search">${I('search')}<input type="search" placeholder="Search ${esc(D.contentTypes[A.contentType].plural)} in ${esc(p.name)}" data-bf-q aria-label="Search Works"></label>
      <span class="t3 cqa-selc" data-bf-selc></span>
    </div>
    <div class="cqa-fchips" role="group" aria-label="Select matching">${chips}</div>
    <ul class="cqa-grid" data-bf-grid>${tiles}</ul>
  </section>`;
  const head = `<header class="cqa-dh">
    <div class="ltxt"><p class="t3 cqa-dh-k">${esc(p.name)} gains ${esc(tg.name)} · derived from ${esc(D.targets[tg.from].name)}</p>
    <h2 id="cqa-dlg-h">Which Works should get ${esc(tg.name)} now?</h2>
    <p class="t2">Selected Works get Conversions at Backfill priority, after everything already queued. The rest move to a Profile without ${esc(tg.name)}, so every Work in ${esc(p.name)} has all its Targets.</p></div>
    <button class="ib" data-bf-cancel aria-label="Cancel and return to the editor">${I('x')}</button>
  </header>`;
  if (isPhone) {
    return `<div class="cqa-dwrap ph" data-bf-dlg role="dialog" aria-modal="true" aria-labelledby="cqa-dlg-h">
      <div class="sheet-scrim" data-bf-cancel></div>
      <div class="cqa-psheet"><span class="grab" aria-hidden="true"></span>
        <div class="cqa-psheet-s">${head}${pick}<div data-bf-sum></div></div>
        <footer class="cqa-pfoot" data-bf-foot></footer>
      </div>
    </div>`;
  }
  return `<div class="cqa-dwrap" data-bf-dlg role="dialog" aria-modal="true" aria-labelledby="cqa-dlg-h">
    <div class="cqa-scrim" data-bf-cancel></div>
    <div class="cqa-dlg">${head}
      <div class="cqa-dbody">${pick}<aside class="cqa-sum"><div class="cqa-sum-s" data-bf-sum></div><footer class="cqa-dfoot" data-bf-foot></footer></aside></div>
    </div>
  </div>`;
}

function tile(S, id) {
  const { D, esc, I } = S;
  const A = D.backfillAsk, w = S.W(id);
  const marks = [
    A.playedOnMobile.includes(id) ? `<span class="cqa-tm" title="Played on a phone or tablet in the last 30 days">${I('smartphone')}</span>` : '',
    A.requestedBy[id] ? `<span class="cqa-tm cqa-tm-av" title="Requested by ${esc(D.members[A.requestedBy[id]])}">${esc(D.members[A.requestedBy[id]].split(' ').map((s) => s[0]).join(''))}</span>` : '',
    A.addedRecently.includes(id) ? `<span class="cqa-tm" title="Added recently">${I('sparkles')}</span>` : '',
  ].join('');
  const dv = A.noCapable.find((x) => x.work === id);
  return `<li class="cqa-tile" data-title="${esc(w.title.toLowerCase())}"><button role="checkbox" aria-checked="${bf.sel.has(id)}" data-pick="${id}" data-work="${id}" aria-label="${esc(w.title)} (${w.year})">
    <span class="cqa-tposter" style="aspect-ratio:${S.ct(w).artwork.ratio}">${S.art(w, 'cqa-tart')}<span class="cqa-check">${I('check')}</span>${dv ? `<span class="cqa-tdv" title="${esc(D.capabilities[dv.cause])}: no online Node can convert it">${I('triangle-alert')}DV P5</span>` : ''}<span class="cqa-tmarks">${marks}</span></span>
    <span class="cqa-tt ell">${esc(w.title)}</span><span class="cqa-ty t3 num">${w.year}</span>
  </button></li>`;
}

function destProfiles(S) {
  const A = S.D.backfillAsk;
  return S.D.profiles.filter((p) => p.contentType === A.contentType && p.id !== A.profile && p.id !== A.existingMatch);
}

function destName(S) {
  const d = bf.dest;
  if (d.kind === 'reuse') return S.D.profiles.find((p) => p.id === d.id).name;
  if (d.kind === 'other') return S.D.profiles.find((p) => p.id === d.other).name;
  return d.name.trim() || 'Untitled Profile';
}

function summary(S) {
  const { D, esc, I, fmtHours, isPhone } = S;
  const A = D.backfillAsk, p = D.profiles.find((x) => x.id === A.profile), tg = D.targets[A.addedTarget], ctd = D.contentTypes[A.contentType];
  const n = bf.sel.size, total = D.cinemaWorks.length, rest = total - n;
  const est = D.estimate(n);
  const finish = new Date(D.now.getTime() + est.hours * 3600000);
  const finishTxt = `${finish.toLocaleDateString('en-US', { weekday: 'short' })} ${finish.toTimeString().slice(0, 5)}`;
  const nodeRows = A.nodes.map((x) => {
    const k = est.byNode[x.node] || 0, nd = S.node(x.node);
    return { id: x.node, k, note: x.note, off: nd.status !== 'online' };
  });
  const bar = n ? nodeRows.filter((r) => r.k).map((r) => `<span class="cqa-split-s ${r.id === 'forge' ? 'a' : 'b'}" style="flex:${r.k}" title="${r.id}: ${r.k}"></span>`).join('') : '<span class="cqa-split-s none"></span>';
  const freeGB = A.disk.freeTB * 1000;
  const dvSel = A.noCapable.filter((x) => bf.sel.has(x.work));
  const cause = dvSel[0] && dvSel[0].cause;
  const offProven = cause ? D.nodes.filter((nd) => nd.status !== 'online' && nd.proven.includes(cause)) : [];
  const trialFail = cause ? D.nodes.filter((nd) => nd.trials.some((t) => t.cap === cause && t.result === 'failed')) : [];

  const stats = `<div class="cqa-sb">
    <p class="cqa-big"><span class="num">${n}</span> <span class="cqa-big-u">of ${total} ${esc(ctd.plural)} get ${esc(tg.name)}</span></p>
    <div class="cqa-kv"><span class="t3">Time</span><span><b class="num">≈ ${n ? fmtHours(est.hours) : '0 min'}</b>${n ? ` <span class="t3">· done around <span class="num">${finishTxt}</span>, after the ${fmtHours(D.today.hoursQueued)} already queued</span>` : ''}</span></div>
    <div class="cqa-split">${bar}</div>
    <ul class="cqa-split-l">${nodeRows.map((r) => `<li><span class="cqa-sw ${r.off ? 'off' : r.id === 'forge' ? 'a' : 'b'}"></span><b>${r.id}</b><span class="num">${r.off ? '—' : r.k}</span><span class="t3">${esc(r.note)}</span></li>`).join('')}</ul>
    <div class="cqa-kv"><span class="t3">Disk</span><span><b class="num">${est.gb} GB</b> <span class="t3">of <span class="num">${A.disk.freeTB} TB</span> free on <span class="mono">${esc(A.disk.root)}</span></span></span></div>
    <div class="cqa-disk"><span style="width:${Math.max(n ? 0.6 : 0, (est.gb / freeGB) * 100)}%"></span></div>
  </div>`;
  const warn = dvSel.length ? `<div class="cqa-dvw">
    <p><span class="chip-warn">${I('triangle-alert')}${dvSel.length} will wait</span></p>
    <p><b>${dvSel.map((x) => esc(S.W(x.work).title)).join(', ')}</b> ${dvSel.length === 1 ? 'has' : 'have'} a Dolby Vision Profile 5 source. ${offProven.map((nd) => `${esc(nd.name)} has proven it but is offline`).concat(trialFail.map((nd) => `${esc(nd.name)}'s trial failed`)).join('; ')}. ${dvSel.length === 1 ? 'It waits' : 'They wait'} as no capable Node until one can.</p>
  </div>` : '';

  const others = destProfiles(S);
  const matchP = A.existingMatch && D.profiles.find((x) => x.id === A.existingMatch);
  const tnames = (pp) => pp.targets.map((t) => D.targets[t].name).join(' + ');
  const opt = (kind, inner) => `<div class="cqa-opt" role="radio" tabindex="0" aria-checked="${bf.dest.kind === kind}" data-dest="${kind}"><span class="cqa-radio"></span><div class="ltxt">${inner}</div></div>`;
  const dest = rest ? `<div class="cqa-dest" role="radiogroup" aria-labelledby="cqa-dest-h">
    <p class="cqa-sh" id="cqa-dest-h">The other <span class="num">${rest}</span> move to</p>
    ${matchP ? opt('reuse', `<b>${esc(matchP.name)}</b><span class="t3">Already matches: ${esc(tnames(matchP))} · <span class="num">${matchP.works}</span> Works</span>`) : ''}
    ${opt('create', `${bf.dest.kind === 'create' ? `<input class="cqa-name" data-bf-name value="${esc(bf.dest.name)}" aria-label="New Profile name" spellcheck="false">` : `<b>${matchP ? 'A new Profile' : esc(bf.dest.name)}</b>`}<span class="t3">${bf.dest.kind === 'create' ? 'New Profile · ' : ''}${esc(tnames(p))}, as ${esc(p.name)} was</span>`)}
    ${opt('other', `<b>Another Profile</b>${bf.dest.kind === 'other' ? `<select class="cqa-sel" data-bf-other aria-label="Profile">${others.map((o) => `<option value="${o.id}"${bf.dest.other === o.id ? ' selected' : ''}>${esc(o.name)} · ${esc(tnames(o))}</option>`).join('')}</select>` : `<span class="t3">${others.map((o) => esc(o.name)).join(', ')}</span>`}`)}
    ${bf.dest.kind === 'other' ? (() => { const o = D.profiles.find((x) => x.id === bf.dest.other); const lost = p.targets.filter((t) => !o.targets.includes(t)).map((t) => D.targets[t].name); return lost.length ? `<p class="cqa-note warn-t">${I('info')}${esc(o.name)} has no ${esc(lost.join(' or '))}; those files leave these Works.</p>` : ''; })() : ''}
  </div>` : `<div class="cqa-dest"><p class="cqa-sh">Nothing moves</p><p class="t3">Every Work in ${esc(p.name)} is selected.</p></div>`;

  const dn = destName(S);
  const def = `<div class="cqa-def">
    <p class="cqa-sh">Default for ${esc(ctd.plural)}</p>
    <p class="t2">${esc(p.name)} is the default Profile for ${esc(ctd.plural)}, so new ${esc(ctd.plural)} get ${esc(tg.name)} too.</p>
    <div class="seg-c cqa-defseg" role="radiogroup" aria-label="Default Profile for ${esc(ctd.plural)}">
      <button role="radio" class="seg-o" data-def="keep" aria-checked="${bf.def === 'keep' || !rest}">Keep ${esc(p.name)}</button>
      <button role="radio" class="seg-o" data-def="switch" aria-checked="${bf.def === 'switch' && !!rest}" title="Switch to ${esc(dn)}"${rest ? '' : ' disabled'}><span class="ell" data-dn>Switch to ${esc(dn)}</span></button>
    </div>
  </div>`;
  return { html: `${stats}${warn}${dest}${def}`, n, rest, est, dn };
}

function footer(S, sum) {
  const { esc, I, isPhone, fmtHours } = S;
  const label = sum.n && sum.rest ? `Convert ${sum.n} · move ${sum.rest}` : sum.n ? `Convert all ${sum.n}` : `Move all ${sum.rest}`;
  const mini = isPhone ? `<p class="cqa-pmini num"><b>${sum.n}</b> selected · ≈ ${sum.n ? fmtHours(sum.est.hours) : '0 min'} · ${sum.est.gb} GB</p>` : '';
  return `${mini}<div class="cqa-fbtns"><button class="btn ghost" data-bf-cancel>Cancel</button><button class="btn pri cqa-confirm" data-bf-confirm>${I('check')}<span data-bf-label>${esc(label)}</span></button></div>`;
}

// ───────────────────────────── wiring

export function wire(S, screen) {
  if (screen === 'queue') return wireQueue(S);
  wireBackfill(S);
}

function wireQueue(S) {
  const app = document.getElementById('app');
  const wireLanes = () => S.wireAmbient('[data-work]', 'dune-2');
  wireLanes();
  app.addEventListener('click', (e) => {
    const q = (s) => e.target.closest(s);
    let el;
    if ((el = q('[data-menu]'))) {
      e.stopPropagation();
      const c = S.D.conversions.find((x) => x.id === el.dataset.menu), w = S.W(c.work), t = S.target(c.target).name;
      const others = S.D.nodes.filter((n) => n.id !== c.state.node && n.status === 'online').map((n) => n.name);
      return S.openMenu(el, { title: w.title, sub: [S.itemLabel(c), t].filter(Boolean).join(' · '), items: [
        ['Move up', 'arrow-up', null, `${w.title} moved up. It takes the next free slot on ${c.state.node}.`],
        ['Pin Target to another Node', 'pin', null, `Pinned ${t} for ${w.title} to ${others[0] || 'another Node'}.`],
        [c.state.is === 'running' ? 'Pause' : 'Hold', 'pause', null, `${w.title} paused. It keeps its place in the lane.`],
        ['Cancel', 'x', 'danger', `Cancelled the ${t} Conversion for ${w.title}.`],
        ['Open Work', 'film', null, 'Opening a Work is not in this prototype'],
      ] });
    }
    if ((el = q('[data-node-menu]'))) {
      e.stopPropagation();
      const n = S.node(el.dataset.nodeMenu);
      return S.openMenu(el, { title: n.name, sub: n.hw, items: [
        ['Change the slot limit', 'gauge'], ['Edit active hours', 'clock'], ['Rerun trials', 'flask-conical', null, `Rerunning trials on ${n.name}…`], ['Pause this Node', 'pause'], ['Open Node', 'network'],
      ] });
    }
    if ((el = q('[data-bf-toggle]'))) {
      const open = el.getAttribute('aria-expanded') !== 'true';
      el.setAttribute('aria-expanded', open);
      el.parentNode.querySelector('.cqa-bf-b').hidden = !open;
      return;
    }
    if ((el = q('[data-show-log]'))) {
      const card = el.closest('article'), log = card.querySelector('[data-log]');
      log.hidden = !log.hidden;
      el.setAttribute('aria-expanded', !log.hidden);
      el.lastChild.textContent = log.hidden ? 'Show log' : 'Hide log';
      return;
    }
    if ((el = q('[data-lane]'))) {
      app.querySelectorAll('[data-lane]').forEach((b) => b.setAttribute('aria-checked', b === el));
      const M = model(S);
      app.querySelector('[data-plane]').innerHTML = phoneLane(S, M.lanes.find((l) => l.n.id === el.dataset.lane));
      window.lucide && window.lucide.createIcons();
      wireLanes();
    }
  });
}

function wireBackfill(S) {
  const app = document.getElementById('app');
  const dlg = app.querySelector('[data-bf-dlg]');
  const A = S.D.backfillAsk, works = S.D.cinemaWorks;
  const filters = FILTERS(A, works);

  const refresh = () => {
    const act = document.activeElement, key = act && act.matches('[data-bf-name]') ? act.selectionStart : null;
    const sum = summary(S);
    dlg.querySelector('[data-bf-sum]').innerHTML = sum.html;
    dlg.querySelector('[data-bf-foot]').innerHTML = footer(S, sum);
    dlg.querySelector('[data-bf-selc]').innerHTML = `<b class="num">${sum.n}</b> selected · <span class="num">${sum.rest}</span> move`;
    dlg.querySelectorAll('[data-pick]').forEach((b) => b.setAttribute('aria-checked', bf.sel.has(b.dataset.pick)));
    dlg.querySelectorAll('[data-filter]').forEach((b) => {
      const ids = filters.find((f) => f.id === b.dataset.filter).ids.filter((id) => works.includes(id));
      b.setAttribute('aria-pressed', ids.length === bf.sel.size && ids.every((id) => bf.sel.has(id)));
    });
    window.lucide && window.lucide.createIcons();
    if (key != null) { const inp = dlg.querySelector('[data-bf-name]'); inp.focus(); inp.setSelectionRange(key, key); }
  };
  const setOpen = (open) => {
    dlg.hidden = !open;
    if (open) { refresh(); dlg.querySelector('[data-bf-confirm]').focus({ preventScroll: true }); }
  };
  refresh();
  S.wireAmbient('[data-pick]', [...bf.sel][0] || works[0]);

  app.addEventListener('click', (e) => {
    const q = (s) => e.target.closest(s);
    let el;
    if (q('[data-open-backfill]')) return setOpen(true);
    if (!dlg.contains(e.target)) return;
    if ((el = q('[data-bf-cancel]'))) { setOpen(false); return S.toast('Cancelled. Nothing was saved; the editor still has your changes.'); }
    if ((el = q('[data-bf-confirm]'))) {
      const sum = summary(S);
      setOpen(false);
      return S.toast(`Saved ${S.D.profiles.find((p) => p.id === A.profile).name}. ${sum.n} Conversions queued as Backfill${sum.rest ? `; ${sum.rest} Works moved to ${sum.dn}` : ''}${bf.def === 'switch' && sum.rest ? `, now the default` : ''}.`);
    }
    if ((el = q('[data-pick]'))) { const id = el.dataset.pick; bf.sel.has(id) ? bf.sel.delete(id) : bf.sel.add(id); return refresh(); }
    if ((el = q('[data-filter]'))) { bf.sel = new Set(filters.find((f) => f.id === el.dataset.filter).ids.filter((id) => works.includes(id))); return refresh(); }
    if ((el = q('[data-def]'))) { if (el.disabled) return; bf.def = el.dataset.def; return refresh(); }
    if ((el = q('[data-dest]')) && !q('input, select')) {
      const k = el.dataset.dest;
      if (bf.dest.kind === k) return;
      bf.dest.kind = k;
      if (k === 'other' && !bf.dest.other) bf.dest.other = destProfiles(S)[0].id;
      if (k === 'reuse') bf.dest.id = A.existingMatch;
      refresh();
      if (k === 'create') { const inp = dlg.querySelector('[data-bf-name]'); inp.focus(); inp.select(); }
    }
  });
  dlg.addEventListener('keydown', (e) => {
    if ((e.key === 'Enter' || e.key === ' ') && e.target.matches('[data-dest]')) { e.preventDefault(); e.target.click(); }
    if (e.key === 'Escape' && !document.querySelector('.menu:not([hidden])')) { setOpen(false); }
  });
  dlg.addEventListener('input', (e) => {
    if (e.target.matches('[data-bf-q]')) {
      const v = e.target.value.trim().toLowerCase();
      dlg.querySelectorAll('.cqa-tile').forEach((t) => { t.hidden = v && !t.dataset.title.includes(v); });
    }
    if (e.target.matches('[data-bf-name]')) {
      bf.dest.name = e.target.value;
      const dn = destName(S);
      dlg.querySelectorAll('[data-dn]').forEach((s) => { s.textContent = `Switch to ${dn}`; });
    }
  });
  dlg.addEventListener('change', (e) => {
    if (e.target.matches('[data-bf-other]')) { bf.dest.other = e.target.value; refresh(); }
  });
}
