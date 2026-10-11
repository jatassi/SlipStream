// PROTOTYPE fixture for the built-in torrent client screens. Extends the direction fixture. Trackers, torrents, people and numbers are invented.

import * as base from '../ui-direction/data.js';

const ART = '../ui-direction/';
const rebased = Object.fromEntries(Object.entries(base.works).map(([id, w]) => [id, { ...w, art: Object.fromEntries(Object.entries(w.art).map(([k, v]) => [k, ART + v])) }]));

export const now = base.now;
export const admin = base.admin;
export const members = base.members;
export const targets = base.targets;
export const profiles = base.profiles;
export const works = rebased;
export const health = base.health;
export const storage = base.storage;
export const library = base.library;
export const requests = base.requests;
export const stages = base.stages;
export const jobs = base.jobs;
export const today = base.today;

// ───────────────────────────── Content Type descriptors (The Content Type contract, #5)
// Core never names a Content Type. It renders Coverage, art and grouping from these.
export const contentTypes = {
  movie: {
    label: 'Movie',
    plural: 'Movies',
    levels: [],
    item: { key: ['edition'], label: '{edition}' },
    layout: { art: 'poster', coverage: 'item' },
  },
  series: {
    label: 'Series',
    plural: 'Series',
    levels: [{ key: 'season', label: 'Season {season}', short: 'S{season:02}' }],
    item: { key: ['season', 'episode'], label: 'S{season:02}E{episode:02}' },
    layout: { art: 'poster', coverage: 'group-when-whole' },
  },
};

const fill = (tpl, v) => tpl.replace(/\{(\w+)(?::(\d+))?\}/g, (_, k, pad) => (pad ? String(v[k]).padStart(+pad, '0') : String(v[k])));

// "S02E07", "Season 1 · 9 episodes", "S03E01–E02", "Standard". Driven only by the descriptor.
export function coverageLabel(workId, coverage) {
  const w = works[workId], ct = contentTypes[w.type], items = coverage.items;
  if (items.length === 1) return fill(ct.item.label, items[0]);
  const lvl = ct.levels[0];
  if (!lvl) return items.map((i) => fill(ct.item.label, i)).join(', ');
  const groups = [...new Set(items.map((i) => i[lvl.key]))];
  if (groups.length === 1 && coverage.whole) return `${fill(lvl.label, items[0])} · ${items.length} episodes`;
  if (groups.length === 1) return `${fill(ct.item.label, items[0])}–E${String(items[items.length - 1].episode).padStart(2, '0')}`;
  return groups.map((g) => fill(lvl.short, { [lvl.key]: g })).join(', ');
}
export const kindLabel = (workId) => contentTypes[works[workId].type].label;

// ───────────────────────────── Nodes holding the Downloading Role
export const nodes = [
  { id: 'atlas', name: 'atlas', kind: 'Controller', where: 'Home · LAN', status: 'online', roles: ['Downloading', 'Storage', 'Metadata', 'Media Server link', 'Subtitles', 'Notifying', 'Serving'], disk: { freeTB: 14.8, totalTB: 56, path: '/Volumes/Vault/Downloads' } },
  { id: 'seedbox-ams', name: 'seedbox-ams', kind: 'Worker', where: 'Amsterdam · remote', status: 'online', roles: ['Searching', 'Downloading'], disk: { freeTB: 1.9, totalTB: 8, path: '/home/ss/downloads' } },
  { id: 'forge', name: 'forge', kind: 'Worker', where: 'Home · LAN', status: 'online', roles: ['Converting'] },
  { id: 'relay-fra', name: 'relay-fra', kind: 'Worker', where: 'Frankfurt · VPS', status: 'degraded', roles: ['Relaying', 'Serving'] },
];
export const downloadingNodes = () => nodes.filter((n) => n.roles.includes('Downloading'));

