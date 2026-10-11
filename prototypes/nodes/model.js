// PROTOTYPE. One planner for Role changes, shared by all three variants. Every guardrail lives here, not in the views.

import { ROLES, ROLE_NAMES, probe, reachTest, jobs } from './data.js';

export const key = (nodeId, role) => `${nodeId}|${role}`;
export const controllerOf = (nodes) => nodes.find((n) => n.kind === 'Controller');
export const holds = (n, r) => n.roles[r]?.state === 'active';
export const draining = (n, r) => n.roles[r]?.state === 'draining';

const STAGE_ROLE = { Searching: 'Searching', Downloading: 'Downloading', Conversion: 'Converting', Import: 'Storage' };
export const roleOfJob = (j) => STAGE_ROLE[j.stage] || null;
export const jobsOn = (n) => jobs.filter((j) => j.node === n.id || (j.node.includes('→') && j.node.split(' → ').includes(n.id)));

export function leftFor(n, r) {
  if (n.roles[r]?.left) return n.roles[r].left;
  const js = jobsOn(n).filter((j) => roleOfJob(j) === r && j.status === 'running');
  const parts = [];
  if (js.length) parts.push(`${js.length} ${r === 'Converting' ? 'Conversion' : r === 'Searching' ? 'search' : r === 'Storage' ? 'import' : 'download'}${js.length > 1 ? (r === 'Searching' ? 'es' : 's') : ''} finishing`);
  if (r === 'Downloading' && n.seeding) parts.push(`${n.seeding.count} torrents still seeding to goals`);
  return parts.join(' · ');
}

// A draft maps key(node, role) to the wanted holding, only where it differs from now.
export function plan(nodes, draft) {
  const ctl = controllerOf(nodes);
  const want = (n, r) => (draft.has(key(n.id, r)) ? draft.get(key(n.id, r)) : holds(n, r));
  let changes = [];
  for (const n of nodes) for (const r of ROLE_NAMES) {
    const k = key(n.id, r);
    if (!draft.has(k) || draft.get(k) === holds(n, r)) continue;
    changes.push(draft.get(k) ? { kind: 'add', node: n, role: r, k } : { kind: 'drain', node: n, role: r, k, left: leftFor(n, r) });
  }
  const auto = new Set();
  for (const r of ROLE_NAMES) {
    if (!ROLES[r].always || nodes.some((n) => want(n, r))) continue;
    auto.add(r);
    if (holds(ctl, r)) changes = changes.filter((c) => !(c.node === ctl && c.role === r));
    else changes.push({ kind: 'handback', node: ctl, role: r, k: key(ctl.id, r) });
  }
  const finalHas = (n, r) => want(n, r) || (n === ctl && auto.has(r));
  return { changes, finalHas, auto, drains: changes.filter((c) => c.kind === 'drain').length };
}

export function cell(nodes, draft, n, r) {
  const p = plan(nodes, draft);
  const now = holds(n, r);
  const on = p.finalHas(n, r);
  const ch = p.changes.find((c) => c.node === n && c.role === r);
  const pr = probe(n.id, r);
  const ctl = n.kind === 'Controller';
  let lock = null;
  if (pr.v === 'blocked' && !now) lock = `Blocked by probe. ${pr.note}`;
  else if (ctl && r === 'Serving') lock = 'The Controller always serves.';
  else if (ctl && ROLES[r].always && on && !nodes.some((m) => m !== n && p.finalHas(m, r))) lock = `Always held. No other Node holds ${r}, so the Controller keeps it.`;
  return { now, on, pending: ch ? ch.kind : null, left: ch?.left, draining: draining(n, r) ? n.roles[r].left : null, handedBack: !!n.roles[r]?.handedBack, lock, warn: pr.v === 'weak' ? pr.note : null, probe: pr };
}

export function toggle(nodes, draft, n, r) {
  const c = cell(nodes, draft, n, r);
  if (c.lock) return c;
  const k = key(n.id, r);
  const next = !(draft.has(k) ? draft.get(k) : holds(n, r));
  if (next === holds(n, r)) draft.delete(k); else draft.set(k, next);
  return c;
}

// Must-reach Roles test from the Node before saving. Results live here so every variant reads the same verdict.
export const tests = new Map();
export function runTest(n, r, onDone) {
  const k = key(n.id, r);
  tests.set(k, { status: 'running', note: `Reaching ${ROLES[r].reach} from ${n.name}` });
  setTimeout(() => {
    const t = reachTest(n.id, r);
    tests.set(k, { status: t.ok ? 'ok' : 'fail', note: t.note });
    onDone && onDone();
  }, 1400);
}
export const testFor = (n, r) => tests.get(key(n.id, r));
export const needsTest = (c) => c.kind === 'add' && !!ROLES[c.role].reach;

export function blockers(p) {
  return p.changes.filter(needsTest).filter((c) => testFor(c.node, c.role)?.status !== 'ok');
}

export function apply(nodes, draft) {
  const p = plan(nodes, draft);
  for (const c of p.changes) {
    if (c.kind === 'add') c.node.roles[c.role] = { state: 'active' };
    if (c.kind === 'handback') c.node.roles[c.role] = { state: 'active', handedBack: true };
    if (c.kind === 'drain') { if (c.left) c.node.roles[c.role] = { state: 'draining', left: c.left }; else delete c.node.roles[c.role]; }
  }
  draft.clear();
  return p;
}

export function summary(p) {
  const n = p.changes.length;
  return `${n} change${n === 1 ? '' : 's'}${p.drains ? ` · ${p.drains} drain${p.drains > 1 ? 's' : ''}` : ''}`;
}

export function changeLine(c) {
  if (c.kind === 'add') return `${c.node.name} takes ${c.role}.`;
  if (c.kind === 'handback') return `${c.role} goes back to ${c.node.name}. It is always held, and no other Node would hold it.`;
  return c.left ? `${c.node.name} drains ${c.role}. No new work. ${c.left}.` : `${c.node.name} leaves ${c.role}. Nothing is in flight.`;
}
