// PROTOTYPE · Topology canvas, shared by variant D (structure) and F (structure with live traffic on the links).

import { GROUPS, ROLES, CONTROLLER_ONLY, CONTROLLER_VERSION, jobs, works } from './data.js';
import { esc, I, dot, plural, wideShell, crumbs, nodeState, loadLine, heartbeatText, hm } from './shell.js';
import { controllerOf, jobsOn, cell, toggle, plan, summary, runTest, testFor, blockers } from './model.js';
import { header, defaultActions, detailBody } from './detail.js';
import { pairingHtml, handoverHtml } from './flows.js';
import { onTick, format } from '../ui-direction/live.js';
import * as A from './a.js';
import * as B from './b.js';
import { canvasHtml, mounted, register, wire, rect, ports, curve, posOf, isNarrow, reduceMotion, suppressClick } from './canvas.js';

const OUTSIDE_SERVICE = { 'Media Server link': 'Plex', Notifying: 'Fastmail' };
const targetOf = (n, r) => (r === 'Serving' ? (zoneOf(n) === 'lan' ? 'LAN' : 'internet') : OUTSIDE_SERVICE[r] || '');
function targetTip(n, r) {
  if (r === 'Serving') return zoneOf(n) === 'lan' ? 'Entry point for browsers and apps on the home LAN' : 'Entry point for members over the internet';
  return ROLES[r].reach ? `Reaches ${ROLES[r].reach}` : '';
}

const VISITORS = { 'relay-fra': '2 members on the portal', atlas: '1 browser on the LAN' };

const ZONES = {
  internet: { label: 'Internet', sub: 'public addresses', icon: 'globe' },
  lan: { label: 'Home LAN', sub: '192.168.1.0/24 · behind CGNAT', icon: 'house' },
  cabin: { label: 'Cabin', sub: 'Starlink · CGNAT', icon: 'satellite-dish' },
};
export function zoneOf(n) {
  if (n.where.startsWith('Home')) return 'lan';
  if (n.where.startsWith('Cabin')) return 'cabin';
  return 'internet';
}

const W = 316;
const COL = { a: 0, b: W + 120, c: 2 * W + 240 };
const SLOT = { atlas: ['lan', 'a'], forge: ['lan', 'b'], cellar: ['lan', 'b'], 'seedbox-ams': ['internet', 'a'], 'relay-fra': ['internet', 'c'], cabin: ['cabin', 'c'] };

function chipState(app, n, r, p) {
  const c = cell(app.S.nodes, app.draft, n, r);
  const ch = p.changes.find((x) => x.node === n && x.role === r);
  const t = ch?.kind === 'add' && ROLES[r].reach ? testFor(n, r) : null;
  if (ch?.kind === 'handback') return { cls: 'p-handback', icon: 'corner-down-left', tail: 'auto', tip: `${r} goes back to ${n.name}. It is always held.` };
  if (ch?.kind === 'drain') return { cls: 'p-drain', icon: 'log-out', tail: ch.left ? 'drains' : 'leaves', tip: ch.left ? `Drains: ${ch.left}` : 'Leaves. Nothing in flight.' };
  if (t?.status === 'running') return { cls: 'p-add testing', spin: true, tail: 'testing', tip: t.note };
  if (t?.status === 'fail') return { cls: 'p-add t-fail', icon: 'x', tail: 'unreachable', tip: t.note };
  if (t?.status === 'ok') return { cls: 'p-add', icon: 'check', tail: 'reached', tip: t.note };
  if (ch?.kind === 'add') return { cls: `p-add${c.warn ? ' weak' : ''}`, icon: 'plus', tail: c.warn ? 'weak' : '', tip: c.warn || 'Adding', to: true };
  if (c.draining) return { cls: 'draining', icon: 'log-out', tail: 'draining', tip: `Draining: ${c.draining}` };
  if (!c.now) return null;
  const about = [ROLES[r].blurb, targetTip(n, r)].filter(Boolean).join('. ');
  return { cls: `${c.handedBack ? 'handed' : ''}${c.warn ? ' weak' : ''}`, tip: c.handedBack ? 'Handed back. No other Node holds it.' : c.warn || about, drag: true, to: true };
}

