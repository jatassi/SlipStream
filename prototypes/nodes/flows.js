// PROTOTYPE. Flows shared by all variants: Pairing (Controller side), Handover, Remove and Force confirms,
// plus the two pages outside the admin shell: the new Node's first-run page and a Serving Worker's status page.

import { CONTROLLER_VERSION, CONTROLLER_ADDRS, JOIN_CODE, IMAGE, ROLES, GROUPS, ROLE_NAMES, CONTROLLER_ONLY, joiningNode, probe, now, torrents, works } from './data.js';
import { esc, I, dot, hm, plural, workRow, itemLabel, mark, statePill } from './shell.js';
import { controllerOf, jobsOn, runTest, testFor, roleOfJob } from './model.js';
import { handoverBlock } from './detail.js';

const CODE = JOIN_CODE.slice(0, 13);
const CODE_STR = CODE.join('-');
const fpShort = (fp) => fp.replace('SHA256 ', '');

// ═════════════ Pairing, Controller side

export const pairing = { step: 'issued', how: 'page', expiresAt: 0, issuedAt: null, node: null, roles: new Set() };
export function startPairing() {
  Object.assign(pairing, { step: 'issued', how: 'page', expiresAt: Date.now() + (59 * 60 + 48) * 1000, issuedAt: new Date(now), node: null, roles: new Set() });
}
const clock = (ms) => { const s = Math.max(0, Math.ceil(ms / 1000)); return `${String(Math.floor(s / 60)).padStart(2, '0')}:${String(s % 60).padStart(2, '0')}`; };
const expiryClock = () => hm(new Date(now.getTime() + 60 * 60 * 1000).toISOString());

function codeBlock(dim) {
  return `<div class="jc${dim ? ' jc-dim' : ''}" role="group" aria-label="Join Code">
    <span class="jc-pre mono">${CODE[0]}</span>
    <span class="jc-grid mono">${CODE.slice(1).map((g, i) => `<span class="jc-g"><span class="jc-i">${i + 1}</span>${g}</span>`).join('')}</span>
  </div>`;
}

const HOW = {
  page: { label: 'First-run page', icon: 'monitor', body: (n) => `<p>Install SlipStream on the new machine and open it in a browser. Its first page asks for a Join Code. Paste this one.</p><p class="t3">The page is on port 7878 until the machine joins, for example <span class="mono">http://cellar.local:7878</span>.</p>` },
  cli: { label: 'Command line', icon: 'terminal', body: () => `<div class="codeline"><code class="mono">slipstream --join ${CODE_STR}</code><button class="btn sm" data-act="copy" data-copy="slipstream --join ${CODE_STR}">${I('copy')}Copy</button></div><p class="t3">Run it once on the new machine. It joins and keeps running.</p>` },
  docker: { label: 'Docker', icon: 'container', body: () => `<div class="codeline"><code class="mono">docker run -d -e SLIPSTREAM_JOIN_CODE=${CODE_STR} ${IMAGE}:${CONTROLLER_VERSION}</code><button class="btn sm" data-act="copy" data-copy="SLIPSTREAM_JOIN_CODE=${CODE_STR}">${I('copy')}Copy</button></div><p class="t3">Or set <span class="mono">SLIPSTREAM_JOIN_CODE</span> in your compose file. The variable is read once, on first start.</p>` },
};

function pairIssued(app) {
  const ctl = controllerOf(app.S.nodes);
  return `<div class="fl-body">
    <p class="fl-lede">Use this Join Code on the new machine. It works once, for one Node, for one hour.</p>
    ${codeBlock(false)}
    <div class="jc-meta"><span class="jc-timer">${dot('draft')}Valid for <b class="num" data-countdown>${clock(pairing.expiresAt - Date.now())}</b></span><span class="grow"></span>
      <button class="btn sm" data-act="copy" data-copy="${CODE_STR}">${I('copy')}Copy</button><button class="btn sm ghost" data-act="pair-revoke">${I('ban')}Revoke</button></div>
    <p class="t3 jc-carries">The code carries ${esc(ctl.name)}'s addresses (<span class="mono">${CONTROLLER_ADDRS.join(', ')}</span>) and its key fingerprint <span class="mono">${esc(fpShort(ctl.link.fingerprint))}</span>, so the new Node can find and check ${esc(ctl.name)}.</p>
    <div class="fl-sub"><h3>Three ways to use it</h3>
      <div class="seg-c" role="radiogroup" aria-label="How to use the code">${Object.entries(HOW).map(([k, h]) => `<button class="seg-o" role="radio" aria-checked="${pairing.how === k}" data-act="pair-how" data-how="${k}">${I(h.icon)}${h.label}</button>`).join('')}</div>
      <div class="how">${HOW[pairing.how].body()}</div>
    </div>
    <p class="fl-wait">${dot('draft')}<span>Waiting for a Node to use this code. Keep this open, or close it and the code still works until ${expiryClock()}.</span></p>
  </div>`;
}

