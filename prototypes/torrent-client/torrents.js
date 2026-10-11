// PROTOTYPE · the built-in torrent client's Torrents screen and torrent inspector on Direction C. Throwaway.
import { mount, h, openSheet, closeAll, toast, setLead, phoneSection } from './shell.js';

let C, D, phone, root;
const { esc, I, art, dot, hm, md } = h;
const ago = (iso) => {
  const m = Math.round((D.now - new Date(iso)) / 60000);
  if (m < 60) return `${m} min ago`;
  const hr = Math.floor(m / 60);
  return hr < 24 ? `${hr} h ago` : `${Math.floor(hr / 24)} d ago`;
};

const W = (id) => D.works[id];
const client = (id) => D.clients.find((c) => c.id === id);
const node = (id) => D.nodes.find((n) => n.id === id);
const tracker = (id) => D.trackers.find((t) => t.id === id);
const job = (id) => D.jobs.find((j) => j.id === id);

const r2 = (x) => x.toFixed(2);
const rate = (x) => (x ? x.toFixed(1) : '0');
const gb = (x) => (x >= 100 ? `${x.toFixed(0)} GB` : `${x.toFixed(1)} GB`);
const dur = (hrs) => {
  if (hrs <= 0) return '0 h';
  if (hrs < 1) return `${Math.max(1, Math.round(hrs * 60))} min`;
  if (hrs < 100) return `${Math.round(hrs)} h`;
  const H = Math.round(hrs), d = Math.floor(H / 24), r = H % 24;
  return d >= 10 || !r ? `${Math.round(hrs / 24)} d` : `${d} d ${r} h`;
};
const when = (iso) => `${md(iso)}, ${hm(iso)}`;
const minsUntil = (iso) => Math.max(0, Math.round((new Date(iso) - D.now) / 60000));
const fill = (tpl, o) => tpl.replace(/\{(\w+)(?::(\d+))?\}/g, (_, k, pad) => (pad ? String(o[k]).padStart(+pad, '0') : String(o[k])));
const sep = '<span class="s-sep" aria-hidden="true">·</span>';
const cov = (t) => D.coverageLabel(t.work, t.coverage);
const nw = (html) => html.replace(/(-SS\d{4}-)/g, '<span class="nw">$1</span>');

// ───────────────────────────── view model: everything a row, a problem or the inspector reads about one torrent

const STATE = { downloading: 'Downloading', queued: 'Queued', checking: 'Checking', seeding: 'Seeding', stalled: 'Stalled', paused: 'Paused', error: 'Tracker error', met: 'Goal met' };

function view(t) {
  const w = W(t.work), c = client(t.client), n = node(c.node), tr = tracker(t.tracker), ct = D.contentTypes[w.type];
  const g = D.goalFor(t), gp = D.goalProgress(t), hn = D.hnr(t);
  const external = c.kind !== 'built-in', done = !!t.completedAt;
  const reasons = [];
  if (t.state === 'error') reasons.push({ tone: 'fail', text: `Unregistered by ${tr.name}` });
  if (t.stale) reasons.push({ tone: 'fail', text: `Last known at ${hm(c.lastSeen)}` });
  if (hn && hn.atRisk) reasons.push({ tone: 'warn', text: `H&R in ${dur(hn.leftInWindowH)}, needs ${dur(hn.needH)}` });
  if (t.state !== 'error' && t.announce.status === 'error') reasons.push({ tone: 'warn', text: 'Announces timing out' });
  if (t.state === 'stalled') reasons.push({ tone: 'warn', text: 'No peers to seed to' });
  if (t.state === 'downloading' && !t.down && c.listen && !c.listen.reachable) reasons.push({ tone: 'warn', text: `Port ${c.listen.port} closed` });
  const buckets = [];
  if (['downloading', 'queued'].includes(t.state) || (t.state === 'checking' && !done)) buckets.push('downloading');
  if (['seeding', 'stalled', 'paused'].includes(t.state) || (t.state === 'checking' && done)) buckets.push(gp.kind === 'met' ? 'met' : 'seeding');
  if (t.state === 'met') buckets.push('met');
  if (reasons.length) buckets.push('problems');
  const rank = reasons.some((r) => r.tone === 'fail') ? 0 : reasons.length ? 1 : buckets.includes('downloading') ? 2 : buckets.includes('met') ? 4 : 3;
  return { t, w, c, n, tr, ct, g, gp, hn, external, reasons, buckets, rank };
}
let V;
const byRank = (a, b) => a.rank - b.rank;

const isDl = (v) => ['downloading', 'queued'].includes(v.t.state) || v.t.state === 'checking';

function stateLabel(v) {
  if (v.gp.kind === 'held') return 'Held by rule';
  return STATE[v.t.state];
}
function stateTone(v) {
  if (v.reasons.some((r) => r.tone === 'fail')) return 'fail';
  if (v.t.state === 'downloading') return 'draft';
  if (v.reasons.length) return 'warn';
  if (v.t.state === 'met' || v.gp.kind === 'met') return 'met';
  return 'idle';
}
function stateSub(v) {
  const t = v.t;
  if (v.reasons.length) return `<span class="${v.reasons[0].tone}-t">${esc(v.reasons[0].text)}</span>`;
  if (t.state === 'queued') return 'Waiting for a slot';
  if (t.state === 'checking') return t.completedAt ? 'Verifying after restart' : 'Verifying pieces';
  if (t.state === 'met') return t.removeAt ? `Removing in ${minsUntil(t.removeAt)} min` : esc(D.whenMetLabel[v.g.whenMet]);
  if (v.gp.kind === 'held') return `${esc(v.tr.name)} rule holds it`;
  if (v.external && t.state !== 'downloading') return `${esc(v.c.name)} owns seeding`;
  return `${t.peers} peers`;
}

// ───────────────────────────── problems: what broke, its cause and the recovery

function problems() {
  const out = [];
  const dead = D.clients.find((c) => c.status === 'unreachable');
  if (dead) {
    const n = node(dead.node);
    out.push({ id: 'client', tone: 'fail', chip: 'Client unreachable', icon: 'unplug', at: dead.lastSeen,
      subject: `${dead.name} on ${n.name}`, sub: `External · ${dead.totals.torrents} torrents`,
      cause: `${dead.error}.`, recovery: `Showing its last known data from ${hm(dead.lastSeen)}. Its torrents keep seeding if ${dead.name} is still running. Check it at ${dead.host}.`,
      actions: [{ label: 'Test connection', icon: 'plug-zap', cls: 'fail', toast: 'Testing a Download Client is not in this prototype' }, { label: 'Open client', go: 'clients' }] });
  }
  V.filter((v) => v.t.state === 'error').forEach((v) => {
    out.push({ id: `err-${v.t.id}`, tone: 'fail', chip: 'Unregistered', icon: 'circle-x', at: v.t.announce.last, work: v.w,
      subject: v.w.title, sub: `${cov(v.t)} · ${v.t.target} · ${v.tr.name}`, t: v.t.id,
      cause: `${v.tr.name} removed this release: ${v.t.announce.msg.split(': ').slice(1).join(': ') || v.t.announce.msg}.`,
      recovery: 'Seeding is over and no hit and run counts. Remove the torrent; the library file stays.',
      actions: [{ label: 'Remove torrent', icon: 'trash-2', cls: 'fail', rm: v.t.id }, { label: 'Open Work', go: 'work' }] });
  });
  const risk = V.filter((v) => v.hn && v.hn.atRisk).sort((a, b) => a.hn.leftInWindowH - b.hn.leftInWindowH);
  if (risk.length) {
    const trs = [...new Set(risk.map((v) => v.tr.name))].join(' and ');
    out.push({ id: 'hnr', tone: 'warn', chip: 'Hit and run risk', icon: 'timer', subject: `${risk.length} torrents on ${trs}`,
      cause: 'Their windows close before they reach the minimum.', torrents: risk.map((v) => v.t.id),
      links: risk.map((v) => `<button class="lnk pr-t" data-open-t="${v.t.id}"><b>${esc(v.w.title)}</b> ${esc(cov(v.t))}</button> <span class="num">needs ${dur(v.hn.needH)} in ${dur(v.hn.leftInWindowH)}</span>`),
      recovery: 'Keep them seeding. Pausing or removing one now counts against your account.', actions: [] });
  }
  D.trackers.filter((tr) => tr.status === 'degraded').forEach((tr) => {
    const n = V.filter((v) => v.tr.id === tr.id).length;
    out.push({ id: `trk-${tr.id}`, tone: 'warn', chip: 'Announces timing out', icon: 'radio-tower', at: tr.since, subject: tr.name,
      cause: `No answer to announces. ${n} torrents keep seeding, but ${tr.name} may not credit seeding time until it answers.`,
      recovery: 'Retrying every 30 min.', actions: [{ label: 'Force reannounce', toast: 'Force reannounce is not in this prototype' }] });
  });
  D.trackers.filter((tr) => tr.status === 'not-allowed').forEach((tr) => {
    const r = client(tr.route), n = node(r.node), b = D.clients.find((c) => c.kind === 'built-in');
    out.push({ id: `deny-${tr.id}`, tone: 'warn', chip: 'Client not allowed', icon: 'shield-x', at: tr.since, subject: tr.name,
      cause: `The built-in client (${b.peerId}) is not on its allow list yet.`,
      recovery: `Grabs from ${tr.name} go to ${r.name} on ${n.name} until ${tr.name} adds ${b.peerId}.`,
      actions: [{ label: 'Open routing', go: 'clients' }] });
  });
  D.clients.filter((c) => c.listen && !c.listen.reachable).forEach((c) => {
    const n = node(c.node);
    out.push({ id: `port-${c.id}`, tone: 'warn', chip: 'Port closed', icon: 'door-closed', subject: n.name,
      cause: `Port ${c.listen.port} is closed from outside, so peers reach it only outbound and uploads are slower.`,
      recovery: `Forward TCP and UDP ${c.listen.port} to ${n.name} on your router, or turn on UPnP.`,
      actions: [{ label: 'Test port', toast: 'Testing a port is not in this prototype' }] });
  });
  return out;
}

