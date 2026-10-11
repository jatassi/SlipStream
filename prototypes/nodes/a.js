// PROTOTYPE · Variant A · Roster. Node-first: a list of Node cards, Node detail as a pushed page, Roles edited in a sheet
// with the five Role Groups as switches and a review step that shows what drains.

import { GROUPS, ROLES, CONTROLLER_VERSION } from './data.js';
import { esc, I, dot, plural, wideShell, crumbs, phoneTop, phonePushed, tabs, statePill, kindChip, roleChip, loadLine, dialText, heartbeatText, workRow, nodeState } from './shell.js';
import { controllerOf, jobsOn, cell, toggle, plan, apply, summary, changeLine, runTest, testFor, needsTest, blockers, key } from './model.js';
import { header, defaultActions, detailBody } from './detail.js';
import { pairingHtml, handoverHtml } from './flows.js';

export const name = 'Roster';

function sentence(nodes) {
  const off = nodes.filter((n) => n.state === 'offline'), warn = nodes.filter((n) => ['degraded', 'updating', 'behind'].includes(n.state));
  const parts = [`<span class="s-live">${dot('draft')}${plural(nodes.length, 'Node')}</span>`];
  if (nodes.length > 1) parts.push(`<span>${nodes.some((n) => n.version !== CONTROLLER_VERSION) ? 'Controller on' : 'all on'} <span class="num">${CONTROLLER_VERSION}</span></span>`);
  off.forEach((n) => parts.push(`<span class="s-fail">${esc(n.name)} offline</span>`));
  warn.forEach((n) => parts.push(`<span class="s-warn">${esc(n.name)} ${esc(nodeState(n).label.toLowerCase())}</span>`));
  return parts.join('<span class="s-sep" aria-hidden="true">·</span>');
}

function card(app, n) {
  const ctl = controllerOf(app.S.nodes);
  const lead = jobsOn(n).find((j) => j.status === 'running');
  return `<li><button class="ncard${n.state === 'offline' ? ' off' : ''}" data-go='${JSON.stringify({ node: n.id })}' data-fid="card-${n.id}" aria-label="${esc(n.name)}, ${esc(nodeState(n).label)}">
    <span class="nc-id">
      <span class="nc-name"><b class="mono">${esc(n.name)}</b>${kindChip(n)}</span>
      <span class="t3 ell">${esc(n.hw)} · ${esc(n.where)}</span>
      <span class="nc-ver t3"><span class="num ${n.version === CONTROLLER_VERSION ? '' : 'warn-t'}">${esc(n.version)}</span> · ${n.install === 'docker' ? 'Docker' : 'native'}</span>
    </span>
    <span class="nc-state">${statePill(n)}</span>
    <span class="nc-roles">${Object.keys(n.roles).length ? Object.keys(n.roles).map((r) => roleChip(n, r)).join('') : '<span class="t3">No Roles yet</span>'}${n.kind === 'Controller' ? '<span class="chip chip-fixed">' + I('lock') + 'Library, decisions, API</span>' : ''}</span>
    <span class="nc-load">
      <span class="num t2">${loadLine(n)}</span>
      <span class="t3 ell">${n.kind === 'Controller' ? (n.single ? 'Every Role in process' : `${app.S.nodes.length - 1} Workers linked`) : `${esc(dialText(n, ctl))}${heartbeatText(n) && n.state !== 'offline' ? ` · ${heartbeatText(n)}` : ''}`}</span>
      ${lead ? `<span class="nc-job t3 ell">${esc(lead.stage)} · <span class="num" data-live-pct="${lead.id}"></span></span>` : ''}
    </span>
    <span class="nc-go">${I('chevron-right')}</span>
  </button></li>`;
}

function singleExplainer() {
  return `<section class="solo"><span class="solo-ic">${I('network')}</span><div><h2>One Node does everything</h2><p class="t2">atlas is the Controller and runs every Role in process. There is no Pairing and nothing listens except the UI. Pair a Node to move Roles to other machines, such as Downloading on a seedbox or Converting on a GPU.</p></div><button class="btn pri" data-act="pair">${I('plus')}Pair a Node</button></section>`;
}

function listPage(app, switcher) {
  const { nodes } = app.S;
  const solo = nodes.length === 1;
  const acts = solo ? '' : `<button class="btn" data-act="handover">${I('crown')}Hand over Controller</button><button class="btn pri" data-act="pair">${I('plus')}Pair a Node</button>`;
  const roster = `<ul class="roster">${nodes.map((n) => card(app, n)).join('')}</ul>`;
  if (app.phone) {
    return `${phoneTop('Nodes')}<p class="ph-sentence">${sentence(nodes)}</p>${solo ? singleExplainer() : ''}<div class="ph-acts">${acts}</div><div class="ph-pad">${roster}</div>${tabs()}`;
  }
  return wideShell(nodes, crumbs({ label: 'Nodes' }), `<div class="pg-h"><div>${switcher ? `<div class="nd-h1"><h1>Nodes</h1>${switcher}</div>` : '<h1>Nodes</h1>'}<p class="sentence sm-sentence">${sentence(nodes)}</p></div><span class="grow"></span>${acts}</div>${solo ? singleExplainer() : ''}${roster}`);
}

