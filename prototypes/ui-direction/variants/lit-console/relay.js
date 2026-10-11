// PROTOTYPE · Direction C · Remote access diagnostics. Three layouts over one fixture, behind ?layout= and ?scenario=. Throwaway.

import { scenarios, HOPS, hopStatus, exposureStatus, contentTypes, RANK } from '../../relay-data.js';

let h, D, R, I, esc;
const params = new URLSearchParams(location.search);
export const scenarioId = scenarios[params.get('scenario')] ? params.get('scenario') : 'plex';
export const LAYOUTS = { path: 'Path', verdict: 'Verdict', matrix: 'Matrix' };
export const layoutId = LAYOUTS[params.get('layout')] ? params.get('layout') : 'verdict';

const HOP = Object.fromEntries(HOPS.map((x) => [x.id, x]));
const GLYPH = { pass: 'check', warn: 'triangle-alert', fail: 'x', skip: 'circle-dashed', off: 'minus' };
const WORD = { pass: 'Passes', warn: 'Warning', fail: 'Fails', skip: 'Not reached', off: 'Not used' };
const CT = Object.fromEntries(contentTypes.map((t) => [t.id, t]));
const hm = (iso) => new Date(iso).toTimeString().slice(0, 5);

const glyph = (s, cls = '') => `<span class="g g-${s} ${cls}" role="img" aria-label="${WORD[s]}">${I(GLYPH[s])}</span>`;
function badge(s, text = WORD[s]) {
  if (s === 'fail') return `<span class="chip-fail">${I('circle-x')}${esc(text)}</span>`;
  if (s === 'warn') return `<span class="chip-warn">${I('triangle-alert')}${esc(text)}</span>`;
  return `<span class="sb sb-${s}">${I(GLYPH[s])}${esc(text)}</span>`;
}

// every check in the diagnostic, by id, with where it lives
function indexChecks() {
  const all = new Map();
  for (const e of R.exposures) for (const hp of HOPS) for (const c of e.hops[hp.id].checks) all.set(c.id, { c, e, hop: hp.id });
  if (R.cgnatCheck) all.set('cgnat', { c: R.cgnatCheck, e: null, hop: 'node' });
  return all;
}
let CK;
const rootOf = (id) => { let x = CK.get(id); while (x && x.c.because && CK.get(x.c.because)) x = CK.get(x.c.because); return x; };

const FIX = [];
function fixBtn(f, cls = '', short = true) {
  if (!f) return '';
  FIX.push(f);
  const ic = f.action === 'copy' ? 'copy' : f.action === 'open' ? 'arrow-up-right' : 'wand-sparkles';
  return `<button class="btn ${cls}" data-fix="${FIX.length - 1}">${I(ic)}${esc(short ? f.button || f.label : f.label)}</button>`;
}

const works = () => R.exposures.filter((e) => exposureStatus(e) === 'pass').length;
function sentence() {
  const n = R.exposures.length, ok = works();
  if (!R.headline.length) return `<span class="s-live">${h.dot('ok')}Remote access works</span><span class="s-sep" aria-hidden="true">·</span><span class="t2">${ok} of ${n} routes reach atlas through ${esc(R.relay.node)}</span>`;
  const worstK = R.headline.some((x) => x.k === 'fail') ? 'fail' : 'warn';
  return `<span class="s-live">${h.dot(worstK)}${ok} of ${n} routes work from outside</span>${R.headline.map((x) => `<span class="s-sep" aria-hidden="true">·</span><span class="s-${x.k}">${esc(x.text)}</span>`).join('')}`;
}

function runBar() {
  return `<div class="rl-run"><span class="t3">Last run <b class="num t1">${hm(R.lastRun)}</b> from ${esc(R.probeFrom)}</span><button class="btn" data-run>${I('refresh-cw')}<span>Run diagnostics again</span></button></div>`;
}

// ───────────────────────────── shared blocks

