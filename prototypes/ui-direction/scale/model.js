// PROTOTYPE. Type-agnostic reads over a Work tree. Nothing here knows which Content Type it is walking.

import { targets, profiles } from './fixture.js';

// One vocabulary for cell states, shared by every variant so the variants differ only in structure.
// `attention` states are the ones a person may act on; `live` ones are moving on their own.
export const STATES = {
  failed: { label: 'Failed', icon: 'circle-x', rank: 0, attention: true },
  upgrade: { label: 'Upgrade available', short: 'Upgrade', icon: 'circle-arrow-up', rank: 1, attention: true },
  missing: { label: 'Missing', icon: 'circle-dashed', rank: 2, attention: true },
  downloading: { label: 'Downloading', icon: 'download', rank: 3, live: true },
  converting: { label: 'Converting', icon: 'replace', rank: 4, live: true },
  met: { label: 'Met', icon: 'check', rank: 5 },
  unaired: { label: 'Not aired', icon: 'calendar-clock', rank: 6 },
};
export const STATE_ORDER = Object.keys(STATES);

export const MISSING_WHY = {
  searching: 'Searching',
  'not-found': 'Not found',
  waiting: 'Waiting on source',
  grace: 'In grace period',
  queued: 'Queued for Conversion',
};

export const profileOf = (work) => profiles[work.profile];
export const targetsOf = (work) => profileOf(work).targets.map((p) => ({ ...targets.find((t) => t.id === p.target), ...p }));
export const editionsOf = (work) => work.editions;

// Depth-first walk. Each Item comes with its ancestor level nodes so a descriptor can label it.
export function items(work, desc) {
  const out = [];
  const visit = (node, path) => {
    if (node.kind === 'item') {
      const parent = path[path.length - 1];
      out.push({ item: node, path, code: desc.item.code(node, parent), label: desc.item.label(node, parent), rowLabel: desc.item.rowLabel(node, parent) });
      return;
    }
    node.children.forEach((c) => visit(c, [...path, node]));
  };
  work.tree.forEach((n) => visit(n, []));
  return out;
}

export function cells(work, desc, within = items(work, desc)) {
  const out = [];
  for (const it of within) {
    for (const ed of editionsOf(work)) {
      for (const t of targetsOf(work)) out.push({ ...it, edition: ed, target: t, cell: it.item.cells[ed.id][t.id] });
    }
  }
  return out;
}

export function counts(list) {
  const c = Object.fromEntries(STATE_ORDER.map((s) => [s, 0]));
  list.forEach((x) => { c[x.cell.state] += 1; });
  return c;
}

// The worst state among a set of cells, by rank: a season with one failure reads as failed.
export const worst = (list) => list.reduce((a, x) => (STATES[x.cell.state].rank < STATES[a].rank ? x.cell.state : a), 'unaired');

export function summaryLine(c) {
  const wanted = STATE_ORDER.filter((s) => s !== 'unaired').reduce((n, s) => n + c[s], 0);
  const parts = [`${c.met} of ${wanted} met`];
  if (c.failed) parts.push(`${c.failed} failed`);
  if (c.downloading + c.converting) parts.push(`${c.downloading + c.converting} in flight`);
  if (c.upgrade) parts.push(`${c.upgrade} upgrade${c.upgrade > 1 ? 's' : ''} available`);
  if (c.missing) parts.push(`${c.missing} missing`);
  if (c.unaired) parts.push(`${c.unaired} not aired`);
  return parts;
}

export const levelNodes = (work) => work.tree.filter((n) => n.kind === 'level');

// A cell only a person can move: it failed, a better release waits, or nothing exists to grab. Waiting and queued cells resolve on their own.
export const needsYou = (cell) => cell.state === 'failed' || cell.state === 'upgrade' || (cell.state === 'missing' && cell.why === 'not-found');