// ───────────────────────────── Download Clients: one built-in per downloading Node, plus external ones attached to one Node each (#4)
export const clients = [
  {
    id: 'builtin-ams', node: 'seedbox-ams', kind: 'built-in', name: 'Built-in', status: 'ok',
    version: 'SlipStream 2.0.0', peerId: '-SS2000-', listen: { port: 51413, reachable: true },
    limits: { downMBs: null, upMBs: 60, activeDownloads: 6, activeSeeds: null },
    totals: { torrents: 412, downloading: 3, seeding: 401, queued: 2, problems: 6, downRate: 32.7, upRate: 31.4, ratio: 2.4, uploadedTB: 96.1 },
  },
  {
    id: 'qbit-ams', node: 'seedbox-ams', kind: 'qbittorrent', name: 'qBittorrent', status: 'ok',
    version: 'qBittorrent 4.6.7', host: 'http://127.0.0.1:8080', category: 'slipstream',
    why: 'Aurora allows qBittorrent but has not added the SlipStream client yet.',
    totals: { torrents: 38, downloading: 1, seeding: 37, queued: 0, problems: 0, downRate: 4.2, upRate: 5.9, ratio: 1.7, uploadedTB: 3.4 },
  },
  {
    id: 'builtin-atlas', node: 'atlas', kind: 'built-in', name: 'Built-in', status: 'ok',
    version: 'SlipStream 2.0.0', peerId: '-SS2000-', listen: { port: 6881, reachable: false, note: 'Port 6881 is closed from outside. Peers can only be reached outbound, so uploads are slower.' },
    limits: { downMBs: null, upMBs: 8, activeDownloads: 3, activeSeeds: 200 },
    totals: { torrents: 57, downloading: 1, seeding: 54, queued: 1, problems: 1, downRate: 0.0, upRate: 0.9, ratio: 1.1, uploadedTB: 2.0 },
  },
  {
    id: 'tr-atlas', node: 'atlas', kind: 'transmission', name: 'Transmission', status: 'unreachable',
    version: 'Transmission 4.0.6', host: 'http://192.168.1.20:9091', category: 'slipstream',
    lastSeen: '2026-10-08T20:52:00', error: 'Connection refused at 192.168.1.20:9091',
    totals: { torrents: 9, downloading: 0, seeding: 9, queued: 0, problems: 0, downRate: 0, upRate: 0, ratio: 3.1, uploadedTB: 0.6 },
  },
];
export const externalKinds = [
  { kind: 'qbittorrent', name: 'qBittorrent', note: 'Allowed on nearly every private tracker' },
  { kind: 'transmission', name: 'Transmission', note: 'Allowed on most private trackers, by exact release' },
  { kind: 'deluge', name: 'Deluge', note: 'Allowed on most private trackers' },
  { kind: 'rtorrent', name: 'rTorrent', note: 'Common on managed seedboxes' },
];

// ───────────────────────────── Trackers, keyed by the indexer that grabbed from them
// rule: the tracker's own minimum, which no seeding goal can go under. Hit and run (H&R) = must seed `minHours` within `windowDays`, or reach `minRatio`.
export const trackers = [
  {
    id: 'aurora', name: 'Aurora', private: true, software: 'Gazelle', status: 'not-allowed',
    message: 'Client is not on the allow list (-SS2000-)', since: '2026-10-02T10:00:00',
    route: 'qbit-ams', rule: { minRatio: 0.6, minHours: null, windowDays: null, note: 'Account ratio must stay above 0.6' },
    stats: { torrents: 38, uploadedTB: 3.4, downloadedTB: 2.0, hnrOpen: 0 },
  },
  {
    id: 'beacon', name: 'Beacon', private: true, software: 'UNIT3D', status: 'ok',
    route: 'builtin-ams', rule: { minRatio: null, minHours: 72, windowDays: 10, note: 'Seed 72 h within 10 days of finishing, or it counts as a hit and run' },
    stats: { torrents: 264, uploadedTB: 71.0, downloadedTB: 28.9, hnrOpen: 4 },
  },
  {
    id: 'cinder', name: 'Cinder', private: true, software: 'UNIT3D', status: 'degraded',
    message: 'Announces timing out since 21:12', since: '2026-10-08T21:12:00',
    route: 'builtin-ams', rule: { minRatio: 1.0, minHours: 168, windowDays: 30, note: 'Seed 7 days or reach ratio 1.0 within 30 days' },
    stats: { torrents: 148, uploadedTB: 25.1, downloadedTB: 11.6, hnrOpen: 1 },
  },
  {
    id: 'public', name: 'Public trackers', private: false, software: 'DHT, PEX and open trackers', status: 'ok',
    route: 'builtin-atlas', rule: null,
    stats: { torrents: 57, uploadedTB: 2.0, downloadedTB: 1.8, hnrOpen: 0 },
  },
];