function dialRows(n, ctl) {
  const opts = [
    ['controller', `${ctl.name} dials ${n.name}`, `${n.name} answers on ${n.addr} from the LAN`],
    ['worker', `${n.name} dials ${ctl.name}`, 'Also reachable. Not needed'],
    ['relay', 'Both dial a relaying Node', 'Not needed. Used only when neither side can reach the other'],
  ];
  return `<ul class="dials">${opts.map(([k, l, d]) => `<li class="${k === n.link.dial ? 'on' : ''}">${k === n.link.dial ? I('circle-check') : I('circle')}<span><b>${esc(l)}</b><span class="t3">${esc(d)}</span></span>${k === n.link.dial ? '<span class="chip chip-live">Detected</span>' : ''}</li>`).join('')}</ul>`;
}

function pairJoined(app) {
  const n = pairing.node, ctl = controllerOf(app.S.nodes);
  return `<div class="fl-body">
    <p class="fl-lede"><b class="mono">${esc(n.name)}</b> used the code at <span class="num">${hm(now.toISOString())}</span>. Check its key before you trust it.</p>
    <div class="fp-cmp">
      <p class="t2">${esc(n.name)} shows this fingerprint on its own screen. They should match.</p>
      <p class="fp-big mono">${fpShort(n.link.fingerprint).split(' ').map((g) => `<span>${g}</span>`).join('')}</p>
    </div>
    <dl class="kvl kv-tight">
      <dt>Machine</dt><dd>${esc(n.hw)} <span class="t3">· ${esc(n.os)} · ${esc(n.arch)}</span></dd>
      <dt>Version</dt><dd><span class="num">${n.version}</span> <span class="t3">· same as ${esc(ctl.name)}</span></dd>
      <dt>Disk</dt><dd>${esc(n.disk)}</dd>
    </dl>
    <div class="fl-sub"><h3>Who dials</h3>${dialRows(n, ctl)}</div>
    <div class="fl-sub"><h3>Probes</h3><ul class="plist plist-2">${ROLE_NAMES.map((r) => { const p = probe(n.id, r); return `<li><span class="${p.v === 'ok' ? 'ok-t' : p.v === 'weak' ? 'warn-t' : 'fail-t'}">${I(p.v === 'ok' ? 'check' : p.v === 'weak' ? 'triangle-alert' : 'lock')}</span><b>${esc(r)}</b><span class="${p.v === 'ok' ? 't3' : p.v === 'weak' ? 'warn-t' : 'fail-t'} ell">${esc(p.note)}</span></li>`; }).join('')}</ul></div>
  </div>
  <div class="fl-foot"><button class="btn ghost danger-t" data-act="pair-reject">Keys differ, reject</button><span class="grow"></span><button class="btn pri" data-act="pair-match">${I('check')}Keys match</button></div>`;
}

function pairRoleCell(n, r) {
  const p = probe(n.id, r), on = pairing.roles.has(r), t = testFor(n, r);
  const locked = p.v === 'blocked';
  let note = '';
  if (locked) note = `<span class="fail-t">${esc(p.note)}</span>`;
  else if (ROLES[r].reach && on) note = !t ? '' : t.status === 'running' ? `<span class="t2"><span class="spin" aria-hidden="true"></span>${esc(t.note)}</span>` : `<span class="${t.status === 'ok' ? 'ok-t' : 'fail-t'}">${I(t.status === 'ok' ? 'check' : 'x')}${esc(t.note)}</span>`;
  else if (p.v === 'weak') note = `<span class="warn-t">${esc(p.note)}</span>`;
  else if (ROLES[r].reach) note = `<span class="t3">Tested from ${esc(n.name)} before saving</span>`;
  else if (n.suggested.includes(r)) note = '<span class="t3">Suggested by probes</span>';
  return `<li class="prc${locked ? ' locked' : ''}"><button class="tick${on ? ' on' : ''}" role="checkbox" aria-checked="${on}" ${locked ? 'aria-disabled="true"' : ''} data-act="pair-role" data-role="${esc(r)}" aria-label="${esc(r)}">${locked ? I('lock') : on ? I('check') : ''}</button><span class="prc-t"><b>${esc(r)}</b>${p.v === 'weak' && !locked ? dot('warn') : ''}${note}</span></li>`;
}

function pairRoles(app) {
  const n = pairing.node;
  const testing = [...pairing.roles].some((r) => ROLES[r].reach && testFor(n, r)?.status !== 'ok');
  return `<div class="fl-body">
    <p class="fl-lede">Pick the Roles ${esc(n.name)} holds. Probes pre-ticked the ones it is good at. You can change them any time.</p>
    <div class="prg">${GROUPS.map((g) => `<div class="prg-g"><h3>${g.id}</h3><ul>${g.roles.map((r) => pairRoleCell(n, r)).join('')}</ul></div>`).join('')}</div>
  </div>
  <div class="fl-foot"><button class="btn ghost" data-act="pair-back">Back</button><span class="grow"></span><span class="t3 num">${plural(pairing.roles.size, 'Role')}</span><button class="btn pri" data-act="pair-finish" ${testing ? 'aria-disabled="true" title="Wait for the reach tests to pass"' : ''}>Finish</button></div>`;
}