function checkItem(c) {
  const bad = c.status === 'warn' || c.status === 'fail';
  const cause = c.because && CK.get(c.because);
  return `<li class="ck ck-${c.status}">
    <div class="ck-h">${glyph(c.status)}<b>${esc(c.label)}</b></div>
    <dl class="ck-kv"><dt>${bad ? 'Observed' : 'Seen'}</dt><dd class="mono">${esc(c.observed)}</dd>${bad ? `<dt>Expected</dt><dd class="mono">${esc(c.expected)}</dd>` : ''}</dl>
    <p class="ck-m">${esc(c.meaning)}</p>
    ${cause && cause.c !== c ? `<p class="ck-b">${I('corner-down-right')}<span>Follows from <b>${esc(cause.c.label)}</b>${cause.e ? ` on ${esc(cause.e.short)}` : ''}</span></p>` : ''}
    ${c.fix ? `<div class="fx">${fixBtn(c.fix, c.status === 'fail' ? 'fail sm' : c.status === 'warn' ? 'warn sm' : 'sm')}<p class="t2">${esc(c.fix.label)}.</p>${c.fix.action === 'copy' ? `<pre class="fx-pre mono">${esc(c.fix.detail)}</pre>` : `<p class="t3">${esc(c.fix.detail)}</p>`}</div>` : ''}
  </li>`;
}

function sectionsBlock(ms) {
  return `<div class="secs"><p class="secs-h">${esc(ms.name)} shares</p><ul class="secs-l">${ms.sections.map((s) => {
    const t = CT[s.type], L = t.layout;
    const unit = s.count === 1 ? L.unit.one : L.unit.many;
    const child = L.child && s.child ? ` · ${s.child.toLocaleString('en-US')} ${s.child === 1 ? L.child.one : L.child.many}` : '';
    return `<li class="sec" title="${esc(`${ms.name} section ${s.title}, Content Type ${t.noun.one}`)}"><span class="sec-art" style="aspect-ratio:${L.aspect}">${I(t.icon)}</span><b>${esc(s.title)}</b><span class="t3 sec-n num">${s.count.toLocaleString('en-US')} ${unit}${child}</span></li>`;
  }).join('')}</ul></div>`;
}

function hopDetail(e, hopId) {
  const hp = e.hops[hopId], H = HOP[hopId], st = hopStatus(hp);
  const ms = e.mediaServer && hopId === 'server' ? R.mediaServers.find((m) => m.id === e.mediaServer) : null;
  return `<div class="hd">
    <div class="hd-h"><span class="hd-ic">${I(H.icon)}</span><div class="hd-t"><span class="t3">${esc(e.short)} · ${H.label}</span><b>${esc(hp.label)}</b></div>${badge(st)}</div>
    ${hp.off ? `<p class="hd-off t2">${esc(hp.off)}</p>` : `<ul class="cks">${hp.checks.map(checkItem).join('')}</ul>`}
    ${ms ? sectionsBlock(ms) : ''}
  </div>`;
}

function recommendBlock(compact = false) {
  const x = R.recommend;
  if (!x) return `<section class="rx rx-calm" aria-label="Recommendation"><span class="rx-ok">${I('check')}</span><p><b>Nothing to do.</b> <span class="t2">${esc(R.relay.node)} has relayed ${esc(R.relay.relayedToday)} today. The next automatic run is at 22:08.</span></p></section>`;
  const group = issueGroups().find((g) => g.lead);
  const also = group ? group.items.filter((i) => i.c.id !== x.check && i.c.id !== 'cgnat') : [];
  return `<section class="rx rx-${x.level}${compact ? ' rx-compact' : ''}" aria-labelledby="rx-t">
    <div class="rx-h"><span class="chip-${x.level}">${I(x.level === 'fail' ? 'circle-x' : 'triangle-alert')}${esc(x.chip)}</span>${x.since ? `<span class="rx-dim num">since ${esc(x.since)}</span>` : ''}<span class="rx-dim">${esc(HOP[x.hop].label)} on ${esc(R.exposures.find((e) => e.id === x.exposure).short)}</span></div>
    <h2 class="rx-t" id="rx-t">${esc(x.title)}</h2>
    <dl class="fail-kv rx-kv"><dt>Cause</dt><dd>${esc(x.cause)}</dd><dt>Do this</dt><dd><b>${esc(x.fix.label)}.</b>${x.fix.action === 'copy' ? `<pre class="fx-pre mono">${esc(x.fix.detail)}</pre>` : ` <span class="rx-dim">${esc(x.fix.detail)}</span>`}</dd>${x.then ? `<dt>Then</dt><dd>${esc(x.then)}</dd>` : ''}</dl>
    <div class="rx-act">${fixBtn(x.fix, x.level)}${x.alt ? fixBtn(x.alt) : ''}${layoutId !== 'verdict' ? `<button class="btn ghost" data-show="${x.exposure}:${x.hop}">${I('crosshair')}Show the hop</button>` : ''}</div>
    ${also.length && !compact ? `<div class="rx-also"><p class="rx-dim">Also explained by this</p><ul>${also.map((i) => `<li>${glyph(i.c.status)}<span><b>${esc(i.e ? i.e.short : 'atlas')}</b> <span class="rx-dim">${esc(i.c.label)}</span></span><span class="mono rx-dim">${esc(i.c.observed)}</span></li>`).join('')}</ul></div>` : ''}
  </section>`;
}

