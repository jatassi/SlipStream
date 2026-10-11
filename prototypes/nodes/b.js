// PROTOTYPE · Variant B · Matrix. Grid-first: Nodes by Roles, cells ticked in place, guardrails inside the cells,
// pending changes collected in a bottom bar. Node detail opens as a right inspector (wide) or a pushed page (phone).

import { GROUPS, ROLES, ROLE_NAMES, CONTROLLER_VERSION } from './data.js';
import { esc, I, dot, plural, wideShell, crumbs, phoneTop, phonePushed, tabs, statePill, kindChip, nodeState, loadLine } from './shell.js';
import { controllerOf, jobsOn, cell, toggle, plan, apply, summary, changeLine, runTest, testFor, needsTest, blockers } from './model.js';
import { header, defaultActions, detailBody } from './detail.js';
import { pairingHtml, handoverHtml } from './flows.js';

export const name = 'Matrix';

let lastNote = '';

function cellBtn(app, n, r) {
  const c = cell(app.S.nodes, app.draft, n, r);
  const t = c.pending === 'add' && ROLES[r].reach ? testFor(n, r) : null;
  const cls = ['mc'];
  let icon = '';
  let label = c.on ? 'held' : 'not held';
  if (c.on) { cls.push('on'); icon = I('check'); }
  if (c.pending) { cls.push(`p-${c.pending}`); label = { add: 'adding', drain: 'draining after apply', handback: 'handed back automatically' }[c.pending]; }
  if (c.pending === 'drain') icon = I('minus');
  if (c.draining && !c.pending) { cls.push('draining'); icon = I('log-out'); label = `draining, ${c.draining}`; }
  if (t?.status === 'running') { cls.push('testing'); icon = '<span class="spin" aria-hidden="true"></span>'; label = 'testing reach'; }
  if (t?.status === 'fail') { cls.push('t-fail'); icon = I('x'); label = `test failed, ${t.note}`; }
  if (c.lock) { cls.push(c.on ? 'lock-on' : 'lock-off'); if (!c.on) icon = I('lock'); label = c.lock; }
  const tip = c.lock || (t ? t.note : '') || (c.draining ? `Draining: ${c.draining}` : '') || c.warn || '';
  return `<button class="${cls.join(' ')}" data-act="b-cell" data-node="${n.id}" data-role="${esc(r)}" data-fid="c-${n.id}-${esc(r)}" role="checkbox" aria-checked="${c.on}" ${c.lock ? 'aria-haspopup="dialog"' : ''} aria-label="${esc(n.name)} ${esc(r)}: ${esc(label)}" title="${esc(tip)}">${icon}${c.warn && !c.lock ? '<span class="mc-w" aria-hidden="true"></span>' : ''}${c.pending === 'handback' ? '<span class="mc-auto">auto</span>' : ''}${c.lock && c.on ? `<span class="mc-l">${I('lock')}</span>` : ''}</button>`;
}

const roleHead = (r) => `<span class="rh-n">${esc(r)}</span><span class="rh-g" aria-hidden="true">${ROLES[r].always ? `<span title="Always held">${I('anchor')}</span>` : ''}${ROLES[r].reach ? `<span title="Tested from the Node before saving">${I('radio-tower')}</span>` : ''}${ROLES[r].gated ? `<span title="Probe-gated">${I('flask-conical')}</span>` : ''}</span>`;

function gridWide(app) {
  const { nodes } = app.S;
  return `<div class="nmx-wrap"><table class="nmx">
    <caption class="sr">Roles by Node. Tick a cell to give a Node a Role.</caption>
    <thead>
      <tr class="nmx-g"><th rowspan="2" class="nmx-corner" scope="col">Node</th>${GROUPS.map((g) => `<th colspan="${g.roles.length}" scope="colgroup" class="gh">${g.id}</th>`).join('')}</tr>
      <tr class="nmx-r">${GROUPS.map((g) => g.roles.map((r, i) => `<th scope="col" class="rh${i === 0 ? ' g0' : ''}">${roleHead(r)}</th>`).join('')).join('')}</tr>
    </thead>
    <tbody>${nodes.map((n) => `<tr class="${app.cfg.node === n.id ? 'sel' : ''}${n.state === 'offline' ? ' off' : ''}">
      <th scope="row" class="nh"><button class="nh-b" data-go='${JSON.stringify({ node: app.cfg.node === n.id ? '' : n.id, view: 'nodes' })}' data-fid="nh-${n.id}">${dot(nodeState(n).k)}<span class="nh-t"><b class="mono">${esc(n.name)}</b><span class="t3">${n.kind === 'Controller' ? 'Controller' : 'Worker'} · <span class="num ${n.version === CONTROLLER_VERSION ? '' : 'warn-t'}">${esc(n.version)}</span></span></span></button><span class="nh-load num t3">${loadLine(n)}</span></th>
      ${GROUPS.map((g) => g.roles.map((r, i) => `<td class="${i === 0 ? 'g0' : ''}">${cellBtn(app, n, r)}</td>`).join('')).join('')}
    </tr>`).join('')}
    <tr class="ghost"><th scope="row" class="nh"><button class="btn sm ghost" data-act="pair">${I('plus')}Pair a Node</button></th><td colspan="10" class="t3">A new row appears here with Roles pre-ticked from its probes.</td></tr>
    </tbody></table></div>`;
}

