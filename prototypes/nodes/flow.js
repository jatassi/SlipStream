// PROTOTYPE · Variant E · Flow canvas. Organized around how work moves: stages from Search to Notify are columns, Nodes
// are lanes, a station sits wherever a Node holds that stage's Role, and live jobs move between stations.

import { ROLES, jobs, works, CONTROLLER_VERSION } from './data.js';
import { esc, I, dot, plural, wideShell, crumbs, nodeState, hm, itemLabel } from './shell.js';
import { controllerOf, jobsOn } from './model.js';
import { header, defaultActions, detailBody } from './detail.js';
import { pairingHtml, handoverHtml } from './flows.js';
import { format, onTick } from '../ui-direction/live.js';
import * as A from './a.js';
import * as B from './b.js';
import { canvasHtml, mounted, wire, curve, isNarrow, reduceMotion } from './canvas.js';
import { SERVICES } from './topology.js';

export const name = 'Flow';
const ID = 'E';

export const STAGES = [
  { id: 'search', label: 'Search', role: 'Searching', svc: 'indexers', verb: 'searching' },
  { id: 'download', label: 'Download', role: 'Downloading', svc: 'swarm', verb: 'downloading' },
  { id: 'import', label: 'Import', role: 'Storage', svc: null, verb: 'importing' },
  { id: 'convert', label: 'Convert', role: 'Converting', svc: null, verb: 'converting' },
  { id: 'subs', label: 'Subtitles', role: 'Subtitles', svc: 'subs', verb: 'subtitling' },
  { id: 'msl', label: 'Media Server', role: 'Media Server link', svc: 'plex', verb: 'notifying Plex' },
  { id: 'notify', label: 'Notify', role: 'Notifying', svc: 'smtp', verb: 'notifying' },
];
const STAGE_OF_JOB = { Searching: 'search', Downloading: 'download', Transfer: 'download', Import: 'import', Conversion: 'convert' };
const IDX = Object.fromEntries(STAGES.map((s, i) => [s.id, i]));

let selJob = null;

const holdersOf = (nodes, role) => nodes.filter((n) => n.roles[role]);
const live = (n, role) => n.roles[role]?.state === 'active' && n.state !== 'offline';

export function routeOf(j, nodes) {
  const byId = (id) => nodes.find((n) => n.id === id) || controllerOf(nodes);
  const pick = (role) => nodes.find((n) => live(n, role)) || holdersOf(nodes, role)[0] || null;
  const [from, to] = j.stage === 'Transfer' ? j.node.split(' → ').map(byId) : [null, null];
  const at = {
    search: j.stage === 'Searching' ? byId(j.node) : pick('Searching'),
    download: j.stage === 'Downloading' ? byId(j.node) : from || pick('Downloading'),
    import: j.stage === 'Import' ? byId(j.node) : to || pick('Storage'),
    convert: j.stage === 'Conversion' ? byId(j.node) : null,
    subs: pick('Subtitles'), msl: pick('Media Server link'), notify: pick('Notifying'),
  };
  const now = IDX[STAGE_OF_JOB[j.stage]] + (j.stage === 'Transfer' && from === to ? 1 : 0);
  const steps = STAGES.filter((s) => at[s.id]).map((s) => ({ stage: s.id, node: at[s.id], t: IDX[s.id] < now ? 'done' : IDX[s.id] === now ? 'now' : 'next' }));
  if (j.stage === 'Transfer' && from !== to) steps.find((s) => s.stage === 'download').t = 'done';
  if (j.stage === 'Transfer' && from === to) steps.find((s) => s.stage === 'import').t = 'now';
  return steps;
}

const SW = 156, SH = 62, COLW = 196, X0 = 250, Y0 = 128;
const LANE = 104;

function laneOrder(nodes) {
  const first = (n) => { const i = STAGES.findIndex((s) => n.roles[s.role]); return i < 0 ? 99 : i; };
  return [...nodes].sort((a, b) => first(a) - first(b));
}

