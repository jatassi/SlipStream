// PROTOTYPE · Tracker detail page, three variants on Direction C. Row rendering, goal meter and goal editor are copied from the torrent client prototype. Throwaway.
import { mount, h, openSheet, closeAll, toast } from '../torrent-client/shell.js';

let C, D, phone, root, TR, V, variant;
const { esc, I, art, dot, hm, md } = h;

const MARK = `<svg class="mark" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" aria-hidden="true"><path d="M2.5 6.5c6 0 9.5 0 12.5 3.2"/><path d="M2.5 12h9.5"/><path d="M2.5 17.5c6 0 9.5 0 12.5-3.2"/><circle cx="18.6" cy="12" r="2.6" fill="currentColor" stroke="none"/></svg>`;
const W = (id) => D.works[id];
const client = (id) => D.clients.find((c) => c.id === id);
const node = (id) => D.nodes.find((n) => n.id === id);
const tracker = (id) => D.trackers.find((t) => t.id === id);
const job = (id) => D.jobs.find((j) => j.id === id);

const r1 = (x) => Number(x).toFixed(1);
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
const hrs = (x) => (x % 24 === 0 && x >= 168 ? `${x / 24} days` : `${+x.toFixed(1)} h`);
const gh = (x) => (x % 24 === 0 && x >= 24 ? (x === 24 ? '1 day' : `${x / 24} days`) : `${+x.toFixed(1)} h`);
const when = (iso) => `${md(iso)}, ${hm(iso)}`;
const minsUntil = (iso) => Math.max(0, Math.round((new Date(iso) - D.now) / 60000));
const sep = '<span class="s-sep" aria-hidden="true">·</span>';
const cov = (t) => D.coverageLabel(t.work, t.coverage);
const nw = (html) => html.replace(/(-SS\d{4}-)/g, '<span class="nw">$1</span>');
const plural = (n, one, many) => `${n} ${n === 1 ? one : many}`;
const ago = (iso) => {
  const m = Math.round((D.now - new Date(iso)) / 60000);
  if (m < 60) return `${m} min ago`;
  const hr = Math.floor(m / 60);
  return hr < 24 ? `${hr} h ago` : `${Math.floor(hr / 24)} d ago`;
};
const fmtN = (n) => n.toLocaleString('en-US');

// ───────────────────────────── torrent view model (copied from torrents.js)

const STATE = { downloading: 'Downloading', queued: 'Queued', checking: 'Checking', seeding: 'Seeding', stalled: 'Stalled', paused: 'Paused', error: 'Tracker error', met: 'Goal met' };

function view(t) {
  const w = W(t.work), c = client(t.client), n = node(c.node), tr = tracker(t.tracker), ct = D.contentTypes[w.type];
  const g = D.goalFor(t), gp = D.goalProgress(t), hn = D.hnr(t);
  const external = c.kind !== 'built-in';
  const reasons = [];
  if (t.state === 'error') reasons.push({ tone: 'fail', text: `Unregistered by ${tr.name}` });
  if (t.stale) reasons.push({ tone: 'fail', text: `Last known at ${hm(c.lastSeen)}` });
  if (hn && hn.atRisk) reasons.push({ tone: 'warn', text: `H&R in ${dur(hn.leftInWindowH)}, needs ${dur(hn.needH)}` });
  if (t.state !== 'error' && t.announce.status === 'error') reasons.push({ tone: 'warn', text: 'Announces timing out' });
  if (t.state === 'stalled') reasons.push({ tone: 'warn', text: 'No peers to seed to' });
  if (t.state === 'downloading' && !t.down && c.listen && !c.listen.reachable) reasons.push({ tone: 'warn', text: `Port ${c.listen.port} closed` });
  const buckets = [];
  const done = !!t.completedAt;
  if (['downloading', 'queued'].includes(t.state) || (t.state === 'checking' && !done)) buckets.push('downloading');
  if (['seeding', 'stalled', 'paused'].includes(t.state) || (t.state === 'checking' && done)) buckets.push(gp.kind === 'met' ? 'met' : 'seeding');
  if (t.state === 'met') buckets.push('met');
  if (reasons.length) buckets.push('problems');
  const rank = reasons.some((r) => r.tone === 'fail') ? 0 : reasons.length ? 1 : buckets.includes('downloading') ? 2 : buckets.includes('met') ? 4 : 3;
  return { t, w, c, n, tr, ct, g, gp, hn, external, reasons, buckets, rank };
}
const byRank = (a, b) => a.rank - b.rank;
const isDl = (v) => ['downloading', 'queued'].includes(v.t.state) || (v.t.state === 'checking' && !v.t.completedAt);

function stateLabel(v) { return v.gp.kind === 'held' ? 'Held by rule' : STATE[v.t.state]; }
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