function pairDead(app, kind) {
  const msg = {
    expired: [`This Join Code expired at ${expiryClock()}.`, 'A Join Code lasts one hour. Nothing joined with it.'],
    used: ['This Join Code was already used.', `cellar joined with it at ${hm(now.toISOString())}. A Node at 10.0.4.17 tried it after that and was turned away. Each code pairs one Node.`],
    revoked: ['You revoked this Join Code.', 'No Node can use it.'],
  }[kind];
  return `<div class="fl-body">
    ${codeBlock(true)}
    <div class="fl-dead"><span class="chip-${kind === 'used' ? 'warn' : 'fail'}">${I(kind === 'expired' ? 'timer-off' : kind === 'used' ? 'user-check' : 'ban')}${kind === 'expired' ? 'Expired' : kind === 'used' ? 'Used' : 'Revoked'}</span><p><b>${msg[0]}</b> <span class="t2">${msg[1]}</span></p></div>
  </div>
  <div class="fl-foot"><span class="grow"></span><button class="btn pri" data-act="pair-new">${I('refresh-cw')}Issue a new code</button></div>`;
}

export function pairingHtml(app) {
  const steps = ['Code', 'Check key', 'Roles'];
  const at = { issued: 0, expired: 0, used: 0, revoked: 0, joined: 1, roles: 2 }[pairing.step];
  const head = `<div class="fl-head"><div><h2>Pair a Node</h2><ol class="fl-steps">${steps.map((s, i) => `<li class="${i < at ? 'done' : i === at ? 'cur' : ''}">${i < at ? I('check') : `<span class="num">${i + 1}</span>`}${s}</li>`).join('')}</ol></div><span class="grow"></span>${app.flowClose || ''}</div>`;
  const body = { issued: pairIssued, joined: pairJoined, roles: pairRoles, expired: (a) => pairDead(a, 'expired'), used: (a) => pairDead(a, 'used'), revoked: (a) => pairDead(a, 'revoked') }[pairing.step](app);
  return `<div class="flow" data-flow="pair">${head}${body}</div>`;
}

export function pairSim(app, what) {
  if (what === 'join') {
    if (pairing.step !== 'issued') startPairing();
    pairing.node = joiningNode();
    pairing.roles = new Set(pairing.node.suggested);
    pairing.step = 'joined';
  }
  if (what === 'expire') pairing.step = 'expired';
  if (what === 'used') pairing.step = 'used';
}

export const pairActions = {
  'pair-revoke': (app) => { pairing.step = 'revoked'; app.toast('Join Code revoked. No Node can use it'); },
  'pair-new': () => startPairing(),
  'pair-how': (app, el) => { pairing.how = el.dataset.how; },
  'pair-reject': (app) => { startPairing(); app.toast('Rejected cellar. Its key is not trusted. A fresh code is ready'); },
  'pair-match': () => { pairing.step = 'roles'; },
  'pair-back': () => { pairing.step = 'joined'; },
  'pair-role': (app, el) => {
    const r = el.dataset.role, n = pairing.node;
    if (probe(n.id, r).v === 'blocked') return app.toast(`${r} is blocked on ${n.name}. ${probe(n.id, r).note}`);
    if (pairing.roles.has(r)) pairing.roles.delete(r);
    else { pairing.roles.add(r); if (ROLES[r].reach) runTest(n, r, () => app.refresh()); }
  },
  'pair-finish': (app) => {
    const n = pairing.node;
    if ([...pairing.roles].some((r) => ROLES[r].reach && testFor(n, r)?.status !== 'ok')) return app.toast('Wait for the reach tests to pass');
    n.roles = Object.fromEntries([...pairing.roles].map((r) => [r, { state: 'active' }]));
    app.S.nodes.push(n);
    const held = [...pairing.roles];
    pairing.step = 'issued';
    pairing.node = null;
    app.toast(`${n.name} paired${held.length ? `. It holds ${held.join(', ')}` : '. It holds no Roles yet'}`);
    app.go({ view: 'nodes', node: n.id });
  },
};

export function tickCountdown(app) {
  const el = app.root.querySelector('[data-countdown]');
  if (!el) return;
  const left = pairing.expiresAt - Date.now();
  el.textContent = clock(left);
  if (left <= 0 && pairing.step === 'issued') { pairing.step = 'expired'; app.refresh(); }
}

// ═════════════ Handover