function stations(nodes) {
  const order = laneOrder(nodes);
  const out = [];
  STAGES.forEach((s, i) => {
    holdersOf(nodes, s.role).forEach((n) => {
      const x = X0 + i * COLW;
      const y = Y0 + order.indexOf(n) * LANE + (LANE - SH) / 2;
      out.push({ stage: s.id, node: n, x, y, key: `${s.id}:${n.id}` });
    });
  });
  return out;
}

function hopPath(a, b) {
  const skip = IDX[b.stage] - IDX[a.stage] > 1 && a.y === b.y;
  const p = { x: a.x + SW, y: a.y + SH / 2, side: 'r' }, q = { x: b.x, y: b.y + SH / 2, side: 'l' };
  if (!skip) return curve(p, q);
  const lift = 44;
  const pp = { x: a.x + SW / 2, y: a.y, side: 't' }, qq = { x: b.x + SW / 2, y: b.y, side: 't' };
  const c1 = { x: pp.x, y: pp.y - lift }, c2 = { x: qq.x, y: qq.y - lift };
  const at = (t) => { const u = 1 - t; return { x: u * u * u * pp.x + 3 * u * u * t * c1.x + 3 * u * t * t * c2.x + t * t * t * qq.x, y: u * u * u * pp.y + 3 * u * u * t * c1.y + 3 * u * t * t * c2.y + t * t * t * qq.y }; };
  return { d: `M${pp.x},${pp.y} C${c1.x},${c1.y} ${c2.x},${c2.y} ${qq.x},${qq.y}`, at };
}

const RANK = { potential: 0, next: 1, done: 2, now: 3 };

function hopsAll(nodes, st) {
  const at = new Map(st.map((s) => [s.key, s]));
  const hops = new Map();
  const add = (a, b, t, j) => {
    const k = `${a.key}>${b.key}`;
    if (!hops.has(k)) hops.set(k, { a, b, t: 'potential', jobs: [] });
    const h = hops.get(k);
    if (RANK[t] > RANK[h.t]) h.t = t;
    if (j) h.jobs.push({ j, t });
  };
  const chain = [['search', 'download'], ['download', 'import'], ['import', 'convert'], ['convert', 'subs'], ['import', 'subs'], ['subs', 'msl'], ['msl', 'notify']];
  for (const [x, y] of chain) for (const a of st.filter((s) => s.stage === x)) for (const b of st.filter((s) => s.stage === y)) {
    if (x === 'import' && y === 'subs' && st.some((s) => s.stage === 'convert')) continue;
    add(a, b, 'potential');
  }
  for (const j of jobs.filter((x) => x.status === 'running' || x.status === 'failed')) {
    const r = routeOf(j, nodes);
    for (let i = 0; i < r.length - 1; i++) {
      const a = at.get(`${r[i].stage}:${r[i].node.id}`), b = at.get(`${r[i + 1].stage}:${r[i + 1].node.id}`);
      if (!a || !b) continue;
      const t = r[i + 1].t === 'now' || (j.stage === 'Transfer' && r[i].stage === 'download') ? 'now' : r[i + 1].t === 'done' ? 'done' : 'next';
      add(a, b, j.status === 'failed' && t === 'next' ? 'potential' : t, j);
    }
  }
  return [...hops.values()];
}

function linkNote(a, b, ctl) {
  if (a === b) return '';
  const w = a === ctl ? b : b === ctl ? a : null;
  if (w?.link?.relay) return `via ${w.link.relay} · ${w.link.latency} ms`;
  if (w) return `${w.link.latency} ms`;
  return `via ${ctl.name}`;
}