// failures and warnings grouped by root cause; one group per (hop, label) of the root
function issueGroups() {
  const groups = new Map();
  const lead = R.recommend && rootOf(R.recommend.check);
  for (const [, x] of CK) {
    if (x.c.status !== 'fail' && x.c.status !== 'warn') continue;
    const r = rootOf(x.c.id);
    const key = `${r.hop}|${r.c.label}`;
    if (!groups.has(key)) groups.set(key, { key, root: r, items: [], lead: lead && `${lead.hop}|${lead.c.label}` === key });
    groups.get(key).items.push(x);
  }
  return [...groups.values()].sort((a, b) => (b.lead - a.lead) || (RANK[b.root.c.status] - RANK[a.root.c.status]));
}

function relayBlock() {
  const r = R.relay;
  if (!r) {
    const cands = R.nodes.filter((n) => n.kind !== 'Controller' && n.cgnat.status === 'pass');
    return `<section class="rl-card" aria-labelledby="relay-h"><div class="pnl-h"><h2 id="relay-h">Relay</h2><span class="grow"></span>${badge('fail', 'None')}</div>
      <p class="t2 rl-p">No Node holds Relaying, so atlas has no way in from the internet.</p>
      <ul class="cands">${cands.map((n) => `<li><span class="ltxt"><b class="mono">${esc(n.id)}</b><span class="t3">${esc(n.where)} · public IP <span class="mono">${esc(n.pub)}</span> · paired</span></span>${fixBtn(R.recommend.fix, 'sm')}</li>`).join('')}
        <li><span class="ltxt"><b>A new VPS</b><span class="t3">Any machine with a public IP, paired with a Join Code</span></span>${fixBtn(R.recommend.alt, 'ghost sm')}</li></ul>
    </section>`;
  }
  return `<section class="rl-card" aria-labelledby="relay-h"><div class="pnl-h"><h2 id="relay-h">Relay</h2><span class="t3"><b class="mono t1">${esc(r.node)}</b> · ${esc(r.where)}</span><span class="grow"></span>${badge(r.status === 'pass' ? 'pass' : r.status, r.status === 'pass' ? 'Up' : r.status === 'warn' ? 'Degraded' : 'Failing')}</div>
    <dl class="rl-kv">
      <dt>Public IPv4</dt><dd class="mono">${esc(r.ipv4)}</dd>
      <dt>Public IPv6</dt><dd class="mono">${esc(r.ipv6)}</dd>
      <dt>Node link</dt><dd>${esc(r.link.direction)} <span class="t3">· up ${esc(r.link.uptime)}</span></dd>
      <dt>Pinned key</dt><dd class="mono">${esc(r.link.key)}</dd>
      <dt>Round trip</dt><dd class="num">${esc(r.link.rtt)} <span class="t3">· ${esc(r.link.throughput)}</span></dd>
      <dt>Relayed today</dt><dd class="num">${esc(r.relayedToday)}</dd>
      <dt>Roles</dt><dd>${r.roles.map((x) => `<span class="chip">${esc(x)}</span>`).join(' ')}</dd>
    </dl></section>`;
}

const FACTS = [['cgnat', 'CGNAT'], ['inbound', 'Inbound'], ['portmap', 'Port mapping'], ['reach', 'Reaches the Controller']];
const fact = (f, cls = '') => `<span class="fact ${cls}">${glyph(f.status, 'g-sm')}<span>${esc(f.value)}</span></span>`;