// ───────────────────────────── Seeding goals: the built-in client's policy. External clients keep their own seeding rules (#4).
// A goal is met when `mode` is satisfied AND the tracker's rule is satisfied. Then `whenMet` runs. Library files are never touched: import hard-linked or copied them.
export const seedingGoals = {
  default: { ratio: 2.0, hours: 14 * 24, mode: 'either', whenMet: 'remove' },
  perTracker: {
    beacon: { ratio: 2.0, hours: 14 * 24, mode: 'either', whenMet: 'remove' },
    cinder: { ratio: null, hours: null, mode: 'forever', whenMet: 'keep', why: 'Bonus points for long seeding' },
    public: { ratio: 1.0, hours: null, mode: 'either', whenMet: 'remove' },
  },
  diskGuard: { minFreePct: 10, action: 'Pause new downloads on that Node' },
};
export const whenMetLabel = { remove: 'Remove the torrent and its download copy', stop: 'Stop seeding, keep the torrent', keep: 'Keep seeding' };

export function goalFor(t) {
  const tr = trackers.find((x) => x.id === t.tracker);
  const g = seedingGoals.perTracker[t.tracker] || seedingGoals.default;
  return { ...g, source: seedingGoals.perTracker[t.tracker] ? `${tr.name} goal` : 'Default goal', rule: tr.rule, external: clients.find((c) => c.id === t.client).kind !== 'built-in' };
}

// How far a seeding torrent is toward its goal: 0..1, which side leads, and whether the tracker's rule still holds it.
export function goalProgress(t) {
  const g = goalFor(t);
  if (g.external) return { kind: 'external', label: 'Seeding rules live in the external client' };
  if (t.state === 'error') return { kind: 'error', label: 'The tracker no longer accepts this torrent' };
  if (t.state === 'checking' && t.completedAt) return { kind: 'checking', label: 'Goal paused while the pieces recheck' };
  if (t.state === 'downloading' || t.state === 'queued' || t.state === 'checking') return { kind: 'pending', label: 'Goal starts when the download finishes' };
  if (g.mode === 'forever') return { kind: 'forever', label: 'Seeds until you remove it', ruleMet: ruleMet(t, g.rule) };
  const r = g.ratio ? t.ratio / g.ratio : 0;
  const h = g.hours ? t.seededH / g.hours : 0;
  const p = Math.min(1, g.mode === 'both' ? Math.min(r || 1, h || 1) : Math.max(r, h));
  const lead = r >= h ? 'ratio' : 'time';
  const rm = ruleMet(t, g.rule);
  return { kind: p >= 1 ? (rm ? 'met' : 'held') : 'seeding', p, lead, ruleMet: rm };
}
export function ruleMet(t, rule) {
  if (!rule) return true;
  if (rule.minRatio != null && t.ratio >= rule.minRatio) return true;
  if (rule.minHours != null && t.seededH >= rule.minHours) return true;
  return rule.minRatio == null && rule.minHours == null;
}
// Hours left in the H&R window, for torrents the rule still holds.
export function hnr(t) {
  const tr = trackers.find((x) => x.id === t.tracker);
  if (t.state === 'error' || !tr.rule || !tr.rule.minHours || ruleMet(t, tr.rule) || !t.completedAt) return null;
  const windowEnd = new Date(t.completedAt).getTime() + tr.rule.windowDays * 864e5;
  const leftInWindowH = (windowEnd - now.getTime()) / 36e5;
  const needH = tr.rule.minHours - t.seededH;
  return { needH, leftInWindowH, atRisk: leftInWindowH - needH < 24 };
}