function stationHtml(app, s) {
  const st = STAGES[IDX[s.stage]];
  const n = s.node;
  const draining = n.roles[st.role]?.state === 'draining';
  const js = jobs.filter((j) => j.status === 'running' && routeOf(j, app.S.nodes).some((x) => x.t === 'now' && x.stage === s.stage && x.node === n));
  const off = n.state === 'offline';
  const behind = ['updating', 'behind'].includes(n.state);
  const rate = js.filter((j) => j.rate != null);
  const units = {};
  rate.forEach((j) => { units[j.unit] = (units[j.unit] || 0) + j.rate; });
  const rateTxt = Object.entries(units).map(([u, v]) => `${v.toFixed(v < 20 ? 1 : 0)} ${u}`).join(' + ');
  const head = off ? 'Waiting' : draining ? 'Draining' : js.length ? `${js.length} ${st.verb}` : 'Idle';
  const sub = off ? `<span class="fail-t">${esc(n.name)} is offline</span>` : draining ? esc(n.roles[st.role].left) : behind && js.length ? '<span class="warn-t">finishing, starts nothing new</span>' : rateTxt ? `<span class="num">${rateTxt}</span>` : `<span class="t3">${esc(st.role)}</span>`;
  return `<div class="fx-st${js.length && !off ? ' busy' : ''}${off ? ' off' : ''}${draining ? ' draining' : ''}${behind ? ' behind' : ''}${n.roles[st.role]?.handedBack ? ' handed' : ''}" data-stage="${s.stage}" data-node="${n.id}" data-bound style="left:${s.x}px;top:${s.y}px;width:${SW}px;height:${SH}px" data-go='${JSON.stringify({ node: n.id, view: 'nodes' })}'>
    <span class="fx-st-h">${dot(off ? 'fail' : behind ? 'warn' : js.length ? 'draft' : 'idle')}<b>${esc(head)}</b>${n.roles[st.role]?.handedBack ? '<span class="cv-tail">handed back</span>' : ''}</span>
    <span class="fx-st-l">${sub}</span>
  </div>`;
}

function laneHead(app, n, y) {
  const s = nodeState(n);
  const extra = ['Metadata', 'Relaying', 'Serving'].filter((r) => n.roles[r]);
  return `<div class="fx-lane${n.state === 'offline' ? ' off' : ''}" style="top:${y}px;height:${LANE}px;width:${X0 + STAGES.length * COLW}px" data-bound></div>
  <div class="fx-lh${app.cfg.node === n.id ? ' sel' : ''}" style="top:${y + 12}px;height:${LANE - 24}px" data-node="${n.id}" data-go='${JSON.stringify({ node: app.cfg.node === n.id ? '' : n.id, view: 'nodes' })}' role="button" tabindex="0">
    <span class="fx-lh-n">${dot(s.k)}<b class="mono">${esc(n.name)}</b>${n.kind === 'Controller' ? I('crown') : ''}<span class="grow"></span><span class="num ${n.version === CONTROLLER_VERSION ? 't3' : 'warn-t'}">${esc(n.version)}</span></span>
    <span class="t3 ell">${esc(n.where)}${n.link?.relay ? ` · via ${esc(n.link.relay)}` : ''}</span>
    <span class="fx-lh-x t3 ell">${n.state === 'offline' ? `<span class="fail-t">Offline since ${hm(n.lastSeen)}</span>` : extra.length ? `also ${extra.join(', ')}` : esc(s.label)}</span>
  </div>`;
}

function colHead(s, i, nodes, height) {
  const svc = SERVICES.find((x) => x.id === s.svc);
  const none = !holdersOf(nodes, s.role).length;
  return `<div class="fx-col" data-col="${s.id}" style="left:${X0 + i * COLW - 20}px;width:${SW + 40}px;height:${height}px" data-bound></div>
  <div class="fx-ch" data-hl="${s.id}" style="left:${X0 + i * COLW}px;width:${SW}px">
    <span class="fx-ch-i num t3">${i + 1}</span><b>${esc(s.label)}</b>
    <span class="t3">${esc(s.role)}${ROLES[s.role].always ? ` ${I('anchor')}` : ''}</span>
    <span class="fx-ch-s t3">${svc ? `${I(svc.icon)}${esc(svc.label)}` : s.id === 'import' ? `${I('hard-drive')}library roots` : `${I('cpu')}proven by trial run`}</span>
    ${none ? '<span class="fx-ch-none t3">No holder. Skipped.</span>' : ''}
  </div>`;
}