function nodesTable() {
  return `<section class="rl-card rl-nodes" aria-labelledby="nodes-h"><div class="pnl-h"><h2 id="nodes-h">Node reachability</h2><span class="t3">What each Node's router reports and what ${esc(R.probeFrom)} sees from outside</span></div>
    <table class="ntab"><thead><tr><th scope="col">Node</th><th scope="col">LAN · WAN · public</th><th scope="col">CGNAT</th><th scope="col">Inbound</th><th scope="col">Port mapping</th><th scope="col">Reaches the Controller</th><th scope="col" class="r">Latency</th></tr></thead>
    <tbody>${R.nodes.map((n) => `<tr class="${n.cgnat.status === 'fail' ? 'nt-fail' : ''}"><th scope="row"><b class="mono">${esc(n.id)}</b><span class="t3">${esc(n.kind)} · ${esc(n.where)}</span></th>
      <td class="mono ips"><span>${esc(n.lan)}</span><span class="${n.wan !== n.pub ? 'ip-x' : ''}">${esc(n.wan)}</span><span>${esc(n.pub)}</span></td>
      <td>${fact(n.cgnat)}</td><td>${fact(n.inbound)}</td><td>${fact(n.portmap)}</td><td>${fact(n.reach)}</td><td class="num r">${esc(n.latency || '')}</td></tr>${n.cgnat.status === 'fail' ? `<tr class="nt-fail nt-noterow"><td></td><td colspan="6"><p class="nt-note">${esc(n.cgnat.note)}</p></td></tr>` : ''}`).join('')}</tbody></table></section>`;
}

function nodesList() {
  return `<ul class="nlist2">${R.nodes.map((n) => {
    const st = [n.cgnat, n.inbound, n.portmap, n.reach].reduce((a, f) => (RANK[f.status] > RANK[a] ? f.status : a), 'pass');
    if (st === 'pass' || st === 'off') return `<li class="nl2 nl2-ok"><div class="nl2-h">${glyph('pass')}<b class="mono">${esc(n.id)}</b><span class="t3 ell">${esc(n.cgnat.value)} · ${esc(n.reach.value)}</span><span class="grow"></span><span class="mono t3 nl2-ip">${esc(n.pub)}</span></div></li>`;
    return `<li class="nl2 ${st === 'fail' ? 'nl2-fail' : ''}"><div class="nl2-h">${glyph(st === 'off' ? 'pass' : st)}<b class="mono">${esc(n.id)}</b><span class="t3">${esc(n.kind)}</span><span class="grow"></span><span class="mono t3 nl2-ip">${esc(n.pub)}</span></div>
      <dl class="nl2-kv">${FACTS.map(([k, l]) => `<dt>${l}</dt><dd>${fact(n[k])}</dd>`).join('')}</dl>
      ${n.cgnat.status === 'fail' ? `<p class="nt-note">${esc(n.cgnat.note)}</p>` : ''}</li>`;
  }).join('')}</ul>`;
}

function mediaBlock() {
  return `<section class="rl-card" aria-labelledby="ms-h"><div class="pnl-h"><h2 id="ms-h">Media Servers</h2><span class="t3">through the Media Server link on atlas</span></div>
    ${R.mediaServers.map((m) => `<div class="msv"><div class="msv-h"><b>${esc(m.name)}</b><span class="t3 mono">atlas :${m.port}</span><span class="grow"></span>${badge(m.remote.status, m.remote.value)}</div>
      <p class="t3 msv-l">Last inbound connection <span class="num t2">${esc(m.lastInbound)}</span></p>${sectionsBlock(m)}</div>`).join('')}
  </section>`;
}

// ───────────────────────────── layout: Path

let sel;
function defaultSel() {
  if (R.recommend) return { e: R.recommend.exposure, hop: R.recommend.hop };
  return { e: 'plex', hop: 'server' };
}

function hopNode(e, hp, i) {
  const x = e.hops[hp.id], st = hopStatus(x), on = sel.e === e.id && sel.hop === hp.id;
  return `<button class="hn hn-${st}${i === 0 ? ' hn-first' : ''}" data-hop="${e.id}:${hp.id}" aria-pressed="${on}" aria-label="${esc(`${e.short}, ${hp.label}: ${x.label}, ${WORD[st]}`)}">
    <span class="hn-k">${esc(hp.label)}${glyph(st, 'g-sm')}</span>
    <b class="hn-l">${esc(x.off ? 'Not used' : x.label).replace(/\./g, '.<wbr>')}</b>
    <span class="hn-s mono">${esc(x.off ? x.label : x.sub)}</span>
  </button>`;
}