function actBtn(a, extra = '') {
  const attr = a.go ? `data-go="${a.go}"` : a.rm ? `data-rm="${a.rm}"` : `data-toast="${esc(a.toast)}"`;
  return `<button class="btn ${a.cls || ''} ${extra}" ${attr}>${a.icon ? I(a.icon) : ''}${esc(a.label)}</button>`;
}

function failWide(p) {
  return nw(`<div class="pr" id="pr-${p.id}">
    <div class="fail-h"><span class="chip-fail">${I(p.icon)}${esc(p.chip)}</span><span class="fail-dim num">${hm(p.at)} · ${ago(p.at)}</span></div>
    <div class="pr-id">${p.work ? art(p.work, 'thumb-md') : `<span class="pr-ic">${I('hard-drive')}</span>`}<div>${p.t ? `<button class="lnk" data-open-t="${p.t}"><b>${esc(p.subject)}</b></button>` : `<b>${esc(p.subject)}</b>`}<p class="t2">${esc(p.sub)}</p></div></div>
    <dl class="fail-kv"><dt>Cause</dt><dd>${esc(p.cause)}</dd><dt>Recovery</dt><dd>${esc(p.recovery)}</dd></dl>
    <div class="pr-act">${p.actions.map((a) => actBtn(a, 'sm')).join('')}</div>
  </div>`);
}

function warnWide(p) {
  const body = p.links ? `${esc(p.cause)}<span class="pr-more"> ${p.links.join('<span class="warn-dim">;</span> ')}. <span class="warn-dim">${esc(p.recovery)}</span></span>` : `<span class="warn-dim">${esc(p.cause)} ${esc(p.recovery)}</span>`;
  return nw(`<div class="warn-line pr-w" id="pr-${p.id}"><span class="chip-warn">${I(p.icon)}${esc(p.chip)}</span>
    <p><b>${esc(p.subject)}.</b> ${body}</p>
    ${p.at ? `<span class="warn-dim num pr-since">since ${hm(p.at)}</span>` : ''}
    ${p.actions.map((a) => actBtn({ ...a, cls: 'warn' }, 'sm')).join('')}</div>`);
}

function problemsWide(ps) {
  const f = ps.filter((p) => p.tone === 'fail'), w = ps.filter((p) => p.tone === 'warn');
  return `<section class="probs fail" id="problems" aria-label="Problems">
    ${f.length ? `<div class="fail-main pr-grid">${f.map(failWide).join('')}</div>` : ''}
    ${w.map(warnWide).join('')}
  </section>`;
}

function problemsPhone(ps) {
  const f = ps.filter((p) => p.tone === 'fail'), w = ps.filter((p) => p.tone === 'warn');
  return `<section class="pfail pr-ph" id="problems" aria-label="Failures">${f.map((p) => `<div class="pr-pi">
      <div class="pfail-h"><span class="chip-fail">${I(p.icon)}${esc(p.chip)}</span><span class="fail-dim num">${ago(p.at)}</span></div>
      <div class="pfail-id">${p.work ? art(p.work, 'thumb-md') : `<span class="pr-ic">${I('hard-drive')}</span>`}<div><b>${esc(p.subject)}</b><p class="t2">${esc(p.sub)}</p></div></div>
      <p class="pr-c">${nw(esc(p.cause))} <span class="fail-dim">${nw(esc(p.recovery))}</span></p>
      <div class="pfail-act">${p.actions.map((a) => actBtn(a)).join('')}</div></div>`).join('')}</section>
    <section class="pwarn pr-wl" aria-label="Warnings">${w.map((p) => `<button class="pr-wi" data-pr="${p.id}" aria-haspopup="dialog"><span class="chip-warn">${I(p.icon)}${esc(p.chip)}</span><span class="pr-ws ell">${esc(p.subject)}</span>${I('chevron-right', 'pr-chev')}</button>`).join('')}</section>`;
}

function problemSheet(id, from) {
  const p = PROBS.find((x) => x.id === id);
  openSheet(nw(`<div class="psh" role="dialog" aria-modal="true" aria-labelledby="psh-h">
    <p class="sheet-t"><span class="chip-warn">${I(p.icon)}${esc(p.chip)}</span><b id="psh-h">${esc(p.subject)}</b>${p.at ? `<span class="t3 num">since ${hm(p.at)}</span>` : ''}</p>
    <dl class="psh-kv"><dt>Cause</dt><dd>${esc(p.cause)}</dd>${p.links ? `<dd class="psh-links">${p.links.map((l) => `<p>${l}</p>`).join('')}</dd>` : ''}<dt>Recovery</dt><dd>${esc(p.recovery)}</dd></dl>
    ${p.actions.map((a) => actBtn({ ...a, cls: 'warn' })).join('')}
    <button class="sheet-x" data-close>Close</button></div>`), from);
}

// ───────────────────────────── progress: the download bar glows; the seeding goal is a two-lane meter with the tracker's floor

function liveOrStatic(t) {
  const j = t.job && t.state === 'downloading' ? job(t.job) : null;
  return { j, p: j ? j.progress : t.progress };
}

function etaText(t) {
  if (!t.down) return 'no peers sending';
  return C.format.eta(((t.sizeGB - t.downloadedGB) * 1024) / t.down / 60);
}

function dlBar(v, big = false) {
  const t = v.t, { j, p } = liveOrStatic(t);
  const glow = t.state === 'downloading';
  const label = `${v.w.title} ${cov(t)}, ${t.state === 'checking' ? 'verifying pieces' : 'download progress'}`;
  const pctTxt = C.format.pct(p);
  const pct = j ? `<span data-live-pct="${j.id}">${pctTxt}</span>` : pctTxt;
  let tail;
  if (j) tail = `<span data-live-eta="${j.id}">${C.format.eta(j.etaMin)}</span> left${sep}<span data-live-rate="${j.id}">${C.format.rate(j)}</span>`;
  else if (t.state === 'queued') tail = `${gb(t.sizeGB)} waiting`;
  else if (t.state === 'checking') tail = 'verifying pieces';
  else tail = t.down ? `${etaText(t)} left${sep}${rate(t.down)} MB/s` : `<span class="warn-t">no peers sending</span>`;
  return `<div class="pg${big ? ' pg-lg' : ''}"><div class="bar${glow ? '' : ' bar-q'}" role="progressbar" aria-valuemin="0" aria-valuemax="100" aria-valuenow="${Math.floor(p * 100)}" aria-label="${esc(label)}" style="--p:${p}" ${j ? `data-live-progress="${j.id}"` : ''}><span class="fill"></span></div><p class="pg-t num"><b>${pct}</b>${sep}${tail}</p></div>`;
}