function detailPage(app, n) {
  const edit = `<button class="btn" data-act="a-edit" data-node="${n.id}">${I('sliders-horizontal')}Edit Roles</button>`;
  const body = `${header(app, n, defaultActions(app, n, edit))}${detailBody(app, n, '')}`;
  if (app.phone) return `${phonePushed('Nodes', { node: '' })}<div class="ph-pad nd-ph">${body}</div>${tabs()}`;
  return wideShell(app.S.nodes, crumbs({ label: 'Nodes', go: { node: '' } }, { label: n.name }), body);
}

function flowPage(app, title, html) {
  if (app.phone) return `${phonePushed('Nodes', { view: 'nodes' })}<div class="ph-pad fl-page">${html}</div>`;
  return wideShell(app.S.nodes, crumbs({ label: 'Nodes', go: { view: 'nodes' } }, { label: title }), `<div class="fl-page">${html}</div>`);
}

export function render(app, switcher = '') {
  const { cfg, S } = app;
  app.flowClose = '';
  if (cfg.view === 'pair') return { html: flowPage(app, 'Pair a Node', pairingHtml(app)) };
  if (cfg.view === 'handover') return { html: flowPage(app, 'Hand over the Controller', handoverHtml(app)) };
  const n = S.nodes.find((m) => m.id === cfg.node);
  if (n) return { html: detailPage(app, n), ambient: jobsOn(n).find((j) => j.status === 'running')?.work };
  return { html: listPage(app, switcher) };
}

// ── Edit Roles sheet

const ed = { step: 'edit', node: null };

function roleLine(app, n, r) {
  const c = cell(app.S.nodes, app.draft, n, r);
  const t = testFor(n, r);
  let note = '';
  if (c.lock) note = `<span class="${c.probe.v === 'blocked' ? 'fail-t' : 't3'}">${esc(c.lock)}</span>`;
  else if (c.pending === 'handback') note = `<span class="t2">${I('corner-down-left')}Goes back to ${esc(n.name)}. It is always held.</span>`;
  else if (c.pending === 'drain') note = `<span class="t2">${I('log-out')}${c.left ? `Drains. ${esc(c.left)}.` : 'Leaves. Nothing in flight.'}</span>`;
  else if (c.pending === 'add' && ROLES[r].reach) note = !t ? '' : t.status === 'running' ? `<span class="t2"><span class="spin" aria-hidden="true"></span>${esc(t.note)}</span>` : `<span class="${t.status === 'ok' ? 'ok-t' : 'fail-t'}">${I(t.status === 'ok' ? 'check' : 'x')}${esc(t.note)}</span>`;
  else if (c.draining) note = `<span class="t2">Draining. ${esc(c.draining)}.</span>`;
  else if (c.warn) note = `<span class="warn-t">${esc(c.warn)}</span>`;
  const tags = [ROLES[r].always ? 'Always held' : '', ROLES[r].reach ? 'Tested before saving' : ''].filter(Boolean);
  return `<li class="er${c.lock ? ' locked' : ''}${c.pending ? ` p-${c.pending}` : ''}">
    <button class="tick${c.on ? ' on' : ''}${c.pending ? ` p-${c.pending}` : ''}" role="checkbox" aria-checked="${c.on}" ${c.lock ? 'aria-disabled="true"' : ''} data-act="a-tick" data-role="${esc(r)}" data-fid="er-${esc(r)}" aria-label="${esc(r)}">${c.lock && !c.on ? I('lock') : c.on ? I('check') : ''}</button>
    <span class="er-t"><span class="er-n"><b>${esc(r)}</b>${c.warn && !c.lock ? dot('warn') : ''}${tags.map((t2) => `<span class="tag">${t2}</span>`).join('')}</span><span class="t3">${esc(ROLES[r].blurb)}</span>${note ? `<span class="er-note">${note}</span>` : ''}</span>
  </li>`;
}