function lane(e) {
  const st = exposureStatus(e);
  return `<section class="lane lane-${st}" aria-label="${esc(e.name)}">
    <div class="lane-h"><b>${esc(e.name)}</b>${e.named ? '<span class="chip">named port</span>' : ''}<span class="mono t3 ell">${esc(e.address)}</span><span class="grow"></span>${badge(st === 'skip' ? 'fail' : st, st === 'pass' ? 'Works' : st === 'warn' ? 'Degraded' : 'Unreachable')}</div>
    <ol class="route"><li class="rt-start" title="A player on the internet, probed from ${esc(R.probeFrom)}">${I('monitor-smartphone')}<span class="t3">Player</span></li>${HOPS.map((hp, i) => `<li>${hopNode(e, hp, i)}</li>`).join('')}</ol>
  </section>`;
}

function pathWide() {
  sel = defaultSel();
  const e = R.exposures.find((x) => x.id === sel.e);
  return `${recommendBlock(true)}
    <div class="pw-grid">
      <div class="lanes">${R.exposures.map(lane).join('')}<button class="lane lane-add" data-toast="Naming a port is not in this prototype" disabled>${I('plus')}Add a port<span class="t3">relay-fra forwards any TCP port to a Node you pick</span></button></div>
      <aside class="pdet" aria-label="Hop detail" aria-live="polite">${hopDetail(e, sel.hop)}</aside>
    </div>
    <div class="pw-foot">${nodesTable()}<div class="pw-side">${relayBlock()}</div></div>`;
}

function pathPhone() {
  sel = defaultSel();
  return `${recommendBlock(true)}
    <div class="xpick" role="tablist" aria-label="Route">${R.exposures.map((e) => `<button role="tab" class="xp" data-pick="${e.id}" aria-selected="${e.id === sel.e}">${h.dot(exposureStatus(e) === 'pass' ? 'idle' : exposureStatus(e) === 'warn' ? 'warn' : 'fail')}<span>${esc(e.short)}</span></button>`).join('')}</div>
    <div class="vroutes">${R.exposures.map((e) => vroute(e)).join('')}</div>
    ${phoneSection('Relay', relayBlock())}
    ${phoneSection('Node reachability', nodesList())}`;
}

function vroute(e) {
  const st = exposureStatus(e);
  return `<section class="vr" data-route="${e.id}" ${e.id === sel.e ? '' : 'hidden'} aria-label="${esc(e.name)}">
    <div class="vr-h"><span class="ltxt"><b>${esc(e.name)}</b><span class="mono t3 ell">${esc(e.address)}</span></span>${badge(st === 'skip' ? 'fail' : st, st === 'pass' ? 'Works' : st === 'warn' ? 'Degraded' : 'Unreachable')}</div>
    <ol class="vlist"><li class="vh vh-start"><span class="vh-rail">${I('monitor-smartphone')}</span><span class="ltxt"><b>Player</b><span class="t3">probed from ${esc(R.probeFrom)}</span></span></li>
    ${HOPS.map((hp) => { const x = e.hops[hp.id], s = hopStatus(x); return `<li><button class="vh vh-${s}" data-hop="${e.id}:${hp.id}"><span class="vh-rail">${glyph(s)}</span><span class="ltxt"><span class="vh-k">${esc(hp.label)}</span><b class="ell">${esc(x.off ? 'Not used' : x.label)}</b><span class="mono t3 ell">${esc(x.off ? x.label : x.sub)}</span></span>${I('chevron-right', 'vh-chev')}</button></li>`; }).join('')}</ol>
  </section>`;
}

// ───────────────────────────── layout: Verdict