function chip(n, r, s) {
  const attrs = s.drag ? `data-chip="${esc(r)}" data-node="${n.id}" tabindex="0" aria-roledescription="draggable Role"` : `data-role-on="${esc(r)}"`;
  const to = s.to && targetOf(n, r);
  return `<span class="cv-chip ${s.cls}" ${attrs} title="${esc(s.tip)}">${s.spin ? '<span class="spin" aria-hidden="true"></span>' : s.icon ? I(s.icon) : ''}${esc(r)}${to ? `<span class="cv-to">${esc(to)}</span>` : ''}${s.tail ? `<span class="cv-tail">${esc(s.tail)}</span>` : ''}${s.cls.includes('weak') && !s.tail ? dot('warn') : ''}</span>`;
}

function alertStrip(app, n) {
  if (n.state === 'offline') return `<div class="cv-alert fail">${I('unplug')}<span><b>Offline</b> since ${hm(n.lastSeen)}. Portable jobs requeue at ${hm(n.requeueAt)}.</span></div>`;
  if (n.state === 'updating') return `<div class="cv-alert warn">${I('refresh-cw')}<span><b>Updating to ${CONTROLLER_VERSION}</b> · starts nothing new</span><span class="bar cv-up" style="--p:${n.update.progress}"><span class="fill"></span></span></div>`;
  if (n.state === 'behind') return `<div class="cv-alert warn">${I('container')}<span><b>On ${esc(n.version)}.</b> Pull ${CONTROLLER_VERSION} to recreate the container.</span></div>`;
  if (n.state === 'degraded') return `<div class="cv-alert warn">${I('triangle-alert')}<span>${esc(n.warn)}</span></div>`;
  return '';
}

function metaLine(app, n) {
  if (n.kind === 'Controller') {
    const w = app.S.nodes.length - 1;
    return n.single ? 'Every Role in process' : `${plural(w, 'Worker')} pin its key`;
  }
  if (n.state === 'offline') return `<span class="fail-t">${esc(heartbeatText(n))}</span>`;
  return `${I('heart-pulse')}<span class="num">${n.link.heartbeat} s</span><span class="s-sep">·</span><span class="num">${n.link.latency} ms</span>${n.link.relay ? `<span class="s-sep">·</span>via ${esc(n.link.relay)}` : ''}`;
}

function inbound(app, n, p) {
  if (n.state === 'offline') return '';
  const rows = [];
  const swarm = trafficOf(app.S.nodes).filter((f) => f.svc === 'swarm' && f.node === n);
  if (swarm.length) {
    const one = swarm.length === 1 ? swarm[0] : null;
    const k = `in:${n.id}`;
    aggs.set(k, swarm);
    const rate = one ? `<span class="num" data-live-rate="${one.j.id}">${format.rate(one.j)}</span>` : `<span class="num" data-agg="${k}">${aggRate(swarm)}</span>`;
    rows.push(`<div class="cv-in" title="${esc(swarm.map((f) => f.w.title).join('\n'))}">${I('cloud-download')}<span class="ell">From the swarm · ${one ? esc(one.w.title) : plural(swarm.length, 'download')}</span>${rate}</div>`);
  }
  const serving = chipState(app, n, 'Serving', p);
  if (VISITORS[n.id] && serving?.drag) rows.push(`<div class="cv-in">${I('users')}<span class="ell">${VISITORS[n.id]}</span></div>`);
  return rows.join('');
}

function box(app, n, p, pos, traffic) {
  const st = nodeState(n);
  const groups = GROUPS.map((g) => {
    const chips = g.roles.map((r) => { const s = chipState(app, n, r, p); return s ? chip(n, r, s) : ''; }).join('');
    return chips ? `<div class="cv-g"><span class="cv-gl">${g.id}</span><span class="cv-chips">${chips}</span></div>` : '';
  }).join('');
  const sel = app.cfg.node === n.id;
  return `<div class="cv-box st-${n.state}${n.kind === 'Controller' ? ' ctl' : ''}${sel ? ' sel' : ''}" data-box="${n.id}" data-move="${n.id}" data-bound data-go='${JSON.stringify({ node: sel ? '' : n.id, view: 'nodes' })}' role="button" tabindex="0" aria-label="${esc(n.name)}, ${esc(st.label)}. Open details" style="left:${pos.x}px;top:${pos.y}px">
    <div class="cv-bh">${dot(st.k)}<b class="mono">${esc(n.name)}</b>${n.kind === 'Controller' ? `<span class="chip chip-ctl">${I('crown')}Controller</span>` : ''}<span class="grow"></span><span class="num cv-ver ${n.version === CONTROLLER_VERSION ? 't3' : 'warn-t'}">${esc(n.version)}</span></div>
    <div class="cv-bs t3">${esc(n.hw)} · ${esc(n.os)} ${esc(n.arch)}${n.install === 'docker' ? ' · Docker' : ''}</div>
    <div class="cv-bm t3">${metaLine(app, n)}</div>
    ${alertStrip(app, n)}
    <div class="cv-roles">${groups || '<p class="t3 cv-none">No Roles. Drag one here.</p>'}</div>
    ${traffic ? inbound(app, n, p) : ''}
    ${n.kind === 'Controller' ? `<div class="cv-ctl">${I('lock')}<span><b>Controller only</b> ${CONTROLLER_ONLY}</span></div>` : ''}
    <div class="cv-bf num t3">${loadLine(n)}</div>
  </div>`;
}