function lanes(v) {
  const t = v.t, g = v.g, rule = v.tr.rule || {};
  const forever = g.mode === 'forever';
  const out = [];
  const rMax = forever ? rule.minRatio : g.ratio;
  if (rMax) out.push({ k: 'Ratio', cur: t.ratio, max: rMax, floor: rule.minRatio, fmt: r2 });
  const hMax = forever ? rule.minHours : g.hours;
  if (hMax) out.push({ k: 'Seeded', cur: t.seededH, max: hMax, floor: rule.minHours, fmt: dur });
  return out.map((l) => ({ ...l, p: Math.min(1, l.cur / l.max), f: l.floor != null ? Math.min(1, l.floor / l.max) : null, floorMet: l.floor != null && l.cur >= l.floor }));
}

function goalText(v) {
  const ls = lanes(v);
  if (v.g.mode === 'forever') return v.gp.ruleMet ? 'Seeds until removed · rule met' : `Seeds until removed · rule ${ls.map((l) => `${l.fmt(l.cur)} of ${l.fmt(l.max)}`).join(' or ')}`;
  return ls.map((l) => (l.k === 'Ratio' ? `Ratio ${l.fmt(l.cur)} of ${l.fmt(l.max)}` : `${l.fmt(l.cur)} of ${l.fmt(l.max)}`)).join(' · ');
}

function goalMeter(v, big = false) {
  const ls = lanes(v), k = v.gp.kind, risk = v.hn && v.hn.atRisk;
  const cls = k === 'met' ? ' gm-met' : k === 'held' ? ' gm-held' : '';
  const floorTxt = (l) => `${v.tr.name} minimum ${l.fmt(l.floor)}${l.floorMet ? ', met' : ', not met yet'}`;
  const lane = (l) => {
    const tick = l.f != null ? `<span class="gm-k${l.floorMet ? ' k-ok' : risk ? ' risk' : ''}" style="left:${(l.f * 100).toFixed(1)}%" title="${esc(floorTxt(l))}"></span>` : '';
    if (!big) return `<span class="gm-l"><span class="gm-f" style="width:${(l.p * 100).toFixed(1)}%"></span>${tick}</span>`;
    const lab = l.f != null ? `<span class="gm-kl${l.f > 0.66 ? ' r' : ''}${l.floorMet ? '' : risk ? ' risk' : ''}" style="left:${(l.f * 100).toFixed(1)}%">${esc(v.tr.name)} minimum ${l.fmt(l.floor)}</span>` : '';
    return `<div class="gmb-r"><p class="gmb-h"><span>${l.k}</span><span class="num"><b>${l.fmt(l.cur)}</b> <span class="t3">of ${l.fmt(l.max)}</span></span></p><span class="gm-l">${'<span class="gm-f" style="width:' + (l.p * 100).toFixed(1) + '%"></span>'}${tick}</span>${lab}</div>`;
  };
  const p = v.g.mode === 'forever' ? Math.max(...ls.map((l) => l.p)) : v.gp.p ?? 0;
  const aria = `Seeding goal, ${goalText(v)}${ls.filter((l) => l.f != null).map((l) => `. ${floorTxt(l)}`).join('')}`;
  if (big) return `<div class="gmb${cls}" role="progressbar" aria-valuemin="0" aria-valuemax="100" aria-valuenow="${Math.round(p * 100)}" aria-label="${esc(aria)}">${ls.map(lane).join('')}</div>`;
  return `<div class="pg"><div class="gm${cls}" role="progressbar" aria-valuemin="0" aria-valuemax="100" aria-valuenow="${Math.round(p * 100)}" aria-label="${esc(aria)}">${ls.map(lane).join('')}</div><p class="pg-t num">${k === 'met' ? `<b>Met</b>${sep}` : k === 'held' ? `<b class="warn-t">Held</b>${sep}` : ''}${esc(goalText(v))}</p></div>`;
}

function progCell(v) {
  const t = v.t;
  if (isDl(v)) return dlBar(v);
  if (t.state === 'error') return `<p class="pg-t pg-x"><span class="fail-t">Seeding stopped</span>${sep}no hit and run counts</p>`;
  if (v.external) return `<p class="pg-t pg-x">${I('external-link')}<span>Seeding rules live in ${esc(v.c.name)}</span></p>`;
  return goalMeter(v);
}

// ───────────────────────────── torrent rows (one markup; container queries reflow it from table row to phone card)

const COLS = {
  node: ['work', 'state', 'prog', 'ratio', 'rate', 'trk', 'menu'],
  state: ['work', 'state', 'prog', 'ratio', 'rate', 'where', 'trk', 'menu'],
  tracker: ['work', 'state', 'prog', 'ratio', 'rate', 'where', 'menu'],
};
const HEAD = { work: 'Work', state: 'State', prog: 'Progress or seeding goal', ratio: 'Ratio', rate: 'MB/s', where: 'Node · client', trk: 'Tracker', menu: '<span class="sr">Actions</span>' };

function trkChip(v) {
  const a = v.t.announce, tone = a.status === 'error' ? (v.t.state === 'error' ? 'fail' : 'warn') : null;
  const msg = a.status === 'error' ? a.msg : a.status === 'updating' ? 'Updating' : 'Working';
  return `<span class="chip node trk-c" title="${esc(`${v.tr.name}: ${msg}`)}">${tone ? dot(tone) : ''}${esc(v.tr.name)}<span class="sr">, ${esc(msg)}</span></span>`;
}

function bigCell(v) {
  const t = v.t;
  if (isDl(v)) { const { j, p } = liveOrStatic(t); return j ? `<b data-live-pct="${j.id}">${C.format.pct(p)}</b>` : `<b>${C.format.pct(p)}</b>`; }
  return `<b>${r2(t.ratio)}</b><span class="t3">ratio</span>`;
}

function row(v, layout) {
  const t = v.t, cols = COLS[layout];
  const cell = {
    work: `<div class="c-work" role="cell">${art(v.w, 'thumb-tr')}<span class="tw"><b class="ell">${esc(v.w.title)}</b><span class="t3 ell">${esc(cov(t))} · ${esc(t.target)}</span></span></div>`,
    state: `<div class="c-state" role="cell"><p class="tst">${dot(stateTone(v))}<b>${stateLabel(v)}</b></p><p class="st-sub t3 ell">${stateSub(v)}</p><p class="nm t3 ell">${esc(v.n.name)} · ${esc(v.c.name)}${layout !== 'tracker' ? ` · ${esc(v.tr.name)}` : ''}${sep}<span class="num">↓ ${rate(t.down)} ↑ ${rate(t.up)}</span></p></div>`,
    prog: `<div class="c-prog" role="cell">${progCell(v)}</div>`,
    ratio: `<div class="c-ratio num" role="cell">${r2(t.ratio)}</div>`,
    rate: `<div class="c-rate num" role="cell"><span class="${t.down ? '' : 't3'}">${I('arrow-down')}${rate(t.down)}</span><span class="${t.up ? '' : 't3'}">${I('arrow-up')}${rate(t.up)}</span></div>`,
    where: `<div class="c-where" role="cell"><span class="ell">${esc(v.n.name)}<span class="t3"> · ${esc(v.c.name)}</span></span></div>`,
    trk: `<div class="c-trk" role="cell">${trkChip(v)}</div>`,
    menu: `<div class="c-menu" role="cell"><button class="ib" data-menu="${t.id}" aria-haspopup="${phone ? 'dialog' : 'menu'}" aria-label="Actions for ${esc(v.w.title)} ${esc(cov(t))}">${I('ellipsis')}</button></div>`,
  };
  const q = `${v.w.title} ${t.release} ${v.tr.name}`.toLowerCase();
  return `<div class="tr${t.stale ? ' stale' : ''}" role="row" tabindex="0" data-t="${t.id}" data-work="${v.w.id}" data-amb data-ct="${v.w.type}" data-node="${v.n.id}" data-b="${v.buckets.join(' ')}" data-q="${esc(q)}" aria-label="${esc(`${v.w.title}, ${cov(t)}, ${stateLabel(v)}`)}">
    ${cols.map((c) => cell[c]).join('')}<div class="c-big num" role="cell">${bigCell(v)}</div></div>`;
}