function issueCard(g) {
  const r = g.root, c = r.c, st = c.status;
  const where = [...new Set(g.items.map((i) => (i.e ? i.e.short : 'atlas')))];
  const others = g.items.filter((i) => i.c.id !== c.id);
  const chip = st === 'fail' ? `Failed at ${r.hop === 'node' ? 'atlas' : HOP[r.hop].label}` : `${r.hop === 'node' ? 'atlas' : HOP[r.hop].label} warning`;
  return `<section class="ic2 ic2-${st}">
    <div class="rx-h"><span class="chip-${st}">${I(st === 'fail' ? 'circle-x' : 'triangle-alert')}${esc(chip)}</span><span class="rx-dim">${esc(where.join(', '))}</span></div>
    <h3 class="ic2-t">${esc(c.label)}</h3>
    <dl class="fail-kv rx-kv"><dt>Observed</dt><dd class="mono">${esc(c.observed)}</dd><dt>Expected</dt><dd class="mono">${esc(c.expected)}</dd><dt>Cause</dt><dd>${esc(c.meaning)}</dd></dl>
    ${c.fix ? `<div class="rx-act">${fixBtn(c.fix, st)}<span class="rx-dim">${esc(c.fix.label)}.</span></div>${c.fix.action === 'copy' ? `<pre class="fx-pre mono">${esc(c.fix.detail)}</pre>` : ''}` : ''}
    ${others.length ? `<div class="rx-also"><p class="rx-dim">Also explained by this</p><ul>${others.map((i) => `<li>${glyph(i.c.status)}<span><b>${esc(i.e ? i.e.short : 'atlas')}</b> <span class="rx-dim">${esc(i.c.label)}</span></span><span class="mono rx-dim">${esc(i.c.observed)}</span></li>`).join('')}</ul></div>` : ''}
  </section>`;
}

function passing() {
  return R.exposures.map((e) => {
    const cs = HOPS.flatMap((hp) => e.hops[hp.id].checks.map((c) => ({ c, hp })));
    const ok = cs.filter((x) => x.c.status === 'pass');
    const skip = cs.filter((x) => x.c.status === 'skip').length;
    const offs = HOPS.filter((hp) => e.hops[hp.id].off).length;
    const st = exposureStatus(e);
    const ms = e.mediaServer && R.mediaServers.find((m) => m.id === e.mediaServer);
    return `<details class="px"${R.recommend ? '' : ''}><summary>${glyph(st === 'skip' ? 'fail' : st)}<b>${esc(e.name)}</b><span class="mono t3 ell px-a">${esc(e.address)}</span><span class="grow"></span><span class="t3 num">${ok.length} pass${skip ? ` · ${skip} not reached` : ''}${offs ? ` · ${offs} not used` : ''}</span>${I('chevron-down', 'px-chev')}</summary>
      <ul class="px-l">${ok.map(({ c, hp }) => `<li>${glyph('pass', 'g-sm')}<span class="px-hop t3">${esc(hp.label)}</span><span>${esc(c.label)}</span><span class="mono t3 ell">${esc(c.observed)}</span></li>`).join('') || '<li class="t3">Nothing passes yet.</li>'}</ul>
      ${ms && !R.recommend ? sectionsBlock(ms) : ''}
    </details>`;
  }).join('');
}

function verdictMain() {
  const groups = issueGroups().filter((g) => !g.lead);
  return `${recommendBlock()}
    ${groups.length ? `<div class="ics">${groups.map(issueCard).join('')}</div>` : ''}
    <section aria-labelledby="pass-h"><div class="sec-h"><h2 id="pass-h">${R.recommend ? 'Everything else' : 'Routes'}</h2><span class="t3">${R.recommend ? 'Checks that pass, by route' : 'Every check passes'}</span></div><div class="pxs">${passing()}</div></section>`;
}

function verdictWide() {
  return `<div class="vd-grid"><div class="vd-main">${verdictMain()}</div><aside class="vd-side">${relayBlock()}<section class="rl-card" aria-labelledby="nr-h"><div class="pnl-h"><h2 id="nr-h">Node reachability</h2></div>${nodesList()}</section>${mediaBlock()}</aside></div>`;
}

function verdictPhone() {
  return `${verdictMain()}${phoneSection('Relay', relayBlock())}${phoneSection('Node reachability', nodesList())}${phoneSection('Media Servers', mediaBlock())}`;
}

// ───────────────────────────── layout: Matrix