function jobTitle(j) { const w = works[j.work]; return `${w.title}${j.itemKey ? ` ${itemLabel(j.work, j.itemKey)}` : ''}`.replace(/ Standard$/, ''); }

function jobTray(app, nodes) {
  const js = jobs.filter((j) => j.status === 'running' || j.status === 'failed');
  return `<div class="fx-tray cv-ui" aria-label="Live jobs"><span class="fx-tray-h t3">Live jobs<br>trace one</span>${js.map((j) => {
    const n = nodes.find((m) => j.node.startsWith(m.id));
    const held = n?.state === 'offline';
    return `<button class="fx-job${selJob === j.id ? ' on' : ''}${j.status === 'failed' ? ' failed' : ''}" data-act="e-job" data-job="${j.id}" aria-pressed="${selJob === j.id}" title="${esc(jobTitle(j))} · ${esc(j.stage)} · ${esc(j.node)}">
      <b class="ell">${esc(works[j.work].title)}</b>
      <span class="t3 ell">${esc(j.stage)} · ${j.status === 'failed' ? '<span class="fail-t">failed</span>' : held ? '<span class="fail-t">held</span>' : `<span class="num" data-live-pct="${j.id}">${format.pct(j.progress)}</span>`}</span>
    </button>`;
  }).join('')}</div>`;
}

function pathCard(app, nodes) {
  const j = jobs.find((x) => x.id === selJob);
  if (!j) return '';
  const r = routeOf(j, nodes);
  const ctl = controllerOf(nodes);
  const parts = r.map((s, i) => {
    const prev = r[i - 1];
    const hop = prev && prev.node !== s.node ? `<span class="fx-hop t3">${I('arrow-right')}${esc(linkNote(prev.node, s.node, ctl))}</span>` : prev ? `<span class="fx-hop t3">${I('arrow-right')}</span>` : '';
    return `${hop}<span class="fx-step t-${s.t}"><b>${esc(STAGES[IDX[s.stage]].label)}</b><span class="mono">${esc(s.node.name)}</span></span>`;
  }).join('');
  return `<div class="fx-path cv-ui"><div class="fx-path-h"><b>${esc(jobTitle(j))}</b><span class="t3">${esc(j.target)}</span><span class="grow"></span><button class="ib" data-act="e-job" data-job="${j.id}" aria-label="Clear path">${I('x')}</button></div><div class="fx-steps">${parts}</div>${j.status === 'failed' ? `<p class="fail-t">${esc(j.cause)}</p>` : ''}</div>`;
}

function beadAt(h, j) {
  const c = hopPath(h.a, h.b);
  return c.at(Math.max(0.04, Math.min(0.96, j.progress)));
}