function place(cv, app, id) {
  const nodes = app.S.nodes;
  const el = (sel) => cv.querySelector(sel);
  const sz = (e) => ({ w: e.offsetWidth, h: e.offsetHeight });
  const slotOf = (n) => SLOT[n.id] || [zoneOf(n), zoneOf(n) === 'lan' ? 'b' : zoneOf(n) === 'cabin' ? 'c' : 'b'];
  const pos = {};
  const row1 = nodes.filter((n) => slotOf(n)[0] === 'internet');
  const r1h = Math.max(0, ...row1.map((n) => sz(el(`[data-box="${n.id}"]`)).h));
  const y1 = 40, y2 = row1.length ? y1 + r1h + 130 : y1;
  const stack = {};
  for (const n of nodes) {
    const [zone, col] = slotOf(n);
    const k = `${zone}|${col}`;
    const y = stack[k] ?? (zone === 'internet' ? y1 : y2);
    pos[n.id] = { x: COL[col], y };
    stack[k] = y + sz(el(`[data-box="${n.id}"]`)).h + 28;
  }
  for (const n of nodes) { const p = posOf(id, n.id, pos[n.id]); const e = el(`[data-box="${n.id}"]`); e.style.left = `${p.x}px`; e.style.top = `${p.y}px`; }
  for (const z of Object.keys(ZONES)) {
    const members = nodes.filter((n) => zoneOf(n) === z).map((n) => el(`[data-box="${n.id}"]`));
    const ze = el(`[data-zone="${z}"]`);
    if (!members.length) { ze.hidden = true; continue; }
    const rs = members.map(rect);
    const x0 = Math.min(...rs.map((r) => r.x)) - 26, y0 = Math.min(...rs.map((r) => r.y)) - 46;
    const x1 = Math.max(...rs.map((r) => r.x + r.w)) + 26, yb = Math.max(...rs.map((r) => r.y + r.h)) + 24;
    Object.assign(ze.style, { left: `${x0}px`, top: `${y0}px`, width: `${x1 - x0}px`, height: `${yb - y0}px` });
    ze.hidden = false;
  }
}

function linkState(n) {
  if (n.state === 'offline') return 'fail';
  if (['updating', 'behind'].includes(n.state)) return 'warn';
  return 'ok';
}

function linkLabel(n, ctl) {
  if (n.state === 'offline') return `no heartbeat since ${hm(n.lastSeen)}`;
  if (n.link.dial === 'relay') return `relayed · ${n.link.latency} ms`;
  return `${n.link.latency} ms`;
}