function matrix(phone) {
  return `<div class="rmx-wrap"><table class="rmx${phone ? ' rmx-ph' : ''}"><caption class="sr">Every route by every hop</caption>
    <thead><tr><th scope="col" class="rmx-c">Route</th>${HOPS.map((hp) => `<th scope="col"><span class="rmx-th">${I(hp.icon)}<span>${esc(phone ? ({ link: 'Link', server: 'Media', origin: 'Origin' }[hp.id] || hp.label) : hp.label)}</span></span></th>`).join('')}</tr></thead>
    <tbody>${R.exposures.map((e) => { const st = exposureStatus(e); return `<tr><th scope="row" class="rmx-r"><b>${esc(phone ? e.short : e.name)}</b>${phone ? '' : `<span class="mono t3 ell">${esc(e.address)}</span>`}</th>
      ${HOPS.map((hp) => { const x = e.hops[hp.id], s = hopStatus(x); return `<td class="rc rc-${s}"><button class="rc-b" data-cell="${e.id}:${hp.id}" aria-haspopup="dialog" aria-label="${esc(`${e.short}, ${hp.label}: ${WORD[s]}`)}">${glyph(s)}${phone ? '' : `<span class="rc-t"><b>${esc(x.off ? 'Not used' : x.label)}</b><span class="mono">${esc(x.off ? '' : x.sub)}</span></span>`}</button></td>`; }).join('')}</tr>`; }).join('')}
      <tr class="rmx-add"><th scope="row" colspan="${HOPS.length + 1}"><button class="btn ghost sm" disabled>${I('plus')}Add a port</button><span class="t3">relay-fra forwards any TCP port to a Node you pick</span></th></tr>
    </tbody></table>
    ${phone ? '' : `<p class="rmx-legend t3">${['pass', 'warn', 'fail', 'skip', 'off'].map((s) => `<span>${glyph(s, 'g-sm')}${WORD[s]}</span>`).join('')}</p>`}</div>`;
}

function matrixWide() {
  return `${recommendBlock(true)}<section aria-labelledby="mx-h"><div class="sec-h"><h2 id="mx-h">Routes</h2><span class="t3">Player on the left, the service on atlas on the right. Select a cell for its checks.</span></div>${matrix(false)}</section>
    <div class="pw-foot">${nodesTable()}<div class="pw-side">${relayBlock()}${mediaBlock()}</div></div>`;
}

function matrixPhone() {
  return `${recommendBlock(true)}${phoneSection('Routes', matrix(true), `<span class="t3">tap a cell</span>`)}${phoneSection('Node reachability', nodesList())}${phoneSection('Relay', relayBlock())}${phoneSection('Media Servers', mediaBlock())}`;
}

// ───────────────────────────── shells

const phoneSection = (title, inner, extra = '') => `<section class="psec"><div class="psec-h"><h2>${title}</h2>${extra}</div><div class="psec-b">${inner}</div></section>`;

export function wide(helpers) {
  setup(helpers);
  const body = { path: pathWide, verdict: verdictWide, matrix: matrixWide }[layoutId]();
  return `<h1 class="sr">Remote access</h1>
    <section class="pulse rl-pulse" aria-label="Remote access status"><p class="sentence">${sentence()}</p>${runBar()}</section>
    ${body}`;
}

export function phone(helpers) {
  setup(helpers);
  const body = { path: pathPhone, verdict: verdictPhone, matrix: matrixPhone }[layoutId]();
  return `<div class="statusbar" aria-hidden="true"></div>
    <header class="ph-nav"><button class="back tap" data-go="home">${I('chevron-left')}Home</button><span class="grow"></span><button class="ib tap" data-run aria-label="Run diagnostics again">${I('refresh-cw')}</button></header>
    <div class="rl-ph">
      <header class="ph-head"><h1>Remote access</h1></header>
      <p class="ph-sentence">${sentence()}</p>
      <p class="rl-ph-run t3">Last run <span class="num t2">${hm(R.lastRun)}</span> from ${esc(R.probeFrom)}</p>
      ${body}
    </div>
    ${h.tabs(h.TABS_ADMIN, 'more')}`;
}

function setup(helpers) {
  h = helpers; D = h.D; I = h.I; esc = h.esc;
  R = scenarios[scenarioId].build();
  CK = indexChecks();
  FIX.length = 0;
}

// ───────────────────────────── interaction