export const handover = { step: 'pick', to: null, pin: '', run: 0, sub: 0, timer: null };
export function startHandover(toId) { clearInterval(handover.timer); Object.assign(handover, { step: toId ? 'preflight' : 'pick', to: toId || null, pin: '', run: 0, sub: 0, timer: null }); }

const lan = (n) => n.where.startsWith('Home');
const canDial = (a, b) => !lan(b) || lan(a);
function reachRow(to, w) {
  if (w.state === 'offline') return { v: 'warn', t: `${w.name} is offline. The old Controller redirects it when it reconnects.` };
  if (canDial(to, w)) return { v: 'ok', t: `${to.name} dials ${w.name}${lan(to) && lan(w) ? ' on the LAN' : ` at ${w.addr}`}` };
  if (canDial(w, to)) return { v: 'ok', t: `${w.name} dials ${to.name} at ${to.addr}` };
  return { v: 'fail', t: `Neither side can reach the other. Add Relaying to a Node both can reach.` };
}

function hoPick(app) {
  const ctl = controllerOf(app.S.nodes);
  const ws = app.S.nodes.filter((n) => n.kind === 'Worker');
  if (!ws.length) {
    return `<div class="fl-body"><div class="fl-empty">${I('network')}<p><b>There is no other Node to hand over to.</b> <span class="t2">A Handover moves the Controller to a paired Node. Pair one first.</span></p><button class="btn pri" data-act="pair">${I('plus')}Pair a Node</button></div>${deadLink()}</div>`;
  }
  return `<div class="fl-body">
    <p class="fl-lede">Pick the Node that becomes the Controller. ${esc(ctl.name)} becomes a Worker and keeps its Roles.</p>
    <ul class="ho-pick" role="radiogroup" aria-label="Receiving Node">${ws.map((n) => {
      const block = handoverBlock(n, app.S.nodes);
      const sel = handover.to === n.id;
      return `<li><button class="ho-o${sel ? ' on' : ''}" role="radio" aria-checked="${sel}" ${block ? 'aria-disabled="true"' : ''} data-act="ho-to" data-node="${n.id}">
        <span class="ho-radio" aria-hidden="true"></span>
        <span class="ho-id"><b class="mono">${esc(n.name)}</b><span class="t3">${esc(n.hw)} · ${esc(n.where)} · <span class="num">${esc(n.version)}</span></span></span>
        ${block ? `<span class="ho-why warn-t">${esc(block)}</span>` : statePill(n)}
      </button></li>`;
    }).join('')}</ul>
    <p class="t3">Only paired, online Nodes on ${CONTROLLER_VERSION} can receive the Controller.</p>
    ${deadLink()}
  </div>
  <div class="fl-foot"><span class="grow"></span><button class="btn pri" data-act="ho-next" ${handover.to ? '' : 'aria-disabled="true"'}>Check ${handover.to ? esc(handover.to) : 'Node'}</button></div>`;
}
const deadLink = () => `<button class="lnk ho-dead" data-act="ho-dead">${I('life-buoy')}<span>The Controller's machine is dead?</span></button>`;

const V = { ok: ['check', 'ok-t'], warn: ['triangle-alert', 'warn-t'], fail: ['x', 'fail-t'] };
function hoChecks(app) {
  const ctl = controllerOf(app.S.nodes), to = app.S.nodes.find((n) => n.id === handover.to);
  const others = app.S.nodes.filter((n) => n !== to);
  const behind = others.filter((n) => n.version !== CONTROLLER_VERSION);
  return [
    { h: 'Version lockstep', rows: [{ v: 'ok', t: `${to.name} and ${ctl.name} both run ${CONTROLLER_VERSION}` }, ...behind.map((n) => ({ v: 'warn', t: `${n.name} runs ${n.version}. It updates from ${to.name} afterwards.` }))] },
    { h: `Every Worker can reach ${to.name}`, rows: others.map((w) => reachRow(to, w)) },
    { h: 'Disk for the snapshot', rows: [{ v: 'ok', t: `Snapshot is about 1.8 GB in the backup format. ${to.name} has ${to.disk}.` }] },
    { h: 'What moves and what stays', rows: [
      { v: 'ok', t: `Moves to ${to.name}: ${CONTROLLER_ONLY}.` },
      { v: 'ok', t: `${to.name} starts Serving, because the Controller always serves.` },
      { v: 'ok', t: `Every Role stays on its Node. ${ctl.name} keeps Storage, so it still imports into ${ctl.libraryRoot}.` },
    ] },
  ];
}
function hoPreflight(app) {
  const checks = hoChecks(app);
  const bad = checks.some((c) => c.rows.some((r) => r.v === 'fail'));
  return `<div class="fl-body">
    <p class="fl-lede">Preflight for <b class="mono">${esc(handover.to)}</b>. Writes freeze for about 2 minutes. Jobs keep running and report to the new Controller.</p>
    ${checks.map((c) => `<div class="pf"><h3>${esc(c.h)}</h3><ul>${c.rows.map((r) => `<li class="${V[r.v][1]}">${I(V[r.v][0])}<span class="${r.v === 'ok' ? 't1' : ''}">${esc(r.t)}</span></li>`).join('')}</ul></div>`).join('')}
  </div>
  <div class="fl-foot"><button class="btn ghost" data-act="ho-back">Back</button><span class="grow"></span><button class="btn pri" data-act="ho-confirm" ${bad ? 'aria-disabled="true"' : ''}>Continue</button></div>`;
}