function world(app) {
  const nodes = app.S.nodes;
  const ctl = controllerOf(nodes);
  const st = stations(nodes);
  const hops = hopsAll(nodes, st);
  const height = nodes.length * LANE + (Y0 - 28);
  let svg = `<defs>${['now', 'done', 'next', 'potential', 'sel'].map((k) => `<marker id="fx-a-${k}" viewBox="0 0 10 10" refX="9" refY="5" markerWidth="6" markerHeight="6" orient="auto"><path d="M0,1 L9,5 L0,9 z" class="fx-ah-${k}"/></marker>`).join('')}</defs>`;
  let labels = '', beads = '';
  const order = [...hops].sort((x, y) => RANK[x.t] - RANK[y.t]);
  for (const h of order) {
    const c = hopPath(h.a, h.b);
    const on = selJob && h.jobs.some((x) => x.j.id === selJob);
    const cross = h.a.node !== h.b.node;
    const held = h.a.node.state === 'offline' || h.b.node.state === 'offline';
    const cls = `fx-e t-${h.t}${cross ? ' cross' : ''}${held ? ' held' : ''}${selJob ? (on ? ' sel' : ' dim') : ''}`;
    svg += `<path class="${cls}" d="${c.d}" data-from="${h.a.key}" data-to="${h.b.key}" marker-end="url(#fx-a-${on ? 'sel' : h.t})"/>`;
    if (h.t === 'now' && !held && !reduceMotion()) svg += `<path class="fx-flow${selJob && !on ? ' dim' : ''}" d="${c.d}"/>`;
    const nowJob = cross ? h.jobs.find((x) => x.t === 'now' && x.j.stage === 'Transfer') : null;
    if (nowJob && !held) {
      const m = beadAt(h, nowJob.j);
      beads += `<div class="fx-bead" data-bead="${nowJob.j.id}" style="left:${m.x}px;top:${m.y}px" title="${esc(jobTitle(nowJob.j))}">${esc(works[nowJob.j.work].title.split(' ').slice(0, 2).join(' '))}</div>`;
    }
    const m = c.at(0.5);
    const showNote = cross && !held;
    const side = h.b.y > h.a.y ? 'to-l' : h.b.y < h.a.y ? 'to-r' : '';
    if (nowJob && !held) labels += `<div class="fx-l now ${side}${selJob && !on ? ' dim' : ''}" style="left:${m.x}px;top:${m.y}px">${I('arrow-right-left')}<span class="num" data-live-rate="${nowJob.j.id}">${format.rate(nowJob.j)}</span><span class="t3">${esc(linkNote(h.a.node, h.b.node, ctl))}</span></div>`;
    else if (showNote && h.t !== 'potential') labels += `<div class="fx-l ${side}${selJob && !on ? ' dim' : ''}" style="left:${m.x}px;top:${m.y}px"><span class="t3">${esc(linkNote(h.a.node, h.b.node, ctl))}</span></div>`;
    if (held && h.t !== 'potential') labels += `<div class="fx-l held" style="left:${m.x}px;top:${m.y}px">${I('pause')}<span>held</span></div>`;
  }
  const heads = STAGES.map((s, i) => colHead(s, i, nodes, height)).join('');
  const lanesHtml = laneOrder(nodes).map((n, i) => laneHead(app, n, Y0 + i * LANE)).join('');
  return `${heads}${lanesHtml}<svg class="cv-svg fx-svg" aria-hidden="true">${svg}</svg>${st.map((s) => stationHtml(app, s)).join('')}<div class="cv-labels">${labels}${beads}</div>`;
}

function stateNote(app) {
  const nodes = app.S.nodes;
  const off = nodes.find((n) => n.state === 'offline');
  if (nodes.length === 1) return 'One Node. Every stage runs in process on atlas, so nothing crosses a Node link.';
  if (off) return `${off.name} is offline. Its searches and downloads are held. Searches requeue at ${hm(off.requeueAt)} only if another Node holds Searching.`;
  if (nodes.some((n) => n.state === 'updating' || n.state === 'behind')) return 'Nodes on an older version finish what they hold and take no new work until they update.';
  if (nodes.some((n) => Object.values(n.roles).some((r) => r.state === 'draining'))) return 'Draining stations finish their jobs and take nothing new. New work routes to the other holders.';
  return '';
}