function liveOrStatic(t) {
  const j = t.job && t.state === 'downloading' ? job(t.job) : null;
  return { j, p: j ? j.progress : t.progress };
}
function dlBar(v) {
  const t = v.t, { j, p } = liveOrStatic(t);
  const glow = t.state === 'downloading';
  const pctTxt = C.format.pct(p);
  const pct = j ? `<span data-live-pct="${j.id}">${pctTxt}</span>` : pctTxt;
  let tail;
  if (j) tail = `<span data-live-eta="${j.id}">${C.format.eta(j.etaMin)}</span> left${sep}<span data-live-rate="${j.id}">${C.format.rate(j)}</span>`;
  else if (t.state === 'queued') tail = `${gb(t.sizeGB)} waiting`;
  else if (t.state === 'checking') tail = 'verifying pieces';
  else tail = t.down ? `${C.format.eta(((t.sizeGB - t.downloadedGB) * 1024) / t.down / 60)} left${sep}${rate(t.down)} MB/s` : '<span class="warn-t">no peers sending</span>';
  return `<div class="pg"><div class="bar${glow ? '' : ' bar-q'}" role="progressbar" aria-valuemin="0" aria-valuemax="100" aria-valuenow="${Math.floor(p * 100)}" aria-label="${esc(v.w.title)} download progress" style="--p:${p}" ${j ? `data-live-progress="${j.id}"` : ''}><span class="fill"></span></div><p class="pg-t num"><b>${pct}</b>${sep}${tail}</p></div>`;
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
function goalMeter(v) {
  const ls = lanes(v), k = v.gp.kind, risk = v.hn && v.hn.atRisk;
  const cls = k === 'met' ? ' gm-met' : k === 'held' ? ' gm-held' : '';
  const lane = (l) => {
    const tick = l.f != null ? `<span class="gm-k${l.floorMet ? ' k-ok' : risk ? ' risk' : ''}" style="left:${(l.f * 100).toFixed(1)}%"></span>` : '';
    return `<span class="gm-l"><span class="gm-f" style="width:${(l.p * 100).toFixed(1)}%"></span>${tick}</span>`;
  };
  const p = v.g.mode === 'forever' ? Math.max(0, ...ls.map((l) => l.p)) : v.gp.p ?? 0;
  return `<div class="pg"><div class="gm${cls}" role="progressbar" aria-valuemin="0" aria-valuemax="100" aria-valuenow="${Math.round(p * 100)}" aria-label="Seeding goal, ${esc(goalText(v))}">${ls.map(lane).join('')}</div><p class="pg-t num">${k === 'met' ? `<b>Met</b>${sep}` : k === 'held' ? `<b class="warn-t">Held</b>${sep}` : ''}${esc(goalText(v))}</p></div>`;
}
function progCell(v) {
  const t = v.t;
  if (isDl(v)) return dlBar(v);
  if (t.state === 'error') return `<p class="pg-t pg-x"><span class="fail-t">Seeding stopped</span>${sep}no hit and run counts</p>`;
  if (v.external) return `<p class="pg-t pg-x">${I('external-link')}<span>Seeding rules live in ${esc(v.c.name)}</span></p>`;
  return goalMeter(v);
}
function bigCell(v) {
  const t = v.t;
  if (isDl(v)) { const { j, p } = liveOrStatic(t); return j ? `<b data-live-pct="${j.id}">${C.format.pct(p)}</b>` : `<b>${C.format.pct(p)}</b>`; }
  return `<b>${r2(t.ratio)}</b><span class="t3">ratio</span>`;
}

const COLS = ['work', 'state', 'prog', 'ratio', 'rate', 'where', 'menu'];
const HEAD = { work: 'Work', state: 'State', prog: 'Progress or seeding goal', ratio: 'Ratio', rate: 'MB/s', where: 'Node · client', menu: '<span class="sr">Actions</span>' };

function row(v) {
  const t = v.t;
  const cell = {
    work: `<div class="c-work" role="cell">${art(v.w, 'thumb-tr')}<span class="tw"><b class="ell">${esc(v.w.title)}</b><span class="t3 ell">${esc(cov(t))} · ${esc(t.target)}</span></span></div>`,
    state: `<div class="c-state" role="cell"><p class="tst">${dot(stateTone(v))}<b>${stateLabel(v)}</b></p><p class="st-sub t3 ell">${stateSub(v)}</p><p class="nm t3 ell">${esc(v.n.name)} · ${esc(v.c.name)}${sep}<span class="num">↓ ${rate(t.down)} ↑ ${rate(t.up)}</span></p></div>`,
    prog: `<div class="c-prog" role="cell">${progCell(v)}</div>`,
    ratio: `<div class="c-ratio num" role="cell">${r2(t.ratio)}</div>`,
    rate: `<div class="c-rate num" role="cell"><span class="${t.down ? '' : 't3'}">${I('arrow-down')}${rate(t.down)}</span><span class="${t.up ? '' : 't3'}">${I('arrow-up')}${rate(t.up)}</span></div>`,
    where: `<div class="c-where" role="cell"><span class="ell">${esc(v.n.name)}<span class="t3"> · ${esc(v.c.name)}</span></span></div>`,
    menu: `<div class="c-menu" role="cell"><button class="ib" data-menu="${t.id}" aria-haspopup="${phone ? 'dialog' : 'menu'}" aria-label="Actions for ${esc(v.w.title)} ${esc(cov(t))}">${I('ellipsis')}</button></div>`,
  };
  const q = `${v.w.title} ${t.release}`.toLowerCase();
  return `<div class="tr${t.stale ? ' stale' : ''}" role="row" tabindex="0" data-t="${t.id}" data-work="${v.w.id}" data-amb data-b="${v.buckets.join(' ')}" data-q="${esc(q)}" aria-label="${esc(`${v.w.title}, ${cov(t)}, ${stateLabel(v)}`)}">
    ${COLS.map((c) => cell[c]).join('')}<div class="c-big num" role="cell">${bigCell(v)}</div></div>`;
}
const head = () => `<div class="tr thead" role="row">${COLS.map((c) => `<div class="c-${c}" role="columnheader">${HEAD[c]}</div>`).join('')}</div>`;

// ───────────────────────────── small controls (copied from settings.js)

function seg(label, opts, cur, attrs, { disabled = false, cls = '' } = {}) {
  return `<div class="seg-c ${cls}" role="radiogroup" aria-label="${esc(label)}"${disabled ? ' aria-disabled="true"' : ''}>${opts.map(([v, l, n]) => `<button type="button" role="radio" class="seg-o" aria-checked="${v === cur}" tabindex="${v === cur ? 0 : -1}" ${disabled ? 'disabled' : ''} ${attrs(v)}><span>${l}</span>${n != null ? `<span class="seg-n num">${n}</span>` : ''}</button>`).join('')}</div>`;
}
const swBtn = (on, attrs, label) => `<button type="button" class="btn tog" role="switch" aria-checked="${on}" ${attrs}><span>${esc(label)}</span><span class="sw" aria-hidden="true"><span></span></span></button>`;
const swBare = (on, attrs, label) => `<button type="button" class="swb" role="switch" aria-checked="${on}" aria-label="${esc(label)}" ${attrs}><span class="sw" aria-hidden="true"><span></span></span></button>`;
const unitInput = (id, value, unit, attrs, ph = 'Unlimited') => `<span class="inp-u"><input class="inp num" id="${id}" type="text" inputmode="decimal" autocomplete="off" value="${value ?? ''}" placeholder="${ph}" ${attrs}><span class="u" aria-hidden="true">${unit}</span></span>`;

// ───────────────────────────── seeding goal state: the same goal Settings › Seeding goals edits

const WHEN_SHORT = { remove: 'Remove', stop: 'Stop seeding', keep: 'Keep seeding' };
const WHEN_THEN = { remove: 'remove the torrent and its download copy', stop: 'stop seeding and keep the torrent', keep: 'keep seeding' };
const MODES = [['either', 'Whichever first'], ['both', 'Both'], ['forever', 'Forever']];
let G, units;
function initGoals() {
  const d = D.seedingGoals.default;
  G = {
    def: { ...d },
    per: {
      beacon: { override: false, ...d },
      cinder: { override: true, ...D.seedingGoals.perTracker.cinder },
      public: { override: true, ...D.seedingGoals.perTracker.public },
      aurora: { override: false, ...d },
    },
  };
  units = { def: 'd', beacon: 'd', cinder: 'd', public: 'd', aurora: 'd' };
}
const effGoal = () => (G.per[TR.id].override ? G.per[TR.id] : G.def);
const goalOf = (key) => (key === 'def' ? G.def : G.per[key]);

function floorText(t) {
  const r = t.rule;
  if (!r) return 'No tracker rule. Public torrents can stop any time.';
  if (!r.minHours && r.minRatio) return `${t.name} requires an account ratio above ${r.minRatio}. It doesn't count hits and runs per torrent.`;
  const parts = [`${hrs(r.minHours)} of seeding`];
  if (r.minRatio) parts.push(`ratio ${r1(r.minRatio)}`);
  return `${t.name} requires ${parts.join(' or ')} within ${r.windowDays} days of finishing, or it counts as a hit and run.`;
}
function goalWords(g) {
  if (g.mode === 'forever') return '<b>Seed until you remove it.</b>';
  const sides = [];
  if (g.ratio != null) sides.push(`<b class="num">ratio ${r1(g.ratio)}</b>`);
  if (g.hours != null) sides.push(`<b class="num">${gh(g.hours)}</b>`);
  const then = `Then ${WHEN_THEN[g.whenMet]}.`;
  if (!sides.length) return `Seed only until the tracker's rule is met. ${then}`;
  if (sides.length === 1) return `Seed until ${sides[0]} and the tracker's rule is met. ${then}`;
  return g.mode === 'both' ? `Seed until ${sides[0]} and ${sides[1]}, and the tracker's rule is met. ${then}` : `Seed until ${sides[0]} or ${sides[1]}, whichever comes first, and the tracker's rule is met. ${then}`;
}
const goalShort = (g) => (g.mode === 'forever' ? 'Seed forever' : `${[g.ratio != null ? `ratio ${r1(g.ratio)}` : '', g.hours != null ? gh(g.hours) : ''].filter(Boolean).join(g.mode === 'both' ? ' and ' : ' or ') || "the tracker's rule"}, then ${WHEN_SHORT[g.whenMet].toLowerCase()}`);
function underFloor(g, t) {
  const r = t.rule;
  if (!r || g.mode === 'forever') return null;
  const lows = [];
  if (r.minHours != null && g.hours != null && g.hours < r.minHours) lows.push(`${gh(g.hours)} is under ${t.name}'s ${hrs(r.minHours)}`);
  if (r.minRatio != null && g.ratio != null && g.ratio < r.minRatio) lows.push(`ratio ${r1(g.ratio)} is under ${t.name}'s ${r1(r.minRatio)}`);
  return lows.length ? `${lows.join(', and ')}. ${t.name}'s rule still holds each torrent until it's met.` : null;
}
function goalEditor(key) {
  const g = goalOf(key), u = units[key], forever = g.mode === 'forever';
  const tv = g.hours == null ? '' : u === 'd' ? +(g.hours / 24).toFixed(2) : g.hours;
  const unitSeg = seg('Time unit', [['d', 'days'], ['h', 'hours']], u, (v) => `data-g="${key}" data-f="unit" data-v="${v}"`, { disabled: forever, cls: 'seg-u' });
  return `<div class="ge ge-c${forever ? ' forever' : ''}" data-ge="${key}">
    <div class="fld"><label for="${key}-ratio">Ratio</label>${unitInput(`${key}-ratio`, g.ratio == null ? '' : r1(g.ratio), 'ratio', `data-g="${key}" data-f="ratio" ${forever ? 'disabled' : ''}`, 'None')}</div>
    <div class="fld fld-time"><label for="${key}-time">Time seeded</label><span class="time-w">${unitInput(`${key}-time`, tv, '', `data-g="${key}" data-f="time" ${forever ? 'disabled' : ''}`, 'None')}${unitSeg}</span></div>
    <div class="fld fld-mode"><span class="lbl">Goal is met at</span>${seg('Goal is met at', MODES, g.mode, (v) => `data-g="${key}" data-f="mode" data-v="${v}"`, { cls: 'seg-mode' })}</div>
    <div class="fld fld-wm"><span class="lbl">When met</span>${seg('When met', Object.entries(WHEN_SHORT), g.whenMet, (v) => `data-g="${key}" data-f="whenMet" data-v="${v}" title="${esc(D.whenMetLabel[v])}"`, { disabled: forever })}</div>
    <p class="ferr" data-ferr="${key}" hidden></p>
  </div>`;
}
function goalMetNow(t, g) {
  const R = g.ratio, T = g.hours;
  if (g.mode === 'forever') return false;
  if (R == null && T == null) return true;
  if (g.mode === 'both') return (R == null || t.ratio >= R) && (T == null || t.seededH >= T);
  return (R != null && t.ratio >= R) || (T != null && t.seededH >= T);
}
// What this tracker's goal does to its torrents right now. Recomputed on every edit.
function nowLine() {
  const g = effGoal(), o = { now: 0, held: 0, toward: 0, forever: 0, pending: 0, ext: 0 };
  V.forEach((v) => {
    const t = v.t;
    if (v.external) return o.ext++;
    if (t.state === 'error') return;
    if (!t.completedAt) return o.pending++;
    if (g.mode === 'forever') return o.forever++;
    if (!goalMetNow(t, g)) return o.toward++;
    if (!D.ruleMet(t, TR.rule)) return o.held++;
    o.now++;
  });
  const parts = [];
  if (o.now) parts.push(`<b>${plural(o.now, 'meets its goal', 'meet their goal')} now</b> and ${g.whenMet === 'remove' ? (o.now === 1 ? 'will be removed' : 'will be removed') : g.whenMet === 'stop' ? 'will stop seeding' : 'keep seeding'}`);
  if (o.held) parts.push(`${o.held} held by ${esc(TR.name)}'s rule`);
  if (o.toward) parts.push(`${o.toward} seeding toward it`);
  if (o.forever) parts.push(`${o.forever} seed forever`);
  if (o.pending) parts.push(`${o.pending} still downloading`);
  return `With this goal, of the ${V.length} torrents shown: ${parts.join(', ') || 'none affected'}.`;
}

function goalBox() {
  const t = TR, p = G.per[t.id], on = p.override;
  if (t.id === 'aurora') {
    const c = client(t.routes['seedbox-ams']);
    return `<div class="gbox" data-goalbox>
      <p class="tg-floor">${I('shield')}<span>${esc(floorText(t))}</span></p>
      <p class="tg-x">${I('info')}<span>Aurora's torrents live in ${esc(c.name)} on ${esc(c.node)}, whose own seeding rules apply, so a seeding goal does nothing here. Route Aurora back to the built-in client and ${on ? 'its own goal' : 'the default goal'} applies.</span><button class="lnk" data-jump="routing"><b>Tracker routing</b></button></p>
    </div>`;
  }
  const body = on
    ? `${goalEditor(t.id)}<p class="tg-sent" data-sent="${t.id}">${goalWords(p)}</p>${p.why ? `<p class="t3 tg-why">${I('message-square')}${esc(p.why)}</p>` : ''}`
    : `<p class="tg-inh" data-inh>Uses the default: <b>${esc(goalShort(G.def))}</b>. <button class="lnk" data-go="goals"><b>Edit the default</b></button></p>`;
  const low = underFloor(effGoal(), t);
  return `<div class="gbox${on ? ' own' : ''}" data-goalbox>
    <div class="gbox-h"><p class="tg-floor">${I(t.rule ? 'shield' : 'shield-off')}<span>${esc(floorText(t))}</span></p><span class="grow"></span>${phone ? swBare(on, `data-ovr="${t.id}"`, `${t.name} has its own goal`) : swBtn(on, `data-ovr="${t.id}"`, 'Own goal')}</div>
    ${body}
    <div data-low>${low ? `<p class="info-n">${I('info')}<span>${esc(low)}</span></p>` : ''}</div>
    <p class="gnow t2" data-now>${nowLine()}</p>
    <p class="also t3">${I('link-2')}<span>The same goal as <button class="lnk" data-go="goals"><b>Settings › Seeding goals</b></button>. A change here shows there too.</span></p>
  </div>`;
}

// ───────────────────────────── sections

function trackerStatus(tr) {
  if (tr.status === 'not-allowed') return `<span class="chip-warn">${I('shield-x')}Client not allowed</span>`;
  if (tr.status === 'degraded') return `<span class="chip-warn">${I('radio-tower')}Timing out</span>`;
  return `<span class="tk-ok">${dot('idle')}Working</span>`;
}
const openWindows = () => V.filter((v) => v.hn);
const atRisk = () => V.filter((v) => v.hn && v.hn.atRisk).sort((a, b) => a.hn.leftInWindowH - b.hn.leftInWindowH);

function stats(compact = false) {
  const s = TR.stats;
  const items = [
    ['Uploaded', `${s.uploadedTB} TB`], ['Downloaded', `${s.downloadedTB} TB`], ['Ratio', r2(s.uploadedTB / s.downloadedTB)],
    ['Torrents', fmtN(s.torrents)], ['Seeding', `${TR.seedingTB} TB`], ['Grabs · 30 d', String(TR.grabs30)],
  ];
  if (TR.hnrModel === 'window') items.push(['Open H&R windows', String(openWindows().length)]);
  return `<dl class="td-st num${compact ? ' td-st-c' : ''}">${items.map(([k, x]) => `<div><dt>${k}</dt><dd>${x}</dd></div>`).join('')}</dl>`;
}

function idLine() {
  return `<span class="chip node">${TR.private ? `${I('lock')}Private` : `${I('globe')}Public`}</span><span class="t3">${esc(TR.software)}</span>${trackerStatus(TR)}`;
}

function header() {
  const rule = TR.rule ? `${I('scale')}<span>${esc(TR.rule.note)}</span>` : `${I('globe')}<span>No tracker rule. Seeding goal only.</span>`;
  return `<header class="td-h">
    <div class="td-id"><h1>${esc(TR.name)}</h1>${idLine()}${TR.site ? `<span class="mono t3 td-site">${esc(TR.site)}</span>` : ''}</div>
    <p class="tk-rule t2 td-rule">${rule}</p>
    ${stats()}
  </header>`;
}

function issues() {
  const out = [];
  const r = client(Object.values(TR.routes)[0]);
  if (TR.status === 'not-allowed') out.push({ tone: 'warn', chip: 'Client not allowed', icon: 'shield-x', at: TR.since, text: `<b>${esc(TR.name)} doesn't allow the built-in client (-SS2000-) yet.</b> Grabs go to ${esc(r.name)} on ${esc(r.node)} until it does.`, act: ['Open routing', 'routing'] });
  if (TR.status === 'degraded') out.push({ tone: 'warn', chip: 'Announces timing out', icon: 'radio-tower', at: TR.since, text: `<b>No answer to announces.</b> ${V.filter((v) => v.t.announce.status === 'error').length} torrents keep seeding, but ${esc(TR.name)} may not credit seeding time until it answers. Retrying every 30 min.`, toast: 'Force reannounce' });
  D.indexersFor(TR.id).filter((i) => i.health === 'warn').forEach((i) => out.push({ tone: 'warn', chip: 'Indexer rate limited', icon: 'gauge', text: `<b>${esc(i.name)} Indexer.</b> ${esc(i.healthMsg)}`, act: ['Open Indexer', 'indexers'] }));
  const risk = atRisk();
  if (risk.length) out.push({ tone: 'warn', chip: 'Hit and run risk', icon: 'timer', text: `<b>${plural(risk.length, 'torrent', 'torrents')} close their window before reaching ${esc(TR.name)}'s minimum.</b> ${risk.map((v) => `${esc(v.w.title)} ${esc(cov(v.t))} <span class="num">needs ${dur(v.hn.needH)} in ${dur(v.hn.leftInWindowH)}</span>`).join('; ')}. Keep them seeding.`, act: ['Show', 'hnr'] });
  V.filter((v) => v.t.state === 'error').forEach((v) => out.push({ tone: 'fail', chip: 'Unregistered', icon: 'circle-x', at: v.t.announce.last, text: `<b>${esc(v.w.title)} ${esc(cov(v.t))}.</b> ${esc(TR.name)} removed this release. Seeding is over and no hit and run counts. The library file stays.`, rm: v.t.id }));
  return out;
}

function attention() {
  const is = issues();
  if (!is.length) return '';
  const line = (x) => {
    const btn = x.act ? `<button class="btn sm ${x.tone}" data-jump="${x.act[1]}">${esc(x.act[0])}</button>` : x.rm ? `<button class="btn sm fail" data-toast="Removing a torrent is not in this prototype">${I('trash-2')}Remove</button>` : x.toast ? `<button class="btn sm warn" data-toast="${esc(x.toast)} is not in this prototype">${esc(x.toast)}</button>` : '';
    return `<div class="att-i att-${x.tone}"><span class="chip-${x.tone}">${I(x.icon)}${esc(x.chip)}</span><p>${nw(x.text)}</p>${x.at ? `<span class="att-t num">since ${hm(x.at)}</span>` : '<span></span>'}${btn}</div>`;
  };
  return `<section class="att" aria-label="Needs attention" id="sec-attention">${is.map(line).join('')}</section>`;
}

// H&R: a countdown meter per open window. The bar is the whole window; the hatched block is the seeding still needed, starting now.
function winMeter(v) {
  const Wh = v.tr.rule.windowDays * 24, left = v.hn.leftInWindowH, need = v.hn.needH;
  const el = Math.max(0, Wh - left) / Wh, nd = Math.min(need, left) / Wh;
  const end = new Date(new Date(v.t.completedAt).getTime() + Wh * 36e5).toISOString();
  return `<div class="pg"><div class="wnd${v.hn.atRisk ? ' risk' : ''}" role="meter" aria-valuemin="0" aria-valuemax="${Math.round(Wh)}" aria-valuenow="${Math.round(Wh - left)}" aria-label="${esc(`${dur(need)} of seeding needed, ${dur(left)} left in the window`)}"><span class="wnd-e" style="width:${(el * 100).toFixed(1)}%"></span><span class="wnd-n" style="left:${(el * 100).toFixed(1)}%;width:${(nd * 100).toFixed(1)}%"></span></div>
    <p class="pg-t num"><b class="${v.hn.atRisk ? 'warn-t' : ''}">${dur(need)}</b> to seed${sep}${dur(left)} left${sep}closes ${md(end)}</p></div>`;
}

function winRow(v) {
  return `<li class="win" data-t="${v.t.id}" data-work="${v.w.id}" data-amb>${art(v.w, 'thumb-tr')}<span class="tw"><b class="ell">${esc(v.w.title)}</b><span class="t3 ell">${esc(cov(v.t))} · ${dur(v.t.seededH)} seeded</span></span>${winMeter(v)}${v.hn.atRisk ? `<span class="chip-warn">${I('timer')}At risk</span>` : `<span class="t3 win-s num">${dur(v.hn.leftInWindowH - v.hn.needH)} spare</span>`}</li>`;
}

function strip() {
  const hh = D.hnrHistory[TR.id];
  const max = Math.max(1, ...hh.days.map((d) => d[0]));
  const start = new Date(D.now.getTime() - 29 * 864e5);
  return `<div class="hstrip" role="img" aria-label="${esc(`Last 30 days: ${hh.cleared30} cleared, ${hh.hits30} hits and runs`)}">${hh.days.map(([c, x], i) => `<span class="hs-d${x ? ' hit' : ''}" title="${md(new Date(start.getTime() + i * 864e5).toISOString())}: ${c} cleared${x ? ', 1 hit and run' : ''}"><span style="height:${x ? 100 : Math.max(8, (c / max) * 100)}%"></span></span>`).join('')}</div>
    <p class="hs-ax t3 num"><span>${md(start.toISOString())}</span><span>Today</span></p>`;
}

function historyBlock() {
  const hh = D.hnrHistory[TR.id];
  return `<div class="hist">
    <div class="hist-n num"><div><b>${hh.cleared30}</b><span>cleared · 30 d</span></div><div><b class="${hh.hits30 ? 'fail-t' : ''}">${hh.hits30}</b><span>hits and runs · 30 d</span></div><div><b>${hh.hitsAll}</b><span>recorded on your account</span></div></div>
    ${strip()}
    <ul class="ev">${hh.events.map((e) => `<li class="ev-i ev-${e.kind}">${e.kind === 'hit' ? `<span class="chip-fail">${I('circle-x')}Hit and run</span>` : `<span class="chip">${I('check')}Cleared</span>`}<span class="ev-b"><b>${esc(e.title)}</b><span class="t3">${esc(e.detail)}</span></span><span class="t3 num ev-at">${md(e.at)}</span></li>`).join('')}</ul>
  </div>`;
}

function hnrBlock() {
  if (TR.hnrModel === 'none') return `<p class="empty-n">${I('shield-off')}<span>No tracker rule. Public torrents owe nothing, so there are no hit and run windows.</span></p>`;
  if (TR.hnrModel === 'ratio') {
    const a = D.hnrHistory.aurora.account, p = Math.min(1, a.ratio / 3), f = a.min / 3;
    return `<div class="acct"><p class="tg-floor">${I('scale')}<span>${esc(TR.name)} counts your account ratio, not windows per torrent.</span></p>
      <div class="gmb-r"><p class="gmb-h"><span>Account ratio</span><span class="num"><b>${r2(a.ratio)}</b> <span class="t3">minimum ${r2(a.min)}</span></span></p><span class="gm-l"><span class="gm-f" style="width:${(p * 100).toFixed(1)}%"></span><span class="gm-k k-ok" style="left:${(f * 100).toFixed(1)}%"></span></span></div>
      <p class="t3">${a.buffer} more could be downloaded before the ratio reaches ${r2(a.min)}.</p></div>`;
  }
  const wins = openWindows().sort((a, b) => (a.hn.leftInWindowH - a.hn.needH) - (b.hn.leftInWindowH - b.hn.needH));
  const pend = V.filter((v) => !v.external && !v.t.completedAt).length;
  return `<div class="hnr-g">
    <div class="hnr-open"><h3 class="sub-h">Open windows <span class="t3 num">${wins.length}</span></h3>
      ${wins.length ? `<ul class="wins">${wins.map(winRow).join('')}</ul>` : '<p class="t3">Nothing inside a window right now.</p>'}
      ${pend ? `<p class="t3 hnr-f">${plural(pend, 'download opens its', 'downloads open their')} window when ${pend === 1 ? 'it finishes' : 'they finish'}.</p>` : ''}</div>
    <div class="hnr-hist"><h3 class="sub-h">History</h3>${historyBlock()}</div>
  </div>`;
}

// Routing: one row per Node an Indexer for this tracker searches from. Default is the built-in client on that Node.
function routeNodes() {
  return [...new Set([...D.indexersFor(TR.id).map((i) => i.home), ...Object.keys(TR.routes)])];
}
function routeNotes(nid) {
  const c = client(TR.routes[nid]), out = [];
  if (TR.status === 'not-allowed' && c.kind === 'built-in') out.push(['warn', `${TR.name} rejects -SS2000- today, so these grabs will fail until staff add it to the allow list.`]);
  else if (TR.status === 'not-allowed') out.push(['why', `${TR.name}'s allow list doesn't have -SS2000- yet, so grabs go to ${c.name}. Move them back to the built-in once ${TR.name} allows it.`]);
  if (c.status === 'unreachable') out.push(['fail', `${c.name} on ${c.node} is unreachable. These grabs fail until it answers.`]);
  if (c.node !== nid && c.status !== 'unreachable') out.push(['why', `Grabs found on ${nid} download on ${c.node}. The .torrent file moves, the download doesn't.`]);
  return out;
}
const noteHtml = ([k, msg]) => `<p class="rt-n rt-${k}">${I(k === 'why' ? 'info' : k === 'fail' ? 'circle-x' : 'triangle-alert')}<span>${nw(esc(msg))}</span></p>`;
function clientSelect(nid) {
  const cur = TR.routes[nid];
  return `<select class="sel" data-route-node="${nid}" aria-label="${esc(TR.name)} grabs found on ${nid} go to">${D.downloadingNodes().map((n) => `<optgroup label="${n.name}">${D.clients.filter((c) => c.node === n.id).map((c) => `<option value="${c.id}" ${c.id === cur ? 'selected' : ''}>${c.kind === 'built-in' ? 'Built-in' : esc(c.name)} · ${c.node}${c.id === D.defaultRoute(nid) ? ' (default)' : ''}${c.status === 'unreachable' ? ' · unreachable' : ''}</option>`).join('')}</optgroup>`).join('')}</select>`;
}
function routeRow(nid, compact = false) {
  const ix = D.indexersFor(TR.id).filter((i) => i.home === nid), c = client(TR.routes[nid]), isDef = TR.routes[nid] === D.defaultRoute(nid);
  const notes = routeNotes(nid);
  const from = ix.length ? `Found by ${ix.map((i) => esc(i.name)).join(' and ')}` : 'No Indexer searches from here';
  return `<li class="rr${compact ? ' rr-c' : ''}" data-rr="${nid}">
    <div class="rr-from"><span class="t3">Searched from</span><b class="mono">${esc(nid)}</b><span class="t3">${from}</span></div>
    ${compact ? '' : `<span class="rr-ar t3">${I('arrow-right')}</span>`}
    <div class="rr-to">${clientSelect(nid)}<span class="rr-def t3">${isDef ? 'Default: built-in on the Indexer\'s home Node' : `Changed from the default${c.kind !== 'built-in' ? ` · ${esc(c.name)} keeps its own seeding rules` : ''}`}</span></div>
    ${notes.length ? `<div class="rt-notes">${notes.map(noteHtml).join('')}</div>` : ''}
  </li>`;
}
function allowBlock() {
  if (!TR.private) return `<p class="t3 allow">${I('globe')}<span>Public trackers accept any client.</span></p>`;
  if (TR.status !== 'not-allowed') return `<p class="t2 allow">${I('shield-check')}<span>${esc(TR.name)} allows SlipStream's built-in client <span class="nw">-SS2000-</span>.</span></p>`;
  return `<div class="allow allow-w"><p>${I('shield-x')}<span><b>${esc(TR.name)}'s client allow list</b> · ${esc(TR.allowRequest)}</span></p>
    <ul class="allow-l">${TR.allowList.map((a) => `<li class="chip node">${esc(a)}</li>`).join('')}<li class="chip node allow-no"><span class="nw">-SS2000-</span> not yet</li></ul>
    <button class="btn sm" data-toast="Rechecking the allow list is not in this prototype">${I('refresh-cw')}Recheck allow list</button></div>`;
}
function routingBlock(compact = false) {
  return `<div class="rt-wrap" data-routing><ul class="rrs">${routeNodes().map((n) => routeRow(n, compact)).join('')}</ul>${allowBlock()}
    <p class="also t3">${I('link-2')}<span>Also on <button class="lnk" data-go="clients"><b>Settings › Download Clients › Tracker routing</b></button>.</span></p></div>`;
}

// Indexers: each one a way to search this tracker, with its home Node, credentials and health.
function failoverText(i) {
  if (i.cred === 'session') return i.allowFailover ? `If ${i.home} is offline past 10 min, searches move to atlas and sign in again there.` : `Waits for ${i.home} if it goes offline. The cookie was earned there, and signing in from a new address can lock the account.`;
  return `If ${i.home} is offline past ${i.graceMin} min, searches move to ${i.failoverTo}.`;
}
function limitMeter(l, noun) {
  if (!l.max) return `<p class="ix-lim t3 num">${fmtN(l.used)} ${noun} today · no limit</p>`;
  const p = l.used / l.max, hot = p > 0.9;
  return `<div class="ix-lim"><p class="num t2"><b class="${hot ? 'warn-t' : ''}">${fmtN(l.used)}</b> of ${fmtN(l.max)} ${noun} per ${l.per}</p><span class="meter${hot ? ' hot' : ''}"><span style="width:${(Math.min(1, p) * 100).toFixed(1)}%"></span></span></div>`;
}
function ixHealth(i) {
  return i.health === 'ok' ? `<span class="t2 ix-h">${dot('idle')}Healthy · ${i.ms} ms · ${ago(i.lastOk)}</span>` : `<span class="warn-t ix-h">${dot('warn')}Rate limited · last answer ${ago(i.lastOk)}</span>`;
}
function ixCard(i, compact = false) {
  const others = i.trackers.filter((x) => x !== TR.id).map((x) => tracker(x).name);
  const auto = i.auto[TR.id] !== false;
  const top = `<div class="ix-top"><span class="ix-ic">${I(i.kind === 'torznab' ? 'rss' : i.kind === 'native' ? 'plug' : 'book-open')}</span><div class="ix-id"><p><b>${esc(i.name)}</b><span class="chip">${D.KIND_LABEL[i.kind]}</span>${i.kind === 'torznab' ? '<span class="chip node">Generic</span>' : ''}</p><p class="mono t3 ell">${esc(i.def)}</p></div>${compact ? '' : `<button class="btn ghost sm" data-toast="Editing an Indexer is not in this prototype">Edit</button>`}</div>`;
  const kv = `<dl class="kv ix-kv">
    <dt>Home</dt><dd>Searched from <b class="mono">${esc(i.home)}</b></dd>
    <dt>Sign-in</dt><dd>${esc(i.credText)}</dd>
    ${i.solver ? `<dt>Challenges</dt><dd>${i.solver.name} on <span class="mono">${i.solver.node}</span></dd>` : ''}
    <dt>Health</dt><dd>${ixHealth(i)}</dd>
    <dt>Failover</dt><dd>${esc(failoverText(i))}${i.cred === 'session' ? `<span class="ix-fo">${swBare(!!i.allowFailover, `data-fo="${i.id}"`, 'Allow failover')}<span class="t3">Allow failover anyway</span></span>` : ''}</dd>
    ${others.length ? `<dt>Also</dt><dd class="t2">Reaches ${others.map(esc).join(', ')} too</dd>` : ''}
  </dl>`;
  const lim = `<div class="ix-lims">${limitMeter(i.limits.queries, 'queries')}${limitMeter(i.limits.grabs, 'grabs')}</div>`;
  const tog = `<div class="ix-tog">
    <label class="ix-sw">${swBare(auto, `data-auto="${i.id}"`, `Auto search ${TR.name} through ${i.name}`)}<span>Auto search<span class="t3">${auto ? `${esc(TR.name)} included` : `Off for ${esc(TR.name)}`}</span></span></label>
    <label class="ix-sw">${swBare(i.rss.on, `data-rss="${i.id}"`, `RSS for ${i.name}`)}<span>RSS<span class="t3">${i.rss.on ? `every ${i.rss.every} min` : 'off'}</span></span></label>
    <p class="t3 num ix-pri">Priority ${i.priority}</p></div>`;
  return `<article class="ix${i.health !== 'ok' ? ' ix-warn' : ''}${compact ? ' ix-c' : ''}" data-ix="${i.id}">${top}<div class="ix-body">${kv}${lim}${tog}</div></article>`;
}
function indexersBlock(compact = false) {
  const ix = D.indexersFor(TR.id);
  return `<div class="ixs">${ix.map((i) => ixCard(i, compact)).join('')}</div>
    <p class="also t3">${I('link-2')}<span>${plural(ix.length, 'Indexer reaches', 'Indexers reach')} ${esc(TR.name)}. Credentials, limits and home Node are edited on <button class="lnk" data-toast="The Indexers page is not in this prototype"><b>Indexers</b></button>; the toggles here are the same settings.</span></p>`;
}

// Torrents: the By tracker table, with a state filter.
const F = { q: '', bucket: 'all' };
function torrentsBlock() {
  const vs = [...V].sort(byRank);
  const b = [['all', 'All', V.length], ['downloading', 'Downloading'], ['seeding', 'Seeding'], ['problems', 'Problems'], ['met', 'Goal met']].map(([k, l, n]) => [k, l, n ?? V.filter((v) => v.buckets.includes(k)).length]);
  const more = D.moreFor[TR.id];
  const search = `<label class="tf">${I('search')}<input type="search" class="tf-in" placeholder="Filter by Work or release" aria-label="Filter torrents" autocomplete="off" spellcheck="false"></label>`;
  return `<div class="tbar${phone ? ' tbar-ph' : ''}">${seg('Show torrents that are', b, F.bucket, (v) => `data-fb="${v}"`, { cls: 'fseg' })}<span class="grow"></span>${search}</div>
    <div class="tl tl-tracker tl-td" role="table" aria-label="Torrents from ${esc(TR.name)}" data-grp>${phone ? '' : head()}${vs.map(row).join('')}<div class="tempty" role="row" hidden><div role="cell">No torrents here match the filter.</div></div>
    ${more ? `<div class="tk-more"><button class="btn ghost sm" data-toast="Paging through every torrent is not in this prototype">${I('chevron-down')}${more} more from ${esc(TR.name)}</button></div>` : ''}</div>`;
}
function applyFilter() {
  root.querySelectorAll('.tl-td .tr[data-t]').forEach((r) => { r.hidden = !((F.bucket === 'all' || r.dataset.b.split(' ').includes(F.bucket)) && (!F.q || r.dataset.q.includes(F.q))); });
  const e = root.querySelector('.tl-td .tempty');
  if (e) e.hidden = [...root.querySelectorAll('.tl-td .tr[data-t]')].some((r) => !r.hidden);
}

function sec(id, title, inner, extra = '') {
  if (phone) return `<section class="psec td-ps" id="sec-${id}"><div class="psec-h"><h2>${title}</h2>${extra}</div><div class="tcard">${inner}</div></section>`;
  return `<section class="td-sec" id="sec-${id}" aria-labelledby="h-${id}"><div class="sec-h"><h2 id="h-${id}">${title}</h2>${extra ? `<span class="grow"></span>${extra}` : ''}</div><div class="tcard">${inner}</div></section>`;
}
function phoneTop(sub = '') {
  return `<header class="ph-nav"><button class="back tap" data-go="torrents">${I('chevron-left')}Torrents</button><span class="grow"></span><button class="ib tap" data-toast="Tracker actions are not in this prototype" aria-label="More">${I('ellipsis')}</button></header>
    <div class="td-ph-h"><h1>${esc(TR.name)}</h1><p class="td-id">${idLine()}</p><p class="tk-rule t2 td-rule">${TR.rule ? `${I('scale')}<span>${esc(TR.rule.note)}</span>` : `${I('globe')}<span>No tracker rule. Seeding goal only.</span>`}</p>${sub}</div>`;
}

// ═════════════════════════════ A · Dossier

function variantA() {
  const hnrTitle = TR.hnrModel === 'window' ? `Hits and runs` : TR.hnrModel === 'ratio' ? 'Hits and runs · account ratio' : 'Hits and runs';
  const nav = `<nav class="td-nav" aria-label="On this page">${[['seeding', 'Seeding'], ['hnr', 'Hits and runs'], ['routing', 'Tracker routing'], ['indexers', 'Indexers'], ['torrents', 'Torrents']].map(([id, l]) => `<button class="td-nav-i" data-jump="${id}">${l}</button>`).join('')}</nav>`;
  const att = attention();
  const body = `${phone ? '' : att}
    ${sec('seeding', 'Seeding', goalBox(), phone ? '' : `<span class="t3">The rule is ${esc(TR.name)}'s floor; the goal is yours</span>`)}
    ${sec('hnr', hnrTitle, hnrBlock())}
    ${sec('routing', 'Tracker routing', routingBlock(phone), phone ? '' : `<span class="t3">Which Download Client this tracker's grabs go to</span>`)}
    ${sec('indexers', 'Indexers', indexersBlock(), phone ? '' : `<span class="t3">Ways SlipStream searches ${esc(TR.name)}</span>`)}
    ${phone ? `<section class="psec td-ps" id="sec-torrents"><div class="psec-h"><h2>Torrents</h2><span class="t3 num">${fmtN(TR.stats.torrents)}</span></div>${torrentsBlock()}</section>` : `<section class="td-sec" id="sec-torrents"><div class="sec-h"><h2>Torrents</h2><span class="t3 num">${fmtN(TR.stats.torrents)}</span></div>${torrentsBlock()}</section>`}`;
  if (phone) return `${phoneTop(`<div class="td-ph-st">${stats()}</div>`)}<div class="td-ph-att">${att}</div>${body}`;
  return `<div class="td td-a">${header()}${nav}${body}</div>`;
}

// ═════════════════════════════ B · Tabs

const TABS = [['torrents', 'Torrents'], ['seeding', 'Seeding & H&R'], ['routing', 'Routing'], ['indexers', 'Indexers']];
let curTab = 'torrents';
function tabCount(id) {
  if (id === 'torrents') return fmtN(TR.stats.torrents);
  if (id === 'seeding') return TR.hnrModel === 'window' ? String(openWindows().length) : null;
  if (id === 'indexers') return String(D.indexersFor(TR.id).length);
  return null;
}
function tabFlag(id) {
  if (id === 'seeding') return atRisk().length ? 'warn' : '';
  if (id === 'routing') return TR.status === 'not-allowed' ? 'warn' : '';
  if (id === 'indexers') return D.indexersFor(TR.id).some((i) => i.health !== 'ok') ? 'warn' : '';
  if (id === 'torrents') return V.some((v) => v.reasons.some((r) => r.tone === 'fail')) ? 'fail' : '';
  return '';
}
function panelB(id) {
  if (id === 'torrents') return torrentsBlock();
  if (id === 'seeding') {
    return `<div class="b-seed"><section class="tcard"><div class="sec-h"><h2>Seeding goal</h2></div>${goalBox()}</section><section class="tcard"><div class="sec-h"><h2>Hits and runs</h2></div>${hnrBlock()}</section></div>`;
  }
  if (id === 'routing') return `<section class="tcard"><div class="sec-h"><h2>Tracker routing</h2><span class="t3">Which Download Client this tracker's grabs go to</span></div>${routingBlock(phone)}</section>`;
  return indexersBlock();
}
function attStrip() {
  const is = issues();
  if (!is.length) return '';
  return `<div class="b-att">${is.map((x) => `<span class="b-att-i">${dot(x.tone)}<b>${esc(x.chip)}</b></span>`).join('')}<button class="lnk t2" data-show-att>Details</button></div><div class="b-att-full" hidden>${attention()}</div>`;
}
function variantB() {
  const tabs = `<div class="b-tabs" role="tablist" aria-label="${esc(TR.name)} sections">${TABS.map(([id, l]) => { const n = tabCount(id), f = tabFlag(id); return `<button role="tab" class="b-tab" data-ttab="${id}" aria-selected="${id === curTab}" aria-controls="b-panel">${f ? dot(f) : ''}<span>${phone && id === 'seeding' ? 'Seeding' : l}</span>${n != null ? `<span class="seg-n num">${n}</span>` : ''}</button>`; }).join('')}</div>`;
  const panel = `<div class="b-panel" id="b-panel" role="tabpanel" data-panel>${panelB(curTab)}</div>`;
  if (phone) return `${phoneTop(`<div class="td-ph-st">${stats(true)}</div>`)}<div class="td-ph-att">${attStrip()}</div><div class="b-ph-tabs">${tabs}</div><div class="b-ph">${panel}</div>`;
  return `<div class="td td-b">
    <header class="b-h"><div class="td-id"><h1>${esc(TR.name)}</h1>${idLine()}${TR.site ? `<span class="mono t3 td-site">${esc(TR.site)}</span>` : ''}</div>${stats(true)}</header>
    <p class="tk-rule t2 b-rule">${TR.rule ? `${I('scale')}<span>${esc(TR.rule.note)}</span>` : `${I('globe')}<span>No tracker rule. Seeding goal only.</span>`}</p>
    ${attStrip()}
    ${tabs}${panel}</div>`;
}

// ═════════════════════════════ C · Obligations ledger

const GROUPS = [
  ['risk', 'At risk', 'timer', 'The window closes before the minimum. Pausing or removing one counts as a hit and run.'],
  ['owed', 'Owed', 'hourglass', 'Inside the rule window, on track.'],
  ['dl', 'Downloading', 'arrow-down', 'Nothing owed yet. The window opens when each finishes.'],
  ['goal', 'Rule met · seeding to goal', 'sprout', 'Free to remove without a hit and run. Seeding toward your goal.'],
  ['free', 'Free', 'circle-check', 'Goal met or no longer tracked. Removing these costs nothing.'],
  ['ext', 'In an external client', 'external-link', 'The external client keeps its own seeding rules.'],
];
function obl(v) {
  if (v.external) return 'ext';
  if (v.t.state === 'error') return 'free';
  if (!v.t.completedAt) return 'dl';
  if (v.hn) return v.hn.atRisk ? 'risk' : 'owed';
  if (v.t.state === 'met' || v.gp.kind === 'met') return 'free';
  return 'goal';
}
function oblRight(v, g) {
  if (g === 'risk' || g === 'owed') return winMeter(v);
  if (g === 'dl') return dlBar(v);
  if (g === 'goal') return goalMeter(v);
  if (g === 'ext') return `<p class="pg-t pg-x">${I('external-link')}<span>${esc(v.c.name)} on ${esc(v.n.name)} · ratio ${r2(v.t.ratio)}</span></p>`;
  if (v.t.state === 'error') return `<p class="pg-t pg-x"><span class="fail-t">Unregistered</span>${sep}no hit and run counts</p>`;
  return `<p class="pg-t pg-x">${I('check')}<span>${v.t.removeAt ? `Goal met · removing in ${minsUntil(v.t.removeAt)} min` : 'Goal met'}${sep}ratio ${r2(v.t.ratio)}</span></p>`;
}
function oblRow(v, g) {
  return `<li class="ob-r" data-t="${v.t.id}" data-work="${v.w.id}" data-amb tabindex="0">${art(v.w, 'thumb-tr')}<span class="tw"><b class="ell">${esc(v.w.title)}</b><span class="t3 ell">${esc(cov(v.t))} · ${esc(v.t.target)}${v.reasons.length && g !== 'risk' ? ` · <span class="${v.reasons[0].tone}-t">${esc(v.reasons[0].text)}</span>` : ''}</span></span><div class="ob-m">${oblRight(v, g)}</div><button class="ib" data-menu="${v.t.id}" aria-haspopup="${phone ? 'dialog' : 'menu'}" aria-label="Actions for ${esc(v.w.title)}">${I('ellipsis')}</button></li>`;
}
function groups() {
  const by = Object.fromEntries(GROUPS.map(([k]) => [k, []]));
  V.forEach((v) => by[obl(v)].push(v));
  by.risk.sort((a, b) => a.hn.leftInWindowH - a.hn.needH - (b.hn.leftInWindowH - b.hn.needH));
  by.owed.sort((a, b) => a.hn.leftInWindowH - a.hn.needH - (b.hn.leftInWindowH - b.hn.needH));
  return by;
}
function ledger() {
  const by = groups();
  return GROUPS.filter(([k]) => by[k].length || ['risk', 'owed', 'goal', 'free'].includes(k)).filter(([k]) => !(TR.hnrModel !== 'window' && (k === 'risk' || k === 'owed')) && !(TR.id === 'aurora' && by[k].length === 0)).map(([k, l, ic, why]) => {
    const n = by[k].length, open = k !== 'free' && n > 0;
    if (!TR.rule && k === 'goal') { l = 'Seeding to goal'; why = 'No tracker rule. Seeding toward your goal.'; }
    if (!TR.rule && k === 'dl') why = 'No tracker rule, so nothing is owed when they finish.';
    return `<details class="ob ob-${k}" id="ob-${k}" ${open ? 'open' : ''}><summary class="ob-h"><span class="ob-ic">${I(ic)}</span><b>${l}</b><span class="ob-n num">${n}</span><span class="t3 ob-why">${esc(why)}</span>${I('chevron-down', 'ob-chev')}</summary>
      ${n ? `<ul class="ob-l">${by[k].map((v) => oblRow(v, k)).join('')}</ul>` : `<p class="t3 ob-e">${k === 'risk' ? 'Nothing at risk.' : 'None right now.'}</p>`}</details>`;
  }).join('') + (D.moreFor[TR.id] ? `<p class="t3 ob-more">${I('info')}<span>Showing the ${V.length} newest. ${D.moreFor[TR.id]} older torrents from ${esc(TR.name)} ${TR.hnrModel === 'window' ? 'have met the rule' : TR.id === 'aurora' ? 'live in qBittorrent' : 'are seeding to their goal'}. <button class="lnk" data-toast="Paging is not in this prototype"><b>Show all</b></button></span></p>` : '');
}
function ledgerSummary() {
  const by = groups();
  const tiles = TR.id === 'aurora'
    ? [['ext', 'In qBittorrent', by.ext.length]]
    : TR.hnrModel === 'window' ? [['risk', 'At risk', by.risk.length], ['owed', 'Owed', by.owed.length], ['goal', 'Rule met', by.goal.length], ['free', 'Free', by.free.length]]
      : [['dl', 'Downloading', by.dl.length], ['goal', 'Seeding to goal', by.goal.length], ['free', 'Free', by.free.length]];
  return `<div class="ob-sum">${tiles.map(([k, l, n]) => `<button class="ob-t ob-t-${k}${n ? '' : ' zero'}" data-jump-ob="${k}"><b class="num">${n}</b><span>${l}</span></button>`).join('')}</div>`;
}
function historyCard() {
  if (TR.hnrModel === 'none') return `<div class="tcard c-hist"><p class="empty-n">${I('shield-off')}<span>No tracker rule. Public torrents owe nothing: no windows, no hits and runs.</span></p></div>`;
  if (TR.hnrModel === 'ratio') return `<div class="tcard c-hist">${hnrBlock()}</div>`;
  const hh = D.hnrHistory[TR.id];
  const last = hh.events.find((e) => e.kind === 'hit');
  return `<div class="tcard c-hist"><div class="c-hist-h"><p><b class="num">${hh.cleared30}</b> cleared${sep}<b class="num ${hh.hits30 ? 'fail-t' : ''}">${hh.hits30}</b> ${hh.hits30 === 1 ? 'hit and run' : 'hits and runs'} <span class="t3">in the last 30 days</span></p><span class="grow"></span><button class="btn ghost sm" data-hist>History ${I('chevron-right')}</button></div>
    ${strip()}
    ${last ? `<p class="t3 c-last">${I('circle-x')}<span>Last hit and run: ${esc(last.title)}, ${md(last.at)}. ${esc(last.detail)}</span></p>` : ''}</div>`;
}
function goalCard() {
  const t = TR, p = G.per[t.id];
  if (t.id === 'aurora') return `<section class="rc" data-goalcard><div class="rc-h"><h3>Rule and goal</h3></div><p class="tg-floor">${I('shield')}<span>${esc(floorText(t))}</span></p><p class="t2 rc-p">Goals don't apply: Aurora's torrents live in qBittorrent, which keeps its own seeding rules.</p></section>`;
  return `<section class="rc" data-goalcard><div class="rc-h"><h3>Rule and goal</h3><span class="grow"></span><button class="btn sm" data-edit-goal>${I('pencil')}Edit</button></div>
    <dl class="rc-kv"><dt>${I(t.rule ? 'shield' : 'shield-off')}Rule</dt><dd>${t.rule ? esc(t.rule.note) : 'None. Public torrents can stop any time.'}</dd>
    <dt>${I('target')}Goal</dt><dd><b>${esc(goalShort(effGoal()))}</b><span class="t3"> · ${p.override ? 'its own goal' : 'the default goal'}</span></dd></dl>
    <p class="t3 rc-now" data-now>${nowLine()}</p></section>`;
}
function routeCard() {
  return `<section class="rc"><div class="rc-h"><h3>Tracker routing</h3></div>${routingBlock(true)}</section>`;
}
function ixCards() {
  return `<section class="rc"><div class="rc-h"><h3>Indexers</h3><span class="t3 num">${D.indexersFor(TR.id).length}</span></div><div class="ixs ixs-c">${D.indexersFor(TR.id).map((i) => ixCard(i, true)).join('')}</div></section>`;
}
function goalSheet() {
  const p = G.per[TR.id];
  return `<div class="gsheet" role="dialog" aria-modal="true" aria-labelledby="gs-h"><p class="sheet-t"><b id="gs-h">${esc(TR.name)} seeding goal</b><span class="t3">Same goal as Settings › Seeding goals</span></p>${goalBox()}<div class="gs-act"><button class="btn pri" data-close>Done</button></div></div>`;
}
function variantC() {
  const main = `${attention()}${historyCard()}<div class="ledger">${ledger()}</div>`;
  const rail = `${goalCard()}${routeCard()}${ixCards()}`;
  if (phone) return `${phoneTop(`<div class="td-ph-st">${stats(true)}</div>`)}<div class="c-ph">${ledgerSummary()}${main}<h2 class="c-ph-h">Setup</h2>${rail}</div>`;
  return `<div class="td td-c">${header()}<div class="c-grid"><div class="c-main">${ledgerSummary()}${main}</div><aside class="c-rail">${rail}</aside></div></div>`;
}

// ═════════════════════════════ wiring

function refreshGoal() {
  root.querySelectorAll('[data-sent]').forEach((el) => { el.innerHTML = goalWords(goalOf(el.dataset.sent)); });
  root.querySelectorAll('[data-now]').forEach((el) => { el.innerHTML = nowLine(); });
  root.querySelectorAll('[data-low]').forEach((el) => { const low = underFloor(effGoal(), TR); el.innerHTML = low ? `<p class="info-n">${I('info')}<span>${esc(low)}</span></p>` : ''; });
  const gc = root.querySelector('[data-goalcard]');
  if (gc && !gc.closest('.sheet')) { const t = document.createElement('div'); t.innerHTML = goalCard(); gc.replaceWith(t.firstElementChild); }
  C.icons();
}
function setGoalField(key, f, v) {
  const g = goalOf(key);
  if (f === 'unit') {
    units[key] = v;
    const inp = root.querySelector(`#${key}-time`);
    if (inp && g.hours != null) inp.value = v === 'd' ? +(g.hours / 24).toFixed(2) : g.hours;
    return;
  }
  g[f] = v;
  if (f === 'mode') {
    root.querySelectorAll(`[data-ge="${key}"]`).forEach((ge) => {
      const forever = v === 'forever';
      ge.classList.toggle('forever', forever);
      ge.querySelectorAll('input').forEach((i) => { i.disabled = forever; });
      ge.querySelectorAll('.seg-u .seg-o, .fld-wm .seg-o').forEach((b) => { b.disabled = forever; });
    });
  }
}
function onGoalInput(el) {
  const key = el.dataset.g, f = el.dataset.f, g = goalOf(key);
  const raw = el.value.trim().replace(',', '.');
  const errEl = root.querySelector(`[data-ferr="${key}"]`);
  const num = raw === '' ? null : Number(raw);
  const bad = num != null && (!Number.isFinite(num) || num < 0);
  el.setAttribute('aria-invalid', String(bad));
  if (errEl) { errEl.hidden = !bad; errEl.innerHTML = bad ? `${I('circle-alert')}${f === 'ratio' ? 'Use a number like 1.5, or leave it empty' : 'Use a number of days or hours, or leave it empty'}` : ''; if (bad) C.icons(); }
  if (bad) return;
  if (f === 'ratio') g.ratio = num;
  if (f === 'time') g.hours = num == null ? null : Math.round(units[key] === 'd' ? num * 24 : num);
  refreshGoal();
}
function repaintGoalBoxes() {
  root.querySelectorAll('[data-goalbox]').forEach((el) => { const t = document.createElement('div'); t.innerHTML = goalBox(); el.replaceWith(t.firstElementChild); });
  refreshGoal();
}
function jump(id) {
  if (variant === 'B') {
    const map = { routing: 'routing', indexers: 'indexers', hnr: 'seeding', seeding: 'seeding', torrents: 'torrents' };
    return setTab(map[id] || 'torrents');
  }
  if (variant === 'C') {
    const target = id === 'hnr' ? root.querySelector('#ob-risk') || root.querySelector('.c-hist') : id === 'routing' ? root.querySelector('.c-rail .rt-wrap') : id === 'indexers' ? root.querySelector('.ixs-c') : null;
    if (target) { if (target.tagName === 'DETAILS') target.open = true; target.scrollIntoView({ behavior: 'smooth', block: 'start' }); }
    return;
  }
  const el = root.querySelector(`#sec-${id}`);
  if (el) el.scrollIntoView({ behavior: 'smooth', block: 'start' });
}
function setTab(id) {
  curTab = id;
  root.querySelectorAll('[data-ttab]').forEach((b) => b.setAttribute('aria-selected', String(b.dataset.ttab === id)));
  root.querySelector('[data-panel]').innerHTML = panelB(id);
  C.icons();
  wireFilter();
  const p = new URLSearchParams(location.search); p.set('tab', id); history.replaceState(null, '', `?${p}`);
}
function wireFilter() {
  const inp = root.querySelector('.tf-in');
  if (inp) inp.addEventListener('input', () => { F.q = inp.value.trim().toLowerCase(); applyFilter(); });
}
function selectRadio(el) {
  el.closest('[role="radiogroup"]').querySelectorAll('[role="radio"]').forEach((b) => { const on = b === el; b.setAttribute('aria-checked', String(on)); b.tabIndex = on ? 0 : -1; });
}

function click(e, q) {
  let el;
  if ((el = q('[data-g][data-v]'))) { if (el.disabled) return true; selectRadio(el); setGoalField(el.dataset.g, el.dataset.f, el.dataset.v); refreshGoal(); return true; }
  if ((el = q('[data-ovr]'))) {
    const p = G.per[el.dataset.ovr];
    p.override = !p.override;
    if (p.override && p.mode !== 'forever' && p.ratio == null && p.hours == null) Object.assign(p, { ...G.def });
    repaintGoalBoxes();
    const b = root.querySelector(`.sheet [data-ovr]`) || root.querySelector('[data-ovr]');
    if (b) b.focus({ preventScroll: true });
    toast(p.override ? `${TR.name} now has its own seeding goal` : `${TR.name} uses the default goal again`);
    return true;
  }
  if ((el = q('[data-fo]'))) { const i = D.indexers.find((x) => x.id === el.dataset.fo); i.allowFailover = !i.allowFailover; repaintIx(i); return true; }
  if ((el = q('[data-auto]'))) { const i = D.indexers.find((x) => x.id === el.dataset.auto); i.auto[TR.id] = i.auto[TR.id] === false; repaintIx(i); return true; }
  if ((el = q('[data-rss]'))) { const i = D.indexers.find((x) => x.id === el.dataset.rss); i.rss.on = !i.rss.on; repaintIx(i); return true; }
  if ((el = q('[data-ttab]'))) { setTab(el.dataset.ttab); return true; }
  if ((el = q('[data-fb]'))) { selectRadio(el); F.bucket = el.dataset.fb; applyFilter(); return true; }
  if ((el = q('[data-jump]'))) { jump(el.dataset.jump); return true; }
  if ((el = q('[data-jump-ob]'))) { const d = root.querySelector(`#ob-${el.dataset.jumpOb}`); if (d) { d.open = true; d.scrollIntoView({ behavior: 'smooth', block: 'start' }); } return true; }
  if ((el = q('[data-edit-goal]'))) { openSheet(goalSheet(), el); return true; }
  if ((el = q('[data-hist]'))) { openSheet(`<div class="gsheet" role="dialog" aria-modal="true" aria-labelledby="hs-h"><p class="sheet-t"><b id="hs-h">${esc(TR.name)} hits and runs</b><span class="t3">What ${esc(TR.name)} recorded</span></p>${historyBlock()}<div class="gs-act"><button class="btn" data-close>Close</button></div></div>`, el); return true; }
  if ((el = q('[data-show-att]'))) { const f = root.querySelector('.b-att-full'); f.hidden = !f.hidden; el.textContent = f.hidden ? 'Details' : 'Hide'; return true; }
  if (q('select')) return true;
  if ((el = q('[data-t]')) && !q('button, a, input, summary')) { C.go('torrent', { t: el.dataset.t }); return true; }
  return false;
}
function repaintIx(i) {
  root.querySelectorAll(`[data-ix="${i.id}"]`).forEach((el) => { const t = document.createElement('div'); t.innerHTML = ixCard(i, el.classList.contains('ix-c')); el.replaceWith(t.firstElementChild); });
  C.icons();
}

function actions(key) {
  const v = V.find((x) => x.t.id === key);
  return { title: `${v.w.title} · ${cov(v.t)}`, sub: `${stateLabel(v)} on ${v.n.name} · ${v.c.name}`, items: [['Open torrent', 'arrow-up-right', 'go', 'torrent'], ['Force reannounce', 'radio-tower'], ['Open Work', 'arrow-up-right', 'go', 'work'], ['Remove torrent', 'trash-2', 'act-danger', v.t.id]] };
}

export function render(ctx) {
  C = ctx; D = ctx.data; root = ctx.root; phone = ctx.device === 'phone'; variant = ctx.variant;
  TR = tracker(ctx.tracker);
  V = D.torrents.filter((t) => t.tracker === TR.id).map(view);
  curTab = TABS.some(([id]) => id === ctx.tab) ? ctx.tab : 'torrents';
  initGoals();
  root.classList.add(`td-v${variant}`);
  const lead = (V.find((v) => v.t.state === 'downloading') || V[0]).w.id;
  mount(ctx, {
    id: 'tracker',
    title: `${TR.name} · Trackers`,
    nav: 'torrents', tab: 'activity',
    crumbs: `<button class="crumb" data-go="torrents">Torrents</button>${I('chevron-right', 'crumb-sep')}<span class="crumb cur">${esc(TR.name)}</span>`,
    body: { A: variantA, B: variantB, C: variantC }[variant](),
    lead,
    actions,
    act() { closeAll(); toast('Removing a torrent is not in this prototype'); },
    click,
    after(r) {
      wireFilter();
      r.addEventListener('input', (e) => { if (e.target.dataset.g) onGoalInput(e.target); });
      r.addEventListener('change', (e) => {
        const el = e.target;
        if (!el.dataset.routeNode) return;
        const nid = el.dataset.routeNode;
        TR.routes[nid] = el.value;
        r.querySelectorAll(`[data-rr="${nid}"]`).forEach((x) => { const t = document.createElement('div'); t.innerHTML = routeRow(nid, x.classList.contains('rr-c')); x.replaceWith(t.firstElementChild); });
        C.icons();
        const c = client(el.value);
        toast(`${TR.name} grabs found on ${nid} now go to ${c.kind === 'built-in' ? 'the built-in client' : c.name} on ${c.node}`);
      });
      r.addEventListener('keydown', (e) => {
        const tr = e.target.closest && e.target.closest('[data-t]');
        if (tr && e.target === tr && (e.key === 'Enter' || e.key === ' ')) { e.preventDefault(); C.go('torrent', { t: tr.dataset.t }); }
        if (e.target.matches && e.target.matches('input, select') && (e.key === 'ArrowLeft' || e.key === 'ArrowRight')) e.stopPropagation();
      });
    },
  });
}