const head = (layout) => `<div class="tr thead" role="row">${COLS[layout].map((c) => `<div class="c-${c}" role="columnheader">${HEAD[c]}</div>`).join('')}</div>`;
const empty = () => '<div class="tempty" role="row" hidden><div role="cell">No torrents here match the filter.</div></div>';

// ───────────────────────────── layout A · By Node

function clientHead(c) {
  const ext = c.kind !== 'built-in', dead = c.status === 'unreachable', tt = c.totals;
  const status = dead ? `<span class="chip-fail">${I('unplug')}Unreachable</span>` : c.listen && !c.listen.reachable ? `<span class="chip-warn">${I('door-closed')}Port ${c.listen.port} closed</span>` : '';
  const meta = ext
    ? `${esc(c.version)} · <span class="mono">${esc(c.host)}</span> · category ${esc(c.category)}`
    : `${esc(c.version)} · peer ID <span class="mono">${esc(c.peerId)}</span> · port <span class="mono">${c.listen.port}</span> ${c.listen.reachable ? 'open' : '<span class="warn-t">closed</span>'}`;
  const note = dead ? `<p class="cg-note fail-t">Last known data from ${hm(c.lastSeen)}. ${esc(c.error)}.</p>`
    : ext ? `<p class="cg-note t2">${I('info')}Keeps its own seeding rules. SlipStream grabs to it and imports from it.${c.why ? ` ${esc(c.why)}` : ''}</p>` : '';
  return `<div class="cg-h" role="row"><div role="cell" class="cg-c">
    <span class="cg-ic">${I(ext ? 'plug' : 'magnet')}</span>
    <div class="cg-t"><p class="cg-n"><b>${ext ? esc(c.name) : 'Built-in client'}</b>${ext ? '<span class="chip">External</span>' : ''}${status}</p><p class="t3 cg-m">${meta}</p>${note}</div>
    <div class="cg-r"><p class="num t2">${tt.torrents} torrents${sep}${tt.downloading} downloading${sep}${tt.seeding} seeding</p><p class="num cg-rt">${dead ? '<span class="t3">rates unknown</span>' : `${I('arrow-down')}${rate(tt.downRate)} ${I('arrow-up')}${rate(tt.upRate)} MB/s`}</p></div>
    ${phone ? '' : `<button class="btn ghost sm" data-go="${ext ? 'clients' : 'goals'}">${ext ? 'Routing' : 'Seeding goals'}</button>`}
  </div></div>`;
}

function moreRow(c) {
  const n = D.moreTorrents[c.id];
  return n ? `<div class="cg-more" role="row"><div role="cell"><button class="btn ghost sm" data-toast="Paging through every torrent is not in this prototype">${I('chevron-down')}${n} more on ${c.kind === 'built-in' ? 'the built-in client' : esc(c.name)}</button></div></div>` : '';
}

function layoutNode() {
  return D.downloadingNodes().sort((a, b) => V.filter((v) => v.n.id === b.id).length - V.filter((v) => v.n.id === a.id).length).map((n) => {
    const cs = D.clients.filter((c) => c.node === n.id).sort((a, b) => (a.kind === 'built-in' ? -1 : 0) - (b.kind === 'built-in' ? -1 : 0));
    const down = cs.reduce((s, c) => s + (c.status === 'ok' ? c.totals.downRate : 0), 0), up = cs.reduce((s, c) => s + (c.status === 'ok' ? c.totals.upRate : 0), 0);
    const free = n.disk.freeTB / n.disk.totalTB;
    const groups = cs.map((c) => `<div class="cg" role="rowgroup" data-grp>${clientHead(c)}${V.filter((v) => v.c.id === c.id).sort(byRank).map((v) => row(v, 'node')).join('')}${empty()}${moreRow(c)}</div>`).join('');
    const table = `<div class="tl tl-node" role="table" aria-label="Torrents on ${esc(n.name)}">${phone ? '' : head('node')}${groups}</div>`;
    const anyProb = V.some((v) => v.n.id === n.id && v.reasons.length);
    if (phone) return phoneSection(`<span class="mono">${esc(n.name)}</span>`, `<p class="nd-ph t3">${n.kind} · ${esc(n.where)}${sep}${n.disk.freeTB} TB free${sep}↓ ${rate(down)} ↑ ${rate(up)} MB/s</p>${table}`);
    return `<section class="nd" aria-labelledby="nd-${n.id}">
      <header class="nd-h">${dot(anyProb ? 'warn' : 'draft')}<h3 id="nd-${n.id}" class="mono">${esc(n.name)}</h3><span class="chip node">${n.kind}</span><span class="t3">${esc(n.where)}</span><span class="grow"></span>
        <span class="nd-disk num"><span class="t2"><b>${n.disk.freeTB} TB</b> free of ${n.disk.totalTB} TB</span><span class="meter" role="meter" aria-valuemin="0" aria-valuemax="${n.disk.totalTB}" aria-valuenow="${(n.disk.totalTB - n.disk.freeTB).toFixed(1)}" aria-label="Download disk used on ${esc(n.name)}"><span style="width:${((1 - free) * 100).toFixed(1)}%"></span></span></span>
        <span class="nd-rt num">${I('arrow-down')}${rate(down)}<span class="rates-sep"></span>${I('arrow-up')}${rate(up)} MB/s</span></header>
      ${table}</section>`;
  }).join('');
}

// ───────────────────────────── layout B · By state

const BUCKETS = [['downloading', 'Downloading'], ['seeding', 'Seeding'], ['problems', 'Problems'], ['met', 'Goal met'], ['all', 'All']];
const bucketOrder = () => (phone ? [BUCKETS[4], ...BUCKETS.slice(0, 4)] : BUCKETS);

function segs(key, label, opts, cur) {
  return `<div class="seg-c" role="radiogroup" aria-label="${esc(label)}" data-segs="${key}">${opts.map(([v, l, n]) => `<button role="radio" class="seg-o" data-seg="${v}" aria-checked="${v === cur}"><span>${esc(l)}</span>${n != null ? `<span class="seg-n num">${n}</span>` : ''}</button>`).join('')}</div>`;
}
const bucketSeg = () => segs('bucket', 'Show torrents that are', bucketOrder().map(([b, l]) => [b, l, b === 'all' ? V.length : V.filter((v) => v.buckets.includes(b)).length]), F.bucket);
const nodeSeg = () => segs('node', 'Show torrents on', [['', 'All Nodes', V.length], ...D.downloadingNodes().map((n) => [n.id, n.name, V.filter((v) => v.n.id === n.id).length])], F.node);

function layoutState() {
  const rows = [...V].sort(byRank).map((v) => row(v, 'state')).join('');
  return `<div class="tl tl-state" role="table" aria-label="Torrents" data-grp>${phone ? '' : head('state')}${rows}${empty()}</div>`;
}

// ───────────────────────────── layout C · By tracker

function trackerStatus(tr) {
  if (tr.status === 'not-allowed') return `<span class="chip-warn">${I('shield-x')}Not allowed</span>`;
  if (tr.status === 'degraded') return `<span class="chip-warn">${I('radio-tower')}Timing out</span>`;
  return `<span class="tk-ok">${dot('idle')}Working</span>`;
}