function hoConfirm() {
  return `<div class="fl-body fl-pin">
    <span class="pin-ic">${I('shield-check')}</span>
    <p class="fl-lede">Confirm it's you. A Handover is an admin action, so it needs your PIN again.</p>
    <label class="pin"><span class="sr">PIN</span><input class="pin-in mono" data-pin inputmode="numeric" autocomplete="one-time-code" maxlength="4" value="${esc(handover.pin)}" aria-label="4-digit PIN"><span class="pin-boxes" aria-hidden="true">${[0, 1, 2, 3].map((i) => `<span class="${i < handover.pin.length ? 'f' : ''}${i === handover.pin.length ? ' c' : ''}"></span>`).join('')}</span></label>
    <button class="btn ghost sm" data-toast="Passkeys are not in this prototype">${I('key-round')}Use a passkey</button>
  </div>
  <div class="fl-foot"><button class="btn ghost" data-act="ho-back">Back</button><span class="grow"></span><span class="t3">Five wrong PINs bring a 15 minute wait</span></div>`;
}

function hoSteps(app) {
  const to = app.S.nodes.find((n) => n.id === handover.to), ctl = app.S.nodes.find((n) => n.id === handover.from) || controllerOf(app.S.nodes);
  const ws = app.S.nodes.filter((n) => n !== to);
  return [
    { t: 'Freeze writes', d: `${ctl.name} stops taking changes. Jobs keep running.` },
    { t: 'Snapshot in the backup format', d: '1.8 GB', bar: true },
    { t: `Ship to ${to.name}`, d: `${ctl.name} to ${to.name} · BLAKE3 verified`, bar: true },
    { t: `Load on ${to.name}`, d: `${to.name} becomes the Controller` },
    { t: 'Signed notices to every Worker', d: `Signed with ${ctl.name}'s key, naming ${to.name}'s key and addresses`, list: ws.map((w) => (w.state === 'offline' ? { n: w.name, off: true } : { n: w.name, t: w === ctl ? 'now a Worker' : 'follows' })) },
  ];
}
function hoRun(app) {
  const steps = hoSteps(app);
  return `<div class="fl-body"><ol class="run">${steps.map((s, i) => {
    const st = i < handover.run ? 'done' : i === handover.run ? 'cur' : 'todo';
    const prog = s.bar && st === 'cur' ? `<div class="bar run-bar" style="--p:${handover.sub}"><span class="fill"></span></div>` : '';
    const list = s.list && st !== 'todo' ? `<ul class="notices">${s.list.map((w, k) => { const ok = st === 'done' || k < handover.sub * s.list.length; return `<li class="${w.off ? 'warn-t' : ok ? '' : 't3'}">${w.off ? I('clock') : ok ? I('check') : '<span class="spin" aria-hidden="true"></span>'}<b class="mono">${esc(w.n)}</b><span class="${w.off ? 'warn-t' : 't3'}">${w.off ? 'offline. It will be redirected when it reconnects.' : ok ? w.t : 'sending'}</span></li>`; }).join('')}</ul>` : '';
    return `<li class="run-i ${st}"><span class="run-dot">${st === 'done' ? I('check') : st === 'cur' ? '<span class="spin" aria-hidden="true"></span>' : `<span class="num">${i + 1}</span>`}</span><div><b>${esc(s.t)}</b><span class="t3">${esc(s.d)}</span>${prog}${list}</div></li>`;
  }).join('')}</ol></div>
  <div class="fl-foot"><span class="t3">${dot('draft')}Running. Leave this open.</span></div>`;
}
function hoDone(app) {
  const to = app.S.nodes.find((n) => n.id === handover.to), old = app.S.nodes.find((n) => n.id === handover.from);
  return `<div class="fl-body fl-done">
    <span class="done-ic">${I('crown')}</span>
    <h3><span class="mono">${esc(to.name)}</span> is the Controller</h3>
    <p class="t2">Open SlipStream at its address from now on.</p>
    <div class="codeline"><code class="mono">http://${esc(to.name)}.local:7878</code><button class="btn sm" data-act="copy" data-copy="http://${esc(to.name)}.local:7878">${I('copy')}Copy</button></div>
    <p class="t3"><span class="mono">${esc(old.name)}</span> is a Worker now and keeps its Roles. Its old address forwards here for a while.</p>
  </div>
  <div class="fl-foot"><span class="grow"></span><button class="btn" data-act="close-flow">Close</button><button class="btn pri" data-toast="Opening the new Controller is not in this prototype">Open ${esc(to.name)}</button></div>`;
}
function hoDead(app) {
  const ctl = controllerOf(app.S.nodes);
  return `<div class="fl-body">
    <p class="fl-lede">A Handover needs the old Controller running. If ${esc(ctl.name)}'s machine is gone, restore a backup on a new Node instead.</p>
    <ol class="guide">
      <li><b>Install SlipStream on the new machine.</b><span class="t2">Choose Restore a backup on its first page.</span></li>
      <li><b>Pick your latest backup.</b><span class="t2">The last one ran today at <span class="num">03:00</span>, <span class="num">1.7 GB</span>. Backups are listed in Settings, under Backups, and wherever you copied them.</span></li>
      <li><b>The restored Node becomes the Controller.</b><span class="t2">It makes a new key. Old Join Codes and pinned keys no longer apply.</span></li>
      <li><b>Every Worker pairs again.</b><span class="t2">Issue a Join Code for each one. Workers hold no durable state, so their Roles and torrents come back from the restored intent and their files are rechecked.</span></li>
    </ol>
    <p class="nd-info">${I('info')}<span>Changes made after the backup are lost. Torrents keep seeding on their Workers the whole time.</span></p>
  </div>
  <div class="fl-foot"><button class="btn ghost" data-act="ho-back-pick">Back</button><span class="grow"></span></div>`;
}