function groupSwitch(app, n, g) {
  const states = g.roles.map((r) => cell(app.S.nodes, app.draft, n, r));
  const on = states.filter((c) => c.on).length;
  const val = on === g.roles.length ? 'true' : on ? 'mixed' : 'false';
  return `<button class="gsw" role="switch" aria-checked="${val === 'true'}" data-mixed="${val === 'mixed'}" data-act="a-group" data-group="${g.id}" data-fid="g-${g.id}"><span class="gsw-l"><b>${g.id}</b><span class="t3">${g.roles.join(', ')}</span></span><span class="t3 num">${on} of ${g.roles.length}</span><span class="sw" aria-hidden="true"><span></span></span></button>`;
}

export function modalHtml(app, modal) {
  if (modal.kind !== 'a-edit') return null;
  const n = app.S.nodes.find((m) => m.id === modal.node);
  const p = plan(app.S.nodes, app.draft);
  const mine = p.changes;
  const head = `<div class="fl-head"><div><h2>Roles on <span class="mono">${esc(n.name)}</span></h2><p class="t3">${ed.step === 'edit' ? 'Tick a Role Group to tick its Roles. Each Role can still be unticked.' : 'Review. Nothing changes until you apply.'}</p></div><span class="grow"></span><button class="ib" data-act="close" aria-label="Close">${I('x')}</button></div>`;
  if (ed.step === 'review') {
    const bl = blockers(p);
    return `<div class="flow">${head}<div class="fl-body"><ul class="review">${mine.map((c) => `<li class="rv-${c.kind}">${I({ add: 'plus', drain: 'log-out', handback: 'corner-down-left' }[c.kind])}<span>${esc(changeLine(c))}${needsTest(c) ? ` <span class="${testFor(c.node, c.role)?.status === 'ok' ? 'ok-t' : 'fail-t'}">${esc(testFor(c.node, c.role)?.note || '')}</span>` : ''}</span></li>`).join('')}</ul>
      ${p.drains ? `<p class="nd-info">${I('info')}<span>A drain takes no new work and finishes what it holds. Torrents and files stay where they are unless you start a move.</span></p>` : ''}</div>
      <div class="fl-foot"><button class="btn ghost" data-act="a-review-back">Back</button><span class="grow"></span><span class="t3 num">${summary(p)}</span><button class="btn pri" data-act="a-apply" ${bl.length ? 'aria-disabled="true"' : ''}>Apply</button></div></div>`;
  }
  return `<div class="flow">${head}<div class="fl-body er-body">
    ${n.kind === 'Controller' ? `<p class="ctl-only">${I('lock')}<span><b>Always on the Controller:</b> library, decisions, schedules, API, sign-in. These are not Roles.</span></p>` : ''}
    ${GROUPS.map((g) => `<section class="eg">${groupSwitch(app, n, g)}<ul>${g.roles.map((r) => roleLine(app, n, r)).join('')}</ul></section>`).join('')}
  </div>
  <div class="fl-foot"><button class="btn ghost" data-act="close">Cancel</button><span class="grow"></span><span class="t3 num">${mine.length ? summary(p) : 'No changes'}</span><button class="btn pri" data-act="a-review" ${mine.length ? '' : 'aria-disabled="true"'}>Review</button></div></div>`;
}

function tickRole(app, n, r) {
  const c = toggle(app.S.nodes, app.draft, n, r);
  if (c.lock) return app.toast(c.lock);
  const after = cell(app.S.nodes, app.draft, n, r);
  if (after.pending === 'add' && ROLES[r].reach) runTest(n, r, () => app.refresh());
  const p = plan(app.S.nodes, app.draft);
  const hb = p.changes.find((x) => x.kind === 'handback' && x.role === r);
  if (hb) app.toast(`${r} goes back to ${hb.node.name}. It is always held`);
}

export const actions = {
  'a-edit': (app, el) => { app.draft.clear(); ed.step = 'edit'; app.modal = { kind: 'a-edit', node: el.dataset.node, size: 'md' }; },
  'a-tick': (app, el) => { const n = app.S.nodes.find((m) => m.id === app.modal.node); tickRole(app, n, el.dataset.role); },
  'a-group': (app, el) => {
    const n = app.S.nodes.find((m) => m.id === app.modal.node);
    const g = GROUPS.find((x) => x.id === el.dataset.group);
    const cells = g.roles.map((r) => [r, cell(app.S.nodes, app.draft, n, r)]);
    const target = cells.some(([, c]) => !c.on && !c.lock);
    for (const [r, c] of cells) if (!c.lock && c.on !== target) tickRole(app, n, r);
  },
  'a-review': () => { ed.step = 'review'; },
  'a-review-back': () => { ed.step = 'edit'; },
  'a-apply': (app) => {
    const p = plan(app.S.nodes, app.draft);
    if (blockers(p).length) return app.toast('Wait for the reach tests to pass');
    apply(app.S.nodes, app.draft);
    ed.step = 'edit';
    app.modal = null;
    app.toast(`Saved. ${summary(p)}`);
  },
};