function edges(cv, app) {
  const nodes = app.S.nodes;
  const ctl = controllerOf(nodes);
  const B = (id) => cv.querySelector(`[data-box="${id}"]`);
  const out = [];
  const seg = (from, to) => ports(rect(from), rect(to));
  for (const n of nodes) {
    if (n === ctl) continue;
    const st = linkState(n);
    if (n.link.dial === 'relay') {
      const relay = nodes.find((m) => m.id === n.link.relay);
      if (!relay) continue;
      const [a, b] = seg(B(n.id), B(relay.id));
      out.push({ kind: 'relay', key: `${n.id}>${relay.id}`, a, b, st, arrow: 'end', label: linkLabel(n, ctl), node: n.id, ends: [n.id, relay.id] });
      const [c, d] = seg(B(ctl.id), B(relay.id));
      const off = (q) => ({ ...q, x: q.x + (q.side === 't' || q.side === 'b' ? 14 : 0), y: q.y + (q.side === 'l' || q.side === 'r' ? 14 : 0) });
      out.push({ kind: 'relay', key: `${ctl.id}>${relay.id}>${n.id}`, a: off(c), b: off(d), st, arrow: 'end', label: '', node: n.id, ends: [ctl.id, relay.id] });
      continue;
    }
    const fromCtl = n.link.dial !== 'worker';
    const [a, b] = fromCtl ? seg(B(ctl.id), B(n.id)) : seg(B(n.id), B(ctl.id));
    out.push({ kind: 'link', key: `${ctl.id}>${n.id}`, a, b, st, arrow: 'end', label: linkLabel(n, ctl), node: n.id, ends: [ctl.id, n.id] });
  }
  return out;
}

function linkPath(nodes, from, to) {
  const ctl = controllerOf(nodes);
  const hop = (a, b) => {
    const w = a === ctl ? b : a;
    if (w.link?.dial === 'relay') { const r = nodes.find((m) => m.id === w.link.relay); return a === ctl ? [[ctl, r], [r, w]] : [[w, r], [r, ctl]]; }
    return [[a, b]];
  };
  if (from === to) return [];
  if (from === ctl || to === ctl) return hop(from, to);
  return [...hop(from, ctl), ...hop(ctl, to)];
}

export function trafficOf(nodes) {
  const byId = (id) => nodes.find((n) => n.id === id) || controllerOf(nodes);
  const storage = nodes.find((n) => n.roles.Storage?.state === 'active') || controllerOf(nodes);
  const flows = [];
  for (const j of jobs.filter((x) => x.status === 'running')) {
    const w = works[j.work];
    if (j.stage === 'Transfer') {
      const [a, b] = j.node.split(' → ').map(byId);
      if (a && b) flows.push({ j, w, hops: linkPath(nodes, a, b), what: 'ships' });
    } else if (j.stage === 'Conversion') {
      const n = byId(j.node);
      if (n && n !== storage) flows.push({ j, w, hops: linkPath(nodes, storage, n), what: 'reads source' });
    } else if (j.stage === 'Downloading') {
      const n = byId(j.node);
      if (n && n.state !== 'offline') flows.push({ j, w, svc: 'swarm', node: n, what: 'downloads' });
    }
  }
  return flows;
}

function trafficSvg(cv, app) {
  const nodes = app.S.nodes;
  const B = (id) => cv.querySelector(`[data-box="${id}"]`);
  const segs = new Map();
  for (const f of trafficOf(nodes)) {
    if (f.svc) continue;
    for (const [from, to] of f.hops) {
      if (!from || !to || from.state === 'offline' || to.state === 'offline') continue;
      const k = `${from.id}>${to.id}`;
      if (!segs.has(k)) {
        const [pa, pb] = ports(rect(B(from.id)), rect(B(to.id)));
        const shift = (q) => ({ ...q, x: q.x + (q.side === 't' || q.side === 'b' ? -14 : 0), y: q.y + (q.side === 'l' || q.side === 'r' ? -14 : 0) });
        segs.set(k, { a: shift(pa), b: shift(pb), jobs: [] });
      }
      segs.get(k).jobs.push(f);
    }
  }
  let svg = '', labels = '';
  const named = new Set();
  const busy = new Set();
  for (const [k, s] of segs) {
    busy.add(k.split('>').sort().join('|'));
    const c = curve(s.a, s.b);
    const m = c.at(0.5);
    svg += `<path class="cv-flow${reduceMotion() ? ' still' : ''}" d="${c.d}" marker-end="url(#cv-arrow-flow)"/>`;
    const fresh = s.jobs.filter((f) => !named.has(f.j.id));
    fresh.forEach((f) => named.add(f.j.id));
    if (!fresh.length) continue;
    const one = s.jobs.length === 1 ? s.jobs[0] : null;
    const rate = one ? `<span class="num" data-live-rate="${one.j.id}">${format.rate(one.j)}</span>` : `<span class="num" data-agg="${esc(k)}">${aggRate(s.jobs)}</span>`;
    labels += `<div class="cv-tl" style="left:${m.x}px;top:${m.y}px" title="${esc(s.jobs.map((f) => `${f.w.title} · ${f.j.stage}`).join('\n'))}">${I(one?.what === 'reads source' ? 'cpu' : 'arrow-right-left')}<span class="ell">${esc(one ? `${one.w.title} ${one.what}` : `${s.jobs.length} jobs`)}</span>${rate}</div>`;
    aggs.set(k, s.jobs);
  }
  return { svg, labels, busy };
}
const aggs = new Map();
function aggRate(fs) {
  const byUnit = {};
  for (const f of fs) byUnit[f.j.unit] = (byUnit[f.j.unit] || 0) + (f.j.rate || 0);
  return Object.entries(byUnit).map(([u, v]) => `${v.toFixed(v < 20 ? 1 : 0)} ${u}`).join(' + ');
}

