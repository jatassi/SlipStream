// PROTOTYPE · Variant C · By Role. Role-first: the page is the five Role Groups, each Role lists its holder Nodes with live load.
// Guardrails read as Role properties. Adding a holder shows each Node's probe verdict in the add control. Node detail opens in a sheet.

import { GROUPS, ROLES, CONTROLLER_VERSION, probe } from './data.js';
import { esc, I, dot, plural, wideShell, crumbs, phoneTop, tabs, nodeState, statePill } from './shell.js';
import { controllerOf, jobsOn, roleOfJob, cell, apply, plan, changeLine, runTest, testFor, key } from './model.js';
import { header, defaultActions, detailBody } from './detail.js';
import { pairingHtml, handoverHtml } from './flows.js';

export const name = 'By Role';

const adding = new Map();

function roleLoad(n, r) {
  if (n.state === 'offline') return '<span class="fail-t">offline</span>';
  const st = n.roles[r];
  if (st?.state === 'draining') return `draining · ${esc(st.left)}`;
  const js = jobsOn(n).filter((j) => roleOfJob(j) === r && j.status === 'running');
  if (r === 'Converting') return js.length ? `${plural(js.length, 'Conversion')} · GPU ${n.gpu ?? n.cpu}%` : 'idle';
  if (r === 'Downloading') return [js.length ? plural(js.length, 'download') : '', n.seeding ? `${n.seeding.count} seeding` : ''].filter(Boolean).join(' · ') || 'idle';
  if (r === 'Searching') return js.length ? plural(js.length, 'search', 'searches') : 'idle';
  if (r === 'Storage') return `${esc(n.disk.replace(/ of .*/, ''))}${js.length ? ` · ${plural(js.length, 'import')}` : ''}`;
  if (r === 'Serving') return n.kind === 'Controller' ? 'always' : 'entry point';
  if (r === 'Relaying') return '0 links relayed';
  return `CPU ${n.cpu}%`;
}

function holderChip(app, n, r) {
  const c = cell(app.S.nodes, new Map(), n, r);
  const st = n.roles[r];
  const removable = !c.lock && st.state === 'active';
  const why = c.lock || (st.state === 'draining' ? 'Already draining' : '');
  return `<span class="hchip${st.state === 'draining' ? ' draining' : ''}${n.state === 'offline' ? ' off' : ''}">
    <button class="hc-n" data-go='${JSON.stringify({ node: n.id })}'>${dot(nodeState(n).k)}<b class="mono">${esc(n.name)}</b></button>
    <span class="hc-l num t3">${roleLoad(n, r)}</span>
    ${removable ? `<button class="hc-x" data-act="c-remove" data-node="${n.id}" data-role="${esc(r)}" data-fid="x-${n.id}-${esc(r)}" aria-label="Remove ${esc(n.name)} from ${esc(r)}">${I('x')}</button>` : `<span class="hc-lock" title="${esc(why)}">${I(st.state === 'draining' ? 'log-out' : 'lock')}</span>`}
  </span>`;
}

function pendingChip(r) {
  const a = adding.get(r);
  if (!a) return '';
  const t = testFor(a.node, r);
  if (!t || t.status === 'running') return `<span class="hchip pending"><span class="spin" aria-hidden="true"></span><b class="mono">${esc(a.node.name)}</b><span class="hc-l t2">Testing reach to ${esc(ROLES[r].reach)}</span></span>`;
  return `<span class="hchip bad"><span class="fail-t">${I('x')}</span><b class="mono">${esc(a.node.name)}</b><span class="hc-l fail-t">${esc(t.note)}</span><button class="hc-x" data-act="c-dismiss" data-role="${esc(r)}" aria-label="Dismiss">${I('x')}</button></span>`;
}

const TAGS = (r) => [ROLES[r].always && ['anchor', 'Always held'], ROLES[r].reach && ['radio-tower', 'Tested before saving'], ROLES[r].gated && ['flask-conical', 'Probe-gated']].filter(Boolean);

