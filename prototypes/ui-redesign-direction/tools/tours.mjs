// PROTOTYPE tool: one scripted tour per direction for record.mjs. Each step may go to a
// shell and route, hover, click, press a key, type, scroll or move the pointer, then wait.
const desk = (key, open, search, pick) => [
  { go: ['wide', 'home'], wait: 3400 },
  { move: [420, 330], wait: 300 },
  { move: [1180, 300], wait: 1400 },
  { go: ['wide', 'library'], wait: 1000 },
  { hover: open.hover, wait: 1300 },
  { click: open.click, wait: 2800 },
  { scroll: 520, wait: 1600 },
  { key: 'Meta+k', wait: 500 },
  { type: search, wait: 700 },
  { key: 'Enter', wait: 2600 },
  { go: ['wide', 'activity'], wait: 2600 },
  { go: ['wide', 'nodes'], wait: 2200 },
  { go: ['phone', 'home'], wait: 2800 },
  { go: ['phone', `work/${pick}`], wait: 1200 },
  { move: [720, 600], wait: 100 },
  { scroll: 360, wait: 1700 },
  { go: ['phone', `work/${pick}/actions`], wait: 1800 },
  { go: ['portal', 'home'], wait: 2600 },
  { go: ['portal', 'request/r3'], wait: 2800 },
  { go: ['portal-wide', 'home'], wait: 2600 },
];

export const TOURS = {
  A: desk('A', { hover: '.a-tile[data-work="neon-psalm"]', click: '.a-tile[data-work="paper-lanterns"]' }, 'glass', 'brightwater'),
  B: desk('B', { hover: '.b-grid .b-tile[data-open="neon-psalm"]', click: '.b-grid .b-tile[data-open="halcyon-drift"]' }, 'brightwater', 'brightwater'),
  C: [
    { go: ['wide', 'home'], wait: 2600 },
    { hover: '.c-legend__i[data-line-btn="process"]', wait: 1600 },
    { move: [700, 120], wait: 900 },
    ...desk('C', { hover: '.c-tile[data-work="neon-psalm"]', click: '.c-tile[data-work="halcyon-drift"]' }, 'glass', 'brightwater').slice(3),
  ],
};