function gridPhone(app) {
  const { nodes } = app.S;
  return `<div class="tmx-wrap"><table class="tmx">
    <caption class="sr">Roles by Node</caption>
    <thead><tr><th class="tmx-corner" scope="col">Role</th>${nodes.map((n) => `<th scope="col" class="tn"><button data-go='${JSON.stringify({ node: n.id })}' title="${esc(n.name)}">${dot(nodeState(n).k)}<span class="mono ell">${esc(n.name)}</span></button></th>`).join('')}</tr></thead>
    <tbody>${GROUPS.map((g) => `<tr class="tg"><th colspan="${nodes.length + 1}" scope="colgroup">${g.id}</th></tr>${g.roles.map((r) => `<tr><th scope="row" class="tr">${roleHead(r)}</th>${nodes.map((n) => `<td>${cellBtn(app, n, r)}</td>`).join('')}</tr>`).join('')}`).join('')}</tbody>
  </table></div>`;
}

function legend() {
  return `<div class="mx-legend t3" aria-hidden="true">
    <span><span class="mc on mini">${I('check')}</span>Held</span>
    <span><span class="mc on p-add mini">${I('check')}</span>Adding</span>
    <span><span class="mc on p-drain mini">${I('minus')}</span>Drains on apply</span>
    <span><span class="mc on draining mini">${I('log-out')}</span>Draining</span>
    <span><span class="mc lock-off mini">${I('lock')}</span>Blocked by probe</span>
    <span><span class="mc mini"><span class="mc-w"></span></span>Weak probe</span>
    <span>${I('anchor')}Always held</span><span>${I('radio-tower')}Tested before saving</span><span>${I('flask-conical')}Probe-gated</span>
  </div>`;
}

function pendingBar(app) {
  if (!app.draft.size) return '';
  const p = plan(app.S.nodes, app.draft);
  if (!p.changes.length) return '';
  const bl = blockers(p);
  const testing = bl.some((c) => testFor(c.node, c.role)?.status === 'running');
  const failed = bl.find((c) => testFor(c.node, c.role)?.status === 'fail');
  const note = failed ? `<span class="fail-t">${esc(failed.node.name)} can't reach ${esc(ROLES[failed.role].reach)}. Untick it to apply.</span>` : testing ? '<span class="t2"><span class="spin" aria-hidden="true"></span>Testing reach before saving</span>' : lastNote ? `<span class="t2">${esc(lastNote)}</span>` : '';
  return `<div class="pbar" role="region" aria-label="Pending changes"><b class="num">${summary(p)}</b>${note ? `<span class="pbar-n">${note}</span>` : ''}<span class="grow"></span>
    <button class="btn ghost sm" data-act="b-discard">Discard</button><button class="btn sm" data-act="b-review">Review</button><button class="btn pri sm" data-act="b-apply" ${bl.length ? 'aria-disabled="true"' : ''}>Apply</button></div>`;
}

function singleNote() {
  return `<p class="nd-info solo-line">${I('info')}<span>One Node. atlas is the Controller and runs every Role in process, so the grid is one row. Always-held Roles stay locked on until another Node can take them.</span></p>`;
}

function inspector(app, inner, label) {
  return `<aside class="insp" aria-label="${esc(label)}"><div class="insp-in">${inner}</div></aside>`;
}

function nodeInspector(app, n) {
  const close = `<button class="ib" data-go='${JSON.stringify({ node: '' })}' aria-label="Close inspector">${I('x')}</button>`;
  return `<div class="insp-top"><span class="t3">Node</span><span class="grow"></span>${close}</div>${header(app, n, defaultActions(app, n, ''))}${detailBody(app, n, '<span class="t3">Edit in the grid</span>')}`;
}