function draw(cv, app, id, traffic) {
  place(cv, app, id);
  const list = edges(cv, app);
  let svg = `<defs>${['ok', 'warn', 'fail', 'flow'].map((k) => `<marker id="cv-arrow-${k}" viewBox="0 0 10 10" refX="9" refY="5" markerWidth="7" markerHeight="7" orient="auto"><path d="M0,1 L9,5 L0,9 z" class="cv-ah-${k}"/></marker>`).join('')}</defs>`;
  let labels = '';
  const focus = app.cfg.node;
  const t = traffic && cv.dataset.traffic === 'on' ? trafficSvg(cv, app) : null;
  for (const e of list) {
    const c = curve(e.a, e.b);
    const dim = focus && e.node !== focus ? ' dim' : '';
    svg += `<path class="cv-e cv-e-${e.kind} cv-s-${e.st}${dim}" d="${c.d}" ${e.arrow ? `marker-end="url(#cv-arrow-${e.st === 'fail' ? 'fail' : e.st === 'warn' ? 'warn' : 'ok'})"` : ''}/>`;
    const quiet = t && e.ends && t.busy.has([...e.ends].sort().join('|'));
    if (e.label && !quiet) { const m = c.at(0.5); labels += `<div class="cv-el cv-s-${e.st}${dim}" style="left:${m.x}px;top:${m.y}px">${e.kind === 'relay' ? I('git-fork') : I('arrow-right')}${esc(e.label)}</div>`; }
  }
  if (t) { svg += t.svg; labels += t.labels; }
  cv.querySelector('.cv-svg').innerHTML = svg;
  cv.querySelector('.cv-labels').innerHTML = labels;
  window.lucide && window.lucide.createIcons();
}

const drag = { on: false };
let note = '';

function verdict(app, src, tgt, role, add) {
  const nodes = app.S.nodes;
  if (tgt === src) return { k: 'same', note: '' };
  const c = cell(nodes, app.draft, tgt, role);
  if (c.on) return { k: 'same', note: `Already holds ${role}` };
  if (c.lock) return { k: 'blocked', note: c.lock };
  const d = new Map(app.draft);
  toggle(nodes, d, tgt, role);
  if (!add) {
    const cs = cell(nodes, d, src, role);
    if (cs.lock) return { k: 'blocked', note: `${src.name} keeps ${role}. ${cs.lock} Hold ⌥ to add instead.` };
    toggle(nodes, d, src, role);
  }
  const reach = ROLES[role].reach ? ` Reach test to ${ROLES[role].reach} runs on drop.` : '';
  return c.warn ? { k: 'weak', note: `Weak probe. ${c.warn}.${reach}`, d } : { k: 'ok', note: `${c.probe.note}.${reach}`, d };
}

function trashVerdict(app, src, role) {
  const nodes = app.S.nodes;
  const c = cell(nodes, app.draft, src, role);
  if (c.lock) return { k: 'blocked', note: c.lock };
  const d = new Map(app.draft);
  toggle(nodes, d, src, role);
  const hb = plan(nodes, d).changes.find((x) => x.kind === 'handback' && x.role === role);
  if (hb) return { k: 'handback', note: `${role} is always held. With no other holder it goes back to ${hb.node.name}.`, d };
  const left = plan(nodes, d).changes.find((x) => x.kind === 'drain' && x.node === src && x.role === role)?.left;
  return { k: 'ok', note: left ? `${src.name} drains ${role}. ${left}.` : `${src.name} leaves ${role}. Nothing in flight.`, d };
}