function layoutTracker() {
  return D.trackers.map((tr) => {
    const r = client(tr.route), rn = node(r.node), s = tr.stats;
    const vs = V.filter((v) => v.tr.id === tr.id).sort(byRank);
    const routeTxt = `${r.kind === 'built-in' ? 'Built-in' : esc(r.name)} on ${esc(rn.name)}`;
    const stats = `<dl class="tk-st num">
      <div><dt>Uploaded</dt><dd>${s.uploadedTB} TB</dd></div><div><dt>Downloaded</dt><dd>${s.downloadedTB} TB</dd></div>
      <div><dt>Ratio</dt><dd>${r2(s.uploadedTB / s.downloadedTB)}</dd></div>
      <div><dt>Inside H&amp;R window</dt><dd class="${s.hnrOpen ? '' : 't3'}">${tr.rule && tr.rule.minHours ? s.hnrOpen : '—'}</dd></div>
      <div><dt>Grabs go to</dt><dd class="tk-route"><button class="lnk" data-go="clients">${routeTxt}</button></dd></div></dl>`;
    const strip = tr.status !== 'ok' ? nw(`<p class="tk-strip">${esc(tr.message)}${tr.status === 'not-allowed' ? `. Grabs go to ${routeTxt} until ${esc(tr.name)} adds -SS2000-.` : '. Torrents keep seeding; retrying every 30 min.'}</p>`) : '';
    const rule = tr.rule ? `${I('scale')}<span>${esc(tr.rule.note)}</span>` : `${I('globe')}<span>No tracker rules. Seeding goal only.</span>`;
    const table = `<div class="tl tl-tracker tl-flat" role="table" aria-label="Torrents from ${esc(tr.name)}" data-grp>${phone ? '' : head('tracker')}${vs.map((v) => row(v, 'tracker')).join('')}${empty()}</div>`;
    const more = s.torrents - vs.length;
    const moreBtn = more > 0 ? `<div class="tk-more"><button class="btn ghost sm" data-toast="Paging through every torrent is not in this prototype">${I('chevron-down')}${more} more from ${esc(tr.name)}</button></div>` : '';
    return `<section class="tk${tr.status !== 'ok' ? ' tk-warn' : ''}" aria-labelledby="tk-${tr.id}">
      <header class="tk-h">
        <div class="tk-id"><h3 id="tk-${tr.id}">${esc(tr.name)}</h3><span class="chip node">${tr.private ? `${I('lock')}Private` : `${I('globe')}Public`}</span><span class="t3">${esc(tr.software)}</span>${trackerStatus(tr)}</div>
        <p class="tk-rule t2">${rule}</p>
        ${stats}
      </header>${strip}${table}${moreBtn}</section>`;
  }).join('');
}

// ───────────────────────────── filters

const F = { ct: '', q: '', bucket: 'all', node: '' };

function apply() {
  root.querySelectorAll('.tr[data-t]').forEach((r) => {
    const ok = (!F.ct || r.dataset.ct === F.ct) && (!F.q || r.dataset.q.includes(F.q)) && (F.bucket === 'all' || r.dataset.b.split(' ').includes(F.bucket)) && (!F.node || r.dataset.node === F.node);
    r.hidden = !ok;
  });
  root.querySelectorAll('[data-grp]').forEach((g) => {
    const any = [...g.querySelectorAll('.tr[data-t]')].some((r) => !r.hidden);
    const e = g.querySelector(':scope > .tempty');
    if (e) e.hidden = any;
  });
}

function toolbar(layout) {
  const total = D.clients.reduce((n, c) => n + c.totals.torrents, 0);
  const ctSeg = segs('ct', 'Content Type', [['', 'All'], ...Object.entries(D.contentTypes).map(([k, d]) => [k, d.plural])], F.ct);
  const search = `<label class="tf">${I('search')}<input type="search" class="tf-in" placeholder="Filter by Work, release or tracker" aria-label="Filter torrents" autocomplete="off" spellcheck="false"></label>`;
  const title = { node: 'By Node', state: 'All torrents', tracker: 'By tracker' }[layout];
  if (phone) return `<div class="tbar-ph">${search}${ctSeg}${layout === 'state' ? `${bucketSeg()}${nodeSeg()}` : ''}</div>`;
  return `<div class="tbar">${layout === 'state' ? `${bucketSeg()}${nodeSeg()}` : `<h2 id="tl-h">${title}</h2><span class="t3 num">${total} torrents · ${D.downloadingNodes().length} Nodes · ${D.clients.length} Download Clients</span>`}<span class="grow"></span>${search}${ctSeg}</div>`;
}

// ───────────────────────────── the inspector: one torrent in full

function stateChip(v) {
  const t = v.t, tone = stateTone(v);
  if (tone === 'fail') return `<span class="chip-fail">${I('circle-x')}${esc(stateLabel(v))}</span>`;
  if (tone === 'warn' && t.state === 'stalled') return `<span class="chip-warn">${I('triangle-alert')}Stalled</span>`;
  if (t.state === 'downloading') return `<span class="pill-live">${dot('draft')}Downloading</span>`;
  if (tone === 'met') return `<span class="pill-met">${I('check')}Goal met</span>`;
  if (v.gp.kind === 'held') return `<span class="chip ins-chip">${I('lock')}Held by ${esc(v.tr.name)} rule</span>`;
  return `<span class="chip ins-chip">${dot('idle')}${esc(stateLabel(v))}</span>`;
}

function callouts(v) {
  const t = v.t, out = [];
  const card = (tone, chip, icon, body, act = '') => `<div class="ins-pr ins-${tone}"><div class="ins-pr-h"><span class="chip-${tone}">${I(icon)}${esc(chip)}</span></div>${body}${act}</div>`;
  if (t.state === 'error') out.push(card('fail', 'Unregistered', 'circle-x', `<p>${esc(v.tr.name)}: ${esc(t.announce.msg)}.</p><p class="ins-dim">${esc(t.note)}</p>`, `<div class="ins-pr-a"><button class="btn fail sm" data-rm="${t.id}">${I('trash-2')}Remove torrent</button></div>`));
  if (t.stale) out.push(card('fail', 'Client unreachable', 'unplug', `<p>${esc(v.c.name)} on ${esc(v.n.name)} has not answered since ${hm(v.c.lastSeen)}.</p><p class="ins-dim">Every number here is its last report. ${esc(v.c.error)}.</p>`, `<div class="ins-pr-a"><button class="btn fail sm" data-toast="Testing a Download Client is not in this prototype">${I('plug-zap')}Test connection</button></div>`));
  if (v.hn && v.hn.atRisk) out.push(card('warn', 'Hit and run risk', 'timer', `<div class="hnr"><p><b class="num">${dur(v.hn.needH)}</b><span>more seeding needed</span></p><p><b class="num">${dur(v.hn.leftInWindowH)}</b><span>left in the window</span></p></div><p class="ins-dim">Keep it seeding. Pausing or removing it before then counts as a hit and run on ${esc(v.tr.name)}.${t.state === 'stalled' ? ' No peers right now, but seeding time still counts.' : ''}</p>`));
  if (t.state !== 'error' && t.announce.status === 'error') out.push(card('warn', 'Announces timing out', 'radio-tower', `<p>${esc(v.tr.name)} has not answered since ${hm(v.tr.since)}. ${esc(t.announce.msg)}.</p><p class="ins-dim">It keeps ${t.completedAt ? 'seeding' : 'downloading from peers it knows'}, but ${esc(v.tr.name)} may not credit this time until it answers.</p>`, `<div class="ins-pr-a"><button class="btn warn sm" data-toast="Force reannounce is not in this prototype">${I('radio-tower')}Force reannounce</button></div>`));
  if (t.state === 'downloading' && !t.down && v.c.listen && !v.c.listen.reachable) out.push(card('warn', 'Port closed', 'door-closed', `<p>${esc(t.note)}</p><p class="ins-dim">Forward TCP and UDP ${v.c.listen.port} to ${esc(v.n.name)}, or turn on UPnP.</p>`, `<div class="ins-pr-a"><button class="btn warn sm" data-toast="Testing a port is not in this prototype">Test port</button></div>`));
  if (t.state === 'stalled' && !(v.hn && v.hn.atRisk)) out.push(card('warn', 'No peers', 'triangle-alert', `<p>${esc(t.note)}</p>`));
  if (!out.length && t.note) out.push(`<p class="ins-note t2">${I('info')}<span>${esc(t.note)}</span></p>`);
  return out.join('');
}