export function handoverHtml(app) {
  const steps = ['Pick', 'Preflight', 'Confirm', 'Run'];
  const at = { pick: 0, preflight: 1, confirm: 2, run: 3, done: 4, dead: -1 }[handover.step];
  const head = `<div class="fl-head"><div><h2>${handover.step === 'dead' ? 'Replace a dead Controller' : 'Hand over the Controller'}</h2>${at >= 0 ? `<ol class="fl-steps">${steps.map((s, i) => `<li class="${i < at ? 'done' : i === at ? 'cur' : ''}">${i < at ? I('check') : `<span class="num">${i + 1}</span>`}${s}</li>`).join('')}</ol>` : ''}</div><span class="grow"></span>${handover.step === 'run' ? '' : app.flowClose || ''}</div>`;
  const body = { pick: hoPick, preflight: hoPreflight, confirm: hoConfirm, run: hoRun, done: hoDone, dead: hoDead }[handover.step](app);
  return `<div class="flow" data-flow="handover">${head}${body}</div>`;
}

function runHandover(app) {
  handover.from = controllerOf(app.S.nodes).id;
  handover.step = 'run';
  handover.run = 0; handover.sub = 0;
  const steps = hoSteps(app);
  handover.timer = setInterval(() => {
    const s = steps[handover.run];
    if (s.bar || s.list) { handover.sub = Math.min(1, handover.sub + 0.25); if (handover.sub < 1) return app.refresh(); }
    handover.run += 1; handover.sub = 0;
    if (handover.run >= steps.length) {
      clearInterval(handover.timer);
      const to = app.S.nodes.find((n) => n.id === handover.to), old = app.S.nodes.find((n) => n.id === handover.from);
      old.kind = 'Worker'; to.kind = 'Controller';
      old.link = { ...old.link, dial: lan(old) && lan(to) ? 'controller' : 'worker', heartbeat: 1, latency: 1 };
      to.link = { ...to.link, dial: 'self', heartbeat: null };
      to.roles.Serving = { state: 'active' };
      to.libraryRoot = old.libraryRoot;
      handover.step = 'done';
    }
    app.refresh();
  }, 450);
}

export const hoActions = {
  'ho-to': (app, el) => { if (el.getAttribute('aria-disabled') !== 'true') handover.to = el.dataset.node; },
  'ho-next': () => { if (handover.to) handover.step = 'preflight'; },
  'ho-back': () => { handover.step = { preflight: 'pick', confirm: 'preflight' }[handover.step] || 'pick'; },
  'ho-back-pick': () => { handover.step = 'pick'; },
  'ho-dead': () => { handover.step = 'dead'; },
  'ho-confirm': () => { handover.step = 'confirm'; handover.pin = ''; },
  'ho-pin': (app, value) => { handover.pin = value.replace(/\D/g, '').slice(0, 4); if (handover.pin.length === 4) runHandover(app); },
};

// ═════════════ Remove and Force confirms