export function render(app) {
  const { cfg, S } = app;
  const n = S.nodes.find((m) => m.id === cfg.node);
  const solo = S.nodes.length === 1;
  if (app.phone) {
    app.flowClose = '';
    if (cfg.view === 'pair') return { html: `${phonePushed('Nodes', { view: 'nodes' })}<div class="ph-pad fl-page">${pairingHtml(app)}</div>` };
    if (cfg.view === 'handover') return { html: `${phonePushed('Nodes', { view: 'nodes' })}<div class="ph-pad fl-page">${handoverHtml(app)}</div>` };
    if (n) return { html: `${phonePushed('Nodes', { node: '' })}<div class="ph-pad nd-ph">${header(app, n, defaultActions(app, n, ''))}${detailBody(app, n, '<span class="t3">Edit in the grid</span>')}</div>${tabs()}`, ambient: jobsOn(n).find((j) => j.status === 'running')?.work };
    return { html: `${phoneTop('Nodes')}<p class="ph-sentence t2">${plural(S.nodes.length, 'Node')} · tap a cell to change a Role</p>${solo ? singleNote() : ''}${gridPhone(app)}<div class="ph-acts">${solo ? '' : `<button class="btn" data-act="handover">${I('crown')}Hand over</button>`}<button class="btn pri" data-act="pair">${I('plus')}Pair a Node</button></div>${pendingBar(app)}${tabs()}` };
  }
  app.flowClose = `<button class="ib" data-go='${JSON.stringify({ view: 'nodes' })}' aria-label="Close">${I('x')}</button>`;
  const acts = `${solo ? '' : `<button class="btn" data-act="handover">${I('crown')}Hand over Controller</button>`}<button class="btn pri" data-act="pair">${I('plus')}Pair a Node</button>`;
  const body = `<div class="pg-h"><div><h1>Nodes</h1><p class="t2">${plural(S.nodes.length, 'Node')} · ${solo ? 'one Controller' : `${S.nodes.filter((m) => m.state === 'online').length} online`} · tick a cell to change a Role</p></div><span class="grow"></span>${acts}</div>${solo ? singleNote() : ''}${gridWide(app)}${legend()}${pendingBar(app)}`;
  let aside = '';
  if (cfg.view === 'pair') aside = inspector(app, pairingHtml(app), 'Pair a Node');
  else if (n) aside = inspector(app, nodeInspector(app, n), n.name);
  const out = { html: wideShell(S.nodes, crumbs({ label: 'Nodes' }), body) + aside, ambient: n ? jobsOn(n).find((j) => j.status === 'running')?.work : null };
  if (cfg.view === 'handover') Object.assign(out, { sheet: handoverHtml(app), sheetSize: 'md', sheetClose: { view: 'nodes' } });
  return out;
}

export function modalHtml(app, modal) {
  if (modal.kind !== 'b-review') return null;
  const p = plan(app.S.nodes, app.draft);
  const bl = blockers(p);
  return `<div class="flow"><div class="fl-head"><div><h2>Review changes</h2><p class="t3">${summary(p)}</p></div><span class="grow"></span><button class="ib" data-act="close" aria-label="Close">${I('x')}</button></div>
    <div class="fl-body"><ul class="review">${p.changes.map((c) => `<li class="rv-${c.kind}">${I({ add: 'plus', drain: 'log-out', handback: 'corner-down-left' }[c.kind])}<span>${esc(changeLine(c))}${needsTest(c) ? ` <span class="${testFor(c.node, c.role)?.status === 'ok' ? 'ok-t' : testFor(c.node, c.role)?.status === 'fail' ? 'fail-t' : 't2'}">${esc(testFor(c.node, c.role)?.note || '')}</span>` : ''}</span></li>`).join('')}</ul>
    ${p.drains ? `<p class="nd-info">${I('info')}<span>A drain takes no new work and finishes what it holds. Seeding counts as unfinished until its goals are met. Torrents and files stay put unless you start a move.</span></p>` : ''}</div>
    <div class="fl-foot"><button class="btn ghost" data-act="close">Keep editing</button><span class="grow"></span><button class="btn pri" data-act="b-apply" ${bl.length ? 'aria-disabled="true"' : ''}>Apply</button></div></div>`;
}

export const actions = {
  'b-cell': (app, el) => {
    const n = app.S.nodes.find((m) => m.id === el.dataset.node), r = el.dataset.role;
    const c = toggle(app.S.nodes, app.draft, n, r);
    if (c.lock) { app.popover(el, `<p class="pop-t">${I(c.probe.v === 'blocked' ? 'lock' : 'anchor')}<span><b>${esc(n.name)} · ${esc(r)}</b><span class="t2">${esc(c.lock)}</span></span></p>`); return 'keep'; }
    const after = cell(app.S.nodes, app.draft, n, r);
    lastNote = '';
    if (after.pending === 'add' && ROLES[r].reach) runTest(n, r, () => app.refresh());
    if (after.pending === 'drain') lastNote = after.left ? `${n.name} drains ${r}: ${after.left}.` : `${n.name} leaves ${r}. Nothing in flight.`;
    if (after.pending === 'add' && after.warn) lastNote = `Weak probe on ${n.name}: ${after.warn}.`;
    const hb = plan(app.S.nodes, app.draft).changes.find((x) => x.kind === 'handback' && x.role === r);
    if (hb) lastNote = `${r} is always held. With no other holder it goes back to ${hb.node.name}, so its cell ticked itself.`;
  },
  'b-discard': (app) => { app.draft.clear(); lastNote = ''; },
  'b-review': (app) => { app.modal = { kind: 'b-review', size: 'sm' }; },
  'b-apply': (app) => {
    const p = plan(app.S.nodes, app.draft);
    if (blockers(p).length) return app.toast('Wait for the reach tests to pass');
    apply(app.S.nodes, app.draft);
    lastNote = '';
    app.modal = null;
    app.toast(`Applied. ${summary(p)}`);
  },
};