function actions(v) {
  const t = v.t, paused = t.state === 'paused';
  return `<div class="ins-act">
    <button class="btn" data-toast="${paused ? 'Resuming' : 'Pausing'} a torrent is not in this prototype">${I(paused ? 'play' : 'pause')}${paused ? 'Resume' : 'Pause'}</button>
    <button class="btn" data-toast="Force reannounce is not in this prototype">${I('radio-tower')}Force reannounce</button>
    <button class="btn" data-toast="Recheck is not in this prototype">${I('refresh-cw')}Recheck</button>
    <button class="btn ins-rm" data-rm="${t.id}">${I('trash-2')}Remove…</button>
  </div>`;
}

function sec(title, inner, extra = '', id = '') {
  if (phone) return phoneSection(title, `<div class="glist ins-card">${inner}</div>`, extra);
  return `<section class="ins-sec"${id ? ` id="${id}"` : ''}><div class="sec-h"><h3>${title}</h3><span class="grow"></span>${extra}</div>${inner}</section>`;
}

function modeText(g) {
  if (g.mode === 'forever') return `Keep seeding until you remove it${g.why ? `. ${g.why}` : ''}`;
  const parts = [g.ratio ? `ratio ${r2(g.ratio)}` : null, g.hours ? `${dur(g.hours)} seeded` : null].filter(Boolean);
  if (parts.length === 1) return `Seed to ${parts[0]}`;
  return g.mode === 'both' ? `Seed to ${parts.join(' and ')}` : `Seed to ${parts.join(' or ')}, whichever comes first`;
}

function goalBlock(v) {
  const t = v.t, g = v.g, tr = v.tr;
  if (v.external) {
    return sec('Seeding', `<div class="ins-pad"><p class="ins-ext">${I('external-link')}<span><b>${esc(v.c.name)} keeps its own seeding rules.</b> Seeding goals belong to the built-in client. SlipStream grabbed to ${esc(v.c.name)} and imports from it.</span></p>
      ${tr.rule ? `<dl class="kv ins-kv"><dt>${esc(tr.name)} rule</dt><dd>${esc(tr.rule.note)}</dd></dl>` : ''}</div>`, `<button class="btn ghost sm" data-go="clients">${esc(v.c.name)} ${I('chevron-right')}</button>`);
  }
  let top;
  if (t.state === 'error') top = `<p class="ins-mode fail-t">Seeding stopped. ${esc(tr.name)} no longer tracks this torrent, so no hit and run counts.</p>`;
  else if (v.gp.kind === 'pending') top = `<p class="ins-mode">${esc(modeText(g))}</p><p class="t3">Starts when the download finishes.</p>${goalMeter(v, true)}`;
  else top = `<p class="ins-mode">${esc(modeText(g))}</p>${v.gp.kind === 'checking' ? '<p class="t3">Paused while the pieces verify.</p>' : ''}${goalMeter(v, true)}`;
  const win = v.hn ? (() => { const end = new Date(new Date(t.completedAt).getTime() + tr.rule.windowDays * 864e5).toISOString(); return `<dt>Window</dt><dd class="${v.hn.atRisk ? 'warn-t' : ''}">Closes ${when(end)} · ${dur(v.hn.leftInWindowH)} left, ${dur(v.hn.needH)} to seed</dd>`; })() : '';
  const met = t.state === 'met' ? `<dt>Met</dt><dd>${t.removeAt ? `Removing at ${hm(t.removeAt)}, in ${minsUntil(t.removeAt)} min` : 'Yes'}</dd>` : '';
  return sec('Seeding goal', `<div class="ins-pad">${top}
    <dl class="kv ins-kv">
      ${tr.rule ? `<dt>${esc(tr.name)} rule</dt><dd>${esc(tr.rule.note)}. No goal can go under it.</dd>` : `<dt>Rule</dt><dd>${esc(tr.name)} have no rule of their own.</dd>`}
      ${win}${met}
      ${t.state === 'error' ? '' : `<dt>When met</dt><dd>${esc(D.whenMetLabel[g.whenMet])}. Library files stay.</dd>`}
    </dl></div>`, `<button class="btn ghost sm" data-go="goals">${esc(g.source)} ${I('chevron-right')}</button>`);
}

function transferBlock(v) {
  const j = job(v.t.transfer);
  if (!j) return '';
  const [from, to] = j.node.split(' → ');
  return sec('Transfer', `<div class="ins-pad xf">
    <p class="xf-h"><span class="mono">${esc(from)}</span>${I('arrow-right')}<span class="mono">${esc(to)}</span><span class="t3">for import</span><span class="grow"></span><span class="num xf-p" data-live-pct="${j.id}">${C.format.pct(j.progress)}</span></p>
    <div class="bar" role="progressbar" aria-valuemin="0" aria-valuemax="100" aria-valuenow="${Math.floor(j.progress * 100)}" aria-label="Transfer to ${esc(to)}" style="--p:${j.progress}" data-live-progress="${j.id}"><span class="fill"></span></div>
    <p class="num t2"><span data-live-rate="${j.id}">${C.format.rate(j)}</span>${sep}<span data-live-eta="${j.id}">${C.format.eta(j.etaMin)}</span> left${sep}${esc(j.size)}${sep}${esc(j.release)}</p>
    <p class="t3">${esc(from)} keeps seeding its copy. ${esc(to)} imports when the Transfer finishes.</p></div>`, `<button class="btn ghost sm" data-go="home">Open job ${I('chevron-right')}</button>`);
}

const announceUrl = (tr) => (tr.private ? `https://${tr.id}.example/announce/•••••` : 'udp://open.example:1337/announce');

function trackersBlock(v) {
  const t = v.t, a = t.announce, tr = v.tr;
  const status = a.status === 'error' ? `<span class="${t.state === 'error' ? 'fail-t' : 'warn-t'}">${dot(t.state === 'error' ? 'fail' : 'warn')}${t.state === 'error' ? 'Error' : 'Timing out'}</span>` : a.status === 'updating' ? `<span class="t2">${dot('idle')}Updating</span>` : `<span class="t1">${dot('idle')}Working</span>`;
  const next = a.next ? `${hm(a.next)} · in ${minsUntil(a.next)} min` : t.state === 'met' ? 'None, removal pending' : t.stale ? 'Unknown' : 'None';
  return sec('Trackers', `<div class="ins-pad trk-b">
    <p class="trk-n"><b>${esc(tr.name)}</b><span class="chip node">${tr.private ? 'Private' : 'Public'}</span><span class="grow"></span>${status}</p>
    <p class="mono trk-u">${esc(announceUrl(tr))}</p>
    <dl class="kv ins-kv">
      ${a.msg ? `<dt>Message</dt><dd class="${t.state === 'error' ? 'fail-t' : 'warn-t'}">${esc(a.msg)}</dd>` : ''}
      <dt>Last</dt><dd class="num">${a.last ? `${hm(a.last)} · ${ago(a.last)}` : 'Not yet'}</dd>
      <dt>Next</dt><dd class="num">${next}</dd>
      <dt>Swarm</dt><dd class="num">${t.seeds} seeds · ${t.leech} leechers</dd>
    </dl>
    ${tr.private ? '' : '<p class="t3 trk-x">DHT and PEX on. Private trackers turn both off.</p>'}</div>`, `<span class="t3 num">1 tracker</span>`);
}

function fileList(v) {
  const t = v.t, ct = v.ct, items = t.coverage.items, lvl = ct.levels[0];
  const wts = items.map((_, i) => 0.9 + ((i * 37) % 20) / 100), sum = wts.reduce((a, b) => a + b, 0);
  return items.map((it, i) => {
    const label = fill(ct.item.label, it);
    const base = items.length === 1 ? t.release : lvl ? t.release.replace(new RegExp(`${fill(lvl.short, it)}[^.]*`), label) : `${t.release}.${label}`;
    return { name: `${base}.mkv`, item: label, gb: (t.sizeGB * wts[i]) / sum };
  });
}