function roleRow(app, r) {
  const holders = app.S.nodes.filter((n) => n.roles[r]);
  return `<li class="rr">
    <div class="rr-id"><b>${esc(r)}</b><span class="t3">${esc(ROLES[r].blurb)}</span>${TAGS(r).length ? `<span class="rr-tags">${TAGS(r).map(([ic, t]) => `<span class="tag">${I(ic)}${t}</span>`).join('')}</span>` : ''}</div>
    <div class="rr-h">${holders.map((n) => holderChip(app, n, r)).join('')}${pendingChip(r)}${holders.length ? '' : '<span class="t3">No holder</span>'}
      <button class="btn sm ghost add-h" data-act="c-add" data-role="${esc(r)}" data-fid="add-${esc(r)}" aria-haspopup="dialog">${I('plus')}Add</button></div>
  </li>`;
}

function strip(app) {
  return `<div class="nstrip" role="list" aria-label="Nodes">${app.S.nodes.map((n) => `<button class="ns${app.cfg.node === n.id ? ' on' : ''}" role="listitem" data-go='${JSON.stringify({ node: n.id })}'>
      <span class="ns-t">${dot(nodeState(n).k)}<b class="mono">${esc(n.name)}</b>${n.kind === 'Controller' ? I('crown', 'ns-crown') : ''}</span>
      <span class="t3 ns-s">${n.state === 'online' ? `${plural(Object.keys(n.roles).length, 'Role')} · CPU ${n.cpu}%` : esc(nodeState(n).label)} · <span class="num ${n.version === CONTROLLER_VERSION ? '' : 'warn-t'}">${esc(n.version)}</span></span>
    </button>`).join('')}<button class="ns ns-add" data-act="pair">${I('plus')}<span>Pair a Node</span></button></div>`;
}

function groups(app) {
  return GROUPS.map((g) => `<section class="rg" aria-labelledby="rg-${g.id}"><div class="sec-h"><h2 id="rg-${g.id}">${g.id}</h2><span class="t3">${g.roles.join(', ')}</span></div><ul class="rg-l">${g.roles.map((r) => roleRow(app, r)).join('')}</ul></section>`).join('');
}

export function render(app) {
  const { cfg, S } = app;
  const n = S.nodes.find((m) => m.id === cfg.node);
  const solo = S.nodes.length === 1;
  app.flowClose = `<button class="ib" data-act="close" aria-label="Close">${I('x')}</button>`;
  const ctlLine = `<p class="ctl-only">${I('lock')}<span><b>Always on the Controller, never a Role:</b> library, decisions, schedules, API, sign-in. ${esc(controllerOf(S.nodes).name)} holds them.</span></p>`;
  const soloLine = solo ? `<p class="nd-info solo-line">${I('info')}<span>One Node. atlas holds every Role in process. Pair a Node, then add it as a holder under any Role.</span></p>` : '';
  const acts = `${solo ? '' : `<button class="btn" data-act="handover">${I('crown')}Hand over Controller</button>`}<button class="btn pri" data-act="pair">${I('plus')}Pair a Node</button>`;
  let html;
  if (app.phone) html = `${phoneTop('Nodes')}${strip(app)}${soloLine}<div class="ph-pad">${ctlLine}${groups(app)}</div><div class="ph-acts">${solo ? '' : `<button class="btn" data-act="handover">${I('crown')}Hand over</button>`}</div>${tabs()}`;
  else html = wideShell(S.nodes, crumbs({ label: 'Nodes' }), `<div class="pg-h"><div><h1>Nodes</h1><p class="t2">${plural(S.nodes.length, 'Node')} by Role. Each Role lists the Nodes that hold it.</p></div><span class="grow"></span>${acts}</div>${strip(app)}${soloLine}${ctlLine}<div class="rgs">${groups(app)}</div>`);
  const out = { html };
  if (cfg.view === 'pair') Object.assign(out, { sheet: pairingHtml(app), sheetSize: 'md', sheetClose: { view: 'nodes' } });
  else if (cfg.view === 'handover') Object.assign(out, { sheet: handoverHtml(app), sheetSize: 'md', sheetClose: { view: 'nodes' } });
  else if (n) Object.assign(out, { sheet: `<div class="flow"><div class="fl-head"><span class="t3">Node</span><span class="grow"></span>${app.flowClose}</div><div class="fl-body nd-in-sheet">${header(app, n, defaultActions(app, n, ''))}${detailBody(app, n, '<span class="t3">Change holders under each Role</span>')}</div></div>`, sheetSize: 'lg', sheetClose: { node: '' }, ambient: jobsOn(n).find((j) => j.status === 'running')?.work });
  return out;
}