const VICON = { ok: 'check', weak: 'triangle-alert', blocked: 'lock', same: 'minus', handback: 'corner-down-left' };

function paintVerdicts(cv, app) {
  const src = app.S.nodes.find((n) => n.id === drag.src);
  cv.querySelectorAll('[data-box]').forEach((b) => {
    const tgt = app.S.nodes.find((n) => n.id === b.dataset.box);
    const v = verdict(app, src, tgt, drag.role, drag.add);
    b.dataset.verdict = v.k;
    b.querySelector('.cv-verdict')?.remove();
    if (v.k !== 'same' || v.note) b.insertAdjacentHTML('afterbegin', `<div class="cv-verdict v-${v.k}">${I(VICON[v.k])}<span>${esc(v.k === 'same' ? v.note : v.k === 'blocked' ? v.note : `${drag.add ? 'Add' : 'Move'} ${drag.role} here. ${v.note}`)}</span></div>`);
  });
  const tv = trashVerdict(app, src, drag.role);
  const t = cv.querySelector('.cv-trash');
  t.dataset.verdict = tv.k;
  t.innerHTML = `${I(VICON[tv.k] === 'check' ? 'trash-2' : VICON[tv.k])}<span><b>${tv.k === 'blocked' ? `Can't remove ${esc(drag.role)}` : `Drop to remove ${esc(drag.role)} from ${esc(src.name)}`}</b><span class="t2">${esc(tv.note)}</span></span>`;
  window.lucide && window.lucide.createIcons();
}

function commit(app, d, tgt, role) {
  app.draft.clear();
  for (const [k, v] of d) app.draft.set(k, v);
  if (tgt && ROLES[role].reach) runTest(tgt, role, () => app.refresh());
}

function wireDrag(app) {
  app.root.addEventListener('pointerdown', (e) => {
    const c = e.target.closest('.cv [data-chip]');
    if (!c || e.button !== 0) return;
    e.preventDefault();
    const cv = c.closest('.cv');
    const sx = e.clientX, sy = e.clientY;
    Object.assign(drag, { on: false, src: c.dataset.node, role: c.dataset.chip, add: e.altKey, over: null });
    let ghost;
    const move = (ev) => {
      if (!drag.on && Math.hypot(ev.clientX - sx, ev.clientY - sy) < 4) return;
      const r = cv.getBoundingClientRect();
      if (!drag.on) {
        drag.on = true;
        cv.classList.add('dragging');
        c.classList.add('lifted');
        ghost = document.createElement('div');
        ghost.className = 'cv-ghost';
        cv.append(ghost);
      }
      if (ev.altKey !== drag.add || !ghost.firstChild) {
        drag.add = ev.altKey;
        paintVerdicts(cv, app);
        ghost.innerHTML = `${I(drag.add ? 'copy-plus' : 'move')}${esc(drag.role)}<span class="t3">${drag.add ? 'add' : 'move'}</span>`;
        window.lucide && window.lucide.createIcons();
      }
      ghost.style.left = `${ev.clientX - r.left + 12}px`;
      ghost.style.top = `${ev.clientY - r.top + 10}px`;
      const hit = document.elementFromPoint(ev.clientX, ev.clientY);
      const over = hit?.closest('[data-box], .cv-trash');
      cv.querySelectorAll('.drop-over').forEach((x) => x !== over && x.classList.remove('drop-over'));
      if (over) over.classList.add('drop-over');
      drag.over = over ? (over.dataset.box || 'trash') : null;
      ghost.dataset.verdict = over?.dataset.verdict || '';
    };
    const up = () => {
      removeEventListener('pointermove', move); removeEventListener('pointerup', up);
      if (!drag.on) return;
      drag.on = false;
      ghost?.remove();
      const nodes = app.S.nodes;
      const src = nodes.find((n) => n.id === drag.src);
      const role = drag.role;
      const finish = (msg) => { note = msg; cv.classList.remove('dragging'); suppressClick(); app.refresh(); };
      if (!drag.over) return clearDrag(cv);
      if (drag.over === 'trash') {
        const v = trashVerdict(app, src, role);
        if (v.k === 'blocked') return refuse(app, cv, c, v.note);
        commit(app, v.d, null, role);
        return finish(v.note);
      }
      const tgt = nodes.find((n) => n.id === drag.over);
      const v = verdict(app, src, tgt, role, drag.add);
      if (v.k === 'same') { cv.classList.remove('dragging'); return app.refresh(); }
      if (v.k === 'blocked') return refuse(app, cv, cv.querySelector(`[data-box="${tgt.id}"]`), v.note);
      commit(app, v.d, tgt, role);
      finish(`${drag.add ? `${tgt.name} adds` : `${role} moves to ${tgt.name}`}${drag.add ? ` ${role}` : ''}.${v.k === 'weak' ? ` ${v.note}` : ''}`);
    };
    addEventListener('pointermove', move); addEventListener('pointerup', up);
  });
}