function filesBlock(v) {
  const fs = fileList(v), t = v.t, { j, p } = liveOrStatic(t);
  const levelName = v.ct.levels[0] && v.t.coverage.whole ? fill(v.ct.levels[0].label, v.t.coverage.items[0]) : null;
  const rows = fs.map((f) => `<li class="fl">
    <span class="fl-i">${I('file-video')}</span>
    <div class="fl-m"><p class="mono fl-n">${esc(f.name)}</p>${isDl(v) ? `<div class="bar bar-thin${t.state === 'downloading' ? '' : ' bar-q'}" role="progressbar" aria-valuemin="0" aria-valuemax="100" aria-valuenow="${Math.floor(p * 100)}" aria-label="${esc(f.item)} file progress" style="--p:${p}" ${j ? `data-live-progress="${j.id}"` : ''}><span class="fill"></span></div>` : ''}</div>
    <span class="chip fl-it">${esc(f.item)}</span>
    <span class="num fl-s">${gb(f.gb)}</span></li>`).join('');
  return sec('Files', `<ul class="fls">${rows}</ul>`, `<span class="t3 num">${fs.length} ${fs.length === 1 ? 'file' : 'files'} · ${gb(t.sizeGB)}${levelName ? ` · ${esc(levelName)}` : ''}</span>`);
}

function hash(s) {
  let a = 2166136261, out = '';
  for (let k = 0; k < 5; k++) { for (const ch of s + k) a = Math.imul(a ^ ch.charCodeAt(0), 16777619) >>> 0; out += a.toString(16).padStart(8, '0'); }
  return out;
}

function numbersBlock(v) {
  const t = v.t;
  const kv = [
    ['Size', gb(t.sizeGB)], ['Downloaded', gb(t.downloadedGB)], ['Uploaded', gb(t.uploadedGB)], ['Ratio', r2(t.ratio)],
    ['Seeded', t.seededH ? dur(t.seededH) : '—'], ['Peers', String(t.peers)],
    ['Added', when(t.addedAt)], ['Completed', t.completedAt ? when(t.completedAt) : '—'],
  ];
  return sec('Numbers', `<dl class="nums">${kv.map(([k, x]) => `<div><dt>${k}</dt><dd class="num">${x}</dd></div>`).join('')}</dl>
    <dl class="kv ins-kv ins-pad ins-kv2"><dt>Info hash</dt><dd class="mono ell">${hash(t.id)}</dd><dt>Saved to</dt><dd class="mono ins-path">${esc(v.n.disk.path)}/${esc(t.release)}</dd></dl>`, t.stale ? `<span class="t3 num">as of ${hm(v.c.lastSeen)}</span>` : '');
}

function detail(v) {
  const t = v.t;
  return `<header class="ins-h">
      ${art(v.w, phone ? 'poster-sm' : 'thumb-ins')}
      <div class="ins-id">
        <p class="t3 ins-k"><span class="mono">${esc(v.n.name)}</span> · ${esc(v.c.name)}${v.external ? ' <span class="chip">External</span>' : ''}</p>
        <h2 id="ins-title">${esc(v.w.title)}</h2>
        <p class="t2">${esc(cov(t))} · ${esc(t.target)}</p>
        <p class="ins-st">${stateChip(v)}${t.up || t.down ? `<span class="num t2">↓ ${rate(t.down)} ↑ ${rate(t.up)} MB/s</span>` : ''}</p>
      </div>
    </header>
    <p class="mono ins-rel">${esc(t.release)}</p>
    ${isDl(v) ? `<div class="ins-dl">${dlBar(v, true)}</div>` : ''}
    ${callouts(v)}
    ${actions(v)}
    ${goalBlock(v)}
    ${transferBlock(v)}
    ${trackersBlock(v)}
    ${filesBlock(v)}
    ${numbersBlock(v)}`;
}

// ───────────────────────────── remove: library files stay; warn when the tracker's rule is not met

function confirmRemove(id, mode, from) {
  const v = V.find((x) => x.t.id === id), t = v.t, tr = v.tr;
  const atRisk = tr.private && t.completedAt && t.state !== 'error' && !D.ruleMet(t, tr.rule);
  const ruleProg = tr.rule ? [tr.rule.minRatio != null ? `ratio ${r2(t.ratio)} of ${r2(tr.rule.minRatio)}` : null, tr.rule.minHours != null ? `${dur(t.seededH)} of ${dur(tr.rule.minHours)} seeded` : null].filter(Boolean).join(', or ') : '';
  const warn = atRisk ? `<div class="rmd-warn"><span class="chip-warn">${I('triangle-alert')}Counts as a hit and run</span><p>${esc(tr.name)}'s rule is not met yet: ${ruleProg}.${v.hn ? ` It needs ${dur(v.hn.needH)} more within ${dur(v.hn.leftInWindowH)}.` : ''} Removing now counts as a hit and run on your ${esc(tr.name)} account.</p></div>` : '';
  const ext = v.external ? `${v.c.name} removes it${t.stale ? ` when it answers again; it is unreachable now` : ''}.` : '';
  const partial = !t.completedAt;
  const opts = [
    ['torrent', 'Remove torrent', partial ? `Stops the download and keeps the ${gb(t.downloadedGB)} downloaded so far` : `Keeps the download copy in ${v.n.disk.path}`],
    ['data', 'Remove torrent and download copy', `Frees ${gb(t.downloadedGB)} on ${v.n.name}`],
  ];
  const go = (m) => (m === 'data' ? 'Remove torrent and copy' : 'Remove torrent');
  openSheet(`<div class="rmd" role="dialog" aria-modal="true" aria-labelledby="rm-h" aria-describedby="rm-d">
    <div class="rmd-h">${art(v.w, 'thumb-md')}<div><h2 id="rm-h">Remove this torrent?</h2><p class="t2">${esc(v.w.title)} · ${esc(cov(t))} · ${esc(t.target)}</p><p class="t3">${esc(v.n.name)} · ${esc(v.c.name)} · ${esc(tr.name)}</p></div></div>
    ${warn}
    <p class="rmd-lib" id="rm-d">${I('library')}<span><b>Library files stay.</b> Import hard-linked or copied them, so removing a torrent never touches the library.${ext ? ` ${esc(ext)}` : ''}</span></p>
    <div class="rmd-opts" role="radiogroup" aria-label="What to remove">${opts.map(([k, l, s]) => `<button role="radio" class="opt" data-rmopt="${k}" aria-checked="${k === mode}"><span class="opt-r" aria-hidden="true"></span><span class="opt-t"><b>${l}</b><span class="t3">${esc(s)}</span></span></button>`).join('')}</div>
    <div class="rmd-act"><button class="btn" data-close>Cancel</button><button class="btn fail" data-rm-go>${I('trash-2')}<span>${go(mode)}</span></button></div>
  </div>`, from);
  root.querySelector('.rmd').dataset.go = JSON.stringify({ torrent: go('torrent'), data: go('data') });
}

// ───────────────────────────── inspector wiring (desk)

let selId = null;
function setUrl(screen, t) {
  const p = new URLSearchParams(location.search);
  p.set('screen', screen);
  if (t) p.set('t', t);
  history.replaceState(null, '', `?${p}`);
}

function showInspector(id) {
  const v = V.find((x) => x.t.id === id);
  if (!v) return;
  selId = id;
  const ins = root.querySelector('.insp');
  ins.querySelector('.insp-sc').innerHTML = detail(v);
  ins.hidden = false;
  ins.querySelector('.insp-sc').scrollTop = 0;
  root.classList.add('insp-open');
  root.querySelectorAll('.tr.sel').forEach((r) => { r.classList.remove('sel'); r.removeAttribute('aria-selected'); });
  root.querySelectorAll(`.tr[data-t="${id}"]`).forEach((r) => { r.classList.add('sel'); r.setAttribute('aria-selected', 'true'); });
  C.icons();
  setLead(v.w.id);
  setUrl('torrent', id);
}

function closeInspector() {
  if (!selId) return;
  const back = root.querySelector(`.tr[data-t="${selId}"]`);
  selId = null;
  root.querySelector('.insp').hidden = true;
  root.classList.remove('insp-open');
  root.querySelectorAll('.tr.sel').forEach((r) => { r.classList.remove('sel'); r.removeAttribute('aria-selected'); });
  setLead(LEAD);
  setUrl('torrents');
  if (back && !back.hidden) back.focus({ preventScroll: true });
}