export function render(app) {
  if (isNarrow(app)) {
    const out = A.render(app);
    return { ...out, html: out.html.replace('</header>', `</header><p class="cv-fallback">${I('monitor')}<span>The Flow canvas is desktop only. This is A, Roster.</span></p>`) };
  }
  wire(app.root);
  wireHover(app.root);
  const { cfg, S } = app;
  const n = S.nodes.find((m) => m.id === cfg.node);
  app.flowClose = `<button class="ib" data-go='${JSON.stringify({ view: 'nodes' })}' aria-label="Close">${I('x')}</button>`;
  const note = stateNote(app);
  const ui = `<div class="cv-title cv-ui"><h1>Nodes</h1><p class="t2">${plural(S.nodes.length, 'Node')} · ${plural(jobs.filter((j) => j.status === 'running').length, 'live job')} · hover a stage to see its holders</p></div>
    <div class="cv-acts cv-ui">${S.nodes.length > 1 ? `<button class="btn" data-act="handover">${I('crown')}Hand over</button>` : ''}<button class="btn pri" data-act="pair">${I('plus')}Pair a Node</button></div>
    ${note ? `<div class="cv-hint cv-ui fx-note">${I('info')}<span>${esc(note)}</span></div>` : ''}
    ${jobTray(app, S.nodes)}${pathCard(app, S.nodes)}`;
  let aside = '';
  if (cfg.view === 'pair') aside = `<aside class="insp cv-insp" aria-label="Pair a Node"><div class="insp-in">${pairingHtml(app)}</div></aside>`;
  else if (n) aside = `<aside class="insp cv-insp" aria-label="${esc(n.name)}"><div class="insp-in"><div class="insp-top"><span class="t3">Node</span><span class="grow"></span><button class="ib" data-go='${JSON.stringify({ node: '' })}' aria-label="Close inspector">${I('x')}</button></div>${header(app, n, defaultActions(app, n, ''))}${detailBody(app, n, '<span class="t3">Roles are edited in D or F</span>')}</div></aside>`;
  const html = wideShell(S.nodes, crumbs({ label: 'Nodes' }), canvasHtml(ID, world(app), ui)).replace('class="cv"', `class="cv fx" data-padb="150" data-padt="116"${aside ? ' data-insp="1"' : ''}${selJob ? ' data-sel="1"' : ''}`) + aside;
  const out = { html, mounted: mounted(ID), ambient: n ? jobsOn(n).find((j) => j.status === 'running')?.work : null };
  if (cfg.view === 'handover') Object.assign(out, { sheet: handoverHtml(app), sheetSize: 'md', sheetClose: { view: 'nodes' } });
  return out;
}

export function modalHtml(app, modal) { return B.modalHtml(app, modal) || A.modalHtml(app, modal); }

export const actions = {
  ...A.actions,
  ...B.actions,
  'e-job': (app, el) => { selJob = selJob === el.dataset.job ? null : el.dataset.job; },
};

let hoverWired = false;
function wireHover(root) {
  if (hoverWired) return;
  hoverWired = true;
  root.addEventListener('mouseover', (e) => {
    const cv = e.target.closest?.('.cv.fx');
    if (!cv) return;
    const h = e.target.closest('[data-hl]');
    const id = h?.dataset.hl || '';
    if (cv.dataset.hl === id) return;
    if (id) cv.dataset.hl = id; else delete cv.dataset.hl;
    cv.querySelectorAll('.fx-st').forEach((s) => s.classList.toggle('hl', s.dataset.stage === id));
    cv.querySelectorAll('.fx-col').forEach((s) => s.classList.toggle('hl', s.dataset.col === id));
  });
}

onTick(() => {
  const cv = document.querySelector('.cv[data-cv="E"]');
  if (!cv) return;
  cv.querySelectorAll('[data-bead]').forEach((b) => {
    const j = jobs.find((x) => x.id === b.dataset.bead);
    const path = [...cv.querySelectorAll('.fx-e.t-now')].find((p) => p.dataset.from.startsWith('download:') && j.node.startsWith(p.dataset.from.split(':')[1]));
    if (!path) return;
    const pt = path.getPointAtLength(path.getTotalLength() * Math.max(0.04, Math.min(0.96, j.progress)));
    b.style.left = `${pt.x}px`; b.style.top = `${pt.y}px`;
  });
});