function clearDrag(cv) {
  cv.classList.remove('dragging');
  cv.querySelectorAll('.lifted').forEach((x) => x.classList.remove('lifted'));
  cv.querySelectorAll('.cv-verdict').forEach((x) => x.remove());
  cv.querySelectorAll('[data-verdict]').forEach((x) => { delete x.dataset.verdict; });
}

function refuse(app, cv, el, msg) {
  clearDrag(cv);
  el.classList.add('refused');
  setTimeout(() => el.classList.remove('refused'), 600);
  app.toast(msg.replace(/\.$/, ''));
}

function pendingBar(app) {
  const p = plan(app.S.nodes, app.draft);
  if (!p.changes.length) { note = ''; return ''; }
  const bl = blockers(p);
  const testing = bl.some((c) => testFor(c.node, c.role)?.status === 'running');
  const failed = bl.find((c) => testFor(c.node, c.role)?.status === 'fail');
  const msg = failed ? `<span class="fail-t">${esc(failed.node.name)} can't reach ${esc(ROLES[failed.role].reach)}. Drag it off to apply.</span>` : testing ? '<span class="t2"><span class="spin" aria-hidden="true"></span>Testing reach before saving</span>' : note ? `<span class="t2">${esc(note)}</span>` : '';
  return `<div class="pbar cv-pbar" role="region" aria-label="Pending changes"><b class="num">${summary(p)}</b>${msg ? `<span class="pbar-n">${msg}</span>` : ''}<span class="grow"></span>
    <button class="btn ghost sm" data-act="cv-discard">Discard</button><button class="btn sm" data-act="b-review">Review</button><button class="btn pri sm" data-act="b-apply" ${bl.length ? 'aria-disabled="true"' : ''}>Apply</button></div>`;
}

function legend(traffic, relayed) {
  return `<div class="cv-legend cv-ui t3">
    <span><svg width="28" height="8"><path class="cv-e cv-e-link cv-s-ok" d="M1,4 H22" marker-end="url(#cv-arrow-ok)"/></svg>Node link, arrow from the Node that dials</span>
    ${relayed ? `<span><svg width="28" height="8"><path class="cv-e cv-e-relay cv-s-ok" d="M1,4 H22"/></svg>Relayed through ${esc(relayed)}</span>` : ''}
    ${traffic ? '<span><svg width="28" height="8"><path class="cv-flow still" d="M1,4 H22"/></svg>Live traffic</span>' : ''}
  </div>`;
}

function layoutTabs(app) {
  const tab = (layout, label, icon) => {
    const on = (app.cfg.layout || '') === layout;
    return `<button role="tab" aria-selected="${on}" class="${on ? 'on' : ''}" data-go='${JSON.stringify({ layout, node: '' })}'>${I(icon)}${label}</button>`;
  };
  return `<div class="cv-seg nd-tabs" role="tablist" aria-label="Layout">${tab('', 'Diagram', 'waypoints')}${tab('roster', 'Roster', 'list')}</div>`;
}