function openT(id) {
  if (phone) return C.go('torrent', { t: id });
  showInspector(id);
}

// ───────────────────────────── screen

let PROBS, LEAD;

function sentence() {
  const tt = D.clients.reduce((a, c) => ({ d: a.d + c.totals.downloading, s: a.s + c.totals.seeding }), { d: 0, s: 0 });
  const f = PROBS.filter((p) => p.tone === 'fail').length, w = PROBS.length - f;
  return `<span class="s-live">${dot('draft')}${tt.d} downloading</span>${sep}<span>${tt.s} seeding</span>${f ? `${sep}<a class="s-fail" href="#problems">${f} failures</a>` : ''}${w ? `${sep}<a class="s-warn" href="#problems">${w} warnings</a>` : ''}`;
}

function totalsLine() {
  const ok = D.clients.filter((c) => c.status === 'ok');
  const d = ok.reduce((s, c) => s + c.totals.downRate, 0), u = ok.reduce((s, c) => s + c.totals.upRate, 0), up = D.clients.reduce((s, c) => s + c.totals.uploadedTB, 0);
  return `${I('arrow-down')}<b class="num">${rate(d)}</b> MB/s${sep}${I('arrow-up')}<b class="num">${rate(u)}</b> MB/s${sep}<b class="num">${up.toFixed(1)} TB</b> uploaded`;
}

function listBody(layout) {
  return { node: layoutNode, state: layoutState, tracker: layoutTracker }[layout]();
}

function bodyWide(layout) {
  return `<h1 class="sr">Torrents</h1>
    <section class="pulse" aria-label="Torrent status"><p class="sentence">${sentence()}</p><p class="pulse-r t3 tot">${totalsLine()}</p></section>
    ${problemsWide(PROBS)}
    <section class="tsec tsec-${layout}" aria-label="Torrents">${toolbar(layout)}<div class="tbody">${listBody(layout)}</div></section>
    <aside class="insp" aria-labelledby="ins-title" hidden><div class="insp-bar"><span class="t3">Torrent</span><span class="grow"></span><kbd>Esc</kbd><button class="ib" data-ins-close aria-label="Close torrent details">${I('x')}</button></div><div class="insp-sc"></div></aside>`;
}

function bodyPhone(layout) {
  return `<header class="ph-head"><h1>Activity</h1><span class="grow"></span><button class="ib tap" data-open="palette" aria-label="Search">${I('search')}</button><span class="avatar">${D.admin.initials}</span></header>
    <div class="ph-seg"><div class="seg-c" role="radiogroup" aria-label="Activity view"><button role="radio" class="seg-o" aria-checked="false" data-go="home">Jobs</button><button role="radio" class="seg-o" aria-checked="true">Torrents</button></div></div>
    <p class="ph-sentence">${sentence()}</p>
    <p class="ph-tot t3">${totalsLine()}</p>
    ${problemsPhone(PROBS)}
    <section class="psec tsec-ph" aria-label="Torrents"><div class="psec-h"><h2>${{ node: 'By Node', state: 'Torrents', tracker: 'By tracker' }[layout]}</h2><span class="t3 num">${D.clients.reduce((n, c) => n + c.totals.torrents, 0)}</span></div>${toolbar(layout)}${layout === 'state' ? layoutState() : ''}</section>
    ${layout === 'state' ? '' : listBody(layout)}`;
}

function bodyPhoneDetail(v) {
  return `<header class="ph-nav"><button class="back tap" data-go="torrents">${I('chevron-left')}Torrents</button><span class="grow"></span><button class="ib tap" data-menu="${v.t.id}" aria-haspopup="dialog" aria-label="More actions for ${esc(v.w.title)}">${I('ellipsis')}</button></header>
    <div class="pd">${detail(v)}</div>`;
}

function menuFor(key) {
  const v = V.find((x) => x.t.id === key), t = v.t, paused = t.state === 'paused';
  return {
    title: `${v.w.title} · ${cov(t)}`, sub: `${stateLabel(v)} on ${v.n.name} · ${v.c.name}`,
    items: [
      [paused ? 'Resume' : 'Pause', paused ? 'play' : 'pause'],
      ['Force reannounce', 'radio-tower'],
      ['Recheck', 'refresh-cw'],
      ['Open Work', 'arrow-up-right', 'go', 'work'],
      ['Remove torrent', 'trash-2', 'act-danger', `torrent:${t.id}`],
      ['Remove torrent and download copy', 'trash-2', 'act-danger', `data:${t.id}`],
    ],
  };
}

export function render(ctx) {
  C = ctx; D = ctx.data; root = ctx.root; phone = ctx.device === 'phone';
  V = D.torrents.map(view);
  PROBS = problems();
  const layout = ctx.layout;
  const cur = V.find((v) => v.t.id === ctx.torrent) || V.find((v) => v.t.id === 't-hotd-s2');
  const detailScreen = ctx.screen === 'torrent';
  LEAD = (V.find((v) => v.t.state === 'downloading') || V[0]).w.id;
  const phoneDetail = phone && detailScreen;
  root.classList.add(`lc-tl-${layout}`);

  mount(ctx, {
    id: phoneDetail ? 'torrent' : 'torrents',
    title: detailScreen ? `${cur.w.title} · Torrents` : 'Torrents',
    nav: 'torrents', tab: 'activity',
    crumbs: '<span class="crumb cur">Torrents</span>',
    body: phoneDetail ? bodyPhoneDetail(cur) : phone ? bodyPhone(layout) : bodyWide(layout),
    lead: phoneDetail ? cur.w.id : LEAD,
    actions: menuFor,
    act(key, el) { const [mode, id] = key.split(':'); confirmRemove(id, mode, el); },
    click(e, q) {
      let el;
      if ((el = q('[data-seg]'))) {
        const g = el.closest('[data-segs]');
        g.querySelectorAll('.seg-o').forEach((o) => o.setAttribute('aria-checked', String(o === el)));
        F[g.dataset.segs] = el.dataset.seg;
        apply();
        return true;
      }
      if ((el = q('[data-rmopt]'))) {
        const d = el.closest('.rmd');
        d.querySelectorAll('[data-rmopt]').forEach((o) => o.setAttribute('aria-checked', String(o === el)));
        d.querySelector('[data-rm-go] span').textContent = JSON.parse(d.dataset.go)[el.dataset.rmopt];
        return true;
      }
      if ((el = q('[data-rm-go]'))) { closeAll(); toast('Removing a torrent is not in this prototype'); return true; }
      if ((el = q('[data-rm]'))) { confirmRemove(el.dataset.rm, 'torrent', el); return true; }
      if ((el = q('[data-pr]'))) { problemSheet(el.dataset.pr, el); return true; }
      if ((el = q('[data-open-t]'))) { closeAll(); openT(el.dataset.openT); return true; }
      if ((el = q('[data-ins-close]'))) { closeInspector(); return true; }
      if ((el = q('.tr[data-t]')) && !q('button, a, input')) { openT(el.dataset.t); return true; }
      return false;
    },
    after(r) {
      const inp = r.querySelector('.tf-in');
      if (inp) inp.addEventListener('input', () => { F.q = inp.value.trim().toLowerCase(); apply(); });
      r.addEventListener('keydown', (e) => {
        const tr = e.target.closest && e.target.closest('.tr[data-t]');
        if (!tr || e.target !== tr) return;
        if (e.key === 'Enter' || e.key === ' ') { e.preventDefault(); openT(tr.dataset.t); }
        if (e.key === 'ArrowDown' || e.key === 'ArrowUp') {
          e.preventDefault();
          const all = [...r.querySelectorAll('.tr[data-t]')].filter((x) => !x.hidden);
          const next = all[all.indexOf(tr) + (e.key === 'ArrowDown' ? 1 : -1)];
          if (next) { next.focus(); if (selId) showInspector(next.dataset.t); }
        }
      });
      document.addEventListener('keydown', (e) => { if (e.key === 'Escape' && !e.defaultPrevented && selId) { e.preventDefault(); closeInspector(); } });
      if (!phone && detailScreen) showInspector(cur.t.id);
    },
  });
}