// ───────────────────────────── Torrents. Rates in MB/s, sizes in GB, seededH in hours.
// state: downloading | seeding | queued | checking | stalled | paused | error | met
const ep = (season, ...eps) => eps.map((episode) => ({ season, episode }));
export const torrents = [
  { id: 't-sev-207', work: 'severance', coverage: { items: ep(2, 7) }, target: '1080p', client: 'builtin-ams', tracker: 'beacon', state: 'downloading', job: 'j-sev-dl', progress: 0.71, down: 24.6, up: 3.1, sizeGB: 3.9, downloadedGB: 2.8, uploadedGB: 0.4, ratio: 0.14, seededH: 0, seeds: 214, leech: 61, peers: 38, addedAt: '2026-10-08T21:36:00', release: 'Severance.S02E07.1080p.ATVP.WEB-DL.DDP5.1.Atmos.H.264-FLUX', announce: { status: 'ok', last: '2026-10-08T21:38:10', next: '2026-10-08T22:08:10' } },
  { id: 't-opp', work: 'oppenheimer', coverage: { items: [{ edition: 'Standard' }] }, target: '2160p HDR', client: 'builtin-ams', tracker: 'cinder', state: 'downloading', job: 'j-opp-dl', progress: 0.23, down: 8.1, up: 0.6, sizeGB: 78.4, downloadedGB: 18.0, uploadedGB: 1.9, ratio: 0.02, seededH: 0, seeds: 11, leech: 4, peers: 9, addedAt: '2026-10-08T21:05:00', release: 'Oppenheimer.2023.2160p.UHD.BluRay.REMUX.HDR10.DTS-HD.MA.5.1-EPSiLON', announce: { status: 'error', msg: 'Timed out after 30 s · retrying', last: '2026-10-08T21:12:40', next: '2026-10-08T21:42:40' } },
  { id: 't-tlou-203', work: 'tlou', coverage: { items: ep(2, 3) }, target: '1080p', client: 'builtin-ams', tracker: 'beacon', state: 'seeding', transfer: 'j-tlou-xfer', progress: 1, down: 0, up: 11.8, sizeGB: 5.2, downloadedGB: 5.2, uploadedGB: 6.9, ratio: 1.33, seededH: 0.4, completedAt: '2026-10-08T21:16:00', seeds: 402, leech: 118, peers: 22, addedAt: '2026-10-08T20:41:00', release: 'The.Last.of.Us.S02E03.1080p.MAX.WEB-DL.DDP5.1.Atmos.H.264-FLUX', announce: { status: 'ok', last: '2026-10-08T21:31:00', next: '2026-10-08T22:01:00' } },
  { id: 't-dune', work: 'dune-2', coverage: { items: [{ edition: 'Standard' }] }, target: '2160p HDR', client: 'builtin-ams', tracker: 'beacon', state: 'seeding', progress: 1, down: 0, up: 6.2, sizeGB: 74.6, downloadedGB: 74.6, uploadedGB: 134.3, ratio: 1.8, seededH: 46.4, completedAt: '2026-10-06T23:15:00', seeds: 61, leech: 7, peers: 5, addedAt: '2026-10-06T21:04:00', release: 'Dune.Part.Two.2024.2160p.UHD.BluRay.REMUX.DV.HDR10.TrueHD.Atmos.7.1-FraMeSToR', announce: { status: 'ok', last: '2026-10-08T21:20:00', next: '2026-10-08T21:50:00' } },
  { id: 't-hotd-s2', work: 'hotd', coverage: { items: ep(2, 1, 2, 3, 4, 5, 6, 7, 8), whole: true }, target: '1080p', client: 'builtin-ams', tracker: 'beacon', state: 'seeding', progress: 1, down: 0, up: 2.4, sizeGB: 38.2, downloadedGB: 38.2, uploadedGB: 21.0, ratio: 0.55, seededH: 61.0, completedAt: '2026-09-30T00:00:00', seeds: 133, leech: 9, peers: 3, addedAt: '2026-09-28T22:00:00', release: 'House.of.the.Dragon.S02.1080p.MAX.WEB-DL.DDP5.1.Atmos.H.264-FLUX', announce: { status: 'ok', last: '2026-10-08T21:25:00', next: '2026-10-08T21:55:00' } },
  { id: 't-bcs-s6', work: 'bcs', coverage: { items: ep(6, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13), whole: true }, target: '1080p', client: 'builtin-ams', tracker: 'beacon', state: 'met', progress: 1, down: 0, up: 0, sizeGB: 41.7, downloadedGB: 41.7, uploadedGB: 84.0, ratio: 2.01, seededH: 211.0, completedAt: '2026-09-29T19:00:00', seeds: 88, leech: 0, peers: 0, addedAt: '2026-09-29T15:00:00', release: 'Better.Call.Saul.S06.1080p.AMZN.WEB-DL.DDP5.1.H.264-NTb', removeAt: '2026-10-08T21:45:00', announce: { status: 'ok', last: '2026-10-08T21:40:00', next: null } },
  { id: 't-boys-401', work: 'the-boys', coverage: { items: ep(4, 1, 2, 3) }, target: '1080p', client: 'builtin-ams', tracker: 'beacon', state: 'stalled', progress: 1, down: 0, up: 0, sizeGB: 9.8, downloadedGB: 9.8, uploadedGB: 0.9, ratio: 0.09, seededH: 38.0, completedAt: '2026-10-01T04:00:00', seeds: 0, leech: 0, peers: 0, addedAt: '2026-09-30T23:00:00', release: 'The.Boys.S04E01-E03.1080p.AMZN.WEB-DL.DDP5.1.H.264-FLUX', announce: { status: 'ok', last: '2026-10-08T21:30:00', next: '2026-10-08T22:00:00' }, note: 'No leechers. It still counts toward the H&R minimum while it seeds.' },
  { id: 't-batman', work: 'the-batman', coverage: { items: [{ edition: 'Standard' }] }, target: '2160p HDR', client: 'builtin-ams', tracker: 'beacon', state: 'error', progress: 1, down: 0, up: 0, sizeGB: 21.4, downloadedGB: 21.4, uploadedGB: 2.2, ratio: 0.1, seededH: 3.1, completedAt: '2026-10-08T18:10:00', seeds: 0, leech: 0, peers: 0, addedAt: '2026-10-08T16:50:00', release: 'The.Batman.2022.2160p.WEB-DL.HDR10.DDP5.1.Atmos-PLAYWEB', announce: { status: 'error', msg: 'Unregistered torrent: trumped by a proper release', last: '2026-10-08T21:31:00', next: null }, note: 'The tracker removed this release. Seeding is over, so no hit and run counts. Remove it; the library file for 1080p SDR stays.' },
  { id: 't-int', work: 'interstellar', coverage: { items: [{ edition: 'Standard' }] }, target: '2160p HDR', client: 'builtin-ams', tracker: 'cinder', state: 'seeding', progress: 1, down: 0, up: 1.1, sizeGB: 61.0, downloadedGB: 61.0, uploadedGB: 9.8, ratio: 0.16, seededH: 2.1, completedAt: '2026-10-08T19:30:00', seeds: 23, leech: 2, peers: 1, addedAt: '2026-10-08T17:40:00', release: 'Interstellar.2014.2160p.UHD.BluRay.REMUX.HDR10.DTS-HD.MA.5.1-FGT', announce: { status: 'error', msg: 'Timed out after 30 s · retrying', last: '2026-10-08T21:12:40', next: '2026-10-08T21:42:40' } },
  { id: 't-got-s1', work: 'got', coverage: { items: ep(1, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10), whole: true }, target: '1080p', client: 'builtin-ams', tracker: 'cinder', state: 'seeding', progress: 1, down: 0, up: 0.3, sizeGB: 44.0, downloadedGB: 44.0, uploadedGB: 231.4, ratio: 5.26, seededH: 2904.0, completedAt: '2026-06-01T10:00:00', seeds: 410, leech: 3, peers: 1, addedAt: '2026-06-01T08:00:00', release: 'Game.of.Thrones.S01.1080p.BluRay.DDP5.1.x264-NTb', announce: { status: 'error', msg: 'Timed out after 30 s · retrying', last: '2026-10-08T21:12:40', next: '2026-10-08T21:42:40' } },
  { id: 't-sev-206', work: 'severance', coverage: { items: ep(2, 6) }, target: '1080p', client: 'builtin-ams', tracker: 'beacon', state: 'checking', progress: 0.44, down: 0, up: 0, sizeGB: 3.7, downloadedGB: 3.7, uploadedGB: 7.1, ratio: 1.92, seededH: 90.0, completedAt: '2026-10-04T03:40:00', seeds: 0, leech: 0, peers: 0, addedAt: '2026-10-04T03:05:00', release: 'Severance.S02E06.1080p.ATVP.WEB-DL.DDP5.1.Atmos.H.264-FLUX', announce: { status: 'updating' }, note: 'Rechecking after seedbox-ams restarted. Seeding resumes when the pieces verify.' },
  { id: 't-narcos-s3', work: 'narcos', coverage: { items: ep(3, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10), whole: true }, target: '1080p', client: 'builtin-ams', tracker: 'beacon', state: 'queued', progress: 0, down: 0, up: 0, sizeGB: 27.5, downloadedGB: 0, uploadedGB: 0, ratio: 0, seededH: 0, seeds: 57, leech: 6, peers: 0, addedAt: '2026-10-08T21:39:00', release: 'Narcos.S03.1080p.NF.WEB-DL.DDP5.1.x264-NTb', announce: { status: 'ok', last: '2026-10-08T21:39:10', next: '2026-10-08T22:09:10' }, note: 'Waiting for one of 6 download slots.' },
  { id: 't-matrix', work: 'the-matrix', coverage: { items: [{ edition: 'Standard' }] }, target: '2160p HDR', client: 'qbit-ams', tracker: 'aurora', state: 'downloading', progress: 0.58, down: 4.2, up: 0.8, sizeGB: 58.1, downloadedGB: 33.7, uploadedGB: 2.4, ratio: 0.07, seededH: 0, seeds: 18, leech: 5, peers: 12, addedAt: '2026-10-08T20:10:00', release: 'The.Matrix.1999.2160p.UHD.BluRay.REMUX.DV.HDR10.TrueHD.Atmos.7.1-FraMeSToR', announce: { status: 'ok', last: '2026-10-08T21:30:00', next: '2026-10-08T22:00:00' } },
  { id: 't-tdk', work: 'the-dark-knight', coverage: { items: [{ edition: 'Standard' }] }, target: '2160p HDR', client: 'qbit-ams', tracker: 'aurora', state: 'seeding', progress: 1, down: 0, up: 5.1, sizeGB: 66.3, downloadedGB: 66.3, uploadedGB: 92.8, ratio: 1.4, seededH: 130.0, completedAt: '2026-10-03T11:00:00', seeds: 44, leech: 6, peers: 4, addedAt: '2026-10-03T07:00:00', release: 'The.Dark.Knight.2008.2160p.UHD.BluRay.REMUX.HDR10.TrueHD.Atmos.7.1-FraMeSToR', announce: { status: 'ok', last: '2026-10-08T21:28:00', next: '2026-10-08T21:58:00' } },
  { id: 't-prestige', work: 'the-prestige', coverage: { items: [{ edition: 'Standard' }] }, target: '2160p HDR', client: 'builtin-atlas', tracker: 'public', state: 'downloading', progress: 0.09, down: 0.0, up: 0.0, sizeGB: 18.2, downloadedGB: 1.6, uploadedGB: 0, ratio: 0, seededH: 0, seeds: 2, leech: 1, peers: 1, addedAt: '2026-10-08T19:00:00', release: 'The.Prestige.2006.2160p.UHD.BluRay.x265.10bit.HDR.DTS-HD.MA.5.1', announce: { status: 'ok', last: '2026-10-08T21:30:00', next: '2026-10-08T22:00:00' }, note: 'Two seeders and no inbound connections: port 6881 on atlas is closed from outside.' },
  { id: 't-tgm', work: 'top-gun-maverick', coverage: { items: [{ edition: 'Standard' }] }, target: '2160p HDR', client: 'builtin-atlas', tracker: 'public', state: 'seeding', progress: 1, down: 0, up: 0.9, sizeGB: 22.0, downloadedGB: 22.0, uploadedGB: 16.3, ratio: 0.74, seededH: 70.0, completedAt: '2026-10-05T23:00:00', seeds: 340, leech: 41, peers: 3, addedAt: '2026-10-05T21:30:00', release: 'Top.Gun.Maverick.2022.2160p.WEB-DL.DDP5.1.Atmos.DV.HDR.H.265', announce: { status: 'ok', last: '2026-10-08T21:30:00', next: '2026-10-08T22:00:00' } },
  { id: 't-bp', work: 'black-panther', coverage: { items: [{ edition: 'Standard' }] }, target: '2160p HDR', client: 'tr-atlas', tracker: 'beacon', state: 'seeding', stale: true, progress: 1, down: 0, up: 0, sizeGB: 55.0, downloadedGB: 55.0, uploadedGB: 170.0, ratio: 3.1, seededH: 400.0, completedAt: '2026-09-20T00:00:00', seeds: 30, leech: 1, peers: 0, addedAt: '2026-09-19T20:00:00', release: 'Black.Panther.2018.2160p.UHD.BluRay.REMUX.HDR10.TrueHD.Atmos.7.1-FGT', announce: { status: 'ok', last: '2026-10-08T20:50:00', next: null } },
];
// Each built-in client holds hundreds more, shown on demand. Counts come from clients[].totals.
export const moreTorrents = { 'builtin-ams': 400, 'builtin-atlas': 55, 'qbit-ams': 36, 'tr-atlas': 8 };