export function topology({ id, name, traffic }) {
  let wiredDrag = false;
  let trafficOn = traffic;

  function canvas(app) {
    const { S } = app;
    const p = plan(S.nodes, app.draft);
    const ctl = controllerOf(S.nodes);
    const solo = S.nodes.length === 1;
    const world = `${Object.entries(ZONES).map(([z, d]) => `<div class="cv-zone z-${z}" data-zone="${z}" data-bound hidden><span class="cv-zl">${I(d.icon)}<b>${d.label}</b><span class="t3">${d.sub}</span></span></div>`).join('')}
      <svg class="cv-svg" aria-hidden="true"></svg>
      ${S.nodes.map((n) => box(app, n, p, { x: 0, y: 0 }, traffic && trafficOn)).join('')}
      <div class="cv-labels"></div>`;
    const off = S.nodes.filter((n) => n.state === 'offline').length;
    const warn = S.nodes.filter((n) => ['updating', 'behind', 'degraded'].includes(n.state)).length;
    const title = `<div class="cv-title cv-ui"><div class="nd-h1"><h1>Nodes</h1>${layoutTabs(app)}</div><p class="t2">${plural(S.nodes.length, 'Node')}${solo ? ' · one Controller' : ` · Controller ${esc(ctl.name)}`}${off ? ` · <span class="fail-t">${off} offline</span>` : ''}${warn ? ` · <span class="warn-t">${warn} need attention</span>` : ''}</p></div>`;
    const acts = `<div class="cv-acts cv-ui">${traffic ? `<button class="btn ${trafficOn ? 'tog-on' : ''}" data-act="cv-traffic" aria-pressed="${trafficOn}">${I('activity')}Traffic</button>` : ''}${solo ? '' : `<button class="btn" data-act="handover">${I('crown')}Hand over</button>`}<button class="btn pri" data-act="pair">${I('plus')}Pair a Node</button></div>`;
    const hint = `<div class="cv-hint cv-ui t3">${solo ? 'One Node runs every Role in process. Pair a Node to spread Roles across machines.' : 'Drag a Role chip to another Node to move it · hold ⌥ to add · drag empty canvas to pan'}</div>`;
    const trash = '<div class="cv-trash cv-ui"></div>';
    return canvasHtml(id, world, title + acts + hint + (solo ? '' : legend(traffic, S.nodes.find((n) => n.link?.relay)?.link.relay)) + trash);
  }

  function render(app) {
    const narrow = isNarrow(app);
    app.root.classList.toggle('nd-roster', narrow || app.cfg.layout === 'roster');
    if (narrow) return A.render(app);
    if (app.cfg.layout === 'roster') return A.render(app, layoutTabs(app));
    wire(app.root);
    if (!wiredDrag) { wireDrag(app); wiredDrag = true; }
    register(id, { draw: (cv) => draw(cv, app, id, traffic) });
    const { cfg, S } = app;
    const n = S.nodes.find((m) => m.id === cfg.node);
    app.flowClose = `<button class="ib" data-go='${JSON.stringify({ view: 'nodes' })}' aria-label="Close">${I('x')}</button>`;
    let aside = '';
    if (cfg.view === 'pair') aside = `<aside class="insp cv-insp" aria-label="Pair a Node"><div class="insp-in">${pairingHtml(app)}</div></aside>`;
    else if (n) {
      const close = `<button class="ib" data-go='${JSON.stringify({ node: '' })}' aria-label="Close inspector">${I('x')}</button>`;
      aside = `<aside class="insp cv-insp" aria-label="${esc(n.name)}"><div class="insp-in"><div class="insp-top"><span class="t3">Node</span><span class="grow"></span>${close}</div>${header(app, n, defaultActions(app, n, ''))}${detailBody(app, n, '<span class="t3">Drag chips on the canvas</span>')}</div></aside>`;
    }
    const html = wideShell(S.nodes, crumbs({ label: 'Nodes' }), canvas(app)).replace('class="cv"', `class="cv" data-padt="112" data-padb="124"${aside ? ' data-insp="1"' : ''} data-traffic="${trafficOn ? 'on' : 'off'}"`) + aside + pendingBar(app);
    const out = { html, mounted: mounted(id), ambient: n ? jobsOn(n).find((j) => j.status === 'running')?.work : null };
    if (cfg.view === 'handover') Object.assign(out, { sheet: handoverHtml(app), sheetSize: 'md', sheetClose: { view: 'nodes' } });
    return out;
  }

  function modalHtml(app, modal) {
    return B.modalHtml(app, modal) || A.modalHtml(app, modal);
  }

  const actions = {
    ...A.actions,
    ...B.actions,
    'cv-discard': (app) => { app.draft.clear(); note = ''; },
    'cv-traffic': () => { trafficOn = !trafficOn; },
  };

  if (traffic) onTick(() => {
    document.querySelectorAll('.cv [data-agg]').forEach((el) => { const fs = aggs.get(el.dataset.agg); if (fs) el.textContent = aggRate(fs); });
  });

  return { name, render, modalHtml, actions };
}