export function removeHtml(app, n) {
  return `<div class="flow"><div class="fl-head"><div><h2>Remove ${esc(n.name)}</h2></div><span class="grow"></span>${app.flowClose || ''}</div>
    <div class="fl-body"><p class="fl-lede">Removing ${esc(n.name)} deletes its pinned key. To come back it must pair again with a new Join Code.</p>
    <ul class="pf-list">${Object.keys(n.roles).map((r) => `<li>${I('log-out')}<span>${esc(r)} ${n.state === 'offline' ? 'stops here' : 'drains first'}${ROLES[r].always && !app.S.nodes.some((m) => m !== n && m.roles[r]?.state === 'active') ? `, then goes back to ${esc(controllerOf(app.S.nodes).name)}. It is always held.` : '.'}</span></li>`).join('')}</ul>
    <p class="t3">Files and torrents on ${esc(n.name)} stay on its disk. SlipStream stops tracking them once the drain ends.</p></div>
    <div class="fl-foot"><span class="grow"></span><button class="btn" data-act="close">Keep ${esc(n.name)}</button><button class="btn fail" data-act="remove-yes" data-node="${n.id}">${I('trash-2')}Remove ${esc(n.name)}</button></div></div>`;
}

export function forceHtml(app, n) {
  const tied = jobsOn(n).filter((j) => j.status === 'running' && !ROLES[roleOfJob(j)]?.portable && j.stage !== 'Transfer');
  return `<div class="flow"><div class="fl-head"><div><h2>Force ${esc(n.name)}'s tied work lost</h2></div><span class="grow"></span>${app.flowClose || ''}</div>
    <div class="fl-body"><p class="fl-lede">Do this only if ${esc(n.name)} is not coming back. Waiting costs nothing. Force cannot be undone.</p>
    <div class="nd-alert al-fail al-plain"><p><b>Marked lost:</b></p><ul class="wl">${tied.map((j) => workRow(j, { compact: true, held: true, note: 'SlipStream searches again for it', noteCls: 'fail-dim' })).join('')}</ul>
    ${n.seeding ? `<p class="al-seed">${I('arrow-up')}<span><b class="num">${n.seeding.count}</b> seeding torrents stop counting. Private trackers may count unmet goals against your ratio.</span></p>` : ''}</div>
    <p class="t3">Portable jobs are unaffected. They requeue on their own after the grace period.</p></div>
    <div class="fl-foot"><span class="grow"></span><button class="btn" data-act="close">Keep waiting</button><button class="btn fail" data-act="force-yes" data-node="${n.id}">Mark lost</button></div></div>`;
}

// ═════════════ New Node first-run page (?view=worker-join)