function addPop(app, r) {
  const cands = app.S.nodes.filter((n) => !n.roles[r]);
  if (!cands.length) {
    return `<p class="pop-h">Add a holder for ${esc(r)}</p><p class="pop-empty t2">${app.S.nodes.length === 1 ? 'There is no other Node yet.' : `Every Node already holds ${esc(r)}.`}</p>${app.S.nodes.length === 1 ? `<button class="mi" data-act="pair">${I('plus')}<span>Pair a Node</span></button>` : ''}`;
  }
  return `<p class="pop-h">Add a holder for ${esc(r)}${ROLES[r].reach ? `<span class="t3">Tested from the Node before saving</span>` : ''}</p>${cands.map((n) => {
    const p = probe(n.id, r);
    const blocked = p.v === 'blocked';
    return `<button class="mi cand${blocked ? ' blocked' : ''}" ${blocked ? 'aria-disabled="true"' : ''} data-act="c-pick" data-node="${n.id}" data-role="${esc(r)}">
      <span class="cand-v ${p.v === 'ok' ? 'ok-t' : p.v === 'weak' ? 'warn-t' : 'fail-t'}">${I(p.v === 'ok' ? 'check' : p.v === 'weak' ? 'triangle-alert' : 'lock')}</span>
      <span class="cand-t"><b class="mono">${esc(n.name)}</b><span class="${p.v === 'ok' ? 't3' : p.v === 'weak' ? 'warn-t' : 'fail-t'}">${esc(p.note)}</span></span>
      ${n.state === 'offline' ? '<span class="fail-t cand-s">offline</span>' : ''}
    </button>`;
  }).join('')}`;
}

function commit(app, n, r, on) {
  const d = new Map([[key(n.id, r), on]]);
  const p = plan(app.S.nodes, d);
  apply(app.S.nodes, d);
  return p;
}

export const actions = {
  'c-add': (app, el) => { app.popover(el, addPop(app, el.dataset.role), 'cand-pop'); return 'keep'; },
  'c-pick': (app, el) => {
    if (el.getAttribute('aria-disabled') === 'true') return;
    const n = app.S.nodes.find((m) => m.id === el.dataset.node), r = el.dataset.role;
    app.closePop();
    if (ROLES[r].reach) {
      adding.set(r, { node: n });
      runTest(n, r, () => {
        if (testFor(n, r).status === 'ok') { adding.delete(r); commit(app, n, r, true); app.toast(`${n.name} holds ${r}. ${testFor(n, r).note}`); }
        app.refresh();
      });
      return;
    }
    commit(app, n, r, true);
    const p = probe(n.id, r);
    app.toast(`${n.name} holds ${r}${p.v === 'weak' ? `. Weak probe: ${p.note}` : ''}`);
  },
  'c-dismiss': (app, el) => { adding.delete(el.dataset.role); },
  'c-remove': (app, el) => {
    const n = app.S.nodes.find((m) => m.id === el.dataset.node), r = el.dataset.role;
    const p = plan(app.S.nodes, new Map([[key(n.id, r), false]]));
    app.popover(el, `<p class="pop-h">Remove ${esc(n.name)} from ${esc(r)}?</p><ul class="review pop-rv">${p.changes.map((c) => `<li class="rv-${c.kind}">${I(c.kind === 'handback' ? 'corner-down-left' : 'log-out')}<span>${esc(changeLine(c))}</span></li>`).join('')}</ul>
      <div class="pop-act"><button class="btn sm ghost" data-act="close">Cancel</button><button class="btn sm pri" data-act="c-drain" data-node="${n.id}" data-role="${esc(r)}">${p.changes.some((c) => c.kind === 'handback') ? 'Drain and hand back' : p.drains ? 'Drain' : 'Remove'}</button></div>`, 'rm-pop');
    return 'keep';
  },
  'c-drain': (app, el) => {
    const n = app.S.nodes.find((m) => m.id === el.dataset.node), r = el.dataset.role;
    app.closePop();
    const p = commit(app, n, r, false);
    app.toast(p.changes.map(changeLine).join(' ').replace(/\.$/, ''));
  },
};

export function modalHtml() { return null; }