function openCell(btn, eId, hopId) {
  const e = R.exposures.find((x) => x.id === eId);
  h.closeAll();
  h.setLastFocus(btn);
  if (h.isPhone) return openSheet(e, hopId);
  const pop = h.root.querySelector('.pop');
  pop.innerHTML = hopDetail(e, hopId);
  pop.classList.add('rl-pop');
  pop.setAttribute('role', 'dialog');
  pop.setAttribute('aria-label', `${e.short}, ${HOP[hopId].label}`);
  pop.hidden = false;
  h.icons();
  const r = btn.getBoundingClientRect(), rr = h.root.getBoundingClientRect();
  const left = Math.min(r.left - rr.left, rr.width - pop.offsetWidth - 16);
  pop.style.left = `${Math.max(8, left)}px`;
  const below = innerHeight - r.bottom - 16, above = r.top - 70;
  const up = below < 360 && above > below;
  pop.style.maxHeight = `${Math.min(560, up ? above : below)}px`;
  pop.style.top = up ? `${r.top - rr.top - 6 - Math.min(pop.scrollHeight, Math.min(560, above))}px` : `${r.bottom - rr.top + 6}px`;
}

function openSheet(e, hopId) {
  const s = h.root.querySelector('.sheet-wrap');
  s.querySelector('.sheet').innerHTML = `<span class="grab" aria-hidden="true"></span><div class="rl-sheet" role="dialog" aria-label="${esc(`${e.short}, ${HOP[hopId].label}`)}">${hopDetail(e, hopId)}</div><button class="sheet-x" data-close>Done</button>`;
  s.hidden = false;
  h.icons();
  s.querySelector('.sheet-x').focus({ preventScroll: true });
}

function selectHop(eId, hopId, scroll) {
  sel = { e: eId, hop: hopId };
  h.root.querySelectorAll('.hn').forEach((b) => b.setAttribute('aria-pressed', String(b.dataset.hop === `${eId}:${hopId}`)));
  const det = h.root.querySelector('.pdet');
  det.innerHTML = hopDetail(R.exposures.find((x) => x.id === eId), hopId);
  h.icons();
  if (scroll) h.root.querySelector(`.hn[data-hop="${eId}:${hopId}"]`).scrollIntoView({ block: 'center', behavior: 'smooth' });
}

function pick(eId) {
  sel = { e: eId, hop: sel.hop };
  h.root.querySelectorAll('.xp').forEach((b) => b.setAttribute('aria-selected', String(b.dataset.pick === eId)));
  h.root.querySelectorAll('.vr').forEach((v) => { v.hidden = v.dataset.route !== eId; });
}

export function wire(helpers) {
  const { root } = helpers;
  root.addEventListener('click', async (ev) => {
    const q = (s) => ev.target.closest(s);
    let el;
    if ((el = q('[data-fix]'))) {
      const f = FIX[+el.dataset.fix];
      if (f.action === 'copy') { try { await navigator.clipboard.writeText(f.detail); } catch { /* clipboard blocked in previews */ } return h.toast(`Copied. ${f.detail.split('\n').length > 1 ? 'Paste the records' : 'Paste the record'} at your DNS provider`); }
      if (f.action === 'open') return h.toast(`${f.label} is not in this prototype`);
      return h.toast(`${f.label}. Running diagnostics again`);
    }
    if ((el = q('[data-run]'))) {
      el.classList.add('spin');
      el.disabled = true;
      setTimeout(() => { el.classList.remove('spin'); el.disabled = false; h.toast(`Ran again from ${R.probeFrom}. Nothing changed since ${hm(R.lastRun)}`); }, 1400);
      return;
    }
    if ((el = q('[data-hop]'))) {
      const [e, hp] = el.dataset.hop.split(':');
      if (h.isPhone) { h.setLastFocus(el); return openSheet(R.exposures.find((x) => x.id === e), hp); }
      return selectHop(e, hp, false);
    }
    if ((el = q('[data-cell]'))) { ev.stopPropagation(); const [e, hp] = el.dataset.cell.split(':'); return openCell(el, e, hp); }
    if ((el = q('[data-pick]'))) return pick(el.dataset.pick);
    if ((el = q('[data-show]'))) {
      const [e, hp] = el.dataset.show.split(':');
      if (layoutId === 'path') { if (h.isPhone) { pick(e); h.setLastFocus(el); return openSheet(R.exposures.find((x) => x.id === e), hp); } return selectHop(e, hp, true); }
      const cell = root.querySelector(`[data-cell="${e}:${hp}"]`);
      cell.scrollIntoView({ block: 'center' });
      return openCell(cell, e, hp);
    }
  });
}