export const join = { step: 'paste', code: '', err: null };
export function joinHtml() {
  const n = joiningNode();
  const ctlFp = 'SHA256 7F3A 92C1 0B6E D4F8 51AA 3E0C 9D27 B6F1';
  let body;
  if (join.step === 'paste') {
    body = `<h1>Join a SlipStream</h1>
      <p class="t2">This machine is a new Node. Paste the Join Code from your Controller. On the Controller, open Nodes and choose Pair a Node.</p>
      <label class="jn-field"><span>Join Code</span><textarea class="mono" rows="3" data-join-code spellcheck="false" placeholder="SLP1-…">${esc(join.code)}</textarea></label>
      ${join.err ? `<p class="jn-err">${I('circle-x')}<span>${join.err}</span></p>` : ''}
      <div class="jn-act"><button class="btn" data-act="join-paste">${I('clipboard-paste')}Paste</button><span class="grow"></span><button class="btn pri" data-act="join-check">Continue</button></div>
      <p class="jn-alt t3">Not joining anything? <button class="lnk" data-toast="Starting a new SlipStream is not in this prototype"><b>Make this machine a Controller</b></button></p>`;
  } else if (join.step === 'confirm') {
    body = `<h1>Join atlas?</h1>
      <p class="t2">The code points at a Controller named <b class="mono">atlas</b>. Check its key against what atlas shows before you join.</p>
      <dl class="kvl kv-tight jn-kv"><dt>Controller</dt><dd class="mono">atlas</dd><dt>Addresses</dt><dd class="mono">${CONTROLLER_ADDRS.join('<br>')}</dd><dt>Key</dt><dd class="fp mono">${ctlFp}</dd></dl>
      <div class="fp-cmp"><p class="t2">atlas will show this Node's key. Compare it there.</p><p class="fp-big mono">${fpShort(n.link.fingerprint).split(' ').map((g) => `<span>${g}</span>`).join('')}</p></div>
      ${join.err ? `<p class="jn-err">${I('circle-x')}<span>${join.err}</span></p>` : ''}
      <div class="jn-act"><button class="btn ghost" data-act="join-back">Back</button><span class="grow"></span><button class="btn pri" data-act="join-go">Join atlas</button></div>`;
  } else if (join.step === 'joining') {
    body = `<div class="jn-wait"><span class="spin big" aria-hidden="true"></span><h1>Swapping keys with atlas</h1><p class="t2">This takes a few seconds.</p></div>`;
  } else {
    body = `<div class="jn-done"><span class="done-ic">${I('check')}</span><h1>Paired with atlas.</h1><p class="jn-big">Roles come from the Controller.</p>
      <p class="t2">atlas dials this Node at <span class="mono">${n.addr}</span>. Manage <span class="mono">cellar</span> from atlas. You can close this page.</p></div>`;
  }
  return `<div class="jn">
    <header class="jn-top">${mark()}<b>SlipStream</b><span class="t3">New Node · <span class="mono">cellar</span> · ${n.hw}</span></header>
    <main class="jn-card">${body}</main>
    ${join.step === 'paste' || join.step === 'confirm' ? `<footer class="jn-foot t3">This Node's key <span class="mono">${fpShort(n.link.fingerprint)}</span></footer>` : ''}
  </div>`;
}
export const joinActions = {
  'join-paste': () => { join.code = CODE_STR; join.err = null; },
  'join-check': (app) => {
    const c = (app.root.querySelector('[data-join-code]')?.value || '').trim();
    join.code = c;
    if (!/^SLP1-/i.test(c)) { join.err = 'This is not a Join Code. Join Codes start with SLP1.'; return; }
    join.err = null; join.step = 'confirm';
  },
  'join-back': () => { join.step = 'paste'; join.err = null; },
  'join-go': (app) => { join.step = 'joining'; setTimeout(() => { join.step = 'done'; app.refresh(); }, 1600); },
};
export function joinSim(what) {
  if (what === 'join') { join.code = CODE_STR; join.err = null; join.step = 'confirm'; }
  if (what === 'expire') { join.code = join.code || CODE_STR; join.step = 'paste'; join.err = `This Join Code expired at ${expiryClock()}. Ask for a new one on atlas, under Nodes, Pair a Node.`; }
  if (what === 'used') { join.code = join.code || CODE_STR; join.step = 'paste'; join.err = 'This Join Code was already used by another Node. Each code pairs one Node. Ask atlas for a new one.'; }
}

// ═════════════ Serving Worker status page (?view=worker-status)

export function workerStatusHtml(app) {
  const n = app.S.nodes.find((m) => m.id === 'seedbox-ams');
  const js = jobsOn(n).filter((j) => j.status === 'running');
  const roles = [
    ['Searching', 'Paused. New searches need the Controller\'s decisions.'],
    ['Downloading', `Finishing ${plural(js.filter((j) => j.stage === 'Downloading').length, 'download')}. Seeding ${n.seeding.count} torrents to the goals in its last copy of the intent.`],
    ['Serving', 'Shows this page until atlas answers.'],
  ];
  const note = (j) => (j.stage === 'Searching' ? 'Paused until atlas answers' : j.stage === 'Transfer' ? 'Paused. Resumes from verified chunks.' : 'Finishing');
  return `<div class="ws">
    <header class="ws-top">${mark()}<b>SlipStream</b><span class="t3"><span class="mono">seedbox-ams</span> · Worker · status page</span><span class="grow"></span><span class="npill st-warn">${dot('warn')}Controller unreachable</span></header>
    <main class="ws-main">
      <section class="ws-hero">
        <h1>atlas isn't answering.</h1>
        <p class="t2">The full SlipStream lives on atlas, the Controller: the library, decisions and sign-in. Until it answers, seedbox-ams shows only what it holds. Nothing on this page changes settings.</p>
        <p class="t3 num">Last contact ${hm('2026-10-08T21:31:00')} · retrying every 30 s</p>
      </section>
      <section class="nd-sec"><div class="sec-h"><h2>Roles on seedbox-ams</h2></div><div class="nd-card"><ul class="rlist">${roles.map(([r, t]) => `<li class="rrow"><span class="rrow-n"><b>${r}</b><span class="t3">${GROUPS.find((g) => g.roles.includes(r)).id}</span></span><span class="rrow-s"><span class="t2">${esc(t)}</span></span></li>`).join('')}</ul></div></section>
      <section class="nd-sec"><div class="sec-h"><h2>Jobs</h2><span class="t3">Finishing in-flight. Starting nothing new.</span></div><div class="nd-card"><ul class="wl">${js.map((j) => workRow(j, { note: note(j), held: j.stage === 'Searching' || j.stage === 'Transfer' })).join('')}</ul></div></section>
      <section class="nd-sec"><div class="sec-h"><h2>Torrents</h2><span class="t3 num">${n.seeding.count} seeding · ratio ${n.seeding.ratio}</span></div><div class="nd-card"><ul class="tor tor-lg">${torrents.map((t) => `<li><span class="ell">${esc(works[t.work].title)} <span class="t3">${esc(itemLabel(t.work, t.itemKey))}</span></span><span class="num t2">${t.ratio}</span><span class="t3">to ${esc(t.goal)}</span></li>`).join('')}<li class="t3"><span>and ${n.seeding.count - torrents.length} more</span></li></ul></div></section>
      <section class="nd-sec"><div class="sec-h"><h2>Outbox</h2></div><div class="nd-card"><p class="ws-out"><b class="num">7</b> results wait for atlas. Each carries its job id, so sending them again when atlas answers is safe.</p></div></section>
    </main>
  </div>`;
}
